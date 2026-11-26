# Service contracts — Track an order by number and phone (plan item 7)

Module `orders` (package `internal/orders`; additive constants in `internal/ratelimit`) · Requirements ORD-020 to ORD-023 · Story `docs/orders/stories/track-an-order-by-number-and-phone.md` · UI PR #71 · Tables: `docs/architecture/erd/track-an-order-by-number-and-phone.md` (no schema change).

This file **extends** `services/_base.md`; it does not redesign it. The route is `_base.md` §3.12, the limits are §2.7, the error catalog is §2.3, the validation boundary is §2.5, and the matcher is `services/customer-accounts-sign-up-sign-in-my-orders.md` §3.4 and §3.6 — all **fixed and not repeated** except where this file fills a gap. Every path is written the way the Go backend mounts it: **no `/api` prefix** (the browser's `/api/v1/order-lookups` reaches Go as `/v1/order-lookups`). A failed query is `internal` (500), never `unavailable`. This file adds **no error code**, no second error shape, no second limit scheme and no endpoint but the one the contract already names.

What this file decides because `_base.md` or the story left it open: the order of the route's checks and every sentence it sends (§2); the one thing it needs from the shared matcher and the rule for adopting it (§3, §4); what the UI's mock got right and the frontend's swap, exactly (§5); the decisions with their rejected alternatives and the accepted risks (§6); the lines of merged documents this design corrects (§7); the review checklist and how each criterion is proved (§8); and the build order (§9).

## 1. The endpoint, and what changes on `main`

| Method and path | Auth | Limited by | Traces to |
|---|---|---|---|
| `POST /v1/order-lookups` (`_base.md` §3.12) | none — any caller, a guest, a customer and the team alike | `order_lookup_number`, `order_lookup_address` (§2.7) | ORD-021, ORD-022, ORD-023 item 4 |

The plan item's text names the call `POST /v1/orders/lookup`; the contract named it `/v1/order-lookups` first, and the accounts story and its test cases already use that name. **The contract's name stands.** The route is a literal pattern: any other method falls to the server's catch-all and answers the JSON `404 not_found`, never a plain-text 405.

| Change | Where | Size |
|---|---|---|
| The route's handler | new `internal/orders/lookup_route.go` (§2) | ~110 lines |
| One line in `orders.Register`: `mux.HandleFunc("POST /v1/order-lookups", s.lookup)`; the package comment gains one sentence naming the lookup | `internal/orders/orders.go` | 2 lines |
| Two scope constants `OrderLookupNumber = "order_lookup_number"`, `OrderLookupAddress = "order_lookup_address"` — **only if** S14 did not already add them | `internal/ratelimit/ratelimit.go` | 2 lines |
| The matcher, the phone rule and `ratelimit.Hit`'s wait | **S14's** `internal/orders/lookup.go`, `phone.go`, `internal/ratelimit` — adopted, not written (§4) | 0 lines |
| The frontend swap | `src/lib/track-an-order-by-number-and-phone.ts`, two test files, delete one mock (§5) | one line of code |

Nothing else of the backend changes: `GET /v1/orders/{id}` and `POST /v1/orders/{id}/payment` never call `ratelimit`, so a lookup lock cannot touch them (ORD-022 item 6, SC-11), and **neither may ever import it**.

## 2. `POST /v1/order-lookups` — answers, checks and sentences

Body `{"number": "doi 24817", "phone": "+84938410552"}`. Every answer of the route — success and every error — carries `Content-Type: application/json; charset=utf-8` and **`Cache-Control: no-store`** (set first, before any check, so no path can forget it, and so SC-4's "same headers" cannot differ between a miss and a refusal). No answer sets a cookie. The handler never reads the session: `Resolve` runs as for every route and its result is ignored.

**Order of checks.** Each runs only when the earlier ones passed; the first that refuses answers. Steps 1–3 run **before** any database call and touch **no counter**, so a refused request is not a try (story §9).

1. **Media type** — not `application/json` (parameters such as `; charset=utf-8` allowed, `mime.ParseMediaType`) → `415 unsupported_media_type`, "Send the lookup as application/json."
2. **Body cap** — `http.MaxBytesReader` at 64 KiB (`maxBody`) → `413 payload_too_large`, "The request is too large."
3. **Shape** — read with the package's own `object(raw)` (`body.go`): exactly one JSON object, its values left undecoded so each can be judged on its own. Every fault is named at once, at most `maxFaults` (50), and only a key that matches `^[A-Za-z0-9_]{1,40}$` is echoed:

   | Fault | `errors.<key>` |
   |---|---|
   | invalid JSON, an array, `null`, data after the object | `errors.body`: "Send the lookup as one JSON object." |
   | a key other than `number` and `phone` | "The shop does not take this field." |
   | `number` or `phone` missing | "This field is needed." |
   | `number` or `phone` not a string (a number, `null`, an object) | "Send this as text." |

   → `400 bad_request`, message "Check the number and the phone." The **empty string is present** and is judged by the match: `"phone":""` is the 404 of step 5, a missing `phone` key is this 400. Neither string is ever bound to a statement as text (only the number's digits are), so a NUL in either is harmless and needs no refusal.
4. **The match** — `orders.Attempt` (§4) with the typed pair, `httpx.ClientAddr(r)` and a `hit` that gives the order a token (below), under `context.WithTimeout(r.Context(), txTimeout)` (5 s: a stuck lock ends as `internal`, nothing counted, never a hang).
5. **The answer**, from what `Attempt` returned.

**The `hit` callback** runs inside `Attempt`'s transaction, after the pair matched: `token, err = addToken(ctx, tx, m.ID)` — the function that already makes the token at placement (192 bits from `crypto/rand`, 32 URL-safe characters; only its SHA-256 is inserted). The token is kept in a local variable of the handler for the answer. **A new row is made on every right pair; earlier tokens stay valid; none expires.**

**Success** — `200`, the answer is built from four named fields and **never** by marshalling `Matched` or a row:

```json
{ "id": "doi-24817", "number": "DOI-24817", "placed_at": "2026-11-24T09:12:00+07:00", "token": "q3J0n8Zk2LmPa7Vx9TrBwE-fUj1a0Pd5" }
```

`placed_at` is `m.PlacedAt.In(ict).Format("2006-01-02T15:04:05-07:00")` (`ict` is the package's `time.FixedZone("ICT", 7*3600)`), the same text `GET /v1/orders/{id}` gives for the same order. No `phone`, `email`, `address`, `note`, `customer`, `city`, `status`, `total` or `items` — and none of the order's private values appears in any other field (SC-1).

| Code | HTTP | Trigger | `message` |
|---|---|---|---|
| `unsupported_media_type` | 415 | Step 1 | "Send the lookup as application/json." |
| `payload_too_large` | 413 | Step 2 (an addition to §3.12's table: the catalog's rule for any body over its cap, §2.3 and §2.5) | "The request is too large." |
| `bad_request` | 400 | Step 3; `errors` names each fault | "Check the number and the phone." |
| `not_found` | 404 | `orders.ErrNoMatch`: a wrong phone, an unknown number, an empty phone, a phone that matches only in part, a number with no digits or over 18. **One status, one body, one code path** (R5). Counts one failure | "No order matches that number and phone." (the same sentence as `POST /v1/account/orders`, decision A14 of the accounts design) |
| `rate_limited` | 429 | `*orders.LockedError`: the number's counter **or** the address's is locked; counts nothing, moves nothing. `Retry-After` is the wait in whole seconds, **the longer of the two**, clamped to 1–900 | "Too many tries. Please wait N minutes." — N is the wait in minutes rounded up, at least 1 ("1 minute" in the singular), the words of the account route |
| `internal` | 500 | Any other error from `Attempt`, **including a database that cannot be reached** and a statement that fails. Nothing is counted (the transaction rolled back), no token exists | "Something went wrong. Please try again." through the package's `s.internal` (logs the request id and the error; **never** the body, the number, the phone, the address or a token) |

A refused try does not extend the lock; a right pair neither counts nor clears; tries are counted only as `Attempt` counts them (§4). The 404 and the 429 are each written by **one** function, so the bytes cannot differ between two paths that mean the same.

**What the page does with it** (unchanged by this file; `Track` reads status and `code`, not words): `429` by status alone; `404` only with the code `not_found`; anything else — a 5xx, a `0`, a 404 that is not the shop's — is "We could not look that up. Please try again."

## 3. What this route needs from the matcher

The matcher is `docs/architecture/services/customer-accounts-sign-up-sign-in-my-orders.md` §3.4: `Pair`, `Matched`, `ErrNoMatch`, `LockedError`, `NumberKey`, `PhoneKey` and `Attempt(ctx, db, addr, pair, hit)`, with the statement of its steps. **This design does not change it.** It restates only what this story's criteria rely on, so the review can check a merged `lookup.go` against them:

| Behaviour of `Attempt` | Criterion that proves it |
|---|---|
| The number is its **ASCII digits**, 1 to 18; `DOI-24817`, `doi-24817`, `doi 24817`, `  DOI - 24817 `, `24817` are one number; none or more than 18 digits is a miss that counts on the address only | SC-2, SC-4 |
| The phone is its digits with a **leading** `84` read as `0`, compared **in full** with the stored phone read the same way; `552`, `410552`, `938410552`, a longer phone containing it and a one-digit difference never match; `938 410 552 84` is not `0938 410 552` | SC-2, SC-3 |
| One transaction at **`READ COMMITTED`** (`pgx.TxOptions{}`); the first statement is the number's advisory lock; then `Check` of both scopes; a locked scope **rolls back** and returns `*LockedError` (the longer wait) | SC-6, SC-9, SC-10 |
| A miss **commits** its two `Fail`s (number with digits, address) — a miss must not roll back its own count — and returns `ErrNoMatch` | SC-6, SC-7, SC-10 |
| A right pair commits only what `hit` wrote; **nothing is counted or cleared** | SC-8 |
| `Fail` is called with limit 5, window 15 minutes, lock 15 minutes | SC-6, SC-9 |

If the merged file differs from any row, **the difference is fixed in `lookup.go`, with a test, in this story's PR**: both routes depend on it and no copy is made.

## 4. Adopting the matcher: the gate, and the file layout

`internal/orders/lookup.go` and `phone.go` do not exist on `main` today. PR #69 (customer accounts, plan item 15) creates them as its design says (`Attempt` and `AddToAccount`, `ratelimit.Hit` returning the wait, the `orders.account_id` migrations that the matcher's statement reads). The account story's design (§3.4, decision A1) says the matcher is **never written twice**: whichever backend merges first creates it, the second adopts it unchanged.

**Decision T1 — this story's backend is built after PR #69 is on `main`.** Reasons: the matcher selects `orders.account_id`, a column only PR #69's migrations add; PR #69 is already built and in review; and this story has no schema and about 110 lines of its own, so it costs nothing to wait and a second matcher would cost a conflict in a file both routes depend on. If the step is dispatched before PR #69 is on `main`, Dev replies with that precise blocker rather than copying anything; the alternative that needs no waiting is to cut the branch from PR #69's head (not from `main`) and open the pull request against that branch, so the matcher is inherited, never copied.

```
internal/orders/
  lookup.go            S14's: Pair, Matched, ErrNoMatch, LockedError, NumberKey, PhoneKey, Attempt — adopted
  phone.go             S14's: PhoneDigits, ValidPhone — adopted
  lookup_route.go      NEW: (s *service) lookup — §2
  lookup_route_test.go NEW: refusals before any query (a nil DB panics if one is made); the answer's four keys; the 500 for a DB whose BeginTx fails
  lookup_db_test.go    NEW: SC-1 to SC-12 against PostgreSQL (skips without TEST_DATABASE_URL, as every database test; the first run is CI)
```

`lookup_route.go` calls the matcher under whatever name S14 merged (`Attempt` in its design; if the merged function is unexported, the call is the same — the handler lives in the same package). It does no SQL of its own except through `addToken`.

## 5. The mock, read and judged, and the frontend swap

The mock (`src/lib/mock/track-an-order-by-number-and-phone.ts`, PR #71) was read in the PR's diff, together with the client that calls it and the two components that render the answers. It is unreviewed input: the UI review checked that it is isolated in one module and not what it says. The mock's own inside (a 700 ms delay, the sample order, the sliding "recent misses" list, the number `50000` that errors) is test scaffolding and goes with it.

| Frontend (client `track-an-order-by-number-and-phone.ts`, `TrackForm`, `HeldOrders`) | Contract | Judgment | Change at the swap |
|---|---|---|---|
| The transport's contract: **resolves** with the answer, **rejects** with `ApiError(status, code, message)` for a refusal | `src/lib/api.ts`: a 2xx resolves with the parsed body; anything else — and no answer at all (`status` 0, `code` `network`) — rejects with `ApiError`; a non-JSON body (a gateway's own page) gives `code` `internal` | **Sound, and unlike the accounts mock, the same shape as the real client**, so the swap is one line and nothing is reshaped | `send` becomes `api.post<unknown>("order-lookups", body)` (the path has no `/v1`, no `/api`: `api.ts` adds them) |
| `LookupBody {number, phone}` sent **as typed**; the page checks and rewrites nothing | §3.12 and §2 above: both strings, present; the shop reads the digits | Sound — and required: the shop's 400 is for a missing key, never for a bad value | None |
| `LookupAnswer {id, number, placed_at, token}` read by `readAnswer`: four fields, kept as four; extra keys ignored; an answer that is not those four is "could not look that up"; `id` must match `^[a-z0-9][a-z0-9._-]*$` | §2: exactly those four keys; `id` is `doi-<n>`; `placed_at` RFC 3339 `+07:00`; `token` 32 URL-safe characters | Sound. The code is the safer reading of the PR text's "not exactly" (it ignores extras, which the contract never sends) and the PR body is the part that is off | None |
| `failureOf`: `429` by status; `404` **only** with code `not_found`; `status 0` and offline → offline; everything else (5xx, a 404 with code `internal`, a 400, a 413, a 415) → failed; the call is never retried | §2: the shop's 404 carries `not_found`, its 429 `rate_limited`; `api.post` never retries; a gateway's 404 page has code `internal` | **Sound, and it is what keeps a broken `/api` proxy rule from telling a customer "No order matches that number and phone."** | None |
| The 429 sentence says "15 minutes" and the page does not parse `Retry-After` | The header is whole seconds, 1–900 | Sound: the header serves any other client and the tests (SC-6, SC-9) | None |
| `keepOrder` is the one writer of `doi:placed`: id, number, placed time, token, **nothing else**; a looked-up order replaces its entry | The answer holds nothing else to keep | Sound | None |
| `heldOrders()`: the five newest by `placed_at`, from the store, no request | Nothing in the API is needed: a browser holds its own tokens (ORD-023 item 3) | Sound. No endpoint lists "my orders" for a guest, by design | None |
| The mock locks on any five recent misses, and counts a miss even for a number with no digits | §2.7 and §3 above: per number **or** per address, 15 minutes from the fifth failure, a no-digit number on the address only | Scaffolding that is close to the real rule and not the rule | Deleted |

**The swap, exactly.**

| Change | File | Why |
|---|---|---|
| `send` → `api.post<unknown>("order-lookups", body)`; import `api` beside `ApiError`; delete the `postLookup` import; update the transport's comment | `src/lib/track-an-order-by-number-and-phone.ts` | The one place the client reaches the shop; no second client, no `fetch` of its own (SC-29) |
| Delete the module | `src/lib/mock/track-an-order-by-number-and-phone.ts` | The backend step deletes its own mock |
| Repoint both tests' `vi.mock` at `../lib/api`, keeping the real `ApiError` (`vi.mock("../lib/api", async (orig) => ({ ...(await orig<typeof import("../lib/api")>()), api: { …, post: vi.fn() } }))`); every `postLookup` stub becomes `api.post`; one assertion pins the first argument `"order-lookups"` and the second `{number, phone}` exactly as typed | `src/test/track-lookup.test.tsx` (and `track-held-orders.test.tsx` only if it imports the mock) | The page keeps its assertions; only the thing stubbed moves |
| Delete `src/lib/mock/orders.ts` **if** `grep -rn "mock/orders" code/frontend/src` finds no importer afterwards (items 6 and 15 do not import it; `Track.tsx` no longer does) | `src/lib/mock/orders.ts` | The story's clean-up and PR #71's note: whichever of items 6 and 7 lands last deletes it |

**Not edited by the backend PR:** every component and page (`Track.tsx`, `TrackForm.tsx`, `HeldOrders.tsx`), `orders.ts`, `api.ts`, `App.tsx`, `Order.tsx`, `Checkout.tsx`, `theme.css`, `index.css`, `package.json` — and **`content.json`**. The four new `pages.track.*` keys (`searching`, `too_many`, `offline`, `failed`; new keys only, existing keys untouched) are **already in PR #71's diff**, which is the one PR that carries them; the backend PR does not touch the file. The branch is cut from `main` after PR #71 merged (the pipeline merges the UI first); if it is cut earlier, re-apply PR #71's own files **except the mock**, as items 6 and 15 did.

## 6. Decisions, and what was rejected

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| T1 | Build before or after the accounts backend | After (§4): the matcher is S14's | A copy of the matcher in this PR: two matchers drift, the story forbids it, and the second merge conflicts in a file both routes need |
| T2 | Where the handler lives | `internal/orders/lookup_route.go`; `orders.Register` mounts it | A new `lookup` package: ~110 lines that must read `orders` rows and call `addToken`, a second reader and writer of two tables |
| T3 | `Cache-Control` on errors | `no-store` on **every** answer of the route, set first | Only on the 200: SC-4 compares the headers of a miss with those of every other miss, and a shared cache could keep a 404 or a 429 that stands for one caller |
| T4 | How the body is read | Key by key through `object()`, so a **missing** key, an **empty** string and a `null` are three different things | A struct decoded with `DisallowUnknownFields`: a missing `phone` and `"phone":""` both become `""`, and the contract says the first is 400 and the second a miss (A11 of the accounts design) |
| T5 | Does a refused request count as a try | No: 415, 413 and 400 are decided before `Attempt`, and nothing is counted (story §9) | Counting them on the address: a client bug would lock a household, and a malformed body costs one parse |
| T6 | Both scopes locked | The **longer** wait is `Retry-After` | The number's or the address's alone: the header would promise a retry that is refused again |
| T7 | Does the route read the session | No. A customer, a team member and a guest are treated alike, count alike and are minted tokens alike | Different rules by role: two rules and two answers for one pair, and the account link is the accounts story's `POST /v1/account/orders` |
| T8 | Time bound | One `txTimeout` (5 s) over the whole `Attempt`, the wait for the number's lock included | No bound: a held lock would hang requests until the proxy cut them. A separate short `lock_timeout`: ten tries at once each wait for the one before them and a few milliseconds each is nothing, so the extra setting buys nothing |
| T9 | Padding a number (`00024817`) | Not normalised: the digits are the digits, and it matches no order | Stripping leading zeros: two spellings of one order would be two counters, and padding would dodge the number's limit |
| T10 | Isolation of `Attempt` | `READ COMMITTED` (the default) | `REPEATABLE READ`: the second of two waiting tries would count from a snapshot taken before the first committed, and the sixth failure would get through |
| T11 | The 429's words | The account route's: "Too many tries. Please wait N minutes." from the same seconds as `Retry-After` | "Wait 15 minutes" always: wrong from the second try on |

**Accepted risks** (named so a later reader does not rediscover them). **R-1** a right pair mints one token each time and is not limited (ERD file §4): the caller already holds the phone. **R-2** a phone is a weak secret: anyone who holds a number and its phone (a courier, a relative) opens the order, and a token never expires or can be revoked — the SRS's choice, listed for the owner in the story reply. **R-3** the address limit is exact for one try at a time but **a burst of tries at once for different numbers from one address can pass more than five** before the sixth finds the lock (the matcher serialises per number, not per address; the accounts design's R-5). The bound that protects a phone is the number's own, and each number stays at five tries per 15 minutes: under 500 guesses a day against a space of a billion ten-digit phones. Closing R-3 is a second advisory lock on the address, taken before the number's, in `Attempt` — one change in one file for both routes; it is not made here because ORD-022's conflict row and SC-10 ask for one number, and changing a function PR #69 has built would reopen that story's review for a risk its design accepted. **R-4** an unknown number and a wrong phone cost the same statements and so the same time; a number with no digits costs one `Fail` fewer and so slightly less, which tells a caller only what the caller typed.

## 7. Where this design corrects or departs from a merged document

| # | Document and line | Departure | Cost |
|---|---|---|---|
| C1 | `_base.md` §3.12's error table lists 400, 415, 404, 429, 500 | Adds `413 payload_too_large` for a body over 64 KiB, which §2.3 and §2.5 already require of every JSON endpoint | None: an existing code, in an existing shape |
| C2 | The plan item's text: `POST /v1/orders/lookup` | The contract's `POST /v1/order-lookups` stands (§1) | None: the client, the story and the accounts tests already use it |
| C3 | Accounts design §3.4: "whichever backend merges first creates [the matcher]" | This story's backend waits for S14 (§4, T1) instead of racing it | The step starts after PR #69 merges |

Nothing else is changed. The ERD is untouched (`erd/track-an-order-by-number-and-phone.md` §1).

## 8. BE review checklist, and how each criterion is proved

The backend review blocks on any of these.

| # | Requirement |
|---|---|
| R1 | The route is the literal `POST /v1/order-lookups`, mounted without `/api`; another method answers the catch-all JSON 404 |
| R2 | Checks run in the order of §2, and steps 1–3 touch **no** database and **no** counter: a test with a nil `DB` proves 415, 413 and each 400 (a nil `DB` panics if one is made) |
| R3 | One matcher: `NumberKey`, `PhoneKey` and `Attempt` each have **one** definition in the repository; the account route and this one both call it; no `orders` SQL by number and phone exists outside `lookup.go` |
| R4 | `Attempt` is `READ COMMITTED`, takes the number's advisory lock first, `Check`s both scopes, **commits** a miss's `Fail`s and **rolls back** a refusal (§3) |
| R5 | Every miss is the same bytes: status, `Content-Type`, `Cache-Control` and body of a wrong phone, an unknown number, `""`, `"   "`, a number with no digits and a number over 18 digits are compared in one test |
| R6 | The answer is four named fields; `Matched` and the row are never marshalled; `placed_at` equals the one `GET /v1/orders/{id}` gives |
| R7 | The token comes from `addToken`; no log line, error or answer holds a token, a number, a phone, an address or a body; the handler logs through `s.internal` only |
| R8 | `Retry-After` is the longer wait in whole seconds, clamped 1–900; the message's minutes are the same seconds, rounded up |
| R9 | No role check and no session read; a refusal, a miss and a right pair are the same for every caller |
| R10 | `read.go`, the pay route and every other handler do not import `ratelimit` |
| R11 | `txTimeout` bounds the whole call; every non-2xx body is `{"error","message"}` and `errors` appears only on 400; the database unreachable is `500 internal`, never 503 |
| R12 | The frontend diff is exactly the table of §5 and nothing else; no mock remains for this story |

| Criterion | Proved by |
|---|---|
| SC-1, SC-5 | `lookup_db_test.go`: the keys of the answer, the headers, the private values absent as text; `GET /v1/orders/{id}` with the token; two lookups, two tokens, both work; the placing token still works; `order_tokens` holds hashes only |
| SC-2, SC-3 | `lookup_db_test.go`: every combination of the five numbers and six phones; the partial, longer, one-digit and `84` cases (they exercise `NumberKey` and `PhoneKey`, which S14's own unit tests also pin) |
| SC-4 | `lookup_db_test.go`: one comparison of the six miss kinds (R5); no `Set-Cookie`; no `order_tokens` row |
| SC-6 to SC-9 | `lookup_db_test.go`: five misses then the sixth; address and number locks across five numbers and two addresses; a number that does not exist; the right pair in the middle; `Retry-After` falling; the 14- and 16-minute shifts of ERD file §6 |
| SC-10 | `lookup_db_test.go`: ten goroutines, one number — exactly five 404 and five 429, `hits = 5` |
| SC-11 | `lookup_db_test.go`: a locked number, then `GET /v1/orders/doi-24817` with a held token: 200 |
| SC-12 | `lookup_route_test.go` with a nil `DB` (415, 413, 400s, missing key vs `""`) and with a `DB` whose `BeginTx` fails (500 `internal`, generic body, no SQL or stack text) — both run in `go test` with no database; the database variant of the missing-key and `""` cases is in `lookup_db_test.go` |
| SC-13 to SC-28 | The two frontend test files (what is sent, each failure's sentence, the store, the list) and Verify against the running stack |
| SC-24, SC-25, SC-30 | Verify, after the Order page story (plan item 6) is on `main`: the route is proved here, the page's drawing is item 6's |
| SC-29 | The frontend diff against §5; `npm run lint` and `npm test` |

## 9. Build order

1. **Already on `main`:** orders and tokens (PR #50), the auth package and `ratelimit.Check`/`Fail` (PR #53), `api.post` (never retried).
2. **Before this story's backend:** PR #69 (customer accounts) — the matcher, `phone.go`, the scope constants, `Hit`'s wait and the `orders.account_id` migrations (§4). PR #71 (this story's UI) merges first by the pipeline's order.
3. **This story's backend PR:** `lookup_route.go` and its two tests, one line in `Register`, the frontend swap of §5. No migration, no `content.json`, no new dependency.
4. **For Verify (not for building):** the Order page story's backend and UI (PR #65, plan item 6) — SC-13, SC-22, SC-24, SC-25 and SC-30 open `/orders/<id>` with the token this route returns.
