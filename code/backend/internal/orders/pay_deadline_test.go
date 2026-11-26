package orders

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// deadlineDB is an orderDB that remembers the deadline each statement was given.
type deadlineDB struct {
	orderDB
	deadlines []time.Time
	unbounded int
}

func (d *deadlineDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if at, ok := ctx.Deadline(); ok {
		d.deadlines = append(d.deadlines, at)
	} else {
		d.unbounded++
	}
	return d.orderDB.QueryRow(ctx, sql, args...)
}

// The access statement of "Pay now" runs under the same txTimeout as GET's read:
// the request's own context carries no deadline, so without it only the server's
// write timeout would end a statement that hangs.
func TestPayBoundsTheAccessStatementByTheTransactionTimeout(t *testing.T) {
	db := &deadlineDB{orderDB: orderDB{order: sample(nil), token: ownToken}}
	h := mountPay(db, Options{})
	started := time.Now()
	rec := callPay(h, "doi-24818", ownToken, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d %s, want the 409 of an order with no provider", rec.Code, rec.Body)
	}
	if db.unbounded != 0 || len(db.deadlines) != 1 {
		t.Fatalf("the access statement ran %d times without a deadline and %d with one, want one with", db.unbounded, len(db.deadlines))
	}
	if left := db.deadlines[0].Sub(started); left < txTimeout || left > txTimeout+time.Second {
		t.Errorf("the access statement's deadline is %s from the request, want %s", left, txTimeout)
	}
}

// An access statement that runs out of time is the service's own failure: a
// generic 500 that names no SQL, order or token.
func TestPayAnAccessStatementThatTimesOutIsAGeneric500(t *testing.T) {
	db := &orderDB{order: sample(nil), token: ownToken, fails: context.DeadlineExceeded}
	rec := callPay(mountPay(db, Options{}), "doi-24818", ownToken, "")
	e := envelopeOf(t, rec)
	if rec.Code != http.StatusInternalServerError || e.Error != "internal" || e.Message != "Something went wrong. Please try again." {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	for _, leak := range []string{"SELECT", "orders", ownToken, "deadline"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Errorf("the 500 body names %q: %s", leak, rec.Body)
		}
	}
}
