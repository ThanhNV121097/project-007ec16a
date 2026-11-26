package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These run the real Cancel, Delete and dealer reads against PostgreSQL (the
// real migrations, the real placing). They skip without TEST_DATABASE_URL.

func placeFor(t *testing.T, h http.Handler, key string, items ...string) string {
	t.Helper()
	a := placedOf(t, post(h, orderOf(lanPhone, items...), "application/json", key), http.StatusCreated)
	return a.Order["id"].(string)
}

func decodeInto(t *testing.T, raw []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
}

// cancel runs the dealer's cancel the way the handler does: lock the order, cancel it.
func cancel(t *testing.T, pool *pgxpool.Pool, id string) bool {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Error(err)
		return false
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := LockOrder(ctx, tx, id); err != nil {
		t.Error(err)
		return false
	}
	changed, err := Cancel(ctx, tx, id, CancelRule{})
	if err != nil {
		t.Error(err)
		return false
	}
	if err := tx.Commit(ctx); err != nil {
		t.Error(err)
	}
	return changed
}

// SC-7, SC-8, SC-9: a cancel puts the pairs of every line back, once, and the
// shoe's derived lists follow; the second cancel changes nothing.
func TestACancelReturnsEveryPairOnce(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "tc-a", "Shoe A", 1000000, map[string]int{"38": 5, "39": 1})
	addShoe(t, pool, "tc-b", "Shoe B", 2000000, map[string]int{"42": 5})
	id := placeFor(t, h, "cancel-key-0001", item("tc-a", "38", 1), item("tc-b", "42", 2), item("tc-a", "39", 1))
	if pairsLeft(t, pool, "tc-a", "38") != 4 || pairsLeft(t, pool, "tc-b", "42") != 3 || pairsLeft(t, pool, "tc-a", "39") != 0 {
		t.Fatal("the order did not take its pairs")
	}
	if out := scalar[[]string](t, pool, "SELECT sizes_out FROM product_stock WHERE product_id = 'tc-a'"); len(out) != 1 || out[0] != "39" {
		t.Fatalf("sold out before the cancel: %v", out)
	}
	if !cancel(t, pool, id) {
		t.Fatal("the first cancel changed nothing")
	}
	for _, c := range []struct {
		product, size string
		want          int
	}{{"tc-a", "38", 5}, {"tc-a", "39", 1}, {"tc-b", "42", 5}} {
		if got := pairsLeft(t, pool, c.product, c.size); got != c.want {
			t.Errorf("%s %s: %d pairs, want %d", c.product, c.size, got, c.want)
		}
	}
	if low := scalar[[]string](t, pool, "SELECT sizes_low FROM product_stock WHERE product_id = 'tc-a'"); len(low) != 1 || low[0] != "39" {
		t.Errorf("the size is not on its last pairs again: %v", low)
	}
	if cancel(t, pool, id) || pairsLeft(t, pool, "tc-a", "38") != 5 || pairsLeft(t, pool, "tc-b", "42") != 5 {
		t.Error("a second cancel returned the pairs again")
	}
}

// SC-9: two cancels at the same moment, twenty times over: exactly one wins and
// the shelf has the pairs back once.
func TestTwoCancelsAtOnceReturnThePairsOnce(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "tc-c", "Shoe C", 1000000, map[string]int{"40": 100})
	for round := 0; round < 20; round++ {
		id := placeFor(t, h, fmt.Sprintf("race-key-%04d", round), item("tc-c", "40", 2))
		before := pairsLeft(t, pool, "tc-c", "40")
		var wg sync.WaitGroup
		var mu sync.Mutex
		won := 0
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if cancel(t, pool, id) {
					mu.Lock()
					won++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if after := pairsLeft(t, pool, "tc-c", "40"); won != 1 || after != before+2 {
			t.Fatalf("round %d: %d cancels won, %d pairs after %d", round, won, after, before)
		}
	}
}

// ERD E1: the 14 sample orders took nothing from the shelf, so cancelling one
// moves the status and puts no pair on sale.
func TestCancellingASampleOrderReturnsNoPair(t *testing.T) {
	_, pool := shop(t, Options{})
	var id, product, size string
	err := pool.QueryRow(context.Background(), `SELECT o.id, i.product_id, i.size FROM orders o JOIN order_items i ON i.order_id = o.id AND i.line_no = 1
		WHERE o.is_sample AND o.status IN ('placed','confirmed','packed') LIMIT 1`).Scan(&id, &product, &size)
	if err != nil {
		t.Skipf("no open sample order: %v", err)
	}
	// -1 when the shoe has no count for that size: the shelf then has nothing to move.
	shelf := func() int {
		return scalar[int](t, pool, "SELECT coalesce((SELECT pairs FROM product_sizes WHERE product_id = $1 AND size = $2), -1)", product, size)
	}
	before := shelf()
	if !cancel(t, pool, id) {
		t.Fatal("the sample order was not cancelled")
	}
	if after := shelf(); after != before {
		t.Errorf("a sample cancel moved the shelf: %d -> %d", before, after)
	}
	if status := scalar[string](t, pool, "SELECT status FROM orders WHERE id = $1", id); status != "cancelled" {
		t.Errorf("status %s", status)
	}
}

// SC-20 to SC-22: only a cancelled, unpaid order is deleted; its pairs and
// tokens go with it; the shelf and the number counter do not move.
func TestDeleteRemovesOnlyACancelledUnpaidOrder(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "tc-d", "Shoe D", 1000000, map[string]int{"41": 10})
	id := placeFor(t, h, "delete-key-0001", item("tc-d", "41", 1))
	ctx := context.Background()
	remove := func(id string) error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err := Delete(ctx, tx, id); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if err := remove(id); !errors.Is(err, ErrNotCancelled) {
		t.Fatalf("a placed order: %v", err)
	}
	cancel(t, pool, id)
	exec(t, pool, "UPDATE orders SET paid = true WHERE id = $1", id)
	if err := remove(id); !errors.Is(err, ErrPaid) {
		t.Fatalf("a paid order: %v", err)
	}
	exec(t, pool, "UPDATE orders SET paid = false WHERE id = $1", id)
	counter := scalar[int64](t, pool, "SELECT value FROM counters WHERE name = 'order_number'")
	if err := remove(id); err != nil {
		t.Fatalf("a cancelled, unpaid order: %v", err)
	}
	for _, table := range []string{"orders WHERE id", "order_items WHERE order_id", "order_tokens WHERE order_id"} {
		if n := scalar[int](t, pool, "SELECT count(*) FROM "+table+" = $1", id); n != 0 {
			t.Errorf("%s: %d rows left", table, n)
		}
	}
	if pairsLeft(t, pool, "tc-d", "41") != 10 || scalar[int64](t, pool, "SELECT value FROM counters WHERE name = 'order_number'") != counter {
		t.Error("the delete moved the shelf or the counter")
	}
	if err := remove(id); !errors.Is(err, ErrNotFound) {
		t.Errorf("a second delete: %v", err)
	}
}

// SC-2, SC-3: a dealer's session reads the dealer view without a token; the
// order's token alone reads the public one.
func TestTheDealerViewCarriesWhatTheTokenViewNeverDoes(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "tc-e", "Shoe E", 1000000, map[string]int{"36": 9, "37": 9})
	a := placedOf(t, post(h, orderOf(lanPhone, item("tc-e", "36", 1), item("tc-e", "37", 2)), "application/json", "view-key-0001"), http.StatusCreated)
	id := a.Order["id"].(string)

	dealer := getAs(h, auth.Staff, "/v1/orders/"+id)
	if dealer.Code != http.StatusOK || dealer.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("%d %v %s", dealer.Code, dealer.Header(), dealer.Body)
	}
	var view map[string]any
	decodeInto(t, dealer.Body.Bytes(), &view)
	for key, want := range map[string]any{"phone": lanPhone, "address": "12 Example Street, Ward 1", "note": "leave it at the gate", "email": "", "transaction_no": ""} {
		if view[key] != want {
			t.Errorf("%s = %v, want %v", key, view[key], want)
		}
	}
	if items, _ := view["items"].([]any); len(items) != 2 || items[1].(map[string]any)["size"] != "37" || items[1].(map[string]any)["qty"] != float64(2) {
		t.Errorf("items %v", view["items"])
	}

	public := fetch(h, "/v1/orders/"+id, a.Token)
	var visible map[string]any
	decodeInto(t, public.Body.Bytes(), &visible)
	for _, key := range []string{"phone", "email", "address", "note", "transaction_no"} {
		if _, has := visible[key]; has {
			t.Errorf("the token view carries %s", key)
		}
	}
	if getAs(h, auth.Customer, "/v1/orders/"+id).Code != http.StatusNotFound || getAs(h, "", "/v1/orders/"+id).Code != http.StatusNotFound {
		t.Error("a visitor or a customer without the token was let in")
	}
	if getAs(h, auth.Admin, "/v1/orders/nope").Code != http.StatusNotFound {
		t.Error("a dealer was not told an order is missing")
	}
}
