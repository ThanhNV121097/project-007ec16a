package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// errRow is a row that fails to scan.
type errRow struct{ err error }

func (r errRow) Scan(...any) error { return r.err }

// emptyRows is an answer with no rows.
type emptyRows struct{}

func (emptyRows) Close()                                       {}
func (emptyRows) Err() error                                   { return nil }
func (emptyRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (emptyRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (emptyRows) Next() bool                                   { return false }
func (emptyRows) Scan(...any) error                            { return nil }
func (emptyRows) Values() ([]any, error)                       { return nil, nil }
func (emptyRows) RawValues() [][]byte                          { return nil }
func (emptyRows) Conn() *pgx.Conn                              { return nil }
func (emptyRows) TypeMap() *pgtype.Map                         { return nil }

// fakeDB counts what reaches the database. Its transactions cannot be opened,
// so a request that gets as far as placing fails like an unreachable database.
type fakeDB struct {
	begins, queries int
}

func (f *fakeDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	f.begins++
	return nil, errors.New("dial tcp 10.0.0.1:5432: connection refused")
}

func (f *fakeDB) QueryRow(context.Context, string, ...any) pgx.Row {
	f.queries++
	return errRow{pgx.ErrNoRows}
}

func (f *fakeDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	f.queries++
	return emptyRows{}, nil
}

func (f *fakeDB) touched() int { return f.begins + f.queries }

func mount(db DB, opts Options) http.Handler {
	mux := http.NewServeMux()
	Register(mux, db, quiet(), opts)
	return mux
}

const goodBody = `{"items":[{"product":"samba-og-w-core-black","size":"38","qty":2}],"delivery":"standard","payment":"vnpay",` +
	`"customer":"Lan Nguyen","phone":"0912 345 678","address":"12 Example Street, Ward 1","city":"Da Nang","note":""}`

func post(h http.Handler, body, contentType, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type envelope struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors"`
}

func envelopeOf(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var e envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("not an envelope: %s", rec.Body)
	}
	return e
}

func goodRequest() orderRequest {
	return orderRequest{
		Items:    []itemRequest{{Product: "samba", Size: "38", Qty: 2}},
		Delivery: "standard", Payment: "vnpay", Customer: "Lan Nguyen", Phone: "0912 345 678",
		Address: "12 Example Street", City: "Da Nang",
	}
}

// SC-9, SC-10, SC-12, SC-16: the shop applies the page's rules to a request that
// skips the page, and names the field and the limit.
func TestEveryFieldRuleNamesItsField(t *testing.T) {
	many := func(n int) string { return strings.Repeat("a", n) }
	for _, c := range []struct {
		name   string
		change func(*orderRequest)
		field  string
		must   string
	}{
		{"a good request", func(*orderRequest) {}, "", ""},
		{"an empty address", func(r *orderRequest) { r.Address = "" }, "address", "Enter the street address."},
		{"an address of 200 characters", func(r *orderRequest) { r.Address = many(200) }, "", ""},
		{"an address of 201", func(r *orderRequest) { r.Address = many(201) }, "address", "200"},
		{"an empty city", func(r *orderRequest) { r.City = "" }, "city", "Enter the city"},
		{"a city of 101", func(r *orderRequest) { r.City = many(101) }, "city", "100"},
		{"an empty name", func(r *orderRequest) { r.Customer = "" }, "customer", "Enter your name."},
		{"a name of 100", func(r *orderRequest) { r.Customer = many(100) }, "", ""},
		{"a name of 101", func(r *orderRequest) { r.Customer = many(101) }, "customer", "100"},
		{"a name of 100 accented letters", func(r *orderRequest) { r.Customer = strings.Repeat("ạ", 100) }, "", ""},
		{"a note of 500", func(r *orderRequest) { r.Note = many(500) }, "", ""},
		{"a note of 501", func(r *orderRequest) { r.Note = many(501) }, "note", "500"},
		{"a phone of 8 digits", func(r *orderRequest) { r.Phone = "09123456" }, "phone", "Check the phone number"},
		{"a phone of 12 and spaces", func(r *orderRequest) { r.Phone = "0912 34" }, "phone", "Check the phone number"},
		{"a phone of 9 digits", func(r *orderRequest) { r.Phone = "091234567" }, "", ""},
		{"a phone with a plus, spaces and 11 digits", func(r *orderRequest) { r.Phone = "+84 912 345 678" }, "", ""},
		{"a phone of dots", func(r *orderRequest) { r.Phone = "0912.345.678" }, "", ""},
		{"a phone of 21 characters", func(r *orderRequest) { r.Phone = "0912 345 678 901 234 5" }, "phone", "20"},
		{"express delivery", func(r *orderRequest) { r.Delivery = "express" }, "delivery", "standard"},
		{"no delivery", func(r *orderRequest) { r.Delivery = "" }, "delivery", "standard"},
		{"cash on delivery", func(r *orderRequest) { r.Payment = "cod" }, "payment", "not offered"},
		{"bank transfer", func(r *orderRequest) { r.Payment = "bank" }, "payment", "not offered"},
		{"momo", func(r *orderRequest) { r.Payment = "momo" }, "payment", "not offered"},
		{"an unknown way to pay", func(r *orderRequest) { r.Payment = "barter" }, "payment", "not offered"},
		{"no pairs", func(r *orderRequest) { r.Items = nil }, "items", "Add at least one pair."},
		{"51 pairs", func(r *orderRequest) {
			r.Items = nil
			for i := 0; i < 51; i++ {
				r.Items = append(r.Items, itemRequest{Product: fmt.Sprintf("s%d", i), Size: "38", Qty: 1})
			}
		}, "items", "at most 50"},
		{"a quantity of 0", func(r *orderRequest) { r.Items[0].Qty = 0 }, "items.0", "from 1 to 1,000"},
		{"a quantity of 1001", func(r *orderRequest) { r.Items[0].Qty = 1001 }, "items.0", "from 1 to 1,000"},
		{"a pair with no size", func(r *orderRequest) { r.Items[0].Size = "" }, "items.0", "Choose a shoe and a size."},
		{"a pair listed twice", func(r *orderRequest) { r.Items = append(r.Items, r.Items[0]) }, "items.1", "listed twice"},
	} {
		req := goodRequest()
		c.change(&req)
		req.clean()
		got := req.validate()
		if c.field == "" {
			if len(got) != 0 {
				t.Errorf("%s: refused: %v", c.name, got)
			}
			continue
		}
		if !strings.Contains(got[c.field], c.must) {
			t.Errorf("%s: errors.%s = %q, want it to hold %q (all: %v)", c.name, c.field, got[c.field], c.must, got)
		}
	}
}

func TestNameAndAddressOfOnlySpacesAreEmpty(t *testing.T) {
	req := goodRequest()
	req.Customer, req.Address, req.City = "   ", "\t", "  "
	req.clean()
	got := req.validate()
	if got["customer"] == "" || got["address"] == "" || got["city"] == "" {
		t.Errorf("got %v", got)
	}
}

func TestEveryFaultIsReportedAtOnce(t *testing.T) {
	req := goodRequest()
	req.Address, req.Phone, req.Delivery = "", "12", "express"
	req.clean()
	if got := req.validate(); len(got) != 3 {
		t.Errorf("got %v", got)
	}
}

// ORD-003 AC-6, AC-9: a unit price, a delivery fee or a total in the request is
// accepted and ignored, and any other field is refused.
func TestTheBrowsersNumbersAreIgnoredAndAnyOtherFieldIsRefused(t *testing.T) {
	db := &fakeDB{}
	h := mount(db, Options{})
	withExtras := strings.Replace(goodBody, `"qty":2}`, `"qty":2,"unit_price":1,"price":1}`, 1)
	withExtras = strings.Replace(withExtras, `"note":""}`, `"note":"","delivery_fee":50000,"total":1}`, 1)
	rec := post(h, withExtras, "application/json", "tc-key-0001")
	if rec.Code != http.StatusInternalServerError {
		// It reached placing, which cannot open a transaction here: the extras were not the fault.
		t.Errorf("the browser's numbers were not tolerated: %d %s", rec.Code, rec.Body)
	}
	for _, extra := range []string{`"email":"a@b.co"`, `"discount":10`, `"account_id":"x"`, `"paid":true`} {
		body := strings.Replace(goodBody, `"note":""}`, `"note":"",`+extra+`}`, 1)
		if rec := post(h, body, "application/json", "tc-key-0001"); rec.Code != http.StatusBadRequest || envelopeOf(t, rec).Error != "bad_request" {
			t.Errorf("%s: %d %s", extra, rec.Code, rec.Body)
		}
	}
	body := strings.Replace(goodBody, `"qty":2}`, `"qty":2,"colour":"red"}`, 1)
	if rec := post(h, body, "application/json", "tc-key-0001"); rec.Code != http.StatusBadRequest {
		t.Errorf("an unknown field of a pair was accepted: %d", rec.Code)
	}
}

func TestTheBodyIsCheckedBeforeAnythingElse(t *testing.T) {
	db := &fakeDB{}
	h := mount(db, Options{})
	for _, c := range []struct {
		name, body, contentType, key string
		status                      int
		code                        string
	}{
		{"not JSON by its type", goodBody, "text/plain", "tc-key-0001", 415, "unsupported_media_type"},
		{"no type", goodBody, "", "tc-key-0001", 415, "unsupported_media_type"},
		{"JSON with a charset", goodBody, "application/json; charset=utf-8", "", 400, "bad_request"},
		{"too large", `{"note":"` + strings.Repeat("a", 70<<10) + `"}`, "application/json", "tc-key-0001", 413, "payload_too_large"},
		{"not JSON", "not json", "application/json", "tc-key-0001", 400, "bad_request"},
		{"an empty body", "", "application/json", "tc-key-0001", 400, "bad_request"},
		{"an array", "[]", "application/json", "tc-key-0001", 400, "bad_request"},
		{"data after the object", goodBody + `{}`, "application/json", "tc-key-0001", 400, "bad_request"},
		{"a quantity that is text", strings.Replace(goodBody, `"qty":2`, `"qty":"2"`, 1), "application/json", "tc-key-0001", 400, "bad_request"},
		{"a quantity with a fraction", strings.Replace(goodBody, `"qty":2`, `"qty":2.5`, 1), "application/json", "tc-key-0001", 400, "bad_request"},
		{"a NUL in the name", strings.Replace(goodBody, `Lan Nguyen`, `Lan\u0000`, 1), "application/json", "tc-key-0001", 400, "bad_request"},
		{"no Idempotency-Key", goodBody, "application/json", "", 400, "bad_request"},
		{"a short Idempotency-Key", goodBody, "application/json", "short", 400, "bad_request"},
		{"an Idempotency-Key with a space", goodBody, "application/json", "has a space in it", 400, "bad_request"},
		{"an Idempotency-Key of 101", goodBody, "application/json", strings.Repeat("k", 101), 400, "bad_request"},
	} {
		rec := post(h, c.body, c.contentType, c.key)
		if rec.Code != c.status || envelopeOf(t, rec).Error != c.code {
			t.Errorf("%s: %d %s, want %d %s", c.name, rec.Code, rec.Body, c.status, c.code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("%s: content type %q", c.name, ct)
		}
	}
	if rec := post(h, goodBody, "application/json", ""); envelopeOf(t, rec).Errors["Idempotency-Key"] == "" {
		t.Errorf("the header is not named: %s", rec.Body)
	}
	if db.touched() != 0 {
		t.Errorf("a request refused at the boundary reached the database %d times", db.touched())
	}
}

// A field fault is a 422 that names the field, and nothing is placed: the only
// thing such a request may read is the sizes of the shoes it names.
func TestAFieldFaultIsA422AndPlacesNothing(t *testing.T) {
	db := &fakeDB{}
	h := mount(db, Options{})
	body := strings.Replace(goodBody, `"address":"12 Example Street, Ward 1"`, `"address":""`, 1)
	rec := post(h, body, "application/json", "tc-key-0001")
	e := envelopeOf(t, rec)
	if rec.Code != http.StatusUnprocessableEntity || e.Error != "invalid" || e.Errors["address"] == "" || e.Message != "Check the details of your order." {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
	if db.begins != 0 {
		t.Errorf("a transaction was opened for an order with a field fault")
	}
}

// SC-21: with the database unreachable, placing answers 500 with a generic body.
func TestAnUnreachableDatabaseIsAGeneric500(t *testing.T) {
	rec := post(mount(&fakeDB{}, Options{}), goodBody, "application/json", "tc-key-0001")
	if rec.Code != http.StatusInternalServerError || envelopeOf(t, rec).Error != "internal" {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	for _, leak := range []string{"dial", "10.0.0.1", "5432", "refused", "SELECT", "0912", "Example Street"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Errorf("the body leaks %q: %s", leak, rec.Body)
		}
	}
}

// SC-36: no token, a token that cannot be one and an id that cannot be one are the
// same 404 and never reach the database.
func TestAnOrderWithoutItsTokenIsTheSame404(t *testing.T) {
	db := &fakeDB{}
	h := mount(db, Options{})
	var first string
	for _, c := range []struct{ id, token string }{
		{"doi-24817", ""}, {"doi-24817", "has a space"}, {"doi-24817", strings.Repeat("a", 129)}, {"doi-24817", "a/b"},
		{"DOI-24817", "q3J0n8Zk2LmPa7Vx9TrBwE"}, {strings.Repeat("a", 101), "q3J0n8Zk2LmPa7Vx9TrBwE"}, {"-x", "q3J0n8Zk2LmPa7Vx9TrBwE"},
	} {
		req := httptest.NewRequest(http.MethodGet, "/v1/orders/"+c.id, nil)
		if c.token != "" {
			req.Header.Set("X-Order-Token", c.token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%+v: %d", c, rec.Code)
		}
		if first == "" {
			first = rec.Body.String()
		}
		if rec.Body.String() != first || !strings.Contains(first, `"not_found"`) {
			t.Errorf("%+v: body %q differs from %q", c, rec.Body, first)
		}
	}
	if db.touched() != 0 {
		t.Error("a request that cannot hold a token reached the database")
	}
}

// SC-36 and the privacy rule: the statement that builds the public view lists its
// columns and does not select a customer's phone, email, address or note.
func TestThePublicViewSelectsNoPrivateColumn(t *testing.T) {
	for _, name := range []string{"phone", "email", "address", "note", "is_sample", "source", "invented", "idempotency_key"} {
		for _, sql := range []string{viewSQL, tokenSQL} {
			if strings.Contains(sql, "o."+name) || strings.Contains(sql, "f."+name) || strings.Contains(sql, "'"+name+"'") {
				t.Errorf("the public view selects %s", name)
			}
		}
	}
}

// ORD-006: the shop never lists cod, bank or momo; what is offered is code.
func TestOnlyVNPayIsOffered(t *testing.T) {
	if len(offered) != 1 || offered[0] != "vnpay" {
		t.Errorf("offered: %v", offered)
	}
}

func TestAWayToPayWithNoProviderIsAvailableAndOneThatSaysNoIsNot(t *testing.T) {
	s := &service{providers: map[string]Provider{"closed": stubProvider{open: false}, "open": stubProvider{open: true}}}
	if !s.available("vnpay") || !s.available("open") || s.available("closed") {
		t.Error("availability does not follow the provider")
	}
}

type stubProvider struct {
	open bool
	url  string
	err  error
}

func (p stubProvider) Available() bool { return p.open }

func (p stubProvider) PaymentURL(context.Context, Payable, Client) (string, error) {
	return p.url, p.err
}

func TestClientOfReadsTheRightmostForwardedAddress(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	req.RemoteAddr = "10.0.0.9:51000"
	req.Host = "shop.example"
	if c := clientOf(req); c.IP != "10.0.0.9" || c.Origin != "http://shop.example" {
		t.Errorf("got %+v", c)
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 198.51.100.4")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "doi.example")
	if c := clientOf(req); c.IP != "198.51.100.4" || c.Origin != "https://doi.example" {
		t.Errorf("got %+v", c)
	}
}
