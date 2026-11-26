package orders

import (
	"context"
	"errors"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/stock"
	"github.com/jackc/pgx/v5"
)

// ErrNotFound is an order that does not exist.
var ErrNotFound = errors.New("orders: no such order")

// Locked is one order's row as it is read under its lock: what the rules of a
// move, a cancel and a delete judge, and nothing more.
type Locked struct {
	ID, Number, Status, Payment string
	Paid                        bool
	Total                       int64
	PlacedAt                    time.Time
}

const lockOrderSQL = `SELECT id, number, status, paid, total, placed_at, payment_method_id FROM orders WHERE id = $1 FOR UPDATE`

// LockOrder reads one order's row FOR UPDATE, inside the caller's transaction:
// the one function every writer of an order takes first (before any count of
// pairs), so a cancel, a save, a delete and a payment notice queue on the row.
// ErrNotFound when there is none.
func LockOrder(ctx context.Context, tx pgx.Tx, id string) (Locked, error) {
	var l Locked
	err := tx.QueryRow(ctx, lockOrderSQL, id).Scan(&l.ID, &l.Number, &l.Status, &l.Paid, &l.Total, &l.PlacedAt, &l.Payment)
	if errors.Is(err, pgx.ErrNoRows) {
		return Locked{}, ErrNotFound
	}
	return l, err
}

// CancelRule narrows a cancel. The zero value is the dealer's: any order that is
// placed, confirmed or packed. The 30-minute expiry of unpaid online orders
// (the payments story) adds its own four terms by naming the ways to pay it
// gives up on and the moment before which an order must have been placed.
type CancelRule struct {
	PlacedBefore time.Time
	Ways         []string
}

const cancelSQL = `UPDATE orders SET status = 'cancelled'
WHERE id = $1 AND status IN ('placed','confirmed','packed')
  AND ($2::text[] IS NULL OR (NOT paid AND NOT is_sample AND payment_method_id = ANY($2) AND placed_at <= $3))
RETURNING is_sample`

const pairsSQL = `SELECT product_id, size, sum(qty)::integer FROM order_items
WHERE order_id = $1 GROUP BY product_id, size ORDER BY product_id, size`

// Cancel moves one order to cancelled and puts its pairs back on the shelf, ONCE,
// inside the caller's transaction. changed=false means the order was not in a
// state this cancel may act on (already cancelled, shipped, delivered, or out of
// the rule) and nothing was touched. The status test is in the UPDATE's own
// WHERE, so two cancels, or a cancel and the expiry, serialise on the order's
// row and exactly one of them sees a row change.
//
// The 14 sample orders took nothing from the shelf, so cancelling one moves the
// status and returns no pair. For every other order the counts are locked with
// stock.Lock (the call checkout makes, in the same (shoe, size) order) before
// stock.Return adds to them.
func Cancel(ctx context.Context, tx pgx.Tx, id string, rule CancelRule) (changed bool, err error) {
	var ways []string
	if len(rule.Ways) > 0 {
		ways = rule.Ways
	}
	var sample bool
	switch err := tx.QueryRow(ctx, cancelSQL, id, ways, rule.PlacedBefore).Scan(&sample); {
	case errors.Is(err, pgx.ErrNoRows):
		return false, nil
	case err != nil:
		return false, err
	}
	if sample {
		return true, nil
	}
	rows, err := tx.Query(ctx, pairsSQL, id)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var lines []stock.Line
	var keys []stock.Key
	for rows.Next() {
		var l stock.Line
		if err := rows.Scan(&l.Product, &l.Size, &l.Qty); err != nil {
			return false, err
		}
		lines = append(lines, l)
		keys = append(keys, l.Key)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	rows.Close()
	if _, err := stock.Lock(ctx, tx, keys); err != nil {
		return false, err
	}
	if err := stock.Return(ctx, tx, lines); err != nil {
		return false, err
	}
	return true, nil
}
