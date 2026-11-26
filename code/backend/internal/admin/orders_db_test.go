package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/dbtest"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

// These run the real handlers, the real SQL and the real migrations against
// PostgreSQL. They read TEST_DATABASE_URL, as CI's backend job sets it, and
// skip without it.

func newOrdersEnv(t *testing.T) *env {
	t.Helper()
	pool := dbtest.Migrated(t)
	if err := auth.EnsureAdmin(context.Background(), pool, "dealer@example.com", "Dealer-Pass1", quiet()); err != nil {
		t.Fatal(err)
	}
	svc := auth.New(pool, quiet())
	mux := http.NewServeMux()
	svc.Register(mux)
	orders.Register(mux, pool, quiet(), orders.Options{})
	RegisterOrders(mux, pool, quiet())
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { orderGone(w) })
	e := &env{t: t, pool: pool, svc: svc, h: svc.Resolve()(httpx.RecordRoute(mux))}
	code, cookie := e.signIn("dealer@example.com", "Dealer-Pass1")
	if code != 200 {
		t.Fatalf("the admin cannot sign in: %d", code)
	}
	e.admin = cookie
	return e
}

// call sends one request; headers are name, value pairs.
func (e *env) call(method, path, body string, cookie *http.Cookie, headers ...string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	if cookie != nil {
		req.AddCookie(&http.Cookie{Name: "session", Value: cookie.Value})
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func (e *env) shoe(id string, pairs map[string]int) {
	e.t.Helper()
	ctx := context.Background()
	if _, err := e.pool.Exec(ctx, "INSERT INTO products (id, name, price, status) VALUES ($1, $2, 1000000, 'live')", id, "Shoe "+id); err != nil {
		e.t.Fatal(err)
	}
	for size, n := range pairs {
		if _, err := e.pool.Exec(ctx, "INSERT INTO product_sizes (product_id, size, size_value, pairs) VALUES ($1, $2, $2::numeric, $3)", id, size, n); err != nil {
			e.t.Fatal(err)
		}
	}
}

func (e *env) pairs(id, size string) int {
	return e.count("SELECT pairs FROM product_sizes WHERE product_id = $1 AND size = $2", id, size)
}

// place makes an order through checkout and answers its id and token.
func (e *env) place(key, product, size string, qty int) (id, token string) {
	e.t.Helper()
	body := fmt.Sprintf(`{"items":[{"product":%q,"size":%q,"qty":%d}],"delivery":"standard","payment":"vnpay","customer":"Lan Nguyen",`+
		`"phone":"0912 345 678","address":"12 Example Street, Ward 1","city":"Da Nang","note":""}`, product, size, qty)
	rec := e.call(http.MethodPost, "/v1/orders", body, nil, "Idempotency-Key", key)
	if rec.Code != http.StatusCreated {
		e.t.Fatalf("place: %d %s", rec.Code, rec.Body)
	}
	v := decode(e.t, rec)
	return v["order"].(map[string]any)["id"].(string), v["token"].(string)
}

func (e *env) put(id, body string) *httptest.ResponseRecorder {
	return e.call(http.MethodPut, "/v1/orders/"+id, body, e.admin)
}

func orderErrors(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	errs, _ := decode(t, rec)["errors"].(map[string]any)
	return errs
}

// The whole life of an order in a dealer's hands, through the real handlers.
func TestADealerMovesCancelsAndDeletesAnOrderAgainstPostgres(t *testing.T) {
	e := newOrdersEnv(t)
	e.shoe("tc-order-a", map[string]int{"38": 5})
	id, token := e.place("dealer-key-0001", "tc-order-a", "38", 2)
	if e.pairs("tc-order-a", "38") != 3 {
		t.Fatal("the order did not take its pairs")
	}

	// SC-1: no session and a forged one change nothing and tell nothing.
	for _, cookie := range []*http.Cookie{nil, {Name: "session", Value: "forged"}} {
		if rec := e.call(http.MethodPut, "/v1/orders/"+id, `{"status":"confirmed"}`, cookie); rec.Code != 401 {
			t.Errorf("PUT without a session: %d", rec.Code)
		}
		if rec := e.call(http.MethodDelete, "/v1/orders/nope", "", cookie); rec.Code != 401 {
			t.Errorf("DELETE without a session: %d", rec.Code)
		}
	}

	// SC-5, SC-17, SC-18: refused bodies apply nothing.
	if rec := e.put(id, `{"status":"packed"}`); rec.Code != 422 || orderErrors(t, rec)["status"] != "An order cannot go from placed to packed." {
		t.Errorf("a skip: %d %s", rec.Code, rec.Body)
	}
	if rec := e.put(id, `{"status":"confirmed","total":1}`); rec.Code != 422 || orderErrors(t, rec)["total"] != "This cannot be changed." {
		t.Errorf("a frozen field: %d %s", rec.Code, rec.Body)
	}
	for _, body := range []string{`{"paid":true}`, `{"paid":false}`} {
		if rec := e.put(id, body); rec.Code != 422 || orderErrors(t, rec)["paid"] != "Only the payment provider marks this order paid." {
			t.Errorf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	if rec := e.put(id, `{"colour":"red"}`); rec.Code != 400 {
		t.Errorf("an unknown field: %d", rec.Code)
	}
	if rec := e.call(http.MethodGet, "/v1/orders/"+id, "", nil, "X-Order-Token", token); decode(t, rec)["status"] != "placed" {
		t.Fatalf("a refused body changed the order: %s", rec.Body)
	}
	if rec := e.put(id, `{}`); rec.Code != 200 {
		t.Errorf("an empty body: %d %s", rec.Code, rec.Body)
	}

	// SC-4, SC-12, SC-14: one step at a time; the courier and the customer's details are saved and the customer sees the courier's.
	if rec := e.put(id, `{"status":"confirmed"}`); rec.Code != 200 || decode(t, rec)["phone"] != "0912 345 678" {
		t.Errorf("confirmed: %d %s", rec.Code, rec.Body)
	}
	rec := e.put(id, `{"status":"packed","carrier":"GHN","tracking":"GHN-PT27XM","eta":"2026-11-26","address":"9 New Street","note":"Ring twice"}`)
	if v := decode(t, rec); rec.Code != 200 || v["status"] != "packed" || v["address"] != "9 New Street" || v["note"] != "Ring twice" || v["carrier"] != "GHN" {
		t.Errorf("packed: %d %s", rec.Code, rec.Body)
	}
	seen := decode(t, e.call(http.MethodGet, "/v1/orders/"+id, "", nil, "X-Order-Token", token))
	if seen["carrier"] != "GHN" || seen["tracking"] != "GHN-PT27XM" || seen["eta"] != "2026-11-26" || seen["address"] != nil {
		t.Errorf("the customer's page: %v", seen)
	}
	if rec := e.put(id, `{"eta":""}`); decode(t, rec)["eta"] != "" {
		t.Errorf("an empty date did not clear it: %s", rec.Body)
	}

	// SC-7, SC-9: cancelling puts the pairs back once.
	if rec := e.put(id, `{"status":"cancelled"}`); rec.Code != 200 || decode(t, rec)["status"] != "cancelled" || e.pairs("tc-order-a", "38") != 5 {
		t.Fatalf("cancelled: %d %s, %d pairs", rec.Code, rec.Body, e.pairs("tc-order-a", "38"))
	}
	if rec := e.put(id, `{"status":"cancelled"}`); rec.Code != 422 || orderErrors(t, rec)["status"] != "An order cannot go from cancelled to cancelled." || e.pairs("tc-order-a", "38") != 5 {
		t.Errorf("a second cancel: %d %s", rec.Code, rec.Body)
	}
	// SC-16: a cancelled order keeps its courier and customer, and its note stays open.
	if rec := e.put(id, `{"carrier":"GHTK","note":"after"}`); rec.Code != 422 || orderErrors(t, rec)["carrier"] == nil {
		t.Errorf("the courier of a cancelled order: %d %s", rec.Code, rec.Body)
	}
	if rec := e.put(id, `{"note":"after"}`); rec.Code != 200 || decode(t, rec)["note"] != "after" {
		t.Errorf("the note of a cancelled order: %d %s", rec.Code, rec.Body)
	}

	// SC-20 to SC-22: only a cancelled, unpaid order is deleted.
	other, _ := e.place("dealer-key-0002", "tc-order-a", "38", 1)
	if rec := e.call(http.MethodDelete, "/v1/orders/"+other, "", e.admin); rec.Code != 409 || decode(t, rec)["message"] != "Cancel the order first." {
		t.Errorf("deleting a placed order: %d %s", rec.Code, rec.Body)
	}
	e.put(other, `{"status":"cancelled"}`)
	e.count("UPDATE orders SET paid = true WHERE id = $1 RETURNING 1", other)
	if rec := e.call(http.MethodDelete, "/v1/orders/"+other, "", e.admin); rec.Code != 409 || !strings.HasPrefix(decode(t, rec)["message"].(string), "A paid order cannot be deleted") {
		t.Errorf("deleting a paid order: %d %s", rec.Code, rec.Body)
	}
	if rec := e.call(http.MethodDelete, "/v1/orders/"+id, "", e.admin); rec.Code != 204 || rec.Body.Len() != 0 {
		t.Errorf("deleting a cancelled order: %d %s", rec.Code, rec.Body)
	}
	if rec := e.call(http.MethodDelete, "/v1/orders/"+id, "", e.admin); rec.Code != 404 {
		t.Errorf("a second delete: %d", rec.Code)
	}
	if rec := e.call(http.MethodGet, "/v1/orders/"+id, "", nil, "X-Order-Token", token); rec.Code != 404 {
		t.Errorf("the old token: %d", rec.Code)
	}
	// Both orders were cancelled, so every pair is back, and no delete moved a count.
	if e.pairs("tc-order-a", "38") != 5 {
		t.Errorf("the delete moved the shelf: %d", e.pairs("tc-order-a", "38"))
	}
	// SC-23: a missing order is 404 even for a body that would be refused.
	if rec := e.put("doi-0", `{"total":1}`); rec.Code != 404 {
		t.Errorf("a missing order: %d %s", rec.Code, rec.Body)
	}
}
