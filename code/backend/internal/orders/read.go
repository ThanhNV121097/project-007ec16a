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

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// readView is the public view of the order with that id, for code that has
// already been let in (authorize found it, the order was just saved, or its key
// just replayed).
func readView(ctx context.Context, q querier, id string) (json.RawMessage, error) {
	var body string
	if err := q.QueryRow(ctx, viewSQL, id).Scan(&body); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

// Read returns the public view of the order to a caller who holds its token or is
// the customer it is linked to, inside one read-only REPEATABLE READ transaction,
// so the access decision and the view are of the same moment. An id, a token or an
// account id that cannot be one never reaches the database: a caller with neither
// a well-formed token nor an account costs no transaction. Every other way of not
// being let in, an order that does not exist included, is ErrNotFound.
func Read(ctx context.Context, db DB, id string, a Access) (json.RawMessage, error) {
	if !a.couldOpen(id) {
		return nil, ErrNotFound
	}
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
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

// read answers GET /v1/orders/{id}: a dealer session gets the dealer view; the
// order's X-Order-Token, or the customer the order is linked to, gets the public
// view; everyone else gets the same 404.
func (s *service) read(w http.ResponseWriter, r *http.Request) {
	if u, ok := auth.FromContext(r.Context()); ok && (u.Role == auth.Admin || u.Role == auth.Staff) {
		s.readDealer(w, r)
		return
	}
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), txTimeout)
	defer cancel()
	body, err := Read(ctx, s.db, id, Access{Token: r.Header.Get("X-Order-Token"), AccountID: callerAccount(r)})
	if err != nil {
		s.fail(w, r, id, err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, body)
}
