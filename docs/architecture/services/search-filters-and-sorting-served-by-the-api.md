# Service contracts — Search, filters and sorting served by the API (plan item 4, "S4")

Module `catalogue` · Requirements CAT-001, CAT-002 (options and paging), CAT-003 items 1–5 (the request side) · Story `docs/catalogue/stories/search-filters-and-sorting-served-by-the-api.md` · UI PR #38 · Tables: `docs/architecture/erd/search-filters-and-sorting-served-by-the-api.md`.

This file defines **the query language the list endpoints answer, the one new endpoint, and the frontend swap**, and nothing else. It extends `services/_base.md` §2.4 (the project's list contract) and `services/site-data-from-the-database.md` (S3's reader); it reuses the error envelope and closed error catalog (`_base.md` §2.3), adds **no code to the catalog**, no second list format and no second pagination scheme. Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/product-filters` reaches Go as `/v1/product-filters`.

## 1. Endpoints

| # | Method and path | Auth | Package | Traces to |
|---|---|---|---|---|
| 3.2 | `GET /v1/{collection}` — **extended**, no new route: `q`, any-of, `_from` / `_to` / `_ranges`, `sort` and (products) `size` get their meaning. Applies to the ten collections S3 mounts (`orders` while S3's reader still serves it) | public; a dealer's wider view is the sign-in story's, unchanged | `internal/data` | CAT-001, CAT-003 items 1–5, DATA-002 item 7 |
| 3.4 | `GET /v1/product-filters` — **new** | public | `internal/catalogue` (new, this one route) | CAT-002 item 8, SC-18 |

`GET /v1/product-filters` is one literal pattern mounted by `catalogue.Register(mux, pool, log)`, one line in `server.New` after `data.Register`. `POST`, `PUT`, `DELETE` and `PATCH` on it fall through to the catch-all `404 not_found` (as S3 says for every read-only path). No other route changes.

## 2. The mock, judged (UI PR #38)

`src/lib/mock/search-filters-and-sorting-served-by-the-api.ts` (`listProducts`, `getProductFilters`) and the request type in `src/lib/shop-request.ts` are **unreviewed input** (the UI review covered isolation, not contents). Read against the contract below:

| What the UI does | What the contract does | Judgment | Frontend change at the swap |
|---|---|---|---|
| Answer `{items, total, page, limit}`; `total` is every match | Identical (`_base.md` §2.4) | Sound | None |
| `ProductFilters { colours: string[], sizes: string[] }`; colours in the closed-set order, sizes ascending by value | Identical (§4) | Sound | None |
| Request fields `status`, `page`, `limit`, `q`, `sort`, `category`, `line`, `colour`, `size` (EU sizes), as arrays | Same names; a list is written `a,b` (any-of). `size` is an **EU size** on the API and the page size is `limit`, so the two never meet | Sound | `toParams` joins lists with `,` |
| `price: PriceRange[]` — a union of inclusive ranges, either end open | The wire carries it as `price_ranges=..2999999,4000000..` (§3.4). The in-app type is right as it is | Sound as the in-app form; only its wire spelling was open (story §10.1) | New `toParams` in `shop-request.ts`; `bandRanges` and `ProductsRequest` unchanged |
| Sorts `-released` (default, sent explicitly), `price`, `-price`, `-rating` | Same fields; any public field may be sorted (§3.6) | Sound | None |
| "A shoe with no rating comes last" by `rating ?? 0` | Same outcome by `NULLIF(rating, 0)` and `NULLS LAST`, and also for `sort=rating` ascending (CAT-001 item 5) | Sound (the mock only covers `-rating`) | None |
| Equal values keep the records' order | `seq` is the tie-break of every sort (SC-8) | Sound | None |
| Every word found in name, model, colourway, code, the line's name, the category's name or description; accents, case and `đ` ignored | Same fields; folding by `doi_fold` (§3.5) | Sound | None |
| A size matches only a pair that lists it and has not sold it out | Same (§3.7) | Sound | None |
| Appending a page drops a shoe already shown | Pages are offset-paged, so a shoe added between two requests can repeat; the de-duplication is needed, not defensive | Sound | None |
| `total` is `null` until the first answer, so "No pair matches that." is never claimed early | Not an API concern | Sound | None |
| `ProductCard` receives the shop's already-loaded `line` | A page of twelve cards makes no request of its own | Sound — this is the answer to story §10.3 (T3) | None |
| The shop reads `lines` with `pageSize: 30`; the header reads them with `pageSize: 20` | Two different URLs cannot share one in-flight request (S3 services §5.1), so `/shop` asks for the lines twice and SC-22 ("no more than one request for the lines") fails | **Departure** | `Shop.tsx`: `pageSize: 20` (T3) |
| `useShopList` is the only importer of the mock | The swap is two imports in one file | Sound | `shop-list.ts` imports the new client (§6) |

## 3. The query language

### 3.1 How a key of the query string is read

In this order; the first rule that fits decides.

1. `page`, `limit` — unchanged from S3 (`limit` above 100 is served as 100; `limit` is the only page size the API knows).
2. `q` — §3.5. 3. `sort` — §3.6.
4. `size`, **on `products` only** — §3.7. On any other collection `size` falls through to rule 5.
5. A key equal to the API name of a **public** column of the collection (or `id`) — an any-of filter, §3.3. A key equal to the name of a column that is not public (`phone`, `email`, `address`, `note` on `orders`; `price_listed` on `products`; `_source`, `_invented`) — `400`, as S3 says, whatever the value.
6. A key `<name>_from`, `<name>_to` or `<name>_ranges` where `<name>` is a public column of kind whole number, decimal, date or instant — a range, §3.4. Where `<name>` is a column that is not public — `400` naming the key (`phone_from` is as refused as `phone`). Where `<name>` is of any other kind — ignored.
7. Anything else — **ignored**, so a URL a person edited still answers (DATA-002 AC-11). No column of any collection may end in `_from`, `_to` or `_ranges`; the registry test asserts it, so rule 5 can never swallow a range key.

Conditions of different keys are combined with **AND** (CAT-001 item 8). The only text spliced into SQL is a column or table expression from the registry; every value, `LIMIT` and `OFFSET` is a bound parameter.

### 3.2 Values

- A repeated key uses its **first** value (S3).
- The value is trimmed. In a list (`a,b`) every item is trimmed and empty items are dropped; a value that leaves nothing (`category=`, `category=,`, `q=%20`) makes the key **ignored** (CAT-001 item 3, AC-12).
- A value holding invalid UTF-8 or a NUL byte is `400` naming the key (`<key> holds a character that is not allowed.`): PostgreSQL would refuse it with an error, and a visitor must not be able to turn an edited URL into a logged `500`. This holds for **every** key this section reads, `q` and `sort` included.
- **A comma is always a separator.** There is no escape. A text value that contains a comma (`customer=Lê, Văn`) cannot be matched by an exact filter; the dealer finds it with `q`. This settles S3's open question 1.
- Names are case-sensitive.

### 3.3 Any-of: `<field>=a,b`

A record is kept when the field equals `a` **or** `b`. Each item is read as the field's kind; one item that cannot be is `400` naming the field (the S3 sentences stand: `price must be a whole number.`, `rating must be a number.`, `featured must be true or false.`, `released must be a date written YYYY-MM-DD.`, `placed_at must be a date and time such as 2026-11-23T16:05:00+07:00.`).

| Kind | An item is | One item | Several items |
|---|---|---|---|
| text, id, reference | any non-empty text | `col = $n` | `col = ANY($n::text[])` |
| whole number | `^-?[0-9]{1,18}$` | `col = $n::bigint` | `col = ANY($n::bigint[])` |
| decimal | `^-?[0-9]+(\.[0-9]{1,6})?$` | `col = $n::numeric` | `col = ANY($n::numeric[])` |
| true/false | `true` or `false` | `col = $n::boolean` | `col = ANY($n::boolean[])` |
| date | a real date `YYYY-MM-DD`, year 1–9999 (`0000-01-01` passes Go's parser and is refused by PostgreSQL: it is `400`) | `col = $n::date` | `col = ANY($n::date[])` |
| instant (`orders.placed_at`) | RFC 3339 with an offset | `col = $n::timestamptz` | `col = ANY($n::timestamptz[])` |
| list (`sizes`, `sizes_low`, `sizes_out`) | any non-empty text | `$n = ANY(col)` | `col && $n::text[]` (any element in common) |

A NULL column never matches (a shoe with no line is in no `line=` answer). **One item is written as the plain `=`, never as a one-element `ANY`**: the shop's `status=live` then keeps the equality that `idx_products_status_released_seq` is walked on, in order. Several `status` or `category` values are `ANY`.

### 3.4 Ranges: `<field>_from`, `<field>_to`, `<field>_ranges`

Both ends are **included**. Applies to whole-number, decimal, date and instant columns: `price`, `rating`, `review_count`, `stock`, `qty`, `total`, `fee`, `since`, `cm`, `released`, `date`, `eta`, `placed_at`.

- **Numbers** (whole-number and decimal columns alike) are read as decimals (`^-?[0-9]+(\.[0-9]{1,6})?$`) and compared as numbers: `price >= $n::numeric`. `price_to=2999999.5` is a number and is accepted.
- **Dates** (`released`, `date`, `eta`): `YYYY-MM-DD`, compared as dates.
- **`placed_at`** is an instant but is ranged by **Vietnam calendar days**: `placed_at_from=2026-11-01` is `placed_at >= ($n::date::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')` and `placed_at_to=2026-11-30` is `placed_at < (($n::date + 1)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')`, so `idx_orders_placed_at` is usable and the whole of the last day is in. (`placed_at=<RFC 3339>` stays an exact-instant any-of filter, as in S3.)
- An empty end is ignored. `from` greater than `to` is **not an error**: nothing matches (`total: 0`).
- A value that is not a number or a date is `400` naming the key: `price_from must be a number.` / `released_to must be a date written YYYY-MM-DD.`

**A union of ranges — `<field>_ranges`** (story §10.1, decision T1). A comma-separated list of ranges, each written `from..to`; either end may be left empty, not both. A record is kept when it is in **any** of them. Examples: `price_ranges=..2999999,4000000..` is "under 3.000.000 ₫ or 4.000.000 ₫ and over"; `released_ranges=2026-01-01..2026-03-31,2026-07-01..2026-09-30`. SQL: `((col <= $a) OR (col >= $b))`, each range the `AND` of the ends it has, the ranges `OR`ed, every end its own bound parameter. A part with no `..`, with both ends empty, or an end that is not a number or date is `400` naming the key (`price_ranges must be ranges written from..to, such as ..2999999,4000000..`); more than **20** ranges is `400` (`price_ranges can hold at most 20 ranges.`). The first `..` splits a part, so `4.5..5` is the range 4.5 to 5. `_ranges` combines with `_from` / `_to` of the same field by AND, like every pair of different keys.

### 3.5 `q`: free text

1. The value is trimmed. Empty means **no search** (`q=%20%20`, SC-16).
2. More than **100 characters** (runes, after trimming) is `400`, `errors.q` = `q can have at most 100 characters.` (SC-15: 101 characters).
3. The text is split on white space (`strings.Fields`); exact repeats are dropped. Each word is bound as its own text parameter.
4. A word is folded **in SQL, once per statement**: `(SELECT doi_fold($n))`. An uncorrelated scalar subquery is evaluated once per execution, whether the plan is custom or generic; `doi_fold($n)` written inline would be evaluated for every shoe. Case, accents and `đ` are therefore folded by the one function that folded the stored text (D6), never by a second implementation in Go.
5. A record matches when **every** word is found, **as part of a value**, in at least one searched field: `strpos(<folded haystack>, <folded word>) > 0`. `strpos` reads `%` and `_` as themselves (CAT-001 AC-8, AC-9); no `LIKE` is used, so nothing is escaped. A word that folds to nothing (a lone combining mark) is found everywhere.
6. The searched fields are fixed per collection; a column that is not public is **never** searched:

| Collection | For each word `w`, a record matches when |
|---|---|
| `products` | `strpos(t.search_text, w) > 0` — the five own fields (ERD §2.1) — **or** `t.line_id IN (SELECT l.id FROM lines l WHERE strpos(doi_fold(l.name), w) > 0)` **or** `t.category_id IN (SELECT c.id FROM categories c WHERE strpos(doi_fold(c.name), w) > 0)` |
| `orders` | `strpos(doi_fold(t.number \|\| ' ' \|\| t.customer), w) > 0` — and nothing else: no phone, email, address, note, city |
| `reviews` | `strpos(doi_fold(t.name \|\| ' ' \|\| t.body), w) > 0` **or** `t.product_id IN (SELECT p.id FROM products p WHERE strpos(doi_fold(p.name), w) > 0)` — the customer's name, the review text, the shoe's name |
| `lines`, `categories` | `strpos(doi_fold(t.name \|\| ' ' \|\| t.blurb), w) > 0` |
| `delivery` | the same over `name`, `days`, `note` |
| `payments` | over `name`, `note` |
| `faqs` | over `topic`, `question`, `answer` |
| `shop` | over `standard_days`, `dispatch_cutoff`, `reply_time` |
| `sizes` | over `eu`, `uk`, `us_m`, `us_w` |

   Pictures (`image*`), design keys (`icon`, `swatch`), ids and references are not "words" and are not searched. The words are `AND`ed; the per-word groups above are the registry's `Search` value of each collection.
7. `q` is never written to a log (S3: the access log records the path, never the query string).

### 3.6 `sort`

- Absent or empty: `ORDER BY t.seq` (S3's order). Otherwise `sort=<field>` ascending or `sort=-<field>` descending, **one** field: a comma, a space, an empty name (`sort=-`) or a name that is not a public column of the collection (or `id`) is `400`, `errors.sort` = `sort must be a field of this list, such as price or -price.`; a column that is not public is `400`, `<field> cannot be sorted.`
- The order is always `<key>, t.seq ASC`: **equal values keep the seed's order whichever way the key goes**, so a page never repeats or skips a record (SC-8). The tie-break is unique, so the order is total.
- **A record with no value comes last in both directions.** For a column that can be NULL (`lines.since`, `reviews.rating`, `sizes.cm`, `orders.eta`, `orders.product` / `size` / `qty` — they come from a `LEFT JOIN` —, `products.line`, `products.category`) the key is written `… ASC NULLS LAST` / `… DESC NULLS LAST`. For a column that is `NOT NULL` **no `NULLS` clause is written** (it would stop `released DESC` from using the index; ERD §4, S4-2). `products.rating` is the one stored `0` that means "none": its key is `NULLIF(t.rating, 0)` with `NULLS LAST` (ERD E6).
- Key by kind: whole number, decimal, true/false, date, instant — the column itself. Text — `doi_fold(left(<column>, 100)), <column>` (so `Đại` sorts with the D's and `samba` with `Samba`, whatever collation the database was created with; 100 characters bound the cost of a sort on a long text column, which only the dealer's lists can ask for). List columns — the array, compared element by element.
- A reference column (`line`, `category`, `product`, `payment`, `delivery`) sorts by its id, not by the referenced name.

### 3.7 `size`: the EU size filter, shoes only

`size=a,b` keeps the shoes in which **at least one given size is listed and is not sold out**. A size on its last pairs is not sold out. `size=99` is `200` with `total: 0` and `items: []`. Items are trimmed and empty items dropped; an item may be any text (a size is `36 2/3`; the browser sends it percent-encoded). One condition, spelled for whichever form the stock is in on `main`:

| Form | Where the stock is | The condition (`$n` is the size list as `text[]`) |
|---|---|---|
| **A** — before S5 | the S3 columns `sizes` and `sizes_out` of `products` | `EXISTS (SELECT 1 FROM unnest($n::text[]) AS w(s) WHERE w.s = ANY(t.sizes) AND NOT (w.s = ANY(t.sizes_out)))` |
| **B** — after S5 | `product_sizes` | `EXISTS (SELECT 1 FROM product_sizes sz WHERE sz.product_id = t.id AND sz.size = ANY($n::text[]) AND sz.pairs > 0)` — served by `idx_product_sizes_size_in_stock`. The inner alias is `sz`, not `ps`: after S5 the products `FROM` already uses `ps` for `product_stock`, and a shadowed alias is legal SQL that nobody reads correctly |

The two are the same fact (the shoe lists the size, and its pairs are not 0); S5's `product_stock` makes `sizes_out` exactly "pairs = 0" and `sizes` exactly "has a row". **Whichever of S4 and S5 merges second rewrites this one clause** and `product-filters`' size statement (§4); S5's services file §5 says the same from its side. Tests reach the shoes through one fixture helper that knows the form (ERD §6), so no test changes when the form does.

On any other collection `size` is not this filter: `orders` has a `size` column (`oi.size`, the first pair's) and is filtered by it exactly, as any-of text; a collection with no `size` column ignores the key.

### 3.8 The statement

S3's one statement per list stays: the count and the page, together, so they are one snapshot and `total` can never disagree with the page.

```sql
SELECT (SELECT count(*) FROM <from> WHERE <cond>) AS total,
       coalesce((SELECT json_agg(p) FROM (
           SELECT <public select list> FROM <from>
           WHERE <cond> ORDER BY <order> LIMIT $L OFFSET $O) p), '[]'::json)::text AS items
```

`<cond>` is built in a fixed order — filters in the registry's column order, then ranges (`_from`, `_to`, `_ranges`), then `size`, then the words in the order typed — joined by `AND`, with `TRUE` when there is none. Bound parameters are numbered in that order, `LIMIT` and `OFFSET` last. `<order>` is §3.6. The page subquery carries the `ORDER BY`; if the sort key is projected, `json_agg(p ORDER BY …)` repeats it so the order does not rest on the planner's habit (S3's review asked for it); either way SC-8 is a DB test that reads every page of size one and finds no record twice.

The request `GET /v1/products?status=live&category=women&line=samba&size=38&price_to=2999999&q=og&sort=price` (SC-10), Form B:

```sql
WHERE t.status = $1 AND t.category_id = $2 AND t.line_id = $3
  AND t.price <= $4::numeric
  AND EXISTS (SELECT 1 FROM product_sizes sz WHERE sz.product_id = t.id AND sz.size = ANY($5::text[]) AND sz.pairs > 0)
  AND (strpos(t.search_text, (SELECT doi_fold($6))) > 0
       OR t.line_id IN (SELECT l.id FROM lines l WHERE strpos(doi_fold(l.name), (SELECT doi_fold($6))) > 0)
       OR t.category_id IN (SELECT c.id FROM categories c WHERE strpos(doi_fold(c.name), (SELECT doi_fold($6))) > 0))
ORDER BY t.price ASC, t.seq ASC LIMIT $7 OFFSET $8
```

The shop's default (`status=live&sort=-released&limit=12`) is `WHERE t.status = $1 ORDER BY t.released DESC, t.seq ASC`, which is exactly `idx_products_status_released_seq`.

### 3.9 Answer and errors

**Success** — `200`, `{ "items": [ … ], "total": n, "page": p, "limit": l }`, exactly S3's: `items` is always an array, `total` every match, `page` and `limit` as applied. Headers as S3: `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`.

**Errors** — every non-2xx body is the project's `{error, message, errors}`; `message` is "Some parameters are not valid." for a 400. `errors` is keyed by the **parameter's own name** and holds **every** fault of the request (S3: `page` and `limit` first).

| Code | HTTP | Trigger | `errors.<key>` |
|---|---|---|---|
| `bad_request` | 400 | `q` over 100 characters | `q` — `q can have at most 100 characters.` |
| `bad_request` | 400 | `sort` that is not `[-]<public field>` | `sort` — `sort must be a field of this list, such as price or -price.` |
| `bad_request` | 400 | `sort` on a column that is not public | `sort` — `<field> cannot be sorted.` |
| `bad_request` | 400 | A value that is not the field's kind | `<field>` — the S3 sentence |
| `bad_request` | 400 | A range end that is not a number / date | `<field>_from`, `<field>_to` — `… must be a number.` / `… must be a date written YYYY-MM-DD.` |
| `bad_request` | 400 | A `_ranges` part that is not `from..to`, empty at both ends, or more than 20 parts | `<field>_ranges` — the two sentences of §3.4 |
| `bad_request` | 400 | A filter, or a range key, on a column that is not public | `<key>` — `<key> cannot be used as a filter.` |
| `bad_request` | 400 | Invalid UTF-8 or a NUL byte in any value | `<key>` — `<key> holds a character that is not allowed.` |
| `bad_request` | 400 | `page` / `limit` not whole numbers from 1 | as S3 |
| `internal` | 500 | The database cannot be reached, a statement fails or times out (5 s) | none. Generic body `Something went wrong. Please try again.`: no SQL, table, column or stack trace (SC-17). `unavailable` 503 is not used for a failed query |

No `401`, `403`, `404` or `422`: a list is public, the collection is fixed by the route, and a bad value is a malformed parameter, not a field of a body.

### 3.10 What the criteria ask, as requests

| Request | Reads as |
|---|---|
| `products?q=samba%20og` and `?q=og%20samba` | both words, either order, in the five own fields or the line's / category's name (SC-1) |
| `products?q=CUA%20L%C3%92` | folded to `cua lo`, finds "Cửa Lò" (SC-2) |
| `products?q=100%25`, `?q=_` | the literal `100%` and `_` (SC-4) |
| `products?category=women,kids&line=samba` | `category` any-of AND `line` (SC-5) |
| `products?price_from=3000000&price_to=3999999` | inclusive range as numbers (SC-6) |
| `products?price_ranges=..2999999,4000000..&status=live` | the shop with "Under" and "4.000.000 ₫ and over" ticked (SC-36) |
| `products?sort=-rating` | highest first, unrated last; ties by `seq` (SC-7) |
| `products?size=38,40` | either size listed and not sold out (SC-9) |
| `products?size=99` | `200`, `total: 0`, `items: []` (SC-16) |
| `orders?q=nam`, `orders?size=39%201%2F3` | number or customer contains "nam"; `size` is the first pair's size, exact (SC-13, SC-16) |
| `reviews?product=samba-og-w-core-black&sort=-date` | that shoe's reviews, newest first (SC-14) |
| `products?sort=zzz`, `?price_from=abc`, `?released_to=2026-13-40`, `?q=<101 characters>` | `400` naming `sort`, `price_from`, `released_to`, `q` (SC-15) |

## 4. `GET /v1/product-filters`

**Purpose** — the options of the shop page's Colour group and Size (EU) pills: **every** colour and size of the live shoes, whatever filters the visitor has set (CAT-002 item 8, AC-20). **Traces to** — CAT-002, SC-18, SC-29. **Auth** — none. **Parameters** — none; any given is ignored, as is every header but the standard ones.

**Success** — `200`

```json
{ "colours": ["black", "white", "pink"], "sizes": ["35 1/2", "36", "36 2/3", "37 1/3"] }
```

| Field | Type | Description |
|---|---|---|
| `colours` | string[] | Each distinct non-empty `colour` of the shoes with `status = 'live'`, in the order of the closed set: `black`, `white`, `pink`, `purple`, `brown`, `yellow`, `blue`, `grey`, `green` (`ck_products_colour`). `[]` when there is none, never `null` |
| `sizes` | string[] | Each distinct size **listed** by at least one live shoe — in stock or sold out, a listed size is a size the shop sells — by numeric value ascending: `35 1/2`, `36`, `36 2/3`, `37 1/3`, …, `45 1/3` (`36 2/3` follows `36`; plain text order would put `40` before `9`). `[]` when there is none |

**Drafts never contribute, even to a dealer** (the contract's existing rule): the statement is `status = 'live'` for every caller, and no session is read.

**One statement, read with `QueryRow`** (S3's `Querier` has that one method), returning the two arrays as JSON text; `$1` is the closed colour set as `text[]`, owned by the package as one constant and checked against `schema.json`'s `colour` options by a test (the one place that set is written outside the migration).

Form A (before S5) — sizes are sorted by the handler, with `sizeValue`: `35 1/2` is 35.5, `36 2/3` is 36 + 2/3, a whole number is itself; a size that does not read as a number sorts last, by text:

```sql
SELECT coalesce((SELECT json_agg(c.colour ORDER BY array_position($1::text[], c.colour))
                 FROM (SELECT DISTINCT t.colour FROM products t
                       WHERE t.status = 'live' AND t.colour <> '') c), '[]'::json)::text AS colours,
       coalesce((SELECT json_agg(s.size)
                 FROM (SELECT DISTINCT x AS size FROM products t, unnest(t.sizes) AS x
                       WHERE t.status = 'live') s), '[]'::json)::text AS sizes
```

Form B (after S5) — the same colours, and the sizes ordered by `product_sizes.size_value` in SQL (ERD §3.11 of the project file), so `sizeValue` and the handler's sort are deleted by S5:

```sql
… coalesce((SELECT json_agg(s.size ORDER BY s.size_value, s.size)
            FROM (SELECT DISTINCT ps.size, ps.size_value FROM product_sizes ps
                  JOIN products t ON t.id = ps.product_id WHERE t.status = 'live') s), '[]'::json)::text AS sizes
```

**Headers** — `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store` (a dealer's edit shows on the next read), `X-Content-Type-Options: nosniff`.

| Code | HTTP | Trigger |
|---|---|---|
| `internal` | 500 | The database cannot be reached, the statement fails or times out (5 s). Generic body; the cause is logged beside the route; no query string is logged |

p95 ≤ 300 ms at 10,000 shoes (estimated tens of milliseconds).

## 5. Packages and code that changes

| Package | Change |
|---|---|
| `internal/data` (S3, in PR #39) | **The grammar of §3 goes here**, in `query.go`'s parameter loop, as S3's services file §4.1 already says: `parseList` also returns `ranges`, `words`, `sizes` and `sort`; `where()` builds the any-of forms, ranges, words and size; `listSQL` gains `ORDER BY`. `collections.go`: each `Column` gains `Nullable bool` and an optional `SortExpr` (`products.rating`: `NULLIF(t.rating, 0)`); each `Collection` gains a `Search` value (the per-word condition of §3.5) and `products` a `Sizes` clause (§3.7). The store keeps its one-statement, `QueryRow`-only shape. The `access` classes are untouched: a non-public column is neither filtered, sorted, ranged nor searched. S3's value rule that "a comma list is ignored" and "`q`, `sort`, `_from`, `_to`, `size` are ignored" is **replaced** by §3 |
| `internal/catalogue` (**new**) | `Register(mux, db, log)` mounts `GET /v1/product-filters` (§4): `handler.go`, `store.go` (its own one-method `Querier`), the colour-order constant, `sizeValue` (Form A only), and tests. Reads `products` (and `product_sizes` after S5), writes nothing. It is the shop's read side named in the project's ownership table ("read through `data` / `catalogue`"). It does **not** import `data`'s registry or store (overview §3: packages do not import each other's stores) |
| `internal/server` | One line: `catalogue.Register(mux, pool, log)` after `data.Register`; the route test (`routes_test.go`) asserts `GET /v1/product-filters` answers `200` and `POST` answers `404` |
| `migrations/` | One pair, `products_search_text` (ERD §5) |
| **The orders seam** | `parseList` and `listSQL` already take the `*Collection` they work on and look nothing up by name; that stays true. S5 removes `orders` from the registry **that `data` mounts**. The `orders` Collection value — its columns, the `LEFT JOIN`, the `number` + `customer` search, the sort keys — must **move to `internal/orders`**, not be deleted, and the two functions are exported (`data.ParseList`, `data.ListSQL`) the day the dealer's orders list (sign-in / manage-orders stories) calls them: the language is written once and the dealer's lists get it free (ADMIN SRS §7 depends on that). Until then SC-13 is a DB test on the Collection value, and over HTTP it is readable only while S3's reader still serves `orders` |
| Frontend | §6. `src/data/index.ts` and `src/lib/table.ts` are **not** touched |

Unit and DB tests the backend PR carries (Dev writes them; each names a criterion): the grammar table (every sentence of §3.9, every kind of §3.3); the SQL-capture test extended to `q`, `sort`, `size` and `_ranges` for **every** collection (no private column named, `search_text` never in a select list, a hostile `q` such as `'); DROP TABLE products; --` appears only as an argument); the registry tests (no column ends in `_from`, `_to`, `_ranges`; the closed colour set equals `schema.json`'s); the DB tests of ERD §6 (SC-1 to SC-16, SC-18, SC-19 with 10,000 shoes); SC-17 with a failing `Querier` as S3's test does.

## 6. Frontend change at the swap

The UI PR (#38) is merged first (§8). The backend PR then does exactly this, and nothing else:

| Change | Where |
|---|---|
| New `toParams(req: ProductsRequest): Record<string, string>`: `status=live`, `page`, `limit=12`, `sort` (always, `-released` included), `q` when not empty, `category` / `line` / `colour` / `size` each as `a,b` when not empty, and `price_ranges` as `from..to` parts joined by `,` (`[{to: 2999999}, {from: 4000000}]` → `..2999999,4000000..`). A test pins the union and that `limit` stays 12 whatever `size` holds | `src/lib/shop-request.ts`, `src/test/shop-request.test.ts` |
| New API client: `listProducts(req)` = `api.get<Page<Product>>("products", toParams(req))` and `getProductFilters()` = `api.get<ProductFilters>("product-filters")`, over `src/lib/api.ts` (S3 services §5.1). `ApiError` propagates; `useShopList` already turns a rejection into `error` and keeps `total` null | `src/lib/search-filters-and-sorting-served-by-the-api.ts` (the layout's `src/lib/{story-slug}.ts`) |
| Point the two imports at it; the hook's own logic (latest request wins, page 1 on a new or earlier query, de-duplication) is unchanged | `src/lib/shop-list.ts` |
| Delete the mock — including the `doi:mock` fault knobs plan item 16 added to it — and fix the `vi.mock` path in the test | `src/lib/mock/search-filters-and-sorting-served-by-the-api.ts`, `src/test/shop-list.test.tsx` |
| Read the lines with `pageSize: 20`, the header's, so `/shop` makes one lines request (T3) | `src/pages/Shop.tsx` |

Not changed: `src/data/index.ts` (the shop does not go through `useCollection`, so `Filter` needs no way to say "union"), `src/lib/table.ts` (the address keeps `size=<eu size>`; the page size is the constant 12 and never `t.size`; how "Clear all" removes the size is plan item 16's, already built so that this does not get harder), `ProductCard`, `Grid`, `FilterRail`, `Select`, every other page, `content.json`, `theme.css`, `App.tsx`. A grep of `src/` for `sort:` with `useCollection` finds only public fields today (`-placed_at` on orders, `-date` on reviews, the dealer's table columns); Dev repeats the grep at the swap, because an unknown `sort` is now a `400` where S3 ignored it.

Plan item 16's loading and error views read their states from `useShopList`; once the mock is gone they are exercised by the browser's request interception and offline mode, which is also how SC-40 here is read ("held for 3 seconds", "a 500 or the browser offline").

## 7. Decisions, and what was rejected

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| T1 | How one request carries a union of price bands (story §10.1) | A generic `<field>_ranges=from..to,from..to` on every numeric and date field (§3.4). The shop sends `price_ranges=..2999999,4000000..`. Combines with `_from` / `_to` by AND | Repeating `price_from` / `price_to` (`price_from=0&price_to=2999999&price_from=4000000`): pairs ends by position, which no URL parser promises. Band names on the server (`price=under-3m,4m-plus`): `price` is an exact-value any-of, and the API would have to know the shop's copy. Two requests merged in the browser: `total` and "Show more" would be wrong. A richer `Filter` in `src/data/index.ts`: the shop does not use it |
| T2 | `size` on `/shop` (story §10.2) | On the API `size` is the EU-size filter of shoes and the page size is `limit`; the shop always sends `limit=12`, so the two never meet. `table.ts` is untouched | Renaming the address parameter (`sizes=`, `eu=`): a shared link and every home / footer entry would change for no gain on the API side |
| T3 | One lookup per line, not one per card (story §10.3) | The shop passes the lines it already holds to the cards (already built, PR #38). **No cache in the data module.** `/shop` reads the lines with the same `pageSize` as the header, so the two identical simultaneous `GET`s are one request (`api.ts` shares them) | A cache in `src/data/index.ts`: a dealer's edit must be on the page at once (`no-store`), and `setCollection` would need to invalidate it. A `lines` field in `product-filters`: SC-18 pins that answer to `{colours, sizes}` |
| T4 | Where the code lives | The grammar in **`internal/data`**, next to the registry, the access classes and the one list statement; `internal/catalogue` for `product-filters` (the story and overview call the package `catalogue`) | Grammar in `catalogue` "over `data`": it needs the registry, so it either imports `data`'s internals (forbidden) or copies them (drift), and a second handler for `GET /v1/products` is a duplicate pattern, which Go's mux refuses at start-up. Moving `Column` and `Kind` out of PR #39: rework of reviewed code |
| T5 | Where `status=live` is enforced | Unchanged: the shop sends it and the exact filter does the rest (SC-11, SC-23); the API refusing a draft to a visitor whatever is asked is ADMIN-013 | Enforcing it here: needs a session to tell dealer from visitor, which does not exist yet |
| T6 | A comma inside a text value | Always a separator; no escape syntax | An escape (`\,`): every client would need it and none of the criteria does |
| T7 | Sort order and missing values | §3.6: `seq` last; NULLs last in both directions; `NULLIF(rating, 0)`; text by `doi_fold(left(…, 100))`; no `NULLS` clause on `NOT NULL` columns | `NULLS LAST` everywhere: the default shop order would stop using its index. Database collation for text: a `C` collation sorts `Đ` after `Z` |
| T8 | Folding a typed word | In SQL, once per statement (`(SELECT doi_fold($n))`) | In Go: a second fold that must equal the first on every character. Inline `doi_fold($n)`: evaluated per row under a generic plan |
| T9 | A NUL byte or invalid UTF-8 in a value | `400` naming the key, for every key | Letting PostgreSQL refuse it: a `500` and an error log that any visitor can produce (the review of PR #39 found it for filters) |
| T10 | The `size` clause before and after S5 | Two spellings of one fact, §3.7; the second story to merge rewrites the clause | Waiting for S5, or building a compatibility view: a view over columns S5 drops |
| T11 | The orders search while S5 empties the data registry | The `orders` Collection moves to `internal/orders` with its search and sort (§5) | Deleting it with S5 and rebuilding it for the dealer's list: two implementations of one grammar |
| T12 | Caching `product-filters` | `Cache-Control: no-store`, no server cache | A cache: a dealer who hides the last live shoe in a colour would still see the option for a while |

## 8. Sequencing and dependencies

| Matter | Rule |
|---|---|
| PR #24 → PR #39 (S3) → this story's backend | In that order. The new migration is stamped later than `20261125100200` and uses `doi_fold` and `products` from S3's migrations 1 and 2 |
| This story's backend and S5's backend (PR #32 is its UI) | **Either order.** Each rewrites the `size` condition and the `product-filters` size statement if the other merged first (§3.7, §4); the migrations are independent (ERD §5) |
| PR #38 (this story's UI) | Merges before this story's backend PR, which edits `shop-list.ts` and deletes the mock |
| PR #40 (plan item 16, stacked on #38) | Edits the mock (fault knobs) and `shop-list.ts`. Whichever of #40 and this backend PR merges second rebases; the mock's deletion wins and the knobs go with it |
| PR #39's review finding on `bindValue` (NUL, invalid UTF-8, year 0) | If it has landed, this story reuses the rule; if not, §3.2 and §3.3 add it for every key |
| Dealer's lists (ADMIN-010/011/013/016) | Use this language with no change; they need only the registry's public columns and, for orders, the seam of §5 |
| `docs/architecture/erd.md` and `services.md` | Generated from the per-story files when this design merges; not edited here |

## 9. Non-functional

| Aspect | Target and how |
|---|---|
| p95 latency | `GET /v1/products` with a two-word `q`, one `category` and a `sort`: ≤ 500 ms (SC-19), design target ≤ 300 ms (the project list budget), at 10,000 shoes, 200 sequential requests. By `search_text` (ERD E1) and the structured conditions running first; `product-filters` ≤ 300 ms |
| Payload | At most 100 records a page, as S3 |
| Query bounds | `q` ≤ 100 characters; ≤ 20 ranges per key; 5 s statement timeout (S3's); one statement per request (two roundtrips never) |
| Security | Every value is a bound parameter; the only spliced text is registry text; no `LIKE`, so no wildcard injection; a non-public column can be neither filtered, sorted, ranged nor searched; `q` and every filter value stay out of logs |
| Compatibility | Additive within `/v1` (project §8): parameters S3 accepted and ignored now act. A client that sent an unknown `sort` and relied on its being ignored now gets `400` (§6 lists the check). The answer's shape is unchanged |
| Observability | Failed statement: one error line with the collection or route and the error text, never a query string (S3) |

## 10. Where each criterion is met

| Criteria | Met by |
|---|---|
| SC-1, SC-2, SC-3, SC-4 | §3.5; ERD §2.1, E3 |
| SC-5, SC-6 | §3.3, §3.4 |
| SC-7, SC-8 | §3.6 |
| SC-9, SC-10, SC-30 | §3.7 |
| SC-11, SC-23 | T5; `status=live` is an exact filter |
| SC-12 | S3's paging, unchanged |
| SC-13, SC-14 | §3.5 table, §3.6; §5 "orders seam" |
| SC-15, SC-16 | §3.2, §3.4, §3.5, §3.6, §3.9 |
| SC-17 | §3.9 `internal` row; S3's `fail` |
| SC-18, SC-29 | §4 |
| SC-19 | ERD E1, §6; §9 |
| SC-20 to SC-22, SC-24 to SC-28, SC-31 to SC-34, SC-37 to SC-39 | The UI PR (#38) with `toParams` (§6); SC-22's one lines request by T3 |
| SC-35, SC-36 | §3.4 `_ranges`, §6 |
| SC-40 | The UI PR; read through request interception (§6) |

## 11. Open questions

None blocks the story.

| # | Question | Default used | Owner |
|---|---|---|---|
| 1 | The estimate in ERD E1 is a model, not a measurement | SC-19 (the DB test, then Verify against the running API) is the measurement; a miss is fixed in the statement, not by another column | TL |
| 2 | `price_ranges` is a new name in the project's list grammar | Written into this file, which becomes part of `services.md` when it merges; no SRS criterion names it | TL |
