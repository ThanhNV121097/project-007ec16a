package orders

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

var (
	idPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
)

// Access says who is asking to open an order and what they hold: the order's
// token, or the id of the customer account the order may be linked to. An empty
// field is none. AccountID is set only from a customer's session, never from the
// request. The dealer story adds its field here and extends authorize, the one
// function that decides, and nothing else.
type Access struct{ Token, AccountID string }

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

// The access statement finds the order by its id AND by a way in. Its column list
// is fixed in accessHead: no code that runs after it can return a private column.
// holdsToken is the one text in this package's read statements that names
// order_tokens.
const (
	accessHead = `SELECT o.id, o.number, o.placed_at, o.status, o.paid, o.total, o.delivery_fee,
       o.delivery_option_id, o.payment_method_id, o.carrier, o.tracking, o.eta,
       o.customer, o.city
FROM orders o
WHERE o.id = $1
  AND `
	holdsToken = `EXISTS (SELECT 1 FROM order_tokens t WHERE t.order_id = o.id AND t.token_hash = $2)`
)

// accessSQL is the statement of a caller who holds a token and no customer
// session: the order and the hash of its token, two parameters, as the order page
// was built.
const accessSQL = accessHead + holdsToken

// accessAccountSQL is the statement of a signed-in customer: the same, with the
// third way in, the customer account the order is linked to. $2 is NULL without a
// valid token, so "no token", "wrong token", "another order's token", "another
// customer's order" and "no such order" run the same statement and find nothing
// alike.
const accessAccountSQL = accessHead + `(` + holdsToken + `
       OR (o.account_id IS NOT NULL AND o.account_id = $3::uuid))`

// couldOpen says whether this id and this access can possibly name an order: the
// id has the shape of one and the caller holds a well-formed token or an account
// id. A request that cannot is answered 404 with no query made, so a guest with no
// token costs none.
func (a Access) couldOpen(id string) bool {
	return len(id) <= 100 && idPattern.MatchString(id) && (a.hasToken() || a.hasAccount())
}

func (a Access) hasToken() bool   { return tokenPattern.MatchString(a.Token) }
func (a Access) hasAccount() bool { return auth.IsID(a.AccountID) }

// authorize is the one function that decides who may open an order: GET
// /v1/orders/{id}, GET /v1/account/orders/{id} and POST /v1/orders/{id}/payment
// all call it, and the customer's way in holds for modeRead and modePay alike. The
// token is hashed here and only the hash is a parameter, so the secret is never
// compared, never a SQL parameter and never logged. ErrNotFound for every refusal;
// any other error is the database's and is a 500.
func authorize(ctx context.Context, q querier, id string, a Access, _ mode) (Ref, error) {
	if !a.couldOpen(id) {
		return Ref{}, ErrNotFound
	}
	var row pgx.Row
	if a.hasAccount() {
		var token any
		if a.hasToken() {
			token = hashToken(a.Token)
		}
		row = q.QueryRow(ctx, accessAccountSQL, id, token, a.AccountID)
	} else {
		// couldOpen held, so a caller with no account holds a well-formed token.
		row = q.QueryRow(ctx, accessSQL, id, hashToken(a.Token))
	}
	var r Ref
	err := row.Scan(
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
