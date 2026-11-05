package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	natsStoreDir, removeNATSStoreDir, err := natsStoreDirectory(logger)
	if err != nil {
		logger.Error("failed to prepare NATS store directory", slog.String("error", err.Error()))
		return
	}
	defer removeNATSStoreDir()

	config := appConfig{
		DatabasePath:  "presentation.db",
		NATSStoreDir:  natsStoreDir,
		InviteKey:     readEnvOrDefault(logger, "INVITE_KEY", "dev-key"),
		SessionSecret: readEnvOrDefault(logger, "SESSION_SECRET", "local-dev-session-secret"),
	}

	// Ctrl+C (dev) and SIGTERM (systemctl stop on the server) end the context, so the deferred cleanup
	// runs: the database is closed and the temp NATS directory removed.
	ctx, stopListeningForSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopListeningForSignals()

	app, closeApp, err := newApp(ctx, logger, config)
	if err != nil {
		logger.Error("failed to start app", slog.String("error", err.Error()))
		return
	}
	defer closeApp()

	// HOST is empty locally, so phones on the LAN can reach the app. On the server it is 127.0.0.1:
	// Caddy is the only way in, with TLS (deploy/juleavslutning.service.tmpl).
	address := strings.TrimSpace(os.Getenv("HOST")) + ":" + readEnvOrDefault(logger, "PORT", "8080")
	server := &http.Server{Addr: address, Handler: app}
	serverStopped := make(chan error, 1)
	go func() { serverStopped <- server.ListenAndServe() }()
	logger.Info("starting HTTP server", slog.String("address", address))

	select {
	case err := <-serverStopped:
		logger.Error("HTTP server error", slog.String("error", err.Error()))
	case <-ctx.Done():
		logger.Info("shutting down")
		// Live-update streams stay open until the phone leaves, so a graceful shutdown would wait for them
		// forever; after a short grace period the remaining connections are closed.
		gracePeriod, cancelGracePeriod := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelGracePeriod()
		if err := server.Shutdown(gracePeriod); err != nil {
			if closeErr := server.Close(); closeErr != nil {
				logger.Error("failed to close HTTP server", slog.String("error", closeErr.Error()))
			}
		}
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

	pages.PresenterRoutes(router, sessionStore, logger)

	// Keyed access: /<key>/...
	router.Route("/{inviteKey}", func(inviteRouter chi.Router) {
		inviteRouter.Use(inviteKeyMiddleware(config.InviteKey))
		root.RootLayoutRoute(inviteRouter, db, sessionStore, logger, nc)
	})

	closeApp = func() {
		// Wait for NATS to stop before returning: the caller removes its store directory afterwards.
		ns.NatsServer.Shutdown()
		ns.NatsServer.WaitForShutdown()
		closeDatabase()
	}
	return router, closeApp, nil
}

// natsStoreDirectory returns where embedded NATS keeps its JetStream data. NATS state never has to survive a
// restart: anything that must is stored in SQLite. On the server, NATS_STORE_DIR points into the service's
// systemd RuntimeDirectory (/run/..., in memory, wiped on restart), like conorganizer. Locally, a fresh temp
// directory is created per run and removed on exit.
func natsStoreDirectory(logger *slog.Logger) (dir string, remove func(), err error) {
	if dir := strings.TrimSpace(os.Getenv("NATS_STORE_DIR")); dir != "" {
		return dir, func() {}, nil
	}
	dir, err = os.MkdirTemp("", "juleavslutning-nats-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp NATS store directory: %w", err)
	}
	return dir, func() {
		if err := os.RemoveAll(dir); err != nil {
			logger.Error("failed to remove temp NATS store directory", slog.String("dir", dir), slog.String("error", err.Error()))
			return
		}
		logger.Info("removed temp NATS store directory", slog.String("dir", dir))
	}, nil
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
