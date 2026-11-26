# ERD — Manage shoes, lines, categories, reviews and orders (plan item 10, "S9")

Module `admin` · Requirements ADMIN-010, ADMIN-011, ADMIN-012, ADMIN-013, ADMIN-014, ADMIN-015, ADMIN-004 (write half) · Story `docs/admin/stories/manage-shoes-lines-categories-reviews-and-orders.md` · UI PR #79 · Engine PostgreSQL 16, pgx v5, hand-written SQL.

The order half of the original item (moving an order along, cancelling, deleting an order) is plan item 17 and is **not** designed here. This story writes `products`, `product_sizes`, `lines`, `categories`, `reviews`, `shop_terms` (two figures) and `uploads`, and nothing else.

## 1. Verdict: one new table, no change to any existing one

| Change | What |
|---|---|
| **New table** | `uploads` (project ERD §3.20, migration 10 `uploads`). The columns, checks and the `bytea` decision are already written there and are not restated; §6 gives the exact DDL the migration must hold so that nobody has to re-derive it |
| **Existing tables** | **No column, constraint, index, trigger or view is added, changed or dropped.** Every rule this story enforces already has its backstop in the schema (§3) |
| **Seed / data migration** | None. The twelve placeholder shoes, their reviews, their ratings and the shop's 4.8 from 181 reviews stay exactly as seeded until an event of §4 recalculates them (story decision 6) |
| **Indexes** | None added. Every new statement is by primary key, by a unique key or by an index that exists (§5) |

**Why nothing else is needed.** The counts live only in `product_sizes` (D3); a shoe's `sizes`, `sizes_low`, `sizes_out` and `stock` are read from the view `product_stock` and are never stored. A shoe's `rating` and `review_count` and the shop's two figures are stored columns the project ERD already declares as "recalculated in the same transaction as every review change" (§3.3, §3.6); this story is the code that does it. A second copy of stock would need a column; there is none and none is added.

## 2. Who writes what

Each row is **the only code allowed to write that table**, matching the ownership table of `services/_base.md` §1.1.

| Table | Writer | Statements this story adds |
|---|---|---|
| `products` | `internal/admin` | `INSERT`; `UPDATE` of the columns the body carries (column names come from a Go allow-list, never from the request); `UPDATE … SET featured = false` on the previously featured shoe; `UPDATE … SET rating, review_count` (recalculation); `DELETE` |
| `lines`, `categories` | `internal/admin` | `INSERT`, `UPDATE`, `DELETE` |
| `reviews` | `internal/admin` | `INSERT`, `UPDATE`, `DELETE` |
| `shop_terms` | `internal/admin` | `UPDATE … SET rating, review_count WHERE id = 'shop'` — two columns, nothing else |
| `product_sizes` | **`internal/stock`**, called by `internal/admin` | `Set` (insert or update counts), `Remove` (delete sizes), `SetOne` (compare-and-set of one count). `internal/orders` keeps `Take` and, from plan item 17, `Return` |
| `uploads` | `internal/admin` | `INSERT` (never `UPDATE` or `DELETE`) |
| `order_items`, `orders` | — | **Read only**: `SELECT EXISTS (SELECT 1 FROM order_items WHERE product_id = $1)` before a shoe is deleted |

`internal/stock` is not given a second owner: `admin` never issues SQL against `product_sizes`; it calls the three functions of `services/manage-shoes-lines-categories-reviews-and-orders.md` §9.

## 3. What the schema already enforces, and how each failure is read

| Rule of the story | Enforced by | SQLSTATE the code maps |
|---|---|---|
| A price is never below 0 | `ck_products_price_not_negative` | `23514` is a bug (the API refused it first with 422): logged, answered `500 internal` |
| A count is never below 0, even if every other layer fails | `ck_product_sizes_pairs_not_negative` | same |
| One shoe at a time opens the home page | `uq_products_featured` (partial unique). The handler clears the old one first; the index is the backstop | `23505` on `uq_products_featured`: the whole transaction is retried once (it cannot happen under the write lock of §4; the retry is for a second server instance) |
| An id is unique | primary keys of the ten tables | `23505` on `<table>_pkey`: the id is generated again (next suffix, or new hex) up to 5 times, then `500` |
| A line or category used by a shoe is not deleted | `fk_products_line_id`, `fk_products_category_id` `ON DELETE RESTRICT` | `23503` on delete: re-count, answer `409` with the sentence |
| A shoe an order refers to is not deleted | `order_items.product_id → products(id)` `RESTRICT` | `23503` on delete: answer `409` with the sentence (this is the backstop; the handler checks first, under the lock) |
| A review dies with its shoe; a shoe's counts die with it | `reviews.product_id` and `product_sizes.product_id` `ON DELETE CASCADE` | — |
| A review points at a shoe, a shoe at a line and category | the foreign keys | `23503` on insert or update: the handler checked first; if it fires, answer `422` naming `product` / `line` / `category` |
| A closed set (status, colour, fit) | `ck_products_status`, `ck_products_colour`, `ck_reviews_status`, `ck_reviews_fit` | `23514` is a bug: `500` |
| A review's stars are 1 to 5 or empty | `ck_reviews_rating_range`, nullable `rating` | same |
| A required name is not blank | `ck_products_name_not_blank`, `ck_lines_name_not_blank`, `ck_categories_name_not_blank`, `ck_reviews_name_not_blank` | same |
| An upload's name, type and size | `ck_uploads_name_format`, `ck_uploads_content_type`, `ck_uploads_size_range` | same |

Lengths (200, 5,000), the date format of `released` and `date`, the number of sizes and the size of a count have **no** `CHECK`: the project convention (§2.1 of the ERD) keeps them in the API, because seeded text may predate them. Two technical bounds exist only to keep the view's `sum(pairs)::integer` and `bigint` money safe and are API rules, stated in the services file §3: a count is at most 1,000,000, a shoe lists at most 50 sizes, a price is at most 999,999,999,999.

## 4. Derived figures: the exact statements

These are written once, in `internal/admin`, and every event below calls the same two functions.

**A shoe's own figures** (`recomputeShoe(tx, id)`):

```sql
UPDATE products p
SET rating = coalesce(r.mean, 0), review_count = coalesce(r.n, 0)
FROM (SELECT round(avg(rating)::numeric, 1) AS mean, count(*)::integer AS n
      FROM reviews WHERE product_id = $1 AND status = 'live') r
WHERE p.id = $1
```

`avg` ignores a review with no stars; `count(*)` counts every live review (ADMIN-013 item 2: "its review count is the number of its live reviews"). `round(numeric, 1)` rounds half away from zero: 5, 4, 4 is 4.3; 4, 4, 5, 4 is 4.3. A shoe whose live reviews all have no stars has `rating` 0 and a non-zero count.

**The shop's figures** (`recomputeShop(tx)`):

```sql
UPDATE shop_terms t
SET rating = coalesce(r.mean, 0), review_count = coalesce(r.n, 0)
FROM (SELECT round(avg(v.rating)::numeric, 1) AS mean, count(*)::integer AS n
      FROM reviews v JOIN products p ON p.id = v.product_id
      WHERE v.status = 'live' AND p.status = 'live') r
WHERE t.id = 'shop'
```

Zero rows updated means the seed is missing: an error, logged, `500`.

**The events**, and nothing else, call them (decision E2 explains the shoe-save wording of ADMIN-013 item 2):

| Event | `recomputeShoe` for | `recomputeShop` |
|---|---|---|
| A review is **created** | its shoe | yes |
| A review's `rating`, `status` or `product` **changes** to a different value | its old shoe and its new shoe (in id order; one shoe when `product` did not change) | yes |
| A review is **deleted**, whatever its status | its shoe | yes |
| A shoe's `status` **changes** (`live` ↔ `draft`) | — (its own reviews did not change) | yes |
| A live shoe is **deleted** | — (its reviews and counts go with it) | yes |
| A shoe is created; a shoe is saved with the same status; a review's `name`, `body`, `date`, `fit`, `size_bought` change | — | — |

A seeded placeholder rating therefore stays until one of these events touches that shoe (ADMIN-013 item 4, SC-13), and a price-only save changes nothing else (SC-8).

**The visitor's view and the dealer's `pairs`** are read statements, not schema; they are specified in the services file §2 and use only `products.status`, `reviews.status`, `reviews.product_id` and `product_sizes`.

## 5. Access patterns

| # | Pattern | Frequency | Index used |
|---|---|---|---|
| 1 | The write lock: `SELECT pg_advisory_xact_lock(hashtextextended('admin:records', 0))` — first statement of every `POST`, `PUT` and `DELETE` on the four collections | Per dealer write (a few a minute at most) | none (a lock, not a row) |
| 2 | Lock one shoe: `SELECT status, featured FROM products WHERE id = $1 FOR NO KEY UPDATE` (`FOR UPDATE` before a delete) | Per shoe write | `products` primary key |
| 3 | The shoe to turn off when another is featured: `SELECT id FROM products WHERE id = $1 OR featured ORDER BY id FOR NO KEY UPDATE`, then `UPDATE products SET featured = false WHERE featured AND id <> $1` | Rare | `uq_products_featured` (the partial index holds the one featured row); the primary key |
| 4 | A free slug: `SELECT id FROM <table> WHERE id = $1 OR id LIKE $2` (`$2` is the slug plus `-%`; the slug holds only `a–z`, `0–9`, `-`, so no escaping) | Per create | primary key; the table is hundreds of rows, so a prefix scan is a few microseconds and no `text_pattern_ops` index is added |
| 5 | The shoe's counts under lock: `SELECT size, pairs FROM product_sizes WHERE product_id = $1 ORDER BY size FOR UPDATE` | Per shoe `PUT` that carries `pairs` | `product_sizes` primary key prefix |
| 6 | Write counts: one `INSERT … SELECT … FROM unnest(…) ON CONFLICT (product_id, size) DO UPDATE SET pairs = EXCLUDED.pairs` and one `DELETE … WHERE product_id = $1 AND size = ANY($2)` | Per shoe `PUT` / `POST` that carries `pairs` | primary key |
| 7 | One size: `UPDATE product_sizes SET pairs = $3 WHERE product_id = $1 AND size = $2 AND pairs = $4` (compare-and-set), then on zero rows `SELECT pairs FROM product_sizes WHERE product_id = $1 AND size = $2` to tell 404 from 409, then `SELECT sizes, sizes_low, sizes_out, stock FROM product_stock WHERE product_id = $1` | Per count edit on the shoe page | primary key |
| 8 | Does an order refer to this shoe: `SELECT EXISTS (SELECT 1 FROM order_items WHERE product_id = $1)` | Per shoe delete | `idx_order_items_product_id` |
| 9 | How many shoes use this line / category: `SELECT count(*) FROM products WHERE line_id = $1` | Per line / category delete | `idx_products_line_id`, `idx_products_category_id` |
| 10 | Do these addresses name uploaded pictures: `SELECT name FROM uploads WHERE name = ANY($1)` (at most three names) | Per shoe or category save that carries an `/api/v1/uploads/…` address | `uq_uploads_name` |
| 11 | Recalculation (§4): the shoe's live reviews; the shop's live reviews of live shoes | Per review event | `idx_reviews_product_id_date`; the shop figure scans `reviews ⨝ products` (dozens to hundreds of rows) |
| 12 | The dealer's read of a shoe or a list adds `pairs`: `(SELECT coalesce(jsonb_object_agg(z.size, z.pairs), '{}'::jsonb) FROM product_sizes z WHERE z.product_id = t.id)` per returned row | Every admin list page (at most 100 rows) | `product_sizes` primary key prefix |
| 13 | The visitor's read adds `t.status = 'live'` (shoes) or `t.status = 'live' AND EXISTS (SELECT 1 FROM products p WHERE p.id = t.product_id AND p.status = 'live')` (reviews) to the condition of the one list statement | Every public page | `idx_products_status_released_seq` serves the shoes' default order exactly as before; reviews are a scan of dozens of rows |
| 14 | Store and serve a picture: one `INSERT INTO uploads …`; one `SELECT content_type, size_bytes, data FROM uploads WHERE name = $1` | Per upload; per picture request (long cache headers keep it off the database) | `uq_uploads_name` |

No statement is a join of more than two tables and none is unbounded: every list is paged (`limit` at most 100), and the only scans are over the shop's own hundreds of rows.

## 6. Concurrency: what is locked, in which order, and what each race ends as

**One lock order for the whole project, so nothing here can deadlock with another story:**

1. (writers of this story only) the write lock, `pg_advisory_xact_lock(hashtextextended('admin:records', 0))`;
2. `products` rows, **in id order** — `FOR NO KEY UPDATE` here, `FOR SHARE` in `Place`;
3. `product_sizes` rows, **in `(product_id, size)` order**, `FOR UPDATE`;
4. `reviews` rows;
5. `shop_terms`.

`orders.Place` takes (its own advisory key) → products `FOR SHARE` by id → counts `FOR UPDATE` by `(product_id, size)` → counter → inserts (checkout design, E3/E4). Plan item 17's `Cancel` takes the order row, then counts in the same order. Neither takes the write lock of step 1, and this story never takes an `orders` row, so a cycle would need a statement that holds a count and then asks for a product, which no one issues.

**Why a write lock at all.** Without it a review save, a status save and a featured save would each have to reason about two others over `reviews`, `shop_terms` and the one featured row. One advisory lock makes every dealer write of the four collections a queue of one: the recalculations of §4 always read a settled state (ADMIN-013 "two reviews change at once"), two saves that both set `featured` cannot interleave (SC-10), and two `PUT`s of one price both succeed with the later winning (SC-9). The cost is that two staff saves in the same instant wait for each other for the few milliseconds a save takes; at a handful of saves a minute this is invisible, and a save still answers within 1 s at p95 (SC-39). The lock is transaction-scoped, so a crash or rollback releases it. A transaction that fails with `40P01` or `40001` is retried once as a whole.

| Race | Ends as |
|---|---|
| Two `PUT`s of one shoe's price | Both `200`; the second to take the lock wins; the record is whole |
| Two saves that turn `featured` on, on two shoes | Both `200`; the later one is the only featured shoe; `uq_products_featured` is never violated |
| A staff save of counts and an order for the same shoe | They meet at the product row (`FOR NO KEY UPDATE` against `FOR SHARE`): one waits for the other. If the order commits first, the save's `seen_pairs` no longer matches and the save is `409` (SC-23); if the save commits first, the order takes its pairs from the new counts |
| A count edit (`PATCH`) and an order for the same size | They meet at the `product_sizes` row. The order's `UPDATE … WHERE pairs >= $q` and the edit's `UPDATE … WHERE pairs = $seen` are each atomic; whichever runs second re-reads the committed row. A count never goes below 0 and an order's pairs are never overwritten |
| A count edit and the shoe being deleted | The delete holds the product row and cascades to the counts; the edit then finds no row and answers `404` |
| A shoe delete and an order placed at the same moment | The delete waits for the order's `FOR SHARE` on the product; once the order commits, the "does an order refer to it" check sees it and answers `409`. `order_items` `RESTRICT` is the backstop |
| Two creates of the same name | Serialised by the lock; the second gets `-2`. A second server instance racing the first hits `23505` on the key and takes the next suffix (up to 5 times) |
| A review save and a shoe going to draft | Serialised by the lock; the shop's figures are the figures of the final state |

## 7. Migration plan

One migration, `20261126…_uploads` (the timestamp is chosen by Dev when the PR is cut, later than the newest file on `main`; the name fixes the order inside this story). It is migration 10 of the project table (§8).

**Forward** (`.up.sql`, one transaction — there is no `CONCURRENTLY`):

```sql
CREATE TABLE uploads (
    id           uuid        NOT NULL DEFAULT gen_random_uuid(),
    name         text        NOT NULL,
    content_type text        NOT NULL,
    size_bytes   integer     NOT NULL,
    data         bytea       NOT NULL,
    uploaded_by  uuid        NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uploads_pkey PRIMARY KEY (id),
    CONSTRAINT uq_uploads_name UNIQUE (name),
    CONSTRAINT fk_uploads_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT ck_uploads_name_format CHECK (name ~ '^[a-f0-9]{32}\.(jpg|png|webp)$'),
    CONSTRAINT ck_uploads_content_type CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),
    CONSTRAINT ck_uploads_size_range CHECK (size_bytes BETWEEN 1 AND 5242880)
);
ALTER TABLE uploads ALTER COLUMN data SET STORAGE EXTERNAL;
```

No `updated_at` and no trigger (a row is immutable, §3.20). `users` exists (migration 7, merged).

**Backward** (`.down.sql`): `DROP TABLE uploads;`. The server never runs it. Running it orphans every picture address held on a shoe or a category, which is why it exists for a local reset only.

**Safe on a populated database.** The table is new and empty; the only lock taken on an existing table is the brief `SHARE ROW EXCLUSIVE` the foreign key takes on `users`. No existing row is read, rewritten or revalidated. The migration runs before the server listens, so no request ever sees the new routes without the table. Application code of this story has **no** schema dependency on any other pending migration: it reads `product_sizes` and `product_stock` (migration 4, merged) and `users` (merged), and does not need `orders.account_id` (plan item 15) or the payment tables (plan item 8).

**Test data.** The seed has at most 3 pairs per size (0 in each sold-out size, 1 in each last-pairs size). A test that needs more sets them with plain SQL on a shoe it creates, as the checkout tests do.

## 8. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| E1 | How are dealer writes kept from interleaving | One transaction-scoped advisory lock, first statement of every write (§6) | Row locks only: three tables (`reviews`, `products`, `shop_terms`) and the one featured row would each need their own order and retry rule, and the recalculations could read a half-settled state |
| E2 | ADMIN-013 item 2 says a shoe's rating is recalculated "when its shoe is saved"; SC-8 says a price-only save changes nothing else and SC-13 says a seeded rating stays until a review event | **A shoe save never recalculates that shoe's own figures** (they depend only on its reviews, which a shoe save does not touch). A change of `status` recalculates the shop's figures only | Recalculating on every shoe save: the first price edit on a placeholder would silently replace its seeded 4.7 with a figure from the two review rows the seed holds, contradicting SC-8 and SC-13 |
| E3 | A live review with no stars | Counts in `review_count`, adds nothing to the average | Counting only starred reviews: the count would disagree with the number of reviews a visitor can read |
| E4 | Where the counts of one shoe are checked against the form | In the transaction, under the shoe's lock, per size, compare-and-set (services §5) | Comparing in the handler before the transaction: an order could land between the read and the write |
| E5 | `uploads.data` as `bytea` | As the project ERD decided (D10): the container has no volume | Files on disk: forced out by the runtime |
| E6 | Index for `LIKE 'slug-%'` on ids | None | `text_pattern_ops` index: hundreds of rows; the scan is faster than the index's upkeep |
