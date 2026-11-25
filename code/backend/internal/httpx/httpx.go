// Package httpx holds what every handler of the API shares: the JSON answer,
// the error envelope and the middleware wrapped around the whole server.
package httpx

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the one shape of every non-2xx answer the API gives:
//
//	{"error": {"code": "not_found", "message": "...", "fields": {"page": "..."}}}
//
// code is a stable machine word, message a sentence a person can read, and
// fields names each input at fault with its own sentence (422, 400). Nothing
// in it is ever SQL, a stack trace or another person's data.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is the inside of ErrorBody.
type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// JSON writes v as the answer with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// The status line is out; a failure here is a client that went away.
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes the error envelope. fields may be nil.
func Error(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	JSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message, Fields: fields}})
}
