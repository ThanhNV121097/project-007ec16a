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

func serve(db Querier) http.Handler {
	mux := http.NewServeMux()
	Register(mux, db, quiet())
	return mux
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
		"/v1/lines?_source=owner", "/v1/products?_invented=x", "/v1/products?price_listed=US%24130",
		"/v1/products?category=a%00b", "/v1/products?colour=%ff%fe", "/v1/products?released=0000-01-01",
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
	for _, q := range []string{"zzz=1", "q=samba", "sort=-price", "price_from=3000000&price_to=3999999", "category=women,kids", "size=40"} {
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
		if len(rec.sqls) != 3 {
			t.Fatalf("%s: %d statements, want 3", c.Name, len(rec.sqls))
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
	for _, target := range []string{"/v1/products", "/v1/products/samba", "/v1/shop/shop"} {
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
