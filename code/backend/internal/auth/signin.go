package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/ratelimit"
	"github.com/jackc/pgx/v5"
)

// Sign-in limits: five failures in fifteen minutes, for one lower-cased email
// or one client address, lock it for fifteen minutes.
const (
	signInLimit   = 5
	signInWindow  = 15 * time.Minute
	signInLock    = 15 * time.Minute
	maxBodyBytes  = 64 << 10
	signInTimeout = 15 * time.Second
)

// Register mounts the three auth routes. Every path is a literal /v1/... with
// no /api prefix: the edge proxy strips it. A wrong method on these paths falls
// to the server's catch-all and answers the JSON 404.
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/sign-in", s.signIn)
	mux.HandleFunc("POST /v1/auth/sign-out", s.signOut)
	mux.HandleFunc("GET /v1/auth/me", s.me)
}

// account is what the three auth endpoints say about a person.
type account struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  Role   `json:"role"`
}

type signInBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// stored is the row sign-in reads (Q2).
type stored struct {
	id, name, role, hash string
	active               bool
}

const wrongCredentials = "Wrong email or password."

// signIn is POST /v1/auth/sign-in. The order of its checks is the contract:
// media type, body, fields, limits, lookup, ONE verification, the failure
// counters, the limits again, the session.
func (s *Service) signIn(w http.ResponseWriter, r *http.Request) {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		refuse(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send the request as JSON.", nil)
		return
	}
	var in signInBody
	if !decodeJSON(w, r, &in) {
		return
	}
	if faults := checkFields(in); faults != nil {
		refuse(w, http.StatusUnprocessableEntity, "invalid", "Check the form.", faults)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), signInTimeout)
	defer cancel()
	emailKey := strings.ToLower(strings.TrimSpace(in.Email))
	emailSubject := ratelimit.Subject(ratelimit.SignInEmail, emailKey)
	addressSubject := ratelimit.Subject(ratelimit.SignInAddress, httpx.ClientAddr(r))

	if wait, err := s.lockedFor(ctx, emailSubject, addressSubject); err != nil {
		s.internal(w, r, err)
		return
	} else if wait > 0 {
		tooMany(w, wait)
		return
	}

	row, found, err := s.account(ctx, emailKey)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	// One verification, always: the row's hash when there is a row (even a
	// switched-off one, so it costs what a wrong password costs), the dummy
	// hash otherwise.
	hash := s.dummy
	if found {
		hash = row.hash
	}
	match, err := s.verify(ctx, hash, in.Password)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	if !found || !row.active || !match {
		s.notSignedIn(ctx, w, r, emailSubject, addressSubject)
		return
	}

	// A lock may have begun while the hash ran: a right password completes only
	// if fewer than five failures were recorded by now.
	if wait, err := s.lockedFor(ctx, emailSubject, addressSubject); err != nil {
		s.internal(w, r, err)
		return
	} else if wait > 0 {
		tooMany(w, wait)
		return
	}

	token, err := s.IssueSession(ctx, s.db, row.id, row.hash)
	if errors.Is(err, ErrSessionRefused) {
		s.notSignedIn(ctx, w, r, emailSubject, addressSubject)
		return
	}
	if err != nil {
		s.internal(w, r, err)
		return
	}
	SetCookie(w, r, token)
	reply(w, http.StatusOK, account{Email: emailKey, Name: DisplayName(row.name, emailKey), Role: Role(row.role)})
}

// notSignedIn counts the failure on both scopes and answers the one 401 every
// kind of failure gets: unknown email, wrong password, switched-off account,
// and an admin that was never made.
func (s *Service) notSignedIn(ctx context.Context, w http.ResponseWriter, r *http.Request, subjects ...[]byte) {
	scopes := []string{ratelimit.SignInEmail, ratelimit.SignInAddress}
	for i, subject := range subjects {
		if err := ratelimit.Fail(ctx, s.db, scopes[i], subject, signInLimit, signInWindow, signInLock); err != nil {
			s.internal(w, r, err)
			return
		}
	}
	refuse(w, http.StatusUnauthorized, "invalid_credentials", wrongCredentials, nil)
}

// lockedFor is the longer of the two waits, zero when neither is locked.
func (s *Service) lockedFor(ctx context.Context, emailSubject, addressSubject []byte) (time.Duration, error) {
	byEmail, err := ratelimit.Check(ctx, s.db, ratelimit.SignInEmail, emailSubject)
	if err != nil {
		return 0, err
	}
	byAddress, err := ratelimit.Check(ctx, s.db, ratelimit.SignInAddress, addressSubject)
	if err != nil {
		return 0, err
	}
	return max(byEmail, byAddress), nil
}

// tooMany answers 429 with the wait in whole seconds (1 to 900) in Retry-After
// and in minutes, rounded up, in the sentence.
func tooMany(w http.ResponseWriter, wait time.Duration) {
	secs := min(max(int(math.Ceil(wait.Seconds())), 1), int(signInLock.Seconds()))
	mins := max(int(math.Ceil(float64(secs)/60)), 1)
	unit := " minutes."
	if mins == 1 {
		unit = " minute."
	}
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	refuse(w, http.StatusTooManyRequests, "rate_limited", "Too many tries. Please wait "+strconv.Itoa(mins)+unit, nil)
}

// accountSQL is Q2.
const accountSQL = `SELECT id::text, name, role, is_active, password_hash FROM users WHERE email = $1`

// account reads the row for a lower-cased email. An address with a NUL byte can
// never have been stored (and Postgres refuses it as text), so it finds no row
// and takes the normal 401 path.
func (s *Service) account(ctx context.Context, email string) (stored, bool, error) {
	if strings.IndexByte(email, 0) >= 0 {
		return stored{}, false, nil
	}
	var row stored
	err := s.db.QueryRow(ctx, accountSQL, email).Scan(&row.id, &row.name, &row.role, &row.active, &row.hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return stored{}, false, nil
	}
	if err != nil {
		return stored{}, false, err
	}
	return row, true, nil
}

// checkFields is the 422 check: every fault at once, nothing hashed and nothing
// counted. Lengths are in characters.
func checkFields(in signInBody) map[string]string {
	faults := map[string]string{}
	switch n := utf8.RuneCountInString(strings.TrimSpace(in.Email)); {
	case n == 0:
		faults["email"] = "An email address is needed"
	case n > 200:
		faults["email"] = "That email address is too long"
	}
	switch n := utf8.RuneCountInString(in.Password); {
	case n == 0:
		faults["password"] = "A password is needed"
	case n > 100:
		faults["password"] = "That password is too long"
	}
	if len(faults) == 0 {
		return nil
	}
	return faults
}

var plainName = regexp.MustCompile(`^[A-Za-z0-9_]{1,40}$`)

// decodeJSON reads exactly one JSON object of at most 64 KiB into dst, refusing
// a field dst does not define. It writes the answer itself when it fails: 400
// for bad JSON, a wrong type, an unknown field or trailing data, 413 for size.
// A field is named in the answer only when its name is plain, so the answer
// never echoes whatever a client typed.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err == nil {
		if _, next := dec.Token(); next != io.EOF {
			err = next
			if err == nil {
				err = errors.New("trailing data")
			}
		}
	}
	if err == nil {
		return true
	}

	var tooBig *http.MaxBytesError
	var wrongType *json.UnmarshalTypeError
	fault := func(name, sentence string) map[string]string {
		if !plainName.MatchString(name) {
			return nil
		}
		return map[string]string{name: sentence}
	}
	switch {
	case errors.As(err, &tooBig):
		refuse(w, http.StatusRequestEntityTooLarge, "payload_too_large", "That request is too large.", nil)
	case errors.As(err, &wrongType):
		refuse(w, http.StatusBadRequest, "bad_request", "Check the form.", fault(wrongType.Field, "This field has the wrong type."))
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		name, _ := strconv.Unquote(strings.TrimPrefix(err.Error(), "json: unknown field "))
		refuse(w, http.StatusBadRequest, "bad_request", "Check the form.", fault(name, "This field is not allowed."))
	default:
		refuse(w, http.StatusBadRequest, "bad_request", "The request is not valid JSON.", nil)
	}
	return false
}

// signOut is POST /v1/auth/sign-out. It takes no body and answers 204 always:
// with a live session, an unknown one, an expired one or none. Only when the
// delete itself fails is it a 500, and then the cookie stays, so the person can
// try again.
func (s *Service) signOut(w http.ResponseWriter, r *http.Request) {
	if hasBody(w, r) {
		refuse(w, http.StatusBadRequest, "bad_request", "This request takes no body.", nil)
		return
	}
	if hash, ok := cookieHash(r); ok {
		ctx, cancel := context.WithTimeout(r.Context(), lookupTimeout)
		defer cancel()
		if _, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash); err != nil {
			s.internal(w, r, err)
			return
		}
	}
	ClearCookie(w, r)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// hasBody is true when the request carries at least one byte of body.
func hasBody(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case r.ContentLength > 0:
		return true
	case r.ContentLength == 0:
		return false
	}
	var one [1]byte
	n, _ := http.MaxBytesReader(w, r.Body, 1).Read(one[:])
	return n > 0
}

// me is GET /v1/auth/me: who Resolve found, from the context, with no second
// query. The account's own name, or the one made from its email.
func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	u, ok := FromContext(r.Context())
	if !ok {
		refuse(w, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.", nil)
		return
	}
	reply(w, http.StatusOK, account{Email: u.Email, Name: u.DisplayName(), Role: u.Role})
}

// internal answers a failed statement: the generic 500 and nothing of the cause
// (no SQL, no host), which goes to the log under the route pattern.
func (s *Service) internal(w http.ResponseWriter, r *http.Request, err error) {
	logFailure(s.log, "auth request failed", r.Pattern, err)
	refuse(w, http.StatusInternalServerError, "internal", internalMessage, nil)
}
