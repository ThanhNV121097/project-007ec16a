package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These cases need no database: the handler is served over a reader that
// answers from memory. The SQL itself is proved in overview_db_test.go.

// overviewClock is the SRS's fixed clock: 25/11/2026 12:00 Vietnam time.
var overviewClock = time.Date(2026, 11, 25, 12, 0, 0, 0, time.FixedZone("ICT", 7*3600))

const overviewGeneric = `{"error":"internal","message":"Something went wrong. Please try again."}`

// fakeOverview answers what it was given and records how it was asked.
type fakeOverview struct {
	answer   overviewAnswer
	err      error
	calls    int
	asked    time.Time
	deadline time.Time
}

func (f *fakeOverview) read(ctx context.Context, now time.Time) (overviewAnswer, error) {
	f.calls++
	f.asked = now
	f.deadline, _ = ctx.Deadline()
	return f.answer, f.err
}

func overviewSample() overviewAnswer {
	return overviewAnswer{
		Month:         overviewMonth{Sales: 47660000, Orders: 13},
		WaitingOrders: 2,
		PairsOnShelf:  73,
		Shoes:         3,
		RunningLow: []overviewLowShoe{
			{ID: "gazelle-indoor-solar-gold", Name: "Gazelle Indoor Solar Gold", Category: "men", Image: "/images/ref/gazelle.jpg", Stock: 5, SizesLow: []string{"40", "41"}},
		},
		SalesByDay:  []overviewDay{{Date: "2026-11-24", Sales: 5890000}, {Date: "2026-11-25", Sales: 3290000}},
		PairsByLine: []overviewLine{{Line: "samba", Name: "Samba", Pairs: 8}},
		LatestOrders: []overviewOrder{
			{ID: "doi-24816", Number: "DOI-24816", Customer: "Trang", Product: "handball-spezial-w-earth-strata", ProductName: "Handball Spezial Earth Strata", Size: "37 1/3", Status: "placed", PlacedAt: "2026-11-25T08:40:00+07:00", Total: 3290000},
		},
	}
}

// overviewEmpty is the answer of a shop with nothing in it: every list made, none nil.
func overviewEmpty() overviewAnswer {
	return overviewAnswer{
		RunningLow:   make([]overviewLowShoe, 0),
		SalesByDay:   make([]overviewDay, 0),
		PairsByLine:  make([]overviewLine, 0),
		LatestOrders: make([]overviewOrder, 0),
	}
}

func overviewRequest(role auth.Role, method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	if role != "" {
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "00000000-0000-4000-8000-0000000000aa", Email: "who@example.com", Role: role, Active: true}))
	}
	return req
}

func overviewServe(r overviewReader, log *slog.Logger, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	(&overviewHandler{reader: r, log: log, now: func() time.Time { return overviewClock }}).mount(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func overviewGet(r overviewReader, role auth.Role, target string) *httptest.ResponseRecorder {
	return overviewServe(r, quiet(), overviewRequest(role, http.MethodGet, target))
}

func sortedKeys(t *testing.T, v any) []string {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("not an object: %v", v)
	}
	return slices.Sorted(maps.Keys(m))
}

// SC-1, SC-2: the guard answers first; the reader is never asked for a visitor
// or a customer, and the refusal carries none of the figures.
func TestOverviewIsDealerOnly(t *testing.T) {
	fake := &fakeOverview{answer: overviewSample()}
	for role, want := range map[auth.Role]struct {
		status int
		code   string
	}{"": {401, "unauthenticated"}, auth.Customer: {403, "forbidden"}} {
		rec := overviewGet(fake, role, "/v1/admin/overview")
		if rec.Code != want.status {
			t.Errorf("role %q: status %d, want %d", role, rec.Code, want.status)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] != want.code {
			t.Errorf("role %q: body %s", role, rec.Body)
		}
		for _, key := range []string{"month", "waiting_orders", "running_low", "latest_orders"} {
			if _, found := body[key]; found {
				t.Errorf("role %q: the refusal holds %q", role, key)
			}
		}
	}
	if fake.calls != 0 {
		t.Errorf("the reader ran %d times for a visitor and a customer", fake.calls)
	}
	for _, role := range []auth.Role{auth.Admin, auth.Staff} {
		if rec := overviewGet(fake, role, "/v1/admin/overview"); rec.Code != 200 {
			t.Errorf("role %q: status %d", role, rec.Code)
		}
	}
}

// Another method on the path is the server's catch-all, not this handler.
func TestOverviewRefusesOtherMethods(t *testing.T) {
	fake := &fakeOverview{answer: overviewSample()}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		if rec := overviewServe(fake, quiet(), overviewRequest(auth.Admin, method, "/v1/admin/overview")); rec.Code != 404 {
			t.Errorf("%s: status %d", method, rec.Code)
		}
	}
	if fake.calls != 0 {
		t.Error("the reader ran for a method that is not GET")
	}
}

// SC-3, SC-12: exactly the eight keys, each row exactly its own, whole numbers,
// never cached, and an unknown parameter changes nothing.
func TestOverviewShape(t *testing.T) {
	fake := &fakeOverview{answer: overviewSample()}
	rec := overviewGet(fake, auth.Staff, "/v1/admin/overview")
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control %q", cc)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := []string{"latest_orders", "month", "pairs_by_line", "pairs_on_shelf", "running_low", "sales_by_day", "shoes", "waiting_orders"}
	if got := sortedKeys(t, body); !reflect.DeepEqual(got, want) {
		t.Errorf("keys %v, want %v", got, want)
	}
	rows := map[string][]string{
		"month":         {"orders", "sales"},
		"running_low":   {"category", "id", "image", "name", "sizes_low", "stock"},
		"sales_by_day":  {"date", "sales"},
		"pairs_by_line": {"line", "name", "pairs"},
		"latest_orders": {"customer", "id", "number", "placed_at", "product", "product_name", "size", "status", "total"},
	}
	for key, keys := range rows {
		row := body[key]
		if list, ok := row.([]any); ok {
			row = list[0]
		}
		if got := sortedKeys(t, row); !reflect.DeepEqual(got, keys) {
			t.Errorf("%s holds %v, want %v", key, got, keys)
		}
	}
	if regexp.MustCompile(`[0-9]\.[0-9]|[0-9][eE][+-]?[0-9]`).Match(rec.Body.Bytes()) {
		t.Errorf("a number is not a whole number: %s", rec.Body)
	}

	other := overviewGet(fake, auth.Staff, "/v1/admin/overview?zzz=1&now=2020-01-01&month=1&date=2020-01-01")
	if other.Code != 200 || other.Body.String() != rec.Body.String() {
		t.Errorf("an unknown parameter changed the answer: %d %s", other.Code, other.Body)
	}
	if !fake.asked.Equal(overviewClock) {
		t.Errorf("the clock was %v, want %v", fake.asked, overviewClock)
	}
}

// SC-12: an empty shop is arrays, never null, in the raw bytes.
func TestOverviewEmptyListsAreArrays(t *testing.T) {
	rec := overviewGet(&fakeOverview{answer: overviewEmpty()}, auth.Admin, "/v1/admin/overview")
	raw := rec.Body.String()
	for _, list := range []string{"running_low", "sales_by_day", "pairs_by_line", "latest_orders"} {
		if !strings.Contains(raw, `"`+list+`":[]`) {
			t.Errorf("%s is not []: %s", list, raw)
		}
	}
	if strings.Contains(raw, "null") {
		t.Errorf("a null in %s", raw)
	}
}

// §3 step 4: the clock is read once per request and a deadline is set.
func TestOverviewReadsTheClockOnceWithADeadline(t *testing.T) {
	fake := &fakeOverview{answer: overviewSample()}
	overviewGet(fake, auth.Admin, "/v1/admin/overview")
	if fake.calls != 1 {
		t.Errorf("the reader ran %d times", fake.calls)
	}
	if left := time.Until(fake.deadline); fake.deadline.IsZero() || left <= 0 || left > overviewTimeout {
		t.Errorf("deadline %v (in %v), want within %v", fake.deadline, left, overviewTimeout)
	}
}

// SC-13: any failure is the generic 500 and nothing of the cause is in the body.
func TestOverviewDatabaseFailureIsGeneric500(t *testing.T) {
	cause := errors.New(`ERROR: relation "orders" does not exist: SELECT sum(total) FROM orders WHERE phone = '0999000TCOV'`)
	for _, failure := range []error{cause, context.DeadlineExceeded} {
		var logs bytes.Buffer
		log := slog.New(slog.NewTextHandler(&logs, nil))
		rec := overviewServe(&fakeOverview{err: failure}, log, overviewRequest(auth.Admin, http.MethodGet, "/v1/admin/overview"))
		if rec.Code != 500 || strings.TrimSpace(rec.Body.String()) != overviewGeneric {
			t.Errorf("%v: %d %s", failure, rec.Code, rec.Body)
		}
		if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("%v: headers %v", failure, rec.Header())
		}
		for _, leak := range []string{"orders", "SELECT", "0999000TCOV", "relation"} {
			if strings.Contains(rec.Body.String(), leak) {
				t.Errorf("%v: the body holds %q", failure, leak)
			}
		}
		if !strings.Contains(logs.String(), "overview.read") {
			t.Errorf("%v: the cause was not logged: %q", failure, logs.String())
		}
	}
}

// SC-13 through the real store: a pool that cannot give a connection is a 500
// with the same envelope.
func TestOverviewClosedPoolIsGeneric500(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://nobody:none@127.0.0.1:1/none?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	rec := overviewGet(overviewStore{db: pool}, auth.Admin, "/v1/admin/overview")
	if rec.Code != 500 || strings.TrimSpace(rec.Body.String()) != overviewGeneric {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
}

// §3: a browser that went away is not an error.
func TestOverviewAbandonedRequestWritesNothing(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := overviewRequest(auth.Admin, http.MethodGet, "/v1/admin/overview").WithContext(auth.WithUser(ctx, auth.User{Role: auth.Admin, Active: true}))
	rec := overviewServe(&fakeOverview{err: context.Canceled}, log, req)
	if rec.Body.Len() != 0 || logs.Len() != 0 {
		t.Errorf("body %q, log %q", rec.Body, logs.String())
	}
}

// SC-16: none of the five statements names a column that holds a customer's
// phone, email, address, city or note, and none is a SELECT *.
func TestOverviewSQLNamesNoPrivateColumn(t *testing.T) {
	private := regexp.MustCompile(`(?i)\b(phone|email|address|city|note)\b`)
	star := regexp.MustCompile(`(?i)select\s+\*`)
	for name, sql := range map[string]string{
		"figures": overviewFiguresSQL, "days": overviewDaysSQL, "lines": overviewLinesSQL,
		"low": overviewLowSQL, "latest": overviewLatestSQL,
	} {
		if m := private.FindString(sql); m != "" {
			t.Errorf("%s names %q", name, m)
		}
		if star.MatchString(sql) {
			t.Errorf("%s is a SELECT *", name)
		}
	}
}
