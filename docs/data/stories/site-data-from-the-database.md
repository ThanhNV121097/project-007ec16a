# Story — Site data from the database

Module: `data` · Plan item 3 · Priority P1 · Requirements: DATA-001, DATA-002, DATA-003, DATA-004, DATA-005 (`docs/data/SRS.md`)
Shape: fullstack. Technical constraints: `docs/architecture/overview.md`, `erd.md` (§2.1 field mapping, §8 migrations 1–3), `services.md` (§2.3 errors, §2.4 lists).

## 1. The story

As the **dealer** (the shop's owner or staff), I want every record I kept in the Studio — shoes, lines, categories, reviews, orders, shop terms, the delivery option, ways to pay, questions and the size chart — to live in the shop's database and be read by the site from a paged API, so that none of my records is lost, renamed or re-priced when the site is rebuilt, and the stories after this one (search, checkout, sign-in, admin) have somewhere to read and write.

Other actors, from the SRS: the **visitor** reads every collection with no session (minus the fields of DATA-004); the **developer** rewrites the frontend's one data module. Nobody writes a record in this story: records enter the database by migration only. Writes arrive with "Manage shoes, lines, categories, reviews and orders".

## 2. In scope

1. **Ten collection tables**, one per collection of `code/frontend/src/data/schema.json`: `categories`, `lines`, `products`, `reviews`, `orders`, `faqs`, and the four the ERD renames (D2) — `delivery_options` (collection `delivery`), `payment_methods` (`payments`), `shop_terms` (`shop`), `size_chart` (`sizes`) — plus `order_items` (one row per sample order, ERD D4). A column per schema field, text id as key, the record's source note as column `source`, its invented-field list as `invented`, `seq` for the stable order. Money is `bigint` VND, lists are `text[]`, references are foreign keys (ERD §3).
2. **Foundation objects of the first migration** (ERD §8 migration 1): the SQL functions `doi_fold(text)` and `set_updated_at()` and the table `rate_limits`. No endpoint uses them yet; they are created here so the search, track and sign-in stories, which run in parallel after this one, never create them twice.
3. **The seed migration**: every record of the ten JSON files, in each file's order, exactly as the files stand when this story starts (after "Free delivery on every order, wording to match"). Plain inserts: an id that already exists fails the migration loudly.
4. **`GET /v1/<collection>`** — a page of records, `{items, total, page, limit}`, `page` ≥ 1, `limit` default 12 and at most 100, exact-match filters on any non-private field (list fields: contains; true/false fields: `true`/`false`), the seed's order — and **`GET /v1/<collection>/<id>`**, for all ten collections, public, with no session.
5. **Privacy**: an order's `phone`, `email`, `address` and `note`, every record's `_source` and `_invented`, and a shoe's `price_listed` are left out of the database query for a caller with no session, and a filter on any of them is a 400.
6. **Frontend**: `src/data/index.ts` rewritten to call the API (same exports, same returns), the new `src/lib/api.ts` it calls (base URL, same-origin cookies, `ApiError`, 15-second timeout, one retry of a failed `GET`), and the ten JSON files other than `schema.json` deleted.
7. **The pull-request note** DATA-004 item 4 and the SRS §7 require (section 6, SC-25).

## 3. Out of scope

| Left out | Why, and where it goes |
|---|---|
| Free-text search (`q`), "any of" lists (`a,b`), ranges (`_from`/`_to`), sorting, the size filter | Accepted and **ignored** here (DATA-002 items 7–8). They are the story "Search, filters and sorting served by the API" |
| Creating, changing or deleting a record, uploads, validation of what a dealer types | "Manage shoes, lines, categories, reviews and orders". The `/admin` edit forms keep editing the browser's copy exactly as today and save nothing |
| Hiding draft shoes and hidden reviews from visitors | Needs the dealer's session to tell the two apart ("Dealer sign-in", then "Manage…"). The seed holds 12 live shoes and 36 shown reviews, so nothing hidden exists to leak until a dealer can create one |
| Placing, numbering or token-protecting an order; per-size stock counts (`product_sizes`, `product_stock`) | "Checkout and placing an order". Until then a shoe's `sizes`, `sizes_low`, `sizes_out` and `stock` are plain columns seeded as the file has them (ERD D3) |
| Sign-in, sessions, who is a dealer, a dealer reading private fields or `_source` | "Dealer sign-in and protected admin area" |
| Recalculating a shoe's or the shop's rating from the reviews | "Manage…" (ADMIN-013). The seeded `rating` and `review_count` are stored as they are |
| Any loading, empty or error screen, skeleton or retry button; any change to a component or page | The design draws none (SRS §5, design-system §4). Decision D-1 below |
| Rate-limit behaviour | Only the `rate_limits` table is created; "Track an order…" and "Dealer sign-in…" use it |
| The contact form | It posts to Agent Team's own API; the backend has no endpoint for it (overview §2) |
| CORS, compose, deploy and CI files | Committed with the repository; nobody edits them |

## 4. UI scope

No screen is added or redrawn, and no component or page file is edited. Every page that reads a record — Home, Shop, a shoe page, Cart, Checkout, Size chart, Returns, Shipping, Track, About, Contact, the header's list of lines, and the `/admin` pages — draws the same shoes, prices, names and rows as before; the only difference is that they arrive over the network from the database instead of from the bundle (SC-37).

States: the data module reports `loading` and `error`. **No page draws them** — the approved design has no loading or error screen, and SRS §5 forbids this story from inventing one. While an answer is awaited, or after a failure, a page shows what it already shows for empty data.

## 5. Decisions made here, so nobody has to ask

- **D-1 Loading and error views are not drawn.** This answers the open design question in `design-system.md` §4 and `overview.md` §8 (row 3). The hook's `loading` and `error` are real and tested (SC-29); drawing them is a design change the owner can ask for after the site is up. The one visible flaw this leaves — a shoe page that may say "not found" for a moment before its record arrives — is the story "Product page waits for the shoe before saying it is missing".
- **D-2 `src/lib/api.ts` is created by this story.** The overview (§5) gives it to this story so that checkout and sign-in, which run in parallel after it, share one client. The SRS said "no file outside `src/data/`"; DATA-005 item 1 and AC-13 were amended in this pull request to allow exactly that one file. No component, page, style, `content.json`, `theme.css`, `index.css` or `package.json` is touched.
- **D-3 A page past the last keeps its old behaviour.** `useCollection` has always shown the last page instead of an empty table when a filter narrowed the list while the person stood on page 7. With a server that answers `items: []` for a page past the end, the hook asks once for the last page (SC-32).
- **D-4 Timeouts and retries come from the service design** (`services.md`, timeouts table): `api.ts` aborts a request after 15 s and tries a failed `GET` once more; a 404 is an answer, not a failure, and is never retried (SC-33).
- **D-5 Between this story and "Search, filters and sorting served by the API" the shop is degraded on purpose.** The Shop page's search box, sort order, price band, size filter and a category, line or colour filter with more than one box ticked send parameters this story's API ignores, so they change nothing; one ticked box (`category=women`) and `status=live` work. The Home, Cart, Checkout, shoe and `/admin` pages ask for up to 500 or 1000 records and receive at most 100, which with 12 shoes and 14 orders changes nothing; the dashboard's `sort=-placed_at` is ignored and it sees orders in seed order. The search story should be built straight after this one.
- **D-6 One story, two pull requests.** The frontend pull request (step 4) holds `index.ts`, `api.ts`, their tests against a stubbed `fetch`, and the deletion of the ten JSON files. The backend pull request (step 7) holds the three migrations, the handlers and their tests; it reads the JSON files from `main` while it builds the seed. The frontend merges first, the backend straight after (steps 11–12), so the site never runs the new data module without its API for longer than the gap between two merges.
- **D-7 The public read of orders is accepted for now** (DATA-004 item 4): no real order exists until checkout does, and the 14 samples carry only invented customers. It is reported to the owner and written into the backend pull request (SC-25).

## 6. Acceptance criteria

Each is observable by someone with the running site, its API, its database or the repository. `SC-n [REQ AC-m]` names the SRS criterion it narrows; where the SRS gives a figure that depends on the JSON files, the figure here is the one at the time of writing, and the file as it stands when the story starts wins.

### 6.1 Storage and seed (DATA-001)

- **SC-1 [DATA-001 AC-1]:** On an empty database the migrations create the ten collection tables and seed each with exactly as many records as its JSON file holds: categories 3, delivery 1, faqs 12, lines 5, orders 14, payments 4, products 12, reviews 36, shop 1, sizes 17.
- **SC-2 [DATA-001 AC-2, AC-3, boundary]:** Values keep their type, their order and their emptiness. `samba-og-w-core-black` reads back with `price` 3290000 (a number), `sizes` `["36","36 2/3","37 1/3","38","38 2/3","39 1/3","40","40 2/3","41 1/3"]`, `sizes_low` `["37 1/3","40"]`, `sizes_out` `["41 1/3"]` (same order), `stock` 18, `rating` 4.8, `review_count` 23, `featured` true, `is_new` true, `released` `2026-11-03` and every other field as in the file. The size row `eu-35-1-2` has `cm` as the number 21.6. `samba-og-w-crystal-white` has `sizes_out` as the empty list `[]`, not null.
- **SC-3 [DATA-001 AC-4]:** Read directly from the database, order `doi-24817` equals the file field for field, private ones included: `number` `DOI-24817`, `status` `shipped`, `product` `samba-og-w-crystal-white`, `size` `39 1/3`, `qty` 1, `total` 3090000, `payment` `momo`, `paid` true, `delivery` `standard`, `carrier` `GHN`, `tracking` `GHN-PT27XM`, `eta` `2026-11-26`, `customer` `Hoàng Nam`, `phone` `0938 410 552`, `email` empty, `address` `42 Example Street, Hải Châu`, `city` `Đà Nẵng`, `note` `Gift — no price inside the box.` Its `placed_at` is the file's `2026-11-23T16:05:00` read as Vietnam time and written back as `2026-11-23T16:05:00+07:00`.
- **SC-4 [DATA-001 AC-5, item 2]:** After seeding, every shoe's `line` and `category`, every review's `product`, and every order's `product`, `payment` and `delivery` names an existing record — zero empty, zero dangling — and the database itself refuses a shoe whose `line` is `nope` (a foreign key, not an application check).
- **SC-5 [DATA-001 AC-6, item 5, conflict]:** After one shoe's price and one review's status are changed in the database, restarting the server (which runs the migrations again) keeps both changes and leaves every collection's count as it was, with no record twice. A seed that meets an id already present — a test inserts a `samba` line with another name first — fails the migration loudly, naming the id, and leaves that existing row untouched.
- **SC-6 [DATA-001 AC-7, DATA-002 item 5, AC-10]:** Every collection lists in the order of its file: the first shoe is `samba-og-w-core-black` and the last `samba-og-kids-cloud-white`; a record added after seeding comes after them; two identical requests with no change between them return the same items in the same order.
- **SC-7 [DATA-001 AC-8, AC-9, item 3]:** The `shop` record has none of `free_delivery_from`, `express_days`, `express_cities`, `exchange_days`, `refund_days`, and its `standard_days` and `dispatch_cutoff` are the file's (`2–4`, `15:00`). `delivery` holds exactly one record, `standard`, with `fee` 0 and no `express`. The seed migration itself contains no express row, no non-zero fee and none of those five columns, whatever the files held when it was written.
- **SC-8 [DATA-001 AC-10]:** All 14 sample orders have `delivery` `standard` and a `total` equal to their shoe's price times their pairs (fee 0): `doi-24815` is 5890000, `doi-24810` is 3290000, `doi-24806` is 3090000, and `doi-24812` (two pairs at 2790000) is 5580000.
- **SC-9 [DATA-001 item 1, ERD §8 migration 1]:** After the migrations, `doi_fold('Cửa Lò')` is `cua lo`, `doi_fold('Đại')` is `dai`, `doi_fold('100%')` is `100%`; `set_updated_at()` exists and changing a record on any collection table moves its `updated_at`; `rate_limits` exists; running the migrations a second time leaves all three unchanged.

### 6.2 The paged list (DATA-002)

- **SC-10 [DATA-002 AC-1, AC-12, item 1]:** With no parameters, `GET /v1/<collection>` for each of the ten collections answers 200 with exactly the keys `items`, `total`, `page`, `limit`, with `page` 1 and `limit` 12. `products` gives 12 items and `total` 12; every other collection's `total` equals its file's count (`sizes`: `total` 17 with 12 items). Through the proxy the same answer is at `/api/v1/<collection>`.
- **SC-11 [DATA-002 AC-2, AC-4, item 3]:** `GET /v1/products?page=2&limit=5` returns `adizero-adios-wonder-blue`, `adizero-evo-sl-charcoal`, `megaride-s2-cloud-white`, `megaride-bones-halo-green`, `megaride-ag-xl-core-black` in that order, with `total` 12, `page` 2, `limit` 5. `?page=9&limit=5` returns `items` `[]`, `total` 12, `page` 9.
- **SC-12 [DATA-002 AC-3, item 2]:** `?limit=500` is answered with `limit` 100 and at most 100 items; with 130 shoes in the table the first page holds exactly 100 and `total` is 130.
- **SC-13 [DATA-002 AC-5, AC-6, AC-7, AC-8, item 4]:** Filters are exact and combine with AND. `products?category=women` gives `total` 4, every item `category` `women`; adding `&status=live` keeps 4; `products?category=men&is_new=true` gives only `gazelle-indoor-solar-gold`; `products?featured=true` gives only `samba-og-w-core-black`; `faqs?topic=returns` gives exactly `f-returns-size` and `f-returns-wrong`; `reviews?product=samba-og-w-core-black` gives 3. On a list field a record matches when its list contains the value: `products?sizes_low=40` gives `samba-og-w-core-black` and `adizero-evo-sl-charcoal` (total 2), and `sizes_low=4` gives total 0 (no substring match).
- **SC-14 [DATA-002 AC-11, items 7–8]:** `?zzz=1`, `?q=samba`, `?sort=-price`, `?price_from=3000000&price_to=3999999`, `?category=women,kids` and `?size=40` on `products` each answer exactly as the request with no parameter does (same items, order and `total` 12).
- **SC-15 [DATA-002 invalid input]:** `page=0`, `page=-1`, `page=abc` and `page=1.5` each answer 400 with the error envelope and `errors.page`; the same four values for `limit` answer 400 with `errors.limit`; no `items` come back.
- **SC-16 [DATA-002 permission, DATA-004 AC-3, items 1–2]:** `orders?phone=0938%20410%20552` answers 400 naming `phone` and returns no order; the same for `email`, `address` and `note`; and for `_source` and `_invented` on any collection and `price_listed` on `products`. A filter on one of them is never answered with an empty list.
- **SC-17 [DATA-002 upstream failure, DATA-003 upstream failure]:** With the database unreachable, a list and a record request each answer 500 with `{"error":"internal","message":…}`; the body has no record data, SQL, stack trace, host name or connection string.

### 6.3 One record (DATA-003)

- **SC-18 [DATA-003 AC-1, AC-2, AC-4, AC-5]:** `GET /v1/products/gazelle-indoor-solar-gold` answers 200 with name `Gazelle Indoor Solar Gold`, price 3590000 and its three lists, and equals its copy in the list field for field. `GET /v1/shop/shop` answers 200 with `standard_days` and `dispatch_cutoff` and no `free_delivery_from`. `GET /v1/delivery/standard` answers 200 with `fee` 0; `GET /v1/delivery/express` answers 404.
- **SC-19 [DATA-003 AC-3, not found]:** `GET /v1/products/nope`, `GET /v1/nothing`, `GET /v1/nothing/x` and an id containing a character outside `a-z 0-9 . _ -` each answer 404 with the error envelope (`error` `not_found`) — never an empty 200.

### 6.4 What a stranger may not read (DATA-004)

- **SC-20 [DATA-004 AC-1, AC-2, item 1]:** With no session, `GET /v1/orders/doi-24817` answers 200 without the keys `phone`, `email`, `address` and `note` (`email` is absent although the file holds an empty one), with `customer` `Hoàng Nam` and `city` `Đà Nẵng` present; none of the 14 items of `GET /v1/orders` has any of the four keys.
- **SC-21 [DATA-004 AC-4, AC-5, item 2]:** No item of `GET /v1/products` and no `GET /v1/products/<id>` has `_source`, `_invented` or `price_listed`; `GET /v1/lines/samba` has no `_source` or `_invented` and has `name`, `blurb`, `since` and `fit_note`; no response of any of the ten collections has `_source` or `_invented`.
- **SC-22 [DATA-004 AC-6, AC-7, AC-8, items 3, 5]:** Every shoe still has `swatch`, `image`, `image_back` and `image_detail`; every way to pay has `icon` and `online`; `GET /v1/faqs` answers 200 with no session and `total` 12, and `GET /v1/faqs/f-returns-size` answers 200.
- **SC-23 [DATA-004 AC-9, NFR Privacy]:** Against the deployed API with no cookie, every collection is listed page by page (`limit=100`) and every record read by id; no body contains any sample order's phone, address or note text (for example `0938 410 552`, `42 Example Street, Hải Châu`, `Gift — no price inside the box.`), any source address (`featuresneakerboutique`) or any listed price (`US$130`). The deployment's private-field check passes.
- **SC-24 [DATA-004 item 1]:** A backend test captures the SQL that a no-session read sends to the database: for `orders` the column list names none of `phone`, `email`, `address`, `note`; for every collection it names neither `source` nor `invented`; for `products` it does not name `price_listed`. The columns are left out of the query, not deleted from the answer afterwards.
- **SC-25 [DATA-004 item 4, SRS §7 note]:** The backend pull request states (a) that the public read of orders is temporary and ends when "Checkout and placing an order", "Track an order by number and phone" and "Dealer sign-in and protected admin area" make orders readable only with an order token or a dealer session, (b) that until then a stranger can read each order's number, time, status, shoe, size, pairs, total, payment, paid, delivery, carrier, tracking number, expected date, customer name and city, and (c) which handler and which column allow-list leave the private and bookkeeping fields out of the query, for each of the ten collections.

### 6.5 The frontend reads the API (DATA-005)

The hook criteria are proved with the API stubbed at `fetch`; SC-26 and SC-37 are proved on the running site.

- **SC-26 [DATA-005 AC-1]:** On an empty cache, the built Home page requests `/api/v1/products?…status=live…`, `/api/v1/categories…` and `/api/v1/lines…`; no request ends in `.json`, and the built bundle holds none of the deleted files' records (a search of `dist/` for `Samba OG Black Crystal` finds nothing).
- **SC-27 [DATA-005 AC-2, AC-3, AC-4, item 4]:** `useCollection("products", { where: { status: "live" }, pageSize: 12 })` sends one request carrying `page=1&limit=12&status=live`; `items` holds the 12 returned and `total` is the API's `total`. Against 30 live shoes, one `loadMore()` sends `page=2`, giving 24 items and `hasMore` true; calling it until the last page arrives gives 30 items, `hasMore` false and no further request.
- **SC-28 [DATA-005 AC-5, AC-6, AC-7, item 3]:** `where: { category: ["women","kids"] }` sends `category=women,kids`; `{ price: { from: 3000000, to: 3999999 } }` sends `price_from=3000000&price_to=3999999` (a range with one end sends only that end); `{ colour: [], status: "" }` sends neither; `q` and `sort` are sent as given; `pageSize` is sent as `limit`.
- **SC-29 [DATA-005 AC-8, AC-9, item 6]:** `loading` is true from the moment a request starts until it ends, whether it succeeds or fails. When a request fails (no connection, or a status that is not 2xx after the one retry of SC-33) `error` is an `Error`, `loading` is false and the items already loaded are kept (page 1 loaded, page 2 answers 500: 12 items stay); `error` is null otherwise.
- **SC-30 [DATA-005 AC-10, item 5]:** When the name or the query changes, the old items are dropped and the first page of the new query is asked for; if the older query's answer arrives last, `items` still shows the newer query's records.
- **SC-31 [DATA-005 AC-11, item 7]:** `useRecord("products","nope")` has `loading` true until the answer; on a 404 it gives `item` undefined, `loading` false, `error` null; on a 200 it gives the record; on a 500 it gives an `error` and `loading` false; changing the id drops the old record.
- **SC-32 [DATA-005 AC-12, item 4, D-3]:** When 500 was asked for and the answer says `limit` 100 with `total` 130, `pages` is 2 and `total` is 130, never `items.length`. With `page: 3` in the query, `items` is that one page alone and `hasMore` is false. If the page asked for is past the last (the answer is `items: []` with `total` above 0), the hook asks once for the last page and `page` is that number.
- **SC-33 [DATA-005 AC-15, item 2, D-4]:** With `VITE_API_URL` set to `https://api.example.test`, a collection read goes to `https://api.example.test/v1/<collection>`; unset, it goes to `/api/v1/<collection>`. Requests are same-origin so a cookie travels with them. A non-2xx answer becomes an `ApiError` carrying the envelope's `error` code and `message`. A request unanswered after 15 seconds is aborted and counts as a failure. A failed `GET` (no connection, or a 5xx answer) is tried once more before `error` is set; a 404 is never retried.
- **SC-34 [DATA-005 AC-13, item 9]:** After the change `src/data/` holds only `index.ts` and `schema.json` (unchanged by this story); the change set adds `src/lib/api.ts`, deletes the other ten JSON files and edits no component, page, style, `content.json` or `package.json`; a search of `src/` finds no import of a deleted file.
- **SC-35 [DATA-005 AC-14, items 1, 8]:** `useCollection`, `useRecord`, `schema`, `setCollection`, `collections` and the types `Source`, `Row`, `Field`, `Collection`, `Filter`, `Query` are still exported with the signatures they have today; `useCollection` returns `{ items, total, pages, page, hasMore, loading, error, loadMore }`; `schema` is `schema.json`; `npm run lint`, the type-check, `npm run build` and `npm test` all pass.
- **SC-36 [DATA-005 item 1]:** The Studio's and `/admin`'s in-browser edits keep working: after a collection has been read, `collections()` holds the rows loaded so far for it; after `setCollection("products", rows)`, `collections().products` is `rows`, a mounted `useCollection("products")` redraws with them, and no request is sent. Reloading the page brings back what the database holds, because nothing is saved in this story.
- **SC-37 [DATA-005 purpose, §5]:** On the running site with the seeded database, the pages draw the records the design drew before the move: Home opens on `Samba OG Black Crystal` at 3.290.000 ₫; Shop lists the 12 shoes and its category links show Women, Men and Kids; the shoe page of `gazelle-indoor-solar-gold` shows its name, price, nine listed sizes with `40` sold out, and its three newest reviews; the header lists the five lines; the size chart has 17 rows; Returns, Shipping, Size chart and Track show their questions; `/admin/orders` lists the 14 orders. Differences are only the interim ones of D-5.

### 6.6 Speed (SRS §6)

- **SC-38 [DATA-002, NFR Performance]:** With 1,000 shoes seeded, `GET /v1/products?page=1&limit=12` answers within 300 ms at the 95th percentile over 200 sequential requests against the running API on the project's own host.

## 7. Dependencies

- **"Free delivery on every order, wording to match" must be merged first.** The seed copies `delivery.json`, `shop.json`, `orders.json` and `schema.json` as they stand. If, when this story starts, a non-zero fee, an `express` row, `free_delivery_from`, `express_*`, `exchange_days` or `refund_days` is still in a file, seed `standard` with fee 0, leave the rest out and re-price nothing else (DATA-001 item 3). If the three express orders still carry their old totals (5940000, 3340000, 3140000), stop and tell PM: the migration's own assertion (each total is its shoe's price times its pairs) will fail loudly, and the totals must not be edited silently.
- **Technical design** (`erd.md` §3, §8 migrations 1–3; `services.md` §2.3–2.4) is written; the technical-design step of this story only confirms it.
- **No external account and no secret.** The backend reads `DATABASE_URL`, which compose or `run_project` supplies. The edge proxy strips `/api`, so the browser-to-API check (SC-26, SC-37) runs through `run_project`, not plain `docker compose up` (overview §9).
- **Frontend lint** caps a function at complexity 20 and a file at 300 lines: build the query string in small functions, not one chain of guards.
- **Stories waiting on this one:** "Search, filters and sorting served by the API", "Checkout and placing an order", "Dealer sign-in and protected admin area", and "Product page waits for the shoe before saying it is missing".

## 8. Notes for Dev and TL

- Table, column and endpoint design is already in `erd.md` and `services.md`; the field-name mapping is `erd.md` §2.1 (`_source`→`source`, `line`→`line_id`, `payment`→`payment_method_id`, an order's `product`/`size`/`qty` from its first `order_items` row). Collection and column names that reach SQL come from an allow-list built from `schema.json`, never from the request.
- A text field the file leaves empty is `""` in the answer and a number the file leaves out is `null` (ERD §2.1); `released`, `date` and `eta` are `YYYY-MM-DD`.
- `ProductCard` reads `useRecord("lines", …)` once per card, so twelve cards ask for the same line. Sharing one in-flight request between identical simultaneous reads is allowed and welcome but no criterion requires it.
- The migration may only be written after the JSON it copies is final; do not hard-code counts that differ from the files — the counts in SC-1 are the ones at the time of writing.

## 9. Traceability

| Requirement | Criteria |
|---|---|
| DATA-001 | SC-1, SC-2, SC-3, SC-4, SC-5, SC-6, SC-7, SC-8, SC-9 |
| DATA-002 | SC-6, SC-10, SC-11, SC-12, SC-13, SC-14, SC-15, SC-16, SC-17, SC-38 |
| DATA-003 | SC-17, SC-18, SC-19 |
| DATA-004 | SC-16, SC-20, SC-21, SC-22, SC-23, SC-24, SC-25 |
| DATA-005 | SC-26, SC-27, SC-28, SC-29, SC-30, SC-31, SC-32, SC-33, SC-34, SC-35, SC-36, SC-37 |

Every DATA-001…005 acceptance criterion appears in one of the 38 criteria above; none was moved to the out-of-scope list.
