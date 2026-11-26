# ERD — Dealer dashboard figures and charts (plan item 11, "S10")

Module `admin` · Requirements ADMIN-020, ADMIN-021, ADMIN-004 (overview rows) · Story `docs/admin/stories/dealer-dashboard-figures-and-charts.md` · UI PR #76 · Engine PostgreSQL 16, pgx v5, hand-written SQL.

## 1. Verdict: this story adds nothing to the schema

**No table, column, constraint, index, trigger, view, function or migration is created, changed or dropped.** The story asked TL to decide whether an index helps the aggregates at 10,000 orders. It does not (§4). `erd/_base.md` §8 already said "S10 (dashboard: reads) needs no migration" and §5 pattern 12 already named the indexes this story reads; this design confirms both against the merged migrations (`20261125100100_collections`, `20261125120000_stock_per_size`).

| What the overview needs to know | Where the schema already says it |
|---|---|
| Orders placed in a calendar month, and in each of 14 days, in Vietnam time | `orders.placed_at` (`timestamptz`, so the instant is stored and the zone is only ever applied when it is read), `idx_orders_placed_at (placed_at DESC)` |
| An order that is cancelled is left out | `orders.status` (`ck_orders_status`), `idx_orders_status_placed_at (status, placed_at DESC)` |
| Orders waiting to be packed | `orders.status IN ('placed','confirmed')`, the same index |
| A sale's value | `orders.total` (`bigint`, VND, `>= 0`) — the shop's own sum; never recomputed from `order_items` here |
| Pairs on the shelf, the number of shoes, the shoes running low and their last-pairs sizes | the view `product_stock` (`stock`, `sizes_low`; one row per shoe, drafts included, via `LEFT JOIN`) over `product_sizes`. **The one definition of stock** (D3): the overview adds no second copy of it |
| Pairs sold by line | `order_items.qty` joined to `orders` (cancelled left out) and `products.line_id`, then to `lines`; `idx_order_items_product_id` |
| The order's first pair (shoe and size) | `order_items` with `line_no = 1` (`uq_order_items_order_id_line_no`), then `products.name` |
| Ties "in the shoes' own order", "in the lines' own order", "newest first" | `products.seq`, `lines.seq`, `orders.seq` — the unique, stable position every list of the project already uses (`erd/_base.md` §3.1) |

## 2. Entities this story touches — none changed, none written

The story is **read-only**: one `REPEATABLE READ, READ ONLY` transaction, five `SELECT`s. The database itself refuses a write inside it.

| Entity (defined elsewhere) | Columns read | Never read |
|---|---|---|
| `orders` | `id`, `seq`, `number`, `placed_at`, `status`, `customer`, `total` | **`phone`, `email`, `address`, `city`, `note`**, `carrier`, `tracking`, `eta`, `paid`, `payment_method_id`, `delivery_*`, `account_id`, `idempotency_key`, `is_sample`, `source`, `invented`. The five columns a customer's privacy depends on are in no `SELECT` list of this story, so no struct, map or `SELECT *` can carry them into a body (a test reads the SQL text for them, services file §8) |
| `order_items` | `order_id`, `line_no`, `product_id`, `size`, `qty` | `unit_price` (the order's `total` already is the sale), `id`, `created_at` |
| `products` | `id`, `seq`, `name`, `category_id`, `line_id`, `image` | everything else (price, status, description, bookkeeping). **Drafts are read like live shoes** — the dealer sees the whole shelf |
| `lines` | `id`, `seq`, `name` | `blurb`, `since`, `fit_note`, bookkeeping |
| `product_stock` (view over `products` ⟕ `product_sizes`) | `product_id`, `stock`, `sizes_low` | `sizes`, `sizes_out` |
| `users`, `sessions` | nothing here — the guard (`auth.Resolve`, `auth.Require`) reads them before the handler runs | — |

Nothing is written, so there is no lifecycle, retention or deletion rule to state: the answer is computed on every request and kept nowhere (no cache, no summary table — decision O3).

## 3. The shop's calendar

Every instant is stored as `timestamptz`. The overview reads the shop's day in **one expression**, from **one instant** the caller passes in (the handler's clock; the real clock in the running server, a fixed instant in a test):

```sql
($1::timestamptz AT TIME ZONE 'Asia/Ho_Chi_Minh')::date          -- "today", the shop's calendar date
```

| Window | From (inclusive) | To (exclusive) |
|---|---|---|
| This month | `date_trunc('month', today::timestamp) AT TIME ZONE 'Asia/Ho_Chi_Minh'` | the same plus `interval '1 month'` |
| The 14 days | `(today - 13)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh'` | `(today + 1)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh'` |
| One day of the 14 | `(placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date` equals that date | — |

`timestamp AT TIME ZONE zone` reads a local time in that zone and gives the instant; `timestamptz AT TIME ZONE zone` gives the local time. Written this way, an order at exactly `00:00:00` Vietnam time on the first of a month is **in** the new month and one at `23:59:59` the day before is **in** the old one, **whatever zone its value was written in** (a `+07:00` literal, a UTC one, `-05:00`): the comparison is between instants, so no session setting (`TimeZone`) can change a figure. Vietnam has no daylight saving and `Asia/Ho_Chi_Minh` is a fixed UTC+7, which is why the answer's text offset can be the literal `+07:00` (the same convention as `orders/read.go` and `orders/list.go`).

The upper bounds are **calendar boundaries, not "now"**: an order stamped later today than the clock still belongs to today and to this month. On the running shop this cannot happen (no order is in the future); in a test with a fixed clock it is the only reading that does not depend on the order's own time of day.

The 14 days may begin in the month before: `month` counts the current month only, `sales_by_day` counts the 14 days wherever they fall, so the 14 amounts add up to `month.sales` only when no day of the window is in an earlier month (they do at the SRS clock, 25/11/2026; they do not on 3/12/2026, by design — SC-6 and SC-15 say which sum is compared with which window).

## 4. Access patterns

| # | Pattern | Frequency | Index used | Cost at 10,000 orders, 1,000 shoes |
|---|---|---|---|---|
| 1 | **The figures** (one row): `sum(total)` and `count(*)` of non-cancelled orders in the month's range; `count(*)` where `status IN ('placed','confirmed')`; `sum(stock)` and `count(*)` of `product_stock` | Once per page view of `/admin` (a few a day; "Try again") | `idx_orders_placed_at` range scan for the month (≈ 2,500 orders if the shop sold 10,000 in four months), then the heap for `status` and `total`; `idx_orders_status_placed_at` for the waiting count (leading column `status`); `product_sizes` primary key order for the view's aggregate | milliseconds: three small scans and one aggregate of ≈ 10,000 `product_sizes` rows |
| 2 | **The 14 days**: group the window's non-cancelled orders by Vietnam date, joined to a 14-row series so a day with no sale is present as 0 | same | `idx_orders_placed_at` range scan (the window is ≤ 14 days) | milliseconds: the window holds a few hundred orders |
| 3 | **Pairs by line**: `order_items` ⨝ non-cancelled `orders` ⨝ `products` grouped by `line_id`, then `lines` left-joined to it | same | `orders` primary key and `products` primary key probes for each item; **every non-cancelled order is read** (all time, not the month) | ≈ 15,000–30,000 item rows: a hash aggregate of a few tens of milliseconds at most. **The only statement whose cost grows with the whole history** |
| 4 | **Running low**: `product_stock` ⨝ `products`, `cardinality(sizes_low) > 0 AND stock < 15`, fewest first, ties by `seq` | same | the view's aggregate (all shoes, because the filter is on aggregates); `products` primary key | milliseconds: ≈ 10,000 `product_sizes` rows aggregated, a handful of rows returned |
| 5 | **The five latest**: `ORDER BY placed_at DESC, seq DESC LIMIT 5`, each with its first pair (`line_no = 1`) and the shoe's name | same | `idx_orders_placed_at` gives the order; `uq_order_items_order_id_line_no` and `products` primary key give the five lookups | milliseconds |

**The five statements run in one transaction on one snapshot**, so the month's total, the waiting count, the 14 amounts, the line counts and the five latest are figures of the **same instant of the database**. With one statement per autocommit, an order placed between statement 1 and statement 2 would be in the chart and not in the month, and a test that adds them up would fail by exactly that order once in a while.

## 5. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| O1 | Does the story need a table, column or migration? | **No.** Every figure is a sum, a count or a list over columns that exist (§1). Nothing is stored for the overview | A `daily_sales` summary table: written by every order and every cancel, read by one page a few times a day, and a second place that must be kept equal to `orders` (ADMIN-020 AC-19 says the figures **equal** the sums over the stored orders) |
| O2 | Does an index help the aggregates at 10,000 orders and 1,000 shoes? | **No new index.** The month and the 14 days are range scans of `idx_orders_placed_at`; the waiting count is served by `idx_orders_status_placed_at`; the by-line sum reads each item once and probes by primary key; every other statement is over a few thousand rows. The p95 bound of 500 ms (SC-17) is two orders of magnitude above the cost in §4. **If SC-17 fails on the running shop, the failure is reported to TL and a follow-up design item adds the index, `CONCURRENTLY` in a file of its own (`erd/_base.md` §8 migrations 6, 9, 12 are the pattern); Dev does not add one on his own.** The first candidate, not built: `idx_orders_live_placed_at ON orders (placed_at DESC) INCLUDE (total) WHERE status <> 'cancelled'`, which makes the month and 14-day sums index-only | Building it now: every order placement and every status change would pay for it, for a read of a few milliseconds. An index that nobody has measured a need for is the cheapest thing to add later and the dearest to carry |
| O3 | Cache the answer, or keep a materialised copy? | **No.** `Cache-Control: no-store` on the answer; no in-process cache. A dealer who presses "Try again" or returns to `/admin` must see the order that arrived a minute ago (story decision 5; ADMIN-020 AC-16, AC-17) | A 30-second in-process cache: a stale figure is the one defect this page cannot have, and the cost it would save is a few milliseconds |
| O4 | One transaction or five? | **One `REPEATABLE READ, READ ONLY` transaction**, five statements, as the staff list does for its count and page (`staff_store.go`). It takes no lock and blocks no writer (MVCC), and it holds one pooled connection for the length of the request | One giant `SELECT … json_build_object(…)` statement: one snapshot as well, but a 60-line statement that nobody can test a part of; separate autocommit statements: no common snapshot (§4) |
| O5 | Is "pairs sold by line" this month's, or all time? | **All time**, as `services/_base.md` §3.25 and ADMIN-020 AC-4 / AC-19 say ("the sums over the stored, non-cancelled orders' … pairs"). Statement 3 therefore reads every non-cancelled order's items. At the stated 10,000 orders this is the dearest statement and still tens of milliseconds; at 1,000,000 orders it would not be. That is not a promised volume. **If it ever is, the answer is a window (a month or a year) — a change of the requirement, decided by PM — not a cache** | A month window now: it would change the SRS's own literal (Samba 8 over the month) and the old page's behaviour, for a cost nobody has measured |
| O6 | Where does the SQL live? | In `internal/admin/overview_store.go`, **read-only**, over `orders`, `order_items`, `products`, `lines` and the view `product_stock`. The ownership table of `services/_base.md` §1.1 names the only **writer** of each table; this file writes none of them, and its transaction is `READ ONLY`, so the database enforces it | A function in each owner (`orders.Totals`, `stock.Low`, `data.Lines`…): five exports used by one page, and the shared snapshot would still need one transaction passed through all of them |
| O7 | How is a shoe's stock counted for the shelf figure? | From **`product_stock`**, both `pairs_on_shelf` (`sum(stock)`) and `shoes` (`count(*)`). The view is "the only source of a shoe's `stock`" (D3), so the shelf figure, the running-low `stock` and the stock on a product page cannot disagree | `sum(product_sizes.pairs)` and `count(*) FROM products`: the same numbers today, and a second definition of stock that has to be kept equal to the view |
| O8 | What is "a shoe's own order", "a line's own order" and the order of equal orders? | `products.seq` for shoes, `lines.seq` for lines, `orders.seq` for orders. Ties in the five latest are broken by the **later** `seq` first (the order that was added last is the newer one). This is the order `GET /v1/products` and `GET /v1/lines` give with no `sort` (`services/_base.md` §2.4), which is how the test cases check it | An `ORDER BY name`: not the project's tie-break anywhere |
| O9 | An order with no first pair, a shoe with no category | Cannot happen through the API (`order_items` line 1 is written with the order; `RESTRICT` stops a shoe being deleted while ordered), but a repaired database could have it. The statement uses `LEFT JOIN` and `coalesce(…, '')`, so an order is **never dropped from the five latest** and a NULL never reaches a `string`. A shoe with no category answers `category: ""` (the project's rule: `NULL` is `""` for text, `erd/_base.md` §2.2) | An `INNER JOIN`: one damaged row would silently remove an order from the dealer's list |
| O10 | Is the running-low list capped? | **No** (story decision 6; `services/_base.md` §3.25 lists "every shoe running low"). Its bound is the number of shoes — at 1,000 shoes at most ≈ 150 KB. The page's "Running low" figure is the **length of the list**, so a cap would make the figure wrong unless the answer also carried a total, which is a contract change nobody has asked for | `LIMIT 10`: the figure would read 10 for a shop with 40 shoes running low |

### Conflicts with merged decisions

None. I read `erd/_base.md` §2, §3.2–§3.4, §3.9–§3.13, §5 pattern 12, §8, `erd/checkout-and-placing-an-order.md` (the view `product_stock`) and the merged migrations; `services/_base.md` §1.1, §2.2–§2.4, §3.25. Three statements are **refinements of silence**, not overrides: the upper bound of a window is a calendar boundary (§3), "a shoe's own order" is `seq` (O8), and the 14-day and month windows are compared as instants, not as session-zone text.

## 6. Migration plan

| Step | Forward | Backward | Safe on a populated table |
|---|---|---|---|
| This story | **No migration file.** The next free timestamp is not consumed | Revert the backend PR; nothing in any database changes | Not applicable: no statement is added to any migration, no table, row or index is touched at deploy time. Deployable and revertable with the schema exactly as migrations 1–12 of `erd/_base.md` §8 left it |

The story **depends on** `product_stock`, `idx_orders_placed_at`, `idx_orders_status_placed_at` and `idx_order_items_product_id` being as the merged migrations made them. The backend PR therefore ships one test, `TestOverviewSchemaAssumptions` (skips without `TEST_DATABASE_URL`), that reads the catalogue and fails loudly if any of these is not true: `orders.placed_at` is `timestamp with time zone`; the view `product_stock` exists and has the columns `product_id`, `sizes_low` (`text[]`) and `stock` (`integer`); the three indexes exist and are valid (`pg_index.indisvalid`); `orders`, `products` and `lines` each have the unique `seq` column. No merged migration is edited (convention `migrations`).

## 7. Arranging and checking test data

No statement below touches the schema. A database test makes its own schema with `dbtest.Migrated(t)` (the 14 sample orders, 12 shoes, 5 lines, the placeholder counts), so the SRS literals can be asserted at a **fixed clock of 25/11/2026 12:00 Vietnam time** (`time.Date(2026, 11, 25, 12, 0, 0, 0, time.FixedZone("ICT", 7*3600))`). On the running shop the clock is real and the page is compared with the endpoint and the endpoint with sums, never with a literal.

```sql
-- The oracle of SC-15: written independently of the handler's statements (a plain join and a filter, no CTE, no generate_series).
-- This month (the month of the clock, Vietnam time):
SELECT coalesce(sum(total), 0), count(*) FROM orders
 WHERE status <> 'cancelled'
   AND to_char(placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM') = $1;            -- $1 = '2026-11'
-- One day:
SELECT coalesce(sum(total), 0) FROM orders
 WHERE status <> 'cancelled'
   AND to_char(placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD') = $1;         -- $1 = '2026-11-25'
-- One line's pairs:
SELECT coalesce(sum(i.qty), 0) FROM order_items i
  JOIN orders o ON o.id = i.order_id JOIN products p ON p.id = i.product_id
 WHERE o.status <> 'cancelled' AND p.line_id = $1;                                    -- $1 = 'samba'
-- Waiting:
SELECT count(*) FROM orders WHERE status = 'placed' OR status = 'confirmed';

-- SC-9 / SC-10: a database holding only the shoes a case names (a test's own schema, so it may empty the tables):
DELETE FROM order_items; DELETE FROM orders; DELETE FROM products;                     -- product_sizes goes by CASCADE
INSERT INTO products (id, name, price, status) VALUES ('t-a', 'A', 1, 'live'), ('t-b', 'B', 1, 'draft');
INSERT INTO product_sizes (product_id, size, size_value, pairs) VALUES ('t-a', '40', 40, 2), ('t-a', '41', 41, 3), ('t-b', '42', 42, 10);

-- SC-11: an order at the first instant of a month, written three ways; all three are in December (month starts 2026-11-30 17:00 UTC):
INSERT INTO orders (id, number, placed_at, status, payment_method_id, delivery_option_id, customer, phone, address, city, total)
VALUES ('t-1', 'DOI-90001', '2026-12-01T00:00:00+07:00', 'delivered', 'vnpay', 'standard', 'T', '0900000001', 'A', 'C', 1000000),
       ('t-2', 'DOI-90002', '2026-11-30T17:00:00+00:00', 'delivered', 'vnpay', 'standard', 'T', '0900000002', 'A', 'C', 2000000),
       ('t-3', 'DOI-90003', '2026-11-30T12:00:00-05:00', 'delivered', 'vnpay', 'standard', 'T', '0900000003', 'A', 'C', 4000000);
-- and one second earlier, in November, whatever its zone:
--   '2026-11-30T23:59:59+07:00'  '2026-11-30T16:59:59+00:00'  '2026-11-30T11:59:59-05:00'

-- SC-14 (cancel half, no cancel route on main until item 17): the status set by SQL
UPDATE orders SET status = 'cancelled' WHERE id = 'doi-24816';

-- SC-16 / SC-17: the big set (generated; every status, 1-3 pairs, the sentinels)
INSERT INTO orders (id, number, placed_at, status, payment_method_id, delivery_option_id, customer, phone, email, address, city, note, total)
SELECT 'tc-ov-big-' || g, 'DOI-' || (100000 + g), now() - (random() * interval '120 days'),
       (ARRAY['placed','confirmed','packed','shipped','delivered','cancelled'])[1 + g % 6],
       'vnpay', 'standard', 'Big ' || g, '0999000TCOV', 'tcov@example.invalid', 'TCOV ADDRESS', 'C', 'TCOV NOTE', 1000000 + (g % 7) * 100000
FROM generate_series(1, 10000) g;
-- (and one to three order_items rows each, line_no 1..n, from any ten of the shoes; then 1,000 shoes with 5-12 sizes of 3-9 pairs)
ANALYZE orders; ANALYZE order_items; ANALYZE product_sizes;                              -- before timing SC-17
```

## 8. Hand-over

| Who | What they must know from this design |
|---|---|
| Dev, backend step | **No migration.** Two files in `internal/admin` (`overview.go`, `overview_store.go`), the five statements of the services file §5 written as given, one `Register` line in `server.go`, the swap of the services file §9. Every identifier of the two files starts with `overview` so that "Manage shoes…" (also adding files to `internal/admin`, in parallel) collides with nothing |
| Test lead | The statements of §7 are the fixture and the oracle. The SRS literals (47.660.000 ₫, 13 orders, the 14 amounts, the line counts, the five latest) hold at the **fixed clock** in a backend test over the migrated sample set; on the running shop compare sums, as the story's §8 says. SC-14's cancel half is a SQL `UPDATE` until item 17 lands |
| "Manage shoes, lines, categories, reviews and orders" (item 10) and "Dealer handles orders" (item 17) | They write the rows this story reads (pairs, shoes, lines, order status). A shoe's pairs change through `internal/stock` and an order's status through `internal/orders`; this story takes no lock, so neither waits for it and it waits for neither |
