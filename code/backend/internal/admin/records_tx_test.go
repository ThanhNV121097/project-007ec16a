package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// deadlockTx is a transaction that takes every statement and commits; whatever
// else a save asks of it is a test bug and panics on the nil Tx.
type deadlockTx struct{ pgx.Tx }

func (deadlockTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (deadlockTx) Commit(context.Context) error   { return nil }
func (deadlockTx) Rollback(context.Context) error { return nil }

type deadlockPool struct {
	Pool
	begun int
}

func (p *deadlockPool) Begin(context.Context) (pgx.Tx, error) {
	p.begun++
	return deadlockTx{}, nil
}

// PostgreSQL ends one side of a deadlock with 40P01; the save is run once more,
// and no more than once (services §5: "a deadlock is retried once").
func TestASaveIsRunAgainOnceWhenPostgresEndsItInADeadlock(t *testing.T) {
	deadlock := &pgconn.PgError{Code: "40P01"}
	other := errors.New("some other failure")
	for _, c := range []struct {
		name string
		runs []error // what each run of the save returns, in turn
		want error
	}{
		{"no deadlock", []error{nil}, nil},
		{"a deadlock, then success", []error{deadlock, nil}, nil},
		{"two deadlocks", []error{deadlock, deadlock}, deadlock},
		{"another failure is not retried", []error{other}, other},
	} {
		t.Run(c.name, func(t *testing.T) {
			pool := &deadlockPool{}
			h := &recHandler{db: pool, log: quiet()}
			ran := 0
			err := h.inTx(context.Background(), func(pgx.Tx) error {
				err := c.runs[ran]
				ran++
				return err
			})
			if ran != len(c.runs) || pool.begun != len(c.runs) {
				t.Errorf("the save ran %d times in %d transactions, want %d", ran, pool.begun, len(c.runs))
			}
			if !errors.Is(err, c.want) {
				t.Errorf("answered %v, want %v", err, c.want)
			}
		})
	}
}

// A count has a ceiling, so the sum of a shoe's sizes stays inside the integer
// product_stock adds them in.
func TestACountAboveAMillionPairsIsRefused(t *testing.T) {
	in := recSpecByName["products"].read(recBody(t, `{"pairs":{"38":1000000,"39":1000001,"40":2147483647}}`), false)
	if in.faults["pairs.38"] != "" {
		t.Errorf("a million pairs was refused: %q", in.faults["pairs.38"])
	}
	for _, key := range []string{"pairs.39", "pairs.40"} {
		if in.faults[key] != recManyPairs {
			t.Errorf("%s: %q, want %q", key, in.faults[key], recManyPairs)
		}
	}
	if bad := recSpecByName["products"].read(recBody(t, `{"pairs":{"38":-1}}`), false); bad.faults["pairs.38"] != recWholePairs {
		t.Errorf("a negative count: %q", bad.faults["pairs.38"])
	}
}
