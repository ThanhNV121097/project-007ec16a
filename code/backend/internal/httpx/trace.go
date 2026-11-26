package httpx

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// Trace is what the access log line learns about a request on its way through
// the chain: the route pattern that answered it and the role of the person
// signed in. AccessLog puts one in the request context; Resolve fills Role and
// RecordRoute fills Pattern. Both are written by the goroutine that serves the
// request, so no lock is needed.
type Trace struct {
	Role    string
	Pattern string
}

type traceKey struct{}

// TraceFrom returns the request's Trace, or a throw-away one when AccessLog is
// not mounted (a test of a single handler), so callers never test for nil.
func TraceFrom(ctx context.Context) *Trace {
	if t, ok := ctx.Value(traceKey{}).(*Trace); ok {
		return t
	}
	return &Trace{}
}

// RecordRoute wraps the mux directly. After the mux has answered it copies the
// route pattern it matched (the mux sets Request.Pattern on the request it was
// given) into the Trace, so the access log names the route and never the raw
// path an attacker chose.
func RecordRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		TraceFrom(r.Context()).Pattern = r.Pattern
	})
}

// ClientAddr is the address every limit counts: the right-most non-empty entry
// of all X-Forwarded-For header lines (the one the edge proxy appended; the
// left-most entries are whatever the client wrote), when it is an IP address,
// in its canonical text; otherwise the host part of the connection's address.
func ClientAddr(r *http.Request) string {
	if lines := r.Header.Values("X-Forwarded-For"); len(lines) > 0 {
		parts := strings.Split(strings.Join(lines, ","), ",")
		for i := len(parts) - 1; i >= 0; i-- {
			entry := strings.TrimSpace(parts[i])
			if entry == "" {
				continue
			}
			if ip, err := netip.ParseAddr(entry); err == nil {
				return ip.Unmap().String()
			}
			break
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
