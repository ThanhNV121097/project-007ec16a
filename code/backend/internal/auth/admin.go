package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// The admin password the owner supplies is 10 to 100 characters.
const (
	minAdminPassword = 10
	maxAdminPassword = 100
)

// ErrEmailTaken is CreateUser's answer to an address another account holds.
var ErrEmailTaken = errors.New("auth: the email address is taken")

// NewUser is the account CreateUser makes. Email must already have passed
// NormalizeEmail and PasswordHash come from HashPassword: CreateUser never
// hashes (hashing belongs outside any transaction). Phone, Address and City may
// be empty.
type NewUser struct {
	Email, Name, Phone, Address, City string
	Role                              Role
	PasswordHash                      string
}

// CreateUser inserts an account, active. The unique index on email is the
// arbiter of a race: its violation is ErrEmailTaken.
func CreateUser(ctx context.Context, q Querier, n NewUser) (User, error) {
	var id string
	err := q.QueryRow(ctx, `INSERT INTO users (email, name, phone, address, city, password_hash, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
		n.Email, n.Name, n.Phone, n.Address, n.City, n.PasswordHash, string(n.Role)).Scan(&id)
	if constraintOf(err) == "uq_users_email" {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Email: n.Email, Name: n.Name, Role: n.Role, Active: true}, nil
}

// EnsureAdmin makes the admin account from the owner's settings. It runs once
// after the migrations and before the server listens, in one transaction, and
// logs neither the password, nor the hash, nor the address.
//
// A problem with the settings (one unset, a password under 10 characters, an
// address that cannot be stored, an address another account holds) is logged
// and skipped: it returns nil and the server starts. An admin row made by an
// earlier run stays and keeps working, so removing a setting never locks the
// owner out; changing the password is how to do that. Only a database error is
// returned.
//
// A row that already matches the settings is not touched (no new salt, no new
// updated_at). A changed email or password rewrites the row, switches it on
// and ends every session it holds.
func EnsureAdmin(ctx context.Context, db Beginner, email, password string, log *slog.Logger) error {
	if email == "" || password == "" {
		log.Info("admin account not made: ADMIN_EMAIL or ADMIN_PASSWORD is not set")
		return nil
	}
	norm, ok := NormalizeEmail(email)
	if !ok {
		log.Warn("admin account not made: ADMIN_EMAIL is not an address that can be stored")
		return nil
	}
	switch n := utf8.RuneCountInString(password); {
	case n < minAdminPassword:
		log.Warn("admin account not made: ADMIN_PASSWORD is too short (shorter than 10 characters)")
		return nil
	case n > maxAdminPassword:
		log.Warn("admin account not made: ADMIN_PASSWORD is too long (longer than 100 characters)")
		return nil
	}
	// Hashed before the transaction opens, so no row lock is held meanwhile.
	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("ensure admin: %w", err)
	}
	for range 2 {
		retry, err := ensureAdminOnce(ctx, db, norm, password, hash, log)
		if err != nil {
			return err
		}
		if !retry {
			return nil
		}
	}
	return errors.New("ensure admin: the admin row kept changing under it")
}

// ensureAdminOnce is one transaction. retry is true when another instance made
// the admin row first: the caller reads again and takes the row-exists path.
func ensureAdminOnce(ctx context.Context, db Beginner, email, password, newHash string, log *slog.Logger) (retry bool, err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("ensure admin: %s", describe(err))
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	var id, storedEmail, storedHash string
	var active bool
	err = tx.QueryRow(ctx, `SELECT id::text, email, password_hash, is_active FROM users WHERE role = 'admin' FOR UPDATE`).
		Scan(&id, &storedEmail, &storedHash, &active)

	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `INSERT INTO users (email, name, password_hash, role, is_active) VALUES ($1, '', $2, 'admin', true)`, email, newHash)
		switch constraintOf(err) {
		case "":
		case "uq_users_one_admin":
			return true, nil
		case "uq_users_email":
			log.Warn("admin account not made: ADMIN_EMAIL belongs to another account")
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("ensure admin: %s", describe(err))
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("ensure admin: %s", describe(err))
		}
		log.Info("admin account made")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("ensure admin: %s", describe(err))
	}

	same := active && storedEmail == email
	if same {
		match, _, err := check(ctx, storedHash, password)
		if err != nil {
			return false, fmt.Errorf("ensure admin: %s", describe(err))
		}
		same = match
	}
	if same {
		log.Info("admin account unchanged")
		return false, nil
	}

	_, err = tx.Exec(ctx, `UPDATE users SET email = $2, password_hash = $3, is_active = true WHERE id = $1`, id, email, newHash)
	if constraintOf(err) == "uq_users_email" {
		log.Warn("admin account not changed: ADMIN_EMAIL belongs to another account")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("ensure admin: %s", describe(err))
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, id); err != nil {
		return false, fmt.Errorf("ensure admin: %s", describe(err))
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("ensure admin: %s", describe(err))
	}
	log.Info("admin account changed to match the settings; its sessions were ended")
	return false, nil
}

// describe is an error as a sentence that cannot hold a value: a database error
// is its SQLSTATE and constraint, never its message or detail.
func describe(err error) string {
	if c := constraintOf(err); c != "" {
		return "unique constraint " + c
	}
	return err.Error()
}
