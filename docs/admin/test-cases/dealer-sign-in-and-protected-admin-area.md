# Test cases — Dealer sign-in and protected admin area

Module: `admin` · Plan item 8 · Requirements: ADMIN-001, ADMIN-002, ADMIN-003, ADMIN-004 (`docs/admin/SRS.md`)
Story: `docs/admin/stories/dealer-sign-in-and-protected-admin-area.md` — criteria SC-1 … SC-40
Contract: `docs/architecture/services.md` §2.2 (authentication), §2.3 (error contract), §2.5 (validation boundary), §2.7 (limits), §3.7 `POST /v1/auth/sign-in`, §3.8 `POST /v1/auth/sign-out`, §3.9 `GET /v1/auth/me`; `docs/architecture/erd.md` §3.18 `users`, §3.19 `sessions`, §3.21 `rate_limits`.

**Risk level: high.** This is the lock on the whole management area: a mistake here lets a stranger read customers' phones and addresses, or locks the owner out of their own shop. Every refusal (401, 403, 429), every boundary the story names (9 / 10 characters, 6 days 23 hours / exactly 7 days / 8 days, the fifth and sixth failed try, 200 / 201 characters) and every role — visitor, customer, staff, admin — gets its own case, allowed and denied. The pure copy cases are one read each.

## How these cases are observed

- **Addresses.** Requests are written the way the browser sends them, `GET /api/v1/…`; the edge proxy strips `/api` and the backend serves `/v1/…`. `B` is the running story's base address. A request that must carry a cookie sends `Cookie: session=<value>` taken from the `Set-Cookie` of a sign-in; a request that must not carry one sends none.
- **Error envelope.** "The error envelope" means: the status named, `error` equal to the code named, `message` a non-empty sentence, `errors` present only where the case says so (`bad_request`, `invalid`; never on 401, 403, 429, 500), and `Content-Type` starting `application/json`. No body contains SQL, `pgx`, a host name, a path, a stack trace, a hash or a token.
- **Instruments.** `fetch_url` for every response (status, body, `Set-Cookie`, `Retry-After`, `Cache-Control`); `set_db_state` to arrange or move stored rows and `set_backend_state` to stop or restart the backend (a stopped backend makes the gateway answer 502); `set_db_available` takes the database away while the backend runs. `render_url`, `interact_page` and `measure_styles` for the browser; `interact_page` reads the page's request log where a case says "no request". A case that arranges state puts it back before the next one (see "Reset" below).
- **Reset.** Sign-in failures from the verifier's own address add up across cases (5 in 15 minutes locks the address, SC-10). So every case that sends a failed sign-in begins and ends with `DELETE FROM rate_limits;`. Fixture accounts are removed at the end of the run with `DELETE FROM users WHERE role <> 'admin';` (sessions go with them, SC-16). Any `session` cookie a case obtained is discarded when the case ends.
- **Time.** The 15-minute lock and the 7-day expiry are checked by moving stored timestamps with `set_db_state`, never by waiting.
- **Selectors the browser cases use** (no class name is relied on): `form`, `input[type='email']`, `input[type='password']`, `form button[type='submit']`, `h1`, `[role='alert']`, `nav[aria-label='Sections']` (the left menu of the area), `button[aria-label='Sign out']` (the sign-out button), `a[href='/admin/sign-in']` (the footer's "Dealer sign-in"), `a[href^='/admin']` (every link into the area). The sign-in page's data roots carry `[data-dashboard='true']`. Management palette values (`src/theme.css`, `[data-dashboard]`): ground `rgb(248, 250, 252)`, ink `rgb(15, 23, 42)`, ink-soft `rgb(100, 116, 139)`, accent `rgb(37, 99, 235)`, danger `rgb(220, 38, 38)`.
- **Sample data.** The orders list, shoes list and overview show the first design's sample records until the data story's database is seeded; any order id or shoe id below (`doi-24817`, `samba-og-w-core-black`) is one of them.

### Fixtures

- **Admin.** The account made from the settings the story runs with, called `A_EMAIL` / `A_PASS` below. Cases that name a literal address or password (`dealer@example.com`, `lan.tran@example.com`, a 9- or 10-character password) say "start the backend with …": the verifier stops the backend, sets those settings and starts it again (`set_backend_state`), and at the end restarts it with the settings it ran with.
- **Staff and customer rows.** The story adds no way to create them in a running shop (story §8), so the verifier inserts them. The password of every fixture account is `correct horse battery staple` (28 characters; staff need ≥ 10, customers 8–100). The hash is the argon2id string below — the encoded hash printed in the argon2-cffi README for exactly that password (argon2id, v=19, m=65536, t=3, p=4) — and `$` inside a quoted SQL string is literal:

```sql
INSERT INTO users (email, name, role, is_active, password_hash) VALUES
 ('minh.le@example.com',          'Minh Le',    'staff',    true,  '$argon2id$v=19$m=65536,t=3,p=4$MIIRqgvgQbgj220jfp0MPA$YfwJSVjtjSU0zzV/P3S9nnQ/USre2wvJMjfCIjrTQbg'),
 ('lan.nguyen@example.com',       'Lan Nguyen', 'customer', true,  '$argon2id$v=19$m=65536,t=3,p=4$MIIRqgvgQbgj220jfp0MPA$YfwJSVjtjSU0zzV/P3S9nnQ/USre2wvJMjfCIjrTQbg'),
 ('off.staff@example.com',        'Off Staff',  'staff',    false, '$argon2id$v=19$m=65536,t=3,p=4$MIIRqgvgQbgj220jfp0MPA$YfwJSVjtjSU0zzV/P3S9nnQ/USre2wvJMjfCIjrTQbg'),
 ('mai_linh-pham@example.com',    '',           'staff',    true,  '$argon2id$v=19$m=65536,t=3,p=4$MIIRqgvgQbgj220jfp0MPA$YfwJSVjtjSU0zzV/P3S9nnQ/USre2wvJMjfCIjrTQbg');
```

  Call them `STAFF` (Minh Le), `CUSTOMER` (Lan Nguyen), `OFF` (switched off) and `NAMELESS` (no name). `PW` is the fixture password. **Fixture self-check:** the first sign-in of `STAFF` with `PW` (TC-12) proves the fixture. If it is refused while the admin signs in with `A_PASS`, the hash string, not the story, is at fault: the verifier makes a fixture hash another way (restart with a known `ADMIN_PASSWORD`, then `UPDATE users SET password_hash = (SELECT password_hash FROM users WHERE role = 'admin') WHERE role <> 'admin';`, so the fixtures share the admin's password) and says so in the report; cases that depend on `PW` are then run with that password.
- **A wrong password** is `not-the-password` (16 characters). **A forged cookie** is `session=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA`.
- **Slow answer.** Where a case needs the shop to take about 3 seconds, the request is held with the interaction instrument's request delay; where it has none, a failed sign-in is slowed by the database: `CREATE FUNCTION doi_test_sleep() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(1.5); RETURN NEW; END $$; CREATE TRIGGER doi_test_slow BEFORE INSERT OR UPDATE ON rate_limits FOR EACH ROW EXECUTE FUNCTION doi_test_sleep();` (each failed try writes two counters, about 3 seconds in all), removed afterwards with `DROP TRIGGER doi_test_slow ON rate_limits; DROP FUNCTION doi_test_sleep();`.

## Ambiguities found in the documents (reported, not silently resolved)

1. **SC-20 says no response body contains "the email typed", but `POST /v1/auth/sign-in` 200 and `GET /v1/auth/me` return the account's `email` by contract (services §3.7, §3.9).** TC-44 reads the sentence as: no *error* answer (400, 401, 413, 422, 429, 500) echoes the typed email or password, and no answer of any kind carries a password, a hash or a token. PM/TL to confirm.
2. **A sign-in with an over-long email (> 200) or password (> 100) is a 422 (services §3.7 constraints), but neither the story nor the contract says whether it counts as a failed try.** TC-18 asserts the status and the body only and resets the counters afterwards.
3. **SC-3's second sentence (a cookie issued before a changed `ADMIN_PASSWORD` stops opening `/v1/auth/me`) is the story's own addition to the SRS** (story §8, last bullet) and is not in `services.md` §3.9. TC-04 follows the story.
4. **SC-10 depends on how the verifier reaches the backend.** "Right-most `X-Forwarded-For` entry" can be exercised only when the case's header is the last entry the backend sees. If the gateway in front of the backend appends its own entry, every request shares that entry: the forged-left-most half (TC-25) still behaves as specified, the different-right-most half (TC-26) cannot be observed and is reported UNVERIFIED, not failed.
5. **SC-12 `Secure` "absent over plain HTTP" depends on the scheme the verifier's address uses.** If the gateway answers on HTTPS and sets `X-Forwarded-Proto` itself, the "absent" half of TC-29 cannot be seen and is reported UNVERIFIED.
6. **SC-40 and ADMIN-001 AC-13 speak of "the only link to `/admin`", but the footer's "Dealer sign-in" points at `/admin/sign-in`** (`content.json`). The cases read it as: the only link whose address starts with `/admin` is that one.
7. **SC-7 "a similar time" is a timing claim the contract words "to within noise".** No case asserts a duration; the dummy-hash code path is the unit-test case TC-14.

---

## A. Backend — the admin account and sign-in (SC-1 … SC-8)

**Scenario**: TC-01 — the admin is made from the settings and signs in
**Given**: The database is empty (no `users` row), and the backend is started with `ADMIN_EMAIL=dealer@example.com` and `ADMIN_PASSWORD=Dealer-Pass1` (12 characters).
**When**: `POST /api/v1/auth/sign-in` with `Content-Type: application/json` and body `{"email":"dealer@example.com","password":"Dealer-Pass1"}`; then `GET /api/v1/auth/me` with the cookie from the first answer.
**Then**: The first answer is 200 and its JSON is exactly the three keys `{"email":"dealer@example.com","name":"Dealer","role":"admin"}` (no `id`, no `password`, no `hash`, no `token`); it has a `Set-Cookie` for `session`. The second answer is 200 with the same three values and `role` `admin`. `GET /healthz` is 200.
Traces: SC-1 (ADMIN-003 AC-1)
Check: fetch_url

**Scenario**: TC-02 — the email is matched without regard to case and surrounding spaces
**Given**: The account of TC-01.
**When**: `POST /api/v1/auth/sign-in` with `{"email":" Dealer@Example.com ","password":"Dealer-Pass1"}`; then `GET /api/v1/auth/me` with its cookie.
**Then**: The first answer is 200 with `email` `dealer@example.com`, `name` `Dealer`, `role` `admin`; the second is 200 with the same. A `SELECT count(*) FROM users` (where the verifier's SQL returns rows) is still 1: no second account was made by the differently written address.
Traces: SC-1 (ADMIN-001 boundary)
Check: fetch_url

**Scenario**: TC-03 — the stored password is an argon2id hash, salted, never the typed password
**Given**: The admin of TC-01 exists. (The fixture rows share one hash on purpose, so the salt claim is read on two rows the server's own code made, not on the fixtures.)
**When**: `SELECT email, password_hash FROM users WHERE role = 'admin'` is read; then a second account is made by the server's own code with the same password as the admin's (the backend PR's Go test calls `auth.CreateUser` twice with one password) and both hashes are read.
**Then**: The admin's `password_hash` starts with `$argon2id$`; it is not equal to `Dealer-Pass1` and does not contain it. The two hashes made from one password differ (a salt per password). The hash never appears in any response of TC-01 and TC-02.
Traces: SC-2 (ADMIN-003 AC-2)
Check: manual — the instrument needed is a `SELECT` whose rows reach the verifier (`set_db_state` if it returns rows) and a way to create two accounts with one password (the server has no route for it; the backend PR's Go test with `TEST_DATABASE_URL`, read with `run_tests`). Nothing served over HTTP shows a stored hash, by design.

**Scenario**: TC-04 — a changed `ADMIN_PASSWORD` refuses the old password and ends old sessions
**Given**: The backend runs with `dealer@example.com` / `Dealer-Pass1`; the verifier signed in and holds cookie `C1`; `GET /api/v1/auth/me` with `C1` is 200. The backend is then restarted with `ADMIN_PASSWORD=Dealer-Pass2-New` (the email unchanged).
**When**: `POST /api/v1/auth/sign-in` with the old password; `POST /api/v1/auth/sign-in` with `Dealer-Pass2-New`; `GET /api/v1/auth/me` with `C1`. `DELETE FROM rate_limits` runs before and after.
**Then**: The old password answers 401 `invalid_credentials` with message `Wrong email or password.`; the new one answers 200 `role` `admin`; `C1` answers 401 `unauthenticated`.
Traces: SC-3 (ADMIN-003 AC-3, story item 3)
Check: fetch_url

**Scenario**: TC-05 — a changed `ADMIN_EMAIL` changes the same row: still exactly one admin
**Given**: The backend runs with `dealer@example.com` / `Dealer-Pass1`. It is restarted with `ADMIN_EMAIL=owner@example.com` and the same password.
**When**: `POST /api/v1/auth/sign-in` for `owner@example.com` with the password; for `dealer@example.com` with the password; then `INSERT INTO users (email, name, role, password_hash) VALUES ('second.admin@example.com', '', 'admin', 'x')` is attempted.
**Then**: The new address answers 200 `role` `admin`; the old address answers 401 `invalid_credentials`. The `INSERT` is refused by the database (unique violation, `uq_users_one_admin`: there is exactly one admin row); `SELECT count(*) FROM users WHERE role = 'admin'` is 1 where the verifier's SQL returns rows.
Traces: SC-3 (ADMIN-003 item 1)
Check: fetch_url

**Scenario**: TC-06 — a restart with unchanged settings changes nothing and keeps sessions
**Given**: The backend runs with `dealer@example.com` / `Dealer-Pass1`; the verifier holds cookie `C1` from a sign-in. The backend is restarted with the same two settings.
**When**: `GET /api/v1/auth/me` with `C1`; `POST /api/v1/auth/sign-in` with the same credentials.
**Then**: `/me` with `C1` is 200 `role` `admin` (the existing session survived the restart); the sign-in is 200. `SELECT count(*) FROM sessions` is 2 where the verifier's SQL returns rows (no session was ended or created by the restart).
Traces: SC-3 (ADMIN-003 AC-3 converse; story SC-3 last sentence)
Check: fetch_url

**Scenario**: TC-07 — a 9-character password: the server starts, makes and changes no account, sign-in is refused
**Given**: Empty database. The backend is started with `ADMIN_EMAIL=dealer@example.com` and `ADMIN_PASSWORD=Short-Pw9` (9 characters). A second run starts from TC-01's account (password `Dealer-Pass1`) and restarts with `ADMIN_PASSWORD=Short-Pw9`.
**When**: `GET /healthz`; `POST /api/v1/auth/sign-in` with `dealer@example.com` / `Short-Pw9`. In the second run: `POST /api/v1/auth/sign-in` with `dealer@example.com` / `Dealer-Pass1`. `DELETE FROM rate_limits` between calls.
**Then**: `/healthz` is 200 (the server started). In the empty-database run the sign-in answers 401 `invalid_credentials` and `SELECT count(*) FROM users` is 0. In the second run the old password `Dealer-Pass1` still signs in (200): the too-short setting changed nothing.
Traces: SC-4 (ADMIN-003 AC-4, boundary)
Check: fetch_url

**Scenario**: TC-08 — the start-up log says the password is too short and never prints it
**Given**: The backend is started as in TC-07 (9-character password).
**When**: The backend's start-up log is read.
**Then**: One line says the admin password is too short (or under 10 characters) and the admin account was skipped; no line contains `Short-Pw9`, any part of it, or a hash.
Traces: SC-4 (ADMIN-003 AC-4; SC-20)
Check: manual — the instrument needed is a read of the backend's log output; no HTTP route returns it.

**Scenario**: TC-09 — exactly 10 characters is accepted
**Given**: Empty database. The backend is started with `ADMIN_EMAIL=dealer@example.com` and `ADMIN_PASSWORD=Ten-Chars1` (10 characters).
**When**: `POST /api/v1/auth/sign-in` with `dealer@example.com` / `Ten-Chars1`.
**Then**: 200 `{"email":"dealer@example.com","name":"Dealer","role":"admin"}`.
Traces: SC-4 (ADMIN-003 boundary)
Check: fetch_url

**Scenario**: TC-10 — settings unset or empty: the server starts, no account exists, sign-in is refused for everyone
**Given**: Empty database. Three starts, each on a database with no `users` row: (a) neither `ADMIN_EMAIL` nor `ADMIN_PASSWORD` set; (b) `ADMIN_EMAIL=dealer@example.com` with `ADMIN_PASSWORD=` empty; (c) `ADMIN_EMAIL=` empty with `ADMIN_PASSWORD=Dealer-Pass1`.
**When**: In each start: `GET /healthz`; `POST /api/v1/auth/sign-in` with `dealer@example.com` / `Dealer-Pass1`; the same with `anyone@example.com` / `Dealer-Pass1`. `DELETE FROM rate_limits` between starts.
**Then**: In each start `/healthz` is 200 and `SELECT count(*) FROM users` is 0 (where the verifier's SQL returns rows). Both sign-ins answer 401 with the byte-identical body `{"error":"invalid_credentials","message":"Wrong email or password."}` and no `Set-Cookie`.
Traces: SC-5 (ADMIN-003 AC-5), SC-7 (the "settings unset" refusal)
Check: fetch_url

**Scenario**: TC-11 — the example settings file names both keys with no value
**Given**: `code/backend/.env.example` in the pull request.
**When**: Its lines for `ADMIN_EMAIL` and `ADMIN_PASSWORD` are read.
**Then**: Each key appears once, as `ADMIN_EMAIL=` and `ADMIN_PASSWORD=` with nothing after the `=`; the comment above says the password is 10 characters or more; no address or password value appears anywhere in the file for these two keys.
Traces: SC-5 (ADMIN-003 AC-10; story item 11)
Check: manual — a repository file is not served by the running story; read it from the pull request (`get_pr`).

**Scenario**: TC-12 — a staff row and a customer row sign in with their own passwords; the name is the row's own
**Given**: The four fixture rows are inserted (see "Fixtures"). The backend runs with the admin of the run.
**When**: `POST /api/v1/auth/sign-in` for `minh.le@example.com` / `PW`, then `GET /api/v1/auth/me` with its cookie; the same for `lan.nguyen@example.com` / `PW`. `PW` is also sent to `minh.le@example.com` with one character changed.
**Then**: The staff answers are 200 `{"email":"minh.le@example.com","name":"Minh Le","role":"staff"}` (both calls); the customer answers are 200 `{"email":"lan.nguyen@example.com","name":"Lan Nguyen","role":"customer"}`; the altered password answers 401 `invalid_credentials`.
Traces: SC-6 (ADMIN-003 AC-11, AC-12)
Check: fetch_url

**Scenario**: TC-13 — unknown email, wrong password, switched-off account: one status, one body, no cookie
**Given**: The admin and the fixture rows exist; `DELETE FROM rate_limits` runs before and after.
**When**: `POST /api/v1/auth/sign-in` with (a) `nobody@example.com` / `PW`; (b) the admin's email with `not-the-password`; (c) `off.staff@example.com` with its right password `PW` (the account is switched off); (d) `minh.le@example.com` with `not-the-password`.
**Then**: All four answer 401 with the byte-identical body `{"error":"invalid_credentials","message":"Wrong email or password."}` (no `errors` key; nothing that says which part was wrong or that an account exists or is off) and none sets a cookie. A `Content-Type` starting `application/json` on all four.
Traces: SC-7 (ADMIN-003 AC-8, AC-13; ADMIN-001 AC-7)
Check: fetch_url

**Scenario**: TC-14 — an argon2id verification runs for every attempt, against a dummy hash when there is no usable account
**Given**: The backend PR's unit test of `auth.SignIn` (or its helper) with a stubbed store.
**When**: The test calls it for an unknown email, a switched-off account with the right password, a wrong password, and (settings unset) any email, counting calls to the argon2id compare.
**Then**: The compare ran exactly once per call in all four, the unknown-email and switched-off calls compared against a fixed dummy hash (not an early return), and all four returned the same error value.
Traces: SC-7 (services §3.7 Notes)
Check: manual — a property of a code path: no HTTP response shows that a hash was computed and a timing assertion would be noise (ambiguity 7). The instrument is `run_tests` on the backend PR plus reading the test.

**Scenario**: TC-15 — a blank email or password is 422 and is not a failed try
**Given**: The admin exists; `DELETE FROM rate_limits` runs before and after.
**When**: `POST /api/v1/auth/sign-in` with (a) `{"email":"","password":"x"}`; (b) `{"email":"   ","password":"x"}`; (c) `{"email":"<A_EMAIL>","password":""}`; (d) `{"email":"","password":""}`; then six more requests like (a); then the admin's right credentials.
**Then**: (a) and (b) answer 422 `invalid` with `errors.email` exactly `An email address is needed` and no `errors.password`; (c) answers 422 with `errors.password` (a non-empty sentence) and no `errors.email`; (d) answers 422 with both keys. Each has `message` a non-empty sentence. After the six extra blank-email requests the right credentials answer 200, not 429; `SELECT count(*) FROM rate_limits` is 0 where the verifier's SQL returns rows.
Traces: SC-8 (services §3.7 `invalid`)
Check: fetch_url

**Scenario**: TC-16 — a request that is not JSON-typed is 415
**Given**: The admin exists.
**When**: `POST /api/v1/auth/sign-in` with `Content-Type: text/plain` and the body `{"email":"a@b.co","password":"x"}`; again with `Content-Type: application/x-www-form-urlencoded` and body `email=a%40b.co&password=x`.
**Then**: Both answer 415 with the error envelope, `error` `unsupported_media_type`, no `errors` key, no `Set-Cookie`.
Traces: SC-8 (services §2.1, §3.7 `unsupported_media_type`)
Check: fetch_url

**Scenario**: TC-17 — invalid JSON, an undefined field and a wrong type are 400
**Given**: The admin exists; `Content-Type: application/json` on every request.
**When**: `POST /api/v1/auth/sign-in` with (a) body `{"email":` (truncated); (b) `{"email":"a@b.co","password":"x","remember":true}`; (c) `{"email":"a@b.co","password":"x","role":"admin"}`; (d) `{"email":123,"password":"x"}`; (e) `{"email":"a@b.co","password":["x"]}`; (f) `[]`.
**Then**: All six answer 400 with the error envelope, `error` `bad_request`, and an `errors` object; for (b) `errors.remember` and for (c) `errors.role` is a sentence (the field is refused, never ignored: no 200 and no 401); for (d) `errors.email`, for (e) `errors.password`. None sets a cookie.
Traces: SC-8 (services §2.5, §3.7 `bad_request`)
Check: fetch_url

**Scenario**: TC-18 — the length limits: 200 / 201 characters of email, 100 / 101 of password
**Given**: The admin exists; `DELETE FROM rate_limits` runs before and after (ambiguity 2).
**When**: `POST /api/v1/auth/sign-in` with (a) an email of exactly 200 characters (`a` × 188 + `@example.com`) and password `x`; (b) an email of 201 characters (`a` × 189 + `@example.com`); (c) the admin's email and a password of exactly 100 characters; (d) the admin's email and a password of 101 characters.
**Then**: (a) and (c) answer 401 `invalid_credentials` (well-formed, no such account / wrong password). (b) answers 422 `invalid` with `errors.email` a sentence; (d) answers 422 `invalid` with `errors.password` a sentence.
Traces: contract (POST /v1/auth/sign-in constraints)
Check: fetch_url

**Scenario**: TC-19 — a body over the 64 KiB cap is 413
**Given**: The admin exists.
**When**: `POST /api/v1/auth/sign-in` with `Content-Type: application/json` and a body of `{"email":"a@b.co","password":"` + 70,000 `x` + `"}` (over 65,536 bytes).
**Then**: 413 with the error envelope and `error` `payload_too_large`; no `errors` key; no `Set-Cookie`.
Traces: contract (services §2.3 `payload_too_large`, §2.5 body cap)
Check: fetch_url

---

## B. Backend — limits, cookie, `me`, sign-out, expiry (SC-9 … SC-16)

**Scenario**: TC-20 — five wrong passwords lock the email, even against the right password
**Given**: The admin exists; `DELETE FROM rate_limits` has run.
**When**: `POST /api/v1/auth/sign-in` five times with the admin's email and `not-the-password`; then a sixth time with the admin's email and `A_PASS`.
**Then**: Each of the five answers 401 `invalid_credentials` (the fifth is not yet 429). The sixth answers 429 with the error envelope, `error` `rate_limited`, a non-empty `message`, no `errors` key, no `Set-Cookie`, and a `Retry-After` header that is a whole number of seconds from 1 to 900.
Traces: SC-9 (ADMIN-003 AC-6)
Check: fetch_url

**Scenario**: TC-21 — a refused call neither counts nor extends the lock; a success neither counts nor clears
**Given**: The admin exists; `DELETE FROM rate_limits` has run. (a) The email of TC-20 is locked and `R1` is its `Retry-After`. (b) A second run from a clean table: four wrong passwords, then one success with `A_PASS`.
**When**: (a) The sixth try is repeated twice more, then `Retry-After` is read again. (b) After the success, one more wrong password is sent, then the right password.
**Then**: (a) Both repeats answer 429 and each `Retry-After` is less than or equal to `R1` (it counts down; a counted refusal would restart it at about 900). (b) The four wrong passwords answer 401 and the success 200 (the success did not clear the four); the one further wrong password answers 401 and the right password after it answers 429: the fifth failure locked the email.
Traces: SC-9 (services §2.7 "without counting"; "a success neither counts nor clears")
Check: fetch_url

**Scenario**: TC-22 — the lock ends 15 minutes after the fifth failure
**Given**: The admin's email is locked as in TC-20. Through `set_db_state`: `UPDATE rate_limits SET window_started_at = window_started_at - interval '14 minutes', locked_until = locked_until - interval '14 minutes' WHERE scope = 'sign_in_email'` — and the same for scope `sign_in_address` — so the fifth failure was 14 minutes ago.
**When**: The right password is sent; then the same two `UPDATE`s with `interval '2 minutes'` more (16 minutes in all) and the right password is sent again.
**Then**: At 14 minutes the answer is 429 `rate_limited` with `Retry-After` between 1 and 60. At 16 minutes the answer is 200 `role` `admin`.
Traces: SC-9 (ADMIN-003 AC-9, boundary)
Check: fetch_url

**Scenario**: TC-23 — a window older than 15 minutes starts again from one
**Given**: Clean `rate_limits`. Four wrong passwords are sent for the admin's email; then, through `set_db_state`, both counters' `window_started_at` are moved back 16 minutes.
**When**: Four more wrong passwords are sent; then the right password.
**Then**: The four new wrong passwords all answer 401 (not 429: the old four were forgotten) and the right password answers 200.
Traces: SC-9 (erd §3.21 rules: "a window older than 15 minutes with no lock starts again at 1")
Check: fetch_url

**Scenario**: TC-24 — five failures from one address across five emails lock the address
**Given**: Clean `rate_limits`. Every request carries `X-Forwarded-For: 203.0.113.9` (one address).
**When**: `POST /api/v1/auth/sign-in` for `a1@example.com`, `a2@example.com`, `a3@example.com`, `a4@example.com`, `a5@example.com`, each with `x`; then the admin's email with `A_PASS`; then `a6@example.com` with `x`.
**Then**: The five answer 401 `invalid_credentials`. The admin's right credentials answer 429 `rate_limited` with `Retry-After` 1–900, and so does `a6@example.com`: the address is locked for any email.
Traces: SC-10 (ADMIN-003 AC-7)
Check: fetch_url

**Scenario**: TC-25 — forged left-most `X-Forwarded-For` entries share one lock
**Given**: Clean `rate_limits`.
**When**: Five failed sign-ins (`x@example.com` / `x`) with the headers `X-Forwarded-For: 1.1.1.1, 203.0.113.9`, `2.2.2.2, 203.0.113.9`, `3.3.3.3, 203.0.113.9`, `4.4.4.4, 203.0.113.9`, `5.5.5.5, 203.0.113.9`; then the admin's right credentials with `X-Forwarded-For: 6.6.6.6, 203.0.113.9`.
**Then**: The five answer 401; the sixth answers 429 `rate_limited`: changing only the left-most entry did not make a new address.
Traces: SC-10 (services §2.2 "Client address")
Check: fetch_url

**Scenario**: TC-26 — a different right-most entry is a different address
**Given**: Clean `rate_limits`; the address `203.0.113.9` is locked as in TC-24 (the header is the last `X-Forwarded-For` entry the backend sees — ambiguity 4).
**When**: The admin's right credentials are sent with `X-Forwarded-For: 203.0.113.10`; then with `X-Forwarded-For: 203.0.113.9`.
**Then**: The first answers 200; the second answers 429.
Traces: SC-10 (services §2.2 "Client address")
Check: fetch_url

**Scenario**: TC-27 — two sign-ins at the same moment both succeed with separate sessions
**Given**: The admin exists; `DELETE FROM sessions` has run.
**When**: Two `POST /api/v1/auth/sign-in` requests with the admin's right credentials are sent together (two calls in one batch); then `GET /api/v1/auth/me` with each cookie; then `POST /api/v1/auth/sign-out` with the first cookie; then `GET /api/v1/auth/me` with each cookie again.
**Then**: Both sign-ins answer 200 with a `Set-Cookie` each, and the two cookie values differ. Both `/me` calls answer 200. After sign-out of the first, the first cookie answers 401 and the second still answers 200. `SELECT count(*) FROM sessions` was 2 before the sign-out where the verifier's SQL returns rows.
Traces: SC-11 (ADMIN-003 Conflict)
Check: fetch_url

**Scenario**: TC-28 — the session cookie's attributes
**Given**: The admin exists.
**When**: `POST /api/v1/auth/sign-in` with the right credentials; the response headers are read; a second sign-in is made.
**Then**: Exactly one `Set-Cookie` header is present and it sets `session`; its attributes include `HttpOnly`, `SameSite=Lax`, `Path=/` and `Max-Age=604800` (compared case-insensitively). A refused sign-in (wrong password) sets no cookie. The two cookie values differ.
Traces: SC-12 (ADMIN-002 AC-1)
Check: fetch_url

**Scenario**: TC-29 — `Secure` follows `X-Forwarded-Proto`; the token is long and appears in no body
**Given**: The admin exists.
**When**: `POST /api/v1/auth/sign-in` (a) with `X-Forwarded-Proto: https`; (b) with no such header, over the verifier's plain address (ambiguity 5); the cookie values and the bodies of (a), (b) and of `GET /api/v1/auth/me` are read.
**Then**: (a)'s `Set-Cookie` includes `Secure`; (b)'s does not. Each cookie value is at least 22 characters from the URL-safe alphabet (128 bits or more) and is not a counter or the email. No response body contains the cookie value; each sign-in body has exactly the keys `email`, `name`, `role`.
Traces: SC-12 (services §2.2 Cookie)
Check: fetch_url

**Scenario**: TC-30 — `GET /v1/auth/me`: the answer, `no-store`, and 401 without a session
**Given**: The admin signed in (cookie `C`).
**When**: `GET /api/v1/auth/me` with `C`; `GET /api/v1/auth/me` with no cookie; `GET /api/v1/auth/me` with the forged cookie.
**Then**: With `C`: 200, `Cache-Control` contains `no-store`, body exactly the keys `email`, `name`, `role` with `role` `admin`. With no cookie and with the forged cookie: 401 with the error envelope, `error` `unauthenticated`, no `errors` key, and no user data in the body.
Traces: SC-13 (ADMIN-002 AC-2, AC-3)
Check: fetch_url

**Scenario**: TC-31 — the name: the row's own, else the email's part before `@` capitalised
**Given**: The fixture rows exist and the backend is started with `ADMIN_EMAIL=lan.tran@example.com` and a password of 10+ characters.
**When**: Each of these signs in and `GET /api/v1/auth/me` is read: the admin `lan.tran@example.com`; `NAMELESS` (`mai_linh-pham@example.com`, name empty); `STAFF` (`minh.le@example.com`, name `Minh Le`); then `UPDATE users SET name = 'Chị Hoa' WHERE email = 'minh.le@example.com'` and `STAFF` again.
**Then**: The `name` is `Lan Tran` for the admin; `Mai Linh Pham` for `NAMELESS` (split on `_` and `-`); `Minh Le` for `STAFF`; then `Chị Hoa` (the account's own name wins and is returned unchanged, diacritics intact). The name is put back to `Minh Le` afterwards.
Traces: SC-13 (ADMIN-002 AC-4, item 2)
Check: fetch_url

**Scenario**: TC-32 — sign-out ends the session at the server
**Given**: The admin signed in (cookie `C`); `GET /api/v1/auth/me` with `C` is 200.
**When**: `POST /api/v1/auth/sign-out` with `Cookie: session=C` and no body; then `GET /api/v1/auth/me` with `C`.
**Then**: Sign-out answers 204 with an empty body and a `Set-Cookie` that clears `session` (`Max-Age=0`). `/me` with `C` then answers 401 `unauthenticated`. `SELECT count(*) FROM sessions` for that user is 0 where the verifier's SQL returns rows.
Traces: SC-14 (ADMIN-002 AC-10, AC-11)
Check: fetch_url

**Scenario**: TC-33 — sign-out with no cookie or an unknown one is 204; a body is 400
**Given**: Nobody signed in for the cases (a)–(c); the admin signed in for (d).
**When**: `POST /api/v1/auth/sign-out` (a) with no cookie and no body; (b) with the forged cookie and no body; (c) with no cookie and the body `{}` and `Content-Type: application/json`; (d) with a valid cookie and the body `{"all":true}`.
**Then**: (a) and (b) answer 204 (nothing to end). (c) and (d) answer 400 `bad_request` with the error envelope; after (d) the valid cookie still answers 200 on `/me` (a refused request ended nothing).
Traces: SC-14 (ADMIN-002 AC-14; services §3.8)
Check: fetch_url

**Scenario**: TC-34 — expiry at 6 days 23 hours, exactly 7 days and 8 days; no sliding
**Given**: `STAFF` signed in (cookie `C`). Three states are set in turn with `set_db_state` on that user's session row: (a) `created_at = now() - interval '6 days 23 hours', expires_at = now() + interval '1 hour'`; (b) `created_at = now() - interval '7 days', expires_at = now()`; (c) `created_at = now() - interval '8 days', expires_at = now() - interval '1 day'`.
**When**: `GET /api/v1/auth/me` with `C` after each state; in (a) the response headers are read and the call is repeated five times.
**Then**: (a) 200 `role` `staff`, and no `Set-Cookie` on any of the six answers (the session is neither refreshed nor extended); `expires_at` still reads `now() + 1 hour` (as set) where the verifier's SQL returns rows. (b) 401 `unauthenticated` (exactly 7 days is expired). (c) 401 `unauthenticated`. The 401 bodies carry no user data.
Traces: SC-15 (ADMIN-002 AC-13, boundary)
Check: fetch_url

**Scenario**: TC-35 — the sessions table holds only the SHA-256 of the token
**Given**: `STAFF` signed in with cookie value `T`.
**When**: `SELECT encode(token_hash, 'hex'), octet_length(token_hash) FROM sessions` is read, and `SELECT count(*) FROM sessions WHERE token_hash = sha256(convert_to('<T>', 'UTF8'))` is run; every column of every `sessions` row is searched for `T`.
**Then**: `octet_length` is 32; the count is 1; `T` appears in no column (the table's columns are `id`, `user_id`, `token_hash`, `created_at`, `expires_at`).
Traces: SC-15 (services §2.2 Mechanism; erd §3.19)
Check: manual — the instrument needed is a `SELECT` whose result reaches the verifier (`set_db_state` where it returns rows); otherwise the backend PR's Go test. No HTTP route returns a stored token hash.

**Scenario**: TC-36 — a switched-off staff account is refused on its next request with the same cookie
**Given**: `STAFF` signed in (cookie `C`); `GET /api/v1/auth/me` with `C` is 200 and `GET /api/v1/orders` with `C` is 200.
**When**: `UPDATE users SET is_active = false WHERE email = 'minh.le@example.com'`; then `GET /api/v1/auth/me` with `C`, `GET /api/v1/orders` with `C`, and a new `POST /api/v1/auth/sign-in` with `PW`. The row is switched back on afterwards.
**Then**: `/me` and the orders list answer 401 `unauthenticated` at once (no wait, no restart); the new sign-in answers 401 `invalid_credentials`.
Traces: SC-16 (ADMIN-003 item 7; ADMIN-002 item 10)
Check: fetch_url

**Scenario**: TC-37 — deleting the user ends its sessions
**Given**: `STAFF` signed in twice (cookies `C1`, `C2`); both answer 200 on `/me`.
**When**: `DELETE FROM users WHERE email = 'minh.le@example.com'`; then `GET /api/v1/auth/me` with `C1` and with `C2`; `SELECT count(*) FROM sessions` is read.
**Then**: Both answer 401 `unauthenticated`; the `sessions` count for that user is 0 where the verifier's SQL returns rows (the cascade). The fixture row is inserted again afterwards if later cases need it.
Traces: SC-16 (ADMIN-003 item 7; erd §3.19 cascade)
Check: fetch_url

**Scenario**: TC-38 — a changed role takes effect on the next request
**Given**: `STAFF` signed in (cookie `C`); `GET /api/v1/orders` with `C` is 200.
**When**: `UPDATE users SET role = 'customer' WHERE email = 'minh.le@example.com'`; then `GET /api/v1/auth/me` and `GET /api/v1/orders` with `C`; then the role is set back to `staff` and both are read again.
**Then**: With role `customer`: `/me` is 200 with `role` `customer` and the orders list is 403 `forbidden`. With role `staff` again: `/me` says `staff` and the orders list is 200. No new sign-in was needed at any step.
Traces: SC-16 (ADMIN-002 item 10)
Check: fetch_url

---

## C. Backend — the guard (SC-17 … SC-20) and the contract's failure paths

**Scenario**: TC-39 — `GET /v1/orders` for each kind of caller
**Given**: Sessions: `CA` (admin), `CS` (`STAFF`), `CC` (`CUSTOMER`); the forged cookie; a cookie with an empty value (`session=`).
**When**: `GET /api/v1/orders` with no cookie; with the forged cookie; with `session=`; with `CC`; with `CS`; with `CA`.
**Then**: No cookie, forged and empty: 401 `unauthenticated` (error envelope, no `errors`, no order data in the body). `CC`: 403 `forbidden` (error envelope, no `errors`, no order data). `CS` and `CA`: 200 with the list shape `{"items":[…],"total":<number>,"page":1,"limit":12}`; the two dealer bodies have the same keys, and `total` is the same for both.
Traces: SC-17 (ADMIN-002 AC-9, AC-18; ADMIN-004 item 1, AC-8, AC-10; DATA-004 item 4)
Check: fetch_url

**Scenario**: TC-40 — public reads stay open with no session, a forged one and a customer's
**Given**: The cookies of TC-39.
**When**: `GET /api/v1/products` with no cookie, with the forged cookie, and with `CC`; `GET /api/v1/auth/me` with the forged cookie.
**Then**: The three product reads answer 200 with `items`, `total`, `page`, `limit` (a bad cookie is "no user", not an error). `/me` with the forged cookie answers 401.
Traces: SC-17 (ADMIN-004 AC-8, item 2)
Check: fetch_url

**Scenario**: TC-41 — `auth.Require` answers 401, 403 or 200 by role, on routes the test registers
**Given**: The backend PR's Go test registers two routes: one wrapped `auth.Require(admin, staff)` and one wrapped `auth.Require(admin)`.
**When**: Each route is called with no session, a customer, a staff and an admin session.
**Then**: `Require(admin, staff)`: none 401, customer 403, staff 200, admin 200. `Require(admin)`: none 401, staff 403, customer 403, admin 200. The 401 and 403 bodies use the error envelope (`unauthenticated`, `forbidden`) and carry no data.
Traces: SC-18 (ADMIN-004 AC-10, AC-11, AC-12)
Check: manual — no route of the running story is wrapped `auth.Require(admin)` alone yet (the staff story adds them), so the admin-only half cannot be called over HTTP; the staff-and-admin half is already observed on `GET /v1/orders` (TC-39). The instrument is `run_tests` on the backend PR plus reading the test.

**Scenario**: TC-42 — the migration's constraints refuse what they must
**Given**: The admin and the fixture rows exist.
**When**: Through `set_db_state`, each is attempted: (a) `INSERT INTO users (email, name, role, password_hash) VALUES ('second.admin@example.com', '', 'admin', 'x')`; (b) the same with `role` `'owner'`; (c) `email` `'Upper@Example.com'` with role `'customer'`; (d) a staff row whose `email` equals the admin's; (e) a customer row with `password_hash` `''`; (f) `INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ((SELECT id FROM users LIMIT 1), '\x00', now() + interval '1 day')`; (g) a `sessions` row with a 32-byte `token_hash` and `expires_at = now() - interval '1 day'` (expiry before creation).
**Then**: (a) is refused by `uq_users_one_admin`; (b) by `ck_users_role`; (c) by `ck_users_email_lowercase`; (d) by `uq_users_email`; (e) by the non-empty hash check; (f) by the 32-byte check; (g) by `ck_sessions_expiry_after_created`. After all seven, `POST /api/v1/auth/sign-in` for `second.admin@example.com` / `x` and for `upper@example.com` / `x` answer 401 (no such row was made), and the admin still signs in (200).
Traces: SC-19 (erd §3.18, §3.19)
Check: fetch_url

**Scenario**: TC-43 — a second boot changes nothing; the build, vet and tests pass with no database
**Given**: A database where the migration has applied once; the backend PR's head.
**When**: (a) The backend is stopped and started twice; `GET /healthz`, a sign-in and `GET /api/v1/auth/me` with a cookie held from before the restarts are read. (b) `go build ./...`, `go vet ./...` and `go test ./...` run with `TEST_DATABASE_URL` unset.
**Then**: (a) `/healthz` is 200 each time; the sign-in is 200; the old cookie still answers 200; `SELECT count(*) FROM schema_migrations` is the same before and after. (b) All three succeed and the database tests report skipped, not failed.
Traces: SC-19 (erd §8 migration 7)
Check: manual — part (a) is an HTTP observation, but (b) needs a Go toolchain run: the instrument is `run_tests` (backend) on the pull request's head, which has no database, exactly as the story's own CI note says. Part (a) is read with `fetch_url` and `set_backend_state` in the same run.

**Scenario**: TC-44 — error answers never echo the typed email or password; no answer carries a hash or token
**Given**: The admin and the fixtures exist; `DELETE FROM rate_limits` runs before and after. The typed values are the unique strings `probe-ZX81@example.com` and `Probe-Secret-ZX81`.
**When**: Sign-in is attempted with them as a 401 (unknown account), a 422 (blank password with that email), a 400 (an undefined field added), a 429 (the address locked by five earlier failures), and a 415 (`text/plain`); then the admin's and `STAFF`'s successful sign-in and `/me` bodies, a sign-out answer and a 403 and a 401 on `GET /api/v1/orders` are read.
**Then**: None of the 400, 401, 415, 422, 429 bodies or headers contains `probe-ZX81`, `Probe-Secret-ZX81` or the email in any case (the field *names* `errors.email` / `errors.password` of a 422 are the contract's and are not values). No body or header of any answer listed — error or success — contains `$argon2`, a cookie value or token, or a field named `id`; the success bodies of sign-in and `/me` contain no key `password`, `hash` or `token`.
Traces: SC-20 (Security NFR; ambiguity 1)
Check: fetch_url

**Scenario**: TC-45 — the logs carry no secret; a request log line has method, route pattern, status and role only
**Given**: The sequence of TC-44, plus a sign-in as `STAFF`, a sign-out, and a start-up with the admin settings.
**When**: The backend's log output for those requests is read.
**Then**: No line contains the passwords (`PW`, `A_PASS`, `Probe-Secret-ZX81`, `Dealer-Pass1`), a hash (`$argon2`), a cookie value or token, or any email typed (`probe-ZX81@example.com`, `minh.le@example.com`, `A_EMAIL`). A line for a request names the method, the route pattern (`/v1/auth/sign-in`, not a path with values), the status and the role (`admin`, `staff`, `customer` or none), and nothing else about the person.
Traces: SC-20 (services §7)
Check: manual — the instrument needed is a read of the backend's log output; no HTTP route returns it. Where the verifier cannot read it, `run_tests` on the backend PR's log-capturing test stands in.

**Scenario**: TC-46 — the database is unreachable: sign-in, `me` and the orders list answer 500 `internal`
**Given**: `STAFF` signed in (cookie `C`) and the admin's password known; then `set_db_available` takes the database away while the backend keeps running.
**When**: `POST /api/v1/auth/sign-in` with the admin's right credentials; `GET /api/v1/auth/me` with `C`; `GET /api/v1/orders` with `C`; `POST /api/v1/auth/sign-out` with `C`. The database is given back afterwards.
**Then**: Each answers 500 (not 401, not 503, not 502, not 200) with `error` `internal`, a generic `message`, no `errors` key, no `Set-Cookie` that opens a session, and a body with none of `SELECT`, `FROM`, `pgx`, `postgres`, `dial`, `connection refused`, a host name or port. After the database is given back, `/me` with `C` is 200 again.
Traces: contract (services §2.3 `internal`; §3.7, §3.8, §3.9 `internal`)
Check: fetch_url

**Scenario**: TC-47 — a failed query is 500 `internal`, not a silent success
**Given**: `STAFF` signed in (cookie `C`). Through `set_db_state` the table `sessions` is renamed to `sessions_gone` (the backend stays up and the database is reachable).
**When**: `POST /api/v1/auth/sign-in` with the admin's right credentials; `GET /api/v1/auth/me` with `C`; `POST /api/v1/auth/sign-out` with `C`. The table is renamed back afterwards.
**Then**: Each answers 500 with the error envelope, `error` `internal`, no `errors` key; the sign-in sets no usable `session` cookie (a session could not be made); the body contains no table name (`sessions_gone`), no SQL and no stack trace. After the rename back, `/me` with `C` is 200.
Traces: contract (services §2.3 `internal`; §3.8 "internal 500 only if the delete itself fails")
Check: fetch_url

---

## D. The sign-in page `/admin/sign-in` (SC-21 … SC-30)

The browser cases use a fresh browser context with no cookie and an empty local storage unless a case says otherwise. The default window is 1280×800.

**Scenario**: TC-48 — the screen's words, in their order
**Given**: No session; a 1280×800 window.
**When**: `/admin/sign-in` is loaded.
**Then**: One `img` has alt `Boxes stacked in a stockroom`. In document order after it: the wordmark; one `h1` reading `Đôi — dealer sign-in`; the line `Sign in to see orders, stock and sales.`; three lines reading, in this order, `Orders to confirm, pack and send, newest first.`, `Stock for every pair, size by size, and what sold this month.`, `The sizes running out, before a customer finds them gone.` (each with an icon beside it); a `form` with a label `Email` (input placeholder `you@example.com`), a label `Password`, and a button `Sign in`; and after the form the note `Accounts are for the dealer and staff. Ask the owner for one.` The text `Signed in as` is absent.
Traces: SC-21 (ADMIN-001 AC-1, item 10)
Check: render_url

**Scenario**: TC-49 — the picture sits at the left, the column at the right, in the management palette
**Given**: No session; a 1280×800 window; `/admin/sign-in` loaded.
**When**: The boxes and styles of `img[alt='Boxes stacked in a stockroom']`, `h1`, the line under it, the first of the three lines, `form`, the note after the form, `form button[type='submit']` and `[data-dashboard='true']` (the page root) are measured.
**Then**: The picture's right edge is at or left of the `h1`'s left edge, and the picture's left edge is 0. Top edges increase in the order `h1` < line < first of the three lines < `form` < note. The root's background is `rgb(248, 250, 252)` and the `h1`'s colour `rgb(15, 23, 42)`. The submit button's background is `rgb(37, 99, 235)`, its text `rgb(255, 255, 255)` and its width equals the form's width (±1px).
Traces: SC-21 (ADMIN-001 AC-1, item 10)
Check: measure_styles

**Scenario**: TC-50 — neither the customer header nor the footer is drawn
**Given**: No session; a 1280×800 window.
**When**: `/admin/sign-in` is loaded and its elements are searched.
**Then**: There is no `footer` element, no `header` element, no `[data-testid='bag-button']`, no link to `/shop`, `/cart` or `/track`, and no link reading `Dealer sign-in`; the page holds exactly one `h1`.
Traces: SC-21 (ADMIN-001 item 10)
Check: render_url

**Scenario**: TC-51 — on a phone the picture is above the form and the page does not scroll sideways
**Given**: No session; `/admin/sign-in` loaded at 390×844, at 320×568, at 1023×800 and at 1024×800.
**When**: At each width the picture's bottom edge, the `h1`'s top edge, the submit button's right edge and `document.documentElement.scrollWidth` are measured.
**Then**: At 390, 320 and 1023 the picture's bottom edge is at or above the `h1`'s top edge and the picture is at least 90% of the window's width; at 1024 the picture's right edge is at or left of the `h1`'s left edge. At every width `scrollWidth` is at most the window's width and the submit button's right edge is inside the window.
Traces: SC-22 (ADMIN-001 AC-2; Responsive NFR)
Check: measure_styles

**Scenario**: TC-52 — the fields: placeholder, hidden password, visible labels
**Given**: No session; a 1280×800 window on `/admin/sign-in`.
**When**: `input[type='email']` and `input[type='password']` are read; `abc-123` is typed into the password field; the labels are measured.
**Then**: The email input's placeholder is `you@example.com`. The password input's `type` is `password` and after typing, its `type` is still `password` (what is typed is not shown as text). A visible `label` reading `Email` is tied to the email input and a visible `label` reading `Password` to the password input (each input's accessible name is its label; the labels' computed `display` is not `none` and `visibility` is `visible`).
Traces: SC-23 (ADMIN-001 AC-3)
Check: interact_page

**Scenario**: TC-53 — Enter in either field submits the form
**Given**: No session; a 1280×800 window on `/admin/sign-in`; the page's request log is read.
**When**: (a) `x@example.com` is typed into Email, the Password field is left empty, and Enter is pressed in the Email field. (b) Then the field is cleared, the admin's `A_EMAIL` is typed into Email and `A_PASS` into Password and Enter is pressed in the Password field.
**Then**: (a) The text `A password is needed` appears (the form was submitted by Enter in the first field) and no request to `/api/v1/auth/sign-in` was sent. (b) The address becomes `/admin` and the area's menu `nav[aria-label='Sections']` is drawn.
Traces: SC-23 (ADMIN-001 AC-5)
Check: interact_page

**Scenario**: TC-54 — the admin signs in and lands on the overview, passing through sign-in only once
**Given**: No session; the admin exists; a 1280×800 window on `/admin/sign-in`.
**When**: `A_EMAIL` is typed into Email and `A_PASS` into Password and `Sign in` is clicked; the navigation log is read; then `/admin/orders` and `/admin/products` are opened.
**Then**: The address becomes `/admin`; `nav[aria-label='Sections']` and the overview's `h1` are drawn; after the click `/admin/sign-in` is not navigated to again (the area is not bounced back to the sign-in page); no error text is on the page. `/admin/orders` stays at that address with at least one row link `a[href^='/admin/orders/']`; `/admin/products` stays at that address with at least one `a[href^='/admin/products/']`.
Traces: SC-24 (ADMIN-001 AC-4; ADMIN-002 AC-20)
Check: interact_page

**Scenario**: TC-55 — a staff account signs in and reaches the same three pages
**Given**: The fixture rows exist; no session; a 1280×800 window on `/admin/sign-in`.
**When**: `minh.le@example.com` / `PW` is typed and `Sign in` clicked; then `/admin/orders`, `/admin/products` are opened.
**Then**: The address becomes `/admin` and the overview is drawn (menu and `h1`); `/admin/orders` shows at least one `a[href^='/admin/orders/']`; `/admin/products` shows at least one `a[href^='/admin/products/']`; `GET /api/v1/auth/me` from the page says `role` `staff`.
Traces: SC-24 (ADMIN-002 AC-16, AC-20)
Check: interact_page

**Scenario**: TC-56 — wrong credentials show one sentence and say nothing about which part was wrong
**Given**: The admin exists; `DELETE FROM rate_limits` runs before and after; no session; a 1280×800 window on `/admin/sign-in`.
**When**: (a) `A_EMAIL` and `not-the-password` are typed and `Sign in` clicked; the page's text and the two fields are read. (b) Then `nobody@example.com` and `not-the-password`, `Sign in`.
**Then**: In both (a) and (b) the page shows exactly `Wrong email or password.` once, and no text containing `unknown`, `no account`, `incorrect password`, `not found` or `locked`. The password field is empty; the email field still holds what was typed. The address stays `/admin/sign-in`.
Traces: SC-25 (ADMIN-001 AC-6, AC-7, item 5)
Check: interact_page

**Scenario**: TC-57 — the error line is announced, in the danger colour, under the fields
**Given**: The state of TC-56 (a) after the error is shown.
**When**: The element holding `Wrong email or password.` is read and measured against `input[type='password']`.
**Then**: The element, or an ancestor inside the form's block, has `role='alert'` or an `aria-live` of `assertive` or `polite` (it is announced to assistive technology). Its computed colour is `rgb(220, 38, 38)`; its top edge is at or below the password input's bottom edge; it lies inside the `[data-dashboard='true']` scope.
Traces: SC-25 (story §5 "UI scope")
Check: measure_styles

**Scenario**: TC-58 — a blank email or blank password is stopped in the browser, with no request
**Given**: No session; a 1280×800 window on `/admin/sign-in`; the page's request log is read.
**When**: (a) `Sign in` is clicked with both fields empty. (b) `   ` (spaces) is typed into Email and `Sign in` clicked. (c) `x@example.com` is typed into Email, Password is left empty, and `Sign in` is clicked.
**Then**: (a) and (b) show `An email address is needed` in the same field block as `input[type='email']` (beside the email field). (c) shows `A password is needed` in the same field block as `input[type='password']`. In all three no request to `/api/v1/auth/sign-in` is recorded and the button still reads `Sign in`.
Traces: SC-26 (ADMIN-001 AC-8)
Check: interact_page

**Scenario**: TC-59 — while the shop takes 3 seconds the button reads "Signing in…" and is disabled
**Given**: The admin exists; `DELETE FROM rate_limits` runs before and after; the answer to `POST /api/v1/auth/sign-in` is slowed to about 3 seconds (the instrument's request delay or the `doi_test_slow` trigger of "Fixtures"); a 1280×800 window on `/admin/sign-in` with `A_EMAIL` and `not-the-password` typed.
**When**: `Sign in` is clicked; one second later the button is read and clicked again, and Enter is pressed in the Password field; when the answer arrives the button is read again.
**Then**: One second after the click `form button[type='submit']` reads exactly `Signing in…` and is `disabled`; the request log shows exactly one `POST /api/v1/auth/sign-in` despite the second click and the Enter. After the answer the button reads `Sign in` and is enabled, and `Wrong email or password.` is shown.
Traces: SC-27 (ADMIN-001 AC-9, item 6)
Check: interact_page

**Scenario**: TC-60 — the shop cannot be reached: the message, an enabled button, the email kept
**Given**: The admin exists; the backend is stopped (`set_backend_state`; the gateway answers 502); no session; a 1280×800 window on `/admin/sign-in` with `A_EMAIL` and `A_PASS` typed.
**When**: `Sign in` is clicked. Then the backend is started again and `Sign in` is clicked once more.
**Then**: The page shows exactly `We cannot reach the server. Please try again.`; `form button[type='submit']` reads `Sign in` and is enabled; the email field still holds `A_EMAIL`; the address stays `/admin/sign-in`; `Wrong email or password.` is not shown. After the backend is back the second click lands on `/admin`.
Traces: SC-28 (ADMIN-001 AC-10, item 7)
Check: interact_page

**Scenario**: TC-61 — a 5xx answer shows the same unreachable message
**Given**: The admin exists; `set_db_available` takes the database away so `POST /api/v1/auth/sign-in` answers 500; no session; a 1280×800 window on `/admin/sign-in` with `A_EMAIL` and `A_PASS` typed.
**When**: `Sign in` is clicked. The database is given back afterwards.
**Then**: The page shows exactly `We cannot reach the server. Please try again.`; the button reads `Sign in` and is enabled; the email field still holds `A_EMAIL`; `Wrong email or password.` is not shown.
Traces: SC-28 (ADMIN-001 AC-10; services §2.3 `internal`)
Check: interact_page

**Scenario**: TC-62 — the sixth try shows "Too many tries. Please wait 15 minutes."
**Given**: The admin exists; `DELETE FROM rate_limits` runs before and after; no session; a 1280×800 window on `/admin/sign-in`.
**When**: `A_EMAIL` with `not-the-password` is submitted five times (the text read after each); then `A_EMAIL` with the right `A_PASS` is submitted a sixth time.
**Then**: After each of the first five the page shows `Wrong email or password.` (never the too-many message). After the sixth it shows exactly `Too many tries. Please wait 15 minutes.`, the button reads `Sign in` and is enabled, the address stays `/admin/sign-in` (the right password did not sign in), and the email field still holds `A_EMAIL`.
Traces: SC-29 (ADMIN-001 AC-11, item 8)
Check: interact_page

**Scenario**: TC-63 — "Signed in as <email>. Continue" with a session, and Continue goes to `/admin`
**Given**: The admin signed in through the form (so the browser holds the cookie); a 1280×800 window.
**When**: `/admin/sign-in` is opened; the text under the form is read; `Continue` is clicked.
**Then**: Under the form (below the submit button) the page shows `Signed in as <A_EMAIL>. Continue`; clicking `Continue` makes the address `/admin` with the area drawn.
Traces: SC-30 (ADMIN-001 AC-12, item 9)
Check: interact_page

**Scenario**: TC-64 — the line is absent with no session and not drawn while `me` is awaited
**Given**: (a) No session. (b) The admin signed in, and the answer to `GET /api/v1/auth/me` is held for 2 seconds (the instrument's request delay).
**When**: (a) `/admin/sign-in` is loaded and read. (b) `/admin/sign-in` is opened and the page's text is sampled at 0.5 s and 1.5 s, then after the answer.
**Then**: (a) The text `Signed in as` is absent. (b) At 0.5 s and 1.5 s the text `Signed in as` is absent; after the answer it reads `Signed in as <A_EMAIL>. Continue`.
Traces: SC-30 (ADMIN-001 AC-12, item 9)
Check: interact_page

---

## E. The gate and the session (SC-31 … SC-39)

**Scenario**: TC-65 — with no session every area route ends at the sign-in page, and none of its content is drawn
**Given**: No session; a fresh browser context; the page's request log is read.
**When**: Each of `/admin`, `/admin/orders`, `/admin/orders/doi-24817`, `/admin/products`, `/admin/products/samba-og-w-core-black`, `/admin/lines`, `/admin/categories`, `/admin/reviews` is opened in turn.
**Then**: For each, the settled address is `/admin/sign-in` and the page shows the `h1` `Đôi — dealer sign-in`. At no sampled moment is `nav[aria-label='Sections']` or an area page's `h1` present, and the only API request sent is `GET /api/v1/auth/me` (no `/api/v1/orders`, `/api/v1/products`, … request leaves the page).
Traces: SC-31 (ADMIN-002 AC-6, AC-15, item 4)
Check: interact_page

**Scenario**: TC-66 — while `me` takes 2 seconds the area draws nothing and does not redirect
**Given**: (a) The admin signed in. (b) No session. In both, the answer to `GET /api/v1/auth/me` is held for 2 seconds (the instrument's request delay).
**When**: `/admin/orders` is opened and the address and the page are sampled at 0.5 s and 1.5 s and after the answer.
**Then**: (a) At 0.5 s and 1.5 s the address is still `/admin/orders`, and there is no `nav[aria-label='Sections']`, no `h1`, no table or list and no figure; after the answer the area is drawn at the same address `/admin/orders`. (b) At 0.5 s and 1.5 s the address is still `/admin/orders` (no redirect yet) and nothing of the area is drawn; after the answer the address is `/admin/sign-in`.
Traces: SC-32 (ADMIN-002 AC-7, item 3)
Check: interact_page

**Scenario**: TC-67 — a reload of an area page keeps the session and the page
**Given**: The admin signed in through the form; `/admin/orders` is open and shows its list.
**When**: The page is reloaded.
**Then**: The address stays `/admin/orders`; the orders list is drawn (at least one `a[href^='/admin/orders/']`); the request log shows `GET /api/v1/orders` answered 200 (the data call carried the session cookie), not 401.
Traces: SC-33 (ADMIN-002 AC-5, item 4)
Check: interact_page

**Scenario**: TC-68 — a forged storage entry opens nothing; a real sign-in writes nothing to storage
**Given**: No cookie. In the page's `localStorage` the key `agentteam:session` is set to `{"email":"forged@example.com","name":"Forged","role":"admin"}` and in `sessionStorage` the same key to the same value.
**When**: `/admin` is opened; then `GET /api/v1/orders` is sent with `fetch_url` and no cookie. Then the storage entries are removed, the admin signs in through the form, and `localStorage` and `sessionStorage` are read.
**Then**: The settled address is `/admin/sign-in` and no area content is drawn; `GET /api/v1/orders` answers 401 `unauthenticated`. After the real sign-in `localStorage` holds no `agentteam:session` key and no other key, and `sessionStorage` holds no key (the session lives only in the cookie).
Traces: SC-34 (ADMIN-002 AC-8, AC-9, item 5)
Check: interact_page

**Scenario**: TC-69 — `session.ts` neither reads nor writes browser storage
**Given**: The pull request's `src/lib/session.ts` (and `AdminGate.tsx`).
**When**: Their text is searched.
**Then**: Neither contains `localStorage`, `sessionStorage` or the key `agentteam:session`; both reach the server only through `src/lib/api.ts`.
Traces: SC-34 (code review)
Check: manual — a property of the source; no running page shows what a file reads. The instrument is `get_pr` on the pull request.

**Scenario**: TC-70 — the sign-out button on the overview, a list page and a record page
**Given**: The admin signed in through the form. The `session` cookie value is noted from the instrument's cookie jar as `C` before each press, where the instrument shows cookies.
**When**: For each of `/admin`, `/admin/orders`, `/admin/orders/doi-24817` in turn (signing in again between them): `button[aria-label='Sign out']` is clicked; the address is read; `GET /api/v1/auth/me` is sent with `fetch_url` and with `Cookie: session=C`.
**Then**: Each time the address becomes `/admin/sign-in` and the area's menu is gone. `/me` with `C` answers 401 `unauthenticated`, and `GET /api/v1/orders` with `C` answers 401; the browser's own `GET /api/v1/auth/me` answers 401.
Traces: SC-35 (ADMIN-002 AC-10, AC-11)
Check: interact_page

**Scenario**: TC-71 — Back after sign-out shows the sign-in page, not the admin page or its data
**Given**: The admin signed in; the visitor opened `/admin` then `/admin/orders` and clicked `button[aria-label='Sign out']` there; the address is `/admin/sign-in`.
**When**: The browser's Back button is pressed (once, then once more).
**Then**: After each Back the settled address is `/admin/sign-in`; no `nav[aria-label='Sections']`, no order row link `a[href^='/admin/orders/']` and no figure is present at any sampled moment; no `GET /api/v1/orders` is answered 200 after the sign-out.
Traces: SC-35 (ADMIN-002 AC-12)
Check: interact_page

**Scenario**: TC-72 — the menu's foot for the admin: initial, name, role
**Given**: The backend is started with `ADMIN_EMAIL=lan.tran@example.com` and a password of 10+ characters; the admin signs in through the form; a 1280×800 window on `/admin`.
**When**: The block containing `button[aria-label='Sign out']` (the foot of the left menu) is read.
**Then**: The initial tile reads `L`, the name reads `Lan Tran` and the role line reads `admin` (not the email).
Traces: SC-36 (ADMIN-002 AC-4, item 7)
Check: interact_page

**Scenario**: TC-73 — the menu's foot for a staff account
**Given**: The fixture row `minh.le@example.com` (name `Minh Le`, role `staff`); the visitor signs in through the form; a 1280×800 window on `/admin`.
**When**: The foot of the left menu is read.
**Then**: The initial tile reads `M`, the name reads `Minh Le` and the role line reads `staff`.
Traces: SC-36 (ADMIN-002 item 7)
Check: interact_page

**Scenario**: TC-74 — a customer signs in at `/admin/sign-in`, lands on `/admin` and is shown the team-only message
**Given**: The fixture row `lan.nguyen@example.com` (role `customer`); no session; a 1280×800 window on `/admin/sign-in`; the page's request log is read.
**When**: `lan.nguyen@example.com` / `PW` is submitted; the address and page are read; then `/admin/orders` is opened.
**Then**: The sign-in succeeds (no error text) and the address becomes `/admin`. On `/admin` and again on `/admin/orders` (the address stays `/admin/orders`, no redirect) the page shows exactly `This area is for the shop's team` (compared with an ASCII or a typographic apostrophe) and a link `Back to the shop` whose `href` is `/`; there is no `nav[aria-label='Sections']`, no table, list, figure or `h1` of an area page; the request log holds no request to `/api/v1/orders` (or any other dealer endpoint). `GET /api/v1/orders` with the customer's cookie answers 403 (TC-39).
Traces: SC-37 (ADMIN-001 item 4; ADMIN-002 AC-17, items 9, 10)
Check: interact_page

**Scenario**: TC-75 — the team-only message is plain text in the management palette; its link goes home
**Given**: The customer of TC-74 signed in; a 1280×800 window on `/admin/orders`.
**When**: The page's root and the message are measured; the elements are searched; `Back to the shop` is clicked.
**Then**: The page root `[data-dashboard='true']` has background `rgb(248, 250, 252)` and the message's colour is `rgb(15, 23, 42)`. There is no `footer`, no `header`, no `[data-testid='bag-button']` and no `nav[aria-label='Sections']`. After the click the address is `/` and the customer site's header (`[data-testid='bag-button']`) is drawn.
Traces: SC-37 (story §5 "Gate")
Check: measure_styles

**Scenario**: TC-76 — `me` cannot be reached: the area is not drawn and the sign-in page shows
**Given**: The admin signed in through the form (the browser holds a valid cookie); the backend is then stopped (`set_backend_state`; the gateway answers 502).
**When**: `/admin/orders` is opened.
**Then**: The settled address is `/admin/sign-in`; at no sampled moment is `nav[aria-label='Sections']` or an order row present; the text `Signed in as` is absent (the visitor is not treated as signed in). After the backend is started again, opening `/admin/orders` draws the orders list at `/admin/orders` (the cookie was never lost).
Traces: SC-38 (ADMIN-002 Upstream failure)
Check: interact_page

**Scenario**: TC-77 — `me` answers 5xx: the same
**Given**: The admin signed in through the form; `set_db_available` takes the database away so `GET /api/v1/auth/me` answers 500.
**When**: `/admin/orders` is opened. The database is given back afterwards.
**Then**: The settled address is `/admin/sign-in`; nothing of the area is drawn at any sampled moment; `Signed in as` is absent. After the database is back, `/admin/orders` draws the list.
Traces: SC-38 (ADMIN-002 Upstream failure)
Check: interact_page

**Scenario**: TC-78 — the module still exports its five names, and `useRole` is the role or nothing
**Given**: The pull request's `src/lib/session.ts` and its frontend tests.
**When**: Its exports are listed and `useRole()` is called in a test for sessions of each role and for none.
**Then**: `useSession`, `signIn`, `signOut`, `useRole` and `allowed` are all exported; `useRole()` returns `"admin"`, `"staff"` and `"customer"` for a session of each role and `""` when nobody is signed in. The browser's behaviour for the four states is observed in TC-72/TC-73 (foot shows `admin` / `staff`), TC-74 (customer gets the message) and TC-65 (nobody is sent to sign-in).
Traces: SC-39 (ADMIN-002 AC-19, item 8)
Check: manual — a hook's return value is not visible in a page except through the behaviours named above; the instrument is `run_tests` (frontend) and reading the test and the file in the pull request (`get_pr`).

**Scenario**: TC-79 — what the pull request changes, and what it must not
**Given**: The frontend pull request's file list and diff (`get_pr`).
**When**: The changed files are read.
**Then**: No `src/pages/Admin*.tsx` file other than `AdminSignIn.tsx` changed; `DashboardLayout.tsx` and `RequireRole.tsx` did not change; `src/App.tsx`'s diff is confined to the `SignedIn` function and its imports (no `<Route>` line differs: the eight area routes, `/admin/sign-in` and the twelve customer routes read as before); the gate's body lives in `src/components/admin/AdminGate.tsx`; `content.json`, `theme.css`, `index.css` and `canvas-mode.ts` are untouched. The role-chooser prop (`roles`) is gone from `SignInForm`.
Traces: SC-39 (story item 9, 10)
Check: manual — a property of a diff, not of the running page. The instrument is `get_pr`.

---

## F. Customer pages gain no way into the area (SC-40)

**Scenario**: TC-80 — on all twelve customer routes the only link into `/admin` is the footer's "Dealer sign-in"
**Given**: No session; a 1280×800 window.
**When**: Each of `/`, `/shop`, `/shop/samba-og-w-core-black`, `/size-chart`, `/cart`, `/checkout`, `/orders/doi-24817`, `/track`, `/shipping`, `/returns`, `/about`, `/contact` is loaded and its links are listed.
**Then**: On each, exactly one link has an address starting `/admin` (`a[href^='/admin']`): the footer's, with text `Dealer sign-in` and `href` `/admin/sign-in`. No link anywhere on the page (header, bag, main) reads `Sign in`, `Log in`, `Account` or `My account`.
Traces: SC-40 (ADMIN-001 AC-13; ambiguity 6)
Check: render_url

**Scenario**: TC-81 — the phone menu and the bag drawer add no sign-in link
**Given**: No session; a 390×844 window on `/`.
**When**: The menu button is pressed and the open menu's links are read; the menu is closed; the bag button `[data-testid='bag-button']` is pressed and the open drawer's links are read.
**Then**: Neither the open menu nor the open bag drawer contains a link whose address starts `/admin` or whose text is `Sign in`, `Log in`, `Account` or `My account`; on the page behind them the only `a[href^='/admin']` is the footer's `Dealer sign-in`.
Traces: SC-40 (ADMIN-001 AC-13)
Check: interact_page

**Scenario**: TC-82 — a signed-in customer sees the same: the footer link only
**Given**: The customer fixture signed in through `/admin/sign-in` (TC-74); a 1280×800 window.
**When**: `/`, `/shop` and `/track` are loaded and their `a[href^='/admin']` links are listed.
**Then**: On each page the only link starting `/admin` is the footer's `Dealer sign-in` (`/admin/sign-in`); no header entry reads `Account`, `Sign in` or the customer's name (this story adds none).
Traces: SC-40 (ADMIN-001 AC-13)
Check: interact_page

---

## Traceability — every acceptance criterion has at least one case

| Criterion | Cases | Criterion | Cases |
|---|---|---|---|
| SC-1 | TC-01, TC-02 | SC-21 | TC-48, TC-49, TC-50 |
| SC-2 | TC-03 | SC-22 | TC-51 |
| SC-3 | TC-04, TC-05, TC-06 | SC-23 | TC-52, TC-53 |
| SC-4 | TC-07, TC-08, TC-09 | SC-24 | TC-54, TC-55 |
| SC-5 | TC-10, TC-11 | SC-25 | TC-56, TC-57 |
| SC-6 | TC-12 | SC-26 | TC-58 |
| SC-7 | TC-13, TC-14 (and TC-10) | SC-27 | TC-59 |
| SC-8 | TC-15, TC-16, TC-17 | SC-28 | TC-60, TC-61 |
| SC-9 | TC-20, TC-21, TC-22, TC-23 | SC-29 | TC-62 |
| SC-10 | TC-24, TC-25, TC-26 | SC-30 | TC-63, TC-64 |
| SC-11 | TC-27 | SC-31 | TC-65 |
| SC-12 | TC-28, TC-29 | SC-32 | TC-66 |
| SC-13 | TC-30, TC-31 | SC-33 | TC-67 |
| SC-14 | TC-32, TC-33 | SC-34 | TC-68, TC-69 |
| SC-15 | TC-34, TC-35 | SC-35 | TC-70, TC-71 |
| SC-16 | TC-36, TC-37, TC-38 | SC-36 | TC-72, TC-73 |
| SC-17 | TC-39, TC-40 | SC-37 | TC-74, TC-75 |
| SC-18 | TC-41 | SC-38 | TC-76, TC-77 |
| SC-19 | TC-42, TC-43 | SC-39 | TC-78, TC-79 |
| SC-20 | TC-44, TC-45 (and TC-08) | SC-40 | TC-80, TC-81, TC-82 |
| Contract (`POST /v1/auth/sign-in`, `…/sign-out`, `GET /v1/auth/me`: shapes, envelopes, 400/413/415/422/429/500) | TC-16, TC-17, TC-18, TC-19, TC-30, TC-33, TC-46, TC-47 | | |

**Roles.** Visitor: TC-39 (401), TC-65, TC-30. Customer: TC-12, TC-39 (403), TC-74, TC-75, TC-82. Staff: TC-12, TC-34, TC-36, TC-38, TC-39 (200), TC-55, TC-73. Admin: TC-01, TC-39 (200), TC-54, TC-72. The admin-only half of `auth.Require` is TC-41 (no route is admin-only until the staff story; ADMIN-004 AC-11, AC-12 are proved there on the test routes).

**Not in this story, so not covered here (story §4):** the 401/403 checks on the shoe, line, category, review, order and upload write routes (ADMIN-004 AC-1 to AC-6, AC-9 — "Manage shoes, lines, categories, reviews and orders"); `GET /v1/admin/overview` (ADMIN-020/021); `/v1/admin/staff` (ADMIN-030/031 — AC-11, AC-12 of ADMIN-004 are proved on test routes, TC-41); customer sign-up and `/account/*`; "Your session ended. Sign in again." (ADMIN-011).

**Coverage split.** 82 cases: 71 observed with an instrument (`fetch_url` 41, `interact_page` 25, `render_url` 2, `measure_styles` 3) and 11 `manual` for what no instrument can see — a stored hash (TC-03, TC-35), the start-up and request logs (TC-08, TC-45), a repository file (TC-11), a unit-test code path (TC-14), the admin-only guard on test routes (TC-41), the Go toolchain with no database (TC-43), and the source and diff of the frontend pull request (TC-69, TC-78, TC-79).
