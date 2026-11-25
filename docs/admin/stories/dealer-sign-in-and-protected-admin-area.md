# Story — Dealer sign-in and protected admin area

Module: `admin` · Plan item 8 · Requirements narrowed: ADMIN-001, ADMIN-002, ADMIN-003, ADMIN-004 (`docs/admin/SRS.md`)
Design: the approved `/admin/sign-in` split screen and the management shell (`design/design-system.md` §2.2); constraints in `docs/architecture/overview.md` and `services.md` §2.2, §2.7, §3.7–3.9.

## 1. User story

As **the shop's team** (the admin made from the owner's settings, and the staff the admin adds later), I want to sign in with my email and password and have every `/admin` page and every dealer endpoint refuse anyone else, so that the owner and staff can run the shop while strangers and customers cannot read orders or change records.

The same sign-in, sign-out and "who am I" endpoints serve a **customer** too (owner decision: three kinds of account). A customer can sign in through them and is then refused by the team's area; the customer's own screens are another story.

## 2. What exists today

- `src/lib/session.ts` is the design's mock: any email signs in, no password is checked, and the "session" is a `localStorage` key (`agentteam:session`). `SignedIn` in `src/App.tsx` opens every `/admin/*` page to anyone holding that key.
- `components/admin/SignInPage.tsx` (`SignInForm`) and `pages/AdminSignIn.tsx` draw the approved split screen and already have an Email field, a Password field, "Sign in", the note, and "Signed in as … Continue" — but nothing is checked and none of the failure states exist.
- The backend has no `users`, no `sessions`, no sign-in route, and `GET /v1/orders` is readable by anyone (the interim state `DATA-004` item 4 announces as temporary).

## 3. In scope

**Backend (`internal/auth`, `internal/ratelimit`, one migration `users_and_sessions`)**
1. `POST /v1/auth/sign-in`, `POST /v1/auth/sign-out`, `GET /v1/auth/me` as written in `services.md` §3.7–3.9, for all three roles.
2. One `users` table (role `admin`, `staff` or `customer`, `is_active`, argon2id hash) and a `sessions` table (opaque token, SHA-256 stored, 7 days fixed). At most one admin row.
3. The admin account made at start-up from `ADMIN_EMAIL` / `ADMIN_PASSWORD` (`auth.EnsureAdmin`): created, password replaced when the setting changes, skipped and logged when the password is under 10 characters or either setting is unset. The server always starts.
4. Sign-in limits: 5 failures in 15 minutes per lower-cased email **or** per client address (right-most `X-Forwarded-For`) → 429 with `Retry-After`; unknown email, wrong password and switched-off account answer identically (an argon2id verification runs for every attempt). Uses the `rate_limits` table the data story creates; this story writes the `internal/ratelimit` helper (`Check`, `Fail`, `Hit`, `Purge`) if it is not on `main` yet, and the lookup and sign-up stories reuse it.
5. `auth.Resolve` (puts the signed-in user, or none, in the request context; re-reads `users` on every request so a switched-off account or a changed role takes effect at once) and `auth.Require(roles…)` (401 with no session, 403 for another role). Every later dealer route is wrapped with it.
6. `GET /v1/orders` (the list) becomes dealer-only through `auth.Require(admin, staff)`, ending the interim public read. A dealer gets the same list shape as today; widening it (private order fields, drafts) belongs to "Manage shoes, lines, categories, reviews and orders".

**Frontend**
7. `src/lib/session.ts` rewritten over `/v1/auth/*` through the one API client (`src/lib/api.ts`, `credentials: "include"`) behind the same names: `useSession`, `signIn`, `signOut`, `useRole` (and the existing `allowed`). One additive export is allowed so the gate can tell "still asking the server" from "nobody is signed in".
8. `SignInForm` (`components/admin/SignInPage.tsx`) and `pages/AdminSignIn.tsx` rewired: real password check and the states "Signing in…", "Wrong email or password.", "We cannot reach the server. Please try again.", "Too many tries. Please wait 15 minutes.", "An email address is needed" and "Signed in as <email>. Continue". The role chooser prop of `SignInForm` goes: the server decides the role and nothing passes it.
9. The `/admin/*` gate: the `SignedIn` function in `src/App.tsx` keeps its name (no `<Route>` line changes) and works only from the answer to `GET /v1/auth/me`: nothing of the area is drawn and no redirect happens while the answer is awaited; no session → `/admin/sign-in`; admin or staff → the page; customer → the plain message "This area is for the shop's team" with a "Back to the shop" link to `/`. The gate's body lives in a new component (`src/components/admin/AdminGate.tsx`) so `App.tsx` carries only the delegation.
10. The sign-out button already in `DashboardLayout` keeps working through `signOut` with no change to that file.

**Settings and docs**
11. `ADMIN_EMAIL` and `ADMIN_PASSWORD` (already named, valueless, in `code/backend/.env.example`) stay documented there. The owner supplies both in the console (declared with `request_secret`).

## 4. Out of scope (and where it goes)

- **Product, stock, picture-upload and order work of any kind** — creating, editing, deleting shoes, lines, categories, reviews and orders, per-size counts, `POST /v1/uploads`, and the 401/403 checks on those write routes (ADMIN-004 AC-1 to AC-6, AC-9). A scope note was attached to this item by mistake and has been withdrawn; all of it is **Manage shoes, lines, categories, reviews and orders**. This story ships the guard and proves it on `GET /v1/orders` and on test routes (SC-17, SC-18).
- **Dashboard figures** (`GET /v1/admin/overview`, ADMIN-020/021) — **Dealer dashboard figures and charts**. The Overview page keeps showing what it shows today; this story only lets the right people reach it.
- **Staff accounts** — `/admin/staff`, `/v1/admin/staff`, creating, switching off and resetting staff (ADMIN-030/031) — **Staff accounts managed by the owner**. No staff screen or endpoint here; a staff row can already sign in (SC-6) and Test inserts one directly.
- **Customer sign-up, `/account/*` screens, the header's "Account" entry, linking orders to accounts** — **Customer accounts**. Until it lands no customer page gains a sign-in link (SC-40); after it, ADMIN-001 AC-13's second sentence applies.
- **Widening reads for a dealer** (draft shoes, hidden reviews, private order fields) and **order reads by token or linked customer** — "Manage shoes…", "Checkout and placing an order" (ORD-008).
- **"Your session ended. Sign in again." on a failed save** (ADMIN-011) — "Manage shoes…". Until then a data call refused for a dead session shows the data module's own error state.
- Change password, forgot password, email verification, two-step sign-in, "remember me", "sign out everywhere", sliding or refreshed sessions, returning to the page you were on after sign-in (sign-in always lands on `/admin`): not in the SRS and no screen in the design.
- Email or text messages of any kind (a sign-in alert, a reset link): the shop sends none.
- A CSRF token: `SameSite=Lax`, JSON-only bodies and side-effect-free `GET`s are the defence (`services.md` §2.2).
- **Owner-editable wording of the new state messages.** SRS open question 5 would add them to `content.json`, but `overview.md` §5 freezes `content.json` for stories (parallel stories would collide in it). The words in SC-25 to SC-30 live in the sign-in component; the existing words (heading, line, three lines, note, image alt) keep coming from `content.json` untouched. A later chore can move them if the owner wants to edit them in the Studio.
- The Studio's "signed-in" preview of admin pages writes the old `agentteam:session` key; after this story that key opens nothing (SC-34 requires it). The approved design is unchanged and nothing here edits `canvas-mode.ts`.

## 5. UI scope

The approved design is not redrawn. Screens and states this story touches:

| Screen | Approved design | Added by this story |
|---|---|---|
| `/admin/sign-in` (`pages/AdminSignIn.tsx` + `SignInForm`) | Split screen: stockroom picture left (above the form under 1024px), wordmark, heading, line, three icon lines, Email, Password, "Sign in", note; "Signed in as … Continue" | States the design never drew: signing in, wrong credentials, cannot reach the server, too many tries, "A password is needed". The error line sits under the form's fields in the management palette, is announced to assistive technology and never says which part was wrong |
| Gate on every `/admin/*` page (`AdminGate`) | Not drawn | Three outcomes: checking (**nothing** drawn, no redirect), signed out (redirect to `/admin/sign-in`), customer (the plain team-only message with "Back to the shop"). The message page is plain text in the management palette: no menu, no figures, no customer header or footer |
| Left menu foot (`DashboardLayout`) | Initial tile, name over role, sign-out icon | Unchanged file; it now shows the server's name and role and its sign-out reaches the server |
| Customer pages | Footer link "Dealer sign-in" | Nothing added: no other link to `/admin`, no "Account" entry |

Copy for the new states (SRS open question 5): "Signing in…", "Wrong email or password.", "We cannot reach the server. Please try again.", "Too many tries. Please wait 15 minutes.", "An email address is needed", "A password is needed" (this story's own wording: the SRS names only the email case), "This area is for the shop's team", "Back to the shop". English, as written.

**Mock for the UI stage.** The UI PR may use `src/lib/mock/dealer-sign-in-and-protected-admin-area.ts` standing in for the three endpoints (one sample account per role, wrong-credentials, 429 and unreachable answers on demand). It must not accept an arbitrary email and must not read or write `localStorage`. The backend PR deletes it. Criteria that need the real server (reload keeps the session, sign-out ends it on the server, a forged cookie, the 401/403 matrix) are proved at the Verify step; criteria marked **UI** can be judged on the mock.

## 6. Acceptance criteria

Tags: **BE** = proved against the API (Go tests with `TEST_DATABASE_URL`, and HTTP calls to the running story); **UI** = proved in the browser, judged at the UI review on the mock; **UI+BE** = proved in the browser against the running story. Where a test needs time to pass (15 minutes, 7 days) it moves the stored window or an injected clock, never sleeps.

### Backend

- **SC-1 [ADMIN-003 AC-1; ADMIN-001 boundary]** (BE): With `ADMIN_EMAIL=dealer@example.com` and a 12-character `ADMIN_PASSWORD`, the server starts on an empty database; `POST /v1/auth/sign-in` with them answers 200 `{"email":"dealer@example.com","name":"Dealer","role":"admin"}`, and `GET /v1/auth/me` with the cookie says role `admin`. Signing in as ` Dealer@Example.com ` (other case, surrounding spaces) reaches the same account.
- **SC-2 [ADMIN-003 AC-2]** (BE): The stored `users.password_hash` is an argon2id encoded string (`$argon2id$…`); it is not the typed password and two accounts given the same password hold different hashes (a salt per password).
- **SC-3 [ADMIN-003 AC-3, item 1]** (BE): After the server restarts with a changed `ADMIN_PASSWORD`, the old password is refused (401) and the new one signs in, and a cookie issued before the change no longer opens `GET /v1/auth/me`. A restart with a changed `ADMIN_EMAIL` changes the same row: there is still exactly one `admin` row. A restart with unchanged settings changes nothing and keeps existing sessions.
- **SC-4 [ADMIN-003 AC-4, boundary]** (BE): With a 9-character `ADMIN_PASSWORD` the server starts, makes or changes no account, logs a line saying the password is too short (never the value), and sign-in answers the generic 401. With exactly 10 characters the account is made.
- **SC-5 [ADMIN-003 AC-5, AC-10]** (BE): With `ADMIN_EMAIL` or `ADMIN_PASSWORD` unset or empty the server starts, `/healthz` is 200, no account exists and sign-in is refused for any email. `code/backend/.env.example` names both keys with no value.
- **SC-6 [ADMIN-003 AC-11, AC-12, item 7]** (BE): A `staff` row and a `customer` row inserted into `users` each sign in with their own password through the same endpoint, and `GET /v1/auth/me` says `staff` and `customer`. The name is the row's own `name`.
- **SC-7 [ADMIN-003 AC-8, AC-13; ADMIN-001 AC-7]** (BE): An unknown email, a wrong password for a real account, a switched-off (`is_active = false`) account with its right password, and any email while the admin settings are unset all answer the **same** status (401) and the **same** body (`{"error":"invalid_credentials","message":"Wrong email or password."}`); an argon2id verification runs in every one of those cases, against a fixed dummy hash when there is no usable account (proved by a unit test on the code path).
- **SC-8 [ADMIN-001 item 3; services §3.7]** (BE): A blank email answers 422 with `errors.email` "An email address is needed"; a blank password answers 422 with `errors.password`; neither counts as a failed try. A body that is not JSON-typed answers 415; a field the contract does not define, or a wrong type, answers 400.
- **SC-9 [ADMIN-003 AC-6, AC-9]** (BE): After five wrong passwords for one email, a sixth try with the **right** password answers 429 `rate_limited` with `Retry-After` in whole seconds (at most 900), and a refused call neither counts nor extends the lock. A success neither counts nor clears: four failures, a success, then one more failure locks the email. Fifteen minutes after the fifth failure the right password signs in.
- **SC-10 [ADMIN-003 AC-7]** (BE): Five failures from one client address across five different emails lock that address: a sixth try from it, for any email and even with a right password, answers 429. The address is the right-most `X-Forwarded-For` entry (else the connection's address): requests that differ only in forged left-most entries share one lock, and a different right-most entry is a different address.
- **SC-11 [ADMIN-003 Conflict]** (BE): Two sign-ins for one account at the same moment both answer 200 with different cookies and leave two `sessions` rows.
- **SC-12 [ADMIN-002 AC-1]** (BE): A successful sign-in sets exactly one cookie, `session`, with `HttpOnly`, `SameSite=Lax`, `Path=/` and `Max-Age=604800` (7 days); `Secure` is present when the request carries `X-Forwarded-Proto: https` and absent over plain HTTP. The token is at least 128 bits from a cryptographic source and does not appear in any response body.
- **SC-13 [ADMIN-002 AC-2, AC-3, AC-4, item 2]** (BE): `GET /v1/auth/me` with a live session answers 200 `{"email","name","role"}` with `Cache-Control: no-store`; with no cookie it answers 401 `unauthenticated`. The name is the account's own `name` when set; otherwise the part of the email before `@` split on `.`, `_`, `-` and capitalised (`lan.tran@example.com` → `Lan Tran`).
- **SC-14 [ADMIN-002 AC-10, AC-11, AC-14, item 6]** (BE): `POST /v1/auth/sign-out` with a session answers 204, deletes the `sessions` row and clears the cookie (`Max-Age=0`); the same old cookie then gets 401 from `GET /v1/auth/me`. With no cookie, or an unknown one, it still answers 204. A request body answers 400.
- **SC-15 [ADMIN-002 AC-13, boundary]** (BE): A session 6 days 23 hours old answers 200; exactly 7 days old, and 8 days old, answer 401 on `GET /v1/auth/me`. The expiry is fixed at sign-in and is not extended by use. The `sessions` table holds only the SHA-256 of the token; the raw token is in no column.
- **SC-16 [ADMIN-003 item 7; ADMIN-002 item 10]** (BE): A staff account signed in, then switched off (`is_active = false`), gets 401 from `GET /v1/auth/me` on its next request with the same cookie. Deleting the user ends its sessions (cascade). Changing a row's role from `staff` to `customer` takes effect on the next request (a dealer endpoint answers 403).
- **SC-17 [ADMIN-002 AC-9, AC-18; ADMIN-004 item 1, AC-8; DATA-004 item 4]** (BE): `GET /v1/orders` answers 401 with no cookie and with an unknown or forged `session` cookie, 403 for a `customer` session, and 200 for an `admin` and a `staff` session. `GET /v1/products` with no session is still 200 (public reads stay open).
- **SC-18 [ADMIN-004 AC-10, AC-11, AC-12]** (BE): Proved on routes registered by the test: `auth.Require(admin, staff)` answers none 401, customer 403, staff 200, admin 200; `auth.Require(admin)` answers none 401, staff 403, customer 403, admin 200. The 401 and 403 bodies use the error envelope (`unauthenticated`, `forbidden`) and carry no data.
- **SC-19 [ADMIN-003 data touched; `erd.md` §8 migration 7]** (BE): The migration creates `users` (unique lower-case `email`, `role` limited to `admin`, `staff`, `customer`, `is_active`, `password_hash`) and `sessions` (unique 32-byte `token_hash`, `expires_at`, cascade on the user), and refuses a second `admin` row. It applies on boot on an empty database; a second boot changes nothing; `go build`, `go vet` and `go test ./...` pass with no database (database tests skip).
- **SC-20 [Security NFR; services §7]** (BE): During failed and successful sign-ins and a sign-out, no log line and no response body contains the password, the hash, the cookie value, a token, or the email typed; a log line for the request carries method, route pattern, status and role only.

### Sign-in page

- **SC-21 [ADMIN-001 AC-1, item 10]** (UI): At 1280px `/admin/sign-in` shows the stockroom picture (alt "Boxes stacked in a stockroom") at the left and, at the right, in this order: the wordmark, the heading "Đôi — dealer sign-in", the line "Sign in to see orders, stock and sales.", the three icon lines, the form, and the note "Accounts are for the dealer and staff. Ask the owner for one." It uses the management palette and shows neither the customer header nor the footer.
- **SC-22 [ADMIN-001 AC-2; Responsive NFR]** (UI): At 390px (and 320px) the picture is above the form and the page has no horizontal scroll.
- **SC-23 [ADMIN-001 AC-3, AC-5]** (UI): "Email" shows the placeholder `you@example.com`; "Password" hides what is typed; both have visible labels; pressing Enter in either field submits the form.
- **SC-24 [ADMIN-001 AC-4; ADMIN-002 AC-16, AC-20]** (UI+BE): The right email and password of an admin account, and of a staff account, each take the address to `/admin` and the overview shows, without passing through `/admin/sign-in` again. Both then open the orders, the shoes and the overview.
- **SC-25 [ADMIN-001 AC-6, AC-7, item 5]** (UI+BE): A right email with a wrong password, and an email with no account, each show exactly "Wrong email or password." (announced to assistive technology) and nothing about which part was wrong; the password field is emptied and the email stays.
- **SC-26 [ADMIN-001 AC-8]** (UI): With the email blank, "Sign in" shows "An email address is needed" beside the email field and sends no request. With an email and a blank password it shows "A password is needed" beside the password field and sends no request.
- **SC-27 [ADMIN-001 AC-9, item 6]** (UI): While the shop takes 3 seconds to answer, the button reads "Signing in…" and is disabled (a second press sends no second request); it returns to "Sign in" when the answer arrives.
- **SC-28 [ADMIN-001 AC-10, item 7]** (UI): When the shop cannot be reached (network failure, or a 5xx answer) the page shows "We cannot reach the server. Please try again." and the button is enabled; the typed email stays.
- **SC-29 [ADMIN-001 AC-11, item 8]** (UI+BE): After five wrong tries a sixth shows "Too many tries. Please wait 15 minutes." with the button enabled.
- **SC-30 [ADMIN-001 AC-12, item 9]** (UI+BE): With a session, opening `/admin/sign-in` shows under the form "Signed in as <email>. Continue" and Continue goes to `/admin`. With no session that line is absent, and it is not drawn while `GET /v1/auth/me` is awaited.

### The gate and the session

- **SC-31 [ADMIN-002 AC-6, AC-15, item 4]** (UI+BE): With no session, each of `/admin`, `/admin/orders`, `/admin/orders/:id`, `/admin/products`, `/admin/products/:id`, `/admin/lines`, `/admin/categories` and `/admin/reviews` ends at the address `/admin/sign-in`, and none of the page's content is drawn first.
- **SC-32 [ADMIN-002 AC-7, item 3]** (UI+BE): When `GET /v1/auth/me` takes 2 seconds, an admin page opened with a session draws no menu, heading, list or figure of the area and does not redirect until the answer arrives, then draws at the same address. With no session nothing of the area is drawn during the wait either.
- **SC-33 [ADMIN-002 AC-5, item 4]** (UI+BE): A signed-in dealer on `/admin/orders` who reloads the page stays on `/admin/orders` and sees the orders list (the data calls carry the session cookie with no change to the pages).
- **SC-34 [ADMIN-002 AC-8, item 5]** (UI+BE): With a forged `agentteam:session` JSON (`email`, `name`, `role: "admin"`) in the browser's local storage and no cookie, opening `/admin` ends at `/admin/sign-in`, and `GET /v1/orders` without the cookie answers 401. `src/lib/session.ts` neither reads nor writes `localStorage` or `sessionStorage` (code review).
- **SC-35 [ADMIN-002 AC-10, AC-11, AC-12, item 6]** (UI+BE): The sign-out button at the foot of the left menu (accessible name "Sign out"), pressed on the overview, on a list page and on a record page, takes the address to `/admin/sign-in`; `GET /v1/auth/me` then answers 401 and the old cookie opens nothing. The browser's Back button does not show the admin page or its data: it shows `/admin/sign-in`.
- **SC-36 [ADMIN-002 AC-4, item 7]** (UI+BE): For the admin made from `lan.tran@example.com` the foot of the menu reads the initial `L`, the name `Lan Tran` and the role `admin`; a staff account with the name "Minh Le" reads `M`, `Minh Le`, `staff`.
- **SC-37 [ADMIN-001 item 4; ADMIN-002 AC-17, items 9, 10]** (UI+BE): A customer's right credentials entered at `/admin/sign-in` sign the customer in and the address becomes `/admin`. There, and later at `/admin/orders`, the page shows "This area is for the shop's team" and a "Back to the shop" link to `/`; no menu, list or figure of the area is drawn, and no request is sent to a dealer data endpoint (for example `GET /v1/orders`).
- **SC-38 [ADMIN-002 Upstream failure]** (UI+BE): When `GET /v1/auth/me` cannot be reached (or answers 5xx) while `/admin/orders` is opened, the area is not drawn, the visitor is not treated as signed in and `/admin/sign-in` is shown.
- **SC-39 [ADMIN-002 AC-19, item 8]** (UI): `src/lib/session.ts` still exports `useSession`, `signIn`, `signOut`, `useRole` and `allowed`. `useRole()` returns `"admin"`, `"staff"` and `"customer"` for a session of each role and `""` when nobody is signed in. The PR changes no `Admin*.tsx` page other than `AdminSignIn.tsx`, no `DashboardLayout.tsx` and no `RequireRole.tsx`; its change to `src/App.tsx` is confined to the `SignedIn` gate and its imports (no `<Route>` line changes).
- **SC-40 [ADMIN-001 AC-13]** (UI): On every customer route (`/`, `/shop`, `/shop/:id`, `/size-chart`, `/cart`, `/checkout`, `/orders/:id`, `/track`, `/shipping`, `/returns`, `/about`, `/contact`), for a visitor who is not signed in as admin or staff, the only link to `/admin` is the footer's "Dealer sign-in"; no header, bag or page element gains a sign-in link.

## 7. Dependencies

- **Must land first — Site data from the database:** the Go server skeleton with its `Register` wiring and error envelope, the `rate_limits` table (and `doi_fold` / `set_updated_at()` helpers that `users` uses), `src/lib/api.ts` (base URL, `credentials: "include"`, `ApiError`), the working `GET /v1/orders` list (SC-17) and `src/data/index.ts` (SC-33 needs the orders list).
- **Feeds:** Manage shoes, lines, categories, reviews and orders; Dealer dashboard figures and charts; Staff accounts managed by the owner; Customer accounts (all use `auth.Resolve` / `auth.Require`, the `users` table and the session). Checkout and placing an order uses `auth.Resolve` for "linked customer" and dealer order reads; it does not have to wait for this story. If it lands first and already guards the orders list, SC-17 is a regression test for its route and this story adds nothing there.
- **Owner's settings:** `ADMIN_EMAIL` and `ADMIN_PASSWORD` are declared in the console and already supplied. The story builds and its Go tests pass without them; only a real sign-in needs them.
- **No external account:** no email service, no third party.

## 8. Notes for Test, Dev and TL

- **Fixtures.** This story adds no way to create a staff or customer account in a running shop. Go tests insert rows through `auth.CreateUser`. For the running story at the Verify step, Test inserts `staff` and `customer` rows with SQL, using an argon2id hash of a known test password that the test-case document carries; the admin comes from the owner's settings.
- **Time.** The 15-minute lock and the 7-day expiry are checked by moving stored timestamps (or an injected `now`), not by waiting.
- **Contract.** Status codes, bodies, cookie and limits are those of `services.md` §2.2, §2.7 and §3.7–3.9; where this story's wording and that file differ, `services.md` wins and TL is told.
- **Size.** 40 criteria: 20 backend, 20 browser. No part of "Manage shoes…" is in here; if a review wants a criterion cut, the browser-only checks on customer pages (SC-40) are the first to go.
