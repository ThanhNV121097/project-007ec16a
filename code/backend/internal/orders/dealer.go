package orders

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
)

// The dealer's transaction_no is VNPay's transaction number of the order's one
// success notice, "" when there is none (erd/dealer-handles-orders-status-courier-payment.md
// section 6). It is derived, never a column of orders: a read-only scalar
// subquery over the payments story's two tables, written once here with the
// order's alias as its only blank and used by the dealer's single read (alias o)
// and by the team's list (alias t). This package reads those tables and imports
// nothing from payments, so the import graph stays payments -> orders. The
// ORDER BY ... LIMIT 1 is there so that a second provider can never make the
// subquery return two rows.
const (
	transactionNoHead = `coalesce((SELECT n.transaction_no FROM payment_notices n JOIN payment_attempts a ON a.id = n.attempt_id ` +
		`WHERE a.order_id = `
	transactionNoTail = `.id AND n.is_success ORDER BY n.received_at LIMIT 1), '')`

	// transactionNoOfOrder is the number in the single read, where the order is o.
	transactionNoOfOrder = transactionNoHead + `o` + transactionNoTail
	// transactionNoOfListRow is the number in the list, where the order is t.
	transactionNoOfListRow = transactionNoHead + `t` + transactionNoTail
)

// dealerViewSQL is the dealer's view of one order: the public view's keys and
// phone, email, address, note and transaction_no. It is a statement of its own,
// beside the public viewSQL and not derived from it, so the public statement
// can never gain a private column. It never selects is_sample, idempotency_key,
// account_id, source or invented.
const dealerViewSQL = `SELECT json_build_object(
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
        WHERE i.order_id = o.id), '[]'::json),
    'phone', o.phone,
    'email', o.email,
    'address', o.address,
    'note', o.note,
    'transaction_no', ` + transactionNoOfOrder + `
)::text
FROM orders o
LEFT JOIN order_items f ON f.order_id = o.id AND f.line_no = 1
WHERE o.id = $1`

// ReadDealer is the one reader of the dealer view: GET /v1/orders/{id} and the
// answer of PUT /v1/orders/{id} both call it, so they cannot differ. ErrNotFound
// when there is no such order.
func ReadDealer(ctx context.Context, q querier, id string) (json.RawMessage, error) {
	var body string
	err := q.QueryRow(ctx, dealerViewSQL, id).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

// readDealer answers GET /v1/orders/{id} to a dealer's session, whatever token
// the request carries. It sits outside authorize on purpose: authorize hands
// back a reference that can never hold a private column, and that is what the
// visitor's guarantee stands on.
func (s *service) readDealer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !idPattern.MatchString(id) || len(id) > 100 {
		notFound(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	body, err := ReadDealer(ctx, s.db, id)
	if errors.Is(err, ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		s.internal(w, r, "read order for the dealer", err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, body)
}
