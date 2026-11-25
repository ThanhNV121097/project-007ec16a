# Story — Search, filters and sorting served by the API

Module: `catalogue` · Plan item 4 · Requirements: CAT-001, CAT-002, and the request side of CAT-003 (`docs/catalogue/SRS.md`)
Design: the approved `/shop` page; `design/design-system.md` §2.1.8 ProductCard, §2.1.9 Grid, §2.1.14 FilterRail, §2.1.15 Select

## 1. The story

*As a* **visitor**, *I want* the shop to answer a combined question — "women's Samba in EU 38 under 3 million" — with exactly the matching live pairs, in the order I chose, a dozen at a time with the true count, *so that* it stays quick on a phone and I can send the list to a friend as a link.

The **dealer** (staff or admin) is a second reader of the same endpoints: the dealer's lists (`admin` module) use the same query language. This story builds it once, for both.

## 2. Why this story exists

Today `/shop` loads up to 500 shoes into the browser and filters them there. Once the records sit behind the paged API of the data story (100 a page at most) that stops working, and that story's list endpoint only filters by exact field value. This story gives the list endpoints their query language and makes the shop page read filtered, sorted pages from it.

## 3. Split from the plan item

The plan item's four requirements carry 82 acceptance criteria in the SRS. One story cannot be built, reviewed and verified well past about 40, so the plan item is cut in two:

| Part | Requirements | Plan item |
|---|---|---|
| **This story** — what is asked and what comes back | CAT-001 in full; CAT-002 in full; CAT-003 items 1–5 (the address-bar filters, price bands, any-of / all-of, the search box) | 4 |
| **Follow-up** — how the visitor drives it on a phone, gets back to it, and what they see while waiting or when nothing comes | CAT-003 items 6–8 (phone "Filter" button, Back, entries, single-option groups); CAT-004 in full (Clear all, empty state, loading, errors) | 15, "Shop phone filters, Clear all, loading and errors" |

Plan item 15 changes the same page and hook files as this story, so it is built after this one merges. Until it lands, a failed request leaves the results area without a message (SC-40 only guarantees it does not claim "No pair matches that.").

## 4. In scope

**Backend** (package `internal/catalogue`, over the data module's list endpoints):

- The query language on `GET /v1/<collection>`: `q` (free text, case-, accent- and `đ`-insensitive, every word must hit), `<field>=a,b` (any of), `<field>_from` / `<field>_to` (inclusive ranges on numbers and dates), `sort=<field>` / `sort=-<field>` on any field, and `size=a,b` on shoes (EU size listed and not sold out). The searched fields per collection are those of CAT-001 item 2. `total` is every match.
- `GET /v1/product-filters`: the colours and EU sizes the shop page lists (CAT-002 item 8), per `docs/architecture/services.md` §3.4.
- A way for one request to carry a union of price bands (see §8, TL's decision).
- 400 answers naming the parameter for a bad `sort`, a non-number or non-date range value and a `q` over 100 characters.
- Search over `orders` (order number and customer name only) and `reviews` (customer name, text, shoe's name), because the dealer's lists use them.

**Frontend** (`/shop`, `src/pages/Shop.tsx` and the components it uses):

- The list reads from the API: the first twelve live shoes that match, `limit=12` and `status=live` on every shoe request, "Show more" asks for the next twelve and adds them below.
- The heading, the picture beside it, the `<N> matching` count (the API's `total`), the shoe cards, the sort control and its four options, as CAT-002 describes them.
- The Colour options and the Size (EU) pills come from `GET /v1/product-filters`, so they do not shrink as the visitor filters.
- Every address parameter reaches the request: `category`, `line`, `colour`, `price` (band names), `size`, `q`, `sort`. Ticking a box or a pill returns the list to its first twelve.
- "Under", "3.000.000 ₫ – 4.000.000 ₫" and "4.000.000 ₫ and over" work alone and together (a union of ranges).
- `size` in the address is a size filter on `/shop`; the shop never reads it as a page size (its request is always `limit=12`).

## 5. Out of scope

| Left out | Why, and where it goes |
|---|---|
| Storing and seeding records, the paged envelope, exact-value filters, `useCollection` / `useRecord` calling the API | `data` module, plan item 3. This story only extends what it built |
| Phone "Filter" button and its count, Back returning to the same list, entries from home and footer showing the tick, a group with one option not drawn (CAT-003 items 6–8, AC-9, AC-13 to AC-19, AC-21) | Follow-up plan item 15. They exist on today's page; this story must not break them but does not prove them |
| "Clear all" (including removing `size`), the empty state's button and its wording, "Loading the shoes…", "Loading…" on "Show more", "We could not load the shoes.", "Try again" (CAT-004) | Follow-up plan item 15. The SRS's open question 1 is decided there: the copy in CAT-004, in the empty state's type and spacing, added to the site's content |
| The header's search button and menu | `site` module, SITE-001. This story covers where the search lands: `/shop?q=…` |
| Keeping drafts and hidden reviews from a visitor whatever is asked | `admin` module, ADMIN-013. Until it lands the shop page sends `status=live` itself |
| What the dealer's lists show, columns and saving | `admin` module |
| Which sizes are in stock on a shoe's own page | `site` module, SITE-005 |
| Search suggestions, search-as-you-type, ranking by relevance | Not in the design. The search runs on Enter; results keep the chosen sort |
| New tables, columns or a migration | None expected: the ERD lists this story as queries only (`erd.md` §8). If TL finds the performance criterion needs an index on a table that has rows, it is created `CONCURRENTLY` |
| Stock counts, `sizes_low` display, "Out of stock" wording | `orders` / `site` modules |

## 6. UI scope

Screen: **Shop `/shop`** only. No other page is edited; `ProductCard`, `Grid`, `FilterRail`, `Select`, `Bleed`, `HeadPicture` are reused as drawn.

| State | In this story |
|---|---|
| Default, results, "Show more" visible | Yes — the first twelve, the count, the button |
| All matches shown | Yes — no button |
| Filtered, sorted, searched | Yes — read from the address |
| Empty ("No pair matches that.") | Existing behaviour is kept; it must not appear while the first page is on its way or after a failed request (SC-40). Its wording and button are proven in plan item 15 |
| Phone with the rail closed / open | Existing behaviour is kept; proven in plan item 15 |
| Loading, "Show more" loading, error, "Show more" error | Not drawn in the design; built in plan item 15 |

No new strings: every word this story shows already exists under `pages.shop.*` in `src/content.json`, which stories do not edit. Prices read as `vi-VN` (`3.290.000 ₫`). The search input keeps its label; the sort control keeps its name; size pills keep `aria-pressed`.

## 7. Decisions taken instead of asking

- **Word matching** (SRS open question 2): every typed word must be found as part of a value in at least one searched field, as CAT-001 item 1 says.
- **Where the shop reads its colour and size options** — one endpoint (`product-filters`), not a scan of 500 shoes in the browser.
- **The shop asks for twelve, never more than 100**; "Show more" is a second request for page 2, then 3, not a larger limit.
- **Loading and error copy** (SRS open question 1) is decided in plan item 15, not here.

## 8. Acceptance criteria

Forty criteria, numbered `SC-n`. Each names the requirement and the SRS criterion it narrows (`CAT-001 AC-3`); the SRS says what it means in full. `<seed>` means the seeded records of the data story; where a case needs more (30 live shoes, a draft shoe, a shoe containing `100%` or `_`, one with "Cửa Lò" or "Đại", 10,000 shoes) Test builds its own fixtures in a test database.

### 8.1 The list endpoints answer the query (backend)

- **SC-1 [CAT-001 AC-1, AC-2]:** `GET /v1/products?q=samba%20og` and `?q=og%20samba` return the same set: every item contains both words in a searched field, the Samba OG shoes are among them, and `total` equals the number of items across all pages.
- **SC-2 [CAT-001 AC-3, AC-4, AC-5]:** Case, accents and `đ` are ignored: a shoe whose description contains "Cửa Lò" is returned for `q=cua%20lo` and for `q=CUA%20LÒ`; a shoe whose name contains "Đại" is returned for `q=dai`.
- **SC-3 [CAT-001 AC-6, AC-7]:** `q` also searches the line's and the category's name: `q=gazelle` returns every shoe of the line "Gazelle" although "gazelle" is not in those shoes' own fields, and `q=women` returns the women's shoes.
- **SC-4 [CAT-001 AC-8, AC-9]:** `%` and `_` mean themselves: `q=100%25` returns the shoe whose description contains `100%` and no other; `q=_` returns only shoes containing an underscore.
- **SC-5 [CAT-001 AC-10, AC-11, AC-12]:** `category=women,kids` returns only shoes of either category and `total` is their sum; adding `line=samba` returns only Samba shoes of those categories (different parameters combine with "and"); `category=` (empty) answers exactly as no parameter.
- **SC-6 [CAT-001 AC-13, AC-14]:** `price_from=3000000&price_to=3999999` returns the shoes priced 3090000 and 3290000 and not those priced 2790000 or 5940000; a shoe priced exactly 3000000 is returned when both ends are 3000000 (both ends are included; numbers compare as numbers).
- **SC-7 [CAT-001 AC-15, AC-16, AC-17, AC-18]:** `sort=price` rises by price, `sort=-price` falls by price, `sort=-released` puts the most recently released shoe first, `sort=-rating` puts the highest-rated first and shoes with no rating last.
- **SC-8 [CAT-001 AC-19]:** Two shoes with the same price read in pages of one with `sort=price` each appear exactly once across the pages (equal values keep the seed's order).
- **SC-9 [CAT-001 AC-20, AC-21, AC-22, AC-23]:** `size=38` returns only shoes that list `38` and do not list it as sold out — Handball Spezial Earth Strata (38 sold out) is not among them; `size=40` returns Samba OG Black Crystal (40 on its last pairs, not sold out); `size=38,40` returns a shoe when either size is available.
- **SC-10 [CAT-001 AC-24]:** `category=women&line=samba&size=38&price_to=2999999` returns exactly `samba-og-w-wonder-quartz` and `total` is 1.
- **SC-11 [CAT-001 AC-25]:** A shoe in draft that matches every other condition is not returned when the request carries `status=live`.
- **SC-12 [CAT-001 AC-26]:** With 30 matches, `q=samba&page=2&limit=12` returns 12 items from the second page of matches and `total` 30.
- **SC-13 [CAT-001 AC-27]:** `GET /v1/orders?q=nam` returns orders whose number or customer name contains "nam" and not an order whose address or note contains "nam" while its number and customer do not; a `q` that is only an order's phone digits returns nothing (a private field is never searched). Tested as whoever may read the orders list on main at that time.
- **SC-14 [CAT-001 AC-28]:** `GET /v1/reviews?product=samba-og-w-core-black&sort=-date` returns that shoe's reviews, newest first; `GET /v1/reviews?q=<a reviewer's name | a word of a review | the shoe's name>` returns that review.
- **SC-15 [CAT-001 Invalid input]:** `sort` naming a field the collection does not have, `price_from` or `price_to` that is not a number, a date range holding a value that is not a date, and a `q` of 101 characters each answer 400 with an error body naming the parameter (`sort`, `price_from`, `released_to`, `q`) and no records.
- **SC-16 [CAT-001 Boundary]:** `q=%20%20` answers as no search; `size=99` answers 200 with `total` 0 and `items: []`; `GET /v1/orders?size=<value>` filters orders by their own `size` field exactly.
- **SC-17 [CAT-001 Upstream failure]:** With the database unreachable a query answers 500 with the generic error body: no SQL, no table or column name, no stack trace.

### 8.2 The shop's options and speed (backend)

- **SC-18 [CAT-002 AC-18, AC-19]:** `GET /v1/product-filters` answers `{"colours": [...], "sizes": [...]}`: `colours` is every distinct main colour a live shoe has, in the closed-set order; `sizes` is every size at least one live shoe lists (in stock or not), by value ascending (`36`, `36 2/3`, `37 1/3`, …, `45 1/3`); a draft shoe contributes neither; any query parameter is ignored.
- **SC-19 [CAT-001 Performance, SRS §6]:** With 10,000 shoes, 200 sequential requests carrying a two-word `q`, one `category` filter and a `sort` have a 95th-percentile time of 500 ms or less against the running API.

### 8.3 The shop page reads pages from the API

- **SC-20 [CAT-002 AC-1, AC-4]:** With 30 live shoes `/shop` shows 12 cards, `30 matching` and a "Show more" button; with 12 live shoes it shows 12 cards, `12 matching` and no button.
- **SC-21 [CAT-002 AC-2, AC-3, Boundary]:** With 30 live shoes, pressing "Show more" once shows 24 cards and the count still reads `30 matching`; pressing it twice shows 30 cards and no button; no card appears twice. With 13 matches there are 12 cards and a button, and pressing it adds one card.
- **SC-22 [CAT-002 AC-5]:** In the browser's network log `/shop` with 30 live shoes sends shoe requests that carry `limit=12` and `status=live`; no request asks for more than 100 shoes, and the page makes no more than one request for the lines (not one per card).
- **SC-23 [CAT-002 AC-6, AC-7]:** A shoe set to draft is not among the cards and not in the count on `/shop`, nor on `/shop?q=<its name>`.
- **SC-24 [CAT-002 AC-8, AC-9]:** `/shop` shows the heading "All shoes"; `/shop?q=samba` shows `Results for “samba”`.
- **SC-25 [CAT-002 AC-10, AC-11, AC-12]:** `/shop?category=women` shows the heading "Women" and the Women category's picture; `/shop?category=women&line=samba` shows "Women · Samba"; `/shop?category=women,men` shows "All shoes" and the default picture.
- **SC-26 [CAT-002 AC-13]:** The card for Samba OG Black Crystal shows `Samba OG Black Crystal`, `3.290.000 ₫` and `Samba · 8 sizes in stock`, and links to `/shop/samba-og-w-core-black`.
- **SC-27 [CAT-002 AC-8, AC-17]:** The sort control, labelled "Sort", reads "Newest first" by default and offers exactly "Newest first", "Price, low to high", "Price, high to low" and "Best rated".
- **SC-28 [CAT-002 AC-14, AC-15, AC-16]:** Choosing "Price, low to high" puts the cards in rising price order and the address carries `sort=price`; "Best rated" puts the highest-rated shoe first; "Newest first" removes `sort` from the address and puts the newest release first.
- **SC-29 [CAT-002 AC-18, AC-19, AC-20]:** The Size (EU) group has one pill per size any live shoe lists, from `35 1/2` to `45 1/3` in growing order, and the Colour group one option per colour a live shoe has, named as in the site's colour names; on `/shop?category=women` both groups still list the colours and sizes of every live shoe.
- **SC-30 [CAT-002 AC-21]:** On `/shop?category=women&size=38` the count equals the `total` of `GET /v1/products?status=live&category=women&size=38`, and every card shown has `38` available.
- **SC-31 [CAT-002 Not found]:** `/shop?line=zzz` shows an empty list with the count `0 matching` and no error message.
- **SC-32 [CAT-002 Conflict]:** When a filter is changed while an earlier request is still in flight, the newest request's results are shown; an older answer arriving later never replaces them.

### 8.4 What the visitor sets reaches the request (address bar)

- **SC-33 [CAT-003 AC-1, AC-2, AC-3, AC-20]:** Ticking "Women" under For puts `category=women` in the address and lists women's shoes; ticking "Men" as well makes it `category=women,men` and lists shoes of either; ticking the line "Samba" with Women ticked lists only women's Samba shoes (a group matches any of its ticks, different groups all of them); after any tick the list is back to its first twelve with no earlier "Show more" pages.
- **SC-34 [CAT-003 AC-4]:** Pressing the size pill `38` puts `size=38` in the address, leaves only shoes with `38` available, shows the pill as pressed, and does not change how many cards the list shows (`size` is never read as a page size on `/shop`).
- **SC-35 [CAT-003 AC-5, AC-7]:** With Women, Samba and size 38 set, ticking "Under 3.000.000 ₫" lists exactly `samba-og-w-wonder-quartz` with the count `1 matching`; ticking "3.000.000 ₫ – 4.000.000 ₫" alone lists a shoe priced 3.290.000 ₫ and not one priced 2.790.000 ₫.
- **SC-36 [CAT-003 AC-6]:** Ticking "Under 3.000.000 ₫" and "4.000.000 ₫ and over" together lists shoes priced below 3.000.000 ₫ or from 4.000.000 ₫, with a `total` that counts only those, and none priced from 3.000.000 ₫ to 3.999.999 ₫.
- **SC-37 [CAT-003 AC-8]:** The address of a list with Women, Samba, size 38 and a price band set, opened in a new window, ticks the same boxes and pill and shows the same list.
- **SC-38 [CAT-003 AC-10, AC-11, AC-12]:** Typing `samba og` in the search box and pressing Enter puts `q=samba%20og` in the address, the heading reads `Results for “samba og”` and the box still shows `samba og`; clearing the box and pressing Enter removes `q` and the heading reads "All shoes"; searching `gazelle` from the header lands on `/shop?q=gazelle` with the Gazelle shoes.
- **SC-39 [CAT-003 Invalid input]:** `/shop?price=abc` lists as if no price were set; `/shop?sort=zzz` lists in the default order and the sort control reads "Newest first".
- **SC-40 [CAT-004 AC-7, AC-9, narrowed to "not shown"]:** While the first request is held for 3 seconds, and after it fails (a 500 or the browser offline), the heading "No pair matches that." is not shown. The loading and error views themselves are plan item 15.

## 9. Dependencies

| Needs | Why |
|---|---|
| Plan item 3, "Site data from the database" (must land first) | The list endpoint and its envelope, the ten tables and the seed, `doi_fold(text)` (created by that story's `helpers_and_rate_limits` migration), `src/lib/api.ts`, and `src/data/index.ts` calling the API with `loadMore`, `hasMore`, `loading`, `error` and "the latest query wins" (DATA-005 AC-10) |
| Plan item 5, "Checkout and placing an order" (parallel, order matters at merge) | It moves a shoe's `sizes`, `sizes_low`, `sizes_out` from columns on `products` into per-size counts (`product_sizes`, view `product_stock`) and drops the columns. The size condition (SC-9, SC-10, SC-30) must be written against whichever form is on main when this story is built; whichever story merges second updates the other's query. TL sequences the two |
| Plan item 15, "Shop phone filters, Clear all, loading and errors" | Built after this story; adds the states this story deliberately leaves out |
| Data for Test | The seed holds 12 placeholder shoes. Criteria that need 30 live shoes, a draft shoe, `100%` and `_` text, "Cửa Lò", "Đại" or 10,000 shoes use fixtures the tests insert into their own database (`TEST_DATABASE_URL`) — never the seed |
| External accounts, keys, provider setup | None |

## 10. Decisions this story leaves to TL (technical design, step 6)

These are not questions for the owner; they are the design the SRS names as TL's.

1. **How one request carries a union of price bands.** `price_from` / `price_to` hold one range; "Under 3.000.000 ₫" plus "4.000.000 ₫ and over" are two (SRS §7). `services.md` §2.4 says nothing about it; TL extends it. `Filter` in `src/data/index.ts` cannot express it either, so this story may change that module for this one purpose, without changing any exported name or signature (DATA-005 item 1). Touch nothing else there unless the data story left another part of this query out of the request.
2. **`size` on `/shop`.** The address keeps `size=<eu size>` (SC-34). The shop's own request is always `limit=12`, so the shop never reads `size` as a page size. `src/lib/table.ts` is a scaffold file, and the dealer's lists rely on `size` as page size; the way "Clear all" later removes the size is decided in plan item 15, but this story must not make it harder.
3. **One lookup per line, not one per card.** `ProductCard` asks for its line by id. If the data module does not cache record reads, a page of twelve cards makes twelve requests (SC-22 would fail). TL decides between a cache in the data module and passing the shop's already-loaded lines to the cards.
4. **Index or scan.** `erd.md` D6 chose a scan with `doi_fold` and `strpos` and no migration, good to about 20,000 shoes; SC-19 is the test that decides whether that holds at 10,000 shoes and a 500 ms p95.
5. **Where `status=live` is enforced.** The shop page sends it (SC-22, SC-23); the API refusing drafts to a visitor whatever is asked is ADMIN-013.

## 11. Requirement coverage

| Requirement | Where it is proven |
|---|---|
| CAT-001 items 1–9 | SC-1 to SC-19 (item 9: SC-11; the API-side guarantee is ADMIN-013) |
| CAT-002 items 1–8 | SC-18, SC-20 to SC-32 |
| CAT-003 items 1–5 | SC-33 to SC-39 |
| CAT-003 items 6–8 | Out of scope here — plan item 15 |
| CAT-004 items 1–5 | Out of scope here — plan item 15; SC-40 guards only the "not shown" half of AC-7 and AC-9 |
