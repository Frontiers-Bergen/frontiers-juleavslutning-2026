package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages"
	root "github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages/root"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// Package app wires the whole application: database, embedded NATS, sessions and routes. It lives in the
// repository root because it embeds schema.sql and the feature CSS files (go:embed only reaches files in its own
// folder and below). The program itself is cmd/juleavslutning; the HTTP tests start the same app through New.

type Config struct {
	DatabasePath string
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
// closeApp stops NATS and closes the database.
func New(logger *slog.Logger, config Config) (app http.Handler, closeApp func(), err error) {
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

	ns, nc, err := startNATS()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			stopNATS(ns, nc)
		}
	}()

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
		stopNATS(ns, nc)
		closeDatabase()
	}
	return router, closeApp, nil
}

// startNATS runs the NATS server inside this process, without a network port: nothing outside the app can reach
// it (with a port it listened on every address, unprotected, and the server's firewall does not block high ports).
// Only core messages are used, so there is no JetStream and no store directory: NATS only says that something
// changed, and state lives in SQLite.
func startNATS() (*natsserver.Server, *nats.Conn, error) {
	ns, err := natsserver.NewServer(&natsserver.Options{
		DontListen: true,
		// The NATS server would otherwise catch Ctrl+C/SIGTERM itself and exit the whole process at once,
		// skipping main's shutdown (closing the database).
		NoSigs: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create the embedded NATS server: %w", err)
	}
	ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		ns.Shutdown()
		return nil, nil, errors.New("the embedded NATS server did not start within 10 s")
	}
	nc, err := nats.Connect("", nats.InProcessServer(ns))
	if err != nil {
		ns.Shutdown()
		return nil, nil, fmt.Errorf("failed to connect to the embedded NATS server: %w", err)
	}
	return ns, nc, nil
}

func stopNATS(ns *natsserver.Server, nc *nats.Conn) {
	nc.Close()
	ns.Shutdown()
	ns.WaitForShutdown()
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
