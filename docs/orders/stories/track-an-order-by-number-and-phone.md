# Story — Track an order by number and phone

Module: `orders` · Plan item 7 · Requirements: ORD-020 (all 5 acceptance criteria), ORD-021 (all 12), ORD-022 (all 9), ORD-023 (all 7) (`docs/orders/SRS.md`)
Design system: `design/design-system.md` (§2.1.6 filled pill, §2.1.16 text field and label, §2.1.13 disclosure rows) · Architecture: `docs/architecture/overview.md`, `services.md` (§2.7 the lookup limits, §3.12 `POST /v1/order-lookups`, §3.11 `GET /v1/orders/{id}`, and the "number-and-phone matcher has one home" paragraph of the customer-accounts design), `erd.md` (`order_tokens`, `rate_limits`)

> Wording. The plan item's text names the route `POST /v1/orders/lookup`. `services.md` §3.12 fixed it earlier as **`POST /v1/order-lookups`**, and the customer-accounts story and its test cases already call that name, so this story uses it: same route, same body, same answer. If TL's technical design ever renames it, every criterion below keeps its meaning.
> The SRS writes "…" and "—" as the page shows them; the tests check the SRS characters.

## 1. User story

As a **customer** (a guest who ordered, or who ordered in another browser or lost this browser's record of it), I want to type my order number and the phone I ordered with and open my order, and to see on `/track` the orders this browser already holds, so that I can find out where my pair is without an account and without anyone else being able to look at my order.

Other actors this story touches: **Stranger** (anyone typing numbers and phones at random: every miss looks the same, and guessing is stopped after five tries) and **Shop team** (they do not use this screen; carrier, tracking number and expected date are set by them in later stories and only *shown* here).

## 2. How this plan item was cut, and what exists today

This is the "look it up again" half of the brief's *Order status tracking*. Placing the order and the Order page are plan items 5 and 6; this one is the door back into that page.

**Today's `Track.tsx`** (the approved design's page, never connected to a shop):

- reads the whole `orders` record — `useCollection("orders", { pageSize: 500 })` — into the browser and matches there, so every customer's phone would sit in the page's memory and in the network log;
- accepts a phone when only its **last four digits** match;
- builds "Ordered on this phone" from `src/lib/mock/orders.ts` (`usePlacedOrders`, the design's `doi:orders` key), not from the orders this browser really holds.

All three are replaced: the match moves to the shop, which answers with an id, a number, a time and a token and nothing else; the phone must match in full; and the list is read from the browser's store of order tokens that Checkout writes (`src/lib/orders.ts`, key `doi:placed`).

**Exists when this item starts** (its dependencies, §8): the `orders` and `order_tokens` tables with the seeded sample orders and their phones; `GET /v1/orders/{id}` answering the public view to an order's `X-Order-Token` and 404 to everything else; `src/lib/orders.ts` with the store and its reader/writer; `internal/ratelimit` and the `rate_limits` table whose scopes already include `order_lookup_number` and `order_lookup_address`; `src/lib/api.ts` with `api.post` (never retried); and the Order page that reads an order with a held token and draws the five steps, the cancelled sentence, carrier, tracking number and expected date (plan item 6).

## 3. In scope

**Backend (Go, PostgreSQL)**

- `POST /v1/order-lookups` as `services.md` §3.12 fixes it. Body `{"number","phone"}`; public; limited by `order_lookup_number` and `order_lookup_address`. The number is read as its digits (`DOI-24817`, `doi 24817`, `24817`); the phone as its digits with a leading `84` read as `0`, compared **in full**. On a right pair: `200` with `{"id","number","placed_at","token"}` and nothing else, `Cache-Control: no-store`; a **new** `order_tokens` row (stored hashed) is made on every success and earlier tokens stay valid. Every miss — wrong phone, unknown number, empty phone, a number with no digits — is the **same** `404 not_found` with the same body, and counts as one failure.
- The limit of ORD-022: 5 failures within 15 minutes for one number, or from one network address, and the sixth and later tries of that number or from that address are refused with `429 rate_limited` and `Retry-After` for 15 minutes counted from the fifth failure, whether or not they would have matched. A right pair neither counts nor clears; a refused try counts nothing and does not extend the wait; many tries at once let no more than five failures through. The order page is not affected by a lock (it never calls this route).
- The matcher is **one function** in `internal/orders` shared with the customer-accounts story's `POST /v1/account/orders` (that design names it `orders.Attempt` in `internal/orders/lookup.go`; whichever of the two backends merges first creates it, the second adopts it unchanged). This story's technical design reads that paragraph; it does not redesign it.

**Frontend (Vite + React + Tailwind tokens)**

- `src/pages/Track.tsx` rewritten from the mock to the shop: the form posts the typed number and phone to the lookup, and draws the states the design never drew — searching, not found, too many tries, offline, could not look that up — then, on a match, stores the order in the browser and goes to `/orders/<id>`.
- A client file for this story, `src/lib/track-an-order-by-number-and-phone.ts` (the story-slug convention of overview §5), on top of `src/lib/api.ts`. It holds the one call and the reading of its answers; no second client.
- "Ordered on this phone" built from the orders this browser holds a token for (the reader in `src/lib/orders.ts`): up to five, newest first, each its number and date linking to `/orders/<id>`; nothing drawn when there are none. A looked-up order is saved through the **same** store as a placed one (an additive export in `src/lib/orders.ts` if the store has no writer for it; no second store, no second storage key).
- Clean-up: `Track.tsx` stops importing `src/lib/mock/orders.ts` and stops reading the `orders` collection; the module `src/lib/mock/orders.ts` is deleted in this PR if no file imports it afterwards (the Order page story removed the other importer).
- The new words are content keys the owner can edit (§5).

**What the Order page shows after a lookup is not built here.** An order opened through the lookup is the same Order page as one just placed (ORD-023 item 2): its five steps, cancelled sentence, carrier, tracking number and expected date are item 6's work. This story proves the hand-off end to end (SC-24, SC-25), not the page's own drawing (item 6's SC-12 to SC-22).

## 4. Out of scope — and where each piece is proven

| Not here | Why | Proven in |
|---|---|---|
| Drawing the Order page: the five steps, cancelled sentence, carrier, tracking number, expected date, "Pay now", the not-found and could-not-load views | Built by the Order page story; this story only sends the browser there with a token | "Order page and Pay now" (its SC-7 to SC-32) |
| Placing an order, the order token's first minting, the browser store's writing at checkout, `GET /v1/orders/{id}` and its privacy rules | Built and proven by the Checkout story; consumed here | "Checkout and placing an order" |
| Adding a guest order to a signed-in account (`POST /v1/account/orders`), the lookup counters being shared by it, and "My orders" | Own plan item; it calls this story's matcher | "Customer accounts: sign up, sign in, my orders" (its SC-13 to SC-15) |
| The dealer moving an order along and setting carrier, tracking number and expected date; the dealer's order screens | In this story those fields are set directly in the test database | "Dealer sign-in and protected admin area", "Manage shoes, lines, categories, reviews and orders" |
| An email or text message with the order number or a code | Not in the brief or the plan; the words "confirmation message" on the page are the design's (SRS open question 4), nothing is built for them | The owner |
| A way to forget an order in this browser, or tokens that expire | Not in the SRS; see "Accepted for the owner" in the story reply | The owner |
| Looking an order up by phone alone, or by email | Not in the SRS; the lookup needs both number and phone | — |
| Changing a placed order; "Pay now"; VNPay | Other plan items | "Order page and Pay now", "Online payment with VNPay" |
| Theme tokens, `src/index.css`, `package.json`, `src/App.tsx` (the `/track` route already exists), `Order.tsx`, `Checkout.tsx` | Frozen or owned by another story | — |

## 5. UI scope

The approved design's Track screen (moodboard `/track`: "Track an order" at h1 and the two fields and the pill at the left, the five statuses as disclosure rows at the right, then "While you wait" with the shop's questions) is the base and stays as drawn. The work is replacing the mock with the shop, drawing the states the design left out, and making "Ordered on this phone" real. Tokens only (`bg-ground`, `text-ink`, `border-field`, `text-danger` …), no arbitrary colours, no CSS modules, every file under 300 lines.

| Screen / state | What it shows | SRS |
|---|---|---|
| Track `/track`, default (1280 px and 390 px) | h1 "Track an order"; the lead; the picture; fields "Order number" (placeholder `DOI-24817`) and "Phone" (`type="tel"`), each with its visible label; the filled pill "Find my order" with its arrow; "What each step means" as five closed rows; the compact promises; "Ordered on this phone" when this browser holds orders; "While you wait" with the line, the "Message us" pill and the questions of topic "orders" and "shipping" as closed rows | ORD-020 items 1–4 |
| Searching | The button disabled (`disabled` attribute, 60 % opacity) and its label "Searching…" (design system §2.1.6); the fields keep their values | ORD-021 item 7 |
| Not found | Under the fields, one alert line with the alert icon and `role="alert"`: "No order matches that number and phone. Check both, or message us." — the same line, the same elements, for a wrong phone, an unknown number, an empty phone and a number with no digits | ORD-021 item 4 |
| Too many tries (not drawn) | The same place and style: "Too many tries. Please wait 15 minutes, or message us with your order number." | ORD-022 item 3 |
| Offline (not drawn) | "You seem to be offline. Check your connection and try again." | ORD-021 failure row |
| Could not look that up (not drawn) | "We could not look that up. Please try again." (the shop unreachable, a 5xx answer, or any answer the page does not know) | ORD-021 failure row |
| "Ordered on this phone" | A hairline list, up to five rows, each `<number>` left and `<date>` right (`dd/mm/yyyy`, tabular figures) linking to `/orders/<id>`; the heading and list are absent when there are none | ORD-023 items 1, 3 |

At most one message is shown at a time; starting a new try clears the old one. Typed values stay after every failure. The page does no checking of its own before it sends (an empty phone is sent and the shop's 404 is what is shown), so nothing on the page tells a visitor which half was wrong.

**Copy and content keys.** The existing `pages.track.*` keys (`heading`, `lead`, `number`, `phone`, `placeholder`, `submit`, `not_found`, `steps_heading`, `steps.*`, `yours`, `faq_heading`, `help`, `contact_cta`) are reused as they are. Four words are new and become content keys under `pages.track.*` so the owner can edit them: "Searching…", "Too many tries. Please wait 15 minutes, or message us with your order number.", "You seem to be offline. Check your connection and try again.", "We could not look that up. Please try again." `src/content.json` is frozen for stories (overview §5); the PR lists the four keys and they are added as **new keys only** (existing keys untouched), by the one-time exception item 6 was given, in the PR TL names in the technical design.

**Hooks for tests.** Keep `data-testid="track-submit"` on the button; add `track-error` (the one alert line), `track-orders` (the "Ordered on this phone" list) — so no case depends on translated words alone.

## 6. Acceptance criteria

Each criterion names the SRS requirement and criterion it narrows (`SC-n [REQ AC-m]`). Test writes at least one case per `SC`; TL's reviews count this list. "Shop" means the running API. The seeded order `DOI-24817` (id `doi-24817`) has the phone `0938 410 552`. Because no dealer session exists yet, an order's status, carrier, tracking number and expected date are set directly in the test database. A "lock" is the state of the `order_lookup_*` counters; the test moves them back in time with the database instead of waiting 15 minutes. A "different address" is a different right-most `X-Forwarded-For` entry.

### A. `POST /v1/order-lookups` (backend)

- **SC-1 [ORD-021 AC-1, AC-9, item 1; Privacy NFR]** (BE): `POST /v1/order-lookups` with `{"number":"DOI-24817","phone":"0938 410 552"}` answers 200, `Content-Type: application/json`, `Cache-Control: no-store`, and a body whose keys are exactly `id` (`doi-24817`), `number` (`DOI-24817`), `placed_at` (a timestamp) and `token` (at least 22 URL-safe characters). The body has no `phone`, `email`, `address`, `note`, `customer`, `city`, `status`, `total` or `items`, and holds none of the order's private values as text.
- **SC-2 [ORD-021 AC-2, AC-3, items 2, 3]** (BE): The same order is found by each of these numbers — `DOI-24817`, `doi-24817`, `doi 24817`, `  DOI - 24817 `, `24817` — and each of these phones — `0938 410 552`, `0938410552`, `+84938410552`, `+84 938 410 552`, `84938410552`, `(0938) 410-552`. Every combination answers 200 with the same `id`.
- **SC-3 [ORD-021 AC-7, item 6]** (BE): A phone that is only part of the stored one — `552`, `410552`, `938410552` — a longer one that contains it, and one that differs in a single digit each answer the generic 404 of SC-4, never 200. A leading `84` is read as `0` only at the start of the phone: `0938 410 552` and `84 938 410 552` are the same phone, `938 410 552 84` is not.
- **SC-4 [ORD-021 AC-4, AC-5, AC-6, AC-8, item 4, Invalid-input row]** (BE): A wrong phone for an existing number, a number that does not exist (`DOI-99999`), an empty phone (`""`), a phone of only spaces, a number with no digits (`DOI-`, `""`) and a number over 18 digits all answer **404 `not_found` with one and the same status, headers that matter (`Content-Type`, `Cache-Control`) and body** — the same `message`, no `errors`, nothing that says which half was wrong or whether the number exists. No `Set-Cookie` and no `order_tokens` row is made by any of them.
- **SC-5 [ORD-021 item 1; ORD-008 AC-5]** (BE): The `token` of SC-1 opens the order: `GET /v1/orders/doi-24817` with it as `X-Order-Token` answers 200 with the public view and no `phone`, `email`, `address`, `note`; the same token on any other order answers 404 `not_found` with the body of an order that does not exist. Two successful lookups return two different tokens and both work; a token the order already held (from placing it) still works after a lookup. The database holds only a hash of each token: the token text appears in no table.
- **SC-6 [ORD-022 AC-1, AC-2, AC-9, items 1, 3]** (BE): Five wrong phones for `DOI-24817`, one after another, each answer the generic 404 of SC-4. The sixth try, even with the right phone, answers **429** `rate_limited` with a non-empty `message`, no `errors` key, a `Retry-After` header that is a whole number of seconds from 1 to 900 and no token; every later try, right or wrong, answers 429 as long as the lock lasts. No `order_tokens` row is made by a refused try.
- **SC-7 [ORD-022 AC-4, AC-5, AC-6, item 2]** (BE): (a) Five failures from one address across five different order numbers, then a lookup from that address for a sixth number, even a right pair: 429. (b) Five failures against `DOI-24817` from address A, then a right pair for it from address B: 429. (c) Five failures against a number that does not exist (`DOI-99999`), then a sixth try: 429 with the same status, header and body shape as in SC-6, so a locked number reveals nothing about whether it exists. The lock of one number does not stop a right pair for another number from another address.
- **SC-8 [ORD-022 AC-7, item 5]** (BE): Three wrong tries for `DOI-24817`, then a right pair (200), then two wrong tries (both 404): the next try, right or wrong, answers 429 — the right pair neither counted as a failure nor cleared the three. Four wrong tries, then right pairs, answer 200 each time: a success never counts.
- **SC-9 [ORD-022 AC-3, AC-9, item 4]** (BE): With `DOI-24817` locked, repeated refused tries do not move the lock: each `Retry-After` is less than or equal to the one before (it counts down; it never restarts at about 900). When the database says 14 minutes have passed since the fifth failure the answer is still 429 with `Retry-After` between 1 and 60; at 16 minutes a right pair answers 200 and a wrong one answers 404 and counts as the first failure of a new count.
- **SC-10 [ORD-022 Conflict row]** (BE): Ten wrong tries for one number sent at the same moment: exactly five answer 404 and the other five answer 429 (never six or more 404s), and the stored count for that number is five.
- **SC-11 [ORD-022 AC-8, item 6]** (BE): While `DOI-24817` is locked, `GET /v1/orders/doi-24817` with a token the order holds still answers 200 with the order. The lock affects `POST /v1/order-lookups` only.
- **SC-12 [services.md §3.12; ORD-021 failure rows]** (BE): A body that is not valid JSON, has a field that is not a string, or names a field the route does not define answers 400 `bad_request`; a body that is not JSON (`Content-Type: text/plain`) answers 415 `unsupported_media_type`; a body with a missing `phone` key is 400 while `"phone":""` is the 404 of SC-4. None of these makes a token. With the database unreachable the route answers 500 `internal` with the generic envelope: no SQL, stack or other person's data. Every non-2xx body is `{"error","message"}` (plus `errors` only on 400).

### B. The Track page: the lookup and its states (frontend)

- **SC-13 [ORD-021 AC-1, AC-12, items 1, 5]** (UI+BE, against the running stack): On `/track`, typing `DOI-24817` in "Order number" and `0938 410 552` in "Phone" and pressing "Find my order" sends exactly one `POST /api/v1/order-lookups` whose body is `{"number":"DOI-24817","phone":"0938 410 552"}` (as typed), then the browser is at `/orders/doi-24817` and the page's h1 contains "on its way". The pair's token is not in the address.
- **SC-14 [ORD-021 AC-2, AC-3, items 2, 3]** (UI+BE): The same order opens when `doi 24817` and `+84938410552` are typed, and when `24817` and `0938410552` are typed. The page does not rewrite what was typed before it sends it (the shop reads the digits).
- **SC-15 [ORD-021 AC-4, AC-5, AC-7, AC-8, item 4]** (UI+BE): Each of — `DOI-24817` with `0938 410 553`; `DOI-99999` with `0938 410 552`; `DOI-24817` with `552`; `DOI-24817` with an empty phone; empty fields — shows the same alert line "No order matches that number and phone. Check both, or message us." (`role="alert"`, alert icon, `data-testid="track-error"`) and the browser stays at `/track`. The text and the elements are identical for all five; both fields keep what was typed; the button is enabled again.
- **SC-16 [ORD-021 AC-11, item 7]** (UI): While the shop takes 3 seconds to answer, the button is disabled and reads "Searching…" and the typed values stay; a second press sends no second request (exactly one `POST` leaves the browser); when the answer is a refusal the button reads "Find my order" again and is enabled. Starting a try removes the previous message.
- **SC-17 [ORD-022 AC-2, item 3]** (UI+BE): After five wrong tries from the page a sixth — even with the right phone — shows "Too many tries. Please wait 15 minutes, or message us with your order number." in the same alert place and style, the browser stays on `/track`, the fields keep their values and the button is enabled. The 429 is read from the shop's status, not from its words.
- **SC-18 [ORD-021 Upstream-failure row]** (UI): With the browser offline (`navigator.onLine` false) pressing the button shows "You seem to be offline. Check your connection and try again." and sends nothing. When the shop cannot be reached while online (connection refused, a stopped API, a timeout) or answers 500, 502 or 503, the page shows "We could not look that up. Please try again." — neither the not-found line nor the too-many-tries line — the values stay and the button is enabled; the call is not retried by the client (one request per press).
- **SC-19 [ORD-021 AC-10, item 5; ORD-023 AC-4]** (UI): After a right pair the browser's storage holds an entry for the order with its id, number, placed time and token and **nothing else about it** — no phone, name, city, total or items — in the same store checkout writes (`doi:placed`), with no second key; reopening `/track` lists that order. Looking the same order up again leaves one entry for it (not two) and the newest token.
- **SC-20 [ORD-021 item 1; Privacy NFR]** (UI): Across a whole lookup — right pair, wrong pair, locked pair — no request the page makes has the phone or the number in its address or query string, no request lists orders (no `GET /api/v1/orders`, with or without a query), and the typed phone appears in no `localStorage` or `sessionStorage` entry and in no `console` output. The only request that carries the phone is the lookup's `POST` body. No answer the page receives holds any order's `phone`, `email`, `address` or `note`.

### C. "Ordered on this phone"

- **SC-21 [ORD-023 AC-1, AC-3, item 1]** (UI): With the browser holding tokens for six orders, `/track` shows "Ordered on this phone" with exactly five rows, newest placed first, each its number (tabular figures) and its date as `dd/mm/yyyy`; the sixth, oldest order is not listed. With no tokens the heading and the list are not drawn at all (no empty box, no "none yet" line). Building the list makes no request to the shop.
- **SC-22 [ORD-023 AC-2, item 3]** (UI): Pressing a row goes to `/orders/<id>` and that order shows, read with the token this browser holds for it. An order placed or looked up in another browser is not listed here; opening `/track` in a fresh browser draws no list even though the shop holds orders.
- **SC-23 [ORD-023 AC-7; ORD-009 Storage-refused row]** (UI): When the browser refuses storage, a fresh load of `/track` draws no list. A right lookup in that browser still opens the order for this page load; after a reload it is not listed.

### D. The order opened by a lookup

- **SC-24 [ORD-023 AC-5, item 2]** (UI+BE): The seeded order `DOI-24817` is set to status `shipped`, carrier `GHN`, tracking number `GHN-PT27XM`, expected date `2026-11-26`. A browser with no token looks it up from `/track` with the right pair and arrives at `/orders/doi-24817`: the h1 reads "Your order is on its way.", the five steps are drawn with "On its way" ringed, "Carrier" reads `GHN`, "Tracking number" reads `GHN-PT27XM`, and the line reads "Expected by 26/11/2026." The order's phone, email, address and note appear nowhere on the page.
- **SC-25 [ORD-023 AC-6, item 2]** (UI+BE): With the same order set to `cancelled`, the lookup opens the order and the page shows "This order was cancelled." in place of the steps and no "Expected by" line.

### E. The page around the form, and the clean-up

- **SC-26 [ORD-020 AC-1, AC-2, AC-3, AC-4]** (UI): `/track` still shows the h1 "Track an order", the lead "Type the order number from your confirmation message and the phone number you ordered with.", the picture, the two labelled fields (the first with placeholder `DOI-24817`) and "Find my order" with its arrow; the five rows Placed, Confirmed, Packed, On its way, Delivered, closed on load, each opening onto its sentence of the SRS (ORD-020 item 2); "While you wait" with its line, the "Message us" pill to `/contact`, and — when the shop has one question of topic "orders" and four of "shipping" — five closed question rows.
- **SC-27 [ORD-020 AC-5; Responsive NFR]** (UI): At 390 px the page is one column with the form above the step rows; at 320 px and 390 px there is no horizontal page scroll, also while the longest message ("Too many tries. Please wait 15 minutes, or message us with your order number.") and a six-order list are shown. From 1024 px the form is at the left and the step rows at the right as drawn.
- **SC-28 [Accessibility NFR]** (UI): Both fields have a visible label linked to them; pressing Enter in either field submits; Tab reaches the fields, the button and every list row in reading order with visible focus; every message of SC-15, SC-17 and SC-18 is announced to assistive technology (`role="alert"`), and the "Searching…" label is on the button itself.
- **SC-29 [overview §5; ORD-021 item 1]** (UI): `Track.tsx` imports nothing from `src/lib/mock/` and calls no collection hook on `orders` (it reads only the shop's `faqs` through the existing rows); the lookup goes through `src/lib/track-an-order-by-number-and-phone.ts` on `src/lib/api.ts` (`api.post`, no second client, no `fetch` of its own); the order is saved through the one store in `src/lib/orders.ts`; `src/lib/mock/orders.ts` is deleted if no file imports it afterwards. The PR edits none of `theme.css`, `index.css`, `package.json`, `App.tsx`, `Order.tsx`, `Checkout.tsx`; `content.json` gains only the four new `pages.track.*` keys; no file is over 300 lines; `npm run lint` and `npm test` pass; the PR lists the four keys.
- **SC-30 [ORD-021 AC-1, AC-12; ORD-009 AC-5]** (UI+BE, against the running stack): A customer places a real order through checkout as `Lan Nguyen`, phone `0912345678` (the order is the next `DOI-<n>`), then, in a **second browser with nothing stored**, types that number in lower case without the prefix and `+84 912 345 678` on `/track` and presses "Find my order": the order opens at `/orders/<id>` and shows the same number, total and "Delivering to Lan Nguyen, …" as in the first browser; the second browser's `/track` now lists it. A third browser typing the same number with `0912345679` sees the not-found line of SC-15.

## 7. Traceability

| Requirement | Criteria it owns | Criterion list |
|---|---|---|
| ORD-020 | AC-1 to AC-5 | AC-1, AC-2, AC-3, AC-4 SC-26; AC-5 SC-27. Item 4's "orders" and "shipping" rows are the shop's own questions, unchanged |
| ORD-021 | AC-1 to AC-12, items 1–7, failure rows | AC-1 SC-1, SC-13; AC-2 SC-2, SC-14; AC-3 SC-2, SC-14; AC-4 SC-4, SC-15; AC-5 SC-4, SC-15; AC-6 SC-4; AC-7 SC-3, SC-15; AC-8 SC-4, SC-15; AC-9 SC-1; AC-10 SC-19; AC-11 SC-16; AC-12 SC-13, SC-30. Item 5 SC-13, SC-19; item 6 SC-3; item 7 SC-16; Not-found and Invalid-input rows SC-4, SC-15; Upstream-failure row SC-12, SC-18 |
| ORD-022 | AC-1 to AC-9, items 1–6, Conflict row | AC-1 SC-6; AC-2 SC-6, SC-17; AC-3 SC-9; AC-4, AC-5, AC-6 SC-7; AC-7 SC-8; AC-8 SC-11; AC-9 SC-6, SC-9. Conflict row SC-10; item 5 SC-8 |
| ORD-023 | AC-1 to AC-7, items 1–3 | AC-1, AC-3 SC-21; AC-2 SC-22; AC-4 SC-19; AC-5 SC-24; AC-6 SC-25; AC-7 SC-23. Item 3 SC-22 |
| Non-functional | Privacy, Security, Responsive, Accessibility | Privacy SC-1, SC-20; Security SC-6 to SC-10; Responsive SC-27; Accessibility SC-28 |

Criteria count: 30, under the 40 a verification round is sized for. Twelve are backend-only (SC-1 to SC-12), so the route can be verified without a browser.

## 8. Dependencies

**Must land first**

- **"Site data from the database"** (done) — the `orders` table with the seeded sample orders and their phones, `src/lib/api.ts`, the `rate_limits` table (its scopes already include `order_lookup_number` and `order_lookup_address`) and `internal/ratelimit` (`Check`, `Fail`; merged with "Dealer sign-in and protected admin area").
- **"Checkout and placing an order"** (plan item 5) — `order_items` and `order_tokens`, the token minted and stored hashed, `GET /v1/orders/{id}` with the token rules (the lookup's token is the one this route reads with), `orders` taken out of the public records so `GET /v1/orders` is not a public list (SC-20 relies on it), and `src/lib/orders.ts` with the browser's store.
- **"Order page and Pay now"** (plan item 6) — the real `/orders/:id` that reads an order with a held token and draws the steps, the cancelled sentence, carrier, tracking number and expected date. SC-13, SC-22, SC-24, SC-25 and SC-30 need it; build and verify this story directly after it. It also removes `Order.tsx`'s import of `src/lib/mock/orders.ts`, which lets this story delete that module.

**Lands after, and relies on this story**

- **"Customer accounts: sign up, sign in, my orders"** — its `POST /v1/account/orders` calls the same matcher and the same counters (its SC-13 to SC-15). The matcher has one home (`internal/orders/lookup.go`, `Attempt`; `NumberKey`, `PhoneKey`): whichever backend merges first creates it as that design says, the second adopts it. It is never written twice.
- **"Manage shoes, lines, categories, reviews and orders"** — the dealer's setting of carrier, tracking number, expected date and status, which this story only shows.

**External accounts, keys, data:** none. No email or SMS service and no VNPay key are needed. Test data: the seeded sample orders (`DOI-24817`, phone `0938 410 552`) and orders placed through the real checkout. Because the limits count per address, the test run empties `rate_limits` before each case that tries a lookup, and moves the counters' times with the database to pass the 15 minutes; no backend switch for delay, failure or lock is added (project decision S4).

## 9. Decisions recorded here (the SRS leaves them open)

- **The route is `POST /v1/order-lookups`**, as `services.md` §3.12 and the accounts design say; the plan item's `POST /v1/orders/lookup` names the same call.
- **A refused request is not a try.** A body the route refuses with 400 or 415 is not a lookup and counts as no failure; only a well-formed lookup that misses counts (SC-12).
- **An absent field and an empty one differ.** A body without `phone` is 400; `"phone":""` is a miss (404) as ORD-021 AC-8 says. The page always sends both fields, so a visitor sees only the 404 line.
- **The page does no checking of its own.** An empty phone or number is sent and the shop's answer is shown, as ORD-021 AC-8 and its Invalid-input row say; the cost is one failure against the address.
- **The wait is worded, not read.** The 429 sentence says "15 minutes" whatever the remaining wait; the page does not parse `Retry-After`. The header stays for any other client and for the tests (SC-6, SC-9).
- **"Newest first" is by placed time**, the date each row shows, so the dates run downwards; a looked-up older order does not jump to the top (SC-21). Only the five newest are shown; the others stay stored.
- **The design's mock key is dropped.** Orders the design's mock kept under `doi:orders` are not read and not migrated; the list is read only from the orders store of `src/lib/orders.ts`.
- **A lookup is public to everyone.** A signed-in customer, a team member and a guest are treated alike; no session is read. Linking an order to an account is the accounts story's.
- **The sample orders are findable by their sample phones**, like any order. They are placeholders the owner removes with the rest of the first design's sample records.
- **Every success mints a new token** and earlier tokens stay valid (`erd.md` D5); none expires. That is the contract the Order page and "Ordered on this phone" are built on.
