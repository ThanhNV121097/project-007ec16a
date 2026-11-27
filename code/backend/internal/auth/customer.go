package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// What the customer endpoints (internal/account) stand on. internal/account holds
// no SQL: every read and write of users it needs is one of the functions below.
//
// Every statement here carries role = 'customer' AND is_active in its WHERE, so a
// staff or admin row can never be read or edited through them.

// Profile is what an account says about its owner: the saved delivery details
// and the email, never a password or a hash.
type Profile struct{ Name, Email, Phone, Address, City string }

// ProfilePatch is a change to the saved details: nil leaves a field as it is and
// "" clears it (address, city).
type ProfilePatch struct{ Name, Phone, Address, City *string }

// ErrNoAccount is the answer to an id that holds no customer row, or one that
// was switched off.
var ErrNoAccount = errors.New("auth: no active customer account")

// GetProfile reads a customer's own details.
func GetProfile(ctx context.Context, q Querier, id string) (Profile, error) {
	if !IsID(id) {
		return Profile{}, ErrNoAccount
	}
	var p Profile
	err := q.QueryRow(ctx, `SELECT name, email, phone, address, city FROM users
		WHERE id = $1 AND role = 'customer' AND is_active`, id).Scan(&p.Name, &p.Email, &p.Phone, &p.Address, &p.City)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNoAccount
	}
	return p, err
}

// UpdateProfile changes the fields of the patch that are not nil and returns the
// details as saved. The values are already trimmed and checked.
func UpdateProfile(ctx context.Context, q Querier, id string, p ProfilePatch) (Profile, error) {
	if !IsID(id) {
		return Profile{}, ErrNoAccount
	}
	var out Profile
	err := q.QueryRow(ctx, `UPDATE users SET name = COALESCE($2::text, name), phone = COALESCE($3::text, phone),
		address = COALESCE($4::text, address), city = COALESCE($5::text, city)
		WHERE id = $1 AND role = 'customer' AND is_active
		RETURNING name, email, phone, address, city`, id, p.Name, p.Phone, p.Address, p.City).
		Scan(&out.Name, &out.Email, &out.Phone, &out.Address, &out.City)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNoAccount
	}
	return out, err
}

// RegisterCustomer makes a customer account and its first session in ONE
// transaction, so a failure never leaves an account its owner cannot use. n.Role
// is forced to customer here, whatever the caller set, and n.PasswordHash comes
// from HashPassword, which the caller ran before this call. ErrEmailTaken is the
// answer to an address any account holds: the unique index decides a race. The
// returned token is the cookie value.
func (s *Service) RegisterCustomer(ctx context.Context, db Beginner, n NewUser) (string, error) {
	n.Role = Customer
	tx, err := db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	u, err := CreateUser(ctx, tx, n)
	if err != nil {
		return "", err
	}
	token, err := s.IssueSession(ctx, tx, u.ID, n.PasswordHash)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return token, nil
}
