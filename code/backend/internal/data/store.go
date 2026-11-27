package data

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// errNotFound is a record that does not exist.
var errNotFound = errors.New("record not found")

// ErrNotFound is errNotFound for the packages that read one record through
// Record.
var ErrNotFound = errNotFound

// Querier is the slice of *pgxpool.Pool the reader needs. Both statements the
// reader sends return at most one row, so one method is enough — and one method
// is all a test has to fake to see the SQL a visitor's read sends.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// store reads the collections from the database.
type store struct{ db Querier }

// page is one page of a collection: the records as a JSON array, built by the
// database, and how many records match in all.
type page struct {
	items json.RawMessage
	total int64
}

func (s store) list(ctx context.Context, c *Collection, req listRequest) (page, error) {
	sql, args := listSQL(c, req)
	var items string
	var p page
	if err := s.db.QueryRow(ctx, sql, args...).Scan(&p.total, &items); err != nil {
		return page{}, err
	}
	p.items = json.RawMessage(items)
	return p, nil
}

// Record reads one record of a registry collection in the team's view, with the
// querier it is given (a transaction, so a write can answer with what it saved).
// A collection that is not served, or an id that is not there, is ErrNotFound.
func Record(ctx context.Context, q Querier, collection, id string) (json.RawMessage, error) {
	c, ok := byName[collection]
	if !ok {
		return nil, errNotFound
	}
	return store{db: q}.record(ctx, c, id, true)
}

func (s store) record(ctx context.Context, c *Collection, id string, dealer bool) (json.RawMessage, error) {
	var body string
	err := s.db.QueryRow(ctx, recordSQL(c, dealer), id).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}
