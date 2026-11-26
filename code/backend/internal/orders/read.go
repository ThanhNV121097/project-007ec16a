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
<<<<<<< refs/heads/feature/dealer-handles-orders-status-courier-payment-be
	if u, ok := auth.FromContext(r.Context()); ok && (u.Role == auth.Admin || u.Role == auth.Staff) {
		s.readDealer(w, r)
		return
	}
	id, token := r.PathValue("id"), r.Header.Get("X-Order-Token")
	if len(id) > 100 || !idPattern.MatchString(id) || !tokenPattern.MatchString(token) {
=======
	id, access := r.PathValue("id"), Access{Token: r.Header.Get("X-Order-Token")}
	if !access.couldOpen(id) {
>>>>>>> refs/heads/main
		notFound(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), txTimeout)
	defer cancel()
	body, err := s.snapshot(ctx, id, access)
	if err != nil {
		s.fail(w, r, id, err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, body)
}
