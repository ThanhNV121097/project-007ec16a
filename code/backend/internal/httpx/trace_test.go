package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientAddrIsTheRightMostForwardedEntry(t *testing.T) {
	cases := []struct {
		name   string
		lines  []string
		remote string
		want   string
	}{
		{"no header uses the connection", nil, "192.0.2.7:5555", "192.0.2.7"},
		{"one entry", []string{"203.0.113.9"}, "10.0.0.1:1", "203.0.113.9"},
		{"forged left-most entries are ignored", []string{"1.1.1.1, 2.2.2.2, 203.0.113.9"}, "10.0.0.1:1", "203.0.113.9"},
		{"several header lines read as one list", []string{"1.1.1.1", "203.0.113.9"}, "10.0.0.1:1", "203.0.113.9"},
		{"a different right-most entry is a different address", []string{"1.1.1.1, 203.0.113.10"}, "10.0.0.1:1", "203.0.113.10"},
		{"trailing blanks are skipped", []string{"203.0.113.9, , "}, "10.0.0.1:1", "203.0.113.9"},
		{"not an address falls back to the connection", []string{"1.1.1.1, not-an-ip"}, "192.0.2.7:5555", "192.0.2.7"},
		{"an IPv4-mapped address is unmapped", []string{"::ffff:203.0.113.9"}, "10.0.0.1:1", "203.0.113.9"},
		{"IPv6 is canonical", []string{"2001:DB8:0:0:0:0:0:1"}, "10.0.0.1:1", "2001:db8::1"},
		{"a remote address with no port is kept as it is", nil, "192.0.2.7", "192.0.2.7"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = c.remote
			for _, l := range c.lines {
				req.Header.Add("X-Forwarded-For", l)
			}
			if got := ClientAddr(req); got != c.want {
				t.Errorf("ClientAddr = %q, want %q", got, c.want)
			}
		})
	}
}

func TestAccessLogNamesTheRoutePatternAndTheRoleNeverThePath(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/things/{id}", func(w http.ResponseWriter, r *http.Request) {
		TraceFrom(r.Context()).Role = "staff"
		JSON(w, http.StatusOK, map[string]string{})
	})
	h := Recover(log)(AccessLog(log)(RecordRoute(mux)))

	req := httptest.NewRequest(http.MethodGet, "/v1/things/secret-token-123?email=a@b.example", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
	line := buf.String()
	for _, want := range []string{"method=GET", `pattern="GET /v1/things/{id}"`, "status=200", "role=staff"} {
		if !strings.Contains(line, want) {
			t.Errorf("the log line lacks %s: %s", want, line)
		}
	}
	for _, leak := range []string{"secret-token-123", "a@b.example", "path="} {
		if strings.Contains(line, leak) {
			t.Errorf("the log line holds %q: %s", leak, line)
		}
	}

	buf.Reset()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nowhere/attacker-chosen", nil))
	if line := buf.String(); strings.Contains(line, "attacker-chosen") || !strings.Contains(line, "status=404") {
		t.Errorf("an unmatched route: %s", line)
	}
}

func TestTraceFromWithoutAccessLogIsAThrowAway(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	TraceFrom(req.Context()).Role = "admin"
	if TraceFrom(req.Context()).Role != "" {
		t.Error("a throw-away trace kept a value")
	}
}
