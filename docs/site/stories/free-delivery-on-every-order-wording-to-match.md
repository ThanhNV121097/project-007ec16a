# Story — Free delivery on every order, wording to match

Module: `site` · Plan item 2 · Requirement: SITE-016 (with the lines of SITE-004, SITE-006, SITE-007, SITE-009, SITE-012 and, where this story touches checkout, the order page and the track page, ORD-001, ORD-006, ORD-007 and ORD-020 that it must agree with)
Shape: frontend and JSON only — no backend, no database, no new secret.

## 1. User story

As a **customer** (a visitor about to buy), I want every page to say that delivery costs nothing on every order and that I pay online through VNPay, so that the total I see in the bag and at checkout is the total I pay and nothing on the site offers me something the shop does not do.

## 2. What the owner decided (the reason for this story)

- **Free delivery** — every order ships free: no fee, no free-shipping threshold, no priced express option.
- **Online payment** — customers pay online only, through VNPay (card, domestic ATM card, QR). There is no cash on delivery.

The built design still shows fees, a 2.000.000 ₫ threshold, an Express option, cash on delivery, and words that say so. This story makes the built site say what the owner decided. It lands **before** "Site data from the database" and "Checkout and placing an order", which seed and build on these files.

## 3. In scope

### 3.1 Records (`code/frontend/src/data/`)

| File | Change |
|---|---|
| `delivery.json` | One record, `standard` (name Standard, days "2–4 working days", the note unchanged), fee 0. The `express` record is removed. |
| `shop.json` | `free_delivery_from`, `express_days`, `express_cities` removed. `standard_days` ("2–4"), `dispatch_cutoff` ("15:00"), `reply_time`, `rating`, `review_count` stay. (`exchange_days` and `refund_days` were removed by the returns story and stay gone.) |
| `schema.json` | The `shop` collection drops the three fields above; `orders.delivery` options list `["standard", "Standard"]` only. The `delivery.fee` field and the `cod` option of `orders.payment` stay (the `cod` records stay in the data, never offered). |
| `orders.json` | DOI-24815, DOI-24810 and DOI-24806 (express) become delivery `standard`, with totals 5.890.000 ₫, 3.290.000 ₫ and 3.090.000 ₫ (the 50.000 ₫ express fee leaves each total). Every sample order's total equals its shoe's price × pairs. Nothing else on a sample order changes (status, carrier, tracking, expected date, payment, customer). |
| `payments.json` | The four rows stay in their order. Only the `vnpay` note changes: "Pay by card or banking app on VNPay's secure page." (the clause "the order is marked paid at once" goes — only VNPay's own notice will ever mark an order paid). `cod`, `bank` and `momo` are untouched; `momo`'s same clause is reworded by the MoMo story when it is first offered. |
| `faqs.json` | `f-delivery-time` answer: "Two to four working days anywhere in Vietnam by standard delivery." `f-payment-cod` ("Can I pay cash on delivery?") is replaced, in the same position, by `f-payment-how`: question "How do I pay?", answer "Online through VNPay, by card, domestic ATM card or QR." `f-payment-genuine` is unchanged. `f-orders-number` answer: "On the page you saw after paying, starting with DOI-." |
| `reviews.json` | `r-04` loses "Express arrived next morning." `r-07` loses "Paid cash at the door, checked the box first." (a live review of the Samba OG Wonder Quartz that tells buyers cash on delivery exists; the SRS names only `r-04` — the SRS is amended in the same PR). Both are sample reviews (`_source: assumed`). |

### 3.2 Code (`code/frontend/src/`)

| File | Change |
|---|---|
| `lib/shop.ts` | `ShopTerms` loses `free_delivery_from`, `express_days`, `express_cities`. |
| `pages/Cart.tsx` | No fee and no threshold: no read of the shop terms or of the delivery record. Delivery always reads the `pages.cart.free` string; Total = Subtotal. |
| `pages/Checkout.tsx` | Step 02 is one row, "Free delivery", already selected, not a choice (no radio input, no delivery state, no Express branch, no hard-coded express days); "Free" at the right. Step 03 lists the `vnpay` record alone, already selected, not a choice; `cod`, `bank` and `momo` are not listed; the initial payment is no longer `cod`; the cash-on-delivery "what happens next" branch (`next_cod`) is gone. The summary shows Delivery as "Free" and Total = pairs total. |
| `lib/mock/orders.ts` | No `delivery_fee` on a new or placed order; `total` = the pairs total; the expected date uses the standard option only. |
| `pages/Order.tsx` | The "Delivery" detail reads `<name> · <arrives in> · Free` (the `pages.cart.free` string); the total is the pairs total. |
| `pages/Shipping.tsx` | Two figures (not three); no "Free over" line, no cut-off line, no cities; the options table's cost cell reads the `pages.cart.free` string, never `0 ₫`; the figures row is two columns from 640px. |
| `components/PromiseRow.tsx` | The third cell is "Pay online" with the card icon, read from `promises.pay_title` / `promises.pay_line` (renamed from `cod_*`). |
| `pages/Product.tsx` | "Payment options" lists the `vnpay` record alone (SITE-006 AC-2; SRS open question 7 — the same story edits it). The delivery sentence is already right and stays. |

### 3.3 Words (`src/content.json`)

This story is the one place the architecture overview's "a story never edits `content.json`" is lifted: the owner's two decisions *are* words in that file. Exactly these keys change, and no others:

| Key | New value |
|---|---|
| `pages.cart.pay_line` | "You pay online with VNPay — card, domestic ATM card or QR — at the next step." |
| `pages.checkout.next_cod` | removed |
| `pages.checkout.delivery_title` (new) | "Free delivery" (the Checkout row's name) |
| `pages.contact.quick[1].label` | "Free delivery and arrival times" |
| `pages.order.due` | "awaiting payment" (was "to pay on delivery") |
| `pages.shipping.lead` | "Every pair leaves boxed, tagged and tracked, and delivery is free on every order." |
| `pages.shipping.description` | one sentence saying delivery is free on every order and how long it takes; no "costs", no "express" |
| `pages.shipping.figures.free` | "Delivery on every order, whatever it comes to — that is every pair in the shop." |
| `pages.shipping.figures.express` | removed |
| `pages.shipping.columns.fee` | renamed `cost`, value "Cost" |
| `pages.shipping.cutoff`, `pages.shipping.free_from` | removed |
| `pages.shipping.options_heading` | "Delivery options" |
| `pages.track.steps.confirmed` | "Paid online; the pair is set aside for you." |
| `promises.cod_title` / `promises.cod_line` | renamed `pay_title` "Pay online" / `pay_line` "Card, domestic ATM card or QR, through VNPay." |

The delivery days (2–4 working days), the carriers (GHN / GHTK) and the coverage line ("Nationwide with GHN or GHTK, tracked from our door to yours.") stay as the design suggested. The owner has not given real ones and will correct them in the Studio (SRS open question 6).

## 4. Out of scope

- **Backend, API and database** — the data story seeds the files as this story leaves them; checkout's order, price re-read and stock are the Checkout story.
- **Checkout's "what happens next" line for VNPay** (`pages.checkout.next_online`, which still says the order is "marked paid at once and packed today"), the "VNPay's keys are missing" state, and deleting `lib/mock/orders.ts` — belong to the Checkout story (ORD-001 AC-10, ORD-006). This story only removes the cash-on-delivery branch and the fee.
- **Real delivery areas, days and carrier** — not given by the owner; the design's suggestions stay (see 3.3).
- **Bank transfer and MoMo** — their records and the `momo` note stay for the later payments story.
- **The five sample orders that were paid "Cash on delivery"** (DOI-24816, -24814, -24811, -24808, -24804) keep their stored payment. They are placeholder records; their page names the stored `cod` record. They are not an offer.
- **"Text message" / "confirmation message" wording** on the order, track and checkout pages ("we text you when the pair is on its way") — ORD open question 4, the owner's to answer.
- **The return-fee FAQ `f-returns-cost`** named in the plan item no longer exists (the returns story removed the invented return terms); nothing to do. The Returns page and its exchange wording are untouched.
- **The shoe page's same-day cut-off** ("Sent today if you order before 15:00", the delivery row's "sent the same day …") stays; only the Delivery page drops the cut-off.
- **Design documents** — `design/design-system.md` §2.1.17 and §2.1.21 still say COD and "size exchange"; PM syncs them after this lands. No code is written for that.
- **Theme tokens, layout, pictures, colours** — no change beyond the two-column figures row.

## 5. UI scope

Screens touched, all drawn already in the approved design; no new screen and no new state:

| Screen | What changes |
|---|---|
| Delivery `/shipping` | Lead, two figures, "Delivery options" table with a "Cost" column showing "Free", no cut-off line, no "Free over" line |
| Bag `/cart` | Delivery "Free" always; Total = Subtotal; payment line; third promise |
| Checkout `/checkout` | Step 02 single pre-selected "Free delivery" row; step 03 VNPay alone; summary Delivery "Free", Total = pairs total |
| Order `/orders/:id` | Delivery detail ends "· Free"; unpaid payment reads "awaiting payment"; total = pairs total |
| Shoe `/shop/:id` | "Payment options" row lists VNPay alone |
| Home `/`, bag (compact) | Promise row: "Free delivery", "No exchanges", "Pay online", "Genuine adidas" |
| Contact `/contact` | "Already answered" link label |
| Track `/track` | "Confirmed" step sentence |

Phone first: every page above keeps one column from 320px and no horizontal page scroll at 390px; the Delivery table scrolls inside its own box.

## 6. Acceptance criteria

Each criterion narrows the SRS criterion in its brackets. `SC` is this story's number; `AC` is the SRS's.

### Records

- **SC-1 [SITE-016 AC-2]:** The `delivery` records are read: there is exactly one, `standard`, with fee 0, days "2–4 working days" and the note "Nationwide with GHN or GHTK, tracked from our door to yours."; no record has the id `express`.
- **SC-2 [SITE-016 AC-1]:** The `shop` record is read: it has no `free_delivery_from`, `express_days` or `express_cities` (and still none of `exchange_days`, `refund_days`); `standard_days` is "2–4" and `dispatch_cutoff` is "15:00".
- **SC-3 [SITE-016 item 2]:** `schema.json` is read: the `shop` collection declares none of the three removed fields, and the `orders.delivery` options are `[["standard","Standard"]]` only; the `delivery.fee` field and the `orders.payment` options are unchanged.
- **SC-4 [SITE-016 AC-4]:** The sample orders are read: each has delivery `standard`; DOI-24815 totals 5.890.000 ₫, DOI-24810 3.290.000 ₫, DOI-24806 3.090.000 ₫; for every sample order, total = the shoe's price × pairs; no other field of any sample order differs from before.
- **SC-5 [SITE-016 AC-8]:** `payments.json` is read: the four rows `cod`, `bank`, `vnpay`, `momo` are present in that order; the `vnpay` note is "Pay by card or banking app on VNPay's secure page." and has no "marked paid at once"; the other three rows are byte-for-byte as before.

### Questions and reviews

- **SC-6 [SITE-016 AC-6]:** The questions of topic "payment" are read: "Can I pay cash on delivery?" is gone; a row "How do I pay?" answers "Online through VNPay, by card, domestic ATM card or QR." and comes first; "Are the shoes genuine?" is unchanged; no payment row mentions cash on delivery.
- **SC-7 [SITE-016 AC-8]:** `f-delivery-time` reads "Two to four working days anywhere in Vietnam by standard delivery." and contains no "Express"; the other three questions of topic "shipping" are unchanged, so `/shipping` still shows four closed rows.
- **SC-8 [SITE-016 AC-8]:** `f-orders-number` reads "On the page you saw after paying, starting with DOI-." and contains no "text message".
- **SC-9 [SITE-016 AC-8, item 6]:** Review r-04 reads "Bought them as a birthday gift for my sister. She has not taken them off."; review r-07 reads "The pink is softer than in photos, which I prefer."; no review mentions express delivery or paying cash.

### Delivery page

- **SC-10 [SITE-009 AC-1]:** `/shipping` loads: the lead reads "Every pair leaves boxed, tagged and tracked, and delivery is free on every order."; the first figure reads "Free" and its line "Delivery on every order, whatever it comes to — that is every pair in the shop."
- **SC-11 [SITE-009 AC-2]:** The same page: the second figure reads `2–4` with "Working days by standard delivery, anywhere in Vietnam."; there are exactly two figures and no third; at 1280px and at 640px they sit side by side in two columns, at 390px one above the other.
- **SC-12 [SITE-009 AC-3; SITE-016 AC-5]:** The text of `/shipping` is searched: "express", "Free over", "30.000", "50.000", "0 ₫", any amount followed by "₫", any city list and the cut-off "15:00" appear nowhere on it.
- **SC-13 [SITE-009 AC-4]:** The section headed "Delivery options" shows a table with headers Option, Arrives, Where, Cost and one row: Standard · 2–4 working days · the note · "Free" right-aligned.
- **SC-14 [SITE-009 AC-5]:** In that row "Free" is the only cost: no line under it, no Express row. The word is the same string as the bag's Delivery line (`pages.cart.free`): changing that string changes both.
- **SC-15 [SITE-009 AC-8]:** The shop's `standard_days` is changed to "3–5": the second figure reads `3–5`. The delivery record's `days` is changed to "3–5 working days": the row's Arrives cell reads it. Each shows its own stored value.
- **SC-16 [SITE-009 AC-6, AC-7, AC-9]:** At 390px the table scrolls sideways inside its own box and the page has no horizontal scroll; "Track an order" goes to `/track`.

### Bag, promises and payment words

- **SC-17 [SITE-016 AC-3; SITE-007 AC-8]:** A bag of one pair at 3.090.000 ₫: `/cart` shows Subtotal `3.090.000 ₫`, Delivery "Free", Total `3.090.000 ₫`.
- **SC-18 [SITE-016 AC-3, boundary; SITE-007 AC-9]:** A bag worth 1.500.000 ₫, one worth 1.999.999 ₫, one worth exactly 2.000.000 ₫ and one worth 4.590.000 ₫ (two pairs): each shows Delivery "Free", never `0 ₫`, and Total equal to Subtotal.
- **SC-19 [SITE-007 AC-10; SITE-016 item 5]:** Under "Check out" on `/cart` the line reads "You pay online with VNPay — card, domestic ATM card or QR — at the next step."; the page names neither cash on delivery, bank transfer nor MoMo.
- **SC-20 [SITE-004 AC-10; SITE-016 item 5]:** With standard days "2–4", the four promises on the home page and in the bag read "Free delivery" ("2–4 working days, anywhere in Vietnam."), "No exchanges" ("Check the size chart before you order."), "Pay online" ("Card, domestic ATM card or QR, through VNPay.") and "Genuine adidas"; the third cell's icon is a card, not a banknote; none mentions cash on delivery, a fee, a threshold or exchange days.

### Checkout

- **SC-21 [SITE-016 item 3; ORD-001 AC-3, AC-4]:** `/checkout` with a pair in the bag: step 02 shows exactly one row, "Free delivery", with "2–4 working days" and the record's note, "Free" at its right, already selected; the row has no radio input and no choice, and no Express row exists. The row keeps `data-testid="delivery-standard"`.
- **SC-22 [SITE-016 AC-3, boundary; ORD-001 AC-5, AC-7, AC-8]:** The summary "Your order": for 3.090.000 ₫ it reads Subtotal `3.090.000 ₫`, Delivery "Free", Total `3.090.000 ₫`; for 1.500.000 ₫ Total reads `1.500.000 ₫`; for two pairs of 3.090.000 ₫ and 1.500.000 ₫ Subtotal and Total both read `4.590.000 ₫`. Delivery is never `0 ₫`.
- **SC-23 [SITE-016 item 6; ORD-006 AC-1, AC-3]:** Step 03 lists "VNPay — ATM, Visa, Mastercard" and its note alone, already selected, with no choice; "Cash on delivery", "Bank transfer (VietQR)" and "MoMo wallet" are not listed; the line under the totals never reads "we confirm by text message" or "you pay the courier in cash".
- **SC-24 [SITE-016 item 3]:** A visitor places an order with the form filled: the order saved in this browser has delivery `standard`, no delivery fee, and total equal to the pairs total; `/orders/<id>?placed=1` opens with the heading "Thank you — your order is placed." and that same total.
- **SC-25 [ORD-001 AC-2; NFR Responsive]:** At 390px checkout is a single column with no horizontal page scroll.

### Order and track pages

- **SC-26 [SITE-016 AC-3; ORD-007 AC-22]:** The order page of a shipped sample order (DOI-24817) reads Delivery `Standard · 2–4 working days · Free` and Total `3.090.000 ₫`; DOI-24815 reads Total `5.890.000 ₫`, DOI-24810 `3.290.000 ₫`, DOI-24806 `3.090.000 ₫`.
- **SC-27 [SITE-016 AC-7; ORD-007 AC-13]:** The order page of an unpaid order (DOI-24816) reads `<way to pay> · awaiting payment`; the words "to pay on delivery" appear nowhere.
- **SC-28 [SITE-016 AC-7; ORD-020 item 2]:** On `/track` the "Confirmed" step opens onto "Paid online; the pair is set aside for you."

### Shoe page and contact page

- **SC-29 [SITE-006 AC-2; SITE-016 item 5]:** On any shoe page the "Payment options" row lists one entry: the card icon, "VNPay — ATM, Visa, Mastercard." and the note "Pay by card or banking app on VNPay's secure page."; cash on delivery, bank transfer and MoMo are not listed.
- **SC-30 [SITE-006 AC-3, AC-4]:** The "Delivery & return policy" row still reads "Free standard delivery in 2–4 working days, sent the same day if you order before 15:00. Exchanges are not accepted." with links "Delivery" (`/shipping`) and "Return policy" (`/returns`).
- **SC-31 [SITE-012 AC-13; SITE-016 item 4]:** On `/contact` the four "Already answered" links read "Find your size", "Free delivery and arrival times", "Return policy", "Track an order" and go to `/size-chart`, `/shipping`, `/returns`, `/track`.

### Whole site

- **SC-32 [SITE-016 AC-1, item 3]:** `code/frontend/src` is searched: there is no `free_delivery_from`, `express_days` or `express_cities`; no whole-word `express` (the licence text in `components/ui/LICENSE.md` aside); no delivery fee of `30000` or `50000` as a whole number (`3000000` price bands are unrelated); no variable, function or branch that computes a delivery fee or a free-delivery threshold in `Cart`, `Checkout`, `Order`, `Shipping`, `Product`, `PromiseRow`, `shop.ts` or `mock/orders.ts`; and `ShopTerms` declares none of the removed fields.
- **SC-33 [SITE-016 AC-7, item 6]:** The strings "cash on delivery", "pay on delivery", "pay the courier" and "paid cash" appear in no value of `content.json`, `faqs.json` or `reviews.json`, and no component offers a cash-on-delivery row. The only places the `cod` id remains are the `cod` record of `payments.json`, the `cod` option of `schema.json`, and the five sample orders that name it (section 4).
- **SC-34 [SITE-016 item 7]:** The diff of `content.json` changes only the keys listed in 3.3; no other page's words, colours or spacing differ; `npm run lint`, `npm run build` and `npm test` pass.

## 7. Dependencies

- **Lands first:** nothing. "The site as designed" is done; the returns story (PR #3) already removed the invented exchange and refund terms from `shop.json`, `schema.json`, `shop.ts` and the Returns page.
- **Must land before:** "Site data from the database" (it seeds `delivery`, `shop`, `orders`, `payments`, `faqs` and `reviews` from the files as this story leaves them — its DATA-001 item 3 and AC-8, AC-9 assume them) and "Checkout and placing an order" (it builds on the one-row delivery step and the VNPay-only payment step).
- **External accounts and secrets:** none. VNPay's keys belong to the payments story.
- **Files outside the usual story ownership:** this story edits `src/content.json` and `src/data/schema.json` (both normally frozen) and six pages and components that other stories also own. Nothing else is building against them yet: the two stories that touch them are queued behind this one.

## 8. Decisions taken in this plan

1. **The `vnpay` note, `f-orders-number`, `f-delivery-time` and review r-04 change** although the plan item says "payments.json rows stay as they are": the SRS (SITE-016 item 5, AC-8) names them because each states something the shop does not do (an order "marked paid at once", a text message, express). The `cod`, `bank` and `momo` rows are untouched, as the plan item says.
2. **Review r-07 is added to the list.** "Paid cash at the door, checked the box first." is shown on a live shoe page and tells buyers cash on delivery exists. The SRS named r-04 only; `docs/site/SRS.md` is amended in this PR so story and SRS agree.
3. **Checkout step 03 lists VNPay alone now**, not later. The plan item leaves "what is offered" to the Checkout story, but the acceptance ("no offer of cash on delivery") cannot hold while the page lists the `cod` record and starts with it selected. The Checkout story later replaces the source of this one row with the shop's own answer and adds the keys-missing state.
4. **The delivery row and the VNPay row are plain selected rows, not a radio group.** A radio with one option is a choice that is not one; the design's selected-dot look stays.
5. **`promises.cod_*` keys are renamed `pay_*`** so no key in the site's content still says `cod`; the Studio shows the new names.
6. **The promise icon changes from a banknote to a card** with the words: a banknote beside "Pay online" would still read as cash.

## 9. How a reviewer checks it

- Search `code/frontend/src` for `free_delivery_from`, `express_days`, `express_cities`, `\bexpress\b` (ignore `components/ui/LICENSE.md`), `\b(30000|50000)\b`, `cash on delivery`, `pay on delivery`, `\bcod\b` (expected: only the places SC-33 lists).
- Put a 1.500.000 ₫ pair, then a 3.090.000 ₫ pair, in the bag; read `/cart` and `/checkout`; place one order and read `/orders/<id>?placed=1`.
- Read `/shipping`, `/track`, `/contact`, a shoe page and the home page's promise row at 1280px and 390px.
