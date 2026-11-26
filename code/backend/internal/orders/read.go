package orders

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
)

// viewSQL is the public view of an order: its columns are listed here and the
// statement does not select phone, email, address, note or is_sample, so no code
// that runs after it can return them. Each pair carries the shoe's live name,
// colourway and picture. The whole order and its pairs come from one statement,
// so they agree.
const viewSQL = `SELECT json_build_object(
    'id', o.id,
    'number', o.number,
    'placed_at', to_char(o.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD"T"HH24:MI:SS') || '+07:00',
    'status', o.status,
    'paid', o.paid,
    'total', o.total,
    'delivery_fee', o.delivery_fee,
    'delivery', o.delivery_option_id,
    'payment', o.payment_method_id,
    'carrier', o.carrier,
    'tracking', o.tracking,
    'eta', coalesce(to_char(o.eta, 'YYYY-MM-DD'), ''),
    'customer', o.customer,
    'city', o.city,
    'product', coalesce(f.product_id, ''),
    'size', coalesce(f.size, ''),
    'qty', coalesce(f.qty, 0),
    'items', coalesce((
        SELECT json_agg(json_build_object(
            'product', i.product_id, 'name', p.name, 'colourway', p.colourway, 'image', p.image,
            'size', i.size, 'qty', i.qty, 'unit_price', i.unit_price) ORDER BY i.line_no)
        FROM order_items i JOIN products p ON p.id = i.product_id
        WHERE i.order_id = o.id), '[]'::json)
)::text
FROM orders o
LEFT JOIN order_items f ON f.order_id = o.id AND f.line_no = 1
WHERE o.id = $1`

<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
// accessSQL is viewSQL for a caller who holds a way in: the order is found by its
// id AND either by the hash of a token it holds or by the customer account it is
// linked to. $2 is NULL without a token and $3 is NULL without a customer
// session, so "no token", "wrong token", "another customer's order" and "no such
// order" run the same statement and find nothing alike.
const accessSQL = viewSQL + `
  AND (EXISTS (SELECT 1 FROM order_tokens k WHERE k.order_id = o.id AND k.token_hash = $2)
       OR (o.account_id IS NOT NULL AND o.account_id = $3::uuid))`

// Querier is the slice of a pool or a transaction a read needs.
type Querier interface {
=======
type querier interface {
>>>>>>> refs/heads/main
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// readView is the public view of the order with that id, for code that has
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
// already been let in (the order was just saved, or its key just replayed).
func readView(ctx context.Context, q Querier, id string) (json.RawMessage, error) {
=======
// already been let in (authorize found it, the order was just saved, or its key
// just replayed).
func readView(ctx context.Context, q querier, id string) (json.RawMessage, error) {
>>>>>>> refs/heads/main
	var body string
	if err := q.QueryRow(ctx, viewSQL, id).Scan(&body); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
// Access is who asks for an order and what they hold: the order's token, or the
// id of the customer account the order may be linked to. An empty field is none.
// AccountID is set only from a customer's session, never from the request.
type Access struct{ Token, AccountID string }

// ErrNotFound is the one answer to every way of not being let in.
var ErrNotFound = errors.New("orders: order not found")

// Read returns the public view of the order, to a caller who holds its token or
// is the customer it is linked to. An id, a token or an account id that cannot be
// one never reaches the database: a caller with neither a well-formed token nor
// an account (a guest with no token) costs no query. Every other way of not being
// let in, an order that does not exist included, is ErrNotFound.
func Read(ctx context.Context, q Querier, id string, a Access) (json.RawMessage, error) {
	hasToken, hasAccount := tokenPattern.MatchString(a.Token), auth.IsID(a.AccountID)
	if len(id) > 100 || !idPattern.MatchString(id) || (!hasToken && !hasAccount) {
		return nil, ErrNotFound
=======
// snapshot lets the caller in and reads the order's view inside one read-only
// REPEATABLE READ transaction, so the access decision and the view are of the
// same moment.
func (s *service) snapshot(ctx context.Context, id string, a Access) (json.RawMessage, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ref, err := authorize(ctx, tx, id, a, modeRead)
	if err != nil {
		return nil, err
	}
	body, err := readView(ctx, tx, ref.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return body, nil
}

// read answers GET /v1/orders/{id}. The order's X-Order-Token is the only
// credential in this story. An id or a token that cannot be one never reaches the
// database, and every way of not being let in answers the same 404.
func (s *service) read(w http.ResponseWriter, r *http.Request) {
	id, access := r.PathValue("id"), Access{Token: r.Header.Get("X-Order-Token")}
	if !access.couldOpen(id) {
		notFound(w)
		return
>>>>>>> refs/heads/main
	}
	var token, account any
	if hasToken {
		token = hashToken(a.Token)
	}
	if hasAccount {
		account = a.AccountID
	}
	var body string
	err := q.QueryRow(ctx, accessSQL, id, token, account).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

// read answers GET /v1/orders/{id}: to the order's X-Order-Token, or to the
// customer the order is linked to, and with the same 404 to everyone else.
func (s *service) read(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), txTimeout)
	defer cancel()
<<<<<<< refs/heads/feature/customer-accounts-sign-up-sign-in-my-orders-be
	body, err := Read(ctx, s.db, r.PathValue("id"), Access{Token: r.Header.Get("X-Order-Token"), AccountID: callerAccount(r)})
	if errors.Is(err, ErrNotFound) {
		WriteNotFound(w)
		return
	}
=======
	body, err := s.snapshot(ctx, id, access)
>>>>>>> refs/heads/main
	if err != nil {
		s.fail(w, r, id, err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, body)
}
