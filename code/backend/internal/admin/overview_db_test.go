package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/dbtest"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These cases run the five real statements against PostgreSQL, over the
// migrated sample set (14 orders, 12 shoes, 5 lines). They read
// TEST_DATABASE_URL, as CI's backend job sets it, and skip without it.
// Expected values that are not the SRS's own literals come from the oracle SQL
// of the ERD (§7), written apart from the handler's statements.

type overviewEnv struct {
	t    *testing.T
	pool *pgxpool.Pool
	n    int
}

func newOverviewEnv(t *testing.T) *overviewEnv {
	t.Helper()
	return &overviewEnv{t: t, pool: dbtest.Migrated(t)}
}

func (e *overviewEnv) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
}

func (e *overviewEnv) scalar(sql string, args ...any) int64 {
	e.t.Helper()
	var n int64
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func (e *overviewEnv) text(sql string, args ...any) string {
	e.t.Helper()
	var s string
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return s
}

func (e *overviewEnv) ids(sql string, args ...any) []string {
	e.t.Helper()
	rows, err := e.pool.Query(context.Background(), sql, args...)
	if err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			e.t.Fatal(err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		e.t.Fatal(err)
	}
	return out
}

// read is the answer of the real store as of now.
func (e *overviewEnv) read(now time.Time) overviewAnswer {
	e.t.Helper()
	a, err := overviewStore{db: e.pool}.read(context.Background(), now)
	if err != nil {
		e.t.Fatalf("read: %v", err)
	}
	return a
}

// body is the raw JSON the handler writes for a staff member as of now.
func (e *overviewEnv) body(now time.Time) []byte {
	e.t.Helper()
	mux := http.NewServeMux()
	RegisterOverview(mux, e.pool, quiet(), func() time.Time { return now })
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, overviewRequest(auth.Staff, http.MethodGet, "/v1/admin/overview"))
	if rec.Code != 200 {
		e.t.Fatalf("overview: %d %s", rec.Code, rec.Body)
	}
	return rec.Body.Bytes()
}

type overviewItem struct {
	product, size string
	qty           int
}

// order inserts an order with the private columns filled with sentinels, and
// its pairs. placed is a literal with its own zone.
func (e *overviewEnv) order(placed, status string, total int64, items ...overviewItem) string {
	e.t.Helper()
	e.n++
	id := fmt.Sprintf("tc-ov-%d", e.n)
	e.exec(`INSERT INTO orders (id, number, placed_at, status, payment_method_id, delivery_option_id, customer, phone, email, address, city, note, total)
		VALUES ($1, $2, $3::timestamptz, $4, 'cod', 'standard', 'Tester', '0999000TCOV', 'tcov@example.invalid', 'TCOV ADDRESS', 'City', 'TCOV NOTE', $5)`,
		id, fmt.Sprintf("DOI-%d", 90000+e.n), placed, status, total)
	for i, it := range items {
		e.exec(`INSERT INTO order_items (order_id, line_no, product_id, size, qty, unit_price) VALUES ($1, $2, $3, $4, $5, 0)`,
			id, i+1, it.product, it.size, it.qty)
	}
	return id
}

// clear empties the orders and the shoes (the sizes go with them), and the lines when asked.
func (e *overviewEnv) clear(lines bool) {
	e.t.Helper()
	e.exec(`DELETE FROM order_items`)
	e.exec(`DELETE FROM orders`)
	e.exec(`DELETE FROM products`)
	if lines {
		e.exec(`DELETE FROM lines`)
	}
}

type overviewSize struct {
	size  string
	pairs int
}

func (e *overviewEnv) shoe(id, status string, sizes ...overviewSize) {
	e.t.Helper()
	e.exec(`INSERT INTO products (id, name, price, status) VALUES ($1, $2, 1000000, $3)`, id, "Shoe "+id, status)
	for _, s := range sizes {
		value, err := strconv.ParseFloat(s.size, 64)
		if err != nil {
			e.t.Fatal(err)
		}
		e.exec(`INSERT INTO product_sizes (product_id, size, size_value, pairs) VALUES ($1, $2, $3, $4)`, id, s.size, value, s.pairs)
	}
}

func dayTotals(a overviewAnswer) map[string]int64 {
	out := map[string]int64{}
	for _, d := range a.SalesByDay {
		out[d.Date] = d.Sales
	}
	return out
}

func linePairs(a overviewAnswer) map[string]int64 {
	out := map[string]int64{}
	for _, l := range a.PairsByLine {
		out[l.Line] = l.Pairs
	}
	return out
}

func numbersOf(a overviewAnswer) []string {
	out := []string{}
	for _, o := range a.LatestOrders {
		out = append(out, o.Number)
	}
	return out
}

func shoeIDs(a overviewAnswer) []string {
	out := []string{}
	for _, s := range a.RunningLow {
		out = append(out, s.ID)
	}
	return out
}

// lowOracle is the running-low list read from the pairs themselves: a size of
// one or two pairs, fewer than 15 in all, fewest first, ties in the shoes' order.
func (e *overviewEnv) lowOracle() []string {
	return e.ids(`SELECT p.id FROM products p
		WHERE EXISTS (SELECT 1 FROM product_sizes s WHERE s.product_id = p.id AND s.pairs BETWEEN 1 AND 2)
		  AND (SELECT coalesce(sum(s.pairs), 0) FROM product_sizes s WHERE s.product_id = p.id) < 15
		ORDER BY (SELECT coalesce(sum(s.pairs), 0) FROM product_sizes s WHERE s.product_id = p.id), p.seq`)
}

// SC-4 to SC-8: the SRS's literals over the sample set at the SRS clock.
func TestOverviewSampleSetAtTheSRSClock(t *testing.T) {
	e := newOverviewEnv(t)
	a := e.read(overviewClock)

	if a.Month != (overviewMonth{Sales: 47660000, Orders: 13}) {
		t.Errorf("month %+v", a.Month)
	}
	if a.WaitingOrders != 2 {
		t.Errorf("waiting %d", a.WaitingOrders)
	}
	days := []int64{2790000, 3590000, 3090000, 3790000, 0, 2290000, 5090000, 3290000, 0, 5580000, 3590000, 5380000, 5890000, 3290000}
	if len(a.SalesByDay) != 14 || a.SalesByDay[0].Date != "2026-11-12" || a.SalesByDay[13].Date != "2026-11-25" {
		t.Fatalf("days %+v", a.SalesByDay)
	}
	var sum int64
	for i, d := range a.SalesByDay {
		sum += d.Sales
		if d.Sales != days[i] {
			t.Errorf("%s: %d, want %d", d.Date, d.Sales, days[i])
		}
	}
	if sum != 47660000 {
		t.Errorf("the 14 days add up to %d", sum)
	}

	var lines []string
	for _, l := range a.PairsByLine {
		lines = append(lines, fmt.Sprintf("%s %d", l.Name, l.Pairs))
	}
	if want := []string{"Samba 8", "Gazelle 2", "Megaride 2", "Handball Spezial 1", "Adizero 1"}; !reflect.DeepEqual(lines, want) {
		t.Errorf("lines %v, want %v", lines, want)
	}

	if want := []string{"DOI-24816", "DOI-24815", "DOI-24814", "DOI-24817", "DOI-24813"}; !reflect.DeepEqual(numbersOf(a), want) {
		t.Errorf("latest %v, want %v", numbersOf(a), want)
	}
	first := a.LatestOrders[0]
	name := e.text(`SELECT name FROM products WHERE id = 'handball-spezial-w-earth-strata'`)
	want := overviewOrder{ID: "doi-24816", Number: "DOI-24816", Customer: "Trang", Product: "handball-spezial-w-earth-strata", ProductName: name,
		Size: "37 1/3", Status: "placed", PlacedAt: "2026-11-25T08:40:00+07:00", Total: 3290000}
	if first != want {
		t.Errorf("first order %+v, want %+v", first, want)
	}

	// The shelf and the shoes running low are the seed's placeholder counts: read from the pairs.
	if a.PairsOnShelf != e.scalar(`SELECT coalesce(sum(pairs), 0) FROM product_sizes`) || a.Shoes != e.scalar(`SELECT count(*) FROM products`) {
		t.Errorf("shelf %d, shoes %d", a.PairsOnShelf, a.Shoes)
	}
	if got, want := shoeIDs(a), e.lowOracle(); !reflect.DeepEqual(got, want) {
		t.Errorf("running low %v, want %v", got, want)
	}
	for _, s := range a.RunningLow {
		if s.Stock >= 15 || len(s.SizesLow) == 0 {
			t.Errorf("not running low: %+v", s)
		}
	}
}

// SC-5, SC-14: a status moves the figures; a new order is counted on the next read.
func TestOverviewCancelledAndWaiting(t *testing.T) {
	e := newOverviewEnv(t)
	base := e.read(overviewClock)

	for _, status := range []string{"placed", "confirmed", "packed", "shipped", "delivered", "cancelled"} {
		e.exec(`UPDATE orders SET status = $1 WHERE id = 'doi-24816'`, status)
		a := e.read(overviewClock)
		waiting, gone := int64(1), status == "cancelled"
		if status == "placed" || status == "confirmed" {
			waiting = 2
		}
		if a.WaitingOrders != waiting {
			t.Errorf("DOI-24816 %s: waiting %d, want %d", status, a.WaitingOrders, waiting)
		}
		if gone != (a.Month.Orders == base.Month.Orders-1) || (gone && a.Month.Sales != base.Month.Sales-3290000) {
			t.Errorf("DOI-24816 %s: month %+v", status, a.Month)
		}
		if got := a.LatestOrders[0]; got.Number != "DOI-24816" || got.Status != status {
			t.Errorf("DOI-24816 %s: the newest order is %+v", status, got)
		}
	}
	e.exec(`UPDATE orders SET status = 'placed' WHERE id = 'doi-24816'`)

	// Cancelling a shipped order removes it from the month, its day and its line, not from the five latest.
	e.exec(`UPDATE orders SET status = 'cancelled' WHERE id = 'doi-24813'`)
	a := e.read(overviewClock)
	if a.Month.Orders != base.Month.Orders-1 || a.Month.Sales != base.Month.Sales-3590000 {
		t.Errorf("after the cancel: %+v", a.Month)
	}
	if dayTotals(a)["2026-11-22"] != 0 || linePairs(a)["gazelle"] != linePairs(base)["gazelle"]-1 {
		t.Errorf("after the cancel: day %d, gazelle %d", dayTotals(a)["2026-11-22"], linePairs(a)["gazelle"])
	}
	if last := a.LatestOrders[4]; last.Number != "DOI-24813" || last.Status != "cancelled" {
		t.Errorf("the cancelled order among the five latest: %+v", last)
	}
	e.exec(`UPDATE orders SET status = 'shipped' WHERE id = 'doi-24813'`)

	// A new order is in the month, the waiting count, its day and its line on the next read.
	e.order("2026-11-25T11:59:00+07:00", "placed", 1000000, overviewItem{"samba-og-w-core-black", "38", 2})
	b := e.read(overviewClock)
	if b.Month.Orders != base.Month.Orders+1 || b.Month.Sales != base.Month.Sales+1000000 || b.WaitingOrders != base.WaitingOrders+1 {
		t.Errorf("after placing: %+v waiting %d", b.Month, b.WaitingOrders)
	}
	if dayTotals(b)["2026-11-25"] != dayTotals(base)["2026-11-25"]+1000000 || linePairs(b)["samba"] != linePairs(base)["samba"]+2 {
		t.Errorf("after placing: day %d, samba %d", dayTotals(b)["2026-11-25"], linePairs(b)["samba"])
	}
	if b.LatestOrders[0].Total != 1000000 || b.LatestOrders[0].Size != "38" {
		t.Errorf("the new order is not the newest: %+v", b.LatestOrders[0])
	}
}

// SC-9, SC-10: the shelf and the shoes running low, in a shop of the shoes a case names.
func TestOverviewShelfAndRunningLow(t *testing.T) {
	e := newOverviewEnv(t)
	e.clear(false)
	e.shoe("t-a", "live", overviewSize{"40", 6}, overviewSize{"41", 6}, overviewSize{"42", 6})
	e.shoe("t-b", "live", overviewSize{"40", 8}, overviewSize{"41", 8}, overviewSize{"42", 8})
	e.shoe("t-c", "live", overviewSize{"40", 10}, overviewSize{"41", 10}, overviewSize{"42", 11})
	a := e.read(overviewClock)
	if a.PairsOnShelf != 73 || a.Shoes != 3 || len(a.RunningLow) != 0 {
		t.Errorf("three shoes: shelf %d, shoes %d, low %v", a.PairsOnShelf, a.Shoes, a.RunningLow)
	}
	e.shoe("t-d", "draft", overviewSize{"40", 10})
	if a := e.read(overviewClock); a.PairsOnShelf != 83 || a.Shoes != 4 {
		t.Errorf("with a draft: shelf %d, shoes %d", a.PairsOnShelf, a.Shoes)
	}

	e.clear(false)
	e.shoe("low-r14", "live", overviewSize{"40", 1}, overviewSize{"41", 6}, overviewSize{"42", 7})
	e.shoe("low-a", "live", overviewSize{"40", 1}, overviewSize{"41", 4})
	e.shoe("low-r15", "live", overviewSize{"40", 1}, overviewSize{"41", 7}, overviewSize{"42", 7})
	e.shoe("low-r18", "live", overviewSize{"40", 6}, overviewSize{"41", 6}, overviewSize{"42", 6})
	e.shoe("low-r9", "live", overviewSize{"40", 3}, overviewSize{"41", 3}, overviewSize{"42", 3})
	e.shoe("low-r3", "live", overviewSize{"40", 3}, overviewSize{"41", 3})
	e.shoe("low-b", "live", overviewSize{"40", 2}, overviewSize{"41", 3})
	e.shoe("low-draft", "draft", overviewSize{"40", 1}, overviewSize{"41", 3})
	a = e.read(overviewClock)
	if want := []string{"low-draft", "low-a", "low-b", "low-r14"}; !reflect.DeepEqual(shoeIDs(a), want) {
		t.Fatalf("running low %v, want %v", shoeIDs(a), want)
	}
	if want := e.lowOracle(); !reflect.DeepEqual(shoeIDs(a), want) {
		t.Errorf("running low %v, the oracle says %v", shoeIDs(a), want)
	}
	stocks := []int64{4, 5, 5, 14}
	for i, s := range a.RunningLow {
		if s.Stock != stocks[i] || !reflect.DeepEqual(s.SizesLow, []string{"40"}) || s.Category != "" || s.Name != "Shoe "+s.ID {
			t.Errorf("row %d: %+v", i, s)
		}
	}
	if a.Shoes != 8 || a.PairsOnShelf != 14+5+15+18+9+6+5+4 {
		t.Errorf("shelf %d, shoes %d", a.PairsOnShelf, a.Shoes)
	}
}

// zoneDB opens each read inside a session whose TimeZone is zone, to show no
// figure depends on it.
type zoneDB struct {
	pool *pgxpool.Pool
	zone string
}

func (z zoneDB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	tx, err := z.pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('TimeZone', $1, true)`, z.zone); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

// SC-11: the first instant of a month, written in three zones, whatever the session's zone.
func TestOverviewMonthAndDayBoundaries(t *testing.T) {
	e := newOverviewEnv(t)
	e.clear(false)
	for _, o := range []struct {
		placed string
		total  int64
	}{
		{"2026-11-30T23:59:59+07:00", 1000}, {"2026-11-30T16:59:59+00:00", 2000}, {"2026-11-30T11:59:59-05:00", 4000},
		{"2026-12-01T00:00:00+07:00", 10000}, {"2026-11-30T17:00:00+00:00", 20000}, {"2026-11-30T12:00:00-05:00", 40000},
	} {
		e.order(o.placed, "delivered", o.total)
	}
	ict := time.FixedZone("ICT", 7*3600)
	for _, zone := range []string{"UTC", "America/New_York", "Asia/Ho_Chi_Minh"} {
		store := overviewStore{db: zoneDB{pool: e.pool, zone: zone}}
		read := func(now time.Time) overviewAnswer {
			a, err := store.read(context.Background(), now)
			if err != nil {
				t.Fatalf("%s: %v", zone, err)
			}
			return a
		}
		dec := read(time.Date(2026, 12, 1, 12, 0, 0, 0, ict))
		if dec.Month != (overviewMonth{Sales: 70000, Orders: 3}) {
			t.Errorf("%s: December %+v", zone, dec.Month)
		}
		if days := dayTotals(dec); days["2026-12-01"] != 70000 || days["2026-11-30"] != 7000 || dec.SalesByDay[13].Date != "2026-12-01" {
			t.Errorf("%s: days %v", zone, days)
		}
		for _, edge := range []time.Time{time.Date(2026, 12, 1, 0, 0, 0, 0, ict), time.Date(2026, 12, 1, 23, 59, 59, 0, ict)} {
			if a := read(edge); a.SalesByDay[13].Date != "2026-12-01" || a.Month.Orders != 3 {
				t.Errorf("%s: the clock at %v reads %s, %+v", zone, edge, a.SalesByDay[13].Date, a.Month)
			}
		}
		nov := read(time.Date(2026, 11, 30, 23, 59, 59, 0, ict))
		if nov.SalesByDay[13].Date != "2026-11-30" || nov.Month != (overviewMonth{Sales: 7000, Orders: 3}) || dayTotals(nov)["2026-11-30"] != 7000 {
			t.Errorf("%s: November %+v %v", zone, nov.Month, dayTotals(nov))
		}
	}
}

// SC-7, SC-12: pairs go to their own shoe's line; every line is present; a shoe with no line is in no row.
func TestOverviewLinesAndPairs(t *testing.T) {
	e := newOverviewEnv(t)
	base := e.read(overviewClock)
	e.exec(`INSERT INTO lines (id, name) VALUES ('tc-ov-line', 'Tc Line')`)
	e.exec(`INSERT INTO products (id, name, price, status, line_id) VALUES ('tc-ov-noline', 'No line', 1000000, 'live', NULL)`)

	e.order("2026-11-25T10:00:00+07:00", "placed", 1000000, overviewItem{"samba-og-w-core-black", "38", 1}, overviewItem{"gazelle-indoor-solar-gold", "42", 1})
	e.order("2026-11-25T10:01:00+07:00", "placed", 1000000, overviewItem{"tc-ov-noline", "40", 2})
	e.order("2026-11-25T10:02:00+07:00", "cancelled", 2000000, overviewItem{"samba-og-w-core-black", "38", 5})
	a := e.read(overviewClock)
	pairs, was := linePairs(a), linePairs(base)
	if pairs["samba"] != was["samba"]+1 || pairs["gazelle"] != was["gazelle"]+1 || pairs["adizero"] != was["adizero"] || pairs["tc-ov-line"] != 0 {
		t.Errorf("pairs by line %v, before %v", pairs, was)
	}
	if len(a.PairsByLine) != len(base.PairsByLine)+1 || a.PairsByLine[len(a.PairsByLine)-1].Line != "tc-ov-line" {
		t.Errorf("a line nothing sold is the last row with 0: %+v", a.PairsByLine)
	}
	for i := 1; i < len(a.PairsByLine); i++ {
		if a.PairsByLine[i].Pairs > a.PairsByLine[i-1].Pairs {
			t.Errorf("not most pairs first: %+v", a.PairsByLine)
		}
	}

	// No orders at all: every line, in the lines' own order, with 0.
	e.exec(`DELETE FROM order_items`)
	e.exec(`DELETE FROM orders`)
	a = e.read(overviewClock)
	var got []string
	for _, l := range a.PairsByLine {
		if l.Pairs != 0 {
			t.Errorf("%s has %d pairs with no orders", l.Line, l.Pairs)
		}
		got = append(got, l.Line)
	}
	if want := e.ids(`SELECT id FROM lines ORDER BY seq`); !reflect.DeepEqual(got, want) {
		t.Errorf("lines %v, want %v", got, want)
	}
}

// SC-12: nothing at all is zeros and empty arrays, never null.
func TestOverviewEmptyShop(t *testing.T) {
	e := newOverviewEnv(t)
	e.clear(true)
	raw := string(e.body(overviewClock))
	for _, list := range []string{"running_low", "pairs_by_line", "latest_orders"} {
		if !strings.Contains(raw, `"`+list+`":[]`) {
			t.Errorf("%s is not []: %s", list, raw)
		}
	}
	if strings.Contains(raw, "null") {
		t.Errorf("a null in %s", raw)
	}
	a := e.read(overviewClock)
	if a.Month != (overviewMonth{}) || a.WaitingOrders != 0 || a.PairsOnShelf != 0 || a.Shoes != 0 || len(a.SalesByDay) != 14 {
		t.Errorf("empty shop: %+v", a)
	}
	for _, d := range a.SalesByDay {
		if d.Sales != 0 {
			t.Errorf("%s: %d", d.Date, d.Sales)
		}
	}
}

// matchesOracle compares every figure of an answer with sums taken from the
// stored rows by statements written apart from the handler's (ERD §7).
func (e *overviewEnv) matchesOracle(a overviewAnswer, now time.Time) {
	t := e.t
	t.Helper()
	cancelled := `status <> 'cancelled'`
	month := overviewMonth{
		Sales:  e.scalar(`SELECT coalesce(sum(total), 0)::bigint FROM orders WHERE `+cancelled+` AND to_char(placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM') = $1`, now.Format("2006-01")),
		Orders: e.scalar(`SELECT count(*) FROM orders WHERE `+cancelled+` AND to_char(placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM') = $1`, now.Format("2006-01")),
	}
	if a.Month != month {
		t.Errorf("%s: month %+v, the stored orders say %+v", now.Format("2006-01-02"), a.Month, month)
	}
	if want := e.scalar(`SELECT count(*) FROM orders WHERE status = 'placed' OR status = 'confirmed'`); a.WaitingOrders != want {
		t.Errorf("waiting %d, want %d", a.WaitingOrders, want)
	}
	if len(a.SalesByDay) != 14 {
		t.Fatalf("%d days", len(a.SalesByDay))
	}
	for i, d := range a.SalesByDay {
		day := now.AddDate(0, 0, i-13).Format("2006-01-02")
		want := e.scalar(`SELECT coalesce(sum(total), 0)::bigint FROM orders WHERE `+cancelled+` AND to_char(placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD') = $1`, day)
		if d.Date != day || d.Sales != want {
			t.Errorf("day %d: %+v, want %s %d", i, d, day, want)
		}
	}
	if want := e.scalar(`SELECT count(*) FROM lines`); int64(len(a.PairsByLine)) != want {
		t.Errorf("%d lines, want %d", len(a.PairsByLine), want)
	}
	for i, l := range a.PairsByLine {
		want := e.scalar(`SELECT coalesce(sum(i.qty), 0)::bigint FROM order_items i JOIN orders o ON o.id = i.order_id JOIN products p ON p.id = i.product_id
			WHERE o.status <> 'cancelled' AND p.line_id = $1`, l.Line)
		if l.Pairs != want {
			t.Errorf("line %s: %d, want %d", l.Line, l.Pairs, want)
		}
		if i > 0 && l.Pairs > a.PairsByLine[i-1].Pairs {
			t.Errorf("lines are not most pairs first: %+v", a.PairsByLine)
		}
	}
	if a.PairsOnShelf != e.scalar(`SELECT coalesce(sum(pairs), 0)::bigint FROM product_sizes`) || a.Shoes != e.scalar(`SELECT count(*) FROM products`) {
		t.Errorf("shelf %d, shoes %d", a.PairsOnShelf, a.Shoes)
	}
	if got, want := shoeIDs(a), e.lowOracle(); !reflect.DeepEqual(got, want) {
		t.Errorf("running low %v, want %v", got, want)
	}
	if got, want := numbersOf(a), e.ids(`SELECT number FROM orders ORDER BY placed_at DESC, seq DESC LIMIT 5`); !reflect.DeepEqual(got, want) {
		t.Errorf("latest %v, want %v", got, want)
	}
}

// SC-15: for the sample set and for a generated one, every figure equals the
// sum over the stored non-cancelled orders and pairs.
func TestOverviewEqualsIndependentSums(t *testing.T) {
	e := newOverviewEnv(t)
	e.matchesOracle(e.read(overviewClock), overviewClock)

	e.exec(`INSERT INTO orders (id, number, placed_at, status, payment_method_id, delivery_option_id, customer, phone, address, city, total)
		SELECT 'tc-ov-gen-' || g, 'DOI-' || (200000 + g), '2026-11-25T12:00:00+07:00'::timestamptz - (g * interval '7 hours 13 minutes'),
		       (ARRAY['placed', 'confirmed', 'packed', 'shipped', 'delivered', 'cancelled'])[1 + g % 6],
		       'vnpay', 'standard', 'Gen ' || g, '0900000000', 'Addr', 'City', 1000000 + (g % 7) * 100000
		  FROM generate_series(1, 300) g`)
	e.exec(`WITH shoes AS (SELECT id, row_number() OVER (ORDER BY seq) - 1 AS k FROM products)
		INSERT INTO order_items (order_id, line_no, product_id, size, qty, unit_price)
		SELECT 'tc-ov-gen-' || g, n, s.id, '40', 1 + (g + n) % 3, 1000000
		  FROM generate_series(1, 300) g, generate_series(1, 3) n, shoes s
		 WHERE n <= 1 + g % 3 AND s.k = (g + n) % (SELECT count(*) FROM products)`)
	ict := time.FixedZone("ICT", 7*3600)
	for _, now := range []time.Time{overviewClock, time.Date(2026, 10, 15, 12, 0, 0, 0, ict), time.Date(2026, 9, 1, 0, 0, 0, 0, ict), time.Date(2026, 8, 31, 23, 59, 59, 0, ict)} {
		e.matchesOracle(e.read(now), now)
	}
}

// bigShop is the shop of SC-16 and SC-17: 10,000 orders of one to three pairs
// carrying sentinels in every private column, and 1,000 shoes of 5 to 12 sizes
// of 3 to 9 pairs.
func (e *overviewEnv) bigShop() {
	e.exec(`INSERT INTO orders (id, number, placed_at, status, payment_method_id, delivery_option_id, customer, phone, email, address, city, note, total)
		SELECT 'tc-ov-big-' || g, 'DOI-' || (100000 + g), '2026-11-25T12:00:00+07:00'::timestamptz - random() * interval '120 days',
		       (ARRAY['placed', 'confirmed', 'packed', 'shipped', 'delivered', 'cancelled'])[1 + g % 6],
		       'vnpay', 'standard', 'Big ' || g, '0999000TCOV', 'tcov@example.invalid', 'TCOV ADDRESS', 'C', 'TCOV NOTE', 1000000 + (g % 7) * 100000
		  FROM generate_series(1, 10000) g`)
	e.exec(`WITH shoes AS (SELECT id, row_number() OVER (ORDER BY seq) - 1 AS k FROM products)
		INSERT INTO order_items (order_id, line_no, product_id, size, qty, unit_price)
		SELECT 'tc-ov-big-' || g, n, s.id, '40', 1 + (g + n) % 3, 1000000
		  FROM generate_series(1, 10000) g, generate_series(1, 3) n, shoes s
		 WHERE n <= 1 + g % 3 AND s.k = (g + n) % (SELECT count(*) FROM products)`)
	e.exec(`INSERT INTO products (id, name, price, status) SELECT 'tc-ov-shoe-' || g, 'Shoe ' || g, 1000000, 'live' FROM generate_series(1, 1000) g`)
	e.exec(`INSERT INTO product_sizes (product_id, size, size_value, pairs)
		SELECT 'tc-ov-shoe-' || g, (35 + n)::text, 35 + n, 3 + (g + n) % 7 FROM generate_series(1, 1000) g, generate_series(1, 5 + g % 8) n`)
	for _, table := range []string{"orders", "order_items", "products", "product_sizes"} {
		e.exec(`ANALYZE ` + table)
	}
}

// SC-16: with 10,000 orders the answer holds five of them, none of a customer's
// private columns, and is no larger than the answer for 14.
func TestOverviewHoldsNoPrivateData(t *testing.T) {
	e := newOverviewEnv(t)
	small := e.body(overviewClock)
	e.bigShop()
	big := e.body(overviewClock)

	for _, sentinel := range []string{"0999000TCOV", "tcov@example.invalid", "TCOV ADDRESS", "TCOV NOTE"} {
		if strings.Contains(string(big), sentinel) {
			t.Errorf("the answer holds %q", sentinel)
		}
	}
	var a overviewAnswer
	if err := json.Unmarshal(big, &a); err != nil {
		t.Fatal(err)
	}
	if len(a.LatestOrders) != 5 {
		t.Errorf("%d orders in the answer", len(a.LatestOrders))
	}
	if want := e.ids(`SELECT number FROM orders ORDER BY placed_at DESC, seq DESC LIMIT 5`); !reflect.DeepEqual(numbersOf(a), want) {
		t.Errorf("latest %v, want %v", numbersOf(a), want)
	}
	if len(big) > 16*1024 || len(big) > 3*len(small) {
		t.Errorf("the answer is %d bytes for 10,000 orders and %d for 14", len(big), len(small))
	}
	e.matchesOracle(a, overviewClock)
}

// SC-17: 100 sequential requests at 10,000 orders and 1,000 shoes answer in at
// most 500 ms at the 95th percentile.
func TestOverviewP95(t *testing.T) {
	e := newOverviewEnv(t)
	e.bigShop()
	mux := http.NewServeMux()
	RegisterOverview(mux, e.pool, quiet(), func() time.Time { return overviewClock })
	took := make([]time.Duration, 0, 100)
	for range 100 {
		rec := httptest.NewRecorder()
		start := time.Now()
		mux.ServeHTTP(rec, overviewRequest(auth.Admin, http.MethodGet, "/v1/admin/overview"))
		took = append(took, time.Since(start))
		if rec.Code != 200 {
			t.Fatalf("status %d: %s", rec.Code, rec.Body)
		}
	}
	slices.Sort(took)
	if p95 := took[94]; p95 > 500*time.Millisecond {
		t.Errorf("p95 %v, the bound is 500ms (report to TL, ERD O2)", p95)
	}
}

// ERD §6: the schema this story stands on is as the merged migrations made it.
func TestOverviewSchemaAssumptions(t *testing.T) {
	e := newOverviewEnv(t)
	if got := e.text(`SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'orders' AND column_name = 'placed_at'`); got != "timestamp with time zone" {
		t.Errorf("orders.placed_at is %q", got)
	}
	column := func(name string) string {
		return e.text(`SELECT data_type || '/' || udt_name FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'product_stock' AND column_name = $1`, name)
	}
	if got := column("sizes_low"); got != "ARRAY/_text" {
		t.Errorf("product_stock.sizes_low is %q", got)
	}
	if got := column("stock"); got != "integer/int4" {
		t.Errorf("product_stock.stock is %q", got)
	}
	column("product_id")
	for _, index := range []string{"idx_orders_placed_at", "idx_orders_status_placed_at", "idx_order_items_product_id"} {
		if n := e.scalar(`SELECT count(*) FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid WHERE c.relname = $1 AND i.indisvalid AND c.relnamespace = to_regnamespace(current_schema())`, index); n != 1 {
			t.Errorf("index %s is missing or not valid", index)
		}
	}
	for table, constraint := range map[string]string{"orders": "uq_orders_seq", "products": "uq_products_seq", "lines": "uq_lines_seq"} {
		if n := e.scalar(`SELECT count(*) FROM pg_constraint WHERE conrelid = to_regclass($1) AND contype = 'u' AND conname = $2`, table, constraint); n != 1 {
			t.Errorf("%s has no unique %s", table, constraint)
		}
	}
}

// SC-1, SC-2: with real sessions, a visitor and a forged cookie get 401, a
// customer 403, and the admin and a staff member the same answer.
func TestOverviewSessionsAgainstPostgres(t *testing.T) {
	pool := dbtest.Migrated(t)
	ctx := context.Background()
	if err := auth.EnsureAdmin(ctx, pool, "dealer@example.com", "Dealer-Pass1", quiet()); err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword("Team-Pass-12345")
	if err != nil {
		t.Fatal(err)
	}
	for email, role := range map[string]auth.Role{"minh@example.com": auth.Staff, "customer@example.com": auth.Customer} {
		if _, err := auth.CreateUser(ctx, pool, auth.NewUser{Email: email, Name: "Member", Role: role, PasswordHash: hash}); err != nil {
			t.Fatal(err)
		}
	}
	svc := auth.New(pool, quiet())
	mux := http.NewServeMux()
	svc.Register(mux)
	RegisterOverview(mux, pool, quiet(), func() time.Time { return overviewClock })
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	h := svc.Resolve()(httpx.RecordRoute(mux))

	asked := 0
	do := func(req *http.Request, cookie string) *httptest.ResponseRecorder {
		asked++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:4000", asked)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "session", Value: cookie})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	signIn := func(email, password string) string {
		body, _ := json.Marshal(map[string]string{"email": email, "password": password})
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/sign-in", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := do(req, "")
		for _, c := range rec.Result().Cookies() {
			if c.Name == "session" && c.Value != "" {
				return c.Value
			}
		}
		t.Fatalf("%s cannot sign in: %d %s", email, rec.Code, rec.Body)
		return ""
	}
	get := func(cookie string) *httptest.ResponseRecorder {
		return do(httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil), cookie)
	}

	for _, cookie := range []string{"", "forged", strings.Repeat("a", 43)} {
		if rec := get(cookie); rec.Code != 401 || strings.Contains(rec.Body.String(), "waiting_orders") {
			t.Errorf("cookie %q: %d %s", cookie, rec.Code, rec.Body)
		}
	}
	if rec := get(signIn("customer@example.com", "Team-Pass-12345")); rec.Code != 403 || strings.Contains(rec.Body.String(), "waiting_orders") {
		t.Errorf("a customer: %d %s", rec.Code, rec.Body)
	}
	dealer := get(signIn("dealer@example.com", "Dealer-Pass1"))
	member := get(signIn("minh@example.com", "Team-Pass-12345"))
	if dealer.Code != 200 || member.Code != 200 || dealer.Body.String() != member.Body.String() {
		t.Errorf("the admin got %d, a staff member %d, bodies equal: %v", dealer.Code, member.Code, dealer.Body.String() == member.Body.String())
	}
	if rec := get(signIn("dealer@example.com", "Dealer-Pass1")); rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control %q", rec.Header().Get("Cache-Control"))
	}
}
