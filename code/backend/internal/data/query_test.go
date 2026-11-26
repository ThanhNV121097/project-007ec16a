package data

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

// ordersShape is what the orders collection looked like while the generic reader
// served it. Orders are read by internal/orders now and are no longer in the
// registry, but the grammar still has to read an instant, a date, a nullable
// column of a left join and a private column, so these tests keep the shape.
var ordersShape = build("orders", "orders t LEFT JOIN order_items oi ON oi.order_id = t.id AND oi.line_no = 1",
	text("id"), text("number"),
	Column{Name: "placed_at", Kind: Instant,
		Select: "to_char(t.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD\"T\"HH24:MI:SS') || '+07:00'",
		Filter: "t.placed_at"},
	text("status"),
	Column{Name: "product", Kind: Text, Select: "coalesce(oi.product_id,'')", Filter: "oi.product_id", Nullable: true},
	Column{Name: "size", Kind: Text, Select: "coalesce(oi.size,'')", Filter: "oi.size", Nullable: true},
	Column{Name: "qty", Kind: Int, Select: "coalesce(oi.qty,0)", Filter: "oi.qty", Nullable: true},
	col("total", Int),
	ref("payment", "payment_method_id"), col("paid", Bool), ref("delivery", "delivery_option_id"),
	text("carrier"), text("tracking"),
	Column{Name: "eta", Kind: Date, Select: "coalesce(to_char(t.eta,'YYYY-MM-DD'),'')", Filter: "t.eta", Nullable: true},
	text("customer"),
	private("phone"), private("email"), private("address"),
	text("city"),
	private("note")).
	searching(over("number", "customer"))

// collectionOf is a collection of the registry, or the shape of the orders.
func collectionOf(name string) *Collection {
	if name == "orders" {
		return ordersShape
	}
	return byName[name]
}

func parse(t *testing.T, collection, query string) (listRequest, map[string]string) {
	t.Helper()
	q, err := url.ParseQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	return parseList(collectionOf(collection), q)
}

// whereOf is the condition and the bound values a query string stands for.
func whereOf(t *testing.T, collection, query string) (string, []any) {
	t.Helper()
	req, faults := parse(t, collection, query)
	if faults != nil {
		t.Fatalf("%s?%s: %v", collection, query, faults)
	}
	return where(collectionOf(collection), req)
}

func orderOf(t *testing.T, collection, query string) string {
	t.Helper()
	req, faults := parse(t, collection, query)
	if faults != nil {
		t.Fatalf("%s?%s: %v", collection, query, faults)
	}
	return orderBy(req)
}

func TestPageAndLimitDefaults(t *testing.T) {
	req, faults := parse(t, "products", "")
	if faults != nil || req.page != 1 || req.limit != 12 || len(req.filters) != 0 || len(req.words) != 0 || req.sort != nil {
		t.Errorf("got %+v %v", req, faults)
	}
}

func TestLimitAboveAHundredIsServedAsAHundred(t *testing.T) {
	for _, v := range []string{"101", "500", "99999999999999999999999"} {
		req, faults := parse(t, "products", "limit="+v)
		if faults != nil || req.limit != 100 {
			t.Errorf("limit=%s: got %d %v, want 100", v, req.limit, faults)
		}
	}
	req, _ := parse(t, "products", "page=2&limit=5")
	if req.page != 2 || req.limit != 5 {
		t.Errorf("got %+v", req)
	}
}

func TestPageAndLimitThatAreNotWholeNumbersFromOneAreFaults(t *testing.T) {
	for _, name := range []string{"page", "limit"} {
		for _, v := range []string{"0", "-1", "abc", "1.5", "", "+3", "0000000000000"} {
			_, faults := parse(t, "products", name+"="+v)
			if faults[name] == "" {
				t.Errorf("%s=%q was accepted", name, v)
			}
		}
	}
	if _, faults := parse(t, "products", "page=2147483648"); faults["page"] == "" {
		t.Error("a page above the maximum was accepted")
	}
	_, faults := parse(t, "products", "page=0&limit=0&price=x")
	if len(faults) != 3 {
		t.Errorf("every fault is reported together, got %v", faults)
	}
}

func TestAFilterOnAPrivateColumnIsAFaultWhateverItsValue(t *testing.T) {
	cases := map[string][]string{
		"orders":   {"phone", "email", "address", "note", "_source", "_invented"},
		"products": {"price_listed", "_source", "_invented"},
		"lines":    {"_source", "_invented"},
		"faqs":     {"_source"},
	}
	for collection, names := range cases {
		for _, name := range names {
			for _, v := range []string{"x", "", "a,b"} {
				req, faults := parse(t, collection, name+"="+url.QueryEscape(v))
				if faults[name] != name+" cannot be used as a filter." || len(req.filters) != 0 {
					t.Errorf("%s?%s=%q: got %v", collection, name, v, faults)
				}
			}
		}
	}
	// note is an ordinary field of delivery; price_listed is no field of orders.
	if _, faults := parse(t, "delivery", "note=x"); faults != nil {
		t.Errorf("delivery?note was refused: %v", faults)
	}
	if _, faults := parse(t, "orders", "price_listed=x"); faults != nil {
		t.Errorf("orders?price_listed was refused: %v", faults)
	}
}

// A range key on a column that is not public is as refused as the column.
func TestARangeOnAPrivateColumnIsAFaultNamingTheKey(t *testing.T) {
	for collection, keys := range map[string][]string{
		"orders":   {"phone_from", "email_to", "address_ranges", "note_from", "_source_to"},
		"products": {"price_listed_from", "price_listed_ranges", "_invented_to"},
	} {
		for _, key := range keys {
			for _, v := range []string{"1", "", "..5"} {
				req, faults := parse(t, collection, key+"="+v)
				if faults[key] != key+" cannot be used as a filter." || len(req.ranges) != 0 {
					t.Errorf("%s?%s=%q: got %v", collection, key, v, faults)
				}
			}
		}
	}
}

func TestUnknownNamesAndEmptyValuesAreIgnored(t *testing.T) {
	plain, _ := parse(t, "products", "")
	plainSQL, _ := listSQL(byName["products"], plain)
	for _, q := range []string{
		"zzz=1", "Category=women", "category=", "category=,", "category=%20,%20", "q=", "q=%20%20", "sort=", "sort=%20",
		"size=", "size=,", "price_from=", "price_to=%20", "price_ranges=", "name_from=a", "featured_to=1", "sizes_from=x",
		"limit=12&page=1", "id_from=a", "status_ranges=a..b",
	} {
		req, faults := parse(t, "products", q)
		if faults != nil {
			t.Errorf("%q: %v", q, faults)
			continue
		}
		sql, args := listSQL(byName["products"], req)
		if sql != plainSQL || len(args) != 2 {
			t.Errorf("%q: the statement differs from none: %s %v", q, sql, args)
		}
	}
	// size is a shoe filter only: anywhere else it is an ordinary field, or
	// nothing at all.
	if req, faults := parse(t, "lines", "size=38"); faults != nil || len(req.filters) != 0 || len(req.sizes) != 0 {
		t.Errorf("lines?size: %+v %v", req, faults)
	}
}

func TestFiltersAreReadByKind(t *testing.T) {
	req, faults := parse(t, "products", "category=women&status=live&price=3290000&featured=true&rating=4.8&released=2026-11-03&sizes_low=40")
	if faults != nil || len(req.filters) != 7 {
		t.Fatalf("got %d filters, %v", len(req.filters), faults)
	}
	cond, args := where(byName["products"], req)
	// Filters follow the registry's column order, not the query string's.
	for _, want := range []string{"t.category_id = $1", "t.price = $2::bigint", "t.status = $3", "t.featured = $4::boolean", "t.released = $5::date", "$6 = ANY(ps.sizes_low)", "t.rating = $7::numeric"} {
		if !strings.Contains(cond, want) {
			t.Errorf("where %q lacks %q", cond, want)
		}
	}
	if len(args) != 7 || args[1] != int64(3290000) || args[3] != true {
		t.Errorf("args %v", args)
	}
}

func TestAnyOfListsAreBoundAsOneArrayOfTheFieldsKind(t *testing.T) {
	for _, c := range []struct {
		collection, query, cond string
		arg                     any
	}{
		{"products", "category=women,kids", "t.category_id = ANY($1::text[])", []string{"women", "kids"}},
		{"products", "category=%20women%20,%20,kids,women", "t.category_id = ANY($1::text[])", []string{"women", "kids"}},
		{"products", "price=2790000,3290000", "t.price = ANY($1::bigint[])", []int64{2790000, 3290000}},
		{"products", "rating=4.8,4.9", "t.rating = ANY($1::text[]::numeric[])", []string{"4.8", "4.9"}},
		{"products", "featured=true,false", "t.featured = ANY($1::boolean[])", []bool{true, false}},
		{"products", "released=2026-11-03,2026-11-10", "t.released = ANY($1::text[]::date[])", []string{"2026-11-03", "2026-11-10"}},
		{"products", "sizes_low=40,38", "ps.sizes_low && $1::text[]", []string{"40", "38"}},
		// One item left after the empty ones are dropped is the plain form, so
		// status=live keeps the equality an index is walked on.
		{"products", "category=,women,", "t.category_id = $1", "women"},
		{"products", "status=live", "t.status = $1", "live"},
	} {
		cond, args := whereOf(t, c.collection, c.query)
		if cond != c.cond || len(args) != 1 || !reflect.DeepEqual(args[0], c.arg) {
			t.Errorf("%s?%s: %q %#v, want %q %#v", c.collection, c.query, cond, args, c.cond, c.arg)
		}
	}
}

func TestAListOfMomentsIsBoundAsInstants(t *testing.T) {
	cond, args := whereOf(t, "orders", "placed_at=2026-11-23T16:05:00%2B07:00,2026-11-22T16:05:00%2B07:00")
	moments, ok := args[0].([]time.Time)
	if cond != "t.placed_at = ANY($1::timestamptz[])" || len(args) != 1 || !ok || len(moments) != 2 {
		t.Fatalf("%q %#v", cond, args)
	}
	if want, _ := time.Parse(time.RFC3339, "2026-11-23T16:05:00+07:00"); !moments[0].Equal(want) {
		t.Errorf("got %v, want %v", moments[0], want)
	}
}

func TestOneBadItemOfAListIsAFaultNamingTheField(t *testing.T) {
	for q, field := range map[string]string{
		"price=1,abc": "price", "featured=true,yes": "featured", "rating=4.8,x": "rating", "released=2026-11-03,today": "released",
	} {
		if _, faults := parse(t, "products", q); faults[field] == "" {
			t.Errorf("%s was accepted", q)
		}
	}
	if _, faults := parse(t, "orders", "placed_at=2026-11-23T16:05:00%2B07:00,yesterday"); faults["placed_at"] == "" {
		t.Error("a list holding yesterday was accepted")
	}
}

func TestAValueThatDoesNotReadAsItsKindIsAFault(t *testing.T) {
	for q, field := range map[string]string{
		"price=abc": "price", "price=1.5": "price", "featured=yes": "featured", "rating=x": "rating",
		"released=2026-13-40": "released", "released=today": "released", "review_count=1e3": "review_count",
	} {
		if _, faults := parse(t, "products", q); faults[field] == "" {
			t.Errorf("%s was accepted", q)
		}
	}
	if _, faults := parse(t, "orders", "placed_at=yesterday"); faults["placed_at"] == "" {
		t.Error("placed_at=yesterday was accepted")
	}
	if _, faults := parse(t, "orders", "placed_at=2026-11-23T16:05:00%2B07:00"); faults != nil {
		t.Errorf("a real instant was refused: %v", faults)
	}
}

// A value PostgreSQL cannot hold — a NUL, bytes that are not UTF-8, the year
// 0000, a number of a hundred digits — would fail the statement and be answered
// 500. It is the caller's typo, so it is a fault at the boundary, for every key.
func TestAValuePostgreSQLWouldRefuseIsAFaultNotAnError(t *testing.T) {
	for _, c := range []struct{ collection, query, field string }{
		{"products", "category=a%00b", "category"},
		{"products", "status=%00", "status"},
		{"products", "colour=%ff%fe", "colour"},
		{"products", "sizes_low=40%00", "sizes_low"},
		{"products", "category=women,a%00b", "category"},
		{"reviews", "product=%c3%28", "product"},
		{"products", "id=%80", "id"},
		{"products", "released=0000-01-01", "released"},
		{"reviews", "date=0000-12-31", "date"},
		{"orders", "eta=0000-01-01", "eta"},
		{"orders", "placed_at=0000-01-01T00:00:00%2B07:00", "placed_at"},
		{"products", "rating=" + strings.Repeat("9", 19), "rating"},
		{"products", "price=" + strings.Repeat("9", 19), "price"},
		{"products", "q=a%00b", "q"},
		{"products", "q=%ff", "q"},
		{"products", "sort=pri%00ce", "sort"},
		{"products", "sort=%c3%28", "sort"},
		{"products", "size=38%00", "size"},
		{"products", "size=%ff", "size"},
		{"products", "price_from=1%00", "price_from"},
		{"products", "price_to=%ff", "price_to"},
		{"products", "price_ranges=..1%00", "price_ranges"},
		{"products", "released_from=0000-01-01", "released_from"},
		{"orders", "placed_at_to=0000-12-31", "placed_at_to"},
	} {
		req, faults := parse(t, c.collection, c.query)
		if faults[c.field] == "" || len(req.filters) != 0 || len(req.words) != 0 {
			t.Errorf("%s?%s: got %v", c.collection, c.query, faults)
		}
	}
	// The year 0001 and ordinary accented text are fine.
	for _, c := range []struct{ collection, query string }{
		{"products", "released=0001-01-01"},
		{"products", "colourway=C%E1%BB%ADa%20L%C3%B2"},
		{"orders", "placed_at=0001-01-01T00:00:00Z"},
		{"products", "q=C%E1%BB%ADa%20L%C3%B2"},
	} {
		if req, faults := parse(t, c.collection, c.query); faults != nil || len(req.filters)+len(req.words) == 0 {
			t.Errorf("%s?%s: got %+v, %v", c.collection, c.query, req, faults)
		}
	}
}

func TestTheFirstOfARepeatedNameIsUsed(t *testing.T) {
	req, _ := parse(t, "products", "category=women&category=men")
	if len(req.filters) != 1 || req.filters[0].arg != "women" {
		t.Errorf("got %+v", req.filters)
	}
	req, _ = parse(t, "products", "q=samba&q=gazelle&sort=price&sort=-price&size=38&size=40")
	if !reflect.DeepEqual(req.words, []string{"samba"}) || req.sort == nil || req.sort.desc || !reflect.DeepEqual(req.sizes, []string{"38"}) {
		t.Errorf("got %+v", req)
	}
}

func TestRangesAreInclusiveAndComparedAsNumbersAndDates(t *testing.T) {
	cond, args := whereOf(t, "products", "price_from=3000000&price_to=3999999")
	if cond != "(t.price >= $1::numeric) AND (t.price <= $2::numeric)" || !reflect.DeepEqual(args, []any{"3000000", "3999999"}) {
		t.Errorf("%q %v", cond, args)
	}
	// One end, a fraction, and an end above the other are all fine.
	for _, q := range []string{"price_to=2999999.5", "price_from=-5", "price_from=5&price_to=3", "rating_from=4.5"} {
		if _, faults := parse(t, "products", q); faults != nil {
			t.Errorf("%s: %v", q, faults)
		}
	}
	cond, _ = whereOf(t, "products", "released_from=2026-11-01&released_to=2026-11-30")
	if cond != "(t.released >= $1::date) AND (t.released <= $2::date)" {
		t.Errorf("%q", cond)
	}
	// An order's moment is ranged by Vietnam calendar days, the last day whole.
	cond, _ = whereOf(t, "orders", "placed_at_from=2026-11-01&placed_at_to=2026-11-30")
	for _, want := range []string{
		"t.placed_at >= ($1::date::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')",
		"t.placed_at < (($2::date + 1)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')",
	} {
		if !strings.Contains(cond, want) {
			t.Errorf("%q lacks %q", cond, want)
		}
	}
	// A column of an order that is a left join ranges on the joined table.
	if cond, _ = whereOf(t, "orders", "qty_from=2"); cond != "(oi.qty >= $1::numeric)" {
		t.Errorf("%q", cond)
	}
}

func TestAUnionOfRangesIsOrOfTheRangesAndAndOfTheEnds(t *testing.T) {
	cond, args := whereOf(t, "products", "price_ranges=..2999999,4000000..")
	if cond != "((t.price <= $1::numeric) OR (t.price >= $2::numeric))" || !reflect.DeepEqual(args, []any{"2999999", "4000000"}) {
		t.Errorf("%q %v", cond, args)
	}
	cond, args = whereOf(t, "products", "rating_ranges=4.5..5,..1")
	if cond != "((t.rating >= $1::numeric AND t.rating <= $2::numeric) OR (t.rating <= $3::numeric))" || !reflect.DeepEqual(args, []any{"4.5", "5", "1"}) {
		t.Errorf("%q %v", cond, args)
	}
	// _ranges combines with _from and _to of the same field by AND.
	cond, args = whereOf(t, "products", "price_ranges=..2999999&price_from=1000000")
	if cond != "(t.price >= $1::numeric) AND (t.price <= $2::numeric)" || !reflect.DeepEqual(args, []any{"1000000", "2999999"}) {
		t.Errorf("%q %v", cond, args)
	}
	if _, faults := parse(t, "products", "released_ranges=2026-01-01..2026-03-31,2026-07-01..2026-09-30"); faults != nil {
		t.Errorf("date ranges: %v", faults)
	}
	twenty := strings.TrimSuffix(strings.Repeat("1..2,", 20), ",")
	if _, faults := parse(t, "products", "price_ranges="+twenty); faults != nil {
		t.Errorf("twenty ranges: %v", faults)
	}
}

func TestARangeThatIsNotARangeIsAFaultNamingTheKey(t *testing.T) {
	for q, want := range map[string]string{
		"price_from=abc":                  "price_from must be a number.",
		"price_to=1e3":                    "price_to must be a number.",
		"price_from=1.1234567":            "price_from must be a number.",
		"rating_to=x":                     "rating_to must be a number.",
		"released_to=2026-13-40":          "released_to must be a date written YYYY-MM-DD.",
		"released_from=today":             "released_from must be a date written YYYY-MM-DD.",
		"price_ranges=3000000":            "price_ranges must be ranges written from..to, such as ..2999999,4000000..",
		"price_ranges=..":                 "price_ranges must be ranges written from..to, such as ..2999999,4000000..",
		"price_ranges=..2999999,":         "price_ranges must be ranges written from..to, such as ..2999999,4000000..",
		"price_ranges=a..9":               "price_ranges must be a number.",
		"price_ranges=" + twentyOne():     "price_ranges can hold at most 20 ranges.",
		"released_ranges=..soon":          "released_ranges must be a date written YYYY-MM-DD.",
		"released_ranges=2026-01-01":      "released_ranges must be ranges written from..to, such as ..2026-03-31,2026-07-01..",
		"released_ranges=0000-01-01..":    "released_ranges must be a date written YYYY-MM-DD.",
		"price_from=" + strings.Repeat("9", 41): "price_from must be a number.",
	} {
		field := q[:strings.Index(q, "=")]
		req, faults := parse(t, "products", q)
		if faults[field] != want || len(req.ranges) != 0 {
			t.Errorf("%.60s: got %v, want %q", q, faults, want)
		}
	}
}

func twentyOne() string {
	return strings.TrimSuffix(strings.Repeat("1..2,", 21), ",")
}

func TestWordsAreSplitDedupedAndEachBoundOnce(t *testing.T) {
	req, faults := parse(t, "products", "q=%20samba%20%20og%20Samba%20")
	if faults != nil || !reflect.DeepEqual(req.words, []string{"samba", "og", "Samba"}) {
		t.Errorf("got %v %v", req.words, faults)
	}
	req, _ = parse(t, "products", "q=samba+samba")
	if !reflect.DeepEqual(req.words, []string{"samba"}) {
		t.Errorf("an exact repeat was kept: %v", req.words)
	}
	cond, args := whereOf(t, "products", "q=samba%20og")
	for _, want := range []string{"strpos(t.search_text, (SELECT doi_fold($1))) > 0", "(SELECT doi_fold($2))", "t.line_id IN (SELECT l.id FROM lines l WHERE strpos(doi_fold(l.name), (SELECT doi_fold($1))) > 0)",
		"t.category_id IN (SELECT c.id FROM categories c WHERE strpos(doi_fold(c.name), (SELECT doi_fold($2))) > 0)"} {
		if !strings.Contains(cond, want) {
			t.Errorf("%q lacks %q", cond, want)
		}
	}
	if strings.Count(cond, ") AND (") != 1 || !reflect.DeepEqual(args, []any{"samba", "og"}) {
		t.Errorf("every word must hold: %q %v", cond, args)
	}
	// No LIKE: % and _ are read as themselves by strpos.
	if strings.Contains(strings.ToUpper(cond), "LIKE") {
		t.Errorf("a search must not use LIKE: %s", cond)
	}
	// A hostile word is an argument and nothing else.
	cond, args = whereOf(t, "products", "q="+url.QueryEscape("'); DROP TABLE products; --"))
	if strings.Contains(cond, "DROP") || strings.Contains(cond, "'); ") || len(args) != 5 {
		t.Errorf("%q %v", cond, args)
	}
}

func TestQHoldsAtMostAHundredCharactersNotBytes(t *testing.T) {
	for _, ok := range []string{strings.Repeat("a", 100), strings.Repeat("đ", 100), " " + strings.Repeat("a", 100) + " "} {
		if _, faults := parse(t, "products", "q="+url.QueryEscape(ok)); faults != nil {
			t.Errorf("%d characters refused: %v", len([]rune(ok)), faults)
		}
	}
	_, faults := parse(t, "products", "q="+strings.Repeat("a", 101))
	if faults["q"] != "q can have at most 100 characters." {
		t.Errorf("got %v", faults)
	}
	if _, faults = parse(t, "products", "q="+url.QueryEscape(strings.Repeat("đ", 101))); faults["q"] == "" {
		t.Error("101 characters of 2 bytes were accepted")
	}
}

func TestEveryCollectionSearchesOnlyItsNamedFields(t *testing.T) {
	want := map[string][]string{
		"orders":     {"t.number", "t.customer"},
		"reviews":    {"t.name", "t.body", "p.name"},
		"lines":      {"t.name", "t.blurb"},
		"categories": {"t.name", "t.blurb"},
		"delivery":   {"t.name", "t.days", "t.note"},
		"payments":   {"t.name", "t.note"},
		"faqs":       {"t.topic", "t.question", "t.answer"},
		"shop":       {"t.standard_days", "t.dispatch_cutoff", "t.reply_time"},
		"sizes":      {"t.eu", "t.uk", "t.us_m", "t.us_w"},
	}
	for name, fields := range want {
		search := collectionOf(name).Search
		for _, f := range fields {
			if !strings.Contains(search, f) {
				t.Errorf("%s: the search does not look in %s: %s", name, f, search)
			}
		}
		for _, private := range []string{"phone", "email", "address", "note"} {
			if name == "orders" && strings.Contains(search, private) {
				t.Errorf("orders: the search looks in %s", private)
			}
		}
	}
	for _, c := range registry {
		if !strings.Contains(c.Search, "{w}") {
			t.Errorf("%s has no search", c.Name)
		}
		// Nothing but a word and registry text: the sentinel is replaced whole.
		cond, _ := whereOf(t, c.Name, "q=x")
		if strings.Contains(cond, "{") || strings.Contains(cond, "}") {
			t.Errorf("%s: an unreplaced placeholder: %s", c.Name, cond)
		}
	}
}

func TestSortIsOneFieldThenTheSeedOrder(t *testing.T) {
	for _, c := range []struct{ collection, query, order string }{
		{"products", "", "t.seq"},
		{"products", "sort=", "t.seq"},
		{"products", "sort=%20", "t.seq"},
		{"products", "sort=price", "t.price ASC, t.seq ASC"},
		{"products", "sort=-price", "t.price DESC, t.seq ASC"},
		// A NOT NULL column carries no NULLS clause: -released is the order of
		// idx_products_status_released_seq.
		{"products", "sort=-released", "t.released DESC, t.seq ASC"},
		// A shoe with no rating (stored 0) comes last, whichever way it runs.
		{"products", "sort=-rating", "NULLIF(t.rating, 0) DESC NULLS LAST, t.seq ASC"},
		{"products", "sort=rating", "NULLIF(t.rating, 0) ASC NULLS LAST, t.seq ASC"},
		{"products", "sort=name", "doi_fold(left(t.name, 100)) ASC, t.name ASC, t.seq ASC"},
		{"products", "sort=-line", "doi_fold(left(t.line_id, 100)) DESC NULLS LAST, t.line_id DESC NULLS LAST, t.seq ASC"},
		{"products", "sort=-sizes", "ps.sizes DESC, t.seq ASC"},
		{"products", "sort=%20-price%20", "t.price DESC, t.seq ASC"},
		{"reviews", "sort=-date", "t.date DESC, t.seq ASC"},
		{"reviews", "sort=-rating", "t.rating DESC NULLS LAST, t.seq ASC"},
		{"orders", "sort=eta", "t.eta ASC NULLS LAST, t.seq ASC"},
		{"orders", "sort=-qty", "oi.qty DESC NULLS LAST, t.seq ASC"},
		{"orders", "sort=-placed_at", "t.placed_at DESC, t.seq ASC"},
		{"sizes", "sort=cm", "t.cm ASC NULLS LAST, t.seq ASC"},
		{"lines", "sort=-since", "t.since DESC NULLS LAST, t.seq ASC"},
	} {
		if got := orderOf(t, c.collection, c.query); got != c.order {
			t.Errorf("%s?%s: %q, want %q", c.collection, c.query, got, c.order)
		}
	}
}

func TestASortThatIsNotAPublicFieldIsAFault(t *testing.T) {
	generic := "sort must be a field of this list, such as price or -price."
	for _, q := range []string{"sort=zzz", "sort=price,name", "sort=price%20name", "sort=-", "sort=--price", "sort=Price", "sort=price%3B", "sort=t.price"} {
		req, faults := parse(t, "products", q)
		if faults["sort"] != generic || req.sort != nil {
			t.Errorf("%s: %v", q, faults)
		}
	}
	for collection, names := range map[string][]string{"products": {"price_listed", "_source", "_invented"}, "orders": {"phone", "email", "address", "note"}} {
		for _, name := range names {
			for _, sign := range []string{"", "-"} {
				if _, faults := parse(t, collection, "sort="+sign+name); faults["sort"] != name+" cannot be sorted." {
					t.Errorf("%s?sort=%s%s: %v", collection, sign, name, faults)
				}
			}
		}
	}
	// A field of another collection is no field of this one.
	if _, faults := parse(t, "reviews", "sort=price"); faults["sort"] != generic {
		t.Errorf("reviews?sort=price: %v", faults)
	}
}

func TestSizeIsTheEUSizeFilterOfShoesOnly(t *testing.T) {
	cond, args := whereOf(t, "products", "size=38")
	// A size is available when product_sizes holds a pair of it for the shoe.
	for _, want := range []string{"FROM product_sizes sz", "sz.product_id = t.id", "sz.size = ANY($1::text[])", "sz.pairs > 0"} {
		if !strings.Contains(cond, want) {
			t.Errorf("%q lacks %q", cond, want)
		}
	}
	if !reflect.DeepEqual(args, []any{[]string{"38"}}) {
		t.Errorf("%q %v", cond, args)
	}
	// A size is text: 36 2/3 comes percent-encoded; items are trimmed and a repeat dropped.
	_, args = whereOf(t, "products", "size=36%202%2F3,%2040%20,,40")
	if !reflect.DeepEqual(args, []any{[]string{"36 2/3", "40"}}) {
		t.Errorf("%v", args)
	}
	// On orders it is an ordinary field: the first pair's size, exactly.
	cond, args = whereOf(t, "orders", "size=39%201%2F3")
	if cond != "oi.size = $1" || !reflect.DeepEqual(args, []any{"39 1/3"}) {
		t.Errorf("%q %v", cond, args)
	}
}

// SC-10: every condition of one request applies together, in a fixed order, and
// every value is a bound argument.
func TestTheConditionsOfOneRequestApplyTogether(t *testing.T) {
	req, faults := parse(t, "products", "status=live&category=women&line=samba&size=38&price_to=2999999&q=og&sort=price")
	if faults != nil {
		t.Fatal(faults)
	}
	sql, args := listSQL(byName["products"], req)
	want := []any{"samba", "women", "live", "2999999", []string{"38"}, "og", int64(12), int64(0)}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args %#v, want %#v", args, want)
	}
	for _, frag := range []string{
		"t.line_id = $1 AND t.category_id = $2 AND t.status = $3 AND (t.price <= $4::numeric) AND EXISTS (SELECT 1 FROM product_sizes sz WHERE sz.product_id = t.id AND sz.size = ANY($5::text[]) AND sz.pairs > 0)",
		"strpos(t.search_text, (SELECT doi_fold($6)))",
		"ORDER BY t.price ASC, t.seq ASC LIMIT $7 OFFSET $8",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("the statement lacks %q:\n%s", frag, sql)
		}
	}
	// The count and the page read the same conditions: they are one snapshot.
	if strings.Count(sql, "t.status = $3") != 2 || strings.Count(sql, "ORDER BY") != 1 {
		t.Errorf("count and page differ:\n%s", sql)
	}
}

func TestValidID(t *testing.T) {
	for _, ok := range []string{"samba", "doi-24817", "eu-35-1-2", "a.b_c", strings.Repeat("a", 100)} {
		if !validID(ok) {
			t.Errorf("%q refused", ok)
		}
	}
	for _, bad := range []string{"", "Samba", "a b", "a/b", "..", "-a", ".a", "a%2Fb", strings.Repeat("a", 101)} {
		if validID(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}
