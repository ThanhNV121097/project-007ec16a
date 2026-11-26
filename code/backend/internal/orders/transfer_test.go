package orders

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// PAY-020: the QR code is only for an unpaid, not cancelled bank-transfer order.
func TestCheckTransferRefusesEveryOrderThatHasNothingToPayInto(t *testing.T) {
	for _, c := range []struct {
		name string
		in   PayState
		want string
	}{
		{"an unpaid bank order", PayState{"placed", "bank", false, settledAt}, ""},
		{"a confirmed unpaid bank order", PayState{"confirmed", "bank", false, settledAt}, ""},
		{"cancelled beats paid", PayState{"cancelled", "bank", true, settledAt}, msgCancelled},
		{"cancelled", PayState{"cancelled", "bank", false, settledAt}, msgCancelled},
		{"paid", PayState{"confirmed", "bank", true, settledAt}, msgPaid},
		{"paid beats the way", PayState{"confirmed", "vnpay", true, settledAt}, msgPaid},
		{"a VNPay order", PayState{"placed", "vnpay", false, settledAt}, msgNotBank},
		{"a MoMo order", PayState{"placed", "momo", false, settledAt}, msgNotBank},
		{"a sample order paid on delivery", PayState{"placed", "cod", false, settledAt}, msgNotBank},
	} {
		got := CheckTransfer(c.in)
		switch {
		case c.want == "" && got != nil:
			t.Errorf("%s: refused with %q", c.name, got.Message)
		case c.want != "" && (got == nil || got.Message != c.want):
			t.Errorf("%s: got %v, want %q", c.name, got, c.want)
		}
	}
}

// ORD-006 and PAY-022: the list a request is checked against is the one the server passed in.
func TestAWayIsAcceptedOnlyWhenTheShopListsIt(t *testing.T) {
	for _, c := range []struct {
		way    string
		listed []string
		ok     bool
	}{
		{"vnpay", defaultOffered, true},
		{"bank", defaultOffered, false},
		{"bank", []string{"vnpay", "bank"}, true},
		{"momo", []string{"vnpay", "bank"}, false},
		{"momo", []string{"vnpay", "bank", "momo"}, true},
		{"cod", []string{"vnpay", "bank", "momo"}, false},
	} {
		req := goodRequest()
		req.Payment = c.way
		req.clean()
		got := req.validate(c.listed)["payment"]
		if (got == "") != c.ok {
			t.Errorf("%s in %v: errors.payment = %q", c.way, c.listed, got)
		}
	}
}

// refRow is the access statement's row for one order.
type refRow struct {
	status, payment string
	paid            bool
}

func (r refRow) Scan(dest ...any) error {
	*(dest[0].(*string)) = "doi-24818"
	*(dest[1].(*string)) = "DOI-24818"
	*(dest[2].(*time.Time)) = settledAt
	*(dest[3].(*string)) = r.status
	*(dest[4].(*bool)) = r.paid
	*(dest[5].(*int64)) = 3140000
	*(dest[6].(*int64)) = 0
	*(dest[7].(*string)) = "standard"
	*(dest[8].(*string)) = r.payment
	*(dest[12].(*string)) = "Lan Nguyen"
	*(dest[13].(*string)) = "Da Nang"
	return nil
}

// rowDB answers every single-row read with one order.
type rowDB struct{ row pgx.Row }

func (rowDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("no transaction")
}
func (d rowDB) QueryRow(context.Context, string, ...any) pgx.Row { return d.row }
func (rowDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return emptyRows{}, nil
}

// bankStub is a provider that takes transfers.
type bankStub struct {
	open bool
	seen []int64
}

func (b *bankStub) Available() bool { return b.open }
func (*bankStub) PaymentURL(context.Context, Payable, Client) (string, error) {
	return "", nil
}
func (b *bankStub) Transfer(number string, total int64) (TransferDetails, error) {
	b.seen = append(b.seen, total)
	return TransferDetails{
		Holder: "NGUYEN VAN AN", Bank: "Vietcombank", AccountNumber: "0071001234567",
		Amount: total, Note: number, Payload: "payload", QR: []string{"1"},
	}, nil
}

func getTransfer(h http.Handler, target, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if token != "" {
		req.Header.Set("X-Order-Token", token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// SC-11 to SC-13: the details are the stored order's, to its token alone.
func TestTheTransferRouteAnswersTheStoredOrdersDetails(t *testing.T) {
	bank := &bankStub{open: true}
	h := mount(rowDB{row: refRow{status: "placed", payment: "bank"}}, Options{Providers: map[string]Provider{"bank": bank}})
	// An amount in the address changes nothing: the total is the order's.
	rec := getTransfer(h, "/v1/orders/doi-24818/transfer?amount=1", "a-token")
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("%d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 7 || body["amount"] != float64(3140000) || body["note"] != "DOI-24818" || body["holder"] != "NGUYEN VAN AN" {
		t.Errorf("body %v", body)
	}
	for _, leak := range []string{"phone", "email", "address", "Lan", "Da Nang", "status", "doi-24818"} {
		if strings.Contains(rec.Body.String(), `"`+leak) && leak != "doi-24818" {
			t.Errorf("the answer holds %q", leak)
		}
	}
	if len(bank.seen) != 1 || bank.seen[0] != 3140000 {
		t.Errorf("the provider was asked for %v", bank.seen)
	}
}

func TestTheTransferRouteRefusesWhatHasNothingToPayInto(t *testing.T) {
	for _, c := range []struct {
		name    string
		row     refRow
		bank    Provider
		want    int
		message string
	}{
		{"a paid order", refRow{"confirmed", "bank", true}, &bankStub{open: true}, 409, msgPaid},
		{"a cancelled order", refRow{"cancelled", "bank", false}, &bankStub{open: true}, 409, msgCancelled},
		{"a VNPay order", refRow{"placed", "vnpay", false}, &bankStub{open: true}, 409, msgNotBank},
		{"the account removed", refRow{"placed", "bank", false}, &bankStub{open: false}, 503, msgUnavailable},
		{"no provider that takes transfers", refRow{"placed", "bank", false}, settleProvider{open: true}, 503, msgUnavailable},
	} {
		h := mount(rowDB{row: c.row}, Options{Providers: map[string]Provider{"bank": c.bank}})
		rec := getTransfer(h, "/v1/orders/doi-24818/transfer", "a-token")
		var body struct {
			Message string            `json:"message"`
			Errors  map[string]string `json:"errors"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != c.want || body.Message != c.message || body.Errors != nil {
			t.Errorf("%s: %d %s", c.name, rec.Code, rec.Body)
		}
	}
}

// SC-13: no token, a token that opens nothing and a bad id are one 404, and a request
// that cannot name an order asks the database nothing.
func TestTheTransferRouteHasOneNotFound(t *testing.T) {
	db := &fakeDB{}
	h := mount(db, Options{Providers: map[string]Provider{"bank": &bankStub{open: true}}})
	none := getTransfer(h, "/v1/orders/doi-24818/transfer", "")
	if none.Code != http.StatusNotFound || db.touched() != 0 {
		t.Errorf("no token: %d, %d queries", none.Code, db.touched())
	}
	wrong := getTransfer(h, "/v1/orders/doi-24818/transfer", "another-token")
	if wrong.Code != http.StatusNotFound || wrong.Body.String() != none.Body.String() {
		t.Errorf("another token: %d %s, want %s", wrong.Code, wrong.Body, none.Body)
	}
	bad := getTransfer(h, "/v1/orders/..%2Fx/transfer", "a-token")
	if bad.Code != http.StatusNotFound {
		t.Errorf("a bad id: %d", bad.Code)
	}
}
