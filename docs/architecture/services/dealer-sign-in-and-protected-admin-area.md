# Service contracts — Dealer sign-in and protected admin area (plan item 9, "S8")

Module `admin` (package `internal/auth`, helper `internal/ratelimit`) · Requirements ADMIN-001 to ADMIN-004 · Story `docs/admin/stories/dealer-sign-in-and-protected-admin-area.md` · Test cases `docs/admin/test-cases/dealer-sign-in-and-protected-admin-area.md` · UI PR #34 · Tables: `docs/architecture/erd/dealer-sign-in-and-protected-admin-area.md`.

This file **extends** `services/_base.md`; it does not redesign it. The three endpoints (`_base.md` §3.7 `POST /v1/auth/sign-in`, §3.8 `POST /v1/auth/sign-out`, §3.9 `GET /v1/auth/me`), the session and cookie rules (§2.2), the error catalog (§2.3: `bad_request`, `unauthenticated`, `invalid_credentials`, `forbidden`, `invalid`, `rate_limited`, `internal`, `unsupported_media_type`, `payload_too_large`), the limits (§2.7) and the role table are **fixed and are not repeated**. Every path is written the way the Go backend mounts it — **no `/api` prefix**; the browser's `/api/v1/auth/me` reaches Go as `/v1/auth/me`. A failed query is `internal` (500), never `unavailable`. This file adds **no error code**, no second error shape and no second list format.

What this file decides, because `_base.md` left it open: the exact answers (§1), what the mock got right and wrong (§2), the Go surface other stories call (§3), the order of checks and the limits (§4), the start-up admin (§5), how `GET /v1/orders` becomes team-only whichever of two stories merges first (§6), every change to code that already exists (§7), the decisions with their rejected alternatives (§8), the test-case ambiguities answered (§9) and the BE review checklist (§10).

## 1. Endpoints and their exact answers

| # | Method and path | Auth | Traces to |
|---|---|---|---|
| 3.7 | `POST /v1/auth/sign-in` | none; limited by `sign_in_email` and `sign_in_address` | ADMIN-001, -002, -003 |
| 3.8 | `POST /v1/auth/sign-out` | none required | ADMIN-002 items 1, 6; AC-10, -11, -14 |
| 3.9 | `GET /v1/auth/me` | a session (any role) | ADMIN-002 items 1–2, 8–10; AC-2 to AC-4, -13 |

Every route is a literal pattern. A wrong method on these paths falls to the catch-all and answers `404 not_found` in the envelope (the server registers `"/"`), never a plain-text 405. Every answer below carries `Content-Type: application/json; charset=utf-8` (except `204`) and `Cache-Control: no-store`; the three bodies never contain an id, a hash, a token, the typed password or, on an error, the typed email.

### 1.1 `POST /v1/auth/sign-in`

Request, fields and constraints: `_base.md` §3.7 (`email` non-empty after trimming and ≤ 200 characters, `password` non-empty and ≤ 100 characters, never trimmed). Characters are counted in runes.

| Outcome | Status | Body | Notes |
|---|---|---|---|
| Signed in (any of the three roles) | `200` | `{"email":"dealer@example.com","name":"Dealer","role":"admin"}` | Exactly these three keys, `email` as stored (lower case). `Set-Cookie: session=<token>; Path=/; Max-Age=604800; HttpOnly; SameSite=Lax` plus `Secure` when the request came over HTTPS (§3.5). Exactly one `Set-Cookie` |
| `Content-Type` is not `application/json` (parameters such as `; charset=utf-8` allowed) | `415` | `unsupported_media_type` — "Send the request as JSON." | Nothing is counted |
| Not JSON, a wrong type (`"email": 5`), a field the contract does not define, trailing data after the object, or a body over 64 KiB | `400` `bad_request` (`413 payload_too_large` for the size) | `errors` names the field when it is one | Nothing is counted |
| `email` blank / over 200 / `password` blank / over 100 | `422` `invalid` — "Check the form." | `errors.email`: "An email address is needed" / "That email address is too long"; `errors.password`: "A password is needed" / "That password is too long". Both keys appear when both are at fault | **Not a failed try** and no password hash is computed (SC-8; answers the ambiguity in the test cases, §9 item 2) |
| Unknown email · wrong password · switched-off account · any email while no admin exists | `401` `invalid_credentials` | `{"error":"invalid_credentials","message":"Wrong email or password."}` — byte for byte the same in all four | One argon2id verification ran in every case (§4); the failure is counted (§4.2) |
| Email or address locked | `429` `rate_limited` | "Too many tries. Please wait N minutes." (N = the wait in minutes, rounded up, at least 1) and header `Retry-After: <whole seconds, 1–900>` | A refused call neither counts nor extends the lock; the right password does not help. The sign-in page shows its own fixed sentence for any 429 |
| Database failure | `500` `internal` | "Something went wrong. Please try again." | The page shows "We cannot reach the server" for any 5xx |

### 1.2 `POST /v1/auth/sign-out`

No body: any request body (`Content-Length` above 0, or a chunked body with one byte) is `400 bad_request`, nothing is deleted and the cookie is not cleared. Otherwise `204`, always — with a live session, an unknown one, an expired one, or no cookie. With a well-formed `session` cookie the matching `sessions` row is deleted (Q4); the answer clears the cookie: `Set-Cookie: session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax` (+ `Secure` as above). `500 internal` only when the delete itself fails (then the cookie is **not** cleared, so the browser still holds a token the server may still honour and the person can try again).

### 1.3 `GET /v1/auth/me`

`200 {"email","name","role"}` (`role` is `admin`, `staff` or `customer`) from the context `auth.Resolve` filled, with no second query. `401 unauthenticated` — "Sign in to continue." — for no cookie, a malformed one, an unknown token, an expired session (`expires_at <= now()`), and a switched-off or deleted account; the body is the same for all of them. `500 internal` when the session lookup itself fails (the failure is `Resolve`'s, §3.2).

## 2. The mock, read and judged

`src/lib/mock/dealer-sign-in-and-protected-admin-area.ts` (PR #34) is imported only by `src/lib/session.ts`. Read in full; it is unreviewed input and its shape is compared with §1.

| Mock | Contract | Judgment | Frontend change at the swap |
|---|---|---|---|
| `Answer = {status, body}`; `postSignIn(email, password)` returns `200 {email,name,role}` | Same three keys, same names, same types | Sound | `session.ts` stops reading `answer.status` and reads the thrown `ApiError` instead (table below) |
| `401 {error:"invalid_credentials", message:"Wrong email or password."}` | Byte for byte the same | Sound | None |
| `429 {error:"rate_limited", message:"Too many tries. Please wait 15 minutes."}`, no header | Message carries the real wait; `Retry-After` header | The mock's fixed 15 minutes is wrong when 3 minutes remain. The page never shows the server's sentence for a 429 (it has its own, SC-29), so nothing visible changes | None |
| `down@example.com` throws a `TypeError` | A refused connection throws inside `fetch`, which `api.ts` turns into `ApiError(0, "network")`; a 5xx is an `ApiError` too | Sound | `unreachable` is now "status 0 or ≥ 500" |
| `getMe()` → `200 {email,name,role}` or `401 {error:"unauthenticated"}` | Same | Sound | None |
| `postSignOut()` → `204`, body `null`, 400 ms latency | `204`, no body | Sound | `api.post` must accept an empty `204` |
| The sample accounts, the per-email failure map, `LATENCY_MS` | Replaced by the database | Test scaffolding | The mock file is **deleted** with its sample accounts; `src/test/dealer-sign-in.test.tsx` stubs `api` (`vi.mock("../lib/api")`) instead of the mock path and keeps every case |

**Swap, exactly.** `session.ts` keeps `useAuth`, `useSession`, `useRole`, `signIn`, `signOut`, `allowed`, `Session`, `SignInResult` and the `epoch` guard unchanged; `Role` (`"admin" | "staff" | "customer"`) moves into `session.ts` from the mock. Only the three calls change:

| Call | New body | Mapping |
|---|---|---|
| `check()` (was `getMe`) | `api.get<{email,name,role}>("auth/me")` | Resolves → `toSession(body)`; **any** rejection (401, 5xx, network) → nobody, as `check()`'s `catch` already does (SC-38) |
| `signIn` (was `postSignIn`) | `api.post<{email,name,role}>("auth/sign-in", {email, password})` | Resolves → `toSession` then `"ok"`; `ApiError` status `429` → `"limited"`; `401` or `422` → `"wrong"`; status `0` or `≥ 500` → `"unreachable"`; a 200 whose body is not an account → `"unreachable"` |
| `signOut` (was `postSignOut`) | `api.post("auth/sign-out")` with no body | Every rejection is swallowed, as today |

**`src/lib/api.ts` exports only `get` today** (PR #39). This story needs `api.post<T>(path, body?)`: `credentials: "include"`, `Content-Type: application/json` only when a body is given, the same 15 s timeout, the same `ApiError` mapping, **never retried** (a POST is not idempotent), and a `204` resolves `undefined` instead of failing (today `send` treats an empty body as a failure, which is right for a GET). If the checkout or order-page backend PR has already added a `post`, reuse it and extend only for the `204`; otherwise this PR adds it as an additive export. Whichever merges second keeps one `post`.

Nothing else in the frontend changes: no `Admin*.tsx`, no `DashboardLayout.tsx`, no `RequireRole.tsx`, no `App.tsx` line (SC-39). `credentials: "include"` on `api.get` already carries the cookie to every existing data call.

## 3. The Go surface (`internal/auth`)

Package layout: `auth.go` (types, context, `Resolve`, `Require`), `signin.go` (the three handlers and `Register`), `password.go` (hash, verify, dummy), `session.go` (token, cookie, Q3/Q4), `admin.go` (`EnsureAdmin`), `purge.go`, `email.go`, with `*_test.go` beside each. `internal/ratelimit` is its own package (§4.1) — `auth` calls it; nothing else writes `rate_limits`.

### 3.1 Types and what other stories call

```go
type Role string
const (Admin Role = "admin"; Staff Role = "staff"; Customer Role = "customer")

type User struct { ID, Email, Name string; Role Role }   // Name is the stored name, possibly ""
func (u User) DisplayName() string                       // §3.4

func FromContext(ctx context.Context) (User, bool)       // what Resolve found
func Require(roles ...Role) httpx.Middleware             // 401 / 403 / pass
func NormalizeEmail(raw string) (string, bool)           // the one rule for every write to users.email
func HashPassword(plain string) (string, error)          // argon2id, outside any transaction
func CreateUser(ctx, q Querier, n NewUser) (User, error) // NewUser{Email, Name, Phone, Address, City, Role, PasswordHash}
var ErrEmailTaken error                                  // CreateUser's translation of 23505 on uq_users_email
func EnsureAdmin(ctx, db Beginner, email, password string, log *slog.Logger) error
func RunPurge(ctx context.Context, db Querier, log *slog.Logger)        // blocks until ctx is done
func New(db Querier, log *slog.Logger) *Service          // holds the dummy hash and the hash slots
func (s *Service) Resolve() httpx.Middleware
func (s *Service) Register(mux *http.ServeMux)
func (s *Service) IssueSession(ctx, q Querier, userID, passwordHash string) (token string, err error) // Q3, shared with sign-up
func SetCookie(w http.ResponseWriter, r *http.Request, token string)
func ClearCookie(w http.ResponseWriter, r *http.Request)
```

`Querier` is the slice of `pgxpool.Pool` the package needs (`Query`, `QueryRow`, `Exec`) so a test passes a pool or a transaction; `Beginner` adds `Begin`. `Require` takes `auth.Admin`, `auth.Staff`, `auth.Customer` (and `auth.Require("admin")` compiles with an untyped constant). `CreateUser` is what this story's tests use to insert staff and customer rows, and what the staff and accounts stories call; it **never** hashes (R6): callers pass `PasswordHash` from `HashPassword`. `SetActive`, `SetPassword`, `UpdateProfile` and `Delete` of `_base.md` §1.1 are **not built here**; the staff and accounts stories add them to this package under the rules R1–R6 of the ERD file §6 (every one of them ends the user's sessions in its own transaction).

### 3.2 `Resolve` and `Require`

- `server.New` wraps the mux with `Resolve`: `Recover → AccessLog → CORS → Resolve → route-recorder → mux`. It reads the `session` cookie **only on paths under `/v1/`** (so `/healthz` never depends on the database) and **only when the cookie is well formed**: exactly 43 characters of `[A-Za-z0-9_-]` (a 32-byte token in unpadded base64url). Anything else is "no session" with no query.
- A well-formed cookie runs Q1 with a 3-second timeout. One row → the user goes in the context and `Trace.Role` is set (§3.6). No row → no user. Any other error → logged with the route pattern (never the cookie) and the request is answered `500 internal`, **for every route and every method** (it is not a 401: the person may be signed in). `Resolve` never denies by role; only `Require` does.
- `Require(roles…)` reads the context only. No user → `401 unauthenticated` "Sign in to continue."; a user whose role is not listed → `403 forbidden` "You do not have access to this."; otherwise the next handler. Both bodies are the bare envelope (no `errors`, no data) with `Cache-Control: no-store`. **A `Require` mounted where no `Resolve` ran answers 401 to everyone** — it fails closed, so a test must wrap its routes with `Resolve` (SC-18).
- "Dealer" = `Require(auth.Admin, auth.Staff)`. Account management (the staff story) = `Require(auth.Admin)`. A customer is `403` on both; no session is `401` on both.
- Because Q1 runs on every request that carries a cookie, a role change, a switch-off or a removal takes effect on the next request (SC-16). There is no cache.

### 3.3 Passwords

- **New hashes**: argon2id, `m=19456` KiB, `t=2`, `p=1`, a 16-byte random salt, a 32-byte key, encoded `$argon2id$v=19$m=19456,t=2,p=1$<unpadded base64 salt>$<unpadded base64 key>`. Dependency `golang.org/x/crypto` (go.mod changed through `run_go_mod_tidy`, never by hand).
- **Verification reads the parameters from the stored string**, not from constants. The test accounts' hash is `m=65536,t=3,p=4` (test cases, Fixtures) and a later change of the constants must not lock anybody out. Accepted: version 19 only, `m ≤ 262144`, `t ≤ 10`, `p ≤ 16`, key 16–64 bytes; compared with `subtle.ConstantTimeCompare`. A string that does not parse, or is out of range, verifies **false after running the dummy verification** (so a damaged row costs the same time as any other).
- **At most 4 hashes or verifications run at once** (a buffered channel `hashSlots`); a fifth waits, bounded by the request's context. This is the only protection the unauthenticated, expensive sign-in has against a flood from many addresses, each of which would otherwise cost 19 MiB at once in a container with a memory limit.
- **The dummy hash** is made once in `auth.New` — at start-up, never lazily, so the first unknown email does not pay for making it — from 32 random bytes with the current parameters, and used for every attempt that has no usable account.
- Passwords are never trimmed, logged, stored or returned; `Verify` and `HashPassword` log nothing.

### 3.4 Names and emails

- `NormalizeEmail(raw)`: trim surrounding whitespace, lower-case, then accept only printable ASCII (`!`–`~`, no space), 3–200 characters, exactly one `@`, a non-empty part before it, a domain that contains a `.` and neither starts nor ends with one. Returns `("", false)` otherwise. Used by every writer of `users.email` (start-up, staff, sign-up). **Sign-in does not call it**: it looks up `strings.ToLower(strings.TrimSpace(typed))`, so an address that could never be stored simply finds no row and takes the normal 401 path.
- `DisplayName()`: the stored `name` when it is not blank; otherwise the part of the email before `@`, split on `.`, `_` and `-`, empty parts dropped, **each word's first letter upper-cased**, joined with one space (`lan.tran@…` → `Lan Tran`, `mai_linh-pham@…` → `Mai Linh Pham`, `dealer@…` → `Dealer`); if nothing is left, the email.

### 3.5 Session token and cookie

- Token: 32 bytes from `crypto/rand`, unpadded base64url → 43 characters (256 bits; the contract needs ≥ 128). Stored only as `sha256(raw token bytes)` in `sessions.token_hash`; the token appears in no body and no log. Expiry: `now() + interval '7 days'` **in Q3**, never sliding.
- Cookie: `session`, `Path=/`, `HttpOnly`, `SameSite=Lax`, `Max-Age=604800` on sign-in and `0` on sign-out. `Secure` when the **last** entry of `X-Forwarded-Proto` (the one the edge proxy appended) equals `https` case-insensitively, or `r.TLS != nil`; otherwise absent. No `Domain`.
- A sign-in does **not** end the session the request already carried (no rotation): the token is server-made, so there is nothing to fixate, and ending it would make "two sign-ins, two rows" (SC-11) depend on whether the browser sent a cookie.

### 3.6 Log line

`httpx.AccessLog` writes `method`, **`pattern`** (the matched route pattern such as `POST /v1/auth/sign-in`; empty when nothing matched — the raw path is no longer logged, so an attacker's path never reaches the log), `status`, `ms` and **`role`** (`admin`, `staff`, `customer`, or empty). It still never logs a query string. Mechanism, so Dev does not invent another: `AccessLog` puts a `*httpx.Trace{Role, Pattern string}` in the request context (`httpx.TraceFrom(ctx)` returns a throw-away value when `AccessLog` is not mounted, so tests need no setup); `Resolve` sets `Role`; a tiny `httpx.RecordRoute(mux)` wrapper placed **directly around the mux** sets `Pattern` from `r.Pattern` after `mux.ServeHTTP` returns (the mux fills `r.Pattern` on the request it was given). Sign-in, sign-out and `Resolve` write no log line of their own except `internal` failures: `log.Error("…", "route", pattern, "error", err)`. **A constraint error is logged by its SQLSTATE and constraint name only**, never `err.Error()` or `PgError.Detail`, because Postgres puts the offending email in the detail (`Key (email)=(…) already exists`).

### 3.7 `httpx.ClientAddr(r) string`

The client address every limit uses (`_base.md` §2.2): the **right-most** non-empty entry of all `X-Forwarded-For` header lines joined with `,`, trimmed, if it parses as an IP (`netip.ParseAddr`, `Unmap`ped, in its canonical text); otherwise the host part of `r.RemoteAddr`. Never the left-most. It lives in `httpx` because the order-lookup and sign-up stories use it too.

## 4. Sign-in: order of checks, and the limits

### 4.1 `internal/ratelimit` (the only writer of `rate_limits`)

```go
func Subject(scope, subject string) []byte                                  // sha256(scope ‖ ":" ‖ subject), 32 bytes
func Check(ctx, q Querier, scope string, subject []byte) (wait time.Duration, err error)   // 0 = not locked
func Fail(ctx, q Querier, scope string, subject []byte, limit int, window, lock time.Duration) error
func Hit(ctx, q Querier, scope string, subject []byte, limit int, window time.Duration) (refused bool, err error) // sign-up
func Purge(ctx, q Querier) (int64, error)
```

All durations are bound as seconds (`make_interval(secs => $n)`); the clock is the database's `now()`.

- **`Check`** — `SELECT ceil(extract(epoch FROM locked_until - now()))::int FROM rate_limits WHERE scope = $1 AND subject_hash = $2 AND locked_until > now()`; no row means 0. The answer is at least 1 when locked, at most the lock (900), which is the `Retry-After`.
- **`Fail`** — one atomic upsert, so concurrent failures are counted exactly (a `SELECT` then an `UPDATE` would lose some):

```sql
INSERT INTO rate_limits AS r (scope, subject_hash, hits, window_started_at, locked_until)
VALUES ($1, $2, 1, now(), CASE WHEN $3 <= 1 THEN now() + make_interval(secs => $5) END)
ON CONFLICT (scope, subject_hash) DO UPDATE SET
  hits = CASE
    WHEN r.locked_until > now() THEN r.hits                                  -- locked: a failure never counts
    WHEN r.window_started_at <= now() - make_interval(secs => $4) OR r.locked_until <= now() THEN 1   -- a new window
    ELSE r.hits + 1 END,
  window_started_at = CASE
    WHEN r.locked_until > now() THEN r.window_started_at
    WHEN r.window_started_at <= now() - make_interval(secs => $4) OR r.locked_until <= now() THEN now()
    ELSE r.window_started_at END,
  locked_until = CASE
    WHEN r.locked_until > now() THEN r.locked_until                          -- never extended
    WHEN r.window_started_at <= now() - make_interval(secs => $4) OR r.locked_until <= now()
         THEN CASE WHEN $3 <= 1 THEN now() + make_interval(secs => $5) END
    WHEN r.hits + 1 >= $3 THEN now() + make_interval(secs => $5)             -- the fifth failure starts the lock
    ELSE NULL END
```

  (In an `UPDATE`, every right-hand side sees the **old** row, which is what the `CASE`s rely on.) The `updated_at` trigger is S3's.
- **`Hit`** — the sign-up counter of `_base.md` §3.21 (window 1 hour, the eleventh attempt sets `locked_until = window_started_at + window`); written here beside `Fail` so the accounts story does not create a second package, and tested there. Same upsert shape; it counts every call and reports whether this call is refused.
- **`Purge`** — `DELETE FROM rate_limits WHERE window_started_at < now() - interval '1 day' AND (locked_until IS NULL OR locked_until < now())`.

Scopes and numbers for sign-in: `sign_in_email` and `sign_in_address`, `limit 5`, `window 15 min`, `lock 15 min`. The subject of the first is `strings.ToLower(strings.TrimSpace(typed))`, of the second `httpx.ClientAddr(r)`.

### 4.2 The order of checks

1. **Media type** → `415`. 2. **Body**: cap 64 KiB (`413`), decode with `DisallowUnknownFields`, exactly one JSON object (`400`). 3. **Field checks** → `422` with every fault at once (nothing hashed, nothing counted). 4. **Limits**: `Check` both scopes; if either is locked → `429` with `Retry-After` = the larger wait (nothing hashed, nothing counted). 5. **Q2** by the lower-cased email. 6. **One verification**, always: against the row's hash when the row exists, against the dummy hash otherwise; an unparseable stored hash verifies false (§3.3). An inactive account's real hash is verified too, so it costs what a wrong password costs. 7. **Not signed in** (no row, inactive, or password false) → `Fail` on both scopes, then `401`. 8. **Signed in so far** → `Check` both scopes **again** (a lock may have begun while the hash ran); locked → `429`, no session. This second check is what makes "a sixth try with the right password is refused" hold under concurrent tries: a right password completes only if fewer than five failures were recorded when it finished. 9. **Q3**: no row inserted (the account changed after Q2, rule R2) → the same as step 7. 10. Set the cookie, answer `200`. A success neither counts nor clears the counters (SC-9).

A failure of any query is `500 internal` and counts nothing.

## 5. Start-up (`cmd/api/main.go`)

Order, a contract: configuration → connect → migrate → **`auth.EnsureAdmin`** → `go auth.RunPurge` → listen. The write rules for the admin row are in the ERD file §5. A settings problem (unset, short password, bad address, address taken) is logged and skipped and **the server always starts** (SC-4, SC-5); a **database error** in `EnsureAdmin` returns the error and the process exits non-zero, like a failed migration, so a deployment that cannot write the admin row is restarted instead of running with nobody able to sign in. The password is compared with `ADMIN_PASSWORD` exactly as supplied (never trimmed; length counted in characters, 10 to 100); the email is read through `NormalizeEmail`. The values appear in no log line.

`RunPurge` runs once at start and then every hour until the context is done: `DELETE FROM sessions WHERE expires_at < now()` (Q5), then `ratelimit.Purge`; each failure is logged and the loop goes on. It is the hourly ticker `_base.md` §3.19 names; the unpaid-order ticker of the payments story is a separate goroutine.

The owner supplies `ADMIN_EMAIL` and `ADMIN_PASSWORD` in the console (declared with `request_secret`; the story says they are already supplied). The backend `.env.example` already names both with no value; its comment should read "10 to 100 characters".

## 6. `GET /v1/orders` becomes team-only — two states, one rule

Story item 6 and SC-17: `GET /v1/orders` answers `401` with no valid session, `403` for a customer, and `200` with the **same list** it gives today for admin and staff. `GET /v1/orders/{id}` is **not** this story's: it stays as it is until the checkout story replaces it (the order's token, a dealer session, or the linked customer — `_base.md` §3.11). Which code answers the list when this backend PR is cut is decided by `main`, and **both** states are designed so the PR is correct whichever lands first:

| State of `main` | What this PR does |
|---|---|
| **A. `orders` is still a registry collection of `internal/data`** (S3 merged, checkout's backend not yet) | Add a bool `TeamOnly` to `data.Collection` (set on `orders` only). In `data.Register`, mount the list route of a `TeamOnly` collection as `mux.Handle("GET /v1/"+c.Name, auth.Require(auth.Admin, auth.Staff)(h.list(c)))`; the `/{id}` route is unchanged. `data` imports `auth` (allowed: packages share through `httpx` and `auth`); `auth` imports neither `data` nor `orders`. The comment on `Register` ("public: no session, no cookie is read") is corrected for this one route. The checkout PR deletes the `orders` collection with the flag |
| **B. `internal/orders` already mounts `GET /v1/orders` as the unconditional `401` stub** (checkout's backend merged first) | Replace the stub with `auth.Require(auth.Admin, auth.Staff)(list)`. The 200 body is the list envelope `{items,total,page,limit}` of the public columns of an order — exactly what S3's reader answered (no `phone`, `email`, `address`, `note`; the dealer's wider view is the manage story's) — served from the `orders` collection definition that the search and checkout designs moved into `internal/orders`. If that package has no list code yet, export the generic list handler constructor from `internal/data` and call it with that definition rather than writing a second query language; **no new filter, sort or paging rule** is added. The checkout story's own tests assert the stub's `401` for a request with no session; they stay true (401), and this PR adds the 403 and 200 cases |

Either way the tests of SC-17 are the same: no cookie → 401; a forged cookie (`session=AAAA…`, 43 characters) → 401; customer → 403; staff → 200; admin → 200; `GET /v1/products` with no session → 200. The BE review confirms by `grep -rn '"GET /v1/orders"' internal/` and reading the route table test that **exactly one** handler answers it and it sits behind `Require`.

## 7. Changes to code that already exists (and nothing else)

| File | Change |
|---|---|
| `internal/config/config.go`, `config_test.go`, `.env.example` | Fields `AdminEmail` (`ADMIN_EMAIL`, trimmed) and `AdminPassword` (`ADMIN_PASSWORD`, as supplied). Neither is required and neither is ever logged or put in an error. The test asserts both read, both empty without them |
| `cmd/api/main.go` | `EnsureAdmin` and `go RunPurge`, in the order of §5 |
| `internal/server/server.go` | `svc := auth.New(pool, log)`; `svc.Register(mux)` beside the other `Register` lines; the chain of §3.2 (`httpx.RecordRoute(mux)` directly around the mux, then `svc.Resolve()`, then CORS, AccessLog, Recover). `server_test.go` keeps its cases |
| `internal/httpx` | `Trace`, `TraceFrom`, `RecordRoute`, `ClientAddr`; `AccessLog` logs `pattern` and `role` instead of `path` (§3.6). `httpx_test.go` gains the cases |
| `internal/data` **or** `internal/orders` | §6 |
| `go.mod`, `go.sum` | `golang.org/x/crypto`, through `run_go_mod_tidy` |
| `migrations/` | the `users_and_sessions` pair (ERD file §3) |
| `code/frontend/src/lib/api.ts` | additive `post` (§2) |
| `code/frontend/src/lib/session.ts` | the three calls (§2); `Role` type moves in |
| `code/frontend/src/lib/mock/dealer-sign-in-and-protected-admin-area.ts` | **deleted** |
| `code/frontend/src/test/dealer-sign-in.test.tsx` | stubs `api` instead of the mock; the cases and their words are unchanged |

New packages: `internal/auth`, `internal/ratelimit`. Go tests that need a database read `TEST_DATABASE_URL` and skip without it; the rest run in CI with none. Required unit tests that need no database, because SC-7 and SC-18 are proved on the code path: (1) `signin` with a stub store and a counting verifier — the verifier is called exactly once for unknown email, wrong password, inactive account, no admin, and an unparseable stored hash, and **zero** times for a blank field, a bad media type and a locked address; (2) `Require` on two routes behind `Resolve` with a stub store for each of none/customer/staff/admin; (3) `NormalizeEmail`, `DisplayName` and the PHC parser with the fixture hash (`m=65536,t=3,p=4`); (4) `ClientAddr` with forged left-most entries.

## 8. Decisions, and what was rejected

| # | Decision | Rejected, and why |
|---|---|---|
| T1 | `Resolve` runs a query only for a well-formed cookie on a `/v1/` path, answers `500` when that query fails, and never denies; `Require` denies | *Lazy resolution at first use* — a route could forget to ask and treat a dealer as a visitor; one query per cookie-carrying request is cheap at this scale. *`401` on a lookup failure* — it tells the browser the person is signed out when the database is down, and the gate would redirect a signed-in dealer to sign-in for no reason |
| T2 | The 429 `Check` runs **twice** (before and after the hash) | One check — a burst of concurrent tries lets a right guess finish after the lock began. *A transaction holding an advisory lock per email across the hash* — it holds a pool connection for the whole hash and turns a flood into an outage of the whole API |
| T3 | Verification reads its parameters from the stored string; 4 hashes at most at once | Constants only — the first change of cost would lock everybody out; no cap — a flood from many addresses exhausts the container's memory before any address reaches its limit |
| T4 | `EnsureAdmin` writes nothing when nothing changed, ends **all** the admin's sessions on any change of email or password, and leaves an existing admin alone when a setting is removed | Re-hash on every start — a new salt each restart, an `updated_at` that lies, and a window where two instances disagree. Ending sessions only on a password change — a changed `ADMIN_EMAIL` usually means a different person. Locking out on a missing setting — an empty variable on one redeploy would end the owner's access |
| T5 | The admin row cannot take an address another account holds; it is logged and skipped | Taking over the other account's row — would silently turn a customer into the admin |
| T6 | `GET /v1/orders` guard in two states (§6) | A path-prefix guard in `server.New` — it would still reach checkout's unconditional `401` stub for a dealer; waiting for the checkout story — the plan runs both at once |
| T7 | `AccessLog` logs the route pattern and the role, not the path | Logging the path — an attacker chooses it, and SC-20 asks for the pattern |
| T8 | `httpx.ClientAddr` lives in `httpx` | In `auth` — the order-lookup story would import `auth` for an address |
| T9 | No rotation of the session the request already carried (§3.5) | Rotation — adds a rule the SRS never states and makes SC-11 depend on the cookie jar |

## 9. The test cases' ambiguities, answered

1. **SC-20 "no response body contains the email typed"** — read as the test cases read it: no **error** answer (400, 401, 413, 415, 422, 429, 500) echoes the typed email or password, and no answer of any kind carries a password, a hash or a token. The `200` of sign-in and of `GET /v1/auth/me` carry the account's own `email`, as the contract says.
2. **An over-long email or password** — `422` (§1.1), not counted, nothing hashed.
3. **SC-3's second sentence** (a cookie issued before a changed `ADMIN_PASSWORD` stops working) — adopted, and extended to a changed `ADMIN_EMAIL` (ERD file §5). `_base.md` §3.9 is not changed: a `401 unauthenticated` is already what an ended session answers.
4. **SC-10 "right-most entry"** — `httpx.ClientAddr` (§3.7). If the verifier's gateway appends its own entry, every request shares it; that is the verifier's observation limit, not a defect.
5. **SC-12 `Secure`** — decided by the **last** `X-Forwarded-Proto` entry or `r.TLS` (§3.5).
6. **"The only link to `/admin`"** — a frontend reading, unchanged by this design.
7. **"A similar time"** — one argon2id verification on every attempt that reaches step 6; the unit test of §7 is the proof.

## 10. Backend review checklist (item 9 BE review)

- [ ] R1 `Resolve`'s query has `AND u.is_active` and `expires_at > now()`; R2 the session insert is the `INSERT … SELECT … FOR SHARE OF u` of Q3 with `u.password_hash = $3`; R3 `EnsureAdmin`'s update writes `is_active = true`.
- [ ] Exactly one verification per attempt that passes step 4; none for 415/400/422/429; the dummy hash is made in `New`.
- [ ] Verification reads `m`, `t`, `p` from the string and passes the fixture hash.
- [ ] The cookie flags and the `Secure` rule of §3.5; `Cache-Control: no-store` on the three routes; the token is in no body and no log.
- [ ] `Fail` is the one upsert (no read-then-write); `Check` is called twice on a success path; a refused call writes nothing.
- [ ] `EnsureAdmin` is one transaction with `FOR UPDATE`, writes nothing when unchanged, ends sessions on a change, never logs a value or `PgError.Detail`.
- [ ] A route-table test: `GET /v1/orders` is behind `Require` and is answered by one handler (§6); `GET /v1/products` is still public.
- [ ] No `auth` import in `ratelimit`; no package writes `users`, `sessions` or `rate_limits` outside `auth` / `ratelimit`; no route is mounted under `/api`.
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` pass with no database; the migration applies on boot twice with no change; no SQL string is built by concatenation.
- [ ] Frontend: `session.ts` reads and writes no `localStorage` or `sessionStorage`; the mock is deleted and nothing imports it; `api.post` is never retried and accepts a `204`; no `Admin*.tsx` (other than `AdminSignIn.tsx`), `DashboardLayout.tsx`, `RequireRole.tsx` or `<Route>` line changed.

## 11. Deployment order

Migration `users_and_sessions` applies before the listener (nothing reads it until `EnsureAdmin`, which runs right after). With neither setting supplied the shop runs and nobody can sign in as admin; supplying them and restarting makes the account. No data is migrated and no existing endpoint changes except `GET /v1/orders` (§6) and the access-log fields (§3.6). Rolling the code back leaves the two tables in place, which nothing older reads.
