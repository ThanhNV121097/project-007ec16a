// Package ratelimit keeps the "too many tries" counters. It is the only writer
// of the rate_limits table: a subject (a lower-cased email, a network address)
// is stored as a hash, so the table holds no email and no address.
//
// The clock is always the database's now(): the code never compares times
// itself, so a test moves stored timestamps instead of waiting.
package ratelimit

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Scopes of rate_limits.scope (ck_rate_limits_scope).
const (
	SignInEmail   = "sign_in_email"
	SignInAddress = "sign_in_address"
)

// Querier is the slice of a pool or a transaction the counters need.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Subject is the 32-byte key of a subject inside a scope: sha256(scope ‖ ":" ‖ subject).
func Subject(scope, subject string) []byte {
	sum := sha256.Sum256([]byte(scope + ":" + subject))
	return sum[:]
}

// Check says how long the subject is locked for: zero when it is not. A locked
// subject has at least one second left and at most its lock.
func Check(ctx context.Context, q Querier, scope string, subject []byte) (time.Duration, error) {
	var secs int
	err := q.QueryRow(ctx, `SELECT ceil(extract(epoch FROM locked_until - now()))::int
		FROM rate_limits WHERE scope = $1 AND subject_hash = $2 AND locked_until > now()`,
		scope, subject).Scan(&secs)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return time.Duration(max(secs, 1)) * time.Second, nil
}

// failSQL is one atomic upsert, so concurrent failures are counted exactly (a
// SELECT then an UPDATE would lose some). In an UPDATE every right-hand side
// sees the old row, which is what the CASEs rely on. $3 is the limit, $4 the
// window and $5 the lock, both in seconds.
const failSQL = `INSERT INTO rate_limits AS r (scope, subject_hash, hits, window_started_at, locked_until)
VALUES ($1, $2, 1, now(), CASE WHEN $3 <= 1 THEN now() + make_interval(secs => $5) END)
ON CONFLICT (scope, subject_hash) DO UPDATE SET
  hits = CASE
    WHEN r.locked_until > now() THEN r.hits
    WHEN r.window_started_at <= now() - make_interval(secs => $4) OR r.locked_until <= now() THEN 1
    ELSE r.hits + 1 END,
  window_started_at = CASE
    WHEN r.locked_until > now() THEN r.window_started_at
    WHEN r.window_started_at <= now() - make_interval(secs => $4) OR r.locked_until <= now() THEN now()
    ELSE r.window_started_at END,
  locked_until = CASE
    WHEN r.locked_until > now() THEN r.locked_until
    WHEN r.window_started_at <= now() - make_interval(secs => $4) OR r.locked_until <= now()
         THEN CASE WHEN $3 <= 1 THEN now() + make_interval(secs => $5) END
    WHEN r.hits + 1 >= $3 THEN now() + make_interval(secs => $5)
    ELSE NULL END`

// Fail counts one failed try. The limit-th failure inside the window starts a
// lock of the given length; a failure while locked counts for nothing and
// never extends the lock.
func Fail(ctx context.Context, q Querier, scope string, subject []byte, limit int, window, lock time.Duration) error {
	_, err := q.Exec(ctx, failSQL, scope, subject, limit, window.Seconds(), lock.Seconds())
	return err
}

// hitSQL counts one attempt, whatever its outcome. The attempt after the
// limit-th inside the window is refused and locks the subject until the
// window's end. $3 is the limit, $4 the window in seconds.
const hitSQL = `INSERT INTO rate_limits AS r (scope, subject_hash, hits, window_started_at, locked_until)
VALUES ($1, $2, 1, now(), CASE WHEN 1 > $3 THEN now() + make_interval(secs => $4) END)
ON CONFLICT (scope, subject_hash) DO UPDATE SET
  hits = CASE
    WHEN r.window_started_at <= now() - make_interval(secs => $4) THEN 1
    WHEN r.locked_until > now() THEN r.hits
    ELSE r.hits + 1 END,
  window_started_at = CASE
    WHEN r.window_started_at <= now() - make_interval(secs => $4) THEN now()
    ELSE r.window_started_at END,
  locked_until = CASE
    WHEN r.window_started_at <= now() - make_interval(secs => $4) THEN NULL
    WHEN r.locked_until > now() THEN r.locked_until
    WHEN r.hits + 1 > $3 THEN r.window_started_at + make_interval(secs => $4)
    ELSE NULL END
RETURNING coalesce(locked_until > now(), false)`

// Hit counts one attempt (the sign-up counter) and says whether this call is
// refused: the limit-th attempt in the window is let through, the next one and
// every one after it until the window ends is refused.
func Hit(ctx context.Context, q Querier, scope string, subject []byte, limit int, window time.Duration) (refused bool, err error) {
	err = q.QueryRow(ctx, hitSQL, scope, subject, limit, window.Seconds()).Scan(&refused)
	return refused, err
}

// Purge deletes the counters nobody needs any more: a window a day old with no
// lock still running.
func Purge(ctx context.Context, q Querier) (int64, error) {
	tag, err := q.Exec(ctx, `DELETE FROM rate_limits
		WHERE window_started_at < now() - interval '1 day' AND (locked_until IS NULL OR locked_until < now())`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
