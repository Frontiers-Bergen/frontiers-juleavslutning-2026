package main

import (
	"context"
	"errors"
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
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := godotenv.Load(); err != nil {
		logger.Info("no .env file loaded (this is fine in prod)", slog.String("error", err.Error()))
	}

	config := appConfig{
		DatabasePath:  "presentation.db",
		NATSStoreDir:  "nats-data",
		InviteKey:     readEnvOrDefault(logger, "INVITE_KEY", "dev-key"),
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
	// NATSStoreDir is where JetStream keeps its data. Required: the NATS default is one shared temp dir,
	// and two app instances on the same machine (dev server, tests, task screenshots) corrupted each other's
	// JetStream data, which showed up as "context deadline exceeded" when loading the deck.
	NATSStoreDir string
	// InviteKey is the secret first path segment of every deck URL (/<InviteKey>/). Everyone at the venue
	// uses the same link; there is no remote audience.
	InviteKey     string
	SessionSecret string
}

// newApp wires the whole application: database, embedded NATS, sessions and routes.
// main and the HTTP tests both use it, so tests exercise the real router.
// The embedded NATS server shuts down when ctx is cancelled; closeApp closes the database.
func newApp(ctx context.Context, logger *slog.Logger, config appConfig) (app http.Handler, closeApp func(), err error) {
	if config.NATSStoreDir == "" {
		return nil, nil, errors.New("appConfig.NATSStoreDir is required; the NATS default store dir is shared by every instance on the machine")
	}

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
	if err := featureStylesRoutes(router); err != nil {
		return nil, nil, err
	}
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

	pages.PresenterRoutes(router, sessionStore, logger)

	// Keyed access: /<key>/...
	router.Route("/{inviteKey}", func(inviteRouter chi.Router) {
		inviteRouter.Use(inviteKeyMiddleware(config.InviteKey))
		root.RootLayoutRoute(inviteRouter, db, sessionStore, logger, nc)
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

// inviteKeyMiddleware lets only requests with the invite key through; anything else goes to the
// "you need an invite link" page.
func inviteKeyMiddleware(inviteKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if chi.URLParam(r, "inviteKey") != inviteKey {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
