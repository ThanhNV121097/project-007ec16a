package auth

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ---- a database that answers the statements of this package ----

type fakeAccount struct {
	id, email, name, role, hash string
	active                      bool
}

// fakeDB stands in for the pool. It matches a statement by a fragment of its
// SQL and keeps the state the flow needs: accounts, sessions and the counters.
type fakeDB struct {
	mu       sync.Mutex
	accounts map[string]fakeAccount // by email
	sessions map[string]User        // by hex of the token hash
	locks    map[string]int         // scope+subject -> seconds left
	fails    map[string]int
	issueOK  bool
	failOn   string // a fragment of SQL that fails with a database error
	queries  int
}

func newFakeDB() *fakeDB {
	f := &fakeDB{accounts: map[string]fakeAccount{}, sessions: map[string]User{}, locks: map[string]int{}, fails: map[string]int{}, issueOK: true}
	for _, a := range []fakeAccount{
		{"u-admin", "admin@example.com", "", "admin", "hash:Right-Pass-1", true},
		{"u-staff", "staff@example.com", "Minh Le", "staff", "hash:Right-Pass-1", true},
		{"u-cust", "customer@example.com", "Lan", "customer", "hash:Right-Pass-1", true},
		{"u-off", "off@example.com", "Off", "staff", "hash:Right-Pass-1", false},
	} {
		f.accounts[a.email] = a
	}
	return f
}

type rowVals []any

func (r rowVals) Scan(dest ...any) error {
	for i, d := range dest {
		switch p := d.(type) {
		case *string:
			*p = r[i].(string)
		case *bool:
			*p = r[i].(bool)
		case *int:
			*p = r[i].(int)
		case *time.Time:
			*p = r[i].(time.Time)
		default:
			return fmt.Errorf("fake row: unsupported %T", d)
		}
	}
	return nil
}

type rowErr struct{ err error }

func (r rowErr) Scan(...any) error { return r.err }

func (f *fakeDB) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries++
	if f.failOn != "" && strings.Contains(sql, f.failOn) {
		return rowErr{errors.New("dial tcp 10.0.0.5:5432: connection refused")}
	}
	switch {
	case strings.Contains(sql, "FROM sessions s JOIN users u"):
		u, ok := f.sessions[hex.EncodeToString(args[0].([]byte))]
		if !ok {
			return rowErr{pgx.ErrNoRows}
		}
		return rowVals{u.ID, u.Email, u.Name, string(u.Role)}
	case strings.Contains(sql, "FROM users WHERE email = $1"):
		a, ok := f.accounts[args[0].(string)]
		if !ok {
			return rowErr{pgx.ErrNoRows}
		}
		return rowVals{a.id, a.name, a.role, a.active, a.hash}
	case strings.Contains(sql, "INSERT INTO sessions"):
		if !f.issueOK {
			return rowErr{pgx.ErrNoRows}
		}
		for _, a := range f.accounts {
			if a.id == args[0].(string) {
				f.sessions[hex.EncodeToString(args[1].([]byte))] = User{ID: a.id, Email: a.email, Name: a.name, Role: Role(a.role), Active: true}
			}
		}
		return rowVals{time.Now().Add(7 * 24 * time.Hour)}
	case strings.Contains(sql, "SELECT ceil(extract"):
		if s := f.locks[args[0].(string)+hex.EncodeToString(args[1].([]byte))]; s > 0 {
			return rowVals{s}
		}
		return rowErr{pgx.ErrNoRows}
	}
	panic("fakeDB: unexpected query: " + sql)
}

func (f *fakeDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries++
	if f.failOn != "" && strings.Contains(sql, f.failOn) {
		return pgconn.CommandTag{}, errors.New("dial tcp 10.0.0.5:5432: connection refused")
	}
	switch {
	case strings.Contains(sql, "INSERT INTO rate_limits"):
		key := args[0].(string) + hex.EncodeToString(args[1].([]byte))
		f.fails[key]++
		if f.fails[key] >= 5 {
			f.locks[key] = 900
		}
	case strings.Contains(sql, "DELETE FROM sessions WHERE token_hash"):
		delete(f.sessions, hex.EncodeToString(args[0].([]byte)))
	}
	return pgconn.NewCommandTag("OK"), nil
}

func (f *fakeDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("fakeDB: Query is not used")
}

func (f *fakeDB) totalFails() int {
	n := 0
	for _, v := range f.fails {
		n += v
	}
	return n
}

// ---- the harness ----

type harness struct {
	db     *fakeDB
	svc    *Service
	h      http.Handler
	verify *atomic.Int32
	logs   *bytes.Buffer
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	logs := &bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(logs, nil))
	db := newFakeDB()
	svc := New(db, log)
	svc.dummy = "dummy-hash"
	var calls atomic.Int32
	svc.verify = func(_ context.Context, encoded, plain string) (bool, error) {
		calls.Add(1)
		return encoded == "hash:"+plain, nil
	}
	mux := http.NewServeMux()
	svc.Register(mux)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, http.StatusOK, map[string]string{"data": "x"}) })
	mux.Handle("GET /v1/team", Require(Admin, Staff)(ok))
	mux.Handle("GET /v1/boss", Require(Admin)(ok))
	mux.Handle("GET /healthz", ok)
	var h http.Handler = httpx.RecordRoute(mux)
	h = svc.Resolve()(h)
	h = httpx.AccessLog(log)(h)
	h = httpx.Recover(log)(h)
	return &harness{db: db, svc: svc, h: h, verify: &calls, logs: logs}
}

func (h *harness) do(method, target, body string, mod func(*http.Request)) *httptest.ResponseRecorder {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.RemoteAddr = "192.0.2.10:4000"
	if mod != nil {
		mod(req)
	}
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	return rec
}

func creds(email, password string) string {
	b, _ := json.Marshal(map[string]string{"email": email, "password": password})
	return string(b)
}

func (h *harness) signIn(email, password string) *httptest.ResponseRecorder {
	return h.do(http.MethodPost, "/v1/auth/sign-in", creds(email, password), nil)
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	return nil
}

func withCookie(c *http.Cookie) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "session", Value: c.Value}) }
}

const goodPassword = "Right-Pass-1"

// ---- SC-1, SC-12, SC-13: signing in, the cookie and who am I ----

func TestSignInAnswersTheAccountAndSetsOneSafeCookie(t *testing.T) {
	h := newHarness(t)
	rec := h.signIn(" Admin@Example.com ", goodPassword)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"email":"admin@example.com","name":"Admin","role":"admin"}` {
		t.Errorf("body %s", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content type %q", ct)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("the sign-in answer may be cached")
	}
	if n := len(rec.Result().Header.Values("Set-Cookie")); n != 1 {
		t.Fatalf("%d Set-Cookie headers", n)
	}
	c := sessionCookie(rec)
	if c == nil || len(c.Value) != 43 || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 604800 || c.Secure {
		t.Fatalf("cookie %+v", c)
	}
	if strings.Contains(rec.Body.String(), c.Value) {
		t.Error("the token is in the body")
	}

	https := h.do(http.MethodPost, "/v1/auth/sign-in", creds("staff@example.com", goodPassword), func(r *http.Request) {
		r.Header.Set("X-Forwarded-Proto", "http, https")
	})
	if c := sessionCookie(https); c == nil || !c.Secure {
		t.Errorf("over https the cookie must be Secure: %+v", c)
	}
	plain := h.do(http.MethodPost, "/v1/auth/sign-in", creds("staff@example.com", goodPassword), func(r *http.Request) {
		r.Header.Set("X-Forwarded-Proto", "https, http")
	})
	if c := sessionCookie(plain); c == nil || c.Secure {
		t.Errorf("the LAST forwarded protocol decides: %+v", c)
	}

	me := h.do(http.MethodGet, "/v1/auth/me", "", withCookie(sessionCookie(rec)))
	if me.Code != http.StatusOK || strings.TrimSpace(me.Body.String()) != `{"email":"admin@example.com","name":"Admin","role":"admin"}` || me.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("me: %d %s %v", me.Code, me.Body, me.Header())
	}
}

func TestEachRoleSignsInWithItsOwnNameAndIsSaidToBeThatRole(t *testing.T) {
	h := newHarness(t)
	for email, want := range map[string]string{
		"staff@example.com":    `{"email":"staff@example.com","name":"Minh Le","role":"staff"}`,
		"customer@example.com": `{"email":"customer@example.com","name":"Lan","role":"customer"}`,
	} {
		rec := h.signIn(email, goodPassword)
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
			t.Errorf("%s: %d %s", email, rec.Code, rec.Body)
		}
		me := h.do(http.MethodGet, "/v1/auth/me", "", withCookie(sessionCookie(rec)))
		if strings.TrimSpace(me.Body.String()) != want {
			t.Errorf("%s me: %s", email, me.Body)
		}
	}
}

func TestMeWithoutAUsableCookieIs401AndMalformedCookiesCostNoQuery(t *testing.T) {
	h := newHarness(t)
	for name, mod := range map[string]func(*http.Request){
		"no cookie":        nil,
		"too short":        func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "session", Value: "abc"}) },
		"not base64url":    func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "session", Value: strings.Repeat("!", 43)}) },
		"a forged 43 char": func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "session", Value: strings.Repeat("A", 43)}) },
	} {
		before := h.db.queries
		rec := h.do(http.MethodGet, "/v1/auth/me", "", mod)
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"error":"unauthenticated"`) {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
		wantQueries := before
		if name == "a forged 43 char" {
			wantQueries = before + 1
		}
		if h.db.queries != wantQueries {
			t.Errorf("%s: %d queries, want %d", name, h.db.queries-before, wantQueries-before)
		}
	}
}

func TestACookieIsNotReadOffTheV1Routes(t *testing.T) {
	h := newHarness(t)
	rec := h.signIn("admin@example.com", goodPassword)
	before := h.db.queries
	res := h.do(http.MethodGet, "/healthz", "", withCookie(sessionCookie(rec)))
	if res.Code != http.StatusOK || h.db.queries != before {
		t.Errorf("/healthz: %d, %d queries", res.Code, h.db.queries-before)
	}
}

// ---- SC-7: one answer for every kind of failure, one verification each ----

func TestEveryFailureAnswersTheSameAndRunsOneVerification(t *testing.T) {
	const want = `{"error":"invalid_credentials","message":"Wrong email or password."}`
	for name, c := range map[string]struct{ email, password string }{
		"unknown email":      {"nobody@example.com", goodPassword},
		"wrong password":     {"admin@example.com", "Wrong-Pass-9"},
		"switched-off":       {"off@example.com", goodPassword},
		"address that cannot be stored": {"no\x00pe@example.com", goodPassword},
	} {
		h := newHarness(t)
		rec := h.signIn(c.email, c.password)
		if rec.Code != http.StatusUnauthorized || strings.TrimSpace(rec.Body.String()) != want {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
		if n := h.verify.Load(); n != 1 {
			t.Errorf("%s: %d verifications, want 1", name, n)
		}
		if h.db.totalFails() != 2 {
			t.Errorf("%s: %d counters written, want 2 (email and address)", name, h.db.totalFails())
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s: a cookie was set", name)
		}
	}
}

func TestADamagedStoredHashVerifiesFalseAfterTheDummy(t *testing.T) {
	h := newHarness(t)
	for _, damaged := range []string{"", "plaintext", "$argon2id$v=19$m=999999999,t=2,p=1$c2FsdHNhbHQ$a2V5", "$argon2i$v=19$m=19456,t=2,p=1$c2FsdHNhbHQ$a2V5a2V5a2V5a2V5a2V5"} {
		ok, err := h.svc.verifyPassword(context.Background(), damaged, "anything")
		if ok || err != nil {
			t.Errorf("%q verified %v, %v", damaged, ok, err)
		}
	}
}

// ---- SC-8: what is refused before anything is counted or hashed ----

func TestRefusedRequestsCostNoVerificationAndCountNothing(t *testing.T) {
	h := newHarness(t)
	for name, c := range map[string]struct {
		body, contentType string
		status            int
		code              string
		fields            []string
	}{
		"blank email":        {creds("  ", goodPassword), "application/json", 422, "invalid", []string{"email"}},
		"blank password":     {creds("admin@example.com", ""), "application/json", 422, "invalid", []string{"password"}},
		"both blank":         {`{}`, "application/json", 422, "invalid", []string{"email", "password"}},
		"too long email":     {creds(strings.Repeat("a", 201)+"@example.com", "x"), "application/json", 422, "invalid", []string{"email"}},
		"too long password":  {creds("admin@example.com", strings.Repeat("p", 101)), "application/json", 422, "invalid", []string{"password"}},
		"charset parameter":  {creds("", "x"), "application/json; charset=utf-8", 422, "invalid", []string{"email"}},
		"not json type":      {creds("admin@example.com", goodPassword), "text/plain", 415, "unsupported_media_type", nil},
		"no content type":    {creds("admin@example.com", goodPassword), "", 415, "unsupported_media_type", nil},
		"unknown field":      {`{"email":"a@b.example","password":"x","role":"admin"}`, "application/json", 400, "bad_request", []string{"role"}},
		"wrong type":         {`{"email":5,"password":"x"}`, "application/json", 400, "bad_request", []string{"email"}},
		"not json":           {`email=a`, "application/json", 400, "bad_request", nil},
		"an array":           {`[]`, "application/json", 400, "bad_request", nil},
		"empty body":         {``, "application/json", 400, "bad_request", nil},
		"trailing data":      {creds("admin@example.com", goodPassword) + `{}`, "application/json", 400, "bad_request", nil},
		"a body over 64 KiB": {`{"email":"` + strings.Repeat("a", 70000) + `"}`, "application/json", 413, "payload_too_large", nil},
	} {
		rec := h.do(http.MethodPost, "/v1/auth/sign-in", c.body, func(r *http.Request) {
			r.Header.Del("Content-Type")
			if c.contentType != "" {
				r.Header.Set("Content-Type", c.contentType)
			}
			if c.body == "" {
				r.Body = http.NoBody
			}
		})
		var body httpx.ErrorBody
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != c.status || body.Error != c.code {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
			continue
		}
		for _, f := range c.fields {
			if body.Errors[f] == "" {
				t.Errorf("%s: no errors.%s in %s", name, f, rec.Body)
			}
		}
		if c.status == 422 && body.Message != "Check the form." {
			t.Errorf("%s: message %q", name, body.Message)
		}
	}
	if n := h.verify.Load(); n != 0 || h.db.totalFails() != 0 || h.db.queries != 0 {
		t.Errorf("%d verifications, %d counters, %d queries; want none", n, h.db.totalFails(), h.db.queries)
	}

	rec := h.do(http.MethodPost, "/v1/auth/sign-in", creds("", ""), nil)
	var body httpx.ErrorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Errors["email"] != "An email address is needed" || body.Errors["password"] != "A password is needed" {
		t.Errorf("sentences: %v", body.Errors)
	}
}

// ---- SC-9, SC-10: the limits ----

func TestFiveWrongPasswordsLockTheEmailEvenForTheRightPassword(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 5; i++ {
		if rec := h.signIn("admin@example.com", "Wrong-Pass-9"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("try %d: %d", i+1, rec.Code)
		}
	}
	before := h.verify.Load()
	failsBefore := h.db.totalFails()
	rec := h.signIn("admin@example.com", goodPassword)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "900" {
		t.Fatalf("%d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	var body httpx.ErrorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "rate_limited" || body.Message != "Too many tries. Please wait 15 minutes." {
		t.Errorf("body %+v", body)
	}
	if h.verify.Load() != before || h.db.totalFails() != failsBefore {
		t.Error("a refused call was verified or counted")
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("a refused call set a cookie")
	}
}

func TestFiveFailuresFromOneAddressLockItForAnyEmailAndForgedLeftEntriesShareTheLock(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 5; i++ {
		email := fmt.Sprintf("nobody%d@example.com", i)
		rec := h.do(http.MethodPost, "/v1/auth/sign-in", creds(email, "x"), func(r *http.Request) {
			r.Header.Set("X-Forwarded-For", fmt.Sprintf("10.9.9.%d, 203.0.113.9", i))
		})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("try %d: %d", i+1, rec.Code)
		}
	}
	locked := h.do(http.MethodPost, "/v1/auth/sign-in", creds("admin@example.com", goodPassword), func(r *http.Request) {
		r.Header.Set("X-Forwarded-For", "172.16.0.1, 203.0.113.9")
	})
	if locked.Code != http.StatusTooManyRequests {
		t.Errorf("the sixth try from the address: %d", locked.Code)
	}
	other := h.do(http.MethodPost, "/v1/auth/sign-in", creds("admin@example.com", goodPassword), func(r *http.Request) {
		r.Header.Set("X-Forwarded-For", "10.9.9.0, 203.0.113.77")
	})
	if other.Code != http.StatusOK {
		t.Errorf("a different right-most entry is a different address: %d %s", other.Code, other.Body)
	}
}

func TestASuccessNeitherCountsNorClears(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 4; i++ {
		h.signIn("admin@example.com", "Wrong-Pass-9")
	}
	if rec := h.signIn("admin@example.com", goodPassword); rec.Code != http.StatusOK {
		t.Fatalf("the success: %d", rec.Code)
	}
	if h.db.totalFails() != 8 {
		t.Errorf("a success changed the counters: %d writes, want 8", h.db.totalFails())
	}
	h.signIn("admin@example.com", "Wrong-Pass-9")
	if rec := h.signIn("admin@example.com", goodPassword); rec.Code != http.StatusTooManyRequests {
		t.Errorf("four failures, a success and one more must lock: %d", rec.Code)
	}
}

func TestALockThatBeganWhileTheHashRanRefusesTheRightPassword(t *testing.T) {
	h := newHarness(t)
	h.svc.verify = func(_ context.Context, encoded, plain string) (bool, error) {
		// Five other failures land while this one hashes.
		for scope, subject := range map[string]string{} {
			_, _ = scope, subject
		}
		for k := range h.db.locks {
			delete(h.db.locks, k)
		}
		h.db.mu.Lock()
		h.db.locks["sign_in_email"+hex.EncodeToString(subjectOf("sign_in_email", "admin@example.com"))] = 600
		h.db.mu.Unlock()
		return encoded == "hash:"+plain, nil
	}
	rec := h.signIn("admin@example.com", goodPassword)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "600" || len(rec.Result().Cookies()) != 0 {
		t.Errorf("%d %v", rec.Code, rec.Header())
	}
	if len(h.db.sessions) != 0 {
		t.Error("a session was made behind a lock")
	}
}

func TestAnAccountThatChangedAfterTheLookupGetsNoSession(t *testing.T) {
	h := newHarness(t)
	h.db.issueOK = false
	rec := h.signIn("admin@example.com", goodPassword)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "invalid_credentials") || len(rec.Result().Cookies()) != 0 {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
	if h.db.totalFails() != 2 {
		t.Errorf("the failure was not counted: %d", h.db.totalFails())
	}
}

func TestADatabaseFailureIsAGeneric500AndCountsNothing(t *testing.T) {
	for _, fragment := range []string{"FROM users WHERE email", "SELECT ceil(extract", "INSERT INTO sessions"} {
		h := newHarness(t)
		h.db.failOn = fragment
		rec := h.signIn("admin@example.com", goodPassword)
		var body httpx.ErrorBody
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != http.StatusInternalServerError || body.Error != "internal" {
			t.Errorf("%s: %d %s", fragment, rec.Code, rec.Body)
		}
		for _, leak := range []string{"dial", "10.0.0.5", "refused", "admin@example.com", goodPassword} {
			if strings.Contains(rec.Body.String(), leak) {
				t.Errorf("%s: the body holds %q", fragment, leak)
			}
		}
		if h.db.totalFails() != 0 {
			t.Errorf("%s: a failed query counted a failure", fragment)
		}
	}
}

// ---- SC-14: sign-out ----

func TestSignOutDeletesTheSessionClearsTheCookieAndAlwaysAnswers204(t *testing.T) {
	h := newHarness(t)
	in := h.signIn("staff@example.com", goodPassword)
	cookie := sessionCookie(in)

	out := h.do(http.MethodPost, "/v1/auth/sign-out", "", withCookie(cookie))
	if out.Code != http.StatusNoContent || out.Body.Len() != 0 {
		t.Fatalf("%d %q", out.Code, out.Body)
	}
	cleared := sessionCookie(out)
	if cleared == nil || cleared.MaxAge != -1 || cleared.Value != "" || !cleared.HttpOnly || cleared.Path != "/" {
		t.Errorf("cookie %+v", cleared)
	}
	if len(h.db.sessions) != 0 {
		t.Error("the session row is still there")
	}
	if me := h.do(http.MethodGet, "/v1/auth/me", "", withCookie(cookie)); me.Code != http.StatusUnauthorized {
		t.Errorf("the old cookie still opens me: %d", me.Code)
	}

	for name, mod := range map[string]func(*http.Request){
		"no cookie":      nil,
		"an unknown one": func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "session", Value: strings.Repeat("B", 43)}) },
		"a malformed":    func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "session", Value: "x"}) },
	} {
		if rec := h.do(http.MethodPost, "/v1/auth/sign-out", "", mod); rec.Code != http.StatusNoContent {
			t.Errorf("%s: %d", name, rec.Code)
		}
	}
}

func TestSignOutWithABodyIs400AndDeletesNothing(t *testing.T) {
	h := newHarness(t)
	cookie := sessionCookie(h.signIn("staff@example.com", goodPassword))
	rec := h.do(http.MethodPost, "/v1/auth/sign-out", `{}`, withCookie(cookie))
	if rec.Code != http.StatusBadRequest || sessionCookie(rec) != nil || len(h.db.sessions) != 1 {
		t.Errorf("%d, cookie %v, %d sessions", rec.Code, sessionCookie(rec), len(h.db.sessions))
	}
	chunked := h.do(http.MethodPost, "/v1/auth/sign-out", "x", func(r *http.Request) {
		r.ContentLength = -1
		withCookie(cookie)(r)
	})
	if chunked.Code != http.StatusBadRequest || len(h.db.sessions) != 1 {
		t.Errorf("a chunked body: %d", chunked.Code)
	}
	empty := h.do(http.MethodPost, "/v1/auth/sign-out", "", func(r *http.Request) {
		r.ContentLength = -1
		r.Body = io.NopCloser(strings.NewReader(""))
	})
	if empty.Code != http.StatusNoContent {
		t.Errorf("an empty chunked body: %d", empty.Code)
	}
}

func TestSignOutWhoseDeleteFailsIs500AndKeepsTheCookie(t *testing.T) {
	h := newHarness(t)
	cookie := sessionCookie(h.signIn("staff@example.com", goodPassword))
	h.db.failOn = "DELETE FROM sessions"
	rec := h.do(http.MethodPost, "/v1/auth/sign-out", "", withCookie(cookie))
	if rec.Code != http.StatusInternalServerError || sessionCookie(rec) != nil {
		t.Errorf("%d, cookie %v", rec.Code, sessionCookie(rec))
	}
}

// ---- SC-18: Require ----

func TestRequireAnswers401ForNoneAnd403ForAnotherRole(t *testing.T) {
	h := newHarness(t)
	cookies := map[string]*http.Cookie{}
	for _, who := range []string{"admin", "staff", "customer"} {
		cookies[who] = sessionCookie(h.signIn(who+"@example.com", goodPassword))
	}
	for _, c := range []struct {
		route, who string
		want       int
	}{
		{"/v1/team", "", 401}, {"/v1/team", "customer", 403}, {"/v1/team", "staff", 200}, {"/v1/team", "admin", 200},
		{"/v1/boss", "", 401}, {"/v1/boss", "customer", 403}, {"/v1/boss", "staff", 403}, {"/v1/boss", "admin", 200},
	} {
		var mod func(*http.Request)
		if c.who != "" {
			mod = withCookie(cookies[c.who])
		}
		rec := h.do(http.MethodGet, c.route, "", mod)
		if rec.Code != c.want {
			t.Errorf("%s as %q: %d, want %d", c.route, c.who, rec.Code, c.want)
		}
		if c.want != 200 {
			var body httpx.ErrorBody
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			wantCode := map[int]string{401: "unauthenticated", 403: "forbidden"}[c.want]
			if body.Error != wantCode || body.Errors != nil || strings.Contains(rec.Body.String(), "data") {
				t.Errorf("%s as %q: body %s", c.route, c.who, rec.Body)
			}
		}
	}
}

func TestRequireWithoutResolveFailsClosed(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	rec := httptest.NewRecorder()
	Require(Admin, Staff, Customer)(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("%d, want 401 for everyone", rec.Code)
	}
}

func TestAFailedSessionLookupIs500ForEveryRoute(t *testing.T) {
	h := newHarness(t)
	cookie := sessionCookie(h.signIn("admin@example.com", goodPassword))
	h.db.failOn = "FROM sessions s JOIN users u"
	for _, route := range []string{"/v1/auth/me", "/v1/team", "/v1/boss"} {
		rec := h.do(http.MethodGet, route, "", withCookie(cookie))
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "refused") {
			t.Errorf("%s: %d %s", route, rec.Code, rec.Body)
		}
	}
}

// ---- SC-20: nothing secret reaches a log line ----

func TestNoLogLineHoldsAPasswordATokenOrAnEmail(t *testing.T) {
	h := newHarness(t)
	h.signIn("nobody@example.com", "Typed-Secret-77")
	h.signIn("admin@example.com", "Typed-Secret-88")
	in := h.signIn("admin@example.com", goodPassword)
	cookie := sessionCookie(in)
	h.do(http.MethodPost, "/v1/auth/sign-out", "", withCookie(cookie))
	h.db.failOn = "FROM users WHERE email"
	h.signIn("admin@example.com", "Typed-Secret-99")

	logs := h.logs.String()
	for _, secret := range []string{"Typed-Secret", goodPassword, cookie.Value, "nobody@example.com", "admin@example.com", "hash:"} {
		if strings.Contains(logs, secret) {
			t.Errorf("a log line holds %q:\n%s", secret, logs)
		}
	}
	for _, want := range []string{"method=POST", `pattern="POST /v1/auth/sign-in"`, "status=401", "status=200", "status=204"} {
		if !strings.Contains(logs, want) {
			t.Errorf("the log lacks %s:\n%s", want, logs)
		}
	}
}

func TestAnAccessLineNamesTheRole(t *testing.T) {
	h := newHarness(t)
	cookie := sessionCookie(h.signIn("staff@example.com", goodPassword))
	h.logs.Reset()
	h.do(http.MethodGet, "/v1/team", "", withCookie(cookie))
	if line := h.logs.String(); !strings.Contains(line, "role=staff") || !strings.Contains(line, `pattern="GET /v1/team"`) {
		t.Errorf("log: %s", line)
	}
}
