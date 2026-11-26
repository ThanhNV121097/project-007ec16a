package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func routed() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id")})
	})
	mux.HandleFunc("GET /v1/orders/", func(w http.ResponseWriter, _ *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"a": "b"})
	})
	return CleanPaths(mux)
}

func serveGet(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// TC-72: an address with a dot segment or a doubled slash is the same 404 as an
// address nobody has: no redirect, no HTML, no address echoed.
func TestAnUncleanAddressIsTheEnvelope404NotARedirect(t *testing.T) {
	var first string
	for _, target := range []string{
		"/v1/orders/../admin", "/v1/orders/./x", "//v1/orders/x", "/v1/orders//x", "/v1/orders/x/..", "/v1/orders/a/../../admin",
	} {
		rec := serveGet(routed(), target)
		if rec.Code != http.StatusNotFound || rec.Header().Get("Location") != "" {
			t.Errorf("%s: %d, Location %q", target, rec.Code, rec.Header().Get("Location"))
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("%s: content type %q", target, ct)
		}
		if first == "" {
			first = rec.Body.String()
		}
		if rec.Body.String() != first || strings.Contains(rec.Body.String(), "admin") {
			t.Errorf("%s: body %q differs or echoes the address", target, rec.Body)
		}
	}
}

func TestACleanAddressReachesTheRouter(t *testing.T) {
	for _, target := range []string{"/v1/orders/doi-24817", "/v1/orders/", "/v1/orders/..%2Fadmin", "/v1/orders/a%2Fb"} {
		if rec := serveGet(routed(), target); rec.Code != http.StatusOK {
			t.Errorf("%s: %d, want 200 from the router", target, rec.Code)
		}
	}
}
