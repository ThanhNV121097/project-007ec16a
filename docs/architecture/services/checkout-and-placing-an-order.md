# Service contracts — Checkout and placing an order (plan item 5, "S5")

Module `orders` · Requirements ORD-002 to ORD-006, ORD-008 (privacy half), ORD-009, ORD-010 · Story `docs/orders/stories/checkout-and-placing-an-order.md` · UI PR #32 · Tables: `docs/architecture/erd/checkout-and-placing-an-order.md`.

This file defines **the endpoints this story ships, the payment seam, and the three changes it makes to code that already exists**, and nothing else. It reuses the project's error envelope and closed error catalog (`services/_base.md` §2.3), its idempotency table (§2.6) and its validation boundary (§2.5); it adds **no code to the catalog**, no second list format and no second error shape. Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/orders` reaches Go as `/v1/orders`. The Order page's own endpoint `POST /v1/orders/{id}/payment` is the next item's (`_base.md` §3.13).

## 1. Endpoints

| # | Method and path | Auth | Traces to |
|---|---|---|---|
| 3.10 | `POST /v1/orders` | none (a customer session links the order — accounts story) | ORD-002 to ORD-006, ORD-009, ORD-010 |
| 3.11 | `GET /v1/orders/{id}` | the order's `X-Order-Token` | ORD-008 AC-4 to AC-6, AC-8, AC-10 |
| 3.14 | `GET /v1/orders` | dealer — answers `401` to everyone until the sign-in story | ORD-008 AC-1, AC-9 |
| 3.5 | `GET /v1/payment-options` | none | ORD-006 AC-1, AC-3, AC-5 |

Routes are literal patterns, one per line above. **Changes to code that already exists** (§5): the `data` reader stops serving `orders`; its `products` expressions move to the view `product_stock`; the `stock` package gains `Take`.

## 2. The mock, read and judged

The UI PR adds `src/lib/mock/checkout-and-placing-an-order.ts`, imported only by `src/lib/orders.ts`. It is unreviewed input; its shape is compared with the contract below. **Where the contract (`_base.md` §3.5, §3.10) already decided a shape it wins**; the mock matches it everywhere that matters, and the differences cost the frontend nothing.

| Mock | Contract (this file, `_base.md`) | Judgment | Frontend change at the swap |
|---|---|---|---|
| Request `{items:[{product,size,qty}], delivery:"standard", payment:"vnpay", customer, phone, address, city, note}`, no amount | Same, field for field. The name is **`customer`** (the stored column's name), so `errors.customer` names it | Sound | None |
| Answer `{order:{id,number,placed_at,status,paid,total,delivery_fee,delivery,payment}, token, payment_url:""}` | `order` is the full public view of `_base.md` §2.8 (also `items`, `carrier`, `tracking`, `eta`, `customer`, `city`, and the first pair's `product`, `size`, `qty`); `token`; `payment_url` | The client's `PlacedOrder` type names six fields and ignores the rest. `placed_at` becomes RFC 3339 with `+07:00`; `formatDateTime` already builds a `Date` from it | None |
| Token: 48 hex characters | 32 base64url characters | The page treats it as an opaque string and keeps it in `doi:placed` | None |
| `409 {error:"stock_conflict", errors:{"items.<i>": sentence}}`; the "only 2 left" count is a constant (`LOW_LEFT`) | Same keys; the number in "only <n> left" is the real remaining count | The sentences are the shop's data, shown as sent | None |
| Repeated key → `200`, first order, new token; same key with other phone digits → `409 conflict` | Same (§3.1 step 0). The real replay also answers `Idempotent-Replayed: true` | Sound | None |
| `422` with `errors` sentences | Same shape; the sentences are in §3.1 | Sound | None |
| `503 payment_unavailable`, `is_available:false` through `localStorage["doi:mock"]` | **Not produced by this story**: `is_available` is `true` and placing never answers 503 until the VNPay story adds the key check (§4) | The states stay drawn; only the VNPay story can make the shop trigger them | None. The knob goes with the mock |
| `getPaymentOptions()` → `{items:[vnpay],total:1,page:1,limit:100}` | Same (§3.4) | Sound | None |
| Prices read with a `pageSize: 500` products read | A page is capped at 100 and `id=a,b` (any-of) belongs to the search story | Cannot survive the swap | `Checkout` reads **each distinct shoe in the bag with `GET /v1/products/{id}`** (a bag holds a few); how a `404` line is drawn is Dev's, the shop's answer to placing is the authority |
| Offline detected with `navigator.onLine` before sending | — | Sound, a browser state | None |

## 3. Contracts

### 3.1 `POST /v1/orders`

**Purpose** — place an order: the shop re-reads every price, takes the pairs off the shelf, saves the order and answers with its token and the address to pay at. **Auth** — none. `Content-Type: application/json`. **Headers** — `Idempotency-Key` (required).

**Request body** — exactly `_base.md` §3.10:

```json
{
  "items": [ { "product": "samba-og-w-core-black", "size": "38", "qty": 2 } ],
  "delivery": "standard",
  "payment": "vnpay",
  "customer": "Lan Nguyen",
  "phone": "0912 345 678",
  "address": "12 Example Street, Ward 1",
  "city": "Da Nang",
  "note": ""
}
```

`items[].price`, `items[].unit_price`, `delivery_fee` and `total` are the only tolerated extras: decoded into discarded fields and ignored (ORD-003 AC-6, AC-9). Every other undefined field, a wrong type (`"qty": "2"`, `"qty": 2.5`) or a body that is not an object is `400 bad_request` (§2.5). Text is trimmed before it is checked and stored; limits count characters (runes).

**Order of checks** — body cap 64 KiB and JSON (`400`, `413`, `415`) → the header (`400`) → **every field at once** (`422`, one `errors` entry per fault; a request with a field fault never reaches the database except for the one read below) → the keys of the chosen provider (`503`, §4: not produced yet) → the transaction (`409`, `201`, `200`).

| Field | Rule | `errors.<name>` sentence |
|---|---|---|
| `Idempotency-Key` | present, `^[A-Za-z0-9._-]{8,100}$` | `400`, `errors."Idempotency-Key"`: "Send an Idempotency-Key header of 8 to 100 letters, digits, dots, dashes or underscores." |
| `items` | 1 to 50 lines | `items`: "Add at least one pair." / "An order can hold at most 50 pairs." |
| `items[i].product`, `.size` | non-empty; at most 100 and 20 characters | `items.<i>`: "Choose a shoe and a size." |
| `items[i].qty` | integer 1 to 1,000 | `items.<i>`: "Quantity must be from 1 to 1,000." |
| a repeated (`product`, `size`) | the second and later lines | `items.<i>`: "This pair is listed twice." |
| a size the shoe does not list | only when the shoe **exists and is `live`**; decided by one read of the named shoes' sizes before the transaction | `items.<i>`: "`<shoe name> · EU <size>` — not a size this shoe comes in." |
| `delivery` | the string `standard` | `delivery`: "Delivery must be the standard option." (`express`, any other id, a missing value) |
| `payment` | the id of a way to pay **offered** (§3.4): `vnpay`. `cod`, `bank`, `momo` and unknown ids are refused whatever the settings | `payment`: "That way to pay is not offered." |
| `customer` | 1 to 100 characters | "Enter your name." / "Name can have at most 100 characters." |
| `phone` | at most 20 characters, and at least 9 **digits**: the ASCII digits it contains (spaces, `+`, `-`, brackets and any other character are simply not counted) | "Phone can have at most 20 characters." / "Check the phone number — we text you when the pair is on its way." |
| `address` | 1 to 200 characters | "Enter the street address." / "Street address can have at most 200 characters." |
| `city` | 1 to 100 characters | "Enter the city or province." / "City or province can have at most 100 characters." |
| `note` | at most 500 characters; default `""` | "Note can have at most 500 characters." |

A fault in a limit names the limit (ORD-002 item 3). The top-level `message` is "Check the details of your order." A shoe that does not exist or is not `live` is **not** a `422`: it is the `409` below ("no longer in the shop"), whatever size the line names.

**Success** — `201 Created`, `Location: /v1/orders/doi-24818`, `Cache-Control: no-store`:

```json
{
  "order": {
    "id": "doi-24818", "number": "DOI-24818", "placed_at": "2026-11-25T10:00:00+07:00",
    "status": "placed", "paid": false, "total": 6580000, "delivery_fee": 0,
    "delivery": "standard", "payment": "vnpay", "carrier": "", "tracking": "", "eta": "2026-12-01",
    "customer": "Lan Nguyen", "city": "Da Nang",
    "product": "samba-og-w-core-black", "size": "38", "qty": 2,
    "items": [ { "product": "samba-og-w-core-black", "name": "Samba OG Black Crystal", "colourway": "Core Black / Core Black / Cloud White",
                 "image": "/images/ref/samba-og-w-core-black.jpg", "size": "38", "qty": 2, "unit_price": 3290000 } ]
  },
  "token": "q3J0n8Zk1x…32 URL-safe characters",
  "payment_url": ""
}
```

`order` is the public view of `_base.md` §2.8 and **never** holds `phone`, `email`, `address` or `note`: the statement that builds it does not select them. `token` is in this answer and in a replay's only. `payment_url` is `""` whenever no provider gave an address (§4).

**What the shop computes** — `unit_price` = the shoe's price read in the transaction; `total` = Σ `unit_price × qty` (computed in 64-bit integers; an overflow is `422` `items`: "That order is too large."); `delivery_fee` = the constant 0 (never read from `delivery_options`); `status` `placed`; `paid` `false`; `payment` `vnpay`; `delivery` `standard`; `account_id` empty. The expected date:

1. `now` is the Go clock read **after** the stock rows are locked (so the number order and the time order agree), in the fixed offset UTC+7 (`time.FixedZone`, no tzdata needed in the image; Vietnam has no daylight saving). It is also `placed_at`.
2. `days` = the **largest whole number** in `shop_terms.standard_days` ("2–4" → 4); `cutoff` = `shop_terms.dispatch_cutoff` as `HH:MM` (15:00). Both are read in the transaction. A text with no number is `500 internal` (migration 5 asserts it cannot be).
3. `base` = the date of `now`; if it is a Saturday or Sunday **or** `now` is at or after `cutoff`, `base` = the next weekday after it.
4. `eta` = `base` plus `days` weekdays (each step moves to the next Monday-to-Friday day). Vietnamese holidays are not skipped.

Monday 10:00 → that Friday; Monday 16:00 → base Tuesday → the next Monday; Saturday → base Monday → the Friday of the next week; Friday 15:00 → base Monday → next Friday (SC-17). The function is pure and takes the instant as an argument; the service takes a clock so a test can set it.

**The transaction** — one, `READ COMMITTED`, context limit 5 s, `SET LOCAL lock_timeout = '3s'` (a stuck lock is an `internal` error the customer retries safely, never a hang). **Lock order — every later piece of code that locks several of these takes them in this order**: advisory key lock → `products` rows (by id) → `product_sizes` rows (by `(product_id, size)`) → `counters` → the inserts. Orders therefore never deadlock with one another.

0. `SELECT pg_advisory_xact_lock(hashtextextended($key, 0))`, then `SELECT id, phone FROM orders WHERE idempotency_key = $key`. The lock makes two tabs with one key run one after the other; a hash collision between two different keys only makes them wait, never merges them (the read is by the exact key). **Found:** if the stored phone's digits differ from the request's → rollback, `409 conflict` ("That order key was already used for a different order.") and nothing else is said; if they match → **the replay**: insert a new `order_tokens` row, read the order (public view, with `items`) in the same transaction, commit, then ask the provider for a fresh address if the order is unpaid and not cancelled → `200`, `Idempotent-Replayed: true`, the same `order`, the new `token`, `payment_url`. No pair leaves the shelf again; the items of the repeated request are **not** compared (the key and the phone identify the submission; the browser makes a new key when the bag or a detail changes). **Not found:** continue.
1. `SELECT id, name, price, status FROM products WHERE id = ANY($ids) ORDER BY id FOR SHARE` — the price and `live` status are held still until commit; a dealer's edit of them waits a few milliseconds.
2. `SELECT product_id, size, pairs FROM product_sizes WHERE (product_id, size) IN (SELECT * FROM unnest($products::text[], $sizes::text[])) ORDER BY product_id, size FOR UPDATE`. Under `READ COMMITTED` a waiter re-reads the row after the lock is released, so it sees the pairs the earlier order left.
3. Evaluate **every** line, in the order sent: the shoe is missing or not `live`, **or its size row is gone** (staff took the size off since the pre-read) → `<name or id> · EU <size> — no longer in the shop.`; else `pairs = 0` → `… — sold out in your size.`; else `pairs < qty` → `… — only <pairs> left in your size.`; else fine. The name is the shoe's `name` (the shoe's id stands in for one that does not exist). If any line fails: rollback and `409 stock_conflict` naming **all** failing lines as `errors."items.<i>"`; nothing was taken for the lines that were fine, and the shelf is exactly as before.
4. Take the pairs: `UPDATE product_sizes SET pairs = pairs - $q WHERE product_id = $p AND size = $s AND pairs >= $q`, one statement per line in the locked order (or one `UPDATE … FROM unnest(…)`); anything other than one row updated is `internal` (it cannot happen under the lock; the `CHECK pairs >= 0` is the last floor). This is `stock.Take(ctx, tx, lines)`.
5. `UPDATE counters SET value = value + 1 WHERE name = 'order_number' RETURNING value` → `DOI-<value>` / `doi-<value>`.
6. `INSERT INTO orders` (all columns including `idempotency_key`, `placed_at`, `eta`, `total`, `delivery_fee` 0, `is_sample` false), `INSERT INTO order_items` (one statement; `line_no` 1, 2, … in the order sent, each `unit_price`), `INSERT INTO order_tokens` (SHA-256 of a fresh token: 24 bytes from `crypto/rand`, base64url, no padding). Commit.
7. After the commit and **outside** the transaction (§4): the chosen provider is asked for an address (3 s limit). An error or a timeout is logged and leaves `payment_url` empty; the order stays saved and unpaid.

A unique violation on `uq_orders_idempotency_key` or `uq_orders_number` cannot happen under the advisory lock and the counter; if one does it is `internal` and logged, and the customer's retry with the same key replays the order.

**Errors**

| Code | HTTP | Trigger |
|---|---|---|
| `bad_request` | 400 | Invalid JSON; a wrong type; an undefined field; a missing or malformed `Idempotency-Key` (`errors."Idempotency-Key"`) |
| `payload_too_large` | 413 | Body over 64 KiB |
| `unsupported_media_type` | 415 | Not `application/json` |
| `invalid` | 422 | Any field rule of the table above; `errors` names every fault. No order, no pair taken |
| `stock_conflict` | 409 | Any pair sold out, short, or no longer in the shop; `errors."items.<i>"` for **each**. No order, the shelf and the counter unchanged. The key is not remembered: the same key sent again is evaluated afresh |
| `conflict` | 409 | The key was used by an order with another phone |
| `internal` | 500 | The database cannot be reached, a statement fails or times out, a lock times out. Generic body; nothing saved, nothing taken. The cause is logged with the request id (never the key, the token, the phone or the address) |
| `payment_unavailable` | 503 | **Reserved, not produced by this story.** The VNPay story adds it: with no keys the chosen provider is unavailable and no order is made (§4) |

**Notes** — 50 simultaneous orders for one size holding 10 pairs sell exactly 10; the other 40 are refused with `stock_conflict`, no count falls below 0 (the locks queue them; the floor `CHECK` is the backstop). Placement is **not** rate-limited (project open question 2); a hoarding order is undone by the 30-minute expiry of the VNPay story. The access log records the route and the status, never the headers or the body.

### 3.2 `GET /v1/orders/{id}`

**Purpose** — read one order. **Auth** — the header `X-Order-Token: <token>` that opens **this** order; never in an address. In this story the token is the only credential: the dealer and linked-customer paths of `_base.md` §3.11 are added by the sign-in and accounts stories through one seam, `orders.Read(ctx, id, access)`, where `access` is a struct that today holds only `Token` and later gains `Dealer` and `AccountID`. The handler does not import `auth`.

**Checks, in order** — (1) `{id}` must match `^[a-z0-9][a-z0-9._-]*$` and be at most 100 characters, else `404`; (2) the token must be present, at most 128 characters and made of `[A-Za-z0-9_-]`, else it counts as absent: `404` with **no query made**; (3) one read-only snapshot (one statement with a `json_agg` of the pairs, or a `REPEATABLE READ` read-only transaction) so the order and its pairs agree:

```sql
SELECT … FROM orders o
WHERE o.id = $1
  AND EXISTS (SELECT 1 FROM order_tokens t WHERE t.order_id = o.id AND t.token_hash = sha256($2))
```

then the pairs: `order_items JOIN products` by `line_no`, giving each pair's `name`, `colourway` and `image` **live** (a shoe set to draft since still names its pair). The statement lists its columns and does **not** select `phone`, `email`, `address`, `note` or `is_sample`: they cannot be returned by a bug in code that runs after it (SC-36). "No token", "wrong token", "another order's token", "no such id", "malformed id" and every one of the 14 seeded orders (no token exists for them until the track story) run the same statement shape and answer the same body.

**Success** `200`, `Cache-Control: no-store`: the public view of `_base.md` §2.8 with `items`, as in §3.1's example `order`. `eta` is `""` when the order has none.

| Code | HTTP | Trigger |
|---|---|---|
| `not_found` | 404 | Any of the six cases above. Body `{"error":"not_found","message":"Not found."}`, byte for byte the same for all (SC-36) |
| `internal` | 500 | Database failure |

**Notes** — the token never appears in a log line, an address, a response header or any page text (SC-37): the access log records the route (`/v1/orders/doi-24818`) and the status only, never headers. A lookup lock (track story) does not affect this endpoint.

### 3.3 `GET /v1/orders`

**Purpose** — the dealer's list (`_base.md` §3.14). **In this story** the route is mounted by the `orders` package and answers **`401 unauthenticated`** ("Sign in to see orders.") to **every** request, with any query string — it is answered **before** the query string is read, so `?phone=0912…` is `401`, never `400` and never a list (SC-35). No dealer session can exist before the sign-in story, and the list's body (the dealer view, paging, filters) is the "Manage … orders" story's. The route is a literal pattern, so no generic reader can answer it. The sign-in story's `auth.Require(admin, staff)` replaces the unconditional `401` there (and `403` for a customer); this story's code does not import `auth`.

| Code | HTTP | Trigger |
|---|---|---|
| `unauthenticated` | 401 | Always (this story) |

### 3.4 `GET /v1/payment-options`

**Purpose** — the ways to pay the shop can complete; Checkout step 03 reads it instead of the `payments` collection. **Auth** — none; any parameter is ignored. **Traces to** — ORD-006 AC-1, AC-3, AC-5.

**Success** `200`, `Cache-Control: no-store`:

```json
{ "items": [ { "id": "vnpay", "name": "VNPay — ATM, Visa, Mastercard", "note": "Pay by card or banking app on VNPay's secure page.",
               "icon": "card", "online": true, "is_available": true } ],
  "total": 1, "page": 1, "limit": 100 }
```

**Rules** — what is **offered** is code, never data: the ordered list of ids in the `orders` package, today `["vnpay"]`. The rows come from `SELECT id, name, note, icon, online FROM payment_methods WHERE id = ANY($1) ORDER BY seq`, so `cod`, `bank` and `momo` are never listed although `GET /v1/payments` still returns all four rows (SC-30). `is_available` is the registered provider's `Available()`; **a way to pay with no provider registered is reported available** (the story's rule: `true` until the VNPay story adds the key check; no VNPay provider ships here). Errors: `internal` 500.

## 4. The payment seam

Placing goes through one interface that each way to pay plugs into (ORD-006 item 1). The **consumer** (`orders`) declares it, so a provider package satisfies it without `orders` importing it — `payments` already imports `orders` (`MarkPaid`, `Cancel`) and the reverse would be a cycle:

```go
type Provider interface {
    // Available: the shop can take a payment through this provider now (its keys are set).
    Available() bool
    // PaymentURL returns the address the customer pays at. An error means "none could be given".
    PaymentURL(ctx context.Context, o Payable, c Client) (string, error)
}
type Payable struct { ID, Number string; Total int64; PlacedAt time.Time } // read from the saved order, never from the browser
type Client  struct { IP, Origin string }                                  // right-most X-Forwarded-For; the public origin
// orders.Options{ Providers map[string]Provider /* key = payment_methods.id */, Now func() time.Time }
```

| Case | Result |
|---|---|
| No provider registered for the order's way to pay (this story: `vnpay` has none) | The order is saved unpaid; `payment_url` is `""` |
| A registered provider returns `https://pay.example.test/x` | Exactly that string is `payment_url` (SC-31) |
| A registered provider returns an error or takes longer than 3 s | Logged with the order's id; `payment_url` is `""`; the order stays saved and unpaid |
| Every case | `paid` is `false`: nothing but a provider's own confirmation will ever mark an order paid |

The next item's `POST /v1/orders/{id}/payment` calls the **same** `PaymentURL` for an unpaid order; with no provider it answers `409` "Online payment is not available yet." The VNPay story registers the provider, makes `Available()` the key check, adds `503 payment_unavailable` to placing **between the field checks and the transaction** (§3.1 "order of checks"), and writes `payment_attempts` inside `PaymentURL`. Nothing in `Place` changes for it. Tests use two fakes: one returning an address, one returning an error (SC-31).

## 5. Packages, and the changes to code that already exists

| Package | Change |
|---|---|
| `internal/orders` (new) | `Register(mux, pool, log, Options)` mounts the four routes of §1. Functions `Place`, `Read`, the pure expected-date function, token and key helpers, the provider seam. The only writer of `orders`, `order_items`, `order_tokens`, `counters` |
| `internal/stock` (new, no routes) | The only writer of `product_sizes`. Two functions here: `Lock(ctx, tx, keys)` (§3.1 step 2: the rows `FOR UPDATE` in `(product_id, size)` order, returning the pairs held) and `Take(ctx, tx, lines)` (step 4). `Return`, `Set` and `SetOne` arrive with the cancel and staff stories |
| `internal/data` (S3) | (1) **Remove the `orders` collection** from the registry and from `server.New`'s mounts. Go's mux panics on a duplicate pattern, so the removal and `orders.Register` are one PR. The drift test (`collections_test.go`) keeps reading `schema.json`, which still lists `orders`: it learns one explicit `servedElsewhere = {"orders"}` list and a route-table test asserts `GET /v1/orders` answers `401` and `GET /v1/orders/doi-24817` answers `404`; the per-collection privacy test in `store_test.go` loses its `orders` loop. (2) `products`: `FROM products t JOIN product_stock ps ON ps.product_id = t.id`; the four list fields' expressions become `ps.sizes`, `ps.sizes_low`, `ps.sizes_out`, `ps.stock`; the list kind's filter is `$n = ANY(ps.sizes_low)` and so on; **names, kinds and the 29 public fields do not change**, so `GET /v1/products` is byte-for-byte the same shape (SC-29). (3) Every other statement that names `products.sizes`, `sizes_low`, `sizes_out` or `stock` is rewritten in the same PR: `grep -rn "sizes_low\|sizes_out" internal/` must find only the registry's view expressions. If the search story's backend has merged, its `size=a,b` clause becomes `EXISTS (SELECT 1 FROM product_sizes ps WHERE ps.product_id = t.id AND ps.size = ANY($n) AND ps.pairs > 0)` here; if it merges after, it is built on `product_sizes` directly |
| `internal/server` | `orders.Register(mux, pool, log, orders.Options{})` after `data.Register`; no provider is passed |
| `internal/httpx`, `internal/config` | Reused as they are. No new setting, no new secret, no `.env.example` change: nothing here needs a credential |

## 6. The frontend at the swap (the backend PR's frontend half)

| Change | Where |
|---|---|
| Replace the two mock imports (`getPaymentOptions`, `postOrder`) with `src/lib/api.ts` calls; map an `ApiError` to the `Wire` shape (status and `{error, message, errors}`); send `Idempotency-Key` as a request header. If `api.ts` has no way to add a header, an optional `headers` argument is the one permitted edit to it | `src/lib/orders.ts` |
| Delete `src/lib/mock/checkout-and-placing-an-order.ts` (its bridge into `mock/orders.ts` goes with it). `src/lib/mock/orders.ts` **stays**: `Order.tsx` and `Track.tsx` still import it | `src/lib/mock/` |
| Read each distinct shoe in the bag with `GET /v1/products/{id}` instead of one `pageSize: 500` read | `src/pages/Checkout.tsx` |
| An options read that **finished** without `vnpay` (empty or failed) is drawn as the unavailable state (alert, button disabled) rather than an empty step (the UI review's first "should fix") | `src/pages/Checkout.tsx`, `CheckoutSummary.tsx` |
| Follow `payment_url` only when it is an `https:` address (UI review nit) | `src/pages/Checkout.tsx` |
| No new content key; `src/content.json`, `theme.css`, `index.css`, `App.tsx`, `Order.tsx`, `Track.tsx`, `Product.tsx` are not touched | — |

## 7. Decisions, and conflicts with merged ones

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| S1 | The project text says `vnpay`'s `is_available` is true only with both VNPay keys set | **For this story only: `true`** while no provider is registered; the key check is the VNPay story's (the story fixes this). The contract's rule is untouched and becomes true again the day the provider registers | Reading `VNPAY_TMN_CODE` here: the keys are not requested until the VNPay story, so Checkout would be disabled in every environment |
| S2 | Where the provider interface lives | In `orders`, declared by the consumer (§4) | In `payments`: `orders.Place` would import `payments`, which imports `orders` — a cycle |
| S3 | Two tabs, one key (the project text: "the second waits on the unique index and gets the replay") | An advisory lock on the key first, then the replay read (§3.1 step 0); the unique index remains the backstop. Cost: one extra statement per order | The index alone: the second order would take stock and the counter before it reached the index, so with the last pair gone it would answer `409 stock_conflict` to a retry of an order that was saved. Same outcome the text promises, a mechanism that delivers it |
| S4 | Row locks during placement (the project text: `FOR UPDATE` on the shoes) | `products` `FOR SHARE`, `product_sizes` `FOR UPDATE` (§3.1) | `FOR UPDATE` on `products`: every order of one shoe queues behind every other for no gain |
| S5 | "A size the shoe does not list is a `422`, decided before the transaction" — the database must be read to know | One read of the named shoes' sizes alongside the field checks. It reports `422` only for a shoe that exists and is `live`; a missing or draft shoe is `409` (SC-25), and a size that vanishes after the read is `409` "no longer in the shop" inside the transaction | Deciding everything inside the transaction: a request with a field fault would open a transaction and take locks to say something a read could say |
| S6 | The orders list until a session exists | `401` unconditionally, without importing `auth`; the sign-in story swaps one line | Importing `auth` now: the package is not on `main`; two stories would edit one handler |
| S7 | The name field's key in `errors` | **`customer`**, the wire name the mock and the column use; the story's "name" is the page's word | Renaming the wire field `name`: breaks the merged contract and the stored column for a label |
| S8 | What counts as a phone digit | The ASCII digits the string contains; other characters are not counted and not refused (`0912.345.678` has 10 digits and is accepted). At most 20 characters | Refusing a phone that holds anything but digits and `+ - ( )`: stricter than the SRS asks and refuses what Vietnamese users type |
| S9 | The expected date's source and clock | `shop_terms.standard_days` (largest number) and `dispatch_cutoff`, at the fixed offset UTC+7; the mock's hard-coded 4 dies with it (§3.1) | `time.LoadLocation("Asia/Ho_Chi_Minh")`: needs tzdata in a minimal image, for a zone with no daylight saving |
| S10 | A replay with different items but the same key and phone | Returns the first order; items are not compared | Comparing items: a second code path and a new refusal the story does not ask for; the browser makes a new key whenever the bag or a detail changes (ORD-010 item 3) |
| S11 | Key refused for stock | A `409 stock_conflict` stores nothing, so the key stays free | Remembering refusals: a customer who restocks the bag with the same key would see a stale refusal |
| S12 | Checkout's price read | One `GET /v1/products/{id}` per distinct shoe (§2) | `id=a,b` any-of: belongs to the search story and is *ignored* by the reader until it lands |

## 8. Ordering with the open stories, and hand-over

| Matter | Rule |
|---|---|
| PR #24 (free delivery) → PR #39 (site data) → this story's backend | In that order. The migrations are stamped later than S3's `…seed_collections`; the checkout UI branch is stacked on #24 and is re-diffed against `main` once #24 merges |
| Search story (PR #38 is its UI) | If its backend merges first, this PR rewrites its `size=` clause (§5 `data` item 3); if second, it uses `product_sizes` and `product_stock` and needs no change |
| "Order page and Pay now" | Reads `GET /v1/orders/{id}` with the token this story stores; calls the same `PaymentURL` seam; before the VNPay story its `POST /v1/orders/{id}/payment` answers `409` "Online payment is not available yet." |
| "Track an order by number and phone" | Inserts `order_tokens` rows and reads the same `X-Order-Token` rule; reuses `rate_limits` |
| "Online payment with VNPay" | §4 |
| "Dealer sign-in", "Customer accounts" | Add `Dealer` and `AccountID` to the `access` struct of `orders.Read`, the session path to `GET /v1/orders`, and `orders.account_id` (migrations 11, 12 of the project ERD) |
| "Manage … orders" and the cancel stories | Add `stock.Return`; lock `orders` then `product_sizes`, never the reverse of §3.1's lock order |
| Test data | The running-API verification uses seeded sizes: 3 pairs in an ordinary size, 1 pair in each seeded last-pairs size, 0 in each sold-out size. Counts of 2 and 10 (SC-23, SC-24's 50-order case) are set by a backend test with plain SQL on a shoe the test creates; no staff endpoint exists yet. A test that inserts an order by hand moves `counters` too |

## 9. Where each criterion is enforced

| Criteria | Enforced by |
|---|---|
| SC-9 to SC-12 | The field table of §3.1 (page rules mirror it; the shop repeats them) |
| SC-13 to SC-18, SC-20 | §3.1 transaction steps 5–6, "What the shop computes", the success body |
| SC-17 | The pure expected-date function (a table test over the clock: Mon 10:00, Mon 16:00, Sat, Fri 15:00) |
| SC-19 | `erd/checkout-and-placing-an-order.md` §7 |
| SC-21, SC-22 | `internal` row of §3.1; the statement count and key-only access (ERD §3) |
| SC-23 to SC-29 | §3.1 steps 1–4, `product_stock`, the `data` registry change |
| SC-30, SC-31 | §3.4 and §4 |
| SC-33, SC-34 | `Idempotency-Key` rows, step 0 and the `conflict` row; the page's key keeper (UI PR) |
| SC-35 to SC-37 | §3.2, §3.3 and the log rule |
| SC-1 to SC-8, SC-26, SC-32 | The UI PR and §6; SC-8 is proven with a stubbed response here |
