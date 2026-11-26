package stock

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeTx struct {
	tags  []pgconn.CommandTag
	err   error
	args  [][]any
	calls int
}

func (f *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, f.err }

func (f *fakeTx) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	f.args = append(f.args, args)
	if f.err != nil {
		return pgconn.CommandTag{}, f.err
	}
	tag := f.tags[f.calls]
	f.calls++
	return tag, nil
}

func TestTakeRemovesEveryLineInLockOrder(t *testing.T) {
	tag := pgconn.NewCommandTag("UPDATE 1")
	tx := &fakeTx{tags: []pgconn.CommandTag{tag, tag, tag}}
	err := Take(context.Background(), tx, []Line{
		{Key{"samba", "40"}, 2}, {Key{"gazelle", "38"}, 1}, {Key{"samba", "38"}, 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]any{{"gazelle", "38", 1}, {"samba", "38", 3}, {"samba", "40", 2}}
	for i, a := range tx.args {
		if a[0] != want[i][0] || a[1] != want[i][1] || a[2] != want[i][2] {
			t.Errorf("statement %d: %v, want %v", i, a, want[i])
		}
	}
}

// A line that changes no row means a count moved under the lock: the order must
// be rolled back, never saved with pairs it did not take.
func TestTakeRefusesALineThatChangesNoRow(t *testing.T) {
	tx := &fakeTx{tags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 0")}}
	if err := Take(context.Background(), tx, []Line{{Key{"samba", "38"}, 1}}); err == nil {
		t.Fatal("a line that changed no row was accepted")
	}
}

func TestTakePassesADatabaseErrorOn(t *testing.T) {
	cause := errors.New("connection reset")
	tx := &fakeTx{err: cause}
	if err := Take(context.Background(), tx, []Line{{Key{"samba", "38"}, 1}}); !errors.Is(err, cause) {
		t.Fatalf("got %v", err)
	}
}

// Return puts every line back in lock order, adds the quantity, and treats a size
// that no longer exists (no row changed) as nothing to do rather than an error.
func TestReturnPutsPairsBackInLockOrderAndSkipsAMissingSize(t *testing.T) {
	tx := &fakeTx{tags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1"), pgconn.NewCommandTag("UPDATE 0"), pgconn.NewCommandTag("UPDATE 1")}}
	err := Return(context.Background(), tx, []Line{
		{Key{"samba", "40"}, 2}, {Key{"gazelle", "38"}, 1}, {Key{"samba", "38"}, 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]any{{"gazelle", "38", 1}, {"samba", "38", 3}, {"samba", "40", 2}}
	if len(tx.args) != len(want) {
		t.Fatalf("%d statements, want %d", len(tx.args), len(want))
	}
	for i, a := range tx.args {
		if a[0] != want[i][0] || a[1] != want[i][1] || a[2] != want[i][2] {
			t.Errorf("statement %d: %v, want %v", i, a, want[i])
		}
	}
}

func TestReturnPassesADatabaseErrorOn(t *testing.T) {
	cause := errors.New("connection reset")
	if err := Return(context.Background(), &fakeTx{err: cause}, []Line{{Key{"samba", "38"}, 1}}); !errors.Is(err, cause) {
		t.Fatalf("got %v", err)
	}
}

func TestLockWithNoKeysAsksNothing(t *testing.T) {
	held, err := Lock(context.Background(), &fakeTx{err: errors.New("must not be asked")}, nil)
	if err != nil || len(held) != 0 {
		t.Fatalf("got %v %v", held, err)
	}
}
