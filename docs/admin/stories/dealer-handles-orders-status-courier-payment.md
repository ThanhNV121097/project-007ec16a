# Story — Dealer handles orders: status, courier, payment

Module: `admin` · Plan item 17 · Requirements narrowed: ADMIN-016 and the order parts of ADMIN-004 (AC-6, AC-9), ADMIN-011 (AC-12, AC-14, AC-15, AC-20) and ADMIN-014 (AC-7, AC-8) (`docs/admin/SRS.md` §4.1, §4.2). Split out of plan item 10, "Manage shoes, lines, categories, reviews and orders", so that story stays near 40 criteria.
Design system: `design/design-system.md` §2.2 (management area, scope `[data-dashboard]`) · Architecture: `docs/architecture/overview.md`; `erd.md` (`orders`, `order_items`, `product_sizes`, view `product_stock`); `services.md` (§2.5, §2.8 the order and its dealer view, §3.11 read one order, §3.14 list, §3.15 `PUT /v1/orders/{id}`, §3.16 `DELETE /v1/orders/{id}`). The contracts there are the starting point; TL's design step for this story may extend them (section 8).

## 1. User story

As **the shop's team** — the admin, and the staff the admin adds; both may do everything below — I want to move an order along from placed to delivered, cancel it when it cannot go, give the courier's details, correct the customer's details, see whether it is paid, and delete a cancelled one, so that customers see the truth about their order and the shelf shows the pairs that are really there.

Other actors this story touches:

- **Customer / visitor** — sees the status, carrier, tracking number and expected date on their own order page the next time it loads. Never sees phone, email, address or note, and has no way to reach `PUT` or `DELETE` on an order (401 with no session, 403 with a customer session).
- **Owner** — decided (binding, carried over from plan item 10): the shop takes **online payment only, with no cash on delivery**. The dealer cannot mark an order paid by hand, whatever its way to pay; only VNPay's own notice (plan item 8) sets `paid`. The order page in admin shows "Paid" or "Awaiting payment" and VNPay's transaction number when there is one, read-only, with no control to change either.
- **VNPay** — its notice, and the 30-minute expiry of unpaid orders, belong to plan item 8. This story only reads their result and builds the one cancel that item 8 will also call.

## 2. What exists today

- `src/pages/AdminOrders.tsx` (the list) and `src/pages/AdminOrder.tsx` (the record page with the sentence and the tabs Order, Delivery, Customer, built from `schema.json`'s `orders` fields and its `section`s) are drawn as designed, but "Save" and "Delete" write into the browser's memory only. Plan item 10 replaces that with real saving in `components/admin/store.ts`, `ListPage.tsx` and `RecordPage.tsx` (saving state, field errors beside their fields, "We could not save. Please try again.", "Your session ended. Sign in again.", the delete refusal inside the open question). **This story must not edit those three files**, nor `Form.tsx` or `fields.tsx`; it edits only `AdminOrders.tsx`, `AdminOrder.tsx` and its own components.
- `src/pages/admin-nav.ts` already reads the menu's Orders count (`status` placed or confirmed, `limit` 1, `total`). It is not edited here; SC-26 proves the count is true.
- `internal/orders` (plan item 5, merged) has `POST /v1/orders`, `GET /v1/orders/{id}` for the order's own token (the public view: never phone, email, address, note), `GET /v1/orders` for the team (`auth.Require(admin, staff)`; `list.go` marks `phone`, `email`, `address`, `note` private so the reader never selects them) and `GET /v1/payment-options`. It has **no** `Update`, `Cancel` or `Delete`, no `PUT` or `DELETE` route, and no dealer view.
- `internal/stock` (merged) has `Lock` and `Take`. `Return` (services §1) does not exist yet; this story adds it. Plan item 10 adds `Set` and `SetOne` to the same package.
- The customer's order page (`Order.tsx`) and its read already carry `carrier`, `tracking` and `eta`, so the customer sees the dealer's values with no change on the customer side.
- `src/lib/mock/orders.ts` stands in for orders for the Track page and **stays** until the Track story removes it.

## 3. In scope

**Backend (Go, PostgreSQL) — dealer session only: 401 with none, 403 for a customer, the same for admin and staff**

1. `PUT /v1/orders/{id}` (services §3.15) through `orders.Update` and `orders.Cancel`, and `DELETE /v1/orders/{id}` (§3.16) through `orders.Delete`. `PUT` is partial: only the fields sent change.
2. **Status.** One step forward along placed → confirmed → packed → shipped → delivered, or `cancelled` from placed, confirmed or packed. A skip, a step back, leaving delivered or cancelled, and cancelling a shipped order are 422 "An order cannot go from <status> to <status>." naming `status`. The current status sent again is a no-op; delivered and cancelled accept no `status` at all.
3. **Cancelling** is one transaction: a conditional `UPDATE … WHERE status IN ('placed','confirmed','packed')`, and only if a row changed, `stock.Return` for every pair of the order. A second cancel changes no row and is 422, so the pairs return exactly once, also when two cancels arrive together (and when plan item 8's expiry ticker or a late notice races the dealer). The shoes' last-pairs list, sold-out list and total follow, because they are derived from the counts.
4. **Courier.** `carrier`, `tracking` (text, 200 characters at most) and `eta` (`YYYY-MM-DD`, or `""` to clear) may be set while the order is not delivered or cancelled; the order's own page shows them the next time it loads.
5. **Customer details.** `customer`, `phone`, `email`, `address`, `city` may change while the order is not delivered or cancelled (the limits of services §2.5: text at most 200 characters, a phone of at least 9 digits, an email empty or a valid address); `note` (5,000 characters at most) at any time.
6. **What never changes.** `number`, `placed_at`, `product`, `size`, `qty`, `items`, `total`, `delivery_fee`, `payment`, `delivery` and `transaction_no` are refused with 422 naming the field. **`paid` is refused with 422 "Only the payment provider marks this order paid." for every order, whatever its way to pay** (owner decision above). A refused body applies nothing.
7. **Delete.** Only a cancelled order: otherwise 409 "Cancel the order first."; a cancelled order that is also paid is refused with 409 "A paid order cannot be deleted; refund it in VNPay's portal first." (services open question 5); otherwise 204, and the order's pairs, tokens and payment records go with it.
8. **The dealer's reads.** `GET /v1/orders` and `GET /v1/orders/{id}` answer a dealer session with the **dealer view**: `phone`, `email`, `address`, `note` and `transaction_no` (VNPay's number, `""` when none) besides the public fields, and on the single read `items` with every pair of a multi-pair order. No visitor, token or customer read ever carries them (ORD-008 governs token and linked-customer reads; this story does not widen them).
9. A save or delete answers within 1 s at p95.

**Frontend (Vite + React + Tailwind tokens)**

10. `/admin/orders` (columns number, placed, status, shoe, total, payment, customer; filters Status and Payment; search "Search by number or customer"; no Add button) reads the shop; the "shoe" column of a multi-pair order shows its first pair. The menu's Orders count is the number of orders waiting (placed or confirmed).
11. `/admin/orders/:id` with tabs Order, Delivery, Customer, built on item 10's record-page plumbing: the dealer edits only status, carrier, tracking, expected date, customer, phone, email, address, city and note; the status control offers only the moves the shop allows from the current status; an order that is delivered or cancelled is drawn locked except for its note. The private fields (phone, email, address, note) are on this page only.
12. The one-pair sentence of the design is kept; an order of several pairs shows "<customer> in <city> ordered <n> pairs on <date> — <total>." with each pair listed (picture, name, EU size, quantity, unit price).
13. A read-only block shows the number, placed time, way to pay, "Paid" or "Awaiting payment" and, when there is one, VNPay's transaction number, with no control to change any of them.
14. The page's client is `src/lib/dealer-handles-orders-status-courier-payment.ts` on top of `src/lib/api.ts`. Tests beside the existing ones.

## 4. Out of scope — and where each piece goes

| Not here | Why | Goes to |
|---|---|---|
| Marking an order paid or unpaid by hand, cash on delivery, a "Mark as paid" button | Owner decision: online payment only; only VNPay's notice sets `paid`. Until the bank-transfer story exists every `paid` in a body is refused (ADMIN-016 item 4) | "More ways to pay: VietQR transfer and MoMo" (plan item 12, P3) lets the dealer mark a **bank-transfer** order paid and no other |
| VNPay's signed notice, writing `paid` and the transaction number, the payment return page, the 30-minute expiry of unpaid orders | They are the payments story. This story only **reads** `paid` and `transaction_no` and builds the one `orders.Cancel` the expiry ticker also calls | "Online payment with VNPay" (plan item 8) |
| Refunds | Made by the dealer in the gateway's portal; not recorded in the shop (SRS §3) | — (see ⚠️ in the reply) |
| Changing the pairs, sizes or prices of a placed order | Not built (SRS open question 4): cancel and order again | — |
| A confirmation question before "Cancelled" is saved | Not in the SRS or the design | — (see ⚠️ in the reply) |
| Email or SMS to the customer when the status or courier changes | The shop sends none anywhere; the customer reads the order page or `/track` | — (see ⚠️ in the reply) |
| A history of who changed which order, an undo | Not in the SRS | — (see ⚠️ in the reply) |
| Bulk actions (move or cancel many orders), printing a packing slip or invoice, exporting orders | Not in the design | — |
| Adding an order by hand | Orders are created only by customers at checkout (ADMIN-010 item 1); there is no "Add" on Orders | — |
| Saving, field-error, "could not save" and "session ended" states themselves; the delete question's own layout | Built once, in item 10's `store.ts`, `ListPage.tsx`, `RecordPage.tsx`. This story proves they work on orders (SC-31 to SC-38) and does not rebuild them | "Manage shoes, lines, categories, reviews and orders" (plan item 10) |
| The customer's order page, "Pay now", the tracking page and the account's order list | Their own stories; this story proves only that the customer's page shows the courier's details and none of the private fields | Plan items 6, 7, 15 |
| The overview's figures ("To pack", "Latest orders") | Read the same table, built separately | "Dealer dashboard figures and charts" (plan item 11) |
| Owner-editable wording in the Studio for the new words | `src/content.json` is frozen for stories (`overview.md` §5); the new words live in this story's components and the existing keys (`pages.admin.orders.*`, `orders.steps.*`) keep coming from `content.json` untouched. A later chore can move them | — |

## 5. Decisions made here (the SRS leaves them open)

1. **The sentence names the stored status, not the customer's word.** The refusal reads "An order cannot go from placed to packed." with the stored words `placed`, `confirmed`, `packed`, `shipped`, `delivered`, `cancelled` — `shipped` is not written "On its way" there (SRS ADMIN-016 AC-2). The lists and the status control use the labels: Placed, Confirmed, Packed, On its way, Delivered, Cancelled.
2. **A field that may not change is refused whatever its value.** `paid`, `transaction_no`, `total` and the other fields of item 6 in section 3 are 422 when they are present in the body, even with the value the order already has (services §3.15 as written). The order page therefore sends only the ten fields the dealer can edit and never the whole record. If item 10's `saveRow` sends whole records, TL's design for this story names how the page limits what it sends; the server's rule does not loosen.
3. **All or nothing.** A body with one valid change and one refused field applies neither (`{"status":"confirmed","total":1}` leaves the order placed).
4. **The server is the judge of a move; the page only helps.** The status control offers the current status and the legal moves — placed: Confirmed, Cancelled; confirmed: Packed, Cancelled; packed: On its way, Cancelled; shipped: Delivered (no Cancelled); delivered and cancelled: none, drawn locked. A page that has gone stale and sends a move the shop refuses shows the shop's sentence beside Status (SC-32).
5. **A delivered order stays for good.** Only a cancelled order can be deleted and a delivered one cannot be cancelled, so a delivered order can never be deleted (SRS ADMIN-014 item 5 and open question 2). A shipped order that comes back is handled outside the screens.
6. **A paid order may be cancelled.** The pairs go back on sale; `paid` stays true; the order cannot then be deleted until the owner has refunded it in VNPay's portal by hand (ADMIN-016 conflict row; services §3.16). A late VNPay notice for a cancelled order leaves it cancelled and marks it paid (plan item 8).
7. **A change of phone changes how the customer finds the order.** `/track` compares the number typed to the phone stored, so after the dealer corrects a phone the new one is the one that works.
8. **"At once" means the next load.** The customer's order page shows new status, carrier, tracking and expected date the next time it is read; nothing is pushed.
9. **The expected date is typed as `2026-11-26`.** The field carries the hint "Year-month-day, for example 2026-11-26"; any other form is refused by the shop and its sentence shows beside the field.
10. **The "shoe" column shows the shoe's name**, the first pair's shoe (`line_no` 1), once, as the design does for a one-pair order.

## 6. UI scope

The approved design draws the orders list, the order page with its sentence and the tabs Order, Delivery, Customer, and the delete question (`design/design-system.md` §2.2; `src/pages/AdminOrders.tsx`, `AdminOrder.tsx`, `src/components/admin/*`). Those stay as they are. What the design never drew is built from its own parts (`Field`, `Button`, `Notice`, `Badge`, `Confirm`, `Panel`), tokens only, in the management palette.

| Screen / state | What it shows | SRS |
|---|---|---|
| List `/admin/orders` | Title "Orders", no "Add", search "Search by number or customer", filters Status and Payment, columns number, placed, status, shoe, total, payment, customer, a row menu "Edit" / "Delete", a row opens the order; now reads the shop, newest first | ADMIN-011 items 2–3, AC-12, AC-14; ADMIN-016 item 7 |
| Menu | The Orders item with the count of orders waiting | ADMIN-011 AC-15 |
| Order page `/admin/orders/:id` | Breadcrumb, header with "Delete" and "Save changes", the picture and sentence, tabs Order, Delivery, Customer, "Saved." after a save | ADMIN-011 item 4; ADMIN-016 |
| Order tab | Status control (only the moves the shop allows), then read-only: number, placed time, shoe and pairs, total, way to pay, "Paid" or "Awaiting payment", "VNPay transaction <number>" when there is one. **Not drawn:** the read-only block and the limited status control | ADMIN-016 items 1, 4 |
| Delivery tab | Delivery option (read-only), Carrier, Tracking number, Expected (with the hint). **Not drawn:** the hint, the refusal beside Expected | ADMIN-016 item 3 |
| Customer tab | Customer, Phone, Email, Address, City or province, Internal note — real values | ADMIN-016 items 5, 8 |
| Several pairs | Sentence "<customer> in <city> ordered <n> pairs on <date> — <total>." and under it one row per pair: picture, name, "EU <size>", quantity, unit price. **Not drawn in the approved design** | ADMIN-016 item 6, AC-16 |
| Delivered or cancelled order | Status, Delivery and Customer fields drawn read-only; the note still editable. **Not drawn** | ADMIN-016 items 3, 5, AC-14 |
| Refusals | The shop's sentence beside Status, Expected, Email and the other faulty field; "Cancel the order first." or the paid sentence inside the open delete question. **Not drawn** (the places come from item 10's plumbing) | ADMIN-011 items 6–8; ADMIN-014 item 5 |
| Record not here | "This record is not here", "It may have been deleted since the link was made.", "Back to the list" — only after the shop has answered | ADMIN-011 item 9 |

New words (English, as written, in this story's components): "Paid", "Awaiting payment", "VNPay transaction", "EU <size>", "<customer> in <city> ordered <n> pairs on <date> — <total>.", "Year-month-day, for example 2026-11-26". Money as `vi-VN`, dates as the design's `formatDateTime`.

Phone first: one column from 320 px, no horizontal page scroll at 390 px (a table may scroll inside its own container), tabs and "Save changes" reachable by thumb, each pair row wraps instead of cutting. Every field has a visible label; field errors, "Saving…" and "Saved." are announced to assistive technology.

**Mock for the UI stage.** The UI PR may use `src/lib/mock/dealer-handles-orders-status-courier-payment.ts` standing in for `PUT` / `DELETE` and the dealer reads (private fields, `items`, `transaction_no`), with on-demand answers: slow (2 s), 422 with field errors, 409, 401, unreachable. It must not write `localStorage`, and it must not touch `src/lib/mock/orders.ts` (the Track page's). The backend PR deletes it. Criteria marked **UI** are judged on the mock at the UI review; **BE** criteria are proved against the API; **UI+BE** in a browser against the running story.

## 7. Acceptance criteria

Forty criteria. Every criterion is checked against the running story: pages in a browser, endpoints with real sessions — an admin, a staff account, a customer row inserted into `users` for the test, and no session. Orders are placed through checkout or inserted as rows (for a paid or a cancelled-and-paid order, until plan item 8 can make them). "The dealer view" is `services.md` §2.8's order with `phone`, `email`, `address`, `note` and `transaction_no`; "the public view" is the one a token opens.

Tags: **BE** = proved against the API (Go tests with `TEST_DATABASE_URL`, and HTTP calls to the running story); **UI** = proved in the browser, judged at the UI review on the mock; **UI+BE** = proved in a browser against the running story.

### 7.1 Who may call, and what the dealer reads

- **SC-1 [ADMIN-004 AC-6, AC-7, AC-9, AC-10; ADMIN-016 AC-19]** (BE): With no session, and with a forged or expired `session` cookie, `PUT /v1/orders/<id>` and `DELETE /v1/orders/<id>` each answer 401 with the error envelope and no order data; a signed-in customer's session answers 403; in every case the order, its pairs and the shelf counts are as they were. A visitor's call for an id that does not exist is 401, not 404. An admin session and a staff session get the same answers as each other in every case below.
- **SC-2 [ADMIN-016 item 8; ADMIN-011 AC-20; ORD-008 AC-3, AC-7]** (BE): With an admin session and with a staff session, `GET /v1/orders/<id>` with no token answers 200 with the dealer view — `phone`, `email`, `address`, `note` and `transaction_no` besides the public fields — and, for an order of two pairs, `items` with both pairs in the order sent (`product`, `name`, `colourway`, `image`, `size`, `qty`, `unit_price`). `GET /v1/orders` carries the same five fields on every item, with `product`, `size` and `qty` of the first pair and no `items`. Both answer `Cache-Control: no-store`.
- **SC-3 [ORD-008 item 3, AC-1, AC-2, AC-5, AC-9; ADMIN-011 item 10]** (BE): No visitor or customer read carries those five fields: `GET /v1/orders/<id>` with the order's own token and no session answers 200 without `phone`, `email`, `address`, `note` or `transaction_no`; with no token and no session, and with the session of a customer the order is not linked to, it answers 404 with one and the same body as for an id that does not exist; `GET /v1/orders` answers 401 with no session and 403 for a customer; a dealer's `?phone=`, `?email=`, `?address=` or `?note=` filter answers 400.

### 7.2 Moving an order along

- **SC-4 [ADMIN-016 item 1, AC-1]** (BE): A placed order takes `PUT {"status":"confirmed"}` → 200 with status `confirmed`; then `packed`, `shipped`, `delivered`, each in its own `PUT`, each 200. After each, the read with the order's token shows the new status.
- **SC-5 [ADMIN-016 item 1, AC-2, AC-3, AC-4]** (BE): Each of these answers 422 with `errors.status` exactly "An order cannot go from <from> to <to>." (the stored words, for example "An order cannot go from placed to packed.") and leaves the order as it was: placed → packed, placed → shipped, placed → delivered, confirmed → placed, shipped → packed, delivered → shipped, delivered → placed, cancelled → confirmed. A status that is none of the six (`archived`) is 422 naming `status`.
- **SC-6 [ADMIN-016 item 1]** (BE): The current status sent again on a placed, confirmed, packed or shipped order is a no-op: 200 and nothing changes. On a delivered or cancelled order a `status` — even the current one — answers 422 "An order cannot go from <status> to <status>." and nothing changes.

### 7.3 Cancelling

- **SC-7 [ADMIN-016 item 2, AC-5; ORD-005 item 3]** (BE): A packed order of 2 pairs of one size, with 3 pairs left on the shelf after it was placed, takes `PUT {"status":"cancelled"}` → 200, status `cancelled`; the dealer's `GET /v1/products/<id>` then shows 5 pairs for that size and the shoe's `stock` 2 higher. For an order that took the last pair of a size: before the cancel the size is in the shoe's sold-out list and `GET /v1/products?size=<size>` does not contain the shoe; after it the size is in the last-pairs list and the shoe is in that result again.
- **SC-8 [ADMIN-016 item 2]** (BE): An order of two pairs — 1 pair of shoe A in size 38 and 2 pairs of shoe B in size 42 — cancels in one step: A's 38 count is up by 1, B's 42 count by 2, and no other count on the shelf changes. A placed order and a confirmed order cancel the same way.
- **SC-9 [ADMIN-016 item 2, AC-6]** (BE): Cancelling the same order again answers 422 "An order cannot go from cancelled to cancelled." and every count is as after the first cancel (the pairs are not returned twice). Two cancels of one order sent at the same moment, repeated 20 times with fresh orders, give exactly one 200 and one 422 each time, and the shelf has the pairs back once.
- **SC-10 [ADMIN-016 item 2, AC-7; open question 2]** (BE): A shipped order and a delivered order each answer a cancel with 422 "An order cannot go from shipped to cancelled." / "An order cannot go from delivered to cancelled."; the order and the shelf are unchanged.
- **SC-11 [ADMIN-016 item 2, Conflict row]** (BE): A paid order that is placed, confirmed or packed can be cancelled: 200, status `cancelled`, `paid` still true, its pairs back on the shelf. The dealer view still reads `paid` true.

### 7.4 Courier, customer details and the note

- **SC-12 [ADMIN-016 item 3, AC-8]** (BE): A shipped order takes `PUT` with `carrier` `GHN`, `tracking` `GHN-PT27XM` and `eta` `2026-11-26` → 200 with the three values; the next read with the order's token (the customer's order page) shows the same three. The same on a placed, a confirmed and a packed order. Only the fields sent change; `eta` `""` clears the date, and `carrier` and `tracking` `""` clear theirs.
- **SC-13 [ADMIN-016 item 3; services §2.5]** (BE): A `carrier` or `tracking` of 201 characters, and an `eta` of `26/11/2026`, `2026-13-40` or `tomorrow`, answer 422 naming the field and save nothing. On a delivered order and on a cancelled order, `carrier`, `tracking` and `eta` each answer 422 naming the field; the stored values are as they were.
- **SC-14 [ADMIN-016 item 5, AC-13]** (BE): A packed order takes one `PUT` changing `customer`, `phone`, `email`, `address` and `city` → 200, and the dealer view shows the five new values; a later `PUT` of only `city` changes only the city. An empty `email` is accepted.
- **SC-15 [ADMIN-016 item 5; services §2.5]** (BE): An empty `customer`, a `phone` of 8 digits, an `email` of `abc`, an `address` of 201 characters and a `note` of 5,001 characters each answer 422 with `errors` naming the field; two bad fields name both; nothing is saved. A `note` of exactly 5,000 characters is accepted.
- **SC-16 [ADMIN-016 item 5, AC-14]** (BE): On a delivered order and on a cancelled order, `customer`, `phone`, `email`, `address` and `city` each answer 422 naming the field (`address` among them) and the stored details are as they were; a `note` answers 200 and changes.

### 7.5 What never changes, and payment

- **SC-17 [ADMIN-016 item 5, AC-15]** (BE): A `PUT` carrying `number`, `placed_at`, `product`, `size`, `qty`, `items`, `total`, `delivery_fee`, `payment` or `delivery` answers 422 naming that field (several at once name all); nothing in the body is applied — `{"status":"confirmed","total":1}` leaves the order placed. An `id` in the body is ignored (the id in the address is the one changed); a field the order does not have (`colour`) and a body that is not JSON answer 400.
- **SC-18 [Owner decision; ADMIN-016 item 4, AC-11, AC-12]** (BE): A `PUT` carrying `paid` — true or false, on an unpaid order and on a paid one, on orders whose way to pay is `vnpay`, `cod`, `bank` and `momo` — answers 422 with `errors.paid` "Only the payment provider marks this order paid."; `paid` is as it was and nothing else in the body is applied. A `PUT` carrying `transaction_no` answers 422 naming it.
- **SC-19 [ADMIN-016 item 4, AC-9, AC-10]** (BE): An order marked paid by VNPay's signed notice with transaction `14123456` reads `paid` true and `transaction_no` "14123456" in the dealer view; an unpaid order reads `paid` false and `transaction_no` `""`; no `PUT` changes either. The number half is proved only once plan item 8's payment tables exist; until then it is UNVERIFIED and `transaction_no` is `""` for every order.

### 7.6 Deleting

- **SC-20 [ADMIN-014 item 5, AC-7]** (BE): `DELETE` of a placed, a confirmed, a packed, a shipped and a delivered order each answer 409 "Cancel the order first."; the order, its pairs, its tokens and the shelf counts are untouched.
- **SC-21 [ADMIN-014 item 5, AC-8; services §3.16]** (BE): `DELETE` of a cancelled, unpaid order answers 204; `GET /v1/orders/<id>` as a dealer and with the old token then answer 404; its pairs, tokens and payment attempts are gone; the shelf counts are not changed by the delete (the pairs came back at the cancel); a second `DELETE` is 404; the next order placed gets a number higher than the deleted one's.
- **SC-22 [ADMIN-014 item 5; services open question 5]** (BE): `DELETE` of a cancelled order that is also paid answers 409 "A paid order cannot be deleted; refund it in VNPay's portal first."; the order and its payment records are untouched.
- **SC-23 [ADMIN-016 Failure table; ADMIN NFR Performance]** (BE): `PUT` and `DELETE` of an id that does not exist, or that cannot be an id, answer 404 for a dealer; a `PUT` with `{}` answers 200 and changes nothing. 100 sequential `PUT`s and 100 sequential `DELETE`s of cancelled orders each answer within 1 s at p95.

### 7.7 The list and the menu

- **SC-24 [ADMIN-011 AC-12, AC-14, items 2–3]** (UI+BE): `/admin/orders` shows the title "Orders", **no "Add" button**, the search placeholder "Search by number or customer", the columns number, placed, status, shoe, total, payment, customer, and the filters Status (Placed, Confirmed, Packed, On its way, Delivered, Cancelled) and Payment (the ways to pay); with no sort chosen the newest order is first; an order placed through checkout is there on the next load; a row opens `/admin/orders/<id>`.
- **SC-25 [ADMIN-016 item 7, AC-20; ADMIN-011 AC-14]** (UI+BE): For an order of two pairs the "shoe" column shows the name of the first pair's shoe, once; a one-pair order shows its shoe. Status "On its way" keeps only shipped orders, Payment "VNPay" only VNPay orders, and a search for a customer's name or an order number finds the order.
- **SC-26 [ADMIN-011 AC-15]** (UI+BE): With two orders waiting (one placed, one confirmed) every admin page shows the menu's Orders item with the count 2; with none waiting no count is shown; cancelled, packed, shipped and delivered orders are not counted; after an order is saved as packed, a reload shows the count one lower.

### 7.8 The order page

- **SC-27 [ADMIN-016 item 8, AC-18; ADMIN-011 AC-20, item 10]** (UI+BE): `/admin/orders/<id>` has the tabs Order, Delivery and Customer; the Customer tab shows the customer, phone, email, address, city and note as stored. The same order opened as a customer (`/orders/<id>` with its token) and its `/track` result show none of the phone, email, address or note, in the page and in the network answers.
- **SC-28 [ADMIN-016 AC-17, item 6]** (UI+BE): An order of one pair shows the shoe's picture and the sentence of the design, "<customer> in <city> ordered <qty> × <shoe>, EU <size>, on <date> — <total>.", with no list of pairs.
- **SC-29 [ADMIN-016 AC-16, item 6; open question 4]** (UI+BE): An order of two pairs shows "<customer> in <city> ordered 2 pairs on <date> — <total>." and under it one row for each pair, in the order sent: picture, name, "EU <size>", quantity, unit price in `vi-VN` money. A pair whose shoe has since gone to draft still shows its name and picture; the unit prices times the quantities add up to the total.
- **SC-30 [ADMIN-016 item 4, AC-9, AC-10]** (UI+BE): The page shows the way to pay as plain text, "Paid" or "Awaiting payment" and, for an order VNPay confirmed, "VNPay transaction 14123456"; an unpaid order shows no transaction number. Number, placed time, shoe, size, pairs, total, way to pay, Paid, the transaction number and the delivery option are not inputs, switches, selects or buttons on any tab: the only form controls are status, carrier, tracking number, expected, customer, phone, email, address, city and note.

### 7.9 Saving and deleting from the page

- **SC-31 [ADMIN-016 item 1, AC-1; ADMIN-011 items 4–5]** (UI+BE): On a placed order the status control offers Placed (selected), Confirmed and Cancelled; after Confirmed and "Save changes", "Saved." shows and a reload — and the list — show Confirmed. The request carries only fields the dealer can edit (never number, placed, shoe, size, pairs, total, payment, paid, delivery). From confirmed the control offers Packed and Cancelled, from packed On its way and Cancelled, from shipped Delivered only.
- **SC-32 [ADMIN-016 AC-2, AC-3, AC-7; ADMIN-011 item 6]** (UI+BE): The page of a confirmed order is open in two windows; in one the dealer cancels it; in the other, choosing Packed and saving shows the shop's sentence "An order cannot go from cancelled to packed." beside Status with the typed values kept, and "We could not save. Please try again." is not shown.
- **SC-33 [ADMIN-016 AC-5, AC-14; items 2, 3, 5]** (UI+BE): Choosing Cancelled and saving shows "Saved."; the page then reads Cancelled, the status control, the Delivery fields and the Customer fields are read-only and only the note can be changed; the shoe's page shows the size back on the shelf (a size that was sold out is available again). A delivered order is locked the same way.
- **SC-34 [ADMIN-016 item 3, AC-8]** (UI+BE): On the Delivery tab the dealer types carrier `GHN`, tracking number `GHN-PT27XM` and expected `2026-11-26` (the hint "Year-month-day, for example 2026-11-26" is shown) and saves; after a reload they are there, and the customer's order page opened with the order's token shows the carrier, tracking number and expected date. An expected date typed as `26/11/2026` leaves the page open with the shop's sentence beside Expected and every typed value kept.
- **SC-35 [ADMIN-016 item 5, AC-13; ADMIN-011 item 6]** (UI+BE): On the Customer tab the dealer changes the address of a packed order and its note and saves; "Saved." shows and a reload shows both. An email `abc` leaves the page open with the shop's sentence beside Email — marked invalid and announced — with every typed value kept and nothing saved.
- **SC-36 [ADMIN-011 item 7, Failure rows; plumbing is plan item 10's]** (UI+BE): On an order page with the shop unreachable, "Save changes" shows "We could not save. Please try again." with the typed values kept and the button enabled; after the dealer's session has ended (the cookie removed in another tab) it shows "Your session ended. Sign in again." with a link to `/admin/sign-in`, and the typed values stay.
- **SC-37 [ADMIN-014 item 5, AC-7, AC-8; ADMIN-011 item 8]** (UI+BE): "Delete" on the order page and on a list row's menu asks "Delete <title>?" with "Cancel" and the confirm button, where <title> is the heading the page shows for the order; "Cancel" sends no request. Confirming on a placed order leaves the question open showing "Cancel the order first." and the order stays; on a cancelled, paid order it shows "A paid order cannot be deleted; refund it in VNPay's portal first."; on a cancelled, unpaid order the order is gone after a reload and the page returns to the list.
- **SC-38 [ADMIN-011 item 9, AC-10, AC-11]** (UI+BE): `/admin/orders/nope`, and the page of an order that was just deleted, show "This record is not here", "It may have been deleted since the link was made." and "Back to the list"; while the order is being fetched (2 seconds) "This record is not here" is not shown.
- **SC-39 [SRS Responsive and Accessibility NFRs; ADMIN-016 item 6]** (UI): At 390 px and at 320 px the list and the order page have no horizontal page scroll (a table may scroll inside its container); the tabs, the status control, "Save changes" and every pair row are reachable and readable; every field has a visible label.

### 7.10 Hygiene

- **SC-40 [Overview §5; ADMIN-011 item 1]** (UI+BE): After the backend PR, no file under `src/lib/mock/` is imported by `AdminOrders.tsx` or `AdminOrder.tsx` (`src/lib/mock/orders.ts` stays for the Track page); the page's client is `src/lib/dealer-handles-orders-status-courier-payment.ts` on `src/lib/api.ts`, with no second file with `api` in its name and no write to `localStorage`; the story's PRs do not edit `store.ts`, `ListPage.tsx`, `RecordPage.tsx`, `Form.tsx`, `fields.tsx`, `content.json`, `schema.json` or `theme.css`; `npm run lint`, `npm run build` and `npm test`, and `go build ./...`, `go vet ./...` and `go test ./...` all pass.

Also traced: ADMIN-011 AC-15's count and the list's Payment filter read the same orders as SC-24 to SC-26; ADMIN-016's Conflict row (a payment notice racing a cancel) is proved from the order side by SC-9 and SC-11 and from the notice's side by plan item 8.

## 8. Dependencies, and notes for the technical design step

**Must be on `main` first**

- **Plan item 10, "Manage shoes, lines, categories, reviews and orders"** — the write plumbing this story's page stands on (`store.ts`, `ListPage.tsx`, `RecordPage.tsx`: saving, field errors, "could not save", "session ended", the delete refusal inside the open question), and `src/lib/api.ts`'s `put` and `delete`. The UI stage of this story starts from item 10's merged files; the backend stage does not wait on item 10's backend except for `internal/stock` (below).
- **Plan item 5, "Checkout and placing an order"** — merged (PR #50): `internal/orders`, `order_items`, `product_sizes`, the view `product_stock`, `stock.Lock` / `stock.Take`.
- **Plan item 9, "Dealer sign-in and protected admin area"** — merged (PR #53): `auth.Require`, sessions.

**Soft dependency:** plan item 8, "Online payment with VNPay". Until its payment tables exist, `transaction_no` is `""` for every order and SC-19's number half is UNVERIFIED; SC-11 and SC-22 use order rows inserted with `paid` true (a real notice makes the same rows once item 8 has landed). Nothing else waits on it.

**Files other stories also edit — whichever merges second adapts**

- `internal/orders/read.go` (single read): plan item 6's backend (PR #65) extracts `authorize` and the access modes there, and item 15 adds the linked customer. This story adds the **dealer** mode, which answers the dealer view (services §3.11, "(1) a dealer session → the dealer view"). Item 6's `authorize` must keep the customer's `AccountID` in the read mode, as the notes on those items already say.
- `internal/orders/list.go`: its private columns become readable for a dealer; the `data.Column` access rule for `phone`, `email`, `address`, `note` and the new `transaction_no` is TL's design.
- `internal/stock/stock.go`: this story adds `Return`; item 10 adds `Set` and `SetOne`. Additive functions in one file.
- `orders.Cancel`: plan item 8's expiry ticker calls the same function. Whichever of the two merges first creates it as in services §3.15 (conditional status update, then `stock.Return` once per pair); the second adopts it.

**External accounts and secrets:** none new. No email or SMS is sent anywhere. VNPay's keys belong to plan item 8.

**Notes for TL's technical design (not binding where the design finds better)**

1. **Lock order.** `orders.Cancel` locks the order row first and then the counts it returns (`stock.Lock`, in (shoe, size) order). Checkout locks the advisory key, the shoes, the counts, the counter, then inserts (`orders.go`'s package comment). TL's design pins the order so that a cancel cannot deadlock with `Place`, with item 10's `SetOne`, or with a second cancel.
2. **Whole record versus fields.** `RecordPage`'s `saveRow` may send the whole record. The server refuses `total`, `number` and the rest when present (decision 2), so TL names how the page limits the body — the page's `fields` prop already restricts what the form shows, but not what `values` carries.
3. **Read-only drawing.** `FieldSpec` has no read-only flag and this story may not edit `fields.tsx` or `Form.tsx`. The page therefore passes only the editable fields in `fields` (a locked order passes `note` only) and draws the read-only facts in its own component beside the form.
4. **The dealer's record read.** `useRecord("orders", id)` reads `GET /v1/orders/{id}`; the answer must carry `items` and the five dealer fields (SC-2). The list's `product` is an id; the "shoe" column resolves its name the way other lists do.
5. **The mock** (section 6) is deleted by the backend stage; `src/lib/mock/orders.ts` is not.
6. **Order of work inside the story:** `stock.Return` and `orders.Cancel` → `Update` and `Delete` with their handlers → the dealer reads → the page.

## 9. Size

Forty criteria: 23 backend (SC-1 to SC-23), 16 browser (SC-24 to SC-39, of which SC-39 is judged on the mock) and 1 hygiene (SC-40). Every requirement this plan item owns appears in at least one criterion: ADMIN-016 items 1–8 and AC-1 to AC-20 (SC-4 to SC-19, SC-27 to SC-35; AC-19 in SC-1), ADMIN-004 AC-6 and AC-9 (SC-1), ADMIN-011 AC-12, AC-14, AC-15, AC-20 (SC-24 to SC-27), ADMIN-014 AC-7 and AC-8 (SC-20 to SC-22, SC-37). ADMIN-016 AC-11 and AC-12 sit in SC-18 as one rule with the owner's decision.
