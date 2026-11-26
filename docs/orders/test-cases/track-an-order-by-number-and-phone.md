# Test cases — Track an order by number and phone

Module: `orders` · Plan item 7 · Requirements: ORD-020 (5 acceptance criteria), ORD-021 (12), ORD-022 (9), ORD-023 (7) (`docs/orders/SRS.md`)
Story: `docs/orders/stories/track-an-order-by-number-and-phone.md` — criteria SC-1 … SC-30 (one or more cases each; the table at the end walks the list)
Contract: `docs/architecture/services.md` §2.1 (base, timestamps), §2.2 (who may look an order up), §2.3 (error envelope), §2.5 (validation boundary: an undefined field is 400, a body over its cap is 413), §2.7 (the lookup limits), §3.11 (`GET /v1/orders/{id}`), §3.12 (`POST /v1/order-lookups`); `docs/architecture/erd.md` (`orders`, `order_tokens`, `rate_limits`)

**Risk level: high.** This route is the only door to an order for someone who holds no token, and it is open to anyone on the internet. It sits on the privacy line of ORD-008 (no phone, email, address or note leaves the shop) and on the guessing line of ORD-022 (a ten-digit phone behind five tries per fifteen minutes). A wrong answer here either lets a stranger read an order or locks a real customer out, so every miss, every lock boundary, every status code and every header the story fixes is written out value by value, and each lock case is also written where the failure happens (five misses, the sixth refused). The page cases are medium risk: they carry the same privacy rules and the five states the design never drew, and the fixed copy is checked word for word.

## How these cases are observed

- **Windows.** Default 1280×800. Phone cases say 390×844 (and 320×640 where the criterion names it). Breakpoint cases use the widths they name (1023 and 1024).
- **Addresses.** The API is read through the gateway as `/api/v1/...` (the backend itself mounts `/v1/...`). Pages: `/track`, `/orders/<id>`, `/contact`, `/checkout`, `/shop/<shoe>`.
- **Copy, word for word** (the SRS writes `…` as U+2026 and `—` as U+2014; the page uses the same characters, never `...` or `-`):
  - **NOTFOUND** = `No order matches that number and phone. Check both, or message us.`
  - **TOOMANY** = `Too many tries. Please wait 15 minutes, or message us with your order number.`
  - **OFFLINE** = `You seem to be offline. Check your connection and try again.`
  - **FAILED** = `We could not look that up. Please try again.`
  - **SEARCHING** = `Searching…` · the button's resting label is `Find my order`.
- **Selectors the story fixes.** `[data-testid='track-submit']` (the button), `[data-testid='track-error']` (the one alert line), `[data-testid='track-orders']` (the "Ordered on this phone" list). On the Order page: `[data-testid='order-status']` (the h1), `[data-testid='order-total']`, `[data-testid='order-missing']`. Beyond those no class is relied on: the two fields are found **by their visible labels** `Order number` and `Phone` (the story does not pin their ids), the five steps are `main ol` with one `li` per step (the current one carries `aria-current="step"`), the details list is `main dl` (`dt` label, `dd` value), the closed rows are `details` elements each with a `summary`.
- **SEED.** The seeded order `doi-24817` / `DOI-24817`. Its phone is `0938 410 552`. Before the first case its private values are read once with `set_db_state`: `SELECT phone, customer, city, email, address, note, status, carrier, tracking, eta, placed_at, total FROM orders WHERE id = 'doi-24817'` (call this **PRIV**), and the phones of the other seeded orders: `SELECT id, phone FROM orders WHERE is_sample AND id <> 'doi-24817'` (**OTHERPHONE** is the phone of `doi-24816`; if its digits equal the SEED phone's digits the next order's phone is used instead).
- **LOOKUP(number, phone [, addr])** — `POST /api/v1/order-lookups` with `Content-Type: application/json` and the body `{"number": <number>, "phone": <phone>}` exactly as written in the case.
- **ADDR(a)** — the request also carries `X-Forwarded-For: <a>`. The address the shop counts is the **right-most** entry (services §2.2), so a different address is a different right-most entry (`198.51.100.31`, `.32`, …). Where a case sends no ADDR, all its requests come from the verifier's own address.
- **RL-RESET** — arrange with `set_db_state`: `DELETE FROM rate_limits`. Because the limits count per address, **every case that tries a lookup starts with RL-RESET and ends with it**, so a lock left by one case never reaches the next.
- **RL(scope)** — `SELECT scope, hits, window_started_at, locked_until FROM rate_limits ORDER BY scope` (rows of `order_lookup_number` and `order_lookup_address`; the subject is stored hashed, so a case that holds one subject at a time reads one row per scope).
- **AGE(m)** — move the counters back in time instead of waiting: `UPDATE rate_limits SET window_started_at = window_started_at - interval '<m> minutes', locked_until = locked_until - interval '<m> minutes'`. AGE accumulates: AGE(14) then AGE(2) is sixteen minutes in all.
- **TOK(order)** — `SELECT count(*) FROM order_tokens WHERE order_id = '<order>'`, and **TOKALL** = `SELECT count(*) FROM order_tokens`.
- **SEED-SET(…)** — `UPDATE orders SET <assignments> WHERE id = 'doi-24817'`. Nobody can move an order along through the API yet (no dealer session), so status, carrier, tracking, expected date, `email`, `note` and `placed_at` are set this way, as the story says. **SEED-SAVE** reads the old values first and **SEED-RESTORE** puts every one back at the end of the case (the seeded order is not deleted by RESET). **SETSHIP** = SEED-SET(`status = 'shipped', carrier = 'GHN', tracking = 'GHN-PT27XM', eta = '2026-11-26'`).
- **RESET** — `DELETE FROM orders WHERE is_sample = false` (items and tokens go with them); `UPDATE counters SET value = 24817 WHERE name = 'order_number'`; RL-RESET; every SEED edit and every renamed table put back; the database made available again.
- **PLACE(customer, phone)** — `POST /api/v1/orders` with a fresh `Idempotency-Key: tc-track-<n>` and the body `{"items":[{"product":"samba-og-w-core-black","size":"38","qty":1}],"delivery":"standard","payment":"vnpay","customer":<customer>,"phone":<phone>,"address":"12 Example Street, Ward 1","city":"Da Nang","note":"Gate code 4821"}`. It answers 201 with `order` and `token`. After RESET the first order is `doi-24818` / `DOI-24818`, the next `doi-24819`. Unless a case says otherwise the customer is `Lan Nguyen` and the phone `0912345678`.
- **A fresh browser** starts with empty `localStorage`, `sessionStorage` and cookies. **The browser's store** is `localStorage["doi:placed"]`, a JSON array of `{id, number, placed_at, token}` (story §3 and SC-19). **STORE(entries)** writes that key **before the page's own scripts run**; if the instrument cannot do that, the entries are written after the first load and the page reloaded.
- **Stubbing in the browser.** "STUB(path, answer)" intercepts the page's request and answers it with the status, headers and body the case gives; "DELAY(path, ms)" lets the real request through and holds the real answer for that many milliseconds; "ABORT(path)" fails the request at the network. The log of requests records, for each, the method, URL (with query string), headers and body; the log of the console records every `console.*` call. A case that needs one of these and the instrument cannot do it is UNVERIFIED (instrument lacked: request interception) and says so. "At 0.5 s" means half a second after the action, read from the page.
- **Dates and money.** Dates are `dd/mm/yyyy`; timestamps in the API are RFC 3339 with the Vietnam offset (`2026-11-24T09:12:00+07:00`). Entries written by a case use noon Vietnam time (`…T12:00:00+07:00`) so the same date shows in any browser zone from UTC−5 to UTC+11.

## Ambiguities found in the documents (reported, not silently resolved)

1. **The route's name.** The plan item's text says `POST /v1/orders/lookup`; the story, `services.md` §3.12 and the customer-accounts cases say `POST /v1/order-lookups`. Every case uses `/v1/order-lookups`.
2. **A database the service cannot reach: 500 or 503.** The story (SC-12) and `services.md` §2.3/§3.12 give this route `internal` 500 for a database failure (503 `unavailable` is for `/healthz` and a shutting-down process). TC-19 follows the story with the database refusing connections (`set_db_available`); TC-18 arranges a failed query (a renamed table), which is 500 under every reading. If TL's design answers 503 for a refused connection, TC-19 fails against the story and the story should say so.
3. **Two addresses through the gateway.** SC-7 needs a second network address. If the gateway appends its own address to `X-Forwarded-For` (so a client-sent entry is no longer the right-most), the verifier cannot vary the address from outside; TC-08, TC-09 (the second-address step) and TC-10 are then UNVERIFIED (instrument lacked: a second network address), never failed. The verifier detects this once: five failures from ADDR(A) on five unknown numbers followed by a right pair for another number from ADDR(B) — a 429 there means the header is not honoured.
4. **The 429 sentence of the API.** The story fixes the page's sentence ("15 minutes" whatever the wait) and the `Retry-After` range (whole seconds, 1 to 900). It does not fix the API's own `message`; the customer-accounts design words it "Too many tries. Please wait N minutes." The cases assert a non-empty message, the code `rate_limited`, no `errors` key and no `token`, not a sentence.
5. **Which counter a degenerate number counts on.** The accounts design counts a number with no digits (or over 18 digits) on the address scope only; the story says only that it "counts as one failure". No case pins the scope; TC-04 reads that these tries do not make a token and answer the same 404.
6. **The fifth failure still answers 404.** "The sixth and later tries are refused" means the fifth wrong try itself is answered with the generic 404 and is the one that starts the lock (SC-6, SC-9). A 429 on the fifth try fails TC-07.
7. **Whether `/track` lists a looked-up order before a reload when storage is refused.** SC-23 fixes only "a fresh load draws no list" and "after a reload it is not listed". TC-37 asserts those two and nothing about the same page load.
8. **The Order page is plan item 6** (PR #41 was still open when these cases were written). TC-22, TC-23, TC-36, TC-39, TC-40 and TC-47 need its real `/orders/:id`; if it is not on the verified build they fail on the hand-off, and the verifier notes that the Order page is missing, not the lookup.
9. **The four new content keys are not named in the story** (TL names them in the technical design). The cases check the four sentences, not the key names.
10. **A row's number and date are on one link.** The story draws `<number>` left and `<date>` right inside the link to `/orders/<id>`; the cases read the two as two elements in that link, left and right by their boxes.

---

## A. `POST /v1/order-lookups` — the backend (SC-1 … SC-12)

**Scenario**: TC-01 — a right pair answers 200 with exactly four keys, no-store, and none of the order's private values
**Given**: RL-RESET; PRIV read; TOK(`doi-24817`) = K0.
**When**: LOOKUP(`DOI-24817`, `0938 410 552`); then `GET /api/v1/orders/doi-24817` with `X-Order-Token: <the answer's token>`.
**Then**: The lookup answers 200; `Content-Type` starts with `application/json`; `Cache-Control` contains `no-store`; there is no `Set-Cookie` header. The body's keys are exactly `id`, `number`, `placed_at`, `token` (no others, whatever their order): `id` is `doi-24817`; `number` is `DOI-24817`; `placed_at` matches `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?\+07:00$` and equals the `placed_at` of the GET's answer; `token` matches `^[A-Za-z0-9_-]{22,}$`. The body has none of the keys `phone`, `email`, `address`, `note`, `customer`, `city`, `status`, `total`, `items`, and its text contains none of PRIV's values (the phone in any grouping — `0938 410 552`, `0938410552`, `938410552` — the email, address and note when they are not empty, the customer's name and the city). TOK(`doi-24817`) is K0 + 1. The GET with the token answers 200 with `id` `doi-24817`.
Traces: SC-1 (ORD-021 AC-1, AC-9, item 1; Privacy NFR)
Check: fetch_url

**Scenario**: TC-02 — the number and the phone are read as digits: every spelling opens the same order
**Given**: RL-RESET; one RL-RESET is repeated between combinations only if a combination answered anything but 200.
**When**: LOOKUP is sent for every pair of a number from {`DOI-24817`, `doi-24817`, `doi 24817`, `  DOI - 24817 ` (a space before and after, spaces around the dash), `24817`} with a phone from {`0938 410 552`, `0938410552`, `+84938410552`, `+84 938 410 552`, `84938410552`, `(0938) 410-552`} — 30 requests.
**Then**: Every one answers 200 with `id` `doi-24817` and a token. After the 30, `RL(...)` has no row (a success never counts) and TOK(`doi-24817`) has grown by exactly 30.
Traces: SC-2 (ORD-021 AC-2, AC-3, items 2, 3)
Check: fetch_url

**Scenario**: TC-03 — the phone must match in full; `84` is read as `0` only at the start
**Given**: SEED; RL-RESET before **every** request of this case (each miss counts, and five would lock the number).
**When**: LOOKUP(`DOI-24817`, p) for each p: (a) only the end of the phone: `552`, `410552`, `938410552`; (b) a longer phone that contains it: `0938410552123`, `00938410552`, `10938410552`; (c) one digit different: `0938 410 553`, `0938 411 552`, `0838 410 552`, `1938 410 552`; (d) `938 410 552 84` (an `84` at the end). Then, with RL-RESET before each, LOOKUP(`DOI-24817`, p) for p in `0938 410 552`, `84 938 410 552`, `+84 938 410 552`.
**Then**: Every request of (a), (b), (c) and (d) answers 404 `not_found` with one and the same body as TC-04's; none answers 200. The last three answer 200 with `id` `doi-24817`.
Traces: SC-3 (ORD-021 AC-7, item 6)
Check: fetch_url

**Scenario**: TC-04 — nine kinds of miss answer one and the same 404, make no token and set no cookie
**Given**: SEED and the phone OTHERPHONE of `doi-24816`; TOKALL = T0 read; RL-RESET before **every** request.
**When**: LOOKUP is sent for each of: (a) `DOI-24817` with the wrong phone `0938 410 553`; (b) `DOI-24817` with OTHERPHONE (a real phone, of another order); (c) `DOI-99999` with `0938 410 552`; (d) `DOI-24817` with `""`; (e) `DOI-24817` with `"     "` (five spaces); (f) `DOI-` with `0938 410 552`; (g) `""` with `0938 410 552`; (h) `1234567890123456789` (19 digits) with `0938 410 552`; (i) `DOI-24817` with `552`.
**Then**: All nine answer 404 with the same `Content-Type`, the same `Cache-Control` (or none, the same for all), no `Set-Cookie`, and **byte-identical bodies** of the form `{"error":"not_found","message":M}` for one non-empty M, with no `errors` key. The body holds none of the typed numbers or phones (`99999`, `1234567890123456789`, `0938 410 553`, the digits of OTHERPHONE) and does not say which half was wrong or whether the number exists. TOKALL is T0 after all nine (no `order_tokens` row was made).
Traces: SC-4 (ORD-021 AC-4, AC-5, AC-6, AC-8, item 4, Invalid-input row, Not-found row)
Check: fetch_url

**Scenario**: TC-05 — the token of a lookup opens that order and no other, and a stranger's read looks like a missing order
**Given**: RL-RESET; a token T from LOOKUP(`DOI-24817`, `0938 410 552`); PRIV read.
**When**: `GET /api/v1/orders/doi-24817` with `X-Order-Token: T`; the same GET for `doi-24816` with T; for `doi-99999` with T; for `doi-24817` with no token; for `doi-24817` with T whose last character is changed.
**Then**: The first answers 200 with the order's public view (`id` `doi-24817`, `number`, `status`, `total`, `items`) and no key `phone`, `email`, `address` or `note`, and none of PRIV's phone, email, address or note as text. The other four answer 404 `{"error":"not_found","message":M}` with one and the same body, no `errors` key — the body of an order that does not exist.
Traces: SC-5 (ORD-021 item 1; ORD-008 AC-5)
Check: fetch_url

**Scenario**: TC-06 — every success makes a new token, earlier tokens stay valid, and the database holds only hashes
**Given**: RESET; PLACE(`Lan Nguyen`, `0912345678`) answers `doi-24818` with the token T0; TOK(`doi-24818`) = 1.
**When**: LOOKUP(`doi 24818`, `+84 912 345 678`) answers T1; LOOKUP(`24818`, `0912345678`) answers T2; `GET /api/v1/orders/doi-24818` with each of T0, T1, T2; TOK(`doi-24818`) read; for each of T0, T1, T2 the four queries `SELECT count(*) FROM order_tokens t WHERE t::text LIKE '%<token>%'`, the same for `orders`, `order_items` and `rate_limits`.
**Then**: T0, T1, T2 are pairwise different; all three GETs answer 200 with `id` `doi-24818`; TOK(`doi-24818`) is 3; every one of the twelve queries answers 0 (the token text is in no table).
Traces: SC-5 (ORD-021 item 1, ORD-023 AC-4; ORD-003 AC-14; erd D5)
Check: fetch_url

**Scenario**: TC-07 — five wrong phones are answered 404; the sixth try, even right, is 429 with `Retry-After`, and so is every later try
**Given**: RL-RESET; TOK(`doi-24817`) = K0; TOKALL = T0.
**When**: LOOKUP(`DOI-24817`, p) for p = `0938 410 551`, `0938 410 553`, `0938 410 554`, `0938 410 555`, `0938 410 556` one after another (the fifth wrong try included); then LOOKUP(`DOI-24817`, `0938 410 552`) (the right pair); then LOOKUP(`DOI-24817`, `0938 410 557`), LOOKUP(`DOI-24817`, `0938 410 552`) again.
**Then**: The five wrong tries answer 404 with one and the same body (TC-04's). The sixth answers 429 `{"error":"rate_limited","message":M}` with a non-empty M, no `errors` key, no `token` key, `Content-Type` JSON and a `Retry-After` header whose value matches `^[0-9]+$` and is between 1 and 900 (and, being sent within seconds of the fifth failure, at least 850). The seventh and eighth also answer 429. TOK(`doi-24817`) is K0 and TOKALL is T0 (a refused try makes no token).
Traces: SC-6 (ORD-022 AC-1, AC-2, AC-9, items 1, 3)
Check: fetch_url

**Scenario**: TC-08 — five failures from one address, across five different numbers, lock the address
**Given**: RL-RESET; ADDR A = `198.51.100.31`, B = `198.51.100.32`; TOK(`doi-24817`) = K0.
**When**: From A: LOOKUP(n, `0938 410 552`) for n = `90001`, `90002`, `90003`, `90004`, `90005` (five numbers that do not exist); then from A LOOKUP(`DOI-24817`, `0938 410 552`); then from B LOOKUP(`DOI-24817`, `0938 410 552`).
**Then**: The five answer 404 with one and the same body. The lookup from A answers 429 with `Retry-After` between 1 and 900 and no token, though the pair is right and `DOI-24817` itself has no failure. The lookup from B answers 200 (the number is not locked, only the address was). TOK(`doi-24817`) is K0 + 1 (only B's).
Traces: SC-7 (ORD-022 AC-4, item 2)
Check: fetch_url

**Scenario**: TC-09 — five failures against one number from address A lock that number for address B too, and not other numbers
**Given**: RESET; PLACE(`Lan Nguyen`, `0912345678`) answers `doi-24818`; ADDR A = `198.51.100.41`, B = `198.51.100.42`.
**When**: From A: LOOKUP(`DOI-24817`, p) for five wrong phones; then from B LOOKUP(`DOI-24817`, `0938 410 552`); then from B LOOKUP(`DOI-24818`, `0912 345 678`); then from A LOOKUP(`DOI-24818`, `0912 345 678`).
**Then**: The five answer 404. B's right pair for `DOI-24817` answers 429 with `Retry-After` between 1 and 900 and no token (the number is locked whoever asks). B's right pair for `DOI-24818` answers 200 (the lock of one number does not stop another number from another address). A's right pair for `DOI-24818` answers 429 (address A has five failures). If the verifier cannot vary the address (ambiguity 3) the second and third steps are UNVERIFIED.
Traces: SC-7 (ORD-022 AC-5; "the lock of one number does not stop a right pair for another number from another address")
Check: fetch_url

**Scenario**: TC-10 — a number that does not exist locks like one that does: same status, header and body shape
**Given**: RL-RESET; six addresses `198.51.100.51` … `198.51.100.56`, then `.61` … `.66` (so only the number's counter reaches five, never an address's).
**When**: First, for the existing number: from `.51` … `.55` one wrong LOOKUP(`DOI-24817`, `0938 410 55<k>`) each (k = 1, 3, 4, 5, 6); from `.56` LOOKUP(`DOI-24817`, `0938 410 552`) — response X. RL-RESET. Then, for the missing number: from `.61` … `.65` one LOOKUP(`DOI-99999`, `0938 410 552`) each; from `.66` LOOKUP(`DOI-99999`, `0938 410 552`) — response Y.
**Then**: X and Y both answer 429 with `Content-Type` JSON and a `Retry-After` matching `^[0-9]+$` between 1 and 900; their bodies have the same keys (`error`, `message`; no `errors`, no `token`), `error` is `rate_limited` in both, and the two `message`s are equal (the two locks are fresh, so the same wait in minutes; if the minute count differs because a minute boundary fell between them the verifier repeats the pair). Nothing in Y says that the number does not exist.
Traces: SC-7 (ORD-022 AC-6, item 2)
Check: fetch_url

**Scenario**: TC-11 — a right pair neither counts as a failure nor clears the count
**Given**: RL-RESET; SEED.
**When**: Sequence 1: three wrong LOOKUPs for `DOI-24817` (`0938 410 551`, `0938 410 553`, `0938 410 554`); one right pair (`0938 410 552`); two wrong (`0938 410 555`, `0938 410 556`); then one more wrong try (`0938 410 557`); then one right pair. Sequence 2 (RL-RESET first): four wrong tries; then the right pair three times; then one wrong try; then the right pair again.
**Then**: Sequence 1: the three wrong answer 404; the right pair answers 200; the two wrong answer 404 (the right pair neither counted nor cleared: these are the fourth and fifth failures, and the fifth still answers 404); the next wrong try answers 429 and the right pair answers 429 (no 429 comes before the sixth failing try, so the right pair did not count; and the lock did come, so it did not clear the three). Sequence 2: the four wrong answer 404; each of the three right pairs answers 200 (a success never counts); the wrong try answers 404 (it is the fifth failure, not the eighth); the last right pair answers 429. In both, `RL(order_lookup_number)` reads `hits` 5 at the end.
Traces: SC-8 (ORD-022 AC-7, item 5)
Check: fetch_url

**Scenario**: TC-12 — refused tries do not move the lock: the wait only counts down
**Given**: RL-RESET; five wrong LOOKUPs for `DOI-24817` (the lock begins); RL read: `hits` H and `locked_until` L for both scopes.
**When**: Ten more tries in a row, alternating the right pair and a wrong phone (`0938 410 558`), each answer's `Retry-After` recorded; RL read again.
**Then**: All ten answer 429. The ten `Retry-After` values are a non-increasing sequence (each less than or equal to the one before) and none exceeds the first; none jumps back up to about 900. RL after: `hits` is still H (5) and `locked_until` is exactly L for both scopes (a refused try counts nothing and does not extend the wait).
Traces: SC-9 (ORD-022 AC-3, AC-9, item 4)
Check: fetch_url

**Scenario**: TC-13 — after 14 minutes the lock still holds; after 16 minutes the count starts again
**Given**: RL-RESET; five wrong LOOKUPs for `DOI-24817`; TOK(`doi-24817`) = K0.
**When**: AGE(14); LOOKUP(`DOI-24817`, `0938 410 552`). Then AGE(2) (sixteen minutes in all); LOOKUP(`DOI-24817`, `0938 410 552`). Then RL-RESET and five wrong tries again, AGE(16), and LOOKUP(`DOI-24817`, `0938 410 553`) followed by RL read; then four more wrong tries and one right try.
**Then**: At 14 minutes the right pair answers 429 with `Retry-After` between 1 and 60 and makes no token. At 16 minutes the right pair answers 200 with a token (TOK is K0 + 1). In the second run, after 16 minutes the wrong try answers 404 and `RL(order_lookup_number)` and `RL(order_lookup_address)` both read `hits` 1 (the first failure of a new count, not a sixth); the four further wrong tries answer 404 and the right try after them answers 429.
Traces: SC-9 (ORD-022 AC-3, AC-9, item 4; ORD-022 AC-3 "after the wait the count starts again")
Check: fetch_url

**Scenario**: TC-14 — ten wrong tries sent at the same moment let exactly five failures through
**Given**: RL-RESET; SEED.
**When**: Ten LOOKUP(`DOI-24817`, `0938 410 5<nn>`) with ten different wrong phones are sent as one parallel batch from one address; then `RL(...)` is read.
**Then**: Exactly five answer 404 and exactly five answer 429 (never six or more 404s); `RL(order_lookup_number)` reads `hits` 5 and `RL(order_lookup_address)` reads `hits` 5. If the instrument can only send the ten one after another the case is UNVERIFIED (instrument lacked: concurrent requests): a sequential run proves TC-07, not the conflict row.
Traces: SC-10 (ORD-022 Conflict row)
Check: fetch_url

**Scenario**: TC-15 — a lookup lock does not touch the order read with a held token
**Given**: RL-RESET; a token T from LOOKUP(`DOI-24817`, `0938 410 552`) taken **before** the lock; then five wrong LOOKUPs for `DOI-24817` (locked); LOOKUP right pair confirms 429.
**When**: `GET /api/v1/orders/doi-24817` with `X-Order-Token: T`; `GET /api/v1/orders/doi-24817` with no token.
**Then**: The first answers 200 with the order (`id` `doi-24817`), with no `Retry-After` header; the second answers 404 `not_found` (the lock makes no difference to who may read). RL still shows the lock (the reads did not clear it).
Traces: SC-11 (ORD-022 AC-8, item 6)
Check: fetch_url

**Scenario**: TC-16 — a body that is not valid, has a wrong type, an undefined field or a missing key is 400 and counts as no try
**Given**: RL-RESET; TOKALL = T0; `Content-Type: application/json` on every request.
**When**: POST `/api/v1/order-lookups` with each body: (a) `{"number":"DOI-24817","phone":` (cut off); (b) `not json`; (c) `{"number":24817,"phone":"0938410552"}`; (d) `{"number":"DOI-24817","phone":938410552}`; (e) `{"number":"DOI-24817","phone":null}`; (f) `{"number":"DOI-24817","phone":"0938 410 552","email":"lan@example.com"}`; (g) `{"number":"DOI-24817","phone":"0938 410 552","order":"doi-24817"}`; (h) `{"number":"DOI-24817"}`; (i) `{"phone":"0938 410 552"}`; (j) `{}`; (k) `[]`, then `"DOI-24817"`, then `null`; (l) an empty body. Then LOOKUP(`DOI-24817`, `0938 410 552`) six times.
**Then**: Every body of (a) to (l) answers 400 `{"error":"bad_request","message":M,...}` with a non-empty M and no `token` key; `errors` is an object where present, and names the field: (c) `errors.number`, (d) and (e) `errors.phone`, (f) `errors.email` (an undefined field is refused, never a 200 that ignores it), (g) `errors.order`, (h) `errors.phone`, (i) `errors.number`, (j) a non-empty `errors`. `SELECT count(*) FROM rate_limits` is 0 after (a) to (l) (a refused request is not a try) and TOKALL is T0. The six right pairs afterwards answer 200 each, never 429.
Traces: SC-12 (services.md §3.12 `bad_request`, §2.5; story §9 "a refused request is not a try")
Check: fetch_url

**Scenario**: TC-17 — a body that is not JSON is 415, one over the cap is 413, and neither makes a token or counts
**Given**: RL-RESET; TOKALL = T0.
**When**: POST `/api/v1/order-lookups` with (a) `Content-Type: text/plain` and the valid body `{"number":"DOI-24817","phone":"0938 410 552"}`; (b) `Content-Type: application/x-www-form-urlencoded` and the body `number=DOI-24817&phone=0938410552`; (c) `Content-Type: application/json` and a body of 70,000 bytes: `{"number":"DOI-24817","phone":"` followed by 70,000 digits `1` and `"}`. Then LOOKUP right pair six times.
**Then**: (a) and (b) answer 415 `{"error":"unsupported_media_type","message":M}` with no `errors` key and no token. (c) answers 413 `{"error":"payload_too_large","message":M}` with no `errors` key (services §2.5, the 64 KiB cap). `SELECT count(*) FROM rate_limits` is 0 and TOKALL is T0 after the three; the six right pairs answer 200.
Traces: SC-12 (services.md §3.12 `unsupported_media_type`, §2.5 `payload_too_large`)
Check: fetch_url

**Scenario**: TC-18 — a failed query answers 500 with the generic envelope and nothing of the database
**Given**: RL-RESET; TOKALL = T0; `ALTER TABLE rate_limits RENAME TO rate_limits_x` (put back afterwards).
**When**: LOOKUP(`DOI-24817`, `0938 410 552`) and LOOKUP(`DOI-24817`, `0938 410 553`). Then the table is renamed back and `ALTER TABLE order_tokens RENAME TO order_tokens_x` is applied: LOOKUP(`DOI-24817`, `0938 410 552`); the table is renamed back; LOOKUP(`DOI-24817`, `0938 410 552`).
**Then**: Each failing request answers 500 `{"error":"internal","message":M}` with a generic M and no `errors` key; the body has no `token` key, none of PRIV's values and none of `rate_limits`, `order_tokens`, `relation`, `SELECT`, `INSERT`, `SQLSTATE`, `pgx`, `goroutine` or `.go:`. The last request, with both tables back, answers 200.
Traces: SC-12 (services.md §3.12 `internal`; §2.3 "a query or insert that failed")
Check: fetch_url

**Scenario**: TC-19 — the database refusing connections: 500 with the same generic envelope
**Given**: RL-RESET; `set_db_available` takes the database away while the service keeps running.
**When**: LOOKUP(`DOI-24817`, `0938 410 552`) and LOOKUP(`DOI-24817`, `0938 410 553`); then the database is made available again and LOOKUP(`DOI-24817`, `0938 410 552`) is sent.
**Then**: The two requests answer 500 `{"error":"internal","message":M}` (the story's and services §2.3's code for a database that cannot be reached), no `errors` key, no SQL, stack, host name or other person's data. After the database is back the right pair answers 200. A 503 `unavailable` instead of 500 fails this case against the story (ambiguity 2).
Traces: SC-12 (story SC-12 "with the database unreachable the route answers 500 `internal`")
Check: fetch_url

**Scenario**: TC-20 — a lookup reads no session: any cookie, valid or not, is treated like none
**Given**: RL-RESET; SEED; `SELECT account_id FROM orders WHERE id = 'doi-24817'` is NULL.
**When**: LOOKUP(`DOI-24817`, `0938 410 552`) with `Cookie: session=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa` (43 characters, no such session); with `Cookie: session=garbage`; and LOOKUP(`DOI-24817`, `0938 410 553`) with the first cookie.
**Then**: The two right pairs answer 200 with exactly the four keys of TC-01 (never 401 or 403); the wrong pair answers the 404 of TC-04, byte for byte. `orders.account_id` of `doi-24817` is still NULL.
Traces: contract (services.md §2.2: "look one up … yes" for every caller; story §9 "a lookup is public to everyone")
Check: fetch_url

**Scenario**: TC-21 — a team session and a signed-in customer look an order up like anyone else, and nothing is linked
**Given**: RL-RESET; SEED; the admin signed in as the dealer sign-in cases do (`POST /api/v1/auth/sign-in`, cookie held) and, when the accounts story is on the verified build, a customer signed up through `POST /api/v1/account/sign-up` (cookie held). If neither can be signed in the case is UNVERIFIED (instrument lacked: a sign-in credential).
**When**: With each cookie: LOOKUP(`DOI-24817`, `0938 410 552`) and LOOKUP(`DOI-24817`, `0938 410 553`).
**Then**: Each right pair answers 200 with exactly `id`, `number`, `placed_at`, `token` (a team member gets no dealer view of the order here, no `phone`); each wrong pair answers the same 404 as a visitor's, byte for byte. `orders.account_id` of `doi-24817` is still NULL afterwards (the lookup links nothing).
Traces: contract (services.md §2.2 role table; story §9 "a lookup is public to everyone"; SRS actors Customer and Shop team)
Check: fetch_url

---

## B. The Track page: the lookup and its states (SC-13 … SC-20)

Every case of this section opens `/track` in a **fresh browser** at 1280×800 with the request log and the console log recording from the first request, starts with RL-RESET and ends with it. Cases that open the order set SETSHIP first and SEED-RESTORE after.

**Scenario**: TC-22 — a right pair sends one POST with the typed values and opens the order
**Given**: SETSHIP; RL-RESET; a fresh browser.
**When**: `/track` is opened; `DOI-24817` is typed into the field labelled `Order number` and `0938 410 552` into the field labelled `Phone`; `[data-testid='track-submit']` is pressed.
**Then**: The request log holds exactly one request to `/api/v1/order-lookups`: method POST, no query string, `Content-Type` JSON, and a body that parses to exactly `{"number":"DOI-24817","phone":"0938 410 552"}` (two keys, values as typed). The browser's address becomes `/orders/doi-24817` (pathname), with an empty `location.search` and `location.hash`, and the address contains neither the token of the answer nor `0938`. `[data-testid='order-status']` (the h1) contains `on its way` (any case).
Traces: SC-13 (ORD-021 AC-1, AC-12, items 1, 5)
Check: interact_page

**Scenario**: TC-23 — what is typed is sent as typed; other spellings open the same order
**Given**: SETSHIP; RL-RESET; a fresh browser for each of two runs.
**When**: Run 1: `doi 24817` and `+84938410552` are typed and the button pressed. Run 2 (a fresh browser): `24817` and `0938410552` are typed and the button pressed.
**Then**: Run 1's POST body is exactly `{"number":"doi 24817","phone":"+84938410552"}`; run 2's is exactly `{"number":"24817","phone":"0938410552"}` (the page rewrites nothing; the shop reads the digits). Both end at `/orders/doi-24817`.
Traces: SC-14 (ORD-021 AC-2, AC-3, items 2, 3)
Check: interact_page

**Scenario**: TC-24 — five kinds of miss show the same line in the same elements, and the typed values stay
**Given**: SETSHIP; a fresh browser; RL-RESET before **each** of the five tries (so the fifth cannot meet a lock).
**When**: One page load, five tries, each by typing both fields afresh (select all, type) and pressing the button: (1) `DOI-24817` and `0938 410 553`; (2) `DOI-99999` and `0938 410 552`; (3) `DOI-24817` and `552`; (4) `DOI-24817` and an empty phone; (5) both fields empty. After each, `[data-testid='track-error']`, its `outerHTML`, its parent's tag and class, the button and the fields' values are read.
**Then**: After every try the request log gained exactly one POST to `/api/v1/order-lookups` (in (4) with `"phone":""` and in (5) with `"number":"","phone":""` — the page does no checking of its own) and its answer is 404; exactly one `[data-testid='track-error']` exists; its text equals NOTFOUND; it has `role="alert"` (or an ancestor within the form has it) and contains an icon element (`svg`) before the sentence; the address is still `/track`; both fields hold what was typed in that try; the button reads `Find my order` and has no `disabled` attribute. The five `outerHTML` strings are byte-identical, and so are the parents' tag and class — a visitor cannot tell which half was wrong.
Traces: SC-15 (ORD-021 AC-4, AC-5, AC-7, AC-8, item 4, Not-found and Invalid-input rows)
Check: interact_page

**Scenario**: TC-25 — while the shop answers slowly the button is disabled, reads "Searching…", and a second press sends nothing
**Given**: SETSHIP; RL-RESET; a fresh browser; DELAY(`POST /api/v1/order-lookups`, 3000); `DOI-24817` and `0938 410 552` typed.
**When**: The button is pressed; at 0.5 s the button and the fields are read; at 1 s the button is clicked again; at 1.5 s Enter is pressed in the phone field; at 4 s the page is read.
**Then**: At 0.5 s `[data-testid='track-submit']` has the `disabled` attribute, its text is `Searching…` (U+2026), its computed `opacity` is 0.6 (within 0.01), and the two fields still hold `DOI-24817` and `0938 410 552`. The request log holds exactly one POST to `/api/v1/order-lookups` after 4 s (neither the second click nor Enter sent one). At 4 s the address is `/orders/doi-24817`.
Traces: SC-16 (ORD-021 AC-11, item 7; design-system §2.1.6)
Check: interact_page

**Scenario**: TC-26 — after a refusal the button is back, and starting a new try removes the old message
**Given**: RL-RESET; a fresh browser; `DOI-24817` and `0938 410 553` typed; the first answer not delayed.
**When**: The button is pressed (the line of NOTFOUND shows); then DELAY(`POST /api/v1/order-lookups`, 3000) is set and the button is pressed again; at 0.5 s and at 4 s the page is read.
**Then**: After the first press `[data-testid='track-error']` reads NOTFOUND and the button reads `Find my order`, enabled. At 0.5 s of the second try `[data-testid='track-error']` does not exist (the old message is gone) and the button reads `Searching…`, disabled; no other alert is on the page. At 4 s exactly one `[data-testid='track-error']` exists and reads NOTFOUND (one message at a time, not two); the button reads `Find my order` and is enabled; the fields still hold `DOI-24817` and `0938 410 553`.
Traces: SC-16 (ORD-021 AC-11; story §5 "at most one message is shown at a time; starting a new try clears the old one")
Check: interact_page

**Scenario**: TC-27 — after five wrong tries from the page the sixth, even right, shows the "too many tries" line in the same place
**Given**: RL-RESET; a fresh browser; the real shop.
**When**: Five tries with `DOI-24817` and the wrong phones `0938 410 553`, `554`, `555`, `556`, `557` (each pressed on the page; after the first, `[data-testid='track-error']`'s parent tag and class are recorded); then `0938 410 552` is typed and the button pressed (sixth); then the button is pressed a seventh time.
**Then**: The first five show NOTFOUND. After the sixth press the request log shows its answer status 429, `[data-testid='track-error']` reads TOOMANY exactly, has `role="alert"` (or an ancestor within the form does) and an icon, and has the same parent tag and class and the same own `class` attribute as the NOTFOUND line had; the address is `/track`; the fields hold `DOI-24817` and `0938 410 552`; the button reads `Find my order` and is enabled; the browser's store holds no entry for `doi-24817`. The seventh press shows TOOMANY again.
Traces: SC-17 (ORD-022 AC-2, item 3)
Check: interact_page

**Scenario**: TC-28 — the page reads the shop's status, not its words
**Given**: A fresh browser; `/track` open; `DOI-24817` and `0938 410 552` typed. Each answer is a STUB of `POST /api/v1/order-lookups`, one per press, in this order: (a) 429, `Retry-After: 5`, body `{"error":"rate_limited","message":"Slow down"}`; (b) 429 with the body `oops` (not JSON) and no `Retry-After`; (c) 404 with the body `{"error":"not_found","message":"Too many tries."}`; (d) 429 with `Retry-After: 900`.
**When**: The button is pressed once for each stub.
**Then**: (a), (b) and (d): `[data-testid='track-error']` reads TOOMANY exactly (not `Slow down`, not `5`, not `900`, not `Retry`; the wait is worded as 15 minutes whatever the header). (c): the line reads NOTFOUND exactly. In each the address is `/track` and the fields keep their values.
Traces: SC-17 ("the 429 is read from the shop's status, not from its words"); story §9 "the wait is worded, not read"
Check: interact_page

**Scenario**: TC-29 — offline: its own line, nothing sent, the values stay
**Given**: SETSHIP; RL-RESET; a fresh browser; `/track` open with `DOI-24817` and `0938 410 552` typed; then the browser is set offline (`navigator.onLine` is `false`, requests fail). If the instrument cannot emulate offline the case is UNVERIFIED (instrument lacked: network emulation).
**When**: The button is pressed; the request log is read; the browser is set online again and the button is pressed again.
**Then**: After the first press `[data-testid='track-error']` reads OFFLINE exactly (not NOTFOUND, not FAILED, not TOOMANY), has `role="alert"` and an icon; the request log holds no request to `/api/v1/order-lookups` made by this press; the fields hold their values; the button reads `Find my order` and is enabled; the address is `/track`. After the second press (online) the address becomes `/orders/doi-24817`.
Traces: SC-18 (ORD-021 Upstream-failure row "offline")
Check: interact_page

**Scenario**: TC-30 — the shop cannot be reached or answers 5xx: "We could not look that up", never the not-found or too-many line, one request per press
**Given**: SETSHIP; RL-RESET; a fresh browser; `/track` open with `DOI-24817` and `0938 410 552` typed. Six arrangements, one press each, in turn: (a) ABORT(`POST /api/v1/order-lookups`) (connection refused); (b) the backend stopped, so the gateway answers 502 (started again afterwards); (c) STUB 500 with `{"error":"internal","message":"x"}`; (d) STUB 502 with an HTML body; (e) STUB 503 with `{"error":"unavailable","message":"x"}`; (f) DELAY(`POST /api/v1/order-lookups`, 20000) (a timeout; the press is read at 17 s — UNVERIFIED if the instrument cannot hold a request that long).
**When**: The button is pressed once for each arrangement; the request log is counted per press; after (f) the arrangements are removed and the button pressed once more.
**Then**: In (a) to (f) `[data-testid='track-error']` reads FAILED exactly (not NOTFOUND, TOOMANY or OFFLINE), has `role="alert"` and an icon; the fields keep their values; the button reads `Find my order` and is enabled; the address is `/track`; and the request log holds exactly **one** POST to `/api/v1/order-lookups` per press (the call is never retried by the client). The last press answers 200 and the address becomes `/orders/doi-24817`.
Traces: SC-18 (ORD-021 Upstream-failure row; story §6 "one request per press")
Check: interact_page

**Scenario**: TC-31 — the browser keeps the id, number, time and token of a looked-up order and nothing else, once, in the one store
**Given**: SETSHIP; RL-RESET; a fresh browser; PRIV read.
**When**: A right pair is typed and sent from `/track` (the answer `A1` is read from the request log); the address becomes `/orders/doi-24817`; `localStorage` and `sessionStorage` are read in full; `/track` is opened again and its list read. Then, in the same browser, `24817` and `0938410552` are looked up again (answer `A2`); the store is read; `GET /api/v1/orders/doi-24817` is sent with A1's token.
**Then**: After the first lookup `localStorage["doi:placed"]` is a JSON array of one entry whose keys are exactly `id` (`doi-24817`), `number` (`DOI-24817`), `placed_at` (equal to A1's `placed_at`) and `token` (equal to A1's `token`); no other key of `localStorage` or `sessionStorage` exists for orders (no `doi:orders`, no second key), and no value of either contains `0938`, `410 552`, `410552`, PRIV's customer name, city or total. `/track` lists one row, `DOI-24817`. After the second lookup the array still has **one** entry for `doi-24817` and its `token` is A2's (a different token from A1's); A1's token still opens the order (the GET answers 200).
Traces: SC-19 (ORD-021 AC-10, item 5; ORD-023 AC-4)
Check: interact_page

**Scenario**: TC-32 — across a whole lookup the phone is only in the POST body, no list of orders is read, and nothing private comes back
**Given**: SETSHIP with `email = 'seed.private@example.com'` and `note = 'Leave with the guard'` (SEED-SAVE first); RL-RESET; a fresh browser; the request log, the console log and every response body recorded; PRIV read.
**When**: In one browser: a wrong pair on `/track` (`DOI-24817`, `0938 410 553`); the right pair (`DOI-24817`, `0938 410 552`), which opens the order; back to `/track`; then five wrong tries followed by a sixth with the right pair (locked, TOOMANY). The page's visible text, `localStorage`, `sessionStorage`, the console log and all recorded bodies are searched afterwards.
**Then**: No request URL (path or query string) contains `0938`, `410552`, `410%20552` or `%2B84938`, at any time. No request made while the address is `/track` has `24817`, `DOI` or `doi-` in its URL (the order read on `/orders/doi-24817` carries the order's id, not the typed number, and is excluded). No request is `GET /api/v1/orders` or `GET /api/v1/orders?...` (no list of orders is read, with or without a query). The typed phone (in any grouping: `0938 410 552`, `0938410552`, `938410552`) appears in the body of the lookup `POST`s and in no other request, no `localStorage` or `sessionStorage` value, no console entry and no recorded response body. No recorded response body, searched at any depth, has a key `phone`, `email`, `address` or `note`, nor the values `seed.private@example.com`, `Leave with the guard` or PRIV's address.
Traces: SC-20 (ORD-021 item 1; Privacy NFR; SC-20 "no request lists orders")
Check: interact_page

---

## C. "Ordered on this phone" (SC-21 … SC-23)

**Scenario**: TC-33 — six held orders: five rows, newest first by placed time, the sixth not listed, no call to the shop
**Given**: A fresh browser; STORE with these six entries in this **array order** (so insertion order differs from time order), each `token` a distinct text of at least 22 characters (`tc-token-30001-abcdefghijklm` …): `DOI-30003` placed `2026-11-23T12:00:00+07:00`; `DOI-30006` `2026-11-26T12:00:00+07:00`; `DOI-30001` `2026-11-21T12:00:00+07:00`; `DOI-30005` `2026-11-25T12:00:00+07:00`; `DOI-30002` `2026-11-22T12:00:00+07:00`; `DOI-30004` `2026-11-24T12:00:00+07:00` (`id` is the number in lower case: `doi-30003` …). The request log is recording; 1280×800.
**When**: `/track` is opened; the heading `Ordered on this phone`, `[data-testid='track-orders']`, its rows, each row's link `href`, number element, date element, boxes and computed `font-variant-numeric` are read.
**Then**: The heading `Ordered on this phone` is shown and `[data-testid='track-orders']` has exactly five rows, top to bottom: `DOI-30006` `26/11/2026`, `DOI-30005` `25/11/2026`, `DOI-30004` `24/11/2026`, `DOI-30003` `23/11/2026`, `DOI-30002` `22/11/2026`; `DOI-30001` (`21/11/2026`) is nowhere on the page. Each row is a link whose `href` is `/orders/doi-3000N` for its own number; in each the number's box is at the left of the date's box, and both have a computed `font-variant-numeric` containing `tabular-nums`. The request log holds no request to `/api/v1/orders…` and none to `/api/v1/order-lookups` (building the list makes no call to the shop).
Traces: SC-21 (ORD-023 AC-1, AC-3, item 1)
Check: interact_page

**Scenario**: TC-34 — with no tokens the heading and the list are not drawn at all
**Given**: Three fresh browsers: (a) nothing stored; (b) `localStorage["doi:placed"]` is `[]`; (c) it is `{"not":"a list"}`; 1280×800 and 390×844.
**When**: `/track` is opened in each at both sizes and the page and the console log are read.
**Then**: In all three, at both sizes: the text `Ordered on this phone` is not on the page, `[data-testid='track-orders']` does not exist, no empty `ul`, no empty box and no line such as `none yet` stands in the right-hand column (which holds only `What each step means`, its five rows and the promises); the page draws without a thrown error (the console log holds no uncaught exception).
Traces: SC-21 (ORD-023 AC-3, "Absent data" row)
Check: interact_page

**Scenario**: TC-35 — a looked-up older order takes its place by date, not the top
**Given**: SEED-SET(`placed_at = '2026-11-10T12:00:00+07'`) after SEED-SAVE; SETSHIP; RL-RESET; STORE with `DOI-30006` (`2026-11-26T12:00:00+07:00`) and `DOI-30005` (`2026-11-25T12:00:00+07:00`).
**When**: On `/track` the right pair for `DOI-24817` is looked up (the browser goes to `/orders/doi-24817`); the browser goes back to `/track` (the header's `Track an order` link or the Back button); the list is read; then `/track` is reloaded and read again.
**Then**: Both reads show three rows in this order: `DOI-30006` `26/11/2026`, `DOI-30005` `25/11/2026`, `DOI-24817` `10/11/2026` — the looked-up order is last, because the rows follow the placed date, and `DOI-24817` did not jump to the top.
Traces: SC-21 (story §9 "newest first is by placed time … a looked-up older order does not jump to the top")
Check: interact_page

**Scenario**: TC-36 — a row opens that order with the token this browser holds; a fresh browser lists nothing
**Given**: RESET; SETSHIP; PLACE(`Lan Nguyen`, `0912345678`) answers `doi-24818` with the token T1; LOOKUP(`DOI-24817`, `0938 410 552`) answers T2 and `placed_at` P2; browser B1 fresh with STORE of the two real entries (`doi-24818` with T1 and its `placed_at`; `doi-24817` with T2 and P2); browser B2 fresh with nothing stored.
**When**: In B1 `/track` is opened; the first row is pressed; the browser goes back; the second row is pressed. In B2 `/track` is opened.
**Then**: In B1 the rows are `DOI-24818` then `DOI-24817` (the newer first). The first press goes to `/orders/doi-24818`, where `[data-testid='order-status']` exists, `[data-testid='order-missing']` does not, and the page's order read carried `X-Order-Token: T1`; the second goes to `/orders/doi-24817`, whose h1 reads `Your order is on its way.`. In B2 the heading `Ordered on this phone` and `[data-testid='track-orders']` are absent although the shop holds orders (an order placed or looked up elsewhere is not listed).
Traces: SC-22 (ORD-023 AC-2, item 3)
Check: interact_page

**Scenario**: TC-37 — a browser that refuses storage draws no list; a lookup still opens the order for this page load
**Given**: SETSHIP; RL-RESET; before the page's scripts run `Storage.prototype.setItem` is replaced with a function that throws and `localStorage.getItem` returns `null`. If the instrument cannot run a script before the page's own the case is UNVERIFIED (instrument lacked: a script injected before page load).
**When**: `/track` is opened; then `DOI-24817` and `0938 410 552` are typed and the button pressed; the order page is read; the browser is then reloaded on `/track`.
**Then**: The first load of `/track` draws no `Ordered on this phone` heading and no `[data-testid='track-orders']`. The lookup opens `/orders/doi-24817` with the h1 `Your order is on its way.` and the page shows no failure message. After the reload of `/track` no list is drawn (the order is not listed), and the console log holds no uncaught exception.
Traces: SC-23 (ORD-023 AC-7; ORD-009 "Storage refused" row)
Check: interact_page

**Scenario**: TC-38 — the design's mock key is not read: only the real store builds the list
**Given**: A fresh browser; before the page loads, `localStorage["doi:orders"]` is written with an entry in the old mock's style for an order `DOI-24999` with a phone `0900000000` (any JSON array of one order object); `localStorage["doi:placed"]` is absent; 1280×800.
**When**: `/track` is opened; then `DOI-24999` and `0900000000` are typed and the button pressed.
**Then**: The page draws no `Ordered on this phone` list (the old key is neither read nor migrated). The press sends one POST to `/api/v1/order-lookups` and shows NOTFOUND (the order exists only in the browser's old key; the shop does not know it, and the page does not match it in the browser). The request log holds no `GET /api/v1/orders…` request.
Traces: SC-29 (story §9 "the design's mock key is dropped"; ORD-021 item 1)
Check: interact_page

---

## D. The order opened by a lookup (SC-24, SC-25)

These two cases need the real Order page (plan item 6, ambiguity 8). They prove the hand-off from `/track` end to end, not the page's own drawing.

**Scenario**: TC-39 — a shipped order opens with its carrier, tracking number and expected date, and none of its private details
**Given**: SEED-SAVE; SETSHIP (`status` `shipped`, `carrier` `GHN`, `tracking` `GHN-PT27XM`, `eta` `2026-11-26`); SEED-SET(`email = 'seed.private@example.com'`, `note = 'Leave with the guard'`); PRIV read; RL-RESET; a fresh browser (no token); 1280×800.
**When**: On `/track` the right pair `DOI-24817` / `0938 410 552` is typed and sent; at `/orders/doi-24817` the h1, the `main ol` steps, `main dl`, the page's visible text and `document.documentElement.outerHTML` (every attribute) are read.
**Then**: The h1 (`[data-testid='order-status']`) reads `Your order is on its way.`. `main ol` has five `li` labelled `Placed`, `Confirmed`, `Packed`, `On its way`, `Delivered` in that order, and only `On its way` carries `aria-current="step"` (and shows the current-step ring the design draws: a non-`none` outline or ring on the `li` or its marker). `main dl` has `Carrier` → `GHN` and `Tracking number` → `GHN-PT27XM`. The text `Expected by 26/11/2026.` is on the page. None of PRIV's phone (in any grouping), `seed.private@example.com`, PRIV's address and `Leave with the guard` appears in the visible text or in any attribute of the page.
Traces: SC-24 (ORD-023 AC-5, item 2)
Check: interact_page

**Scenario**: TC-40 — a cancelled order opens with the cancelled sentence in place of the steps
**Given**: SEED-SAVE; SEED-SET(`status = 'cancelled'`) with the carrier, tracking and expected date of SETSHIP left in place; RL-RESET; a fresh browser.
**When**: The right pair is sent from `/track`; at `/orders/doi-24817` the page is read.
**Then**: The browser is at `/orders/doi-24817`. The page's text contains `This order was cancelled.`; there is no `main ol` with five steps and no element with `aria-current="step"`; the text `Expected by` is not on the page; the h1 does not contain `on its way`. SEED-RESTORE afterwards.
Traces: SC-25 (ORD-023 AC-6, item 2)
Check: interact_page

---

## E. The page around the form, the layout, accessibility, clean-up and the end-to-end journey (SC-26 … SC-30)

**Scenario**: TC-41 — `/track` shows its heading, lead, picture, two labelled fields and the button as the design draws them
**Given**: A fresh browser; 1280×800; RL-RESET.
**When**: `/track` is opened and the h1, the element that follows it, `main img`, the two labels and inputs, and `[data-testid='track-submit']` are read.
**Then**: The h1 reads `Track an order`; the element after it reads exactly `Type the order number from your confirmation message and the phone number you ordered with.`; a picture in `main` has `naturalWidth` above 0. A visible label `Order number` is linked to an input (the label's control is that input) whose `placeholder` is `DOI-24817`; a visible label `Phone` is linked to an input whose `type` is `tel`. The button's text is `Find my order` and it holds one `svg` (the arrow) after the text. `[data-testid='track-error']` does not exist before any try.
Traces: SC-26 (ORD-020 AC-1, item 1)
Check: interact_page

**Scenario**: TC-42 — the five step rows are closed on load and each opens onto its sentence
**Given**: A fresh browser; 1280×800.
**When**: `/track` is opened; the heading `What each step means` and the five `details` under it are read; each `summary` is pressed in turn (and read) .
**Then**: The heading `What each step means` is shown; the five `details` are in the order `Placed`, `Confirmed`, `Packed`, `On its way`, `Delivered` and none has the `open` attribute on load. Pressing a row opens it onto exactly: `Placed` → `We have the order and are checking the pair against it.`; `Confirmed` → `Paid online; the pair is set aside for you.`; `Packed` → `Boxed with its tags and our receipt, waiting for the courier.`; `On its way` → `With GHN or GHTK; the tracking number is on the order's page.` (the apostrophe as the SRS writes it, U+0027); `Delivered` → `In your hands. If anything is wrong with the delivery, message us.`.
Traces: SC-26 (ORD-020 AC-2, item 2)
Check: interact_page

**Scenario**: TC-43 — "While you wait": the line, the "Message us" pill and the shop's five questions, closed
**Given**: The shop holds one question of topic `orders` and four of `shipping` (`GET /api/v1/faqs?topic=orders` has `total` 1 and `GET /api/v1/faqs?topic=shipping` has `total` 4; their `question` and `answer` are read); a fresh browser; 1280×800.
**When**: `/track` is opened; the heading `While you wait`, the line, the pill and the `details` in that section are read; each `summary` is pressed in turn; the pill is pressed.
**Then**: The heading `While you wait` is shown with the line exactly `Most orders arrive in two to four working days. If yours is late or the number is not found, message us with the order number.`. A link `Message us` with `href` `/contact` is there; pressing it goes to `/contact`. The section holds exactly five `details`, all closed on load, whose summaries equal the five `question`s read from the shop, and each opens onto its own `answer`.
Traces: SC-26 (ORD-020 AC-3, AC-4, item 4)
Check: interact_page

**Scenario**: TC-44 — one column on a phone with the form above the step rows; side by side from 1024 px; no sideways scroll with the longest message and six orders
**Given**: A fresh browser each time; STORE with the six entries of TC-33; for the message, STUB(`POST /api/v1/order-lookups`) answers 429 so TOOMANY shows after a press with `DOI-24817` and `0938 410 552`.
**When**: `/track` is opened at 1280×800, 1024×800, 1023×800, 390×844 and 320×640; at 390 and 320 the message is shown (both fields typed, button pressed). In each: the boxes of the form (`form`), of the heading `What each step means`, of the five step rows, of `[data-testid='track-error']` and of `[data-testid='track-orders']`, and `document.documentElement.scrollWidth` and `window.innerWidth` are read.
**Then**: At 390 and 320 the form's left edge equals the left edge of the heading `What each step means` (within 1 px: one column) and the form's bottom is above that heading's top; `scrollWidth` is at most `innerWidth` at both sizes while TOOMANY and the six-order list are shown, and the alert's right edge is within the window. At 1023 the form is above the step rows (its bottom above the heading's top). At 1024 and 1280 the form's right edge is at or before the step column's left edge and the form's top is within the vertical span of the step rows (the form at the left, the rows at the right).
Traces: SC-27 (ORD-020 AC-5; Responsive NFR)
Check: interact_page

**Scenario**: TC-45 — labels, Enter, Tab order, visible focus, announced messages and the button's own label
**Given**: A fresh browser; 1280×800; STORE with the six entries of TC-33; RL-RESET; the real shop for the wrong pair, STUBs for the other messages.
**When**: (a) The two labels and inputs are read: each `label[for]` matches its input's `id`, the label's box is visible (width and height above 0, not clipped) and the input's accessible name is its label text. (b) `DOI-24817` and `0938 410 553` are typed; Enter is pressed in the `Order number` field, then in the `Phone` field. (c) From the top of the page Tab is pressed repeatedly; the order in which the two inputs, the button and the five list links receive focus is recorded and, at each of those stops, the focused element's `outline-style`, `outline-width` and `box-shadow` are inspected. (d) The four messages are produced in turn — NOTFOUND (the real 404 of (b)), TOOMANY (STUB 429), FAILED (STUB 500), OFFLINE (browser offline) — and `[data-testid='track-error']` and its ancestors are read. (e) With DELAY(`POST /api/v1/order-lookups`, 3000) the button is pressed and its accessible name is read at 0.5 s.
**Then**: (a) both labels are linked and visible and each input's accessible name is `Order number` and `Phone`. (b) each Enter sends exactly one POST to `/api/v1/order-lookups` (two in all) and shows NOTFOUND. (c) focus reaches `Order number`, then `Phone`, then `[data-testid='track-submit']`, then, later in the page, the rows `DOI-30006`, `DOI-30005`, `DOI-30004`, `DOI-30003`, `DOI-30002` in that order (reading order); at every one of these stops either `outline-style` is not `none` with `outline-width` of at least 1px, or `box-shadow` is not `none`. (d) in each of the four, the line has `role="alert"` (or sits inside an element that has it) and its icon is decorative (`aria-hidden="true"`) or has an accessible name; no message is colour alone. (e) the button's accessible name is `Searching…`.
Traces: SC-28 (Accessibility NFR)
Check: interact_page

**Scenario**: TC-46 — the page no longer reads the mock or the orders collection, and the story's file rules hold
**Given**: The story's PR (the head being verified) and its diff (`get_pr`); `run_tests` for the frontend.
**When**: The diff and the files are read; `npm run lint` and `npm test` are run (`run_tests`, frontend).
**Then**: `src/pages/Track.tsx` has no import from `src/lib/mock/` and no `useCollection("orders", …)` (its only collection read is the shop's `faqs`, through the existing rows); the lookup is made through `src/lib/track-an-order-by-number-and-phone.ts` on `src/lib/api.ts` (`api.post`; no other client and no `fetch` of its own); the order is saved through `src/lib/orders.ts` (an additive export at most; one key, `doi:placed`); `src/lib/mock/orders.ts` is deleted when no file imports it afterwards; the PR edits none of `theme.css`, `index.css`, `package.json`, `App.tsx`, `Order.tsx`, `Checkout.tsx`; `content.json` gains only the four new `pages.track.*` keys (existing keys untouched) and the PR lists them; no file is over 300 lines; lint and the frontend tests pass. The running page's side of this rule is observed in TC-32 (no list of orders is read, no phone outside the POST) and TC-38 (the old key is not read). `manual` because no HTTP instrument reads a diff or a file list; `get_pr` and `run_tests` are the evidence.
Traces: SC-29 (overview §5; ORD-021 item 1)
Check: manual

**Scenario**: TC-47 — an order placed in one browser is found in a second browser with the number and the phone, and not in a third with a wrong phone
**Given**: RESET; browser A fresh at 1280×800 (FORM: name `Lan Nguyen`, phone `0912345678`, street address `12 Example Street, Ward 1`, city `Da Nang`, note `Gate code 4821`); B and C fresh with nothing stored. If the verified build's checkout sends the browser to a payment address instead of the order page, the order is placed with PLACE(`Lan Nguyen`, `0912345678`) and A's reading is taken from `GET /api/v1/orders/doi-24818` with its token.
**When**: In A: the pair is added from `/shop/samba-og-w-core-black` (size pill and `[data-testid='add-to-bag']`), `/checkout` is opened, FORM typed and `[data-testid='place-order']` pressed; at `/orders/doi-24818?placed=1` the number, `[data-testid='order-total']` (TOTAL) and the `Delivering to` row are read. In B: `/track` is opened, `24818` and `+84 912 345 678` typed, the button pressed; the order page is read; `/track` is opened again. In C: `/track` is opened, `24818` and `0912345679` typed, the button pressed.
**Then**: A's order is `DOI-24818`. In B the address becomes `/orders/doi-24818`; the page shows `DOI-24818`, `[data-testid='order-total']` equals TOTAL, and `Delivering to` reads `Lan Nguyen, Da Nang` exactly as in A; B's reopened `/track` lists a row `DOI-24818` linking to `/orders/doi-24818`. In C the address stays `/track`, `[data-testid='track-error']` reads NOTFOUND (with the same `outerHTML` as the one of TC-24), no list is drawn and `localStorage` holds no `doi:placed` entry.
Traces: SC-30 (ORD-021 AC-1, AC-12; ORD-009 AC-5)
Check: interact_page

---

## Coverage walk — every story criterion has a case

| Criterion | Cases | Criterion | Cases |
|---|---|---|---|
| SC-1 | TC-01 | SC-16 | TC-25, TC-26 |
| SC-2 | TC-02 | SC-17 | TC-27, TC-28 |
| SC-3 | TC-03 | SC-18 | TC-29, TC-30 |
| SC-4 | TC-04 | SC-19 | TC-31 |
| SC-5 | TC-05, TC-06 | SC-20 | TC-32 |
| SC-6 | TC-07 | SC-21 | TC-33, TC-34, TC-35 |
| SC-7 | TC-08, TC-09, TC-10 | SC-22 | TC-36 |
| SC-8 | TC-11 | SC-23 | TC-37 |
| SC-9 | TC-12, TC-13 | SC-24 | TC-39 |
| SC-10 | TC-14 | SC-25 | TC-40 |
| SC-11 | TC-15 | SC-26 | TC-41, TC-42, TC-43 |
| SC-12 | TC-16, TC-17, TC-18, TC-19 | SC-27 | TC-44 |
| SC-13 | TC-22 | SC-28 | TC-45 |
| SC-14 | TC-23 | SC-29 | TC-38, TC-46 |
| SC-15 | TC-24 | SC-30 | TC-47 |
| Contract (`POST /v1/order-lookups`, §2.2, §2.5) | TC-16, TC-17, TC-18, TC-19, TC-20, TC-21 | | |

**Criteria walked: 30 of 30** (SC-1 … SC-30).

**The SRS criteria the story's list narrows, walked once more.** ORD-020 AC-1 TC-41; AC-2 TC-42; AC-3 TC-43; AC-4 TC-43; AC-5 TC-44. ORD-021 AC-1 TC-01, TC-22; AC-2 TC-02, TC-23; AC-3 TC-02, TC-23; AC-4 TC-04, TC-24; AC-5 TC-04, TC-24; AC-6 TC-04; AC-7 TC-03, TC-24; AC-8 TC-04, TC-24; AC-9 TC-01; AC-10 TC-31; AC-11 TC-25, TC-26; AC-12 TC-22, TC-47. ORD-022 AC-1 TC-07; AC-2 TC-07, TC-27; AC-3 TC-13; AC-4 TC-08; AC-5 TC-09; AC-6 TC-10; AC-7 TC-11; AC-8 TC-15; AC-9 TC-07, TC-12, TC-13; the Conflict row TC-14. ORD-023 AC-1 TC-33; AC-2 TC-36; AC-3 TC-33, TC-34; AC-4 TC-31; AC-5 TC-39; AC-6 TC-40; AC-7 TC-37. Non-functional: Privacy TC-01, TC-32, TC-39; Security TC-07 … TC-15; Responsive TC-44; Accessibility TC-45.

**Where a criterion names a failure it has its own case where the failure happens:** SC-3 and SC-4 (nine kinds of miss, TC-03, TC-04), SC-6 (the sixth try refused, TC-07), SC-7 (the address, the number, a missing number, TC-08 … TC-10), SC-8 (a right pair in the middle of failures, TC-11), SC-9 (the wait counting down, then ending, TC-12, TC-13), SC-10 (ten at once, TC-14), SC-12 (malformed bodies, wrong media type, an oversized body, a failed query and an unreachable database, TC-16 … TC-19), SC-15 (the not-found line five ways, TC-24), SC-17 (the lock reached from the page, TC-27, TC-28), SC-18 (offline, an aborted call, a stopped API and three 5xx answers, TC-29, TC-30), SC-21 (no tokens, a damaged store, TC-34), SC-23 (storage refused, TC-37). Roles: a visitor (every case), a stranger with a wrong pair or a locked address (TC-04, TC-07 … TC-10), a customer and the shop team (TC-20, TC-21), the order's own holder not affected by a lock (TC-15).

**Instruments (47 cases).** `fetch_url` 21 (TC-01 … TC-21); `interact_page` 25 (TC-22 … TC-45 and TC-47: every case whose `When` is somebody typing, clicking, pressing a key, reloading or going back, or where a request is stubbed, delayed or aborted; the layout, opacity, focus and font readings are taken inside `interact_page` as computed values, the same readings `measure_styles` returns, because each needs the page in a state — a typed value, a stored list, a shown message — that a bare address cannot reach); `manual` 1 (TC-46: a source diff, a file list and the lint and test runs are not an HTTP response or a page; `get_pr` and `run_tests` are the evidence). Cases that depend on the verifier being able to intercept, delay or abort a request, run a script before the page's own, emulate offline, send ten requests at once, vary the network address or sign in say so in their own text; if the instrument cannot do it the case is UNVERIFIED with that reason, never failed.
