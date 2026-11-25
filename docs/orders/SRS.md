# SRS — Orders: checkout and tracking

Module: `orders`
Design: [View the approved design](http://localhost:9401)
Design system: `design/design-system.md`

> One file per module. This one covers placing an order, keeping stock true when pairs sell, the order page, and
> looking an order up again. Paying online is in `docs/payments/SRS.md`; the dealer moving an order along is in
> `docs/admin/SRS.md`.

## 1. Purpose

A customer picks a pair, enters where to send it, chooses how to pay and places the order; later they come back
and see where it is. This module turns the browser's bag into an order the shop can pack and send, keeps the
number of pairs on the shelf honest as pairs sell, and lets only the customer (and the dealer) read an order. Without it the
shop has no sales, the "In stock" figures drift from the shelf, and every customer's phone and address would sit in a public list.

## 2. Actors

| Actor | Who they are | What they may do in this module |
|---|---|---|
| Customer | A visitor who orders, as a guest or signed in (`accounts` module) | Place an order; read an order they hold the order token for, or one linked to their own account; look an order up by number and phone |
| Stranger | Anyone else, including a customer without the token | Learn nothing about any order |
| Shop team | The owner (admin) or staff, signed in (`admin` module) | Read every order and its private details; list orders |
| Payment provider | A gateway such as VNPay | Tells the shop a payment happened (`payments` module) |

## 3. Scope

**In scope** — the plan items, by title:

- Checkout and placing an order
- Track an order by number and phone

**Out of scope**

- The bag itself (drawer, bag page, browser storage) — belongs to the `site` module (SITE-007); the bag stays in the browser.
- Charging a card, VNPay's notice, the Payment result page, cancelling unpaid online orders after 30 minutes — belong to the `payments` module.
- The dealer's moving an order to confirmed / packed / shipped / delivered / cancelled, setting carrier, tracking number and expected date — belongs to the `admin` module; this module only shows the result. Nobody marks an order paid by hand: only VNPay's notice does.
- Bank transfer (VietQR) and MoMo as ways to pay — belong to the later `payments` story; their rows stay in the records and are not offered at checkout until then.
- Customer accounts and a signed-in order history — belong to the `accounts` module; this module keeps guest checkout, the order token, "Ordered on this phone" and the lookup by number and phone working unchanged for a guest.
- Email or text confirmations — not in the brief or the plan; the design's words say a text message is sent, which this module does not build (open question 4).

## 4. Functional requirements

### 4.1 Checkout and placing an order

**Requirement ORD-001 — The checkout page**

*As a* customer, *I want to* see what I am buying, choose delivery and payment and see the total before I place the order, *so that* I know what I will pay.

Behaviour:

1. `/checkout` with pairs in the bag shows the heading "Checkout"; from 1024px the three steps "01 Contact and address", "02 Delivery", "03 Payment"; the line "Your details travel over a secure connection."; a picture; and the form.
2. Step 01 has the fields "Full name", "Phone", "Street address, ward, district", "City or province", and "Note for the courier (optional)" (two lines). Each has its label. The phone field opens a phone keypad on a phone. Browsers may fill name, phone and address from their saved details.
3. Step 02 shows one delivery row, already selected and not a choice: icon, "Free delivery", the standard option's note and arrival time (`<arrives in>`) and, at the right, "Free". There is no other delivery option, no fee and no threshold.
4. Step 03 shows one way to pay as a row, already selected: icon, "VNPay — ATM, Visa, Mastercard" and its note (ORD-006). Cash on delivery, bank transfer and MoMo are not offered. When VNPay's keys are missing the row stays, "Online payment is not available right now. Please try again later." shows under it in alert style, and "Place order" is disabled.
5. The summary "Your order" lists each pair: picture, name in capitals, `<variant> · Qty <n>`, line total; then Subtotal, Delivery ("Free", never `0 ₫`) and Total, which equals Subtotal. The prices are the shop's current prices, read when the page opens, not the price saved in the bag when the pair was added.
6. Under the totals an information line says what happens next: "After you press Place order you pay on the <method name> page; we pack your order after the payment is confirmed."
7. The "Place order" button has a lock icon; while an order is being placed it reads "Placing your order…" and is disabled. Under it: "By placing the order you accept the delivery information and that exchanges are not accepted."
8. `/checkout` with an empty bag shows the heading "Nothing to check out yet.", the line "Pick a pair — the size chart is one tap from every shoe.", a "Keep shopping" pill, and "New this month" with up to four live new shoes.
9. The total shown here is the total stored when the order is placed (ORD-003).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | One pair in the bag; 1280px | `/checkout` loads | The heading "Checkout", the three steps 01–03, "Your details travel over a secure connection.", and the five fields with their labels are shown |
| AC-2 | The same at 390px | `/checkout` loads | The three-step line is not shown; the form is a single column |
| AC-3 | The delivery record `standard` | The page loads | Exactly one delivery row, "Free delivery", is shown, selected and not a choice, with the standard option's arrival time and "Free" at the right |
| AC-4 | Pairs total 3.090.000 ₫ | The page loads | The delivery row shows "Free" and no row named Express exists |
| AC-5 | Pairs total 1.500.000 ₫ (under 2.000.000 ₫) | The page loads | The delivery row shows "Free"; Delivery reads "Free" and Total reads `1.500.000 ₫` |
| AC-6 | The bag line was added at 3.000.000 ₫ and the shoe now costs 3.090.000 ₫ | The page loads | The summary line shows `3.090.000 ₫` |
| AC-7 | One pair at 3.090.000 ₫, standard free | The page loads | Subtotal `3.090.000 ₫`, Delivery "Free", Total `3.090.000 ₫` |
| AC-8 | Two pairs at 3.090.000 ₫ and 1.500.000 ₫ | The page loads | Subtotal and Total both read `4.590.000 ₫` and Delivery reads "Free" |
| AC-9 | The page is loaded | The visitor reads step 03 | One row, "VNPay — ATM, Visa, Mastercard", selected; no row for cash on delivery, bank transfer or MoMo |
| AC-10 | VNPay is the way to pay | The visitor reads under the totals | "After you press Place order you pay on the VNPay — ATM, Visa, Mastercard page; we pack your order after the payment is confirmed." |
| AC-11 | The page is loaded | The visitor reads the foot of the summary | The "Place order" button and the sentence "By placing the order you accept the delivery information and that exchanges are not accepted." |
| AC-12 | An empty bag | `/checkout` loads | "Nothing to check out yet." with the line "Pick a pair — the size chart is one tap from every shoe.", the "Keep shopping" pill and the "New this month" row |
| AC-13 | A completed checkout | The order's stored total is read | It equals the Total shown on the page when "Place order" was pressed |
| AC-14 | VNPay's keys are not set (after the VNPay story) | `/checkout` loads with a pair in the bag | The VNPay row is shown with "Online payment is not available right now. Please try again later." in alert style under it, and "Place order" is disabled |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Absent data | The bag is empty | AC-12 |
| Boundary | Pairs total under, at or over 2.000.000 ₫ | The delivery row shows "Free" and Total equals the pairs total in every case |
| Not applicable | Roles | Not applicable: checkout is open to any visitor |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Bag lines: shoe id, size, quantity | read | yes | From the browser |
| Shoe: current price | read | yes | From the shop |
| Delivery option "standard": name, days, note | read | yes | Its fee is always 0, drawn as "Free" |
| Ways to pay the shop can complete | read | yes | VNPay only (ORD-006) |

**Requirement ORD-002 — Details are checked before and after the order is sent**

*As a* customer, *I want to* be told what is missing, *so that* the courier can reach me.

Behaviour:

1. Pressing "Place order" with a name, street address or city that is empty or only spaces shows "Fill in your name, street address and city." and sends nothing.
2. A phone with fewer than 9 digits (spaces, `+`, `-` and brackets do not count) shows "Check the phone number — we text you when the pair is on its way." and sends nothing. Both messages can show together, each on its own line with an alert icon.
3. The shop checks the same rules again and refuses an order that breaks them, so a request that skips the page is refused too. It also refuses a name over 100 characters, a phone over 20 characters, a street address over 200, a city over 100 and a note over 500; the message names the limit.
4. Fields keep what was typed after a failed attempt.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Name empty, other fields filled | The visitor presses "Place order" | "Fill in your name, street address and city." is shown and no order is created |
| AC-2 | City is `   ` (spaces) | The visitor presses "Place order" | The same message |
| AC-3 | Phone `0912 34` | The visitor presses "Place order" | "Check the phone number — we text you when the pair is on its way." is shown and no order is created |
| AC-4 | Phone `+84 912 345 678` | The visitor presses "Place order" | The phone is accepted (11 digits) |
| AC-5 | Phone has 9 digits `091234567` | The visitor presses "Place order" | The phone is accepted |
| AC-6 | Phone has 8 digits | The visitor presses "Place order" | The phone message is shown |
| AC-7 | Name empty and phone `12` | The visitor presses "Place order" | Both messages are shown, one per line |
| AC-8 | A request to the shop with an empty address | It is sent without the page | 422 naming `address`; no order is created |
| AC-9 | A request with a note of 501 characters | It is sent | 422 naming `note` and the limit 500 |
| AC-10 | A request with a note of exactly 500 characters | It is sent | The order is created |
| AC-11 | Validation failed | The visitor reads the form | Every typed value is still in its field |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | Any rule above is broken | The message above; nothing is created, no pair leaves the shelf |
| Boundary | Name 100 characters; 101 | Accepted; refused naming the limit |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Name | text | yes | At most 100 characters |
| Phone | text | yes | At least 9 digits; at most 20 characters |
| Street address, ward, district | text | yes | At most 200 characters |
| City or province | text | yes | At most 100 characters |
| Note for the courier | text | no | At most 500 characters |

**Requirement ORD-003 — Placing the order: the shop computes everything**

*As a* customer, *I want* my order to be saved at the shop's own prices, *so that* the total is right and nobody can order at a made-up price.

Behaviour:

1. Placing an order sends the pairs (shoe, EU size, quantity), the delivery option, the payment method, name, phone, address, city and note. It sends no price, fee or total.
2. The shop re-reads each shoe's price from the database and computes the total itself: the sum of price × quantity. Delivery is free: the stored delivery fee is always 0, no threshold is read, and the total is never more than that sum.
3. The order is saved with the number `DOI-<n>` and the id `doi-<n>`, where n is one more than the highest number held, never repeating; the status "placed"; not paid; and an expected date from the delivery option: the day the order is placed, in Vietnam time, plus the longest working days of the `standard` option (4 for "2–4"), skipping Saturdays and Sundays; an order placed on or after the shop's same-day cut-off (15:00) or on a weekend starts counting from the next working day.
4. Each pair of the order keeps the unit price it was sold at, so a later price change does not change the order.
5. An order can hold several pairs; the orders already on record, which hold one shoe each, keep working and show as orders of one pair.
6. The answer holds the order (without phone, email, address or note), an order token only the answer carries, and a payment address — empty only while VNPay is not yet implemented or cannot give one (ORD-006).
7. The order token cannot be guessed (at least 128 bits of randomness) and opens only this order.
8. The delivery is always `standard`. A request that names any other delivery option is refused with 422 naming `delivery`; a delivery fee sent in the request is ignored. The stored fee is 0 either way.
9. An order placed by a signed-in customer is linked to that customer's account; an order placed as a guest is linked to none (ACC-005).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The highest stored number is DOI-24817 | An order is placed | It is `DOI-24818`, id `doi-24818`, status "placed", paid false |
| AC-2 | Two orders placed at the same moment | Both succeed | Their numbers differ |
| AC-3 | A shoe priced 3.090.000 ₫, quantity 2, standard delivery | An order is placed | The stored total is `6.180.000 ₫` and the stored delivery fee is 0 |
| AC-4 | A shoe priced 1.500.000 ₫, quantity 1, standard | An order is placed | The stored total is `1.500.000 ₫` (under 2.000.000 ₫, still free) and the stored delivery fee is 0 |
| AC-5 | A request naming the delivery option `express` | An order is placed | 422 naming `delivery`; no order is created |
| AC-6 | The request lists a price of 1 ₫ for the shoe | It is sent | The stored total ignores it and equals the shop's price |
| AC-7 | An order placed on Monday 10:00 Vietnam time, standard | It is saved | Its expected date is Friday of that week |
| AC-8 | An order placed on Monday 16:00 Vietnam time, standard | It is saved | Its expected date is the next Monday (counting begins on Tuesday) |
| AC-9 | A request carrying a delivery fee of 50.000 ₫ | An order is placed | The fee is ignored: the stored delivery fee is 0 and the total is the sum of the pairs |
| AC-10 | An order placed on a Saturday, standard | It is saved | Counting begins on the following Monday; the expected date is the Friday of that week |
| AC-11 | An order with two shoes | It is read back | Two pairs with their sizes, quantities and unit prices |
| AC-12 | The seeded order `doi-24817` | It is read by the dealer | It shows as one pair of `samba-og-w-crystal-white`, EU 39 1/3 |
| AC-13 | The shoe's price changes after the order | The order is read | The pair's unit price and the total are as sold |
| AC-14 | An order was placed | The answer is read | It carries an order token of at least 22 URL-safe characters and no `phone`, `email`, `address` or `note` |
| AC-15 | An unknown delivery option id | The order is placed | 422 naming the delivery option; no order is created |
| AC-16 | A payment method that is not offered | The order is placed | 422 naming the payment method; no order is created |
| AC-17 | A line with quantity 0 or a size the shoe does not list | The order is placed | 422 naming that line; no order is created |
| AC-18 | A basket under 2.000.000 ₫ and a basket over it | Each is placed | Each stored total equals exactly the sum of its pairs' price × quantity and each stored delivery fee is 0 |
| AC-19 | A signed-in customer and a guest | Each places an order | The first order is linked to the customer's account and the second to none |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | Empty list of pairs | 422; no order |
| Conflict | The shoe's price changes between the page loading and the order | The order is placed at the shop's current price; the Order page shows that total |
| Upstream failure | The database cannot be reached | 500 with a generic body; no pair leaves the shelf and no order exists |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Order: number, id, placed time, status, total, payment, paid, delivery, expected date, customer, customer account (when signed in), phone, address, city, note | write | yes | Phone, address and note private |
| Order pair: shoe, EU size, quantity, unit price | write | yes | One or more per order |
| Order token | write | yes | Unguessable |

**Requirement ORD-004 — Taking pairs off the shelf as the order is saved**

*As a* customer, *I want to* be told if the pair I chose was sold a moment ago, *so that* I am never charged for a pair the shop does not have.

Behaviour:

1. Placing an order checks that each shoe is live, that the size is one the shoe lists, and that the size has at least the quantity asked on the shelf; it takes those pairs off the shelf in the same step that saves the order. Either the whole order is saved and the pairs are taken, or nothing is.
2. When two customers ask for the last pair of a size at the same moment, one order is saved and the other is refused; stock never goes below zero.
3. A refused order names every pair that failed and why: the size is sold out ("sold out in your size"), fewer pairs remain than asked ("only <n> left in your size"), or the shoe is no longer in the shop. Nothing is taken from the shelf for the pairs that were fine.
4. On a refusal the page shows, for each named pair, `<shoe name> · EU <size> — sold out in your size.` (or the other two sentences), keeps the bag as it is and keeps the typed details.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Size `38` of a shoe has 2 pairs | An order for 1 pair of `38` is placed | The order is saved and 1 pair is left in `38` |
| AC-2 | Size `38` has 1 pair | Two orders for it are sent at the same time | Exactly one succeeds (201); the other is refused (409) naming that pair; 0 pairs remain, never −1 |
| AC-3 | The same | The refused customer reads the page | `<shoe name> · EU 38 — sold out in your size.` is shown and the bag still holds the pair |
| AC-4 | Size `38` has 2 pairs | An order for 3 pairs of `38` is placed | Refused naming that pair with "only 2 left in your size"; 2 pairs remain |
| AC-5 | A bag of two pairs, the second sold out | The order is placed | Refused naming only the second pair; the first pair's stock is unchanged |
| AC-6 | A shoe was set to draft after it went into the bag | The order is placed | Refused naming the pair as no longer in the shop |
| AC-7 | An order is refused | The shop's stock is read | It equals what it was before the attempt |
| AC-8 | An order is refused | The visitor reads the checkout form | The typed details are still in the fields |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Conflict | Two orders race for the last pair | One saved, one refused (AC-2) |
| Boundary | Quantity equals the pairs left | Accepted; the size is then sold out |
| Not found | A shoe id that does not exist | Refused as no longer in the shop |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Per-size pairs on the shelf | read / write | yes | Reduced by the order, never below 0 |

**Requirement ORD-005 — The stock shown follows the real count**

*As a* customer, *I want* the sizes shown as in stock, last pairs or sold out to be true, *so that* I can trust what the shoe page tells me.

Behaviour:

1. The shop keeps a count of pairs for each size of each shoe. A size with 0 pairs is in the shoe's sold-out list; a size with 1 or 2 pairs is in its last-pairs list; a size with 3 or more is in neither. The shoe's "pairs in stock" is the sum over its sizes. The counts are the ones staff type in the shoe form (ADMIN-015); there is no second copy.
2. These lists and the total change the moment an order is saved, so the next time a shoe page loads it shows the new state: a size reaching zero is struck through; one or two left carries the last-pairs dot.
3. Pairs come back to the shelf, and the lists and total with them, when an order is cancelled (`admin` and `payments` modules).
4. The shoes that came with the first design are placeholders: the shop team may set them to draft, or delete them when no order refers to them (ADMIN-014), and replace their stock with true counts (ADMIN-015). How a placeholder's per-size counts are first filled, so that it can be ordered in the meantime, is TL's technical design; the lists and the total it shows are the only figures it carries.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Size `38` of a shoe has 3 pairs and is in neither list | An order for 1 pair of `38` is saved | `38` is in the shoe's last-pairs list |
| AC-2 | Size `38` has 2 pairs | An order for 1 is saved | `38` stays in the last-pairs list |
| AC-3 | Size `38` has 1 pair | An order for 1 is saved | `38` is in the sold-out list and no longer in the last-pairs list |
| AC-4 | The same | The shoe page is loaded afterwards | The `38` pill is struck through and cannot be chosen |
| AC-5 | A shoe's counts are 3, 2, 0 and 5 | Its stock is read | "Pairs in stock" is 10 |
| AC-6 | The shoe from AC-5 | An order for 2 pairs of the size with 5 is saved | Pairs in stock is 8 |
| AC-7 | A placeholder shoe from the first design | The shop team sets it to draft | It leaves `/shop`, search and the home page; it can be deleted only when no order refers to it |
| AC-8 | An order is refused (ORD-004) | The lists are read | They are unchanged |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Concurrency | Many orders for one shoe at once | Counts, lists and total agree after all of them (no lost update) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Per-size pairs | read / write | yes | Whole number, at least 0 |
| Shoe: last-pairs sizes, sold-out sizes, pairs in stock | write | yes | Derived from the per-size pairs |

**Requirement ORD-006 — Ways to pay: only those the shop can complete**

*As a* customer, *I want to* be offered only ways to pay that work, *so that* I never choose one the shop cannot take.

Behaviour:

1. Payments go through one interface that each way to pay plugs into. VNPay is the only way to pay at launch — the owner chose "Online payment" — and there is no cash on delivery. The interface ships with the Checkout story; VNPay plugs into it with the `payments` module.
2. Step 03 of checkout shows VNPay as its single way to pay, already selected. Cash on delivery, bank transfer (VietQR) and MoMo are not offered: their rows stay in the records, and nothing but VNPay appears at checkout or in the shoe page's "Payment options" until the later story adds bank transfer and MoMo (PAY-022).
3. Placing an order with a way to pay that is not offered is refused.
4. The order's answer carries a payment address: the address of VNPay's payment page. When it is not empty, the page sends the browser there right after the order is saved. Until the VNPay story lands the address is always empty: the order is saved unpaid, the browser goes to `/orders/<id>?placed=1`, the order page says "awaiting payment" and "Pay now" answers 409 "online payment is not available yet".
5. If VNPay cannot give an address at that moment, the order is still saved, the answer's payment address is empty, and the customer lands on the order page where "Pay now" is offered (ORD-007).
6. Every order starts unpaid, awaiting payment, and only VNPay's notice marks it paid (PAY-002). Once the VNPay story has landed and VNPay's keys are missing, the shop cannot complete any order: checkout shows "Online payment is not available right now. Please try again later." in alert style under the VNPay row and disables "Place order"; a request to place an order answers 503, creates no order and takes no pair off the shelf; the bag and the typed details are kept.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The records hold cash on delivery, bank transfer, VNPay and MoMo | `/checkout` loads | Step 03 lists "VNPay — ATM, Visa, Mastercard" alone, selected |
| AC-2 | VNPay is implemented and its keys are set | `/checkout` loads | Step 03 lists VNPay alone, selected, and "Place order" is enabled |
| AC-3 | The same records | `/checkout` loads before the VietQR and MoMo story | "Cash on delivery", "Bank transfer (VietQR)" and "MoMo wallet" are not listed |
| AC-4 | The same | A shoe page's "Payment options" is opened | Only VNPay is listed (SITE-006) |
| AC-5 | An order is placed with `bank`, and another with `cod` | The shop receives each | 422 naming the payment method; no order is created |
| AC-6 | The VNPay story has not landed | VNPay is chosen and the order is placed | The answer's payment address is empty, the order is unpaid, the browser goes to `/orders/<id>?placed=1` and the order page says "awaiting payment" |
| AC-7 | A way to pay whose provider returns the address `https://pay.example.test/x` | The order is placed | The answer carries that address and the browser is sent to it |
| AC-8 | The provider fails to return an address | The order is placed | The order exists, unpaid; the answer's payment address is empty; the browser goes to `/orders/<id>?placed=1` where "Pay now" is shown |
| AC-9 | Any order just placed | It is read | `paid` is false |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | A way to pay that is not offered | 422 (AC-5) |
| Upstream failure | The provider is unreachable when placing | The order is saved without an address (AC-8) |
| Conflict | Not applicable: nothing is paid by this module | — |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Ways to pay: id, name, note, online, icon | read | yes | |
| Order: payment (way to pay id), paid | write | yes | paid starts false |
| Payment address | write | no | Empty for cash on delivery |

**Requirement ORD-007 — The order page**

*As a* customer, *I want to* see my order — where it is, what is in it, where it goes — and pay for it if it is still unpaid, *so that* I always know where I stand.

Behaviour:

1. `/orders/<id>` shows, for an order the visitor holds the token for, the line `Order <number> · placed <date, time>`; the answer as the page heading; a line under it; the five steps; the first pair's picture (a link to the shoe); the pairs; the total; and the delivery and payment details.
2. The heading is: "We have your order." (placed), "Your order is confirmed.", "Your order is packed.", "Your order is on its way." (shipped), "Your order was delivered.", "This order was cancelled." When the address carries `placed`, the heading is "Thank you — your order is placed." instead of the status heading.
3. The line under the heading is "Expected by <date>." while the order is not delivered or cancelled and has an expected date; for a delivered order "Delivered. If the size is not right, the exchange is free the first time."; nothing for a cancelled order.
4. The five steps are Placed, Confirmed, Packed, On its way, Delivered, as one line from 640px and as a list below it. Steps up to and including the current one are filled and the current one is ringed. A cancelled order shows, in place of the steps, "This order was cancelled. Nothing was charged; money paid online is back within three working days." with an alert icon.
5. "In this order" lists each pair: name (a link to the shoe), `<variant> · Qty <n>`, line price; then "Total".
6. "Delivery and payment" lists: Delivering to (`<customer>, <city>`), Delivery (`<name> · <arrives in>`), Payment (`<way to pay> · paid` when paid; `<way to pay> · to pay on delivery` for cash on delivery unpaid; `<way to pay> · awaiting payment` for an online way to pay unpaid), Carrier and Tracking number. A row with nothing in it is not drawn.
7. Two closed rows: "Need to change something?" (the sentence "Send us the order number and what to change; the address or the size can change until the order is packed." and a "Message us" link to `/contact`) and "Wrong size?" ("Exchange it within the exchange window; the first exchange is free." and a "How exchanges work" link to `/returns`). A "Keep shopping" pill goes to `/shop`.
8. For an online order that is unpaid and not cancelled the page shows a "Pay now" button. Pressing it asks the shop for a fresh payment address and sends the browser there. It is not shown for cash on delivery, a paid order or a cancelled order. While it waits the button is disabled.
9. The shop refuses "Pay now" with 409 for cash on delivery, for an order already paid and for a cancelled order, and with 503 when the way to pay's provider is not available.
10. An order the visitor holds no token for, or one that does not exist, shows the heading "We can't find that order.", the line "Check the number in your confirmation message, then look it up with the phone you ordered with." and a "Track an order" pill to `/track`. That view is shown only after the shop has answered, never while the order is still being fetched.
11. A refresh of the page keeps showing the order: the token is kept in this browser.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The customer has just placed order `doi-24818` | `/orders/doi-24818?placed=1` loads | The heading reads "Thank you — your order is placed." and the line reads `Order DOI-24818 · placed <date, time>` |
| AC-2 | The same | The page is refreshed | The order is still shown |
| AC-3 | The same, in another browser with no token | `/orders/doi-24818` loads | "We can't find that order." with the line and the "Track an order" pill |
| AC-4 | An order of status "shipped", expected `2026-11-26` | The page loads with the token | The heading reads "Your order is on its way." and the line "Expected by 26/11/2026." |
| AC-5 | An order of status "delivered" | The page loads | The heading reads "Your order was delivered." and the line "Delivered. If the size is not right, the exchange is free the first time." |
| AC-6 | An order of status "cancelled" | The page loads | The heading reads "This order was cancelled.", the steps are replaced by the cancelled sentence, and no "Expected by" line is shown |
| AC-7 | An order of status "packed" | The page loads | Placed, Confirmed, Packed are filled, Packed is ringed, On its way and Delivered are not filled |
| AC-8 | A 390px window | The page loads | The five steps are a vertical list |
| AC-9 | Carrier "GHN", tracking "GHN-PT27XM" set | The page loads | "Carrier" shows `GHN` and "Tracking number" shows `GHN-PT27XM` |
| AC-10 | Carrier and tracking empty | The page loads | Neither row is drawn |
| AC-11 | A paid cash-on-delivery order | The page loads | "Payment" reads `Cash on delivery · paid` |
| AC-12 | An unpaid cash-on-delivery order | The page loads | "Payment" reads `Cash on delivery · to pay on delivery`; no "Pay now" |
| AC-13 | An unpaid VNPay order | The page loads | "Payment" reads `VNPay — ATM, Visa, Mastercard · awaiting payment`, and "Pay now" is shown |
| AC-14 | The same | The visitor presses "Pay now" | The shop is asked for a fresh payment address and the browser goes to it; while waiting the button is disabled |
| AC-15 | A paid VNPay order | The page loads | "Pay now" is not shown |
| AC-16 | A cancelled VNPay order | The page loads | "Pay now" is not shown |
| AC-17 | `POST /v1/orders/<id>/payment` for a cash-on-delivery order | It is called with the token | 409 |
| AC-18 | The same for a paid order | It is called with the token | 409 |
| AC-19 | The same for a cancelled order | It is called with the token | 409 |
| AC-20 | The same with no token | It is called | 404 |
| AC-21 | An order of two pairs | The page loads | Both pairs are listed under "In this order", each with `<variant> · Qty <n>` and its line price, and the total is the stored total |
| AC-22 | Delivery "Standard · 2–4 working days", customer "Lan Nguyen", city "Da Nang" | The page loads | "Delivering to" reads `Lan Nguyen, Da Nang` and "Delivery" reads `Standard · 2–4 working days` |
| AC-23 | The page loads | The visitor opens "Need to change something?" and "Wrong size?" | Their sentences and the links "Message us" (`/contact`) and "How exchanges work" (`/returns`) are shown |
| AC-24 | The order answer is slow (2 seconds) | The page opens with a valid token | "We can't find that order." is not shown before the answer arrives |
| AC-25 | The page loads | The visitor reads it | The order's address, phone, email and note appear nowhere on the page |
| AC-26 | The first pair has a picture | The page loads | It is shown at the right and links to the shoe's page |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | No order, wrong id, or no token | AC-3: the same view for all three, so a stranger cannot tell which |
| Permission | Token of another order | Treated as no token: the not-found view |
| Upstream failure | "Pay now" cannot get an address (503 or network) | The page stays; a message "We could not open the payment page. Please try again." shows under the button; the button is enabled again |
| Upstream failure | The shop is unreachable when the page opens | Nothing is drawn in place of the order (never the not-found view) |
| Boundary | An order without an expected date | No "Expected by" line |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Order: number, placed time, status, total, payment, paid, delivery, carrier, tracking, expected date, customer, city | read | yes | Never phone, email, address, note |
| Order pairs: shoe, size, quantity, unit price | read | yes | |
| Shoe: name, first picture | read | yes | |
| Ways to pay, delivery options | read | yes | Names for the details |

**Requirement ORD-008 — Only the customer and the dealer can read an order**

*As a* customer, *I want* nobody else to read my order, *so that* my phone and address stay private.

Behaviour:

1. Orders are not a public collection. The list of orders answers 401 with no session and 403 for a signed-in caller who is not the dealer; only the dealer lists orders.
2. One order is read with its order token, or with the dealer's session. Without either the answer is 404, the same as for an order that does not exist.
3. The order's phone, email, address and note are never returned to a visitor, token or not. The dealer's order page shows them (`admin` module).
4. The token is sent only to the browser that placed the order or looked it up (ORD-021). It is not in any address or message the shop sends.
5. Until the dealer's sign-in story exists, nobody holds a dealer session, so the list of orders answers 401 to everyone.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | No session | `GET /v1/orders` | 401 |
| AC-2 | A session that is not the dealer's | `GET /v1/orders` | 403 |
| AC-3 | The dealer's session | `GET /v1/orders` | 200 with the paged list |
| AC-4 | An order exists | `GET /v1/orders/<id>` with no token and no session | 404 |
| AC-5 | The same, with the order's own token | `GET /v1/orders/<id>` | 200 with the order, without `phone`, `email`, `address`, `note` |
| AC-6 | The same, with another order's token | `GET /v1/orders/<id>` | 404 |
| AC-7 | The same, with a dealer session | `GET /v1/orders/<id>` | 200 with the order including its phone, email, address and note |
| AC-8 | A missing id | `GET /v1/orders/nope` with no token | 404 with the same body as AC-4 |
| AC-9 | Orders after this module is deployed | `GET /v1/orders?phone=…` with no session | 401 |
| AC-10 | An order was placed | The order's own page and the network log are read | The token appears only in the placing answer and in this browser's storage |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Permission | No session / wrong role | 401 / 403 (AC-1, AC-2) |
| Permission | No token / wrong token | 404 (AC-4, AC-6) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Order token | read | yes | Held in the placing browser |

**Requirement ORD-009 — After "Place order": where the customer lands**

*As a* customer, *I want to* land on my order once it is placed, *so that* I see it was taken.

Behaviour:

1. When the order is saved with no payment address, the bag is emptied, the order's id, number, time and token are saved in this browser, and the browser goes to `/orders/<id>?placed=1`.
2. When the order is saved with a payment address, the bag is emptied, the same is saved, and the browser goes to that address.
3. When the order is not saved, the bag and the typed details are kept.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A bag of one pair; cash on delivery | "Place order" succeeds | The bag is empty and the address is `/orders/<id>?placed=1` |
| AC-2 | The same | The browser's storage is read | It holds the order's id, number, time and token |
| AC-3 | An online way to pay with an address | "Place order" succeeds | The bag is empty and the browser is at the payment address |
| AC-4 | The order is refused (sold out) | The visitor reads the bag | The bag still holds every pair |
| AC-5 | The flow "Choose a pair, pay and see the order placed" | A pair is chosen, the form is filled with `Lan Nguyen`, `0912345678`, `12 Example Street, Ward 1`, `Da Nang`, cash on delivery is chosen and "Place order" pressed | The order heading contains "Thank you" |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Storage refused | The browser refuses local storage | The order page shows once for this page load; "Ordered on this phone" does not list it (the customer finds it with Track) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Browser store of orders: id, number, placed time, token | write | yes | Kept in this browser |

**Requirement ORD-010 — Failed, offline and repeated submissions**

*As a* customer, *I want to* know when the order did not go through, and never be charged twice, *so that* I can try again with confidence.

Behaviour:

1. When the shop cannot be reached or answers an error, the page shows, above the button, "We could not place your order. Nothing was charged. Please try again." The bag and typed details stay, and the button is enabled.
2. When the browser is offline, the page shows "You seem to be offline. Check your connection and try again."
3. Each attempt carries a key made once for that bag and those details. Sending the same key again returns the order the first send made, takes no more pairs off the shelf, and sends the same answer. A new key is made when the bag changes.
4. The button is disabled while an order is being placed, so a second press does nothing.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The shop answers 500 | "Place order" is pressed | "We could not place your order. Nothing was charged. Please try again." is shown, the bag and details are kept, the button is enabled |
| AC-2 | The browser is offline | "Place order" is pressed | "You seem to be offline. Check your connection and try again." is shown |
| AC-3 | The first send saved the order but its answer was lost | The customer presses "Place order" again | One order exists, the pair was taken once, and the page lands on that order |
| AC-4 | A request with a key already used | It is sent twice | The two answers carry the same order number and the shelf lost the pairs once |
| AC-5 | The bag changes after a failure | "Place order" is pressed | A new key is used and a new order can be made |
| AC-6 | An order is being placed | The visitor presses the button again | Nothing more is sent |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Upstream failure | The shop is down or answers 5xx | AC-1 |
| Conflict | The same key from two tabs at once | One order is made; the second tab receives it |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Attempt key | write | yes | Stored with the order it made |

### 4.2 Track an order by number and phone

**Requirement ORD-020 — The track page**

*As a* customer, *I want to* look up my order with its number and my phone, *so that* I can see where it is.

Behaviour:

1. `/track` shows the heading "Track an order", the lead "Type the order number from your confirmation message and the phone number you ordered with.", a picture, and a form: "Order number" (placeholder `DOI-24817`), "Phone", and a "Find my order" button with an arrow.
2. "What each step means" shows five closed rows — Placed, Confirmed, Packed, On its way, Delivered — each opening onto its sentence ("We have the order and are checking the pair against it." / "Paid, or confirmed by phone for cash on delivery; the pair is set aside for you." / "Boxed with its tags and our receipt, waiting for the courier." / "With GHN or GHTK; the tracking number is on the order's page." / "In your hands. The size exchange window starts today.").
3. Under them the four promises in compact form, and, when this browser holds orders, "Ordered on this phone" (ORD-023).
4. "While you wait" holds the line "Most orders arrive in two to four working days. If yours is late or the number is not found, message us with the order number.", a "Message us" pill to `/contact`, and the shop's questions of topic "orders" and "shipping" as closed rows.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | `/track` | It loads | The heading, the lead, the two labelled fields (the first with placeholder `DOI-24817`) and "Find my order" are shown |
| AC-2 | `/track` | The visitor opens each of the five rows | The five sentences above are shown |
| AC-3 | `/track` | The visitor reads "While you wait" | The line, the "Message us" pill to `/contact`, and the questions are shown |
| AC-4 | The shop has one question of topic "orders" and four of "shipping" | `/track` loads | Five closed rows are shown |
| AC-5 | A 390px window | `/track` loads | The page is one column and the form is above the step rows |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: this requirement is the static page; the lookup's outcomes are in ORD-021 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Questions: topic, question, answer | read | no | Topics "orders" and "shipping" |

**Requirement ORD-021 — Find the order with its number and phone**

*As a* customer, *I want* the right number and phone to open my order, and anything else to tell me nothing, *so that* nobody can look at other people's orders.

Behaviour:

1. The page sends the typed number and phone to the shop. The shop verifies them against the order and, if both are right, answers with the order's id, number, time and an order token — the same token the order page reads with — and nothing else about the order.
2. The number is accepted with or without the `DOI-` prefix, in any case, with spaces or dashes around the digits: `DOI-24817`, `doi 24817` and `24817` are the same order.
3. The phone is compared by its digits, so `0912 345 678`, `0912345678` and `+84912345678` are the same phone. A leading `84` stands for the leading `0`.
4. A wrong number, a wrong phone and a number that does not exist all answer the same: 404 with the same body. The page shows "No order matches that number and phone. Check both, or message us." in alert style.
5. On a match the token is kept in this browser and the browser goes to `/orders/<id>`.
6. The phone must match in full, not by its last digits.
7. While the shop is answering, the button is disabled and reads "Searching…".

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The seeded order `DOI-24817`, phone `0938 410 552` | The visitor types `DOI-24817` and `0938 410 552` and presses "Find my order" | The browser goes to `/orders/doi-24817` and the heading contains "on its way" |
| AC-2 | The same | `doi 24817` and `+84938410552` are typed | The same order opens |
| AC-3 | The same | `24817` and `0938410552` are typed | The same order opens |
| AC-4 | The same | `DOI-24817` and `0938 410 553` are typed | The not-found message is shown |
| AC-5 | The same | `DOI-99999` and `0938 410 552` are typed | The not-found message is shown |
| AC-6 | AC-4 and AC-5 | Their two answers are compared | Same status and same body |
| AC-7 | The same | `DOI-24817` and `552` (the last digits only) are typed | The not-found message is shown |
| AC-8 | The same | The phone is empty | The shop answers 404 and the not-found message is shown |
| AC-9 | A right pair | The lookup answers | The body holds `id`, `number`, `placed_at`, `token` and has no `phone`, `email`, `address`, `note`, `customer` or `city` |
| AC-10 | A right pair | The browser's storage is read | It holds the id, number, time and token |
| AC-11 | The shop is slow (3 seconds) | The button is pressed | It is disabled and reads "Searching…" until the answer arrives |
| AC-12 | The design's flow "Track an order by its number" | `DOI-24817` and `0938 410 552` are typed and "Find my order" pressed | The order heading contains "on its way" |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | Wrong number or wrong phone | One generic answer (AC-4, AC-5, AC-6) |
| Invalid input | The number has no digits | The same not-found message |
| Upstream failure | The shop cannot be reached | "You seem to be offline. Check your connection and try again." when offline; otherwise "We could not look that up. Please try again." |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Typed order number and phone | read | yes | Compared on digits |
| Order token | read | yes | Returned on a match |

**Requirement ORD-022 — Too many wrong tries are refused**

*As a* customer, *I want* guessing to be stopped, *so that* strangers cannot find my order by trying numbers and phones.

Behaviour:

1. After 5 failed lookups within 15 minutes of each other for one order number, or from one network address, the sixth and later lookups of that order number, or from that address, are refused for 15 minutes counted from the fifth failure, whether or not they would have matched. A right lookup neither adds to the count nor clears it.
2. A lookup of a number that does not exist counts as a failure against that number and against the address like any other.
3. A refused lookup answers 429 with a note of the wait. The page shows "Too many tries. Please wait 15 minutes, or message us with your order number."
4. After the wait the count starts again.
5. A match never counts as a failure.
6. The order page opened with the token already held in this browser is not affected by a lookup lock.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Order `DOI-24817` | Five wrong phones are tried one after another | Each answers the generic not-found |
| AC-2 | The same | The sixth try is sent, even with the right phone | 429 and the page shows "Too many tries. Please wait 15 minutes, or message us with your order number." |
| AC-3 | The same | 15 minutes pass after the fifth failure and a right pair is sent | The order opens |
| AC-4 | Five failures from one address across five different order numbers | A lookup from that address for a sixth order | 429 |
| AC-5 | Five failures against one order number from address A | A lookup of that order from address B | 429 for that order number as well |
| AC-6 | Five failures against a number that does not exist | A sixth try | 429 (no difference from an existing number) |
| AC-7 | Three wrong tries, then a right one, then two wrong tries, all within 15 minutes | A further try is made | 429: the right try neither counted as a failure nor cleared the three |
| AC-8 | A lookup lock is on for `DOI-24817` | The order page `/orders/doi-24817` is opened in the browser that placed it | The order is shown |
| AC-9 | A refused lookup | The 429 is read | It carries the wait in seconds |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Permission | Locked | 429 (AC-2) |
| Conflict | Many lookups arrive at once | No more than five failures are answered before the lock begins |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Failure counts per order number and per address | write | yes | Reset after the wait |

**Requirement ORD-023 — "Ordered on this phone" and the order page reached by lookup**

*As a* customer, *I want* the orders I placed or looked up on this browser to be listed, and any of them to show how far along it is, *so that* I do not have to retype the number.

Behaviour:

1. When this browser holds at least one order token, "Ordered on this phone" lists up to five of them, newest first, each as the order number and its date, linking to `/orders/<id>`. With none, the heading and list are not drawn.
2. An order opened through the lookup shows the same order page as one just placed: the five steps or the cancelled sentence, and the carrier, tracking number and expected date once the dealer has set them.
3. Every order listed is one the browser has a token for; an order placed or looked up on another browser is not listed.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | This browser holds tokens for six orders | `/track` loads | Five rows are shown, newest first, each `<number>` and `<date>` |
| AC-2 | The same | The visitor presses a row | The browser goes to `/orders/<id>` and the order shows |
| AC-3 | No tokens | `/track` loads | "Ordered on this phone" is not drawn |
| AC-4 | An order was looked up with the right number and phone | `/track` is reopened | That order is listed |
| AC-5 | The dealer set carrier `GHN`, tracking `GHN-PT27XM` and expected `2026-11-26` on a shipped order | The customer looks it up and opens it | The page shows the carrier, the tracking number and "Expected by 26/11/2026." |
| AC-6 | The order is cancelled | The customer opens it | The cancelled sentence is shown in place of the steps |
| AC-7 | The browser refuses storage | `/track` loads | The list is not drawn |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Absent data | No tokens held | AC-3 |
| Not applicable | Roles | Not applicable: the list is per browser |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Browser store of orders: id, number, placed time, token | read | yes | |

## 5. Screens

| Screen | Section in the design | Functions it serves | States that must exist |
|---|---|---|---|
| Checkout `/checkout` | Checkout | ORD-001, ORD-002, ORD-006, ORD-009 | default; empty bag; placing ("Placing your order…"); validation messages |
| Checkout — order refused, failed, offline | Not drawn in the approved design; required by the plan item | ORD-004, ORD-010 | refused (named pairs); failed; offline |
| Order `/orders/:id` | Order | ORD-007, ORD-023 | placed (heading "We have your order." or "Thank you…"); confirmed; packed; shipped; delivered; cancelled; not found |
| Order — "Pay now", awaiting payment | Not drawn in the approved design; required by the plan item | ORD-007 | awaiting payment with "Pay now"; "Pay now" failed |
| Track `/track` | Track | ORD-020, ORD-021, ORD-023 | default; not found |
| Track — searching, too many tries, offline | Not drawn in the approved design; required by the plan item | ORD-021, ORD-022 | searching; too many tries; offline |

## 6. Non-functional requirements

| Area | Requirement |
|---|---|
| Integrity | Under 50 simultaneous orders for one size holding 10 pairs, exactly 10 pairs are sold, no count is below 0, and every refused order names its pair |
| Performance | Placing an order answers within 1.5 s at p95 over 100 sequential orders against the running API |
| Privacy | An order's phone, email, address and note appear in no response to anyone but the dealer; a lookup response holds only id, number, time and token; tokens hold at least 128 bits of randomness |
| Security | A lookup is refused after 5 failures for 15 minutes, per order number and per address |
| Responsive | Checkout, order and track pages work from 320px up with no horizontal page scroll |
| Accessibility | Every field has a visible label; validation messages and the lookup messages are announced to assistive technology (alert); radio rows can be chosen with the keyboard |
| Localisation | Copy is in English; money as `vi-VN`; dates `dd/mm/yyyy`; expected dates and "today" use Vietnam time |

## 7. Dependencies and assumptions

- **Depends on:** the `data` module (records, paging); the `admin` module (dealer session; the dealer moving orders along); the `payments` module (providers, the unpaid-order expiry that returns pairs).
- **Assumption:** between this story and the dealer sign-in story the dealer's order screens cannot read orders (ORD-008 item 5); the dealer's area is a mock until then.
- **Assumption:** a person can lock lookups of one order for 15 minutes by failing five times; the order link held in the customer's own browser is not affected (ORD-022 AC-8).
- **Assumption:** working days are Monday to Friday; Vietnamese public holidays are not skipped.

| # | Open question | Proposed default | Who decides |
|---|---|---|---|
| 1 | Seeded stock numbers cannot all be split into counts that agree with the lists: for example a shoe with 5 pairs in stock and 8 sizes. The numbers were placeholders (they are marked invented in the records) | Per-size counts are made to agree with the lists (sold out 0, last pairs 1 or 2, other sizes 3 or more), and each shoe's "pairs in stock" becomes their sum, so it may differ from the number seeded; the dealer enters the true stock by saving the shoe (`admin` module, ADMIN-015) | Stakeholder |
| 2 | Copy for the states the design does not draw: "We could not place your order. Nothing was charged. Please try again."; "You seem to be offline. Check your connection and try again."; "<shoe name> · EU <size> — sold out in your size." / "only <n> left in your size." / "no longer in the shop."; "Pay now"; "awaiting payment"; "We could not open the payment page. Please try again."; "Searching…"; "Too many tries. Please wait 15 minutes, or message us with your order number."; "We could not look that up. Please try again." | Use this copy, added to the site's content so the owner can edit it | Stakeholder |
| 3 | The Order page says "the address or the size can change until the order is packed", but no screen lets the dealer change the pairs of an order | The dealer can correct customer details (`admin` module, ADMIN-016); a size change is done by cancelling the order and the customer ordering again | Stakeholder |
| 4 | The design's words say "we confirm by text message" and "we text you when the pair is on its way"; the brief and plan build no text messages | Not built; the dealer confirms by phone from the order page; the words are the owner's to edit in the Studio | Stakeholder |
| 5 | "the order is marked paid at once and packed today" (design copy for online ways to pay) is true only after the provider's notice arrives | Keep as designed | Stakeholder |
| 6 | The drawer lets a customer raise a line above 5 pairs; the bag page stops at 5 | The shop checks stock only; no fixed cap beyond the pairs on the shelf | Stakeholder |

## 8. Traceability

| Plan item | Requirement ids | Test cases |
|---|---|---|
| Checkout and placing an order | ORD-001, ORD-002, ORD-003, ORD-004, ORD-005, ORD-006, ORD-007, ORD-008, ORD-009, ORD-010 | `test-cases/checkout-and-placing-an-order.md` |
| Track an order by number and phone | ORD-020, ORD-021, ORD-022, ORD-023 | `test-cases/track-an-order-by-number-and-phone.md` |
