// Package auth is who is asking: accounts, sessions, sign-in and the two
// pieces of middleware every protected route uses.
//
// Resolve runs on every request under /v1, reads the session cookie and puts
// the signed-in user (or nobody) in the request context. Require(roles...) then
// answers 401 with no session and 403 for another role; it reads only the
// context, so a Require mounted where no Resolve ran answers 401 to everyone.
//
// "Dealer" is Require(auth.Admin, auth.Staff); account management is
// Require(auth.Admin).
//
// This package and internal/ratelimit are the only code that writes the users,
// sessions and rate_limits tables.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Role is what an account may do.
type Role string

const (
	Admin    Role = "admin"
	Staff    Role = "staff"
	Customer Role = "customer"
)

// User is a signed-in account as Resolve and CreateUser return it. Name is the
// stored name, possibly empty (the admin made from the settings has none);
// Active is true on every row either of them returns.
type User struct {
	ID, Email, Name string
	Role            Role
	Active          bool
}

// DisplayName is the name shown for the account: see the package function.
func (u User) DisplayName() string { return DisplayName(u.Name, u.Email) }

type userKey struct{}

// WithUser returns ctx with u as the signed-in user. Resolve is the only code
// that does this outside a test.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

// FromContext returns the user Resolve found for the request, if any.
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey{}).(User)
	return u, ok
}

// Querier is the slice of *pgxpool.Pool (or of a transaction) the package needs.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Beginner is a Querier that can open a transaction.
type Beginner interface {
	Querier
	Begin(ctx context.Context) (pgx.Tx, error)
}

const (
	internalMessage = "Something went wrong. Please try again."
	lookupTimeout   = 3 * time.Second
)

// Service holds what sign-in needs besides the database: the dummy hash every
// attempt without a usable account is verified against, and the verifier.
type Service struct {
	db    Querier
	log   *slog.Logger
	dummy string
	// verify reports whether plain matches the stored hash. A field so a test
	// can count the verifications an attempt costs.
	verify func(ctx context.Context, encoded, plain string) (bool, error)
}

// New makes the service. The dummy hash is made here, at start-up, so the
// first unknown email does not pay for making it.
func New(db Querier, log *slog.Logger) *Service {
	s := &Service{db: db, log: log, dummy: makeDummy()}
	s.verify = s.verifyPassword
	return s
}

// reply writes a JSON answer that must never be cached.
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, status, v)
}

// refuse writes the error envelope, never cached.
func refuse(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.Error(w, status, code, message, fields)
}

// logFailure logs a failed statement. A constraint error is logged by its
// SQLSTATE and constraint name only: Postgres puts the offending value (an
// email) in the error's detail.
func logFailure(log *slog.Logger, msg, route string, err error) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		log.Error(msg, "route", route, "sqlstate", pg.Code, "constraint", pg.ConstraintName)
		return
	}
	log.Error(msg, "route", route, "error", err)
}

// constraintOf returns the name of the unique constraint a statement broke, or
// "" when it broke none.
func constraintOf(err error) string {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return pg.ConstraintName
	}
	return ""
}

// Resolve puts the signed-in user in the request context. It reads the cookie
// only under /v1 (so /healthz never depends on the database) and only when the
// cookie is well formed. A lookup that fails is a 500 for every route: the
// person may well be signed in, so "not signed in" would be a lie. Resolve
// never denies by role; Require does.
func (s *Service) Resolve() httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/v1/") {
				next.ServeHTTP(w, r)
				return
			}
			hash, ok := cookieHash(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), lookupTimeout)
			u, found, err := s.lookup(ctx, hash)
			cancel()
			if err != nil {
				logFailure(s.log, "session lookup failed", r.Method+" "+r.URL.Path[:min(len(r.URL.Path), 0)], err)
				refuse(w, http.StatusInternalServerError, "internal", internalMessage, nil)
				return
			}
			if found {
				httpx.TraceFrom(r.Context()).Role = string(u.Role)
				r = r.WithContext(WithUser(r.Context(), u))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// lookupSQL is Q1: the account behind a live session. A session exactly seven
// days old has expires_at = now() and is not > now(), so it is expired.
const lookupSQL = `SELECT u.id::text, u.email, u.name, u.role
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now() AND u.is_active`

func (s *Service) lookup(ctx context.Context, tokenHash []byte) (User, bool, error) {
	var u User
	var role string
	err := s.db.QueryRow(ctx, lookupSQL, tokenHash).Scan(&u.ID, &u.Email, &u.Name, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	u.Role, u.Active = Role(role), true
	return u, true, nil
}

// Require lets through only a signed-in user whose role is listed: 401 with no
// session, 403 for another role. Both answers are the bare envelope.
func Require(roles ...Role) httpx.Middleware {
	allowed := make(map[Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := FromContext(r.Context())
			if !ok {
				refuse(w, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.", nil)
				return
			}
			if !allowed[u.Role] {
				refuse(w, http.StatusForbidden, "forbidden", "You do not have access to this.", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
