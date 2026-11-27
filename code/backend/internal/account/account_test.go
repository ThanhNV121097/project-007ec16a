package account

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
)

// serveAs runs one request through the real routes with no database at all: a
// nil DB panics the moment a handler reaches it, so a test here proves the
// request was refused BEFORE any query. role "" is nobody; otherwise a user of
// that role is put in the context as Resolve does.
func serveAs(role, method, path, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	Register(mux, Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if role != "" {
		u := auth.User{ID: "6f1c1e2a-0d4b-4c58-9b53-2f1f0f0a1111", Email: "lan@example.com", Name: "Lan", Role: auth.Role(role), Active: true}
		req = req.WithContext(auth.WithUser(req.Context(), u))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func envelope(t *testing.T, rec *httptest.ResponseRecorder) (code string, fields []string) {
	t.Helper()
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("content type %q", got)
	}
	var body struct {
		Error  string            `json:"error"`
		Errors map[string]string `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("not an envelope: %q", rec.Body)
	}
	for k := range body.Errors {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	return body.Error, fields
}

func TestEverySessionRouteNeedsACustomerBeforeTheBodyIsRead(t *testing.T) {
	for _, route := range []struct{ method, path string }{
		{"GET", "/v1/account"}, {"PUT", "/v1/account"}, {"GET", "/v1/account/orders"},
		{"GET", "/v1/account/orders/doi-24817"}, {"POST", "/v1/account/orders"},
	} {
		if rec := serveAs("", route.method, route.path, "not json"); rec.Code != 401 {
			t.Errorf("%s %s with no session: %d", route.method, route.path, rec.Code)
		} else if code, _ := envelope(t, rec); code != "unauthenticated" {
			t.Errorf("%s %s: %q", route.method, route.path, code)
		}
		for _, role := range []string{"staff", "admin"} {
			if rec := serveAs(role, route.method, route.path, "not json"); rec.Code != 403 {
				t.Errorf("%s %s as %s: %d", route.method, route.path, role, rec.Code)
			}
		}
	}
}

func TestAWrongMethodOrAnAccountIdFindsNoRoute(t *testing.T) {
	for _, route := range []struct{ method, path string }{
		{"DELETE", "/v1/account"}, {"PATCH", "/v1/account"}, {"GET", "/v1/account/sign-up"},
	} {
		mux := http.NewServeMux()
		Register(mux, Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(route.method, route.path, nil))
		if rec.Code != 404 {
			t.Errorf("%s %s: %d", route.method, route.path, rec.Code)
		}
	}
}

func TestSignUpRefusesTheRequestBeforeTheAddressIsCounted(t *testing.T) {
	bad := strings.Repeat("x", 70<<10)
	for _, c := range []struct {
		name, body string
		code       int
		fields     []string
	}{
		{"an undefined field", `{"name":"a","role":"admin"}`, 400, []string{"role"}},
		{"is_active", `{"is_active":true}`, 400, []string{"is_active"}},
		{"a number for the email", `{"email":5}`, 400, []string{"email"}},
		{"an array", `[]`, 400, nil},
		{"null", `null`, 400, nil},
		{"trailing data", `{} {}`, 400, nil},
		{"too large", `{"name":"` + bad + `"}`, 413, nil},
	} {
		rec := serveAs("", "POST", "/v1/account/sign-up", c.body)
		code, fields := envelope(t, rec)
		if rec.Code != c.code || strings.Join(fields, ",") != strings.Join(c.fields, ",") {
			t.Errorf("%s: %d %s %v", c.name, rec.Code, code, fields)
		}
	}
	mux := http.NewServeMux()
	Register(mux, Deps{Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	req := httptest.NewRequest("POST", "/v1/account/sign-up", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 415 {
		t.Errorf("text/plain: %d", rec.Code)
	}
}

func TestPutAccountRefusesBeforeItWritesAndNamesEveryFault(t *testing.T) {
	for _, c := range []struct {
		name, body string
		code       int
		fields     []string
	}{
		{"an id", `{"id":"x"}`, 400, []string{"id"}},
		{"an unknown key", `{"nickname":"x"}`, 400, []string{"nickname"}},
		{"a number for the name", `{"name":5}`, 400, []string{"name"}},
		{"null for the address", `{"address":null}`, 400, []string{"address"}},
		{"the email", `{"email":"x@example.com"}`, 422, []string{"email"}},
		{"the role", `{"role":"admin"}`, 422, []string{"role"}},
		{"the role, as a number", `{"role":1}`, 422, []string{"role"}},
		{"a password beside a good phone", `{"password":"x","phone":"0912 345 678"}`, 422, []string{"password"}},
		{"a short phone and a long address", `{"phone":"0912 34","address":"` + strings.Repeat("a", 201) + `"}`, 422, []string{"address", "phone"}},
		{"a blank name", `{"name":"   "}`, 422, []string{"name"}},
		{"a long city", `{"city":"` + strings.Repeat("c", 101) + `"}`, 422, []string{"city"}},
	} {
		rec := serveAs("customer", "PUT", "/v1/account", c.body)
		_, fields := envelope(t, rec)
		if rec.Code != c.code || strings.Join(fields, ",") != strings.Join(c.fields, ",") {
			t.Errorf("%s: %d %v", c.name, rec.Code, fields)
		}
	}
}

func TestAddOrderNeedsBothKeysAsStrings(t *testing.T) {
	for _, c := range []struct {
		name, body string
		fields     []string
	}{
		{"no phone", `{"number":"DOI-24817"}`, []string{"phone"}},
		{"no number", `{"phone":"0912 345 678"}`, []string{"number"}},
		{"a number as a number", `{"number":24817,"phone":"0912 345 678"}`, []string{"number"}},
		{"null phone", `{"number":"DOI-24817","phone":null}`, []string{"phone"}},
		{"an account id", `{"number":"DOI-24817","phone":"x","account":"y"}`, []string{"account"}},
	} {
		rec := serveAs("customer", "POST", "/v1/account/orders", c.body)
		_, fields := envelope(t, rec)
		if rec.Code != 400 || strings.Join(fields, ",") != strings.Join(c.fields, ",") {
			t.Errorf("%s: %d %v", c.name, rec.Code, fields)
		}
	}
}

func TestTheListRefusesAPageOrALimitThatIsNotAWholeNumber(t *testing.T) {
	for _, q := range []string{"page=0", "page=-1", "page=x", "page=", "limit=0", "limit=abc", "limit=-5", "page=99999999999"} {
		if rec := serveAs("customer", "GET", "/v1/account/orders?"+q, ""); rec.Code != 400 {
			t.Errorf("%s: %d", q, rec.Code)
		}
	}
}

func TestPagingDefaultsToTenAndServesAbove100As100(t *testing.T) {
	for q, want := range map[string][2]int{
		"":                      {1, 10},
		"page=3&limit=25":       {3, 25},
		"limit=500":             {1, 100},
		"limit=99999999999999999999": {1, 100},
		"sort=oldest&q=x":       {1, 10},
	} {
		values, _ := url.ParseQuery(q)
		page, limit, faults := paging(values)
		if len(faults) > 0 || page != want[0] || limit != want[1] {
			t.Errorf("%q: page %d limit %d %v", q, page, limit, faults)
		}
	}
}

func TestAnOrderIdThatCannotBeOneIsTheGenericNotFoundWithNoQuery(t *testing.T) {
	rec := serveAs("customer", "GET", "/v1/account/orders/DOI%2024817", "")
	if rec.Code != 404 || rec.Body.String() != "{\"error\":\"not_found\",\"message\":\"Not found.\"}\n" {
		t.Errorf("%d %q", rec.Code, rec.Body)
	}
}

func TestSignUpRulesAndTheirSentences(t *testing.T) {
	good := signUpInput{name: "Lan Nguyen", email: "Lan@Example.com", phone: "0912 345 678", password: "lanpass123"}
	check := func(mod func(*signUpInput)) map[string]string {
		in := good
		mod(&in)
		_, faults := in.check()
		return faults
	}
	if email, faults := good.check(); len(faults) != 0 || email != "lan@example.com" {
		t.Fatalf("a good sign-up: %q %v", email, faults)
	}
	empty := check(func(in *signUpInput) { *in = signUpInput{} })
	if empty["name"] != "Fill in your name" || empty["email"] != "Enter a valid email address" ||
		empty["phone"] != "Check the phone number" || empty["password"] != "Use at least 8 characters" || len(empty) != 4 {
		t.Errorf("empty: %v", empty)
	}
	cases := []struct {
		name  string
		mod   func(*signUpInput)
		field string
		want  string
	}{
		{"a name of 100", func(in *signUpInput) { in.name = strings.Repeat("a", 100) }, "name", ""},
		{"a name of 101", func(in *signUpInput) { in.name = strings.Repeat("a", 101) }, "name", "Use at most 100 characters"},
		{"a name of spaces", func(in *signUpInput) { in.name = "    " }, "name", "Fill in your name"},
		{"a password of 7", func(in *signUpInput) { in.password = "1234567" }, "password", "Use at least 8 characters"},
		{"a password of 8", func(in *signUpInput) { in.password = "12345678" }, "password", ""},
		{"a password of spaces", func(in *signUpInput) { in.password = "        " }, "password", ""},
		{"a password of 101", func(in *signUpInput) { in.password = strings.Repeat("p", 101) }, "password", "Use at most 100 characters"},
		{"a phone of 0912 34", func(in *signUpInput) { in.phone = "0912 34" }, "phone", "Check the phone number"},
		{"a phone with +84", func(in *signUpInput) { in.phone = "+84 912 345 678" }, "phone", ""},
		{"an email with no dot", func(in *signUpInput) { in.email = "lan@example" }, "email", "Enter a valid email address"},
		{"an email of 201", func(in *signUpInput) { in.email = strings.Repeat("a", 189) + "@example.com" }, "email", "Enter a valid email address"},
		{"an email of 200", func(in *signUpInput) { in.email = strings.Repeat("a", 188) + "@example.com" }, "email", ""},
		{"a NUL in the name", func(in *signUpInput) { in.name = "La\x00n" }, "name", "Fill in your name"},
	}
	for _, c := range cases {
		if got := check(c.mod)[c.field]; got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
