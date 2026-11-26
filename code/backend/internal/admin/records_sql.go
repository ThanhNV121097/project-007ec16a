package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"syscall"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

// recRefusal is an answer the contract names (404, 409, 422) that a write
// decides while its transaction is open: returned from the function the
// transaction runs, it rolls everything back and is written as the answer.
type recRefusal struct {
	status        int
	code, message string
	fields        map[string]string
}

func (r *recRefusal) Error() string { return r.code }

func recInvalid(fields map[string]string) *recRefusal {
	return &recRefusal{status: http.StatusUnprocessableEntity, code: "invalid", message: "Check the form.", fields: fields}
}

func recMissing() *recRefusal {
	return &recRefusal{status: http.StatusNotFound, code: "not_found", message: "Not found."}
}

func recNotFound(w http.ResponseWriter) { refuse(w, http.StatusNotFound, "not_found", "Not found.", nil) }

// recordsLock is the one advisory lock every write of a record takes first, so
// two saves of one record, two saves that both turn "Opens the home page" on and
// a review event racing a shoe's save are done one after the other.
const recordsLock = "admin:records"

// inTx runs fn in one transaction that holds the records lock. A nil result
// commits; any error rolls back. PostgreSQL ends one side of a deadlock (a save
// of one shoe meeting an order that takes two shoes' rows) with SQLSTATE 40P01
// and the transaction is run once more; fn only judges and writes what the
// request said, so running it twice is the same as running it once.
func (h *recHandler) inTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	err := h.inTxOnce(ctx, fn)
	if isDeadlock(err) {
		err = h.inTxOnce(ctx, fn)
	}
	return err
}

func (h *recHandler) inTxOnce(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, recordsLock); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// fail answers a write that did not complete: the refusal the contract names, or
// 503 for a database that refused the connection, or the generic 500 — a
// statement that failed for any other reason is a bug and does not get better on
// a retry. The cause goes to the log, never to the caller.
func (h *recHandler) fail(w http.ResponseWriter, operation string, err error) {
	var refusal *recRefusal
	if errors.As(err, &refusal) {
		refuse(w, refusal.status, refusal.code, refusal.message, refusal.fields)
		return
	}
	auth.LogFailure(h.log, "records request failed", operation, err)
	var op *net.OpError
	if errors.As(err, &op) && errors.Is(op.Err, syscall.ECONNREFUSED) {
		refuse(w, http.StatusServiceUnavailable, "unavailable", "The shop is briefly unavailable. Please try again.", nil)
		return
	}
	refuse(w, http.StatusInternalServerError, "internal", internalMessage, nil)
}

// recExists says whether a record of a table exists. The table is one of the
// spec's, never the request's.
func recExists(ctx context.Context, tx pgx.Tx, table, id string) (bool, error) {
	var found bool
	err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM "+table+" WHERE id = $1)", id).Scan(&found)
	return found, err
}

// checkRefs names every reference of a body that points at nothing.
func checkRefs(ctx context.Context, tx pgx.Tx, values []recValue, faults map[string]string) error {
	for _, v := range values {
		id, isRef := v.arg.(string)
		if v.field.kind != recRef || !isRef || id == "" {
			continue
		}
		found, err := recExists(ctx, tx, v.field.refTable, id)
		if err != nil {
			return err
		}
		if !found {
			faults[v.field.name] = v.field.unknown
		}
	}
	return nil
}

var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slugOf is a name, already folded to plain lower case by the database, as an id:
// anything but letters and digits becomes a hyphen, trimmed, at most 80
// characters, "item" when nothing is left.
func slugOf(folded string) string {
	s := strings.Trim(notSlug.ReplaceAllString(strings.ToLower(folded), "-"), "-")
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	if s == "" {
		return "item"
	}
	return s
}

// newID makes a record's id: a shoe, line or category gets the slug of its name
// with -2, -3 … added while it is taken; a review gets r- and eight hex
// characters. The records lock is held, so nothing else takes an id meanwhile.
func newID(ctx context.Context, tx pgx.Tx, s *recSpec, name string) (string, error) {
	if s.table == "reviews" {
		for range 10 {
			var b [4]byte
			if _, err := rand.Read(b[:]); err != nil {
				return "", err
			}
			id := "r-" + hex.EncodeToString(b[:])
			if found, err := recExists(ctx, tx, s.table, id); err != nil || !found {
				return id, err
			}
		}
		return "", errors.New("records: no free review id after 10 tries")
	}
	var folded string
	if err := tx.QueryRow(ctx, `SELECT doi_fold($1)`, name).Scan(&folded); err != nil {
		return "", err
	}
	base := slugOf(folded)
	for n := 1; n <= 10000; n++ {
		id := base
		if n > 1 {
			id = fmt.Sprintf("%s-%d", base, n)
		}
		if found, err := recExists(ctx, tx, s.table, id); err != nil || !found {
			return id, err
		}
	}
	return "", errors.New("records: no free id after 10000 tries")
}

const productRatingSQL = `UPDATE products p SET rating = s.average, review_count = s.shown
FROM (SELECT coalesce(round(avg(rating)::numeric, 1), 0) AS average, count(*)::integer AS shown
      FROM reviews WHERE product_id = $1 AND status = 'live') s
WHERE p.id = $1`

const shopRatingSQL = `UPDATE shop_terms t SET rating = s.average, review_count = s.shown
FROM (SELECT coalesce(round(avg(r.rating)::numeric, 1), 0) AS average, count(*)::integer AS shown
      FROM reviews r JOIN products p ON p.id = r.product_id
      WHERE r.status = 'live' AND p.status = 'live') s
WHERE t.id = 'shop'`

// recomputeRatings makes a shoe's rating and review count follow its shown
// reviews, and the shop's follow the shown reviews of shoes on sale, in the
// transaction of the event that changed them. A shoe may be "" (none).
func recomputeRatings(ctx context.Context, tx pgx.Tx, shoes ...string) error {
	done := map[string]bool{}
	for _, id := range shoes {
		if id == "" || done[id] {
			continue
		}
		done[id] = true
		if _, err := tx.Exec(ctx, productRatingSQL, id); err != nil {
			return err
		}
	}
	return recomputeShop(ctx, tx)
}

// recomputeShop makes the shop's rating and review count follow the shown
// reviews of shoes on sale.
func recomputeShop(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, shopRatingSQL)
	return err
}
