package httpx

import (
	"net/http"
	"path"
	"strings"
)

// CleanPaths answers an address that is not in its clean form (a "." or ".."
// segment, a doubled slash) with the 404 envelope. Left alone, the router answers
// it with a redirect to the cleaned address and an HTML body: a different answer
// from every other address the API does not have, which tells a stranger that
// something is behind it. An encoded slash is not a separator and passes through
// as part of one segment, as the router reads it.
func CleanPaths(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.EscapedPath(); r.Method != http.MethodConnect && strings.HasPrefix(p, "/") && !clean(p) {
			Error(w, http.StatusNotFound, "not_found", "Not found.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clean reports whether p is what path.Clean makes of it, a trailing slash kept.
func clean(p string) bool {
	np := path.Clean(p)
	if strings.HasSuffix(p, "/") && np != "/" {
		np += "/"
	}
	return np == p
}
