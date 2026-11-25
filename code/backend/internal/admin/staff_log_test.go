package admin

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5/pgconn"
)

// T8: around a POST, a password PUT and forced 500s, the server's log holds no
// password, no hash, no typed email and no row from a database error's detail.
func TestNothingSecretIsLogged(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	fs := newFakeStore()
	mux := http.NewServeMux()
	(&staffHandler{store: fs, log: log}).mount(mux)
	var h http.Handler = httpx.RecordRoute(mux)
	signedIn := h
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signedIn.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), auth.User{Role: auth.Admin})))
	})
	h = httpx.AccessLog(log)(h)

	send := func(method, path, body string) int {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	const typedEmail, typedPassword = "typed.person@example.com", "hunter2-secret-pw"
	if code := send(http.MethodPost, "/v1/admin/staff", `{"name":"Typed","email":"`+typedEmail+`","password":"`+typedPassword+`"}`); code != 201 {
		t.Fatalf("POST: %d", code)
	}
	if code := send(http.MethodPut, "/v1/admin/staff/"+staffID, `{"password":"`+typedPassword+`"}`); code != 200 {
		t.Fatalf("PUT: %d", code)
	}

	// A CHECK violation carries the whole row, hash included, in its detail.
	fs.failed = &pgconn.PgError{
		Code: "23514", ConstraintName: "ck_users_password_hash_not_blank",
		Message: "new row for relation \"users\" violates check constraint",
		Detail:  "Failing row contains (x, " + typedEmail + ", $argon2id$v=19$m=19456,t=2,p=1$salt$key).",
	}
	for _, call := range []struct{ method, path, body string }{
		{http.MethodPost, "/v1/admin/staff", `{"name":"Typed","email":"` + typedEmail + `","password":"` + typedPassword + `"}`},
		{http.MethodPut, "/v1/admin/staff/" + staffID, `{"password":"` + typedPassword + `"}`},
		{http.MethodDelete, "/v1/admin/staff/" + staffID, ""},
		{http.MethodGet, "/v1/admin/staff", ""},
	} {
		if code := send(call.method, call.path, call.body); code != 500 {
			t.Fatalf("%s: %d", call.method, code)
		}
	}

	got := logs.String()
	for _, secret := range []string{typedPassword, "$argon2id", typedEmail, "Failing row", "violates check constraint"} {
		if strings.Contains(got, secret) {
			t.Errorf("the log holds %q:\n%s", secret, got)
		}
	}
	for _, want := range []string{"sqlstate=23514", "constraint=ck_users_password_hash_not_blank", "route=staff.create", "route=staff.update", "route=staff.delete", "route=staff.list", "status=500"} {
		if !strings.Contains(got, want) {
			t.Errorf("the log lacks %q:\n%s", want, got)
		}
	}
}
