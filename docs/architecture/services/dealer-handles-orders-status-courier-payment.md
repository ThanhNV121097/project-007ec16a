# Service contracts — Dealer handles orders: status, courier, payment (plan item 17, "S16")

Module `admin` (handlers) and `orders` (the rules and the SQL) · Requirements ADMIN-016, ADMIN-004 (AC-6, AC-9), ADMIN-011 (AC-12, AC-14, AC-15, AC-20), ADMIN-014 (AC-7, AC-8) · Story `docs/admin/stories/dealer-handles-orders-status-courier-payment.md` · UI PR #77 · Tables: `docs/architecture/erd/dealer-handles-orders-status-courier-payment.md` (**none changed**).

This file **extends** `services/_base.md` §3.11, §3.14, §3.15 and §3.16 and the order of §2.8. Their routes, auth, field names and error codes **stand**. It adds the order in which a request is refused, every field rule and sentence, the dealer's two reads, what a refused body does (nothing), the Go surface, the tests the backend PR ships, and the frontend's swap from its mock. It adds **no error code** to the catalog (§2.3), no second error or list shape, no table, no setting and no secret. Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/orders/doi-24818` reaches Go as `/v1/orders/doi-24818`.

## 1. Endpoints

| # | Method and path | Auth | What this story does |
|---|---|---|---|
| 3.11 | `GET /v1/orders/{id}` | token, linked customer, **dealer** | Adds the dealer branch (§3.1). The public statement, the token path and the customer path are not edited |
| 3.14 | `GET /v1/orders` | dealer | Serves `phone`, `email`, `address`, `note`, `transaction_no` and `product_name`; the default order is newest first (§3.2) |
| 3.15 | `PUT /v1/orders/{id}` | dealer | New (§3.3) |
| 3.16 | `DELETE /v1/orders/{id}` | dealer | New (§3.4) |

"Dealer" is `auth.Require(auth.Admin, auth.Staff)`: no session `401 unauthenticated`, a customer `403 forbidden`, **both before the body, the id or the database are read**, so a visitor's call for an id that does not exist is `401`, never `404` (SC-1). An admin and a staff session get identical answers everywhere below.

**Mounting.** `admin.RegisterOrders(mux, db, log)` in `internal/admin/orders.go` mounts two **literal** patterns, `PUT /v1/orders/{id}` and `DELETE /v1/orders/{id}`, each wrapped in `auth.Require(auth.Admin, auth.Staff)`. `server.New` gains **one line**, `admin.RegisterOrders(mux, pool, log)`, after `admin.RegisterRecords` (item 10 adds that line too; whichever PR merges second keeps both). `PUT /v1/orders`, `PATCH`, and `POST /v1/orders/{id}` have no pattern and answer `404 not_found` through the catch-all, for every caller. **On every answer of the two routes** (the success and every refusal the handler writes): `Content-Type: application/json; charset=utf-8` except `204`, `Cache-Control: no-store`, and the `X-Request-Id` the middleware echoes. No new setting, secret or `.env.example` line.

## 2. The mock, read and judged

The UI PR (#77) adds `src/lib/mock/dealer-handles-orders-status-courier-payment.ts`, imported only by `src/lib/dealer-handles-orders-status-courier-payment.ts` through one `transport` constant (the UI review checked that; it did not review the mock's contents). It is unreviewed input: its shape is compared with `_base.md` §2.8, §3.11 and §3.14–§3.16, **which decide it where they speak**.

| Mock / frontend assumption | Contract | Judgment | Frontend change at the swap |
|---|---|---|---|
| `read` answers a `DealerOrder`: `id`, `number`, `placed_at` (RFC 3339, `+07:00`), `status`, `product`, `size`, `qty`, `total`, `delivery_fee`, `payment`, `paid`, `delivery`, `carrier`, `tracking`, `eta` (`""` when none), `customer`, `city`, `phone`, `email`, `address`, `note`, `transaction_no`, `items[]` (`product`, `name`, `colourway`, `image`, `size`, `qty`, `unit_price`) | The dealer view of `_base.md` §2.8 with `items` (§3.1) | **Sound, field for field.** Names, types, units and "`""` when unset" match. `delivery_fee` is the one field the page's type names and never draws | None |
| `update` answers the same `DealerOrder`, **with `items`**; the page merges `{...order, ...answer}` | The `PUT` answers the dealer view read in the same transaction, `items` included (§3.3) | **Sound.** The merge stays as a safety net | None |
| `transport` throws an `ApiError(status, code, message, errors)`; a failure to connect is status `0` | `src/lib/api.ts` already does exactly this (`failure()` reads `error`, `message`, `errors`; `put` and `delete` exist on `main`) | **Sound.** The mock imports the real `ApiError`, so the page's `instanceof` checks hold | `transport` becomes three `api` calls (§5) |
| `404` is "missing" only when `status === 404 && code === "not_found"` | The server's own body for a missing or malformed id is `not_found`; a gateway's 404 reaches `api.ts` as code `internal` | **Sound, and the right guard.** Without the code check a broken proxy rule would tell the dealer every order is gone | None |
| The 422 sentences: `"An order cannot go from <a> to <b>."`, `"An order that is <status> cannot be changed."`, `"This cannot be changed."`, `"Only the payment provider marks this order paid."`, `"Write the date as year-month-day, for example 2026-11-26."`, `"Enter the customer's name."` | §3.3 adopts all six **as written** | **Sound.** The page shows `errors[key]` as sent | None |
| The 422 sentences `"Keep this to 200 characters."` and `"Keep the note to 5,000 characters."`, `"This must be text."` | §3.3 words them `<Label> can have at most 200 characters.` etc.; a value of the wrong type is a **400** (`<Label> must be text.`), not a 422 | **Differs in words, not in shape.** The page never reads a sentence's words and `errors[key]` is still keyed by field | None. The mock's 422 for a non-text value is the one behavioural difference; the page cannot send one |
| The 400 for an unknown key carries no `errors` | §3.3: `400 bad_request` with `errors.<key>` "Not a field of this order." | Sound | None |
| `remove` answers 409 `conflict` with the sentence in `message` and no `errors`; the page reads `e.message` for 409 and 404 | §3.4 does exactly this (the envelope carries `errors` only on the codes of §2.3) | **Sound** | None |
| `expired` is a `401` only | The server answers `401` to no or an ended session; `403` only to a customer session, which never reaches `/admin` (the gate) | Sound | None |
| The list's "shoe" column resolves the first pair's id to a name from the **first 100 shoes** and falls back to the id | The list has only the first pair's `product` id; a shop of hundreds of shoes would show ids for the rest, so SC-25 ("shows the name of the first pair's shoe") would fail beyond shoe 100 | **Not sound at the shop's scale.** The server adds `product_name` to each list item (§3.2, as the overview's `latest_orders` already carries one) | `AdminOrders.tsx` reads `r.product_name` for the column and drops its `useCollection("products", { pageSize: 100 })` read |
| The list's default order is whatever `ListPage` gets (no sort) | `_base.md` §2.4's default is `seq` ascending, oldest first | **The contract's default is wrong for this list** (ADMIN-011 AC-12: newest first), and `ListPage` has no default sort the story may set | None: the server answers newest first for this list only (§3.2) |
| Seeded `transaction_no` on four sample orders (`14123455`…`14123457`) | The 14 sample orders have no payment rows: `transaction_no` is `""` until item 8 makes a payment | Mock-only data | None; `OrderFacts` already draws the line only when the string is non-empty |
| `?mock=slow` / `offline` / `expired` | **No delay, fault or sleep route, header or query parameter is added to the backend** | Mock-only controls | Deleted with the mock; the tests' `replaceState("?mock=…")` cases become `vi.mock("../lib/api")` stubs that throw the real `ApiError` (`status` 0 for offline, 401 for expired, a promise held for two seconds for slow) |

## 3. Contracts

### 3.1 `GET /v1/orders/{id}` — the dealer branch (project §3.11)

**Rule.** Before the token path, `orders.read` asks `auth.FromContext(ctx)`. A session of role `admin` or `staff` is answered **the dealer view, whatever header the request carries** (a dealer's `X-Order-Token` is not looked at). Any other caller — no session, a customer — takes the path that exists today, byte for byte: token → public view; linked customer → public view; anything else → `404`. A dealer branch is **never** added to `authorize` (decision D2): `authorize` returns a `Ref` that can never hold a private column, and that guarantee is what SC-3 stands on.

1. `{id}` is matched against `^[a-z0-9][a-z0-9._-]*$` and at most 100 characters; else `404 not_found`, no query.
2. One statement, `dealerViewSQL`, a constant of its own beside the public `viewSQL` and **not derived from it by string edits**: it lists every column it returns. It is the public view's eighteen keys (`id`, `number`, `placed_at`, `status`, `paid`, `total`, `delivery_fee`, `delivery`, `payment`, `carrier`, `tracking`, `eta`, `customer`, `city`, `product`, `size`, `qty`, `items`) plus `phone`, `email`, `address`, `note` and `transaction_no` (ERD §6). Like the public statement it is one `SELECT json_build_object(…)`, so the order and its pairs agree. It never selects `is_sample`, `idempotency_key`, `account_id`, `source` or `invented`.
3. No row → `404 not_found`, the body `{"error":"not_found","message":"Not found."}` the project writes everywhere.

**Success** `200`, `Cache-Control: no-store`: the dealer view.

```json
{
  "id": "doi-24818", "number": "DOI-24818", "placed_at": "2026-11-26T09:12:00+07:00", "status": "packed",
  "paid": true, "total": 6380000, "delivery_fee": 0, "delivery": "standard", "payment": "vnpay",
  "carrier": "GHN", "tracking": "GHN-PT27XM", "eta": "2026-11-30", "customer": "Lan Nguyen", "city": "Da Nang",
  "product": "samba-og-w-core-black", "size": "38", "qty": 1,
  "items": [
    { "product": "samba-og-w-core-black", "name": "Samba OG Black", "colourway": "Core Black", "image": "/images/ref/samba-og-w-core-black.jpg", "size": "38", "qty": 1, "unit_price": 3090000 },
    { "product": "gazelle-indoor-solar-gold", "name": "Gazelle Indoor Solar Gold", "colourway": "Solar Gold", "image": "/images/ref/gazelle-indoor-solar-gold.jpg", "size": "42", "qty": 1, "unit_price": 3290000 }
  ],
  "phone": "0912 345 678", "email": "", "address": "12 Example Street, Ward 1", "note": "Call before delivering.", "transaction_no": "14123456"
}
```

| Code | HTTP | Trigger |
|---|---|---|
| `not_found` | 404 | The id does not exist or cannot be an id (a dealer is told the truth about a missing order; a visitor keeps the one body for every cause) |
| `internal` | 500 | The statement fails or times out (the 5-second context). Generic body; the cause is logged beside the order id and the request id |

**Notes.** A dealer's browser that opens a customer's order page sends its session cookie and so receives the dealer view in the network answer (the contract's "(1) a dealer session"); the customer pages draw none of the private fields and SC-27 is checked with **no dealer session** (Test: open the customer page in a window with no admin cookie, as item 10's D2 already says for the shop pages). `GET /v1/orders/{id}/payment`-style routes are untouched: `POST /v1/orders/{id}/payment` answers a dealer session without the token `404` (§3.13).

### 3.2 `GET /v1/orders` — the dealer's list (project §3.14)

**Auth, query, answer** — §3.14 and §2.4 **stand**: `auth.Require(auth.Admin, auth.Staff)` before the query string is read; `page`, `limit` (default 12, above 100 served as 100), `q`, `sort`, `status=a,b`, `payment=a,b`, `placed_at_from` / `_to`, `total_from` / `_to`, and the answer `{ "items", "total", "page", "limit" }` with `Cache-Control: no-store`. The grammar is `data`'s one reader; this story writes no second query language. Three things change:

| Change | Rule |
|---|---|
| **The private columns are served** | Each item is the dealer view **without `items`**: `id`, `number`, `placed_at`, `status`, `product`, **`product_name`**, `size`, `qty` (the first pair's), `total`, `payment`, `paid`, `delivery`, `carrier`, `tracking`, `eta`, `customer`, `city`, `phone`, `email`, `address`, `note`, `transaction_no`. `phone`, `email`, `address`, `note` and `transaction_no` are `Private` columns: **selected only by this handler**, never filterable, sortable or searchable. A filter, range or `sort` on one of them is `400 bad_request` naming it, for every caller including a dealer (`?phone=`, `?email=`, `?address=`, `?note=`, `?transaction_no=`; `_base.md` §2.4's list of private fields gains `transaction_no`) |
| **`product_name`** | The first pair's shoe's name, `""` when the order has no pair. A public column: it may be filtered and sorted like `product`. A shoe an order refers to cannot be deleted, so the name is always there |
| **Default order** | With no `sort`, `placed_at DESC, seq DESC` (newest first, ties by position). With a `sort`, the contract's rule stands: the chosen field, then `seq ASC`. The deviation is this collection's alone (`Collection.DefaultOrder`; every other collection keeps `seq ASC`) |

`GET /v1/orders` for the menu's count stays `?status=placed,confirmed&limit=1`, read as `total`. The search `q` still looks in the order number and the customer's name only; a private column is never searched.

| Code | HTTP | Trigger |
|---|---|---|
| `unauthenticated` / `forbidden` | 401 / 403 | No session / a customer, **before** the query string is read |
| `bad_request` | 400 | `page`/`limit`/`q`/`sort` malformed, a range that is not a number or date, a filter or sort on a private column |
| `internal` | 500 | The statement fails or times out |

### 3.3 `PUT /v1/orders/{id}` (project §3.15)

**Purpose** — move an order along, set the courier's details, correct the customer's details, edit the note. **Partial**: only the fields sent change. **All or nothing**: a body with one valid change and one refused field applies neither (`{"status":"confirmed","total":1}` leaves the order `placed`).

**Order in which a request is refused.** The first row that fails answers; rows 7 to 11 collect **every** fault they can see before answering, so the page shows each beside its field in one save.

| # | Check | Answer |
|---|---|---|
| 1 | The route exists | `404 not_found` (catch-all) |
| 2 | A dealer session | `401` / `403` |
| 3 | `Content-Type` is `application/json` (a `charset` is allowed) | `415 unsupported_media_type` |
| 4 | The body is at most **64 KiB** | `413 payload_too_large` |
| 5 | The body is **one JSON object**, nothing after it (an empty body is not one) | `400 bad_request`, `errors.body` "Send one JSON object." |
| 6 | `{id}` matches `^[a-z0-9][a-z0-9._-]*$`, at most 100 characters | `404 not_found`, no query |
| 7 | Every key is one of the **ten editable fields**, one of the **eleven refused names** + `paid`, or `id`; every editable value is a JSON string. Nothing else is read from the database yet | `400 bad_request`, `errors.<key>`: "Not a field of this order." for a key the order does not have (`colour`, `seq`, `is_sample`, `account_id`); "<Label> must be text." for `null`, a number, a boolean, an array or an object. `id` is read and discarded, whatever its value |
| 8 | The refused names present: `number`, `placed_at`, `product`, `size`, `qty`, `items`, `total`, `delivery_fee`, `payment`, `delivery`, `transaction_no`, `paid` — **by name, whatever the value and the JSON type**, even the value the order already has | collected as `422`: `paid` → "Only the payment provider marks this order paid."; every other → "This cannot be changed." |
| 9 | The value rules of every editable field present (table below) | collected as `422` |
| 10 | Open a transaction (`READ COMMITTED`, 5 s context), `SET LOCAL lock_timeout = '3s'`, `orders.LockOrder(id)` | `404 not_found` for no such order (this wins over every `422`) |
| 11 | The **state rules**, judged on the status the order has under the lock (ERD §3): the move, and a delivered or cancelled order's locked fields | collected as `422` |
| 12 | Any fault from 8, 9 or 11 | roll back; `422 invalid`, `errors` naming every field at fault, `message` "Check the details of this order." Nothing in the body is applied |
| 13 | Apply (§4: one `UPDATE`, or the cancel with its pairs), read the dealer view **in the same transaction**, commit | `200` the dealer view of §3.1 |

**The ten editable fields** (the labels are the schema's own, and the page's)

| Field (Label) | Rule | Sentence on a fault |
|---|---|---|
| `status` (Status) | one of `placed`, `confirmed`, `packed`, `shipped`, `delivered`, `cancelled` (the stored words, never "On its way") | not one of the six: "Status must be placed, confirmed, packed, shipped, delivered or cancelled." |
| `carrier` (Carrier), `tracking` (Tracking number) | text, at most **200** characters, `""` clears | "Carrier can have at most 200 characters." · "Tracking number can have at most 200 characters." |
| `eta` (Expected) | `""` (clears; stored as no date) or a real calendar date written `YYYY-MM-DD`, from 1900 to 2999. `26/11/2026`, `2026-13-40`, `2026-2-3`, `tomorrow` are refused | "Write the date as year-month-day, for example 2026-11-26." |
| `customer` (Customer) | required, at most 200 | "Enter the customer's name." · "Customer can have at most 200 characters." |
| `phone` (Phone) | at least **9 digits** (the ASCII digits it contains; the one rule checkout uses), at most 200 characters | "Check the phone number: it needs at least 9 digits." · "Phone can have at most 200 characters." |
| `email` (Email) | `""`, or an address `auth.NormalizeEmail` accepts (at most 200). Stored trimmed, as typed | "Enter a valid email address, or leave it empty." |
| `address` (Address) | required, at most 200 | "Enter the street address." · "Address can have at most 200 characters." |
| `city` (City or province) | required, at most 200 | "Enter the city or province." · "City or province can have at most 200 characters." |
| `note` (Internal note) | at most **5,000** characters (5,000 is accepted, 5,001 is not), `""` clears | "Internal note can have at most 5,000 characters." |

Text is **trimmed** before it is judged and stored; limits count characters (runes), not bytes. A value holding U+0000 is refused with "<Label> holds a character that is not allowed." (PostgreSQL cannot store it). Required means present and not empty after trimming; the table's `CHECK`s (`ck_orders_customer_not_blank`, `phone`, `address`, `city`) are the backstop, never the message.

**The state rules (row 11)** — with `S` the stored status under the lock:

| Body says | `S` is | Result |
|---|---|---|
| `status` = `T` (one of the six) | `delivered` or `cancelled` | **422** `errors.status` "An order cannot go from S to T." — **even when `T` equals `S`**: a delivered or cancelled order accepts no `status` at all |
| `status` = `S` | `placed`, `confirmed`, `packed` or `shipped` | no-op: nothing changes, no error |
| `status` = `T` | one step forward (`placed→confirmed`, `confirmed→packed`, `packed→shipped`, `shipped→delivered`) or `placed`/`confirmed`/`packed` → `cancelled` | the move |
| `status` = `T` | anything else: a skip, a step back, `shipped → cancelled` | **422** `errors.status` "An order cannot go from S to T." (the stored words: `placed`, `confirmed`, `packed`, `shipped`, `delivered`, `cancelled`) |
| `carrier`, `tracking`, `eta`, `customer`, `phone`, `email`, `address` or `city` | `delivered` or `cancelled` | **422** naming each such field present: "An order that is S cannot be changed." (this sentence replaces any value sentence for the same key) |
| `note` | any | allowed |

Because the rule reads `S` before the move, one body may move an order and correct it together: `{"status":"delivered","eta":"2026-11-27"}` on a `shipped` order is a `200`; the same body on a `delivered` order is two `422`s.

**Cancelling** (`status: "cancelled"` from `placed`, `confirmed` or `packed`) is `orders.Cancel` inside the same transaction (ERD §3, §5): the status changes by a conditional `UPDATE`, and **only if a row changed** the order's pairs are locked in `(product_id, size)` order and returned by `stock.Return` (not for one of the 14 sample orders, ERD E1). A second cancel, sent after the first or at the same moment, finds `cancelled` under the lock at row 11 and is `422` "An order cannot go from cancelled to cancelled."; the pairs went back exactly once. The same holds against the 30-minute expiry and a late VNPay notice (item 8 calls the same `Cancel`). The shoes' last-pairs lists, sold-out lists and totals follow with no statement of their own (they are the view `product_stock`).

**Example.**

```
PUT /v1/orders/doi-24818
{ "status": "shipped", "carrier": "GHN", "tracking": "GHN-PT27XM", "eta": "2026-11-30", "note": "" }
→ 200  { …the dealer view of §3.1, status "shipped", carrier "GHN", tracking "GHN-PT27XM", eta "2026-11-30", note "" }

PUT /v1/orders/doi-24818   { "status": "packed", "email": "abc", "total": 1 }      (the order is placed)
→ 422  { "error": "invalid", "message": "Check the details of this order.",
         "errors": { "status": "An order cannot go from placed to packed.",
                     "email": "Enter a valid email address, or leave it empty.",
                     "total": "This cannot be changed." } }          (the order is untouched)
```

`PUT` with `{}` is `200` and changes nothing (no statement writes, `updated_at` stays). The customer's order page and `/track` read the new `status`, `carrier`, `tracking` and `eta` the next time they load; nothing is pushed (story decision 8). A changed `phone` is the one `/track` matches from then on (decision 7): the matcher reads the stored value.

| Code | HTTP | Trigger |
|---|---|---|
| `unauthenticated` / `forbidden` | 401 / 403 | Row 2 |
| `unsupported_media_type` | 415 | Row 3 |
| `payload_too_large` | 413 | Row 4 |
| `bad_request` | 400 | Rows 5 and 7: not one JSON object, a key the order does not have, a value of the wrong type. `errors` names it |
| `not_found` | 404 | Rows 1, 6, 10: no such order, an id that cannot be one |
| `invalid` | 422 | Rows 8, 9, 11: a refused name, a value rule, a state rule. `errors` names **every** field at fault |
| `internal` | 500 | A failed statement, a lock that timed out (3 s), the 5-second context, a deadlock (none is possible by construction, ERD §4: it is logged and answered, not retried). Generic body; nothing is stored. The cause is logged beside the order id and the request id, **never a field value** |

### 3.4 `DELETE /v1/orders/{id}` (project §3.16)

**Purpose** — remove an order. **Only a cancelled, unpaid order.** **Auth** — dealer (`401` / `403` before anything is read). No body is read; one that is sent is ignored.

| # | Check | Answer |
|---|---|---|
| 1 | The route exists; a dealer session | `404` / `401` / `403` |
| 2 | `{id}` matches `^[a-z0-9][a-z0-9._-]*$`, at most 100 characters | `404 not_found`, no query |
| 3 | Open a transaction, `SET LOCAL lock_timeout = '3s'`, `orders.LockOrder(id)` | `404 not_found` for no such order (a second `DELETE` of the same id is this) |
| 4 | The order is not `cancelled` — `placed`, `confirmed`, `packed`, `shipped` or `delivered`, paid or not | `409 conflict`, `message` **"Cancel the order first."** |
| 5 | The order is `cancelled` **and** `paid` | `409 conflict`, `message` **"A paid order cannot be deleted; refund it in VNPay's portal first."** |
| 6 | `DELETE FROM orders WHERE id = $1`; commit | `204`, no body |

The two `409` bodies are `{"error":"conflict","message":"<the sentence>"}` with **no `errors` key** (the envelope carries `errors` only on the codes of §2.3): item 10's delete question and the order page both show `message`. Rule 4 is judged before rule 5, so a placed, paid order is told to cancel it first. On `204` the order's pairs, tokens, payment attempts and notices are gone with it (ERD §7), the shelf counts and the order-number counter are untouched, the order's token and the dealer's `GET` now answer `404`, and the next order placed is numbered above it.

| Code | HTTP | Trigger |
|---|---|---|
| `unauthenticated` / `forbidden` | 401 / 403 | Row 1 |
| `not_found` | 404 | Rows 1–3: no route, an id that cannot be one, no such order |
| `conflict` | 409 | Rows 4–5, nothing deleted |
| `internal` | 500 | A failed statement, a lock timeout, the 5-second context. Generic body; nothing deleted |

**Notes.** Idempotent by being final: the repeat is `404`. Not rate limited (a dealer session). One log line at `info` per write — `order write`, `action` (`update`, `cancel`, `delete`), `id`, `fields` (the **names** that changed, never a value) and the request id the access log carries — and none for a refusal.

## 4. What the Go code does, and where

Names are Dev's; the split is not. Each file stays well under 300 lines. The import graph stays acyclic: `admin → orders, data, stock, auth, httpx`; `orders → data, stock, auth, httpx`; `payments → orders`; `orders` imports neither `admin` nor `payments`.

| File | Holds |
|---|---|
| `internal/admin/orders.go` | `RegisterOrders(mux, db, log)`; the pipeline of §3.3 rows 1–13 and §3.4 (transaction, answers, the log line); reuses the helpers of `staff*.go` (`reply`, `refuse`, `notFound`, `invalid`, `Pool`) as `records.go` does |
| `internal/admin/orders_input.go` | **Pure**, table-tested: reads the body into an `orders.Patch`, collects the 400 faults (rows 5, 7) and the 422 faults (rows 8, 9). The field table (key, label, rule, sentences) is the only place a column name for the `UPDATE` comes from |
| `internal/orders/update.go` | `Patch` (one `*string` per editable field, `nil` = not sent); `Judge(stored Locked, p Patch) map[string]string` — the **pure** state rules of row 11, the one function that knows the six statuses' moves; `Update(ctx, tx, id, stored, p)` — applies the move and the fields (§3.3 row 13); `Delete(ctx, tx, id)` — rows 3–6 of §3.4, returning `ErrNotFound` or a `*Refusal` (the type the pay route's 409 already uses) |
| `internal/orders/settle.go` | Item 8's `PayWindow`, `LockOrder`, `MarkPaid`, `Cancel`, `CheckPayable` (item 8 services §5.2). This story changes **`Cancel`** in two ways and nothing else (ERD E1, E2): `RETURNING is_sample` and no pairs for a sample; `stock.Lock` of the order's keys before `stock.Return` |
| `internal/orders/dealer.go` | `dealerViewSQL`, `ReadDealer(ctx, q, id)` (the one reader of the dealer view: the `GET` and the `PUT`'s answer both call it, so they cannot differ), the `transaction_no` fragment (ERD §6), and the team list's mount |
| `internal/orders/list.go` | `dealerList` gains `product_name` (`LEFT JOIN products pr ON pr.id = oi.product_id`; `coalesce(pr.name,'')`, filter `pr.name`) and `transaction_no` (`Private`), sets `DefaultOrder: "t.placed_at DESC, t.seq DESC"`; `phone`, `email`, `address`, `note` stay `Private` |
| `internal/data` | `Collection.DefaultOrder string` (empty = `t.seq`, what `orderBy` returns today when no `sort`); `ListHandlerServing(db, log, c, served ...Access)`; `selectList` and `listSQL` take the set of access classes they serve. **`ListHandler` is `ListHandlerServing(db, log, c, Public)`**, so every registry list and every test of it is unchanged. Filters, ranges, sorts and words keep their `Access == Public` test, so a `Private` column is served and still refused as a filter |
| `internal/orders/orders.go` | `Register` mounts the team's list as `auth.Require(admin, staff)(data.ListHandlerServing(db, log, dealerList, data.Public, data.Private))`, and `read` gains the dealer branch of §3.1 |
| `internal/server/server.go` | `admin.RegisterOrders(mux, pool, log)` |

The `Collection` comment in `data/collections.go` that says handlers put "ONLY the Public columns" in their SELECT is rewritten to say that only a handler that names `Private` in its served set selects it (today: the team's orders list), and that nothing but `Public` is ever filterable, sortable or searchable. **If item 10's `View` and `selectList(view)` are on `main` first**, the served set is the same idea one level down: `ListHandlerServing` passes the classes its caller names, `ListHandler` stays pinned to `Public` (item 10's D13), and the second PR to merge adapts to the other's signatures.

## 5. The frontend at the swap (the backend PR's frontend half)

All of it is in `src/lib/dealer-handles-orders-status-courier-payment.ts`, `AdminOrders.tsx`, the mock's deletion and the test. No component changes.

| # | Change | Why |
|---|---|---|
| 1 | `transport` becomes `read: (id) => api.get<DealerOrder>(\`orders/${encodeURIComponent(id)}\`)`, `update: (id, body) => api.put<DealerOrder>(…, body)`, `remove: (id) => api.delete(…)`. `api.ts` is **not edited** (`put` and `delete` are on `main`; `delete` resolves on `204`) | The mock is deleted |
| 2 | A `read` answer that is not an object with an `items` array is `failed`, not a page that throws on `order.items.map` | A 200 that is not the dealer view (a proxy's page) must not blank the order page |
| 3 | `AdminOrders.tsx`: the "shoe" column renders `r.product_name` (falling back to the id when it is `""`), and the `useCollection("products", { pageSize: 100 })` read and its `Map` are deleted | §2, row 10 |
| 4 | Delete `src/lib/mock/dealer-handles-orders-status-courier-payment.ts` and the `?mock=` reading. **`src/lib/mock/orders.ts` stays** (the Track page imports it) | SC-40; the product must not read a `mock` parameter |
| 5 | `src/test/dealer-handles-orders.test.tsx`: keep every assertion; the `replaceState("?mock=…")` cases become `vi.mock("../lib/api")` stubs that throw the **real** `ApiError` — `new ApiError(0, "network", …)` for offline, `(401, "unauthenticated", …)` for expired, a promise resolved after two seconds for slow, `(422, "invalid", …, errors)` for a refusal, `(409, "conflict", "Cancel the order first.")` for the delete. One case: a `read` of `{}` is `failed` | SC-31 to SC-38 |
| 6 | **How the whole page and the list limit what they send (story decision 2).** The order page sends `changes(order, values)`: only the editable fields that changed, the note alone for a locked order (already built; it is the only caller of `update` besides the list). The **list's row menu "Edit"** is item 10's `RecordDialog`, whose body is `bodyOf(specs, values, existing)`: the keys of its `fields` prop whose value differs from the row, never the whole row. `AdminOrders` passes exactly the ten editable fields as `fields`, so that body is a subset of them, changed fields only; `bodyOf` adds `pairs` / `seen_pairs` only when the row has `sizes`, which an order never does. The server's rule does **not** loosen: a body naming `total`, `number`, `paid` or any refused field is `422` whatever the value. If item 10's backend ever changes `bodyOf` so that it sends a field outside `fields`, the first symptom is a `422` naming it on every list save, and the fix belongs to item 10 | The story names this as TL's to design; no code of this story's runs in the list's dialog |
| 7 | The list's Delete is item 10's `DeleteQuestion` → `store.deleteRow("orders", id)` → `DELETE /v1/orders/{id}`; its 409 `message` is shown inside the open question. Nothing here | SC-37 |

Not edited by this PR: `store.ts`, `ListPage.tsx`, `RecordPage.tsx`, `Form.tsx`, `fields.tsx`, `content.json`, `schema.json`, `theme.css`, `App.tsx`, `api.ts`, `admin-nav.ts`, `lib/mock/orders.ts`.

## 6. Tests the backend PR ships

Database cases run in CI (`TEST_DATABASE_URL`); the pure ones run everywhere. **The seeded shoes hold at most 3 pairs per size**: a case that needs more sets them with plain SQL first.

| Test | Proves |
|---|---|
| `Judge` table test: every `(S, T)` of the 6 × 6 grid, `S == T` on each status, every locked field × `delivered` and `cancelled`, `note` on both | SC-4 to SC-6, SC-13, SC-16 |
| `orders_input` table tests: each field at its limit (200 / 201, 5,000 / 5,001, phone 8 / 9 digits, `eta` `2026-11-26`, `""`, `26/11/2026`, `2026-13-40`, `2026-2-3`, `tomorrow`), whitespace-only, U+0000, `null` and a number for a text field (400), `colour` (400), every refused name with its own value (422), `paid` true and false, `id` ignored, `{}`, a body that is an array or empty, two faults named together | SC-13, SC-15, SC-17, SC-18 |
| SQL text tests: the public `viewSQL` and the public list SQL name none of `phone`, `email`, `address`, `note`, `transaction_no`, `is_sample`, `idempotency_key`, `account_id`; `dealerViewSQL` names the five and none of the last three; the team list's SELECT names the five and its filter, sort and search expressions name none | SC-2, SC-3 |
| Route-table test: `PUT` and `DELETE /v1/orders/{id}` exist behind `auth.Require`; `PUT /v1/orders`, `PATCH /v1/orders/x`, `POST /v1/orders/x` are `404`; no session `401`, a customer `403` for both, **for an id that does not exist too** | SC-1, SC-23 |
| Handler cases with real sessions (admin, staff, a `users` row of role `customer`, none, a forged and an expired cookie): `401` / `403` leave the order, its pairs and the counts as they were; a refused body leaves the order untouched (`{"status":"confirmed","total":1}`) | SC-1, SC-17 |
| DB: place orders through `POST /v1/orders` and cancel them — 2 pairs of one size (3 left → 5), a two-shoe order (A·38 +1, B·42 +2, nothing else moves), a last-pair order (sold-out → last-pairs, the shoe is in `GET /v1/products?size=` again), from `placed`, `confirmed` and `packed`; a paid cancelled order is `paid` still; a **sample** cancel changes no count | SC-7, SC-8, SC-11, E1 |
| DB: the same order cancelled twice in a row (`422` "cancelled to cancelled", counts unchanged), and **two cancels at the same moment, 20 fresh orders**: exactly one `200` and one `422` each time, pairs back once | SC-9 |
| DB: `Cancel` against a concurrent `Place` of the same shoe and size (50 goroutines of each, `-race`), and against `SetOne` / a shoe save when item 10 is on `main`: no deadlock (`40P01`), no count below 0, the shelf equals placed − taken + returned | ERD §4 |
| DB: `shipped` and `delivered` cancel → `422` "… from shipped to cancelled." / "… delivered to cancelled."; every skip, step back and `delivered`/`cancelled` start of SC-5; the courier fields on `shipped`, `placed`, `confirmed`, `packed`; `eta` `""` clears to no date; the customer fields on `packed` | SC-4, SC-10, SC-12, SC-14 |
| DB: the customer's token read shows the new status, carrier, tracking and `eta`; a changed phone is the one `POST /v1/order-lookups` matches | SC-4, SC-12, story decision 7 |
| DB: delete — each non-cancelled status `409` "Cancel the order first." with the order, pairs, tokens and counts untouched; cancelled unpaid → `204`, pairs, tokens, attempts and notices gone, counts and the counter unchanged, a second `DELETE` `404`, the next placed order numbered higher; cancelled and `paid` → `409` the paid sentence, everything kept | SC-20 to SC-22 |
| DB: the dealer reads — a two-pair order in the order sent with `items`; `transaction_no` `""` (and, once item 8's tables exist, the number of a success notice inserted by the test); `GET /v1/orders` carries the five fields and `product_name` on every item, none has `items`; newest first; `?phone=`, `?email=`, `?address=`, `?note=`, `?transaction_no=` and `sort=phone` are `400`; a customer's token read and a stranger's read carry none of the five; `Cache-Control: no-store` | SC-2, SC-3, SC-19, SC-24 to SC-26 |
| DB: `PUT` and `DELETE` of an unknown id and of `NOPE!` are `404`; 100 sequential `PUT`s and 100 `DELETE`s of cancelled orders, p95 under 1 s | SC-23 |

## 7. Where each criterion is enforced

| Criteria | Enforced by |
|---|---|
| SC-1 | §1 (`auth.Require` first, on both literal patterns), §3.3 rows 1–2, §3.4 row 1 |
| SC-2, SC-3 | §3.1 (`dealerViewSQL`, the dealer branch outside `authorize`), §3.2 (`Private` served by one handler, refused as a filter), §4 |
| SC-4 to SC-6 | §3.3 state rules (row 11, `Judge`) |
| SC-7 to SC-11 | §3.3 "Cancelling", ERD §3–§5 (`Cancel`, `Lock`, `Return`, E1) |
| SC-12 to SC-16 | §3.3 field table and state rules |
| SC-17, SC-18 | §3.3 rows 7–8 (400 for an unknown key, 422 by name for the twelve refused names, all or nothing) |
| SC-19 | §3.1, ERD §6 (UNVERIFIED for the number until item 8's tables exist) |
| SC-20 to SC-22 | §3.4, ERD §7 |
| SC-23 | §3.3 row 10, §3.4 row 3; ERD §5 (about ten statements, each by key) |
| SC-24 to SC-26 | §3.2 (default order, filters, `product_name`, the menu's count) |
| SC-27 to SC-30 | §3.1 and the UI PR (the page); §3.1 notes on a dealer's browser |
| SC-31 to SC-38 | The UI PR; §5 rows 1–7 (the swap) over §3.3 and §3.4's sentences |
| SC-39 | The UI PR (judged on the mock); Verify at 390 and 320 px |
| SC-40 | §5 row 4 and the "not edited" list |

## 8. Order of work, and merge hygiene

1. **Branch.** Cut the backend branch from the UI branch (`feature/dealer-handles-orders-status-courier-payment-ui`), since it deletes the mock that branch adds; PR #77 merges first. Grep the head for `^<<<<<<<` and `^>>>>>>>` before any merge (the project has been bitten by markers on `main`).
2. **Order of work.** `stock.Return` and `orders.Cancel` (with E1 and E2) → `Judge`, `Update`, `Delete` and `orders_input` → `admin.RegisterOrders` → the dealer reads (`data` served set and default order, `dealerList`, `ReadDealer`, the `read` branch) → the frontend swap.
3. **Item 8 (PR #83) shares `Cancel`, `LockOrder` and `stock.Return`.** Whichever merges first creates them as item 8's services file §5.2 and §5.4 write them **plus E1 and E2**; the second adopts them, keeping the superset. If PR #83 is on `main` when this backend is cut, `transaction_no` is the fragment of ERD §6 from the first commit; if it is not, it is the constant `''` in one place and SC-19's number half stays UNVERIFIED, and the PR says so.
4. **Item 10 shares `data`, `stock`, `server.go` and the `Collection` comment.** Every addition here is additive and independent; the second PR to merge adapts to the first's signatures (§4, last paragraph).
5. **Items 6 and 15 share `read.go`.** The dealer branch sits **before** whatever `authorize` / `Access` extraction those PRs make and does not call it, so it is the same few lines on either side of the merge. The token and customer paths must keep passing their own tests unchanged.
6. Database tests (`TEST_DATABASE_URL`) first run in CI; Verify confirms they were not skipped. The admin screens have not been walked in a browser at 390 and 320 px (the UI review said so): Verify must.

## 9. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| D1 | Where do the handlers live? | `admin` (the dealer's writes; `_base.md` §1.1 says "`admin` handlers call `orders.Update` / `Cancel` / `Delete`"), the rules and SQL in `orders` (its only writer) | Everything in `orders`: the dealer's input judge and sentences would sit beside checkout's. Everything in `admin`: it would write another package's table |
| D2 | How does a dealer read an order without a token? | A branch in `read`, before and outside `authorize`; its own statement `dealerViewSQL` | `Access.Dealer` inside `authorize` (item 6's table suggests it): `authorize` returns a `Ref` that cannot hold a private column, and that is what the SC-3 test stands on; a dealer mode that returns more columns would weaken it for every later story |
| D3 | `PUT` answers one fault or all? | **All**, in one `errors` map | First fault only: the page would refuse a second field after the dealer fixed the first, one round trip each |
| D4 | `404` or `422` for a faulty body to a missing order? | `404`: the lock comes before the state and value-by-state rules, and the missing order is told first | `422` first: a deleted order's page would show field errors for a record that is not there |
| D5 | Is a refused name refused when it carries the stored value? | **Yes**, whatever the value and type (story decision 2, `paid: false` on an unpaid order is `422`) | Tolerating an unchanged value: it would let a whole-record client work by accident and let a future change to `total` slip through a "same value" test |
| D6 | One body moves the order and corrects it? | Yes: the locked-field rule reads the status before the move | After the move: `{"status":"delivered","eta":…}` on a shipped order would be refused for a state the order is only entering |
| D7 | `email` stored as typed? | Validated with `auth.NormalizeEmail`, stored trimmed **as typed** | Storing its lower-cased result: a dealer who types `Long@Example.com` would see another text come back |
| D8 | A body with an unknown key | `400` with `errors.<key>` (§2.5's rule), not `422` | `422` like the shoe collections: those are one of §2.5's three SRS-forced exceptions; ADMIN-016 AC-15 forces `422` only for "a field that may not change" |
| D9 | Retry a deadlock once, as item 10's writes do? | No | A retry would hide a lock order that is wrong. ERD §4 makes a deadlock impossible by construction; if one is reported it is a bug to see |
| D10 | `product_name` and the default order are departures from `_base.md` | Both stated here (§2, §3.2), both additive to the frontend's contract | Keeping the first 100 shoes lookup, and `seq ASC`: the first breaks SC-25 beyond the hundredth shoe, the second SC-24 from the first day |

## 10. Open questions

None blocks the backend. Q1 (a customer who pays after the dealer cancelled and deleted the order) and Q2 (the list's Edit dialog offers every status) are in `erd/dealer-handles-orders-status-courier-payment.md` §8 with their defaults. **Migration plan:** none — forward, backward and populated-table safety are in that file's §9.
