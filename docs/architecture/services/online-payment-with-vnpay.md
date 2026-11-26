# Service contracts — Online payment with VNPay (plan item 8, "S7")

Module `payments` (and the changes it makes in `orders`, `stock`, `config` and `server`) · Requirements PAY-001 to PAY-005, the payment half of ORD-006 (items 4–6, AC-7, AC-8, AC-10, AC-11) and ORD-007 item 9 · Story `docs/payments/stories/online-payment-with-vnpay.md` · UI PR #75 · Tables: `docs/architecture/erd/online-payment-with-vnpay.md`.

This file defines **the one new route (VNPay's notice), what changes in the three routes that already exist, the address and its signature, the three functions `orders` and `stock` export so that `payments` never writes another package's table, the 30-minute expiry, the settings, and the frontend's swap from its mock** — and nothing else. It extends project `services/_base.md` §3.5, §3.10, §3.11, §3.13, §3.17 and §2.8, S5's seam (`services/checkout-and-placing-an-order.md` §4) and S6's route (`services/order-page-and-pay-now.md` §3–§5, §12). It reuses the error envelope, the closed error catalog, the no-body rule and the `Idempotency-Key` table of `_base.md`, and adds **no error code** and no second list or error shape. The one exception to the envelope is the notice, and it is the project's own (`_base.md` §3.17). Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/payments/vnpay/ipn` reaches Go as `/v1/payments/vnpay/ipn`.

## 1. Endpoints

| # | Method and path | Auth | What this story does |
|---|---|---|---|
| 3.17 | `GET /v1/payments/vnpay/ipn` | VNPay's signature | **New** (§3.4): the only thing that marks an order paid |
| 3.13 | `POST /v1/orders/{id}/payment` | the order's `X-Order-Token` | Its last answer changes from `409` "not available yet" to `200 {"payment_url"}` or `503`; one refusal is added (the 30 minutes ran out) and one bound (50 attempts); each success writes one `payment_attempts` row (§3.1) |
| 3.10 | `POST /v1/orders` | none | `503 payment_unavailable` between the field checks and the transaction; `payment_url` is a real address (§3.2) |
| 3.5 | `GET /v1/payment-options` | none | `is_available` becomes the key check; no change of contract (§3.3) |
| 3.11 | `GET /v1/orders/{id}` | token, dealer, linked customer | **No change.** The result page polls it (§2) |
| — | The 30-minute expiry | none (a goroutine in the API process) | New (§6) |

**Mounting.** `payments.Register(mux, …)` mounts the one new route, the **literal** pattern `GET /v1/payments/vnpay/ipn`, and `server.New` gains that one line beside the provider registration (§7). Any other method on that path falls to the catch-all and answers `404 not_found` in the project envelope (it is not VNPay calling). The path has **three** segments after `/v1`, so it can never be shadowed by the `payments` collection's `GET /v1/payments/{id}` (two); a route-table test pins both (`GET /v1/payments/vnpay` is still the record of the way to pay, `GET /v1/payments/vnpay/ipn` is the notice).

## 2. The mock, read and judged

The UI PR (#75) adds `src/lib/mock/online-payment-with-vnpay.ts`, imported only by `src/lib/online-payment-with-vnpay.ts` (two places: `transport`, and the `demoHeld` line of `heldFor`). It is unreviewed input: its shape is compared with the contract, and **where `_base.md` §2.8 / §3.11 / §3.13 and S6 already decided a shape, they win.**

| Mock | Contract | Judgment | Frontend change at the swap |
|---|---|---|---|
| `read(id, token)` answers `{status, body}` with `body = {id, number, status, paid}` | The public view of §2.8 (`GET /v1/orders/{id}`, §3.11) holds those four and more (`placed_at`, `items`, `total`, …) | **Sound.** The page reads only these four; names, types (`paid` is a boolean, `status` one of the six) and meaning match field for field | None |
| Every answer is `{status, body}`, a failure to connect is status `0` | `src/lib/api.ts` resolves a 2xx with the parsed body and **throws `ApiError`** (`status`, `code`, `message`; a failure to connect is status 0, code `network`) | **The mock's shape is not the real client's.** It is hidden inside one file | `lib/online-payment-with-vnpay.ts` only: `transport.read/pay` become `api.get/post` calls, an `ApiError` is mapped to `{status, body: {error: code}}`, a network failure to status 0 (§8) |
| **Any** `404` on a read is "no match" | A 404 is `not_found` only when it is the API's own body. A 404 that is not the envelope (a gateway's own page, a mis-routed `/api`) reaches `api.ts` as `ApiError` 404 with code `internal` (S3 §5.1; S6 decision P10) | **Too loose, and the one departure that matters.** With "any 404", a broken proxy rule would tell every customer "We can't match this payment to an order on this browser", and SC-33 (never the can't-match view when the shop cannot be asked) would fail in production while passing against the mock | `readOrder` returns `no-match` **only** for status 404 **and** `error === "not_found"`; every other failure is `check-failed`. A test with a 404 that has another code |
| `pay(id, token)` answers 200 `{payment_url}`, 409, 503 and 404 | §3.1: 200, 404, 400, 409 (six sentences), 503, 500 | **Sound** for what the page reads: only the status and `payment_url`; the page draws one line for 503 and for every other failure and re-reads the order after a 409. The 409 sentences are never read. The page follows an address only when `payAddress()` accepts it (`https:`) | None. **Note for Test:** a run whose `VNPAY_PAYMENT_URL` is `http:` builds the address but no browser page follows it; use the default (https) or an https stand-in |
| The 400 ms delay, `paidAfterMs`, `readFails`, the eight demo orders and `demoHeld` | — | Mock-only controls. **No delay, fault, sleep or "pay now" route, header or query parameter is added to the backend**: the 6-second and 30-second cases are made by the Test lead with a signed notice sent at the right moment, the 500 by an unreachable database | Go with the mock; delete `demoHeld` and the `[…storedOrders(), ...demoHeld]` line |
| In `no-match` after a **404 for a held token** the hook still returns `order = {id, number}`, so `Actions` draws "View your order" | The page's own rule (SC-32, story §5): one same view, "Track an order" pill, nothing about any order | TL's UI review, should-fix 1 | `usePaymentReturn` returns `order: null` whenever the phase is `no-match`; a test with a held token and a `not_found` 404 |
| `Wire` is declared again beside `lib/orders.ts`'s `Wire` with a different `body` type | — | Nit (UI review 2) | Keep one name in each file or, at the swap, name this one `Answer`; not required |
| Reference pattern `^(DOI-\d{1,15})-\d{1,9}$` | The shop's reference is `<order number>-<n>`, `n` from 1 to 51 | **Sound.** The number is read **only** to choose which stored token to ask with; it is never sent anywhere else | None |
| Content keys `pages.payment_return.*` added to `content.json` in the UI PR | The file is frozen for stories | Accepted by the UI review as the same one-time lift the orders and staff stories received: **new keys only**, additive | None. The backend PR does not touch `content.json` |

## 3. Contracts

### 3.1 `POST /v1/orders/{id}/payment` (project §3.13) — what changes

**Purpose, auth, request** — as S6 §3: "Pay now" and the result page's "Try again"; the order's `X-Order-Token` (the linked customer's session joins through `authorize`, S6 §12.4); no body. **Headers on every answer** — `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`.

**Order of checks** — S6 §3.1 steps 1 and 2 are unchanged (access `404`, a body `400`). Steps 3 to 5 become:

| # | Check | Refusal | Notes |
|---|---|---|---|
| 3 | **State**, in this order: cancelled → paid → a way to pay the shop does not complete online → **its 30 minutes have run out** | `409 conflict`, one of four **different** sentences (below) | `decide(ref, providers, now)`: the clock is `Options.Now`, passed in. The state rules live in **one** function (§5.3) that the route calls with the row `authorize` returned and the provider calls again with the row it has locked, so the sentences cannot drift. "Run out" is `now ≥ placed_at + 30 minutes` |
| 4 | **Availability**: the `vnpay` provider is **always registered** in a running shop; `Available()` is false while either key is missing | `503 payment_unavailable`, "Online payment is not available right now. Please try again later." | After state, as the story says: a stranger still gets `404`, a paid order still `409`, an order whose time ran out `409`. With **no** provider registered the sentence "Online payment is not available yet." is still what `decide` says — a shop built with `orders.Options{}` (tests) — but **no running shop gives it** (SC-6) |
| 5 | **The address**: the provider's `PaymentURL`, inside the route's 5-second deadline and the provider's own 3-second child (S6 §12.2, Q2) | `200 {"payment_url": "<address>"}`; an error, a timeout or an empty address is `500 internal` — except a `*orders.Refusal`, which is the `409` it names | Inside it the order's row is locked and the state is judged **again** (a notice or a cancel may have landed since step 3); a refusal there is the same `409` with the same sentence |

**Success** `200` `{"payment_url":"https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?vnp_Amount=314000000&vnp_Command=pay&…&vnp_SecureHash=<hex>"}` — built by the backend with **no call to VNPay** (§4.2). **Each success inserts exactly one `payment_attempts` row and changes nothing else**; every refusal inserts none.

| Code | HTTP | Trigger | Body message |
|---|---|---|---|
| `not_found` | 404 | Step 1, byte for byte S6 §3.3 (a dealer session without the token is a `404` too) | "Not found." |
| `bad_request` | 400 | Step 2: a body was sent | "This call takes no body." |
| `conflict` | 409 | Cancelled (paid or not) | "This order was cancelled, so it cannot be paid." |
| `conflict` | 409 | Already paid (and not cancelled) | "This order is already paid." |
| `conflict` | 409 | Unpaid, not cancelled, a way to pay that is not VNPay (`cod`, `bank`, `momo`, any other id): the 14 sample orders | "This order cannot be paid online." |
| `conflict` | 409 | Unpaid, not cancelled, **its 30 minutes have run out** — a sample VNPay order, or a placed order the check has not reached yet (within 30 seconds of the deadline). **New** | "The time to pay this order has run out." |
| `conflict` | 409 | The order already has **50** attempts. **New** (ERD E9) | "This order has had too many payment attempts. Please place a new order." |
| `payment_unavailable` | 503 | `VNPAY_TMN_CODE` or `VNPAY_HASH_SECRET` is not set | "Online payment is not available right now. Please try again later." |
| `internal` | 500 | The access statement or the attempt transaction fails or times out; the address cannot be built (§4.2: an origin that is not `http(s)://host[:port]`, an amount that overflows `×100`); an empty address. Generic body; the cause is logged beside the order id and request id, **never the token, the secret or the address** | "Something went wrong. Please try again." |

The 409 bodies carry **no** `errors` key. The page branches on none of the sentences; a person reads the message. Every earlier reference still leads to its order, so a customer who opened a second page and paid on the first is found (PAY-001 item 5).

**Not idempotent, by design.** Each success is a new attempt with a new reference (`_base.md` §2.6); two presses at the same moment get two different references (§4.2: the order's row lock). **No rate limit beyond the 50-attempt bound.**

### 3.2 `POST /v1/orders` (project §3.10) — what changes

Order of checks, with the new step in bold: body (415, 413, 400) → `Idempotency-Key` (400) → every field at once (422, including `payment` not offered: `cod`, `bank`, `momo` and unknown ids are `422` naming `payment`, SC-10) → **the keys (503)** → the transaction (409, 201, 200).

| Change | Rule |
|---|---|
| **The 503 gate** | After the field checks and before `place`: if the provider registered for `req.Payment` is not `Available()`, answer `503 payment_unavailable` with "Online payment is not available right now. Please try again later." **No order, no counter move, no pair taken, no token, no attempt** — nothing has been written or locked yet (SC-8). The gate also answers a replay of a used `Idempotency-Key` while the keys are missing; the first send's order stays, and the customer finds it by number and phone (`/track`). Accepted: a replay is rare and the keys are an owner setting |
| **`payment_url`** | After the commit, outside the transaction, `PaymentURL` is asked (S5 §4) with a 3-second limit. It makes its own transaction (§4.2): the order's row is locked and one attempt is stored. **An error, a timeout or an empty address leaves `payment_url` `""`** with a log line; the order stays saved, unpaid, `placed`, its pairs taken (SC-7), and the browser lands on the order page, which offers "Pay now". A replay builds a **new** attempt when the order is still payable (§2.6) |
| `Place` | **Unchanged.** A line of `create.go`, not of the transaction |

### 3.3 `GET /v1/payment-options` (project §3.5) — what changes

Nothing in the contract. `is_available` is `provider.Available()`: `true` only when **both** `VNPAY_TMN_CODE` and `VNPAY_HASH_SECRET` are set (non-blank after trimming). With a key missing the row is **still listed** with `is_available: false`, so Checkout draws the VNPay row, the alert and a disabled "Place order" (SC-9). The rule S5 added for the time before this story — "a way to pay with no provider registered is reported available" — stays in the code for a shop built without providers (tests); no running shop reaches it, because `server.New` always registers `vnpay` (§7).

### 3.4 `GET /v1/payments/vnpay/ipn` (project §3.17)

**Purpose** — VNPay's server-to-server notice of a payment's result, and **the only thing that marks an order paid**. **Traces to** — PAY-002, PAY-003 item 9, PAY-005. **Auth** — none: the signature is the proof. The browser never calls it, the result page never marks anything, and a session is neither read nor needed. **Request** — a query string only; a body is ignored. **Method** — `GET` only.

**Two exceptions to the project's contract, both VNPay's** (project §3.17, restated because they decide the code): every `vnp_*` parameter is accepted — no undefined-field `400`, the signature is the validation, and an extra `vnp_` field is simply signed with the rest — and **every** answer is `HTTP 200` with `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store` and a body of **exactly** `RspCode` and `Message`, never the error envelope:

| `RspCode` | `Message` | Meaning |
|---|---|---|
| `00` | `Confirm Success` | The notice was received (a success was recorded, or a failure was) |
| `01` | `Order not found` | The reference is none of ours |
| `02` | `Order already confirmed` | The order is already paid, or a twin notice won the race |
| `04` | `Invalid amount` | The amount is not the attempt's |
| `97` | `Invalid signature` | Not VNPay's, not ours, not parseable, or the keys are not set |
| `99` | `Unknown error` | The database failed; **nothing is half-written**, and VNPay sends it again |

**Query parameters** — what VNPay sends: `vnp_TmnCode`, `vnp_Amount`, `vnp_BankCode`, `vnp_BankTranNo`, `vnp_CardType`, `vnp_PayDate`, `vnp_OrderInfo`, `vnp_TransactionNo`, `vnp_ResponseCode`, `vnp_TransactionStatus`, `vnp_TxnRef`, `vnp_SecureHashType`, `vnp_SecureHash`, and any other `vnp_` field. A parameter that does not start with `vnp_` is ignored and is not signed. The return address VNPay gives the customer's browser carries **the same signed fields**; a customer who replays their own return address here delivers VNPay's true word, which is no exception to "only VNPay's signature marks an order paid" — and a replayed failure is one row (ERD E3).

**Processing, in this order.** The first rule that fails answers and changes nothing; every check before the first write is arithmetic or cryptography, so a stranger costs no database work.

| # | Check | Answer | Writes |
|---|---|---|---|
| 0 | The keys are set; the raw query is at most **8 KiB**; it parses (`url.ParseQuery`; a malformed escape or a `;` is a failure); **no `vnp_` name appears twice** | else `97` | nothing |
| 1 | `vnp_SecureHash` is present, 128 hex characters, and equals the HMAC-SHA512 of the received `vnp_` parameters (§4.3), compared in constant time and without regard to case; **and** `vnp_TmnCode` is the shop's. An empty query fails here | else `97` | nothing |
| 2 | `vnp_TxnRef` is the `reference` of a `payment_attempts` row of provider `vnpay` — any earlier reference of an order qualifies | else `01` | nothing |
| 3 | `vnp_Amount` is 1 to 18 ASCII digits, is divisible by 100, and `÷ 100` equals the attempt's `amount` (which is the order's `total`, and cannot change) | else `04` (a non-numeric amount too) | nothing |
| 4 | The order is not already `paid` | else `02` | nothing |
| 5 | **A failure notice** — `vnp_ResponseCode` is not `00` **or** `vnp_TransactionStatus` is not `00` (a cancelled payment, code `24`, included; an absent status too): insert a `payment_notices` row with `is_success = false`, `ON CONFLICT … DO NOTHING` (§6 of the ERD) | `00` | one row, or none for an identical twin; the order is untouched, stays unpaid, its status unchanged |
| 6 | **A success notice** — both are `00`: one transaction: `orders.MarkPaid` (`UPDATE … WHERE id = $1 AND NOT paid`; **0 rows → `02`**, rolled back), then insert the notice (`ON CONFLICT (attempt_id) WHERE is_success DO NOTHING`; **0 rows → `02`**, rolled back), commit | `00` | `orders.paid = true`; `status` `placed → confirmed`, every other status unchanged; one success row |
| — | A database error at any statement, a deadline, a lock timeout | `99` | the transaction is rolled back: the order unpaid, no notice kept |

**What a success does to the order** — `paid` is set and a `placed` order becomes `confirmed`; `confirmed`, `packed`, `shipped` and `delivered` keep their status. **An order already `cancelled` is marked paid and stays `cancelled`** (PAY-002 item 9, SC-16): the dealer sees it paid and refunds it in VNPay's portal. **No pair moves**: the pairs were taken at placing and returned at the cancel; a late payment neither takes them again nor returns them again.

**What is kept** (ERD §2.2) — `vnp_TransactionNo`, `vnp_BankCode`, `vnp_Amount ÷ 100`, `vnp_ResponseCode`, `vnp_TransactionStatus`, `is_success` (both `00`), `vnp_PayDate` read as Vietnam time into `paid_time` (NULL when absent or unparseable — a payment is never refused for a date). A missing text field is `''`. **No card number is received, and none is stored.**

**Two identical notices at the same moment** — both pass checks 0–4 (the order is unpaid when each reads it); both run `MarkPaid`; the second waits on the order's row, finds `paid` true when it resumes and gets `02`. Exactly one `00`, one row. Twenty at once: one `00`, nineteen `02` (SC-12).

**Notes** — p95 ≤ 2 s over 100 sequential notices (about four statements). The handler runs under a 5-second deadline and `SET LOCAL lock_timeout = '3s'`. **The log line** holds the code answered and, once the reference is known, the order's number and the request id — never the query, the signature, a transaction number or the hash secret; the access log records the route and never the query (project §2.1). With the keys unset the endpoint answers `97` to everything. VNPay cannot reach a `.localhost` run: the cases send **the same request VNPay sends, signed with the sandbox secret** (§4.4). **No IP allow-list and no rate limit**: the proxy chain hides VNPay's address and VNPay's retries would be refused by a limit; the signature, checked before any database work, is the guard (§10, S12).

## 4. The address, the signature, the settings

### 4.1 Settings and `Available()`

| Key | Kind | Rule |
|---|---|---|
| `VNPAY_TMN_CODE` | owner's setting (requested in the console), read only from the environment | Trimmed. Blank = not set. Sent in the address as `vnp_TmnCode`; never secret from VNPay, never sent to a browser except inside an address |
| `VNPAY_HASH_SECRET` | **secret**, requested in the console, read only from the environment | Trimmed. Blank = not set. Held in a type that prints as `[redacted]` under `%v`, `%+v`, `%#v` and `slog`, so it cannot reach a log line or an error by accident. Never in a variable that begins with `VITE_`, an address, a response or a log |
| `VNPAY_PAYMENT_URL` | optional | VNPay's payment page. **Default** `https://sandbox.vnpayment.vn/paymentv2/vpcpay.html`. Must be an absolute `http://` or `https://` address with a host and **no query and no fragment**; anything else stops the server at start-up with a message that names the key and **not its value**. Going live changes this value, never code (PAY-005 item 3) |
| `PUBLIC_BASE_URL` | optional | The shop's own origin, `https://shop.example.com` (an `http://` one for a local run): scheme and host with an optional port, **no path, query or fragment**; a trailing `/` is removed; anything else stops the server at start-up. When set it **is** the origin of `vnp_ReturnUrl`; when empty the origin is read from the request (§4.2) |

`Available()` is **both** of the first two non-blank. **A missing key never stops the server** (SC-8): it starts, `/healthz` answers 200, the options list says `is_available: false`, the route and the placing answer `503`, and the notice answers `97` to everything. `.env.example` names all four keys with **no value** (SC-39).

### 4.2 The address — `PaymentURL`

`PaymentURL(ctx, o orders.Payable, c orders.Client) (string, error)` is S5's seam. **Only `o.ID` is trusted**: the provider re-reads the order itself, so an amount, a number or a time passed in from anywhere is never used (SC-1). Inside one transaction:

1. `SET LOCAL lock_timeout = '2s'`; `orders.LockOrder(ctx, tx, o.ID)` — the order's row `FOR UPDATE` (§5.2). A missing order is an error (`500`).
2. The state is judged again under the lock by the one function of §5.3 (cancelled, paid, not online, 30 minutes run out) and a refusal is returned **as the `*orders.Refusal`** — the route answers it `409`, `create` logs it and leaves `payment_url` `""`.
3. `SELECT count(*) FROM payment_attempts WHERE order_id = $1` — `k`. **`k = 50` is a refusal** ("This order has had too many payment attempts. Please place a new order."). The reference is `<order number>-<k+1>` (`DOI-24818-1`, `DOI-24818-2`, …): at most 100 characters, different for every attempt, holding the order number. Two asks at the same moment cannot read the same `k`: the second waited on step 1.
4. `INSERT INTO payment_attempts (order_id, provider, reference, amount, expires_at)` with the locked row's `total` and `placed_at + 30 minutes`.
5. The parameters are built (below), signed (§4.3) and joined into the address. **A failure here rolls the transaction back** — no attempt row without an address. Then commit.

| Parameter | Value | Source |
|---|---|---|
| `vnp_Version` | `2.1.0` | constant |
| `vnp_Command` | `pay` | constant |
| `vnp_TmnCode` | the terminal code | `VNPAY_TMN_CODE` |
| `vnp_Amount` | `total × 100`, decimal digits (314000000 for 3.140.000 ₫) | the **locked row**; computed with an overflow check — an overflow is an error, which no real order reaches |
| `vnp_CurrCode` | `VND` | constant |
| `vnp_TxnRef` | the reference | step 3 |
| `vnp_OrderInfo` | `Thanh toan don hang DOI-24818` (ASCII, no diacritics, at most 255) | the locked row's number |
| `vnp_OrderType` | `other` | constant |
| `vnp_Locale` | `en` | constant (the shop's pages are English) |
| `vnp_ReturnUrl` | `<origin>/payment/return` | `PUBLIC_BASE_URL` when set; else `X-Forwarded-Proto` and `X-Forwarded-Host` of the request, else its scheme and `Host` (the `Origin` of `orders.Client`). **The origin is validated either way**: scheme `http` or `https`, a host of letters, digits, `.` and `-` with an optional `:port`, nothing else — a request that forges a host that fails this makes the call an error (`500`), not an address that sends a customer somewhere unintended |
| `vnp_IpAddr` | the right-most `X-Forwarded-For` entry (`orders.Client.IP`), else the connection's address; `127.0.0.1` when neither parses as an IP | the request |
| `vnp_CreateDate` | now, `yyyyMMddHHmmss`, Vietnam time (fixed `+07:00`, no tzdata — S9) | the clock |
| `vnp_ExpireDate` | `placed_at + 30 minutes`, same format and zone — **the order's deadline, not now + 30** (SC-4: an order placed 10 minutes ago expires in 20) | the locked row |
| `vnp_SecureHash` | §4.3, **last** | |

If `vnp_ExpireDate` is not later than `vnp_CreateDate` once formatted (less than one second left), the address is **not built**: the refusal is "The time to pay this order has run out." An order with 30 seconds left is built, expiring at the same deadline (SC-4, PAY-001 boundary row). `vnp_BankCode` is not sent: the customer chooses on VNPay's page.

The address is `VNPAY_PAYMENT_URL` + `?` + the signed query string of §4.3 — the very string that was signed, so it cannot differ from it. **The secret is never in it**: the terminal code and the signature are (SC-38).

### 4.3 Signing and verifying (pure functions, one file)

- **The string that is signed** is made from **every `vnp_` parameter except `vnp_SecureHash` and `vnp_SecureHashType`**; parameters whose value is **empty are left out** (VNPay's own reference code does so, and the shop's address has none); names are sorted bytewise; each is `QueryEscape(name) + "=" + QueryEscape(value)` — Go's `url.QueryEscape` is `application/x-www-form-urlencoded`, a space is `+`; joined by `&`.
- **The signature** is the lower-case hex HMAC-SHA512 of that string, keyed with `VNPAY_HASH_SECRET`.
- **To verify**, the same string is built from what was received and compared with the received hash: it is hex-decoded (a wrong length or a non-hex character fails at once), then `hmac.Equal` — constant time, so case does not matter.
- **Known limit, written so nobody rediscovers it.** The sign string uses Go's escaping. VNPay's reference code (Java's `URLEncoder`) does not escape `*` and escapes `~`; the values the shop sends (digits, letters, spaces, `-`, a URL) and the values VNPay sends back (digits, letters, codes, the order info the shop wrote) contain none of the characters where they differ. If a real sandbox notice ever fails to verify, this function and the empty-value rule are the two places to look, and they are one file.
- **Test vectors.** Dev computes at least two signatures with an **independent** tool (`openssl dgst -sha512 -hmac "<secret>"` over a typed-out string) and writes them as literals in the test, so the algorithm cannot change without a test failing: one with a space, a `:` and a `/` in values (the return address), one with an empty value that must be left out.

### 4.4 How a case makes VNPay's notice (for Test)

A case that needs VNPay's call builds the parameters VNPay would send for an attempt (`vnp_TmnCode` = the run's terminal code, `vnp_Amount` = the order's total × 100, `vnp_BankCode`, `vnp_BankTranNo`, `vnp_CardType`, `vnp_OrderInfo`, `vnp_PayDate`, `vnp_ResponseCode`, `vnp_TransactionNo`, `vnp_TransactionStatus`, `vnp_TxnRef` = the attempt's reference), signs them by §4.3 with the sandbox secret the run holds, and sends `GET /api/v1/payments/vnpay/ipn?<the same string>&vnp_SecureHash=<hex>` with no session and no token. **A case that needs VNPay's real page is marked as such**, and is only fully observed with a public address registered in VNPay's portal. Without the keys every case that needs a built address or a verified notice is UNVERIFIED (the instrument it lacked: `VNPAY_TMN_CODE` and `VNPAY_HASH_SECRET`); only SC-8 and SC-9 are observed.

## 5. Packages, and the changes to code that already exists

### 5.1 Who changes what

| Package | Change |
|---|---|
| `internal/payments` (new) | The **only writer** of `payment_attempts` and `payment_notices`. Files (names are Dev's; each stays well under 300 lines): the provider (`Available`, `PaymentURL`), the address builder and the signature (pure, no I/O), the attempt transaction, the notice handler and its processing, the expiry loop, `Register`. It imports `orders` (the seam's types, `LockOrder`, `MarkPaid`, `Cancel`, `CheckPayable`, `Refusal`) and `httpx`; it never writes `orders`, `order_items` or `product_sizes` itself, and it imports no other package's store |
| `internal/orders` | **Imports nothing from `payments`** (the project's S2 rule: `payments` imports `orders`, so the reverse would be a cycle). New `settle.go` holds `PayWindow`, `LockOrder`, `MarkPaid`, `Cancel` and `CheckPayable` (§5.2, §5.3). `pay.go`: `decide` takes the clock and calls `CheckPayable`; one new sentence. `create.go`: the 503 gate (§3.2). Nothing in `Place` changes. The package still holds no HTTP client |
| `internal/stock` | One new function, `Return` (§5.4), called only by `orders.Cancel` |
| `internal/config` | Four settings and the `Secret` type (§7) |
| `internal/server` | `orders.Register` is given the provider; one `payments.Register` line (§7) |
| `cmd/api` | `go payments.RunExpiry(ctx, pool, log, time.Now)` beside `go auth.RunPurge` |
| `data`, `auth`, `admin`, `catalogue`, `ratelimit`, every migration already merged | **No change** |

### 5.2 Three functions `orders` exports, each inside the caller's transaction

```go
// PayWindow is how long an unpaid order may wait, counted from placed_at. The route's refusal,
// payment_attempts.expires_at, VNPay's vnp_ExpireDate and the expiry all use this one value.
const PayWindow = 30 * time.Minute

// LockOrder reads one order's row FOR UPDATE; ErrNotFound when there is none.
func LockOrder(ctx context.Context, tx pgx.Tx, id string) (Locked, error)
//   SELECT id, number, status, paid, total, placed_at, payment_method_id FROM orders WHERE id = $1 FOR UPDATE

// MarkPaid sets paid, and moves a "placed" order to "confirmed", once: changed=false means it was
// already paid. It is the only statement in the project that sets orders.paid for an online order.
func MarkPaid(ctx context.Context, tx pgx.Tx, id string) (changed bool, err error)
//   UPDATE orders SET paid = true, status = CASE WHEN status = 'placed' THEN 'confirmed' ELSE status END
//   WHERE id = $1 AND NOT paid

// Cancel moves one order to "cancelled" and puts its pairs back on the shelf, ONCE. changed=false
// means the order was not in a state this cancel may act on, and nothing was returned.
func Cancel(ctx context.Context, tx pgx.Tx, id string, rule CancelRule) (changed bool, err error)
```

**`Cancel` is the one single-cancel rule**, written here because this story's expiry is its first caller and the dealer's cancel (items 10 and 17) must be the same statement, so that **a pair is never returned twice**:

```sql
UPDATE orders SET status = 'cancelled'
WHERE id = $1 AND status IN ('placed','confirmed','packed')
  -- the 30-minute rule adds these four, and only it:
  AND NOT paid AND NOT is_sample AND payment_method_id = ANY($2) AND placed_at <= $3
RETURNING id
```

Zero rows: return `changed=false`, touch nothing (the order was already cancelled, paid, shipped, a sample, too young, or another run won). One row: read the pairs — `SELECT product_id, size, sum(qty)::integer FROM order_items WHERE order_id = $1 GROUP BY product_id, size ORDER BY product_id, size` — and call `stock.Return` with them **in the same transaction**. `CancelRule` is the zero value for a dealer's cancel and an expiry value (`PlacedBefore = now − PayWindow`, `Ways = thirtyMinuteWays`) for the 30-minute rule. Because the status test is in the `UPDATE`'s own `WHERE`, two cancels of one order at the same moment serialise on the row: the second re-reads it after the first commits, finds it `cancelled`, changes zero rows and returns nothing (SC-23). A shoe's sold-out list, last-pairs list and "pairs in stock" are the view `product_stock`, so they are right the moment the counts are (ERD §3).

### 5.3 One function holds the state rules

```go
// PayState is what the rules read of an order; Ref (the route's row) and Locked (the provider's) both give one.
type PayState struct{ Status, Payment string; Paid bool; PlacedAt time.Time }

// CheckPayable returns nil, or the *Refusal of the FIRST rule that fails, in this order:
//   cancelled → paid → a way to pay that is not offered → now >= PlacedAt + PayWindow
func CheckPayable(s PayState, now time.Time) *Refusal
```

`decide(ref, providers, now)` is `CheckPayable` followed by "no provider registered" and "not `Available()`" (the project's S6 table, one case added, as S6 §12.4 said S7 would). The provider's step 2 (§4.2) calls the same function with the locked row, so a refusal found under the lock has the **same sentence** the route would have given. The two sentences this story adds are constants beside S6's four: "The time to pay this order has run out." (`CheckPayable`) and, owned by `payments` because only it counts attempts, "This order has had too many payment attempts. Please place a new order."

### 5.4 `stock.Return`

```go
// Return puts pairs back, one statement per line in (product, size) order, in the caller's transaction.
func Return(ctx context.Context, tx Tx, lines []Line) error
//   UPDATE product_sizes SET pairs = pairs + $3 WHERE product_id = $1 AND size = $2
```

Unlike `Take`, **zero rows is not an error**: a size the dealer took off the shoe since the order was placed has no count to return to (checkout ERD §2.1 lifecycle — that is correct, the size no longer exists). Lines are sorted, so two cancels never lock the same two counts in opposite orders. `stock_test.go` gains: two lines of one shoe, a line for a size that no longer exists, a sort check.

## 6. The expiry — an unpaid order is given up after 30 minutes (PAY-003)

`payments.RunExpiry(ctx, db, log, now func() time.Time)` runs in the API process, started by `main` after the migrations.

| Aspect | Rule |
|---|---|
| Cadence | **Once at start-up, then every 30 seconds** until `ctx` is cancelled. So an order is cancelled within 31 minutes of being placed (30:00 plus at most one tick), and an order that fell due while the server was down is cancelled at boot. An order exactly 30 minutes old is cancelled by the next run (`placed_at <= now − PayWindow`) |
| Candidates | `SELECT id FROM orders WHERE NOT paid AND NOT is_sample AND status IN ('placed','confirmed','packed') AND payment_method_id = ANY($1) AND placed_at <= $2 ORDER BY placed_at LIMIT 100` — the partial index's predicate, repeated literally, so `idx_orders_unpaid_open` serves it. `$1` is `thirtyMinuteWays = {"vnpay"}` — a constant in code, as "offered" is (order-page ERD E2): MoMo joins it with PAY-021, and a bank transfer is never in it (its 24-hour rule is PAY-020's). `$2 = now − PayWindow`, the **application's clock** (ERD E6) |
| Per candidate | **One transaction each**: `SET LOCAL lock_timeout = '3s'`, `orders.Cancel(ctx, tx, id, expiry rule)`, commit. The candidate list is read outside any lock, so the conditions are **judged again under the row lock** by the `UPDATE`'s own `WHERE` (§5.2): an order paid, cancelled by the dealer, shipped or moved by anyone since the list was read is left exactly as it is, and its pairs are not touched |
| Left alone | An order that is paid; already cancelled (by the dealer at minute 5, say — its pairs were returned then, and are **not** returned again at minute 31); `shipped` or `delivered` while unpaid (the dealer cannot cancel a shipped order, so the rule does not either: it stays as it is, unpaid); one of the 14 samples `DOI-24804` to `DOI-24817` (`is_sample`, so the dashboard's figures hold; an order placed through checkout is never made one); one younger than 30 minutes. A `confirmed` or `packed` unpaid order **is** cancelled (SC-22) |
| A batch | At most 100 per query; the run repeats while a query is full, and **stops after 25 seconds** whatever is left (the rest is the next tick's). One order failing (a lock timeout, a database error) is logged with its id and the run goes on to the next |
| A panic | Recovered per run and logged; the loop keeps ticking |
| Two runs at once | Cannot happen in one process (one loop); in two processes the row lock and the `UPDATE`'s own `WHERE` make the second change nothing (SC-23). No advisory lock is needed |
| A notice at the same moment | Both serialise on the order's row. **If the cancel wins**, the order is `cancelled` with its pairs returned once, and the notice then finds it unpaid and marks it `paid` while it stays `cancelled` (SC-16): the dealer refunds it. **If the notice wins**, the order is paid and the cancel's `WHERE NOT paid` changes zero rows: nothing is returned. In neither case are pairs returned twice or taken again |
| Logging | One `Info` line per order given up (the order's id and number — not secret); an `Error` per failure; **nothing** when nothing is due, so a quiet shop's log has no tick lines |
| Shutdown | `ctx` is cancelled; an in-flight transaction is rolled back and the next boot finds it still due |

The result page and the order page need nothing from the expiry but its effect: the order reads `cancelled`, "Pay now" answers `409` (SC-25), and the cancelled view is the one the order page already draws.

## 7. Wiring and settings

```go
// config.Config gains (§4.1):
VNPayTmnCode, VNPayPaymentURL, PublicBaseURL string
VNPayHashSecret Secret                    // prints as [redacted]; read only inside payments' signing function

// server.New — the provider is registered even with no keys, so the answer is 503, never "not available yet":
vn := payments.NewVNPay(payments.VNPayConfig{TmnCode: cfg.VNPayTmnCode, HashSecret: cfg.VNPayHashSecret,
        PaymentURL: cfg.VNPayPaymentURL, PublicBaseURL: cfg.PublicBaseURL}, pool, log, nil /* time.Now */)
orders.Register(mux, pool, log, orders.Options{Providers: map[string]orders.Provider{"vnpay": vn}})
payments.Register(mux, vn, log)           // GET /v1/payments/vnpay/ipn
```

- **`config.Load`**: `VNPAY_TMN_CODE`, `VNPAY_HASH_SECRET`, `VNPAY_PAYMENT_URL`, `PUBLIC_BASE_URL` are read once, trimmed. A **malformed** `VNPAY_PAYMENT_URL` or `PUBLIC_BASE_URL` (§4.1) is an error that stops the start-up and names the key, never the value; a **missing** key never does. `config_test.go` gains the cases; no error text or `%+v` of the config holds the secret.
- **`.env.example`** (`code/backend/`): the two existing lines stay, and two are added with **no value**: `VNPAY_PAYMENT_URL=` ("empty means VNPay's sandbox page; going live changes this value, not the code") and `PUBLIC_BASE_URL=` ("the shop's public origin, no path; leave empty when the edge proxy forwards `X-Forwarded-Proto` and `X-Forwarded-Host`"). Nothing else in `.env.example`, compose or the deploy files changes: the owner's values reach the backend through the console (`run_project`) and `.env.deploy` as the two keys already do.
- **The owner's side** (already requested in the console: `request_secret` ×2, `request_setup` ×1): register `<public origin>/api/v1/payments/vnpay/ipn` in VNPay's merchant portal. Until that is done on a publicly reachable deployment VNPay cannot confirm a payment, and a real sandbox payment stays "confirming" on the result page — the shop is not at fault and no code works around it.
- **Never in a log**: the access log records the route and not the query; the notice's own log line is §3.4's. A test pays, sends a bad-signature notice and a 503, with a distinctive secret, and asserts the secret is in **no** captured log line, response body or header (SC-38).

## 8. The frontend at the swap (the backend PR's frontend half)

All of it is in `src/lib/online-payment-with-vnpay.ts`, its test, and the deletion of the mock. `PaymentReturn.tsx` and `PaymentResult.tsx` change by one expression (row 3).

| # | Change | Why |
|---|---|---|
| 1 | `transport.read(id, token)` becomes `api.get("orders/" + encodeURIComponent(id), undefined, { "X-Order-Token": token })` and `transport.pay(id, token)` becomes `api.post("orders/" + encodeURIComponent(id) + "/payment", undefined, { "X-Order-Token": token })` — **no body, sent once, never retried** (`api.ts` retries a failed GET once and a POST never). An `ApiError` becomes `{status: e.status, body: {error: e.code}}`; a network failure (`status` 0, code `network`) and anything else thrown becomes `{status: 0, body: undefined}`. The token goes **only** in that header, never in an address, a query or a log, and only the token of the order the reference names is ever sent | §2 row 2; S6 §7 rows 1, 2, 4 |
| 2 | `readOrder`: **`no-match` only for status 404 and `error === "not_found"`**; a resolved body that is not an order, and every other failure (any other 404, 401, 5xx, network), is `failed`, drawn as the "could not check" state | §2 row 3 (the one departure from the mock) |
| 3 | `usePaymentReturn` returns `order: null` whenever the phase is `no-match`, so a held token the shop answers `not_found` for draws the one cannot-match view with the "Track an order" pill and nothing about any order | TL's UI review, should-fix 1 |
| 4 | Delete `src/lib/mock/online-payment-with-vnpay.ts` and the `[…storedOrders(), ...demoHeld]` line of `heldFor` (it reads `storedOrders()` of `src/lib/orders.ts` only — the token store's own reader; no second store, no new storage key) | The mock is deleted with the story that replaces it; the CI wiring warning for a mock still imported goes quiet |
| 5 | `src/test/payment-return.test.tsx`: the spies on `transport` become `vi.mock("../lib/api")` keeping the **real `ApiError`**. Keep the 14 cases' assertions; add: the header is sent with the held token of exactly the reference's order and never in the address; **no token held → no request at all**; a held token and a `not_found` 404 → no-match with no "View your order"; a 404 with another code → "could not check"; a network failure → "could not check"; the `POST` has no body and is sent once; a 409 re-reads the order | SC-30, SC-32, SC-33 |
| 6 | `api.ts` is **not edited here**: the call uses the one optional trailing `headers` parameter that "Order page and Pay now" adds (`decision.api_client_headers`, PR #65). No second fetch wrapper | §11 gate |

**Not edited by this story** (SC-37): `Checkout.tsx`, `Order.tsx`, `PayNow.tsx`, `src/lib/orders.ts`, `src/lib/order-page-and-pay-now.ts`, `theme.css`, `index.css`, `App.tsx` (its import and route line are already in PR #75), `content.json` (the `pages.payment_return.*` keys are already in PR #75), `design/pages.json` (its line too), `package.json`. **No mock module is added**; there is none for this story after the swap.

## 9. What Dev's backend PR carries as tests

Database cases run in CI (Postgres via `TEST_DATABASE_URL`) and **skip without it**: their first real run is CI, and Verify must confirm they ran and did not skip. The pure ones run everywhere.

| Test | Proves |
|---|---|
| **Signature**: the two independent vectors of §4.3; names sorted; an empty value left out; a repeated `vnp_` name; a hash one character off, with another secret, absent, 127 or 129 characters, non-hex, upper-case (accepted); a foreign `vnp_TmnCode` | SC-3, SC-13 |
| **Address**: every parameter of §4.2 exact for a 3.140.000 ₫ order (`vnp_Amount=314000000`, `DOI-24818-1` then `-2`); the amount comes from the locked row, not from `Payable` and not from `?amount=`; `vnp_ExpireDate` = `placed_at + 30 minutes` in `+07:00` for an order 10 minutes old (20 minutes from now) and for one 29 minutes 30 seconds old; one with under a second left is the "run out" refusal; `PUBLIC_BASE_URL` over the forwarded host over `Host`; a forged host that fails the origin rule is an error; `VNPAY_PAYMENT_URL` is the base of every address; the secret is in no address; the reference is at most 100 characters; a total whose `×100` overflows is an error | SC-1, SC-3, SC-4, SC-39 |
| **`CheckPayable` and `decide`** over every status × paid × way to pay × provider (none, closed, available) × clock (29:59, 30:00, 31:00): the order of the rules, the six sentences, the provider never asked for a refused order | SC-4, SC-6 |
| **Attempts, against PostgreSQL**: two asks → `-1`, `-2`, two rows; **20 simultaneous asks → 20 distinct references `-1`…`-20`**; the 51st ask → `409` and no row; an order 31 minutes old → `409`, no row; a cancelled, a paid and a `cod` sample order → `409`, no row; a success notice for the **first** reference still pays the order | SC-2, SC-4, SC-6 |
| **Keys missing** (a provider built with no keys): `Available()` false; `GET /v1/payment-options` lists `vnpay` with `is_available` false; `POST /v1/orders` with a valid body → `503` **after** a 422 and before any write (no order, `counters` unchanged, no pair taken, a counting fake of `DB` sees no write); "Pay now" → `503` after `404` and `409`; the notice → `97` to everything; `/healthz` 200 | SC-8, SC-9 |
| **Placing with keys**: `201` with a non-empty `payment_url` whose `vnp_Amount` is the order's `total × 100`; a `payment`of `cod`, `bank` or `momo` → `422` naming `payment`, no order; an attempt that cannot be stored (a `DB` that fails on the insert) → `201`, `payment_url` `""`, the order saved, unpaid, `placed`, its pairs taken | SC-5, SC-7, SC-10 |
| **The notice's table test** over a fake `DB`: every row of §3.4 — `00`, `01`, `02`, `04`, `97`, `99` — as HTTP 200 with a body of exactly `RspCode` and `Message`, never the envelope; the order of the checks (a notice wrong in two ways gets the first code); an extra `vnp_` field signed with the rest is accepted; no query, an 8 KiB-plus query and a malformed escape are `97`; a notice with the keys unset is `97`; a failing store → `99`, **nothing written**, and the same notice sent again is processed normally | SC-13, SC-14, SC-17 |
| **The notice against PostgreSQL**: a success pays the order, `placed → confirmed` (and `packed` stays `packed`), keeps the transaction number, bank, `amount ÷ 100`, codes and pay time; the same notice again → `02`; **two at once and, in a second run, twenty** → exactly one `00`, the rest `02`, one success row; a failure (`24`, or status not `00`) → `00`, order unpaid, one failure row, a **second identical failure → still `00`, still one row**; a success for the next attempt afterwards pays; a success for a **cancelled** order → `00`, `paid` true, status `cancelled`, **the shelf unchanged**; a wrong amount `04`, an unknown reference `01`; 100 runs of "bad signature / unknown reference / wrong amount" change no order and write no row; 100 runs of two identical notices pay once | SC-11 to SC-16, SC-19 |
| **`orders.Cancel` and `stock.Return`**: a 2-pair order of size `38` returns 2 and the view's lists and total agree; an order of two shoes returns both; a size taken off the shoe meanwhile is skipped without an error; **two `Cancel`s at once return the pairs once**; the dealer's rule (no expiry conditions) and the expiry's rule differ exactly by the four extra conditions | SC-20, SC-23 |
| **The expiry**: a loop test with a fake clock (runs at start-up, then every 30 s; stops on `ctx`; a panic in one run does not stop the next); against PostgreSQL: a 31-minute `vnpay` order `placed` is cancelled and its pairs returned; an order at **exactly** 30 minutes is cancelled by the next run; **left unchanged**: 29 minutes, paid at 2 hours, `shipped` and `delivered` unpaid at 2 hours (status, paid, pairs), a **sample** unpaid and 2 hours old (and the pairs of the 14), an order the dealer cancelled at minute 5 (its pairs are not returned again at minute 31); **cancelled**: `confirmed` and `packed` unpaid at 2 hours; **a notice and a run at the same moment**, repeated: always exactly one of "paid and untouched, pairs still taken" or "cancelled once, pairs returned once, then paid with status `cancelled`" | SC-20 to SC-25 |
| **Settings and secrecy**: `config.Load` reads the four keys trimmed, defaults the payment page, refuses a malformed `VNPAY_PAYMENT_URL` or `PUBLIC_BASE_URL` naming the key and not the value, starts with a key missing; `%v` and `%+v` of the config and every error hold no secret; **a distinctive secret is in no captured log line, response body or header** after a payment, a bad-signature notice and a 503; `.env.example` names the four keys with no value | SC-38, SC-39 |
| **Route table**: `GET /v1/payments/vnpay` is still the way-to-pay record, `GET /v1/payments/vnpay/ipn` is the notice, a `POST` to it is the `404` envelope, and the notice's route pattern is the only new one | §1 |
| **Schema** (ERD §8): the three files apply in order; each constraint refuses what it is for | ERD §2 |

## 10. Decisions, and conflicts with merged ones

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| S1 | Is the provider registered when the keys are missing? | **Always.** `Available()` is the key check. A shop with no keys answers `503 payment_unavailable` (SC-8); a shop with no provider would answer `409 "not available yet"` and `is_available: true`, which is the pre-story behaviour the owner's decision ends | Registering only with both keys: the route would answer a 409 for a missing key, Checkout would not draw its alert, and `GET /v1/payment-options` would say `true` |
| S2 | Where is the 503 for placing? | In `create`, after the field checks, before `place` (§3.2); it also answers a replay while the keys are missing | Inside `Place`: a transaction opened and a counter touched to say something a setting already says. After the transaction: an order with pairs taken that can never be paid |
| S3 | Which clock decides "30 minutes"? | **The application's**, passed in everywhere (ERD E6); `PayWindow` is one constant | `now()` in SQL: skew between two hosts, and untestable without sleeping |
| S4 | What does the route say when the 30 minutes have run out but the check has not yet cancelled the order? | `409 conflict`, "The time to pay this order has run out." — a new sentence under the existing code. The same for a sample VNPay order, which the check never cancels (VNPay would refuse an expired address) | A new code `expired` or HTTP `410`: the catalog is closed and `conflict` already means "a state rule"; the page branches on no sentence |
| S5 | Bound on attempts | 50 per order, a `409` (ERD E9) | A rate limit; no bound |
| S6 | A failure notice for an order already paid | `02`, nothing written — the order of the checks is the contract (SC-14: "…then already paid") | Recording it with `00`: the order's story is closed; a stale failure adds nothing |
| S7 | Statement order of a success notice | `MarkPaid` first, then the notice (ERD E4) | Notice first (project text) |
| S8 | What exactly is verified | §3.4 rows 0–1: keys set, ≤ 8 KiB, parses, no repeated `vnp_` name, 128 hex, constant time, our `vnp_TmnCode`. A repeated name is refused because "the first value" and "the last value" are two different truths and the signature covers neither | Accepting the first of a repeated name, as `getParameter` would: two readers could disagree about which one was signed |
| S9 | Who may write which table, with `payments` on top of `orders` | `payments` writes only its two tables and calls `orders.LockOrder`, `MarkPaid`, `Cancel`; `Cancel` calls `stock.Return`; **`orders` never imports `payments`** (project S2) | `payments` writing `orders` directly: the order's `status` would have two writers and the single-cancel rule two copies |
| S10 | The expiry's mechanism | One transaction **per order**, the candidate list read unlocked and judged again by the `UPDATE`'s own `WHERE`; every 30 seconds and at start-up (§6) | One batch under `FOR UPDATE SKIP LOCKED` every 60 s (project §4): a failing order would roll back up to 99 others, and 60 s plus processing can pass 31 minutes. Cost: with a second instance, one waits up to `lock_timeout` on an order the other is cancelling, then changes zero rows |
| S11 | Does the browser's return get a server endpoint that verifies the return address's signature? | **No.** `GET /v1/orders/{id}` with the order's token is the truth the page needs; the page never believes the address about money or status | A `…/return` endpoint that verifies the signature and reports: a second place that could be made to say "paid", and the page would then be believing something other than the order |
| S12 | IP allow-list or rate limit on the notice address | **Neither** (§3.4) | An allow-list of VNPay's addresses: the proxy chain hides them and a list goes stale. A limit: refuses VNPay's own retries |
| S13 | How does the secret stay out of logs | A type that prints `[redacted]`, read as a string only inside the signing function, plus the captured-log test (§9) | Discipline: one `%+v` of a config in an error is enough to leak it |
| S14 | A forged `Host` / `X-Forwarded-Host` | The origin is validated (§4.2); `PUBLIC_BASE_URL` is the way to pin it in a deployment | Trusting the header: a caller could have the return address point anywhere it likes (the answer goes only to the caller, but the rule costs one regexp) |
| S15 | A replay of a used placing key | Builds a **new** attempt when the order is still payable (project §2.6); it counts toward the 50 | Returning the old address: it may be expired or its page already used |
| S16 | The 404 of the result page | `no-match` only for the API's own `not_found` (§2) | The mock's "any 404" |
| S17 | A malformed optional setting | Stops the start-up and names the key (a typo in `VNPAY_PAYMENT_URL` would otherwise send customers to the wrong page for ever); **a missing key never does** (SC-8) | Falling back to the sandbox silently in production |

### Conflicts with merged decisions

1. **Project §4 (asynchronous work) puts the expiry in the `orders` package, every 60 s, as one batch with `FOR UPDATE SKIP LOCKED`; the project's ownership table says `payments` calls `orders.Cancel`.** Both cannot hold. Resolved: `payments.RunExpiry` (PAY-003 is the payments module's, and its set of ways grows with PAY-021 and PAY-020) calls `orders.Cancel`, which `orders` owns; the cadence is 30 s and the mechanism is S10. The observable contract — within 31 minutes, once, idempotent under row locks — is the project's and the SRS's.
2. **Project §3.17 step 6 and ERD §3.17** (statement order of the success path; failed notices "may repeat"): ERD E4 and E3. Answers and ownership unchanged.
3. **S6 §3.1** "the route writes nothing, opens no transaction and takes no lock": true until a provider exists, and S6 §12.4 said this story would change it. On success it now opens one transaction, locks the order's row and writes one attempt. Every refusal still writes nothing.
4. **S6 §5 `decide(ref, providers)`** gains the clock and one case, as S6 §12.4 said; it stays pure. S6 listed four 409 sentences (Q4); the route now has **six** (cancelled, already paid, cannot be paid online, run out, too many attempts, and — only in a shop with no provider — "not available yet").
5. **S5 §4** "nothing in `Place` changes": true. The gate is in `create`.
6. **Settings (`_base.md` §5.2)** — "none is a default in the source": the one exception is `VNPAY_PAYMENT_URL`'s sandbox default, which the same table already states. `PUBLIC_BASE_URL` and `VNPAY_PAYMENT_URL` are validated at start-up (S17), which the table does not say; it adds a refusal for a malformed value and none for a missing one.

I read `services/_base.md` §2.1–2.8, §3.5, §3.10, §3.11, §3.13, §3.17, §4, §5.1, §5.2, §7; `services/checkout-and-placing-an-order.md` §3.1, §4, §5, §7; `services/order-page-and-pay-now.md` §3–§12; `docs/payments/SRS.md` (all five requirements); `docs/orders/SRS.md` ORD-006, ORD-007; and the code on `main` (`internal/orders/{orders,create,place,options,read}.go`, `internal/stock/stock.go`, `internal/config`, `cmd/api/main.go`, `internal/server/server.go`, the migrations) and the diff of PR #65.

## 11. Ordering with the open stories, and hand-over

| Matter | Rule |
|---|---|
| **Gate: PR #65 ("Order page and Pay now", backend) must be on `main` first** | This story's backend edits `decide` and `payURL` (it adds the clock and a case) and the frontend swap uses the `headers` parameter of `api.ts` that #65 adds. Nothing here can be cut earlier. PR #50 (checkout) is merged |
| PR #69 ("Customer accounts", backend) | Edits `authorize` and `place`; this story does not touch either. Whichever of #65 and #69 merges second adapts (S6 §12.4); this story is cut after both |
| UI PR #75 | Merges first with its mock (approved, zero Blocking). The backend PR is cut from `main` after it, swaps the client (§8) and deletes the mock. If #75 is still open when the backend PR is built, re-apply its own files without the mock, as #65 did for #41, and close #75 as superseded |
| "Dealer handles orders" (item 17) and "Manage … orders" (item 10) | Read an order's VNPay transaction number by ERD access pattern 6; cancel through `orders.Cancel` (§5.2). **Open for item 17's design:** `Cancel` returns an order's pairs whatever the order is, and the 14 sample orders never took a pair from `product_sizes` (S5's backfill counts the shelf as it was seeded). A dealer cancelling a sample order would put a pair on the shelf that was never taken; the expiry never reaches a sample, so this story is unaffected, and the guard (return no pairs when `is_sample`) belongs to whichever story first lets a dealer cancel one |
| "More ways to pay" (PAY-020 to PAY-022) | Adds MoMo to `thirtyMinuteWays`, a 24-hour way for bank transfers in the same loop, `provider = 'momo'` attempts and its own notice route, the QR block on the order page. **No migration for the index**, no change to the notice's checks |
| The owner | Supplies `VNPAY_TMN_CODE` and `VNPAY_HASH_SECRET` (sandbox first, from a sandbox merchant at sandbox.vnpayment.vn) and registers `<public origin>/api/v1/payments/vnpay/ipn` in VNPay's merchant portal (declared in the console already). Until both are in, the shop cannot complete any order: checkout shows "Online payment is not available right now. Please try again later." and placing answers `503` |
| Test lead | The keys are a precondition of the run (story §6); every criterion that needs an address or a verified notice is UNVERIFIED without them, and only SC-8 and SC-9 are observed. A notice is signed by the case itself (§4.4). Age an order with `UPDATE orders SET placed_at = …` and set a status directly (ERD §9); a 31-minute order is also unpayable (`409` before any attempt). The `vnpay` sample orders are unpaid or paid as seeded: a sample is never cancelled by the check and "Pay now" for an unpaid one is the `409` "run out". **No test-only route, header or knob exists**; the 6-second and 30-second cases of the result page are made by sending a signed notice at the right moment. DB-backed Go tests skip without `TEST_DATABASE_URL`: confirm they ran |
| Real VNPay | The signature and the empty-value rule (§4.3) are proven offline against vectors made by an independent tool, **not** against VNPay's own servers. A first sandbox payment on a public address is the one observation that closes that risk; if a real notice fails to verify (`97`), `payments`' one signing file is where to look |
| Dev, backend step | The three migrations (ERD §8); `payments` (new, §5.1); `orders/settle.go`, the 503 gate and the changes to `decide` (§3, §5); `stock.Return`; config, `server.New`, `main`, `.env.example` (§7); the swap (§8); the tests of §9. If anything in §3–§7 cannot be done as written, reply with the blocker rather than widening the contract |

## 12. Migration plan

**Three migrations, no data migration, no setting migration** — the plan, the order, the backward steps and the reason each is safe are in `docs/architecture/erd/online-payment-with-vnpay.md` §8. In short: two new empty tables (nothing to lock or rewrite) and one `CONCURRENTLY` index on `orders` (no write lock). **Forward**: the server applies them on boot, before listening, so the notice route and the expiry never run against a schema that is not there. **Backward**: revert the backend PR; the tables stay and the previous build neither reads nor writes them; the two `DROP TABLE` files exist for local resets only, because they delete the record a refund is made against. Deploying with the keys unset is safe: the shop starts, completes no order, and the notice answers `97`.

## 13. Where each criterion is enforced

| Criteria | Enforced by |
|---|---|
| SC-1, SC-2, SC-3, SC-4 | §4.2 (the locked row, the reference, the deadline), §4.3, §3.1 step 3 and `CheckPayable` |
| SC-5, SC-7, SC-10 | §3.2 (`payment_url`, the field checks, `Place` unchanged); S5's `Checkout.tsx` follows the address |
| SC-6 | §3.1 (the six sentences, the order of checks, S6's single 404); §8 row 1 (the page edits nothing) |
| SC-8, SC-9 | §3.2 gate, §3.1 step 4, §3.3, §3.4 row 0, §4.1; Checkout's own alert (S5) |
| SC-11 to SC-17 | §3.4 (rows 0–6), ERD §2.2 and §6, §5.2 `MarkPaid` |
| SC-18 | §3.4's first paragraph, S11: nothing but the notice writes `paid`; `POST …/payment` writes an attempt only |
| SC-19 | §3.4 (rows before the first write cost no database work; p95), §9 |
| SC-20 to SC-25 | §5.2 `Cancel`, §6, ERD §2.3 and §6 |
| SC-26 | §3.1 (a fresh reference on each press), §6 (the cancel), S6's page |
| SC-27 to SC-35 | PR #75 (the page and its tests); §8 rows 2, 3 and 5 for the 404 rule and the cannot-match view |
| SC-36 | The whole chain: §3.2, §3.4, §6 and the page |
| SC-37 | §8 "Not edited by this story" |
| SC-38, SC-39 | §4.1, §7 (`Secret`, `.env.example`), §9 "Settings and secrecy" |
