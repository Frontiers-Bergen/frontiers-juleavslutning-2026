// Command juleavslutning runs the app: it reads the configuration from the environment (and .env), starts the
// HTTP server, and shuts down cleanly on Ctrl+C or SIGTERM. The app itself is package app in the repository root.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	app "github.com/frontiers-bergen/frontiers-juleavslutning-2026"
	"github.com/joho/godotenv"
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

	config := app.Config{
		DatabasePath:  "presentation.db",
		NATSStoreDir:  natsStoreDir,
		InviteKey:     readEnvOrDefault(logger, "INVITE_KEY", "dev-key"),
		SessionSecret: readEnvOrDefault(logger, "SESSION_SECRET", "local-dev-session-secret"),
	}

	// Ctrl+C (dev) and SIGTERM (systemctl stop on the server) end the context, so the deferred cleanup
	// runs: the database is closed and the temp NATS directory removed.
	ctx, stopListeningForSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopListeningForSignals()

	handler, closeApp, err := app.New(ctx, logger, config)
	if err != nil {
		logger.Error("failed to start app", slog.String("error", err.Error()))
		return
	}
	defer closeApp()

	// HOST is empty locally, so phones on the LAN can reach the app. On the server it is 127.0.0.1:
	// Caddy is the only way in, with TLS (deploy/juleavslutning.service.tmpl).
	address := strings.TrimSpace(os.Getenv("HOST")) + ":" + readEnvOrDefault(logger, "PORT", "8080")
	server := &http.Server{Addr: address, Handler: handler}
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
