package data

import (
	"context"
	"fmt"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/db"
	"github.com/ThanhNV121097/project-007ec16a/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests read the query language against PostgreSQL, through the real
// handler and the real pool, on shoes they insert themselves. They skip without
// TEST_DATABASE_URL, like every test that needs a database.

// shoe is one fixture. A shoe's sizes are columns until the checkout story
// moves them into per-size counts; insertShoe is the one place that knows the
// form, so the tests above it do not change when it does.
type shoe struct {
	id, name, model, description string
	line, category, colour       string
	status, released             string
	price                        int64
	rating                       float64
	sizes, low, out              []string
}

func insertShoe(t *testing.T, pool *pgxpool.Pool, s shoe) {
	t.Helper()
	if s.status == "" {
		s.status = "live"
	}
	if s.released == "" {
		s.released = "2026-11-01"
	}
	orNil := func(v string) any {
		if v == "" {
			return nil
		}
		return v
	}
	list := func(v []string) []string {
		if v == nil {
			return []string{}
		}
		return v
	}
	_, err := pool.Exec(context.Background(), `INSERT INTO products
		(id, name, model, description, line_id, category_id, colour, status, released, price, rating, sizes, sizes_low, sizes_out)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::date, $10, $11::numeric, $12, $13, $14)`,
		s.id, s.name, s.model, s.description, orNil(s.line), orNil(s.category), s.colour, s.status, s.released, s.price,
		fmt.Sprintf("%.1f", s.rating), list(s.sizes), list(s.low), list(s.out))
	if err != nil {
		t.Fatalf("insert %s: %v", s.id, err)
	}
}

// emptyShop is a seeded database with no shoes, orders or reviews: the lines
// and categories stay, so a fixture can name them.
func emptyShop(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := seeded(t)
	for _, sql := range []string{"DELETE FROM order_items", "DELETE FROM orders", "DELETE FROM reviews", "DELETE FROM products"} {
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	return pool
}

func listOf(t *testing.T, pool *pgxpool.Pool, target string) ([]string, float64) {
	t.Helper()
	code, body, raw := get(t, pool, target)
	if code != 200 {
		t.Fatalf("%s: %d %s", target, code, raw)
	}
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatalf("%s: items is %T, want a list: %s", target, body["items"], raw)
	}
	total, _ := body["total"].(float64)
	got := ids(items)
	if got == nil {
		got = []string{}
	}
	return got, total
}

// expectSet checks which records a request returns, whatever their order, and
// that total counts them all.
func expectSet(t *testing.T, pool *pgxpool.Pool, target string, want ...string) {
	t.Helper()
	got, total := listOf(t, pool, target)
	have := append([]string{}, got...)
	need := append([]string{}, want...)
	sort.Strings(have)
	sort.Strings(need)
	if !reflect.DeepEqual(have, need) || total != float64(len(want)) {
		t.Errorf("%s: got %v (total %v), want %v", target, got, total, want)
	}
}

// expectOrder checks the records a request returns, in order.
func expectOrder(t *testing.T, pool *pgxpool.Pool, target string, want ...string) {
	t.Helper()
	got, total := listOf(t, pool, target)
	if !reflect.DeepEqual(got, append([]string{}, want...)) || total != float64(len(want)) {
		t.Errorf("%s: got %v (total %v), want %v", target, got, total, want)
	}
}

func expectAmong(t *testing.T, pool *pgxpool.Pool, target string, want ...string) {
	t.Helper()
	got, _ := listOf(t, pool, target)
	for _, w := range want {
		found := false
		for _, g := range got {
			found = found || g == w
		}
		if !found {
			t.Errorf("%s: %s is not among %v", target, w, got)
		}
	}
}

// SC-1 to SC-4, SC-8, SC-9: every word, either order, any case, accents and
// the stroked d ignored, and % and _ read as themselves.
func TestSearchFindsEveryWordIgnoringCaseAccentsAndTheStrokedD(t *testing.T) {
	pool := emptyShop(t)
	insertShoe(t, pool, shoe{id: "samba-og-black", name: "Samba OG Black", model: "Samba OG", description: "Worn in Cửa Lò once", line: "samba", category: "women"})
	insertShoe(t, pool, shoe{id: "dai-lo", name: "Đại Lộ Runner", description: "A road shoe", line: "megaride", category: "men"})
	insertShoe(t, pool, shoe{id: "hundred", name: "Pure Leather", description: "Now 100% leather", line: "adizero", category: "men"})
	insertShoe(t, pool, shoe{id: "under-score", name: "Under_score Low", description: "Plain", line: "adizero", category: "men"})
	insertShoe(t, pool, shoe{id: "plain", name: "Plain Court", description: "Nothing special", line: "adizero", category: "men"})

	expectSet(t, pool, "/v1/products?q=samba%20og", "samba-og-black")
	expectSet(t, pool, "/v1/products?q=og%20samba", "samba-og-black")
	expectSet(t, pool, "/v1/products?q=SAMBA+Og", "samba-og-black")
	expectAmong(t, pool, "/v1/products?q=cua%20lo", "samba-og-black")
	expectAmong(t, pool, "/v1/products?q=CUA%20L%C3%92", "samba-og-black")
	expectAmong(t, pool, "/v1/products?q=C%E1%BB%ACA%20l%C3%B2", "samba-og-black")
	expectAmong(t, pool, "/v1/products?q=dai", "dai-lo")
	expectAmong(t, pool, "/v1/products?q=%C4%90%E1%BA%A0I", "dai-lo")
	// A word is found as part of a value.
	expectAmong(t, pool, "/v1/products?q=lea", "hundred")
	// % and _ mean themselves.
	expectSet(t, pool, "/v1/products?q=100%25", "hundred")
	expectSet(t, pool, "/v1/products?q=_", "under-score")
	expectSet(t, pool, "/v1/products?q=%25", "hundred")
	expectSet(t, pool, "/v1/products?q=zzzz")
	// Only spaces is no search; so is an empty q.
	all, _ := listOf(t, pool, "/v1/products")
	for _, q := range []string{"q=%20%20", "q="} {
		if got, total := listOf(t, pool, "/v1/products?"+q); !reflect.DeepEqual(got, all) || total != 5 {
			t.Errorf("%s: got %v (%v), want %v", q, got, total, all)
		}
	}
}

// SC-3: the line's and the category's name are searched, live.
func TestSearchReachesTheLinesAndCategoriesNameLive(t *testing.T) {
	pool := emptyShop(t)
	insertShoe(t, pool, shoe{id: "g-one", name: "Indoor Solar", line: "gazelle", category: "men"})
	insertShoe(t, pool, shoe{id: "g-two", name: "Indoor Mauve", line: "gazelle", category: "women"})
	insertShoe(t, pool, shoe{id: "s-one", name: "Court Low", line: "samba", category: "women"})
	insertShoe(t, pool, shoe{id: "k-one", name: "Court Mid", line: "samba", category: "kids"})

	expectSet(t, pool, "/v1/products?q=gazelle", "g-one", "g-two")
	expectSet(t, pool, "/v1/products?q=women", "g-two", "s-one")
	expectSet(t, pool, "/v1/products?q=gazelle%20women", "g-two")
	// A rename is found at once, with no write to the shoes.
	if _, err := pool.Exec(context.Background(), "UPDATE lines SET name = 'Zebra' WHERE id = 'gazelle'"); err != nil {
		t.Fatal(err)
	}
	expectSet(t, pool, "/v1/products?q=zebra", "g-one", "g-two")
	expectSet(t, pool, "/v1/products?q=gazelle")
}

// SC-5, SC-6, SC-7, SC-8: any-of, ranges, sort, and a page that never repeats
// or skips a record.
func TestFiltersRangesAndSortsOfTheShop(t *testing.T) {
	pool := emptyShop(t)
	insertShoe(t, pool, shoe{id: "p1", name: "One", line: "samba", category: "women", price: 2790000, rating: 4.7, released: "2026-11-17"})
	insertShoe(t, pool, shoe{id: "p2", name: "Two", line: "samba", category: "women", price: 3090000, rating: 4.9, released: "2026-11-10"})
	insertShoe(t, pool, shoe{id: "p3", name: "Three", line: "gazelle", category: "women", price: 3290000, rating: 0, released: "2026-11-03"})
	insertShoe(t, pool, shoe{id: "p4", name: "Four", line: "gazelle", category: "men", price: 3290000, rating: 4.8, released: "2026-10-02"})
	insertShoe(t, pool, shoe{id: "p5", name: "Five", line: "megaride", category: "men", price: 5940000, rating: 4.6, released: "2026-09-01"})
	insertShoe(t, pool, shoe{id: "p6", name: "Six", line: "samba", category: "kids", price: 3000000, rating: 4.5, released: "2026-08-01"})

	expectSet(t, pool, "/v1/products?category=women,kids", "p1", "p2", "p3", "p6")
	expectSet(t, pool, "/v1/products?category=women,kids&line=samba", "p1", "p2", "p6")
	expectSet(t, pool, "/v1/products?category=women&line=samba,megaride", "p1", "p2")
	everything, _ := listOf(t, pool, "/v1/products")
	if got, _ := listOf(t, pool, "/v1/products?category="); !reflect.DeepEqual(got, everything) {
		t.Errorf("an empty category filtered: %v, want %v", got, everything)
	}

	expectSet(t, pool, "/v1/products?price_from=3000000&price_to=3999999", "p2", "p3", "p4", "p6")
	expectSet(t, pool, "/v1/products?price_from=3000000&price_to=3000000", "p6")
	expectSet(t, pool, "/v1/products?price_from=3000001", "p2", "p3", "p4", "p5")
	expectSet(t, pool, "/v1/products?price_to=2999999.5", "p1")
	expectSet(t, pool, "/v1/products?price_from=5&price_to=3")
	expectSet(t, pool, "/v1/products?price_ranges=..2999999,4000000..", "p1", "p5")
	expectSet(t, pool, "/v1/products?price_ranges=..2999999,3000000..3000000", "p1", "p6")
	expectSet(t, pool, "/v1/products?released_from=2026-10-02&released_to=2026-11-03", "p3", "p4")
	expectSet(t, pool, "/v1/products?released_ranges=..2026-08-01,2026-11-10..", "p1", "p2", "p6")
	expectSet(t, pool, "/v1/products?price=3290000,2790000", "p1", "p3", "p4")

	expectOrder(t, pool, "/v1/products?sort=price", "p1", "p6", "p2", "p3", "p4", "p5")
	expectOrder(t, pool, "/v1/products?sort=-price", "p5", "p3", "p4", "p2", "p6", "p1")
	expectOrder(t, pool, "/v1/products?sort=-released", "p1", "p2", "p3", "p4", "p5", "p6")
	expectOrder(t, pool, "/v1/products?sort=released", "p6", "p5", "p4", "p3", "p2", "p1")
	// A shoe with no rating is last, whichever way the sort runs.
	expectOrder(t, pool, "/v1/products?sort=-rating", "p2", "p4", "p1", "p5", "p6", "p3")
	expectOrder(t, pool, "/v1/products?sort=rating", "p6", "p5", "p1", "p4", "p2", "p3")
	expectOrder(t, pool, "/v1/products?sort=name", "p5", "p4", "p1", "p6", "p3", "p2")
	expectOrder(t, pool, "/v1/products?status=live&category=women,kids&sort=-price", "p3", "p2", "p6", "p1")

	// SC-8: pages of one, in turn, show every shoe exactly once, though two
	// have the same price.
	var seen []string
	for page := 1; page <= 6; page++ {
		got, total := listOf(t, pool, fmt.Sprintf("/v1/products?sort=price&limit=1&page=%d", page))
		if len(got) != 1 || total != 6 {
			t.Fatalf("page %d: %v (%v)", page, got, total)
		}
		seen = append(seen, got...)
	}
	if want := []string{"p1", "p6", "p2", "p3", "p4", "p5"}; !reflect.DeepEqual(seen, want) {
		t.Errorf("pages of one read %v, want %v", seen, want)
	}
}

// SC-9, SC-16: a size matches a shoe that lists it and has not sold it out; a
// size on its last pairs is still available.
func TestSizeMatchesOnlyAvailableSizes(t *testing.T) {
	pool := emptyShop(t)
	insertShoe(t, pool, shoe{id: "a", name: "A", sizes: []string{"36", "38", "40"}, out: []string{"38"}})
	insertShoe(t, pool, shoe{id: "b", name: "B", sizes: []string{"38", "40"}, low: []string{"40"}})
	insertShoe(t, pool, shoe{id: "c", name: "C", sizes: []string{"36 2/3", "38"}})
	insertShoe(t, pool, shoe{id: "d", name: "D", sizes: []string{"36"}, out: []string{"36"}})

	expectSet(t, pool, "/v1/products?size=38", "b", "c")
	expectSet(t, pool, "/v1/products?size=40", "a", "b")
	expectSet(t, pool, "/v1/products?size=38,40", "a", "b", "c")
	expectSet(t, pool, "/v1/products?size=36%202%2F3", "c")
	expectSet(t, pool, "/v1/products?size=36", "a")
	expectSet(t, pool, "/v1/products?size=99")
	expectSet(t, pool, "/v1/products?size=99,98")
	expectSet(t, pool, "/v1/products?size=38&sort=-price&q=b", "b")
	all, _ := listOf(t, pool, "/v1/products")
	if got, _ := listOf(t, pool, "/v1/products?size=%20,"); !reflect.DeepEqual(got, all) {
		t.Errorf("an empty size filtered: %v", got)
	}
}

// SC-11: a draft shoe that matches everything else is not returned when the
// request carries status=live; a request without it still sees it (the API
// refusing a draft to a visitor whatever is asked is the admin story's).
func TestADraftShoeIsLeftOutByStatusLive(t *testing.T) {
	pool := emptyShop(t)
	insertShoe(t, pool, shoe{id: "live-one", name: "Samba Live", line: "samba", category: "women", price: 2790000, sizes: []string{"38"}})
	insertShoe(t, pool, shoe{id: "draft-one", name: "Samba Draft", line: "samba", category: "women", price: 2790000, sizes: []string{"38"}, status: "draft"})
	expectSet(t, pool, "/v1/products?q=samba&status=live&category=women&size=38&price_to=2999999", "live-one")
	expectSet(t, pool, "/v1/products?q=samba", "live-one", "draft-one")
}

// SC-12: the true count over several pages.
func TestTotalIsEveryMatchAcrossPages(t *testing.T) {
	pool := emptyShop(t)
	for i := 1; i <= 30; i++ {
		insertShoe(t, pool, shoe{id: fmt.Sprintf("bulk-%02d", i), name: fmt.Sprintf("Samba Bulk %d", i), line: "samba", category: "women"})
	}
	insertShoe(t, pool, shoe{id: "other", name: "Other", line: "adizero", category: "men"})
	for page, n := range map[int]int{1: 12, 2: 12, 3: 6, 4: 0} {
		got, total := listOf(t, pool, fmt.Sprintf("/v1/products?q=samba&page=%d&limit=12", page))
		if len(got) != n || total != 30 {
			t.Errorf("page %d: %d items, total %v; want %d items, total 30", page, len(got), total, n)
		}
	}
}

// SC-10, SC-9: the seeded shoes.
func TestTheSeededShoesAnswerTheStoryExamples(t *testing.T) {
	pool := seeded(t)
	expectSet(t, pool, "/v1/products?status=live&category=women&line=samba&size=38&price_to=2999999", "samba-og-w-wonder-quartz")
	expectOrder(t, pool, "/v1/products?category=women&line=samba&size=38&price_to=2999999&q=og&sort=price", "samba-og-w-wonder-quartz")
	// 38 is sold out on the Handball Spezial; 40 is on its last pairs on the Samba OG Black Crystal.
	got, _ := listOf(t, pool, "/v1/products?size=38&limit=100")
	for _, id := range got {
		if id == "handball-spezial-w-earth-strata" {
			t.Error("a sold-out size matched")
		}
	}
	expectAmong(t, pool, "/v1/products?size=40", "samba-og-w-core-black")
	expectAmong(t, pool, "/v1/products?q=samba%20og", "samba-og-w-core-black", "samba-og-w-crystal-white", "samba-og-w-wonder-quartz")
	expectAmong(t, pool, "/v1/products?q=gazelle", "gazelle-indoor-solar-gold")
	expectSet(t, pool, "/v1/products?q=samba&category=women,kids&sort=-price", "samba-og-w-core-black", "samba-og-w-crystal-white", "samba-og-w-wonder-quartz", "samba-og-kids-wonder-mauve", "samba-og-kids-cloud-white")
	got, _ = listOf(t, pool, "/v1/products?price_from=3000000&price_to=3999999")
	for _, id := range got {
		if id == "samba-og-w-wonder-quartz" || id == "adizero-adios-wonder-blue" {
			t.Errorf("%s is outside the band", id)
		}
	}
	// A search never serves the column it reads.
	_, body, _ := get(t, pool, "/v1/products?q=samba&limit=1")
	if item := body["items"].([]any)[0].(map[string]any); item["search_text"] != nil || len(item) != 29 {
		t.Errorf("a shoe has %d fields, search_text %v", len(item), item["search_text"])
	}
}

// SC-13, SC-14, SC-16: orders are searched by number and customer only, and
// reviews by the customer's name, the text and the shoe's name.
func TestOrdersAndReviewsAreSearchedByTheirNamedFields(t *testing.T) {
	pool := seeded(t)
	if _, err := pool.Exec(context.Background(), `INSERT INTO orders (id, number, payment_method_id, delivery_option_id, customer, phone, address, city, note, total)
		VALUES ('doi-90001', 'DOI-90001', 'momo', 'standard', 'Trần Lan', '0912345678', 'Zxqv Nam Street 12', 'Zxqv City', 'nam nam nam zxqv', 1000)`); err != nil {
		t.Fatal(err)
	}
	got, _ := listOf(t, pool, "/v1/orders?q=nam&limit=100")
	has := map[string]bool{}
	for _, id := range got {
		has[id] = true
	}
	if !has["doi-24817"] || has["doi-90001"] {
		t.Errorf("q=nam: got %v", got)
	}
	expectSet(t, pool, "/v1/orders?q=90001", "doi-90001")
	expectAmong(t, pool, "/v1/orders?q=tran%20lan", "doi-90001")
	// A phone number, an address and a note are never searched.
	expectSet(t, pool, "/v1/orders?q=0912345678")
	expectSet(t, pool, "/v1/orders?q=0938")
	expectSet(t, pool, "/v1/orders?q=zxqv")
	// size on orders is the first pair's size, exactly.
	expectAmong(t, pool, "/v1/orders?size=39%201%2F3&limit=100", "doi-24817")

	expectOrder(t, pool, "/v1/reviews?product=samba-og-w-core-black&sort=-date", "r-01", "r-02", "r-03")
	expectAmong(t, pool, "/v1/reviews?q=Ph%C6%B0%C6%A1ng", "r-01")
	expectAmong(t, pool, "/v1/reviews?q=phuong", "r-01")
	expectAmong(t, pool, "/v1/reviews?q=crystals", "r-01")
	expectAmong(t, pool, "/v1/reviews?q=wonder%20quartz", "r-07", "r-08", "r-09")
	expectSet(t, pool, "/v1/reviews?q=zzzz")
}

// SC-15, SC-16: a bad parameter is a 400 naming it, with no records.
func TestABadParameterIsAFaultAndAnEmptyAnswerIsNot(t *testing.T) {
	pool := seeded(t)
	for target, key := range map[string]string{
		"/v1/products?sort=zzz":                       "sort",
		"/v1/products?price_from=abc":                 "price_from",
		"/v1/products?price_to=1e3":                   "price_to",
		"/v1/products?released_to=2026-13-40":         "released_to",
		"/v1/products?price_ranges=3000000":           "price_ranges",
		"/v1/products?q=" + strings.Repeat("a", 101):  "q",
		"/v1/products?sort=price_listed":              "sort",
		"/v1/orders?sort=phone":                       "sort",
		"/v1/orders?phone_from=1":                     "phone_from",
		"/v1/products?q=a%00b":                        "q",
		"/v1/orders?placed_at_from=yesterday":         "placed_at_from",
		"/v1/products?size=38%00":                     "size",
		"/v1/products?category=women&price_from=zero": "price_from",
	} {
		code, body, raw := get(t, pool, target)
		errs, _ := body["errors"].(map[string]any)
		if code != 400 || body["error"] != "bad_request" || errs[key] == nil || body["items"] != nil {
			t.Errorf("%.80s: %d %s", target, code, raw)
		}
	}
	expectSet(t, pool, "/v1/products?size=99")
	_, body, _ := get(t, pool, "/v1/products?size=99")
	if items, ok := body["items"].([]any); !ok || len(items) != 0 || body["total"] != float64(0) {
		t.Errorf("size=99: %v", body)
	}
}

// The database keeps search_text: it follows every write of the shoe and it
// cannot be written.
func TestSearchTextIsKeptByTheDatabaseAndCannotBeWritten(t *testing.T) {
	pool := emptyShop(t)
	ctx := context.Background()
	insertShoe(t, pool, shoe{id: "cua-lo", name: "Samba", model: "OG", description: "Cửa Lò"})
	var text string
	if err := pool.QueryRow(ctx, "SELECT search_text FROM products WHERE id = 'cua-lo'").Scan(&text); err != nil || !strings.Contains(text, "cua lo") || !strings.Contains(text, "samba og") {
		t.Fatalf("search_text %q, %v", text, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE products SET description = 'Đại lộ' WHERE id = 'cua-lo'"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT search_text FROM products WHERE id = 'cua-lo'").Scan(&text); err != nil || strings.Contains(text, "cua lo") || !strings.Contains(text, "dai lo") {
		t.Errorf("search_text did not follow the update: %q, %v", text, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE products SET search_text = 'x' WHERE id = 'cua-lo'"); err == nil {
		t.Error("search_text was written")
	}
	expectSet(t, pool, "/v1/products?q=dai%20lo", "cua-lo")
	expectSet(t, pool, "/v1/products?q=cua")
}

// The migration on a table that already has rows keeps every one of them, fills
// search_text for each, and its down file takes the column away again.
func TestSearchTextMigrationKeepsEveryRowOfAPopulatedTable(t *testing.T) {
	pool := schemaPool(t)
	ctx := context.Background()
	before := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") && !strings.Contains(e.Name(), "products_search_text") {
			data, err := fs.ReadFile(migrations.FS, e.Name())
			if err != nil {
				t.Fatal(err)
			}
			before[e.Name()] = &fstest.MapFile{Data: data}
		}
	}
	if err := db.Migrate(ctx, pool, before, quiet()); err != nil {
		t.Fatalf("the earlier migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO products (id, name, model, description, price, status)
		SELECT 'old-' || g, 'Samba OG ' || g, 'Samba OG', 'Cửa Lò ' || g, 2000000 + g, 'live' FROM generate_series(1, 2000) g`); err != nil {
		t.Fatal(err)
	}
	rows := count(t, pool, "SELECT count(*) FROM products")
	if err := db.Migrate(ctx, pool, migrations.FS, quiet()); err != nil {
		t.Fatalf("the search_text migration on %d shoes: %v", rows, err)
	}
	if after := count(t, pool, "SELECT count(*) FROM products"); after != rows {
		t.Errorf("%d shoes became %d", rows, after)
	}
	if n := count(t, pool, "SELECT count(*) FROM products WHERE id LIKE 'old-%' AND (search_text = '' OR search_text NOT LIKE '%cua lo%')"); n != 0 {
		t.Errorf("%d shoes have no search_text", n)
	}
	down, err := fs.ReadFile(migrations.FS, "20261125130000_products_search_text.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("down: %v", err)
	}
	if n := count(t, pool, `SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema()
		AND table_name = 'products' AND column_name = 'search_text'`); n != 0 {
		t.Error("the down file left the column")
	}
	if after := count(t, pool, "SELECT count(*) FROM products"); after != rows {
		t.Errorf("the down file lost shoes: %d of %d", after, rows)
	}
}

// SC-19: with 10,000 shoes a two-word search with one category and a sort
// answers within 500 ms at the 95th percentile, over 200 sequential requests.
func TestASearchOverTenThousandShoesIsFast(t *testing.T) {
	pool := emptyShop(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (id, name, model, description, line_id, category_id, colour, price, status, released, rating, sizes)
		SELECT 'perf-' || g, 'Samba OG Colourway ' || g, 'Samba OG', 'Leather upper, gum sole and a note about Cửa Lò ' || g,
		       (ARRAY['samba', 'gazelle', 'adizero'])[1 + g % 3], (ARRAY['women', 'men', 'kids'])[1 + g % 3],
		       'black', 2000000 + (g % 40) * 100000, 'live', DATE '2026-01-01' + (g % 300), (g % 50) / 10.0, ARRAY['38', '39']
		FROM generate_series(1, 10000) g`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "ANALYZE products"); err != nil {
		t.Fatal(err)
	}
	const runs = 200
	took := make([]time.Duration, 0, runs)
	for i := 0; i < runs; i++ {
		start := time.Now()
		code, body, raw := get(t, pool, "/v1/products?q=samba%20og&category=women&sort=-price&status=live")
		took = append(took, time.Since(start))
		if code != 200 || body["total"] == float64(0) {
			t.Fatalf("%d %.200s", code, raw)
		}
	}
	sort.Slice(took, func(i, j int) bool { return took[i] < took[j] })
	p50, p95 := took[runs/2], took[runs*95/100]
	t.Logf("p50 %v, p95 %v over %d requests on 10,000 shoes", p50, p95, runs)
	if p95 > 500*time.Millisecond {
		t.Errorf("p95 %v is over 500 ms (p50 %v)", p95, p50)
	}
}
