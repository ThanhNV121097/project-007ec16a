# Test cases — Product page waits for the shoe before saying it is missing

Module: `site` · Plan item 13 (bug, P2) · Requirement: SITE-014 (with the not-found row of SITE-005)
Story: `docs/site/stories/product-page-waits-for-the-shoe-before-saying-it-i.md` — criteria SC-1 … SC-15
Shape: frontend only. The story adds no endpoint and no table. The four contract cases in section G read `GET /v1/products/{id}` (services.md §3.3), which already exists; the page depends on its 200 / 404 / 500 answers and nothing else. No request body is sent by this page, so there is no "undefined body field" case.

**Risk level: medium.** Nothing is written, no money moves and one file changes. But the shoe page is the busiest page of the shop and the defect is the first thing every visitor on a slow phone would read ("That pair is not in the shop any more." for a pair that is only loading), and the correct behaviour is four states that differ only in timing and in what the shop answered. So every row of the story's state table gets its own case, the two ways a stale record could leak between addresses get their own cases, and the two failure rows (404 and "cannot answer") are exercised at both a desktop and a phone width. Pure copy checks are one read each.

## How these cases are observed

- **Windows.** Default 1280×800. Phone cases say 390×844. The call/message dock (`[data-contact-dock]`) is drawn below 1024px only while a phone or Zalo number is set in the site's content; the shipped `contact.phone` and `contact.zalo` are both empty (SITE-003 AC-3), so the dock is absent as shipped. Cases therefore compare the dock on the shoe page with the dock on `/shop` at the same width ("as on any page") and never assume it is there.
- **Selectors.** Existing hooks only, no Tailwind class: `main` (the one element between the header and the footer: `<header>` … `<main><Outlet/></main>` … `<footer>`), `header`, `footer`, `[data-testid='product-name']` (the shoe's `h1`), `[data-testid='add-to-bag']`, `[data-testid^='size-']` (size pills; `size-38` is the `38` pill), `p[role='status']` (the stock line), `[data-testid='bag-count']`, `[data-testid='drawer-checkout']` (visible only while the bag drawer is open), `a[href='/shop']` (the back link or the pill), `section#reviews` ("What buyers said"), `a[aria-label='<shoe name>']` (a colour dot), `h1` and `a` read by their text. The missing view is `main > section` holding the `h1` "That pair is not in the shop any more." and the pill "All shoes".
- **The two words.** `MISSING` is exactly `That pair is not in the shop any more.` and `PILL` is exactly `All shoes`. Cases compare them word for word, case-sensitively, with the full stop.
- **Arrangement H(n, url) — hold a request.** The browser's request to the named URL is held for n seconds before it reaches the API (a route that delays only that URL and lets every other request through at once). H(2, shoe) means `GET /api/v1/products/<the id in the address>`. **Where a case says "hold 10 s", the hold is lengthened only so that a measuring tool, which loads the page first and reads afterwards, still reads inside the wait; the criterion is the same.** If no instrument can hold a single URL, every case that uses H is reported UNVERIFIED naming that missing capability (a per-URL delay), and the stubbed-hook rows of the story (TC-25) are then the only evidence of the waiting rows.
- **Readings.** "A reading" is one read of `main`'s visible text and of the elements the case names. "At the first paint" is the earliest reading the instrument can take after the navigation commits; "at 0.5 / 1 / 1.5 s" are readings taken that long after the navigation started, within the ±0.25 s the instrument allows. A case that says "polled" takes a reading at least every 250 ms.
- **Arrangement U — the shop cannot be reached.** The API container is stopped, so the gateway answers every `/api/…` request with 502 (a failed request as far as the page can tell). It is started again at the end of the case and a reload of `/shop` is checked to draw shoes.
- **Arrangement D — the database cannot be reached.** The API keeps running while its database refuses connections (`set_db_available` false). `GET /api/v1/products/<id>` then answers 500 `internal` (services.md §2.3: "a database that cannot be reached"; §3.3), never 200 and never 404. The database is put back at the end of the case.
- **Arrangement T — a draft shoe.** The row of `samba-og-kids-cloud-white` has its `status` set to `draft` (`set_db_state`), and is set back to `live` at the end of the case. A visitor (no session) is used throughout.
- **Seeded shoes the cases name.** `samba-og-w-crystal-white` — `Samba OG Crystal White`, price `3.090.000 ₫`, style code `IH9055`, line Samba, category Women, sizes `36 … 41 1/3` (nine), model `Samba OG`, three live reviews. `samba-og-w-wonder-quartz` — `Samba OG Wonder Quartz`, same model (a colour dot of the first). `samba-og-kids-cloud-white` — `Samba OG Kids Cloud White`. Before the data story lands these values are in `src/data/products.json`; once it has landed they are the API's seed, and a case that compares a price or a size list reads `GET /api/v1/products/<id>` first and compares to that.
- **Prices** are shown as `vi-VN` writes them (`3.090.000 ₫`); the space before `₫` may be a no-break space.
- **Unit-test cases** (TC-05, TC-25, TC-26) and the gate cases (TC-23, TC-24) are observed through `run_tests` (`npm test` and the lint/build that CI runs) and the pull request's diff. None of the five instrument names runs a vitest file or reads a diff, so they are labelled `manual` with that reason; the verifier reports what `run_tests` and `get_pr` returned.

## Ambiguities found in the documents (reported, not silently resolved)

1. **SC-3 needs the data story's hook.** Today `useRecord` returns `loading: false` for an id it cannot find, so a server-side render with the real hook of an unknown id contains the missing view. SC-3 (and TC-05) therefore hold only once "Site data from the database" has landed, as story §7 says. Until then TC-05 can only be run for a known id.
2. **SC-12 says "500"; the contract says a database that cannot be reached is `internal` 500, not `unavailable` 503** (503 is for `/healthz` and shutdown, services.md §2.3). TC-19 and TC-30 follow the contract. If the API answers 503 there, TC-30 fails against the contract (a finding for the data story), while the page rule of TC-19 — any non-404 failure draws nothing — is unchanged.
3. **SC-11 says "connection refused or the browser is offline".** The only way here to take the shop away is to stop the API, which gives a 502 from the gateway; a browser that is truly offline (a thrown network error) cannot be arranged. The page treats both as `error` set, and the stubbed `error` row of TC-25 is where a thrown error is represented. TC-17 is written for the 502; the offline half is not separately observed.
4. **SC-2 says "no 'All shoes' link of any kind" "between the header and the footer".** Cases read this inside `main`; the header menu and the footer keep their own links (a footer or menu link with the same words is not a failure).
5. **SC-13 asks for the address to change inside the page** (shoe → unknown id) with no reload. The only route in the interface that keeps the page mounted is a colour dot or a card, which cannot name an unknown id; TC-21 therefore changes the address with a history push and a `popstate` event, which the router treats as a navigation. If the instrument cannot do that, TC-21 is UNVERIFIED naming the missing capability (an in-page navigation to a typed address).
6. **The waiting and unreachable states leave `main` empty, so the footer moves up and then down** (story §5 "Known cost"). That is accepted by the story, so no case asserts a reserved height; TC-04 and TC-18 assert the opposite (the empty `main` has height 0) because that is what the story says is drawn.

---

## A. While the shop has not answered (SC-1 … SC-4)

**Scenario**: TC-01 — a shoe held for 2 seconds never shows the missing text while it waits (desktop)
**Given**: A 1280×800 window. Arrangement H(2, shoe) for `samba-og-w-crystal-white`; the shoe is live, so its answer after 2 s is the record.
**When**: `/shop/samba-og-w-crystal-white` is opened and readings of `main` are taken at the first paint and at 0.5 s, 1 s and 1.5 s.
**Then**: In each of the four readings the text `MISSING` is not on the page (searched in `document.body`'s text, not only in `main`), no `h1` has that text, and no `a` whose text is `PILL` exists inside `main`. (The shoe itself is not yet there either; that is TC-03.)
Traces: SC-1 (SITE-014 AC-1)
Check: interact_page

**Scenario**: TC-02 — the same wait on a phone shows no missing text
**Given**: A 390×844 window. Arrangement H(2, shoe) for `samba-og-w-crystal-white`.
**When**: `/shop/samba-og-w-crystal-white` is opened and readings are taken at the first paint and at 0.5 s, 1 s and 1.5 s.
**Then**: In each reading `MISSING` is not in `document.body`'s text and no `a` with the text `PILL` is inside `main`. After the answer (a reading at 2.5 s) `[data-testid='product-name']` reads `Samba OG Crystal White`.
Traces: SC-1 (SITE-014 AC-1; NFR phone first)
Check: interact_page

**Scenario**: TC-03 — during the wait nothing is drawn between the header and the footer
**Given**: A 1280×800 window. Arrangement H(2, shoe) for `samba-og-w-crystal-white`.
**When**: `/shop/samba-og-w-crystal-white` is opened and `main`, `header` and `footer` are read at 0.5 s, 1 s and 1.5 s.
**Then**: In each reading `main`'s trimmed text is the empty string and `main` contains no `h1`, no `a`, no `button` (so no price, no size pill, no "Add to bag", no "All shoes" link), and `[data-testid='product-name']` and `[data-testid='add-to-bag']` do not exist. `header` and `footer` exist and each has text (the header holds the shop's name and menu; the footer its columns).
Traces: SC-2 (SITE-014 items 1 and 3)
Check: interact_page

**Scenario**: TC-04 — on a phone the empty wait is empty, scrolls sideways by nothing, and keeps the dock the page has everywhere
**Given**: A 390×844 window. Arrangement H(10, shoe) for `samba-og-w-crystal-white` (hold 10 s, see "How these cases are observed"). `/shop` has been opened once in the same window and `[data-contact-dock]` read there (present or absent, and its `display`).
**When**: `/shop/samba-og-w-crystal-white` is opened and, 2 s later, `main`, `[data-contact-dock]` and the document are measured.
**Then**: `main`'s computed height is `0px`. `document.documentElement.scrollWidth` is at most 390. `[data-contact-dock]` is present with the same computed `display` as it had on `/shop` at 390 wide (and absent if it was absent there). `header` and `footer` both have a computed height greater than 0.
Traces: SC-2 (SITE-014 items 1 and 3; story §5 "Phone first")
Check: measure_styles

**Scenario**: TC-05 — the very first render, before any request is sent, contains neither the missing text nor the pill
**Given**: The data story has landed (ambiguity 1), so the real `useRecord` reports `loading: true` with `item` undefined for an id it has not yet asked about. The story's test file (or the verifier's own call of `renderToString`) renders `<Product />` with the real hook inside a router at `/shop/not-a-shoe`, and again at `/shop/samba-og-w-crystal-white`. A server-side render runs no effects, so no request has been sent.
**When**: The HTML string of each render is read.
**Then**: Neither string contains `That pair is not in the shop any more.` and neither contains an `a` whose text is `All shoes`; neither contains `data-testid="product-name"`.
Traces: SC-3 (SITE-014 AC-1)
Check: manual — a server-side render of the real hook is a vitest call; no listed instrument runs one. Observed through `run_tests` (`npm test`): the test that does this passes; if the story's test file has none, the verifier states that.

**Scenario**: TC-06 — pressing a colour dot whose answer takes 2 seconds shows neither the missing text nor the previous shoe (desktop)
**Given**: A 1280×800 window on `/shop/samba-og-w-crystal-white`, already drawn (its name, price `3.090.000 ₫` and three colour dots are on the page). Then arrangement H(2, shoe) is set for `samba-og-w-wonder-quartz` only.
**When**: The colour dot `a[aria-label='Samba OG Wonder Quartz']` is pressed and readings are taken at once and at 0.5 s, 1 s and 1.5 s, then at 2.5 s.
**Then**: In the readings before the answer `location.pathname` is `/shop/samba-og-w-wonder-quartz`; `MISSING` is not on the page; the text `Samba OG Crystal White` and the price `3.090.000 ₫` are not in `main` (the previous shoe is not shown under the new address); `[data-testid='product-name']` does not exist; `main`'s trimmed text is empty. In the reading at 2.5 s `[data-testid='product-name']` reads `Samba OG Wonder Quartz` and `MISSING` is still not on the page.
Traces: SC-4 (SITE-014 AC-1; SITE-005 AC-6)
Check: interact_page

**Scenario**: TC-07 — opening a related pair whose answer takes 2 seconds, on a phone, behaves the same
**Given**: A 390×844 window on `/shop/samba-og-w-crystal-white`, drawn. The first card under "Looking for more?" is read: its `href` (`/shop/<id>`) and its shoe's name. Arrangement H(2, shoe) is then set for that card's id only.
**When**: The card is scrolled into view and pressed; readings are taken at once and at 0.5 s, 1 s and 1.5 s, then at 2.5 s.
**Then**: Before the answer `location.pathname` is the card's `href`; `MISSING` is not on the page; `Samba OG Crystal White` is not in `main`; `main`'s trimmed text is empty. At 2.5 s `[data-testid='product-name']` reads the card's shoe name and `MISSING` is not on the page.
Traces: SC-4 (SITE-014 AC-1; story §3 "The record must be the one the address names")
Check: interact_page

---

## B. When the shoe arrives (SC-5 … SC-7)

**Scenario**: TC-08 — the shoe is drawn whole in the first paint after its answer, and the missing text was never there
**Given**: A 1280×800 window. Arrangement H(2, shoe) for `samba-og-w-crystal-white`.
**When**: `/shop/samba-og-w-crystal-white` is opened and the page is polled (every 250 ms or faster) from the navigation until 3 s after it, reading in every poll whether `MISSING` is in `document.body`'s text and whether `[data-testid='product-name']` exists.
**Then**: `MISSING` is in no poll at any time. In the first poll where `[data-testid='product-name']` exists (no earlier than about 1.75 s and no later than about 2.5 s after the navigation), the same poll also finds: the heading text `Samba OG Crystal White`; `a[href='/shop']` with the text `All shoes` inside `main`; a price `3.090.000 ₫`; at least nine `[data-testid^='size-']` pills; and `[data-testid='add-to-bag']`. There is no later poll in which one of these appears for the first time (no second wait).
Traces: SC-5 (SITE-014 item 4; SITE-005 AC-1)
Check: interact_page

**Scenario**: TC-09 — the shoe draws at its own answer; the parts with their own requests fill in when theirs arrive
**Given**: A 1280×800 window. The shoe's request is answered at 1 s; the requests for its line, category, shop terms, payment options, reviews and for the list of live shoes (`/api/v1/lines/…`, `/api/v1/categories/…`, `/api/v1/shop/…`, `/api/v1/payments…`, `/api/v1/reviews…`, `/api/v1/products?…`) are each held until 4 s (H(1, shoe) and H(4, each of the others)).
**When**: `/shop/samba-og-w-crystal-white` is opened and readings are taken at 1.5 s, 2.5 s and 3.5 s, then at 5 s.
**Then**: At 1.5 s, 2.5 s and 3.5 s: `[data-testid='product-name']` reads `Samba OG Crystal White`, the price `3.090.000 ₫`, the size pills and `[data-testid='add-to-bag']` are on the page; the style-code line under the name reads `IH9055` alone; `section#reviews` ("What buyers said") and the heading "Looking for more?" are not on the page; and `MISSING` is not on the page. At 5 s: the style-code line reads `Samba · Women · IH9055`, `section#reviews` and "Looking for more?" are on the page, three colour dots `a[aria-label='Samba OG Black Crystal']`, `a[aria-label='Samba OG Crystal White']` and `a[aria-label='Samba OG Wonder Quartz']` are on the page, and `MISSING` has not been on the page in any reading.
Traces: SC-6 (SITE-014 item 4)
Check: interact_page

**Scenario**: TC-10 — a shoe answered at once reads exactly as before
**Given**: A 1280×800 window; no request held. `GET /api/v1/products/samba-og-w-crystal-white` has been read (`fetch_url`) for its `sizes`, `sizes_low` and `sizes_out`.
**When**: `/shop/samba-og-w-crystal-white` is loaded and read after scripts run.
**Then**: `[data-testid='product-name']` reads `Samba OG Crystal White`. The line under it reads `Samba · Women · IH9055`. The price reads `3.090.000 ₫`. `p[role='status']` reads what SITE-005 AC-9 or AC-10 says for that record's own lists: with `sizes_out` empty and `sizes_low` `["36","39 1/3"]` (the seed) it reads `In stock in 9 sizes. Last pairs in 36, 39 1/3.`; with no `sizes_low` it would read `In stock in all 9 sizes.`. `MISSING` is not on the page, and `[data-testid='add-to-bag']` exists.
Traces: SC-7 (SITE-005 AC-1, AC-9, AC-10)
Check: render_url

**Scenario**: TC-11 — choosing size 38 and pressing "Add to bag" puts one pair in the bag and opens the drawer
**Given**: A 1280×800 window on `/shop/samba-og-w-crystal-white`, loaded, `localStorage` entry `agentteam:cart` removed and the page reloaded so the bag is empty (`[data-testid='bag-count']` reads `0` or is absent).
**When**: `[data-testid='size-38']` is pressed, `p[role='status']` is read, then `[data-testid='add-to-bag']` is pressed.
**Then**: After the size is pressed `[data-testid='size-38']` has `aria-checked` `true` and `p[role='status']` reads `In stock in EU 38. Sent today if you order before 15:00.` (the cut-off being the shop's `dispatch_cutoff`). After "Add to bag": `agentteam:cart` holds exactly one line, with `id` `samba-og-w-crystal-white`, `name` `Samba OG Crystal White`, `price` 3090000, `variant` `EU 38`, `qty` 1; `[data-testid='bag-count']` reads `1`; `[data-testid='drawer-checkout']` is visible (the drawer is open). With no size chosen first (a reload and "Add to bag" at once) the bag stays empty and `p[role='status']` reads `Choose a size first.`.
Traces: SC-7 (SITE-005 AC-11, AC-13, AC-14)
Check: interact_page

---

## C. When the shop says "not found" (SC-8 … SC-10)

**Scenario**: TC-12 — an unknown id shows nothing until the 404 arrives, then the missing view
**Given**: A 1280×800 window. Arrangement H(2, shoe) for the id `not-a-shoe` (the API answers 404 `not_found`, after the 2 s hold). No shoe has that id.
**When**: `/shop/not-a-shoe` is opened and readings are taken at the first paint and at 0.5 s, 1 s and 1.5 s, then at 2.5 s.
**Then**: In the four readings before the answer `MISSING` is not on the page, no `h1` exists in `main`, no `a` with the text `PILL` is in `main`, and `main`'s trimmed text is empty (the page never reads "no request sent yet" or "waiting" as "not found"). In the reading at 2.5 s `main` holds an `h1` whose text is exactly `That pair is not in the shop any more.` and an `a[href='/shop']` whose text is exactly `All shoes`; `[data-testid='product-name']`, `[data-testid='add-to-bag']`, every `[data-testid^='size-']` and `section#reviews` do not exist.
Traces: SC-8 (SITE-014 AC-2); SC-3 (SITE-014 AC-1, the first reading)
Check: interact_page

**Scenario**: TC-13 — an address the API rejects as not a valid id also ends in the missing view
**Given**: A 1280×800 window; no request held. The id `Not-A-Shoe` (upper case) does not match the API's id pattern, so the API answers 404 for it (services.md §3.3).
**When**: `/shop/Not-A-Shoe` is loaded and read after scripts run.
**Then**: `main` holds the `h1` `That pair is not in the shop any more.` and the pill `All shoes` linking to `/shop`; no `[data-testid='product-name']` exists. (Not a blank page, and not the text of a failed request.)
Traces: SC-8 (SITE-014 AC-2); contract (GET /v1/products/{id}, id pattern)
Check: render_url

**Scenario**: TC-14 — the missing view holds one pill to `/shop`, no shoe parts, and the pill works
**Given**: A 1280×800 window on `/shop/not-a-shoe` after its 404 has been answered and the missing view is on the page.
**When**: `main` is read, then the pill (`main a` with the text `All shoes`) is pressed.
**Then**: Before pressing: `main` holds exactly one `a` whose text is `All shoes` (a count of 1), its `href` attribute is exactly `/shop`; there is no price (no text matching `\d[\d.]*\s?₫`), no `[data-testid^='size-']`, no `[data-testid='add-to-bag']`, no `section#reviews`, no "Looking for more?" and no "What buyers said" in `main`. After pressing: `location.pathname` is `/shop` and the shop page's own heading is on the page.
Traces: SC-9 (SITE-014 AC-2; SITE-005 not-found row)
Check: interact_page

**Scenario**: TC-15 — on a phone the missing view is one column and nothing scrolls sideways
**Given**: A 390×844 window on `/shop/not-a-shoe` after its 404 has been answered (no hold).
**When**: The missing view (`main > section`), its `h1` and its pill (`main a[href='/shop']`) and the document are measured.
**Then**: The section's computed `flex-direction` is `column`; the pill's top edge is at or below the `h1`'s bottom edge and the pill's left edge equals the `h1`'s left edge (heading, then pill, in one column); `document.documentElement.scrollWidth` is at most 390.
Traces: SC-9 (story §5 "Phone first"; SITE-005 not-found row)
Check: measure_styles

**Scenario**: TC-16 — a draft shoe's address is answered 404: nothing first, then the missing view, never the shoe
**Given**: A 1280×800 window. Arrangement T (`samba-og-kids-cloud-white` set to `draft`), a visitor with no session, and arrangement H(2, shoe) for that id.
**When**: `/shop/samba-og-kids-cloud-white` is opened and polled from the navigation until 3 s after it; then the shoe is set back to `live`, the hold is removed and the page is reloaded.
**Then**: While draft: in every poll `[data-testid='product-name']` does not exist and the text `Samba OG Kids Cloud White` is not in `document.body`'s text; in the polls before about 2 s `MISSING` is not on the page and `main`'s text is empty; in the polls after the answer `main` holds the `h1` `That pair is not in the shop any more.` and the pill `All shoes`. After it is set back to `live` and reloaded: `[data-testid='product-name']` reads `Samba OG Kids Cloud White` and `MISSING` is not on the page.
Traces: SC-10 (SITE-014 AC-3; ADMIN-013 AC-11)
Check: interact_page

---

## D. When the shop cannot answer (SC-11, SC-12)

**Scenario**: TC-17 — with the shop unreachable the page stays empty and never says the pair is gone (desktop and phone)
**Given**: Arrangement U (the API stopped; the gateway answers 502 to every `/api/…` request). A 1280×800 window, then the same case at 390×844.
**When**: `/shop/samba-og-w-crystal-white` is opened and readings are taken at the first paint and at 1 s, 2 s, 3 s, 4 s and 6 s (the data module may retry a failed read once before it gives up; the readings span both).
**Then**: In every reading, at both widths, `MISSING` is not in `document.body`'s text; no `a` with the text `PILL` exists in `main`; no `h1`, `[data-testid='product-name']`, `[data-testid^='size-']` or `[data-testid='add-to-bag']` exists; `main`'s trimmed text is empty; `header` and `footer` exist and have text. The page does not redirect (`location.pathname` is still `/shop/samba-og-w-crystal-white`).
Traces: SC-11 (SITE-014 AC-4)
Check: interact_page

**Scenario**: TC-18 — the unreachable state adds nothing that could scroll on a phone
**Given**: Arrangement U. A 390×844 window.
**When**: `/shop/samba-og-w-crystal-white` is opened and, 6 s later, `main` and the document are measured.
**Then**: `main`'s computed height is `0px`; `document.documentElement.scrollWidth` is at most 390.
Traces: SC-11 (SITE-014 AC-4; story §5 "Phone first")
Check: measure_styles

**Scenario**: TC-19 — the shop answering with a failure (500) behaves like the shop being away
**Given**: Arrangement D (the API running, its database refusing connections); `GET /api/v1/products/samba-og-w-crystal-white` answers 500 `internal` (read once with `fetch_url` to prove the arrangement). A 1280×800 window.
**When**: `/shop/samba-og-w-crystal-white` is opened and readings are taken at the first paint and at 1 s, 2 s, 4 s and 6 s.
**Then**: In every reading `MISSING` is not on the page, no `a` with the text `PILL` exists in `main`, no `[data-testid='product-name']` exists, `main`'s trimmed text is empty, and `header` and `footer` have text. After the database is put back and the page is reloaded, `[data-testid='product-name']` reads `Samba OG Crystal White`.
Traces: SC-12 (SITE-014 AC-4; DATA-005 failure row)
Check: interact_page

---

## E. One address does not leak into the next (SC-13)

**Scenario**: TC-20 — after a 404, opening a live shoe draws it with no missing view, even while that shoe's answer is slow
**Given**: A 1280×800 window. Arrangement H(2, shoe) for `samba-og-w-crystal-white` is set after the first step below.
**When**: `/shop/not-a-shoe` is opened and its missing view is seen; its pill `All shoes` is pressed (`/shop` opens); the card of `Samba OG Crystal White` is pressed; readings are taken at once and at 0.5 s, 1 s and 1.5 s after the card is pressed, then at 2.5 s.
**Then**: Before the answer `location.pathname` is `/shop/samba-og-w-crystal-white`, `MISSING` is not on the page and `main`'s trimmed text is empty (the earlier 404 has left nothing behind). At 2.5 s `[data-testid='product-name']` reads `Samba OG Crystal White` and `MISSING` is not on the page.
Traces: SC-13 (SITE-014 AC-1, AC-2)
Check: interact_page

**Scenario**: TC-21 — after a drawn shoe, an unknown id in the same page shows nothing, then the missing view only after its answer
**Given**: A 1280×800 window on `/shop/samba-og-w-crystal-white`, drawn (ambiguity 5). Arrangement H(2, shoe) is set for `not-a-shoe` (404 after 2 s).
**When**: The address is changed inside the page, with no reload, to `/shop/not-a-shoe` (a history push and a `popstate` event); readings are taken at once and at 0.5 s, 1 s and 1.5 s, then at 2.5 s.
**Then**: Before the answer `location.pathname` is `/shop/not-a-shoe`, `MISSING` is not on the page, the text `Samba OG Crystal White` and the price `3.090.000 ₫` are not in `main`, `[data-testid='product-name']` does not exist and `main`'s trimmed text is empty. At 2.5 s `main` holds the `h1` `That pair is not in the shop any more.` and the pill `All shoes`, and no shoe part.
Traces: SC-13 (SITE-014 AC-1, AC-2)
Check: interact_page

**Scenario**: TC-22 — after an unreachable shop, a shoe whose answer comes normally is drawn
**Given**: A 1280×800 window. Arrangement U for the first step; the API is started again before the second.
**When**: `/shop/samba-og-w-crystal-white` is opened with the API stopped and `main` is seen to stay empty for 3 s; the API is started again; the header's menu link to the shop (`header a[href='/shop']`) is pressed (an in-page navigation, no reload) and the card of `Samba OG Crystal White` is pressed.
**Then**: While stopped `MISSING` is not on the page and `main`'s text is empty. After the API is back and the card is pressed, `[data-testid='product-name']` reads `Samba OG Crystal White`, the price reads `3.090.000 ₫` and `MISSING` is not on the page in any reading taken from the press until the shoe is drawn.
Traces: SC-13 (SITE-014 AC-4, AC-1)
Check: interact_page

---

## F. Boundaries of the change (SC-14, SC-15)

**Scenario**: TC-23 — the change set is `Product.tsx` and one test file, and nothing else
**Given**: The pull request for this story.
**When**: Its changed files and the added lines of `Product.tsx` are read (`get_pr`).
**Then**: The changed files are exactly `code/frontend/src/pages/Product.tsx` and one new file under `code/frontend/src/test/`. No file under `src/data/` or `src/components/` is changed, and neither is `src/App.tsx`, `src/content.json`, `src/theme.css` or `src/index.css`; `content.json` gains no key. The lines added to `Product.tsx` contain no `#hex`, `rgb(`, `hsl(` or `[…]` arbitrary-value class and no new user-visible string (the two strings drawn in the missing view are still `pages.product.missing` and `pages.product.back`, read through `<T k=…>`).
Traces: SC-14 (SITE-014 items 1–4; architecture §5)
Check: manual — a property of the diff; no running-site instrument can see which files a pull request touches. Read from `get_pr`.

**Scenario**: TC-24 — `Product.tsx` stays within the lint gates and lint, build and tests pass
**Given**: The pull request's head.
**When**: `run_tests` runs the frontend suite, and the CI logs of the pull request (`get_ci_logs`) are read for `npm run lint` and `npm run build`.
**Then**: `npm test` passes; `npm run lint` reports no error — in particular `max-lines` (300 lines of code per file, blank lines and comments not counted) and `complexity` (20 per function) hold for `Product.tsx`; `npm run build` (`tsc --noEmit && vite build`) passes.
Traces: SC-14 (SITE-014 items 1–4)
Check: manual — lint and build are CI jobs and `npm test` is a vitest run; none of the five instruments runs either. Observed through `run_tests` and the CI logs.

**Scenario**: TC-25 — the test file covers each row of the state table with the hook stubbed
**Given**: The new file under `code/frontend/src/test/` and `run_tests` output.
**When**: The file is read and its tests are run.
**Then**: With `useRecord("products", id)` stubbed (the rest of `../data` returning empty lists) and `Product` rendered inside a router at `/shop/<id>`, there is one test per row and each asserts which of the shoe and the missing view is on the page: (1) a record for the address's id with `loading: false, error: null` → `[data-testid='product-name']` shows the shoe's name and `That pair is not in the shop any more.` is absent, and the same record with `loading: true` (and again with `error` set) still draws the shoe; (2) `item` undefined, `loading: true` → neither the shoe nor the missing text nor the pill is on the page; (3) `item` undefined, `loading: false`, `error` an `Error` → the same empty result; (4) `item` undefined, `loading: false`, `error: null` → the `h1` `That pair is not in the shop any more.` and an `a[href='/shop']` with the text `All shoes`, and no `[data-testid='product-name']`. All tests pass.
Traces: SC-15 (SITE-014 AC-1 to AC-4)
Check: manual — reading a test file and its vitest result; no listed instrument runs `npm test`. Observed through `run_tests` and `get_pr`.

**Scenario**: TC-26 — the loading and error tests would fail against today's page
**Given**: `Product.tsx` as it is on the base commit (it draws the missing view whenever `useRecord(...).item` is undefined, whatever `loading` and `error` say) and the new test file.
**When**: The loading test (2) and the error test (3) of TC-25 are read against the base page's `if (!item) { return <missing view> }`.
**Then**: Each of the two tests asserts that `That pair is not in the shop any more.` is not on the page while `item` is undefined; against the base page that assertion fails, because the base page draws the `h1` for any undefined `item`. The record-in-hand test (1) and the answered-not-found test (4) pass against both versions.
Traces: SC-15 (SITE-014 AC-1, AC-4)
Check: manual — the instruments run the branch's code only; none checks out the base commit and runs the new tests against it. Established by reading the base `Product.tsx` (from the pull request's diff) against the test assertions.

---

## G. The API the page reads — `GET /v1/products/{id}` (services.md §3.3)

These four cases do not belong to this story's own change; they pin the answers the page branches on, so that a wrong status from the data story's API is named as the cause rather than blamed on the page. They run against the running API after the data story has landed.

**Scenario**: TC-27 — a live shoe is answered 200 with its record
**Given**: The API is running with the seeded shoes.
**When**: `GET /api/v1/products/samba-og-w-crystal-white` is fetched with no session.
**Then**: Status 200, `Content-Type` JSON; the body is one object (not wrapped in `items`) with `id` `samba-og-w-crystal-white`, `name` `Samba OG Crystal White`, `price` 3090000, `code` `IH9055`, `status` `live`, and `sizes` an array of the nine EU sizes starting `36`, `36 2/3`.
Traces: contract (GET /v1/products/{id})
Check: fetch_url

**Scenario**: TC-28 — an unknown id and an id outside the pattern are answered 404 with the error envelope
**Given**: The API is running; no shoe has the id `not-a-shoe`.
**When**: `GET /api/v1/products/not-a-shoe` and `GET /api/v1/products/Not-A-Shoe` are fetched with no session.
**Then**: Both answer status 404 with a JSON body whose `error` is `not_found` and whose `message` is a sentence; the body has no `errors` key (it exists only on 400/422), no stack trace and no SQL; the two bodies are identical to each other.
Traces: contract (GET /v1/products/{id}); SC-8 (SITE-014 AC-2)
Check: fetch_url

**Scenario**: TC-29 — a draft shoe is answered 404 with exactly the body of an unknown id
**Given**: Arrangement T (`samba-og-kids-cloud-white` is `draft`); no session.
**When**: `GET /api/v1/products/samba-og-kids-cloud-white` and `GET /api/v1/products/not-a-shoe` are fetched; then the shoe is set back to `live` and the first is fetched again.
**Then**: While draft the first answers 404 with `error` `not_found` and a body byte-identical to the second's (a visitor cannot tell a draft from an unknown id); after it is set back to `live` the first answers 200 with `name` `Samba OG Kids Cloud White`.
Traces: SC-10 (SITE-014 AC-3; ADMIN-013 AC-1); contract (GET /v1/products/{id})
Check: fetch_url

**Scenario**: TC-30 — a database that cannot be reached is answered 500 `internal`, never 404
**Given**: Arrangement D (the API running, its database refusing connections).
**When**: `GET /api/v1/products/samba-og-w-crystal-white` is fetched.
**Then**: Status 500 (not 404, not 200), JSON body with `error` `internal` and a generic `message` that names no table, query, path or connection string; the database is put back at the end and the same request then answers 200.
Traces: SC-12 (SITE-014 AC-4; DATA-005 failure row); contract (GET /v1/products/{id}, `internal`)
Check: fetch_url

---

## Coverage map — every acceptance criterion of the story has a case

| Criterion | Cases |
|---|---|
| SC-1 | TC-01, TC-02 |
| SC-2 | TC-03, TC-04 |
| SC-3 | TC-05, TC-12 (first reading) |
| SC-4 | TC-06, TC-07 |
| SC-5 | TC-08 |
| SC-6 | TC-09 |
| SC-7 | TC-10, TC-11 |
| SC-8 | TC-12, TC-13, TC-28 |
| SC-9 | TC-14, TC-15 |
| SC-10 | TC-16, TC-29 |
| SC-11 | TC-17, TC-18 |
| SC-12 | TC-19, TC-30 |
| SC-13 | TC-20, TC-21, TC-22 |
| SC-14 | TC-23, TC-24 |
| SC-15 | TC-25, TC-26 |
| contract (GET /v1/products/{id}) | TC-13, TC-27, TC-28, TC-29, TC-30 |

Thirty cases: 21 `interact_page`, 3 `fetch_url` only... see the counts below.

- `interact_page`: TC-01, TC-02, TC-03, TC-06, TC-07, TC-08, TC-09, TC-11, TC-12, TC-14, TC-16, TC-17, TC-19, TC-20, TC-21, TC-22 (16)
- `render_url`: TC-10, TC-13 (2)
- `measure_styles`: TC-04, TC-15, TC-18 (3)
- `fetch_url`: TC-27, TC-28, TC-29, TC-30 (4)
- `manual` (reasons given in each case): TC-05, TC-23, TC-24, TC-25, TC-26 (5) — 5 of 30, none for an interaction or a visual fact.

## Self-check against the story's criteria list

- SC-1 (no missing text at first paint, 0.5, 1, 1.5 s; no pill): TC-01 desktop, TC-02 phone.
- SC-2 (nothing between header and footer; header, footer, dock draw as usual): TC-03 reads `main` empty and `header`/`footer` present; TC-04 measures `main` height 0 and the dock against `/shop`.
- SC-3 (first render before the request): TC-05 (server-side render with the real hook), TC-12 (earliest reading for an unknown id).
- SC-4 (colour dot to a slow shoe; no previous shoe name; no missing text): TC-06; the related-pair card variant: TC-07.
- SC-5 (shoe drawn whole in the first paint after the answer; never missing text): TC-08.
- SC-6 (shoe at 1 s, the rest at 4 s): TC-09.
- SC-7 (the shoe reads as before; adding size 38 to the bag): TC-10, TC-11.
- SC-8 (404 after 2 s: nothing first, then h1 and pill, no shoe parts): TC-12, TC-13, TC-28.
- SC-9 (pill to `/shop`, exactly one "All shoes" link, no shoe parts; pressing it): TC-14; phone layout: TC-15.
- SC-10 (draft shoe: 404, missing view after the answer, shoe never drawn): TC-16, TC-29.
- SC-11 (unreachable for at least 5 s: nothing): TC-17 (readings to 6 s, both widths), TC-18.
- SC-12 (500): TC-19, TC-30.
- SC-13 (three sequences: 404 → live shoe; shoe → unknown id; unreachable → shoe): TC-20, TC-21, TC-22 respectively.
- SC-14 (files, tokens, 300-line gate, lint/build/test): TC-23, TC-24.
- SC-15 (a test per table row; loading and error tests fail on today's page): TC-25, TC-26.
