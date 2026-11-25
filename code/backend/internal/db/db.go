// Package db opens the Postgres pool and applies the schema migrations.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// connectTimeout bounds how long start-up waits for a database that is still
// coming up (a restarting container, a shared server). Past it the server
// exits and the orchestrator restarts it.
const connectTimeout = 30 * time.Second

// Connect opens a pool on dsn and waits until the database answers.
// Pool size and other pgx options ride in the DSN (`pool_max_conns=10`).
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	deadline := time.Now().Add(connectTimeout)
	for delay := 250 * time.Millisecond; ; delay = min(delay*2, 3*time.Second) {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return pool, nil
		}
		if ctx.Err() != nil || time.Now().Add(delay).After(deadline) {
			pool.Close()
			return nil, fmt.Errorf("database not reachable: %w", err)
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
}
