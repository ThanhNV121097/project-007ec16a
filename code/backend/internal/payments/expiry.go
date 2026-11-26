package payments

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
	"github.com/jackc/pgx/v5"
)

const (
	// expiryEvery is the cadence: an order is given up within 31 minutes of being
	// placed (30:00 plus at most one tick).
	expiryEvery = 30 * time.Second
	// expiryBatch is the most orders one query returns.
	expiryBatch = 100
	// expiryBudget stops a run whatever is left; the rest is the next tick's.
	expiryBudget = 25 * time.Second
)

// thirtyMinuteWays are the ways to pay whose unpaid orders the 30-minute rule
// gives up, and transferWays those the 24-hour rule gives up. Both are code, as the
// offered list is, and a rule applies by the order's way, not by today's settings:
// an unpaid MoMo or bank order is still given up after the owner removes its
// settings, so its pairs never stay held for ever.
var (
	thirtyMinuteWays = []string{"vnpay", "momo"}
	transferWays     = []string{"bank"}
)

// dueSQL repeats the partial index's predicate literally so idx_orders_unpaid_open
// serves it, and returns each order's way so the rule for it can be chosen. The
// shortest window bounds every rule ($1), so the index range is tight; a bank
// order must also be past its own, longer one ($4). A sample order (the 14 that
// came with the design) is never due.
const dueSQL = `SELECT id, payment_method_id FROM orders
WHERE NOT paid AND NOT is_sample AND status IN ('placed', 'confirmed', 'packed')
  AND placed_at <= $1
  AND (payment_method_id = ANY($2) OR (payment_method_id = ANY($3) AND placed_at <= $4))
ORDER BY placed_at LIMIT 100`

// dueOrder is an order a rule may give up, with the way that chooses the rule.
type dueOrder struct{ ID, Way string }

func (o dueOrder) transfer() bool { return slices.Contains(transferWays, o.Way) }

// rule is the name the log line carries.
func (o dueOrder) rule() string {
	if o.transfer() {
		return "24h"
	}
	return "30m"
}

// cancelRule is what the cancel judges again under the order's row lock.
func (o dueOrder) cancelRule(now time.Time) orders.CancelRule {
	if o.transfer() {
		return orders.CancelRule{PlacedBefore: now.Add(-orders.TransferWindow), Ways: transferWays}
	}
	return orders.CancelRule{PlacedBefore: now.Add(-orders.PayWindow), Ways: thirtyMinuteWays}
}

// RunExpiry gives up an unpaid online order after 30 minutes: once at start-up
// (an order that fell due while the server was down is cancelled at boot), then
// every 30 seconds until ctx ends. The clock is the application's, passed in; nil
// is time.Now. Nothing is logged when nothing is due.
func RunExpiry(ctx context.Context, db DB, log *slog.Logger, now func() time.Time) {
	if now == nil {
		now = time.Now
	}
	runExpiry(ctx, db, log, now)
	tick := time.NewTicker(expiryEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			runExpiry(ctx, db, log, now)
		}
	}
}

// runExpiry is one run. A panic is recovered and logged so the loop keeps ticking.
// One order failing (a lock timeout, a database error) is logged with its id and
// the run goes on to the next.
func runExpiry(ctx context.Context, db DB, log *slog.Logger, now func() time.Time) {
	defer func() {
		if p := recover(); p != nil {
			log.Error("the expiry run panicked", "panic", fmt.Sprint(p))
		}
	}()
	budget, cancel := context.WithTimeout(ctx, expiryBudget)
	defer cancel()
	for budget.Err() == nil {
		ids, err := due(budget, db, now())
		if err != nil {
			log.Error("the expiry could not list the orders that are due", "error", err)
			return
		}
		given := 0
		for _, o := range ids {
			if budget.Err() != nil {
				return
			}
			changed, err := giveUp(budget, db, o, now)
			if err != nil {
				log.Error("the expiry could not give up an order", "order", o.ID, "rule", o.rule(), "error", err)
				continue
			}
			if changed {
				given++
				log.Info("gave up an unpaid order", "order", o.ID, "rule", o.rule())
			}
		}
		// Repeat only while a query is full and the last one made progress.
		if len(ids) < expiryBatch || given == 0 {
			return
		}
	}
}

func due(ctx context.Context, db DB, now time.Time) ([]dueOrder, error) {
	rows, err := db.Query(ctx, dueSQL, now.Add(-orders.PayWindow), thirtyMinuteWays, transferWays, now.Add(-orders.TransferWindow))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []dueOrder
	for rows.Next() {
		var o dueOrder
		if err := rows.Scan(&o.ID, &o.Way); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// giveUp is one transaction for one order. The list was read without a lock, so
// the conditions are judged again under the row lock by Cancel's own UPDATE: an
// order paid, cancelled by the dealer, shipped or moved by anyone since is left
// exactly as it is and its pairs are not touched.
func giveUp(ctx context.Context, db DB, o dueOrder, now func() time.Time) (bool, error) {
	var changed bool
	err := retryDeadlock(func() error {
		tx, err := db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return fmt.Errorf("begin: %w", err)
		}
		defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
		if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '3s'"); err != nil {
			return fmt.Errorf("set lock timeout: %w", err)
		}
		changed, err = orders.Cancel(ctx, tx, o.ID, o.cancelRule(now()))
		if err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
		return nil
	})
	return changed, err
}
