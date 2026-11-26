# Database design — Dealer sign-in and protected admin area (plan item 9, "S8")

Module `admin` · Requirements ADMIN-001 to ADMIN-004 · Story `docs/admin/stories/dealer-sign-in-and-protected-admin-area.md` · UI PR #34 · Contracts: `docs/architecture/services/dealer-sign-in-and-protected-admin-area.md`.

This file **extends** `erd/_base.md`. That file already designs `users` (§3.18), `sessions` (§3.19) and `rate_limits` (§3.21), and migration 7 (§8) says this story creates the first two. Nothing in them is redesigned or restated here. This file adds: the exact migration, the four places where I made a choice `_base.md` left open, the queries each index serves, and the write rules for the `users` row the server makes itself. `rate_limits` is **not touched**: S3's migration 1 created it with every scope this story and the later ones use, and its columns, checks and purge index are exactly what the rules below need.

## 1. What this story creates

| Object | From `_base.md` | This story's note |
|---|---|---|
| `users` | §3.18 — every column, check and index as written | Creates **all** of `_base.md`'s columns, including `phone`, `address`, `city`. The customer-accounts story (plan item 15) therefore needs no `ALTER TABLE users`; it only writes those columns (this closes the question its story file leaves to this design) |
| `uq_users_one_admin` | §3.18 / D9 | As written |
| `sessions` | §3.19 — every column, check, foreign key and index as written | As written |
| `trg_users_updated_at` | §3.0 | `BEFORE UPDATE … EXECUTE FUNCTION set_updated_at()`; the function is S3's migration 1, which is why this migration is stamped after it. `sessions` has no `updated_at` and no trigger |

No other table, view, function, extension or column changes. `orders.account_id` and `uploads.uploaded_by` (which point at `users`) belong to the stories that need them and are created **after** this migration.

## 2. Choices `_base.md` left open

| # | Choice | Reason, and what it costs |
|---|---|---|
| U1 | **`CHECK ck_users_email_ascii`: `email ~ '^[!-~]+$'`** (printable ASCII, no space) in addition to `ck_users_email_lowercase`. | The shared `auth.NormalizeEmail` (services file §3) refuses non-ASCII addresses so two spellings of one letter can never be two accounts; `lower()` in a `CHECK` is locale-dependent for non-ASCII, so the code rule needs a database backstop that does not depend on locale. Cost: an address with an accented letter cannot be an account (the owner's shop sells to people who type ASCII addresses; the staff design already assumes it). A row written by hand that breaks it fails with a constraint error, never silently |
| U2 | **No cap on rows in `sessions` per user.** | Two sign-ins at once must both succeed (SC-11) and a person has a phone and a laptop. Rows are 100 bytes, live 7 days and are purged hourly (§4). A cap would need a rule for *which* session to drop, which the SRS never asks for |
| U3 | **The admin row's `name` is `''`.** The displayed name is computed (services file §3.4), never stored. | `_base.md` §3.18 says so; recorded so the start-up code never writes `'Dealer'` into the column and later "fixes" it |
| U4 | **No `last_sign_in_at`, no failure counter and no address on `users` or `sessions`.** | Failure counting lives only in `rate_limits` (hashed). Storing an address would put a personal datum in a table nothing needs it from; the SRS reads none |

## 3. Migration plan

Files `code/backend/migrations/<UTC timestamp>_users_and_sessions.up.sql` and `.down.sql`, **stamped later than S3's `…seed_collections` (`20261125100200`)** and earlier than any later story that references `users`; suggested `20261125140000_users_and_sessions`. Applied on boot before the server listens (one transaction with its `schema_migrations` row); a merged file is never edited.

**Forward**

```sql
CREATE TABLE users (
    id            uuid        NOT NULL DEFAULT gen_random_uuid(),
    email         text        NOT NULL,
    name          text        NOT NULL DEFAULT '',
    phone         text        NOT NULL DEFAULT '',
    address       text        NOT NULL DEFAULT '',
    city          text        NOT NULL DEFAULT '',
    password_hash text        NOT NULL,
    role          text        NOT NULL,
    is_active     boolean     NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT ck_users_email_lowercase CHECK (email = lower(email) AND email <> '' AND length(email) <= 200),
    CONSTRAINT ck_users_email_ascii CHECK (email ~ '^[!-~]+$'),
    CONSTRAINT ck_users_password_hash_not_blank CHECK (password_hash <> ''),
    CONSTRAINT ck_users_role CHECK (role IN ('admin', 'staff', 'customer'))
);
CREATE UNIQUE INDEX uq_users_one_admin ON users ((true)) WHERE role = 'admin';
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE sessions (
    id         uuid        NOT NULL DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL,
    token_hash bytea       NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    CONSTRAINT pk_sessions PRIMARY KEY (id),
    CONSTRAINT uq_sessions_token_hash UNIQUE (token_hash),
    CONSTRAINT ck_sessions_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT ck_sessions_expiry_after_created CHECK (expires_at > created_at),
    CONSTRAINT fk_sessions_user_id FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX idx_sessions_user_id    ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);
```

**Backward** — `DROP TABLE IF EXISTS sessions; DROP TABLE IF EXISTS users;` (indexes, constraints and the trigger go with the tables; `set_updated_at()` stays, it is S3's). The server never runs it; it exists for a local reset.

**Safety on a populated table.** Both tables are new, so the forward file locks nothing that exists and is safe whatever else is in the database. Rolling it back deletes every account and session, which is why it is a local-reset file: in a deployment a wrong migration is fixed by a new file. The admin row is **not** made by the migration (it needs the owner's settings); the server makes it at start-up (§5), so an empty `users` table after migrating is correct.

**Order against other stories.** Needs only S3's migration 1 (`set_updated_at()`). It touches no object S4's or S5's migrations touch, so the files apply in either order; the runner applies whatever is pending by name.

## 4. Queries, and the index each one uses

All SQL is parameterised; the only strings spliced in are none. "The clock" is always the database's `now()`: the application never compares times itself, so a test moves stored timestamps instead of waiting (story §8).

| # | When | Statement (shape) | Index |
|---|---|---|---|
| Q1 | Every `/v1` request that carries a well-formed `session` cookie (`auth.Resolve`) | `SELECT u.id::text, u.email, u.name, u.role FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = $1 AND s.expires_at > now() AND u.is_active` | `uq_sessions_token_hash`, then the primary key of `users`. One row or none. A session exactly 7 days old has `expires_at = now()` and is not `>` it, so it is expired (SC-15) |
| Q2 | Sign-in, after the checks that cost nothing | `SELECT id::text, name, role, is_active, password_hash FROM users WHERE email = $1` | `uq_users_email` |
| Q3 | Sign-in, after the password verified | `INSERT INTO sessions (user_id, token_hash, expires_at) SELECT u.id, $2, now() + interval '7 days' FROM users u WHERE u.id = $1 AND u.is_active AND u.password_hash = $3 FOR SHARE OF u RETURNING expires_at` | primary key of `users`. Zero rows returned means the account was switched off, deleted or given a new password after Q2 read it: the answer is the generic 401 and **no session exists** (rule R2, §6) |
| Q4 | Sign-out | `DELETE FROM sessions WHERE token_hash = $1` | `uq_sessions_token_hash` |
| Q5 | In-process purge, hourly | `DELETE FROM sessions WHERE expires_at < now()` | `idx_sessions_expires_at` |
| Q6 | Ending every session of one user (switch-off, password reset, `EnsureAdmin` change) — by the later stories' functions and by `EnsureAdmin` | `DELETE FROM sessions WHERE user_id = $1` | `idx_sessions_user_id` |
| Q7 | Start-up | the `EnsureAdmin` statements of §5 | `uq_users_one_admin` (the `WHERE role = 'admin'` lookup) |
| Q8 | Sign-in limits | the `rate_limits` statements of the services file §4 | its primary key `(scope, subject_hash)`; the purge uses `idx_rate_limits_window_started_at` |

## 5. Writes to `users` made by the server itself: the admin row

`auth.EnsureAdmin` is the only code that writes the admin row. It runs once after migrations and before the listener starts, in **one transaction**, and never logs the password, the hash or (on a rejected address) the address.

| State found | Action | Sessions |
|---|---|---|
| `ADMIN_EMAIL` or `ADMIN_PASSWORD` unset or empty | Nothing. Logged (`admin account not made: settings missing`). **An admin row made by an earlier run stays and keeps working**: removing a setting never locks the owner out; changing the password is how to do that | unchanged |
| The address is not accepted by `NormalizeEmail`, or the password has fewer than 10 characters (counted in characters, not trimmed) or more than 100 | Nothing. Logged with the reason only (`admin password shorter than 10 characters`) | unchanged |
| No row with `role = 'admin'` | `INSERT` with the normalised email, `name = ''`, a fresh argon2id hash, `role = 'admin'`, `is_active = true` | — |
| A row exists, its email equals the setting, the password verifies against the stored hash, `is_active` is true | **Nothing is written** (no new salt, no `updated_at` change) | kept |
| A row exists and the email differs, **or** the password does not verify, or `is_active` is false | `UPDATE` email, `password_hash` (new hash, new salt) and `is_active = true` (R3, §6); then `DELETE FROM sessions WHERE user_id = $id` | **all ended** (SC-3; the owner's way to lock out someone who has the old password) |
| The new email is already used by a staff or customer row (`uq_users_email`, SQLSTATE 23505) | Roll back; the existing admin row is left exactly as it was. Logged (`ADMIN_EMAIL belongs to another account`). The server starts | unchanged |
| Two instances start together and both try to insert | The loser gets 23505 on `uq_users_one_admin`; it re-reads once and takes the "row exists" path | — |

The row is found with `SELECT id::text, email, password_hash, is_active FROM users WHERE role = 'admin' FOR UPDATE`, so a concurrent sign-in or a second instance waits for the decision. The role of a row is never changed by any statement: the admin row never becomes staff and a staff row never becomes admin.

## 6. Invariants every later writer of `users` and `sessions` keeps

These are stated here because the staff, accounts and manage stories are designed in parallel and must not each rediscover them. The BE review of this story checks R1–R3; the later stories' reviews check R4–R6.

| # | Rule |
|---|---|
| R1 | **`auth.Resolve` requires `u.is_active`** (Q1). A switched-off account never has a live session, even if a delete failed to run |
| R2 | **A session row is only ever inserted by Q3's shape** (or its twin for a freshly created account, which passes the hash it just wrote): `INSERT … SELECT … FROM users WHERE id AND is_active AND password_hash = $given … FOR SHARE OF u`. A sign-in racing a switch-off or a password reset therefore either sees the change and inserts nothing, or inserts first and has its session deleted by the writer's `DELETE` (READ COMMITTED: the writer's second statement sees the committed row) |
| R3 | **`EnsureAdmin` writes `is_active = true`** whenever it updates the row (§5) |
| R4 | A function that switches an account off, replaces its password, or deletes it ends its sessions **in the same transaction** as the `users` write (Q6). Deleting a user relies on `ON DELETE CASCADE`; switching off and a password reset do the explicit `DELETE` |
| R5 | Every email written to `users` passes `auth.NormalizeEmail` first; the unique index `uq_users_email` is the arbiter of a race, and its 23505 is translated by the caller (422 on staff creation and sign-up, never 500) |
| R6 | Hashing happens **before** the transaction opens (`auth.HashPassword`), never inside it: an argon2id hash takes tens of milliseconds and must not hold a row lock |

## 7. What is deliberately not stored

Raw session tokens (only the SHA-256, 32 bytes); the typed email of a failed sign-in (only `sha256('sign_in_email:' ‖ email)` in `rate_limits`); client addresses (only a hash); any role on the session (the role is read from `users` on every request, so a changed role is effective at once, SC-16).
