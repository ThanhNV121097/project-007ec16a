# Story — Order page and Pay now

Module: `orders` · Plan item 6 · Requirements: ORD-007 (all 26 acceptance criteria), ORD-006 items 4–6 and AC-12, ORD-009 AC-5 (`docs/orders/SRS.md`)
Design system: `design/design-system.md` (§2.1.6 filled pill, §2.1.20 OrderSteps) · Architecture: `docs/architecture/overview.md`, `services.md` (§2.8 the order's public view, §3.11 `GET /v1/orders/{id}`, §3.13 `POST /v1/orders/{id}/payment`), `erd.md` (§3.12–3.14)

> Wording. The SRS writes the page's separators as `·` and `—` (`Order DOI-24818 · placed …`, `Thank you — your order is placed.`, `VNPay — ATM, Visa, Mastercard · paid`). The plan item's text writes them as ASCII hyphens only because it was typed that way; the SRS characters are the ones the page shows and the tests check.

## 1. User story

As a **customer** (a guest holding the order's token in this browser — no account needed), I want to open my order and see where it is, what is in it and where it goes, and press "Pay now" while it is still unpaid, so that I always know where I stand and can finish paying without placing the order again.

Other actors this story touches: **Stranger** (anyone without the token learns nothing: one "not found" view and one 404 body for a missing order, a missing token and another order's token) and **Payment provider** (not live yet: "Pay now" is built against the shop's `POST /v1/orders/{id}/payment`, which refuses with 409 until "Online payment with VNPay" makes the same call answer with an address). The shop team and a signed-in customer reading an order without a token arrive with their own stories (§4).

## 2. How this plan item was cut, and what exists today

"Checkout and placing an order" was cut at the line between **placing** and **reading** (its story §2). This item is the reading half, and it is built directly after that one: between the two merges `/orders/:id` is still the design's mock page.

**Exists when this item starts** (its dependencies, §8): `GET /v1/orders/{id}` answering the public view to the order's `X-Order-Token` and 404 to everything else; `src/lib/orders.ts` with the browser's store of placed orders (id, number, placed time, token) and the code that sends the browser to `/orders/<id>?placed=1`; the `orders`, `order_items` and `order_tokens` tables; `GET /v1/delivery` and `GET /v1/payments` (the records that name `standard` and `vnpay`); `src/lib/api.ts` with `credentials: "include"` and the error envelope as `ApiError`; `src/data/index.ts` on the API; the route `/orders/:id` in `App.tsx`; the shared `OrderSteps`, `SpecList`, `Disclosure`, `Bleed`, `ProductImage`, `Rule` and `OutlinePill` components.

**Today's `Order.tsx`** reads the design's mock (`src/lib/mock/orders.ts` and the `orders` record), looks the shoe up by `record.product` and draws one pair. That is what this story replaces.

## 3. In scope

**Backend (Go, PostgreSQL)**

- `POST /v1/orders/{id}/payment` as `services.md` §3.13 fixes it, for the part that exists before the VNPay story: no request body; access by the order's `X-Order-Token` (the same access check `GET /v1/orders/{id}` uses — written once, so the accounts and dealer stories extend one function). Checks run in this order: **access** (404, one body for a missing order, a malformed id, no token and another order's token) → **a body was sent** (400) → **state** (409: the order is paid; the order is cancelled; its way to pay is not an online one the shop completes) → **availability** (409 `conflict`, "Online payment is not available yet." — for every other order until the VNPay story). No payment attempt row is written and nothing is called outside the API.
- When "Online payment with VNPay" lands, the same route answers `200 {"payment_url": "…"}` (and 503 `payment_unavailable` when its keys are missing) **with no change to the Order page**: the page reads `payment_url` from a 200 answer and treats every other outcome as a failure (§6).

**Frontend (Vite + React + Tailwind tokens)**

- `src/pages/Order.tsx` rewritten from the mock to the API: reads `GET /v1/orders/{id}` with the token this browser holds for that id (header `X-Order-Token`, read from the store in `src/lib/orders.ts`), always with `credentials: "include"`; draws the page of ORD-007 for the six statuses; "Pay now" and its failure line; the not-found view only after the shop has answered; the loading and could-not-load states the design never drew.
- A new client file for this story, `src/lib/order-page-and-pay-now.ts` (the story-slug convention of overview §5), holding the order read and the "Pay now" call on top of `src/lib/api.ts`. It reads the token through the store's own reader in `src/lib/orders.ts`; it adds no second store, no second storage key and does not edit `src/lib/orders.ts`. Page parts that push `Order.tsx` past 300 lines go to their own components (for example `PayNow.tsx`).
- Clean-up: `Order.tsx` stops importing `src/lib/mock/orders.ts` and every mock-record read; the module itself is deleted in this PR **only if nothing imports it** — `Track.tsx` still does, so it stays, and "Track an order by number and phone" deletes it (§8). Words: "Free", never `0 ₫`; no "cash on delivery", "to pay on delivery" or "Express" anywhere on the page.

## 4. Out of scope — and where each piece is proven

| Not here | Why | Proven in |
|---|---|---|
| VNPay itself: building the payment address, the notice (IPN), `/payment/return`, the 30-minute expiry that returns pairs; the 503 `payment_unavailable` answer and the key check (ORD-007 item 9, last clause) | Needs the owner's VNPay keys and notice address. The page already treats a 503 like any other failure (SC-27) | "Online payment with VNPay" |
| A customer reading their own order with no token (ORD-008 AC-11, AC-12), and "Pay now" by the linked customer's session | No customer session exists until accounts land. This page already asks the shop with cookies and without the header when no token is held, so the accounts story needs no change to it | "Customer accounts: sign up, sign in, my orders" (its SC-10, SC-27) |
| The dealer or staff reading an order with the private fields (ORD-008 AC-2, AC-3, AC-7), the dealer's order screens, moving an order along, setting carrier, tracking number and expected date | No dealer session until dealer sign-in. In this story carrier, tracking, expected date and status are set directly in the test database | "Dealer sign-in and protected admin area", "Manage shoes, lines, categories, reviews and orders" |
| Looking an order up by number and phone, "Ordered on this phone", `Track.tsx`, deleting `src/lib/mock/orders.ts` when `Track.tsx` still imports it (ORD-020 to ORD-023) | Own plan item | "Track an order by number and phone" |
| Placing the order, the Checkout page, the token store and its writing, the idempotency key, `GET /v1/orders/{id}` itself and its privacy rules (ORD-008 AC-1, AC-4, AC-5, AC-6, AC-8, AC-9, AC-10) | Built and proven by the previous item (its SC-32, SC-35 to SC-37). This page only consumes them | "Checkout and placing an order" |
| Marking an order paid, by VNPay's notice or by hand; showing VNPay's transaction number; the cancelling of an unpaid order | Only the provider's notice marks paid (PAY-002); the dealer cannot (ADMIN-016 item 4) | "Online payment with VNPay", "Manage shoes, lines, categories, reviews and orders" |
| Changing a placed order's pairs or size; a text or email confirmation; the sentence "Check the number in your confirmation message" being true | The design's words stay (SRS open questions 3 and 4); nothing is built for them | The owner |
| Bank transfer (VietQR) and MoMo as ways to pay | Later P3 story; their rows stay in the records | "More ways to pay" |
| Theme tokens, `src/content.json`, `src/App.tsx`, `Track.tsx`, `Checkout.tsx`, `Product.tsx` | Frozen or owned by another story. The new words are listed in the PR as content keys for TL to add (§5) | — |

## 5. UI scope

The approved design's Order screen (moodboard `/orders/:id`: the answer first at h1, the number and date, the five steps as a hairline, the pairs with their tiles, disclosure rows) is the base. The work is replacing the mock with the API, drawing several pairs, and drawing the states the design left out. Tokens only (`bg-ground`, `text-ink`, `border-line` …), no arbitrary colours, no CSS modules, every file under 300 lines.

| Screen / state | What it shows | SRS |
|---|---|---|
| Order `/orders/:id`, loading | Nothing that looks like an answer: no heading, no not-found text. A quiet placeholder block (`aria-busy="true"`) in the page's place, the same treatment the data story gave the other pages when it has one | ORD-007 item 10, AC-24 |
| Order, found — one pair and several pairs, 1280 px and 390 px | Line `Order <number> · placed <date, time>`; h1; the line under it; five steps (one line from 640 px, a list below); the first pair's picture at the right (from 1024 px) linking to the shoe; "In this order" with every pair; Total; "Delivery and payment"; two closed rows; "Keep shopping" | ORD-007 items 1–7 |
| Order — six statuses | The h1 and line for placed, confirmed, packed, shipped, delivered and cancelled; with `?placed` the "Thank you — your order is placed." h1 instead | ORD-007 items 2–4 |
| Order — awaiting payment (not in the approved design) | Payment row `<way to pay> · awaiting payment` and the **"Pay now"** filled pill (design system §2.1.6, the page's one commit action; "Keep shopping" stays an outline pill) | ORD-007 items 6, 8 |
| Pay now — waiting | The button disabled (`disabled` attribute, 60 % opacity) and its label the progress sentence "Opening the payment page…", as the design system says a sending button does | ORD-007 item 8 |
| Pay now — failed (not drawn) | Under the button, in alert style with the alert icon and `role="alert"`: "We could not open the payment page. Please try again."; the button enabled again | ORD-007 failure row; ORD-006 AC-12 |
| Order — not found | h1 "We can't find that order.", the line "Check the number in your confirmation message, then look it up with the phone you ordered with." and a "Track an order" pill to `/track` (the design's existing view) | ORD-007 item 10 |
| Order — could not load (not drawn) | Neither order content nor the not-found view: one line "We could not load your order. Please try again." with a "Try again" button that asks the shop again | ORD-007 failure row |

**Copy and content keys.** Every new word is a content key the owner can edit, like the rest of the site: "Pay now", "Opening the payment page…", "We could not open the payment page. Please try again.", "We could not load your order. Please try again.", "Try again". `src/content.json` is frozen for stories (overview §5): the UI PR lists each new key under `pages.order.*` and TL says where it is added. The existing `pages.order.*` keys (headlines, `thanks`, `number`, `eta`, `done`, `paid`, `missing`, `missing_line`, `track_cta`, the closed rows) are reused as they are; `pages.order.due` reads "awaiting payment" after "Free delivery on every order".

**Hooks for tests.** Keep the existing `data-testid="order-status"` on the h1; add `pay-now` (the button), `pay-now-error` (the failure line), `order-missing` (the not-found view), `order-total` (the Total amount), `order-load-error` (the could-not-load line) — so no case depends on translated words alone.

## 6. Acceptance criteria

Each criterion names the SRS requirement and criterion it narrows (`SC-n [REQ AC-m]`). Test writes at least one case per `SC`; TL's reviews count this list. "Shop" means the running API; amounts are VND with `vi-VN` formatting; dates `dd/mm/yyyy`. Because no dealer session exists yet, an order's status, paid flag, carrier, tracking number and expected date are set directly in the test database; an order token for a sample order is a row inserted into `order_tokens` by the test (hash of a token the test knows).

### A. `POST /v1/orders/{id}/payment` (backend)

- **SC-1 [ORD-007 AC-20, permission row; ORD-006 item 4]** (BE): With no `X-Order-Token`, with another order's token, for an id that does not exist and for a malformed id, `POST /v1/orders/<id>/payment` answers 404 `not_found` with **one and the same body** (the body of `GET /v1/orders/<id>` for the same cases); no payment row is written and the order is unchanged.
- **SC-2 [ORD-007 AC-18, item 9]** (BE): For an order that is **paid**, called with its own token, it answers 409 `conflict` with a sentence saying the order is already paid; nothing changes.
- **SC-3 [ORD-007 AC-19, item 9]** (BE): For an order that is **cancelled** (paid or not), called with its own token, it answers 409 `conflict` with a sentence saying the order was cancelled.
- **SC-4 [ORD-007 AC-17, item 9]** (BE): For an unpaid, not cancelled sample order whose way to pay is not an online one the shop completes (`cod`, `bank` or `momo`), called with its own token, it answers 409 `conflict` with a sentence saying it cannot be paid online. The three sentences of SC-2 to SC-4 are three different sentences.
- **SC-5 [ORD-006 AC-12, items 4, 6; ORD-007 item 9]** (BE): For an order just placed through `POST /v1/orders` (VNPay, unpaid, status `placed`, VNPay story not landed), called with the token the placing answer returned, it answers 409 `conflict` with the message "Online payment is not available yet."; the order is still unpaid, no `payment_attempts` row exists for it and the API made no outbound call. The same answer follows a second and a third call.
- **SC-6 [services.md §3.13]** (BE): A call that carries a non-empty body, with the order's own token, answers 400 `bad_request` and changes nothing; a stranger sending a body still gets the 404 of SC-1 (access is checked first). With the database unreachable the route answers 500 `internal` with the generic envelope: no SQL, stack or other person's data.

### B. Finding the order, and what the page draws (frontend)

- **SC-7 [ORD-007 AC-1, items 1, 2; ORD-009 AC-1]** (UI): After placing order `doi-24818`, `/orders/doi-24818?placed=1` shows the h1 "Thank you — your order is placed." and the line `Order DOI-24818 · placed <date, time>` (the placed time as the site's one date-time format writes it, in Vietnam time). The same page without `placed` in the address shows the status h1 instead ("We have your order." for status `placed`).
- **SC-8 [ORD-007 AC-2, item 11; ORD-009 failure row]** (UI): A refresh of `/orders/doi-24818?placed=1` still shows the order: the token is read from this browser's store for that id. When the browser refuses storage, the page shows the order right after placing (the token lives in the page's memory) and, after a refresh, the not-found view of SC-9.
- **SC-9 [ORD-007 AC-3, item 10, permission and not-found rows]** (UI): In a browser holding no token, with another order's token, and for an id that does not exist, `/orders/<id>` shows the same view: the h1 "We can't find that order.", the line "Check the number in your confirmation message, then look it up with the phone you ordered with." and a "Track an order" pill to `/track`. The three cases are indistinguishable on the page (same text, same elements). With no token held the page still asks the shop once, with cookies and without the `X-Order-Token` header, and shows this view only after the shop's 404.
- **SC-10 [ORD-007 AC-24, item 10]** (UI): When the shop's answer to `GET /v1/orders/<id>` takes 2 seconds, then from the first paint until the answer arrives neither "We can't find that order." nor the order's content is on the page; when the answer arrives with a valid token the order appears. Also for an unknown id: the not-found view appears only after the 404 has come back.
- **SC-11 [ORD-007 failure row "shop unreachable"]** (UI): When `GET /v1/orders/<id>` cannot be reached (network failure) or answers 500, the page draws no order content and **not** the not-found view; it shows the line "We could not load your order. Please try again." with a "Try again" button, and pressing it asks the shop again and draws the order when the answer is good.
- **SC-12 [ORD-007 item 2, AC-4]** (UI): The h1 for each status is: placed "We have your order."; confirmed "Your order is confirmed."; packed "Your order is packed."; shipped "Your order is on its way."; delivered "Your order was delivered."; cancelled "This order was cancelled." There is exactly one h1 on the page.
- **SC-13 [ORD-007 AC-4, item 3]** (UI): An order of status `shipped` with expected date `2026-11-26` shows the h1 "Your order is on its way." and the line "Expected by 26/11/2026." The line is shown for placed, confirmed, packed and shipped orders that have an expected date.
- **SC-14 [ORD-007 AC-5, item 3]** (UI): An order of status `delivered` shows the h1 "Your order was delivered." and the line "Delivered. If your delivery is wrong or damaged, message us." (also when it has an expected date, which is then not shown).
- **SC-15 [ORD-007 AC-6, item 4, boundary row]** (UI): An order of status `cancelled` shows the h1 "This order was cancelled.", the five steps are replaced by "This order was cancelled. If you have a question about it, message us." with an alert icon, and no "Expected by" line. An order of any other status with an empty expected date shows no "Expected by" line.
- **SC-16 [ORD-007 AC-7, item 4]** (UI): An order of status `packed` shows five steps — Placed, Confirmed, Packed, On its way, Delivered — of which Placed, Confirmed and Packed are filled, Packed alone is ringed and carries `aria-current="step"`, and On its way and Delivered are not filled.
- **SC-17 [ORD-007 AC-8, item 4; Responsive NFR]** (UI): At 390 px the five steps are a vertical list; from 640 px they are one line. At 320 px and at 390 px the page does not scroll sideways, with a 100-character customer name and a two-pair order in place.
- **SC-18 [ORD-007 AC-26, item 1]** (UI): The first pair's picture is shown at the right of the heading from 1024 px (above the pairs below that) and links to that shoe's page `/shop/<id>`. A first pair with no picture draws no picture and no empty box, and the page holds together.

### C. The pairs, the total, the details (frontend)

- **SC-19 [ORD-007 AC-21, item 5; ORD-003 AC-11]** (UI): An order of two pairs lists both under "In this order", each with its name (a link to `/shop/<id>`), `EU <size> · Qty <n>` and its line price (`unit_price × qty`, as sold, not today's price); "Total" shows the order's **stored** total (not a sum computed by the page). A shoe that is now a draft still shows its name and links to its page.
- **SC-20 [ORD-007 AC-22, item 6; SITE-016]** (UI): For delivery `standard` (name "Standard", `days` "2–4 working days") and customer "Lan Nguyen", city "Da Nang", "Delivering to" reads `Lan Nguyen, Da Nang` and "Delivery" reads `Standard · 2–4 working days · Free`. The word "Free" is drawn, never `0 ₫`; the words "Express", "cash on delivery" and "to pay on delivery" appear nowhere. The names come from the shop's delivery and payment records: a raw id such as `standard` or `vnpay` is never drawn as a value, and the order is drawn only once the names it needs have arrived.
- **SC-21 [ORD-007 AC-11, AC-13, item 6]** (UI): A paid VNPay order shows "Payment" as `VNPay — ATM, Visa, Mastercard · paid`; an unpaid one shows `VNPay — ATM, Visa, Mastercard · awaiting payment`.
- **SC-22 [ORD-007 AC-9, AC-10, item 6]** (UI): With carrier `GHN` and tracking `GHN-PT27XM` set, "Carrier" shows `GHN` and "Tracking number" shows `GHN-PT27XM`; with both empty, neither row is drawn; with only the carrier set, only "Carrier" is drawn.
- **SC-23 [ORD-007 AC-23, item 7]** (UI): "Need to change something?" and "Wrong size?" are two rows closed on load; opened, the first shows "Send us the order number and what to change; the address or the size can change until the order is packed." and a "Message us" link to `/contact`, the second shows "Exchanges are not accepted. For a wrong or damaged delivery, or any question, message us." and a "Return policy" link to `/returns`. A "Keep shopping" pill goes to `/shop`.
- **SC-24 [ORD-007 AC-25; ORD-008 AC-5, AC-10]** (UI+BE): The order's address, phone, email and note appear nowhere on the page (not in its text, attributes or the answer the page received), whatever the order holds; and the order token appears only as the `X-Order-Token` header of the page's two requests — never in the address, query string, page text, link or `console` output.

### D. "Pay now" (frontend)

- **SC-25 [ORD-007 AC-13, AC-15, AC-16, item 8]** (UI): "Pay now" is shown for an unpaid order of status placed, confirmed, packed or shipped; it is not shown for a paid order (any status) and not for a cancelled order (paid or not).
- **SC-26 [ORD-007 AC-12; ORD-006 AC-12, item 4]** (UI+BE): With the VNPay story not landed, pressing "Pay now" on an unpaid order sends one `POST /v1/orders/<id>/payment` with the token and no body; the shop answers 409 "Online payment is not available yet." and the page shows, under the button, "We could not open the payment page. Please try again." in alert style (`role="alert"`). The page stays on the order, the browser has not moved, the button is enabled again and the line is gone when the next press starts.
- **SC-27 [ORD-007 failure row; ORD-006 item 5]** (UI): The same line and the same enabled button follow every other failure of the call: 503, 500, 404, a network error, a 200 answer with no `payment_url`, an empty one, and one that is not an `http://` or `https://` address.
- **SC-28 [ORD-007 AC-14, item 8; ORD-006 AC-7]** (UI): With the call answered `200 {"payment_url":"https://pay.example.test/x"}` (stubbed) the browser goes to exactly that address. While the answer is awaited — also when it takes 2 seconds — the button is disabled and reads "Opening the payment page…", and pressing it again sends nothing: exactly one `POST` leaves the browser. When the browser comes back to the page with its Back button the button is enabled again and reads "Pay now".
- **SC-29 [ORD-006 item 4; ORD-007 item 8, last sentence of item 9]** (UI+BE): "Pay now" does not depend on the VNPay story having landed: the page uses nothing but the status code and `payment_url`, so a shop that starts answering 200 with an address needs no change to the page (proven by SC-28's stub).

### E. The whole flow, and the clean-up

- **SC-30 [ORD-009 AC-5; ORD-006 AC-6]** (UI+BE, against the running stack): A visitor opens a real shoe, chooses an EU size that is in stock, adds the pair to the bag, opens `/checkout`, fills "Full name" `Lan Nguyen`, "Phone" `0912345678`, "Street address, ward, district" `12 Example Street, Ward 1`, "City or province" `Da Nang`, finds VNPay selected as the only way to pay, presses "Place order", and — the shop's `payment_url` being empty — lands on `/orders/<id>?placed=1` whose h1 contains "Thank you". On that page the number is the next `DOI-<n>`, "Payment" ends "awaiting payment", "Total" is the shoe's price, and "Pay now" is shown; the page text holds none of `0912345678`, `12 Example Street` or the token. This holds until "Online payment with VNPay" lands; that story re-proves the end of the flow with the payment page.
- **SC-31 [ORD-007 clean-up; overview §5]** (UI): `Order.tsx` imports nothing from `src/lib/mock/` and reads no `orders` or `products` record through the collection hooks (the order and its pairs come from `GET /v1/orders/{id}` alone); the only record reads left are the `delivery` and `payments` names. `src/lib/mock/orders.ts` is deleted in this PR if and only if no file imports it afterwards; while `Track.tsx` imports it, the module stays and the PR says so. The PR edits neither `Track.tsx`, `Checkout.tsx`, `App.tsx`, `src/lib/orders.ts`, `theme.css`, `index.css`, `content.json` nor `package.json`; `npm run lint` and `npm test` pass; the new words are listed as content keys in the PR description.
- **SC-32 [Accessibility NFR]** (UI): The page works with the keyboard alone: Tab reaches "Pay now", the two closed rows, the shoe links and "Keep shopping" in reading order, Enter or Space presses "Pay now", focus is visible; the failure line of SC-26 and the could-not-load line of SC-11 are announced to assistive technology.

## 7. Traceability

| Requirement | Criteria it owns | Criterion list |
|---|---|---|
| ORD-007 | AC-1 to AC-26 | AC-1 SC-7; AC-2 SC-8; AC-3 SC-9; AC-4 SC-12, SC-13; AC-5 SC-14; AC-6 SC-15; AC-7 SC-16; AC-8 SC-17; AC-9, AC-10 SC-22; AC-11 SC-21; AC-12 SC-26; AC-13 SC-21, SC-25; AC-14 SC-28; AC-15, AC-16 SC-25; AC-17 SC-4; AC-18 SC-2; AC-19 SC-3; AC-20 SC-1; AC-21 SC-19; AC-22 SC-20; AC-23 SC-23; AC-24 SC-10; AC-25 SC-24; AC-26 SC-18. Items 1–11 and the failure, permission and boundary rows: SC-5 to SC-32 |
| ORD-006 | items 4–6; AC-12 (AC-6, AC-7, AC-8 are re-proven at this page's edge) | SC-5, SC-26, SC-27, SC-28, SC-29, SC-30; **AC-10, AC-11 and the 503 half** are out of scope (§4) |
| ORD-009 | AC-5 | SC-30 (AC-1 SC-7 is the page's half of the previous story's SC-32) |
| ORD-008 | AC-5, AC-10 (consumed) | SC-24; **AC-11, AC-12, AC-2, AC-3, AC-7** are out of scope (§4) |

Criteria count: 32, well under the 40 a verification round is sized for.

## 8. Dependencies

**Must land first**

- **"Free delivery on every order, wording to match"** — it edits the mock `Order.tsx` (delivery detail ends "· Free", `pages.order.due` reads "awaiting payment", total = pairs total). This story rewrites that file, so it starts from the merged result and keeps those words.
- **"Site data from the database"** — `src/lib/api.ts`, `src/data/index.ts` on the API, the `delivery` and `payments` records over `GET /v1/delivery` and `GET /v1/payments`.
- **"Checkout and placing an order"** — `GET /v1/orders/{id}` with the token, `orders` / `order_items` / `order_tokens`, `src/lib/orders.ts` (the browser store and the landing at `/orders/<id>?placed=1`). Build and verify this story directly after it: without it `/orders/:id` has no real order to show and SC-30 cannot run.

**Lands after, and relies on this story**

- **"Online payment with VNPay"** — makes `POST /v1/orders/{id}/payment` answer 200 `{"payment_url"}` / 503, adds `payment_attempts`, `/payment/return` (whose "Try again" reuses the same route) and the expiry. It changes no line of `Order.tsx`.
- **"Track an order by number and phone"** — replaces `Track.tsx`'s read of the mock, then deletes `src/lib/mock/orders.ts` if this story could not.
- **"Customer accounts: sign up, sign in, my orders"** — its SC-27 opens this page for an order linked to the signed-in customer with no token; it adds the session path to the shared access check and edits no line of `Order.tsx`.

**External accounts, keys, data:** none are needed to build or verify this story; no VNPay key is requested here. Test data: the 14 seeded orders (all single-pair, with the sample ways to pay `momo`, `cod`, `bank`, `vnpay`) and orders placed through the real checkout; statuses, carrier, tracking, expected date and paid are set directly in the test database; sample orders get a token by inserting an `order_tokens` row.

## 9. Decisions recorded here (the SRS leaves them open)

- **Could not load.** The SRS says "nothing is drawn in place of the order" when the shop is unreachable, and "never the not-found view". The page draws no order and no not-found view, and one plain line with "Try again" so a customer is not left on a blank page (SC-11).
- **Pay now while waiting.** The button follows the design system's filled pill: disabled, label "Opening the payment page…" (SC-28).
- **Pay now for any unpaid, not cancelled order**, as the SRS says, including a seeded sample order whose way to pay is not online: the shop answers 409 and the page shows the failure line. Real orders are all VNPay, and the samples are placeholders the owner removes.
- **`?placed` is a flag, not a check.** The "Thank you" h1 is drawn whenever the address carries `placed`, as ORD-007 item 2 says, whatever the order's status.
- **The token is never in a URL.** The page asks with the `X-Order-Token` header only (services.md §3.11).
- **Where the token comes from.** The store in `src/lib/orders.ts` is the only reader; there is no `localStorage` key of this story's own.
- **For TL's technical design:** which package owns the route; the one shared "may this caller open this order" function behind `GET /v1/orders/{id}` and `POST …/payment`; the three 409 sentences of SC-2 to SC-4 (they must differ).
