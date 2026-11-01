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
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/handler"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages"
	root "github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages/root"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/services"
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
	db, err := openDatabase("presentation.db")
	if err != nil {
		logger.Error("failed to open database", slog.String("error", err.Error()))
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", slog.String("error", err.Error()))
		}
	}()

	localKey := readEnvOrDefault(logger, "ATTENDEE_LOCAL_KEY", "local-dev-key")
	remoteKey := readEnvOrDefault(logger, "ATTENDEE_REMOTE_KEY", "remote-dev-key")

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	sessionSecret := readEnvOrDefault(logger, "SESSION_SECRET", "local-dev-session-secret")
	sessionStore := sessions.NewCookieStore([]byte(sessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(24 * time.Hour / time.Second),
		HttpOnly: true,
	}

	natsPort, err := toolbelt.FreePort()
	if err != nil {
		logger.Error("failed to get free port for nats", slog.String("error", err.Error()))
		return
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

	rootCtx := context.Background()

	ns, err := embeddednats.New(rootCtx, embeddednats.WithNATSServerOptions(&natsserver.Options{
		JetStream: true,
		Port:      natsPort,
	}))
	if err != nil {
		panic(fmt.Sprintf("failed to start embedded nats server: %v", err))
	}

	ebs, err := services.NewEmojiBalloonService(ns)
	if err != nil {
		logger.Error("failed to create emoji balloon service", slog.String("error", err.Error()))
		return
	}
	demoHandler, err := handler.NewDemoHandler(ns, ebs)
	if err != nil {
		logger.Error("failed to create demo handler", slog.String("error", err.Error()))
		return
	}
	router.Get("/demo", demoHandler.DemoRoute)
	router.Get("/demo/sse", demoHandler.DemoSSE)

	nc, err := ns.Client()
	if err != nil {
		panic(fmt.Sprintf("failed to create nats client: %v", err))
	}

	js, err := jetstream.New(nc)
	if err != nil {
		panic(fmt.Sprintf("failed to create jetstream context: %v", err))
	}

	kv, err := js.CreateOrUpdateKeyValue(rootCtx, jetstream.KeyValueConfig{
		Bucket:      "presentation",
		Description: "Frontiers Meetup Presentation Bucket",
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create or update key value store: %v", err))
	}

	pages.PresenterRoutes(router, sessionStore, logger)

	// Keyed access: /<key>/...
	router.Route("/{inviteKey}", func(inviteRouter chi.Router) {
		inviteRouter.Use(inviteKeyMiddleware(sessionStore, localKey, remoteKey, logger))
		root.RootLayoutRoute(inviteRouter, db, sessionStore, kv, logger, localKey, remoteKey, ns, ebs)
	})

	address := ":8080"
	logger.Info("starting HTTP server", slog.String("address", address))

	httpServerError := http.ListenAndServe(address, router)
	if httpServerError != nil {
		logger.Error("HTTP server error", slog.String("error", httpServerError.Error()))
	}
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
