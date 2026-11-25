# ERD — Free delivery on every order, wording to match (plan item 2)

Module `site` · Requirement SITE-016 (with SITE-004, 006, 007, 009, 012 and ORD-001, 006, 007, 020 where they touch checkout, the order page and the track page) · Story `docs/site/stories/free-delivery-on-every-order-wording-to-match.md` · UI PR #24 · Engine PostgreSQL 16.

## 1. Verdict: this story adds nothing to the schema

**No table, column, constraint, index, trigger, function or migration is created, changed or dropped by this story.** The story is frontend and JSON only: it edits the files in `code/frontend/src/data/` that "Site data from the database" (S3) later copies into the first migrations. The database does not exist yet when this story lands.

The project ERD already holds the owner's two decisions, written when the architecture was designed, so there is nothing to extend:

| Owner decision | Where the schema already says it |
|---|---|
| Free delivery on every order: no fee, no threshold, no express | `erd/_base.md` §1 "Deliberately outside the database": *Delivery fee thresholds, express options — Nowhere*. `delivery_options` holds `standard`, fee 0. `shop_terms` has **no column** for `free_delivery_from`, `express_days` or `express_cities` (S3 ERD §4.5). `orders.delivery_option_id` is always `standard`; `orders.delivery_fee` is 0 (S3 ERD §4.10) |
| Pay online only, through VNPay; no cash on delivery | `payment_methods` keeps the four rows because sample orders refer to all four (FK `RESTRICT`); *which are offered* is code and settings, not data (S3 ERD §4.7) — `GET /v1/payment-options` never lists `cod` |

This file therefore does three things only: it pins **what S3's seed must find in the files** once this story has landed (§2), it **checks every sample-order total** against the seed's assertion (§3), and it records the decisions that keep the owner's rules from drifting between the pages and the database (§5).

## 2. The seed input after this story (the contract S3 reads)

S3's migration 3 is literal SQL generated from these files **as this story leaves them**. Each row below is a fact S3's seed and its assertions rely on (S3 ERD §5 and §6).

| File (`code/frontend/src/data/`) | Table | Content after this story | What S3 must do with it |
|---|---|---|---|
| `delivery.json` | `delivery_options` | **1 record**: `standard`, name "Standard", days "2–4 working days", fee `0`, note "Nationwide with GHN or GHTK, tracked from our door to yours." No `express` | Seed as written. Row count of the table = 1 |
| `shop.json` | `shop_terms` | `id` `shop`, `standard_days` "2–4", `dispatch_cutoff` "15:00", `reply_time`, `rating`, `review_count`. **No** `free_delivery_from`, `express_days`, `express_cities`, `exchange_days`, `refund_days` | Seed the five columns. There is no column for the removed fields (S3 ERD §4.5), so a stale file could never place them |
| `schema.json` | — (not seeded) | `shop` collection lists no removed field; `orders.delivery` options `[["standard","Standard"]]`; `delivery.fee` and the `cod` option of `orders.payment` unchanged | The registry's drift-guard test (services S3 §4) reads this file: it passes only once the three fields are gone from it |
| `orders.json` | `orders` + `order_items` | All 14 sample orders have `delivery` = `standard`. DOI-24815 total 5890000, DOI-24810 total 3290000, DOI-24806 total 3090000. Every total = the shoe's price × qty | `order_items.unit_price` = `total / qty`, `delivery_fee` = 0. The seed's assertion 2 (`total = unit_price × qty + delivery_fee`, `unit_price` = the shoe's price) passes |
| `payments.json` | `payment_methods` | Four rows in the order `cod`, `bank`, `vnpay`, `momo`. Only `vnpay`'s note changed: "Pay by card or banking app on VNPay's secure page." | Seed all four as written (sample orders refer to all four) |
| `faqs.json` | `faqs` | `f-delivery-time`, `f-payment-how` (replaces `f-payment-cod` in the same position), `f-orders-number` reworded; 12 rows as before | Seed as written. Topic `payment` still passes `ck_faqs_topic` |
| `reviews.json` | `reviews` | `r-04` and `r-07` lose their last sentence; 36 rows as before | Seed as written |

Row counts S3 reports are unchanged by this story except `delivery_options` (the design's two rows become the one S3's ERD already counts). No count is hard-coded in the migration: the generator writes the number it emitted.

## 3. The 14 sample totals, checked against the seed's assertion

S3's assertion 2 requires, for each sample order, `total = unit_price × qty + delivery_fee` with `unit_price` equal to its shoe's price and `delivery_fee = 0`. I checked every order in `orders.json` against `products.json` as the PR #24 branch leaves them:

| Order | Shoe | Shoe price | Pairs | Total in file | Holds |
|---|---|---|---|---|---|
| DOI-24817 | samba-og-w-crystal-white | 3090000 | 1 | 3090000 | yes |
| DOI-24816 | handball-spezial-w-earth-strata | 3290000 | 1 | 3290000 | yes |
| DOI-24815 | megaride-bones-halo-green | 5890000 | 1 | 5890000 | yes (was 5940000) |
| DOI-24814 | samba-og-kids-cloud-white | 2290000 | 1 | 2290000 | yes |
| DOI-24813 | gazelle-indoor-solar-gold | 3590000 | 1 | 3590000 | yes |
| DOI-24812 | samba-og-w-wonder-quartz | 2790000 | 2 | 5580000 | yes |
| DOI-24811 | adizero-adios-wonder-blue | 4090000 | 1 | 4090000 | yes |
| DOI-24810 | samba-og-w-core-black | 3290000 | 1 | 3290000 | yes (was 3340000) |
| DOI-24809 | megaride-s2-cloud-white | 5090000 | 1 | 5090000 | yes |
| DOI-24808 | samba-og-kids-wonder-mauve | 2290000 | 1 | 2290000 | yes |
| DOI-24807 | adizero-evo-sl-charcoal | 3790000 | 1 | 3790000 | yes |
| DOI-24806 | samba-og-w-crystal-white | 3090000 | 1 | 3090000 | yes (was 3140000) |
| DOI-24805 | gazelle-indoor-solar-gold | 3590000 | 1 | 3590000 | yes |
| DOI-24804 | samba-og-w-wonder-quartz | 2790000 | 1 | 2790000 | yes |

All 14 hold, so S3's seed can run on these files without editing a total. (On `main` today three of them still hold the old totals; the assertion would stop S3 there — see §6.)

## 4. Entities this story touches — none changed

| Entity (defined elsewhere) | Effect of this story |
|---|---|
| `delivery_options` | None. One seeded row, fee 0. Defined in S3 ERD §4.6 |
| `shop_terms` | None. Five seeded columns. Defined in S3 ERD §4.5 |
| `orders`, `order_items` | None. `delivery_option_id` = `standard`, `delivery_fee` = 0 on every row, `total` = the pairs' total. S3 ERD §4.10–4.11 |
| `payment_methods` | None. Four rows; the `vnpay` note's text changes in the seed file only. S3 ERD §4.7 |
| `faqs`, `reviews` | None. Text values only |

## 5. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| F1 | Does the owner's "free delivery" need a column, a flag or a setting? | **No.** "Free" is a rule of the shop, not a value anyone edits. The server writes `orders.delivery_fee = 0` as a **constant** when it places an order (services §3.10 already says so) and **never reads `delivery_options.fee`** to price one | A `shop_terms.free_delivery boolean`, or reading `delivery_options.fee` at checkout: either would let one stray edit (a hand-run `UPDATE`, a future dealer form) make the server charge what no page says, which breaks the story's first sentence — the total on the page is the total paid. If the owner later wants a fee, it is a new story with a migration and every page's words, which is the honest size of that decision |
| F2 | Drop `delivery_options.fee` and `orders.delivery_fee` now that both are always 0? | **Keep both.** `schema.json` still declares `delivery.fee` (SC-3 forbids removing it); `orders.delivery_fee` records "the fee as sold" so an order's history is exact whatever the shop decides next year | Dropping them: two always-zero columns saved, at the price of a breaking edit to the frozen schema and a migration on a table that will hold real orders. A column nobody reads costs nothing; a dropped column cannot be taken back |
| F3 | Keep the `cod`, `bank` and `momo` rows in `payment_methods`? | **Keep.** Five sample orders (DOI-24816, -24814, -24811, -24808, -24804) were "paid" by `cod`, and `orders.payment_method_id` is `RESTRICT`: history must keep saying how an order was paid | Deleting `cod` and re-pointing the five orders to `vnpay`: it rewrites sample history for a wording reason, and the orders would then read "paid by VNPay" for an order that never was |
| F4 | The sample orders paid by `cod` | Seeded exactly as the file has them (payment `cod`, `paid` as stored). They are placeholders; their page names the stored record. They are not an offer | Setting them to `vnpay` — see F3 |
| F5 | Where does "offered" live, now that checkout lists one way to pay? | In code and settings: `GET /v1/payment-options` (services §3.5) lists `vnpay` always and **never** `cod`. Until the Checkout story swaps the source, the UI reads the one `vnpay` record from the `payments` collection (story §8 decision 3) | A boolean `payment_methods.offered`: a second copy of a rule that already depends on the owner's keys (`vnpay` is offered only with `VNPAY_TMN_CODE` and `VNPAY_HASH_SECRET`), and a column no dealer form edits |

### Conflicts with merged decisions

None. I checked this story against `erd/_base.md` (D1–D10, §1, §3.x), the S3 ERD (§4.5, §4.6, §4.7, §4.10, §5, §6, open question 1) and `overview.md` §6 row 2: all say the same thing as the story. The one tension that existed — S3's seed assertion would stop on the three old express totals while PR #24 is open — is the ordering rule of §6, not a design conflict.

## 6. Migration plan

| Step | Forward | Backward | Safe on a populated table |
|---|---|---|---|
| This story | **No migration file.** The files in `src/data/` are edited in PR #24; nothing runs against any database | Revert PR #24 | Not applicable: no table exists yet, and nothing writes to one |

**Ordering rule (the real risk).** S3 copies the files **as they stand when its seed is generated**. PR #24 must therefore be merged before S3's backend PR is, or S3's own seed assertion fails on `main`'s old files (three orders still total 5940000 / 3340000 / 3140000; `delivery.json` still has an `express` row at fee 50000). That failure is by design — S3 ERD §5 says Dev stops and tells PM rather than editing a total. It is also the safety net: **no database can have been seeded from the old files**, because the seed would not have completed. So there is no populated table to repair, and no corrective migration is needed. A developer's local database that somehow holds old values is reset (migrations are never edited once merged — convention `migrations`).

**Backward path of the seed (S3's, unchanged).** S3's `…seed_collections.down.sql` deletes the seeded ids; it is a local-reset tool, not run by the server. This story adds nothing to it.

## 7. Hand-over

| Story | What it must know from this design |
|---|---|
| S3 "Site data from the database" | Seed from the files as §2 describes; do not merge before PR #24; the registry's `shop` entry lists five fields; the `delivery` collection answers one record and `GET /v1/delivery/express` is `404` |
| S5 "Checkout and placing an order" | `delivery_fee` is the constant 0 (F1); `POST /v1/orders` requires `delivery: "standard"` and rejects `cod`; `eta` comes from the **longest number of `shop_terms.standard_days`** (the mock's hard-coded `STANDARD_DAYS = 4` is deleted with the mock) |
| PM | `design/design-system.md` §2.1.17 and §2.1.21 still say cash on delivery and "size exchange"; this design changes no table, so it is a wording sync only |
