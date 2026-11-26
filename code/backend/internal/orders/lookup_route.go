package orders

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"math"
	"mime"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
)

// POST /v1/order-lookups: a customer who typed an order number and the phone it
// was placed with gets the order's id, number, time and a NEW access token, the
// one GET /v1/orders/{id} reads with. The match, the two counters and the lock
// on the number are Attempt's (lookup.go); this file is the request's side of it.

const (
	lookupMediaSentence    = "Send the lookup as application/json."
	lookupTooLargeSentence = "The request is too large."
	lookupCheckSentence    = "Check the number and the phone."
	lookupBodySentence     = "Send the lookup as one JSON object."
	lookupNeededSentence   = "This field is needed."
	lookupNoMatchSentence  = "No order matches that number and phone."
)

// lookupKeyName is a key that is safe to echo in an answer: only such a key is
// named in `errors`, whatever else a body holds.
var lookupKeyName = regexp.MustCompile(`^[A-Za-z0-9_]{1,40}$`)

// lookedUp is the answer to a right pair: these four keys and nothing else. It is
// built field by field and never from the order, so no private value can reach it.
type lookedUp struct {
	ID       string `json:"id"`
	Number   string `json:"number"`
	PlacedAt string `json:"placed_at"`
	Token    string `json:"token"`
}

// readLookup reads the request: 415 when it is not JSON, 413 over 64 KiB, 400 for
// anything that is not exactly one object with a string `number` and a string
// `phone` and no other key. An empty string is present and is judged by the match.
// It writes the answer itself when it refuses, before anything is counted.
func readLookup(w http.ResponseWriter, r *http.Request) (Pair, bool) {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		httpx.Error(w, http.StatusUnsupportedMediaType, "unsupported_media_type", lookupMediaSentence, nil)
		return Pair{}, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Error(w, http.StatusRequestEntityTooLarge, "payload_too_large", lookupTooLargeSentence, nil)
			return Pair{}, false
		}
		httpx.Error(w, http.StatusBadRequest, "bad_request", lookupCheckSentence, fieldErrors{"body": lookupBodySentence})
		return Pair{}, false
	}
	fields, err := object(raw)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", lookupCheckSentence, fieldErrors{"body": lookupBodySentence})
		return Pair{}, false
	}

	var p Pair
	texts := map[string]*string{"number": &p.Number, "phone": &p.Phone}
	faults := fieldErrors{}
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		dst, defined := texts[name]
		switch {
		case !defined:
			if len(faults) < maxFaults && lookupKeyName.MatchString(name) {
				faults[name] = undefinedSentence
			}
			if len(faults) == 0 {
				faults["body"] = undefinedSentence
			}
		case !lookupText(dst, fields[name]):
			faults[name] = textSentence
		}
	}
	for name := range texts {
		if _, present := fields[name]; !present {
			faults[name] = lookupNeededSentence
		}
	}
	if len(faults) > 0 {
		httpx.Error(w, http.StatusBadRequest, "bad_request", lookupCheckSentence, faults)
		return Pair{}, false
	}
	return p, true
}

// lookupText reads a JSON string into dst. null, a number, an object and a list
// are not text.
func lookupText(dst *string, raw json.RawMessage) bool {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return false
	}
	return json.Unmarshal(raw, dst) == nil
}

// lookupWait sets Retry-After to the wait in whole seconds, from 1 to the lock's
// own 15 minutes, and returns the wait in words for the sentence.
func lookupWait(w http.ResponseWriter, wait time.Duration) string {
	secs := min(max(int(math.Ceil(wait.Seconds())), 1), int(lookupLock.Seconds()))
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	if mins := (secs + 59) / 60; mins > 1 {
		return strconv.Itoa(mins) + " minutes"
	}
	return "1 minute"
}

// lookup answers POST /v1/order-lookups. Every answer of it, the refusals too, is
// never cached; the header is set first so no path can forget it. The route is
// public and reads no session: a guest, a customer and the team are alike.
//
// A body the route refuses (415, 413, 400) is not a try and counts nothing. Past
// that, Attempt decides: a right pair gets a new token (one row in order_tokens,
// earlier tokens stay valid), a miss counts one failure and is the one 404, a
// locked number or address is the 429 with its wait. Anything else, an
// unreachable database included, is a 500: a failed statement is a defect, and a
// 503 would tell the client the same request will work shortly.
func (s *service) lookup(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	pair, ok := readLookup(w, r)
	if !ok {
		return
	}
	var token string
	m, err := Attempt(r.Context(), s.db, httpx.ClientAddr(r), pair, func(ctx context.Context, tx pgx.Tx, m Matched) error {
		var err error
		token, err = addToken(ctx, tx, m.ID)
		return err
	})
	var locked *LockedError
	switch {
	case errors.As(err, &locked):
		words := lookupWait(w, locked.Wait)
		httpx.Error(w, http.StatusTooManyRequests, "rate_limited", "Too many tries. Please wait "+words+".", nil)
	case errors.Is(err, ErrNoMatch):
		httpx.Error(w, http.StatusNotFound, "not_found", lookupNoMatchSentence, nil)
	case err != nil:
		s.internal(w, r, "order lookup", err)
	default:
		httpx.JSON(w, http.StatusOK, lookedUp{
			ID: m.ID, Number: m.Number, Token: token,
			PlacedAt: m.PlacedAt.In(ict).Format("2006-01-02T15:04:05-07:00"),
		})
	}
}
