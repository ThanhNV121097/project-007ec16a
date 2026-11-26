# ERD — Track an order by number and phone (plan item 7)

Module `orders` · Requirements ORD-020 to ORD-023 · Story `docs/orders/stories/track-an-order-by-number-and-phone.md` · UI PR #71 · Engine PostgreSQL 16, pgx v5, hand-written SQL.

## 1. Verdict: this story adds nothing to the schema

**No table, column, constraint, index, trigger, view, function or migration is created, changed or dropped by this story.** It is one route, `POST /v1/order-lookups`, that reads one `orders` row by its number, compares a phone in code, inserts one `order_tokens` row and moves two `rate_limits` counters. Every one of those is already in the schema.

| What the route needs | Where the schema already says it |
|---|---|
| Find the order by the typed number's digits | `orders.number` (`DOI-<n>`, `ck_orders_number_format`) read by `uq_orders_number`: `WHERE number = 'DOI-' \|\| $1` with `$1` the typed digits, one row or none (project ERD §3.12) |
| The stored phone to compare with | `orders.phone` (private, `NOT NULL`). It is read into the transaction's memory, compared by `PhoneKey` (digits, a leading `84` read as `0`, whole phone) and never returned, logged or stored again |
| The id, number and time of a right order | `orders.id`, `orders.number`, `orders.placed_at` (the answer is those three and the token) |
| Whether the order is a guest's, another account's or none | `orders.account_id` (**S14's column**, migration `20261125150000_orders_account_id`). The shared matcher reads it for `POST /v1/account/orders`; this route ignores it. It is the one reason this story's backend waits for S14 (services file §4) |
| A new token for the order | `order_tokens (order_id, token_hash)`: `token_hash` is `bytea`, 32 bytes, unique (`uq_order_tokens_token_hash`); `order_id` is a `CASCADE` foreign key (project ERD §3.14, decision D5: "the lookup mints a new token") |
| Five failures in 15 minutes per number and per address, a 15-minute lock, a wait to report | `rate_limits` (project ERD §3.21): scopes `order_lookup_number` and `order_lookup_address` are already in `ck_rate_limits_scope` |

## 2. Entities this story touches — none changed

| Entity (defined elsewhere) | Read | Written |
|---|---|---|
| `orders` | `id`, `number`, `placed_at`, `phone`, and (through the shared matcher) `status`, `total`, `account_id` — by one statement, by `number` | Nothing. A lookup never changes an order |
| `order_tokens` | Nothing | **One `INSERT` per right pair** (`order_id`, `token_hash`). Never an update, never a delete; earlier tokens stay valid |
| `rate_limits` | `locked_until` of two rows (`Check`) | **One upsert per miss and per scope** (`Fail`): `order_lookup_number` (only when the typed number has digits) and `order_lookup_address`. A right pair, a refused try and a request refused before the match write nothing |
| `order_items`, `products`, `users`, `sessions` | Nothing | Nothing |

### 2.1 Subjects of the two counters

`rate_limits.subject_hash = sha256(scope ‖ ':' ‖ subject)` (`ratelimit.Subject`). For this story:

| Scope | Subject text | Notes |
|---|---|---|
| `order_lookup_number` | the **typed number's ASCII digits**, 1 to 18 of them (`24817` for `DOI-24817`, `doi 24817` and `  DOI - 24817 `) | A number that does not exist counts like any other (ORD-022 item 2), which is why the subject is what was typed and never an order id. Leading zeros are kept (`00024817` is another subject and matches no order), so a counter cannot be dodged by padding a number |
| `order_lookup_address` | `httpx.ClientAddr(r)`, the right-most `X-Forwarded-For` entry | Never the left-most (the client writes it) |

A number with no digits, or with more than 18, counts on the address scope only: a shared bucket for "no digits" would let one visitor lock every other empty try. Nothing here is stored in clear: the table holds neither a number nor an address.

## 3. Access patterns

| # | Pattern | Frequency | Index used |
|---|---|---|---|
| 1 | Take the number's lock: `SELECT pg_advisory_xact_lock(hashtextextended('order_lookup:' \|\| $1, 0))` | Once per try that has digits | None (a lock in memory, released at the end of the transaction) |
| 2 | Is the number or the address locked: `SELECT ceil(extract(epoch FROM locked_until - now()))::int FROM rate_limits WHERE scope = $1 AND subject_hash = $2 AND locked_until > now()` — twice | Every try | `rate_limits` primary key `(scope, subject_hash)`: one row each |
| 3 | Find the order: `SELECT … FROM orders WHERE number = 'DOI-' \|\| $1` | Every try with digits that is not refused | `uq_orders_number`: one index probe, never a scan |
| 4 | Count a miss: `ratelimit.Fail` (one `INSERT … ON CONFLICT DO UPDATE` per scope; the SQL is merged and is not restated here) | Every miss | `rate_limits` primary key |
| 5 | Give the order a token: `INSERT INTO order_tokens (order_id, token_hash) VALUES ($1, $2)` | Every right pair | `order_tokens` primary key; the unique index on `token_hash` is checked on the insert |

All five run in **one transaction at the default isolation (`READ COMMITTED`)** — the transaction `orders.Attempt` opens (services file §4). The number's advisory lock is what serialises tries for one number, and the second of two waiting tries must **see the first one's committed counter**: a `REPEATABLE READ` or `SERIALIZABLE` transaction would take its snapshot before the wait and would count a stale value, which is how a sixth failure would get through. No statement is slow: every one is a probe by key.

**No index is added and none is dropped.** `idx_rate_limits_window_started_at` already serves the hourly purge of the counters this route creates (`auth.RunPurge` calls `ratelimit.Purge`).

## 4. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| E1 | Keep a log of lookups (who looked, from where, when) for the owner? | **No.** The counters are the only record, and they hold hashes | An `order_lookups` table: it would store a phone-and-address trail for a feature that no requirement asks the owner to read, and a second place that must be purged |
| E2 | Mark a token with the lookup that made it, an expiry or a last-used time? | **No.** A token opens one order and does not expire (SRS; project ERD §3.14 "no expiry is specified"). Marks would be columns nothing reads | `order_tokens.created_via`, `expires_at`: a rule the SRS does not have, and an expiry would silently break "Ordered on this phone", which holds tokens in the browser for good |
| E3 | Reuse a customer's earlier token instead of minting a new one? | **Cannot.** Only a hash is stored, so an earlier token is unrecoverable; a new one is the only way to hand a token back (project ERD D5) | Storing the token to hand it back: a database leak would then open every order |
| E4 | Index or generate a column of the stored phone's digits so the phone can be matched in SQL? | **No.** The match is by number first (unique), so exactly one row's phone is compared, in Go, by the one `PhoneKey` function that the account story shares. A second copy of the rule in SQL would be a second implementation that must be kept equal, and a stored digits column is one more copy of a private value | A generated `phone_key` column with an index: faster for a query nobody makes (a lookup by phone alone is out of scope) and a duplicate of `orders.phone` |
| E5 | Cap the number of tokens an order may hold? | **No** (accepted risk R-1). A cap would have to refuse a right pair, which the contract says always succeeds, or delete a token the customer's other browser holds | A cap of N tokens per order: breaks a device the customer did not touch |

**Accepted risk R-1.** A right pair mints one `order_tokens` row per call and a right pair neither counts nor is limited (ORD-022 items 1 and 5), so a person who already holds a number and its phone can add rows without bound. Each row is 32 bytes of hash plus a uuid and a time, and the rows go with the order (`ON DELETE CASCADE`). The caller must already know the phone, so this is not a way in for a stranger; if the owner ever sees growth, the fix is a per-order cap decided with a product rule, not a schema change made here.

### Conflicts with merged decisions

None. I read `erd/_base.md` §3.12, §3.14, §3.21 and decisions D5, D8; `erd/customer-accounts-sign-up-sign-in-my-orders.md` (the `orders.account_id` column and the matcher's statement A7) and `erd/checkout-and-placing-an-order.md` (the token's insert). Nothing in them is restated or changed.

## 5. Migration plan

| Step | Forward | Backward | Safe on a populated table |
|---|---|---|---|
| This story | **No migration file.** The next free migration number is not consumed | Revert the backend PR; nothing in any database changes. Tokens and counters it made are ordinary rows of existing tables, valid and harmless without the route | Not applicable: no statement is added to any migration; no table, row or index is touched at deploy time |

No merged migration is edited. **Dependency, not a migration:** the shared matcher selects `orders.account_id`, so S14's migrations `20261125150000_orders_account_id` and `20261125150100_orders_account_id_index` must be on `main` before this story's backend merges (services file §4).

## 6. Arranging and reading test data

No dealer session and no test switch exists in this story (project decision S4: no backend switch for delay, failure or lock). These statements are all a test needs; none touches the schema and each is undone by the case that ran it.

```sql
-- A clean slate for the lookup counters (the limits count per address, so run it before each case that tries a lookup)
DELETE FROM rate_limits WHERE scope IN ('order_lookup_number', 'order_lookup_address');

-- SC-24 / SC-25: the order's state, set directly
UPDATE orders SET status = 'shipped', carrier = 'GHN', tracking = 'GHN-PT27XM', eta = '2026-11-26' WHERE id = 'doi-24817';
UPDATE orders SET status = 'cancelled' WHERE id = 'doi-24817';

-- SC-9: move a lock back in time (14 or 16 minutes) instead of waiting; the database's now() is the only clock
UPDATE rate_limits
   SET window_started_at = window_started_at - interval '14 minutes',
       locked_until      = locked_until      - interval '14 minutes'
 WHERE scope IN ('order_lookup_number', 'order_lookup_address');

-- SC-10 / SC-8: the stored count for one number (24817)
SELECT hits, locked_until FROM rate_limits
 WHERE scope = 'order_lookup_number' AND subject_hash = sha256(convert_to('order_lookup_number:' || '24817', 'UTF8'));

-- SC-4 / SC-5 / SC-6: tokens made for an order (a miss and a refused try make none)
SELECT count(*) FROM order_tokens WHERE order_id = 'doi-24817';
-- the token text is in no table: its hash is the row, and the token itself is never equal to it
SELECT count(*) FROM order_tokens WHERE token_hash = sha256(convert_to($1, 'UTF8'));   -- $1 = the token from the answer: 1
```

A lock exists for 15 minutes from the fifth failure (`locked_until = now() + 15 minutes` at that moment), so moving both columns back by 14 minutes leaves 1 minute of lock (`Retry-After` 1 to 60) and by 16 minutes ends it: the next miss then counts as the first failure of a new count (`Fail` resets a row whose lock has ended). The 14 sample orders have no token until a test mints one with the route.

## 7. Hand-over

| Who | What they must know from this design |
|---|---|
| Dev, backend step | **No migration.** One route, one handler file and one line in `Register`, over the matcher S14 creates (services file §2, §4). Do not write a second matcher |
| Test lead | The statements of §6 are the whole fixture; subjects are `sha256('<scope>:<digits or address>')` |
| Customer accounts (S14) | Nothing to change: this story adopts `lookup.go`, `phone.go` and the scope constants as S14 merges them |
