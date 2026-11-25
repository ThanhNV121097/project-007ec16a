# Story — Shop phone filters, Clear all, loading and errors

Module: `catalogue` · Plan item 16 (the story of plan item 4 calls it "15") · Priority P1 · Requirements: CAT-003 items 6–8, CAT-004 in full (`docs/catalogue/SRS.md`)
Design: the approved `/shop` page; `design/design-system.md` §1.5 Layout and breakpoints, §2.1.14 FilterRail, §2.1.15 Select, and the empty-state rule in the design's principles ("Never a blank area")

## 1. The story

*As a* **visitor** shopping on my phone, *I want* to open and close the filters from one button, to come back to my list after looking at a shoe, to start over in one tap, and to be told plainly when nothing matches, while the shoes are loading and when loading failed, *so that* I am never left looking at a blank page and can always get back to the whole shop.

The actor is the visitor only. The dealer's lists use the same table hook; this story must not change what they do (SC-19).

## 2. Why this story exists

It is the second half of the split of plan item 4, "Search, filters and sorting served by the API" (`docs/catalogue/stories/search-filters-and-sorting-served-by-the-api.md` §3). Item 4 builds what is asked and what comes back: the query language, the shop reading twelve shoes at a time, the address-bar filters. This story builds how the visitor drives it on a phone, gets back to it from a shoe, and what they see while waiting, when nothing comes and when the request fails.

Two things are true of the page today and are the reason the story is needed:

- **"Clear all" leaves the size set.** `src/lib/table.ts` reads `size` as the page size and its `clear()` keeps it; its `filtering` flag also ignores it. So with only a size pill pressed there is no "Clear all" link at all, and with other filters set pressing it leaves the pill pressed (CAT-004 AC-1 and AC-2 fail).
- **A failed or slow request looks like "no results".** The empty state is drawn whenever the list is empty and not loading, so a failed request reads "No pair matches that." The design draws no loading or error view; the plan item requires them.

## 3. Build order

**Build only after plan item 4 has merged.** Both change `src/pages/Shop.tsx`, `src/lib/table.ts` and the shop's components. This story adds to what item 4 builds: the list already read from the API a dozen at a time with "Show more", the address-bar filters, the price-band union and `size` sent as a filter. It also needs the data story's hook (plan item 3), which already returns `items`, `total`, `hasMore`, `loading`, `error` and `loadMore`.

## 4. In scope

**Frontend only — `/shop`.** No backend work, no endpoint, no table, no migration: everything reads what item 4 serves. The Story BE and BE review steps have nothing to build.

1. **Phone filters.** Below 1024px the filter rail is hidden and a "Filter" button opens and closes it; the button reads "Filter (n)" with the number of active filters when there are any. From 1024px the rail is always shown and there is no button.
2. **Back.** Opening a shoe and pressing the browser's Back returns to the same list with the same filters, order and search, showing the first twelve.
3. **Entries.** The home page's category tiles (`?category=<id>`), the "Gifts" tile and the footer's "Gifts" (`?price=under-3m`), the footer's Women / Men / Kids / All shoes links and a home-page line row (`?line=<id>`) land on the filtered list with the matching tick shown.
4. **Single-option groups.** A filter group with fewer than two options is not drawn — the checkbox groups already behave so; the Size (EU) pills, drawn by the page itself, do not yet.
5. **Clear all.** Removes every filter, the search words and the size pills and keeps the order. It is shown under the rail only when something is set. The address parameter stays `size` (SRS CAT-003 item 1); how the shop's hook stops treating it as a page size is TL's decision (§8).
6. **Empty state.** Heading "No pair matches that.", the line "Try another size or fewer filters — or ask us, and we will tell you when it comes in.", a "Clear all" button with an arrow, and the count `0 matching`. The words and the button exist today; this story makes the state appear only when it is true and proves it.
7. **Loading.** "Loading the shoes…" while the first page is fetched (the empty-state heading is not shown); "Show more" is disabled and reads "Loading…" while its page is fetched.
8. **Errors.** "We could not load the shoes." with a "Try again" button when the first page cannot be fetched; when "Show more" fails the shoes already shown stay, the message appears under them and "Try again" asks for the same page. Offline behaves like a 5xx.
9. **Four new strings** in the site's content under `pages.shop.*`, so the owner can edit them in the Studio (§7, SC-31).

## 5. Out of scope

| Left out | Why, and where it goes |
|---|---|
| The API, the query language, `GET /v1/product-filters`, the paged list, `limit=12` and `status=live`, the address-bar filters, the price-band union, headings, count, sort, "Show more" paging itself (CAT-001, CAT-002, CAT-003 items 1–5) | Plan item 4. This story builds on them and proves only the behaviours in §9 |
| The header's search button and menu, its "Start from a line" links | `site` module, SITE-001. Where its search lands (`/shop?q=…`) is proved by item 4 (SC-38) |
| The dealer's lists, their "Clear filters" button, their page size | `admin` module. They keep `size` as the page size and keep it on "Clear filters" (SC-19) |
| Loading and error views on any page other than `/shop` (home, a shoe's page, the size chart…) | Not in the plan item; the data story leaves them as an open design question. The shoe page's "missing shoe" bug is plan item 13 |
| Restoring the cards of earlier "Show more" pages, or the scroll position, after Back | The SRS asks for the first twelve (CAT-003 item 7) |
| A "Clear all" beside the "Filter" button while the rail is closed | The design puts "Clear all" under the rail. On a phone it is reached by opening the rail; the empty state also carries one |
| A "notify me when it comes in" feature, a contact link in the empty state | The owner-approved line says "ask us" and the page has no such feature; the contact page and dock are one tap away from every page |
| Automatic retry with back-off, error reporting, a separate offline banner | The data hook already tries a failed read once more; the visitor's own "Try again" is the only other retry |
| New design tokens, edits to `theme.css`, `index.css`, `package.json`, the lint config | Frozen for stories; the loading and error views use the existing tokens only |
| Keeping drafts away from a visitor whatever is asked | `admin` module, ADMIN-013 |

## 6. UI scope

Screen: **Shop `/shop`** only; entries are tested from Home and the footer, which are not edited (their links already point to the right addresses — if one does not, that is a defect report to TL, not a change to those pages). Files this story may touch: `src/pages/Shop.tsx`, `src/components/FilterRail.tsx`, the shop's own new components, `src/lib/table.ts` (as TL rules) and the data hook for the retry (as TL rules). `ProductCard`, `Grid`, `Select`, `Bleed`, `HeadPicture`, `Icon` are reused as drawn.

| State | In this story |
|---|---|
| Default, 1280px: rail at the left, no "Filter" button | Yes (SC-4) |
| Phone, rail closed, "Filter" / "Filter (n)" | Yes (SC-1, SC-2) |
| Phone, rail open | Yes (SC-3) |
| A group with a single option | Yes (SC-12, SC-13) |
| Empty ("No pair matches that.") | Yes (SC-20, SC-21) |
| Loading the first page | Yes — **not in the approved design** (SC-22, SC-23) |
| "Show more" loading | Yes — not in the design (SC-24) |
| Load failed | Yes — not in the design (SC-25, SC-26) |
| "Show more" failed | Yes — not in the design (SC-27) |

**How the undrawn states look** (SRS open question 1, decided in §7): they sit in the results area in the empty state's own block — the same `border-b border-line` band, top and bottom spacing and left alignment — in the empty state's own type: the error message in the empty-state heading's size and the loading line in the empty-state line's size (`text-sm`, `text-ink-soft`). "Try again" is the same outline pill as the empty state's "Clear all" button (without the arrow). The error may use `text-danger`, the design's colour for what has gone wrong. No raw colour, no arbitrary-value class, no CSS module. The `<N> matching` line is not drawn while the first page is loading or after it failed, so it never reads `0 matching` beside "Loading the shoes…". Test hooks that exist today (`data-testid="shop-count"`, `filter-size-<size>`) stay.

The page stays under the 300-line lint gate: the results area (grid, empty, loading, error, "Show more") and the size pills move into components of their own if `Shop.tsx` would otherwise pass it.

Words: the Filter button, "Clear all", the empty state and "Show more" use keys that exist today under `pages.shop.*`. The four new strings are in §7.

## 7. Decisions taken instead of asking

- **Loading and error copy** (SRS open question 1): the four strings of CAT-004, exactly as written, in the empty state's type and spacing, added to the site's content as **new keys only**, existing keys untouched: `pages.shop.loading` = "Loading the shoes…", `pages.shop.error` = "We could not load the shoes.", `pages.shop.retry` = "Try again", `pages.shop.loading_more` = "Loading…" (the ellipsis is the single character `…`). `src/content.json` is frozen to stories by the architecture overview; this is the same one-time lift for new keys that the staff-accounts story asks for. TL confirms it at the UI review; if TL prefers to add the keys, the UI PR lists them instead.
- **What "active filters" counts** (the number in "Filter (n)"): every ticked box and every pressed size pill, one each — `category=women,men` is 2. The search words and the sort are not filters and are not counted; the search box and the sort control are visible on a phone anyway.
- **"Clear all" with only a sort set** is not shown: it keeps the sort, so pressing it would change nothing ("Nothing is set", CAT-004 AC-3).
- **While "Show more" is failing** one control is shown, not two: the message and "Try again" take the place of the "Show more" button, and "Try again" asks for the same page.
- **No count while loading or failed.** The `<N> matching` line is drawn only when a list has been fetched; it never shows `0 matching` for a list that has not arrived. The empty state's `0 matching` is a real count.
- **A filter ticked on a group that is not drawn** (a one-colour catalogue opened at `?colour=black`) still applies, still counts in "Filter (n)", and "Clear all" removes it.
- **Words matching, price bands, the shape of the address** are item 4's decisions and are not reopened here.

## 8. Decisions this story leaves to TL (technical design, step 6)

These are not questions for the owner; the SRS and the plan item name them as TL's.

1. **How "Clear all" removes the size without touching the dealer's lists.** The address parameter stays `size` on `/shop` — CAT-003 item 1, CAT-004 AC-1 and AC-2 and item 4's SC-34 all name it. (If TL finds that cannot work, the address changes and so do those criteria: send it back to PM before building.) `useTableQuery` is called in `Shop.tsx` and, separately, inside `FilterRail`; whatever TL picks (an option on the hook, a parameter of the shop's own) must give the same `clear()` and `filtering` answers in both calls, and an unchanged answer to every caller that does not opt in. `table.ts` is a scaffold file: TL rules the edit.
2. **Retry and "more is loading" in the data hook.** The hook returns `{ items, total, pages, page, hasMore, loading, error, loadMore }`, keeps the items already loaded when a read fails, and has no way to ask again. "Try again" needs one, additively (no exported name or signature changes): re-ask the page that failed, never skip it and never repeat one already shown. "More is loading" can be derived (`loading` with items on screen) or returned. An older request's failure arriving after a newer request's answer must not set `error` (SC-30). The hook already retries a failed read once and aborts one unanswered after 15 seconds, so an error appears after that retry.
3. **Where the four strings live** (§7): TL confirms the exception for new keys in `src/content.json`.
4. **Where the group-hiding rule sits** for the size pills: inside `FilterRail`'s groups, or in a small component the page draws. Either way a group with fewer than two options is not drawn and the "Size (EU)" heading is not left over an empty grid.

## 9. Acceptance criteria

Thirty-three criteria, numbered `SC-n`. Each names the requirement and the SRS criterion it narrows (`CAT-003 AC-13`); the SRS says what it means in full. All are about the page in a browser; none needs a new endpoint. **Fixtures:** the seed holds 12 placeholder shoes, so a case needing 30 live shoes, a draft shoe, a one-colour or one-size catalogue builds them in its own test database. **Failure and delay** are made by the test's request interception (hold a request for 3 seconds, answer 500, abort, go offline) — the product carries no switch for them. **Viewports:** 320, 390, 1023, 1024 and 1280px wide.

### 9.1 Phone filters (CAT-003 item 6)

- **SC-1 [CAT-003 AC-13, item 6]:** At 390px `/shop` loads with the filter rail not visible (not on screen and not read out by assistive technology) and a "Filter" button in the search and sort band; with nothing set the button reads "Filter" with no brackets, and the first shoe card is visible without pressing anything.
- **SC-2 [CAT-003 AC-14, item 6]:** At 390px with `category=women&line=samba&size=38` the button reads "Filter (3)" (a space before the bracket). Each ticked box and each pressed size pill counts one — `category=women,men` alone reads "Filter (2)" — while search words and a sort do not: `q=samba&sort=price` alone reads "Filter". Pressing "Clear all" brings it back to "Filter".
- **SC-3 [CAT-003 AC-15, item 6]:** At 390px pressing "Filter" opens the rail — the For, Line, Colour and Price groups, the Size (EU) pills and, when something is set, "Clear all" — and the button reports `aria-expanded="true"`; pressing it again closes the rail and reports `false`. Ticking a box while the rail is open leaves it open and updates the number in the button.
- **SC-4 [CAT-003 AC-16, item 6]:** At 1280px the rail is shown with all its groups and there is no "Filter" button. The switch is at 1024px: at 1024px the rail is shown and there is no button; at 1023px the rail is hidden and the button is shown.
- **SC-5 [SRS §6 Responsive]:** At 320px and at 390px the page has no horizontal scroll (the document is no wider than the window) in each of: the default list, the rail open with every group, the empty state, "Loading the shoes…", "We could not load the shoes." and the "Show more" error.

### 9.2 Entries and Back (CAT-003 items 7 and 8)

- **SC-6 [CAT-003 AC-17, item 8]:** Pressing the "Kids" tile on the home page opens `/shop?category=kids`: the heading reads "Kids", only kids' shoes are listed, and "Kids" is ticked under For (at 1280px; at 390px the button reads "Filter (1)" and "Kids" is ticked once the rail is opened).
- **SC-7 [CAT-003 AC-18, item 8]:** Pressing the "Gifts" tile on the home page opens `/shop?price=under-3m`: "Under 3.000.000 ₫" is ticked, every card shown is priced below 3.000.000 ₫, and the heading reads "All shoes".
- **SC-8 [CAT-003 AC-19]:** Pressing "Gifts" in the footer, from any page, gives the same result as SC-7.
- **SC-9 [CAT-003 item 8]:** The footer's Women, Men and Kids links open `/shop?category=women`, `…=men` and `…=kids` with that box ticked and the heading "Women", "Men" or "Kids"; the footer's "All shoes" opens `/shop` with nothing ticked; a line row on the home page opens `/shop?line=<id>` with that line ticked and the line's name as the heading.
- **SC-10 [CAT-003 AC-9, item 7]:** With Women, Samba, size 38, a price band, the search `og` and `sort=price` set, opening a shoe's card and pressing the browser's Back returns to `/shop` with the same address, the same boxes ticked and pill pressed, `og` in the search box, "Price, low to high" in the sort control and the same count and cards.
- **SC-11 [CAT-003 item 7]:** With 30 live shoes, "Show more" pressed once (24 cards), opening a card and pressing Back shows the first twelve only — 12 cards, `30 matching`, a "Show more" button, none of the second page's cards — with any filters and order still set.

### 9.3 Groups with one option (CAT-003 item 6 rail, AC-21)

- **SC-12 [CAT-003 AC-21]:** When every live shoe lists the same single size and has the same single colour, the Size (EU) group (heading and pills) and the Colour group are not drawn, and the For, Line and Price groups still are; a group with two options is drawn. While the request for the colours and sizes is held, neither group is drawn (no heading over an empty list), and both appear when it answers.
- **SC-13 [CAT-003 AC-21, boundary]:** In that one-colour catalogue, `/shop?colour=black` lists only black shoes, counts the filter (at 390px "Filter (1)"), shows "Clear all", and pressing it removes `colour` from the address.

### 9.4 Clear all (CAT-004 item 1)

- **SC-14 [CAT-004 AC-1]:** With Women, Samba, size 38, a price band and the search `og` set, pressing "Clear all" under the rail leaves none of `category`, `line`, `colour`, `size`, `price`, `q` in the address; no box is ticked, no pill is pressed, the search box is empty; the heading reads "All shoes" and the list is every live shoe, first twelve, with the same count as a fresh `/shop`; at 390px the button reads "Filter".
- **SC-15 [CAT-004 AC-2]:** With only `size=38` set a "Clear all" link is shown under the rail; pressing it removes `size` from the address, un-presses the pill and lists every live shoe.
- **SC-16 [CAT-004 AC-2, item 1]:** With only `q=samba` set the link is shown; pressing it removes `q`, empties the search box and shows the heading "All shoes".
- **SC-17 [CAT-004 AC-3]:** With nothing set no "Clear all" link is shown; with only `sort=price` set none is shown either; ticking any box makes it appear.
- **SC-18 [CAT-004 AC-4]:** With `sort=price` and filters set, pressing "Clear all" removes the filters and keeps `sort=price`: the control still reads "Price, low to high" and the cards rise by price.
- **SC-19 [CAT-004 item 1, TL constraint]:** The dealer's lists are unchanged: for any caller of `useTableQuery` that does not opt in to the shop's behaviour, `clear()` still keeps `size` (the page size) and `filtering` still ignores it — a unit test of the hook — and `/admin/products?size=50&status=live`, signed in, still shows 50 rows a page and "Clear filters" keeps `size=50`. On `/shop`, `size=38` is never read as a page size: the list is still twelve a page.

### 9.5 Empty state (CAT-004 item 2)

- **SC-20 [CAT-004 AC-5, Not found]:** `/shop?q=zzzz` and `/shop?line=zzz` each show the heading "No pair matches that.", the line "Try another size or fewer filters — or ask us, and we will tell you when it comes in.", a "Clear all" button with an arrow and the count `0 matching`; no card, no "Show more", no loading or error message.
- **SC-21 [CAT-004 AC-6]:** Pressing the empty state's "Clear all" shows the unfiltered list: `q` or `line` leaves the address, the search box is empty, the count is that of the whole shop, and a chosen sort is kept.

### 9.6 Loading (CAT-004 item 3)

- **SC-22 [CAT-004 AC-7, item 3]:** With the first request held 3 seconds `/shop` shows "Loading the shoes…" for the whole wait; at no moment before the answer is "No pair matches that." visible; no card and no `0 matching` is shown; when the answer arrives the message is gone and the 12 cards and the count appear.
- **SC-23 [CAT-004 AC-7, item 3]:** The same holds when ticking a box asks for a new first page that is held 3 seconds: "Loading the shoes…" is shown, "No pair matches that." is not, and the previous list's cards are not left on screen.
- **SC-24 [CAT-004 AC-8, item 3]:** With 30 live shoes and the second page held 3 seconds, pressing "Show more" disables the button and it reads "Loading…" with the first 12 cards staying; pressing it again sends no second request; when the page arrives there are 24 cards, the count still reads `30 matching` and the button reads "Show more" again.

### 9.7 Errors (CAT-004 item 4)

- **SC-25 [CAT-004 AC-9, Upstream failure]:** When every request for the shoes answers 500, `/shop` shows "We could not load the shoes." and a "Try again" button; "No pair matches that.", "Loading the shoes…", any card and `0 matching` are not shown, and no status code, error code or server message appears on the page.
- **SC-26 [CAT-004 AC-10]:** After SC-25, with the server answering again, pressing "Try again" shows "Loading the shoes…" and then the list; the message and the button are gone, and the request sent carries the same filters, `limit=12` and `status=live`, page 1.
- **SC-27 [CAT-004 AC-11, item 4]:** With 30 live shoes and the second page answering 500, pressing "Show more" keeps the first 12 cards and `30 matching` on screen and shows "We could not load the shoes." with "Try again" under them, in place of the "Show more" button. "Try again" sends page 2 again with the same filters; when it succeeds there are 24 cards, none twice and none missing, and the message is gone; when it fails again the 12 cards and the message stay.
- **SC-28 [CAT-004 AC-12]:** With the browser offline `/shop` shows what SC-25 shows; with the browser offline when "Show more" is pressed it shows what SC-27 shows.
- **SC-29 [CAT-004 item 4, boundary]:** With "We could not load the shoes." on screen, ticking a box asks for the first page of the new list: the message is replaced by "Loading the shoes…" and then, when it succeeds, by the list.
- **SC-30 [CAT-002 Conflict, CAT-004 item 4]:** When a filter is changed while an earlier request is in flight and the earlier request fails after the newer one has succeeded, the newer list stays and no error message appears.

### 9.8 Words, access and look (CAT-004 item 5, SRS §6)

- **SC-31 [CAT-004 item 5, open question 1]:** The four strings read exactly "Loading the shoes…", "We could not load the shoes.", "Try again" and "Loading…"; they are drawn from the site's content (`pages.shop.*`, new keys only, no existing key changed or removed), so editing a value in the Studio changes the page; no string is typed into a component; the language is English throughout.
- **SC-32 [CAT-004 item 5, SRS §6 Accessibility]:** "Loading the shoes…" is exposed as a status (`role="status"`) and "We could not load the shoes." as an alert (`role="alert"`), each present in the page when shown so assistive technology announces it; the disabled "Show more" carries the `disabled` attribute; "Try again" and "Clear all" are real buttons with visible text; the "Filter" button is named and carries `aria-expanded`; the size pills carry `aria-pressed`; the search input keeps its label and the sort control its name "Sort".
- **SC-33 [CAT-004 item 5]:** The loading, error and "Show more" error views use the design's tokens only (`npm run lint` and the CI token job pass; no raw colour, arbitrary-value class or CSS module), sit in the empty state's block, and "Try again" has the same computed font size, letter-spacing, border and radius as the empty state's "Clear all" button.

## 10. Dependencies

| Needs | Why |
|---|---|
| **Plan item 4, "Search, filters and sorting served by the API"** (must merge first) | Shop reads the API a dozen at a time with "Show more", the address-bar filters, the price-band union, `size` sent as a filter, `product-filters` for the colours and sizes. This story edits the same `Shop.tsx`, `table.ts` and components |
| **Plan item 3, "Site data from the database"** (merged before item 4) | The data hook with `loading`, `error`, `loadMore`, "the latest query wins", the one retry of a failed read and `src/lib/api.ts`'s `ApiError` |
| Plan item 13, "Product page waits for the shoe before saying it is missing" (independent) | The shoe page opened and left in SC-10 and SC-11; no shared file |
| Data for Test | 30 live shoes, a one-colour and one-size catalogue, 12 live shoes and a draft shoe built as fixtures in a test database, never the seed; request interception for delay, 500 and offline |
| External accounts, keys, provider setup | None |
| New tokens or Studio changes | None |

## 11. Requirement coverage

| Requirement | Where it is proven |
|---|---|
| CAT-003 item 6 (phone Filter button, count, rail from 1024px) | SC-1 to SC-5, SC-12, SC-13 |
| CAT-003 item 7 (Back) | SC-10, SC-11 |
| CAT-003 item 8 (entries, tick shown) | SC-6 to SC-9 |
| CAT-003 AC-9, AC-13 to AC-19, AC-21 | SC-10, SC-1, SC-2, SC-3, SC-4, SC-6, SC-7, SC-8, SC-12 |
| CAT-004 item 1 (Clear all) and AC-1 to AC-4 | SC-14 to SC-19 |
| CAT-004 item 2 (empty state) and AC-5, AC-6 | SC-20, SC-21 |
| CAT-004 item 3 (loading) and AC-7, AC-8 | SC-22 to SC-24 |
| CAT-004 item 4 (errors) and AC-9 to AC-12 | SC-25 to SC-30 |
| CAT-004 item 5 (undrawn states, new strings) | SC-31 to SC-33 |
| SRS §6 Responsive, Accessibility, Localisation | SC-5, SC-31, SC-32 |
| CAT-003 items 1–5, CAT-001, CAT-002 | Out of scope here — plan item 4 |
