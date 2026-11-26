package stock

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// What the shop's team writes to the counts. Like Take, every function runs in
// the caller's transaction; the package is still the only writer of
// product_sizes.

var (
	wholeSize   = regexp.MustCompile(`^[0-9]{1,3}$`)
	decimalSize = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,2}$`)
	mixedSize   = regexp.MustCompile(`^([0-9]{1,3}) ([0-9]{1,3})/([0-9]{1,3})$`)
)

// MaxSizeRunes is the longest a size may be written (ck_product_sizes_size_not_blank).
const MaxSizeRunes = 20

// ParseSize reads an EU size as people write it — 38, 36 2/3, 35 1/2, 35.5 —
// with white space collapsed to single spaces. It returns the size as it will be
// stored and listed, and its value as a number (36 2/3 is 36.67), the key a
// shoe's sizes are ordered by. ok is false for a size the shop cannot read as a
// number, or one longer than 20 characters.
func ParseSize(raw string) (size string, value float64, ok bool) {
	size = strings.Join(strings.Fields(raw), " ")
	if size == "" || utf8.RuneCountInString(size) > MaxSizeRunes {
		return size, 0, false
	}
	var v float64
	switch {
	case wholeSize.MatchString(size), decimalSize.MatchString(size):
		n, err := strconv.ParseFloat(size, 64)
		if err != nil {
			return size, 0, false
		}
		v = n
	case mixedSize.MatchString(size):
		m := mixedSize.FindStringSubmatch(size)
		whole, _ := strconv.ParseFloat(m[1], 64)
		num, _ := strconv.ParseFloat(m[2], 64)
		den, _ := strconv.ParseFloat(m[3], 64)
		if den == 0 {
			return size, 0, false
		}
		v = whole + num/den
	default:
		return size, 0, false
	}
	v = math.Round(v*100) / 100
	if v <= 0 || v >= 1000 {
		return size, 0, false
	}
	return size, v, true
}

// Pairs reads the counts of one shoe and takes them FOR UPDATE, in size order —
// the order Lock uses — so an order that is placing pairs of the shoe finishes
// first and what is read is what stands.
func Pairs(ctx context.Context, tx Tx, product string) (map[string]int, error) {
	rows, err := tx.Query(ctx, `SELECT size, pairs FROM product_sizes WHERE product_id = $1 ORDER BY size FOR UPDATE`, product)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var size string
		var pairs int
		if err := rows.Scan(&size, &pairs); err != nil {
			return nil, err
		}
		out[size] = pairs
	}
	return out, rows.Err()
}

const putSQL = `INSERT INTO product_sizes (product_id, size, size_value, pairs) VALUES ($1, $2, $3::numeric, $4)
ON CONFLICT (product_id, size) DO UPDATE SET pairs = EXCLUDED.pairs, size_value = EXCLUDED.size_value`

// Put writes the count of one size of a shoe, listing the size when the shoe has
// none of it. The size is as ParseSize returned it.
func Put(ctx context.Context, tx Tx, product, size string, value float64, pairs int) error {
	_, err := tx.Exec(ctx, putSQL, product, size, strconv.FormatFloat(value, 'f', 2, 64), pairs)
	return err
}

// Remove takes sizes off a shoe: they lose their counts.
func Remove(ctx context.Context, tx Tx, product string, sizes []string) error {
	if len(sizes) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `DELETE FROM product_sizes WHERE product_id = $1 AND size = ANY($2::text[])`, product, sizes)
	return err
}

// ErrNoSize is SetOne's answer for a shoe that does not list the size.
var ErrNoSize = errors.New("stock: the shoe does not list that size")

// ErrMoved is SetOne's answer when the count is no longer the one the caller saw:
// an order took pairs since, and nothing was overwritten.
var ErrMoved = errors.New("stock: the count moved since it was read")

const setOneSQL = `UPDATE product_sizes SET pairs = $3 WHERE product_id = $1 AND size = $2 AND pairs = $4`

// SetOne changes one size's count to pairs, only if it still holds seen: the
// number the person last saw. No row changed means the size is not listed
// (ErrNoSize) or an order moved the count (ErrMoved).
func SetOne(ctx context.Context, tx Tx, product, size string, pairs, seen int) error {
	tag, err := tx.Exec(ctx, setOneSQL, product, size, pairs, seen)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	rows, err := tx.Query(ctx, `SELECT 1 FROM product_sizes WHERE product_id = $1 AND size = $2`, product, size)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return ErrMoved
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return ErrNoSize
}

// Lists are a shoe's sizes, last pairs, sold-out sizes and total, derived from
// its counts by the view product_stock.
type Lists struct {
	Sizes []string `json:"sizes"`
	Low   []string `json:"sizes_low"`
	Out   []string `json:"sizes_out"`
	Stock int      `json:"stock"`
}

// ReadLists reads the derived lists of a shoe as the counts stand in this
// transaction.
func ReadLists(ctx context.Context, tx Tx, product string) (Lists, error) {
	rows, err := tx.Query(ctx, `SELECT sizes, sizes_low, sizes_out, stock FROM product_stock WHERE product_id = $1`, product)
	if err != nil {
		return Lists{}, err
	}
	defer rows.Close()
	l := Lists{Sizes: []string{}, Low: []string{}, Out: []string{}}
	if rows.Next() {
		if err := rows.Scan(&l.Sizes, &l.Low, &l.Out, &l.Stock); err != nil {
			return Lists{}, err
		}
	}
	if err := rows.Err(); err != nil {
		return Lists{}, err
	}
	for _, s := range []*[]string{&l.Sizes, &l.Low, &l.Out} {
		if *s == nil {
			*s = []string{}
		}
	}
	return l, nil
}
