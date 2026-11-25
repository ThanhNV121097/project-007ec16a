package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/db"
	"github.com/ThanhNV121097/project-007ec16a/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests run the real migrations and the real reader against PostgreSQL.
// They read TEST_DATABASE_URL, as CI's backend job sets it, and skip without it
// (on a developer's machine and in run_tests it is not set).

// schemaPool returns a pool whose search_path is a schema of its own, so a test
// never touches the tables a real migration made in the same database.
func schemaPool(t *testing.T) *pgxpool.Pool {
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
	schema := fmt.Sprintf("data_test_%d", time.Now().UnixNano())
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
	return pool
}

// seeded returns a pool on an empty schema with every migration applied.
func seeded(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := schemaPool(t)
	if err := db.Migrate(context.Background(), pool, migrations.FS, quiet()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func tableCounts(t *testing.T, pool *pgxpool.Pool) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{"categories", "lines", "products", "reviews", "shop_terms", "delivery_options",
		"payment_methods", "faqs", "size_chart", "orders", "order_items", "rate_limits"} {
		out[table] = count(t, pool, "SELECT count(*) FROM "+table)
	}
	return out
}

// get reads through the real handler and the real pool.
func get(t *testing.T, pool *pgxpool.Pool, target string) (int, map[string]any, string) {
	t.Helper()
	res := request(serve(pool), target)
	var body map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	return res.Code, body, res.Body.String()
}

func ids(items any) []string {
	var out []string
	for _, it := range items.([]any) {
		out = append(out, it.(map[string]any)["id"].(string))
	}
	return out
}

func TestSeedHoldsEveryRecordOfEveryFile(t *testing.T) {
	pool := seeded(t)
	want := map[string]int{"categories": 3, "lines": 5, "products": 12, "reviews": 36, "shop_terms": 1,
		"delivery_options": 1, "payment_methods": 4, "faqs": 12, "size_chart": 17, "orders": 14, "order_items": 14, "rate_limits": 0}
	if got := tableCounts(t, pool); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// The free-delivery shape: one standard option at fee 0, no express, and
	// none of the dropped shop columns.
	if n := count(t, pool, "SELECT count(*) FROM delivery_options WHERE id = 'standard' AND fee = 0"); n != 1 {
		t.Errorf("standard at fee 0: %d", n)
	}
	if n := count(t, pool, `SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema()
		AND table_name = 'shop_terms' AND column_name IN ('free_delivery_from', 'express_days', 'express_cities', 'exchange_days', 'refund_days')`); n != 0 {
		t.Errorf("shop_terms still has %d dropped columns", n)
	}
	// Every sample order is one free standard line priced as its pairs.
	if n := count(t, pool, `SELECT count(*) FROM orders o JOIN order_items oi ON oi.order_id = o.id
		WHERE o.delivery_option_id = 'standard' AND o.delivery_fee = 0 AND o.total = oi.unit_price * oi.qty`); n != 14 {
		t.Errorf("%d of 14 sample orders are free standard orders priced as their pairs", n)
	}
	// The database itself refuses a shoe whose line does not exist.
	_, err := pool.Exec(context.Background(), "UPDATE products SET line_id = 'nope' WHERE id = 'samba-og-w-core-black'")
	if err == nil {
		t.Error("a shoe was given the line nope")
	}
}

func TestTheHelpersExist(t *testing.T) {
	pool := seeded(t)
	ctx := context.Background()
	for in, want := range map[string]string{"Cửa Lò": "cua lo", "Đại": "dai", "100%": "100%"} {
		var got string
		if err := pool.QueryRow(ctx, "SELECT doi_fold($1)", in).Scan(&got); err != nil || got != want {
			t.Errorf("doi_fold(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	// updated_at moves by trigger, whatever a handler forgets.
	var before, after time.Time
	if err := pool.QueryRow(ctx, "SELECT updated_at FROM products WHERE id = 'samba-og-w-core-black'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := pool.Exec(ctx, "UPDATE products SET price = price WHERE id = 'samba-og-w-core-black'"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT updated_at FROM products WHERE id = 'samba-og-w-core-black'").Scan(&after); err != nil || !after.After(before) {
		t.Errorf("updated_at did not move: %v -> %v (%v)", before, after, err)
	}
}

// SC-5: a restart runs the migrations again; nothing is inserted twice and
// nothing changed after seeding is overwritten.
func TestRestartKeepsChangesAndInsertsNothing(t *testing.T) {
	pool := seeded(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE products SET price = 3300000 WHERE id = 'samba-og-w-core-black'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE reviews SET status = 'hidden' WHERE id = 'r-01'"); err != nil {
		t.Fatal(err)
	}
	before := tableCounts(t, pool)
	if err := db.Migrate(ctx, pool, migrations.FS, quiet()); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if after := tableCounts(t, pool); !reflect.DeepEqual(before, after) {
		t.Errorf("counts moved: %v -> %v", before, after)
	}
	if n := count(t, pool, "SELECT price FROM products WHERE id = 'samba-og-w-core-black'"); n != 3300000 {
		t.Errorf("the price was overwritten: %d", n)
	}
	if n := count(t, pool, "SELECT count(*) FROM reviews WHERE id = 'r-01' AND status = 'hidden'"); n != 1 {
		t.Error("the review's status was overwritten")
	}
}

// SC-5: a seed that meets an id already present fails loudly, names the id and
// leaves that row untouched.
func TestSeedMeetingAnExistingIDFailsLoudly(t *testing.T) {
	pool := schemaPool(t)
	ctx := context.Background()
	// The schema the seed lands in is every migration stamped before it. A later
	// migration (the per-size stock, the order checkout) assumes the seeded rows
	// are there and rewrites the columns the seed fills, so it cannot run first.
	const seedVersion = "20261125100200"
	before := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") && e.Name() < seedVersion {
			data, err := fs.ReadFile(migrations.FS, e.Name())
			if err != nil {
				t.Fatal(err)
			}
			before[e.Name()] = &fstest.MapFile{Data: data}
		}
	}
	if err := db.Migrate(ctx, pool, before, quiet()); err != nil {
		t.Fatalf("schema only: %v", err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO lines (id, name) VALUES ('samba', 'Not the Samba')"); err != nil {
		t.Fatal(err)
	}
	err = db.Migrate(ctx, pool, migrations.FS, quiet())
	if err == nil || !strings.Contains(err.Error(), "samba") {
		t.Fatalf("want an error naming samba, got %v", err)
	}
	if n := count(t, pool, "SELECT count(*) FROM lines WHERE id = 'samba' AND name = 'Not the Samba'"); n != 1 {
		t.Error("the existing row was changed")
	}
	if n := count(t, pool, "SELECT count(*) FROM products"); n != 0 {
		t.Errorf("the failed seed left %d products behind", n)
	}
}

func TestListsPageFilterAndKeepTheSeedOrder(t *testing.T) {
	pool := seeded(t)
	code, body, raw := get(t, pool, "/v1/products")
	if code != 200 || body["total"] != float64(12) || body["page"] != float64(1) || body["limit"] != float64(12) {
		t.Fatalf("%d %s", code, raw)
	}
	all := ids(body["items"])
	if len(all) != 12 || all[0] != "samba-og-w-core-black" || all[11] != "samba-og-kids-cloud-white" {
		t.Errorf("order: %v", all)
	}

	_, body, _ = get(t, pool, "/v1/products?page=2&limit=5")
	want := []string{"adizero-adios-wonder-blue", "adizero-evo-sl-charcoal", "megaride-s2-cloud-white", "megaride-bones-halo-green", "megaride-ag-xl-core-black"}
	if got := ids(body["items"]); !reflect.DeepEqual(got, want) || body["total"] != float64(12) {
		t.Errorf("page 2: %v", got)
	}
	_, body, _ = get(t, pool, "/v1/products?page=9&limit=5")
	if len(body["items"].([]any)) != 0 || body["total"] != float64(12) || body["page"] != float64(9) {
		t.Errorf("page 9: %v", body)
	}
	_, body, _ = get(t, pool, "/v1/products?limit=500")
	if body["limit"] != float64(100) {
		t.Errorf("limit: %v", body["limit"])
	}

	for target, want := range map[string]float64{
		"/v1/products?category=women": 4, "/v1/products?category=women&status=live": 4,
		"/v1/products?category=men&is_new=true": 1, "/v1/products?featured=true": 1,
		"/v1/faqs?topic=returns": 2, "/v1/reviews?product=samba-og-w-core-black": 3,
		"/v1/products?sizes_low=40": 2, "/v1/products?sizes_low=4": 0,
		"/v1/products?price=3290000": 2, "/v1/products?rating=4.8": 4,
		"/v1/sizes?cm=21.6": 1, "/v1/delivery?fee=0": 1,
		"/v1/products?zzz=1&q=samba&sort=-price&category=women,kids": 12,
	} {
		code, body, raw := get(t, pool, target)
		if code != 200 || body["total"] != want {
			t.Errorf("%s: %d total %v, want %v: %.200s", target, code, body["total"], want, raw)
		}
	}
	_, body, _ = get(t, pool, "/v1/products?category=men&is_new=true")
	if got := ids(body["items"]); !reflect.DeepEqual(got, []string{"gazelle-indoor-solar-gold"}) {
		t.Errorf("men and new: %v", got)
	}
	_, body, _ = get(t, pool, "/v1/products?sizes_low=40")
	if got := ids(body["items"]); !reflect.DeepEqual(got, []string{"samba-og-w-core-black", "adizero-evo-sl-charcoal"}) {
		t.Errorf("sizes_low=40: %v", got)
	}
}

func TestRecordsKeepTheirTypesListsAndNulls(t *testing.T) {
	pool := seeded(t)
	code, shoe, raw := get(t, pool, "/v1/products/samba-og-w-core-black")
	if code != 200 {
		t.Fatalf("%d %s", code, raw)
	}
	for key, want := range map[string]any{
		"price": float64(3290000), "rating": 4.8, "review_count": float64(23), "stock": float64(20), "featured": true, "is_new": true,
		"released": "2026-11-03", "line": "samba", "category": "women", "swatch": "black",
		"sizes":     []any{"36", "36 2/3", "37 1/3", "38", "38 2/3", "39 1/3", "40", "40 2/3", "41 1/3"},
		"sizes_low": []any{"37 1/3", "40"}, "sizes_out": []any{"41 1/3"},
	} {
		if !reflect.DeepEqual(shoe[key], want) {
			t.Errorf("%s: %#v, want %#v", key, shoe[key], want)
		}
	}
	_, white, _ := get(t, pool, "/v1/products/samba-og-w-crystal-white")
	if !reflect.DeepEqual(white["sizes_out"], []any{}) {
		t.Errorf("an empty list must be [], got %#v", white["sizes_out"])
	}
	_, size, _ := get(t, pool, "/v1/sizes/eu-35-1-2")
	if size["cm"] != 21.6 {
		t.Errorf("cm: %#v", size["cm"])
	}
	_, shop, _ := get(t, pool, "/v1/shop/shop")
	if shop["standard_days"] != "2–4" || shop["dispatch_cutoff"] != "15:00" || shop["rating"] != 4.8 {
		t.Errorf("shop: %v", shop)
	}
	for _, gone := range []string{"free_delivery_from", "express_days", "express_cities", "exchange_days", "refund_days"} {
		if _, has := shop[gone]; has {
			t.Errorf("shop still has %s", gone)
		}
	}

	// A record equals its copy in the list, field for field.
	_, list, _ := get(t, pool, "/v1/products?limit=100")
	for _, it := range list["items"].([]any) {
		one := it.(map[string]any)
		_, again, _ := get(t, pool, "/v1/products/"+one["id"].(string))
		if !reflect.DeepEqual(one, again) {
			t.Errorf("%v: the list and the record differ", one["id"])
		}
	}

	for target, want := range map[string]int{"/v1/products/nope": 404, "/v1/delivery/express": 404, "/v1/delivery/standard": 200} {
		if code, _, _ := get(t, pool, target); code != want {
			t.Errorf("%s: %d, want %d", target, code, want)
		}
	}
}

// SC-20 to SC-23: what a stranger reads. Every collection is listed page by
// page and every record read by id; no body holds a customer's details or the
// owner's notes.
func TestNoBodyHoldsAPrivateOrBookkeepingValue(t *testing.T) {
	pool := seeded(t)
	secrets := []string{"0938 410 552", "42 Example Street, Hải Châu", "Gift — no price inside the box.", "0977 302 118",
		"7 Example Lane", "Customer cancelled by phone", "featuresneakerboutique", "US$130", "size.ly", "5.769.000"}
	for _, c := range registry {
		_, list, raw := get(t, pool, "/v1/"+c.Name+"?limit=100")
		bodies := []string{raw}
		for _, id := range ids(list["items"]) {
			_, rec, one := get(t, pool, "/v1/"+c.Name+"/"+id)
			bodies = append(bodies, one)
			for _, key := range []string{"_source", "_invented", "price_listed"} {
				if _, has := rec[key]; has {
					t.Errorf("%s/%s has %s", c.Name, id, key)
				}
			}
		}
		for _, b := range bodies {
			for _, secret := range secrets {
				if strings.Contains(b, secret) {
					t.Errorf("%s: a body holds %q", c.Name, secret)
				}
			}
		}
	}

	// Orders are no collection of the generic reader: it answers neither the list nor a record.
	for _, target := range []string{"/v1/orders", "/v1/orders/doi-24817"} {
		if code, _, _ := get(t, pool, target); code != http.StatusNotFound {
			t.Errorf("%s answered %d, want 404 (served by internal/orders)", target, code)
		}
	}
	_, pay, _ := get(t, pool, "/v1/payments")
	for _, it := range pay["items"].([]any) {
		p := it.(map[string]any)
		if p["icon"] == "" || p["icon"] == nil || p["online"] == nil {
			t.Errorf("a way to pay lost its icon or online flag: %v", p)
		}
	}
}
