package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
)

func objectOf(t *testing.T, body string) map[string]json.RawMessage {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("%s: %v", body, err)
	}
	return obj
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// SC-13, SC-15: each value rule at its limit, with the sentence the page shows
// beside the field.
func TestOrderFieldRulesAtTheirLimits(t *testing.T) {
	many := func(n int) string { return strings.Repeat("a", n) }
	for _, c := range []struct {
		name, key, value, fault string
	}{
		{"a carrier of 200", "carrier", many(200), ""},
		{"a carrier of 201", "carrier", many(201), "Carrier can have at most 200 characters."},
		{"a tracking number of 201", "tracking", many(201), "Tracking number can have at most 200 characters."},
		{"an empty carrier clears it", "carrier", "", ""},
		{"a day", "eta", "2026-11-26", ""},
		{"no day clears it", "eta", "", ""},
		{"a day written 26/11/2026", "eta", "26/11/2026", "Write the date as year-month-day, for example 2026-11-26."},
		{"a day that is not one", "eta", "2026-13-40", "Write the date as year-month-day, for example 2026-11-26."},
		{"a day without zeros", "eta", "2026-2-3", "Write the date as year-month-day, for example 2026-11-26."},
		{"a word", "eta", "tomorrow", "Write the date as year-month-day, for example 2026-11-26."},
		{"a year before 1900", "eta", "1899-12-31", "Write the date as year-month-day, for example 2026-11-26."},
		{"no customer", "customer", "", "Enter the customer's name."},
		{"a customer of spaces", "customer", "   ", "Enter the customer's name."},
		{"a customer of 201", "customer", many(201), "Customer can have at most 200 characters."},
		{"a phone of 8 digits", "phone", "0912 3456", "Check the phone number: it needs at least 9 digits."},
		{"a phone of 9 digits", "phone", "+84 912 345 67", ""},
		{"an email that is not one", "email", "abc", "Enter a valid email address, or leave it empty."},
		{"no email", "email", "", ""},
		{"an email as typed", "email", "Long@Example.com", ""},
		{"an address of 200", "address", many(200), ""},
		{"an address of 201", "address", many(201), "Address can have at most 200 characters."},
		{"no address", "address", "", "Enter the street address."},
		{"no city", "city", "", "Enter the city or province."},
		{"a note of 5,000", "note", many(5000), ""},
		{"a note of 5,001", "note", many(5001), "Internal note can have at most 5,000 characters."},
		{"a note of 5,000 accented letters", "note", strings.Repeat("ạ", 5000), ""},
		{"a note holding NUL", "note", "a\x00b", "Internal note holds a character that is not allowed."},
		{"a status", "status", "packed", ""},
		{"a status that is none", "status", "archived", "Status must be placed, confirmed, packed, shipped, delivered or cancelled."},
		{"a status in the customer's words", "status", "On its way", "Status must be placed, confirmed, packed, shipped, delivered or cancelled."},
	} {
		b := parseOrderBody(objectOf(t, `{`+quote(c.key)+`:`+quote(c.value)+`}`))
		if len(b.bad) != 0 || b.faults[c.key] != c.fault || (c.fault == "" && len(b.faults) != 0) {
			t.Errorf("%s: bad %v, faults %v, want %q", c.name, b.bad, b.faults, c.fault)
		}
	}
}

// Text is trimmed before it is judged and kept; nothing sent is lost, and two
// faults are named together.
func TestOrderBodyTrimsAndNamesEveryFault(t *testing.T) {
	b := parseOrderBody(objectOf(t, `{"customer":"  Lan  ","note":"","id":"another"}`))
	if len(b.faults) != 0 || b.patch.Customer == nil || *b.patch.Customer != "Lan" || b.patch.Note == nil || *b.patch.Note != "" || b.patch.Status != nil {
		t.Errorf("patch %+v faults %v", b.patch, b.faults)
	}
	b = parseOrderBody(objectOf(t, `{"email":"abc","eta":"tomorrow","total":1}`))
	if len(b.faults) != 3 || b.faults["email"] == "" || b.faults["eta"] == "" || b.faults["total"] != "This cannot be changed." {
		t.Errorf("faults %v", b.faults)
	}
}

// SC-17, SC-18: the twelve names that never change are refused by name whatever
// the value and its type; paid says why.
func TestFrozenOrderFieldsAreRefusedByNameWhateverTheValue(t *testing.T) {
	for _, name := range frozenOrderFields {
		for _, value := range []string{`1`, `"x"`, `null`, `true`, `false`, `[]`, `{}`, `""`} {
			b := parseOrderBody(objectOf(t, `{`+quote(name)+`:`+value+`}`))
			want := "This cannot be changed."
			if name == "paid" {
				want = "Only the payment provider marks this order paid."
			}
			if len(b.bad) != 0 || len(b.faults) != 1 || b.faults[name] != want {
				t.Errorf("%s: %s: bad %v, faults %v", name, value, b.bad, b.faults)
			}
		}
	}
}

// SC-17: a key the order does not have, and a value that is not text, are 400s.
func TestOrderBodyRefusesWhatTheContractDoesNotDefine(t *testing.T) {
	for _, c := range []struct{ body, key, sentence string }{
		{`{"colour":"red"}`, "colour", "Not a field of this order."},
		{`{"seq":3}`, "seq", "Not a field of this order."},
		{`{"is_sample":false}`, "is_sample", "Not a field of this order."},
		{`{"account_id":"x"}`, "account_id", "Not a field of this order."},
		{`{"note":null}`, "note", "Internal note must be text."},
		{`{"carrier":3}`, "carrier", "Carrier must be text."},
		{`{"status":["packed"]}`, "status", "Status must be text."},
		{`{"eta":true}`, "eta", "Expected must be text."},
	} {
		b := parseOrderBody(objectOf(t, c.body))
		if b.bad[c.key] != c.sentence || len(b.bad) != 1 {
			t.Errorf("%s: bad %v", c.body, b.bad)
		}
	}
}

func orderMux() *http.ServeMux {
	mux := http.NewServeMux()
	RegisterOrders(mux, nil, quiet())
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { orderGone(w) })
	return mux
}

func callOrders(role auth.Role, method, path, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if role != "" {
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "u1", Role: role, Active: true}))
	}
	rec := httptest.NewRecorder()
	orderMux().ServeHTTP(rec, req)
	return rec
}

// SC-1, SC-23: no session is 401 and a customer 403, for PUT and DELETE, for an
// id that exists and one that does not, before the body or the database is read
// (the pool here is nil: reaching it would panic).
func TestOnlyTheTeamReachesAnOrderWrite(t *testing.T) {
	for _, c := range []struct {
		role   auth.Role
		status int
	}{{"", 401}, {auth.Customer, 403}} {
		for _, call := range []struct{ method, path, body string }{
			{"PUT", "/v1/orders/doi-24816", `{"status":"confirmed"}`},
			{"PUT", "/v1/orders/doi-1", `not json`},
			{"DELETE", "/v1/orders/doi-24816", ""},
			{"DELETE", "/v1/orders/NOPE!", ""},
		} {
			rec := callOrders(c.role, call.method, call.path, call.body, "text/plain")
			if rec.Code != c.status || strings.Contains(rec.Body.String(), "doi-") {
				t.Errorf("%q %s %s: %d %s", c.role, call.method, call.path, rec.Code, rec.Body)
			}
		}
	}
}

// The boundary of a PUT answers before the database does, in the order of the
// contract: media type, size, one JSON object, the id, then the keys.
func TestAnOrderPutIsRefusedAtTheBoundary(t *testing.T) {
	big := `{"note":"` + strings.Repeat("a", 70<<10) + `"}`
	for _, c := range []struct {
		name, method, path, body, contentType string
		status                                int
		code                                  string
		key                                   string
	}{
		{"not JSON", "PUT", "/v1/orders/doi-1", `{}`, "text/plain", 415, "unsupported_media_type", ""},
		{"a charset is allowed", "PUT", "/v1/orders/NOPE!", `{}`, "application/json; charset=utf-8", 404, "not_found", ""},
		{"too large", "PUT", "/v1/orders/doi-1", big, "application/json", 413, "payload_too_large", ""},
		{"empty body", "PUT", "/v1/orders/doi-1", ``, "application/json", 400, "bad_request", "body"},
		{"an array", "PUT", "/v1/orders/doi-1", `[]`, "application/json", 400, "bad_request", "body"},
		{"null", "PUT", "/v1/orders/doi-1", `null`, "application/json", 400, "bad_request", "body"},
		{"data after the object", "PUT", "/v1/orders/doi-1", `{} {}`, "application/json", 400, "bad_request", "body"},
		{"an id that cannot be one", "PUT", "/v1/orders/NOPE!", `{"status":"confirmed"}`, "application/json", 404, "not_found", ""},
		{"an unknown key", "PUT", "/v1/orders/doi-1", `{"colour":"red"}`, "application/json", 400, "bad_request", "colour"},
		{"a value that is not text", "PUT", "/v1/orders/doi-1", `{"note":null}`, "application/json", 400, "bad_request", "note"},
		{"no pattern for PUT on the list", "PUT", "/v1/orders", `{}`, "application/json", 404, "not_found", ""},
		{"no pattern for PATCH", "PATCH", "/v1/orders/doi-1", `{}`, "application/json", 404, "not_found", ""},
		{"no pattern for POST", "POST", "/v1/orders/doi-1", `{}`, "application/json", 404, "not_found", ""},
	} {
		rec := callOrders(auth.Staff, c.method, c.path, c.body, c.contentType)
		var e struct {
			Error  string            `json:"error"`
			Errors map[string]string `json:"errors"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &e)
		if rec.Code != c.status || e.Error != c.code || (c.key != "" && e.Errors[c.key] == "") {
			t.Errorf("%s: %d %s", c.name, rec.Code, rec.Body)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json; charset=utf-8") {
			t.Errorf("%s: content type %q", c.name, ct)
		}
		if rec.Header().Get("Cache-Control") != "no-store" && c.status != 404 {
			t.Errorf("%s: cache control %q", c.name, rec.Header().Get("Cache-Control"))
		}
	}
}
