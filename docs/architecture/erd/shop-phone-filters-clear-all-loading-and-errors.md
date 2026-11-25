# ERD — Shop phone filters, Clear all, loading and errors (plan item 16)

Module `catalogue` · Requirements CAT-003 items 6–8, CAT-004 in full · Story `docs/catalogue/stories/shop-phone-filters-clear-all-loading-and-errors.md` · UI PR #40 · Engine PostgreSQL 16.

## 1. Verdict: this story adds nothing to the schema

**No table, column, constraint, index, trigger, function or migration is created, changed or dropped by this story.** It is `/shop` in the browser: it changes how the page reads the address bar (`size` is an EU size, not a page size), how it asks again after a failed read, and what it draws while it waits, when nothing matches and when a read failed. Every row it shows is served by plan item 4 from tables that already exist.

| What the page needs | Where the schema already says it |
|---|---|
| Twelve live shoes that match what the visitor set, and the true count | `products` (`erd/_base.md` §3.4) read by the one statement of plan item 4 (`erd/search-filters-and-sorting-served-by-the-api.md` §4, S4-1 and S4-2: `status = 'live'`, any-of `category` / `line` / `colour`, price ranges, `size`, words, `ORDER BY` one field then `seq`). `total` and the page come from one statement, so the count on screen can never disagree with the cards |
| The colour options and the Size (EU) pills | `GET /v1/product-filters`, computed from the live shoes on every request (same file, S4-4 and decision E5). A catalogue with one colour or one size answers a one-item list; the page, not the database, decides that a group of fewer than two options is not drawn |
| Category and line names (the heading, the For and Line groups) | `categories` and `lines`, read through S3's reader. Not changed |
| "No pair matches that." | Not a stored fact: it is `total = 0` in a `200` answer. A filter value that names nothing (`?line=zzz`, `?colour=zzz`, `?size=99`) is an ordinary any-of filter that matches no row, so it is `200` with `items: []`, never an error (S4 services §3.3, §3.7) |

## 2. Entities this story touches — none changed

| Entity (defined elsewhere) | Effect of this story |
|---|---|
| `products` | Read, through item 4's list and `product-filters`. Not changed. The index that serves the shop's default list (`idx_products_status_released_seq`) and every other access pattern are item 4's (S4 §4); this story adds no query |
| `categories`, `lines` | Read by the page for names. Not changed |
| `product_sizes` / `product_stock` (S5) or the S3 size columns | Read by item 4's `size` condition and `product-filters`, in whichever form is on `main`. This story does not care which |

## 3. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| D1 | Does Back need the list to be stored (a "last list" row, a session value, browser storage or history state) so it can return to the same cards? | **No.** The address bar is the whole state (CAT-003 data touched: "nothing stored on the server"). Back re-reads the list from the address and shows the first twelve, which is exactly what CAT-003 item 7 asks. Nothing is written for it | Storing the 24 cards or the scroll position of an earlier "Show more": the SRS asks for the first twelve, the story leaves restoring them out (§5), and a stored list needs a lifetime and a cleanup rule for a promise nobody made |
| D2 | Do "loading" and "could not load" need a stored state (a status flag, a cache table, a last-read timestamp)? | **No.** Both are client states of one request in flight or one request that failed. Nothing is written and nothing is read from the database to tell them apart; the failed read is told from the empty read by the HTTP answer alone (services file §2) | A server-side "is the shop up" row: the page learns it from the request itself, and a second source could disagree with the first |
| D3 | The empty state says "ask us, and we will tell you when it comes in". Does that need a "notify me" table? | **No.** It is the owner's wording, and the page has no such feature (story §5). The contact page and the dock are one tap from every page | A `stock_notifications` table: no consent text, no contact policy, no sending of anything exists, and every address collected would be personal data with no purpose the SRS gives |
| D4 | Is the number in "Filter (n)" or the `<N> matching` line a database figure? | **No.** `n` is counted in the browser from the address (ticked boxes plus pressed size pills); `<N>` is the list answer's `total`. Neither is a column, a view or a second query | A "count of active filters" the API computes: it would be a second reading of what the address already says, and could disagree with the boxes on screen |

### Conflicts with merged decisions

None. I read `erd/_base.md` §3.4, `erd/search-filters-and-sorting-served-by-the-api.md` §1–§5 and `erd/product-page-waits-for-the-shoe-before-saying-it-i.md`. One interaction is recorded in the services file §6: PR #40 and plan item 4's backend PR both edit files of the shop's list; it is an ordering fact between two pull requests, not a change to any table.

## 4. Migration plan

| Step | Forward | Backward | Safe on a populated table |
|---|---|---|---|
| This story | **No migration file.** Nothing runs against any database | Revert the frontend PR (#40) | Not applicable: no table is touched and no statement is added |

No migration is added and none is edited (a merged migration is never edited).

## 5. Hand-over

| Who | What they must know |
|---|---|
| Dev, backend step of this story | **There is no backend work**: no table, no endpoint, no migration, no mock module of this story's own to delete. The step has nothing to implement; the orchestrator may close it without a pull request. The fault knobs PR #40 adds to item 4's mock, the mock's stand-in reading of the shoes through `api.get` (UI fix 1) and its test `src/test/shop-mock.test.ts` are deleted with that mock in item 4's backend step and the catch-up of #40 (services file §3, §3.1, §5 K6). Still no schema consequence |
| Test lead | Fixtures (30 live shoes, a one-colour and one-size catalogue, a draft shoe, 60 live shoes for the dealer's page size) are built with SQL in a test database, never the seed. The shoe sizes are written in whichever form `main` holds; no column is added for any fixture |
| Plan item 4's backend | Nothing to add to its schema. Its statements are the ones this story's states are read from |
