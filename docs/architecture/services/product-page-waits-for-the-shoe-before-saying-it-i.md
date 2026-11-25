# Service contracts — Product page waits for the shoe before saying it is missing (plan item 13)

> **Revision 2 (the design step ran again after "Site data from the database" merged, PR #39).** The contract below stands; nothing in it is redesigned. What changed is evidence: §3.1 reads the merged `useRecord` against H1–H5 (all hold), §3.2 replaces the `renderToString` check, which cannot run, with a check that can, §5 and §8 are updated to the state of `main`, and §8 records that `main`'s `Product.tsx` has the bug today.

Module `site` · Requirement SITE-014 (with the not-found row of SITE-005; ADMIN-013 item 1 for the draft case) · Story `docs/site/stories/product-page-waits-for-the-shoe-before-saying-it-i.md` · UI PR #27 · Tables: `docs/architecture/erd/product-page-waits-for-the-shoe-before-saying-it-i.md` (none changed).

Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/products/samba-og-w-crystal-white` reaches Go as `/v1/products/samba-og-w-crystal-white`.

## 1. Verdict: this story adds, changes and removes no endpoint

The story is one page file and one test file. It reads **`GET /v1/products/{id}`** (project contract §3.3, served by S3's reader — `services/site-data-from-the-database.md` §1.5) and nothing else that is new. No request, response field, status code or error code is introduced; the project's error envelope `{error, message, errors}` and its closed lowercase code catalog are reused untouched; offset paging is not used (a single record is not a list).

What this design adds is **(a)** the table of answers the page branches on, so that a wrong status from the API is named as the cause and not blamed on the page (§2); **(b)** what the page needs from the data hook beyond what the data story's documents say (§3: four requirements, H1–H4, and one already-agreed behaviour, H5); **(c)** the verdict on the UI PR's test stub, which is the only "mock" (§4); and **(d)** one ordering fact about the draft case (§5).

## 2. The one read, and what the page does with each answer

`GET /v1/products/{id}` — public, no session needed, no request body, no query parameter that matters (the query string is ignored). `{id}` is read from the address (`/shop/:id`) and sent through `encodeURIComponent`. It must match `^[a-z0-9][a-z0-9._-]*$` and be at most 100 characters before any query is made; anything else is `404 not_found`, the same as an id that does not exist.

| What the shop answers | Body | What `useRecord` reports (S3 §5.2, with §3 below) | What the page draws between the header and the footer |
|---|---|---|---|
| `200` | the record alone, unwrapped (S3 §1.5) | `item` set, `loading` false, `error` null | The shoe |
| `404` with `error: "not_found"` — an unknown id, an id outside the pattern, a draft shoe for a visitor | `{"error":"not_found","message":"Not found."}` (no `errors` key) | `item` undefined, `loading` false, `error` null | The missing view: the h1 `pages.product.missing` and the "All shoes" pill to `/shop` |
| `500` `internal` — the database cannot be reached, a statement fails or times out | `{"error":"internal","message":"Something went wrong. Please try again."}` | `item` undefined, `loading` false, `error` set | Nothing |
| `502`, `503`, `504` from the edge proxy while the API is stopped, restarting or unreachable | not the API's envelope (a gateway page or empty body) | `error` set (S3 §5.1: a body that is not the envelope throws `ApiError` with code `internal`) | Nothing |
| No answer: connection refused, browser offline, the 15-second timeout | none | `error` set (`ApiError`, `status` 0, code `network`) | Nothing |
| Any other 4xx or 5xx (for example a `404` that is **not** the API's `not_found`, §3 H3) | whatever it holds | `error` set | Nothing |
| The request has been sent and nothing has come back, and the first render of an address before the request is sent | — | `item` undefined, `loading` **true** (§3 H1) | Nothing |

`401` and `403` cannot happen on this route: it is public, and a signed-in dealer is answered `200` for a draft shoe (ADMIN-013 item 1) — the page follows the answer and adds no status check and no "preview" banner (story §4).

Response headers are S3's: `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store` (a dealer's edit, or a shoe set back to live, is on the page at once).

### 2.1 Waiting is bounded, and the page adds no timer

Waiting for the shoe is the request's own life. `src/lib/api.ts` (S3 §5.1) aborts a request after **15 s** and tries a failed `GET` **once more** (no connection, timeout, 5xx; never a 404), so the longest an unreachable or hung shop keeps the page empty is about 30 seconds before `error` is set. The page then stays empty until the visitor reloads or leaves (story §4: no retry, no polling). I accept that: a shorter timeout for records would be a second knob in `api.ts`, and the story asks for "nothing until the shop answers".

## 3. What this page needs from `useRecord(name, id)` — the contract S3's hook must meet

DATA-005 item 7 and AC-8/AC-9/AC-11, and S3 §5.2, say `useRecord` returns `{ item, loading, error }`, that a request in flight is `loading`, that a 404 is `item` undefined with `loading` false and `error` null, and that any other failure sets `error`. They do **not** say four things this page depends on (H1–H4); H5 is already in S3's wording and is repeated so a refactor does not lose it. They extend S3; they do not change an export, a signature or a returned field. S3 has now merged (PR #39), so the hook is read, not promised: §3.1.

| # | Requirement on the hook | Why this page needs it | How it is checked |
|---|---|---|---|
| H1 | **`loading` is true on the first render of a `(name, id)` that has no answer yet** — derived while rendering, not set by an effect afterwards. A hook that starts at `loading: false` and flips to `true` in `useEffect` renders one frame of `{ item: undefined, loading: false, error: null }`, which is exactly "answered not found" | SC-3: the page must not read "no request yet" as "not found". Without it the missing view flashes for a frame on every shoe page and on every address change, which is the bug this story exists to remove | A render of `Product` with the **real** hook and a `fetch` that is held pending: the first commit contains neither `That pair is not in the shop any more.` nor the pill (TC-05, §3.2). `renderToString` cannot be used (§3.1). `data.test.tsx` already asserts `loading: true` and `item` undefined on the hook's first render |
| H2 | **A record answered for another `(name, id)` is never returned as this one's.** When `id` changes, `item` is `undefined` (or the record's own `id` differs from the one asked) until the new answer arrives; `loading` is true for that interval | SC-4, SC-13: going from one shoe to another must not show the first shoe's name or price under the new address, and must not show the missing view before the new answer. The page also compares `item.id` with the address (belt and braces, UI PR #27), but a hook that leaks is still a defect: other pages read it | S3's hook unit tests: change `id`, read the very next render |
| H3 | **"Not found" means status `404` and the API's own code `not_found`.** A `404` whose body is not the envelope — a gateway's or a static server's own 404 page, a mis-routed `/api` — arrives from `api.ts` as `ApiError` `status` 404, code `internal` (S3 §5.1), and the hook reports it as an **error**, not as "not found" | SITE-014 item 2 says the missing view follows an answer from the shop. If any 404 counted, a broken proxy rule would tell every visitor that every shoe has been withdrawn | A hook test with a `404` whose body is `<html>…` → `error` set, `item` undefined. The API's real 404 (`{"error":"not_found",…}`) still gives `error` null, so DATA-005 AC-11 holds unchanged |
| H4 | **A late answer for an earlier id never replaces the current one** (DATA-005 AC-10, "the latest query wins", applied to `useRecord` as well as `useCollection`) | SC-4/SC-13 hop between ids quickly (a colour dot, a related-pair card); the first shoe's slow answer must not land under the second address | A hook test: ask `a`, ask `b`, answer `b` then `a`; the hook holds `b` |
| H5 | **A failure after a record is in hand keeps the record** (S3 §5.2 "keeps the items already loaded"), and `error` may be set beside it. Not new | The page draws the shoe "whatever the other flags say" (story §3, UI PR #27 test) | Already in S3's wording; listed so a refactor does not drop it for `useRecord` |

None of these changes a response the API sends. If the hook ever stops meeting H1–H5, the finding is raised against the data story and `src/data/` is **not** patched here (story §4).

### 3.1 The merged hook, read against H1–H5

Read in `code/frontend/src/data/index.ts` on `main` (lines 379–421) and in `src/lib/api.ts`. Every row holds; none needs a change.

| # | Where it is met in `useRecord` | Existing test | Verdict |
|---|---|---|---|
| H1 | `useState({ key: "", loading: true, … })` and, **during render**, `fresh = state.key === key`, `waiting = id !== undefined && (!fresh \|\| state.loading)`. The first render of any id has `fresh` false, so `loading` is true and `item` undefined before any effect runs | `data.test.tsx` "is loading until the answer, then holds the record" asserts it on the first render | Holds |
| H2 | `item: fresh ? state.item : undefined`, `error: fresh ? state.error : null`: another key's state is never returned | "drops the old record the moment the id changes" (rerender with `b`: `item` undefined, `loading` true) | Holds |
| H3 | `isNotFound(e)` is `e instanceof ApiError && e.status === 404 && e.code === "not_found"`; `api.ts` `failure()` gives a body that is not the envelope code `internal` | "reads a 404 that does not carry the API's not_found code as an error" and "reads a 404 as no record and no error" | Holds. The API's own 404 is `httpx.Error(w, 404, "not_found", "Not found.", nil)` (`internal/data/handler.go`) |
| H4 | The effect's cleanup sets `live = false`, and `setState` runs only `if (live)`; a late answer is also ignored because `fresh` compares `state.key` with the current key | **None for `useRecord`** ("latest wins" is tested for `useCollection`) | Holds by the code. The one gap is a test: the colour-dot sequence in §3.2 (a slow first answer landing after the second address) closes it, so it belongs in the product-page real-hook test, not in `src/data/` |
| H5 | `setState` replaces the whole state, so `item` and `error` are never set together | — | Holds vacuously. The page's "a record in hand wins whatever the flags say" stays the safe rule if a refresh is ever added |

Three more facts about the hook that matter to this page:

- **`useEdited` has no `getServerSnapshot`** (`useSyncExternalStore(subscribe, getSnapshot)` with two arguments). React refuses to server-render a component that uses it, so the parenthetical in story SC-3 and the `renderToString` of TC-05 **cannot be run as written**. This is a limit of the instrument, not a defect of the page or of the hook, and the criterion itself ("the first render shows no missing text") is met by H1. §3.2 gives the check that can run. Not patched here: `src/data/` is the data story's.
- **The Studio's hand-edited copy is an answer.** When the dealer's Studio has set a copy of `products` (`edit !== undefined`), the hook answers from it at once with `loading: false` and no request, so an unknown id draws the missing view immediately. That is correct (nothing is in flight) and is not a violation of SITE-014; the visitor's site never has an edited copy.
- **A failed GET is asked twice** (`api.ts`: one retry on status 0 or 5xx, never on a 404), and identical GETs asked at the same moment **share one request** (`shared()`, keyed by URL). Both matter to how a test holds or fails a request (§3.2).

### 3.2 Holding one request in a test, with no backend route

The waiting rows (SC-1, 2, 4, 6, 8, 11, 12, 13) need a request that is held, delayed or failed on purpose, and the verify notes record that no tool of the pipeline holds one URL. Decision S4 (below) keeps the backend free of any such route, so the instrument is in the frontend's own test layer, with the **real** `useRecord` and the **real** `api.ts`:

| Need | How |
|---|---|
| Hold the shoe's request | `vi.stubGlobal("fetch", …)` as `data.test.tsx`'s `serve()` does; the stub answers `/api/v1/products/<id>` with a promise the test resolves (or with fake timers advanced by 2 000 ms), and every other URL the page asks for (the live list, `lines/<id>`, `categories/<id>`, `shop/shop`, `payments/vnpay`, `reviews`) at once, or at 4 000 ms for SC-6 |
| Prove "nothing at the first paint" (SC-3) | Render `Product` in a `MemoryRouter` at `/shop/<id>` beside a probe component whose `useLayoutEffect` copies the container's `innerHTML`. Layout effects run in the first commit, **before** the passive effect that sends the request, so the copy is the first paint. Assert it is empty. A second copy taken while the request is held is also empty |
| Answer 404 / 500 / offline | 404: `new Response(JSON.stringify({error:"not_found",message:"Not found."}), {status:404})`, asked **once**. 500 with the envelope, or `fetch` rejecting: asked **twice** (the retry), then `error` is set. A 404 whose body is `<html>` is an `error`, not the missing view (H3) |
| Unreachable for 5 seconds (SC-11) | With fake timers, `fetch` rejects; advance 5 000 ms in steps and assert the container is empty at each |
| Colour dot to a slow shoe (SC-4, SC-13) | Navigate to the second id, hold its URL, assert empty, then resolve it. The first id's answer, resolved **after** the second, must not replace the second (H4) |
| Do not leak between tests | `shared()` keeps a held GET in `inflight` until it settles, so a request left pending joins the next test that asks the same URL. Settle (resolve or reject) every held request before the test ends, or give each test its own id |

This is test code in `src/test/` and edits nothing in `src/data/`. It complements the nine stubbed tests of PR #27 (which prove the page's decision); it proves that the real hook and the page agree. It is the story's backend-independent evidence for the rows a running site cannot hold; the running site still carries the 404, 500 and unreachable rows, whose faults need no hold (§5).

## 4. The "mock", read and judged

The UI PR adds **no mock module** (the story needs no data of its own). The only stand-in is inside `src/test/product-page.test.tsx`: the hook is stubbed per state, and every other record and list comes back empty.

| What the stub returns | What the contract says | Judgment |
|---|---|---|
| `useRecord` → `{ item, loading, error }` with `item` a `Product` or undefined, `loading` boolean, `error` an `Error` or `null` | DATA-005 item 7; S3 §5.2 | Matches field for field and in kind. No difference |
| `useCollection` → `{ items: [], total: 0, pages: 1, page: 1, hasMore: false, loading: false, error: null, loadMore }` | DATA-005 item 4 (`{ items, total, pages, page, hasMore, loading, error, loadMore }`) | Matches. The real hook reports `loading: true` for a moment; the page tolerates an undefined or empty related list (reviewed in the UI PR, SC-6), so the stub's `loading: false` hides nothing |
| The 404 stub is `{ item: undefined, loading: false, error: null }` | S3 §5.2 and H3 | Matches the only form of "answered not found" the hook is allowed to report |
| The test shoe's `id` is `samba-og-w-crystal-white` with `sizes` `["36","38","40"]` | The seeded record has nine sizes from `36` | Harmless: the stub is not a fixture of the API. TC-10 and TC-11 read the real record |

Revision 2 re-read the UI PR at head `5518928`: still no mock module, still nine tests, and the stub's `useRecord` shape still matches the merged hook field for field (`{ item, loading, error }`); the verdicts above stand.

**Frontend change required when the API lands: none.** Nothing is swapped and no file is deleted: the page already reads through `src/data/index.ts`, which S3 re-points at the API. The stub never ships (it is in `src/test/`).

## 5. Ordering: what can be verified when

| Criteria | Needs | Until then |
|---|---|---|
| SC-3 / TC-05 (real hook, no flash on the first render) | S3's hook with H1 (**landed**, §3.1) and the UI PR's page | Runnable now through §3.2 (first-commit probe, real hook, `fetch` held). Not through `renderToString`, which React refuses for `useEdited` (§3.1). Against the running site it needs a build **of the UI PR's head**: a bundle built from `main`'s `Product.tsx` draws the missing view on every first paint, which is the bug, not a failure of the fix |
| SC-1, 2, 4, 5, 6, 8, 9, 11, 12, 13 and TC-27, 28, 30 | S3's reader and hook (**landed**), served through `run_project` from a build of PR #27's head | The 404 row (unknown id), the 500 row (database down) and the unreachable row (API stopped, a gateway answer) need no hold and can be run against the site. The **held** rows (SC-1, 2, 4, 6, 8's pre-answer half) have no instrument on the running site; their evidence is §3.2 plus the stubbed tests (TC-25). Report them as covered by unit evidence, not as unverified failures |
| **SC-10 / TC-29 (a draft shoe, as a visitor)** | **The visitor rule of ADMIN-013 item 1: `GET /v1/products/{id}` answers a no-session caller `404 not_found` for a `draft` shoe** | **S3 does not serve it.** S3's decision C1 (`erd/site-data-from-the-database.md` §7) serves every row, because no draft can exist before an endpoint can create one, and S3 §4.1 hands the rule to "Manage shoes, lines, categories, reviews and orders" (plan item 10). Test's arrangement T sets a shoe to `draft` directly in the database, which S3's reader would still answer `200` |

The last row is a conflict between this story's SC-10 and a merged decision, so it is stated and resolved here rather than quietly overridden:

| Option | Cost |
|---|---|
| **A (chosen). Leave S3's C1 and §4.1 as merged; SC-10's draft half waits for plan item 10** | Zero rework: S3 is being built against C1 and the manage story already carries the rule (S3 §4.1: it may not merge a write route for `products` without it). The page-side half of SC-10 (an answer `404` draws the missing view only after the answer, and never draws the shoe) is the same path as SC-8 and is verified there with an unknown id. Only "the API answers a visitor 404 for a draft" is deferred |
| B. Add the visitor predicate (`status = 'live'`) to S3's `products` reader now | Reopens an in-flight story and overrides its recorded decision C1; adds the dealer widening question (`auth.Resolve` does not exist until the sign-in story) to S3. It buys nothing for production, where no draft can exist yet |

**Decision:** option A. For the Test lead: TC-29 and the draft half of SC-10 are **UNVERIFIED, blocked by plan item 10 (ADMIN-013 item 1)** until that story lands — not a failure of this story. A draft produced by a direct database update before that story is **not** a valid arrangement, because the visitor rule is not there to hide it.

## 6. Contracts reused unchanged

| What | Where | How this story uses it |
|---|---|---|
| Error envelope and the closed code catalog | `_base.md` §2.3 | The page branches on `not_found` (404) versus anything else. Codes reachable here: `not_found`, `internal`. No new code; `unavailable` (503) stays for `/healthz` and shutdown, a failed query is `internal` |
| Public read, `Cache-Control: no-store` | S3 §1.3, §1.5 | The shoe is as fresh as the dealer's last save |
| One API client, one retry of a failed `GET`, 15 s timeout | S3 §5.1 | The page never calls `fetch` itself and adds no timer or retry (story §4) |
| Request id, access log | `_base.md` §2.1 | The access log records the path; a shoe's id is not a secret. Nothing new is logged |
| Idempotency, rate limits, background jobs, outbound calls | — | None. A read is naturally idempotent, no limit is added, nothing is queued, the API calls nobody |

## 7. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| S1 | Which answer means "the pair is missing"? | **Only a `404` carrying the API's `not_found`** (H3) | Any `404`, or any 4xx: a gateway's own 404 page would withdraw every shoe at once. The story's whole point is that the missing view follows the shop's answer |
| S2 | Does the page need a lighter "does it exist" call (`HEAD /v1/products/{id}`, an `exists` flag) so it can show the missing view sooner? | **No.** The record is the answer; one request per shoe | A second route for the same row: two requests to learn what one tells, and a route to keep in step with the visitor rule |
| S3 | Should a draft or a deleted shoe answer differently from an unknown id so the page can word it differently? | **No.** One body for all three (ERD D1); one missing view | `403`/`410` for a draft: it tells a stranger the shoe exists before it is published |
| S4 | Does the API need a way to be slow or to fail on request so that SC-1, SC-11 and SC-12 can be shown? | **No.** The hold is a browser-side or gateway-side delay of one URL, a stopped API container gives the 502, and `set_db_available` gives the 500. **No test-only delay, fault or sleep route, header or query parameter is added to the backend** | A `?delay=` parameter or `/v1/_debug/fail`: a production route that exists only for a test, and a way for a stranger to make the shop slow |
| S5 | Should the page show something after a long wait (30 s, §2.1)? | **No** (story §4: no loading or error view; the approved design draws none) | A timeout view or "Try again": a design addition and new words in `content.json`, which the owner can ask for as its own story |

### Conflicts with merged decisions

One, in §5: SC-10's draft half against S3's decision C1 / §4.1. Resolved as option A with the alternative recorded. No other: I read `services/_base.md` §2.2, §2.3, §3.3, `services/site-data-from-the-database.md` §1.5, §4, §5 and `docs/data/SRS.md` DATA-005, and every status and code above is in them.

## 8. Hand-over

| Who | What they must know |
|---|---|
| Everyone, **merge order** | `main`'s `Product.tsx` (line 202) still reads only `item` from `useRecord`, and `main`'s `useRecord` has been the real API hook since PR #39, so **every shoe page on `main` flashes the missing view while its request is in flight today.** PR #27 is the fix and has no dependency left (S3 landed). It should merge ahead of any story that waits on a shoe page, and the site served for any verification must be built from its head, not from `main` |
| Dev, backend step of this story | **There is no backend work.** No route, handler, migration, test fixture or `/api` prefix to check. The step has nothing to implement; the orchestrator may close it without a pull request |
| S3 "Site data from the database" | Merged and met H1–H5 (§3.1); no action. If `useRecord` is refactored, keep its first-render `loading: true` (H1) and the `not_found`-only rule (H3) |
| "Manage shoes, lines, categories, reviews and orders" | The visitor rule (`draft` → `404 not_found`, byte-identical to an unknown id) is what makes SC-10 and TC-29 of this story verifiable (§5); a dealer session gets `200` |
| Test lead | Report TC-29 and the draft half of SC-10 as blocked by plan item 10, not as a failure. TC-05 is no longer blocked: run it as §3.2 describes (first-commit probe with the real hook), since `renderToString` cannot run. Run the rest of §5 against a build of PR #27's head |
| "Free delivery on every order" (PR #24) | Shares `Product.tsx` with this story but in `DetailRows`, not in `Product()` (story §7); whichever merges second rebases |
