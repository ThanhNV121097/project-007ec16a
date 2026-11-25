package catalogue

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// fakeRow answers Scan with two canned JSON texts, or with an error.
type fakeRow struct {
	colours, sizes string
	err            error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*string) = r.colours
	*dest[1].(*string) = r.sizes
	return nil
}

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

func serve(db Querier) http.Handler {
	mux := http.NewServeMux()
	Register(mux, db, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return mux
}

func do(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

// SC-18: the colours in the closed-set order and the sizes by value, never null.
func TestFiltersAreTheColoursAndSizesOfTheLiveShoes(t *testing.T) {
	rec := &recorder{row: fakeRow{
		colours: `["black","white","pink"]`,
		sizes:   `["40","36 2/3","9","35 1/2","45 1/3","36","37 1/3","S"]`,
	}}
	res := do(serve(rec), http.MethodGet, "/v1/product-filters?category=women&size=38&zzz=1")
	if res.Code != http.StatusOK {
		t.Fatalf("got %d %s", res.Code, res.Body)
	}
	var body Filters
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	wantSizes := []string{"9", "35 1/2", "36", "36 2/3", "37 1/3", "40", "45 1/3", "S"}
	if !reflect.DeepEqual(body.Colours, []string{"black", "white", "pink"}) || !reflect.DeepEqual(body.Sizes, wantSizes) {
		t.Errorf("got %+v", body)
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
	// Any query parameter is ignored: the statement carries only the colour set.
	if len(rec.sqls) != 1 || len(rec.args[0]) != 1 || !reflect.DeepEqual(rec.args[0][0], colourOrder) {
		t.Errorf("statements %d, args %v", len(rec.sqls), rec.args)
	}
	if !strings.Contains(rec.sqls[0], "t.status = 'live'") {
		t.Errorf("only live shoes contribute: %s", rec.sqls[0])
	}
}

func TestNoShoesIsTwoEmptyListsNotNull(t *testing.T) {
	res := do(serve(&recorder{row: fakeRow{colours: `[]`, sizes: `[]`}}), http.MethodGet, "/v1/product-filters")
	if got := strings.TrimSpace(res.Body.String()); got != `{"colours":[],"sizes":[]}` {
		t.Errorf("got %s", got)
	}
}

// SC-17: whatever fails, the caller gets the generic 500 and nothing of the cause.
func TestAFailedStatementIsAGeneric500(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.1:5432: connection refused")
	for name, row := range map[string]fakeRow{
		"a failed statement": {err: cause},
		"unreadable colours": {colours: "not json", sizes: "[]"},
		"unreadable sizes":   {colours: "[]", sizes: "{"},
	} {
		res := do(serve(&recorder{row: row}), http.MethodGet, "/v1/product-filters")
		if res.Code != http.StatusInternalServerError || !strings.Contains(res.Body.String(), `"error":"internal"`) {
			t.Errorf("%s: %d %s", name, res.Code, res.Body)
		}
		for _, leak := range []string{"dial", "10.0.0.1", "5432", "SELECT", "refused", "colours", "json"} {
			if strings.Contains(res.Body.String(), leak) {
				t.Errorf("%s: the body leaks %q: %s", name, leak, res.Body)
			}
		}
	}
}

func TestOnlyAGetIsAnswered(t *testing.T) {
	h := serve(&recorder{})
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if res := do(h, method, "/v1/product-filters"); res.Code == http.StatusOK || res.Code == http.StatusCreated || res.Code == http.StatusNoContent {
			t.Errorf("%s answered %d", method, res.Code)
		}
	}
}

func TestSizeValueReadsSizesTheWayAFootGrows(t *testing.T) {
	for in, want := range map[string]float64{"36": 36, "35 1/2": 35.5, "36 2/3": 36 + 2.0/3, "45 1/3": 45 + 1.0/3, " 40 ": 40} {
		if got, ok := sizeValue(in); !ok || got != want {
			t.Errorf("sizeValue(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "S", "36 x", "36 1/0", "36 1"} {
		if _, ok := sizeValue(bad); ok {
			t.Errorf("sizeValue(%q) read as a number", bad)
		}
	}
}

// The closed colour set is written outside the migration in one place; this is
// the tripwire for it drifting from the schema's own options.
func TestColourOrderIsTheSchemasColourSet(t *testing.T) {
	raw, err := os.ReadFile("../../../frontend/src/data/schema.json")
	if err != nil {
		t.Skipf("schema.json is not beside the backend (a backend-only checkout): %v", err)
	}
	var schema struct {
		Collections map[string]struct {
			Fields []struct {
				Key     string            `json:"key"`
				Options []json.RawMessage `json:"options"`
			} `json:"fields"`
		} `json:"collections"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range schema.Collections["products"].Fields {
		if f.Key != "colour" {
			continue
		}
		for _, o := range f.Options {
			var one string
			var pair []string
			switch {
			case json.Unmarshal(o, &one) == nil:
				got = append(got, one)
			case json.Unmarshal(o, &pair) == nil && len(pair) > 0:
				got = append(got, pair[0])
			}
		}
	}
	if !reflect.DeepEqual(got, colourOrder) {
		t.Errorf("schema colours %v, the package's %v", got, colourOrder)
	}
}
