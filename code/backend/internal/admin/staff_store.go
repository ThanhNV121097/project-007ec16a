package admin

import (
	"context"
	"errors"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Pool is the slice of *pgxpool.Pool the staff endpoints need.
type Pool interface {
	auth.Querier
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// errAdminAccount is what update answers for a target whose role is admin,
// found under the row lock.
var errAdminAccount = errors.New("admin: the target is the admin account")

// change is what a PUT applies: nil means "left out, unchanged".
type change struct {
	name   *string
	active *bool
	hash   *string
}

// store is what the handlers ask of the accounts. pgStore answers from
// PostgreSQL through internal/auth; a test answers from memory.
type store interface {
	list(ctx context.Context, limit, offset int64) ([]auth.User, int64, error)
	lookup(ctx context.Context, id string) (auth.User, error)
	emailTaken(ctx context.Context, email string) (bool, error)
	create(ctx context.Context, name, email, hash string) (auth.User, error)
	update(ctx context.Context, id string, c change) (auth.User, error)
	remove(ctx context.Context, id string) error
}

type pgStore struct{ db Pool }

// list reads the count and the page in one REPEATABLE READ, READ ONLY
// transaction so total and items agree.
func (s pgStore) list(ctx context.Context, limit, offset int64) ([]auth.User, int64, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	return auth.Team(ctx, tx, limit, offset)
}

func (s pgStore) lookup(ctx context.Context, id string) (auth.User, error) {
	return auth.Target(ctx, s.db, id, false)
}

func (s pgStore) emailTaken(ctx context.Context, email string) (bool, error) {
	return auth.EmailTaken(ctx, s.db, email)
}

// create inserts a staff account, active. The unique index on email decides a
// race: its violation is auth.ErrEmailTaken.
func (s pgStore) create(ctx context.Context, name, email, hash string) (auth.User, error) {
	return auth.CreateUser(ctx, s.db, auth.NewUser{Name: name, Email: email, PasswordHash: hash, Role: auth.Staff})
}

// update applies the fields sent in one READ COMMITTED transaction that first
// locks the target. A deadlock (the purge of expired sessions can meet a
// switch-off) is retried once.
func (s pgStore) update(ctx context.Context, id string, c change) (auth.User, error) {
	u, err := s.updateOnce(ctx, id, c)
	if isDeadlock(err) {
		u, err = s.updateOnce(ctx, id, c)
	}
	return u, err
}

func (s pgStore) updateOnce(ctx context.Context, id string, c change) (auth.User, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return auth.User{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	u, err := auth.Target(ctx, tx, id, true)
	if err != nil {
		return auth.User{}, err
	}
	if u.Role == auth.Admin {
		return auth.User{}, errAdminAccount
	}
	if c.name != nil {
		if err := auth.SetName(ctx, tx, id, *c.name); err != nil {
			return auth.User{}, err
		}
		u.Name = *c.name
	}
	if c.active != nil {
		if err := auth.SetActive(ctx, tx, id, *c.active); err != nil {
			return auth.User{}, err
		}
		u.Active = *c.active
	}
	if c.hash != nil {
		if err := auth.SetPassword(ctx, tx, id, *c.hash); err != nil {
			return auth.User{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.User{}, err
	}
	return u, nil
}

func (s pgStore) remove(ctx context.Context, id string) error {
	return auth.DeleteStaff(ctx, s.db, id)
}

func isDeadlock(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "40P01"
}
