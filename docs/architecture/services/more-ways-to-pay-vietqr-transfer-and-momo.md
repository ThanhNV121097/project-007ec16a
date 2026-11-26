# Service contracts — More ways to pay: VietQR transfer and MoMo (plan item 12, "S11")

Module `payments` (and the changes it makes in `orders`, `admin`, `config` and `server`) · Requirements PAY-020, PAY-021, PAY-022 and the changes they make to ORD-006 / ORD-007 item 8, ADMIN-016 item 4 (AC-11, AC-12), SITE-006 item 3 · Story `docs/payments/stories/more-ways-to-pay-vietqr-transfer-and-momo.md` · UI PR #91 · Tables: `docs/architecture/erd/more-ways-to-pay-vietqr-transfer-and-momo.md` (**no new table, column or index**).

This file defines **the two new routes (the bank-transfer details and MoMo's notice), what changes in the five routes that already exist, the two providers behind the payment seam, the VietQR payload and the QR picture, MoMo's create-payment request, the 24-hour rule beside the 30-minute one, the settings, and the frontend's swap from its mock** — and nothing else. It extends `services/_base.md` §3.5, §3.10, §3.11, §3.13, §3.15, §3.16, `services/online-payment-with-vnpay.md` (the seam, `payments`, the expiry, the notice's shape), `services/order-page-and-pay-now.md` (`authorize`, the 404 body) and `services/dealer-handles-orders-status-courier-payment.md` (§3.3, §3.4). It reuses the error envelope, the closed error catalog, the no-body rule and the `Idempotency-Key` table of `_base.md`, and adds **no error code**: `payment_unavailable` is widened from "VNPay's keys" to "the settings the order's way to pay needs" (catalog row §2.3, meaning only). Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/orders/doi-24818/transfer` reaches Go as `/v1/orders/doi-24818/transfer`, and MoMo's `/api/v1/payments/momo/ipn` as `/v1/payments/momo/ipn`.

## 1. Endpoints

| # | Method and path | Auth | What this story does |
|---|---|---|---|
| 3.5 | `GET /v1/payment-options` | none | The list is `vnpay`, then `bank` when its settings are valid, then `momo` when its settings are valid, **in that order** (§3.1) |
| 3.10 | `POST /v1/orders` | none | A way not listed is `422`; `bank` answers `payment_url: ""` and writes no attempt; `momo` answers MoMo's address (§3.2) |
| 3.13 | `POST /v1/orders/{id}/payment` | the order's `X-Order-Token` | One refusal added (a bank-transfer order, `409`); a MoMo order gets a MoMo address (§3.3) |
| **3.21** | `GET /v1/orders/{id}/transfer` | the order's `X-Order-Token`, or its linked customer | **New** (§3.4): the shop's account, the amount, the note and the QR of one unpaid bank-transfer order |
| **3.22** | `POST /v1/payments/momo/ipn` | MoMo's signature | **New** (§3.5): with VNPay's notice, the only thing that marks an online order paid |
| 3.15 | `PUT /v1/orders/{id}` | dealer | `{"paid": true}` is accepted for a bank-transfer order and refused for every other (§3.6) |
| 3.16 | `DELETE /v1/orders/{id}` | dealer | The sentence for a paid, cancelled order follows the order's way to pay (§3.6) |
| 3.11 | `GET /v1/orders/{id}` | token, dealer, linked customer | **No change.** The public view already carries `payment`, `paid`, `status`, `total`, `number` |
| — | The time rules | none (a goroutine in the API process) | MoMo joins the 30-minute rule; bank transfer gets a 24-hour rule in the same loop (§5) |

(`3.21` and `3.22` are the next free numbers of this file; if another story took them, take the next free ones — nothing refers to the numbers but this file's headings.)

**Mounting.** `orders.Register` gains `GET /v1/orders/{id}/transfer` (a literal pattern beside `GET /v1/orders/{id}` and `POST /v1/orders/{id}/payment`); `payments.Register` gains `POST /v1/payments/momo/ipn` and is **always mounted**, whatever the settings — with MoMo's keys missing it answers `400` to everything, so the owner's registration of the address never meets a `404`. Any other method on either path falls to the catch-all and answers `404 not_found` in the project envelope. The notice path has **three** segments after `/v1`, so it can never be shadowed by the `payments` collection's `GET /v1/payments/{id}` (two); a route-table test pins both (`GET /v1/payments/momo` is still the record of the way to pay, `GET /v1/payments/momo/ipn` is a `404`, `POST` to it is the notice).

## 2. The mock, read and judged

The UI PR (#91) adds `src/lib/mock/more-ways-to-pay-vietqr-transfer-and-momo.ts`, imported only by `src/lib/more-ways-to-pay-vietqr-transfer-and-momo.ts`. It is unreviewed input: its shape is compared with the contract, and **where `_base.md`, S5–S7 or item 17 already decided a shape, they win.**

| Mock | Contract | Judgment | Frontend change at the swap |
|---|---|---|---|
| `mockOptions()` answers `PaymentOption` items (`id`, `name`, `note`, `icon`, `online`, `is_available`); `no-vnpay` lists VNPay closed, then bank, then MoMo | §3.1 answers exactly those six members, in the order `vnpay`, `bank`, `momo` | **Sound**, field for field. The mock's notes are the records' (`bank`) and the migration's new `momo` note ("Confirm the payment in the MoMo app.") | `readPaymentOptions` loses its first two lines (§7) |
| `getTransfer(number, total)` builds the answer from the **page's** `number` and `total` | §3.4 reads the amount and the note from the stored order; the browser sends none (SC-12) | **Not a contract the real call can keep**: the call takes the order's id and token, not two values from the page. The page's `PublicOrder` already holds the same values, so nothing it draws changes | `readTransfer(order)` becomes one `api.get` by id with the token (§7); the two arguments go |
| Members `holder`, `bank`, `account_number`, `amount`, `note`, `payload` | §3.4 keeps all six **with the same names, types and meaning** (`amount` is an integer in VND; `note` is the order's number) | **Sound.** The names are the mock's by decision (S3) | None |
| The QR picture is `standInModules(payload)`, a pattern seeded from the text — not a code a banking app can read | §3.4 adds `qr`: the real modules, drawn by the backend from the same payload (S2) | **The stand-in must not ship.** The decision the story left to this design is made: the backend draws the modules | `TransferDetails` gains `qr: string[]`; `TransferQr` takes `rows` instead of `payload`; `qrModules` and `standInModules` are deleted (§7) |
| `503` is "gone" (the account was removed); every other failure, a `404` included, is "could not load" | §3.4: `503 payment_unavailable` when the account is not set; **`409` when the order is paid, cancelled or no longer a bank-transfer order**; one `404` body | **Sound, plus one case the mock never drew.** A customer who pays in a second tab and then refreshes the first gets `409` for a block the page still draws; "We could not load the bank transfer details" would be a false alarm for an order that is now fine | `readTransfer` maps `409` like `503`: `gone` (nothing drawn; the next read of the order shows the truth) |
| `putPaid(id, status)` answers `{id, paid, status}`; `422` carries `errors.paid`; `401` is "expired" | §3.6 answers the **whole dealer view** (item 17 §3.3 row 13: `paid`, `status`, `transaction_no`, `items`, …); `422` names `errors.paid`; `401`/`403` | **Sound for what the page reads**: `markPaid` reads only `status`. The sentence of a refusal is never shown (the page says "We could not save") | None |
| Every answer is `{status, body}`; a failure to connect throws `TypeError` | `src/lib/api.ts` resolves a 2xx with the parsed body and **throws `ApiError`** (`status`, `code`, `message`, `errors`; a failure to connect is status 0, code `network`) | **The mock's shape is not the real client's**, hidden inside one file | The three `transport` functions become `api.get`, `api.get`, `api.put`; `ApiError` maps to the results `readTransfer` and `markPaid` already return |
| `viaDemo(order)` inside the real `readOrder` of `lib/order-page-and-pay-now.ts`: any visitor can add `?mock=bank` to their own order link and see a fake account | — | **A defect to delete, not a shape** (TL's UI review, should-fix 3) | Delete the `viaDemo` import and call; delete the mock module and the `?mock=` reading |
| `?mock=slow`, `error`, `gone`, `long`, `offline`, `refused`, `expired` | **No delay, fault or sleep route, header or query parameter is added to the backend** | Mock-only controls | Go with the mock; the tests' `?mock=` cases become `vi.mock("../lib/api")` stubs that throw the **real** `ApiError` (`new ApiError(0, "network", …)` for offline, `(503, "payment_unavailable", …)` for gone, `(401, "unauthenticated", …)` for expired, `(422, "invalid", …, {paid: …})` for a refusal, a promise held for two seconds for slow) |
| `readReturn(params)` reads VNPay's `vnp_TxnRef`/`vnp_ResponseCode` and MoMo's `orderId`/`resultCode` (`"0"` becomes `"00"`) | MoMo sends the customer to `redirectUrl` with `orderId` (our reference, `DOI-24818-1`), `resultCode` and the other notice fields as query parameters | **Sound.** The page never trusts these for a result; it asks the shop for the order's state (PAY-004 item 1). The reference pattern `^(DOI-\d{1,15})-\d{1,9}$` matches `DOI-24818-1` | None |
| `OrderState.payment` (optional) read from `GET /v1/orders/{id}` | The public view carries `'payment', o.payment_method_id` | **Sound** | None |
| The `bank` and `momo` ids named in `lib/more-ways-to-pay-…ts` (`BANK`, `MOMO`) | PAY-022 item 2: no page holds a **list** of ways to pay | **Accepted** (TL's UI review): the two ids behave differently from VNPay's (no payment page; its own words on the result page); the list itself always comes from the shop | None |
| New content keys `pages.order.transfer.*`, `pages.payment_return.momo.timed_out.*` listed in the PR body, with fallbacks in `lib/payment-words.ts` | `content.json` is frozen for stories | **Added by this design's PR** (new keys only, additive; the same one-time lift the orders and VNPay stories received) | At the backend step: put `data-edit={key}` on each word (or draw it with `<T k=…>`) and delete the fallbacks (§7) |

## 3. Contracts

### 3.1 `GET /v1/payment-options` (project §3.5) — what changes

**Auth** — none; any parameter is ignored. **Success** `200` `{"items":[…],"total":3,"page":1,"limit":100}`; each item as before (`id`, `name`, `note`, `icon`, `online`, `is_available`).

| Rule | |
|---|---|
| **What is listed** | `vnpay` **always**. `bank` only when all four `VIETQR_*` settings are set and valid (§4.1). `momo` only when `MOMO_PARTNER_CODE`, `MOMO_ACCESS_KEY` and `MOMO_SECRET_KEY` are set and `MOMO_API_URL` is valid. A way with one value blank, or malformed, is **not set**. `cod` is never listed. The settings are read once at start-up, so the list is fixed for the life of the process |
| **Order** | **The code's order, `vnpay`, `bank`, `momo`** — not the `payment_methods` rows' `seq` order (`cod`, `bank`, `vnpay`, `momo`), which would put bank transfer first (PAY-022 AC-1: "VNPay first, then bank transfer, then MoMo"). The statement is `SELECT id, name, note, icon, online FROM payment_methods WHERE id = ANY($1) ORDER BY array_position($1::text[], id)` with `$1` the listed ids. **This departs from `_base.md` §3.5 ("rows come in the `payment_methods` order")**; no row is reordered (S6) |
| **`is_available`** | `provider.Available()`. It is `false` only for `vnpay` with a key missing; a listed `bank` or `momo` is available by construction (it is listed only because its settings are valid) |
| **A listed way with no row** | Cannot happen: the four rows are referenced by sample orders (FK `RESTRICT`) and cannot be deleted |

Example with the three settings present:

```json
{ "items": [
  { "id": "vnpay", "name": "VNPay — ATM, Visa, Mastercard", "note": "Pay by card or banking app on VNPay's secure page.", "icon": "card",   "online": true, "is_available": true },
  { "id": "bank",  "name": "Bank transfer (VietQR)",        "note": "Scan the QR code on the next screen with any banking app. We send the pair as soon as the money lands.", "icon": "qr", "online": true, "is_available": true },
  { "id": "momo",  "name": "MoMo wallet",                   "note": "Confirm the payment in the MoMo app.", "icon": "wallet", "online": true, "is_available": true } ],
  "total": 3, "page": 1, "limit": 100 }
```

Errors: `internal` 500 (the statement fails). Headers: `Cache-Control: no-store`.

### 3.2 `POST /v1/orders` (project §3.10) — what changes

Order of checks is unchanged: body (415, 413, 400) → `Idempotency-Key` (400) → every field at once (422) → the 503 gate → the transaction (409, 201, 200).

| Change | Rule |
|---|---|
| **`payment` must be listed** | `payment` is validated against the **listed ids** of §3.1 (not a constant): `cod`, an unknown id, `momo` without its settings and `bank` without its account are `422` naming `payment`, "That way to pay is not offered.", with no order made and no pair taken (SC-2) |
| **The 503 gate** | After the field checks, before `place`: if the provider registered for `req.Payment` is not `Available()`, `503 payment_unavailable`. Only `vnpay` can reach it (a way that is unavailable is not listed, so `bank` and `momo` stop at the `422`). Placing with `bank` while VNPay's keys are missing **succeeds** (SC-3) |
| **`bank`** | `Place` is unchanged. After the commit the bank provider's `PaymentURL` returns `""` and **writes nothing**: `201` with `payment_url: ""`, the order `placed`, unpaid, its pairs taken, its token (SC-5). The browser lands on `/orders/<id>?placed=1`, which draws the QR block |
| **`momo`** | After the commit, outside the transaction, the MoMo provider's `PaymentURL` is asked with the route's 3-second limit **for the whole of it** — the attempt's transaction and MoMo's answer (§4.2). `payment_url` is MoMo's `payUrl`. **A refusal, a timeout, an unreachable MoMo or an answer with no address leaves `payment_url` `""`** with a log line; the order stays saved, unpaid, `placed`, its pairs taken, and the order page offers "Pay now" (SC-8) |
| **A replay of a used key** | `bank` answers `payment_url: ""` again; `momo` builds a **new** attempt when the order is still payable, as VNPay does (`_base.md` §2.6) |

### 3.3 `POST /v1/orders/{id}/payment` (project §3.13) — what changes

**Auth, request, headers** — as `online-payment-with-vnpay.md` §3.1: the order's `X-Order-Token` (the linked customer's session joins through `authorize`), no body, `Cache-Control: no-store`.

**Order of checks** — access `404`, a body `400`, then the state, availability and the address. **The state rules** are `CheckPayable(state, now)`, still the one function, now in this order: cancelled → paid → **paid by bank transfer** → a way that is not paid on a provider's page → the time to pay has run out.

| Code | HTTP | Trigger | Body message |
|---|---|---|---|
| `not_found` | 404 | Access, byte for byte S6 §3.3 | "Not found." |
| `bad_request` | 400 | A body was sent | "This call takes no body." |
| `conflict` | 409 | Cancelled (paid or not) | "This order was cancelled, so it cannot be paid." |
| `conflict` | 409 | Already paid (and not cancelled) | "This order is already paid." |
| `conflict` | 409 | Unpaid, not cancelled, **its way to pay is `bank`**. **New** (SC-10) | "This order is paid by bank transfer. Scan the QR code on its page." |
| `conflict` | 409 | Unpaid, not cancelled, a way that is not `vnpay` or `momo` (`cod`: the sample orders) | "This order cannot be paid online." |
| `conflict` | 409 | Unpaid, not cancelled, 30 minutes run out (`vnpay`, `momo`) | "The time to pay this order has run out." |
| `conflict` | 409 | 50 attempts already | "This order has had too many payment attempts. Please place a new order." |
| `payment_unavailable` | 503 | The provider's settings are not set (VNPay's keys, or MoMo's) | "Online payment is not available right now. Please try again later." |
| `internal` | 500 | The access statement or the attempt transaction fails; **MoMo refused, did not answer within the limit, or gave no usable address** (§4.2); VNPay's address cannot be built. Generic body; the cause is logged beside the order id and the request id, **never the token, a secret or an address** | "Something went wrong. Please try again." |

**Success** `200` `{"payment_url": "<address>"}`: for `vnpay`, the signed address of the VNPay story; for `momo`, MoMo's `payUrl`. **Each success inserts exactly one `payment_attempts` row and changes nothing else; every refusal before the attempt inserts none.** A MoMo attempt whose call to MoMo then fails stays (E3 of the ERD) and answers `500`. `bank` never reaches the provider: `CheckPayable` refuses it first, so "Pay now" for a bank order **stores nothing** (SC-10).

The state is judged again under the order's row lock inside the MoMo provider (`orders.LockOrder` + `CheckPayable`, as VNPay's), so a notice or a cancel that landed between the route's read and the lock gives the same `409` sentence the route would have given.

### 3.4 `GET /v1/orders/{id}/transfer` (new, 3.21)

**Purpose** — what the order page needs to show "Pay by bank transfer": the shop's account, the amount, the note to type and the QR code. **Traces to** — PAY-020 items 2 and 4, AC-3, AC-4, AC-9; story decisions 6 and 7. **Auth** — the order's `X-Order-Token`, or the signed-in customer the order is linked to (the same `authorize(…, modePay)` as "Pay now", so item 15's `AccountID` rule applies to it with no line of its own). **A dealer session without the token is a `404`**, as for "Pay now": the dealer's way into an order is `GET /v1/orders/{id}`, which carries no account. **Request** — no body is read and every query parameter is ignored: nothing the browser sends can change the amount, the note or the account. **Headers on every answer** — `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`.

**Order of checks** — the first that fails answers; nothing is written by this route at any step.

| # | Check | Refusal |
|---|---|---|
| 1 | `{id}` matches `^[a-z0-9][a-z0-9._-]*$` (at most 100) and the token matches `^[A-Za-z0-9_-]{1,128}$` | `404 not_found`, no query made |
| 2 | The access statement finds the order by its id **and** the hash of the token | `404 not_found`: the one body of S6 §3.3 for no token, another order's token, a dealer session, an unknown id |
| 3 | **State**, `CheckTransfer(state)`, in this order: cancelled → paid → the order's way to pay is not `bank` | `409 conflict`, one of three different sentences (below) |
| 4 | The bank provider is `Available()` (all four settings valid) | `503 payment_unavailable` |
| 5 | The payload and the QR are built from the settings and the **stored** `total` and `number` | `200`; an encoder failure is `500` |

**Success** `200`:

```json
{
  "holder": "NGUYEN VAN AN",
  "bank": "Vietcombank",
  "account_number": "0071001234567",
  "amount": 3140000,
  "note": "DOI-24818",
  "payload": "00020101021238570010A00000072701270006970436011300710012345670208QRIBFTTA5303704540731400005802VN62130809DOI-248186304XXXX",
  "qr": ["111111100010…", "…"]
}
```

| Field | Type | Source |
|---|---|---|
| `holder` | string | `VIETQR_ACCOUNT_NAME`, trimmed |
| `bank` | string | `VIETQR_BANK_NAME`, trimmed (the name a person reads; the BIN is only in the payload) |
| `account_number` | string | `VIETQR_ACCOUNT_NUMBER`, trimmed |
| `amount` | integer | The order's stored `total`, VND, from the access statement's row; never from the request |
| `note` | string | The order's `number` (`DOI-24818`) |
| `payload` | string | The VietQR text of §4.3: the BIN, the account, `amount` and `note`, with its checksum |
| `qr` | array of strings | The QR code of `payload`, one string per row of modules, `"1"` dark and `"0"` light, **square, without the quiet zone** (the page draws its own 4-module margin). 21 to ~57 rows |

The answer carries **no** phone, email, address or note of the order, no id, no status, and it is the **only** call of the shop that carries the account: not `GET /v1/payment-options`, not `GET /v1/orders/{id}`, not any collection (`GET /v1/payments` serves only the four records' name, note, online flag and icon).

| Code | HTTP | Trigger | Body message |
|---|---|---|---|
| `not_found` | 404 | Checks 1–2 | "Not found." |
| `conflict` | 409 | Cancelled | "This order was cancelled, so it cannot be paid." |
| `conflict` | 409 | Already paid | "This order is already paid." |
| `conflict` | 409 | Unpaid, not cancelled, **not** a bank-transfer order. **New** | "This order is not paid by bank transfer." |
| `payment_unavailable` | 503 | The account is no longer set or valid (the owner removed a setting after the order was placed) | "Online payment is not available right now. Please try again later." |
| `internal` | 500 | The access statement fails or times out (the route's 5-second context); the QR cannot be encoded. Generic body; the cause is logged beside the order id and the request id, **never the account's values or the payload** | "Something went wrong. Please try again." |

The 409 bodies carry **no** `errors` key. The page branches on the status (`409` and `503` mean "draw nothing"; everything else "could not load") and never on a sentence. A paid or cancelled bank order has no QR, even for its own token (SC-13, SC-14): once the money has landed there is nothing to pay into.

### 3.5 `POST /v1/payments/momo/ipn` (new, 3.22)

**Purpose** — MoMo's server-to-server notice of a payment's result: with VNPay's, **the only thing that marks an online order paid**. **Traces to** — PAY-021 items 3–4 and AC-4 to AC-6, PAY-002, story decision 8. **Auth** — none: no session is read and no token is needed; the signature is the proof. The browser never calls it and `/payment/return` never marks anything. **Method** — `POST` only.

**Request** — `Content-Type: application/json` (not enforced: MoMo sets it, and a stranger gains nothing from the wrong one), a body of at most **16 KiB** read through `http.MaxBytesReader`, **one JSON object** with nothing after it, decoded with numbers kept as their text (`json.Number`) so the signed string is exactly what MoMo sent. A member MoMo adds that is not listed here is ignored and is not signed.

| Member | JSON type | Required | Used for |
|---|---|---|---|
| `partnerCode` | string | **yes** | Must equal `MOMO_PARTNER_CODE` |
| `orderId` | string | **yes** | The attempt's `reference` (`DOI-24818-1`) |
| `requestId` | string | **yes** | Signed only |
| `amount` | integer | **yes** | Whole VND; must equal the attempt's `amount` |
| `transId` | integer | **yes** | MoMo's transaction number → `transaction_no` |
| `resultCode` | integer | **yes** | `0` is success; anything else is a failure → `response_code` |
| `responseTime` | integer | **yes** | Epoch milliseconds → `paid_time` |
| `signature` | string | **yes** | 64 hexadecimal characters |
| `orderInfo`, `orderType`, `message`, `payType`, `extraData` | string | no (`""` when absent) | Signed as received; `payType` → `bank_code` |

An integer member is a JSON number whose text is `-?[0-9]{1,18}` — a fraction, an exponent, a string or `null` there is a refusal. A string member that is a number, an object or `null` is a refusal.

**Processing, in this order** (SC-25: signature, then reference, then amount, then "already processed"). The first rule that fails answers and changes nothing; every check before the first write is arithmetic or cryptography, so a stranger costs no database work.

| # | Check | Answer | Writes |
|---|---|---|---|
| 0 | MoMo's settings are valid; the body is at most 16 KiB, is one JSON object, and holds every required member with the right type | else `400` | nothing |
| 1 | `signature` equals the lower-case hex **HMAC-SHA256**, keyed with `MOMO_SECRET_KEY`, of the string of §4.2 (the thirteen fields sorted by name, with `accessKey` from `MOMO_ACCESS_KEY`), hex-decoded and compared with `hmac.Equal` (constant time; case does not matter); **and** `partnerCode` equals the shop's | else `400` | nothing |
| 2 | `orderId` is the `reference` of a `payment_attempts` row of provider **`momo`** (any earlier reference of an order qualifies; a VNPay reference does not) | else `400` | nothing |
| 3 | `amount` equals the attempt's `amount` | else `400` | nothing |
| 4 | The order is not already `paid` | else `204` (already processed) | nothing |
| 5 | **A failure notice** — `resultCode` is not `0` (the customer cancelled, the payment failed or was refused): insert a `payment_notices` row with `is_success = false`, `ON CONFLICT … DO NOTHING` | `204` | one row, or none for an identical twin; the order is untouched, unpaid, its status unchanged |
| 6 | **A success notice** — `resultCode` is `0`: one transaction, `orders.MarkPaid` (**0 rows → `204`, rolled back**), then the notice row (`ON CONFLICT (attempt_id) WHERE is_success DO NOTHING`; **0 rows → `204`, rolled back**), commit | `204` | `orders.paid = true`; `placed → confirmed`, every other status unchanged; one success row |
| — | A database error at any statement, a deadline, a lock timeout (a deadlock `40P01` is retried once, as VNPay's) | `500`, so MoMo sends the notice again | the transaction is rolled back: the order unpaid, no row kept |

| Status | When | Body |
|---|---|---|
| `204` | The notice was **accepted** (a success or a failure was recorded) **or already processed** (the order was paid; a twin won the race). Both are what MoMo needs to stop retrying | none |
| `400` | **Refused**: the settings are not set (SC-28: the server still starts), an unreadable or oversize body, a missing or mistyped member, a bad signature, a partner code that is not the shop's, an unknown reference, a wrong amount | The project envelope `{"error":"bad_request","message":"The notice was refused."}` with **no `errors` key**: a stranger is not told which check failed, and the body is never echoed |
| `500` | The database failed; nothing is half-written | `{"error":"internal","message":"Something went wrong. Please try again."}` |

Every answer carries `Cache-Control: no-store`. This is **not** an exception to the project's error shape (unlike VNPay's `{"RspCode","Message"}`): the refusal and the failure use the catalog's codes; only the `204` has no body, because MoMo asks for no body.

**What a success does to the order** — as VNPay's notice (`online-payment-with-vnpay.md` §3.4): `paid` is set and a `placed` order becomes `confirmed`; any other status is kept, **an order already `cancelled` is marked paid and stays `cancelled`** (SC-27: the dealer refunds it in MoMo's portal), and **no pair moves**. **What is kept** (ERD §2.1): `transId`, `payType`, `amount`, `resultCode`, `responseTime`. `extraData`, `orderInfo`, `message` and `signature` are not kept; **no wallet or card number is received.**

**Two identical notices at the same moment** — both pass checks 0–4, both run `MarkPaid`; the second waits on the order's row, finds `paid` and changes nothing: **both answer `204`**, one row is kept (SC-24; twenty at once: twenty `204`, one row). **Notes** — p95 ≤ 2 s over 100 sequential signed notices (SC-38); the handler runs under a 5-second deadline and `SET LOCAL lock_timeout = '3s'`. **The log line** holds the status answered, the order's number once the reference is known and the request id — never the body, the signature, a transaction number or a key. **No IP allow-list and no rate limit** (VNPay S12: the proxy chain hides MoMo's address and MoMo's retries would be refused by a limit; the signature, checked before any database work, is the guard).

### 3.6 `PUT /v1/orders/{id}` and `DELETE /v1/orders/{id}` (items 17's §3.15, §3.16) — what changes

Item 17's design (`dealer-handles-orders-status-courier-payment.md` §3.3) refuses `paid` by name for every order. This story makes **one** exception and changes **nothing else** of that route: its order of checks, its field table, its state rules and its sentences stand.

| Row of item 17 §3.3 | Change |
|---|---|
| 7 (the keys) | `paid` is still a known key; a value that is not JSON `true` falls to row 8 |
| 8 (the refused names) | `paid` **leaves** the list of eleven refused names **when its value is JSON `true`**. `paid: false`, `null`, a number, a string, an array or an object is still `422` `errors.paid` "Only the payment provider marks this order paid." (SC-19). The other eleven names are refused as before, whatever the value |
| 11 (the state rules, judged under the lock) | A new rule: `paid: true` is allowed **only when the order's way to pay is `bank`**; for any other way (`vnpay`, `momo`, `cod`: the sample orders hold some) it is collected as the same `422` `errors.paid` with the same sentence (SC-19). It holds for **every status**, `delivered` and `cancelled` included: `paid` is not one of the locked fields |
| 12 (any fault) | Unchanged: every fault of rows 8, 9 and 11 is collected, the transaction is rolled back and nothing in the body is applied — so `{"paid": true, "note": "x"}` on a VNPay order changes neither, and the same body on a bank order applies both |
| 13 (apply) | After the status move and the fields, `orders.MarkPaid(tx, id)` runs when the body said `paid: true` and the order is a bank-transfer order: `UPDATE orders SET paid = true, status = CASE WHEN status = 'placed' THEN 'confirmed' ELSE status END WHERE id = $1 AND NOT paid`. **Zero rows is not an error** (the order was already paid): the answer is `200` and nothing changed (SC-18). The dealer view is read in the same transaction |

**Result** — `{"paid": true}` alone on an unpaid bank order at `placed` answers `200` with the dealer view, `paid` `true`, `status` `"confirmed"`, nothing else changed; at `confirmed`, `packed`, `shipped` or `delivered` the status stays and `paid` becomes `true`; at `cancelled` it answers `200` with `paid` `true` and the status still `"cancelled"`, its pairs neither taken nor returned (SC-17, SC-18). `transaction_no` stays `""` (a bank order has no success notice). A body that moves the status **and** marks paid reads the status before the move (item 17 D6): `{"status":"confirmed","paid":true}` on a `placed` bank order is one `200`.

```
PUT /v1/orders/doi-24818   { "paid": true }                      (a bank-transfer order, placed, unpaid)
→ 200  { …the dealer view…, "paid": true, "status": "confirmed" }

PUT /v1/orders/doi-24819   { "paid": true }                      (a VNPay order)
→ 422  { "error": "invalid", "message": "Check the details of this order.",
         "errors": { "paid": "Only the payment provider marks this order paid." } }
```

**Auth and the other errors** — unchanged (`401`/`403` before the body, `404` for no such order, `415`, `413`, `400`, `500`): SC-20. **Concurrency** — the route already holds `orders.LockOrder`: two marks at once leave the order paid once and both answer `200` (SC-21); a mark and the 24-hour cancel serialise on the row (ERD §6). **Log** — one `info` line when the mark changed a row: the order's number and the dealer's user id, no setting or value of the account.

**`DELETE`** — item 17 §3.4 row 5's sentence for a paid, cancelled order names VNPay's portal, which is wrong for an order paid another way. It now follows the way to pay, **VNPay's sentence unchanged**: `vnpay` → "A paid order cannot be deleted; refund it in VNPay's portal first."; `momo` → "…refund it in MoMo's portal first."; `bank` → "…refund it by bank transfer first."; any other way → "A paid order cannot be deleted; refund it first." Status, code and body shape stay `409 conflict` with no `errors`.

## 4. The two providers

### 4.1 Settings

Read once by `config.Load`, from the environment only, trimmed; none is required to start, and **a way whose settings are missing or malformed is simply not listed** (§3.1). None is a default in the source except `MOMO_API_URL`'s sandbox. All are named in `code/backend/.env.example` with **no value** (SC-36).

| Key | Kind | Rule |
|---|---|---|
| `MOMO_PARTNER_CODE` | owner's setting | Non-blank. Sent in the request and checked in every notice |
| `MOMO_ACCESS_KEY` | **secret** | Non-blank. Held in `config.Secret` (prints `[redacted]` under `%v`, `%+v`, `%#v` and `slog`); read as a string only inside the signing function. Appears **only** inside the string that is signed: never in a request, a response, a log, an address or a browser file |
| `MOMO_SECRET_KEY` | **secret** | Non-blank. Same type and the same rule; it keys the signature |
| `MOMO_API_URL` | optional | MoMo's base address, **default `https://test-payment.momo.vn`** (the sandbox); `/v2/gateway/api/create` is appended by the code. An absolute `http://` or `https://` address with a host, no query, no fragment, no user info; a trailing `/` is removed. Going live changes this value (`https://payment.momo.vn`), never code; a run with no route to MoMo points it at a stand-in (SC-36) |
| `VIETQR_BANK_BIN` | owner's setting | Exactly six digits (the bank's NAPAS BIN, for example `970436`) |
| `VIETQR_BANK_NAME` | owner's setting | 1–100 characters after trimming, no control character; what the customer reads |
| `VIETQR_ACCOUNT_NUMBER` | owner's setting | 1–19 ASCII letters or digits |
| `VIETQR_ACCOUNT_NAME` | owner's setting | 1–100 characters after trimming, no control character |

`Bank.Set()` is **all four** valid; `MoMo.Available()` is the three non-blank and `MOMO_API_URL` valid. **A missing or malformed value never stops the server** (SC-36): the way is not listed, `/healthz` answers 200, and each such key is logged once at `warn` **by name only** ("VIETQR_BANK_BIN is not valid; bank transfer is not offered"), never a value. This **departs from `online-payment-with-vnpay.md` S17**, which stops the start-up for a malformed `VNPAY_PAYMENT_URL`: VNPay is the shop's only way to pay, and a wrong page would silently send every customer to the wrong place; these two ways are optional extras whose absence is a normal state, a typo costs the extra and not the shop, and "not listed" can never send money to the wrong place (S7). `PUBLIC_BASE_URL` (the VNPay story's) is the origin of both MoMo addresses; it is the way to pin them in a deployment.

### 4.2 MoMo — the create-payment request and the notice's signature

`PaymentURL(ctx, o orders.Payable, c orders.Client)` is the seam of S5. **Only `o.ID` is trusted**: the provider re-reads the order, so an amount, a number or a time passed in from anywhere is never used (SC-7).

1. **Reserve** (one transaction, `SET LOCAL lock_timeout = '2s'`): `orders.LockOrder`; `orders.CheckPayable(locked.State(), now)` (a refusal is returned as the `*orders.Refusal`, the route's `409`); `count(*)` of the order's attempts, **50 is a refusal** (`payments.MsgTooManyAttempts`); the reference `<order number>-<count+1>` (at most 50 characters here: a longer one is a defect, `500`); `INSERT INTO payment_attempts` with `provider = 'momo'`, the locked row's `total` and `placed_at + 30 minutes`; **commit**. A deadlock (`40P01`) is retried once.
2. **Ask MoMo, with no transaction and no lock held**: `POST <MOMO_API_URL>/v2/gateway/api/create`, `Content-Type: application/json`, the body below, under the **remaining** time of the caller's context (the route's 3 seconds cover steps 1 and 2 together). Redirects are not followed; the response body is read up to 64 KiB; **no retry**: a second call would be a second payment address for one attempt.
3. **Read the answer**: HTTP `200`, a JSON object whose `resultCode` is `0`, whose `payUrl` is an absolute `http(s)` address with a host (at most 2,048 characters), and whose `orderId`, when present, equals the reference. Anything else — another status, a body that is not JSON, a non-zero code (for example an amount above MoMo's limit), no address, a timeout, a refused connection — is an error: the route answers `500`, placing leaves `payment_url` `""`. The log line holds the order's number, the HTTP status and MoMo's `resultCode` and `message`; **never** the request or response body, the signature, `payUrl` or a key.

| Request member | Value | Source |
|---|---|---|
| `partnerCode` | the shop's partner code | `MOMO_PARTNER_CODE` |
| `requestId` | the reference (`DOI-24818-1`): different for every attempt | step 1 |
| `orderId` | the same reference | step 1 |
| `amount` | `total`, **whole VND, a JSON integer** (`3140000`; MoMo's unit is not ×100) | the **locked row** |
| `orderInfo` | `Thanh toan don hang DOI-24818` (ASCII, no diacritics) | the locked row's number |
| `redirectUrl` | `<origin>/payment/return` | the origin rule |
| `ipnUrl` | `<origin>/api/v1/payments/momo/ipn` (the **public** path, with `/api`) | the origin rule |
| `requestType` | `captureWallet` | constant |
| `extraData` | `""` | constant |
| `lang` | `en` | constant (the shop's pages are English) |
| `signature` | below | |

**The origin** is VNPay's rule (`online-payment-with-vnpay.md` §4.2, `vnp_ReturnUrl`): `PUBLIC_BASE_URL` when set, else the forwarded scheme and host, else the request's, **validated either way** (scheme `http`/`https`, a host of letters, digits, `.` and `-`, an optional port); a forged host makes the call an error, never an address that sends a customer — or MoMo — somewhere else.

**The signatures** (HMAC-SHA256, lower-case hex, keyed with `MOMO_SECRET_KEY`; values are used **verbatim**, not URL-encoded; the string is `name=value` joined by `&` with the names in this order):

- **The request**: `accessKey=<MOMO_ACCESS_KEY>&amount=<amount>&extraData=&ipnUrl=<ipnUrl>&orderId=<orderId>&orderInfo=<orderInfo>&partnerCode=<partnerCode>&redirectUrl=<redirectUrl>&requestId=<requestId>&requestType=captureWallet`. The access key is in this string and **not** in the request.
- **The notice** (§3.5): `accessKey=<MOMO_ACCESS_KEY>&amount=…&extraData=…&message=…&orderId=…&orderInfo=…&orderType=…&partnerCode=…&payType=…&requestId=…&responseTime=…&resultCode=…&transId=…`, each value as MoMo sent it (`""` for an absent optional string, the integer's own text for a number).
- **To verify**, the same string is built from what was received, the received `signature` is hex-decoded (a wrong length or a non-hex character fails at once) and compared with `hmac.Equal`.
- **Test vectors.** Dev computes at least **two signatures per direction with an independent tool** (`printf '%s' '<string>' | openssl dgst -sha256 -hmac '<secret>'`) over typed-out strings and writes them as literals: the request for a 3.140.000 ₫ order, and a notice with an empty `extraData`, so the algorithm cannot change without a test failing. **Known limit, written so nobody rediscovers it**: this design was written without being able to open MoMo's current documentation page (its server refused the reader); the field list and the order above are MoMo's published v2 one-time-payment contract as quoted by its own samples. If a real sandbox notice ever fails to verify, the signing file and the required-members rule of §3.5 are the two places to look, and they are one file.

### 4.3 The VietQR payload and the QR picture

**The payload** is the EMVCo merchant-presented text that Vietnamese banking apps read (NAPAS 247, "VietQR"): tag, two-digit length, value, concatenated; every value ASCII. Built by a pure function of `(bin, account, amount, note)`:

| Tag | Length | Value |
|---|---|---|
| `00` | `02` | `01` — payload format |
| `01` | `02` | `12` — point of initiation: **dynamic** (it carries an amount) |
| `38` | computed | Merchant account information: `00` `10` `A000000727` (NAPAS); `01` `<len>` of (`00` `06` the BIN, `01` `<len>` the account number); `02` `08` `QRIBFTTA` (transfer to an account) |
| `53` | `03` | `704` — Vietnamese đồng |
| `54` | computed | The amount in whole VND, digits only, no leading zero, 1 to 13 digits |
| `58` | `02` | `VN` |
| `62` | computed | Additional data: `08` `<len>` the **note** (the order's number, at most 25 characters) |
| `63` | `04` | The checksum: **CRC-16/CCITT-FALSE** (polynomial `0x1021`, initial `0xFFFF`, no reflection, no final xor) of **every character before it plus the characters `6304`**, as four upper-case hexadecimal digits |

The string up to the checksum, for BIN `970436`, account `0071001234567`, `3140000` and `DOI-24818`, is `00020101021238570010A00000072701270006970436011300710012345670208QRIBFTTA5303704540731400005802VN62130809DOI-248186304`. **Test vectors**: the CRC of the ASCII string `123456789` is `29B1` (the algorithm's published check value), and Dev computes the checksum of the string above, and of one more with a different amount, with an independent tool (for example `python3 -c "import binascii;print(format(binascii.crc_hqx(b'<string>', 0xFFFF), '04X'))"`) and writes both as literals. A test splits the payload back into tags and asserts the BIN, the account, the amount and the note, and that the length of every tag is right and the checksum validates.

**The picture** is drawn by the **backend** from that payload with `github.com/skip2/go-qrcode` (MIT, no dependencies): `qrcode.New(payload, qrcode.Medium)`, `DisableBorder = true`, `Bitmap()`, each row written as a string of `0`/`1` (§3.4 `qr`). The page adds the 4-module quiet zone itself (`TransferQr`'s `QUIET`). No third-party QR or image service is called and nothing about the account leaves the shop's own origin (story decision 6, SC-12). `go.mod` and `go.sum` are changed only by `run_go_mod_tidy` at the backend step. **A test decodes the picture**: the rows are rendered to a bitmap with a quiet zone and decoded with a pure-Go decoder as a **test-only** dependency (`github.com/makiuchi-d/gozxing`), and the decoded text equals `payload`; if that dependency cannot be added, the fallback is a structural test (a square of `17 + 4·version` modules, the three finder patterns in three corners, the encoder's own version) and the Verify run scans the picture in a browser with a decoder — never a pass reported for an unscanned code.

### 4.4 The bank provider

`payments.Bank` satisfies `orders.Provider`: `Available()` is `Set()`; `PaymentURL` returns `("", nil)` and **writes nothing** (a bank order has no attempt and no address). It also satisfies `orders.Transferer` (declared by `orders`, the consumer, so `orders` imports nothing from `payments`): `Transfer(number string, total int64) (orders.TransferDetails, error)` builds §3.4's members from the settings. It is **registered even with no account** (as VNPay is registered with no keys), so an existing bank order of a shop whose owner removed the account answers `503`, not "not available yet"; whether it is **listed** is the separate `Offered` list (§6).

## 5. The time rules — `payments.RunExpiry`

The loop, its cadence (once at start-up, then every 30 seconds), its 25-second budget, its per-order transaction, its panic recovery and its logging are the VNPay story's (`online-payment-with-vnpay.md` §6) and **do not change**. What changes is the set of orders and the clock each is judged by:

| Rule | Ways | Window | Cancelled when |
|---|---|---|---|
| **30 minutes** (PAY-003) | `vnpay`, `momo` | `orders.PayWindow` | unpaid, status `placed`, `confirmed` or `packed`, placed at or before `now − 30 min`, not a sample |
| **24 hours** (PAY-020 item 5, new) | `bank` | `orders.TransferWindow = 24 * time.Hour` (new, beside `PayWindow`) | the same conditions with `now − 24 h` |

- **One candidate query** (ERD §6) returns `id` and the order's way; **the cancel is `orders.Cancel(ctx, tx, id, orders.CancelRule{PlacedBefore: now − window, Ways: that rule's ways})`**, so the conditions are judged **again under the row lock** by the `UPDATE`'s own `WHERE` (a mark-paid, a dealer's cancel, a shipped move that landed since the list was read leaves the order exactly as it is, and its pairs are not touched). `thirtyMinuteWays` becomes `{vnpay, momo}`; `transferWays` is `{bank}`; both are code, as the offered list was.
- **The 30-minute rule never touches a bank order** (its way is not in that set): one placed 2 hours ago is unchanged (SC-31). One placed exactly 24 hours ago is cancelled by the next run (`placed_at <= now − 24 h`).
- **Left alone by both**: paid orders, already cancelled ones (a dealer's early cancel returned its pairs once; the rule does not return them again), `shipped` and `delivered` orders (the dealer cannot cancel a shipped order, so the rule does not either), and the 14 sample orders `DOI-24804` to `DOI-24817` and the pairs on the shelf for them (SC-32). **A rule applies by the order's way, not by today's settings**: an unpaid MoMo or bank order is still given up after the owner removes its settings.
- A late payment for an order a rule already cancelled is recorded and `paid` is set while the status stays `cancelled` (SC-27, SC-18); the dealer refunds it by hand.
- **Logging** — the existing `Info` line gains a `rule` field (`30m` or `24h`); nothing is logged when nothing is due.

## 6. Go surface — who changes what

Names are Dev's where not written; the split is not. Each file stays well under 300 lines. The import graph stays acyclic: `payments → orders`, `admin → orders`, `orders` imports neither `payments` nor `admin`.

| Package | Change |
|---|---|
| `internal/config` | Fields for the eight settings (`MoMoPartnerCode`, `MoMoAccessKey` and `MoMoSecretKey` as `Secret`, `MoMoAPIURL`, a `Bank` value of four strings with `Set()`), their validation (§4.1) in new files (`momo.go`, `bank.go`), each invalid or missing key logged once by name; `config_test.go` gains the cases; no `%v`/`%+v` of the config or error text holds a secret |
| `internal/orders` | `Options` gains **`Offered []string`** (the listed ids in order; nil means `["vnpay"]`); the package variable `offered` goes (it becomes the service's field), `orderRequest.validate` takes the list, `paymentOptions` orders by `array_position`. `settle.go`: `pageWays = {vnpay, momo}`, `TransferWindow`, the bank sentence, `CheckPayable` in the order of §3.3, and `CheckTransfer(PayState) *Refusal` (cancelled, paid, not bank); `MarkPaid`'s comment says who may call it (E-conflict 2). New **`transfer.go`**: the `Transferer` interface, `TransferDetails` (the §3.4 members and JSON tags) and the route `GET /v1/orders/{id}/transfer` (`authorize(…, modePay)`, `CheckTransfer`, the provider's `Available()` and `Transfer`, the one error mapper `fail`). Item 17's files: `Patch.Paid *bool`, `Judge`'s bank rule, `Update`'s `MarkPaid`, the way-specific `DELETE` sentence (§3.6), and `admin/orders_input.go` reading `paid` (JSON `true` is accepted, any other value is the by-name `422`) |
| `internal/payments` | The **only writer** of `payment_attempts` and `payment_notices` still. New: `momo.go` (the provider: `Available`, `PaymentURL`, the reservation), `momo_request.go` (build, sign, send, read — pure where it can be), `momo_sign.go` (the two signatures), `momo_notice.go` (the handler of §3.5), `bank.go` (§4.4), `vietqr.go` (the payload and the CRC, pure), `qr.go` (the rows). `store.go`: `find` takes the provider (`WHERE a.reference = $1 AND a.provider = $2`) and the notice insert takes the row's fields, so VNPay's and MoMo's notices share one store. `expiry.go`: the rules of §5. `payments.go`: `Register(mux, v *VNPay, m *MoMo)` mounts both notices |
| `internal/server` | Registers **three providers always** (`vnpay`, `bank`, `momo`) and builds `Offered` from the settings: `vnpay`, then `bank` when `Bank.Set()`, then `momo` when `MoMo.Available()`; `payments.Register(mux, vnpay, momo)` |
| `cmd/api` | **No change**: `payments.RunExpiry` already runs |
| `code/backend/.env.example` | The eight keys, **no value**, each with its one-line comment (the owner supplies them in the console; `MOMO_API_URL` empty means MoMo's sandbox) |
| `go.mod`, `go.sum` | `github.com/skip2/go-qrcode`; test-only `github.com/makiuchi-d/gozxing` (§4.3); by `run_go_mod_tidy` only |

**Why the transfer route lives in `orders`.** `authorize` is `orders`' private function and the one place that decides who may open an order; a second copy in `payments` would be a second place that could be made to say yes. `orders` declares what it needs (`Transferer`) and the bank provider satisfies it, the same direction as `Provider`.

## 7. The frontend's swap from the mock

All of it is in `src/lib/more-ways-to-pay-vietqr-transfer-and-momo.ts`, `TransferQr.tsx`, `lib/order-page-and-pay-now.ts` (one line), `lib/payment-words.ts`, `TransferBlock.tsx`, the mock's deletion and the tests. `Checkout.tsx`, `Product.tsx`, `PaymentRows.tsx`, `PaymentResult.tsx`, `AdminOrderPayment.tsx` change **not at all** (they already read the shop's answers).

| # | Change | Why |
|---|---|---|
| 1 | `readPaymentOptions` becomes `api.get<{ items: PaymentOption[] }>("payment-options")` (the `mockOptions()` lines go); a non-array `items` is still `[]` | §3.1 |
| 2 | `readTransfer(order)` becomes `api.get<TransferDetails>("orders/" + encodeURIComponent(order.id) + "/transfer", undefined, { "X-Order-Token": token })`, the token read **as `lib/order-page-and-pay-now.ts` reads it for `GET /v1/orders/{id}`** (with none held the header is left out and a linked customer's session speaks). An `ApiError` of status **503 or 409 → `{state: "gone"}`**; any other failure, a network failure (status 0) and a 200 whose body fails `detailsOf` → `{state: "failed"}`. The `number` and `total` arguments go | §2, rows 2 and 5 |
| 3 | `TransferDetails` gains `qr: string[]`. `detailsOf` accepts the body only when the five text members are non-empty strings, `amount` is a number, and `qr` is an array of at least 21 strings of one length equal to the array's length, each matching `^[01]+$`; else `null` (the block says "could not load") | §3.4 |
| 4 | `qrModules` and `standInModules` are replaced by `qrRows(qr: string[]): boolean[][]`; **`TransferQr` takes `rows` instead of `payload`** and keeps its `QUIET` margin, its `role="img"`, its name and its colours. `TransferBlock` passes `qrRows(details.qr)` | S2 |
| 5 | `markPaid(id, status)` becomes `api.put<{ status?: unknown }>("orders/" + encodeURIComponent(id), { paid: true })` — the body is `{"paid": true}` and **nothing else**; an `ApiError` 401 or 403 → `expired`, every other failure → `failed`; the answer's `status` is read, as now | §3.6 |
| 6 | Delete `src/lib/mock/more-ways-to-pay-vietqr-transfer-and-momo.ts`, the `viaDemo` import and call in `lib/order-page-and-pay-now.ts`, and every `?mock=` reading. **`src/lib/mock/orders.ts` stays** (the Track page imports it) | TL's UI review, should-fix 3; SC-39 |
| 7 | **Words.** The keys `pages.order.transfer.*` and `pages.payment_return.momo.timed_out.*` are **already in `content.json`** (this design's PR). Draw each word with `<T k=…>` or put `data-edit={key}` on its element, and **delete the hard-coded fallbacks of `lib/payment-words.ts`** (decision 10: no hard-coded words) | TL's UI review, should-fix 1 |
| 8 | **Dealer page.** `AdminOrder.tsx` is edited on top of `main`, but item 17's UI PR #77 rewrites that page and merges first: drop this PR's edit to `AdminOrder.tsx` and mount `AdminOrderPayment` inside #77's `OrderFacts` "Payment status" fact (it already takes `transactionNo` and `onMarked` for that) | TL's UI review, should-fix 2 |
| 9 | The three test files that `?mock=` or import the mock become `vi.mock("../lib/api")` stubs that throw the **real** `ApiError` (§2, row "`?mock=`…"); keep every assertion; add: the transfer call carries the order's token in the header and never in an address; **no token held → no header**; `409` and `503` draw nothing, `404`/`500`/network draw "could not load"; the `qr` rows reach the picture (a 21-row body draws a 29-unit viewBox); a body with a ragged `qr` is "could not load"; `PUT` carries `{"paid": true}` only | SC-13, SC-15, SC-22 |

## 8. Tests the backend PR ships

Database cases run in CI (`TEST_DATABASE_URL`) and **skip without it**: their first real run is CI, and Verify must confirm they ran and did not skip. The pure ones run everywhere. **No test-only route, header or knob is added to the backend**: MoMo's side is a `httptest` stand-in the test points `MOMO_API_URL` at, and the notices are signed by the test with the run's secret.

| Test | Proves |
|---|---|
| **MoMo signatures**: the independent vectors of §4.2 (request and notice); an empty `extraData`; names in order; a signature one character off, absent, made with another secret, 63 or 65 characters, non-hex, upper-case (accepted); a foreign partner code | SC-7, SC-25 |
| **The request**: every member of §4.2 exact for a 3.140.000 ₫ order (`amount` `3140000`, `DOI-24818-1` then `-2`, `redirectUrl`, `ipnUrl` with `/api`); the amount comes from the locked row, not from `Payable` and not from the request; the access key and the secret appear **nowhere** in the body; `MOMO_API_URL` is the base of every call; the origin rule and a forged host; 20 simultaneous asks → 20 distinct references; the 51st → `409` and no row; an order 31 minutes old, a paid, a cancelled and a bank order → `409`, no row, **no call to the stand-in** | SC-6, SC-7, SC-9 |
| **MoMo answers**: unreachable, slower than 3 s, `resultCode` non-zero, no `payUrl`, a body that is not JSON, a redirect → placing is `201` with `payment_url ""`, the attempt row exists, the order is saved, unpaid, `placed`, pairs taken; "Pay now" → `500` and the page's retry works; the stand-in sees no transaction held (a second order can be locked while it sleeps) | SC-8, E3 |
| **The notice's table test** over a fake store: every row of §3.5 — `204`, `400`, `500` — the order of the checks (a notice wrong in two ways gets the first), an extra member is ignored, a missing or mistyped member, an 8 KiB-plus and a 16 KiB-plus body, settings unset → `400` to everything, a failing store → `500` with **nothing written** and the same notice afterwards processed normally | SC-25, SC-28 |
| **The notice against PostgreSQL**: a success pays the order, `placed → confirmed` (`packed` stays `packed`), keeps `transId`, `payType`, the amount, `'0'` and the time; the same notice again → `204`, one row; **two at once and, in a second run, twenty** → all `204`, one success row; a failure (`resultCode` ≠ 0) → `204`, unpaid, one failure row, an identical twin still one; a success for the next attempt afterwards pays; a success for a **cancelled** order → `204`, `paid` true, status `cancelled`, the shelf unchanged; a wrong amount, an unknown reference and a VNPay reference → `400`; 100 runs of each refusal change no order and write no row; 100 runs of two identical notices pay once; the CHECK refuses `('0','00')` | SC-23 to SC-28, SC-38 |
| **`CheckPayable` and `CheckTransfer`** over every status × paid × way (`vnpay`, `bank`, `momo`, `cod`, unknown) × clock (29:59, 30:00, 31:00): the order of the rules and the sentences, the provider never asked for a refused order | SC-9, SC-10, SC-13 |
| **The transfer route**: `200` with exactly the seven members and `Cache-Control: no-store` for the order's token and for a linked customer; the **same 404 body** for no token, another order's token, a dealer session without the token and an unknown id; `409` ×3 for a paid, a cancelled and a VNPay order; `503` after the account is unset; `?amount=1` and a body change nothing; the answer holds no phone, email, address or note; **no other route's body holds any of the four settings** (options, the public order read, `GET /v1/payments`) | SC-11 to SC-13 |
| **VietQR and the picture**: §4.3's vectors, tag-by-tag parse, the amount is the stored total, a note over 25 characters is an error, a BIN or account with a non-digit never reaches the builder; the decode round trip | SC-12 |
| **The dealer's mark**: `PUT {"paid": true}` as admin and as staff on a bank order at `placed` → `200`, `confirmed`, nothing else changed; at `confirmed`, `packed`, `shipped`, `delivered` → status kept, `paid` true; at `cancelled` → `paid` true, status `cancelled`, the shelf unchanged; already paid → `200`, no change; `paid: true` on `vnpay`, `momo`, `cod`, `paid: false`/`null`/`1`/`"true"` anywhere, and `paid: true` with another editable field on a non-bank order → `422` naming `paid`, **nothing applied**; no session `401`, a customer `403`; two at once → paid once, both `200`; `{paid: true}` against the 24-hour cancel, repeated → exactly one of the two outcomes of ERD §6 | SC-17 to SC-21 |
| **The rules**: a `momo` order at 31 minutes, `placed`/`confirmed`/`packed`, is cancelled once and its pairs return once, the lists agreeing; at 29 minutes, paid, cancelled, `shipped`, `delivered` unchanged; a `bank` order at 25 h and at exactly 24 h is cancelled, at 23 h and at 2 h unchanged, paid or `shipped`/`delivered` at 25 h unchanged; samples of every way unchanged and their shelf too; a bank order the dealer cancelled early is not returned again; two runs at once cancel once; a notice or a mark racing a run, repeated, ends in exactly one of the two outcomes | SC-30 to SC-32 |
| **Options and placing**: all, VNPay-only, bank-only, MoMo-only, none; a blank or malformed value counts as not set; order `vnpay`, `bank`, `momo`; `POST /v1/orders` with a way not listed → `422`, no order, no pair; `bank` → `201` with `payment_url ""`, no attempt, VNPay's keys missing; `momo` → `201` with the stand-in's address and one attempt; `vnpay` without keys → `503` and nothing created | SC-1 to SC-6 |
| **Settings and secrecy**: `config.Load` with none of the eight → starts, nothing listed; malformed values start the server and list nothing, logging the key's name and not its value; a distinctive access key and secret key are in **no** captured log line, response body or header after a payment, a bad notice, a `503` and a failed call to MoMo; `.env.example` names the eight keys with no value | SC-36 |
| **Route table**: `GET /v1/orders/{id}/transfer` and `POST /v1/payments/momo/ipn` exist; `POST` to the first and `GET` to the second are the `404` envelope; `GET /v1/payments/momo` is still the record; `GET /v1/payments/momo/ipn` is `404` | §1 |
| **Schema** (ERD §8): the two files apply in order; the check refuses what it is for; the note migration is conditional and idempotent | SC-37 |

## 9. Decisions, and where this departs from the merged design

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| S1 | Where does `GET /v1/orders/{id}/transfer` live? | **In `orders`**, calling `authorize(…, modePay)`; the account comes from a `Transferer` that `orders` declares and `payments.Bank` satisfies (§6) | In `payments`: `authorize` is `orders`' private function, and a second copy would be a second place that could be made to say yes; exporting `authorize` would widen the surface that item 6 and item 15 keep narrow |
| S2 | Who draws the QR: the browser or the backend? | **The backend**: the payload is built in Go (where the checksum is tested against independent vectors) and the same function's output is drawn into modules and decoded in a test, so "the code a customer scans is the payload the test read" is one end-to-end proof. The browser gets rows to paint and **no new dependency** (`package.json` is untouched, no lockfile churn against the open PRs) | A browser library (`qrcode`, `qrcode-generator`): a dependency in every build, a second place that must agree with the payload, and a decoding test that needs a canvas in jsdom. A third-party image service: refused by story decision 6 (the account would leave the shop). A hand-written encoder: a QR encoder is exactly the code not to write twice |
| S3 | Names of the transfer members | **The mock's** (`holder`, `bank`, `account_number`, `amount`, `note`, `payload`) plus `qr`; the UI is built and reviewed against them | Fresh names (`account_name`, `bank_name`, …): every difference is rework of approved code for no gain |
| S4 | MoMo's answers | `204` accepted or already processed; `400` refused, with the project envelope and **no `errors`**; `500` on a database failure (story decision 8). A repeated notice is `204`, not an error, so MoMo stops retrying | `200` with a body: MoMo asks for `204`. `409` for "already processed": MoMo would retry a notice that has been handled. An echo of the failing check: it would tell a stranger which of signature, reference and amount was wrong |
| S5 | Registered versus listed | **Providers are always registered** (`vnpay`, `bank`, `momo`); **`Offered` lists** the ones whose settings are valid. So a `momo` order of a shop whose keys were removed answers `503` on "Pay now" (SC-9) and a bank order's transfer route answers `503` (SC-13), while placing with an unlisted way is `422` (SC-2) | Registering only what is configured: "Pay now" for an existing MoMo order would answer "not available yet" (`409`), the sentence the VNPay story removed for the same reason (S1 there) |
| S6 | Order of the options | **The code's order** by `array_position`, not `payment_methods.seq` | Reordering `seq` in a migration: it changes the dealer's `payments` list and every sample order's context for a rule the code states in one line; the PAY-022 order is about what is offered, which is code |
| S7 | A malformed optional setting | **The way is not listed, one `warn` by key name, the server starts** (§4.1) | VNPay's S17 (stop the start-up): the shop would be down for a typo in an extra |
| S8 | Does MoMo's call happen inside the attempt's transaction? | **No**: reserve and commit, then call (ERD E3) | See the ERD: a held row lock and pooled connection across network I/O, or an unrecorded reference MoMo may be paid against |
| S9 | How does the dealer mark a bank order paid? | `PUT /v1/orders/{id}` with `{"paid": true}`, the route the story, the SRS and the UI already name | `POST /v1/orders/{id}/mark-paid`: a second way to write `paid` and a second authorization path, for one boolean the existing route's one lock already serialises |
| S10 | May a non-bank order ever be marked paid by hand? | **No**, whatever the status (ADMIN-016 AC-11, AC-12, SC-19) | Allowing it for a MoMo or VNPay order "when the notice was lost": a lost notice is the provider's to resend; a person marking paid on trust is the hole PAY-002 exists to close |
| S11 | Required and optional members of MoMo's notice | The eight of §3.5 are required; five strings default to `""` | Requiring all thirteen: a real failure notice that omits `payType` would be refused and retried for nothing. Requiring only the signature: a notice with no `amount` could not be checked. **Risk named**: if MoMo's real notice omits a required member, a real payment is refused (`400`) and MoMo retries; the first sandbox payment on a public address closes it |
| S12 | The MoMo payment page's own expiry | **Not sent** (default: no `orderExpireTime`) | See Q1 |
| S13 | Does a bank mark-paid need an audit row? | No (ERD E7): `orders.updated_at` and one log line | A table of dealer actions: its own item |
| S14 | Which sentence does the dealer's `DELETE` use for a paid, cancelled order? | The way's own (§3.6), VNPay's unchanged | Leaving "refund it in VNPay's portal" on a bank or MoMo order: it sends the dealer to the wrong place for a refund |

**Conflicts with merged decisions, stated and resolved**

1. **`_base.md` §3.5** ("rows come in the `payment_methods` order") against PAY-022 AC-1 and SC-1 ("VNPay first, then bank transfer, then MoMo"): resolved by S6; the `_base.md` text is superseded for this route.
2. **`_base.md` §2.3**, the `payment_unavailable` row ("`VNPAY_TMN_CODE` or `VNPAY_HASH_SECRET` is not set"): its meaning widens to "the settings the order's way to pay needs are not set". Code, status and envelope are unchanged; no code is added.
3. **`online-payment-with-vnpay.md` §3.1** (the state sentences, and "a way to pay that is not VNPay: `cod`, `bank`, `momo` … 'This order cannot be paid online.'") and **order-page ERD E2** ("not offered online" is `offered`): `momo` is now payable on a provider's page and `bank` has its own sentence; `cod` keeps the old one. `CheckPayable` reads the **code constant `pageWays`**, not the listed ids, so an order whose way was later unlisted is still judged by what it is (S5). Its tests change with it (`settle_test.go`: "momo is not offered yet" becomes payable).
4. **Item 17 §3.3 row 8 and §3.4 row 5**: §3.6 above, two narrow amendments, both stated as changes to rows and not as a second contract.
5. **VNPay story S17** (a malformed optional setting stops the server): not applied to these two ways (S7).
6. **VNPay story §6** (`thirtyMinuteWays = {"vnpay"}`, "a bank transfer is never in it"): extended as that file said it would be (`momo` joins; the 24-hour rule is `bank`'s own).
7. **`orders.MarkPaid`'s comment** and `erd/…` conflict 2: it names its three callers now.

No other. I read `services/_base.md` §2.1–2.8, §3.5, §3.10, §3.11, §3.13, §3.15, §3.16, §3.17, §5.1, §5.2, §7; `services/online-payment-with-vnpay.md` (all); `services/order-page-and-pay-now.md` §3, §12; `services/dealer-handles-orders-status-courier-payment.md` (all); `docs/payments/SRS.md` PAY-020 to PAY-022 and the story's 39 criteria; and the code on `main` (`internal/orders/{orders,options,create,pay,settle,access,read}.go`, `internal/payments/*.go`, `internal/config`, `internal/server/server.go`, `cmd/api/main.go`, the migrations).

**Open questions** (none blocks the backend; each has its default)

| # | Question | Default | Who decides |
|---|---|---|---|
| Q1 | MoMo's payment page may stay open longer than the shop's 30 minutes, so a customer can pay an order the rule has already given up (recorded and refunded by hand, SC-27). MoMo's create-payment API may accept an optional expiry in minutes outside the signed members (`orderExpireTime`); this design could not open MoMo's documentation page (its server refused the reader) to confirm it | Not sent. At the first sandbox payment Dev checks the documentation; if the member is accepted, send the minutes left to the order's deadline (never below MoMo's minimum) and add one test; if not, nothing changes | Dev at the backend step; TL if the answer changes the contract |
| Q2 | Some banking apps drop or reject a hyphen in the transfer note (`DOI-24818`) | As the SRS words it: the note is the order's number exactly. The dealer matches by amount and number and can read a note without its hyphen | Owner, after the first real transfer |
| Q3 | MoMo's real notice may omit a member this design requires (S11) | As S11 | Dev, at the first sandbox payment |

## 10. Order of work, gates and migration plan

1. **Branch and gates.** Cut the backend branch from the **UI branch** `feature/more-ways-to-pay-vietqr-transfer-and-momo-ui` (it deletes the mock that branch adds); UI PR #91 merges first. **Item 17 must be on `main` first** (UI PR #77, then BE PR #88): §3.6 amends its `Judge`, `Update`, `orders_input` and `DELETE`. Items 10 (PR #85) and 15 (PR #69) also add `Register` lines to `server.go` and touch the order-access code: the second of each pair to merge keeps every line and `AccountID` in **both** access modes, so the transfer route inherits the linked customer's access. Grep the head for `^<<<<<<<` and `^>>>>>>>` before any merge.
2. **Order of work.** `config` (settings) → `orders` (`Offered`, `CheckPayable`, `CheckTransfer`, `TransferWindow`, the transfer route) → `payments` (`vietqr`, `qr`, `bank`, then `momo*`, the store, the rules) → item 17's `paid` branch and the `DELETE` sentence → `server` wiring and `.env.example` → the frontend swap (§7) → the tests of §8.
3. **Migrations** — forward, backward and safety are in the ERD §8: two files, stamped after `20261126110000_uploads`; neither rewrites a row of a table that already holds data except one conditional `UPDATE` of one note; the check change cannot fail on existing rows (all satisfy its first arm); the down files are for local resets, the first refusing (`23514`) to drop a check under an existing MoMo success, which is intended.
4. **The owner** supplies the seven settings in the console (already requested) and registers `<public origin>/api/v1/payments/momo/ipn` in MoMo's merchant portal (declared in the console already); `PUBLIC_BASE_URL` should be set in a deployment so MoMo's two addresses do not depend on forwarded headers. Until the keys are in, every criterion that needs them is **UNVERIFIED** with the missing instrument named, never a pass; MoMo cannot reach a `.localhost` deployment, so "paid exactly once" is observed with signed test notices there.
5. **Database tests** first run in CI; Verify confirms they were not skipped. The order page at 390, 320 and 1024 px with a long holder and account number has **not** been walked in a browser (UI review): Verify must (`/orders/<id>` of a real bank order, not `?mock=`, which is gone).

| Who | What they must know from this design |
|---|---|
| Dev, backend step | §6 for the files, §3 for every answer, §4.3 for the payload and its vectors, §7 for the swap. **Nothing here is optional except what §9 marks as a default.** If anything cannot be done as written, reply with the blocker rather than widening the contract |
| Test lead | Settings are a precondition (UNVERIFIED without them, never a pass). MoMo's side is a stand-in at `MOMO_API_URL` and notices are signed by the case (§8); the 24-hour and 31-minute cases move `placed_at` (ERD §9). The refusal bodies carry no `errors` and no echo |
| Items 10, 15, 17 | §10 item 1 |

## 11. Traceability

| Criteria | Enforced by |
|---|---|
| SC-1, SC-2, SC-3 | §3.1 (what is listed, the order), §3.2 (`422`, the 503 gate), §6 (`Offered`), §4.1 |
| SC-4 | The UI PR (the shoe page reads §3.1); §7 row 1 |
| SC-5 | §3.2 (`bank`), §4.4 |
| SC-6, SC-7, SC-8, SC-9 | §3.2 (`momo`), §3.3, §4.2, ERD E3 |
| SC-10 | §3.3 (the new `409`), `CheckPayable` |
| SC-11, SC-12, SC-13 | §3.4, §4.3, §6 (`transfer.go`) |
| SC-14, SC-15, SC-16 | §3.4 (`409`/`503` draw nothing), §7 rows 2–4, the UI PR |
| SC-17, SC-18, SC-19, SC-20, SC-21 | §3.6, ERD §6 |
| SC-22 | §3.6 (`transaction_no` is provider-agnostic: ERD §3 pattern 8), the UI PR, §7 row 5 |
| SC-23 to SC-29 | §3.5, ERD §2.1 and §6 |
| SC-30, SC-31, SC-32 | §5, ERD §6 |
| SC-33, SC-34 | The UI PR and the unchanged `GET /v1/orders/{id}` (§1) |
| SC-35 | The whole chain: §3.2 to §3.6, §5, the UI |
| SC-36 | §4.1, §6 (`.env.example`), §8 "Settings and secrecy" |
| SC-37 | ERD §2.3 and §8 |
| SC-38 | §3.5 notes, §8 |
| SC-39 | §6 and §7: the pages change only where they named VNPay or "Pay now"; no mock is left behind |
