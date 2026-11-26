# Service contracts — Customer accounts: sign up, sign in, my orders (plan item 15, "S14")

Module `accounts` (package `internal/account`; additive changes in `internal/auth`, `internal/orders`, `internal/ratelimit`) · Requirements ACC-001 to ACC-008 · Story `docs/accounts/stories/customer-accounts-sign-up-sign-in-my-orders.md` · Test cases `docs/accounts/test-cases/customer-accounts-sign-up-sign-in-my-orders.md` · UI PR #43 · Tables: `docs/architecture/erd/customer-accounts-sign-up-sign-in-my-orders.md`.

This file **extends** `services/_base.md`; it does not redesign it. The four endpoint groups of `_base.md` §3.18 to §3.21, the three routes they touch (§3.10, §3.11, §3.13), the session and cookie rules (§2.2), the error catalog (§2.3), pagination (§2.4), the validation boundary (§2.5) and the limits (§2.7) are **fixed and are not repeated** except where this file fills a gap or corrects a line. Every path is written the way the Go backend mounts it — **no `/api` prefix**; the browser's `/api/v1/account` reaches Go as `/v1/account`. A failed query is `internal` (500), never `unavailable`. This file adds **no error code**, no second error shape and no second list format.

What this file decides, because `_base.md` or the story left it open: the order of each endpoint's checks and every sentence the shop sends (§2); the Go surface other stories call, and the one shared place of the number-and-phone matcher (§3); what the UI's mock got right and wrong, and the frontend's swap (§4, §5); the limits (§6); the decisions with their rejected alternatives (§7); the lines of merged documents this design corrects (§8); the BE review checklist and how each criterion is proved (§9); and the build order (§10).

## 1. Endpoints, and the three routes this story changes

| # | Method and path | Auth | Traces to |
|---|---|---|---|
| 3.18 | `POST /v1/account/sign-up` | none; limited by `sign_up_address` | ACC-001 |
| 3.19 | `GET /v1/account`, `PUT /v1/account` | `auth.Require(auth.Customer)` | ACC-004, ACC-005 item 1, ACC-008 |
| 3.20 | `GET /v1/account/orders`, `GET /v1/account/orders/{id}` | `auth.Require(auth.Customer)` | ACC-003, ACC-008 |
| 3.21 | `POST /v1/account/orders` | `auth.Require(auth.Customer)`; limited by `order_lookup_number`, `order_lookup_address` | ACC-006 |
| 3.10 | `POST /v1/orders` | none; a **customer** session links the order | ACC-005 |
| 3.11 | `GET /v1/orders/{id}` | token, **or the linked customer's session**, or a dealer | ACC-003 item 4, ACC-008 item 3 |
| 3.13 | `POST /v1/orders/{id}/payment` | token, **or the linked customer's session** | order-page design §4 |

Every route is a **literal pattern** (`POST /v1/account/sign-up`, `GET /v1/account`, `PUT /v1/account`, `GET /v1/account/orders`, `GET /v1/account/orders/{id}`, `POST /v1/account/orders`). A wrong method on these paths falls to the server's catch-all and answers `404 not_found` in the envelope, never a plain-text 405; `DELETE /v1/account` and the like do not exist (deleting an account is out of scope). **No route takes an account id** and the handlers never read `?id=`, `?account=` or `?user=` (ACC-008 item 2): the account is always `auth.FromContext(ctx)`. Every answer of every route below carries `Content-Type: application/json; charset=utf-8` and `Cache-Control: no-store`, and none ever holds a password, a hash, a token, a role (except sign-up's own three-key body), an id of a user, or another person's data.

For the three routes that need a session, `auth.Require(auth.Customer)` runs **before the body is read**: no session is `401 unauthenticated` ("Sign in to continue."), a staff or admin session is `403 forbidden` ("You do not have access to this."), whatever the body or the media type. A switched-off or removed account has no session (`Resolve` requires `is_active`), so it is `401`.

## 2. Answers, checks and sentences

### 2.1 `POST /v1/account/sign-up` (§3.18)

**Order of checks.** Each runs only when the earlier ones passed; the first that refuses answers.

1. **Media type** — not `application/json` (parameters such as `; charset=utf-8` allowed) → `415 unsupported_media_type`, "Send the request as JSON." Not counted.
2. **Body** — cap 64 KiB (`413 payload_too_large`); exactly one JSON object, decoded with `DisallowUnknownFields` into `{name, email, phone, password}` as strings (`400 bad_request`, `errors.<field>` when the fault is a field: an undefined field — `role` and `is_active` included, SC-1 — or a wrong type). A missing or `null` field is the empty string and is judged in step 4. Not counted.
3. **The address limit** — `ratelimit.Hit` on `sign_up_address` for `httpx.ClientAddr(r)`, 10 an hour (§3.3). Refused → `429 rate_limited`, "Too many tries. Please wait and try again.", `Retry-After` the whole seconds left (1 to 3600). **From here on every request counts**, whether it ends 201, 409 or 422; a request refused here counts nothing and does not move the lock (SC-4). Nothing is hashed and nothing is written for a refused call.
4. **Fields** — every fault at once → `422 invalid`, "Check the form.", `errors` naming each (table below). Nothing is hashed and no cookie is set.
5. **Hash** — `auth.HashPassword(password)` runs **outside any transaction** (rule R6) and through the same bounded slots as sign-in's verification (at most 4 hashes or verifications at once, `auth` design §3.3): a sign-up flood from many addresses is bounded by the slots exactly as a sign-in flood is. This is requirement **R7** of this story's BE review: if the merged `HashPassword` is not bounded, this PR bounds it.
6. **One transaction** — `auth.CreateUser` (role `customer`, `Phone` as typed and trimmed, `Address` and `City` `''`) then `(*auth.Service).IssueSession` with the hash just written (Q3's twin), then commit. `auth.ErrEmailTaken` (SQLSTATE 23505 on `uq_users_email`) → rollback → `409 conflict`. There is **no `SELECT` of the email before the insert**: the unique index is the only arbiter, so two sign-ups at the same moment give one `201` and one `409` and one row (SC-2), and a taken email costs the same time as a free one because the hash ran in step 5 either way. Account and session are made together, so a failure never leaves an account the customer cannot use behind a `500`.
7. `auth.SetCookie` (§2.2 of the base: `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` over HTTPS, `Max-Age=604800`) and `201 Created`:

```json
{ "name": "Lan Nguyen", "email": "lan@example.com", "role": "customer" }
```

Exactly these three keys; `name` as stored (trimmed), `email` as stored (lower case), `role` always `customer`. A sign-up does **not** end a session the request already carried (no rotation, as sign-in), so a team member who signs up while signed in keeps their old session row too; the new cookie replaces it in the browser.

**Field rules, and the sentences — the same words as the page's own (`src/lib/account-words.ts`, `W.rules`), so the shop's sentence and the page's never disagree.** Text is trimmed (the password never is); lengths count runes; a value containing U+0000 is refused with the field's own sentence, because PostgreSQL cannot store it and the insert would otherwise fail as a `500`.

| Field | Rule | `errors.<field>` |
|---|---|---|
| `name` | 1 to 100 characters | "Fill in your name" (missing, empty, or only spaces) · "Use at most 100 characters" |
| `email` | `auth.NormalizeEmail` accepts it (printable ASCII, one `@`, a dot in the domain, no space) and it is at most 200 characters | "Enter a valid email address" (every fault, the length included) |
| `phone` | at most 20 characters and at least 9 ASCII digits (`orders.ValidPhone`, the checkout's rule, §3.6) | "Check the phone number" |
| `password` | 8 to 100 characters, counted as typed | "Use at least 8 characters" (missing, empty, or short) · "Use at most 100 characters" |

| Code | HTTP | Trigger |
|---|---|---|
| `unsupported_media_type` | 415 | Step 1 |
| `bad_request` | 400 | Step 2: not an object, a wrong type, an undefined field; `errors.<field>` names it |
| `payload_too_large` | 413 | Step 2 |
| `rate_limited` | 429 | Step 3 |
| `invalid` | 422 | Step 4 |
| `conflict` | 409 | Step 6. Message "An account with this email already exists."; `errors.email` "An account with this email already exists. Sign in instead." — for a customer's, a staff member's and the admin's email, in any case (SC-2). The only `conflict` that carries `errors` |
| `internal` | 500 | Any other failure; "Something went wrong. Please try again." Nothing was created; the cause is logged by route and request id, a constraint error by SQLSTATE and constraint name **only** (never `err.Error()` or `PgError.Detail`, which holds the email) |

**Notes.** The password and its hash appear in no response, no log line and no error (SC-1). The server confirms that an email is taken (SRS ACC-001 item 4 demands it), so sign-up can be used to test whether an address has an account; the only brake is the 10-an-hour limit (accepted risk R-1, §7).

### 2.2 `GET /v1/account` and `PUT /v1/account` (§3.19)

`GET` → `200 {"name","email","phone","address","city"}` — those five keys in that order and nothing else (A3). No row for the session's id (the account vanished since `Resolve`) → `401 unauthenticated`.

**`PUT` order of checks** (after `Require`): (1) media type → `415`; (2) body cap `413`, one JSON **object** (`400` for an array, `null`, a string or trailing data); (3) **every key must be one of `name`, `phone`, `address`, `city`, `email`, `role`, `password`** — any other key (`id` included: this route "takes no id", so the project's tolerated `id` of a collection `PUT` does not apply, test finding 5) is `400 bad_request`, `errors.<key>`; a wrong type for `name`, `phone`, `address` or `city` (a number, `null`, an object) is `400`; (4) **then** `422 invalid`, "Check the form.", with `errors` naming every fault together: `email`, `role` and `password` present **whatever their value or type** ("This cannot be changed here"), and each limit that is broken; (5) the write (A4) — only when the body has at least one of the four fields; `{}` is `200` with the current values and no write; (6) `200` the same five keys as `GET`.

| Field | Rule | `errors.<field>` |
|---|---|---|
| `name` | trimmed, 1 to 100 characters, no U+0000 | "Fill in your name" · "Use at most 100 characters" |
| `phone` | trimmed, at most 20 characters, at least 9 ASCII digits | "Check the phone number" |
| `address` | trimmed, at most 200 characters; `""` clears it | "Use at most 200 characters" |
| `city` | trimmed, at most 100 characters; `""` clears it | "Use at most 100 characters" |
| `email`, `role`, `password` | refused if present | "This cannot be changed here" |

A refused body changes nothing, **even for the other fields in the same body** (SC-7): step 4 runs before step 5. Two `PUT`s at once: the later wins, the answer shows what was saved last. Saving never touches an order: an order keeps what was typed at checkout, and placing an order never writes the account (ACC-005 item 2). Codes: `unauthenticated` 401, `forbidden` 403, `unsupported_media_type` 415, `bad_request` 400, `payload_too_large` 413, `invalid` 422, `internal` 500.

### 2.3 `GET /v1/account/orders` and `GET /v1/account/orders/{id}` (§3.20)

**The list.** `200 {"items":[…],"total":n,"page":p,"limit":l}`; an item is exactly `{"id","number","placed_at","status","total"}` — `placed_at` RFC 3339 with `+07:00`, `status` one of the six, `total` an integer — and never `phone`, `email`, `address`, `note`, `customer`, `city` or `account_id` (the statement does not select them). Newest first: `placed_at DESC`, then the order's own `seq DESC` (A5).

| Parameter | Rule |
|---|---|
| `page` | whole number ≥ 1, default 1; anything else `400 bad_request`, `errors.page`. A page past the last is `200` with `items: []` and the true `total` |
| `limit` | whole number ≥ 1; anything else `400`, `errors.limit`. **Default 10 for this route** (the one departure from the base's 12, §8 C1: the story's SC-8 pins `"limit":10` for a call with no parameters, and "Show more" works in tens). **Above 100 is served as 100** and the answer says `100` (§2.4 of the base wins over the line in §3.20 that lists "invalid `limit`" as `400`: the clamp is the project rule, `400` is for `0`, negative and non-numeric values — test finding 4) |
| anything else | ignored (`?sort=`, `?q=`, `?account=`) |

A customer with no order gets `{"items":[],"total":0,"page":1,"limit":10}`. The page and the total are read in one `REPEATABLE READ`, read-only transaction, so a "Show more" never shows a row twice or skips one that a concurrent placement moved.

**One order.** `GET /v1/account/orders/{id}` → `200` the public view with `items`, **the same bytes** as `GET /v1/orders/{id}` gives for the same order: it calls `orders.Read(ctx, id, Access{AccountID: <the session's id>})` and writes the view the same way. `{id}` must match `^[a-z0-9][a-z0-9._-]*$` and be at most 100 characters, else `404` with no query. An order that is another customer's, a guest's (linked to none), or does not exist is `404 not_found`, `{"error":"not_found","message":"Not found."}`, written by the **one function** that writes the order not-found body (order-page design §4), so the bytes cannot differ between this route, `GET /v1/orders/{id}` and the pay route (SC-9, SC-10).

Codes: `unauthenticated` 401, `forbidden` 403, `bad_request` 400 (list only), `not_found` 404 (`{id}` only), `internal` 500.

### 2.4 `POST /v1/account/orders` (§3.21)

Body `{"number": "doi 24818", "phone": "+84912345678"}`. After `Require`: (1) media type `415`; (2) body cap `413`, one object with exactly the keys `number` and `phone`, both strings and **both present** (`400`, `errors.<key>` for a missing key, a `null`, a wrong type or an undefined field; an **empty string is present** and is judged by the match, as §3.12 says: "their content is judged by the match, not by a `422`"); (3) **`orders.Attempt`** (§3.4), which does everything else in one transaction:

1. takes a transaction advisory lock on the typed order number (so tries for one number run one after another), then `ratelimit.Check`s `order_lookup_number` (the typed digits) and `order_lookup_address` (`httpx.ClientAddr`). Either locked → `429 rate_limited`, "Too many tries. Please wait N minutes." (N = the wait in minutes, rounded up, at least 1), `Retry-After` the whole seconds left (1 to 900). **A refused call counts nothing** and a right pair does not help (SC-15);
2. finds the order by number and compares phones (§3.6). No such order, or a phone that does not match in full, or an empty phone → **a miss**;
3. a match is handed to this route's callback: an order linked to **no one** is linked to this account (A8); one **already linked to this account** is left as it is (`200`, no second row, no write); one linked to **another** account is a miss — *indistinguishable from the others*; a race that linked the order between the find and the link (zero rows updated) is a miss as well;
4. a miss counts one failure on both scopes (`order_lookup_number` only when the typed number has digits), then answers `404`; a success neither counts nor clears.

**Success** `200` `{"id","number","placed_at","status","total"}` (the same item shape as the list). The guest browser's tokens keep working and `POST /v1/order-lookups` still finds the order (SC-13): nothing is deleted or changed about the order but its owner (I4).

**Not found** `404 not_found`, message "No order matches that number and phone." — **one status and one body** for a wrong phone, an unknown number, an empty phone, a phone that matches only in part, and an order of another customer's account (SC-14). The body never says which.

Codes: `unauthenticated` 401, `forbidden` 403, `unsupported_media_type` 415, `bad_request` 400, `payload_too_large` 413, `not_found` 404, `rate_limited` 429, `internal` 500. A database failure inside `Attempt` rolls everything back (nothing counted, nothing linked) and is `500`.

### 2.5 The three routes this story changes

**`POST /v1/orders` (§3.10) links the order.** The handler reads `auth.FromContext(r.Context())`; when the user's role is `customer` it passes that id as the placement's `AccountID`, and for every other case — no session, a session that ended since the page opened (`Resolve` then found nobody), a staff session, the admin — it passes `""`. The insert binds `account_id` as `(SELECT u.id FROM users u WHERE u.id = $n AND u.role = 'customer' AND u.is_active)` (ERD A9, invariant I1), so the rule also holds for a caller written later and an account that vanishes between `Resolve` and the insert gives a guest order, never a refusal (SC-12, SC-38). The typed `customer`, `phone`, `address`, `city` and `note` are stored as typed whatever the account holds, and the account is never written (SC-11). The answer is unchanged: it never mentions the account. **A replay** of a used `Idempotency-Key` returns the first order and **never links or relinks it** (I3), whoever sends it.

**`GET /v1/orders/{id}` (§3.11) gains its third way in** by extending `authorize`, the one function the order-page design made for exactly this (order-page design §4): `Access` gains `AccountID` (`""` = none), set by the handler to the session's id **only for role `customer`**, and the access statement becomes `… WHERE o.id = $1 AND (EXISTS (SELECT 1 FROM order_tokens t WHERE t.order_id = o.id AND t.token_hash = $2) OR (o.account_id IS NOT NULL AND o.account_id = $3::uuid))` (A6). Its steps change in one place: **a missing or malformed token no longer ends the request before the query when the caller is a customer** — step 2 of that design ("no token → `ErrNotFound` with no query") becomes "no valid token **and** no `AccountID` **and** not a dealer → `ErrNotFound` with no query", so a guest with no token still costs no query and a customer with no token costs one. A customer who sends a wrong token for an order linked to them still reads it (the first way that applies wins, and the token is only one of them); a customer B asking for A's order, a guest order with no token and an id that does not exist all run the same statement and answer the same bytes (SC-10). The answer is the public view; it never holds `phone`, `email`, `address` or `note`, whatever the way in.

**`POST /v1/orders/{id}/payment` (§3.13)** reads the same `Access` through the same function in `modePay`, so the linked customer may press "Pay now" with no header; nothing else about it changes. A dealer session without the token is still `404` there.

## 3. The Go surface

### 3.1 Packages and who imports whom

```
account  ──►  auth, orders, ratelimit (Hit, for sign-up only), httpx
orders   ──►  auth (FromContext, Role constants — in the handlers only), ratelimit, stock, httpx
auth     ──►  ratelimit, httpx
```

No cycle: `auth` never imports `orders` or `account`; `orders` never imports `account`. **`internal/account` contains no SQL at all.** It reads and writes `users` only through `auth` functions, and `orders` only through `orders` functions, so "a package never imports another package's store" (`_base.md` §1.1) holds and each table keeps its one writer. This is also the answer to the story's question "how does `GET /v1/orders/{id}` find the session of the customer the order is linked to without importing another package's store": the `orders` handler asks `auth` **who the caller is** (`auth.FromContext`, a public function of the package that owns sessions) and passes the id as a plain string in `Access.AccountID`; the SQL that compares it with `orders.account_id` is `orders`' own. `orders` imports `auth` from this story on (the checkout design said its handler "does not import `auth`" only because the package did not exist yet; "Dealer sign-in" and this story both add the import, as the checkout design's dependency table anticipated: "Add `Dealer` and `AccountID` to the `access` struct").

Signatures below are the **shape**: where the merged `orders` or `auth` code already uses package functions over a `q`, or methods on a service `Register` builds, the new functions follow that convention; what is passed and returned is the contract.

### 3.2 `internal/auth` — two functions added

```go
type Profile struct{ Name, Email, Phone, Address, City string }
type ProfilePatch struct{ Name, Phone, Address, City *string } // nil = leave as it is, "" = clear (address, city)
var ErrNoAccount = errors.New("account not found")             // no customer row for that id, or it is switched off

func GetProfile(ctx context.Context, q Querier, id string) (Profile, error)                 // A3
func UpdateProfile(ctx context.Context, q Querier, id string, p ProfilePatch) (Profile, error) // A4
```

Both carry `role = 'customer' AND is_active` in their `WHERE`, so a staff row can never be read or edited through them. `UpdateProfile` is the function `_base.md` §1.1 already names for `users`; the staff story's `SetName` stays its own. Sign-up uses what item 9 already ships: `NormalizeEmail`, `HashPassword`, `CreateUser` (`NewUser{Email, Name, Phone, Role: Customer, PasswordHash}`), `ErrEmailTaken`, `(*Service).IssueSession`, `SetCookie`, `DisplayName`. **One `CreateUser`, never two** (item 9 §3.1): if the staff backend merged first and declared them, this PR adopts them.

### 3.3 `internal/ratelimit` — `Hit`, pinned

Item 9's design (§4.1) says `Hit` is "written beside `Fail` so the accounts story does not create a second package" and only sketches it as `Hit(…) (refused bool, err error)`. Two changes, so that sign-up has what it needs: it returns the **wait** (the `Retry-After` is a number of seconds, §8 C3), and its SQL is fixed here so the item 9 PR and this PR cannot differ.

```go
// Hit counts one attempt and reports how long the subject is locked: 0 = this attempt may go on.
func Hit(ctx context.Context, q Querier, scope string, subject []byte, limit int, window time.Duration) (wait time.Duration, err error)
```

```sql
INSERT INTO rate_limits AS r (scope, subject_hash, hits, window_started_at, locked_until)
VALUES ($1, $2, 1, now(), NULL)
ON CONFLICT (scope, subject_hash) DO UPDATE SET
  hits = CASE
    WHEN r.window_started_at <= now() - make_interval(secs => $4) THEN 1      -- the hour is over: a new window
    WHEN r.hits >= $3 THEN r.hits                                             -- at the limit: refused, not counted
    ELSE r.hits + 1 END,
  window_started_at = CASE
    WHEN r.window_started_at <= now() - make_interval(secs => $4) THEN now()
    ELSE r.window_started_at END,
  locked_until = CASE
    WHEN r.window_started_at <= now() - make_interval(secs => $4) THEN NULL
    WHEN r.hits >= $3 THEN r.window_started_at + make_interval(secs => $4)    -- the lock ends with the window, never extended
    ELSE r.locked_until END
RETURNING CASE WHEN locked_until > now() THEN greatest(1, ceil(extract(epoch FROM locked_until - now()))::int) END
```

(In an `UPDATE`, every right-hand side sees the **old** row — the same fact `Fail`'s `CASE`s rely on.) For `$3 = 10`, `$4 = 3600`: attempts 1 to 10 return NULL (go on) and raise `hits` to 10; the **eleventh** finds `hits = 10 ≥ 10`, leaves `hits` at 10, sets `locked_until` to the window's end and returns the seconds left; the twelfth and later find `hits ≥ 10` again and return the shrinking wait without moving anything (SC-4: "a call refused with 429 does not extend the lock"); once `window_started_at` is an hour old the next call starts a new window. A concurrent pair is exact: the upsert takes the row lock. The subject is `ratelimit.Subject("sign_up_address", httpx.ClientAddr(r))`; the clock is the database's `now()`, so a test moves `window_started_at` back instead of sleeping. `Purge` already removes a row a day after its window and once unlocked. Sign-in's `Fail` and this `Hit` share the table and never a row (different scopes), so sign-in failures do not count toward sign-up and the reverse (SC-4).

### 3.4 `internal/orders` — what this story adds

```go
// access: Access gains AccountID (order-page design §4); authorize's statement is A6.
type Access struct{ Token string; AccountID string; Dealer bool } // "" = none; Dealer is item 9's

// Placement gains the caller's account. The handler sets it ONLY for role customer.
// Place binds it through the guarded subselect of invariant I1 and writes it on the insert only (I3).
//   Placement.AccountID string

// "My orders", newest first (A5). page >= 1, 1 <= limit <= 100.
type AccountOrder struct {
    ID, Number, Status string
    PlacedAt           time.Time
    Total              int64
} // JSON: id, number, placed_at (+07:00), status, total
func ListFor(ctx context.Context, db Beginner, accountID string, page, limit int) (rows []AccountOrder, total int, err error)

// Link makes a guest order the account's (A8). linked is false when no row changed.
func Link(ctx context.Context, q Querier, accountID, orderID string) (linked bool, err error)
```

**The number-and-phone matcher has one home, `internal/orders/lookup.go`.** The story (§8) says: if the matcher is not on `main` when this design is written, TL places it in one shared place both stories import and it is never written twice. "Track an order by number and phone" (plan item 7) has no merged backend and no design yet, so this is that place. The owner of `orders` is the right package: it owns the table, and its ownership row already lists `Link`, the lookup and `ratelimit` as its callers. Both `POST /v1/order-lookups` (item 7) and `POST /v1/account/orders` (this story) call the **same function**; whichever backend merges first creates the file exactly as below and the second adopts it unchanged. Item 7's design must read this section rather than redesign it.

```go
type Pair struct{ Number, Phone string }          // as typed
type Matched struct {                             // never holds the phone
    ID, Number, Status string
    PlacedAt           time.Time
    Total              int64
    AccountID          string                     // "" = a guest order
}
var ErrNoMatch = errors.New("no order matches that number and phone")
type LockedError struct{ Wait time.Duration }      // 1s to 900s; implements error

// NumberKey: the ASCII digits of the typed number; ok only when there are 1 to 18 of them.
// "DOI-24818", "doi 24818" and "24818" all give "24818"; the stored number is "DOI-" + the digits.
func NumberKey(raw string) (digits string, ok bool)
// PhoneKey: the ASCII digits of the phone, a leading "84" read as "0". "" when there are none.
// "+84 912 345 678" and "0912345678" both give "0912345678". Compared in full; "552" never matches.
func PhoneKey(raw string) string

// Attempt is one try, for both callers. All of it is ONE transaction:
//   1. if the number has digits, SELECT pg_advisory_xact_lock(hashtextextended('order_lookup:' || digits, 0));
//   2. ratelimit.Check order_lookup_number (only with digits) and order_lookup_address; either locked ->
//      rollback, return *LockedError (nothing counted);
//   3. SELECT … FROM orders WHERE number = 'DOI-' || digits (A7); no row, or PhoneKey(typed) == ""
//      or != PhoneKey(stored) -> miss;
//   4. otherwise call hit(ctx, tx, matched):
//        nil          -> commit; return the matched order. Nothing is counted or cleared;
//        ErrNoMatch   -> miss (hit must have written NOTHING before returning it);
//        any other    -> rollback; return it (500);
//   miss: ratelimit.Fail order_lookup_number (only with digits) and order_lookup_address
//        (limit 5, window 15 min, lock 15 min), commit, return ErrNoMatch.
func Attempt(ctx context.Context, db Beginner, addr string, p Pair,
    hit func(ctx context.Context, tx pgx.Tx, m Matched) error) (Matched, error)
```

`POST /v1/order-lookups` passes a `hit` that inserts an `order_tokens` row and returns; `POST /v1/account/orders` passes one that switches on `m.AccountID`: `""` → `Link` (zero rows → `ErrNoMatch`), the caller's own id → `nil`, anything else → `ErrNoMatch`. A number with no digits (or over 18) counts on the address scope only: a shared "empty number" bucket would let one visitor lock every other empty try. The advisory lock is what makes "five failures, then a sixth refused" hold when tries for one number arrive at once, and is the reason `Attempt` is one function: the sign-in design's second `Check` after its hash covers a *right* password finishing late; a lookup has no slow step, so serialising per number closes the gap instead, and the number is the thing a guesser aims at (a ten-digit phone behind five guesses per 15 minutes). Tries across *different* numbers from one address may exceed five in a burst; each number stays at five.

### 3.5 `internal/account` — the package

```
internal/account/
  handler.go   Register(mux, Deps): the six routes of §1, each wrapped as §1 says
  signup.go    §2.1
  profile.go   §2.2
  orders.go    §2.3, §2.4
  validate.go  the field rules and every sentence of §2.1 and §2.2, in one table
  *_test.go
```

`Deps{ Auth *auth.Service; DB orders.Beginner; Log *slog.Logger }`. `server.New` calls `account.Register` after `auth` and `orders` have registered (the mux panics on a duplicate pattern, and none repeats); the middleware chain is the sign-in design's, unchanged: `Recover → AccessLog → CORS → Resolve → route-recorder → mux`. The access log gets `role` from `Resolve` as before, and **no handler of this package logs a body, a header, a password or a hash**; an `internal` failure logs `route`, the request id and the error (a constraint error by SQLSTATE and constraint name only).

### 3.6 One phone rule, one place

The checkout's phone rule ("at most 20 characters, at least 9 ASCII digits; spaces, `+`, `-` and brackets are simply not counted", checkout design S8) is needed by checkout, sign-up, `PUT /v1/account` and the matcher. It lives in `internal/orders/phone.go` as `PhoneDigits(raw) string` and `ValidPhone(raw) bool`, beside `PhoneKey`. If checkout's backend wrote it as a private helper, this PR **moves** it there (no behaviour change; checkout's own tests are the safety net) and `account` imports it.

## 4. The mock, read and judged

The mock module (`src/lib/mock/customer-accounts-sign-up-sign-in-my-orders.ts`) is the one file of PR #43 whose body the pull-request view does not carry: it is listed as added without its diff, and this task forbids standing on the UI branch to open it. It is imported only by the story's client, `src/lib/customer-accounts-sign-up-sign-in-my-orders.ts`, which **is** in the diff in full, together with every component that renders the answers. So the shapes the frontend renders, the statuses it branches on and the words it shows are read from the client and the components and from the PR's own account of the mock ("follows `services.md` §3.7–3.9 and §3.18–3.21 for status and body; nothing in it was invented beyond that"). What the mock does inside — its latency, the sample customers, the `down…`/`limit…`/`slow…` switches — is test scaffolding and goes with it. It is unreviewed input; where `_base.md` already decided a shape, that wins.

| Frontend (client and components) | Contract | Judgment | Change at the swap |
|---|---|---|---|
| `ask()` takes `Promise<Reply>` — a `{status, body}` that **resolves** for every answer — and sorts by `status` | `src/lib/api.ts` resolves a 2xx with the parsed body and **throws `ApiError`** (`status`, `code`, `message`, `errors`) for anything else, and for no answer (`status` 0, `code` `network`) | **The mock's shape is not the real client's**, hidden inside one file | `ask()` takes a `Promise<T>` and catches `ApiError`. A status is sorted into its kind **only when `code` is the envelope's own for that status**: 401 with `unauthenticated` or `invalid_credentials` → `unauthenticated`; 403 `forbidden`; 404 `not_found`; 409 `conflict`; 422 `invalid`; 429 `rate_limited`. **Everything else — status 0, a 5xx, a 400, 413, 415, and a 401, 403 or 404 whose code is `internal` (a gateway's own page, a mis-routed `/api`) — is `unreachable`.** A broken proxy rule must never tell a customer "No order matches that number and phone" or "Wrong email or password" (the same departure the order-page design makes for its 404). A 2xx body that lacks the expected keys (`Me`, `Account`, `OrdersPage`, `OrderRow`) is `unreachable` too |
| `Me` `{email, name, role}` from `GET /v1/auth/me`, and sign-up's answer read as the same type | `/v1/auth/me` → `{email,name,role}`; sign-up → `{name,email,role}` | Sound: the same three keys, order is irrelevant to JSON | None |
| `Account` `{name, email, phone, address, city}`; `Details` is the four editable | §2.2: the same five keys | Sound | None |
| `OrdersPage {items,total,page,limit}`, `OrderRow {id,number,placed_at,status,total}`; `listOrders(page, 10)`; `STATUS_WORDS` covers the six statuses | §2.3: the same keys and types; `placed_at` RFC 3339 `+07:00` (`formatDate` builds a `Date` from it); `limit=10` sent explicitly and also the route's default | Sound; `PAGE_SIZE = 10` stays | `listOrders` → `api.get("account/orders", {page, limit})` |
| `conflict` and `invalid` read `errors` as `field → sentence`; sign-up keys `name`, `email`, `phone`, `password`; details keys `name`, `phone`, `address`, `city` | §2.1, §2.2: the same JSON field names, and the shop's sentences are the page's own (`W.rules`) | Sound — and now guaranteed, because §2 fixes the sentences. The page shows the shop's sentence for a details refusal and its own for "email taken" (identical words) | None |
| `limited` shows the page's own sentence and reads nothing from the body | `429` with `Retry-After` | Sound | None |
| `addOrder(number, phone)` sends both **as typed**; `OrderRow` answers; `notfound` and `limited` are the two refusals the block words | §2.4: the shop normalises; `404` and `429` | Sound; the page never needs to normalise | `api.post("account/orders", {number, phone})` |
| `customerSignOut()` = `shop.signOut()` then `refreshMe()` | `POST /v1/auth/sign-out` answers `204` with no body | Needs `api.post` to accept an empty `204` (item 9 adds it) | `await signOut()` of `session.ts` (it clears the screen's session at once and tells the server, swallowing a failure), then `await refreshSession()` so the shop's answer, not the page, decides the header (SC-34) |
| **`useMe` / `refreshMe` keep their own copy of `GET /v1/auth/me`**, separate from `session.ts`'s `useAuth` | One endpoint, one truth | **Wrong once the real session lands.** `session.ts` caches its first answer. A visitor who opened `/admin` (so `useAuth` answered "nobody", and the page went to `/admin/sign-in`) and then signed in as staff or admin on `/account/sign-in` leaves `useAuth` at "nobody": `/admin` bounces to `/admin/sign-in` again, which SC-23 forbids ("without passing through `/admin/sign-in`"). A customer who has just signed up and opens `/admin` would be sent to the sign-in page instead of reading "This area is for the shop's team" (SC-40). The review's "separate caches" nit is therefore a defect, not a style point | **One cache, `session.ts`'s.** It gains one **additive** export, `refreshSession(): Promise<void>` (bump `epoch`, run the existing `check()`; it does not set `phase` back to "checking", so nothing flashes). `useMe()` becomes `const { checking, session } = useAuth(); return { checking, me: session }`; `refreshMe` is `refreshSession`; the client's own `state`, `asked`, `listeners`, `publish` and `subscribe` are deleted. `customerSignUp` and `customerSignIn` end with `await refreshSession()` on success. `useSession`, `signIn`, `signOut`, `useRole`, `allowed` and `useAuth` keep their names and meanings (story §3 item 8) |
| `SavedDetails` on a successful save draws the saved values and "Saved." and does nothing else | `PUT /v1/account` may change `name`, which the header's entry and "Hello, <name>" read from `/v1/auth/me` | **A gap**: after a rename the header shows the old name until a reload | `saveAccount` (the client file) ends with `await refreshSession()` when the answer is `ok`. No component changes |
| `useSavedDetails` calls `GET /v1/account` only when `me.role === "customer"` | §2.2 | Sound: a guest or a team member never sends it, so no `401` or `403` noise and no console error | None |
| Every call carries the cookie | `credentials: "include"` | `api.ts` sets it on every request | None |
| Calls to `/v1/account/*` that answer `unauthenticated` or `forbidden` call `refreshMe()` | A real `401` means the session ended | Sound with the real server | None |

## 5. The frontend swap, exactly

| Change | File | Why |
|---|---|---|
| Point `shop` at `src/lib/api.ts` and rewrite `ask()` as in §4; `getMe` → `api.get("auth/me")`; `signUp` → `api.post("account/sign-up", input)`; `signIn` → `api.post("auth/sign-in", {email, password})`; `getAccount` → `api.get("account")`; `putAccount` → `api.put("account", details)`; `listOrders`, `addOrder` as §4 | `src/lib/customer-accounts-sign-up-sign-in-my-orders.ts` | The mock is deleted; the file keeps its exports and their types, so **no component changes** |
| `useMe`, `refreshMe`, `customerSignOut`, `saveAccount`, `customerSignUp`, `customerSignIn` as §4 | the same file | One cache of `/v1/auth/me` |
| Add `refreshSession()` | `src/lib/session.ts` | Additive; everything the story may do to this file |
| Add `api.post<T>(path, body?)` and `api.put<T>(path, body)`: `credentials: "include"`, `Content-Type: application/json` only when a body is given, the 15-second timeout, the same `ApiError` mapping (`errors` kept), **never retried**, a `204` resolves `undefined`. If item 9's backend (or the checkout's) already added `post`, reuse it and add only `put` | `src/lib/api.ts` (additive) | A `POST` and a `PUT` are not idempotent; `api.ts` has `get` only today |
| Delete `src/lib/mock/customer-accounts-sign-up-sign-in-my-orders.ts`; point the three test files' mock path at `../lib/api` (and keep their assertions); stub `GET auth/me` where a component reads `useMe` | `src/test/customer-accounts-*.test.ts(x)`, `account-rules.test.ts` is untouched | The mock is deleted with the story's backend |

Not edited by this PR: any component or page of the story, `SiteHeader.tsx`, `Checkout.tsx` (its starting values already come from `useSavedDetailsFill`), `Order.tsx` (item 6's backend sends the cookie and no header when it holds no token), `App.tsx`, `content.json`, `theme.css`, `index.css`. The words of this story stay in `src/lib/account-words.ts` (story §4: `content.json` is frozen).

## 6. Limits, in one place

| Scope | What counts | Limit | Lock | Called by |
|---|---|---|---|---|
| `sign_up_address` | Every `POST /v1/account/sign-up` that parsed (steps 1–2 of §2.1 passed), whether it ends 201, 409 or 422; never a call refused 429 | 10 in 1 hour per address | Until the window's hour ends; `Retry-After` 1 to 3600 | `account` through `ratelimit.Hit` |
| `order_lookup_number`, `order_lookup_address` | A **miss** of `POST /v1/account/orders` or of `POST /v1/order-lookups`: wrong phone, unknown number, empty phone, an order of another account. The subject is the typed digits (a number that does not exist counts like any other). A right pair neither counts nor clears | 5 in 15 minutes per number **or** per address | 15 minutes from the fifth; `Retry-After` 1 to 900 | `orders.Attempt` |
| `sign_in_email`, `sign_in_address` | Unchanged (item 9). A customer, a staff member and the admin sign in through the same route and the same counters | 5 in 15 minutes | 15 minutes | `auth` |

The address is `httpx.ClientAddr(r)`: the right-most `X-Forwarded-For` entry. Sign-up attempts never count toward sign-in and the reverse; the three families use different scopes and therefore different rows. Nothing else of this story is limited: `GET /v1/account`, `PUT /v1/account` and the order reads need a session, and a session is the thing a stranger does not have.

## 7. Decisions, and what was rejected

| # | Question | Decision | Rejected alternative, and why |
|---|---|---|---|
| A1 | Where the number-and-phone matcher and its counters live | `internal/orders/lookup.go`, one `Attempt` function both routes call (§3.4) | A copy in `account`: two matchers drift, and the story forbids writing it twice. `account` owning the lookup: `orders` owns the table and the track story's route is in `orders`. A new `lookup` package: ~150 lines that must read `orders` rows, which would make a second reader of the table |
| A2 | Concurrent tries for one number | A transaction advisory lock on the typed digits inside `Attempt`, so the sixth try always sees five recorded failures | A second `Check` after the work (the sign-in design's answer to a slow hash): a lookup has no slow step to wait on, and a burst would still pass the first `Check` together. An in-process mutex: wrong the day two instances run |
| A3 | Which sign-up requests count against the hour | Those that **parsed** (not 415, not 400), counted **before** the field checks | Counting every byte received: a body that is not JSON creates nothing and costs a parse, and counting it would let a test of 400 and 415 use up the hour. Counting after field validation: an attacker could probe with invalid bodies for free. Counting only successes: SC-4 says a refused 422 or 409 counts |
| A4 | How an email already taken is found | The insert decides (`uq_users_email`); the password is hashed first, in every case | A `SELECT` before the insert: a race between the check and the insert, and a taken email would answer faster than a free one. A hash after the check: same timing signal |
| A5 | Account and first session | One transaction (`CreateUser` + `IssueSession`) | Two: a failure between them leaves an account whose creator saw a `500` and whose retry is a `409` |
| A6 | Default page size of `GET /v1/account/orders` | 10 (§2.3), explicit in the story's SC-8; the page also sends `limit=10` | The base's 12 with the client sending 10: SC-8 calls with no parameters and expects `"limit":10` |
| A7 | Who tells `GET /v1/orders/{id}` which customer is calling | `auth.FromContext` in the handler, an id string in `Access.AccountID`, one SQL predicate in `authorize` (A6) | `orders` reading `sessions` or calling `auth` with a cookie: a second reader of `sessions`. A middleware that loads the order into the context: it would also run on routes with no order |
| A8 | Who may own an order | The role-and-active guard **inside** the two statements that write the column (ERD I1, A8) | The handler's `Require(Customer)` alone: `Place` is reached by a public route and by whatever is written later |
| A9 | A replay of an `Idempotency-Key` | Returns the first order, never links or relinks it (I3) | Linking on replay: an order's owner would depend on who retried it last |
| A10 | `PUT /v1/account` precedence | 400 (malformed, unknown key including `id`, wrong type) before 422 (forbidden fields and limits, all at once); `{}` is a no-op `200` | Tolerating `id` as collection `PUT`s do: this route "takes no id", and a client that sends one has a bug worth a `400`. 422 first: a body with an unknown key would be answered as a validation problem |
| A11 | A lookup body with a key missing | `400` (the contract says both are "required to be present"); an **empty string** is present and is a miss | Treating a missing key as `""`: the same bytes would mean two things. Refusing an empty phone with `422`: SC-14 and ACC-006 AC-8 want `404` and one body |
| A12 | A U+0000 in a text field | `422` naming the field with its own sentence | Letting the insert fail: a `500` and a log line for something a client can send |
| A13 | Does `internal/account` hold SQL | No (§3.5) | SQL in the handlers: three packages writing `users` and `orders` |
| A14 | The message of a miss | "No order matches that number and phone." for the lookup **and** the add (the page shows its own words either way) | A message per route: item 7's page would read a different sentence for the same event |
| A15 | Sign-up while a session already exists | Allowed; the new cookie replaces the old one in the browser; the old session row lives its 7 days (no rotation, as sign-in) | Ending the old session: "two sign-ins, two rows" (item 9's SC-11) would depend on whether a cookie was sent |
| A16 | Can a seeded sample order be added | Yes, with its pair, like any guest order | A special case: the lookup already answers for a sample order (it mints a token), and a refusal here would be a second rule to test for no stated reason |
| A17 | The header's and `/admin`'s two caches of `/v1/auth/me` | One: `session.ts`'s, with an additive `refreshSession()` (§4) | Two caches kept in step by each calling the other: two places to forget. `session.ts` importing the story's client: a scaffold file knowing a story's file |

**Accepted risks** (named so a later reader does not rediscover them): **R-1** sign-up says when an email is taken (SRS ACC-001 item 4), so it can be used to test whether an address has an account; the brake is 10 an hour per address. **R-2** there is no "forgot password" and no email check (SRS open question 5): a customer who loses the password loses the account, and nobody can reset it; the owner may ask for a story. **R-3** accounts are not verified and no CAPTCHA exists, so many addresses can each make ten accounts an hour; the brakes are the four hash slots (the server cannot be made to hash faster than that) and the cost per row. **R-4** a guest order can be added by anyone who holds its number **and** phone, and once added the guest's browser keeps its token: there is no unlink (out of scope). **R-5** the per-address burst across many different numbers can exceed five tries (§3.4); each number stays at five.

## 8. Where this design corrects or departs from a merged document

| # | Document and line | Departure | Cost |
|---|---|---|---|
| C1 | `_base.md` §2.4: default `limit` 12 | `GET /v1/account/orders` defaults to **10** (SC-8) | One route with its own default; the client sends it anyway |
| C2 | `_base.md` §3.20: "`bad_request` — Invalid `page` / `limit`" | §2.4's clamp wins: `limit` above 100 is served as 100; `400` for `0`, negative, non-numeric | None |
| C3 | Item 9 design §4.1: `Hit(…) (refused bool, err error)`, SQL "same upsert shape" | Returns the wait; SQL pinned in §3.3 | If item 9's PR merged the sketch as written, this PR changes its signature (its only other caller: none) |
| C4 | Order-page design §4, step 2: "token absent → `ErrNotFound` with no query" | Absent token **and** no `AccountID` **and** no dealer | None for a guest (still no query); one statement for a customer |
| C5 | Checkout design §3.2: "the handler does not import `auth`" | `orders` imports `auth.FromContext` | None: the same design said the sign-in and accounts stories add it |
| C6 | `erd/_base.md` §8 rows 11 and 12 | Lock sentence corrected; leading `DROP INDEX CONCURRENTLY IF EXISTS` (ERD file §5) | None |
| C7 | PR #43's review nit "`useMe` keeps a separate cache" | Upgraded to a **defect** with a stated failure (SC-23, SC-40) and fixed in the BE step (§4) | Two more edits in the client file and `session.ts` |
| C8 | Item 9 design §3.1/§3.3: `HashPassword` is a package function, but the "at most 4 at once" slots belong to `auth.New`'s `Service` | Requirement R7: sign-up hashes through the same bound | If the merged code leaves `HashPassword` unbounded, this PR bounds it |
| C9 | `_base.md` §3.12, §3.21: both fields "required to be present" | A missing key is `400`; an empty string is a miss (A11) | None |

## 9. How each criterion is proved, and what the backend review checks

| Criteria | Mechanism |
|---|---|
| SC-1 to SC-4 | §2.1, §3.2, §3.3, ERD A1, A2. The hash slots (R7), the exact `Hit` SQL and a table test over attempts 1, 10, 11, 12 and a moved window |
| SC-5 | Nothing new: the shared `POST /v1/auth/sign-in` (item 9); the customer's path through it is a test of this story |
| SC-6, SC-7 | §2.2, ERD A3, A4 |
| SC-8, SC-9 | §2.3, ERD A5, A6; one `orders.WriteNotFound` |
| SC-10 | §2.5, ERD A6; byte comparison of the four 404s |
| SC-11, SC-12 | §2.5, ERD A9, invariant I1; sessions of customer, staff, admin, none and an ended one |
| SC-13 to SC-15 | §2.4, §3.4 `Attempt` and the matcher; counters shared with the lookup because they are the same code |
| SC-16 | `auth.Require` of items 9 and 10; the account routes of §1 take no id |
| SC-17 | ERD §5 and §8 |
| SC-18 to SC-40 | The UI PR (#43) for the six judged on the mock; the swap of §4 and §5 for the rest, proved against the running story |

**Notes for Test.** Until the VNPay story lands, `POST /v1/orders` never answers `503 payment_unavailable` (checkout design §3.1: the code is reserved), so the run's cases that place an order (SC-11, SC-12, SC-37, SC-38) can use the real call without keys; after it lands they need the keys, as the test cases say. The 15-minute and 1-hour windows move `rate_limits.window_started_at` and `locked_until`, never a sleep. `SC-4.5` needs the edge proxy's `X-Forwarded-For`.

**The backend review checks** (the list a reviewer opens the PR with):

1. `internal/account` imports no `pgx` and writes no SQL; no handler logs a body, a header, a password, a hash or a token; a constraint error is logged by SQLSTATE and constraint name only.
2. Sign-up's order is §2.1's: media type → body → `Hit` → fields → hash (bounded, outside the transaction) → one transaction → cookie. There is no `SELECT` of the email. A test fires 12 sign-ups with a moved window and checks 201/409/422 counted, the eleventh `429` with `Retry-After` ≤ 3600, the twelfth not extending it, and a new window after the hour.
3. The sentences of §2.1 and §2.2 equal `src/lib/account-words.ts` `W.rules` letter for letter (a test reads both).
4. `PUT /v1/account`: unknown key and wrong type `400`; `email`/`role`/`password` `422` with the other fields untouched; `{}` writes nothing; `"address": ""` clears.
5. `authorize` is still the **only** place `FROM order_tokens` is read; a customer B, a guest with no token, a wrong token and a missing id give byte-identical 404s; a guest with no token costs **no query** (a fake querier counts calls).
6. `Place` writes `account_id` only in the insert and only through the guarded subselect; a staff, an admin, a switched-off customer and an ended session store NULL; a replay with a customer session does not link.
7. `orders.Attempt`: one transaction; the advisory lock; the five misses return identical bytes; a refused call counts nothing and does not extend the lock; a success counts and clears nothing; `hit` writes nothing before `ErrNoMatch`; `grep order_lookup_` finds the two scopes **only** in `lookup.go`; the matcher is written once.
8. The migrations are exactly the ERD file's §5, with the runner notes of its §8.
9. The frontend swap is §4–§5: the client sorts a status only with its envelope code; **one** cache of `/v1/auth/me`; `refreshSession` is additive; no `localStorage` or `sessionStorage`; the mock and its imports are gone; `api.post`/`api.put` are never retried.

## 10. Build order, and what blocks the backend

PR #43 (the UI, reviewed) merges first. This story's backend then builds **on top of** the backends of two stories that are still open:

| Needs | From | If it is not on `main` when Dev starts |
|---|---|---|
| `internal/orders` (`Place`, `Read`, `authorize`, `order_tokens`, the 404 writer), `orders` columns of S5 | "Checkout and placing an order" and "Order page and Pay now" backends | Reply with the blocker, do not build them here (as item 6's backend did) |
| `internal/auth` (`Require`, `Resolve`, `CreateUser`, `IssueSession`, `HashPassword`, cookie helpers), `internal/ratelimit` (`Check`, `Fail`, `Hit`), `httpx.ClientAddr`, the real `session.ts`, `api.post` | "Dealer sign-in and protected admin area" backend | Same |

Independent of this story and free to merge in either order: "Staff accounts managed by the owner" (shares `CreateUser`; first merged wins, the second adopts) and "Track an order by number and phone" (shares `orders/lookup.go`; first merged creates it as §3.4 says, the second adopts it and reads this file rather than redesigning). The migration stamps (`20261125150000`, `20261125150100`) are later than item 9's `20261125140000`, so the files apply in order whatever order the PRs merge.
