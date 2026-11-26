package payments

import (
	"context"
	"fmt"
	"log/slog"
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
// gives up. It is code, as the offered list is: a bank transfer is never in it
// (its 24-hour rule is a later story's).
var thirtyMinuteWays = []string{"vnpay"}

// dueSQL repeats the partial index's predicate literally so idx_orders_unpaid_open
// serves it. A sample order (the 14 that came with the design) is never due.
const dueSQL = `SELECT id FROM orders
WHERE NOT paid AND NOT is_sample AND status IN ('placed', 'confirmed', 'packed')
  AND payment_method_id = ANY($1) AND placed_at <= $2
ORDER BY placed_at LIMIT 100`

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
		for _, id := range ids {
			if budget.Err() != nil {
				return
			}
			changed, err := giveUp(budget, db, id, now)
			if err != nil {
				log.Error("the expiry could not give up an order", "order", id, "error", err)
				continue
			}
			if changed {
				given++
				log.Info("gave up an unpaid order", "order", id)
			}
		}
		// Repeat only while a query is full and the last one made progress.
		if len(ids) < expiryBatch || given == 0 {
			return
		}
	}
}

func due(ctx context.Context, db DB, now time.Time) ([]string, error) {
	rows, err := db.Query(ctx, dueSQL, thirtyMinuteWays, now.Add(-orders.PayWindow))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// giveUp is one transaction for one order. The list was read without a lock, so
// the conditions are judged again under the row lock by Cancel's own UPDATE: an
// order paid, cancelled by the dealer, shipped or moved by anyone since is left
// exactly as it is and its pairs are not touched.
func giveUp(ctx context.Context, db DB, id string, now func() time.Time) (bool, error) {
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
		changed, err = orders.Cancel(ctx, tx, id, orders.CancelRule{PlacedBefore: now().Add(-orders.PayWindow), Ways: thirtyMinuteWays})
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
