# Service contracts — Free delivery on every order, wording to match (plan item 2)

Module `site` · Requirement SITE-016 · Story `docs/site/stories/free-delivery-on-every-order-wording-to-match.md` · UI PR #24 · Tables: `docs/architecture/erd/free-delivery-on-every-order-wording-to-match.md` (none changed).

## 1. Verdict: this story adds, changes and removes no endpoint

Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/delivery` reaches Go as `/v1/delivery`.

The story is frontend and JSON only, and the backend it will one day read from is built by later stories. No request, response, status code or error code is introduced here, and the project's error catalog (`{error, message, errors}`, lowercase codes) and offset paging (`{items, total, page, limit}`) are reused untouched. What this design adds is the **list of answers the existing endpoints must give once the files are as this story leaves them**, and the three server-side rules that make the owner's decisions hold on the write side too. They are checks for S3, S5 and the Test lead, not new contracts.

## 2. What the existing reads answer after this story

All are public `GET`s of S3 (`docs/architecture/services/site-data-from-the-database.md`). The frontend already reads these records through `src/data/index.ts`, so S3 changes the source, not any page.

| Request | Answer the files imply | Read by |
|---|---|---|
| `GET /v1/delivery` | `{"items":[{"id":"standard","name":"Standard","days":"2–4 working days","fee":0,"note":"Nationwide with GHN or GHTK, tracked from our door to yours."}],"total":1,"page":1,"limit":12}` — one item, never two | `Shipping` (the options table; its Cost cell reads the string `pages.cart.free`, **not** `fee`), `Checkout` step 02 (name, days, note) |
| `GET /v1/delivery/standard` | the item above, unwrapped | `Checkout`, `Order` (the "Delivery" detail) |
| `GET /v1/delivery/express` | `404 not_found` — "Not found." The id never exists | nothing |
| `GET /v1/shop/shop` | `{"id":"shop","standard_days":"2–4","dispatch_cutoff":"15:00","reply_time":"…","rating":4.8,"review_count":181}` — five fields besides `id`; **no** `free_delivery_from`, `express_days`, `express_cities`, `exchange_days`, `refund_days` | `PromiseRow`, `Shipping` (second figure), `Product` (delivery sentence) |
| `GET /v1/payments` | four items in the order `cod`, `bank`, `vnpay`, `momo`; `vnpay`'s `note` is "Pay by card or banking app on VNPay's secure page." with no "marked paid at once" | dealer screens and the five sample orders' pages (they name the stored `cod` record) |
| `GET /v1/payments/vnpay` | the `vnpay` item, unwrapped | `Checkout` step 03 and `Product` "Payment options" until the Checkout story moves them to `GET /v1/payment-options` |
| `GET /v1/faqs?topic=payment` | first row `f-payment-how` ("How do I pay?"), then `f-payment-genuine`; **no** row mentions cash on delivery | `Returns`, `Size chart`, `Track` pages |
| `GET /v1/faqs?topic=shipping` | four rows (`f-delivery-time` … `f-delivery-gift`), unchanged count | `Shipping` |
| `GET /v1/orders/{id}` for DOI-24815, -24810, -24806 | `delivery` `"standard"`; `total` 5890000, 3290000, 3090000 | `Order`, dealer screens |

These are the same fields, kinds and names as the JSON; S3's §6 table of "where the API differs from the files" is unaffected.

## 3. The write side already holds the owner's decisions

`POST /v1/orders` and the payment routes are S5's and S7's contracts, defined in `services/_base.md` §3.10, §3.13. They already enforce what this story only words; this table is what Test and Dev must be able to show, so the two sides cannot disagree.

| Rule | Where it is fixed | What the caller sees |
|---|---|---|
| Delivery is always `standard` | `_base.md` §3.10 request table: "`delivery` must be `standard`" | `delivery: "express"` or any other id → `422 invalid`, `errors.delivery` |
| `delivery_fee` is a constant 0, `total` = Σ `unit_price × qty` | `_base.md` §3.10 "What the shop computes" | A body that carries `delivery_fee: 50000` or `total: 1` is read and **discarded** (the tolerated extras, §2.5); the stored order has fee 0 and the shop's own total |
| `cod` is never a way to pay | `_base.md` §3.10 request table and §3.5 | `payment: "cod"` → `422 invalid`, `errors.payment`; `GET /v1/payment-options` never lists it |
| Only VNPay's own notice marks an order paid | `_base.md` §3.17 (the notice) | No other route sets `paid`; `POST /v1/orders` always answers `paid: false` |

No new code is added to the catalog and no existing code changes meaning. The codes reachable through this story's records are `not_found` (404, `GET /v1/delivery/express` or an unknown id), `bad_request` (400, a malformed `page`/`limit`) and `internal` (500, a failed query — never `unavailable`).

## 4. The mock, read and judged

The UI PR adds **no mock module**: the story trims `src/lib/mock/orders.ts` (the placed-order mock the Checkout story deletes) and edits JSON. There is therefore no mock to swap in this story; the file's shape is judged here because the Checkout story, which replaces it with the real client, is queued behind this one and must not inherit a field the contract does not have.

| Mock (`lib/mock/orders.ts` after PR #24) | Contract (`_base.md` §2.8, §3.10) | Judgment | Frontend change when the API lands (Checkout story) |
|---|---|---|---|
| `NewOrder` has **no** `delivery` field; `placeOrder` stores `delivery: "standard"` | `delivery` is a required request field and must be `standard` | The mock is right that the **user** has no choice, but the real request still carries the field | `lib/orders.ts` sends the literal `"delivery": "standard"`. It is a constant, not state: no delivery state, no radio, no Express branch comes back (SC-21, SC-32) |
| `NewOrder.payment` and `online` | `payment` is the offered id; `online` is not a request field (an undefined field is `400`) | `online` is derived from the chosen method; sending it would be refused | The request sends `payment: "vnpay"` and no `online` |
| `PlacedOrder.delivery_fee` **removed** | `Order.delivery_fee` is in the answer, always 0 | Nothing on the site reads it. The answer carries a field the client's type need not declare | None: the client's `Order` type omits `delivery_fee`. The page's "Free" is the string `pages.cart.free`, never a number |
| `PlacedOrder.subtotal` and `total` (`total = subtotal`) | Only `total`, plus `items[].unit_price` and `qty` | `subtotal` is a mock convenience, equal to `total` while the fee is 0 | The client derives it from `items` if a page needs it, or drops it; the displayed Total is the answer's `total` |
| `PlacedOrder.eta` from a hard-coded `STANDARD_DAYS = 4` | `eta` is computed by the shop (S3 ERD §4.5: the longest number of `standard_days`) | The constant is a mock stand-in for the shop's own rule | None: the page shows the answer's `eta`; the constant dies with the mock |
| `PlacedOrder.paid = input.online` | `paid` is `false` on every new order; only the VNPay notice sets it | The mock marks an online order paid at once; the shop never does (PAY-002) | The Checkout story's mock/clients already treat a new order as unpaid; this story does not touch `next_online` (story §4) |

Where the contract's `Order` shape differs from the mock, the contract wins (services `_base.md` decided it first). None of these differences is introduced by this story.

## 5. Contracts reused unchanged

| What | Where | How this story uses it |
|---|---|---|
| Error envelope and the closed code catalog | `_base.md` §2.3 | Reads of `delivery`, `shop`, `payments`, `faqs`, `orders` use `not_found`, `bad_request`, `internal` only |
| Offset paging | `_base.md` §2.4 | `GET /v1/delivery` is `{items, total: 1, page: 1, limit: 12}` |
| Records served as the JSON names them | S3 §3 | The only change is that three fields no longer exist on `shop` and one record no longer exists on `delivery` |
| Idempotency, rate limits, jobs, outbound calls | — | None in this story |

## 6. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| G1 | Does any page need an endpoint that says "delivery is free"? | **No.** "Free" is the content string `pages.cart.free`, shown wherever delivery is shown. The Cart, Checkout and Order pages never read `delivery.fee`; the Shipping page's Cost cell reads the same string | An endpoint or a `shop` field that returns the fee: it would put a number back in front of five pages that the owner decided have none, and the "total on the page is the total paid" guarantee would depend on every page remembering to ignore it |
| G2 | Server-side price and fee | The server computes `delivery_fee = 0` and `total` itself and ignores any client amount (existing contract). It does **not** read `delivery_options.fee` (ERD F1) | Reading the row: a stale or edited value would be charged while the pages say "Free" |
| G3 | A threshold or an express option "for later" | None kept. No field, no branch, no unused enum value: `orders.delivery` has one option and the API accepts one id | Keeping `express` in the contract "disabled": a refused value is cheaper to add back than a dead one is to explain, and SC-32 forbids the word in `src` |

### Conflicts with merged decisions

None. I read `services/_base.md` §2.3–2.8, §3.2–3.5, §3.10, §3.17, the S3 services file §1–§4, and `overview.md` §8 (rows 2 and 5): each already says `delivery` is `standard`, fee 0, and `cod` is never offered.

## 7. Rollout and ordering

| Check | Rule |
|---|---|
| S3 before or after PR #24 | **After.** S3's seed assertion (`total = unit_price × qty + delivery_fee`) and its drift-guard test (`collections_test.go` reads `schema.json`) both fail on `main`'s old files. This is the intended tripwire (S3 ERD open question 1, S3 services §4): Dev stops and tells PM, never edits a total or a schema key |
| S5 after this story | Builds on the one-row delivery step and the VNPay-only payment step PR #24 leaves; its `lib/orders.ts` follows §4 |
| Migration | None (ERD §6). No endpoint is added, so there is no route to mount, no `/api` prefix to check and no new test fixture |
| Verification for the Test lead | The reads of §2 against the files; and, once S5 lands, the three write-side refusals of §3 (`express` → 422, `cod` → 422, a client `delivery_fee` ignored) |
