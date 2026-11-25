# ERD — Product page waits for the shoe before saying it is missing (plan item 13)

> **Revision 2 (the design step ran again after "Site data from the database" merged, PR #39):** re-read against the merged `erd/site-data-from-the-database.md` and `internal/data`; the verdict is unchanged, no schema is added, and §5 is brought up to date.

Module `site` · Requirement SITE-014 (with the not-found row of SITE-005; ADMIN-013 item 1 for the draft case) · Story `docs/site/stories/product-page-waits-for-the-shoe-before-saying-it-i.md` · UI PR #27 · Engine PostgreSQL 16.

## 1. Verdict: this story adds nothing to the schema

**No table, column, constraint, index, trigger, function or migration is created, changed or dropped by this story.** It edits one page (`Product.tsx`) and adds one test file. The page decides what to draw from what `GET /v1/products/{id}` answers, and the answers it branches on already exist.

| What the page needs to know | Where the schema already says it |
|---|---|
| "A live shoe with this id exists" → the record | `products` row with `status = 'live'` (ERD `_base.md` §3.4; the column list is S3's public view) |
| "No live shoe has this id" → 404 | No row, **or** a row with `status = 'draft'`. Both answer the same way (see D1). `ck_products_status` allows exactly `live` and `draft` |
| "The shop could not answer" → neither of the above | Not data: a failed query is `500 internal`, a stopped API is a gateway answer. Nothing is stored for it |

The read this story depends on is a primary-key lookup: `SELECT <public columns> FROM products t WHERE t.id = $1` (S3 services §1.5). The primary key already serves it, so **no index is added**. The later visitor rule (`AND t.status = 'live'`, see services §5) is a filter on that single row and needs no index either.

## 2. Entities this story touches — none changed

| Entity (defined elsewhere) | Effect of this story |
|---|---|
| `products` (`erd/_base.md` §3.4; seeded by S3) | Read by id only, through S3's reader. Not changed. `status` is the one column whose value decides 200 or 404 for a visitor (once ADMIN-013's visitor rule lands) |
| `lines`, `categories`, `shop_terms`, `payment_methods`, `reviews` | Read by the same page through their own S3 requests (`GET /v1/lines/{id}`, `/v1/categories/{id}`, `/v1/shop/shop`, …). Not changed; the page does not wait for them |

## 3. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| D1 | Should the database tell a withdrawn shoe (a draft, or one the dealer deleted) apart from an id that never existed, so the page can say something different? | **No.** A draft and an unknown id answer 404 with one and the same body (`erd/_base.md` §3.4 lifecycle; ADMIN-013 AC-1, AC-11). The page has one missing view and says "That pair is not in the shop any more." for both | A tombstone table, a `deleted_at` column or a `410 Gone` for deleted shoes: it would let any visitor probe which ids ever existed, and tell a stranger that a draft shoe exists before the dealer has published it. The story asks for one missing view, so there is nothing to show for the difference |
| D2 | Does "waiting" or "unreachable" need a stored state (a status flag, a cache table, a last-seen timestamp)? | **No.** Both are client states of one in-flight or failed request. Nothing is written for them and nothing is read from the database to tell them apart | A server-side "is the shop up" row: the page already learns it from the request itself (a refused connection, a gateway 502, a `500 internal`), and a second source could disagree with the first |
| D3 | Does the page need a new column so a shoe can be fetched faster (a slug, a denormalised card row)? | **No.** `GET /v1/products/{id}` is one statement over the primary key. The shoe's own record is the only thing the shoe view waits for; the line, category, terms, payments, reviews and related pairs are separate reads by design (story §3) | A single "product page" join returning everything: it would make the shoe wait for its slowest part, which is the opposite of SC-6 |

### Conflicts with merged decisions

None. I read `erd/_base.md` §3.4 and §3.1, `erd/site-data-from-the-database.md` §7 (decision C1), and `services/_base.md` §2.2–2.3. One interaction is recorded in `services/product-page-waits-for-the-shoe-before-saying-it-i.md` §5 (the draft case waits for ADMIN-013's visitor rule); it is an ordering fact, not a change to any table.

## 4. Migration plan

| Step | Forward | Backward | Safe on a populated table |
|---|---|---|---|
| This story | **No migration file.** Nothing runs against any database | Revert the frontend PR (#27) | Not applicable: no table is touched, no statement is added |

No migration is added and none is edited (convention `migrations`: a merged migration is never edited).

## 5. Hand-over

| Story | What it must know from this design |
|---|---|
| "Site data from the database" (S3) | Merged (PR #39). Nothing to add to the schema; its `useRecord` meets what this page needs (`services/product-page-waits-for-the-shoe-before-saying-it-i.md` §3.1) |
| "Manage shoes, lines, categories, reviews and orders" | `products.status = 'draft'` must answer a visitor's `GET /v1/products/{id}` with the same 404 as an unknown id (services §5). No schema change is needed for it |
| Dev (backend step of this story) | There is no backend work: no table, no endpoint, no migration. The step has nothing to implement |
