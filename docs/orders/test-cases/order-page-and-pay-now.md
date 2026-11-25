# Test cases — Order page and Pay now

Module: `orders` · Plan item 6 · Requirements: ORD-007 (all 26 acceptance criteria), ORD-006 items 4–6 and AC-12, ORD-009 AC-5
Story: `docs/orders/stories/order-page-and-pay-now.md` — criteria SC-1 … SC-32 (one or more cases each; the table at the end walks the list)
Contract: `docs/architecture/services.md` §2.3 (error envelope), §2.5 (validation boundary), §2.8 (the order's public view), §3.11 (`GET /v1/orders/{id}`), §3.13 (`POST /v1/orders/{id}/payment`); `docs/architecture/erd.md` §3.12–3.14 (`orders`, `order_items`, `order_tokens`)

**Risk level: high.** The route decides who may act on an order and is the first step of taking money; the page is the only place a customer learns where a paid order is, and it sits on the privacy line of ORD-008 (phone, address, email, note and the token must never reach it). Every refusal, every status, every failure line and every boundary of the story is therefore written out value by value. Copy that the design fixes is checked word for word.

## How these cases are observed

- **Windows.** Default 1280×800. Phone cases say 390×844 (and 320×640 where the criterion names it). Breakpoint cases use the widths they name (639, 640, 1023, 1024).
- **Addresses.** The API is read through the gateway as `/api/v1/...` (the backend itself mounts `/v1/...`). Pages: `/orders/<id>`, `/track`, `/shop/<id>`, `/contact`, `/returns`, `/checkout`.
- **Money and dates.** `vi-VN` writes `3.290.000 ₫` (the space before `₫` may be a no-break space: compare with either). Dates are `dd/mm/yyyy`; a date and time are `dd/mm/yyyy, HH:mm` in Vietnam time (`25/11/2026, 10:00`). API amounts are integers of VND. The word separators of the page are `·` (U+00B7) and `—` (U+2014), never an ASCII hyphen.
- **Selectors the story fixes.** `[data-testid='order-status']` (the h1), `[data-testid='pay-now']` (the button), `[data-testid='pay-now-error']` (its failure line), `[data-testid='order-missing']` (the not-found view), `[data-testid='order-total']` (the Total amount), `[data-testid='order-load-error']` (the could-not-load line). Beyond those no class is relied on: the details list is `main dl` (each row a `dt` label and a `dd` value), the five steps are `main ol` with one `li` per step, the two closed rows are `details` elements each with a `summary`, a loading placeholder carries `aria-busy="true"`, "the line" is the text of the element that follows the h1.
- **FORM** (typed in every UI case that places an order): name `Lan Nguyen`, phone `0912345678`, street address `12 Example Street, Ward 1`, city `Da Nang`, note `Gate code 4821`.
- **REQ** (the body of every API case that places an order), sent as `POST /api/v1/orders` with `Content-Type: application/json` and a fresh header `Idempotency-Key: op-<case>-<n>`:

```json
{ "items": [ { "product": "samba-og-w-core-black", "size": "38", "qty": 1 } ],
  "delivery": "standard", "payment": "vnpay",
  "customer": "Lan Nguyen", "phone": "0912345678",
  "address": "12 Example Street, Ward 1", "city": "Da Nang", "note": "Gate code 4821" }
```

  It answers 201 with `order` (`id`, `number`, …), `token` and `payment_url` `""` (the VNPay story has not landed). **PLACE** below means this request; its `order.id` and `token` are read from the answer. The first order after RESET is `doi-24818` / `DOI-24818`.
- **RESET** (before and after every group that placed orders): `DELETE FROM orders WHERE is_sample = false` (items and tokens go with them); `UPDATE counters SET value = 24817 WHERE name = 'order_number'`; every PRICE, record edit, draft or image edit of this file put back; any renamed table renamed back; the database made available again.
- **SET(order, "<assignments>")** — arrange with `set_db_state`: `UPDATE orders SET <assignments> WHERE id = '<order>'`. Nobody can move an order along through the API yet (no dealer session), so status, `paid`, `carrier`, `tracking`, `eta`, `payment_method_id`, `total`, `placed_at`, `customer`, `city` and `email` are set this way, as the story says. The order is deleted by RESET, so no value needs putting back.
- **SNAPSHOT(order)** — `SELECT status, paid, total, payment_method_id, carrier, tracking, eta, customer, city, updated_at FROM orders WHERE id = '<order>'` (`updated_at` is moved by a trigger on any update, so an equal `updated_at` proves nothing was written), plus `SELECT count(*) FROM order_tokens WHERE order_id = '<order>'` and `SELECT count(*) FROM payment_attempts WHERE order_id = '<order>'`. Until the VNPay story lands `payment_attempts` does not exist: the error `relation "payment_attempts" does not exist` satisfies "no attempt row". "Unchanged" means the snapshot is equal before and after.
- **SEEDTOKEN(order)** — to open a seeded sample order: `INSERT INTO order_tokens (order_id, token_hash) VALUES ('<order>', sha256('tc-seed-token-0001'::bytea))`, then send `X-Order-Token: tc-seed-token-0001`; the row is deleted afterwards. If the backend hashes the token in another form the arranged row opens nothing and the case is UNVERIFIED (instrument lacked: a way to mint a token for a seeded order), not failed.
- **HOLD(order)** — the browser opened for a UI case holds that order's token the way a real customer's browser does. The order is placed **through the page in that browser**: add the pair to the bag from `/shop/<shoe>` (size pill, `[data-testid='add-to-bag']`), open `/checkout`, type FORM, press `[data-testid='place-order']`; the browser lands on `/orders/<id>?placed=1`. Any further state of that order is then produced with SET and a reload. A second order held by the same browser is placed through the page too. Once the first order has shown the shape of the browser's store (the one `localStorage` entry that holds the order's id, number, placed time and token), a verifier may write an entry of that same shape for an order it placed through the API; if the shape cannot be learned the case is placed through the page. A case that says "a fresh browser" starts with empty `localStorage`, `sessionStorage` and cookies.
- **ORDER-A** — one pair: `samba-og-w-core-black`, size `38`, qty 1, FORM, held (HOLD). The shoe's price P (3.290.000 ₫) and name N are read first from `GET /api/v1/products/samba-og-w-core-black`.
- **ORDER-B** — two pairs, held (HOLD): `samba-og-w-core-black` `38` qty 1 (3.290.000 ₫) first, `samba-og-kids-cloud-white` `36` qty 2 (2.290.000 ₫ each) second; stored total 7.870.000 ₫. Names are read from `GET /api/v1/products/<id>` before any draft is set.
- **PRICE(shoe, v)** — `UPDATE products SET price = <v> WHERE id = '<shoe>'`; the old price is read first and put back.
- **Stubbing in the browser.** "STUB(path, answer)" intercepts the page's request to the API path and answers it with the status and body the case gives; "DELAY(path, ms)" lets the real request through and holds the real answer for that many milliseconds; "ABORT(path)" fails the request at the network. A case that needs one and the instrument cannot intercept is UNVERIFIED (instrument lacked: request interception) and says so. The log of requests records, for each, the method, URL, headers and body; the log of the console records every `console.*` call.
- **Timings.** "At 1 s" means one second after the navigation started, read from the page. No case depends on a time shorter than 0.3 s.

## Ambiguities found in the documents (reported, not silently resolved)

1. **A paid and cancelled order.** SC-2 says a paid order answers the "already paid" sentence; SC-3 says a cancelled order (paid or not) answers the "cancelled" sentence. For an order that is both, the cases follow SC-3's "paid or not" (TC-03 expects the cancelled sentence). If TL's design checks paid first, TC-03 (paid and cancelled) fails against the story and the story should say so.
2. **The three 409 sentences are not fixed.** The story fixes only that they differ (and the exact "Online payment is not available yet." of SC-5). The cases assert a keyword (`paid`, `cancel`, `online`), that the three are pairwise different and that none equals the SC-5 sentence. If TL fixes the sentences in the technical design the cases should pin them.
3. **Unpaid and delivered.** ORD-007 item 8 shows "Pay now" for "an unpaid order that is not cancelled"; SC-25 lists placed, confirmed, packed and shipped. No case pins "Pay now" for an unpaid delivered order. The route's answer for it is covered (TC-06: 409 "not available yet").
4. **Database unreachable: 500 or 503?** The story (SC-6) and `services.md` §2.3 give `internal` 500 for a database the service cannot read; 503 `unavailable` is for `/healthz` and a shutting-down process. The cases follow the story. A 503 on this route fails TC-10 and TC-14 against the story.
5. **The browser's store has no name in the story.** `src/lib/orders.ts` is built by the previous item; its key and shape are not pinned. HOLD places the order through the page so no case depends on the key.
6. **Vietnam time.** `formatDateTime` formats in the browser's own time zone; SC-7 requires Vietnam time. TC-17 can tell the two apart only in a browser whose zone is not UTC+7; the case records the zone it ran in. In a UTC+7 browser it cannot fail, and says so.
7. **What the page draws when the order arrives but a name record does not.** The story fixes the order being drawn only once `GET /v1/delivery` and `GET /v1/payments` have answered (SC-20) and says nothing about those two failing. No case pins it.
8. **`aria-busy`.** The story's UI table gives the loading placeholder `aria-busy="true"` ("the same treatment the data story gave the other pages when it has one"). TC-22 asserts the attribute because the table names it; the data story's own treatment, if different, is TL's to reconcile.
9. **`?placed` is a flag.** The story (§9) draws "Thank you" whenever the address carries `placed`, whatever its value or the order's status; TC-16 pins that, including `?placed=0`.
10. **Only a 200 answer carries an address.** The page "reads `payment_url` from a 200 answer"; no case sends a 201 or 204 as a success. TC-50 sends 200 answers whose body is wrong, which are failures.

---

## A. `POST /v1/orders/{id}/payment` (SC-1 … SC-6) and the contract of `GET /v1/orders/{id}`

**Scenario**: TC-01 — a stranger, a wrong token, a missing order and a malformed id all answer one and the same 404
**Given**: RESET; two orders placed by PLACE: O1 `doi-24818` (token T1) and O2 `doi-24819` (token T2); SNAPSHOT(O1) and the row count of `orders` read.
**When**: `POST /api/v1/orders/<id>/payment` is sent with no body and no `Content-Type`, and for each of these the same URL and headers are also sent as a `GET`: (a) O1 with no `X-Order-Token`; (b) O1 with `X-Order-Token: T2`; (c) O1 with an empty `X-Order-Token:`; (d) O1 with T1 whose last character is changed; (e) O1 with T1 only in the query (`?token=T1`) and no header; (f) `doi-99999` (does not exist) with T1; (g) `nope` with T1; (h) `DOI-24818` (upper case, malformed) with T1; (i) `..%2Fadmin` with T1; (j) a 300-character id of `a` with T1.
**Then**: Every POST answers 404 `{"error":"not_found","message":S}` with the one same sentence S, the same `Content-Type: application/json`, and no `errors` key; for each pair the POST body equals the GET body byte for byte; no body holds the id sent, a token, `DOI-` or `doi-`. SNAPSHOT(O1) is equal before and after (including `updated_at`, no payment attempt row, still one token row), and the `orders` row count is unchanged.
Traces: SC-1 (ORD-007 AC-20, permission row; ORD-006 item 4)
Check: fetch_url

**Scenario**: TC-02 — an order that is paid answers 409 saying it is already paid, in every status
**Given**: RESET; O1 placed (token T1); for each status s in `placed`, `confirmed`, `packed`, `shipped`, `delivered`: SET(O1, "status = '<s>', paid = true"); SNAPSHOT(O1) read before each call.
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with `X-Order-Token: T1` and no body, once per status.
**Then**: Each answers 409 `{"error":"conflict","message":S_paid}` with no `errors` key; S_paid is a non-empty sentence that contains `paid` (any case), and it is the same sentence in the five answers. SNAPSHOT(O1) is equal before and after each call.
Traces: SC-2 (ORD-007 AC-18, item 9)
Check: fetch_url

**Scenario**: TC-03 — a cancelled order answers 409 saying it was cancelled, whether or not it was paid
**Given**: RESET; O1 placed (T1); the four orders are arranged in turn on O1: (a) SET "status = 'cancelled', paid = false"; (b) SET "status = 'cancelled', paid = true"; (c) as (a) with SET "payment_method_id = 'cod'"; (d) as (a) with `eta` set to `2026-11-26`; SNAPSHOT taken before each call.
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with `X-Order-Token: T1` and no body for each.
**Then**: Each answers 409 `{"error":"conflict","message":S_cancelled}` with no `errors` key; S_cancelled contains `cancel` (any case) and is one sentence for all four (so (b) answers the cancelled sentence, not the paid one — SC-3 "paid or not", ambiguity 1). SNAPSHOT is equal before and after each call.
Traces: SC-3 (ORD-007 AC-19, item 9)
Check: fetch_url

**Scenario**: TC-04 — an order whose way to pay is not an online one answers 409 saying so; the three sentences differ
**Given**: RESET; one order placed by PLACE and then set unpaid, `status = 'placed'`, with `payment_method_id` `cod` (O1), `bank` (O2) and `momo` (O3); a fourth, O4, set to `status = 'shipped'`, `payment_method_id = 'cod'`; and, with SEEDTOKEN, up to three seeded sample orders from `SELECT id FROM orders WHERE is_sample AND NOT paid AND status <> 'cancelled' AND payment_method_id IN ('cod','bank','momo')`; S_paid (TC-02) and S_cancelled (TC-03) read from the same run; SNAPSHOT of each.
**When**: `POST /api/v1/orders/<id>/payment` is sent with the order's own token and no body for each.
**Then**: Each answers 409 `{"error":"conflict","message":S_online}` with no `errors` key; S_online contains `online` (any case) and is the same sentence for all of them. S_online, S_paid and S_cancelled are three different strings, and none of the three equals `Online payment is not available yet.`. SNAPSHOT of each is equal before and after. The seeded orders' token rows are deleted afterwards.
Traces: SC-4 (ORD-007 AC-17, item 9)
Check: fetch_url

**Scenario**: TC-05 — a VNPay order just placed answers 409 "Online payment is not available yet." every time, and nothing is written
**Given**: RESET; O1 placed by PLACE (VNPay, unpaid, status `placed`; `payment_url` `""`; token T1); SNAPSHOT(O1), the shoe's `stock` S0 (`GET /api/v1/products/samba-og-w-core-black`) read.
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with `X-Order-Token: T1` and no body three times in turn; after each, `GET /api/v1/orders/doi-24818` is sent with T1.
**Then**: Each POST answers 409 `{"error":"conflict","message":"Online payment is not available yet."}` (exactly that sentence) with no `errors` key and no `payment_url`. After each, the GET answers 200 with `paid` `false` and `status` `placed`; SNAPSHOT(O1) is equal to the first (still one token row, no `payment_attempts` row, or the table does not exist); `stock` is S0 (nothing returned to the shelf).
Traces: SC-5 (ORD-006 AC-12, items 4, 6; ORD-007 item 9)
Check: fetch_url

**Scenario**: TC-06 — until the VNPay story lands, every unpaid VNPay order gets the same answer, in any status
**Given**: RESET; O1 placed (T1), `payment_method_id` `vnpay`, `paid = false`; for status s in `confirmed`, `packed`, `shipped`, `delivered`: SET(O1, "status = '<s>'").
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with T1 and no body for each status.
**Then**: Each answers 409 `{"error":"conflict","message":"Online payment is not available yet."}`; SNAPSHOT(O1) is equal before and after each call.
Traces: SC-5 (ORD-007 item 9, "until the VNPay story lands")
Check: fetch_url

**Scenario**: TC-07 — the route makes no call outside the API
**Given**: The payment handler's source and its package (`orders`) on the PR's head; the backend's go.mod and imports.
**When**: The handler is read for any use of `net/http`'s client (`http.Get`, `http.Post`, `http.Client`, `http.DefaultClient`), `net.Dial` or any call to a payment provider, and the backend's tests are listed for the route.
**Then**: The handler imports and calls nothing that sends a request outside the process; it writes no row; the backend's tests for the route include the 409 "not available yet" case.
Traces: SC-5 ("the API made no outbound call"; `services.md` §2 "the API makes no outbound HTTP call")
Check: manual — no instrument observes the API's outbound traffic over HTTP; the evidence is the handler's source read from the PR diff (`get_pr`) and `run_tests` on the backend, which the verifier runs.

**Scenario**: TC-08 — a call that carries a body is refused 400 and changes nothing
**Given**: RESET; O1 placed (T1), unpaid, status `placed`; SNAPSHOT(O1).
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with `X-Order-Token: T1` and: (a) `Content-Type: application/json`, body `{}`; (b) body `{"amount":1}`; (c) body `{"payment_url":"https://pay.example.test/x"}`; (d) body `[]`; (e) body `null`; (f) `Content-Type: text/plain`, body `x`. Then, as controls: (g) `Content-Type: application/json` with an empty body; (h) no body and no `Content-Type`; (i) `Content-Length: 0`.
**Then**: (a)–(f) each answer 400 `{"error":"bad_request","message":<sentence>}` (an `errors` key, if present, is an object); none answers 404, 409, 415, 422 or 200. (g), (h) and (i) answer 409 "Online payment is not available yet." (an empty body is not "a body sent"). SNAPSHOT(O1) is equal before and after all nine requests.
Traces: SC-6 (services.md §3.13 `bad_request`)
Check: fetch_url

**Scenario**: TC-09 — a stranger who sends a body still gets the 404 (access is checked first)
**Given**: RESET; O1 (T1) and O2 (T2) placed; SNAPSHOT(O1).
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with body `{}` (`Content-Type: application/json`) and: (a) no token; (b) `X-Order-Token: T2`; (c) an empty token; and `POST /api/v1/orders/doi-99999/payment` with body `{}` and T1.
**Then**: All four answer 404 `{"error":"not_found","message":S}` with the same S and body as TC-01 (a); none answers 400. SNAPSHOT(O1) is unchanged.
Traces: SC-6 ("a stranger sending a body still gets the 404 of SC-1")
Check: fetch_url

**Scenario**: TC-10 — with the database away the route answers 500 `internal`, generic, and recovers
**Given**: RESET; O1 placed (T1); `set_db_available` takes the database away while the backend keeps running.
**When**: `POST /api/v1/orders/doi-24818/payment` is sent (a) with T1 and no body; (b) with no token; (c) with T1 and body `{}`; the database is then made available again and (a) is sent once more.
**Then**: (a), (b) and (c) answer 500 `{"error":"internal","message":<sentence>}` with no `errors` key; no body contains `SELECT`, `pgx`, `relation`, `connection`, `.go`, `goroutine`, a token, `0912345678` or `Example Street`. After the database is back, (a) answers 409 "Online payment is not available yet." as in TC-05.
Traces: SC-6 (database unreachable); contract (POST /v1/orders/{id}/payment, `internal`)
Check: fetch_url

**Scenario**: TC-11 — the checks run in the order access, body, state, availability
**Given**: RESET; O1 placed (T1) and O2 (T2); O1 is arranged in turn as (a) paid, status `placed`; (b) cancelled; (c) `payment_method_id = 'cod'`, unpaid; (d) VNPay, unpaid, status `placed`.
**When**: For each of (a)–(d) the call is sent with T1 and body `{}`; then for (a) with T2 and body `{}`; then for (a) the call is sent with T1 and no body.
**Then**: With T1 and a body each of (a)–(d) answers 400 `bad_request` (the body check comes before the state check); with T2 and a body, (a) answers 404 (access comes first); with T1 and no body (a) answers 409 with S_paid (the state check comes before the availability sentence, which (d) shows with no body: 409 "Online payment is not available yet.").
Traces: SC-6 (services.md §3.13 "Checks run in that order"; story §3)
Check: fetch_url

**Scenario**: TC-12 — every refusal of the route has the one error envelope and a request id
**Given**: One request of each kind from TC-01 (a), TC-02, TC-03, TC-04, TC-05, TC-08 (a) and TC-10, each sent with the header `X-Request-Id: op-request-0001`.
**When**: Each answer's headers and body are read.
**Then**: Every body is a JSON object whose keys are `error`, `message` and, only for the 400, `errors`; `error` is the lower-case code of its status (`not_found` 404, `conflict` 409, `bad_request` 400, `internal` 500); `message` is a non-empty sentence; `Content-Type` is `application/json`; the answer carries `X-Request-Id: op-request-0001`; no body holds SQL, a file path, a stack trace, a hash, a token or another person's data; no answer sets a cookie.
Traces: contract (POST /v1/orders/{id}/payment; §2.3 error contract)
Check: fetch_url

**Scenario**: TC-13 — the order read answers the public view, with no private field whatever the order holds
**Given**: RESET; O1 placed by REQ (phone `0912345678`, address `12 Example Street, Ward 1`, note `Gate code 4821`; token T1); SET(O1, "email = 'lan.secret@example.com', carrier = 'GHN', tracking = 'GHN-PT27XM', eta = '2026-11-26', status = 'shipped'"); a second order O2 `doi-24819` of two pairs (ORDER-B's pairs, token T2) placed by PLACE with two lines and no changes made to it afterwards.
**When**: `GET /api/v1/orders/doi-24818` is sent with `X-Order-Token: T1`; `GET /api/v1/orders/doi-24819` with T2.
**Then**: Both answer 200 with `Cache-Control: no-store` and these keys: `id`, `number`, `placed_at` (RFC 3339 ending `+07:00`), `status`, `items`, `total`, `delivery_fee` (0), `payment` (`vnpay`), `paid` (`false`), `delivery` (`standard`), `carrier`, `tracking`, `eta`, `customer` (`Lan Nguyen`), `city` (`Da Nang`). O1: `carrier` `GHN`, `tracking` `GHN-PT27XM`, `eta` `2026-11-26`; for a fresh order `carrier` and `tracking` are `""` and `eta` is `""` (never `null`). Each element of `items` has `product`, `name`, `colourway`, `image`, `size`, `qty`, `unit_price`; O2's `items` are in the order sent (core-black first) and `total` is Σ `unit_price × qty`. No body has the keys `phone`, `email`, `address`, `note` or `transaction_no`, nor the strings `0912345678`, `Example Street`, `lan.secret@example.com`, `Gate code 4821` or the token.
Traces: SC-24 (ORD-007 AC-25; ORD-008 AC-5), contract (GET /v1/orders/{id})
Check: fetch_url

**Scenario**: TC-14 — the order read fails as the contract says when the database is away
**Given**: RESET; O1 placed (T1); `set_db_available` takes the database away while the backend runs.
**When**: `GET /api/v1/orders/doi-24818` is sent with T1; the database is made available again; the same request is sent.
**Then**: The first answers 500 `{"error":"internal","message":<sentence>}` with no `errors` key and no SQL or stack in the body (not 404, not 503); after the database is back the same request answers 200 with the order.
Traces: contract (GET /v1/orders/{id}, `internal`); SC-11 (the page's "shop answers 500" half)
Check: fetch_url

---

## B. Finding the order (SC-7 … SC-11)

**Scenario**: TC-15 — after placing, `?placed=1` shows "Thank you — your order is placed." and the number and time line
**Given**: RESET; a fresh browser at 1280×800; `samba-og-w-core-black` size `38` has at least one pair; the request log on.
**When**: HOLD(ORDER-A) is done: the pair is added from `/shop/samba-og-w-core-black`, `/checkout` is filled with FORM and `[data-testid='place-order']` is pressed. Then `GET /api/v1/orders/doi-24818` is read with the token from the placing answer for its `placed_at` (or `SELECT to_char(placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'DD/MM/YYYY, HH24:MI') FROM orders WHERE id = 'doi-24818'` with `set_db_state`); then `/orders/doi-24818` (no `placed`) is opened.
**Then**: After placing the address is `/orders/doi-24818?placed=1`; `[data-testid='order-status']` reads exactly `Thank you — your order is placed.`; the line above it reads exactly `Order DOI-24818 · placed <D>` where D is the `placed_at` as `dd/mm/yyyy, HH:mm` in Vietnam time (the `·` is U+00B7, the `—` U+2014, no ASCII hyphen anywhere in the two). On `/orders/doi-24818` the h1 reads `We have your order.` and the same line is shown. `document.querySelectorAll('h1').length` is 1 in both.
Traces: SC-7 (ORD-007 AC-1, items 1, 2; ORD-009 AC-1)
Check: interact_page

**Scenario**: TC-16 — `placed` in the address is a flag: any value, any status
**Given**: ORDER-A held; SET(ORDER-A, "status = 'shipped'") for (c)–(f).
**When**: These addresses are opened in turn: (a) `/orders/doi-24818` with status `placed`; (b) `/orders/doi-24818?placed` (no value); (c) `/orders/doi-24818?placed=1` with status `shipped`; (d) `?placed=0`; (e) `?placed=true`; (f) `?other=1`.
**Then**: (a) the h1 reads `We have your order.`. (b), (c), (d) and (e) read `Thank you — your order is placed.`. (f) reads `Your order is on its way.` (a parameter that is not `placed` changes nothing). Each page has exactly one h1.
Traces: SC-7 (ORD-007 item 2; story §9 "`?placed` is a flag")
Check: interact_page

**Scenario**: TC-17 — the placed time is written in Vietnam time, whatever zone the browser is in
**Given**: ORDER-A held; (a) SET(ORDER-A, "placed_at = '2026-11-25 05:30:00+07'") (that moment is 22:30 on 24/11 in UTC); (b) SET(ORDER-A, "placed_at = '2026-12-31 23:59:00+07'"); the browser's zone read once with `Intl.DateTimeFormat().resolvedOptions().timeZone` and, if the instrument can set it, the browser is run once in `UTC` and once in `America/Los_Angeles` as well as its own zone.
**When**: `/orders/doi-24818` is opened for (a) and for (b).
**Then**: (a) the line reads `Order DOI-24818 · placed 25/11/2026, 05:30`; (b) reads `Order DOI-24818 · placed 31/12/2026, 23:59` — in every zone tried. The result names the zone each run used; in a UTC+7 browser the case cannot tell a wrong zone from a right one and reports that (ambiguity 6).
Traces: SC-7 ("in Vietnam time")
Check: interact_page

**Scenario**: TC-18 — a refresh, a return and a typed address all keep showing the order
**Given**: ORDER-A held; the browser is at `/orders/doi-24818?placed=1`; the placing answer's token T1 was read from the log; the request log on.
**When**: The page is reloaded twice; then `/shop` is opened and the browser goes back; then `/orders/doi-24818` is typed as a full address (no `placed`).
**Then**: After each step the order is shown: `[data-testid='order-status']` reads `Thank you — your order is placed.` for the `?placed=1` loads and `We have your order.` for the typed address; `[data-testid='order-missing']` never exists. Every `GET /api/v1/orders/doi-24818` in the log carries `X-Order-Token` equal to T1; T1 is in no address.
Traces: SC-8 (ORD-007 AC-2, item 11)
Check: interact_page

**Scenario**: TC-19 — a browser that refuses storage shows the order once, and the not-found view after a refresh
**Given**: RESET; before the page's scripts run `Storage.prototype.setItem` is replaced with a function that throws and `localStorage.getItem` returns `null`; the bag is filled in this page load (size pill and `[data-testid='add-to-bag']` on `/shop/samba-og-w-core-black`, then client-side navigation to `/checkout`, no reload); FORM typed. If the instrument cannot run a script before the page's own the case is UNVERIFIED (instrument lacked: a script injected before page load).
**When**: `[data-testid='place-order']` is pressed; the order page is read; then the page is reloaded.
**Then**: After pressing, the address becomes `/orders/doi-24818?placed=1` and the page shows the order: the h1 `Thank you — your order is placed.` and the line `Order DOI-24818 · placed <D>` (the token lives in the page's memory). After the reload the page shows the not-found view of TC-20 (`[data-testid='order-missing']` with the h1 `We can't find that order.`) and no order content. `SELECT count(*) FROM orders WHERE id = 'doi-24818'` is 1 (the order exists; only the browser lost its token).
Traces: SC-8 (ORD-007 item 11; ORD-009 "Storage refused" row)
Check: interact_page

**Scenario**: TC-20 — no token, another order's token and an unknown id all show the same not-found view
**Given**: RESET; O1 `doi-24818` placed by PLACE (so it exists); (a) a fresh browser with no storage; (b) a browser that holds the token of ORDER-A placed through the page, which is order `doi-24819` (HOLD), opening O1's address; (c) the same browser as (b) opening `doi-99999`; (d) the same browser opening `Not%20An%20Id`; 1280×800.
**When**: `/orders/doi-24818` is opened in (a) and (b), `/orders/doi-99999` in (c), `/orders/Not%20An%20Id` in (d); the view is read after the request has finished (no pending request).
**Then**: In all four: `[data-testid='order-missing']` exists, holds the h1 `We can't find that order.` (the page has exactly one h1), the line `Check the number in your confirmation message, then look it up with the phone you ordered with.` and a link `Track an order` whose `href` is `/track`; `[data-testid='order-status']`, "In this order", "Total" and "Pay now" are absent. The `outerHTML` of `[data-testid='order-missing']` is identical in (a), (b), (c) and (d), and the page text contains neither the id typed nor `DOI-24818`. Following the link goes to `/track`.
Traces: SC-9 (ORD-007 AC-3, item 10, "Not found" and "Permission" rows)
Check: interact_page

**Scenario**: TC-21 — with no token held the page still asks the shop once, with cookies and without the token header
**Given**: O1 `doi-24818` exists (PLACE); a fresh browser; a first load of `/` after which `document.cookie = 'op_probe=1; path=/'` is set; the request log on.
**When**: `/orders/doi-24818` is opened as a full page load.
**Then**: The log holds exactly one `GET /api/v1/orders/doi-24818`; it has no `X-Order-Token` header (not even an empty one) and no `token` in its URL; it carries the browser's cookies (the `Cookie` header holds `op_probe=1`; if the instrument's log cannot show `Cookie` that one clause is UNVERIFIED, instrument lacked: the request's `Cookie` header); the answer is 404; only after it `[data-testid='order-missing']` is shown. No second `GET` for the order follows in the next 2 seconds.
Traces: SC-9 ("the page still asks the shop once, with cookies and without the header")
Check: interact_page

**Scenario**: TC-22 — while the answer is 2 seconds away neither the not-found view nor the order is on the page
**Given**: ORDER-A held (token T1); DELAY(`GET /api/v1/orders/doi-24818`, 2000); 1280×800.
**When**: `/orders/doi-24818` is opened as a full page load, and the page is read at 0.3 s, 1 s and 1.8 s, then at 3 s.
**Then**: At 0.3 s, 1 s and 1.8 s: the text of the page does not contain `We can't find that order.`, `In this order`, `Total`, `Pay now` or `DOI-24818`; `[data-testid='order-status']`, `[data-testid='order-missing']` and `[data-testid='order-load-error']` do not exist; `main` holds no `h1`; an element with `aria-busy="true"` is in `main`. At 3 s: the order is shown (`[data-testid='order-status']` reads `We have your order.`), no element with `aria-busy="true"` remains in `main` and `[data-testid='order-missing']` never appeared.
Traces: SC-10 (ORD-007 AC-24, item 10)
Check: interact_page

**Scenario**: TC-23 — an unknown id shows the not-found view only after the 404 has come back
**Given**: DELAY(`GET /api/v1/orders/doi-99999`, 2000) (the real 404 is held for two seconds); (a) a fresh browser; (b) a browser holding ORDER-A's token.
**When**: `/orders/doi-99999` is opened and the page is read at 0.3 s, 1 s, 1.8 s and 3 s.
**Then**: At 0.3 s, 1 s and 1.8 s: `We can't find that order.` is not in the page, `[data-testid='order-missing']` does not exist, no order content shows, and an element with `aria-busy="true"` is in `main`. At 3 s: `[data-testid='order-missing']` exists with the h1 and line of TC-20. Same in (a) and (b).
Traces: SC-10 ("for an unknown id the not-found view appears only after the 404")
Check: interact_page

**Scenario**: TC-24 — the shop unreachable or failing: one line and "Try again", never the not-found view
**Given**: ORDER-A held (T1); the request log on.
**When**: `/orders/doi-24818` is opened with (a) ABORT(`GET /api/v1/orders/doi-24818`); (b) STUB 500 `{"error":"internal","message":"Something went wrong."}`; (c) STUB 502 with a non-JSON body `Bad Gateway`. In each, after the line shows, `Try again` is pressed once while the failure still stands; then the interception is removed and `Try again` is pressed again.
**Then**: While failing: `[data-testid='order-load-error']` shows exactly `We could not load your order. Please try again.`; a button whose text is `Try again` is in the page; `[data-testid='order-status']`, `[data-testid='order-missing']`, the text `We can't find that order.` and every other order content are absent. The first press sends exactly one more `GET /api/v1/orders/doi-24818` (with `X-Order-Token` T1) and the line is still shown. After the interception is removed the second press sends one more `GET`, the order is drawn (`[data-testid='order-status']` reads `We have your order.`) and `[data-testid='order-load-error']` and the `Try again` button are gone.
Traces: SC-11 (ORD-007 "Upstream failure" row; story §9 "Could not load")
Check: interact_page

**Scenario**: TC-25 — the real 500 (database away) shows the same line, and the page recovers
**Given**: ORDER-A held (T1) (placed while the database was up); `set_db_available` takes the database away while the backend runs.
**When**: `/orders/doi-24818` is opened and read at 4 s; the database is made available again; `Try again` is pressed.
**Then**: At 4 s `[data-testid='order-load-error']` shows `We could not load your order. Please try again.` with the `Try again` button; there is no `[data-testid='order-missing']`, no `We can't find that order.` and no order content, and the page is not left on its loading placeholder. After the database is back and `Try again` is pressed the order is drawn.
Traces: SC-11 (ORD-007 failure row, a real 500 from the shop); contract (GET /v1/orders/{id})
Check: interact_page

---

## C. The answer, the line, the steps and the picture (SC-12 … SC-18)

**Scenario**: TC-26 — the h1 of each status, and exactly one h1
**Given**: ORDER-A held; for each status s in `placed`, `confirmed`, `packed`, `shipped`, `delivered`, `cancelled`: SET(ORDER-A, "status = '<s>'") and the page reloaded.
**When**: `/orders/doi-24818` is opened (no `placed`), and again with `?placed=1`, for each status.
**Then**: Without `placed`, `[data-testid='order-status']` reads exactly: placed `We have your order.`; confirmed `Your order is confirmed.`; packed `Your order is packed.`; shipped `Your order is on its way.`; delivered `Your order was delivered.`; cancelled `This order was cancelled.`. With `?placed=1` it reads `Thank you — your order is placed.` for all six. Every page has exactly one `h1`, and it is `[data-testid='order-status']`.
Traces: SC-12 (ORD-007 AC-4, item 2)
Check: interact_page

**Scenario**: TC-27 — "Expected by" under the answer for an order that is not yet delivered or cancelled
**Given**: ORDER-A held; the status and expected date are set in turn with SET: (a) `shipped`, `eta = '2026-11-26'`; (b) `placed`, same date; (c) `confirmed`, same; (d) `packed`, same; (e) `shipped`, `eta = '2026-12-01'`; (f) `shipped`, `eta = '2027-01-01'`; (g) `shipped`, `eta = '2026-02-03'`.
**When**: `/orders/doi-24818` is opened for each.
**Then**: (a) the h1 reads `Your order is on its way.` and the line under it reads exactly `Expected by 26/11/2026.`; (b), (c) and (d) show the same line with their own h1; (e) `Expected by 01/12/2026.`; (f) `Expected by 01/01/2027.`; (g) `Expected by 03/02/2026.` (day first, no day lost to a time zone, no month swapped).
Traces: SC-13 (ORD-007 AC-4, item 3)
Check: interact_page

**Scenario**: TC-28 — a delivered order says so, and shows no expected date even when it has one
**Given**: ORDER-A held; (a) SET "status = 'delivered', eta = NULL"; (b) SET "status = 'delivered', eta = '2026-11-26'".
**When**: `/orders/doi-24818` is opened for (a) and (b).
**Then**: Both: the h1 reads `Your order was delivered.` and the line under it reads exactly `Delivered. If your delivery is wrong or damaged, message us.`; the text `Expected by` is nowhere in the page.
Traces: SC-14 (ORD-007 AC-5, item 3)
Check: interact_page

**Scenario**: TC-29 — a cancelled order shows its sentence in place of the steps, and no date line; an order with no date shows none
**Given**: ORDER-A held; (a) SET "status = 'cancelled', eta = NULL, paid = false"; (b) SET "status = 'cancelled', eta = '2026-11-26'"; (c) SET "status = 'cancelled', paid = true"; then, for (d) and (e), SET "eta = NULL" with status `placed` and `shipped`.
**When**: `/orders/doi-24818` is opened for each.
**Then**: (a), (b), (c): the h1 reads `This order was cancelled.`; the text `This order was cancelled. If you have a question about it, message us.` is shown and the element that holds it contains an `svg` (the alert icon); `main` holds no `ol` and none of the labels `Placed`, `Confirmed`, `Packed`, `On its way`, `Delivered` as steps; the text `Expected by` and the text `Delivered. If your delivery` are nowhere in the page. (d) and (e): the text `Expected by` is nowhere in the page.
Traces: SC-15 (ORD-007 AC-6, item 4, "Boundary" row)
Check: interact_page

**Scenario**: TC-30 — a packed order: three steps filled, Packed alone ringed and current, two not filled
**Given**: ORDER-A held; SET "status = 'packed'"; 1280×800. The dot of a step is the small round element inside each `li` of `main ol` (equal width and height of at most 16 px, `border-radius` at least 50 %).
**When**: `/orders/doi-24818` is opened and the five `li` of `main ol` are read in order, with the computed style of each dot and each label.
**Then**: The labels read, in order, `Placed`, `Confirmed`, `Packed`, `On its way`, `Delivered` (case as drawn; capitals by style are fine). The dots of Placed, Confirmed and Packed have computed `background-color` `rgb(8, 8, 8)` (the ink); the dots of On its way and Delivered do not (they are hollow on the ground `rgb(248, 248, 248)` with a visible 1 px border). Exactly one `li` has `aria-current="step"` and it is Packed; its dot has `outline-style: solid`, `outline-width: 1px`, `outline-offset: 2px`, `outline-color: rgb(8, 8, 8)`; no other dot has an outline. The labels of the three done steps are ink `rgb(8, 8, 8)`; the two upcoming labels are `rgb(107, 107, 107)`.
Traces: SC-16 (ORD-007 AC-7, item 4; design-system §2.1.20)
Check: interact_page

**Scenario**: TC-31 — the steps of the other four statuses
**Given**: ORDER-A held; the page opened for status `placed`, `confirmed`, `shipped` and `delivered` in turn (SET and reload); 1280×800.
**When**: The dots and `aria-current` of the five `li` are read as in TC-30.
**Then**: `placed`: one dot filled (Placed), Placed is the `aria-current="step"` and ringed. `confirmed`: two filled, Confirmed current. `shipped`: four filled, On its way current. `delivered`: all five filled, Delivered current. In each, exactly one `li` has `aria-current="step"`, only its dot has an outline, and every step after it is not filled.
Traces: SC-16 (ORD-007 AC-7, item 4: "up to and including the current one")
Check: interact_page

**Scenario**: TC-32 — the steps are a vertical list below 640 px and one line from 640 px
**Given**: ORDER-A held, status `packed`.
**When**: `/orders/doi-24818` is opened at widths 390, 639, 640, 1024 and 1280 (height 800) and the bounding box of each of the five `li` of `main ol` is read.
**Then**: At 390 and 639 the five `li` have the same left edge (within 1 px) and strictly increasing tops (a vertical list). At 640, 1024 and 1280 they have the same top (within 2 px) and strictly increasing lefts, and the right edge of the last is at most the window's width (one line). At 390 `document.documentElement.scrollWidth` is at most 390.
Traces: SC-17 (ORD-007 AC-8, item 4; Responsive NFR)
Check: interact_page

**Scenario**: TC-33 — no sideways scroll at 320 and 390 px with a 100-character name and city and two pairs
**Given**: ORDER-B held; SET(ORDER-B, "customer = repeat('Nguyen Van ', 9) || 'A', city = repeat('Thanh pho ', 9) || 'Da Nang Vn'") (a name and a city of exactly 100 characters each, made of words); status `shipped`, `eta = '2026-11-26'`, carrier `GHN`, tracking `GHN-PT27XM`.
**When**: `/orders/doi-24818` is opened at 320×640 and at 390×844; `document.documentElement.scrollWidth`, `window.innerWidth` and `window.scrollX` after `window.scrollTo(500, 0)` are read.
**Then**: At both widths `scrollWidth` is at most `innerWidth` (320 and 390), `scrollX` stays 0, and the page text holds the whole 100-character name (the `Delivering to` value starts with `Nguyen Van Nguyen Van`) and both pairs.
Traces: SC-17 (ORD-007 AC-8; Responsive NFR)
Check: interact_page

**Scenario**: TC-34 — the same with a name and a city that are one 100-letter word
**Given**: ORDER-B held; SET(ORDER-B, "customer = repeat('N', 100), city = repeat('D', 100)").
**When**: `/orders/doi-24818` is opened at 320×640 and at 390×844 and read as in TC-33.
**Then**: At both widths `scrollWidth` is at most `innerWidth`, `scrollX` stays 0 after `scrollTo(500, 0)`, and the `Delivering to` value holds the 100 `N`s, a comma and the 100 `D`s.
Traces: SC-17 (ORD-007 AC-8, "a 100-character customer name")
Check: interact_page

**Scenario**: TC-35 — the first pair's picture is at the right of the heading from 1024 px and above the pairs below it, and links to the shoe
**Given**: ORDER-B held (first pair `samba-og-w-core-black`); status `placed`.
**When**: `/orders/doi-24818` is opened at 1280, 1024 and 1023 wide (height 800); the first `a[href='/shop/samba-og-w-core-black']` that holds an `img` is read: its box, the `img`'s `naturalWidth`, and the boxes of the h1 and of the heading `In this order`; the link is then clicked.
**Then**: At 1280 and 1024 the `img` has `naturalWidth` above 0; its left edge is at or beyond the h1's right edge (within 1 px), its centre is in the right half of the window, and its top is at or above the h1's bottom. At 1023 the picture is above the heading `In this order` (its bottom is above that heading's top) and is not beside the h1. The only picture of the head is the first pair's: no `a[href='/shop/samba-og-kids-cloud-white']` holds an `img` above `In this order`. The click goes to `/shop/samba-og-w-core-black`.
Traces: SC-18 (ORD-007 AC-26, item 1)
Check: interact_page

**Scenario**: TC-36 — a first pair with no picture draws no picture and no empty box
**Given**: (a) ORDER-A held with `UPDATE products SET image = '' WHERE id = 'samba-og-w-core-black'` (the old value read first and put back); (b) ORDER-B held with the same edit on `samba-og-w-core-black` only (the second shoe keeps its picture). If the database refuses an empty image the case is UNVERIFIED (instrument lacked: a shoe without a picture).
**When**: `/orders/doi-24818` is opened at 1280×800 and at 390×844.
**Then**: In (a) and (b): `main` holds no `img` whose `src` is empty or whose `naturalWidth` is 0, and no link to `/shop/samba-og-w-core-black` that has neither text nor a picture; in (b) the second shoe's picture is not used in the head (no `a[href='/shop/samba-og-kids-cloud-white']` holding an `img` above `In this order`). The h1, the line, the five steps, `In this order`, `Total`, `Delivery and payment`, `Keep shopping` are all present, and `scrollWidth` is at most the window's width at both sizes.
Traces: SC-18 ("a first pair with no picture draws no picture and no empty box")
Check: interact_page

---

## D. The pairs, the total and the details (SC-19 … SC-24)

**Scenario**: TC-37 — two pairs, each with its name, size and quantity, its price as sold, and the stored total
**Given**: ORDER-B held; names N1 (core-black) and N2 (kids-cloud-white) read from `GET /api/v1/products/<id>`; PRICE(`samba-og-w-core-black`, 3000000) and PRICE(`samba-og-kids-cloud-white`, 1000000) (today's prices differ from the sold ones); SET(ORDER-B, "status = 'placed', total = 7000000"); 1280×800.
**When**: `/orders/doi-24818` is opened; then SET(ORDER-B, "total = 7870000") and the page reloaded; then `UPDATE products SET status = 'draft' WHERE id = 'samba-og-kids-cloud-white'` (put back afterwards) and the page reloaded.
**Then**: Under the heading `In this order` the list has two rows, in the order sent. Row 1: a link whose text equals N1 (any case) to `/shop/samba-og-w-core-black`, the text `EU 38 · Qty 1`, the price `3.290.000 ₫` (not `3.000.000 ₫`). Row 2: a link whose text equals N2 to `/shop/samba-og-kids-cloud-white`, `EU 36 · Qty 2`, `4.580.000 ₫` (2 × 2.290.000, not `2.000.000 ₫`). A label `Total` is shown and `[data-testid='order-total']` reads `7.000.000 ₫` in the first read (the stored total, not the sum the page could compute) and `7.870.000 ₫` after the second. After the shoe is a draft the second row still shows N2 as a link to `/shop/samba-og-kids-cloud-white` with the same size, quantity and price.
Traces: SC-19 (ORD-007 AC-21, item 5; ORD-003 AC-11)
Check: interact_page

**Scenario**: TC-38 — the shop's answer for two pairs: as sold, in order, readable though the shoe is now a draft
**Given**: As TC-37 (ORDER-B placed by PLACE with two lines instead of through the page; token T); the same PRICE edits; SET(ORDER-B, "total = 7000000"); the second shoe set to draft.
**When**: `GET /api/v1/orders/doi-24818` is sent with T; `GET /api/v1/products/samba-og-kids-cloud-white` is sent with no session.
**Then**: The order answers 200 with `items` of two elements in the order sent: `items[0]` `product` `samba-og-w-core-black`, `size` `38`, `qty` 1, `unit_price` 3290000; `items[1]` `product` `samba-og-kids-cloud-white`, `size` `36`, `qty` 2, `unit_price` 2290000; each has a non-empty `name` and `image`; `total` is 7000000 (the stored value). The product read answers 404 (it is a draft), so the name and image in `items[1]` came from the shop's live read, not from the public read.
Traces: SC-19 (ORD-007 AC-21; services.md §2.8 `items`)
Check: fetch_url

**Scenario**: TC-39 — Delivering to, Delivery and Payment read as the design writes them; "Free", never `0 ₫`; no Express, no cash on delivery
**Given**: ORDER-A held, status `placed`, unpaid; `GET /api/v1/delivery` and `GET /api/v1/payments` read first (record `standard`: `name` `Standard`, `days` `2–4 working days`; record `vnpay`: `name` `VNPay — ATM, Visa, Mastercard`); 1280×800.
**When**: `/orders/doi-24818` is opened and the heading `Delivery and payment` and its `main dl` are read; the page's visible text and `document.documentElement.outerHTML` are searched.
**Then**: The rows are, in order: `Delivering to` → `Lan Nguyen, Da Nang`; `Delivery` → `Standard · 2–4 working days · Free`; `Payment` → `VNPay — ATM, Visa, Mastercard · awaiting payment`; there is no `Carrier` and no `Tracking number` row. The value of `Delivery` ends with the word `Free`. No text in `main` matches `(^|[^\d.,])0\s*₫` (a standalone zero amount; amounts such as `3.290.000 ₫` do not match); no `dd` equals or contains the raw id `standard` or `vnpay` in lower case; the page's text and HTML contain none of `Express`, `cash on delivery`, `to pay on delivery` (any case).
Traces: SC-20 (ORD-007 AC-22, item 6; SITE-016)
Check: interact_page

**Scenario**: TC-40 — the names come from the shop's records, not from the page
**Given**: ORDER-A held; the old values of `delivery_options` (`standard`) and `payment_methods` (`vnpay`) read; `UPDATE delivery_options SET name = 'Careful', days = '3–5 working days' WHERE id = 'standard'` and `UPDATE payment_methods SET name = 'VNPay QR' WHERE id = 'vnpay'`; the old values put back at the end.
**When**: `/orders/doi-24818` is opened as a full page load.
**Then**: `Delivery` reads `Careful · 3–5 working days · Free` and `Payment` reads `VNPay QR · awaiting payment`; neither `Standard` as the delivery value nor `VNPay — ATM, Visa, Mastercard` is on the page. After the old values are put back and the page reloaded, the rows read as in TC-39.
Traces: SC-20 ("the names come from the shop's delivery and payment records")
Check: interact_page

**Scenario**: TC-41 — the order is drawn only once the names it needs have arrived
**Given**: ORDER-A held; (a) DELAY(`GET /api/v1/delivery` and its query forms, 2000) and DELAY(`GET /api/v1/payments`, 2000); (b) only the delivery request delayed; (c) only the payments request delayed; the order's own answer is not delayed.
**When**: `/orders/doi-24818` is opened as a full page load; the page is read at 0.3 s, 1 s, 1.8 s and 3 s.
**Then**: At 0.3 s, 1 s and 1.8 s in each of (a), (b), (c): `[data-testid='order-status']` does not exist, `main` holds no `dl` and none of the strings `standard`, `vnpay`, `Delivering to` or `Payment` as a value, `[data-testid='order-missing']` does not exist, and an element with `aria-busy="true"` is in `main`. At 3 s the whole order is drawn with the rows of TC-39.
Traces: SC-20 ("the order is drawn only once the names it needs have arrived"; a raw id is never drawn as a value)
Check: interact_page

**Scenario**: TC-42 — Payment says paid or awaiting payment
**Given**: ORDER-A held, VNPay; (a) unpaid, status `placed`; (b) SET "paid = true"; (c) SET "status = 'confirmed', paid = true"; (d) SET "status = 'confirmed', paid = false".
**When**: `/orders/doi-24818` is opened for each.
**Then**: (a) and (d): `Payment` reads exactly `VNPay — ATM, Visa, Mastercard · awaiting payment`. (b) and (c): `Payment` reads exactly `VNPay — ATM, Visa, Mastercard · paid`. In none does the text `to pay on delivery` appear.
Traces: SC-21 (ORD-007 AC-11, AC-13, item 6)
Check: interact_page

**Scenario**: TC-43 — Carrier and Tracking number appear only when set
**Given**: ORDER-A held; (a) SET "carrier = 'GHN', tracking = 'GHN-PT27XM', status = 'shipped'"; (b) SET "carrier = '', tracking = ''"; (c) SET "carrier = 'GHN', tracking = ''"; (d) SET "carrier = '', tracking = 'GHN-PT27XM'".
**When**: `/orders/doi-24818` is opened for each and the `dt` labels of `main dl` are read.
**Then**: (a) `Carrier` → `GHN` and `Tracking number` → `GHN-PT27XM`, after `Payment`, in that order. (b) neither label is in the page. (c) only `Carrier` → `GHN`. (d) only `Tracking number` → `GHN-PT27XM`. `Delivering to`, `Delivery` and `Payment` are present in all four.
Traces: SC-22 (ORD-007 AC-9, AC-10, item 6)
Check: interact_page

**Scenario**: TC-44 — two closed rows, their sentences and links, and "Keep shopping"
**Given**: ORDER-A held, status `placed`; 1280×800.
**When**: `/orders/doi-24818` is opened; the two `details` are read; the first `summary` is clicked and read; the second `summary` is clicked and read; each link is clicked in turn (going back each time); `Keep shopping` is clicked.
**Then**: On load there are two `details`, neither with `open`, their `summary` texts are `Need to change something?` and `Wrong size?`. After the first click it is open and shows exactly `Send us the order number and what to change; the address or the size can change until the order is packed.` and a link `Message us` with `href` `/contact`; after the second click it shows exactly `Exchanges are not accepted. For a wrong or damaged delivery, or any question, message us.` and a link `Return policy` with `href` `/returns`. The links land on `/contact` and `/returns`. A link `Keep shopping` with `href` `/shop` is in the page and lands on `/shop`.
Traces: SC-23 (ORD-007 AC-23, item 7)
Check: interact_page

**Scenario**: TC-45 — phone, email, address and note appear nowhere on the page or in the answer it received
**Given**: ORDER-A held with FORM (phone `0912345678`, address `12 Example Street, Ward 1`, note `Gate code 4821`); SET(ORDER-A, "email = 'lan.secret@example.com'"); the request log and the response bodies recorded; the token T1 read from the placing answer.
**When**: `/orders/doi-24818` is opened (also at `?placed=1`) at 1280×800 and at 390×844; `Pay now` is pressed once; the page's visible text, `document.documentElement.outerHTML` (every attribute: `href`, `src`, `title`, `alt`, `aria-*`, `data-*`) and the recorded response bodies of the order read and the payment call are searched.
**Then**: None of `0912345678`, `0912 345 678`, `12 Example Street`, `Ward 1`, `lan.secret@example.com`, `Gate code 4821`, `example.com` appears in the visible text or in any attribute of the page; the response bodies of `GET /api/v1/orders/doi-24818` and of the 409 hold no key `phone`, `email`, `address` or `note` and none of those strings. `Da Nang` and `Lan Nguyen` (the public `city` and `customer`) are shown.
Traces: SC-24 (ORD-007 AC-25; ORD-008 AC-5)
Check: interact_page

**Scenario**: TC-46 — the token is only ever the `X-Order-Token` header of the page's two requests
**Given**: ORDER-A held (token T1 read from the placing answer); the request log (method, URL, headers, body) and the console log on.
**When**: `/orders/doi-24818` is opened and reloaded once; `Pay now` is pressed once (the live 409); the page, the log, `location.href`, `document.title`, `document.cookie`, `sessionStorage` and `localStorage` are read.
**Then**: The requests that carry an `X-Order-Token` header are exactly the `GET /api/v1/orders/doi-24818` of each load and the one `POST /api/v1/orders/doi-24818/payment`; its value is T1. No other request (the delivery and payments reads, any picture, font or script) carries it. T1 is in no request URL or query string, in no request body, in no `Referer` and in no cookie; not in `location.href`, `document.title`, the visible text, any attribute of the page (`href`, `src`, `data-*`, `aria-*`) or any `console` message; `sessionStorage` and `document.cookie` do not hold it. Exactly one `localStorage` entry holds it (the browser's store of placed orders).
Traces: SC-24 (ORD-007 AC-25; ORD-008 AC-10, item 4)
Check: interact_page

---

## E. "Pay now" (SC-25 … SC-29)

**Scenario**: TC-47 — "Pay now" for an unpaid order that is not cancelled, and for no other
**Given**: ORDER-A held, VNPay; the states below set with SET and the page reloaded each time.
**When**: `/orders/doi-24818` is opened for each state: unpaid with status `placed`, `confirmed`, `packed`, `shipped`; paid with status `placed`, `confirmed`, `packed`, `shipped`, `delivered`; cancelled unpaid; cancelled paid. Then the not-found view (`/orders/doi-99999`), the could-not-load view (ABORT of the order read) and the loading placeholder (DELAY 2000, read at 1 s) are opened.
**Then**: For the four unpaid states exactly one `[data-testid='pay-now']` exists, it is a `button` whose text is `Pay now`, and it is enabled. For the five paid states and the two cancelled states there is none (and no text `Pay now` anywhere). In the not-found, could-not-load and loading views there is none.
Traces: SC-25 (ORD-007 AC-13, AC-15, AC-16, item 8; ambiguity 3)
Check: interact_page

**Scenario**: TC-48 — the shop refuses "Pay now" with 409; the page shows the failure line and stays put
**Given**: ORDER-A held (T1), unpaid, status `placed`; the request log with bodies; a marker `window.__op = 1` set in the page so a reload can be seen; 1280×800.
**When**: `[data-testid='pay-now']` is pressed once.
**Then**: The log holds exactly one `POST /api/v1/orders/doi-24818/payment`, with `X-Order-Token` equal to T1 and no body (no request body and no `Content-Length` above 0). The shop's answer is 409 `{"error":"conflict","message":"Online payment is not available yet."}`. Under the button, `[data-testid='pay-now-error']` is visible, reads exactly `We could not open the payment page. Please try again.`, has `role="alert"` and holds an `svg` (the alert icon); its top is at or below the button's bottom. The address is unchanged, `window.__op` is still 1 (no navigation, no reload), the order's h1 and the details are still shown, and the button is enabled, reads `Pay now`, and has no `disabled` attribute.
Traces: SC-26 (ORD-007 AC-12; ORD-006 AC-12, item 4)
Check: interact_page

**Scenario**: TC-49 — the failure line is gone when the next press starts
**Given**: As TC-48 after the first press (the line is shown); the next call is held for 1.5 s (DELAY of the real `POST /api/v1/orders/doi-24818/payment`, 1500).
**When**: `[data-testid='pay-now']` is pressed again and the page is read at 0.3 s and at 2.5 s; then pressed a third time.
**Then**: At 0.3 s `[data-testid='pay-now-error']` does not exist and the button is disabled and reads `Opening the payment page…`. At 2.5 s the line is back with the same sentence and the button is enabled and reads `Pay now`. The third press sends a third `POST` and behaves the same. In all, one `POST` per press.
Traces: SC-26 ("the line is gone when the next press starts"; the button enabled again)
Check: interact_page

**Scenario**: TC-50 — every other failure of the call draws the same line and the same enabled button
**Given**: ORDER-A held (T1), unpaid; for each row below the `POST /api/v1/orders/doi-24818/payment` is answered by a STUB (a fresh page load per row; the marker `window.__op = 1` set; dialogs recorded): (a) 503 `{"error":"payment_unavailable","message":"Online payment is not available right now."}`; (b) 500 `{"error":"internal","message":"Something went wrong."}`; (c) 404 `{"error":"not_found","message":"Not found."}`; (d) ABORT (a network error); (e) 200 `{}`; (f) 200 `{"payment_url":""}`; (g) 200 `{"payment_url":null}`; (h) 200 `{"payment_url":42}`; (i) 200 with the body `ok` (not JSON); (j) 200 `{"payment_url":"javascript:alert(1)"}`; (k) 200 `{"payment_url":"ftp://pay.example.test/x"}`; (l) 200 `{"payment_url":"//pay.example.test/x"}`; (m) 200 `{"payment_url":"pay.example.test/x"}`; (n) 200 `{"payment_url":"data:text/html,hi"}`.
**When**: `[data-testid='pay-now']` is pressed once for each row.
**Then**: For every row: exactly one `POST` is in the log; `[data-testid='pay-now-error']` reads exactly `We could not open the payment page. Please try again.` with `role="alert"`; the button is enabled, reads `Pay now` and has no `disabled` attribute; the address is unchanged, `window.__op` is still 1, no dialog opened and no request to `pay.example.test` was made; the order is still shown. After (d) a second press (the stub removed) sends a second `POST`.
Traces: SC-27 (ORD-007 "Upstream failure" row; ORD-006 item 5)
Check: interact_page

**Scenario**: TC-51 — an address in a 200 answer takes the browser to exactly that address
**Given**: ORDER-A held (T1), unpaid; the `POST /api/v1/orders/doi-24818/payment` answered with 200 and, in turn, (a) `{"payment_url":"https://pay.example.test/x"}`; (b) `{"payment_url":"https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?vnp_Amount=329000000&vnp_OrderInfo=Thanh+toan+don+hang+DOI-24818&vnp_SecureHash=ab12cd"}`; (c) `{"payment_url":"http://pay.example.test/x"}`; every navigation to `pay.example.test` and `sandbox.vnpayment.vn` is intercepted and answered with an empty page, so no real payment page is called.
**When**: `[data-testid='pay-now']` is pressed once for each.
**Then**: The browser's address becomes exactly the `payment_url` of the answer (for (b) the query is unchanged, no re-encoding, no added parameter, and no token anywhere in it); exactly one `POST` was sent, with `X-Order-Token` T1 and no body; `[data-testid='pay-now-error']` was never shown.
Traces: SC-28 (ORD-007 AC-14, item 8; ORD-006 AC-7)
Check: interact_page

**Scenario**: TC-52 — while the answer is awaited the button is disabled, says what is happening, and a second press sends nothing
**Given**: ORDER-A held, unpaid; the `POST` answered after 2000 ms with (a) 200 `{"payment_url":"https://pay.example.test/x"}`; (b) 409 `{"error":"conflict","message":"x"}`; the log of requests on.
**When**: `[data-testid='pay-now']` is pressed once; at 0.3 s, 1 s and 1.8 s the button is read and is clicked three more times, and Enter and Space are pressed with focus on the page's body and on the button's place; the page is read again at 3 s.
**Then**: At every reading before the answer: the button has the `disabled` attribute and its text is exactly `Opening the payment page…`; its computed `opacity` is 0.6; `[data-testid='pay-now-error']` does not exist; the log holds exactly one `POST` after all the extra presses. At 3 s in (a) the address is `https://pay.example.test/x`; in (b) the failure line is shown and the button is enabled and reads `Pay now`.
Traces: SC-28 (ORD-007 AC-14, item 8; story §9 "Pay now while waiting")
Check: interact_page

**Scenario**: TC-53 — coming back with the Back button, "Pay now" is usable again
**Given**: As TC-51 (a): the page has been sent to `https://pay.example.test/x` (answered with an empty page).
**When**: The browser's Back button is pressed (`history.back()`); the page is read; `[data-testid='pay-now']` is pressed again with the next call answered 409 by the real shop.
**Then**: The address is `/orders/doi-24818` again, the order is shown, `[data-testid='pay-now']` is enabled, has no `disabled` attribute and reads `Pay now` (not `Opening the payment page…`). The new press sends one more `POST` and, with the real 409, shows the failure line of TC-48.
Traces: SC-28 (ORD-007 AC-14; ORD-006 AC-7: "comes back with its Back button")
Check: interact_page

**Scenario**: TC-54 — the page needs nothing from the answer but the status and `payment_url`
**Given**: ORDER-A held, unpaid; the call answered by STUB in turn: (a) 200 `{"payment_url":"https://pay.example.test/x","expires_at":"2026-11-25T10:30:00+07:00","reference":"DOI-24818-1","provider":"vnpay"}`; (b) 409 `{"error":"conflict","message":"This order is already paid."}`; (c) 409 `{"error":"conflict","message":"This order was cancelled."}`; (d) 409 `{"error":"payment_unavailable","message":"zzz"}`; (e) 409 `{}`.
**When**: `[data-testid='pay-now']` is pressed once for each.
**Then**: (a) the browser goes to `https://pay.example.test/x` (extra fields change nothing). (b), (c), (d), (e) each show exactly `We could not open the payment page. Please try again.` and nothing of the shop's `message` (`already paid`, `cancelled`, `zzz` are nowhere on the page); the button is enabled. This is the same page build as TC-48, so a shop that starts answering 200 with an address (the VNPay story) needs no change to the page.
Traces: SC-29 (ORD-006 item 4; ORD-007 item 8, last sentence of item 9)
Check: interact_page

**Scenario**: TC-55 — "Pay now" is the page's one filled pill; "Keep shopping" stays an outline pill
**Given**: ORDER-A held, unpaid, status `placed`; 1280×800.
**When**: `/orders/doi-24818` is opened; the computed styles of `[data-testid='pay-now']` and of `main a[href='/shop']` (`Keep shopping`) are read; the filled elements are counted.
**Then**: `[data-testid='pay-now']`: `background-color` `rgb(8, 8, 8)`, `color` `rgb(248, 248, 248)`, `text-transform` `uppercase`, `font-size` 12px, `font-weight` 500, a `border-radius` of at least half its height (a pill). `Keep shopping`: `background-color` transparent (`rgba(0, 0, 0, 0)`), a 1px border of `rgb(8, 8, 8)`, `color` `rgb(8, 8, 8)`. No other button or link in `main` has the ink fill: exactly one filled pill.
Traces: SC-25 (ORD-007 item 8; design-system §2.1.6, §2.1.5)
Check: interact_page

---

## F. The whole flow, the clean-up and the keyboard (SC-30 … SC-32)

**Scenario**: TC-56 — choose a pair, place the order, land on the order page that says "Thank you" and offers "Pay now"
**Given**: RESET (the next number is `DOI-24818`); a fresh browser at 1280×800 with an empty bag; the running stack (no stub, no intercepted request); `samba-og-w-core-black`'s price P read from `GET /api/v1/products/samba-og-w-core-black`, and one EU size that is not in its `sizes_out`.
**When**: `/shop/samba-og-w-core-black` is opened; that size's pill is chosen; `[data-testid='add-to-bag']` is pressed; `/checkout` is opened; `Full name` is typed `Lan Nguyen`, `Phone` `0912345678`, `Street address, ward, district` `12 Example Street, Ward 1`, `City or province` `Da Nang`; the payment row is read; `[data-testid='place-order']` is pressed (the shop's `payment_url` is `""`, so the browser stays on the site).
**Then**: Step 03 holds one payment row, `[data-testid='pay-vnpay']`, with its radio checked and no other radio with `name="payment"`. The browser lands on `/orders/doi-24818?placed=1`; `[data-testid='order-status']` contains `Thank you` (it reads `Thank you — your order is placed.`); the line above it begins `Order DOI-24818 · placed `; `Payment` reads `VNPay — ATM, Visa, Mastercard · awaiting payment`; `[data-testid='order-total']` reads P as `vi-VN` (for example `3.290.000 ₫`); `[data-testid='pay-now']` is shown and enabled; the bag is empty (`[data-testid='bag-count']` reads 0 or is absent). The visible text and HTML of the page contain none of `0912345678`, `12 Example Street` and the order's token (read from the placing answer). Pressing `Pay now` shows the failure line of TC-48 (the VNPay story has not landed).
Traces: SC-30 (ORD-009 AC-5; ORD-006 AC-6)
Check: interact_page

**Scenario**: TC-57 — the page reads the order from the API alone: no mock, no order list, no shoe record
**Given**: A fresh browser that has placed ORDER-A through the page (HOLD); the request log on; `localStorage` read before and after for a key named `doi:orders` (the design's mock store); the seeded order `doi-24817` (the one the mock page used to draw) exists.
**When**: `/orders/doi-24818` is opened as a full page load; then, in a second fresh browser with no storage, `/orders/doi-24817` is opened.
**Then**: The first page's requests include `GET /api/v1/orders/doi-24818`, `GET /api/v1/delivery` and `GET /api/v1/payments` (any query form) and no `GET /api/v1/orders` list (no request whose path is exactly `/api/v1/orders`) and no `GET /api/v1/products/samba-og-w-core-black`; the page shows `DOI-24818`, which the mock could never know. `localStorage` has no `doi:orders` key before or after. The second browser shows the not-found view of TC-20 for `doi-24817` (the page no longer draws the mock's record for an id the browser holds no token for).
Traces: SC-31 (ORD-007 clean-up; overview §5), SC-9
Check: interact_page

**Scenario**: TC-58 — the clean-up and the frozen files, read from the pull request
**Given**: The PR for this story on its head commit (`get_pr`): its file list and diff; the frontend's tests and lint (`run_tests` with target `frontend`).
**When**: The diff is read for: the imports of `src/pages/Order.tsx`; every `useCollection`, `useRecord` and `useShoe`-style hook it calls; `src/lib/mock/orders.ts`; the files the PR edits; the line count of every file the PR adds or changes; the PR description.
**Then**: `Order.tsx` imports nothing from `src/lib/mock/` and calls no collection hook on `orders` or `products`; its only record reads are `delivery` and `payments`. `src/lib/mock/orders.ts` is deleted if and only if no file imports it after the PR (while `Track.tsx` imports it the file stays and the PR description says so). The PR adds `src/lib/order-page-and-pay-now.ts`, which reads the token through the existing reader in `src/lib/orders.ts` and adds no second store or storage key. The PR edits none of `src/pages/Track.tsx`, `src/pages/Checkout.tsx`, `src/App.tsx`, `src/lib/orders.ts`, `src/theme.css`, `src/index.css`, `src/content.json`, `package.json`. No file is over 300 lines. The PR description lists the new words as content keys under `pages.order.*` (`Pay now`, `Opening the payment page…`, `We could not open the payment page. Please try again.`, `We could not load your order. Please try again.`, `Try again`). `run_tests` for the frontend (which runs lint and tests as CI does) passes.
Traces: SC-31 (ORD-007 clean-up; story §3 "Clean-up")
Check: manual — no HTTP instrument reads a source diff or a file list; the evidence is `get_pr` (file list and diff) and `run_tests`, which the verifier runs and quotes. The runtime half of SC-31 is TC-57.

**Scenario**: TC-59 — the keyboard alone reaches everything, in reading order, and Enter and Space work
**Given**: ORDER-B held, unpaid, status `placed`; (a) 390×844, (b) 1280×800; focus starts at the top of the document (the page just loaded).
**When**: Tab is pressed repeatedly (at most 80 times) and, for each stop inside `main`, the focused element's tag, accessible name, `href`, `data-testid` and top edge are recorded; then, with focus on the first `summary`, Enter is pressed and then Space; with focus on the second `summary`, Enter; then with focus on `[data-testid='pay-now']` Enter is pressed (the shop answers its real 409), and, after the line shows, with focus on the button again, Space.
**Then**: The stops inside `main` are exactly the focusable elements of `main` in document order (no `tabindex` above 0 anywhere in `main`, none skipped, none visited twice) and they include, each once: the picture link `/shop/samba-og-w-core-black`, the two pair-name links, `[data-testid='pay-now']`, the `summary` `Need to change something?`, the `summary` `Wrong size?`, the links shown in an open row, and `Keep shopping`. In (a), the single column, the top edges of the stops never decrease from one stop to the next (document order is reading order). Enter toggles the first `summary` open and Space toggles it closed again; Enter opens the second. Enter on `Pay now` sends one `POST` and shows `[data-testid='pay-now-error']`; Space on it sends a second.
Traces: SC-32 (Accessibility NFR)
Check: interact_page

**Scenario**: TC-60 — focus is visible on every control of the page
**Given**: ORDER-B held, unpaid; 1280×800.
**When**: Tab is pressed to each of: the picture link, a pair-name link, `[data-testid='pay-now']`, each `summary`, `Keep shopping`; at each stop the focused element's computed `outline-style`, `outline-width`, `outline-offset` and `box-shadow` are inspected.
**Then**: At every stop either `outline-style` is not `none` and `outline-width` is at least 1px, or `box-shadow` is not `none`; none of the stops has both `outline-style: none` and `box-shadow: none`.
Traces: SC-32 (Accessibility NFR: "focus is visible")
Check: interact_page

**Scenario**: TC-61 — the two failure lines are announced to assistive technology
**Given**: ORDER-A held, unpaid. (a) the page open and the order shown; (b) the order read ABORTed.
**When**: (a) `[data-testid='pay-now']` is pressed so the live 409 produces the failure line; the DOM is read before and after for the line and its attributes. (b) `/orders/doi-24818` is opened so the could-not-load line shows.
**Then**: (a) `[data-testid='pay-now-error']` is not in the page before the press; after it, it has `role="alert"` (or sits inside an element with `role="alert"` or `aria-live="assertive"`) and reads `We could not open the payment page. Please try again.`. (b) `[data-testid='order-load-error']` has `role="alert"`, or `role="status"`, or `aria-live` of `assertive` or `polite` on it or an ancestor inside `main`, and reads `We could not load your order. Please try again.`. In both the alert icon is decorative (`aria-hidden="true"`) or carries an accessible name; neither line is only a colour change.
Traces: SC-32 (Accessibility NFR: "announced to assistive technology"); SC-11, SC-26
Check: interact_page

---

## Coverage walk — every story criterion has a case

| Criterion | Cases | Criterion | Cases |
|---|---|---|---|
| SC-1 | TC-01, TC-09 | SC-17 | TC-32, TC-33, TC-34 |
| SC-2 | TC-02, TC-11 | SC-18 | TC-35, TC-36 |
| SC-3 | TC-03, TC-11 | SC-19 | TC-37, TC-38 |
| SC-4 | TC-04 | SC-20 | TC-39, TC-40, TC-41 |
| SC-5 | TC-05, TC-06, TC-07 | SC-21 | TC-42, TC-56 |
| SC-6 | TC-08, TC-09, TC-10, TC-11 | SC-22 | TC-43 |
| SC-7 | TC-15, TC-16, TC-17 | SC-23 | TC-44 |
| SC-8 | TC-18, TC-19 | SC-24 | TC-13, TC-45, TC-46 |
| SC-9 | TC-20, TC-21, TC-57 | SC-25 | TC-47, TC-55 |
| SC-10 | TC-22, TC-23 | SC-26 | TC-48, TC-49, TC-61 |
| SC-11 | TC-24, TC-25, TC-14, TC-61 | SC-27 | TC-50 |
| SC-12 | TC-26 | SC-28 | TC-51, TC-52, TC-53 |
| SC-13 | TC-27 | SC-29 | TC-54 |
| SC-14 | TC-28 | SC-30 | TC-56 |
| SC-15 | TC-29 | SC-31 | TC-57, TC-58 |
| SC-16 | TC-30, TC-31 | SC-32 | TC-59, TC-60, TC-61 |
| Contract (`POST /v1/orders/{id}/payment`, `GET /v1/orders/{id}`) | TC-12, TC-13, TC-14 | | |

**Criteria walked: 32 of 32.** Where a criterion names a failure it has its own case where the failure happens: SC-1 (a stranger, TC-01), SC-2/3/4 (the three refusals, TC-02/03/04), SC-5 (the unavailable answer, repeated, TC-05/06), SC-6 (a body, a stranger's body, a database away, TC-08/09/10), SC-8 (storage refused, TC-19), SC-9 (no token, wrong token, no order, TC-20/21), SC-10 (a slow answer, TC-22/23), SC-11 (network failure, 500, 502 and a real 500, TC-24/25), SC-15 (no date, a cancelled order, TC-29), SC-18 (no picture, TC-36), SC-19 (a draft shoe, a changed price, TC-37), SC-27 (fourteen failures of the call, TC-50), SC-28 (waiting, Back, TC-52/53).

**Instruments (61 cases).** `fetch_url` 14 (TC-01 … TC-06, TC-08 … TC-14, TC-38 — 15 listed numbers less TC-07, which is `manual`); `interact_page` 45 (TC-15 … TC-37, TC-39 … TC-57, TC-59 … TC-61 — every case whose `When` is somebody typing, clicking, pressing a key, reloading or going back, or where a request is stubbed, delayed or aborted; the visual and computed readings — step dots, outline, opacity, fill colour, box positions, scroll width — are taken inside `interact_page` as computed values, the same readings `measure_styles` returns, because each needs the order held in the browser's own store and a state that a bare URL cannot reach); `manual` 2 (TC-07, TC-58). `render_url` and `measure_styles` are not used as separate instruments: the page cannot show an order without a token held in the browser, and `render_url` and `measure_styles` open a fresh address; where a verifier prefers `measure_styles` for a computed reading (TC-30, TC-55) it answers the same question.

Each `manual` case names why no instrument answers it: TC-07 (the API's outbound traffic is not observable over HTTP; the handler's source is read), TC-58 (a source diff and a file list are not an HTTP response; `get_pr` and `run_tests` are the evidence). Cases that depend on the verifier being able to intercept, delay or abort a request, run a script before the page's own, emulate a time zone, read a request's `Cookie` header or press the Back button say so in their own text; if the instrument cannot do it the case is UNVERIFIED with that reason, never failed.
