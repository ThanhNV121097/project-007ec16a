package orders

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Patch is what a dealer's PUT changes: a nil field was not sent. The values
// are already trimmed and judged by the handler's boundary; this package trusts
// them and judges only what depends on the order's own state.
type Patch struct {
	Status, Carrier, Tracking, ETA, Customer, Phone, Email, Address, City, Note *string
}

type field struct {
	name string
	val  *string
}

// fields lists the ten editable fields in a fixed order. Each name is also the
// column it writes, so the only text spliced into the UPDATE is one of these.
func (p Patch) fields() []field {
	return []field{
		{"status", p.Status}, {"carrier", p.Carrier}, {"tracking", p.Tracking}, {"eta", p.ETA},
		{"customer", p.Customer}, {"phone", p.Phone}, {"email", p.Email}, {"address", p.Address},
		{"city", p.City}, {"note", p.Note},
	}
}

// Names are the fields the body sent, never their values.
func (p Patch) Names() []string {
	var out []string
	for _, f := range p.fields() {
		if f.val != nil {
			out = append(out, f.name)
		}
	}
	return out
}

// moves is the status machine: one step forward, or cancelled from the first
// three. A delivered or cancelled order has no move.
var moves = map[string][]string{
	"placed":    {"confirmed", "cancelled"},
	"confirmed": {"packed", "cancelled"},
	"packed":    {"shipped", "cancelled"},
	"shipped":   {"delivered"},
}

// lockedAfter are the fields a delivered or cancelled order no longer takes;
// only the note stays open.
var lockedAfter = []string{"carrier", "tracking", "eta", "customer", "phone", "email", "address", "city"}

// Judge applies the state rules to a patch against the order as it stands under
// its lock, and answers every fault by field. It reads the status BEFORE the
// move, so one body may move a shipped order to delivered and set its date.
func Judge(stored Locked, p Patch) map[string]string {
	faults := map[string]string{}
	final := len(moves[stored.Status]) == 0
	if p.Status != nil {
		to := *p.Status
		// The current status sent again is a no-op, except on an order that accepts none.
		if !(to == stored.Status && !final) && !slices.Contains(moves[stored.Status], to) {
			faults["status"] = fmt.Sprintf("An order cannot go from %s to %s.", stored.Status, to)
		}
	}
	if final {
		for _, f := range p.fields() {
			if f.val != nil && slices.Contains(lockedAfter, f.name) {
				faults[f.name] = fmt.Sprintf("An order that is %s cannot be changed.", stored.Status)
			}
		}
	}
	return faults
}

// Update applies a judged patch inside the caller's transaction, which holds the
// order's row (LockOrder, whose answer is stored). The fields and a move that is
// not a cancel are one UPDATE; a cancel goes through Cancel, which puts the
// pairs back. A patch that names nothing issues no statement, so {} changes
// nothing, not even updated_at.
func Update(ctx context.Context, tx pgx.Tx, stored Locked, p Patch) error {
	cancel := p.Status != nil && *p.Status == "cancelled" && stored.Status != "cancelled"
	args := []any{stored.ID}
	var sets []string
	for _, f := range p.fields() {
		if f.val == nil || (f.name == "status" && (cancel || *f.val == stored.Status)) {
			continue
		}
		args = append(args, *f.val)
		ph := "$" + strconv.Itoa(len(args))
		if f.name == "eta" {
			ph = "NULLIF(" + ph + ",'')::date"
		}
		sets = append(sets, f.name+" = "+ph)
	}
	if len(sets) > 0 {
		args = append(args, stored.Status)
		sql := "UPDATE orders SET " + strings.Join(sets, ", ") + " WHERE id = $1 AND status = $" + strconv.Itoa(len(args))
		tag, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("orders: saving %s changed %d rows under its lock, want 1", stored.ID, tag.RowsAffected())
		}
	}
	if cancel {
		changed, err := Cancel(ctx, tx, stored.ID, CancelRule{})
		if err != nil {
			return err
		}
		if !changed {
			return fmt.Errorf("orders: cancelling %s changed no row under its lock", stored.ID)
		}
	}
	return nil
}

// The two reasons a delete is refused, each the sentence the dealer reads.
var (
	ErrNotCancelled = errors.New("Cancel the order first.")
	ErrPaid         = errors.New("A paid order cannot be deleted; refund it in VNPay's portal first.")
)

// Delete removes a cancelled, unpaid order, and with it (ON DELETE CASCADE) its
// pairs, tokens and payment records. It locks the order first; ErrNotFound,
// ErrNotCancelled and ErrPaid refuse it, in that order of checking. The shelf
// and the order-number counter are not touched.
func Delete(ctx context.Context, tx pgx.Tx, id string) error {
	o, err := LockOrder(ctx, tx, id)
	if err != nil {
		return err
	}
	switch {
	case o.Status != "cancelled":
		return ErrNotCancelled
	case o.Paid:
		return ErrPaid
	}
	_, err = tx.Exec(ctx, "DELETE FROM orders WHERE id = $1", id)
	return err
}
