# Service contracts — Dealer dashboard figures and charts (plan item 11, "S10")

Module `admin` · Requirements ADMIN-020, ADMIN-021, ADMIN-004 (overview rows) · Story `docs/admin/stories/dealer-dashboard-figures-and-charts.md` · UI PR #76 · Tables: `docs/architecture/erd/dealer-dashboard-figures-and-charts.md` (**none changed**).

This file **extends** `services/_base.md` §3.25. The route, its auth, its field names, its definitions and its error codes **stand**. It adds the order in which a request is refused, the five statements that compute the answer (written out, because the figures are only as right as the SQL), the clock, the Go surface, the tests the backend PR ships, and the frontend's swap from its mock. It adds **no error code** to the catalog (§2.3), no second error or list shape, no table, no setting and no secret. Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/admin/overview` reaches Go as `/v1/admin/overview`.

## 1. Endpoint

| # | Method and path | Auth | Traces to |
|---|---|---|---|
| 3.25 | `GET /v1/admin/overview` | dealer: `auth.Require(auth.Admin, auth.Staff)` | ADMIN-020 items 1–4, ADMIN-021 item 3, ADMIN-004 |

**Mounting.** One literal pattern, registered by one function in `internal/admin/overview.go`:

```go
// RegisterOverview mounts GET /v1/admin/overview. now is the clock: the real one in the running
// server (server.New passes time.Now), a fixed instant in a test; nil means time.Now.
func RegisterOverview(mux *http.ServeMux, db overviewDB, log *slog.Logger, now func() time.Time)
```

`server.New` gains **one line**, `admin.RegisterOverview(mux, pool, log, time.Now)`, after `admin.RegisterStaff`. ("Manage shoes…" also adds a line there, `admin.RegisterRecords`; whichever PR merges second keeps both lines.) Any other method or path falls to the server's catch-all and answers `404 not_found`; no handler is written for it. **No new setting, secret or `.env.example` line, and no environment variable or request parameter moves the clock.**

**On every answer** (200 and every refusal the handler writes): `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`, and the `X-Request-Id` the middleware echoes. `no-store` is what makes "every visit asks again" true: without it a browser may serve the second visit from its cache, and `src/lib/api.ts` has no cache switch of its own.

## 2. The mock, read and judged

The UI PR (#76) adds `src/lib/mock/dealer-dashboard-figures-and-charts.ts`, imported only by `src/lib/overview.ts` through one `read` constant (checked at the UI review, which did not review the mock's contents). It is unreviewed input; its shape is compared here with `_base.md` §3.25, which decides it.

| Mock / frontend type | Contract | Judgment | Frontend change at the swap |
|---|---|---|---|
| Top level `month {sales, orders}`, `waiting_orders`, `pairs_on_shelf`, `shoes`, `running_low`, `sales_by_day`, `pairs_by_line`, `latest_orders`; money and counts are numbers | §3.25: the same eight names; whole-number VND and counts | **Sound.** Field for field | None |
| `LowShoe {id, name, category, image, stock, sizes_low}`; `category` is the category's id | §3.25: the same six; `category` is the category id | **Sound.** The mock types `category` as `string`; the API says `""` for a shoe with no category (the project's rule for unset text) — the page does not read it | None |
| `DaySales {date, sales}`; `date` is `"2026-11-12"` text and the page takes the day of the month from the text, never through a `Date` | §3.25: `date` is the shop's own day, `YYYY-MM-DD`, cut at Vietnam midnight | **Sound, and it is the right thing to do**: the API gives the shop's days, the page must not re-cut them in the browser's zone | None |
| `LinePairs {line, name, pairs}` in the answer's order | §3.25: the same; most pairs first | **Sound.** The page draws the order it is given and sorts nothing | None |
| `LatestOrder {id, number, customer, product, product_name, size, status, placed_at, total}`; `placed_at` has `+07:00` | §3.25: the same nine; `placed_at` RFC 3339 with `+07:00` | **Sound.** The mock's `product` is the seed's id `handball-spezial-w-earth-strata`; the example in `_base.md` §3.25 (`handball-spezial-earth-strata`) is **wrong** and the seed's id is right — corrected in §4 below | None |
| `isOverview` accepts an answer only if the five numbers are numbers and the four lists are **arrays**; anything else is the one error view | The answer holds the four lists as arrays **always**, `[]` when empty, never `null` (SC-12) | **A contract point this design makes explicit.** A Go `nil` slice encodes as `null`, and the page would then say "We could not load the overview." for a shop with no orders. The store therefore builds every list with `make(…, 0, n)` and a test reads the raw body for `[]` (§8) | None |
| `isOverview` does not check the shape of each row (UI review nit) | The contract fixes every row | **Accepted.** A row the API cannot get wrong (it is written from a Go struct) need not be re-checked in the browser | None |
| `readOverview()` resolves with the sample after 400 ms and never fails; the loading, failure and retry views are driven by the tests | `api.get` rejects with `ApiError` (`status` 0 for no connection or the 15 s timeout; 401, 403, 5xx as the shop answers) and tries a failed GET (no answer or 5xx, never a 4xx) once more | **Same idea, other type.** `useOverview` already treats any rejection, a 401 or 403 included, as the one error view (story decision 3), so no branch changes | None to the hook; the tests' `vi.mock` is repointed (below) |
| The time on a latest-orders row is formatted by the browser (`formatDateTime`), the chart's day labels by the shop | The API gives the instant with `+07:00`; what zone a row is shown in is not the API's | **Not a backend matter.** UI review and test-case §5 point 5 already ask PM whether the row should be pinned to Vietnam time; the answer is the same either way | None here |

**The frontend change at the swap is one line and one deletion**, nothing else (§9). No page, component, `content.json` key, `AdminBars` or `AdminLowStock` changes.

## 3. Order of checks

Each step runs only when the earlier ones passed; a request is answered by **the first step that refuses it**.

| # | Check | Refusal |
|---|---|---|
| 1 | `auth.Resolve` has found a live session (re-read from `users` on this request, so a switched-off staff member is a `401`, not a `403`) | `401 unauthenticated` — a forged or expired cookie too |
| 2 | The session's role is `admin` or `staff` | `403 forbidden` for `customer`. **Nothing of the request has been read yet** and no figure has been computed: the guard answers first (SC-1, SC-2) |
| 3 | **The query string and the body are never read.** An unknown parameter is ignored and the answer is identical (decision V1); in particular `?now=…`, `?date=…`, `?month=…` move nothing | — |
| 4 | The clock is read **once**, `now := h.now()`, and passed to the store; the request's context gets a 5-second deadline (`overviewTimeout`) | — |
| 5 | One `REPEATABLE READ, READ ONLY` transaction (`BeginTx`); the five statements of §5 in that order; every list is built non-nil | `500 internal` on any failure of `BeginTx` or of a statement, including a deadline |
| 6 | Success | `200` the answer of §4 |

A request the browser abandoned (`context.Canceled` while `r.Context()` is itself done) is not an error: nothing is written and nothing is logged as one. A deadline that ran out is `500`.

## 4. The answer

`200 OK`, one JSON object of **exactly** these eight keys, written from a Go struct (§7), never from a row or a map:

```json
{
  "month":  { "sales": 47660000, "orders": 13 },
  "waiting_orders": 2,
  "pairs_on_shelf": 73,
  "shoes": 3,
  "running_low": [
    { "id": "gazelle-indoor-solar-gold", "name": "Gazelle Indoor Solar Gold", "category": "men",
      "image": "/images/…", "stock": 5, "sizes_low": ["40", "41"] }
  ],
  "sales_by_day": [ { "date": "2026-11-12", "sales": 2790000 }, { "date": "2026-11-13", "sales": 3590000 } ],
  "pairs_by_line": [ { "line": "samba", "name": "Samba", "pairs": 8 } ],
  "latest_orders": [
    { "id": "doi-24816", "number": "DOI-24816", "customer": "Trang", "product": "handball-spezial-w-earth-strata",
      "product_name": "Handball Spezial Earth Strata", "size": "37 1/3", "status": "placed",
      "placed_at": "2026-11-25T08:40:00+07:00", "total": 3290000 }
  ]
}
```

(The example is shortened: a real answer holds 14 days and up to five orders. `_base.md` §3.25 wrote `handball-spezial-earth-strata`; the seeded shoe is `handball-spezial-w-earth-strata`.)

| Field | Type | Definition (the text of `_base.md` §3.25 stands; this column adds what it left open) |
|---|---|---|
| `month.sales` | integer | Σ `orders.total` of orders **not cancelled** whose `placed_at` is in the shop's current calendar month, `[00:00:00 on the 1st, 00:00:00 on the 1st of the next month)` in Vietnam time (ERD §3). `0` with none |
| `month.orders` | integer | The count of those orders |
| `waiting_orders` | integer | Orders with status `placed` or `confirmed`, **of any date** |
| `pairs_on_shelf` | integer | Σ `product_stock.stock`, **drafts included** |
| `shoes` | integer | The number of rows of `product_stock` — every shoe, drafts included |
| `running_low[]` | array | Shoes with at least one size holding 1 or 2 pairs **and** `stock` < 15, drafts included. **Fewest `stock` first, ties by `products.seq`**; the list is not capped (ERD O10). Each: `id`, `name`, `category` (the category's id, `""` when none), `image` (the shoe's `image`, `""` when none), `stock` (integer, pairs in all), `sizes_low` (the sizes holding 1 or 2 pairs, growing order, as on the shoe) |
| `sales_by_day[]` | array | **Exactly 14** entries: the shop's today and the 13 days before it, **oldest first**. `date` is `YYYY-MM-DD`; `sales` is Σ `total` of non-cancelled orders whose Vietnam date is that day, `0` when none |
| `pairs_by_line[]` | array | **Every line**, `0` when nothing sold: `line` (its id), `name`, `pairs` = Σ `order_items.qty` over **all** non-cancelled orders (not only this month, ERD O5) of shoes of that line. **Most pairs first, ties by `lines.seq`.** A shoe with no line is in no row |
| `latest_orders[]` | array | The **five** newest orders of **any** status (cancelled included, status `cancelled`), newest first, ties by the later `seq`. Each: `id`, `number`, `customer`, `product` and `product_name` and `size` (of the order's **first pair**, `line_no = 1`), `status`, `placed_at` (`YYYY-MM-DDTHH:MM:SS+07:00`), `total`. **No `phone`, `email`, `address`, `city`, `note`, token, payment or item list — and no other order** |

Money and counts are JSON integers (never a string, never a fraction); every list is a JSON array — `[]`, never `null`. With no orders, no shoes and no lines the answer is `200` with every number `0`, `running_low` `[]`, 14 entries of `0`, `pairs_by_line` `[]` and `latest_orders` `[]` (SC-12). With lines and no orders every line is present with `0`.

## 4.1 Errors

| Code | HTTP | Trigger | `message` |
|---|---|---|---|
| `unauthenticated` | 401 | No session, a forged, expired or ended one, a switched-off account | the guard's (`auth.Require` / `auth.Resolve` — this story adds no wording) |
| `forbidden` | 403 | A `customer` session | the guard's |
| `internal` | 500 | The database cannot be reached, `BeginTx` or any statement fails, the 5-second deadline runs out | "Something went wrong. Please try again." — **nothing else**: no SQL, table, column, connection detail or id. The cause is logged through `auth.LogFailure` (a database error by SQLSTATE and constraint only, so a row can never reach the log) |

These are `_base.md` §2.3's own codes and sentences. There is no `400` (no input is read), no `404` for this route (a different method or path is the catch-all's), no `429` and no `503`: `unavailable` is reserved for the process shutting down and `/healthz`; a database that cannot be reached is `internal` (`_base.md` §2.3). **With no database even `auth.Resolve` answers `500` first**, so the case that proves this handler's own `500` is a working session over a failed statement (SC-13).

## 5. The five statements

All five run inside the one read-only snapshot of §3 step 5, in this order. `$1` is `now`, a `time.Time` (pgx sends the instant; no zone is involved). Dev writes them as constants in `overview_store.go`; **a statement may be reformatted but not changed in meaning** — a change to what one computes is a change to this file in the same PR. None of them has a `SELECT *`, and none names `phone`, `email`, `address`, `city` or `note`.

**5.1 The figures** — one row, scanned into five `int64`.

```sql
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
       (SELECT coalesce(sum(stock), 0)::bigint AS pairs, count(*)::bigint AS shoes FROM product_stock) p
```

**5.2 The 14 days** — exactly 14 rows, oldest first, scanned into `(date string, sales int64)`.

```sql
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
 ORDER BY d.day
```

The series is an **integer** `generate_series` added to a `date`, never `generate_series(date, date, interval)`: that form resolves to `timestamptz` and its `::date` would then depend on the session's `TimeZone`.

**5.3 Pairs by line** — one row per line, scanned into `(line, name string, pairs int64)`.

```sql
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
 ORDER BY coalesce(sold.pairs, 0) DESC, l.seq
```

**5.4 Running low** — scanned into `(id, name, category, image string, stock int64, sizes_low []string)`.

```sql
SELECT p.id, p.name, coalesce(p.category_id, '') AS category, p.image, v.stock, v.sizes_low
  FROM product_stock v
  JOIN products p ON p.id = v.product_id
 WHERE cardinality(v.sizes_low) > 0 AND v.stock < 15
 ORDER BY v.stock, p.seq
```

**5.5 The five latest orders** — scanned into nine columns.

```sql
SELECT o.id, o.number, o.customer,
       coalesce(f.product_id, '') AS product, coalesce(p.name, '') AS product_name, coalesce(f.size, '') AS size,
       o.status,
       to_char(o.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD"T"HH24:MI:SS') || '+07:00' AS placed_at,
       o.total
  FROM orders o
  LEFT JOIN LATERAL (SELECT i.product_id, i.size FROM order_items i WHERE i.order_id = o.id AND i.line_no = 1) f ON true
  LEFT JOIN products p ON p.id = f.product_id
 ORDER BY o.placed_at DESC, o.seq DESC
 LIMIT 5
```

`placed_at` is formatted by the same expression `orders/read.go` and `orders/list.go` use, so an order reads the same second in the overview, the list and the order page.

## 6. Limits and cost

| Aspect | Decision |
|---|---|
| Deadline | `overviewTimeout = 5 * time.Second` on the request's context: ten times the p95 bound, and shorter than the browser client's 15 s, so the handler answers `500` before `api.ts` gives up |
| Connections | One pooled connection per request, for the length of five statements. The overview takes no lock, so it neither waits for an order being placed nor makes one wait |
| Size of the answer | A few kilobytes: 14 days, ≤ 5 orders, one row per line, and the running-low list (≤ the number of shoes). It is the same size with 14 orders and with 10,000 (SC-16) |
| p95 | ≤ 500 ms at 10,000 orders and 1,000 shoes (SC-17); the estimate of ERD §4 is two orders of magnitude below it. No index and no cache is added to meet it (ERD O2, O3) |
| Retries | None server-side. `api.ts` tries a failed GET (no answer or 5xx) once more; a `401` or `403` is never retried |

## 7. The Go surface

Two new files in `internal/admin`, **every identifier starting with `overview` or `Overview`** so that "Manage shoes…" (also adding files to the package, in parallel) collides with nothing. The handler holds **no SQL**; the store holds all of it. The package's existing `reply`, `refuse`, `internalMessage` and `Pool` (`staff.go`, `staff_store.go`) are reused, not redefined.

```go
// overview.go
type overviewDB interface{ BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) } // *pgxpool.Pool and admin.Pool satisfy it

type overviewReader interface {
	read(ctx context.Context, now time.Time) (overviewAnswer, error)
}

type overviewHandler struct {
	reader overviewReader
	log    *slog.Logger
	now    func() time.Time
}

func RegisterOverview(mux *http.ServeMux, db overviewDB, log *slog.Logger, now func() time.Time)  // §1
func (h *overviewHandler) mount(mux *http.ServeMux)  // mux.Handle("GET /v1/admin/overview", auth.Require(auth.Admin, auth.Staff)(http.HandlerFunc(h.get)))
func (h *overviewHandler) get(w http.ResponseWriter, r *http.Request)  // §3 steps 3–6; reply(w, 200, a); refuse(w, 500, "internal", internalMessage, nil)

// The answer. These structs ARE the privacy rule: a field that is not here cannot be sent.
type overviewAnswer struct {
	Month         overviewMonth     `json:"month"`
	WaitingOrders int64             `json:"waiting_orders"`
	PairsOnShelf  int64             `json:"pairs_on_shelf"`
	Shoes         int64             `json:"shoes"`
	RunningLow    []overviewLowShoe `json:"running_low"`
	SalesByDay    []overviewDay     `json:"sales_by_day"`
	PairsByLine   []overviewLine    `json:"pairs_by_line"`
	LatestOrders  []overviewOrder   `json:"latest_orders"`
}
type overviewMonth struct { Sales int64 `json:"sales"`; Orders int64 `json:"orders"` }
type overviewLowShoe struct {
	ID string `json:"id"`; Name string `json:"name"`; Category string `json:"category"`; Image string `json:"image"`
	Stock int64 `json:"stock"`; SizesLow []string `json:"sizes_low"`
}
type overviewDay struct { Date string `json:"date"`; Sales int64 `json:"sales"` }
type overviewLine struct { Line string `json:"line"`; Name string `json:"name"`; Pairs int64 `json:"pairs"` }
type overviewOrder struct {
	ID string `json:"id"`; Number string `json:"number"`; Customer string `json:"customer"`
	Product string `json:"product"`; ProductName string `json:"product_name"`; Size string `json:"size"`
	Status string `json:"status"`; PlacedAt string `json:"placed_at"`; Total int64 `json:"total"`
}

// overview_store.go
type overviewStore struct{ db overviewDB }
func (s overviewStore) read(ctx context.Context, now time.Time) (overviewAnswer, error)
// BeginTx(pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}); defer Rollback(context.WithoutCancel(ctx));
// the five statements of §5; every list is made with make(T, 0, n) so it encodes as [] and never as null; rows.Err() is checked.
```

`get` on a store error: `errors.Is(err, context.Canceled) && r.Context().Err() != nil` → return with nothing written; otherwise `auth.LogFailure(h.log, "overview request failed", "overview.read", err)` and the `500` above.

## 8. Tests the backend PR ships

Database cases use `dbtest.Migrated(t)` (a schema of its own with the 14 sample orders, 12 shoes and 5 lines; skipped without `TEST_DATABASE_URL`, proved first in CI) and the **fixed clock** `time.Date(2026, 11, 25, 12, 0, 0, 0, time.FixedZone("ICT", 7*3600))`. A database test serves the handler with `auth.WithUser` in the context as the staff harness does, so `auth.Resolve` is not what fails when a case breaks the database. The expected values of a case that is not the sample are computed by the oracle SQL of ERD §7, which is **not** the handler's statement.

| Test | What it proves | SC |
|---|---|---|
| `TestOverviewIsDealerOnly` (no database: a fake reader that fails the test if called) | No user → `401`, a `customer` → `403`, both before the reader runs, with the error envelope and none of the eight keys; `admin` and `staff` → `200` | SC-1, SC-2 |
| `TestOverviewShape` (fake reader) | The body's top-level keys are exactly the eight; each row has exactly its keys (decode into `map[string]any`); numbers have no decimal point; `Cache-Control: no-store`; `Content-Type: application/json; charset=utf-8`; `?zzz=1&now=2020-01-01&month=1` gives the **identical** body; with an empty answer every list is `[]` in the **raw** bytes, never `null` | SC-3, SC-12 |
| `TestOverviewDatabaseFailureIsGeneric500` | A reader that fails (an error text holding SQL, a table name and the sentinel `0999000TCOV`) → `500`, `{"error":"internal","message":"Something went wrong. Please try again."}` and nothing else; the log holds the cause by SQLSTATE only. A **closed pool** through the real store gives the same | SC-13 |
| `TestOverviewAbandonedRequestWritesNothing` | A cancelled request context → no body, no `500` logged as an error | §3 |
| `TestOverviewSampleSetAtTheSRSClock` | `month` 47660000 / 13; `waiting_orders` 2; `sales_by_day` 14 entries `2026-11-12`…`2026-11-25` with the SRS amounts, summing to 47660000, `2026-11-16` and `2026-11-20` 0; `pairs_by_line` Samba 8, Gazelle 2, Megaride 2, Handball Spezial 1, Adizero 1 in that order; `latest_orders` DOI-24816, 24815, 24814, 24817, 24813 with `placed_at` `…+07:00` and `product`/`size` of the first pair. **Running low is compared with the oracle, not with a literal:** the seed's placeholder counts (ERD Q1) decide which shoes qualify | SC-4 to SC-8 |
| `TestOverviewCancelledAndWaiting` | Setting each status on DOI-24816 in turn moves `waiting_orders` only for `placed` / `confirmed`; cancelling DOI-24813 removes it from `month`, its day and its line and **keeps it among the five latest as `cancelled`**; placing an order through `orders.Place` (or inserting one) changes `month`, `waiting_orders`, its day and its line on the next read | SC-5, SC-7, SC-14 |
| `TestOverviewShelfAndRunningLow` | In an emptied schema: shoes of 18, 24, 31 pairs → `pairs_on_shelf` 73, `shoes` 3; a draft of 10 adds 10 and counts; 5, 14, 15, 18 pairs and sizes of 1 or 2 pairs → 5 and 14 listed, 5 first, 15 and 18 absent; 9 pairs with no 1–2 size absent; a size of exactly 2 in, exactly 3 out; a qualifying draft in; two equal stocks keep `products.seq` order; each row has the six keys | SC-9, SC-10 |
| `TestOverviewMonthAndDayBoundaries` | With the clock at `2026-12-01 12:00 +07:00`, orders at `2026-11-30 23:59:59 +07:00`, `2026-11-30 16:59:59 +00:00`, `2026-11-30 11:59:59 -05:00` are November's; `2026-12-01 00:00:00 +07:00`, `2026-11-30 17:00:00 +00:00`, `2026-11-30 12:00:00 -05:00` are December's: `month` holds only the second three, `2026-12-01` holds them and `2026-11-30` holds the first three. The same test runs with the **session `TimeZone` set to `UTC` and to `America/New_York`** and gives the same answer. A clock at 00:00:00 and at 23:59:59 Vietnam time reads the same "today" it names | SC-11 |
| `TestOverviewLinesAndPairs` | An order of two pairs, a Samba and a Gazelle, adds each pair's `qty` to its own shoe's line; a line nothing sold is present with 0 and, with ties, in `lines.seq` order; a shoe with no line is in no row; no orders at all → every line present with 0 | SC-7, SC-12 |
| `TestOverviewEqualsIndependentSums` | A generated set (a few hundred orders over three months, every status, one to three pairs, several shoes and lines, written in `+07:00` and UTC) → `month`, each of the 14 days, each line's pairs and `waiting_orders` equal the oracle SQL, field by field; **the sample set too** | SC-15 |
| `TestOverviewHoldsNoPrivateData` | 10,000 generated orders carrying the sentinels `0999000TCOV`, `tcov@example.invalid`, `TCOV ADDRESS`, `TCOV NOTE` → the raw body contains none of them, holds exactly five orders, and is within a few kilobytes of the body for 14 orders. **Also a test over the store's own SQL text:** none of `phone`, `email`, `address`, `note`, `city` appears in any of the five constants | SC-16 |
| `TestOverviewP95` | 10,000 orders and 1,000 shoes (5–12 sizes of 3–9 pairs), `ANALYZE`, 100 sequential requests through the handler → p95 ≤ 500 ms. If it fails on a shared runner it is reported to TL (ERD O2), not relaxed | SC-17 |
| `TestOverviewSchemaAssumptions` | ERD §6 | — |

Frontend tests: `dealer-dashboard-overview.test.tsx` keeps its 13 cases; only its source of the answer changes (§9).

## 9. The frontend's swap from its mock

| Step | In `code/frontend` |
|---|---|
| 1 | `src/lib/overview.ts`: delete `import { readOverview } from "./mock/dealer-dashboard-figures-and-charts"`; add `import { api } from "./api"`; make the one source `const read: () => Promise<unknown> = () => api.get<unknown>("admin/overview");` and fix its comment. **Nothing else in the file changes**: the types, `isOverview`, `dayOfMonth`, `useOverview` and the retry stay as approved |
| 2 | Delete `src/lib/mock/dealer-dashboard-figures-and-charts.ts` (it is the story's only mock) |
| 3 | `src/test/dealer-dashboard-overview.test.tsx`: move `sampleOverview` and `emptyOverview` into the test file; repoint `vi.mock` from the mock module to `../lib/api`, keeping the real `ApiError` (`vi.mock("../lib/api", async (original) => ({ ...(await original<typeof import("../lib/api")>()), api: { ...(await original<typeof import("../lib/api")>()).api, get: vi.fn() } }))`), and make `read` the mocked `api.get`; the failure cases reject with `new ApiError(500, "internal", "…")`, `new ApiError(0, "network", "…")`, `new ApiError(401, …)` and `new ApiError(403, …)` |
| 4 | `grep -rn "lib/mock/dealer-dashboard" code/frontend/src` finds nothing (CI warns while a mock is imported). `AdminOverview.tsx` already imports no `useCollection`, `Order`, `Product` or `Line` |

**No `content.json` key, component, page, route (`/admin` is routed inside `SignedIn` since "Dealer sign-in"), `src/App.tsx` or `api.ts` change.** The GET carries the session cookie (`credentials: "include"`); a failed GET is two requests (no answer or 5xx) and a `401` / `403` is one, which the page shows, as it shows every failure, as "We could not load the overview." (story decision 3).

**Branch.** The backend branch is cut from the UI branch `feature/dealer-dashboard-figures-and-charts-ui`: `overview.ts`, its test and the mock exist only there. PR #76 merges first (flow steps 11–12), then the backend PR.

## 10. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| V1 | An undefined query parameter on a route that takes none (SC-3; test-case §5 point 4) | **Ignored**: `200` and a body identical to the one without it. No input is read, so none can be refused. This is `_base.md` §2.4's rule for a URL a person edited, and it holds for `?now=` as much as for `?zzz=` | `400 bad_request`: this route has no parameter to name, and a dealer's bookmark with a stray `?utm_source=` would stop working. A `?now=` that moved the clock: it would let a test route into production |
| V2 | A database that cannot be reached (test-case §5 point 3) | **`500 internal`**, as `_base.md` §2.3 says ("including … a database that cannot be reached"). `503 unavailable` is the process shutting down and `/healthz` | `503` for `BeginTx` and `500` for a statement: two codes for one condition, and the page shows both as the same sentence |
| V3 | Where does "today" come from? | **A clock the handler is given**: `RegisterOverview(…, now func() time.Time)`, `time.Now` from `server.New`, a fixed instant in a test. It is read once per request and handed to the store as `$1`; the database's `now()` is **not** used for "today" here (it is the clock of every other story's windows, but a test cannot move it) | An environment variable or a query parameter for the date: a deployed server's figures would be one request away from being wrong. `now()` inside the statements: the SRS literals could then only be proved on the day they happen to be true |
| V4 | One snapshot or five autocommit statements | **One `REPEATABLE READ, READ ONLY` transaction** (ERD O4) | See ERD O4 |
| V5 | Where does the SQL live, and may `internal/admin` hold it? | In `overview_store.go`. `staff.go` says `internal/admin` "holds no SQL" because the **accounts** have an owner (`internal/auth`); the overview is a **read model across five tables** that no package owns, and it writes none of them (ERD O6) | Five exported functions in five packages for one page |
| V6 | How is a latest order's `placed_at` written? | By the statement, with the expression `orders/read.go` and `orders/list.go` already use (`to_char(… AT TIME ZONE 'Asia/Ho_Chi_Minh', …) \|\| '+07:00'`). `Asia/Ho_Chi_Minh` is a fixed UTC+7, so the literal offset is exact | Marshalling a Go `time.Time` in a fixed zone: also `+07:00`, but a third place that formats the same instant |
| V7 | Logging a failure | `auth.LogFailure(log, "overview request failed", "overview.read", err)` — the helper `staff.go` uses: a database error is logged by SQLSTATE and constraint only | `log.Error(…, "error", err)` with the raw error: a failed `CHECK` or a driver message can carry a row, and this route reads orders |
| V8 | A cap on `running_low`, or a total beside it | **No** (ERD O10). The list is every shoe running low; the page's figure is its length | A cap plus a `running_low_total` field: a contract change and a frontend change for a list a few dozen long |
| V9 | May the page's four figures be given as separate endpoints so that a slow one does not hold the others? | **No.** One request is the story's point (ADMIN-021 item 3, SC-25): the "Today" sentence's counts must equal the figures above it, and five statements on one snapshot are what guarantee it | `GET /v1/admin/overview/month`, `…/low`: five round trips for a page that is read a few times a day, and figures taken at five instants |

### Conflicts with merged decisions

None. I read `services/_base.md` §1.1, §2.1–§2.4, §2.9, §3.25, §3.26–§3.27 (the guard and error conventions), `services/staff-accounts-managed-by-the-owner.md` (the shape of this file and `internal/admin`'s helpers), `services/manage-shoes-lines-categories-reviews-and-orders.md` §1 (the parallel use of `internal/admin` and of the `server.go` region) and the merged `internal/admin`, `internal/server`, `internal/orders` and `internal/auth` code. **One correction, not a conflict:** the example in `_base.md` §3.25 names a seeded shoe wrongly (`handball-spezial-earth-strata`); §4 above gives the seed's id. Two statements are **refinements of silence** (V1: what an unknown parameter does; the 5-second deadline and the abandoned-request rule of §3), and neither changes a status, field or code.

## 11. Migration plan

| Step | Forward | Backward | Safe on a populated table |
|---|---|---|---|
| This story | **No migration file.** | Revert the backend PR (and the frontend swap with it): the endpoint is gone and the page reads its mock again, which exists only while PR #76 is unmerged — after both merge, a revert of the backend PR alone leaves `/admin` on its error view, **so the two are reverted together** | Not applicable: nothing is added to any migration; nothing in any database changes at deploy time, so the deploy needs no order between the backend and the frontend beyond "the backend first": a frontend that asks before the route exists gets the catch-all `404`, which the page shows as the error view and "Try again" recovers from |

## 12. Hand-over

| Who | What they must know from this design |
|---|---|
| Dev, backend step | **No migration.** `internal/admin/overview.go` and `overview_store.go` with the five statements of §5 as given; one line in `server.go`; the swap of §9 (the BE branch is cut from the UI branch and PR #76 merges first); the tests of §8. Reuse `reply`, `refuse`, `internalMessage` and `auth.LogFailure`; define nothing whose name does not start with `overview`. Every list is made non-nil. **Do not add an index, a cache or a setting** |
| Test lead | The SRS literals hold at the fixed clock in the backend test over the migrated sample set (SC-4 to SC-8, SC-11). On the running shop the clock is real: compare the page with the endpoint and the endpoint with the sums of ERD §7, never with a literal (story §8). Running low on the seeded shop is whatever the placeholder counts give, **not** "two shoes" — that is the substituted answer of the page cases. SC-14's cancel half is a SQL `UPDATE` until "Dealer handles orders" (item 17) merges |
| "Dealer handles orders" (item 17) and "Manage shoes…" (item 10) | They write what this reads: an order's status (a cancelled order leaves every figure on the next read and stays among the five latest) and a shoe's pairs (through `internal/stock`, the one definition behind `product_stock`). Neither needs to know the overview exists. `GET /v1/orders` and the menu's `limit=1` read are theirs, not this route's |
| PM | Two product questions the design does not decide and the story already carries: whether a latest-orders row's time should be pinned to Vietnam time (the chart's day labels already are), and whether "pairs sold by line" should be this month's rather than all time (ERD O5) |
