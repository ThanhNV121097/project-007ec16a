package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
)

// The edge proxy strips /api, so every route is /v1/...; what the contract
// does not list answers the JSON envelope, never an empty 200 or Go's plain 404.
func TestUnlistedCollectionsAndWritesAnswerNotFound(t *testing.T) {
	h := New(config.Config{}, nil, quiet())
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/v1/nothing"},
		{http.MethodGet, "/v1/nothing/x"},
		{http.MethodGet, "/api/v1/products"},
		{http.MethodPost, "/v1/products"},
		{http.MethodPost, "/v1/product-filters"},
		{http.MethodDelete, "/v1/product-filters"},
		{http.MethodDelete, "/v1/products/samba"},
	} {
		rec := get(h, c.method, c.path)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"not_found"`) {
			t.Errorf("%s %s: %d %s", c.method, c.path, rec.Code, rec.Body)
		}
	}
}
