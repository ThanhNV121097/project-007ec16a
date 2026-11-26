package orders

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	ownToken   = "q3J0n8Zk2LmPa7Vx9TrBwE"
	otherToken = "Zr8KpQ2mV7nXa4Lc1TbYdG"
)

// refRow is the row of the access statement: its 14 columns, in its order.
type refRow struct{ ref Ref }

func (r refRow) Scan(dest ...any) error {
	if len(dest) != 14 {
		return fmt.Errorf("the access statement has 14 columns, the scan names %d", len(dest))
	}
	*dest[0].(*string), *dest[1].(*string), *dest[2].(*time.Time) = r.ref.ID, r.ref.Number, r.ref.PlacedAt
	*dest[3].(*string), *dest[4].(*bool), *dest[5].(*int64), *dest[6].(*int64) = r.ref.Status, r.ref.Paid, r.ref.Total, r.ref.DeliveryFee
	*dest[7].(*string), *dest[8].(*string), *dest[9].(*string), *dest[10].(*string) = r.ref.Delivery, r.ref.Payment, r.ref.Carrier, r.ref.Tracking
	*dest[11].(**time.Time), *dest[12].(*string), *dest[13].(*string) = r.ref.ETA, r.ref.Customer, r.ref.City
	return nil
}

// orderDB holds one order and the one token that opens it; every other question
// finds nothing. It counts what reaches it (fakeDB.queries) and opens no
// transaction.
type orderDB struct {
	fakeDB
	order Ref
	token string
	fails error
}

func (d *orderDB) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	d.queries++
	if d.fails != nil {
		return errRow{d.fails}
	}
	id, _ := args[0].(string)
	hash, _ := args[1].([]byte)
	if id == d.order.ID && bytes.Equal(hash, hashToken(d.token)) {
		return refRow{d.order}
	}
	return errRow{pgx.ErrNoRows}
}

func sample(change func(*Ref)) Ref {
	r := Ref{
		ID: "doi-24818", Number: "DOI-24818", Status: "placed", Payment: "vnpay", Delivery: "standard",
		Customer: "Lan Nguyen", City: "Da Nang", Total: 3290000, PlacedAt: time.Date(2026, 11, 25, 3, 0, 0, 0, time.UTC),
	}
	if change != nil {
		change(&r)
	}
	return r
}

// sampleNow is ten minutes after the sample order was placed: inside the 30
// minutes the shop gives to pay, so a refusal in these tests is always the one the
// case names and never "the time to pay has run out".
var sampleNow = time.Date(2026, 11, 25, 3, 10, 0, 0, time.UTC)

// mountPay mounts the order's routes on the clock of sampleNow, unless the test
// brings its own.
func mountPay(db DB, opts Options) http.Handler {
	if opts.Now == nil {
		opts.Now = func() time.Time { return sampleNow }
	}
	return mount(db, opts)
}

func callPay(h http.Handler, id, token, body string) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+id+"/payment", r)
	if token != "" {
		req.Header.Set("X-Order-Token", token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// fakeProvider is the test provider of the seam: it counts the asks.
type fakeProvider struct {
	open  bool
	url   string
	err   error
	asked *int
}

func (p fakeProvider) Available() bool { return p.open }

func (p fakeProvider) PaymentURL(context.Context, Payable, Client) (string, error) {
	*p.asked++
	return p.url, p.err
}

// SC-2 to SC-5 and the precedence: decide over every state, every way to pay and
// every provider.
func TestDecideAnswersEveryCombination(t *testing.T) {
	var asked int
	available := fakeProvider{open: true, url: "https://pay.example.test/x", asked: &asked}
	closed := fakeProvider{open: false, asked: &asked}
	for _, status := range []string{"placed", "confirmed", "packed", "shipped", "delivered", "cancelled"} {
		for _, paid := range []bool{false, true} {
			for _, way := range []string{"vnpay", "cod", "bank", "momo", "barter"} {
				for name, providers := range map[string]map[string]Provider{
					"none": nil, "closed": {way: closed}, "available": {way: available},
				} {
					ref := sample(func(r *Ref) { r.Status, r.Paid, r.Payment = status, paid, way })
					p, err := decide(ref, providers, sampleNow)
					var refusal *Refusal
					case_ := fmt.Sprintf("%s paid=%v %s provider=%s", status, paid, way, name)
					switch {
					case status == "cancelled":
						if !errors.As(err, &refusal) || refusal.Message != msgCancelled {
							t.Errorf("%s: %v", case_, err)
						}
					case paid:
						if !errors.As(err, &refusal) || refusal.Message != msgPaid {
							t.Errorf("%s: %v", case_, err)
						}
					case way != "vnpay":
						if !errors.As(err, &refusal) || refusal.Message != msgNotOnline {
							t.Errorf("%s: %v", case_, err)
						}
					case name == "none":
						if !errors.As(err, &refusal) || refusal.Message != msgNotYet {
							t.Errorf("%s: %v", case_, err)
						}
					case name == "closed":
						if !errors.Is(err, ErrPaymentUnavailable) {
							t.Errorf("%s: %v", case_, err)
						}
					default:
						if err != nil || p == nil {
							t.Errorf("%s: %v", case_, err)
						}
					}
				}
			}
		}
	}
	if asked != 0 {
		t.Errorf("decide asked a provider %d times", asked)
	}
	messages := map[string]bool{msgCancelled: true, msgPaid: true, msgNotOnline: true, msgNotYet: true}
	if len(messages) != 4 || msgNotYet != "Online payment is not available yet." {
		t.Error("the four sentences must differ and the last must be exactly as the contract words it")
	}
}

// SC-1: no token, another order's token, an unknown id and a malformed id are one
// and the same 404; a request that cannot hold a token or an id reaches nothing,
// and GET answers the same bytes for the same.
func TestPayStrangersGetOneBody(t *testing.T) {
	db := &orderDB{order: sample(nil), token: ownToken}
	h := mountPay(db, Options{})
	unformed := []struct{ id, token string }{
		{"doi-24818", ""}, {"doi-24818", "has a space"}, {"doi-24818", strings.Repeat("a", 129)}, {"doi-24818", "a/b"},
		{"DOI-24818", ownToken}, {strings.Repeat("a", 101), ownToken}, {"-x", ownToken},
	}
	var want string
	for _, c := range unformed {
		rec := callPay(h, c.id, c.token, "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("%+v: %d", c, rec.Code)
		}
		if want == "" {
			want = rec.Body.String()
		}
		if rec.Body.String() != want || !strings.Contains(want, `"not_found"`) {
			t.Errorf("%+v: body %q differs from %q", c, rec.Body, want)
		}
		get := httptest.NewRequest(http.MethodGet, "/v1/orders/"+c.id, nil)
		if c.token != "" {
			get.Header.Set("X-Order-Token", c.token)
		}
		got := httptest.NewRecorder()
		h.ServeHTTP(got, get)
		if got.Code != http.StatusNotFound || got.Body.String() != want {
			t.Errorf("%+v: GET answered %d %q, POST %q", c, got.Code, got.Body, want)
		}
	}
	if db.touched() != 0 {
		t.Errorf("a request that cannot hold a token reached the database %d times", db.touched())
	}
	for name, c := range map[string]struct{ id, token string }{
		"another order's token": {"doi-24818", otherToken},
		"an unknown order":      {"doi-99999", ownToken},
	} {
		rec := callPay(h, c.id, c.token, "")
		if rec.Code != http.StatusNotFound || rec.Body.String() != want {
			t.Errorf("%s: %d %q, want %q", name, rec.Code, rec.Body, want)
		}
	}
}

// SC-2 to SC-5 on the route: the refusals, the order they come in, the headers.
func TestPayRefusesByTheOrdersStateAndWritesNothing(t *testing.T) {
	for _, c := range []struct {
		name   string
		change func(*Ref)
		want   string
	}{
		{"a paid order", func(r *Ref) { r.Paid = true }, msgPaid},
		{"a cancelled unpaid order", func(r *Ref) { r.Status = "cancelled" }, msgCancelled},
		{"a cancelled paid order", func(r *Ref) { r.Status, r.Paid = "cancelled", true }, msgCancelled},
		{"a cash order", func(r *Ref) { r.Payment = "cod" }, msgNotOnline},
		{"a bank order", func(r *Ref) { r.Payment = "bank" }, msgNotOnline},
		{"a momo order", func(r *Ref) { r.Payment = "momo" }, msgNotOnline},
		{"a shipped order paid by VNPay with no provider", func(r *Ref) { r.Status = "shipped" }, msgNotYet},
		{"a placed order paid by VNPay with no provider", nil, msgNotYet},
	} {
		db := &orderDB{order: sample(c.change), token: ownToken}
		h := mountPay(db, Options{})
		// A second and a third press answer what the first did.
		for i := 0; i < 3; i++ {
			rec := callPay(h, "doi-24818", ownToken, "")
			e := envelopeOf(t, rec)
			if rec.Code != http.StatusConflict || e.Error != "conflict" || e.Message != c.want || e.Errors != nil {
				t.Errorf("%s (press %d): %d %s", c.name, i+1, rec.Code, rec.Body)
			}
			if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Errorf("%s: headers %v", c.name, rec.Header())
			}
		}
		if db.begins != 0 {
			t.Errorf("%s: the route opened a transaction", c.name)
		}
		if db.queries != 3 {
			t.Errorf("%s: %d statements for 3 presses", c.name, db.queries)
		}
	}
}

// SC-6: a body is a 400 after access and before state; nothing of it is read
// further; a stranger who sends one learns nothing but the 404.
func TestPayTakesNoBody(t *testing.T) {
	h := mountPay(&orderDB{order: sample(func(r *Ref) { r.Paid = true }), token: ownToken}, Options{})
	for _, body := range []string{"{}", "x", `{"a":1}`, "null"} {
		rec := callPay(h, "doi-24818", ownToken, body)
		e := envelopeOf(t, rec)
		if rec.Code != http.StatusBadRequest || e.Error != "bad_request" || e.Message != "This call takes no body." || e.Errors["body"] != "This call takes no request body." {
			t.Errorf("%q: %d %s", body, rec.Code, rec.Body)
		}
	}
	none := callPay(h, "doi-24818", ownToken, "")
	if none.Code != http.StatusConflict {
		t.Errorf("no body on a paid order: %d", none.Code)
	}
	for _, c := range []struct{ id, token string }{{"doi-24818", otherToken}, {"doi-24818", ""}, {"doi-99999", ownToken}} {
		if rec := callPay(h, c.id, c.token, "{}"); rec.Code != http.StatusNotFound {
			t.Errorf("%+v with a body: %d %s", c, rec.Code, rec.Body)
		}
	}
}

type brokenBody struct{}

func (brokenBody) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
func (brokenBody) Close() error             { return nil }

func TestABodyThatCannotBeReadIsABody(t *testing.T) {
	h := mountPay(&orderDB{order: sample(nil), token: ownToken}, Options{})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/doi-24818/payment", nil)
	req.Body = brokenBody{}
	req.Header.Set("X-Order-Token", ownToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

// SC-6: when the database fails, both routes answer a generic 500 that holds no
// SQL, no table, no token and no path.
func TestAFailingDatabaseIsAGeneric500OnBothRoutes(t *testing.T) {
	for name, db := range map[string]DB{
		"the access statement fails": &orderDB{order: sample(nil), token: ownToken, fails: errors.New(`ERROR: relation "orders" does not exist: SELECT o.id FROM orders o`)},
		"no transaction can open":    &fakeDB{},
	} {
		h := mountPay(db, Options{})
		get := httptest.NewRequest(http.MethodGet, "/v1/orders/doi-24818", nil)
		get.Header.Set("X-Order-Token", ownToken)
		got := httptest.NewRecorder()
		h.ServeHTTP(got, get)
		for route, rec := range map[string]*httptest.ResponseRecorder{"POST": callPay(h, "doi-24818", ownToken, ""), "GET": got} {
			if name == "no transaction can open" && route == "POST" {
				continue // POST opens no transaction: this fake finds no order, which is a 404
			}
			e := envelopeOf(t, rec)
			if rec.Code != http.StatusInternalServerError || e.Error != "internal" || e.Message != "Something went wrong. Please try again." {
				t.Errorf("%s, %s: %d %s", name, route, rec.Code, rec.Body)
			}
			for _, leak := range []string{"SELECT", "orders", "relation", ownToken, "/v1", "dial", "refused"} {
				if strings.Contains(rec.Body.String(), leak) {
					t.Errorf("%s, %s: the body holds %q: %s", name, route, leak, rec.Body)
				}
			}
		}
	}
}

// The seam: a provider registered for vnpay changes no line of the route. Its
// address is returned unchanged; a key check that says no is a 503; an error, a
// timeout's worth of nothing or an empty address is a 500; a state refusal comes
// before the provider is asked.
func TestAProviderPlugsIntoTheRouteWithoutChangingIt(t *testing.T) {
	for _, c := range []struct {
		name      string
		provider  fakeProvider
		change    func(*Ref)
		status    int
		code      string
		want      string
		wantAsked int
	}{
		{"an address", fakeProvider{open: true, url: "https://pay.example.test/x"}, nil, 200, "", `{"payment_url":"https://pay.example.test/x"}`, 1},
		{"keys missing", fakeProvider{open: false}, nil, 503, "payment_unavailable", "", 0},
		{"an error", fakeProvider{open: true, err: errors.New("gateway down at 10.0.0.2")}, nil, 500, "internal", "", 1},
		{"an empty address", fakeProvider{open: true}, nil, 500, "internal", "", 1},
		{"a paid order", fakeProvider{open: true, url: "https://pay.example.test/x"}, func(r *Ref) { r.Paid = true }, 409, "conflict", "", 0},
		{"a cancelled order", fakeProvider{open: true, url: "https://pay.example.test/x"}, func(r *Ref) { r.Status = "cancelled" }, 409, "conflict", "", 0},
		{"a way the shop does not complete", fakeProvider{open: true, url: "https://pay.example.test/x"}, func(r *Ref) { r.Payment = "cod" }, 409, "conflict", "", 0},
	} {
		asked := 0
		c.provider.asked = &asked
		h := mountPay(&orderDB{order: sample(c.change), token: ownToken}, Options{Providers: map[string]Provider{"vnpay": c.provider, "cod": c.provider}})
		rec := callPay(h, "doi-24818", ownToken, "")
		if rec.Code != c.status || asked != c.wantAsked {
			t.Errorf("%s: %d %s, asked %d", c.name, rec.Code, rec.Body, asked)
		}
		if c.want != "" && strings.TrimSpace(rec.Body.String()) != c.want {
			t.Errorf("%s: body %s, want %s", c.name, rec.Body, c.want)
		}
		if c.code != "" && envelopeOf(t, rec).Error != c.code {
			t.Errorf("%s: %s", c.name, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "10.0.0.2") {
			t.Errorf("%s: the provider's error reached the client: %s", c.name, rec.Body)
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: no-store is missing", c.name)
		}
	}
}

// Any other method on the path is the catch-all's 404, not a handler of its own.
func TestOnlyPostReachesThePaymentHandler(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, &orderDB{order: sample(nil), token: ownToken}, quiet(), Options{})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req := httptest.NewRequest(method, "/v1/orders/doi-24818/payment", nil)
		req.Header.Set("X-Order-Token", ownToken)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"not_found"`) {
			t.Errorf("%s: %d %s", method, rec.Code, rec.Body)
		}
	}
}

// The route makes no outbound call: the package holds no HTTP client.
func TestThePackageHoldsNoHTTPClient(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, call := range []string{"http.Client", "http.DefaultClient", "http.Get(", "http.Post(", "http.NewRequest", "http.DefaultTransport"} {
			if bytes.Contains(src, []byte(call)) {
				t.Errorf("%s uses %s", f.Name(), call)
			}
		}
	}
}

// The token is hashed in Go and only the hash is a parameter of the statement.
func TestTheTokenIsNeverASQLParameter(t *testing.T) {
	var seen []any
	q := argsRecorder{seen: &seen}
	if _, err := authorize(context.Background(), q, "doi-24818", Access{Token: ownToken}, modePay); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	if len(seen) != 2 || seen[0] != "doi-24818" || !bytes.Equal(seen[1].([]byte), hashToken(ownToken)) || len(seen[1].([]byte)) != 32 {
		t.Errorf("parameters: %v", seen)
	}
	for _, p := range seen {
		if s, ok := p.(string); ok && strings.Contains(s, ownToken) {
			t.Error("the token itself is a parameter")
		}
	}
	if strings.Count(accessSQL, "order_tokens") != 1 || strings.Contains(viewSQL, "order_tokens") {
		t.Error("order_tokens must be named by the access statement alone")
	}
}

type argsRecorder struct{ seen *[]any }

func (a argsRecorder) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	*a.seen = args
	return errRow{pgx.ErrNoRows}
}
