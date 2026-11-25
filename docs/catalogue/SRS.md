# SRS — Search, filters and sorting served by the API

Module: `catalogue`
Design: [View the approved design](http://localhost:9401)
Design system: `design/design-system.md`

> One file per module. This one covers the shop page `/shop` and the query language the list endpoints
> answer, which the header search, the home page links and the dealer's lists all use.

## 1. Purpose

A customer who knows what they want — "women's Samba in EU 38 under 3 million" — must reach it in a few taps,
on a phone, and be able to send that list to a friend as a link. This module makes the shop page read filtered,
sorted, paged results from the API instead of loading every shoe into the browser, and makes the list endpoints
answer free-text search, "any of" filters, ranges, sorting and an EU-size filter. Without it, search and filters
stop working as soon as the records sit behind a paged API of at most 100 records a page.

## 2. Actors

| Actor | Who they are | What they may do in this module |
|---|---|---|
| Visitor | Anyone with the address | Search, filter, sort and page the shoes |
| Dealer | The shop owner or staff, signed in | Use the same query language on the dealer's lists (`admin` module), including shoes in draft and hidden reviews |

## 3. Scope

**In scope** — the plan item, by title:

- Search, filters and sorting served by the API

**Out of scope**

- Storing and seeding the records, the paged envelope and exact-value filters — belong to the `data` module.
- The header's search button and menu — belong to the `site` module (SITE-001); this module covers where they land.
- What the dealer's lists show and how a dealer saves — belongs to the `admin` module.
- Showing which sizes are in stock on a shoe's own page — belongs to the `site` module (SITE-005).
- Search suggestions or search-as-you-type — not in the design; the search runs when the visitor presses Enter.

## 4. Functional requirements

### 4.1 Search, filters and sorting served by the API

**Requirement CAT-001 — The list endpoints answer search, "any of", ranges, sort and size**

*As a* visitor, *I want* the shop to answer a combined question in one request, *so that* I get exactly the matching pairs and the true count.

Behaviour:

1. `q` is free text. Case and accents are ignored, and `đ` is read as `d`, so `cua lo` finds "Cửa Lò". The text is split into words; a record matches when every word is found, as part of a value, in at least one of the searched fields. `samba og` and `og samba` both find "Samba OG".
2. For shoes the searched fields are name, model, colourway, style code, description, the line's name and the category's name. For orders: order number and customer name. For reviews: the customer's name, the review text and the shoe's name. For lines and categories: name and one-line blurb. For every other collection: all its text fields. A private field (DATA-004) is never searched.
3. `<field>=a,b` keeps records whose field equals `a` or `b`. Different parameters are combined with "and". An empty value is ignored.
4. `<field>_from` and `<field>_to` keep records whose field is at least, or at most, the value; both ends are included. Numbers compare as numbers (`price`, `rating`, `total`); dates written as `2026-11-03` compare as dates.
5. `sort` is a field name for ascending order or `-` and a field name for descending: for example `price`, `-price`, `-released`, `-rating`. Any field of the collection may be sorted. Records equal on the sorted field keep a fixed order (the seed's order), so a page never repeats or skips a record. A record with no value for the field comes last.
6. `size=a,b` applies to shoes only: it keeps the shoes in which at least one of the given EU sizes is available — that is, listed in the shoe's sizes and not in its sold-out list. A size on its last pairs is still available.
7. `total` is the number of every match across all pages, not the page's length.
8. The conditions of one request apply together: `category=women&line=samba&size=38&price_to=2999999&q=og&sort=price`.
9. A shoe of status "draft" and a review of status "hidden" are kept out of the results by the `status=live` condition the pages send. That the API never shows them to a visitor, whatever is asked, belongs to the `admin` module (ADMIN-013).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The seeded shoes | `GET /v1/products?q=samba%20og` | Every item's name, model or description contains "samba" and "og"; the Samba OG shoes are among them; `total` equals the number of items across all pages |
| AC-2 | The same | `GET /v1/products?q=og%20samba` | The same set as AC-1 |
| AC-3 | A shoe whose description contains "Cửa Lò" | `GET /v1/products?q=cua%20lo` | The shoe is among the items |
| AC-4 | The same | `GET /v1/products?q=CUA%20LÒ` | The shoe is among the items |
| AC-5 | A shoe whose name contains "Đại" | `GET /v1/products?q=dai` | The shoe is among the items |
| AC-6 | A line named "Gazelle" | `GET /v1/products?q=gazelle` | Every shoe of that line is among the items, though "gazelle" is in the line's name and not the shoe's id |
| AC-7 | Category `women` named "Women" | `GET /v1/products?q=women` | The women's shoes are among the items |
| AC-8 | A shoe whose description contains `100%` | `GET /v1/products?q=100%25` | That shoe is returned and no other; `%` is read as itself |
| AC-9 | A shoe whose name contains `_` | `GET /v1/products?q=_` | Only shoes containing an underscore are returned; `_` is read as itself |
| AC-10 | The seeded shoes | `GET /v1/products?category=women,kids` | Every item's category is `women` or `kids`; `total` is their sum |
| AC-11 | The same | `GET /v1/products?category=women,kids&line=samba` | Every item is a Samba in `women` or `kids` |
| AC-12 | The same | `GET /v1/products?category=` | The answer equals `GET /v1/products` |
| AC-13 | Shoes priced 2790000, 3090000, 3290000, 5940000 | `GET /v1/products?price_from=3000000&price_to=3999999` | The 3090000 and 3290000 shoes only |
| AC-14 | A shoe priced exactly 3000000 | `price_from=3000000` and `price_to=3000000` | The shoe is returned (both ends included) |
| AC-15 | The seeded shoes | `GET /v1/products?sort=price` | Items rise by price |
| AC-16 | The same | `GET /v1/products?sort=-price` | Items fall by price |
| AC-17 | The same | `GET /v1/products?sort=-released` | The most recently released shoe is first |
| AC-18 | The same | `GET /v1/products?sort=-rating` | The highest-rated shoe is first; shoes with no rating are last |
| AC-19 | Two shoes with the same price | `sort=price` pages of 1 are read in turn | Each shoe appears exactly once across the pages |
| AC-20 | The seeded shoes | `GET /v1/products?size=38` | Every item lists `38` in `sizes` and not in `sizes_out` |
| AC-21 | Handball Spezial Earth Strata lists `38` in its sold-out list | `GET /v1/products?size=38` | That shoe is not among the items |
| AC-22 | Samba OG Black Crystal has `40` among its last-pairs sizes and not sold out | `GET /v1/products?size=40` | The shoe is among the items |
| AC-23 | The seeded shoes | `GET /v1/products?size=38,40` | A shoe is returned when it has either size available |
| AC-24 | The seeded shoes | `GET /v1/products?category=women&line=samba&size=38&price_to=2999999` | Exactly `samba-og-w-wonder-quartz`; `total` is 1 |
| AC-25 | A shoe in draft that matches every other condition | `GET /v1/products?q=…&status=live` | The draft shoe is not returned |
| AC-26 | 30 matches | `GET /v1/products?q=samba&page=2&limit=12` | 12 items from the second page of matches, and `total` 30 |
| AC-27 | The seeded orders | `GET /v1/orders?q=nam` | Orders whose number or customer contains "nam" are returned; an order whose address or note contains "nam" but whose number and customer do not is not |
| AC-28 | The seeded reviews | `GET /v1/reviews?product=samba-og-w-core-black&sort=-date` | That shoe's reviews, newest first |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | `sort` names a field the collection does not have | 400 naming `sort` |
| Invalid input | `price_from` or `price_to` is not a number, or a date range holds a value that is not a date | 400 naming the parameter |
| Invalid input | `q` is longer than 100 characters | 400 naming `q` |
| Boundary | `size` is used on a collection that is not shoes | The parameter is an ordinary field filter: orders have a `size` field and are filtered by it exactly; a collection with no such field ignores it |
| Boundary | `q` is only spaces | Treated as no search |
| Boundary | A size that no shoe has (`size=99`) | 200 with `total` 0 and `items: []`, not an error |
| Permission | A search term matches only a private field (an order's phone) | No match; private fields are never searched (AC-27) |
| Upstream failure | The database cannot be reached | 500 with a generic error body |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoe: name, model, colourway, style code, description, line, category, price, colour, released, rating, sizes, sold-out sizes, status | read | yes | Searched, filtered or sorted as above |
| Order: number, customer | read | yes | Searched |
| Review: customer, text, shoe | read | yes | Searched |

**Requirement CAT-002 — The shop page shows filtered, sorted pages with "Show more"**

*As a* visitor, *I want* the shop page to show the matching pairs a dozen at a time with the true count, *so that* it stays quick on a phone.

Behaviour:

1. `/shop` asks the API for the first twelve live shoes that match what the visitor set, in the chosen order. It does not load every shoe.
2. The heading reads "All shoes". With search words it reads `Results for “<words>”`. With no search words and exactly one category and/or exactly one line chosen, it reads their names joined by " · " (for example "Women · Samba"). The picture beside it is that category's picture when exactly one category is chosen, otherwise the shop's default picture.
3. Under the heading the line `<N> matching` shows the API's `total`, which is every match, not the twelve on screen.
4. Each result is a shoe card: its picture (the pair, and its second picture under the pointer), the name in capitals with the price, and `<line name> · <n> sizes in stock`, linking to the shoe's page. Cards are in a grid up to three across.
5. When more matches exist than are shown, a "Show more" button follows the grid. Pressing it asks for the next twelve and adds them below those already shown; it is not drawn once every match is shown.
6. Only live shoes are listed. A shoe set to draft by the dealer does not appear in the list, in search, or in any count.
7. The sort control, labelled "Sort", offers "Newest first" (the default), "Price, low to high", "Price, high to low" and "Best rated", which sort by release date (newest first), price, price (descending) and rating (highest first).
8. The Colour options list every main colour that at least one live shoe has, and the Size (EU) pills list every size that at least one live shoe lists, in growing order (`36`, `36 2/3`, `37 1/3`, …), whatever the other filters are.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | 30 live shoes | `/shop` loads | 12 cards are shown, the count reads `30 matching`, and a "Show more" button is shown |
| AC-2 | The same | The visitor presses "Show more" | 24 cards are shown and the count still reads `30 matching` |
| AC-3 | The same | The visitor presses "Show more" twice | 30 cards are shown and no "Show more" button is shown |
| AC-4 | 12 live shoes | `/shop` loads | 12 cards, `12 matching`, no "Show more" |
| AC-5 | A browser network log | `/shop` loads with 30 live shoes | The shoe requests carry `limit=12` and `status=live`; no request asks for more than 100 shoes |
| AC-6 | A shoe set to draft | `/shop` loads | The shoe is not among the cards and not in the count |
| AC-7 | A shoe set to draft that would match the search words | `/shop?q=<its name>` loads | The shoe is not listed |
| AC-8 | Nothing chosen | `/shop` loads | The heading reads "All shoes" and the sort control reads "Newest first" |
| AC-9 | `/shop?q=samba` | The page loads | The heading reads `Results for “samba”` |
| AC-10 | `/shop?category=women` | The page loads | The heading reads "Women" and the picture beside it is the Women category's picture |
| AC-11 | `/shop?category=women&line=samba` | The page loads | The heading reads "Women · Samba" |
| AC-12 | `/shop?category=women,men` | The page loads | The heading reads "All shoes" and the picture is the default one |
| AC-13 | A card for "Samba OG Black Crystal", price 3.290.000 ₫, line Samba, 8 sizes in stock | The page loads | The card shows `Samba OG Black Crystal`, `3.290.000 ₫` and `Samba · 8 sizes in stock` and links to `/shop/samba-og-w-core-black` |
| AC-14 | The sort control | The visitor chooses "Price, low to high" | The cards rise by price and the address carries `sort=price` |
| AC-15 | The sort control | The visitor chooses "Best rated" | The highest-rated shoe is first |
| AC-16 | The sort control | The visitor chooses "Newest first" | The address carries no `sort` and the newest release is first |
| AC-17 | The sort control | The visitor reads the options | They are "Newest first", "Price, low to high", "Price, high to low", "Best rated" |
| AC-18 | Live shoes list sizes from 35 1/2 to 45 1/3 | The visitor reads the Size (EU) pills | One pill per size any live shoe lists, from `35 1/2` to `45 1/3`, in growing order |
| AC-19 | The seeded live shoes use nine colours | The visitor reads the Colour group | One option per colour that a live shoe has, named as in the site's colour names (for example "Black") |
| AC-20 | `/shop?category=women` | The visitor reads the Colour and Size groups | They still list the colours and sizes of every live shoe, not only the women's |
| AC-21 | `/shop?category=women&size=38` | The page loads | The count equals the API's `total` for the same filters, and each card shown has `38` available |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | The address names a category, line or colour that does not exist (`?line=zzz`) | The list is empty: the empty state of CAT-004 is shown with `0 matching`; no error |
| Conflict | The visitor changes a filter while a request is in flight | The newest request's results are shown; an older answer arriving later never replaces them |
| Boundary | Exactly 12 matches | No "Show more" |
| Boundary | 13 matches | 12 cards and "Show more"; pressing it adds one card |
| Upstream failure | A request fails | CAT-004 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Live shoes: id, name, price, pictures, line, category, colour, sizes, sold-out sizes, rating, release date | read | yes | Twelve per request |
| Categories, lines | read | yes | Names for the filter groups and the heading |

**Requirement CAT-003 — Everything the visitor sets lives in the address bar**

*As a* visitor, *I want* the filters, search words and order to be in the address, *so that* I can send a list as a link and come back to it from a shoe.

Behaviour:

1. The shop's address holds: `category`, `line`, `colour`, `price` (band names), `size` as comma-separated lists; `q` for search words; `sort` for the order (left out for "Newest first"). A visitor who opens the address sees the same list.
2. The Filter rail has four groups of tick boxes — For (the categories), Line, Colour, Price — and one group of size pills, "Size (EU)". Ticking a box or a pill adds its value to the address and returns the list to its first twelve.
3. The price bands are "Under 3.000.000 ₫" (up to 2.999.999 ₫), "3.000.000 ₫ – 4.000.000 ₫" (3.000.000 ₫ to 3.999.999 ₫) and "4.000.000 ₫ and over". Ticking more than one band shows shoes in any of the ticked bands and none from a band left unticked.
4. Ticking several boxes of one group shows shoes matching any of them; ticking in different groups shows shoes matching all groups.
5. The search box holds the typed words and runs on Enter or when the form is submitted; words are trimmed; empty words remove the search. A search from the header lands here with the words in the box.
6. On a phone (below 1024px) the rail is hidden and a "Filter" button opens and closes it; the button shows the number of active filters in brackets when there are any, for example "Filter (3)". From 1024px the rail is always shown at the left and the button is not.
7. Opening a shoe and pressing the browser's Back returns to the same list with the same filters, order and number of cards shown being the first twelve.
8. Entries into the shop: the header's search, "All shoes", Women/Men/Kids links, a home-page category tile (`?category=<id>`), a line row (`?line=<id>`) and "Gifts" (`?price=under-3m`) land on the filtered list; the matching tick is shown.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | `/shop` | The visitor ticks "Women" under For | The address carries `category=women` and the list shows women's shoes |
| AC-2 | The same | The visitor ticks "Men" as well | The address carries `category=women,men` and the list shows shoes of either |
| AC-3 | Women ticked | The visitor ticks the line "Samba" | The list shows only women's Samba shoes |
| AC-4 | The same | The visitor presses the size pill `38` | The address carries `size=38`; only shoes with `38` available remain; the pill is pressed |
| AC-5 | Women, Samba, size 38 | The visitor ticks the price band "Under 3.000.000 ₫" | The list shows exactly `samba-og-w-wonder-quartz`, and the count reads `1 matching` |
| AC-6 | The seeded shoes | The visitor ticks "Under 3.000.000 ₫" and "4.000.000 ₫ and over" | Shoes priced below 3.000.000 ₫ or at least 4.000.000 ₫ are shown; none priced from 3.000.000 ₫ to 3.999.999 ₫ is shown |
| AC-7 | The seeded shoes | The visitor ticks "3.000.000 ₫ – 4.000.000 ₫" | A shoe priced 3.290.000 ₫ is shown and one priced 2.790.000 ₫ is not |
| AC-8 | Women, Samba, size 38 and a price band set | The address is copied and opened in a new window | The same filters are ticked and the same list is shown |
| AC-9 | A list with two filters | The visitor opens a shoe and presses Back | The same two filters are still set and the same list is shown |
| AC-10 | The search box | The visitor types `samba og` and presses Enter | The address carries `q=samba%20og`, the heading reads `Results for “samba og”`, and the box still shows `samba og` |
| AC-11 | A search is set | The visitor clears the box and presses Enter | The address carries no `q` and the heading reads "All shoes" |
| AC-12 | The header search | The visitor searches `gazelle` from the home page | `/shop?q=gazelle` shows the Gazelle shoes |
| AC-13 | A 390px window | `/shop` loads | The rail is hidden and a "Filter" button is shown |
| AC-14 | The same, with 3 filters active | The visitor reads the button | It reads "Filter (3)" |
| AC-15 | The same | The visitor presses the button | The rail opens; pressing again closes it |
| AC-16 | A 1280px window | `/shop` loads | The rail is shown and no "Filter" button is shown |
| AC-17 | Home page | The visitor presses the "Kids" tile | `/shop?category=kids` shows kids' shoes with "Kids" ticked |
| AC-18 | Home page | The visitor presses the "Gifts" tile | `/shop?price=under-3m` shows shoes below 3.000.000 ₫ with "Under 3.000.000 ₫" ticked |
| AC-19 | Footer | The visitor presses "Gifts" | The same as AC-18 |
| AC-20 | A filter is ticked | The visitor looks at the page | The list returns to the first twelve (no earlier "Show more" pages remain) |
| AC-21 | A group with a single option | The rail renders | That group is not drawn |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | The address carries an unknown price band (`?price=abc`) | The band is ignored and the list is as if no price were set |
| Invalid input | The address carries an unknown sort (`?sort=zzz`) | The list is shown in the default order and the control reads "Newest first" |
| Conflict | Two filters are changed within one second | The final address and list reflect both, not only the first |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Address parameters: category, line, colour, price, size, q, sort | read / write | no | Kept in the address only; nothing stored on the server |

**Requirement CAT-004 — Clear, empty, loading and error states of the shop page**

*As a* visitor, *I want* to be told when nothing matches, when the list is loading and when it failed, *so that* I am never left looking at a blank page.

Behaviour:

1. "Clear all" removes every filter, the search words and the size pills, and keeps the order; it is shown under the rail when anything is set. After it the list is the unfiltered shop.
2. When no live shoe matches, and nothing is loading, the results area shows the heading "No pair matches that.", the line "Try another size or fewer filters — or ask us, and we will tell you when it comes in.", and a "Clear all" button with an arrow that does the same as item 1. "0 matching" is the count.
3. While the first page of a list is being fetched, the results area shows the status "Loading the shoes…" and the empty-state heading is not shown. While "Show more" is fetching, the button is disabled and reads "Loading…".
4. When the first page cannot be fetched, the results area shows the message "We could not load the shoes." and a "Try again" button that asks again; the empty-state heading is not shown. When "Show more" fails, the shoes already shown stay, the message appears under them, and "Try again" asks for the same page.
5. The loading and error states are not drawn in the approved design; items 3 and 4 are required by the plan item and use the empty state's own type and spacing (open question 1).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Women, Samba, size 38 and a band set | The visitor presses "Clear all" under the rail | The address carries none of `category`, `line`, `size`, `price`, `q` and the list is all live shoes |
| AC-2 | Only the size `38` is set | The visitor looks under the rail | A "Clear all" link is shown, and pressing it removes `size` from the address |
| AC-3 | Nothing is set | The visitor looks under the rail | No "Clear all" link is shown |
| AC-4 | A sort is set (`sort=price`) and filters are set | The visitor presses "Clear all" | The filters are gone and `sort=price` is kept |
| AC-5 | `/shop?q=zzzz` matches nothing | The list loads | The heading "No pair matches that.", the line "Try another size or fewer filters — or ask us, and we will tell you when it comes in." and a "Clear all" button are shown; the count reads `0 matching` |
| AC-6 | The empty state is shown | The visitor presses its "Clear all" | The unfiltered list is shown |
| AC-7 | The request is held for 3 seconds | `/shop` loads | During the wait "Loading the shoes…" is shown and "No pair matches that." is not |
| AC-8 | "Show more" is held for 3 seconds | The visitor presses it | The button is disabled and reads "Loading…" until the shoes arrive |
| AC-9 | The shoe request answers 500 | `/shop` loads | "We could not load the shoes." and a "Try again" button are shown; "No pair matches that." is not |
| AC-10 | The same, then the server recovers | The visitor presses "Try again" | The list loads |
| AC-11 | The second page answers 500 | The visitor presses "Show more" | The first twelve cards stay, the error message appears under them, and "Try again" asks for page 2 again |
| AC-12 | The browser is offline | `/shop` loads | The same message and button as AC-9 are shown |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | No match | The empty state (AC-5) |
| Upstream failure | The API is unreachable or answers 5xx | The error message and "Try again" (AC-9, AC-11, AC-12); nothing partly written |
| Conflict | Not applicable: the shop page only reads | — |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Address parameters | write | no | As CAT-003 |

## 5. Screens

| Screen | Section in the design | Functions it serves | States that must exist |
|---|---|---|---|
| Shop `/shop` | Shoes (catalogue): heading and count, search and sort band, filter rail, card grid, "Show more" | CAT-001, CAT-002, CAT-003, CAT-004 | default; empty ("No pair matches that."); phone with the rail closed; phone with the rail open |
| Shop `/shop` — loading, load failed, "Show more" loading, "Show more" failed | Not drawn in the approved design; required by the plan item (CAT-004) | CAT-004 | loading; error; more-loading; more-error |

## 6. Non-functional requirements

| Area | Requirement |
|---|---|
| Performance | With 10,000 shoes seeded, a request carrying `q` (two words), one `category` filter and a `sort` answers within 500 ms at p95 over 200 sequential requests against the running API |
| Performance | The shop page requests no more than 12 shoes at a time for its list |
| Responsive | `/shop` works from a 320px window up with no horizontal page scroll; the rail is behind the "Filter" button below 1024px |
| Accessibility | The search input has a label; the sort control and the "Filter" button are named; the size pills carry their pressed state; the loading and error messages are announced to assistive technology |
| Localisation | Copy is in English; prices in the bands read as `vi-VN` (`3.000.000 ₫`) |

## 7. Dependencies and assumptions

- **Depends on:** the `data` module, for the list endpoint, its envelope and the seeded shoes.
- **Depends on:** the `admin` module for the dealer's side of "draft" and "hidden": this module requires that a visitor's results never contain them; until `admin` lands the shop page itself sends `status=live`.
- **Assumption:** the existing shop page keeps its size filter in an address parameter called `size`, which the table helper in `src/lib/table.ts` also reads as the page size and which its "Clear" keeps. Because of that, "Clear all" today leaves the size set (AC-2 and AC-1 of CAT-004 fail on the current page). The story must make Clear remove the size, for example by giving the size filter its own address parameter; `table.ts` is a scaffold file, so TL decides how.
- **Assumption:** "more than one price band" is a union of ranges, which a single `price_from`/`price_to` pair cannot say; TL decides how a request carries it.
- **Assumption:** the sort fields beyond the four named in the plan (`price`, `-price`, `-released`, `-rating`) are required by the dealer's lists, which sort by column.

| # | Open question | Proposed default | Who decides |
|---|---|---|---|
| 1 | The design draws no loading, load-failed or "Show more" states, but the plan item requires them to be visible | Use the copy in CAT-004 ("Loading the shoes…", "We could not load the shoes.", "Try again", "Loading…") in the empty state's type and spacing, added to the site's content so the owner can edit it | Stakeholder |
| 2 | Words matching: every typed word must be found (so `samba og` finds "Samba OG"), as part of any searched field | As written in CAT-001 | Stakeholder |

## 8. Traceability

| Plan item | Requirement ids | Test cases |
|---|---|---|
| Search, filters and sorting served by the API | CAT-001, CAT-002, CAT-003, CAT-004 | `test-cases/search-filters-and-sorting-served-by-the-api.md` |
