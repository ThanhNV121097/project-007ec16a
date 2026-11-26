package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/dbtest"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests run the real SQL of internal/auth against PostgreSQL, through the
// real handlers and the real sign-in. They read TEST_DATABASE_URL, as CI's
// backend job sets it, and skip without it.

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	svc    *auth.Service
	h      http.Handler
	admin  *http.Cookie
	mu     sync.Mutex
	bodies strings.Builder
}

var addrs atomic.Int64

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := dbtest.Migrated(t)
	if err := auth.EnsureAdmin(context.Background(), pool, "dealer@example.com", "Dealer-Pass1", quiet()); err != nil {
		t.Fatal(err)
	}
	svc := auth.New(pool, quiet())
	mux := http.NewServeMux()
	svc.Register(mux)
	RegisterStaff(mux, pool, quiet())
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	e := &env{t: t, pool: pool, svc: svc, h: svc.Resolve()(httpx.RecordRoute(mux))}
	code, cookie := e.signIn("dealer@example.com", "Dealer-Pass1")
	if code != 200 {
		t.Fatalf("the admin cannot sign in: %d", code)
	}
	e.admin = cookie
	return e
}

func (e *env) do(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.RemoteAddr = fmt.Sprintf("192.0.2.%d:4000", 1+addrs.Add(1)%250)
	if cookie != nil {
		req.AddCookie(&http.Cookie{Name: "session", Value: cookie.Value})
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	e.mu.Lock()
	e.bodies.WriteString(rec.Body.String())
	e.mu.Unlock()
	return rec
}

// signIn answers the status and the session cookie (nil when there is none).
func (e *env) signIn(email, password string) (int, *http.Cookie) {
	e.t.Helper()
	b, _ := json.Marshal(map[string]string{"email": email, "password": password})
	rec := e.do(http.MethodPost, "/v1/auth/sign-in", string(b), nil)
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			return rec.Code, c
		}
	}
	return rec.Code, nil
}

func (e *env) count(sql string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func (e *env) add(name, email, password string) map[string]any {
	e.t.Helper()
	b, _ := json.Marshal(map[string]string{"name": name, "email": email, "password": password})
	rec := e.do(http.MethodPost, "/v1/admin/staff", string(b), e.admin)
	if rec.Code != 201 {
		e.t.Fatalf("add %s: %d %s", email, rec.Code, rec.Body)
	}
	return decode(e.t, rec)
}

func (e *env) meStatus(c *http.Cookie) int { return e.do(http.MethodGet, "/v1/auth/me", "", c).Code }

// leftover is the invariant of SC-27, SC-29, SC-30 and SC-32: no session row
// belongs to a switched-off or removed account.
func (e *env) leftover() int {
	return e.count(`SELECT count(*) FROM sessions s LEFT JOIN users u ON u.id = s.user_id WHERE u.id IS NULL OR NOT u.is_active`)
}

// T3 and T5: the whole life of an account, against the real SQL.
func TestAStaffAccountLivesAndDiesAgainstPostgres(t *testing.T) {
	e := newEnv(t)
	hash, err := auth.HashPassword("Customer-Pass-1")
	if err != nil {
		t.Fatal(err)
	}
	cust, err := auth.CreateUser(context.Background(), e.pool, auth.NewUser{Email: "customer@example.com", Name: "Customer", Role: auth.Customer, PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}

	lan := e.add(" Lan Tran ", "Lan@Example.com", "lan-password-1")
	e.add("Mai Le", "mai@example.com", "lan-password-1")
	if lan["email"] != "lan@example.com" || lan["name"] != "Lan Tran" || lan["role"] != "staff" || lan["is_active"] != true {
		t.Fatalf("the new account: %v", lan)
	}
	wantAccount(t, lan)
	if n := e.count(`SELECT count(DISTINCT password_hash) FROM users WHERE email IN ('lan@example.com', 'mai@example.com') AND password_hash LIKE '$argon2id$%'`); n != 2 {
		t.Errorf("two accounts with one password must hold two different argon2id hashes, got %d", n)
	}

	// The list: the admin first, then by creation; customers never.
	list := decode(t, e.do(http.MethodGet, "/v1/admin/staff?limit=100", "", e.admin))
	items := list["items"].([]any)
	if len(items) != 3 || list["total"] != float64(3) {
		t.Fatalf("list: %v", list)
	}
	for i, want := range []string{"dealer@example.com", "lan@example.com", "mai@example.com"} {
		if got := items[i].(map[string]any)["email"]; got != want {
			t.Errorf("row %d is %v, want %s", i, got, want)
		}
	}
	if items[0].(map[string]any)["name"] != "Dealer" {
		t.Errorf("the admin's display name: %v", items[0])
	}
	page2 := decode(t, e.do(http.MethodGet, "/v1/admin/staff?page=2&limit=1", "", e.admin))
	if len(page2["items"].([]any)) != 1 || page2["total"] != float64(3) || page2["items"].([]any)[0].(map[string]any)["email"] != "lan@example.com" {
		t.Errorf("page 2: %v", page2)
	}

	// The new account signs in at once, and the staff endpoints refuse it.
	code, lanCookie := e.signIn("lan@example.com", "lan-password-1")
	if code != 200 || lanCookie == nil {
		t.Fatalf("the new account cannot sign in: %d", code)
	}
	me := decode(t, e.do(http.MethodGet, "/v1/auth/me", "", lanCookie))
	if me["role"] != "staff" || me["name"] != "Lan Tran" {
		t.Errorf("me: %v", me)
	}
	if code := e.do(http.MethodGet, "/v1/admin/staff", "", lanCookie).Code; code != 403 {
		t.Errorf("staff on the staff list: %d", code)
	}
	_, custCookie := e.signIn("customer@example.com", "Customer-Pass-1")
	if code := e.do(http.MethodGet, "/v1/admin/staff", "", custCookie).Code; code != 403 {
		t.Errorf("a customer on the staff list: %d", code)
	}
	if code := e.do(http.MethodGet, "/v1/admin/staff", "", nil).Code; code != 401 {
		t.Errorf("nobody on the staff list: %d", code)
	}

	// A name alone ends no session.
	lanURL := "/v1/admin/staff/" + lan["id"].(string)
	if rec := e.do(http.MethodPut, lanURL, `{"name":"Lan T."}`, e.admin); rec.Code != 200 || decode(t, rec)["name"] != "Lan T." {
		t.Fatalf("rename: %d %s", rec.Code, rec.Body)
	}
	if code := e.meStatus(lanCookie); code != 200 {
		t.Errorf("a rename ended the session: %d", code)
	}

	// A reset ends the sessions, and only the new password signs in.
	if rec := e.do(http.MethodPut, lanURL, `{"password":"brand-new-pass-2"}`, e.admin); rec.Code != 200 {
		t.Fatalf("reset: %d %s", rec.Code, rec.Body)
	}
	if code := e.meStatus(lanCookie); code != 401 {
		t.Errorf("the session before a reset answers %d", code)
	}
	if code, _ := e.signIn("lan@example.com", "lan-password-1"); code != 401 {
		t.Errorf("the old password: %d", code)
	}
	code, lanCookie = e.signIn("lan@example.com", "brand-new-pass-2")
	if code != 200 {
		t.Fatalf("the new password: %d", code)
	}

	// Switching off ends the sessions and the sign-in refuses like a wrong password.
	if rec := e.do(http.MethodPut, lanURL, `{"is_active":false}`, e.admin); rec.Code != 200 || decode(t, rec)["is_active"] != false {
		t.Fatalf("switch off: %d %s", rec.Code, rec.Body)
	}
	if code := e.meStatus(lanCookie); code != 401 {
		t.Errorf("a switched-off account's cookie answers %d", code)
	}
	right := e.do(http.MethodPost, "/v1/auth/sign-in", `{"email":"lan@example.com","password":"brand-new-pass-2"}`, nil)
	wrong := e.do(http.MethodPost, "/v1/auth/sign-in", `{"email":"lan@example.com","password":"not-the-password"}`, nil)
	if right.Code != 401 || right.Code != wrong.Code || right.Body.String() != wrong.Body.String() {
		t.Errorf("a switched-off account is not told apart from a wrong password: %d %s / %d %s", right.Code, right.Body, wrong.Code, wrong.Body)
	}
	if n := e.leftover(); n != 0 {
		t.Errorf("%d sessions belong to switched-off accounts", n)
	}

	// Switching on brings no old session back.
	if rec := e.do(http.MethodPut, lanURL, `{"is_active":true}`, e.admin); rec.Code != 200 {
		t.Fatalf("switch on: %d %s", rec.Code, rec.Body)
	}
	if code := e.meStatus(lanCookie); code != 401 {
		t.Errorf("the cookie from before the switch-off answers %d", code)
	}
	code, lanCookie = e.signIn("lan@example.com", "brand-new-pass-2")
	if code != 200 {
		t.Fatalf("sign-in after switch on: %d", code)
	}

	// The admin is never changed or removed; a customer is not found.
	adminURL := "/v1/admin/staff/" + items[0].(map[string]any)["id"].(string)
	for _, c := range []struct{ method, body string }{
		{http.MethodPut, `{"is_active":false}`}, {http.MethodPut, `{"name":"x"}`}, {http.MethodPut, `{`}, {http.MethodDelete, ""},
	} {
		if rec := e.do(c.method, adminURL, c.body, e.admin); rec.Code != 409 || decode(t, rec)["message"] != adminSentence {
			t.Errorf("%s the admin %q: %d %s", c.method, c.body, rec.Code, rec.Body)
		}
	}
	if n := e.count(`SELECT count(*) FROM users WHERE role = 'admin' AND is_active`); n != 1 || e.meStatus(e.admin) != 200 {
		t.Errorf("the admin was changed: %d", n)
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		body := ""
		if method == http.MethodPut {
			body = `{"is_active":false}`
		}
		if rec := e.do(method, "/v1/admin/staff/"+cust.ID, body, e.admin); rec.Code != 404 {
			t.Errorf("%s a customer: %d", method, rec.Code)
		}
	}
	if n := e.count(`SELECT count(*) FROM users WHERE id = $1 AND is_active AND role = 'customer'`, cust.ID); n != 1 {
		t.Error("the customer was changed")
	}

	// Removing frees the email, ends the session and touches nothing else.
	if rec := e.do(http.MethodDelete, lanURL, "", e.admin); rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatalf("remove: %d %s", rec.Code, rec.Body)
	}
	if code := e.meStatus(lanCookie); code != 401 {
		t.Errorf("a removed account's cookie answers %d", code)
	}
	if n := e.count(`SELECT count(*) FROM users WHERE id = $1`, lan["id"]) + e.count(`SELECT count(*) FROM sessions WHERE user_id = $1`, lan["id"]); n != 0 {
		t.Errorf("the removed account left %d rows", n)
	}
	if rec := e.do(http.MethodDelete, lanURL, "", e.admin); rec.Code != 404 {
		t.Errorf("a second DELETE: %d", rec.Code)
	}
	again := e.add("Lan Tran", "lan@example.com", "another-pass-3")
	if again["id"] == lan["id"] {
		t.Error("a new account must have a new id")
	}
	if n := e.leftover(); n != 0 {
		t.Errorf("%d sessions belong to switched-off or removed accounts", n)
	}

	if all := e.bodies.String(); strings.Contains(all, "argon2") || strings.Contains(all, "password_hash") {
		t.Error("a hash reached a response")
	}
}

// T4: two POSTs of one email at the same moment make one account.
func TestTwoAddsOfOneEmailMakeOneAccountAgainstPostgres(t *testing.T) {
	e := newEnv(t)
	for round := range 15 {
		email := fmt.Sprintf("race%d@example.com", round)
		body := fmt.Sprintf(`{"name":"Racer","email":%q,"password":"a-long-password"}`, email)
		recs := make([]*httptest.ResponseRecorder, 2)
		var wg sync.WaitGroup
		for i := range recs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				recs[i] = e.do(http.MethodPost, "/v1/admin/staff", body, e.admin)
			}()
		}
		wg.Wait()
		created, refused := 0, 0
		for _, rec := range recs {
			switch {
			case rec.Code == 201:
				created++
			case rec.Code == 422 && errorsOf(t, rec)["email"] != nil:
				refused++
			default:
				t.Fatalf("round %d: %d %s", round, rec.Code, rec.Body)
			}
		}
		if created != 1 || refused != 1 || e.count(`SELECT count(*) FROM users WHERE email = $1`, email) != 1 {
			t.Fatalf("round %d: %d created, %d refused", round, created, refused)
		}
	}
}

// T6: a sign-in racing a switch-off, a reset or a removal never leaves a live
// session behind. Rounds are repeated; the invariant is asserted after each.
func TestSessionsNeverOutliveASwitchOffResetOrRemovalAgainstPostgres(t *testing.T) {
	e := newEnv(t)
	hash, err := auth.HashPassword("Race-Password-1")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for round := range 25 {
		for i, change := range []string{`{"is_active":false}`, `{"password":"Reset-Password-2"}`, ""} {
			u, err := auth.CreateUser(ctx, e.pool, auth.NewUser{Email: fmt.Sprintf("r%d-%d@example.com", round, i), Name: "R", Role: auth.Staff, PasswordHash: hash})
			if err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				_, _ = e.svc.IssueSession(ctx, e.pool, u.ID, hash) // refused or made: both are allowed
			}()
			go func() {
				defer wg.Done()
				if change == "" {
					e.do(http.MethodDelete, "/v1/admin/staff/"+u.ID, "", e.admin)
				} else {
					e.do(http.MethodPut, "/v1/admin/staff/"+u.ID, change, e.admin)
				}
			}()
			wg.Wait()
			if n := e.leftover(); n != 0 {
				t.Fatalf("round %d, change %q: %d sessions belong to an account that is off or gone", round, change, n)
			}
			if i == 1 {
				if n := e.count(`SELECT count(*) FROM sessions WHERE user_id = $1`, u.ID); n != 0 {
					t.Fatalf("round %d: a session survived a password reset", round)
				}
			}
		}
	}
}

// T7: the schema this story stands on (ERD §6).
func TestStaffSchemaAssumptions(t *testing.T) {
	e := newEnv(t)
	def := func(index string) string {
		var d string
		if err := e.pool.QueryRow(context.Background(), `SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND indexname = $1`, index).Scan(&d); err != nil {
			t.Fatalf("index %s: %v", index, err)
		}
		return d
	}
	if d := def("uq_users_email"); !strings.Contains(d, "UNIQUE") || !strings.Contains(d, "(email)") {
		t.Errorf("uq_users_email: %s", d)
	}
	if d := def("uq_users_one_admin"); !strings.Contains(d, "UNIQUE") || !strings.Contains(d, "role = 'admin'") {
		t.Errorf("uq_users_one_admin: %s", d)
	}
	def("idx_sessions_user_id")
	if n := e.count(`SELECT count(*) FROM pg_constraint WHERE conname IN ('ck_users_email_lowercase', 'ck_users_role') AND conrelid = to_regclass('users')`); n != 2 {
		t.Errorf("the users CHECK constraints: %d of 2", n)
	}
	onDelete := func(table, constraint string) string {
		var c string
		if err := e.pool.QueryRow(context.Background(), `SELECT confdeltype::text FROM pg_constraint WHERE conname = $1 AND conrelid = to_regclass($2)`, constraint, table).Scan(&c); err != nil {
			t.Fatalf("%s: %v", constraint, err)
		}
		return c
	}
	if c := onDelete("sessions", "fk_sessions_user_id"); c != "c" {
		t.Errorf("sessions.user_id must cascade, deletion type %q", c)
	}
	// Tables other stories add: checked when they exist.
	if e.count(`SELECT count(*) FROM pg_tables WHERE schemaname = current_schema() AND tablename = 'uploads'`) == 1 {
		var c string
		err := e.pool.QueryRow(context.Background(), `SELECT confdeltype::text FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
			WHERE c.contype = 'f' AND c.conrelid = to_regclass('uploads') AND c.confrelid = to_regclass('users') AND a.attname = 'uploaded_by'`).Scan(&c)
		if err != nil || c != "n" {
			t.Errorf("uploads.uploaded_by must be ON DELETE SET NULL: %q %v", c, err)
		}
	}
	if e.count(`SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'orders' AND column_name = 'account_id'`) == 1 {
		var c string
		err := e.pool.QueryRow(context.Background(), `SELECT confdeltype::text FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
			WHERE c.contype = 'f' AND c.conrelid = to_regclass('orders') AND c.confrelid = to_regclass('users') AND a.attname = 'account_id'`).Scan(&c)
		if err != nil || c != "n" {
			t.Errorf("orders.account_id must be ON DELETE SET NULL: %q %v", c, err)
		}
	}
}
