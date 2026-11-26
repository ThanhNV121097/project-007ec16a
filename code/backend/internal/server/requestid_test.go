package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
)

// TC-78 (services.md §2.1): every answer of the API, refusals included, carries
// X-Request-Id: the caller's own when it is well formed, otherwise 16 hex characters.
func TestEveryAnswerCarriesARequestID(t *testing.T) {
	h := New(config.Config{}, nil, quiet())
	generated := regexp.MustCompile(`^[0-9a-f]{16}$`)
	for _, c := range []struct{ method, path, contentType string }{
		{http.MethodGet, "/v1/nothing", ""},
		{http.MethodGet, "/v1/orders", ""},
		{http.MethodGet, "/v1/orders/../admin", ""},
		{http.MethodPost, "/v1/orders", "text/plain"},
		{http.MethodPost, "/v1/orders", "application/json"},
	} {
		for _, sent := range []string{"", "tc-request-0001"} {
			req := httptest.NewRequest(c.method, c.path, strings.NewReader("not json"))
			if c.contentType != "" {
				req.Header.Set("Content-Type", c.contentType)
			}
			if sent != "" {
				req.Header.Set("X-Request-Id", sent)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			got := rec.Header().Get("X-Request-Id")
			if sent != "" && got != sent {
				t.Errorf("%s %s: sent %q, answer carries %q", c.method, c.path, sent, got)
			}
			if sent == "" && !generated.MatchString(got) {
				t.Errorf("%s %s: no id sent, answer carries %q", c.method, c.path, got)
			}
		}
	}
}
