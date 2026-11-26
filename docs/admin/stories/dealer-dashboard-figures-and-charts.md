# Story — Dealer dashboard figures and charts

Module: `admin` · Plan item 11 · Requirements: ADMIN-020, ADMIN-021, and the overview rows of ADMIN-004 (`docs/admin/SRS.md` §4.3, §4.1)
Design system: `design/design-system.md` (§1.1 management area; the `Stat`, `Panel` and chart parts) · Architecture: `docs/architecture/overview.md` (§8 row 10), `erd.md` (`orders`, `order_items`, `products`, `product_sizes`, `lines`), `services.md` (§3.25)

## 1. User story

As the **dealer** (the admin, or a staff member the admin added), I want the Overview page at `/admin` to show this month's sales, what waits to be packed, the pairs on the shelf, the shoes running low, sales for each of the last 14 days, pairs sold by line, the latest orders and the shoes running low — all added up by the shop in one request — so that I see what needs me today however many orders there are, and no customer's order is sent to my browser to be added up.

Other actors this story touches:

- **Customer** — signed in, is refused: `GET /v1/admin/overview` answers 403 and `/admin` shows "This area is for the shop's team".
- **Visitor** — no session: the endpoint answers 401 and `/admin` goes to `/admin/sign-in`.

## 2. In scope

**Backend (Go, PostgreSQL) — dealer session only**

- `GET /v1/admin/overview` (`services.md` §3.25), wrapped in `auth.Require("admin", "staff")`: 401 with no session, 403 for a customer. No parameters. `Cache-Control: no-store`. The answer, in one request, every figure computed by SQL aggregates and never by reading orders into Go to add them up:
  - `month.sales` and `month.orders` — the sum of the totals and the count of orders placed in the current calendar month (from 00:00:00 on the 1st, Vietnam time), status not `cancelled`;
  - `waiting_orders` — orders with status `placed` or `confirmed`;
  - `pairs_on_shelf` and `shoes` — the sum of every shoe's pairs (`product_sizes.pairs`) and the number of shoes, drafts included;
  - `running_low` — shoes with a size holding one or two pairs **and** fewer than 15 pairs in all (drafts included), fewest pairs first, ties in the shoes' own order; each with `id`, `name`, `category` (the category id), `image`, `stock`, `sizes_low`;
  - `sales_by_day` — exactly 14 entries, today and the 13 days before it, oldest first, each `{date, sales}`, days cut at Vietnam midnight, `0` when nothing sold, cancelled orders excluded;
  - `pairs_by_line` — every line, `0` when none, the sum of `order_items.qty` of non-cancelled orders joined to the shoe's line, most pairs first, ties in the lines' own order; a shoe with no line is in no row;
  - `latest_orders` — the five newest orders of any status (a cancelled one is included and shown as cancelled), newest first: `id`, `number`, `customer`, `product`, `product_name`, `size` (of the first pair), `status`, `placed_at` (RFC 3339 with `+07:00`), `total`. **No phone, email, address or note, and no other order.**
- "Today" and "this month" come from a clock the handler is given (the real clock in the running server, a fixed instant in tests). No environment variable moves the clock of a deployed server.
- Database failure → 500 with the generic error envelope (`internal`), no SQL or stack in the body.
- No new table. TL's technical design decides whether an index helps the aggregates at 10,000 orders; one on a table that already has rows is made `CONCURRENTLY` (`overview.md` §4).

**Frontend (Vite + React + Tailwind tokens)**

- `src/pages/AdminOverview.tsx` is rewritten to read that one endpoint through `src/lib/overview.ts` (a `useOverview` hook on `src/lib/api.ts`: loading, ready, error, retry) instead of `useCollection` over orders, shoes and lines (`pageSize: 1000`). It keeps `OverviewPage`, `Stat`, `Panel`, `AdminBars` and the page's words; the figures, panels, rows and links are what the design draws.
- The four figures open their lists: "Sales this month" → `/admin/orders`, "To pack" → `/admin/orders?status=confirmed`, "Pairs in stock" → `/admin/products`, "Running low" → `/admin/products`.
- Five panels as designed: "Today" (the sentence), "Sales, last 14 days" (14 columns), "Pairs sold by line" (one bar per line), "Latest orders" (up to five rows), "Running low" (rows, or "Nothing is running low.").
- A **loading** view ("Loading the overview…" under the title) and a **load-failed** view ("We could not load the overview." with a "Try again" button) — the design draws neither (SRS §5, open question 5).
- `src/components/AdminLowStock.tsx`: its row type narrows from the whole `Product` to what the answer carries (`id`, `name`, `image`, `stock`, `sizes_low`). This story owns that file (no other page imports it).
- Three new keys in `src/content.json`, additive only: `pages.admin.overview.loading`, `.error`, `.retry` (SRS open question 5 asks for the copy to be editable by the owner). `content.json` is frozen for stories by `overview.md` §5; "Staff accounts managed by the owner" lifted that for new keys only, and this story does the same — existing keys are untouched. TL confirms in the UI review; if TL prefers, the UI PR lists the keys instead.
- A mock for the UI stage only, `src/lib/mock/dealer-dashboard-figures-and-charts.ts` (the SRS sample answer, an empty answer and a delayed/failing one); the backend stage deletes it and `useOverview` talks to the real endpoint.

## 3. Out of scope — and where each piece is proven

| Not here | Why | Proven in |
|---|---|---|
| Changing an order's status, cancelling, courier, marking paid | "Dealer handles orders: status, courier, payment" (ADMIN-016). This story only reads orders. A cancelled order that arrives by that story simply drops out of the figures | That item; SC-14 here proves the exclusion with a status set by the test |
| Editing a shoe, its sizes or its pairs | "Manage shoes, lines, categories, reviews and orders" (ADMIN-010 to ADMIN-015); this story reads the counts it leaves | That item |
| The Orders list honouring `?status=confirmed`, the Shoes list's own filters, the left menu's "waiting" badge | They are the lists' and the menu's business (`useAdminNav` asks for one order, `limit=1`). This story proves only the address each figure links to (SC-20) and that the page makes no list request of its own (SC-26). `admin-nav.ts` is not touched | The Orders / Shoes lists; SRS open question 6 |
| "Your session ended. Sign in again." and the redirect after a session ends mid-view | ADMIN-011 (item 10) and the `/admin/*` gate (item 9). Here any failed load, a 401 included, shows "We could not load the overview." | Those items |
| More charts, a date-range picker, a month or year view, clicking a bar to drill down, export, print | The SRS allows two charts (§3 out of scope: "charts beyond the two on the overview") | — |
| Refreshing by itself (polling, live updates) | Not in the SRS; the page reads once per visit and again on "Try again" | — |
| Counting only **paid** orders as sales, or showing paid and unpaid apart | The SRS defines sales as every non-cancelled order placed in the month; see ⚠️ in the reply | — (owner's decision) |
| Another zone, another currency | Vietnam time and VND only (SRS §6 Localisation) | — |
| A cache or a materialised table for the figures | Not needed unless SC-17 (p95) fails; TL's design decides | SC-17 |
| Staff accounts, customer accounts, sign-in | Their own items; this story only proves who may read the endpoint | Items 9, 14, 15 |

## 4. Decisions made here (the SRS leaves them open)

1. **The days of the chart are the shop's, not the browser's.** The page labels each column with the day of the month taken from the answer's `date` text (`"2026-11-12"` → `12`), without making a `Date` of it, so a dealer whose laptop is set to another zone sees the same 14 days the shop counted. (The old page cut the days in the browser's zone from the latest order's date; the new one has no browser arithmetic on sales.)
2. **"Today" is the shop's clock.** With the sample orders last placed on 25/11/2026 and today 26/11/2026, the 14 days run 13/11 to 26/11 and the last column is a short pale one. The figures of SRS ADMIN-020 AC-1 to AC-6 (47.660.000 ₫ over 13 orders …) hold at the SRS's fixed clock, 25/11/2026 12:00 Vietnam time, which the backend tests set; against the running shop on another day, the figures are checked as sums (ADMIN-020 AC-19), not as those literals (SC-15, §8).
3. **Every failure is one view.** A network failure, a 5xx, a 401 or a 403 all draw "We could not load the overview." with "Try again"; the page never shows a figure of 0 for an answer it did not get (ADMIN-021 AC-14).
4. **Loading and failure replace the figures and panels, not the page.** The title "Overview", the left menu and its foot stay; the status text sits under the title (ADMIN-021 item 4). While "Try again" is waiting for its answer the loading view shows again.
5. **A fresh read on every visit.** Leaving `/admin` and coming back asks again; nothing from an earlier answer is drawn first. (`api.ts` still shares an in-flight identical GET.)
6. **The low-stock list is not capped.** `services.md` §3.25 lists every shoe running low, fewest first; the catalogue is a few dozen shoes. The "Running low" figure is the length of that list.
7. **One answer feeds the whole page.** The four figures, the "Today" sentence and the panels all come from the same response, so the sentence's counts always equal the figures above it.
8. **Wire names are the API's** (`waiting_orders`, `pairs_on_shelf`, `sales_by_day`, `pairs_by_line`, `latest_orders`, `running_low`, `sizes_low`); the SRS's prose names ("To pack", "Pairs in stock") are labels only.

## 5. UI scope

The page exists and is approved (`/admin`, "Overview" in `design/pages.json`): this story changes where its numbers come from and adds the two states the design never drew. Management palette (`[data-dashboard]`: slate neutrals, blue accent, 6px radius, Inter), tokens only — no raw colour, no arbitrary-value class, no CSS modules; every word through `useContent` / `<T>` because `eslint` refuses bare words in pages; each file under 300 lines.

| Screen / state | What it shows | SRS |
|---|---|---|
| `/admin`, default (1280 px) | Title "Overview"; four `Stat` figures across (label, value, note, icon, link); under them the five panels, one above the next: "Today", "Sales, last 14 days", "Pairs sold by line", "Latest orders", "Running low" | ADMIN-021 items 1, 2 |
| `/admin`, default (390 px and 320 px) | Figures in one column (two across from the `sm` breakpoint, four from `lg`); the panels at full width; the 14 columns fit with their day labels; a latest-orders row wraps its two halves; the left menu is behind its button | ADMIN-021 item 5, AC-16 |
| Figures | "Sales this month" `<amount>` + "<n> orders"; "To pack" `<count>` + "placed or confirmed"; "Pairs in stock" `<count>` + "across <n> shoes"; "Running low" `<count>` + "last pairs left" | ADMIN-021 item 1 |
| Panel "Sales, last 14 days" | 14 columns labelled with the day of the month, each with its amount as text (the existing `AdminBars` carries it as the column's text), a day with no sales a short pale column | ADMIN-021 item 2 |
| Panel "Pairs sold by line" | One bar per line, its name at the left and its count at the right, the answer's order (most pairs first); a line with 0 shows 0 and an empty bar | ADMIN-021 item 2 |
| Panel "Latest orders" | Up to five rows `<number> · <customer> · <shoe> · EU <size>` and at the right `<status> · <dd/mm/yyyy, hh:mm> · <total>`, each a link to `/admin/orders/<id>`; a cancelled order reads "Cancelled"; no rows when there are no orders | ADMIN-021 item 2, ADMIN-020 item 2 |
| Panel "Running low" | Rows with the shoe's picture, name, "Last pairs in <sizes>" and the pairs left, each a link to `/admin/products/<id>`; "Nothing is running low." when none | ADMIN-021 item 2 |
| Loading *(not drawn)* | Under the title "Loading the overview…" (`role="status"`); no figure, no panel, no 0, no "Nothing is running low." | ADMIN-021 item 4, AC-14 |
| Load failed *(not drawn)* | Under the title "We could not load the overview." (`role="alert"`) and a "Try again" button; no figure, no panel; pressing it shows the loading view and asks again | ADMIN-021 item 4, AC-15 |
| Customer / visitor on the address | The gate's "This area is for the shop's team" with "Back to the shop"; or `/admin/sign-in` — nothing of the page, and no overview request, until the gate lets a dealer in | ADMIN-021 AC-17, ADMIN-002 |

The loading and load-failed views are built the way the staff page's are (`StaffAccounts.tsx`: a centred `text-sm` line in `text-ink-soft`, an alert in `text-ink` with the outline `Button`), so the two pages answer a slow or failed read the same way.

## 6. Acceptance criteria

Twenty-nine criteria; ADMIN-020 has 19 of its own and ADMIN-021 has 17, and each is traced (the SRS pair is wider than one slice because several of its criteria are the same observation at the API and on the page — those are one `SC` here, marked **UI+BE**). Each is checked against the running story: the endpoint with real sessions (an admin, a staff account, a customer row inserted for the test, and no session), the page in a browser at 1280 px and 390 px. "The sample set" means the 14 seeded orders DOI-24804 to DOI-24817 with the totals "Free delivery on every order" left them, and "the SRS clock" means 25/11/2026 12:00 Vietnam time, set by a backend test; on the running shop with the real clock, the figures are compared with sums taken from the same database (§8).

### 6.1 Who may read it (BE)

- **SC-1 [ADMIN-004 AC-5; ADMIN-020 AC-14, "Not permitted"]:** With no session, `GET /v1/admin/overview` answers 401 with the generic error body and no figure; a forged cookie value answers 401 too.
- **SC-2 [ADMIN-004 AC-7, AC-10; ADMIN-020 AC-15; ADMIN-002 AC-20]:** A customer session gets 403 with the generic error body and no figure; an admin session and a staff session each get 200 with the same shape.
- **SC-3 [ADMIN-020 item 1; `services.md` §3.25]:** A dealer's answer is 200 JSON with exactly the fields `month` (`sales`, `orders`), `waiting_orders`, `pairs_on_shelf`, `shoes`, `running_low`, `sales_by_day`, `pairs_by_line`, `latest_orders`; money and counts are whole numbers; the response carries `Cache-Control: no-store`; an unknown query parameter changes nothing.

### 6.2 The figures (BE)

- **SC-4 [ADMIN-020 AC-1, item 2]:** On the sample set at the SRS clock, `month.sales` is 47660000 and `month.orders` 13: DOI-24811 (cancelled, 4.090.000 ₫) is in neither.
- **SC-5 [ADMIN-020 AC-2, item 1]:** The same read gives `waiting_orders` 2 (DOI-24816 placed, DOI-24815 confirmed); a packed, shipped, delivered or cancelled order is not counted.
- **SC-6 [ADMIN-020 AC-3, AC-6, item 1]:** `sales_by_day` has exactly 14 entries, `2026-11-12` to `2026-11-25`, oldest first. `2026-11-25` is 3290000, `2026-11-24` 5890000, `2026-11-23` 5380000, `2026-11-19` 3290000, `2026-11-14` 3090000; `2026-11-20` (the cancelled order's day) and `2026-11-16` are 0; the 14 amounts add up to 47660000.
- **SC-7 [ADMIN-020 AC-4, AC-13, item 4]:** `pairs_by_line` is Samba 8, Gazelle 2, Megaride 2, Handball Spezial 1, Adizero 1, in that order (the equal counts in the lines' own order). A line nothing sold is present with 0. An order of two pairs, one Samba and one Gazelle, adds each pair's own quantity to its own shoe's line. A shoe with no line is in no row. Cancelled orders add nothing.
- **SC-8 [ADMIN-020 AC-5, item 1, 2]:** `latest_orders` holds the five newest orders: DOI-24816, DOI-24815, DOI-24814, DOI-24817, DOI-24813, in that order. Each carries only `id`, `number`, `customer`, `product`, `product_name`, `size`, `status`, `placed_at`, `total`, with `product` and `size` of the order's first pair; `placed_at` is RFC 3339 with `+07:00`. A cancelled order that is among the five newest is listed, with status `cancelled`.
- **SC-9 [ADMIN-020 AC-10, AC-11, item 1]:** Shoes whose pairs in stock are 18, 24 and 31 give `pairs_on_shelf` 73 and `shoes` 3. A shoe in draft with 10 pairs adds its 10 pairs and counts as a shoe.
- **SC-10 [ADMIN-020 AC-8, AC-9, item 1]:** Of shoes with 5, 14, 15 and 18 pairs — the first three with a size holding one or two pairs, the fourth without — `running_low` lists the 5-pair and the 14-pair shoes, the 5-pair one first; the 15-pair shoe (not fewer than 15) and the 18-pair shoe are absent. A shoe with 9 pairs and no size of one or two pairs is absent. A shoe with a size of exactly 2 pairs is in, one with a size of exactly 3 is out, and a draft shoe that qualifies is in. Each row carries `id`, `name`, `category` (the category id), `image`, `stock`, `sizes_low`; equal `stock` keeps the shoes' own order.

### 6.3 Time, boundaries and absence (BE)

- **SC-11 [ADMIN-020 AC-7, item 3; boundary]:** With the clock at 01/12/2026 12:00 Vietnam time, an order placed at 23:30 on 30/11/2026 and one at 00:30 on 01/12/2026 (stored with their zone) fall in November's and December's months: `month` holds only the second, and the `2026-12-01` entry of `sales_by_day` holds the second while `2026-11-30` holds the first. An order at exactly 00:00:00 Vietnam time on the first of a month belongs to the new month; one at 23:59:59 the day before belongs to the old one, whatever zone the stored value was written in.
- **SC-12 [ADMIN-020 AC-12; absent data]:** With no orders, no shoes and no lines, the answer is 200 with `month` 0 and 0, `waiting_orders` 0, `pairs_on_shelf` 0, `shoes` 0, `running_low` `[]`, 14 entries of 0 in `sales_by_day`, `pairs_by_line` `[]` and `latest_orders` `[]` (arrays, never `null`). With shoes and lines but no orders, every line is present with 0.
- **SC-13 [ADMIN-020 "Upstream failure"]:** When the database cannot be reached the answer is 500 with the `internal` envelope and the generic sentence; the body holds no SQL, no table name and no connection detail, and the server logs the cause.

### 6.4 Sums, change and size (BE / UI+BE)

- **SC-14 [ADMIN-020 AC-16, AC-17; ADMIN-021 item 3]** (UI+BE)**:** After a new order is placed (through checkout) the next read of the overview counts it in `month`, in `waiting_orders`, in its day and in its lines, and `/admin`, opened again, shows the new figures without a manual reload of anything else. After an order is cancelled — its status set by the test with SQL if no cancel route is on `main` yet — the next read has it in none of the figures, and `latest_orders` still lists it as cancelled while it is among the five newest.
- **SC-15 [ADMIN-020 AC-19, item 1]:** For the sample set and for a generated set (a few hundred orders over three months, every status, one to three pairs each, several shoes and lines), `month.sales`, `month.orders`, each of the 14 days, each line's pairs and `waiting_orders` equal the sums computed independently over the stored non-cancelled orders and pairs (plain SQL written for the test, not the handler's query). On the running shop with the real clock the same sums are taken from the dealer's own order list and the database, and match.
- **SC-16 [ADMIN-020 AC-18; Privacy NFR]:** With 10,000 orders in the database the response holds no order other than the five latest, and nowhere in the response — any status, any field — is there a phone, an email, an address or a note. The body is the same few kilobytes as with 14 orders.
- **SC-17 [Performance NFR; `services.md` §3.25]:** With 10,000 orders and 1,000 shoes, 100 sequential requests of a dealer session answer with a p95 of at most 500 ms.

### 6.5 The page's figures and links (UI+BE)

- **SC-18 [ADMIN-021 AC-1 to AC-4, item 1; ADMIN-002 AC-16, AC-20]:** Signed in as the admin, and again as a staff member, `/admin` shows the title "Overview" and four figures read from the overview answer: "Sales this month" with the amount in the shop's money format (`47.660.000 ₫`) and the note "13 orders"; "To pack" 2 with "placed or confirmed"; "Pairs in stock" 73 with "across 3 shoes"; "Running low" with its count (two shoes: 2) and "last pairs left". (The literals are for an answer carrying those numbers; on the running shop each value equals the same field of `GET /v1/admin/overview`.)
- **SC-19 [ADMIN-021 AC-6, item 2]:** "Today" reads the sentence with the answer's numbers: with 2 waiting, 5 running low, 13 orders and `47.660.000 ₫` it reads "2 orders are waiting to be packed and 5 shoes are down to their last pairs. This month so far: 13 orders, 47.660.000 ₫."; its counts always equal the figures above it.
- **SC-20 [ADMIN-021 AC-5, item 1]:** Pressing each figure opens, in order, `/admin/orders`, `/admin/orders?status=confirmed`, `/admin/products` and `/admin/products` (the address is right; what each list then shows is the list's own story).

### 6.6 The charts and rows (UI+BE)

- **SC-21 [ADMIN-021 AC-7, item 2; decision 1]:** "Sales, last 14 days" shows 14 columns, one per entry of `sales_by_day`, in the answer's order, each labelled with the day of the month of its `date` text (for the SRS clock: 12, 13, … 25) and carrying its amount as text; a day of 0 is a short pale column and a day with sales a taller blue one in proportion to the largest. With the browser set to another time zone (a viewer in New York or Sydney) the labels are the same, since the page does not turn `date` into a local day.
- **SC-22 [ADMIN-021 AC-8, item 2]:** "Pairs sold by line" shows one row per entry of `pairs_by_line` in the answer's order — Samba 8, Gazelle 2, Megaride 2, Handball Spezial 1, Adizero 1 — each with a bar and its count at the right; a line of 0 shows "0" and an empty bar; with no lines the panel is present and empty.
- **SC-23 [ADMIN-021 AC-9, AC-10, item 2; ADMIN-020 item 2]:** "Latest orders" shows up to five rows in the answer's order; the first reads `DOI-24816 · Trang · Handball Spezial Earth Strata · EU 37 1/3` with `Placed · 25/11/2026, 08:40 · 3.290.000 ₫` at the right, and pressing it makes the address `/admin/orders/doi-24816`. A cancelled order among the five reads "Cancelled" as its status. With no orders the panel has its title and no rows.
- **SC-24 [ADMIN-021 AC-11, AC-12, item 2]:** "Running low" shows, for each shoe of `running_low`, its picture, its name, "Last pairs in <sizes>" (the sizes joined with ", ") and its pairs left, as a link to `/admin/products/<id>`, in the answer's order; with an empty list it shows "Nothing is running low." and the figure above reads 0.

### 6.7 What the page asks for and what it shows while waiting (UI+BE)

- **SC-25 [ADMIN-021 AC-13, item 3]:** Opening `/admin` in a browser with the network watched sends one `GET /api/v1/admin/overview` for the figures and charts, one `GET /api/v1/auth/me` for the gate, and the left menu's own waiting-orders count (`GET /api/v1/orders` with `limit=1`); it sends no request for orders, shoes or lines with a limit above 1, and no request to `/v1/lines`, `/v1/products` or `/v1/reviews` at all. Nothing in the page's traffic carries a customer's phone, email, address or note.
- **SC-26 [ADMIN-021 item 4, AC-14; Accessibility NFR]:** With the overview answer held for 3 seconds the page shows its title and left menu and, under the title, "Loading the overview…" (announced politely); during the wait it draws no figure, no figure of 0, no panel, no empty chart and not "Nothing is running low."; when the answer arrives the figures and panels replace it.
- **SC-27 [ADMIN-021 item 4, AC-15; Accessibility NFR]:** When the overview answers 500, or the network fails, or the answer is 401 or 403, the page shows the title, the menu, "We could not load the overview." (announced) and a "Try again" button, and no figure or panel. Pressing "Try again" shows "Loading the overview…" and asks again; when the shop then answers 200 the figures and panels are drawn and the message is gone; when it fails again the message returns.
- **SC-28 [ADMIN-021 item 5, AC-16; Responsive NFR]:** At 390 px and at 320 px `/admin` has no horizontal page scroll; the four figures stack in one column (two across at the `sm` breakpoint, four from `lg`); all five panels fit the width with the 14 day labels readable and a latest-orders row wrapping onto two lines; the left menu is behind its button, and its sheet opens and closes. The figures' links and "Try again" can be reached with a thumb (a row or button at least 44 px high where the design allows).

### 6.8 Gate (UI)

- **SC-29 [ADMIN-021 AC-17; ADMIN-002 AC-6, AC-17]:** With no session, `/admin` goes to `/admin/sign-in` and the browser sends no request to `/v1/admin/overview`; with a customer session `/admin` shows "This area is for the shop's team" with a link to `/`, draws no figure or panel, and sends no request to `/v1/admin/overview` either. A dealer reloading `/admin` stays on it.

### 6.9 Where each SRS criterion is proved

| SRS criterion | Proved by |
|---|---|
| ADMIN-020 AC-1 / AC-2 / AC-3 / AC-4 / AC-5 / AC-6 | SC-4 / SC-5 / SC-6 / SC-7 / SC-8 / SC-6 |
| ADMIN-020 AC-7 | SC-11 |
| ADMIN-020 AC-8, AC-9 | SC-10 |
| ADMIN-020 AC-10, AC-11 | SC-9 |
| ADMIN-020 AC-12 | SC-12 |
| ADMIN-020 AC-13 | SC-7 |
| ADMIN-020 AC-14, AC-15 | SC-1, SC-2 |
| ADMIN-020 AC-16, AC-17 | SC-14 |
| ADMIN-020 AC-18 | SC-16 (and SC-17 for the 500 ms bound of the NFR) |
| ADMIN-020 AC-19 | SC-15 |
| ADMIN-020 "Upstream failure" | SC-13 |
| ADMIN-021 AC-1 to AC-4 / AC-5 / AC-6 | SC-18 / SC-20 / SC-19 |
| ADMIN-021 AC-7 / AC-8 / AC-9, AC-10 / AC-11, AC-12 | SC-21 / SC-22 / SC-23 / SC-24 |
| ADMIN-021 AC-13 / AC-14 / AC-15 / AC-16 / AC-17 | SC-25 / SC-26 / SC-27 / SC-28 / SC-29 |
| ADMIN-004 AC-5, AC-7, AC-10 (overview route only) | SC-1, SC-2 |

## 7. Dependencies

- **Landed on `main`, nothing to wait for:** "Site data from the database" (tables `orders`, `order_items`, `products`, `lines`, `categories`; the 14 sample orders; `product_sizes` with its derived last-pairs lists), "Free delivery on every order" (the sample orders' totals as the SRS quotes them), "Checkout and placing an order" (orders with several pairs; the per-size counts that orders take), "Dealer sign-in and protected admin area" (`auth.Require`, sessions, the `/admin/*` gate, `src/lib/api.ts` with `get`) and "Staff accounts managed by the owner" (a staff session to test with).
- **Not a dependency:** "Manage shoes, lines, categories, reviews and orders" and "Dealer handles orders: status, courier, payment" run in parallel and touch other files; the overview reads what they write. Until the second lands there is no route that cancels an order, so SC-14's cancel half is set up by SQL in the backend test and is UNVERIFIED on the running shop until it merges.
- **No external account, key or setting** beyond the `ADMIN_EMAIL` / `ADMIN_PASSWORD` the sign-in story already asks for.
- **Data that must exist for verification:** the 14 sample orders, at least one multi-pair order (placed through checkout for SC-7 and SC-15), a shoe with a size of one or two pairs under 15 pairs in all (the sample set has them) and one with none, and a customer row in `users` for SC-2 and SC-29.

## 8. Notes for the next steps

- **Test data and the clock.** The SRS's literals (47.660.000 ₫, 13 orders, the 14 daily amounts, the line counts, the five latest) are proved by a backend test that fixes the clock at 25/11/2026 12:00 Vietnam time over the sample set as the seed leaves it; if the seed on `main` has moved, the sums replace the literals (ADMIN-020 AC-19). On the running shop the clock is real, so the 14 days end today (26/11/2026 at the time of writing, an empty last column) and, after 30/11/2026, "this month" holds none of the sample orders; the running-shop cases therefore compare the page with the endpoint and the endpoint with sums, never with a literal. For the page's own checks (SC-18 to SC-24) the verifier may intercept the overview answer with the SRS's sample numbers.
- **One instrument the page cases need:** holding the overview answer for 3 seconds and making it fail (SC-26, SC-27). The staff and shop stories use the same request interception; no test-only route is added to the backend.
- **Files, as a guide for TL's UI review** (each under 300 lines): `src/pages/AdminOverview.tsx` (rewritten; if the row lists grow, split `RecentList` into `src/components/AdminRecentOrders.tsx`), `src/lib/overview.ts` (types of the answer, `useOverview`), `src/components/AdminLowStock.tsx` (row type narrowed), `src/lib/mock/dealer-dashboard-figures-and-charts.ts` (UI stage only, deleted by the backend stage), three additive keys in `src/content.json`, one test file for the page. Backend: `internal/admin/overview.go` (handler, no SQL), `internal/admin/overview_store.go` (the aggregates), their tests, one `Register` line in `internal/server/server.go`. Nothing else in `src/App.tsx`: `/admin` is already routed inside `SignedIn`.
- **At the backend merge:** `AdminOverview.tsx` no longer imports `useCollection`, `Order`, `Product` or `Line`, and the mock module is gone (CI warns while a mock is imported).
