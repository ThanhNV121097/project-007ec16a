package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

const (
	paidSentence = "Only the payment provider marks this order paid."
	frozenField  = "This cannot be changed."
	checkOrder   = "Check the details of this order."
	notAnOrder   = "Not a field of this order."
	maxLine      = 200
	maxNote      = 5000
)

// orderLabels are the ten fields a dealer may send, with the names the page
// and every sentence call them.
var orderLabels = map[string]string{
	"status": "Status", "carrier": "Carrier", "tracking": "Tracking number", "eta": "Expected",
	"customer": "Customer", "phone": "Phone", "email": "Email", "address": "Address",
	"city": "City or province", "note": "Internal note",
}

// frozenOrderFields are refused by name, whatever the value and its JSON type,
// even the value the order already has. paid is one of them for every order: the
// shop takes online payment only and only the payment provider's own notice
// marks an order paid.
var frozenOrderFields = []string{
	"number", "placed_at", "product", "size", "qty", "items", "total", "delivery_fee", "payment", "delivery",
	"transaction_no", "paid",
}

var (
	dayShape    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	orderID     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	statusWords = []string{"placed", "confirmed", "packed", "shipped", "delivered", "cancelled"}
)

// orderBody is a PUT's body after the boundary: bad is what a 400 names (a key
// the order does not have, a value that is not text), faults what a 422 names.
type orderBody struct {
	patch  orders.Patch
	bad    map[string]string
	faults map[string]string
}

// parseOrderBody reads the object of a PUT. A key that is neither an editable
// field, a refused name nor id is a 400, and so is an editable value that is not
// a JSON string; id is read and discarded (the id in the address is the one
// changed). Every refused name and every broken value rule is collected, so the
// page shows each beside its field in one save. Text is trimmed before it is
// judged and stored.
func parseOrderBody(obj map[string]json.RawMessage) orderBody {
	b := orderBody{bad: map[string]string{}, faults: map[string]string{}}
	texts := map[string]string{}
	for key, raw := range obj {
		switch label, editable := orderLabels[key]; {
		case key == "id" || slices.Contains(frozenOrderFields, key):
		case !editable:
			b.bad[key] = notAnOrder
		case !bytes.HasPrefix(bytes.TrimSpace(raw), []byte(`"`)):
			b.bad[key] = label + " must be text."
		default:
			var s string
			if err := json.Unmarshal(raw, &s); err != nil {
				b.bad[key] = label + " must be text."
				continue
			}
			texts[key] = strings.TrimSpace(s)
		}
	}
	if len(b.bad) > 0 {
		return b
	}
	for _, name := range frozenOrderFields {
		if _, sent := obj[name]; !sent {
			continue
		}
		b.faults[name] = frozenField
		if name == "paid" {
			b.faults[name] = paidSentence
		}
	}
	for key, v := range texts {
		if fault := judgeOrderField(key, v); fault != "" {
			b.faults[key] = fault
		}
		v := v
		switch key {
		case "status":
			b.patch.Status = &v
		case "carrier":
			b.patch.Carrier = &v
		case "tracking":
			b.patch.Tracking = &v
		case "eta":
			b.patch.ETA = &v
		case "customer":
			b.patch.Customer = &v
		case "phone":
			b.patch.Phone = &v
		case "email":
			b.patch.Email = &v
		case "address":
			b.patch.Address = &v
		case "city":
			b.patch.City = &v
		case "note":
			b.patch.Note = &v
		}
	}
	return b
}

// judgeOrderField is the value rule of one editable field, as the sentence the
// page shows beside it, or "" when the value is fine. Limits count characters.
func judgeOrderField(key, v string) string {
	label, n := orderLabels[key], utf8.RuneCountInString(v)
	long := func() string { return fmt.Sprintf("%s can have at most %d characters.", label, maxLine) }
	if strings.ContainsRune(v, 0) {
		return label + " holds a character that is not allowed."
	}
	switch key {
	case "status":
		if !slices.Contains(statusWords, v) {
			return "Status must be placed, confirmed, packed, shipped, delivered or cancelled."
		}
	case "carrier", "tracking":
		if n > maxLine {
			return long()
		}
	case "eta":
		if v != "" && !validDay(v) {
			return "Write the date as year-month-day, for example 2026-11-26."
		}
	case "customer":
		if n == 0 {
			return "Enter the customer's name."
		}
		if n > maxLine {
			return long()
		}
	case "phone":
		if n > maxLine {
			return long()
		}
		if digitCount(v) < 9 {
			return "Check the phone number: it needs at least 9 digits."
		}
	case "email":
		if _, ok := auth.NormalizeEmail(v); v != "" && !ok {
			return "Enter a valid email address, or leave it empty."
		}
	case "address":
		if n == 0 {
			return "Enter the street address."
		}
		if n > maxLine {
			return long()
		}
	case "city":
		if n == 0 {
			return "Enter the city or province."
		}
		if n > maxLine {
			return long()
		}
	case "note":
		if n > maxNote {
			return "Internal note can have at most 5,000 characters."
		}
	}
	return ""
}

// validDay says whether v is a real calendar day written YYYY-MM-DD, from 1900
// to 2999. 2026-2-3, 26/11/2026 and 2026-13-40 are not.
func validDay(v string) bool {
	if !dayShape.MatchString(v) {
		return false
	}
	d, err := time.Parse("2006-01-02", v)
	return err == nil && d.Year() >= 1900 && d.Year() <= 2999
}

// digitCount is the number of ASCII digits in s.
func digitCount(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n++
		}
	}
	return n
}
