# Service contracts — Shop phone filters, Clear all, loading and errors (plan item 16)

Module `catalogue` · Requirements CAT-003 items 6–8, CAT-004 in full · Story `docs/catalogue/stories/shop-phone-filters-clear-all-loading-and-errors.md` · UI PR #40 · Tables: `docs/architecture/erd/shop-phone-filters-clear-all-loading-and-errors.md` (none changed).

Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/products` reaches Go as `/v1/products`.

## 1. Verdict: this story adds, changes and removes no endpoint

The story is `/shop` in the browser. It reads four routes that other stories define and serve, and adds no request, response field, status code, error code, header or route:

| Read | Defined by | Used for |
|---|---|---|
| `GET /v1/products?status=live&limit=12&page=n&…` | item 4, `services/search-filters-and-sorting-served-by-the-api.md` §3 (envelope `{items, total, page, limit}`, `_base.md` §2.4) | The list, the count, "Show more" |
| `GET /v1/product-filters` | item 4, same file §4 | The Colour options and the Size (EU) pills |
| `GET /v1/categories`, `GET /v1/lines` | S3, `services/site-data-from-the-database.md` | Names for the heading and the For and Line groups |

All four are public, answer `Cache-Control: no-store`, and fail with the project's envelope `{error, message, errors}` and its closed lowercase catalog (`_base.md` §2.3). A failed query is `internal` (500); `unavailable` (503) is only `/healthz` and shutdown. Offset paging is the project's only scheme and is reused untouched.

What this design adds is **(a)** the table of answers the page branches on, so a wrong status from the API is named as the cause and not blamed on the page (§2); **(b)** how many requests each visitor action sends, which the verifying tester counts (§2.1); **(c)** the mock read and judged (§3); **(d)** the four decisions the story leaves to TL, ruled, and four more that follow from them (§4); **(e)** what plan item 4's backend swap must keep and the order the two pull requests meet in (§5, §6).

## 2. The reads, and what the page draws for each answer

`useShopList` (`src/lib/shop-list.ts`) reports `{ items, total, hasMore, loading, error, loadMore, retry }` for the **current query only** and `ShopResults` draws one thing at a time:

| What the shop answers | What the hook reports | What the results area draws |
|---|---|---|
| `200`, `items` of up to twelve, `total` above `items.length` | `items`, `total` = the API's total, `hasMore` true, `loading` false, `error` null | The cards, then "Show more"; the line `<total> matching` |
| `200`, `total` 0, `items: []` — a search or filter that matches nothing: `q=zzzz`, `line=zzz`, `colour=zzz`, `size=99` | `items: []`, `total` 0, `loading` false, `error` null | The empty state: "No pair matches that.", its line, "Clear all" with an arrow, and `0 matching`. **This is the only way the empty state is reached** |
| The first request of a query is on its way (a first visit, a ticked box, "Try again") | `items: []`, `total` null, `loading` true, `error` null | "Loading the shoes…" (`role="status"`). No cards of the previous list, no count, no empty state |
| The next page is on its way | `items` kept, `total` kept, `loading` true | The cards, and "Show more" disabled and reading "Loading…" |
| The first page is **rejected** — see the list below | `items: []`, `total` null, `loading` false, `error` set | "We could not load the shoes." (`role="alert"`) and "Try again". No count, no empty state, no card |
| Page 2 or later is **rejected** | `items` kept, `total` kept, `loading` false, `error` set | The cards stay and `<total> matching` stays; the message and "Try again" take the place of "Show more" |
| An older query's answer **or failure** arrives after a newer one | Dropped by the hook (`current` flag of the effect) | Whatever the newest query says (SC-30) |

**Rejected** means `api.get` rejects with an `ApiError` or the hook receives any `Error`. All of these are one view, because the page can do one thing about any of them (ask again) and must claim nothing it does not know:

- `500` `internal` — the database cannot be reached, a statement fails or times out (item 4 §3.9).
- `502`, `503`, `504` from the edge proxy while the API is stopped or restarting — a body that is not the envelope arrives as `ApiError` with code `internal` (`api.ts` `failure`).
- No answer — the browser is offline, the connection is refused, the request is aborted, or 15 s pass (`ApiError` `status` 0, code `network`).
- `200` whose body is not JSON — a static server answering for a missing proxy rule. `api.ts` turns an undefined body into `ApiError`, so it is an error and never an empty list.
- Any `4xx`, including `400 bad_request` for an address a person edited by hand (`q` over 100 characters, for example). The shop's own requests never produce one (the page sends only bands and sorts it knows), so the view is the honest one: "could not load".

The page draws **no part** of the rejection: not the status, not the `error` code, not the `message`, not `errors`. SC-25's test answers a 500 with a marker message and reads the page for it; no component reads an `ApiError` field.

### 2.1 What each action sends (the tester counts these)

The one retry is `api.ts`'s, not the hook's: `api.get` tries a failed `GET` once more (no answer, timeout, 5xx; never a 4xx), and each try is abandoned after 15 s. The hook adds none of its own.

| Action | Requests | Notes |
|---|---|---|
| First page, answered | 1 | `GET /v1/products?status=live&limit=12&page=1&sort=-released[&…]` |
| First page, every request answers 500 or no answer | **2**, then the error view | The try and `api.ts`'s retry. Longest wait before the error view appears: 2 × 15 s, all of it under "Loading the shoes…" |
| First page, `4xx` | 1, then the error view | A `4xx` is an answer; it is not retried |
| "Try again" on a failed first page, server back | 1 | Same filters, `limit=12`, `status=live`, `page=1` (SC-26) |
| "Try again", server still failing | 2, then the error view again | The message is replaced by "Loading the shoes…" for the whole attempt |
| "Show more" | 1, the button disabled until it settles | A disabled button sends nothing, so a second press sends no second request (SC-24) |
| "Try again" on a failed page 2 | 1 (2 if it fails again) | The **same** page: `page=2`, the same filters. Page 1's cards stay; none is repeated or skipped (SC-27) |
| A filter changed while a request is in flight | 1 more | The earlier request is **not aborted** (`api.get` takes no signal); its answer or failure is dropped by the hook. A few concurrent `GET`s of twelve shoes is the whole cost |
| Back from a shoe | 1 for the list, plus the options and the names the page reads on every visit | The list is re-read from the address and starts at its first twelve (SC-10, SC-11); nothing is restored from an earlier visit (ERD D1) |

A page is ordered by the sort key and then `seq`, a total order (item 4 §3.6), so asking the same page again returns the same shoes unless the dealer changed the catalogue in between; the hook drops a shoe it already holds before appending, so a shifted offset repeats nothing on screen (item 4 §2).

## 3. The mock, read and judged

This story has **no mock module of its own**. PR #40 adds fault knobs to **plan item 4's** mock, `src/lib/mock/search-filters-and-sorting-served-by-the-api.ts`, and changes `src/lib/shop-list.ts`. Both are unreviewed input as far as their contents go (the UI review covered isolation and the story's behaviours). Read against the contract above:

| What the UI stage does | What the contract does | Judgment | Frontend change at the swap |
|---|---|---|---|
| `localStorage["doi:mock"]` = `shop-slow` (3 s), `shop-error` (every list read fails), `shop-more-error` (page 2 and later fail); `navigator.onLine === false` fails like a 5xx | The real failures are a 5xx, a stopped API, a refused connection and an offline browser, made by the tester's request interception. The story says the product carries **no** switch for delay or failure (story §9) | **Departure — must not survive.** A storage flag that makes the production shop fail is a switch a visitor could be talked into setting, in a file the build ships | The whole file is deleted by item 4's backend step, which already deletes the mock; no importer remains (`shop-list.ts` is its only importer). Nothing else in the page reads `doi:mock` |
| A failed read rejects with `new Error("The shoes could not be read")` | The real rejection is an `ApiError` (`status`, `code`, `message`, `errors`) | Sound: the hook keeps `e instanceof Error ? e : new Error(String(e))`, and no component reads a field of it | None |
| A slow read resolves after 3000 ms | Slowness is the network; neither the hook nor the page has a timer of its own | Sound | None |
| Answer `{items, total, page, limit}` and `ProductFilters { colours, sizes }` | Identical (item 4 §2 and §4) | Sound | None |
| `useShopList` returns `{items, total: number \| null, hasMore, loading, error, loadMore, retry}` | S3's `useCollection` returns `{items, total, pages, page, hasMore, loading, error, loadMore}`. The shop does not go through `useCollection` (item 4 §6) | Sound, and the right shape for the page: `total` is `null` until the query's first answer, so "0 matching" is never claimed for a list that has not come; `pages` and `page` are drawn nowhere. The story's §3 says the shop needs "the data story's hook"; what it needs from S3 is `api.ts` (its `ApiError`, its one retry, its 15 s limit), which `listProducts` reaches through `api.get` | None |
| `useProductFilters` turns a failed read into "no options" | `GET /v1/product-filters` can fail like any read | Accepted, see decision T6 | None |

## 4. Decisions

### 4.1 The four the story leaves to TL (story §8)

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| T1 | How "Clear all" removes the size without touching the dealer's lists | **`useTableQuery({ sizeIsFilter: true })`**, the shop's opt-in (confirmed from the UI review). With it `size` is never read as a page size, `clear()` removes it and `filtering` counts it; without it every caller is unchanged (`size` is the page size, `clear()` keeps it, `filtering` ignores it). The address parameter stays `size`. `FilterRail` takes an optional `table` prop so the page and the rail are **one** instance; every call on one page passes the same value, so the page's `clear()` and the rail's cannot disagree. `ListPage` (the dealer's lists) calls `useTableQuery({ size: 20 })` and sends `size` as `limit`; it never forwards it as a filter, so item 4 making `size` the EU-size filter of `products` does not reach it | Renaming the address parameter (`sizes=`, `eu=`): CAT-003 item 1, CAT-004 AC-1 and AC-2, item 4's SC-34 and S4 decision T2 all name `size`, and every shared link, home tile and footer entry would change. A second copy of the hook for the shop: two sources of truth for `clear()`. Dropping `size` from the hook's own parameters for everyone: the dealer's `?size=50` would stop being a page size (SC-19) |
| T2 | Retry and "more is loading" in the list hook | **`useShopList.retry()`**, additive: it re-runs the read for the same `(query, page)`, so a failed page 2 is asked as page 2 and a failed first page as page 1 — never skipped, never a page already shown. **"More is loading" is derived**: `loading` with `items` on screen. An older request's failure cannot set `error` (the effect's `current` flag, SC-30). The hook also returns only the current query's answer and resets paging with the query, which SC-23 and SC-11 need | Putting retry in `api.ts` or `useCollection`: the shop uses neither's list state, and `api.ts` is every page's client — a retry policy there is a policy for the whole site. Automatic back-off retry: out of scope (story §5); the visitor's "Try again" is the only second chance beyond `api.ts`'s one |
| T3 | Where the four strings live | **`src/content.json`, four new keys, existing keys untouched** — a one-time exception to the freeze in `overview.md` §5, for new keys only: `pages.shop.loading` = "Loading the shoes…", `pages.shop.error` = "We could not load the shoes.", `pages.shop.retry` = "Try again", `pages.shop.loading_more` = "Loading…" (the single character `…`). Confirmed from the diff: they sit in alphabetical position inside `pages.shop`, so a textual conflict can only be with a neighbouring key and the second pull request to merge keeps both | Typing the strings into the components (what the dealer sign-in story did): SC-31 asks that the owner edit them in the Studio. TL adding the keys in a separate pull request: it would hold this story for a four-line change nobody else edits |
| T4 | Where the group-hiding rule for the size pills sits | **Inside `ShopSizes`** — fewer than two sizes draws nothing, heading included (so no "Size (EU)" over an empty grid). `FilterRail` keeps its own rule for the checkbox groups and also stays drawn when something is set but every group has one option, so "Clear all" is never unreachable (SC-13). A value ticked in a group that is not drawn still filters, still counts in "Filter (n)" and is still removed by "Clear all" | A shared "group" component for both: it would rewrite a component every dealer list imports |

### 4.2 Four that follow

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| T5 | Does any rejection other than a 5xx or "no answer" get its own view? | **No.** Every rejection of a list read is "We could not load the shoes." (§2). Unlike the product page, which must tell a `404 not_found` from any other failure (item 13, H3), a list has no "not found": an empty list is a `200` | Showing "No pair matches that." for a `4xx`: a claim the page cannot make. Separate words for a `400`: a second error message nobody drew, for an address only a hand can produce |
| T6 | A read of the options, the categories or the lines fails | **Not shown.** The list is the page's one job; the Colour and Size groups are not drawn while `product-filters` has not answered or has failed, and a heading named from a line or a category falls back to "All shoes" until its name arrives. No error view, no retry of its own | An error line in the rail: no criterion asks for it, it needs new words and a design nobody drew, and the visitor's way out is the same "Try again" or a reload. It can be its own story if the owner asks |
| T7 | Does the backend need a way to be slow or to fail on request, so SC-22 to SC-30 can be shown? | **No.** The hold is request interception in the tester's browser; a 500 is an intercepted answer, or the database taken away (`set_db_available`); a 502 is the API stopped; "offline" is the browser's own. **No delay, fault or sleep route, header or query parameter is added to the backend**, and no storage flag stays in the frontend (§3) | A `?delay=` parameter or a `/v1/_debug/fail`: a production route that exists only for a test, and a way for a stranger to make the shop slow |
| T8 | Is the number in "Filter (n)" or the `<N> matching` line asked of the server? | **No.** `n` is counted in the browser from the address — every ticked box and every pressed pill, one each; the search words and the sort are not counted. `<N>` is the list answer's `total`, drawn only once a list has arrived | A server-computed count of filters: a second reading of what the address already says |

### 4.3 Conflicts with merged decisions

None is overridden. Two interactions, stated so nobody finds them later:

| Where | The interaction | Resolution |
|---|---|---|
| Item 4's services §6 ("Not changed: `src/lib/table.ts`, `FilterRail`") | Item 4's own backend PR leaves both files alone; this story edits them (T1) | Consistent. Item 4 §6 says in so many words that how "Clear all" removes the size is plan item 16's. The two pull requests touch different lines of `FilterRail` and `table.ts`: item 4's backend touches neither |
| Item 4's services §6 ("the hook's own logic … is unchanged") | PR #40 changes `useShopList`: current-query-only answer, `retry()`, paging reset with the query, and the fault knobs in the mock (T2, §3) | Item 4's backend swap changes **imports** in `shop-list.ts`, not logic; #40 changes **logic**, not imports. The merge is mechanical and spelled out in §6 |

## 5. What item 4's backend swap must keep (checked at its review, and at the catch-up of #40)

Item 4's backend (PR #49) replaces the mock with the real client. The loading and error states of this story are only true if the real client behaves as the mock did on failure. Dev's work there, and the review's checks:

| # | Rule | Why this story needs it |
|---|---|---|
| K1 | `listProducts(req)` is `api.get<Page<Product>>("products", toParams(req))` and **rejects** on a non-2xx answer, on a network failure and on an offline browser. It never `catch`es into `{ items: [], total: 0 }` | A resolved empty list is the empty state, "No pair matches that." — a false claim for a failed read (SC-25 to SC-28) |
| K2 | `ApiError` propagates to `useShopList`; the page reads none of its fields (§2) | SC-25: no status, code or server message on the page |
| K3 | `useShopList`'s logic from #40 survives the swap: only the answers of the current query are returned, `retry()` re-asks the same `(query, page)`, paging resets with the query, a stale failure is ignored. Only the two imports change | SC-11, SC-23, SC-26, SC-27, SC-30 |
| K4 | `getProductFilters()` may still resolve to "no options" when its read fails (T6); `listProducts` may not | The rail must not grow an error the story did not ask for; the list must |
| K5 | The lines are read with `pageSize: 20`, the header's, so `/shop` makes one request for them (item 4's T3) | Item 4's SC-22; the line stays in #40's `Shop.tsx` after the catch-up |
| K6 | The mock file, with the `doi:mock` knobs, is deleted; no code reads `doi:mock`; the `vi.mock` path in `src/test/shop-list.test.tsx` points at the new client | T7: the product carries no switch for failure or delay |

## 6. Ordering: how the two pull requests meet

Story §3 says: build only after item 4 has merged. The facts that make that concrete:

- PR #40's branch was cut from **item 4's UI branch** (PR #38), and its diff against `main` still shows item 4's frontend files. Item 4's backend PR (#49) was cut from `main` and re-applies #38's frontend files itself, with the mock left out. So the two do not merge in any order without a catch-up.
- Write tools cannot merge `main` into a branch, and hand-copying `main`'s files is not accepted. **After #49 has merged, the pipeline catches #40 up with `main`.** If #38 merges first, the same applies: its mock is `git rm`ed at that merge.
- Expected result of the catch-up, file by file (K1–K6 above are the review of it):

| File | In #40 | On `main` after #49 | Resolution |
|---|---|---|---|
| `src/lib/mock/search-filters-and-sorting-served-by-the-api.ts` | fault knobs added | deleted | Stays deleted; the knobs go with it |
| `src/lib/shop-list.ts` | current-query-only answer, `retry()`, paging reset | imports the new client | #49's two imports, #40's logic |
| `src/test/shop-list.test.tsx` | five new cases | `vi.mock` of the new client's path | #40's cases, #49's path |
| `src/pages/Shop.tsx` | the page rewired for the states | lines read with `pageSize: 20` | #40's page, with `pageSize: 20` |
| `src/lib/shop-request.ts` | item 4's file, unchanged by #40 | adds `toParams` | #49's |
| `src/lib/table.ts`, `src/components/FilterRail.tsx`, `ShopResults`, `ShopNotice`, `ShopFilterToggle`, `ShopSizes`, `content.json` | this story's | untouched by #49 | #40's |

- **Run local** runs `main` with #40 caught up. **Verify** reads SC-22 to SC-30 through request interception on that running site, and SC-25b with the database taken away. SC-19b (the dealer's list with `size=50`) needs a signed-in dealer; if the dealer sign-in story's backend (plan item 9) is not on `main` when Verify runs, SC-19b is reported as waiting for it, not as a failure of this story, and SC-19a (the unit test of the hook) is the evidence for the hook.

## 7. Verification map

| Criteria | Met by |
|---|---|
| SC-1 to SC-5, SC-6 to SC-10, SC-12, SC-13 | The UI PR (#40) and item 4's served list and `product-filters`; no contract of this file |
| SC-11 (Back shows the first twelve) | The address is the whole state (ERD D1); the hook resets paging with the query (T2) |
| SC-14 to SC-18, SC-19a | T1 |
| SC-19b | T1; the dealer's list sends `size` as `limit` only (§4.1 T1) |
| SC-20, SC-21 | §2 row 2: `200` with `total` 0 is the one road to the empty state |
| SC-22 to SC-24 | §2 rows 3 and 4; §2.1 (the disabled button sends nothing) |
| SC-25 to SC-30 | §2 rejection list, §2.1, K1–K3 |
| SC-31, SC-32, SC-33 | T3; the UI PR |

## 8. Contracts reused unchanged

| What | Where | How this story uses it |
|---|---|---|
| Error envelope and the closed code catalog | `_base.md` §2.3 | Codes reachable here: `internal` (500), and `bad_request` (400) only for an address edited by hand. No new code; `unavailable` stays for `/healthz` and shutdown |
| The list envelope and offset paging | `_base.md` §2.4 | `limit=12`, `page=n`; `total` is every match. No second scheme, no cursor |
| One API client, one retry of a failed `GET`, 15 s limit | S3 §5.1 (`src/lib/api.ts`) | The shop never calls `fetch` itself and adds no timer and no retry of its own beyond the visitor's "Try again" |
| Public reads, `Cache-Control: no-store` | S3 §1.3 | A failed read is never cached by the browser, so "Try again" and Back reach the server |
| Idempotency, rate limits, background work, outbound calls | — | None: every read is naturally idempotent, no limit is added, nothing is queued, the API calls nobody |

## 9. Hand-over

| Who | What they must know |
|---|---|
| Dev, backend step of this story | **There is no backend work**: no route, handler, migration or fixture. The step has nothing to implement; the orchestrator may close it without a pull request |
| Dev, item 4's backend (PR #49) | K1–K6 (§5). The mock and its `doi:mock` knobs are deleted there; nothing of #40 is needed to merge it |
| Dev, the catch-up of #40 | The table of §6. Only imports, the mock's deletion, `pageSize: 20` and the `vi.mock` path differ; do not carry the knobs |
| Test lead | Count requests by §2.1 (a failing read is two requests, because of `api.ts`'s one retry); a failed list is the same view for every rejection (§2); the shop's one way to the empty state is `total` 0; fault, delay and offline come from interception, never from the product |
| UI follow-ups from the review of PR #40 (not backend, not blocking) | "Clear all" sits between Price and the Size pills (SC-3 lists it last; the test cases do not pin the place); keyboard focus after "Try again" falls to the page when its button unmounts; a failure for `(query, page 1)` shows at once if the visitor returns to that query before another read settles (clearing the failure when the effect starts fixes it); `filtering` counts any unknown parameter; a long sort label can wrap at 320px beside "Filter (n)". Dev may take them in the catch-up or leave them; none changes a contract here |
