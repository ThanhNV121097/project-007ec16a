package db

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestListUpOrdersByNameAndIgnoresOtherFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"20260102000000_b.up.sql":   {Data: []byte("SELECT 1")},
		"20260101000000_a.up.sql":   {Data: []byte("SELECT 1")},
		"20260101000000_a.down.sql": {Data: []byte("SELECT 1")},
		"README.md":                 {Data: []byte("x")},
	}
	got, err := listUp(fsys)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, m := range got {
		names = append(names, m.name)
	}
	want := []string{"20260101000000_a.up.sql", "20260102000000_b.up.sql"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestListUpRefusesBadNamesAndRepeatedVersions(t *testing.T) {
	bad := []fstest.MapFS{
		{"init.up.sql": {Data: []byte("x")}},
		{"2026_a.up.sql": {Data: []byte("x")}},
		{
			"20260101000000_a.up.sql": {Data: []byte("x")},
			"20260101000000_b.up.sql": {Data: []byte("x")},
		},
	}
	for i, fsys := range bad {
		if _, err := listUp(fsys); err == nil {
			t.Errorf("case %d: expected an error", i)
		}
	}
}

func TestNoTransactionIsDecidedByConcurrently(t *testing.T) {
	if !needsNoTransaction("create index concurrently if not exists i on t (c);") {
		t.Error("lower-case CONCURRENTLY not detected")
	}
	if needsNoTransaction("CREATE INDEX i ON t (c);") {
		t.Error("a plain index was taken for a concurrent one")
	}
}

func TestSplitStatementsDropsCommentOnlyPieces(t *testing.T) {
	body := "-- first\nCREATE INDEX CONCURRENTLY a ON t (x);\n\n-- second\nCREATE INDEX CONCURRENTLY b ON t (y);\n-- the end\n"
	got := splitStatements(body)
	if len(got) != 2 {
		t.Fatalf("got %d statements %q, want 2", len(got), got)
	}
}

// testPool returns a pool whose search_path is a schema of its own, so the
// test never touches the tables a real migration made in the same database.
// It skips when TEST_DATABASE_URL is not set, as CI's backend job sets it.
func testPool(t *testing.T) *pgxpool.Pool {
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
	schema := fmt.Sprintf("mig_test_%d", time.Now().UnixNano())
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

func count(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestMigrateAppliesOnceAndRunsConcurrentIndexesOutsideATransaction(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	fsys := fstest.MapFS{
		"20260101000000_widgets.up.sql": {Data: []byte(
			"CREATE TABLE widgets (id int PRIMARY KEY, label text);\nINSERT INTO widgets VALUES (1, 'a');")},
		"20260101000000_widgets.down.sql": {Data: []byte("DROP TABLE widgets;")},
		"20260101000001_widget_indexes.up.sql": {Data: []byte(
			"CREATE INDEX CONCURRENTLY IF NOT EXISTS widgets_label_idx ON widgets (label);\n" +
				"CREATE INDEX CONCURRENTLY IF NOT EXISTS widgets_label_id_idx ON widgets (label, id);\n")},
	}
	for run := 1; run <= 2; run++ {
		if err := Migrate(ctx, pool, fsys, quiet()); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}
	if n := count(t, pool, "schema_migrations"); n != 2 {
		t.Errorf("schema_migrations has %d rows, want 2", n)
	}
	if n := count(t, pool, "widgets"); n != 1 {
		t.Errorf("widgets has %d rows after two runs, want 1", n)
	}
}

// Two instances that start together while a third holds the lock: whichever
// gets it next runs CREATE INDEX CONCURRENTLY, which waits for every running
// transaction. The other must be idle, not blocked inside pg_advisory_lock, or
// PostgreSQL finds a cycle and fails one of them with SQLSTATE 40P01.
func TestMigrateWaitingForTheLockDoesNotDeadlockAConcurrentIndex(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fsys := fstest.MapFS{
		"20260101000000_widgets.up.sql": {Data: []byte("CREATE TABLE widgets (id int PRIMARY KEY, label text);")},
		"20260101000001_widget_indexes.up.sql": {Data: []byte(
			"CREATE INDEX CONCURRENTLY IF NOT EXISTS widgets_label_idx ON widgets (label);\n")},
	}

	holder, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := holder.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockKey); err != nil {
		holder.Release()
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- Migrate(ctx, pool, fsys, quiet())
		}()
	}
	time.Sleep(5 * lockPollInterval) // both are now waiting for the lock
	if _, err := holder.Exec(ctx, "SELECT pg_advisory_unlock($1)", migrationLockKey); err != nil {
		holder.Release()
		t.Fatal(err)
	}
	holder.Release()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("migrate: %v", err)
		}
	}
	if n := count(t, pool, "schema_migrations"); n != 2 {
		t.Errorf("schema_migrations has %d rows, want 2", n)
	}
}

func TestMigrateGivesUpWaitingForTheLockWhenItsContextIsDone(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	holder, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Release()
	if _, err := holder.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockKey); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = holder.Exec(ctx, "SELECT pg_advisory_unlock($1)", migrationLockKey) }()

	waitCtx, cancel := context.WithTimeout(ctx, 3*lockPollInterval)
	defer cancel()
	if err := Migrate(waitCtx, pool, fstest.MapFS{}, quiet()); err == nil {
		t.Error("Migrate returned while another instance held the lock")
	}
}

func TestMigrateRollsBackAFailingFileAndRecordsNothing(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	fsys := fstest.MapFS{
		"20260101000000_broken.up.sql": {Data: []byte("CREATE TABLE half_done (id int); SELECT * FROM no_such_table;")},
	}
	if err := Migrate(ctx, pool, fsys, quiet()); err == nil {
		t.Fatal("expected the broken migration to fail")
	}
	if n := count(t, pool, "schema_migrations"); n != 0 {
		t.Errorf("schema_migrations has %d rows, want 0", n)
	}
	var exists bool
	err := pool.QueryRow(ctx, "SELECT to_regclass('half_done') IS NOT NULL").Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("half_done survived a failed migration")
	}
}
