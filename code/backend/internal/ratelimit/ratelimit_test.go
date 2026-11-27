package ratelimit

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/dbtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSubjectIs32BytesAndDependsOnTheScope(t *testing.T) {
	a := Subject(SignInEmail, "dealer@example.com")
	b := Subject(SignInAddress, "dealer@example.com")
	if len(a) != 32 || len(b) != 32 {
		t.Fatalf("lengths %d, %d", len(a), len(b))
	}
	if bytes.Equal(a, b) {
		t.Error("two scopes share a subject hash")
	}
	if bytes.Contains(a, []byte("dealer")) {
		t.Error("the hash holds the subject")
	}
	if !bytes.Equal(a, Subject(SignInEmail, "dealer@example.com")) {
		t.Error("the same subject hashed twice differs")
	}
}

type fakeRow struct {
	secs int
	err  error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*int)) = r.secs
	return nil
}

type fakeQ struct {
	row   fakeRow
	sql   string
	args  []any
	execs int
}

func (f *fakeQ) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.sql, f.args = sql, args
	return f.row
}

func (f *fakeQ) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.sql, f.args = sql, args
	f.execs++
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func TestCheckReadsTheWaitAndTreatsNoRowAsNotLocked(t *testing.T) {
	ctx := context.Background()
	if wait, err := Check(ctx, &fakeQ{row: fakeRow{err: pgx.ErrNoRows}}, SignInEmail, nil); err != nil || wait != 0 {
		t.Errorf("no row: %v, %v", wait, err)
	}
	if wait, err := Check(ctx, &fakeQ{row: fakeRow{secs: 420}}, SignInEmail, nil); err != nil || wait != 420*time.Second {
		t.Errorf("locked: %v, %v", wait, err)
	}
	if wait, _ := Check(ctx, &fakeQ{row: fakeRow{secs: 0}}, SignInEmail, nil); wait != time.Second {
		t.Errorf("a locked subject has at least a second left, got %v", wait)
	}
	boom := errors.New("connection refused")
	if _, err := Check(ctx, &fakeQ{row: fakeRow{err: boom}}, SignInEmail, nil); !errors.Is(err, boom) {
		t.Errorf("a failure must be returned, got %v", err)
	}
}

func TestFailSendsLimitWindowAndLockAsSeconds(t *testing.T) {
	q := &fakeQ{}
	subject := Subject(SignInEmail, "a@b.example")
	if err := Fail(context.Background(), q, SignInEmail, subject, 5, 15*time.Minute, 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	if q.execs != 1 || len(q.args) != 5 || q.args[2] != 5 || q.args[3] != 900.0 || q.args[4] != 900.0 {
		t.Errorf("args: %v", q.args)
	}
}

// ---- against PostgreSQL (skipped without TEST_DATABASE_URL) ----

func rowOf(t *testing.T, pool *pgxpool.Pool, scope string, subject []byte) (hits int, locked bool) {
	t.Helper()
	err := pool.QueryRow(context.Background(),
		`SELECT hits, coalesce(locked_until > now(), false) FROM rate_limits WHERE scope = $1 AND subject_hash = $2`,
		scope, subject).Scan(&hits, &locked)
	if err != nil {
		t.Fatalf("read row: %v", err)
	}
	return hits, locked
}

func shift(t *testing.T, pool *pgxpool.Pool, d string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `UPDATE rate_limits SET
		window_started_at = window_started_at - $1::interval,
		locked_until = locked_until - $1::interval`, d); err != nil {
		t.Fatal(err)
	}
}

const window = 15 * time.Minute

func TestTheFifthFailureLocksAndALockedFailureNeitherCountsNorExtends(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	s := Subject(SignInEmail, "dealer@example.com")
	for i := 1; i <= 4; i++ {
		if err := Fail(ctx, pool, SignInEmail, s, 5, window, window); err != nil {
			t.Fatal(err)
		}
		if wait, err := Check(ctx, pool, SignInEmail, s); err != nil || wait != 0 {
			t.Fatalf("after %d failures: wait %v, %v", i, wait, err)
		}
	}
	if err := Fail(ctx, pool, SignInEmail, s, 5, window, window); err != nil {
		t.Fatal(err)
	}
	wait, err := Check(ctx, pool, SignInEmail, s)
	if err != nil || wait <= 14*time.Minute || wait > window {
		t.Fatalf("after the fifth failure the wait is %v, %v", wait, err)
	}
	var before time.Time
	if err := pool.QueryRow(ctx, `SELECT locked_until FROM rate_limits WHERE scope = $1`, SignInEmail).Scan(&before); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := Fail(ctx, pool, SignInEmail, s, 5, window, window); err != nil {
		t.Fatal(err)
	}
	var after time.Time
	if err := pool.QueryRow(ctx, `SELECT locked_until FROM rate_limits WHERE scope = $1`, SignInEmail).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !after.Equal(before) {
		t.Errorf("a refused failure moved the lock: %v -> %v", before, after)
	}
	if hits, locked := rowOf(t, pool, SignInEmail, s); hits != 5 || !locked {
		t.Errorf("hits %d locked %v, want 5 true", hits, locked)
	}
}

func TestAWindowAndALockThatEndedStartAgain(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	s := Subject(SignInAddress, "203.0.113.9")
	for i := 0; i < 4; i++ {
		if err := Fail(ctx, pool, SignInAddress, s, 5, window, window); err != nil {
			t.Fatal(err)
		}
	}
	shift(t, pool, "16 minutes") // the window is over, no lock was set
	if err := Fail(ctx, pool, SignInAddress, s, 5, window, window); err != nil {
		t.Fatal(err)
	}
	if hits, locked := rowOf(t, pool, SignInAddress, s); hits != 1 || locked {
		t.Fatalf("a new window: hits %d locked %v, want 1 false", hits, locked)
	}
	for i := 0; i < 4; i++ {
		if err := Fail(ctx, pool, SignInAddress, s, 5, window, window); err != nil {
			t.Fatal(err)
		}
	}
	if wait, _ := Check(ctx, pool, SignInAddress, s); wait == 0 {
		t.Fatal("five failures did not lock")
	}
	shift(t, pool, "14 minutes")
	if wait, _ := Check(ctx, pool, SignInAddress, s); wait == 0 || wait > time.Minute+time.Second {
		t.Errorf("14 minutes after the fifth failure the wait is %v, want under a minute", wait)
	}
	shift(t, pool, "2 minutes")
	if wait, _ := Check(ctx, pool, SignInAddress, s); wait != 0 {
		t.Errorf("16 minutes after the fifth failure the wait is %v, want none", wait)
	}
	if err := Fail(ctx, pool, SignInAddress, s, 5, window, window); err != nil {
		t.Fatal(err)
	}
	if hits, locked := rowOf(t, pool, SignInAddress, s); hits != 1 || locked {
		t.Errorf("after the lock ended: hits %d locked %v, want 1 false", hits, locked)
	}
}

func TestConcurrentFailuresAreCountedExactly(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	s := Subject(SignInEmail, "race@example.com")
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Fail(ctx, pool, SignInEmail, s, 5, window, window); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if hits, _ := rowOf(t, pool, SignInEmail, s); hits != 3 {
		t.Errorf("three concurrent failures counted %d", hits)
	}
}

func TestHitLetsTheLimitThroughAndRefusesTheNext(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	s := Subject("sign_up_address", "203.0.113.9")
	for i := 1; i <= 10; i++ {
		wait, err := Hit(ctx, pool, "sign_up_address", s, 10, time.Hour)
		if err != nil || wait != 0 {
			t.Fatalf("attempt %d: wait %v, %v", i, wait, err)
		}
	}
	for i := 11; i <= 13; i++ {
		if wait, err := Hit(ctx, pool, "sign_up_address", s, 10, time.Hour); err != nil || wait < time.Second || wait > time.Hour {
			t.Fatalf("attempt %d: wait %v, %v", i, wait, err)
		}
	}
	shift(t, pool, "61 minutes")
	if wait, err := Hit(ctx, pool, "sign_up_address", s, 10, time.Hour); err != nil || wait != 0 {
		t.Errorf("after the window: wait %v, %v", wait, err)
	}
	if hits, locked := rowOf(t, pool, "sign_up_address", s); hits != 1 || locked {
		t.Errorf("after the window: hits %d locked %v", hits, locked)
	}
}

func TestPurgeDeletesOldUnlockedCountersOnly(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	old := Subject(SignInEmail, "old@example.com")
	fresh := Subject(SignInEmail, "fresh@example.com")
	lockedOld := Subject(SignInEmail, "locked@example.com")
	for _, s := range [][]byte{old, fresh, lockedOld} {
		if err := Fail(ctx, pool, SignInEmail, s, 5, window, window); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE rate_limits SET window_started_at = now() - interval '2 days' WHERE subject_hash = ANY($1)`,
		[][]byte{old, lockedOld}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE rate_limits SET locked_until = now() + interval '5 minutes' WHERE subject_hash = $1`, lockedOld); err != nil {
		t.Fatal(err)
	}
	n, err := Purge(ctx, pool)
	if err != nil || n != 1 {
		t.Fatalf("purged %d, %v; want 1", n, err)
	}
	var left int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM rate_limits`).Scan(&left); err != nil || left != 2 {
		t.Errorf("%d rows left, %v; want 2", left, err)
	}
}
