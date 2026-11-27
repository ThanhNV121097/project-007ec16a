package orders

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// POST /v1/order-lookups against PostgreSQL: the real migrations, the seeded
// sample orders and the real counters, through the real route. Skipped without
// TEST_DATABASE_URL, as every database test is; the first run is CI.
//
// The sample order DOI-24817 (id doi-24817) was placed with the phone 0938 410 552.
const (
	lkSample      = "DOI-24817"
	lkSamplePhone = "0938 410 552"
)

func lkShop(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool := seeded(t)
	mux := http.NewServeMux()
	Register(mux, pool, quiet(), Options{})
	return mux, pool
}

// lkCall is one lookup from the network address addr ("" is the test client's own).
func lkCall(h http.Handler, number, phone, addr string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"number": number, "phone": phone})
	req := httptest.NewRequest(http.MethodPost, "/v1/order-lookups", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	if addr != "" {
		req.Header.Set("X-Forwarded-For", addr)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func lkJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("not JSON: %q", rec.Body)
	}
	return v
}

func lkCount(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	return scalar[int](t, pool, "SELECT count(*)::int FROM "+table)
}

func lkHits(t *testing.T, pool *pgxpool.Pool, scope string) int {
	t.Helper()
	return scalar[int](t, pool, "SELECT hits::int FROM rate_limits WHERE scope = $1", scope)
}

func lkReset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	exec(t, pool, "DELETE FROM rate_limits")
}

// lkMiss makes n wrong tries for DOI-24817 from addr and checks each is the 404.
func lkMiss(t *testing.T, h http.Handler, n int, number, addr string) {
	t.Helper()
	for i := range n {
		if rec := lkCall(h, number, "0938 410 00"+strconv.Itoa(i), addr); rec.Code != 404 {
			t.Fatalf("miss %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
}

func lkRetryAfter(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	n, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	if rec.Code != 429 || err != nil || n < 1 || n > 900 {
		t.Fatalf("want a 429 with Retry-After 1..900: %d %q %s", rec.Code, rec.Header().Get("Retry-After"), rec.Body)
	}
	return n
}

// SC-1, SC-5: a right pair answers four keys and a new token that opens the order.
func TestLookupGivesARightPairItsFourKeysAndANewToken(t *testing.T) {
	h, pool := lkShop(t)
	exec(t, pool, "INSERT INTO order_tokens (order_id, token_hash) VALUES ('doi-24817', $1)", hashToken("held-before-the-lookup-0001"))
	before := lkCount(t, pool, "order_tokens")

	rec := lkCall(h, lkSample, lkSamplePhone, "")
	got := lkJSON(t, rec)
	token, _ := got["token"].(string)
	if rec.Code != 200 || len(got) != 4 || got["id"] != "doi-24817" || got["number"] != "DOI-24817" || got["placed_at"] != "2026-11-23T16:05:00+07:00" {
		t.Fatalf("answer: %d %s", rec.Code, rec.Body)
	}
	if len(token) < 22 || !tokenPattern.MatchString(token) {
		t.Errorf("token %q", token)
	}
	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" || rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Set-Cookie") != "" {
		t.Errorf("headers %v", rec.Header())
	}
	for _, private := range []string{"0938", "410 552", "Hoàng", "Example Street", "Hải Châu", "Gift", "Đà Nẵng", `"phone"`, `"email"`, `"address"`, `"note"`, `"customer"`, `"city"`, `"status"`, `"total"`, `"items"`} {
		if strings.Contains(rec.Body.String(), private) {
			t.Errorf("the answer holds %q", private)
		}
	}

	// One new row, holding only the hash: the token text is in no column.
	if lkCount(t, pool, "order_tokens") != before+1 || scalar[int](t, pool, "SELECT count(*)::int FROM order_tokens WHERE order_id = 'doi-24817' AND token_hash = $1", hashToken(token)) != 1 {
		t.Errorf("the lookup did not add exactly its own hashed token")
	}
	if scalar[int](t, pool, "SELECT count(*)::int FROM order_tokens t WHERE t::text LIKE '%' || $1::text || '%'", token) != 0 {
		t.Errorf("the token text is stored")
	}

	// The new token opens the order and no other; the token the order held still works.
	view := fetch(h, "/v1/orders/doi-24817", token)
	if view.Code != 200 || strings.Contains(view.Body.String(), "0938") || strings.Contains(view.Body.String(), `"phone"`) {
		t.Errorf("the order with the new token: %d %s", view.Code, view.Body)
	}
	if rec := fetch(h, "/v1/orders/doi-24817", "held-before-the-lookup-0001"); rec.Code != 200 {
		t.Errorf("the earlier token: %d", rec.Code)
	}
	elsewhere, missing := fetch(h, "/v1/orders/doi-24816", token), fetch(h, "/v1/orders/doi-99999", token)
	if elsewhere.Code != 404 || elsewhere.Body.String() != missing.Body.String() {
		t.Errorf("the token on another order: %d %s, on none: %d %s", elsewhere.Code, elsewhere.Body, missing.Code, missing.Body)
	}

	// Another right pair mints another token, and both work.
	second, _ := lkJSON(t, lkCall(h, lkSample, lkSamplePhone, ""))["token"].(string)
	if second == "" || second == token || fetch(h, "/v1/orders/doi-24817", second).Code != 200 || fetch(h, "/v1/orders/doi-24817", token).Code != 200 {
		t.Errorf("the second lookup: %q", second)
	}
}

// SC-2: the number and the phone are read as their digits.
func TestLookupReadsTheDigitsOfTheNumberAndThePhone(t *testing.T) {
	h, _ := lkShop(t)
	for _, number := range []string{"DOI-24817", "doi-24817", "doi 24817", "  DOI - 24817 ", "24817"} {
		for _, phone := range []string{"0938 410 552", "0938410552", "+84938410552", "+84 938 410 552", "84938410552", "(0938) 410-552"} {
			if rec := lkCall(h, number, phone, ""); rec.Code != 200 || lkJSON(t, rec)["id"] != "doi-24817" {
				t.Errorf("%q with %q: %d %s", number, phone, rec.Code, rec.Body)
			}
		}
	}
}

// SC-3, SC-4: every miss is one answer, byte for byte, and makes no token.
func TestLookupEveryMissIsTheSameAnswer(t *testing.T) {
	h, pool := lkShop(t)
	tokens := lkCount(t, pool, "order_tokens")
	var first *httptest.ResponseRecorder
	for _, c := range []struct{ name, number, phone string }{
		{"a wrong phone", lkSample, "0938 410 553"},
		{"an unknown number", "DOI-99999", lkSamplePhone},
		{"an empty phone", lkSample, ""},
		{"a phone of spaces", lkSample, "     "},
		{"a number with no digits", "DOI-", lkSamplePhone},
		{"an empty number", "", lkSamplePhone},
		{"a number of 19 digits", "1234567890123456789", lkSamplePhone},
		{"the last three digits", lkSample, "552"},
		{"the last six digits", lkSample, "410552"},
		{"the phone without its 0", lkSample, "938410552"},
		{"a longer phone holding it", lkSample, "0938 410 552 1"},
		{"an 84 that is not at the start", lkSample, "938 410 552 84"},
	} {
		lkReset(t, pool)
		rec := lkCall(h, c.number, c.phone, "")
		if first == nil {
			first = rec
		}
		e := lkJSON(t, rec)
		if rec.Code != 404 || e["error"] != "not_found" || e["errors"] != nil || e["message"] == "" {
			t.Errorf("%s: %d %s", c.name, rec.Code, rec.Body)
		}
		if rec.Body.String() != first.Body.String() || rec.Header().Get("Content-Type") != first.Header().Get("Content-Type") ||
			rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Set-Cookie") != "" {
			t.Errorf("%s is told apart: %v %q", c.name, rec.Header(), rec.Body)
		}
	}
	if lkCount(t, pool, "order_tokens") != tokens {
		t.Errorf("a miss made a token")
	}
}

// SC-6, SC-9 (first half): five misses, then a lock that refuses even the right pair.
func TestLookupTheSixthTryIsRefusedWhateverThePair(t *testing.T) {
	h, pool := lkShop(t)
	tokens := lkCount(t, pool, "order_tokens")
	lkMiss(t, h, 5, lkSample, "")

	var last int
	for i, phone := range []string{lkSamplePhone, "0938 410 000", lkSamplePhone} {
		rec := lkCall(h, lkSample, phone, "")
		e := lkJSON(t, rec)
		wait := lkRetryAfter(t, rec)
		if e["error"] != "rate_limited" || e["message"] == "" || e["errors"] != nil || e["token"] != nil || (i > 0 && wait > last) {
			t.Errorf("try %d: %d %s", i+6, rec.Code, rec.Body)
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("Cache-Control %q", rec.Header().Get("Cache-Control"))
		}
		last = wait
	}
	if lkCount(t, pool, "order_tokens") != tokens {
		t.Errorf("a refused try made a token")
	}
	// A refused try counts nothing: the count stays five.
	if lkHits(t, pool, "order_lookup_number") != 5 || lkHits(t, pool, "order_lookup_address") != 5 {
		t.Errorf("hits %d and %d", lkHits(t, pool, "order_lookup_number"), lkHits(t, pool, "order_lookup_address"))
	}
	// SC-11: the lock is the lookup's only; the order opens with a token it holds.
	exec(t, pool, "INSERT INTO order_tokens (order_id, token_hash) VALUES ('doi-24817', $1)", hashToken("held-while-locked-0000001"))
	if rec := fetch(h, "/v1/orders/doi-24817", "held-while-locked-0000001"); rec.Code != 200 {
		t.Errorf("a locked number hides its order: %d", rec.Code)
	}
}

// SC-7: the address lock, the number lock from another address, an unknown number's lock.
func TestLookupLocksByAddressByNumberAndForANumberThatDoesNotExist(t *testing.T) {
	h, pool := lkShop(t)
	const a, b = "203.0.113.5", "203.0.113.6"

	// (a) five misses from one address, across five numbers: its sixth try is refused.
	for i := range 5 {
		if rec := lkCall(h, "DOI-9999"+strconv.Itoa(i), lkSamplePhone, a); rec.Code != 404 {
			t.Fatalf("miss %d: %d", i+1, rec.Code)
		}
	}
	lkRetryAfter(t, lkCall(h, lkSample, lkSamplePhone, a))

	// (b) five misses for a number from A: a right pair from B is refused.
	lkReset(t, pool)
	lkMiss(t, h, 5, lkSample, a)
	byNumber := lkCall(h, lkSample, lkSamplePhone, b)
	lkRetryAfter(t, byNumber)

	// (c) the same for a number that does not exist: the same answer in shape.
	lkReset(t, pool)
	lkMiss(t, h, 5, "DOI-99999", a)
	unknown := lkCall(h, "DOI-99999", lkSamplePhone, b)
	lkRetryAfter(t, unknown)
	if unknown.Code != byNumber.Code || unknown.Body.String() != byNumber.Body.String() ||
		unknown.Header().Get("Content-Type") != byNumber.Header().Get("Content-Type") || unknown.Header().Get("Cache-Control") != byNumber.Header().Get("Cache-Control") {
		t.Errorf("a locked number tells whether it exists: %q and %q", unknown.Body, byNumber.Body)
	}

	// The lock of one number does not stop another number's right pair from another address.
	lkReset(t, pool)
	lkMiss(t, h, 5, lkSample, a)
	other := scalar[string](t, pool, "SELECT phone FROM orders WHERE id = 'doi-24805'")
	if rec := lkCall(h, "DOI-24805", other, b); rec.Code != 200 || lkJSON(t, rec)["id"] != "doi-24805" {
		t.Errorf("another number from another address: %d %s", rec.Code, rec.Body)
	}
}

// SC-8: a right pair neither counts nor clears.
func TestLookupARightPairNeitherCountsNorClears(t *testing.T) {
	h, pool := lkShop(t)
	lkMiss(t, h, 3, lkSample, "")
	if rec := lkCall(h, lkSample, lkSamplePhone, ""); rec.Code != 200 {
		t.Fatalf("the right pair after three misses: %d", rec.Code)
	}
	lkMiss(t, h, 2, lkSample, "")
	lkRetryAfter(t, lkCall(h, lkSample, lkSamplePhone, ""))

	lkReset(t, pool)
	lkMiss(t, h, 4, lkSample, "")
	for i := range 3 {
		if rec := lkCall(h, lkSample, lkSamplePhone, ""); rec.Code != 200 {
			t.Fatalf("right pair %d after four misses: %d", i+1, rec.Code)
		}
	}
	if lkHits(t, pool, "order_lookup_number") != 4 {
		t.Errorf("a success moved the count to %d", lkHits(t, pool, "order_lookup_number"))
	}
}

// SC-9: the lock counts down, never restarts, and ends 15 minutes after the fifth miss.
func TestLookupTheLockCountsDownAndEndsAfterFifteenMinutes(t *testing.T) {
	h, pool := lkShop(t)
	lkMiss(t, h, 5, lkSample, "")
	first := lkRetryAfter(t, lkCall(h, lkSample, lkSamplePhone, ""))
	if first < 890 {
		t.Errorf("the first wait is %d seconds, want about 900", first)
	}
	for range 3 {
		locked := scalar[string](t, pool, "SELECT locked_until::text FROM rate_limits WHERE scope = 'order_lookup_number'")
		lkRetryAfter(t, lkCall(h, lkSample, "0938 410 000", ""))
		if after := scalar[string](t, pool, "SELECT locked_until::text FROM rate_limits WHERE scope = 'order_lookup_number'"); after != locked {
			t.Errorf("a refused try moved the lock from %s to %s", locked, after)
		}
	}

	// The database's clock is the only one, so time is moved in the table.
	back := func(minutes int) {
		exec(t, pool, "UPDATE rate_limits SET locked_until = locked_until - make_interval(mins => $1::int), window_started_at = window_started_at - make_interval(mins => $1::int)", minutes)
	}
	back(14)
	if wait := lkRetryAfter(t, lkCall(h, lkSample, lkSamplePhone, "")); wait > 60 {
		t.Errorf("after 14 minutes the wait is %d seconds", wait)
	}
	back(2)
	if rec := lkCall(h, lkSample, "0938 410 000", ""); rec.Code != 404 || lkHits(t, pool, "order_lookup_number") != 1 || lkHits(t, pool, "order_lookup_address") != 1 {
		t.Errorf("a miss after 16 minutes: %d, hits %d and %d", rec.Code, lkHits(t, pool, "order_lookup_number"), lkHits(t, pool, "order_lookup_address"))
	}
	if rec := lkCall(h, lkSample, lkSamplePhone, ""); rec.Code != 200 {
		t.Errorf("a right pair after the lock: %d", rec.Code)
	}
}

// SC-10: ten wrong tries at one moment let exactly five failures through.
func TestLookupTenTriesAtOnceLetFiveFailuresThrough(t *testing.T) {
	h, pool := lkShop(t)
	codes := make([]int, 10)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = lkCall(h, lkSample, "0938 410 000", "203.0.113."+strconv.Itoa(10+i)).Code
		}()
	}
	wg.Wait()
	var missed, refused int
	for _, c := range codes {
		switch c {
		case 404:
			missed++
		case 429:
			refused++
		}
	}
	if missed != 5 || refused != 5 || lkHits(t, pool, "order_lookup_number") != 5 {
		t.Errorf("%d answered 404 and %d answered 429 (%v), the count is %d", missed, refused, codes, lkHits(t, pool, "order_lookup_number"))
	}
}

// SC-12: a body the route refuses is not a try; an empty phone is a miss.
func TestLookupARefusedBodyCountsNothingAndAnEmptyPhoneCountsOne(t *testing.T) {
	h, pool := lkShop(t)
	tokens := lkCount(t, pool, "order_tokens")
	for _, c := range []struct{ body, contentType string }{
		{`{"number":"DOI-24817"}`, "application/json"},
		{`{"number":"DOI-24817","phone":"1","email":"a@b.co"}`, "application/json"},
		{`{"number":"DOI-24817","phone":5}`, "application/json"},
		{`{"number":"DOI-24817","phone":"0938 410 552"}`, "text/plain"},
		{`nope`, "application/json"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/order-lookups", strings.NewReader(c.body))
		req.Header.Set("Content-Type", c.contentType)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 400 && rec.Code != 415 {
			t.Errorf("%s: %d", c.body, rec.Code)
		}
	}
	if lkCount(t, pool, "rate_limits") != 0 || lkCount(t, pool, "order_tokens") != tokens {
		t.Errorf("a refused body was counted")
	}
	if rec := lkCall(h, lkSample, "", ""); rec.Code != 404 || lkHits(t, pool, "order_lookup_number") != 1 {
		t.Errorf("an empty phone: %d, hits %d", rec.Code, lkHits(t, pool, "order_lookup_number"))
	}
}
