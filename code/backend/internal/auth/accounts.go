package auth

import (
	"context"
	"errors"
	"log/slog"
	"regexp"

	"github.com/jackc/pgx/v5"
)

// The statements the staff endpoints (internal/admin) stand on. internal/admin
// holds no SQL of its own: every read and write of users and sessions it needs
// is one of the functions below.
//
// Every write here carries role = 'staff' in its WHERE, so no bug in a caller's
// own check can switch off, rename, re-password or delete the admin row, and no
// statement can make or change a role.

// ErrNotFound is the answer to an id that holds no account the staff endpoints
// may touch: no such row, a customer's row, or an id that is not a uuid.
var ErrNotFound = errors.New("auth: account not found")

var canonicalID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsID reports whether s is a canonical lower-case uuid, the only spelling the
// API ever issues. PostgreSQL would also take an upper-case or braced one, and
// would fail with 22P02 on anything else; no query is sent for a bad id.
func IsID(s string) bool { return canonicalID.MatchString(s) }

const teamColumns = `id::text, email, name, role, is_active`

type scanner interface{ Scan(dest ...any) error }

func scanUser(row scanner) (User, error) {
	var u User
	var role string
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &role, &u.Active); err != nil {
		return User{}, err
	}
	u.Role = Role(role)
	return u, nil
}

// Team is the shop's own accounts, the admin first and then by creation (ties
// by id, so two reads give one order), with how many there are in all. Pass a
// REPEATABLE READ read-only transaction so the count and the page agree.
// Customers are never in it.
func Team(ctx context.Context, q Querier, limit, offset int64) ([]User, int64, error) {
	var total int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM users WHERE role IN ('admin', 'staff')`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT `+teamColumns+` FROM users WHERE role IN ('admin', 'staff')
		ORDER BY (role <> 'admin'), created_at, id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// Target is the account an id names, for the caller to judge. The admin row IS
// returned (the handler answers 409 for it); ErrNotFound is for no row, a
// customer's row and an id that is not a uuid. lock adds FOR UPDATE, for use
// inside a write transaction.
func Target(ctx context.Context, q Querier, id string, lock bool) (User, error) {
	if !IsID(id) {
		return User{}, ErrNotFound
	}
	sql := `SELECT ` + teamColumns + ` FROM users WHERE id = $1 AND role <> 'customer'`
	if lock {
		sql += ` FOR UPDATE`
	}
	u, err := scanUser(q.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// EmailTaken reports whether any account of any role holds the address, which
// must already have passed NormalizeEmail.
func EmailTaken(ctx context.Context, q Querier, email string) (bool, error) {
	var taken bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&taken)
	return taken, err
}

// changeStaff runs one UPDATE of a staff row; no row changed is ErrNotFound.
func changeStaff(ctx context.Context, tx pgx.Tx, sql, id string, value any) error {
	if !IsID(id) {
		return ErrNotFound
	}
	tag, err := tx.Exec(ctx, sql, id, value)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// endSessions deletes every session an account holds. It runs after the UPDATE
// of the same transaction, so the row lock is taken before the sessions are
// read (a session sign-in committed while the update waited is seen and ended).
func endSessions(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, id)
	return err
}

// SetName renames a staff account. name is already trimmed and checked.
func SetName(ctx context.Context, tx pgx.Tx, id, name string) error {
	return changeStaff(ctx, tx, `UPDATE users SET name = $2 WHERE id = $1 AND role = 'staff'`, id, name)
}

// SetActive switches a staff account on or off. Switching OFF also ends its
// sessions, here, so no caller can forget to; switching on brings none back.
func SetActive(ctx context.Context, tx pgx.Tx, id string, active bool) error {
	if err := changeStaff(ctx, tx, `UPDATE users SET is_active = $2 WHERE id = $1 AND role = 'staff'`, id, active); err != nil {
		return err
	}
	if active {
		return nil
	}
	return endSessions(ctx, tx, id)
}

// SetPassword replaces a staff account's password hash (from HashPassword) and
// ends its sessions, even when the new password equals the old one.
func SetPassword(ctx context.Context, tx pgx.Tx, id, hash string) error {
	if err := changeStaff(ctx, tx, `UPDATE users SET password_hash = $2 WHERE id = $1 AND role = 'staff'`, id, hash); err != nil {
		return err
	}
	return endSessions(ctx, tx, id)
}

// DeleteStaff removes a staff account. Its sessions go by ON DELETE CASCADE,
// uploads.uploaded_by and orders.account_id by ON DELETE SET NULL; nothing else
// references users.
func DeleteStaff(ctx context.Context, q Querier, id string) error {
	if !IsID(id) {
		return ErrNotFound
	}
	tag, err := q.Exec(ctx, `DELETE FROM users WHERE id = $1 AND role = 'staff'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// LogFailure logs a failed statement under the operation that ran it. A
// database error is logged by SQLSTATE and constraint name only: its detail
// holds the row, the email for a unique violation and, for a CHECK, the whole
// row including the password hash.
func LogFailure(log *slog.Logger, msg, operation string, err error) {
	logFailure(log, msg, operation, err)
}
