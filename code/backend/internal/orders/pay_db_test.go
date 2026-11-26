package orders

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests run the real migrations, the seeded sample orders and the real
// access statement. They read TEST_DATABASE_URL, as CI's backend job sets it, and
// skip without it.

// tokenFor gives a sample order a token the test knows.
func tokenFor(t *testing.T, pool *pgxpool.Pool, orderID, token string) {
	t.Helper()
	exec(t, pool, "INSERT INTO order_tokens (order_id, token_hash) VALUES ($1, $2)", orderID, hashToken(token))
}

// sampleOf is the first seeded order paid with that way to pay.
func sampleOf(t *testing.T, pool *pgxpool.Pool, payment string) string {
	t.Helper()
	return scalar[string](t, pool, "SELECT id FROM orders WHERE is_sample AND payment_method_id = $1 ORDER BY seq LIMIT 1", payment)
}

func updatedAt(t *testing.T, pool *pgxpool.Pool, id string) time.Time {
	t.Helper()
	return scalar[time.Time](t, pool, "SELECT updated_at FROM orders WHERE id = $1", id)
}

func getWith(h http.Handler, id, token string) *httptest.ResponseRecorder {
	return fetch(h, "/v1/orders/"+id, token)
}

// SC-1: no token, another order's token, an order that does not exist, a seeded
// order no token opens and a malformed id answer one body, on POST and on GET.
func TestPayStrangersGetTheBodyOfAMissingOrderOnBothRoutes(t *testing.T) {
	h, pool := shop(t, Options{})
	a, b := sampleOf(t, pool, "vnpay"), sampleOf(t, pool, "cod")
	tokenFor(t, pool, a, ownToken)
	tokenFor(t, pool, b, otherToken)
	untouched := scalar[string](t, pool, "SELECT id FROM orders WHERE is_sample AND id NOT IN ($1, $2) ORDER BY seq LIMIT 1", a, b)
	before := updatedAt(t, pool, a)

	cases := map[string]struct{ id, token string }{
		"no token":                      {a, ""},
		"another order's token":         {a, otherToken},
		"an unknown order":              {"doi-99999", ownToken},
		"a seeded order no token opens": {untouched, ownToken},
		"an upper-case id":              {strings.ToUpper(a), ownToken},
		"an id of 101 characters":       {strings.Repeat("a", 101), ownToken},
		"an id that starts with a dash": {"-x", ownToken},
		"a token with a space":          {a, "has a space"},
		"a token of 129 characters":     {a, strings.Repeat("a", 129)},
	}
	want := getWith(h, a, "").Body.String()
	if !strings.Contains(want, `"not_found"`) {
		t.Fatalf("GET without a token: %s", want)
	}
	for name, c := range cases {
		post, get := callPay(h, c.id, c.token, ""), getWith(h, c.id, c.token)
		if post.Code != http.StatusNotFound || post.Body.String() != want {
			t.Errorf("%s: POST %d %q, want the 404 %q", name, post.Code, post.Body, want)
		}
		if get.Code != http.StatusNotFound || get.Body.String() != want {
			t.Errorf("%s: GET %d %q, want the 404 %q", name, get.Code, get.Body, want)
		}
	}
	if !updatedAt(t, pool, a).Equal(before) {
		t.Error("a stranger changed the order")
	}
}

// SC-2, SC-3, SC-4: paid, cancelled (paid or not) and a way to pay the shop does
// not complete are three different refusals, and the order is left as it was.
func TestPayRefusesByStateAgainstTheSeededOrders(t *testing.T) {
	h, pool := shop(t, Options{})
	for i, c := range []struct {
		name, way, status string
		paid              bool
		want              string
	}{
		{"paid", "vnpay", "packed", true, msgPaid},
		{"cancelled and unpaid", "vnpay", "cancelled", false, msgCancelled},
		{"cancelled and paid", "vnpay", "cancelled", true, msgCancelled},
		{"cash", "cod", "placed", false, msgNotOnline},
		{"bank", "bank", "confirmed", false, msgNotOnline},
		{"momo", "momo", "shipped", false, msgNotOnline},
		{"a delivered order not paid", "vnpay", "delivered", false, msgNotYet},
	} {
		id, token := sampleOf(t, pool, c.way), fmt.Sprintf("sample-token-%02d-xxxxxxxxxxxx", i)
		exec(t, pool, "UPDATE orders SET status = $2, paid = $3 WHERE id = $1", id, c.status, c.paid)
		tokenFor(t, pool, id, token)
		before := updatedAt(t, pool, id)
		rec := callPay(h, id, token, "")
		e := envelopeOf(t, rec)
		if rec.Code != http.StatusConflict || e.Error != "conflict" || e.Message != c.want {
			t.Errorf("%s: %d %s, want 409 %q", c.name, rec.Code, rec.Body, c.want)
		}
		if !updatedAt(t, pool, id).Equal(before) || scalar[bool](t, pool, "SELECT paid FROM orders WHERE id = $1", id) != c.paid {
			t.Errorf("%s: the order changed", c.name)
		}
	}
}

// SC-5: an order just placed through POST /v1/orders is refused "not available
// yet" on every press; nothing is written.
func TestPayOnAnOrderJustPlacedIsNotAvailableYet(t *testing.T) {
	h, pool := shop(t, Options{})
	addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 3})
	a := placedOf(t, post(h, orderOf(lanPhone, item("t-a", "38", 1)), "application/json", "tc-key-0001"), 201)
	id := a.Order["id"].(string)
	before := updatedAt(t, pool, id)
	for i := 0; i < 3; i++ {
		rec := callPay(h, id, a.Token, "")
		e := envelopeOf(t, rec)
		if rec.Code != http.StatusConflict || e.Error != "conflict" || e.Message != "Online payment is not available yet." {
			t.Errorf("press %d: %d %s", i+1, rec.Code, rec.Body)
		}
		if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("press %d: headers %v", i+1, rec.Header())
		}
	}
	if scalar[bool](t, pool, "SELECT paid FROM orders WHERE id = $1", id) || !updatedAt(t, pool, id).Equal(before) ||
		scalar[string](t, pool, "SELECT status FROM orders WHERE id = $1", id) != "placed" {
		t.Error("a press changed the order")
	}
	if scalar[int](t, pool, "SELECT count(*) FROM order_tokens WHERE order_id = $1", id) != 1 {
		t.Error("a press changed the order's tokens")
	}
	if scalar[bool](t, pool, "SELECT to_regclass('payment_attempts') IS NOT NULL") &&
		scalar[int](t, pool, "SELECT count(*) FROM payment_attempts WHERE order_id = $1", id) != 0 {
		t.Error("a payment attempt was written")
	}
	// The order is still read with the same token afterwards.
	if rec := getWith(h, id, a.Token); rec.Code != http.StatusOK {
		t.Errorf("GET after the presses: %d %s", rec.Code, rec.Body)
	}
}

// SC-6: a body is a 400 for the order's own token and a 404 for a stranger.
func TestPayWithABodyAgainstTheSeededOrders(t *testing.T) {
	h, pool := shop(t, Options{})
	id := sampleOf(t, pool, "vnpay")
	tokenFor(t, pool, id, ownToken)
	rec := callPay(h, id, ownToken, "{}")
	if e := envelopeOf(t, rec); rec.Code != http.StatusBadRequest || e.Error != "bad_request" || e.Errors["body"] == "" {
		t.Errorf("own token with a body: %d %s", rec.Code, rec.Body)
	}
	if rec := callPay(h, id, otherToken, "{}"); rec.Code != http.StatusNotFound {
		t.Errorf("a stranger with a body: %d %s", rec.Code, rec.Body)
	}
}

// SC-29: a provider registered for vnpay makes the same route answer 200 with its
// address, and a paid order still says so without asking it.
func TestPayAnswersTheProvidersAddressWhenOneIsRegistered(t *testing.T) {
	asked := 0
	h, pool := shop(t, Options{Providers: map[string]Provider{"vnpay": fakeProvider{open: true, url: "https://pay.example.test/x", asked: &asked}}})
	addShoe(t, pool, "t-a", "Test A", 3090000, map[string]int{"38": 3})
	a := placedOf(t, post(h, orderOf(lanPhone, item("t-a", "38", 1)), "application/json", "tc-key-0001"), 201)
	id := a.Order["id"].(string)
	asked = 0
	rec := callPay(h, id, a.Token, "")
	if strings.TrimSpace(rec.Body.String()) != `{"payment_url":"https://pay.example.test/x"}` || rec.Code != http.StatusOK || asked != 1 {
		t.Errorf("got %d %s, asked %d", rec.Code, rec.Body, asked)
	}
	exec(t, pool, "UPDATE orders SET paid = true WHERE id = $1", id)
	if rec := callPay(h, id, a.Token, ""); rec.Code != http.StatusConflict || asked != 1 {
		t.Errorf("a paid order: %d %s, asked %d", rec.Code, rec.Body, asked)
	}
}
