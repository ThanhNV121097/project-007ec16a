// Package account is the customer's side of the shop: signing up, the saved
// delivery details, "My orders" and adding a guest order to the account.
//
// It holds NO SQL and imports no database driver. It reads and writes users only
// through internal/auth and orders only through internal/orders, so each table
// keeps its one writer. No route takes an account id: every handler works on the
// session's own account, read from the request context, which only a signed-in
// customer's session can fill (auth.Require(auth.Customer) runs before the body).
//
// Routes carry no /api prefix: the edge proxy strips it.
package account

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

// DB is what the package needs from the pool: a querier that can open a
// transaction for auth, and the orders package's own slice.
type DB interface {
	auth.Beginner
	orders.DB
}

// Deps are what Register needs.
type Deps struct {
	Auth *auth.Service
	DB   DB
	Log  *slog.Logger
}

type handler struct {
	auth *auth.Service
	db   DB
	log  *slog.Logger
}

// requestTimeout bounds one request's statements.
const requestTimeout = 10 * time.Second

const internalMessage = "Something went wrong. Please try again."

// Register mounts the six routes. Every pattern is a literal, so a wrong method
// falls to the server's catch-all and answers the JSON 404.
func Register(mux *http.ServeMux, d Deps) {
	h := &handler{auth: d.Auth, db: d.DB, log: d.Log}
	customer := auth.Require(auth.Customer)
	mux.HandleFunc("POST /v1/account/sign-up", h.signUp)
	mux.Handle("GET /v1/account", customer(http.HandlerFunc(h.profile)))
	mux.Handle("PUT /v1/account", customer(http.HandlerFunc(h.saveProfile)))
	mux.Handle("GET /v1/account/orders", customer(http.HandlerFunc(h.listOrders)))
	mux.Handle("GET /v1/account/orders/{id}", customer(http.HandlerFunc(h.oneOrder)))
	mux.Handle("POST /v1/account/orders", customer(http.HandlerFunc(h.addOrder)))
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

// internal answers a failed statement with the generic 500 and nothing of the
// cause, which goes to the log by route (a database error by SQLSTATE and
// constraint name only: its detail holds the row, an email included).
func (h *handler) internal(w http.ResponseWriter, r *http.Request, err error) {
	auth.LogFailure(h.log, "account request failed", r.Pattern, err)
	refuse(w, http.StatusInternalServerError, "internal", internalMessage, nil)
}

// caller is the signed-in customer. Require has already run, so a missing user
// is a wiring defect; it is answered as no session.
func caller(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	u, ok := auth.FromContext(r.Context())
	if !ok || u.Role != auth.Customer {
		refuse(w, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.", nil)
		return auth.User{}, false
	}
	return u, true
}

// withTimeout is the request's context bounded by requestTimeout.
func withTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), requestTimeout)
}

// retryAfter sets Retry-After to the wait in whole seconds, from 1 to the
// ceiling, and returns it.
func retryAfter(w http.ResponseWriter, wait, ceiling time.Duration) int {
	secs := min(max(int(math.Ceil(wait.Seconds())), 1), int(ceiling.Seconds()))
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	return secs
}
