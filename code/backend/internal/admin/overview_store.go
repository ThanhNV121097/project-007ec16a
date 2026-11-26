package admin

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// The five statements of the overview, in the order they run, all inside one
// REPEATABLE READ, READ ONLY transaction so every figure is of the same
// instant. $1 is the handler's clock; the shop's day and month are cut at
// Vietnam midnight and compared as instants, so neither the zone an order was
// written in nor the session's TimeZone can move a figure. None of them names a
// column that holds a customer's phone, email, address, city or note.

// overviewFiguresSQL: this month's sales and orders, the orders waiting to be
// packed and the pairs and shoes on the shelf. Stock comes only from the view
// product_stock, the one definition of it.
const overviewFiguresSQL = `
WITH shop AS (
    SELECT ($1::timestamptz AT TIME ZONE 'Asia/Ho_Chi_Minh')::date AS today
), span AS (
    SELECT date_trunc('month', today::timestamp) AT TIME ZONE 'Asia/Ho_Chi_Minh'                          AS month_from,
           (date_trunc('month', today::timestamp) + interval '1 month') AT TIME ZONE 'Asia/Ho_Chi_Minh'   AS month_to
      FROM shop
)
SELECT m.sales, m.orders, w.waiting, p.pairs, p.shoes
  FROM (SELECT coalesce(sum(o.total), 0)::bigint AS sales, count(*)::bigint AS orders
          FROM orders o, span s
         WHERE o.status <> 'cancelled' AND o.placed_at >= s.month_from AND o.placed_at < s.month_to) m,
       (SELECT count(*)::bigint AS waiting FROM orders WHERE status IN ('placed', 'confirmed')) w,
       (SELECT coalesce(sum(stock), 0)::bigint AS pairs, count(*)::bigint AS shoes FROM product_stock) p`

// overviewDaysSQL: exactly 14 rows, the shop's today and the 13 days before it,
// oldest first, a day with no sale present as 0. The series is an integer one
// added to a date, never generate_series over dates: that form resolves to
// timestamptz and its ::date would depend on the session's TimeZone.
const overviewDaysSQL = `
WITH shop AS (
    SELECT ($1::timestamptz AT TIME ZONE 'Asia/Ho_Chi_Minh')::date AS today
), sold AS (
    SELECT (o.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date AS day, sum(o.total) AS sales
      FROM orders o, shop s
     WHERE o.status <> 'cancelled'
       AND o.placed_at >= ((s.today - 13)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')
       AND o.placed_at <  ((s.today + 1)::timestamp  AT TIME ZONE 'Asia/Ho_Chi_Minh')
     GROUP BY 1
)
SELECT to_char(d.day, 'YYYY-MM-DD') AS date, coalesce(sold.sales, 0)::bigint AS sales
  FROM (SELECT s.today - 13 + i AS day FROM shop s, generate_series(0, 13) AS i) d
  LEFT JOIN sold ON sold.day = d.day
 ORDER BY d.day`

// overviewLinesSQL: every line, the pairs of non-cancelled orders of its shoes
// (all time), most pairs first, ties in the lines' own order.
const overviewLinesSQL = `
WITH sold AS (
    SELECT p.line_id, sum(i.qty)::bigint AS pairs
      FROM order_items i
      JOIN orders   o ON o.id = i.order_id   AND o.status <> 'cancelled'
      JOIN products p ON p.id = i.product_id AND p.line_id IS NOT NULL
     GROUP BY p.line_id
)
SELECT l.id AS line, l.name, coalesce(sold.pairs, 0)::bigint AS pairs
  FROM lines l
  LEFT JOIN sold ON sold.line_id = l.id
 ORDER BY coalesce(sold.pairs, 0) DESC, l.seq`

// overviewLowSQL: the shoes with a size of one or two pairs and fewer than 15
// pairs in all, drafts included, fewest first, ties in the shoes' own order.
const overviewLowSQL = `
SELECT p.id, p.name, coalesce(p.category_id, '') AS category, p.image, v.stock, v.sizes_low
  FROM product_stock v
  JOIN products p ON p.id = v.product_id
 WHERE cardinality(v.sizes_low) > 0 AND v.stock < 15
 ORDER BY v.stock, p.seq`

// overviewLatestSQL: the five newest orders of any status, each with its first
// pair. The time is written as orders/read.go and orders/list.go write it.
const overviewLatestSQL = `
SELECT o.id, o.number, o.customer,
       coalesce(f.product_id, '') AS product, coalesce(p.name, '') AS product_name, coalesce(f.size, '') AS size,
       o.status,
       to_char(o.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD"T"HH24:MI:SS') || '+07:00' AS placed_at,
       o.total
  FROM orders o
  LEFT JOIN LATERAL (SELECT i.product_id, i.size FROM order_items i WHERE i.order_id = o.id AND i.line_no = 1) f ON true
  LEFT JOIN products p ON p.id = f.product_id
 ORDER BY o.placed_at DESC, o.seq DESC
 LIMIT 5`

// overviewStore answers the overview from PostgreSQL. It reads five tables it
// writes none of, and its transaction is READ ONLY, so the database itself
// refuses a write.
type overviewStore struct{ db overviewDB }

// read runs the five statements on one snapshot. Every list is made with make,
// so an empty one is a JSON [] and never null.
func (s overviewStore) read(ctx context.Context, now time.Time) (overviewAnswer, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return overviewAnswer{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	var a overviewAnswer
	err = tx.QueryRow(ctx, overviewFiguresSQL, now).
		Scan(&a.Month.Sales, &a.Month.Orders, &a.WaitingOrders, &a.PairsOnShelf, &a.Shoes)
	if err != nil {
		return overviewAnswer{}, err
	}
	if a.SalesByDay, err = overviewRows(ctx, tx, overviewDaysSQL, now, 14, func(r pgx.Rows) (d overviewDay, err error) {
		err = r.Scan(&d.Date, &d.Sales)
		return
	}); err != nil {
		return overviewAnswer{}, err
	}
	if a.PairsByLine, err = overviewRows(ctx, tx, overviewLinesSQL, nil, 8, func(r pgx.Rows) (l overviewLine, err error) {
		err = r.Scan(&l.Line, &l.Name, &l.Pairs)
		return
	}); err != nil {
		return overviewAnswer{}, err
	}
	if a.RunningLow, err = overviewRows(ctx, tx, overviewLowSQL, nil, 8, func(r pgx.Rows) (l overviewLowShoe, err error) {
		err = r.Scan(&l.ID, &l.Name, &l.Category, &l.Image, &l.Stock, &l.SizesLow)
		if l.SizesLow == nil {
			l.SizesLow = []string{}
		}
		return
	}); err != nil {
		return overviewAnswer{}, err
	}
	if a.LatestOrders, err = overviewRows(ctx, tx, overviewLatestSQL, nil, 5, func(r pgx.Rows) (o overviewOrder, err error) {
		err = r.Scan(&o.ID, &o.Number, &o.Customer, &o.Product, &o.ProductName, &o.Size, &o.Status, &o.PlacedAt, &o.Total)
		return
	}); err != nil {
		return overviewAnswer{}, err
	}
	return a, nil
}

// overviewRows runs one statement and collects its rows into a non-nil slice.
// arg is the statement's $1, or nil for a statement that takes none.
func overviewRows[T any](ctx context.Context, tx pgx.Tx, sql string, arg any, size int, scan func(pgx.Rows) (T, error)) ([]T, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if arg == nil {
		rows, err = tx.Query(ctx, sql)
	} else {
		rows, err = tx.Query(ctx, sql, arg)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]T, 0, size)
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
