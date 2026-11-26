package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

var generated = regexp.MustCompile(`^[0-9a-f]{16}$`)

func serveWithID(h http.Handler, id string, set bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if set {
		req.Header.Set(RequestIDHeader, id)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TC-78: every answer carries X-Request-Id; one the caller sent comes back as sent.
func TestRequestIDIsGeneratedWhenTheCallerSendsNone(t *testing.T) {
	h := RequestID()(ok())
	first := serveWithID(h, "", false).Header().Get(RequestIDHeader)
	second := serveWithID(h, "", false).Header().Get(RequestIDHeader)
	if !generated.MatchString(first) || !generated.MatchString(second) {
		t.Fatalf("generated ids %q and %q are not 16 lower-case hex characters", first, second)
	}
	if first == second {
		t.Errorf("two requests got the same id %q", first)
	}
}

func TestRequestIDEchoesAValidOneAndReplacesTheRest(t *testing.T) {
	h := RequestID()(ok())
	for _, valid := range []string{"tc-request-0001", "abcdefgh", strings.Repeat("a", 64), "A.b_c-d1234"} {
		if got := serveWithID(h, valid, true).Header().Get(RequestIDHeader); got != valid {
			t.Errorf("%q came back as %q", valid, got)
		}
	}
	for _, bad := range []string{"", "abcdefg", strings.Repeat("a", 65), "has space 1234", "slash/1234567", "tokén-12345"} {
		got := serveWithID(h, bad, true).Header().Get(RequestIDHeader)
		if got == bad || !generated.MatchString(got) {
			t.Errorf("%q came back as %q, want a generated id", bad, got)
		}
	}
}

func TestRequestIDIsOnAnAnswerRecoverWrote(t *testing.T) {
	panics := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("secret detail") })
	rec := serveWithID(RequestID()(Recover(quiet())(panics)), "tc-request-0002", true)
	if rec.Code != http.StatusInternalServerError || rec.Header().Get(RequestIDHeader) != "tc-request-0002" {
		t.Errorf("recovered answer: %d, id %q", rec.Code, rec.Header().Get(RequestIDHeader))
	}
}

func TestRequestIDReachesTheHandlerAndIsAbsentWithoutTheMiddleware(t *testing.T) {
	var seen string
	catch := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = RequestIDFrom(r.Context()) })
	serveWithID(RequestID()(catch), "tc-request-0003", true)
	if seen != "tc-request-0003" {
		t.Errorf("the handler saw %q", seen)
	}
	serveWithID(catch, "tc-request-0003", true)
	if seen != "" {
		t.Errorf("without the middleware the handler saw %q", seen)
	}
}

func TestTheAccessLogLineCarriesTheRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	serveWithID(RequestID()(AccessLog(log)(ok())), "tc-request-0004", true)
	if !strings.Contains(buf.String(), `"request_id":"tc-request-0004"`) {
		t.Errorf("the access log line has no request id: %s", buf.String())
	}
}
