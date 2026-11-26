package admin

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
)

// The rules of the staff endpoints' input. Everything here is pure: no
// database, no request. A fault is a sentence keyed by the input's name.

const (
	maxBodyBytes = 64 << 10
	maxName      = 100
	minPassword  = 10
	maxPassword  = 100

	defaultLimit = 12
	maxLimit     = 100
	maxPage      = 2147483647

	notAccepted = "This field is not accepted."
	notText     = "This must be text."
	notFlag     = "This must be true or false."
)

// kind is the JSON type a field must have.
type kind int

const (
	text     kind = iota // a JSON string
	flag                 // true or false
	anything             // read so it can be judged or discarded
)

// What each body may hold. role is read on PUT only so it can be refused with a
// sentence; id is read and discarded (the path's id is the one acted on).
var (
	createFields = map[string]kind{"name": text, "email": text, "password": text}
	updateFields = map[string]kind{"name": text, "is_active": flag, "password": text, "role": anything, "id": anything}
)

// input is what a body said. Has tells which keys were sent: a key left out is
// unchanged, and a key sent as null is a wrong type, not a key left out.
type input struct {
	Name, Email, Password string
	Active                bool
	Has                   map[string]bool
}

var plainKey = regexp.MustCompile(`^[A-Za-z0-9_]{1,40}$`)

// readInput checks the keys and the types of one JSON object against spec. It
// returns the 400 faults, and bad when the body must be refused with 400 at
// all (a key whose name is not plain is refused without being echoed, so bad
// can be true with no fault to name).
func readInput(obj map[string]json.RawMessage, spec map[string]kind) (in input, faults map[string]string, bad bool) {
	in = input{Has: map[string]bool{}}
	faults = map[string]string{}
	for key, raw := range obj {
		want, known := spec[key]
		if !known {
			bad = true
			if plainKey.MatchString(key) {
				faults[key] = notAccepted
			}
			continue
		}
		in.Has[key] = true
		raw = bytes.TrimSpace(raw)
		switch want {
		case text:
			var s string
			if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &s) != nil {
				bad, faults[key] = true, notText
				continue
			}
			switch key {
			case "name":
				in.Name = s
			case "email":
				in.Email = s
			case "password":
				in.Password = s
			}
		case flag:
			switch string(raw) {
			case "true":
				in.Active = true
			case "false":
			default:
				bad, faults[key] = true, notFlag
			}
		}
	}
	return in, faults, bad
}

// judgeName trims the name and checks it: 1 to 100 characters, no control
// character (PostgreSQL cannot store a NUL).
func judgeName(raw string) (name, fault string) {
	name = strings.TrimSpace(raw)
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		return "", "A name is needed"
	case n > maxName:
		return "", "The name can have at most 100 characters."
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", "The name cannot contain control characters."
		}
	}
	return name, ""
}

// judgeEmail normalises the address (trimmed, ASCII, lower case) and checks its
// shape; whether an account holds it is the handler's question.
func judgeEmail(raw string) (email, fault string) {
	if strings.TrimSpace(raw) == "" {
		return "", "An email address is needed"
	}
	email, ok := auth.NormalizeEmail(raw)
	if !ok {
		return "", "That does not look like an email address."
	}
	return email, ""
}

// judgePassword checks 10 to 100 characters, counted as characters and never
// trimmed.
func judgePassword(raw string) (fault string) {
	switch n := utf8.RuneCountInString(raw); {
	case n < minPassword:
		return "The password needs at least 10 characters."
	case n > maxPassword:
		return "The password can have at most 100 characters."
	}
	return ""
}

// judgeCreate judges every field of a POST and collects every fault, so one
// answer names them all. It returns the trimmed name and normalised email.
func judgeCreate(in input) (name, email string, faults map[string]string) {
	faults = map[string]string{}
	var f string
	if name, f = judgeName(in.Name); f != "" {
		faults["name"] = f
	}
	if email, f = judgeEmail(in.Email); f != "" {
		faults["email"] = f
	}
	if f = judgePassword(in.Password); f != "" {
		faults["password"] = f
	}
	return name, email, faults
}

// judgeUpdate judges the fields a PUT sent and leaves in.Name trimmed. role is
// refused whatever its value or type.
func judgeUpdate(in *input) map[string]string {
	faults := map[string]string{}
	if in.Has["name"] {
		name, f := judgeName(in.Name)
		if f != "" {
			faults["name"] = f
		}
		in.Name = name
	}
	if in.Has["password"] {
		if f := judgePassword(in.Password); f != "" {
			faults["password"] = f
		}
	}
	if in.Has["role"] {
		faults["role"] = "The role cannot be changed."
	}
	return faults
}

// readPaging reads page and limit as the project's list scheme does: page is
// digits only from 1; limit is digits only from 1, default 12, and above 100 is
// served as 100. Any other parameter is ignored.
func readPaging(q url.Values) (page, limit int64, faults map[string]string) {
	page, limit = 1, defaultLimit
	faults = map[string]string{}
	if vs, ok := q["page"]; ok {
		n, err := strconv.ParseUint(vs[0], 10, 64)
		if !digits(vs[0]) || err != nil || n < 1 || n > maxPage {
			faults["page"] = "page must be a whole number from 1."
		} else {
			page = int64(n)
		}
	}
	if vs, ok := q["limit"]; ok {
		n, err := strconv.ParseUint(vs[0], 10, 64)
		switch {
		case !digits(vs[0]):
			faults["limit"] = "limit must be a whole number from 1."
		case err != nil || n > maxLimit:
			limit = maxLimit
		case n < 1:
			faults["limit"] = "limit must be a whole number from 1."
		default:
			limit = int64(n)
		}
	}
	return page, limit, faults
}

func digits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
