# Test cases — Free delivery on every order, wording to match

Module: `site` · Plan item 2 · Requirement: SITE-016 (with the lines of SITE-004, SITE-006, SITE-007, SITE-009, SITE-012, and ORD-001, ORD-006, ORD-007, ORD-020 where this story touches them)
Story: `docs/site/stories/free-delivery-on-every-order-wording-to-match.md` — criteria SC-1 … SC-34
Shape: frontend and JSON only. No backend, no database, no endpoint: there is no service-contract case in this file (the story adds no route and the API contract has none for this change), and no request-body case.

**Risk level: medium-high.** The change is words and rows, but it decides the amount a customer is shown and pays (Delivery "Free", Total = pairs total) and removes an offer (cash on delivery) the shop does not honour. Every total, boundary and checkout case is therefore written out value by value; pure copy cases are one read each.

## How these cases are observed

- **Default window** is 1280×800. Cases for the phone say 390×844; the figures row also has a 640 window.
- **Prices** are shown as `vi-VN` writes them: `3.090.000 ₫` (the space before `₫` may be a no-break space; compare with either).
- **The data files** (`delivery`, `shop`, `schema`, `orders`, `payments`, `faqs`, `reviews`) and `content.json` ship inside the running site: when the site is served by the dev server each is a module at `/src/data/<name>.json` and `/src/content.json`; in a production build they are object literals inside the JS files the home page's HTML references (`<script src>` and `modulepreload`, plus the lazily loaded page chunks). A `fetch_url` case reads those files and reads the literal, not the rendered page. Minified keys appear unquoted (`fee:0`); compare the value, not the quoting.
- **Selectors.** Existing test ids are used where the app has them: `[data-testid='bag-count']`, `[data-testid='add-to-bag']`, `[data-testid^='size-']`, `[data-testid='cart-checkout']`, `[data-testid='checkout-total']`, `[data-testid='delivery-standard']`, `[data-testid='pay-vnpay']`, `[data-testid='place-order']`, `[data-testid='order-status']`, `[data-testid='track-submit']`; form fields `#checkout-name`, `#checkout-phone`, `#checkout-address`, `#checkout-city`, `#checkout-note`, `#order-number`, `#order-phone`. The summary on `/cart` and `/checkout` is the `aside dl`, whose three rows are `aside dl > div:nth-child(1)` (Subtotal), `:nth-child(2)` (Delivery), `:nth-child(3)` (Total), each with a `dt` label and a `dd` value. The Delivery table is the page's only `table`. No Tailwind class is relied on.
- **Arrangement A — a bag with one pair.** Open `/shop/samba-og-w-crystal-white` (3.090.000 ₫), click the first enabled `[data-testid^='size-']`, click `[data-testid='add-to-bag']` (the bag drawer opens), then open `/cart`. The bag is the browser's `localStorage` entry `agentteam:cart`.
- **Arrangement B — a bag worth a chosen amount.** The catalogue's cheapest pair is 2.290.000 ₫, so a bag under 2.000.000 ₫ cannot be made from it. Do arrangement A, read the `agentteam:cart` entry (a JSON array of lines `{id, name, price, image, variant, qty}`), change only the `price` of the line(s) to the amount the case names, write it back and reload the page. A two-pair bag of 4.590.000 ₫ is arrangement A plus a second shoe (`/shop/samba-og-kids-cloud-white`) added the same way, with its line's `price` set to 1500000. The line price is what Bag and Checkout add up (`lib/cart.ts`), so this is the only way to reach the amounts the story names.
- **Sample orders** are opened at `/orders/<id>` with the lower-case id, for example `/orders/doi-24817`.

## Ambiguities found in the documents (reported, not silently resolved)

1. **SITE-016 AC-7 says the words "cash on delivery" appear on no customer page; story §4 keeps five sample orders (DOI-24816, -24814, -24811, -24808, -24804) that name the stored `cod` record.** The order page of such an order reads `Cash on delivery · awaiting payment` (or `· paid`). These cases follow the story: the exception is the five order pages only; TC-40 and TC-48 pin both halves. PM should say in the SRS that AC-7 excludes them.
2. **No page renders the questions of topic "payment".** `FaqRows` is mounted for `shipping`, `returns`, `sizing` and `orders` only. SC-6 can therefore only be read from the data (TC-07), not from a page.
3. **Reviews shown on a shoe page may be a subset.** r-04 belongs to `samba-og-w-crystal-white` and r-07 to `samba-og-w-wonder-quartz`; if a page shows fewer reviews than a shoe has, TC-10 is read from the data in TC-11 for the one not shown.
4. **SC-4 "no other field of any sample order differs from before" needs the base commit.** It is a diff read (TC-05, `manual`), not an HTTP observation.
5. **SC-15 asks for a stored value to be changed.** The records ship inside the build and no instrument edits a file and rebuilds, so it is `manual` (TC-20) until the data story serves the records from the API.
6. **The checkout line under the totals (`next_online`) still says the order is "marked paid at once" after this story** — story §4 leaves it to the Checkout story. No case asserts that sentence in either direction; only the cash-on-delivery sentence is asserted gone.

---

## A. Records the site ships (SC-1 … SC-9)

**Scenario**: TC-01 — the delivery data holds one record, `standard`, fee 0
**Given**: The site is running; the delivery data is read as described in "How these cases are observed".
**When**: The delivery array is read.
**Then**: It holds exactly one object: `id` `standard`, `name` `Standard`, `days` `2–4 working days`, `fee` the number `0` (not the string `"0"`), `note` `Nationwide with GHN or GHTK, tracked from our door to yours.` No object has the id `express`; the strings `Next working day` and `Inner districts of Hà Nội` do not appear in the delivery data; no `fee` of `30000` or `50000` appears in it.
Traces: SC-1 (SITE-016 AC-2)
Check: fetch_url

**Scenario**: TC-02 — the shop's terms have lost the three fields and kept the others
**Given**: The site is running; the `shop` data is read.
**When**: The one record `shop` is read.
**Then**: Its `standard_days` is `2–4` and `dispatch_cutoff` is `15:00`; `reply_time` is `the same working day`, `rating` `4.8`, `review_count` `181` (unchanged). The names `free_delivery_from`, `express_days` and `express_cities` appear nowhere in the shipped data or code, and neither do `exchange_days` or `refund_days` (the returns story removed those and they stay gone).
Traces: SC-2 (SITE-016 AC-1)
Check: fetch_url

**Scenario**: TC-03 — the schema describes no removed field and offers `standard` only
**Given**: The site is running; `schema.json` is read.
**When**: The `shop` collection's fields, the `orders` collection's `delivery` and `payment` fields and the `delivery` collection's `fee` field are read.
**Then**: The `shop` fields list has no key `free_delivery_from`, `express_days` or `express_cities`, and none of the labels `Free standard delivery from`, `Express delivery, working days`, `Express covers`. The `orders` field `delivery` has the options `[["standard","Standard"]]` and nothing else (no `express`). The `delivery` collection still declares a field `fee`. The `orders` field `payment` still has the options `[["cod","Cash on delivery"],["bank","Bank transfer"],["vnpay","VNPay"],["momo","MoMo"]]`, in that order.
Traces: SC-3 (SITE-016 item 2)
Check: fetch_url

**Scenario**: TC-04 — every sample order is standard delivery with its pairs' total
**Given**: The site is running; the `orders` data and the `products` data are read.
**When**: The fourteen sample orders are read.
**Then**: Every order has `delivery` `standard`; no order has `delivery` `express`. DOI-24815 has `total` 5890000, DOI-24810 has 3290000, DOI-24806 has 3090000. For every order `total` equals its shoe's `price` × `qty`; at the base commit the totals are: DOI-24817 3.090.000, -24816 3.290.000, -24815 5.890.000, -24814 2.290.000, -24813 3.590.000, -24812 5.580.000 (two pairs of 2.790.000), -24811 4.090.000, -24810 3.290.000, -24809 5.090.000, -24808 2.290.000, -24807 3.790.000, -24806 3.090.000, -24805 3.590.000, -24804 2.790.000 (₫). If a shoe's price differs from the one these figures assume, the rule `total = price × qty` governs.
Traces: SC-4 (SITE-016 AC-4)
Check: fetch_url

**Scenario**: TC-05 — nothing else on a sample order changed
**Given**: The pull request's diff of `src/data/orders.json` and the file at the base commit.
**When**: The two are compared record by record.
**Then**: Only the `delivery` value (three records: DOI-24815, DOI-24810, DOI-24806) and the `total` of those three records differ. `status`, `carrier`, `tracking`, `eta`, `payment`, `paid`, `customer`, `phone`, `email`, `address`, `city`, `note`, `placed_at`, `product`, `size`, `qty`, `_source` and the order of the fourteen records are identical; the five records with `payment` `cod` still say `cod`.
Traces: SC-4 (SITE-016 AC-4, "nothing else changes")
Check: manual — a comparison with the base commit's file; the running site holds only the new version, so no HTTP or browser instrument can see what was there before. Read from the PR diff.

**Scenario**: TC-06 — the four payment rows, VNPay's note reworded and the rest untouched
**Given**: The site is running; the `payments` data is read.
**When**: The array is read.
**Then**: It has four objects in the order `cod`, `bank`, `vnpay`, `momo`. The `vnpay` object is `name` `VNPay — ATM, Visa, Mastercard`, `note` exactly `Pay by card or banking app on VNPay's secure page.`, `online` true, `icon` `card`; its note does not contain `marked paid at once`. The other three are as before: `cod` — `Cash on delivery`, `Pay the courier in cash when the box arrives. You may open it and check the pair first.`, online false, icon `cash`; `bank` — `Bank transfer (VietQR)`, `Scan the QR code on the next screen with any banking app. We send the pair as soon as the money lands.`, online true, icon `qr`; `momo` — `MoMo wallet`, `Confirm the payment in the MoMo app; the order is marked paid at once.`, online true, icon `wallet`.
Traces: SC-5 (SITE-016 AC-8)
Check: fetch_url

**Scenario**: TC-07 — the payment questions: "How do I pay?" first, no cash on delivery
**Given**: The site is running; the `faqs` data is read (no page shows topic "payment", ambiguity 2).
**When**: The rows with `topic` `payment` are read in file order.
**Then**: There are exactly two: first `f-payment-how`, question `How do I pay?`, answer `Online through VNPay, by card, domestic ATM card or QR.`; second `f-payment-genuine`, question `Are the shoes genuine?`, answer `Every pair comes in its adidas box with its tags and our receipt.` (unchanged). No row has the id `f-payment-cod` or the question `Can I pay cash on delivery?`; no payment row's question or answer contains `cash` or `courier`.
Traces: SC-6 (SITE-016 AC-6)
Check: fetch_url

**Scenario**: TC-08 — the delivery questions: four closed rows, the time answer says standard only
**Given**: A 1280×800 window on `/shipping`.
**When**: The rows under "Questions about delivery" are counted; the row "How long does delivery take?" is clicked; then each of the other three rows is clicked in turn.
**Then**: Four rows are drawn and all four are closed before the click. The first row opens onto exactly `Two to four working days anywhere in Vietnam by standard delivery.` and the word `Express` (any case) appears in no answer. The other three rows read as before: `Do you deliver to every province?` — `Yes. GHN and GHTK reach all 63 provinces; island districts can take one or two days longer.`; `Can I change the address after ordering?` — `Yes, until the order is packed. Send us the order number and the new address through the contact form.`; `Can you send a pair as a gift?` — `Yes. Put the recipient's name and address at checkout and add "gift" in the note; we leave the price out of the box.`
Traces: SC-7 (SITE-016 AC-8; SITE-009 AC-6)
Check: interact_page

**Scenario**: TC-09 — the order-number answer no longer promises a text message
**Given**: A 1280×800 window on `/track`.
**When**: The row "Where is my order number?" under the questions is clicked.
**Then**: It opens onto exactly `On the page you saw after paying, starting with DOI-.`; the answer contains neither `text message` nor `message we send`.
Traces: SC-8 (SITE-016 AC-8)
Check: interact_page

**Scenario**: TC-10 — the two reviews on their shoe pages read without the removed sentences
**Given**: A 1280×800 window.
**When**: `/shop/samba-og-w-crystal-white` and `/shop/samba-og-w-wonder-quartz` are loaded and their review lists read (see ambiguity 3 if a review is not among those shown).
**Then**: On the first page the review by Thu Hà reads `Bought them as a birthday gift for my sister. She has not taken them off.` and the page contains no `Express`. On the second the review by Minh Anh reads `The pink is softer than in photos, which I prefer.` and the page contains neither `Paid cash` nor `cash at the door`.
Traces: SC-9 (SITE-016 AC-8, item 6)
Check: render_url

**Scenario**: TC-11 — no review in the data mentions express delivery or paying cash
**Given**: The site is running; the `reviews` data is read.
**When**: Every review's `body` is read.
**Then**: `r-04`'s body is exactly `Bought them as a birthday gift for my sister. She has not taken them off.` and `r-07`'s is exactly `The pink is softer than in photos, which I prefer.`; both keep `status` `live` and `_source` `assumed`; no review body matches `express`, `cash` or `courier` (case-insensitive).
Traces: SC-9 (SITE-016 AC-8, item 6)
Check: fetch_url

---

## B. The Delivery page `/shipping` (SC-10 … SC-16)

**Scenario**: TC-12 — the lead and the first figure say free delivery on every order
**Given**: A 1280×800 window.
**When**: `/shipping` is loaded.
**Then**: The `h1` reads `Delivery`. The lead paragraph under it reads exactly `Every pair leaves boxed, tagged and tracked, and delivery is free on every order.` The first oversized figure reads `Free` and the line under it reads exactly `Delivery on every order, whatever it comes to — that is every pair in the shop.`
Traces: SC-10 (SITE-009 AC-1)
Check: render_url

**Scenario**: TC-13 — the second figure is the standard days, and there is no third
**Given**: A 1280×800 window; the shop's `standard_days` is `2–4`.
**When**: `/shipping` is loaded and the figures row (the section directly under the head picture's section and its rule — the page's second `section`) is read.
**Then**: The row has exactly two figures. The second reads `2–4` and its line reads exactly `Working days by standard delivery, anywhere in Vietnam.` No third figure exists; no text containing `express` (any case) and no text `Working day by express` appears in the row.
Traces: SC-11 (SITE-009 AC-2)
Check: render_url

**Scenario**: TC-14 — the two figures sit side by side from 640px and stack on a phone
**Given**: `/shipping` loaded at 1280×800, then at 640×900, then at 390×844; the figures row is the page's second `section`, its two figures are its two child `div`s.
**When**: At each width the row's computed `grid-template-columns` and the two figures' positions are measured.
**Then**: At 1280 and at 640 the row has two column tracks and the two figures share the same top edge, the second's left edge being to the right of the first's right edge. At 390 the row has one column track and the second figure's top edge is at or below the first figure's bottom edge.
Traces: SC-11 (SITE-009 AC-2; story §3.2 "two columns from 640px")
Check: measure_styles

**Scenario**: TC-15 — the Delivery page names no express, fee, amount, city or cut-off
**Given**: A 1280×800 window.
**When**: `/shipping` is loaded and the text of its main content (header and footer excluded) is searched, with every question row opened first so no answer is hidden.
**Then**: None of these appears: `express` (any case), `Free over`, `30.000`, `50.000`, `0 ₫`, any number followed by `₫` (pattern `\d[\d.]*\s?₫`), `15:00`, `cut-off`, `Hà Nội`, `TP. Hồ Chí Minh`, `inner districts`, `Orders placed before`.
Traces: SC-12 (SITE-009 AC-3; SITE-016 AC-5)
Check: render_url

**Scenario**: TC-16 — the options table: four headers and one Standard row
**Given**: A 1280×800 window.
**When**: `/shipping` is loaded and the page's only `table` is read.
**Then**: The heading above it reads `Delivery options` (not `Options and fees`). The header cells read, in order, `Option`, `Arrives`, `Where`, `Cost` (the style draws them in capitals; the text is compared case-insensitively) and no header reads `Fee`. The body has exactly one row: `Standard`, `2–4 working days`, `Nationwide with GHN or GHTK, tracked from our door to yours.`, `Free`.
Traces: SC-13 (SITE-009 AC-4)
Check: render_url

**Scenario**: TC-17 — "Free" and the Cost header are right-aligned
**Given**: A 1280×800 window on `/shipping`.
**When**: The computed `text-align` of the last header cell (`table thead th:last-child`) and of the last cell of the row (`table tbody tr > td:last-child`) is measured.
**Then**: Both are `right`; the first three cells of the row are `left`.
Traces: SC-13 (SITE-009 AC-4)
Check: measure_styles

**Scenario**: TC-18 — "Free" is the row's only cost
**Given**: A 1280×800 window on `/shipping`; the bag page's Delivery word is read from `/cart` (arrangement A) in the same run.
**When**: The Cost cell `table tbody tr > td:last-child` is read.
**Then**: Its entire text is `Free` — no second line under it, no `Free over`, no amount, no `0 ₫`. The table has no row containing `Express`. The word equals, character for character, the Delivery value on `/cart` (`aside dl > div:nth-child(2) > dd`).
Traces: SC-14 (SITE-009 AC-5)
Check: render_url

**Scenario**: TC-19 — the table's "Free" and the bag's "Free" are the same string in the code
**Given**: The pull request's `Shipping.tsx` and `Cart.tsx`, and `src/content.json`.
**When**: Where each draws the word "Free" is read.
**Then**: Both read it from the content key `pages.cart.free` (one `useContent("pages.cart.free")`, no literal `"Free"` and no `formatPrice(0)` in `Shipping.tsx`'s cost cell), so changing that one string changes both.
Traces: SC-14 (SITE-009 AC-5)
Check: manual — a property of where the code reads a string; the content file cannot be edited and rebuilt by any instrument here, so there is no way to change the key and look. TC-18 observes that the two words are equal today.

**Scenario**: TC-20 — each figure and cell shows its own stored value
**Given**: The shop's `standard_days` changed to `3–5` and the delivery record's `days` changed to `3–5 working days` in a copy of the data.
**When**: `/shipping` is loaded from that copy.
**Then**: The second figure reads `3–5` and the Standard row's Arrives cell reads `3–5 working days`; with the original values they read `2–4` and `2–4 working days` (TC-13, TC-16).
Traces: SC-15 (SITE-009 AC-8)
Check: manual — the records ship inside the build and no instrument edits a file and rebuilds (ambiguity 5); verify by reading that `Shipping.tsx` takes the figure from `terms.standard_days` and the cell from the record's `days`, with no literal `2–4` in it.

**Scenario**: TC-21 — on a phone the table scrolls inside its box and the page does not
**Given**: A 390×844 window.
**When**: `/shipping` is loaded; the table's wrapper (the table's parent element) and the document are measured.
**Then**: The wrapper's computed `overflow-x` is `auto` (or `scroll`) and its `scrollWidth` is greater than its `clientWidth`; `document.documentElement.scrollWidth` is at most 390, so the page itself has no horizontal scroll.
Traces: SC-16 (SITE-009 AC-9; NFR Responsive)
Check: measure_styles

**Scenario**: TC-22 — "Track an order" goes to the tracking page
**Given**: A 1280×800 window on `/shipping`.
**When**: The pill `Track an order` beside "Questions about delivery" is clicked.
**Then**: Its `href` is `/track` and the address after the click is `/track` with the heading `Track an order` shown.
Traces: SC-16 (SITE-009 AC-7)
Check: interact_page

**Scenario**: TC-23 — the page's description says free delivery and how long, in one sentence
**Given**: A 1280×800 window.
**When**: `/shipping` is loaded and the content of `meta[name="description"]` in the document head is read.
**Then**: It is one sentence (a single full stop, at the end) that says delivery is free on every order (contains `free`, case-insensitive) and how long it takes (contains `2–4` or `two to four`); it contains none of `cost`, `fee`, `express`, `where express reaches`.
Traces: SC-34 (story §3.3 key `pages.shipping.description`; SITE-016 item 4)
Check: interact_page

---

## C. Bag, promises and payment words (SC-17 … SC-20)

**Scenario**: TC-24 — a bag of one 3.090.000 ₫ pair: Delivery Free, Total = Subtotal
**Given**: Arrangement A (one pair of `samba-og-w-crystal-white`, 3.090.000 ₫); 1280×800.
**When**: `/cart` is read.
**Then**: `aside dl > div:nth-child(1) > dd` (Subtotal) reads `3.090.000 ₫`; `div:nth-child(2) > dd` (Delivery) reads exactly `Free`; `div:nth-child(3) > dd` (Total) reads `3.090.000 ₫`. The Delivery value is not `0 ₫` and not `30.000 ₫`.
Traces: SC-17 (SITE-016 AC-3; SITE-007 AC-8)
Check: interact_page

**Scenario**: TC-25 — under, at and over the old threshold, Delivery is Free and Total equals Subtotal
**Given**: Arrangement B, four times, each time on a fresh bag: (a) one line priced 1.500.000 ₫; (b) one line priced 1.999.999 ₫; (c) one line priced exactly 2.000.000 ₫; (d) two lines, 3.090.000 ₫ and 1.500.000 ₫.
**When**: `/cart` is read after each arrangement.
**Then**: (a) Subtotal `1.500.000 ₫`, Delivery `Free`, Total `1.500.000 ₫`. (b) Subtotal `1.999.999 ₫`, Delivery `Free`, Total `1.999.999 ₫`. (c) Subtotal `2.000.000 ₫`, Delivery `Free`, Total `2.000.000 ₫`. (d) Subtotal `4.590.000 ₫`, Delivery `Free`, Total `4.590.000 ₫`. In none is Delivery `0 ₫` or any amount (the old design charged 30.000 ₫ under 2.000.000 ₫, so (a) and (b) would read Total `1.530.000 ₫` and `2.029.999 ₫` if the fee survived).
Traces: SC-18 (SITE-016 AC-3, boundary; SITE-007 AC-9)
Check: interact_page

**Scenario**: TC-26 — the line under "Check out" names VNPay only
**Given**: Arrangement A; 1280×800.
**When**: `/cart` is read.
**Then**: The paragraph under the `Check out` link (`[data-testid='cart-checkout']`) reads exactly `You pay online with VNPay — card, domestic ATM card or QR — at the next step.` The page's main content contains none of `cash on delivery`, `Bank transfer`, `MoMo`, `cash`, `pay on delivery` (case-insensitive).
Traces: SC-19 (SITE-007 AC-10; SITE-016 item 5)
Check: interact_page

**Scenario**: TC-27 — the home page's four promises
**Given**: A 1280×800 window; standard days `2–4`.
**When**: `/` is loaded and the promise row (the `ul` of four `li` cells) is read.
**Then**: The four cells read, in order: `Free delivery` / `2–4 working days, anywhere in Vietnam.`; `No exchanges` / `Check the size chart before you order.`; `Pay online` / `Card, domestic ATM card or QR, through VNPay.`; `Genuine adidas` / `Every pair boxed, tagged and sent with our receipt.` The row contains none of `Cash on delivery`, `Open the box`, `fee`, `Free over`, `exchange days`.
Traces: SC-20 (SITE-004 AC-10; SITE-016 item 5)
Check: render_url

**Scenario**: TC-28 — the same four promises in the bag and on the tracking page
**Given**: Arrangement A; 1280×800.
**When**: The promise row on `/cart` (compact) is read, then the promise row on `/track` (compact).
**Then**: On both, the four cells read exactly as in TC-27, in the same order; the third cell is `Pay online` / `Card, domestic ATM card or QR, through VNPay.` and neither row contains `Cash on delivery`.
Traces: SC-20 (SITE-004 AC-10)
Check: interact_page

**Scenario**: TC-29 — the third promise's icon is a card, not a banknote
**Given**: A 1280×800 window on `/`.
**When**: The `svg` inside the third `li` of the promise row is inspected (its class list and drawing).
**Then**: It is Lucide's credit-card icon (class `lucide-credit-card`; a rounded rectangle with a horizontal stripe); its class list does not contain `lucide-banknote`; the first, second and fourth cells keep their own icons (truck, info, quality mark), none a card.
Traces: SC-20 (story §8 decision 6)
Check: interact_page

---

## D. Checkout (SC-21 … SC-25)

**Scenario**: TC-30 — step 02 is one row, "Free delivery", with "Free" at its right
**Given**: Arrangement A; 1280×800. The `Check out` link `[data-testid='cart-checkout']` is clicked.
**When**: `/checkout` is read, step `02 Delivery`.
**Then**: Exactly one element matches `[data-testid^='delivery-']` and it is `[data-testid='delivery-standard']`; no `delivery-express` exists. Its text contains `Free delivery`, `2–4 working days` and `Nationwide with GHN or GHTK, tracked from our door to yours.`, and it ends with `Free` at its right end; it contains neither `Express`, `Next working day`, `30.000` nor `0 ₫`.
Traces: SC-21 (SITE-016 item 3; ORD-001 AC-3, AC-4)
Check: interact_page

**Scenario**: TC-31 — the delivery row and the payment row are not choices
**Given**: Arrangement A; `/checkout` open at 1280×800.
**When**: The page is searched for radio inputs; the row `[data-testid='delivery-standard']` and the row `[data-testid='pay-vnpay']` are each clicked; the Tab key is pressed from `#checkout-note`.
**Then**: The page has no `input[type=radio]` at all; neither click changes the page (the same rows, the same Total, the address is unchanged); focus after Tab is not on a radio input.
Traces: SC-21, SC-23 (story §8 decision 4)
Check: interact_page

**Scenario**: TC-32 — the delivery row is drawn selected
**Given**: Arrangement A; `/checkout` open at 1280×800.
**When**: The round marker at the left of `[data-testid='delivery-standard']` (its first `span`) is measured, and the same for `[data-testid='pay-vnpay']`.
**Then**: In each marker an inner filled dot exists whose computed `background-color` is the ink colour `rgb(8, 8, 8)`, and the marker's computed `border-color` is the ink colour as well.
Traces: SC-21 (story §8 decision 4, "the design's selected-dot look stays"), SC-23
Check: measure_styles

**Scenario**: TC-33 — the order summary: Free, and a total equal to the pairs total, under, at and over 2.000.000 ₫
**Given**: Arrangement B for each of: (a) one line 3.090.000 ₫; (b) one line 1.500.000 ₫; (c) one line 1.999.999 ₫; (d) one line 2.000.000 ₫; (e) two lines 3.090.000 ₫ and 1.500.000 ₫. Each time `/checkout` is opened at 1280×800.
**When**: The summary "Your order" (`aside dl`) and `[data-testid='checkout-total']` are read.
**Then**: Subtotal / Delivery / Total read (a) `3.090.000 ₫` / `Free` / `3.090.000 ₫`; (b) `1.500.000 ₫` / `Free` / `1.500.000 ₫`; (c) `1.999.999 ₫` / `Free` / `1.999.999 ₫`; (d) `2.000.000 ₫` / `Free` / `2.000.000 ₫`; (e) `4.590.000 ₫` / `Free` / `4.590.000 ₫`. `[data-testid='checkout-total']` equals the Total row each time. Delivery is never `0 ₫`.
Traces: SC-22 (SITE-016 AC-3, boundary; ORD-001 AC-5, AC-7, AC-8)
Check: interact_page

**Scenario**: TC-34 — step 03 offers VNPay alone, already selected
**Given**: Arrangement A; `/checkout` open at 1280×800.
**When**: Step `03 Payment` and the line under the totals are read.
**Then**: `[data-testid='pay-vnpay']` is the only element matching `[data-testid^='pay-']`; its text is `VNPay — ATM, Visa, Mastercard` and `Pay by card or banking app on VNPay's secure page.`; `pay-cod`, `pay-bank` and `pay-momo` do not exist. The page's main content contains none of `Cash on delivery`, `Bank transfer (VietQR)`, `MoMo wallet`, `text message`, `pay the courier`, `in cash`. The line under the totals begins `After you press Place order you pay on the VNPay` (VNPay is the selected method; the initial payment is not `cod`).
Traces: SC-23 (SITE-016 item 6; ORD-006 AC-1, AC-3)
Check: interact_page

**Scenario**: TC-35 — an order placed at checkout is free delivery with the pairs total
**Given**: Arrangement A (one pair, 3.090.000 ₫), `/checkout` open at 1280×800.
**When**: `#checkout-name` is filled `Test Buyer`, `#checkout-phone` `0912 345 678`, `#checkout-address` `1 Example Street, Ward 1`, `#checkout-city` `Hà Nội`; `[data-testid='place-order']` is clicked and the redirect is waited for.
**Then**: The address becomes `/orders/doi-<n>?placed=1`; `[data-testid='order-status']` reads exactly `Thank you — your order is placed.`; the order's Total reads `3.090.000 ₫`; the Delivery detail reads `Standard · 2–4 working days · Free`; the Payment detail begins `VNPay — ATM, Visa, Mastercard ·` and does not contain `Cash on delivery`; the bag is empty afterwards (`[data-testid='bag-count']` absent or `0`).
Traces: SC-24 (SITE-016 item 3)
Check: interact_page

**Scenario**: TC-36 — the order saved in the browser carries no delivery fee
**Given**: The order of TC-35 has just been placed.
**When**: The browser's `localStorage` entry `doi:orders` is read (a JSON array, newest first).
**Then**: The first order has `delivery` `standard`, `payment` `vnpay`, `total` 3090000 equal to the sum of its lines' `price × qty`, and no `delivery_fee` key (a stored `0` is also a fee field the story removed). If the instrument cannot read storage, the same facts are read from the order page of TC-35 and the item is reported with that limit.
Traces: SC-24 (story §3.2 `lib/mock/orders.ts`)
Check: interact_page

**Scenario**: TC-37 — an incomplete form places no order and keeps what was typed
**Given**: Arrangement A; `/checkout` open at 1280×800; `#checkout-name` filled `Test Buyer`, `#checkout-phone` filled `0912 345 678`, address and city left empty.
**When**: `[data-testid='place-order']` is clicked.
**Then**: The address stays `/checkout`; an alert reads `Fill in your name, street address and city.`; `#checkout-name` still holds `Test Buyer` and `#checkout-phone` still holds `0912 345 678`; the bag still counts 1 (`[data-testid='bag-count']`); the Total still reads `3.090.000 ₫`; `localStorage` `doi:orders` has no new entry.
Traces: SC-24 (failure path of the same form; ORD-001)
Check: interact_page

**Scenario**: TC-38 — at 390px checkout is one column without sideways scroll
**Given**: Arrangement A; a 390×844 window; `/checkout` open.
**When**: The checkout `form`'s computed `grid-template-columns` and the document's width are measured.
**Then**: The form has one column track (the summary `aside` sits below the three steps: its top edge is at or below the bottom edge of step 03); `document.documentElement.scrollWidth` is at most 390.
Traces: SC-25 (ORD-001 AC-2; NFR Responsive)
Check: measure_styles

---

## E. Order and tracking pages (SC-26 … SC-28)

**Scenario**: TC-39 — sample orders show Standard · 2–4 working days · Free and the pairs total
**Given**: A 1280×800 window.
**When**: `/orders/doi-24817`, `/orders/doi-24815`, `/orders/doi-24810` and `/orders/doi-24806` are loaded and the "Delivery and payment" details and the Total beside "In this order" are read.
**Then**: On all four the Delivery detail reads exactly `Standard · 2–4 working days · Free` (no `Express`, no `Next working day`, no `0 ₫`). Totals read: DOI-24817 `3.090.000 ₫`; DOI-24815 `5.890.000 ₫`; DOI-24810 `3.290.000 ₫`; DOI-24806 `3.090.000 ₫`. The Payment detail of DOI-24817 still reads `MoMo wallet · paid`, and DOI-24815, -24810 and -24806 read `VNPay — ATM, Visa, Mastercard · paid`.
Traces: SC-26 (SITE-016 AC-3; ORD-007 AC-22)
Check: render_url

**Scenario**: TC-40 — an unpaid order says "awaiting payment", never "to pay on delivery"
**Given**: A 1280×800 window.
**When**: `/orders/doi-24816` and `/orders/doi-24814` (both unpaid, stored payment `cod`) are loaded and the Payment detail and the page text are read.
**Then**: The Payment detail reads `Cash on delivery · awaiting payment` — the stored `cod` record's name, then the words `awaiting payment` (ambiguity 1: this is the one place the name remains). The text `to pay on delivery` appears nowhere on either page; the Delivery detail on each reads `Standard · 2–4 working days · Free`; DOI-24816's Total reads `3.290.000 ₫` and DOI-24814's `2.290.000 ₫`.
Traces: SC-27 (SITE-016 AC-7; ORD-007 AC-13)
Check: render_url

**Scenario**: TC-41 — the "Confirmed" step says it was paid online
**Given**: A 1280×800 window on `/track`.
**When**: The row `Confirmed` under "What each step means" is clicked.
**Then**: It opens onto exactly `Paid online; the pair is set aside for you.`; the answer does not contain `cash`, `phone` or `cash on delivery`; the other four steps read as before (`We have the order and are checking the pair against it.`, `Boxed with its tags and our receipt, waiting for the courier.`, `With GHN or GHTK; the tracking number is on the order's page.`, `In your hands. If anything is wrong with the delivery, message us.`).
Traces: SC-28 (SITE-016 AC-7; ORD-020 item 2)
Check: interact_page

---

## F. Shoe page and contact page (SC-29 … SC-31)

**Scenario**: TC-42 — "Payment options" on a shoe page lists VNPay alone
**Given**: A 1280×800 window; run on `/shop/samba-og-w-crystal-white`, `/shop/samba-og-kids-cloud-white` and `/shop/handball-spezial-w-earth-strata`.
**When**: The row `Payment options` is clicked open on each page and its list is read.
**Then**: The list has exactly one entry on each page: a card icon (`svg` with class `lucide-credit-card`), then `VNPay — ATM, Visa, Mastercard.` and `Pay by card or banking app on VNPay's secure page.` The text `Cash on delivery`, `Bank transfer`, `MoMo` and `marked paid at once` is not in the row.
Traces: SC-29 (SITE-006 AC-2; SITE-016 item 5)
Check: interact_page

**Scenario**: TC-43 — the delivery and returns row on a shoe page is unchanged
**Given**: A 1280×800 window on `/shop/samba-og-w-crystal-white`.
**When**: The row `Delivery & return policy` is clicked open; then its link `Delivery` is clicked, and from the shoe page again its link `Return policy`.
**Then**: The text reads exactly `Free standard delivery in 2–4 working days, sent the same day if you order before 15:00. Exchanges are not accepted.` The link `Delivery` has `href` `/shipping` and lands on `/shipping`; the link `Return policy` has `href` `/returns` and lands on `/returns`.
Traces: SC-30 (SITE-006 AC-3, AC-4)
Check: interact_page

**Scenario**: TC-44 — the four "Already answered" links on the contact page
**Given**: A 1280×800 window.
**When**: `/contact` is loaded, the links under the heading `Already answered` are read, and the second is clicked.
**Then**: Four links read, in order, `Find your size` → `/size-chart`, `Free delivery and arrival times` → `/shipping`, `Return policy` → `/returns`, `Track an order` → `/track`; none reads `Delivery times and fees`. After the click the address is `/shipping`.
Traces: SC-31 (SITE-012 AC-13; SITE-016 item 4)
Check: interact_page

---

## G. Whole site (SC-32 … SC-34)

**Scenario**: TC-45 — the shipped code and data hold no removed field, express option or delivery fee
**Given**: The site is running; the site's own modules and data are read as described above (library code is ignored).
**When**: They are searched.
**Then**: None of `free_delivery_from`, `express_days`, `express_cities`, `free_from`, `next_cod`, `cod_title`, `cod_line` appears. The word `express` (whole word, any case) appears nowhere in the site's own data and words. No delivery data or `delivery_fee` field carries `30000` or `50000`. `ShopTerms`' type is erased in a build, so its fields are read in TC-46.
Traces: SC-32 (SITE-016 AC-1, item 3)
Check: fetch_url

**Scenario**: TC-46 — no code computes a delivery fee or a free-delivery threshold
**Given**: The pull request's diff and the files `src/pages/Cart.tsx`, `Checkout.tsx`, `Order.tsx`, `Shipping.tsx`, `Product.tsx`, `src/components/PromiseRow.tsx`, `src/lib/shop.ts`, `src/lib/mock/orders.ts`.
**When**: `code/frontend/src` is searched for `free_delivery_from`, `express_days`, `express_cities`, `\bexpress\b`, `\b(30000|50000)\b`, `delivery_fee`, `feeOf`, `fee`, `cash on delivery`, `pay on delivery`, `\bcod\b`; and the files above are read for a variable, function or branch that computes a delivery fee or a threshold.
**Then**: There are no hits other than: `components/ui/LICENSE.md` (licence text), `3000000`-style price bands (unrelated), the `delivery.fee` field and `0` value in `schema.json` / `delivery.json`, and the `cod` places named in TC-47. `ShopTerms` declares none of `free_delivery_from`, `express_days`, `express_cities`, `exchange_days`, `refund_days`. `Cart.tsx` reads neither the shop terms nor the delivery record; `Checkout.tsx` has no `delivery` state, no `feeOf`, no `"express"` branch, no literal `4`/`1` delivery days and no `useState("cod")`; `lib/mock/orders.ts` has no `delivery_fee` on `NewOrder` or `PlacedOrder`; `Shipping.tsx` has no `express_*`, `free_from` or `cutoff` read.
Traces: SC-32 (SITE-016 AC-1, item 3)
Check: manual — a search of source for logic, variables and branches; a build erases names and types, so no HTTP or browser instrument sees them. Read from the PR files.

**Scenario**: TC-47 — no cash-on-delivery wording in the site's words, questions or reviews; `cod` only where allowed
**Given**: The site is running; `content.json`, `faqs`, `reviews`, `payments` and `schema` are read as described above.
**When**: Every string value in `content.json`, `faqs` and `reviews` is searched (case-insensitive).
**Then**: `cash on delivery`, `pay on delivery`, `pay the courier` and `paid cash` appear in none of those three. Across all the shipped data and words, `Cash on delivery` appears exactly twice (the `cod` record's `name` in `payments`, and the `orders.payment` option label in `schema`) and `Pay the courier in cash` exactly once (the `cod` record's `note`). The id `cod` is found only in: the `cod` record of `payments`, the `cod` option of `schema`, and the five sample orders `doi-24816`, `doi-24814`, `doi-24811`, `doi-24808`, `doi-24804`.
Traces: SC-33 (SITE-016 AC-7, item 6)
Check: fetch_url

**Scenario**: TC-48 — customer pages never offer cash on delivery or express; only the five old orders name the stored record
**Given**: A 1280×800 window.
**When**: These pages are loaded and their main-content text searched: `/`, `/shop`, `/shop/samba-og-w-crystal-white`, `/shop/samba-og-w-wonder-quartz`, `/size-chart`, `/shipping`, `/returns`, `/about`, `/contact`, `/track`, `/orders/doi-24817`, `/orders/doi-24815`; then `/orders/doi-24816` and `/orders/doi-24804`.
**Then**: On the first twelve, `cash on delivery`, `pay on delivery`, `pay the courier` and `express` (whole word, case-insensitive) appear nowhere. On `/orders/doi-24816` and `/orders/doi-24804` the only mention is the Payment detail naming `Cash on delivery` (`· awaiting payment` and `· paid`); neither page contains `to pay on delivery`, `pay the courier` or `express`.
Traces: SC-33 (SITE-016 AC-7), ambiguity 1
Check: render_url

**Scenario**: TC-49 — the content keys changed are exactly the ones the story lists, with the new words
**Given**: The site is running; `content.json` is read as described above.
**When**: The listed keys are read.
**Then**: `pages.cart.pay_line` is `You pay online with VNPay — card, domestic ATM card or QR — at the next step.`; `pages.checkout.delivery_title` is `Free delivery`; `pages.checkout.next_cod` does not exist; `pages.contact.quick[1].label` is `Free delivery and arrival times` (its `href` `/shipping`); `pages.order.due` is `awaiting payment`; `pages.shipping.lead` is `Every pair leaves boxed, tagged and tracked, and delivery is free on every order.`; `pages.shipping.figures.free` is `Delivery on every order, whatever it comes to — that is every pair in the shop.`; `pages.shipping.figures.standard` is `Working days by standard delivery, anywhere in Vietnam.`; `pages.shipping.figures.express`, `pages.shipping.cutoff` and `pages.shipping.free_from` do not exist; `pages.shipping.columns.cost` is `Cost` and `pages.shipping.columns.fee` does not exist; `pages.shipping.options_heading` is `Delivery options`; `pages.track.steps.confirmed` is `Paid online; the pair is set aside for you.`; `promises.pay_title` is `Pay online`, `promises.pay_line` is `Card, domestic ATM card or QR, through VNPay.`, and `promises.cod_title` / `promises.cod_line` do not exist; `promises.delivery_title` (`Free delivery`), `promises.delivery_line` (`{days} working days, anywhere in Vietnam.`), `promises.exchange_title` (`No exchanges`) and `promises.genuine_title` (`Genuine adidas`) are unchanged; `pages.checkout.next_online` is untouched (ambiguity 6).
Traces: SC-34 (story §3.3), SC-10, SC-19, SC-20, SC-28, SC-31
Check: fetch_url

**Scenario**: TC-50 — nothing else changes, and the build, lint and tests are clean
**Given**: The pull request at its head; its CI result and the output of `npm run lint`, `npm run build` and `npm test` in `code/frontend`.
**When**: The diff of `src/content.json` is read key by key and the three commands' results are read.
**Then**: The diff of `content.json` changes only the keys in story §3.3 (TC-49's list) and no other page's words; no colour, spacing or layout class changes beyond the two-column figures row on `Shipping.tsx`; `src/theme.css` and `src/index.css` are not in the diff; lint, build and tests all pass with no new warning.
Traces: SC-34 (SITE-016 item 7)
Check: manual — a diff read and the result of three commands; the instruments here observe a running page, and "looks unchanged" needs a baseline screenshot nobody stored. The pages' words and layout that this story does change are measured in TC-12 to TC-44.

---

## Coverage — criteria against cases

| Criterion | Cases |
|---|---|
| SC-1 | TC-01 |
| SC-2 | TC-02 |
| SC-3 | TC-03 |
| SC-4 | TC-04, TC-05 |
| SC-5 | TC-06 |
| SC-6 | TC-07 |
| SC-7 | TC-08 |
| SC-8 | TC-09 |
| SC-9 | TC-10, TC-11 |
| SC-10 | TC-12 |
| SC-11 | TC-13, TC-14 |
| SC-12 | TC-15 |
| SC-13 | TC-16, TC-17 |
| SC-14 | TC-18, TC-19 |
| SC-15 | TC-20 |
| SC-16 | TC-21, TC-22 |
| SC-17 | TC-24 |
| SC-18 | TC-25 |
| SC-19 | TC-26 |
| SC-20 | TC-27, TC-28, TC-29 |
| SC-21 | TC-30, TC-31, TC-32 |
| SC-22 | TC-33 |
| SC-23 | TC-31, TC-32, TC-34 |
| SC-24 | TC-35, TC-36, TC-37 |
| SC-25 | TC-38 |
| SC-26 | TC-39 |
| SC-27 | TC-40 |
| SC-28 | TC-41 |
| SC-29 | TC-42 |
| SC-30 | TC-43 |
| SC-31 | TC-44 |
| SC-32 | TC-45, TC-46 |
| SC-33 | TC-47, TC-48 |
| SC-34 | TC-23, TC-49, TC-50 |

**Instruments:** `fetch_url` 11 (TC-01, 02, 03, 04, 06, 07, 11, 45, 47, 49 and the shipped-data half of TC-36's fallback), `render_url` 9, `measure_styles` 6, `interact_page` 19, `manual` 5 (TC-05, 19, 20, 46, 50). Each `manual` case names why no instrument answers it. Where an arrangement needs `localStorage` to be written (arrangement B: TC-25, TC-33) and the instrument cannot write it, those two cases are UNVERIFIED for the sub-2.000.000 ₫ baskets only; the catalogue baskets (TC-24, TC-33 (a), TC-35) are unaffected.

**Roles and failure paths.** The story has no roles and no permission rules (SITE-016: "it has no roles"): none is cased. Its one failure path — Place order with an incomplete form — is TC-37; the other failure the story names, a basket under the old threshold, is TC-25 and TC-33 (b)(c). The API contract has no endpoint this story touches, so no status-code or unknown-field case applies.
