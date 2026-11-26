// Command api is the shop's backend: it connects to Postgres, applies every
// pending migration, then serves the HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/db"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/server"
	"github.com/ThanhNV121097/project-007ec16a/backend/migrations"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// run is the start-up order that matters: configuration, database, migrations,
// and only then the listener. A server that listened first would answer
// requests against a schema that is not there yet.
func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, migrations.FS, log); err != nil {
		return err
	}
	// The admin account comes from the owner's settings. A problem with them is
	// logged and skipped (the server always starts); only a database error stops
	// the process, so a deployment that cannot write the row is restarted.
	if err := auth.EnsureAdmin(ctx, pool, cfg.AdminEmail, cfg.AdminPassword, log); err != nil {
		return err
	}
	go auth.RunPurge(ctx, pool, log)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(cfg, pool, log),
		ReadHeaderTimeout: 10 * time.Second,
		// A picture upload (5 MB) over a phone connection is slow.
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	failed := make(chan error, 1)
	go func() { failed <- srv.ListenAndServe() }()
	log.Info("listening", "addr", srv.Addr)

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
