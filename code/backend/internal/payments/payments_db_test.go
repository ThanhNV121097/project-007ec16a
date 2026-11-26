package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/dbtest"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests run the real migrations and the real SQL against PostgreSQL. They
// read TEST_DATABASE_URL, as CI's backend job sets it, and skip without it (on a
// developer's machine and in run_tests it is not set).

type world struct {
	pool *pgxpool.Pool
	v    *VNPay
	h    http.Handler
}

func newWorld(t *testing.T) world {
	t.Helper()
	pool := dbtest.Migrated(t)
	v := NewVNPay(VNPayConfig{TmnCode: "TMN123", HashSecret: config.NewSecret(secret), PaymentURL: config.DefaultVNPayPage}, pool, quiet(), nil)
	mux := http.NewServeMux()
	Register(mux, v, nil)
	return world{pool: pool, v: v, h: mux}
}

func (w world) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := w.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func one[T any](t *testing.T, w world, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := w.pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

// seed is an order of two pairs of 1.570.000 VND (3.140.000 VND in all) whose shoe
// had 10 pairs before it was placed, so a cancel that returns them makes 10 again.
type seed struct {
	age          time.Duration
	status, way  string
	paid, sample bool
}

func (w world) order(t *testing.T, id string, s seed) {
	t.Helper()
	if s.status == "" {
		s.status = "placed"
	}
	if s.way == "" {
		s.way = "vnpay"
	}
	shoe := "shoe-" + id
	w.exec(t, "INSERT INTO products (id, name, price, status) VALUES ($1, $2, 1570000, 'live')", shoe, "Shoe "+id)
	w.exec(t, "INSERT INTO product_sizes (product_id, size, size_value, pairs) VALUES ($1, '38', 38, 8)", shoe)
	w.exec(t, `INSERT INTO orders (id, number, placed_at, status, payment_method_id, paid, delivery_option_id, delivery_fee, eta,
		customer, phone, address, city, note, total, is_sample)
		VALUES ($1, $2, $3, $4, $5, $6, 'standard', 0, '2026-12-01', 'Lan Nguyen', '0912 345 678', '12 Example Street', 'Da Nang', '', 3140000, $7)`,
		id, strings.ToUpper(id), time.Now().Add(-s.age), s.status, s.way, s.paid, s.sample)
	w.exec(t, "INSERT INTO order_items (order_id, line_no, product_id, size, qty, unit_price) VALUES ($1, 1, $2, '38', 2, 1570000)", id, shoe)
}

func (w world) pairs(t *testing.T, id string) int {
	t.Helper()
	return one[int](t, w, "SELECT pairs FROM product_sizes WHERE product_id = $1 AND size = '38'", "shoe-"+id)
}

func (w world) state(t *testing.T, id string) (status string, paid bool) {
	t.Helper()
	status = one[string](t, w, "SELECT status FROM orders WHERE id = $1", id)
	paid = one[bool](t, w, "SELECT paid FROM orders WHERE id = $1", id)
	return status, paid
}

func (w world) ask(id string) (string, error) {
	return w.v.PaymentURL(context.Background(), orders.Payable{ID: id}, shopClient)
}

func (w world) reference(t *testing.T, id string) string {
	t.Helper()
	addr, err := w.ask(id)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(addr)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("vnp_TxnRef")
}

// rspOf is the RspCode of a notice answered through the real handler.
func (w world) rspOf(changes map[string]string) string {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/payments/vnpay/ipn", nil)
	req.URL.RawQuery = noticeQuery(secret, with(goodNotice(), changes))
	w.h.ServeHTTP(rec, req)
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusOK {
		return fmt.Sprintf("HTTP %d %s", rec.Code, rec.Body)
	}
	return got["RspCode"]
}

func TestAttemptsAreNumberedAndTakeTheStoredTotalAndDeadline(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90001", seed{age: 10 * time.Minute})
	for _, want := range []string{"DOI-90001-1", "DOI-90001-2"} {
		// The total the caller names is never used.
		addr, err := w.v.PaymentURL(context.Background(), orders.Payable{ID: "doi-90001", Total: 1000, Number: "X"}, shopClient)
		if err != nil {
			t.Fatal(err)
		}
		u, _ := url.Parse(addr)
		if u.Query().Get("vnp_TxnRef") != want || u.Query().Get("vnp_Amount") != "314000000" || !verify(secret, u.Query()) {
			t.Errorf("%s: %s", want, addr)
		}
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_attempts WHERE order_id = 'doi-90001' AND amount = 3140000 AND provider = 'vnpay'"); n != 2 {
		t.Errorf("%d attempts stored", n)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_attempts a JOIN orders o ON o.id = a.order_id WHERE a.expires_at = o.placed_at + interval '30 minutes'"); n != 2 {
		t.Errorf("%d attempts expire at the order's deadline", n)
	}
}

// PAY-001 item 5, ERD E1: two presses at the same moment never share a reference.
func TestTwentyAsksAtOnceGetTwentyDistinctReferences(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90002", seed{age: time.Minute})
	var wg sync.WaitGroup
	refs := make(chan string, 20)
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr, err := w.ask("doi-90002")
			if err != nil {
				errs <- err
				return
			}
			u, _ := url.Parse(addr)
			refs <- u.Query().Get("vnp_TxnRef")
		}()
	}
	wg.Wait()
	close(refs)
	close(errs)
	for err := range errs {
		t.Errorf("an ask failed: %v", err)
	}
	seen := map[string]bool{}
	for r := range refs {
		seen[r] = true
	}
	if len(seen) != 20 {
		t.Fatalf("%d distinct references: %v", len(seen), seen)
	}
	for i := 1; i <= 20; i++ {
		if !seen[fmt.Sprintf("DOI-90002-%d", i)] {
			t.Errorf("DOI-90002-%d is missing", i)
		}
	}
}

func TestAStateThatCannotBePaidIsRefusedAndWritesNoAttempt(t *testing.T) {
	w := newWorld(t)
	for id, c := range map[string]struct {
		s    seed
		want string
	}{
		"doi-90010": {seed{age: 31 * time.Minute}, orders.MsgRunOut},
		"doi-90011": {seed{age: time.Minute, status: "cancelled"}, "This order was cancelled, so it cannot be paid."},
		"doi-90012": {seed{age: time.Minute, paid: true}, "This order is already paid."},
		"doi-90013": {seed{age: time.Minute, way: "cod"}, "This order cannot be paid online."},
	} {
		w.order(t, id, c.s)
		_, err := w.ask(id)
		var refusal *orders.Refusal
		if !errors.As(err, &refusal) || refusal.Message != c.want {
			t.Errorf("%s: %v, want the refusal %q", id, err, c.want)
		}
		if n := one[int](t, w, "SELECT count(*) FROM payment_attempts WHERE order_id = $1", id); n != 0 {
			t.Errorf("%s: %d attempts written for a refusal", id, n)
		}
	}
	if _, err := w.ask("doi-99999"); !errors.Is(err, orders.ErrNotFound) {
		t.Errorf("an order that does not exist: %v", err)
	}
}

func TestTheFiftyFirstAttemptIsRefused(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90014", seed{age: time.Minute})
	for i := 0; i < 50; i++ {
		if _, err := w.ask("doi-90014"); err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	_, err := w.ask("doi-90014")
	var refusal *orders.Refusal
	if !errors.As(err, &refusal) || refusal.Message != MsgTooManyAttempts {
		t.Fatalf("the 51st attempt: %v", err)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_attempts WHERE order_id = 'doi-90014'"); n != 50 {
		t.Errorf("%d attempts", n)
	}
}

// PAY-002 AC-1 to AC-3: the notice pays the order once, "placed" becomes
// "confirmed", and an earlier reference of the order still leads to it.
func TestASuccessNoticePaysOnceAndKeepsWhatVNPaySaid(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90020", seed{age: 5 * time.Minute})
	w.order(t, "doi-90021", seed{age: 5 * time.Minute, status: "packed"})
	first, second := w.reference(t, "doi-90020"), w.reference(t, "doi-90020")
	packed := w.reference(t, "doi-90021")

	if got := w.rspOf(map[string]string{"vnp_TxnRef": first}); got != "00" {
		t.Fatalf("the first notice: %s", got)
	}
	if status, paid := w.state(t, "doi-90020"); status != "confirmed" || !paid {
		t.Errorf("status %s paid %v", status, paid)
	}
	if n := one[int](t, w, `SELECT count(*) FROM payment_notices n JOIN payment_attempts a ON a.id = n.attempt_id
		WHERE a.order_id = 'doi-90020' AND n.is_success AND n.transaction_no = '14012345' AND n.bank_code = 'NCB'
		AND n.amount = 3140000 AND n.response_code = '00' AND n.transaction_status = '00' AND n.paid_time IS NOT NULL`); n != 1 {
		t.Errorf("%d success notices with the kept facts", n)
	}
	if got := w.rspOf(map[string]string{"vnp_TxnRef": first}); got != "02" {
		t.Errorf("the same notice again: %s", got)
	}
	// The customer paid on the first page after opening a second: the second
	// reference leads to the same order, which is paid already.
	if got := w.rspOf(map[string]string{"vnp_TxnRef": second}); got != "02" {
		t.Errorf("a notice for the order's other reference: %s", got)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_notices n JOIN payment_attempts a ON a.id = n.attempt_id WHERE a.order_id = 'doi-90020'"); n != 1 {
		t.Errorf("%d notices kept", n)
	}

	if got := w.rspOf(map[string]string{"vnp_TxnRef": packed}); got != "00" {
		t.Fatalf("a notice for a packed order: %s", got)
	}
	if status, paid := w.state(t, "doi-90021"); status != "packed" || !paid {
		t.Errorf("a packed order became %s, paid %v", status, paid)
	}
}

// PAY-002 AC-4 and the integrity requirement: twenty copies at one moment, one payment.
func TestTwentyNoticesAtOnceMarkTheOrderPaidOnce(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90030", seed{age: 5 * time.Minute})
	ref := w.reference(t, "doi-90030")
	var wg sync.WaitGroup
	codes := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- w.rspOf(map[string]string{"vnp_TxnRef": ref})
		}()
	}
	wg.Wait()
	close(codes)
	count := map[string]int{}
	for c := range codes {
		count[c]++
	}
	if count["00"] != 1 || count["02"] != 19 {
		t.Fatalf("answers: %v", count)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_notices WHERE is_success"); n != 1 {
		t.Errorf("%d success notices", n)
	}
}

// PAY-002 AC-8: a cancelled payment is received and changes nothing; an identical
// failure is one row; a success afterwards still pays.
func TestAFailureNoticeLeavesTheOrderUnpaidAndTwinsAreOneRow(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90040", seed{age: 5 * time.Minute})
	ref := w.reference(t, "doi-90040")
	cancelled := map[string]string{"vnp_TxnRef": ref, "vnp_ResponseCode": "24", "vnp_TransactionStatus": "02"}
	for i := 0; i < 2; i++ {
		if got := w.rspOf(cancelled); got != "00" {
			t.Fatalf("failure notice %d: %s", i+1, got)
		}
	}
	if status, paid := w.state(t, "doi-90040"); status != "placed" || paid {
		t.Errorf("status %s paid %v", status, paid)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_notices WHERE NOT is_success"); n != 1 {
		t.Errorf("%d failure rows", n)
	}
	if got := w.rspOf(map[string]string{"vnp_TxnRef": ref}); got != "00" {
		t.Fatalf("the success after the failure: %s", got)
	}
	if _, paid := w.state(t, "doi-90040"); !paid {
		t.Error("the success did not pay the order")
	}
}

// PAY-002 AC-5 to AC-7: a notice that is wrong in one way changes no order and writes no row.
func TestWrongNoticesChangeNothing(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90050", seed{age: 5 * time.Minute})
	ref := w.reference(t, "doi-90050")
	for i := 0; i < 20; i++ {
		if got := w.rspOf(map[string]string{"vnp_TxnRef": ref, "vnp_Amount": "100000"}); got != "04" {
			t.Fatalf("a wrong amount: %s", got)
		}
		if got := w.rspOf(map[string]string{"vnp_TxnRef": "DOI-99999-1"}); got != "01" {
			t.Fatalf("an unknown reference: %s", got)
		}
		if got := w.rspOf(map[string]string{"vnp_TxnRef": ref, "vnp_TmnCode": "OTHER"}); got != "97" {
			t.Fatalf("another terminal: %s", got)
		}
	}
	if status, paid := w.state(t, "doi-90050"); status != "placed" || paid {
		t.Errorf("status %s paid %v", status, paid)
	}
	if n := one[int](t, w, "SELECT count(*) FROM payment_notices"); n != 0 {
		t.Errorf("%d notices written for wrong ones", n)
	}
}

// PAY-002 item 9: a payment that arrives after the order was given up is recorded,
// "paid" is set, the status stays cancelled and the shelf is not touched.
func TestAPaymentForACancelledOrderMarksItPaidAndLeavesTheShelf(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90060", seed{age: 5 * time.Minute})
	ref := w.reference(t, "doi-90060")
	w.exec(t, "UPDATE orders SET status = 'cancelled' WHERE id = 'doi-90060'")
	if got := w.rspOf(map[string]string{"vnp_TxnRef": ref}); got != "00" {
		t.Fatalf("the notice: %s", got)
	}
	if status, paid := w.state(t, "doi-90060"); status != "cancelled" || !paid {
		t.Errorf("status %s paid %v", status, paid)
	}
	if n := w.pairs(t, "doi-90060"); n != 8 {
		t.Errorf("the shelf holds %d, a late payment must not move it", n)
	}
}

// PAY-003 AC-1 to AC-10: the rule gives up exactly the orders it is for, returns
// each one's pairs once, and a second run changes nothing.
func TestTheExpiryGivesUpTheRightOrdersAndReturnsTheirPairsOnce(t *testing.T) {
	w := newWorld(t)
	cases := []struct {
		id         string
		s          seed
		wantStatus string
		wantPairs  int
	}{
		{"doi-90101", seed{age: 31 * time.Minute}, "cancelled", 10},
		{"doi-90102", seed{age: 30 * time.Minute}, "cancelled", 10},
		{"doi-90103", seed{age: 29 * time.Minute}, "placed", 8},
		{"doi-90104", seed{age: 2 * time.Hour, paid: true, status: "confirmed"}, "confirmed", 8},
		{"doi-90105", seed{age: 2 * time.Hour, status: "confirmed"}, "cancelled", 10},
		{"doi-90106", seed{age: 2 * time.Hour, status: "packed"}, "cancelled", 10},
		{"doi-90107", seed{age: 2 * time.Hour, status: "shipped"}, "shipped", 8},
		{"doi-90108", seed{age: 2 * time.Hour, status: "delivered"}, "delivered", 8},
		{"doi-90109", seed{age: 2 * time.Hour, sample: true}, "placed", 8},
		{"doi-90110", seed{age: 2 * time.Hour, way: "cod"}, "placed", 8},
		{"doi-90111", seed{age: 2 * time.Hour, status: "cancelled"}, "cancelled", 8},
	}
	for _, c := range cases {
		w.order(t, c.id, c.s)
	}
	samples := one[string](t, w, "SELECT string_agg(id || ':' || status, ',' ORDER BY id) FROM orders WHERE is_sample AND id <> 'doi-90109'")
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
	// The 14 samples that came with the design are never given up.
	if after := one[string](t, w, "SELECT string_agg(id || ':' || status, ',' ORDER BY id) FROM orders WHERE is_sample AND id <> 'doi-90109'"); after != samples {
		t.Errorf("the samples changed:\n%s\n%s", samples, after)
	}
}

// PAY-003 AC-8: several runs at the same moment cancel an order once and return its pairs once.
func TestRunsAtOnceReturnThePairsOnce(t *testing.T) {
	w := newWorld(t)
	ids := []string{"doi-90201", "doi-90202", "doi-90203", "doi-90204", "doi-90205"}
	for _, id := range ids {
		w.order(t, id, seed{age: time.Hour})
	}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runExpiry(context.Background(), w.pool, quiet(), time.Now)
		}()
	}
	wg.Wait()
	for _, id := range ids {
		if status, _ := w.state(t, id); status != "cancelled" {
			t.Errorf("%s: %s", id, status)
		}
		if n := w.pairs(t, id); n != 10 {
			t.Errorf("%s: %d pairs, want 10 (returned once)", id, n)
		}
	}
}

// ERD section 2: each constraint refuses what it is there for.
func TestTheSchemaRefusesWhatItIsFor(t *testing.T) {
	w := newWorld(t)
	w.order(t, "doi-90301", seed{age: time.Minute})
	attempt := func(ref string) (string, error) {
		var id string
		err := w.pool.QueryRow(context.Background(), `INSERT INTO payment_attempts (order_id, provider, reference, amount, expires_at)
			VALUES ('doi-90301', 'vnpay', $1, 3140000, now() + interval '20 minutes') RETURNING id::text`, ref).Scan(&id)
		return id, err
	}
	if _, err := attempt(strings.Repeat("x", 101)); err == nil {
		t.Error("a 101-character reference was stored")
	}
	id, err := attempt("DOI-90301-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := attempt("DOI-90301-1"); err == nil {
		t.Error("a second attempt with an existing reference was stored")
	}
	notice := func(txn, code, status string, success bool) error {
		_, err := w.pool.Exec(context.Background(), `INSERT INTO payment_notices (attempt_id, transaction_no, amount, response_code, transaction_status, is_success)
			VALUES ($1::uuid, $2, 3140000, $3, $4, $5)`, id, txn, code, status, success)
		return err
	}
	if err := notice("1", "24", "02", true); err == nil {
		t.Error("a success notice with response code 24 was stored")
	}
	if err := notice("1", "00", "00", true); err != nil {
		t.Fatal(err)
	}
	if err := notice("2", "00", "00", true); err == nil {
		t.Error("two success notices of one attempt were stored")
	}
	if err := notice("3", "24", "02", false); err != nil {
		t.Fatal(err)
	}
	if err := notice("3", "24", "02", false); err == nil {
		t.Error("two identical failure notices of one attempt were stored")
	}
	if err := notice("4", "24", "02", false); err != nil {
		t.Errorf("a failure notice that differs was refused: %v", err)
	}
}
