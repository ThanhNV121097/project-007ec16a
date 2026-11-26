package orders

import (
	"context"
	"errors"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/ratelimit"
	"github.com/jackc/pgx/v5"
)

// The number-and-phone matcher has ONE home, this file. "Add an order" to an
// account (internal/account) calls Attempt through AddToAccount, and the order
// lookup of the track story calls the same Attempt with its own hit, so the two
// can never match differently and their failures count on the same counters.
//
// This is the only code that names the order_lookup_* scopes.

// Five misses in fifteen minutes, for one order number or one network address,
// lock that number or address for fifteen minutes.
const (
	lookupLimit   = 5
	lookupWindow  = 15 * time.Minute
	lookupLock    = 15 * time.Minute
	lookupTimeout = 5 * time.Second
	maxNumberLen  = 18
)

// Pair is the number and the phone as they were typed.
type Pair struct{ Number, Phone string }

// Matched is the order a right pair found. It never holds the phone.
type Matched struct {
	ID, Number, Status string
	PlacedAt           time.Time
	Total              int64
	AccountID          string // "" = a guest order
}

// ErrNoMatch is the one answer to every pair that does not open an order: a
// wrong phone, an unknown number, an empty phone, an order that is someone else's.
var ErrNoMatch = errors.New("no order matches that number and phone")

// LockedError says the number or the address is locked, for how long (1s to 15 min).
type LockedError struct{ Wait time.Duration }

func (e *LockedError) Error() string { return "too many tries" }

// NumberKey is the ASCII digits of a typed order number, and whether there are
// 1 to 18 of them. "DOI-24818", "doi 24818" and "24818" all give "24818"; the
// stored number is "DOI-" plus the digits.
func NumberKey(raw string) (string, bool) {
	d := digitsOnly(raw)
	return d, len(d) >= 1 && len(d) <= maxNumberLen
}

// findSQL is A7. The phone is read only to be compared here and is never copied
// into the value the function returns.
const findSQL = `SELECT id, number, placed_at, status, total, phone, coalesce(account_id::text, '')
FROM orders WHERE number = $1`

// Attempt is one try of a pair, all of it ONE transaction:
//
//  1. a transaction advisory lock on the typed number's digits, so tries for one
//     number run one after another and the sixth always sees five misses;
//  2. the two counters are checked (the number's only when the number has
//     digits: a shared "no number" bucket would let one visitor lock every other
//     empty try); a locked one ends the call with *LockedError and counts nothing;
//  3. the order is found by its number and the phones are compared in full;
//  4. a match is handed to hit. nil commits and returns the order: nothing is
//     counted and nothing cleared. ErrNoMatch is a miss (hit must have written
//     nothing before returning it). Any other error rolls everything back.
//
// A miss counts one failure on each counter and answers ErrNoMatch.
func Attempt(ctx context.Context, db DB, addr string, p Pair,
	hit func(ctx context.Context, tx pgx.Tx, m Matched) error) (Matched, error) {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	digits, hasNumber := NumberKey(p.Number)

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Matched{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	addressSubject := ratelimit.Subject(ratelimit.OrderLookupAddress, addr)
	var numberSubject []byte
	var byNumber time.Duration
	if hasNumber {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", "order_lookup:"+digits); err != nil {
			return Matched{}, err
		}
		numberSubject = ratelimit.Subject(ratelimit.OrderLookupNumber, digits)
		if byNumber, err = ratelimit.Check(ctx, tx, ratelimit.OrderLookupNumber, numberSubject); err != nil {
			return Matched{}, err
		}
	}
	byAddress, err := ratelimit.Check(ctx, tx, ratelimit.OrderLookupAddress, addressSubject)
	if err != nil {
		return Matched{}, err
	}
	if wait := max(byNumber, byAddress); wait > 0 {
		return Matched{}, &LockedError{Wait: wait}
	}

	if hasNumber {
		var m Matched
		var phone string
		err = tx.QueryRow(ctx, findSQL, "DOI-"+digits).Scan(&m.ID, &m.Number, &m.PlacedAt, &m.Status, &m.Total, &phone, &m.AccountID)
		switch {
		case err == nil:
			if key := PhoneKey(p.Phone); key != "" && key == PhoneKey(phone) {
				switch err := hit(ctx, tx, m); {
				case err == nil:
					if err := tx.Commit(ctx); err != nil {
						return Matched{}, err
					}
					return m, nil
				case !errors.Is(err, ErrNoMatch):
					return Matched{}, err
				}
			}
		case !errors.Is(err, pgx.ErrNoRows):
			return Matched{}, err
		}
	}

	// A miss: one failure on each counter, then the one answer.
	if hasNumber {
		if err := ratelimit.Fail(ctx, tx, ratelimit.OrderLookupNumber, numberSubject, lookupLimit, lookupWindow, lookupLock); err != nil {
			return Matched{}, err
		}
	}
	if err := ratelimit.Fail(ctx, tx, ratelimit.OrderLookupAddress, addressSubject, lookupLimit, lookupWindow, lookupLock); err != nil {
		return Matched{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Matched{}, err
	}
	return Matched{}, ErrNoMatch
}
