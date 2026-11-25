# Test cases — Shop phone filters, Clear all, loading and errors

Module: `catalogue` · Plan item 16 · Story: `docs/catalogue/stories/shop-phone-filters-clear-all-loading-and-errors.md` (SC-1 … SC-33)
Requirements: CAT-003 items 6–8, CAT-004 in full (`docs/catalogue/SRS.md`) · Contract: none of its own — this story adds no endpoint; it reads `GET /v1/products` and `GET /v1/product-filters` as plan item 4 serves them (`docs/architecture/services.md` §3.2, §3.4)

Written before the implementation exists, from the requirements and the story only.

## 1. Risk and depth

| Part | Risk | Why | Depth |
|---|---|---|---|
| Phone filters: the "Filter" button, its count, the 1024px switch (SC-1 … SC-5) | Medium–High | Most customers shop on a phone; a rail that cannot be opened, or a page that scrolls sideways at 320px, loses the sale. The count in the button is the only sign of what is set while the rail is closed | Every viewport the story names (320, 390, 1023, 1024, 1280), the number in the button for each thing it counts and for each thing it must not count, the open/close state, six page states for horizontal scroll |
| Entries and Back (SC-6 … SC-11) | Medium | A link that lands on an unfiltered list, or a Back that loses the filters, is a broken promise of the address bar | Every entry the story lists (tiles, footer, line row), Back with one card and with many, Back after "Show more" |
| Groups with one option (SC-12, SC-13) | Medium | A heading over nothing is a blank spot; a hidden group whose filter still applies is the one place a customer could be filtering without seeing it | One option, two options (the boundary), a held options request, a filter ticked on a group that is not drawn |
| Clear all (SC-14 … SC-19) | **High** | Today it leaves the size set. It is also the visitor's only way out of a dead end, and the story must not change what the dealer's lists do | Every parameter it removes, each single-parameter case, "keeps the sort", "not shown when nothing is set", the dealer's lists unchanged |
| Empty state (SC-20, SC-21) | Medium | Reached from a typo in the search; the copy is fixed by the owner | Copy word for word, two ways to reach it, its "Clear all" for `q`, `line` and `size` |
| Loading and errors (SC-22 … SC-30) | **High** | They replace a blank page and a false "No pair matches that." with the truth; the first-page and second-page failures, offline, retry, and a stale failure are separate failure paths | Each state arranged for real (delay, 500, offline, recovery), what survives (the first twelve cards, the count), what must not show (the empty state, a count, a server message) |
| Words, access and look (SC-31 … SC-33) | Medium | The owner edits the four strings in the Studio; announcements reach assistive technology; the design's tokens are a CI gate | Exact strings and their content keys, roles and attributes, computed type of the new states against the empty state's |

No case is skipped for risk. The one criterion with no browser reading (the source of the strings and the lint gate) is `manual`, with its reason (§8).

## 2. How to read this file

- **Starting state of every case**: the seed of the `data` story — 12 live shoes, 5 lines (`samba`, `gazelle`, …), 3 categories (`women` 4 shoes, `men` 6, `kids` 2), nine colours, sizes `35 1/2` … `45 1/3` — and no row whose id starts `tc-`. A case that needs more names a fixture from §3 in its `Given`. After a case every fixture is deleted and every seed row it changed is put back, so no case depends on another.
- **Plan item 4 has merged** before this story is built (story §3), so the list reads the API twelve at a time, the address carries `category`, `line`, `colour`, `price`, `size`, `q`, `sort`, and the price bands are `under-3m`, `3m-4m`, `4m-plus`. Where a case needs a request, it is `GET /api/v1/products` as the browser sends it.
- **Viewport** is part of the `Given` of every case that depends on it: *phone* is 390×844, *narrow* is 320×640, *desktop* is 1280×800. Where no viewport is named the case runs at desktop.
- **Reading the page** (names the build carries, not class names):
  - the heading is `main h1`; the count is the element `[data-testid="shop-count"]`, whose text reads `<N> matching`;
  - a **card** is a link inside `main` whose `href` starts `/shop/`; cards are counted by distinct `href`;
  - the **Filter button** is the button in the search and sort band whose visible text starts `Filter`; the **rail** is the element labelled `Filters` (`[aria-label="Filters"]`); a **tick box** is the checkbox named by its label; a **size pill** is the button `[data-testid="filter-size-<size>"]` (for `38`, `filter-size-38`) and carries `aria-pressed`;
  - **Clear all in the rail** is a button named "Clear all" inside the rail; **Clear all in the empty state** is the button named "Clear all" inside `main`, outside the rail, beside the heading "No pair matches that.";
  - the search input is the field labelled `Search by name, line, colour or code`; the sort control is the combobox named `Sort`;
  - the loading line is the element with `role="status"` whose text is "Loading the shoes…"; the error message is the element with `role="alert"` whose text is "We could not load the shoes."; "Show more", "Loading…" and "Try again" are buttons named by their text;
  - text a visitor can read is drawn with the site's content keys, so an element holding it carries `data-edit="<key>"` (the `T` component) — SC-31 reads that attribute.
- **"Address" cases** compare query parameters as a set: `?category=women,men` may be written `category=women%2Cmen`, a space as `%20` or `+`, and parameters may come in any order.
- **A 5xx the page is shown** means the browser's request to `/api/v1/products` answers 500 with the body `{"error":"internal","message":"tc-secret-db-detail","errors":{}}`. The message is a marker: SC-25 proves the visitor never sees it.
- **Dealer reads** (SC-19) are made signed in through `/admin/sign-in` with the dealer account of the test environment (staff or admin; both reach the products list).

## 3. Fixtures

Inserted with `set_db_state` (SQL against the test database, never the seed file); columns not listed take their defaults. Sizes and stock are written in whichever form `main` holds when the case runs (the `sizes` / `sizes_low` / `sizes_out` columns, or `product_sizes` rows). A size that is "listed" has an entry; "sold out" means 0 pairs.

| Id | What is inserted or changed | Used by |
|---|---|---|
| FX-LIVE30 | 18 live shoes `tc-live-01` … `tc-live-18`: name "Pair Tc 01" … "Pair Tc 18", line `samba`, category `women`, colour `black`, price 3500000, released 2026-06-01 … 2026-06-18, sizes `38` and `40` listed and in stock. With the seed: **30 live shoes**; women 22; size 38 available 23; band `3m-4m` 23; `under-3m` still 3 | SC-11, SC-14a, SC-19c, SC-24, SC-27, SC-28b |
| FX-LIVE60 | Same as FX-LIVE30 but 48 shoes `tc-live-01` … `tc-live-48`. With the seed: **60 live shoes** | SC-19b |
| FX-ONECOL | Every shoe, live or not, is changed to colour `black` and lists only EU size `40`, in stock. The live catalogue then has one colour and one size; the three categories, five lines and three price bands are untouched | SC-12a, SC-13a, SC-13b |
| FX-TWOCOL | FX-ONECOL, then `samba-og-w-core-black` is changed to colour `white` and lists sizes `40` and `42`. Two colours, two sizes | SC-12b |

## 4. Arrangements the cases rely on

The product carries no switch for delay or failure (story §9). A case arranges them from outside. Each arrangement has its primary means; where the instrument has no such means, the case — or the named part of it — is reported **UNVERIFIED** naming the instrument that was missing, never passed on a weaker reading and never failed.

| Arrangement | Meaning | Primary means | If it cannot be made |
|---|---|---|---|
| HOLD(*n*) | The matching request is kept by the browser's harness for *n* seconds before it is sent on, then answered normally | `interact_page` with request interception on `/api/v1/products` (or `/api/v1/product-filters`) | SC-12c, SC-22, SC-23, SC-24 are UNVERIFIED: no interception in `interact_page` |
| FAIL500 | Every matching request is answered 500 with the marker body of §2, without reaching the API; the interception can be lifted during the case so "the server answers again" | `interact_page` with request interception | SC-25a, SC-26, SC-27, SC-29, SC-30 are UNVERIFIED; SC-25b (database taken away) still observes the first-page error, and SC-28a (backend stopped, gateway 502) the same state |
| FAIL500-PAGE2 | As FAIL500, only for requests that carry `page=2` | `interact_page` with request interception | SC-27, SC-28b UNVERIFIED |
| OFFLINE | The browser has no network. The page shell is loaded first (a single-page app cannot load its own HTML offline), then the network is switched off, then the visitor follows an in-app link to the case's address; where "offline" cannot be switched, every `/api/` request is aborted (the browser's own failure for an unreachable server) | `interact_page` network offline / abort | SC-28a falls back to the backend stopped (gateway 502); SC-28b UNVERIFIED |
| DB-DOWN | The backend stays up and its database refuses connections, so every API read answers 500 `internal` | `set_db_available` (off, then on again) | SC-25b UNVERIFIED |

## 5. Open points found while writing (reported, not resolved here)

1. **"Filter (n)" spacing.** SC-2 asks for a space before the bracket. The button's text must read `Filter (3)` — the space in the text, not only a flex gap between two elements — so the accessible name and the visible text normalised for whitespace both read `Filter (3)`. The cases read both.
2. **Rail contents on a phone.** `design/design-system.md` §2.1.14 says the groups fold behind their headings below 1024px; story SC-3 says the opened rail shows the groups and a tick box can be ticked in it. The cases follow the story: once the Filter button has opened the rail, the group headings and the tick boxes are visible and usable without a second press.
3. **"Under the rail".** Story SC-14 to SC-17 say "Clear all" is shown "under the rail". The cases only require that it is inside the rail (`[aria-label="Filters"]`) and absent when nothing is set; where inside is not pinned.
4. **Unknown address values and "Clear all".** `/shop?price=abc` is "as if no price were set" (CAT-003), but no criterion says whether "Clear all" is then shown. No case pins it.
5. **A server message never reaches the page.** SC-25 lists "no status code, error code or server message". The cases use a marker message in the 500 body and read the page text for it, for `500` and for `internal`.
6. **Dealer lists need more than the seed to show 50 rows.** The seed has 12 shoes, so a page of 50 and a page of 20 look alike. SC-19b uses FX-LIVE60 (60 live shoes): 50 rows on the first page and 10 on the second proves `size=50` is still the page size.
7. **Plan numbering.** The story calls this plan item 16 and says the story of item 4 calls it "15". Nothing in the cases depends on it.

## 6. Phone filters (CAT-003 item 6) — SC-1 … SC-5

**Scenario**: SC-1a — phone: the filter rail is not on screen and not read out
**Given**: The seed. Viewport phone (390×844).
**When**: Open `/shop` and measure the rail `[aria-label="Filters"]` and the element that wraps it.
**Then**: The rail has no box: computed `display` is `none` on it or on an ancestor, and its width and height are 0, so it is also absent from the accessibility tree. A tick box named "Women" and the size pill `38` are not rendered (no box).
Traces: SC-1 (CAT-003 AC-13, item 6)
Check: measure_styles

**Scenario**: SC-1b — phone: a "Filter" button without brackets, and the first card without pressing anything
**Given**: The seed. Viewport phone (390×844). Nothing set.
**When**: Open `/shop`; read the button in the search and sort band; read the first card.
**Then**: A button whose text reads exactly `Filter` (no brackets, no digit) is in the same band as the search input and the sort control, with `aria-expanded="false"`. The first card (the first link whose `href` starts `/shop/`) has a box (not hidden); its top is below the bottom of the Filter button; no visible checkbox exists between the band and that card.
Traces: SC-1 (CAT-003 AC-13)
Check: interact_page

**Scenario**: SC-2a — phone: three filters read "Filter (3)"
**Given**: The seed. Viewport phone.
**When**: Open `/shop?category=women&line=samba&size=38`; read the Filter button.
**Then**: The button's accessible name and its visible text, whitespace collapsed, both read exactly `Filter (3)` — one space between `Filter` and `(`, not `Filter(3)`.
Traces: SC-2 (CAT-003 AC-14)
Check: interact_page

**Scenario**: SC-2b — phone: each tick and pill counts one; search words and sort do not
**Given**: The seed. Viewport phone.
**When**: Open, one after the other, `/shop?category=women,men`; `/shop?price=under-3m,4m-plus&colour=black,white`; `/shop?q=samba&sort=price`; `/shop?q=samba&sort=price&category=women`; read the Filter button each time.
**Then**: In that order the button reads `Filter (2)`; `Filter (4)` (two bands and two colours); `Filter` with no brackets; `Filter (1)`.
Traces: SC-2 (CAT-003 AC-14; story §7 "what active filters counts")
Check: interact_page

**Scenario**: SC-2c — phone: "Clear all" brings the button back to "Filter"
**Given**: The seed. Viewport phone. `/shop?category=women&line=samba&size=38` open; the button reads `Filter (3)`.
**When**: Press "Filter" to open the rail; press "Clear all" in the rail; read the button and the address.
**Then**: The button reads exactly `Filter`; the address carries none of `category`, `line`, `size`.
Traces: SC-2 (CAT-003 AC-14)
Check: interact_page

**Scenario**: SC-3a — phone: the button opens and closes the rail and reports its state
**Given**: The seed. Viewport phone. `/shop?category=women` open (the rail closed).
**When**: Press "Filter"; read the button and the rail; press "Filter" again; read both again. Then open `/shop` (nothing set), press "Filter" and read the rail.
**Then**: After the first press the button has `aria-expanded="true"` and the rail has a box; its headings read `For`, `Line`, `Colour`, `Price` and `Size (EU)`; the size pills are shown (16 on the seed, `35 1/2` … `45 1/3`); the tick box "Women" is visible and checked; a button "Clear all" is in the rail. After the second press `aria-expanded="false"` and the rail has no box. With nothing set the opened rail shows the same groups and has **no** "Clear all".
Traces: SC-3 (CAT-003 AC-15, item 6)
Check: interact_page

**Scenario**: SC-3b — phone: ticking while the rail is open leaves it open and updates the number
**Given**: The seed. Viewport phone. `/shop` open; press "Filter" (the rail is open).
**When**: Tick "Women"; read the button and the rail; press the size pill `38`; read again.
**Then**: After "Women": `aria-expanded` is still `true`, the rail still has a box, the address carries `category=women`, the button reads `Filter (1)`. After the pill: still open, the pill is `aria-pressed="true"`, the address carries `size=38`, the button reads `Filter (2)`.
Traces: SC-3 (CAT-003 AC-15)
Check: interact_page

**Scenario**: SC-4a — desktop: the rail is shown with all its groups and there is no Filter button
**Given**: The seed. Viewport desktop (1280×800).
**When**: Open `/shop`; read the visible text.
**Then**: The group headings `For`, `Line`, `Colour`, `Price` and `Size (EU)` are visible; the tick boxes "Women", "Men", "Kids" and the pills `35 1/2` … `45 1/3` are visible; no button named "Filter" is visible.
Traces: SC-4 (CAT-003 AC-16)
Check: render_url

**Scenario**: SC-4b — the switch is at 1024px
**Given**: The seed.
**When**: Open `/shop` at widths 1280, 1024, 1023 and 390 (height 800) and measure the Filter button and the rail `[aria-label="Filters"]` each time.
**Then**: At 1280 and at 1024 the Filter button's computed `display` is `none` and the rail has a box (width above 0, `display` not `none`) at the left of the cards. At 1023 and at 390 the button's `display` is not `none` and the rail has no box.
Traces: SC-4 (CAT-003 AC-16; story §9.1 "the switch is at 1024px")
Check: measure_styles

**Scenario**: SC-5a — no horizontal scroll in the default list at 320 and 390
**Given**: The seed. Viewport narrow (320×640), then phone (390×844).
**When**: Open `/shop`; wait for the cards; scroll the window right by 100px; read `window.scrollX` and `document.documentElement.scrollWidth` against the window's inner width.
**Then**: At both widths `window.scrollX` is 0 after the scroll and `scrollWidth` is not greater than the inner width.
Traces: SC-5 (SRS §6 Responsive)
Check: interact_page

**Scenario**: SC-5b — no horizontal scroll with the rail open and every group drawn
**Given**: The seed. Viewport narrow, then phone.
**When**: Open `/shop?category=women&size=38`; press "Filter"; scroll right by 100px; read `scrollX` and `scrollWidth`.
**Then**: At both widths the rail is open (`aria-expanded="true"`), all five groups and the pills are visible, `scrollX` stays 0 and `scrollWidth` does not exceed the inner width.
Traces: SC-5 (SRS §6 Responsive)
Check: interact_page

**Scenario**: SC-5c — no horizontal scroll in the empty state
**Given**: The seed. Viewport narrow, then phone.
**When**: Open `/shop?q=zzzz`; scroll right by 100px; read `scrollX` and `scrollWidth`.
**Then**: The heading "No pair matches that." is visible; at both widths `scrollX` stays 0 and `scrollWidth` does not exceed the inner width.
Traces: SC-5 (SRS §6 Responsive)
Check: interact_page

**Scenario**: SC-5d — no horizontal scroll while loading, after a failure and after a failed "Show more"
**Given**: The seed plus FX-LIVE30. Viewport narrow, then phone. Arrangements HOLD(3) on the first products request, FAIL500, and FAIL500-PAGE2.
**When**: (1) Open `/shop` with the first request held; during the wait read the page. (2) Open `/shop` with FAIL500; when "We could not load the shoes." is shown read the page. (3) Open `/shop` normally, lift nothing, set FAIL500-PAGE2 and press "Show more"; when the message shows under the cards read the page. Each reading: scroll right by 100px; read `scrollX` and `scrollWidth`.
**Then**: In all three states at both widths `scrollX` stays 0 and `scrollWidth` does not exceed the inner width; the text "Loading the shoes…" and "We could not load the shoes." wrap inside the window.
Traces: SC-5 (SRS §6 Responsive)
Check: interact_page

## 7. Entries and Back (CAT-003 items 7 and 8) — SC-6 … SC-11

**Scenario**: SC-6a — the "Kids" tile on the home page lands on the kids' list with "Kids" ticked
**Given**: The seed. Viewport desktop. `/` open.
**When**: Press the "Kids" category tile (the link `a[data-record="categories:kids"]`); read the address, `main h1`, the cards, the count and the tick boxes.
**Then**: The address is `/shop` with `category=kids` and no other parameter; the heading reads exactly `Kids`; the cards are exactly `/shop/samba-og-kids-wonder-mauve` and `/shop/samba-og-kids-cloud-white`; the count reads `2 matching`; the tick box "Kids" is checked and no other tick box is.
Traces: SC-6 (CAT-003 AC-17, item 8)
Check: interact_page

**Scenario**: SC-6b — phone: the same entry shows "Filter (1)" and "Kids" ticked in the opened rail
**Given**: The seed. Viewport phone. `/` open.
**When**: Press the "Kids" tile; read the Filter button; press it; read the tick boxes.
**Then**: The address carries `category=kids`; the button read before pressing is `Filter (1)`; after pressing, the rail is open and the tick box "Kids" is checked, no other is.
Traces: SC-6 (CAT-003 AC-17)
Check: interact_page

**Scenario**: SC-7 — the "Gifts" tile lands on the list under 3.000.000 ₫
**Given**: The seed. Viewport desktop. `/` open.
**When**: Press the "Gifts" tile (the link in `main` whose `href` is `/shop?price=under-3m`); read the address, `main h1`, the cards, their prices and the tick boxes.
**Then**: The address is `/shop?price=under-3m`; the heading reads exactly `All shoes`; the tick box "Under 3.000.000 ₫" is checked and no other tick box is; the cards are `/shop/samba-og-w-wonder-quartz`, `/shop/samba-og-kids-wonder-mauve`, `/shop/samba-og-kids-cloud-white`; every price shown is below 3.000.000 ₫ (2.790.000 ₫, 2.290.000 ₫, 2.290.000 ₫); the count reads `3 matching`.
Traces: SC-7 (CAT-003 AC-18, item 8)
Check: interact_page

**Scenario**: SC-8a — the footer's "Gifts" gives the same result from a page that is not the shop
**Given**: The seed. Viewport desktop. `/shipping` open.
**When**: Press "Gifts" in the footer (the link `a[href="/shop?price=under-3m"]` inside `footer`); read the address, the heading, the cards and the ticks.
**Then**: Exactly the result of SC-7: address `/shop?price=under-3m`, heading `All shoes`, "Under 3.000.000 ₫" checked and nothing else, the same three cards, `3 matching`.
Traces: SC-8 (CAT-003 AC-19)
Check: interact_page

**Scenario**: SC-8b — the footer's "Gifts" from a shoe's page and from a filtered shop list
**Given**: The seed. Viewport desktop.
**When**: Open `/shop/samba-og-w-core-black` and press "Gifts" in the footer; read the address and the count. Then open `/shop?category=men&sort=price`, press "Gifts" in the footer; read the address, the tick boxes and the count.
**Then**: Both times the address is exactly `/shop?price=under-3m` (the earlier `category` and `sort` are gone), "Under 3.000.000 ₫" is the only tick box checked (not "Men"), and the count reads `3 matching`.
Traces: SC-8 (CAT-003 AC-19: "from any page")
Check: interact_page

**Scenario**: SC-9a — the footer's Women, Men and Kids links open their list with their box ticked
**Given**: The seed. Viewport desktop. `/` open.
**When**: Press "Women" in the footer; read the address, heading, count and ticks; return to `/`; press "Men" and read; return to `/`; press "Kids" and read.
**Then**: Women: address `/shop?category=women`, heading `Women`, `4 matching`, only "Women" checked. Men: `/shop?category=men`, heading `Men`, `6 matching`, only "Men" checked. Kids: `/shop?category=kids`, heading `Kids`, `2 matching`, only "Kids" checked.
Traces: SC-9 (CAT-003 item 8)
Check: interact_page

**Scenario**: SC-9b — the footer's "All shoes" opens the whole shop with nothing ticked
**Given**: The seed. Viewport desktop. `/shop?category=women&q=og` open (the box shows `og`).
**When**: Press "All shoes" in the footer; read the address, the heading, the search box, the tick boxes and the count.
**Then**: The address is `/shop` with no query; the heading reads `All shoes`; the search input is empty; no tick box is checked and no pill is pressed; the count reads `12 matching`.
Traces: SC-9 (CAT-003 item 8)
Check: interact_page

**Scenario**: SC-9c — a line row on the home page opens that line with its box ticked
**Given**: The seed. Viewport desktop. `/` open.
**When**: Press the line row "Gazelle" (the link `a[href="/shop?line=gazelle"]`); read the address, heading, cards, count and ticks; return to `/`; press the row "Samba" and read.
**Then**: Gazelle: address `/shop?line=gazelle`, heading `Gazelle`, one card `/shop/gazelle-indoor-solar-gold`, `1 matching`, only the tick box "Gazelle" checked under Line. Samba: `/shop?line=samba`, heading `Samba`, `5 matching`, only "Samba" checked.
Traces: SC-9 (CAT-003 item 8)
Check: interact_page

**Scenario**: SC-10a — Back from a shoe returns to the same single-card list with every setting
**Given**: The seed. Viewport desktop. Open `/shop?category=women&line=samba&size=38&price=under-3m&q=og&sort=price`: one card `/shop/samba-og-w-wonder-quartz`, `1 matching`, heading `Results for “og”`.
**When**: Press that card; wait for `/shop/samba-og-w-wonder-quartz`; press the browser's Back; read the address, the ticks, the pill, the search box, the sort control, the cards and the count.
**Then**: The address holds the same six parameters with the same values; "Women", "Samba" and "Under 3.000.000 ₫" are checked and nothing else is; the pill `38` is `aria-pressed="true"` and no other pill is; the search input shows `og`; the sort control reads `Price, low to high`; the one card is `/shop/samba-og-w-wonder-quartz`; the count reads `1 matching`.
Traces: SC-10 (CAT-003 AC-9, item 7)
Check: interact_page

**Scenario**: SC-10b — Back returns the same cards in the same order when there are several
**Given**: The seed. Viewport desktop. Open `/shop?category=women&q=samba&sort=-price`; read the hrefs of the cards in order (the women's Samba shoes) and the count.
**When**: Press the second card; wait for its page; press Back; read the cards, the count, the address, the sort control and the search box.
**Then**: The same hrefs in the same order; the same count; the address has the same three parameters; the sort control reads `Price, high to low`; the search input shows `samba`; "Women" is checked.
Traces: SC-10 (CAT-003 AC-9)
Check: interact_page

**Scenario**: SC-11a — Back after "Show more" shows the first twelve only
**Given**: The seed plus FX-LIVE30 (30 live shoes). Viewport desktop. `/shop` open with 12 cards.
**When**: Press "Show more" (24 cards); record the hrefs of cards 13 to 24; press the first card; wait for its page; press Back; read the cards, the count and the button.
**Then**: Exactly 12 cards (12 distinct hrefs); the count reads `30 matching`; a button named "Show more" is present; none of the recorded hrefs of cards 13 to 24 is among the cards.
Traces: SC-11 (CAT-003 item 7)
Check: interact_page

**Scenario**: SC-11b — Back after "Show more" keeps the filters and the order
**Given**: The seed plus FX-LIVE30 (women 22). Viewport desktop. Open `/shop?category=women&sort=price`: 12 cards, `22 matching`.
**When**: Press "Show more" (22 cards); press the third card; press Back; read the address, the ticks, the sort control, the card prices, the count and the button.
**Then**: The address carries `category=women` and `sort=price`; "Women" is checked; the sort control reads `Price, low to high`; 12 cards whose prices never decrease along the list; the count reads `22 matching`; "Show more" is present.
Traces: SC-11 (CAT-003 item 7: "with any filters and order still set")
Check: interact_page

## 8. Groups with one option (CAT-003 AC-21) — SC-12, SC-13

**Scenario**: SC-12a — one colour and one size: neither group is drawn; the other groups still are
**Given**: The seed plus FX-ONECOL (every live shoe is black and lists only size `40`). Viewport desktop.
**When**: Open `/shop`; read the rail.
**Then**: No heading `Size (EU)` and no size pill (no element `[data-testid^="filter-size-"]`); no heading `Colour` and no colour tick box (neither "Black" nor any other); the headings `For`, `Line` and `Price` are shown with their boxes (three categories, five lines, three price bands). 12 cards and `12 matching` are shown.
Traces: SC-12 (CAT-003 AC-21)
Check: render_url

**Scenario**: SC-12b — two colours and two sizes: both groups are drawn (the boundary)
**Given**: The seed plus FX-TWOCOL (colours black and white; sizes `40` and `42`). Viewport desktop.
**When**: Open `/shop`; read the Colour and Size (EU) groups.
**Then**: The heading `Colour` with exactly two tick boxes, "Black" and "White"; the heading `Size (EU)` with exactly two pills, `40` and `42`, in that order.
Traces: SC-12 (CAT-003 AC-21)
Check: render_url

**Scenario**: SC-12c — while the options request is held, neither group is drawn; both appear when it answers
**Given**: The seed. Viewport desktop. Arrangement HOLD(3) on `/api/v1/product-filters` (the products request is not held).
**When**: Open `/shop`; read the rail at about 1 second and at about 2.5 seconds; read it again after 4 seconds.
**Then**: During the wait: no heading `Size (EU)`, no pills, no heading `Colour` and no colour boxes (no heading over an empty list); the headings `For`, `Line` and `Price` are shown and the 12 cards are shown. After 4 seconds: `Size (EU)` with 16 pills and `Colour` with nine options appear.
Traces: SC-12 (CAT-003 AC-21)
Check: interact_page

**Scenario**: SC-13a — a colour ticked on a group that is not drawn still applies, counts and can be cleared
**Given**: The seed plus FX-ONECOL. Viewport desktop.
**When**: Open `/shop?colour=black`; read the cards, the count and the rail; press "Clear all" in the rail; read the address and the count. Then at viewport phone open `/shop?colour=black` and read the Filter button.
**Then**: Before clearing: 12 cards, `12 matching`; no `Colour` group is drawn; "Clear all" is shown in the rail. After pressing it: the address has no `colour` and no other parameter; 12 cards, `12 matching`. At phone width the button reads `Filter (1)`.
Traces: SC-13 (CAT-003 AC-21, boundary)
Check: interact_page

**Scenario**: SC-13b — a colour filter on a group that is not drawn really filters
**Given**: The seed plus FX-ONECOL (no live shoe is white). Viewport desktop.
**When**: Open `/shop?colour=white`; read the page; press "Clear all" in the empty state; read the address and the count. Then at viewport phone open `/shop?colour=white` and read the Filter button.
**Then**: Before: no card, the count reads `0 matching`, the heading "No pair matches that." is shown and "Clear all" is shown in the rail. After pressing the empty state's "Clear all": the address has no `colour`, 12 cards, `12 matching`. At phone width the button reads `Filter (1)`.
Traces: SC-13 (CAT-003 AC-21, boundary)
Check: interact_page

## 9. Clear all (CAT-004 item 1) — SC-14 … SC-19

**Scenario**: SC-14a — "Clear all" removes every filter, the search words and the size pill
**Given**: The seed plus FX-LIVE30 (30 live shoes). Viewport desktop. Open `/shop?category=women&line=samba&size=38&price=under-3m&q=og`: one card (`/shop/samba-og-w-wonder-quartz`), `1 matching`, heading `Results for “og”`, the pill `38` pressed.
**When**: Press "Clear all" in the rail; read the address, the ticks, the pills, the search box, the heading, the cards and the count.
**Then**: The address carries none of `category`, `line`, `colour`, `size`, `price`, `q`; no tick box is checked; every pill has `aria-pressed="false"`; the search input is empty; the heading reads `All shoes`; 12 cards with 12 distinct hrefs; the count reads `30 matching`, the same as a fresh `/shop`; a "Show more" button is present; no "Clear all" is left in the rail.
Traces: SC-14 (CAT-004 AC-1)
Check: interact_page

**Scenario**: SC-14b — phone: the colour is removed too, and the button goes back to "Filter"
**Given**: The seed. Viewport phone. Open `/shop?category=women&line=samba&colour=black&size=38&price=3m-4m&q=samba`: one card (`/shop/samba-og-w-core-black`); the button reads `Filter (5)`.
**When**: Press "Filter"; press "Clear all" in the rail; read the address, the heading, the button and the cards.
**Then**: The address carries none of `category`, `line`, `colour`, `size`, `price`, `q`; the heading reads `All shoes`; the button reads exactly `Filter`; 12 cards and `12 matching`.
Traces: SC-14 (CAT-004 AC-1; story SC-14 "at 390px the button reads Filter")
Check: interact_page

**Scenario**: SC-15 — with only a size pressed, "Clear all" is shown and removes the size
**Given**: The seed. Viewport desktop. Open `/shop?size=38`.
**When**: Read the rail, the pill and the cards; press "Clear all" in the rail; read the address, the pill, the cards and the count.
**Then**: Before: a button "Clear all" is in the rail; the pill `38` is `aria-pressed="true"`; 5 cards and `5 matching` (not read as a page size of 38). After: the address has no `size`; the pill `38` is `aria-pressed="false"`; 12 cards and `12 matching`; "Clear all" is no longer in the rail.
Traces: SC-15 (CAT-004 AC-2)
Check: interact_page

**Scenario**: SC-16 — with only search words set, "Clear all" is shown and removes them
**Given**: The seed. Viewport desktop. Open `/shop?q=samba`: heading `Results for “samba”`, the search box shows `samba`.
**When**: Read the rail; press "Clear all" in the rail; read the address, the search box and the heading.
**Then**: Before: "Clear all" is in the rail. After: the address has no `q`; the search input is empty (its value is ""); the heading reads `All shoes`; 12 cards.
Traces: SC-16 (CAT-004 AC-2, item 1)
Check: interact_page

**Scenario**: SC-17a — nothing set, or only a sort: no "Clear all"
**Given**: The seed. Viewport desktop.
**When**: Open `/shop`; look for a control named "Clear all" anywhere on the page. Open `/shop?sort=price` and look again.
**Then**: Both times no element named "Clear all" exists on the page (not in the rail and not elsewhere), and 12 cards are shown.
Traces: SC-17 (CAT-004 AC-3; story §7 "Clear all with only a sort set is not shown")
Check: interact_page

**Scenario**: SC-17b — ticking any box makes "Clear all" appear, and unticking it removes it
**Given**: The seed. Viewport desktop. `/shop` open.
**When**: Tick "Women"; look for "Clear all" in the rail; untick "Women"; look again.
**Then**: After ticking: a button "Clear all" is in the rail. After unticking: no element named "Clear all" exists on the page.
Traces: SC-17 (CAT-004 AC-3)
Check: interact_page

**Scenario**: SC-18 — "Clear all" keeps the order
**Given**: The seed. Viewport desktop. Open `/shop?category=women&line=samba&size=38&sort=price`: 3 cards.
**When**: Press "Clear all" in the rail; read the address, the sort control and the card prices.
**Then**: The address is `/shop?sort=price` (only `sort`); the sort control reads `Price, low to high`; 12 cards whose prices never decrease along the list (the first two are the 2.290.000 ₫ Samba Kids pairs); `12 matching`.
Traces: SC-18 (CAT-004 AC-4)
Check: interact_page

**Scenario**: SC-19a — the hook is unchanged for every caller that does not opt in
**Given**: The PR's unit tests for `useTableQuery` (`src/lib/table.ts`).
**When**: Read the PR's test file for the hook and run the frontend tests (`run_tests`, target `frontend`).
**Then**: A test calls the hook without the shop's option and shows `clear()` keeping `size` (and `sort`) while removing every other parameter, and `filtering` staying `false` when only `size` (and `page`, `sort`) is set; a second test calls it with the shop's option and shows `clear()` removing `size` and `filtering` being `true` for `size=38`. The frontend suite passes.
Traces: SC-19 (CAT-004 item 1, TL constraint; story §8.1)
Check: manual — the assertion is a unit test of a hook in code; none of the five page instruments reads a test runner. Reported UNVERIFIED if the PR carries no such test, and the dealer's list in SC-19b is what shows the same behaviour in a browser.

**Scenario**: SC-19b — the dealer's lists still read `size` as the page size and "Clear filters" keeps it
**Given**: The seed plus FX-LIVE60 (60 live shoes). Signed in as the dealer. Viewport desktop.
**When**: Open `/admin/products?size=50&status=live`; count the table's rows; press "Clear filters"; read the address and count the rows. Then open `/admin/products?size=50` and look for "Clear filters". Then open `/admin/products?status=live` and count the rows.
**Then**: First open: 50 rows. After "Clear filters": the address still carries `size=50` and has no `status`; 50 rows. `/admin/products?size=50` with nothing else: 50 rows and **no** "Clear filters" button (size alone is not a filter there). `/admin/products?status=live` (the default page of 20): 20 rows.
Traces: SC-19 (CAT-004 item 1, TL constraint)
Check: interact_page

**Scenario**: SC-19c — on `/shop`, `size=38` is never a page size
**Given**: The seed plus FX-LIVE30 (23 shoes have size 38 available). Viewport desktop.
**When**: Open `/shop?size=38`; read the cards and the count; list the requests to `/api/v1/products`.
**Then**: 12 cards (not 23), the count reads `23 matching`, a "Show more" button is present, and every request to `/api/v1/products` carries `limit=12` (none carries `limit=38`).
Traces: SC-19 (CAT-004 item 1; story SC-19 last sentence)
Check: render_url

## 10. Empty state (CAT-004 item 2) — SC-20, SC-21

**Scenario**: SC-20a — a search that matches nothing shows the empty state word for word
**Given**: The seed. Viewport desktop.
**When**: Open `/shop?q=zzzz`; read the results area; inspect the button "Clear all" in the results area.
**Then**: A heading (`h2`) reads exactly `No pair matches that.`; a paragraph reads exactly `Try another size or fewer filters — or ask us, and we will tell you when it comes in.` (with the em dash); a button named "Clear all" in the results area contains an arrow icon (an `svg`) after its text; the count reads `0 matching`; no card; no button named "Show more"; the text "Loading the shoes…" and "We could not load the shoes." are not on the page.
Traces: SC-20 (CAT-004 AC-5)
Check: interact_page

**Scenario**: SC-20b — a line that does not exist gives the same empty state, not an error
**Given**: The seed. Viewport desktop.
**When**: Open `/shop?line=zzz`; read the page.
**Then**: The same heading, line, arrow button and `0 matching` as SC-20a; the main heading reads `All shoes`; no card; no "Show more"; no loading text and no error message.
Traces: SC-20 (CAT-004 Not found; CAT-002 Not found)
Check: render_url

**Scenario**: SC-21a — the empty state's "Clear all" removes the search words and keeps the order
**Given**: The seed. Viewport desktop. Open `/shop?q=zzzz&sort=price`: the empty state is shown.
**When**: Press "Clear all" in the empty state; read the address, the search box, the sort control, the cards and the count.
**Then**: The address is `/shop?sort=price` (no `q`); the search input is empty; the sort control reads `Price, low to high`; 12 cards in rising price order; the count reads `12 matching`, the count of the whole shop; the empty state is gone.
Traces: SC-21 (CAT-004 AC-6)
Check: interact_page

**Scenario**: SC-21b — the empty state's "Clear all" removes a line that matches nothing
**Given**: The seed. Viewport desktop. Open `/shop?line=zzz&sort=-rating`: the empty state is shown.
**When**: Press "Clear all" in the empty state; read the address, the sort control and the count.
**Then**: The address is `/shop?sort=-rating` (no `line`); the sort control reads `Best rated`; 12 cards; `12 matching`.
Traces: SC-21 (CAT-004 AC-6)
Check: interact_page

**Scenario**: SC-21c — the empty state's "Clear all" removes a size that matches nothing
**Given**: The seed. Viewport desktop. Open `/shop?size=99` (no shoe has size 99; the API answers 200 with `total` 0): the empty state is shown.
**When**: Press "Clear all" in the empty state; read the address, the pills and the count.
**Then**: The address has no `size`; no pill is pressed; 12 cards; `12 matching`.
Traces: SC-21 (CAT-004 AC-6, item 1: the size pills are removed too)
Check: interact_page

## 11. Loading (CAT-004 item 3) — SC-22 … SC-24

**Scenario**: SC-22 — the first page is slow: "Loading the shoes…" for the whole wait, never the empty state
**Given**: The seed. Viewport desktop. Arrangement HOLD(3) on the first request to `/api/v1/products`.
**When**: Open `/shop`; read the page at about 0.3 s, 1.5 s and 2.7 s after navigation, then again once the answer has been drawn (after about 4 s).
**Then**: At each of the first three readings: the element with `role="status"` reads `Loading the shoes…`; the text "No pair matches that." is not on the page; no card is shown; the count element `[data-testid="shop-count"]` is not shown and the text `0 matching` is not on the page. After the answer: the status element is gone; 12 cards and `12 matching` are shown.
Traces: SC-22 (CAT-004 AC-7, item 3)
Check: interact_page

**Scenario**: SC-23 — a new first page after ticking a box: the loading line, no empty state, no cards of the old list
**Given**: The seed. Viewport desktop. `/shop` open with 12 cards. Arrangement HOLD(3) on the next request to `/api/v1/products`.
**When**: Tick "Women"; read the page at about 0.3 s and 2.5 s; read it again after about 4 s.
**Then**: During the wait: the status element reads `Loading the shoes…`; "No pair matches that." is not on the page; no card is on screen (none of the previous 12 hrefs remains); no `0 matching`. After the answer: 4 cards and `4 matching`; the status element is gone.
Traces: SC-23 (CAT-004 AC-7, item 3)
Check: interact_page

**Scenario**: SC-24 — "Show more" while its page is on its way: disabled, "Loading…", cards stay, no second request
**Given**: The seed plus FX-LIVE30. Viewport desktop. `/shop` open with 12 cards and `30 matching`. Arrangement HOLD(3) on the request that carries `page=2`.
**When**: Press "Show more"; read the button and the cards at about 0.3 s; press the button a second time; read the requests; read the page again after about 4 s.
**Then**: At 0.3 s: the button reads exactly `Loading…`, has the `disabled` attribute; the first 12 cards (same hrefs) are still shown; the count still reads `30 matching`. The second press sends no second request: exactly one request carrying `page=2` was made. After the answer: 24 cards, 24 distinct hrefs; `30 matching`; the button reads `Show more` again and is not disabled.
Traces: SC-24 (CAT-004 AC-8, item 3)
Check: interact_page

## 12. Errors (CAT-004 item 4) — SC-25 … SC-30

The data hook retries a failed read once and the page shows the error after that retry, so every "wait for the message" below waits up to 20 seconds and reads as soon as it is there.

**Scenario**: SC-25a — the shoe request answers 500: the message and "Try again", not the empty state
**Given**: The seed. Viewport desktop. Arrangement FAIL500 on every request to `/api/v1/products` (the marker body of §2).
**When**: Open `/shop`; wait for the message; read the page text and the results area.
**Then**: The element with `role="alert"` reads exactly `We could not load the shoes.`; a button named exactly `Try again` is beside it. The page does not contain "No pair matches that.", "Loading the shoes…", `0 matching` or any `matching` count (`[data-testid="shop-count"]` is not shown); no card is shown. The page text contains none of `500`, `internal`, `tc-secret-db-detail`, `Error` or `ApiError`. The header, the search box, the sort control and the footer are still drawn (the page is not blank).
Traces: SC-25 (CAT-004 AC-9, Upstream failure)
Check: interact_page

**Scenario**: SC-25b — the database is unreachable: the same message and button
**Given**: The seed. Viewport desktop. Arrangement DB-DOWN: the backend runs and its database refuses connections, so the API answers 500 `internal`.
**When**: Open `/shop`; wait for the message; read the page text. Put the database back afterwards.
**Then**: The alert reads exactly `We could not load the shoes.` with a button `Try again`; "No pair matches that." is not on the page; no card; the page text contains none of `500`, `internal`, `database`; the heading `main h1` still reads `All shoes`.
Traces: SC-25 (CAT-004 AC-9, Upstream failure)
Check: interact_page

**Scenario**: SC-26 — "Try again" asks for the same first page and the list loads
**Given**: The seed. Viewport desktop. Open `/shop?category=women&sort=price` with arrangement FAIL500 on `/api/v1/products`; the message and "Try again" are shown. The interception is then lifted for the case, and the next request is HOLD(2) so the loading line can be read.
**When**: Press "Try again"; read the page at about 0.5 s and after about 3 s; read the request the press sent.
**Then**: At 0.5 s the status element reads `Loading the shoes…` and the alert and "Try again" are gone. After the answer: 4 cards (the women's shoes) in rising price order, `4 matching`; no alert, no "Try again". The request sent by the press is to `/api/v1/products` and carries `category=women`, `sort=price`, `limit=12` and `status=live`, and either no `page` or `page=1` (never `page=2`); the real API's answer to it is 200.
Traces: SC-26 (CAT-004 AC-10)
Check: interact_page

**Scenario**: SC-27a — "Show more" fails: the shoes stay, the message appears under them, "Try again" asks for page 2 and succeeds
**Given**: The seed plus FX-LIVE30. Viewport desktop. `/shop` open with 12 cards and `30 matching`. Arrangement FAIL500-PAGE2.
**When**: Press "Show more"; wait for the message; read the page. Lift the interception; press "Try again"; read the request it sent and the page; press "Show more" once more; read the page.
**Then**: After the failure: the same 12 cards (same hrefs, same order) and `30 matching` are still shown; the alert reads `We could not load the shoes.` and its top is at or below the bottom of the last card; a button `Try again` is shown **in place of** "Show more" (no button named "Show more" exists and there is no second "Try again"); "No pair matches that." is not shown. After "Try again": the request carries `page=2`, `limit=12`, `status=live` and no other filter than the page had; 24 cards with 24 distinct hrefs, the first 12 unchanged and the next 12 the API's second page (none twice, none missing); `30 matching`; the alert and "Try again" are gone; "Show more" is back. After the next "Show more": 30 cards, no "Show more".
Traces: SC-27 (CAT-004 AC-11, item 4)
Check: interact_page

**Scenario**: SC-27b — "Try again" fails again: the cards and the message stay
**Given**: The seed plus FX-LIVE30. Viewport desktop. `/shop` open. Arrangement FAIL500-PAGE2, kept in force; "Show more" pressed and the message shown.
**When**: Press "Try again"; wait for the answer; read the page and the requests.
**Then**: The request sent carries `page=2` again (not `page=3`); the 12 cards (same hrefs) and `30 matching` stay; the alert `We could not load the shoes.` and the button `Try again` are shown again; no card is added; no "Show more".
Traces: SC-27 (CAT-004 item 4: "when it fails again")
Check: interact_page

**Scenario**: SC-28a — the browser is offline: the first-page error
**Given**: The seed. Viewport desktop. Arrangement OFFLINE: `/` is loaded, the network is switched off, then the visitor presses the header's "Women" link (a list never requested before, `/shop?category=women`).
**When**: Wait for the message; read the page.
**Then**: The same as SC-25a: the alert reads `We could not load the shoes.`, a button `Try again` is shown, no "No pair matches that.", no "Loading the shoes…" left, no card, no `matching` count, no technical text (`Failed to fetch`, `NetworkError`, `TypeError` are not on the page).
Traces: SC-28 (CAT-004 AC-12)
Check: interact_page

**Scenario**: SC-28b — offline when "Show more" is pressed: the shoes stay and the message appears under them
**Given**: The seed plus FX-LIVE30. Viewport desktop. `/shop` open with 12 cards. The network is then switched off (OFFLINE).
**When**: Press "Show more"; wait for the message; read the page; switch the network on; press "Try again"; read the page.
**Then**: After the failure: the same as SC-27a — 12 cards and `30 matching` stay, the alert and `Try again` are under them, no "Show more". After "Try again" with the network back: 24 cards with 24 distinct hrefs and no alert.
Traces: SC-28 (CAT-004 AC-12; story SC-28 second sentence)
Check: interact_page

**Scenario**: SC-29 — ticking a box on the error screen asks for the first page of the new list
**Given**: The seed. Viewport desktop. Arrangement FAIL500 on `/api/v1/products` only (categories and lines answer); `/shop` shows the message and "Try again". The interception is then lifted and the next products request is HOLD(2).
**When**: Tick "Women" in the rail; read the page at about 0.5 s and after about 3 s.
**Then**: At 0.5 s the alert and "Try again" are gone and the status element reads `Loading the shoes…`. After the answer: 4 cards and `4 matching`; no alert, no status element.
Traces: SC-29 (CAT-004 item 4, boundary)
Check: interact_page

**Scenario**: SC-30 — an older request that fails after a newer one succeeded changes nothing
**Given**: The seed. Viewport desktop. `/shop` open with 12 cards. Arrangement: the request carrying `category=women` (and not `men`) is held 3 s and then answered 500 (for its automatic retry too); the request carrying `category=women,men` is answered normally.
**When**: Tick "Women", then within one second tick "Men"; read the page at about 1.5 s; wait 10 seconds so the held request has failed and been retried; read the page again.
**Then**: At 1.5 s: the address carries `category=women,men`, 10 cards, `10 matching`. After the wait: the same 10 cards in the same order, `10 matching`; no element with `role="alert"`, no text "We could not load the shoes.", no "Try again".
Traces: SC-30 (CAT-002 Conflict, CAT-004 item 4)
Check: interact_page

## 13. Words, access and look (CAT-004 item 5, SRS §6) — SC-31 … SC-33

**Scenario**: SC-31a — the four strings read exactly and come from the site's content keys
**Given**: The seed plus FX-LIVE30. Viewport desktop. The states of SC-22 (HOLD), SC-25a (FAIL500) and SC-24 (HOLD on page 2) are produced one after the other.
**When**: In each state read the text and the `data-edit` attribute of the element that holds it (the element itself or a descendant).
**Then**: Loading: the text is exactly `Loading the shoes…` and the element carries `data-edit="pages.shop.loading"`. Error: exactly `We could not load the shoes.` with `data-edit="pages.shop.error"`. Retry button: exactly `Try again` with `data-edit="pages.shop.retry"`. Show-more loading: the button reads exactly `Loading…` with `data-edit="pages.shop.loading_more"`. In each text the ellipsis is the single character `…` (U+2026), not three full stops. Existing words are unchanged and still drawn from their own keys under the shop page's content, each element carrying `data-edit` with the key it has today: `Clear all`, `Show more`, `Filter`, `No pair matches that.` and `Sort`. The language is English throughout.
Traces: SC-31 (CAT-004 item 5, open question 1)
Check: interact_page

**Scenario**: SC-31b — only new keys were added to `content.json`, and no copy is typed into a component
**Given**: The PR's diff of `code/frontend/src/content.json` and of the shop's components, and the PR's CI result.
**When**: Read the diff under `pages.shop`; read the CI lint job.
**Then**: The diff of `content.json` adds exactly the four keys `pages.shop.loading`, `pages.shop.error`, `pages.shop.retry`, `pages.shop.loading_more` with the values of SC-31a and changes or removes no other key; no component contains the four strings as literals (the lint rule that refuses a word written into the page passes).
Traces: SC-31 (CAT-004 item 5: "new keys only, no existing key changed or removed")
Check: manual — a diff and a CI job are read, not a page; no instrument of the five reads source or CI. SC-31a already proves the page draws each string from its key.

**Scenario**: SC-32a — the loading line is a status and the error an alert, present in the page
**Given**: The seed. Viewport desktop. Arrangements HOLD(3) (first request) for the loading state and FAIL500 for the error state.
**When**: In each state read the accessibility role and visibility of the element that holds the message.
**Then**: Loading: the element holding `Loading the shoes…` has `role="status"`, is in the page (not `display: none`, not `aria-hidden`). Error: the element holding `We could not load the shoes.` has `role="alert"`, is in the page and not hidden. After a failed "Show more" (FAIL500-PAGE2, FX-LIVE30) the message under the cards is also `role="alert"`.
Traces: SC-32 (CAT-004 item 5, SRS §6 Accessibility)
Check: interact_page

**Scenario**: SC-32b — the disabled "Show more" and the two buttons are real buttons
**Given**: The seed plus FX-LIVE30. Viewport desktop. Arrangement HOLD(3) on the `page=2` request; for the other two, `/shop?q=zzzz` (the empty state) and FAIL500.
**When**: Press "Show more" and inspect the button during the wait; inspect "Clear all" in the empty state; inspect "Try again" in the error state.
**Then**: The loading "Show more" is a `button` element that carries the `disabled` attribute and whose visible text is `Loading…`. "Clear all" in the empty state is a `button` element with visible text `Clear all`. "Try again" is a `button` element with visible text `Try again`. Each is reachable with Tab and activates with Enter or Space.
Traces: SC-32 (CAT-004 item 5, SRS §6 Accessibility)
Check: interact_page

**Scenario**: SC-32c — the controls keep their names and states
**Given**: The seed. Viewport phone for the Filter button, desktop for the rest.
**When**: Read the Filter button's name and `aria-expanded` closed and open; read `aria-pressed` on the pill `38` before and after pressing it; read the search input's accessible name; read the sort control's accessible name.
**Then**: The Filter button is named `Filter` and carries `aria-expanded` (`false`, then `true` after pressing). The pill `38` carries `aria-pressed="false"`, then `"true"`. The search input's accessible name is `Search by name, line, colour or code`. The sort control is a combobox named `Sort`.
Traces: SC-32 (CAT-004 item 5, SRS §6 Accessibility)
Check: interact_page

**Scenario**: SC-33a — "Try again" looks like the empty state's "Clear all"
**Given**: The seed. Viewport desktop. Two pages are opened in turn: `/shop?q=zzzz` (the empty state) and `/shop` with FAIL500 (the error).
**When**: Measure the button named "Clear all" in the results area of the first page, and the button named "Try again" on the second page.
**Then**: Their computed `font-size`, `letter-spacing`, `text-transform`, `border-top-width`, `border-top-style`, `border-top-color` and `border-top-left-radius` are equal, pair by pair.
Traces: SC-33 (CAT-004 item 5)
Check: measure_styles

**Scenario**: SC-33b — loading and error messages use the empty state's own type, colour and block
**Given**: The seed. Viewport desktop. Three pages are opened in turn: `/shop?q=zzzz` (the empty state), `/shop` with HOLD(3) on the first request (loading), `/shop` with FAIL500 (error).
**When**: Measure the empty state's heading (`main h2`) and its line (the paragraph under it) and the block that wraps them; measure the loading line and the error message and the block that wraps each.
**Then**: The error message's computed `font-size` equals the empty heading's; the loading line's `font-size` and `color` equal the empty line's (14px, the site's `ink-soft` colour). Each of the loading and error blocks has the same computed `border-bottom-width` (1px), `border-bottom-color`, `padding-top` and `padding-bottom` as the empty state's block, and its text is left-aligned (`align-items: flex-start`, `text-align: start`) like the empty state's. The error colour, if it differs from the heading's, is the site's `danger` colour and not a raw value.
Traces: SC-33 (CAT-004 item 5, story §6 "How the undrawn states look")
Check: measure_styles

**Scenario**: SC-33c — only the design's tokens are used
**Given**: The PR's diff and its CI result.
**When**: Run or read the lint job (`npm run lint`) and the design-token job of CI; read the diff of the shop's components for a raw colour (`#…`, `rgb(…)`, `hsl(…)`), an arbitrary-value class (`bg-[…]`, `text-[17px]`), a palette colour from outside the theme (`bg-blue-500`) and a `.module.css` file.
**Then**: Both CI jobs pass; the diff contains none of those; `src/theme.css` and `src/index.css` are unchanged.
Traces: SC-33 (CAT-004 item 5: "tokens only")
Check: manual — a CI result and a diff are read, not a page; the measured colours and sizes of the same states are SC-33a and SC-33b.

## 14. Coverage of the story's acceptance criteria

| Criterion | Cases |
|---|---|
| SC-1 | SC-1a, SC-1b |
| SC-2 | SC-2a, SC-2b, SC-2c |
| SC-3 | SC-3a, SC-3b |
| SC-4 | SC-4a, SC-4b |
| SC-5 | SC-5a, SC-5b, SC-5c, SC-5d |
| SC-6 | SC-6a, SC-6b |
| SC-7 | SC-7 |
| SC-8 | SC-8a, SC-8b |
| SC-9 | SC-9a, SC-9b, SC-9c |
| SC-10 | SC-10a, SC-10b |
| SC-11 | SC-11a, SC-11b |
| SC-12 | SC-12a, SC-12b, SC-12c |
| SC-13 | SC-13a, SC-13b |
| SC-14 | SC-14a, SC-14b |
| SC-15 | SC-15 |
| SC-16 | SC-16 |
| SC-17 | SC-17a, SC-17b |
| SC-18 | SC-18 |
| SC-19 | SC-19a, SC-19b, SC-19c |
| SC-20 | SC-20a, SC-20b |
| SC-21 | SC-21a, SC-21b, SC-21c |
| SC-22 | SC-22 |
| SC-23 | SC-23 |
| SC-24 | SC-24 |
| SC-25 | SC-25a, SC-25b |
| SC-26 | SC-26 |
| SC-27 | SC-27a, SC-27b |
| SC-28 | SC-28a, SC-28b |
| SC-29 | SC-29 |
| SC-30 | SC-30 |
| SC-31 | SC-31a, SC-31b |
| SC-32 | SC-32a, SC-32b, SC-32c |
| SC-33 | SC-33a, SC-33b, SC-33c |
| Contract only | None: the story adds no endpoint, and every request the cases read is plan item 4's (`GET /v1/products`, `GET /v1/product-filters`); SC-26 and SC-27a also read the real API's 200 answer to the request the retry sends |

## 15. Instruments and what could come back unverified

Of 70 cases, 3 are `manual` (SC-19a, SC-31b, SC-33c), each because the thing read is a unit test, a diff or a CI result, which none of the five page instruments reads; each has a page-level case beside it that proves the visible half of the same criterion (SC-19b, SC-31a, SC-33a and SC-33b). Every interaction (a press, a tick, typing, Back, Tab) is `interact_page`; every visual fact — `display` at 1023 and 1024, computed type, border, radius and colour — is `measure_styles`; every other case reads a rendered page (`render_url`).

The only reason a case other than those three could come back UNVERIFIED is the instrument itself, named in §4: if `interact_page` cannot hold, fail or cut a request, then SC-12c, SC-22, SC-23, SC-24, SC-26, SC-27a, SC-27b, SC-28b, SC-29 and SC-30 cannot be arranged (10 cases) and SC-5d is only partly arranged; SC-25a, SC-28a and SC-32a keep a first-page reading through DB-DOWN (SC-25b) and the stopped backend. The expectations are not weakened to fit what the instrument can do.
