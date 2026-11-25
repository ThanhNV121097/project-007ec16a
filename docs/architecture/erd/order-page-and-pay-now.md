# ERD — Order page and Pay now (plan item 6, "S6")

Module `orders` · Requirements ORD-007 (all 26 criteria), ORD-006 items 4–6 and AC-12, ORD-009 AC-5 · Story `docs/orders/stories/order-page-and-pay-now.md` · UI PR #41 · Engine PostgreSQL 16, pgx v5, hand-written SQL.

## 1. Verdict: this story adds nothing to the schema

**No table, column, constraint, index, trigger, view, function or migration is created, changed or dropped by this story.** The story reads one order and its pairs (both built by "Checkout and placing an order", S5) and answers one `POST` that **writes nothing** until "Online payment with VNPay" (S7) exists. Every fact the page and the route need is already stored:

| What the page or the route needs to know | Where the schema already says it |
|---|---|
| "This caller holds a token that opens this order" | `order_tokens.token_hash` (`uq_order_tokens_token_hash`, 32 bytes, S5 ERD §2.3) joined to `orders.id` by `order_tokens.order_id` |
| The order's number, placed time, status, total, way to pay, paid flag, delivery, carrier, tracking, expected date, customer and city | `orders` (`number`, `placed_at`, `status`, `total`, `payment_method_id`, `paid`, `delivery_option_id`, `carrier`, `tracking`, `eta`, `customer`, `city`) — project ERD §3.12 |
| Every pair: shoe, size, quantity, price as sold | `order_items` (`product_id`, `size`, `qty`, `unit_price`, `line_no`) — project ERD §3.13 |
| A pair's name, colourway and picture, read **live** | `products` (`name`, `colourway`, `image`) by `order_items.product_id` (`RESTRICT`, so the shoe is always there, even when it is a draft) |
| The names of the delivery option and of the way to pay | `delivery_options.name/days`, `payment_methods.name` — read by the page through `GET /v1/delivery` and `GET /v1/payments` (S3), **not** joined into the order |
| "Cancelled", "paid" | `orders.status = 'cancelled'` (`ck_orders_status`), `orders.paid` (`NOT NULL DEFAULT false`) |
| "Its way to pay is not an online one the shop completes" | **Not stored.** It is code: the ordered list of ways to pay the shop offers (`["vnpay"]`, `orders` package, S5 services §3.4). See decision E2 |
| "No payment page was ever asked for" | Nothing is stored for a press of "Pay now". `payment_attempts` is S7's table and does not exist yet |

## 2. Entities this story touches — none changed

| Entity (defined elsewhere) | Read by this story | Written by this story |
|---|---|---|
| `orders` | `id`, `number`, `placed_at`, `status`, `paid`, `total`, `delivery_fee`, `delivery_option_id`, `payment_method_id`, `carrier`, `tracking`, `eta`, `customer`, `city` — by the one access statement (services §4). **Never** `phone`, `email`, `address`, `note`, `is_sample`, `idempotency_key`, `account_id` | Nothing |
| `order_tokens` | `token_hash`, `order_id` (the `EXISTS` of the access statement) | Nothing. Only S5's placement, a replay and the track story insert rows |
| `order_items` | `line_no`, `product_id`, `size`, `qty`, `unit_price` (S5's `Read`, unchanged) | Nothing |
| `products` | `name`, `colourway`, `image` (S5's `Read`, unchanged) | Nothing |
| `payment_methods` | **Not read by the API** for this story. `payment_methods.online` is *not* the test of "an online one the shop completes" (E2) | Nothing |
| `payment_attempts`, `payment_notices` | Do not exist yet (S7's migration) | Nothing — no row, no table |

`POST /v1/orders/{id}/payment` is a **read-only call** in this story: no `UPDATE`, no `INSERT`, no lock, no transaction. An order is exactly as it was after any number of presses (SC-2 to SC-5: "nothing changes"; SC-5: "still unpaid, no `payment_attempts` row").

## 3. Access patterns

| # | Pattern | Frequency | Index used |
|---|---|---|---|
| 1 | **Authorise and read the order's row**: `SELECT <public columns> FROM orders o WHERE o.id = $1 AND EXISTS (SELECT 1 FROM order_tokens t WHERE t.order_id = o.id AND t.token_hash = $2)` — one statement, one row or none. Used by `GET /v1/orders/{id}` (as its first step) and by `POST /v1/orders/{id}/payment` (as its only statement) | High (every order page, refresh and "Pay now") | `orders` primary key; `uq_order_tokens_token_hash` (a lookup by hash — no comparison of the secret itself runs). The row is read by id, so the planner never scans |
| 2 | The order's pairs with each shoe's live name, colourway and picture (`GET` only; S5 ERD §3 pattern 2, unchanged) | High | `uq_order_items_order_id_line_no`; `products` primary key |

Nothing here needs another index: both statements are by primary or unique key. **No index is added**, and none is dropped.

## 4. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| E1 | Should pressing "Pay now" leave a row (an audit of requests, a counter of attempts, a timestamp on the order) so the dealer can see who tried to pay? | **No.** This story's call writes nothing. S7 inserts one `payment_attempts` row per call because VNPay needs a unique reference; that row, not a second one, is the record of an attempt | A `pay_requests` table or `orders.last_pay_pressed_at`: a write on a path whose only job before S7 is to refuse, with no requirement that reads it (no criterion asks the dealer to see presses), and a second record that S7's attempts would then contradict |
| E2 | How does the shop know that an order's way to pay is "not an online one the shop completes" (SC-4: `cod`, `bank`, `momo`)? | **Code, not data**: `id` of the order's way to pay is in the ordered list of ways the shop offers (`["vnpay"]`, shared with `GET /v1/payment-options`). The seed's `payment_methods.online` is **not** used: `bank` and `momo` are `online = true` there ("pay online" describes the method, not that the shop can complete it), so reading the column would send a `bank` or `momo` order to "not available yet" and fail SC-4. The project ERD §3.7 already says whether a method is offered "is not stored: it is code plus the owner's settings" | Reading `payment_methods.online`: wrong for two of the three methods SC-4 names. A new column `payment_methods.completed`: a stored copy of a rule that depends on the owner's keys and on which provider code exists, which would drift the day "More ways to pay" ships a provider |
| E3 | Does "unpaid and cancelled", "paid and cancelled" or "expired" need a status value or a column? | **No.** `status = 'cancelled'` and `paid` already say it; the two can both be true (an order cancelled after payment, refunded in VNPay's portal) and the route's answer for that case is the cancelled sentence (services §5 and decision P3). Expiry (S7, 30 minutes) is `placed_at + interval '30 minutes'`, computed, never stored on the order | `orders.payable boolean` or `orders.expires_at`: a derived value stored beside the values it derives from |
| E4 | Index the token lookup further (`order_tokens (order_id, token_hash)`)? | **No.** `uq_order_tokens_token_hash` finds the one row by the hash; the other column is checked on that row. A composite index would double the write cost of every token insert (placement, replay, every lookup of the track story) to speed a read that is already a single index probe | A composite or covering index: no measurable gain on a table that holds a few thousand rows per thousand orders |
| E5 | Which lock does a future attempt take, so S7 and the cancel story do not deadlock with each other and with placement? | Nothing is locked here. For **S7**: the attempt's transaction takes the **order's row** `FOR UPDATE` first, re-checks `paid` / `status` under it, then inserts the attempt. It follows S5's global lock order (`products` → `product_sizes` → `counters` → inserts) because it never touches the earlier tables, and the cancel story's order (`orders` → `product_sizes`, S5 services §8) because it takes `orders` first | Re-checking without a lock: a notice marking the order paid at the same instant could let an attempt be created for a paid order |

### Conflicts with merged decisions

None. I read `erd/_base.md` §3.7, §3.12–3.14 and decisions D4, D5; `erd/checkout-and-placing-an-order.md` §2.3, §3 and decisions E1–E7; `erd/site-data-from-the-database.md` for the four collections the page reads; and `services/_base.md` §2.2, §3.11, §3.13. Nothing in them is restated or changed.

## 5. Migration plan

| Step | Forward | Backward | Safe on a populated table |
|---|---|---|---|
| This story | **No migration file.** The next free migration number is not consumed; S7 and the later stories stamp theirs as the project ERD says | Revert the backend PR; nothing in any database changes | Not applicable: no statement is added to any migration, no table, row or index is touched. The story is deployable and revertable with the schema exactly as S5 left it |

No merged migration is edited (convention `migrations`).

## 6. Arranging test data (no staff screen or dealer session exists yet)

The story sets an order's state directly in the test database. These are the only statements a test needs; none touches the schema and every one is undone by the case that ran it.

```sql
-- A token the test knows, for any order (sample or placed). sha256(bytea) is built in since PostgreSQL 11.
INSERT INTO order_tokens (order_id, token_hash)
VALUES ('doi-24816', sha256(convert_to('tok-doi-24816-a', 'UTF8')));
-- A state. The API enforces the allowed steps; the database allows any of the six statuses.
UPDATE orders SET status = 'shipped', paid = true, carrier = 'GHN', tracking = 'GHN-PT27XM', eta = '2026-11-26' WHERE id = 'doi-24817';
-- Clean up
DELETE FROM order_tokens WHERE order_id = 'doi-24816' AND token_hash = sha256(convert_to('tok-doi-24816-a', 'UTF8'));
```

A token the test makes up must be 1 to 128 characters of `[A-Za-z0-9_-]` (the API treats anything else as no token). The 14 seeded orders have no token until a test inserts one. By way to pay, as seeded: `cod` — DOI-24816 (placed, unpaid), DOI-24814 (packed, unpaid), DOI-24811 (cancelled), DOI-24808 and DOI-24804 (delivered, paid); `bank` — DOI-24812, DOI-24807; `momo` — DOI-24817, DOI-24813, DOI-24809, DOI-24805; `vnpay` — DOI-24815, DOI-24810, DOI-24806. An unpaid `bank` or `momo` order is made by `UPDATE orders SET paid = false WHERE id = …`. A test that only updates and inserts tokens does **not** move `counters`; one that inserts an *order* by hand must (S5 ERD §2.4).

## 7. Hand-over

| Who | What they must know from this design |
|---|---|
| Dev, backend step | **No migration.** The whole backend change is one route and one function (services §3–§6). The schema is exactly S5's |
| Test lead | SC-5's "no `payment_attempts` row exists" cannot count rows of a table that does not exist yet: assert `to_regclass('payment_attempts') IS NULL OR (SELECT count(*) FROM payment_attempts WHERE order_id = $1) = 0`, so the same case stays valid after S7 |
| "Online payment with VNPay" (S7) | Creates `payment_attempts` and `payment_notices`; takes the order row `FOR UPDATE` before inserting an attempt (E5); adds its expiry refusal as one case in the route's decision (services §5); changes no column of `orders` this story reads |
| "Customer accounts", "Dealer sign-in" | Extend the one access statement and function (services §4) — add `OR o.account_id = $3` and the dealer branch there, nowhere else; `orders.account_id` is their migration |
