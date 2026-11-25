# Test cases — Search, filters and sorting served by the API

Module: `catalogue` · Plan item 4 · Story: `docs/catalogue/stories/search-filters-and-sorting-served-by-the-api.md` (SC-1 … SC-40)
Requirements: CAT-001, CAT-002, CAT-003 items 1–5 (`docs/catalogue/SRS.md`) · Contract: `docs/architecture/services.md` §2.3, §2.4, §3.2, §3.4

Written before the implementation exists, from the requirements, the story and the contract only.

## 1. Risk and depth

| Part | Risk | Why | Depth |
|---|---|---|---|
| The query language on the list endpoints (SC-1 … SC-17) | **High** | Every shop, search and dealer list reads it. A wrong filter silently hides a pair a customer would buy or shows one that is sold out in the size asked for; `q` over orders is the one place a customer's phone could leak through a search | Every operator, both ends of every range, the equal-value paging rule, every 400 the story names, the private-field rule, the failure path |
| `GET /v1/product-filters` and speed (SC-18, SC-19) | Medium | Feeds the Colour and Size groups on every shop view; the speed target decides whether the no-index design holds | Shape, order, draft exclusion, 10,000-shoe timing |
| The shop page reading pages from the API (SC-20 … SC-32, SC-40) | Medium–High | The page every customer filters from; it must show the true count, never more than twelve at a time, never a draft | Counts, "Show more" boundaries, headings, cards, sort, option lists, empty list, stale answers, "not claimed empty while loading or failed" |
| What the visitor sets in the address bar (SC-33 … SC-39) | Medium | A list that cannot be shared or that mis-reads a tick is a broken filter | Any-of / all-of, each price band and their union, size pill, share link, search box, bad address values |

## 2. How to read this file

- **Starting state of every case**: the seed of the `data` story — 12 live shoes, 5 lines, 3 categories, 14 orders, 36 reviews — and no row whose id starts `tc-`. A case that needs more says so in its `Given` by naming a fixture from §3. After a case, every fixture is deleted and every seed row it changed is put back, so no case depends on another.
- **Paths** are written as the backend mounts them (`/v1/products`); from outside they are `/api/v1/products`.
- **Fixtures are inserted with `set_db_state`** (SQL against the test database, never the seed file). Columns not listed take their defaults. A fixture's size lists and counts are written in whichever form `main` holds when the case runs: the `sizes` / `sizes_low` / `sizes_out` columns of `products`, or `product_sizes` rows (plan item 5 moves them). A size that is "listed" has a row or list entry; "sold out" means 0 pairs.
- **A 400** means: status 400, body `{"error":"bad_request","message":<non-empty sentence>,"errors":{<parameter>:<non-empty sentence>}}`, no `items` and no `total` key, `Content-Type: application/json; charset=utf-8`.
- **Expected values taken from the seed** are the values in `code/frontend/src/data/*.json` as the data story seeds them. Where a stored figure could legitimately be recalculated later (a shoe's `rating`), the case states the invariant and the seed value is only the example.
- **Differential reads**: "the unfiltered list" means `GET /v1/products?status=live&limit=100`; a filtered answer is compared with that list filtered by hand, so a case does not depend on a figure nobody pinned.
- **Dealer-only reads** (`/v1/orders`) are made with a dealer session from `POST /v1/auth/sign-in` (staff or admin). The story says to test as whoever may read the orders list on `main` at that time; a visitor being refused the orders list belongs to the `admin` module (ADMIN-013, ADMIN-016) and has no case here.
- **Shop page cases** run in a 1440×790 window (the rail is always shown from 1024px; the phone button is plan item 15). Cards are the links inside `main` whose `href` starts `/shop/`, counted by distinct `href`; the heading is `main h1`; the count is the text `<N> matching` under it; the size pills are the buttons in the "Size (EU)" group and carry `aria-pressed`; a tick box is the checkbox named by its label.

## 3. Fixtures

| Id | What is inserted (all `status = live` unless stated; ids start `tc-`) | Used by |
|---|---|---|
| FX-CUALO | `tc-cua-lo`: name "Beach Pair Tc", line `adizero`, category `men`, price 2000000, description `Light canvas pair for the Cửa Lò beach trip.` | SC-2 |
| FX-DAI | `tc-dai`: name `Đại Lộ Runner Tc`, line `adizero`, category `men`, price 2000000 | SC-2 |
| FX-LINEJOIN | `tc-line-join`: name "Zorblax One", line `gazelle`, category `men`; model, colourway, code, description contain neither "gazelle" nor "women" | SC-3 |
| FX-CATJOIN | `tc-cat-join`: name "Quillon Two", line `adizero`, category `women`; no field contains "women" | SC-3 |
| FX-PCT | `tc-pct`: name "Recycled Tc", description `Made from 100% recycled leather.`; and `tc-pct-decoy`: name "Heavy Tc", description `Weighs 1000 grams.` | SC-4 |
| FX-US | `tc-us`: name `Tc_Underscore Runner`; and `tc-us-decoy`: name "Tc Plain Runner". Neither has an underscore in any other field; no seeded shoe has one | SC-4 |
| FX-SAMBACLASSIC | `tc-samba-classic`: name "Samba Classic Tc", model "Samba Classic", line `samba`, category `women`, colourway "White / Black", code "TC0001", description "Plain leather pair." — no field contains "og" | SC-1 |
| FX-3M | `tc-price-3m`: price exactly 3000000, line `adizero`, category `men` | SC-6 |
| FX-NORATING | `tc-no-rating`: name "Unrated Tc", `rating` 0 (the stored "none"), `review_count` 0 | SC-7 |
| FX-SIZE38-2-3 | `tc-size-38-2-3`: lists only EU size `38 2/3`, in stock | SC-9 |
| FX-DRAFT | `tc-draft-samba`: **status `draft`**, name "Samba OG Draft Tc", model "Samba OG", line `samba`, category `women`, price 2500000, lists `38` in stock, description "Samba OG draft shoe." | SC-11, SC-18, SC-23 |
| FX-Q30 | 25 live shoes `tc-q-01` … `tc-q-25`, name "Samba Tc 01" … "Samba Tc 25", line `samba`, category `women`, price 3500000, sizes `38` and `40` in stock. With the 5 seeded Samba OG pairs, `q=samba` has **30** matches | SC-12 |
| FX-LIVE30 | 18 live shoes `tc-live-01` … `tc-live-18`, name "Pair Tc 01" … "Pair Tc 18", line `samba`, category `women`, colour `black`, price 3500000, released 2026-06-01 … 2026-06-18, sizes `38` and `40` in stock (none sold out). With the seed: **30 live shoes**, women 22, women Samba 21, size 38 available 23, band 3.000.000–3.999.999 ₫ 23 | SC-20 … SC-22, SC-33, SC-34 |
| FX-LIVE13 | 1 live shoe `tc-live-01` as above. With the seed: **13 live shoes** | SC-21 |
| FX-10K | 10,000 live shoes `tc-perf-00001` … `tc-perf-10000` generated in SQL (`generate_series`): names built from "Samba", "OG", "Gazelle", "Runner" and a number; lines and categories spread over the seeded ones; ~1,000 match `q=samba og` | SC-19 |
| FX-ORDERS | Two orders: number `DOI-90001`, customer "Lan", address `12 Nam Ky Khoi Nghia`, note ""; number `DOI-90002`, customer "Bich", address `3 Example Street`, note `Ask for Nam at the gate`. Neither number nor customer contains "nam" | SC-13 |
| FX-LINENULL | A line `tc-line-null`, name "Tc Line", `since` null | SC-7 |

## 4. Open points found while writing (reported, not resolved here)

1. **A shoe with no rating, ascending.** CAT-001 item 5 and `services.md` §2.4 say a record with no value for the sorted field comes last either way, but `products.rating` is `NOT NULL DEFAULT 0` (`erd.md` §3.4), so "no rating" is stored as 0 and `sort=rating` would put it first. SC-7 therefore tests "no rating last" on `-rating` for shoes (where it holds) and tests the "last either way" rule on a column that really is nullable (`lines.since`). TL/PM should say what `sort=rating` ascending must do for an unrated shoe; no case pins it.
2. **Database unreachable: 500 or 503.** The SRS ("Upstream failure … 500 with a generic error body"), the story (SC-17) and `services.md` §2.3/§3.2 all say a list answers **500 `internal`** when the database cannot be reached; 503 `unavailable` is reserved for shutdown and `/healthz`. The cases follow the approved documents (500). If the backend is reviewed to answer 503 for a refused connection, the documents and these two cases change together.
3. **`q=samba%20og` or `q=samba+og`.** CAT-003 AC-10 writes the address as `q=samba%20og`; the shop's address helper writes spaces as `+`. Both decode to the same value. The case reads the decoded `q` and accepts either spelling of the space.
4. **The union of price bands** is TL's decision (story §10.1). No case names a request parameter for it; the shop cases observe the cards and the count, and the API cases use `price_from` / `price_to` only.
5. **The SRS example of CAT-001 AC-13** lists shoes priced 2790000, 3090000, 3290000 and 5940000. The seed has no 5940000 shoe and five shoes between 3000000 and 3999999, so SC-6 states the seeded answer (five shoes).
6. **Plan item 5 changes how sizes are stored** while this story queries them; the size cases are written against "listed and not sold out", which holds in both forms.

## 5. The list endpoints answer the query (backend) — SC-1 … SC-17

**Scenario**: SC-1a — `q` finds both words in either order and `total` counts every page
**Given**: The seed. Shoes `samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `samba-og-kids-wonder-mauve`, `samba-og-kids-cloud-white` are the five Samba OG pairs.
**When**: `GET /v1/products?q=samba%20og&limit=100`, then `GET /v1/products?q=og%20samba&limit=100`, then `GET /v1/products?q=samba%20og&limit=2&page=1`, `page=2`, `page=3`.
**Then**: All answer 200 with `Content-Type: application/json; charset=utf-8` and `{items,total,page,limit}`. The first two return the same five ids in the same order and `total` 5. Every item's `name`, `model` or `description` contains "samba" and "og" (any case). The three pages of 2 hold 2, 2 and 1 distinct items whose union is the same five ids, and every page reports `total` 5 (not the page's length).
Traces: SC-1 (CAT-001 AC-1, AC-2)
Check: fetch_url

**Scenario**: SC-1b — every typed word must be found; a part of a word counts
**Given**: The seed plus FX-SAMBACLASSIC (a live Samba shoe with no "og" in any field).
**When**: `GET /v1/products?q=samba%20og&limit=100`, then `GET /v1/products?q=samba&limit=100`, then `GET /v1/products?q=samb%20o&limit=100`.
**Then**: `q=samba og` does not contain `tc-samba-classic` and returns the same five seeded ids (total 5). `q=samba` contains `tc-samba-classic` and the five seeded ids (total 6). `q=samb o` ("as part of a value") returns every shoe that contains "samb" and "o" somewhere, which includes the five Samba OG pairs and `tc-samba-classic`.
Traces: SC-1 (CAT-001 item 1; CAT-001 AC-1)
Check: fetch_url

**Scenario**: SC-2a — "Cửa Lò" is found without accents and in capitals
**Given**: The seed plus FX-CUALO.
**When**: `GET /v1/products?q=cua%20lo`, then `GET /v1/products?q=CUA%20L%C3%92` (that is `CUA LÒ`).
**Then**: Both answer 200 and `items` contains `tc-cua-lo`; both return the same set of ids. Every item in either answer contains "cua" and "lo" as part of a searched value after folding case and accents (a shoe containing only "cua" is not returned).
Traces: SC-2 (CAT-001 AC-3, AC-4)
Check: fetch_url

**Scenario**: SC-2b — `Đại` is found as `dai` and as typed
**Given**: The seed plus FX-DAI.
**When**: `GET /v1/products?q=dai`, then `GET /v1/products?q=%C4%90%E1%BA%A1i` (that is `Đại`), then `GET /v1/products?q=%C4%91%E1%BA%A1i` (that is `đại`).
**Then**: Each answers 200 and `items` contains `tc-dai`; the second and third return the same set of ids as the first.
Traces: SC-2 (CAT-001 AC-5)
Check: fetch_url

**Scenario**: SC-3a — `q` searches the line's name
**Given**: The seed plus FX-LINEJOIN (a shoe of the line `gazelle` whose own fields do not contain "gazelle").
**When**: `GET /v1/products?q=gazelle&limit=100`.
**Then**: 200; `items` contains `tc-line-join` and `gazelle-indoor-solar-gold`; `total` is 2 and equals the number of items; every shoe whose `line` is `gazelle` is among the items.
Traces: SC-3 (CAT-001 AC-6)
Check: fetch_url

**Scenario**: SC-3b — `q` searches the category's name
**Given**: The seed plus FX-CATJOIN (a shoe of category `women` whose own fields do not contain "women").
**When**: `GET /v1/products?q=women&limit=100`.
**Then**: 200; `items` contains `tc-cat-join`, `samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz` and `handball-spezial-w-earth-strata`; every item has `category` `women`; `total` is 5.
Traces: SC-3 (CAT-001 AC-7)
Check: fetch_url

**Scenario**: SC-4a — `%` means itself
**Given**: The seed plus FX-PCT (`tc-pct` has `100%` in its description; `tc-pct-decoy` has "1000 grams").
**When**: `GET /v1/products?q=100%25&limit=100`, then `GET /v1/products?q=%25&limit=100`.
**Then**: The first answers 200 with exactly `tc-pct` (`total` 1) — not `tc-pct-decoy` and no seeded shoe. The second returns only shoes that contain a `%` in a searched field (`tc-pct` only; `total` 1), not the whole catalogue.
Traces: SC-4 (CAT-001 AC-8)
Check: fetch_url

**Scenario**: SC-4b — `_` means itself
**Given**: The seed plus FX-US.
**When**: `GET /v1/products?q=_&limit=100`, then `GET /v1/products?q=tc_underscore&limit=100`.
**Then**: The first answers 200 with exactly `tc-us` (`total` 1) — not `tc-us-decoy` and no seeded shoe (a wildcard `_` would return all). The second also returns exactly `tc-us`.
Traces: SC-4 (CAT-001 AC-9)
Check: fetch_url

**Scenario**: SC-5a — `category=a,b` is "any of" and `total` is their sum
**Given**: The seed (women 4, men 6, kids 2).
**When**: `GET /v1/products?category=women&limit=100`, `?category=kids&limit=100`, `?category=women,kids&limit=100`.
**Then**: The third answers 200 with every item's `category` either `women` or `kids`; its ids are the union of the first two; `total` is 6 = 4 + 2; no `men` shoe is present.
Traces: SC-5 (CAT-001 AC-10)
Check: fetch_url

**Scenario**: SC-5b — different parameters combine with "and"
**Given**: The seed.
**When**: `GET /v1/products?category=women,kids&line=samba&limit=100`, then `GET /v1/products?colour=black,white&limit=100`, then `GET /v1/products?line=samba,gazelle&category=men&limit=100`.
**Then**: The first returns exactly the five Samba pairs (`total` 5; `handball-spezial-w-earth-strata` is absent). The second returns exactly `samba-og-w-core-black`, `megaride-ag-xl-core-black` (black) and `samba-og-w-crystal-white`, `megaride-s2-cloud-white`, `samba-og-kids-cloud-white` (white): `total` 5. The third returns exactly `gazelle-indoor-solar-gold` (`total` 1; the Samba shoes are not `men`).
Traces: SC-5 (CAT-001 AC-11, item 3)
Check: fetch_url

**Scenario**: SC-5c — an empty value is ignored
**Given**: The seed.
**When**: `GET /v1/products?limit=100`, then `GET /v1/products?category=&limit=100`, then `GET /v1/products?category=&line=&size=&q=&limit=100`.
**Then**: All three answer 200 with the same twelve ids in the same order and `total` 12.
Traces: SC-5 (CAT-001 AC-12)
Check: fetch_url

**Scenario**: SC-6a — a price range includes both ends and compares numbers as numbers
**Given**: The seed (prices: 2290000 ×2, 2790000, 3090000, 3290000 ×2, 3590000, 3790000, 4090000, 5090000, 5590000, 5890000).
**When**: `GET /v1/products?price_from=3000000&price_to=3999999&limit=100`; then `?price_from=3290000&price_to=3290000&limit=100`; then `?price_from=999999&limit=100`; then `?price_to=10000000&limit=100`.
**Then**: The first returns exactly `samba-og-w-core-black`, `samba-og-w-crystal-white`, `handball-spezial-w-earth-strata`, `gazelle-indoor-solar-gold`, `adizero-evo-sl-charcoal` (`total` 5): it includes the 3090000 and 3290000 shoes and not the 2790000 shoe (`samba-og-w-wonder-quartz`) or the 4090000 shoe. The second returns exactly `samba-og-w-core-black` and `handball-spezial-w-earth-strata` (`total` 2: both ends included). The third and fourth each return all 12 shoes — a text comparison of "999999" or "10000000" with prices would drop most of them.
Traces: SC-6 (CAT-001 AC-13)
Check: fetch_url

**Scenario**: SC-6b — a shoe priced exactly 3000000 is returned when both ends are 3000000
**Given**: The seed plus FX-3M (live, price exactly 3000000).
**When**: `GET /v1/products?price_from=3000000&price_to=3000000`, then `?price_from=3000001&price_to=3999999&limit=100`, then `?price_from=2000000&price_to=2999999&limit=100`.
**Then**: The first answers 200 with exactly `tc-price-3m` (`total` 1). The second does not contain `tc-price-3m`; the third does not contain it either.
Traces: SC-6 (CAT-001 AC-14)
Check: fetch_url

**Scenario**: SC-6c — decimal and date ranges compare as numbers and dates
**Given**: The seed.
**When**: `GET /v1/products?rating_from=4.8&limit=100` and `GET /v1/products?released_from=2026-11-01&released_to=2026-11-14&limit=100`; and `GET /v1/reviews?date_from=2026-11-20&limit=100` and `GET /v1/reviews?date_from=2026-11-20&date_to=2026-11-20&limit=100`.
**Then**: The first returns exactly the shoes of the unfiltered list whose `rating` is 4.8 or more (on the seed's stored values: `samba-og-w-core-black`, `samba-og-w-crystal-white`, `gazelle-indoor-solar-gold`, `adizero-evo-sl-charcoal`, `megaride-bones-halo-green`, `samba-og-kids-wonder-mauve`, `samba-og-kids-cloud-white`). The second returns exactly `samba-og-w-core-black` (2026-11-03), `samba-og-w-crystal-white` (2026-11-10) and `samba-og-kids-wonder-mauve` (2026-11-14, the end date is included) — `total` 3. The third returns reviews `r-01`, `r-08`, `r-14`, `r-22`, `r-25`, `r-28`, `r-31` (`total` 7). The fourth returns only `r-28` (`total` 1).
Traces: SC-6 (CAT-001 item 4); SC-14
Check: fetch_url

**Scenario**: SC-7a — `sort=price` rises and `sort=-price` falls
**Given**: The seed.
**When**: `GET /v1/products?sort=price&limit=100` and `GET /v1/products?sort=-price&limit=100`.
**Then**: Both answer 200 with 12 items. In the first, `price` never decreases from one item to the next and the first two ids are `samba-og-kids-wonder-mauve`, `samba-og-kids-cloud-white` (both 2290000, in seed order) and the last is `megaride-bones-halo-green` (5890000). In the second, `price` never increases, the first is `megaride-bones-halo-green`, and the two 3290000 shoes appear as `samba-og-w-core-black` then `handball-spezial-w-earth-strata` (equal values keep seed order in both directions).
Traces: SC-7 (CAT-001 AC-15, AC-16)
Check: fetch_url

**Scenario**: SC-7b — `sort=-released` puts the most recent release first
**Given**: The seed.
**When**: `GET /v1/products?sort=-released&limit=100`.
**Then**: 200; `released` never increases along the items; the first is `gazelle-indoor-solar-gold` (2026-11-20), the second `samba-og-w-wonder-quartz` (2026-11-17), the last `megaride-ag-xl-core-black` (2026-08-30).
Traces: SC-7 (CAT-001 AC-17)
Check: fetch_url

**Scenario**: SC-7c — `sort=-rating` puts the best rated first and an unrated shoe last
**Given**: The seed plus FX-NORATING (`rating` 0, the stored "no rating").
**When**: `GET /v1/products?sort=-rating&limit=100`; then `GET /v1/lines?sort=since&limit=100` and `GET /v1/lines?sort=-since&limit=100` with FX-LINENULL added.
**Then**: The first: `rating` never increases along the items; the first item's `rating` equals the highest `rating` in the unfiltered list (4.9 on the seed: `samba-og-w-crystal-white` first, then `megaride-bones-halo-green`, then `samba-og-kids-wonder-mauve`, which are equal and keep seed order); `tc-no-rating` is the last of 13 items. The lines answers both end with `tc-line-null` (no value comes last either way, ascending and descending); the other five lines are ordered by `since` ascending in the first and descending in the second.
Traces: SC-7 (CAT-001 AC-18; contract GET /v1/{collection} `sort`)
Check: fetch_url

**Scenario**: SC-8 — equal values keep seed order, so pages of one repeat and skip nothing
**Given**: The seed; two pairs cost 2290000 and two cost 3290000.
**When**: `GET /v1/products?sort=price&limit=1&page=N` for N = 1 … 12, then N = 13; then the same for `sort=-price`.
**Then**: Pages 1 … 12 each hold one item and `total` 12; the twelve ids are all different and their union is the twelve seeded ids (each shoe exactly once); the order of the ids equals the order of `GET /v1/products?sort=price&limit=100` (and, for the second run, of `sort=-price&limit=100`). Page 13 answers 200 with `items: []` and `total` 12.
Traces: SC-8 (CAT-001 AC-19)
Check: fetch_url

**Scenario**: SC-9a — `size=38` returns only shoes where 38 is listed and not sold out
**Given**: The seed, plus FX-SIZE38-2-3 (only `38 2/3` listed). `handball-spezial-w-earth-strata` lists 38 as sold out.
**When**: `GET /v1/products?size=38&limit=100`.
**Then**: 200; `items` are exactly `samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `samba-og-kids-wonder-mauve`, `samba-og-kids-cloud-white` (`total` 5). Every item lists `38` in `sizes` and not in `sizes_out`. `handball-spezial-w-earth-strata` is not among them, and neither is `tc-size-38-2-3` (`38` is not read as a start of `38 2/3`).
Traces: SC-9 (CAT-001 AC-20, AC-21)
Check: fetch_url

**Scenario**: SC-9b — a size on its last pairs still counts, a sold-out size does not
**Given**: The seed. `samba-og-w-core-black` lists `40` among its last-pairs sizes and not as sold out; `gazelle-indoor-solar-gold` lists `40` as sold out.
**When**: `GET /v1/products?size=40&limit=100`.
**Then**: 200; the items include `samba-og-w-core-black` and exclude `gazelle-indoor-solar-gold`; they are exactly `samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `handball-spezial-w-earth-strata`, `adizero-adios-wonder-blue`, `adizero-evo-sl-charcoal`, `megaride-s2-cloud-white` (`total` 7).
Traces: SC-9 (CAT-001 AC-22)
Check: fetch_url

**Scenario**: SC-9c — several sizes are "either", and a size with a fraction is matched whole
**Given**: The seed plus FX-SIZE38-2-3.
**When**: `GET /v1/products?size=38,40&limit=100`; then `GET /v1/products?size=38%202%2F3&limit=100`.
**Then**: The first returns the union of the answers for `size=38` and `size=40`: nine shoes (`total` 9) — the five of SC-9a and the seven of SC-9b, once each. The second returns the shoes that list `38 2/3` and have it available: `samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `handball-spezial-w-earth-strata`, `samba-og-kids-wonder-mauve` and `tc-size-38-2-3` (`total` 6); `samba-og-kids-cloud-white` is absent (its `38 2/3` is sold out).
Traces: SC-9 (CAT-001 AC-23)
Check: fetch_url

**Scenario**: SC-10a — women's Samba in EU 38 under 3 million returns exactly one pair
**Given**: The seed.
**When**: `GET /v1/products?category=women&line=samba&size=38&price_to=2999999`.
**Then**: 200 with exactly one item, `samba-og-w-wonder-quartz`, and `total` 1.
Traces: SC-10 (CAT-001 AC-24)
Check: fetch_url

**Scenario**: SC-10b — every condition of one request applies together
**Given**: The seed.
**When**: `GET /v1/products?category=women&line=samba&size=38&price_to=2999999&q=og&sort=price&status=live`, then the same with `q=zzzz`, then the same with `size=99`, then the same with `price_to=2000000`.
**Then**: The first returns exactly `samba-og-w-wonder-quartz`, `total` 1. The other three each answer 200 with `total` 0 and `items: []`.
Traces: SC-10 (CAT-001 item 8)
Check: fetch_url

**Scenario**: SC-11 — a draft shoe is kept out by `status=live`
**Given**: The seed plus FX-DRAFT (a draft Samba OG for women with 38 available at 2500000, matching every other condition below).
**When**: `GET /v1/products?q=samba%20og&category=women&line=samba&size=38&price_to=2999999&status=live`, then `GET /v1/products?status=live&limit=100`.
**Then**: The first returns exactly `samba-og-w-wonder-quartz` (`total` 1); `tc-draft-samba` is not among the items. The second returns 12 items and `total` 12, none of them `tc-draft-samba`, and every item's `status` is `live`.
Traces: SC-11 (CAT-001 AC-25)
Check: fetch_url

**Scenario**: SC-12 — page 2 of 30 matches holds twelve and `total` is 30
**Given**: The seed plus FX-Q30 (`q=samba` has exactly 30 matches).
**When**: `GET /v1/products?q=samba&limit=100` (the reference), then `GET /v1/products?q=samba&page=1&limit=12`, `page=2&limit=12`, `page=3&limit=12`, `page=4&limit=12`.
**Then**: The reference has 30 items and `total` 30. Page 2 answers 200 with `page` 2, `limit` 12, 12 items equal to items 13 … 24 of the reference and `total` 30. Pages 1 and 3 hold 12 and 6 items; the three pages together hold the 30 reference ids, each once. Page 4 answers 200 with `items: []` and `total` 30.
Traces: SC-12 (CAT-001 AC-26)
Check: fetch_url

**Scenario**: SC-13a — `q` over orders searches the number and the customer, not the address or note
**Given**: A dealer session. The seed orders (`DOI-24817` has customer "Hoàng Nam") plus FX-ORDERS (an order whose address contains "Nam" and one whose note contains "Nam", neither with "nam" in its number or customer).
**When**: `GET /v1/orders?q=nam`, then `GET /v1/orders?q=NAM`, then `GET /v1/orders?q=hoang%20nam`, then `GET /v1/orders?q=24817`.
**Then**: The first two return exactly `doi-24817` (`total` 1); `DOI-90001` (address) and `DOI-90002` (note) are not returned. The third returns exactly `doi-24817`. The fourth returns exactly `doi-24817` (the order number is searched).
Traces: SC-13 (CAT-001 AC-27)
Check: fetch_url

**Scenario**: SC-13b — a private field is never searched
**Given**: A dealer session. The seed order `DOI-24817` has phone `0938 410 552` and address `42 Example Street, Hải Châu`; `DOI-24816` has phone `0977 302 118`.
**When**: `GET /v1/orders?q=0938410552`, then `?q=0938`, then `?q=410%20552`, then `?q=hai%20chau`, then `?q=example`.
**Then**: Each answers 200 with `total` 0 and `items: []` — a phone, a digit run from it, and words from the address match nothing.
Traces: SC-13 (CAT-001 Permission, AC-27)
Check: fetch_url

**Scenario**: SC-14a — a shoe's reviews, newest first
**Given**: The seed reviews of `samba-og-w-core-black` (`r-01` 2026-11-21, `r-02` 2026-11-16, `r-03` 2026-11-08).
**When**: `GET /v1/reviews?product=samba-og-w-core-black&sort=-date`.
**Then**: 200 with exactly `r-01`, `r-02`, `r-03` in that order and `total` 3.
Traces: SC-14 (CAT-001 AC-28)
Check: fetch_url

**Scenario**: SC-14b — `q` over reviews searches the customer, the text and the shoe's name
**Given**: The seed reviews.
**When**: `GET /v1/reviews?q=hanh`, then `?q=wedding`, then `?q=solar%20gold`, then `?q=earth%20strata&sort=-date`.
**Then**: The first returns `r-05` (customer "Hạnh"); the second returns `r-01` (text "…a wedding…"); the third returns `r-13`, `r-14`, `r-15` (the shoe's name "Gazelle Indoor Solar Gold", although the review text has neither word); the fourth returns `r-10`, `r-11`, `r-12` newest first (2026-11-10, 2026-10-24, 2026-10-12). Each answer's `total` equals its item count.
Traces: SC-14 (CAT-001 item 2)
Check: fetch_url

**Scenario**: SC-14c — lines, categories and other collections are searched on their own fields
**Given**: The seed.
**When**: `GET /v1/lines?q=frozen%20pitches`; `GET /v1/lines?q=roomier`; `GET /v1/categories?q=grade-school`; `GET /v1/categories?q=kids`; `GET /v1/faqs?q=exchanges`.
**Then**: The first returns exactly the line `samba` (its blurb). The second returns `total` 0: "roomier" is only in a line's fit note, which is not a searched field for lines. The third returns exactly the category `kids` (its blurb) and the fourth exactly `kids` (its name). The fifth returns at least `f-returns-size`, and every item contains "exchanges" in one of its text fields (`topic`, `question`, `answer`).
Traces: SC-14 (CAT-001 item 2; contract GET /v1/{collection} searched fields)
Check: fetch_url

**Scenario**: SC-15a — an unknown sort field is a 400 naming `sort`
**Given**: The seed.
**When**: `GET /v1/products?sort=zzz`, then `?sort=-zzz`, then `GET /v1/reviews?sort=price`, then `GET /v1/lines?sort=price`.
**Then**: Each answers 400 `bad_request` with `errors.sort` a non-empty sentence, and no `items`.
Traces: SC-15 (CAT-001 Invalid input)
Check: fetch_url

**Scenario**: SC-15b — a range value that is not a number is a 400 naming that parameter
**Given**: The seed.
**When**: `GET /v1/products?price_from=abc`, then `?price_to=12x`, then `?price_from=3000000&price_to=cheap`, then `?rating_from=high`.
**Then**: Each answers 400 `bad_request`; `errors` names `price_from`, `price_to`, `price_to` and `rating_from` respectively, and only that parameter; no `items`.
Traces: SC-15 (CAT-001 Invalid input)
Check: fetch_url

**Scenario**: SC-15c — a date range value that is not a date is a 400 naming that parameter
**Given**: The seed.
**When**: `GET /v1/products?released_to=2026-13-45`, then `?released_from=yesterday`, then `?released_to=2026-02-30`, then `GET /v1/reviews?date_from=20261103`.
**Then**: Each answers 400 `bad_request`; `errors` names `released_to`, `released_from`, `released_to` and `date_from` respectively; no `items`. `?released_to=2026-11-03` answers 200.
Traces: SC-15 (CAT-001 Invalid input)
Check: fetch_url

**Scenario**: SC-15d — `q` of 101 characters is a 400, 100 is accepted, spaces are not counted
**Given**: The seed.
**When**: `GET /v1/products?q=` + 101 letters "a"; then `q=` + 100 letters "a"; then `q=` + 100 letters "a" followed by 5 encoded spaces (`%20`); then `q=` + 101 letters "a" followed by one space.
**Then**: The first and fourth answer 400 `bad_request` with `errors.q` a non-empty sentence and no `items`. The second and third answer 200 with `total` 0 (no shoe contains that text) — the limit is 100 characters after trimming.
Traces: SC-15 (CAT-001 Invalid input; contract §2.4 `q`)
Check: fetch_url

**Scenario**: SC-15e — a filter or sort on a private field is a 400 for every caller
**Given**: The seed; one request as a visitor, one as a dealer session.
**When**: `GET /v1/products?sort=price_listed`, then `GET /v1/products?price_listed=US%24130`, then (as a dealer) `GET /v1/orders?sort=phone` and `GET /v1/orders?phone=0938410552`.
**Then**: Each answers 400 `bad_request` and `errors` names the field (`sort` for the first and third; `price_listed` and `phone` for the others); no record is returned.
Traces: contract (GET /v1/{collection} private fields, §2.4); SC-13
Check: fetch_url

**Scenario**: SC-15f — a parameter the contract does not define is ignored on a list
**Given**: The seed.
**When**: `GET /v1/products?limit=100`, then `GET /v1/products?zzz=1&limit=100`.
**Then**: Both answer 200 with the same twelve ids in the same order and `total` 12 (a URL a person edited is not refused).
Traces: contract (GET /v1/{collection}, DATA-002 AC-11)
Check: fetch_url

**Scenario**: SC-16a — a `q` of only spaces is no search
**Given**: The seed.
**When**: `GET /v1/products?q=%20%20&limit=100`, then `GET /v1/products?q=%20&limit=100`, then `GET /v1/products?limit=100`.
**Then**: All answer 200 with the same twelve ids in the same order and `total` 12.
Traces: SC-16 (CAT-001 Boundary)
Check: fetch_url

**Scenario**: SC-16b — a size no shoe has is an empty answer, not an error
**Given**: The seed.
**When**: `GET /v1/products?size=99`.
**Then**: 200 with `items: []`, `total` 0, `page` 1.
Traces: SC-16 (CAT-001 Boundary)
Check: fetch_url

**Scenario**: SC-16c — `size` on a collection that is not shoes is an ordinary field filter
**Given**: A dealer session. The seed orders: `DOI-24810` and `DOI-24804` have `size` `38`; `DOI-24816`, `DOI-24814`, `DOI-24812` and `DOI-24806` have `size` `37 1/3`.
**When**: `GET /v1/orders?size=38`, then `GET /v1/orders?size=37%201%2F3`, then `GET /v1/lines?size=38`, then `GET /v1/lines`.
**Then**: The first returns exactly the two orders `doi-24810` and `doi-24804` (`total` 2; no order of `38 2/3`, `43 1/3` or any other size). The second returns exactly the four `37 1/3` orders (`total` 4). The third and fourth return the same five lines (`lines` has no `size` field, so it is ignored).
Traces: SC-16 (CAT-001 Boundary; contract `size=a,b`)
Check: fetch_url

**Scenario**: SC-17 — the database cannot be reached: 500 with a generic body
**Given**: The backend running; `set_db_available` takes the database away so it refuses connections.
**When**: `GET /v1/products?q=samba&category=women&sort=price`.
**Then**: 500; body `{"error":"internal","message":<non-empty sentence>}` with no `errors` key and no `items`; the body text contains none of: `SELECT`, `FROM`, `pgx`, `postgres`, `connection refused`, `doi_fold`, `products`, `goroutine`, `.go:`. After `set_db_available` puts the database back, the same request answers 200.
Traces: SC-17 (CAT-001 Upstream failure; contract §2.3 `internal`)
Check: fetch_url

**Scenario**: SC-17b — a failed query is 500 with the same generic body
**Given**: The backend running; `set_db_state` renames the table `lines` to `lines_tc` (the search joins the line's name).
**When**: `GET /v1/products?q=samba`.
**Then**: 500, `{"error":"internal","message":<non-empty sentence>}`, no `errors` key, no SQL, table or column name (`lines`, `lines_tc`, `line_id`) and no stack trace in the body. After the table is renamed back, the same request answers 200.
Traces: SC-17 (CAT-001 Upstream failure)
Check: fetch_url

## 6. The shop's options and speed (backend) — SC-18, SC-19

**Scenario**: SC-18a — the options of the seeded live shoes, in the closed-set order and growing size order
**Given**: The seed (12 live shoes).
**When**: `GET /v1/product-filters`.
**Then**: 200 `Content-Type: application/json; charset=utf-8`; the body has exactly the keys `colours` and `sizes`. `colours` equals `["black","white","pink","purple","brown","yellow","blue","grey","green"]` (nine, closed-set order, not alphabetical). `sizes` equals `["35 1/2","36","36 2/3","37 1/3","38","38 2/3","39 1/3","40","40 2/3","41 1/3","42","42 2/3","43 1/3","44","44 2/3","45 1/3"]` (16, by value: `36 2/3` follows `36`, `35 1/2` comes first).
Traces: SC-18 (CAT-002 AC-18, AC-19)
Check: fetch_url

**Scenario**: SC-18b — a draft contributes nothing, a sold-out size is still listed, parameters are ignored
**Given**: The seed plus FX-DRAFT changed to list only size `47` and colour `green`; and one live fixture shoe `tc-size-46` listing only size `46` with 0 pairs (sold out); and the seeded `megaride-bones-halo-green` (the only green live shoe) set to `draft`.
**When**: `GET /v1/product-filters`, then `GET /v1/product-filters?colour=black&size=38&q=zzz&category=women&limit=1&page=9`.
**Then**: Both answer 200 with the same body. `colours` has eight entries and no `green` (the draft fixture's green and the draft seed shoe add nothing); `sizes` ends `…,"45 1/3","46"` — `46` (sold out, but listed by a live shoe) is present and `47` (draft only) is absent. Afterwards the seed shoe is set back to `live`.
Traces: SC-18 (CAT-002 item 8; contract §3.4)
Check: fetch_url

**Scenario**: SC-18c — a failed read of the options is 500
**Given**: The backend running; `set_db_available` takes the database away.
**When**: `GET /v1/product-filters`.
**Then**: 500, `{"error":"internal","message":<non-empty sentence>}` with no `errors` key and no SQL, table or column name or stack trace in the body.
Traces: contract (GET /v1/product-filters, `internal` 500)
Check: fetch_url

**Scenario**: SC-19 — 200 sequential searches over 10,000 shoes stay within 500 ms at the 95th percentile
**Given**: The seed plus FX-10K (10,000 live shoes), the backend running with its normal database.
**When**: 200 sequential `GET /v1/products?q=samba%20og&category=women&sort=-price` requests (one at a time, each after the previous answer), each timed from request to full response.
**Then**: Every answer is 200 with `total` equal to the same number and 12 items. The 95th percentile of the 200 durations (the 190th value of the sorted durations) is 500 ms or less. FX-10K is deleted afterwards.
Traces: SC-19 (CAT-001 Performance, SRS §6)
Check: fetch_url

## 7. The shop page reads pages from the API — SC-20 … SC-32

All cases in this section and the next load pages in a 1440×790 window.

**Scenario**: SC-20a — 30 live shoes: twelve cards, `30 matching`, and a "Show more" button
**Given**: The seed plus FX-LIVE30 (30 live shoes).
**When**: Open `/shop` and wait until the cards are drawn.
**Then**: 12 cards (12 distinct `href`s under `/shop/`); the count text reads `30 matching`; a button named exactly "Show more" is present.
Traces: SC-20 (CAT-002 AC-1)
Check: render_url

**Scenario**: SC-20b — 12 live shoes: twelve cards, `12 matching`, no button
**Given**: The seed only (12 live shoes).
**When**: Open `/shop`.
**Then**: 12 cards; the count text reads `12 matching`; no button named "Show more" exists.
Traces: SC-20 (CAT-002 AC-4)
Check: render_url

**Scenario**: SC-21a — "Show more" adds a dozen at a time until everything is shown
**Given**: The seed plus FX-LIVE30. `/shop` is open with 12 cards.
**When**: Press "Show more"; read the page; press "Show more" again; read the page.
**Then**: After the first press: 24 cards, all `href`s different, the count still reads `30 matching`, "Show more" still present. After the second press: 30 cards, all `href`s different (no card twice, none of the 30 live shoes missing), the count still reads `30 matching`, and no "Show more" button exists.
Traces: SC-21 (CAT-002 AC-2, AC-3)
Check: interact_page

**Scenario**: SC-21b — 13 matches: twelve cards and a button; pressing it adds one card
**Given**: The seed plus FX-LIVE13 (13 live shoes).
**When**: Open `/shop`; read the page; press "Show more"; read the page.
**Then**: First read: 12 cards, `13 matching`, "Show more" present. After the press: 13 cards, all `href`s different, `13 matching`, and no "Show more" button.
Traces: SC-21 (CAT-002 Boundary)
Check: interact_page

**Scenario**: SC-22 — the shop asks for twelve live shoes at a time and for the lines once
**Given**: The seed plus FX-LIVE30; a browser whose network requests during the page's life are listed.
**When**: Open `/shop`, wait for the cards, press "Show more" once, then list the requests made.
**Then**: Every request to `/api/v1/products` carries `limit=12` and `status=live` (the second carries `page=2`); no request has a `limit` over 100 (none over 12); the requests whose path starts `/api/v1/lines` number at most one in all (not one per card). The page shows 24 cards, which also shows no request asked for more than twelve at a time.
Traces: SC-22 (CAT-002 AC-5)
Check: render_url

**Scenario**: SC-23a — a draft shoe is not listed and not counted
**Given**: The seed plus FX-DRAFT (a draft shoe named "Samba OG Draft Tc").
**When**: Open `/shop`; then open `/shop?q=Samba%20OG%20Draft%20Tc`.
**Then**: The first shows 12 cards and `12 matching`; no card links to `/shop/tc-draft-samba`. The second shows no card linking to `/shop/tc-draft-samba` and the count reads `0 matching`.
Traces: SC-23 (CAT-002 AC-6, AC-7)
Check: render_url

**Scenario**: SC-24 — the heading is "All shoes", or the search words
**Given**: The seed.
**When**: Open `/shop`; read `main h1` and the sort control; then open `/shop?q=samba` and read `main h1`.
**Then**: The first heading reads exactly `All shoes` and the sort control reads `Newest first`. The second heading reads exactly `Results for “samba”` (curly quotes) and shows 5 cards.
Traces: SC-24 (CAT-002 AC-8, AC-9)
Check: render_url

**Scenario**: SC-25a — one category names the heading and its picture
**Given**: The seed. The Women category's picture is `/images/ref/category-women.jpg`; the shop's default picture is `/images/ref/category-gifts.jpg`.
**When**: Open `/shop?category=women`; read `main h1` and the `src` of the picture in the same section as the heading.
**Then**: The heading reads exactly `Women`; the picture's `src` ends `/images/ref/category-women.jpg`; the count reads `4 matching`.
Traces: SC-25 (CAT-002 AC-10)
Check: render_url

**Scenario**: SC-25b — a category and a line join as "Women · Samba"
**Given**: The seed.
**When**: Open `/shop?category=women&line=samba`; read `main h1` and the picture `src`.
**Then**: The heading reads exactly `Women · Samba` (with the middle dot and single spaces); the picture is the Women category's; the count reads `3 matching`. Then `/shop?line=samba` reads `Samba` with the default picture (`/images/ref/category-gifts.jpg`) and `5 matching`.
Traces: SC-25 (CAT-002 AC-11)
Check: render_url

**Scenario**: SC-25c — two categories fall back to "All shoes" and the default picture
**Given**: The seed.
**When**: Open `/shop?category=women,men`; read `main h1` and the picture `src`.
**Then**: The heading reads exactly `All shoes`; the picture is `/images/ref/category-gifts.jpg`; the count reads `10 matching`.
Traces: SC-25 (CAT-002 AC-12)
Check: render_url

**Scenario**: SC-26a — a card shows name, price, line and sizes in stock, and links to the shoe
**Given**: The seed. `samba-og-w-core-black`: price 3290000, line Samba, nine sizes listed, one sold out.
**When**: Open `/shop`; find the card whose link goes to `/shop/samba-og-w-core-black`; read its text and `href`.
**Then**: The card's text contains `Samba OG Black Crystal`, `3.290.000 ₫` (the space before ₫ may be a no-break space) and `Samba · 8 sizes in stock`; its `href` is exactly `/shop/samba-og-w-core-black`.
Traces: SC-26 (CAT-002 AC-13)
Check: interact_page

**Scenario**: SC-26b — the card's name is drawn in capitals
**Given**: The seed; `/shop` is open.
**When**: Measure the name text inside the card that links to `/shop/samba-og-w-core-black`.
**Then**: The computed `text-transform` of the name element is `uppercase`.
Traces: SC-26 (CAT-002 item 4)
Check: measure_styles

**Scenario**: SC-27 — the sort control is named "Sort", reads "Newest first" and offers four options
**Given**: The seed.
**When**: Open `/shop`; read the control labelled "Sort"; open it and read its options.
**Then**: The control is reachable by the name `Sort`; it reads `Newest first`; its options, in order, are exactly `Newest first`, `Price, low to high`, `Price, high to low`, `Best rated` (four, no others).
Traces: SC-27 (CAT-002 AC-8, AC-17)
Check: interact_page

**Scenario**: SC-28a — "Price, low to high" and "Price, high to low" order the cards and write `sort`
**Given**: The seed; `/shop` is open.
**When**: Choose "Price, low to high" in the control "Sort", read the cards and the address; then choose "Price, high to low", read again.
**Then**: After the first choice the address query contains `sort=price`, the card prices (read from the cards) never decrease, the first card is one of the two 2.290.000 ₫ Samba Kids pairs and the last is `Megaride Bones Halo Green` (5.890.000 ₫). After the second the address contains `sort=-price` (`-price` may be written `%2Dprice`), prices never increase and the first card is `Megaride Bones Halo Green`.
Traces: SC-28 (CAT-002 AC-14)
Check: interact_page

**Scenario**: SC-28b — "Best rated" puts the highest rated first; "Newest first" removes `sort`
**Given**: The seed; `/shop` is open. The API's own order for `GET /v1/products?status=live&sort=-rating&limit=12` is the reference.
**When**: Choose "Best rated", read the first card and the address; then choose "Newest first", read the first card and the address.
**Then**: After "Best rated": the address contains `sort=-rating` and the first card links to the same shoe as the first item of the reference (on the seed `samba-og-w-crystal-white`). After "Newest first": the address contains no `sort` parameter and the first card links to `/shop/gazelle-indoor-solar-gold` (released 2026-11-20, the newest).
Traces: SC-28 (CAT-002 AC-15, AC-16)
Check: interact_page

**Scenario**: SC-29a — the Size (EU) pills and the Colour options list every live shoe's sizes and colours
**Given**: The seed (live shoes list sizes from `35 1/2` to `45 1/3` and use nine colours).
**When**: Open `/shop`; read the pills in the group "Size (EU)" and the options in the group "Colour".
**Then**: 16 pills in this order: `35 1/2`, `36`, `36 2/3`, `37 1/3`, `38`, `38 2/3`, `39 1/3`, `40`, `40 2/3`, `41 1/3`, `42`, `42 2/3`, `43 1/3`, `44`, `44 2/3`, `45 1/3`. Nine Colour options named `Black`, `White`, `Pink`, `Purple`, `Brown`, `Yellow`, `Blue`, `Grey`, `Green` (the site's colour names).
Traces: SC-29 (CAT-002 AC-18, AC-19)
Check: render_url

**Scenario**: SC-29b — the options do not shrink when the list is filtered
**Given**: The seed.
**When**: Open `/shop?category=women`; read the "Size (EU)" pills and the "Colour" options.
**Then**: The same 16 pills and the same nine colours as SC-29a (not only the women's four colours and nine sizes); the count reads `4 matching`.
Traces: SC-29 (CAT-002 AC-20)
Check: render_url

**Scenario**: SC-30 — the count is the API's total and every card has the size available
**Given**: The seed.
**When**: Open `/shop?category=women&size=38`; read the count and the cards; request `GET /v1/products?status=live&category=women&size=38` and read `total`.
**Then**: The count reads `3 matching`, equal to the API's `total` (3); the cards are `samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`; no card links to `/shop/handball-spezial-w-earth-strata` (its 38 is sold out).
Traces: SC-30 (CAT-002 AC-21)
Check: render_url

**Scenario**: SC-31 — a line that does not exist is an empty list, not an error
**Given**: The seed.
**When**: Open `/shop?line=zzz`; then request `GET /v1/products?status=live&line=zzz`.
**Then**: The page shows no cards and the count reads `0 matching`; the page contains no text "We could not load" and no other error message. The API answers 200 with `total` 0 and `items: []`.
Traces: SC-31 (CAT-002 Not found)
Check: render_url

**Scenario**: SC-32 — a late answer to an older request never replaces the newest results
**Given**: The seed; `/shop` is open with 12 cards. If the instrument can hold one response, the response to the first of the two requests below is held until the second has been drawn.
**When**: Within the same moment tick "Women" and then tick "Men" (two requests in flight), and let the page settle; if a response can be held, release the held first answer after the second has been drawn.
**Then**: The address carries `category=women,men`, the count reads `10 matching` and the 10 cards are the women's and men's shoes (never the 4 of the first request alone), and they stay the same 10 after the held answer is released.
Traces: SC-32 (CAT-002 Conflict)
Check: interact_page

## 8. What the visitor sets reaches the request (address bar) — SC-33 … SC-40

**Scenario**: SC-33a — ticking boxes writes the address; one group is "any of", different groups are "all of"
**Given**: The seed; `/shop` open.
**When**: Tick "Women" under For; read the address and the list. Then tick "Men"; read again. Then untick "Men" and tick the line "Samba"; read again. Then tick "Kids"; read again.
**Then**: After "Women": the address carries `category=women`; 4 cards (`samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `handball-spezial-w-earth-strata`); `4 matching`. After "Men" as well: the address carries `category=women,men` (a comma may be written `%2C`); 10 cards; `10 matching`. After removing "Men" and ticking "Samba": `category=women` and `line=samba`; 3 cards (the three women's Samba); `3 matching`. After ticking "Kids" too: `category=women,kids` and `line=samba`; 5 cards, all Samba, in Women or Kids; `5 matching`.
Traces: SC-33 (CAT-003 AC-1, AC-2, AC-3, item 4)
Check: interact_page

**Scenario**: SC-33b — after any tick the list is back to its first twelve
**Given**: The seed plus FX-LIVE30 (women 22, women Samba 21, size 38 available 23). `/shop` is open.
**When**: Press "Show more" (24 cards); tick "Women"; read the cards and the count. Reload `/shop`, press "Show more", then tick the line "Samba"; read. Reload `/shop`, press "Show more", then press the size pill `38`; read.
**Then**: After each tick or pill: exactly 12 cards (not 24), all with different `href`s, a "Show more" button is present, and the count reads `22 matching` after "Women", `23 matching` after "Samba" and `23 matching` after the pill `38`. Pressing "Show more" once more after a tick shows 24 cards, so the earlier "Show more" pages did not survive the tick but the button still works.
Traces: SC-33 (CAT-003 AC-20)
Check: interact_page

**Scenario**: SC-34a — a size pill writes `size`, shows as pressed and leaves only shoes with that size available
**Given**: The seed; `/shop` open.
**When**: Press the pill `38` in "Size (EU)"; read the address, the pill and the list. Press the same pill again; read the address.
**Then**: After the first press: the address carries `size=38`; the pill has `aria-pressed="true"` and every other size pill has `aria-pressed="false"`; 5 cards (`samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `samba-og-kids-wonder-mauve`, `samba-og-kids-cloud-white`), none of them `handball-spezial-w-earth-strata`; `5 matching`. After the second press: the address carries no `size`, the pill is `aria-pressed="false"` and 12 cards show.
Traces: SC-34 (CAT-003 AC-4)
Check: interact_page

**Scenario**: SC-34b — `size` on the shop is never read as a page size
**Given**: The seed plus FX-LIVE30; 23 shoes have size 38 available.
**When**: Open `/shop?size=38`; read the cards and the count; list the requests to `/api/v1/products`.
**Then**: 12 cards (not 23), the count reads `23 matching`, a "Show more" button is present, and every shoe request carries `limit=12` (never `limit=38`).
Traces: SC-34 (CAT-003 AC-4; story §10.2)
Check: render_url

**Scenario**: SC-35a — women's Samba in 38 under 3.000.000 ₫ is exactly one pair
**Given**: The seed.
**When**: Open `/shop?category=women&line=samba&size=38` (3 cards); tick the price band "Under 3.000.000 ₫".
**Then**: The address carries `price=under-3m` besides the other three parameters; exactly one card, linking to `/shop/samba-og-w-wonder-quartz`; the count reads `1 matching`.
Traces: SC-35 (CAT-003 AC-5)
Check: interact_page

**Scenario**: SC-35b — the band "3.000.000 ₫ – 4.000.000 ₫" includes 3.290.000 ₫ and excludes 2.790.000 ₫
**Given**: The seed.
**When**: Open `/shop`; tick the band "3.000.000 ₫ – 4.000.000 ₫"; read the cards and the count.
**Then**: The address carries `price=3m-4m`; 5 cards (`samba-og-w-core-black` at 3.290.000 ₫, `samba-og-w-crystal-white`, `handball-spezial-w-earth-strata`, `gazelle-indoor-solar-gold`, `adizero-evo-sl-charcoal`); no card links to `/shop/samba-og-w-wonder-quartz` (2.790.000 ₫); `5 matching`. Ticking "4.000.000 ₫ and over" instead shows the four shoes of 4.090.000 ₫ or more (`4 matching`).
Traces: SC-35 (CAT-003 AC-7, item 3)
Check: interact_page

**Scenario**: SC-36 — two bands are a union, and the count counts only those
**Given**: The seed.
**When**: Open `/shop`; tick "Under 3.000.000 ₫" and then "4.000.000 ₫ and over"; read the address, the cards and the count.
**Then**: The address carries `price=under-3m,4m-plus` (comma may be `%2C`); the count reads `7 matching`; the 7 cards are `samba-og-w-wonder-quartz`, `samba-og-kids-wonder-mauve`, `samba-og-kids-cloud-white`, `adizero-adios-wonder-blue`, `megaride-s2-cloud-white`, `megaride-ag-xl-core-black`, `megaride-bones-halo-green`; none of the five shoes priced 3.000.000 ₫ to 3.999.999 ₫ is shown.
Traces: SC-36 (CAT-003 AC-6, item 3)
Check: interact_page

**Scenario**: SC-37 — a copied address shows the same ticks and the same list
**Given**: The seed; `/shop` open in one window.
**When**: Tick "Women", the line "Samba", the pill `38` and the band "Under 3.000.000 ₫"; copy the address; open that address in a new, empty browser window; read the rail and the list.
**Then**: In the new window "Women", "Samba" and "Under 3.000.000 ₫" are checked, no other tick box is checked, the pill `38` is `aria-pressed="true"` and no other pill is; one card (`/shop/samba-og-w-wonder-quartz`); `1 matching`; the heading reads `Women · Samba`.
Traces: SC-37 (CAT-003 AC-8)
Check: interact_page

**Scenario**: SC-38a — typing words and pressing Enter writes `q`, sets the heading and keeps the words in the box
**Given**: The seed; `/shop` open. The search input is found by its label `Search by name, line, colour or code`.
**When**: Type `samba og` into that input and press Enter; read the address, `main h1`, the input's value and the count.
**Then**: The address `q` parameter decodes to `samba og` (written `q=samba%20og`, or `q=samba+og`); the heading reads exactly `Results for “samba og”`; the input still shows `samba og`; 5 cards and `5 matching`.
Traces: SC-38 (CAT-003 AC-10)
Check: interact_page

**Scenario**: SC-38b — clearing the box and pressing Enter removes `q`; words are trimmed
**Given**: The seed; `/shop?q=samba%20og` open with the box showing `samba og`.
**When**: Clear the box and press Enter; read the address and heading. Then type `   samba   ` (spaces either side) and press Enter; read the address, heading and box. Then type only spaces and press Enter; read the address and heading.
**Then**: After clearing: the address has no `q`, the heading reads `All shoes`, 12 cards. After `   samba   `: `q` decodes to `samba` (no spaces around it) and the heading reads `Results for “samba”`. After only spaces: the address has no `q` and the heading reads `All shoes`.
Traces: SC-38 (CAT-003 AC-11, item 5)
Check: interact_page

**Scenario**: SC-38c — the header search lands on the shop with the words in the box
**Given**: The seed; the home page `/` open.
**When**: Open the header's search, type `gazelle`, press Enter; read the address, `main h1`, the shop's search input and the cards.
**Then**: The address is `/shop?q=gazelle`; the heading reads `Results for “gazelle”`; the shop's search input shows `gazelle`; one card, linking to `/shop/gazelle-indoor-solar-gold`; `1 matching`.
Traces: SC-38 (CAT-003 AC-12)
Check: interact_page

**Scenario**: SC-39a — an unknown price band is ignored
**Given**: The seed.
**When**: Open `/shop?price=abc`; read the cards, the count and the Price group.
**Then**: 12 cards, `12 matching`, and no price tick box is checked; the page shows no error message.
Traces: SC-39 (CAT-003 Invalid input)
Check: render_url

**Scenario**: SC-39b — an unknown sort falls back to "Newest first"
**Given**: The seed.
**When**: Open `/shop?sort=zzz`; read the control "Sort" and the first card.
**Then**: The control reads `Newest first`; the first card links to `/shop/gazelle-indoor-solar-gold` (newest release); 12 cards and `12 matching`; the page shows no error message (the page did not pass `sort=zzz` on to the API, which would answer 400).
Traces: SC-39 (CAT-003 Invalid input)
Check: render_url

**Scenario**: SC-40a — while the first answer is on its way, "No pair matches that." is not shown
**Given**: The seed. If the instrument can hold the answer to `/api/v1/products` for 3 seconds, hold it; otherwise the read is made at the earliest moment after navigation, before the first cards are drawn.
**When**: Open `/shop` and read the page text during the wait.
**Then**: While no card has been drawn, the text "No pair matches that." is not on the page. When the answer arrives, 12 cards appear and `12 matching` is shown.
Traces: SC-40 (CAT-004 AC-7, narrowed to "not shown")
Check: interact_page

**Scenario**: SC-40b — after a failed request, "No pair matches that." is not shown
**Given**: The seed; `set_db_available` takes the database away so `GET /v1/products` answers 500.
**When**: Open `/shop`; wait 3 seconds; read the page text.
**Then**: The text "No pair matches that." is not on the page and no card is shown. After `set_db_available` puts the database back, reloading `/shop` shows 12 cards and `12 matching`.
Traces: SC-40 (CAT-004 AC-9, narrowed to "not shown")
Check: interact_page

**Scenario**: SC-40c — when the browser cannot reach the API, "No pair matches that." is not shown
**Given**: The seed. The browser is offline if the instrument can switch that on; otherwise the backend is stopped so the gateway answers 502 (a 5xx, the other failure the criterion names), and started again afterwards.
**When**: Open `/shop`; wait 3 seconds; read the page text.
**Then**: The text "No pair matches that." is not on the page and no card is shown.
Traces: SC-40 (CAT-004 AC-12, narrowed to "not shown")
Check: interact_page

## 9. Coverage of the story's acceptance criteria

| Criterion | Cases |
|---|---|
| SC-1 | SC-1a, SC-1b |
| SC-2 | SC-2a, SC-2b |
| SC-3 | SC-3a, SC-3b |
| SC-4 | SC-4a, SC-4b |
| SC-5 | SC-5a, SC-5b, SC-5c |
| SC-6 | SC-6a, SC-6b, SC-6c |
| SC-7 | SC-7a, SC-7b, SC-7c |
| SC-8 | SC-8 |
| SC-9 | SC-9a, SC-9b, SC-9c |
| SC-10 | SC-10a, SC-10b |
| SC-11 | SC-11 |
| SC-12 | SC-12 |
| SC-13 | SC-13a, SC-13b |
| SC-14 | SC-14a, SC-14b, SC-14c |
| SC-15 | SC-15a, SC-15b, SC-15c, SC-15d, SC-15e |
| SC-16 | SC-16a, SC-16b, SC-16c |
| SC-17 | SC-17, SC-17b |
| SC-18 | SC-18a, SC-18b, SC-18c |
| SC-19 | SC-19 |
| SC-20 | SC-20a, SC-20b |
| SC-21 | SC-21a, SC-21b |
| SC-22 | SC-22 |
| SC-23 | SC-23a |
| SC-24 | SC-24 |
| SC-25 | SC-25a, SC-25b, SC-25c |
| SC-26 | SC-26a, SC-26b |
| SC-27 | SC-27 |
| SC-28 | SC-28a, SC-28b |
| SC-29 | SC-29a, SC-29b |
| SC-30 | SC-30 |
| SC-31 | SC-31 |
| SC-32 | SC-32 |
| SC-33 | SC-33a, SC-33b |
| SC-34 | SC-34a, SC-34b |
| SC-35 | SC-35a, SC-35b |
| SC-36 | SC-36 |
| SC-37 | SC-37 |
| SC-38 | SC-38a, SC-38b, SC-38c |
| SC-39 | SC-39a, SC-39b |
| SC-40 | SC-40a, SC-40b, SC-40c |
| Contract only (not a story criterion) | SC-15e (private fields), SC-15f (undefined list parameters ignored), SC-18c (`product-filters` 500) |

Every case uses an instrument: `fetch_url` for the endpoint cases, `render_url`, `interact_page` and `measure_styles` for the page. None is `manual`. The two places where the instrument may fall short are named in their own `Given`: holding a response for 3 seconds (SC-32, SC-40a) and switching the browser offline (SC-40c); each has an arrangement that still exercises the criterion, and the part that needs the missing instrument is reported UNVERIFIED rather than failed.
