package orders

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/stock"
)

const (
	keyHeader    = "Idempotency-Key"
	keySentence  = "Send an Idempotency-Key header of 8 to 100 letters, digits, dots, dashes or underscores."
	messageCheck = "Check the details of your order."
	messageBody  = "The order must be one JSON object with the fields the shop expects."
	maxPairs     = 50
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,100}$`)

// fieldErrors is input name -> sentence, the `errors` of the envelope.
type fieldErrors map[string]string

// itemRequest is one pair of the order. price and unit_price are tolerated and
// ignored: the browser's numbers are never trusted, the shop reads its own prices.
type itemRequest struct {
	Product   string          `json:"product"`
	Size      string          `json:"size"`
	Qty       int             `json:"qty"`
	Price     json.RawMessage `json:"price"`
	UnitPrice json.RawMessage `json:"unit_price"`
}

// orderRequest is POST /v1/orders. delivery_fee and total are tolerated and
// ignored for the same reason; every other field the contract does not define is
// refused with 400 by parseOrder, which names it.
type orderRequest struct {
	Items       []itemRequest   `json:"items"`
	Delivery    string          `json:"delivery"`
	Payment     string          `json:"payment"`
	Customer    string          `json:"customer"`
	Phone       string          `json:"phone"`
	Address     string          `json:"address"`
	City        string          `json:"city"`
	Note        string          `json:"note"`
	DeliveryFee json.RawMessage `json:"delivery_fee"`
	Total       json.RawMessage `json:"total"`
}

// decodeBody reads the request: 415 when it is not JSON, 413 over 64 KiB, 400 for
// anything that is not exactly the object the contract defines (invalid JSON, a
// wrong type, an undefined field, data after the object), with `errors` naming
// each field at fault. It writes the answer itself and says whether to go on.
func decodeBody(w http.ResponseWriter, r *http.Request) (orderRequest, bool) {
	var req orderRequest
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		httpx.Error(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send the order as application/json.", nil)
		return req, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Error(w, http.StatusRequestEntityTooLarge, "payload_too_large", "The order is too large.", nil)
			return req, false
		}
		httpx.Error(w, http.StatusBadRequest, "bad_request", messageBody, fieldErrors{"body": bodySentence})
		return req, false
	}
	req, faults := parseOrder(raw)
	if len(faults) > 0 {
		httpx.Error(w, http.StatusBadRequest, "bad_request", messageBody, faults)
		return req, false
	}
	return req, true
}

// clean trims every text and says which input holds a NUL, which PostgreSQL
// cannot store and which would otherwise fail the statement as a 500.
func (r *orderRequest) clean() (badInput string) {
	for name, p := range map[string]*string{
		"delivery": &r.Delivery, "payment": &r.Payment, "customer": &r.Customer,
		"phone": &r.Phone, "address": &r.Address, "city": &r.City, "note": &r.Note,
	} {
		*p = strings.TrimSpace(*p)
		if strings.ContainsRune(*p, 0) {
			badInput = name
		}
	}
	for i := range r.Items {
		it := &r.Items[i]
		it.Product, it.Size = strings.TrimSpace(it.Product), strings.TrimSpace(it.Size)
		if strings.ContainsRune(it.Product, 0) || strings.ContainsRune(it.Size, 0) {
			badInput = "items." + strconv.Itoa(i)
		}
	}
	return badInput
}

// countDigits is the number of ASCII digits in s: spaces, +, - and brackets are
// simply not counted.
func countDigits(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n++
		}
	}
	return n
}

func digitsOf(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// validate checks every field at once and returns one sentence per fault, each
// naming its limit where a limit is the fault. The page applies the same rules;
// the shop repeats them because a request can skip the page.
func (r *orderRequest) validate(offered []string) fieldErrors {
	f := fieldErrors{}
	switch n := len(r.Items); {
	case n == 0:
		f["items"] = "Add at least one pair."
	case n > maxPairs:
		f["items"] = "An order can hold at most 50 pairs."
	default:
		seen := map[stock.Key]bool{}
		for i, it := range r.Items {
			name := "items." + strconv.Itoa(i)
			k := stock.Key{Product: it.Product, Size: it.Size}
			switch {
			case it.Product == "" || it.Size == "" || utf8.RuneCountInString(it.Product) > 100 || utf8.RuneCountInString(it.Size) > 20:
				f[name] = "Choose a shoe and a size."
			case it.Qty < 1 || it.Qty > 1000:
				f[name] = "Quantity must be from 1 to 1,000."
			case seen[k]:
				f[name] = "This pair is listed twice."
			}
			seen[k] = true
		}
	}
	if r.Delivery != standardDelivery {
		f["delivery"] = "Delivery must be the standard option."
	}
	if !slices.Contains(offered, r.Payment) {
		f["payment"] = "That way to pay is not offered."
	}
	switch n := utf8.RuneCountInString(r.Customer); {
	case n == 0:
		f["customer"] = "Enter your name."
	case n > 100:
		f["customer"] = "Name can have at most 100 characters."
	}
	switch {
	case utf8.RuneCountInString(r.Phone) > 20:
		f["phone"] = "Phone can have at most 20 characters."
	case countDigits(r.Phone) < 9:
		f["phone"] = "Check the phone number — we text you when the pair is on its way."
	}
	switch n := utf8.RuneCountInString(r.Address); {
	case n == 0:
		f["address"] = "Enter the street address."
	case n > 200:
		f["address"] = "Street address can have at most 200 characters."
	}
	switch n := utf8.RuneCountInString(r.City); {
	case n == 0:
		f["city"] = "Enter the city or province."
	case n > 100:
		f["city"] = "City or province can have at most 100 characters."
	}
	if utf8.RuneCountInString(r.Note) > 500 {
		f["note"] = "Note can have at most 500 characters."
	}
	return f
}
