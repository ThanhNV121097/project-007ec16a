package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
)

// fakeStore stands for the database under the notice's table test.
type fakeStore struct {
	attempt             attemptRow
	findErr, failErr    error
	successErr          error
	wonRace             bool
	finds               int
	failures, successes []facts
}

func (f *fakeStore) find(context.Context, string) (attemptRow, error) {
	f.finds++
	return f.attempt, f.findErr
}

func (f *fakeStore) recordFailure(_ context.Context, _ attemptRow, n facts) error {
	if f.failErr == nil {
		f.failures = append(f.failures, n)
	}
	return f.failErr
}

func (f *fakeStore) recordSuccess(_ context.Context, _ attemptRow, n facts) (bool, error) {
	if f.successErr != nil || !f.wonRace {
		return false, f.successErr
	}
	f.successes = append(f.successes, n)
	return true, nil
}

func unpaidAttempt() attemptRow {
	return attemptRow{ID: "11111111-1111-1111-1111-111111111111", OrderID: "doi-24818", Number: "DOI-24818", Amount: 3140000}
}

func goodNotice() map[string]string {
	return map[string]string{
		"vnp_TmnCode": "TMN123", "vnp_Amount": "314000000", "vnp_BankCode": "NCB", "vnp_BankTranNo": "VNP14012345",
		"vnp_CardType": "ATM", "vnp_OrderInfo": "Thanh toan don hang DOI-24818", "vnp_PayDate": "20261126102500",
		"vnp_ResponseCode": "00", "vnp_TransactionNo": "14012345", "vnp_TransactionStatus": "00", "vnp_TxnRef": "DOI-24818-1",
	}
}

// noticeQuery is what VNPay's server sends: the parameters and the hash of them.
func noticeQuery(key string, params map[string]string) string {
	q := signed(key, params)
	return canonicalString(q) + "&vnp_SecureHash=" + q.Get("vnp_SecureHash")
}

func with(params map[string]string, changes map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range params {
		out[k] = v
	}
	for k, v := range changes {
		out[k] = v
	}
	return out
}

func callNotice(h http.Handler, rawQuery string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/v1/payments/vnpay/ipn", nil)
	req.URL.RawQuery = rawQuery
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func codeOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %s", rec.Body)
	}
	if len(got) != 2 || got["Message"] == "" {
		t.Fatalf("the body must be exactly RspCode and Message: %s", rec.Body)
	}
	return got["RspCode"]
}

func newNotice(f *fakeStore) (http.Handler, *VNPay) {
	v := provider("")
	v.store = f
	mux := http.NewServeMux()
	Register(mux, v)
	return mux, v
}

// PAY-002 AC-1 to AC-12: every row of the contract, as HTTP 200 with VNPay's shape.
func TestTheNoticeAnswersEveryCodeInVNPaysShape(t *testing.T) {
	good := goodNotice()
	tampered := noticeQuery(secret, good)
	tampered = tampered[:len(tampered)-1] + map[bool]string{true: "1", false: "0"}[strings.HasSuffix(tampered, "0")]
	paid := unpaidAttempt()
	paid.Paid = true
	cases := []struct {
		name       string
		query      string
		store      fakeStore
		want       string
		successes  int
		failures   int
		wantFinds  int
	}{
		{"a signed success for the right amount", noticeQuery(secret, good), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "00", 1, 0, 1},
		{"a success a twin notice already recorded", noticeQuery(secret, good), fakeStore{attempt: unpaidAttempt()}, "02", 0, 0, 1},
		{"an order already paid", noticeQuery(secret, good), fakeStore{attempt: paid, wonRace: true}, "02", 0, 0, 1},
		{"a customer who cancelled", noticeQuery(secret, with(good, map[string]string{"vnp_ResponseCode": "24", "vnp_TransactionStatus": "02"})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "00", 0, 1, 1},
		{"a response 00 with a transaction status that is not 00", noticeQuery(secret, with(good, map[string]string{"vnp_TransactionStatus": "01"})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "00", 0, 1, 1},
		{"a response 00 with no transaction status", noticeQuery(secret, with(good, map[string]string{"vnp_TransactionStatus": ""})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "00", 0, 1, 1},
		{"a signature one character off", tampered, fakeStore{attempt: unpaidAttempt(), wonRace: true}, "97", 0, 0, 0},
		{"a notice signed with another secret", noticeQuery("another", good), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "97", 0, 0, 0},
		{"no query at all", "", fakeStore{attempt: unpaidAttempt()}, "97", 0, 0, 0},
		{"another terminal, signed correctly", noticeQuery(secret, with(good, map[string]string{"vnp_TmnCode": "OTHER"})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "97", 0, 0, 0},
		{"a repeated vnp_ name", noticeQuery(secret, good) + "&vnp_TxnRef=DOI-24818-2", fakeStore{attempt: unpaidAttempt(), wonRace: true}, "97", 0, 0, 0},
		{"a query over 8 KiB", noticeQuery(secret, good) + "&x=" + strings.Repeat("a", 9000), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "97", 0, 0, 0},
		{"a malformed escape", noticeQuery(secret, good) + "&x=%zz", fakeStore{attempt: unpaidAttempt(), wonRace: true}, "97", 0, 0, 0},
		{"a semicolon", noticeQuery(secret, good) + ";x=1", fakeStore{attempt: unpaidAttempt(), wonRace: true}, "97", 0, 0, 0},
		{"a reference that is no attempt's", noticeQuery(secret, good), fakeStore{findErr: errNoAttempt}, "01", 0, 0, 1},
		{"a wrong amount", noticeQuery(secret, with(good, map[string]string{"vnp_Amount": "100000"})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "04", 0, 0, 1},
		{"an amount that is not a number", noticeQuery(secret, with(good, map[string]string{"vnp_Amount": "abc"})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "04", 0, 0, 1},
		{"an amount that is not a whole number of dong", noticeQuery(secret, with(good, map[string]string{"vnp_Amount": "314000050"})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "04", 0, 0, 1},
		{"an amount of 19 digits", noticeQuery(secret, with(good, map[string]string{"vnp_Amount": "1000000000000000000"})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "04", 0, 0, 1},
		{"an unknown reference AND a wrong amount: the first rule answers", noticeQuery(secret, with(good, map[string]string{"vnp_Amount": "1"})), fakeStore{findErr: errNoAttempt}, "01", 0, 0, 1},
		{"a wrong amount AND an order already paid: the first rule answers", noticeQuery(secret, with(good, map[string]string{"vnp_Amount": "1"})), fakeStore{attempt: paid}, "04", 0, 0, 1},
		{"a field VNPay added, signed with the rest", noticeQuery(secret, with(good, map[string]string{"vnp_NewField": "x"})), fakeStore{attempt: unpaidAttempt(), wonRace: true}, "00", 1, 0, 1},
		{"a parameter that is not vnp_, unsigned", noticeQuery(secret, good) + "&utm=1", fakeStore{attempt: unpaidAttempt(), wonRace: true}, "00", 1, 0, 1},
		{"a database that fails to find", noticeQuery(secret, good), fakeStore{findErr: errors.New("down")}, "99", 0, 0, 1},
		{"a database that fails to record a success", noticeQuery(secret, good), fakeStore{attempt: unpaidAttempt(), successErr: errors.New("down")}, "99", 0, 0, 1},
		{"a database that fails to record a failure", noticeQuery(secret, with(good, map[string]string{"vnp_ResponseCode": "24"})), fakeStore{attempt: unpaidAttempt(), failErr: errors.New("down")}, "99", 0, 0, 1},
	}
	for _, c := range cases {
		store := c.store
		h, _ := newNotice(&store)
		rec := callNotice(h, c.query)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" || rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: %d %v", c.name, rec.Code, rec.Header())
		}
		if got := codeOf(t, rec); got != c.want {
			t.Errorf("%s: RspCode %s, want %s (%s)", c.name, got, c.want, rec.Body)
		}
		if len(store.successes) != c.successes || len(store.failures) != c.failures || store.finds != c.wantFinds {
			t.Errorf("%s: %d successes, %d failures, %d finds; want %d, %d, %d", c.name, len(store.successes), len(store.failures), store.finds, c.successes, c.failures, c.wantFinds)
		}
	}
}

func TestTheNoticeKeepsOnlyWhatVNPaySaid(t *testing.T) {
	store := &fakeStore{attempt: unpaidAttempt(), wonRace: true}
	h, _ := newNotice(store)
	callNotice(h, noticeQuery(secret, goodNotice()))
	if len(store.successes) != 1 {
		t.Fatal("no success was recorded")
	}
	got := store.successes[0]
	if got.TransactionNo != "14012345" || got.BankCode != "NCB" || got.Amount != 3140000 || got.ResponseCode != "00" || got.TransactionStatus != "00" {
		t.Errorf("facts %+v", got)
	}
	if got.PaidTime == nil || got.PaidTime.Format("2006-01-02T15:04:05Z07:00") != "2026-11-26T10:25:00+07:00" {
		t.Errorf("pay time %v", got.PaidTime)
	}
	// A date that does not parse never refuses a payment.
	store = &fakeStore{attempt: unpaidAttempt(), wonRace: true}
	h, _ = newNotice(store)
	callNotice(h, noticeQuery(secret, with(goodNotice(), map[string]string{"vnp_PayDate": "yesterday"})))
	if len(store.successes) != 1 || store.successes[0].PaidTime != nil {
		t.Errorf("a bad date: %+v", store.successes)
	}
}

// PAY-005: with a key missing the server starts and the notice answers 97 to everything.
func TestWithoutKeysTheNoticeAnswers97ToEverything(t *testing.T) {
	v := NewVNPay(VNPayConfig{}, nil, quiet(), nil)
	store := &fakeStore{attempt: unpaidAttempt(), wonRace: true}
	v.store = store
	mux := http.NewServeMux()
	Register(mux, v)
	for _, q := range []string{"", noticeQuery(secret, goodNotice()), noticeQuery("", goodNotice())} {
		if got := codeOf(t, callNotice(mux, q)); got != "97" {
			t.Errorf("RspCode %s", got)
		}
	}
	if store.finds != 0 {
		t.Error("the database was asked")
	}
}

// PAY-005 AC-4: the secret, the signature and the query are in no log line or body.
func TestTheLogAndTheBodyHoldNoSecretNoSignatureAndNoQuery(t *testing.T) {
	var logged bytes.Buffer
	v := NewVNPay(VNPayConfig{TmnCode: "TMN123", HashSecret: config.NewSecret(secret)}, nil, slog.New(slog.NewTextHandler(&logged, nil)), nil)
	v.store = &fakeStore{attempt: unpaidAttempt(), wonRace: true}
	mux := http.NewServeMux()
	Register(mux, v)
	query := noticeQuery(secret, goodNotice())
	hash := query[strings.LastIndex(query, "=")+1:]
	bad := callNotice(mux, query[:len(query)-3]+"abc")
	good := callNotice(mux, query)
	for _, text := range []string{logged.String(), bad.Body.String(), good.Body.String(), bad.Header().Get("Content-Type")} {
		for _, leak := range []string{secret, hash, "14012345", "vnp_"} {
			if strings.Contains(text, leak) {
				t.Errorf("%q is in %q", leak, text)
			}
		}
	}
	if !strings.Contains(logged.String(), "DOI-24818") {
		t.Errorf("the log does not name the order: %s", logged.String())
	}
}
