package orders

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

// AccountOrder is one row of "My orders": an order's public header, never its
// phone, email, address or note.
type AccountOrder struct {
	ID, Number, Status string
	PlacedAtTime       timeValue
	Total              int64
}

// MarshalJSON writes id, number, placed_at (RFC 3339 with +07:00), status, total.
func (o AccountOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID       string `json:"id"`
		Number   string `json:"number"`
		PlacedAt string `json:"placed_at"`
		Status   string `json:"status"`
		Total    int64  `json:"total"`
	}{o.ID, o.Number, o.PlacedAtTime.In(ict).Format("2006-01-02T15:04:05") + "+07:00", o.Status, o.Total})
}

// ListFor is a customer's own orders, newest first (page >= 1, 1 <= limit <= 100),
// with how many they have in all. The page and the total are read in one
// REPEATABLE READ read-only transaction, so "Show more" never shows a row twice
// or skips one that a concurrent placement moved.
func ListFor(ctx context.Context, db DB, accountID string, page, limit int) ([]AccountOrder, int, error) {
	if !auth.IsID(accountID) {
		return []AccountOrder{}, 0, nil
	}
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	rows, err := tx.Query(ctx, `SELECT id, number, placed_at, status, total FROM orders
		WHERE account_id = $1::uuid ORDER BY placed_at DESC, seq DESC LIMIT $2 OFFSET $3`,
		accountID, int64(limit), int64(page-1)*int64(limit))
	if err != nil {
		return nil, 0, err
	}
	items := []AccountOrder{}
	for rows.Next() {
		var o AccountOrder
		if err := rows.Scan(&o.ID, &o.Number, &o.PlacedAtTime, &o.Status, &o.Total); err != nil {
			rows.Close()
			return nil, 0, err
		}
		items = append(items, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM orders WHERE account_id = $1::uuid`, accountID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, int(total), nil
}

// Link makes a guest order the account's (A8), once: only an order linked to no
// one, and only to an active customer. linked is false when no row changed.
func Link(ctx context.Context, q Querier, accountID, orderID string) (linked bool, err error) {
	if !auth.IsID(accountID) {
		return false, nil
	}
	var id string
	err = q.QueryRow(ctx, `UPDATE orders o SET account_id = u.id FROM users u
		WHERE o.id = $2 AND o.account_id IS NULL AND u.id = $1::uuid AND u.role = 'customer' AND u.is_active
		RETURNING o.id`, accountID, orderID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// AddToAccount is "Add an order": the pair is tried exactly as the order lookup
// tries it (Attempt), and a right pair for an order linked to no one links it to
// this account. An order already linked to this account is left as it is; one
// linked to another account is a miss, indistinguishable from every other.
func AddToAccount(ctx context.Context, db DB, addr string, p Pair, accountID string) (AccountOrder, error) {
	m, err := Attempt(ctx, db, addr, p, func(ctx context.Context, tx pgx.Tx, m Matched) error {
		switch m.AccountID {
		case "":
			linked, err := Link(ctx, tx, accountID, m.ID)
			if err != nil {
				return err
			}
			if !linked {
				return ErrNoMatch
			}
			return nil
		case accountID:
			return nil
		default:
			return ErrNoMatch
		}
	})
	if err != nil {
		return AccountOrder{}, err
	}
	return AccountOrder{ID: m.ID, Number: m.Number, Status: m.Status, PlacedAtTime: m.PlacedAt, Total: m.Total}, nil
}

// callerAccount is the id placement links an order to: the session's own, and
// only for a customer. A guest, a staff member and the admin give "".
func callerAccount(r *http.Request) string {
	if u, ok := auth.FromContext(r.Context()); ok && u.Role == auth.Customer {
		return u.ID
	}
	return ""
}

// owner is the value bound to the guarded sub-select of the insert: NULL (a
// guest order) unless the id has the shape of a user id.
func owner(accountID string) any {
	if auth.IsID(accountID) {
		return accountID
	}
	return nil
}
