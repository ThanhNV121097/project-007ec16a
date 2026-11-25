package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
)

// RequestIDHeader is the trace header (docs/architecture/services.md §2.1).
const RequestIDHeader = "X-Request-Id"

// requestIDPattern is what a caller may send as its own id: 8 to 64 letters,
// digits, dots, dashes or underscores. Anything else is replaced, so a header
// can never carry a line break, a token or a phone number into the log.
var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

type requestIDKey struct{}

// RequestID gives the request an id and echoes it on the answer, whatever the
// answer is (a 404, a refusal, a recovered panic): the caller's own when it
// matches the pattern, otherwise 16 random hexadecimal characters. The id is also
// put in the request context, where RequestIDFrom reads it for the access log and
// for the line a failure is logged on. It wraps the whole server, outside Recover.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if !requestIDPattern.MatchString(id) {
				id = newRequestID()
			}
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
		})
	}
}

// RequestIDFrom is the id RequestID gave the request, or "" when it did not run.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// newRequestID is 8 random bytes written as 16 lower-case hexadecimal characters.
// crypto/rand.Read never returns an error: a machine that cannot give random
// bytes stops the program.
func newRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
