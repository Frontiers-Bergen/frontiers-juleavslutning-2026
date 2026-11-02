package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/delaneyj/toolbelt"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages"
	root "github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages/root"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go/jetstream"
)

const inviteSessionName = "invite"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := godotenv.Load(); err != nil {
		logger.Info("no .env file loaded (this is fine in prod)", slog.String("error", err.Error()))
	}

	config := appConfig{
		DatabasePath:  "presentation.db",
		LocalKey:      readEnvOrDefault(logger, "ATTENDEE_LOCAL_KEY", "local-dev-key"),
		RemoteKey:     readEnvOrDefault(logger, "ATTENDEE_REMOTE_KEY", "remote-dev-key"),
		SessionSecret: readEnvOrDefault(logger, "SESSION_SECRET", "local-dev-session-secret"),
	}

	app, closeApp, err := newApp(context.Background(), logger, config)
	if err != nil {
		logger.Error("failed to start app", slog.String("error", err.Error()))
		return
	}
	defer closeApp()

	address := ":" + readEnvOrDefault(logger, "PORT", "8080")
	logger.Info("starting HTTP server", slog.String("address", address))

	httpServerError := http.ListenAndServe(address, app)
	if httpServerError != nil {
		logger.Error("HTTP server error", slog.String("error", httpServerError.Error()))
	}
}

type appConfig struct {
	DatabasePath string
	// NATSStoreDir is where JetStream keeps its data. Empty means the NATS default (a temp dir).
	NATSStoreDir  string
	LocalKey      string
	RemoteKey     string
	SessionSecret string
}

// newApp wires the whole application: database, embedded NATS, sessions and routes.
// main and the HTTP tests both use it, so tests exercise the real router.
// The embedded NATS server shuts down when ctx is cancelled; closeApp closes the database.
func newApp(ctx context.Context, logger *slog.Logger, config appConfig) (app http.Handler, closeApp func(), err error) {
	db, err := openDatabase(config.DatabasePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}
	closeDatabase := func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", slog.String("error", err.Error()))
		}
	}
	defer func() {
		if err != nil {
			closeDatabase()
		}
	}()

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	sessionStore := sessions.NewCookieStore([]byte(config.SessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(24 * time.Hour / time.Second),
		HttpOnly: true,
	}

	natsPort, err := toolbelt.FreePort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get free port for nats: %w", err)
	}

	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	})

	// Root (no key): show a "you need a key" page.
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if err := root.NeedKeyPage().Render(r.Context(), w); err != nil {
			logger.Error("failed to render NeedKeyPage", slog.String("error", err.Error()))
		}
	})

	ns, err := embeddednats.New(ctx, embeddednats.WithNATSServerOptions(&natsserver.Options{
		JetStream: true,
		Port:      natsPort,
		StoreDir:  config.NATSStoreDir,
	}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start embedded nats server: %w", err)
	}

	nc, err := ns.Client()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create nats client: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create jetstream context: %w", err)
	}

	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      "presentation",
		Description: "Frontiers Meetup Presentation Bucket",
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create or update key value store: %w", err)
	}

	pages.PresenterRoutes(router, sessionStore, logger)

	// Keyed access: /<key>/...
	router.Route("/{inviteKey}", func(inviteRouter chi.Router) {
		inviteRouter.Use(inviteKeyMiddleware(sessionStore, config.LocalKey, config.RemoteKey, logger))
		root.RootLayoutRoute(inviteRouter, db, sessionStore, kv, logger, config.LocalKey, config.RemoteKey, nc)
	})

	return router, closeDatabase, nil
}

func readEnvOrDefault(logger *slog.Logger, name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		logger.Warn("missing env var, using fallback (dev only)", slog.String("env", name))
		return fallback
	}
	return v
}

func inviteKeyMiddleware(store sessions.Store, localKey, remoteKey string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inviteKey := chi.URLParam(r, "inviteKey")

			var audience string
			switch inviteKey {
			case localKey:
				audience = "local"
			case remoteKey:
				audience = "remote"
			default:
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			// Persist which audience this key represents (useful later in handlers/templates).
			sess, err := store.Get(r, inviteSessionName)
			if err != nil {
				logger.Error("failed to get invite session", slog.String("error", err.Error()))
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			if sess.Values["audience"] != audience {
				sess.Values["audience"] = audience
				if err := sess.Save(r, w); err != nil {
					logger.Error("failed to save invite session", slog.String("error", err.Error()))
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
