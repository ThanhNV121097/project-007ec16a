package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
)

// T2: the handlers over an in-memory store.

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("not a JSON object: %q", rec.Body)
	}
	return v
}

// errorsOf is the answer's errors map, or nil.
func errorsOf(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	m, _ := decode(t, rec)["errors"].(map[string]any)
	return m
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, want %d: %s", rec.Code, want, rec.Body)
	}
}

// An account is exactly these five keys, and nothing like a password or a hash
// is in any answer (SC-11, SC-21).
func wantAccount(t *testing.T, v map[string]any) {
	t.Helper()
	if len(v) != 5 {
		t.Fatalf("an account has %d keys: %v", len(v), v)
	}
	for _, k := range []string{"id", "name", "email", "role", "is_active"} {
		if _, ok := v[k]; !ok {
			t.Errorf("missing %q in %v", k, v)
		}
	}
}

func TestEveryRouteIsGuardedBeforeAnythingIsRead(t *testing.T) {
	for _, c := range []struct {
		role auth.Role
		want int
	}{{"", 401}, {auth.Staff, 403}, {auth.Customer, 403}} {
		h := newHarness()
		for _, call := range []struct{ method, path, body string }{
			{http.MethodGet, "/v1/admin/staff?page=zero", ""},
			{http.MethodPost, "/v1/admin/staff", "{not json"},
			{http.MethodPut, "/v1/admin/staff/not-a-uuid", "{not json"},
			{http.MethodPut, "/v1/admin/staff/" + adminID, `{"role":"admin"}`},
			{http.MethodDelete, "/v1/admin/staff/" + staffID, ""},
			{http.MethodDelete, "/v1/admin/staff/not-a-uuid", ""},
		} {
			rec := h.do(c.role, call.method, call.path, call.body)
			if rec.Code != c.want {
				t.Errorf("role %q %s %s: %d, want %d", c.role, call.method, call.path, rec.Code, c.want)
			}
		}
		if len(h.store.writes) != 0 || len(h.store.users) != 4 {
			t.Errorf("role %q changed something: %v", c.role, h.store.writes)
		}
	}
}

func TestListIsTheTeamWithTheAdminFirst(t *testing.T) {
	h := newHarness()
	rec := h.do(auth.Admin, http.MethodGet, "/v1/admin/staff?limit=100", "")
	wantStatus(t, rec, 200)
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content type %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("cache control %q", cc)
	}
	v := decode(t, rec)
	items := v["items"].([]any)
	if len(items) != 3 || v["total"] != float64(3) || v["page"] != float64(1) || v["limit"] != float64(100) {
		t.Fatalf("answer: %s", rec.Body)
	}
	first := items[0].(map[string]any)
	wantAccount(t, first)
	if first["role"] != "admin" || first["name"] != "Dealer" || first["is_active"] != true {
		t.Errorf("the admin's row: %v", first)
	}
	for _, it := range items {
		if it.(map[string]any)["role"] == "customer" {
			t.Error("a customer was listed")
		}
	}
	if strings.Contains(rec.Body.String(), "argon2") || strings.Contains(strings.ToLower(rec.Body.String()), "password") {
		t.Errorf("a secret reached the answer: %s", rec.Body)
	}
}

func TestListPagesAndRefusesBadParameters(t *testing.T) {
	h := newHarness()
	v := decode(t, h.do(auth.Admin, http.MethodGet, "/v1/admin/staff?page=2&limit=2", ""))
	if len(v["items"].([]any)) != 1 || v["total"] != float64(3) || v["page"] != float64(2) || v["limit"] != float64(2) {
		t.Errorf("page 2: %v", v)
	}
	if v := decode(t, h.do(auth.Admin, http.MethodGet, "/v1/admin/staff?page=9", "")); v["items"] == nil || len(v["items"].([]any)) != 0 {
		t.Errorf("a page past the end must be an empty list, not null: %v", v)
	}
	if v := decode(t, h.do(auth.Admin, http.MethodGet, "/v1/admin/staff?limit=1000", "")); v["limit"] != float64(100) {
		t.Errorf("limit above 100: %v", v)
	}
	for _, q := range []string{"page=0", "limit=0", "page=abc", "limit=x"} {
		rec := h.do(auth.Admin, http.MethodGet, "/v1/admin/staff?"+q, "")
		wantStatus(t, rec, 400)
		if decode(t, rec)["error"] != "bad_request" || len(errorsOf(t, rec)) != 1 {
			t.Errorf("%s: %s", q, rec.Body)
		}
	}
}

func TestCreateMakesAnActiveStaffAccount(t *testing.T) {
	h := newHarness()
	rec := h.do(auth.Admin, http.MethodPost, "/v1/admin/staff", `{"name":" Lan Tran ","email":"Lan@Example.com","password":"a-twelve-chars"}`)
	wantStatus(t, rec, 201)
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content type %q", ct)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Errorf("a 201 here carries no Location: %q", loc)
	}
	v := decode(t, rec)
	wantAccount(t, v)
	if v["name"] != "Lan Tran" || v["email"] != "lan@example.com" || v["role"] != "staff" || v["is_active"] != true {
		t.Errorf("account: %v", v)
	}
	if strings.Contains(rec.Body.String(), "argon2") || strings.Contains(rec.Body.String(), "a-twelve-chars") {
		t.Errorf("a secret reached the answer: %s", rec.Body)
	}
	hash := h.store.hashes[v["id"].(string)]
	if !strings.HasPrefix(hash, "$argon2id$") || strings.Contains(hash, "a-twelve-chars") {
		t.Errorf("stored hash: %q", hash)
	}
}

func TestCreatePasswordBoundaries(t *testing.T) {
	h := newHarness()
	body := func(email, password string) string {
		b, _ := json.Marshal(map[string]string{"name": "Lan", "email": email, "password": password})
		return string(b)
	}
	for i, c := range []struct {
		password string
		want     int
	}{{strings.Repeat("p", 9), 422}, {strings.Repeat("p", 10), 201}, {strings.Repeat("p", 100), 201}, {strings.Repeat("p", 101), 422}, {strings.Repeat("ế", 10), 201}} {
		email := string(rune('a'+i)) + "@example.com"
		rec := h.do(auth.Admin, http.MethodPost, "/v1/admin/staff", body(email, c.password))
		if rec.Code != c.want {
			t.Errorf("password of %d characters: %d, want %d", len([]rune(c.password)), rec.Code, c.want)
		}
		if c.want == 422 && errorsOf(t, rec)["password"] == nil {
			t.Errorf("422 without naming password: %s", rec.Body)
		}
	}
	if len(h.store.writes) != 3 {
		t.Errorf("a refusal added something: %v", h.store.writes)
	}
}

func TestCreateNamesEveryFieldAtFaultIncludingATakenEmail(t *testing.T) {
	h := newHarness()
	rec := h.do(auth.Admin, http.MethodPost, "/v1/admin/staff", `{"name":"  ","email":"DEALER@example.com","password":"short"}`)
	wantStatus(t, rec, 422)
	got := errorsOf(t, rec)
	if got["name"] == nil || got["password"] == nil || got["email"] != "That email already belongs to an account." {
		t.Errorf("errors: %v", got)
	}
	if v := decode(t, rec); v["error"] != "invalid" || v["message"] != "Check the form." {
		t.Errorf("envelope: %v", v)
	}
	for _, email := range []string{"customer@example.com", "minh@example.com", "MINH@example.com"} {
		rec := h.do(auth.Admin, http.MethodPost, "/v1/admin/staff", `{"name":"x","email":"`+email+`","password":"long-enough-1"}`)
		if rec.Code != 422 || errorsOf(t, rec)["email"] == nil {
			t.Errorf("%s: %d %s", email, rec.Code, rec.Body)
		}
	}
	if len(h.store.writes) != 0 {
		t.Errorf("a refusal added something: %v", h.store.writes)
	}
}

func TestCreateRaceOnTheUniqueIndexIsTheSame422(t *testing.T) {
	h := newHarness()
	h.store.users = h.store.users[:3]
	// The pre-check passes and the INSERT then loses: the store says so.
	racing := &raceStore{fakeStore: h.store}
	mux := http.NewServeMux()
	(&staffHandler{store: racing, log: quiet()}).mount(mux)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/staff", strings.NewReader(`{"name":"x","email":"new@example.com","password":"long-enough-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req.WithContext(auth.WithUser(req.Context(), auth.User{Role: auth.Admin})))
	wantStatus(t, rec, 422)
	if errorsOf(t, rec)["email"] != "That email already belongs to an account." {
		t.Errorf("errors: %s", rec.Body)
	}
}

// raceStore reports no account at the pre-check and a taken email at the insert.
type raceStore struct{ *fakeStore }

func (raceStore) emailTaken(context_ any, _ string) (bool, error) { return false, nil }

func TestCreateRefusesWhatTheContractDoesNotDefine(t *testing.T) {
	h := newHarness()
	for _, c := range []struct {
		name, body, ct string
		want           int
		field          string
	}{
		{"a role", `{"name":"x","email":"a@b.co","password":"long-enough-1","role":"admin"}`, "", 400, "role"},
		{"an unknown key", `{"name":"x","email":"a@b.co","password":"long-enough-1","phone":"1"}`, "", 400, "phone"},
		{"a number for a name", `{"name":5,"email":"a@b.co","password":"long-enough-1"}`, "", 400, "name"},
		{"null for a password", `{"name":"x","email":"a@b.co","password":null}`, "", 400, "password"},
		{"not JSON", `name=x`, "", 400, ""},
		{"an empty body", ``, "", 400, ""},
		{"a list", `[]`, "", 400, ""},
		{"null", `null`, "", 400, ""},
		{"two objects", `{"name":"x"}{"name":"y"}`, "", 400, ""},
		{"another media type", `{"name":"x"}`, "text/plain", 415, ""},
		{"no media type", `{"name":"x"}`, "none", 415, ""},
	} {
		ct := []string{}
		if c.ct == "none" {
			ct = []string{""}
		} else if c.ct != "" {
			ct = []string{c.ct}
		}
		rec := h.do(auth.Admin, http.MethodPost, "/v1/admin/staff", c.body, ct...)
		if rec.Code != c.want {
			t.Errorf("%s: %d, want %d: %s", c.name, rec.Code, c.want, rec.Body)
			continue
		}
		if c.field != "" && errorsOf(t, rec)[c.field] == nil {
			t.Errorf("%s: %q not named: %s", c.name, c.field, rec.Body)
		}
	}
	big := `{"name":"` + strings.Repeat("a", 70<<10) + `"}`
	rec := h.do(auth.Admin, http.MethodPost, "/v1/admin/staff", big)
	if rec.Code != 413 || decode(t, rec)["error"] != "payload_too_large" {
		t.Errorf("a 70 KiB body: %d %s", rec.Code, rec.Body)
	}
	if len(h.store.writes) != 0 {
		t.Errorf("a refusal added something: %v", h.store.writes)
	}
}

func TestAFailedStatementIsA500ThatSaysNothingOfTheCause(t *testing.T) {
	h := newHarness()
	h.store.failed = errors.New("password_hash=$argon2id$secret failing row (lan@example.com)")
	for _, call := range []struct{ method, path, body string }{
		{http.MethodGet, "/v1/admin/staff", ""},
		{http.MethodPost, "/v1/admin/staff", `{"name":"x","email":"a@b.co","password":"long-enough-1"}`},
		{http.MethodPut, "/v1/admin/staff/" + staffID, `{"name":"x"}`},
		{http.MethodDelete, "/v1/admin/staff/" + staffID, ""},
	} {
		rec := h.do(auth.Admin, call.method, call.path, call.body)
		wantStatus(t, rec, 500)
		v := decode(t, rec)
		if v["error"] != "internal" || v["message"] != "Something went wrong. Please try again." || strings.Contains(rec.Body.String(), "secret") {
			t.Errorf("%s %s: %s", call.method, call.path, rec.Body)
		}
	}
}
