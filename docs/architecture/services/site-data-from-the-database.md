# Service contracts — Site data from the database (plan item 3, "S3")

Module `data` · Requirements DATA-002, DATA-003, DATA-004, DATA-005 · Story `docs/data/stories/site-data-from-the-database.md` · Tables: `docs/architecture/erd/site-data-from-the-database.md`.

This file defines **the endpoints this story ships and the one frontend client it adds**, and nothing else. It reuses the project's error envelope, error catalog and offset-paging scheme; it adds no code to the catalog and no second list format. Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/products` reaches Go as `/v1/products`.

There is **no UI pull request and no mock** for this story (the pages already read the JSON files through `src/data/index.ts`). The evidence of what the frontend renders is therefore the ten JSON files themselves; §6 lists every place the API's answer differs from them and what each costs the frontend (nothing is edited).

## 1. Endpoints

Twenty literal routes — `GET /v1/<collection>` and `GET /v1/<collection>/{id}` for each of `categories`, `delivery`, `faqs`, `lines`, `orders`, `payments`, `products`, `reviews`, `shop`, `sizes` — and nothing else. Both are **public: no session, no cookie is read**. There is no `GET /v1/{collection}` wildcard: an unknown collection falls through to the existing catch-all and answers `404 not_found`, and `POST` / `PUT` / `DELETE` on any of these paths answers the same `404` (the catch-all matches before Go's `405`), as the project contract says for collections with no write route.

| # | Method and path | Auth | Traces to |
|---|---|---|---|
| 3.2 | `GET /v1/<collection>` | public | DATA-002, DATA-004 |
| 3.3 | `GET /v1/<collection>/{id}` | public | DATA-003, DATA-004 |

### 1.1 `GET /v1/<collection>` — a page of records

**Request** — query string only. Parameters are read in this order:

| Parameter | Rule | Failure |
|---|---|---|
| `page` | ASCII digits only, value 1 to 2,147,483,647. Default 1 when the parameter is absent. An empty `page=` is not a whole number | `400 bad_request`, `errors.page` "page must be a whole number from 1." — for `0`, `-1`, `abc`, `1.5`, an empty value, a value above the maximum |
| `limit` | ASCII digits only, value ≥ 1. Default **12**. **Above 100 is served as 100** (and a number too long to parse is "above 100"); the answer's `limit` says what was applied | `400 bad_request`, `errors.limit` "limit must be a whole number from 1." — for `0`, `-1`, `abc`, `1.5`, an empty value |
| `<field>` | Any **public** field of the collection (§3), plus `id`. Exact match; several parameters combine with AND (§1.2) | `400 bad_request`, `errors.<field>` when the value cannot be read as the field's kind |
| a **private** field of the collection | `phone`, `email`, `address`, `note` on `orders`; `price_listed` on `products`; `_source` and `_invented` on every collection. **The presence of the key is enough**, whatever its value, even empty | `400 bad_request`, `errors.<field>` "<field> cannot be used as a filter." Nothing is returned, never an empty list |
| `q`, `sort` | Reserved for the search story. **Accepted and ignored** here, whatever they hold | none |
| any other name, and any `<field>_from` / `<field>_to` | Not a field of this collection. **Ignored** (DATA-002 item 8) | none |
| a public field's value that contains a comma (`category=women,kids`) or is empty | A comma list is "any of", which belongs to the search story: **the parameter is ignored**, as is an empty one | none |

Names are case-sensitive. When a name is repeated the first value is used. All faults of one request are reported together (`errors` holds every one), `page` and `limit` first. `private` is per collection: `note` is private on `orders` but an ordinary filter on `delivery` and `payments`; `price_listed` is private on `products` only.

### 1.2 Filter values by kind

Every value is bound as a query parameter; none is ever concatenated into SQL. The kind of each field is in §3.

| Kind | Accepted | Match | Sentence on failure |
|---|---|---|---|
| text (incl. ids and references) | any non-empty string without a comma | exact, case- and accent-sensitive | — |
| whole number | `-?` and 1–18 digits | numeric equality | "<field> must be a whole number." |
| decimal | `-?` digits, optionally `.` and 1–6 digits | numeric equality (`rating=4.8`, `cm=21.6`) | "<field> must be a number." |
| true/false | `true` or `false` | equality | "<field> must be true or false." |
| date | a real date written `YYYY-MM-DD` | equality | "<field> must be a date written YYYY-MM-DD." |
| instant (`orders.placed_at`) | RFC 3339 with an offset | the same instant | "<field> must be a date and time such as 2026-11-23T16:05:00+07:00." |
| list (`sizes`, `sizes_low`, `sizes_out`) | any non-empty string without a comma | **contains**: the list has an element equal to the value (`sizes_low=40` matches `["37 1/3","40"]`; `sizes_low=4` matches nothing — no substring) | — |

A record whose value is NULL (a shoe with no line, a line with no `since`) never matches a value.

### 1.3 Response `200`

```json
{
  "items": [ { "id": "samba-og-w-core-black", "name": "Samba OG Black Crystal", "price": 3290000, "sizes": ["36", "36 2/3"], "…": "…" } ],
  "total": 12,
  "page": 1,
  "limit": 12
}
```

| Field | Type | Description |
|---|---|---|
| `items` | array | The records of the page, in `seq` order — the seed's order, then the order added. **Always an array**, `[]` past the last page, never `null` |
| `total` | integer | Every record that matches the filters, whatever page was asked for; never the page's length |
| `page` | integer | The page asked for, echoed (a page past the last is `200` with `items: []`, the true `total` and the page asked) |
| `limit` | integer | The limit **applied** (100 when more was asked) |

Each item has exactly the public fields of §3 under the names the JSON used. Values keep their types: money is a JSON integer (`3290000`), `rating` and `cm` are JSON numbers, `featured`/`paid`/`online` are booleans, a list is an array of strings (`[]` when empty, never `null`), a text field the data leaves empty is `""`, and a number the data leaves out is `null` (`lines.since`, `reviews.rating`, `sizes.cm`). Headers: `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`. `no-store` because a dealer's edit must be on the page at once and the orders list carries customer names; ETags can be added later without a contract change.

### 1.4 Errors

| Code | HTTP | Trigger |
|---|---|---|
| `bad_request` | 400 | `page` or `limit` not a whole number ≥ 1; a value that does not read as its field's kind; a filter on a private field. `errors` names **every** parameter at fault. `message` is "Some parameters are not valid." |
| `not_found` | 404 | The collection is not one of the ten (including a `POST` to one), via the existing catch-all |
| `internal` | 500 | The database cannot be reached, a statement times out or fails. The body is the generic `{"error":"internal","message":"Something went wrong. Please try again."}`: no record data, SQL, stack trace, host name or connection string. The cause is logged beside the request's route and status, never in the answer. (`unavailable` 503 is for `/healthz` and shutdown, not for a failed query) |

### 1.5 One record — `GET /v1/<collection>/{id}`

`{id}` must match `^[a-z0-9][a-z0-9._-]*$` and be at most 100 characters **before any query is made**; a segment that does not (uppercase, a space, `%2F`, `..`, a 101-character slug) is `404 not_found`, exactly like an id that does not exist. The query string is ignored.

**Response `200`** — the record alone (not wrapped), with the same fields, names, values and headers as the matching item of the list: both are built from the same column list, so they are equal field for field (DATA-003 AC-4).

| Code | HTTP | Trigger |
|---|---|---|
| `not_found` | 404 | The id does not exist or does not match the pattern; the collection does not exist; `delivery/express` (never seeded). Body `{"error":"not_found","message":"Not found."}` — never an empty `200` |
| `internal` | 500 | Database failure, as §1.4 |

Examples of what a stranger sees: `GET /v1/delivery/standard` → `{"id":"standard","name":"Standard","days":"2–4 working days","fee":0,"note":"…"}`; `GET /v1/shop/shop` → `{"id":"shop","standard_days":"2–4","dispatch_cutoff":"15:00","reply_time":"…","rating":4.8,"review_count":181}` with **no** `free_delivery_from`; `GET /v1/orders/doi-24817`:

```json
{"id":"doi-24817","number":"DOI-24817","placed_at":"2026-11-23T16:05:00+07:00","status":"shipped","product":"samba-og-w-crystal-white","size":"39 1/3","qty":1,"total":3090000,"payment":"momo","paid":true,"delivery":"standard","carrier":"GHN","tracking":"GHN-PT27XM","eta":"2026-11-26","customer":"Hoàng Nam","city":"Đà Nẵng"}
```

— sixteen keys; `phone`, `email`, `address` and `note` are **absent** (`email` is absent although the data holds an empty one).

## 2. Contracts reused unchanged

| What | Where it is defined | How this story uses it |
|---|---|---|
| Error envelope `{error, message, errors}` | `internal/httpx.Error`; project §2.3 | Every non-2xx body. `errors` only on 400 |
| Offset paging `{items,total,page,limit}`, default 12, max 100 served as 100, `page` ≥ 1 → else 400 | project §2.4 | §1.3. The `sort`, `q`, `a,b`, `_from`/`_to` and `size=` parts of §2.4 are **not implemented here** and are ignored (§1.1); the search story gives them meaning |
| Default order `seq` ascending | project §2.4 | The only order |
| JSON names `snake_case`, dates `YYYY-MM-DD`, instants RFC 3339 with `+07:00`, money an integer | project §2.1 | §3 |
| Request id header, access log line | project §2.1 | The access log records the path, never the query string (it can hold a filter value); this story adds no log line that does |
| Timeouts | project external-systems table | The request context is 10 s; the data handlers run their statements inside a **5 s** context of their own, and cancel the query when it ends |
| Idempotency, rate limits, background jobs, outbound calls | — | **None** in this story. Reads are naturally idempotent; the edge proxy's own limits apply; nothing is queued; the API calls nobody |

## 3. What a caller with no session may read — the column allow-lists

This is the **public view**: the column list the handlers put in `SELECT`, per collection. Columns not in it are not in the query, so they cannot be read off the wire and no later code can forget to delete them.

Legend — kind: **t** text, **i** whole number, **d** decimal, **b** true/false, **D** date, **T** instant, **L** list of text. *ref* is a text id. A field marked **null** is NULL in the table and `null` in the answer when unset. The `SELECT` expression is the plain column of the same name (alias `t`) unless the line says otherwise.

| Collection → table | Public fields, in output order | Expressions that are not the plain column | Left out of the query |
|---|---|---|---|
| `products` → `products` | `id` `name` `model` `line` *ref* `category` *ref* `price` i `status` `featured` b `is_new` b `description` `code` `colourway` `colour` `swatch` `upper` `lining` `sole` `fit` `care` `released` D `sizes` L `sizes_low` L `sizes_out` L `stock` i `rating` d `review_count` i `image` `image_back` `image_detail` (29) | `line` = `coalesce(t.line_id,'')` (filter on `t.line_id`); `category` = `coalesce(t.category_id,'')` (filter `t.category_id`); `released` = `to_char(t.released,'YYYY-MM-DD')`; `rating` = `t.rating::float8` | `price_listed`, `source`, `invented`, `seq`, `created_at`, `updated_at` |
| `lines` → `lines` | `id` `name` `blurb` `since` i **null** `fit_note` | — | `source`, `invented`, `seq`, `created_at`, `updated_at` |
| `categories` → `categories` | `id` `name` `blurb` `image` | — | same |
| `reviews` → `reviews` | `id` `product` *ref* `name` `rating` i **null** `date` D `size_bought` `fit` `body` `status` | `product` = `t.product_id`; `date` = `to_char(t.date,'YYYY-MM-DD')` | same |
| `orders` → `orders t LEFT JOIN order_items oi ON oi.order_id = t.id AND oi.line_no = 1` | `id` `number` `placed_at` T `status` `product` *ref* `size` `qty` i `total` i `payment` *ref* `paid` b `delivery` *ref* `carrier` `tracking` `eta` D `customer` `city` (16) | `placed_at` = `to_char(t.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh','YYYY-MM-DDThh24:mi:ss') \|\| '+07:00'` (the `T` quoted in the format); `product` = `coalesce(oi.product_id,'')`; `size` = `coalesce(oi.size,'')`; `qty` = `coalesce(oi.qty,0)`; `payment` = `t.payment_method_id`; `delivery` = `t.delivery_option_id`; `eta` = `coalesce(to_char(t.eta,'YYYY-MM-DD'),'')` (filter on `t.eta`) | **`phone`, `email`, `address`, `note`**, `delivery_fee`, `is_sample`, `source`, `invented`, `seq`, `created_at`, `updated_at` |
| `shop` → `shop_terms` | `id` `standard_days` `dispatch_cutoff` `reply_time` `rating` d `review_count` i | `rating` = `t.rating::float8` | `source`, `invented`, `seq`, `created_at`, `updated_at` |
| `delivery` → `delivery_options` | `id` `name` `days` `fee` i `note` | — | same |
| `payments` → `payment_methods` | `id` `name` `note` `online` b `icon` | — | same |
| `faqs` → `faqs` | `id` `topic` `question` `answer` | — | same |
| `sizes` → `size_chart` | `id` `eu` `uk` `us_m` `us_w` `cm` d **null** | `cm` = `t.cm::float8` | same |

`::float8` makes `rating` and `cm` JSON numbers in their shortest form (`4.8`, `21.6`, `25.0` is served as `25` — the same value to JavaScript). A `date` is formatted in SQL, not by the driver's zone, so the answer never depends on the server's time zone; Vietnam has no daylight saving, so the offset is the constant `+07:00`.

The private/bookkeeping classification is therefore **by collection**, not global: `note` is public on `delivery` and `payments`, and a global "never serve a field called `note`" would break the delivery and payment pages.

## 4. How the reader is built

| Concern | Decision |
|---|---|
| Package | `code/backend/internal/data/` (`collections.go`, `query.go`, `handler.go`, `store.go`, tests), registered by one line in `server.New`: `data.Register(mux, pool, log)`. Packages do not import each other's stores (overview §3) |
| The registry | `collections.go` holds the ten collections as Go values: name, table, `FROM` clause, and for each field its API name, kind, `SELECT` expression, filter expression and **access class** — `Public`, `Bookkeeping` or `Private`. The handlers use **only** the `Public` columns. The registry is a hand-written mirror of `schema.json` and §3, because the backend image is built from `code/backend` and cannot embed a file of the frontend; a Go file the reader cannot be read from the request by construction is also what keeps column names out of reach of a query string |
| Drift guard | `collections_test.go` reads `../../../frontend/src/data/schema.json` (the repository root is checked out whole in CI; the test `t.Skip`s when the file is not there, as in a backend-only image build) and fails when: a schema key is neither a registry field nor `_source`/`_invented`; a registry field is not a schema key (except `id`); a schema field marked `"private": true` is not `Private`; or `price_listed`, `_source`, `_invented` is `Public` anywhere. It is also the tripwire for "Free delivery on every order" not having landed: `free_delivery_from` and the express fields would still be in the schema and the test fails |
| SQL | One statement shape per operation, built from registry text only: `SELECT <public expressions> FROM <from> WHERE <filter expr> = $n [AND …] ORDER BY t.seq LIMIT $a OFFSET $b`, and `SELECT count(*) FROM <from> WHERE <same filters>`. The only text spliced into SQL comes from the registry; every value, `LIMIT` and `OFFSET` included, is a bound parameter. `total` and `items` come from one snapshot (one statement with a CTE, or one read-only `REPEATABLE READ` transaction) so a record added between them cannot make `total` disagree with the page |
| N+1 | None: the order's first pair is a join, a list is one statement, a record is one statement |
| Privacy proof | The store depends on a two-method interface (`Query`, `QueryRow`). **`store_test.go` records the SQL a no-session read sends** and asserts, for every collection, that it contains none of the identifiers `source`, `invented`; for `orders`, none of `phone`, `email`, `address`, `note`; for `products`, not `price_listed` (word-boundary match, so `delivery_option_id` is not a false hit). The columns are absent from the query; they are not removed from the answer afterwards (SC-24). The same test covers the list and the record statements |
| Upstream failure | A test hands the store an interface that fails with `dial tcp 10.0.0.1:5432: connection refused` and asserts the answer is `500 internal` and contains neither the address nor `dial` nor `SELECT` (SC-17) |
| Not found | The record handler checks the id pattern, queries, and answers `404 not_found` on no row; the list handler never answers 404 for a known collection |
| Logging | One error-level line per failed statement: collection, error text, request id when the project adds one. Never the query string, never a filter value |
| Orders are not special-cased in the handler | `orders` is one more registry value whose `FROM` has the join. Its private columns are simply not in its `Public` list |

### 4.1 Hand-over to later stories (so nobody has to re-open this design)

| Story | What it changes here |
|---|---|
| Search, filters and sorting | Gives meaning to `q`, `sort`, `a,b`, `<field>_from/_to` and `size=` inside `query.go`'s parameter loop; adds `400` for an unknown or private `sort`. Reuses `doi_fold`. Adds no table |
| Checkout and placing an order | **Removes `orders` from the registry** in the same PR that mounts `GET /v1/orders/{id}` (order token or dealer) and `GET /v1/orders` (dealer): from then on `/v1/orders…` is served by the `orders` package and the temporary public read ends. Moves a shoe's `sizes`, `sizes_low`, `sizes_out`, `stock` from columns to `product_stock` by changing four registry expressions, not their names or kinds |
| Dealer sign-in | `auth.Resolve` puts a viewer in the context. The reader passes the registry the viewer's access classes: a dealer additionally gets `Bookkeeping` (`_source`, `_invented`, `price_listed`) and, for `orders`, the `Private` fields. **No code for this is written now**: the access class is already on every column, so the dealer view is a different argument, not a different reader |
| Manage shoes, lines, categories, reviews and orders | **May not merge a create or edit route for `products` or `reviews` without, in the same PR, the visitor rule of the project contract §2.4**: a visitor sees only `status = live` shoes and shown reviews, a draft answers 404 by id. Until then no draft or hidden record can exist, because no endpoint can create one |

### 4.2 The temporary exposure (DATA-004 item 4, D-7)

Until the checkout, track and sign-in stories land, `GET /v1/orders` and `GET /v1/orders/{id}` answer anyone, and a stranger can read each order's number, time, status, shoe, size, pairs, total, payment, paid, delivery, carrier, tracking number, expected date, customer name and city — but **not** phone, email, address or note, which are not selected. The 14 orders are the first design's samples with invented customers; no real order exists until checkout does. The backend pull request must say so (§7).

## 5. The frontend client this story adds

`src/lib/api.ts` is the project's **single API client** (overview §5): checkout and sign-in, which run in parallel after this story, import it rather than write their own. `src/data/index.ts` is rewritten over it. No component, page, style, `content.json`, `theme.css`, `index.css` or `package.json` is touched, and `schema.json` stays.

### 5.1 `src/lib/api.ts`

| Aspect | Contract |
|---|---|
| Base | `import.meta.env.VITE_API_URL ?? "/api"`. An **empty** `VITE_API_URL` is not unset and is used as given. A request goes to `${base}/v1/<path>` (`/api/v1/products`, or `https://api.example.test/v1/products`) |
| Credentials | `credentials: "include"` on every request, so a session cookie travels (same origin in a deployment) |
| Success | Parses the JSON body and returns it |
| Failure | A non-2xx answer throws `ApiError` carrying `status`, `code` (the envelope's `error`), `message` and `errors` (when present). A body that is not the envelope still throws `ApiError` with `code` `internal` and a generic message. No connection or a timeout throws an `ApiError` with `status` 0 and `code` `network` — a client-side word, never sent by the server |
| Timeout | Aborts after **15 s** (`AbortController`); the abort counts as a failure |
| Retry | A failed `GET` — no connection, a timeout, or a 5xx — is tried **once more** before the error is thrown. **A 404 is an answer, never retried**, and neither is any other 4xx. No `POST`/`PUT`/`PATCH`/`DELETE` is ever retried automatically (the checkout story re-sends its own `Idempotency-Key`) |
| Exports | `api.get(path, params?)`, `ApiError`, `apiBase()`; a later story adds `post`/`put`/`delete` here, not a second client (a second file with `api` in its name makes CI warn) |
| Sharing | Identical simultaneous `GET`s may share one in-flight request: `ProductCard` asks for the same line twelve times on one page. Allowed and welcome; no criterion requires it |

### 5.2 `src/data/index.ts` against this API

The exports and their return shapes are unchanged (`useCollection`, `useRecord`, `schema`, `setCollection`, `collections`, and the types `Source`, `Row`, `Field`, `Collection`, `Filter`, `Query`). What the module sends and relies on:

| Hook input | Request |
|---|---|
| `useCollection(name, { where, q, sort, pageSize, page })` | `GET /v1/<name>?page=<n>&limit=<pageSize or 12>&…` — `page` is 1 on mount, +1 per `loadMore()`; with `query.page` given, that page alone |
| `where: { f: "a" }` / number / boolean | `f=a` — booleans as `true`/`false`, numbers as written |
| `where: { f: ["a","b"] }` | `f=a,b` (sent now; the search story gives it meaning) |
| `where: { f: { from, to } }` | `f_from=…&f_to=…`; a range with one end sends only that end |
| `where: { f: "" }`, `[]`, `{}` or `undefined` | nothing — an unset filter must not narrow the list |
| `q`, `sort` | as given (ignored by this story's API) |
| `useRecord(name, id)` | `GET /v1/<name>/<id>` with `name` and `id` passed through `encodeURIComponent` |

What the hook trusts from the answer: `total` (never `items.length`); `limit` (the **applied** one — asking for 500 and being told 100 makes `pages = ceil(total / 100)`); `items: []` with `total > 0` on a page above 1 means "past the last" — the hook asks **once** for the last page and reports that page (the old behaviour of standing on page 7 while a filter narrows the list; story D-3). A 404 on a record is `item` undefined, `loading` false, `error` null; any other failure sets `error` (an `Error`) and keeps the items already loaded. The latest query wins: an answer to an older query that arrives last is discarded.

`collections()` and `setCollection()` keep their meaning: `collections()` returns the rows loaded so far per name, `setCollection(name, rows)` replaces a name's rows in the module's store and redraws mounted hooks **without a request**. The Studio's and `/admin`'s edits therefore keep editing the browser's copy and save nothing; a reload brings back what the database holds.

## 6. Where the answer differs from the JSON the frontend reads today

There is no mock to judge; the JSON files are the shape the pages render. Every difference, and its cost:

| # | Difference | Frontend change | Why it is right |
|---|---|---|---|
| 1 | No record has `_source` or `_invented`; no shoe has `price_listed` | **None.** `Row._source` is optional; `fields.tsx` already hides `_source`; no page reads `_invented` or `price_listed` (grep of `src/`, SRS DATA-004 item 2) | Bookkeeping is nobody's business but the owner's |
| 2 | An order has no `phone`, `email`, `address`, `note` | **None to code.** Visible effect: until the dealer-sign-in story, the `/admin` order page and edit form draw those four boxes empty. The public Track and Order pages read none of them (they read the order's `number`, `status`, `placed_at`, `customer`, `city`, …) | Left out of the query (DATA-004 item 1). The admin story returns them to a dealer |
| 3 | `placed_at` ends in `+07:00` (`2026-11-23T16:05:00+07:00`), where the JSON had no zone | **None** — see the project ERD file §7 C4: `slice(0,10)` still gives the Vietnam date, comparison of two values of one format still orders, `formatDateTime` shows the same wall time in Vietnam | A zone-less time means three things to three viewers |
| 4 | `eta` is `""` when there is no date; `lines.since`, `reviews.rating`, `sizes.cm` are `null` when absent | **None** — the JSON already wrote `""` for `eta`; the numeric fields were never absent in the seed, and the types read as `number \| undefined` | A field never changes type |
| 5 | A list is always an array, `[]` when empty | **None** — `samba-og-w-crystal-white.sizes_out` was `[]` in the file | — |
| 6 | At most 100 records per request | **None** — the Home, Cart, Checkout, shoe and `/admin` pages ask for up to 500 or 1,000 and get at most 100; the shop holds 12 shoes and 14 orders (SRS §7 assumption: fewer than 100 live shoes). A shop past 100 live shoes needs those pages paged — recorded, not built | DATA-002 item 2 |
| 7 | The search box, sort order, price band, size filter and a multi-value category/line/colour filter on the Shop page change nothing | **None** — story decision D-5: the search story, built straight after this one, makes them work | Accepted and ignored by design (DATA-002 items 7–8) |
| 8 | A page waits for the network | **None** — no loading or error screen is drawn (story D-1); while an answer is awaited, or after a failure, a page shows what it shows for empty data. The shoe page may say "not found" for a moment: the bug story "Product page waits for the shoe before saying it is missing" | The design draws none |

## 7. The pull-request note (DATA-004 item 4, SC-25)

The backend pull request states, in its description:

1. **(a)** the public read of orders is temporary: it ends when "Checkout and placing an order", "Track an order by number and phone" and "Dealer sign-in and protected admin area" make orders readable only with an order token or a dealer session;
2. **(b)** until then a stranger can read each order's number, time, status, shoe, size, pairs, total, payment, paid, delivery, carrier, tracking number, expected date, customer name and city;
3. **(c)** which handler and which column allow-list leave the private and bookkeeping fields out of the query, per collection: the handlers are `data.list` and `data.record` in `internal/data/handler.go`; the allow-lists are the `Public` columns of `internal/data/collections.go`, reproduced from §3 of this file; the test that proves it is `TestPublicSQLNamesNoPrivateColumn` in `store_test.go`.

`faqs` is public by the PM's correction (the Returns, Delivery, Size chart and Track pages read it). No other collection is closed to visitors in this story, and none is behind a session: **there is no session code in this story at all.**

## 8. Non-functional targets

| Aspect | Target |
|---|---|
| p95 latency | `GET /v1/products?page=1&limit=12` ≤ **300 ms** with 1,000 shoes seeded, over 200 sequential requests on the project's own host (SC-38). The statement is an ordered index scan on `uq_products_seq` with `LIMIT 12`, plus a `count(*)` over a table of 1,000 rows |
| Payload | At most 100 records a page; a record is about 1.5 KB (a shoe), so a page is at most about 150 KB |
| Availability | One container; "restart and apply migrations on boot". The server does not listen before the migrations succeeded |
| Security | Reads are unauthenticated by contract; SQL is parameterised, identifiers come from the registry; a private filter is `400`, never an empty list; no body carries SQL, a host or a stack |

## 9. Compatibility

Additive within `/v1` (project §8): the search story adds the meaning of `q`, `sort`, `a,b`, ranges and `size=` to parameters this story already accepts and ignores, so **no client built against this story breaks**. Two changes are breaking by design and announced here: `GET /v1/orders` and `GET /v1/orders/{id}` stop being public (§4.1), and a visitor stops seeing draft shoes and hidden reviews once they can exist.

## 10. Traceability

| Criterion | Where |
|---|---|
| SC-10, SC-11, SC-12, SC-13, SC-14, SC-15 | §1.1–§1.4 |
| SC-16, SC-20, SC-21, SC-22, SC-23, SC-24, SC-25 | §1.1, §3, §4, §7 |
| SC-17 | §1.4, §4 |
| SC-18, SC-19 | §1.5 |
| SC-1 … SC-9 | the ERD file §4–§8 (storage, seed, helpers) |
| SC-26 … SC-37 | §5 |
| SC-38 | §8 |

## 11. Open items

None blocks the story.

| # | Question | Default used | Owner |
|---|---|---|---|
| 1 | The story says a value containing a comma is "any of" and is ignored here. A **text** value that legitimately contains a comma (`customer=Lê, Văn`) is therefore ignored too | Accepted: no page filters on such a value; the search story decides what an escaped comma means | TL, in the search story |
| 2 | Between this story and the checkout story every order, with the customer's name and city, is publicly listable | Accepted and written into the pull request (§7) | Stakeholder informed through the PR |
| 3 | `GET /v1/payments` serves `cod`, `bank` and `momo` although only VNPay is offered at launch | The pages read the offered methods from `GET /v1/payment-options` (a later story) and this collection only for names; the sample orders need all four rows | PM |
