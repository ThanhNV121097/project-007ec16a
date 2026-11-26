package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
)

// These tests read the two views of the shoes and the reviews against
// PostgreSQL: a visitor (and a customer, who is served as one) sees only what is
// on sale and shown; the shop's team sees everything. They skip without
// TEST_DATABASE_URL, like every test that needs a database.

// readAs reads a path as a signed-in account of the given role, the way
// auth.Resolve puts the account on the request; an empty role is a visitor.
func readAs(h http.Handler, role auth.Role, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if role != "" {
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "00000000-0000-4000-8000-0000000000aa", Email: "who@example.com", Role: role, Active: true}))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// listAs is the sorted ids a list answers to the role.
func listAs(t *testing.T, h http.Handler, role auth.Role, target string) []string {
	t.Helper()
	rec := readAs(h, role, target)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s as %q: %d %s", target, role, rec.Code, rec.Body)
	}
	var body struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, it := range body.Items {
		got = append(got, it["id"].(string))
	}
	sort.Strings(got)
	if body.Total != len(got) {
		t.Errorf("%s as %q: total %d for %d items", target, role, body.Total, len(got))
	}
	return got
}

type listCase struct {
	target string
	want   []string
}

func TestADraftShoeAndAHiddenReviewAreTheTeamsAlone(t *testing.T) {
	pool := emptyShop(t)
	insertShoe(t, pool, shoe{id: "live-one", name: "Samba Live", line: "samba", category: "women", price: 2790000, sizes: []string{"38"}})
	insertShoe(t, pool, shoe{id: "draft-one", name: "Samba Draft", line: "samba", category: "women", price: 2790000, sizes: []string{"38"}, status: "draft"})
	if _, err := pool.Exec(context.Background(), `INSERT INTO reviews (id, product_id, name, rating, status) VALUES
		('r-live', 'live-one', 'Lan', 5, 'live'), ('r-hidden', 'live-one', 'Mai', 1, 'hidden')`); err != nil {
		t.Fatal(err)
	}
	h := serve(pool)

	// A visitor and a customer see what is on sale and shown, and nothing else.
	for _, role := range []auth.Role{"", auth.Customer} {
		for _, c := range []listCase{
			{"/v1/products", []string{"live-one"}},
			{"/v1/products?q=samba", []string{"live-one"}},
			{"/v1/products?size=38", []string{"live-one"}},
			{"/v1/products?status=draft", []string{}},
			{"/v1/reviews", []string{"r-live"}},
			{"/v1/reviews?product=live-one", []string{"r-live"}},
			{"/v1/reviews?status=hidden", []string{}},
		} {
			if got := listAs(t, h, role, c.target); !reflect.DeepEqual(got, c.want) {
				t.Errorf("%s as %q: %v, want %v", c.target, role, got, c.want)
			}
		}
		// A draft shoe and a hidden review answer as an id that does not exist.
		missing := readAs(h, role, "/v1/products/nope")
		for _, target := range []string{"/v1/products/draft-one", "/v1/reviews/r-hidden"} {
			rec := readAs(h, role, target)
			if rec.Code != http.StatusNotFound {
				t.Errorf("%s as %q: %d, want 404", target, role, rec.Code)
			}
			if target == "/v1/products/draft-one" && rec.Body.String() != missing.Body.String() {
				t.Errorf("a draft shoe answers %s, an unknown id %s", rec.Body, missing.Body)
			}
		}
		// What is served carries nothing only the team reads.
		var seen map[string]any
		_ = json.Unmarshal(readAs(h, role, "/v1/products/live-one").Body.Bytes(), &seen)
		for _, key := range []string{"pairs", "price_listed", "_source", "_invented"} {
			if _, has := seen[key]; has {
				t.Errorf("a shoe served to %q carries %s", role, key)
			}
		}
	}

	// The team reads everything, and a shoe carries what only the team is served.
	for _, role := range []auth.Role{auth.Admin, auth.Staff} {
		for _, c := range []listCase{
			{"/v1/products", []string{"draft-one", "live-one"}},
			{"/v1/products?q=samba", []string{"draft-one", "live-one"}},
			{"/v1/products?status=draft", []string{"draft-one"}},
			{"/v1/reviews", []string{"r-hidden", "r-live"}},
			{"/v1/reviews?status=hidden", []string{"r-hidden"}},
		} {
			if got := listAs(t, h, role, c.target); !reflect.DeepEqual(got, c.want) {
				t.Errorf("%s as %s: %v, want %v", c.target, role, got, c.want)
			}
		}
		rec := readAs(h, role, "/v1/products/draft-one")
		var shoe map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &shoe)
		pairs, _ := shoe["pairs"].(map[string]any)
		if rec.Code != http.StatusOK || pairs["38"] != float64(3) {
			t.Errorf("a draft shoe as %s: %d %s", role, rec.Code, rec.Body)
		}
		for _, key := range []string{"price_listed", "_source", "_invented"} {
			if _, has := shoe[key]; !has {
				t.Errorf("the team's view of a shoe lacks %s", key)
			}
		}
		if rec := readAs(h, role, "/v1/reviews/r-hidden"); rec.Code != http.StatusOK {
			t.Errorf("a hidden review as %s: %d", role, rec.Code)
		}
	}
}
