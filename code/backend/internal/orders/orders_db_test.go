package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/data"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/db"
	"github.com/ThanhNV121097/project-007ec16a/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests run the real migrations, the real reader and the real placing
// against PostgreSQL. They read TEST_DATABASE_URL, as CI's backend job sets it,
// and skip without it (on a developer's machine and in run_tests it is not set).

// seeded returns a pool whose search_path is a schema of its own, with every
// migration applied, so a test never touches another test's tables.
func seeded(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("orders_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 16
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	if err := db.Migrate(ctx, pool, migrations.FS, quiet()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// monday10 is a Monday, 10:00 in Vietnam: before the cut-off, so a four-day
// order is expected that Friday.
func monday10() time.Time { return time.Date(2026, 11, 23, 10, 0, 0, 0, ict) }

func shop(t *testing.T, opts Options) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool := seeded(t)
	if opts.Now == nil {
		opts.Now = monday10
	}
	mux := http.NewServeMux()
	data.Register(mux, pool, quiet())
	Register(mux, pool, quiet(), opts)
	return mux, pool
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func scalar[T any](t *testing.T, pool *pgxpool.Pool, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

// addShoe makes a live shoe with the given counts (sizes are whole numbers here).
func addShoe(t *testing.T, pool *pgxpool.Pool, id, name string, price int64, pairs map[string]int) {
	t.Helper()
	exec(t, pool, "INSERT INTO products (id, name, price, status) VALUES ($1, $2, $3, 'live')", id, name, price)
	for size, n := range pairs {
		value, err := strconv.Atoi(size)
		if err != nil {
			t.Fatal(err)
		}
		exec(t, pool, "INSERT INTO product_sizes (product_id, size, size_value, pairs) VALUES ($1, $2, $3, $4)", id, size, value, n)
	}
}

func pairsLeft(t *testing.T, pool *pgxpool.Pool, id, size string) int {
	return scalar[int](t, pool, "SELECT pairs FROM product_sizes WHERE product_id = $1 AND size = $2", id, size)
}

func item(product, size string, qty int) string {
	return fmt.Sprintf(`{"product":%q,"size":%q,"qty":%d}`, product, size, qty)
}

func orderOf(phone string, items ...string) string {
	return fmt.Sprintf(`{"items":[%s],"delivery":"standard","payment":"vnpay","customer":"Lan Nguyen","phone":%q,`+
		`"address":"12 Example Street, Ward 1","city":"Da Nang","note":"leave it at the gate"}`, strings.Join(items, ","), phone)
}

const lanPhone = "0912 345 678"

type answer2 struct {
	Order      map[string]any `json:"order"`
	Token      string         `json:"token"`
	PaymentURL string         `json:"payment_url"`
}

func placedOf(t *testing.T, rec *httptest.ResponseRecorder, status int) answer2 {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("got %d %s, want %d", rec.Code, rec.Body, status)
	}
	var a answer2
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatal(err)
	}
	return a
}

func fetch(h http.Handler, target, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if token != "" {
		req.Header.Set("X-Order-Token", token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func record(t *testing.T, h http.Handler, target string) map[string]any {
	t.Helper()
	rec := fetch(h, target, "")
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil || rec.Code != 200 {
		t.Fatalf("%s: %d %s", target, rec.Code, rec.Body)
	}
	return m
}

// SC-13, SC-14, SC-17, SC-18, SC-20, SC-23, SC-36: an order is saved at the
// shop's prices, takes its pairs off the shelf and is read back with its token.
func TestPlacingSavesTheOrderAtTheShopsPrices(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 3})
	addShoe(t, pool, "t-b", "Test B", 1500000, map[string]int{"40": 5})

	// The browser's own numbers are ignored.
	body := strings.Replace(orderOf(lanPhone, item("t-a", "38", 2), item("t-b", "40", 1)), `"qty":2}`, `"qty":2,"unit_price":1}`, 1)
	body = strings.Replace(body, `"note":`, `"total":1,"delivery_fee":50000,"note":`, 1)
	rec := post(h, body, "application/json", "tc-key-0001")
	a := placedOf(t, rec, 201)

	if a.Order["id"] != "doi-24818" || a.Order["number"] != "DOI-24818" || a.Order["status"] != "placed" || a.Order["paid"] != false {
		t.Errorf("order: %v", a.Order)
	}
	if a.Order["total"] != float64(7680000) || a.Order["delivery_fee"] != float64(0) || a.Order["delivery"] != "standard" || a.Order["payment"] != "vnpay" {
		t.Errorf("money: %v", a.Order)
	}
	if a.Order["placed_at"] != "2026-11-23T10:00:00+07:00" || a.Order["eta"] != "2026-11-27" {
		t.Errorf("dates: %v %v", a.Order["placed_at"], a.Order["eta"])
	}
	if a.PaymentURL != "" || len(a.Token) != 32 || rec.Header().Get("Location") != "/v1/orders/doi-24818" {
		t.Errorf("answer: url %q token %q location %q", a.PaymentURL, a.Token, rec.Header().Get("Location"))
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("headers: %v", rec.Header())
	}
	for _, private := range []string{"phone", "email", "address", "note", "0912", "Example Street", "leave it at the gate"} {
		if strings.Contains(rec.Body.String(), private) {
			t.Errorf("the answer holds %q", private)
		}
	}

	// The shelf lost exactly the pairs, the counter moved, the token is stored hashed.
	if pairsLeft(t, pool, "t-a", "38") != 1 || pairsLeft(t, pool, "t-b", "40") != 4 {
		t.Error("the pairs did not leave the shelf")
	}
	if scalar[int64](t, pool, "SELECT value FROM counters WHERE name = 'order_number'") != 24818 {
		t.Error("the counter did not move")
	}
	if scalar[int](t, pool, "SELECT count(*) FROM order_tokens WHERE order_id = 'doi-24818' AND token_hash = $1", hashToken(a.Token)) != 1 ||
		scalar[int](t, pool, "SELECT count(*) FROM order_tokens WHERE token_hash = $1", []byte(a.Token)) != 0 {
		t.Error("the token is not stored as its hash")
	}

	// Read back with its own token; every other way in is the same 404.
	mine := fetch(h, "/v1/orders/doi-24818", a.Token)
	if mine.Code != 200 {
		t.Fatalf("own token: %d %s", mine.Code, mine.Body)
	}
	var view map[string]any
	_ = json.Unmarshal(mine.Body.Bytes(), &view)
	items := view["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["unit_price"] != float64(3090000) || items[0].(map[string]any)["name"] != "Test A" ||
		items[1].(map[string]any)["unit_price"] != float64(1500000) || view["product"] != "t-a" || view["qty"] != float64(2) {
		t.Errorf("pairs: %v", view)
	}
	for _, private := range []string{"phone", "email", "address", "note", "0912", "Example Street"} {
		if strings.Contains(mine.Body.String(), private) {
			t.Errorf("the read holds %q", private)
		}
	}
	other := placedOf(t, post(h, orderOf("0987 654 321", item("t-b", "40", 1)), "application/json", "tc-key-0002"), 201)
	if other.Order["id"] != "doi-24819" {
		t.Errorf("second number: %v", other.Order["id"])
	}
	none := fetch(h, "/v1/orders/doi-24818", "")
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"no token":      none,
		"another token": fetch(h, "/v1/orders/doi-24818", other.Token),
		"no such order": fetch(h, "/v1/orders/doi-99999", a.Token),
		"a seeded one":  fetch(h, "/v1/orders/doi-24817", a.Token),
	} {
		if rec.Code != 404 || rec.Body.String() != none.Body.String() {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}

	// A price change afterwards leaves the pair and the total as sold.
	exec(t, pool, "UPDATE products SET price = 1 WHERE id = 't-a'")
	var again map[string]any
	_ = json.Unmarshal(fetch(h, "/v1/orders/doi-24818", a.Token).Body.Bytes(), &again)
	if again["total"] != float64(7680000) || again["items"].([]any)[0].(map[string]any)["unit_price"] != float64(3090000) {
		t.Errorf("after the price change: %v", again)
	}

	// An order's token is no session: the team's list stays closed to it.
	if rec := fetch(h, "/v1/orders", a.Token); rec.Code != 401 {
		t.Errorf("GET /v1/orders: %d", rec.Code)
	}
}

// SC-23, SC-25: a refusal names every failing pair and changes nothing.
func TestARefusalNamesEveryPairAndChangesNothing(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 3})
	addShoe(t, pool, "t-c", "Test C", 1500000, map[string]int{"38": 0, "40": 2, "42": 2})
	addShoe(t, pool, "t-d", "Test D", 1500000, map[string]int{"38": 1})
	exec(t, pool, "UPDATE products SET status = 'draft' WHERE id = 't-d'")

	rec := post(h, orderOf(lanPhone, item("t-a", "38", 1), item("t-c", "38", 1), item("t-c", "40", 3), item("ghost", "38", 1), item("t-d", "38", 1), item("t-c", "42", 2)),
		"application/json", "tc-key-0003")
	e := envelopeOf(t, rec)
	if rec.Code != 409 || e.Error != "stock_conflict" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	want := map[string]string{
		"items.1": "Test C · EU 38 — sold out in your size.",
		"items.2": "Test C · EU 40 — only 2 left in your size.",
		"items.3": "ghost · EU 38 — no longer in the shop.",
		"items.4": "Test D · EU 38 — no longer in the shop.",
	}
	if !reflect.DeepEqual(e.Errors, want) {
		t.Errorf("errors: %v", e.Errors)
	}
	if pairsLeft(t, pool, "t-a", "38") != 3 || pairsLeft(t, pool, "t-c", "40") != 2 || pairsLeft(t, pool, "t-c", "42") != 2 {
		t.Error("a refusal changed the shelf")
	}
	if scalar[int64](t, pool, "SELECT value FROM counters WHERE name = 'order_number'") != 24817 || scalar[int](t, pool, "SELECT count(*) FROM orders") != 14 {
		t.Error("a refusal moved the counter or saved an order")
	}

	// The last pairs: asking for exactly what is left is accepted, and the size is then sold out.
	placedOf(t, post(h, orderOf(lanPhone, item("t-c", "40", 2)), "application/json", "tc-key-0004"), 201)
	if pairsLeft(t, pool, "t-c", "40") != 0 {
		t.Error("the last pairs did not sell")
	}

	// A size the shoe does not list is a 422 naming the line, not a 409.
	rec = post(h, orderOf(lanPhone, item("t-c", "99", 1)), "application/json", "tc-key-0005")
	if e := envelopeOf(t, rec); rec.Code != 422 || e.Errors["items.0"] != "Test C · EU 99 — not a size this shoe comes in." {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
	// An empty list of pairs, express delivery and cash on delivery are refused, naming the field.
	for field, body := range map[string]string{
		"items":    orderOf(lanPhone),
		"delivery": strings.Replace(orderOf(lanPhone, item("t-a", "38", 1)), `"standard"`, `"express"`, 1),
		"payment":  strings.Replace(orderOf(lanPhone, item("t-a", "38", 1)), `"vnpay"`, `"cod"`, 1),
	} {
		rec := post(h, body, "application/json", "tc-key-0006")
		if rec.Code != 422 || envelopeOf(t, rec).Errors[field] == "" {
			t.Errorf("%s: %d %s", field, rec.Code, rec.Body)
		}
	}
	if pairsLeft(t, pool, "t-a", "38") != 3 {
		t.Error("a 422 took a pair")
	}
}

// SC-24: 50 orders at the same moment for a size holding 10 pairs sell exactly 10.
func TestFiftyOrdersForTenPairsSellExactlyTen(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 10})
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses = map[int]int{}
		numbers  = map[any]bool{}
		named    int
	)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := post(h, orderOf(fmt.Sprintf("09123456%02d", i), item("t-a", "38", 1)), "application/json", fmt.Sprintf("race-key-%03d", i))
			mu.Lock()
			defer mu.Unlock()
			statuses[rec.Code]++
			if rec.Code == 201 {
				var a answer2
				_ = json.Unmarshal(rec.Body.Bytes(), &a)
				numbers[a.Order["number"]] = true
			}
			if rec.Code == 409 {
				var e envelope
				_ = json.Unmarshal(rec.Body.Bytes(), &e)
				if e.Errors["items.0"] == "Test A · EU 38 — sold out in your size." {
					named++
				}
			}
		}(i)
	}
	wg.Wait()
	if statuses[201] != 10 || statuses[409] != 40 || len(statuses) != 2 {
		t.Errorf("statuses: %v", statuses)
	}
	if len(numbers) != 10 || named != 40 {
		t.Errorf("%d different numbers, %d refusals naming the pair", len(numbers), named)
	}
	if pairsLeft(t, pool, "t-a", "38") != 0 || scalar[int](t, pool, "SELECT count(*) FROM orders") != 24 {
		t.Error("the shelf or the orders are wrong after the race")
	}
}

// SC-34: one key is one order; a lost answer lands on the order it made.
func TestTheSameKeyAnswersTheFirstOrderAgain(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 3})
	body := orderOf(lanPhone, item("t-a", "38", 1))
	first := post(h, body, "application/json", "tc-key-0001")
	a := placedOf(t, first, 201)
	second := post(h, body, "application/json", "tc-key-0001")
	b := placedOf(t, second, 200)
	if second.Header().Get("Idempotent-Replayed") != "true" || first.Header().Get("Idempotent-Replayed") != "" {
		t.Error("the replay header is wrong")
	}
	if a.Order["id"] != b.Order["id"] || a.Token == b.Token || len(b.Token) != 32 {
		t.Errorf("replay: %v %v tokens %q %q", a.Order["id"], b.Order["id"], a.Token, b.Token)
	}
	if fetch(h, "/v1/orders/doi-24818", b.Token).Code != 200 || fetch(h, "/v1/orders/doi-24818", a.Token).Code != 200 {
		t.Error("a token of the order does not open it")
	}
	if pairsLeft(t, pool, "t-a", "38") != 2 {
		t.Error("the shelf lost the pair twice")
	}
	// The same digits in another spelling are the same phone; other digits are another order.
	if again := placedOf(t, post(h, orderOf("0912.345.678", item("t-a", "38", 1)), "application/json", "tc-key-0001"), 200); again.Order["id"] != a.Order["id"] {
		t.Error("the same digits in another spelling did not replay the order")
	}
	rec := post(h, orderOf("0912 345 679", item("t-a", "38", 1)), "application/json", "tc-key-0001")
	if e := envelopeOf(t, rec); rec.Code != 409 || e.Error != "conflict" {
		t.Errorf("another phone: %d %s", rec.Code, rec.Body)
	}

	// The same key from two tabs at once makes one order.
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = post(h, orderOf("0987 654 321", item("t-a", "38", 1)), "application/json", "tab-key-0001").Code
		}(i)
	}
	wg.Wait()
	sort.Ints(codes)
	if codes[0] != 200 || codes[1] != 201 || scalar[int](t, pool, "SELECT count(*) FROM orders WHERE idempotency_key = 'tab-key-0001'") != 1 || pairsLeft(t, pool, "t-a", "38") != 1 {
		t.Errorf("two tabs: %v", codes)
	}

	// A refusal does not remember its key: the same key is judged afresh.
	addShoe(t, pool, "t-e", "Test E", 1500000, map[string]int{"38": 0})
	if rec := post(h, orderOf(lanPhone, item("t-e", "38", 1)), "application/json", "tc-key-0009"); rec.Code != 409 {
		t.Fatalf("got %d", rec.Code)
	}
	exec(t, pool, "UPDATE product_sizes SET pairs = 1 WHERE product_id = 't-e'")
	placedOf(t, post(h, orderOf(lanPhone, item("t-e", "38", 1)), "application/json", "tc-key-0009"), 201)
}

// SC-27, SC-28: a shoe's lists and total follow the counts the moment an order is saved.
func TestTheStockListsFollowTheCounts(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "t-f", "Test F", 1500000, map[string]int{"38": 3, "39": 2, "40": 0, "41": 5})
	lists := func() (sizes, low, out []any, stock float64) {
		m := record(t, h, "/v1/products/t-f")
		return m["sizes"].([]any), m["sizes_low"].([]any), m["sizes_out"].([]any), m["stock"].(float64)
	}
	sizes, low, out, stock := lists()
	if !reflect.DeepEqual(sizes, []any{"38", "39", "40", "41"}) || !reflect.DeepEqual(low, []any{"39"}) || !reflect.DeepEqual(out, []any{"40"}) || stock != 10 {
		t.Fatalf("before: %v %v %v %v", sizes, low, out, stock)
	}
	for i, step := range []struct {
		size         string
		qty          int
		low, out     []any
		stock        float64
	}{
		{"38", 1, []any{"38", "39"}, []any{"40"}, 9},
		{"38", 1, []any{"38", "39"}, []any{"40"}, 8},
		{"38", 1, []any{"39"}, []any{"38", "40"}, 7},
		{"41", 2, []any{"39"}, []any{"38", "40"}, 5},
	} {
		placedOf(t, post(h, orderOf(lanPhone, item("t-f", step.size, step.qty)), "application/json", fmt.Sprintf("step-key-%03d", i)), 201)
		_, low, out, stock = lists()
		if !reflect.DeepEqual(low, step.low) || !reflect.DeepEqual(out, step.out) || stock != step.stock {
			t.Errorf("step %d: low %v out %v stock %v, want %v %v %v", i, low, out, stock, step.low, step.out, step.stock)
		}
	}
	// The list and the record agree.
	_, list, _ := func() (int, map[string]any, string) {
		rec := fetch(h, "/v1/products?limit=100", "")
		var m map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &m)
		return rec.Code, m, rec.Body.String()
	}()
	found := false
	for _, it := range list["items"].([]any) {
		if it.(map[string]any)["id"] == "t-f" {
			found = reflect.DeepEqual(it, record(t, h, "/v1/products/t-f"))
		}
	}
	if !found {
		t.Error("the shoe in the list differs from its record")
	}
}

// SC-19, SC-29: the seeded orders each hold one pair, and every shoe keeps its lists.
func TestTheSeededOrdersAndShoesSurviveTheMigration(t *testing.T) {
	_, pool := shop(t, Options{})
	if n := scalar[int](t, pool, "SELECT count(*) FROM orders WHERE is_sample"); n != 14 {
		t.Fatalf("%d seeded orders", n)
	}
	if n := scalar[int](t, pool, `SELECT count(*) FROM (SELECT o.id FROM orders o LEFT JOIN order_items i ON i.order_id = o.id
		WHERE o.is_sample GROUP BY o.id HAVING count(i.id) <> 1) x`); n != 0 {
		t.Errorf("%d seeded orders do not hold exactly one pair", n)
	}
	var product, size string
	if err := pool.QueryRow(context.Background(), "SELECT product_id, size FROM order_items WHERE order_id = 'doi-24817' AND line_no = 1").Scan(&product, &size); err != nil ||
		product != "samba-og-w-crystal-white" || size != "39 1/3" {
		t.Errorf("doi-24817: %q %q %v", product, size, err)
	}
	if n := scalar[int](t, pool, "SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND column_name IN ('sizes', 'sizes_low', 'sizes_out', 'stock')"); n != 0 {
		t.Errorf("%d stock columns remain on products", n)
	}
	if n := scalar[int](t, pool, "SELECT count(*) FROM product_sizes WHERE size_value <= 0 OR pairs < 0"); n != 0 {
		t.Error("a count is wrong")
	}
}

// SC-30: the shop offers VNPay alone, and the other ways to pay keep their rows.
func TestPaymentOptionsListVNPayAlone(t *testing.T) {
	h, _ := shop(t, Options{})
	rec := fetch(h, "/v1/payment-options?anything=1", "")
	var got struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != 200 || got.Total != 1 || len(got.Items) != 1 {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	v := got.Items[0]
	if v["id"] != "vnpay" || v["online"] != true || v["is_available"] != true || v["icon"] == "" || v["name"] == "" || v["note"] == "" {
		t.Errorf("vnpay: %v", v)
	}
	if n := len(record(t, h, "/v1/payments?limit=100")["items"].([]any)); n != 4 {
		t.Errorf("/v1/payments lists %d ways to pay, want 4", n)
	}
}

// SC-31: placing goes through the provider interface, proven with test providers.
func TestThePaymentAddressComesFromTheProvider(t *testing.T) {
	for name, c := range map[string]struct {
		providers map[string]Provider
		want      string
	}{
		"no provider":          {nil, ""},
		"a provider with a url": {map[string]Provider{"vnpay": stubProvider{open: true, url: "https://pay.example.test/x"}}, "https://pay.example.test/x"},
		"a provider that fails": {map[string]Provider{"vnpay": stubProvider{open: true, err: errors.New("keys missing")}}, ""},
	} {
		h, pool := shop(t, Options{Providers: c.providers})
		addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 3})
		a := placedOf(t, post(h, orderOf(lanPhone, item("t-a", "38", 1)), "application/json", "tc-key-0001"), 201)
		if a.PaymentURL != c.want || a.Order["paid"] != false {
			t.Errorf("%s: payment_url %q paid %v", name, a.PaymentURL, a.Order["paid"])
		}
		if scalar[int](t, pool, "SELECT count(*) FROM orders WHERE id = 'doi-24818' AND NOT paid") != 1 {
			t.Errorf("%s: the order was not saved unpaid", name)
		}
	}
}

// SC-22: over 100 sequential orders, placing answers within 1.5 s at p95.
func TestPlacingIsFast(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 1000})
	var took []time.Duration
	for i := 0; i < 100; i++ {
		start := time.Now()
		placedOf(t, post(h, orderOf(lanPhone, item("t-a", "38", 1)), "application/json", fmt.Sprintf("speed-key-%03d", i)), 201)
		took = append(took, time.Since(start))
	}
	sort.Slice(took, func(i, j int) bool { return took[i] < took[j] })
	if p95 := took[94]; p95 > 1500*time.Millisecond {
		t.Errorf("p95 is %v", p95)
	}
}

// The unreachable database: placing answers 500 and nothing is saved.
func TestAClosedPoolIsA500AndSavesNothing(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 3})
	pool.Close()
	rec := post(h, orderOf(lanPhone, item("t-a", "38", 1)), "application/json", "tc-key-0001")
	if rec.Code != 500 || envelopeOf(t, rec).Error != "internal" {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}
