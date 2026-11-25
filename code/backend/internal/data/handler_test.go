package data

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

// fakeRow answers Scan with canned values, or with an error.
type fakeRow struct {
	total int64
	text  string
	err   error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for _, d := range dest {
		switch v := d.(type) {
		case *int64:
			*v = r.total
		case *string:
			*v = r.text
		}
	}
	return nil
}

// recorder is the Querier a test hands the reader: it keeps every statement
// the reader sends, with its arguments, and answers with row.
type recorder struct {
	sqls []string
	args [][]any
	row  fakeRow
}

func (r *recorder) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	r.sqls = append(r.sqls, sql)
	r.args = append(r.args, args)
	return r.row
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// serve is the reader as a dealer sees it: the orders list needs a session, so
// every test that is not about access signs in as an admin.
func serve(db Querier) http.Handler { return serveAs(db, auth.Admin) }

// serveAs mounts the reader with the given role already resolved; an empty role
// is a visitor with no session.
func serveAs(db Querier, role auth.Role) http.Handler {
	mux := http.NewServeMux()
	Register(mux, db, quiet())
	if role == "" {
		return mux
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := auth.User{ID: "u1", Email: "team@example.com", Role: role, Active: true}
		mux.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), u)))
	})
}

func request(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestAListIsTheFourKeyShapeWithTheHeadersOfTheContract(t *testing.T) {
	rec := &recorder{row: fakeRow{total: 12, text: `[{"id":"a","price":3290000}]`}}
	res := request(serve(rec), "/v1/products")
	if res.Code != http.StatusOK {
		t.Fatalf("got %d %s", res.Code, res.Body)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 4 || string(body["total"]) != "12" || string(body["page"]) != "1" || string(body["limit"]) != "12" {
		t.Errorf("got %s", res.Body)
	}
	if string(body["items"]) != `[{"id":"a","price":3290000}]` {
		t.Errorf("items: %s", body["items"])
	}
	for header, want := range map[string]string{
		"Content-Type":           "application/json; charset=utf-8",
		"Cache-Control":          "no-store",
		"X-Content-Type-Options": "nosniff",
	} {
		if got := res.Header().Get(header); got != want {
			t.Errorf("%s: %q, want %q", header, got, want)
		}
	}
}

func TestPageAndLimitReachTheStatementAsOffsetAndLimit(t *testing.T) {
	rec := &recorder{row: fakeRow{total: 12, text: "[]"}}
	res := request(serve(rec), "/v1/products?page=2&limit=5")
	if !strings.Contains(res.Body.String(), `"page":2,"limit":5`) {
		t.Errorf("got %s", res.Body)
	}
	args := rec.args[0]
	if len(args) != 2 || args[0] != int64(5) || args[1] != int64(5) {
		t.Errorf("limit and offset: %v", args)
	}
	rec = &recorder{row: fakeRow{total: 130, text: "[]"}}
	res = request(serve(rec), "/v1/products?limit=500&page=9")
	if !strings.Contains(res.Body.String(), `"total":130,"page":9,"limit":100`) {
		t.Errorf("got %s", res.Body)
	}
	if a := rec.args[0]; a[0] != int64(100) || a[1] != int64(800) {
		t.Errorf("limit and offset: %v", a)
	}
}

func TestABadParameterIsA400WithTheEnvelopeAndNoQuery(t *testing.T) {
	for _, target := range []string{
		"/v1/products?page=0", "/v1/products?page=-1", "/v1/products?page=abc", "/v1/products?page=1.5",
		"/v1/products?limit=0", "/v1/products?limit=-1", "/v1/products?limit=abc", "/v1/products?limit=1.5",
		"/v1/orders?phone=0938%20410%20552", "/v1/orders?email=", "/v1/orders?address=x", "/v1/orders?note=x",
		"/v1/lines?_source=owner", "/v1/products?_invented=x", "/v1/products?price_listed=US%24130",
		"/v1/products?category=a%00b", "/v1/products?colour=%ff%fe", "/v1/products?released=0000-01-01",
		"/v1/products?sort=zzz", "/v1/products?sort=price_listed", "/v1/orders?sort=phone", "/v1/products?price_from=abc",
		"/v1/products?released_to=2026-13-40", "/v1/products?price_ranges=3000000", "/v1/orders?phone_from=1",
		"/v1/products?q=" + strings.Repeat("a", 101), "/v1/products?q=%00", "/v1/products?size=%ff",
	} {
		rec := &recorder{row: fakeRow{total: 1, text: "[]"}}
		res := request(serve(rec), target)
		var body struct {
			Error   string            `json:"error"`
			Message string            `json:"message"`
			Errors  map[string]string `json:"errors"`
			Items   json.RawMessage   `json:"items"`
		}
		_ = json.Unmarshal(res.Body.Bytes(), &body)
		if res.Code != http.StatusBadRequest || body.Error != "bad_request" || len(body.Errors) == 0 || body.Items != nil {
			t.Errorf("%s: %d %s", target, res.Code, res.Body)
		}
		if len(rec.sqls) != 0 {
			t.Errorf("%s reached the database", target)
		}
	}
}

func TestIgnoredParametersSendTheSameStatementAsNone(t *testing.T) {
	plain := &recorder{row: fakeRow{total: 12, text: "[]"}}
	request(serve(plain), "/v1/products")
	for _, q := range []string{"zzz=1", "Category=women", "category=", "q=%20%20", "sort=", "size=", "price_from=", "price_ranges=%20"} {
		rec := &recorder{row: fakeRow{total: 12, text: "[]"}}
		res := request(serve(rec), "/v1/products?"+q)
		if res.Code != http.StatusOK || rec.sqls[0] != plain.sqls[0] || len(rec.args[0]) != 2 {
			t.Errorf("%s: %d, statement differs", q, res.Code)
		}
	}
}

// SC-24: the columns a stranger may not read are left out of the query, not
// deleted from the answer afterwards. The test reads the SQL the reader sends.
func TestPublicSQLNamesNoPrivateColumn(t *testing.T) {
	forbidden := func(collection string) []string {
		names := []string{"source", "invented"}
		switch collection {
		case "orders":
			names = append(names, "phone", "email", "address", "note")
		case "products":
			names = append(names, "price_listed")
		}
		return names
	}
	for _, c := range registry {
		rec := &recorder{row: fakeRow{total: 1, text: "[]"}}
		h := serve(rec)
		request(h, "/v1/"+c.Name)
		request(h, "/v1/"+c.Name+"?page=2&limit=3")
		request(h, "/v1/"+c.Name+"/some-id")
		// The query language: words, sort, sizes, any-of, ranges, a union of ranges.
		request(h, "/v1/"+c.Name+"?q=nam%20samba&sort=-id&size=38,40&price_from=1&price_to=9&price_ranges=..5,6..&id=a,b&category=a,b")
		request(h, "/v1/"+c.Name+"?q=%27%29%3B%20DROP%20TABLE%20products%3B%20--")
		if len(rec.sqls) != 5 {
			t.Fatalf("%s: %d statements, want 5", c.Name, len(rec.sqls))
		}
		// search_text is read by a search and never selected.
		for _, sql := range rec.sqls {
			if strings.Contains(c.selectList(), "search_text") || strings.Contains(sql, "DROP") {
				t.Errorf("%s: the select list names search_text, or a word reached the SQL: %s", c.Name, sql)
			}
		}
		for _, sql := range rec.sqls {
			for _, name := range forbidden(c.Name) {
				if regexp.MustCompile(`(?i)\b` + name + `\b`).MatchString(sql) {
					t.Errorf("%s: the SQL names %q: %s", c.Name, name, sql)
				}
			}
		}
	}
}

func TestEveryStatementFailureIsAGeneric500(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.1:5432: connection refused")
	for _, target := range []string{"/v1/products", "/v1/products/samba", "/v1/orders", "/v1/shop/shop"} {
		rec := &recorder{row: fakeRow{err: cause}}
		res := request(serve(rec), target)
		if res.Code != http.StatusInternalServerError || !strings.Contains(res.Body.String(), `"error":"internal"`) {
			t.Errorf("%s: %d %s", target, res.Code, res.Body)
		}
		for _, leak := range []string{"dial", "10.0.0.1", "5432", "SELECT", "refused", "items"} {
			if strings.Contains(res.Body.String(), leak) {
				t.Errorf("%s: the body leaks %q: %s", target, leak, res.Body)
			}
		}
	}
}

func TestARecordIsAnswered404WhenThereIsNone(t *testing.T) {
	rec := &recorder{row: fakeRow{err: pgx.ErrNoRows}}
	res := request(serve(rec), "/v1/products/nope")
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), `"not_found"`) {
		t.Errorf("got %d %s", res.Code, res.Body)
	}
}

func TestAnIDThatCannotBeOneNeverReachesTheDatabase(t *testing.T) {
	for _, id := range []string{"Samba", "a%20b", "a%2Fb", "a..b%00", strings.Repeat("a", 101), "-x"} {
		rec := &recorder{row: fakeRow{text: `{}`}}
		res := request(serve(rec), "/v1/products/"+id)
		if res.Code != http.StatusNotFound || len(rec.sqls) != 0 {
			t.Errorf("%q: %d, %d statements", id, res.Code, len(rec.sqls))
		}
	}
}

func TestARecordIsTheBareObject(t *testing.T) {
	rec := &recorder{row: fakeRow{text: `{"id":"standard","fee":0}`}}
	res := request(serve(rec), "/v1/delivery/standard")
	if res.Code != http.StatusOK || strings.TrimSpace(res.Body.String()) != `{"id":"standard","fee":0}` {
		t.Errorf("got %d %s", res.Code, res.Body)
	}
	if res.Header().Get("Cache-Control") != "no-store" || len(rec.args[0]) != 1 || rec.args[0][0] != "standard" {
		t.Errorf("headers %v args %v", res.Header(), rec.args)
	}
}

// SC-17: the orders list is the team's. No session is 401, a customer 403, a
// dealer 200 — and a refused caller never reaches the database. The shoes stay
// public, and so does the record route that the checkout story replaces.
func TestTheOrdersListIsForTheTeamOnly(t *testing.T) {
	for _, c := range []struct {
		role auth.Role
		want int
		code string
	}{
		{"", http.StatusUnauthorized, "unauthenticated"},
		{auth.Customer, http.StatusForbidden, "forbidden"},
		{auth.Staff, http.StatusOK, ""},
		{auth.Admin, http.StatusOK, ""},
	} {
		rec := &recorder{row: fakeRow{total: 1, text: "[]"}}
		res := request(serveAs(rec, c.role), "/v1/orders")
		if res.Code != c.want {
			t.Errorf("role %q: %d %s, want %d", c.role, res.Code, res.Body, c.want)
		}
		if c.want != http.StatusOK {
			if !strings.Contains(res.Body.String(), `"error":"`+c.code+`"`) || strings.Contains(res.Body.String(), "items") {
				t.Errorf("role %q: the refusal is %s", c.role, res.Body)
			}
			if len(rec.sqls) != 0 {
				t.Errorf("role %q: a refused request reached the database", c.role)
			}
		}
	}
	for _, target := range []string{"/v1/products", "/v1/lines", "/v1/orders/doi-24817"} {
		rec := &recorder{row: fakeRow{total: 1, text: "{}"}}
		if res := request(serveAs(rec, ""), target); res.Code != http.StatusOK {
			t.Errorf("%s with no session: %d, want 200", target, res.Code)
		}
	}
}

func TestOnlyTheTwentyReadRoutesExist(t *testing.T) {
	h := serve(&recorder{})
	for _, c := range []struct{ method, path string }{
		{http.MethodPost, "/v1/products"}, {http.MethodDelete, "/v1/products/samba"}, {http.MethodPut, "/v1/lines/samba"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code == http.StatusOK || rec.Code == http.StatusCreated || rec.Code == http.StatusNoContent {
			t.Errorf("%s %s answered %d", c.method, c.path, rec.Code)
		}
	}
}
