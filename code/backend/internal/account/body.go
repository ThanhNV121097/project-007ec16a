package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
)

const (
	maxBody        = 64 << 10
	wrongTypeWords = "This field has the wrong type."
	undefinedWords = "This field is not allowed."
)

// object is the request body: exactly one JSON object, key by key, so a key the
// route does not define can be refused instead of ignored.
type object map[string]json.RawMessage

// plainName is a key that is safe to echo in an answer.
var plainName = regexp.MustCompile(`^[A-Za-z0-9_]{1,40}$`)

// readObject reads a request body. It writes the answer itself when it fails:
// 415 when the media type is not application/json (parameters allowed), 413 over
// 64 KiB, 400 for anything that is not exactly one JSON object.
func readObject(w http.ResponseWriter, r *http.Request) (object, bool) {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		refuse(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send the request as JSON.", nil)
		return nil, false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	var obj object
	err := dec.Decode(&obj)
	if err == nil {
		if _, next := dec.Token(); next != io.EOF {
			err = next
			if err == nil {
				err = errors.New("trailing data")
			}
		}
	}
	if err == nil && obj == nil {
		err = errors.New("not an object")
	}
	if err == nil {
		return obj, true
	}
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		refuse(w, http.StatusRequestEntityTooLarge, "payload_too_large", "That request is too large.", nil)
		return nil, false
	}
	refuse(w, http.StatusBadRequest, "bad_request", "The request is not valid JSON.", nil)
	return nil, false
}

// undefined names every key outside allowed (only a plain name is echoed) and
// says whether there was one.
func (o object) undefined(allowed ...string) (map[string]string, bool) {
	ok := make(map[string]bool, len(allowed))
	for _, k := range allowed {
		ok[k] = true
	}
	faults, found := map[string]string{}, false
	for k := range o {
		if ok[k] {
			continue
		}
		found = true
		if plainName.MatchString(k) {
			faults[k] = undefinedWords
		}
	}
	return faults, found
}

type kind int

const (
	absent kind = iota
	isNull
	isText
	wrongType
)

// text reads a key as a string and says what it found.
func (o object) text(key string) (string, kind) {
	raw, ok := o[key]
	if !ok {
		return "", absent
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", isNull
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", wrongType
	}
	return s, isText
}

// badRequest answers 400 with the faults found, nil when there are none.
func badRequest(w http.ResponseWriter, faults map[string]string) {
	refuse(w, http.StatusBadRequest, "bad_request", "Check the form.", faults)
}
