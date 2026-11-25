# Story — Checkout and placing an order

Module: `orders` · Plan item 5 · Requirements: ORD-001, ORD-002, ORD-003, ORD-004, ORD-005, ORD-006, ORD-009, ORD-010, and the privacy half of ORD-008 (`docs/orders/SRS.md`)
Design system: `design/design-system.md` · Architecture: `docs/architecture/overview.md`, `erd.md` (§3.11 stock, §3.12–3.14 orders, §8 migrations 4–6), `services.md` (§3.5, §3.10, §3.11, §3.14, §2.6)

## 1. User story

As a **customer** (a guest — no account is needed), I want to enter where my pairs should be sent, see that delivery is free and what the total is, and place the order, so that the shop saves it at its own prices, sets my pairs aside, and I can go on to pay for it online.

Other actors this story touches: **Stranger** (anyone without the order's token learns nothing about any order) and **Payment provider** (not live yet: the seam ships here, VNPay plugs into it in "Online payment with VNPay"). The shop team reads orders only after the dealer sign-in story.

## 2. How this plan item was cut

The SRS gives this item ten requirements and 121 acceptance criteria; one verification round cannot carry that, and every fix would re-run all of it. The item is cut at the line between **placing** and **reading**:

- **This item — placing:** the Checkout screen, `POST /v1/orders`, taking pairs off the shelf and keeping the stock lists true, the payment seam, where the browser lands, failure and repeat-submit handling, and the privacy lock on orders.
- **New item "Order page and Pay now" (added to the plan directly after this one):** the Order page `/orders/:id` on the API, `POST /v1/orders/{id}/payment`, and the end-to-end flow that ends on a heading containing "Thank you" (ORD-007 in full, ORD-006 AC-12, ORD-009 AC-5).

Consequence the reviewers should know: between the merge of this item and the merge of the next, `/orders/:id` is still the design's mock page, so verification here stops at the address the browser lands on and the token it stores (SC-31).

## 3. In scope

**Backend (Go, PostgreSQL)**

- `POST /v1/orders` as `services.md` §3.10 fixes it: the pairs (shoe id, EU size, quantity), delivery, payment, name, phone, address, city, note; header `Idempotency-Key`. The shop re-reads every price, stores `unit_price` per pair, computes the total as the sum of price × quantity, stores the delivery fee as 0, takes the pairs off the shelf and saves the order, its pairs and a hashed token in **one transaction**. Number `DOI-<n>` / id `doi-<n>` from the counters row, status `placed`, paid false, expected date from the `standard` option.
- A payment-provider interface that each way to pay plugs into; at this point no provider is implemented, so `payment_url` is `""`.
- `GET /v1/payment-options` (§3.5): VNPay alone; `cod`, `bank` and `momo` are never listed.
- Per-size stock: migration 4 `stock_per_size` (table `product_sizes`, view `product_stock`, backfill from the seeded lists, drop the four stock columns on `products`), so a shoe's `sizes`, `sizes_low`, `sizes_out` and `stock` are derived from the real counts. Migration 5 `order_checkout` (`order_tokens`, `counters` seeded from the highest seeded number, `orders.idempotency_key`) and migration 6 `orders_idempotency_index` (`CONCURRENTLY`). The seeded orders each get one `order_items` row.
- Privacy: `orders` stops being a public collection. `GET /v1/orders` answers 401 (no dealer session can exist yet); `GET /v1/orders/{id}` answers the public view to the order's `X-Order-Token` and 404 to everything else.

**Frontend (Vite + React + Tailwind tokens)**

- `src/pages/Checkout.tsx` on the API: current prices, the one delivery row, the one payment row from `GET /v1/payment-options`, real placing, and the states the design never drew (refused with named pairs, failed, offline, payment unavailable).
- `src/lib/orders.ts` (new): the API client for placing an order, the `Idempotency-Key`, and the browser's store of placed orders (id, number, placed time, token) under a key of its own, so the design's mock records are never read as real ones. `src/lib/cart.ts` stays local; the bag and drawer are unchanged.
- Checkout stops importing `src/lib/mock/orders.ts`. The mock module stays in place because `Order.tsx` and `Track.tsx` still import it; the next item and "Track an order by number and phone" retire it.

## 4. Out of scope — and where each piece is proven

| Not here | Why | Proven in |
|---|---|---|
| The Order page `/orders/:id` on the API, "Pay now", `POST /v1/orders/{id}/payment`; ORD-007 (26 criteria), ORD-006 AC-12, ORD-009 AC-5 | Cut from this item (§2) | New item "Order page and Pay now" |
| VNPay itself: the payment address, the notice, the return page, the 30-minute expiry that returns pairs; the 503 `payment_unavailable` and the key check (ORD-006 AC-10, AC-11) | Needs the owner's VNPay keys and notice address | "Online payment with VNPay" (the UI half of the unavailable state is SC-8) |
| Linking an order to a signed-in customer (ORD-003 item 9, AC-19) and a customer reading their own order (ORD-008 AC-11, AC-12) | No customer session exists until accounts land; every order here is a guest's (`account_id` empty) | "Customer accounts" (ACC-005) |
| Dealer and staff reading orders and the orders list (ORD-008 AC-2, AC-3, AC-7); moving an order along; cancelling and returning pairs | No dealer session until dealer sign-in | "Dealer sign-in and protected admin area", "Manage shoes, lines, categories, reviews and orders" |
| The shoe page's "Payment options" listing VNPay alone (ORD-006 AC-4, SITE-006 AC-2) | Edits `Product.tsx`, which belongs to "Free delivery on every order" and the product-page fix | Those items |
| Draft or deleted placeholder shoes, and staff typing per-size counts (ORD-005 AC-7) | Admin screens | "Manage shoes, lines, categories, reviews and orders" (ADMIN-014, ADMIN-015) |
| Track by number and phone, "Ordered on this phone", lookup lock (ORD-020 to ORD-023) | Own plan item | "Track an order by number and phone" |
| Bank transfer (VietQR) and MoMo as ways to pay | Later P3 story; their rows stay in the records | "More ways to pay" |
| A text or email to the customer | Not in the brief; the design's words that promise one are open question 4 of the SRS | Owner's answer |
| Delivery fee, free-delivery threshold, express option, cash on delivery | Removed by the owner's decisions; the earlier item "Free delivery on every order" cleaned the records and pages. Nothing here brings any back | — |

## 5. UI scope

The approved design's Checkout screen is the base; the work is replacing the mock with the API and drawing four states the design left out. Tokens only (`bg-ground`, `text-ink`, `border-field` …), no arbitrary values, no CSS modules; each file under 300 lines (split `Checkout.tsx` if the new states push it past that).

| Screen / state | What it shows | SRS |
|---|---|---|
| Checkout `/checkout`, default (1280 px and 390 px) | Heading, three steps from 1024 px, secure line, five labelled fields, one delivery row "Free delivery", one payment row VNPay, summary with Subtotal / Delivery "Free" / Total, next-step line, "Place order", exchanges sentence | ORD-001, ORD-006 |
| Empty bag | "Nothing to check out yet." and "New this month" | ORD-001 item 8 |
| Placing | Button reads "Placing your order…", disabled | ORD-001 item 7 |
| Validation | Alert lines for the name/address/city and the phone message | ORD-002 |
| Refused (not in the approved design) | One line per named pair above the button: `<shoe name> · EU <size> — sold out in your size.` / `only <n> left in your size.` / `no longer in the shop.` | ORD-004 item 4 |
| Failed, offline (not drawn) | "We could not place your order. Nothing was charged. Please try again." / "You seem to be offline. Check your connection and try again." | ORD-010 |
| Payment unavailable (not drawn) | VNPay row stays; alert "Online payment is not available right now. Please try again later."; "Place order" disabled | ORD-006 item 6 |

Copy notes for TL's UI review: the new strings above and the changed `pages.checkout.next_online` sentence (the design's "the order is marked paid at once and packed today" becomes "…we pack your order after the payment is confirmed.") must stay editable by the owner like the rest of the site's words. `src/content.json` is frozen for stories (overview §5), so the UI PR lists each new key and TL says where it is added. The mock cash-on-delivery sentence (`next_cod`) and any `cod` default in `Checkout.tsx` go away.

## 6. Acceptance criteria

Each criterion names the SRS requirement and criterion it narrows (`SC-n [REQ AC-m]`). Test writes at least one case per `SC`; TL's reviews count this list. "Shop" means the running API; amounts are VND with `vi-VN` formatting.

### A. The checkout page (ORD-001, ORD-006)

- **SC-1 [ORD-001 AC-1]:** With one pair in the bag at 1280 px, `/checkout` shows the heading "Checkout"; the steps "01 Contact and address", "02 Delivery", "03 Payment"; the line "Your details travel over a secure connection."; and the fields "Full name", "Phone", "Street address, ward, district", "City or province" and "Note for the courier (optional)" (two lines), each with its visible label.
- **SC-2 [ORD-001 AC-2]:** At 390 px, and down to 320 px, the three-step line is not shown, the form is one column and the page does not scroll sideways; the phone field opens a phone keypad (`type="tel"`, `inputMode="tel"`).
- **SC-3 [ORD-001 AC-3, AC-4]:** Step 02 shows exactly one delivery row — icon, "Free delivery", the `standard` record's note and arrival time, and "Free" at the right — already selected and not a choice. The word "Express" appears nowhere on the page.
- **SC-4 [ORD-001 AC-5, AC-6, AC-7, AC-8]:** The summary "Your order" lists each pair (picture, name in capitals, `<variant> · Qty <n>`, line total) at the shop's current price read when the page opens: a bag line saved at 3.000.000 ₫ for a shoe now priced 3.090.000 ₫ shows `3.090.000 ₫`. Delivery reads "Free", never `0 ₫`, and Total equals Subtotal: one pair at 3.090.000 ₫ → `3.090.000 ₫`; pairs at 3.090.000 ₫ and 1.500.000 ₫ → `4.590.000 ₫`; one pair at 1.500.000 ₫ (under 2.000.000 ₫) → `1.500.000 ₫`.
- **SC-5 [ORD-001 AC-9, AC-10; ORD-006 AC-1, AC-3]:** Step 03 shows one row, already selected: icon, "VNPay — ATM, Visa, Mastercard" and its note, read from `GET /v1/payment-options`. No row for cash on delivery, bank transfer or MoMo is drawn although their records exist. Under the totals: "After you press Place order you pay on the VNPay — ATM, Visa, Mastercard page; we pack your order after the payment is confirmed."
- **SC-6 [ORD-001 AC-11; ORD-010 AC-6]:** "Place order" carries a lock icon and under it reads "By placing the order you accept the delivery information and that exchanges are not accepted." While an order is being placed the button reads "Placing your order…", is disabled, and pressing it again sends nothing: exactly one `POST /v1/orders` leaves the browser.
- **SC-7 [ORD-001 AC-12]:** With an empty bag `/checkout` shows "Nothing to check out yet.", the line "Pick a pair — the size chart is one tap from every shoe.", a "Keep shopping" pill and "New this month" with up to four live new shoes.
- **SC-8 [ORD-001 AC-14; ORD-006 AC-2, AC-10]:** When `GET /v1/payment-options` reports `vnpay` with `is_available: false`, the VNPay row stays, "Online payment is not available right now. Please try again later." shows under it in alert style and "Place order" is disabled; with `is_available: true` it is enabled. (Proven here with the response stubbed in the browser; the shop reports `false` only after "Online payment with VNPay" lands and its keys are missing.)

### B. Details are checked (ORD-002)

- **SC-9 [ORD-002 AC-1, AC-2, AC-7]:** "Place order" with the name, street address or city empty or only spaces shows "Fill in your name, street address and city." with an alert icon, in a region announced to assistive technology (`role="alert"`), and sends nothing. With that and a phone of `12`, both messages show, one per line.
- **SC-10 [ORD-002 AC-3, AC-4, AC-5, AC-6]:** A phone of `0912 34`, or any phone of 8 digits, shows "Check the phone number — we text you when the pair is on its way." and sends nothing; `091234567` (9 digits) and `+84 912 345 678` (11 digits; spaces, `+`, `-` and brackets do not count) are accepted.
- **SC-11 [ORD-002 AC-11; ORD-004 AC-8]:** After a failed validation, a refusal or a failed send, every typed value is still in its field.
- **SC-12 [ORD-002 AC-8, AC-9, AC-10, boundary]:** The shop applies the same rules to a request that skips the page and answers 422 with `{"error":"invalid","errors":{"<field>":"<sentence>"}}`: an empty address names `address`; a note of 501 characters names `note` and the limit 500, while 500 is created; a name of 100 characters is accepted and 101 refused naming the limit; a phone over 20 characters, an address over 200, a city over 100, or a phone with fewer than 9 digits is refused naming the rule. No order is created and no pair leaves the shelf for any refusal.

### C. The shop computes everything (ORD-003)

- **SC-13 [ORD-003 AC-1, AC-2]:** With `DOI-24817` the highest number held, the next order placed is `DOI-24818`, id `doi-24818`, status "placed", paid false; two orders placed at the same moment both succeed with different numbers.
- **SC-14 [ORD-003 AC-3, AC-4, AC-6, AC-9, AC-18, conflict row]:** The shop computes the total from its own prices: 2 × 3.090.000 ₫ stores `6.180.000 ₫` with delivery fee 0; 1 × 1.500.000 ₫ stores `1.500.000 ₫` with fee 0; a basket under 2.000.000 ₫ and a basket over it each total exactly Σ price × quantity with fee 0. A request that carries a unit price of 1 ₫, a delivery fee of 50.000 ₫ or a total is accepted and those numbers are ignored. When a shoe's price changes between the page loading and the order, the order is saved at the price at the moment of saving.
- **SC-15 [ORD-001 AC-13; ORD-003 item 2]:** For a completed checkout with no price change in between, the stored total equals the Total the page showed when "Place order" was pressed.
- **SC-16 [ORD-003 AC-5, AC-15, AC-16, AC-17, failure row]:** The shop answers 422 naming the field — no order, no pair taken — for: a `delivery` other than `standard` (`express`, an unknown id) → `delivery`; a `payment` that is not offered (`cod`, `bank`, `momo`, an unknown id) → `payment`; a line with quantity 0 or a size the shoe does not list → `items.<index>`; an empty list of pairs.
- **SC-17 [ORD-003 AC-7, AC-8, AC-10]:** The expected date is the base day plus the longest working days of `standard` (4 for "2–4 working days"), in Vietnam time, skipping Saturdays and Sundays: placed Monday 10:00 → that Friday; Monday 16:00 (after the 15:00 cut-off) → the next Monday; Saturday → the Friday of the following week.
- **SC-18 [ORD-003 AC-11, AC-13, item 4]:** An order of two shoes stores two pairs, each with size, quantity and the unit price it was sold at; changing a shoe's price afterwards leaves the pair's unit price and the order total as sold (read back with the order's token).
- **SC-19 [ORD-003 AC-12, item 5]:** Each of the 14 seeded orders is an order of exactly one pair; `doi-24817` is one pair of `samba-og-w-crystal-white`, EU 39 1/3. (Checked in the database by a backend test: no one can read orders as the dealer until the dealer sign-in story.)
- **SC-20 [ORD-003 AC-14; ORD-006 AC-9]:** A successful placement answers 201 with `order` (id, number, placed time, status "placed", paid false, total, delivery, payment, pairs — and no `phone`, `email`, `address` or `note`), `token` of at least 22 URL-safe characters (at least 128 bits of randomness; only its hash is stored) and `payment_url`.
- **SC-21 [ORD-003 failure row]:** With the database unreachable, placing answers 500 with a generic body (no SQL, stack or another person's data); no order exists and no pair left the shelf.
- **SC-22 [NFR Performance]:** Over 100 sequential orders against the running API, placing answers within 1.5 s at p95.

### D. Pairs leave the shelf, and the stock shown stays true (ORD-004, ORD-005)

- **SC-23 [ORD-004 AC-1, boundary]:** Size `38` holds 2 pairs: an order for 1 is saved and 1 is left. An order whose quantity equals the pairs left is accepted and the size is then sold out.
- **SC-24 [ORD-004 AC-2; NFR Integrity]:** Size `38` holds 1 pair and two orders for it are sent at the same time: exactly one answers 201, the other 409 naming that pair, and 0 pairs remain, never −1. With 50 simultaneous orders for one size holding 10 pairs, exactly 10 pairs sell, no count is below 0 and every refusal names its pair.
- **SC-25 [ORD-004 AC-4, AC-5, AC-6, AC-7, not found; ORD-005 AC-8]:** A refused order answers 409 `stock_conflict` naming **every** failing pair with its sentence: `<shoe name> · EU <size> — sold out in your size.`; `<shoe name> · EU <size> — only <n> left in your size.` (3 asked, 2 left → "only 2 left", 2 remain); `<shoe name> · EU <size> — no longer in the shop.` (a shoe set to draft after it went into the bag, or an id that does not exist). In a bag of two pairs whose second is sold out only the second is named and the first pair's stock is unchanged; after any refusal the counts, the lists and the total are exactly as before.
- **SC-26 [ORD-004 AC-3; ORD-009 AC-4]:** On a refusal Checkout shows, for each named pair, its sentence — `<shoe name> · EU 38 — sold out in your size.` — keeps every pair in the bag and the typed details, and enables the button again.
- **SC-27 [ORD-005 AC-1, AC-2, AC-3, AC-5, AC-6]:** A shoe's last-pairs list, sold-out list and "pairs in stock" follow the count of each size the moment an order is saved (read from the shoe's public record): 3 pairs → order 1 → the size is in the last-pairs list; 2 → 1 → it stays there; 1 → 1 → it is in the sold-out list and out of the last-pairs list. Sizes holding 3, 2, 0 and 5 pairs give stock 10; an order for 2 of the size with 5 gives 8.
- **SC-28 [ORD-005 AC-4]:** After the last pair of a size sells, the shoe's page loaded afterwards shows that size's pill struck through and not choosable (the existing size picker; no new UI).
- **SC-29 [ORD-005 item 4; DATA-001 AC-2]:** After stock moves to one count per shoe and size, every existing shoe lists the same sizes, last-pairs sizes and sold-out sizes as before, and every size that has pairs can be ordered. The sample shoes' "pairs in stock" totals may differ from the old figures (`erd.md` question 1), so no criterion pins one.

### E. The payment seam (ORD-006)

- **SC-30 [ORD-006 AC-1, AC-3, AC-5]:** `GET /v1/payment-options` lists `vnpay` alone, with id, name, note, icon, `online` and `is_available` (`true` until the VNPay story adds the key check); `cod`, `bank` and `momo` are never listed, although `GET /v1/payments` still returns all four rows.
- **SC-31 [ORD-006 AC-6, AC-7, AC-8, item 1]:** Placing goes through one payment-provider interface. With no VNPay provider implemented, the order is saved unpaid and `payment_url` is `""`; a test provider returning `https://pay.example.test/x` puts exactly that address in the answer; a provider that fails to give an address leaves `""` and the order is still saved, unpaid. (Proven with test providers; no real provider ships here.)

### F. Where the browser lands (ORD-009)

- **SC-32 [ORD-009 AC-1, AC-2, AC-3]:** When the answer's `payment_url` is empty, the bag is emptied, the browser goes to `/orders/<id>?placed=1`, and the browser's storage holds the order's id, number, placed time and token and nothing else about it. When `payment_url` is an address, the bag is emptied, the same is stored and the browser goes to that address (stubbed). If the browser refuses storage the order is still placed and the browser still goes on; the token then lasts this page load only.

### G. Failed, offline and repeated submissions (ORD-010)

- **SC-33 [ORD-010 AC-1, AC-2]:** When the shop answers 500 or cannot be reached, Checkout shows above the button "We could not place your order. Nothing was charged. Please try again."; when the browser is offline it shows "You seem to be offline. Check your connection and try again." In both cases the bag and the typed details stay and the button is enabled.
- **SC-34 [ORD-010 AC-3, AC-4, AC-5, boundary]:** Each attempt carries an `Idempotency-Key` made once for that bag and those details. The same key sent twice answers the first order again (200, `Idempotent-Replayed: true`, a new token) and the shelf lost the pairs once; if the first send saved the order but its answer was lost, pressing "Place order" again lands on that order; the same key from two tabs at once makes one order. A new key is made when the bag or any typed detail changes. A missing or malformed key answers 400 naming the header; the same key with a different phone answers 409 `conflict`.

### H. Privacy (ORD-008)

- **SC-35 [ORD-008 AC-1, AC-9, items 1 and 5]:** Orders are not a public collection: `GET /v1/orders` with no session answers 401, also with a filter such as `?phone=…`, and — as no dealer session can exist before the dealer sign-in story — for everyone; the generic reader no longer serves `orders`.
- **SC-36 [ORD-008 AC-4, AC-5, AC-6, AC-8]:** `GET /v1/orders/<id>` with no token answers 404; with the order's own token in `X-Order-Token` it answers 200 with the order and without `phone`, `email`, `address` or `note`; with another order's token it answers 404; for an id that does not exist it answers 404 with the same body as the no-token case.
- **SC-37 [ORD-008 AC-10, item 4]:** The order token appears only in the placing answer (and a replay's) and in this browser's storage: it is in no address, no page text and no log line.

## 7. Where every SRS criterion of these requirements is proven

| Requirement | Criteria | Criterion list |
|---|---|---|
| ORD-001 | AC-1 to AC-14 | SC-1 to SC-8, SC-15 (AC-13) |
| ORD-002 | AC-1 to AC-11 | SC-9 to SC-12, SC-11 |
| ORD-003 | AC-1 to AC-18 | SC-13 to SC-21; **AC-19** is out of scope (§4, accounts) |
| ORD-004 | AC-1 to AC-8 | SC-11, SC-23 to SC-26 |
| ORD-005 | AC-1 to AC-6, AC-8 | SC-25, SC-27 to SC-29; **AC-7** is out of scope (§4, admin) |
| ORD-006 | AC-1, AC-2, AC-3, AC-5 to AC-10 (UI half), AC-9 | SC-5, SC-8, SC-16, SC-20, SC-30, SC-31; **AC-4, AC-10 (503 half), AC-11, AC-12** are out of scope (§4) |
| ORD-008 | AC-1, AC-4, AC-5, AC-6, AC-8, AC-9, AC-10 | SC-35 to SC-37; **AC-2, AC-3, AC-7, AC-11, AC-12** are out of scope (§4, sessions) |
| ORD-009 | AC-1 to AC-4 | SC-26, SC-32; **AC-5** moves to the next item (§2) |
| ORD-010 | AC-1 to AC-6 | SC-6, SC-33, SC-34 |
| ORD-007 | all | **Out of scope: the next item (§2)** |

## 8. Dependencies

**Must land first**

- **"Free delivery on every order, wording to match"** — the `delivery` record has one row, `standard`, fee 0; `shop` has no `free_delivery_from`, `express_*`; the mock Checkout, Cart and Order lost their fee logic. The seed copies the files as they stand (overview §8, story 2). Do not bring any of it back.
- **"Site data from the database"** — the ten tables with seeded orders, `src/lib/api.ts` (base URL, `credentials: "include"`, the error envelope as `ApiError`), `src/data/index.ts` on the API, the `products` stock columns that migration 4 moves into `product_sizes`, and the generic reader that this story removes `orders` from.

**Lands after, and relies on this story**

- **"Order page and Pay now"** (added directly after this item) reads `GET /v1/orders/{id}` with the token this story stores, and adds `POST /v1/orders/{id}/payment`.
- **"Track an order by number and phone"** reads the same browser store and mints tokens in `order_tokens`.
- **"Online payment with VNPay"** implements the provider interface, adds the key check to `GET /v1/payment-options` and to placing (503), and the 30-minute expiry.
- **"Dealer sign-in and protected admin area"** and **"Customer accounts"** add the session paths to `GET /v1/orders` and `GET /v1/orders/{id}`, and the account link on placing.

**External accounts, keys, data:** none are needed to build or verify this story. No VNPay key is requested here; the VNPay story declares `VNPAY_TMN_CODE`, `VNPAY_HASH_SECRET` and the notice address. Test data: the seeded shoes and 14 seeded orders; the concurrency cases create a shoe size with the pair count they need.

**Known consequences, not defects**

- While this item and the next are not both merged, `/orders/:id` is the design's mock page and cannot show a real order.
- The admin screens that list or open orders (`AdminOrders`, `AdminOrder`, `AdminOverview`) read nothing from the API until the dealer sign-in story, because `GET /v1/orders` answers 401 to everyone (ORD-008 item 5; `services.md` §6).
- `src/lib/mock/orders.ts` is still imported by `Order.tsx` and `Track.tsx`, so CI's "mock still imported" warning stays on until those stories remove it.
- Files this story must not touch: `src/theme.css`, `src/index.css`, `src/content.json`, `package.json`, `eslint.config.js`, `src/App.tsx` (the `/checkout` and `/orders/:id` routes already exist), `Product.tsx`, `Order.tsx`, `Track.tsx`.

## 9. Decisions recorded here

- Online payment through VNPay is the only way to pay at launch; every order starts unpaid ("awaiting payment") and only the provider's own confirmation will ever mark one paid. No `cod` value is accepted.
- Delivery is free on every order: one option, `standard`; the stored fee is always 0; a request that names another option is refused and a fee in a request is ignored.
- The browser's prices, fee and totals are never trusted; the total shown at checkout and the total stored are both the sum of the pairs at the shop's price.
- Open question 4 of the SRS (the design's words promise a text message that no story builds) stays with the owner; this story keeps the design's checkout wording for it and builds no message.
