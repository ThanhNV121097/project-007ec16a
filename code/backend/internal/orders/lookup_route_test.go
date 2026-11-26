package orders

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// POST /v1/order-lookups with no database: what the route refuses before it
// counts anything, what an unreachable database makes of it, and the small
// rules of its answers. The match itself runs against PostgreSQL in
// lookup_route_db_test.go.

// lkDown is a database nobody can reach. It counts what reaches it, so a request
// refused at the boundary is proved to have touched nothing.
type lkDown struct{ calls int }

type lkNoRow struct{}

func (lkNoRow) Scan(...any) error { return pgx.ErrNoRows }

func (d *lkDown) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	d.calls++
	return nil, errors.New("dial tcp 10.0.0.1:5432: connect: connection refused")
}

func (d *lkDown) QueryRow(context.Context, string, ...any) pgx.Row {
	d.calls++
	return lkNoRow{}
}

func (d *lkDown) Query(context.Context, string, ...any) (pgx.Rows, error) {
	d.calls++
	return nil, errors.New("down")
}

func lkMount(db DB) http.Handler {
	mux := http.NewServeMux()
	Register(mux, db, quiet(), Options{})
	return mux
}

func lkSend(h http.Handler, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/order-lookups", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type lkEnvelope struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors"`
}

func TestLookupRefusesARequestBeforeTheDatabaseAndCountsNothing(t *testing.T) {
	for _, c := range []struct {
		name, body, contentType string
		status                  int
		code                    string
		fields                  []string
	}{
		{"not JSON", `{"number":"DOI-24817","phone":"0938 410 552"}`, "text/plain", 415, "unsupported_media_type", nil},
		{"no content type", `{"number":"DOI-24817","phone":"0938 410 552"}`, "", 415, "unsupported_media_type", nil},
		{"too large", `{"number":"` + strings.Repeat("1", 70<<10) + `","phone":"x"}`, "application/json", 413, "payload_too_large", nil},
		{"not valid JSON", `{"number":`, "application/json", 400, "bad_request", []string{"body"}},
		{"an array", `[]`, "application/json", 400, "bad_request", []string{"body"}},
		{"null", `null`, "application/json", 400, "bad_request", []string{"body"}},
		{"data after the object", `{"number":"1","phone":"1"} {}`, "application/json", 400, "bad_request", []string{"body"}},
		{"a missing phone", `{"number":"DOI-24817"}`, "application/json", 400, "bad_request", []string{"phone"}},
		{"a missing number", `{"phone":"0938 410 552"}`, "application/json", 400, "bad_request", []string{"number"}},
		{"an empty object", `{}`, "application/json", 400, "bad_request", []string{"number", "phone"}},
		{"a number that is a number", `{"number":24817,"phone":"0938 410 552"}`, "application/json", 400, "bad_request", []string{"number"}},
		{"a null phone", `{"number":"DOI-24817","phone":null}`, "application/json", 400, "bad_request", []string{"phone"}},
		{"a phone that is a list", `{"number":"DOI-24817","phone":["0938"]}`, "application/json", 400, "bad_request", []string{"phone"}},
		{"a field the route does not define", `{"number":"DOI-24817","phone":"1","email":"a@b.co"}`, "application/json", 400, "bad_request", []string{"email"}},
		{"a field no one can echo", `{"number":"DOI-24817","phone":"1","a b<>":1}`, "application/json", 400, "bad_request", nil},
		{"JSON with a charset", `{"number":"DOI-24817"}`, "application/json; charset=utf-8", 400, "bad_request", []string{"phone"}},
	} {
		db := &lkDown{}
		rec := lkSend(lkMount(db), c.body, c.contentType)
		var e lkEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
			t.Errorf("%s: not an envelope: %q", c.name, rec.Body)
			continue
		}
		var fields []string
		for k := range e.Errors {
			fields = append(fields, k)
		}
		if rec.Code != c.status || e.Error != c.code || e.Message == "" || db.calls != 0 || len(fields) != len(c.fields) {
			t.Errorf("%s: %d %s %q fields %v, %d database calls", c.name, rec.Code, e.Error, e.Message, fields, db.calls)
		}
		for _, f := range c.fields {
			if e.Errors[f] == "" {
				t.Errorf("%s: no sentence for %q in %v", c.name, f, e.Errors)
			}
		}
		if c.status != 400 && e.Errors != nil {
			t.Errorf("%s: errors on a %d: %v", c.name, c.status, e.Errors)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" || rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Set-Cookie") != "" {
			t.Errorf("%s: headers %v", c.name, rec.Header())
		}
	}
}

func TestLookupAnswers500NotUnavailableWhenTheDatabaseCannotBeReached(t *testing.T) {
	db := &lkDown{}
	rec := lkSend(lkMount(db), `{"number":"DOI-24817","phone":"0938 410 552"}`, "application/json")
	var e lkEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("not an envelope: %q", rec.Body)
	}
	if rec.Code != 500 || e.Error != "internal" || e.Errors != nil || db.calls != 1 {
		t.Errorf("%d %s, %d calls", rec.Code, e.Error, db.calls)
	}
	for _, leak := range []string{"dial", "10.0.0.1", "5432", "refused", "SELECT", "0938"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Errorf("the answer holds %q: %s", leak, rec.Body)
		}
	}
	if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("headers %v", rec.Header())
	}
}

func TestLookupNumberAndPhoneAreComparedOnTheirDigits(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"DOI-24817", "24817", true},
		{"doi-24817", "24817", true},
		{"  DOI - 24817 ", "24817", true},
		{"24817", "24817", true},
		{"123456789012345678", "123456789012345678", true},
		{"1234567890123456789", "1234567890123456789", false},
		{"DOI-", "", false},
		{"", "", false},
	} {
		if got, ok := NumberKey(c.in); got != c.want || ok != c.ok {
			t.Errorf("NumberKey(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
	want := PhoneKey("0938 410 552")
	for _, same := range []string{"0938410552", "+84938410552", "+84 938 410 552", "84938410552", "(0938) 410-552"} {
		if PhoneKey(same) != want {
			t.Errorf("%q is not the phone 0938 410 552", same)
		}
	}
	for _, other := range []string{"552", "410552", "938410552", "0938 410 552 1", "938 410 552 84", "0938 410 553", "", "   "} {
		if got := PhoneKey(other); got == want || (got == "" && other != "" && strings.TrimSpace(other) != "") {
			t.Errorf("%q must not match, key %q", other, got)
		}
	}
}

func TestLookupWaitIsWholeSecondsFromOneToFifteenMinutes(t *testing.T) {
	for _, c := range []struct {
		wait  time.Duration
		after string
		words string
	}{
		{0, "1", "1 minute"},
		{300 * time.Millisecond, "1", "1 minute"},
		{time.Minute, "60", "1 minute"},
		{61 * time.Second, "61", "2 minutes"},
		{14*time.Minute + 59*time.Second, "899", "15 minutes"},
		{15 * time.Minute, "900", "15 minutes"},
		{3 * time.Hour, "900", "15 minutes"},
	} {
		rec := httptest.NewRecorder()
		if words := lookupWait(rec, c.wait); words != c.words || rec.Header().Get("Retry-After") != c.after {
			t.Errorf("%v: Retry-After %q, %q; want %q, %q", c.wait, rec.Header().Get("Retry-After"), words, c.after, c.words)
		}
	}
}

func TestLookupAnswersAreTheFourKeysAndNothingElse(t *testing.T) {
	raw, err := json.Marshal(lookedUp{ID: "doi-24817", Number: "DOI-24817", PlacedAt: "2026-11-23T16:05:00+07:00", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "number", "placed_at", "token"} {
		if _, ok := got[key]; !ok {
			t.Errorf("no %q in %s", key, raw)
		}
	}
	if len(got) != 4 {
		t.Errorf("%d keys: %s", len(got), raw)
	}
}
