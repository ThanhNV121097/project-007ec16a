package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationLockKey serialises Migrate across instances that start together:
// the second one waits, then finds nothing left to apply.
const migrationLockKey int64 = 7424001

// lockPollInterval is how long an instance that did not get the lock waits
// before asking again. See takeLock for why it asks rather than waits.
const lockPollInterval = 100 * time.Millisecond

var (
	upName       = regexp.MustCompile(`^(\d{14})_[a-z0-9_]+\.up\.sql$`)
	concurrently = regexp.MustCompile(`(?i)\bconcurrently\b`)
)

type migration struct{ version, name string }

// Migrate applies every *.up.sql in fsys that schema_migrations does not list,
// in filename order, and records each one. Running it again is a no-op.
//
// A file runs in one transaction with its schema_migrations row. A file whose
// SQL contains CONCURRENTLY runs outside one — Postgres refuses CREATE INDEX
// CONCURRENTLY inside a transaction — one statement at a time, and is recorded
// afterwards, so its statements must be safe to run twice.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS, log *slog.Logger) error {
	files, err := listUp(fsys)
	if err != nil {
		return err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if err := takeLock(ctx, conn); err != nil {
		return fmt.Errorf("take migration lock: %w", err)
	}
	defer func() {
		// The lock belongs to this session: release it even when ctx is done,
		// or the pooled connection would carry it into its next use.
		_, _ = conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", migrationLockKey)
	}()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text PRIMARY KEY,
		name       text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	applied, err := appliedVersions(ctx, conn)
	if err != nil {
		return err
	}
	for _, m := range files {
		if applied[m.version] {
			continue
		}
		body, err := fs.ReadFile(fsys, m.name)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.name, err)
		}
		if err := apply(ctx, conn, m, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		log.Info("migration applied", "version", m.version, "name", m.name)
	}
	return nil
}

// takeLock takes the migration lock by asking for it every lockPollInterval
// until it is free or ctx is done.
//
// It does not block in pg_advisory_lock. A blocked pg_advisory_lock is a running
// statement, so its session holds a snapshot and a virtual transaction id; the
// holder of the lock, running CREATE INDEX CONCURRENTLY, waits for every such
// transaction to end — and the waiter waits for the holder. PostgreSQL detects
// the cycle and fails one of them (SQLSTATE 40P01). pg_try_advisory_lock
// answers at once, so between two asks the waiter is idle and holds nothing.
func takeLock(ctx context.Context, conn *pgxpool.Conn) error {
	for {
		var got bool
		if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", migrationLockKey).Scan(&got); err != nil {
			return err
		}
		if got {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(lockPollInterval):
		}
	}
}

func appliedVersions(ctx context.Context, conn *pgxpool.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()
	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("read schema_migrations: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	return applied, nil
}

func apply(ctx context.Context, conn *pgxpool.Conn, m migration, body string) error {
	const record = "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)"
	if needsNoTransaction(body) {
		for _, stmt := range splitStatements(body) {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				return withDetail(err)
			}
		}
		_, err := conn.Exec(ctx, record, m.version, m.name)
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // a no-op once committed
	if _, err := tx.Exec(ctx, body); err != nil {
		return withDetail(err)
	}
	if _, err := tx.Exec(ctx, record, m.version, m.name); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// listUp returns the *.up.sql files of fsys's root, in filename order. A name
// that does not follow the convention, or a repeated version, is an error:
// silently skipping a migration is the worse failure.
func listUp(fsys fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	var out []migration
	seen := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		m := upName.FindStringSubmatch(name)
		if m == nil {
			return nil, fmt.Errorf("migration %q: name must be YYYYMMDDHHMMSS_description.up.sql", name)
		}
		if other, dup := seen[m[1]]; dup {
			return nil, fmt.Errorf("migrations %q and %q share the version %s", other, name, m[1])
		}
		seen[m[1]] = name
		out = append(out, migration{version: m[1], name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func needsNoTransaction(body string) bool { return concurrently.MatchString(body) }

// withDetail adds PostgreSQL's DETAIL line to a failed statement's error.
// pgconn.PgError.Error() leaves it out, and for a duplicate key it is the part
// that names the id (`Key (id)=(samba) already exists.`): a seed that meets a
// record already there must say which one.
func withDetail(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Detail != "" {
		return fmt.Errorf("%w (%s)", err, pg.Detail)
	}
	return err
}

// splitStatements cuts a no-transaction file at its semicolons and drops the
// pieces that hold nothing but comments. It does not understand $$ bodies,
// which is why such files may hold plain statements only.
func splitStatements(body string) []string {
	var out []string
	for _, part := range strings.Split(body, ";") {
		if hasSQL(part) {
			out = append(out, strings.TrimSpace(part))
		}
	}
	return out
}

func hasSQL(part string) bool {
	for _, line := range strings.Split(part, "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "--") {
			return true
		}
	}
	return false
}
