# Test cases — Online payment with VNPay

Module: `payments` · Plan item 8 · Requirements: PAY-001, PAY-002, PAY-003, PAY-004, PAY-005 (`docs/payments/SRS.md`); the payment half of ORD-006 and ORD-007 item 9 (`docs/orders/SRS.md`)
Story: `docs/payments/stories/online-payment-with-vnpay.md` — criteria SC-1 … SC-39 (one or more cases each; the table at the end walks the list)
Contract: `docs/architecture/services.md` §2.3 (error envelope), §3.5 (`GET /v1/payment-options`), §3.10 (`POST /v1/orders`), §3.11 (`GET /v1/orders/{id}`), §3.13 (`POST /v1/orders/{id}/payment`), §3.17 (`GET /v1/payments/vnpay/ipn`); `docs/architecture/erd.md` §3.12 (`orders`), §3.16 (`payment_attempts`), §3.17 (`payment_notices`)

**Risk level: high.** This item takes money, decides when an order counts as paid, hands pairs back to a shared shelf on a timer, and answers a public, unauthenticated address that anyone on the internet can call. Every notice code, every ordering of the notice's checks, every race, every time boundary and every secret-handling rule is therefore written out value by value. A page that only draws words is checked word for word, but the result page is also the only place a customer learns whether to pay again, so each of its seven states has its own case.

## How these cases are observed

- **Two kinds of run.** **K+** is a run of the story with `VNPAY_TMN_CODE` and `VNPAY_HASH_SECRET` both set to the sandbox values; the verifier knows both (called **TMN** and **SECRET** below). **K−** is a run with at least one of them unset. Almost every case is a K+ case. The cases marked **K−** need the keys unset; a verifier who can only run K+ reports them UNVERIFIED (instrument lacked: a way to unset a setting of the running backend), and says so, rather than failing them. Where a K− page state can be produced in a K+ run by answering `GET /api/v1/payment-options` with `is_available:false` (TC-24) the case says so. Without SECRET no case can sign a notice: every case that signs is then UNVERIFIED (instrument lacked: the hash secret).
- **Windows.** Default 1280×800. Phone cases say 390×844 (and 320×640 where the criterion names it).
- **Addresses.** The API is read through the gateway as `/api/v1/...` (the backend itself mounts `/v1/...`). VNPay's notice address is `/api/v1/payments/vnpay/ipn`. Pages: `/checkout`, `/orders/<id>`, `/payment/return`, `/track`, `/shop`.
- **Money and dates.** `vi-VN` writes `3.140.000 ₫` (the space before `₫` may be a no-break space: compare with either). API amounts are integers of VND; VNPay's own unit is the amount × 100 (`vnp_Amount=314000000`). `vnp_CreateDate`, `vnp_ExpireDate` and `vnp_PayDate` are `yyyyMMddHHmmss` in Vietnam time (UTC+7); the verifier converts the stored `timestamptz` to that zone before comparing.
- **Selectors the story fixes (result page).** `[data-testid='payment-state']` is the h1 and carries `data-state` = one of `paid`, `confirming`, `timed-out`, `not-completed`, `cancelled`, `no-match`, `check-failed`; `[data-testid='payment-line']` is the state line; `[data-testid='payment-view-order']`, `[data-testid='payment-try-again']`, `[data-testid='payment-track']`, `[data-testid='payment-keep-shopping']`. The alert is the element with `role="alert"`. The loading placeholder carries `aria-busy="true"`. Elsewhere: `[data-testid='place-order']`, `[data-testid='checkout-total']`, `[data-testid='pay-vnpay']`, `[data-testid='add-to-bag']`, `[data-testid^='size-']`, `[data-testid='order-status']` (the order page's h1), `[data-testid='pay-now']`, `[data-testid='pay-now-error']`, and `#checkout-name`, `#checkout-phone`, `#checkout-address`, `#checkout-city`, `#checkout-note`. No Tailwind class is relied on.
- **FORM** (typed in every UI case that places an order): name `Lan Nguyen`, phone `0912345678`, street address `12 Example Street, Ward 1`, city `Da Nang`, note `Gate code 4821`.
- **REQ / PLACE** (the body of every API case that places an order), sent as `POST /api/v1/orders` with `Content-Type: application/json` and a fresh header `Idempotency-Key: vp-<case>-<n>`:

```json
{ "items": [ { "product": "samba-og-w-core-black", "size": "38", "qty": 1 } ],
  "delivery": "standard", "payment": "vnpay",
  "customer": "Lan Nguyen", "phone": "0912345678",
  "address": "12 Example Street, Ward 1", "city": "Da Nang", "note": "Gate code 4821" }
```

  In a K+ run it answers 201 with `order` (`id`, `number`, `total`, …), `token` and a non-empty `payment_url`. **PLACE** means this request; the order id and token are read from the answer. The first order after RESET is `doi-24818` / `DOI-24818`; its first attempt's reference is `DOI-24818-1`.
- **PRICE(shoe, v)** — `UPDATE products SET price = <v> WHERE id = '<shoe>'`; the old price is read first and put back. **P3140** means PRICE(`samba-og-w-core-black`, 3140000): one pair of size `38` then totals exactly 3.140.000 ₫ (delivery is free).
- **SHELF(shoe, size, n)** — `UPDATE product_sizes SET pairs = <n> WHERE product_id = '<shoe>' AND size = '<size>'`; the old value is read first and put back. The pairs on the shelf are read with `SELECT pairs FROM product_sizes WHERE product_id = … AND size = …`; the shoe's public record (`GET /api/v1/products/<shoe>`) carries `sizes_low` (sizes with 1–2 pairs), `sizes_out` (sizes with 0 pairs) and `stock` (the sum over all sizes).
- **RESET** (before and after every group that placed orders): `DELETE FROM orders WHERE is_sample = false` (items, tokens, attempts and notices go with them); `UPDATE counters SET value = 24817 WHERE name = 'order_number'`; every PRICE, SHELF, `is_sample`, `status`, `paid` and `placed_at` edit of this file put back; any renamed table renamed back; the database made available again.
- **SET(order, "<assignments>")** — `UPDATE orders SET <assignments> WHERE id = '<order>'`. Nobody can move an order along through a dealer screen yet, so `status`, `paid`, `placed_at`, `is_sample` and `payment_method_id` are set this way, as the story says.
- **SNAPSHOT(order)** — `SELECT status, paid, total, payment_method_id, placed_at, updated_at FROM orders WHERE id = '<order>'` (`updated_at` is moved by a trigger on any update, so an equal `updated_at` proves nothing was written); `SELECT count(*) FROM payment_attempts WHERE order_id = '<order>'`; `SELECT count(*), count(*) FILTER (WHERE is_success) FROM payment_notices n JOIN payment_attempts a ON a.id = n.attempt_id WHERE a.order_id = '<order>'`; and the `pairs` of each size the order holds. "Unchanged" means the snapshot is equal before and after.
- **ADDR(url)** — the `payment_url` parsed as a URL: its origin and path, and its query as a list of `vnp_*` name/value pairs. **REF** is its `vnp_TxnRef`.
- **SIGN(params)** — the signature VNPay's algorithm gives: take every `vnp_` parameter except `vnp_SecureHash` and `vnp_SecureHashType`, sort them by name (ascending, byte order), write each as `name=value` with the value URL-encoded the form way (a space is `+`; the sample values below use only letters, digits, spaces and `-`, so `encodeURIComponent` with `%20` replaced by `+` is the same), join with `&`, and take the lower-case hex HMAC-SHA512 keyed with SECRET. The verifier computes it in the page with `crypto.subtle` (an `interact_page` script), so every case that signs is `Check: interact_page`.
- **NOTICE(ref, changes)** — the request VNPay sends, as `GET /api/v1/payments/vnpay/ipn?<query>` with **no cookie and no `X-Order-Token`**, query: `vnp_TmnCode=TMN`, `vnp_Amount=<order total × 100>`, `vnp_BankCode=NCB`, `vnp_BankTranNo=VNP14012345`, `vnp_CardType=ATM`, `vnp_OrderInfo=Thanh toan don hang <number>`, `vnp_PayDate=<now, Vietnam time>`, `vnp_ResponseCode=00`, `vnp_TransactionNo=<a number no earlier notice used, 8 digits>`, `vnp_TransactionStatus=00`, `vnp_TxnRef=<ref>`, `vnp_SecureHash=SIGN(…)`. "Changes" are applied **before** signing unless the case says "altered after signing". The answer is read as status, `Content-Type` and the JSON body.
- **DUE(order, age)** — `SET(order, "placed_at = now() - interval '<age>'")`.
- **COMPANION and "the check has run".** The expiry check is an in-process timer that runs at least once a minute; no instrument runs it on demand. A case that waits for it adds a **companion** order C: placed through PLACE, unpaid, status `placed`, then DUE(C, '31 minutes'). "The check has run" means C's `status` has become `cancelled`, polled with `SELECT status FROM orders WHERE id = C` every 5 s for up to 90 s. If C is not cancelled after 90 s every case waiting for the check fails (SC-20: the check does not run at least once a minute). The case under test is read **immediately** after C is seen cancelled.
- **HOLD(order)** — the browser opened for a UI case holds that order's token the way a real customer's does. The order is placed **through the page in that browser**: add the pair to the bag from `/shop/<shoe>` (size pill, `[data-testid='add-to-bag']`), open `/checkout`, type FORM, press `[data-testid='place-order']`. In a K+ run the browser then leaves for VNPay's address; the verifier reads that address from the navigation request (VNPay's host page need not load) and navigates back to the shop's own origin, whose `localStorage` still holds the order's entry. Once the first order has shown the shape of that entry (the one `localStorage` entry holding the order's id, number, placed time and token), a verifier may write an entry of that same shape for an order it placed through the API; if the shape cannot be learned the case is placed through the page. A case that says "a fresh browser" starts with empty `localStorage`, `sessionStorage` and cookies.
- **RETURN(ref, code)** — the page `/payment/return?vnp_Amount=<total×100>&vnp_BankCode=NCB&vnp_ResponseCode=<code>&vnp_TxnRef=<ref>&vnp_TransactionStatus=<code>&vnp_SecureHash=<any 128 hex characters>` as VNPay's return redirect writes it. The page must ignore the signature.
- **Stubbing in the browser.** "STUB(path, answer)" intercepts the page's request to the API path and answers it with the status and body the case gives; "DELAY(path, ms)" lets the real request through and holds the real answer; "ABORT(path)" fails the request at the network. The log of requests records, for each, the method, URL, headers and body. A case that needs one and the instrument cannot intercept is UNVERIFIED (instrument lacked: request interception) and says so.
- **Timings.** "At 1 s" means one second after the navigation started, read from the page. Waits of 6 s, 30 s and 60 s are real waits of the page; a verifier whose instrument cannot wait that long reports the case UNVERIFIED (instrument lacked: a wait of that length), not failed.

## Ambiguities found in the documents (reported, not silently resolved)

1. **Reference numbers.** The story writes the two references as `DOI-24818-1` and `DOI-24818-2`; the contract (§3.10 step 5, §3.13) builds attempt 1 when the order is placed and attempt `n+1` on every later ask. The cases pin `-1` on the placing answer and `-2`, `-3` on the next two "Pay now" calls, and otherwise only that references differ.
2. **`?amount=1000` on `POST /v1/orders/{id}/payment`.** SC-1 says it "changes nothing"; the contract defines no query parameter and "A request field the contract does not define is rejected with 400". TC-02 accepts either outcome (200 carrying the stored amount, or 400 `bad_request` with no attempt stored) and fails only a 200 carrying `vnp_Amount=100000` or any other server error. A body field (`{"amount":1000}`) is a different matter: 400 (TC-84). `total`, `delivery_fee`, `items[].price` and `items[].unit_price` in the **placing** body are tolerated and discarded (§3.10), so TC-02 expects 201 for them.
3. **Messages other than `00` and `02`.** The story pins `Confirm Success`, `Order already confirmed` and `Unknown error`; the contract (§3.17) pins `Invalid signature` (97), `Order not found` (01) and `Invalid amount` (04). The cases assert the code exactly and the message as the contract writes it.
4. **A notice that is wrong in two ways.** SC-14 fixes the order of the checks (signature, reference, amount, already paid); the cases pin four pairs of faults (TC-35).
5. **A paid and cancelled order, "Pay now".** SC-6 and the Order page story give an order that is both the cancelled sentence ("paid or not"); no case in this file depends on which of the two a paid-and-cancelled order says, only on 409 `conflict`.
6. **Unpaid shipped or delivered within 30 minutes.** The contract (§3.13) refuses only paid, cancelled, non-online and expired orders, so an unpaid shipped or delivered VNPay order inside its 30 minutes gets a fresh address (TC-17). A dealer cannot realistically ship inside 30 minutes; the case pins the contract, not the likelihood.
7. **Extra query fields on the notice that do not start with `vnp_`.** The contract accepts every `vnp_*` field and says nothing about others (signed fields are the `vnp_` ones). No case sends a non-`vnp_` field.
8. **The 29-minute case has no margin.** An order that is 29 minutes old becomes 30 minutes old while the verifier waits for the check. TC-49 reads `updated_at` so a cancellation at or after the 30th minute is not mistaken for a premature one.
9. **No outbound call.** `PaymentURL` is built locally; no instrument observes the API's outbound traffic over HTTP, so the case that would say "no call to VNPay is made" is covered only by the fact that a K+ run on a `.localhost` address builds an address at all (TC-03).
10. **A real VNPay sandbox payment.** On a `.localhost` run VNPay's servers cannot call the notice address, so the signed notice stands in (SC-36). A real payment on VNPay's sandbox page with the test card is possible only on a public deployment and is not required by any case below.

---

## A. The payment address, "Pay now", and the keys (SC-1 … SC-10)

**Scenario**: TC-01 — the address carries the stored total, the terminal code, VND and the order number
**Given**: K+; RESET; P3140; PLACE one pair of `samba-og-w-core-black` size `38` (order O1 `doi-24818`, token T1, `order.total` 3140000); `SELECT total FROM orders WHERE id = 'doi-24818'` reads 3140000.
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with `X-Order-Token: T1` and no body; the `payment_url` of the placing answer and the one of this answer are both parsed with ADDR.
**Then**: The POST answers 200 with a JSON body whose only key is `payment_url`. In both addresses: `vnp_TmnCode` = TMN; `vnp_Amount` = `314000000`; `vnp_CurrCode` = `VND`; `vnp_TxnRef` starts with `DOI-24818-` (placing: `DOI-24818-1`); `vnp_OrderInfo` contains `DOI-24818`. The amount is the stored total × 100 and no other number.
Traces: SC-1 (PAY-001 AC-1)
Check: fetch_url

**Scenario**: TC-02 — an amount sent by the browser changes nothing
**Given**: K+; RESET; P3140; SNAPSHOT of the shelf.
**When**: (a) PLACE with the extra body members `"total": 1000`, `"delivery_fee": 5` and, inside the item, `"unit_price": 1`, `"price": 1`; (b) with that order's token, `POST /api/v1/orders/doi-24818/payment?amount=1000` with no body; (c) the same URL without the query.
**Then**: (a) answers 201, `order.total` is 3140000 and `order.delivery_fee` is 0 (the members are tolerated and discarded), and its address has `vnp_Amount=314000000`. (b) answers either 200 whose address has `vnp_Amount=314000000` or 400 `{"error":"bad_request",…}` with no attempt row stored for it; it never answers 200 with `vnp_Amount=100000` and never 500. (c) answers 200 with `vnp_Amount=314000000`. `SELECT total FROM orders` still reads 3140000.
Traces: SC-1 (PAY-001 AC-2, item 3)
Check: fetch_url

**Scenario**: TC-03 — the address's host, path and fixed parameters
**Given**: K+; the setting `VNPAY_PAYMENT_URL` is read from the run (default `https://sandbox.vnpayment.vn/paymentv2/vpcpay.html`); O1 placed (T1).
**When**: `POST /api/v1/orders/<id>/payment` with T1; the address is parsed with ADDR.
**Then**: The address's origin and path equal `VNPAY_PAYMENT_URL`'s. The query holds exactly these names (and no other): `vnp_Version` = `2.1.0`, `vnp_Command` = `pay`, `vnp_TmnCode`, `vnp_Amount`, `vnp_CurrCode`, `vnp_TxnRef`, `vnp_OrderInfo` (ASCII only: `Thanh toan don hang DOI-24818`), `vnp_OrderType` = `other`, `vnp_Locale`, `vnp_ReturnUrl`, `vnp_IpAddr` (non-empty), `vnp_CreateDate` (14 digits), `vnp_ExpireDate` (14 digits), `vnp_SecureHash` (128 hexadecimal characters, the last parameter). No parameter holds SECRET.
Traces: SC-1 (PAY-001 item 2), SC-38
Check: fetch_url

**Scenario**: TC-04 — every attempt has its own reference and its own stored row
**Given**: K+; RESET; O1 placed (T1) — attempt 1 is the placing answer's address, reference R1.
**When**: `POST /api/v1/orders/doi-24818/payment` with T1 is sent twice in turn (references R2, R3); `SELECT reference FROM payment_attempts WHERE order_id = 'doi-24818' ORDER BY created_at` is read.
**Then**: R1 = `DOI-24818-1`, R2 = `DOI-24818-2`, R3 = `DOI-24818-3`; the three differ and each is at most 100 characters; the table holds exactly three rows with those references and `provider = 'vnpay'`, `amount` = the order's total, and `expires_at` = `placed_at + 30 minutes`.
Traces: SC-2 (PAY-001 AC-3, item 5)
Check: fetch_url

**Scenario**: TC-05 — ten asks at the same moment never share a reference
**Given**: K+; RESET; O1 placed (T1).
**When**: Ten `POST /api/v1/orders/doi-24818/payment` requests with T1 are issued in one batch (`Promise.all` in one `interact_page` script).
**Then**: All ten answer 200; the ten `vnp_TxnRef` values are pairwise different; none answers 409, 500 or 503; `SELECT count(DISTINCT reference), count(*) FROM payment_attempts WHERE order_id = 'doi-24818'` reads 11 and 11 (the placing's attempt plus ten).
Traces: SC-2 (PAY-001 AC-3, item 5)
Check: interact_page

**Scenario**: TC-06 — a success notice for the first reference still finds the order and pays it
**Given**: K+; RESET; O1 placed with first reference R1 (the placing's address); two more attempts R2, R3 made with "Pay now".
**When**: NOTICE(R1, success, amount = total × 100) is sent.
**Then**: The answer is 200 `{"RspCode":"00","Message":"Confirm Success"}`; `SELECT paid, status FROM orders WHERE id = 'doi-24818'` reads `true` and `confirmed`.
Traces: SC-2 (PAY-001 item 5)
Check: interact_page

**Scenario**: TC-07 — the signature verifies with the shop's secret and with no other
**Given**: K+; O1 placed (T1); the address from the placing answer.
**When**: In a page, SIGN is computed over the address's `vnp_` parameters (except `vnp_SecureHash`) with SECRET, then with `SECRET + "x"`, then with the amount parameter changed to `vnp_Amount=100000` and SECRET.
**Then**: With SECRET the result equals the address's `vnp_SecureHash` exactly (lower-case hex, 128 characters); with `SECRET + "x"` it differs; with the changed amount it differs.
Traces: SC-3 (PAY-001 AC-4, item 4)
Check: interact_page

**Scenario**: TC-08 — the return address is the shop's `/payment/return`
**Given**: K+; O1 placed (T1); the origin O the verifier reaches the shop on (for example `http://shop.localhost`), and the run's `PUBLIC_BASE_URL` if one is set.
**When**: `POST /api/v1/orders/<id>/payment` with T1; `vnp_ReturnUrl` is read from the address.
**Then**: `vnp_ReturnUrl` equals `<PUBLIC_BASE_URL>/payment/return` when `PUBLIC_BASE_URL` is set, otherwise `<O>/payment/return` (the scheme and host the request arrived with); in both it ends exactly in `/payment/return` and holds no query. The override itself (a changed `PUBLIC_BASE_URL`) is observed only when the verifier can run with it; otherwise that half is UNVERIFIED (instrument lacked: a way to change a setting of the running backend).
Traces: SC-3 (PAY-001 AC-5)
Check: fetch_url

**Scenario**: TC-09 — an order placed 10 minutes ago expires in 20 minutes
**Given**: K+; O1 placed (T1); DUE(O1, '10 minutes'); `placed_at` read.
**When**: `POST /api/v1/orders/<id>/payment` with T1; `vnp_CreateDate` and `vnp_ExpireDate` are read from the address.
**Then**: The answer is 200. `vnp_ExpireDate` equals `placed_at + 30 minutes` written as `yyyyMMddHHmmss` in Vietnam time; it is 20 minutes (± 1 minute) after the current Vietnam time and exactly 30 minutes after `placed_at`; `vnp_CreateDate` is the current Vietnam time (± 1 minute). The stored attempt's `expires_at` equals the same moment.
Traces: SC-4 (PAY-001 AC-6; PAY-003 item 6)
Check: fetch_url

**Scenario**: TC-10 — an address is still built just inside the deadline and refused just past it
**Given**: K+; three orders placed: O1, O2, O3 (tokens T1, T2, T3), all unpaid; DUE(O1, '29 minutes 30 seconds'); O2 is set `status = 'shipped'` and DUE(O2, '31 minutes') (a shipped order is not touched by the expiry check, so the refusal can be read without a race); O3 is DUE(O3, '31 minutes') with status `placed`; SNAPSHOT of each.
**When**: Within 20 seconds of arranging, `POST /api/v1/orders/<id>/payment` is sent for O1, O2 and O3 with their tokens and no body.
**Then**: O1 answers 200 with an address whose `vnp_ExpireDate` equals `placed_at + 30 minutes` (no more than 30 seconds from now). O2 and O3 each answer 409 `{"error":"conflict","message":<sentence>}` (O3 may also be answered 409 because the check cancelled it first; either way 409 `conflict`, never 200 and never 500); for O2 the attempt count is unchanged. O1 has one more attempt row; O2 and O3 have none added.
Traces: SC-4 (PAY-001 AC-6, boundary row)
Check: fetch_url

**Scenario**: TC-11 — placing with VNPay answers 201 with an address and keeps the placing contract
**Given**: K+; RESET; SHELF(`samba-og-w-core-black`, `38`, 5); P3140.
**When**: PLACE is sent once; `Location`, `Cache-Control` and the body are read; then the same request is sent again with the **same** `Idempotency-Key`.
**Then**: The first answers 201 with `Location: /v1/orders/doi-24818`, `Cache-Control: no-store`, `order.status` `placed`, `order.paid` `false`, `token` of at least 22 URL-safe characters and `payment_url` a non-empty string beginning with `VNPAY_PAYMENT_URL`; the shelf for size `38` fell by 1. The replay answers 200 with `Idempotent-Replayed: true`, the same `order.id`, a new `token` and a fresh `payment_url` whose `vnp_TxnRef` differs from the first; the shelf did not fall again.
Traces: SC-5 (PAY-001 AC-9; ORD-006 AC-7), contract (POST /v1/orders)
Check: fetch_url

**Scenario**: TC-12 — checkout with VNPay ends at VNPay's address whose amount is the Total shown
**Given**: K+; RESET; P3140; a 1280×800 browser with `samba-og-w-core-black` size `38` added to the bag (`[data-testid='add-to-bag']`) and `/checkout` open; FORM typed.
**When**: The text of `[data-testid='checkout-total']` is read; `[data-testid='place-order']` is pressed.
**Then**: The browser requests `POST /api/v1/orders` (201 in the request log) and then navigates to an address whose origin and path are `VNPAY_PAYMENT_URL`'s; that address's `vnp_Amount` equals the number in the Total × 100 (`3.140.000 ₫` → `314000000`) and the stored `orders.total`; the bag count `[data-testid='bag-count']` reads 0 or is absent when the shop's own origin is opened again.
Traces: SC-5 (PAY-001 AC-9; ORD-006 AC-7; ORD-009 AC-3)
Check: interact_page

**Scenario**: TC-13 — "Pay now" answers a fresh address for an unpaid VNPay order, in every open status
**Given**: K+; RESET; O1 placed (T1); the statuses `placed`, `confirmed`, `packed` set in turn with SET (unpaid, placed seconds ago); SNAPSHOT.
**When**: `POST /api/v1/orders/doi-24818/payment` with T1 and no body, once per status.
**Then**: Each answers 200 `{"payment_url": …}` with the address of TC-01 and a reference never used before for this order; the attempt count rises by one per call; the order's `status`, `paid` and `updated_at` are unchanged.
Traces: SC-6 (PAY-001 item 1; ORD-007 AC-14)
Check: fetch_url

**Scenario**: TC-14 — on the order page "Pay now" is disabled while it waits and then goes to the fresh address
**Given**: K+; HOLD(O1) (unpaid, status `placed`, placed 10 minutes ago by DUE); a 1280×800 browser on `/orders/<id>`; DELAY(`/api/v1/orders/<id>/payment`, 1500).
**When**: `[data-testid='pay-now']` is read, pressed once, and read again at 0.5 s; the browser's next navigation is read.
**Then**: `[data-testid='pay-now']` is visible and enabled before the press; at 0.5 s after the press it is `disabled`; the request log shows exactly one `POST /api/v1/orders/<id>/payment` with the token in `X-Order-Token`; then the browser navigates to an address on `VNPAY_PAYMENT_URL` whose `vnp_TxnRef` is a reference the order did not have before. The story edits no line of the order page: the page shows no new element beyond what the Order page story fixed.
Traces: SC-6 (PAY-001 item 1; ORD-007 AC-14)
Check: interact_page

**Scenario**: TC-15 — a paid, a cancelled and a non-VNPay order still answer their three different 409 sentences
**Given**: K+; RESET; O1 placed and set `paid = true` (status `placed`); O2 placed and set `status = 'cancelled'`; a seeded sample order O3 with `payment_method_id` `cod` or `bank` and `NOT paid` (found with `SELECT id FROM orders WHERE is_sample AND NOT paid AND payment_method_id IN ('cod','bank','momo') AND status <> 'cancelled' LIMIT 1`), opened with a SEEDTOKEN row `INSERT INTO order_tokens (order_id, token_hash) VALUES ('<O3>', sha256('tc-seed-token-0001'::bytea))` and the header `X-Order-Token: tc-seed-token-0001` (if the backend hashes tokens in another form the row opens nothing and this part is UNVERIFIED: instrument lacked: a way to mint a token for a seeded order); SNAPSHOT of each.
**When**: `POST /api/v1/orders/<id>/payment` is sent for O1, O2, O3 with their own tokens and no body.
**Then**: Each answers 409 `{"error":"conflict","message":<sentence>}` with no `errors` key; the three sentences are pairwise different; the O1 sentence contains `paid`, the O2 sentence contains `cancel`, the O3 sentence contains `online` (any case); none equals `Online payment is not available yet.`. SNAPSHOT of each is equal before and after, and no attempt row was added. The seeded order's token row is deleted afterwards.
Traces: SC-6 (PAY-001 item 7; ORD-007 AC-17, AC-18, AC-19)
Check: fetch_url

**Scenario**: TC-16 — with no token, another order's token or an unknown id the answer is the one 404
**Given**: K+; RESET; O1 (T1) and O2 (T2) placed; SNAPSHOT(O1).
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with: (a) no `X-Order-Token`; (b) `X-Order-Token: T2`; (c) an empty token; and `POST /api/v1/orders/doi-99999/payment` with T1; each is also sent as `GET` to `/api/v1/orders/<id>` with the same headers.
**Then**: Every POST answers 404 `{"error":"not_found","message":S}` with the one same sentence S and the same body as the matching GET; no body holds an id, a token or `DOI-`; SNAPSHOT(O1) is equal before and after and no attempt row was added.
Traces: SC-6 (PAY-001 permission row; ORD-007 AC-20)
Check: fetch_url

**Scenario**: TC-17 — "Online payment is not available yet." is never given for a VNPay order once the keys are set
**Given**: K+; RESET; five orders placed, each unpaid and VNPay, set to the statuses `placed`, `confirmed`, `packed`, `shipped`, `delivered`, all placed seconds ago; each with its token.
**When**: `POST /api/v1/orders/<id>/payment` is sent for each with its token and no body.
**Then**: Each answers 200 with an address (the contract refuses only a paid, cancelled, non-online or expired order); none answers the sentence `Online payment is not available yet.`.
Traces: SC-6 (PAY-001 item 7; ORD-007 item 9)
Check: fetch_url

**Scenario**: TC-18 — when the attempt cannot be stored the order still exists and its address is empty
**Given**: K+; RESET; `ALTER TABLE payment_attempts RENAME TO payment_attempts_x` (renamed back afterwards); SHELF(`samba-og-w-core-black`, `38`, 5) read as N.
**When**: PLACE is sent.
**Then**: The answer is 201 (the order was saved) with `payment_url` exactly `""` and a `token`; `SELECT status, paid FROM orders WHERE id = 'doi-24818'` reads `placed` and `false`; the shelf for size `38` is N − 1; no body contains `payment_attempts`, `relation` or `SELECT`. With the table renamed back, `POST /api/v1/orders/doi-24818/payment` with the token answers 200 with an address (the order is payable).
Traces: SC-7 (ORD-006 AC-8, item 5; PAY-001 item 1)
Check: fetch_url

**Scenario**: TC-19 — the browser lands on the order page and shows "Pay now" when the address is empty
**Given**: K+; `payment_attempts` renamed as in TC-18 (or, if no instrument can rename a table, the answer of `POST /api/v1/orders` stubbed to the 201 body of TC-18 with `payment_url` `""`); a 1280×800 browser with a pair in the bag and `/checkout` open; FORM typed.
**When**: `[data-testid='place-order']` is pressed; the final URL and `[data-testid='pay-now']` are read.
**Then**: The browser lands on `/orders/doi-24818?placed=1` (not on VNPay's host); `[data-testid='pay-now']` is visible and enabled; the page's heading contains `Thank you`.
Traces: SC-7 (ORD-006 AC-8, item 5)
Check: interact_page

**Scenario**: TC-20 — K−: the server starts, `/healthz` answers 200 and VNPay is listed as unavailable
**Given**: K− (`VNPAY_TMN_CODE` or `VNPAY_HASH_SECRET` unset; the run says which).
**When**: `GET /healthz` and `GET /api/v1/payment-options` are sent.
**Then**: `/healthz` answers 200. `/api/v1/payment-options` answers 200 with `items` holding exactly one item whose `id` is `vnpay` and whose `is_available` is `false` (`online` true; `name`, `note`, `icon` non-empty strings), and `total` 1.
Traces: SC-8 (PAY-001 AC-8; PAY-005 AC-2; ORD-006 AC-11)
Check: fetch_url

**Scenario**: TC-21 — K−: placing answers 503 after the field checks, before any write
**Given**: K−; RESET (the counter is read as C0); SHELF read for `samba-og-w-core-black` size `38` as N.
**When**: (a) PLACE with a valid body; (b) PLACE with `"customer": ""`; (c) PLACE with `"payment": "cod"`.
**Then**: (a) answers 503 `{"error":"payment_unavailable","message":<sentence>}` with no `errors` key. (b) answers 422 `invalid` with `errors.customer` (the field checks come first); (c) answers 422 `invalid` with `errors.payment` whatever the settings. After all three: `SELECT count(*) FROM orders WHERE NOT is_sample` reads 0, the counter still reads C0, the shelf for size `38` is still N.
Traces: SC-8 (PAY-001 AC-10, item 6; ORD-006 AC-11)
Check: fetch_url

**Scenario**: TC-22 — K−: "Pay now" checks access first, then state, then the keys
**Given**: K−; three orders **inserted** with `set_db_state` (no order can be placed in K−): an `orders` row each with its `order_items` line and an `order_tokens` row holding `sha256` of a token the case knows: O1 unpaid VNPay `placed` (token T1), O2 paid VNPay `placed` (T2); SNAPSHOT of each.
**When**: `POST /api/v1/orders/<O1>/payment` is sent with (a) T1; (b) no token; (c) T2; and `POST /api/v1/orders/<O2>/payment` with (d) T2.
**Then**: (a) answers 503 `payment_unavailable`; (b) and (c) answer 404 `not_found` (a stranger gets the 404 even though the keys are missing); (d) answers 409 `conflict` (a paid order is still 409). SNAPSHOT of each is equal before and after; no attempt row exists.
Traces: SC-8 (PAY-001 AC-8, item 6; ORD-007 item 9)
Check: fetch_url

**Scenario**: TC-23 — K−: the notice address answers 97 to everything
**Given**: K−; an order inserted as in TC-22 with an attempt row `reference` `DOI-24818-1`.
**When**: `GET /api/v1/payments/vnpay/ipn` is sent with (a) no query; (b) a query naming `vnp_TxnRef=DOI-24818-1`, a correct-looking amount and `vnp_SecureHash` of 128 zeros.
**Then**: Both answer HTTP 200 with a body whose `RspCode` is `97` and whose `Message` is `Invalid signature`; the order is unpaid and unchanged.
Traces: SC-8 (PAY-001 AC-8; services §3.17 notes)
Check: fetch_url

**Scenario**: TC-24 — with a key missing, checkout shows the unavailable line and a disabled "Place order"
**Given**: Either a K− run, or a K+ run with `GET /api/v1/payment-options` stubbed to answer 200 `{"items":[{"id":"vnpay","name":"VNPay — ATM, Visa, Mastercard","note":"Pay by card or banking app on VNPay's secure page.","icon":"card","online":true,"is_available":false}],"total":1,"page":1,"limit":100}`; a 1280×800 browser with a pair in the bag; `/checkout` loaded (the stubbed half is the page's reaction only; the backend half is TC-20 to TC-23).
**When**: The VNPay row `[data-testid='pay-vnpay']`, the text under it, and `[data-testid='place-order']` are read; `[data-testid='place-order']` is clicked (by its coordinates if it is disabled).
**Then**: The VNPay row is visible; under it the text is exactly `Online payment is not available right now. Please try again later.` in an element with `role="alert"`; `[data-testid='place-order']` has the `disabled` property true; the click sends no `POST /api/v1/orders` (none in the request log).
Traces: SC-9 (PAY-001 AC-7; ORD-001 AC-14; ORD-006 AC-10)
Check: interact_page

**Scenario**: TC-25 — when the shop answers 503 to placing, the line appears and the bag and typed details are kept
**Given**: K+; a 1280×800 browser with one pair in the bag, `/checkout` loaded with the keys set (payment options answer `is_available:true`), FORM typed; STUB(`POST /api/v1/orders`, 503 `{"error":"payment_unavailable","message":"x"}`).
**When**: `[data-testid='place-order']` is pressed.
**Then**: The page stays on `/checkout`; the text `Online payment is not available right now. Please try again later.` is shown in an element with `role="alert"` under the VNPay row; `#checkout-name`, `#checkout-phone`, `#checkout-address`, `#checkout-city` and `#checkout-note` still hold the FORM values; `[data-testid='bag-count']` still shows the same count; `localStorage["agentteam:cart"]` still holds the pair.
Traces: SC-9 (PAY-001 AC-7; ORD-006 AC-10, item 6)
Check: interact_page

**Scenario**: TC-26 — with both keys set the shop lists VNPay alone and available
**Given**: K+.
**When**: `GET /api/v1/payment-options?anything=1` is sent.
**Then**: 200 `{"items":[…],"total":1,"page":1,"limit":100}` with exactly one item: `id` `vnpay`, `online` true, `is_available` true, non-empty `name` `VNPay — ATM, Visa, Mastercard`, `note` and `icon`. No item has `id` `cod`, `bank` or `momo`. The parameter is ignored (the contract says any parameter is ignored on this endpoint).
Traces: SC-10 (PAY-005 AC-1; ORD-006 AC-1, AC-5), contract (GET /v1/payment-options)
Check: fetch_url

**Scenario**: TC-27 — checkout step 03 shows one selected VNPay row and an enabled "Place order"
**Given**: K+; a 1280×800 browser with a pair in the bag; `/checkout` loaded; FORM typed.
**When**: The payment step is read; `[data-testid='place-order']` is read.
**Then**: Step 03 holds exactly one payment row, `[data-testid='pay-vnpay']`, with the visible text `VNPay — ATM, Visa, Mastercard`, selected (a checked radio or `aria-checked="true"` or `aria-current`); the page text holds none of `Cash on delivery`, `Bank transfer`, `MoMo`; `[data-testid='place-order']` is not disabled; no `role="alert"` line about payment is shown.
Traces: SC-10 (PAY-005 AC-1; ORD-006 AC-1, AC-2, AC-5; owner decision)
Check: interact_page

**Scenario**: TC-28 — a way to pay other than VNPay is refused 422 and creates nothing
**Given**: K+; RESET; the counter and the shelf for `samba-og-w-core-black` size `38` read as C0 and N.
**When**: PLACE is sent three times with `"payment"` set to `cod`, `bank` and `momo`, and a fourth time with `"payment": "VNPay"` (a wrong-case id).
**Then**: Each answers 422 `{"error":"invalid",…}` with `errors.payment` present (a non-empty sentence) and no 201; no order exists (`SELECT count(*) FROM orders WHERE NOT is_sample` is 0), the counter still reads C0, the shelf is still N.
Traces: SC-10 (ORD-006 AC-5; owner decision "no order without VNPay")
Check: fetch_url

---

## B. VNPay's notice is the only thing that marks an order paid (SC-11 … SC-19)

Every case in this group is K+ (a notice cannot be signed otherwise) and signs with SIGN in the page, so each is `interact_page`; the database reads use `set_db_state`. NOTICE carries no cookie and no token, so "no session and no token" is true by construction. After every case RESET.

**Scenario**: TC-29 — a signed success notice pays the order, confirms it and keeps VNPay's data
**Given**: K+; RESET; P3140; O1 `doi-24818` placed (status `placed`, total 3140000, reference R1); SNAPSHOT(O1).
**When**: NOTICE(R1) is sent with `vnp_Amount=314000000`, `vnp_ResponseCode=00`, `vnp_TransactionStatus=00`, `vnp_BankCode=NCB`, `vnp_TransactionNo=14012345`, `vnp_PayDate` = now (Vietnam time).
**Then**: HTTP 200 and the JSON body is exactly `{"RspCode":"00","Message":"Confirm Success"}`. `SELECT paid, status FROM orders WHERE id = 'doi-24818'` reads `true` and `confirmed`. `SELECT transaction_no, bank_code, amount, response_code, transaction_status, is_success, paid_time FROM payment_notices` for the order's attempts holds one row: `14012345`, `NCB`, `3140000`, `00`, `00`, `true`, and `paid_time` equal to the sent pay time in Vietnam time. `payment_notices` has no column that holds a card number (`SELECT column_name FROM information_schema.columns WHERE table_name = 'payment_notices'` lists none of `card`, `pan`, `number` beyond `transaction_no`).
Traces: SC-11 (PAY-002 AC-1, AC-2, items 1–4; Privacy NFR)
Check: interact_page

**Scenario**: TC-30 — an order past "placed" is marked paid and keeps its status
**Given**: K+; RESET; three orders O1, O2, O3 placed; SET O1 `status = 'confirmed'`, O2 `status = 'packed'`, O3 `status = 'shipped'` (all unpaid, placed seconds ago).
**When**: A success NOTICE is sent for each order's first reference with its own amount.
**Then**: Each answers `{"RspCode":"00","Message":"Confirm Success"}`; each order reads `paid = true` and its status is as before (`confirmed`, `packed`, `shipped`); each has one success notice.
Traces: SC-11 (PAY-002 item 4)
Check: interact_page

**Scenario**: TC-31 — the same notice a second time answers 02 and changes nothing
**Given**: K+; RESET; O1 placed; NOTICE(R1) sent once and answered `00`; SNAPSHOT(O1) (including `updated_at` and the notice count).
**When**: The identical request (same query, same signature) is sent again; then a third time.
**Then**: Both answer 200 `{"RspCode":"02","Message":"Order already confirmed"}`; SNAPSHOT(O1) is equal before and after both (one notice row, `updated_at` unchanged, same `status`).
Traces: SC-12 (PAY-002 AC-3, item 6)
Check: interact_page

**Scenario**: TC-32 — two copies, and twenty, sent at the same moment pay the order once
**Given**: K+; RESET; two orders O1 and O2 placed, unpaid.
**When**: Two identical NOTICE requests for O1 are issued in one batch (`Promise.all`); then twenty identical requests for O2 in one batch.
**Then**: For O1: exactly one answer is `{"RspCode":"00","Message":"Confirm Success"}` and the other is `{"RspCode":"02","Message":"Order already confirmed"}`. For O2: exactly one answer is `00` and the other nineteen are `02`. Each order reads `paid = true` once, and `SELECT count(*) FILTER (WHERE is_success)` of its notices is 1. No answer is anything but 200 with `00` or `02`.
Traces: SC-12 (PAY-002 AC-4, item 6)
Check: interact_page

**Scenario**: TC-33 — an invalid signature answers 97 and changes nothing
**Given**: K+; RESET; O1 placed, unpaid; SNAPSHOT(O1) with the notice count.
**When**: Seven requests are sent: (a) a valid NOTICE with the **last character of `vnp_SecureHash` changed** (altered after signing); (b) a NOTICE signed with `SECRET + "x"`; (c) a NOTICE with no `vnp_SecureHash` parameter at all; (d) a NOTICE with `vnp_SecureHash=` empty; (e) a NOTICE whose `vnp_TmnCode` is `NOTOURS`, signed correctly with SECRET; (f) a valid NOTICE with `vnp_Amount` changed after signing; (g) `GET /api/v1/payments/vnpay/ipn` with no query.
**Then**: Each answers HTTP 200 `{"RspCode":"97","Message":"Invalid signature"}`. SNAPSHOT(O1) is equal before and after all seven: unpaid, same status, same `updated_at`, no notice row.
Traces: SC-13 (PAY-002 AC-5, AC-10, item 7)
Check: interact_page

**Scenario**: TC-34 — a signature in upper-case hex and a `vnp_SecureHashType` field are accepted
**Given**: K+; RESET; O1 placed, unpaid; O2 placed, unpaid.
**When**: For O1 a success NOTICE is sent whose `vnp_SecureHash` is the **upper-case** form of the correct hex. For O2 a success NOTICE is sent with the extra field `vnp_SecureHashType=HmacSHA512` (signed over the fields **without** it, as VNPay does).
**Then**: Both answer `{"RspCode":"00","Message":"Confirm Success"}` and both orders read `paid = true` (the hex comparison is case-insensitive; `vnp_SecureHashType` is not part of the signed string).
Traces: SC-13 (PAY-002 item 7; services §3.17 check 1), SC-17
Check: interact_page

**Scenario**: TC-35 — a wrong amount answers 04, an unknown reference 01, and the checks run in order
**Given**: K+; RESET; O1 placed with total 3140000 (P3140), unpaid, reference R1; O2 placed and already paid by a success notice (reference R2); SNAPSHOT of both.
**When**: Seven correctly signed requests are sent (re-signed after each change) unless the line says otherwise: (a) O1, `vnp_Amount=100000`; (b) O1, `vnp_Amount=abc`; (c) O1, `vnp_Amount=314000000.5`; (d) `vnp_TxnRef=DOI-99999-1` with a correct-looking amount; (e) `vnp_TxnRef=DOI-99999-1` **and** `vnp_Amount=100000`; (f) O2's reference R2 with the **wrong** amount (already paid and wrong amount); (g) O1 with the right amount but its signature altered after signing **and** an unknown reference.
**Then**: (a), (b), (c) answer `{"RspCode":"04","Message":"Invalid amount"}`; (d) answers `{"RspCode":"01","Message":"Order not found"}`; (e) answers `01` (reference before amount); (f) answers `04` (amount before "already paid"); (g) answers `97` (signature before reference). All answers are HTTP 200. O1 is still unpaid with no notice row; O2's snapshot is equal before and after.
Traces: SC-14 (PAY-002 AC-6, AC-7, item 2)
Check: interact_page

**Scenario**: TC-36 — a failed or cancelled payment is acknowledged, kept, and leaves the order unpaid
**Given**: K+; RESET; O1 placed (status `placed`, reference R1), then "Pay now" made once for a second reference R2; SNAPSHOT(O1).
**When**: Correctly signed notices with the right amount are sent in turn: (a) R1 with `vnp_ResponseCode=24` and `vnp_TransactionStatus=02` (the customer cancelled); (b) the same request again; (c) R1 with `vnp_ResponseCode=00` and `vnp_TransactionStatus=02`; (d) R1 with `vnp_ResponseCode=07` and `vnp_TransactionStatus=00`; then (e) a success notice for R2.
**Then**: (a)–(d) each answer `{"RspCode":"00","Message":"Confirm Success"}`; after (d) the order reads `paid = false`, its status unchanged, and `payment_notices` holds the failed notices with `is_success = false` (the repeat in (b) is kept as a failed notice too, with no error). (e) answers `00`, after which the order reads `paid = true`, `status = 'confirmed'`, with exactly one `is_success` notice (for R2).
Traces: SC-15 (PAY-002 AC-8, item 5)
Check: interact_page

**Scenario**: TC-37 — a payment that arrives for a cancelled order is recorded and the order stays cancelled
**Given**: K+; RESET; SHELF(`samba-og-w-core-black`, `38`, 5); O1 placed (one pair of size `38`, shelf now 4); the dealer's cancel simulated as it is done by the order moves: SET O1 `status = 'cancelled'` and `UPDATE product_sizes SET pairs = pairs + 1 WHERE product_id = 'samba-og-w-core-black' AND size = '38'` (shelf 5); SNAPSHOT(O1).
**When**: A success NOTICE for R1 with the right amount is sent.
**Then**: The answer is `{"RspCode":"00","Message":"Confirm Success"}`; the order reads `paid = true` and `status = 'cancelled'`; one `is_success` notice is kept with the transaction number; the shelf for size `38` is still 5 (not taken again, not returned again).
Traces: SC-16 (PAY-002 AC-11, item 9; PAY-003 conflict row)
Check: interact_page

**Scenario**: TC-38 — every answer is HTTP 200 JSON with exactly `RspCode` and `Message`, never the error envelope
**Given**: K+; RESET; O1 placed, unpaid, reference R1.
**When**: One request is sent for each of `00` (a success NOTICE), `02` (it again), `01` (unknown reference), `04` (wrong amount), `97` (altered signature), and one success NOTICE for a **second** order with an extra signed field `vnp_Foo=1` added before signing; `Content-Type` and the key sets of the bodies are read.
**Then**: Every response has status 200 and `Content-Type` beginning `application/json`; every body has exactly the two keys `RspCode` and `Message` (no `error`, no `errors`, no `message` in lower case); the codes are `00`, `02`, `01`, `04`, `97`, and `00` for the notice with `vnp_Foo` (it is signed with the rest and is not refused with 400). A notice whose `vnp_Foo=1` is added **after** signing answers `97`.
Traces: SC-17 (PAY-002 AC-12, item 3; contract §3.17)
Check: interact_page

**Scenario**: TC-39 — with the database away the notice answers 99 and the same notice is processed normally afterwards
**Given**: K+; RESET; O1 placed, unpaid; a success NOTICE for R1 is built (signed) and kept; `set_db_available` takes the database away while the backend keeps running.
**When**: The kept notice is sent; then a notice with an altered signature is sent; the database is made available again; the first notice is sent again.
**Then**: The first answers HTTP 200 `{"RspCode":"99","Message":"Unknown error"}` (not 500, not 503, not the error envelope); the altered one answers HTTP 200 `97` (the signature is checked before any query) or `99`, never anything else; after the database is back the first answers `{"RspCode":"00","Message":"Confirm Success"}` and the order reads `paid = true`.
Traces: SC-17 (PAY-002 failure row "Upstream failure")
Check: interact_page

**Scenario**: TC-40 — a store that fails half-way leaves nothing half-written
**Given**: K+; RESET; O1 placed (status `placed`, unpaid, reference R1); `ALTER TABLE payment_notices RENAME TO payment_notices_x` (renamed back afterwards); SNAPSHOT(O1) without the notice count.
**When**: A correctly signed success NOTICE for R1 is sent; the table is renamed back; the same request is sent again.
**Then**: The first answers HTTP 200 `{"RspCode":"99","Message":"Unknown error"}`; at that moment `SELECT paid, status FROM orders WHERE id = 'doi-24818'` still reads `false` and `placed` (the order was not marked paid without its notice row: one transaction). After the rename back, the second answers `{"RspCode":"00","Message":"Confirm Success"}`, the order reads `true` and `confirmed`, and exactly one `is_success` notice exists.
Traces: SC-17 (PAY-002 failure row "nothing is half-written")
Check: interact_page

**Scenario**: TC-41 — opening the return address, whatever it says, never marks an order paid
**Given**: K+; RESET; HOLD(O1) (unpaid, status `placed`); SNAPSHOT(O1) taken at t0; a 1280×800 browser.
**When**: RETURN(R1, `00`) is opened (a success code and a valid-looking signature); the page is left open for 60 s; then SNAPSHOT(O1) is read; `POST /api/v1/orders/<id>/payment` with the token is also sent once.
**Then**: At 60 s `paid` is `false`, `status` is `placed` and `updated_at` equals its value at t0; `SELECT count(*) FROM payment_notices` for the order is 0; the POST changes the order only by adding one attempt row (its snapshot's other fields are equal).
Traces: SC-18 (PAY-002 AC-9, item 8; PAY-004 AC-9)
Check: interact_page

**Scenario**: TC-42 — nobody but a verified notice can mark an order paid
**Given**: K+; RESET; O1 placed (T1), unpaid.
**When**: Without any session, `PUT /api/v1/orders/doi-24818` with `{"paid":true}`, then `PUT` with `{"status":"confirmed"}`, `DELETE /api/v1/orders/doi-24818`, and `POST /api/v1/orders/doi-24818/payment` with a body `{"paid":true}` and T1 are sent; then `GET /api/v1/orders/doi-24818` with T1.
**Then**: The two PUTs and the DELETE answer 401 `{"error":"unauthenticated",…}`; the POST with a body answers 400 `bad_request`; the GET answers 200 with `paid` `false` and `status` `placed`.
Traces: SC-18 (PAY-002 AC-9, item 8), contract (PUT/DELETE /v1/orders/{id})
Check: fetch_url

**Scenario**: TC-43 — in 100 of 100 attempts a bad notice changes no order
**Given**: K+; RESET; one order O1 placed, unpaid, reference R1; a count of `SELECT count(*) FROM orders WHERE paid` and of `payment_notices`, and a SNAPSHOT(O1).
**When**: 100 notices are sent in turn: 34 with an altered signature, 33 with an unknown reference (correctly signed) and 33 with a wrong amount (correctly signed), in a mixed order.
**Then**: All 100 answer HTTP 200; the codes are `97` ×34, `01` ×33, `04` ×33; after the 100 the number of paid orders, the number of notice rows and SNAPSHOT(O1) are equal to their values before.
Traces: SC-19 (NFR Integrity)
Check: interact_page

**Scenario**: TC-44 — in 100 of 100 runs two identical notices at the same moment pay the order once
**Given**: K+; RESET; SHELF(`samba-og-w-core-black`, `38`, 200); 100 orders placed through PLACE (one pair each), unpaid.
**When**: For each order in turn, two identical success NOTICE requests are issued in one batch.
**Then**: For every one of the 100 orders exactly one answer is `00` and the other `02`; the 100 orders read `paid = true`; `SELECT count(*) FILTER (WHERE is_success) FROM payment_notices GROUP BY attempt_id HAVING count(*) FILTER (WHERE is_success) <> 1` returns no row; the total of success notices is 100.
Traces: SC-19 (NFR Integrity)
Check: interact_page

**Scenario**: TC-45 — over 100 sequential signed notices the notice address answers within 2 s at p95
**Given**: K+; RESET; 100 orders placed (as TC-44), or one order with 100 failure notices (response code `24`, which keep the order unpaid and may repeat); a clock in the page (`performance.now()`).
**When**: 100 correctly signed notices are sent one after another and the elapsed time of each is recorded.
**Then**: The 95th-percentile elapsed time is at most 2000 ms and every answer is HTTP 200 with `00` (or `02` where an order was already paid); the maximum is reported.
Traces: SC-19 (NFR Performance)
Check: interact_page

---

## C. An unpaid order is given up after 30 minutes (SC-20 … SC-26)

Every case here is K+ and uses the COMPANION rule of the observation section to know that the check has run. The companion is always one pair of `samba-og-kids-cloud-white` size `36` (a different shoe from the one under test, so the two never share a shelf). Times are arranged with DUE; the shelf is read with `SELECT pairs FROM product_sizes`. After each case RESET.

**Scenario**: TC-46 — an unpaid order 31 minutes old is cancelled and its pairs go back, with the shoe's lists agreeing
**Given**: K+; RESET; SHELF(`samba-og-w-core-black`, `38`, 2); O1 placed with `qty` 2 of size `38` (the shelf for size `38` is now 0); `GET /api/v1/products/samba-og-w-core-black` read as before (`sizes_out` holds `38`; `stock` = S0); DUE(O1, '31 minutes'); the companion C created and DUE.
**When**: C is polled until `cancelled` ("the check has run"); then O1's row, the shelf and `GET /api/v1/products/samba-og-w-core-black` are read.
**Then**: `status` of O1 is `cancelled` and `paid` is `false`; the shelf for size `38` reads 2 (+ 2); the shoe's record now has `38` in `sizes_low` (2 pairs) and not in `sizes_out`, and `stock` = S0 + 2; the sum of `pairs` over the shoe's sizes equals `stock`.
Traces: SC-20 (PAY-003 AC-1, AC-2, items 1–3)
Check: fetch_url

**Scenario**: TC-47 — an order of two shoes returns both
**Given**: K+; RESET; O1 placed with `samba-og-w-core-black` size `38` qty 1 and `samba-og-kids-cloud-white` size `36` qty 2; the two shelves read as A0 and B0 (after the order); DUE(O1, '31 minutes'); the companion C is a **second** order of `samba-og-w-wonder-quartz` size `38` qty 1 (so it does not touch these shelves), DUE.
**When**: C is polled until `cancelled`; then the shelves are read.
**Then**: O1 is `cancelled`; size `38` of `samba-og-w-core-black` is A0 + 1 and size `36` of `samba-og-kids-cloud-white` is B0 + 2; each shoe's `stock` rose by the same amounts.
Traces: SC-20 (PAY-003 AC-2, item 2)
Check: fetch_url

**Scenario**: TC-48 — an order exactly 30 minutes old is cancelled by the next run, and the check runs at least once a minute
**Given**: K+; RESET; O1 placed, DUE(O1, '30 minutes') (placed exactly 30 minutes ago); the time of the arrangement t0 noted.
**When**: O1's status is polled every 5 s; the moment it first reads `cancelled` is t1. Then a second order O2 is placed, DUE(O2, '30 minutes') at t1, and polled the same way until t2.
**Then**: `t1 − t0` is at most 75 s and `t2 − t1` is at most 75 s (a check that runs at least once a minute cancels each within one run); both orders read `cancelled`; at t1 an order O3 placed **now** (not DUE) is still `placed`.
Traces: SC-20 (PAY-003 items 3, boundary row)
Check: fetch_url

**Scenario**: TC-49 — an order 29 minutes old is left alone
**Given**: K+; RESET; O1 placed (one pair of size `38`), DUE(O1, '29 minutes') at t0; the companion C created and DUE at the same moment; SNAPSHOT(O1) and the shelf.
**When**: C is polled until `cancelled`; O1 is read at once.
**Then**: O1 is still `placed`, unpaid, and the shelf is as in the snapshot. If O1 reads `cancelled` by then, its `updated_at` is compared with `placed_at + 30 minutes`: a cancellation **before** that moment fails the case (a premature cancellation); one at or after it means C was not seen in time (the margin is 60 s) and the case is run again with `29 minutes 50 seconds`.
Traces: SC-21 (PAY-003 AC-3)
Check: fetch_url

**Scenario**: TC-50 — a paid order placed 2 hours ago is left alone
**Given**: K+; RESET; O1 placed and paid by a success NOTICE (status `confirmed`); DUE(O1, '2 hours'); SNAPSHOT(O1) and the shelf; the companion C created and DUE.
**When**: C is polled until `cancelled`; O1 is read.
**Then**: SNAPSHOT(O1) is equal (status `confirmed`, `paid` true, same `updated_at` apart from nothing), and the shelf is unchanged.
Traces: SC-21 (PAY-003 AC-4)
Check: interact_page

**Scenario**: TC-51 — an unpaid order the dealer shipped or delivered is left alone, and its pairs stay with it
**Given**: K+; RESET; O1 placed, SET `status = 'shipped'`; O2 placed, SET `status = 'delivered'`; DUE(O1, '2 hours'), DUE(O2, '2 hours'); SNAPSHOT of each and of the shelves; the companion C created and DUE.
**When**: C is polled until `cancelled`; O1 and O2 are read.
**Then**: Both snapshots are equal: statuses `shipped` and `delivered`, `paid` `false`, `updated_at` unchanged; the shelves for their sizes are unchanged (no pair returned).
Traces: SC-21 (PAY-003 AC-10, item 1, item 4)
Check: fetch_url

**Scenario**: TC-52 — an order the dealer cancelled at minute 5 does not return its pairs a second time
**Given**: K+; RESET; SHELF(`samba-og-w-core-black`, `38`, 5); O1 placed (shelf 4); the dealer's cancel simulated as in TC-37 (SET `status = 'cancelled'`, `pairs = pairs + 1`; shelf 5); DUE(O1, '31 minutes'); SNAPSHOT(O1); the companion C created and DUE.
**When**: C is polled until `cancelled`; then, three minutes later (three more runs of the check), O1 and the shelf are read again.
**Then**: Both times the shelf for size `38` reads 5 (not 6); O1's snapshot is equal (status `cancelled`, `updated_at` unchanged: the check wrote nothing).
Traces: SC-21 (PAY-003 AC-6)
Check: fetch_url

**Scenario**: TC-53 — an unpaid order the dealer moved to "confirmed" or "packed" is still given up
**Given**: K+; RESET; O1 placed, SET `status = 'confirmed'`; O2 placed, SET `status = 'packed'`; both DUE(…, '2 hours'); the shelves for their sizes read; the companion C created and DUE.
**When**: C is polled until `cancelled`; O1, O2 and the shelves are read.
**Then**: O1 and O2 are `cancelled` and `paid` is `false`; each shelf has risen by the quantity of its order.
Traces: SC-22 (PAY-003 AC-5)
Check: fetch_url

**Scenario**: TC-54 — two checks running at the same moment cancel an order once and return its pairs once
**Given**: The expiry check's source on the PR's head and its backend tests; an order due for cancelling.
**When**: The check is read for how it cancels (the single `UPDATE … WHERE status IN ('placed','confirmed','packed') AND NOT paid` whose changed row count decides whether pairs are returned) and the backend tests are run with `run_tests` (backend).
**Then**: A backend test runs two checks concurrently over one due order and asserts that the shelf rose by the order's quantity once, and the backend suite passes; the code returns pairs only when a row changed.
Traces: SC-23 (PAY-003 AC-8)
Check: manual — the timer is inside the process and no instrument runs two checks at once; the evidence is the PR's backend test run by `run_tests` and the check's source read from `get_pr`. A repeated check over time is observed in TC-52.

**Scenario**: TC-55 — a success notice arriving as the check runs: one wins, pairs return at most once
**Given**: K+; RESET; SHELF(`samba-og-w-core-black`, `38`, 10); three orders D1, D2, D3 placed (one pair each; shelf 7), each DUE(…, '31 minutes') at the same moment t0; the companion C created and DUE at t0.
**When**: A correctly signed success NOTICE is sent for D1 at t0, for D2 at t0 + 20 s and for D3 at t0 + 40 s (so the once-a-minute check runs somewhere among them); C is polled until `cancelled`; 70 s after C is seen cancelled, D1–D3 and the shelf are read.
**Then**: Each notice answered `00` or `02`, never anything else. Each order is in exactly one of two states: **paid and left alone** (`paid` true, `status` `confirmed`, its pair still taken) or **cancelled after the check won** (`paid` true, `status` `cancelled`, its pair returned). The shelf for size `38` is 7 + (the number of cancelled orders), never more; no order is `cancelled` with `paid` false after a `00` answer.
Traces: SC-23 (PAY-003 AC-8, conflict row; PAY-002 item 9)
Check: interact_page

**Scenario**: TC-56 — the 14 sample orders and their pairs are never touched by the check
**Given**: K+; `SELECT count(*) FROM orders WHERE is_sample` reads 14 (`DOI-24804` … `DOI-24817`); one sample order S1 whose status is `placed`, `confirmed` or `packed` and `NOT paid` is found, or one is arranged by SET (`status = 'placed'`, `paid = false`, `payment_method_id = 'vnpay'`, `placed_at = now() - interval '3 hours'`; the old values are read first and put back); a SNAPSHOT of all 14 (`status, paid, updated_at`) and `SELECT sum(pairs) FROM product_sizes`; one order O1 placed **through checkout** (PLACE).
**When**: The companion C is created, DUE and polled until `cancelled`.
**Then**: The 14 snapshots and the sum of pairs are equal before and after (S1 included); `SELECT is_sample FROM orders WHERE id = 'doi-24818'` for the order placed through checkout reads `false` (it is not a sample, so the check does give it up when it is due).
Traces: SC-24 (PAY-003 AC-9, item 4; SRS open question 7)
Check: fetch_url

**Scenario**: TC-57 — "Pay now" for an order the check cancelled answers 409 saying it was cancelled
**Given**: K+; RESET; O1 placed (T1); DUE(O1, '31 minutes'); the companion C created and DUE; C polled until `cancelled`; O1 read `cancelled`.
**When**: `POST /api/v1/orders/doi-24818/payment` with T1 and no body is sent; `GET /api/v1/orders/doi-24818` with T1.
**Then**: The POST answers 409 `{"error":"conflict","message":S}` where S contains `cancel` (the cancelled sentence of TC-15) and no attempt row was added; the GET answers 200 with `status` `cancelled` and `paid` `false`.
Traces: SC-25 (PAY-003 AC-7, item 5; PAY-001 item 7)
Check: fetch_url

**Scenario**: TC-58 — the order page of a cancelled order shows the cancelled view and no "Pay now"
**Given**: K+; HOLD(O1); DUE(O1, '31 minutes'); the companion C created and DUE; C polled until `cancelled`.
**When**: A 1280×800 browser opens `/orders/<id>` (no `?placed`); `[data-testid='order-status']` and `[data-testid='pay-now']` are read.
**Then**: `[data-testid='order-status']` reads exactly `This order was cancelled.`; `[data-testid='pay-now']` does not exist in the page.
Traces: SC-25 (PAY-003 item 5; ORD-007 AC-16)
Check: interact_page

**Scenario**: TC-59 — a customer who closed VNPay's page can pay later, until the 30 minutes pass
**Given**: K+; HOLD(O1) (placed through the page; the tab was closed at VNPay's address, so the browser is brought back to the shop's origin); DUE(O1, '10 minutes'); the reference of the first attempt R1.
**When**: The browser opens `/orders/<id>`; `[data-testid='pay-now']` is read and pressed; the navigation is read. Then DUE(O1, '31 minutes'), the companion C is created and DUE, C is polled until `cancelled`, and the order page is reloaded.
**Then**: Before the press `[data-testid='pay-now']` is visible and enabled; pressing it navigates to an address on `VNPAY_PAYMENT_URL` whose `vnp_TxnRef` is a new reference (not R1). After the check, the reloaded page shows `This order was cancelled.` and no `[data-testid='pay-now']`.
Traces: SC-26 (PAY-004 item 7, AC-13; PAY-001 item 1)
Check: interact_page

---

## D. The Payment result page `/payment/return` (SC-27 … SC-37)

Unless a case says otherwise: K+; 1280×800; the browser is a HOLD(order) browser (it holds the order's token); the page is opened with RETURN(ref, code); "the request log" is read for the page's calls to `GET /api/v1/orders/<id>`. The page's own copy is checked word for word, including the ellipsis `…` (U+2026) and the apostrophe in `can't`. The one h1 is `[data-testid='payment-state']`.

**Scenario**: TC-60 — an order paid by VNPay's notice shows "Payment received." whatever the address says
**Given**: K+; RESET; HOLD(O1) with number `DOI-24818`; a success NOTICE for R1 has been answered `00` (the order is paid).
**When**: The browser opens RETURN(R1, `00`); then RETURN(R1, `24`); then `/payment/return?vnp_TxnRef=R1` with no response code.
**Then**: Each time `[data-testid='payment-state']` reads exactly `Payment received.` with `data-state="paid"`; `[data-testid='payment-line']` reads exactly `Your order DOI-24818 is paid. We will pack it next.`; `[data-testid='payment-view-order']` exists, reads `View your order` and has `href` ending `/orders/doi-24818`; there is no `[data-testid='payment-try-again']`. The request log holds a `GET /api/v1/orders/doi-24818` that carries `X-Order-Token` (the page asked the shop and did not decide from the address).
Traces: SC-27 (PAY-004 AC-1, items 1–3)
Check: interact_page

**Scenario**: TC-61 — "confirming" while the notice has not arrived, then "Payment received." with no reload
**Given**: K+; RESET; HOLD(O1), unpaid; a marker `window.__keep = 1` is set after the page loads (it vanishes on a reload).
**When**: RETURN(R1, `00`) is opened; at 1 s the page is read; at 6 s a success NOTICE for R1 is sent from the page's script; the page is read at 8 s and 10 s.
**Then**: At 1 s `[data-testid='payment-state']` reads exactly `We are confirming your payment…` (`data-state="confirming"`), `[data-testid='payment-line']` reads exactly `This usually takes a few seconds. Please do not pay again.`, `[data-testid='payment-view-order']` exists and there is no `[data-testid='payment-try-again']`. The request log shows `GET /api/v1/orders/doi-24818` repeated about every 3 s (gaps between 2 s and 4 s). By 10 s the h1 reads `Payment received.` (`data-state="paid"`) and `window.__keep` is still `1` (no reload; one navigation entry).
Traces: SC-28 (PAY-004 AC-2, AC-3)
Check: interact_page

**Scenario**: TC-62 — the state line is a status region so a change is announced
**Given**: K+; HOLD(O1), unpaid; RETURN(R1, `00`) open in the confirming state.
**When**: The element `[data-testid='payment-line']` and its ancestors are inspected for `role`, `aria-live` and `aria-atomic`; the number of `h1` elements is counted.
**Then**: The line, or an element that contains it, has `role="status"` (so it is announced politely); the page holds exactly one `h1`. After the order is paid (a success NOTICE) the same region holds the new line `Your order DOI-24818 is paid. We will pack it next.` (the region is not replaced by a different element that was never in the page).
Traces: SC-28 (Accessibility NFR)
Check: interact_page

**Scenario**: TC-63 — after 30 s with no notice the page says VNPay has not been heard from, and stops asking
**Given**: K+; RESET; HOLD(O1), unpaid, status `placed`; RETURN(R1, `00`) opened at t0.
**When**: The page is read at 29 s, 31 s and 45 s; the request log's `GET /api/v1/orders/doi-24818` calls are counted for 31–45 s; at 60 s the order is read from the database.
**Then**: At 29 s the h1 still reads `We are confirming your payment…`. At 31 s and 45 s the h1 reads exactly `We have not heard from VNPay yet.` (`data-state="timed-out"`), the line reads exactly `If money left your account it will show on your order once VNPay confirms it. Please do not pay again.`, `[data-testid='payment-view-order']` is present and there is **no** `[data-testid='payment-try-again']`. No `GET /api/v1/orders/doi-24818` is sent between 31 s and 45 s. At 60 s `paid` is `false` and `status` `placed`.
Traces: SC-29 (PAY-004 AC-4, AC-9, item 2)
Check: interact_page

**Scenario**: TC-64 — a cancelled or failed payment on an unpaid order shows "Payment was not completed." with "Try again"
**Given**: K+; RESET; HOLD(O1), unpaid, status `placed`.
**When**: The page is opened with RETURN(R1, `24`) (customer cancelled); then RETURN(R1, `07`); then `/payment/return?vnp_TxnRef=R1` with no response code.
**Then**: Each time the h1 reads exactly `Payment was not completed.` (`data-state="not-completed"`); the line reads exactly `You can try again, or place a new order.`; `[data-testid='payment-view-order']` is present; `[data-testid='payment-try-again']` is present, is a `button` (or the page's filled pill) reading `Try again` and is enabled.
Traces: SC-30 (PAY-004 AC-5, item 3)
Check: interact_page

**Scenario**: TC-65 — "Try again" is disabled while it waits and sends the browser to a new address
**Given**: K+; HOLD(O1), unpaid; the page open on RETURN(R1, `24`); the attempts of O1 counted as N; DELAY(`POST /api/v1/orders/<id>/payment`, 1500).
**When**: `[data-testid='payment-try-again']` is pressed once; it is read at 0.5 s; the next navigation is read.
**Then**: At 0.5 s the button is `disabled` (or `aria-disabled="true"`); the request log holds exactly one `POST /api/v1/orders/doi-24818/payment` with `X-Order-Token`; the browser then navigates to an address on `VNPAY_PAYMENT_URL` whose `vnp_TxnRef` is `DOI-24818-<n>` and differs from R1; `SELECT count(*) FROM payment_attempts` for the order is N + 1.
Traces: SC-30 (PAY-004 AC-6)
Check: interact_page

**Scenario**: TC-66 — when the shop answers 503 or cannot be reached, "Try again" stays and the failure is said under it
**Given**: K+; HOLD(O1), unpaid; the page open on RETURN(R1, `24`).
**When**: STUB(`POST /api/v1/orders/<id>/payment`, 503 `{"error":"payment_unavailable","message":"x"}`) and `[data-testid='payment-try-again']` is pressed; then ABORT(`POST /api/v1/orders/<id>/payment`) and it is pressed again.
**Then**: After each press the page is still on `/payment/return` with the h1 `Payment was not completed.`; an element with `role="alert"` under the button reads exactly `We could not open the payment page. Please try again.`; `[data-testid='payment-try-again']` is enabled again; the browser did not navigate.
Traces: SC-30 (PAY-004 item 3; failure behaviour)
Check: interact_page

**Scenario**: TC-67 — when the shop answers 409 the page asks for the order again and draws what it now is
**Given**: K+; RESET; two HOLD orders O1 and O2, both unpaid; the pages for each open on RETURN(R, `24`) in the not-completed state.
**When**: For O1: a success NOTICE is sent (the order is now paid), then `[data-testid='payment-try-again']` is pressed. For O2: SET `status = 'cancelled'`, then the button is pressed.
**Then**: The shop answers the press with 409 (read in the request log). O1's page asks `GET /api/v1/orders/<id>` again and draws `Payment received.` (`data-state="paid"`); O2's page draws `This order was cancelled.` (`data-state="cancelled"`); neither navigates to VNPay.
Traces: SC-30 (PAY-004 item 3; PAY-001 item 7)
Check: interact_page

**Scenario**: TC-68 — an order given up by the 30-minute rule shows "This order was cancelled." and "Keep shopping"
**Given**: K+; RESET; HOLD(O1) and HOLD(O2); O1 `status = 'cancelled'`, unpaid (as the check leaves it); O2 `status = 'cancelled'`, `paid = true` (a payment that arrived late).
**When**: RETURN(R1, `24`) is opened for O1; RETURN(R2, `00`) for O2.
**Then**: For both: the h1 reads exactly `This order was cancelled.` (`data-state="cancelled"`); the line reads exactly `The time to pay ran out, so the pairs went back on the shelf. You can order again.`; `[data-testid='payment-view-order']` is present; `[data-testid='payment-keep-shopping']` is present, reads `Keep shopping` and has `href` `/shop`; there is no `[data-testid='payment-try-again']` (cancelled beats paid).
Traces: SC-31 (PAY-004 AC-7, items 2–3; PAY-002 item 9)
Check: interact_page

**Scenario**: TC-69 — a payment this browser holds no token for shows the one same "can't match" view and nothing about any order
**Given**: K+; RESET; O1 placed in browser A (reference R1); O2 placed through the page in browser B (reference R2, token T2); the browser under test (a) is **fresh**, (b) holds only O2's token (it holds none for O1), (c) holds O1's token.
**When**: In (a) RETURN(R1, `00`); in (b) RETURN(R1, `00`) (another customer's order); in (c) RETURN(`DOI-99999-1`, `00`) (a reference that matches no order); in (c) `/payment/return?vnp_ResponseCode=00` (no reference); in (c) RETURN(`abc`, `00`) (malformed).
**Then**: Every one shows the h1 exactly `We can't match this payment to an order on this browser.` (`data-state="no-match"`), the line exactly `Look the order up with its number and phone.` and `[data-testid='payment-track']` reading `Track an order` with `href` `/track`; no `[data-testid='payment-view-order']`, no `[data-testid='payment-try-again']`; the page text holds none of `DOI-24818`, `DOI-24819`, `paid`, `confirmed`, `₫`. In (b) the request log holds no request carrying O2's token for O1's id (no token of a different order is tried).
Traces: SC-32 (PAY-004 AC-8, item 4; permission row)
Check: interact_page

**Scenario**: TC-70 — until the shop has answered, none of the states is drawn
**Given**: K+; HOLD(O1), unpaid; DELAY(`GET /api/v1/orders/<id>`, 2500).
**When**: RETURN(R1, `00`) is opened; the page is read at 0.5 s and 1.5 s and again at 4 s.
**Then**: At 0.5 s and 1.5 s there is an element with `aria-busy="true"` in the main region; no element has `data-testid='payment-state'` with any `data-state` among the seven values, no element has `data-testid='payment-try-again'`, and the text contains none of `Payment received.`, `We are confirming your payment…`, `Payment was not completed.`, `This order was cancelled.`, `We can't match this payment`. At 4 s the confirming state is drawn and the `aria-busy` placeholder is gone.
Traces: SC-33 (PAY-004 item 1)
Check: interact_page

**Scenario**: TC-71 — when the shop cannot be reached or answers 500, the page says it could not check, never a result
**Given**: K+; HOLD(O1), unpaid; the browser holds O1's token; a second, fresh browser holds nothing.
**When**: STUB(`GET /api/v1/orders/<id>`, 500 `{"error":"internal","message":"x"}`) and RETURN(R1, `00`) is opened; then ABORT(`GET /api/v1/orders/<id>`) and it is opened again; then, in the fresh browser, RETURN(R1, `00`) with ABORT on every `/api/v1/` request.
**Then**: In the two holding runs the element `[data-testid='payment-state']` has `data-state="check-failed"`; an element with `role="alert"` reads exactly `We could not check your payment. Please open your order to see its state.`; `[data-testid='payment-view-order']` is present with `href` ending `/orders/doi-24818`; the page does not show `Payment received.`, `This order was cancelled.` or the `can't match` words, and there is no `[data-testid='payment-try-again']`. In the fresh browser the page shows `[data-testid='payment-track']` (the `Track an order` pill to `/track`) and no `[data-testid='payment-view-order']`, and it never shows `Payment received.` or `This order was cancelled.`.
Traces: SC-33 (PAY-004 failure row "Upstream failure")
Check: interact_page

**Scenario**: TC-72 — refreshing the page draws the same state again from the shop's answer
**Given**: K+; RESET; HOLD(O1) paid by a notice; HOLD(O2) unpaid.
**When**: RETURN(R1, `00`) is opened and reloaded; RETURN(R2, `00`) is opened, read at 20 s, reloaded at 20 s, then read at 45 s and 52 s after the first open (25 s and 32 s after the reload).
**Then**: After the reload O1's page still reads `Payment received.`. O2's page after the reload reads `We are confirming your payment…` again (its 30 s start over): at 45 s (25 s after the reload) it still reads confirming, and at 52 s (32 s after the reload) it reads `We have not heard from VNPay yet.`.
Traces: SC-33 (PAY-004 boundary row)
Check: interact_page

**Scenario**: TC-73 — at 390 px and 320 px every state fits and the page does not scroll sideways
**Given**: K+; RESET; `UPDATE counters SET value = 99999999999 WHERE name = 'order_number'` (the next number is `DOI-100000000000`, the longest the page will draw; set back by RESET); orders placed in HOLD browsers for each state: paid (O1), confirming (O2 opened on RETURN with code `00`), timed-out (O2 after 31 s), not-completed (O3), cancelled (O4), no-match (a fresh browser), check-failed (a stubbed 500); windows 390×844 and 320×640.
**When**: Each state is drawn at each width and read: `document.documentElement.scrollWidth`, `window.innerWidth`, and the bounding boxes of the h1, the line and each button or pill (`payment-view-order`, `payment-try-again`, `payment-keep-shopping`, `payment-track`).
**Then**: In every state at both widths `document.documentElement.scrollWidth` is at most `window.innerWidth`; every one of those boxes has `left` ≥ 0 and `right` ≤ `window.innerWidth`; the longest line (the timed-out one, 105 characters) wraps inside its box (its box's `height` is more than one line height).
Traces: SC-34 (PAY-004 AC-10, item 6; Responsive NFR)
Check: interact_page

**Scenario**: TC-74 — the page sits in the shop's shell, has one h1, and uses the site's own type and colours
**Given**: K+; a fresh browser (the "can't match" view needs no token); 1280×800; the shop's `/track` page for comparison.
**When**: `/payment/return` and `/track` are measured: the count of `header`, `footer` and `h1`; the h1's computed `font-family`, `font-size` and `color`; the page background and the line's `color`; then, in a holding browser, the `color` of the alert in the check-failed state (stubbed 500) and of the checkout's own alert (the K− line of TC-24).
**Then**: `/payment/return` has exactly one `header`, one `footer` and one `h1`, and the header and the footer hold the same links as on `/track`; its h1's `font-family` equals the `/track` h1's; its background colour equals `/track`'s; the state line's `color` equals the soft-ink `color` of `/track`'s secondary line; the failure alert's `color` equals the checkout alert's `color`. No value is read from a raw colour the site's theme does not have: every `color` and `background-color` read is one the other pages also compute.
Traces: SC-35 (PAY-004 AC-12, item 5; Accessibility NFR)
Check: measure_styles

**Scenario**: TC-75 — every word of the page is a content key, none is written in the component
**Given**: The PR's diff of `src/pages/PaymentReturn.tsx`, its components, and the list of `pages.payment_return.*` keys in the PR description.
**When**: The sources are read for string literals that carry the page's words (the fifteen headings and lines of SC-27 to SC-33 and the two pill/link labels).
**Then**: Each of the words is read from a content key listed under `pages.payment_return.*`; no heading, line, link or button text of the page is a literal in the component; the keys' default text equals the words of SC-27 to SC-33.
Traces: SC-35 (SRS open question 3)
Check: manual — no instrument reads a component's source or the Studio's edit of a word; the evidence is the PR diff read with `get_pr`.

**Scenario**: TC-76 — the design's flow with VNPay ends on the result page and then the order, with the order paid once
**Given**: K+; RESET; a fresh 1280×800 browser; the shoe `samba-og-w-core-black` size `38` available.
**When**: The pair is chosen on `/shop/samba-og-w-core-black` (`[data-testid^='size-']`, `[data-testid='add-to-bag']`), `/checkout` is completed with FORM and `[data-testid='place-order']` is pressed (the browser leaves for VNPay's address; its `vnp_TxnRef` is R1); the verifier sends the signed success NOTICE(R1) VNPay would send, sends the identical notice a second time, and navigates the browser to RETURN(R1, `00`); then `[data-testid='payment-view-order']` is pressed.
**Then**: The first notice answers `00`, the second `02`; the order has exactly one success notice and `paid` true. The browser at RETURN shows `Payment received.`; pressing "View your order" lands on `/orders/doi-24818`, whose `[data-testid='order-status']` reads `Your order is confirmed.` and whose "Payment" row reads `VNPay — ATM, Visa, Mastercard · paid`. On a public deployment the verifier may instead pay on VNPay's sandbox with its test card; that is not required.
Traces: SC-36 (PAY-002 AC-13; PAY-004 AC-11; ORD-009 AC-5)
Check: interact_page

**Scenario**: TC-77 — the change set adds the result page and its route and edits no page of the earlier stories
**Given**: The PR's file list and diff (`get_pr`).
**When**: The changed files are read.
**Then**: The change set adds `src/pages/PaymentReturn.tsx` (and any components under `src/components/`), `src/lib/online-payment-with-vnpay.ts`, and backend files; `src/App.tsx` changes by exactly one import and one `<Route path="/payment/return" …>` line (no line removed or changed); none of `src/pages/Checkout.tsx`, `src/pages/Order.tsx`, `src/components/PayNow.tsx`, `src/lib/orders.ts`, `src/lib/order-page-and-pay-now.ts`, `src/theme.css`, `src/index.css` is edited; no file under `src/lib/mock/` is added. In the running page, `/payment/return` resolves to the new page (the fresh-browser "can't match" view) rather than the not-found page.
Traces: SC-37 (PAY-004 AC-14; PAY-001 item 6)
Check: manual — a change set is a property of the PR's files, which no HTTP, browser or style instrument reads; the evidence is `get_pr`. The route half is observed by TC-69 and TC-74.

---

## E. Keys and going live (SC-38, SC-39)

**Scenario**: TC-78 — the hash secret is in no file of the built site
**Given**: K+ (SECRET known to the verifier); the shop's home page `/` and `/payment/return` fetched.
**When**: `GET /` is read for every `<script src>`, `<link href>` and `<link rel="modulepreload">` address; each of those files (the JavaScript bundles and the stylesheet) is fetched; every fetched body is searched for the text of SECRET, for `VNPAY_HASH_SECRET`, for `VNPAY_TMN_CODE` and for `VITE_VNPAY`.
**Then**: None of the searches finds anything in any file; every file answers 200.
Traces: SC-38 (PAY-005 AC-3, item 4; Security NFR)
Check: fetch_url

**Scenario**: TC-79 — after a payment and after a bad notice the hash secret is in no API answer
**Given**: K+; RESET; O1 placed (T1); every answer and every response header below is kept as text.
**When**: The placing answer, `POST /api/v1/orders/<id>/payment`, `GET /api/v1/orders/<id>` with T1, `GET /api/v1/payment-options`, a success NOTICE (`00`), the same again (`02`), a notice with an altered signature (`97`), a wrong amount (`04`), an unknown reference (`01`) and a notice answered `99` (the database away as in TC-39) are all read; in a K− run also the 503 answers of TC-21 and TC-22.
**Then**: The text of SECRET appears in none of the bodies or headers; the payment address holds `vnp_TmnCode` (TMN) and `vnp_SecureHash` and no other value equal to SECRET; the 503 and `97` answers hold no key name or key value.
Traces: SC-38 (PAY-005 AC-3, item 4)
Check: interact_page

**Scenario**: TC-80 — the hash secret is in no line of the server's log
**Given**: A K+ run in which a payment was made (TC-76) and a notice with a bad signature was sent (TC-33).
**When**: The server's log for the run is searched for the text of SECRET.
**Then**: The text appears nowhere in the log; a notice's log line holds the order number and the code answered and not the query's signature or secret.
Traces: SC-38 (PAY-005 AC-4, item 4; services §3.17 notes)
Check: manual — no instrument reads the server's log; the evidence is the PR's logging code read with `get_pr` and the backend test that asserts the log holds no secret, run with `run_tests`.

**Scenario**: TC-81 — neither key is held in a variable that begins with `VITE_`
**Given**: The PR's files: `code/frontend/` (`.env*`, `vite.config.*`, `src/`) and `code/backend/.env.example`.
**When**: They are searched for `VITE_VNPAY`, `VITE_HASH`, `VITE_TMN` and for any `import.meta.env` read of a VNPay value.
**Then**: There is no match; the two keys are read only by the backend from its environment.
Traces: SC-38 (PAY-005 item 4)
Check: manual — a variable's name in source is not served over HTTP; the evidence is the PR's files. The built bundle half is TC-78.

**Scenario**: TC-82 — the settings' example file names the four settings with no value
**Given**: `code/backend/.env.example` on the PR's head.
**When**: It is read.
**Then**: It contains the lines `VNPAY_TMN_CODE=`, `VNPAY_HASH_SECRET=`, `VNPAY_PAYMENT_URL=` and `PUBLIC_BASE_URL=` with nothing after the `=` (no sandbox value, no example secret); the two keys are read from the environment only (no default in code).
Traces: SC-39 (PAY-005 AC-5, item 1)
Check: manual — a repository file is not served by the running story; the evidence is the PR's file read with `get_pr`.

**Scenario**: TC-83 — every address starts at the configured payment page
**Given**: K+; the run's `VNPAY_PAYMENT_URL` read (default `https://sandbox.vnpayment.vn/paymentv2/vpcpay.html`); O1 placed.
**When**: The address of the placing answer, of "Pay now" and of the result page's "Try again" (TC-65) are parsed.
**Then**: Each one's origin and path equal `VNPAY_PAYMENT_URL`'s exactly. That the base changes with the setting and no code (the same build run with another value) is observed only when the verifier can run with a second value; otherwise that half is UNVERIFIED (instrument lacked: a way to change a setting of the running backend).
Traces: SC-39 (PAY-005 item 3)
Check: fetch_url

---

## F. Contract cases for the endpoints this story uses

**Scenario**: TC-84 — a body on "Pay now" is refused 400, after access is checked
**Given**: K+; RESET; O1 (T1) and O2 (T2) placed, unpaid; SNAPSHOT(O1) with the attempt count.
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with T1 and (a) `Content-Type: application/json` body `{"amount":1000}`; (b) body `{}`; (c) body `[]`; (d) `Content-Type: text/plain` body `x`; then, as controls, (e) T1 with no body; (f) T2 with body `{}`; (g) no token with body `{}`.
**Then**: (a)–(d) answer 400 `{"error":"bad_request","message":<sentence>}` with no 200, 404, 409, 415, 422 or 503; (e) answers 200 with an address; (f) and (g) answer 404 `not_found` (access before body). The attempt count rose by exactly one (from (e)); SNAPSHOT(O1) is otherwise equal.
Traces: contract (POST /v1/orders/{id}/payment `bad_request`), SC-1
Check: fetch_url

**Scenario**: TC-85 — with the database away "Pay now" answers 500 `internal`, generic, and recovers
**Given**: K+; RESET; O1 placed (T1); `set_db_available` takes the database away while the backend keeps running.
**When**: `POST /api/v1/orders/doi-24818/payment` is sent with T1; the database is made available again and it is sent once more.
**Then**: The first answers 500 `{"error":"internal","message":<sentence>}` with no `errors` key and no body text containing `SELECT`, `pgx`, `relation`, `connection`, `goroutine`, a token or SECRET; after the database is back the second answers 200 with an address.
Traces: contract (POST /v1/orders/{id}/payment `internal`)
Check: fetch_url

**Scenario**: TC-86 — the order read the result page polls holds `paid` and `status` and nothing private
**Given**: K+; RESET; O1 placed (T1) and paid by a success NOTICE (transaction number `14012345`).
**When**: `GET /api/v1/orders/doi-24818` is sent with T1; then without a token.
**Then**: With T1: 200, `Cache-Control: no-store`, the body has `id` `doi-24818`, `number` `DOI-24818`, `status` `confirmed`, `paid` `true`, `total` 3140000, and none of the keys `phone`, `email`, `address`, `note`, `transaction_no`, `token`; the text of the body does not contain `0912345678`, `Example Street` or `Gate code`. Without a token: 404 `not_found`.
Traces: contract (GET /v1/orders/{id}), SC-27
Check: fetch_url

**Scenario**: TC-87 — placing with a field the contract does not define is refused 400 and creates nothing
**Given**: K+; RESET; the counter and the shelf read as C0 and N.
**When**: PLACE is sent with the extra top-level member `"amount": 1000`; then with `"vnp_Amount": 100000`.
**Then**: Both answer 400 `{"error":"bad_request",…}`; no order exists, the counter is still C0 and the shelf still N.
Traces: contract (POST /v1/orders: "No other undefined field is accepted"), SC-1
Check: fetch_url

---

## Further findings on the documents

11. **"Otherwise the Track an order pill" on the could-not-check line (TC-71).** SC-33 says the line comes with a "View your order" link when this browser holds the order's token, otherwise the "Track an order" pill. A browser that holds no token for the reference is already the "can't match" case of SC-32, which makes no request to the shop. The cases pin that such a browser never shows `View your order`, the paid view or the cancelled view, and accept the `can't match` view or the could-not-check line with the `Track an order` pill.
12. **The could-not-check state has no fixed heading.** SC-33 fixes the alert line and the link or pill and the hook value `data-state="check-failed"`, not an h1 text; TC-71 asserts the data attribute and the alert text, and TC-62 and TC-74 count one `h1`.
13. **The dealer's read of the order.** The story says a notice's data is observed "in the dealer's read of the order where it exists". The dealer's order screens and sign-in credentials are not available to the cases, so TC-29, TC-36, TC-37 read the database; the dealer view's `transaction_no` is not observed (the public view must not carry it, TC-86).
14. **Keys-missing cases need a K− run.** TC-20 to TC-23 cannot be observed in a K+ run; the page half (TC-24) can, with an intercepted answer. Whoever runs the project decides which run is verified; the cases do not weaken either.

---

## Coverage — every acceptance criterion of the story and the cases that serve it

| SC | Cases | SC | Cases |
|---|---|---|---|
| SC-1 | TC-01, TC-02, TC-03, TC-84, TC-87 | SC-21 | TC-49, TC-50, TC-51, TC-52 |
| SC-2 | TC-04, TC-05, TC-06 | SC-22 | TC-53 |
| SC-3 | TC-07, TC-08 | SC-23 | TC-54, TC-55 |
| SC-4 | TC-09, TC-10 | SC-24 | TC-56 |
| SC-5 | TC-11, TC-12 | SC-25 | TC-57, TC-58 |
| SC-6 | TC-13, TC-14, TC-15, TC-16, TC-17 | SC-26 | TC-59 |
| SC-7 | TC-18, TC-19 | SC-27 | TC-60, TC-86 |
| SC-8 | TC-20, TC-21, TC-22, TC-23 | SC-28 | TC-61, TC-62 |
| SC-9 | TC-24, TC-25 | SC-29 | TC-63 |
| SC-10 | TC-26, TC-27, TC-28 | SC-30 | TC-64, TC-65, TC-66, TC-67 |
| SC-11 | TC-29, TC-30 | SC-31 | TC-68 |
| SC-12 | TC-31, TC-32 | SC-32 | TC-69 |
| SC-13 | TC-33, TC-34 | SC-33 | TC-70, TC-71, TC-72 |
| SC-14 | TC-35 | SC-34 | TC-73 |
| SC-15 | TC-36 | SC-35 | TC-74, TC-75 |
| SC-16 | TC-37 | SC-36 | TC-76 |
| SC-17 | TC-34, TC-38, TC-39, TC-40 | SC-37 | TC-77 |
| SC-18 | TC-41, TC-42 | SC-38 | TC-03, TC-78, TC-79, TC-80, TC-81 |
| SC-19 | TC-43, TC-44, TC-45 | SC-39 | TC-82, TC-83 |
| SC-20 | TC-46, TC-47, TC-48 | | |

Every SC-1 … SC-39 has at least one case (87 cases). The criteria that name a failure have a case where the failure happens: SC-7 (the attempt cannot be stored: TC-18, TC-19), SC-8 (the keys are missing: TC-20 to TC-23), SC-9 (503 on placing: TC-25), SC-13 and SC-14 (a bad signature, a wrong amount, an unknown reference: TC-33, TC-35), SC-17 (the database fails: TC-39, TC-40), SC-30 (503, unreachable and 409 on "Try again": TC-66, TC-67), SC-33 (the shop cannot be reached or answers 500: TC-71).

The SRS requirements' own criteria are all reached through the story's table (PAY-001 AC-1 … AC-10 → SC-1 … SC-10; PAY-002 AC-1 … AC-13 → SC-11 … SC-19, SC-36; PAY-003 AC-1 … AC-10 → SC-20 … SC-26; PAY-004 AC-1 … AC-14 → SC-27 … SC-37, SC-26, SC-18; PAY-005 AC-1 … AC-5 → SC-10, SC-8, SC-9, SC-38, SC-39). PAY-003 AC-11 and the bank-transfer and MoMo requirements (PAY-020 … PAY-022) are out of scope for this story, as the story's traceability says.

## What is not observed by an instrument, and why

- **`Check: manual`** — TC-54 (two expiry checks at the same moment: the timer is in the process, no instrument runs two), TC-75 (words as content keys: component source), TC-77 (the change set: the PR's files), TC-80 (the server's log), TC-81 (a variable name in source), TC-82 (`.env.example`). Each names the evidence the verifier reads instead (`get_pr`, `run_tests`).
- **K− only** — TC-20, TC-21, TC-22, TC-23 (UNVERIFIED in a K+ run, instrument lacked: a way to unset a setting of the running backend); TC-24's stubbed half runs in either.
- **Secret-dependent** — every case that signs a notice (TC-05 to TC-07 apart from the unsigned reads, TC-29 to TC-45, TC-55, TC-76, TC-79) needs SECRET and TMN; without them it is UNVERIFIED (instrument lacked: the sandbox hash secret).
- **Waits** — TC-41 (60 s), TC-63 (31 s and 60 s), TC-72 (52 s), TC-73 (31 s per width) and the expiry group (up to 90 s per wait) are real waits; a verifier whose instrument cannot wait that long reports them UNVERIFIED with that instrument named.
