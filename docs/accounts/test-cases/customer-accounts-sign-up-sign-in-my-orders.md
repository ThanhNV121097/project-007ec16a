# Test cases — Customer accounts: sign up, sign in, my orders

Module: `accounts` · Plan item 15 · Requirements ACC-001 to ACC-008 (`docs/accounts/SRS.md`) · Story `docs/accounts/stories/customer-accounts-sign-up-sign-in-my-orders.md` (acceptance criteria SC-1 to SC-40) · Contract `docs/architecture/services.md` §2.1–2.7, §3.7–3.11, §3.18–3.21.

**Risk level: high.** The story creates credentials, stores personal data (phone, address) and draws the line between one customer's orders and another's. A hole here is an account takeover, a leak of someone's address or a way to read orders by guessing ids, so every refusal (401, 403, 404, 409, 422, 429) has its own case, not only the happy path. The pages that only draw copy are covered in proportion: one case for the words, one for each state the story names.

Case ids are `SC-<n>.<k>`: the story criterion it serves and a running number. A case that serves the service contract rather than a criterion is `C-<k>` and says `Traces: contract (...)`.

## 0. Conventions every case relies on

### 0.1 People and data

| Name | Details |
|---|---|
| **Lan** | customer: name `Lan Nguyen`, email `lan@example.com`, phone `0912 345 678`, password `lanpass123` (10 characters) |
| **Bao** | a second customer: name `Bao Tran`, email `bao@example.com`, phone `0987 654 321`, password `baopass123` |
| **Minh** | a staff member: sign up `Minh` / `minh@example.com` / `0905 111 222` / `minhpass123` through `POST /v1/account/sign-up`, then `UPDATE users SET role = 'staff' WHERE email = 'minh@example.com'` with `set_db_state`. (The staff-accounts story is built in parallel, so a staff row is made this way, as the sign-in story does.) |
| **Admin** | the admin account of the running environment: `ADMIN_EMAIL` / `ADMIN_PASSWORD` from the run's settings |
| **The Shoe** | any live shoe from `GET /v1/products` that has at least 15 pairs in one EU size (call it the Size) |
| **Order DOI-24818** | a guest order whose number is `DOI-24818` (id `doi-24818`) for phone `0912 345 678` and linked to no account |

"Sign up", "sign in", "sign out" in a `Given` mean the real calls (`POST /v1/account/sign-up`, `POST /v1/auth/sign-in`, `POST /v1/auth/sign-out`) and "Lan's cookie" is the `session` cookie they returned, sent back in a `Cookie` header. Every `fetch_url` call that sends JSON sets `Content-Type: application/json`.

### 0.2 Standing arrangements

- **R — reset the counters.** `DELETE FROM rate_limits` with `set_db_state`. Sign-up is limited to 10 attempts an hour per address and sign-in to 5 failures per 15 minutes, and every case of this file makes calls from the same address, so a case that depends on a counter starts with R and a case that does not depend on one must not be starved by an earlier one: run R before every case that signs up, signs in wrongly or looks an order up wrongly.
- **Clean accounts.** Before a case that signs up Lan, Bao or Minh, delete their `users` rows (and so their sessions) with `set_db_state` unless the case says they already exist. Orders made for a case are deleted after it (`order_items` and `order_tokens` first).
- **Placing an order.** `POST /v1/orders` with a fresh `Idempotency-Key` (a new UUID) and the body of `services.md` §3.10: one line of the Shoe in its Size, `"delivery":"standard"`, `"payment":"vnpay"`, `customer`, `phone`, `address`, `city`. If the run has no VNPay keys the shop answers 503 `payment_unavailable` and no order can be placed through the API: the order is then inserted with `set_db_state` (an `orders` row with its `order_items`, an `order_tokens` row holding the SHA-256 of a token the case knows, `account_id` as the case says) and the case notes that. The cases that are about *placing* an order (SC-11, SC-12, SC-37, SC-38) then report UNVERIFIED for the part that needs the real call and say so.
- **Twelve orders.** For a case that needs many orders, place or insert them and then give each a distinct `placed_at`, one minute apart, with `set_db_state`, so "newest first" is unambiguous.
- **Backend stopped.** The story's backend is stopped, so the gateway answers 502 to every `/api` call: the browser sees a server error where there is no JSON. It is started again after the case.
- **Slow answer.** A named endpoint's answer is delayed by the stated number of seconds (the page's request is held, for example with `interact_page`'s request interception where it offers one). Where the instrument offers no way to hold one request, the case reports UNVERIFIED and names that instrument; it is never reported as passed from reading code.
- **Table taken away.** A store failure is arranged with `set_db_state` by renaming a table (`ALTER TABLE users RENAME TO users_x`) and renaming it back after the case.
- **Browser.** `interact_page` and `render_url` start from a browser with no cookies and no `localStorage` / `sessionStorage` unless the `Given` says the person is signed in; "signed in as Lan in the browser" means through the page's own sign-in form. The window is 1280×800 unless a width is stated. Selectors below are elements by their place or label (`main h1`, `input[type=password]`, `[role=alert]`, `header a[href="/account"]`), never a class.
- **Error body.** Every non-2xx body is `{"error": <code>, "message": <sentence>, "errors": {field: sentence}}`; `errors` is present only on 400, 422 and a 409 `conflict` of sign-up (`services.md` §2.3). A case that says "the same body" compares the JSON text byte for byte (the `X-Request-Id` header differs and is not part of the body).

---

## 1. Backend — sign-up (`POST /v1/account/sign-up`)

**Scenario**: SC-1.1 — A valid sign-up creates a customer, signs them in and answers 201
**Given**: R done. No account with email `lan@example.com`.
**When**: `POST /v1/account/sign-up` with `{"name":"Lan Nguyen","email":"lan@example.com","phone":"0912 345 678","password":"lanpass123"}`; then `GET /v1/auth/me` with the cookie the answer set.
**Then**: Status 201, `Content-Type` `application/json`, body exactly `{"name":"Lan Nguyen","email":"lan@example.com","role":"customer"}` (those three keys and no other). `Set-Cookie` carries `session=<token of at least 22 characters>` with `HttpOnly`, `SameSite=Lax`, `Path=/` and `Max-Age=604800`. `GET /v1/auth/me` answers 200 `{"email":"lan@example.com","name":"Lan Nguyen","role":"customer"}` with `Cache-Control: no-store`.
Traces: SC-1 (ACC-001 AC-2, item 3; ACC-007 AC-6 cookie)
Check: fetch_url

**Scenario**: SC-1.2 — The email is stored in lower case
**Given**: R done. No account for `lan.nguyen@example.com`.
**When**: Sign up with email `Lan.Nguyen@Example.COM` and otherwise valid details; then `POST /v1/auth/sign-in` with `lan.nguyen@example.com` and the password; then `GET /v1/account` with the cookie.
**Then**: Sign-up answers 201 with `"email":"lan.nguyen@example.com"`; the sign-in answers 200 `"role":"customer"`; `GET /v1/account` shows `"email":"lan.nguyen@example.com"`. A `SELECT email FROM users` read with `set_db_state` shows the lower-case value.
Traces: SC-1 (ACC-001 item 4; ACC-002 Boundary row)
Check: fetch_url

**Scenario**: SC-1.3 — The role is never taken from the request: a body carrying `role` is refused 400 and makes no account
**Given**: R done. No account for `lan@example.com`.
**When**: Sign-up body valid plus `"role":"admin"`; then the same valid body plus `"is_active":true`.
**Then**: Each answers 400 `bad_request` with `errors` naming the field (`role`, then `is_active`), no `Set-Cookie`. Afterwards `POST /v1/auth/sign-in` for `lan@example.com` answers 401 `invalid_credentials` (no account was made), and `SELECT count(*) FROM users WHERE email = 'lan@example.com'` is 0.
Traces: SC-1 (ACC-001 Data touched "Role … never by the request"); contract (POST /v1/account/sign-up 400)
Check: fetch_url

**Scenario**: SC-1.4 — The password is stored only as an argon2id hash and no response ever carries a password or a hash
**Given**: R done. Lan and Bao both signed up with the same password `samepass123` (Lan's and Bao's other details as in 0.1).
**When**: Read both `users.password_hash` values with `set_db_state`; collect the success bodies of sign-up, `POST /v1/auth/sign-in`, `GET /v1/auth/me`, `GET /v1/account`, `PUT /v1/account` (body `{"city":"Hue"}`) and `GET /v1/account/orders` for Lan.
**Then**: Each stored value starts with `$argon2id$`, is not `samepass123`, and the two values are different. No collected body has a key named `password`, `password_hash` or `hash`, and none contains the text `samepass123` or `$argon2`.
Traces: SC-1 (ACC-001 AC-3); SC-6 (ACC-008 AC-6)
Check: fetch_url

**Scenario**: SC-1.5 — The password and the hash are not written to any log line
**Given**: The story running; a sign-up and a failed sign-in made with the password `logprobe-pass-77`.
**When**: The backend's log output is read for the string `logprobe-pass-77`, for `$argon2` and for the session token.
**Then**: None of the three appears in any line.
Traces: SC-1 (ACC-001 item 3; ACC-002 Data touched)
Check: manual — no instrument reads a container's log output; `fetch_url`, `render_url` and `interact_page` observe responses and pages only. The same property is asserted in the backend's Go tests.

**Scenario**: SC-2.1 — An email held by a customer is refused 409, whatever its case
**Given**: R done. Lan exists. `SELECT count(*) FROM users WHERE lower(email) = 'lan@example.com'` is 1.
**When**: Sign up with name `Other Lan`, email `LAN@example.com`, a valid phone and password.
**Then**: Status 409, body `error` = `conflict`, `errors.email` a non-empty sentence, no `Set-Cookie`. The count is still 1.
Traces: SC-2 (ACC-001 AC-4, item 4)
Check: fetch_url

**Scenario**: SC-2.2 — An email held by a staff account is refused 409
**Given**: R done. Minh exists with role `staff`.
**When**: Sign up with email `MINH@Example.com`, otherwise valid.
**Then**: Status 409, `error` = `conflict`, `errors.email` set, no `Set-Cookie`. Minh's row still has role `staff` and no second row exists for the email.
Traces: SC-2 (ACC-001 AC-5)
Check: fetch_url

**Scenario**: SC-2.3 — An email held by the admin account is refused 409
**Given**: R done. The admin account exists.
**When**: Sign up with the admin's email written in upper case, otherwise valid.
**Then**: Status 409, `error` = `conflict`, `errors.email` set, no `Set-Cookie`; the admin's row keeps role `admin` and no other row has that email.
Traces: SC-2 (ACC-001 AC-5)
Check: fetch_url

**Scenario**: SC-2.4 — Two sign-ups with one email at the same moment make one account
**Given**: R done. No account for `race@example.com`.
**When**: Two `POST /v1/account/sign-up` calls with that email (names `Race One` and `Race Two`), issued in the same round so they overlap. Then, with `set_db_state`, a direct `INSERT` of a second `users` row whose email is `RACE@example.com`.
**Then**: One call answers 201 and the other 409 `conflict`; exactly one row exists for the email. The direct insert is refused by the database with a unique violation (SQLSTATE 23505), proving the uniqueness is the table's and not only the handler's.
Traces: SC-2 (ACC-001 Failure table "Conflict")
Check: fetch_url

**Scenario**: SC-3.1 — Empty, omitted and blank fields are each named by a 422 and nothing is made
**Given**: R done.
**When**: Sign up with `{"name":"","email":"","phone":"","password":""}`; then with `{}`; then with a valid body whose `name` is `"   "` (spaces only).
**Then**: The first two answer 422 `invalid` with `errors` holding exactly the keys `name`, `email`, `phone`, `password`, each a non-empty sentence; the third answers 422 with `errors.name` only. None sets a cookie; `SELECT count(*) FROM users` is unchanged after the three calls.
Traces: SC-3 (ACC-001 AC-6, AC-7, item 2)
Check: fetch_url

**Scenario**: SC-3.2 — A name of 100 characters is accepted and 101 is refused naming the limit
**Given**: R done.
**When**: Sign up with a name of 100 letters `a` and email `n100@example.com`; then with 101 letters `a` and email `n101@example.com`.
**Then**: The first answers 201 with the 100-character name in the body. The second answers 422 `invalid`, `errors.name` a sentence that contains `100`, no cookie, no row for `n101@example.com`.
Traces: SC-3 (ACC-001 Boundary row)
Check: fetch_url

**Scenario**: SC-3.3 — An email must look like an email and be at most 200 characters
**Given**: R done.
**When**: Sign up (other fields valid, a new name each time) with each of: `lan.example.com` (no `@`), `lan@example` (no dot in the domain), `la n@example.com` (a space), an email of 201 characters (189 letters `a`, then `@example.com`), and an email of exactly 200 characters (188 letters `a`, then `@example.com`).
**Then**: The first four answer 422 `invalid` with `errors` holding the key `email` and no other key, no cookie. The 200-character one answers 201 with the same address, lower-cased.
Traces: SC-3 (ACC-001 AC-7, item 2)
Check: fetch_url

**Scenario**: SC-3.4 — A phone needs at least 9 digits and at most 20 characters
**Given**: R done.
**When**: Sign up (a new email each time) with phone `0912 34`; then `01234567` (8 digits); then `012 345 678` (9 digits); then `+84 912 345 678`; then `01234567890123456789` (20 characters); then `012345678901234567890` (21 characters).
**Then**: The first two answer 422 `errors.phone` only; the next three answer 201; the last answers 422 `errors.phone` only. Refused calls set no cookie and make no row.
Traces: SC-3 (ACC-001 AC-8, item 2)
Check: fetch_url

**Scenario**: SC-3.5 — A password needs 8 to 100 characters
**Given**: R done.
**When**: Sign up (a new email each time) with passwords of 7 characters (`abc1234`), 8 (`abc12345`), 100 (letter `p` ×100) and 101 (letter `p` ×101).
**Then**: 7 → 422 `errors.password` only; 8 → 201; 100 → 201; 101 → 422 `errors.password` only. Refused calls set no cookie.
Traces: SC-3 (ACC-001 AC-7, item 2)
Check: fetch_url

**Scenario**: SC-3.6 — A password with spaces at its ends is stored as typed
**Given**: R done. An account signed up with the password `  pass word 1  ` (two spaces at each end).
**When**: `POST /v1/auth/sign-in` with that exact password; then with `pass word 1` (the same without the end spaces).
**Then**: The first answers 200; the second answers 401 `invalid_credentials`.
Traces: SC-3 (password "never trimmed")
Check: fetch_url

**Scenario**: SC-3.7 — A body that is not JSON, has a wrong type or an undefined field is rejected before any rule
**Given**: R done.
**When**: Four calls to `POST /v1/account/sign-up`: with `Content-Type: text/plain` and a valid JSON text; with `Content-Type: application/json` and the text `not json`; with a valid body but `"name":123`; with a valid body plus `"nickname":"L"`.
**Then**: 415 `unsupported_media_type`; 400 `bad_request`; 400 `bad_request` with `errors.name`; 400 `bad_request` with `errors.nickname`. No cookie, no account made by any of them.
Traces: SC-3 (last sentence); contract (POST /v1/account/sign-up 400, 415)
Check: fetch_url

**Scenario**: SC-3.8 — A 422 names every field at fault and only those
**Given**: R done.
**When**: Sign up with a valid name, email `lan@`, a valid phone and password `abc1234`.
**Then**: 422 `invalid`; `errors` has exactly the keys `email` and `password`, and `message` is a sentence with no SQL, hash or stack text.
Traces: SC-3 (ACC-001 item 3 "naming each field at fault"); contract (§2.3 error shape)
Check: fetch_url

**Scenario**: SC-4.1 — The eleventh sign-up attempt from one address within an hour is refused 429
**Given**: R done. Ten attempts from one address, each answered as its own rule says: five valid sign-ups with new emails (201), three with the email `bad` (422), two with the email of an existing account (409). None answers 429.
**When**: An eleventh attempt with entirely valid details and an email never used.
**Then**: Status 429, `error` = `rate_limited`, a `Retry-After` header that is a whole number between 1 and 3600, no `Set-Cookie`. `POST /v1/auth/sign-in` for the eleventh's email answers 401 (no account was made).
Traces: SC-4 (ACC-001 AC-11, item 6)
Check: fetch_url

**Scenario**: SC-4.2 — A call refused with 429 does not extend the lock
**Given**: The state at the end of SC-4.1 (the address is locked); `Retry-After` of that 429 is `r1`.
**When**: Three more sign-up attempts, one after another; then read the `rate_limits` row of the `sign_up_address` scope with `set_db_state`.
**Then**: All three answer 429; each `Retry-After` is a whole number and no greater than the one before (and no greater than `r1`); the stored count and window end of the scope are the same before and after the three calls.
Traces: SC-4 ("a call refused with 429 does not extend the lock")
Check: fetch_url

**Scenario**: SC-4.3 — When the hour has passed, sign-up works again
**Given**: The address is locked as in SC-4.1.
**When**: With `set_db_state`, move the `sign_up_address` row's window and lock end back by 61 minutes (or delete the row); then sign up with a new valid email.
**Then**: Status 201 and a `Set-Cookie`.
Traces: SC-4 ("When the hour has passed, sign-up works again")
Check: fetch_url

**Scenario**: SC-4.4 — Sign-up attempts and sign-in failures are counted apart
**Given**: R done. Lan exists.
**When**: (a) Ten sign-up attempts with new emails, then `POST /v1/auth/sign-in` with Lan's right credentials. (b) R, then five wrong-password sign-ins for `lan@example.com`, then a sign-up with a new valid email.
**Then**: (a) the sign-in answers 200, not 429. (b) the five answer 401 and the sign-up answers 201, not 429.
Traces: SC-4 (last sentence)
Check: fetch_url

**Scenario**: SC-4.5 — A forged left-most `X-Forwarded-For` does not get round the sign-up limit
**Given**: R done.
**When**: Eleven sign-up attempts with new valid emails, each carrying a different `X-Forwarded-For` header (`10.0.0.1`, `10.0.0.2`, … `10.0.0.11`).
**Then**: The first ten answer 201 and the eleventh answers 429 `rate_limited` — the address counted is the right-most entry the edge appended, not the one the caller wrote.
Traces: SC-4 ("The address is the right-most `X-Forwarded-For` entry"); contract (§2.2 Client address)
Check: fetch_url

---

## 2. Backend — the shared sign-in used by customers (`POST /v1/auth/sign-in`)

**Scenario**: SC-5.1 — A customer signs in with the right credentials, and the email's case does not matter
**Given**: R done. Lan exists.
**When**: `POST /v1/auth/sign-in` with `{"email":"lan@example.com","password":"lanpass123"}`; then with `{"email":"LAN@Example.com","password":"lanpass123"}`; then `GET /v1/auth/me` with the second cookie.
**Then**: Both answer 200 `{"email":"lan@example.com","name":"Lan Nguyen","role":"customer"}` and each sets a `session` cookie (the two tokens differ); `me` answers 200 with name `Lan Nguyen`.
Traces: SC-5 (ACC-002 AC-2, Boundary row)
Check: fetch_url

**Scenario**: SC-5.2 — A wrong password and an unknown email answer exactly alike
**Given**: R done. Lan exists. A third account `off@example.com` was made and then switched off (`UPDATE users SET is_active = false`).
**When**: Sign in with Lan's email and a wrong password; with `nobody@example.com` and any password; with `off@example.com` and its right password.
**Then**: All three answer 401 with `error` = `invalid_credentials` and `message` = `Wrong email or password.`; the three bodies are byte-identical; none sets a cookie.
Traces: SC-5 (ACC-002 AC-3, Not-permitted row)
Check: fetch_url

**Scenario**: SC-5.3 — Five wrong tries lock the sixth, even with the right password
**Given**: R done. Lan exists.
**When**: Five sign-ins for `lan@example.com` with a wrong password; then a sixth with `lanpass123`; then a sign-in for `bao@example.com` with any password from the same address; then, with `set_db_state`, move the `sign_in_email` and `sign_in_address` rows back 16 minutes and sign in as Lan with the right password.
**Then**: The five answer 401. The sixth answers 429 `rate_limited` with a `Retry-After` between 1 and 900 and no cookie. The Bao attempt also answers 429 (the address is locked). After the move the right password answers 200.
Traces: SC-5 (ACC-002 AC-4)
Check: fetch_url

**Scenario**: SC-5.4 — Staff and the admin sign in through the same route and `me` names their role
**Given**: R done. Minh (staff) and the Admin exist.
**When**: Sign in as Minh; `GET /v1/auth/me` with his cookie; sign in as the Admin; `GET /v1/auth/me` with that cookie.
**Then**: The sign-ins answer 200 with `"role":"staff"` (name `Minh`) and `"role":"admin"`; `me` answers `staff` and `admin`.
Traces: SC-5 (ACC-002 AC-6, item 4)
Check: fetch_url

**Scenario**: SC-5.5 — A blank email or password is a 422 that does not count as a failed try
**Given**: R done. Lan exists.
**When**: Sign in with `{"email":"","password":"x"}` and with `{"email":"lan@example.com","password":""}`, five times each; then sign in with Lan's right credentials.
**Then**: The blank-email call answers 422 `invalid` with `errors.email` = `An email address is needed`; the blank-password call answers 422 with `errors.password` set; the final sign-in answers 200, not 429.
Traces: SC-5 / SC-22 (ACC-002 item 2); contract (POST /v1/auth/sign-in 422)
Check: fetch_url

---

## 3. Backend — saved details (`GET /v1/account`, `PUT /v1/account`)

**Scenario**: SC-6.1 — A new customer's saved details hold the sign-up values and nothing else
**Given**: R done. Lan just signed up (cookie held).
**When**: `GET /v1/account` with Lan's cookie.
**Then**: 200, body exactly `{"name":"Lan Nguyen","email":"lan@example.com","phone":"0912 345 678","address":"","city":""}`: five keys, and no `password`, `password_hash`, `hash`, `role` or `id`.
Traces: SC-6 (ACC-004 AC-1, AC-9; ACC-008 AC-6)
Check: fetch_url

**Scenario**: SC-6.2 — The account endpoints refuse no session (401) and a team session (403)
**Given**: R done. Lan, Minh and the Admin each hold a cookie.
**When**: `GET /v1/account` with no cookie, with Minh's cookie, and with the Admin's cookie.
**Then**: 401 `{"error":"unauthenticated",…}`; 403 `{"error":"forbidden",…}`; 403 `{"error":"forbidden",…}`. None of the three bodies has an `errors` key or any customer's data.
Traces: SC-6 (ACC-004 AC-8); contract (§2.2 `/v1/account…` row)
Check: fetch_url

**Scenario**: SC-6.3 — The route takes no account id: query parameters naming another account are ignored
**Given**: R done. Lan and Bao exist, Lan holds a cookie; Bao's user id is known (`SELECT id FROM users`).
**When**: `GET /v1/account?id=<Bao's id>`, `GET /v1/account?account=bao@example.com`, `GET /v1/account?user=<Bao's id>` with Lan's cookie.
**Then**: Each answers 200 with Lan's own details (`lan@example.com`); nothing of Bao's appears.
Traces: SC-6 (the route "takes no id and ignores `?id=`, `?account=` and `?user=`"); SC-16 (ACC-008 item 2)
Check: fetch_url

**Scenario**: SC-7.1 — Saving an address and a city is read back
**Given**: R done. Lan signed up (cookie held).
**When**: `PUT /v1/account` with `{"address":"12 Example Street, Ward 1","city":"Da Nang"}`; then `GET /v1/account`.
**Then**: The `PUT` answers 200 with the same five-key shape as the `GET`, holding the two new values and Lan's name, email and phone unchanged; the `GET` returns the same body.
Traces: SC-7 (ACC-004 AC-2)
Check: fetch_url

**Scenario**: SC-7.2 — A body with only `phone` changes only the phone
**Given**: Lan has address `12 Example Street, Ward 1` and city `Da Nang` saved.
**When**: `PUT /v1/account` with `{"phone":"0987 111 222"}`; then `GET /v1/account`.
**Then**: 200; `phone` is `0987 111 222`; `name`, `address` and `city` are exactly as before.
Traces: SC-7 (ACC-004 AC-3)
Check: fetch_url

**Scenario**: SC-7.3 — An address or city sent as an empty string clears it
**Given**: Lan has an address and a city saved.
**When**: `PUT /v1/account` with `{"address":"","city":""}`; then `GET /v1/account`.
**Then**: 200 with `"address":""` and `"city":""`; the `GET` agrees; name and phone unchanged.
Traces: SC-7 (ACC-004 item 2)
Check: fetch_url

**Scenario**: SC-7.4 — `email`, `role` and `password` are refused 422 and nothing changes, even beside valid fields
**Given**: Lan signed up; her city is `Da Nang`.
**When**: `PUT /v1/account` with `{"email":"x@example.com"}`; with `{"role":"admin"}`; with `{"password":"newpass12345"}`; with `{"city":"Hue","role":"admin"}`. Then `GET /v1/account`, then sign in with the old password.
**Then**: Each `PUT` answers 422 `invalid` with `errors` naming `email`, `role`, `password` and `role` respectively. The `GET` still shows email `lan@example.com` and city `Da Nang` (the valid `city` of the fourth call was not applied); the sign-in with `lanpass123` answers 200 and `me` still says `customer`.
Traces: SC-7 (ACC-004 AC-4)
Check: fetch_url

**Scenario**: SC-7.5 — The limits of checkout are enforced, each naming its field
**Given**: Lan signed up.
**When**: `PUT /v1/account` with phone `01234567` (8 digits); address of 201 characters; address of exactly 200; city of 101 characters; city of exactly 100; name `""`; name of 101 characters; name of exactly 100 — one body each.
**Then**: 422 `errors.phone`; 422 `errors.address` whose sentence contains `200`; 200; 422 `errors.city`; 200; 422 `errors.name`; 422 `errors.name`; 200. After the refused calls the stored values are those of the last accepted call for each field.
Traces: SC-7 (ACC-004 AC-5)
Check: fetch_url

**Scenario**: SC-7.6 — An undefined field, a wrong type, bad JSON and a wrong content type are refused before any rule
**Given**: Lan signed up.
**When**: `PUT /v1/account` with `{"nickname":"L"}`; with `{"phone":912345678}` (a number); with the text `{bad`; with `Content-Type: text/plain`.
**Then**: 400 `bad_request` with `errors.nickname`; 400 `bad_request` with `errors.phone`; 400 `bad_request`; 415 `unsupported_media_type`. `GET /v1/account` is unchanged.
Traces: SC-7 (ACC-004 "any other undefined field answers 400"); contract (PUT /v1/account 400, 415)
Check: fetch_url

**Scenario**: SC-7.7 — `PUT /v1/account` refuses no session (401) and a team session (403)
**Given**: Minh and the Admin hold cookies.
**When**: `PUT /v1/account` with the valid body `{"city":"Hue"}` and no cookie, then with Minh's cookie, then with the Admin's.
**Then**: 401 `unauthenticated`; 403 `forbidden`; 403 `forbidden`.
Traces: SC-7 (ACC-004 AC-8)
Check: fetch_url

---

## 4. Backend — my orders (`GET /v1/account/orders`, `GET /v1/account/orders/{id}`)

**Scenario**: SC-8.1 — Twelve orders are served ten at a time, newest first
**Given**: Lan has twelve orders linked to her account, `placed_at` one minute apart (twelve distinct values); two further orders with an equal `placed_at` exist for Bao.
**When**: `GET /v1/account/orders`; then `?page=2`; then `?page=3`.
**Then**: Page 1: 200 `{"items":[…10…],"total":12,"page":1,"limit":10}` with `placed_at` strictly descending (the newest order first). Page 2: `"page":2`, two items, the two oldest, `total` 12. Page 3: `"items":[]`, `"total":12`. No item of Bao's appears in any page.
Traces: SC-8 (ACC-003 AC-3, item 3)
Check: fetch_url

**Scenario**: SC-8.2 — Two orders with the same time keep a stable order and each item has the public subset only
**Given**: Lan has two orders with an identical `placed_at` and different sequence numbers.
**When**: `GET /v1/account/orders` twice.
**Then**: The order with the higher sequence number comes first both times (identical bodies). Every item has exactly the keys `id`, `number`, `placed_at` (RFC 3339 ending `+07:00`), `status` (one of `placed`, `confirmed`, `packed`, `shipped`, `delivered`, `cancelled`) and `total` (an integer); none has `phone`, `email`, `address`, `note`, `customer`, `city` or `items`.
Traces: SC-8 (ACC-003 AC-8, item 3; ACC-008 item 4)
Check: fetch_url

**Scenario**: SC-8.3 — A bad `page` or `limit` is a 400 naming it; an undefined query parameter is ignored
**Given**: Lan has twelve orders.
**When**: `GET /v1/account/orders?page=0`, `?page=-1`, `?page=abc`, `?limit=0`, `?limit=-5`, `?limit=x`, `?limit=500`, `?limit=5`, `?zzz=1`.
**Then**: The first three answer 400 `bad_request` with `errors.page`; the next three 400 with `errors.limit`; `limit=500` answers 200 with `"limit":100` (served as 100, `services.md` §2.4) and all twelve items; `limit=5` answers 200 with five items; `zzz=1` answers 200 exactly as the unparameterised call.
Traces: SC-8 ("A bad `page` or `limit` answers 400 naming it; another query parameter is ignored"); contract (§2.4)
Check: fetch_url

**Scenario**: SC-8.4 — A customer with no order gets an empty page
**Given**: Lan just signed up; no order is linked to her.
**When**: `GET /v1/account/orders`.
**Then**: 200 `{"items":[],"total":0,"page":1,"limit":10}`.
Traces: SC-8 (ACC-003 AC-4 data side)
Check: fetch_url

**Scenario**: SC-8.5 — The list refuses no session (401) and a team session (403)
**Given**: Minh and the Admin hold cookies.
**When**: `GET /v1/account/orders` with no cookie, with Minh's, with the Admin's.
**Then**: 401 `unauthenticated`; 403 `forbidden`; 403 `forbidden`; none holds an `items` key.
Traces: SC-8 (ACC-003 AC-9)
Check: fetch_url

**Scenario**: SC-9.1 — A customer reads one of their own orders with its pairs and no private field
**Given**: Lan has an order `doi-30001` with one pair of the Shoe.
**When**: `GET /v1/account/orders/doi-30001` with Lan's cookie.
**Then**: 200 with `id`, `number`, `placed_at`, `status`, `total` and an `items` array whose entries have `product`, `name`, `colourway`, `image`, `size`, `qty`, `unit_price`; no `phone`, `email`, `address` or `note` key anywhere in the body.
Traces: SC-9 (ACC-003 item 3)
Check: fetch_url

**Scenario**: SC-9.2 — Another customer's order, a guest order and a missing order all answer 404 with one body
**Given**: Lan's order `doi-30001`; Order DOI-24818 (a guest order); Bao signed in (cookie held); no order `doi-99999`.
**When**: As Bao: `GET /v1/account/orders/doi-30001`, `/doi-24818`, `/doi-99999`, `/Bad_Id!`.
**Then**: All four answer 404 `not_found`; the bodies of the first three are byte-identical (the fourth is also 404 `not_found`).
Traces: SC-9 (ACC-003 AC-6; ACC-008 AC-3)
Check: fetch_url

**Scenario**: SC-9.3 — The one-order read refuses no session (401) and a team session (403)
**Given**: Lan's order `doi-30001`; Minh and the Admin hold cookies.
**When**: `GET /v1/account/orders/doi-30001` with no cookie, with Minh's, with the Admin's.
**Then**: 401 `unauthenticated`; 403 `forbidden`; 403 `forbidden`.
Traces: SC-9 (ACC-003 AC-9)
Check: fetch_url

---

## 5. Backend — reading an order by link, and placing orders while signed in

**Scenario**: SC-10.1 — The customer an order is linked to reads it with no token
**Given**: Lan's order `doi-30001` is linked to her. No `X-Order-Token` is sent.
**When**: `GET /v1/orders/doi-30001` with Lan's cookie.
**Then**: 200 public view: `id`, `number`, `placed_at`, `status`, `total`, `items`; no `phone`, `email`, `address` or `note` key.
Traces: SC-10 (ACC-003 item 4; ORD-008 AC-11)
Check: fetch_url

**Scenario**: SC-10.2 — Another customer, no session and a missing id all answer 404 with one body
**Given**: Lan's order `doi-30001`; Bao signed in; no order `doi-99999`; Order DOI-24818 exists with a token that belongs to it.
**When**: `GET /v1/orders/doi-30001` with Bao's cookie and no token; the same with no cookie and no token; `GET /v1/orders/doi-99999` with Bao's cookie; `GET /v1/orders/doi-30001` with the token of `doi-24818`.
**Then**: All four answer 404 `not_found` and the four bodies are byte-identical.
Traces: SC-10 (ACC-008 AC-3; ORD-008 AC-12)
Check: fetch_url

**Scenario**: SC-10.3 — The order's own token and a dealer session still work
**Given**: Order DOI-24818 with its token; the Admin holds a cookie.
**When**: `GET /v1/orders/doi-24818` with `X-Order-Token` and no cookie; then with the Admin's cookie and no token.
**Then**: The first answers 200 with the public view (no `phone`); the second answers 200 with the dealer view, which includes `phone`, `email`, `address` and `note`.
Traces: SC-10 ("With the order's own token the read still answers 200"); contract (GET /v1/orders/{id})
Check: fetch_url

**Scenario**: SC-11.1 — An order placed while signed in is linked to the account and the typed details are kept
**Given**: Lan has saved details (`Lan Nguyen`, `0912 345 678`, `12 Example Street, Ward 1`, `Da Nang`) and holds a cookie.
**When**: `POST /v1/orders` with Lan's cookie and typed details `customer` `Gift Receiver`, `phone` `0900 000 111`, `address` `99 Other Road`, `city` `Hue`; then `GET /v1/account`; then `GET /v1/orders/<id>` with the Admin's cookie.
**Then**: 201 with the order and a token. `GET /v1/account` is unchanged (name `Lan Nguyen`, phone `0912 345 678`, the saved address and city). The dealer view of the order shows `customer` `Gift Receiver`, `phone` `0900 000 111`, `address` `99 Other Road`, `city` `Hue`.
Traces: SC-11 (ACC-005 AC-2, items 2, 5)
Check: fetch_url (if the run has no VNPay keys the order is inserted with `account_id` = Lan's id and this case reports UNVERIFIED for the call itself)

**Scenario**: SC-11.2 — The new order is in the owner's list at once and on a second session, and not in another customer's
**Given**: The order of SC-11.1 was just placed. Lan signs in a second time (a second cookie). Bao is signed in.
**When**: `GET /v1/account/orders` with Lan's first cookie, with Lan's second cookie and with Bao's cookie.
**Then**: The order's id is in the first two lists and not in Bao's.
Traces: SC-11 (ACC-005 AC-3, AC-5)
Check: fetch_url

**Scenario**: SC-11.3 — An order placed by a staff or admin session is linked to no account
**Given**: Minh and the Admin hold cookies.
**When**: `POST /v1/orders` once with Minh's cookie and once with the Admin's; then read `orders.account_id` of both with `set_db_state`; then `GET /v1/account/orders` as Lan.
**Then**: Both answer 201; both rows have `account_id` NULL; neither appears in Lan's list.
Traces: SC-11 (ACC-005 AC-7, item 5)
Check: fetch_url (UNVERIFIED for the call itself if no VNPay keys are set; the rule is then read from the Go tests)

**Scenario**: SC-12.1 — An order placed with no session is a guest order that can still be tracked
**Given**: No cookie.
**When**: `POST /v1/orders`; read `orders.account_id` with `set_db_state`; `GET /v1/orders/<id>` with the returned token; `POST /v1/order-lookups` with the order's number and phone; `GET /v1/account/orders` as Lan.
**Then**: 201 with a `token`; `account_id` is NULL; the token read answers 200; the lookup answers 200 with the id and a token; the order is not in Lan's list.
Traces: SC-12 (ACC-005 AC-4, item 3)
Check: fetch_url (see SC-11.1 for the no-keys case)

**Scenario**: SC-12.2 — A session that has ended does not refuse the order: it is placed as a guest
**Given**: Lan signs in, keeps that cookie, and then signs out (`POST /v1/auth/sign-out`). A second case: Lan's session row is made expired with `set_db_state` (`created_at` 8 days back).
**When**: `POST /v1/orders` with the old cookie, in each case.
**Then**: Both answer 201 with the order and a token; `orders.account_id` is NULL for both; neither is in any account's list; the tokens open them.
Traces: SC-12 (ACC-005 AC-6, item 4; ACC-006 item 6)
Check: fetch_url (see SC-11.1 for the no-keys case)

---

## 6. Backend — adding a guest order (`POST /v1/account/orders`)

**Scenario**: SC-13.1 — A right number and phone link a guest order to the account
**Given**: R done. Order DOI-24818 (phone `0912 345 678`, linked to none). Lan holds a cookie.
**When**: `POST /v1/account/orders` with `{"number":"doi 24818","phone":"+84912345678"}`; then `GET /v1/account/orders`.
**Then**: 200 with exactly the keys `id` (`doi-24818`), `number` (`DOI-24818`), `placed_at`, `status`, `total`; the list holds that id; `orders.account_id` of the order is Lan's user id.
Traces: SC-13 (ACC-006 AC-1, item 2)
Check: fetch_url

**Scenario**: SC-13.2 — Adding the same order again answers 200 and makes no second row
**Given**: The state at the end of SC-13.1.
**When**: The same `POST /v1/account/orders` again; then `GET /v1/account/orders`.
**Then**: 200 with the same body; the list holds exactly one item with id `doi-24818` and its `total` is unchanged.
Traces: SC-13 (ACC-006 AC-5, item 5)
Check: fetch_url

**Scenario**: SC-13.3 — The number and the phone are matched on their digits, as the lookup does
**Given**: R done. Order DOI-24818 linked to Lan.
**When**: `POST /v1/account/orders` once for each pair: number `24818`, `DOI-24818`, `doi-24818`, `DOI 24818`, `Doi—24818` with phone `0912345678`; number `doi 24818` with phone `84 912 345 678` and with `(0912) 345-678`.
**Then**: Every call answers 200 with id `doi-24818` (a mismatch would be 404).
Traces: SC-13 (ACC-006 item 2)
Check: fetch_url

**Scenario**: SC-13.4 — Linking leaves the guest browser's token and the lookup working
**Given**: Order DOI-24818 placed as a guest with a token held; Lan has linked it (SC-13.1).
**When**: `GET /v1/orders/doi-24818` with that token and no cookie; `POST /v1/order-lookups` with `{"number":"DOI-24818","phone":"0912 345 678"}`.
**Then**: Both answer 200; the lookup returns `id` `doi-24818` and a `token`.
Traces: SC-13 (ACC-006 AC-6, item 6)
Check: fetch_url

**Scenario**: SC-14.1 — Every kind of miss answers 404 with one body and links nothing
**Given**: R done. Order DOI-24818 (phone `0912 345 678`, unlinked), order `DOI-24819` (unlinked), order `DOI-24820` linked to Bao (its phone `0987 654 321`). Lan holds a cookie.
**When**: As Lan, one call each: number `24818` with the wrong phone `0912 345 679`; number `99999` (no such order) with any phone; number `24818` with phone `""`; number `24818` with phone `345 678` (only the end of the real one); number `24820` with its right phone `0987 654 321`.
**Then**: All five answer 404 `not_found` with byte-identical bodies. `orders.account_id` is still NULL for `doi-24818` and `doi-24819` and still Bao's for `doi-24820`.
Traces: SC-14 (ACC-006 AC-2, AC-3, AC-8, item 3)
Check: fetch_url

**Scenario**: SC-14.2 — An undefined field, a wrong type and a wrong content type are refused and do not count as failures
**Given**: R done. Lan holds a cookie.
**When**: `POST /v1/account/orders` with `{"number":"24818","phone":"0912345678","extra":1}`; with `{"number":24818,"phone":"0912345678"}`; with `Content-Type: text/plain`. Then six right-pair calls for an existing order.
**Then**: 400 `bad_request` with `errors.extra`; 400 `bad_request` with `errors.number`; 415 `unsupported_media_type`; the six right-pair calls answer 200, not 429.
Traces: SC-14 (last sentence); contract (POST /v1/account/orders 400)
Check: fetch_url

**Scenario**: SC-15.1 — Five failures for one number lock it, counted together with the lookup's
**Given**: R done. Order DOI-24818 for phone `0912 345 678`. Lan holds a cookie.
**When**: Five wrong-phone attempts for number `24818` — three through `POST /v1/account/orders` and two through `POST /v1/order-lookups`; then a sixth `POST /v1/account/orders` with the right pair.
**Then**: The five answer 404; the sixth answers 429 `rate_limited` with `Retry-After` between 1 and 900 and links nothing (`account_id` still NULL).
Traces: SC-15 (ACC-006 AC-4, item 4; ORD-022)
Check: fetch_url

**Scenario**: SC-15.2 — Five failures from one address lock the address even for another number
**Given**: R done. Orders DOI-24818 and DOI-24821, both unlinked. Lan holds a cookie.
**When**: Five attempts with five different unknown numbers (`90001` to `90005`); then a right-pair attempt for DOI-24821.
**Then**: The five answer 404; the right pair answers 429 `rate_limited` with `Retry-After` between 1 and 900.
Traces: SC-15 (ACC-006 AC-4 "or from one address")
Check: fetch_url

**Scenario**: SC-15.3 — A success neither counts nor clears
**Given**: R done. Order DOI-24818 unlinked (phone `0912 345 678`). Lan holds a cookie.
**When**: Four wrong-phone attempts for number `24818`; then the right pair (success); then a fifth wrong-phone attempt; then the right pair again.
**Then**: The four answer 404; the success answers 200; the fifth wrong attempt answers 404 (the success did not count as a try); the final right pair answers 429 (the success did not clear the four).
Traces: SC-15 ("a success neither counts nor clears")
Check: fetch_url

**Scenario**: SC-15.4 — Adding an order refuses no session (401) and a team session (403)
**Given**: Order DOI-24818 unlinked; Minh and the Admin hold cookies.
**When**: `POST /v1/account/orders` with the right pair and no cookie, then with Minh's, then with the Admin's.
**Then**: 401 `unauthenticated`; 403 `forbidden`; 403 `forbidden`; the order's `account_id` is still NULL.
Traces: SC-15 (ACC-006 AC-7)
Check: fetch_url

---

## 7. Backend — what a customer can never reach

**Scenario**: SC-16.1 — A customer session is refused on every dealer endpoint
**Given**: Lan holds a cookie.
**When**: With Lan's cookie: `GET /v1/orders`; `POST /v1/products` with `{"name":"X","price":1}`; `GET /v1/admin/overview`; `GET /v1/admin/staff`; `PUT /v1/orders/doi-24818` with `{"status":"confirmed"}`.
**Then**: Each answers 403 with `error` = `forbidden` and no data of any order, product or staff member. A route that is not built when this is run answers 404 and is reported UNVERIFIED for that route, never passed.
Traces: SC-16 (ACC-008 AC-1, item 1)
Check: fetch_url

**Scenario**: SC-16.2 — A customer who knows an order number but not its phone gets the generic miss
**Given**: R done. Lan's order `DOI-30001` is linked to Lan. Bao holds a cookie. No order `DOI-99999`.
**When**: As Bao: `POST /v1/order-lookups` with `{"number":"DOI-30001","phone":"0900 000 000"}`; the same body to `POST /v1/account/orders`; `POST /v1/order-lookups` with number `DOI-99999`.
**Then**: All three answer 404 `not_found` with byte-identical bodies.
Traces: SC-16 (ACC-008 AC-4)
Check: fetch_url

**Scenario**: SC-16.3 — No account route takes an account id, so one customer cannot name another
**Given**: Lan and Bao exist; Bao holds a cookie; Lan's user id is known; Lan has a saved city `Da Nang`.
**When**: As Bao: `GET /v1/account/<Lan's id>`; `GET /v1/account/orders?account=<Lan's id>&user=<Lan's id>`; `PUT /v1/account?id=<Lan's id>` with `{"city":"Hue"}`; then as Lan `GET /v1/account`.
**Then**: The first answers 404 `not_found`; the second answers 200 with only Bao's orders (none of Lan's); the third changes Bao's city only; Lan's city is still `Da Nang`.
Traces: SC-16 (ACC-008 AC-5, item 2)
Check: fetch_url

---

## 8. Backend — migration, performance and build

**Scenario**: SC-17.1 — The schema change is in place and existing orders are not linked
**Given**: The story running on a database that held orders before the story's migration.
**When**: Read `information_schema.columns`, `pg_constraint` and `pg_index` for `orders.account_id` and `idx_orders_account_id_placed_at` with `set_db_state`; count `orders` rows with `account_id IS NOT NULL` that were not linked by a case.
**Then**: `orders.account_id` exists, is nullable and has a foreign key to `users(id)` with `ON DELETE SET NULL`; `idx_orders_account_id_placed_at` exists and is valid (`indisvalid`); every order from before the story has `account_id` NULL.
Traces: SC-17 (migration)
Check: fetch_url (the story answering `GET /healthz` 200 after boot) with the schema read through `set_db_state`

**Scenario**: SC-17.2 — A second boot changes nothing
**Given**: The story running; `SELECT count(*) FROM schema_migrations` read and noted.
**When**: The backend is stopped and started again; then `GET /healthz` and the count read again.
**Then**: `/healthz` answers 200; the count of `schema_migrations` is the same as before; the story's own table and column set is the same.
Traces: SC-17 (second boot)
Check: fetch_url

**Scenario**: SC-17.3 — Deleting a customer leaves their orders in place with no account
**Given**: Lan has two orders linked to her; one of them has a token the case holds.
**When**: With `set_db_state`, `DELETE FROM users WHERE email = 'lan@example.com'`; then read both orders' `account_id`; then `GET /v1/orders/<id>` with the token, and with the Admin's cookie.
**Then**: The delete succeeds with no foreign-key error; both orders still exist with `account_id` NULL; the token read answers 200; the dealer read answers 200.
Traces: SC-17 (`ON DELETE SET NULL`)
Check: fetch_url

**Scenario**: SC-17.4 — `GET /v1/account/orders` answers within 300 ms at p95 for 100 orders
**Given**: Lan has 100 linked orders (inserted with `set_db_state`) and holds a cookie.
**When**: 100 sequential `GET /v1/account/orders` calls, the response time of each noted.
**Then**: The 95th-percentile time is at most 300 ms and every call answers 200 with `"total":100`. If `fetch_url` does not report a response time, the case reports UNVERIFIED and names that missing figure.
Traces: SC-17 (Performance NFR)
Check: fetch_url

**Scenario**: SC-17.5 — `go build`, `go vet` and `go test ./...` pass with no database
**Given**: The PR head, with `TEST_DATABASE_URL` unset.
**When**: The backend's build, vet and test commands run (the CI backend job, or `run_tests` with target `backend`).
**Then**: All three succeed; the database tests are reported as skipped, not failed.
Traces: SC-17 (last sentence)
Check: manual — none of the five instruments runs a build; the CI job and `run_tests` do, and their output is the evidence.

---

## 9. Backend — contract cases for the endpoints this story uses

**Scenario**: C-1 — Every non-2xx body has the shared shape and every answer carries `X-Request-Id`
**Given**: R done.
**When**: `POST /v1/account/sign-up` with an invalid body and the header `X-Request-Id: probe-12345678`; the same without the header; `GET /v1/account` with no cookie.
**Then**: The 422 body has `error` = `invalid`, a `message` sentence and `errors` as an object of field → sentence; the response header `X-Request-Id` is `probe-12345678` in the first call and 16 lower-case hex characters in the second; the 401 body has `error` = `unauthenticated`, a `message`, and no `errors` key. No body mentions SQL, a table name, a path or a stack.
Traces: contract (§2.1 trace header, §2.3 error contract)
Check: fetch_url

**Scenario**: C-2 — A failed query is a 500 `internal` with a generic message
**Given**: R done. Lan holds a cookie.
**When**: Rename `users` to `users_x` with `set_db_state` and call `POST /v1/account/sign-up` with a valid body; rename it back. Then rename `orders` to `orders_x` and call `GET /v1/account/orders` with Lan's cookie; rename it back.
**Then**: Both calls answer 500 with `error` = `internal`, a generic `message`, no `errors` key, and no table, column or SQL text in the body. After the tables are restored the same two calls answer 201 (new email) and 200.
Traces: contract (§2.3 `internal`; POST /v1/account/sign-up; GET /v1/account/orders)
Check: fetch_url

**Scenario**: C-3 — Signing out answers 204 always, clears the cookie and takes no body
**Given**: Lan holds a cookie.
**When**: `POST /v1/auth/sign-out` with Lan's cookie and no body; the same with no cookie; the same with the body `{"x":1}`.
**Then**: 204 with `Set-Cookie: session=` and `Max-Age=0` for the first; 204 for the second; 400 `bad_request` for the third.
Traces: contract (POST /v1/auth/sign-out)
Check: fetch_url

---

## 10. Browser — sign up (`/account/sign-up`)

**Scenario**: SC-18.1 — The sign-up page shows its words, fields and links inside the customer shell
**Given**: A signed-out visitor, 1280px window.
**When**: `/account/sign-up` is loaded.
**Then**: The page has the site's header and footer. `main h1` reads `Create your account`; the line `Optional — you can always buy without one.` is shown; four labelled fields `Name`, `Email`, `Phone`, `Password` (each label is visible text tied to its input); a button `Create account`; and under the form the text `Already have an account? Sign in` where `Sign in` is a link with `href` `/account/sign-in`.
Traces: SC-18 (ACC-001 AC-1, item 1)
Check: render_url

**Scenario**: SC-18.2 — The password is hidden and the fields carry the right input hints
**Given**: `/account/sign-up` loaded in a browser.
**When**: `abc12345` is typed into Password; the four inputs' attributes are inspected.
**Then**: The Password input has `type` `password` (the typed text is not shown); the inputs carry `autocomplete` values `name`, `email`, `tel` and `new-password` respectively; the Phone input has `inputmode` `tel` or `type` `tel` (a phone keypad on a phone).
Traces: SC-18 (last sentence)
Check: interact_page

**Scenario**: SC-18.3 — "Create account" is the filled pill of the page
**Given**: `/account/sign-up` at 1280px.
**When**: The computed style of `form button[type=submit]` is read, together with `--accent` and `--accent-ink` from `:root`.
**Then**: Its background colour equals the `--accent` token, its text colour equals `--accent-ink`, its border radius is at least half its height (a pill) and its height is at least 44px.
Traces: SC-18 ("the filled pill"); SC-39 (buttons at least 44px)
Check: measure_styles

**Scenario**: SC-19.1 — Pressing "Create account" with every field empty shows four sentences and sends nothing
**Given**: `/account/sign-up` loaded; the story's backend is **stopped**, so any request the page sent would end in "We cannot reach the server. Please try again."
**When**: "Create account" is pressed with all four fields empty.
**Then**: Under the fields the page shows exactly `Fill in your name`, `Enter a valid email address`, `Check the phone number` and `Use at least 8 characters`, each under its own field, and does not show `We cannot reach the server. Please try again.` (so no request was sent). The address stays `/account/sign-up`.
Traces: SC-19 (ACC-001 AC-6)
Check: interact_page

**Scenario**: SC-19.2 — Each bad value shows its own sentence under its own field
**Given**: `/account/sign-up` loaded; backend stopped as in SC-19.1.
**When**: Name `Lan Nguyen`, email `lan@`, phone `0912 34`, password `abc1234` are typed and "Create account" is pressed; then email is corrected to `lan@example.com` and the button pressed again.
**Then**: First press: `Enter a valid email address` under Email, `Check the phone number` under Phone and `Use at least 8 characters` under Password, and nothing under Name. After the email is corrected the email sentence is gone and the other two stay.
Traces: SC-19 (ACC-001 AC-7, AC-8)
Check: interact_page

**Scenario**: SC-19.3 — Each error sentence has an alert icon, the danger colour and an alert role
**Given**: The state of SC-19.1.
**When**: The `[role=alert]` elements are inspected and their computed colour read, with `--danger` read from `:root`.
**Then**: There are four `[role=alert]` elements (one per sentence), each with an icon element in front of the sentence, and each sentence's text colour equals the `--danger` token.
Traces: SC-19 (last sentence); SC-39 (errors announced)
Check: measure_styles

**Scenario**: SC-20.1 — A valid sign-up goes to `/account`, the header shows the name and the session survives a reload
**Given**: R done. No account for `lan@example.com`. `/account/sign-up` loaded.
**When**: `Lan Nguyen`, `lan@example.com`, `0912 345 678` and `lanpass123` are typed and "Create account" is pressed; then the page is reloaded.
**Then**: The address becomes `/account`; `header a[href="/account"]` reads `Lan Nguyen`; after the reload the address is still `/account` (not `/account/sign-in`) and the header still shows `Lan Nguyen`.
Traces: SC-20 (ACC-001 AC-2, item 3)
Check: interact_page

**Scenario**: SC-20.2 — An email already held by a customer keeps the form, shows the sentence under Email and empties the password
**Given**: R done. Lan exists. `/account/sign-up` loaded.
**When**: Name `Other Lan`, email `LAN@example.com`, phone `0912 345 678`, password `otherpass123` are typed and "Create account" is pressed.
**Then**: The address stays `/account/sign-up`. Directly under the Email field the page shows exactly `An account with this email already exists. Sign in instead.` as an `[role=alert]`. Name still holds `Other Lan`, Email `LAN@example.com`, Phone `0912 345 678`; Password is empty.
Traces: SC-20 (ACC-001 AC-4)
Check: interact_page

**Scenario**: SC-20.3 — An email held by a staff member or by the admin shows the same sentence
**Given**: R done. Minh (staff) and the Admin exist. `/account/sign-up` loaded.
**When**: Sign up with Minh's email and then (page reloaded) with the Admin's email, other fields valid.
**Then**: Each time the page stays on `/account/sign-up` and shows `An account with this email already exists. Sign in instead.` under Email, with the other typed values kept and Password emptied.
Traces: SC-20 (ACC-001 AC-5)
Check: interact_page

**Scenario**: SC-21.1 — While the shop answers slowly the button reads "Creating your account…" and is disabled
**Given**: R done. `/account/sign-up` loaded with valid values for a new email; the answer of `POST /v1/account/sign-up` is held for 3 seconds (see Slow answer).
**When**: "Create account" is pressed, then pressed again 1 second later, and the button is read at 1 second and after the answer arrives. A second run uses the email of an existing account so the answer is a refusal.
**Then**: At 1 second the button's text is `Creating your account…` and it is disabled; the second press sends no second request (`SELECT` of the `sign_up_address` counter with `set_db_state` shows one attempt for the run, not two). In the refusal run the button reads `Create account` again and is enabled once the answer arrives.
Traces: SC-21 (ACC-001 AC-9, item 5)
Check: interact_page

**Scenario**: SC-21.2 — When the shop cannot be reached the page says so and keeps everything but the password
**Given**: `/account/sign-up` loaded; values `Lan Nguyen`, `lan@example.com`, `0912 345 678`, `lanpass123` typed. Case (a): the backend is stopped (gateway 502). Case (b): backend running with `users` renamed to `users_x` so the call answers 500.
**When**: "Create account" is pressed, in each case.
**Then**: Each time the page shows exactly `We cannot reach the server. Please try again.`; Name, Email and Phone still hold their typed values; Password is empty; the button is enabled; the address stays `/account/sign-up`.
Traces: SC-21 (ACC-001 AC-10)
Check: interact_page

**Scenario**: SC-21.3 — After the limit the page shows the too-many-tries sentence and the button is usable
**Given**: R done, then ten sign-up attempts counted for the address the browser uses (ten calls made from the page, or the counter written with `set_db_state`). `/account/sign-up` loaded with valid values and a new email.
**When**: "Create account" is pressed.
**Then**: The page shows exactly `Too many tries. Please wait and try again.`; the button reads `Create account` and is enabled; no account exists for the email.
Traces: SC-21 (ACC-001 AC-11)
Check: interact_page

---

## 11. Browser — sign in (`/account/sign-in`)

**Scenario**: SC-22.1 — The sign-in page shows its words, fields and links inside the customer shell
**Given**: A signed-out visitor, 1280px.
**When**: `/account/sign-in` is loaded.
**Then**: The site's header and footer are present. `main h1` reads `Sign in`; fields labelled `Email` and `Password` (the password input has `type` `password`); a button `Sign in`; the text `New here? Create an account` where `Create an account` links to `/account/sign-up`; and the line `Buying as a guest needs no account.`
Traces: SC-22 (ACC-002 AC-1, item 1)
Check: render_url

**Scenario**: SC-22.2 — Enter submits from either field and blank fields show their sentence without sending
**Given**: `/account/sign-in` loaded; backend stopped (so a sent request would show "We cannot reach the server. Please try again.").
**When**: "Sign in" is pressed with both blank; then `lan@example.com` is typed in Email alone and Enter is pressed in that field; then Email is cleared, `x` typed in Password alone and Enter pressed there.
**Then**: First: `An email address is needed` under Email and `A password is needed` under Password. Second: `A password is needed` shows and `An email address is needed` does not. Third: `An email address is needed` shows. At no point does `We cannot reach the server. Please try again.` appear. Then with the backend running and Lan's right credentials typed, Enter in the Password field signs in (address `/account`).
Traces: SC-22 (ACC-002 item 1)
Check: interact_page

**Scenario**: SC-23.1 — A customer's right credentials lead to `/account`
**Given**: R done. Lan exists. `/account/sign-in` loaded.
**When**: `lan@example.com` and `lanpass123` are typed and "Sign in" is pressed.
**Then**: The address becomes `/account` and `main h1` reads `My account`.
Traces: SC-23 (ACC-002 AC-2, item 3)
Check: interact_page

**Scenario**: SC-23.2 — A staff member's and the admin's credentials lead to `/admin`, not through the dealer sign-in page
**Given**: R done. Minh (staff) and the Admin exist. `/account/sign-in` loaded.
**When**: Minh's credentials are used; then (new browser) the Admin's.
**Then**: Each time the address becomes `/admin`, the dealer overview shows (the admin shell is drawn and there is no password field on the page), and the address never was `/admin/sign-in` at the end of the navigation.
Traces: SC-23 (ACC-002 AC-6, item 4)
Check: interact_page

**Scenario**: SC-23.3 — A signed-in customer who opens either form is taken to `/account`
**Given**: Lan signed in in the browser.
**When**: `/account/sign-in` is opened; then `/account/sign-up` is opened.
**Then**: Each time the address becomes `/account` and, once settled, the page has no `input[type=password]` (the form is not left on screen).
Traces: SC-23 (ACC-001 AC-12, item 7; ACC-002 AC-7, item 5)
Check: interact_page

**Scenario**: SC-24.1 — A wrong password and an unknown email show the same sentence and nothing more
**Given**: R done. Lan exists. `/account/sign-in` loaded.
**When**: `lan@example.com` with `wrongpass123` is submitted; then (page reloaded) `nobody@example.com` with `wrongpass123`.
**Then**: Both times the page shows exactly `Wrong email or password.` in an `[role=alert]` element, with no other sentence about the email or the password; the Email field still holds what was typed and the Password field is empty; the address stays `/account/sign-in`.
Traces: SC-24 (ACC-002 AC-3)
Check: interact_page

**Scenario**: SC-24.2 — Signing in slowly shows "Signing in…", and an unreachable shop shows the server sentence
**Given**: Lan exists. `/account/sign-in` loaded with Lan's credentials typed. Case (a): the answer of `POST /v1/auth/sign-in` held for 3 seconds. Case (b): the backend stopped.
**When**: "Sign in" is pressed in each case.
**Then**: (a) At 1 second the button reads `Signing in…` and is disabled; when the answer arrives the address is `/account`. (b) The page shows exactly `We cannot reach the server. Please try again.` and the button reads `Sign in` and is enabled.
Traces: SC-24 (ACC-002 AC-5)
Check: interact_page

**Scenario**: SC-24.3 — A sixth try after five wrong ones shows the fifteen-minute sentence
**Given**: R done. Lan exists. `/account/sign-in` loaded.
**When**: Five wrong passwords for `lan@example.com` are submitted one after another; then `lanpass123` is submitted.
**Then**: The sixth submission shows exactly `Too many tries. Please wait 15 minutes.`; the address stays `/account/sign-in`; the button is enabled.
Traces: SC-24 (ACC-002 AC-4)
Check: interact_page

---

## 12. Browser — My account (`/account`)

**Scenario**: SC-25.1 — My account lists the customer's two orders, newest first, with number, date, total, status and link
**Given**: R done. Lan exists and has two orders linked to her account: the older placed at 09:00 on a day `D1`, the newer at 10:00 on a later day `D2` (set with `set_db_state`). Lan signed in in the browser. The browser window is 1280px.
**When**: `/account` is opened.
**Then**: `main h1` reads `My account`; the text `Hello, Lan Nguyen` is shown; the headings `My orders`, `Saved delivery details` and `Add an order` and a button `Sign out` are shown. The "My orders" block holds exactly two rows, the newer first; each row shows its order number (`DOI-…`), its date written `dd/mm/yyyy` in Vietnam time, its total in `vi-VN` money (the API's `total` formatted with dots and `₫`, for example `6.180.000 ₫`), its status in words (`Placed`, `Confirmed`, `Packed`, `On its way`, `Delivered` or `Cancelled` as the order's status says) and is a link whose `href` is `/orders/<id>`. No row contains Lan's phone, email or address text.
Traces: SC-25 (ACC-003 AC-1, items 1, 2)
Check: render_url (after sign-in through `interact_page`; the rows are read from the rendered page)

**Scenario**: SC-25.2 — The same orders show on another device after signing in
**Given**: Lan's two orders as in SC-25.1. A fresh browser with no cookies and no stored data.
**When**: In that browser, `/account/sign-in` is used with Lan's credentials.
**Then**: The address is `/account` and "My orders" lists the same two order numbers in the same order.
Traces: SC-25 (ACC-003 AC-2); SC-11 (ACC-005 AC-3)
Check: interact_page

**Scenario**: SC-26.1 — Twelve orders show ten rows and "Show more"; pressing it adds the last two
**Given**: Lan has twelve linked orders (see Twelve orders) and is signed in.
**When**: `/account` is opened; "Show more" is pressed.
**Then**: Before the press the block has ten rows (links to `/orders/<id>`) and a button `Show more`; after it twelve rows, the two new ones being the two oldest, and no `Show more` button.
Traces: SC-26 (ACC-003 AC-3)
Check: interact_page

**Scenario**: SC-26.2 — With no order the block says so and offers "Keep shopping"
**Given**: Lan has just signed up; no order is linked to her.
**When**: `/account` is opened.
**Then**: The "My orders" block shows `You have no orders yet.` and a pill `Keep shopping` whose `href` is `/shop`; there is no `Show more` button and no order row.
Traces: SC-26 (ACC-003 AC-4)
Check: render_url

**Scenario**: SC-27.1 — A row opens the Order page and shows the order with no token in the browser
**Given**: Lan signed in with one linked order `doi-30001`. The browser's `localStorage` and `sessionStorage` are cleared after signing in (the session cookie stays).
**When**: `/account` is opened and the row for `DOI-30001` is pressed.
**Then**: The address becomes `/orders/doi-30001`; the page shows the order number `DOI-30001`, the shoe's name, size and quantity, and the total; neither storage holds an entry whose value is an order token (no key or value containing a 22+ character URL-safe token for that order).
Traces: SC-27 (ACC-003 AC-5, item 4)
Check: interact_page

**Scenario**: SC-27.2 — Another customer opening the address sees "We can't find that order." only after the shop has answered
**Given**: Lan's order `doi-30001`. Bao signed in a fresh browser; no order token is held. The answer of `GET /v1/orders/doi-30001` is held for 3 seconds.
**When**: Bao opens `/orders/doi-30001`; the page text is read at 1 second and again after the answer has arrived.
**Then**: At 1 second the text `We can't find that order.` is not on the page (no not-found state while the order is being fetched). After the answer the page shows `We can't find that order.` and a pill `Track an order` whose `href` is `/track`; the order's number, shoe and total are not on the page.
Traces: SC-27 (ACC-003 AC-7; ACC-008 AC-3)
Check: interact_page

**Scenario**: SC-28.1 — With no session `/account` goes to the sign-in page
**Given**: A browser with no cookie.
**When**: `/account` is opened.
**Then**: The address becomes `/account/sign-in`; the sign-in form shows and no "My account" heading is drawn.
Traces: SC-28 (ACC-003 AC-10, item 5)
Check: interact_page

**Scenario**: SC-28.2 — While `/v1/auth/me` is awaited nothing of the page is drawn and nothing redirects yet
**Given**: Lan signed in. The answer of `GET /v1/auth/me` is held for 2 seconds. A second run with no cookie.
**When**: `/account` is opened; the page and the address are read at 1 second and again after the answer.
**Then**: At 1 second the address is still `/account`, and the page has no `h1`, no `Saved delivery details`, no `My orders`, no form field and no `Sign out` button. With Lan's cookie, after the answer the page is drawn; in the run with no cookie, after the answer the address becomes `/account/sign-in`.
Traces: SC-28 (ACC-003 item 5 second sentence)
Check: interact_page

**Scenario**: SC-28.3 — A staff or admin session on `/account` is taken to `/admin`
**Given**: Minh signed in the browser through `/admin/sign-in`; then the Admin in a new browser.
**When**: `/account` is opened in each.
**Then**: The address becomes `/admin` and no "My account" heading was left on screen.
Traces: SC-28 (decision 3)
Check: interact_page

**Scenario**: SC-29.1 — While the orders are awaited the block says "Loading your orders…" and the other blocks are usable
**Given**: Lan signed in with orders. The answer of `GET /v1/account/orders` is held for 3 seconds.
**When**: `/account` is opened; at 1 second the text of the "My orders" block is read and `Phone` in "Saved delivery details" is typed into.
**Then**: The "My orders" block shows `Loading your orders…`; the Phone input accepts the typed text; after 3 seconds the loading text is gone and the order rows show.
Traces: SC-29 (ACC-003 AC-11, item 6)
Check: interact_page

**Scenario**: SC-29.2 — When the orders cannot be fetched the block says so and "Try again" asks again
**Given**: Lan signed in with two orders. `orders` is renamed to `orders_x` with `set_db_state` before `/account` is opened, so `GET /v1/account/orders` answers 500 while `/v1/auth/me` still answers 200.
**When**: `/account` is opened; then the table is renamed back and `Try again` is pressed.
**Then**: The block shows `We could not load your orders.` and a button `Try again`; the other blocks are drawn. After the table is restored and the button pressed the message goes and the two order rows show.
Traces: SC-29 (ACC-003 AC-11, Upstream-failure row)
Check: interact_page

**Scenario**: SC-30.1 — The saved details of a new customer hold the sign-up values and empty address and city
**Given**: Lan has just signed up through the page (name `Lan Nguyen`, phone `0912 345 678`).
**When**: `/account` is opened.
**Then**: "Saved delivery details" has fields labelled `Name` (value `Lan Nguyen`), `Phone` (value `0912 345 678`), `Street address, ward, district` (empty) and `City or province` (empty), and a button `Save details`.
Traces: SC-30 (ACC-004 AC-1, item 1)
Check: render_url (after sign-up through `interact_page`; values read from the rendered inputs)

**Scenario**: SC-30.2 — Saving shows "Saving…" while the shop is slow, then "Saved.", and the values survive a reload
**Given**: Lan signed in on `/account`. The answer of `PUT /v1/account` is held for 2 seconds.
**When**: `12 Example Street, Ward 1` is typed into the address field and `Da Nang` into City; "Save details" is pressed; the button is read at 1 second and after the answer; the page is reloaded.
**Then**: At 1 second the button reads `Saving…` and is disabled; after the answer `Saved.` shows and the button reads `Save details` again; after the reload the two fields hold `12 Example Street, Ward 1` and `Da Nang`.
Traces: SC-30 (ACC-004 AC-2, AC-6, item 3)
Check: interact_page

**Scenario**: SC-30.3 — When the shop cannot be reached the block says it could not save and keeps the typed values
**Given**: Lan signed in on `/account` with `12 Example Street, Ward 1` and `Da Nang` typed but not saved; the backend is stopped.
**When**: "Save details" is pressed.
**Then**: The block shows exactly `We could not save. Please try again.`; the fields still hold the typed values; the button is enabled.
Traces: SC-30 (ACC-004 AC-7)
Check: interact_page

**Scenario**: SC-31.1 — The page checks the saved-details limits itself, under each field, and sends nothing
**Given**: Lan signed in on `/account` with `0912 345 678` saved; the backend is stopped (a sent request would show "We could not save. Please try again.").
**When**: Phone is set to `01234567`, the name is emptied, the address set to 201 characters, the city to 101 characters, and "Save details" is pressed.
**Then**: `Check the phone number` shows under Phone, `Fill in your name` under Name, `Use at most 200 characters` under the address and `Use at most 100 characters` under City, each as an `[role=alert]`; `We could not save. Please try again.` does not show. After the backend is started and the page reloaded the fields hold the values saved earlier (name `Lan Nguyen`, phone `0912 345 678`).
Traces: SC-31 (ACC-004 AC-4, AC-5, Invalid-input row)
Check: interact_page

**Scenario**: SC-31.2 — A refusal from the shop is shown under its field in the same way
**Given**: Lan signed in on `/account`. The answer of `PUT /v1/account` is replaced by `422 {"error":"invalid","message":"Check the form.","errors":{"phone":"Check the phone number"}}` (see Slow answer for the way a single answer is replaced with `interact_page`).
**When**: A valid phone is typed and "Save details" is pressed.
**Then**: The sentence from `errors.phone` shows under Phone as an `[role=alert]`; the values already saved are not lost (a reload shows them).
Traces: SC-31 ("When the shop refuses a field (422) its sentence from `errors` is shown the same way")
Check: interact_page (reported UNVERIFIED, naming the instrument, if answers cannot be replaced)

**Scenario**: SC-32.1 — The "Add an order" block shows its fields
**Given**: Lan signed in.
**When**: `/account` is opened.
**Then**: "Add an order" has a field labelled `Order number` whose placeholder is `DOI-24817`, a field labelled `Phone`, and a button `Add order`.
Traces: SC-32 (ACC-006 AC-1, item 1)
Check: render_url

**Scenario**: SC-32.2 — Adding a guest order shows "Added." and the list redraws with it without a reload
**Given**: R done. Order DOI-24818 (phone `0912 345 678`, unlinked). Lan signed in on `/account`; a marker `window.__probe = 1` is set so a page reload would be detected.
**When**: `doi 24818` is typed in `Order number` and `+84912345678` in the add-an-order `Phone` and "Add order" is pressed.
**Then**: `Added.` shows; "My orders" now has a row for `DOI-24818` with no reload (`window.__probe` is still 1); the order's `account_id` is Lan's.
Traces: SC-32 (ACC-006 AC-1, item 2)
Check: interact_page

**Scenario**: SC-32.3 — Adding the same order again shows "Added." and one row
**Given**: The state at the end of SC-32.2.
**When**: The same two values are typed and "Add order" is pressed.
**Then**: `Added.` shows and "My orders" still has exactly one row for `DOI-24818`.
Traces: SC-32 (ACC-006 AC-5)
Check: interact_page

**Scenario**: SC-33.1 — Every kind of miss shows one alert, with "message us" linking to the contact page
**Given**: R done. Order DOI-24818 unlinked; order DOI-24820 linked to Bao (phone `0987 654 321`). Lan signed in on `/account`.
**When**: In "Add an order": number `24818` with phone `0912 345 679` (wrong); number `99999` with any phone; number `24818` with an empty phone; number `24820` with Bao's right phone — pressing "Add order" each time.
**Then**: Each time the block shows an `[role=alert]` with exactly `No order matches that number and phone. Check both, or message us.` in which `message us` is a link with `href` `/contact`; the list of Lan's orders does not change.
Traces: SC-33 (ACC-006 AC-2, AC-3, AC-8, item 3)
Check: interact_page

**Scenario**: SC-33.2 — After five failures a sixth, even with the right pair, shows the too-many-tries sentence
**Given**: R done. Order DOI-24818 unlinked (phone `0912 345 678`). Lan signed in on `/account`.
**When**: Five wrong-phone attempts for `24818` are made in the block; then the right pair is entered.
**Then**: The sixth attempt shows exactly `Too many tries. Please wait 15 minutes, or message us with your order number.` and the order is not added.
Traces: SC-33 (ACC-006 AC-4, item 4)
Check: interact_page

**Scenario**: SC-33.3 — When the shop cannot be reached the block says it could not add the order
**Given**: Lan signed in on `/account`; Order DOI-24818's number and phone typed; the backend is stopped.
**When**: "Add order" is pressed.
**Then**: The block shows exactly `We could not add that order. Please try again.`; the typed values stay; the button is enabled.
Traces: SC-33 (ACC-006 Upstream-failure row)
Check: interact_page

**Scenario**: SC-33.4 — While an add request is in flight the button is disabled
**Given**: Lan signed in; the answer of `POST /v1/account/orders` held for 3 seconds; the right pair typed.
**When**: "Add order" is pressed and the button is read at 1 second; pressed a second time at 1 second.
**Then**: The `Add order` button is disabled at 1 second; the second press sends no second request (one failure or one success is recorded for the pair, not two).
Traces: SC-33 ("While a request is in flight the button is disabled")
Check: interact_page

**Scenario**: SC-34.1 — "Sign out" ends the session, goes to the home page and the header says "Sign in"
**Given**: Lan signed in on `/account`.
**When**: `Sign out` is pressed; then `/account` is opened.
**Then**: After the press the address is `/`; the header entry reads `Sign in`; `GET /v1/auth/me` from the browser answers 401; opening `/account` takes the address to `/account/sign-in`.
Traces: SC-34 (ACC-007 item 3, AC-6)
Check: interact_page

**Scenario**: SC-34.2 — The old cookie opens nothing after sign-out
**Given**: Lan signs in with `POST /v1/auth/sign-in` and keeps the cookie `c1`, then `POST /v1/auth/sign-out` is sent with `c1`.
**When**: `GET /v1/auth/me`, `GET /v1/account` and `GET /v1/account/orders` are sent with `c1`.
**Then**: All three answer 401 `unauthenticated`; a `SELECT count(*) FROM sessions` for Lan's user shows the row is gone.
Traces: SC-34 (ACC-007 AC-7)
Check: fetch_url

---

## 13. Browser — the header's "Account" entry

**Scenario**: SC-35.1 — Signed out, the header shows "Sign in" after the five links, beside search and bag
**Given**: A signed-out visitor, 1280px window.
**When**: `/`, `/shop` and `/contact` are loaded.
**Then**: On each page the header has exactly one `Account` entry: a link with text `Sign in` and `href` `/account/sign-in`, placed after the header's five existing links in document order and on the same row as the search and bag buttons (its vertical centre lies within the header bar and left of the search button).
Traces: SC-35 (ACC-007 AC-1, item 1; SITE-001 AC-1)
Check: render_url (text, `href` and order); measure_styles for the row position of `header a[href="/account/sign-in"]` against the search and bag buttons

**Scenario**: SC-35.2 — Signed in, the entry shows the person's name and goes to the right place
**Given**: Three browsers: Lan (customer `Lan Nguyen`), Minh (staff, name `Minh`) and the Admin, each signed in.
**When**: `/` and `/shop` are loaded in each.
**Then**: Lan's header entry reads `Lan Nguyen` with `href` `/account`; Minh's reads `Minh` with `href` `/admin`; the Admin's reads their name with `href` `/admin`. No entry reads `Sign in` while signed in.
Traces: SC-35 (ACC-007 AC-2, AC-3)
Check: interact_page

**Scenario**: SC-35.3 — A long name is cut with an ellipsis and the search and bag buttons stay on the bar
**Given**: A customer whose name is 100 characters (a letter `W` repeated, with a space every 10) signed in; 1280px window.
**When**: `/` is loaded; the computed style and bounding box of the entry, the search button and the bag button are read.
**Then**: The entry's `text-overflow` is `ellipsis` and `overflow` is `hidden` (its text is wider than its box); the search and bag buttons' right edges are at most 1280px and on the same row as the entry; the page has no horizontal scroll.
Traces: SC-35 ("A name of 100 characters is cut so the search and bag buttons stay on the bar")
Check: measure_styles

**Scenario**: SC-35.4 — On its own address the entry is underlined and marked as the current page
**Given**: A signed-out visitor, then Lan signed in; 1280px.
**When**: `/account/sign-in` is loaded signed out; `/account` is loaded as Lan; the entry's attributes and `text-decoration` are read.
**Then**: Signed out on `/account/sign-in`, the entry has `aria-current="page"` and an underline; signed in as Lan on `/account`, the entry has `aria-current="page"` and an underline. On `/shop` neither has `aria-current`.
Traces: SC-35 (last but one sentence)
Check: interact_page (attributes); measure_styles (`text-decoration-line` of the entry)

**Scenario**: SC-35.5 — At 390px the bar is unchanged and the open menu lists the five links, the entry, then the help links
**Given**: A 390×844 window. Run three times: signed out, signed in as Lan, signed in as Minh.
**When**: `/` is loaded; the bar's buttons are read; the menu button is pressed; the menu's links are read in order.
**Then**: The bar holds only the search, bag and menu buttons (no `Account` entry in the bar). The open menu lists, in this order, the five header links, then one entry (`Sign in` → `/account/sign-in`; `Lan Nguyen` → `/account`; `Minh` → `/admin`), then `Size chart`, `Delivery`, `Return policy`, `Track an order`, `Contact`.
Traces: SC-35 (ACC-007 AC-4; SITE-001 AC-3)
Check: interact_page

**Scenario**: SC-36.1 — A forged value in the browser's storage does not sign anyone in
**Given**: A signed-out browser. Before load, `localStorage` is set to `agentteam:session` = `{"name":"Fake Admin","role":"admin"}`, plus keys `session`, `user` and `role` with similar values; `sessionStorage` gets the same.
**When**: `/` is loaded; then `/admin` is opened.
**Then**: The header entry reads `Sign in` and the text `Fake Admin` is nowhere on the page; `/admin` does not show the overview (the address becomes `/admin/sign-in`).
Traces: SC-36 (ACC-007 AC-8, item 4)
Check: interact_page

**Scenario**: SC-36.2 — While `/v1/auth/me` is awaited the entry shows neither a name nor "Sign in"
**Given**: The answer of `GET /v1/auth/me` is held for 2 seconds. Run once signed out and once as Lan.
**When**: `/` is loaded; the header is read at 1 second and again after the answer.
**Then**: At 1 second the header contains neither the text `Sign in` nor `Lan Nguyen` (the entry is not drawn, or is empty); after the answer it shows `Sign in` (signed out) or `Lan Nguyen` (Lan).
Traces: SC-36 (ACC-007 AC-9)
Check: interact_page

**Scenario**: SC-36.3 — When `/v1/auth/me` answers 401 or cannot be reached the entry reads "Sign in" and no error is drawn
**Given**: A browser holding Lan's cookie. Case (a): the session row is deleted with `set_db_state` (so `me` answers 401). Case (b): the backend is stopped.
**When**: `/` is loaded in each case.
**Then**: The header entry reads `Sign in` with `href` `/account/sign-in`; no `[role=alert]` and no error sentence is in the header or at the top of the page.
Traces: SC-36 (ACC-007 Upstream-failure row)
Check: interact_page

**Scenario**: SC-36.4 — For anyone not signed in as admin or staff, the only link to `/admin` on a customer page is the footer's "Dealer sign-in"
**Given**: A signed-out browser, then Lan signed in.
**When**: For `/`, `/shop`, `/cart`, `/checkout`, `/track`, `/contact`, `/account/sign-in`, `/account/sign-up` (and `/account` as Lan), at 1280px and at 390px with the menu opened, every `a[href^="/admin"]` is listed.
**Then**: On every page, in both states, the list holds exactly one link: inside the footer, with text `Dealer sign-in`.
Traces: SC-36 (ADMIN-001 AC-13; ACC-007 AC-5, item 2)
Check: interact_page

---

## 14. Browser — checkout with and without an account

**Scenario**: SC-37.1 — A signed-in customer finds the saved details filled in, all editable, the note empty
**Given**: Lan has saved name `Lan Nguyen`, phone `0912 345 678`, address `12 Example Street, Ward 1`, city `Da Nang`; she is signed in with one pair of the Shoe in the bag.
**When**: `/checkout` is opened; then `Full name` is changed to `Gift Receiver` and the other three fields edited.
**Then**: Before editing: `Full name` holds `Lan Nguyen`, `Phone` `0912 345 678`, `Street address, ward, district` `12 Example Street, Ward 1`, `City or province` `Da Nang`; the note field is empty. Each field accepts the edit and shows the new value.
Traces: SC-37 (ACC-005 AC-1, item 1)
Check: interact_page

**Scenario**: SC-37.2 — A field with no saved value is empty
**Given**: Lan has just signed up (no saved address or city), signed in, one pair in the bag.
**When**: `/checkout` is opened.
**Then**: `Full name` holds `Lan Nguyen` and `Phone` holds `0912 345 678`; `Street address, ward, district` and `City or province` are empty.
Traces: SC-37 (ACC-005 Absent-data row)
Check: interact_page

**Scenario**: SC-37.3 — A value already typed is never replaced by a saved value that arrives late
**Given**: Lan has saved details; signed in with a pair in the bag; the answer of `GET /v1/account` is held for 3 seconds.
**When**: `/checkout` is opened; at 1 second `Gift Receiver` is typed into `Full name`; the page is read after the answer has arrived.
**Then**: `Full name` still holds `Gift Receiver`; the three untouched fields now hold the saved phone, address and city.
Traces: SC-37 ("never replaced by a saved value that arrives late")
Check: interact_page

**Scenario**: SC-37.4 — Placing an order with different details leaves the saved details unchanged
**Given**: Lan signed in; a pair in the bag; `/checkout` open with her saved details; the run can place an order (VNPay keys set; otherwise this case reports UNVERIFIED and names the missing keys).
**When**: The four fields are changed to `Gift Receiver`, `0900 000 111`, `99 Other Road`, `Hue` and "Place order" is pressed; then `GET /v1/account` is read.
**Then**: The order is placed (the page moves on to the order or to payment); `GET /v1/account` still answers name `Lan Nguyen`, phone `0912 345 678`, address `12 Example Street, Ward 1`, city `Da Nang`; the order appears in Lan's "My orders".
Traces: SC-37 (last sentence); SC-11 (ACC-005 AC-2)
Check: interact_page (then fetch_url for `GET /v1/account`)

**Scenario**: SC-38.1 — A guest buys with no sign-in prompt, the order is tracked, and it is in nobody's account
**Given**: A signed-out browser.
**When**: A pair of the Shoe is added to the bag; the bag, `/checkout` and (after placing) the order page are viewed; the order is then tracked on `/track` with its number and phone; `GET /v1/account/orders` is asked as Lan.
**Then**: No page asks the visitor to sign in or create an account (no text `Sign in` anywhere except the header entry; no redirect to `/account/sign-in`); the order page opens with its token; `/track` finds the order; the order is not in Lan's list. Where the run cannot place an order (no VNPay keys) the placing step is UNVERIFIED and the order is inserted for the tracking step.
Traces: SC-38 (ACC-005 AC-4, item 3; ACC-001 item 1)
Check: interact_page

**Scenario**: SC-38.2 — If the session ends before "Place order" the order is placed as a guest, not refused
**Given**: Lan signed in with a pair in the bag and `/checkout` open with the details filled in. Her session row is then deleted with `set_db_state` (the same as signing out in another tab).
**When**: "Place order" is pressed.
**Then**: The order is placed (no error, no redirect to a sign-in page) and its page opens with its token; `orders.account_id` of the new order is NULL; it is not in Lan's list after she signs in again.
Traces: SC-38 (ACC-005 AC-6, item 4)
Check: interact_page (UNVERIFIED for the placing itself if the run has no VNPay keys)

---

## 15. Browser — phone width, labels, keyboard and the team-only message

**Scenario**: SC-39.1 — The three pages are one column with no horizontal scroll at 390px and 320px, also with long values
**Given**: Windows of 390×844 and 320×640. For `/account` Lan has a name of 100 characters and an address of 200 characters saved and is signed in.
**When**: `/account/sign-up`, `/account/sign-in` and `/account` are loaded at each width; the document's `scrollWidth`, `clientWidth` and the left edge of every form field are read; the long values are typed into Name and the address field.
**Then**: On each page at each width `document.documentElement.scrollWidth` is at most the window width; all fields of a form have the same left edge and none sits beside another (one column); the long values do not widen the page.
Traces: SC-39 (ACC-003 AC-12, item 7; Responsive NFR)
Check: interact_page

**Scenario**: SC-39.2 — Every field has a visible linked label and every message is announced
**Given**: The three pages loaded; each message is provoked: a field error on sign-up (empty submit), `Wrong email or password.` on sign-in, `Saved.` after saving details, `Added.` after adding an order.
**When**: For each input, its `<label for>` or wrapping label is read; for each message, its `role` / `aria-live` is read.
**Then**: Every input on the three pages has a label with visible text that is linked to it (clicking the label focuses the input); every one of the sentences above sits in an element with `role="alert"` or `role="status"` (or `aria-live` `assertive` / `polite`), present when the sentence appears.
Traces: SC-39 (Accessibility NFR)
Check: interact_page

**Scenario**: SC-39.3 — The three pages work with the keyboard alone and focus is visible
**Given**: `/account/sign-up`, `/account/sign-in` and `/account` loaded, no pointer used.
**When**: Tab is pressed from the top of each page, noting the order of the focused elements; on sign-up and sign-in the fields are filled with the keyboard and Enter is pressed in the last field; the focused element's outline and box-shadow are inspected at each stop.
**Then**: Tab visits the fields and buttons in the order they appear on the page (sign-up: Name, Email, Phone, Password, `Create account`, `Sign in` link); Enter submits (sign-up and sign-in reach `/account`); at every stop the focused element has a visible indicator (an `outline-style` that is not `none` with a width above 0, or a non-empty `box-shadow`).
Traces: SC-39 (Accessibility NFR; ACC-001 and ACC-002 "forms work with Enter and the keyboard alone")
Check: interact_page

**Scenario**: SC-39.4 — The pages' buttons are at least 44px high
**Given**: `/account/sign-up`, `/account/sign-in` and `/account` (with twelve orders so `Show more` is drawn, and with the orders failing so `Try again` is drawn in a second run) at 390px.
**When**: The computed height of `Create account`, `Sign in`, `Save details`, `Add order`, `Show more`, `Try again` and `Sign out` is read.
**Then**: Each is at least 44px high.
Traces: SC-39 ("the page's buttons are at least 44px high")
Check: measure_styles

**Scenario**: SC-40.1 — A customer on `/admin` sees only the team-only message
**Given**: Lan has just signed up (customer session).
**When**: `/admin` is opened; then `/admin/orders`; the page text, its links and its network requests are read.
**Then**: Both pages show `This area is for the shop's team` and a link with `href` `/`; no admin menu, table, figure or form is drawn; no request to a dealer data endpoint (`/v1/orders`, `/v1/admin/overview`, `/v1/admin/staff`, `/v1/products` writes) is made — where `interact_page` cannot report requests this last part is UNVERIFIED and the absence of any dealer figure on the page is the evidence.
Traces: SC-40 (ACC-008 AC-2, item 1)
Check: interact_page

---

## 16. Coverage — every acceptance criterion and the cases that serve it

| Criterion | Cases |
|---|---|
| SC-1 | SC-1.1, SC-1.2, SC-1.3, SC-1.4, SC-1.5 |
| SC-2 | SC-2.1, SC-2.2, SC-2.3, SC-2.4 |
| SC-3 | SC-3.1, SC-3.2, SC-3.3, SC-3.4, SC-3.5, SC-3.6, SC-3.7, SC-3.8 |
| SC-4 | SC-4.1, SC-4.2, SC-4.3, SC-4.4, SC-4.5 |
| SC-5 | SC-5.1, SC-5.2, SC-5.3, SC-5.4, SC-5.5 |
| SC-6 | SC-6.1, SC-6.2, SC-6.3 |
| SC-7 | SC-7.1, SC-7.2, SC-7.3, SC-7.4, SC-7.5, SC-7.6, SC-7.7 |
| SC-8 | SC-8.1, SC-8.2, SC-8.3, SC-8.4, SC-8.5 |
| SC-9 | SC-9.1, SC-9.2, SC-9.3 |
| SC-10 | SC-10.1, SC-10.2, SC-10.3 |
| SC-11 | SC-11.1, SC-11.2, SC-11.3 |
| SC-12 | SC-12.1, SC-12.2 |
| SC-13 | SC-13.1, SC-13.2, SC-13.3, SC-13.4 |
| SC-14 | SC-14.1, SC-14.2 |
| SC-15 | SC-15.1, SC-15.2, SC-15.3, SC-15.4 |
| SC-16 | SC-16.1, SC-16.2, SC-16.3 |
| SC-17 | SC-17.1, SC-17.2, SC-17.3, SC-17.4, SC-17.5 |
| SC-18 | SC-18.1, SC-18.2, SC-18.3 |
| SC-19 | SC-19.1, SC-19.2, SC-19.3 |
| SC-20 | SC-20.1, SC-20.2, SC-20.3 |
| SC-21 | SC-21.1, SC-21.2, SC-21.3 |
| SC-22 | SC-22.1, SC-22.2 |
| SC-23 | SC-23.1, SC-23.2, SC-23.3 |
| SC-24 | SC-24.1, SC-24.2, SC-24.3 |
| SC-25 | SC-25.1, SC-25.2 |
| SC-26 | SC-26.1, SC-26.2 |
| SC-27 | SC-27.1, SC-27.2 |
| SC-28 | SC-28.1, SC-28.2, SC-28.3 |
| SC-29 | SC-29.1, SC-29.2 |
| SC-30 | SC-30.1, SC-30.2, SC-30.3 |
| SC-31 | SC-31.1, SC-31.2 |
| SC-32 | SC-32.1, SC-32.2, SC-32.3 |
| SC-33 | SC-33.1, SC-33.2, SC-33.3, SC-33.4 |
| SC-34 | SC-34.1, SC-34.2 |
| SC-35 | SC-35.1, SC-35.2, SC-35.3, SC-35.4, SC-35.5 |
| SC-36 | SC-36.1, SC-36.2, SC-36.3, SC-36.4 |
| SC-37 | SC-37.1, SC-37.2, SC-37.3, SC-37.4 |
| SC-38 | SC-38.1, SC-38.2 |
| SC-39 | SC-39.1, SC-39.2, SC-39.3, SC-39.4 |
| SC-40 | SC-40.1 |
| contract | C-1, C-2, C-3 (and the contract rows the SC cases above also trace: 400 for an undefined field, 415, 401/403, 404 with one body, 429 with `Retry-After`, 500 `internal`) |

All 40 acceptance criteria have at least one case. Criteria that name a failure have a case in which the failure happens: SC-2 (taken email), SC-3 (every refused value), SC-4 and SC-15 (the lock), SC-5 and SC-24 (wrong credentials, lock, unreachable), SC-21, SC-24, SC-29, SC-30, SC-33 (the shop slow, down or failing), SC-12 and SC-38 (the session ended), SC-9, SC-10, SC-27 and SC-16 (another customer's data), SC-16 and SC-40 (a customer on dealer endpoints).

Every role the SRS names is covered, allowed and denied: visitor (SC-6.2, SC-7.7, SC-8.5, SC-9.3, SC-15.4, SC-28.1, SC-38), customer (everywhere), another customer (SC-9.2, SC-10.2, SC-14.1, SC-16.2, SC-16.3, SC-27.2), shop team (SC-5.4, SC-6.2, SC-7.7, SC-8.5, SC-9.3, SC-15.4, SC-23.2, SC-28.3, SC-35.2).

## 17. Checks that are not plain observations, and findings

**Manual (2 of 144 cases):**
- SC-1.5 — the password and hash are never logged. No instrument reads the backend's log output.
- SC-17.5 — `go build` / `go vet` / `go test ./...` with no database. None of the five instruments runs a build; CI's backend job and `run_tests` do.

**Cases that depend on an arrangement the verifying role may not have, and are then reported UNVERIFIED naming the missing instrument, never passed from reading code:**
- A slow answer (SC-21.1, SC-24.2, SC-27.2, SC-28.2, SC-29.1, SC-30.2, SC-33.4, SC-36.2, SC-37.3) needs `interact_page` to hold one request for a stated time; SC-31.2 needs it to replace one answer.
- Placing a real order (SC-11, SC-12, SC-37.4, SC-38) needs the VNPay keys to be set in the run, because `POST /v1/orders` answers 503 `payment_unavailable` without them (`services.md` §3.10). The account-side half of each case is checked with an order inserted by `set_db_state`.
- SC-17.4 needs `fetch_url` to report response times.

**Findings against the documents (nothing was resolved silently):**
1. **Where an error sentence sits.** The SRS says "beside the field"; the story (UI scope, decision 1) puts it under the field in the design system's alert style. These cases follow the story ("under").
2. **A team session on `/account`.** The SRS says only that the account endpoints answer 403 to a team session; the story's decision 3 sends the page to `/admin`. SC-28.3 follows the story. The SRS should say so.
3. **An unreachable database.** `services.md` §2.3 makes "a database that cannot be reached" a 500 `internal`, while the verification guidance reserves 503 for a refused dependency. No case arranges an unreachable database; C-2 arranges a failed query (a table that is gone), which is 500 under both readings.
4. **`limit` above 100 on `GET /v1/account/orders`.** `services.md` §3.20 lists an invalid `limit` as 400, while §2.4 serves a `limit` above 100 as 100. SC-8.3 follows §2.4 (200, `"limit":100`) and 400 only for `0`, negative and non-numeric values.
5. **An `id` in the body of `PUT /v1/account`.** §2.5 tolerates a body `id` only on collection `PUT`s; for `/v1/account`, which "takes no id", the contract is silent. No case sends it.
6. **A 200-character email.** The contract allows up to 200 characters and a shape of one `@`, a dot in the domain and no spaces; a parser that follows RFC 5321 would refuse a local part over 64 characters. SC-3.3 expects the contract (accepted).
7. **Staff rows.** The staff-accounts story is built in parallel, so a staff member is made by promoting a signed-up customer with `set_db_state`; the admin is the environment's own `ADMIN_EMAIL` (a second admin row is refused by the table's partial unique index).
8. **Copy at the API.** The contract fixes the 422 field names but not their sentences; the BE cases assert the key set and a non-empty sentence (and the number of the limit where the SRS names it). The exact words are asserted only where the SRS or the story writes them (the browser cases).
