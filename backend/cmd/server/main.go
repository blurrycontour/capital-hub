// Command server is the Capital-Hub backend entrypoint: it loads configuration,
// opens the database, applies migrations, and serves the HTTP API plus the
// embedded frontend.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aditya/capital-hub/internal/auth"
	"github.com/aditya/capital-hub/internal/config"
	"github.com/aditya/capital-hub/internal/database"
	"github.com/aditya/capital-hub/internal/httpapi"
	"github.com/aditya/capital-hub/internal/logging"
)

func main() {
	if err := run(); err != nil {
		// Logger may not be initialized yet; fall back to stderr.
		os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel, cfg.IsDev())
	logger.Info("starting capital-hub", "env", cfg.Env, "addr", cfg.Addr, "data_dir", cfg.DataDir)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Ensure writable directories exist.
	if err := os.MkdirAll(cfg.UploadsDir(), 0o755); err != nil {
		return err
	}

	db, err := database.Open(ctx, cfg.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		return err
	}

	authSvc := auth.NewService(db, cfg)
	if err := authSvc.EnsureBootstrapAdmin(ctx); err != nil {
		return err
	}
	logger.Info("database ready", "path", cfg.DBPath())

	srv, err := httpapi.New(cfg, db, logger)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return err
	}
	logger.Info("server stopped cleanly")
	return nil
}
