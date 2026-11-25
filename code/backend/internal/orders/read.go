package orders

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
)

var (
	idPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
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

// tokenSQL is viewSQL for a caller who holds a token: the order is found by its
// id AND by the hash of the token, so "no token", "wrong token", "another order's
// token" and "no such order" run the same statement and find nothing alike.
const tokenSQL = viewSQL + `
  AND EXISTS (SELECT 1 FROM order_tokens k WHERE k.order_id = o.id AND k.token_hash = $2)`

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// readView is the public view of the order with that id, for code that has
// already been let in (the order was just saved, or its key just replayed).
func readView(ctx context.Context, q querier, id string) (json.RawMessage, error) {
	var body string
	if err := q.QueryRow(ctx, viewSQL, id).Scan(&body); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

// read answers GET /v1/orders/{id}. The order's X-Order-Token is the only
// credential in this story. An id or a token that cannot be one never reaches the
// database, and every way of not being let in answers the same 404.
func (s *service) read(w http.ResponseWriter, r *http.Request) {
	id, token := r.PathValue("id"), r.Header.Get("X-Order-Token")
	if len(id) > 100 || !idPattern.MatchString(id) || !tokenPattern.MatchString(token) {
		notFound(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), txTimeout)
	defer cancel()
	var body string
	err := s.db.QueryRow(ctx, tokenSQL, id, hashToken(token)).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w)
		return
	}
	if err != nil {
		s.internal(w, "read order", err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, json.RawMessage(body))
}
