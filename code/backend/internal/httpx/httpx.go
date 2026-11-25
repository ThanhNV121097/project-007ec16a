// Package httpx holds what every handler of the API shares: the JSON answer,
// the error envelope and the middleware wrapped around the whole server.
package httpx

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the one shape of every non-2xx answer the API gives:
//
//	{"error": "invalid", "message": "Check the form.", "errors": {"price": "Price cannot be below 0"}}
//
// error is a stable machine word, message a sentence a person can read, and
// errors (only on 400 and 422) names each input at fault with its own
// sentence, which is the `{"errors":{field: sentence}}` the SRS writes.
// Nothing in it is ever SQL, a stack trace or another person's data.
type ErrorBody struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// JSON writes v as the answer with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// The status line is out; a failure here is a client that went away.
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes the error envelope. fields (input name → sentence) may be nil.
func Error(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	JSON(w, status, ErrorBody{Error: code, Message: message, Errors: fields})
}
