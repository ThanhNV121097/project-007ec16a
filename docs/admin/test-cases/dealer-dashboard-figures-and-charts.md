# Test cases — Dealer dashboard figures and charts

Module: `admin` · Plan item 11 · Story: `docs/admin/stories/dealer-dashboard-figures-and-charts.md` (SC-1 … SC-29)
Requirements: ADMIN-020, ADMIN-021, the overview rows of ADMIN-004 (`docs/admin/SRS.md` §4.3, §4.1) · Contract: `docs/architecture/services.md` §2.1, §2.2, §2.3, §2.9, §3.25

Written before the implementation exists, from the requirements, the story and the contract only.

## 1. Risk and depth

| Part | Risk | Why | Depth |
|---|---|---|---|
| Who may read the answer (SC-1, SC-2) | **High** | The route is the only place the shop's takings and stock are summed; a visitor or a customer who can read it learns the shop's revenue | No session, forged cookies, a signed-out cookie, a customer, a staff member, an admin; the guard answers before any figure is computed |
| The figures are right (SC-4 … SC-10, SC-15) | **High** | Money and stock a dealer acts on; cancelled orders, drafts, ties and the 15-pair and 1-or-2-pair limits are each a way to be silently wrong | The SRS's literals on the sample set, every boundary value of the two stock rules, multi-pair orders, lines with nothing sold, a shoe with no line, a cancelled order, a generated set checked against independent SQL |
| Vietnam time, month and day edges (SC-6, SC-11) | **High** | A server on UTC puts an order of 00:30 on the 1st into the old month; the figure is off for exactly the nights an owner looks at it | Orders one second either side of the month's start and of a day's midnight, written in `+07:00` and in UTC; the last day of the series against the Vietnam date of the response |
| Privacy and size (SC-16, SC-25) | **High** | The old page sent every customer's order to the browser; the story exists to stop that | 10,000 orders carrying sentinel phone / email / address / note; the body holds none and stays a few kilobytes; the page's traffic read |
| Performance (SC-17) | Medium | A stated bound at 10,000 orders and 1,000 shoes | 100 sequential reads, p95 |
| Absent data and failure (SC-12, SC-13, SC-26, SC-27) | Medium–High | A page that draws 0 for an answer it never got is worse than a page that says it failed | Empty tables, a failed query, a database that is gone; loading held for 3 s; 500, a dropped connection, 401 and 403 on the page; "Try again" succeeding and failing |
| The page's figures, charts and rows (SC-18 … SC-24) | Medium | A presentation of one answer; each wrong label is cheap to find once it is read | Every label word for word, every link's address, 14 columns in the answer's order, bars against the largest, the day labels under another time zone, empty lists |
| Phone layout and gate (SC-28, SC-29) | Medium | Most dealers' screens are phones; the gate is the story's second line of defence | 390 / 320 px, two-across and four-across breakpoints, menu sheet, 44 px targets; visitor, customer, reload |
| Source rules (S-1) | Low | Process rules checked once from the PR | One `manual` case |

## 2. How to read this file

- **Paths** are written as the backend mounts them (`/v1/admin/overview`); from a browser they are `/api/v1/admin/overview`.
- **Sessions** are named. `ADMIN`: the cookie from `POST /v1/auth/sign-in` with the run's `ADMIN_EMAIL` and `ADMIN_PASSWORD`. `STAFF`: the cookie of FX-STAFF (an active staff account). `CUSTOMER`: the cookie of FX-CUSTOMER. `NONE`: no cookie. `FORGED`: `Cookie: session=` followed by 43 characters `A` (the length of a real token, not one the shop issued); `FORGED-SHORT`: `session=forged`. In a browser case "signs in as X" means the run types the email and password on `/admin/sign-in` and presses "Sign in".
- **The one error body.** Every non-2xx answer is JSON `{"error":<code>,"message":<non-empty sentence>}` with `Content-Type: application/json; charset=utf-8`, and no `errors` key on 401, 403, 404 and 500 (`services.md` §2.3). A body never holds SQL, a table name, a stack trace, a path or a connection detail.
- **"Today", T.** The shop's calendar date in Vietnam time (UTC+7). A case reads it as the `date` of the last entry of `sales_by_day`, and SC-6a checks that entry against the response's own `Date` header. `T−1` is the day before, and so on.
- **Money** is a JSON integer of VND; on the page it is written in the shop's format, `47.660.000 ₫` (the space before `₫` may be a no-break space; the run compares after normalising white space).
- **"Delta" cases.** The shared database holds shoes, orders and lines the verifier did not make, so a case that needs a figure the fixture alone decides reads the answer **before** (the baseline B) and **after** arranging the fixture, and compares the difference; nothing else may change the shop between the two reads (FX-QUIET). The "sample set" cases use FX-SAMPLE instead, which makes the sample set the only live data.
- **Starting state of every case**: no row whose id starts `tc-ov-`; the orders and shoes are as the seed and the earlier stories left them. Anything a case creates is removed afterwards (`DELETE FROM orders WHERE id LIKE 'tc-ov-%'` — items go by cascade; `DELETE FROM products WHERE id LIKE 'tc-ov-%'`; `DELETE FROM lines WHERE id LIKE 'tc-ov-%'`; `DELETE FROM users WHERE email LIKE 'tc-%'`) and every row a case changed is put back, so no case depends on another. Arrangements go through `set_db_state`.
- **Browser requests stubbed, held or failed.** Where a case says the overview answer is replaced, held, failed or answered with a status, the verifier arranges it with request interception on `**/api/v1/admin/overview` inside the `interact_page` run (everything else — the cookie, `/auth/me`, the menu's count — stays real). A case that needs this and finds no interception available is UNVERIFIED for that reason, not failed. A case that sets the browser's time zone or reads the network log and finds no instrument for it is UNVERIFIED in the same way.
- **Elements are found by role, label or text, never by class**: the title is `main h1`; the left menu is `nav`; a figure is the link (`a`) that contains its label text; a panel is the section whose heading reads its title; the day chart is the `figure` inside "Sales, last 14 days" and each column is the element carrying `title="<day>: <amount>"` (the existing `AdminBars` markup); the line chart is the `figure` inside "Pairs sold by line" whose rows are `li`; a latest-orders row and a running-low row are the `a` whose `href` starts `/admin/orders/` or `/admin/products/` inside their panel. If the build carries a column's amount as visible text instead of a `title`, the column is found by that text.
- **Words** are checked word for word: "Overview"; "Sales this month", "<n> orders", "To pack", "placed or confirmed", "Pairs in stock", "across <n> shoes", "Running low", "last pairs left"; panels "Today", "Sales, last 14 days", "Pairs sold by line", "Latest orders", "Running low"; "Last pairs in <sizes>", "Nothing is running low."; "Loading the overview…" (with the single character `…`), "We could not load the overview.", "Try again"; "This area is for the shop's team", "Back to the shop".

## 3. Fixtures

| Id | What is made | Used by |
|---|---|---|
| FX-STAFF | `POST /v1/admin/staff` as ADMIN: name "Tc Staff", `tc-ov-staff@example.com`, password `staff-pass-A1` | SC-2, SC-18 |
| FX-CUSTOMER | As FX-STAFF with `tc-ov-customer@example.com`, then `UPDATE users SET role = 'customer' WHERE email = 'tc-ov-customer@example.com'` (the shop's own hash, so it signs in with that password) | SC-2, SC-29 |
| FX-QUIET | Between a baseline read and the read after the arrangement nothing else writes orders or shoes (no checkout, no other case running) | every delta case |
| FX-SAMPLE | The 14 seeded orders DOI-24804 … DOI-24817 are the only live orders, re-timed so that DOI-24816 (08:40) falls on T. Steps: (1) `CREATE TABLE tc_ov_bak AS SELECT id, status, placed_at FROM orders`; (2) `UPDATE orders SET status = 'cancelled', placed_at = placed_at - interval '3 years' WHERE id NOT IN ('doi-24804' … 'doi-24817')` (every other order drops out of every figure and out of the five latest); (3) `UPDATE orders SET placed_at = placed_at + D * interval '1 day' WHERE id IN (the 14)`, with D the whole days from 2026-11-25 to T. Restore: `UPDATE orders o SET status = b.status, placed_at = b.placed_at FROM tc_ov_bak b WHERE o.id = b.id`, then `DROP TABLE tc_ov_bak`. **The month literals hold only while T is between 25/11/2026 and 30/11/2026**, because DOI-24816 then stays in the month of every order; after that the case replaces them by the SQL sum of the shifted orders that fall in T's month, and says so | SC-4 … SC-8, SC-15 |
| FX-ORDER(n, status, placed_at, total, items) | A test order made by copying every column of `doi-24804` except `id` / `number` (`tc-ov-<n>` / `TC-OV-<n>`), `status`, `placed_at`, `total`, `customer`, `phone`, `email`, `address`, `note`, and one `order_items` row per listed pair (`line_no`, `product_id`, `size`, `qty`, `unit_price`); any unique or counter column the schema asks for is given a fresh value. The `total` is chosen so the delta says which orders were counted (powers of two times 1.000.000) | SC-7 … SC-14 |
| FX-SHOE(name, status, sizes) | A shoe made by copying an existing `products` row with id `tc-ov-<name>`, the given `status` (`live` or `draft`) and its own `product_sizes` rows exactly as listed (`'40':1` means size `40` with 1 pair); nothing else of the shoe matters to the overview | SC-9, SC-10 |
| FX-ANS-A | The overview answer the page cases substitute (the SRS sample, §4 below) | SC-18, SC-20 … SC-24, SC-28 |
| FX-ANS-B | FX-ANS-A with `running_low` holding five shoes (§4) | SC-19 |
| FX-ANS-NOORD | FX-ANS-A with no orders: `month` 0 and 0, `waiting_orders` 0, `running_low` `[]`, 14 entries of 0, every line `pairs` 0, `latest_orders` `[]`; `pairs_on_shelf` 73 and `shoes` 3 as before | SC-22, SC-24 |
| FX-ANS-EMPTY | Everything zero or empty: `month` 0 / 0, `waiting_orders` 0, `pairs_on_shelf` 0, `shoes` 0, `running_low` `[]`, 14 entries of 0, `pairs_by_line` `[]`, `latest_orders` `[]` | SC-22, SC-23 |
| FX-BIG | 10,000 orders `tc-ov-big-<i>` (`generate_series`), every status, placed over the last 120 days, each carrying the sentinels phone `0999000TCOV`, email `tcov@example.invalid`, address `TCOV ADDRESS`, note `TCOV NOTE`, one to three pairs; plus 1,000 shoes `tc-ov-big-<i>` with 5 to 12 sizes each, every size holding 3 to 9 pairs (so none of them is running low and the low-stock list is the one it was). The five newest orders of the shop are five of the generated ones | SC-16, SC-17 |

## 4. The substituted answers

FX-ANS-A (the SRS sample at the SRS clock; `image` is the shoe's own `image` read from `GET /v1/products/<id>`):

```json
{
  "month": { "sales": 47660000, "orders": 13 },
  "waiting_orders": 2,
  "pairs_on_shelf": 73,
  "shoes": 3,
  "running_low": [
    { "id": "gazelle-indoor-solar-gold", "name": "Gazelle Indoor Solar Gold", "category": "men", "image": "<its image>", "stock": 5, "sizes_low": ["40", "41"] },
    { "id": "samba-og-kids-cloud-white", "name": "Samba OG Kids Cloud White", "category": "kids", "image": "<its image>", "stock": 9, "sizes_low": ["36"] }
  ],
  "sales_by_day": [
    { "date": "2026-11-12", "sales": 2790000 }, { "date": "2026-11-13", "sales": 3590000 },
    { "date": "2026-11-14", "sales": 3090000 }, { "date": "2026-11-15", "sales": 3790000 },
    { "date": "2026-11-16", "sales": 0 },       { "date": "2026-11-17", "sales": 2290000 },
    { "date": "2026-11-18", "sales": 5090000 }, { "date": "2026-11-19", "sales": 3290000 },
    { "date": "2026-11-20", "sales": 0 },       { "date": "2026-11-21", "sales": 5580000 },
    { "date": "2026-11-22", "sales": 3590000 }, { "date": "2026-11-23", "sales": 5380000 },
    { "date": "2026-11-24", "sales": 5890000 }, { "date": "2026-11-25", "sales": 3290000 }
  ],
  "pairs_by_line": [
    { "line": "samba", "name": "Samba", "pairs": 8 }, { "line": "gazelle", "name": "Gazelle", "pairs": 2 },
    { "line": "megaride", "name": "Megaride", "pairs": 2 }, { "line": "handball-spezial", "name": "Handball Spezial", "pairs": 1 },
    { "line": "adizero", "name": "Adizero", "pairs": 1 }
  ],
  "latest_orders": [
    { "id": "doi-24816", "number": "DOI-24816", "customer": "Trang", "product": "handball-spezial-w-earth-strata", "product_name": "Handball Spezial Earth Strata", "size": "37 1/3", "status": "placed", "placed_at": "2026-11-25T08:40:00+07:00", "total": 3290000 },
    { "id": "doi-24815", "number": "DOI-24815", "customer": "Long", "product": "megaride-bones-halo-green", "product_name": "Megaride Bones Halo Green", "size": "43 1/3", "status": "confirmed", "placed_at": "2026-11-24T10:12:00+07:00", "total": 5890000 },
    { "id": "doi-24814", "number": "DOI-24814", "customer": "Hương", "product": "samba-og-kids-cloud-white", "product_name": "Samba OG Kids Cloud White", "size": "37 1/3", "status": "packed", "placed_at": "2026-11-23T19:30:00+07:00", "total": 2290000 },
    { "id": "doi-24817", "number": "DOI-24817", "customer": "Hoàng Nam", "product": "samba-og-w-crystal-white", "product_name": "Samba OG Crystal White", "size": "39 1/3", "status": "shipped", "placed_at": "2026-11-23T16:05:00+07:00", "total": 3090000 },
    { "id": "doi-24813", "number": "DOI-24813", "customer": "Tuấn", "product": "gazelle-indoor-solar-gold", "product_name": "Gazelle Indoor Solar Gold", "size": "43 1/3", "status": "shipped", "placed_at": "2026-11-22T11:02:00+07:00", "total": 3590000 }
  ]
}
```

FX-ANS-B is FX-ANS-A with `running_low` holding, fewest pairs first: the two above, then `samba-og-w-core-black` (stock 11, `["38"]`), `megaride-s2-cloud-white` (stock 12, `["42"]`) and `adizero-evo-sl-charcoal` (stock 14, `["43 1/3", "44"]`).

## 5. Open points found while writing (reported, not resolved here)

1. **The month literals depend on the date.** The shop's clock is the real one; no setting moves it. The SRS's `47.660.000 ₫ over 13 orders` therefore holds on the running shop only when the sample orders are re-timed (FX-SAMPLE) and T is still in November 2026. The fixed-clock form (25/11/2026 12:00) and the 14-day window that crosses a month end are proved by the backend's own tests with a clock handed to the handler; they cannot be arranged here, so no case on the running shop claims them.
2. **"Ties in the shoes' own order."** The contract does not say what a shoe's own order is. SC-10b takes it as the order `GET /v1/products?limit=100` (no `sort`) lists them in; if TL rules another, that case changes.
3. **A database that is gone: 500 or 503.** Contract §3.25 and story SC-13 say 500 `internal` for a database failure, and `services.md` §2.3 puts "a database that cannot be reached" under `internal`; the verification brief reserves 503 for a dependency that refused the connection. SC-13b follows the contract (500 `internal`), and a 503 there is reported as a finding against the story's wording, not accepted silently. With no database even the session lookup fails first (`auth.Resolve` answers 500 on a failed lookup), so SC-13a — a failed query behind a working session — is the case that isolates this handler.
4. **An undefined query parameter.** The story (SC-3) says it "changes nothing" and `services.md` §2.9 ignores undefined query parameters; the brief's rule that an undefined *request field* is a 400 is about bodies, and this endpoint has none. SC-3b expects 200 and an identical body. If TL decides a query parameter is refused with 400, that is a one-line change.
5. **The time on a latest-orders row is the browser's.** `formatDateTime` formats in the browser's zone, so a dealer whose laptop is set to New York reads `24/11/2026, 20:40` for the order the shop placed at `25/11/2026 08:40`. Story decision 1 protects only the day labels of the chart. SC-23a is therefore run with the browser in `Asia/Ho_Chi_Minh`; whether the row's time should be pinned to Vietnam time is a question for PM.
6. **SC-25's last sentence is wider than the menu allows.** "Nothing in the page's traffic carries a customer's phone, email, address or note" cannot hold for the left menu's own `GET /v1/orders?…&limit=1`, which returns one order in the dealer view (phone, email, address, note included). SC-25b holds the overview answer and every request the page itself adds to none of those fields and to at most that one order, and reports the menu's read as it is.
7. **No singular form.** The "Today" sentence is a fixed template: with one waiting order it reads "1 orders are waiting …". SC-19b asserts the template verbatim; a plural rule would be new copy for PM.
8. **Zone-less stored times.** `placed_at` is `timestamptz`, so a literal written without a zone is read in the database session's zone and cannot be told apart from an offset one. ADMIN-020 item 3's "a seeded time without a zone is read as Vietnam time" is a seed matter, and is observed in SC-8a (DOI-24816 reads `…T08:40:00+07:00`); the arranged orders carry explicit offsets and UTC.
9. **Instruments.** Interception of one URL (hold, fail, answer), the browser's time zone, the network log and the response time of 100 requests are what the cases below ask of `interact_page` / `fetch_url`; a case whose instrument is missing is reported UNVERIFIED with that reason.
10. **The contract's example id.** `services.md` §3.25's example shows `"product": "handball-spezial-earth-strata"`; the seed's id is `handball-spezial-w-earth-strata`. The cases use the seed's.

## 6. Who may read it, and the shape of the answer (BE) — SC-1 … SC-3

**Scenario**: SC-1a — no session is refused with 401 and no figure
**Given**: `NONE`.
**When**: `GET /v1/admin/overview` with no cookie.
**Then**: Status 401; `Content-Type: application/json; charset=utf-8`; body exactly the error shape with `error` `unauthenticated` and a non-empty `message`, and no `errors` key; the body holds none of the keys `month`, `waiting_orders`, `pairs_on_shelf`, `shoes`, `running_low`, `sales_by_day`, `pairs_by_line`, `latest_orders`, and no digit-only figure.
Traces: SC-1 (ADMIN-004 AC-5; ADMIN-020 AC-14, "Not permitted"); contract (§2.2, §3.25)
Check: fetch_url

**Scenario**: SC-1b — a forged cookie and an ended session are refused with 401
**Given**: `FORGED`, `FORGED-SHORT`, and a third cookie `S`: ADMIN signs in, `S` is the cookie it gets, then `POST /v1/auth/sign-out` with `S` answers 204.
**When**: `GET /v1/admin/overview` is sent with each of the three cookies in turn.
**Then**: Each answers 401 `unauthenticated` in the error shape, none carries a figure, none answers 200, 403 or 500.
Traces: SC-1 (ADMIN-020 AC-14; ADMIN-004 AC-5)
Check: fetch_url

**Scenario**: SC-2a — a customer session is refused with 403 and no figure
**Given**: FX-CUSTOMER signed in (cookie `C`).
**When**: `GET /v1/admin/overview` with `C`.
**Then**: Status 403; error shape with `error` `forbidden` and a non-empty `message`, no `errors` key; none of the eight overview keys is in the body; the customer's own email is not in the body.
Traces: SC-2 (ADMIN-004 AC-7, AC-10; ADMIN-020 AC-15; ADMIN-002 AC-20)
Check: fetch_url

**Scenario**: SC-2b — an admin session and a staff session each get 200 with the same shape and the same figures
**Given**: ADMIN and FX-STAFF signed in; FX-QUIET.
**When**: `GET /v1/admin/overview` with ADMIN's cookie, then with FX-STAFF's, then with ADMIN's again.
**Then**: All three answer 200; the first and the third bodies are identical (nothing changed in between); the second body has exactly the same top-level key set and the same values as the first.
Traces: SC-2 (ADMIN-004 AC-7, AC-10; ADMIN-020 AC-15; ADMIN-002 AC-16)
Check: fetch_url

**Scenario**: SC-3a — the answer has exactly the contract's fields, whole numbers, `no-store` and a trace id
**Given**: ADMIN signed in.
**When**: `GET /v1/admin/overview`, once without and once with the header `X-Request-Id: tc-ov-trace-0001`.
**Then**: 200 with `Content-Type: application/json; charset=utf-8` and `Cache-Control: no-store`. The top-level keys are exactly `month`, `waiting_orders`, `pairs_on_shelf`, `shoes`, `running_low`, `sales_by_day`, `pairs_by_line`, `latest_orders` and no other. `month` has exactly `sales` and `orders`. Every `running_low` item has exactly `id`, `name`, `category`, `image`, `stock`, `sizes_low`; every `sales_by_day` item exactly `date`, `sales`; every `pairs_by_line` item exactly `line`, `name`, `pairs`; every `latest_orders` item exactly `id`, `number`, `customer`, `product`, `product_name`, `size`, `status`, `placed_at`, `total`. Every money and count value (`month.*`, `waiting_orders`, `pairs_on_shelf`, `shoes`, `stock`, `sales`, `pairs`, `total`) appears in the raw body as an integer with no decimal point, exponent or quotes. Every array is an array (never `null`). `date` matches `^\d{4}-\d{2}-\d{2}$`. The first answer carries an `X-Request-Id` header matching `^[A-Za-z0-9._-]{8,64}$`; the second echoes `tc-ov-trace-0001`.
Traces: SC-3 (ADMIN-020 item 1; `services.md` §3.25); contract (§2.1 trace header, §2.1 integer money)
Check: fetch_url

**Scenario**: SC-3b — an undefined query parameter changes nothing
**Given**: ADMIN signed in; FX-QUIET.
**When**: `GET /v1/admin/overview`; then `GET /v1/admin/overview?page=2&limit=1&month=2026-01&date=2026-01-01&sort=-total&q=samba&delay=3&x=1`; then `GET /v1/admin/overview` again.
**Then**: All three answer 200 and the three bodies are identical (same keys, same values, same order); the parameters change neither the month, the 14 days, the lines nor the five latest orders.
Traces: SC-3 ("an unknown query parameter changes nothing"); contract (§2.9, undefined query parameter ignored)
Check: fetch_url

**Scenario**: C-1 — the route answers `GET` only; any other method is the catch-all 404 in the envelope
**Given**: ADMIN signed in.
**When**: `POST`, `PUT`, `DELETE` and `PATCH /v1/admin/overview`, each with `Content-Type: application/json` and body `{}`, then `GET /v1/admin/overview/extra`.
**Then**: Each answers 404 with `error` `not_found` in the error shape (never a plain-text 405, never HTML); none carries a figure; a following `GET /v1/admin/overview` still answers 200 with the same body as before.
Traces: contract (§2.3, `not_found`; the server's catch-all for a wrong method)
Check: fetch_url

## 7. The figures on the sample set (BE) — SC-4 … SC-10

**Scenario**: SC-4 — this month's sales and orders, the cancelled order in neither
**Given**: FX-SAMPLE (D days from 2026-11-25 to T), ADMIN signed in. T is between 25/11/2026 and 30/11/2026; if it is later, the expected pair is the SQL sum and count of the shifted non-cancelled sample orders that fall in T's month, and the report says so.
**When**: `GET /v1/admin/overview`.
**Then**: `month` is exactly `{"sales":47660000,"orders":13}`. It is not `{"sales":51750000,"orders":14}` (DOI-24811, cancelled, 4.090.000 ₫, is in neither figure).
Traces: SC-4 (ADMIN-020 AC-1, item 2)
Check: fetch_url

**Scenario**: SC-5 — orders waiting to be packed are the placed and confirmed ones
**Given**: FX-SAMPLE; ADMIN signed in.
**When**: `GET /v1/admin/overview`; then, one at a time, `UPDATE orders SET status = '<s>' WHERE id = 'doi-24816'` for `<s>` = `packed`, `shipped`, `delivered`, `cancelled`, `confirmed`, `placed` with a read after each (the last statement puts DOI-24816 back).
**Then**: The first read has `waiting_orders` 2 (DOI-24816 placed, DOI-24815 confirmed). After `packed`, `shipped`, `delivered` and `cancelled` it is 1 each time; after `confirmed` and after `placed` it is 2 again.
Traces: SC-5 (ADMIN-020 AC-2, item 1)
Check: fetch_url

**Scenario**: SC-6a — the 14 days run oldest first, and the last one is today in Vietnam
**Given**: FX-SAMPLE; ADMIN signed in.
**When**: `GET /v1/admin/overview`; the `Date` response header is read as a UTC instant and moved +7 hours to find the Vietnam calendar date.
**Then**: `sales_by_day` has exactly 14 entries; `date` of entry 14 equals the Vietnam calendar date of the header (T); entry 1 is `T−13`; the 14 dates are consecutive calendar days, each exactly one day after the one before, with no repeat. Entry `T` is 3290000, `T−1` 5890000, `T−2` 5380000, `T−3` 3590000, `T−4` 5580000, `T−5` 0 (DOI-24811's day, cancelled), `T−6` 3290000, `T−7` 5090000, `T−8` 2290000, `T−9` 0, `T−10` 3790000, `T−11` 3090000, `T−12` 3590000, `T−13` 2790000; the 14 amounts add up to 47660000.
Traces: SC-6 (ADMIN-020 AC-3, AC-6, item 1); SC-11 (item 3)
Check: fetch_url

**Scenario**: SC-6b — a day with no sale is present as 0 and a cancelled order's day stays 0
**Given**: FX-SAMPLE; ADMIN signed in.
**When**: `GET /v1/admin/overview`; then `UPDATE orders SET status = 'cancelled' WHERE id = 'doi-24810'` (T−6) and a second read; then the status is put back.
**Then**: In the first read the entries `T−9` and `T−5` are present with `sales` 0 (a number, not missing, not `null`); in the second read `T−6` is 0 and the other 13 entries equal those of the first read.
Traces: SC-6 (ADMIN-020 AC-3, AC-6)
Check: fetch_url

**Scenario**: SC-7a — pairs sold by line, most pairs first, equal counts in the lines' own order
**Given**: FX-SAMPLE; ADMIN signed in; `GET /v1/lines?limit=100` read to know the lines' own order.
**When**: `GET /v1/admin/overview`.
**Then**: `pairs_by_line` begins Samba 8, Gazelle 2, Megaride 2, Handball Spezial 1, Adizero 1 (each item `{"line":<id>,"name":<name>,"pairs":<n>}`; DOI-24812's quantity 2 is in Samba's 8; DOI-24811, cancelled, adds nothing to Adizero). Gazelle comes before Megaride and Handball Spezial before Adizero because that is their order in `GET /v1/lines`. Any further line follows with `pairs` 0. The list holds exactly one item per line of `GET /v1/lines`.
Traces: SC-7 (ADMIN-020 AC-4, item 4)
Check: fetch_url

**Scenario**: SC-7b — a two-pair order adds each pair's quantity to its own shoe's line
**Given**: FX-QUIET; baseline B read. FX-ORDER(1, `placed`, now − 1 minute, 6000000, [line_no 1 `samba-og-w-core-black` size `38` qty 1, line_no 2 `gazelle-indoor-solar-gold` size `42` qty 1]) and FX-ORDER(2, `placed`, now − 2 minutes, 3000000, [line_no 1 `samba-og-w-core-black` size `38` qty 3]).
**When**: `GET /v1/admin/overview` after the arrangement.
**Then**: Compared with B: Samba's `pairs` is +4 (1 + 3), Gazelle's +1, every other line's unchanged; `month.orders` is +2 and `month.sales` +9000000; `waiting_orders` +2.
Traces: SC-7 (ADMIN-020 AC-13, item 4)
Check: fetch_url

**Scenario**: SC-7c — a line nothing sold is present with 0; a shoe with no line and a cancelled order add to no row
**Given**: FX-QUIET; baseline B read. A line `tc-ov-line` named "Tc Line" with no shoe (`INSERT INTO lines` as a copy of an existing line with that id and name); a shoe `tc-ov-noline` (FX-SHOE, `live`, `'40':3`) whose `line_id` is NULL; FX-ORDER(3, `placed`, now − 1 minute, 1000000, [`tc-ov-noline` `40` qty 2]); FX-ORDER(4, `cancelled`, now − 2 minutes, 2000000, [`samba-og-w-core-black` `38` qty 5]).
**When**: `GET /v1/admin/overview`.
**Then**: `pairs_by_line` holds an item `{"line":"tc-ov-line","name":"Tc Line","pairs":0}`; the sum of all `pairs` equals the sum in B (the two pairs of the shoe with no line are in no row, and the five pairs of the cancelled order are in none); the line items are still ordered most pairs first.
Traces: SC-7 (ADMIN-020 AC-4, AC-13; "a shoe with no line is in no row"; "cancelled orders add nothing")
Check: fetch_url

**Scenario**: SC-8a — the five latest orders and the fields each carries
**Given**: FX-SAMPLE; ADMIN signed in.
**When**: `GET /v1/admin/overview`.
**Then**: `latest_orders` has exactly 5 items with `number` DOI-24816, DOI-24815, DOI-24814, DOI-24817, DOI-24813 in that order. The first is exactly `{"id":"doi-24816","number":"DOI-24816","customer":"Trang","product":"handball-spezial-w-earth-strata","product_name":"Handball Spezial Earth Strata","size":"37 1/3","status":"placed","placed_at":"<T>T08:40:00+07:00","total":3290000}`. Every item's `placed_at` matches `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+07:00$` and the five are in descending time. Every item's key set is the nine of SC-3a and nothing else.
Traces: SC-8 (ADMIN-020 AC-5, item 1, 2; item 3)
Check: fetch_url

**Scenario**: SC-8b — `product` and `size` are those of the order's first pair
**Given**: FX-QUIET; FX-ORDER(5, `placed`, now − 1 minute, 7000000, [line_no 1 `gazelle-indoor-solar-gold` size `42` qty 1, line_no 2 `samba-og-w-core-black` size `38` qty 2]).
**When**: `GET /v1/admin/overview`.
**Then**: `latest_orders[0]` has `id` `tc-ov-5`, `number` `TC-OV-5`, `product` `gazelle-indoor-solar-gold`, `product_name` equal to that shoe's `name` in `GET /v1/products/gazelle-indoor-solar-gold`, `size` `42`, `status` `placed`, `total` 7000000.
Traces: SC-8 (ADMIN-020 item 1; "`product` and `size` of the order's first pair")
Check: fetch_url

**Scenario**: SC-8c — a cancelled order among the five newest is listed as cancelled and counted nowhere
**Given**: FX-QUIET; baseline B read; FX-ORDER(6, `cancelled`, now − 1 minute, 4000000, [`samba-og-w-core-black` `38` qty 2]).
**When**: `GET /v1/admin/overview`.
**Then**: `latest_orders[0].id` is `tc-ov-6` with `status` `cancelled` and `total` 4000000; compared with B: `month`, `waiting_orders`, every `sales_by_day` entry and every `pairs_by_line` item are unchanged.
Traces: SC-8 ("a cancelled order that is among the five newest is listed"); SC-4 (ADMIN-020 item 2)
Check: fetch_url

**Scenario**: SC-9a — pairs on the shelf and the number of shoes
**Given**: FX-QUIET; baseline B read. FX-SHOE(`a`, live, `'40':6,'41':6,'42':6`) (18 pairs), FX-SHOE(`b`, live, `'40':8,'41':8,'42':8`) (24), FX-SHOE(`c`, live, `'40':10,'41':10,'42':11`) (31).
**When**: `GET /v1/admin/overview`.
**Then**: `pairs_on_shelf` is B's + 73 and `shoes` is B's + 3. Both equal the plain SQL results `SELECT sum(pairs) FROM product_sizes` and `SELECT count(*) FROM products`.
Traces: SC-9 (ADMIN-020 AC-10, item 1)
Check: fetch_url

**Scenario**: SC-9b — a shoe in draft is counted: its pairs and the shoe
**Given**: FX-QUIET; baseline B read; FX-SHOE(`draft10`, draft, `'40':5,'41':5`) (10 pairs, not visible to a visitor).
**When**: `GET /v1/admin/overview`.
**Then**: `pairs_on_shelf` is B's + 10 and `shoes` is B's + 1; the shoe is absent from `GET /v1/products` with no cookie (so the count includes what a visitor cannot see).
Traces: SC-9 (ADMIN-020 AC-11, item 1)
Check: fetch_url

**Scenario**: SC-10a — the running-low rule: a 1-or-2-pair size and fewer than 15 pairs in all
**Given**: FX-QUIET; baseline B read. FX-SHOE(`low5`, live, `'40':1,'41':2,'42':2`) (5 pairs, three low sizes); FX-SHOE(`low14`, live, `'40':2,'41':6,'42':6`) (14); FX-SHOE(`edge15`, live, `'40':1,'41':7,'42':7`) (15); FX-SHOE(`high18`, live, `'40':6,'41':6,'42':6`) (18, no low size); FX-SHOE(`plain9`, live, `'40':3,'41':3,'42':3`) (9, no low size); FX-SHOE(`size2`, live, `'40':2,'41':3,'42':3`) (8, a size of exactly 2); FX-SHOE(`size3`, live, `'40':3,'41':3`) (6, no size of 1 or 2); FX-SHOE(`draftlow`, draft, `'40':1,'41':2`) (3).
**When**: `GET /v1/admin/overview`.
**Then**: `running_low` contains `tc-ov-low5`, `tc-ov-low14`, `tc-ov-size2` and `tc-ov-draftlow` and none of `tc-ov-edge15` (15 is not fewer than 15), `tc-ov-high18`, `tc-ov-plain9`, `tc-ov-size3`. Their `stock` values are 5, 14, 8, 3, and the list is ordered by `stock` ascending (`tc-ov-draftlow` 3, `tc-ov-low5` 5, `tc-ov-size2` 8, `tc-ov-low14` 14, each placed among the shoes of B by that same order). The whole list equals the plain SQL result: shoes with `min(pairs) filter (pairs between 1 and 2)` present and `sum(pairs) < 15`, ordered by that sum.
Traces: SC-10 (ADMIN-020 AC-8, AC-9, item 1)
Check: fetch_url

**Scenario**: SC-10b — each running-low row's fields, and equal stock keeps the shoes' own order
**Given**: FX-QUIET; FX-SHOE(`tiea`, live, `'40':2,'41':5`) and FX-SHOE(`tieb`, live, `'40':2,'41':5`) (7 pairs each, made one after the other); `GET /v1/products?limit=100` read as ADMIN for the shoes' own order and for each shoe's `image`, `category`, `sizes_low` and `stock`.
**When**: `GET /v1/admin/overview`.
**Then**: Every `running_low` item has exactly `id`, `name`, `category`, `image`, `stock`, `sizes_low`; for each, `category` equals the shoe's category id, `image`, `stock` and `sizes_low` equal the values `GET /v1/products/<id>` gives, `sizes_low` is an array of strings (`["40"]` for the two tie shoes), and `stock` is an integer. The two tie shoes are adjacent and appear in the same relative order as in `GET /v1/products?limit=100`.
Traces: SC-10 (ADMIN-020 item 1; "each with id, name, category, picture, pairs in stock and last-pairs sizes"; "equal `stock` keeps the shoes' own order")
Check: fetch_url

## 8. Time, boundaries and absence (BE) — SC-11 … SC-13

In these cases S is the first instant of the current month in Vietnam time: `date_trunc('month', now() AT TIME ZONE 'Asia/Ho_Chi_Minh') AT TIME ZONE 'Asia/Ho_Chi_Minh'`, written to the database either with the `+07:00` offset or as the same instant in UTC (`+00`). The orders are `delivered`, so `waiting_orders` does not move. The runs avoid the first 30 minutes of a month (an order of "00:30 on the 1st" would then be in the future).

**Scenario**: SC-11a — an order at exactly the start of the month belongs to the new month, one second before belongs to the old one
**Given**: FX-QUIET; baseline B read. FX-ORDER(10, `delivered`, S − 30 minutes (23:30 on the last day of the previous month), 1000000), FX-ORDER(11, `delivered`, S + 30 minutes (00:30 on the 1st), 2000000), FX-ORDER(12, `delivered`, S written in UTC, 4000000), FX-ORDER(13, `delivered`, S − 1 second written with `+07:00`, 8000000).
**When**: `GET /v1/admin/overview` as ADMIN.
**Then**: `month.sales` is B's + 6000000 and `month.orders` B's + 2 (orders 11 and 12 are in; 10 and 13 are not). If the day of month of T is 14 or less, the 14-day series holds them too: the entry for the 1st is B's + 6000000 and the entry for the last day of the previous month is B's + 9000000.
Traces: SC-11 (ADMIN-020 AC-7, item 3; boundary "00:00:00 on the first of a month")
Check: fetch_url

**Scenario**: SC-11b — a day is cut at Vietnam midnight, whatever zone the stored value was written in
**Given**: FX-QUIET; baseline B read. Dates are in Vietnam time. FX-ORDER(20, `delivered`, `T−4` 23:59:59 with `+07:00`, 1000000); FX-ORDER(21, `delivered`, `T−3` 00:00:00, written in UTC as `(T−4) 17:00:00+00`, 2000000); FX-ORDER(22, `delivered`, `T−3` 23:59:59, written in UTC as `(T−3) 16:59:59+00`, 4000000); FX-ORDER(23, `delivered`, `T−2` 00:00:00 with `+07:00`, 8000000).
**When**: `GET /v1/admin/overview` as ADMIN.
**Then**: Against B, the entry `T−4` is +1000000, `T−3` is +6000000, `T−2` is +8000000, and the other 11 entries are unchanged; the 14 dates are still consecutive and end on T.
Traces: SC-11 (ADMIN-020 AC-7, item 3)
Check: fetch_url

**Scenario**: SC-12a — with no orders, no shoes and no lines the answer is zeros and empty arrays, not an error
**Given**: Every table the deletes touch is copied first (`CREATE TABLE tc_ov_bak_<name> AS TABLE <name>` for `orders`, `order_items`, `order_tokens`, `payment_attempts`, `reviews`, `product_sizes`, `products`, `lines`, and any other table with a foreign key to `orders`, `products` or `lines`); then `DELETE FROM orders; DELETE FROM reviews; DELETE FROM products; DELETE FROM lines` (or the order the foreign keys need). ADMIN's session is untouched. The copies are restored in foreign-key order afterwards and dropped. If the wipe or the restore cannot be done safely on the shared database, SC-12a is UNVERIFIED for that reason and SC-12b still runs.
**When**: `GET /v1/admin/overview` as ADMIN.
**Then**: 200 with `month` `{"sales":0,"orders":0}`, `waiting_orders` 0, `pairs_on_shelf` 0, `shoes` 0, `running_low` `[]`, `sales_by_day` 14 entries each with `sales` 0 ending on T, `pairs_by_line` `[]`, `latest_orders` `[]`; no key is `null`.
Traces: SC-12 (ADMIN-020 AC-12; "Absent data")
Check: fetch_url

**Scenario**: SC-12b — with shoes and lines but no orders every line is present with 0
**Given**: The same copies of `orders`, `order_items`, `order_tokens`, `payment_attempts` are taken and `DELETE FROM orders` is run; shoes, lines and sizes stay.
**When**: `GET /v1/admin/overview` as ADMIN.
**Then**: 200; `month` `{"sales":0,"orders":0}`; `waiting_orders` 0; `sales_by_day` has 14 entries all 0; `latest_orders` is `[]`; `pairs_by_line` has exactly one item per line of `GET /v1/lines?limit=100`, each with `pairs` 0, in the lines' own order; `pairs_on_shelf` and `shoes` equal the SQL sums over `product_sizes` and `products`.
Traces: SC-12 (ADMIN-020 AC-12; "With shoes and lines but no orders, every line is present with 0")
Check: fetch_url

**Scenario**: SC-13a — a failed query answers 500 with the generic envelope, and the next read recovers
**Given**: ADMIN signed in; `ALTER TABLE product_sizes RENAME TO tc_ov_product_sizes` (the aggregates read it; the session tables are untouched).
**When**: `GET /v1/admin/overview`; then the table is renamed back and `GET /v1/admin/overview` is sent again.
**Then**: The first answers 500 with `Content-Type: application/json; charset=utf-8`, `error` `internal`, a non-empty generic `message`, no `errors` key. The body holds none of the words `product_sizes`, `tc_ov_product_sizes`, `order_items`, `orders`, `relation`, `SELECT`, `FROM`, `pq:`, `pgx`, `SQLSTATE`, `goroutine`, `.go:` and no figure. The second answers 200 with the full shape.
Traces: SC-13 (ADMIN-020 "Upstream failure"); contract (§2.3 `internal`, §3.25)
Check: fetch_url

**Scenario**: SC-13b — a database that cannot be reached answers 500 `internal`, not a hang and not a figure
**Given**: ADMIN signed in; `set_db_available` takes the database away while the backend keeps running.
**When**: `GET /v1/admin/overview` with ADMIN's cookie; then the database is made available again and the request is repeated.
**Then**: While it is away the answer is 500 with `error` `internal` and a generic non-empty `message` (a 503 is reported as a finding — see §5 point 3), the body holds no SQL, no table name, no host or port, no figure, and the answer arrives within 15 seconds; after the database is back the request answers 200 with the full shape.
Traces: SC-13 (ADMIN-020 "Upstream failure"); contract (§2.3 `internal`)
Check: fetch_url

**Scenario**: SC-13c — the server logs the cause of a failed read
**Given**: The server log of the run, after SC-13a.
**When**: The log lines for the request id of the 500 answer of SC-13a are read.
**Then**: A line carries that `X-Request-Id`, the route `GET /v1/admin/overview` and the underlying error (the missing relation), and the response body of SC-13a carried none of it. `manual` because no instrument reads the server's log.
Traces: SC-13 ("the server logs the cause"); contract (§2.3, "the cause is logged with the `X-Request-Id`")
Check: manual

## 9. Sums, change and size (BE) — SC-14 … SC-17

**Scenario**: SC-14a — an order placed through checkout is in the next read
**Given**: FX-QUIET; ADMIN signed in; baseline B read. Checkout is called as a visitor (no cookie): `POST /v1/orders` with `Content-Type: application/json`, the header `Idempotency-Key: tc-ov-14a-0001` (8 to 100 characters of letters, digits, dot, dash or underscore; without it the shop answers 400 `bad_request`, `_base.md` §2.6) and the body of `_base.md` §3.10 `{"items":[{"product":"samba-og-w-core-black","size":"38","qty":1}],"delivery":"standard","payment":"vnpay","customer":"Tc Ov","phone":"0912 345 678","address":"12 Example Street","city":"Da Nang","note":""}`. `vnpay` is the only way to pay the shop offers: `cod`, `bank` and `momo` answer 422 with `errors.payment` "not offered", so none of them can place an order here. The size `38` has pairs on the shelf. Two ways the call can go, and the case says which one it saw:
  (1) **The run has the VNPay merchant settings** (`VNPAY_TMN_CODE`, `VNPAY_HASH_SECRET`): the call answers 201 with `order.id`, `order.number`, `order.status` `placed` and `order.total`, plus `token` and `payment_url`.
  (2) **The run has no VNPay settings**: the call answers 503 with `error` `payment_unavailable` (the keys are checked after the fields and before the transaction, `_base.md` §3.10), no order is made and no pair is taken. The run then makes the order itself: FX-ORDER(14, `placed`, now − 1 minute, the price `GET /v1/products/samba-og-w-core-black` gives now, [`samba-og-w-core-black` `38` qty 1]) with `UPDATE product_sizes SET pairs = pairs - 1 WHERE product_id = 'samba-og-w-core-black' AND size = '38'`, as the shop's own placing would have done.
**When**: `GET /v1/admin/overview` as ADMIN.
**Then**: Against B: `month.orders` +1, `month.sales` + the order's `total`, `waiting_orders` +1 (a `vnpay` order unpaid is `placed`), the entry `T` + the `total`, Samba's `pairs` +1; `latest_orders[0].id` is the new order's `id` (path 1: the 201's `order.id`; path 2: `tc-ov-14`) with `status` `placed` and the same `total`. Afterwards the order is deleted (its `order_items` and `order_tokens` go with it; a `payment_attempts` row, if the 201 made one, is deleted first) and the pair put back on the shelf (`UPDATE product_sizes SET pairs = pairs + 1 WHERE product_id = 'samba-og-w-core-black' AND size = '38'`). **Report**: path 1 is a PASS or a FAIL on the figures. On path 2 the 503 is the reason the *placing through checkout* half of SC-14 is not observed here: the case is reported UNVERIFIED with the instrument it lacked (the VNPay keys that let `POST /v1/orders` answer 201 — the verifier cannot set them) and the figures observed on the inserted order are named in `why`; SC-14b and SC-14c, which arrange their orders by SQL, still prove the overview's side.
Traces: SC-14 (ADMIN-020 AC-16); contract (`_base.md` §2.6 `Idempotency-Key`, §3.10 `payment`, `payment_unavailable`)
Check: fetch_url

**Scenario**: SC-14b — an order cancelled after being counted drops out of every figure and stays in the latest list as cancelled
**Given**: FX-QUIET; FX-ORDER(30, `placed`, now − 1 minute, 5000000, [`samba-og-w-core-black` `38` qty 2]); read R1. The cancel route of "Dealer handles orders" is used if it is on the running shop; otherwise `UPDATE orders SET status = 'cancelled' WHERE id = 'tc-ov-30'` (the case reports which).
**When**: `GET /v1/admin/overview` again (R2).
**Then**: R2 against R1: `month.orders` −1, `month.sales` −5000000, `waiting_orders` −1, the entry `T` −5000000, Samba's `pairs` −2; `latest_orders[0]` is `tc-ov-30` with `status` `cancelled` and `total` 5000000.
Traces: SC-14 (ADMIN-020 AC-17; ADMIN-020 item 2)
Check: fetch_url

**Scenario**: SC-14c — `/admin` opened again shows the new figures, with a fresh read and no manual reload of anything else
**Given**: The browser signs in as ADMIN and opens `/admin` at 1280×800 (no interception); the run reads the "Sales this month" figure and its note and counts the requests to `/api/v1/admin/overview`. Then, with the browser still signed in, FX-ORDER(31, `placed`, now − 1 minute, 5000000, [`samba-og-w-core-black` `38` qty 1]) is made by SQL.
**When**: The run follows the left menu's "Orders" link to `/admin/orders`, then the "Overview" link back to `/admin`, and reads the figures and the request count again.
**Then**: Coming back sent one new `GET /api/v1/admin/overview` (the count is up by one — nothing from the earlier visit is drawn first and no reload of the document was needed); "Sales this month" is the earlier amount + `5.000.000 ₫` and its note holds one order more; "To pack" is one higher; the first row of "Latest orders" reads `TC-OV-31`.
Traces: SC-14 (ADMIN-020 AC-16; ADMIN-021 item 3; story decision 5)
Check: interact_page

**Scenario**: SC-15a — the sample set's figures equal independent sums
**Given**: FX-SAMPLE; ADMIN signed in. The expected values are computed by the verifier's own SQL (not the handler's): `month` over `placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh'` in T's month, each of the 14 days, each line's pairs `sum(order_items.qty)` over non-cancelled orders joined to `products.line_id`, `waiting_orders` — and the same figures summed from the dealer's own list `GET /v1/orders?limit=100` (all pages, as ADMIN).
**When**: `GET /v1/admin/overview`.
**Then**: `month.sales`, `month.orders`, each of the 14 `sales_by_day` amounts, each line's `pairs` and `waiting_orders` equal the SQL sums and the sums from the order list, field by field.
Traces: SC-15 (ADMIN-020 AC-19, item 1)
Check: fetch_url

**Scenario**: SC-15b — a generated set of several hundred orders equals independent sums
**Given**: FX-QUIET; 400 orders `tc-ov-gen-<i>` generated by `generate_series`: `placed_at` spread over the last 90 days (about four a day, some of them in the last 14 days and in this month and in the previous two), every status in turn (`placed`, `confirmed`, `packed`, `shipped`, `delivered`, `cancelled`), one to three `order_items` rows each with `qty` 1 to 3 over at least six different shoes of at least three lines, `total` = Σ `qty × unit_price`. The expected values are the verifier's own SQL as in SC-15a.
**When**: `GET /v1/admin/overview` as ADMIN.
**Then**: `month.sales`, `month.orders`, each of the 14 days, each line's `pairs` and `waiting_orders` equal the SQL sums, field by field; the five `latest_orders` equal the five newest rows of `orders` by `placed_at` (any status).
Traces: SC-15 (ADMIN-020 AC-19, item 1)
Check: fetch_url

**Scenario**: SC-16 — 10,000 orders: the answer holds only the five latest, no customer detail, and stays small
**Given**: Baseline length L0 = the byte length of the body of `GET /v1/admin/overview` read as ADMIN with no FX-BIG data. FX-BIG is made with every size of the generated shoes holding 3 to 9 pairs (so none of them is running low); the sentinels are `0999000TCOV`, `tcov@example.invalid`, `TCOV ADDRESS`, `TCOV NOTE`; the five newest orders are five of the generated ones.
**When**: `GET /v1/admin/overview` as ADMIN, the body length L1 is taken, and the body is searched.
**Then**: `latest_orders` has exactly 5 items, all five from the generated set's newest; no item's key set differs from the nine of SC-3a; the body contains none of the four sentinels, none of the keys `phone`, `email`, `address`, `note`, `city` and no `@`; the body contains no order number other than the five (no `TC-OV-BIG-` number beyond the five latest); `|L1 − L0| ≤ 1024` and `L1 ≤ 12288`.
Traces: SC-16 (ADMIN-020 AC-18; Privacy NFR)
Check: fetch_url

**Scenario**: SC-17 — p95 of at most 500 ms with 10,000 orders and 1,000 shoes
**Given**: FX-BIG (10,000 orders, 1,000 shoes, every generated size at 3 to 9 pairs); ADMIN signed in.
**When**: 100 sequential `GET /v1/admin/overview` requests (the first is not discarded), the response time of each as the instrument reports it.
**Then**: Every one answers 200; sorted by time, the 95th of the 100 durations is at most 500 ms. If the instrument reports no time, or cannot issue 100 sequential requests, the case is UNVERIFIED and the report says which instrument it lacked.
Traces: SC-17 (Performance NFR; `services.md` §3.25)
Check: fetch_url

## 10. The page's figures and links (UI+BE) — SC-18 … SC-20

Every case of §10 to §13 runs in a browser at 1280×800, signed in as ADMIN, unless it says otherwise. Where it says "FX-ANS-x is substituted", the run intercepts `GET **/api/v1/admin/overview` and answers it 200 with that body, and the gate's `/api/v1/auth/me` and the menu's own read stay real.

**Scenario**: SC-18a — the title and the four figures read the answer
**Given**: FX-ANS-A is substituted. The browser signs in as ADMIN.
**When**: The run opens `/admin` and reads `main h1`, then the four figures.
**Then**: `main h1` reads "Overview". Exactly four figure links are shown, in this order, each containing its label, value and note: "Sales this month" / `47.660.000 ₫` / "13 orders"; "To pack" / `2` / "placed or confirmed"; "Pairs in stock" / `73` / "across 3 shoes"; "Running low" / `2` / "last pairs left". The page text contains none of `NaN`, `undefined`, `null`, `[object`. "Loading the overview…" and "We could not load the overview." are absent.
Traces: SC-18 (ADMIN-021 AC-1 to AC-4, item 1)
Check: interact_page

**Scenario**: SC-18b — a staff session sees the same page and figures
**Given**: FX-STAFF exists; FX-ANS-A is substituted. The browser signs in as FX-STAFF (`tc-ov-staff@example.com`).
**When**: The run opens `/admin` and reads `main h1` and the four figures.
**Then**: The page is the same as SC-18a: "Overview" and the four figures with the same labels, values and notes; no "This area is for the shop's team".
Traces: SC-18 (ADMIN-002 AC-16, AC-20; ADMIN-021 AC-1 to AC-4)
Check: interact_page

**Scenario**: SC-18c — on the running shop each figure equals the same field of the endpoint
**Given**: No interception. ADMIN's `GET /v1/admin/overview` is read in the same run as the page (FX-QUIET) to give `month`, `waiting_orders`, `pairs_on_shelf`, `shoes` and `running_low`.
**When**: The browser signs in as ADMIN, opens `/admin` and reads the four figures.
**Then**: "Sales this month" reads `month.sales` in the shop's money format with the note `<month.orders> orders`; "To pack" reads `waiting_orders` with "placed or confirmed"; "Pairs in stock" reads `pairs_on_shelf` with `across <shoes> shoes`; "Running low" reads the length of `running_low` with "last pairs left".
Traces: SC-18 (ADMIN-021 AC-1 to AC-4; "on the running shop each value equals the same field of `GET /v1/admin/overview`")
Check: interact_page

**Scenario**: SC-19a — "Today" reads the sentence with the answer's numbers
**Given**: FX-ANS-B is substituted (2 waiting, 5 running low, 13 orders, `47.660.000 ₫`).
**When**: The run opens `/admin` and reads the panel "Today" and the four figures.
**Then**: The panel "Today" has its heading "Today" and the text exactly `2 orders are waiting to be packed and 5 shoes are down to their last pairs. This month so far: 13 orders, 47.660.000 ₫.`; the "To pack" figure reads 2, the "Running low" figure reads 5, the "Sales this month" figure reads `47.660.000 ₫` with the note "13 orders" — the sentence's three numbers and its amount are the figures' own.
Traces: SC-19 (ADMIN-021 AC-6, item 2; story decision 7)
Check: interact_page

**Scenario**: SC-19b — the sentence follows another answer and keeps its template
**Given**: A third answer is substituted: FX-ANS-A with `waiting_orders` 1, `month` `{"sales":3290000,"orders":1}` and `running_low` holding only the first shoe.
**When**: The run opens `/admin` and reads "Today" and the four figures.
**Then**: "Today" reads exactly `1 orders are waiting to be packed and 1 shoes are down to their last pairs. This month so far: 1 orders, 3.290.000 ₫.` (the template is not reworded for one — §5 point 7); "To pack" reads 1, "Running low" reads 1, "Sales this month" reads `3.290.000 ₫` with "1 orders".
Traces: SC-19 (ADMIN-021 AC-6; "its counts always equal the figures above it")
Check: interact_page

**Scenario**: SC-20 — each figure opens the list it counts
**Given**: FX-ANS-A is substituted. The browser is signed in as ADMIN at `/admin`.
**When**: The run reads each figure link's `href`, then presses "Sales this month" and reads the address; goes back to `/admin`; presses "To pack" and reads the address; goes back; presses "Pairs in stock"; goes back; presses "Running low".
**Then**: The four `href` values are `/admin/orders`, `/admin/orders?status=confirmed`, `/admin/products` and `/admin/products` in that order; after each press the browser's address (path and query) is that same value.
Traces: SC-20 (ADMIN-021 AC-5, item 1)
Check: interact_page

## 11. The charts (UI+BE) — SC-21, SC-22

**Scenario**: SC-21a — 14 columns, one per day of the answer, labelled with the shop's day of the month, each carrying its amount
**Given**: FX-ANS-A is substituted.
**When**: The run opens `/admin` and reads the panel "Sales, last 14 days".
**Then**: The panel's heading reads "Sales, last 14 days". The chart has exactly 14 columns, in the answer's order; the day labels under them read, left to right, `12 13 14 15 16 17 18 19 20 21 22 23 24 25`; the column of `14` carries `3.090.000 ₫`, the column of `24` carries `5.890.000 ₫`, the column of `12` carries `2.790.000 ₫`, and the columns of `16` and `20` carry `0 ₫` (each amount in the shop's money format, as the column's `title` of the form `<day>: <amount>` or as visible text, together with its day).
Traces: SC-21 (ADMIN-021 AC-7, item 2; story decision 1)
Check: interact_page

**Scenario**: SC-21b — a day with no sale is a short pale column; a day with sales is a taller blue one in proportion to the largest
**Given**: FX-ANS-A is substituted; the run reads the computed `height` and `background-color` of each column's bar (the `span` inside the column) and of the chart's column area.
**When**: The run opens `/admin` and inspects the bars of the days `16`, `20`, `24`, `25`, `12`.
**Then**: The bar of `24` (the largest, 5.890.000) is as tall as the column area (within 1 px); the bar of `25` is 55.9 % of it (3.290.000 ÷ 5.890.000, within ±2 percentage points); the bar of `12` is 47.4 % (±2). The bars of `16` and `20` are at most 6 px high and their `background-color` differs from the `background-color` of the bar of `24`; the bars of `12`, `24`, `25` all share one `background-color`, which is also the `background-color` of the filled part of the first row of "Pairs sold by line" (the one blue of the area).
Traces: SC-21 (ADMIN-021 AC-7, item 2; "a short pale column … a taller blue one in proportion to the largest")
Check: interact_page

**Scenario**: SC-21c — the day labels are the shop's, whatever zone the browser is in
**Given**: FX-ANS-A is substituted. The browser's time zone is set to `America/New_York`; the same run is repeated with `Australia/Sydney` and with `Asia/Ho_Chi_Minh`.
**When**: The run opens `/admin` in each zone and reads the day labels of "Sales, last 14 days".
**Then**: In all three zones the labels read `12 13 14 … 25` and the column amounts are the same as in SC-21a (in New York a page that turned `2026-11-12` into a local `Date` would read 11 for the first column). If the instrument cannot set the browser's time zone the case is UNVERIFIED and the report says so.
Traces: SC-21 (story decision 1; ADMIN-021 AC-7)
Check: interact_page

**Scenario**: SC-22a — "Pairs sold by line": one row per line in the answer's order, each with a bar and its count
**Given**: FX-ANS-A is substituted.
**When**: The run opens `/admin` and reads the panel "Pairs sold by line" and the computed width of each row's filled bar relative to its track.
**Then**: The heading reads "Pairs sold by line". There are exactly five rows, in this order, each with its name at the left and its count at the right: Samba 8, Gazelle 2, Megaride 2, Handball Spezial 1, Adizero 1. The filled bars are 100 %, 25 %, 25 %, 12.5 % and 12.5 % of their tracks (within 1 percentage point).
Traces: SC-22 (ADMIN-021 AC-8, item 2)
Check: interact_page

**Scenario**: SC-22b — a line of 0 shows "0" and an empty bar; with no lines the panel is present and empty
**Given**: First FX-ANS-A with a sixth line `{"line":"terrex","name":"Terrex","pairs":0}` appended to `pairs_by_line` is substituted; then FX-ANS-EMPTY.
**When**: The run opens `/admin` with each and reads the panel "Pairs sold by line".
**Then**: With the first: the sixth row is `Terrex` with the text `0` at its right and a filled bar of computed width 0 px, below Adizero. With FX-ANS-EMPTY: the heading "Pairs sold by line" is shown and the panel holds no row (no `li`), no error text and no "0".
Traces: SC-22 (ADMIN-021 AC-8; ADMIN-020 AC-12 on the page)
Check: interact_page

## 12. The rows (UI+BE) — SC-23, SC-24

**Scenario**: SC-23a — "Latest orders": five rows in the answer's order; the first reads as designed and opens its order
**Given**: FX-ANS-A is substituted. The browser's time zone is `Asia/Ho_Chi_Minh` (§5 point 5).
**When**: The run opens `/admin`, reads the panel "Latest orders", then presses its first row.
**Then**: The heading reads "Latest orders" and the panel holds exactly five row links in this order of numbers: DOI-24816, DOI-24815, DOI-24814, DOI-24817, DOI-24813, with customers Trang, Long, Hương, Hoàng Nam, Tuấn. The first row's left half reads `DOI-24816 · Trang · Handball Spezial Earth Strata · EU 37 1/3` and its right half `Placed · 25/11/2026, 08:40 · 3.290.000 ₫`; the second row's right half begins `Confirmed · 24/11/2026, 10:12 · 5.890.000 ₫`. Each row's `href` is `/admin/orders/<id>` (`doi-24816`, `doi-24815`, `doi-24814`, `doi-24817`, `doi-24813`). After pressing the first row the browser's address is `/admin/orders/doi-24816`.
Traces: SC-23 (ADMIN-021 AC-9, AC-10, item 2; ADMIN-020 item 2)
Check: interact_page

**Scenario**: SC-23b — a cancelled order among the five reads "Cancelled"
**Given**: FX-ANS-A with the third latest order's `status` changed to `cancelled` is substituted. The browser's time zone is `Asia/Ho_Chi_Minh`.
**When**: The run opens `/admin` and reads the third row of "Latest orders".
**Then**: Its right half reads exactly `Cancelled · 23/11/2026, 19:30 · 2.290.000 ₫`; the word `cancelled` in lower case is not shown anywhere in the panel; the row is still a link to `/admin/orders/doi-24814`.
Traces: SC-23 (ADMIN-021 AC-9, item 2; ADMIN-020 item 2)
Check: interact_page

**Scenario**: SC-23c — with no orders the panel has its title and no rows
**Given**: FX-ANS-EMPTY is substituted.
**When**: The run opens `/admin` and reads the panel "Latest orders".
**Then**: The heading "Latest orders" is shown; the panel holds no link whose `href` starts `/admin/orders/` and no `li`; it holds no placeholder text.
Traces: SC-23 (ADMIN-021 "Absent data"; ADMIN-020 AC-12)
Check: interact_page

**Scenario**: SC-24a — "Running low": each shoe's picture, name, last-pairs sizes and pairs left, as a link, in the answer's order
**Given**: FX-ANS-A is substituted (two shoes).
**When**: The run opens `/admin` and reads the panel "Running low" (the panel, not the figure of the same name).
**Then**: The panel holds two row links in this order: the first has an `img` whose `alt` is `Gazelle Indoor Solar Gold` and whose `src` is that shoe's image and which has loaded (`naturalWidth` > 0), the name `Gazelle Indoor Solar Gold`, the line `Last pairs in 40, 41` and the number `5`, and `href` `/admin/products/gazelle-indoor-solar-gold`; the second has the name `Samba OG Kids Cloud White`, `Last pairs in 36`, the number `9` and `href` `/admin/products/samba-og-kids-cloud-white`. "Nothing is running low." is absent.
Traces: SC-24 (ADMIN-021 AC-11, item 2)
Check: interact_page

**Scenario**: SC-24b — with no shoe running low the panel says so and the figure reads 0
**Given**: FX-ANS-NOORD is substituted (`running_low` `[]`).
**When**: The run opens `/admin` and reads the panel "Running low" and the "Running low" figure.
**Then**: The panel shows "Nothing is running low." and no row link whose `href` starts `/admin/products/`; the figure "Running low" reads `0` with "last pairs left".
Traces: SC-24 (ADMIN-021 AC-12, item 2)
Check: interact_page

**Scenario**: SC-24c — a shop with no orders draws figures of 0, short columns, every line at 0 and no rows
**Given**: FX-ANS-NOORD is substituted.
**When**: The run opens `/admin` and reads the four figures, "Today", the 14 columns' bars, "Pairs sold by line" and "Latest orders".
**Then**: "Sales this month" reads `0 ₫` with "0 orders"; "To pack" 0; "Pairs in stock" 73 with "across 3 shoes"; "Running low" 0. "Today" reads `0 orders are waiting to be packed and 0 shoes are down to their last pairs. This month so far: 0 orders, 0 ₫.` All 14 columns carry `0 ₫`, are at most 6 px high and share the pale `background-color`. "Pairs sold by line" shows five rows each with `0` and an empty bar. "Latest orders" has its heading and no row.
Traces: SC-21, SC-22, SC-23, SC-24 (ADMIN-021 "Absent data"; ADMIN-020 AC-12)
Check: interact_page

## 13. What the page asks for and what it shows while waiting (UI+BE) — SC-25 … SC-28

**Scenario**: SC-25a — one overview read, one gate read, the menu's one-order count, and no list of orders, shoes or lines
**Given**: No interception. The browser is signed in as ADMIN with the network log running and the log cleared.
**When**: The run opens `/admin` (a full load), waits until "Overview" and the figures are shown, and reads the log of requests whose path starts `/api/`.
**Then**: Exactly one `GET /api/v1/admin/overview` and exactly one `GET /api/v1/auth/me`; every request to `/api/v1/orders` carries `limit=1` (the menu's own waiting-orders count) and none carries a larger limit, so no request for orders, shoes or lines has a limit above 1 (none has `limit=1000`); there is no request to `/api/v1/lines`, `/api/v1/products` or `/api/v1/reviews` at all. (Requests to any other path, such as `/api/v1/categories`, are not part of this criterion and are not judged.)
Traces: SC-25 (ADMIN-021 AC-13, item 3)
Check: interact_page

**Scenario**: SC-25b — nothing the page adds to the traffic carries a customer's phone, email, address or note
**Given**: As SC-25a, with the response bodies of the log readable. FX-BIG is not needed; the five latest orders of the shop carry the sentinels `0999000TCOV`, `tcov@example.invalid`, `TCOV ADDRESS`, `TCOV NOTE` in their `phone`, `email`, `address`, `note` (set by SQL on the five newest orders; put back afterwards).
**When**: The run opens `/admin` and reads every response body of the log.
**Then**: The overview response and the `auth/me` response hold none of the four sentinels and none of the keys `phone`, `email`, `address`, `note`, `city` (the `auth/me` body holds the signed-in dealer's own `email`, `name`, `role`, which is the dealer, not a customer); the page's text holds none of the sentinels. The menu's `GET /api/v1/orders?…&limit=1` returns at most one item and is the only response that may carry a customer's fields (§5 point 6); the report names it.
Traces: SC-25 (ADMIN-021 AC-13; ADMIN-020 AC-18; Privacy NFR)
Check: interact_page

**Scenario**: SC-26 — while the answer is awaited the page shows the title, the menu and "Loading the overview…", and no figure
**Given**: The overview answer is held for 3 seconds (interception; the answer is FX-ANS-A when released). The browser is signed in as ADMIN.
**When**: The run opens `/admin` and reads the page as soon as `main h1` appears, again at about 1.5 s and at about 2.5 s, and once more after the answer has been released.
**Then**: In all three reads during the hold: `main h1` reads "Overview"; the left menu (`nav`) is shown with its links; under the title an element with `role="status"` (or `aria-live="polite"`) reads "Loading the overview…"; the page shows none of the four figure labels, none of the panel headings ("Today", "Sales, last 14 days", "Pairs sold by line", "Latest orders", "Running low"), no chart (`figure`), no figure of `0` or `0 ₫`, no "Nothing is running low." and no "We could not load the overview.". After the release: the four figures and the five panels are drawn as in SC-18a and "Loading the overview…" is gone.
Traces: SC-26 (ADMIN-021 item 4, AC-14; Accessibility NFR; story decision 4)
Check: interact_page

**Scenario**: SC-27a — a real 500 shows one failure view with "Try again", and no figure
**Given**: No interception. ADMIN is signed in; `ALTER TABLE product_sizes RENAME TO tc_ov_product_sizes` makes the overview's own read fail while the gate and the session stay working.
**When**: The run opens `/admin` and waits until the page settles (at most 20 seconds), then reads it.
**Then**: `main h1` reads "Overview"; the left menu is shown; an element with `role="alert"` reads "We could not load the overview."; a button named "Try again" is shown; none of the figure labels, panel headings or chart is shown; "Loading the overview…" is gone; no `0` stands for a figure. The table is renamed back afterwards.
Traces: SC-27 (ADMIN-021 item 4, AC-15; story decision 3)
Check: interact_page

**Scenario**: SC-27b — a dropped connection shows the same failure view
**Given**: The overview request is aborted by the interception (every attempt fails at the network, including the client's one retry).
**When**: The run opens `/admin` and waits until the page settles (at most 20 seconds).
**Then**: The same view as SC-27a: "Overview", the menu, "We could not load the overview." in an element with `role="alert"`, a "Try again" button, no figure, no panel.
Traces: SC-27 (ADMIN-021 item 4, AC-15; story decision 3)
Check: interact_page

**Scenario**: SC-27c — a 401 or a 403 on the overview draws the same view, not a redirect and not a figure
**Given**: The overview answer is replaced, in two runs, by `401 {"error":"unauthenticated","message":"Sign in to continue."}` and by `403 {"error":"forbidden","message":"You do not have access to this."}`.
**When**: The run opens `/admin` in each and waits until the page settles.
**Then**: In both runs the address stays `/admin`; `main h1` reads "Overview"; the menu is shown; "We could not load the overview." is in an element with `role="alert"` with a "Try again" button; no figure and no panel is drawn; neither "Sign in" nor "This area is for the shop's team" is shown.
Traces: SC-27 (ADMIN-021 AC-15; story decision 3 "a 401 included")
Check: interact_page

**Scenario**: SC-27d — "Try again" shows the loading view, asks again, and a good answer replaces the message
**Given**: The interception runs in two phases, switched by the run and not counted by request number, because the client sends **two** requests for one failed attempt (`api.ts`: "A failed GET is tried once more" on a 5xx or no answer). Phase 1: every overview request is answered 500. Phase 2, set by the run just before it presses "Try again": the next overview request is held for 2 seconds and then answered 200 with FX-ANS-A.
**When**: The run opens `/admin` in phase 1, waits for the failure view and reads the number N1 of overview requests in the log (expected 2: the try and its one retry), switches to phase 2, presses "Try again" and reads the page at once, again at about 1 s, and after the release.
**Then**: Right after the press: "We could not load the overview." and the "Try again" button are gone and "Loading the overview…" (`role="status"`) is shown with no figure and no panel; the log holds exactly one more overview request than N1 (the held one; a 200 is not retried), so the log holds N1 + 1 in all; after the release the four figures and five panels of SC-18a are drawn and neither the failure message nor the loading line remains. A page that shows the failure view at the second request without a press, or that never asks again after the press, fails.
Traces: SC-27 (ADMIN-021 item 4, AC-15; story decision 4)
Check: interact_page

**Scenario**: SC-27e — when the retry fails again the message returns, and a later retry can still succeed
**Given**: The interception runs in three phases, switched by the run, because a failed attempt is **two** requests (the client retries a failed GET once, `api.ts`). Phase 1: every overview request is answered 500. Phase 1b, set before the first press: the first overview request is held for 1 second and then answered 500, and every one after it is answered 500 at once. Phase 2, set before the second press: every overview request is answered 200 with FX-ANS-A.
**When**: The run opens `/admin` in phase 1 and waits for the failure view (reading the number N1 of overview requests, expected 2), switches to phase 1b, presses "Try again", waits for the page to settle, switches to phase 2 and presses "Try again" again.
**Then**: After the first press the loading line shows (during the 1-second hold), then "We could not load the overview." is back with its "Try again" button and still no figure or panel, and the log holds N1 + 2 overview requests (the new attempt and its one retry); after the second press the figures and panels of SC-18a are drawn, the message and the loading line are gone, and the log holds N1 + 3 overview requests in all (the third attempt succeeds at its first request: 5 in all when N1 is 2).
Traces: SC-27 (ADMIN-021 AC-15; "when it fails again the message returns")
Check: interact_page

**Scenario**: SC-28a — at 390 px the page has no horizontal scroll, the figures are one column and the rows wrap
**Given**: FX-ANS-A is substituted; the browser is 390×844 and its time zone is `Asia/Ho_Chi_Minh`.
**When**: The run opens `/admin` and reads `document.documentElement.scrollWidth` and `clientWidth`, the bounding box of each figure link and of each panel, the 14 day labels, and the first row of "Latest orders".
**Then**: `scrollWidth` ≤ `clientWidth` (no horizontal page scroll); the four figure links have the same left edge and strictly increasing tops (one column), each as wide as the content column; every panel's right edge is inside the viewport; the 14 day labels each have a computed `font-size` of at least 10 px, lie inside the viewport and do not overlap one another; the first latest-orders row's right half (`Placed · 25/11/2026, 08:40 · 3.290.000 ₫`) starts below the bottom of its left half (two lines); the sidebar `nav` is not visible and the header's menu button is.
Traces: SC-28 (ADMIN-021 item 5, AC-16; Responsive NFR)
Check: interact_page

**Scenario**: SC-28b — at 320 px the same holds
**Given**: As SC-28a with the browser at 320×640.
**When**: The run reads the same measurements.
**Then**: Every assertion of SC-28a holds at 320 px: no horizontal page scroll, one column of figures, panels inside the viewport, 14 readable non-overlapping day labels, the first latest-orders row on two lines, the menu behind its button.
Traces: SC-28 (ADMIN-021 AC-16; Responsive NFR)
Check: interact_page

**Scenario**: SC-28c — two figures across at the `sm` breakpoint, four from `lg`
**Given**: FX-ANS-A is substituted.
**When**: The run opens `/admin` at 640×900 and reads the figure links' boxes, then at 1024×800, then at 1280×800.
**Then**: At 640 px figures 1 and 2 share a top and figures 3 and 4 share a lower top, with figure 3 under figure 1 (two across); at 1024 px and at 1280 px all four share one top (four across); at neither width is there a horizontal page scroll.
Traces: SC-28 (ADMIN-021 item 5; "two across at the `sm` breakpoint, four from `lg`")
Check: interact_page

**Scenario**: SC-28d — on a phone the left menu opens as a sheet and closes
**Given**: FX-ANS-A is substituted; the browser is 390×844.
**When**: The run presses the header's menu button, reads the sheet, then presses Escape (and, in a second pass, the sheet's close control) and reads the page and the focus.
**Then**: After the press a dialog (`[role=dialog]`) shows the links "Overview", "Orders", "Shoes", "Lines", "Categories", "Reviews" (and "Staff" for ADMIN); after Escape (and after the close control) the dialog is gone, the page still shows "Overview" and its figures, and focus is back on the menu button.
Traces: SC-28 (ADMIN-021 item 5, AC-16)
Check: interact_page

**Scenario**: SC-28e — the figures' links, the rows and "Try again" are thumb-sized
**Given**: FX-ANS-A is substituted; the browser is 390×844. In a second pass the overview answer is a 500 so the failure view is shown.
**When**: The run reads the height of each figure link, of each "Latest orders" row link and each "Running low" row link; then, in the failure view, the height of the "Try again" button.
**Then**: Each of the four figure links, the five latest-orders rows and the two running-low rows is at least 44 px high; the "Try again" button is at least 44 px high (the view is not drawn in the design, so nothing in the design limits it).
Traces: SC-28 ("reached with a thumb … at least 44 px high where the design allows")
Check: interact_page

**Scenario**: SC-28f — long names and large amounts still fit at 320 px
**Given**: FX-ANS-A is substituted with these changes: a running-low shoe named with 60 characters; the third latest order's `customer` of 40 characters and `product_name` of 60; `pairs_by_line` extended to 10 lines with names of 30 characters; `month.sales` 999999999999 and `sales` of the day `24` 99999999999.
**When**: The browser is 320×640 and the run opens `/admin` and reads `scrollWidth` and `clientWidth` of the page, and of each panel.
**Then**: No horizontal page scroll; no panel's content is wider than the panel (`scrollWidth` ≤ `clientWidth` for each); the long names wrap or are cut inside their rows and no text is drawn outside the viewport.
Traces: SC-28 (ADMIN-021 item 5, AC-16; Responsive NFR)
Check: interact_page

## 14. The gate (UI) — SC-29

**Scenario**: SC-29a — with no session `/admin` goes to the sign-in page and asks for no figure
**Given**: No session; the network log running and cleared.
**When**: The run opens `/admin`.
**Then**: The address becomes `/admin/sign-in`; the sign-in form is shown; no figure, panel or "Overview" title is drawn; the log holds no request to `/api/v1/admin/overview`.
Traces: SC-29 (ADMIN-021 AC-17; ADMIN-002 AC-6)
Check: interact_page

**Scenario**: SC-29b — a customer sees the team-only message and no figure, and the page never asks for the overview
**Given**: FX-CUSTOMER; the browser signs in as that customer; the network log running and cleared.
**When**: The run opens `/admin`.
**Then**: The page shows "This area is for the shop's team" and a link "Back to the shop" whose `href` is `/`; no figure, panel or chart is drawn; the log holds no request to `/api/v1/admin/overview`; pressing "Back to the shop" takes the address to `/`.
Traces: SC-29 (ADMIN-021 AC-17; ADMIN-002 AC-17; ADMIN-004 AC-7)
Check: interact_page

**Scenario**: SC-29c — a dealer who reloads `/admin` stays on it and the page asks again
**Given**: The browser is signed in as ADMIN on `/admin` with the figures drawn; the network log is cleared.
**When**: The run reloads the page (a full document load).
**Then**: The address is still `/admin`; "Overview" and the four figures are drawn again; the log holds one `GET /api/v1/admin/overview` from the reload; the sign-in page is not shown.
Traces: SC-29 (ADMIN-002 AC-17; story decision 5)
Check: interact_page

## 15. Source rules (the PR, not a response)

**Scenario**: S-1a — the backend adds up in SQL, takes its clock from the handler's dependency, and adds no table
**Given**: The backend PR's diff and `run_tests` output for `go build ./...`, `go vet ./...` and `go test ./...`.
**When**: `internal/admin/overview.go`, `internal/admin/overview_store.go`, their tests, `internal/server/server.go` and the migrations directory are read.
**Then**: Every figure comes from a SQL aggregate (`SUM`, `COUNT`, `generate_series`, `GROUP BY`); no Go loop reads orders or order items into memory to add them; the handler holds no SQL; "today" and "this month" come from a clock the handler is given (the real clock in `server.go`, a fixed instant in tests) and no environment variable moves it; the route is wrapped in `auth.Require("admin", "staff")`; `server.go` gains one `Register` line; there is no new table; a new index on a table that already has rows, if any, is created `CONCURRENTLY`; the database tests skip without `TEST_DATABASE_URL` and the rest pass. `manual` because no instrument reads the repository.
Traces: story §2 (backend scope: SQL aggregates, injected clock, no new table); SC-13, SC-16, SC-17
Check: manual

**Scenario**: S-1b — the page reads the one endpoint, narrows its row type, and adds only three words
**Given**: The frontend PR's diff and `run_tests` output for the frontend.
**When**: `src/pages/AdminOverview.tsx`, `src/lib/overview.ts`, `src/components/AdminLowStock.tsx`, `src/content.json`, `src/lib/mock/` and the page's test file are read.
**Then**: `AdminOverview.tsx` reads `useOverview` from `src/lib/overview.ts` (built on `src/lib/api.ts`) and no longer imports `useCollection`, `Order`, `Product` or `Line`; at the backend merge `src/lib/mock/dealer-dashboard-figures-and-charts.ts` is gone and nothing imports it; `AdminLowStock`'s row type holds only `id`, `name`, `image`, `stock`, `sizes_low`; `src/content.json` gains exactly `pages.admin.overview.loading`, `.error`, `.retry` and no existing key changes; every new word is drawn through `useContent` / `<T>`; each file is under 300 lines; no raw colour, arbitrary-value class or CSS module; `npm run lint`, `npm run build` and `npm test` pass. `manual` because no instrument reads the repository.
Traces: story §2 (frontend scope), §8 ("At the backend merge"); SC-25, SC-26, SC-27
Check: manual

## 16. Coverage — every story criterion and every SRS criterion

| Story criterion | Cases |
|---|---|
| SC-1 | SC-1a, SC-1b |
| SC-2 | SC-2a, SC-2b |
| SC-3 | SC-3a, SC-3b, C-1 |
| SC-4 | SC-4 |
| SC-5 | SC-5 |
| SC-6 | SC-6a, SC-6b |
| SC-7 | SC-7a, SC-7b, SC-7c |
| SC-8 | SC-8a, SC-8b, SC-8c |
| SC-9 | SC-9a, SC-9b |
| SC-10 | SC-10a, SC-10b |
| SC-11 | SC-11a, SC-11b |
| SC-12 | SC-12a, SC-12b |
| SC-13 | SC-13a, SC-13b, SC-13c |
| SC-14 | SC-14a, SC-14b, SC-14c |
| SC-15 | SC-15a, SC-15b |
| SC-16 | SC-16 |
| SC-17 | SC-17 |
| SC-18 | SC-18a, SC-18b, SC-18c |
| SC-19 | SC-19a, SC-19b |
| SC-20 | SC-20 |
| SC-21 | SC-21a, SC-21b, SC-21c |
| SC-22 | SC-22a, SC-22b |
| SC-23 | SC-23a, SC-23b, SC-23c |
| SC-24 | SC-24a, SC-24b, SC-24c |
| SC-25 | SC-25a, SC-25b |
| SC-26 | SC-26 |
| SC-27 | SC-27a, SC-27b, SC-27c, SC-27d, SC-27e |
| SC-28 | SC-28a, SC-28b, SC-28c, SC-28d, SC-28e, SC-28f |
| SC-29 | SC-29a, SC-29b, SC-29c |
| Story §2 source rules | S-1a, S-1b |

| SRS criterion | Cases |
|---|---|
| ADMIN-020 AC-1 / AC-2 / AC-3 / AC-4 / AC-5 / AC-6 | SC-4 / SC-5 / SC-6a / SC-7a / SC-8a / SC-6b |
| ADMIN-020 AC-7, "Boundary" | SC-11a, SC-11b |
| ADMIN-020 AC-8, AC-9 | SC-10a |
| ADMIN-020 AC-10, AC-11 | SC-9a, SC-9b |
| ADMIN-020 AC-12, "Absent data" | SC-12a, SC-12b |
| ADMIN-020 AC-13 | SC-7b |
| ADMIN-020 AC-14, AC-15, "Not permitted" | SC-1a, SC-1b, SC-2a, SC-2b |
| ADMIN-020 AC-16 / AC-17 | SC-14a, SC-14c / SC-14b |
| ADMIN-020 AC-18 | SC-16 (size and speed: SC-17) |
| ADMIN-020 AC-19 | SC-15a, SC-15b |
| ADMIN-020 "Upstream failure" | SC-13a, SC-13b, SC-13c |
| ADMIN-021 AC-1 to AC-4 | SC-18a, SC-18b, SC-18c |
| ADMIN-021 AC-5 / AC-6 | SC-20 / SC-19a, SC-19b |
| ADMIN-021 AC-7 / AC-8 | SC-21a, SC-21b, SC-21c / SC-22a, SC-22b |
| ADMIN-021 AC-9, AC-10 / AC-11 / AC-12 | SC-23a, SC-23b / SC-24a / SC-24b |
| ADMIN-021 AC-13 / AC-14 / AC-15 / AC-16 / AC-17 | SC-25a, SC-25b / SC-26 / SC-27a–e / SC-28a–f / SC-29a, SC-29b |
| ADMIN-021 "Absent data" | SC-23c, SC-24c |
| ADMIN-004 AC-5, AC-7, AC-10 (overview route only) | SC-1a, SC-2a, SC-2b |

## 17. Instruments, in one place

- `fetch_url` for every status, header and body of the endpoint (SC-1 … SC-17, C-1); `set_db_state` arranges every fixture and puts it back; `set_db_available` takes the database away for SC-13b.
- `interact_page` for every case of the page (SC-14c, SC-18 … SC-29): it signs in, substitutes, holds or fails the overview answer, sets the viewport and the browser's time zone, reads the network log, presses, reloads and reads computed boxes and colours.
- `manual` is used three times only — SC-13c (the server's log) and S-1a, S-1b (the repository) — each because no instrument reads that place.
- Cases that depend on an instrument feature the task may not offer are named in §5 point 9; they come back UNVERIFIED with the missing feature named, never failed.
