# Story — Product page waits for the shoe before saying it is missing

Module: `site` · Plan item 13 (bug, P2) · Requirement: SITE-014 (with the not-found row of SITE-005)
Shape: frontend only — one page file, no backend, no database, no new secret, no new words.

## 1. User story

As a **visitor** opening a shoe's page, I want the page to say "That pair is not in the shop any more." only when the shop has answered that no live shoe has that address, so that I do not think a pair has been withdrawn when it is only loading, or when the shop could not be reached.

## 2. Why this story exists

`src/pages/Product.tsx` draws the missing view whenever `useRecord("products", id).item` is undefined. Today the records are local JSON, so `item` is there on the first render. Once "Site data from the database" makes `useRecord` ask the API, `item` is undefined while the request is in flight — so every shoe page would flash "That pair is not in the shop any more." before the pair appears, and a visitor on a slow phone connection would read it for seconds. The data story is not allowed to edit pages (DATA-005 item 1), so the page has to learn to wait itself.

`useRecord` already returns `{ item, loading, error }` (DATA-005 item 7); `Product.tsx` reads only `item` today.

## 3. In scope

One decision in `src/pages/Product.tsx`, taken from what `useRecord("products", id)` returns for the id in the address:

| What the record request says | What the page draws between the header and the footer |
|---|---|
| A record for this id is in hand | The shoe, as it is drawn today — whatever the other flags say |
| No record, `loading` is true | Nothing |
| No record, `error` is set (server unreachable or a non-404 failure) | Nothing |
| No record, `loading` is false, `error` is null (the shop answered 404) | The missing view, as it is drawn today: the h1 `pages.product.missing` and the "All shoes" pill (`pages.product.back`) to `/shop` |

Also in scope:

- **The record must be the one the address names.** When the visitor goes from one shoe to another (colour dots, a related-pair card), whatever the hook holds while the new id loads, the page never shows the previous shoe's name or price under the new address and never shows the missing view for the new address before its answer.
- **The rest of the page does not hold the shoe back.** The line, category, shop terms, payment options, reviews, colour dots' siblings and related pairs have their own requests; the shoe draws as soon as its own record arrives and those parts fill in when theirs do (each already tolerates being undefined or empty).
- A test file for the four states above under `src/test/` (the hook stubbed per state). Test files are not "the page", so adding one does not break "Product.tsx only".

## 4. Out of scope

- **Any loading or error view** — a skeleton, a spinner, "Loading…", "We could not load this pair", "Try again". The approved design draws none (`design/design-system.md` lists loading and error views as an open design question), and SITE-014 asks for nothing in their place. The owner can ask for one later; it would be a design change plus new words in `content.json`.
- **A retry when the shop is unreachable.** The page stays empty until the visitor reloads or goes elsewhere (SITE-014 item 3 says "nothing … until the shop answers"; the hook does not poll).
- **`src/data/index.ts`.** What `useRecord` reports and when belongs to "Site data from the database" (DATA-005 item 7, AC-8, AC-11). This story does not patch it; if the hook breaks the contract in section 7, raise it against that story.
- **Loading behaviour of any other page** — the order page (ORD-007 AC-24), the shop list, home, bag, checkout, the dealer screens. Each has its own requirement and story.
- **A dealer opening a draft shoe's address.** The API answers 404 to a visitor and 200 to a signed-in dealer (ADMIN-013 item 1); the page follows the answer. This story adds no status check on the page and no "preview" banner.
- **Words, colours, layout, pictures, tokens.** The missing view keeps its present markup, copy (`pages.product.missing`, `pages.product.back`, both editable in the Studio) and spacing; no key is added to `content.json`.
- **The browser tab title while waiting** (`usePageMeta` stays as it is) and **the stock wording, pictures, size picker and bag behaviour** of the shoe page (SITE-005) — unchanged.
- **Related pairs asking for 500 records** while the API serves at most 100 — a data-story matter, not a waiting matter.

## 5. UI scope

One screen, `/shop/:id` (`Product.tsx`). The shoe view and the missing view are already in the approved design; this story only decides when each shows. The waiting and unreachable states are deliberately empty: the header, the call/message dock and the footer draw as on every page, and nothing sits between them.

| State | Between header and footer |
|---|---|
| Waiting (request in flight) | Empty |
| Shop unreachable / answered with a failure | Empty |
| Shoe arrived | The shoe view (SITE-005, SITE-006), unchanged |
| Shop answered "not found" (unknown or draft id, for a visitor) | The missing view, unchanged |

Phone first: at 390px the missing view is one column (heading, then the pill) with no horizontal page scroll, as today; the empty states add nothing that could scroll.

Known cost of "nothing": while the page is empty the footer sits directly under the header and moves down when the shoe arrives. That is what SITE-014 asks for; a reserved-height placeholder would be a design addition and is not part of this story.

## 6. Acceptance criteria

Each criterion narrows the SRS criterion in its brackets. `SC` is this story's number; `AC` is the SRS's. "The shop answers after N seconds" means the shoe request is held for N seconds by the test (a delayed API route, or the hook stubbed with `loading: true`). The missing view is "the heading `That pair is not in the shop any more.` and the `All shoes` pill under it".

### While the shop has not answered

- **SC-1 [SITE-014 AC-1]:** The shop answers the shoe request after 2 seconds. `/shop/samba-og-w-crystal-white` is opened. At the first paint and again at 0.5, 1 and 1.5 seconds the text "That pair is not in the shop any more." is nowhere on the page, and the missing view's "All shoes" pill is not there either.
- **SC-2 [SITE-014 items 1 and 3]:** During that same wait nothing is drawn between the header and the footer: no heading, no price, no size pills, no "Add to bag", no "All shoes" link of any kind. The header, the footer and (below 1024px) the call/message dock draw as on any page.
- **SC-3 [SITE-014 AC-1]:** The first render after the address opens — before the request has been sent — also shows no missing text: the page never reads "no request yet" as "not found". (A server-side render of the page with the real hook, which runs no effects, contains neither the missing text nor the pill.)
- **SC-4 [SITE-014 AC-1; SITE-005 AC-6]:** On the page of one shoe the visitor presses a colour dot of another shoe, whose answer takes 2 seconds. Until it answers, the missing text does not appear and the first shoe's name is not shown as the heading of the new address; when it answers, the second shoe is drawn.

### When the shoe arrives

- **SC-5 [SITE-014 item 4; SITE-005 AC-1]:** The 2-second answer is the shoe. In the first paint after the answer the page shows the heading (`data-testid="product-name"`) with the shoe's name, the "All shoes" back link, the price, the size pills and "Add to bag" — no second wait — and the missing text has not been on the page at any moment since the address opened.
- **SC-6 [SITE-014 item 4]:** The shoe's answer comes at 1 second and the answers for its line, category, shop terms, payment options, reviews and related pairs at 4 seconds. At 1 second the shoe is drawn (heading, price, sizes, "Add to bag"); the parts that depend on the other answers appear when theirs arrive; the missing text is never shown.
- **SC-7 [SITE-005 AC-1, AC-9, AC-14]:** With the shoe answered at once, `/shop/samba-og-w-crystal-white` still reads as before: the heading is the shoe's name, the line reads `Samba · Women · IH9055`, the price reads `3.090.000 ₫`, the status line reads "In stock in all 8 sizes." for a shoe with 9 sizes and 1 sold out and none on last pairs, and choosing `38` then "Add to bag" puts one pair `EU 38` in the bag and opens the bag drawer.

### When the shop says "not found"

- **SC-8 [SITE-014 AC-2]:** The shop answers 404 (after 2 seconds) for `/shop/not-a-shoe`. Before the answer the missing view is not on the page; when it arrives the page shows the h1 "That pair is not in the shop any more." and the "All shoes" pill, and no shoe parts.
- **SC-9 [SITE-014 AC-2; SITE-005 not-found row]:** In the missing view the "All shoes" pill links to `/shop` and pressing it opens `/shop`. The page holds exactly one "All shoes" link (the pill) and no price, size pills, "Add to bag", reviews or related pairs.
- **SC-10 [SITE-014 AC-3; ADMIN-013 AC-11]:** A shoe is in draft. A visitor opens its address: the shop answers 404, and after that answer the page shows the missing view; the shoe is never drawn, and the missing view is not drawn before the answer.

### When the shop cannot answer

- **SC-11 [SITE-014 AC-4]:** The shop cannot be reached (the connection is refused or the browser is offline) when `/shop/samba-og-w-crystal-white` opens. For at least 5 seconds the missing text and the missing view's "All shoes" pill never appear, no shoe is drawn, and nothing sits between the header and the footer.
- **SC-12 [SITE-014 AC-4; DATA-005 failure row]:** The shop answers the shoe request with a 500. The page behaves as in SC-11: no missing text at any moment, nothing between header and footer.

### One address does not leak into the next

- **SC-13 [SITE-014 AC-1, AC-2, AC-4]:** After a 404 on `/shop/not-a-shoe`, opening a live shoe (from the missing view's pill to `/shop`, then a shoe card) draws that shoe with no missing view. After a shoe has been drawn, opening an unknown id shows nothing, then the missing view only after its answer. After an unreachable shop, opening a shoe whose answer comes normally draws it.

### Boundaries of the change

- **SC-14 [SITE-014 items 1–4; architecture §5]:** The change set edits `src/pages/Product.tsx` and adds its test file under `src/test/`. It touches nothing in `src/data/`, `src/components/`, `src/App.tsx`, `src/content.json`, `src/theme.css` or `src/index.css`; it adds no key to `content.json`, no raw colour and no arbitrary-value class; `Product.tsx` stays under the 300-line lint gate; `npm run lint`, `npm run build` and `npm test` pass.
- **SC-15 [SITE-014 AC-1 to AC-4]:** The test file covers each row of the section 3 table with the hook stubbed — record in hand, loading, error, and answered-not-found — and asserts for each which of the shoe and the missing view is on the page; the tests fail against today's `Product.tsx` for the loading and error rows.

## 7. Dependencies

- **Lands first: "Site data from the database" (plan item 3).** Until `useRecord` asks an API, `loading` is always false and there is nothing to wait for; the criteria with a delayed or failing shop (SC-1, SC-4, SC-6, SC-8 to SC-13) are verified against the running API through `run_project`, the rest by the stubbed tests. The code of this story can be written and its tests pass before that story lands, because `useRecord` already returns `{ item, loading, error }`.
- **What this story needs from `useRecord(name, id)` (DATA-005 item 7, AC-11; DATA-003 item 2):** for a new id the first render already reports `loading` true with `item` undefined (SC-3); a 404 gives `item` undefined, `loading` false, `error` null; any other failure gives `error` set and `item` undefined; changing `id` does not leave the previous id's record in place as if it were the new one. If the data story's hook does not do this, raise it against that story and do not patch `src/data/` here.
- **Shares a file with "Free delivery on every order, wording to match" (plan item 2):** that story edits `DetailRows` ("Payment options") in `Product.tsx`; this one edits the default export `Product()`. The regions do not overlap; whichever lands second rebases.
- **External accounts and secrets:** none.
- **Words:** none added. The two strings used (`pages.product.missing`, `pages.product.back`) already exist and stay editable in the Studio.

## 8. How it is checked

- **Unit tests (`npm test`):** `Product` rendered inside a router at `/shop/<id>` with `../data` stubbed per state (SC-3, SC-5, SC-8, SC-11, SC-12, SC-15), plus a render with the real hook through `renderToString` for SC-3.
- **Running site (`run_project`):** the API's `GET /v1/products/{id}` held for 2 seconds (SC-1, SC-2, SC-5, SC-6), answering 404 for an unknown and a draft id (SC-8 to SC-10), refusing the connection and answering 500 (SC-11, SC-12), and the navigation sequences of SC-4 and SC-13 at 390px and 1280px.
