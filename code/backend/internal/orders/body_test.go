package orders

import (
	"net/http"
	"strings"
	"testing"
)

const itemsText = `"items":[{"product":"samba-og-w-core-black","size":"38","qty":2}]`

// TC-76: a field the contract does not define, or a value of the wrong type, is
// refused 400 and the answer names it; nothing reaches the database.
func TestEveryMalformedBodyIs400AndNamesItsField(t *testing.T) {
	db := &fakeDB{}
	h := mount(db, Options{})
	for _, c := range []struct{ name, body, field string }{
		{"an extra field", strings.Replace(goodBody, `"note":""}`, `"note":"","coupon":"SAVE10"}`, 1), "coupon"},
		{"an extra email", strings.Replace(goodBody, `"note":""}`, `"note":"","email":"lan@example.com"}`, 1), "email"},
		{"an extra field on a pair", strings.Replace(goodBody, `"qty":2}`, `"qty":2,"colour":"black"}`, 1), "items.0.colour"},
		{"a quantity as text", strings.Replace(goodBody, `"qty":2`, `"qty":"2"`, 1), "items.0.qty"},
		{"a quantity with a fraction", strings.Replace(goodBody, `"qty":2`, `"qty":1.5`, 1), "items.0.qty"},
		{"a shoe that is a number", strings.Replace(goodBody, `"product":"samba-og-w-core-black"`, `"product":7`, 1), "items.0.product"},
		{"items as text", strings.Replace(goodBody, itemsText, `"items":"samba-og-w-core-black"`, 1), "items"},
		{"a pair that is not an object", strings.Replace(goodBody, itemsText, `"items":["samba"]`, 1), "items.0"},
		{"a name that is a number", strings.Replace(goodBody, `"customer":"Lan Nguyen"`, `"customer":5`, 1), "customer"},
		{"an array", `[]`, "body"},
		{"not JSON", `not json`, "body"},
		{"null", `null`, "body"},
		{"data after the object", goodBody + `{}`, "body"},
	} {
		rec := post(h, c.body, "application/json", "tc-key-0001")
		e := envelopeOf(t, rec)
		if rec.Code != http.StatusBadRequest || e.Error != "bad_request" || e.Errors[c.field] == "" {
			t.Errorf("%s: %d %s, want 400 bad_request naming %s", c.name, rec.Code, rec.Body, c.field)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("%s: content type %q", c.name, ct)
		}
	}
	if db.touched() != 0 {
		t.Errorf("a body refused at the boundary reached the database %d times", db.touched())
	}
}

func TestEveryFaultOfTheBodyIsNamedAtOnce(t *testing.T) {
	body := strings.Replace(goodBody, `"qty":2}`, `"qty":"2","colour":"red"}`, 1)
	body = strings.Replace(body, `"note":""}`, `"note":"","coupon":"SAVE10"}`, 1)
	_, faults := parseOrder([]byte(body))
	for _, want := range []string{"coupon", "items.0.qty", "items.0.colour"} {
		if faults[want] == "" {
			t.Errorf("%s is not named: %v", want, faults)
		}
	}
}

// ORD-003 AC-6, AC-9: the four numbers a browser may send are read and never used.
func TestTheFourToleratedNumbersAreReadWithoutAFault(t *testing.T) {
	body := strings.Replace(goodBody, `"qty":2}`, `"qty":2,"unit_price":1,"price":1}`, 1)
	body = strings.Replace(body, `"note":""}`, `"note":"","delivery_fee":50000,"total":1}`, 1)
	req, faults := parseOrder([]byte(body))
	if len(faults) != 0 {
		t.Fatalf("refused: %v", faults)
	}
	if len(req.Items) != 1 || req.Items[0].Product != "samba-og-w-core-black" || req.Items[0].Size != "38" || req.Items[0].Qty != 2 {
		t.Errorf("items: %+v", req.Items)
	}
	if req.Customer != "Lan Nguyen" || req.Phone != "0912 345 678" || req.City != "Da Nang" || req.Delivery != "standard" || req.Payment != "vnpay" {
		t.Errorf("fields: %+v", req)
	}
}

// A list over 50 pairs is read as far as the 51st and then refused by validate.
func TestAListOverTheLimitStillReachesValidation(t *testing.T) {
	pair := `{"product":"samba","size":"38","qty":1}`
	body := strings.Replace(goodBody, itemsText, `"items":[`+strings.Repeat(pair+",", 59)+pair+`]`, 1)
	req, faults := parseOrder([]byte(body))
	if len(faults) != 0 {
		t.Fatalf("refused: %v", faults)
	}
	if got := req.validate(defaultOffered)["items"]; !strings.Contains(got, "at most 50") {
		t.Errorf("items: %q (read %d pairs)", got, len(req.Items))
	}
}

// A body of 64 KiB cannot make an answer larger than itself.
func TestTheNumberOfFaultsNamedIsBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{`)
	for i := 0; i < 5000; i++ {
		b.WriteString(`"x`)
		b.WriteString(strings.Repeat("a", i%7))
		b.WriteString(string(rune('a' + i%26)))
		b.WriteString(strings.Repeat("b", i/26))
		b.WriteString(`":1,`)
	}
	b.WriteString(`"note":""}`)
	_, faults := parseOrder([]byte(b.String()))
	if len(faults) == 0 || len(faults) > maxFaults {
		t.Errorf("%d faults named, want 1 to %d", len(faults), maxFaults)
	}
}
