package admin

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
)

const adminSentence = "The admin account cannot be switched off, changed or removed."

func TestTheAdminAccountIs409BeforeTheBodyIsRead(t *testing.T) {
	h := newHarness()
	for _, c := range []struct{ name, body, ct string }{
		{"switch off", `{"is_active":false}`, ""},
		{"a new name", `{"name":"Boss"}`, ""},
		{"a new password", `{"password":"another-long-one"}`, ""},
		{"a role", `{"role":"staff"}`, ""},
		{"an invalid body", `{not json`, ""},
		{"an empty body", ``, ""},
		{"another media type", `{"name":"x"}`, "text/plain"},
	} {
		ct := []string{}
		if c.ct != "" {
			ct = []string{c.ct}
		}
		rec := h.do(auth.Admin, http.MethodPut, "/v1/admin/staff/"+adminID, c.body, ct...)
		v := decode(t, rec)
		if rec.Code != 409 || v["error"] != "conflict" || v["message"] != adminSentence || v["errors"] != nil {
			t.Errorf("PUT %s: %d %s", c.name, rec.Code, rec.Body)
		}
	}
	rec := h.do(auth.Admin, http.MethodDelete, "/v1/admin/staff/"+adminID, "")
	if v := decode(t, rec); rec.Code != 409 || v["message"] != adminSentence {
		t.Errorf("DELETE: %d %s", rec.Code, rec.Body)
	}
	if len(h.store.writes) != 0 || !h.store.users[0].Active || h.store.users[0].Role != auth.Admin {
		t.Errorf("the admin changed: %v %+v", h.store.writes, h.store.users[0])
	}
}

func TestACustomersIdAnUnknownIdAndANonUUIDAreOneAnswer(t *testing.T) {
	h := newHarness()
	ids := []string{customerID, unknownID, "not-a-uuid", strings.ToUpper(unknownID), "{" + staffID + "}", strings.ReplaceAll(staffID, "-", "")}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		var first string
		for i, id := range ids {
			body := `{"role":"admin"}`
			if method == http.MethodDelete {
				body = ""
			}
			rec := h.do(auth.Admin, method, "/v1/admin/staff/"+id, body)
			if rec.Code != 404 || decode(t, rec)["message"] != "That account was not found." || decode(t, rec)["error"] != "not_found" {
				t.Errorf("%s %s: %d %s", method, id, rec.Code, rec.Body)
			}
			if i == 0 {
				first = rec.Body.String()
			} else if rec.Body.String() != first {
				t.Errorf("%s %s answers another body: %s", method, id, rec.Body)
			}
		}
	}
	if len(h.store.writes) != 0 || h.store.users[3].Role != auth.Customer || !h.store.users[3].Active {
		t.Errorf("a customer was touched: %v", h.store.writes)
	}
}

func TestUpdateAppliesExactlyTheFieldsSent(t *testing.T) {
	put := func(body string) (*harness, map[string]any) {
		h := newHarness()
		rec := h.do(auth.Admin, http.MethodPut, "/v1/admin/staff/"+staffID, body)
		wantStatus(t, rec, 200)
		v := decode(t, rec)
		wantAccount(t, v)
		return h, v
	}
	h, v := put(`{"name":"  Minh Le "}`)
	if v["name"] != "Minh Le" || v["is_active"] != true || strings.Join(h.store.writes, ",") != "name Minh Le" {
		t.Errorf("name alone: %v %v", v, h.store.writes)
	}
	h, v = put(`{"is_active":false}`)
	if v["is_active"] != false || v["name"] != "Minh Nguyen" || strings.Join(h.store.writes, ",") != "active false" {
		t.Errorf("is_active alone: %v %v", v, h.store.writes)
	}
	h, v = put(`{"is_active":true,"name":"Minh"}`)
	if v["name"] != "Minh" || strings.Join(h.store.writes, ",") != "name Minh,active true" {
		t.Errorf("two fields: %v %v", v, h.store.writes)
	}
	h, v = put(`{"password":"another-long-one"}`)
	hash := h.store.hashes[staffID]
	if strings.Join(h.store.writes, ",") != "password" || !strings.HasPrefix(hash, "$argon2id$") || strings.Contains(hash, "another-long-one") {
		t.Errorf("password alone: %v %q", h.store.writes, hash)
	}
	if _, has := v["password"]; has || strings.Contains(strings.ToLower(strings.Join(keys(v), ",")), "hash") {
		t.Errorf("a secret key in the account: %v", v)
	}
	h, v = put(`{}`)
	if len(h.store.writes) != 0 || v["name"] != "Minh Nguyen" {
		t.Errorf("{} must change nothing: %v %v", v, h.store.writes)
	}
	h, v = put(`{"id":"anything at all","name":"Minh"}`)
	if v["id"] != staffID || strings.Join(h.store.writes, ",") != "name Minh" {
		t.Errorf("id in the body is discarded: %v %v", v, h.store.writes)
	}
}

func keys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestUpdateRefusesWhatTheContractDoesNotDefine(t *testing.T) {
	h := newHarness()
	for _, c := range []struct {
		name, body string
		want       int
		field      string
	}{
		{"role", `{"role":"admin"}`, 422, "role"},
		{"role staff", `{"role":"staff"}`, 422, "role"},
		{"role null", `{"role":null}`, 422, "role"},
		{"role with a valid name", `{"name":"Minh","role":"admin"}`, 422, "role"},
		{"email", `{"email":"x@example.com"}`, 400, "email"},
		{"email and role together", `{"email":"x@example.com","role":"admin"}`, 400, "email"},
		{"null name", `{"name":null}`, 400, "name"},
		{"number password", `{"password":123456789012}`, 400, "password"},
		{"is_active as text", `{"is_active":"no"}`, 400, "is_active"},
		{"is_active null", `{"is_active":null}`, 400, "is_active"},
		{"empty name", `{"name":"   "}`, 422, "name"},
		{"101-character name", `{"name":"` + strings.Repeat("a", 101) + `"}`, 422, "name"},
		{"9-character password", `{"password":"123456789"}`, 422, "password"},
		{"101-character password", `{"password":"` + strings.Repeat("a", 101) + `"}`, 422, "password"},
		{"empty password", `{"password":""}`, 422, "password"},
		{"not JSON", `{`, 400, ""},
		{"empty body", ``, 400, ""},
		{"a list", `[1]`, 400, ""},
	} {
		rec := h.do(auth.Admin, http.MethodPut, "/v1/admin/staff/"+staffID, c.body)
		if rec.Code != c.want {
			t.Errorf("%s: %d, want %d: %s", c.name, rec.Code, c.want, rec.Body)
			continue
		}
		if c.field != "" && errorsOf(t, rec)[c.field] == nil {
			t.Errorf("%s: %q not named: %s", c.name, c.field, rec.Body)
		}
	}
	if rec := h.do(auth.Admin, http.MethodPut, "/v1/admin/staff/"+staffID, `{"name":"x"}`, "text/plain"); rec.Code != 415 {
		t.Errorf("another media type: %d", rec.Code)
	}
	if len(h.store.writes) != 0 || h.store.users[1].Name != "Minh Nguyen" {
		t.Errorf("a refusal changed something: %v", h.store.writes)
	}
}

func TestDeleteRemovesTheAccountAndAnswers204WithNoBody(t *testing.T) {
	h := newHarness()
	rec := h.do(auth.Admin, http.MethodDelete, "/v1/admin/staff/"+staffID, "")
	if rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatalf("%d %q", rec.Code, rec.Body)
	}
	if rec := h.do(auth.Admin, http.MethodDelete, "/v1/admin/staff/"+staffID, ""); rec.Code != 404 {
		t.Errorf("a second DELETE: %d", rec.Code)
	}
	if rec := h.do(auth.Admin, http.MethodPut, "/v1/admin/staff/"+staffID, `{"name":"x"}`); rec.Code != 404 {
		t.Errorf("PUT after DELETE: %d", rec.Code)
	}
	if v := decode(t, h.do(auth.Admin, http.MethodGet, "/v1/admin/staff", "")); v["total"] != float64(2) {
		t.Errorf("the list after a removal: %v", v)
	}
}

func TestOtherMethodsOnTheseAddressesAreTheJSON404(t *testing.T) {
	h := newHarness()
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/v1/admin/staff/" + staffID},
		{http.MethodPatch, "/v1/admin/staff/" + staffID},
		{http.MethodPut, "/v1/admin/staff"},
		{http.MethodDelete, "/v1/admin/staff"},
	} {
		rec := h.do(auth.Admin, c.method, c.path, "")
		if rec.Code != 404 || decode(t, rec)["error"] != "not_found" {
			t.Errorf("%s %s: %d %s", c.method, c.path, rec.Code, rec.Body)
		}
	}
}
