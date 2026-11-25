package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	cookieName   = "session"
	cookieMaxAge = 7 * 24 * 60 * 60
	tokenLength  = 43 // 32 bytes, unpadded base64url
)

// ErrSessionRefused is IssueSession's answer when the account was switched off,
// deleted or given a new password after the caller read it: no session exists.
var ErrSessionRefused = errors.New("auth: the account changed, no session was made")

// newToken makes a session token: 32 bytes from crypto/rand, which is 256 bits
// (the contract needs at least 128). Only its SHA-256 is ever stored.
func newToken() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, raw); err != nil {
		return "", nil, fmt.Errorf("auth: no randomness for a token: %w", err)
	}
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw), sum[:], nil
}

// tokenHash is the SHA-256 of a well-formed cookie value: exactly 43 characters
// of the base64url alphabet that decode to 32 bytes in their canonical form.
// Anything else is not a token, and no query is sent for it.
func tokenHash(cookie string) ([]byte, bool) {
	if len(cookie) != tokenLength {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(cookie)
	if err != nil || len(raw) != 32 {
		return nil, false
	}
	sum := sha256.Sum256(raw)
	return sum[:], true
}

func cookieHash(r *http.Request) ([]byte, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil, false
	}
	return tokenHash(c.Value)
}

// issueSQL is Q3. It inserts a session only if the account is still active and
// still has the hash the password was verified against; FOR SHARE keeps a
// concurrent switch-off or password reset from slipping between the check and
// the insert. No row inserted means the account changed.
const issueSQL = `INSERT INTO sessions (user_id, token_hash, expires_at)
SELECT u.id, $2, now() + interval '7 days'
FROM users u
WHERE u.id = $1 AND u.is_active AND u.password_hash = $3
FOR SHARE OF u
RETURNING expires_at`

// IssueSession makes a session for an account whose password the caller has
// just verified against passwordHash and returns the cookie value. The expiry
// is fixed at seven days, here, in the database's clock.
func (s *Service) IssueSession(ctx context.Context, q Querier, userID, passwordHash string) (string, error) {
	token, hash, err := newToken()
	if err != nil {
		return "", err
	}
	var expires time.Time
	err = q.QueryRow(ctx, issueSQL, userID, hash, passwordHash).Scan(&expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrSessionRefused
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

// SetCookie sets the session cookie: HttpOnly, SameSite=Lax, Path=/, seven
// days, and Secure when the request came over HTTPS.
func SetCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: token, Path: "/", MaxAge: cookieMaxAge,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: overHTTPS(r),
	})
}

// ClearCookie tells the browser to forget the session cookie (Max-Age=0).
func ClearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: overHTTPS(r),
	})
}

// overHTTPS is true when the connection itself is TLS or the LAST entry of
// X-Forwarded-Proto (the one the edge proxy appended) is https.
func overHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	parts := strings.Split(strings.Join(r.Header.Values("X-Forwarded-Proto"), ","), ",")
	return strings.EqualFold(strings.TrimSpace(parts[len(parts)-1]), "https")
}
