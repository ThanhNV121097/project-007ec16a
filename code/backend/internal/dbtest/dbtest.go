// Package dbtest gives a test a real PostgreSQL schema of its own. Tests that
// need a database read TEST_DATABASE_URL, as CI's backend job sets it, and skip
// without it (on a developer's machine and in run_tests it is not set).
//
// Only _test.go files import this package.
package dbtest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/db"
	"github.com/ThanhNV121097/project-007ec16a/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

var seq atomic.Int64

// Pool returns a pool whose search_path is a schema made for this test and
// dropped when it ends, so a test never touches the tables a real migration
// made in the same database. It skips the test when TEST_DATABASE_URL is unset.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("dbtest_%d_%d", time.Now().UnixNano(), seq.Add(1))
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	return pool
}

// Migrated is Pool with every migration applied.
func Migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := Pool(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := db.Migrate(context.Background(), pool, migrations.FS, quiet); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}
