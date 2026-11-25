package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Middleware wraps a handler.
type Middleware func(http.Handler) http.Handler

// Recover turns a panic in a handler into a logged 500 with the generic
// envelope, so one bad request never takes the process down.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if v == http.ErrAbortHandler {
					panic(v)
				}
				log.Error("panic in handler", "method", r.Method, "path", r.URL.Path,
					"panic", fmt.Sprint(v), "stack", string(debug.Stack()))
				Error(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.", nil)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

<<<<<<< refs/heads/feature/checkout-and-placing-an-order-be-v2
// AccessLog writes one line per request: the request id, method, path (never
// the query string, which can carry a token), status and duration.
=======
// AccessLog writes one line per request: method, the route pattern that
// matched (empty when none did; never the raw path or the query string, which
// an attacker chooses and which can carry a token), status, duration and the
// role of the person signed in (empty for a visitor). The pattern and the role
// reach it through the request's Trace.
>>>>>>> refs/heads/main
func AccessLog(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			trace := &Trace{}
			r = r.WithContext(context.WithValue(r.Context(), traceKey{}, trace))
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
<<<<<<< refs/heads/feature/checkout-and-placing-an-order-be-v2
			log.Info("request", "request_id", RequestIDFrom(r.Context()), "method", r.Method, "path", r.URL.Path,
				"status", rec.status, "ms", time.Since(start).Milliseconds())
=======
			log.Info("request", "method", r.Method, "pattern", trace.Pattern,
				"status", rec.status, "ms", time.Since(start).Milliseconds(), "role", trace.Role)
>>>>>>> refs/heads/main
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the real writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// CORS lets the listed browser origins call the API with credentials. With no
// origin listed it is the identity: in deployment the browser reaches the API
// on its own origin under /api, and nothing cross-origin is needed.
func CORS(allowed []string) Middleware {
	set := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		set[o] = true
	}
	return func(next http.Handler) http.Handler {
		if len(set) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Add("Vary", "Origin")
			if origin := r.Header.Get("Origin"); set[origin] {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					h.Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
					h.Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
