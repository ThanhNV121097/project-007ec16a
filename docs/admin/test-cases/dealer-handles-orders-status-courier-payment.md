# Test cases — Dealer handles orders: status, courier, payment

Module: `admin` · Plan item 17 · Story: `docs/admin/stories/dealer-handles-orders-status-courier-payment.md` (SC-1 … SC-40)
Requirements: ADMIN-016 and the order parts of ADMIN-004 (AC-6, AC-9), ADMIN-011 (AC-12, AC-14, AC-15, AC-20), ADMIN-014 (AC-7, AC-8) in `docs/admin/SRS.md` · Contract: `docs/architecture/services.md` §2.1, §2.2, §2.3, §2.4, §2.5, §2.8, §3.11, §3.14, §3.15, §3.16

Written before the implementation exists, from the requirements, the story and the contract only. The technical design of this story (extra endpoints, a changed shape) does not exist yet; no case invents one.

## 1. Risk and depth

| Part | Risk | Why | Depth |
|---|---|---|---|
| Who may call `PUT` / `DELETE` and who reads the private fields (SC-1 … SC-3) | **High** | One missing guard lets a visitor cancel orders, edit a delivery address or read every customer's phone; one extra field in a customer read leaks private data | 401 / 403 on both verbs with no session, a forged cookie, an expired session and a customer; an unknown id answers 401 to a visitor; admin and staff answers compared byte for byte; the five private fields absent from every token, linked-customer and anonymous read |
| Cancelling returns the pairs exactly once (SC-7 … SC-11) | **High** | Stock is money: a double return oversells, a missed return hides shoes the shop has; a cancel races the expiry ticker, a late notice and a second cancel | Exact counts before and after, the sold-out list and the size filter, a two-shoe order, a paid order, a second cancel, 20 concurrent pairs of cancels, a failed return rolling the status back |
| Status moves (SC-4 … SC-6) | **High** | The customer reads the status as the truth; a skip or a step back puts a shipped parcel back on the shelf in the customer's eyes | Every legal step, eight refused moves with the exact sentence, an unknown status, the no-op rule, delivered and cancelled closed |
| What never changes and payment (SC-17 … SC-19) | **High** | The owner decided online payment only: a hand-marked "paid" or an edited total is a financial record rewritten | Every refused field by name, all-or-nothing, `paid` true and false on unpaid and paid orders of all four ways to pay, `transaction_no`, no change from a `PUT` |
| Delete (SC-20 … SC-23) | **High** | Irreversible; a paid order carries a payment record the owner may still need to refund | Every status refused, cancelled-unpaid accepted with everything gone, cancelled-paid refused, repeat delete, the next number, 100 sequential calls at p95 |
| Courier, customer details, note (SC-12 … SC-16) | Medium–High | Wrong values reach the customer's order page and the courier | Each field alone and together, limits one inside and one past, every refusal naming its field, delivered and cancelled orders locked except the note |
| The list, the menu and the order page (SC-24 … SC-30) | Medium | Screens that read the shop; the multi-pair sentence and the read-only payment block are not in the approved design | Copy word for word, columns and filters, first-pair shoe, counts of orders waiting, the controls inventory, the customer's page proving no private field |
| Saving and deleting from the page (SC-31 … SC-38) | Medium | The states the design never drew, on the plumbing item 10 built | Each state on a real order: saved, stale move, locked, courier, field errors, unreachable shop, ended session, delete questions, record not here |
| Phone layout (SC-39) and hygiene (SC-40) | Medium / Low | Most customers use phones but the dealer works on a phone too; process rules checked once | 390 and 320 px readings; the observable half of the hygiene rule by network and storage, the repository half `manual` |
| Contract rules (C-1 … C-7) | Medium | The backend is reviewed against these | Envelope, 415, 413, 400 for an undefined or wrongly typed field, `internal` on a failed write, the database refused |

## 2. How to read this file

- **Paths** are written as the backend mounts them (`/v1/orders/{id}`); from a browser or `fetch_url` through the gateway they are `/api/v1/orders/{id}`. Every `POST` and `PUT` is sent with `Content-Type: application/json` unless a case says otherwise.
- **Sessions** are named. `ADMIN`: the cookie from `POST /v1/auth/sign-in` with the run's `ADMIN_EMAIL` / `ADMIN_PASSWORD`. `STAFF`: the cookie of FX-STAFF (below). `CUSTOMER`: the cookie of FX-CUSTOMER. `NONE`: no cookie. `FORGED`: `session=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA` (43 characters, no such session). `EXPIRED`: STAFF's cookie after `UPDATE sessions SET created_at = now() - interval '8 days', expires_at = now() - interval '1 day' WHERE user_id = (SELECT id FROM users WHERE email = 'tc-sa@example.com')`. In a browser case "signs in as X" means the run types the email and password on `/admin/sign-in` and presses "Sign in".
- **Starting state of every case**: the 14 sample orders `doi-24804` … `doi-24817` are untouched and no case edits, cancels or deletes one (they are read-only evidence). Anything a case creates is removed or put back afterwards (RESET below), so no case depends on another.
- **The error body.** Every non-2xx answer is JSON `{"error":<code>,"message":<non-empty sentence>}` with `Content-Type: application/json; charset=utf-8`, plus `"errors":{<field>:<non-empty sentence>}` **only** on 400 and 422 (`services.md` §2.3). Codes used here: `unauthenticated` 401, `forbidden` 403, `not_found` 404, `conflict` 409, `invalid` 422, `bad_request` 400, `unsupported_media_type` 415, `payload_too_large` 413, `internal` 500. A body never holds SQL, a stack trace, a file path, a table name, a hash or a typed secret.
- **"Unchanged"** for an order means: `GET /v1/orders/<id>` as ADMIN returns, field for field, what it returned before the call (`status`, `paid`, `carrier`, `tracking`, `eta`, `customer`, `phone`, `email`, `address`, `city`, `note`, `total`, `items`). For the shelf it means every `product_sizes.pairs` count read by `SELECT product_id, size, pairs FROM product_sizes ORDER BY 1, 2` is equal to the earlier reading.
- **Sentences the SRS or the story fixes** are asserted word for word: "An order cannot go from <from> to <to>." (stored words `placed`, `confirmed`, `packed`, `shipped`, `delivered`, `cancelled`), "Only the payment provider marks this order paid.", "Cancel the order first.", "A paid order cannot be deleted; refund it in VNPay's portal first.", "Year-month-day, for example 2026-11-26", "We could not save. Please try again.", "Your session ended. Sign in again.", "This record is not here", "It may have been deleted since the link was made.", "Back to the list", "Saved.", "Saving…", "Paid", "Awaiting payment", "VNPay transaction <number>", "EU <size>". Any other sentence in `errors.<field>` is asserted only as non-empty English text under the right field name.
- **Browser words** are the story §6: title "Orders"; search "Search by number or customer"; filters "Status" (Placed, Confirmed, Packed, On its way, Delivered, Cancelled) and "Payment"; tabs Order, Delivery, Customer; fields Status, Carrier, Tracking number, Expected, Customer, Phone, Email, Address, City or province, Internal note; buttons "Delete", "Save changes", "Cancel"; the question "Delete <title>?".
- **Elements are found by role and label, never by class**: the title is `main h1`; a table is `table` with `th` headers and `tbody tr` rows; a tab is `[role=tab]`; a dialog or question is `[role=dialog]` or `[role=alertdialog]`; a field is found by its visible label; "the sentence beside a field" is the text of the element the field's `aria-describedby` names (or of the element in the same field wrapper) and the field carries `aria-invalid="true"`; "announced" means that sentence sits in an element with `role="alert"` or in an `aria-live` region; a row's menu button is the one `button` in the row's last cell; the left menu is `nav`.
- **Browser requests stubbed, delayed or failed.** Where a case says the shop's answer is delayed, failed, aborted or answered with a status, the verifier arranges it with request interception on the named URL inside the `interact_page` run (everything else, the cookie included, stays real). A case that needs this and finds no interception available is UNVERIFIED for that reason, not failed. "The shop unreachable" is also arranged by stopping the backend (the gateway then answers 502) where the instrument allows.
- **Concurrent requests** (SC-9b) are sent with `Promise.all` from the page of a signed-in browser (`interact_page`), 20 rounds, each round with its own fresh order.
- **Money** is `vi-VN` (`3.290.000 ₫`; the space before `₫` may be a no-break space). **DATE(x)** is the order's `placed_at` written by the page's own `formatDateTime`: `dd/mm/yyyy, hh:mm` in `vi-VN`, in the browser's zone; the verifier computes it in the page with `Intl.DateTimeFormat("vi-VN")` from `placed_at` and compares text, not a hard-coded string.

## 3. Fixtures

| Id | What is made | Used by |
|---|---|---|
| FX-STAFF | Made by `POST /v1/admin/staff` as ADMIN: name "Tc Staff A", email `tc-sa@example.com`, password `staff-pass-A1`, active | every case that names `STAFF` or `EXPIRED` (SC-1, SC-2, SC-1d, SC-23a) |
| FX-CUSTOMER | Made as FX-STAFF with name "Tc Customer", `tc-customer@example.com`, `customer-pass-1`, then `UPDATE users SET role = 'customer' WHERE email = 'tc-customer@example.com'` (a customer row whose hash the backend wrote; the sign-in endpoint serves any row) | SC-1, SC-3, SC-27 |
| FX-CUSTOMER-2 | The same with `tc-customer2@example.com`, `customer-pass-2` — a second customer to whom no order is linked | SC-3 |
| REQ | The request body of every order placed for a case, sent as `POST /v1/orders` with `Content-Type: application/json` and a fresh header `Idempotency-Key: tc-ho-<case>-<n>`: `{"items":[{"product":"samba-og-w-core-black","size":"38","qty":1}],"delivery":"standard","payment":"vnpay","customer":"Lan Nguyen","phone":"0912 345 678","address":"12 Example Street, Ward 1","city":"Da Nang","note":"Call before delivering."}`. The answer's `order.id` is `<id>` and `token` is `<token>`; a placed order is `status` `placed`, `paid` false, total 3.290.000 ₫ at the shoe's current price | every case that needs an order |
| FX-ORDER(state) | An order placed with REQ, then put in the state with `set_db_state`: `UPDATE orders SET status = '<state>' WHERE id = '<id>'` (placed, confirmed, packed, shipped, delivered, cancelled). A state reached this way touches no shelf count; a case about stock moves the order by real `PUT`s instead and says so | most |
| FX-PAID(id) | `UPDATE orders SET paid = true WHERE id = '<id>'` (what VNPay's notice writes; the real notice is plan item 8's) | SC-11, SC-18, SC-22, SC-37 |
| FX-WAY(id, way) | `UPDATE orders SET payment_method_id = '<way>' WHERE id = '<id>'` with `way` one of `vnpay`, `cod`, `bank`, `momo` (the four `payment_methods` rows exist) | SC-18, SC-25 |
| FX-TWO | An order of two pairs: REQ with `items` `[{"product":"samba-og-w-core-black","size":"38","qty":1},{"product":"gazelle-indoor-solar-gold","size":"42","qty":2}]` (shoe A 38 × 1, shoe B 42 × 2), customer `Lan Nguyen`, city `Da Nang`; total = 1 × A's price + 2 × B's price | SC-2, SC-8, SC-25, SC-29 |
| SHELF(product, size, n) | `UPDATE product_sizes SET pairs = <n> WHERE product_id = '<product>' AND size = '<size>'`; the old value is read first (`SELECT pairs …`) and put back at the end of the case. The dealer's `GET /v1/products/<id>` then carries `pairs` (size → count), `sizes_low`, `sizes_out`, `stock` | SC-7, SC-8, SC-9, SC-33 |
| READ-SHELF(product) | `GET /v1/products/<product>` as ADMIN: `pairs`, `sizes_low`, `sizes_out`, `stock`; and the public `GET /v1/products/<product>` and `GET /v1/products?size=<size>` as NONE | SC-7, SC-8, SC-33 |
| SEEDTOKEN(id) | Only for a sample order: `INSERT INTO order_tokens (order_id, token_hash) VALUES ('<id>', sha256('tc-seed-token-0001'::bytea))`, header `X-Order-Token: tc-seed-token-0001`; removed afterwards. If the backend hashes the token some other way the row opens nothing and the case is UNVERIFIED, not failed | none needed by a placed order |
| RESET | End of every case that created anything: `DELETE FROM orders WHERE is_sample = false`; `UPDATE counters SET value = 24817 WHERE name = 'order_number'` (only when no later case relies on a number); every SHELF put back; a renamed table renamed back; the database made available again; `DELETE FROM users WHERE email LIKE 'tc-%'`; any `payment_attempts` / `payment_notices` row a case inserted deleted with its order | all |
| E200 / E201 | A text of exactly 200 / 201 characters (`a` × 200, `a` × 201); N5000 / N5001 the same at 5,000 / 5,001 characters | SC-13, SC-15 |

## 4. Open points found while writing (reported, not resolved here)

1. **The answer of `PUT` carries which shape?** `services.md` §3.15 says "200 the dealer view"; §2.8 says a single read carries `items`. The cases assert the five private fields and the changed values on the answer and read `items` only from `GET`.
2. **Order of 404 and 422 on `PUT`.** The contract does not say whether a missing order is answered before the body is validated. Every case that expects 404 sends a valid body; none sends an invalid body to a missing id.
3. **Limits the story does not pin for `PUT`.** `services.md` §2.5 gives checkout limits (name 100, city 100, phone 20) and dealer limits (text 200); §3.15 says "the limits of §2.5". The cases pin only values that are inside both readings (`customer` and `city` of 100 characters accepted; `address` of 200 accepted and 201 refused; 201 characters refused for every text field) and a phone of 9 digits accepted, 8 refused. A `customer` of 150 characters is deliberately not asserted.
4. **A wrongly typed editable field** (`{"carrier":5}`). §2.5 lists three 422 exceptions, none for this; the cases expect 400 `bad_request` naming the field.
5. **The delete question's title.** The story says `<title>` is "the heading the page shows for the order"; the current helper `titleOf` falls back to the order's id (`doi-24818`) because an order has no name. SC-37 compares the question with the text of `main h1` on the page and does not pin a particular title.
6. **A database that refuses connections.** `services.md` §2.3 says a database that cannot be reached is `internal` 500, the verification brief reserves 503 for a refused dependency. With no database even the cookie cannot be resolved. C-7 pins what both readings share (a JSON error envelope, no 2xx, no leak, nothing changed afterwards) and records the status it saw.
7. **A body that is refused names which fields?** SC-17 says "a refused body applies nothing". For `{"status":"confirmed","total":1}` the cases assert `errors.total` is present; they do not assert whether `errors` also holds `status` (it should not, `placed → confirmed` is legal).
8. **Plan item 8's payment tables.** Until `payment_attempts` and `payment_notices` exist the transaction number is `""` for every order; the number half of SC-19 and the "payment attempts are gone" half of SC-21 are UNVERIFIED then, and say so.
9. **The status control's labels.** `schema.json` labels `shipped` "Shipped"; the story (decision 1) says the status control and the lists read "On its way". The cases follow the story.

## 5. Who may call, and what the dealer reads — SC-1 … SC-3

**Scenario**: SC-1a — `PUT` with no session, a forged cookie and an expired session answers 401 and changes nothing
**Given**: FX-STAFF; `<id>` is an order placed with REQ (status `placed`); the shelf is read (the order's pair is counted); `EXPIRED` is made from FX-STAFF's own cookie.
**When**: `PUT /v1/orders/<id>` with `{"status":"confirmed"}` is sent with `NONE`, with `FORGED` and with `EXPIRED`; then `PUT /v1/orders/doi-99999` (an id that does not exist) with `{"status":"confirmed"}` and `NONE`; then ADMIN reads the order and the shelf.
**Then**: Each of the four answers is 401 `{"error":"unauthenticated","message":<non-empty>}` with `Content-Type: application/json; charset=utf-8`, no `errors` key and no order data (the body holds only `error` and `message`); the call for `doi-99999` is 401, not 404; the order is unchanged (still `placed`) and the shelf is unchanged.
Traces: SC-1 (ADMIN-004 AC-6, AC-9; ADMIN-016 AC-19)
Check: fetch_url

**Scenario**: SC-1b — `DELETE` with no session, a forged cookie and an expired session answers 401 and deletes nothing
**Given**: FX-STAFF and `EXPIRED`; `<id>` is FX-ORDER(cancelled) with an unpaid order (so only the guard stands between the call and a 204); the shelf is read.
**When**: `DELETE /v1/orders/<id>` with `NONE`, `FORGED` and `EXPIRED`; `DELETE /v1/orders/doi-99999` with `NONE`; then ADMIN reads the order and counts `order_items` and `order_tokens` rows of it with `set_db_state`.
**Then**: Each answer is 401 `{"error":"unauthenticated",…}` with no `errors` key and no order data; the unknown id is 401, not 404; the order is still readable as ADMIN with `status` `cancelled`; its one `order_items` row and its token row still exist; the shelf is unchanged.
Traces: SC-1 (ADMIN-004 AC-9; ADMIN-014 AC-10)
Check: fetch_url

**Scenario**: SC-1c — a signed-in customer's `PUT` and `DELETE` answer 403 and change nothing
**Given**: FX-CUSTOMER signed in (`CUSTOMER`); `<id>` placed with REQ (status `placed`); `<cancelled>` is FX-ORDER(cancelled); the shelf is read.
**When**: `CUSTOMER` sends `PUT /v1/orders/<id>` with `{"status":"confirmed"}`, `PUT /v1/orders/<id>` with `{}`, `DELETE /v1/orders/<cancelled>`, and `PUT` and `DELETE` of `doi-99999`; then ADMIN reads both orders and the shelf.
**Then**: All five answers are 403 `{"error":"forbidden","message":<non-empty>}` with no `errors` key and no order data (an unknown id is also 403: the guard answers before any lookup); `<id>` is still `placed`; `<cancelled>` is still there with its pair row; the shelf is unchanged.
Traces: SC-1 (ADMIN-004 AC-7, AC-9)
Check: fetch_url

**Scenario**: SC-1d — an admin session and a staff session get the same answers in every case
**Given**: FX-STAFF; two identical fresh orders `<a>` (for ADMIN) and `<s>` (for STAFF), each placed with REQ.
**When**: In turn, on each order with its own session: `PUT {"status":"packed"}`; `PUT {"status":"confirmed"}`; `PUT {"total":1}`; `PUT {"colour":"red"}`; `PUT {"note":"Ring twice."}`; `PUT {"paid":true}`; `DELETE`; `PUT {"status":"cancelled"}`; `DELETE`; `DELETE` again. Then `GET /v1/orders/<id>` and `GET /v1/orders?limit=1&sort=-placed_at` with each session.
**Then**: For every pair of answers the status code is the same (422, 200, 422, 400, 200, 422, 409, 200, 204, 404), the `error` code is the same, and the `message` and `errors` are the same text; the two 200 bodies are equal once `id`, `number` and `placed_at` are set aside; both reads answer 200 with the same keys.
Traces: SC-1 (ADMIN-004 AC-10; "an admin session and a staff session get the same answers")
Check: fetch_url

**Scenario**: SC-2a — a dealer reads one order with the private fields and every pair
**Given**: FX-STAFF; `<two>` is FX-TWO placed with phone `0912 345 678`, address `12 Example Street, Ward 1`, note `Call before delivering.` and no email; `<one>` is a sample order, `doi-24817`.
**When**: ADMIN and then STAFF send `GET /v1/orders/<two>` and `GET /v1/orders/doi-24817` with no `X-Order-Token`.
**Then**: Each answer is 200 with `Cache-Control` containing `no-store`. For `<two>`: `phone`'s digits are `0912345678`, `email` is `""`, `address` is "12 Example Street, Ward 1", `note` is "Call before delivering.", `transaction_no` is `""`; `items` has exactly 2 objects in the order sent, each with exactly the keys `product`, `name`, `colourway`, `image`, `size`, `qty`, `unit_price` — the first `product` `samba-og-w-core-black`, `size` "38", `qty` 1, the second `gazelle-indoor-solar-gold`, "42", 2; `product`, `size`, `qty` at the top are the first pair's; the sum of `unit_price × qty` equals `total`. For `doi-24817`: `phone` "0938 410 552" (digits), `address` "42 Example Street, Hải Châu", `note` "Gift — no price inside the box.", `transaction_no` `""`, `carrier` "GHN", `tracking` "GHN-PT27XM", `eta` "2026-11-26". The ADMIN and STAFF answers are equal.
Traces: SC-2 (ADMIN-016 item 8; ADMIN-011 AC-20; ORD-008 AC-3, AC-7)
Check: fetch_url

**Scenario**: SC-2b — the dealer's list carries the same five fields on every item, the first pair and no `items`
**Given**: `<two>` (FX-TWO) placed; `GET /v1/orders` read by ADMIN and by STAFF.
**When**: `GET /v1/orders?limit=100&sort=-placed_at` as ADMIN and as STAFF.
**Then**: Both answers are 200 with `Cache-Control` containing `no-store` and the shape `{"items":[…],"total":n,"page":1,"limit":100}`; every item has the string fields `phone`, `email`, `address`, `note`, `transaction_no` and none has an `items` key; the item for `<two>` has `product` `samba-og-w-core-black`, `size` "38", `qty` 1 (the first pair); the first item is `<two>` (newest first with the sort); `total` is at least 15 (14 samples plus the new order).
Traces: SC-2 (ADMIN-016 item 8, item 7; ORD-008 AC-3)
Check: fetch_url

**Scenario**: SC-3a — a read with the order's own token and no session never carries the five private fields
**Given**: `<id>` placed with REQ; `<token>` from the placing answer; the order's phone, address and note are the FORM values.
**When**: `GET /v1/orders/<id>` with `X-Order-Token: <token>` and no cookie; again with `X-Order-Token: <token>` and the `CUSTOMER-2` cookie; the raw body text is searched.
**Then**: Both answers are 200 with the public view: the keys `id`, `number`, `placed_at`, `status`, `items`, `total`, `payment`, `paid`, `delivery`, `carrier`, `tracking`, `eta`, `customer`, `city` are there and **none** of `phone`, `email`, `address`, `note`, `transaction_no` is; the body text contains neither "0912", "Example Street" nor "Call before delivering".
Traces: SC-3 (ORD-008 item 3, AC-5; ADMIN-011 item 10)
Check: fetch_url

**Scenario**: SC-3b — without a token or with the wrong access, an order answers 404 with one and the same body
**Given**: FX-CUSTOMER-2 signed in (no order is linked to it); `<id>` and `<other>` placed with REQ, with their own tokens.
**When**: `GET /v1/orders/<id>` with (1) no token and no cookie; (2) the `CUSTOMER-2` cookie and no token; (3) `<other>`'s token; (4) the header `X-Order-Token: not-a-token`; and (5) `GET /v1/orders/doi-99999` with no token; (6) `GET /v1/orders/-x` with no token.
**Then**: All six answers are 404 `{"error":"not_found","message":<one and the same sentence>}` with no `errors` key and a body byte-identical to each other (an order that exists cannot be told from one that does not).
Traces: SC-3 (ORD-008 item 2, AC-4, AC-6, AC-8, AC-12)
Check: fetch_url

**Scenario**: SC-3c — the order list is closed to visitors and customers, and a dealer's filter on a private field is 400
**Given**: FX-CUSTOMER signed in; ADMIN and STAFF signed in.
**When**: `GET /v1/orders` with `NONE`; with `CUSTOMER`; `GET /v1/orders?phone=0912` with `NONE`; and, as ADMIN and as STAFF, `GET /v1/orders?phone=0912`, `?email=a`, `?address=Example`, `?note=Gift`, and `?sort=phone`.
**Then**: `NONE` answers 401 `unauthenticated` (also for `?phone=0912`, ORD-008 AC-9); `CUSTOMER` answers 403 `forbidden`; each of the five dealer calls answers 400 `{"error":"bad_request","message":<non-empty>,"errors":{<phone|email|address|note|sort>:<non-empty>}}` naming the parameter; no answer holds an order.
Traces: SC-3 (ORD-008 AC-1, AC-2, AC-9; contract §2.4 private fields)
Check: fetch_url

## 6. Moving an order along — SC-4 … SC-6

**Scenario**: SC-4 — a placed order moves one step at a time to delivered, each step read back with the order's token
**Given**: `<id>` placed with REQ and its `<token>`; ADMIN signed in.
**When**: ADMIN sends `PUT /v1/orders/<id>` with `{"status":"confirmed"}`, then `{"status":"packed"}`, then `{"status":"shipped"}`, then `{"status":"delivered"}` — each in its own request; after each, `GET /v1/orders/<id>` with `X-Order-Token: <token>` and no cookie.
**Then**: Each `PUT` answers 200 with `status` equal to the one sent (and the five private fields present); each token read answers 200 with the same `status` and no private field; the final read shows `delivered`.
Traces: SC-4 (ADMIN-016 item 1, AC-1)
Check: fetch_url

**Scenario**: SC-5a — eight impossible moves are 422 with the stored words and leave the order as it was
**Given**: Orders made with FX-ORDER for each starting status: `placed`, `confirmed`, `shipped`, `delivered`, `cancelled`; the shelf is read.
**When**: ADMIN sends `PUT` with: placed → `{"status":"packed"}`; placed → `{"status":"shipped"}`; placed → `{"status":"delivered"}`; confirmed → `{"status":"placed"}`; shipped → `{"status":"packed"}`; delivered → `{"status":"shipped"}`; delivered → `{"status":"placed"}`; cancelled → `{"status":"confirmed"}`.
**Then**: Each answers 422 `{"error":"invalid",…,"errors":{"status":"An order cannot go from <from> to <to>."}}` word for word — for example "An order cannot go from placed to packed.", "An order cannot go from delivered to shipped.", "An order cannot go from cancelled to confirmed." (`shipped` is written `shipped`, never "On its way"); each order is unchanged and the shelf is unchanged.
Traces: SC-5 (ADMIN-016 item 1, AC-2, AC-3, AC-4)
Check: fetch_url

**Scenario**: SC-5b — a status that is none of the six is 422 naming `status`
**Given**: A placed order `<id>`.
**When**: ADMIN sends `PUT` with `{"status":"archived"}`, then `{"status":""}`, then `{"status":"Confirmed"}` (capitalised).
**Then**: Each answers 422 `{"error":"invalid",…,"errors":{"status":<non-empty>}}`; the order is still `placed`.
Traces: SC-5 (ADMIN-016 item 1)
Check: fetch_url

**Scenario**: SC-6a — the current status sent again is a no-op on placed, confirmed, packed and shipped orders
**Given**: Four orders made with FX-ORDER at `placed`, `confirmed`, `packed`, `shipped`; each read before.
**When**: ADMIN sends `PUT` with the order's own status, for example `{"status":"packed"}` on the packed one; then reads each order.
**Then**: Each `PUT` answers 200 with the same `status`; every field of every order equals the earlier reading (nothing but the status was asked and nothing changed); the shelf is unchanged.
Traces: SC-6 (ADMIN-016 item 1)
Check: fetch_url

**Scenario**: SC-6b — a delivered or cancelled order accepts no `status` at all, not even its own
**Given**: FX-ORDER(delivered) `<d>` and FX-ORDER(cancelled) `<c>`; each read before.
**When**: ADMIN sends `PUT {"status":"delivered"}` to `<d>`, `PUT {"status":"cancelled"}` to `<c>`, and `PUT {"status":"delivered","note":"x"}` to `<d>`; then reads both.
**Then**: The first answers 422 with `errors.status` "An order cannot go from delivered to delivered."; the second 422 "An order cannot go from cancelled to cancelled."; the third 422 naming `status` the same way and the note of `<d>` is unchanged (a refused body applies nothing); both orders are unchanged.
Traces: SC-6 (ADMIN-016 item 1; story §3 item 2, §5 decision 3)
Check: fetch_url

## 7. Cancelling — SC-7 … SC-11

**Scenario**: SC-7a — cancelling a packed order of 2 pairs gives 5 pairs back and raises the shoe's stock by 2
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); an order `<id>` placed with REQ but `qty` 2 (the shelf now holds 3 of `38`), moved with real `PUT`s to `confirmed` then `packed`; READ-SHELF taken (`pairs["38"]` is 3, `stock` is S, `sizes_low` and `sizes_out` do not hold "38").
**When**: ADMIN sends `PUT /v1/orders/<id>` with `{"status":"cancelled"}`; READ-SHELF again.
**Then**: The `PUT` answers 200 with `status` `cancelled`; the dealer's `pairs["38"]` is 5, `stock` is S + 2, "38" is in neither `sizes_low` nor `sizes_out`; every other size's count is as before. The order's own token read shows `cancelled`.
Traces: SC-7 (ADMIN-016 item 2, AC-5; ORD-005 item 3)
Check: fetch_url

**Scenario**: SC-7b — a size that sold out is back in the last-pairs list and in the size filter after the cancel
**Given**: SHELF(`samba-og-w-core-black`, `38`, 1); `<id>` placed with REQ (`qty` 1; `38` now has 0 pairs); READ-SHELF: `sizes_out` holds "38", `sizes_low` does not, and `GET /v1/products?size=38&limit=100` as `NONE` does not list `samba-og-w-core-black`.
**When**: ADMIN sends `PUT /v1/orders/<id>` with `{"status":"cancelled"}`; READ-SHELF again.
**Then**: 200 `cancelled`; `pairs["38"]` is 1, `sizes_low` holds "38", `sizes_out` does not; `GET /v1/products?size=38&limit=100` as `NONE` lists `samba-og-w-core-black`; the public product's `stock` is one higher than before the cancel.
Traces: SC-7 (ADMIN-016 item 2; ORD-005 item 3, AC-3)
Check: fetch_url

**Scenario**: SC-7c — a cancel whose return of the pairs fails rolls the status back and returns nothing
**Given**: A packed order `<id>` of 2 pairs of size `38` (as SC-7a); the shelf is read; `ALTER TABLE product_sizes RENAME TO product_sizes_tc` through `set_db_state`.
**When**: ADMIN sends `PUT /v1/orders/<id>` with `{"status":"cancelled"}`; then the table is renamed back and the order and the shelf are read; then the same `PUT` is sent again.
**Then**: The first `PUT` answers 500 `{"error":"internal","message":<generic sentence>}` with no `errors` key and a body that holds neither "product_sizes", "relation", "SQL" nor a stack trace; after the rename the order is still `packed` (the status update rolled back with the failed return) and the shelf equals the earlier reading; the second `PUT` answers 200 `cancelled` and the pairs come back once.
Traces: SC-9 (ADMIN-016 item 2: one transaction, "pairs back exactly once"); contract (§3.15 `internal`)
Check: fetch_url

**Scenario**: SC-8a — a two-shoe order cancels in one step and only its own counts change
**Given**: `<two>` (FX-TWO: shoe A `samba-og-w-core-black` 38 × 1, shoe B `gazelle-indoor-solar-gold` 42 × 2) placed and left `placed`; the whole shelf read as `SELECT product_id, size, pairs FROM product_sizes ORDER BY 1, 2` after placing.
**When**: ADMIN sends `PUT /v1/orders/<two>` with `{"status":"cancelled"}`; the whole shelf is read again.
**Then**: 200 `cancelled`; A's `38` is +1, B's `42` is +2, and every other row of the shelf is equal to the earlier reading (the diff is exactly those two rows).
Traces: SC-8 (ADMIN-016 item 2)
Check: fetch_url

**Scenario**: SC-8b — a confirmed order and a packed order cancel the same way
**Given**: Two more FX-TWO orders, one moved by real `PUT`s to `confirmed` and one to `packed`; the shelf read after each placing.
**When**: ADMIN cancels each with `PUT {"status":"cancelled"}`; the shelf is read after each.
**Then**: Each answers 200 `cancelled`; each time A's `38` is +1, B's `42` is +2 and nothing else on the shelf changes.
Traces: SC-8 (ADMIN-016 item 2)
Check: fetch_url

**Scenario**: SC-9a — cancelling the same order again is 422 and the pairs are not returned twice
**Given**: `<id>` placed with REQ, cancelled once by ADMIN (200); the shelf read after the first cancel.
**When**: ADMIN sends `PUT {"status":"cancelled"}` again; then reads the order and the shelf.
**Then**: 422 with `errors.status` "An order cannot go from cancelled to cancelled."; the order stays `cancelled`; every count equals the reading after the first cancel.
Traces: SC-9 (ADMIN-016 item 2, AC-6)
Check: fetch_url

**Scenario**: SC-9b — two cancels at the same moment give one 200, one 422, and the pairs back once, twenty times
**Given**: ADMIN signed in in a browser page; SHELF(`samba-og-w-core-black`, `38`, 100) read before.
**When**: In 20 rounds, the page places a fresh order with REQ (`qty` 2, key `tc-ho-race-<n>`), reads the count of `38`, then sends `PUT {"status":"cancelled"}` twice at once with `Promise.all`, reads the order and the count of `38` again.
**Then**: In every round exactly one answer is 200 (`status` `cancelled`) and the other is 422 with `errors.status` "An order cannot go from cancelled to cancelled."; the count of `38` read after the round is exactly 2 higher than the count read after the order was placed (never 4, never unchanged); at the end of the 20 rounds the count of `38` equals the starting 100 again (every order was cancelled once), and it was never above 100 at any reading.
Traces: SC-9 (ADMIN-016 item 2; story §3 item 3, concurrency)
Check: interact_page

**Scenario**: SC-10 — a shipped order and a delivered order cannot be cancelled
**Given**: FX-ORDER(shipped) `<s>` and FX-ORDER(delivered) `<d>`; the shelf read.
**When**: ADMIN sends `PUT {"status":"cancelled"}` to each.
**Then**: 422 with `errors.status` "An order cannot go from shipped to cancelled." for `<s>` and "An order cannot go from delivered to cancelled." for `<d>`; both orders keep their status and the shelf is unchanged.
Traces: SC-10 (ADMIN-016 item 2, AC-7; open question 2)
Check: fetch_url

**Scenario**: SC-11 — a paid order that is placed, confirmed or packed can be cancelled and stays paid
**Given**: Three orders placed with REQ (one left `placed`, one moved by `PUT` to `confirmed`, one to `packed`), each marked FX-PAID; the shelf read after placing each.
**When**: ADMIN sends `PUT {"status":"cancelled"}` to each, then `GET /v1/orders/<id>` as ADMIN, and `GET /v1/orders?status=cancelled&limit=100`.
**Then**: Each `PUT` answers 200 with `status` `cancelled` and `paid` true; each pair is back on the shelf (+ the order's `qty`); the reads show `status` `cancelled` and `paid` true; the three orders are in the cancelled list with `paid` true.
Traces: SC-11 (ADMIN-016 item 2, Conflict row; story §5 decision 6)
Check: fetch_url

## 8. Courier, customer details and the note — SC-12 … SC-16

**Scenario**: SC-12a — a shipped order takes carrier, tracking number and expected date, and the customer's read shows them; `""` clears each
**Given**: FX-ORDER(shipped) `<s>` placed with REQ (so `carrier`, `tracking`, `eta` start at `""`) and its `<token>`.
**When**: ADMIN sends, in turn: (1) `PUT {"carrier":"GHN","tracking":"GHN-PT27XM","eta":"2026-11-26"}`; (2) `PUT {"tracking":"GHN-PT27XX"}`; (3) `PUT {"eta":""}`; (4) `PUT {"carrier":"","tracking":""}`; after each, `GET /v1/orders/<s>` with `X-Order-Token: <token>` and no cookie.
**Then**: (1) answers 200 with `carrier` "GHN", `tracking` "GHN-PT27XM", `eta` "2026-11-26", and the token read shows the same three; (2) changes only `tracking` (carrier "GHN" and eta "2026-11-26" stay); (3) leaves `eta` `""` and carrier "GHN", tracking "GHN-PT27XX"; (4) leaves `carrier` `""` and `tracking` `""`; `status` stays `shipped` and the customer, phone, address and note do not change in any step.
Traces: SC-12 (ADMIN-016 item 3, AC-8)
Check: fetch_url

**Scenario**: SC-12b — a placed, a confirmed and a packed order take the same courier details
**Given**: Three orders made with FX-ORDER at `placed`, `confirmed` and `packed`, each with its `<token>`.
**When**: ADMIN sends `PUT {"carrier":"GHN","tracking":"GHN-PT27XM","eta":"2026-11-26"}` to each; then reads each with its token.
**Then**: Each `PUT` answers 200 with the three values; each token read shows the three values and the order's own status.
Traces: SC-12 (ADMIN-016 item 3)
Check: fetch_url

**Scenario**: SC-13a — over-long carrier or tracking and badly written dates are 422 naming the field, and nothing is saved
**Given**: A packed order `<id>` with `carrier` "GHN", `tracking` "GHN-1", `eta` "2026-11-27" (set by `PUT`); E200 / E201 as in the fixtures.
**When**: ADMIN sends `PUT` with: `{"carrier":E201}`; `{"tracking":E201}`; `{"eta":"26/11/2026"}`; `{"eta":"2026-13-40"}`; `{"eta":"2026-02-30"}`; `{"eta":"tomorrow"}`; and the two-fault body `{"carrier":E201,"eta":"tomorrow","note":"kept?"}`; then reads the order.
**Then**: Each of the first six answers 422 `{"error":"invalid",…,"errors":{<carrier|tracking|eta>:<non-empty>}}` naming that field; the seventh names both `carrier` and `eta`; the order still reads `carrier` "GHN", `tracking` "GHN-1", `eta` "2026-11-27" and its note is unchanged.
Traces: SC-13 (ADMIN-016 item 3; services §2.5)
Check: fetch_url

**Scenario**: SC-13b — a carrier and a tracking number of exactly 200 characters are accepted
**Given**: A placed order `<id>`.
**When**: ADMIN sends `PUT {"carrier":E200,"tracking":E200,"eta":"2026-12-01"}`; then reads the order.
**Then**: 200; the order holds the 200-character `carrier` and `tracking` and `eta` "2026-12-01".
Traces: SC-13 (ADMIN-016 item 3, boundary one inside the limit)
Check: fetch_url

**Scenario**: SC-13c — on a delivered order and a cancelled order, carrier, tracking and expected date are each refused
**Given**: FX-ORDER(delivered) `<d>` and FX-ORDER(cancelled) `<c>`, each given `carrier` "GHN", `tracking` "GHN-1", `eta` "2026-11-26" with `set_db_state` (`UPDATE orders SET carrier = 'GHN', tracking = 'GHN-1', eta = '2026-11-26' WHERE id = '<id>'`).
**When**: ADMIN sends to each order, one request per field: `{"carrier":"GHTK"}`, `{"tracking":"GHTK-2"}`, `{"eta":"2026-12-01"}`, and `{"eta":""}`.
**Then**: All eight answer 422 `{"error":"invalid",…,"errors":{<carrier|tracking|eta>:<non-empty>}}` naming the field sent; both orders still read "GHN", "GHN-1", "2026-11-26".
Traces: SC-13 (ADMIN-016 item 3, AC-14 reasoning for delivered and cancelled)
Check: fetch_url

**Scenario**: SC-14 — a packed order takes the customer's five details in one `PUT`, a later `PUT` of one field changes only that field, and an empty email is accepted
**Given**: A packed order `<p>` (customer "Lan Nguyen", phone "0912 345 678", address "12 Example Street, Ward 1", city "Da Nang", email `""`).
**When**: ADMIN sends (1) `PUT {"customer":"Tran Thi Binh","phone":"0987 654 321","email":"binh@example.com","address":"5 New Road, Ward 2","city":"Ha Noi"}`; (2) `PUT {"city":"Hue"}`; (3) `PUT {"email":""}`; after each, `GET /v1/orders/<p>` as ADMIN.
**Then**: (1) 200 and the dealer view shows exactly the five new values (the phone's digits `0987654321`); (2) 200 and only `city` changed to "Hue" (customer, phone, email, address as after (1)); (3) 200 and `email` is `""` with the rest as after (2); `status` stays `packed` and the order's pairs and total do not change.
Traces: SC-14 (ADMIN-016 item 5, AC-13)
Check: fetch_url

**Scenario**: SC-15a — an empty customer, a short phone, a bad email, a long address and a long note are each 422 naming the field
**Given**: A placed order `<id>` with the FORM values; E201, N5001 as in the fixtures.
**When**: ADMIN sends `PUT` with: `{"customer":""}`; `{"customer":"   "}`; `{"phone":"0912 345 67"}` (8 digits); `{"phone":""}`; `{"email":"abc"}`; `{"address":E201}`; `{"address":""}`; `{"city":""}`; `{"note":N5001}`; then the two-fault body `{"customer":"","email":"abc"}`; then reads the order.
**Then**: Each of the first nine answers 422 `{"error":"invalid",…,"errors":{<that field>:<non-empty>}}` naming exactly the field sent; the tenth names both `customer` and `email`; the order is unchanged.
Traces: SC-15 (ADMIN-016 item 5; services §2.5)
Check: fetch_url

**Scenario**: SC-15b — values on the inside of the limits are accepted: a 9-digit phone, a 200-character address, a 5,000-character note
**Given**: A placed order `<id>`; E200 and N5000 as in the fixtures.
**When**: ADMIN sends `PUT {"phone":"091234567"}`, `PUT {"address":E200}`, `PUT {"customer":<100 characters>,"city":<100 characters>}`, `PUT {"note":N5000}`; then reads the order.
**Then**: Each answers 200; the order holds the 9-digit phone, the 200-character address, the 100-character customer and city and the 5,000-character note, each exactly as sent (no trimming of interior characters, no truncation).
Traces: SC-15 (ADMIN-016 item 5; services §2.5 boundaries)
Check: fetch_url

**Scenario**: SC-16a — on a delivered order and a cancelled order, the five customer details are refused and the note still changes
**Given**: FX-ORDER(delivered) `<d>` and FX-ORDER(cancelled) `<c>` with the FORM values.
**When**: ADMIN sends to each order, one request per field: `{"customer":"Other"}`, `{"phone":"0987 654 321"}`, `{"email":"a@example.com"}`, `{"address":"1 Other Street"}`, `{"city":"Hue"}`; then `{"note":"Left at the door."}`; then `{"note":N5001}`; then reads both.
**Then**: The ten detail requests answer 422 `{"error":"invalid",…,"errors":{<field sent>:<non-empty>}}` (the `address` request names `address`); both orders still read the FORM values; the note request answers 200 and the note reads "Left at the door."; the N5001 note answers 422 naming `note` and the note stays "Left at the door.".
Traces: SC-16 (ADMIN-016 item 5, AC-14)
Check: fetch_url

## 9. What never changes, and payment — SC-17 … SC-19

**Scenario**: SC-17a — each field the dealer may never change is 422 naming it, even with the value the order already has
**Given**: A placed order `<id>`; its current dealer view read (`number`, `placed_at`, `product`, `size`, `qty`, `total`, `delivery_fee`, `payment`, `delivery`).
**When**: ADMIN sends one `PUT` per field: `{"number":"DOI-1"}`; `{"placed_at":"2026-11-01T00:00:00+07:00"}`; `{"product":"gazelle-indoor-solar-gold"}`; `{"size":"40"}`; `{"qty":3}`; `{"items":[]}`; `{"total":1}`; `{"delivery_fee":5000}`; `{"payment":"cod"}`; `{"delivery":"standard"}` (the value it already has); `{"total":3290000}` (the order's own total); then reads the order.
**Then**: Each of the eleven answers 422 `{"error":"invalid",…,"errors":{<the field sent>:<non-empty>}}`; the order is unchanged after every one.
Traces: SC-17 (ADMIN-016 item 5, AC-15; story §5 decision 2)
Check: fetch_url

**Scenario**: SC-17b — several refused fields name all of them, and a refused body applies nothing
**Given**: A placed order `<id>`.
**When**: ADMIN sends `PUT {"number":"DOI-1","total":1,"payment":"cod"}`; then `PUT {"status":"confirmed","total":1}`; then `PUT {"carrier":"GHN","note":"kept?","qty":2}`; then reads the order.
**Then**: The first answers 422 with `errors` holding `number`, `total` and `payment`; the second 422 with `errors.total` (and the order is still `placed`, not `confirmed`); the third 422 with `errors.qty` and the order's `carrier` is still `""` and `note` is unchanged.
Traces: SC-17 (ADMIN-016 item 5; story §5 decision 3)
Check: fetch_url

**Scenario**: SC-17c — an `id` in the body is ignored, a field the order does not have is 400, and a body that is not JSON is 400
**Given**: Two placed orders `<a>` and `<b>`.
**When**: ADMIN sends `PUT /v1/orders/<a>` with `{"id":"<b>","note":"Only A."}`; `PUT /v1/orders/<a>` with `{"colour":"red"}`; `PUT /v1/orders/<a>` with the text `not json` (`Content-Type: application/json`); then reads both orders.
**Then**: The first answers 200 with `id` `<a>` and note "Only A."; `<b>`'s note is unchanged; the second answers 400 `{"error":"bad_request",…,"errors":{"colour":<non-empty>}}` (not 422); the third answers 400 `{"error":"bad_request",…}`; `<a>` holds only the first change.
Traces: SC-17 (ADMIN-016 item 5; services §2.5 tolerated `id`, undefined field 400)
Check: fetch_url

**Scenario**: SC-18a — `paid` is refused for every way to pay, on unpaid and on paid orders, true and false
**Given**: Eight orders placed with REQ: for each way in `vnpay`, `cod`, `bank`, `momo` one unpaid order and one paid order (FX-WAY then FX-PAID); each read before.
**When**: ADMIN sends `PUT {"paid":true}` and then `PUT {"paid":false}` to each of the eight (sixteen requests); then reads each.
**Then**: All sixteen answer 422 `{"error":"invalid",…,"errors":{"paid":"Only the payment provider marks this order paid."}}` word for word; every order's `paid` is what it was before (the four unpaid still `false`, the four paid still `true`) and `payment` is unchanged.
Traces: SC-18 (Owner decision; ADMIN-016 item 4, AC-11, AC-12)
Check: fetch_url

**Scenario**: SC-18b — a body carrying `paid` or `transaction_no` applies nothing else
**Given**: A placed, unpaid order `<id>` with note "" and status `placed`.
**When**: ADMIN sends `PUT {"paid":true,"status":"confirmed","note":"Paid by hand."}`; then `PUT {"transaction_no":"14123456"}`; then `PUT {"paid":false,"transaction_no":""}`; then reads the order.
**Then**: The first answers 422 with `errors.paid` as above; the second 422 with `errors.transaction_no` non-empty; the third 422 with both `paid` and `transaction_no`; the order is still `placed`, unpaid, note `""`, `transaction_no` `""`.
Traces: SC-18 (ADMIN-016 item 4; story §3 item 6)
Check: fetch_url

**Scenario**: SC-19a — an unpaid order reads "not paid" with no transaction number, a paid sample reads paid, and no `PUT` changes either
**Given**: A placed, unpaid order `<u>`; the sample order `doi-24810` (`paid` true, `vnpay`, delivered) and the sample `doi-24816` (`paid` false, placed, `cod`).
**When**: ADMIN reads `GET /v1/orders/<u>`, `GET /v1/orders/doi-24810` and `GET /v1/orders/doi-24816`; sends `PUT {"paid":true}` and `PUT {"transaction_no":"1"}` to `<u>` and reads it again; reads `GET /v1/orders?payment=vnpay&limit=100`.
**Then**: `<u>` has `paid` false and `transaction_no` `""`; `doi-24810` has `paid` true; `doi-24816` has `paid` false and `transaction_no` `""`; for every sample order read, `transaction_no` is `""` unless a payment notice exists for it; both `PUT`s answer 422 and `<u>` reads as before; every item of the list carries a boolean `paid` and a string `transaction_no`.
Traces: SC-19 (ADMIN-016 item 4, AC-9, AC-10 — the unpaid half)
Check: fetch_url

**Scenario**: SC-19b — an order marked paid by VNPay's signed notice reads its transaction number, and no `PUT` changes it
**Given**: Plan item 8's notice address `GET /v1/payments/vnpay/ipn` exists and the run holds the sandbox hash secret; a placed order `<id>` with a payment attempt (`POST /v1/orders/<id>/payment` with its token) whose `reference` is `<ref>`. If the notice address answers 404 or the tables `payment_attempts` / `payment_notices` do not exist, this case is **UNVERIFIED** (no instrument can produce VNPay's signed notice without item 8's endpoint) and is reported so, not failed.
**When**: A signed success notice (`vnp_TxnRef=<ref>`, `vnp_Amount=<total × 100>`, `vnp_TransactionNo=14123456`, `vnp_ResponseCode=00`, `vnp_TransactionStatus=00`) is sent; ADMIN reads `GET /v1/orders/<id>` and the list; then sends `PUT {"paid":false}` and `PUT {"transaction_no":"1"}` and reads again.
**Then**: The notice answers `{"RspCode":"00",…}`; the dealer view reads `paid` true and `transaction_no` "14123456" (also in the list item); both `PUT`s answer 422 and the order reads `paid` true, "14123456" as before; the token read of the same order shows `paid` true and no `transaction_no` key.
Traces: SC-19 (ADMIN-016 item 4, AC-9)
Check: fetch_url

## 10. Deleting — SC-20 … SC-23

**Scenario**: SC-20 — a placed, confirmed, packed, shipped or delivered order cannot be deleted
**Given**: Five orders made with FX-ORDER, one for each of those statuses, each with its pair row and token; the shelf read.
**When**: ADMIN sends `DELETE /v1/orders/<id>` to each; then reads each order and counts its `order_items` and `order_tokens` rows with `set_db_state`.
**Then**: Each answers 409 `{"error":"conflict","message":"Cancel the order first."}` with no `errors` key; every order is still readable with its own status; each still has its 1 `order_items` row and its token row(s); the shelf is unchanged.
Traces: SC-20 (ADMIN-014 item 5, AC-7)
Check: fetch_url

**Scenario**: SC-21 — a cancelled, unpaid order is deleted with its pairs and tokens, the shelf stays as the cancel left it, and numbers are not reused
**Given**: `<id>` placed with REQ (token `<token>`) and cancelled with a real `PUT`; the shelf read after the cancel; the order's number `DOI-<n>`; `GET /v1/orders?limit=1` total T read; if `payment_attempts` exists, one attempt row of the order (`INSERT INTO payment_attempts (order_id, provider, reference, amount, expires_at) VALUES ('<id>','vnpay','DOI-<n>-1',3290000, now())`).
**When**: ADMIN sends `DELETE /v1/orders/<id>`; then `GET /v1/orders/<id>` as ADMIN and with `X-Order-Token: <token>`; counts the order's rows in `order_items`, `order_tokens` and (if it exists) `payment_attempts`; reads the shelf and the list total; sends `DELETE` again; places a new order with REQ.
**Then**: The `DELETE` answers 204 with an empty body; both `GET`s answer 404 `not_found` with one body; the three counts are 0 (the attempt row is gone; if that table does not exist that half is UNVERIFIED); the shelf equals the reading after the cancel (the delete returns nothing a second time); the list's `total` is T − 1 and the id is absent; the second `DELETE` answers 404; the new order's number is strictly higher than `DOI-<n>`.
Traces: SC-21 (ADMIN-014 item 5, AC-8; services §3.16)
Check: fetch_url

**Scenario**: SC-22 — a cancelled order that is also paid cannot be deleted
**Given**: `<id>` placed with REQ, marked FX-PAID, cancelled with a real `PUT` (200, `paid` true); if `payment_attempts` exists, one attempt row inserted as in SC-21; the shelf read.
**When**: ADMIN sends `DELETE /v1/orders/<id>`; then reads the order, counts its `order_items`, `order_tokens` and attempt rows, and reads the shelf.
**Then**: 409 `{"error":"conflict","message":"A paid order cannot be deleted; refund it in VNPay's portal first."}` with no `errors` key; the order still reads `cancelled` and `paid` true; the pair row, the token row(s) and the attempt row (if any) are all still there; the shelf is unchanged.
Traces: SC-22 (ADMIN-014 item 5; services open question 5)
Check: fetch_url

**Scenario**: SC-23a — an unknown id or an id that cannot be one is 404 for a dealer, and `PUT {}` changes nothing
**Given**: A placed order `<p>` and a cancelled order `<c>`, each read before.
**When**: ADMIN and then STAFF send `PUT {"note":"x"}` and `DELETE` to `/v1/orders/` followed by each of: `doi-99999`; `-x`; `a%20b`; `DOI-24817` (upper case is not an id the shop issues). Then ADMIN sends `PUT {}` to `<p>` and to `<c>`.
**Then**: Each of the 16 calls answers 404 `{"error":"not_found","message":<one and the same sentence>}` with no `errors` key (the `PUT` bodies are valid, so the id alone decides); sample order `doi-24817` still reads as before; both `PUT {}` answer 200 and both orders read exactly as before.
Traces: SC-23 (ADMIN-016 Failure table "Not found"; services §2.5 path ids)
Check: fetch_url

**Scenario**: SC-23b — a hundred sequential saves and a hundred sequential deletes each answer within 1 s at p95
**Given**: ADMIN signed in in a browser page; SHELF(`samba-og-w-core-black`, `38`, 500); one placed order `<id>` for the saves; 100 cancelled orders for the deletes, each placed with REQ and made `cancelled` with `UPDATE orders SET status = 'cancelled' WHERE id = '<id>'` (so the delete carries no stock return).
**When**: The page sends 100 sequential `PUT /v1/orders/<id>` (alternating `{"note":"a"}` and `{"note":"b"}`) and then 100 sequential `DELETE /v1/orders/<each>`, timing each call with `performance.now()`.
**Then**: Every `PUT` answers 200 and every `DELETE` 204; the 95th percentile (the 95th value of the sorted durations) of each series is at most 1000 ms.
Traces: SC-23 (ADMIN NFR Performance: a save or delete within 1 s at p95)
Check: interact_page

## 11. The list and the menu (browser) — SC-24 … SC-26

Two more fixtures serve this part and the next: **FX-TWO-UI** is an order of two lines with one pair each — shoe A `samba-og-w-core-black` size 38 × 1 and shoe B `gazelle-indoor-solar-gold` size 42 × 1 (so "<n> pairs" is 2 whether the sentence counts lines or pairs of shoes), customer `Lan Nguyen`, city `Da Nang`. **BROWSER-ORDER** is an order placed in the browser itself: the bag `agentteam:cart` is written with one line of `samba-og-w-core-black`, variant `EU 38`, qty 1, the checkout form is filled with the FORM values (`Lan Nguyen`, `0912 345 678`, `12 Example Street, Ward 1`, `Da Nang`, `Call before delivering.`) and `[data-testid='place-order']` is pressed in a browser with no session, so that browser's own storage holds the order's token; if the browser is sent to VNPay's page the run navigates to `/orders/<id>` instead.

**Scenario**: SC-24a — the orders list: title, no "Add", search, columns and the two filters
**Given**: ADMIN signed in in a browser at 1280×800; the shop holds the 14 sample orders and BROWSER-ORDER `<new>`; `GET /api/v1/payments` read for the names of the ways to pay.
**When**: The browser opens `/admin/orders` and reads the page.
**Then**: `main h1` reads "Orders"; there is no `button` or link whose text is "Add" (or begins "Add "); a search field has the placeholder "Search by number or customer"; the table's `th` cells read, in order, the labels of number, placed, status, shoe, total, payment, customer (the schema's "Order number", "Placed", "Status", "Shoe", "Total", "Payment", "Customer"; the first may read "Number"); a filter named "Status" offers exactly Placed, Confirmed, Packed, On its way, Delivered, Cancelled; a filter named "Payment" offers exactly the names `GET /api/v1/payments` returned; with no sort chosen the first `tbody tr` is `<new>` and the first column of each row reads `DOI-<n>` in descending placed order (the first row's number equals `items[0].number` of `GET /api/v1/orders?sort=-placed_at&limit=1` as ADMIN).
Traces: SC-24 (ADMIN-011 AC-12, AC-14, items 2–3)
Check: interact_page

**Scenario**: SC-24b — an order placed through checkout is on the next load, and a row opens its page
**Given**: ADMIN signed in in a second browser context with `/admin/orders` open; BROWSER-ORDER `<new>` is placed after that page loaded.
**When**: The run reloads `/admin/orders`; reads the first row; clicks it.
**Then**: Before the reload the first row is not `<new>` (the page is read from the shop, not pushed); after the reload the first row shows `<new>`'s number, status "Placed", customer "Lan Nguyen", total `3.290.000 ₫` and payment "VNPay"; clicking the row changes the address to `/admin/orders/<new>` and `main h1` is the order's heading (not "Orders").
Traces: SC-24 (ADMIN-011 item 2, AC-14)
Check: interact_page

**Scenario**: SC-25a — the "shoe" column shows the first pair's shoe once
**Given**: FX-TWO-UI `<two>` and the sample orders `doi-24817` (one pair, `samba-og-w-crystal-white`) and `doi-24812` (one line of 2 pairs, `samba-og-w-wonder-quartz`); the shoe names read from `GET /api/v1/products/<id>`.
**When**: ADMIN opens `/admin/orders` and reads the rows of `<two>`, `doi-24817` and `doi-24812`.
**Then**: The `<two>` row's shoe cell reads shoe A's name once and does not contain shoe B's name; the `doi-24817` row's shoe cell reads the name of `samba-og-w-crystal-white` once; the `doi-24812` row's reads the name of `samba-og-w-wonder-quartz` once (not "2 ×" repeated, not the shoe's id).
Traces: SC-25 (ADMIN-016 item 7, AC-20; story §5 decision 10)
Check: interact_page

**Scenario**: SC-25b — the Status and Payment filters and the search narrow the list to the right orders
**Given**: ADMIN on `/admin/orders`; the API totals read: `GET /api/v1/orders?status=shipped&limit=1`, `GET /api/v1/orders?payment=vnpay&limit=1`; a placed order `<new>` for customer "Lan Nguyen".
**When**: The run chooses Status "On its way"; reads the rows; clears it and chooses Payment "VNPay"; reads the rows; clears it; types `24817` into the search; reads the rows; replaces it with `Nguyen`; reads the rows.
**Then**: With "On its way" every row's status cell reads "On its way" and the number of rows (over all pages) equals the API's `total` for `status=shipped` (at least the 3 shipped samples); with "VNPay" every row's payment cell reads "VNPay" and the count equals the API's `total` for `payment=vnpay`; the search for `24817` leaves exactly one row, `DOI-24817`; the search for `Nguyen` includes `<new>`'s row and no row whose customer lacks "Nguyen".
Traces: SC-25 (ADMIN-016 item 7, AC-20; ADMIN-011 AC-14, item 2)
Check: interact_page

**Scenario**: SC-26a — the menu's Orders item counts the orders waiting, on every admin page, and shows none when none wait
**Given**: Only the 14 sample orders exist: `doi-24816` is placed and `doi-24815` is confirmed, so 2 wait; ADMIN signed in at 1280×800.
**When**: The run opens `/admin`, `/admin/orders`, `/admin/products` and `/admin/lines` and reads the `nav` link whose `href` is `/admin/orders`; then `UPDATE orders SET status = 'packed' WHERE id IN ('doi-24816','doi-24815')` through `set_db_state`, reloads `/admin/orders` and reads the link again; then the two rows are put back (`placed` and `confirmed`).
**Then**: On all four pages the link's text is "Orders" with the count `2` shown beside it; after the change no count is shown (the link's text is exactly "Orders", no digit); after the rows are put back the count `2` is shown again.
Traces: SC-26 (ADMIN-011 AC-15)
Check: interact_page

**Scenario**: SC-26b — cancelled, packed, shipped and delivered orders are not counted, and saving an order as packed lowers the count by one
**Given**: The two waiting samples (count 2); FX-ORDER(cancelled), FX-ORDER(packed), FX-ORDER(shipped), FX-ORDER(delivered), one each; then `<w>` placed with REQ (count becomes 3).
**When**: The run reads the menu's Orders count after the four extra orders exist; after `<w>` is placed; then opens `/admin/orders/<w>`, chooses Confirmed and presses "Save changes", reads the count after a reload; then chooses Packed, presses "Save changes", and reads the count after a reload.
**Then**: The count is 2 with the four non-waiting orders added and 3 after `<w>`; after `<w>` is saved as Confirmed the count after a reload is still 3; after it is saved as Packed the count after a reload is 2.
Traces: SC-26 (ADMIN-011 AC-15)
Check: interact_page

## 12. The order page (browser) — SC-27 … SC-30

**Scenario**: SC-27a — the tabs Order, Delivery and Customer, and the Customer tab shows the stored values
**Given**: ADMIN signed in at 1280×800; the sample order `doi-24817` (customer "Hoàng Nam", phone "0938 410 552", email `""`, address "42 Example Street, Hải Châu", city "Đà Nẵng", note "Gift — no price inside the box."); a second order `<id>` placed with REQ and given `email` `lan@example.com` by `PUT`.
**When**: The run opens `/admin/orders/doi-24817`, reads the tabs, clicks "Customer" and reads each field's value; then the same for `<id>`.
**Then**: Exactly three `[role=tab]` read, in order, "Order", "Delivery", "Customer"; on the Customer tab the fields labelled "Customer", "Phone", "Email", "Address", "City or province" and "Internal note" hold, for `doi-24817`: "Hoàng Nam", "0938 410 552", "" , "42 Example Street, Hải Châu", "Đà Nẵng", "Gift — no price inside the box."; for `<id>`: "Lan Nguyen", "0912 345 678", "lan@example.com", "12 Example Street, Ward 1", "Da Nang", "Call before delivering." — each equal to the dealer API's value for that field.
Traces: SC-27 (ADMIN-016 item 8, AC-18; ADMIN-011 AC-20)
Check: interact_page

**Scenario**: SC-27b — the customer's own order page and the tracking page show none of the private fields, in the page and in the network answers
**Given**: BROWSER-ORDER `<new>` (the browser holds its token; phone "0912 345 678", address "12 Example Street, Ward 1", note "Call before delivering."); the dealer has set `email` `lan@example.com` and carrier `GHN` by `PUT`; no dealer session in this browser.
**When**: The run opens `/orders/<new>` and records every `/api/v1/orders/…` answer's text and the page's visible text; then opens `/track`, types the order's number and phone `0912 345 678`, submits, and records the page text and every network answer.
**Then**: Neither page's visible text contains "0912", "345 678", "Example Street", "Call before delivering" or "lan@example.com"; no recorded network answer contains a `phone`, `email`, `address`, `note` or `transaction_no` key, or any of those values; the order page does show "GHN" (the dealer's carrier) and the order's number.
Traces: SC-27 (ADMIN-016 AC-18, ADMIN-011 item 10; ORD-008 item 3)
Check: interact_page

**Scenario**: SC-28 — an order of one pair shows the shoe's picture and the design's sentence, with no list of pairs
**Given**: ADMIN signed in; `<id>` placed with REQ (Lan Nguyen, Da Nang, 1 × `samba-og-w-core-black`, EU 38, total 3.290.000 ₫); the shoe's name `N` and `image` read from `GET /api/v1/products/samba-og-w-core-black`.
**When**: The run opens `/admin/orders/<id>` and reads the picture and the sentence.
**Then**: An `img` near the top has `src` equal to the shoe's `image`, a non-empty `alt` and `naturalWidth` above 0; a paragraph reads exactly "Lan Nguyen in Da Nang ordered 1 × N, EU 38, on DATE(placed_at) — 3.290.000 ₫."; no `li` and no `tr` inside `main` contains the text "EU 38" (there is no list of pairs) and the page does not contain "ordered 1 pairs" or "ordered 1 pair ".
Traces: SC-28 (ADMIN-016 AC-17, item 6)
Check: interact_page

**Scenario**: SC-29a — an order of two pairs shows the multi-pair sentence and one row for each pair, in the order sent
**Given**: ADMIN signed in; FX-TWO-UI `<two>` (A 38 × 1 at price PA, B 42 × 1 at price PB); the dealer API's `items` read.
**When**: The run opens `/admin/orders/<two>` and reads the sentence and the rows of the pairs.
**Then**: A paragraph reads exactly "Lan Nguyen in Da Nang ordered 2 pairs on DATE(placed_at) — <total>." with `<total>` the order's total in `vi-VN` money; under it there are exactly 2 pair rows, in this order: first A's picture (`src` = `items[0].image`), A's name, "EU 38", the quantity 1 and A's unit price; then B's picture, B's name, "EU 42", quantity 1, B's unit price; PA + PB equals the total; the one-pair sentence form "ordered 1 ×" is absent.
Traces: SC-29 (ADMIN-016 AC-16, item 6)
Check: interact_page

**Scenario**: SC-29b — a pair whose shoe has gone to draft still shows its name and picture; quantities times unit prices add up to the total
**Given**: FX-TWO `<two2>` (A 38 × 1, B 42 × 2); `UPDATE products SET status = 'draft' WHERE id = 'gazelle-indoor-solar-gold'` through `set_db_state` (put back afterwards); the dealer API's `items` read.
**When**: ADMIN opens `/admin/orders/<two2>` and reads the two pair rows and the sum.
**Then**: Row 2 still reads B's name (not its id, not blank) and shows its picture (`naturalWidth` above 0), "EU 42", quantity 2 and B's unit price (the unit price, not the line total); the numbers shown, unit price × quantity over the two rows, add up to the order's total; the page shows no error text.
Traces: SC-29 (ADMIN-016 item 6; SRS open question 4; ADMIN-014 AC-2)
Check: interact_page

**Scenario**: SC-30a — way to pay, "Paid" or "Awaiting payment", and VNPay's transaction number are plain text
**Given**: ADMIN signed in; `<u>` placed with REQ (vnpay, unpaid); the sample `doi-24810` (paid, `vnpay`); `<t>` an order marked paid with a transaction `14123456` (an attempt row and a success notice row made with `set_db_state`: `INSERT INTO payment_attempts (order_id, provider, reference, amount, expires_at) VALUES ('<t>','vnpay','DOI-<n>-1',<total>, now())` and `INSERT INTO payment_notices (attempt_id, transaction_no, amount, response_code, transaction_status, is_success) SELECT id, '14123456', amount, '00', '00', true FROM payment_attempts WHERE reference = 'DOI-<n>-1'`, then `UPDATE orders SET paid = true`); if those tables do not exist the `<t>` half is **UNVERIFIED** (no instrument can make VNPay's notice without item 8) and is reported so.
**When**: The run opens each order's page, clicks the "Order" tab and reads the text of the read-only block.
**Then**: For `<u>`: the block shows the way to pay "VNPay" and "Awaiting payment", and the text "VNPay transaction" is absent; for `doi-24810`: "Paid" and no "VNPay transaction"; for `<t>`: "Paid" and "VNPay transaction 14123456". In every case the order's number and placed time (DATE) and the total are shown as text.
Traces: SC-30 (ADMIN-016 item 4, AC-9, AC-10)
Check: interact_page

**Scenario**: SC-30b — the only form controls on any tab are the ten editable fields
**Given**: ADMIN signed in; a placed order `<id>` (vnpay, unpaid); a paid sample `doi-24810`.
**When**: For each of the two orders the run clicks "Order", "Delivery" and "Customer" in turn and lists every visible `input`, `select`, `textarea`, `[role=switch]`, `[role=checkbox]`, `[role=combobox]` inside `main` with its accessible label.
**Then**: Across the three tabs the labelled controls are exactly: Status; Carrier, Tracking number, Expected; Customer, Phone, Email, Address, City or province, Internal note — ten, and no other; there is no control labelled or named number, placed, shoe, size, pairs, total, payment, paid, delivery option or transaction; the only `button`s besides the three tabs are "Delete", "Save changes" and a notice's close button.
Traces: SC-30 (ADMIN-016 item 4, AC-9, AC-10, item 5)
Check: interact_page

## 13. Saving and deleting from the page (browser) — SC-31 … SC-38

In every case of this part the browser is signed in as ADMIN at 1280×800 unless a case says otherwise, and "the request" is the `PUT` or `DELETE` the page sends, recorded by the run's request interception (the request itself is let through unless the case says it is held, failed or answered). "Shows X beside Status" means X is the text of the element the Status control's `aria-describedby` names, the control has `aria-invalid="true"`, and the text sits in a `role="alert"` or `aria-live` element.

**Scenario**: SC-31a — a placed order is moved to Confirmed from the page; "Saved." shows; only editable fields are sent
**Given**: `<id>` placed with REQ (status `placed`); `/admin/orders/<id>` open on the "Order" tab.
**When**: The run reads the Status control's options, chooses "Confirmed", presses "Save changes", reads the notice and the recorded request, reloads, and opens `/admin/orders`.
**Then**: The control offers exactly "Placed" (selected), "Confirmed" and "Cancelled"; after the press a `PUT /api/v1/orders/<id>` is recorded whose JSON body has only keys from {`status`, `carrier`, `tracking`, `eta`, `customer`, `phone`, `email`, `address`, `city`, `note`} and `status` is `confirmed` — it has none of `number`, `placed_at`, `product`, `size`, `qty`, `items`, `total`, `delivery_fee`, `payment`, `paid`, `delivery`, `transaction_no`; the page shows "Saved." (announced); after the reload the control reads "Confirmed" and the list's row for `<id>` reads "Confirmed"; the dealer API reads `confirmed`.
Traces: SC-31 (ADMIN-016 item 1, AC-1; ADMIN-011 items 4–5)
Check: interact_page

**Scenario**: SC-31b — the status control offers only the moves the shop allows from each status
**Given**: Orders made with FX-ORDER at `confirmed`, `packed` and `shipped`.
**When**: The run opens each order's page on the "Order" tab and reads the Status control's options and the selected one.
**Then**: Confirmed: "Confirmed" (selected), "Packed", "Cancelled". Packed: "Packed" (selected), "On its way", "Cancelled". Shipped: "On its way" (selected) and "Delivered" only — there is no "Cancelled" option. In no case is a step back, a skip or a stored word ("shipped") offered.
Traces: SC-31 (ADMIN-016 item 1; story §5 decision 4)
Check: interact_page

**Scenario**: SC-32 — a stale page that sends a move the shop refuses shows the shop's sentence beside Status and keeps the typed values
**Given**: `<c>` is FX-ORDER(confirmed); two browser windows W1 and W2 open on `/admin/orders/<c>`, both showing "Confirmed".
**When**: In W1 the run chooses "Cancelled" and presses "Save changes" ("Saved." shows). In W2 (still showing "Confirmed") the run clicks "Customer", types `Stale try.` into "Internal note", clicks "Order", chooses "Packed" and presses "Save changes".
**Then**: In W2 the text "An order cannot go from cancelled to packed." shows beside Status (marked invalid, announced); the note field still holds `Stale try.` and the Status control still holds the chosen "Packed"; the text "We could not save. Please try again." is absent and the button reads "Save changes" and is enabled; the dealer API reads `cancelled` and a note that is not `Stale try.` (a refused body applied nothing). Other refusals the shop adds for fields the page also sent may show beside their own fields; this case pins only Status's sentence.
Traces: SC-32 (ADMIN-016 AC-2, AC-3, AC-7; ADMIN-011 item 6)
Check: interact_page

**Scenario**: SC-33a — cancelling from the page returns the pairs, and the order is then drawn locked except for its note
**Given**: SHELF(`samba-og-w-core-black`, `38`, 2); `<id>` placed with REQ and `qty` 2 (the size `38` is sold out); the shoe page `/shop/samba-og-w-core-black` shows the `38` pill (the `[data-testid^='size-']` whose text is "38") disabled or `aria-disabled`; `/admin/orders/<id>` open.
**When**: The run chooses "Cancelled", presses "Save changes", reads the page; clicks "Delivery" and "Customer" and reads each field's state; types `Parcel came back.` into "Internal note" and saves; reloads; then opens `/shop/samba-og-w-core-black` in a visitor browser.
**Then**: "Saved." shows after the cancel; the page reads "Cancelled" (the status text) and the Status control is not an editable control (no enabled `select`, not focusable as an input); on "Delivery" the fields Carrier, Tracking number and Expected are read-only or disabled; on "Customer" Customer, Phone, Email, Address and City or province are read-only or disabled and "Internal note" is enabled and editable; the note save shows "Saved." and the reload holds `Parcel came back.`; the visitor's shoe page now shows the `38` pill enabled (not disabled, not struck through).
Traces: SC-33 (ADMIN-016 AC-5, AC-14, items 2, 3, 5)
Check: interact_page

**Scenario**: SC-33b — a delivered order is locked the same way
**Given**: FX-ORDER(delivered) `<d>` with the FORM values; its page open.
**When**: The run reads the Status control, the Delivery fields and the Customer fields, then types `Signed for.` into "Internal note", saves and reloads.
**Then**: The status reads "Delivered" and is not editable; Carrier, Tracking number, Expected, Customer, Phone, Email, Address and City or province are read-only or disabled; the note saves ("Saved.") and the reload holds `Signed for.`.
Traces: SC-33 (ADMIN-016 AC-14, items 3, 5)
Check: interact_page

**Scenario**: SC-34a — courier details typed on the Delivery tab are saved and the customer's order page shows them
**Given**: BROWSER-ORDER `<new>` placed in a customer browser (holding the token) and moved to `shipped` by `PUT`; the dealer's browser on `/admin/orders/<new>`.
**When**: The dealer clicks "Delivery"; the run reads the hint under Expected; types `GHN` into "Carrier", `GHN-PT27XM` into "Tracking number" and `2026-11-26` into "Expected"; presses "Save changes"; reloads and reads the three fields. Then the customer browser reloads `/orders/<new>` and reads the page.
**Then**: The hint "Year-month-day, for example 2026-11-26" is visible on the Delivery tab (tied to the Expected field); "Saved." shows; after the dealer's reload the three fields hold `GHN`, `GHN-PT27XM`, `2026-11-26`; the customer's page text contains "GHN", "GHN-PT27XM" and the date 26 November 2026 in the page's own date style (a text containing 26, 2026 and 11 or Nov); the customer's page still shows no phone, email, address or note.
Traces: SC-34 (ADMIN-016 item 3, AC-8)
Check: interact_page

**Scenario**: SC-34b — an expected date written `26/11/2026` leaves the page open with the shop's sentence beside Expected and every typed value kept
**Given**: A packed order `<id>` (carrier, tracking and eta `""`); its page on the "Delivery" tab.
**When**: The run types `GHN` into "Carrier", `GHN-PT27XM` into "Tracking number", `26/11/2026` into "Expected", presses "Save changes", reads the page, then reads the dealer API.
**Then**: A non-empty sentence shows beside Expected (marked invalid, announced) — the shop's words from `errors.eta`; "Saved." is absent; the three fields still hold `GHN`, `GHN-PT27XM`, `26/11/2026`; the page is still `/admin/orders/<id>`; the API's `carrier`, `tracking` and `eta` are all `""`. Correcting Expected to `2026-11-26` and pressing "Save changes" then shows "Saved." and the API holds the three values.
Traces: SC-34 (ADMIN-016 item 3; story §5 decision 9; ADMIN-011 item 6)
Check: interact_page

**Scenario**: SC-35a — the address and the note of a packed order are changed on the Customer tab and saved
**Given**: A packed order `<p>` with the FORM values; its page open on the "Customer" tab.
**When**: The run replaces "Address" with `5 New Road, Ward 2` and "Internal note" with `Ring twice.`, presses "Save changes", reads the page, reloads and reads both fields.
**Then**: "Saved." shows (announced); after the reload "Address" holds `5 New Road, Ward 2` and "Internal note" holds `Ring twice.`; the dealer API reads the same two values and the order is still `packed` with its phone and city unchanged.
Traces: SC-35 (ADMIN-016 item 5, AC-13)
Check: interact_page

**Scenario**: SC-35b — an email `abc` leaves the page open with the shop's sentence beside Email, every typed value kept and nothing saved
**Given**: A placed order `<id>` with the FORM values; the "Customer" tab open.
**When**: The run types `abc` into "Email" and `9 Other Lane` into "Address", presses "Save changes", reads the page and the API; then types `lan@example.com` into "Email" and presses "Save changes" again.
**Then**: After the first press a non-empty sentence shows beside Email (marked invalid with `aria-invalid="true"`, announced); "Saved." is absent; "Email" still holds `abc` and "Address" still holds `9 Other Lane`; the API's `email` is `""` and `address` is "12 Example Street, Ward 1" (nothing saved); after the second press "Saved." shows and the API holds `lan@example.com` and `9 Other Lane`.
Traces: SC-35 (ADMIN-016 item 5, AC-13; ADMIN-011 item 6)
Check: interact_page

**Scenario**: SC-36a — with the shop unreachable, "Save changes" shows "We could not save. Please try again." and keeps the typed values
**Given**: A placed order `<id>`; its page on the "Customer" tab; the run's interception aborts every `PUT /api/v1/orders/<id>` (a network failure); a second run answers the same request with a 502.
**When**: The run types `7 Typed Lane` into "Address" and `Typed first.` into "Internal note", presses "Save changes", reads the page; then lets the request through and presses "Save changes" again.
**Then**: In both runs the text "We could not save. Please try again." shows (announced); "Address" holds `7 Typed Lane` and "Internal note" holds `Typed first.`; the button reads "Save changes" and is enabled (it is not left on "Saving…"); the API still holds the old address; when the request goes through the second press shows "Saved." and the API holds the typed values.
Traces: SC-36 (ADMIN-011 item 7, Failure rows)
Check: interact_page

**Scenario**: SC-36b — after the dealer's session has ended, "Save changes" shows "Your session ended. Sign in again." with a link to the sign-in page, and the typed values stay
**Given**: A placed order `<id>`; its page open; a second tab of the same browser; or, if one tab only is available, `DELETE FROM sessions WHERE user_id = (SELECT id FROM users WHERE email = '<ADMIN_EMAIL lower-cased>')` through `set_db_state` after the page has loaded.
**When**: In the second tab the run sends `POST /api/v1/auth/sign-out` (the cookie is cleared); back in the first tab it types `Typed after.` into "Internal note" and presses "Save changes".
**Then**: The save is answered 401 and the page shows "Your session ended. Sign in again." (announced) with a link whose `href` is `/admin/sign-in`; "Internal note" still holds `Typed after.`; the button is enabled; the API (read with a new sign-in) holds the old note.
Traces: SC-36 (ADMIN-011 "Not permitted" row)
Check: interact_page

**Scenario**: SC-37a — "Delete" asks "Delete <title>?" on the order page and on a list row's menu, and "Cancel" sends no request
**Given**: A placed order `<id>`; the order page `/admin/orders/<id>` and `/admin/orders` open in turn.
**When**: On the order page the run reads `main h1`, presses "Delete", reads the question, presses "Cancel", reads the page; then on `/admin/orders` opens `<id>`'s row menu, presses "Delete", reads the question, presses "Cancel".
**Then**: Each question is a `[role=dialog]` or `[role=alertdialog]` titled "Delete T?" where T is exactly the text of the page's `main h1` for that order, with the buttons "Cancel" and a confirm button; after "Cancel" the question is gone, no `DELETE` request was recorded in either surface, and the dealer API still reads the order.
Traces: SC-37 (ADMIN-014 AC-7, AC-8, item 6; ADMIN-011 item 8)
Check: interact_page

**Scenario**: SC-37b — confirming on a placed order leaves the question open with "Cancel the order first."
**Given**: A placed order `<id>`; its page open.
**When**: The run presses "Delete", then the confirm button; reads the question and the address; repeats from the list row's menu.
**Then**: The question stays open and shows "Cancel the order first." inside it (the 409's sentence); the address is still `/admin/orders/<id>`; the order is still `placed` in the API; from the list the question shows the same sentence and the row is still in the table.
Traces: SC-37 (ADMIN-014 AC-7; ADMIN-011 item 8)
Check: interact_page

**Scenario**: SC-37c — confirming on a cancelled, paid order shows the paid sentence and keeps the order
**Given**: An order cancelled by a real `PUT` and marked FX-PAID; its page open.
**When**: The run presses "Delete", then the confirm button, and reads the question.
**Then**: The question stays open and shows "A paid order cannot be deleted; refund it in VNPay's portal first." word for word; the API still reads the order, `cancelled` and `paid` true.
Traces: SC-37 (ADMIN-014 item 5; services open question 5)
Check: interact_page

**Scenario**: SC-37d — confirming on a cancelled, unpaid order deletes it and returns to the list
**Given**: Two cancelled, unpaid orders `<a>` and `<b>`; `<a>`'s page open; the list open for `<b>`.
**When**: On `<a>`'s page the run presses "Delete" and the confirm button, reads the address, reloads `/admin/orders`; then from the list opens `<b>`'s row menu, presses "Delete" and confirms.
**Then**: After the confirm on the page the request `DELETE /api/v1/orders/<a>` is answered 204 and the browser lands on `/admin/orders`; after a reload `<a>` is not in the list; from the list `<b>`'s row disappears and stays gone after a reload; `GET /api/v1/orders/<a>` and `/<b>` as ADMIN answer 404.
Traces: SC-37 (ADMIN-014 AC-8)
Check: interact_page

**Scenario**: SC-38a — an order id that does not exist, and an order that was just deleted, show "This record is not here"
**Given**: A cancelled, unpaid order `<gone>` whose page is open in tab T1; the same order deleted by `DELETE /v1/orders/<gone>` in another tab.
**When**: The run opens `/admin/orders/nope`, reads the page, presses "Back to the list"; then reloads T1's page of `<gone>` and reads it.
**Then**: Both pages show "This record is not here", "It may have been deleted since the link was made." and a button "Back to the list"; pressing it goes to `/admin/orders`.
Traces: SC-38 (ADMIN-011 item 9, AC-10)
Check: interact_page

**Scenario**: SC-38b — while an order is being fetched, "This record is not here" is not shown
**Given**: A placed order `<id>` and the run's interception holds `GET /api/v1/orders/<id>` for 2 seconds; a second run holds `GET /api/v1/orders/nope` for 2 seconds.
**When**: The run opens `/admin/orders/<id>` and reads the page at 500 ms and at 3 s; then opens `/admin/orders/nope` and reads it at 500 ms and at 3 s.
**Then**: For `<id>` the text "This record is not here" is absent at 500 ms and at 3 s, and at 3 s the order's sentence shows; for `nope` the text is absent at 500 ms and present at 3 s (after the shop's 404).
Traces: SC-38 (ADMIN-011 item 9, AC-11)
Check: interact_page

## 14. Phone layout and hygiene — SC-39, SC-40

**Scenario**: SC-39a — at 390 px and 320 px the list and the order page do not scroll sideways and every control is reachable
**Given**: ADMIN signed in; a window of 390×844 and then 320×640; `<one>` a one-pair order, `<two>` FX-TWO-UI.
**When**: The run opens `/admin/orders`, then `/admin/orders/<one>` and `/admin/orders/<two>`; reads `document.documentElement.scrollWidth` and `window.innerWidth`; reads the bounding box of each tab, of the Status control, of "Save changes" and of each pair row; reads, for each form field on each tab, its `label`; reads the nearest scrollable ancestor of the orders table.
**Then**: On all three pages at both sizes `scrollWidth` is at most `innerWidth`; if the list's table is wider than the window it sits in its own container with horizontal scrolling (the container's `scrollWidth` is above its `clientWidth` and the page's is not); each of the three tabs, the Status control and "Save changes" has a box inside the window's width and `document.elementFromPoint` at its centre returns the control (nothing covers it); each pair row of `<two>` has `scrollWidth` at most its `clientWidth` (the text wraps, it is not cut); every field on every tab has a visible `label` (non-empty text, not hidden) tied to it.
Traces: SC-39 (SRS Responsive and Accessibility NFRs; ADMIN-016 item 6)
Check: interact_page

**Scenario**: SC-39b — field errors, "Saving…" and "Saved." are announced to assistive technology
**Given**: A packed order `<id>`; its page at 390×844; a held (2 s) `PUT`.
**When**: The run types `abc` into "Email", presses "Save changes" and, while the request is held, reads the button and the live regions; after the answer reads them again; then fixes the email and saves.
**Then**: While held the button reads "Saving…" and is disabled and the text "Saving…" is in or under an `aria-live` or `role="status"` region; the 422's sentence sits in a `role="alert"` or `aria-live` element and Email has `aria-invalid="true"`; "Saved." is in an `aria-live` or `role="status"` element.
Traces: SC-39 (SRS Accessibility NFR; ADMIN-011 item 5)
Check: interact_page

**Scenario**: SC-40a — the order screens use no browser-memory mock: nothing is written to browser storage and the requests are the shop's
**Given**: ADMIN signed in; the browser's `localStorage` and `sessionStorage` key lists read before; a placed order `<id>`.
**When**: The run opens `/admin/orders`, opens `<id>`, changes the note and saves, then cancels the order from the page, then deletes it from the list; records every request to `/api/v1/orders…`; reads the storage key lists and values again.
**Then**: No key is added to either storage, none is named `doi:orders` (the design's old mock key) and no stored value contains `<id>`, the order's number, "0912" or "Example Street"; the recorded requests include `GET /api/v1/orders` (the list), `GET /api/v1/orders/<id>`, `PUT /api/v1/orders/<id>` twice and `DELETE /api/v1/orders/<id>`, each answered by the shop (a status from the server, not a stub), and no request goes to any other address for orders.
Traces: SC-40 (Overview §5; ADMIN-011 item 1)
Check: interact_page

**Scenario**: SC-40b — the repository and build rules of the story hold
**Given**: The story's two pull requests at their heads (the UI PR and the backend PR).
**When**: A reader lists each PR's changed files, searches `code/frontend/src/pages/AdminOrders.tsx` and `AdminOrder.tsx` for imports, and runs the project's checks.
**Then**: After the backend PR no file under `src/lib/mock/` is imported by `AdminOrders.tsx` or `AdminOrder.tsx`, and `src/lib/mock/orders.ts` still exists (the Track page); the page's client is `src/lib/dealer-handles-orders-status-courier-payment.ts` built on `src/lib/api.ts`, there is no second file with `api` in its name and no source line writes `localStorage` for orders; neither PR edits `store.ts`, `ListPage.tsx`, `RecordPage.tsx`, `Form.tsx`, `fields.tsx`, `content.json`, `schema.json` or `theme.css`; `npm run lint`, `npm run build`, `npm test`, `go build ./...`, `go vet ./...` and `go test ./...` pass. The verifier answers the build half with `run_tests` / `check_build` and the file rules from the PR's file list and a repository search, and reports what each said.
Traces: SC-40 (Overview §5; ADMIN-011 item 1)
Check: manual — these are rules about a diff and a repository, which no HTTP or browser reading shows; the verifier's repository and build tools may still answer them, and whatever they say is recorded.

## 15. Contract cases — `GET /v1/orders`, `PUT /v1/orders/{id}`, `DELETE /v1/orders/{id}` (services §2.1, §2.3, §2.4, §2.5, §3.14 – §3.16)

**Scenario**: C-1 — a `PUT` with another content type is 415, after the guard and before the body is read
**Given**: A placed order `<id>`.
**When**: ADMIN sends `PUT /v1/orders/<id>` with `Content-Type: text/plain` and the body `{"note":"x"}`; `NONE` sends the same; `CUSTOMER` sends the same.
**Then**: ADMIN's answer is 415 `{"error":"unsupported_media_type","message":<non-empty>}` with no `errors` key and the note is unchanged; `NONE`'s is 401 `unauthenticated`; `CUSTOMER`'s is 403 `forbidden` (the guard answers first).
Traces: contract (§2.1 content type; §3.15; ADMIN-004)
Check: fetch_url

**Scenario**: C-2 — a `PUT` body over 64 KiB is 413 and changes nothing
**Given**: A placed order `<id>`.
**When**: ADMIN sends `PUT /v1/orders/<id>` with `{"note":"<70,000 times a>"}` (a body of about 70 KB, over the 65,536-byte cap).
**Then**: 413 `{"error":"payload_too_large","message":<non-empty>}` with no `errors` key (not the 422 a 5,001-character note gets); the note is unchanged.
Traces: contract (§2.5 body cap 64 KiB; §2.3 `payload_too_large`)
Check: fetch_url

**Scenario**: C-3 — an undefined field, a wrongly typed field and a body that is not a JSON object are 400 and apply nothing
**Given**: A placed order `<id>` with note `""`.
**When**: ADMIN sends `PUT /v1/orders/<id>` with, in turn: `{"nickname":"x"}`; `{"note":"kept?","nickname":"x"}`; `{"carrier":5}`; `{"eta":20261126}`; `{"status":["confirmed"]}`; `{"note":true}`; `[]`; `"text"`; an empty body; then reads the order.
**Then**: Each answers 400 `{"error":"bad_request","message":<non-empty>,…}`; the first, second, third, fourth, fifth and sixth carry `errors.<the offending field>` (never 200 with the field ignored, never 422); the order is unchanged after all nine (the note is still `""`).
Traces: contract (§2.5 an undefined or wrongly typed field is 400; §3.15 `bad_request`); SC-17
Check: fetch_url

**Scenario**: C-4 — every refusal carries the one error shape, the JSON content type and a request id
**Given**: A placed order `<id>` and a cancelled order `<c>`; FX-CUSTOMER; a valid trace id `tc-trace-0001`.
**When**: The run collects the non-2xx answers of: `PUT /v1/orders/<id>` `{"status":"packed"}` (422) with the header `X-Request-Id: tc-trace-0001`; `PUT {"colour":"x"}` (400); `DELETE /v1/orders/<id>` (409); `DELETE /v1/orders/doi-99999` (404); `PUT` with `NONE` (401); `PUT` with `CUSTOMER` (403); and `GET /v1/orders` with `NONE` (401).
**Then**: Every answer has `Content-Type: application/json; charset=utf-8` and a JSON body with a non-empty `error` (the codes `invalid`, `bad_request`, `conflict`, `not_found`, `unauthenticated`, `forbidden`, `unauthenticated`) and a non-empty `message`; `errors` is present on the 422 and the 400 only; the answer with the sent trace id echoes `X-Request-Id: tc-trace-0001` and every other answer carries a generated `X-Request-Id` matching `^[A-Za-z0-9._-]{8,64}$`; no body holds SQL, a stack trace, a path or a table name.
Traces: contract (§2.3 error contract; §2.1 trace header)
Check: fetch_url

**Scenario**: C-5 — the dealer's list follows the paging and query language
**Given**: The 14 sample orders and three more placed with REQ (17 in all); ADMIN signed in.
**When**: `GET /v1/orders` with no query; `?limit=5&page=2`; `?page=0`; `?limit=0`; `?limit=abc`; `?limit=500`; `?page=99`; `?sort=nope`; `?zzz=1`; `?status=placed,confirmed&limit=1`; `?payment=vnpay&limit=100`; `?q=24817`; `?placed_at_from=2026-11-24&limit=100`; and, with `SELECT count(*) FROM orders WHERE status IN ('placed','confirmed')`, the number of waiting orders.
**Then**: The default answer is 200 `{"items":[…],"total":17,"page":1,"limit":12}` with 12 items; `limit=5&page=2` has 5 items and `page` 2; `page=0`, `limit=0` and `limit=abc` answer 400 `bad_request` with `errors.page` / `errors.limit` / `errors.limit`; `limit=500` answers 200 with `limit` 100; `page=99` answers 200 with `items` `[]` and `total` 17; `sort=nope` answers 400 with `errors.sort`; `?zzz=1` answers 200 exactly as with no query; `status=placed,confirmed&limit=1` has one item and a `total` equal to the SQL count (the menu's number); `payment=vnpay` holds only `vnpay` orders; `q=24817` holds `DOI-24817` and no order whose number and customer lack "24817"; `placed_at_from=2026-11-24` holds only orders placed on or after that day.
Traces: contract (§2.4 list query language; §3.14)
Check: fetch_url

**Scenario**: C-6 — a failed query is 500 `internal` with a generic message, for the read, the list, the save and the delete
**Given**: A placed order `<id>` and a cancelled order `<c>`; `ALTER TABLE orders RENAME TO orders_tc` through `set_db_state` (the sign-in tables are untouched, so the session still resolves).
**When**: ADMIN sends `GET /v1/orders`, `GET /v1/orders/<id>`, `PUT /v1/orders/<id>` `{"note":"x"}` and `DELETE /v1/orders/<c>`; then the table is renamed back and ADMIN reads both orders.
**Then**: All four answer 500 `{"error":"internal","message":<one generic sentence>}` with no `errors` key and a body that holds neither "orders_tc", "relation", "SQL", a path nor a stack trace (not 404, not 503); after the rename back both orders read exactly as before and `<c>` still exists.
Traces: contract (§2.3 `internal`; §3.14, §3.15, §3.16 `internal`)
Check: fetch_url

**Scenario**: C-7 — with the database refusing connections, the order routes answer a JSON error and nothing changes
**Given**: A placed order `<id>` read before; ADMIN's cookie held; `set_db_available(false)`.
**When**: ADMIN sends `PUT /v1/orders/<id>` `{"status":"confirmed"}`, `GET /v1/orders/<id>` and `DELETE /v1/orders/<id>`; then `set_db_available(true)` and ADMIN reads the order.
**Then**: Each of the three answers a non-2xx JSON error with `error` and `message` — 500 `internal` by the contract's §2.3 and §3.15 (a 503 `unavailable` is recorded as an open point, see 6 in section 4, not as a failure of the story) — and no order data, no SQL, no stack trace; no answer is 200, 204 or 404; after the database is back the order reads exactly as before (still `placed`).
Traces: contract (§2.3 `internal` / `unavailable`; §3.15, §3.16)
Check: fetch_url

## 16. Coverage — the story's forty criteria

Read back against the story's list (SC-1 … SC-40). Every criterion has at least one case; a criterion that names a failure has the case where it happens (SC-7c a failed return, SC-9b the race, SC-17 / SC-18 refused bodies, SC-32 the stale move, SC-35b the bad email, SC-36 the unreachable shop and the ended session, SC-37b / c the delete refusals, SC-38 the missing record).

| Criterion | Cases | | Criterion | Cases |
|---|---|---|---|---|
| SC-1 | SC-1a, SC-1b, SC-1c, SC-1d, C-1 | | SC-21 | SC-21 |
| SC-2 | SC-2a, SC-2b | | SC-22 | SC-22 |
| SC-3 | SC-3a, SC-3b, SC-3c | | SC-23 | SC-23a, SC-23b, C-5, C-6, C-7 |
| SC-4 | SC-4 | | SC-24 | SC-24a, SC-24b |
| SC-5 | SC-5a, SC-5b | | SC-25 | SC-25a, SC-25b |
| SC-6 | SC-6a, SC-6b | | SC-26 | SC-26a, SC-26b |
| SC-7 | SC-7a, SC-7b, SC-7c | | SC-27 | SC-27a, SC-27b |
| SC-8 | SC-8a, SC-8b | | SC-28 | SC-28 |
| SC-9 | SC-9a, SC-9b, SC-7c | | SC-29 | SC-29a, SC-29b |
| SC-10 | SC-10 | | SC-30 | SC-30a, SC-30b |
| SC-11 | SC-11 | | SC-31 | SC-31a, SC-31b |
| SC-12 | SC-12a, SC-12b | | SC-32 | SC-32 |
| SC-13 | SC-13a, SC-13b, SC-13c | | SC-33 | SC-33a, SC-33b |
| SC-14 | SC-14 | | SC-34 | SC-34a, SC-34b |
| SC-15 | SC-15a, SC-15b | | SC-35 | SC-35a, SC-35b |
| SC-16 | SC-16a | | SC-36 | SC-36a, SC-36b |
| SC-17 | SC-17a, SC-17b, SC-17c, C-3 | | SC-37 | SC-37a, SC-37b, SC-37c, SC-37d |
| SC-18 | SC-18a, SC-18b | | SC-38 | SC-38a, SC-38b |
| SC-19 | SC-19a, SC-19b (UNVERIFIED until item 8) | | SC-39 | SC-39a, SC-39b |
| SC-20 | SC-20 | | SC-40 | SC-40a, SC-40b |

Contract-only cases: C-1 … C-7 (415, 413, 400, the error shape, the list query language, a failed query, a refused database).

**Cases that may come back UNVERIFIED, and why:** SC-19b and the number half of SC-30a need plan item 8's notice address or its payment tables; SC-21's "payment attempts are gone" half needs `payment_attempts`; SC-36b needs either a second tab or `set_db_state` on `sessions`; SC-32, SC-36a, SC-38b and SC-39b need request interception or holding inside `interact_page`; C-7's status is read, not pinned (open point 6). SC-40b is `manual` and answered by repository and build tools where they exist.
