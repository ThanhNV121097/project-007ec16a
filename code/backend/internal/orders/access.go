package orders

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	idPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
)

// Access says who is asking to open an order. Today that is only a token; the
// dealer and accounts stories add their fields here and extend authorize, the one
// function that decides, and nothing else.
type Access struct{ Token string }

// mode is what the caller wants to do with the order. Today the two behave alike;
// the dealer's way in (no token) will be modeRead only, never modePay.
type mode int

const (
	modeRead mode = iota
	modePay
)

// Ref is the order's row exactly as the access statement returns it: the public
// columns, never phone, email, address, note, is_sample, idempotency_key or
// account_id.
type Ref struct {
	ID, Number, Status, Delivery, Payment, Carrier, Tracking, Customer, City string
	PlacedAt                                                                  time.Time
	Paid                                                                      bool
	Total, DeliveryFee                                                        int64
	ETA                                                                       *time.Time
}

// ErrNotFound is the one answer for every way of not being let in: a bad id, no
// token, a token that opens another order, an order that does not exist.
var ErrNotFound = errors.New("order not found")

// accessSQL finds the order by its id AND by the hash of the token, so "no token",
// "wrong token", "another order's token" and "no such order" run the same
// statement and find nothing alike. Its column list is fixed here: no code that
// runs after it can return a private column. It is the one read statement in this
// package that names order_tokens.
const accessSQL = `SELECT o.id, o.number, o.placed_at, o.status, o.paid, o.total, o.delivery_fee,
       o.delivery_option_id, o.payment_method_id, o.carrier, o.tracking, o.eta,
       o.customer, o.city
FROM orders o
WHERE o.id = $1
  AND EXISTS (SELECT 1 FROM order_tokens t WHERE t.order_id = o.id AND t.token_hash = $2)`

// couldOpen says whether this id and this access can possibly name an order. A
// request that cannot is answered 404 with no query made.
func (a Access) couldOpen(id string) bool {
	return len(id) <= 100 && idPattern.MatchString(id) && tokenPattern.MatchString(a.Token)
}

// authorize is the one function that decides who may open an order: GET
// /v1/orders/{id} and POST /v1/orders/{id}/payment both call it. The token is
// hashed here and only the hash is a parameter, so the secret is never compared,
// never a SQL parameter and never logged. ErrNotFound for every refusal; any other
// error is the database's and is a 500.
func authorize(ctx context.Context, q querier, id string, a Access, _ mode) (Ref, error) {
	if !a.couldOpen(id) {
		return Ref{}, ErrNotFound
	}
	var r Ref
	err := q.QueryRow(ctx, accessSQL, id, hashToken(a.Token)).Scan(
		&r.ID, &r.Number, &r.PlacedAt, &r.Status, &r.Paid, &r.Total, &r.DeliveryFee,
		&r.Delivery, &r.Payment, &r.Carrier, &r.Tracking, &r.ETA, &r.Customer, &r.City)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ref{}, ErrNotFound
	}
	if err != nil {
		return Ref{}, fmt.Errorf("open order: %w", err)
	}
	return r, nil
}
