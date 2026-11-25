# Test cases — Checkout and placing an order

Module: `orders` · Plan item 5 · Requirements: ORD-001, ORD-002, ORD-003, ORD-004, ORD-005, ORD-006, ORD-009, ORD-010 and the privacy half of ORD-008
Story: `docs/orders/stories/checkout-and-placing-an-order.md` — criteria SC-1 … SC-37 (one or more cases each; the table at the end walks the list)
Contract: `docs/architecture/services.md` §2.3 (error envelope), §2.5 (validation boundary), §2.6 (idempotency), §3.5 (`GET /v1/payment-options`), §3.10 (`POST /v1/orders`), §3.11 (`GET /v1/orders/{id}`), §3.14 (`GET /v1/orders`)

**Risk level: high.** This item writes money-bearing records, takes goods off a shared shelf under concurrency, stores customers' phones and addresses, and decides who may read them. Every total, boundary, race, refusal and privacy rule is therefore written out value by value; a copy-only line is one read.

## How these cases are observed

- **Windows.** Default 1280×800. Phone cases say 390×844 (and 320×640 where the criterion names it).
- **Addresses.** The API is read through the gateway as `/api/v1/...` (the backend itself mounts `/v1/...`). Pages are `/checkout`, `/shop/<id>`, `/orders/<id>`.
- **Money.** `vi-VN` writes `3.090.000 ₫`; the space before `₫` may be a no-break space, compare with either. API amounts are integers of VND.
- **Selectors the app already carries.** `#checkout-name`, `#checkout-phone`, `#checkout-address`, `#checkout-city`, `#checkout-note`; `[data-testid='place-order']`, `[data-testid='checkout-total']`, `[data-testid='delivery-standard']`, `[data-testid='pay-vnpay']`, `[data-testid='bag-count']`, `[data-testid^='size-']`, `[data-testid='add-to-bag']`; the summary is `aside dl` with rows `aside dl > div:nth-child(1)` (Subtotal), `:nth-child(2)` (Delivery), `:nth-child(3)` (Total), each with `dt` and `dd`. No Tailwind class is relied on. Where a case says "the alert" it means the element with `role="alert"` that holds the message.
- **FORM** (the details typed in every UI case that places an order unless it says otherwise): name `Lan Nguyen`, phone `0912 345 678`, street address `12 Example Street, Ward 1`, city `Da Nang`, note `Call before delivering.`
- **REQ** (the request body of every API case unless it says otherwise), sent as `POST /api/v1/orders` with `Content-Type: application/json` and a header `Idempotency-Key: tc-<case>-<n>` (8–100 characters of `[A-Za-z0-9._-]`; a fresh key per request unless the case reuses one):

```json
{ "items": [ { "product": "samba-og-w-core-black", "size": "38", "qty": 1 } ],
  "delivery": "standard", "payment": "vnpay",
  "customer": "Lan Nguyen", "phone": "0912 345 678",
  "address": "12 Example Street, Ward 1", "city": "Da Nang", "note": "" }
```

- **SHELF(product, size, n)** — arrange with `set_db_state`: `UPDATE product_sizes SET pairs = <n> WHERE product_id = '<product>' AND size = '<size>'`. Read the old value first and put it back at the end of the case. The shoe's public record (`GET /api/v1/products/<id>`) then carries `sizes`, `sizes_low` (1–2 pairs), `sizes_out` (0 pairs) and `stock` (the sum of all sizes). The exact per-size count is not public (the dealer view is not available until the dealer sign-in story), so a case that needs "pairs left" reads it with `set_db_state` as `SELECT pairs FROM product_sizes …` and also from the public `stock` figure's change.
- **PRICE(product, v)** — `UPDATE products SET price = <v> WHERE id = '<product>'`; put the old price back at the end. The catalogue's cheapest pair is 2.290.000 ₫, so a basket under 2.000.000 ₫ needs this.
- **BAG(lines)** — the browser's bag is the `localStorage` entry `agentteam:cart`, a JSON array of `{id, name, price, image, variant, qty}` where `variant` is `EU <size>` (for example `EU 38`, `EU 36 2/3`). Write the entry with `interact_page`, then load `/checkout`. A line's saved `price` is whatever the case says; the page must use the shop's current price. Shoes used: `samba-og-w-crystal-white` (3.090.000 ₫), `samba-og-w-core-black` (3.290.000 ₫), `samba-og-kids-cloud-white` (2.290.000 ₫), `samba-og-w-wonder-quartz` (2.790.000 ₫).
- **RESET** (end of every group that placed orders): `DELETE FROM orders WHERE is_sample = false` (items and tokens go with them); `UPDATE counters SET value = 24817 WHERE name = 'order_number'`; every SHELF and PRICE put back; any renamed table renamed back; the database made available again. A case that needs the number to be exactly `DOI-24818` starts from RESET.
- **SEEDTOKEN(order)** — to read a seeded order with a token (nobody can read orders as the dealer yet): `INSERT INTO order_tokens (order_id, token_hash) VALUES ('<order>', sha256('tc-seed-token-0001'::bytea))`, then send `X-Order-Token: tc-seed-token-0001`. Remove the row afterwards. (This assumes the token is hashed as its text; if the backend hashes some other form, the arranged row will not open the order and the case is UNVERIFIED, not failed — the story's backend test covers it.)
- **Parallel requests.** "At the same time" means the requests are issued in one batch (several `fetch_url` calls in one round, or one `interact_page` that sends them with `Promise.all`). The case names how many.
- **Stubbing in the browser.** Cases that say "the answer is stubbed" intercept the page's request to the named API path and answer it with the body the case gives (`interact_page` with request interception). A case that needs the browser offline sets the page's network to offline after it loaded. Where the instrument cannot intercept or emulate, that case is UNVERIFIED and the `Check:` line says which instrument was missing.
- **No instrument sets the shop's clock.** Cases whose answer depends on the moment of placing either compute the expected value from the `placed_at` the shop returned, or are `manual` and say so.

## Ambiguities found in the documents (reported, not silently resolved)

1. **Database unreachable: 500 or 503?** SC-21 and `services.md` §2.3/§3.10 give `internal` 500 for "a database that cannot be reached" on `POST /v1/orders`; `unavailable` 503 is only `/healthz` and a shutting-down process. These cases follow the story and the contract (500). If the backend answers 503 `unavailable` to a refused database on this endpoint, TC-36 fails against the story.
2. **The delivery row's label.** The `delivery` record named `standard` has `name` "Standard" (SITE-016); ORD-001 AC-3 and SC-3 say the row reads "Free delivery". The cases pin the visible text "Free delivery" plus the record's note and arrival time; they do not say whether the label comes from the record or from the site's words.
3. **Same key, different bag.** `services.md` §2.3 lists `conflict` for "an idempotency key reused with different details", but §2.6 and §3.10 define the conflict only for a different phone. The cases pin the phone rule (409 for other digits, replay for the same digits written differently) and write no case for "same key, same phone, other pairs", which the contract does not fix.
4. **The browser's order store has no name in the story.** SC-32 says "the browser's storage holds the order's id, number, placed time and token and nothing else about it" under "a key of its own". The cases read every `localStorage` and `sessionStorage` entry and look for the entry that holds the order id; they do not pin its key name, and they pin that `doi:orders` (the design's mock key) is untouched.
5. **Exact sentences for limit refusals.** The story says a refusal "names the limit" (500, 100, 20, 200) without fixing the sentence. The cases assert the field key, status, code and that the sentence contains the limit's number.
6. **Empty list of pairs.** SC-16 says 422 and `services.md` §3.10 says `errors.items` for an empty or oversized list; the cases expect the key `items` for an empty list and `items.<index>` for a bad line.

---

## A. The checkout page (SC-1 … SC-8)

**Scenario**: TC-01 — one pair in the bag: heading, steps, secure line, five labelled fields
**Given**: BAG with one line of `samba-og-w-crystal-white`, variant `EU 38`, qty 1; 1280×800.
**When**: `/checkout` loads.
**Then**: The `h1` reads exactly `Checkout`. The three steps read `01 Contact and address`, `02 Delivery`, `03 Payment` (the number and the words may sit in separate spans; the visible text of each step is the number then the words) and are visible. The line `Your details travel over a secure connection.` is visible. Five fields each have a visible `label` tied to them: `#checkout-name` ← `Full name`; `#checkout-phone` ← `Phone`; `#checkout-address` ← `Street address, ward, district`; `#checkout-city` ← `City or province`; `#checkout-note` ← `Note for the courier (optional)`. `#checkout-note` is a `textarea` with `rows` 2; the other four are single-line `input`s.
Traces: SC-1 (ORD-001 AC-1)
Check: interact_page

**Scenario**: TC-02 — at 390 px the step line is gone, the form is one column, the phone field opens a phone keypad
**Given**: The same bag; a 390×844 window.
**When**: `/checkout` loads; the visible text, the position of each field and the `#checkout-phone` attributes are read.
**Then**: The text `01 Contact and address` / `02 Delivery` / `03 Payment` as a step line is not visible (the ordered list that holds the three steps has computed `display: none`). The left edge of `#checkout-name` equals the left edge of `#checkout-phone` and the top of `#checkout-phone` is below the bottom of `#checkout-name` (one column, no side-by-side fields). `#checkout-phone` has `type="tel"` and `inputmode="tel"`. `document.documentElement.scrollWidth` is at most 390.
Traces: SC-2 (ORD-001 AC-2)
Check: interact_page

**Scenario**: TC-03 — at 320 px the page does not scroll sideways, with a pair in the bag and with an empty bag
**Given**: A 320×640 window; (a) the BAG of TC-01, (b) an empty bag (no `agentteam:cart` entry).
**When**: `/checkout` loads in each case and `document.documentElement.scrollWidth` and `window.innerWidth` are read; then the page is scrolled to its end.
**Then**: In both, `scrollWidth` is at most 320 (equal to `innerWidth`); no horizontal scroll position other than 0 can be reached (`window.scrollX` stays 0 after `scrollTo(500, 0)`); the one-column form of TC-02 holds in (a).
Traces: SC-2 (ORD-001 AC-2; SRS §6 Responsive)
Check: interact_page

**Scenario**: TC-04 — step 02 is one delivery row, selected, not a choice; "Express" appears nowhere
**Given**: BAG of TC-01; 1280×800; `GET /api/v1/delivery` read first (one record, `standard`, `fee` 0, a `note` and `days`).
**When**: `/checkout` loads and step 02 is read.
**Then**: Exactly one element matches `[data-testid^='delivery-']` and it is `[data-testid='delivery-standard']`. It holds an icon (an `svg`), the text `Free delivery`, the `standard` record's `note` word for word, the record's `days` (for example `2–4 working days`), and at its right the text `Free`. Its radio input is checked and no other radio with `name="delivery"` exists. The word `Express` (any case) appears nowhere in the page's text, and no text `0 ₫` appears in step 02.
Traces: SC-3 (ORD-001 AC-3, AC-4)
Check: interact_page

**Scenario**: TC-05 — the summary line uses the shop's current price, not the bag's saved price
**Given**: BAG with one line of `samba-og-w-crystal-white`, variant `EU 38`, qty 1, with the saved `price` 3000000; the shop's price is 3.090.000 ₫ (`GET /api/v1/products/samba-og-w-crystal-white` → `price` 3090000); 1280×800.
**When**: `/checkout` loads and the summary under the heading `Your order` is read.
**Then**: The pair's line shows a picture (an `img` whose `naturalWidth` is above 0), its name in capitals (computed `text-transform: uppercase` or the text itself in capitals, matching the record's `name`), the text `EU 38 · Qty 1`, and the line total `3.090.000 ₫` (not `3.000.000 ₫`). Subtotal reads `3.090.000 ₫`. With the same line at `qty` 2 the line total reads `6.180.000 ₫` and `EU 38 · Qty 2`.
Traces: SC-4 (ORD-001 AC-5, AC-6)
Check: interact_page

**Scenario**: TC-06 — Delivery reads "Free" and Total equals Subtotal under, at and over 2.000.000 ₫
**Given**: Five bags, each a fresh BAG, 1280×800: (a) one pair of `samba-og-w-crystal-white` at the shop's 3.090.000 ₫; (b) that pair plus one `samba-og-kids-cloud-white` with PRICE 1500000 (4.590.000 ₫ in all); (c) one `samba-og-kids-cloud-white` with PRICE 1500000; (d) the same shoe with PRICE 1999999; (e) the same shoe with PRICE 2000000. PRICE put back afterwards.
**When**: `/checkout` loads for each and the three rows of `aside dl` are read.
**Then**: (a) Subtotal `3.090.000 ₫`, Delivery `Free`, Total `3.090.000 ₫`. (b) Subtotal `4.590.000 ₫`, Delivery `Free`, Total `4.590.000 ₫`. (c) `1.500.000 ₫` / `Free` / `1.500.000 ₫`. (d) `1.999.999 ₫` / `Free` / `1.999.999 ₫`. (e) `2.000.000 ₫` / `Free` / `2.000.000 ₫`. In none is Delivery `0 ₫` or any amount, and `[data-testid='checkout-total']` equals the Total row.
Traces: SC-4 (ORD-001 AC-7, AC-8, AC-5, boundary)
Check: interact_page

**Scenario**: TC-07 — step 03 is VNPay alone, selected, read from the API; the next-step sentence is the new wording
**Given**: BAG of TC-01; 1280×800; `GET /api/v1/payment-options` read first (one item `vnpay` with `name`, `note`).
**When**: `/checkout` loads and step 03 and the line under the totals are read.
**Then**: Exactly one element matches `[data-testid^='pay-']` and it is `[data-testid='pay-vnpay']`; its radio is checked; it holds an icon, the text `VNPay — ATM, Visa, Mastercard` and the item's `note` word for word. No element with `data-testid` `pay-cod`, `pay-bank` or `pay-momo` exists and the page's text contains none of `Cash on delivery`, `Bank transfer`, `MoMo` (any case) although `GET /api/v1/payments` returns those rows. Under the totals the information line reads exactly `After you press Place order you pay on the VNPay — ATM, Visa, Mastercard page; we pack your order after the payment is confirmed.` and the page's text contains neither `marked paid at once` nor `packed today`.
Traces: SC-5 (ORD-001 AC-9, AC-10; ORD-006 AC-1, AC-3)
Check: interact_page

**Scenario**: TC-08 — "Place order" has a lock icon and the exchanges sentence under it
**Given**: BAG of TC-01; 1280×800.
**When**: `/checkout` loads and the foot of the summary is read.
**Then**: `[data-testid='place-order']` reads `Place order`, contains an `svg` before the text (the lock), and is enabled. The sentence `By placing the order you accept the delivery information and that exchanges are not accepted.` is the text immediately after the button.
Traces: SC-6 (ORD-001 AC-11)
Check: interact_page

**Scenario**: TC-09 — while an order is being placed the button says so, is disabled, and a second press sends nothing
**Given**: RESET; SHELF(`samba-og-w-crystal-white`, `38`, 5); BAG of TC-01; FORM typed; the answer of `POST /api/v1/orders` is held for 3 seconds (interception) before it is passed on to the shop; a request log on `/api/v1/orders`.
**When**: `[data-testid='place-order']` is pressed, then pressed again twice at once (a double-click) and Enter is pressed in `#checkout-city`, all within the first second.
**Then**: Within 200 ms of the first press the button's text is `Placing your order…` and its `disabled` property is true. The request log holds exactly one `POST /api/v1/orders` after the three further presses and after the answer arrives. The shop holds exactly one order for the bag (`SELECT count(*) FROM orders WHERE is_sample = false` reads 1 after a RESET before the case) and the pair left the shelf once (the shoe's `stock` is one lower).
Traces: SC-6 (ORD-001 AC-11; ORD-010 AC-6, item 4)
Check: interact_page

**Scenario**: TC-10 — an empty bag says so and offers up to four live new shoes
**Given**: A fresh browser with no `agentteam:cart` entry; 1280×800. `GET /api/v1/products?is_new=true&limit=100` read first (the live new shoes; five at the seed). Part (b): `UPDATE products SET status = 'draft' WHERE id = 'samba-og-w-core-black'` (a live new shoe becomes a draft), put back afterwards.
**When**: `/checkout` loads (a) with five live new shoes, (b) after the draft change.
**Then**: The heading reads `Nothing to check out yet.`; the line `Pick a pair — the size chart is one tap from every shoe.` is visible; a `Keep shopping` pill is a link to `/shop`; a row headed `New this month` shows exactly 4 shoe tiles in (a) (up to four, though five are new), each one a live new shoe from the API list; in (b) the draft shoe's name appears nowhere on the page and 4 tiles still show (the other new shoes). No form, no `[data-testid='place-order']` exists.
Traces: SC-7 (ORD-001 AC-12)
Check: render_url

**Scenario**: TC-11 — VNPay unavailable: the row stays, the alert shows, "Place order" is disabled; available: enabled
**Given**: BAG of TC-01; FORM typed. (a) The answer of `GET /api/v1/payment-options` is stubbed as `{"items":[{"id":"vnpay","name":"VNPay — ATM, Visa, Mastercard","note":"<the real note>","icon":"card","online":true,"is_available":false}],"total":1,"page":1,"limit":100}`. (b) The real answer (`is_available: true`).
**When**: `/checkout` loads and, in (a), `[data-testid='place-order']` is pressed.
**Then**: (a) `[data-testid='pay-vnpay']` is still drawn; directly under it the text `Online payment is not available right now. Please try again later.` shows in an element with `role="alert"` with an alert icon (`svg`); `[data-testid='place-order']` has `disabled` true and the press sends no `POST /api/v1/orders`. (b) No such text, no alert, and the button's `disabled` is false.
Traces: SC-8 (ORD-001 AC-14; ORD-006 AC-2, AC-10)
Check: interact_page

---

## B. Details are checked (SC-9 … SC-12)

**Scenario**: TC-12 — an empty or blank name, address or city shows the message and sends nothing
**Given**: BAG of TC-01; a request log on `/api/v1/orders`. Three runs, each with FORM typed except one field: (a) name empty; (b) street address `   ` (three spaces); (c) city `   `. The phone is valid.
**When**: `[data-testid='place-order']` is pressed in each run.
**Then**: In each run an element with `role="alert"` is present and holds, with an alert icon (`svg`) beside it, exactly the line `Fill in your name, street address and city.`; the phone message is not shown; the log holds no `POST /api/v1/orders`; the URL stays `/checkout`; the bag still counts 1 (`[data-testid='bag-count']`).
Traces: SC-9 (ORD-002 AC-1, AC-2)
Check: interact_page

**Scenario**: TC-13 — empty name and phone `12` show both messages, one per line
**Given**: BAG of TC-01; FORM typed except name empty and phone `12`.
**When**: `[data-testid='place-order']` is pressed.
**Then**: The `role="alert"` region holds exactly two lines, in this order: `Fill in your name, street address and city.` then `Check the phone number — we text you when the pair is on its way.`, each with its own alert icon and on its own line (two separate list items or blocks, different vertical positions); no request is sent.
Traces: SC-9 (ORD-002 AC-7)
Check: interact_page

**Scenario**: TC-14 — the phone rule: 8 digits refused, 9 digits and 11 digits accepted, punctuation not counted
**Given**: BAG of TC-01 each time (fresh bag per row); FORM typed except the phone; a request log on `/api/v1/orders`.
**When**: The phone is typed and `[data-testid='place-order']` is pressed, for each row of: `0912 34` (6 digits); `09123456` (8 digits); `(091) 234-56` (8 digits, punctuation does not count); `091234567` (9 digits); `+84 912 345 678` (11 digits).
**Then**: For the first three rows the line `Check the phone number — we text you when the pair is on its way.` shows in a `role="alert"` region and the log has no `POST /api/v1/orders`. For the last two rows that line does not show, exactly one `POST /api/v1/orders` is sent and answered 201, and the page leaves `/checkout`. RESET afterwards.
Traces: SC-10 (ORD-002 AC-3, AC-4, AC-5, AC-6)
Check: interact_page

**Scenario**: TC-15 — after a failed validation every typed value is still in its field
**Given**: BAG of TC-01; typed: name `Lan Nguyen`, phone `12`, address `12 Example Street, Ward 1`, city `Da Nang`, note with two lines `Call before delivering.` ⏎ `Gate code 4821`.
**When**: `[data-testid='place-order']` is pressed (validation fails on the phone).
**Then**: `#checkout-name`, `#checkout-phone`, `#checkout-address`, `#checkout-city` and `#checkout-note` hold exactly the typed values (the note keeps its line break); the phone message shows.
Traces: SC-11 (ORD-002 AC-11)
Check: interact_page

**Scenario**: TC-16 — the shop refuses empty and blank values for a request that skips the page
**Given**: RESET; shoe `samba-og-w-core-black` size `38` with SHELF 5; the shoe's public `stock` read as S0.
**When**: Six REQ requests are sent, each with one change: (a) `address` `""`; (b) `address` `"   "`; (c) `customer` `""`; (d) `customer` `"   "`; (e) `city` `""`; (f) `phone` `""`.
**Then**: Each answers 422 with `{"error":"invalid","message":<a sentence>,"errors":{…}}` whose `errors` has the key of the field at fault — (a), (b) `address`; (c), (d) `customer`; (e) `city`; (f) `phone` — with a sentence. No response has a `token` or `order`.
Traces: SC-12 (ORD-002 AC-8, item 3)
Check: fetch_url

**Scenario**: TC-17 — the note limit: 500 characters is accepted, 501 is refused naming 500
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5).
**When**: REQ is sent with `note` of (a) 500 characters (`x` repeated); (b) 501 characters; (c) 500 characters of `ạ` (500 characters, 1,500 bytes).
**Then**: (a) 201 with an `order`. (b) 422 `invalid` with `errors.note` whose sentence contains `500`; no order. (c) 201 (limits count characters, not bytes). RESET afterwards.
Traces: SC-12 (ORD-002 AC-9, AC-10)
Check: fetch_url

**Scenario**: TC-18 — the other limits: name 100/101, phone 20/21 and under 9 digits, address 200/201, city 100/101
**Given**: SHELF(`samba-og-w-core-black`, `38`, 20); REQ otherwise.
**When**: REQ is sent with, one change per request: `customer` of 100 `a` (accepted), of 101 `a`, of 100 `Đ` (100 characters, 200 bytes); `phone` `09123456789012345678` (20 characters), `091234567890123456789` (21), `0912 3456` (8 digits); `address` of 200 characters, of 201; `city` of 100 characters, of 101.
**Then**: 201 for: name of 100 `a`; name of 100 `Đ`; phone of 20 characters; address of 200; city of 100. 422 `invalid` for the others, with `errors.customer` containing `100`; `errors.phone` containing `20` (the 21-character phone); `errors.phone` containing `9` (the 8-digit phone); `errors.address` containing `200`; `errors.city` containing `100`. Each refusal creates no order. RESET afterwards.
Traces: SC-12 (ORD-002 item 3, boundary)
Check: fetch_url

**Scenario**: TC-19 — a refused request creates no order and takes no pair
**Given**: RESET (next number `DOI-24818`); SHELF(`samba-og-w-core-black`, `38`, 5); `GET /api/v1/products/samba-og-w-core-black` read: `stock` S0 and the three lists L0.
**When**: Three refused requests are sent (an empty address, a note of 501 characters, a phone of 8 digits); then the product is read again; then one valid REQ is sent.
**Then**: The product's `stock` equals S0 and `sizes`, `sizes_low`, `sizes_out` equal L0 after the refusals. The valid request's order is `DOI-24818` / `doi-24818` (the refusals used no number).
Traces: SC-12 (ORD-002 "Invalid input"; ORD-004 AC-7)
Check: fetch_url

---

## C. The shop computes everything (SC-13 … SC-22)

**Scenario**: TC-20 — numbering continues from the highest number held
**Given**: RESET (the highest number held is `DOI-24817`); SHELF(`samba-og-w-core-black`, `38`, 10).
**When**: REQ is sent twice in turn (two keys).
**Then**: The first answers 201 with `order.id` `doi-24818`, `order.number` `DOI-24818`, `order.status` `placed`, `order.paid` `false`, header `Location: /v1/orders/doi-24818`, `Cache-Control: no-store`. The second answers 201 with `doi-24819` / `DOI-24819`.
Traces: SC-13 (ORD-003 AC-1)
Check: fetch_url

**Scenario**: TC-21 — two orders at the same moment both succeed with different numbers
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 10).
**When**: Two REQs with different keys are sent in one parallel batch; then ten more in one parallel batch.
**Then**: Both of the first two answer 201 and their `order.number`s differ; of the ten, all answer 201 and the ten numbers are pairwise different, with no gap (the twelve numbers are `DOI-24818` … `DOI-24829` in some order).
Traces: SC-13 (ORD-003 AC-2)
Check: fetch_url

**Scenario**: TC-22 — the stored total is the sum of the shop's prices, delivery fee 0
**Given**: SHELF(`samba-og-w-crystal-white`, `38`, 10) and SHELF(`samba-og-kids-cloud-white`, `38`, 10); PRICE put back afterwards.
**When**: REQ is sent for (a) `samba-og-w-crystal-white` `38` qty 2 (price 3.090.000 ₫); (b) `samba-og-kids-cloud-white` `38` qty 1 with PRICE 1500000; (c) the shoe of (b) with PRICE 1999999, qty 1; (d) PRICE 2000000, qty 1; (e) `samba-og-w-crystal-white` `38` qty 1 plus `samba-og-kids-cloud-white` `38` qty 3 with PRICE 1500000 (a basket over 2.000.000 ₫ with three pairs of the cheap shoe).
**Then**: (a) `order.total` `6180000`, `order.delivery_fee` `0`. (b) `1500000`, fee `0`. (c) `1999999`, fee 0. (d) `2000000`, fee 0. (e) `7590000` (3.090.000 + 3 × 1.500.000), fee 0. In every case `total` equals Σ `items[].unit_price × items[].qty` exactly.
Traces: SC-14 (ORD-003 AC-3, AC-4, AC-18)
Check: fetch_url

**Scenario**: TC-23 — a unit price, a fee and a total in the request are ignored
**Given**: SHELF(`samba-og-w-crystal-white`, `38`, 10).
**When**: REQ is sent for one pair of `samba-og-w-crystal-white` `38` with the tolerated extras `"price": 1` and `"unit_price": 1` on the line and `"delivery_fee": 50000` and `"total": 1` on the body.
**Then**: 201; `order.total` `3090000`; `order.delivery_fee` `0`; `order.items[0].unit_price` `3090000`. A further request that adds any other undefined field (`"discount": 10`) is covered by TC-76 (400).
Traces: SC-14 (ORD-003 AC-6, AC-9; contract: tolerated extras)
Check: fetch_url

**Scenario**: TC-24 — a price that changes between the page and the order: the order is saved at the price at saving
**Given**: SHELF(`samba-og-w-crystal-white`, `38`, 10); the product read: `price` 3090000 (what a page that had just loaded would show).
**When**: PRICE(`samba-og-w-crystal-white`, 3190000) is applied; then REQ for one pair of that shoe in `38` is sent; the order is read back with `GET /api/v1/orders/<id>` and `X-Order-Token: <token>`; PRICE put back.
**Then**: The answer's `order.total` is `3190000` and `order.items[0].unit_price` is `3190000`; the read-back shows the same.
Traces: SC-14 (ORD-003 "Conflict" row)
Check: fetch_url

**Scenario**: TC-25 — the Total the page showed is the total the shop stored
**Given**: SHELF(`samba-og-w-crystal-white`, `38`, 10) and SHELF(`samba-og-kids-cloud-white`, `38`, 10); BAG with one `samba-og-w-crystal-white` `EU 38` qty 1 and one `samba-og-kids-cloud-white` `EU 38` qty 2, saved prices 1 (the page must not use them); FORM typed; no price change in between.
**When**: `/checkout` loads and `[data-testid='checkout-total']` is read; `[data-testid='place-order']` is pressed; the placing answer's `order.total` is read; the token the browser stored is read and `GET /api/v1/orders/<id>` is sent with `X-Order-Token: <that token>`.
**Then**: The Total on the page is `7.670.000 ₫` (3.090.000 + 2 × 2.290.000); the placing answer's `order.total` is `7670000`; the read-back `total` is `7670000`.
Traces: SC-15 (ORD-001 AC-13; ORD-003 item 2)
Check: interact_page

**Scenario**: TC-26 — a delivery other than `standard` is refused
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `stock` S0 read.
**When**: REQ is sent with `delivery` set to (a) `express`; (b) `overnight` (unknown id); (c) `""`; (d) `Standard` (wrong case).
**Then**: Each answers 422 `invalid` with `errors.delivery` and a sentence; no `order`; the product's `stock` equals S0 afterwards.
Traces: SC-16 (ORD-003 AC-5, AC-15)
Check: fetch_url

**Scenario**: TC-27 — a way to pay that is not offered is refused
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `stock` S0 read; `GET /api/v1/payments` still lists `cod`, `bank`, `vnpay`, `momo`.
**When**: REQ is sent with `payment` set to (a) `cod`; (b) `bank`; (c) `momo`; (d) `paypal` (unknown); (e) `""`; and once with `payment` `vnpay`.
**Then**: (a)–(e) answer 422 `invalid` with `errors.payment`; no order; `stock` equals S0 after them. The `vnpay` request answers 201 (and `stock` is then S0 − 1). RESET afterwards.
Traces: SC-16 (ORD-003 AC-16; ORD-006 AC-5)
Check: fetch_url

**Scenario**: TC-28 — a bad line or an empty list of pairs is refused naming the line
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5) and SHELF(`samba-og-w-crystal-white`, `38`, 5); `stock` of each shoe read.
**When**: REQ is sent with `items` set to: (a) one line qty 0; (b) one line qty −1; (c) one line of size `99` (a size the shoe does not list); (d) `[]`; (e) two lines, the first valid and the second qty 0; (f) the same `product` and `size` twice; (g) a line with qty 1001; (h) 51 distinct lines.
**Then**: Each answers 422 `invalid`: (a), (b), (c), (g) with `errors["items.0"]`; (d) with `errors.items`; (e) with `errors["items.1"]` (and no key for the valid line); (f) with `errors["items.1"]`; (h) with `errors.items`. No order is created and the `stock` of both shoes is as read, also for (e), whose first line was valid.
Traces: SC-16 (ORD-003 AC-17, "Invalid input" row; contract §3.10 `items`)
Check: fetch_url

**Scenario**: TC-29 — the expected date follows the rule for the moment of placing
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `GET /api/v1/shop` read: `standard_days` `2–4` (last number 4), `dispatch_cutoff` `15:00`.
**When**: REQ is sent once; `placed_at` (RFC 3339 with `+07:00`) and `eta` (`YYYY-MM-DD`) are read.
**Then**: From `placed_at`'s date and time (Vietnam time) the verifier derives the base day — the same day when it is Monday to Friday and the time is before 15:00, otherwise the next Monday-to-Friday day — and adds 4 working days skipping Saturdays and Sundays; `order.eta` equals that date. (Monday 10:00 → that Friday; Monday 16:00 → next Monday; Friday 15:30 → Friday of the following week; Saturday or Sunday → Friday of the following week; the derivation applies whichever of these `placed_at` falls in.) The date has no time part.
Traces: SC-17 (ORD-003 AC-7, AC-8, AC-10)
Check: fetch_url

**Scenario**: TC-30 — the days come from the standard option's last number
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `UPDATE shop_terms SET standard_days = '1–2'` (put back `2–4` afterwards).
**When**: REQ is sent; `placed_at` and `eta` are read.
**Then**: `eta` equals the base day of TC-29 plus 2 working days (not 4). With `standard_days` put back to `2–4` a further request gives base + 4.
Traces: SC-17 (ORD-003 item 3)
Check: fetch_url

**Scenario**: TC-31 — the three named moments: Monday 10:00, Monday 16:00, Saturday
**Given**: A Go test (or any test) that places an order with the shop's clock fixed at Vietnam time (+07:00) on 2026-11-23 (a Monday) at 10:00, 16:00 and 14:59 and 15:00, on 2026-11-28 (a Saturday) and on 2026-11-29 (a Sunday).
**When**: The test's results are read (the verifier runs the backend tests and reads the test that fixes the clock).
**Then**: Monday 10:00 → `2026-11-27` (Friday); Monday 14:59 → `2026-11-27`; Monday 15:00 → `2026-11-30` (Monday); Monday 16:00 → `2026-11-30`; Saturday 28/11 any time → `2026-12-04` (the Friday of the following week); Sunday → `2026-12-04`.
Traces: SC-17 (ORD-003 AC-7, AC-8, AC-10)
Check: manual — no instrument sets the running shop's clock, so the three fixed moments can only be proven by a backend test with an injected clock; TC-29 proves the same rule on whichever moment the verification runs at.

**Scenario**: TC-32 — a two-shoe order keeps two pairs at the price they were sold at
**Given**: SHELF(`samba-og-w-crystal-white`, `38`, 10) and SHELF(`samba-og-w-core-black`, `38`, 10).
**When**: REQ is sent with two lines: `samba-og-w-crystal-white` `38` qty 1 and `samba-og-w-core-black` `38` qty 2; the order is read back with its token (read A). Then PRICE(`samba-og-w-crystal-white`, 3500000) and PRICE(`samba-og-w-core-black`, 3600000) are applied and the order is read again (read B); PRICE put back.
**Then**: Read A: `items` has 2 entries in the order sent — `{product: "samba-og-w-crystal-white", size: "38", qty: 1, unit_price: 3090000}` and `{product: "samba-og-w-core-black", size: "38", qty: 2, unit_price: 3290000}` (each with `name`, `colourway`, `image`); `total` `9670000`; `product`, `size`, `qty` at the order's top level are the first pair's. Read B: the same two `unit_price`s and the same `total` `9670000`.
Traces: SC-18 (ORD-003 AC-11, AC-13, item 4)
Check: fetch_url

**Scenario**: TC-33 — each seeded order is one pair line; `doi-24817` is one pair of `samba-og-w-crystal-white`, EU 39 1/3
**Given**: SEEDTOKEN for each of the 14 seeded orders (`SELECT id FROM orders WHERE is_sample` gives them; `DOI-24804` … `DOI-24817`); `GET`s with `X-Order-Token: tc-seed-token-0001`.
**When**: `GET /api/v1/orders/<id>` is sent for each; the seed record's `qty` and `total` are read with `set_db_state` (`SELECT id, total FROM orders WHERE is_sample`).
**Then**: Each answers 200 with `items` of exactly one entry whose `qty` is the seeded order's quantity, `unit_price × qty` equal to the order's `total`, and `delivery_fee` 0. `doi-24817` answers `items[0].product` `samba-og-w-crystal-white`, `items[0].size` `39 1/3`, `items[0].qty` 1, `items[0].unit_price` 3090000, `total` 3090000, and the top-level `product`, `size`, `qty` the same. No answer carries `phone`, `email`, `address` or `note` (the seeded phone `0938 410 552` appears nowhere in any body). The token rows are removed afterwards.
Traces: SC-19 (ORD-003 AC-12, item 5)
Check: fetch_url

**Scenario**: TC-34 — the placing answer: 201, the public order, a token, a payment address
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 5); a request with REQ and `note` `Gate code 4821`.
**When**: The answer is read.
**Then**: Status 201, `Content-Type` `application/json; charset=utf-8`, `Location` `/v1/orders/doi-24818`, `Cache-Control: no-store`. The body has exactly the keys `order`, `token`, `payment_url`. `order` has `id` `doi-24818`, `number` `DOI-24818`, `placed_at` matching `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+07:00$`, `status` `placed`, `paid` `false`, `total` `3290000`, `delivery_fee` `0`, `delivery` `standard`, `payment` `vnpay`, `customer` `Lan Nguyen`, `city` `Da Nang`, `items` of one entry with `product`, `name`, `colourway`, `image`, `size` `38`, `qty` 1, `unit_price` 3290000. `order` has none of the keys `phone`, `email`, `address`, `note`, and the strings `0912`, `12 Example Street` and `Gate code` appear nowhere in the body. `token` is a string matching `^[A-Za-z0-9_-]{22,}$`. `payment_url` is the string `""` (not `null`, not absent).
Traces: SC-20 (ORD-003 AC-14; ORD-006 AC-9)
Check: fetch_url

**Scenario**: TC-35 — tokens are long, URL-safe and never repeat
**Given**: SHELF(`samba-og-w-core-black`, `38`, 40).
**When**: REQ is sent 30 times with different keys; the 30 `token`s are read.
**Then**: Every token matches `^[A-Za-z0-9_-]{22,}$`; all 30 are pairwise different; no token is a prefix or a suffix of another; the first character of the tokens is not the same in all 30 (a counter or a fixed prefix would fail this); the average length is at least 22 characters (at least 128 bits when decoded).
Traces: SC-20 (ORD-003 AC-14, item 7)
Check: fetch_url

**Scenario**: TC-36 — with the database unreachable, placing answers 500 with a generic body
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 5); `stock` S0 read; then `set_db_available` takes the database away while the service keeps running.
**When**: REQ (valid) is sent; then the database is made available again; the product and the next placing are read.
**Then**: The answer is 500 with `{"error":"internal","message":<a short sentence>}` and no `errors` key; the body and the headers contain none of `SQL`, `pgx`, `postgres`, `connection refused`, `goroutine`, `.go:`, `0912`, `Example Street`, a table name or a stack trace; `X-Request-Id` is present. After the database is back, `stock` equals S0 and the next order gets the number it would have had (`DOI-24818` after RESET): nothing was saved and no pair left the shelf.
Traces: SC-21 (ORD-003 "Upstream failure" row; contract §2.3 `internal`)
Check: fetch_url

**Scenario**: TC-37 — a failure inside the transaction rolls everything back
**Given**: RESET; SHELF(`samba-og-w-core-black`, `38`, 5); `stock` S0 read; `ALTER TABLE order_items RENAME TO order_items_x` (put back afterwards), so the insert of the pairs fails after the pairs were taken and the number issued.
**When**: REQ (valid) is sent; the table is renamed back; the product is read; REQ is sent again.
**Then**: The first answers 500 `internal` with a generic body (as TC-36). After the table is back the product's `stock` equals S0, `sizes_low`/`sizes_out` are unchanged, and the second request answers 201 with `DOI-24818` (the failed attempt used neither a pair nor a number). Placing with `order_tokens` renamed instead gives the same results.
Traces: SC-21 (ORD-003 failure row; ORD-004 item 1 "either the whole order … or nothing")
Check: fetch_url

**Scenario**: TC-38 — 100 sequential orders answer within 1.5 s at p95
**Given**: SHELF(`samba-og-w-core-black`, `38`, 1000); the instrument reports the elapsed time of each response.
**When**: REQ is sent 100 times one after another (keys `tc-perf-001` … `tc-perf-100`, qty 1).
**Then**: All 100 answer 201; sorted by elapsed time, the 95th value is at most 1.5 s. RESET afterwards (the 100 orders are deleted and the shelf put back).
Traces: SC-22 (SRS §6 Performance)
Check: fetch_url — if the instrument returns no per-request time, the case is UNVERIFIED (instrument lacked: per-response latency).

---

## D. Pairs leave the shelf, and the stock shown stays true (SC-23 … SC-29)

**Scenario**: TC-39 — an order for 1 of 2 pairs is saved and 1 is left
**Given**: SHELF(`samba-og-w-core-black`, `38`, 2); the shoe's `stock` S0 read; `SELECT pairs` for that size reads 2 (and `38` is in the shoe's `sizes_low`).
**When**: REQ is sent for `samba-og-w-core-black` `38` qty 1; the shoe is read; `SELECT pairs` is read.
**Then**: 201; `stock` is S0 − 1; pairs left is 1 (read from the database); `38` is in `sizes_low`.
Traces: SC-23 (ORD-004 AC-1)
Check: fetch_url

**Scenario**: TC-40 — an order for exactly the pairs left is accepted and the size sells out
**Given**: SHELF(`samba-og-w-core-black`, `38`, 3); `stock` S0 read.
**When**: REQ is sent with qty 3; the shoe is read; REQ is sent again with qty 1.
**Then**: The first answers 201; `stock` is S0 − 3; pairs left 0; `38` is in `sizes_out` and not in `sizes_low`. The second answers 409 `stock_conflict` with `errors["items.0"]` `<shoe name> · EU 38 — sold out in your size.`
Traces: SC-23 (ORD-004 boundary "Quantity equals the pairs left")
Check: fetch_url

**Scenario**: TC-41 — two orders race for the last pair: one 201, one 409, never −1
**Given**: SHELF(`samba-og-w-core-black`, `38`, 1); `stock` S0 and the shoe's name N read.
**When**: Two REQs for qty 1 of `38` with different keys are sent in one parallel batch; the batch is repeated four times, each time re-arranging SHELF(…, 1).
**Then**: In every batch exactly one answers 201 and exactly one answers 409 `stock_conflict` with `errors["items.0"]` equal to `<N> · EU 38 — sold out in your size.`; afterwards pairs left is 0 (never −1: `SELECT pairs` is 0), `stock` is S0 − 1 and `38` is in `sizes_out`.
Traces: SC-24 (ORD-004 AC-2)
Check: fetch_url

**Scenario**: TC-42 — fifty simultaneous orders for a size holding 10 pairs sell exactly 10
**Given**: SHELF(`samba-og-w-core-black`, `38`, 10); `stock` S0 and the name N read; RESET.
**When**: 50 REQs for qty 1 of `38` with 50 different keys are sent in one parallel batch (50 `fetch_url` calls in one round, or one in-page `Promise.all`).
**Then**: Exactly 10 answer 201 and 40 answer 409 `stock_conflict`; every 409 has `errors["items.0"]` `<N> · EU 38 — sold out in your size.`; the 10 numbers are pairwise different; afterwards `SELECT pairs` is 0, no row of `product_sizes` has pairs below 0, `stock` is S0 − 10 and `38` is in `sizes_out`. If the instrument cannot send 50 requests at the same moment the case is UNVERIFIED (instrument lacked: a parallel batch of 50).
Traces: SC-24 (ORD-004 AC-2; SRS §6 Integrity)
Check: fetch_url

**Scenario**: TC-43 — more pairs asked than left: refused naming "only <n> left", the 2 pairs stay
**Given**: SHELF(`samba-og-w-core-black`, `38`, 2); `stock` S0, lists L0 and the name N read.
**When**: REQ is sent for `38` qty 3.
**Then**: 409 `{"error":"stock_conflict","message":<sentence>,"errors":{"items.0":"<N> · EU 38 — only 2 left in your size."}}`; `SELECT pairs` is still 2; `stock` is S0; the lists equal L0.
Traces: SC-25 (ORD-004 AC-4)
Check: fetch_url

**Scenario**: TC-44 — a bag whose second pair is sold out: only the second is named, the first pair's stock is unchanged
**Given**: SHELF(`samba-og-w-crystal-white`, `38`, 5) and SHELF(`samba-og-w-core-black`, `38`, 0); `stock` of both shoes read; RESET.
**When**: REQ is sent with two lines: `samba-og-w-crystal-white` `38` qty 1 (first), `samba-og-w-core-black` `38` qty 1 (second).
**Then**: 409 `stock_conflict` whose `errors` has exactly one key, `items.1`, with `<name of samba-og-w-core-black> · EU 38 — sold out in your size.`; the crystal-white shoe's `stock` and `38` pairs (5) are unchanged; the next successful placing is `DOI-24818`.
Traces: SC-25 (ORD-004 AC-5)
Check: fetch_url

**Scenario**: TC-45 — a shoe that is no longer in the shop, or never was
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `UPDATE products SET status = 'draft' WHERE id = 'samba-og-w-wonder-quartz'` (put back afterwards); the id `no-such-shoe` does not exist.
**When**: REQ is sent for (a) `samba-og-w-wonder-quartz` `38` qty 1 (a draft, as if set to draft after it went into the bag); (b) `no-such-shoe` `38` qty 1.
**Then**: (a) 409 `stock_conflict` with `errors["items.0"]` `<name of the draft shoe> · EU 38 — no longer in the shop.` (b) 409 `stock_conflict` with `errors["items.0"]` `no-such-shoe · EU 38 — no longer in the shop.` (the id stands in for the missing name). No order; the shelf unchanged.
Traces: SC-25 (ORD-004 AC-6, "Not found" row)
Check: fetch_url

**Scenario**: TC-46 — every failing pair is named at once, and nothing is taken for the fine one
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5) (fine), SHELF(`samba-og-w-crystal-white`, `38`, 0) (sold out), SHELF(`samba-og-kids-cloud-white`, `38`, 1) (short for qty 2), the draft shoe of TC-45; `stock` of the three shoes read.
**When**: REQ is sent with four lines in this order: core-black `38` qty 1; crystal-white `38` qty 1; kids-cloud-white `38` qty 2; the draft wonder-quartz `38` qty 1.
**Then**: 409 `stock_conflict` whose `errors` keys are exactly `items.1`, `items.2`, `items.3` with, in turn: `<crystal-white name> · EU 38 — sold out in your size.`, `<kids-cloud-white name> · EU 38 — only 1 left in your size.`, `<wonder-quartz name> · EU 38 — no longer in the shop.` Core-black's `38` still holds 5 pairs and its `stock` is as read.
Traces: SC-25 (ORD-004 item 3, AC-5)
Check: fetch_url

**Scenario**: TC-47 — after any refusal the counts, lists and totals are exactly as before
**Given**: SHELF(`samba-og-w-core-black`, `38`, 2); the shoe read as P0 (`stock`, `sizes`, `sizes_low`, `sizes_out`, `price`); RESET.
**When**: Five refused requests are sent (qty 3; qty 5; a size of `99` (422); a draft shoe line (409); an empty address (422)); the shoe is read again as P1; then one valid REQ.
**Then**: P1 equals P0 field for field; `SELECT pairs` is 2; the valid request's order is `DOI-24818` (no number was used) with `total` 3290000.
Traces: SC-25 (ORD-004 AC-7; ORD-005 AC-8)
Check: fetch_url

**Scenario**: TC-48 — the page names each refused pair, keeps the bag and the details, and enables the button
**Given**: BAG with one `samba-og-w-crystal-white` `EU 38` qty 1 and one `samba-og-w-core-black` `EU 36` qty 1 (written to the bag before the shelf changes); FORM typed with note `Call before delivering.`; then SHELF(`samba-og-w-crystal-white`, `38`, 0) and SHELF(`samba-og-w-core-black`, `36`, 0); 1280×800.
**When**: `[data-testid='place-order']` is pressed.
**Then**: The page stays at `/checkout`. Above the button two lines show, one per pair, each exactly `<shoe name> · EU 38 — sold out in your size.` and `<shoe name> · EU 36 — sold out in your size.` (the names as `GET /api/v1/products/<id>` gives them). The `agentteam:cart` entry still holds both lines unchanged; `[data-testid='bag-count']` reads 2; `#checkout-name`, `#checkout-phone`, `#checkout-address`, `#checkout-city`, `#checkout-note` hold exactly the FORM values; `[data-testid='place-order']` reads `Place order` and is enabled; no order is added to the browser's store. With only the second pair sold out only that sentence shows. With SHELF(`samba-og-w-crystal-white`, `38`, 1) and qty 3 in the bag the line reads `<shoe name> · EU 38 — only 1 left in your size.` With the shoe set to draft it reads `… — no longer in the shop.`
Traces: SC-26 (ORD-004 AC-3, AC-8, item 4; ORD-009 AC-4), SC-11
Check: interact_page

**Scenario**: TC-49 — the last-pairs list, the sold-out list and the stock follow the count as orders are saved
**Given**: SHELF(`samba-og-w-core-black`, `38`, 3); the shoe read.
**When**: REQ for `38` qty 1 is sent three times in turn, reading `GET /api/v1/products/samba-og-w-core-black` before and after each.
**Then**: Before: `38` is in neither `sizes_low` nor `sizes_out`. After the first (2 pairs): `38` is in `sizes_low`. After the second (1 pair): `38` is still in `sizes_low`. After the third (0 pairs): `38` is in `sizes_out` and not in `sizes_low`. `sizes` still lists `38` in every read, and each read shows `stock` one lower than the one before. Each read happens right after the 201 (no wait).
Traces: SC-27 (ORD-005 AC-1, AC-2, AC-3)
Check: fetch_url

**Scenario**: TC-50 — pairs in stock is the sum over the sizes and drops by what an order takes
**Given**: `samba-og-w-wonder-quartz` with `UPDATE product_sizes SET pairs = CASE size WHEN '36' THEN 3 WHEN '36 2/3' THEN 2 WHEN '37 1/3' THEN 0 WHEN '38' THEN 5 ELSE 0 END WHERE product_id = 'samba-og-w-wonder-quartz'` (the old counts read first and put back).
**When**: The shoe is read; REQ for `samba-og-w-wonder-quartz` `38` qty 2 is sent; the shoe is read again.
**Then**: First read: `stock` 10; `sizes_low` is `["36 2/3"]`; `sizes_out` holds `37 1/3` and every other size except `36`, `36 2/3` and `38`; `36` and `38` are in neither list. Second read: `stock` 8; `38` (3 pairs left) is in neither list.
Traces: SC-27 (ORD-005 AC-5, AC-6)
Check: fetch_url

**Scenario**: TC-51 — a size that sold out shows struck through and cannot be chosen on the shoe page
**Given**: SHELF(`samba-og-w-core-black`, `38`, 1); then REQ for that pair is sent (201), so `38` is in `sizes_out`.
**When**: `/shop/samba-og-w-core-black` loads afterwards (1280×800); the pill for EU 38 (`[data-testid='size-38']`, the pill whose text is `38`) is inspected and clicked; then the pill for a size still in stock (for example `36`) is clicked.
**Then**: The `38` pill has computed `text-decoration-line` `line-through` and is disabled (`disabled` true or `aria-disabled="true"`); clicking it leaves no size selected (no pill marked pressed or checked) and `[data-testid='add-to-bag']` does not add a pair; clicking the in-stock pill selects it. The page uses the existing size picker (no new control).
Traces: SC-28 (ORD-005 AC-4)
Check: interact_page

**Scenario**: TC-52 — after the move to one count per size, every shoe lists what it listed before
**Given**: A database in which no case of this file has placed an order or changed a count yet (run this first, or after RESET with every SHELF put back); the 12 shoes' lists from the first design (`src/data/products.json`): core-black sizes `36, 36 2/3, 37 1/3, 38, 38 2/3, 39 1/3, 40, 40 2/3, 41 1/3`, low `37 1/3, 40`, out `41 1/3`; crystal-white same nine sizes, low `36, 39 1/3`, out none; wonder-quartz same nine, low `38 2/3`, out `41 1/3`; handball-spezial-w-earth-strata `36, 36 2/3, 37 1/3, 38, 38 2/3, 39 1/3, 40, 40 2/3`, low `36, 40 2/3`, out `38`; gazelle-indoor-solar-gold `40, 40 2/3, 41 1/3, 42, 42 2/3, 43 1/3, 44, 44 2/3, 45 1/3`, low `42, 42 2/3`, out `40`; adizero-adios-wonder-blue the same nine, low `44 2/3`, out `45 1/3`; adizero-evo-sl-charcoal `40 … 44 2/3` (eight), low `40, 44 2/3`, out none; megaride-s2-cloud-white `40, 40 2/3, 41 1/3, 42, 42 2/3, 43 1/3, 44`, low `41 1/3`, out `44`; megaride-bones-halo-green `40 2/3, 41 1/3, 42, 42 2/3, 43 1/3, 44`, low `42 2/3, 44`, out none; megaride-ag-xl-core-black `41 1/3, 42, 42 2/3, 43 1/3, 44, 44 2/3, 45 1/3`, low `42, 45 1/3`, out `44 2/3`; samba-og-kids-wonder-mauve `35 1/2, 36, 36 2/3, 37 1/3, 38, 38 2/3`, low `35 1/2`, out none; samba-og-kids-cloud-white the same six, low `38`, out `38 2/3`.
**When**: `GET /api/v1/products?limit=100` is read.
**Then**: For each of the 12 shoes `sizes`, `sizes_low` and `sizes_out` equal the lists above, in the same order, and each is an array (an empty list is `[]`, not `null`); `stock` is a whole number equal to the sum of the shoe's counts and is not pinned to the first design's figure (18, 24, 31, 9, 12, 15, 20, 6, 5, 16, 14, 16 may differ). The response has no `pairs` key on any shoe.
Traces: SC-29 (ORD-005 item 4; DATA-001 AC-2)
Check: fetch_url

**Scenario**: TC-53 — every size that has pairs can be ordered; a size with none cannot
**Given**: The 12 shoes as in TC-52 (read first); RESET afterwards.
**When**: For each shoe, REQ is sent for its first size that is not in `sizes_out` with qty 1; and, for each shoe whose `sizes_out` is not empty, for a size in `sizes_out` with qty 1.
**Then**: Every request for a size not in `sizes_out` answers 201 (12 orders); every request for a size in `sizes_out` answers 409 `stock_conflict` with `— sold out in your size.`
Traces: SC-29 (ORD-005 item 4)
Check: fetch_url

---

## E. The payment seam (SC-30, SC-31)

**Scenario**: TC-54 — `GET /v1/payment-options` lists VNPay alone, available; the records keep all four rows
**Given**: The seeded `payment_methods` (`cod`, `bank`, `vnpay`, `momo`); no VNPay key is configured because the VNPay story has not landed.
**When**: `GET /api/v1/payment-options` and `GET /api/v1/payments` are sent; then `GET /api/v1/payment-options?zzz=1`.
**Then**: The first answers 200 `{"items":[…],"total":1,"page":1,"limit":100}` with exactly one item whose keys are `id`, `name`, `note`, `icon`, `online`, `is_available`: `id` `vnpay`, `name` `VNPay — ATM, Visa, Mastercard`, `note` and `icon` equal to the `vnpay` row of `/payments`, `online` `true`, `is_available` `true`. No item has the id `cod`, `bank` or `momo`. `/payments` answers 200 with `total` 4 and the four ids `cod`, `bank`, `vnpay`, `momo`. The third answer equals the first (an undefined query parameter on a list is ignored).
Traces: SC-30 (ORD-006 AC-1, AC-3, AC-5)
Check: fetch_url

**Scenario**: TC-55 — `GET /v1/payment-options` fails as the contract says when the database is away
**Given**: `set_db_available` takes the database away while the service runs.
**When**: `GET /api/v1/payment-options` is sent; the database is made available again.
**Then**: 500 `{"error":"internal","message":<sentence>}` with no `errors` key and no SQL or stack in the body; after the database is back the same request answers 200 as in TC-54.
Traces: contract (GET /v1/payment-options)
Check: fetch_url

**Scenario**: TC-56 — with no provider implemented the order is saved unpaid and `payment_url` is `""`
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5).
**When**: REQ is sent; the order is read back with the token.
**Then**: 201 with `payment_url` equal to the string `""`; `order.paid` `false` in the answer and in the read-back; `order.status` `placed`; the order exists (the read-back answers 200).
Traces: SC-31 (ORD-006 AC-6, AC-9)
Check: fetch_url

**Scenario**: TC-57 — a test provider's address is passed through; a provider that fails leaves `""` and the order is still saved
**Given**: The backend's tests for the payment-provider interface: one provider returning `https://pay.example.test/x`, one that returns an error instead of an address.
**When**: The tests that place an order through each are read and run (`run_tests` on the backend).
**Then**: With the first provider the placing answer's `payment_url` is exactly `https://pay.example.test/x` and the order is unpaid; with the second `payment_url` is `""`, the order is saved, unpaid, and the shelf lost its pairs; with no provider (TC-56) it is `""`. All three tests pass and placing in the handler goes through the one interface, not a branch on the way to pay.
Traces: SC-31 (ORD-006 AC-7, AC-8, item 1)
Check: manual — a provider cannot be injected over HTTP and no real provider ships in this story; the evidence is the backend tests named by the story ("Proven with test providers"), read and run by the verifier.

---

## F. Where the browser lands (SC-32)

**Scenario**: TC-58 — an empty payment address empties the bag, stores the order's four facts, and goes to the order page
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); BAG with one `samba-og-w-core-black` `EU 38` qty 1; FORM typed; RESET (the next number is `DOI-24818`); 1280×800; no entry named `doi:orders` before.
**When**: `[data-testid='place-order']` is pressed; after the page leaves `/checkout`, the address, `localStorage` and `sessionStorage` are read.
**Then**: The address is `/orders/doi-24818?placed=1`. `agentteam:cart` is absent (the bag counts 0). Exactly one `localStorage` entry other than `agentteam:*` holds the order, and it holds the id `doi-24818`, the number `DOI-24818`, a placed time and a token of at least 22 characters, and nothing else about the order — no `phone`, `0912`, `Lan Nguyen`, `Example Street`, `Da Nang`, `Call before delivering`, `total` or `items` appears in any storage entry. The entry named `doi:orders` does not exist and holds nothing about this order (the design's mock store is not used).
Traces: SC-32 (ORD-009 AC-1, AC-2)
Check: interact_page

**Scenario**: TC-59 — a payment address sends the browser there, with the bag emptied and the order stored
**Given**: As TC-58, but the answer of `POST /api/v1/orders` is stubbed to the real answer with `"payment_url": "https://pay.example.test/x"` (the page's navigation to that address is intercepted and answered with an empty page).
**When**: `[data-testid='place-order']` is pressed.
**Then**: The browser requests `https://pay.example.test/x` (its address becomes `https://pay.example.test/x`); `agentteam:cart` is absent; the same four facts as TC-58 are stored for the order; the address was not `/orders/…?placed=1` at any moment after the answer.
Traces: SC-32 (ORD-009 AC-3, ORD-006 AC-7)
Check: interact_page

**Scenario**: TC-60 — a browser that refuses storage still places the order and goes on
**Given**: Before the page's scripts run, `Storage.prototype.setItem` is replaced with a function that throws and `localStorage.getItem` returns `null`; SHELF(`samba-og-w-core-black`, `38`, 5); the bag is filled in this page load (the shoe page's size pill and `[data-testid='add-to-bag']`, then client-side navigation to `/checkout` without a reload); FORM typed.
**When**: `[data-testid='place-order']` is pressed.
**Then**: `POST /api/v1/orders` answers 201 (an order exists); the page shows no failure message; the address becomes `/orders/<id>?placed=1`; the token is held in memory for this page load only (after a reload the browser holds no copy: no storage entry).
Traces: SC-32 (ORD-009 "Storage refused" row)
Check: interact_page

---

## G. Failed, offline and repeated submissions (SC-33, SC-34)

**Scenario**: TC-61 — the shop answers 500: the failure line shows, the bag and details stay, the button is enabled; pressing again succeeds
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); BAG with one `samba-og-w-core-black` `EU 38` qty 1; FORM typed; `ALTER TABLE order_tokens RENAME TO order_tokens_x` so placing answers 500 while the catalogue still reads (put back afterwards); RESET; 1280×800.
**When**: `[data-testid='place-order']` is pressed; then the table is renamed back and the button is pressed again.
**Then**: After the first press, above the button the text `We could not place your order. Nothing was charged. Please try again.` shows; the URL is still `/checkout`; `agentteam:cart` holds the pair; `#checkout-name` … `#checkout-note` hold the FORM values; `[data-testid='place-order']` reads `Place order` and is enabled; no order is added to the browser's store. After the second press the address becomes `/orders/doi-24818?placed=1` (the failed attempt used no number) and the shelf lost one pair.
Traces: SC-33 (ORD-010 AC-1)
Check: interact_page

**Scenario**: TC-62 — the shop cannot be reached: the same line, the same survival
**Given**: As TC-61 but the page is loaded and FORM typed first; then the backend is stopped (the gateway answers 502) or the request to `/api/v1/orders` is aborted at the network.
**When**: `[data-testid='place-order']` is pressed.
**Then**: The text `We could not place your order. Nothing was charged. Please try again.` shows above the button; the bag and the typed details are unchanged; the button is enabled; the URL is `/checkout`. The backend is started again afterwards.
Traces: SC-33 (ORD-010 AC-1, "Upstream failure" row)
Check: interact_page

**Scenario**: TC-63 — offline: its own line, the same survival
**Given**: BAG of TC-61; FORM typed; the page's network is set offline (`navigator.onLine` is `false`, requests fail) after the page loaded.
**When**: `[data-testid='place-order']` is pressed.
**Then**: The text `You seem to be offline. Check your connection and try again.` shows above the button (not the `We could not place your order` line); the bag and the typed details are unchanged; the button is enabled. Back online, pressing the button again places the order. If the instrument cannot emulate offline the case is UNVERIFIED (instrument lacked: network emulation).
Traces: SC-33 (ORD-010 AC-2)
Check: interact_page

**Scenario**: TC-64 — the same key twice answers the first order again, takes the pairs once
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `stock` S0 read; RESET.
**When**: REQ is sent with `Idempotency-Key: tc-replay-0001`; the same body is sent again with the same key; then `GET /api/v1/orders/doi-24818` is read with the first token and with the second token.
**Then**: First: 201, `order.number` `DOI-24818`, token T1. Second: status 200, header `Idempotent-Replayed: true`, the same `order` (`id`, `number`, `total`, `placed_at`, `items`), a token T2 different from T1, `payment_url` `""`; the first answer has no `Idempotent-Replayed` header. `stock` is S0 − 1 (not S0 − 2). Both T1 and T2 open the order (200). The next new key's order is `DOI-24819`. The same key sent with the phone written `0912345678` instead of `0912 345 678` (same digits) is also a replay (200).
Traces: SC-34 (ORD-010 AC-3, AC-4)
Check: fetch_url

**Scenario**: TC-65 — the first send saved the order but its answer was lost: pressing again lands on that order
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); BAG with one pair; FORM typed; RESET; the first `POST /api/v1/orders` is let through to the shop and its answer is dropped before the page sees it (the page's request fails after the shop saved the order); later requests pass.
**When**: `[data-testid='place-order']` is pressed (failure line shows); it is pressed again.
**Then**: After the first press the failure line `We could not place your order. Nothing was charged. Please try again.` shows and the bag still holds the pair. After the second press the address is `/orders/doi-24818?placed=1`; exactly one order exists (the next placing by someone else is `DOI-24819`); the shelf lost one pair, not two; the two requests carried the same `Idempotency-Key`. If the instrument cannot let a request through and drop its answer, the case is UNVERIFIED (instrument lacked: request interception that forwards then aborts); TC-64 proves the shop's half.
Traces: SC-34 (ORD-010 AC-3)
Check: interact_page

**Scenario**: TC-66 — the same key from two tabs at once makes one order
**Given**: SHELF(`samba-og-w-core-black`, `38`, 10); `stock` S0 read; RESET.
**When**: Two identical REQs with the same `Idempotency-Key: tc-twotabs-0001` are sent in one parallel batch; the batch is repeated with five new keys.
**Then**: In every pair both answers carry the same `order.id`; exactly one answers 201 and the other 200 with `Idempotent-Replayed: true`; the numbers used are `DOI-24818` … `DOI-24823` with no gap or repeat; `stock` is S0 − 6.
Traces: SC-34 (ORD-010 "Conflict" row)
Check: fetch_url

**Scenario**: TC-67 — one key per bag and set of details; a new key when either changes
**Given**: BAG with one pair; FORM typed; a log of the `Idempotency-Key` header of every `POST /api/v1/orders`; `order_tokens` renamed so every attempt fails with 500 (put back afterwards), so the page stays on `/checkout`.
**When**: `[data-testid='place-order']` is pressed (attempt 1); pressed again with nothing changed (attempt 2); the note is edited to `Call before delivering, please.` and the button pressed (attempt 3); the quantity of the pair is raised to 2 in the bag and the button pressed (attempt 4); the note is edited back and the quantity set back to 1 and the button pressed (attempt 5).
**Then**: Each key is 8–100 characters of `[A-Za-z0-9._-]`. Attempt 2's key equals attempt 1's. Attempt 3's key differs from attempts 1–2. Attempt 4's key differs from attempt 3's. Attempt 5's key is not equal to attempt 4's. Every placed request carries the header.
Traces: SC-34 (ORD-010 AC-3, AC-5)
Check: interact_page

**Scenario**: TC-68 — a missing or malformed key is refused 400 naming the header
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `stock` S0 read; RESET.
**When**: REQ is sent with: (a) no `Idempotency-Key` header; (b) `Idempotency-Key: abcdefg` (7 characters); (c) a 101-character key; (d) `Idempotency-Key: has space!!` (characters outside the set); (e) an empty value; (f) no header and also an invalid body (empty address); then REQ with a key of exactly 8 characters and another of exactly 100 characters.
**Then**: (a)–(f) answer 400 `{"error":"bad_request","message":<sentence>,"errors":{"Idempotency-Key":<sentence>}}` — also (f): the header is checked before the fields, so it is not a 422; no order; `stock` is S0 after them. The 8-character and the 100-character keys answer 201 (and use `DOI-24818`, `DOI-24819`).
Traces: SC-34 (ORD-010 boundary; contract §2.6, §3.10)
Check: fetch_url

**Scenario**: TC-69 — the same key with another phone is a conflict that reveals nothing
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); a first order placed with `Idempotency-Key: tc-phone-0001` and phone `0912 345 678` (order O1, token T1).
**When**: The same key is sent with phone `0987 654 321`.
**Then**: 409 `{"error":"conflict","message":<sentence>}` with no `errors` key; the body and headers contain neither `DOI-`, `doi-`, T1, a new token nor the first phone; `stock` is S0 − 1 (the first order's pair only).
Traces: SC-34 (ORD-010 conflict; contract §2.6)
Check: fetch_url

---

## H. Privacy (SC-35 … SC-37)

**Scenario**: TC-70 — the orders list is not public: 401 for everyone, with or without a filter
**Given**: RESET; one order placed by REQ (token T1); 14 seeded orders exist.
**When**: `GET /api/v1/orders` is sent with: (a) no cookie and no header; (b) `?phone=0938410552`; (c) `?q=DOI-24817`; (d) `?sort=-placed_at&limit=1`; (e) the header `X-Order-Token: T1`; (f) a forged cookie `session=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`; (g) `?page=0`.
**Then**: Every answer is 401 `{"error":"unauthenticated","message":<sentence>}` with no `errors` key and no `items`, `total`, `phone` or order data in the body; (g) is 401 too (the session is checked before the query). No answer is 200 or 403.
Traces: SC-35 (ORD-008 AC-1, AC-9, items 1 and 5)
Check: fetch_url

**Scenario**: TC-71 — one order is read only with its own token, and never with the private fields
**Given**: Orders O1 (token T1) and O2 (token T2) placed by REQ with phones `0912 345 678` and `0987 654 321`, address `12 Example Street, Ward 1`, note `Gate code 4821`; the seeded `doi-24817`.
**When**: `GET /api/v1/orders/<O1>` is sent with (a) no token; (b) `X-Order-Token: T1`; (c) `X-Order-Token: T2`; (d) `X-Order-Token: ` (empty); (e) a token with one character changed; (f) T1 in the query `?token=T1` with no header; and `GET /api/v1/orders/doi-24817` with no token and with T1.
**Then**: (a), (c), (d), (e), (f) answer 404 with the same body (`{"error":"not_found","message":<sentence>}`, no `errors` key); (b) answers 200 with the order (`id`, `number`, `status`, `total`, `items` with their `unit_price`s, `customer`, `city`, `paid`, `payment`, `delivery`) and with none of the keys `phone`, `email`, `address`, `note`; the strings `0912`, `Example Street` and `Gate code` do not appear in the body; `Cache-Control: no-store`. The seeded order answers 404 with no token and with T1, and its phone `0938 410 552` appears in no answer.
Traces: SC-36 (ORD-008 AC-4, AC-5, AC-6)
Check: fetch_url

**Scenario**: TC-72 — a stranger cannot tell an order that exists from one that does not
**Given**: Order O1 with token T1; the unknown id `nope`; a malformed id `DOI-24818` (upper case) and `..%2Fadmin`.
**When**: The bodies and statuses of: `GET /api/v1/orders/<O1>` with no token; `GET /api/v1/orders/nope` with no token; `GET /api/v1/orders/nope` with T1; `GET /api/v1/orders/DOI-24818`; `GET /api/v1/orders/..%2Fadmin` are compared.
**Then**: All five answer 404 with byte-for-byte the same body (`error` `not_found` and the same `message`) and the same `Content-Type`; none holds the id sent back to the caller.
Traces: SC-36 (ORD-008 AC-8)
Check: fetch_url

**Scenario**: TC-73 — the token is in the placing answer and this browser's storage only
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); BAG with one pair; FORM typed; a log of every request URL and header the page sends; 1280×800.
**When**: The order is placed; after the page has landed the placing answer's `token` (read from the intercepted answer), the address, the page's text, `document.cookie`, `sessionStorage`, `localStorage` and the request log are read.
**Then**: The token is in the placing answer and in exactly one `localStorage` entry. It is not in `location.href`, not in `document.title`, not in the page's visible text or HTML, not in `document.cookie`, not in `sessionStorage`, not in any request URL or query string in the log and not in the `Referer` of any request; no response sets a cookie holding it.
Traces: SC-37 (ORD-008 AC-10, item 4)
Check: interact_page

**Scenario**: TC-74 — a read of the order does not echo the token, and a replay returns a new one
**Given**: Order O1 with token T1 (REQ with key K).
**When**: `GET /api/v1/orders/<O1>` is sent with `X-Order-Token: T1`; REQ is sent again with the same key K.
**Then**: The read's body and headers do not contain T1. The replay's body holds a token T2 ≠ T1 and nothing else secret; neither answer sets a cookie (`Set-Cookie` absent).
Traces: SC-37 (ORD-008 item 4; contract §2.6)
Check: fetch_url

**Scenario**: TC-75 — the token is in no log line
**Given**: The backend's access log and error log while TC-70 … TC-74 and TC-36 ran.
**When**: The log lines for those requests are read (and the code that writes them: `AccessLog` and the order handlers).
**Then**: No log line contains a token, an `Idempotency-Key`, a phone, an address or a note; a log line for a placing holds the method, the path, the status, the duration and `X-Request-Id`.
Traces: SC-37 (ORD-008 item 4; contract §2.6 "A key is never logged")
Check: manual — no instrument reads the backend's log output; the evidence is the logging code and the log text the verifier is given (TL's backend review reads the same lines).

---

## I. Contract cases for `POST /v1/orders` (the error envelope and the request boundary)

**Scenario**: TC-76 — a field the contract does not define, or a wrong type, is refused 400
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `stock` S0 and the next number read.
**When**: REQ is sent with, one change per request: (a) an extra body field `"coupon": "SAVE10"`; (b) an extra body field `"email": "lan@example.com"`; (c) an extra field on a line, `"items": [{"product":"samba-og-w-core-black","size":"38","qty":1,"colour":"black"}]`; (d) `"qty": "1"` (a string); (e) `"qty": 1.5`; (f) `"items": "samba-og-w-core-black"`; (g) `"customer": 5`; (h) the body `[]`; (i) the body `not json`.
**Then**: Every one answers 400 `{"error":"bad_request","message":<sentence>,"errors":{…}}` (for (h) and (i) `errors` may be absent or name the body); (a) `errors` has the key `coupon`, (b) the key `email`, (c) a key that names the extra line field (for example `items.0.colour`), (d)–(g) a key naming the field with the wrong type; none answers 200, 201 or 422 and none ignores the field. No order is created and `stock` is S0 afterwards. (The four tolerated extras of TC-23 are the only undefined fields accepted.)
Traces: contract (POST /v1/orders §2.5, §3.10)
Check: fetch_url

**Scenario**: TC-77 — a wrong content type is 415 and a body over 64 KiB is 413
**Given**: SHELF(`samba-og-w-core-black`, `38`, 5); `stock` S0 read.
**When**: REQ is sent with `Content-Type: text/plain`; with `Content-Type: application/x-www-form-urlencoded`; and as JSON with a `note` padded so the body is 70,000 bytes.
**Then**: The first two answer 415 `{"error":"unsupported_media_type","message":<sentence>}`; the third answers 413 `{"error":"payload_too_large","message":<sentence>}`; none has an `errors` key; no order is created and `stock` is S0.
Traces: contract (POST /v1/orders §2.1, §2.3)
Check: fetch_url

**Scenario**: TC-78 — every refusal of placing has the one error envelope, and a request id
**Given**: One request of each kind from TC-16, TC-26, TC-43, TC-68, TC-69, TC-76, TC-77 and TC-36.
**When**: Each answer's headers and body are read; a request is sent with `X-Request-Id: tc-request-0001`.
**Then**: Every body is a JSON object whose keys are `error`, `message` and, only for 400, 422 and the `stock_conflict` 409, `errors`; `error` is one of `bad_request`, `invalid`, `stock_conflict`, `conflict`, `unsupported_media_type`, `payload_too_large`, `internal` (lower case, as the status says); `message` is a non-empty sentence; no body holds SQL, a file path, a stack trace, a hash, a token or another person's data. Every answer carries `X-Request-Id`, and the one sent as `tc-request-0001` comes back as `tc-request-0001`.
Traces: contract (§2.3 error contract; §2.1 trace header)
Check: fetch_url

---

## Coverage walk — every story criterion has a case

| Criterion | Cases | Criterion | Cases |
|---|---|---|---|
| SC-1 | TC-01 | SC-20 | TC-34, TC-35 |
| SC-2 | TC-02, TC-03 | SC-21 | TC-36, TC-37 |
| SC-3 | TC-04 | SC-22 | TC-38 |
| SC-4 | TC-05, TC-06 | SC-23 | TC-39, TC-40 |
| SC-5 | TC-07 | SC-24 | TC-41, TC-42 |
| SC-6 | TC-08, TC-09 | SC-25 | TC-43, TC-44, TC-45, TC-46, TC-47 |
| SC-7 | TC-10 | SC-26 | TC-48 |
| SC-8 | TC-11 | SC-27 | TC-49, TC-50 |
| SC-9 | TC-12, TC-13 | SC-28 | TC-51 |
| SC-10 | TC-14 | SC-29 | TC-52, TC-53 |
| SC-11 | TC-15 (validation), TC-48 (refusal), TC-61 (failed send) | SC-30 | TC-54 (TC-55 contract) |
| SC-12 | TC-16, TC-17, TC-18, TC-19 | SC-31 | TC-56, TC-57 |
| SC-13 | TC-20, TC-21 | SC-32 | TC-58, TC-59, TC-60 |
| SC-14 | TC-22, TC-23, TC-24 | SC-33 | TC-61, TC-62, TC-63 |
| SC-15 | TC-25 | SC-34 | TC-64, TC-65, TC-66, TC-67, TC-68, TC-69 |
| SC-16 | TC-26, TC-27, TC-28 | SC-35 | TC-70 |
| SC-17 | TC-29, TC-30, TC-31 | SC-36 | TC-71, TC-72 |
| SC-18 | TC-32 | SC-37 | TC-73, TC-74, TC-75 |
| SC-19 | TC-33 | Contract (`POST /v1/orders`, `GET /v1/payment-options`) | TC-23, TC-55, TC-76, TC-77, TC-78 |

**Instruments (78 cases).** `fetch_url` 48 (TC-16 … TC-24, TC-26 … TC-30, TC-32 … TC-47, TC-49, TC-50, TC-52 … TC-56, TC-64, TC-66, TC-68 … TC-72, TC-74, TC-76 … TC-78 — TC-31 is `manual`, TC-25, TC-48 and TC-51 are `interact_page`), `interact_page` 26 (TC-01 … TC-09, TC-11 … TC-15, TC-25, TC-48, TC-51, TC-58 … TC-63, TC-65, TC-67, TC-73), `render_url` 1 (TC-10), `manual` 3 (TC-31, TC-57, TC-75). `measure_styles` is not used as a separate instrument: the visual facts (the hidden step line, the struck-through size pill, the capitals of a name) are read inside `interact_page` as computed values (`display`, `text-decoration-line`, `text-transform`), and the picture in TC-05 as an `img` whose `naturalWidth` is above 0. If the verifier prefers `measure_styles` for one of those readings it answers the same question.
Each `manual` case names why no instrument answers it: TC-31 (no instrument sets the shop's clock; a backend test with an injected clock is the evidence, and TC-29 checks the rule live), TC-57 (a provider cannot be injected over HTTP; the story proves it with test providers), TC-75 (no instrument reads the backend's log output).
Cases that depend on the verifier being able to intercept or emulate in the browser (TC-09's held answer, TC-11's stub, TC-59's stub, TC-62, TC-63, TC-65) or on a 50-request parallel batch (TC-42) or on per-request timing (TC-38) say so in their own text; if the instrument cannot do it, the case is UNVERIFIED with that reason, never failed.
