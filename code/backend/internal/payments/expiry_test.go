package payments

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// downDB is a database that is not there: every statement fails, and Query can panic.
type downDB struct {
	panics bool
	asked  int
}

func (d *downDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) { return nil, errors.New("down") }
func (d *downDB) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (d *downDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("down")
}
func (d *downDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	d.asked++
	if d.panics {
		panic("boom")
	}
	return nil, errors.New("down")
}

// The loop runs once at start-up and stops when its context ends, with the database
// failing or panicking: a failed run never ends the loop.
func TestTheExpiryLoopRunsAtStartUpAndStopsWithItsContext(t *testing.T) {
	for name, db := range map[string]*downDB{"a failing database": {}, "a panic": {panics: true}} {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			RunExpiry(ctx, db, quiet(), nil)
			close(done)
		}()
		deadline := time.Now().Add(2 * time.Second)
		for db.asked == 0 && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: the loop did not stop", name)
		}
		if db.asked == 0 {
			t.Errorf("%s: no run at start-up", name)
		}
	}
}

// The query repeats the partial index's predicate literally and names the rule's four
// extra conditions, so the index serves it and a sample is never due.
func TestTheDueQueryRepeatsTheIndexPredicate(t *testing.T) {
	for _, part := range []string{"NOT paid", "NOT is_sample", "status IN ('placed', 'confirmed', 'packed')", "placed_at <= $1", "payment_method_id = ANY($2)", "payment_method_id = ANY($3) AND placed_at <= $4", "ORDER BY placed_at LIMIT 100"} {
		if !strings.Contains(dueSQL, part) {
			t.Errorf("the due query lacks %q", part)
		}
	}
	if len(thirtyMinuteWays) != 2 || thirtyMinuteWays[0] != "vnpay" || thirtyMinuteWays[1] != "momo" {
		t.Errorf("the ways the 30-minute rule covers: %v", thirtyMinuteWays)
	}
	if len(transferWays) != 1 || transferWays[0] != "bank" {
		t.Errorf("the ways the 24-hour rule covers: %v", transferWays)
	}
}

// PAY-003 and PAY-020 item 5: the rule for an order follows its way, with its own window.
func TestTheRuleOfAnOrderFollowsItsWay(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	for way, want := range map[string]struct {
		rule   string
		window time.Duration
	}{
		"vnpay": {"30m", 30 * time.Minute}, "momo": {"30m", 30 * time.Minute}, "bank": {"24h", 24 * time.Hour},
	} {
		o := dueOrder{ID: "doi-1", Way: way}
		r := o.cancelRule(now)
		if o.rule() != want.rule || !r.PlacedBefore.Equal(now.Add(-want.window)) || len(r.Ways) == 0 {
			t.Errorf("%s: rule %s, placed before %v, ways %v", way, o.rule(), r.PlacedBefore, r.Ways)
		}
		for _, w := range r.Ways {
			if (w == "bank") != (way == "bank") {
				t.Errorf("%s: the rule covers %v", way, r.Ways)
			}
		}
	}
}
