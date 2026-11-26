// Package stock is the only writer of product_sizes: the one count of pairs per
// shoe and EU size. A shoe's sizes, last-pairs list, sold-out list and total are
// derived from these counts by the view product_stock, so no list can disagree
// with a count. It has no routes: orders (and, later, the staff screens and the
// cancel story) call it inside their own transaction.
package stock

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Tx is the slice of a transaction the stock functions need.
type Tx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Key names one count: a shoe and an EU size exactly as the shoe lists it.
type Key struct{ Product, Size string }

// Line is a number of pairs to take from one count.
type Line struct {
	Key
	Qty int
}

const lockSQL = `SELECT product_id, size, pairs FROM product_sizes
WHERE (product_id, size) IN (SELECT * FROM unnest($1::text[], $2::text[]))
ORDER BY product_id, size
FOR UPDATE`

// Lock takes the named counts FOR UPDATE, in (shoe, size) order — the order every
// transaction that locks several of them must use, so two orders never deadlock —
// and returns the pairs each holds. A count that does not exist is absent from the
// answer. Under READ COMMITTED a waiter reads the row again once the lock is
// released, so it sees the pairs the earlier order left.
func Lock(ctx context.Context, tx Tx, keys []Key) (map[Key]int, error) {
	held := make(map[Key]int, len(keys))
	if len(keys) == 0 {
		return held, nil
	}
	products := make([]string, len(keys))
	sizes := make([]string, len(keys))
	for i, k := range keys {
		products[i], sizes[i] = k.Product, k.Size
	}
	rows, err := tx.Query(ctx, lockSQL, products, sizes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k Key
		var pairs int
		if err := rows.Scan(&k.Product, &k.Size, &pairs); err != nil {
			return nil, err
		}
		held[k] = pairs
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return held, nil
}

const returnSQL = `UPDATE product_sizes SET pairs = pairs + $3 WHERE product_id = $1 AND size = $2`

// Return puts pairs back, one statement per line in (shoe, size) order, in the
// caller's transaction: two cancels never lock the same two counts in opposite
// orders. Unlike Take, a statement that changes no row is not an error: a size the
// dealer took off the shoe since the order was placed has no count to return to,
// and that is correct, the size no longer exists.
func Return(ctx context.Context, tx Tx, lines []Line) error {
	sorted := append([]Line(nil), lines...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Product != sorted[j].Product {
			return sorted[i].Product < sorted[j].Product
		}
		return sorted[i].Size < sorted[j].Size
	})
	for _, l := range sorted {
		if _, err := tx.Exec(ctx, returnSQL, l.Product, l.Size, l.Qty); err != nil {
			return err
		}
	}
	return nil
}

const takeSQL = `UPDATE product_sizes SET pairs = pairs - $3 WHERE product_id = $1 AND size = $2 AND pairs >= $3`

// Take removes the pairs of every line, one statement per line in lock order. The
// caller has locked the counts with Lock and checked them, so every statement
// changes exactly one row; anything else is an error and the transaction must be
// rolled back. The database's CHECK (pairs >= 0) is the last floor under it.
func Take(ctx context.Context, tx Tx, lines []Line) error {
	sorted := append([]Line(nil), lines...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Product != sorted[j].Product {
			return sorted[i].Product < sorted[j].Product
		}
		return sorted[i].Size < sorted[j].Size
	})
	for _, l := range sorted {
		tag, err := tx.Exec(ctx, takeSQL, l.Product, l.Size, l.Qty)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("stock: taking %d pairs of %s in EU %s changed %d rows, want 1", l.Qty, l.Product, l.Size, tag.RowsAffected())
		}
	}
	return nil
}
