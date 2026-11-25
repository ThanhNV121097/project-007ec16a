# Service contracts — Staff accounts managed by the owner (plan item 14, "S13")

Module `admin` · Requirements ADMIN-030, ADMIN-031, ADMIN-004 (the account-endpoint half) · Story `docs/admin/stories/staff-accounts-managed-by-the-owner.md` · UI PR #36 · Tables: `docs/architecture/erd/staff-accounts-managed-by-the-owner.md` (none changed).

This file defines **the four endpoints this story ships, the order in which each refuses a request, the functions `internal/auth` must offer them, the two statements of "Dealer sign-in" this story relies on, and the frontend's swap from its mock** — and nothing else. It reuses the project's error envelope and closed error catalog (`services/_base.md` §2.3), its list scheme (§2.4), validation boundary (§2.5) and roles (§2.2). It adds **no error code to the catalog**, no second list or error shape, no table and no setting. Every path is written the way the Go backend mounts it — **no `/api` prefix**: the browser's `/api/v1/admin/staff` reaches Go as `/v1/admin/staff`.

## 1. Endpoints

| # | Method and path | Auth | Traces to |
|---|---|---|---|
| 3.26 | `GET /v1/admin/staff` | role `admin` only | ADMIN-030 items 1, 4, 5 |
| 3.26 | `POST /v1/admin/staff` | role `admin` only | ADMIN-030 items 2–5 |
| 3.27 | `PUT /v1/admin/staff/{id}` | role `admin` only | ADMIN-031 items 1, 2, 3, 5 |
| 3.27 | `DELETE /v1/admin/staff/{id}` | role `admin` only | ADMIN-031 items 1, 4, 5 |

**The contracts of `_base.md` §3.26 and §3.27 stand.** This file adds the order of checks, every sentence, every code, and five points where those sections are silent (§4.5). It changes none of their statuses, fields or codes.

**Mounting.** The four routes are four literal patterns registered by one function, `admin.RegisterStaff(mux, deps)`, in `internal/admin/staff.go` (the ownership table of `_base.md` §1.1 puts the staff handlers in `admin`, over the functions of `auth`). `server.New` gains **one line** calling it; `internal/admin` has no other file this story edits, so "Manage shoes, lines, categories, reviews and orders" (which also adds files to `internal/admin`, in parallel) collides with nothing. Each handler is wrapped `auth.Require("admin")`. Any other method on these paths (for example `GET /v1/admin/staff/{id}`) falls through to the server's catch-all and answers `404 not_found` (project contract §3, "no wildcard"); no handler is written for it. **No new package, setting, secret or `.env.example` line.**

**On every answer of all four routes:** `Content-Type: application/json; charset=utf-8` (except `204`, which has no body), `Cache-Control: no-store`. `201` carries no `Location`.

## 2. The mock, read and judged

The UI PR adds `src/lib/mock/staff.ts`, imported only by `src/lib/staff.ts` (checked at the UI review). It is unreviewed input; its shape is compared with the contract. **Where `_base.md` §2.3, §2.4, §3.26 and §3.27 decided a shape, they win.**

| Mock | Contract | Judgment | Frontend change at the swap |
|---|---|---|---|
| List answers `{items,total,page,limit:100}`; an item is `id`, `name`, `email`, `role` (`"admin"` \| `"staff"`), `is_active` | `_base.md` §3.26: the same five names and types | **Sound.** Wire names, types and the envelope match field for field | None to the types |
| `useStaff` keeps `items` only; `StaffAccounts` passes `total={items.length}` to `DataTable` | The answer's `total` is the count of all matches (§2.4) | **Too thin**, and the UI review already asked for it (should-fix 1) | `useStaff` also returns the answer's `total`; `AdminStaff` hands it to `StaffAccounts`, which passes it to `DataTable` |
| The admin's row is named "Shop owner" | The admin made from settings has **no stored name** (`users.name = ''`) | **A mock that hides a real case.** A row with an empty name cell would show at SC-1 | None in the frontend. The API gives the same display name `/v1/auth/me` gives (decision S6): `auth.DisplayName`, §5 |
| `mockCreate` / `mockUpdate` throw `MockFault(422, {error:"validation_failed", message, errors})` | `422` is `{"error":"invalid","message":"Check the form.","errors":{…}}` | The page reads only the status and `errors`, never the code | `failure()` reads `ApiError.status === 422 && ApiError.errors` (§7) |
| 409 for the admin row: `{error:"conflict", message:"The admin account cannot be switched off, changed or removed."}` | The same code and the same sentence | **Sound** (the sentence is the SRS's) | None |
| 404 for an unknown id: `{error:"not_found", message:"That account was not found."}` | `404 not_found`, one body for an unknown id, a customer's id and a non-uuid id | **Sound, and the sentence is kept** (decision S9): a dialog that says "Not found." tells a person nothing | None |
| No answer when `navigator.onLine` is false (`MockFault(0, {error:"network"})`) | `api.ts` throws `ApiError` with `status` 0 and `code` `network` for no connection or the 15-second timeout | **Same idea, other type** | `failure()` treats `status` 0 and every 5xx as `unreachable`; `MockFault` goes with the mock |
| After a refusal the hook asks the list again so the table is "the shop's answer" | Nothing in the contract forbids it; a `GET` is free of side effects | **Sound** (ADMIN-031 item 6) | None |
| `mockRemove` resolves with nothing | `DELETE` answers `204` with **no body** | The real client must not parse JSON from an empty answer | `api.ts`'s `delete` resolves `void` on `204` (§7) |
| The mock accepts `{}` for `mockUpdate` as "nothing to change" and `id` as an ignored extra | `{}` is a no-op `200`; `id` in the body is a tolerated extra (§4.5) | Same behaviour | None |
| The mock adds a created account at the end of the list | The list is ordered admin first, then by creation (`created_at`, `id`) | Same order | None |

**Nothing else changes in the frontend**: no page, no component beyond the `total` pass-through, no `content.json` key. The field and refusal sentences come from the API (§4.4) and are shown as they arrive.

## 3. Order of checks

Each step runs only when the earlier ones passed; a request is answered by **the first step that refuses it**. Steps 1–2 are common to all four routes.

| # | Check | Refusal |
|---|---|---|
| 1 | `auth.Resolve` has found a session | `401 unauthenticated` |
| 2 | The session's role is `admin` (re-read from `users` on this request, so a staff member just switched off is a `401`, not a `403`) | `403 forbidden` for `staff` and `customer`. **Nothing of the request has been read yet**: no body, no id, no query other than the session's |

### 3.1 `GET /v1/admin/staff`

| # | Check | Refusal |
|---|---|---|
| 3 | `page` and `limit` as `_base.md` §2.4 (`page` ≥ 1 default 1, `limit` ≥ 1 default 12, above 100 served as 100); any other parameter is ignored | `400 bad_request`, `errors.page` / `errors.limit` |
| 4 | One `REPEATABLE READ, READ ONLY` transaction: the count, then the page (ERD §3 #1) | `500 internal` |

### 3.2 `POST /v1/admin/staff`

| # | Check | Refusal |
|---|---|---|
| 3 | `Content-Type` is `application/json` (a charset parameter is fine) | `415 unsupported_media_type` |
| 4 | The body is at most **64 KiB** (`http.MaxBytesReader`) | `413 payload_too_large` |
| 5 | The body is one JSON object (no second value after it) whose keys are among `name`, `email`, `password` and whose values are strings. A key outside that set — **`role` included** — and a value of another type (a number, `null`, a list) are refused, naming the field | `400 bad_request`, `errors.<field>` (§4.4) |
| 6 | **Every** field is judged and every fault collected (§4.4): name, email (shape, then whether any account holds it), password | `422 invalid`, `errors` naming each field at fault, **a taken email included** |
| 7 | The password is hashed (`auth.HashPassword`); no transaction is open | — |
| 8 | One `INSERT` (ERD §3 #4). `23505` on `uq_users_email` is the race of S2 | `422 invalid`, `errors.email` "That email already belongs to an account." Any other database error: `500 internal` |
| — | Success | `201` the account (§4.1) |

### 3.3 `PUT /v1/admin/staff/{id}`

| # | Check | Refusal |
|---|---|---|
| 3 | `{id}` is a canonical uuid (`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`) | `404 not_found`, **no query made** |
| 4 | `auth.Target(id)`: the row exists and its role is not `customer` | `404 not_found` for no row and for a customer — **the same bytes** as step 3's |
| 5 | The target's role is not `admin` | `409 conflict`, "The admin account cannot be switched off, changed or removed." — **before the body is read**, so an admin id with an unparseable, wrong-typed or `role`-carrying body is still `409` |
| 6 | `Content-Type` `application/json`; body at most 64 KiB; one JSON object | `415`; `413`; `400` (an empty body is `400`; `{}` is accepted) |
| 7 | Keys are among `name`, `is_active`, `password`, **`role`** (read so it can be refused) and `id` (read and discarded, §4.5); every other key — **`email` included** — and a wrong type for `name` (not a string), `password` (not a string) or `is_active` (not a boolean; `null` is a wrong type) are refused naming the field | `400 bad_request`, `errors.<field>` — wins over every `422` below, because a body the contract does not define is judged before its values |
| 8 | Each field sent is judged and every fault collected: `name` (trimmed, 1–100, §4.4), `password` (10–100 characters), and **`role`, whatever its value or type** | `422 invalid`, `errors.name` / `errors.password` / `errors.role` ("The role cannot be changed."). Nothing is changed |
| 9 | `password`, if sent, is hashed (no transaction is open). `{}`: stop here and answer `200` with the target of step 4 | — |
| 10 | One transaction (`READ COMMITTED`): `auth.Target(id, lock)` (`FOR UPDATE`) — gone meanwhile → `404`; role `admin` → `409` (cannot happen; the same code path guards it); then `SetName`, `SetActive`, `SetPassword` for the fields sent, in that order (§5: `SetActive(false)` and `SetPassword` end the account's sessions inside the call); `COMMIT`. A deadlock (`40P01`) retries the whole transaction once | `404`; `500 internal` |
| — | Success | `200` the account as saved (§4.2) |

### 3.4 `DELETE /v1/admin/staff/{id}`

| # | Check | Refusal |
|---|---|---|
| 3–5 | As `PUT`: canonical uuid → `404`; `auth.Target` → `404` for none or a customer; role `admin` → `409` with the same sentence. A request body, if any, is neither read nor limited | as `PUT` |
| 6 | One statement, `auth.DeleteStaff`: `DELETE FROM users WHERE id = $1 AND role = 'staff'`. The sessions go by `CASCADE`; `uploads.uploaded_by` and `orders.account_id` by `SET NULL` | zero rows (removed between steps 4 and 6) `404`; `500 internal` |
| — | Success | `204`, no body. A second `DELETE` of the same id is `404` (project idempotency table §2.6) |

## 4. Contracts

### 4.1 The account, and the two answers that carry it

An account is **exactly five keys**, in every answer that holds one:

```json
{ "id": "0b1c7a52-1f0e-4a52-9d57-0f0d3c1a0002", "name": "Minh Nguyen", "email": "minh@example.com", "role": "staff", "is_active": true }
```

`id` is a canonical lower-case uuid string; `role` is `"admin"` or `"staff"` (never `"customer"`: a customer is never listed or returned); `is_active` is a JSON boolean; `name` is the **display name** (decision S6: the stored name, or the email's local part with its words capitalised when none is stored — `auth.DisplayName`). The answer is written from a struct of these five fields, not from a row or a map, so no password, hash or other column can reach a body by accident (SC-11).

**`GET /v1/admin/staff`** → `200 {"items":[<account>…],"total":n,"page":p,"limit":l}`. The accounts whose role is `admin` or `staff`: **the admin first**, then by creation (`ORDER BY (role <> 'admin'), created_at, id` — unique, so two reads give one order). `total` is the number of all such accounts. The page sends `?limit=100`; with no parameter the project default is 12.

**`POST /v1/admin/staff`** — body `{"name","email","password"}`, all three required strings:

| Field | Rule |
|---|---|
| `name` | trimmed, 1–100 characters, no control character; stored trimmed |
| `email` | `auth.NormalizeEmail` (decision S7): trimmed, ASCII only, lower-cased, one `@`, a dot in the domain, no space or control character, ≤ 200 characters; belongs to **no** account of any role. Stored lower-case |
| `password` | 10–100 **characters** (not bytes), **never trimmed**, stored as an argon2id hash, never returned, shown or logged |

→ `201` the account, `role: "staff"`, `is_active: true`. It can sign in at once.

### 4.2 `PUT /v1/admin/staff/{id}`

Body: any of `name` (as above), `is_active` (boolean), `password` (10–100 characters, as above); a field left out is unchanged; the fields sent are applied together or not at all (one transaction). `role` is judged and refused (`422`); `id` is discarded; any other key is `400`. → `200` the account **as saved**.

| Sent | Effect on sessions |
|---|---|
| `is_active: false` | The account's `sessions` rows are deleted in the same transaction (sent again for an account already off: still deleted, harmlessly) |
| `password` | Deleted in the same transaction, even when the new password equals the old one |
| `is_active: true` | Nothing is restored; the rows deleted earlier are gone (SC-29) |
| `name` alone, or `{}` | Nothing ends (SC-38) |

### 4.3 `DELETE /v1/admin/staff/{id}`

→ `204`. The account goes; its sessions go with it (`CASCADE`); the email can be added again at once, as a new account with a new id; no earlier session is valid for it. Nothing else changes (SC-33).

### 4.4 Errors, and every sentence

The body of every refusal is the project envelope (`{"error","message"}`, plus `errors` **only** on `400` and `422`). Messages this story owns:

| Code | HTTP | Trigger | `message` |
|---|---|---|---|
| `unauthenticated` / `forbidden` | 401 / 403 | Steps 1–2. Owned by `auth.Require` ("Dealer sign-in"); this story adds no wording | the guard's |
| `bad_request` | 400 | Not a JSON object, trailing data, an empty body, an undefined key, a wrong type | "The request could not be read." |
| `payload_too_large` | 413 | Body over 64 KiB | "The request is too large." |
| `unsupported_media_type` | 415 | Not `application/json` | "Send the request as JSON." |
| `not_found` | 404 | Non-uuid id, no such id, a customer's id — **one body for all three** | "That account was not found." |
| `conflict` | 409 | The target's role is `admin` (PUT, DELETE) | "The admin account cannot be switched off, changed or removed." (exactly these words — SRS ADMIN-031 item 5; no `errors`) |
| `invalid` | 422 | A rule below; `errors` holds **every** field at fault | "Check the form." |
| `internal` | 500 | Any failed query, insert, hash or transaction, a deadlock that survived its one retry | "Something went wrong. Please try again." — nothing else: no SQL, no id, no email, no hash |

`errors` sentences (English; the frontend shows them as they arrive, tests assert the field name and that a sentence is there — wording may change):

| Field | Status | Condition | Sentence |
|---|---|---|---|
| `name` | 422 | missing (POST), empty or only spaces | "A name is needed" |
| `name` | 422 | over 100 characters after trimming | "The name can have at most 100 characters." |
| `name` | 422 | contains a control character (NUL included — PostgreSQL cannot store it) | "The name cannot contain control characters." |
| `email` | 422 | missing, empty or only spaces | "An email address is needed" |
| `email` | 422 | not an address by `NormalizeEmail`, or over 200 characters | "That does not look like an email address." |
| `email` | 422 | held by any account, any case (pre-check or the `23505` race) | "That email already belongs to an account." |
| `password` | 422 | missing, or under 10 characters (an empty string included) | "The password needs at least 10 characters." |
| `password` | 422 | over 100 characters | "The password can have at most 100 characters." |
| `role` | 422 | `PUT` carries it, any value, any type | "The role cannot be changed." |
| any key outside the allowed set (`role` on `POST`, `email` on `PUT` included) | 400 | — | "This field is not accepted." |
| `name`, `email`, `password` | 400 | not a string (a number, `null`, a list, an object) | "This must be text." |
| `is_active` | 400 | not a boolean | "This must be true or false." |

### 4.5 Five points where `_base.md` §3.26 and §3.27 are silent, decided here

1. **The admin row's name.** The list shows the display name, not the stored `''` (decision S6).
2. **The shape of `{id}`.** `_base.md` §2.5 matches path segments against a slug pattern, which a uuid does not fit; §3.27 says "a uuid; anything else is `404`". The id must be a canonical lower-case uuid; an upper-case, braced or hyphen-less spelling is `404` (PostgreSQL would accept them; the API never issues them). No query is made for a bad id, and PostgreSQL's `22P02` can never reach a body as a `500`.
3. **`PUT {}` and `id` in the body.** `{}` is a valid partial update that changes nothing: `200` with the account as it is (decision S10). `id` in the body is one of the two tolerated extras of `_base.md` §2.5 ("`id` in the body of a `PUT`"): read and discarded, any value; the path's id is the one acted on.
4. **A taken email beside other faults.** The `422` names every field at fault in one answer (SC-18), so the email is looked up even when `name` or `password` already failed; the password is hashed only when nothing failed (decision S11).
5. **Messages** for 400, 404, 413, 415, 422 and 500 are the sentences above, because the frontend shows the shop's `message` of a refusal in the open dialog (SC-39).

## 5. What `internal/auth` offers, and what this story relies on in "Dealer sign-in"

### 5.1 The functions (the ownership table of `_base.md` §1.1 names the package; this fixes the surface this story calls)

`internal/admin` contains **no statement on `users` or `sessions`**; it calls these. `Q` is the pool or a `pgx.Tx`.

```go
type Account struct{ ID, Name, Email, Role string; Active bool }
func (a Account) DisplayName() string          // §3.7: stored name, else email local part, words split on . _ - capitalised

var ErrNotFound   = errors.New("account not found")
var ErrEmailTaken = errors.New("email already belongs to an account")

func NormalizeEmail(raw string) (email string, ok bool)     // pure; decision S7; also used by sign-in and sign-up
func HashPassword(password string) (string, error)          // argon2id, no database; the hasher sign-in verifies against
func Team(ctx context.Context, tx pgx.Tx, limit, offset int) ([]Account, int, error)   // ERD §3 #1
func Target(ctx context.Context, q Q, id string, lock bool) (Account, error)           // ErrNotFound: no row or a customer; the admin row IS returned (the handler answers 409); lock adds FOR UPDATE
func EmailTaken(ctx context.Context, q Q, email string) (bool, error)
func CreateUser(ctx context.Context, q Q, u NewUser) (Account, error)                  // NewUser{Name, Email, PasswordHash, Role}; ErrEmailTaken only for SQLSTATE 23505 on constraint uq_users_email
func SetName(ctx context.Context, tx pgx.Tx, id, name string) error                    // UPDATE … WHERE id = $1 AND role = 'staff'; 0 rows → ErrNotFound
func SetActive(ctx context.Context, tx pgx.Tx, id string, active bool) error           // same guard; active == false also DELETE FROM sessions WHERE user_id = $1
func SetPassword(ctx context.Context, tx pgx.Tx, id, hash string) error                // same guard; also DELETE FROM sessions WHERE user_id = $1
func DeleteStaff(ctx context.Context, q Q, id string) error                            // DELETE … WHERE id = $1 AND role = 'staff'; 0 rows → ErrNotFound
```

`SetActive(false)` and `SetPassword` end the sessions **themselves**, so no caller — this story's handler, a later one, a test — can switch an account off or reset it and forget to. The `UPDATE` runs first and the `DELETE FROM sessions` second, in that order, so the row lock is taken before the sessions are read (ERD §4). A violation of any constraint other than `uq_users_email` is returned wrapped and is `500`; **the logged error carries the SQLSTATE and the constraint name only, never `PgError.Detail`**, which holds the row — the email for `23505`, and for a `CHECK` the whole row **including the password hash**.

### 5.2 Three statements of "Dealer sign-in" this story depends on

Item 9's backend lands first. Its review checks these; if one is not as written when this story's backend PR starts, **this PR edits that one statement** and says so in its description (the item-9 tests that exist must still pass).

| # | Where | Required | Why this story needs it |
|---|---|---|---|
| R1 | `auth.Resolve` | `… FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = $1 AND s.expires_at > now() AND u.is_active` — every request, no cache | A switched-off account's cookie answers `401` at once (SC-27); a role change or removal does too |
| R2 | `auth.SignIn`, the insert of the session, after the password has been verified against the hash `$3` read from the row `$1` | `INSERT INTO sessions (user_id, token_hash, expires_at) SELECT u.id, $2, now() + interval '7 days' FROM users u WHERE u.id = $1 AND u.is_active AND u.password_hash = $3 FOR SHARE OF u` — **zero rows inserted is answered exactly as a wrong password** (`invalid_credentials`, counted as a failure) | Closes the two races of ERD §4: a switch-off or a password reset that commits between "verified" and "inserted" can no longer leave a live session behind (SC-30) |
| R3 | `auth.EnsureAdmin` | Its `INSERT … ON CONFLICT` / `UPDATE` of the admin row writes `is_active = true` | An admin row can never be left off by a stray flag: there is no endpoint that could switch it on again |

## 6. Code seams and tests (what Dev builds)

**Files** (names are Dev's; each stays under 300 lines): `internal/admin/staff.go` (`RegisterStaff`, the four handlers, the retry-once on `40P01`); `internal/admin/staff_input.go` (decode to `map[string]json.RawMessage` so a missing key, `null` and a wrong type are told apart; the allowed-key sets; the rules of §4.4 — **pure**, no database); `internal/auth/accounts.go` (§5.1) and `normalize.go` if `NormalizeEmail` does not exist yet. The handler maps `auth.ErrNotFound` → the one `404` body, `auth.ErrEmailTaken` → the `422`, anything else → `500` with the cause logged against `X-Request-Id`.

**Logging.** The access log records method, route and status only — never a header, a cookie or a body. A `500` logs the request id, the operation (`staff.create`, `staff.update`, `staff.delete`, `staff.list`) and the error with SQLSTATE and constraint name. **Never** the password, the hash, the email typed, a request body or a pgx `Detail` (SC-21).

**Targets.** `GET` and `DELETE` p95 ≤ 300 ms; `POST` and a `PUT` carrying a password p95 ≤ 1 s (the argon2id hash is 50–150 ms and the only slow step); a `PUT` without a password ≤ 300 ms.

**Tests** (`go test ./...` passes with no database; DB cases read `TEST_DATABASE_URL` and skip without it):

| # | Test | Proves |
|---|---|---|
| T1 | Table test over the pure validators: name 0/1/100/101 characters, only spaces, NUL, a tab; email 200/201 characters, upper-case, a non-ASCII letter, no `@`, two `@`, no dot, a space; password 9/10/100/101 characters **counted in characters** (a 10-character Vietnamese password is accepted, a 101-rune one refused), leading and trailing spaces kept | SC-15 to SC-17, S7 |
| T2 | Handler tests over a fake `auth`: the guard answers before anything is read (a bad body sent as staff is `403`); a customer's id with a bad body is `404`; the admin's id with an invalid, empty or `role`-carrying body is `409`; `PUT` with `role` is `422` naming `role`, `POST` with `role` is `400`, `PUT` with `email` is `400`, `PUT` with `email` and `role` together is `400`; `{}` is `200`; `id` in the body is ignored; `null` is `400`; no response body of any status contains `argon2` or a key named like a password | SC-9, SC-34 to SC-38, SC-11 |
| T3 | DB: add, list order (admin first, then creation, ties by id), `total` with `page=2`, role `customer` never listed, the admin's display name, the email stored lower-case, two equal passwords → two different hashes that both begin `$argon2id$` | SC-11, SC-13, SC-21 |
| T4 | DB: two concurrent `POST`s of one email, repeated 50 rounds: one `201` and one `422` naming `email` each round, one row; a fake store that returns `23505` for another constraint name is `500` | SC-20, S2 |
| T5 | DB: `PUT` of `is_active:false` and of `password` delete the account's sessions; `name` alone and `is_active:true` delete none; `DELETE` removes the user, the sessions and frees the email; an `uploads` row (when the table exists) keeps its row with `uploaded_by` `NULL`; the counts of orders, shoes, lines, categories and reviews do not move | SC-24, SC-26, SC-29, SC-32, SC-33 |
| T6 | DB race: 100 rounds of one goroutine signing in (R2) while another switches the account off, and 100 rounds of sign-in against a password reset and against a removal; after every round `SELECT count(*) FROM sessions s JOIN users u ON u.id = s.user_id WHERE NOT u.is_active` is 0, and after a reset no session row predates it | SC-30, ERD §4 |
| T7 | `TestStaffSchemaAssumptions` (ERD §6) | the schema this story stands on |
| T8 | Log capture around a `POST`, a password `PUT` and a forced `500` (a store that fails with a `CHECK` error carrying a `Detail`): the captured output holds no password, no `$argon2id`, no typed email and no `Failing row` text | SC-21 |

## 7. The frontend's swap from its mock (Dev, backend step)

| Change | File |
|---|---|
| Four calls become `api.get("admin/staff", { limit: 100 })`, `api.post("admin/staff", body)`, `api.put(\`admin/staff/${id}\`, patch)`, `api.delete(\`admin/staff/${id}\`)` — paths as `api.ts` takes them (`${base}/v1/<path>`, as `api.get("products")` in the search story), no `/v1`, no leading slash | `src/lib/staff.ts` |
| `failure(e)` reads `ApiError`: `status === 422 && errors` → fields; any other 4xx → `refused` with `e.message`; `status` 0, any 5xx, a timeout → `unreachable`. `MockFault` is gone. A `401` or `403` (the admin's own session ended mid-visit) is a `refused` with the guard's sentence: no new state; the gate sends the admin to sign-in on the next navigation | `src/lib/staff.ts` |
| `useStaff` returns the answer's `total`; `AdminStaff` passes it to `StaffAccounts`, which passes it to `DataTable` instead of `items.length` (UI review should-fix 1). It keeps asking `limit=100` and draws what it gets (story §3) | `src/lib/staff.ts`, `src/pages/AdminStaff.tsx`, `src/components/StaffAccounts.tsx` |
| `api.ts` gains `post`, `put` and `delete` **if they are not there yet** (the checkout story may add `post` first; one client, never a second file). They send `Content-Type: application/json` (the API answers `415` to anything else) and are **never retried**; `delete` resolves with nothing on `204` instead of parsing an empty body | `src/lib/api.ts` |
| Delete the mock | `src/lib/mock/staff.ts` |
| `src/test/staff.test.tsx` mocks `../lib/staff` and needs no change except the one test that signs in with the current mock session API: whichever of PR #34 and PR #36 merges second adapts it to the real session (UI review should-fix 2) | `src/test/staff.test.tsx` |
| List `/admin/staff` with `"roles": ["admin"]` (`"auth": true, "dashboard": true`) so the walk covers the page and the role-leak check sees it (UI review should-fix 3; TL, at this PR's merge) | `design/pages.json` |

`Page`, the components, `content.json` and the routes are untouched; the 12 UI tests keep passing against the real hook's shape.

## 8. Decisions made in this design (S1–S8 are in the ERD file)

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| S9 | What does the one `404` say? | "That account was not found." — the mock's sentence, kept, for an unknown id, a customer's id and a non-uuid id alike | The generic "Not found.": it is what a person would read in the Switch-off question when someone else removed the account first. A different sentence per cause: it would tell the admin which ids are customers |
| S10 | What is `PUT {}`? Is `id` in the body refused? | `{}` is a no-op `200` (partial update, "a field left out is unchanged"); `id` is discarded as `_base.md` §2.5 allows for every `PUT` | `{}` as `400`/`422`: no field to name. `id` as `400`: the one tolerated extra of §2.5 would then differ from route to route |
| S11 | When is the password hashed? | **After** every rule passed and **before** any transaction: invalid requests cost no hashing, and no row lock is held while 19 MiB of argon2id runs | Hashing first (simpler code): every `422` would cost 100 ms and a lock-free but memory-heavy step for nothing. Hashing inside the transaction: holds the user's row lock for the hash |
| S12 | Rate limit or audit log on these routes? | **Neither.** They are admin-only behind a session; the SRS asks for no log of who added or removed whom (story §3) | A limit: nothing to protect from. An audit table: S1 in the ERD |
| S13 | Is the target looked up before the body is read? | **Yes** (an unlocked lookup, then a locked re-check inside the transaction): the SRS orders `409` before the body and a test sends `{"role":"admin"}` to a customer's id expecting `404`. A body is never read while a row lock is held | Reading the body first: the admin's id with a bad body would answer `400`, not `409`. Locking first and then reading the body: a slow client holds the lock |

### Conflicts with merged decisions

None. Refinements of silence only: §4.5 points 1–5 and the display-name and ASCII-email decisions of the ERD file. `_base.md` §3.26 and §3.27 are quoted, not edited; the merged `erd.md` and `services.md` are untouched (they are regenerated from the per-story files when this design merges).

## 9. Hand-over

| Who | What they must know from this design |
|---|---|
| Dev, backend step | Four routes, the functions of §5.1, the swap of §7, T1–T8. **No migration.** Check R1–R3 first and edit what is missing |
| Test lead | SC-9/SC-10 are guard-first: a bad body or a bad id as staff or with no session is `403` / `401`, never `404`/`400`. SC-37: `PUT {"role":"admin"}` to a customer's id is `404` (target before body); to the admin's id `409`. SC-36: `POST` with `role` is `400`, `PUT` with `role` is `422`. The 404 sentence is "That account was not found." |
| "Dealer sign-in" (item 9) | R1, R2, R3 and the shared `NormalizeEmail` are checked at its backend review |
| "Customer accounts" (item 15) | Reuses `NormalizeEmail`, `HashPassword`, `CreateUser` and `DisplayName` from §5.1 rather than writing a second copy |
