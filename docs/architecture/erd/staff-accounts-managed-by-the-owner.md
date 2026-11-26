# ERD — Staff accounts managed by the owner (plan item 14, "S13")

Module `admin` · Requirements ADMIN-030, ADMIN-031, ADMIN-004 (account-endpoint half) · Story `docs/admin/stories/staff-accounts-managed-by-the-owner.md` · UI PR #36 · Engine PostgreSQL 16, pgx v5, hand-written SQL.

## 1. Verdict: this story adds nothing to the schema

**No table, column, constraint, index, trigger, view, function or migration is created, changed or dropped.** The story's story file expected this ("no new table is expected"), and the design confirms it: every fact the four endpoints need is already in `users` and `sessions` (project ERD §3.18, §3.19, created by "Dealer sign-in and protected admin area", migration 7 `users_and_sessions`) or in the one foreign key `uploads.uploaded_by` (§3.20).

| What the endpoints need to know or keep | Where the schema already says it |
|---|---|
| An account: name, email, role, switched on or off | `users.name`, `email`, `role` (`ck_users_role`), `is_active` |
| "Not a second admin", "never without an admin" | `uq_users_one_admin` (at most one admin row). "Never zero" is code: no statement of this story can touch a row whose role is not `staff` (decision S4) |
| "The email belongs to no account of any role, any case" | `uq_users_email` on the stored lower-case email (`ck_users_email_lowercase`). The unique index is the arbiter of the add race (decision S2) |
| A password kept as a salted slow hash | `users.password_hash` (argon2id string, `<> ''`) |
| A switched-off, reset or removed account loses its sessions | `sessions` rows keyed by `user_id`: `idx_sessions_user_id` finds them, `ON DELETE CASCADE` removes them with the user |
| "Nothing the person saved is removed" when staff are deleted | `uploads.uploaded_by` → `users(id)` `ON DELETE SET NULL` (§3.20); `orders.account_id` → `users(id)` `ON DELETE SET NULL` (§3.12, never set for a team member) |
| The team list, admin first, then by creation | `users.role`, `users.created_at`, `users.id` (decision S3: no new index) |

## 2. Entities this story touches — none changed

| Entity (defined elsewhere) | Read by this story | Written by this story |
|---|---|---|
| `users` | `id`, `name`, `email`, `role`, `is_active`, `created_at` (the list, the target lookup, the answer to a write). **Never** `phone`, `address`, `city`; `password_hash` is read by no statement of this story | `INSERT` (role `staff`, always); `UPDATE name`, `is_active`, `password_hash` of a row whose role is `staff`; `DELETE` of a row whose role is `staff`. Every such statement carries `role = 'staff'` in its `WHERE` |
| `sessions` | Nothing | `DELETE … WHERE user_id = $1` when an account is switched off or its password is reset, in the same transaction as the `users` update. Rows also leave by `CASCADE` when the user is deleted |
| `uploads` | Nothing | Nothing. Its `uploaded_by` is cleared by the database when a staff row is deleted (S9's table, if it exists by then) |
| `orders` | Nothing | Nothing. `account_id` is `NULL` for a team member's order; a staff delete changes no order, shoe, line, category or review (SC-33) |
| `rate_limits` | Nothing | Nothing. These routes are admin-only and are not limited (decision S12) |

## 3. Access patterns

| # | Pattern | Frequency | Index used |
|---|---|---|---|
| 1 | **The team page**: `SELECT id, name, email, role, is_active FROM users WHERE role IN ('admin','staff') ORDER BY (role <> 'admin'), created_at, id LIMIT $1 OFFSET $2`, and `SELECT count(*) FROM users WHERE role IN ('admin','staff')`, both in one `REPEATABLE READ, READ ONLY` transaction so `total` and `items` agree | Rare (admin opens the page; "Try again") | **Scan** of `users`: a handful of team rows among the customers. At 100,000 customers a scan of a narrow table is a few tens of milliseconds, and the page is opened a few times a day. `idx_users_team (role) WHERE role <> 'customer'` is the one ERD §3.18 names for later; it stays unbuilt (decision S3) |
| 2 | **Who is this id** (before the body is read, and again under a lock): `SELECT … FROM users WHERE id = $1` (`FOR UPDATE` inside a write transaction) | Per `PUT` / `DELETE` | `users` primary key |
| 3 | **Is this email taken** (so a 422 can name `email` beside the other faulty fields): `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)` | Per `POST` that passes the shape rules | `uq_users_email` |
| 4 | **Create**: `INSERT INTO users (name, email, password_hash, role) VALUES ($1,$2,$3,'staff') RETURNING …` | Rare | `uq_users_email` (the constraint is also the race arbiter, S2) |
| 5 | **Change**: `UPDATE users SET name = $2` / `SET is_active = $2` / `SET password_hash = $2` `WHERE id = $1 AND role = 'staff'` (one statement per field sent, one transaction); then, if the account was switched off or its password reset, `DELETE FROM sessions WHERE user_id = $1` | Rare | `users` primary key; `idx_sessions_user_id` |
| 6 | **Remove**: `DELETE FROM users WHERE id = $1 AND role = 'staff'` (sessions go by `CASCADE`, `uploads.uploaded_by` and `orders.account_id` by `SET NULL`) | Rare | `users` primary key; `idx_sessions_user_id` for the cascade; `uploads.uploaded_by` has no index (ERD §3.20: a scan of at most a few hundred rows) |
| 7 | *Relied on, owned by "Dealer sign-in"*: every request's `sessions ⨝ users` read, and sign-in's session insert (§4, rules R1 and R2) | Very high | `uq_sessions_token_hash`; `users` primary key |

Nothing here needs another index: statements 2–6 are by primary or unique key, and 1 is a scan whose size is the customers' count and whose frequency is a few reads a day. **No index is added, none is dropped.**

## 4. Concurrency: what is locked, in which order, and what each race ends as

**Lock order for this story and for sign-in: `users` row first, then `sessions` rows.** `Resolve` takes no lock (a plain `SELECT`, snapshot per statement), so a busy site never waits on a staff change. A statement of this story that deletes `sessions` rows runs **after** the `UPDATE users` of the same transaction has taken the row lock, and in `READ COMMITTED` every statement sees what had committed before it began — so a session that sign-in committed while the update waited is seen and deleted.

| Race | What happens | Ends as |
|---|---|---|
| Two `POST`s of one email | Both pass the pre-check (§3 #3); both `INSERT`; the unique index lets one in and the other waits, then fails with `23505` on `uq_users_email` | One `201`; the other `422` naming `email` (the handler maps exactly that constraint name; any other `23505` is `500`). One row (SC-20) |
| **Sign-in** × **switch-off** | Sign-in creates its session with one statement that locks the user row and re-checks it: `INSERT INTO sessions … SELECT … FROM users u WHERE u.id = $1 AND u.is_active AND u.password_hash = $3 FOR SHARE OF u` (rule R2). `FOR SHARE` conflicts with the `UPDATE` of a switch-off | If the switch-off holds the row first, the insert waits, re-reads the committed row, finds `is_active = false`, inserts **nothing**, and sign-in answers as a wrong password. If sign-in is first, the switch-off's `UPDATE` waits for it, then its `DELETE FROM sessions` sees and removes the new session. **No `sessions` row ever belongs to a switched-off user** — the invariant the test asserts (SC-30) |
| Sign-in × **password reset** | Same statement; its `u.password_hash = $3` is the hash the sign-in verified | A reset that commits first changes the hash, so the insert adds nothing and sign-in fails; a sign-in that commits first has its session deleted by the reset. The old password never leaves a live session behind |
| Sign-in × **remove** | `FOR SHARE` conflicts with `DELETE`; the foreign key also takes a key lock | The insert finds no row (or fails the foreign key) and adds nothing; or the delete waits and the cascade removes the session |
| `PUT` × `PUT`, `PUT` × `DELETE` | Each `PUT` transaction first locks the target (`SELECT … FOR UPDATE`, §3 #2) | Serialised. A `PUT` that waits behind a `DELETE` re-reads, finds no row and answers `404`; two `PUT`s apply in turn, the later value of a field wins |
| The 10-minute purge of expired sessions × a switch-off | Both delete `sessions` rows; the purge locks them in index order, the switch-off by `user_id` | Rows touched by both can deadlock (`40P01`) once in a long while. PostgreSQL aborts one; the handler retries its transaction **once** (services §5.1) and the purge simply runs again in ten minutes |
| `EnsureAdmin` at start-up × a staff add | `EnsureAdmin` writes only the row of role `admin` | No shared row; an add of `ADMIN_EMAIL`'s address before start-up is a taken email (`422`) |

Isolation is `READ COMMITTED` everywhere except the team page's read-only snapshot. The password is hashed **before** any transaction opens (hashing takes tens of milliseconds and 19 MiB), so no row lock is held while it runs, and a request that fails validation is never hashed.

## 5. Decisions

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| S1 | Does the story need a table, column or migration (a `created_by`, an audit log, a "last signed in", a `password_changed_at`)? | **No.** Nothing in the SRS reads one: no log of who added or removed whom, no last-sign-in time, no list of sessions (story §3, out of scope) | `users.created_by` / an audit table: written by every action and read by nothing, with a foreign key that would then make removing the admin's staff harder. Add it with the requirement that reads it |
| S2 | Who decides that two adds of one email are one add? | **The unique index.** The handler pre-checks the email only so a `422` can name every faulty field at once (a short password and a taken email in one answer, SC-18); the `INSERT` still catches `23505` on `uq_users_email` and gives the same `422` sentence. Any other constraint violation is `500` | An advisory lock or `SERIALIZABLE` around check-then-insert: holds a lock through a hash and still needs the index for a second instance. Mapping the race to `409`: the SRS and `services.md` §3.26 say `422` naming `email` |
| S3 | Index the team list (`idx_users_team`)? | **No.** The list is read a few times a day, over a table whose size is the customers' count; ERD §3.18 already names the index and the volume at which it would earn its place (≈100,000 customers). Building it now would slow every sign-up for a read nobody waits on | `CREATE INDEX CONCURRENTLY idx_users_team … WHERE role <> 'customer'`: a migration and a write cost per sign-up for no promised volume |
| S4 | Where does "the shop is never left without an admin" live? | **In every statement and in one pre-check.** The handler answers `409` for any target whose role is `admin` before the body is read; and, whatever that check says, every `UPDATE` and `DELETE` of this story has `role = 'staff'` in its `WHERE`, so a bug in the check still cannot touch the admin row. `uq_users_one_admin` stops a second admin from being made | A trigger refusing `DELETE` or `is_active = false` on an admin row: `EnsureAdmin` legitimately updates that row (email, password), the trigger would need an escape hatch, and it is a second place to keep in step with a rule that is already in every statement |
| S5 | How does an account end its sessions "at once"? | **Delete the `sessions` rows in the same transaction** as the switch-off or reset (`auth.SetActive(false)` and `auth.SetPassword` do it themselves, so no caller can forget), and let `Resolve` re-read `users.is_active` on every request (rule R1). Switch-on restores nothing: the rows are gone | `users.session_epoch` / `credentials_version` compared on every request: one more column and one more compare on the hottest path, and the old rows would stay in the table until the purge. Relying on `Resolve`'s `is_active` filter alone: a switch-on would bring the old cookies back to life (SC-29 forbids it) |
| S6 | What does the admin's own row show as its name? | **The same display name `/v1/auth/me` gives**: the stored name, or — when it is empty, as it is for the admin made from settings — the email's local part with its words capitalised (`auth.DisplayName`, services §3.7). Nothing is stored; the list, `POST` and `PUT` answers all apply the same function | Returning the raw `''`: the first row of the table would have an empty name cell (SC-1), and the one rule `/v1/auth/me` already has would be written twice differently. A stored display name written at start-up: a copy that drifts when `ADMIN_EMAIL` changes |
| S7 | Which emails can be stored? | **ASCII only**, trimmed, lower-cased by an ASCII-only fold (`A`–`Z`), exactly one `@`, a non-empty local part, a domain with a dot that is neither first nor last, no space or control character, ≤ 200 characters (`auth.NormalizeEmail`, shared with sign-in and sign-up). Anything else is "That does not look like an email address." | Go's `strings.ToLower` on any Unicode address: Go and PostgreSQL's `lower()` disagree on some non-ASCII letters, and `lower()` depends on the database's locale (an Alpine image may run in the `C` locale). An address Go lowered but `lower()` would not leaves `ck_users_email_lowercase` failing — a `500` where the person should see a sentence. Relaxing the merged `CHECK`: it is the guarantee that case-insensitive uniqueness holds |
| S8 | Who may be deleted, and what is kept? | A row whose role is `staff` only. The account and its sessions go; `uploads.uploaded_by` and `orders.account_id` are set to `NULL` by their own foreign keys; nothing else references `users` | Soft delete (`deleted_at`): the project uses it nowhere (ERD §2), and the SRS wants the email free again at once (ADMIN-031 item 4) |

### Conflicts with merged decisions

None. I read `erd/_base.md` §2, §3.12, §3.18–3.20, §5 patterns 15–17, §8 migration 7 and the lock rules; `services/_base.md` §1.1, §2.2, §2.5, §3.7, §3.9, §3.26, §3.27. Two statements are **refinements of silence**, not overrides: S6 (the list's `name` for an account with no stored name) and S7 (what "looks like an email" means for non-ASCII text); neither changes a column or a merged rule.

## 6. Migration plan

| Step | Forward | Backward | Safe on a populated table |
|---|---|---|---|
| This story | **No migration file.** The next free timestamp is not consumed | Revert the backend PR; nothing in any database changes | Not applicable: no statement is added to any migration. Deployable and revertable with the schema exactly as "Dealer sign-in" (migration 7) left it. Rows created through the endpoints are ordinary `users` rows and need no clean-up on a revert |

This story **depends on** migration 7 having made `users` and `sessions` as `erd/_base.md` §3.18–3.19 say. The backend PR therefore ships one test, `TestStaffSchemaAssumptions` (skips without `TEST_DATABASE_URL`), that reads the catalogue and fails loudly if any of these is not true: `uq_users_email` is unique on `email`; `uq_users_one_admin` is a unique partial index `WHERE role = 'admin'`; `ck_users_email_lowercase` and `ck_users_role` exist; `sessions.user_id` references `users` with `ON DELETE CASCADE`; `idx_sessions_user_id` exists; and, when the table `uploads` exists, `uploads.uploaded_by` references `users` with `ON DELETE SET NULL` and `orders.account_id` (when the column exists) the same. No merged migration is edited (convention `migrations`).

**As built (PR #56, merged).** The story shipped **no migration**: `code/backend/migrations/` holds `20261125140000_users_and_sessions` as its last user-table file and nothing named for this story. `TestStaffSchemaAssumptions` is in `internal/admin/staff_db_test.go` and skips without `TEST_DATABASE_URL`. The ERD was checked again against the code: every statement of `internal/auth/accounts.go` is one of access patterns 1–6 of §3 (the list is the `ORDER BY (role <> 'admin'), created_at, id` read; every write carries `role = 'staff'`), so §1–§5 stand unchanged.

## 7. Arranging test data (no customer sign-up exists until "Customer accounts" lands)

No statement below touches the schema; each is undone by the case that ran it.

```sql
-- A customer row whose hash the backend itself wrote, so it signs in with a password the test knows:
-- 1) make an account through POST /v1/admin/staff (name, email tc-customer@example.com, password customer-pass-1) as the admin;
-- 2) turn it into a customer (the sign-in endpoint serves any row of any role):
UPDATE users SET role = 'customer' WHERE email = 'tc-customer@example.com';
-- A switched-off staff row that has never had a session:
UPDATE users SET is_active = false WHERE email = 'tc-sb@example.com';
-- The invariant of SC-30 (must be 0 after any number of sign-ins racing any number of switch-offs):
SELECT count(*) FROM sessions s JOIN users u ON u.id = s.user_id WHERE NOT u.is_active;
-- What a removal must have done (SC-32): both counts 0 for the removed id
SELECT count(*) FROM users WHERE id = $1;  SELECT count(*) FROM sessions WHERE user_id = $1;
-- A picture the removed account uploaded, if the uploads table exists (SC-33): afterwards uploaded_by IS NULL and the row remains
INSERT INTO uploads (name, content_type, size_bytes, data, uploaded_by)
VALUES (repeat('a',32) || '.png', 'image/png', 1, '\x00'::bytea, (SELECT id FROM users WHERE email = 'tc-sa@example.com'));
-- Clean up (sessions go by CASCADE)
DELETE FROM users WHERE email LIKE 'tc-%' OR email IN ('lan@example.com','mai@example.com');
```

`SELECT password_hash FROM users …` is for the schema test and the "two equal passwords give two different hashes" case only; the answer of no endpoint may contain it, and a test that prints it must not be left in a log.

## 8. Hand-over

| Who | What they must know from this design |
|---|---|
| Dev, backend step | **No migration.** Four routes in `internal/admin/staff*.go`, the functions of services §5 in `internal/auth`, the swap of services §7. Two statements of "Dealer sign-in" are checked and, if they are not as rules R1 and R2 say, edited in this PR (services §5.2) |
| Test lead | The invariant query of §7 is the strongest assertion for SC-27, SC-29, SC-30 and SC-32: no session row may belong to a switched-off or removed user. The race cases are repeated rounds of two requests in the same instant (SC-20, SC-30) and cannot prove absence — the invariant after many rounds, plus the Go race test of services §6, together do |
| "Dealer sign-in and protected admin area" (item 9) | Its BE review checks rules R1 (`Resolve` requires `u.is_active`), R2 (the session `INSERT … SELECT … FOR SHARE OF u … AND u.password_hash = $3`), R3 (`EnsureAdmin` writes `is_active = true` on the admin row) and the shared `NormalizeEmail` of decision S7 |
| "Customer accounts" (item 15) | Uses the same `auth.NormalizeEmail` and `auth.HashPassword`; an address staff creation would refuse as non-ASCII is refused at sign-up with the same sentence, so no row can break `ck_users_email_lowercase` from either side. `orders.account_id` is its migration and is never set for a team member |
| "Manage shoes, lines, categories, reviews and orders" (item 10) | Its `uploads` migration must keep `uploaded_by … ON DELETE SET NULL` exactly as §3.20 says; the schema test above fails if it does not. It also creates the `internal/admin` package or finds this story's `staff.go` there; each registers its own routes (services §1) so neither edits the other's file |
