# Test cases — Site data from the database

Module: `data` · Plan item 3 · Requirements: DATA-001 … DATA-005 (`docs/data/SRS.md`) · Story: `docs/data/stories/site-data-from-the-database.md` (SC-1 … SC-38)
Contract: `docs/architecture/services.md` §2.1, §2.3 (error catalog), §2.4 (lists), §2.5 (path ids), §3.2, §3.3.

## 0. How to read this file

**Risk level: high.** The story moves the owner's whole catalogue and sample orders into a database in one seed (a lost or re-priced record cannot be seen again once the JSON is deleted), and it is the one place where an order's phone, email, address and note must be kept off the wire. So every seeded value that a criterion names is asserted by value, every private field is asserted absent from every response shape (list, record, filtered, paged), and the failure paths the SRS names (bad paging, private filter, unknown id, database unreachable, seed conflict) each have their own case.

**Conventions**

- Requests are written the way the browser sends them, `GET /api/v1/…`. The edge proxy strips `/api`; the backend serves `/v1/…` (services.md §1). Every API case is checked through the running story's `/api` path with no cookie and no `Authorization` header unless a case says otherwise.
- "Public view" is the answer to a caller with no session. This story has no session at all (sign-in is a later story), so every caller is a visitor; the SRS rows for the dealer ("same as a visitor in this module", "sessions do not exist yet") are covered by the visitor cases and need no case of their own.
- **State of the seed assumed by every case:** the story "Free delivery on every order, wording to match" has merged, so `delivery` holds only `standard` (fee 0), `shop` has no `free_delivery_from`, `express_days`, `express_cities`, `exchange_days` or `refund_days`, and the three former express orders carry `doi-24815` 5890000, `doi-24810` 3290000, `doi-24806` 3090000. Counts: categories 3, delivery 1, faqs 12, lines 5, orders 14, payments 4, products 12, reviews 36, shop 1, sizes 17. If the files held anything else when the story started, the file as it stood wins (SRS DATA-001 item 3) and the verifier says so.
- An error body is `{"error": <code>, "message": <sentence>, "errors": {<name>: <sentence>}}`; `errors` is present only on 400 (services.md §2.3). A case that says "the error envelope" means: the status, `error` equal to the named code, `message` a non-empty sentence, `errors` present exactly when the status is 400, and a `Content-Type` that starts with `application/json`.
- A case that arranges database state (`set_db_state`, `set_db_available`, `set_backend_state`) puts back what it changed before the next case; the **Then** of each says what to confirm after the restore.
- The key sets named for each collection are the public view: the fields of `schema.json` for that collection, plus `id`, minus `phone`, `email`, `address`, `note` (orders), `price_listed` (products) and every `_source` / `_invented`.

**Instrument split.** `fetch_url` for every response; `render_url` for what the pages draw and which requests the browser sends; `interact_page` for the admin edit and the shop filter; `measure_styles` is not needed (the story draws nothing new, SRS §5). A case marked `manual` is one no instrument can observe: SQL objects and the SQL a handler sends, text of a pull request, the repository's file list, the build and the unit tests against a stubbed `fetch`, and a 200-request percentile. Each says what the verifier reads instead (`get_pr`, `run_tests`) and the reason. 15 of 67 cases are `manual`.

**Ambiguities found (reported, not resolved here)**

1. `services.md` §2.4 "What a caller sees" describes the *final* public view (drafts and hidden reviews absent, orders not public). This story deliberately serves orders publicly and filters nothing by status (SRS DATA-004 item 4; story §3). The cases follow the story. No case asserts anything about drafts or hidden reviews, because none exist in the seed.
2. A filter on a private field name on a collection that does not have that field (`/api/v1/products?phone=x`): DATA-002 item 8 says ignore it, services.md §2.4 says a private filter is 400 but names `(orders)`. No case asserts either; TL to confirm.
3. The seed `eu-40-2-3` has `cm` 25.0 in the file; JSON gives it as the number `25` or `25.0` — a case compares numerically, not by text.

---

## 1. Storage and seed (DATA-001; SC-1 … SC-9)

**Scenario**: TC-01 Every collection holds exactly as many records as its file
**Given**: The migrations have run on an empty database and the server is up; no change has been made since.
**When**: `GET /api/v1/<collection>?limit=100` is sent for each of the ten collections.
**Then**: Each answers 200 and `total` and `items.length` are, in order: `categories` 3 (`women`, `men`, `kids`), `delivery` 1 (`standard`), `faqs` 12, `lines` 5 (`samba`, `handball-spezial`, `gazelle`, `adizero`, `megaride`), `orders` 14, `payments` 4 (`cod`, `bank`, `vnpay`, `momo`), `products` 12, `reviews` 36 (`r-01` … `r-36`), `shop` 1 (`shop`), `sizes` 17. Within each collection every `id` appears once.
Traces: SC-1 (DATA-001 AC-1)
Check: fetch_url

**Scenario**: TC-02 The shoe `samba-og-w-core-black` reads back with every value and type of its file
**Given**: The seeded database.
**When**: `GET /api/v1/products/samba-og-w-core-black` is sent.
**Then**: 200 and the record equals, field for field: `id` `samba-og-w-core-black`, `name` `Samba OG Black Crystal`, `model` `Samba OG`, `line` `samba`, `category` `women`, `price` the JSON number `3290000` (not a string, no fraction), `status` `live`, `featured` `true`, `is_new` `true`, `code` `IH9052`, `colourway` `Core Black / Core Black / Cloud White`, `colour` `black`, `swatch` `black`, `upper` `Leather set with small crystals, suede T-toe`, `lining` `Synthetic leather`, `sole` `Gum rubber cupsole`, `fit` `Narrow; take your usual size`, `released` `2026-11-03`, `stock` `18` (number), `rating` `4.8` (number), `review_count` `23` (number), `image` `/images/ref/samba-og-w-core-black.jpg`, `image_back` `/images/ref/samba-og-w-core-black-3.jpg`, `image_detail` `/images/ref/samba-og-w-core-black-2.jpg` (the `-3` and `-2` are not swapped), `description` beginning `The 1950s indoor football shoe with its black upper covered in small crystals` and `care` beginning `Brush the crystals with a soft dry brush`; `sizes` is exactly `["36","36 2/3","37 1/3","38","38 2/3","39 1/3","40","40 2/3","41 1/3"]`, `sizes_low` exactly `["37 1/3","40"]`, `sizes_out` exactly `["41 1/3"]` — each an array of strings in that order.
Traces: SC-2 (DATA-001 AC-2)
Check: fetch_url

**Scenario**: TC-03 A decimal size-chart value stays a number
**Given**: The seeded database.
**When**: `GET /api/v1/sizes/eu-35-1-2` and `GET /api/v1/sizes/eu-40-2-3` are sent.
**Then**: 200 each. `eu-35-1-2` has `eu` `35 1/2`, `uk` `3`, `us_m` `3.5`, `us_w` `4.5` (these four are strings) and `cm` equal to the JSON number `21.6` (typeof number, not the string `"21.6"`, not `21.60` text). `eu-40-2-3` has `cm` numerically equal to `25` (JSON `25` or `25.0`).
Traces: SC-2 (DATA-001 AC-3)
Check: fetch_url

**Scenario**: TC-04 An empty list is stored and returned as `[]`, never null or absent
**Given**: The seeded database. The file gives `sizes_out` as an empty list for `samba-og-w-crystal-white`, `adizero-evo-sl-charcoal`, `megaride-bones-halo-green` and `samba-og-kids-wonder-mauve`.
**When**: `GET /api/v1/products/samba-og-w-crystal-white` is sent, then `GET /api/v1/products?limit=100` and the four shoes are read from the list.
**Then**: For all four, `sizes_out` is present and `=== []` (an empty JSON array), in the record and in the list copy. `samba-og-w-crystal-white` still has `sizes_low` `["36","39 1/3"]` and `sizes` of nine entries.
Traces: SC-2 (DATA-001 boundary)
Check: fetch_url

**Scenario**: TC-05 A text field the file leaves empty is returned as an empty string
**Given**: The seeded database. The file leaves `carrier` and `tracking` empty on `doi-24816` and `eta` empty on `doi-24811`.
**When**: `GET /api/v1/orders/doi-24816` and `GET /api/v1/orders/doi-24811` are sent.
**Then**: `doi-24816` has `carrier` `""` and `tracking` `""`; `doi-24811` has `eta` `""`. Each key is present with the empty string — not `null`, not missing.
Traces: SC-2 (service contract §2.1: "a text field is `""` when unset")
Check: fetch_url

**Scenario**: TC-06 The sample order `doi-24817` reads back with its public values and its Vietnam-time stamp
**Given**: The seeded database.
**When**: `GET /api/v1/orders/doi-24817` is sent with no cookie.
**Then**: 200 and `number` `DOI-24817`, `placed_at` exactly `2026-11-23T16:05:00+07:00`, `status` `shipped`, `product` `samba-og-w-crystal-white`, `size` `39 1/3`, `qty` `1` (number), `total` `3090000` (number), `payment` `momo`, `paid` `true` (boolean), `delivery` `standard`, `carrier` `GHN`, `tracking` `GHN-PT27XM`, `eta` `2026-11-26`, `customer` `Hoàng Nam`, `city` `Đà Nẵng`.
Traces: SC-3 (DATA-001 AC-4, public fields)
Check: fetch_url

**Scenario**: TC-07 The private values of `doi-24817` are in the database as the file has them
**Given**: The seeded database, read by the database owner (SQL), not through the API.
**When**: The order's row and its first `order_items` row are read with a `SELECT` (through `set_db_state` where the verifier holds it, otherwise the backend PR's seed test).
**Then**: `phone` `0938 410 552`, `email` empty string, `address` `42 Example Street, Hải Châu`, `note` `Gift — no price inside the box.`, plus the public values of TC-06 with `placed_at` equal to `2026-11-23T16:05:00+07:00` as a timestamp with time zone.
Traces: SC-3 (DATA-001 AC-4, private fields)
Check: manual — a read of the private columns is available over no HTTP route by design (DATA-004); the instrument needed is a SQL `SELECT` whose rows are returned to the verifier, or the seed test in the backend PR (`run_tests`, backend).

**Scenario**: TC-08 Every reference in the seeded data names an existing record
**Given**: The seeded database. The lists of TC-01 are read with `limit=100`.
**When**: The ids of `lines`, `categories`, `products`, `payments` and `delivery` are collected and every reference is checked against them.
**Then**: Every shoe's `line` is one of the five line ids and its `category` one of `women`, `men`, `kids`; every review's `product` is a shoe id (all 36); every order's `product` is a shoe id, its `payment` one of `cod`, `bank`, `vnpay`, `momo`, its `delivery` is `standard`, which is the only delivery id. Zero references are `""`, `null` or unmatched.
Traces: SC-4 (DATA-001 AC-5)
Check: fetch_url

**Scenario**: TC-09 The database itself refuses a dangling reference
**Given**: The seeded database. Through `set_db_state`, an `INSERT` of a shoe with id `zz-dangling-line` and `line` `nope` is attempted; then an `INSERT` of a review with id `zz-dangling-review` and `product` `nope`; then an `INSERT` of an order with `payment` `nope`.
**When**: Each statement runs; then `GET /api/v1/products/zz-dangling-line`, `GET /api/v1/reviews/zz-dangling-review`, `GET /api/v1/orders/zz-dangling-order` and the three list totals are read.
**Then**: Each `INSERT` fails with a foreign-key violation (the refusal comes from the database, not from the handler). The three record reads are 404 `not_found`; `total` stays 12 for products, 36 for reviews, 14 for orders. Nothing is left to clean up.
Traces: SC-4 (DATA-001 AC-5, item 2)
Check: fetch_url

**Scenario**: TC-10 A changed record and its collection's count survive a restart
**Given**: The seeded database. Through `set_db_state`: `price` of `samba-og-w-core-black` set to `3300000` and `status` of review `r-01` set to `hidden`.
**When**: The backend is stopped and started again (`set_backend_state`), which runs the migrations again; then `GET /api/v1/products/samba-og-w-core-black`, `GET /api/v1/reviews/r-01` and the ten lists are read.
**Then**: The shoe's `price` is `3300000` and review `r-01` has `status` `hidden` (the restart did not re-seed them). Every collection's `total` is the number of TC-01 (12 products, 36 reviews, …) and no id appears twice in any list. After the check the two values are put back to `3290000` and `live`.
Traces: SC-5 (DATA-001 AC-6, item 5)
Check: fetch_url

**Scenario**: TC-11 A seed that meets an existing id fails loudly and leaves the row untouched
**Given**: The seeded database. Through `set_db_state`: the `name` of line `samba` set to `Samba (changed)`, and the row of the seed migration removed from `schema_migrations` so the next start runs the seed again against rows that already exist.
**When**: The backend is stopped and started again.
**Then**: The start fails: the backend does not come up (the gateway answers 502 on `/api/v1/lines`, `/healthz` is not 200), and its log names the id `samba` (read where the verifier can see the log). Nothing is merged: after the row is put back in `schema_migrations` and the backend started once more, `GET /api/v1/lines/samba` has `name` `Samba (changed)` and `GET /api/v1/lines` `total` is 5. Finally the name is put back to `Samba`.
Traces: SC-5 (DATA-001 conflict)
Check: fetch_url

**Scenario**: TC-12 Every collection lists in the order of its file
**Given**: The seeded database.
**When**: `GET /api/v1/<collection>?limit=100` is sent for each of the ten collections.
**Then**: `products` ids in order: `samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `handball-spezial-w-earth-strata`, `gazelle-indoor-solar-gold`, `adizero-adios-wonder-blue`, `adizero-evo-sl-charcoal`, `megaride-s2-cloud-white`, `megaride-bones-halo-green`, `megaride-ag-xl-core-black`, `samba-og-kids-wonder-mauve`, `samba-og-kids-cloud-white` (first `samba-og-w-core-black`, last `samba-og-kids-cloud-white`). `orders`: `doi-24817`, `doi-24816`, … down to `doi-24804` (file order, which is not sorted). `reviews`: `r-01` … `r-36`. `faqs` begins `f-delivery-time`, `f-delivery-where` and ends `f-orders-number`. `sizes` begins `eu-35-1-2` and ends `eu-46`. `payments`: `cod`, `bank`, `vnpay`, `momo`. `lines`: `samba`, `handball-spezial`, `gazelle`, `adizero`, `megaride`. `categories`: `women`, `men`, `kids`.
Traces: SC-6 (DATA-001 AC-7, DATA-002 item 5)
Check: fetch_url

**Scenario**: TC-13 Two identical requests return the same items in the same order
**Given**: The seeded database; no change between the requests.
**When**: `GET /api/v1/products?page=2&limit=5` is sent twice, and `GET /api/v1/orders?limit=7` twice.
**Then**: Within each pair the two bodies are identical (same `items` in the same order, same `total`, `page`, `limit`).
Traces: SC-6 (DATA-002 AC-10)
Check: fetch_url

**Scenario**: TC-14 A record added after the seed comes after the seeded ones
**Given**: The seeded database. Through `set_db_state` a valid shoe `zz-added-last` (line `samba`, category `women`, price 1000000, status `live`) is inserted.
**When**: `GET /api/v1/products?limit=100` and `GET /api/v1/products?page=2&limit=12` are sent.
**Then**: `total` is 13; the 13th item of the first answer is `zz-added-last` and the first twelve are the ids of TC-12 in the same order; the second answer holds exactly `zz-added-last`. The shoe is deleted afterwards and `total` is 12 again.
Traces: SC-6 (DATA-002 item 5)
Check: fetch_url

**Scenario**: TC-15 The shop's terms have no delivery-threshold or return columns
**Given**: The seeded database.
**When**: `GET /api/v1/shop/shop` and `GET /api/v1/shop` are sent.
**Then**: Both 200 (the list has `total` 1). The record has `id` `shop`, `standard_days` `2–4` (en dash), `dispatch_cutoff` `15:00`, `reply_time` `the same working day`, `rating` `4.8` and `review_count` `181` (numbers). None of the keys `free_delivery_from`, `express_days`, `express_cities`, `exchange_days`, `refund_days` is present, and the list copy equals the record copy.
Traces: SC-7 (DATA-001 AC-8; DATA-003 AC-2)
Check: fetch_url

**Scenario**: TC-16 Delivery holds one free option and no express row
**Given**: The seeded database.
**When**: `GET /api/v1/delivery`, `GET /api/v1/delivery/standard` and `GET /api/v1/delivery/express` are sent.
**Then**: The list is 200 with `total` 1 and one item whose `id` is `standard`; the record answer has `fee` equal to the JSON number `0` (not `null`, not `"0"`), a non-empty `name` and `days`; `express` answers 404 `not_found`.
Traces: SC-7 (DATA-001 AC-9), SC-18 (DATA-003 AC-5)
Check: fetch_url

**Scenario**: TC-17 The seed migration contains no express row, no fee and none of the five dropped columns
**Given**: The backend pull request's seed migration (`code/backend/migrations/*_seed*.up.sql`) and the table-creating migration.
**When**: Both files are read.
**Then**: Neither creates or inserts `free_delivery_from`, `express_days`, `express_cities`, `exchange_days` or `refund_days`; the `delivery_options` insert holds one row, `standard`, with fee `0`; no insert has an `express` id; the order inserts' totals are the values of TC-18; every insert is a plain `INSERT` (no `ON CONFLICT DO NOTHING` or `DO UPDATE` that would merge silently).
Traces: SC-7 (DATA-001 item 3; story §6.1 last sentence), SC-5 (conflict)
Check: manual — the content of a migration file is not observable on the running API; the verifier reads the diff with `get_pr`.

**Scenario**: TC-18 Every sample order is standard delivery, free, and totals the price of its pairs
**Given**: The seeded database. `GET /api/v1/orders?limit=100` and `GET /api/v1/products?limit=100` are read.
**When**: For each of the 14 orders, the shoe named by `product` is looked up and `price × qty` computed.
**Then**: Every order has `delivery` `standard` and `total` equal to its shoe's `price × qty`. In particular `doi-24815` is `5890000`, `doi-24810` is `3290000`, `doi-24806` is `3090000`, and `doi-24812` (two pairs of a 2790000 shoe) is `5580000`; `doi-24817` stays `3090000`. No order has `delivery` `express`.
Traces: SC-8 (DATA-001 AC-10)
Check: fetch_url

**Scenario**: TC-19 The foundation objects exist and behave
**Given**: The migrated database, and again after the migrations are run a second time (a restart).
**When**: Through `set_db_state`, a `DO` block asserts: `doi_fold('Cửa Lò') = 'cua lo'`, `doi_fold('Đại') = 'dai'`, `doi_fold('100%') = '100%'`; the function `set_updated_at()` exists; the table `rate_limits` exists; and after `UPDATE` of one row in each of the ten collection tables, that row's `updated_at` is later than before.
**Then**: Each assertion holds on the first run and the second; the second start inserts no row and changes none of the three objects.
Traces: SC-9 (DATA-001 item 1; ERD §8 migration 1)
Check: manual — an SQL function's result is not returned by any route (no endpoint uses it yet); the instrument needed is a SQL statement whose result or error reaches the verifier (`set_db_state` with a `DO` block). If the verifier has it, it uses it; otherwise the backend PR's migration test is read with `run_tests`.

---

## 2. The paged list (DATA-002; SC-6, SC-10 … SC-17)

**Scenario**: TC-20 Every collection answers the same four-key shape with defaults 1 and 12
**Given**: The seeded database.
**When**: `GET /api/v1/<collection>` with no parameter is sent for each of the ten collections.
**Then**: Each answers 200 with `Content-Type` starting `application/json` and a body whose keys are exactly `items`, `total`, `page`, `limit` (no other key), `page` `1`, `limit` `12`, `items` an array and `total` a number. `products`: 12 items, `total` 12. `sizes`: 12 items, `total` 17. `reviews`: 12 items, `total` 36. `orders`: 12 items, `total` 14. `faqs`: 12 items, `total` 12. `categories` 3/3, `delivery` 1/1, `lines` 5/5, `payments` 4/4, `shop` 1/1.
Traces: SC-10 (DATA-002 AC-1, AC-12, item 1), contract (GET /v1/{collection})
Check: fetch_url

**Scenario**: TC-21 Every collection's items carry the JSON's field names and JSON types, and nothing internal
**Given**: The seeded database.
**When**: The first item of `GET /api/v1/<collection>` is read for each of the ten collections.
**Then**: The key set of the item is exactly — `categories`: `id`, `name`, `blurb`, `image`; `delivery`: `id`, `name`, `days`, `fee`, `note`; `faqs`: `id`, `topic`, `question`, `answer`; `lines`: `id`, `name`, `blurb`, `since`, `fit_note`; `orders`: `id`, `number`, `placed_at`, `status`, `product`, `size`, `qty`, `total`, `payment`, `paid`, `delivery`, `carrier`, `tracking`, `eta`, `customer`, `city`; `payments`: `id`, `name`, `note`, `online`, `icon`; `products`: `id`, `name`, `model`, `line`, `category`, `price`, `status`, `featured`, `is_new`, `description`, `code`, `colourway`, `colour`, `swatch`, `upper`, `lining`, `sole`, `fit`, `care`, `released`, `sizes`, `sizes_low`, `sizes_out`, `stock`, `rating`, `review_count`, `image`, `image_back`, `image_detail`; `reviews`: `id`, `product`, `name`, `rating`, `date`, `size_bought`, `fit`, `body`, `status`; `shop`: `id`, `standard_days`, `dispatch_cutoff`, `reply_time`, `rating`, `review_count`; `sizes`: `id`, `eu`, `uk`, `us_m`, `us_w`, `cm`. No item has `seq`, `created_at`, `updated_at`, `source`, `invented`, `line_id`, `category_id`, `product_id`, `payment_method_id`, `delivery_option_id`. Types: product `price` `3290000` is a number, `featured` is `true`/`false` (booleans), `sizes` is an array; order `total` and `qty` numbers, `paid` boolean; payment `online` boolean; line `since` number; review `rating` number; shop `rating` number.
Traces: SC-10 (DATA-002 AC-9, item 6); SC-2; contract (§2.1 field names)
Check: fetch_url

**Scenario**: TC-22 Paging walks the stored order and a page past the last is empty with the true total
**Given**: 12 seeded shoes.
**When**: `GET /api/v1/products?page=2&limit=5`, `?page=3&limit=5`, `?page=12&limit=1`, `?page=13&limit=1` and `?page=9&limit=5` are sent.
**Then**: Page 2 of 5 holds, in this order, `adizero-adios-wonder-blue`, `adizero-evo-sl-charcoal`, `megaride-s2-cloud-white`, `megaride-bones-halo-green`, `megaride-ag-xl-core-black`, with `total` 12, `page` 2, `limit` 5. Page 3 of 5 holds `samba-og-kids-wonder-mauve` and `samba-og-kids-cloud-white` (two items, `total` 12). `page=12&limit=1` holds exactly `samba-og-kids-cloud-white`. `page=13&limit=1` and `page=9&limit=5` are 200 with `items` equal to `[]`, `total` 12 and `page` 13 and 9 respectively (not 404, not 400).
Traces: SC-11 (DATA-002 AC-2, AC-4, item 3)
Check: fetch_url

**Scenario**: TC-23 A limit above 100 is served as 100 and the answer says so
**Given**: 12 seeded shoes.
**When**: `GET /api/v1/products?limit=500`, `?limit=101`, `?limit=100`, `?limit=1` are sent.
**Then**: `limit=500` answers 200 with `limit` `100` and 12 items; `limit=101` answers `limit` `100`; `limit=100` answers `limit` `100`; `limit=1` answers `limit` `1` with exactly 1 item (`samba-og-w-core-black`) and `total` 12. None is a 400.
Traces: SC-12 (DATA-002 AC-3, item 2)
Check: fetch_url

**Scenario**: TC-24 With 130 shoes the first page of `limit=500` holds exactly 100
**Given**: Through `set_db_state`, 118 more valid shoes (`zz-bulk-001` … `zz-bulk-118`, line `samba`, category `women`, status `live`) are inserted, so the table holds 130.
**When**: `GET /api/v1/products?limit=500` and `GET /api/v1/products?limit=500&page=2` are sent.
**Then**: The first answer has `limit` `100`, 100 items (the twelve seeded ones first, in TC-12 order), `total` 130, `page` 1. The second has 30 items, `total` 130, `page` 2, `limit` 100. The 118 shoes are deleted afterwards and `total` is 12 again.
Traces: SC-12 (DATA-002 AC-3, boundary `pageSize` above 100)
Check: fetch_url

**Scenario**: TC-25 Exact filters on shoes combine with AND and `total` counts every match
**Given**: 12 seeded shoes.
**When**: These are sent: `products?category=women`; `products?category=women&status=live`; `products?category=women&status=draft`; `products?category=men&is_new=true`; `products?featured=true`; `products?featured=false`; `products?line=samba`; `products?colour=black`; `products?price=3290000`; `products?stock=18`; `products?category=women&limit=3&page=2`; `products?name=Samba%20OG%20Black%20Crystal`.
**Then**: `category=women` → `total` 4, items `samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `handball-spezial-w-earth-strata` in that order, every item `category` `women`. Adding `status=live` → `total` 4; `status=draft` → 200, `items` `[]`, `total` 0. `category=men&is_new=true` → only `gazelle-indoor-solar-gold` (`total` 1). `featured=true` → only `samba-og-w-core-black`; `featured=false` → `total` 11. `line=samba` → `total` 5. `colour=black` → `samba-og-w-core-black` and `megaride-ag-xl-core-black` (2). `price=3290000` → `samba-og-w-core-black` and `handball-spezial-w-earth-strata` (2). `stock=18` → only `samba-og-w-core-black`. `category=women&limit=3&page=2` → 1 item (`handball-spezial-w-earth-strata`), `total` 4 (the number of matches, not the page's length), `page` 2, `limit` 3. `name=Samba OG Black Crystal` → `total` 1.
Traces: SC-13 (DATA-002 AC-5, AC-6, item 4)
Check: fetch_url

**Scenario**: TC-26 A filter value must equal the stored value exactly
**Given**: 12 seeded shoes.
**When**: `products?category=wom`, `products?category=Women`, `products?category=women%20`, `products?name=Samba`, `products?category=zzz` are sent.
**Then**: Each answers 200 with `items` `[]` and `total` 0 — a partial, differently-cased or padded value matches nothing, and a no-match is an empty list, not a 404 and not the unfiltered list.
Traces: SC-13 (DATA-002 item 4)
Check: fetch_url

**Scenario**: TC-27 Filters on the other collections, including Vietnamese text and booleans
**Given**: The seeded database.
**When**: These are sent: `faqs?topic=returns`; `faqs?topic=shipping`; `reviews?product=samba-og-w-core-black`; `reviews?fit=small`; `reviews?product=gazelle-indoor-solar-gold&status=live`; `orders?status=delivered`; `orders?paid=false`; `orders?payment=cod`; `orders?delivery=standard`; `orders?city=H%C3%A0%20N%E1%BB%99i`; `orders?customer=Ho%C3%A0ng%20Nam`; `payments?online=true`; `payments?online=false`; `sizes?eu=40`; `lines?since=1950`.
**Then**: `faqs?topic=returns` → exactly `f-returns-size` and `f-returns-wrong` (`total` 2); `faqs?topic=shipping` → 4. `reviews?product=samba-og-w-core-black` → 3 (`r-01`, `r-02`, `r-03`); `reviews?fit=small` → 4; the gazelle reviews with `status=live` → 3. `orders?status=delivered` → 7; `orders?paid=false` → 3 (`doi-24816`, `doi-24814`, `doi-24811`); `orders?payment=cod` → 5; `orders?delivery=standard` → 14; `orders?city=Hà Nội` → 4 (`doi-24815`, `doi-24813`, `doi-24806`, `doi-24804`); `orders?customer=Hoàng Nam` → only `doi-24817`. `payments?online=true` → `bank`, `vnpay`, `momo`; `online=false` → `cod`. `sizes?eu=40` → only `eu-40`. `lines?since=1950` → only `samba`. Every answer is 200 in the four-key shape.
Traces: SC-13 (DATA-002 AC-7, item 4)
Check: fetch_url

**Scenario**: TC-28 On a list field a record matches when its list contains the value
**Given**: 12 seeded shoes.
**When**: These are sent: `products?sizes_low=40`; `products?sizes_low=40%202%2F3`; `products?sizes_low=4`; `products?sizes_out=41%201%2F3`; `products?sizes=36`; `products?sizes_low=40&category=women`.
**Then**: `sizes_low=40` → `samba-og-w-core-black` and `adizero-evo-sl-charcoal` (`total` 2) and not `handball-spezial-w-earth-strata`, whose list holds `40 2/3` (a list element matches by equality, not by prefix). `sizes_low=40 2/3` → only `handball-spezial-w-earth-strata`. `sizes_low=4` → `total` 0 (no substring match). `sizes_out=41 1/3` → `samba-og-w-core-black` and `samba-og-w-wonder-quartz`. `sizes=36` → 6 shoes. `sizes_low=40&category=women` → only `samba-og-w-core-black`.
Traces: SC-13 (DATA-002 AC-8, item 4)
Check: fetch_url

**Scenario**: TC-29 Parameters this module does not implement are accepted and ignored
**Given**: 12 seeded shoes; `GET /api/v1/products` is the baseline (12 items, seed order, `total` 12).
**When**: These are sent: `products?zzz=1`; `products?q=samba`; `products?sort=-price`; `products?price_from=3000000&price_to=3999999`; `products?category=women,kids`; `products?size=40`; `products?category=women&q=samba&sort=price&zzz=1`; and `faqs?zzz=1`, `orders?zzz=1`, `sizes?zzz=1` against their own baselines.
**Then**: The first six answers are identical to the baseline (same items, same order, `total` 12, `page` 1, `limit` 12). The seventh equals `products?category=women` alone (4 items, seed order). The three other collections' answers equal their baselines. None is a 400.
Traces: SC-14 (DATA-002 AC-11, items 7–8)
Check: fetch_url

**Scenario**: TC-30 A page or a limit that is not a whole number of at least 1 is a 400 naming it
**Given**: The seeded database.
**When**: `GET /api/v1/products?page=0`, `?page=-1`, `?page=abc`, `?page=1.5`, `?limit=0`, `?limit=-1`, `?limit=abc`, `?limit=1.5` are sent.
**Then**: Each answers 400 with the error envelope: `error` `bad_request`, a non-empty `message`, and `errors.page` (first four) or `errors.limit` (last four) holding a non-empty sentence. No answer has an `items` key and none carries a record.
Traces: SC-15 (DATA-002 invalid input), contract (§2.3, §2.4)
Check: fetch_url

**Scenario**: TC-31 A filter on a private or bookkeeping field is a 400 and never an empty list
**Given**: The seeded orders, `doi-24817` among them.
**When**: These are sent: `orders?phone=0938%20410%20552`; `orders?email=x`; `orders?address=42%20Example%20Street%2C%20H%E1%BA%A3i%20Ch%C3%A2u`; `orders?note=Gift`; `orders?_source=assumed`; `orders?_invented=x`; `products?_source=x`; `products?_invented=description`; `products?price_listed=US%24130`; `lines?_source=x`; `categories?_invented=x`; and a mixed one, `orders?status=shipped&phone=0938%20410%20552`.
**Then**: Each answers 400 with the error envelope, `error` `bad_request`, and `errors.<the field's name>` (`phone`, `email`, `address`, `note`, `_source`, `_invented`, `price_listed`) holding a sentence. No answer has `items`, and none contains the text `0938 410 552`, `42 Example Street`, `Gift` or `Hoàng Nam` (the message names the field, never a value held by a customer). The mixed request is also 400 — a valid filter beside a private one does not rescue it.
Traces: SC-16 (DATA-002 permission, DATA-004 AC-3, items 1–2)
Check: fetch_url

**Scenario**: TC-32 Text sent as a filter value or a field name never reaches the SQL
**Given**: 12 seeded shoes.
**When**: These are sent: `products?category=women'%20OR%20'1'%3D'1`; `products?category=women%3B%20DROP%20TABLE%20products%3B--`; `products?category%22%3D1--=x`; then `GET /api/v1/products` again.
**Then**: The first two answer 200 with `items` `[]` and `total` 0 (the value is data, not SQL). The third names no field and is ignored: 200 and equal to the baseline `GET /api/v1/products`. None answers 500 and the final `GET /api/v1/products` still has `total` 12.
Traces: contract (§2.4: only an allow-listed column name is spliced into SQL), SC-13, SC-14
Check: fetch_url

**Scenario**: TC-33 With the database unreachable a list and a record answer 500 with a generic body
**Given**: The backend is up and `set_db_available` takes the database away.
**When**: `GET /api/v1/products`, `GET /api/v1/products?category=women`, `GET /api/v1/products/samba-og-w-core-black`, `GET /api/v1/orders` and `GET /api/v1/shop/shop` are sent.
**Then**: Each answers 500 (not 503, not 502, not 200 with an empty list) with `{"error":"internal","message":<a sentence>}` and no `errors` key. The body contains no record text (`Samba`, `Hoàng`), no SQL (`SELECT`, `FROM`), no `pgx`, `postgres`, `dial`, `connection refused`, no host name, port or `DATABASE_URL`, no stack trace or file path. After `set_db_available` gives the database back, `GET /api/v1/products` is 200 with `total` 12.
Traces: SC-17 (DATA-002 upstream failure, DATA-003 upstream failure), contract (§2.3 `internal`)
Check: fetch_url

**Scenario**: TC-34 A failed query is a 500 `internal`, not a leak and not an empty answer
**Given**: Through `set_db_state`, the table `faqs` is renamed to `faqs_gone` (the backend stays up and the database is reachable).
**When**: `GET /api/v1/faqs`, `GET /api/v1/faqs/f-returns-size` and, as a control, `GET /api/v1/products` are sent.
**Then**: The two `faqs` requests answer 500 with `{"error":"internal","message":<a sentence>}`, no `errors` key, and no SQL, table name (`faqs`, `faqs_gone`), driver text or stack trace in the body. The control answers 200 with `total` 12. After the table is renamed back, `GET /api/v1/faqs` is 200 with `total` 12.
Traces: SC-17 (DATA-002 upstream failure), contract (§2.3 `internal`)
Check: fetch_url

---

## 3. One record (DATA-003; SC-18, SC-19)

**Scenario**: TC-35 A shoe by id carries its name, price and lists, and equals its list copy
**Given**: The seeded database.
**When**: `GET /api/v1/products/gazelle-indoor-solar-gold` is sent, then `GET /api/v1/products?limit=100` and the same shoe is read from the list.
**Then**: 200 with `name` `Gazelle Indoor Solar Gold`, `price` `3590000`, `sizes` `["40","40 2/3","41 1/3","42","42 2/3","43 1/3","44","44 2/3","45 1/3"]`, `sizes_low` `["42","42 2/3"]`, `sizes_out` `["40"]`, `category` `men`, `line` `gazelle`. The record and the list item have the same keys and equal values for every field (deep equality).
Traces: SC-18 (DATA-003 AC-1, AC-4)
Check: fetch_url

**Scenario**: TC-36 For every collection a record equals its list copy field for field
**Given**: The seeded database and the ten full lists (`limit=100`).
**When**: For each collection, the record endpoint is read for its first and last list item, and for `doi-24817` (orders), `f-returns-size` (faqs), `r-13` (reviews), `vnpay` (payments), `women` (categories), `megaride` (lines), `eu-36-2-3` (sizes).
**Then**: Each answers 200 and is deeply equal to the list item with the same id (same keys, same values, same types, lists in the same order).
Traces: SC-18 (DATA-003 AC-4)
Check: fetch_url

**Scenario**: TC-37 An unknown id, an unknown collection and an id outside the allowed characters are 404
**Given**: The seeded database.
**When**: These are sent: `GET /api/v1/products/nope`; `GET /api/v1/nothing`; `GET /api/v1/nothing/x`; `GET /api/v1/products/Samba-OG`; `GET /api/v1/orders/DOI-24817`; `GET /api/v1/products/a%20b`; `GET /api/v1/products/a%2Fb`; `GET /api/v1/products/-x`; `GET /api/v1/products/%27%3B%20DROP%20TABLE%20products%3B--`; `GET /api/v1/lines/gazelle-indoor-solar-gold`.
**Then**: Each answers 404 with the error envelope — `error` `not_found`, a non-empty `message`, no `errors` key — and never 200 with an empty body, never 400, never 500. `GET /api/v1/products` still has `total` 12 afterwards.
Traces: SC-19 (DATA-003 AC-3, not found), contract (§2.3 `not_found`, §2.5 id pattern `^[a-z0-9][a-z0-9._-]*$`)
Check: fetch_url

**Scenario**: TC-38 Nothing writes a record in this story
**Given**: The seeded database.
**When**: `POST /api/v1/faqs` with body `{"question":"x"}`, `PUT /api/v1/delivery/standard` with `{"fee":50000}`, `DELETE /api/v1/shop/shop`, and `POST /api/v1/products` with `{"name":"x"}` are sent (JSON content type), then the lists are read again.
**Then**: The first three answer 404 `not_found` (a collection with no write route answers every write with `not_found`, services.md §3); the `POST` to `products` is not a 2xx. `faqs` still has `total` 12, `delivery/standard` still has `fee` 0, `shop/shop` is still 200, `products` still has `total` 12.
Traces: contract (§3: routes are literal; a collection with no write route answers `POST`/`PUT`/`DELETE` with `404 not_found`), SC-5 (nobody writes a record in this story)
Check: fetch_url

---

## 4. What a stranger may not read (DATA-004; SC-16, SC-20 … SC-25)

**Scenario**: TC-39 An order read with no session has no phone, email, address or note
**Given**: The seeded order `doi-24817` (its file holds an empty `email`); the request carries no cookie.
**When**: `GET /api/v1/orders/doi-24817` is sent.
**Then**: 200. The body has none of the keys `phone`, `email` (absent, although the file holds an empty one), `address`, `note`, `_source`, `_invented`; it has `customer` `Hoàng Nam` and `city` `Đà Nẵng`. The raw body text contains none of `0938 410 552`, `42 Example Street`, `Hải Châu`, `Gift`.
Traces: SC-20 (DATA-004 AC-1, item 1)
Check: fetch_url

**Scenario**: TC-40 No order in any list shape carries a private field
**Given**: The seeded orders; no cookie.
**When**: `GET /api/v1/orders` (default page), `GET /api/v1/orders?limit=100`, `GET /api/v1/orders?limit=5&page=1`, `?limit=5&page=2`, `?limit=5&page=3`, and a filtered `GET /api/v1/orders?status=shipped` are sent.
**Then**: Every one answers 200 and no item in any of them has the keys `phone`, `email`, `address`, `note`, `_source` or `_invented`. The `limit=100` answer has 14 items; the three pages of 5 hold 5, 5 and 4. Each item still has `customer` and `city`. No body contains a string matching `0[0-9]{3} [0-9]{3} [0-9]{3}` (a phone) or `Example (Street|Lane|Road|Avenue)` (an address) or `Customer cancelled by phone` (a note).
Traces: SC-20 (DATA-004 AC-2)
Check: fetch_url

**Scenario**: TC-41 Shoes, lines and every other collection carry no bookkeeping
**Given**: The seeded database; no cookie.
**When**: `GET /api/v1/products?limit=100`, each of the 12 shoes by id, `GET /api/v1/lines/samba`, and `GET /api/v1/<collection>?limit=100` for the other eight collections are read.
**Then**: No item of `products` and no `products/<id>` record has `_source`, `_invented` or `price_listed`; no body contains `featuresneakerboutique`, `US$130`, `5.769.000 ₫`. `lines/samba` has no `_source` or `_invented` and has `name` `Samba`, `blurb`, `since` `1950` and `fit_note`. No item of any of the ten collections has `_source` or `_invented`.
Traces: SC-21 (DATA-004 AC-4, AC-5, item 2)
Check: fetch_url

**Scenario**: TC-42 Layout-only fields are still returned
**Given**: The seeded database; no cookie.
**When**: `GET /api/v1/products?limit=100` and `GET /api/v1/payments` are read.
**Then**: Every shoe has a non-empty string `swatch`, `image`, `image_back` and `image_detail`. Every way to pay has a non-empty `icon` (`cash`, `qr`, `card`, `wallet` for `cod`, `bank`, `vnpay`, `momo`) and an `online` boolean (`false` for `cod`, `true` for the other three).
Traces: SC-22 (DATA-004 AC-6, AC-7, item 3)
Check: fetch_url

**Scenario**: TC-43 The questions are public
**Given**: The seeded database; no cookie.
**When**: `GET /api/v1/faqs` and `GET /api/v1/faqs/f-returns-size` are sent.
**Then**: Both answer 200 (not 401, not 403). The list has `total` 12; the record has `topic` `returns` and a non-empty `question` and `answer`.
Traces: SC-22 (DATA-004 AC-8, item 5)
Check: fetch_url

**Scenario**: TC-44 A sweep of every collection and every record finds no private text on the wire
**Given**: The deployed API; no cookie.
**When**: Each of the ten collections is listed page by page with `limit=100` until a page is empty, and every record found is read by id; all bodies are searched.
**Then**: No body contains any of the 14 sample orders' phone numbers (`0938 410 552`, `0977 302 118`, `0912 774 630`, `0905 661 207`, `0983 215 904`, `0934 880 312`, `0919 040 776`, `0908 552 671`, `0937 118 249`, `0915 330 482`, `0939 602 115`, `0904 771 390`, `0935 248 016`, `0903 118 274`), any address (`42 Example Street, Hải Châu` and the other thirteen), any note (`Gift — no price inside the box.`, `Customer cancelled by phone; ordered 43 1/3 instead in store.`), any source address (`featuresneakerboutique`, `size.ly`) or any listed price (`US$130`, `US$90`, `5.769.000 ₫`). No body has a key `_source`, `_invented`, `price_listed`, `phone`, `email` or `address`; the key `note` appears only in `delivery` and `payments` bodies (public fields there), never in an `orders` body.
Traces: SC-23 (DATA-004 AC-9, NFR Privacy)
Check: fetch_url

**Scenario**: TC-45 The SQL a no-session read sends names none of the hidden columns
**Given**: The backend pull request's tests, which capture the SQL a no-session list and record read sends to the database.
**When**: The capture for each of the ten collections (list and record) is read.
**Then**: For `orders` the column list names none of `phone`, `email`, `address`, `note`; for every collection it names neither `source` nor `invented`; for `products` it does not name `price_listed`. The columns are left out of the query, not removed from the answer afterwards (no `SELECT *` followed by a delete of keys).
Traces: SC-24 (DATA-004 item 1)
Check: manual — what SQL a handler sends is not observable over HTTP; the verifier reads the handlers and their test with `get_pr` and runs `run_tests` (backend). The instrument that would observe it is a query log the verifier can read.

**Scenario**: TC-46 The backend pull request states the interim and the handler that enforces it
**Given**: The backend pull request's description.
**When**: It is read.
**Then**: It says (a) that the public read of orders is temporary and ends when "Checkout and placing an order", "Track an order by number and phone" and "Dealer sign-in and protected admin area" make orders readable only with an order token or a dealer session; (b) that until then a stranger can read each order's number, time, status, shoe, size, pairs, total, payment, paid, delivery, carrier, tracking number, expected date, customer name and city; (c) which handler and which column allow-list leave the private and bookkeeping fields out of the query, for each of the ten collections.
Traces: SC-25 (DATA-004 item 4, SRS §7 note)
Check: manual — it is the text of a pull request, read with `get_pr`; no running instrument reads it.

---

## 5. The frontend reads the API (DATA-005; SC-26 … SC-37)

**Scenario**: TC-47 The Home page asks the API for its records and asks for no JSON file
**Given**: The built site on an empty cache (a fresh browser profile), the seeded database.
**When**: `/` is rendered and the requests the browser sent are listed.
**Then**: The browser sent `GET /api/v1/products` with `status=live`, `page=1` and `limit=500`; `GET /api/v1/categories` with `page=1` and `limit=12`; `GET /api/v1/lines` with `page=1` and `limit=24` (the header's own `lines` request carries `limit=20`). Every request to the API is on the page's own origin and path-prefixed `/api/v1/`. No request ends in `.json`, and none targets `/src/data/` or `/assets/*.json`. The API answers each with 200 (the `limit=500` answer says `limit` 100).
Traces: SC-26 (DATA-005 AC-1), SC-27 (page and limit sent), SC-28 (`pageSize` sent as `limit`, a `where` entry as `<field>=<value>`), SC-33 (same-origin base `/api`)
Check: render_url — if the instrument does not list the requests, this case is UNVERIFIED for the request shape and TC-58 and TC-59 (the PR's stubbed-`fetch` tests) carry it.

**Scenario**: TC-48 The built bundle holds none of the deleted files' records
**Given**: The built site; the page's script assets are found from the HTML of `/`.
**When**: `GET /` is read for its `<script src>` and `<link rel=modulepreload>` addresses, and each JavaScript asset is fetched.
**Then**: No asset contains `Samba OG Black Crystal`, `featuresneakerboutique`, `US$130`, `Cửa Lò`-style order text such as `Customer cancelled by phone`, or the five line ids as record objects (`"since":1950`).
Traces: SC-26 (DATA-005 AC-1; `dist/` search)
Check: fetch_url

**Scenario**: TC-49 Home draws the records the design drew
**Given**: The seeded database, the site on an empty cache.
**When**: `/` is rendered at 1440 × 790 and again at 390 × 844.
**Then**: The page opens on the featured shoe `Samba OG Black Crystal` with its price `3.290.000 ₫` (the site's vi-VN format; the space before `₫` may be a non-breaking space); the header lists the five lines `Samba`, `Handball Spezial`, `Gazelle`, `Adizero`, `Megaride` (reached through the menu control at 390 wide); the three categories `Women`, `Men`, `Kids` are shown; no raw `undefined`, `null`, `[object Object]` or `NaN` is in the visible text.
Traces: SC-37 (DATA-005 purpose, §5)
Check: render_url

**Scenario**: TC-50 The Shop page shows the twelve shoes, and one ticked box narrows to the women's four
**Given**: The seeded database; `/shop` with no filter.
**When**: `/shop` is opened and read; then the filter rail's `Women` option is ticked.
**Then**: With no filter, twelve shoe cards are shown, one link `/shop/<id>` for each of the twelve ids of TC-12, `Samba OG Black Crystal` first. After ticking `Women`, exactly four cards remain (`samba-og-w-core-black`, `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `handball-spezial-w-earth-strata`) and the request that produced them carried `category=women`. (The search box, sort order, price band, size filter and a two-box tick change nothing until the next story — story D-5 — and are not asserted.)
Traces: SC-37 (D-5: one ticked box works), SC-13
Check: interact_page

**Scenario**: TC-51 A shoe page draws the record, its sizes, its reviews and sends the query it should
**Given**: The seeded database.
**When**: `/shop/gazelle-indoor-solar-gold` is rendered and the requests are listed.
**Then**: The page shows the name `Gazelle Indoor Solar Gold` and the price `3.590.000 ₫`; nine sizes are listed (`40`, `40 2/3`, `41 1/3`, `42`, `42 2/3`, `43 1/3`, `44`, `44 2/3`, `45 1/3`) with `40` shown as sold out; three reviews are shown (`Quang`, `Tuấn`, `Hiếu`). The browser sent `GET /api/v1/products/gazelle-indoor-solar-gold` (200) and `GET /api/v1/reviews` with `product=gazelle-indoor-solar-gold`, `status=live`, `sort=-date`, `page=1`, `limit=3`.
Traces: SC-37 (shoe page), SC-28 (`q`/`sort` and `where` sent as given), SC-31 (record on 200)
Check: render_url

**Scenario**: TC-52 The size chart, Returns, Shipping and Track pages show their rows and questions
**Given**: The seeded database.
**When**: `/size-chart`, `/returns`, `/shipping` and `/track` are rendered.
**Then**: `/size-chart` shows 17 chart rows, from EU `35 1/2` to EU `46`, with `21.6` as the foot length of the first; and the three sizing questions `Do adidas shoes run small?`, `I am between two sizes.`, `How do I measure a child's foot?`. `/returns` shows `The size is wrong. What do I do?` and `My delivery is wrong or damaged.`. `/shipping` shows `How long does delivery take?`, `Do you deliver to every province?`, `Can I change the address after ordering?`, `Can you send a pair as a gift?` and the one delivery option `Standard`, with no `Express` option. `/track` shows the question `Where is my order number?`. No page shows a loading or error screen once its answers have arrived.
Traces: SC-37 (size chart 17 rows; Returns, Shipping, Size chart and Track show their questions), DATA-004 AC-8
Check: render_url

**Scenario**: TC-53 `/admin/orders` lists the 14 orders
**Given**: The seeded database; the dealer area is entered the way the design does today (open `/admin/sign-in`, enter any email address, submit — the local session of `src/lib/session.ts`).
**When**: `/admin/orders` is opened.
**Then**: 14 order rows are listed, `DOI-24817` among them with the customer `Hoàng Nam` and the total `3.090.000 ₫`; the rows' count text (if the page shows one) reads 14. (The dashboard's `sort=-placed_at` is ignored in this story: seed order is accepted, D-5.)
Traces: SC-37 (`/admin/orders` lists the 14 orders)
Check: interact_page

**Scenario**: TC-54 A shoe the API does not have ends in the missing-shoe state, with one request and no error
**Given**: The seeded database; no shoe `nope`.
**When**: `/shop/nope` is rendered and the requests are listed.
**Then**: `GET /api/v1/products/nope` was sent exactly once and answered 404 (a 404 is an answer and is not retried); once it has answered the page shows the heading `That pair is not in the shop any more.` and the `All shoes` pill linking to `/shop`; the page is not blank and shows no failure message and no uncaught JavaScript error. (The moment before the answer is the open flaw of story D-1 and is not asserted.)
Traces: SC-31 (DATA-005 AC-11: 404 gives item undefined, loading false, error null), SC-33 (a 404 is never retried)
Check: render_url

**Scenario**: TC-55 With the API unreachable a page shows its empty-data appearance and the failed read is retried once
**Given**: The backend is stopped with `set_backend_state` (the gateway answers 502).
**When**: `/` and `/shop` are rendered and the requests are listed.
**Then**: Each page renders without a blank screen and without an uncaught JavaScript error: the header and footer are drawn, no shoe card is drawn, and no error screen, message or button the approved design does not have appears. Each failed `GET` URL (for example `/api/v1/products?…`) was sent exactly twice — the original and one retry — and no third time. After the backend is started again and the page reloaded, `/` shows `Samba OG Black Crystal` again.
Traces: SC-29 (DATA-005 AC-9, upstream failure: nothing drawn that the design does not draw), SC-33 (one retry of a failed `GET`)
Check: render_url

**Scenario**: TC-56 A 500 from the API behaves the same as an unreachable one
**Given**: `set_db_available` takes the database away so the API answers 500 (TC-33).
**When**: `/shop` is rendered and the requests are listed.
**Then**: The page renders with its header and footer and no shoe cards, no blank screen, no uncaught JavaScript error and no error screen; each failed `GET` URL was sent exactly twice (a 5xx is retried once). After `set_db_available` gives the database back and the page is reloaded, twelve shoe cards are drawn.
Traces: SC-29 (DATA-005 AC-9), SC-33 (5xx retried once)
Check: render_url

**Scenario**: TC-57 The in-browser edits of `/admin` keep working and save nothing
**Given**: The seeded database; the dealer area entered as in TC-53; `/admin/products` opened so the shoes are loaded.
**When**: The first shoe (`Samba OG Black Crystal`) is opened from the list, its `Name` is changed to `Samba OG Edited`, and the form is saved; the list is read; then the page is reloaded and the list is read again.
**Then**: After saving, the list shows `Samba OG Edited`; no `POST`, `PUT`, `PATCH` or `DELETE` request was sent to `/api` during the edit; `GET /api/v1/products/samba-og-w-core-black` still has `name` `Samba OG Black Crystal`. After the reload the list shows `Samba OG Black Crystal` again (nothing is saved in this story).
Traces: SC-36 (DATA-005 item 1: `setCollection` and `collections()`)
Check: interact_page

**Scenario**: TC-58 The first page of `useCollection` and `loadMore` against 30 live shoes (stubbed `fetch`)
**Given**: The PR's hook tests, with `fetch` stubbed to serve 30 live shoes 12 at a time.
**When**: `useCollection("products", { where: { status: "live" }, pageSize: 12 })` mounts; `loadMore()` is called once; then called until the last page arrives; then once more.
**Then**: One request is sent on mount, carrying `page=1&limit=12&status=live`; `items` holds the 12 returned and `total` is the API's `30`. After one `loadMore()`: a request carrying `page=2`, `items` 24, `hasMore` `true`. After the last page: 30 items, `hasMore` `false`, and the extra call sends no request. `total` is never `items.length`.
Traces: SC-27 (DATA-005 AC-2, AC-3, AC-4, item 4)
Check: manual — it needs a `fetch` stub and a mounted hook, which only the PR's vitest suite provides; no page calls `loadMore()` until the catalogue story, so no running page exercises it. The verifier runs `run_tests` (frontend) and reads the named tests.

**Scenario**: TC-59 The query string `useCollection` builds (stubbed `fetch`)
**Given**: The PR's hook tests with `fetch` stubbed.
**When**: `useCollection` mounts with, in turn: `where: { category: ["women","kids"] }`; `where: { price: { from: 3000000, to: 3999999 } }`; `where: { price: { from: 3000000 } }`; `where: { price: { to: 3999999 } }`; `where: { colour: [], status: "" }`; `q: "samba"`, `sort: "-price"`; `pageSize: 5`.
**Then**: The requests carry, respectively: `category=women,kids`; `price_from=3000000&price_to=3999999`; only `price_from=3000000`; only `price_to=3999999`; neither `colour` nor `status` (and no empty `colour=` or `status=`); `q=samba` and `sort=-price` as given; `limit=5`.
Traces: SC-28 (DATA-005 AC-5, AC-6, AC-7, item 3)
Check: manual — a request URL built by a hook cannot be read from a running page for these inputs (the shop page does not send them until the catalogue story); the verifier reads the PR's stubbed-`fetch` tests with `run_tests` (frontend).

**Scenario**: TC-60 `loading` and `error` follow the request, and loaded items stay on a failure (stubbed `fetch`)
**Given**: The PR's hook tests: `fetch` held open; then answered 200; then page 1 answered 200 (12 items) and page 2 answered 500 twice (the original and the one retry).
**When**: The hook is read while the request is in flight, after success, and after the failed `loadMore()`; also with the first request failing with a network error.
**Then**: `loading` is `true` from the start of the request until it ends and `false` after, whether it succeeded or failed; `error` is `null` after success; after the page-2 failure `error` is an `Error` instance, `loading` is `false` and `items` still holds the 12 of page 1; a network failure gives `error` an `Error`, `loading` `false`.
Traces: SC-29 (DATA-005 AC-8, AC-9, item 6)
Check: manual — it needs a `fetch` that fails on cue (page 2 only) which no running site offers; the verifier reads the PR's tests with `run_tests` (frontend). The running-site half (nothing drawn, one retry) is TC-55 and TC-56.

**Scenario**: TC-61 The latest query wins when an older answer arrives last (stubbed `fetch`)
**Given**: The PR's hook tests: query A (`category=women`) answered slowly, query B (`category=men`) answered at once.
**When**: The hook mounts with A, the query changes to B before A answers, and A's answer then arrives; also the name changes from `products` to `lines` with an old answer in flight.
**Then**: When the query changes the old items are dropped at once and page 1 of the new query is asked for; after A's late answer, `items` still holds B's records (and only B's); after the name changes, `items` never holds a record of the old collection.
Traces: SC-30 (DATA-005 AC-10, item 5; concurrency)
Check: manual — it needs control of the order in which two answers arrive; only the PR's stubbed-`fetch` tests give that. The verifier reads them with `run_tests` (frontend).

**Scenario**: TC-62 `useRecord` on 200, 404, 500 and a changed id (stubbed `fetch`)
**Given**: The PR's hook tests: `fetch` answering 200 with a shoe, 404, then 500 (twice).
**When**: `useRecord("products", id)` is read for each answer, and then the id is changed to another.
**Then**: Before the answer `loading` is `true`; on 200 `item` is the record, `loading` `false`, `error` `null`; on 404 `item` is `undefined`, `loading` `false`, `error` `null`; on 500 (after one retry) `error` is an `Error`, `loading` `false`; changing the id drops the old record (`item` is `undefined` until the new answer).
Traces: SC-31 (DATA-005 AC-11, item 7)
Check: manual — the 500 and the id change need a stub; the 200 and 404 halves are observed on the running site in TC-51 and TC-54. The verifier reads the PR's tests with `run_tests` (frontend).

**Scenario**: TC-63 `pages`, `total`, a named `page` and a page past the last (stubbed `fetch`)
**Given**: The PR's hook tests: `fetch` answering `{items: […100], total: 130, page: 1, limit: 100}` to a `limit=500` request; an answer of `items: []` with `total` 130 for a page past the last.
**When**: `useCollection("products", { pageSize: 500 })` is read; then `useCollection("products", { page: 3, pageSize: 12 })`; then `useCollection("products", { page: 9, pageSize: 12 })` against the past-the-last answer.
**Then**: The first has `pages` `2` (it trusts the answer's `limit` 100, not the 500 asked) and `total` `130`; the second has `items` that one page alone and `hasMore` `false`; the third asks once more for the last page (`page=11` of 130 at 12) and its `page` is `11`, `items` that page, with no third request.
Traces: SC-32 (DATA-005 AC-12, item 4, D-3)
Check: manual — it needs an answer shaped by the test; only the PR's stubbed-`fetch` tests give it. The verifier reads them with `run_tests` (frontend).

**Scenario**: TC-64 `api.ts`: base address, same-origin, `ApiError`, 15-second timeout, one retry, no retry of a 404 (stubbed `fetch`)
**Given**: The PR's tests of `src/lib/api.ts`: `VITE_API_URL` set to `https://api.example.test`, then unset; `fetch` answering `{"error":"internal","message":"x"}` with 500, then 404, then never answering; fake timers.
**When**: A collection is read in each setting.
**Then**: With `VITE_API_URL` set the request goes to `https://api.example.test/v1/<collection>`; unset, to `/api/v1/<collection>`, same-origin (cookies are sent). A non-2xx answer becomes an `ApiError` carrying the envelope's `error` code and `message`. A request unanswered at 15 seconds is aborted and counts as a failure. A failed `GET` (a network failure or a 5xx) is sent a second time before `error` is set (two calls); a 404 is sent once.
Traces: SC-33 (DATA-005 AC-15, item 2, D-4)
Check: manual — it needs a build-time environment variable, a fake clock and a `fetch` that never answers, which no running instrument can arrange. The verifier reads the PR's tests with `run_tests` (frontend); the retry and the 404 are also observed on the running site in TC-54, TC-55 and TC-56.

**Scenario**: TC-65 The change set keeps `src/data/` to two files and touches no component, page or style
**Given**: The frontend pull request's file list and the repository at its head.
**When**: The list of changed files and the contents of `code/frontend/src/data/` are read; `src/` is searched for imports of a deleted file.
**Then**: `src/data/` holds only `index.ts` and `schema.json` (`schema.json` byte-for-byte unchanged); the change adds `src/lib/api.ts` and its tests, rewrites `src/data/index.ts`, deletes `categories.json`, `delivery.json`, `faqs.json`, `lines.json`, `orders.json`, `payments.json`, `products.json`, `reviews.json`, `shop.json`, `sizes.json`, and edits no file under `src/components/`, `src/pages/`, no `.css`, no `src/content.json`, no `src/App.tsx`, no `package.json`; no file in `src/` imports a deleted JSON file (no `import.meta.glob` of `./*.json` remains).
Traces: SC-34 (DATA-005 AC-13, item 9)
Check: manual — a file list is not observable on the running site; the verifier reads it with `get_pr` and `grep`.

**Scenario**: TC-66 The public names are unchanged and lint, type-check, build and tests pass
**Given**: The frontend pull request at its head.
**When**: `src/data/index.ts` is read for its exports, and `npm run lint`, the type-check, `npm run build` and `npm test` are run (`run_tests` frontend, and CI).
**Then**: `useCollection`, `useRecord`, `schema`, `setCollection`, `collections` and the types `Source`, `Row`, `Field`, `Collection`, `Filter`, `Query` are all exported with the signatures they have on `main`; `useCollection` returns `{ items, total, pages, page, hasMore, loading, error, loadMore }`; `schema` is `schema.json`; each of the four commands exits 0 (no lint error from the complexity cap of 20 or the 300-line cap).
Traces: SC-35 (DATA-005 AC-14, items 1, 8)
Check: manual — a build and a lint run are not observable on the running site; the verifier uses `run_tests` (frontend) and the CI logs.

---

## 6. Speed (NFR; SC-38)

**Scenario**: TC-67 `GET /v1/products?page=1&limit=12` is fast with 1,000 shoes
**Given**: 1,000 shoes seeded (through `set_db_state`, 988 extra valid live shoes) on the project's own host.
**When**: `GET /api/v1/products?page=1&limit=12` is sent 200 times in sequence and each duration recorded.
**Then**: The 95th percentile of the 200 durations is at most 300 ms; every answer is 200 with 12 items and `total` 1000. The extra shoes are deleted afterwards.
Traces: SC-38 (DATA-002 NFR Performance)
Check: manual — one `fetch_url` reports one response, not a percentile over 200 sequential requests; the instrument needed is a timed loop of 200 requests whose durations reach the verifier. The `total` 1000 and 12-item shape of a single answer are still read with `fetch_url`.

---

## 7. Coverage walk (story criteria → cases)

| Story criterion | Cases |
|---|---|
| SC-1 | TC-01 |
| SC-2 | TC-02, TC-03, TC-04, TC-05 |
| SC-3 | TC-06, TC-07 |
| SC-4 | TC-08, TC-09 |
| SC-5 | TC-10, TC-11, TC-17, TC-38 |
| SC-6 | TC-12, TC-13, TC-14 |
| SC-7 | TC-15, TC-16, TC-17 |
| SC-8 | TC-18 |
| SC-9 | TC-19 |
| SC-10 | TC-20, TC-21 |
| SC-11 | TC-22 |
| SC-12 | TC-23, TC-24 |
| SC-13 | TC-25, TC-26, TC-27, TC-28, TC-32 |
| SC-14 | TC-29, TC-32 |
| SC-15 | TC-30 |
| SC-16 | TC-31 |
| SC-17 | TC-33, TC-34 |
| SC-18 | TC-16, TC-35, TC-36 |
| SC-19 | TC-37 |
| SC-20 | TC-39, TC-40 |
| SC-21 | TC-41 |
| SC-22 | TC-42, TC-43 |
| SC-23 | TC-44 |
| SC-24 | TC-45 |
| SC-25 | TC-46 |
| SC-26 | TC-47, TC-48 |
| SC-27 | TC-47, TC-58 |
| SC-28 | TC-47, TC-51, TC-59 |
| SC-29 | TC-55, TC-56, TC-60 |
| SC-30 | TC-61 |
| SC-31 | TC-51, TC-54, TC-62 |
| SC-32 | TC-63 |
| SC-33 | TC-47, TC-54, TC-55, TC-56, TC-64 |
| SC-34 | TC-65 |
| SC-35 | TC-66 |
| SC-36 | TC-57 |
| SC-37 | TC-49, TC-50, TC-51, TC-52, TC-53 |
| SC-38 | TC-67 |

**Roles walked.** Visitor (no session): every case. Dealer: the SRS says it is the same as a visitor in this module and that dealer reads of private fields arrive with `admin` (DATA-004 "Permission" row, not applicable); the `/admin` pages of TC-53 and TC-57 are the dealer's screens and use the design's local session. Developer: TC-65, TC-66.

**Failure paths walked.** Bad `page` and `limit` (TC-30); private filter (TC-31); unknown id, collection and id characters (TC-37); database unreachable (TC-33); failed query (TC-34); seed conflict (TC-11); dangling reference (TC-09); API down and API 500 as the pages see them (TC-55, TC-56); 404 as the hook sees it (TC-54); write attempts (TC-38). Boundaries: empty list (TC-04), empty text (TC-05), `limit` 100/101/500 (TC-23, TC-24), page past the last (TC-22), exact value vs partial (TC-26, TC-28), 130 and 1,000 rows (TC-24, TC-67).
