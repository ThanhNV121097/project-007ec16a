package payments

import (
	"context"
	"errors"
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
	for _, part := range []string{"NOT paid", "NOT is_sample", "status IN ('placed', 'confirmed', 'packed')", "payment_method_id = ANY($1)", "placed_at <= $2", "ORDER BY placed_at LIMIT 100"} {
		if !contains(dueSQL, part) {
			t.Errorf("the due query lacks %q", part)
		}
	}
	if len(thirtyMinuteWays) != 1 || thirtyMinuteWays[0] != "vnpay" {
		t.Errorf("the ways the rule covers: %v", thirtyMinuteWays)
	}
}

func contains(s, part string) bool {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
