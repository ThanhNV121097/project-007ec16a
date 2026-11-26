package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

// These run the real migrations and SQL like payments_db_test.go: they need
// TEST_DATABASE_URL and skip without it.

// momoWorld is a database, a MoMo provider whose API is a stand-in, and the notice route.
func momoWorld(t *testing.T, calls *atomic.Int32, resultCode int) (world, *MoMo) {
	t.Helper()
	w := newWorld(t)
	api := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]any{"resultCode": resultCode, "message": "stand-in", "orderId": req["orderId"], "payUrl": "https://test-payment.momo.vn/pay/abc"})
	}))
	t.Cleanup(api.Close)
	m := NewMoMo(MoMoConfig{
		PartnerCode: momoPartner, AccessKey: config.NewSecret(momoAccess), SecretKey: config.NewSecret(momoSecretKey),
		APIURL: api.URL, PublicBaseURL: "https://shop.example.test",
	}, w.pool, quiet(), nil)
	mux := http.NewServeMux()
	Register(mux, w.v, m)
	w.h = mux
	return w, m
}

func (w world) momoStatus(t *testing.T, number string, changes map[string]any) int {
	t.Helper()
	fields := momoWith(goodMoMoNotice(), map[string]any{"orderId": number + "-1", "requestId": number + "-1"})
	for k, v := range changes {
		fields[k] = v
	}
	return postMoMo(w.h, signedBody(t, fields, momoSecretKey)).Code
}

func (w world) askMoMo(m *MoMo, id string) (string, error) {
	return m.PaymentURL(context.Background(), orders.Payable{ID: id}, shopClient)
}

// PAY-021: the attempt is stored, a verified success pays the order once and keeps
// MoMo's own codes, and the check accepts them.
func TestAMoMoOrderIsPaidOnceByItsVerifiedNotice(t *testing.T) {
	var calls atomic.Int32
	w, m := momoWorld(t, &calls, 0)
	w.order(t, "doi-91001", seed{age: 5 * time.Minute, way: "momo"})
	addr, err := w.askMoMo(m, "doi-91001")
	if err != nil || addr != "https://test-payment.momo.vn/pay/abc" || calls.Load() != 1 {
		t.Fatalf("%q %v after %d calls", addr, err, calls.Load())
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_attempts WHERE order_id = 'doi-91001' AND provider = 'momo' AND reference = 'DOI-91001-1' AND amount = 3140000"); n != 1 {
		t.Fatalf("%d attempts", n)
	}
	if got := w.momoStatus(t, "DOI-91001", nil); got != 204 {
		t.Fatalf("the notice: %d", got)
	}
	if status, paid := w.state(t, "doi-91001"); status != "confirmed" || !paid {
		t.Errorf("status %s paid %v", status, paid)
	}
	if n := one[int](t, w, `SELECT count(*) FROM payment_notices n JOIN payment_attempts a ON a.id = n.attempt_id
		WHERE a.order_id = 'doi-91001' AND n.is_success AND n.response_code = '0' AND n.transaction_status = ''
		AND n.transaction_no = '4014012345' AND n.bank_code = 'qr' AND n.amount = 3140000 AND n.paid_time IS NOT NULL`); n != 1 {
		t.Errorf("%d success notices with MoMo's facts", n)
	}
	if got := w.momoStatus(t, "DOI-91001", nil); got != 204 {
		t.Errorf("the same notice again: %d", got)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_notices n JOIN payment_attempts a ON a.id = n.attempt_id WHERE a.order_id = 'doi-91001'"); n != 1 {
		t.Errorf("%d notices kept", n)
	}
}

// The integrity requirement: twenty copies at one moment, one payment.
func TestTwentyMoMoNoticesAtOnceMarkTheOrderPaidOnce(t *testing.T) {
	var calls atomic.Int32
	w, m := momoWorld(t, &calls, 0)
	w.order(t, "doi-91002", seed{age: 5 * time.Minute, way: "momo"})
	if _, err := w.askMoMo(m, "doi-91002"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	codes := make(chan int, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- w.momoStatus(t, "DOI-91002", nil)
		}()
	}
	wg.Wait()
	close(codes)
	for c := range codes {
		if c != 204 {
			t.Errorf("a notice answered %d", c)
		}
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_notices WHERE is_success"); n != 1 {
		t.Errorf("%d success notices", n)
	}
}

// PAY-021 AC-6: a notice wrong in one way changes no order and writes no row.
func TestWrongMoMoNoticesChangeNothing(t *testing.T) {
	var calls atomic.Int32
	w, m := momoWorld(t, &calls, 0)
	w.order(t, "doi-91003", seed{age: 5 * time.Minute, way: "momo"})
	w.order(t, "doi-91004", seed{age: 5 * time.Minute})
	if _, err := w.askMoMo(m, "doi-91003"); err != nil {
		t.Fatal(err)
	}
	vnpay := w.reference(t, "doi-91004")
	for i := 0; i < 10; i++ {
		for name, changes := range map[string]map[string]any{
			"a wrong amount":      {"amount": 1000},
			"an unknown reference": {"orderId": "DOI-99999-1"},
			"a VNPay reference":    {"orderId": vnpay},
		} {
			if got := w.momoStatus(t, "DOI-91003", changes); got != 400 {
				t.Fatalf("%s: %d", name, got)
			}
		}
	}
	if status, paid := w.state(t, "doi-91003"); status != "placed" || paid {
		t.Errorf("status %s paid %v", status, paid)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_notices"); n != 0 {
		t.Errorf("%d notices written for wrong ones", n)
	}
}

// A failure is one row however often MoMo repeats it; a success afterwards still pays; and
// a success for an order already given up is recorded while the status stays cancelled.
func TestMoMoFailuresAndLatePayments(t *testing.T) {
	var calls atomic.Int32
	w, m := momoWorld(t, &calls, 0)
	w.order(t, "doi-91005", seed{age: 5 * time.Minute, way: "momo"})
	w.order(t, "doi-91006", seed{age: 5 * time.Minute, way: "momo"})
	for _, id := range []string{"doi-91005", "doi-91006"} {
		if _, err := w.askMoMo(m, id); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if got := w.momoStatus(t, "DOI-91005", map[string]any{"resultCode": 1006}); got != 204 {
			t.Fatalf("failure notice %d: %d", i+1, got)
		}
	}
	if status, paid := w.state(t, "doi-91005"); status != "placed" || paid {
		t.Errorf("status %s paid %v", status, paid)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_notices WHERE NOT is_success AND response_code = '1006'"); n != 1 {
		t.Errorf("%d failure rows", n)
	}
	if got := w.momoStatus(t, "DOI-91005", nil); got != 204 {
		t.Fatalf("the success after the failure: %d", got)
	}
	if _, paid := w.state(t, "doi-91005"); !paid {
		t.Error("the success did not pay the order")
	}

	w.exec(t, "UPDATE orders SET status = 'cancelled' WHERE id = 'doi-91006'")
	if got := w.momoStatus(t, "DOI-91006", nil); got != 204 {
		t.Fatalf("the late success: %d", got)
	}
	if status, paid := w.state(t, "doi-91006"); status != "cancelled" || !paid {
		t.Errorf("status %s paid %v", status, paid)
	}
	if n := w.pairs(t, "doi-91006"); n != 8 {
		t.Errorf("the shelf holds %d, a late payment must not move it", n)
	}
}

// ERD E3: the attempt is committed before MoMo is asked, so an answer that is no address
// leaves an attempt and no address; and an order that cannot be paid never reaches MoMo.
func TestTheAttemptIsReservedBeforeMoMoAnswersAndRefusedOrdersNeverAskIt(t *testing.T) {
	var calls atomic.Int32
	w, m := momoWorld(t, &calls, 22)
	w.order(t, "doi-91007", seed{age: time.Minute, way: "momo"})
	if addr, err := w.askMoMo(m, "doi-91007"); err == nil || addr != "" {
		t.Fatalf("%q %v", addr, err)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_attempts WHERE order_id = 'doi-91007'"); n != 1 || calls.Load() != 1 {
		t.Errorf("%d attempts after %d calls", n, calls.Load())
	}
	for id, s := range map[string]seed{
		"doi-91008": {age: 31 * time.Minute, way: "momo"},
		"doi-91009": {age: time.Minute, way: "momo", paid: true},
		"doi-91010": {age: time.Minute, way: "momo", status: "cancelled"},
		"doi-91011": {age: time.Minute, way: "bank"},
	} {
		w.order(t, id, s)
		before := calls.Load()
		if _, err := w.askMoMo(m, id); err == nil {
			t.Errorf("%s: an address was given", id)
		}
		if calls.Load() != before || one[int](t, w, "SELECT count(*) FROM payment_attempts WHERE order_id = $1", id) != 0 {
			t.Errorf("%s: MoMo was asked or an attempt was written for a refusal", id)
		}
	}
}

// PAY-003 and PAY-020 item 5: the rule follows the order's way, with its own window.
func TestTheTwoRulesFollowTheWayOfTheOrder(t *testing.T) {
	w := newWorld(t)
	cases := []struct {
		id         string
		s          seed
		wantStatus string
		wantPairs  int
	}{
		{"doi-91101", seed{age: 31 * time.Minute, way: "momo"}, "cancelled", 10},
		{"doi-91102", seed{age: 29 * time.Minute, way: "momo"}, "placed", 8},
		{"doi-91103", seed{age: 2 * time.Hour, way: "momo", status: "packed"}, "cancelled", 10},
		{"doi-91104", seed{age: 2 * time.Hour, way: "momo", status: "shipped"}, "shipped", 8},
		{"doi-91105", seed{age: 2 * time.Hour, way: "momo", paid: true}, "placed", 8},
		{"doi-91111", seed{age: 25 * time.Hour, way: "bank"}, "cancelled", 10},
		{"doi-91112", seed{age: 24*time.Hour + time.Second, way: "bank"}, "cancelled", 10},
		{"doi-91113", seed{age: 23 * time.Hour, way: "bank"}, "placed", 8},
		{"doi-91114", seed{age: 2 * time.Hour, way: "bank"}, "placed", 8},
		{"doi-91115", seed{age: 25 * time.Hour, way: "bank", paid: true}, "placed", 8},
		{"doi-91116", seed{age: 25 * time.Hour, way: "bank", status: "shipped"}, "shipped", 8},
		{"doi-91117", seed{age: 25 * time.Hour, way: "bank", status: "delivered"}, "delivered", 8},
		{"doi-91118", seed{age: 25 * time.Hour, way: "bank", status: "confirmed"}, "cancelled", 10},
		{"doi-91119", seed{age: 25 * time.Hour, way: "bank", sample: true}, "placed", 8},
		{"doi-91120", seed{age: 25 * time.Hour, way: "momo", sample: true}, "placed", 8},
		{"doi-91121", seed{age: 25 * time.Hour, way: "bank", status: "cancelled"}, "cancelled", 8},
	}
	for _, c := range cases {
		w.order(t, c.id, c.s)
	}
	for run := 1; run <= 2; run++ {
		runExpiry(context.Background(), w.pool, quiet(), time.Now)
		for _, c := range cases {
			if status, _ := w.state(t, c.id); status != c.wantStatus {
				t.Errorf("run %d, %s: status %s, want %s", run, c.id, status, c.wantStatus)
			}
			if n := w.pairs(t, c.id); n != c.wantPairs {
				t.Errorf("run %d, %s: %d pairs on the shelf, want %d", run, c.id, n, c.wantPairs)
			}
		}
	}
}

// ERD section 2.1: the check refuses a MoMo success written with VNPay's codes, accepts
// MoMo's own, and still refuses a VNPay success the codes do not support.
func TestTheCheckAcceptsMoMosSuccessAndRefusesWhatItDoesNotSupport(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-91201", seed{age: time.Minute})
	id := one[string](t, w, `INSERT INTO payment_attempts (order_id, provider, reference, amount, expires_at)
		VALUES ('doi-91201', 'momo', 'DOI-91201-1', 3140000, now() + interval '20 minutes') RETURNING id::text`)
	notice := func(code, status string) error {
		_, err := w.pool.Exec(context.Background(), `INSERT INTO payment_notices (attempt_id, transaction_no, amount, response_code, transaction_status, is_success)
			VALUES ($1::uuid, $2, 3140000, $3, $4, true)`, id, code+status, code, status)
		return err
	}
	for name, c := range map[string]struct {
		code, status string
		ok           bool
	}{
		"MoMo's codes with VNPay's status": {"0", "00", false},
		"VNPay's code with no status":      {"00", "", false},
		"a failure code marked success":    {"1006", "", false},
		"MoMo's own codes":                 {"0", "", true},
	} {
		if err := notice(c.code, c.status); (err == nil) != c.ok {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// ERD section 2.3: the MoMo note no longer says the order is marked paid at once.
func TestTheMoMoNoteIsReworded(t *testing.T) {
	w := newWorld(t)
	if note := one[string](t, w, "SELECT note FROM payment_methods WHERE id = 'momo'"); note != "Confirm the payment in the MoMo app." {
		t.Errorf("the note is %q", note)
	}
}
