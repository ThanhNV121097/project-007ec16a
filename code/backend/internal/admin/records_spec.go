package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// The rules of what the shop's team may write to the shoes, lines, categories
// and reviews. They mirror the collections of the frontend's schema.json and
// are everything pure: no database, no request. A fault is a sentence keyed by
// the field's JSON name, short enough to sit beside the field it is about.

const (
	recMaxText  = 200
	recMaxLong  = 5000
	recMaxMoney = 1_000_000_000_000

	recNotPart  = "This field is not part of the record."
	recDerived  = "The shop works this out; it cannot be set."
	recNoID     = "The shop makes the id."
	recNotText  = "Enter text."
	recNotChars = "This holds a character that is not allowed."
	recNotEmpty = "This cannot be empty."
)

// recKind is what a field holds.
type recKind int

const (
	recText    recKind = iota // text, 200 characters
	recLong                   // long text, 5,000 characters
	recPicture                // a picture's address, 200 characters
	recWhole                  // a whole number, 0 or more (a price)
	recFlag                   // true or false
	recChoice                 // one of a closed list
	recDay                    // YYYY-MM-DD
	recRef                    // the id of a record of another table; "" clears it
	recStars                  // a whole number 1 to 5, or null
	recYear                   // a whole number above 0, or null
)

// recField is one writable field of a collection.
type recField struct {
	name, column, label string
	kind                recKind
	required            bool
	choices             []string
	// refTable is the table a recRef names; unknown is the sentence for an id
	// that names nothing there, missing the one for a required field left out.
	refTable, unknown, missing string
}

// place is the SQL placeholder of the field's n-th argument.
func (f recField) place(n int) string {
	if f.kind == recDay {
		return fmt.Sprintf("$%d::date", n)
	}
	return fmt.Sprintf("$%d", n)
}

func (f recField) missingSentence() string {
	if f.missing != "" {
		return f.missing
	}
	return f.label + " is needed"
}

// recSpec is one collection the team writes.
type recSpec struct {
	name    string // the collection and the path segment
	table   string
	fields  []recField
	derived map[string]bool // fields the shop works out: sending one is a fault
	shoes   bool            // carries the pairs in each size
}

func (s *recSpec) field(name string) (recField, bool) {
	for _, f := range s.fields {
		if f.name == name {
			return f, true
		}
	}
	return recField{}, false
}

func derived(names ...string) map[string]bool {
	m := map[string]bool{"_source": true, "_invented": true}
	for _, n := range names {
		m[n] = true
	}
	return m
}

func txt(name, label string) recField { return recField{name: name, column: name, label: label, kind: recText} }
func long(name, label string) recField {
	return recField{name: name, column: name, label: label, kind: recLong}
}
func picture(name, label string) recField {
	return recField{name: name, column: name, label: label, kind: recPicture}
}
func choice(name, label string, choices ...string) recField {
	return recField{name: name, column: name, label: label, kind: recChoice, choices: choices}
}

var shoeColours = []string{"", "black", "white", "pink", "purple", "brown", "yellow", "blue", "grey", "green"}

var recSpecs = []*recSpec{
	{
		name: "products", table: "products", shoes: true,
		derived: derived("sizes_low", "sizes_out", "stock", "rating", "review_count"),
		fields: []recField{
			{name: "name", column: "name", label: "Name", kind: recText, required: true},
			txt("model", "Model"),
			{name: "line", column: "line_id", label: "Line", kind: recRef, refTable: "lines", unknown: "That line does not exist."},
			{name: "category", column: "category_id", label: "Category", kind: recRef, refTable: "categories", unknown: "That category does not exist."},
			{name: "price", column: "price", label: "Price", kind: recWhole, required: true},
			txt("price_listed", "Listed price"),
			choice("status", "Status", "live", "draft"),
			{name: "featured", column: "featured", label: "Opens the home page", kind: recFlag},
			{name: "is_new", column: "is_new", label: "New this month", kind: recFlag},
			long("description", "Description"), txt("code", "Style code"), txt("colourway", "Colourway"),
			choice("colour", "Main colour", shoeColours...),
			txt("swatch", "Swatch"), txt("upper", "Upper"), txt("lining", "Lining"), txt("sole", "Sole"), txt("fit", "Fit"),
			long("care", "Care"),
			{name: "released", column: "released", label: "In the shop since", kind: recDay},
			picture("image", "Side view"), picture("image_back", "Front pair"), picture("image_detail", "Back pair"),
		},
	},
	{
		name: "lines", table: "lines", derived: derived(),
		fields: []recField{
			{name: "name", column: "name", label: "Name", kind: recText, required: true},
			long("blurb", "Blurb"),
			{name: "since", column: "since", label: "First made", kind: recYear},
			long("fit_note", "How it fits"),
		},
	},
	{
		name: "categories", table: "categories", derived: derived(),
		fields: []recField{
			{name: "name", column: "name", label: "Name", kind: recText, required: true},
			long("blurb", "Blurb"),
			picture("image", "Picture"),
		},
	},
	{
		name: "reviews", table: "reviews", derived: derived(),
		fields: []recField{
			{name: "product", column: "product_id", label: "Shoe", kind: recRef, required: true, refTable: "products",
				unknown: "That shoe does not exist.", missing: "Choose the shoe this review is for"},
			{name: "name", column: "name", label: "Customer name", kind: recText, required: true},
			{name: "rating", column: "rating", label: "Stars", kind: recStars},
			{name: "date", column: "date", label: "Date", kind: recDay},
			txt("size_bought", "Size bought"),
			choice("fit", "How it fit", "", "small", "true", "large"),
			long("body", "Review"),
			choice("status", "Status", "live", "hidden"),
		},
	},
}

var recSpecByName = func() map[string]*recSpec {
	m := make(map[string]*recSpec, len(recSpecs))
	for _, s := range recSpecs {
		m[s.name] = s
	}
	return m
}()

// recValue is one field's judged value: arg is what the statement binds (a
// string, an int64, a bool, or nil for NULL).
type recValue struct {
	field recField
	arg   any
}

// recInput is what a body said, judged without the database.
type recInput struct {
	values []recValue // in the spec's field order
	faults map[string]string
	stock  stockPlan
}

func (in recInput) value(name string) (recValue, bool) {
	for _, v := range in.values {
		if v.field.name == name {
			return v, true
		}
	}
	return recValue{}, false
}

// faultName is a key of a body as it may be shown back: a plain one as it is,
// anything else cut short, so a fault never echoes a long or odd key.
func faultName(key string) string {
	if plainKey.MatchString(key) {
		return key
	}
	return "request"
}

// read judges a body against the spec. On a create every required field must be
// sent; on a change only what is sent is judged and the body's id is ignored.
func (s *recSpec) read(obj map[string]json.RawMessage, create bool) recInput {
	in := recInput{faults: map[string]string{}}
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, known := s.field(key)
		switch {
		case key == "id":
			if create {
				in.faults["id"] = recNoID
			}
		case s.derived[key]:
			in.faults[faultName(key)] = recDerived
		case s.shoes && (key == "pairs" || key == "seen_pairs" || key == "sizes"):
		case !known:
			in.faults[faultName(key)] = recNotPart
		}
	}
	for _, f := range s.fields {
		raw, sent := obj[f.name]
		if !sent {
			if create && f.required {
				in.faults[f.name] = f.missingSentence()
			}
			continue
		}
		arg, fault := judgeValue(f, raw)
		if fault != "" {
			in.faults[f.name] = fault
			continue
		}
		in.values = append(in.values, recValue{field: f, arg: arg})
	}
	if s.shoes {
		in.stock = readStock(obj, create, in.faults)
	}
	return in
}

func isNull(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }

// isBlankText says whether a value is text with nothing in it: for stars and a
// year, "" means what null means, no value.
func isBlankText(raw json.RawMessage) bool {
	var v string
	return json.Unmarshal(raw, &v) == nil && strings.TrimSpace(v) == ""
}

// judgeValue reads one value of a field. A null clears an optional field and is
// a fault for a required one, or for one that always holds something.
func judgeValue(f recField, raw json.RawMessage) (any, string) {
	if isNull(raw) {
		return judgeNull(f)
	}
	switch f.kind {
	case recText, recLong, recPicture:
		return judgeText(f, raw)
	case recWhole:
		n, ok := wholeNumber(raw)
		switch {
		case !ok:
			return nil, "Enter a whole number."
		case n < 0:
			return nil, f.label + " cannot be below 0"
		case n > recMaxMoney:
			return nil, f.label + " is too large"
		}
		return n, ""
	case recFlag:
		var b bool
		if json.Unmarshal(raw, &b) != nil {
			return nil, "This must be true or false."
		}
		return b, ""
	case recChoice:
		var v string
		if json.Unmarshal(raw, &v) != nil || !inList(f.choices, v) {
			return nil, "Choose one of the choices."
		}
		return v, ""
	case recDay:
		var v string
		if json.Unmarshal(raw, &v) != nil || !isDay(v) {
			return nil, "Use a date written YYYY-MM-DD."
		}
		return v, ""
	case recRef:
		var v string
		if json.Unmarshal(raw, &v) != nil {
			return nil, recNotText
		}
		v = strings.TrimSpace(v)
		if v == "" {
			if f.required {
				return nil, f.missingSentence()
			}
			return nil, ""
		}
		if !recIDPattern.MatchString(v) || len(v) > 100 {
			return nil, f.unknown
		}
		return v, ""
	case recStars:
		if isBlankText(raw) {
			return nil, ""
		}
		n, ok := wholeNumber(raw)
		if !ok || n < 1 || n > 5 {
			return nil, "Stars must be a whole number from 1 to 5."
		}
		return n, ""
	case recYear:
		if isBlankText(raw) {
			return nil, ""
		}
		n, ok := wholeNumber(raw)
		if !ok || n < 1 || n > 9999 {
			return nil, "Enter a year, a whole number above 0."
		}
		return n, ""
	}
	return nil, recNotText
}

func judgeNull(f recField) (any, string) {
	switch f.kind {
	case recText, recLong, recPicture:
		if f.required {
			return nil, f.missingSentence()
		}
		return "", ""
	case recChoice:
		if inList(f.choices, "") {
			return "", ""
		}
	case recRef:
		if f.required {
			return nil, f.missingSentence()
		}
		return nil, ""
	case recStars, recYear:
		return nil, ""
	}
	return nil, recNotEmpty
}

func judgeText(f recField, raw json.RawMessage) (any, string) {
	var v string
	if json.Unmarshal(raw, &v) != nil {
		return nil, recNotText
	}
	v = strings.TrimSpace(v)
	if !utf8.ValidString(v) || strings.ContainsRune(v, 0) {
		return nil, recNotChars
	}
	if v == "" && f.required {
		return nil, f.missingSentence()
	}
	limit := recMaxText
	if f.kind == recLong {
		limit = recMaxLong
	}
	if n := utf8.RuneCountInString(v); n > limit {
		return nil, fmt.Sprintf("%s can have at most %s characters.", f.label, groupDigits(limit))
	}
	if f.kind == recPicture && strings.HasPrefix(strings.ToLower(v), "data:") {
		return nil, "Choose a picture to upload."
	}
	return v, ""
}

// groupDigits writes 5000 as 5,000.
func groupDigits(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	return s[:len(s)-3] + "," + s[len(s)-3:]
}

// wholeNumber reads a JSON number that is a whole number: 3, never 2.5, 1e3 or
// "3".
func wholeNumber(raw json.RawMessage) (int64, bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return 0, false
	}
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := strconv.ParseInt(n.String(), 10, 64)
	return i, err == nil
}

func inList(list []string, v string) bool {
	for _, c := range list {
		if c == v {
			return true
		}
	}
	return false
}

var dayShape = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

func isDay(v string) bool {
	if !dayShape.MatchString(v) {
		return false
	}
	d, err := time.Parse("2006-01-02", v)
	return err == nil && d.Year() >= 1
}

var recIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// recValidID says whether a path segment can be a record's id; one that cannot
// never reaches the database.
func recValidID(id string) bool { return len(id) <= 100 && recIDPattern.MatchString(id) }
