# SRS — Payments: VNPay, and later VietQR and MoMo

Module: `payments`
Design: [View the approved design](http://localhost:9401)
Design system: `design/design-system.md`

> One file per module. This one covers paying for an order online. Placing the order, the order page and its
> "Pay now" button are in `docs/orders/SRS.md`; the dealer marking an order paid is in `docs/admin/SRS.md`.

## 1. Purpose

The shop is in Vietnam and prices in VND, so customers pay by card, banking app or QR, and the owner chose online payment as the only way to pay (there is no cash on delivery).
This module lets a customer pay an order on VNPay's page (Visa, Mastercard, JCB, domestic ATM cards
and QR), makes the shop mark the order paid only when VNPay itself says so, and gives up unpaid orders after 30
minutes so their pairs go back on the shelf. A later story adds a VietQR bank transfer and MoMo. Without it
"online payment" from the brief does not exist and an order can be "paid" by anyone who types an address.

## 2. Actors

| Actor | Who they are | What they may do in this module |
|---|---|---|
| Customer | A visitor with an order | Pay on VNPay's page; see the result; try again; pay later from the order page |
| VNPay | The payment gateway | Calls the shop's notice address to say a payment succeeded or failed |
| Owner | The shop owner | Supplies VNPay's keys, registers the notice address in VNPay's merchant portal |
| Dealer | The shop owner or staff, signed in | Marks a bank-transfer order paid (later story) |

## 3. Scope

**In scope** — the plan items, by title:

- Online payment with VNPay
- More ways to pay: VietQR transfer and MoMo (left for later, not needed to open)

**Out of scope**

- Placing an order, the order page, its "Pay now" button and which ways to pay checkout offers — belong to the `orders` module (ORD-006, ORD-007). This module is the provider they call; it edits no page of that story.
- Cash on delivery — not offered: the owner chose online payment only. Nobody marks an order paid by hand; only the later bank-transfer story lets the dealer do so (PAY-020).
- Refunds — not in the brief or plan; a refund is made by the dealer in VNPay's own merchant portal and is not recorded by the shop.
- Saving cards or customers' payment details — not built; VNPay's page takes them.
- Instalments and other VNPay products — not in the brief.

## 4. Functional requirements

### 4.1 Online payment with VNPay

**Requirement PAY-001 — A signed payment address for each attempt**

*As a* customer, *I want to* be sent to VNPay with the right amount for my order, *so that* I pay exactly what I owe.

Behaviour:

1. When an order is placed with VNPay, and whenever the customer asks to pay an unpaid VNPay order ("Pay now" on the order page, "Try again" on the result page), the shop builds VNPay's payment address for that order.
2. The address carries the shop's terminal code, the amount, the currency VND, a description naming the order number, a reference that is different for every attempt, the address VNPay sends the customer back to (`/payment/return` on the shop), and the moment the payment page stops accepting payment — the order's 30-minute deadline (PAY-003).
3. The amount is the order's stored total, read from the database; no amount from the browser is used.
4. The address is signed with the shop's hash secret (HMAC-SHA512 over the parameters, as VNPay requires).
5. Every earlier reference of the order still leads to the same order.
6. VNPay is the only way to pay at checkout, and no order can be completed without it. When either key is missing, checkout still shows the VNPay row but shows under it "Online payment is not available right now. Please try again later." in alert style and disables "Place order"; a request to place an order answers 503, creates no order and takes no pair off the shelf; "Pay now" for an unpaid order answers 503.
7. "Pay now" for an order that is paid, cancelled or not a VNPay order is refused with 409 (ORD-007).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A VNPay order totalling 3.140.000 ₫ | The payment address is built | It carries the amount for 3.140.000 VND in VNPay's unit, the currency VND, and a reference holding the order number |
| AC-2 | The browser's request names an amount of 1.000 ₫ | The address is built | The amount is still the stored total |
| AC-3 | The same order | The address is built twice | The two references differ |
| AC-4 | A built address | Its signature is recomputed with the same secret | It matches; with another secret it does not |
| AC-5 | A built address | Its return address is read | It is the shop's `/payment/return` |
| AC-6 | An order created 10 minutes ago | The address is built | Its payment-page expiry is 20 minutes from now (the order's 30-minute deadline) |
| AC-7 | The keys are not set | `/checkout` loads with a pair in the bag | The VNPay row, the line "Online payment is not available right now. Please try again later." in alert style and a disabled "Place order" are shown |
| AC-8 | The keys are not set | "Pay now" is pressed for a VNPay order | The shop answers 503 |
| AC-9 | Checkout with VNPay chosen | "Place order" succeeds | The browser goes to a VNPay address whose amount equals the Total shown at checkout |
| AC-10 | The keys are not set | A request to place an order is sent | 503; no order exists and no pair left the shelf |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Upstream failure | The keys are missing | AC-7, AC-8, AC-10 |
| Permission | "Pay now" without the order's token | 404 (ORD-008) |
| Boundary | Order within its deadline by under a minute | The address is built with an expiry at the deadline |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Order: number, total, paid, status, created time | read | yes | |
| Payment attempt: reference, order, created time | write | yes | One per attempt; unique reference |
| Terminal code, hash secret | read | yes | Held by the shop only; never sent to a browser |

**Requirement PAY-002 — VNPay's notice is the only thing that marks an order paid**

*As a* dealer, *I want* an order to be marked paid only when VNPay confirms it, *so that* nobody can get pairs without paying.

Behaviour:

1. VNPay calls the shop's notice address (`GET /api/v1/payments/vnpay/ipn`) from its own servers with the result of a payment. The address needs no sign-in; the signature is what proves who is calling.
2. The shop checks, in this order: the signature (computed with the hash secret) is valid; the reference belongs to an order; the amount equals the order's total in VNPay's unit; the order has not already been marked paid by an earlier notice.
3. It answers with HTTP 200 and the JSON VNPay requires, `{"RspCode": "<code>", "Message": "<text>"}`: `00` confirm success; `01` order not found; `02` order already confirmed; `04` invalid amount; `97` invalid signature; `99` unknown error.
4. When the notice says the payment succeeded (response code `00` and transaction status `00`), the order is marked paid, VNPay's transaction number, bank and time are kept with it, and the answer is `00`. An order still at "placed" moves to "confirmed" in the same step.
5. When the notice says the payment failed or the customer cancelled, the order stays unpaid and the answer is `00` (the shop has received it).
6. A second notice for an order already marked paid answers `02` and changes nothing. Two notices arriving at the same moment mark the order paid once.
7. A notice with a bad signature, an unknown reference or a wrong amount changes nothing.
8. The browser returning to the shop never marks anything paid, whatever its address says.
9. A notice that confirms a payment for an order already cancelled (PAY-003) is acknowledged with `00`, the payment is recorded and the order's "paid" is set while its status stays "cancelled", so the dealer sees it on the order and refunds it (open question 2).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | An unpaid VNPay order of 3.140.000 ₫ and a correctly signed success notice for that amount | The notice arrives | The shop answers `{"RspCode":"00","Message":"Confirm Success"}`, the order is paid, and its transaction number is kept |
| AC-2 | The same order at status "placed" | The notice is processed | The status is "confirmed" |
| AC-3 | The same notice | It arrives a second time | The answer is `02` and the order is unchanged |
| AC-4 | Two copies of the same notice | They arrive at the same moment | One answers `00`, the other `02`; the order is paid once; one transaction is kept |
| AC-5 | A notice whose signature was altered by one character | It arrives | The answer is `97` and the order is unpaid |
| AC-6 | A correctly signed notice for the wrong amount | It arrives | The answer is `04` and the order is unpaid |
| AC-7 | A correctly signed notice for a reference that matches no order | It arrives | The answer is `01` |
| AC-8 | A correctly signed notice whose response code says the customer cancelled | It arrives | The answer is `00` and the order is unpaid |
| AC-9 | An unpaid order | The browser opens `/payment/return?...` with a success code and a valid-looking signature | The order stays unpaid |
| AC-10 | A request to the notice address with no query | It arrives | 200 with a JSON `RspCode` of `97` |
| AC-11 | A signed success notice for an order already cancelled | It arrives | The answer is `00`, the order's status is still "cancelled" and "paid" is true |
| AC-12 | Any answer to the notice address | Its headers are read | The status is 200 and the body is JSON with `RspCode` and `Message` |
| AC-13 | The VNPay sandbox's test card | A payment is made from checkout | The order is marked paid exactly once |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | Bad signature / unknown reference / wrong amount | `97` / `01` / `04`; no change |
| Conflict | Duplicate or simultaneous notices | `02` for all but the first (AC-3, AC-4) |
| Upstream failure | The database fails while processing | `99`, so VNPay tries the notice again later; nothing is half-written |
| Permission | Anyone can call the address | Only a correctly signed notice has any effect |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Order: paid, status | write | yes | Only by a verified notice |
| Payment record: reference, VNPay transaction number, bank, amount, response code, time | write | yes | Kept for each notice received |

**Requirement PAY-003 — An unpaid online order is given up after 30 minutes**

*As a* customer, *I want* the pairs I did not pay for to go back on sale, *so that* others can buy them.

Behaviour:

1. An order whose way to pay is VNPay (and, from the later story, MoMo — PAY-021) that is still unpaid 30 minutes after it was placed and whose status is placed, confirmed or packed is cancelled. The rule is scoped to those orders: an unpaid bank-transfer order is cancelled only after 24 hours (PAY-020 item 5), and an unpaid order that is already shipped or delivered is left alone — the dealer cannot cancel a shipped order (ADMIN-016 item 2), so the rule does not either; it stays as it is, unpaid.
2. Cancelling puts the order's pairs back on the shelf; the shoes' sold-out and last-pairs lists and "pairs in stock" are brought up to date (ORD-005).
3. The check runs at least once a minute, so an order is cancelled within 31 minutes of being placed.
4. Orders that are paid, orders already cancelled, orders that are shipped or delivered, and orders whose way to pay is bank transfer are left alone by this rule. The sample orders that came with the first design (DOI-24804 to DOI-24817) are never cancelled by this rule, so the figures in `docs/admin/SRS.md` hold (open question 7).
5. The order page of a cancelled order shows the cancelled view (ORD-007); "Pay now" answers 409.
6. The payment page VNPay shows stops accepting payment at the same deadline (PAY-001).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A VNPay order placed 31 minutes ago, unpaid | The check has run | The order's status is "cancelled" |
| AC-2 | The same order held 2 pairs of size `38` | The order is cancelled | Size `38` has 2 more pairs and the shoe's lists and total agree with the counts |
| AC-3 | A VNPay order placed 29 minutes ago | The check runs | It is unchanged |
| AC-4 | A paid VNPay order placed 2 hours ago | The check runs | It is unchanged |
| AC-5 | An unpaid VNPay order placed through checkout 2 hours ago, which the dealer moved to "confirmed" (and the same for "packed") | The check runs | It is cancelled and its pairs return |
| AC-6 | A VNPay order the dealer cancelled at minute 5 | The check runs at minute 31 | Its pairs are not returned a second time |
| AC-7 | A cancelled VNPay order | "Pay now" is pressed with its token | 409 |
| AC-8 | Two checks run at the same moment | Both see an order to cancel | The order is cancelled once and its pairs return once |
| AC-9 | A sample order from the first design, unpaid and placed more than 30 minutes ago | The check runs | It is unchanged |
| AC-10 | An unpaid VNPay order placed 2 hours ago, which the dealer moved to "shipped" (and the same for "delivered") | The check runs | It is unchanged: status stays shipped (or delivered), unpaid, and its pairs are not returned |
| AC-11 | An unpaid bank-transfer order placed 2 hours ago (after the later story, PAY-020) | The 30-minute check runs | It is unchanged; only the 24-hour rule of PAY-020 cancels it |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Conflict | A paid notice arrives as the check runs | Exactly one wins: either the order is paid and left alone, or it is cancelled and PAY-002 item 9 applies |
| Boundary | The order is exactly 30 minutes old | It is cancelled by the next run |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Order: status, created time, way to pay, paid | read / write | yes | |
| Per-size pairs | write | yes | Returned on cancel |

**Requirement PAY-004 — The payment result page**

*As a* customer, *I want* to be told plainly whether my payment worked after VNPay sends me back, *so that* I know whether to pay again.

Behaviour:

1. VNPay sends the customer to `/payment/return` with its result in the address. The page works out which order it is from the reference in the address and the orders this browser holds tokens for, and asks the shop for the order's state with that token. It never decides the result from the address.
2. The page says one of these, in plain words:
   - Paid: heading "Payment received." and the line "Your order <number> is paid. We will pack it next."
   - Confirming: heading "We are confirming your payment…" and the line "This usually takes a few seconds. Please do not pay again." — shown when the address says the payment succeeded but the shop has not yet had VNPay's notice. The page asks again every 3 seconds for up to 30 seconds and switches to Paid when the notice arrives. If 30 seconds pass without it, the heading becomes "We have not heard from VNPay yet." and the line "If money left your account it will show on your order once VNPay confirms it. Please do not pay again."
   - Not completed: heading "Payment was not completed." and the line "You can try again, or place a new order." — shown when the order is unpaid and the address says the payment failed or was cancelled.
   - Cancelled: heading "This order was cancelled." and the line "The time to pay ran out, so the pairs went back on the shelf. You can order again."
3. Every state has a "View your order" link to `/orders/<id>`. "Not completed" also has a "Try again" button; "Cancelled" has a "Keep shopping" pill to `/shop`. "Try again" asks the shop for a new payment address and sends the browser to it; it is not shown for a paid or cancelled order, while confirming, or after the confirming wait timed out (a customer who may already have paid is not invited to pay again; the order page's "Pay now" remains).
4. When the page cannot match the reference to an order this browser holds a token for, it says "We can't match this payment to an order on this browser." with the line "Look the order up with its number and phone." and a "Track an order" pill to `/track`.
5. The page has its own address in the site's routes and is drawn in the shop's page shell (header and footer), in the site's own type and colours.
6. The page works on a phone.
7. A customer who closes the tab can pay later, until the 30-minute deadline, from the order page's "Pay now".

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The order was marked paid by VNPay's notice | `/payment/return` opens with its reference | "Payment received." and "Your order DOI-24818 is paid. We will pack it next." and a "View your order" link to `/orders/doi-24818` |
| AC-2 | The address says success, the notice has not arrived | The page opens | "We are confirming your payment…" with the line "This usually takes a few seconds. Please do not pay again." and no "Try again" |
| AC-3 | The same, and the notice arrives after 6 seconds | The page waits | It switches to "Payment received." with no reload |
| AC-4 | The same, and no notice arrives for 30 seconds | The page waits | The heading reads "We have not heard from VNPay yet." with the line and "View your order", and no "Try again" |
| AC-5 | The address says the customer cancelled; the order is unpaid | The page opens | "Payment was not completed." with the line, "View your order" and "Try again" |
| AC-6 | The same | "Try again" is pressed | The shop gives a new payment address with a new reference and the browser goes to it |
| AC-7 | The order was cancelled by the 30-minute rule | The page opens | "This order was cancelled." with the line, "View your order" and a "Keep shopping" pill, and no "Try again" |
| AC-8 | The reference is of an order this browser has no token for | The page opens | "We can't match this payment to an order on this browser." with "Look the order up with its number and phone." and a "Track an order" pill |
| AC-9 | The address says success but the order is unpaid and the notice never arrives | The page and the order are read after 60 seconds | The order is unpaid |
| AC-10 | A 390px window | The page opens | The heading, the line and the buttons fit with no horizontal scroll |
| AC-11 | The design's flow with VNPay | A pair is chosen, checkout is completed with VNPay and a sandbox payment is made | The browser ends on `/payment/return` and then, through "View your order", on the order |
| AC-12 | The page is open | The visitor looks at it | The site's header and footer surround it and its heading is the page's one h1 |
| AC-13 | A customer who closed the tab at VNPay's page | The order page is opened within 30 minutes | "Pay now" is shown |
| AC-14 | The change set of this story | It is listed | It adds the result page and its route line in `src/App.tsx`, and backend files; it edits no page of the Checkout story (`Checkout.tsx`, `Order.tsx`) |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | The reference matches no order held by this browser | AC-8 |
| Permission | The reference is of someone else's order | The browser holds no token for it: AC-8; nothing about that order is shown |
| Upstream failure | The shop cannot be reached | The page shows "We could not check your payment. Please open your order to see its state." with a "View your order" link when the order is known, otherwise the "Track an order" pill |
| Boundary | The page is refreshed | The same state is drawn again from the shop's answer |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Address: VNPay's reference and response code | read | yes | Used only to find the order and decide whether to wait; never to mark paid |
| Order: number, status, paid | read | yes | By token |

**Requirement PAY-005 — Keys, notice address and going live**

*As the* owner, *I want* to put VNPay's keys in once and register one address, *so that* payments work in the sandbox first and in production later.

Behaviour:

1. The shop reads `VNPAY_TMN_CODE` (the terminal code) and `VNPAY_HASH_SECRET` from its settings; the owner supplies them in the console, sandbox ones first, from a sandbox merchant at `sandbox.vnpayment.vn`.
2. The owner registers the shop's notice address (`/api/v1/payments/vnpay/ipn` on the deployed site) in VNPay's merchant portal for the terminal. The console shows the exact address.
3. Moving to production changes the keys and VNPay's payment page address in the shop's settings, not its code.
4. The secrets are never sent to a browser, never written to a log, and never held in a variable that begins with `VITE_`.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Both keys set | The server starts | VNPay is the only way to pay at checkout and "Place order" is enabled |
| AC-2 | A key missing | The server starts | The server still starts; checkout shows "Online payment is not available right now. Please try again later." with "Place order" disabled, and placing an order answers 503 |
| AC-3 | The built site | Its files and responses are searched for the hash secret | It appears nowhere |
| AC-4 | The server's log after a payment | It is searched for the hash secret | It appears nowhere |
| AC-5 | The settings' example file | It is read | It names `VNPAY_TMN_CODE` and `VNPAY_HASH_SECRET` with no value |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Absent data | A key is missing | AC-2 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| VNPAY_TMN_CODE, VNPAY_HASH_SECRET | read | yes for VNPay | Settings only |

### 4.2 More ways to pay: VietQR transfer and MoMo

This story is left for later and is not needed to open the shop. Until it is built, checkout offers VNPay only (ORD-006). Its requirements are written at the level the later story needs; its keys are requested from the owner when the story starts.

**Requirement PAY-020 — Bank transfer with a VietQR code**

*As a* customer, *I want to* pay by scanning a QR code with my banking app, *so that* I can pay without a card.

Behaviour:

1. "Bank transfer (VietQR)" is offered at checkout only when the shop's own bank account — bank, account number and account holder, supplied by the owner — is set.
2. After an order is placed with it, the order page shows a QR code that carries the shop's account, the order's total as the amount, and the order number as the transfer note, with the account holder, bank, account number, amount and note written beside it for typing by hand, and the line "Scan this QR code with any banking app. We send the pair as soon as the money lands."
3. Nothing marks the order paid automatically. The dealer marks it paid on the order's page when the money has landed. This is the one case where a dealer marks an order paid by hand, and it exists only from this later story: until then the `admin` module refuses every such attempt (ADMIN-016).
4. "Pay now" answers 409 for a bank-transfer order, so the order page draws no "Pay now" for an unpaid one and shows the QR code in its place; the QR code stays on the order page while the order is unpaid and is not shown once it is paid or cancelled.
5. A bank-transfer order still unpaid 24 hours after it was placed, whose status is placed, confirmed or packed, is cancelled and its pairs return to the shelf (open question 5). A shipped or delivered order is left alone, as in PAY-003 item 1.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The shop's bank account is set | `/checkout` loads | "Bank transfer (VietQR)" is listed |
| AC-2 | The account is not set | `/checkout` loads | It is not listed |
| AC-3 | An order of 3.140.000 ₫ placed with it | The order page opens | A QR code is shown with the account holder, bank, account number, `3.140.000 ₫` and the order number as the note |
| AC-4 | The QR code's content is decoded | It is read | It names the shop's account, the amount `3140000` and the order number |
| AC-5 | The dealer marks the order paid | The customer reopens the order page | "Payment" reads `Bank transfer (VietQR) · paid` and no QR code is shown |
| AC-6 | An unpaid bank-transfer order | "Pay now" is called | 409 |
| AC-7 | An unpaid bank-transfer order placed 25 hours ago | The check runs | It is cancelled and its pairs are back on the shelf |
| AC-8 | An unpaid bank-transfer order placed 23 hours ago | The check runs | It is unchanged |
| AC-9 | A visitor with no token | The order's QR details are asked for | 404 |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Absent data | The account is not set | Not offered (AC-2) |
| Permission | A stranger or a customer tries to mark paid | 401 / 403 (the `admin` module's rule); the order stays unpaid |
| Conflict | The dealer marks paid at the moment the 24-hour check cancels | One wins; a cancelled order marked paid keeps status "cancelled" and "paid" true so the dealer refunds it |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shop's bank, account number, account holder | read | yes for this way to pay | Settings supplied by the owner |
| Order: total, number, paid | read / write | yes | |

**Requirement PAY-021 — MoMo wallet**

*As a* customer, *I want to* pay with my MoMo wallet, *so that* I can pay without a card.

Behaviour:

1. MoMo is offered at checkout only when its merchant keys are set (supplied by the owner when the story starts).
2. The order's payment address comes from MoMo, signed with the shop's secret, with the amount read from the order and a reference that is different for every attempt.
3. MoMo's own notice to the shop's notice address is the only thing that marks the order paid. The shop checks the signature, the order, the amount and that it was not processed before, exactly as for VNPay (PAY-002); a bad signature, an unknown order or a wrong amount changes nothing; a repeated notice changes nothing.
4. An order still unpaid 30 minutes after it was placed is cancelled and its pairs return to the shelf (PAY-003).
5. MoMo sends the customer back to `/payment/return`, which reads the order's state from the shop exactly as for VNPay (PAY-004) and says "MoMo" where it says "VNPay".
6. The owner registers the shop's MoMo notice address in MoMo's merchant portal; the console shows the exact address.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | MoMo's keys are set | `/checkout` loads | "MoMo wallet" is listed |
| AC-2 | The keys are not set | `/checkout` loads | It is not listed |
| AC-3 | An order of 3.140.000 ₫ placed with MoMo | The payment address is built | It carries the amount 3140000 and a signature that verifies with the shop's secret |
| AC-4 | A correctly signed MoMo success notice for the right amount | It arrives | The order is marked paid once |
| AC-5 | The same notice | It arrives a second time | The order is unchanged |
| AC-6 | A notice with an altered signature | It arrives | The order stays unpaid and MoMo is told the notice was refused |
| AC-7 | An unpaid MoMo order placed 31 minutes ago | The check runs | It is cancelled and its pairs return |
| AC-8 | A customer returning from MoMo | `/payment/return` opens | The page reads the order's state from the shop and says "MoMo" |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | Bad signature / wrong amount / unknown order | No change (AC-6) |
| Conflict | Duplicate notices | The order is marked paid once (AC-5) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| MoMo merchant keys | read | yes for this way to pay | Settings only; never sent to a browser |
| Payment record: reference, MoMo transaction number, amount, time | write | yes | |

**Requirement PAY-022 — Switch both on at checkout**

*As a* customer, *I want to* see the new ways to pay at checkout once the shop can take them, *so that* I can choose them.

Behaviour:

1. Checkout lists "Bank transfer (VietQR)" and "MoMo wallet" when each is configured (PAY-020, PAY-021), in the order of the records, after VNPay; it lists neither when neither is.
2. No page holds a list of ways to pay: checkout, the shoe page's "Payment options" and the order page draw what the shop's answer says it can complete (ORD-006). As built after the VNPay story, `Checkout.tsx`, `src/lib/orders.ts` and `Product.tsx` name VNPay alone, so this story edits them, only where they named VNPay (and the order page only where it decides "Pay now"); it edits nothing else of those pages.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | VNPay, VietQR and MoMo are all configured | `/checkout` loads | Three rows: VNPay first, then bank transfer, then MoMo; no cash on delivery |
| AC-2 | Only VietQR is configured of the two | `/checkout` loads | VNPay is listed first, bank transfer after it, and MoMo is not listed |
| AC-3 | The change set of this story | It is listed | No page names a way to pay in code (the ids come from the shop's answer); `Checkout.tsx`, `Product.tsx` and the order page's parts are edited only where they named VNPay alone or decided "Pay now" |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: no failure state of its own; absent configuration hides the row (AC-2) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Ways to pay the shop can complete | read | yes | |

## 5. Screens

| Screen | Section in the design | Functions it serves | States that must exist |
|---|---|---|---|
| Payment result `/payment/return` | Not in the approved design: a new page defined by the plan item "Online payment with VNPay", built from the site's own parts (display heading, soft line, pills) in the page shell | PAY-004, PAY-021 | paid; confirming; confirming timed out; not completed; cancelled; cannot match the order; could not check |
| Order page — QR code block | Not in the approved design: defined by the plan item "More ways to pay" | PAY-020 | bank-transfer order unpaid (QR shown); paid (QR not shown) |

Every other screen this module touches (checkout, order page, "Pay now") is specified in `docs/orders/SRS.md`.

## 6. Non-functional requirements

| Area | Requirement |
|---|---|
| Integrity | Two identical notices sent at the same moment mark an order paid exactly once; a notice with a bad signature, an unknown reference or a wrong amount changes no order, in 100 of 100 attempts |
| Performance | The notice address answers within 2 s at p95 over 100 sequential signed notices |
| Security | The terminal code and hash secret (and later MoMo's secret) are held in the server's settings only: absent from every browser file, response and log, and never in a `VITE_` variable |
| Privacy | Per payment the shop keeps the provider's transaction number, bank code, amount, result code and time; it receives and keeps no card number |
| Responsive | The result page works from 320px up with no horizontal page scroll |
| Accessibility | The result page's heading is the page's one h1; its state line is announced to assistive technology when it changes (confirming to paid) |
| Localisation | Copy in English; amounts as `vi-VN` on the shop's pages; amounts to VNPay in VND |

## 7. Dependencies and assumptions

- **Depends on:** the `orders` module's payment interface (ORD-006) and order token (ORD-008); the `admin` module, which shows an order's payment state read-only (ADMIN-016).
- **Depends on:** the owner supplying `VNPAY_TMN_CODE` and `VNPAY_HASH_SECRET` (sandbox first) in the console, and registering the notice address in VNPay's merchant portal. VNPay's servers cannot reach a deployment on a `.localhost` address, so on one the "paid exactly once" checks are made with signed test notices built with the sandbox secret.
- **Assumption:** "pay later from the order page" means within the 30 minutes before the order is given up. After that the order is cancelled and the customer places a new one.
- **Assumption:** VNPay accepts a notice answer of `00` for a payment that failed, as its notice documentation says, so it stops sending it.

| # | Open question | Proposed default | Who decides |
|---|---|---|---|
| 1 | The design's words for the order's "Confirmed" step now say "Paid online; the pair is set aside for you." (ORD-020), so a paid order is confirmed | A verified notice moves an order from "placed" to "confirmed" along with marking it paid (PAY-002 item 4) | Stakeholder |
| 2 | VNPay's notice can reach a minute after the 30-minute deadline for a customer who paid at the last second; the pairs may already be sold | The payment is recorded, "paid" is set and the order stays cancelled so the dealer sees it and refunds it in VNPay's portal | Stakeholder |
| 3 | The result page is not in the approved design | The wording of PAY-004, in the site's own styles; the owner can edit the words in the Studio | Stakeholder |
| 4 | A customer who paid and whose notice is late sees "We have not heard from VNPay yet." with no "Try again" | As written (PAY-004) | Stakeholder |
| 5 | A bank transfer takes hours, so the 30-minute rule does not suit it | Cancel an unpaid bank-transfer order after 24 hours | Stakeholder |
| 6 | The owner's bank, account number and holder name for VietQR, and MoMo's merchant keys, are not yet supplied | Requested in the console when the later story starts | Owner |
| 7 | The rule covers every unpaid online order, so the sample orders that came with the design (some unpaid) would be cancelled and would change the dashboard figures | The rule skips the sample orders DOI-24804 to DOI-24817; it applies to every VNPay (and, later, MoMo) order placed through checkout whose status is placed, confirmed or packed; unpaid shipped or delivered orders and bank-transfer orders (24 hours, PAY-020) are left alone | TL |

## 8. Traceability

| Plan item | Requirement ids | Test cases |
|---|---|---|
| Online payment with VNPay | PAY-001, PAY-002, PAY-003, PAY-004, PAY-005 | `test-cases/online-payment-with-vnpay.md` |
| More ways to pay: VietQR transfer and MoMo | PAY-020, PAY-021, PAY-022 | `test-cases/more-ways-to-pay-vietqr-transfer-and-momo.md` |
