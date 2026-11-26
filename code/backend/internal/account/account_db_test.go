package account

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/dbtest"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The whole customer story against PostgreSQL, through the real routes: auth,
// orders and account mounted as the server mounts them. Skipped without
// TEST_DATABASE_URL, as every database test is; the first run is CI.

type rig struct {
	t    *testing.T
	h    http.Handler
	pool *pgxpool.Pool
	svc  *auth.Service
}

func newRig(t *testing.T) *rig {
	t.Helper()
	pool := dbtest.Migrated(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := auth.New(pool, log)
	mux := http.NewServeMux()
	svc.Register(mux)
	orders.Register(mux, pool, log, orders.Options{})
	Register(mux, Deps{Auth: svc, DB: pool, Log: log})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, http.StatusNotFound, "not_found", "Not found.", nil)
	})
	return &rig{t: t, h: svc.Resolve()(mux), pool: pool, svc: svc}
}

type answer struct {
	code   int
	body   string
	header http.Header
	cookie string
}

// call sends one request. cookie is a session token, addr the client address.
func (g *rig) call(method, path, body, cookie, addr string, headers ...string) answer {
	g.t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: cookie})
	}
	if addr != "" {
		req.Header.Set("X-Forwarded-For", addr)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	g.h.ServeHTTP(rec, req)
	a := answer{code: rec.Code, body: rec.Body.String(), header: rec.Header()}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			a.cookie = c.Value
		}
	}
	return a
}

func (a answer) json(t *testing.T) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(a.body), &v); err != nil {
		t.Fatalf("not JSON: %q", a.body)
	}
	return v
}

func (g *rig) signUp(email, name, addr string) answer {
	g.t.Helper()
	return g.call("POST", "/v1/account/sign-up",
		fmt.Sprintf(`{"name":%q,"email":%q,"phone":"0912 345 678","password":"lanpass123"}`, name, email), "", addr)
}

func (g *rig) staffCookie() string {
	g.t.Helper()
	ctx := context.Background()
	hash, err := auth.HashPassword("staffpass1234")
	if err != nil {
		g.t.Fatal(err)
	}
	u, err := auth.CreateUser(ctx, g.pool, auth.NewUser{Email: "minh@example.com", Name: "Minh", PasswordHash: hash, Role: auth.Staff})
	if err != nil {
		g.t.Fatal(err)
	}
	token, err := g.svc.IssueSession(ctx, g.pool, u.ID, hash)
	if err != nil {
		g.t.Fatal(err)
	}
	return token
}

func (g *rig) text(sql string, args ...any) string {
	g.t.Helper()
	var s string
	if err := g.pool.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		g.t.Fatalf("%s: %v", sql, err)
	}
	return s
}

func TestSignUpSessionAndSavedDetailsAgainstPostgres(t *testing.T) {
	g := newRig(t)
	up := g.signUp("lan@example.com", "Lan Nguyen", "")
	body := up.json(t)
	if up.code != 201 || up.cookie == "" || len(body) != 3 || body["name"] != "Lan Nguyen" || body["email"] != "lan@example.com" || body["role"] != "customer" {
		t.Fatalf("sign-up: %d %s", up.code, up.body)
	}
	if me := g.call("GET", "/v1/auth/me", "", up.cookie, "").json(t); me["role"] != "customer" {
		t.Errorf("me: %v", me)
	}
	if hash := g.text(`SELECT password_hash FROM users WHERE email = 'lan@example.com'`); !strings.HasPrefix(hash, "$argon2id$") || strings.Contains(hash, "lanpass123") {
		t.Errorf("the password is not stored as a salted slow hash: %q", hash)
	}
	if strings.Contains(up.body, "argon2") || strings.Contains(up.body, "lanpass123") {
		t.Errorf("the answer holds the password or its hash: %s", up.body)
	}

	// An email held by any account, in any case, is a 409 naming the email.
	for _, email := range []string{"LAN@example.com", "lan@example.com"} {
		dup := g.signUp(email, "Other", "")
		if dup.code != 409 || dup.json(t)["errors"].(map[string]any)["email"] != "An account with this email already exists. Sign in instead." {
			t.Errorf("%s: %d %s", email, dup.code, dup.body)
		}
	}
	g.staffCookie()
	if dup := g.signUp("Minh@Example.com", "Minh", ""); dup.code != 409 {
		t.Errorf("a staff email: %d %s", dup.code, dup.body)
	}
	if n := g.text(`SELECT count(*)::text FROM users WHERE email = 'lan@example.com'`); n != "1" {
		t.Errorf("%s accounts for one email", n)
	}

	// The saved details: sign-up's values, empty address and city, then edits.
	got := g.call("GET", "/v1/account", "", up.cookie, "").json(t)
	if len(got) != 5 || got["phone"] != "0912 345 678" || got["address"] != "" || got["city"] != "" || got["email"] != "lan@example.com" {
		t.Errorf("GET /v1/account: %v", got)
	}
	put := g.call("PUT", "/v1/account", `{"phone":"+84 912 345 679"}`, up.cookie, "").json(t)
	if put["phone"] != "+84 912 345 679" || put["name"] != "Lan Nguyen" {
		t.Errorf("a phone-only PUT: %v", put)
	}
	if bad := g.call("PUT", "/v1/account", `{"name":"Changed","email":"x@example.com"}`, up.cookie, ""); bad.code != 422 {
		t.Errorf("an email in the body: %d", bad.code)
	}
	if name := g.text(`SELECT name FROM users WHERE email = 'lan@example.com'`); name != "Lan Nguyen" {
		t.Errorf("a refused body changed the name to %q", name)
	}
	saved := g.call("PUT", "/v1/account", `{"address":"12 Example Street, Ward 1","city":"Da Nang"}`, up.cookie, "").json(t)
	if saved["address"] != "12 Example Street, Ward 1" || saved["city"] != "Da Nang" || saved["phone"] != "+84 912 345 679" {
		t.Errorf("address and city: %v", saved)
	}
	if cleared := g.call("PUT", "/v1/account", `{"address":""}`, up.cookie, "").json(t); cleared["address"] != "" || cleared["city"] != "Da Nang" {
		t.Errorf("an empty address clears it: %v", cleared)
	}
	if empty := g.call("PUT", "/v1/account", `{}`, up.cookie, ""); empty.code != 200 {
		t.Errorf("{}: %d", empty.code)
	}

	// No session is 401 and a team session is 403, on every account route.
	staff := g.staffCookie2()
	for _, route := range []struct{ method, path string }{{"GET", "/v1/account"}, {"PUT", "/v1/account"}, {"GET", "/v1/account/orders"}, {"POST", "/v1/account/orders"}} {
		if a := g.call(route.method, route.path, `{}`, "", ""); a.code != 401 {
			t.Errorf("%s %s with no session: %d", route.method, route.path, a.code)
		}
		if a := g.call(route.method, route.path, `{}`, staff, ""); a.code != 403 {
			t.Errorf("%s %s as staff: %d", route.method, route.path, a.code)
		}
	}
}

// staffCookie2 signs the staff member in through the real sign-in route, so the
// second cookie of an account that already exists is the one a browser would hold.
func (g *rig) staffCookie2() string {
	g.t.Helper()
	a := g.call("POST", "/v1/auth/sign-in", `{"email":"minh@example.com","password":"staffpass1234"}`, "", "")
	if a.code != 200 || a.cookie == "" {
		g.t.Fatalf("staff sign-in: %d %s", a.code, a.body)
	}
	return a.cookie
}

func TestSignUpIsLimitedToTenAnHourPerAddressAgainstPostgres(t *testing.T) {
	g := newRig(t)
	for i := 1; i <= 10; i++ {
		if a := g.signUp(fmt.Sprintf("user%d@example.com", i), "User", "203.0.113.7"); a.code != 201 {
			t.Fatalf("sign-up %d: %d %s", i, a.code, a.body)
		}
	}
	a := g.signUp("user11@example.com", "User", "203.0.113.7")
	if a.code != 429 || a.header.Get("Retry-After") == "" || a.json(t)["message"] != "Too many tries. Please wait and try again." {
		t.Fatalf("the eleventh: %d %s %v", a.code, a.body, a.header)
	}
	if n := g.text(`SELECT count(*)::text FROM users WHERE email = 'user11@example.com'`); n != "0" {
		t.Error("a refused sign-up made an account")
	}
	if other := g.signUp("user12@example.com", "User", "203.0.113.8"); other.code != 201 {
		t.Errorf("another address: %d", other.code)
	}
	if _, err := g.pool.Exec(context.Background(), `UPDATE rate_limits SET window_started_at = now() - interval '61 minutes', locked_until = now() - interval '1 minute' WHERE scope = 'sign_up_address'`); err != nil {
		t.Fatal(err)
	}
	if again := g.signUp("user13@example.com", "User", "203.0.113.7"); again.code != 201 {
		t.Errorf("after the hour: %d %s", again.code, again.body)
	}
}

type seeded struct{ id, number, phone string }

func (g *rig) guestOrders(n int) []seeded {
	g.t.Helper()
	rows, err := g.pool.Query(context.Background(), `SELECT id, number, phone FROM orders ORDER BY placed_at, seq LIMIT $1`, n)
	if err != nil {
		g.t.Fatal(err)
	}
	defer rows.Close()
	var out []seeded
	for rows.Next() {
		var s seeded
		if err := rows.Scan(&s.id, &s.number, &s.phone); err != nil {
			g.t.Fatal(err)
		}
		out = append(out, s)
	}
	if len(out) != n {
		g.t.Fatalf("the seed holds %d orders, want %d", len(out), n)
	}
	return out
}

func pair(number, phone string) string {
	return fmt.Sprintf(`{"number":%q,"phone":%q}`, number, phone)
}

func TestMyOrdersAddAnOrderAndTheLookupLimitAgainstPostgres(t *testing.T) {
	g := newRig(t)
	const addrA, addrB = "203.0.113.20", "203.0.113.30"
	a := g.signUp("a@example.com", "Customer A", addrA).cookie
	b := g.signUp("b@example.com", "Customer B", addrB).cookie
	list := g.guestOrders(2)
	first, second := list[0], list[1]

	if empty := g.call("GET", "/v1/account/orders", "", a, addrA).json(t); empty["total"] != float64(0) || empty["limit"] != float64(10) || len(empty["items"].([]any)) != 0 {
		t.Errorf("no orders yet: %v", empty)
	}

	// A adds both orders: the number in any spelling, the phone as stored.
	typed := strings.ToLower(strings.Replace(first.number, "-", " ", 1))
	added := g.call("POST", "/v1/account/orders", pair(typed, first.phone), a, addrA)
	row := added.json(t)
	if added.code != 200 || len(row) != 5 || row["id"] != first.id || row["number"] != first.number {
		t.Fatalf("add: %d %s", added.code, added.body)
	}
	if again := g.call("POST", "/v1/account/orders", pair(first.number, first.phone), a, addrA); again.code != 200 {
		t.Errorf("adding it again: %d %s", again.code, again.body)
	}
	if g.call("POST", "/v1/account/orders", pair(second.number, second.phone), a, addrA).code != 200 {
		t.Fatal("the second order was not added")
	}
	mine := g.call("GET", "/v1/account/orders", "", a, addrA)
	page := mine.json(t)
	items := page["items"].([]any)
	if page["total"] != float64(2) || len(items) != 2 {
		t.Fatalf("my orders: %s", mine.body)
	}
	newest, oldest := items[0].(map[string]any), items[1].(map[string]any)
	if len(newest) != 5 || !strings.HasSuffix(newest["placed_at"].(string), "+07:00") || newest["placed_at"].(string) < oldest["placed_at"].(string) {
		t.Errorf("an item or the order of items: %s", mine.body)
	}
	for _, leak := range []string{`"phone"`, `"email"`, `"address"`, `"note"`, `"account_id"`} {
		if strings.Contains(mine.body, leak) {
			t.Errorf("the list holds %s: %s", leak, mine.body)
		}
	}
	if paged := g.call("GET", "/v1/account/orders?page=2&limit=1", "", a, addrA).json(t); paged["page"] != float64(2) || len(paged["items"].([]any)) != 1 {
		t.Errorf("page 2 of 1: %v", paged)
	}

	// The linked customer opens the order with no token; no one else does, and
	// every way of not being let in is the same bytes.
	own := g.call("GET", "/v1/account/orders/"+first.id, "", a, addrA)
	viaOrders := g.call("GET", "/v1/orders/"+first.id, "", a, addrA)
	if own.code != 200 || viaOrders.code != 200 || own.body != viaOrders.body {
		t.Errorf("the linked customer: %d / %d", own.code, viaOrders.code)
	}
	for _, leak := range []string{`"phone"`, `"email"`, `"address"`, `"note"`} {
		if strings.Contains(own.body, leak) {
			t.Errorf("the order view holds %s", leak)
		}
	}
	refusals := []answer{
		g.call("GET", "/v1/account/orders/"+first.id, "", b, addrB),
		g.call("GET", "/v1/orders/"+first.id, "", b, addrB),
		g.call("GET", "/v1/orders/"+first.id, "", "", addrB),
		g.call("GET", "/v1/orders/"+first.id, "", "", addrB, "X-Order-Token", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		g.call("GET", "/v1/account/orders/doi-0", "", b, addrB),
	}
	for i, r := range refusals {
		if r.code != 404 || r.body != refusals[0].body {
			t.Errorf("refusal %d: %d %s", i, r.code, r.body)
		}
	}

	// B cannot add A's order: the same 404 as a wrong phone and an unknown number.
	notFound := g.call("POST", "/v1/account/orders", pair(first.number, first.phone), b, addrB)
	wrong := g.call("POST", "/v1/account/orders", pair(second.number, "0999 999 999"), b, addrB)
	unknown := g.call("POST", "/v1/account/orders", pair("DOI-1", "0999 999 999"), b, addrB)
	empty := g.call("POST", "/v1/account/orders", pair(second.number, ""), b, addrB)
	for i, r := range []answer{notFound, wrong, unknown, empty} {
		if r.code != 404 || r.body != notFound.body || r.json(t)["message"] != "No order matches that number and phone." {
			t.Errorf("miss %d: %d %s", i, r.code, r.body)
		}
	}
	if n := g.text(`SELECT count(*)::text FROM orders WHERE account_id IS NOT NULL`); n != "2" {
		t.Errorf("%s orders are linked, want 2", n)
	}

	// A fifth miss locks the address; the sixth try, with a right pair, is a 429
	// that links nothing.
	g.call("POST", "/v1/account/orders", pair(second.number, "0999 999 998"), b, addrB)
	limited := g.call("POST", "/v1/account/orders", pair(second.number, second.phone), b, addrB)
	if limited.code != 429 || limited.header.Get("Retry-After") == "" || !strings.HasPrefix(limited.json(t)["message"].(string), "Too many tries. Please wait ") {
		t.Fatalf("the sixth try: %d %s %v", limited.code, limited.body, limited.header)
	}
	if owner := g.text(`SELECT coalesce(account_id::text, '') FROM orders WHERE id = $1`, second.id); owner != g.text(`SELECT id::text FROM users WHERE email = 'a@example.com'`) {
		t.Errorf("a refused try moved the order to %q", owner)
	}
}

func TestPlacingAnOrderLinksOnlyACustomerAgainstPostgres(t *testing.T) {
	g := newRig(t)
	var product, size string
	err := g.pool.QueryRow(context.Background(), `SELECT s.product_id, s.size FROM product_sizes s JOIN products p ON p.id = s.product_id
		WHERE p.status = 'live' AND s.pairs > 0 ORDER BY s.product_id, s.size LIMIT 1`).Scan(&product, &size)
	if err != nil {
		t.Fatalf("no shoe to order: %v", err)
	}
	// The seed shelves at most 3 pairs of a size and this test sells three, so
	// the shelf is filled up first: the test is about who the order is linked to,
	// never about stock.
	if _, err := g.pool.Exec(context.Background(), `UPDATE product_sizes SET pairs = 10 WHERE product_id = $1 AND size = $2`, product, size); err != nil {
		t.Fatalf("fill the shelf: %v", err)
	}
	place := func(cookie, key, addr string) map[string]any {
		t.Helper()
		body := fmt.Sprintf(`{"items":[{"product":%q,"size":%q,"qty":1}],"delivery":"standard","payment":"vnpay",
			"customer":"Typed Name","phone":"0912 345 678","address":"1 Typed Street","city":"Typed City","note":""}`, product, size)
		a := g.call("POST", "/v1/orders", body, cookie, addr, "Idempotency-Key", key)
		if a.code != 201 && a.code != 200 {
			t.Fatalf("place: %d %s", a.code, a.body)
		}
		return a.json(t)["order"].(map[string]any)
	}
	owner := func(order map[string]any) string {
		return g.text(`SELECT coalesce(account_id::text, '') FROM orders WHERE id = $1`, order["id"])
	}

	customer := g.signUp("lan@example.com", "Lan Nguyen", "203.0.113.40").cookie
	lan := g.text(`SELECT id::text FROM users WHERE email = 'lan@example.com'`)
	staff := g.staffCookie()
	if got := owner(place(customer, "customer-key-0001", "203.0.113.40")); got != lan {
		t.Errorf("a customer's order is linked to %q, want %q", got, lan)
	}
	if got := g.call("GET", "/v1/account", "", customer, "").json(t); got["name"] != "Lan Nguyen" || got["address"] != "" {
		t.Errorf("placing an order changed the saved details: %v", got)
	}
	guest := place("", "guest-key-0000001", "203.0.113.41")
	if got := owner(guest); got != "" {
		t.Errorf("a guest order is linked to %q", got)
	}
	if got := owner(place(staff, "staff-key-0000001", "203.0.113.42")); got != "" {
		t.Errorf("a staff order is linked to %q", got)
	}
	// A replay with a customer's session never links the guest's order.
	g.call("POST", "/v1/orders", fmt.Sprintf(`{"items":[{"product":%q,"size":%q,"qty":1}],"delivery":"standard","payment":"vnpay",
		"customer":"Typed Name","phone":"0912 345 678","address":"1 Typed Street","city":"Typed City","note":""}`, product, size),
		customer, "203.0.113.40", "Idempotency-Key", "guest-key-0000001")
	if got := owner(guest); got != "" {
		t.Errorf("a replay linked the guest order to %q", got)
	}
}
