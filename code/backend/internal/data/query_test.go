package data

import (
	"net/url"
	"strings"
	"testing"
)

func parse(t *testing.T, collection, query string) (listRequest, map[string]string) {
	t.Helper()
	q, err := url.ParseQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	return parseList(byName[collection], q)
}

func TestPageAndLimitDefaults(t *testing.T) {
	req, faults := parse(t, "products", "")
	if faults != nil || req.page != 1 || req.limit != 12 || len(req.filters) != 0 {
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

func TestUnknownReservedAndAnyOfParametersAreIgnored(t *testing.T) {
	for _, q := range []string{
		"zzz=1", "q=samba", "sort=-price", "price_from=3000000&price_to=3999999",
		"category=women,kids", "size=40", "category=", "Category=women",
	} {
		req, faults := parse(t, "products", q)
		if faults != nil || len(req.filters) != 0 {
			t.Errorf("%q: got %d filters, %v", q, len(req.filters), faults)
		}
	}
}

func TestFiltersAreReadByKind(t *testing.T) {
	req, faults := parse(t, "products", "category=women&status=live&price=3290000&featured=true&rating=4.8&released=2026-11-03&sizes_low=40")
	if faults != nil || len(req.filters) != 7 {
		t.Fatalf("got %d filters, %v", len(req.filters), faults)
	}
	cond, args := where(req.filters)
	// Filters follow the registry's column order, not the query string's.
	for _, want := range []string{"t.category_id = $1", "t.price = $2::bigint", "t.status = $3", "t.featured = $4::boolean", "t.released = $5::date", "$6 = ANY(t.sizes_low)", "t.rating = $7::numeric"} {
		if !strings.Contains(cond, want) {
			t.Errorf("where %q lacks %q", cond, want)
		}
	}
	if len(args) != 7 || args[1] != int64(3290000) || args[3] != true {
		t.Errorf("args %v", args)
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
// 500. It is the caller's typo, so it is a fault at the boundary.
func TestAValuePostgreSQLWouldRefuseIsAFaultNotAnError(t *testing.T) {
	for _, c := range []struct{ collection, query, field string }{
		{"products", "category=a%00b", "category"},
		{"products", "status=%00", "status"},
		{"products", "colour=%ff%fe", "colour"},
		{"products", "sizes_low=40%00", "sizes_low"},
		{"reviews", "product=%c3%28", "product"},
		{"products", "id=%80", "id"},
		{"products", "released=0000-01-01", "released"},
		{"reviews", "date=0000-12-31", "date"},
		{"orders", "eta=0000-01-01", "eta"},
		{"orders", "placed_at=0000-01-01T00:00:00%2B07:00", "placed_at"},
		{"products", "rating=" + strings.Repeat("9", 19), "rating"},
		{"products", "price=" + strings.Repeat("9", 19), "price"},
	} {
		req, faults := parse(t, c.collection, c.query)
		if faults[c.field] == "" || len(req.filters) != 0 {
			t.Errorf("%s?%s: got %v", c.collection, c.query, faults)
		}
	}
	// The year 0001 and ordinary accented text are fine.
	for _, c := range []struct{ collection, query string }{
		{"products", "released=0001-01-01"},
		{"products", "colourway=C%E1%BB%ADa%20L%C3%B2"},
		{"orders", "placed_at=0001-01-01T00:00:00Z"},
	} {
		if req, faults := parse(t, c.collection, c.query); faults != nil || len(req.filters) != 1 {
			t.Errorf("%s?%s: got %d filters, %v", c.collection, c.query, len(req.filters), faults)
		}
	}
}

func TestTheFirstOfARepeatedNameIsUsed(t *testing.T) {
	req, _ := parse(t, "products", "category=women&category=men")
	if len(req.filters) != 1 || req.filters[0].arg != "women" {
		t.Errorf("got %+v", req.filters)
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
