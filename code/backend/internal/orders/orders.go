// Package orders places an order and reads it back by its own token.
//
// An order holds a customer's name, phone and address, so it is no public
// collection: GET /v1/orders answers 401 to everyone until the dealer signs in,
// and GET /v1/orders/{id} answers the public view (never phone, email, address or
// note) to the order's X-Order-Token and 404 to everything else.
//
// Placing goes through one transaction that takes the pairs off the shelf and
// saves the order, its pairs and a hashed token together. The lock order, which
// every later piece of code that locks several of these must keep, is: the
// advisory lock on the Idempotency-Key, the shoes (FOR SHARE, by id), the counts
// of pairs (FOR UPDATE, by shoe and size), the order-number counter, the inserts.
//
// Routes carry no /api prefix: the edge proxy strips it.
package orders

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
)

const (
	// standardDelivery is the one delivery option. Delivery is free on every
	// order: the stored fee is the constant 0 and delivery_options.fee is never read.
	standardDelivery = "standard"
	// maxBody is the largest request body read.
	maxBody = 64 << 10
	// txTimeout bounds the whole placing transaction.
	txTimeout = 5 * time.Second
	// providerTimeout bounds the ask for a payment address.
	providerTimeout = 3 * time.Second
)

// offered is the ordered list of ways to pay the shop offers. What is offered is
// code, never data: cod, bank and momo keep their rows in payment_methods and are
// never listed or accepted.
var offered = []string{"vnpay"}

// Provider is one way to pay. The consumer (this package) declares it, so a
// provider package satisfies it without importing orders.
type Provider interface {
	// Available says whether the shop can take a payment through this provider now.
	Available() bool
	// PaymentURL returns the address the customer pays at. An error means that none
	// could be given.
	PaymentURL(ctx context.Context, o Payable, c Client) (string, error)
}

// Payable is what a provider is told about an order: read from the saved order,
// never from the browser.
type Payable struct {
	ID, Number string
	Total      int64
	PlacedAt   time.Time
}

// Client is where the customer is: the right-most X-Forwarded-For address and the
// public origin of the site.
type Client struct{ IP, Origin string }

// Options are what the server passes in. The zero value is a shop with no
// provider and the real clock.
type Options struct {
	// Providers is keyed by payment_methods.id.
	Providers map[string]Provider
	// Now is the clock; time.Now when nil.
	Now func() time.Time
}

// DB is the slice of *pgxpool.Pool this package uses.
type DB interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type service struct {
	db        DB
	log       *slog.Logger
	providers map[string]Provider
	now       func() time.Time
}

// Register mounts the four routes of the story on mux.
func Register(mux *http.ServeMux, db DB, log *slog.Logger, opts Options) {
	s := &service{db: db, log: log, providers: opts.Providers, now: opts.Now}
	if s.now == nil {
		s.now = time.Now
	}
	mux.HandleFunc("POST /v1/orders", s.create)
	mux.HandleFunc("GET /v1/orders/{id}", s.read)
	mux.HandleFunc("GET /v1/orders", s.list)
	mux.HandleFunc("GET /v1/payment-options", s.paymentOptions)
}

// list is the dealer's list. No dealer session can exist before the sign-in
// story, so it answers 401 to every request, with any query string, before the
// query string is read. The sign-in story puts its guard here.
func (s *service) list(w http.ResponseWriter, _ *http.Request) {
	httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Sign in to see orders.", nil)
}

func (s *service) internal(w http.ResponseWriter, r *http.Request, what string, err error) {
	// The cause is logged beside the request id and never returned: it can hold
	// SQL or a host name.
	s.log.Error(what, "request_id", httpx.RequestIDFrom(r.Context()), "error", err)
	httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.", nil)
}

func notFound(w http.ResponseWriter) {
	httpx.Error(w, http.StatusNotFound, "not_found", "Not found.", nil)
}

func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }
