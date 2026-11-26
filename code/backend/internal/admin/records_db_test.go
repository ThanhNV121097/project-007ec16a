package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/data"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/dbtest"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

// These tests run the real SQL of the team's writes against PostgreSQL, through
// the real handlers, the real sign-in and the real readers. They read
// TEST_DATABASE_URL, as CI's backend job sets it, and skip without it.

func newRecordsEnv(t *testing.T) *env {
	t.Helper()
	pool := dbtest.Migrated(t)
	if err := auth.EnsureAdmin(context.Background(), pool, "dealer@example.com", "Dealer-Pass1", quiet()); err != nil {
		t.Fatal(err)
	}
	svc := auth.New(pool, quiet())
	mux := http.NewServeMux()
	svc.Register(mux)
	RegisterRecords(mux, pool, quiet())
	data.Register(mux, pool, quiet())
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	e := &env{t: t, pool: pool, svc: svc, h: svc.Resolve()(httpx.RecordRoute(mux))}
	code, cookie := e.signIn("dealer@example.com", "Dealer-Pass1")
	if code != 200 {
		t.Fatalf("the admin cannot sign in: %d", code)
	}
	e.admin = cookie
	return e
}

func (e *env) text(sql string, args ...any) string {
	e.t.Helper()
	var s string
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return s
}

func (e *env) send(method, path string, body any, cookie *http.Cookie) (*httptest.ResponseRecorder, map[string]any) {
	e.t.Helper()
	text := ""
	if body != nil {
		b, _ := json.Marshal(body)
		text = string(b)
	}
	rec := e.do(method, path, text, cookie)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func (e *env) upload(name string, content []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", name)
	_, _ = part.Write(content)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/v1/uploads", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "session", Value: cookie.Value})
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func list(v any) []string {
	var out []string
	items, _ := v.([]any)
	for _, i := range items {
		out = append(out, fmt.Sprint(i))
	}
	return out
}

func TestTheTeamPostsShoesAndTheirPairsAgainstPostgres(t *testing.T) {
	e := newRecordsEnv(t)
	hash, err := auth.HashPassword("Customer-Pass1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.CreateUser(context.Background(), e.pool, auth.NewUser{Email: "tc-customer@example.com", Name: "Customer", Role: auth.Customer, PasswordHash: hash}); err != nil {
		t.Fatal(err)
	}
	_, customer := e.signIn("tc-customer@example.com", "Customer-Pass1")
	line := e.text(`SELECT id FROM lines ORDER BY seq LIMIT 1`)
	category := e.text(`SELECT id FROM categories ORDER BY seq LIMIT 1`)

	shoe := map[string]any{"name": "Tc Samba Probe", "price": 3290000, "line": line, "category": category, "status": "live",
		"pairs": map[string]int{"38": 4, "39": 2, "40": 0}}
	if rec, _ := e.send(http.MethodPost, "/v1/products", shoe, nil); rec.Code != 401 {
		t.Fatalf("no session: %d", rec.Code)
	}
	if rec, _ := e.send(http.MethodPost, "/v1/products", shoe, customer); rec.Code != 403 {
		t.Fatalf("a customer: %d", rec.Code)
	}
	rec, made := e.send(http.MethodPost, "/v1/products", shoe, e.admin)
	if rec.Code != 201 || made["id"] != "tc-samba-probe" || rec.Header().Get("Location") != "/v1/products/tc-samba-probe" {
		t.Fatalf("create: %d %s %v", rec.Code, rec.Body, rec.Header())
	}
	if fmt.Sprint(list(made["sizes_low"])) != "[39]" || fmt.Sprint(list(made["sizes_out"])) != "[40]" || made["stock"] != float64(6) || made["pairs"] == nil {
		t.Fatalf("the dealer view: %v", made)
	}
	if _, again := e.send(http.MethodPost, "/v1/products", shoe, e.admin); again["id"] != "tc-samba-probe-2" {
		t.Fatalf("a second shoe of the name: %v", again)
	}

	// A visitor sees the shoe, its lists and no pairs; a size sold out does not match.
	vrec, seen := e.send(http.MethodGet, "/v1/products/tc-samba-probe", nil, nil)
	if vrec.Code != 200 || seen["pairs"] != nil || seen["stock"] != float64(6) {
		t.Fatalf("visitor view: %d %s", vrec.Code, vrec.Body)
	}
	if body := e.do(http.MethodGet, "/v1/products?size=40&limit=100", "", nil).Body.String(); strings.Contains(body, "tc-samba-probe") {
		t.Error("a size with no pairs matched the size filter")
	}
	if body := e.do(http.MethodGet, "/v1/products?size=39&limit=100", "", nil).Body.String(); !strings.Contains(body, "tc-samba-probe") {
		t.Error("a size on its last pairs did not match the size filter")
	}

	// A change of the price alone leaves the counts as they were.
	if rec, got := e.send(http.MethodPut, "/v1/products/tc-samba-probe", map[string]any{"price": 2600000}, e.admin); rec.Code != 200 || got["price"] != float64(2600000) || got["stock"] != float64(6) {
		t.Fatalf("price alone: %d %s", rec.Code, rec.Body)
	}

	// An order took two pairs of 38 after the form was read: nothing is overwritten.
	if _, err := e.pool.Exec(context.Background(), `UPDATE product_sizes SET pairs = pairs - 2 WHERE product_id = 'tc-samba-probe' AND size = '38'`); err != nil {
		t.Fatal(err)
	}
	put := map[string]any{"price": 2700000, "pairs": map[string]int{"38": 6, "39": 2, "40": 0}, "seen_pairs": map[string]int{"38": 4, "39": 2, "40": 0}}
	rec, refused := e.send(http.MethodPut, "/v1/products/tc-samba-probe", put, e.admin)
	errs, _ := refused["errors"].(map[string]any)
	if rec.Code != 409 || errs["pairs.38"] != recTaken {
		t.Fatalf("a count an order moved: %d %s", rec.Code, rec.Body)
	}
	if e.count(`SELECT pairs FROM product_sizes WHERE product_id = 'tc-samba-probe' AND size = '38'`) != 2 || e.count(`SELECT price FROM products WHERE id = 'tc-samba-probe'`) != 2600000 {
		t.Fatal("a refused save stored something")
	}
	patch := map[string]any{"size": "38", "pairs": 6, "seen_pairs": 4}
	if rec, _ := e.send(http.MethodPatch, "/v1/products/tc-samba-probe/sizes", patch, e.admin); rec.Code != 409 {
		t.Fatalf("patch with a stale count: %d %s", rec.Code, rec.Body)
	}
	patch["seen_pairs"] = 2
	if rec, got := e.send(http.MethodPatch, "/v1/products/tc-samba-probe/sizes", patch, e.admin); rec.Code != 200 || got["stock"] != float64(8) {
		t.Fatalf("patch with the count seen: %d %s", rec.Code, rec.Body)
	}
	if rec, _ := e.send(http.MethodPatch, "/v1/products/tc-samba-probe/sizes", map[string]any{"size": "99", "pairs": 1, "seen_pairs": 0}, e.admin); rec.Code != 404 {
		t.Fatalf("a size the shoe does not list: %d", rec.Code)
	}

	// A size left out is taken off, with its count; the others stay.
	if rec, got := e.send(http.MethodPut, "/v1/products/tc-samba-probe", map[string]any{"pairs": map[string]int{"38": 6, "39": 2}}, e.admin); rec.Code != 200 || fmt.Sprint(list(got["sizes"])) != "[38 39]" {
		t.Fatalf("taking a size off: %d %s", rec.Code, rec.Body)
	}

	// A draft is the team's: a visitor finds it nowhere.
	if rec, _ := e.send(http.MethodPut, "/v1/products/tc-samba-probe", map[string]any{"status": "draft"}, e.admin); rec.Code != 200 {
		t.Fatalf("draft: %d", rec.Code)
	}
	if e.do(http.MethodGet, "/v1/products/tc-samba-probe", "", nil).Code != 404 || e.do(http.MethodGet, "/v1/products/tc-samba-probe", "", e.admin).Code != 200 {
		t.Error("a draft shoe is not 404 for a visitor and 200 for the dealer")
	}
	if rec := e.do(http.MethodGet, "/v1/products?status=draft", "", nil); !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Errorf("?status=draft for a visitor: %s", rec.Body)
	}
	e.send(http.MethodPut, "/v1/products/tc-samba-probe", map[string]any{"status": "live"}, e.admin)

	// Ratings follow the shown reviews.
	for _, stars := range []int{5, 4} {
		if rec, _ := e.send(http.MethodPost, "/v1/reviews", map[string]any{"product": "tc-samba-probe", "name": "Lan", "rating": stars}, e.admin); rec.Code != 201 {
			t.Fatalf("review: %d %s", rec.Code, rec.Body)
		}
	}
	if got := e.text(`SELECT rating::text || '/' || review_count FROM products WHERE id = 'tc-samba-probe'`); got != "4.5/2" {
		t.Errorf("rating and count %s, want 4.5/2", got)
	}
	_, hidden := e.send(http.MethodPost, "/v1/reviews", map[string]any{"product": "tc-samba-probe", "name": "Mai", "rating": 1, "status": "hidden"}, e.admin)
	if got := e.text(`SELECT rating::text || '/' || review_count FROM products WHERE id = 'tc-samba-probe'`); got != "4.5/2" {
		t.Errorf("a hidden review moved the rating: %s", got)
	}
	if e.do(http.MethodGet, "/v1/reviews/"+fmt.Sprint(hidden["id"]), "", nil).Code != 404 {
		t.Error("a hidden review is visible to a visitor")
	}
	e.send(http.MethodPut, "/v1/reviews/"+fmt.Sprint(hidden["id"]), map[string]any{"status": "live"}, e.admin)
	if got := e.text(`SELECT rating::text || '/' || review_count FROM products WHERE id = 'tc-samba-probe'`); got != "3.3/3" {
		t.Errorf("showing a review: %s, want 3.3/3", got)
	}
	shopTotal := e.count(`SELECT count(*) FROM reviews r JOIN products p ON p.id = r.product_id WHERE r.status = 'live' AND p.status = 'live'`)
	if e.count(`SELECT review_count FROM shop_terms WHERE id = 'shop'`) != shopTotal {
		t.Error("the shop's review count does not follow the shown reviews")
	}

	// Deleting: a shoe an order refers to stays; one no order refers to goes with its reviews and counts.
	ordered := e.text(`SELECT product_id FROM order_items ORDER BY order_id LIMIT 1`)
	rec, kept := e.send(http.MethodDelete, "/v1/products/"+ordered, nil, e.admin)
	if rec.Code != 409 || kept["message"] != "Orders refer to this shoe. Set it to draft instead." {
		t.Fatalf("a shoe an order refers to: %d %s", rec.Code, rec.Body)
	}
	if rec, _ := e.send(http.MethodDelete, "/v1/lines/"+line, nil, e.admin); rec.Code != 409 && e.count(`SELECT count(*) FROM products WHERE line_id = $1`, line) > 0 {
		t.Errorf("a line some shoe uses: %d", rec.Code)
	}
	if rec, _ := e.send(http.MethodDelete, "/v1/products/tc-samba-probe", nil, e.admin); rec.Code != 204 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if e.count(`SELECT count(*) FROM reviews WHERE product_id = 'tc-samba-probe'`)+e.count(`SELECT count(*) FROM product_sizes WHERE product_id = 'tc-samba-probe'`) != 0 {
		t.Error("a deleted shoe left reviews or counts")
	}
	if rec, _ := e.send(http.MethodDelete, "/v1/products/tc-samba-probe", nil, e.admin); rec.Code != 404 {
		t.Errorf("a second delete: %d", rec.Code)
	}
}

func TestThePicturesAreStoredByContentAgainstPostgres(t *testing.T) {
	e := newRecordsEnv(t)
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1}, 1024)...)
	rec := e.upload("../../x.txt", png, e.admin)
	var made map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &made)
	if rec.Code != 201 || !strings.HasPrefix(made["url"], "/api/v1/uploads/") || strings.Contains(made["url"], "x.txt") || !strings.HasSuffix(made["url"], ".png") {
		t.Fatalf("a PNG named x.txt: %d %s", rec.Code, rec.Body)
	}
	got := e.do(http.MethodGet, strings.TrimPrefix(made["url"], "/api"), "", nil)
	if got.Code != 200 || !bytes.Equal(got.Body.Bytes(), png) || got.Header().Get("Content-Type") != "image/png" || got.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("serving it: %d %v", got.Code, got.Header())
	}
	if rec := e.upload("x.jpg", []byte("this is text, not a picture"), e.admin); rec.Code != 415 {
		t.Errorf("text named x.jpg: %d", rec.Code)
	}
	if rec := e.upload("x.png", nil, e.admin); rec.Code != 422 {
		t.Errorf("an empty file: %d", rec.Code)
	}
	jpeg := func(n int) []byte { return append([]byte("\xff\xd8\xff\xe0"), make([]byte, n-4)...) }
	if rec := e.upload("max.jpg", jpeg(5<<20), e.admin); rec.Code != 201 {
		t.Errorf("exactly 5 MiB: %d", rec.Code)
	}
	if rec := e.upload("big.jpg", jpeg(5<<20+1), e.admin); rec.Code != 413 {
		t.Errorf("one byte more: %d", rec.Code)
	}
	if n := e.count(`SELECT count(*) FROM uploads`); n != 2 {
		t.Errorf("%d pictures stored, want 2", n)
	}
}
