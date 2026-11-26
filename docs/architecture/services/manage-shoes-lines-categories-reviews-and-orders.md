# Service contracts — Manage shoes, lines, categories, reviews and orders (plan item 10, "S9")

Module `admin` · Requirements ADMIN-010, ADMIN-011, ADMIN-012, ADMIN-013, ADMIN-014, ADMIN-015, ADMIN-004 (write half) · Story `docs/admin/stories/manage-shoes-lines-categories-reviews-and-orders.md` · UI PR #79 · Tables: `docs/architecture/erd/manage-shoes-lines-categories-reviews-and-orders.md` (one new table, `uploads`).

This file **extends** `services/_base.md` §2.4 (the two views), §3.2 / §3.3 (reads), §3.6 (serving a picture), §3.22 (writes), §3.23 (one size's count) and §3.24 (upload). Their routes, statuses, fields and error codes **stand**; this file adds the order in which a write is refused, every field rule and sentence, the guard that lets a whole-form save carry counts without overwriting an order's pairs, the rules `_base.md` leaves open, the functions each package must offer, and the frontend's swap from its mock. It adds **no error code** to the catalog (§2.3) and no second list or error shape. Every path is written as the Go backend mounts it — **no `/api` prefix**.

The order half of the original item (an order's status, courier, cancel, delete) is plan item 17 and is not here. `PUT` / `DELETE /v1/orders/{id}` are **not** mounted by this story; until item 17's backend lands they answer `404` through the catch-all.

## 1. Endpoints

| # | Method and path | Auth | Status | Traces to |
|---|---|---|---|---|
| 3.22 | `POST /v1/products`, `/v1/lines`, `/v1/categories`, `/v1/reviews` | dealer | `201` + record | ADMIN-010, ADMIN-012 |
| 3.22 | `PUT /v1/<collection>/{id}` (same four) | dealer | `200` + record | ADMIN-010, ADMIN-013, ADMIN-015 |
| 3.22 | `DELETE /v1/<collection>/{id}` (same four) | dealer | `204` | ADMIN-014 |
| 3.23 | `PATCH /v1/products/{id}/sizes` | dealer | `200` | ADMIN-015 item 4 |
| 3.24 | `POST /v1/uploads` | dealer | `201` | ADMIN-012 |
| 3.6 | `GET /v1/uploads/{name}` | public | `200` image | ADMIN-012 |
| 3.2 / 3.3 | `GET /v1/products`, `/products/{id}`, `/reviews`, `/reviews/{id}` and the other seven collections' reads | public / dealer | unchanged routes, **new rule: two views** (§2) | ADMIN-013 |

"Dealer" is `auth.Require(auth.Admin, auth.Staff)`: a customer session answers `403 forbidden`, no session `401 unauthenticated`, both before the body is read. A collection with no write route — `delivery`, `payments`, `shop`, `sizes`, `faqs` — has **no** pattern for `POST` / `PUT` / `DELETE`, so the server's catch-all answers `404 not_found` to every caller, visitor and dealer alike (SC-3). The same holds for `PUT` / `DELETE /v1/products` with no id and for any other method on these paths.

**Mounting.** One function, `admin.RegisterRecords(mux, db, log)` (new files in `internal/admin`, §9), registers the twelve literal patterns, the `PATCH`, `POST /v1/uploads` and `GET /v1/uploads/{name}`. `server.New` gains one line after `admin.RegisterStaff`. No new setting, secret or `.env.example` line. On every answer of the write routes: `Content-Type: application/json; charset=utf-8` (not on `204`), `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`.

## 2. Reading: the visitor's view and the dealer's view

`_base.md` §2.4 already says what the two views are; on `main` only the visitor's exists and it serves **every** shoe and review. This story makes the split real.

**Which view a request gets.** The one the **session** says: `auth.FromContext` with role `admin` or `staff` is the dealer view; no session, an unreadable cookie or a `customer` session is the visitor view. There is no query parameter, header or separate route that asks for a view (decision D2).

| Collection | Visitor and customer | Dealer (admin, staff) |
|---|---|---|
| `products` | Only `status = 'live'`. Not `price_listed`, `_source`, `_invented`, `pairs` | Every shoe, drafts included, **plus** `price_listed`, `_source`, `_invented` and `pairs` |
| `reviews` | Only `status = 'live'` **and** whose shoe is `live` (decision D3) | Every review, plus `_source`, `_invented` |
| `lines`, `categories`, `delivery`, `payments`, `shop`, `sizes`, `faqs` | Every record, no bookkeeping | Every record, plus `_source`, `_invented` |
| `orders` (`GET /v1/orders`, `internal/orders`) | not public | **Unchanged by this story** — it keeps serving exactly the columns it serves today; `ListHandler` ignores the view (story decision 7) |

`pairs` is an object `size → whole number` whose keys are exactly the shoe's `sizes`: `{"38": 4, "39": 2, "40": 0}`; `{}` for a shoe with no sizes. It is read from `product_sizes`, never stored on the shoe.

**What the visitor view changes, everywhere the shoe appears.** One condition, `t.status = 'live'`, is part of the one statement every read uses (`data.listSQL`, `data.recordSQL`), so it holds for the list and its `total`, `q` search, every filter and range, the `size` filter, sorting, and a shoe by id with no case to forget:

- a draft shoe is absent from `GET /v1/products`, from a `q` search and from a `size=` match; `GET /v1/products/<id>` of a draft is `404 not_found` with **exactly** the body of an id that does not exist (`Not found.`); `GET /v1/products?status=draft` is `200` with `total: 0` (the condition and the filter are ANDed);
- a hidden review is absent from `GET /v1/reviews` and `?product=<id>` and is `404` by id;
- `GET /v1/product-filters` is **unchanged**: it already reads live shoes only, "even for a dealer" (§3.4).

A `Bookkeeping` column or `pairs` is still **never filterable or sortable**, for any caller: `?pairs=…`, `?sort=pairs`, `?_source=…` answer `400 bad_request` naming the field (§2.4, "Private fields", which now lists `pairs`).

**Cost to the frontend: none.** The shapes are those of §2.8 and the registry's names; the admin pages already call these routes, now with the dealer's cookie. A signed-in dealer's own browser therefore also sees the dealer view on the customer pages (`/shop`, a draft's `/shop/<id>`): a draft shows to them. That is by design (D2); the verifier checks SC-11 and SC-27's customer-side halves with **no session** (a private window, or `curl` without the cookie).

**`limit`.** The admin pickers ask for `pageSize: 200`; `limit` above 100 is served as 100 (§2.4), so the review form's shoe picker lists the first 100 shoes in `seq` order. Accepted at launch (12 shoes); a search-as-you-type picker is a later chore, not this story.

## 3. Writing: how a request is read and refused

### 3.1 The order of checks, the same for the four collections

| # | Check | Answer |
|---|---|---|
| 1 | The route exists | `404 not_found` (catch-all) |
| 2 | A dealer session | `401 unauthenticated` / `403 forbidden` |
| 3 | `Content-Type` is `application/json` (a `charset` is allowed) — `POST`, `PUT`, `PATCH` | `415 unsupported_media_type` |
| 4 | The body is at most **256 KiB** (two long texts of 5,000 characters of 4-byte letters, plus a dozen short ones, are under 60 KiB) | `413 payload_too_large` |
| 5 | The body is **one JSON object** (and nothing after it) | `400 bad_request`, `errors.body` "Send one JSON object." |
| 6 | `PUT` / `DELETE`: `{id}` matches `^[a-z0-9][a-z0-9._-]*$` and is at most 100 characters | `404 not_found`, no query sent |
| 7 | Every field is judged on its own (§3.3), **all faults collected**; nothing is read from the database except the existence of the record (next row) | — |
| 8 | If step 7 found faults: `PUT` first checks the record exists (`SELECT EXISTS`, no lock) | `404` when it does not exist, else `422 invalid` with **every** fault of steps 7 and 9 that can be known without a lock |
| 9 | In the transaction, under the lock (ERD §6): the record exists; every named line, category, shoe and uploaded picture exists | `404` / `422 invalid` (the faults of step 7 and these, together) |
| 10 | The state rules: the counts of a shoe against `seen_pairs` (§5) | `409 conflict` |
| 11 | Write, recalculate (§6), read the record back in the dealer view **inside the same transaction**, commit | `201` / `200` |

A refusal at any step stores nothing: the transaction is rolled back, and a `422` never opens one when step 7 alone decides it (a `POST`). A failed statement is `500 internal` (`Something went wrong. Please try again.`) with the cause in the log beside the request id and never in the body. A `POST` is not idempotent: sending it twice makes two records (the UI disables the button while a save is in flight).

### 3.2 Reading the body

- A field that is not in the collection's table below, or is read-only, is **`422`** naming the key (the SRS-forced exception of §2.5): sentence `Not a field of this record.` for an unknown key, `The shop works this out; it cannot be set.` for a derived one, `The shop makes the id.` for `id` on a `POST`. `id` in the body of a `PUT` is read and discarded (the path's id is the one changed).
- A value of the wrong JSON type (a string for a number, a number for text, an array for an object) is `422` naming the key, never `400`.
- Whole numbers are JSON numbers written as digits (`-?[0-9]+`): `2.5`, `1e2`, `2.0` and `"3"` are not whole numbers.
- Text is trimmed; leading and trailing white space never reaches the database. Length counts **characters** (runes), so a Vietnamese letter with marks counts once. A value that is not valid UTF-8 or holds U+0000 is refused: `<Label> holds a character that is not allowed.`
- `PUT` changes **only the fields sent**; `{}` is a `200` that changes nothing. `null` **clears** an optional field (text → `""`, a line or category → unset, `since` or a review's stars → empty) and is `422` on a required one (`Name is needed.`) and on a field with no empty value (`status`, `featured`, `is_new`, `released`, `date`).
- On a `POST` a field that is absent takes its default: shoe `status` `draft` (ERD Q6), `featured` and `is_new` `false`, `released` and a review's `date` today in Vietnam, a review's `status` `live`, every text `""`. The Add dialog always sends the status.

### 3.3 Fields, rules and sentences

Maximum lengths: **text 200**, **long text 5,000** characters. "Required" means present, not `null` and not empty after trimming. A sentence is a short English sentence that fits beside its field; the wording of those not quoted in the story may be improved, the **key and the status may not**.

**`products`** — required `name`, `price`

| Field | Rule | Sentences |
|---|---|---|
| `name` | text, required | `Name is needed.` · `Name can have at most 200 characters.` |
| `model`, `code`, `colourway`, `swatch`, `upper`, `lining`, `sole`, `fit`, `price_listed` | text | `<Label> can have at most 200 characters.` · `<Label> must be text.` |
| `description`, `care` | long text | `… at most 5,000 characters.` |
| `line`, `category` | the id of an existing line / category; `""` or `null` unsets | `No such line.` · `No such category.` |
| `price` | whole number 0 to 999,999,999,999 (VND); 0 is accepted | `Price is needed.` · `Price must be a whole number.` · `Price cannot be below 0` · `Price is too large (999,999,999,999 at most).` |
| `status` | `live` or `draft`; default `draft` | `Status must be live or draft.` |
| `featured`, `is_new` | true / false | `<Label> must be true or false.` |
| `colour` | `""`, black, white, pink, purple, brown, yellow, blue, grey, green | `Choose one of the listed colours.` |
| `released` | a real calendar date `YYYY-MM-DD`, 1900–2999 (ERD Q2) | `In the shop since must be a date written YYYY-MM-DD.` |
| `image`, `image_back`, `image_detail` | `""`, or `/images/` followed by letters, digits, `.`, `_`, `-`, `/` (no `..`), or an address the shop made (`/api/v1/uploads/<32 hex>.<jpg\|png\|webp>`) that exists in `uploads`; never `data:` text, never another site's address | `Use a picture uploaded to the shop.` |
| `sizes`, `pairs`, `seen_pairs` | §5 | §5 |
| read-only | `sizes_low`, `sizes_out`, `stock`, `rating`, `review_count`, `_source`, `_invented` | `The shop works this out; it cannot be set.` |

**`lines`** — required `name`: `name` text; `blurb`, `fit_note` long text; `since` a whole number 1 to 9999 or `null` (`First made must be a year from 1 to 9999.`).
**`categories`** — required `name`: `name` text; `blurb` long text; `image` a picture as above.
**`reviews`** — required `product`, `name`:

| Field | Rule | Sentences |
|---|---|---|
| `product` | the id of an existing shoe, **required**; may be a draft shoe | `Shoe is needed.` · `No such shoe.` |
| `name` | text, required (the customer) | `Customer is needed.` |
| `rating` | whole number 1 to 5, or `null` / empty | `Stars must be a whole number from 1 to 5.` — so `6`, `0`, `-1`, `2.5` and `"x"` each name `rating` |
| `date` | a real calendar date `YYYY-MM-DD`; default today (Vietnam) | `Date must be a date written YYYY-MM-DD.` |
| `size_bought` | text | |
| `fit` | `""`, `small`, `true`, `large` | `How it fit must be small, true or large.` |
| `body` | long text | |
| `status` | `live` or `hidden`; default `live` | `Status must be live or hidden.` |

`Label` is the schema's own label for the field (`Name`, `Price`, `Customer`, `Stars`, `Main colour`). A name of exactly 200 characters is accepted, one of 201 is refused (SC-7).

### 3.4 Errors of the write routes (the project catalog, nothing new)

| Code | HTTP | Trigger |
|---|---|---|
| `bad_request` | 400 | The body is not one JSON object (§3.1 step 5) |
| `unsupported_media_type` | 415 | Not JSON (upload: §10) |
| `payload_too_large` | 413 | Body over 256 KiB (upload: 5 MiB + 64 KiB) |
| `unauthenticated` / `forbidden` | 401 / 403 | No session / a customer |
| `not_found` | 404 | `PUT` / `DELETE` / `PATCH` of an id that does not exist or does not match the id pattern; any write to a collection with no write route |
| `invalid` | 422 | Any field rule; `errors` names every field at fault |
| `conflict` | 409 | A delete rule (§7); a count that moved (§5, §8). **A `409` of `PUT /v1/products/{id}` carries `errors`** keyed `pairs.<size>` (the envelope rule of §2.3 gains this one case: `errors` is present on `bad_request`, `invalid`, `stock_conflict`, the sign-up `conflict` and this one) |
| `internal` | 500 | A failed query, a timeout, a deadlock that survived its one retry |

## 4. Ids

| Collection | The shop makes |
|---|---|
| shoe, line, category | The slug of the name: Unicode-decompose (`norm.NFD`), drop the combining marks, `đ` and `Đ` become `d`, lower-case, every run of anything but `a–z` and `0–9` becomes one `-`, trim `-`, at most 80 characters, `item` if nothing is left. Then the **smallest free** of `slug`, `slug-2`, `slug-3` … (taken ids are read in one statement, ERD §5 #4). "Samba OG New Colour" is `samba-og-new-colour`; "Đôi giày" is `doi-giay`. `golang.org/x/text` is already in `go.sum` as an indirect dependency; the PR runs `run_go_mod_tidy` to make it direct |
| review | `r-` and 8 lower-case hex characters from `crypto/rand` |

A `23505` on the key (a second server instance won the race) picks the next suffix or a new hex string, up to 5 times, then `500`. The id never changes; it satisfies the schema's `ck_{table}_id_slug`.

## 5. A shoe's sizes and pairs

The counts live **only** in `product_sizes` (D3). A shoe's `sizes`, `sizes_low`, `sizes_out` and `stock` are read from `product_stock` on every read and are never accepted. Sentences quoted by the story are exact, punctuation included (`Enter the pairs in this size` has no final full stop).

### 5.1 The three body fields

| Field | Type | On |
|---|---|---|
| `pairs` | object `size → whole number` | `POST`, `PUT` |
| `sizes` | array of text, at most 50 | `POST`, `PUT` — an API client's way to list sizes; the frontend never sends it |
| `seen_pairs` | object `size → whole number`: **the counts the form last read** | `PUT` only, and only with `pairs`. **Additive: new in this design** |

On a `POST` there is nothing to have seen: `seen_pairs` is tolerated only as `{}` and is otherwise `422` (`A new shoe has no counts to compare.`). `seen_pairs` without `pairs` is `422` (`Send seen_pairs together with pairs.`).

### 5.2 Reading a size

One pure function, `stock.ParseSize(text)`, is the only reader. White space (any Unicode space) is collapsed to one space and trimmed; the size is then **stored as written**, in one of:

| Form | Examples | Value (hundredths, rounded half up) |
|---|---|---|
| whole | `38`, `40` | 38.00, 40.00 |
| whole and fraction | `36 2/3`, `35 1/2` | 36.67, 35.50 |
| decimal | `35.5`, `36.67` | 35.50, 36.67 |

ASCII digits only; one to three digits before the point or space, a fraction with a denominator from 1 to 99 and a numerator smaller than it, at most two decimals; the value is above 0 and below 1,000; at most 20 characters. Anything else is unreadable. **Two sizes with the same value are duplicates** (`35 1/2` and `35.5`): the primary key is the text, but a shop that lists both would show two pills for one size. A shoe lists at most **50** sizes and a count is at most **1,000,000** (technical bounds that keep `sum(pairs)::integer` of the view safe; not a business ceiling).

| Fault | Key | Sentence |
|---|---|---|
| Unreadable size | `pairs.<size as typed>` (or `sizes`) | `Write the size as a number, such as 38, 36 2/3 or 35.5.` |
| Over 20 characters | same | `A size can have at most 20 characters.` |
| A size twice (same value) | the later one, `pairs.<size>` (or `sizes`) | `This size is listed twice.` |
| More than 50 sizes | `sizes` / `pairs` | `A shoe can list at most 50 sizes.` |
| `sizes` and `pairs` list different sets | `sizes` | `Sizes and pairs must list the same sizes.` |
| A size with no count (absent, `null`, `""`) | `pairs.<size>` | `Enter the pairs in this size` |
| A count that is not a whole number (`2.5`, `"abc"`, `true`) | `pairs.<size>` | `Pairs must be a whole number.` |
| A count below 0 | `pairs.<size>` | `Pairs cannot be below 0` |
| A count above 1,000,000 | `pairs.<size>` | `Pairs can be at most 1,000,000.` |

`sizes` without `pairs`: on a `POST` every size has no count (`422` for each, `Enter the pairs in this size`); on a `PUT` a size the shoe already holds keeps its count, a new size has none (`422`), and a stored size that is not listed is removed. Neither `sizes` nor `pairs` leaves every count and every list untouched, so a price-only `PUT` changes no count (SC-8, SC-24).

### 5.3 What a save does with the counts

The shoe row is locked first (`FOR NO KEY UPDATE`, which waits for any order in flight on the shoe), then every count of that shoe (`FOR UPDATE`, ERD §5 #5). Every size in the union of the stored sizes, `pairs` and `seen_pairs` is then decided by three values: **expected** (`seen_pairs`, absent = the form did not know it), **desired** (`pairs`, absent = take it off) and **actual** (stored, absent = no row).

**With `seen_pairs`** (what the frontend sends when a count or a size was touched):

| Condition | What happens |
|---|---|
| desired equals expected (both absent, or the same number) | **Untouched.** Nothing is written and nothing is compared, even if an order moved the stored count since: the form's old number is never written back. The stored count stands |
| desired differs from expected, and actual equals expected | **Write desired**: insert, update, or delete (a size left out of `pairs`) |
| desired differs from expected, both counts exist, and actual differs from expected | **Conflict**: the count moved since the person looked. `pairs.<size>`: `Pairs were just taken by an order. Check the number and save again.` |
| desired differs from expected, and a row exists where none was expected, or none exists where one was | **Conflict**: `This size was changed by someone else. Reload the page and check the sizes.` |

If **any** size conflicts, nothing from the save is stored — not the other sizes, not the price — and the answer is `409 conflict`, its `message` the sentence of the first conflicting size and its `errors` every conflicting size. Validation faults (`422`) win over conflicts, because a conflict needs the database.

**Without `seen_pairs`** (an API client, a `POST`): `pairs` is authoritative — every listed size is inserted or set to its count and every stored size not listed is removed (`_base.md` §3.22, SC-20, SC-24). There is no guard because the caller said it did not read the counts.

The writes go through `stock.Set` (one upsert statement) and `stock.Remove` (one delete); the handler issues no SQL on `product_sizes`.

### 5.4 The other rules of a shoe save

- **`featured: true`** first locks the shoe and the currently featured shoe in id order (ERD §5 #3), turns the other off, then sets this one, all in the one transaction. `featured: false` clears only this shoe. Nothing stops the shop having no featured shoe.
- **A change of `status`** (the stored value differs from the one sent) recalculates the shop's two figures (ERD §4). A save that sends the same status, a create, and every other field change recalculate nothing. A shoe saved with `status: draft` leaves every visitor read at once and keeps its reviews, counts, orders and id.
- A shoe's `rating` and `review_count` are never accepted and never recalculated by a shoe save (decision E2).
- The answer is the shoe in the **dealer view**, read in the same transaction after the write, so it shows the derived lists as stored.

## 6. Reviews, lines and categories

**Reviews.** The events that recalculate a shoe's and the shop's figures are exactly those of ERD §4: a create, a change of `rating`, `status` or `product` to a different value, and a delete. A review moved to another shoe recalculates both shoes (in id order) and the shop. A `PUT` that changes only the customer's name or text recalculates nothing. A review of a draft shoe may be created, changed and deleted; it is simply outside the shop's figures while the shoe is a draft.

**Lines and categories.** Plain records: no derived value. A rename changes what a shoe's `line` / `category` search finds at once, because the search reads the name live (`data` registry). A shoe's `line` after its line is deleted is not possible: the delete is refused while any shoe uses it (§7).

## 7. Deleting

Each delete runs in one transaction that takes the write lock, then the record `FOR UPDATE` (a missing record is `404`; a second `DELETE` of the same id is `404`).

| Collection | Refused when (`409 conflict`, nothing deleted) | Otherwise |
|---|---|---|
| `products` | `SELECT EXISTS (SELECT 1 FROM order_items WHERE product_id = $1)` — `Orders refer to this shoe. Set it to draft instead.` The sample orders count: a placeholder shoe one of the 14 sample orders names is refused too (SC-14) | `DELETE` (its reviews and counts go by `CASCADE`); if the shoe was `live`, recalculate the shop's figures |
| `lines` | `SELECT count(*) FROM products WHERE line_id = $1` is `n > 0` — `<n> shoes use this. Move them first.` | `DELETE` |
| `categories` | the same count on `category_id` | `DELETE` |
| `reviews` | never | `DELETE`, then recalculate its shoe and the shop (ERD §4) |

`23503` on the `DELETE` (an order or a shoe arrived after the check; impossible under the lock for shoes, possible only through another instance) is answered with the same `409` and its sentence, the count being read again. The sentence keeps the contract's wording for `n = 1` ("1 shoes use this. Move them first."): the contract text is the one tests quote, and improving the grammar is a later wording change, not a reason to differ from `_base.md` §3.22 now.

## 8. `PATCH /v1/products/{id}/sizes` — one size's count

`_base.md` §3.23 stands. This section adds what it leaves open.

**Request** `{"size": "36 2/3", "pairs": 6, "seen_pairs": 4}` — all three required. `pairs` and `seen_pairs`: whole numbers 0 to 1,000,000 (`422` naming the key: `Pairs must be a whole number.` / `Pairs cannot be below 0`). `size` must be text (`422` `size`, `Choose a size.` when absent or not text); white space is collapsed as in §5.2 and the result is matched **exactly** against the stored size. A size the shoe does not hold — including one that does not read as a number — is `404`.

**Order of checks.** Session → content type → body → `{id}` pattern (`404`) → the three fields (`422`, every fault) → the statement.

**The statement** (no write lock: one row, one atomic compare-and-set; ERD §6):

```sql
UPDATE product_sizes SET pairs = $3 WHERE product_id = $1 AND size = $2 AND pairs = $4
```

One row changed: `200`. No row: `SELECT pairs FROM product_sizes WHERE product_id = $1 AND size = $2` — no row means the shoe or the size does not exist (`404`), a row means the count moved (`409 conflict`, `Pairs were just taken by an order. Check the number and save again.`, nothing overwritten). `pairs` equal to `seen_pairs` changes nothing and is `200`. A `PATCH` never changes `rating`, status or any other table, and it recalculates nothing.

**Success** `200` `{"size":"36 2/3","pairs":6,"sizes":[…],"sizes_low":[…],"sizes_out":[…],"stock":18}` — read from `product_stock` after the update; `size` is the stored size. The `409` carries no `errors` and no count: the client reads the shoe again (`GET /v1/products/{id}`, dealer view, `pairs`) and shows the stored number, as the UI already does.

## 9. Packages: what each one gains

Nothing is created outside these files, and no package imports another package's SQL (`_base.md` §1.1). The import graph stays acyclic: `admin → data, stock, auth, httpx`; `data → auth, httpx`; `stock → pgx` only.

### 9.1 `internal/data` — the two views (reads)

| Change | Detail |
|---|---|
| `type View` (`Public`, `Dealer`) and `ViewOf(ctx)` | `Dealer` when `auth.FromContext` holds role `admin` or `staff`, else `Public`. Read once per request by the registry's two handlers |
| `Collection.Live string` | The condition that limits a `Public` caller to what is on sale or shown, written over the alias `t`: products `t.status = 'live'`; reviews `t.status = 'live' AND EXISTS (SELECT 1 FROM products lp WHERE lp.id = t.product_id AND lp.status = 'live')`; empty for every other collection |
| `selectList(view)`, `where(…, view)`, `recordSQL(c, view)`, `listSQL` | `Public` serves `Access == Public` columns and ANDs `Live` into the one `WHERE` shared by the count and the page; `Dealer` serves `Public` and `Bookkeeping` columns, no `Live`. `Private` is served to neither (orders unchanged) |
| New column of `products`: `pairs` | `Access: Bookkeeping`, new `Kind` `Object`, `Select: (SELECT coalesce(jsonb_object_agg(z.size, z.pairs), '{}'::jsonb) FROM product_sizes z WHERE z.product_id = t.id)`. Not filterable, sortable or searchable (`400` naming it) |
| `ListHandler(db, log, c)` | **Behaviour unchanged**: it is built with the `Public` column set and no `Live`, whatever the session, so `GET /v1/orders` serves what item 9 left (story decision 7). Item 17's design decides the orders' dealer shape |
| Exported `Read(ctx, q Querier, collection, id string, v View) (json.RawMessage, error)` and `ErrNotFound` | The one reader of a record's JSON, so a write's answer is built by the **same statement** as a `GET` and can never differ from it. `q` is a pool or the writer's transaction |
| Tests | `collections_test.go` learns that `pairs` is a dealer-only extra of the registry that `schema.json` does not list (as it already knows the bookkeeping columns). Tests that served as admin only to read the public shape serve as a visitor now. New cases: the draft shoe in list / `q` / `size` / `status=draft` / by id (404 with the missing id's body); the hidden review; a review of a draft shoe; the dealer's `pairs`, `price_listed`, `_source`; a customer session reads as a visitor; `?pairs=1` and `?sort=pairs` are `400` |

`internal/catalogue` is **not** changed (`product-filters` already reads live shoes only). `grep -rn "status = 'live'" internal/` finds only the registry's two `Live` strings and `catalogue`'s `filtersSQL`.

### 9.2 `internal/stock` — the only writer of `product_sizes` (additive, one file)

Item 17 adds `Return` to the same file; the two additions are independent and whichever PR merges second adopts the other's lines.

```go
// Size is a size as the shop reads it: Text is stored as written (white space collapsed), Value is its number with two decimals ("36.67").
type Size struct{ Text, Value string }
var ErrUnreadable, ErrTooLong error
func ParseSize(raw string) (Size, error)                       // §5.2, pure

type Count struct { Size Size; Pairs int }
func LockShoe(ctx, tx Tx, product string) (map[string]int, error)       // SELECT size, pairs … WHERE product_id = $1 ORDER BY size FOR UPDATE
func Set(ctx, tx Tx, product string, counts []Count) error              // one INSERT … ON CONFLICT (product_id, size) DO UPDATE SET pairs
func Remove(ctx, tx Tx, product string, sizes []string) error            // one DELETE … size = ANY($2)
func SetOne(ctx, tx Tx, product, size string, pairs, seen int) (bool, error) // UPDATE … WHERE pairs = $seen; true when one row changed
func Pairs(ctx, tx Tx, product, size string) (pairs int, found bool, err error) // the 404-or-409 read after a failed SetOne
func Lists(ctx, tx Tx, product string) (Derived, bool, error)           // sizes, sizes_low, sizes_out, stock from product_stock
```

`Lock`, `Take` and the lock order are unchanged. `LockShoe` orders by `size` in SQL exactly as `Lock` does, so `Place`, item 17's `Cancel` and a save agree.

### 9.3 `internal/admin` — the writers (new files; `staff*.go` are not touched)

| File | Holds |
|---|---|
| `records.go` | `RegisterRecords(mux, db Pool, log)`; the twelve routes, the `PATCH`; the pipeline of §3.1; reuses the package's `reply`, `refuse`, `notFound`, `invalid`, `Pool` from `staff*.go` rather than copying them |
| `records_spec.go` | The four collections as Go tables: field, kind, label, required, clearable, length, closed set, column name — the **only** place a column name for an `INSERT` / `UPDATE` comes from. `records_spec_test.go` reads `../../../frontend/src/data/schema.json` (skipped when absent, as `collections_test.go` does) and fails when a schema field is neither in a table nor listed as derived |
| `records_input.go` | Body reading and every judge of §3.2–§3.3 and §5.2; pure, table-tested |
| `records_slug.go` | The slug and the free-id choice of §4 |
| `records_write.go` | One function per collection and verb: the transaction of §3.1 steps 9–11, the retry of `40P01` / `40001` once and of `23505` as §4 says |
| `records_figures.go` | `recomputeShoe`, `recomputeShop` (ERD §4) — nothing else writes `rating` or `review_count` |
| `records_counts.go` | §5.3 over `stock` |
| `uploads.go` | `POST` and `GET` of §10 |

Each write logs **one line** at `info`: `catalogue write`, `collection`, `action` (`create`, `update`, `delete`, `count`, `upload`), `id`, and the request id the access log already carries; never a field value, a body or a file name.

`server.New` adds `admin.RegisterRecords(mux, pool, log)`. `go.mod` makes `golang.org/x/text` a direct requirement (`run_go_mod_tidy`).

## 10. Pictures: `POST /v1/uploads` and `GET /v1/uploads/{name}`

`_base.md` §3.24 and §3.6 stand. Added here:

**Order of checks** (so the answer for a file that breaks two rules is fixed): session (`401` / `403`) → `Content-Type` is `multipart/form-data` with a boundary (`400`) → the request is wrapped in `http.MaxBytesReader` at 5 MiB + 64 KiB (`413` when it bursts) → the first part is named `file` (`400`) → **an empty file is `422`** (`errors.file` `Choose a picture.`) → the content is judged from its first 512 bytes with `http.DetectContentType` and **only** `image/jpeg`, `image/png`, `image/webp` pass (`415`, `Choose a JPEG, PNG or WebP picture.` — the file's name, extension and declared type are never read) → the rest is read to 5 MiB + 1 byte and **a file over 5,242,880 bytes is `413`** (`That picture is too large (5 MB at most).`; exactly 5,242,880 is accepted) → any further part is `400` → store. So a 6 MB GIF is `415`, a 6 MB JPEG `413`, a renamed text file `415`, a real PNG called `x.txt` `201`.

**Store.** The name is 16 bytes of `crypto/rand` as 32 lower-case hex characters plus `.jpg`, `.png` or `.webp` from the **detected** type; the sent file name is discarded, so `../../x.png` can name nothing and nothing is written anywhere but `uploads`. One `INSERT INTO uploads (name, content_type, size_bytes, data, uploaded_by)`; `uploaded_by` is the signed-in user's id; a `23505` on `uq_uploads_name` draws a new name (3 tries). `201`, `Location: /api/v1/uploads/<name>` and `{"url": "/api/v1/uploads/<name>"}` (the address is browser-facing; the route that serves it has no prefix).

**Limits.** Deadline 30 s for this route (the project's one exception to 10 s, `_base.md` §10); the file is held in memory, at most 5 MiB + 1 byte per request, and the route is dealer-only. A refusal stores nothing and leaves nothing behind.

**Serve.** `GET /v1/uploads/{name}` matches `^[a-f0-9]{32}\.(jpg|png|webp)$` before any query (`404` otherwise and for an unknown name), one `SELECT content_type, size_bytes, data FROM uploads WHERE name = $1`, `200` with the bytes, `Content-Type` as stored, `Content-Length`, `Cache-Control: public, max-age=31536000, immutable`, `X-Content-Type-Options: nosniff`. Public, no session read. Nothing deletes a picture, whether its shoe is deleted or its picture replaced (story §4).

## 11. The frontend at the swap (the backend PR's frontend half)

| Change | Where |
|---|---|
| `patch(path, body)` and `upload(path, file)` are added to `src/lib/api.ts`: `patch` is `put` with the verb changed; `upload` sends a `FormData` with one part `file`, **no** `Content-Type` header (the browser adds the boundary), the route's own 30 s abort, **no retry**. Nothing existing in the file changes; none of these writes is ever retried automatically | `src/lib/api.ts` and its test file |
| `transport` points at `api`: `create` → `post(collection, body)`; `update` → `put(`${collection}/${id}`, body)`; `remove` → `delete(`${collection}/${id}`)`; `read` → `get(`${collection}/${id}`)` (the dealer view, because the cookie is sent); `setPairs` → `patch(`products/${id}/sizes`, { size, pairs, seen_pairs })`; `upload` → `upload("uploads", file)`. An `ApiError` becomes `{ status, body }` with `body` the envelope; **no answer (network, the abort) is status 0**. A `204` has no body. 401 and 403 are "session ended" (a customer never reaches these screens) | `src/lib/manage-shoes-lines-categories-reviews-and-orders.ts` |
| Delete the mock; repoint the `vi.mock` of `src/test/admin*.test.ts(x)` to `../lib/api` keeping the real `ApiError` | `src/lib/mock/…`, tests |
| `payload.ts` sends `seen_pairs` (every size the form **read**, size → count) with `pairs` (every size the form holds) **only when a size or a count was touched**, on a `PUT`; on a `POST` it sends `pairs` and **no `seen_pairs`** (or `{}`); it never sends `id`, `sizes` or a derived field; a size that was never read and is left blank is **not sent** as `0` (the UI review's should-fix 3) | `components/admin/payload.ts` |
| "Save changes" waits while a count row is saving (should-fix 2), and `Choice` / `RefControl` carry `aria-invalid` / `aria-describedby` (should-fix 1) — both carried over from the UI review | `RecordForm.tsx`, `StockCounts.tsx`, `Choice.tsx`, `Form.tsx` |
| No new route, no `content.json`, `schema.json`, `theme.css` or `App.tsx` change. The words stay in the admin components | — |

## 12. The mock, read and judged

**Evidence read, stated plainly.** The UI PR's mock module and client are in its file list but their bodies are beyond what the PR's diff returns, and this task may not stand on the UI branch to open them. This design therefore judges the shapes the frontend **renders and sends**, from what the PR states and the UI review recorded (the `transport` methods and their `{status, body}` answers, the five assumptions of the PR, the table of mock answers) and from the merged components visible in the diff (`RecordForm`, `SaveNote`, `PictureControl`, `DeleteQuestion`, `ListTable`, `RecordDialog`). **Where the mock and this file differ, this file wins**, and the BE PR lists every difference it finds when it swaps the transport.

| What the frontend assumes | Judgment | Frontend change |
|---|---|---|
| `seen_pairs` next to `pairs`, sent only when a size or count was touched; `pairs` then carries every size | **Accepted, with the server rule of §5.3** (an untouched size keeps its stored count; a touched one must match what was seen) | `POST` never sends a non-empty `seen_pairs`; a never-read blank is not sent (§11) |
| A `409` that carries `errors` (`pairs.<size>`) is shown beside that size, a `409` with only a sentence as a sentence | **Sound.** The server sends exactly this (§3.4, §5.3) | None |
| One size's `409`: the client reads the shoe again and takes `pairs[size]` from the dealer view | **Sound.** The dealer view carries `pairs` (§2); the `409` needs no extra field (§8) | None |
| `PUT` sends changed fields only: `null` for a cleared number, `""` for cleared text, line, category or picture; `POST` sends what was filled in plus `status`; neither sends `id`, a derived field or `sizes` | **Sound** and exactly what §3.2 reads | None |
| A save with nothing changed sends no request | **Sound**; the server would also answer `{}` with `200` | None |
| `403` is "session ended" | **Acceptable**: the server answers `403` only to a customer session | None |
| The answer of `PATCH` is read as `size`, `pairs`, `sizes_low`, `sizes_out`, `stock` | **Sound**; §8 also returns `sizes` | None |
| Today's shoes carry no `pairs` | Right for today's API; the dealer view of §2 supplies it | The Stock tab shows real counts once the swap is made |
| Uploads: `415` / `413` sentences, "We could not upload that picture. Please try again." when there is no answer | **Sound**; the sentences are the story's | None |
| The record page draws nothing while a record loads or when the shop cannot be reached | A UI decision the UI review accepted | None |

## 13. Decisions, and conflicts with merged ones

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| D1 | Where counts live | Only `product_sizes`, written by `stock`; every list derived from the view | A typed copy on `products`: it would drift from what an order takes (ADMIN-015 item 1) |
| D2 | How the two views are chosen | By the **session's role** alone (`_base.md` §2.4, SC-11) | `?view=dealer` or `/v1/admin/products`: changes merged contract and the approved admin pages, which read `useCollection`. Cost accepted: a signed-in dealer's browser sees drafts on customer pages; checks of what a visitor sees are made with no session |
| D3 | Does a visitor read a live review of a draft shoe | **No** (§2) | Review status alone: a draft shoe's id and what people said about it would be public, and the list would disagree with the shop's own figure, which already excludes them |
| D4 | How a whole-form save avoids overwriting an order's pairs | `seen_pairs`, compare-and-set per touched size (§5.3) — the additive guard story §8 note 1 asked for | A `version` column on `products` with `If-Match`: a migration, and it would make every non-count field "last write wins" no longer true (SC-9); a `409` on any stale form would refuse a price edit because an order took a pair |
| D5 | Ratings on a shoe save | Never (ERD E2) | Recalculate on every save: contradicts SC-8 and SC-13 |
| D6 | A 409 body | `PUT /v1/products/{id}`'s `409` carries `errors` | A message only: the form could not put the sentence beside the size |
| D7 | Empty / type / size order of an upload | empty → type → size (§10) | Size first: a 6 MB GIF would be read in full to be told it is too large rather than the wrong type |
| D8 | Duplicate sizes | By numeric value | By text: `35 1/2` and `35.5` would both be sold as two sizes |
| D9 | Technical bounds | count ≤ 1,000,000, ≤ 50 sizes, price ≤ 999,999,999,999 | None: without them a count of 2,000,000,000 makes `sum(pairs)::integer` of the view fail and the shoe unreadable |
| D10 | Pictures on a record | Only `""`, `/images/…` or an upload that exists (§3.3) | Any string: `data:` text and dangling addresses (SC-35 forbids the first) |
| D11 | Slug in Go, not in SQL | `norm.NFD` + marks dropped (§4) | `doi_fold`: it folds only Vietnamese letters, so `Müller` would become `m-ller` |
| D12 | The write lock | One advisory lock (ERD E1) | Row locks only |
| D13 | `ListHandler` | Pinned to the public column set (§9.1) | Letting it follow the session: orders' `_source` / `_invented` would appear in `GET /v1/orders`, which item 9 and the story say stays as it is |

**Conflicts with merged decisions: none.** §3.22 stands (only additions: `seen_pairs`, the `409`'s `errors`, bounds, the sentences); §2.4 stands (the views are now real; D3 narrows reviews to what the shop's own figure already uses); §3.23 and §3.24 stand.

## 14. Migration and rollout

Forward: migration `uploads` (ERD §7), then the code; the server applies it before it listens, so no request sees a route without its table. Backward: `DROP TABLE uploads` for a local reset; rolling the **code** back to the previous build is safe because it simply has no write routes (the table and its pictures stay). Populated database: the table is new and empty; nothing else in the schema changes. No seed, no setting, no secret.

## 15. Where each criterion is enforced, and notes for Dev and Test

| Criteria | Enforced by |
|---|---|
| SC-1, SC-2, SC-3 | `auth.Require` on every pattern; no pattern for the five read-only collections (§1); the catch-all's `404` |
| SC-4 | §4 (ids), `Location`, `data.Read` in the same transaction (§3.1 step 11) |
| SC-5, SC-6, SC-7 | §3.2–§3.3 (all faults collected; unknown and derived keys; whole numbers; lengths in runes; `""` unsets) |
| SC-8, SC-9, SC-10 | §3.2 (partial `PUT`, `id` discarded, `null`), §5.3 (no counts sent, none changed), §5.4, ERD §6 |
| SC-11 | §2, §9.1 |
| SC-12, SC-13 | ERD §4 (the two statements and the event table) |
| SC-14, SC-15, SC-16 | §7 |
| SC-17, SC-18, SC-19 | §10 |
| SC-20, SC-22 | §5.1–§5.2 |
| SC-21, SC-23 | §8 (the `PATCH`); §5.3 (the whole-form guard) |
| SC-24 | §5.2 last paragraph, §5.3 |
| SC-25 | §9.2 (`stock.Take` and the counts are the same rows) |
| SC-26 to SC-38 | The UI PR, §11, §12 |
| SC-39 | ERD §5 (about ten statements, all by key), ERD §6 |

**Notes for the BE step.**

1. Cut the BE branch from the UI branch (`feature/manage-shoes-lines-categories-reviews-and-orders-ui`), since it deletes the mock that branch adds; PR #79 merges first. Grep the head for `^<<<<<<<` and `^>>>>>>>` before merge.
2. Order of work: `stock` functions and `ParseSize` → `data` views and their tests → `admin` input and slug (pure) → the transactions → uploads → `api.ts` and the frontend swap.
3. Plan item 17 (PR #77) edits `stock.go`, `server.go` and `internal/orders` in parallel: each is an additive edit; the second to merge adopts the other's lines.
4. Orders: the list's row menu goes through `store.ts`, which posts `PUT` / `DELETE` to `/v1/orders/{id}`. Those routes are item 17's; until they exist they answer `404`.
5. Database tests (`TEST_DATABASE_URL`) first run in CI. The seeded shoes have at most 3 pairs per size; tests that need more set them with plain SQL.

**Notes for Test.** SC-25 needs the checkout (`POST /v1/orders`) with a size holding 1 pair. SC-11, SC-26 and SC-27's customer-side halves are checked with **no session**. Sentences the story quotes are asserted exactly; the others are asserted by key, status and "short English".

## 16. Open questions

None blocks the BE step; each has its default.

| # | Question | Owner | Default |
|---|---|---|---|
| Q1 | A signed-in dealer sees drafts on the customer pages of their own browser (D2). Does the owner want a "preview" that is not the default? | PM / owner | As designed; a preview toggle is a later item |
| Q2 | "1 shoes use this. Move them first." reads badly for one shoe (§7) | PM | The contract's wording stands |
| Q3 | The review form's shoe picker lists the first 100 shoes (§2) | PM | Accepted at launch |
| Q4 | The first review event on a placeholder replaces its seeded rating (and, on the first status change, the shop's 4.8 from 181) with the true figure from the review rows the seed holds | PM | As ADMIN-013 item 4 says |
