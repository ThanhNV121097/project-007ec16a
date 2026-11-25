# Service contracts — Order page and Pay now (plan item 6, "S6")

Module `orders` · Requirements ORD-007, ORD-006 items 4–6 and AC-12, ORD-009 AC-5 · Story `docs/orders/stories/order-page-and-pay-now.md` · UI PR #41 · Tables: `docs/architecture/erd/order-page-and-pay-now.md` (none changed).

This file defines **the one endpoint this story ships, the one function that decides who may open an order (shared with `GET /v1/orders/{id}`), the order of the route's refusals, and the frontend's swap from its mock**, and nothing else. It reuses the project's error envelope and closed error catalog (`services/_base.md` §2.3), its no-body and path rules (§2.5), and S5's `orders` package, provider seam and access seam (`services/checkout-and-placing-an-order.md` §3.2, §4, §5). It adds **no error code to the catalog**, no second list or error shape, no table and no setting. Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/orders/doi-24818/payment` reaches Go as `/v1/orders/doi-24818/payment`.

## 1. Endpoints

| # | Method and path | Auth | Traces to |
|---|---|---|---|
| 3.13 | `POST /v1/orders/{id}/payment` | the order's `X-Order-Token` (the linked customer's session joins it in the accounts story, through §4) | ORD-006 items 4–6 and AC-12; ORD-007 items 8–9 and AC-17 to AC-20 |
| 3.11 | `GET /v1/orders/{id}` | unchanged (S5 §3.2) | **No change of contract.** Its access step becomes the shared function of §4; its answers are byte-for-byte what S5 fixed |

The route is **one literal pattern**, `POST /v1/orders/{id}/payment`, mounted by `orders.Register` (S5 §5 — the package that owns `orders`) beside S5's four. It is the fifth route of that call and the only line `server.New` and `orders.Register` gain. Any other method on that path falls through to the server's catch-all and answers `404 not_found` (project contract §3, "no wildcard"); it is not given its own handler. **No new package, setting, secret or `.env.example` line.**

## 2. The mock, read and judged

The UI PR adds `src/lib/mock/order-page-and-pay-now.ts`, imported only by `src/lib/order-page-and-pay-now.ts`. It is unreviewed input; its shape is compared with the contract. **Where `_base.md` §2.8 / §3.11 / §3.13 and S5 already decided a shape, they win.**

| Mock | Contract | Judgment | Frontend change at the swap |
|---|---|---|---|
| `getOrder` answers `{status: 200, body: PublicOrder}`: `id`, `number`, `placed_at` (RFC 3339, `+07:00`), `status`, `items[]`, `total`, `payment`, `paid`, `delivery`, `carrier`, `tracking`, `eta`, `customer`, `city` | The public view of `_base.md` §2.8 with `items` (S5 §3.1 example `order`) | **Sound.** Names, types, units and "text is `""` when unset" match field for field. The real answer also carries `delivery_fee`, `product`, `size`, `qty`, which the page's type does not name and ignores | None |
| `items[]`: `product`, `name`, `colourway`, `image`, `size`, `qty`, `unit_price` (as sold) | Same (§2.8); `name`, `colourway`, `image` are read live from the shoe | **Sound.** `image` is `""` for a shoe that has none; the page already draws no picture and no empty box for that (SC-18) | None |
| Every answer is `{status, body}`, success or not; a network failure **throws** | `src/lib/api.ts` (S3 §5.1) resolves a 2xx with the parsed body and **throws `ApiError`** (`status`, `code`, `message`) for any other status and for a failure to connect (`status` 0, `code` `network`) | **The mock's shape is not the real client's.** It is hidden inside one file | `src/lib/order-page-and-pay-now.ts` only: its two calls become `api.ts` calls and `readOrder` / `requestPayment` read `ApiError` instead of `answer.status`. `Order.tsx` and every component are untouched (§7) |
| Any `404` is "not found" (`answer.status === 404`) | "Not found" is `404` **and** the API's code `not_found`. A `404` that is not the envelope (a gateway's own page, a mis-routed `/api`) reaches the client as `ApiError` 404 with code `internal` (S3 §5.1) | **Too loose, and the one departure that matters.** If any 404 counted, a broken proxy rule would tell every customer that their order does not exist, and SC-11 ("never the not-found view when the shop cannot be asked") would fail in production while passing against the mock | `readOrder` returns `missing` only for `ApiError` with `status === 404 && code === "not_found"`; every other failure is `failed` (§7, row 3) |
| The not-found body is `{error:"not_found", message:"We can't find that."}` | `{"error":"not_found","message":"Not found."}` (S5 §3.2) | The page never reads `message` | None |
| `postPayment` answers 409 `conflict` "Online payment is not available yet." for every found order, 404 for an unknown one | 404, 400, four different 409 sentences, 500; later 200 / 503 (§3, §5) | **Sound for what the page reads**: only the status code and `payment_url`. The page draws one failure line for all of them (SC-27) | None |
| `requestPayment` succeeds only on `200` with an `http(s)://` `payment_url` | Same | Sound; this is what makes SC-29 true | None |
| `doi-slow`, `doi-error`, `doi-offline`, `doi-flaky`, and a read of `localStorage["doi:orders"]` | — | Mock-only controls. **No delay, fault or sleep route, header or query parameter is added to the backend** (project decision S4 of the product-page design): the 2-second answer is a gateway- or browser-side hold of one URL, the 500 is `set_db_available`, the network failure is a stopped container | Go with the mock |
| The mock draws orders placed by the mock checkout from `doi:orders` | The order placed through `POST /v1/orders` is read with the token S5's store holds | Goes with the mock | Deleted with it |
| The whole page reads `delivery` and `payments` names through the collection hooks | Unchanged: `GET /v1/delivery`, `GET /v1/payments` (S3) serve them | **Sound, and the contract wins**: `_base.md` §2.8 fixes `delivery` and `payment` as **ids**, so the names come from the records; nothing is embedded in the order | None. A failed names read leaves the two rows out and the order still drawn (`OrderDetails` skips a row whose record is missing); that is the page's own, already-approved degradation |

## 3. Contract — `POST /v1/orders/{id}/payment` (project §3.13)

**Purpose** — "Pay now": ask the shop for the address of the payment page of an unpaid order. In this story **every order that can be reached is refused** (state, or "not available yet"): the route's job is to refuse correctly and to be the exact place where the VNPay story plugs in. **Auth** — the header `X-Order-Token: <token>` that opens **this** order, the same credential and the same check as `GET /v1/orders/{id}` (§4); never in an address or a body. No dealer path: a dealer session without the token is `404` (§3.13). **Request** — **no body, no query parameter that matters** (a query string is ignored). The browser sends `credentials: "include"` and, when it holds a token for this id, the header; with none held it sends the call without the header (§7). **Headers on every answer** — `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`.

### 3.1 Order of checks

Each step runs only when the earlier ones passed. A request is answered by **the first step that refuses it**.

| # | Check | Refusal | Notes |
|---|---|---|---|
| 1 | **Access.** `{id}` matches `^[a-z0-9][a-z0-9._-]*$` and is at most 100 characters; the header is present, at most 128 characters and only `[A-Za-z0-9_-]`; **and** one statement finds the order with a token that opens it (§4) | `404 not_found`, `{"error":"not_found","message":"Not found."}` | A bad id or a missing / malformed token is answered **with no query made**. A missing order, another order's token and a seeded order that no token opens run the same statement and answer the same bytes. The body is the **same Go function's output** as `GET /v1/orders/{id}`'s 404 (SC-1) |
| 2 | **A body was sent.** Read at most **one byte** of the request body (`io.ReadFull` over `io.LimitReader(r.Body, 1)`); a byte read, or a read error other than `io.EOF`, is a body — whatever its content type or declared length | `400 bad_request`, message "This call takes no body.", `errors: {"body": "This call takes no request body."}` | After step 1, so a stranger who sends a body still learns nothing but `404` (SC-6). No content-type check and no `415` on this route: a call with no body has no content type (decision P5). Nothing of the body is parsed, stored or logged |
| 3 | **State** (§5): cancelled → paid → way to pay not one the shop completes | `409 conflict`, one of three **different** sentences | Reads only the row step 1 already returned: no second statement |
| 4 | **Availability** (§5): a provider is registered for the order's way to pay | No provider → `409 conflict` "Online payment is not available yet." A provider that is not `Available()` → `503 payment_unavailable` (S7 only) | Until S7 registers `vnpay` the first outcome is the answer for **every** order that gets this far (SC-5) |
| 5 | **The address.** The provider's `PaymentURL` (S5 §4) is asked, with a 3-second limit | `200 {"payment_url": "<address>"}`. An error, a timeout or an empty address is `500 internal` | Unreachable in this story's production build (no provider is registered); reached by the two fakes of S5's SC-31 in tests. S7 owns what happens inside `PaymentURL` (the attempt row, the reference, the 30-minute deadline) |

**The route writes nothing, opens no transaction and takes no lock in this story.** A press leaves the order, its tokens and every other row exactly as they were. It makes **no outbound call**: `orders` imports no HTTP client (a review rule, SC-5).

### 3.2 Success — `200`

```json
{ "payment_url": "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?vnp_Amount=…&vnp_SecureHash=…" }
```

`payment_url` is the string the provider returned, unchanged. **No code in this story can produce this answer**: it needs a registered provider, which the VNPay story adds. It is specified here so that story changes no line of the route, the handler or the page. The browser follows the address only when it is an `http://` or `https://` one (the page's rule, SC-27).

### 3.3 Errors

| Code | HTTP | Trigger | Body |
|---|---|---|---|
| `not_found` | 404 | Step 1: bad id, no token, malformed or over-long token, a token that does not open this order, an order that does not exist | `{"error":"not_found","message":"Not found."}` — byte-for-byte the body of `GET /v1/orders/{id}` for the same cases |
| `bad_request` | 400 | Step 2: the request carried a body | `{"error":"bad_request","message":"This call takes no body.","errors":{"body":"This call takes no request body."}}` |
| `conflict` | 409 | The order was **cancelled** (paid or not) | message "This order was cancelled, so it cannot be paid." |
| `conflict` | 409 | The order is **already paid** (and not cancelled) | message "This order is already paid." |
| `conflict` | 409 | Unpaid, not cancelled, and its way to pay is **not an online one the shop completes** (`cod`, `bank`, `momo`, any other id) | message "This order cannot be paid online." |
| `conflict` | 409 | Unpaid, not cancelled, an offered way to pay, **no provider registered** (until S7) | message "Online payment is not available yet." — **exactly** these words (SC-5, ORD-006 AC-12) |
| `payment_unavailable` | 503 | **S7 only**: a provider is registered and its keys are not set (`Available()` is false). Message owned by that story; this story's fake-driven test checks the status and the code only | `{"error":"payment_unavailable","message":"…"}` |
| `internal` | 500 | The access statement fails or times out; a provider returns an error, takes over 3 s, or returns an empty address. Generic body, `{"error":"internal","message":"Something went wrong. Please try again."}`: no SQL, no stack, no id, no token, nobody's data (SC-6). The cause is logged with the request id and the order id — **never the token** | generic |

The four 409 sentences are four different sentences (SC-2 to SC-5). All carry the one code `conflict` because the catalog is closed (decision P4): the page branches on none of them, and a person reads the message. The 409 bodies have **no** `errors` key (it exists only on `bad_request`, `invalid`, `stock_conflict` and the sign-up `conflict`).

### 3.4 Notes

- **Idempotent and safe to repeat.** With nothing written, a second or third press (SC-5) answers what the first did. The project's idempotency table (`_base.md` §2.6) says this route has no key; S7 changes its meaning (each call a new attempt), not its shape.
- **No rate limit.** A token is 192 bits and the only credential; a flood of valid-token calls costs one single-row indexed read each. S7 may add one with the attempt row; none exists in the contract.
- **CSRF.** The credential is a header a cross-site form cannot set; when the cookie path arrives (accounts story) `SameSite=Lax` keeps the cookie off a cross-site `POST` (`_base.md` §2.2).
- **Logging.** The access log records the method, the route (`/v1/orders/doi-24818/payment`) and the status; never a header or a body. The order id is not a secret. A `500` logs `order`, the request id and the error; a `409` logs nothing beyond the access line.
- **CORS.** `X-Order-Token` is already reflected by the CORS middleware (`_base.md` §2.1); a same-origin deployment sends no preflight.

## 4. The one function that decides who may open an order

The story requires the access check to be **written once**, so the dealer and accounts stories extend one function and cannot make `GET` and `POST` disagree. It lives in `internal/orders` (S5 §5's package) and both routes call it.

```go
// Access says who is asking. Today only a token; the dealer and accounts stories add fields.
type Access struct{ Token string }

type mode int
const ( modeRead mode = iota; modePay )   // today the two behave alike; the dealer branch is modeRead only

// Ref is the order's row exactly as the access statement returns it: the public columns, never
// phone, email, address, note, is_sample, idempotency_key or account_id.
type Ref struct {
    ID, Number, Status, Delivery, Payment, Carrier, Tracking, Customer, City string
    PlacedAt time.Time; Paid bool; Total, DeliveryFee int64; ETA *time.Time
}

var ErrNotFound = errors.New("order not found")           // → 404 not_found, one body, for every cause

func authorize(ctx context.Context, q querier, id string, a Access, m mode) (Ref, error)
```

`querier` is the one-method interface `QueryRow(ctx, sql string, args ...any) pgx.Row`, satisfied by the pool and by a transaction, so a unit test can hand it a fake that fails (SC-6's `500`) without stopping a database.

**Steps, in order** — (1) `id` must match `^[a-z0-9][a-z0-9._-]*$` and be at most 100 characters, else `ErrNotFound` with no query; (2) the token must be non-empty, at most 128 characters and only `[A-Za-z0-9_-]`, else `ErrNotFound` with no query; (3) the token is hashed **in Go** with SHA-256 (`sha256.Sum256`, 32 bytes) and the hash is passed as a `bytea` parameter — the token itself is never a SQL parameter, never logged and never compared; (4) one statement:

```sql
SELECT o.id, o.number, o.placed_at, o.status, o.paid, o.total, o.delivery_fee,
       o.delivery_option_id, o.payment_method_id, o.carrier, o.tracking, o.eta,
       o.customer, o.city
FROM orders o
WHERE o.id = $1
  AND EXISTS (SELECT 1 FROM order_tokens t WHERE t.order_id = o.id AND t.token_hash = $2)
```

`pgx.ErrNoRows` is `ErrNotFound`; any other error is returned wrapped and is `500 internal`. The column list is fixed in the source, so no later code can return a private column (S5 SC-36).

**What changes in `GET /v1/orders/{id}`** — nothing a caller can see. If S5's `Read` was written as one statement with the access `WHERE` inline, this story's backend PR **extracts** that predicate into `authorize` and gives `Read` one snapshot — `BEGIN ISOLATION LEVEL REPEATABLE READ, READ ONLY`, `authorize`, the pairs (`order_items JOIN products` by `line_no`, S5 §3.2), commit — which S5 §3.2 allows ("a `REPEATABLE READ` read-only transaction"). S5's own tests for the six not-found cases, the private fields and the token-in-no-log rule are the safety net; they must pass unchanged. After the PR, `FROM order_tokens` appears in **one** read statement in the package (the access `EXISTS`), plus the inserts of S5's placement and the track story.

**How later stories extend it (they edit this function and nothing else):**

| Story | Change | `modePay` |
|---|---|---|
| "Dealer sign-in and protected admin area" | `Access.Dealer bool`; in `modeRead` a dealer reads without a token and gets the dealer view | **Never** — a dealer session without the token is `404` on this route (§3.13) |
| "Customer accounts" | `Access.AccountID` (zero = none) and `… AND (EXISTS (token) OR o.account_id = $3)` | Yes — the linked customer may press "Pay now" with no header; the page needs no change (it already asks with cookies and no header when it holds no token) |
| "Track an order by number and phone" | Inserts `order_tokens` rows | No change |

## 5. What the route refuses, and in which order (steps 3 and 4)

`decide(ref Ref) (Provider, error)` is **pure** — it reads `ref` and the provider map and nothing else — so a table test covers every combination. The offered ways to pay are the ordered list S5 §3.4 fixes (`["vnpay"]`).

| Status | `paid` | Way to pay | Provider for it | Answer |
|---|---|---|---|---|
| `cancelled` | any | any | any | `409 conflict` "This order was cancelled, so it cannot be paid." |
| not `cancelled` | `true` | any | any | `409 conflict` "This order is already paid." |
| not `cancelled` | `false` | not in the offered list (`cod`, `bank`, `momo`, anything) | any | `409 conflict` "This order cannot be paid online." |
| not `cancelled` | `false` | offered (`vnpay`) | none registered | `409 conflict` "Online payment is not available yet." |
| not `cancelled` | `false` | offered | registered, `Available()` false | `503 payment_unavailable` (S7) |
| not `cancelled` | `false` | offered | registered, available, `PaymentURL` returns an address | `200 {"payment_url"}` (S7) |
| not `cancelled` | `false` | offered | registered, available, `PaymentURL` errors, times out (3 s) or returns `""` | `500 internal` |

`placed`, `confirmed`, `packed`, `shipped` and `delivered` are all payable while unpaid — the page offers "Pay now" for exactly that set (SC-25) and the shop does not second-guess it. **Precedence: cancelled before paid** (decision P3): an order cancelled after it was paid (refunded in VNPay's portal) must say it was cancelled, which SC-3 requires ("cancelled, paid or not"). **S7 adds one case** to `decide` — "its thirty minutes have run out" — between the third and fourth rows, and replaces nothing.

## 6. Code seams and tests (what Dev builds)

**Files** (names are Dev's; the split is not): `access.go` — `Access`, `Ref`, `ErrNotFound`, `authorize`, the token filter and hash; `pay.go` — the four sentences as constants, `Refusal` (a type with the message), `ErrPaymentUnavailable`, `decide`, and `Pay(ctx, ref, client)` which runs steps 3–5; the handler in the package's handler file. Each file stays well under 300 lines.

```go
func (s *Service) payHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Cache-Control", "no-store")
    ref, err := authorize(r.Context(), s.pool, r.PathValue("id"), Access{Token: r.Header.Get("X-Order-Token")}, modePay)
    if err != nil { s.fail(w, r, err); return }                       // 1  access
    if hasBody(r) { httpx.Error(w, 400, "bad_request", "This call takes no body.",
                    map[string]string{"body": "This call takes no request body."}); return }   // 2  body
    url, err := s.Pay(r.Context(), ref, clientOf(r))                  // 3–5  state, availability, address
    if err != nil { s.fail(w, r, err); return }
    httpx.JSON(w, 200, map[string]string{"payment_url": url})
}
```

`fail` is **the one error mapper both routes use**: `ErrNotFound` → `404` through the single function that writes the not-found body (so the bytes cannot differ between `GET` and `POST`); `*Refusal` → `409 conflict` with its message; `ErrPaymentUnavailable` → `503 payment_unavailable`; anything else → log `order`, request id and the error, then `500 internal` with the generic message. `clientOf(r)` is the helper S5's `Place` already uses to build `Client{IP, Origin}`; this story does not write a second one. `server.New` is unchanged (S5 already calls `orders.Register`).

**Tests the backend PR carries** (database cases run in CI, which has Postgres via `TEST_DATABASE_URL`; the pure ones run everywhere):

| Test | Proves |
|---|---|
| `decide` table test over every row of §5 and every status | SC-2 to SC-5 and the precedence |
| Handler cases, token arranged by `docs/architecture/erd/order-page-and-pay-now.md` §6: no header, another order's token, unknown id, malformed ids (`DOI-24818`, 101 characters, `-x`), a token with a space or 129 characters — **bodies compared byte-for-byte to each other and to `GET /v1/orders/{id}`'s** | SC-1 |
| Paid sample, cancelled sample (paid and unpaid), unpaid `cod`, `bank` and `momo` samples, each with its token; the order row (and its `updated_at`) unchanged after | SC-2, SC-3, SC-4 |
| An order placed through `POST /v1/orders` (S5), then three `POST …/payment` with the returned token → three identical `409` "Online payment is not available yet."; the order still unpaid; `to_regclass('payment_attempts') IS NULL` or no row for it | SC-5 |
| A body (`{}`) with the right token → `400`; the same with a stranger's token → `404`; a `querier` that returns an error → `500` with the generic body, which contains none of `SELECT`, `orders`, the token or a path | SC-6 |
| Two fake providers (the ones S5's SC-31 uses) registered for `vnpay`: returns `https://pay.example.test/x` → `200` with exactly that string; `Available()` false → `503 payment_unavailable`; returns an error / `""` → `500`; a cancelled or paid order with an available provider is still `409` and **the provider is not called** | The seam and SC-29's "no change to the page" |
| Every answer of the route carries `Cache-Control: no-store`; the package imports no `net/http` client | §3.4, SC-5 |
| `GET /v1/orders/{id}` cases of S5 re-run unchanged after the extraction | §4 |

## 7. The frontend at the swap (the backend PR's frontend half)

All of it is in `src/lib/order-page-and-pay-now.ts`, its test, and the five words. `Order.tsx` and the components keep their code (two of them change a word source, row 8).

| # | Change | Why |
|---|---|---|
| 1 | Replace the two mock imports with `src/lib/api.ts` calls: `GET /v1/orders/${encodeURIComponent(id)}` and `POST /v1/orders/${encodeURIComponent(id)}/payment`. Both send `credentials: "include"` (`api.ts`'s default) | The mock is deleted |
| 2 | Send `X-Order-Token` **only when the browser's store holds a token for that id**, read through the store's own reader in `src/lib/orders.ts` (S5's UI; no second store, no new storage key, that file is not edited). With none held, send the same call without the header. The token is never in the address, the query or a log | §3, §4; SC-8, SC-9, SC-24; the accounts story's no-token read |
| 3 | `readOrder`: **`missing` only for an `ApiError` with `status === 404 && code === "not_found"`**; a resolved body that is not an object with an `items` array, and every other failure (any other 404, 401, 5xx, `network`), is `failed` | §2, row 4: the one departure from the mock |
| 4 | The `POST` sends **no body** (`body` undefined). If `api.ts`'s post helper always serialises something (`"null"`, `"{}"`), use its lower-level request function: **any byte is a `400`**. It is **never retried** (`api.ts` retries a failed `GET` once, never a `POST`); a network failure sends exactly one request | §3.1 step 2; SC-26, SC-28 |
| 5 | `requestPayment`: success is a resolved answer with a string `payment_url` that starts `http://` or `https://`; every rejection and every other body is `{ok:false}` — the same line and the same enabled button (SC-27). The 15-second timeout of `api.ts` bounds "Opening the payment page…" | SC-27, SC-28 |
| 6 | `src/test/order-page.test.tsx`: stub `../lib/api` instead of the mock module; keep the 37 cases' assertions; add cases that the header is sent with a held token and not without one, `credentials` is `include`, the token is not in the URL, and the `POST` has no body and is sent once on a network failure | SC-24, SC-26, SC-28 |
| 7 | Delete `src/lib/mock/order-page-and-pay-now.ts`. Delete `src/lib/mock/orders.ts` **only if** `grep -rn "mock/orders" code/frontend/src` finds no importer afterwards; `Track.tsx` still imports it, so today it stays and the PR says so | SC-31 |
| 8 | **Content keys — a one-time lift of the freeze, new keys only, additive, in this PR** (decision P11), under `pages.order`, in the file's alphabetical order: `load_failed` "We could not load your order. Please try again." (after `items`); `pay_failed` "We could not open the payment page. Please try again.", `pay_now` "Pay now", `paying` "Opening the payment page…" (after `paid`, in that order); `retry` "Try again" (before `spec`). Then delete `src/lib/order-words.ts` and read each word with `useContent("pages.order.<key>")` in `OrderStates.tsx` and `PayNow.tsx`. Existing keys are untouched — `pages.order.due` changes only through PR #24 | The owner edits these words like the rest of the site |

Not edited by this PR: `Order.tsx`, `OrderHead.tsx`, `OrderLines.tsx`, `OrderDetails.tsx`, `Track.tsx`, `Checkout.tsx`, `Product.tsx`, `App.tsx`, `src/lib/orders.ts`, `theme.css`, `index.css`, `package.json`.

## 8. Where each criterion is enforced

| Criteria | Enforced by |
|---|---|
| SC-1 | §3.1 step 1, §4 (one function, one not-found writer); the byte-comparison test of §6 |
| SC-2, SC-3, SC-4 | §3.3's three sentences; §5's rows 1–3 |
| SC-5 | §3.3's fourth sentence, §5's row 4, §3.1's "writes nothing, no outbound call" |
| SC-6 | §3.1 step 2 after step 1; §3.3 `internal`; the failing-`querier` test |
| SC-7 to SC-23, SC-25, SC-32 | The UI PR (#41); the page already reads the §2.8 view as the contract fixes it |
| SC-24 | §4's fixed column list (never a private column); §3.4 logging; §7 rows 1–2 (header only, never an address) |
| SC-26, SC-27, SC-28, SC-29 | §3.3 (every failure is a status the page treats alike), §7 rows 4–5; the fake provider's `200` proves "no change to the page" |
| SC-30 | The running stack: S5's placement, §4's read, §3 |
| SC-31 | §7 rows 6–8 |

## 9. Decisions, and conflicts with merged ones

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| P1 | Which package owns the route? | **`orders`** — the owner of `orders`, `order_items`, `order_tokens` (S5 §5 ownership table), the package that already declares the provider seam. The route is the fifth line of `orders.Register` | A new `pay` package, or `payments`: `payments` does not exist and imports `orders` (`MarkPaid`, `Cancel`), so the route there would force `orders` to be imported back; a `pay` package would copy the access check |
| P2 | How is "written once" met? | One function, `authorize(ctx, q, id, access, mode)`, called by `GET` and `POST`; `Read` takes one `REPEATABLE READ` read-only snapshot around it (S5 §3.2 allowed either form). The dealer branch is `modeRead` only; the accounts branch is both | Copying the predicate into the pay handler (two places to extend, and the first divergence is a security bug); calling `Read` and discarding the view (builds the pairs and joins `products` to answer a `409`); a middleware that loads the order into the context (it would also run on `GET /v1/orders`, which has no order) |
| P3 | Which 409 wins when several apply? | **Cancelled, then paid, then "not an online one", then "not available yet"** (§5). SC-3 says "cancelled (paid or not)"; a paid order of any way to pay says it is paid; only an unpaid, live order reaches the question of how | Paid before cancelled: a paid order that was then cancelled would read "already paid", which tells the customer the pair is coming |
| P4 | New error codes for "already paid" / "cancelled" / "cannot be paid online"? | **No.** The catalog is closed (`_base.md` §2.3): all four refusals are `conflict`, which that table already defines for "payment not possible for this order". Four sentences separate them for a person; the page branches on none | `already_paid`, `order_cancelled`: a code the only consumer would never read, and a change to a catalog every story shares |
| P5 | A `POST` with no body: `415`? `400`? | **`400 bad_request` when a body arrives, no `415`, no content-type check**; one byte is read, after access. This route has no body, so it has no content type: the browser's call carries neither, and the project's "a `POST` with another content type is `415`" (`_base.md` §2.1) cannot apply to a call that has none | `415` on a missing or wrong `Content-Type`: a body-less `fetch` sends none and would be refused; parsing `{}` as valid: it blesses a body the contract says does not exist, and S7 would inherit a route that accepts input it never reads |
| P6 | What is "an online one the shop completes"? | The ordered list of offered ways to pay in code (`["vnpay"]`), not `payment_methods.online` — `bank` and `momo` are `online = true` in the seed (ERD E2) | Reading the column: SC-4 would fail for `bank` and `momo` |
| P7 | Does this story call the provider seam or leave it to S7? | **It calls it** (S5 §4: "the next item's route calls the same `PaymentURL`"), with a 3-second limit; no provider is registered, so production never reaches step 5. The step is covered by fakes now so S7 registers a provider and changes **no line** of the route, which is what the story promises the page | Stopping at "always 409" and rewriting the handler in S7: the promise "no change to the Order page" would rest on a handler nobody has run with a provider. Mapping a provider error to `409`: the catalog marks `conflict` not retryable and the customer should retry an outage (`internal`, retryable) |
| P8 | No provider registered: which status? | `409 conflict` "Online payment is not available yet." exactly as `_base.md` §3.13 and ORD-006 AC-12 say | `503 unavailable`: that code means the process is shutting down or `/healthz` found no database. `501`: not in the catalog. `payment_unavailable` is reserved for "a provider is registered and its keys are not set" (S7) |
| P9 | Caching of the answers | `Cache-Control: no-store` on every answer of the route (200, 4xx, 5xx); a refusal must never be replayed from a cache after the order was paid | None considered |
| P10 | Which 404 is "not found" for the page? | Only the API's own (`error: "not_found"`); any other 404 is "could not load" (§2, §7 row 3) | The mock's "any 404": a broken proxy would tell every customer their order is gone |
| P11 | Who adds the five content keys, and when? | **Dev, in the backend PR, new keys only** (§7 row 8) — the one-time lift of `overview.md` §5's freeze that the staff-accounts and shop stories already received, confirmed here. This revises the earlier note that TL adds them in the backend step: PR #24 (open) edits `content.json` too, and a design PR that touched the file now would put a conflict on it; the backend PR is built after #24 by the dependency order, so it cannot | Leaving the fallbacks in `src/lib/order-words.ts` for good: the owner could never edit those words in the Studio. Adding them in this design PR: the conflict above |

### Conflicts with merged decisions

1. **`_base.md` §2.1 ("a `POST`, `PUT` or `PATCH` with another content type … is `415`")** against this route, which has no body. Resolved as P5: the rule applies to endpoints with a JSON body; this one's refusal of a body is `400`. Cost: the carve-out is written here and in the route's tests, not in the middleware.
2. **S5 §3.2's `Read` as "one statement"** against the story's "access check written once". S5 itself offered "or a `REPEATABLE READ` read-only transaction"; the choice is made here (P2). Cost: one extra statement per order page (the access row, then the pairs, inside one snapshot) — two indexed reads where there was one aggregated statement. Accepted: the extraction is what lets the dealer and accounts stories change one place.
3. **`_base.md` §3.13's list of 409 causes is not ordered.** The story's SC-3 ("cancelled, paid or not") fixes the order of the first two (P3); no earlier document says otherwise.

No other. I read `services/_base.md` §2.1–2.6, §3.11, §3.13; `services/checkout-and-placing-an-order.md` §2–§5, §7, §8; `services/site-data-from-the-database.md` §5.1 (the API client); and `docs/orders/SRS.md` ORD-006 and ORD-007.

## 10. Ordering with the open stories, and hand-over

| Matter | Rule |
|---|---|
| Build order | PR #24 (free delivery; changes `pages.order.due`) → PR #39 (site data: `api.ts`, `delivery` and `payments` reads) → S5's PRs (the UI's `src/lib/orders.ts` token store, PR #32; the backend's `internal/orders` with `Read`) → **this** story's backend PR on top. Until S5's backend merges there is no `orders` package to extend and `/orders/:id` has no real order to show |
| PR #41 (this story's UI) | Keeps merging first; it ships the mock-backed page. The backend PR swaps it (§7) |
| "Online payment with VNPay" (S7) | Registers the `vnpay` provider in `orders.Options{Providers}`; its `Available()` is the key check; its `PaymentURL` writes `payment_attempts` under the order's row lock (ERD E5) and may refuse an expired order (one new `case` in `decide`, §5). `/payment/return`'s "Try again" calls this same route. **No line of `Order.tsx`, `PayNow.tsx` or the client changes** |
| "Customer accounts", "Dealer sign-in" | Extend `authorize` only (§4 table) |
| "Track an order by number and phone" | Inserts tokens through the same hash; deletes `src/lib/mock/orders.ts` if this story could not (§7 row 7) |
| Test lead | SC-1 to SC-6 are called directly on the API; the arrangement SQL is in the ERD file §6. SC-5's "no `payment_attempts` row" is asserted with `to_regclass` (ERD §7). SC-30 needs S5's placement and a real shoe in a size with pairs. The 2-second answer of SC-10, the 500 and the network failure of SC-11 are produced outside the backend (a hold on one URL, `set_db_available`, a stopped container); **no test-only route exists** |
| Dev, backend step | One route, one extracted function, the tests of §6, the swap of §7. **No migration, no new setting, no new package.** If anything in §3–§5 cannot be done as written, reply with the blocker rather than widening the contract |

## 11. Migration plan

**None.** This story creates no table, column, index, constraint or function and adds no `.sql` file (ERD file §5). Forward: nothing runs. Backward: revert the backend PR; the schema is exactly S5's before and after, so there is no populated table to protect and no irreversible step to justify.
