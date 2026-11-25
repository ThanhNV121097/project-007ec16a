# SRS — Site data from the database

Module: `data`
Design: [View the approved design](http://localhost:9401)
Design system: `design/design-system.md`

> One file per module. This one covers moving the records the design shows from JSON files into the
> project's database and serving them through a paged read API, and the one frontend module that reads it.
> Search, filters and sorting are in `docs/catalogue/SRS.md`; creating and editing records is in
> `docs/admin/SRS.md`; placing orders is in `docs/orders/SRS.md`.

## 1. Purpose

The approved design reads its records — shoes, lines, categories, reviews, orders, shop terms, delivery
options, ways to pay, questions and the size chart — from JSON files bundled with the site. This module moves
them into the database, serves them over a paged read API, and rewires the one data module of the frontend to
call it, so that every page keeps looking exactly as designed while the records live on the server. Without it
the owner's records are lost on every rebuild and no later story (orders, admin, search) has anywhere to write.

## 2. Actors

| Actor | Who they are | What they may do in this module |
|---|---|---|
| Visitor | A caller with no session | Read every collection through the list and record endpoints, except the fields in DATA-004 |
| Dealer | The shop owner or staff | Same as a visitor in this module; the dealer's own reads of private fields arrive with the `admin` module |
| Developer | Builds the site | Rewrites the frontend data module; deletes the JSON files |

## 3. Scope

**In scope** — the plan item, by title:

- Site data from the database

**Out of scope**

- Free-text search, "any of" filters, ranges, sorting and the size filter — belong to the `catalogue` module.
- Creating, changing and deleting records, uploads, validation of what a dealer types — belong to the `admin` module.
- Order creation, order privacy by token, per-size stock counts — belong to the `orders` module.
- Signing in and who is a dealer — belongs to the `admin` module.
- Any change to a component or page — this module changes the data module only.

## 4. Functional requirements

### 4.1 Site data from the database

**Requirement DATA-001 — Every collection is stored and seeded exactly as the owner left it**

*As a* dealer, *I want* every record I edited in the Studio to be in the database, *so that* none of my records is lost, renamed or re-priced by the move.

Behaviour:

1. The database holds ten collections, each with its own table of one column per field the collection's description in `schema.json` lists: categories, delivery, faqs, lines, orders, payments, products, reviews, shop, sizes. The record's id is its key and is text.
2. Field types are kept as the schema names them: a price is a number in the currency the schema names (VND, no fractions added); a number stays a number; an image or url is text; a list is a list of text; a reference to another collection holds that record's id and must name an existing record; a true/false field is true/false. The record's source note is kept as its own column.
3. One migration creates the tables and a second inserts every record of every JSON file in `src/data/` as the files stand when this story starts (after the story "Free delivery on every order, wording to match"), exactly as they are: no record is dropped, no id, name, price, list or flag is changed, and each collection keeps the order its file had.
4. After seeding, every reference resolves: each shoe's line and category, each review's shoe and each order's shoe name an existing record.
5. Running the migrations again, or restarting the server, never inserts a record twice and never overwrites a change made after seeding.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The ten JSON files before the change | The migrations have run on an empty database | Each collection holds exactly as many records as its file did |
| AC-2 | The shoe `samba-og-w-core-black` (price 3290000, sizes `["36","36 2/3",…,"41 1/3"]`, last-pairs `["37 1/3","40"]`, sold-out `["41 1/3"]`, stock 18) | The record is read back | Every field equals the file's value, including the three lists in the same order |
| AC-3 | The size row `eu-35-1-2` with cm 21.6 | The record is read back | `cm` is the number `21.6` |
| AC-4 | The order `doi-24817` (total 3090000, paid true, eta `2026-11-26`) | The record is read back by the database owner | Every field equals the file's value, including the private ones |
| AC-5 | The seeded database | Every reference field is checked | Each names an existing record; none is empty or dangling |
| AC-6 | A record was changed after seeding | The server restarts and the migrations run again | The changed value is still there and the collection's count is unchanged |
| AC-7 | The seeded database | The shoes are listed in stored order | They come back in the order of the file: `samba-og-w-core-black` first |
| AC-8 | The record `shop` | It is read back | `free_delivery_from` is 2000000 and `exchange_days` is 14 |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Conflict | The migration meets an id that already exists | The migration fails loudly and leaves the existing record untouched; it never merges silently |
| Boundary | A list field is empty in the file | It is stored as an empty list and returned as `[]` |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Ten collections | read / write (migration) | yes | One column per schema field; id is the key; source note kept |

**Requirement DATA-002 — The paged list of a collection**

*As a* visitor, *I want to* ask for a collection a page at a time, optionally narrowed by one field's exact value, *so that* a page of the site never has to download a whole table.

Behaviour:

1. `GET /v1/<collection>` answers `{"items": [...], "total": n, "page": p, "limit": l}` for each of the ten collections. The browser reaches it as `/api/v1/<collection>`.
2. `page` starts at 1 and defaults to 1. `limit` defaults to 12 and is at most 100: a larger limit is served as 100 and the answer's `limit` says 100.
3. `total` is the number of records that match, whatever page is asked for. A page past the last returns `items: []` with the true total.
4. `<field>=<value>` for any field of the collection's schema keeps only records whose field equals the value exactly; several such parameters all apply together. For a list field the record matches when its list contains the value. A true/false field takes `true` or `false`.
5. Records come in a stable order: the order of the seed, then records in the order they were added.
6. Each record has the same field names the JSON used (`id`, `name`, `price`, `sizes_low`, …). Values are JSON numbers, strings, booleans and arrays — not text-wrapped numbers.
7. Parameters this module does not implement (`q`, `sort`, a `_from` or `_to` suffix, a comma list) are accepted and ignored here; the `catalogue` module gives them meaning.
8. A parameter that names no field of the collection and is not one of the reserved names is ignored.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | 12 shoes seeded | `GET /v1/products` | 200 with `items` of length 12, `total` 12, `page` 1, `limit` 12 |
| AC-2 | The same | `GET /v1/products?page=2&limit=5` | `items` holds shoes 6 to 10 in seed order, `total` 12, `page` 2, `limit` 5 |
| AC-3 | The same | `GET /v1/products?limit=500` | `items` has at most 100 entries and the answer's `limit` is 100 |
| AC-4 | The same | `GET /v1/products?page=9&limit=5` | `items` is `[]`, `total` is 12, `page` is 9 |
| AC-5 | 12 shoes, 4 in category `women` | `GET /v1/products?category=women` | `total` is 4 and every item has `category` `women` |
| AC-6 | The same | `GET /v1/products?category=women&status=live` | `total` counts shoes that are both |
| AC-7 | The same | `GET /v1/faqs?topic=returns` | Only questions of topic `returns` are returned |
| AC-8 | A shoe whose last-pairs list is `["37 1/3","40"]` | `GET /v1/products?sizes_low=40` | That shoe is among the items |
| AC-9 | Any collection | The first item is read | Its field names are those of the JSON file; `price` is a number such as `3290000`, `featured` is `true`/`false`, `sizes` is an array |
| AC-10 | Two requests for the same page | Both are sent without a change in between | The two answers list the same items in the same order |
| AC-11 | Any collection | `GET /v1/<collection>?zzz=1` | The answer is the same as without the parameter |
| AC-12 | The seeded database | `GET /v1/categories`, `/v1/delivery`, `/v1/lines`, `/v1/payments`, `/v1/shop`, `/v1/sizes`, `/v1/reviews`, `/v1/orders`, `/v1/faqs` | Each answers 200 in the same four-key shape with `total` equal to the file's record count |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | `page` is 0, negative or not a whole number | 400 with an error body naming `page`; nothing else is returned |
| Invalid input | `limit` is 0, negative or not a whole number | 400 naming `limit` |
| Not found | The collection name is not one of the ten | 404 |
| Permission | A filter names a private field (DATA-004) | 400 naming the field; nothing is returned (a filter on a private field would reveal what it hides) |
| Upstream failure | The database cannot be reached | 500 with a generic error body; no record data, SQL or stack trace in it (the error envelope is in the service contract) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Ten collections | read | yes | Public, unauthenticated reads, except DATA-004 |

**Requirement DATA-003 — One record by id**

*As a* visitor, *I want to* read one record by its id, *so that* a shoe page, a line or the shop's terms load directly.

Behaviour:

1. `GET /v1/<collection>/<id>` answers the record with the same field names as in the list.
2. An id that does not exist answers 404, and so does a collection that does not exist.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The shoe `gazelle-indoor-solar-gold` | `GET /v1/products/gazelle-indoor-solar-gold` | 200 and the record with its name, price and lists |
| AC-2 | The shop's terms record | `GET /v1/shop/shop` | 200 and a record with `free_delivery_from` 2000000 |
| AC-3 | No shoe has the id `nope` | `GET /v1/products/nope` | 404 |
| AC-4 | The record's list-endpoint copy and its record-endpoint copy | Both are read | They are equal field for field |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | Unknown id or collection | 404 with an error body; not an empty 200 |
| Upstream failure | The database cannot be reached | 500 with a generic error body |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| One record of any collection | read | yes | As DATA-002 |

**Requirement DATA-004 — Fields a visitor may not read**

*As a* customer, *I want* my phone, email, address and note never to be readable by strangers, *so that* ordering from the shop does not publish my details.

Behaviour:

1. The fields the schema marks private — an order's phone, email, address and note — are never part of any response to a caller with no session: not in a list, not in a record, and they cannot be used as a filter. They are left out of the database query itself, not removed afterwards.
2. The records' bookkeeping — the source note (`_source`), the list of invented fields (`_invented`) and a shoe's `price_listed` (the price on another shop's page) — is not part of any response to a caller with no session either. They are nobody's business but the owner's; no page of the design reads them.
3. Every other field is returned, including the ones the design only reads for layout (a shoe's `swatch`, a way to pay's `icon`).
4. Until the orders and sign-in stories land, the orders collection is still readable by anyone, minus the four private fields: order number, time, status, shoe, size, pairs, total, payment, paid, delivery, carrier, tracking, expected date, customer name and city are visible to a stranger. This is temporary; the stories "Checkout and placing an order", "Track an order by number and phone" and "Dealer sign-in and protected admin area" end it by making orders readable only with an order token or a dealer session. The pull request states this.
5. No collection is closed to visitors in this module: questions (`faqs`) are public, because the Returns, Delivery, Size chart and Track pages read them.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The order `doi-24817` | `GET /v1/orders/doi-24817` with no session | 200 without the keys `phone`, `email`, `address` and `note`; `customer` and `city` are present |
| AC-2 | The seeded orders | `GET /v1/orders` with no session | No item contains `phone`, `email`, `address` or `note` |
| AC-3 | The seeded orders | `GET /v1/orders?phone=0938%20410%20552` | 400 naming `phone`; no order is returned |
| AC-4 | The seeded shoes | `GET /v1/products` with no session | No item contains `_source`, `_invented` or `price_listed` |
| AC-5 | The seeded lines | `GET /v1/lines/samba` | No `_source` or `_invented` key; `name`, `blurb`, `since`, `fit_note` are present |
| AC-6 | The seeded shoes | `GET /v1/products` | Each item still has `swatch`, `image`, `image_back`, `image_detail` |
| AC-7 | The seeded ways to pay | `GET /v1/payments` | Each item has `icon` and `online` |
| AC-8 | The seeded questions | `GET /v1/faqs` with no session | 200 with `total` equal to the file's count (the collection is public) |
| AC-9 | The deployed API | A request with no session asks for each collection's list | No response body anywhere contains the text of an order's phone, email, address or note |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Permission | A visitor asks for a private field as a filter | 400 naming the field (DATA-002) |
| Permission | A dealer session reads orders | Not applicable in this module: sessions do not exist yet; the `admin` module returns private fields to a dealer |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Orders: phone, email, address, note | read | — | Never returned to a caller with no session |
| `_source`, `_invented`, shoe `price_listed` | read | — | Never returned to a caller with no session |

**Requirement DATA-005 — The frontend reads the API through its data module**

*As a* developer, *I want* the whole frontend to read records through one module that now calls the API, *so that* no page changes while the records move to the server.

Behaviour:

1. Only `src/data/index.ts` changes. Every name it exports today stays exported with the same signature and the same returned fields: `useCollection`, `useRecord`, `schema`, `setCollection`, `collections`, and the types `Source`, `Row`, `Field`, `Collection`, `Filter`, `Query`. No component or page file is edited.
2. The API's address is `import.meta.env.VITE_API_URL`, or `/api` when it is not set. Calls are same-origin so a session cookie travels with them.
3. `useCollection(name, query)` asks for the first page when it mounts. Its `where` entries become `<field>=<value>`; an array becomes `<field>=a,b`; a range becomes `<field>_from` and `<field>_to`; empty filters are left out; `q` and `sort` are sent as given; `pageSize` becomes `limit`.
4. It returns `{ items, total, pages, page, hasMore, loading, error, loadMore }`. `items` holds the pages loaded so far; `loadMore()` asks for the next page and appends it; `total` is the API's `total`, never `items.length`. When the query names a `page`, `items` is that one page alone and `hasMore` is false.
5. When the name or the query changes, the old items are dropped and the first page of the new query is asked for. An answer to an older query that arrives late never replaces the newer query's items.
6. `loading` is true from the moment a request starts until it ends, success or failure. `error` is an `Error` when a request fails (the server cannot be reached, or answers a status that is not 2xx) and is null otherwise; the items already loaded stay.
7. `useRecord(name, id)` returns `{ item, loading, error }`. A 404 gives `item` undefined, `loading` false and `error` null; a request that fails any other way gives `error`.
8. `schema` is still `schema.json`, which stays in `src/data/`.
9. Every file in `src/data/` other than `index.ts` and `schema.json` is deleted, and no code imports one.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The built site on an empty cache | The home page loads | The browser requests `/api/v1/products?status=live…`, `/api/v1/categories…` and `/api/v1/lines…`; no JSON record file is requested |
| AC-2 | `useCollection("products", { where: { status: "live" }, pageSize: 12 })` | It mounts | One request asks for `page=1&limit=12&status=live`; `items` has 12, `total` is the API's total |
| AC-3 | 30 live shoes and `pageSize` 12 | `loadMore()` is called once | A request asks for `page=2`; `items` has 24; `hasMore` is true |
| AC-4 | The same, called until all are loaded | The last page arrives | `items` has 30 and `hasMore` is false |
| AC-5 | `where: { category: ["women","kids"] }` | The request is built | It carries `category=women,kids` |
| AC-6 | `where: { price: { from: 3000000, to: 3999999 } }` | The request is built | It carries `price_from=3000000&price_to=3999999` |
| AC-7 | `where: { colour: [] , status: "" }` | The request is built | Neither `colour` nor `status` is sent |
| AC-8 | A request is in flight | The hook is read | `loading` is true; when it ends `loading` is false |
| AC-9 | The server answers 500 | The hook is read | `error` is an `Error`, `loading` is false and any items already loaded are kept |
| AC-10 | The query changes while an older answer is still in flight | The older answer arrives last | `items` still shows the newer query's records |
| AC-11 | `useRecord("products","nope")` | The server answers 404 | `item` is undefined, `loading` is false, `error` is null |
| AC-12 | The server asked for `limit=500` answers `limit` 100 | `useCollection` computes `pages` | It uses 100: 130 matches give 2 pages |
| AC-13 | The repository after the change | Its files are listed | `src/data/` holds only `index.ts` and `schema.json`; the change set touches no file outside `src/data/` |
| AC-14 | The project | It is type-checked and built | Both succeed with no component edited |
| AC-15 | `VITE_API_URL` is `https://api.example.test` | A collection is read | The request goes to `https://api.example.test/v1/<collection>` |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Upstream failure | The API cannot be reached or answers 5xx | `error` is set, `loading` is false, items already loaded stay; the approved design draws no error screen, so what a page shows is unchanged from its empty-data appearance |
| Not found | `useRecord` meets a 404 | `item` is undefined with no error (a page decides what "missing" looks like) |
| Boundary | `pageSize` above 100 | The API serves 100 and the hook trusts the answer's `limit` |
| Concurrency | Queries change quickly | The latest query wins (AC-10) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| All collections | read | yes | Through the API only |

## 5. Screens

| Screen | Section in the design | Functions it serves | States that must exist |
|---|---|---|---|
| (none) | This module has no screen of its own | DATA-005 | Not applicable: the pages that read records are specified in their own modules; the approved design draws no loading or error screen for data |

## 6. Non-functional requirements

| Area | Requirement |
|---|---|
| Performance | With 1,000 shoes seeded, `GET /v1/products?page=1&limit=12` answers within 300 ms at p95 over 200 sequential requests against the running API on the project's own host |
| Privacy | An order's phone, email, address and note, and the bookkeeping fields in DATA-004, never appear in a response to a caller with no session; a check over every collection's list and record response proves it |
| Integrity | After seeding, the number of records per collection equals the number in its JSON file, and every reference resolves |

## 7. Dependencies and assumptions

- **Depends on:** the repository's `src/data/*.json` files and `schema.json`, as the owner left them when this story is built.
- **Assumption:** the shop keeps fewer than 100 live shoes, so one page of at most 100 holds every shoe that the home, bag and checkout pages ask for with a large page size. If the catalogue grows past 100, those pages see only the first 100 until they are changed.
- **Assumption:** the interim public read of orders (DATA-004 item 4) is acceptable for the short time before the orders and sign-in stories land; no real customer orders exist before then.
- **Note for the pull request:** it says that the public read of orders is temporary, that customer name and city are visible until then, and which handler leaves the private and bookkeeping fields out of the query.

| Open question | Proposed default | Who decides |
|---|---|---|
| Unknown parameters are ignored, not rejected, so that `q`, `sort`, ranges and comma lists (used by the pages today) do not break between this story and the catalogue story | Ignore unknown parameters | TL |
| Whether the records' source notes (`_source`) should be readable by the dealer through the API | Not returned to a caller with no session; the `admin` module decides whether a dealer session sees them | Stakeholder |

## 8. Traceability

| Plan item | Requirement ids | Test cases |
|---|---|---|
| Site data from the database | DATA-001, DATA-002, DATA-003, DATA-004, DATA-005 | `test-cases/site-data-from-the-database.md` |
