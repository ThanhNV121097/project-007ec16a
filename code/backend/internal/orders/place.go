package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/stock"
	"github.com/jackc/pgx/v5"
)

// invalidError is a field fault found inside the transaction (422).
type invalidError struct{ fields fieldErrors }

func (e *invalidError) Error() string { return "the order is not valid" }

// stockError is one or more pairs the shelf cannot give (409 stock_conflict).
type stockError struct{ fields fieldErrors }

func (e *stockError) Error() string { return "pairs could not be set aside" }

// errKeyConflict is an Idempotency-Key already used by an order with another phone.
var errKeyConflict = errors.New("that order key was already used for a different order")

type shoe struct {
	name   string
	price  int64
	status string
}

// placed is what a successful placing, or a replay of one, hands to the handler.
type placed struct {
	view      json.RawMessage
	token     string
	replayed  bool
	payable   Payable
	payment   string
	paid      bool
	cancelled bool
}

// known is the order an Idempotency-Key already belongs to.
type known struct {
	id, phone, number, status, payment string
	total                              int64
	at                                 time.Time
	paid                               bool
}

// addToken gives an order one more access token and returns it. The database
// keeps only its hash.
func addToken(ctx context.Context, tx pgx.Tx, orderID string) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO order_tokens (order_id, token_hash) VALUES ($1, $2)", orderID, hashToken(token)); err != nil {
		return "", err
	}
	return token, nil
}

// replay answers a repeated key with the first order and a fresh token. The key
// alone is a bearer secret, so the replay also wants the same phone digits. No
// pair leaves the shelf again, and the pairs of the repeated request are not
// compared: the browser makes a new key whenever the bag or a detail changes.
func replay(ctx context.Context, tx pgx.Tx, k known, req orderRequest) (placed, error) {
	if digitsOf(k.phone) != digitsOf(req.Phone) {
		return placed{}, errKeyConflict
	}
	token, err := addToken(ctx, tx, k.id)
	if err != nil {
		return placed{}, err
	}
	view, err := readView(ctx, tx, k.id)
	if err != nil {
		return placed{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return placed{}, err
	}
	return placed{
		view: view, token: token, replayed: true,
		payable: Payable{ID: k.id, Number: k.number, Total: k.total, PlacedAt: k.at},
		payment: k.payment, paid: k.paid, cancelled: k.status == "cancelled",
	}, nil
}

// loadShoes reads the named shoes FOR SHARE, in id order: their price and status
// are held still until the transaction ends, and orders of one shoe still run
// side by side. The stock counts are where orders queue.
func loadShoes(ctx context.Context, tx pgx.Tx, ids []string) (map[string]shoe, error) {
	rows, err := tx.Query(ctx, "SELECT id, name, price, status FROM products WHERE id = ANY($1) ORDER BY id FOR SHARE", ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	shoes := map[string]shoe{}
	for rows.Next() {
		var id string
		var sh shoe
		if err := rows.Scan(&id, &sh.name, &sh.price, &sh.status); err != nil {
			return nil, err
		}
		shoes[id] = sh
	}
	return shoes, rows.Err()
}

// refusals judges every pair, in the order sent, and names each one the shelf
// cannot give: nothing is taken for the pairs that were fine.
func refusals(items []itemRequest, shoes map[string]shoe, held map[stock.Key]int) fieldErrors {
	out := fieldErrors{}
	for i, it := range items {
		sh, ok := shoes[it.Product]
		label := it.Product
		if ok {
			label = sh.name
		}
		prefix := fmt.Sprintf("%s · EU %s — ", label, it.Size)
		pairs, listed := held[stock.Key{Product: it.Product, Size: it.Size}]
		name := fmt.Sprintf("items.%d", i)
		switch {
		case !ok || sh.status != "live" || !listed:
			out[name] = prefix + "no longer in the shop."
		case pairs == 0:
			out[name] = prefix + "sold out in your size."
		case pairs < it.Qty:
			out[name] = fmt.Sprintf("%sonly %d left in your size.", prefix, pairs)
		}
	}
	return out
}

// totalOf is the sum of price × quantity over the pairs, in 64-bit integers, and
// false when it would overflow.
func totalOf(items []itemRequest, shoes map[string]shoe) (int64, bool) {
	var total int64
	for _, it := range items {
		price, qty := shoes[it.Product].price, int64(it.Qty)
		if price > 0 && qty > math.MaxInt64/price {
			return 0, false
		}
		line := price * qty
		if total > math.MaxInt64-line {
			return 0, false
		}
		total += line
	}
	return total, true
}

const insertOrderSQL = `INSERT INTO orders (id, number, placed_at, status, payment_method_id, paid, delivery_option_id,
    delivery_fee, eta, customer, phone, address, city, note, total, is_sample, idempotency_key)
VALUES ($1, $2, $3, 'placed', $4, false, $5, 0, $6::date, $7, $8, $9, $10, $11, $12, false, $13)`

const insertItemsSQL = `INSERT INTO order_items (order_id, line_no, product_id, size, qty, unit_price)
SELECT $1, t.n, t.product, t.size, t.qty, t.price
FROM unnest($2::smallint[], $3::text[], $4::text[], $5::integer[], $6::bigint[]) AS t(n, product, size, qty, price)`

// place saves one order in one transaction, or answers why it cannot. The shop's
// own prices make the total; the delivery fee is the constant 0; the status is
// placed and paid is false: nothing but a provider's own confirmation will ever
// mark an order paid. Lock order: key, shoes, counts, counter, inserts.
func (s *service) place(ctx context.Context, key string, req orderRequest) (placed, error) {
	ctx, cancel := context.WithTimeout(ctx, txTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return placed{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	// A stuck lock is an internal error the customer retries safely, never a hang.
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '3s'"); err != nil {
		return placed{}, err
	}
	// Two tabs with one key run one after the other; the second finds the order.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", key); err != nil {
		return placed{}, err
	}
	var k known
	err = tx.QueryRow(ctx, `SELECT id, phone, number, total, placed_at, status, paid, payment_method_id
		FROM orders WHERE idempotency_key = $1`, key).
		Scan(&k.id, &k.phone, &k.number, &k.total, &k.at, &k.status, &k.paid, &k.payment)
	switch {
	case err == nil:
		return replay(ctx, tx, k, req)
	case !errors.Is(err, pgx.ErrNoRows):
		return placed{}, err
	}

	ids := make([]string, 0, len(req.Items))
	keys := make([]stock.Key, 0, len(req.Items))
	lines := make([]stock.Line, 0, len(req.Items))
	for _, it := range req.Items {
		ids = append(ids, it.Product)
		count := stock.Key{Product: it.Product, Size: it.Size}
		keys = append(keys, count)
		lines = append(lines, stock.Line{Key: count, Qty: it.Qty})
	}
	shoes, err := loadShoes(ctx, tx, ids)
	if err != nil {
		return placed{}, err
	}
	held, err := stock.Lock(ctx, tx, keys)
	if err != nil {
		return placed{}, err
	}
	// The clock is read after the counts are locked, so the order of the numbers
	// and the order of the times agree.
	now := s.now().In(ict)

	if refused := refusals(req.Items, shoes, held); len(refused) > 0 {
		return placed{}, &stockError{fields: refused}
	}
	total, ok := totalOf(req.Items, shoes)
	if !ok {
		return placed{}, &invalidError{fields: fieldErrors{"items": "That order is too large."}}
	}

	var daysText, cutoff string
	if err := tx.QueryRow(ctx, "SELECT standard_days, dispatch_cutoff FROM shop_terms WHERE id = 'shop'").Scan(&daysText, &cutoff); err != nil {
		return placed{}, err
	}
	days, err := largestNumber(daysText)
	if err != nil {
		return placed{}, err
	}
	eta, err := expectedDate(now, days, cutoff)
	if err != nil {
		return placed{}, err
	}

	if err := stock.Take(ctx, tx, lines); err != nil {
		return placed{}, err
	}
	var n int64
	if err := tx.QueryRow(ctx, "UPDATE counters SET value = value + 1 WHERE name = 'order_number' RETURNING value").Scan(&n); err != nil {
		return placed{}, err
	}
	number, id := fmt.Sprintf("DOI-%d", n), fmt.Sprintf("doi-%d", n)

	if _, err := tx.Exec(ctx, insertOrderSQL, id, number, now, req.Payment, standardDelivery, eta.Format("2006-01-02"),
		req.Customer, req.Phone, req.Address, req.City, req.Note, total, key); err != nil {
		return placed{}, err
	}
	nums, products, sizes, qtys, prices := make([]int16, len(req.Items)), make([]string, len(req.Items)),
		make([]string, len(req.Items)), make([]int32, len(req.Items)), make([]int64, len(req.Items))
	for i, it := range req.Items {
		nums[i], products[i], sizes[i], qtys[i], prices[i] = int16(i+1), it.Product, it.Size, int32(it.Qty), shoes[it.Product].price
	}
	if _, err := tx.Exec(ctx, insertItemsSQL, id, nums, products, sizes, qtys, prices); err != nil {
		return placed{}, err
	}
	token, err := addToken(ctx, tx, id)
	if err != nil {
		return placed{}, err
	}
	view, err := readView(ctx, tx, id)
	if err != nil {
		return placed{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return placed{}, err
	}
	return placed{
		view: view, token: token,
		payable: Payable{ID: id, Number: number, Total: total, PlacedAt: now},
		payment: req.Payment,
	}, nil
}
