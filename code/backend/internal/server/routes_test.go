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
		{http.MethodPost, "/v1/delivery"},
		{http.MethodPut, "/v1/payments/vnpay"},
		{http.MethodDelete, "/v1/faqs/shipping-1"},
		{http.MethodPost, "/v1/shop"},
		{http.MethodPut, "/v1/sizes/38"},
		{http.MethodPost, "/v1/product-filters"},
		{http.MethodDelete, "/v1/product-filters"},
	} {
		rec := get(h, c.method, c.path)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"not_found"`) {
			t.Errorf("%s %s: %d %s", c.method, c.path, rec.Code, rec.Body)
		}
	}
}

// The team's writes exist and are closed to a visitor: 401, before any body is
// read or any database is asked.
func TestTheTeamsWritesAnswerUnauthenticatedWithNoSession(t *testing.T) {
	h := New(config.Config{}, nil, quiet())
	for _, c := range []struct{ method, path string }{
		{http.MethodPost, "/v1/products"},
		{http.MethodPut, "/v1/lines/samba"},
		{http.MethodDelete, "/v1/categories/women"},
		{http.MethodPost, "/v1/reviews"},
		{http.MethodPatch, "/v1/products/samba/sizes"},
		{http.MethodPost, "/v1/uploads"},
	} {
		rec := get(h, c.method, c.path)
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"unauthenticated"`) {
			t.Errorf("%s %s: %d %s", c.method, c.path, rec.Code, rec.Body)
		}
	}
}
