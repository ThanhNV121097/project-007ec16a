package orders

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/data"
	"github.com/jackc/pgx/v5"
)

// asRole serves h as a person with that role, already resolved (Resolve is the
// server's; here only Require is under test). An empty role is a visitor.
func asRole(h http.Handler, role auth.Role) http.Handler {
	if role == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := auth.User{ID: "u1", Email: "team@example.com", Role: role, Active: true}
		h.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), u)))
	})
}

func getAs(h http.Handler, role auth.Role, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	asRole(h, role).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// listDB answers the list reader's one statement with an empty page and keeps
// the statements it was sent.
type listDB struct {
	fakeDB
	sqls []string
}

func (l *listDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	l.sqls = append(l.sqls, sql)
	return emptyPage{}
}

type emptyPage struct{}

func (emptyPage) Scan(dest ...any) error {
	*(dest[0].(*int64)) = 0
	*(dest[1].(*string)) = "[]"
	return nil
}

// SC-35: orders are not a public collection. The list answers 401 with no
// session and 403 for a customer, with any query string, before the database is
// touched; a dealer (admin or staff) reads it.
func TestTheOrdersListIsForTheTeamOnly(t *testing.T) {
	for _, c := range []struct {
		role   auth.Role
		target string
		status int
		code   string
	}{
		{"", "/v1/orders", http.StatusUnauthorized, "unauthenticated"},
		{"", "/v1/orders?phone=0912%20345%20678", http.StatusUnauthorized, "unauthenticated"},
		{"", "/v1/orders?page=0&limit=abc", http.StatusUnauthorized, "unauthenticated"},
		{"", "/v1/orders?status=placed", http.StatusUnauthorized, "unauthenticated"},
		{auth.Customer, "/v1/orders", http.StatusForbidden, "forbidden"},
		{auth.Customer, "/v1/orders?status=placed", http.StatusForbidden, "forbidden"},
		{auth.Staff, "/v1/orders", http.StatusOK, ""},
		{auth.Admin, "/v1/orders?status=placed&q=lan&sort=-placed_at", http.StatusOK, ""},
	} {
		db := &listDB{}
		rec := getAs(mount(db, Options{}), c.role, c.target)
		if rec.Code != c.status {
			t.Errorf("%q %s: %d %s, want %d", c.role, c.target, rec.Code, rec.Body, c.status)
		}
		if c.status != http.StatusOK {
			if envelopeOf(t, rec).Error != c.code || strings.Contains(rec.Body.String(), "items") {
				t.Errorf("%q %s: the refusal is %s", c.role, c.target, rec.Body)
			}
			if len(db.sqls) != 0 || db.touched() != 0 {
				t.Errorf("%q %s: a refused request reached the database", c.role, c.target)
			}
		}
	}
}

// A dealer's list names only the public columns: a customer's phone, email,
// address and note and the owner's notes are not in the statement, and a filter
// or a sort on one of them is a 400 that sends nothing.
func TestTheTeamsListSelectsNoPrivateColumn(t *testing.T) {
	db := &listDB{}
	h := mount(db, Options{})
	if rec := getAs(h, auth.Staff, "/v1/orders"); rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if len(db.sqls) != 1 {
		t.Fatalf("%d statements, want 1", len(db.sqls))
	}
	for _, name := range []string{"phone", "email", "address", "note", "source", "invented", "idempotency_key"} {
		if strings.Contains(db.sqls[0], "t."+name) {
			t.Errorf("the list selects %s: %s", name, db.sqls[0])
		}
	}
	for _, target := range []string{
		"/v1/orders?phone=0912%20345%20678", "/v1/orders?email=a", "/v1/orders?address=x", "/v1/orders?note=x",
		"/v1/orders?sort=phone", "/v1/orders?phone_from=1", "/v1/orders?_source=owner",
	} {
		rec := getAs(h, auth.Admin, target)
		if rec.Code != http.StatusBadRequest || envelopeOf(t, rec).Error != "bad_request" {
			t.Errorf("%s: %d %s", target, rec.Code, rec.Body)
		}
	}
	if len(db.sqls) != 1 {
		t.Error("a refused parameter reached the database")
	}
}

// Only public columns, in the order the list has always had.
func TestTheTeamsListHasTheSixteenPublicColumns(t *testing.T) {
	var public []string
	for _, c := range dealerList.Columns {
		if c.Access == data.Public {
			public = append(public, c.Name)
		}
	}
	want := "id,number,placed_at,status,product,size,qty,total,payment,paid,delivery,carrier,tracking,eta,customer,city"
	if strings.Join(public, ",") != want {
		t.Errorf("public columns %v, want %s", public, want)
	}
	for _, name := range []string{"phone", "email", "address", "note"} {
		found := false
		for _, c := range dealerList.Columns {
			if c.Name == name {
				found = c.Access == data.Private
			}
		}
		if !found {
			t.Errorf("orders.%s must exist and not be public", name)
		}
	}
}

// The team reads the seeded orders through the real reader and statement; none
// of them shows a customer's phone, email, address or note.
func TestTheTeamReadsTheSeededOrdersWithoutTheirPrivateDetails(t *testing.T) {
	h, _ := shop(t, Options{})
	if rec := getAs(h, "", "/v1/orders"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}
	read := func(target string) (int, []map[string]any) {
		rec := getAs(h, auth.Staff, target)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", target, rec.Code, rec.Body)
		}
		var body struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Total, body.Items
	}
	total, items := read("/v1/orders?limit=100")
	if total != 14 || len(items) != 14 {
		t.Fatalf("total %d, %d items, want 14", total, len(items))
	}
	for _, o := range items {
		for _, key := range []string{"phone", "email", "address", "note", "_source", "_invented"} {
			if _, has := o[key]; has {
				t.Errorf("%v has %s", o["id"], key)
			}
		}
	}
	for target, want := range map[string]int{
		"/v1/orders?status=shipped": 3, "/v1/orders?product=gazelle-indoor-solar-gold": 2, "/v1/orders?qty=2": 1,
		"/v1/orders?placed_at=2026-11-23T16:05:00%2B07:00": 1, "/v1/orders?eta=2026-11-26": 2,
	} {
		if got, _ := read(target); got != want {
			t.Errorf("%s: %d orders, want %d", target, got, want)
		}
	}
	_, one := read("/v1/orders?q=24817")
	if len(one) != 1 || one[0]["product"] != "samba-og-w-crystal-white" || one[0]["size"] != "39 1/3" || one[0]["qty"] != float64(1) ||
		one[0]["total"] != float64(3090000) || one[0]["placed_at"] != "2026-11-23T16:05:00+07:00" {
		t.Errorf("DOI-24817: %v", one)
	}
}
