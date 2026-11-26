# Test cases — More ways to pay: VietQR transfer and MoMo

Module: `payments` · Plan item 12 (P3) · Requirements: PAY-020, PAY-021, PAY-022 (`docs/payments/SRS.md` §4.2), and the changes they make to ORD-006, ORD-007 item 8, ADMIN-016 item 4 (AC-11, AC-12) and SITE-006 item 3
Story: `docs/payments/stories/more-ways-to-pay-vietqr-transfer-and-momo.md` — criteria SC-1 … SC-39 (one or more cases each; the table at the end walks the list)
Contract: `docs/architecture/services.md` §2.3 (error envelope), §2.5 (validation boundary), §2.6 (idempotency), §2.8 (order shapes), §3.5 (`GET /v1/payment-options`), §3.10 (`POST /v1/orders`), §3.11 (`GET /v1/orders/{id}`), §3.13 (`POST /v1/orders/{id}/payment`), §3.15 (`PUT /v1/orders/{id}`), §3.17 (the VNPay notice, for its shape only); `docs/architecture/erd.md` §3.7 (`payment_methods`), §3.12 (`orders`), §3.16 (`payment_attempts`), §3.17 (`payment_notices`)

Written before the implementation and before this story's technical design. The transfer-details call and the MoMo notice's body are this story's own and are **not** in the contract yet: no case invents a name for them. The cases call the first **TRANSFER(id)** and the second **MNOTICE** (both defined below) and the verifier uses the path and member names the merged design fixes.

**Risk level: high.** This item takes money (MoMo), publishes the shop's own bank account to customers, lets a person mark an order paid by hand for the first time in the shop's life, answers a public unauthenticated address that anyone on the internet can call, and hands pairs back to a shared shelf on two timers. Every notice answer, every ordering of the notice's checks, every race, every boundary of both timers, every refusal of a hand-marked payment and every secret-handling rule is written out value by value. The pages (checkout rows, the QR block, the dealer's button, the result page) are checked word for word, state by state, at phone width as well as desktop, and each failure a criterion names has a case where it happens.

## How these cases are observed

- **Settings are the profile of the run.** The story's settings are read once from the environment, so which ways to pay exist depends on how the running backend was started. A case names the **profile** it needs; the verifier learns the running one first from `GET /api/v1/payment-options` (TC-01, TC-04, TC-05, TC-06).
  - **P-ALL**: VNPay's keys (`VNPAY_TMN_CODE`, `VNPAY_HASH_SECRET`), the bank account (`VIETQR_BANK_BIN`, `VIETQR_BANK_NAME`, `VIETQR_ACCOUNT_NUMBER`, `VIETQR_ACCOUNT_NAME`, all four) and MoMo's keys (`MOMO_PARTNER_CODE`, `MOMO_ACCESS_KEY`, `MOMO_SECRET_KEY`, all three) set. This is the run the story is meant to be verified in; every case not naming another profile is a P-ALL case.
  - **P-BANK**: VNPay and the bank account set, MoMo not. **P-MOMO**: VNPay and MoMo set, no bank account. **P-VNPAY**: VNPay only. **P-NOVNPAY**: the bank account set, VNPay's keys missing (MoMo either way). **P-NONE**: nothing set. **P-PARTIAL**: one of the four bank values blank and one of the three MoMo values blank.
  - A verifier who can run only one profile reports every case that needs another **UNVERIFIED** and names the instrument lacked: *a way to set `VNPAY_*`, `MOMO_*` and `VIETQR_*` of the running backend*. It does not fail them. Where a case's page half can be answered by an intercepted `GET /api/v1/payment-options` (STUB below) the case says so, and that half runs in any profile.
  - **The values the verifier knows** (called below): **BIN, ACC, HOLDER, BANKNAME** (the four bank values), **MPC, MAK, MSK** (MoMo's partner code, access key and secret key), **TMN, SECRET** (VNPay's). Without MSK no case can sign a MoMo notice: every case that signs is then UNVERIFIED (instrument lacked: MoMo's secret key). Without a stand-in for MoMo (below) every case that reads the request the shop sent to MoMo is UNVERIFIED (instrument lacked: a stand-in for MoMo's create-payment API that records requests).
- **Windows.** Default 1280×800. Phone cases say 390×844 (and 320×640 where the criterion names it); the desktop-beside case says 1024×768.
- **Addresses.** The API is read through the gateway as `/api/v1/...` (the backend itself mounts `/v1/...`). MoMo's notice address is `/api/v1/payments/momo/ipn`. Pages: `/checkout`, `/shop/<shoe>`, `/orders/<id>`, `/payment/return`, `/track`, `/admin/orders/<id>`, `/admin/sign-in`.
- **Money and dates.** `vi-VN` writes `3.140.000 ₫` (the space before `₫` may be a no-break space: compare with either). API amounts are integers of VND. MoMo's unit is whole VND (`3140000`), not ×100 as VNPay's is.
- **Selectors the story fixes.** `[data-testid='pay-vnpay']`, `[data-testid='pay-bank']`, `[data-testid='pay-momo']` (the checkout rows; `data-selected` is `"true"` or `"false"`, `aria-disabled="true"` when the way is unavailable); `[data-testid='transfer-block']`, `-qr`, `-holder`, `-bank`, `-account`, `-amount`, `-note`, `-failed` (the QR block and its parts, each prefixed `transfer-`); `[data-testid='admin-mark-paid']`; the result page's `[data-testid='payment-state']` (the h1; `data-state` is one of `paid`, `confirming`, `timed-out`, `not-completed`, `cancelled`, `no-match`, `check-failed`), `payment-line`, `payment-view-order`, `payment-try-again`, `payment-track`, `payment-keep-shopping`. Existing hooks: `place-order`, `checkout-total`, `add-to-bag`, `[data-testid^='size-']`, `bag-count`, `order-status` (the order page's h1), `order-total`, `pay-now`, `pay-now-error`, `order-loading`, `order-missing`, and `#checkout-name`, `#checkout-phone`, `#checkout-address`, `#checkout-city`, `#checkout-note`. The alert is the element with `role="alert"`; the loading placeholder carries `aria-busy="true"`. **The order summary** is the element that contains `[data-testid='checkout-total']`; **the "Payment" row** of the order page is the `dt` reading `Payment` and its `dd`. No Tailwind class is relied on.
- **FORM** (typed in every UI case that places an order): name `Lan Nguyen`, phone `0912345678`, street address `12 Example Street, Ward 1`, city `Da Nang`, note `Gate code 4821`.
- **REQ / PLACE** — the body of every API case that places an order, sent as `POST /api/v1/orders` with `Content-Type: application/json` and a fresh header `Idempotency-Key: mw-<case>-<n>`:

```json
{ "items": [ { "product": "samba-og-w-core-black", "size": "38", "qty": 1 } ],
  "delivery": "standard", "payment": "vnpay",
  "customer": "Lan Nguyen", "phone": "0912345678",
  "address": "12 Example Street, Ward 1", "city": "Da Nang", "note": "Gate code 4821" }
```

  **PLACE** is this request; **PLACE-BANK** and **PLACE-MOMO** are the same with `"payment": "bank"` and `"payment": "momo"`. The order id and token are read from the answer. The first order after RESET is `doi-24818` / `DOI-24818`; its first attempt's reference is `DOI-24818-1`.
- **PRICE / P3140 / SHELF / RESET / SET / DUE / SNAPSHOT.** As in `docs/payments/test-cases/online-payment-with-vnpay.md`: **P3140** is `UPDATE products SET price = 3140000 WHERE id = 'samba-og-w-core-black'` (the old price read first and put back), so one pair of size `38` totals exactly 3.140.000 ₫ (delivery is free); **SHELF(shoe, size, n)** sets `product_sizes.pairs` (old value put back); **RESET** is `DELETE FROM orders WHERE is_sample = false` (items, tokens, attempts and notices go with them), `UPDATE counters SET value = 24817 WHERE name = 'order_number'`, every PRICE, SHELF, `status`, `paid`, `placed_at`, `payment_method_id`, `is_sample` edit and renamed table put back and the database made available again; **SET(order, "<assignments>")** is `UPDATE orders SET <assignments> WHERE id = '<order>'`; **DUE(order, age)** is `SET(order, "placed_at = now() - interval '<age>'")`; **SNAPSHOT(order)** is `SELECT status, paid, total, payment_method_id, placed_at, updated_at FROM orders WHERE id = '<order>'` (`updated_at` is moved by a trigger on any update, so an equal `updated_at` proves nothing was written), the count of `payment_attempts` of the order, `SELECT count(*), count(*) FILTER (WHERE is_success) FROM payment_notices n JOIN payment_attempts a ON a.id = n.attempt_id WHERE a.order_id = '<order>'`, and the `pairs` of each size the order holds. "Unchanged" means the snapshot is equal before and after. The pairs of a shoe are read with `SELECT pairs FROM product_sizes WHERE product_id = … AND size = …`; the shoe's public record `GET /api/v1/products/<shoe>` carries `sizes_low` (1–2 pairs), `sizes_out` (0 pairs) and `stock` (the sum of all sizes).
- **FX-BANK(order) and FX-MOMO(order).** An order whose way to pay is bank transfer / MoMo, `is_sample` false, placed through checkout. Made with PLACE-BANK / PLACE-MOMO where the profile lists the way; otherwise (the way is not listed in this run) made with PLACE and then `UPDATE orders SET payment_method_id = 'bank'` (or `'momo'`) and `DELETE FROM payment_attempts WHERE order_id = '<order>'`, which is what a real order of that way holds (a bank order has no attempt). If VNPay's keys are also missing, PLACE answers 503 and the order is inserted with `set_db_state` (an `orders` row with its `order_items` line and an `order_tokens` row holding `sha256` of a token the case knows). The case says when it used the arranged form.
- **Sessions.** `ADMIN`: the cookie from `POST /api/v1/auth/sign-in` with the run's `ADMIN_EMAIL` / `ADMIN_PASSWORD`; `STAFF`: the cookie of an active staff account made as `POST /api/v1/admin/staff` as ADMIN (`tc-sa@example.com`, `staff-pass-A1`); `CUSTOMER`: the cookie of a customer account (`tc-customer@example.com`); `NONE`: no cookie; `FORGED`: `session=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA`. In a browser case "signs in as ADMIN" means typing the email and password on `/admin/sign-in` and pressing "Sign in". RESET also deletes `users` whose email starts `tc-`.
- **STAND-IN.** MoMo's servers cannot be reached from a `.localhost` run and the run may have no route to MoMo's sandbox, so where the shop would call MoMo the case points `MOMO_API_URL` at a **stand-in** the verifier runs: it answers `POST <MOMO_API_URL>/v2/gateway/api/create` with HTTP 200 `{"partnerCode":MPC,"requestId":<echo>,"orderId":<echo>,"amount":<echo>,"responseTime":<ms>,"message":"Successful.","resultCode":0,"payUrl":"https://stand-in.test/pay/<orderId>"}`, records every request it receives (method, path, headers, body) and serves that record at its own address (read with `fetch_url`). It has three other modes, switched by the verifier: **slow** (answers after 4 s), **refuse** (HTTP 200, `resultCode` 22 `"Invalid amount."`, no `payUrl`) and **closed** (nothing listens on the address). Where the verifier can reach MoMo's real sandbox instead, the cases that read the request are UNVERIFIED and the others read the address the sandbox gave. A case that needs MoMo's real payment page is not required by any case below.
- **MSIGN(create request)** — the signature MoMo's create-payment API requires: lower-case hex HMAC-SHA256 keyed with MSK over `accessKey=<MAK>&amount=<amount>&extraData=<extraData>&ipnUrl=<ipnUrl>&orderId=<orderId>&orderInfo=<orderInfo>&partnerCode=<MPC>&redirectUrl=<redirectUrl>&requestId=<requestId>&requestType=<requestType>` (fields in alphabetical order, values as sent).
- **MNOTICE(ref, changes)** — the request MoMo sends to the shop, as `POST /api/v1/payments/momo/ipn` with `Content-Type: application/json`, **no cookie and no `X-Order-Token`**, JSON body: `partnerCode` MPC, `orderId` `<ref>`, `requestId` (a string no earlier notice used), `amount` `<order total, a number>`, `orderInfo` `Pay for DOI-24818`, `orderType` `momo_wallet`, `transId` (a number no earlier notice used, 10 digits), `resultCode` `0`, `message` `Successful.`, `payType` `qr`, `responseTime` (milliseconds now), `extraData` `""`, and `signature` = lower-case hex HMAC-SHA256 keyed with MSK over `accessKey=<MAK>&amount=<amount>&extraData=<extraData>&message=<message>&orderId=<orderId>&orderInfo=<orderInfo>&orderType=<orderType>&partnerCode=<partnerCode>&payType=<payType>&requestId=<requestId>&responseTime=<responseTime>&resultCode=<resultCode>&transId=<transId>`. "Changes" are applied **before** signing unless the case says "altered after signing". The answer is read as status, `Content-Type` and body. The verifier computes the signature in the page with `crypto.subtle`, so every case that signs is `Check: interact_page`. If the merged design's notice differs from this layout (TL confirms it against MoMo's current documentation), the verifier follows the design's member list and signing string; no assertion below changes.
- **RETURN-MOMO(ref, code)** — the page `/payment/return?partnerCode=MPC&orderId=<ref>&requestId=<any>&amount=<total>&orderInfo=Pay+for+<number>&orderType=momo_wallet&transId=<any>&resultCode=<code>&message=<text>&payType=qr&responseTime=<ms>&extraData=&signature=<any 64 hex characters>` as MoMo's return redirect writes it. The page must ignore the signature. **RETURN-VNPAY(ref, code)** is the VNPay story's `/payment/return?vnp_Amount=<total×100>&vnp_BankCode=NCB&vnp_ResponseCode=<code>&vnp_TxnRef=<ref>&vnp_TransactionStatus=<code>&vnp_SecureHash=<any 128 hex characters>`.
- **EMV(payload)** — the QR payload read as EMVCo TLV (two-digit tag, two-digit length, value; nested for tags 38 and 62) and checked: tag `00` is `01`; tag `53` is `704`; tag `54` is the amount in whole VND; tag `58` is `VN`; tag `38` holds the NAPAS 247 identifier `A000000727` and, inside it, the bank BIN and the account number; tag `62` holds the transfer note; the last eight characters are `6304` and four upper-case hex digits, which equal CRC-16/CCITT-FALSE (polynomial `0x1021`, initial value `0xFFFF`) of every character before those four digits. The verifier computes the CRC in the page.
- **HOLD(order).** The browser opened for a UI case holds that order's token the way a real customer's does: the order is placed **through the page in that browser** (size pill, `[data-testid='add-to-bag']`, `/checkout`, FORM, the chosen row, `[data-testid='place-order']`); for a VNPay or MoMo order the browser then leaves for the provider's address, which the verifier reads from the navigation request and from which it returns to the shop's own origin, whose `localStorage` still holds the order's entry. Once the first order has shown the shape of that entry, a verifier may write an entry of the same shape for an order it placed through the API; if the shape cannot be learned the case is placed through the page. A case that says "a fresh browser" starts with empty `localStorage`, `sessionStorage` and cookies.
- **COMPANION and "the check has run".** Both timers are in-process and run at least once a minute (the 30-minute rule and the 24-hour rule share one ticker); no instrument runs them on demand. A case that waits for them adds two **companions**: C30, an unpaid VNPay order of `samba-og-kids-cloud-white` size `36` at `placed`, DUE(C30, '31 minutes'), and C24, an FX-BANK order of the same shoe and size at `placed`, DUE(C24, '25 hours'). "The check has run" means both `status` values have become `cancelled`, polled with `SELECT status FROM orders WHERE id = …` every 5 s for up to 90 s. If either is not cancelled after 90 s every case waiting for the check fails (the check does not run at least once a minute). The case under test is read **immediately** after both are seen cancelled; a case about only one rule still waits for both, so the other cannot be mistaken for a missing run.
- **Stubbing in the browser.** "STUB(path, answer)" intercepts the page's request to the API path and answers it with the status and body the case gives; "DELAY(path, ms)" lets the real request through and holds the real answer; "ABORT(path)" fails the request at the network. The request log records, for each request, the method, URL, headers and body. A case that needs one and the instrument cannot intercept is UNVERIFIED (instrument lacked: request interception) and says so. **STUB-OPTIONS(list)** is STUB(`GET /api/v1/payment-options`, 200 `{"items":<list>,"total":<n>,"page":1,"limit":100}`); each item is `{id, name, note, icon, online:true, is_available}` with the names `VNPay — ATM, Visa, Mastercard`, `Bank transfer (VietQR)`, `MoMo wallet`.
- **Timings.** "At 1 s" means one second after the navigation started, read from the page. Waits of 6 s, 30 s and 60 s are real waits; a verifier whose instrument cannot wait that long reports the case UNVERIFIED (instrument lacked: a wait of that length), not failed.

## Ambiguities found in the documents (reported, not silently resolved)

1. **The transfer-details call has no name yet.** The story proposes `GET /v1/orders/{id}/transfer` and leaves the name, the member names and the QR drawing to TL's design. Every case about it calls **TRANSFER(id)** — sent with `X-Order-Token` and no body — and reads "the holder, the bank, the account number, the amount, the note and the QR payload" under whatever member names the design fixes. SC-13 says "exactly" those six: the cases assert that the answer's members are those six and no other.
2. **The MoMo notice's layout is MoMo's, not the shop's.** The story fixes the address, the method (`POST`) and the answers (204, 400, 5xx) and leaves "the codes against MoMo's current documentation" to TL. MNOTICE follows MoMo's documented IPN (v2); if the design differs, the verifier follows the design. A 204 has no body: the cases assert the status and an empty body.
3. **Return address of MoMo.** SC-33 names `orderId=<number>-<n>` and `resultCode=<code>`; MoMo's real return address carries more members (RETURN-MOMO). The page must ignore all but those two.
4. **Where the transfer note sits in the QR payload.** The story says "the order number as the note". The standard places it in tag 62 (additional data); the cases assert that tag 62's value contains `DOI-24818` and accept any sub-tag, because the design may use sub-tag 08 (purpose) or 01 (bill number).
5. **`?amount=1000` on "Pay now".** SC-7 says an amount "on 'Pay now' changes nothing"; the contract (§3.13) defines no query parameter and sends no body. As in the VNPay story's cases, either 200 carrying the stored amount or 400 `bad_request` with no attempt stored is accepted; a 200 carrying `1000` or any 5xx fails. A body (`{"amount":1000}`) is a different matter: 400 (TC-100).
6. **`paid` of the wrong type on a bank order.** The story accepts `{"paid": true}` for a bank order and refuses every other `paid`; §3.15 refuses the name `paid` "whatever the value and the JSON type" with 422 and an undefined key with 400. For `{"paid":"true"}`, `{"paid":1}` and `{"paid":null}` the cases accept 400 or 422 and fail only a 200 or a change.
7. **"Whatever its status".** SC-22 says an unpaid bank-transfer order shows "Mark as paid" "whatever its status", which includes `delivered` and `cancelled`; PAY-020's conflict row and SC-18 say a cancelled order can be marked paid. TC-55 asserts the button for all six statuses.
8. **Unpaid shipped or delivered within the 30 minutes.** The contract (§3.13) refuses "Pay now" only for paid, cancelled, not-online and expired orders; the cases for MoMo follow the VNPay story's reading (an unpaid shipped MoMo order inside its 30 minutes gets a fresh address). A dealer cannot realistically ship inside 30 minutes; the cases pin the contract, not the likelihood.
9. **A MoMo notice's kept columns.** SC-23 says "MoMo's transaction number, amount, result code and time are kept"; the ERD §3.17 columns are `transaction_no`, `amount`, `response_code`, `is_success`, `paid_time`. The cases read those; if the design renames or adds a column the verifier reads the one that holds the same fact. `payment_notices` holds only verified notices (ERD §3.17), so "keeps nothing" for a refused notice is an unchanged row count.
10. **The 29-minute and 23-hour cases have no margin on the clock.** An order 29 minutes old becomes 30 minutes old while the verifier waits for the check; TC-71 reads `updated_at` so a cancellation at or after the deadline is not mistaken for a premature one. The 23-hour order has an hour of margin.
11. **The settings of a run cannot be changed by the verifier.** Every case that needs a profile other than the running one is UNVERIFIED (instrument lacked). A story with a third or more of its cases unverified waits for the stakeholder, so the run should be P-ALL with a stand-in; the case list at the end counts which cases that rescues.

---

## A. What checkout and the shoe page offer (SC-1 … SC-4)

**Scenario**: TC-01 — with everything set the shop lists VNPay, bank transfer and MoMo, in that order, all available
**Given**: P-ALL.
**When**: `GET /api/v1/payment-options?anything=1` is sent, and `GET /api/v1/payment-options` without the parameter.
**Then**: Both answer 200 `{"items":[…],"total":3,"page":1,"limit":100}` with `items[].id` exactly `vnpay`, `bank`, `momo` in that order. Each item has `online` true, `is_available` true and non-empty `name`, `note` and `icon`; the names are `VNPay — ATM, Visa, Mastercard`, `Bank transfer (VietQR)` and `MoMo wallet`; the note of `bank` is `Scan the QR code on the next screen with any banking app. We send the pair as soon as the money lands.`. No item has `id` `cod`. Each item's members are only `id`, `name`, `note`, `icon`, `online`, `is_available`; the two answers are equal (the parameter is ignored); the body contains none of BIN, ACC, HOLDER or BANKNAME.
Traces: SC-1 (PAY-022 AC-1; PAY-020 AC-1; PAY-021 AC-1), contract (GET /v1/payment-options)
Check: fetch_url

**Scenario**: TC-02 — checkout draws three radio rows in a radiogroup, VNPay selected, "Place order" enabled
**Given**: P-ALL (or STUB-OPTIONS with the three ways available); a 1280×800 browser with `samba-og-w-core-black` size `38` in the bag; `/checkout` loaded; FORM typed.
**When**: Step 03 is read: the element with `role="radiogroup"`, the three rows, `[data-testid='place-order']`, and the page text.
**Then**: Exactly one `role="radiogroup"` in step 03 holds exactly three rows, in document order `[data-testid='pay-vnpay']`, `[data-testid='pay-bank']`, `[data-testid='pay-momo']`, with the visible names `VNPay — ATM, Visa, Mastercard`, `Bank transfer (VietQR)`, `MoMo wallet` (each row also shows its icon and note). `pay-vnpay` has `data-selected="true"` and the other two `data-selected="false"`; none has `aria-disabled="true"`; `[data-testid='place-order']` is not disabled; no `role="alert"` line about payment is shown; the page text holds none of `Cash on delivery`, `Pay on delivery`.
Traces: SC-1 (PAY-022 AC-1, item 1; ORD-006 AC-1)
Check: interact_page

**Scenario**: TC-03 — choosing another row selects it alone, by click and by keyboard, and the summary names it
**Given**: As TC-02.
**When**: `[data-testid='pay-bank']` is clicked and read, and the order summary is read; `[data-testid='pay-momo']` is clicked and the summary read; focus is put on the radiogroup's selected row, ArrowUp is pressed (twice) and the rows read; then ArrowDown and Space are pressed on `pay-vnpay` and the rows read.
**Then**: After the first click `pay-bank` has `data-selected="true"` and the other two `"false"`, and the summary text contains `Bank transfer (VietQR)` and not `VNPay` or `MoMo wallet`; after the second click only `pay-momo` is selected and the summary names `MoMo wallet`. By keyboard exactly one row is selected after each key and the summary names it; the focus ring is visible on the focused row (its computed `outline-style` is not `none` or its `box-shadow` is not `none`). Clicking anywhere on a row (its name, its note, its dot) selects it.
Traces: SC-1 (PAY-022 AC-1; story §6 Checkout)
Check: interact_page

**Scenario**: TC-04 — with the bank account set and MoMo not, the options are VNPay then bank transfer
**Given**: P-BANK.
**When**: `GET /api/v1/payment-options` is sent.
**Then**: 200 with `items[].id` exactly `vnpay`, `bank` (in that order), `total` 2, both `is_available` true; no `momo` item.
Traces: SC-2 (PAY-022 AC-2; PAY-020 AC-1, AC-2)
Check: fetch_url

**Scenario**: TC-05 — with MoMo's keys set and no bank account, the options are VNPay then MoMo
**Given**: P-MOMO.
**When**: `GET /api/v1/payment-options` is sent.
**Then**: 200 with `items[].id` exactly `vnpay`, `momo` (in that order), `total` 2, both `is_available` true; no `bank` item.
Traces: SC-2 (PAY-021 AC-1, AC-2; PAY-022 item 1)
Check: fetch_url

**Scenario**: TC-06 — with neither set, VNPay alone, exactly as the VNPay story left it
**Given**: P-VNPAY.
**When**: `GET /api/v1/payment-options` is sent.
**Then**: 200 with exactly one item: `id` `vnpay`, `online` true, `is_available` true, `name` `VNPay — ATM, Visa, Mastercard`, non-empty `note` and `icon`; `total` 1; no `bank`, `momo` or `cod` item.
Traces: SC-2 (ORD-006 AC-3, item 3; PAY-022 item 1)
Check: fetch_url

**Scenario**: TC-07 — checkout draws one row for each way the shop's answer lists, and no row for a way it does not list
**Given**: A 1280×800 browser with a pair in the bag; four runs of `/checkout`, each with STUB-OPTIONS: (a) `vnpay`, `bank` (both available); (b) `vnpay`, `momo`; (c) `vnpay` alone; (d) `vnpay`, `bank`, `momo` and a fourth invented way `{"id":"zalopay","name":"ZaloPay","note":"Test note.","icon":"wallet","online":true,"is_available":true}`.
**When**: Step 03 is read in each run.
**Then**: (a) two rows `pay-vnpay`, `pay-bank`, no `pay-momo`, `pay-vnpay` selected; (b) `pay-vnpay`, `pay-momo`, no `pay-bank`; (c) one row `pay-vnpay`, selected, and no other row exists; (d) four rows, the fourth reading `ZaloPay` with the note `Test note.` and choosable (selecting it makes its `data-selected` `"true"` and the summary name `ZaloPay`) — the page holds no list of ways to pay in code, it draws what the shop says. In every run the page text holds none of `Cash on delivery`.
Traces: SC-2 (PAY-022 AC-2; PAY-022 item 2), SC-39 (PAY-022 AC-3 as amended)
Check: interact_page

**Scenario**: TC-08 — a bank account with a value blank, or MoMo with a key blank, counts as not set
**Given**: P-PARTIAL (the run says which bank value and which MoMo key are blank). If the run is not P-PARTIAL this case is UNVERIFIED (instrument lacked: a way to start the backend with one `VIETQR_*` value and one `MOMO_*` key blank); the backend's own tests read with `run_tests` are supporting evidence only.
**When**: `GET /api/v1/payment-options` is sent; then `POST /api/v1/orders` is sent with PLACE-BANK and with PLACE-MOMO; the counter and the shelf for `samba-og-w-core-black` size `38` are read before and after.
**Then**: The options list `vnpay` and nothing else. Both placing requests answer 422 `{"error":"invalid",…,"errors":{"payment":<non-empty sentence>}}`; no order is created (`SELECT count(*) FROM orders WHERE NOT is_sample` is 0), the counter and the shelf are unchanged.
Traces: SC-2 (PAY-020 AC-2; PAY-021 AC-2; story §4 settings)
Check: fetch_url

**Scenario**: TC-09 — cash on delivery and unknown ways are refused 422 whatever the settings, and create nothing
**Given**: Any profile; RESET; the counter C0 and the shelf N for `samba-og-w-core-black` size `38` read.
**When**: PLACE is sent four times with `"payment"` set to `cod`, `barter`, `VNPay` (wrong case) and `""`.
**Then**: Each answers 422 `{"error":"invalid",…}` with `errors.payment` a non-empty sentence and no 201; no order exists, the counter still reads C0, the shelf is still N.
Traces: SC-2 (ORD-006 AC-5; owner decision "no cash on delivery")
Check: fetch_url

**Scenario**: TC-10 — a way that is not listed in this run is refused 422: `momo` without its keys, `bank` without the account
**Given**: A profile in which the way is not listed (`bank` in P-MOMO, P-VNPAY, P-NONE; `momo` in P-BANK, P-VNPAY, P-NONE — the run says which half it can run); RESET; counter and shelf read.
**When**: PLACE-BANK (or PLACE-MOMO) is sent.
**Then**: 422 `invalid` with `errors.payment`; no order exists; the counter and the shelf are unchanged. The check is a 422 even when VNPay's keys are also missing (field checks come before the 503 gate).
Traces: SC-2 (PAY-020 AC-2; PAY-021 AC-2; ORD-006 AC-3)
Check: fetch_url

**Scenario**: TC-11 — VNPay's keys missing and the bank account set: VNPay is listed unavailable, a bank order can be placed, a VNPay one answers 503
**Given**: P-NOVNPAY; RESET; counter C0 and shelf N for `samba-og-w-core-black` size `38` read.
**When**: `GET /api/v1/payment-options`; then PLACE-BANK; then PLACE.
**Then**: The options list `vnpay` with `is_available` false and `bank` with `is_available` true (MoMo, if set, true). PLACE-BANK answers 201 with `payment_url` exactly `""`, `order.status` `placed`, `order.paid` false and a `token`; the shelf for size `38` is N − 1 and the counter C0 + 1. PLACE answers 503 `{"error":"payment_unavailable","message":<sentence>}` with no `errors` key; after it the counter is still C0 + 1 and the shelf N − 1 (nothing created, no pair taken).
Traces: SC-3 (PAY-001 item 6; story decision 3), SC-5 (works while VNPay's keys are missing)
Check: fetch_url

**Scenario**: TC-12 — checkout with VNPay unavailable and bank listed: the alert under VNPay, VNPay cannot be chosen, bank selected, "Place order" enabled
**Given**: P-NOVNPAY, or any run with STUB-OPTIONS of `vnpay` with `is_available` false and `bank` with `is_available` true; a pair in the bag; `/checkout` loaded; FORM typed.
**When**: The rows, the text under `pay-vnpay`, `[data-testid='place-order']` are read; `[data-testid='pay-vnpay']` is clicked and Space is pressed on it; the rows are read again.
**Then**: Under the VNPay row an element with `role="alert"` reads exactly `Online payment is not available right now. Please try again later.`; `pay-vnpay` has `aria-disabled="true"` and `data-selected="false"` before and after the click and the key; `pay-bank` has `data-selected="true"`; `[data-testid='place-order']` is not disabled.
Traces: SC-3 (PAY-001 item 6; story decision 3; PAY-022 item 1)
Check: interact_page

**Scenario**: TC-13 — with VNPay alone listed and unavailable the page is the VNPay story's: the alert and a disabled "Place order"
**Given**: P-NONE, or any run with STUB-OPTIONS of `vnpay` alone with `is_available` false; a pair in the bag; `/checkout` loaded; FORM typed.
**When**: The rows, the alert and `[data-testid='place-order']` are read; `[data-testid='place-order']` is clicked (by coordinates, it is disabled).
**Then**: One row `pay-vnpay` with `aria-disabled="true"`; under it `role="alert"` reads exactly `Online payment is not available right now. Please try again later.`; `[data-testid='place-order']` has the `disabled` property true; the click sends no `POST /api/v1/orders` (none in the request log).
Traces: SC-3 (PAY-001 AC-7; story decision 3 last sentence)
Check: interact_page

**Scenario**: TC-14 — the shoe page's "Payment options" lists exactly what the shop's answer lists, in its order, with a name and a full stop and the note, and no way is named in the page's code
**Given**: A 1280×800 browser on `/shop/samba-og-w-core-black`; four runs with STUB-OPTIONS: (a) `vnpay`, `bank`, `momo`; (b) `vnpay` alone; (c) `bank` alone; (d) `vnpay` and the invented `zalopay` of TC-07 (name `ZaloPay`, note `Test note.`).
**When**: The section headed `Payment options` is read in each run: its items, their order, their icons (an `svg` or `img` in each item) and their text.
**Then**: (a) three items in the order VNPay, Bank transfer (VietQR), MoMo wallet; (b) one item; (c) one item; (d) two items, the second reading `ZaloPay.` followed by `Test note.`. Every item's text is the way's `name` immediately followed by `.` and then its `note`, and each has an icon; the page text holds no `Cash on delivery` in any run, and no way the answer did not list.
Traces: SC-4 (SITE-006 AC-2 as changed; ORD-006 item 2)
Check: interact_page

**Scenario**: TC-15 — the shoe page, read as the running shop serves it, lists the same ways as the options call
**Given**: The running profile; `GET /api/v1/payment-options` read as the list L.
**When**: `/shop/samba-og-w-core-black` is rendered.
**Then**: The `Payment options` section holds exactly one item for each member of L in L's order (P-ALL: VNPay, Bank transfer (VietQR), MoMo wallet; P-VNPAY: VNPay alone, as before this story), each reading the name, a full stop and the note of L's member; none for cash on delivery.
Traces: SC-4 (SITE-006 AC-2 as changed)
Check: render_url

---

## B. Placing and paying by each way (SC-5 … SC-10)

**Scenario**: TC-16 — placing with bank transfer answers 201 with an empty payment address, an unpaid order, the pairs set aside and a token
**Given**: P-BANK, P-ALL or P-NOVNPAY; RESET; P3140; SHELF(`samba-og-w-core-black`, `38`, 5) read as N.
**When**: PLACE-BANK is sent; then `GET /api/v1/orders/doi-24818` with the answer's token; then PLACE-BANK is sent again with the **same** `Idempotency-Key`.
**Then**: The first answers 201 with `Location: /v1/orders/doi-24818`, `Cache-Control: no-store`, `order.status` `placed`, `order.paid` false, `order.payment` `bank`, `order.total` 3140000, a `token` of at least 22 URL-safe characters and a `payment_url` member that is present and exactly `""`; the shelf for size `38` is N − 1; `SELECT count(*) FROM payment_attempts WHERE order_id = 'doi-24818'` is 0 (no VNPay or MoMo attempt is stored). The GET answers 200 with the same order. The replay answers 200 with `Idempotent-Replayed: true`, the same `order.id`, a new `token`, `payment_url` `""`, and the shelf is still N − 1.
Traces: SC-5 (PAY-020 item 2; ORD-006 items 4–5, AC-9), contract (POST /v1/orders)
Check: fetch_url

**Scenario**: TC-17 — checkout with "Bank transfer (VietQR)" ends on the order page with the QR block and an empty bag
**Given**: P-BANK or P-ALL; RESET; a fresh 1280×800 browser; the pair chosen on `/shop/samba-og-w-core-black` (`[data-testid^='size-']` for `38`, `[data-testid='add-to-bag']`), `/checkout` open, FORM typed.
**When**: `[data-testid='pay-bank']` is clicked and `[data-testid='place-order']` is pressed; the browser's navigations, the final address, `[data-testid='transfer-block']`, `[data-testid='bag-count']` and the browser's storage are read.
**Then**: The request log holds `POST /api/v1/orders` answered 201; the browser's only navigation is to `/orders/doi-24818?placed=1` on the shop's own origin (it never leaves for another host); `[data-testid='transfer-block']` is visible there; `[data-testid='order-status']` contains `Thank you`; `bag-count` reads 0 or is absent and `agentteam:cart` is absent from `localStorage`; one other `localStorage` entry holds the order's id `doi-24818`, number `DOI-24818`, a placed time and a token.
Traces: SC-5 (PAY-020 item 2; ORD-006 items 4–5, AC-9; ORD-009 AC-3)
Check: interact_page

**Scenario**: TC-18 — placing with MoMo answers 201 with the address MoMo gave and stores one MoMo attempt
**Given**: P-MOMO or P-ALL; the stand-in in its normal mode and `MOMO_API_URL` pointing at it (or MoMo's reachable sandbox); RESET; P3140; the shelf for `38` read as N.
**When**: PLACE-MOMO is sent; the attempts are read with `SELECT provider, reference, amount FROM payment_attempts WHERE order_id = 'doi-24818'`; the stand-in's record is read.
**Then**: 201 with `Location: /v1/orders/doi-24818`, `order.status` `placed`, `order.paid` false, `order.payment` `momo`, a `token`, and a non-empty `payment_url` equal to the `payUrl` the stand-in answered (`https://stand-in.test/pay/DOI-24818-1`; with MoMo's real sandbox, an address beginning `https://test-payment.momo.vn/`); exactly one attempt row, `provider` `momo`, `reference` `DOI-24818-1`, `amount` 3140000; the shelf for `38` is N − 1. The stand-in's record holds exactly one create request, whose `amount` is 3140000 (not 314000000).
Traces: SC-6 (PAY-021 item 2; ORD-006 AC-7)
Check: fetch_url

**Scenario**: TC-19 — checkout with "MoMo wallet" goes to MoMo's address, whose amount is the Total shown, and empties the bag
**Given**: P-MOMO or P-ALL; the stand-in normal; RESET; P3140; a fresh 1280×800 browser with the pair in the bag; `/checkout` open; FORM typed.
**When**: The text of `[data-testid='checkout-total']` is read; `[data-testid='pay-momo']` is clicked; `[data-testid='place-order']` is pressed; the browser's next navigation and the stand-in's record are read; then the shop's own origin is opened.
**Then**: The total reads `3.140.000 ₫`; the request log holds `POST /api/v1/orders` answered 201; the browser then navigates to the `payUrl` the stand-in gave (`https://stand-in.test/pay/DOI-24818-1`); the amount in the stand-in's recorded request is 3140000, the Total in whole VND and the stored `orders.total`; when the shop's origin is opened again `bag-count` is 0 or absent, `agentteam:cart` is absent and one `localStorage` entry holds the order's id, number, time and token.
Traces: SC-6 (PAY-021 item 2; ORD-006 AC-7)
Check: interact_page

**Scenario**: TC-20 — the request the shop sends to MoMo carries the partner code, the stored amount, a reference holding the order number, the return and notice addresses and no secret
**Given**: P-MOMO or P-ALL; the stand-in normal; RESET; P3140.
**When**: PLACE-MOMO is sent with the extra members `"total": 1000`, `"delivery_fee": 5` and, inside the item, `"unit_price": 1` and `"price": 1` (the contract tolerates and discards these); the stand-in's recorded request R1 is read.
**Then**: The placing answers 201 with `order.total` 3140000. R1's body is JSON with `partnerCode` MPC; `amount` 3140000; `orderId` `DOI-24818-1` (at most 100 characters); `redirectUrl` equal to `<PUBLIC_BASE_URL>/payment/return` when `PUBLIC_BASE_URL` is set and otherwise `<origin the request arrived on>/payment/return`, with no query; `ipnUrl` equal to the same origin plus `/api/v1/payments/momo/ipn`; non-empty `requestId` and `requestType`; `signature` 64 hexadecimal characters. MSK appears nowhere in R1's URL, headers or body. The request went to the base `MOMO_API_URL` (it reached the stand-in, so the setting changed the base with no change to code).
Traces: SC-7 (PAY-021 AC-3, items 2–3; ORD-003 AC-6), SC-36 (`MOMO_API_URL`)
Check: fetch_url

**Scenario**: TC-21 — every attempt has its own reference, and ten asked at the same moment never share one
**Given**: P-MOMO or P-ALL; the stand-in normal; RESET; PLACE-MOMO answered (reference `DOI-24818-1`, token T1).
**When**: `POST /api/v1/orders/doi-24818/payment` with T1 and no body is sent twice in turn; then ten such requests are issued in one batch (`Promise.all` in one script); `SELECT reference FROM payment_attempts WHERE order_id = 'doi-24818' ORDER BY created_at` is read, and the stand-in's record.
**Then**: The two in turn answer 200 `{"payment_url":…}` (the only member) with references `DOI-24818-2` and `DOI-24818-3` in the stand-in's record; all ten of the batch answer 200 and none answers 409, 500 or 503; the table holds 13 rows with 13 distinct references, each at most 100 characters and each beginning `DOI-24818-`, `provider` `momo`, `amount` 3140000; the stand-in saw 13 create requests with 13 distinct `orderId` values.
Traces: SC-7 (PAY-021 AC-3, item 2; PAY-001 AC-3 for MoMo)
Check: interact_page

**Scenario**: TC-22 — the signature of the request verifies with the shop's secret key and with no other
**Given**: As TC-21: the stand-in's recorded create requests.
**When**: In a page, MSIGN is computed over R1's members with MSK, then with `MSK + "x"`, then with `amount` changed to `1000` and MSK.
**Then**: With MSK the result equals R1's `signature` exactly (lower-case hex, 64 characters); with `MSK + "x"` it differs; with the changed amount it differs. The same holds for a request recorded after a "Pay now".
Traces: SC-7 (PAY-021 AC-3: "a signature that verifies with the shop's secret key")
Check: interact_page

**Scenario**: TC-23 — an amount named on "Pay now" changes nothing
**Given**: P-MOMO or P-ALL; the stand-in normal; RESET; P3140; PLACE-MOMO answered (token T1); the stand-in's record cleared.
**When**: `POST /api/v1/orders/doi-24818/payment?amount=1000` with T1 and no body, then the same URL without the query.
**Then**: The first answers either 200 whose recorded create request has `amount` 3140000, or 400 `{"error":"bad_request",…}` with no attempt row added for it; it never answers 200 with `amount` 1000 and never 5xx. The second answers 200 with `amount` 3140000. `SELECT total FROM orders WHERE id = 'doi-24818'` still reads 3140000.
Traces: SC-7 (PAY-021 item 2; "an amount … on 'Pay now' changes nothing")
Check: fetch_url

**Scenario**: TC-24 — when MoMo gives no address at placing, the order still exists, unpaid, with its pairs taken and an empty address
**Given**: P-MOMO or P-ALL; RESET; P3140; the shelf for `38` read as N; three runs, one per stand-in mode: **closed** (nothing listens), **slow** (answers after 4 s) and **refuse** (`resultCode` 22). If the stand-in cannot be switched, the runs it can make are run and the others are UNVERIFIED (instrument lacked: a stand-in with those modes).
**When**: PLACE-MOMO is sent in each run (a fresh order each); the time to the answer is read; then `GET /api/v1/orders/<id>` with the token.
**Then**: Each placing answers 201 within 5 s (the provider's limit is 3 s) with `payment_url` exactly `""`, `order.status` `placed`, `order.paid` false and a `token`; the GET answers 200; the shelf has fallen by one pair per order; no body contains `resultCode`, `Invalid amount`, `partnerCode`, MSK or MAK; no answer is 5xx.
Traces: SC-8 (PAY-021 item 2; ORD-006 AC-8, item 5)
Check: fetch_url

**Scenario**: TC-25 — the order page after a failed MoMo address shows "Pay now", and when "Pay now" fails too the page says so and enables it again
**Given**: As TC-24 in the **closed** mode (or a run with no route to MoMo); a 1280×800 browser; a pair in the bag; `/checkout` open; FORM typed; `[data-testid='pay-momo']` chosen.
**When**: `[data-testid='place-order']` is pressed; the address and `[data-testid='pay-now']` are read; `[data-testid='pay-now']` is pressed; then STUB(`POST /api/v1/orders/<id>/payment`, 503 `{"error":"payment_unavailable","message":"x"}`) and it is pressed again; then ABORT of the same and pressed again.
**Then**: The browser lands on `/orders/doi-24818?placed=1` (not on another host) where `[data-testid='pay-now']` is visible and enabled. After each press the page is still on the order page, an element with `role="alert"` and `data-testid='pay-now-error'` under the button reads exactly `We could not open the payment page. Please try again.`, `[data-testid='pay-now']` is enabled again, and the browser did not navigate.
Traces: SC-8 (ORD-006 AC-8, item 5; ORD-007 failure row)
Check: interact_page

**Scenario**: TC-26 — "Pay now" for an unpaid MoMo order inside its 30 minutes answers a fresh address, in every open status
**Given**: P-MOMO or P-ALL; the stand-in normal; RESET; PLACE-MOMO answered (T1); the statuses `placed`, `confirmed`, `packed` set in turn with SET (unpaid, placed seconds ago); SNAPSHOT.
**When**: `POST /api/v1/orders/doi-24818/payment` with T1 and no body, once per status.
**Then**: Each answers 200 `{"payment_url":…}` whose address is the stand-in's `payUrl` for a reference never used before for this order (`DOI-24818-2`, `-3`, `-4`); the attempt count rises by one per call; the order's `status`, `paid` and `updated_at` are unchanged.
Traces: SC-9 (PAY-021 item 2; PAY-001 item 7; ORD-007 AC-14)
Check: fetch_url

**Scenario**: TC-27 — on the order page "Pay now" for a MoMo order is disabled while it waits and then goes to the fresh address
**Given**: P-MOMO or P-ALL; the stand-in normal; HOLD(O1) with MoMo (unpaid, `placed`), DUE(O1, '10 minutes'); a 1280×800 browser on `/orders/doi-24818`; DELAY(`POST /api/v1/orders/doi-24818/payment`, 1500).
**When**: `[data-testid='pay-now']` is read, pressed once and read again at 0.5 s; the next navigation is read.
**Then**: Before the press the button is visible and enabled; at 0.5 s it is `disabled`; the request log holds exactly one `POST /api/v1/orders/doi-24818/payment` with the token in `X-Order-Token`; then the browser navigates to the stand-in's `payUrl` for `DOI-24818-2`.
Traces: SC-9 (PAY-021 item 2; ORD-007 AC-14)
Check: interact_page

**Scenario**: TC-28 — a paid, a cancelled and an expired MoMo order answer three different 409 sentences; a stranger gets the one 404
**Given**: Any profile with VNPay's keys (the 409 and 404 answers are decided before MoMo's keys are read, so FX-MOMO arranged orders serve); RESET; three MoMo orders: O1 paid (`paid = true`, status `placed`), O2 `status = 'cancelled'`, O3 `status = 'shipped'` with DUE(O3, '31 minutes') (a shipped order is untouched by the timers, so the refusal is read without a race); each with its token; a fourth order O4 (T4) unpaid; SNAPSHOT of each.
**When**: `POST /api/v1/orders/<id>/payment` with each order's own token and no body; then for O4 with (a) no token, (b) O1's token, (c) an empty token, and for `doi-99999` with T4; each is also sent as `GET /api/v1/orders/<id>` with the same headers.
**Then**: O1, O2, O3 each answer 409 `{"error":"conflict","message":<sentence>}` with no `errors` key; the three sentences are pairwise different; O1's contains `paid`, O2's contains `cancel`; no attempt row is added and each SNAPSHOT is equal. Every request of (a), (b), (c) and the unknown id answers 404 `{"error":"not_found","message":S}` with the one same sentence S and the same body as the matching GET; no body holds an id, a token or `DOI-`.
Traces: SC-9 (PAY-001 item 7; ORD-007 AC-17, AC-18, AC-19, AC-20)
Check: fetch_url

**Scenario**: TC-29 — with MoMo's keys unset, "Pay now" for a MoMo order checks access, then state, then the keys
**Given**: P-VNPAY, P-BANK or P-NONE (MoMo not set); FX-MOMO arranged orders: O1 unpaid `placed` (T1), O2 paid (T2); SNAPSHOT of each.
**When**: `POST /api/v1/orders/<id>/payment` is sent for O1 with (a) T1, (b) no token, (c) T2; and for O2 with (d) T2.
**Then**: (a) answers 503 `{"error":"payment_unavailable","message":<sentence>}` with no `errors` key; (b) and (c) answer 404 `not_found`; (d) answers 409 `conflict`. Each SNAPSHOT is equal and no attempt row was added.
Traces: SC-9 (PAY-021 item 1; PAY-001 item 6), SC-28 (keys unset)
Check: fetch_url

**Scenario**: TC-30 — "Pay now" for a bank-transfer order answers 409 saying it is paid by bank transfer and stores nothing
**Given**: A run where `bank` is listed (P-BANK or P-ALL) or the arranged form of FX-BANK; RESET; FX-BANK orders O1 (unpaid, `placed`; T1), O2 (the same, status `confirmed`), O3 (paid), O4 (`status = 'cancelled'`); SNAPSHOT of each.
**When**: `POST /api/v1/orders/<id>/payment` with each order's own token and no body; then for O1 with no token.
**Then**: O1 and O2 answer 409 `{"error":"conflict","message":S1}` with no `errors` key, where S1 contains `bank transfer` (any case); O3's sentence contains `paid`, O4's contains `cancel`, and the three sentences are pairwise different; no attempt row exists for any of the four; every SNAPSHOT is equal. O1 with no token answers 404 `not_found` with the same body as `GET /api/v1/orders/<id>` with no token.
Traces: SC-10 (PAY-020 item 4, AC-6; ORD-007 item 8)
Check: fetch_url

**Scenario**: TC-31 — the order page draws the QR block and no "Pay now" for an unpaid bank order, and "Pay now" and no block for an unpaid VNPay or MoMo order
**Given**: P-ALL (or the arranged form of FX-BANK / FX-MOMO with the transfer call and MoMo's address answered by STUB); HOLD orders, each unpaid, `placed`, DUE(…, '5 minutes'): OB (bank), OV (VNPay), OM (MoMo); a 1280×800 browser.
**When**: `/orders/<id>` is opened for each and `[data-testid='pay-now']` and `[data-testid='transfer-block']` are counted.
**Then**: OB: `pay-now` count 0 and `transfer-block` count 1. OV: `pay-now` count 1 (visible, enabled) and `transfer-block` count 0. OM: `pay-now` count 1 and `transfer-block` count 0.
Traces: SC-10 (PAY-020 item 4; ORD-007 item 8; story decision 2)
Check: interact_page

---

## C. The bank-transfer QR code (SC-11 … SC-16)

**Scenario**: TC-32 — an unpaid bank order's page shows the QR code and, as text, the holder, bank, account number, amount and note, straight after placing, later, and after a refresh
**Given**: P-BANK or P-ALL; RESET; P3140; HOLD(OB) placed through the page with `pay-bank` (order `DOI-24818`, total 3.140.000 ₫).
**When**: The browser opens `/orders/doi-24818?placed=1`; then `/orders/doi-24818` (no `placed`); then that page is reloaded. Each time `[data-testid='transfer-block']` and its parts are read.
**Then**: Each time the block is visible with the heading `Pay by bank transfer`; `[data-testid='transfer-qr']` is visible; the labelled values read: `Account holder` with HOLDER in `[data-testid='transfer-holder']`, `Bank` with BANKNAME in `transfer-bank`, `Account number` with ACC in `transfer-account`, `Amount` with `3.140.000 ₫` in `transfer-amount`, `Transfer note` with `DOI-24818` in `transfer-note`; and the line `Scan this QR code with any banking app. We send the pair as soon as the money lands.` is inside the block. The page text holds none of `0912345678`, `12 Example Street`, `Gate code 4821` or the order's token.
Traces: SC-11 (PAY-020 AC-3, item 2)
Check: interact_page

**Scenario**: TC-33 — the QR payload names the shop's bank and account, the amount `3140000` and the order number, and its checksum is valid
**Given**: P-BANK or P-ALL; RESET; P3140; FX-BANK order `doi-24818` (token T1).
**When**: TRANSFER(`doi-24818`) is sent with T1; the payload member is read and EMV(payload) is run in a page.
**Then**: Tag `00` is `01`; tag `53` is `704`; tag `58` is `VN`; tag `54` is exactly `3140000`; tag `38` holds `A000000727`, the BIN equal to BIN and the account number equal to ACC; tag `62`'s value contains `DOI-24818`; the last eight characters are `6304` followed by four upper-case hex digits that equal the CRC-16/CCITT-FALSE of everything before them. The payload holds no phone, address or customer name.
Traces: SC-12 (PAY-020 AC-4)
Check: interact_page

**Scenario**: TC-34 — the amount in the QR is the stored total; one sent by the browser is ignored
**Given**: P-BANK or P-ALL; RESET; P3140.
**When**: PLACE-BANK is sent with `"total": 1000` and an item `"unit_price": 1` (tolerated and discarded); then TRANSFER(`doi-24818`) with the token; then TRANSFER with `?amount=1` appended.
**Then**: Placing answers 201 with `order.total` 3140000; the first TRANSFER's amount member is 3140000 and its payload's tag `54` is `3140000`; the second answers either 200 with the same amount and payload, or 400 `{"error":"bad_request",…}`; it never answers 200 with `1` as the amount and never 5xx.
Traces: SC-12 (PAY-020 AC-4; "one sent by the browser is ignored")
Check: fetch_url

**Scenario**: TC-35 — no request for the QR or its picture leaves the shop's own origin, and no secret rides in an address
**Given**: P-BANK or P-ALL; HOLD(OB) with a bank order; a 1280×800 browser with the request log on.
**When**: `/orders/doi-24818?placed=1` is opened and read until `[data-testid='transfer-qr']` has been visible for 3 s; the request log is read.
**Then**: Every request's origin equals the shop's own origin (or the scheme is `data:` or `blob:` for a picture the page drew); no request goes to another host (no QR, chart or image service); the TRANSFER call is made once or twice with the token in the `X-Order-Token` header; no request URL (path or query) holds the token, ACC or `000201`.
Traces: SC-12 (PAY-020 item 2; story decision 6)
Check: interact_page

**Scenario**: TC-36 — the transfer details answer 200, `no-store`, exactly six members, to the order's token or to the linked customer's session
**Given**: P-BANK or P-ALL; RESET; P3140; OB placed with PLACE-BANK (token T1; phone `0912345678`, note `Gate code 4821`); a customer CUSTOMER signed in who places a second bank order OL with the cookie (the order is linked to the account).
**When**: TRANSFER(OB) with T1; TRANSFER(OL) with the CUSTOMER cookie and no token.
**Then**: Both answer 200 with `Content-Type: application/json`, `Cache-Control: no-store`, and a body whose members are exactly: the holder (HOLDER), the bank (BANKNAME), the account number (ACC), the amount (3140000), the note (`DOI-24818` / its own number) and the QR payload — no other member. The body holds none of `0912345678`, `Example Street`, `Gate code`, `Lan Nguyen`, `@`.
Traces: SC-13 (PAY-020 AC-9; ORD-008), contract (§3.11 access rule)
Check: fetch_url

**Scenario**: TC-37 — with no token, another order's token, a dealer session without the token, or an unknown id the answer is the one same 404
**Given**: P-BANK or P-ALL; RESET; two bank orders O1 (T1) and O2 (T2); ADMIN and STAFF cookies; CUSTOMER's cookie (the customer is linked to neither).
**When**: TRANSFER(O1) is sent with: (a) no token; (b) T2; (c) an empty token; (d) the ADMIN cookie and no token; (e) the STAFF cookie and no token; (f) the CUSTOMER cookie and no token; and TRANSFER(`doi-99999`) with T1 and TRANSFER(`Bad_ID`) with T1; each is also sent as `GET /api/v1/orders/<id>` with the same headers.
**Then**: Every TRANSFER answers 404 `{"error":"not_found","message":S}` with the one same sentence S, equal byte for byte to the matching GET's 404 body; no body holds an id, a token, `DOI-`, ACC, BIN or HOLDER.
Traces: SC-13 (PAY-020 AC-9, permission row; ORD-008)
Check: fetch_url

**Scenario**: TC-38 — a paid, a cancelled and a non-bank order answer 409 and reveal no account
**Given**: P-BANK or P-ALL; RESET; bank orders O1 (paid, `placed`) and O2 (`status = 'cancelled'`), a VNPay order O3 and a MoMo order O4 (FX-MOMO form if MoMo is not listed), each with its token.
**When**: TRANSFER(<each>) is sent with its own token.
**Then**: Each answers 409 `{"error":"conflict","message":<non-empty sentence>}` with no `errors` key; no body holds ACC, BIN, HOLDER or `000201`.
Traces: SC-13 (PAY-020 item 4; ORD-007 item 8)
Check: fetch_url

**Scenario**: TC-39 — the account no longer set: 503 `payment_unavailable`, after the access check
**Given**: A profile without the bank account (P-VNPAY, P-MOMO or P-NONE); an FX-BANK order arranged by `UPDATE orders SET payment_method_id = 'bank'` (token T1) — the account was removed after the order was placed.
**When**: TRANSFER(O1) is sent with T1; then with no token.
**Then**: The first answers 503 `{"error":"payment_unavailable","message":<sentence>}` with no `errors` key and no account detail; the second answers 404 `not_found` (a stranger learns nothing, even about the settings).
Traces: SC-13 (PAY-020 item 1; story §3 item 7)
Check: fetch_url

**Scenario**: TC-40 — no other call of the shop carries the bank account
**Given**: P-BANK or P-ALL; a bank order OB (T1) and the ADMIN cookie.
**When**: These are read and their bodies searched: `GET /api/v1/payment-options`; `GET /api/v1/payments`; `GET /api/v1/payments/bank`; the placing answer of OB; `GET /api/v1/orders/<OB>` with T1; `GET /api/v1/orders/<OB>` and `GET /api/v1/orders?limit=100` as ADMIN; `GET /api/v1/account/orders` as the linked customer; the page `/orders/<OB>` rendered with T1 before the block loads (the HTML shell).
**Then**: No body contains ACC, BIN, HOLDER, BANKNAME or `000201` (the start of every VietQR payload). `GET /api/v1/payments/bank` is the record `bank` with the name `Bank transfer (VietQR)` and its note only.
Traces: SC-13 (story §3 item 7)
Check: fetch_url

**Scenario**: TC-41 — after the dealer marks a bank order paid the page shows the paid view with no block; a cancelled bank order shows the cancelled view with no block
**Given**: P-BANK or P-ALL; RESET; HOLD(O1) and HOLD(O2), bank orders `placed`; ADMIN marks O1 paid with `PUT /api/v1/orders/<O1>` `{"paid":true}` and cancels O2 with `PUT` `{"status":"cancelled"}`; HOLD(O3), a bank order DUE(O3, '25 hours') that the 24-hour rule then cancels (COMPANION run).
**When**: A 1280×800 browser opens `/orders/<O1>`, `/orders/<O2>` and `/orders/<O3>`; `[data-testid='order-status']`, the "Payment" row, `[data-testid='transfer-block']`, `[data-testid='pay-now']` and the request log are read.
**Then**: O1: the h1 reads exactly `Your order is confirmed.`; the "Payment" row reads `Bank transfer (VietQR) · paid`; no `transfer-block`; no `pay-now`; the request log holds no TRANSFER call. O2 and O3: the h1 reads exactly `This order was cancelled.`; no `transfer-block`; no `pay-now`.
Traces: SC-14 (PAY-020 AC-5, item 4; ORD-007 AC-6, AC-11)
Check: interact_page

**Scenario**: TC-42 — while the details load, the rest of the page is drawn and the block is a quiet placeholder of the QR's size
**Given**: P-BANK or P-ALL (or STUB of TRANSFER with a real answer); HOLD(OB); DELAY(TRANSFER, 2500).
**When**: `/orders/<id>` is opened and read at 0.5 s, 1.5 s and 4 s.
**Then**: At 0.5 s and 1.5 s `[data-testid='order-status']` and `[data-testid='order-total']` are visible, there is an element with `aria-busy="true"` inside `main` in the block's place, and none of `[data-testid='order-missing']`, `[data-testid='order-load-error']`, `[data-testid='transfer-failed']` and `[data-testid='pay-now']` exists. At 4 s `transfer-qr` is visible, the placeholder is gone, and the placeholder's bounding box (read at 1.5 s) equals the QR's (read at 4 s) in width and height to within 4 px.
Traces: SC-15 (PAY-020 failure rows; story §6)
Check: interact_page

**Scenario**: TC-43 — when the details cannot be read the block says so in an alert and nothing else changes; when the account was removed the block is not drawn
**Given**: HOLD(OB) with a bank order; the page's normal reading of `/orders/<id>` taken first as the reference (`order-status` text, `order-total` text, the lines).
**When**: The page is opened with (a) STUB(TRANSFER, 500 `{"error":"internal","message":"x"}`); (b) ABORT(TRANSFER); (c) STUB(TRANSFER, 503 `{"error":"payment_unavailable","message":"x"}`).
**Then**: (a) and (b): an element `[data-testid='transfer-failed']` with `role="alert"` reads exactly `We could not load the bank transfer details. Please refresh the page.`; no `transfer-qr`; no `order-missing`; `order-status` and `order-total` read as in the reference; no `pay-now`. (c): no `transfer-block`, no `transfer-failed`, no `role="alert"` in the main region, `order-status` and `order-total` as in the reference.
Traces: SC-15 (PAY-020 failure rows; story §6)
Check: interact_page

**Scenario**: TC-44 — at 390 px and 320 px the QR is whole and the details stack under it; from 1024 px they sit beside it; the page does not scroll sideways
**Given**: HOLD(OB) with a bank order; the details are the run's own when they are already long (an account number of 19 digits and a holder name of 40 characters or more); otherwise STUB(TRANSFER) answers the real answer with the account number replaced by `1234567890123456789`, the holder by `NGUYEN THI MINH ANH VAN LONG TRAN DANG KHOA`, and the bank by `Ngan hang Thuong mai Co phan Ky Thuong Viet Nam`; windows 390×844, 320×640 and 1024×768.
**When**: `/orders/<id>?placed=1` is read at each width: `document.documentElement.scrollWidth`, `window.innerWidth`, the bounding boxes of `transfer-qr` and the five detail elements, and the QR element's `scrollWidth` and `clientWidth`.
**Then**: At 390 and 320: `scrollWidth` is at most `innerWidth`; the QR box has `left` ≥ 0, `right` ≤ `innerWidth`, equal width and height (to within 1 px) of at least 160 px, and `scrollWidth` equals `clientWidth` (not cropped); every detail box has `top` ≥ the QR's `bottom` and `right` ≤ `innerWidth`; the long values wrap inside their boxes. At 1024: the first detail's `left` ≥ the QR's `right` and its `top` < the QR's `bottom` (beside it).
Traces: SC-16 (Responsive NFR)
Check: interact_page

**Scenario**: TC-45 — the QR has an accessible name and every detail is real, selectable text
**Given**: HOLD(OB) with a bank order `DOI-24818`; `/orders/doi-24818?placed=1` open.
**When**: `[data-testid='transfer-qr']` is inspected for its role and accessible name; for each of the five detail elements the contents are selected (`Range.selectNodeContents`) and `getSelection().toString()` is read; the QR element's descendants are searched for text.
**Then**: The QR element has role `img` (or is an `img` or `svg` with `role="img"`) and the accessible name exactly `QR code to pay order DOI-24818 by bank transfer`; each selection reads its value (HOLDER, BANKNAME, ACC, `3.140.000 ₫`, `DOI-24818`); the QR element holds none of those values as text.
Traces: SC-16 (Accessibility NFR)
Check: interact_page

**Scenario**: TC-46 — the five details can be selected and copied by the visitor
**Given**: As TC-45.
**When**: The computed `user-select` of `[data-testid='transfer-holder']`, `-bank`, `-account`, `-amount` and `-note` is measured.
**Then**: None computes `user-select: none`; each element's `display` is not `none` and its `visibility` is `visible`.
Traces: SC-16 (PAY-020 item 2: "written beside it for typing by hand")
Check: measure_styles

---

## D. The dealer marks a bank-transfer order paid (SC-17 … SC-22)

Every case here arranges its orders with FX-BANK (the arranged form is allowed), signs in with the named session, reads the shelf before and after, and ends with RESET.

**Scenario**: TC-47 — an admin and a staff account mark an unpaid bank order paid; it becomes confirmed and nothing else changes
**Given**: RESET; FX-BANK orders O1 and O2, `placed`, unpaid (tokens T1, T2); the dealer views of both read as B1 and B2 and the shelf for `38` read as N.
**When**: ADMIN sends `PUT /api/v1/orders/<O1>` `{"paid":true}`; STAFF sends the same for O2; then each order is read with its token and as ADMIN.
**Then**: Both answer 200 with the dealer view: `paid` true, `status` `confirmed`, `payment` `bank`, and every other member equal to B1 / B2 (`total`, `carrier`, `tracking`, `eta`, `customer`, `city`, `phone`, `address`, `note`, `items`, `transaction_no` `""`). The token read shows `paid` true and `status` `confirmed`; the shelf is as before (no pair taken or returned).
Traces: SC-17 (PAY-020 item 3; ADMIN-016 item 4, AC-11)
Check: fetch_url

**Scenario**: TC-48 — on a confirmed, packed, shipped or delivered bank order the status stays and `paid` becomes true
**Given**: RESET; four unpaid FX-BANK orders set (SET) to `confirmed`, `packed`, `shipped`, `delivered`; SNAPSHOT of each.
**When**: ADMIN sends `PUT {"paid":true}` to each.
**Then**: Each answers 200 with `paid` true and `status` unchanged (`confirmed`, `packed`, `shipped`, `delivered`); the shelf is unchanged; nothing else in the order differs.
Traces: SC-17 (PAY-020 item 3)
Check: fetch_url

**Scenario**: TC-49 — a bank order the dealer cancelled and then marks paid stays cancelled with `paid` true, its pairs neither taken nor returned again; marking a paid one paid again changes nothing
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 5); FX-BANK order O1 (shelf 4); FX-BANK order O2 marked paid once (SNAPSHOT S2).
**When**: ADMIN sends `PUT {"status":"cancelled"}` to O1 (shelf read), then `PUT {"paid":true}` to O1 (shelf and view read); ADMIN sends `PUT {"paid":true}` to O2 again.
**Then**: After the cancel the shelf reads 5. After the mark: 200, `status` `cancelled`, `paid` true; the shelf still reads 5 (not 4, not 6); the token read shows `cancelled` and `paid` true. O2's second `PUT` answers 200 and SNAPSHOT(O2) equals S2 (`updated_at` unchanged).
Traces: SC-18 (PAY-020 conflict row; PAY-002 item 9)
Check: fetch_url

**Scenario**: TC-50 — a bank order the 24-hour rule cancelled and then marked paid stays cancelled with `paid` true, its pairs not returned again
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 5); FX-BANK order O1 (shelf 4), DUE(O1, '25 hours'); the companions; the check run (both companions cancelled); the shelf read (5).
**When**: ADMIN sends `PUT {"paid":true}` to O1; the order and the shelf are read.
**Then**: 200 with `status` `cancelled`, `paid` true; the shelf still reads 5.
Traces: SC-18 (PAY-020 conflict row, item 5)
Check: fetch_url

**Scenario**: TC-51 — every other way to set `paid` stays refused with 422 and applies nothing
**Given**: RESET; unpaid orders: OV (VNPay), OM (MoMo, FX-MOMO form), OC (`UPDATE orders SET payment_method_id = 'cod'` on a placed order), OB (bank); a paid bank order PB and a paid VNPay order PV (`paid = true`); each with `note` `""` and `status` `placed`; SNAPSHOT of each.
**When**: ADMIN sends `PUT {"paid":true}` to OV, OM and OC; `PUT {"paid":false}` to OB, PB, OV and PV; `PUT {"paid":true,"note":"x","status":"confirmed"}` to OV; `PUT {"transaction_no":"1"}` to OB; `PUT {"paid":true,"transaction_no":"1"}` to OB.
**Then**: Every one answers 422 `{"error":"invalid",…}`; each `errors.paid` (where `paid` was sent) reads exactly `Only the payment provider marks this order paid.`; the `transaction_no` bodies carry `errors.transaction_no` (non-empty); every SNAPSHOT is equal afterwards (OV still `placed`, note `""`, unpaid; PB and PV still paid; OB still unpaid).
Traces: SC-19 (ADMIN-016 item 4, AC-11, AC-12; owner decision)
Check: fetch_url

**Scenario**: TC-52 — no session, a forged session, a customer (even the order's own) and the order's token alone cannot mark an order paid
**Given**: RESET; an unpaid bank order OB placed by CUSTOMER with the cookie (so it is linked to that account; token T1); `FORGED`.
**When**: `PUT /api/v1/orders/<OB>` `{"paid":true}` is sent with (a) no cookie, (b) `FORGED`, (c) the CUSTOMER cookie, (d) no cookie and `X-Order-Token: T1`; and `PUT /api/v1/orders/doi-99999` `{"paid":true}` with no cookie.
**Then**: (a), (b), (d) and the unknown id answer 401 `{"error":"unauthenticated","message":<non-empty>}` with no `errors` key (an unknown id is 401, not 404, to a visitor); (c) answers 403 `{"error":"forbidden",…}`; after each the order is unpaid and `placed`.
Traces: SC-20 (ADMIN-004 AC-6, AC-7; PAY-020 permission row)
Check: fetch_url

**Scenario**: TC-53 — two mark-paid requests at the same moment leave the order paid once, confirmed once, and neither fails
**Given**: RESET; ADMIN signed in a page; ten rounds, each with its own fresh unpaid FX-BANK order at `placed`.
**When**: In each round two `PUT /api/v1/orders/<id>` `{"paid":true}` are issued in one batch (`Promise.all`).
**Then**: In every round both answer 200 (never 409, 422 or 5xx); the order reads `paid` true and `status` `confirmed`; the shelf is unchanged; `SELECT count(*) FROM orders WHERE id = '<id>' AND paid AND status = 'confirmed'` is 1.
Traces: SC-21 (PAY-020 conflict row; PAY-002 item 6)
Check: interact_page

**Scenario**: TC-54 — a mark-paid arriving as the 24-hour check cancels the order ends as exactly one of two outcomes, never with the pairs returned twice
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 10); three FX-BANK orders D1, D2, D3 (shelf 7), each DUE(…, '25 hours') at the same moment t0; the companions created and DUE at t0; ADMIN signed in a page.
**When**: `PUT {"paid":true}` is sent for D1 at t0, for D2 at t0 + 20 s and for D3 at t0 + 40 s; the companions are polled until both are `cancelled`; 70 s after that D1–D3 and the shelf are read.
**Then**: Each `PUT` answered 200. Each order is in exactly one of two states: **paid and left alone** (`paid` true, `status` `confirmed`, its pair still taken) or **cancelled once, then paid** (`paid` true, `status` `cancelled`, its pair returned once). The shelf for `38` is 7 + the number of cancelled orders, never more; no order is `cancelled` with `paid` false after a 200.
Traces: SC-21 (PAY-020 conflict row; PAY-002 items 6, 9)
Check: interact_page

**Scenario**: TC-55 — on the dealer's order page "Mark as paid" sits beside "Awaiting payment", sends only `{"paid": true}`, shows "Saving…" while it waits, then the page reads "Paid"
**Given**: Any profile with VNPay's keys (the arranged form of FX-BANK is allowed; the page does not read the bank account); FX-BANK orders at `placed` (O1) and, set by SET, at `confirmed`, `packed`, `shipped`, `delivered`, `cancelled` (O2…O6), all unpaid; a 1280×800 browser signed in as ADMIN; `/admin/orders/<O1>` open on the Order tab; DELAY(`PUT /api/v1/orders/<O1>`, 1500).
**When**: The tab's payment block is read; `[data-testid='admin-mark-paid']` is pressed once and read at 0.5 s; after the answer the block, the status control and the page are read; the page is reloaded; then `/admin/orders/<O2>` … `<O6>` are opened and the button counted.
**Then**: Before the press the block reads `Awaiting payment` and `[data-testid='admin-mark-paid']` is visible, reads exactly `Mark as paid` and sits in the same block. At 0.5 s the button is disabled and reads `Saving…`. The request log holds exactly one `PUT /api/v1/orders/<O1>` whose JSON body has the single member `paid` equal to `true`. After the answer the block reads `Paid` (no `Awaiting payment`), the button is gone, `Saved.` is shown and the status control reads `Confirmed`; after the reload the block still reads `Paid`. For O2 … O6 the button is present (count 1) whatever the status.
Traces: SC-22 (ADMIN-016 item 4, AC-9, AC-10; PAY-020 item 3)
Check: interact_page

**Scenario**: TC-56 — a refusal or a network failure says so in an alert and enables the button again; an ended session says to sign in again
**Given**: As TC-55 with a fresh unpaid FX-BANK order `placed` open on the Order tab.
**When**: The button is pressed under each of: STUB(PUT, 422 `{"error":"invalid","message":"x","errors":{"paid":"Only the payment provider marks this order paid."}}`); STUB(PUT, 500 `{"error":"internal","message":"x"}`); ABORT(PUT); then STUB(PUT, 401 `{"error":"unauthenticated","message":"x"}`); then STUB(PUT, 403 `{"error":"forbidden","message":"x"}`).
**Then**: After the 422, the 500 and the abort an element with `role="alert"` reads exactly `We could not save. Please try again.`, the button reads `Mark as paid` and is enabled, and the block still reads `Awaiting payment`. After the 401 and the 403 an element with `role="alert"` reads exactly `Your session ended. Sign in again.`. The order is still unpaid in the database after every one.
Traces: SC-22 (ADMIN-016 item 4; failure behaviour)
Check: interact_page

**Scenario**: TC-57 — no other order has a "Mark as paid" button or any control that changes payment, and a MoMo order reads "MoMo transaction <number>"
**Given**: Any profile with VNPay's keys for the first half (orders arranged with FX-BANK / FX-MOMO); P-MOMO or P-ALL for the MoMo-transaction half; ADMIN signed in a browser; unpaid VNPay order OV, unpaid MoMo order OM, a paid bank order PB (paid by `PUT`), a paid VNPay order PV; a MoMo success MNOTICE for a second MoMo order PM (transaction number `7000000001`) and, if TMN/SECRET are held, a VNPay success notice for PV with number `14123456`.
**When**: `/admin/orders/<id>` is opened for OV, OM, PB, PV and PM; the page's `button`, `select`, `input[type=checkbox]`, `[role=switch]` controls and `[data-testid='admin-mark-paid']` are listed; the dealer's read `GET /api/v1/orders/<PM>` is taken.
**Then**: `[data-testid='admin-mark-paid']` does not exist on OV, OM, PB, PV or PM; no control on any of them is labelled or named for payment or `paid`. PB and PV read `Paid`, OV and OM read `Awaiting payment`. PM reads `MoMo transaction 7000000001` (and PV `VNPay transaction 14123456` when held); the dealer's read of PM carries `transaction_no` `7000000001`. Without MSK the PM half is UNVERIFIED (instrument lacked: MoMo's secret key).
Traces: SC-22 (ADMIN-016 item 4, AC-9, AC-10; story §4 "dealer's reads")
Check: interact_page

---

## E. MoMo's notice is the only thing that marks a MoMo order paid (SC-23 … SC-29)

Every case in this group needs MSK (a notice cannot be signed without it) and signs with MNOTICE in the page, so each is `interact_page`; database reads use `set_db_state`. A MoMo order is placed with PLACE-MOMO against the stand-in (normal mode), so its first reference is `DOI-24818-1` (**R1**); a second attempt made with "Pay now" is `-2` (**R2**). MNOTICE carries no cookie and no token, so "no session and no token" is true by construction. After every case RESET.

**Scenario**: TC-58 — a correctly signed success notice answers 204, marks the order paid and confirmed, and keeps MoMo's transaction; an order already packed stays packed
**Given**: P-MOMO or P-ALL; RESET; P3140; O1 placed with PLACE-MOMO (`placed`, unpaid); O2 a second MoMo order set to `packed`; SNAPSHOT of both.
**When**: MNOTICE(R1) is sent with `amount` 3140000, `resultCode` 0 and `transId` 7000000001; then MNOTICE(R of O2) with `transId` 7000000002 and O2's total.
**Then**: Both answer HTTP 204 with an empty body. O1 reads `paid` true and `status` `confirmed`; O2 reads `paid` true and `status` `packed`. `SELECT transaction_no, amount, response_code, is_success, paid_time FROM payment_notices` for O1's attempt is one row: `7000000001`, 3140000, `0`, true, a time within five minutes of now. No stored column of any row holds a phone, wallet or card number (no value of twelve or more digits).
Traces: SC-23 (PAY-021 AC-4, item 3; PAY-002 AC-1, AC-2; Privacy NFR)
Check: interact_page

**Scenario**: TC-59 — a signed success notice for the first reference still finds the order and pays it after later attempts
**Given**: P-MOMO or P-ALL; RESET; O1 placed with PLACE-MOMO (R1); "Pay now" made twice more (R2, R3).
**When**: MNOTICE(R1) with the right amount is sent.
**Then**: 204; O1 reads `paid` true and `status` `confirmed`; the notice row belongs to the attempt of R1.
Traces: SC-7 (PAY-021 item 2: "every earlier reference still leads to the order"), SC-23
Check: interact_page

**Scenario**: TC-60 — the same notice a second time answers 204 and changes nothing
**Given**: As TC-58 after the first notice for O1; SNAPSHOT(O1) taken after it.
**When**: The identical request (same body, same signature) is sent twice more.
**Then**: Both answer 204 with an empty body; SNAPSHOT(O1) is equal (`updated_at` unchanged); the table holds exactly one successful notice for the order.
Traces: SC-24 (PAY-021 AC-5; PAY-002 AC-3)
Check: interact_page

**Scenario**: TC-61 — two copies, and twenty, sent at the same moment leave the order paid once with one successful record and no server error
**Given**: P-MOMO or P-ALL; RESET; two fresh unpaid MoMo orders, one for each run.
**When**: Run one: two identical MNOTICE requests are issued in one batch (`Promise.all`). Run two: twenty identical requests in one batch.
**Then**: In both runs every request answers 204 (none 5xx, none 400); the order reads `paid` true and `status` `confirmed`; `SELECT count(*) FILTER (WHERE is_success) FROM payment_notices` for the order's attempts is exactly 1; the order's `updated_at` moved once (a single write).
Traces: SC-24 (PAY-021 AC-5; PAY-002 AC-4)
Check: interact_page

**Scenario**: TC-62 — every refused notice answers 400, leaves the order unpaid, keeps nothing and echoes nothing
**Given**: P-MOMO or P-ALL; RESET; O1 unpaid MoMo (R1, total 3140000); SNAPSHOT(O1), the shelf and the count of `payment_notices`.
**When**: Nine requests are sent, each an otherwise correct MNOTICE(R1) except: (a) `signature` altered by one character after signing; (b) no `signature` member; (c) signed with `MSK + "x"`; (d) `partnerCode` `OTHER` (correctly signed with MSK); (e) `amount` 1000 (correctly signed); (f) `orderId` `DOI-99999-1` (correctly signed); (g) the body `not json`; (h) valid JSON lacking `amount`; (i) an empty body.
**Then**: Each answers 400; no answer's body contains the request's `signature`, its `orderId`, `DOI-24818`, MSK, MAK or MPC; SNAPSHOT(O1) is equal after each; the count of `payment_notices` is unchanged and the shelf is unchanged.
Traces: SC-25 (PAY-021 AC-6, item 3; PAY-002 AC-5 to AC-7)
Check: interact_page

**Scenario**: TC-63 — the checks run signature, then reference, then amount, then "already processed"
**Given**: P-MOMO or P-ALL; RESET; O1 paid by a first success notice (answered 204); SNAPSHOT(O1) after it; O2 unpaid (R of O2).
**When**: For the **paid** O1: (a) a correctly signed notice for O1 with `amount` 1000; (b) the genuine notice with its signature altered by one character; (c) a correctly signed notice with `partnerCode` `OTHER`; then (d) the genuine notice unchanged. For the **unpaid** O2: (e) a notice with a bad signature **and** `amount` 1000; (f) a correctly signed notice with `amount` 1000 **and** an unknown reference.
**Then**: (a), (b), (c), (e), (f) answer 400 — never 204 (a refusal for the signature, partner code, reference or amount comes before "already processed"); (d) answers 204; SNAPSHOT(O1) equal throughout; O2 stays unpaid with no record.
Traces: SC-25 (PAY-002 AC-5 to AC-7; "a notice wrong in two ways is refused for the first")
Check: interact_page

**Scenario**: TC-64 — a signed notice that says the payment failed answers 204, leaves the order unpaid and is kept; the next attempt's success then pays it
**Given**: P-MOMO or P-ALL; RESET; O1 placed with PLACE-MOMO (R1, `placed`); a second attempt R2 made with "Pay now"; SNAPSHOT(O1).
**When**: MNOTICE(R1) with `resultCode` 1006 and `message` `Transaction is denied by user.` is sent; the order and `payment_notices` are read; then MNOTICE(R2) with `resultCode` 0 is sent.
**Then**: The first answers 204; O1 reads `paid` false and `status` `placed` (unchanged `updated_at`); one notice row for R1's attempt with `is_success` false and `response_code` `1006`. The second answers 204; O1 reads `paid` true and `status` `confirmed`.
Traces: SC-26 (PAY-021 item 3; PAY-002 AC-8)
Check: interact_page

**Scenario**: TC-65 — a signed success notice for an order already cancelled answers 204, records the payment, and the order stays cancelled with its pairs untouched
**Given**: P-MOMO or P-ALL; RESET; SHELF(`samba-og-w-core-black`, `38`, 5); O1 placed with PLACE-MOMO (shelf 4); the dealer's cancel simulated (`SET status = 'cancelled'` and `pairs = pairs + 1`; shelf 5); SNAPSHOT.
**When**: MNOTICE(R1) with the right amount is sent.
**Then**: 204; O1 reads `paid` true and `status` `cancelled`; one successful notice is kept; the shelf for `38` still reads 5 (not taken again, not returned again).
Traces: SC-27 (PAY-021 item 3; PAY-002 AC-11, item 9)
Check: interact_page

**Scenario**: TC-66 — when a store write fails the notice answers a server error, nothing is half-written, and the same notice afterwards is processed normally
**Given**: P-MOMO or P-ALL; RESET; O1 unpaid MoMo (R1); `ALTER TABLE payment_notices RENAME TO payment_notices_x` (renamed back afterwards); SNAPSHOT(O1).
**When**: MNOTICE(R1) success is sent; the order is read; the table is renamed back; the identical request is sent again.
**Then**: The first answers 500 (a failed query is `internal`), never 2xx or 400, and its body holds no `payment_notices`, `relation`, `SELECT` or `INSERT`; O1 is still unpaid at `placed` (the order was not marked paid without its record). The second answers 204; O1 reads `paid` true and `status` `confirmed`; exactly one successful notice row.
Traces: SC-28 (PAY-002 failure rows "nothing is half-written"; contract: a failed query is 500)
Check: interact_page

**Scenario**: TC-67 — with the database refusing connections the notice answers a server error so MoMo sends it again, and the same notice is processed once the database is back
**Given**: P-MOMO or P-ALL; RESET; O1 unpaid MoMo (R1); the verifier makes the database unavailable with `set_db_available` while the backend keeps running.
**When**: MNOTICE(R1) success is sent; the database is made available again; the identical request is sent.
**Then**: The first answers 5xx (500 or 503) and never 2xx or 400, with a body that holds no connection string, host or SQL; the second answers 204 and O1 reads `paid` true and `status` `confirmed` with one successful notice.
Traces: SC-28 (PAY-002 failure rows "Upstream failure"; story decision 8: 5xx so MoMo sends it again)
Check: interact_page

**Scenario**: TC-68 — with MoMo's keys unset the notice address refuses every request with 400, changes nothing, and the server still starts
**Given**: P-VNPAY, P-BANK or P-NONE (MoMo's three keys not all set); an order O1 arranged as FX-MOMO (unpaid; reference `DOI-24818-1` in a `payment_attempts` row inserted for it); SNAPSHOT.
**When**: `GET /healthz`; then `POST /api/v1/payments/momo/ipn` with (a) an empty body, (b) a JSON body with the members of an MNOTICE for `DOI-24818-1` signed with an invented secret, (c) `not json`.
**Then**: `/healthz` answers 200. All three answer 400 with no secret in the body; SNAPSHOT(O1) is equal afterwards and the shop shows no 5xx.
Traces: SC-28 (PAY-021 item 1; "with MoMo's keys unset the notice address refuses every request with 400")
Check: fetch_url

**Scenario**: TC-69 — the browser never marks anything paid: the return address and "Pay now" leave a MoMo order unpaid after 60 seconds
**Given**: P-MOMO or P-ALL; RESET; HOLD(O1) with a MoMo order, unpaid, `placed`; SNAPSHOT(O1).
**When**: The browser opens RETURN-MOMO(R1, 0) with an arbitrary `signature`; the page is left open for 60 s; during that time `POST /api/v1/orders/doi-24818/payment` is sent with the token; the order is read from the database and with its token at 60 s.
**Then**: At 60 s O1 reads `paid` false and `status` `placed`; no `payment_notices` row exists for it; the only requests the page sent about the order are `GET /api/v1/orders/doi-24818` (and the one `POST …/payment`): none marks anything.
Traces: SC-29 (PAY-021 item 3; PAY-004 AC-9)
Check: interact_page

---

## F. Giving up unpaid orders (SC-30 … SC-32)

Every case here uses the COMPANION rule of the observation section to know that both rules have run. The companions are one pair of `samba-og-kids-cloud-white` size `36` each (a different shoe from the ones under test, so the shelves never overlap). Times are arranged with DUE; the shelves are read with `SELECT pairs FROM product_sizes`. After each case RESET.

**Scenario**: TC-70 — an unpaid MoMo order 31 minutes old is cancelled whether placed, confirmed or packed, and its pairs go back once with the shoe's lists agreeing
**Given**: P-MOMO or P-ALL (or the arranged FX-MOMO form: the 30-minute rule reads the order's way, not the settings); RESET; SHELF(`samba-og-w-core-black`, `38`, 2); O1 a MoMo order of that shoe, size `38`, qty 2 (shelf 0, `sizes_out` holds `38`, `stock` S0), `placed`; O2 a MoMo order of `samba-og-w-wonder-quartz` size `38` qty 1 set to `confirmed`; O3 a MoMo order of `gazelle-indoor-solar-gold` size `42` qty 1 set to `packed`; each DUE(…, '31 minutes'); the shelves of O2 and O3 read; the companions created.
**When**: The companions are polled until both are `cancelled`; then O1–O3, the shelves and `GET /api/v1/products/samba-og-w-core-black` are read; 70 s later the shelves are read again.
**Then**: O1, O2, O3 are `cancelled` and `paid` false; the shelf for `38` of the first shoe reads 2, the others have risen by 1 each; the first shoe's record now has `38` in `sizes_low` and not in `sizes_out` and `stock` = S0 + 2, and for each shoe the sum of `pairs` equals `stock`; the shelves read 70 s later are the same (the pairs went back once).
Traces: SC-30 (PAY-021 AC-7, item 4; PAY-003 AC-1, AC-2, AC-5)
Check: fetch_url

**Scenario**: TC-71 — a MoMo order 29 minutes old, a paid one, a cancelled one and an unpaid shipped or delivered one are left alone
**Given**: As TC-70 with: M1 unpaid `placed`, DUE(M1, '29 minutes'); M2 paid, `placed`, DUE(M2, '2 hours'); M3 `status = 'cancelled'` (cancelled by the dealer: pair returned by simulation as in TC-65), DUE(M3, '2 hours'); M4 unpaid `shipped` and M5 unpaid `delivered`, both DUE(…, '2 hours'); SNAPSHOT of each and of their shelves; the companions created at the same moment.
**When**: The companions are polled until both are `cancelled`; M1–M5 and the shelves are read at once.
**Then**: SNAPSHOT equal for M2, M3, M4, M5 (statuses `placed` paid, `cancelled`, `shipped`, `delivered`; `updated_at` unchanged) and their shelves are unchanged (no pair returned for the shipped and the delivered). M1 is still `placed`; if it reads `cancelled`, its `updated_at` is compared with `placed_at + 30 minutes`: earlier fails the case (a premature cancellation), at or after means the check ran after the 30th minute and the case is arranged again.
Traces: SC-30 (PAY-003 AC-3, AC-4, AC-10; PAY-021 item 4)
Check: fetch_url

**Scenario**: TC-72 — an unpaid bank order 25 hours old is cancelled whether placed, confirmed or packed, its pairs go back once and the lists agree
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 2); B1 an FX-BANK order of that shoe size `38` qty 2 (shelf 0), `placed`; B2 an FX-BANK order of `samba-og-w-wonder-quartz` size `38` qty 1 set to `confirmed`; B3 an FX-BANK order of `gazelle-indoor-solar-gold` size `42` qty 1 set to `packed`; each DUE(…, '25 hours'); the companions created.
**When**: The companions are polled until both are `cancelled`; B1–B3, the shelves and the first shoe's record are read; 70 s later the shelves are read again.
**Then**: B1–B3 are `cancelled` and `paid` false; the first shoe's size `38` reads 2, the other two shelves rose by 1 each; the first shoe's `sizes_low` holds `38`, `sizes_out` does not, `stock` = its earlier value + 2, and every shoe's `pairs` sum equals its `stock`; the later reading is equal (returned once). The page half of a rule-cancelled bank order (the cancelled view and no QR block) is TC-41.
Traces: SC-31 (PAY-020 AC-7, item 5; PAY-003 AC-11)
Check: fetch_url

**Scenario**: TC-73 — the 24-hour rule's boundaries: exactly 24 hours is cancelled, 23 hours and 2 hours are not, and a bank order is never touched by the 30-minute rule
**Given**: RESET; FX-BANK orders: E24 DUE(…, '24 hours'); E23 DUE(…, '23 hours'); E2 DUE(…, '2 hours'); E40 DUE(…, '40 minutes'); P25 paid DUE(…, '25 hours'); S25 `shipped` and D25 `delivered`, both unpaid DUE(…, '25 hours'); SNAPSHOT of each and of the shelves; the companions created.
**When**: The companions are polled until both are `cancelled`; the seven orders and the shelves are read at once.
**Then**: E24 is `cancelled` and its pair is back on the shelf; E23, E2, E40, P25, S25 and D25 have equal SNAPSHOTs (unpaid and unchanged, P25 paid and unchanged, S25 `shipped`, D25 `delivered`) and their shelves are unchanged (no pair returned for any of them).
Traces: SC-31 (PAY-020 AC-7, AC-8, item 5; PAY-003 AC-11)
Check: fetch_url

**Scenario**: TC-74 — the 14 sample orders and their pairs are untouched by both rules, and an order placed through checkout is never a sample
**Given**: RESET; `SELECT count(*) FROM orders WHERE is_sample` reads 14 (`DOI-24804` … `DOI-24817`); three samples are arranged (old values read first and put back): `doi-24816` set to `payment_method_id = 'bank'`, `status = 'placed'`, `paid = false`, `placed_at = now() - interval '30 hours'`; `doi-24815` set to `momo`, `confirmed`, unpaid, 3 hours; `doi-24814` set to `vnpay`, `packed`, unpaid, 3 hours; SNAPSHOT of all 14 (`status, paid, updated_at`) and `SELECT product_id, size, pairs FROM product_sizes` excluding `samba-og-kids-cloud-white`; one order placed through checkout.
**When**: The companions are created, DUE and polled until both are `cancelled`; the 14 snapshots and the sum are read.
**Then**: All 14 snapshots are equal before and after (the three arranged ones included) and the pairs on the shelves are equal; `SELECT is_sample FROM orders WHERE id = 'doi-24818'` for the order placed through checkout reads `false`.
Traces: SC-32 (PAY-003 AC-9, item 4; SRS open question 7)
Check: fetch_url

**Scenario**: TC-75 — an order the dealer cancelled early is not returned again when its deadline passes, under either rule
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 5); a MoMo order M1 and an FX-BANK order B1 of that shoe and size (shelf 3); the dealer cancels both with `PUT {"status":"cancelled"}` as ADMIN (shelf 5); DUE(M1, '31 minutes'), DUE(B1, '25 hours'); SNAPSHOT of both; the companions created.
**When**: The companions are polled until both are `cancelled`; M1, B1 and the shelf are read; three minutes later (three more runs) they are read again.
**Then**: Both times the shelf for `38` reads 5 (not 6, not 7); the snapshots of M1 and B1 are equal (`status` `cancelled`, `updated_at` unchanged: the rules wrote nothing).
Traces: SC-32 (PAY-003 AC-6; "an order the dealer cancelled early is not returned again")
Check: fetch_url

**Scenario**: TC-76 — two checks at the same moment over one due order cancel it once and return its pairs once
**Given**: The PR's expiry code (both rules) and its backend tests.
**When**: The code is read with `get_pr` for how it cancels (a single conditional `UPDATE … WHERE status IN ('placed','confirmed','packed') AND NOT paid` whose changed-row count decides whether pairs are returned, as `orders.Cancel` does) and `run_tests` (backend) is run.
**Then**: A backend test runs two checks concurrently over one due order (MoMo and bank) and asserts the shelf rose by the order's quantity once; the backend suite passes; the code returns pairs only when a row changed.
Traces: SC-32 (PAY-003 AC-8)
Check: manual — the timer is inside the process and no instrument runs two checks at once; the evidence is the PR's backend test run by `run_tests` and the rule's source read from `get_pr`. A repeated check over time is observed in TC-75.

---

## G. The result page for MoMo (SC-33, SC-34)

Unless a case says otherwise: P-MOMO or P-ALL with the stand-in normal; 1280×800; the browser is a HOLD(order) browser (it holds the order's token); the page is opened with RETURN-MOMO(ref, code); "the request log" is read for the page's calls to `GET /api/v1/orders/<id>`. The page's own copy is checked word for word, including the ellipsis `…` (U+2026) and the apostrophe in `can't`. The one h1 is `[data-testid='payment-state']`.

**Scenario**: TC-77 — an order paid by MoMo's notice shows "Payment received." whatever the address says
**Given**: RESET; HOLD(O1) with a MoMo order `DOI-24818`; a success MNOTICE(R1) has been answered 204.
**When**: The browser opens RETURN-MOMO(R1, 0); then RETURN-MOMO(R1, 1006); then `/payment/return?orderId=R1` with no result code.
**Then**: Each time `[data-testid='payment-state']` reads exactly `Payment received.` with `data-state="paid"`; `[data-testid='payment-line']` reads exactly `Your order DOI-24818 is paid. We will pack it next.`; `[data-testid='payment-view-order']` reads `View your order` and has `href` ending `/orders/doi-24818`; there is no `[data-testid='payment-try-again']`. The request log holds a `GET /api/v1/orders/doi-24818` carrying `X-Order-Token` (the page asked the shop and did not decide from the address).
Traces: SC-33 (PAY-021 AC-8; PAY-004 AC-1)
Check: interact_page

**Scenario**: TC-78 — "confirming" while MoMo's notice has not arrived, then "Payment received." with no reload
**Given**: RESET; HOLD(O1) MoMo, unpaid; a marker `window.__keep = 1` is set after the page loads (it vanishes on a reload).
**When**: RETURN-MOMO(R1, 0) is opened; at 1 s the page is read; at 6 s a success MNOTICE(R1) is sent from the page's script; the page is read at 8 s and 10 s.
**Then**: At 1 s the h1 reads exactly `We are confirming your payment…` (`data-state="confirming"`), `[data-testid='payment-line']` reads exactly `This usually takes a few seconds. Please do not pay again.`, `payment-view-order` exists and there is no `payment-try-again`. The request log shows `GET /api/v1/orders/doi-24818` repeated about every 3 s (gaps between 2 s and 4 s). By 10 s the h1 reads `Payment received.` (`data-state="paid"`) and `window.__keep` is still `1` (no reload).
Traces: SC-33 (PAY-021 AC-8; PAY-004 AC-2, AC-3)
Check: interact_page

**Scenario**: TC-79 — after 30 seconds with no notice the page says MoMo has not been heard from, and stops asking
**Given**: RESET; HOLD(O1) MoMo, unpaid, `placed`; RETURN-MOMO(R1, 0) opened at t0.
**When**: The page is read at 29 s, 31 s and 45 s; the request log's `GET /api/v1/orders/doi-24818` calls are counted for 31–45 s; at 60 s the order is read from the database.
**Then**: At 29 s the h1 still reads `We are confirming your payment…`. At 31 s and 45 s it reads exactly `We have not heard from MoMo yet.` (`data-state="timed-out"`), the line reads exactly `If money left your account it will show on your order once MoMo confirms it. Please do not pay again.`, `payment-view-order` is present and there is no `payment-try-again`. No `GET /api/v1/orders/doi-24818` is sent between 31 s and 45 s. At 60 s `paid` is false and `status` `placed`.
Traces: SC-33 (PAY-021 AC-8; PAY-004 AC-4)
Check: interact_page

**Scenario**: TC-80 — the provider the page names is the one the shop says the order uses, never one read from the address
**Given**: RESET; HOLD(OM) a MoMo order and HOLD(OV) a VNPay order (VNPay's keys set), both unpaid.
**When**: OM is opened with a VNPay-shaped address RETURN-VNPAY(R1, `00`) and read at 31 s; OV is opened with a MoMo-shaped address RETURN-MOMO(R1, 0) and read at 31 s; then OM with RETURN-VNPAY(R1, `24`) and OV with RETURN-MOMO(R1, 1006) are read at once.
**Then**: For OM at 31 s the h1 reads `We have not heard from MoMo yet.` and the page text holds no `VNPay`; for OV at 31 s the h1 reads `We have not heard from VNPay yet.` and the text holds no `MoMo`; the two not-completed views read `Payment was not completed.` with `You can try again, or place a new order.` and name neither provider. If VNPay's keys are not set the OV half is UNVERIFIED (instrument lacked: a VNPay order, which needs VNPay's keys).
Traces: SC-33 (story decision 9; PAY-004 item 1)
Check: interact_page

**Scenario**: TC-81 — a MoMo payment that was not completed shows "Try again", which asks for a new MoMo address and goes to it
**Given**: RESET; HOLD(O1) MoMo, unpaid, `placed`; the attempts of O1 counted as N; RETURN-MOMO(R1, 1006) open; DELAY(`POST /api/v1/orders/doi-24818/payment`, 1500).
**When**: The page is read; `[data-testid='payment-try-again']` is pressed once; it is read at 0.5 s; the next navigation is read.
**Then**: The h1 reads exactly `Payment was not completed.` (`data-state="not-completed"`), the line reads exactly `You can try again, or place a new order.`, `payment-view-order` is present and `payment-try-again` reads `Try again` and is enabled. At 0.5 s the button is `disabled`; the request log holds exactly one `POST /api/v1/orders/doi-24818/payment` with `X-Order-Token`; the browser then navigates to the stand-in's `payUrl` for `DOI-24818-2`; `SELECT count(*) FROM payment_attempts` for the order is N + 1.
Traces: SC-33 (PAY-021 AC-8; PAY-004 AC-5, AC-6)
Check: interact_page

**Scenario**: TC-82 — a cancelled MoMo order shows "This order was cancelled." with "Keep shopping" and no "Try again"
**Given**: RESET; HOLD(O1) MoMo with `status = 'cancelled'`, unpaid; HOLD(O2) MoMo with `status = 'cancelled'` and `paid = true` (a payment that arrived late).
**When**: RETURN-MOMO(R1, 1006) is opened for O1 and RETURN-MOMO(R2, 0) for O2.
**Then**: For both: the h1 reads exactly `This order was cancelled.` (`data-state="cancelled"`); the line reads exactly `The time to pay ran out, so the pairs went back on the shelf. You can order again.`; `payment-view-order` is present; `[data-testid='payment-keep-shopping']` reads `Keep shopping` and has `href` `/shop`; there is no `payment-try-again`.
Traces: SC-33 (PAY-021 AC-8; PAY-004 AC-7)
Check: interact_page

**Scenario**: TC-83 — the VNPay page's behaviour and words are unchanged
**Given**: VNPay's keys set; HOLD(OV) a VNPay order, unpaid, `placed` (reference R1); the page's earlier behaviour as the VNPay story's cases fix it.
**When**: RETURN-VNPAY(R1, `24`) is opened; then RETURN-VNPAY(R1, `00`) is opened and read at 31 s; then, if SECRET is held, a signed success VNPay notice is sent and RETURN-VNPAY(R1, `00`) is opened again.
**Then**: The first reads `Payment was not completed.` with `You can try again, or place a new order.` and a `Try again` button; the second reads at 31 s `We have not heard from VNPay yet.` with `If money left your account it will show on your order once VNPay confirms it. Please do not pay again.` and no `Try again`; the third reads `Payment received.` and `Your order DOI-24818 is paid. We will pack it next.`. Without SECRET the third part is UNVERIFIED (instrument lacked: VNPay's hash secret).
Traces: SC-33 (PAY-004 AC-1 to AC-5 for VNPay; "The page's VNPay behaviour and words are unchanged")
Check: interact_page

**Scenario**: TC-84 — a MoMo return this browser holds no token for, with no reference, or of neither provider's shape shows the one "can't match" view and nothing about any order
**Given**: RESET; O1 placed in browser A (MoMo, reference R1); O2 placed through the page in browser B (token T2); the browser under test is (a) fresh, (b) holds only O2's token, (c) holds O1's token.
**When**: In (a) RETURN-MOMO(R1, 0); in (b) RETURN-MOMO(R1, 0); in (c) RETURN-MOMO(`DOI-99999-1`, 0); in (c) `/payment/return?resultCode=0`; in (c) `/payment/return?orderId=abc&resultCode=0`; in (c) `/payment/return?foo=bar`.
**Then**: Every one shows the h1 exactly `We can't match this payment to an order on this browser.` (`data-state="no-match"`), the line exactly `Look the order up with its number and phone.` and `[data-testid='payment-track']` reading `Track an order` with `href` `/track`; no `payment-view-order`, no `payment-try-again`; the page text holds none of `DOI-24818`, `DOI-24819`, `paid`, `confirmed`, `₫`, `MoMo`, `VNPay`. In (b) the request log holds no request carrying O2's token for O1's id.
Traces: SC-34 (PAY-021 AC-8; PAY-004 AC-8, permission row)
Check: interact_page

**Scenario**: TC-85 — when the shop cannot be reached or answers 500, the page says it could not check, never a result
**Given**: HOLD(O1) MoMo, unpaid; a second, fresh browser that holds nothing.
**When**: STUB(`GET /api/v1/orders/<id>`, 500 `{"error":"internal","message":"x"}`) and RETURN-MOMO(R1, 0) is opened; then ABORT(`GET /api/v1/orders/<id>`) and it is opened again; then, in the fresh browser, RETURN-MOMO(R1, 0) with ABORT on every `/api/v1/` request.
**Then**: In the two holding runs `[data-testid='payment-state']` has `data-state="check-failed"`; an element with `role="alert"` reads exactly `We could not check your payment. Please open your order to see its state.`; `payment-view-order` is present with `href` ending `/orders/doi-24818`; the page does not show `Payment received.`, `This order was cancelled.` or the `can't match` words, and there is no `payment-try-again`. In the fresh browser the page shows `payment-track` (`Track an order`, `/track`) and no `payment-view-order`, and never shows `Payment received.` or `This order was cancelled.`.
Traces: SC-34 (PAY-004 failure row "Upstream failure")
Check: interact_page

**Scenario**: TC-86 — at 390 px and 320 px every MoMo state fits and the page does not scroll sideways
**Given**: RESET; `UPDATE counters SET value = 99999999999 WHERE name = 'order_number'` (the next number is `DOI-100000000000`, the longest the page will draw; set back by RESET); MoMo orders placed in HOLD browsers for each state: paid, confirming (opened on `resultCode` 0), timed-out (the same after 31 s), not-completed, cancelled, no-match (a fresh browser), check-failed (a stubbed 500); windows 390×844 and 320×640.
**When**: Each state is drawn at each width and read: `document.documentElement.scrollWidth`, `window.innerWidth`, and the bounding boxes of the h1, the line and each of `payment-view-order`, `payment-try-again`, `payment-keep-shopping`, `payment-track`.
**Then**: In every state at both widths `scrollWidth` is at most `innerWidth`; every box has `left` ≥ 0 and `right` ≤ `innerWidth`; the longest line (the timed-out one) wraps inside its box (its `height` is more than one line height).
Traces: SC-34 (PAY-004 AC-10; Responsive NFR)
Check: interact_page

---

## H. Flows, settings and wording (SC-35 … SC-39)

**Scenario**: TC-87 — the bank-transfer flow: choose, pay by QR block, the dealer marks it paid, the customer sees it confirmed
**Given**: P-BANK or P-ALL; RESET; a fresh 1280×800 browser; `samba-og-w-core-black` size `38` available; ADMIN's cookie available to a script.
**When**: The pair is chosen on `/shop/samba-og-w-core-black`; `/checkout` is completed with FORM, `[data-testid='pay-bank']` chosen and `[data-testid='place-order']` pressed; the landing page is read; the verifier sends `PUT /api/v1/orders/doi-24818` `{"paid":true}` as ADMIN, then the identical request again; the customer's browser reopens `/orders/doi-24818`.
**Then**: The browser lands on `/orders/doi-24818?placed=1` with `[data-testid='transfer-block']` visible and no `pay-now`. Both `PUT`s answer 200; the order has been marked paid once (`SELECT count(*) FROM orders WHERE id = 'doi-24818' AND paid AND status = 'confirmed'` is 1 and its `updated_at` did not move on the second `PUT`). The reopened page shows the h1 `Your order is confirmed.`, the "Payment" row `Bank transfer (VietQR) · paid`, no `transfer-block` and no `pay-now`.
Traces: SC-35 (PAY-020 AC-3 to AC-5; PAY-022)
Check: interact_page

**Scenario**: TC-88 — the MoMo flow: choose, go to MoMo, the signed notice, the result page, the order, paid exactly once
**Given**: P-MOMO or P-ALL; the stand-in normal; RESET; a fresh 1280×800 browser; the pair available.
**When**: The pair is chosen; `/checkout` is completed with FORM and `[data-testid='pay-momo']`; `[data-testid='place-order']` is pressed (the browser leaves for the stand-in's address whose reference is R1); the verifier sends the signed success MNOTICE(R1) MoMo would send, sends the identical notice again, and navigates the browser to RETURN-MOMO(R1, 0); then `[data-testid='payment-view-order']` is pressed.
**Then**: The first notice answers 204 and the second 204; the order has exactly one successful notice and `paid` true. The result page shows `Payment received.`; "View your order" lands on `/orders/doi-24818` whose h1 reads `Your order is confirmed.` and whose "Payment" row reads `MoMo wallet · paid`.
Traces: SC-35 (PAY-021 AC-4, AC-8; PAY-022)
Check: interact_page

**Scenario**: TC-89 — `.env.example` names the eight settings with no value
**Given**: The PR's `code/backend/.env.example` (read with `get_pr`).
**When**: The file is read for the names `MOMO_PARTNER_CODE`, `MOMO_ACCESS_KEY`, `MOMO_SECRET_KEY`, `MOMO_API_URL`, `VIETQR_BANK_BIN`, `VIETQR_BANK_NAME`, `VIETQR_ACCOUNT_NUMBER`, `VIETQR_ACCOUNT_NAME`.
**Then**: Each of the eight appears once as a name with nothing after the `=` (or an empty value), and no real key, number or holder name is written in the file or in a comment.
Traces: SC-36 (PAY-020 item 1; PAY-021 items 1, 6)
Check: manual — a repository file is not served by the running story; the evidence is the PR's file read with `get_pr`.

**Scenario**: TC-90 — with none of the eight settings set the server starts, `/healthz` answers 200 and neither way is listed
**Given**: P-VNPAY or P-NONE (none of `MOMO_*` and `VIETQR_*` set).
**When**: `GET /healthz` and `GET /api/v1/payment-options` are sent.
**Then**: `/healthz` answers 200; the options list no `bank` and no `momo` item (VNPay alone; `is_available` is true in P-VNPAY and false in P-NONE).
Traces: SC-36 (PAY-020 item 1; PAY-021 item 1)
Check: fetch_url

**Scenario**: TC-91 — MoMo's secret key and access key appear in no served file and no API answer, the refusals and the server errors included
**Given**: P-MOMO or P-ALL, with MSK and MAK known; RESET; an order O1 and the stand-in's record.
**When**: These are fetched and their bodies and headers searched: the site's home page; every script and stylesheet the home page references; `/api/v1/payment-options`; the placing answer of PLACE-MOMO; "Pay now" answers 200, 404, 409, 503; MNOTICE answers 204, and 400 for a bad signature, an unknown reference and a wrong amount; the 5xx answer of TC-66 (renamed table); the answers of TRANSFER and of `PUT /api/v1/orders/<id>`.
**Then**: None of the bodies, headers or files contains MSK or MAK; none of the files contains a variable whose name starts `VITE_` and holds a MoMo or VietQR setting.
Traces: SC-36 (Security NFR; PAY-021 data row)
Check: interact_page

**Scenario**: TC-92 — the logs hold no MoMo secret; no secret is held in a `VITE_` variable; `MOMO_API_URL` is a setting, not code
**Given**: The PR's backend and frontend files and its tests.
**When**: The logging code on the notice, "Pay now" and MoMo's call is read with `get_pr`, the code that reads the settings, `code/frontend/.env*` and every `import.meta.env.VITE_*`; `run_tests` (backend) is run.
**Then**: No log call prints MSK, MAK, a signature, a request body or the settings; no `VITE_` variable names a MoMo or VietQR setting; the base of every call to MoMo is read from `MOMO_API_URL` (its default is MoMo's sandbox, and no address is written in the provider's code besides that default); the backend suite passes, including its test that the log holds no secret after a payment, after a bad notice and after a failed call to MoMo.
Traces: SC-36 (PAY-021 item 6; Security NFR)
Check: manual — no instrument reads the server's log or a variable's name in source; the evidence is the PR's code read with `get_pr` and the backend tests run with `run_tests`. That `MOMO_API_URL` moves the base is also observed in TC-20.

**Scenario**: TC-93 — after the migration the MoMo record's note has dropped "marked paid at once", the bank record is unchanged, and all four records remain
**Given**: The running story, after its migrations.
**When**: `GET /api/v1/payments`, `GET /api/v1/payments/momo`, `GET /api/v1/payments/bank` and `GET /api/v1/payment-options` are sent; `SELECT id, name, note, online, icon FROM payment_methods ORDER BY seq` is read.
**Then**: `GET /api/v1/payments/momo` is the record `momo` with `name` `MoMo wallet` and `note` exactly `Confirm the payment in the MoMo app.` (it holds no `marked paid at once`); `GET /api/v1/payments/bank` has `note` exactly `Scan the QR code on the next screen with any banking app. We send the pair as soon as the money lands.`; the collection holds four records, `cod`, `bank`, `vnpay`, `momo`, each with `name`, `note`, `online` and `icon`; the table holds four rows; where the options list `momo` its note is the new one.
Traces: SC-37 (PAY-021 data row; SITE-016 item 5)
Check: fetch_url

**Scenario**: TC-94 — the migration changes the note only while it still has its first text, and running it twice changes nothing more
**Given**: The PR's migration pair for `payment_methods` and its migration test.
**When**: The migration's statement is read with `get_pr` and `run_tests` (backend) is run.
**Then**: The `UPDATE` is conditioned on the note still equalling `Confirm the payment in the MoMo app; the order is marked paid at once.`; it touches no other column and no other row; a backend test applies it twice (and once over a note the owner already changed) and finds the second application and the changed note untouched; the migration is a new file with a new version (no merged file edited).
Traces: SC-37 (idempotence; "never an edit to a merged file")
Check: manual — a migration's own text and its re-run cannot be observed over HTTP, in a browser or by styles; the evidence is the PR's files read with `get_pr` and the migration test run with `run_tests`.

**Scenario**: TC-95 — in 100 of 100 attempts a bad MoMo notice changes no order
**Given**: P-MOMO or P-ALL; RESET; one unpaid MoMo order O1 (R1); SNAPSHOT(O1) and the count of `payment_notices`.
**When**: 100 notices are sent in turn, cycling through: a signature altered by one character, a wrong signature (`MSK + "x"`), an unknown reference, a wrong amount, a wrong partner code (each correctly signed otherwise).
**Then**: All 100 answer 400; SNAPSHOT(O1) is equal after the hundredth; the count of `payment_notices` is unchanged.
Traces: SC-38 (NFR Integrity)
Check: interact_page

**Scenario**: TC-96 — in 100 of 100 runs two identical notices at the same moment mark the order paid once
**Given**: P-MOMO or P-ALL; RESET; 100 fresh unpaid MoMo orders (a shelf large enough: SHELF `38` set to 120).
**When**: For each order two identical correctly signed success notices are issued in one batch.
**Then**: In every run both answer 204; in every run exactly one successful notice row exists and the order reads `paid` true and `status` `confirmed`; the count of orders with a successful notice is 100 and the count of successful notice rows is 100.
Traces: SC-38 (NFR Integrity; PAY-002 AC-4)
Check: interact_page

**Scenario**: TC-97 — over 100 sequential signed notices the address answers within 2 seconds at p95
**Given**: P-MOMO or P-ALL; RESET; 100 fresh unpaid MoMo orders, each with its reference.
**When**: One correctly signed success notice is sent for each, in turn; each request's round-trip time is measured in the page.
**Then**: All 100 answer 204; the 95th-percentile time is at most 2000 ms.
Traces: SC-38 (NFR Performance)
Check: interact_page

**Scenario**: TC-98 — the change set edits the pages only where they named VNPay alone or decided "Pay now", and adds the QR block, the client file and the backend files
**Given**: The PR's file list and diff (`get_pr`), for the frontend half and the backend half.
**When**: The changed files are read.
**Then**: `Checkout.tsx`, `src/lib/orders.ts`, `Product.tsx`, the order page's parts (`OrderDetails.tsx`, `src/lib/order-page-and-pay-now.ts`), `PaymentReturn.tsx` and its parts and the dealer's order page change only on lines that named VNPay alone or decided "Pay now"; no page holds a list of ways to pay in code (no string `"vnpay"`, `"bank"` or `"momo"` as a way listed or switched on to draw a row, except where a provider's own words are chosen from the order's `payment`); the change set adds the QR block component under `src/components/`, `src/lib/more-ways-to-pay-vietqr-transfer-and-momo.ts`, and backend files (the two providers, the transfer-details call, the MoMo notice, both time rules, the migrations); it edits none of `src/theme.css`, `src/index.css`, `src/App.tsx`, `src/lib/mock/orders.ts` and, unless TL's own tool added a QR library, `package.json`; it adds no file under `src/lib/mock/`. The observable half is TC-07 and TC-14, where an invented way draws a row.
Traces: SC-39 (PAY-022 AC-3 as amended; story decision 1)
Check: manual — a change set is a property of the PR's files, which no HTTP, browser or style instrument reads; the evidence is `get_pr`. The page half (no list in code) is observed by TC-07 and TC-14 with an invented way.

---

## I. Contract cases for the endpoints this story touches that already exist

Only endpoints the contract already has are written here (`POST /v1/orders`, `POST /v1/orders/{id}/payment`, `PUT /v1/orders/{id}`); this story's own endpoints are designed after these cases, so no case invents their error shapes beyond what the story fixes.

**Scenario**: TC-99 — placing with a member the contract does not define is 400 for every way to pay, and creates nothing
**Given**: Any profile; RESET; the counter C0 and the shelf N for `samba-og-w-core-black` size `38` read.
**When**: Three placing requests are sent: PLACE-BANK with the extra member `"account": "0123456789"`; PLACE-MOMO with `"momo_phone": "0912345678"`; PLACE with `"qr": true`.
**Then**: Each answers 400 `{"error":"bad_request","message":<non-empty>,"errors":{<the member sent>:<non-empty sentence>}}` (not 422, not 201, whatever the settings: the decode step comes before the field and payment checks); no order exists; the counter still reads C0; the shelf is still N.
Traces: contract (POST /v1/orders: "No other undefined field is accepted"), SC-2
Check: fetch_url

**Scenario**: TC-100 — "Pay now" with a body is 400 whatever the order's way to pay, and stores nothing
**Given**: Any profile with VNPay's keys; RESET; an unpaid VNPay order OV, an FX-MOMO order OM and an FX-BANK order OB (arranged forms allowed), each with its token; SNAPSHOT of each.
**When**: `POST /api/v1/orders/<id>/payment` with the order's own token, `Content-Type: application/json` and the body `{"amount":1000}` is sent for each.
**Then**: Each answers 400 `{"error":"bad_request","message":<non-empty>}`; no attempt row is added for any; every SNAPSHOT is equal afterwards.
Traces: contract (POST /v1/orders/{id}/payment `bad_request`), SC-7, SC-9, SC-10
Check: fetch_url

**Scenario**: TC-101 — `PUT` of `paid` with an undefined member or the wrong type is refused and applies nothing
**Given**: RESET; an unpaid FX-BANK order OB at `placed`; ADMIN; SNAPSHOT(OB).
**When**: ADMIN sends `PUT /api/v1/orders/<OB>` with (a) `{"paid":true,"colour":"red"}`; (b) `{"paid":"true"}`; (c) `{"paid":1}`; (d) `{"paid":null}`; (e) the text `not json` with `Content-Type: application/json`.
**Then**: (a) answers 400 `{"error":"bad_request",…,"errors":{"colour":<non-empty>}}` and the order is still unpaid (the valid `paid` is not applied); (b), (c), (d) answer 400 or 422 with an `errors` member naming `paid` and never 200; (e) answers 400 `bad_request`; SNAPSHOT(OB) is equal after each.
Traces: contract (PUT /v1/orders/{id}: undefined field 400, §2.5), SC-19
Check: fetch_url

**Scenario**: TC-102 — with the database refusing connections placing, marking paid and the transfer details answer an error envelope and change nothing
**Given**: P-BANK or P-ALL; RESET; an FX-BANK order OB (T1) and ADMIN's cookie; the verifier makes the database unavailable with `set_db_available` while the backend keeps running.
**When**: PLACE-BANK, `PUT /api/v1/orders/<OB>` `{"paid":true}` as ADMIN and TRANSFER(OB) with T1 are sent; the database is made available again; OB, the counter and the shelf are read.
**Then**: Each of the three answers a JSON body `{"error":<code>,"message":<non-empty>}` with status 500 or 503 (never 2xx and never 401, 404 or 422 for a request that is valid once the database is back) and no SQL, host, password or stack trace; afterwards no order was created by the placing (counter and shelf unchanged) and OB is unpaid at `placed`.
Traces: contract (§2.3 `internal` 500; `unavailable` 503), SC-13, SC-17
Check: fetch_url

---

## Coverage — every acceptance criterion and the cases that serve it

| SC | Cases | SC | Cases |
|---|---|---|---|
| SC-1 | TC-01, TC-02, TC-03 | SC-21 | TC-53, TC-54 |
| SC-2 | TC-04, TC-05, TC-06, TC-07, TC-08, TC-09, TC-10, TC-99 | SC-22 | TC-55, TC-56, TC-57 |
| SC-3 | TC-11, TC-12, TC-13 | SC-23 | TC-58, TC-59 |
| SC-4 | TC-14, TC-15 | SC-24 | TC-60, TC-61 |
| SC-5 | TC-11, TC-16, TC-17 | SC-25 | TC-62, TC-63 |
| SC-6 | TC-18, TC-19 | SC-26 | TC-64 |
| SC-7 | TC-20, TC-21, TC-22, TC-23, TC-59, TC-100 | SC-27 | TC-65 |
| SC-8 | TC-24, TC-25 | SC-28 | TC-29, TC-66, TC-67, TC-68 |
| SC-9 | TC-26, TC-27, TC-28, TC-29, TC-100 | SC-29 | TC-69 |
| SC-10 | TC-30, TC-31, TC-100 | SC-30 | TC-70, TC-71 |
| SC-11 | TC-32 | SC-31 | TC-72, TC-73 |
| SC-12 | TC-33, TC-34, TC-35 | SC-32 | TC-74, TC-75, TC-76 |
| SC-13 | TC-36, TC-37, TC-38, TC-39, TC-40, TC-102 | SC-33 | TC-77, TC-78, TC-79, TC-80, TC-81, TC-82, TC-83 |
| SC-14 | TC-41 | SC-34 | TC-84, TC-85, TC-86 |
| SC-15 | TC-42, TC-43 | SC-35 | TC-87, TC-88 |
| SC-16 | TC-44, TC-45, TC-46 | SC-36 | TC-20, TC-89, TC-90, TC-91, TC-92 |
| SC-17 | TC-47, TC-48, TC-102 | SC-37 | TC-93, TC-94 |
| SC-18 | TC-49, TC-50 | SC-38 | TC-95, TC-96, TC-97 |
| SC-19 | TC-51, TC-101 | SC-39 | TC-07, TC-14, TC-98 |
| SC-20 | TC-52 | | |

Every SC-1 … SC-39 has at least one case (102 cases). The criteria that name a failure have a case where the failure happens: SC-8 (MoMo gives no address: TC-24; "Pay now" fails too: TC-25), SC-13 (a stranger, a paid or non-bank order, the account removed, the database refused: TC-37, TC-38, TC-39, TC-102), SC-15 (the details cannot be read: TC-43), SC-19 and SC-20 (the refusals and the denied roles: TC-51, TC-52, TC-101), SC-22 (a refusal, a network failure, an ended session: TC-56), SC-25 (every refused notice: TC-62, TC-63), SC-26 (a failed payment: TC-64), SC-28 (the store fails, the database refuses, the keys are unset: TC-66, TC-67, TC-68), SC-34 (no match, the shop cannot be reached: TC-84, TC-85), SC-3 (VNPay's keys missing: TC-11, TC-12, TC-13).

The SRS criteria are all reached through the story's own table (PAY-020 AC-1 … AC-9 → SC-1, SC-2, SC-11, SC-12, SC-14, SC-10, SC-31, SC-13; PAY-021 AC-1 … AC-8 → SC-1, SC-2, SC-7, SC-23, SC-24, SC-25, SC-30, SC-33; PAY-022 AC-1 … AC-3 → SC-1, SC-2, SC-39; PAY-003 AC-11 → SC-31).

## What the cases need, and what they leave to a person

- **`Check: manual`** — five cases, each naming the evidence the verifier reads instead: TC-76 (two checks at the same moment: the timer is inside the process), TC-89 (`.env.example`: a repository file), TC-92 (the server's log and a variable's name in source), TC-94 (the migration's own text and its re-run), TC-98 (the change set). That is 5 of 102.
- **Cases that run in any profile** (nothing beyond VNPay's keys, or only intercepted answers): TC-06, TC-07, TC-09, TC-12 and TC-13 (their stubbed halves), TC-14, TC-15, TC-28, TC-29, TC-30, TC-39, TC-47 … TC-56 and the first half of TC-57 (the dealer's side uses arranged bank orders), TC-68, TC-70 … TC-75, TC-90, TC-93, TC-99, TC-100, TC-101.
- **Need the bank account (P-BANK or P-ALL)**: TC-01, TC-02, TC-03 (or STUB-OPTIONS), TC-04, TC-16, TC-17, TC-32 … TC-38, TC-40 … TC-46 (TC-44 with its long values stubbed), TC-87, TC-102.
- **Need MoMo's keys and a stand-in for MoMo's create-payment API (P-MOMO or P-ALL)**: TC-05, TC-18 … TC-27, TC-77 … TC-82, TC-84 … TC-86, TC-88, and — with MSK — TC-58 … TC-67, TC-69, TC-91, TC-95 … TC-97. TC-20 … TC-23 additionally read the stand-in's record; without the stand-in they are UNVERIFIED (instrument lacked: a stand-in for MoMo's create-payment API that records requests).
- **Need a profile other than the running one**: TC-08 (P-PARTIAL), TC-10 (the way not listed), TC-11 and TC-12's unstubbed half (P-NOVNPAY), TC-13's unstubbed half (P-NONE), TC-39 and TC-68 (a profile without the bank account / MoMo keys; they run in P-VNPAY), TC-90 (P-VNPAY or P-NONE).
- **Real waits**: TC-54 (up to 90 s plus 70 s), TC-69 (60 s), TC-70 … TC-75 (up to 90 s each, plus 70 s in TC-70, TC-72, TC-75), TC-78 (10 s), TC-79 (60 s), TC-80 (31 s per half), TC-83 (31 s), TC-86 (31 s per width). A verifier whose instrument cannot wait that long reports those UNVERIFIED with that instrument named.
- **Interception** (STUB, DELAY, ABORT): TC-07, TC-12, TC-13, TC-14, TC-25, TC-27, TC-42, TC-43, TC-44, TC-55, TC-56, TC-81, TC-85. A verifier that cannot intercept reports each UNVERIFIED (instrument lacked: request interception).
- **A story verified only in P-VNPAY** would leave roughly half of these cases UNVERIFIED for the missing settings and wait for the stakeholder; the settings are already requested in the console with the plan (story §1), so the run that decides the story is P-ALL with a stand-in. Where the owner has not supplied real values, any well-formed test values do (a six-digit BIN, a ten-digit account number, an upper-case holder name; MoMo's public sandbox partner code, access key and secret key): none of the cases needs real money to move.
