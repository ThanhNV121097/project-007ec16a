package orders

import (
	"context"
	"errors"
<<<<<<< refs/heads/feature/dealer-handles-orders-status-courier-payment-be
=======
	"fmt"
	"slices"
>>>>>>> refs/heads/main
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/stock"
	"github.com/jackc/pgx/v5"
<<<<<<< refs/heads/feature/dealer-handles-orders-status-courier-payment-be
)

// ErrNotFound is an order that does not exist.
var ErrNotFound = errors.New("orders: no such order")

// Locked is one order's row as it is read under its lock: what the rules of a
// move, a cancel and a delete judge, and nothing more.
=======
	"github.com/jackc/pgx/v5/pgconn"
)

// This file is what the payments package calls so that it never writes the
// orders table itself: the order's row, its paid flag and its cancel have one
// writer each, here. Each function runs inside the caller's transaction.

// PayWindow is how long an unpaid order may wait, counted from placed_at. The
// route's refusal, payment_attempts.expires_at, the provider's expiry and the
// 30-minute rule all use this one value.
const PayWindow = 30 * time.Minute

// MsgRunOut is the sentence of the 409 for an order whose time to pay is over.
const MsgRunOut = "The time to pay this order has run out."

// PayState is what the payment rules read of an order; Ref (the route's row) and
// Locked (the provider's) both give one.
type PayState struct {
	Status, Payment string
	Paid            bool
	PlacedAt        time.Time
}

// CheckPayable returns nil, or the *Refusal of the FIRST rule that fails, in this
// order: cancelled, paid, a way to pay that is not offered online, and the
// window of PayWindow from PlacedAt being over at now (the exact deadline counts
// as over).
func CheckPayable(s PayState, now time.Time) *Refusal {
	switch {
	case s.Status == "cancelled":
		return &Refusal{msgCancelled}
	case s.Paid:
		return &Refusal{msgPaid}
	case !slices.Contains(offered, s.Payment):
		return &Refusal{msgNotOnline}
	case !now.Before(s.PlacedAt.Add(PayWindow)):
		return &Refusal{MsgRunOut}
	}
	return nil
}

// Tx is the slice of a transaction these functions use; pgx.Tx satisfies it.
type Tx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Locked is an order's row as LockOrder read it, held FOR UPDATE until the
// transaction ends.
>>>>>>> refs/heads/main
type Locked struct {
	ID, Number, Status, Payment string
	Paid                        bool
	Total                       int64
	PlacedAt                    time.Time
}

<<<<<<< refs/heads/feature/dealer-handles-orders-status-courier-payment-be
const lockOrderSQL = `SELECT id, number, status, paid, total, placed_at, payment_method_id FROM orders WHERE id = $1 FOR UPDATE`

// LockOrder reads one order's row FOR UPDATE, inside the caller's transaction:
// the one function every writer of an order takes first (before any count of
// pairs), so a cancel, a save, a delete and a payment notice queue on the row.
// ErrNotFound when there is none.
func LockOrder(ctx context.Context, tx pgx.Tx, id string) (Locked, error) {
=======
// State is the row's part the payment rules read.
func (l Locked) State() PayState {
	return PayState{Status: l.Status, Payment: l.Payment, Paid: l.Paid, PlacedAt: l.PlacedAt}
}

const lockOrderSQL = `SELECT id, number, status, paid, total, placed_at, payment_method_id FROM orders WHERE id = $1 FOR UPDATE`

// LockOrder reads one order's row FOR UPDATE; ErrNotFound when there is none.
// Every transaction that touches an order and something else takes the order's
// row first.
func LockOrder(ctx context.Context, tx Tx, id string) (Locked, error) {
>>>>>>> refs/heads/main
	var l Locked
	err := tx.QueryRow(ctx, lockOrderSQL, id).Scan(&l.ID, &l.Number, &l.Status, &l.Paid, &l.Total, &l.PlacedAt, &l.Payment)
	if errors.Is(err, pgx.ErrNoRows) {
		return Locked{}, ErrNotFound
	}
<<<<<<< refs/heads/feature/dealer-handles-orders-status-courier-payment-be
	return l, err
}

// CancelRule narrows a cancel. The zero value is the dealer's: any order that is
// placed, confirmed or packed. The 30-minute expiry of unpaid online orders
// (the payments story) adds its own four terms by naming the ways to pay it
// gives up on and the moment before which an order must have been placed.
=======
	if err != nil {
		return Locked{}, fmt.Errorf("lock order: %w", err)
	}
	return l, nil
}

const markPaidSQL = `UPDATE orders SET paid = true, status = CASE WHEN status = 'placed' THEN 'confirmed' ELSE status END
WHERE id = $1 AND NOT paid`

// MarkPaid sets paid, and moves a "placed" order to "confirmed", once: changed is
// false when the order was already paid. An order that is already cancelled is
// marked paid and stays cancelled, so the dealer sees it and refunds it. It is the
// only statement in the project that sets orders.paid for an online order, and
// only a verified notice from the provider calls it.
func MarkPaid(ctx context.Context, tx Tx, id string) (changed bool, err error) {
	tag, err := tx.Exec(ctx, markPaidSQL, id)
	if err != nil {
		return false, fmt.Errorf("mark paid: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// CancelRule says which orders a Cancel may act on beyond "placed, confirmed or
// packed". The zero value is the dealer's cancel. The 30-minute rule sets both
// fields: only an unpaid order that is not a sample, whose way to pay is in Ways
// and that was placed at or before PlacedBefore.
>>>>>>> refs/heads/main
type CancelRule struct {
	PlacedBefore time.Time
	Ways         []string
}

<<<<<<< refs/heads/feature/dealer-handles-orders-status-courier-payment-be
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
=======
// The status test is in the UPDATE's own WHERE, so two cancels of one order at the
// same moment serialise on the row: the second finds it cancelled, changes zero
// rows and returns no pair.
const cancelSQL = `UPDATE orders SET status = 'cancelled'
WHERE id = $1 AND status IN ('placed', 'confirmed', 'packed')
RETURNING id`

const cancelExpirySQL = `UPDATE orders SET status = 'cancelled'
WHERE id = $1 AND status IN ('placed', 'confirmed', 'packed')
  AND NOT paid AND NOT is_sample AND payment_method_id = ANY($2) AND placed_at <= $3
RETURNING id`

const pairsOfSQL = `SELECT product_id, size, sum(qty)::integer FROM order_items WHERE order_id = $1
GROUP BY product_id, size ORDER BY product_id, size`

// Cancel moves one order to "cancelled" and puts its pairs back on the shelf,
// ONCE, in the caller's transaction. changed is false when the order was not in a
// state this cancel may act on (already cancelled, paid, shipped, a sample, too
// young, or another run won): nothing was returned then.
func Cancel(ctx context.Context, tx Tx, id string, rule CancelRule) (changed bool, err error) {
	var gone string
	if rule.Ways == nil {
		err = tx.QueryRow(ctx, cancelSQL, id).Scan(&gone)
	} else {
		err = tx.QueryRow(ctx, cancelExpirySQL, id, rule.Ways, rule.PlacedBefore).Scan(&gone)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cancel order: %w", err)
	}
	lines, err := pairsOf(ctx, tx, id)
	if err != nil {
		return false, err
	}
	if err := stock.Return(ctx, tx, lines); err != nil {
		return false, fmt.Errorf("return pairs: %w", err)
	}
	return true, nil
}

func pairsOf(ctx context.Context, tx Tx, id string) ([]stock.Line, error) {
	rows, err := tx.Query(ctx, pairsOfSQL, id)
	if err != nil {
		return nil, fmt.Errorf("read the pairs of an order: %w", err)
	}
	defer rows.Close()
	var lines []stock.Line
	for rows.Next() {
		var l stock.Line
		if err := rows.Scan(&l.Product, &l.Size, &l.Qty); err != nil {
			return nil, fmt.Errorf("read the pairs of an order: %w", err)
		}
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the pairs of an order: %w", err)
	}
	return lines, nil
}
>>>>>>> refs/heads/main
