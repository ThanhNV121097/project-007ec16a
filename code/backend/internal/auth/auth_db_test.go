package auth

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/dbtest"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests run the real migration and the real SQL against PostgreSQL. They
// read TEST_DATABASE_URL, as CI's backend job sets it, and skip without it.

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func countRows(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func mustUser(t *testing.T, pool *pgxpool.Pool, email, password string, role Role) User {
	t.Helper()
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	u, err := CreateUser(context.Background(), pool, NewUser{Email: email, Name: strings.ToUpper(string(role)[:1]) + string(role)[1:], Role: role, PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// SC-19: the tables keep the rules of the design.
func TestUsersAndSessionsRefuseWhatTheDesignRefuses(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	ok := NewUser{Email: "a@example.com", Role: Admin, PasswordHash: "h"}
	u, err := CreateUser(ctx, pool, ok)
	if err != nil || !u.Active || u.ID == "" {
		t.Fatalf("%+v %v", u, err)
	}
	if _, err := CreateUser(ctx, pool, NewUser{Email: "b@example.com", Role: Admin, PasswordHash: "h"}); constraintOf(err) != "uq_users_one_admin" {
		t.Errorf("a second admin: %v", err)
	}
	if _, err := CreateUser(ctx, pool, NewUser{Email: "a@example.com", Role: Staff, PasswordHash: "h"}); err != ErrEmailTaken {
		t.Errorf("a taken address: %v", err)
	}
	for name, n := range map[string]NewUser{
		"upper case email": {Email: "C@Example.com", Role: Staff, PasswordHash: "h"},
		"non-ASCII email":  {Email: "dôi@example.com", Role: Staff, PasswordHash: "h"},
		"unknown role":     {Email: "e@example.com", Role: "owner", PasswordHash: "h"},
		"blank hash":       {Email: "f@example.com", Role: Staff, PasswordHash: ""},
	} {
		if _, err := CreateUser(ctx, pool, n); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, '\x00'::bytea, now() + interval '1 day')`, u.ID); err == nil {
		t.Error("a token hash of one byte was accepted")
	}
	mustExec(t, pool, `INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, sha256('t'::bytea), now() + interval '1 day')`, u.ID)
	if _, err := pool.Exec(ctx, `INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, sha256('t'::bytea), now() + interval '1 day')`, u.ID); err == nil {
		t.Error("a token hash was stored twice")
	}
	mustExec(t, pool, `DELETE FROM users WHERE id = $1`, u.ID)
	if n := countRows(t, pool, `SELECT count(*) FROM sessions`); n != 0 {
		t.Errorf("deleting the user left %d sessions", n)
	}
}

// SC-1 to SC-5: the admin made from the settings.
func TestEnsureAdminMakesChangesAndLeavesAnUnchangedRowAlone(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	adminRow := func() (email, hash string, active bool, updated string) {
		if err := pool.QueryRow(ctx, `SELECT email, password_hash, is_active, updated_at::text FROM users WHERE role = 'admin'`).Scan(&email, &hash, &active, &updated); err != nil {
			t.Fatal(err)
		}
		return
	}
	session := func() {
		mustExec(t, pool, `INSERT INTO sessions (user_id, token_hash, expires_at)
			SELECT id, sha256(convert_to(gen_random_uuid()::text, 'UTF8')), now() + interval '1 day' FROM users WHERE role = 'admin'`)
	}

	if err := EnsureAdmin(ctx, pool, "Dealer@Example.com", "Dealer-Pass1", log); err != nil {
		t.Fatal(err)
	}
	email, hash, active, updated := adminRow()
	if email != "dealer@example.com" || !strings.HasPrefix(hash, "$argon2id$") || !active || strings.Contains(hash, "Dealer-Pass1") {
		t.Fatalf("made: %q %q %v", email, hash, active)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM users WHERE role = 'admin' AND name = ''`); n != 1 {
		t.Error("the admin's name is not the empty string")
	}

	session()
	if err := EnsureAdmin(ctx, pool, "dealer@example.com", "Dealer-Pass1", log); err != nil {
		t.Fatal(err)
	}
	if e2, h2, _, u2 := adminRow(); e2 != email || h2 != hash || u2 != updated {
		t.Error("an unchanged setting wrote the row")
	}
	if n := countRows(t, pool, `SELECT count(*) FROM sessions`); n != 1 {
		t.Errorf("an unchanged setting ended the sessions: %d left", n)
	}

	if err := EnsureAdmin(ctx, pool, "dealer@example.com", "Dealer-Pass2-New", log); err != nil {
		t.Fatal(err)
	}
	_, h3, _, _ := adminRow()
	if h3 == hash {
		t.Error("the password was not replaced")
	}
	if match, _, _ := check(ctx, h3, "Dealer-Pass1"); match {
		t.Error("the old password still verifies")
	}
	if match, _, _ := check(ctx, h3, "Dealer-Pass2-New"); !match {
		t.Error("the new password does not verify")
	}
	if n := countRows(t, pool, `SELECT count(*) FROM sessions`); n != 0 {
		t.Errorf("a changed password left %d sessions", n)
	}

	session()
	if err := EnsureAdmin(ctx, pool, "lan.tran@example.com", "Dealer-Pass2-New", log); err != nil {
		t.Fatal(err)
	}
	if e4, _, _, _ := adminRow(); e4 != "lan.tran@example.com" || countRows(t, pool, `SELECT count(*) FROM users WHERE role = 'admin'`) != 1 {
		t.Errorf("a changed email: %q", e4)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM sessions`); n != 0 {
		t.Errorf("a changed email left %d sessions", n)
	}

	_, before, _, _ := adminRow()
	logs.Reset()
	if err := EnsureAdmin(ctx, pool, "lan.tran@example.com", "Short-Pw9", log); err != nil {
		t.Fatal(err)
	}
	if err := EnsureAdmin(ctx, pool, "", "", log); err != nil {
		t.Fatal(err)
	}
	if _, after, _, _ := adminRow(); after != before || !strings.Contains(logs.String(), "too short") || strings.Contains(logs.String(), "Short-Pw9") {
		t.Errorf("a short password or an unset setting changed the row, or the log is wrong: %s", logs.String())
	}

	mustExec(t, pool, `UPDATE users SET is_active = false WHERE role = 'admin'`)
	if err := EnsureAdmin(ctx, pool, "lan.tran@example.com", "Dealer-Pass2-New", log); err != nil {
		t.Fatal(err)
	}
	if _, _, active, _ := adminRow(); !active {
		t.Error("a switched-off admin was not switched on again")
	}

	mustUser(t, pool, "taken@example.com", "Customer-Pass-1", Customer)
	if err := EnsureAdmin(ctx, pool, "taken@example.com", "Another-Pass-1", log); err != nil {
		t.Fatalf("an address another account holds must be skipped, not fail: %v", err)
	}
	if e5, _, _, _ := adminRow(); e5 != "lan.tran@example.com" {
		t.Errorf("the admin took another account's address: %q", e5)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM users WHERE email = 'taken@example.com' AND role = 'customer'`); n != 1 {
		t.Error("the customer's row was changed")
	}
}

// R2 of the design: Q3 inserts a session only for an account that is still
// active with the hash the password was verified against, and the statement
// (INSERT ... SELECT ... FOR SHARE OF u) is accepted by Postgres.
func TestIssueSessionInsertsNothingForAnAccountThatChanged(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	svc := New(pool, discard())
	u := mustUser(t, pool, "staff@example.com", "Staff-Pass-1", Staff)
	var hash string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, u.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}

	token, err := svc.IssueSession(ctx, pool, u.ID, hash)
	if err != nil || len(token) != 43 {
		t.Fatalf("a good account: %q %v", token, err)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM sessions WHERE user_id = $1 AND expires_at = created_at + interval '7 days'`, u.ID); n != 1 {
		t.Errorf("the session is not seven days long: %d", n)
	}
	hashed, _ := tokenHash(token)
	if n := countRows(t, pool, `SELECT count(*) FROM sessions WHERE token_hash = $1`, hashed); n != 1 {
		t.Error("the stored hash is not the SHA-256 of the token")
	}

	mustExec(t, pool, `DELETE FROM sessions`)
	mustExec(t, pool, `UPDATE users SET is_active = false WHERE id = $1`, u.ID)
	if _, err := svc.IssueSession(ctx, pool, u.ID, hash); err != ErrSessionRefused {
		t.Errorf("a switched-off account: %v", err)
	}
	mustExec(t, pool, `UPDATE users SET is_active = true, password_hash = 'a-new-hash' WHERE id = $1`, u.ID)
	if _, err := svc.IssueSession(ctx, pool, u.ID, hash); err != ErrSessionRefused {
		t.Errorf("an account given a new password: %v", err)
	}
	if _, err := svc.IssueSession(ctx, pool, "00000000-0000-0000-0000-000000000000", hash); err != ErrSessionRefused {
		t.Errorf("an account that does not exist: %v", err)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM sessions`); n != 0 {
		t.Errorf("%d sessions were made for changed accounts", n)
	}
}

// SC-1, SC-3, SC-6, SC-9, SC-11, SC-13 to SC-17 against the real SQL.
func TestSignInSessionsAndGuardsAgainstPostgres(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	if err := EnsureAdmin(ctx, pool, "dealer@example.com", "Dealer-Pass1", discard()); err != nil {
		t.Fatal(err)
	}
	staff := mustUser(t, pool, "staff@example.com", "Staff-Pass-1", Staff)
	mustUser(t, pool, "customer@example.com", "Customer-Pass-1", Customer)

	svc := New(pool, discard())
	mux := http.NewServeMux()
	svc.Register(mux)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, http.StatusOK, map[string]string{}) })
	mux.Handle("GET /v1/team", Require(Admin, Staff)(ok))
	h := svc.Resolve()(httpx.RecordRoute(mux))

	do := func(method, target, body, addr string, c *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.RemoteAddr = addr
		if c != nil {
			req.AddCookie(&http.Cookie{Name: "session", Value: c.Value})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	const addr = "192.0.2.10:4000"
	signIn := func(email, password string) *httptest.ResponseRecorder {
		return do(http.MethodPost, "/v1/auth/sign-in", creds(email, password), addr, nil)
	}

	in := signIn(" Dealer@Example.com ", "Dealer-Pass1")
	if in.Code != http.StatusOK || strings.TrimSpace(in.Body.String()) != `{"email":"dealer@example.com","name":"Dealer","role":"admin"}` {
		t.Fatalf("admin sign-in: %d %s", in.Code, in.Body)
	}
	admin := sessionCookie(in)
	if me := do(http.MethodGet, "/v1/auth/me", "", addr, admin); me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"role":"admin"`) {
		t.Errorf("me: %d %s", me.Code, me.Body)
	}
	if rec := signIn("staff@example.com", "Staff-Pass-1"); !strings.Contains(rec.Body.String(), `"name":"Staff","role":"staff"`) {
		t.Errorf("staff sign-in: %d %s", rec.Code, rec.Body)
	}
	customer := sessionCookie(signIn("customer@example.com", "Customer-Pass-1"))
	if rec := do(http.MethodGet, "/v1/team", "", addr, customer); rec.Code != http.StatusForbidden {
		t.Errorf("a customer on a team route: %d", rec.Code)
	}
	if rec := do(http.MethodGet, "/v1/team", "", addr, admin); rec.Code != http.StatusOK {
		t.Errorf("the admin on a team route: %d", rec.Code)
	}

	// SC-11: two sign-ins at once, two cookies, two more rows.
	before := countRows(t, pool, `SELECT count(*) FROM sessions`)
	var wg sync.WaitGroup
	got := make([]*httptest.ResponseRecorder, 2)
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[i] = signIn("dealer@example.com", "Dealer-Pass1")
		}()
	}
	wg.Wait()
	if got[0].Code != 200 || got[1].Code != 200 || sessionCookie(got[0]).Value == sessionCookie(got[1]).Value {
		t.Errorf("concurrent sign-ins: %d %d", got[0].Code, got[1].Code)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM sessions`); n != before+2 {
		t.Errorf("%d sessions, want %d", n, before+2)
	}

	// SC-15: fixed expiry. 6 days 23 hours old is open; exactly 7 days, and 9, are not.
	hashed, _ := tokenHash(admin.Value)
	age := func(created, expires string) {
		mustExec(t, pool, `UPDATE sessions SET created_at = `+created+`, expires_at = `+expires+` WHERE token_hash = $1`, hashed)
	}
	age(`now() - interval '6 days 23 hours'`, `now() + interval '1 hour'`)
	if rec := do(http.MethodGet, "/v1/auth/me", "", addr, admin); rec.Code != http.StatusOK {
		t.Errorf("6 days 23 hours old: %d", rec.Code)
	}
	age(`now() - interval '7 days'`, `now()`)
	if rec := do(http.MethodGet, "/v1/auth/me", "", addr, admin); rec.Code != http.StatusUnauthorized {
		t.Errorf("exactly 7 days old: %d", rec.Code)
	}
	age(`now() - interval '9 days'`, `now() - interval '2 days'`)
	if rec := do(http.MethodGet, "/v1/auth/me", "", addr, admin); rec.Code != http.StatusUnauthorized {
		t.Errorf("9 days old: %d", rec.Code)
	}

	// SC-16: a switched-off account, a changed role and a removed account act at once.
	cookie := sessionCookie(signIn("staff@example.com", "Staff-Pass-1"))
	mustExec(t, pool, `UPDATE users SET is_active = false WHERE id = $1`, staff.ID)
	if rec := do(http.MethodGet, "/v1/auth/me", "", addr, cookie); rec.Code != http.StatusUnauthorized {
		t.Errorf("a switched-off account: %d", rec.Code)
	}
	mustExec(t, pool, `UPDATE users SET is_active = true WHERE id = $1`, staff.ID)
	if rec := do(http.MethodGet, "/v1/team", "", addr, cookie); rec.Code != http.StatusOK {
		t.Errorf("staff on a team route: %d", rec.Code)
	}
	mustExec(t, pool, `UPDATE users SET role = 'customer' WHERE id = $1`, staff.ID)
	if rec := do(http.MethodGet, "/v1/team", "", addr, cookie); rec.Code != http.StatusForbidden {
		t.Errorf("a changed role: %d", rec.Code)
	}
	mustExec(t, pool, `DELETE FROM users WHERE id = $1`, staff.ID)
	if n := countRows(t, pool, `SELECT count(*) FROM sessions WHERE user_id = $1`, staff.ID); n != 0 {
		t.Errorf("a removed account left %d sessions", n)
	}

	// SC-3: a changed ADMIN_PASSWORD refuses the old password and ends old cookies.
	live := sessionCookie(signIn("dealer@example.com", "Dealer-Pass1"))
	if err := EnsureAdmin(ctx, pool, "dealer@example.com", "Dealer-Pass2-New", discard()); err != nil {
		t.Fatal(err)
	}
	if rec := do(http.MethodGet, "/v1/auth/me", "", addr, live); rec.Code != http.StatusUnauthorized {
		t.Errorf("a cookie from before the change: %d", rec.Code)
	}
	if signIn("dealer@example.com", "Dealer-Pass1").Code != http.StatusUnauthorized || signIn("dealer@example.com", "Dealer-Pass2-New").Code != http.StatusOK {
		t.Error("the old password must be refused and the new one accepted")
	}

	// SC-9: five wrong passwords lock the email; the right one then gets 429 with
	// Retry-After; once the lock is fifteen minutes old it signs in.
	mustExec(t, pool, `DELETE FROM rate_limits`)
	for i := 0; i < 5; i++ {
		if rec := signIn("dealer@example.com", "Wrong-Pass-9"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("wrong password %d: %d", i+1, rec.Code)
		}
	}
	locked := signIn("dealer@example.com", "Dealer-Pass2-New")
	if locked.Code != http.StatusTooManyRequests || locked.Header().Get("Retry-After") == "" || len(locked.Result().Cookies()) != 0 {
		t.Fatalf("the sixth try: %d %v", locked.Code, locked.Header())
	}
	mustExec(t, pool, `UPDATE rate_limits SET window_started_at = window_started_at - interval '16 minutes', locked_until = locked_until - interval '16 minutes'`)
	if rec := signIn("dealer@example.com", "Dealer-Pass2-New"); rec.Code != http.StatusOK {
		t.Errorf("sixteen minutes after the fifth failure: %d %s", rec.Code, rec.Body)
	}

	// Sign-out ends the session on the server.
	out := do(http.MethodPost, "/v1/auth/sign-out", "", addr, live)
	if out.Code != http.StatusNoContent {
		t.Errorf("sign-out: %d", out.Code)
	}
	fresh := sessionCookie(signIn("dealer@example.com", "Dealer-Pass2-New"))
	if do(http.MethodPost, "/v1/auth/sign-out", "", addr, fresh).Code != http.StatusNoContent ||
		do(http.MethodGet, "/v1/auth/me", "", addr, fresh).Code != http.StatusUnauthorized {
		t.Error("the signed-out cookie still opens me")
	}
}

func TestPurgeDeletesExpiredSessionsOnly(t *testing.T) {
	pool := dbtest.Migrated(t)
	u := mustUser(t, pool, "staff@example.com", "Staff-Pass-1", Staff)
	mustExec(t, pool, `INSERT INTO sessions (user_id, token_hash, created_at, expires_at) VALUES
		($1, sha256('old'::bytea), now() - interval '9 days', now() - interval '2 days'),
		($1, sha256('new'::bytea), now(), now() + interval '7 days')`, u.ID)
	purgeOnce(context.Background(), pool, discard())
	if n := countRows(t, pool, `SELECT count(*) FROM sessions`); n != 1 {
		t.Errorf("%d sessions left, want 1", n)
	}
}
