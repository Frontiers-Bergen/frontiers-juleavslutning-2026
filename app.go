package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/delaneyj/toolbelt"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages"
	root "github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages/root"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	natsserver "github.com/nats-io/nats-server/v2/server"
)

// Package app wires the whole application: database, embedded NATS, sessions and routes. It lives in the
// repository root because it embeds schema.sql and the feature CSS files (go:embed only reaches files in its own
// folder and below). The program itself is cmd/juleavslutning; the HTTP tests start the same app through New.

type Config struct {
	DatabasePath string
	// NATSStoreDir is where JetStream keeps its data. Required: the NATS default is one shared temp dir,
	// and two app instances on the same machine (dev server, tests, task screenshots) corrupted each other's
	// JetStream data, which showed up as "context deadline exceeded" when loading the deck.
	NATSStoreDir string
	// InviteKey is the secret first path segment of every deck URL (/<InviteKey>/). Everyone at the venue
	// uses the same link; there is no remote audience.
	InviteKey string
	// PresenterKey unlocks presenter mode at /presenter; empty means nobody can be presenter.
	PresenterKey  string
	SessionSecret string
	// TugOfWarRules sets the tug of war's countdown, round length and winning lead; zero means the defaults.
	TugOfWarRules root.TugOfWarRules
}

// New wires the whole application: database, embedded NATS, sessions and routes.
// main and the HTTP tests both use it, so tests exercise the real router.
// The embedded NATS server shuts down when ctx is cancelled; closeApp closes the database.
func New(ctx context.Context, logger *slog.Logger, config Config) (app http.Handler, closeApp func(), err error) {
	if config.NATSStoreDir == "" {
		return nil, nil, errors.New("Config.NATSStoreDir is required; the NATS default store dir is shared by every instance on the machine")
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
		// The NATS server would otherwise catch Ctrl+C/SIGTERM itself and exit the whole process at once,
		// skipping main's shutdown (closing the database, removing the temp store directory).
		NoSigs: true,
	}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start embedded nats server: %w", err)
	}

	nc, err := ns.Client()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create nats client: %w", err)
	}

	tugOfWarRules := config.TugOfWarRules
	if tugOfWarRules == (root.TugOfWarRules{}) {
		tugOfWarRules = root.DefaultTugOfWarRules
	}

	pages.PresenterRoutes(router, sessionStore, logger, config.PresenterKey)

	// Keyed access: /<key>/...
	router.Route("/{inviteKey}", func(inviteRouter chi.Router) {
		inviteRouter.Use(inviteKeyMiddleware(config.InviteKey))
		root.RootLayoutRoute(inviteRouter, db, sessionStore, logger, nc, tugOfWarRules)
	})

	closeApp = func() {
		// Wait for NATS to stop before returning: the caller removes its store directory afterwards.
		ns.NatsServer.Shutdown()
		ns.NatsServer.WaitForShutdown()
		closeDatabase()
	}
	return router, closeApp, nil
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
