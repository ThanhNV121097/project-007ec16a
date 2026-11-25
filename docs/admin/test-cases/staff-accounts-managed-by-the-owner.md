# Test cases — Staff accounts managed by the owner

Module: `admin` · Plan item 14 · Story: `docs/admin/stories/staff-accounts-managed-by-the-owner.md` (SC-1 … SC-41)
Requirements: ADMIN-030, ADMIN-031, the account-endpoint half of ADMIN-004 (`docs/admin/SRS.md` §4.4, §4.1) · Contract: `docs/architecture/services.md` §2.1, §2.2, §2.3, §2.4, §2.5, §2.7, §3.7, §3.9, §3.26, §3.27

Written before the implementation exists, from the requirements, the story and the contract only.

## 1. Risk and depth

| Part | Risk | Why | Depth |
|---|---|---|---|
| Who may call the four endpoints (SC-9, SC-10, SC-40) | **High** | Account management is the only way to mint a session that can read every customer's phone and address; a staff member who can add a staff member can add a second door | All four methods × staff, customer and no session; the guard answers before the body is read; nothing created, changed or removed |
| The admin cannot be locked out or duplicated (SC-34 … SC-37) | **High** | One wrong `PUT` or `DELETE` leaves the shop with no owner and no way back (there is no change-password screen); a `role` that is accepted anywhere makes a second admin | Every field of `PUT` against the admin's id, an invalid body, `role` on `PUT` and `POST`, customer / unknown / non-uuid ids, "exactly one admin" read back |
| Sessions end when they must (SC-24, SC-27 … SC-30, SC-32) | **High** | A person who left, or whose password was reset, must lose access at once; the sign-in race is the one place a switched-off account could keep a live cookie | Old cookie after reset, switch-off, removal; second browser; sign-in racing the switch-off; switching on does not revive |
| Passwords (SC-16, SC-21 … SC-24) | **High** | Stored secrets and a 10 / 100 boundary the SRS fixes | 9 / 10 / 100 / 101 characters, characters not bytes, never trimmed, argon2id, never in a response or a log |
| Adding and reading the team (SC-11 … SC-20) | Medium–High | Validation of three fields, uniqueness across roles, a race on one email | Every field's refusal, every boundary, several wrong fields at once, a 422 never losing typed values, two concurrent adds |
| `PUT` partial updates (SC-38) | Medium | "Exactly the fields sent": a stray write of name, active or password is silent damage | Each field alone, pairs, `email` refused with nothing changed |
| The page, menu and states (SC-1 … SC-8, SC-25 … SC-33, SC-39) | Medium | A new screen built from existing parts; the states the design never drew | Default, menu, loading, load failed, both dialogs, both questions, refusals, 390 and 320 px, staff redirect, customer and visitor gates |
| Contract rules (C-1 … C-8) | Medium | The backend is reviewed against these | Envelope, content type, 400 / 413 / 415, paging, failed write |
| Content keys and the build (SC-41) | Low | Process rules checked once | Visible copy word for word; source rules read from the PR and CI |

## 2. How to read this file

- **Paths** are written as the backend mounts them (`/v1/admin/staff`); from a browser they are `/api/v1/admin/staff`. A verifier using `fetch_url` sends `Content-Type: application/json` on every `POST` and `PUT` unless a case says otherwise.
- **Sessions** are named. `ADMIN`: the cookie from `POST /v1/auth/sign-in` with the run's `ADMIN_EMAIL` and `ADMIN_PASSWORD` (≥ 10 characters). `STAFF`: the cookie of FX-STAFF-A. `CUSTOMER`: the cookie of FX-CUSTOMER. `NONE`: no cookie. In a browser case "signs in as X" means the run types the email and password on `/admin/sign-in` and presses "Sign in" (the cookie is then the browser's own).
- **Starting state of every case**: the `users` table holds the admin row and nothing else the case does not name; no row whose email starts `tc-` and none of `lan@example.com`, `mai@example.com`. Anything a case creates is removed afterwards (`DELETE FROM users WHERE email LIKE 'tc-%' OR email IN ('lan@example.com','mai@example.com')` through `set_db_state`; sessions go by cascade), and any pre-existing staff row the case had to hide is put back, so no case depends on another. The browser cases run at 1280×800 unless a case names a size.
- **The one error body.** Every non-2xx answer is JSON `{"error":<code>,"message":<non-empty sentence>}` with `Content-Type: application/json; charset=utf-8`, plus `"errors":{<field>:<non-empty sentence>}` **only** on 400 and 422 (`services.md` §2.3). Codes used here: `unauthenticated` 401, `forbidden` 403, `not_found` 404, `conflict` 409, `invalid` 422, `bad_request` 400, `unsupported_media_type` 415, `payload_too_large` 413, `internal` 500. A body never holds SQL, a stack trace, a path, a hash or the typed password.
- **A team account in a JSON answer** is exactly `{"id","name","email","role","is_active"}`: `id` a uuid string, `is_active` a JSON boolean. Nothing else, and no key containing `pass` or `hash`.
- **Sentences from the API** (`errors.<field>`) are asserted only as non-empty English text under the right field name; TL may reword them. The two sentences the SRS fixes are asserted word for word: "The admin account cannot be switched off, changed or removed." and, at sign-in, "Wrong email or password.".
- **Browser words** are checked word for word and come from the story §5: title "Staff"; button "Add staff member"; columns Name, Email, Role, Status; roles "Admin" and "Staff"; statuses "Active" and "Switched off"; menu "Reset password", "Switch off", "Switch on", "Remove"; add dialog "Add staff member" with "Name", "Email", "Starting password", hint "At least 10 characters.", "Cancel", "Add"; reset dialog "Reset password for <name>", "New password", "Cancel", "Save", notice "Saved."; questions "Switch off <name>?" and "Remove <name>?" with "Cancel" and "Switch off" / "Remove"; "Loading the staff…", "We could not load the staff.", "Try again", "Saving…", "We could not save. Please try again.".
- **Elements are found by role and label, never by class**: the title is `main h1`; the table is `table` with `th` headers and `tbody tr` rows; a dialog or question is `[role=dialog]` (or `[role=alertdialog]`); a field is found by its visible label; "the sentence beside a field" is the text of the element the field's `aria-describedby` names, or of the element in the same field wrapper, and the field carries `aria-invalid="true"`; "announced" means that sentence sits inside an element with `role="alert"` or an `aria-live` region; a row's menu button is the one `button` in the row's last cell; the left menu is `nav`. The admin row has no `button`.
- **Browser requests stubbed or delayed.** Where a case says the shop's answer is delayed, failed, aborted or answered with a status, the verifier arranges it with request interception on the named URL inside the `interact_page` run (everything else, the cookie included, stays real). A case that needs this and finds no interception available is UNVERIFIED for that reason, not failed.
- **Sign-in failures are counted** (`services.md` §2.7: 5 failures in 15 minutes per lower-cased email **and** per client address; a success neither counts nor clears). Cases below that need a refused sign-in use an account of their own, and the verifier spaces refused sign-ins so no more than four land from one address in 15 minutes. A 429 on a case that expects 401 `invalid_credentials` is an ordering problem to re-run after the lock (≤ 900 s), not a failure of the story.
- **Concurrent requests** (SC-20, SC-30) are two `fetch_url` calls issued in the same round with the same start state; the case is repeated until the stated number of rounds has been run.

## 3. Fixtures

| Id | What is made (emails start `tc-`; passwords as written) | Used by |
|---|---|---|
| FX-ADMIN | The admin account made from `ADMIN_EMAIL` / `ADMIN_PASSWORD`; its id is read from `GET /v1/admin/staff` (the item with `role: "admin"`) | most |
| FX-STAFF-A | Made by `POST /v1/admin/staff` as ADMIN: name "Tc Staff A", `tc-sa@example.com`, password `staff-pass-A1` (13 characters), active | SC-3, SC-7, SC-9, SC-24 … SC-33, SC-38, SC-40 |
| FX-STAFF-B | The same with "Tc Staff B", `tc-sb@example.com`, `staff-pass-B1`; then `UPDATE users SET is_active = false WHERE email = 'tc-sb@example.com'` | SC-1, SC-3 |
| FX-CUSTOMER | Made as FX-STAFF-A with name "Tc Customer", `tc-customer@example.com`, password `customer-pass-1`, then `UPDATE users SET role = 'customer' WHERE email = 'tc-customer@example.com'` — this gives a customer row whose hash the backend itself wrote, so it signs in with that password (the sign-in endpoint serves any row, ADMIN-003 AC-12; no customer sign-up exists yet) | SC-8, SC-9, SC-11, SC-15, SC-37 |
| FX-NOSTAFF | Every `role = 'staff'` row is copied aside and deleted; the admin row and FX-CUSTOMER stay; the copies are put back afterwards | SC-1b |
| FX-SHOP-COUNTS | The totals of `GET /v1/orders?limit=1`, `GET /v1/products?limit=1` (dealer view, drafts included), `GET /v1/lines?limit=1`, `GET /v1/categories?limit=1`, `GET /v1/reviews?limit=1` read as ADMIN before and after | SC-33 |
| FX-UPLOAD | Only if the `uploads` table exists: `INSERT INTO uploads (name, content_type, size_bytes, data, uploaded_by) VALUES (repeat('a',32) \|\| '.png', 'image/png', 1, '\x00'::bytea, (SELECT id FROM users WHERE email = 'tc-sa@example.com'))` | SC-33 |
| E200 / E201 | An address of exactly 200 / 201 characters: `a`×60 `@` `b`×60 `.` `c`×60 `.` `d`×13 `.com` (200); the same with `d`×14 (201). A valid address by RFC (local part ≤ 64, every label ≤ 63) apart from the length rule | SC-15 |

## 4. Open points found while writing (reported, not resolved here)

1. **The admin row's name cell.** `users.name` is empty for the admin made from settings (`erd.md` §3.18) and `/v1/auth/me` derives a name from the email (ADMIN-002 item 2). The story does not say which the table shows. SC-1 expects the same name `/v1/auth/me` gives, and reads `GET /v1/admin/staff` only for the `role`, `email` and `is_active` of that row; if TL decides the cell is blank, SC-1a changes with it.
2. **Order of 404 and 422 on `PUT`.** `services.md` §3.27 fixes "the admin is checked before the body" (409) but not whether a customer's or unknown id is checked before the body. SC-37 sends a valid body to those ids (404 either way) and, for `{"role":"admin"}` sent to a customer's id, expects 404 on the reading that the target is loaded first, as it must be to find out it is the admin. If TL rules that a refused field wins, SC-37b changes and the customer's role staying `customer` is the part that cannot change.
3. **A database that refuses connections.** `services.md` §2.3 puts "a database that cannot be reached" under 500 `internal`, while the verification brief reserves 503 for a refused dependency; with no database even the session lookup fails, so the answer of the four routes is decided by `auth.Resolve`, not by this story. No case here pins it; C-7 arranges a failed *write* (500) instead.
4. **The gate's handling of a 401 in the middle of a visit** (SC-27b, SC-27c) is the dealer sign-in item's; the story flags it (§7). If the gate only asks `/v1/auth/me` at load, SC-27b fails and the finding is that item's work, while SC-27c (reload) still holds.
5. **Interception and mid-run changes.** SC-4, SC-5, SC-18c, SC-19, SC-27, SC-39 need the shop's answer delayed or failed, or an account changed between two steps of one browser run. They are written `interact_page` because nothing else operates the page; if the instrument cannot do it they come back UNVERIFIED with that reason.
6. **Two criteria are source rules.** The log half of SC-21 and the file and build half of SC-41 live in server logs and the repository, not in a response; they are `manual` and the only two.

## 5. The page, the menu and its states (browser) — SC-1 … SC-6

**Scenario**: SC-1a — the page lists the admin first, then each staff member
**Given**: FX-STAFF-A (active), FX-STAFF-B (switched off) and FX-CUSTOMER exist. The browser signs in as the admin.
**When**: The browser opens `/admin/staff` at 1280×800 and reads the page.
**Then**: `main h1` reads "Staff"; a button reads "Add staff member"; the table's `th` cells read, in order, "Name", "Email", "Role", "Status". There are exactly 3 `tbody tr`. Row 1 is the admin's: name equal to the `name` `GET /v1/auth/me` gives the admin, the `ADMIN_EMAIL` lower-cased, "Admin", "Active", and no `button` in the row. Row 2: "Tc Staff A", `tc-sa@example.com`, "Staff", "Active", one menu button. Row 3: "Tc Staff B", `tc-sb@example.com`, "Staff", "Switched off", one menu button. No row shows "Tc Customer" or `tc-customer@example.com`, and no cell holds a password or a hash.
Traces: SC-1 (ADMIN-030 AC-1, item 1)
Check: interact_page

**Scenario**: SC-1b — with no staff member the table holds only the admin's row and the button is still shown
**Given**: FX-NOSTAFF. The browser signs in as the admin.
**When**: The browser opens `/admin/staff` and reads the page.
**Then**: `main h1` reads "Staff"; the button "Add staff member" is shown and enabled; there is exactly 1 `tbody tr`, the admin's, with role "Admin", status "Active" and no `button`; the text "Loading the staff…" and "We could not load the staff." are absent.
Traces: SC-1 (ADMIN-030 AC-1, item 1)
Check: interact_page

**Scenario**: SC-2a — the admin's sidebar holds a lit "Staff" link
**Given**: The browser signs in as the admin and opens `/admin/orders` at 1280×800.
**When**: The run reads `nav` for a link named "Staff", clicks it, and reads the address, the page and the link's colours; then reads the same link's computed `background-color` and `color` and those of the "Orders" link.
**Then**: Exactly one link named "Staff" with `href` `/admin/staff` is in the sidebar and it is the last link of the menu; after the click the address is `/admin/staff` and `main h1` reads "Staff"; on `/admin/staff` the "Staff" link's computed `background-color` or `color` differs from the "Orders" link's (it is lit), and on `/admin/orders` it does not differ from the "Shoes" link's.
Traces: SC-2 (ADMIN-030 AC-12, items 1, 6)
Check: interact_page

**Scenario**: SC-2b — on a phone the admin's menu sheet holds the "Staff" link
**Given**: The browser signs in as the admin and opens `/admin/orders` at 390×844.
**When**: The run confirms no sidebar link is visible, presses the header's menu button, reads the sheet for "Staff", and clicks it.
**Then**: Before the press no link named "Staff" is visible; after it the sheet shows one link "Staff" with `href` `/admin/staff`; the click takes the address to `/admin/staff` and `main h1` reads "Staff".
Traces: SC-2 (ADMIN-030 AC-12, items 1, 6)
Check: interact_page

**Scenario**: SC-2c — a staff session's menu has no "Staff" link and no empty group
**Given**: FX-STAFF-A. The browser signs in as FX-STAFF-A.
**When**: At 1280×800 the run reads the sidebar on `/admin`, `/admin/orders` and `/admin/products`; then at 390×844 it opens the menu sheet on `/admin` and reads it.
**Then**: On all three pages and in the sheet, no `a` has the text "Staff" or the `href` `/admin/staff`; every group heading in the menu is followed, before the next heading or the end of the menu, by at least one link (no heading stands with nothing under it); the links "Orders" and "Shoes" are present.
Traces: SC-2 (ADMIN-030 AC-12, item 6)
Check: interact_page

**Scenario**: SC-3a — each staff row's menu offers Reset password, Switch off and Remove in that order; the admin's row has no menu
**Given**: FX-STAFF-A (active). The browser signs in as the admin and opens `/admin/staff`.
**When**: The run presses the menu button in the "Tc Staff A" row and reads the items; then reads the admin's row for a button.
**Then**: The menu holds exactly three items, in order: "Reset password", "Switch off", "Remove"; "Switch on" is absent; the admin's row contains no `button`.
Traces: SC-3 (ADMIN-030 item 1; ADMIN-031 item 1)
Check: interact_page

**Scenario**: SC-3b — a switched-off row offers Switch on in place of Switch off
**Given**: FX-STAFF-B (switched off). The browser signs in as the admin and opens `/admin/staff`.
**When**: The run presses the menu button in the "Tc Staff B" row and reads the items.
**Then**: Exactly three items, in order: "Reset password", "Switch on", "Remove"; "Switch off" is absent.
Traces: SC-3 (ADMIN-031 item 1)
Check: interact_page

**Scenario**: SC-3c — "Remove" is marked as dangerous
**Given**: FX-STAFF-A. The browser signs in as the admin, opens `/admin/staff` and opens the "Tc Staff A" row's menu.
**When**: The computed `color` of the menu item "Remove" is read, then that of "Reset password" and "Switch off".
**Then**: The colour of "Remove" differs from the colour of both other items (the danger tone), and "Remove" is the last item.
Traces: SC-3 (ADMIN-031 item 1)
Check: interact_page

**Scenario**: SC-4 — while the list is awaited the page says so and draws no table
**Given**: The browser signs in as the admin; the answer to `GET /api/v1/admin/staff` is delayed 3 seconds.
**When**: The browser opens `/admin/staff`; the run reads the page 1 second after the open and again 4 seconds after it.
**Then**: At 1 second the text "Loading the staff…" is visible, there is no `table`, and no empty-state text ("Nothing here yet", "No staff") and no "We could not load the staff." is shown. At 4 seconds the text is gone and the table with the admin's row is drawn.
Traces: SC-4 (ADMIN-030 AC-13, item 7)
Check: interact_page

**Scenario**: SC-5a — a server error on the list shows the load-failed state
**Given**: The browser signs in as the admin; the answer to `GET /api/v1/admin/staff` is replaced by status 500 `{"error":"internal","message":"x"}`.
**When**: The browser opens `/admin/staff` and reads the page.
**Then**: The text "We could not load the staff." and a button "Try again" are shown; there is no `table`, no row and no "Loading the staff…".
Traces: SC-5 (ADMIN-030 item 7)
Check: interact_page

**Scenario**: SC-5b — "Try again" asks again and draws the table once the shop answers
**Given**: The browser signs in as the admin; the first request to `/api/v1/admin/staff` is aborted (no network); the next ones are let through.
**When**: The browser opens `/admin/staff`, reads the page, presses "Try again" and reads the page again.
**Then**: Before the press: "We could not load the staff." and "Try again", no `table`. After it: the table with the admin's row is drawn and neither the message nor the button is shown.
Traces: SC-5 (ADMIN-030 item 7)
Check: interact_page

**Scenario**: SC-6a — no horizontal page scroll at 390 px and 320 px, and the menu is the sheet
**Given**: FX-STAFF-A and FX-STAFF-B exist. The browser signs in as the admin.
**When**: At 390×844, then at 320×640, the run opens `/admin/staff` and reads `document.documentElement.scrollWidth` and `clientWidth`, whether any sidebar link is visible, and the table's container.
**Then**: At both sizes `scrollWidth` is not greater than `clientWidth` (no horizontal page scroll); no sidebar link is visible and a menu button is; the title, the "Add staff member" button and the admin's row are visible; if the table is wider than the screen its own container scrolls (its `scrollWidth` exceeds its `clientWidth`) while the page does not.
Traces: SC-6 (ADMIN-030 item 1; Responsive NFR)
Check: interact_page

**Scenario**: SC-6b — the row menu, the add dialog and its buttons are reachable on a phone
**Given**: FX-STAFF-A exists. The browser signs in as the admin.
**When**: At 390×844 and at 320×568 the run opens `/admin/staff`, presses the "Tc Staff A" row's menu button, reads where the items are, closes it, presses "Add staff member", and reads the dialog's size and the position of "Cancel" and "Add" (scrolling the dialog's own container if it scrolls), then presses "Cancel".
**Then**: At both sizes every menu item's box lies inside the viewport horizontally; the dialog's box is not taller than the viewport or scrolls inside itself; "Cancel" and "Add" can be brought inside the viewport and "Cancel" can be clicked (the dialog closes); the page's `scrollWidth` stays not greater than `clientWidth`.
Traces: SC-6 (ADMIN-030 item 1; Responsive NFR)
Check: interact_page

## 6. Who may reach it — SC-7 … SC-10

**Scenario**: SC-7a — a staff session on `/admin/staff` is sent to `/admin` and never sees the list
**Given**: FX-STAFF-A. The browser signs in as FX-STAFF-A (the address becomes `/admin`).
**When**: The run types the address `/admin/staff` into the browser (a fresh navigation), reads the address and the page as soon as the overview shows, then reloads `/admin/staff` once more and reads the same.
**Then**: Both times the address becomes `/admin` (the history entry is replaced: Back does not return to `/admin/staff`); the overview is shown (`main h1` reads "Overview"); no `h1` reading "Staff", no button "Add staff member", no `table` with a row of "Tc Staff A" and no text `tc-sa@example.com` is ever present in the page's text at any reading, including the first one taken.
Traces: SC-7 (ADMIN-030 AC-7, item 6)
Check: interact_page

**Scenario**: SC-7b — the staff session's browser sends no request for the list
**Given**: As SC-7a, with the run's record of the requests the page makes switched on before `/admin/staff` is opened.
**When**: The run opens `/admin/staff` as in SC-7a and waits until `/admin` is shown plus 2 seconds.
**Then**: The record holds no request to `/api/v1/admin/staff` (any method) from the page; the overview's own requests may appear.
Traces: SC-7 (ADMIN-030 AC-7, item 6)
Check: interact_page

**Scenario**: SC-8a — a signed-in customer sees the team-only message and nothing of the list
**Given**: FX-CUSTOMER. The browser signs in at `/admin/sign-in` as FX-CUSTOMER (`tc-customer@example.com`, `customer-pass-1`).
**When**: The browser opens `/admin/staff` and the run reads the page after the answer to `GET /api/v1/auth/me` has arrived.
**Then**: The text "This area is for the shop's team" is shown with a link "Back to the shop" whose `href` is `/`; there is no `h1` "Staff", no button "Add staff member", no `table`, no figure, no sidebar link, and no request to `/api/v1/admin/staff` appears in the page's request record.
Traces: SC-8 (ADMIN-030 AC-9, item 6; ADMIN-002 item 9)
Check: interact_page

**Scenario**: SC-8b — a visitor with no session goes to the sign-in page and sees nothing of the area
**Given**: No session; the browser's storage holds nothing.
**When**: The browser opens `/admin/staff` and the run reads the address and the page at 0.5 s and at 3 s.
**Then**: The address becomes `/admin/sign-in`; at no reading is "Staff", "Add staff member", a table, a row or a team email shown; the sign-in form is.
Traces: SC-8 (ADMIN-030 AC-9, item 6; ADMIN-002 item 9)
Check: interact_page

**Scenario**: SC-9a — a staff session is refused by all four endpoints and nothing changes
**Given**: FX-STAFF-A and a second staff member FX-VICTIM (`tc-victim@example.com`, "Tc Victim", `victim-pass-1`) exist; the session is STAFF (FX-STAFF-A).
**When**: STAFF sends, in turn: `GET /v1/admin/staff`; `POST /v1/admin/staff` with `{"name":"Tc Denied","email":"tc-denied@example.com","password":"denied-pass-1"}`; `PUT /v1/admin/staff/<FX-VICTIM id>` with `{"is_active":false}`; `DELETE /v1/admin/staff/<FX-VICTIM id>`; then the guard-first probes: `POST /v1/admin/staff` with the invalid body `{"nope":1}`, `PUT /v1/admin/staff/<FX-ADMIN id>` with `{"is_active":false}`, `PUT /v1/admin/staff/00000000-0000-0000-0000-000000000000` with `{}`, and `DELETE /v1/admin/staff/not-a-uuid`. Then ADMIN reads `GET /v1/admin/staff` and FX-VICTIM signs in.
**Then**: Every one of the eight answers is 403 with body `{"error":"forbidden","message":<non-empty>}` and no `errors` key (never 400, 404, 409 or 422: the guard answers before the body or the id is read). The admin's read shows no `tc-denied@example.com`, FX-VICTIM still `is_active: true` with the name "Tc Victim", and FX-VICTIM's sign-in answers 200.
Traces: SC-9 (ADMIN-030 AC-8; ADMIN-031 AC-11; ADMIN-004 AC-11)
Check: fetch_url

**Scenario**: SC-9b — a customer session is refused by all four endpoints and nothing changes
**Given**: FX-CUSTOMER and FX-VICTIM exist; the session is CUSTOMER.
**When**: CUSTOMER sends the same eight requests as SC-9a; then ADMIN reads `GET /v1/admin/staff`, and FX-CUSTOMER signs in again.
**Then**: All eight answer 403 `{"error":"forbidden","message":…}` with no `errors` key; no `tc-denied@example.com` exists; FX-VICTIM is unchanged and active; FX-CUSTOMER still has role `customer` and still signs in (`GET /v1/auth/me` says `customer`).
Traces: SC-9 (ADMIN-030 AC-9; ADMIN-031 AC-11; ADMIN-004 AC-11)
Check: fetch_url

**Scenario**: SC-10 — with no session each of the four calls is 401
**Given**: FX-VICTIM exists; the session is NONE, and then a made-up cookie `session=0000000000000000000000000000000000000000000`.
**When**: Each of the eight requests of SC-9a is sent once with no cookie and once with the made-up cookie.
**Then**: All sixteen answer 401 `{"error":"unauthenticated","message":<non-empty>}` with no `errors` key; the admin's read afterwards shows FX-VICTIM unchanged and no `tc-denied@example.com`.
Traces: SC-10 (ADMIN-030 AC-10; ADMIN-004 AC-12; ADMIN-031 "Not permitted")
Check: fetch_url

## 7. Reading the team — SC-11

**Scenario**: SC-11a — the list has the contract's shape, the admin first, no customer
**Given**: FX-STAFF-A is made, then FX-STAFF-B a moment later, and FX-CUSTOMER exists; the session is ADMIN.
**When**: ADMIN sends `GET /v1/admin/staff?limit=100`.
**Then**: 200 `Content-Type: application/json; charset=utf-8` with keys `items`, `total`, `page`, `limit` and nothing else; `page` 1, `limit` 100, `total` 3, `items` of length 3 in the order admin, `tc-sa@example.com`, `tc-sb@example.com`. Each item has exactly the keys `id`, `name`, `email`, `role`, `is_active`: `id` a uuid string, `role` `"admin"` for the first and `"staff"` for the others, `is_active` a JSON boolean (`true`, `true`, `false`); no item has `role` `customer` and `tc-customer@example.com` is absent from the whole body.
Traces: SC-11 (ADMIN-030 AC-11, items 4, 5)
Check: fetch_url

**Scenario**: SC-11b — paging of the team list
**Given**: As SC-11a; the session is ADMIN.
**When**: ADMIN sends `GET /v1/admin/staff?limit=1`, `?limit=1&page=2`, `?limit=1&page=3`, `?limit=1&page=4`, and no parameters at all.
**Then**: The first three answer 200 with one item each — the admin, then `tc-sa@example.com`, then `tc-sb@example.com` — and `total` 3 on every page; the fourth answers 200 with `items: []` and `total` 3; the call with no parameters answers 200 with `limit` 12 and 3 items.
Traces: SC-11 (ADMIN-030 item 4); contract (GET /v1/admin/staff, §2.4)
Check: fetch_url

**Scenario**: SC-11c — no password, hash or field named like one in any response
**Given**: FX-STAFF-A with password `staff-pass-A1`; the session is ADMIN.
**When**: ADMIN sends `GET /v1/admin/staff`; `POST` with a valid body (password `leak-check-pass-1`); `PUT` on FX-STAFF-A with `{"password":"leak-check-pass-2"}`; `PUT` with `{"name":"Tc Renamed"}`; `PUT` with `{"password":"zq9-tiny"}` (422); `POST` with an email already taken (422); `PUT` on the admin's id (409); `DELETE` of the new account (204). STAFF also sends the list (403).
**Then**: No response body, at any status, contains the strings `staff-pass-A1`, `leak-check-pass-1`, `leak-check-pass-2`, `zq9-tiny`, `$argon2`, `argon2id`, or a key containing `pass` or `hash` outside the `errors` object's own field names; the 422 for the short password has the field name `password` only as a key of `errors`, with a sentence that does not repeat the typed value; no response sets a cookie.
Traces: SC-11 (ADMIN-030 AC-11, item 5)
Check: fetch_url

## 8. Adding a staff member — SC-12 … SC-21

**Scenario**: SC-12a — the add dialog has three labelled fields, a password hint and two buttons
**Given**: The browser signs in as the admin and opens `/admin/staff`.
**When**: The run presses "Add staff member" and reads the dialog.
**Then**: A `[role=dialog]` titled "Add staff member" is shown with three fields found by their visible labels "Name", "Email" and "Starting password"; the "Starting password" input has `type="password"`; the text "At least 10 characters." is shown next to it; the buttons are "Cancel" and "Add", and "Add" is enabled; all three fields are empty and none is marked invalid.
Traces: SC-12 (ADMIN-030 item 2)
Check: interact_page

**Scenario**: SC-12b — Cancel and Escape close the dialog, send nothing and the next opening is empty
**Given**: FX-STAFF-A exists. The browser signs in as the admin and opens `/admin/staff`; the run records the table's rows and `GET /v1/admin/staff`'s `total` (2).
**When**: The run presses "Add staff member", types "Tc Typed", `tc-typed@example.com` and `typed-pass-12`, presses "Cancel", and reads the page; presses "Add staff member" again and reads the fields; types the same three values, presses Escape, and reads the page; ADMIN then reads `GET /v1/admin/staff`.
**Then**: After "Cancel" and after Escape the dialog is gone and the table still holds exactly the same two rows; the second opening shows all three fields empty; no request with method `POST` to `/api/v1/admin/staff` was sent (the request record) and ADMIN's read has `total` 2 and no `tc-typed@example.com`.
Traces: SC-12 (ADMIN-030 item 2)
Check: interact_page

**Scenario**: SC-13a — adding "Lan Tran" from the page closes the dialog and shows the row without a reload
**Given**: The browser signs in as the admin, opens `/admin/staff` and sets `window.name = "tc-marker"` (or any value that a reload would lose); no staff exist (FX-NOSTAFF).
**When**: The run presses "Add staff member", types "Lan Tran", `lan@example.com` and a 12-character password `lan-pass-0012`, and presses "Add".
**Then**: The request `POST /api/v1/admin/staff` answers 201; the dialog closes; the table shows a row with the cells "Lan Tran", `lan@example.com`, "Staff", "Active" after the admin's row; the page was not reloaded (the marker survives); "Saved." is not shown.
Traces: SC-13 (ADMIN-030 AC-2, item 4)
Check: interact_page

**Scenario**: SC-13b — `POST` answers 201 with the account and stores the email in lower case
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with `{"name":"Mai Le","email":"Mai@Example.com","password":"mai-pass-0012"}`, then `GET /v1/admin/staff`.
**Then**: 201 `Content-Type: application/json; charset=utf-8` with exactly `{"id":<uuid>,"name":"Mai Le","email":"mai@example.com","role":"staff","is_active":true}`; the list holds that account once with `email` `mai@example.com`, and no item whose email contains an upper-case letter.
Traces: SC-13 (ADMIN-030 AC-2, item 4); contract (POST /v1/admin/staff)
Check: fetch_url

**Scenario**: SC-13c — text is trimmed
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with `{"name":"  Lan Tran  ","email":"  lan@example.com  ","password":"lan-pass-0012"}`.
**Then**: 201 with `name` "Lan Tran" and `email` `lan@example.com` (no surrounding spaces); signing in with `lan@example.com` and `lan-pass-0012` answers 200.
Traces: SC-13 (ADMIN-030 AC-2); contract (§2.5 text is trimmed)
Check: fetch_url

**Scenario**: SC-14a — the staff member just added signs in on the sign-in page and reaches `/admin`
**Given**: ADMIN has created "Lan Tran", `lan@example.com`, `lan-pass-0012` through the API.
**When**: A fresh browser opens `/admin/sign-in`, types the email and password, presses "Sign in", and reads the address, the page and the response of `GET /api/v1/auth/me` made by the page.
**Then**: The address becomes `/admin`; the overview shows (`main h1` reads "Overview"); `GET /api/v1/auth/me` answers 200 `{"email":"lan@example.com","name":"Lan Tran","role":"staff"}`; the left menu's foot reads "Lan Tran".
Traces: SC-14 (ADMIN-030 AC-3, item 3; ADMIN-002 AC-16)
Check: interact_page

**Scenario**: SC-14b — the new account signs in through the API at once
**Given**: As SC-14a, with nothing else done between the creation and the sign-in.
**When**: `POST /v1/auth/sign-in` with `{"email":"Lan@Example.com","password":"lan-pass-0012"}`, then `GET /v1/auth/me` with the cookie it sets, then `GET /v1/admin/staff` with that cookie.
**Then**: Sign-in answers 200 `{"email":"lan@example.com","name":"Lan Tran","role":"staff"}` and sets a cookie named `session` that is `HttpOnly` with `SameSite=Lax`, `Path=/` and `Max-Age=604800`; `GET /v1/auth/me` answers 200 with the same three fields; the list answers 403.
Traces: SC-14 (ADMIN-030 AC-3, item 3; ADMIN-002 AC-16)
Check: fetch_url

**Scenario**: SC-15a — an email that is taken, empty, malformed or too long is refused with 422 naming `email`
**Given**: ADMIN has created `lan@example.com`; FX-CUSTOMER (`tc-customer@example.com`) and the admin (`ADMIN_EMAIL`) exist; the session is ADMIN; the team's `total` is read first.
**When**: ADMIN sends `POST /v1/admin/staff` with a valid name and password and each of these emails in turn: `lan@example.com`; `Lan@Example.com`; ` lan@example.com ` (surrounding spaces); `ADMIN_EMAIL` with its letters upper-cased; `tc-customer@example.com`; `TC-Customer@Example.com`; `""`; `"   "`; `not-an-email`; `lan@`; `@example.com`; `lan @example.com`; E201; and, with the field left out of the body altogether, no `email` key.
**Then**: Every one answers 422 `{"error":"invalid","message":<non-empty>,"errors":{"email":<non-empty sentence>}}` and no other key in `errors` (name and password were valid); the team list's `total` is unchanged after each, and no customer or admin account changed.
Traces: SC-15 (ADMIN-030 AC-4, item 2, "Invalid input"); contract (§3.26 taken email is 422, not 409)
Check: fetch_url

**Scenario**: SC-15b — the longest allowed email is accepted
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with a valid name and password and the email E200 (exactly 200 characters).
**Then**: 201 with `email` equal to E200 (lower case, as sent) and `role` `staff`; the account appears in the list; signing in with E200 and the password answers 200.
Traces: SC-15 (ADMIN-030 AC-4; boundary ≤ 200)
Check: fetch_url

**Scenario**: SC-15c — a refused email in the dialog shows its sentence beside Email and keeps the dialog open
**Given**: ADMIN has created `lan@example.com`. The browser signs in as the admin and opens `/admin/staff`.
**When**: The run presses "Add staff member", types "Lan Again", `Lan@Example.com` and `again-pass-12`, presses "Add", and reads the dialog; then clears Email, presses "Add", and reads it; then types `not-an-email`, presses "Add", and reads it.
**Then**: Each time the dialog stays open; a non-empty sentence is shown beside the "Email" field and the field has `aria-invalid="true"`; no sentence is shown beside "Name" or "Starting password"; the values typed in Name and Starting password are still there; the table still has the two rows it had (admin, Lan Tran).
Traces: SC-15 (ADMIN-030 AC-4, item 2)
Check: interact_page

**Scenario**: SC-16a — password boundary: 9 refused, 10 and 100 accepted, 101 refused
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with a valid name, a fresh email each time and passwords of exactly 9, 10, 100 and 101 characters (letters and digits).
**Then**: 9 → 422 `{"error":"invalid",…,"errors":{"password":<non-empty>}}` and no account; 10 → 201; 100 → 201; 101 → 422 naming `password` and no account. After the four calls the list holds exactly the two accounts made by the 10- and 100-character calls (plus the admin). The 422 bodies do not contain the typed password.
Traces: SC-16 (ADMIN-030 AC-5; boundary)
Check: fetch_url

**Scenario**: SC-16b — the limits count characters, and a password is never trimmed
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with valid names and fresh emails and these passwords: 9 × `é`; 10 × `é` (20 bytes); 100 × `é` (200 bytes); 101 × `é`; `abcdefghi ` (nine letters and one trailing space, 10 characters). Then each accepted account signs in with exactly the string it was made with, and the last one also with `abcdefghi` (no trailing space).
**Then**: 9 × `é` → 422 naming `password`; 10 × `é` → 201; 100 × `é` → 201; 101 × `é` → 422; `abcdefghi ` → 201. The sign-ins with the exact strings answer 200; the sign-in with `abcdefghi` (trimmed) answers 401 `invalid_credentials`.
Traces: SC-16 (ADMIN-030 AC-5; boundary); contract (§2.5 characters not bytes, passwords never trimmed)
Check: fetch_url

**Scenario**: SC-16c — a refused password in the dialog keeps the dialog open and the other values
**Given**: The browser signs in as the admin and opens `/admin/staff`.
**When**: The run presses "Add staff member", types "Tc Short", `tc-short@example.com` and a 9-character password, presses "Add", reads the dialog; then retypes the password with 10 characters and presses "Add".
**Then**: After the first press the dialog is open, a non-empty sentence is beside "Starting password" and the field is `aria-invalid="true"`, Name still reads "Tc Short" and Email still reads `tc-short@example.com`, and no row was added; after the second press the dialog closes and a row "Tc Short", `tc-short@example.com`, "Staff", "Active" appears.
Traces: SC-16 (ADMIN-030 AC-5, item 2)
Check: interact_page

**Scenario**: SC-17a — name rules: empty, spaces and 101 characters refused; 100 accepted
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with a valid email (fresh each time) and password and these names: `""`; `"     "`; no `name` key; 101 × `a`; 100 × `a`; 100 × `Đ` (200 bytes); 101 × `Đ`.
**Then**: The first three and the 101-character ones answer 422 `{"error":"invalid",…,"errors":{"name":<non-empty>}}` with no other key and create nothing; the 100 × `a` and 100 × `Đ` calls answer 201 with the name unchanged.
Traces: SC-17 (ADMIN-030 AC-6, item 2; boundary); contract (§2.5 characters not bytes)
Check: fetch_url

**Scenario**: SC-17b — "Add" with the name empty or only spaces keeps the other fields
**Given**: The browser signs in as the admin and opens `/admin/staff`.
**When**: The run presses "Add staff member", types `tc-noname@example.com` and `noname-pass-12` leaving Name empty, presses "Add", reads the dialog; then types five spaces in Name, presses "Add", reads it again.
**Then**: Both times the dialog stays open, a non-empty sentence is beside "Name" and Name is `aria-invalid="true"`; Email still reads `tc-noname@example.com`; Starting password still holds a 14-character value; no row was added.
Traces: SC-17 (ADMIN-030 AC-6, item 2)
Check: interact_page

**Scenario**: SC-18a — one 422 names every field at fault
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with `{"name":"","email":"nope","password":"abc"}`, then with `{}`, then with `{"name":"Ok","email":"ok@example.com","password":"abc"}`.
**Then**: The first answers 422 with `errors` holding exactly the keys `name`, `email` and `password`, each a non-empty sentence; the second the same three keys; the third only `password`. No account is created by any of them.
Traces: SC-18 (ADMIN-030 item 2; Invalid input row)
Check: fetch_url

**Scenario**: SC-18b — several wrong fields in the dialog: a sentence beside each, fields marked invalid, values kept, sentences announced
**Given**: The browser signs in as the admin and opens `/admin/staff`.
**When**: The run presses "Add staff member", types a space in Name, `nope` in Email and `abc` in Starting password, presses "Add", and reads the dialog and the live regions.
**Then**: The dialog stays open; each of the three fields has its own non-empty sentence beside it and `aria-invalid="true"`; the three sentences sit inside an element with `role="alert"` or an `aria-live` region; Email still reads `nope` and Starting password still holds 3 characters (the password too is kept); no row was added.
Traces: SC-18 (ADMIN-030 item 2; Accessibility NFR)
Check: interact_page

**Scenario**: SC-18c — while the request is in flight "Add" says "Saving…" and is disabled, so a double press adds one account
**Given**: The browser signs in as the admin and opens `/admin/staff`; the answer to `POST /api/v1/admin/staff` is delayed 3 seconds.
**When**: The run presses "Add staff member", types "Tc Twice", `tc-twice@example.com` and `twice-pass-12`, presses "Add", reads the button 0.5 s later, presses it a second time (and a third time by pressing Enter in the Email field), waits 4 seconds and reads the page; ADMIN reads `GET /v1/admin/staff`.
**Then**: At 0.5 s the submit button reads "Saving…" and is `disabled`; after the wait the dialog is closed and the table shows exactly one "Tc Twice" row; ADMIN's read holds exactly one account with `tc-twice@example.com` and a `total` one higher than before.
Traces: SC-18 (ADMIN-030 item 2; item 7)
Check: interact_page

**Scenario**: SC-19 — with the shop unreachable "Add" keeps the dialog and the values; pressing it again creates the account once
**Given**: The browser signs in as the admin and opens `/admin/staff`; the request `POST /api/v1/admin/staff` is aborted (no network) for the first press and let through afterwards.
**When**: The run presses "Add staff member", types "Tc Retry", `tc-retry@example.com` and `retry-pass-12`, presses "Add", reads the dialog and the table; then presses "Add" again and reads the page; ADMIN reads `GET /v1/admin/staff`.
**Then**: After the first press the dialog is still open; the text "We could not save. Please try again." is shown inside it (announced: in a `role="alert"` or `aria-live` element); Name, Email and Starting password still hold what was typed; "Add" is enabled again and the table has not changed. After the second press the dialog closes, one "Tc Retry" row is shown, and ADMIN's read holds exactly one account with `tc-retry@example.com`.
Traces: SC-19 (ADMIN-030 AC-14, item 7)
Check: interact_page

**Scenario**: SC-20 — two adds of one email at the same moment: one 201, one 422 naming `email`
**Given**: The session is ADMIN; five fresh emails `tc-race-1@example.com` … `tc-race-5@example.com`.
**When**: For each email, in one round, ADMIN sends two `POST /v1/admin/staff` requests with that email (names "Tc Race A" and "Tc Race B", valid passwords) at the same moment; afterwards ADMIN reads `GET /v1/admin/staff` and the verifier runs `SELECT email, count(*) FROM users WHERE email LIKE 'tc-race-%' GROUP BY email` through `set_db_state`.
**Then**: For every email exactly one of the two answers is 201 and the other is 422 with `errors.email` (never two 201, never a 500, never a 409); the list holds each email once and the SQL shows a count of 1 for each.
Traces: SC-20 (ADMIN-030 "Conflict")
Check: fetch_url

**Scenario**: SC-21a — the stored password is an argon2id hash that is salted and never returned
**Given**: The session is ADMIN. Two accounts are made by `POST /v1/admin/staff` with the same password `same-pass-0012`: `tc-h1@example.com` and `tc-h2@example.com`.
**When**: The verifier reads `SELECT email, password_hash FROM users WHERE email IN ('tc-h1@example.com','tc-h2@example.com')` through `set_db_state`; the two creation responses and `GET /v1/admin/staff` are read.
**Then**: Each `password_hash` starts with `$argon2id$` and is not `same-pass-0012`; the two hashes differ from each other; no response body (the two 201s and the list) contains `same-pass-0012`, `$argon2` or the stored hash text.
Traces: SC-21 (ADMIN-030 item 5; ADMIN-003 AC-2)
Check: fetch_url

**Scenario**: SC-21b — neither the typed password nor the hash is written to the server's log
**Given**: A creation of an account with password `log-probe-pass-9` has been made, and one reset, one refused add (422) and one sign-in with it.
**When**: The server's log lines for those four requests (found by their `X-Request-Id`) are read.
**Then**: No line contains `log-probe-pass-9` or `$argon2id$`.
Traces: SC-21 (ADMIN-030 item 5)
Check: manual — no instrument available to a verifier reads the backend's log output; the requests are made with `fetch_url` and a person reads the log lines for their request ids

## 9. Reset a password — SC-22 … SC-24

**Scenario**: SC-22a — the reset dialog and its "Saved." notice
**Given**: FX-STAFF-A exists (name "Tc Staff A", password A `staff-pass-A1`). The browser signs in as the admin and opens `/admin/staff`.
**When**: The run opens the "Tc Staff A" row's menu and presses "Reset password"; reads the dialog; presses "Cancel" and reads the page; opens the dialog again, types the 12-character password B `staff-pass-B2` in "New password" and presses "Save"; reads the page.
**Then**: The dialog is titled "Reset password for Tc Staff A", has a field found by the label "New password" and the buttons "Cancel" and "Save"; "Cancel" closes it without a `PUT` request and without "Saved."; after "Save" the dialog closes, the text "Saved." is shown above the table, and the table still holds the same rows.
Traces: SC-22 (ADMIN-031 AC-1, item 2)
Check: interact_page

**Scenario**: SC-22b — after a reset the old password is refused and the new one signs in
**Given**: FX-STAFF-A (password A `staff-pass-A1`); the session is ADMIN.
**When**: ADMIN sends `PUT /v1/admin/staff/<FX-STAFF-A id>` with `{"password":"staff-pass-B2"}`; then `POST /v1/auth/sign-in` with the email and A; then with the email and B.
**Then**: The `PUT` answers 200 with exactly `{"id","name":"Tc Staff A","email":"tc-sa@example.com","role":"staff","is_active":true}` and no password or hash; sign-in with A answers 401 `{"error":"invalid_credentials","message":"Wrong email or password."}`; sign-in with B answers 200 with `role` `staff`.
Traces: SC-22 (ADMIN-031 AC-1, item 2)
Check: fetch_url

**Scenario**: SC-23a — a new password of 9, 0 or 101 characters is refused with 422 naming `password` and the old one still works
**Given**: FX-STAFF-A (password A); the session is ADMIN; the team's list is read first.
**When**: ADMIN sends `PUT /v1/admin/staff/<id>` with `{"password":<9 characters>}`, `{"password":""}` and `{"password":<101 characters>}`; then STAFF's cookie (taken before) is used on `GET /v1/auth/me`; then `POST /v1/auth/sign-in` with A.
**Then**: All three `PUT`s answer 422 `{"error":"invalid",…,"errors":{"password":<non-empty>}}`; `GET /v1/auth/me` with the earlier cookie still answers 200 (a refused reset ends no session); sign-in with A answers 200; the list is unchanged.
Traces: SC-23 (ADMIN-031 AC-2)
Check: fetch_url

**Scenario**: SC-23b — a refused password in the reset dialog stays open with its sentence and shows no "Saved."
**Given**: FX-STAFF-A exists. The browser signs in as the admin and opens `/admin/staff`.
**When**: The run opens "Reset password" for "Tc Staff A", types a 9-character password, presses "Save", and reads the page; then clears the field and presses "Save" again.
**Then**: Each time the dialog stays open, a non-empty sentence is beside "New password" and the field is `aria-invalid="true"`; the text "Saved." is not shown anywhere; the table is unchanged. FX-STAFF-A still signs in with A (`POST /v1/auth/sign-in` answers 200).
Traces: SC-23 (ADMIN-031 AC-2, item 2)
Check: interact_page

**Scenario**: SC-24 — a staff member's open session ends when the password is reset
**Given**: FX-STAFF-A with two sessions S1 and S2 (two sign-ins with password A); the session is ADMIN.
**When**: ADMIN sends `PUT /v1/admin/staff/<id>` with `{"password":"staff-pass-B2"}`; then S1 sends `GET /v1/auth/me` and `GET /v1/orders`, and S2 sends `GET /v1/auth/me`; then the staff member signs in with B and sends `GET /v1/auth/me` and `GET /v1/orders`.
**Then**: Every request of S1 and S2 after the reset answers 401 `{"error":"unauthenticated","message":<non-empty>}`; sign-in with B answers 200 and the new cookie answers 200 on both endpoints.
Traces: SC-24 (ADMIN-031 "Data touched", item 1)
Check: fetch_url

## 10. Switch an account off and on — SC-25 … SC-30

**Scenario**: SC-25 — the switch-off question can be cancelled with the button or Escape
**Given**: FX-STAFF-A (active). The browser signs in as the admin and opens `/admin/staff`.
**When**: The run opens the "Tc Staff A" row's menu, presses "Switch off", reads the question; presses "Cancel" and reads the row; opens the question again and presses Escape and reads the row; ADMIN then reads `GET /v1/admin/staff`.
**Then**: The question is a `[role=dialog]` or `[role=alertdialog]` titled "Switch off Tc Staff A?" with a one-line note and the buttons "Cancel" and "Switch off"; after "Cancel" and after Escape it is gone, the row still reads "Active" and no `PUT` request was sent; the read shows `is_active: true` for FX-STAFF-A.
Traces: SC-25 (ADMIN-031 AC-12, item 3)
Check: interact_page

**Scenario**: SC-26a — confirming "Switch off" redraws the row as "Switched off" with "Switch on" in its menu
**Given**: FX-STAFF-A (active). The browser signs in as the admin and opens `/admin/staff`.
**When**: The run opens the row's menu, presses "Switch off", presses the question's "Switch off", reads the row, then opens the row's menu.
**Then**: The request is `PUT /api/v1/admin/staff/<id>` with the body `{"is_active":false}` and answers 200; the question closes; the row reads "Switched off" in its Status cell; the menu offers "Reset password", "Switch on", "Remove" (no "Switch off").
Traces: SC-26 (ADMIN-031 AC-3, item 3)
Check: interact_page

**Scenario**: SC-26b — `PUT {"is_active":false}` answers with the account switched off
**Given**: FX-STAFF-A (active); the session is ADMIN.
**When**: ADMIN sends `PUT /v1/admin/staff/<id>` with `{"is_active":false}`, then `GET /v1/admin/staff`.
**Then**: 200 with exactly `{"id","name":"Tc Staff A","email":"tc-sa@example.com","role":"staff","is_active":false}`; the list shows the same account with `is_active: false` and the other fields unchanged.
Traces: SC-26 (ADMIN-031 AC-3, item 3); contract (PUT /v1/admin/staff/{id})
Check: fetch_url

**Scenario**: SC-27a — a switch-off ends every request of the person's earlier sessions
**Given**: FX-STAFF-A signed in from two places: cookies S1 and S2; the session is ADMIN.
**When**: ADMIN sends `PUT /v1/admin/staff/<id>` with `{"is_active":false}`; then S1 sends `GET /v1/auth/me`, `GET /v1/orders` and `GET /v1/admin/overview`; S2 sends `GET /v1/auth/me` and `POST /v1/auth/sign-out`.
**Then**: Every one answers 401 `{"error":"unauthenticated","message":<non-empty>}` (the sign-out answers 204, as it always does); none answers 200 or 403.
Traces: SC-27 (ADMIN-031 AC-3, item 3)
Check: fetch_url

**Scenario**: SC-27b — an open admin page of a switched-off person ends on the sign-in page at its next request
**Given**: The browser signs in as FX-STAFF-A and opens `/admin/orders`; then, between two steps of the run, the account is switched off (`PUT {"is_active":false}` sent with the ADMIN cookie, or `UPDATE users SET is_active = false WHERE email = 'tc-sa@example.com'`, which every request re-reads).
**When**: The run clicks the sidebar's "Shoes" link (a client-side navigation that asks the shop for data) and reads the address after 3 seconds.
**Then**: The address is `/admin/sign-in` and no shoe, order or figure is shown.
Traces: SC-27 (ADMIN-031 AC-3, item 3)
Check: interact_page

**Scenario**: SC-27c — a reload of an admin page by a switched-off person lands on the sign-in page
**Given**: As SC-27b, the account switched off between two steps.
**When**: The run reloads `/admin/orders` and reads the address and page after 3 seconds.
**Then**: The address is `/admin/sign-in`; nothing of the area is shown.
Traces: SC-27 (ADMIN-031 AC-3, item 3)
Check: interact_page

**Scenario**: SC-28a — a switched-off account's right password is refused exactly like a wrong one
**Given**: FX-STAFF-A switched off (SC-26b); a second active account FX-VICTIM (`tc-victim@example.com`, `victim-pass-1`) for the comparison.
**When**: `POST /v1/auth/sign-in` is sent with `{"email":"tc-sa@example.com","password":"staff-pass-A1"}` (right password, account off) and with `{"email":"tc-victim@example.com","password":"wrong-password-1"}` (wrong password); the two responses are compared status, headers `Content-Type` and body byte for byte, and neither sets a cookie.
**Then**: Both answer 401 with the identical body `{"error":"invalid_credentials","message":"Wrong email or password."}`; neither has a `Set-Cookie` header.
Traces: SC-28 (ADMIN-031 AC-4; ADMIN-003 AC-13)
Check: fetch_url

**Scenario**: SC-28b — the sign-in page shows "Wrong email or password." for a switched-off account
**Given**: FX-STAFF-A switched off.
**When**: A fresh browser opens `/admin/sign-in`, types `tc-sa@example.com` and `staff-pass-A1` and presses "Sign in".
**Then**: The address stays `/admin/sign-in`, the text "Wrong email or password." is shown, the password field is empty, and nothing says the account is switched off.
Traces: SC-28 (ADMIN-031 AC-4; ADMIN-003 AC-13; ADMIN-001 AC-6)
Check: interact_page

**Scenario**: SC-29a — "Switch on" asks no question and the row reads "Active" again
**Given**: FX-STAFF-B (switched off). The browser signs in as the admin and opens `/admin/staff`.
**When**: The run opens the "Tc Staff B" row's menu, presses "Switch on", and reads the page at once and after 2 seconds.
**Then**: No `[role=dialog]` or `[role=alertdialog]` ever appears; a `PUT` with `{"is_active":true}` answers 200; the row reads "Active" and its menu now offers "Switch off".
Traces: SC-29 (ADMIN-031 AC-5, item 3, decision 5)
Check: interact_page

**Scenario**: SC-29b — after "Switch on" the person signs in with the same password but the old cookie stays dead
**Given**: FX-STAFF-A signed in (cookie S1), then switched off by ADMIN, as in SC-27a.
**When**: ADMIN sends `PUT /v1/admin/staff/<id>` with `{"is_active":true}`; then S1 sends `GET /v1/auth/me`; then `POST /v1/auth/sign-in` with the email and `staff-pass-A1`; then the new cookie S3 sends `GET /v1/auth/me`.
**Then**: The `PUT` answers 200 with `is_active: true`; S1 answers 401; sign-in answers 200; S3 answers 200 with `role` `staff`; S1 still answers 401 after S3 exists.
Traces: SC-29 (ADMIN-031 AC-5, item 3)
Check: fetch_url

**Scenario**: SC-30 — no session of a switched-off account ever answers 200
**Given**: FX-STAFF-A active with password A; the session is ADMIN.
**When**: In ten rounds, each starting with the account active again and no earlier sessions, ADMIN's `PUT /v1/admin/staff/<id>` `{"is_active":false}` and the staff member's `POST /v1/auth/sign-in` are sent at the same moment; when both have finished, the cookie the sign-in returned (if it returned one) sends `GET /v1/auth/me` and `GET /v1/orders`.
**Then**: In every round the sign-in answered 401 `invalid_credentials`, or answered 200 and its cookie now answers 401 on both requests; in no round does any request of that account answer 200 after the `PUT` has answered 200; and `SELECT count(*) FROM sessions WHERE user_id = <id>` is 0 after every round.
Traces: SC-30 (ADMIN-031 "Concurrency")
Check: fetch_url

## 11. Remove an account — SC-31 … SC-33

**Scenario**: SC-31 — the removal question can be cancelled with the button or Escape
**Given**: FX-STAFF-A. The browser signs in as the admin and opens `/admin/staff`.
**When**: The run opens the "Tc Staff A" row's menu, presses "Remove", reads the question; presses "Cancel" and reads the page; opens it again, presses Escape, reads the page; ADMIN reads `GET /v1/admin/staff`.
**Then**: The question is titled "Remove Tc Staff A?", carries the note "The account is deleted and cannot be restored. Their work stays." and the buttons "Cancel" and "Remove"; after "Cancel" and after Escape it is gone, the row "Tc Staff A" is still in the table and no `DELETE` request was sent; the read still holds the account.
Traces: SC-31 (ADMIN-031 AC-12, item 4)
Check: interact_page

**Scenario**: SC-32a — confirming "Remove" takes the row out of the table
**Given**: FX-STAFF-A and FX-STAFF-B. The browser signs in as the admin and opens `/admin/staff`.
**When**: The run opens the "Tc Staff A" row's menu, presses "Remove", presses the question's "Remove", and reads the page.
**Then**: A `DELETE /api/v1/admin/staff/<id>` answers 204; the question closes; the "Tc Staff A" row is gone and the "Tc Staff B" row and the admin's row remain.
Traces: SC-32 (ADMIN-031 AC-6, item 4)
Check: interact_page

**Scenario**: SC-32b — after removal the person's cookie is dead and the email can be added again
**Given**: FX-STAFF-A signed in (cookie S1); the session is ADMIN; the old id is read.
**When**: ADMIN sends `DELETE /v1/admin/staff/<id>`; then S1 sends `GET /v1/auth/me`; then ADMIN sends `DELETE` of the same id again; then `POST /v1/admin/staff` with the same name, email `tc-sa@example.com` and a new password; then S1 sends `GET /v1/auth/me` again and the old password signs in.
**Then**: The first `DELETE` answers 204 with an empty body; S1 answers 401; the second `DELETE` answers 404 `{"error":"not_found",…}`; the new `POST` answers 201 with a new `id` different from the old one; S1 answers 401 again (no earlier session is valid for the new account); sign-in with the old password answers 401 `invalid_credentials` and with the new one 200; `SELECT count(*) FROM sessions WHERE user_id = '<old id>'` is 0.
Traces: SC-32 (ADMIN-031 AC-6, item 4)
Check: fetch_url

**Scenario**: SC-33a — removing a staff member changes no order, shoe, line, category or review
**Given**: FX-STAFF-A; FX-SHOP-COUNTS read before; the session is ADMIN.
**When**: ADMIN sends `DELETE /v1/admin/staff/<FX-STAFF-A id>`; FX-SHOP-COUNTS is read again.
**Then**: The `DELETE` answers 204; the five totals (orders, shoes, lines, categories, reviews) are each equal to what they were before.
Traces: SC-33 (ADMIN-031 item 4)
Check: fetch_url

**Scenario**: SC-33b — a picture the removed account uploaded is kept with its uploader cleared
**Given**: The `uploads` table exists (otherwise this case is not applicable and is reported so); FX-STAFF-A and FX-UPLOAD.
**When**: ADMIN sends `DELETE /v1/admin/staff/<FX-STAFF-A id>`; the verifier reads `SELECT name, uploaded_by FROM uploads WHERE name = repeat('a',32) || '.png'` and sends `GET /v1/uploads/<that name>`.
**Then**: The `DELETE` answers 204; the SQL returns one row whose `uploaded_by` is `NULL`; the `GET` still answers 200 (the picture is served). The row is deleted afterwards.
Traces: SC-33 (ADMIN-031 item 4; erd `uploads.uploaded_by` SET NULL)
Check: fetch_url

## 12. Never lock the owner out — SC-34 … SC-38

**Scenario**: SC-34a — every `PUT` on the admin's own id is 409 with the SRS sentence, whatever the body
**Given**: The session is ADMIN; the admin's id is read from `GET /v1/admin/staff`; FX-STAFF-A exists for comparison.
**When**: ADMIN sends `PUT /v1/admin/staff/<admin id>` with these bodies in turn: `{"is_active":false}`; `{"name":"Boss"}`; `{"password":"a-new-owner-pass-1"}`; `{"is_active":true}`; `{"role":"staff"}`; `{"email":"x@example.com"}`; `{"name":""}` (invalid); `{"is_active":"yes"}` (wrong type); `{}`; the text `not json` (with `Content-Type: application/json`); then `GET /v1/auth/me` and `GET /v1/admin/staff`; then sign-in with `ADMIN_PASSWORD`.
**Then**: Every `PUT` answers 409 `{"error":"conflict","message":"The admin account cannot be switched off, changed or removed."}` — the message word for word, no `errors` key, never 400 or 422 (the target is checked before the body is read). Afterwards `GET /v1/auth/me` answers 200 with role `admin` and the admin's earlier name; the list shows the admin with `is_active: true`, the same name and email; sign-in with `ADMIN_PASSWORD` answers 200 and with `a-new-owner-pass-1` answers 401.
Traces: SC-34 (ADMIN-031 AC-7, item 5)
Check: fetch_url

**Scenario**: SC-35a — `DELETE` of the admin's id is 409 and the admin remains
**Given**: The session is ADMIN; the admin's id is known.
**When**: ADMIN sends `DELETE /v1/admin/staff/<admin id>`; then `GET /v1/auth/me`; then `GET /v1/admin/staff`.
**Then**: The `DELETE` answers 409 `{"error":"conflict","message":"The admin account cannot be switched off, changed or removed."}` with no `errors` key; `GET /v1/auth/me` still answers 200 with role `admin`; the list still holds the admin first.
Traces: SC-35 (ADMIN-031 AC-8, item 5)
Check: fetch_url

**Scenario**: SC-35b — after every attempt on the admin exactly one account of role admin exists
**Given**: Every request of SC-34a and SC-35a has been sent.
**When**: The verifier reads `SELECT count(*), bool_and(is_active) FROM users WHERE role = 'admin'` through `set_db_state`, and ADMIN reads `GET /v1/admin/staff?limit=100`.
**Then**: The SQL returns count 1 and `is_active` true; the list holds exactly one item with `role` `admin`, and it is the first.
Traces: SC-35 (ADMIN-031 AC-7, AC-8, item 5)
Check: fetch_url

**Scenario**: SC-36a — `PUT` with `role` is 422 naming `role` and the role does not change
**Given**: FX-STAFF-A; the session is ADMIN.
**When**: ADMIN sends `PUT /v1/admin/staff/<id>` with `{"role":"admin"}`; with `{"role":"staff"}`; with `{"role":"customer"}`; with `{"name":"Tc Changed","role":"admin"}`; then reads `GET /v1/admin/staff` and `GET /v1/auth/me` as the staff member (cookie from before) and sends the staff member's `GET /v1/admin/staff`.
**Then**: All four answer 422 `{"error":"invalid",…,"errors":{"role":<non-empty>}}` (not 400); the list still shows FX-STAFF-A with role `staff` and the name "Tc Staff A" (the combined body changed nothing); the staff member's session still answers 200 on `/v1/auth/me` with role `staff` and 403 on `/v1/admin/staff`; the verifier's `SELECT count(*) FROM users WHERE role = 'admin'` is 1.
Traces: SC-36 (ADMIN-031 AC-9, item 5)
Check: fetch_url

**Scenario**: SC-36b — `POST` with `role` is 400 and creates nothing, so no endpoint can make an admin
**Given**: The session is ADMIN; the team's `total` is read first.
**When**: ADMIN sends `POST /v1/admin/staff` with `{"name":"Tc Boss","email":"tc-boss@example.com","password":"boss-pass-0012","role":"admin"}`, then the same with `"role":"staff"`; then reads the list.
**Then**: Both answer 400 `{"error":"bad_request","message":<non-empty>,"errors":{"role":<non-empty>}}`; the list's `total` is unchanged and `tc-boss@example.com` is absent; the count of `role = 'admin'` rows is 1.
Traces: SC-36 (ADMIN-031 AC-9, item 5); contract (§2.5 an undefined field is 400)
Check: fetch_url

**Scenario**: SC-37a — `PUT` and `DELETE` on a customer's id, an unknown uuid and a non-uuid answer one and the same 404
**Given**: FX-CUSTOMER (id read with `SELECT id FROM users WHERE email = 'tc-customer@example.com'`); the session is ADMIN.
**When**: ADMIN sends `PUT` with `{"is_active":false}` and `DELETE` to `/v1/admin/staff/` followed by each of: FX-CUSTOMER's id; `11111111-2222-4333-8444-555555555555` (an unknown uuid); `not-a-uuid`; `123`. The eight answers are compared.
**Then**: Every one of the eight answers is 404 with the byte-identical body `{"error":"not_found","message":<one and the same sentence>}` and no `errors` key (a customer's id cannot be told from one that does not exist); no 409, 422 or 500.
Traces: SC-37 (ADMIN-031 AC-10, item 1; "Not found" row)
Check: fetch_url

**Scenario**: SC-37b — a customer's account is left exactly as it was, and no body turns it into staff or admin
**Given**: FX-CUSTOMER; the session is ADMIN.
**When**: ADMIN sends `PUT /v1/admin/staff/<FX-CUSTOMER id>` with `{"name":"Hijack"}`, `{"password":"hijacked-pass-1"}`, `{"role":"admin"}` and `{"role":"staff"}`, then `DELETE` of the same id; then the verifier reads the row, and FX-CUSTOMER signs in.
**Then**: Each of the five answers 404 (the target is loaded before the body is read; see open point 2); the row still has name "Tc Customer", role `customer`, `is_active` true and its original hash; sign-in with `customer-pass-1` answers 200 with role `customer`, and with `hijacked-pass-1` answers 401; `GET /v1/admin/staff` still does not list it.
Traces: SC-37 (ADMIN-031 AC-10, item 1)
Check: fetch_url

**Scenario**: SC-38a — `PUT` applies exactly the fields sent
**Given**: FX-STAFF-A signed in (cookie S1); the session is ADMIN.
**When**: In turn, with S1 sent to `GET /v1/auth/me` after each step: (1) `PUT` `{"name":"Tc Renamed"}`; (2) `PUT` `{"is_active":false}` then `{"is_active":true}`; (3) a fresh sign-in S2 with the current password, then `PUT` `{"password":"staff-pass-C3"}`; (4) `PUT` `{"name":"Tc Both","password":"staff-pass-D4"}`; (5) `PUT` `{"name":"Tc Pair","is_active":false}`.
**Then**: (1) answers 200 with name "Tc Renamed" and everything else unchanged, S1 answers 200 afterwards (a rename ends no session) and the password A still signs in; (2) switching off then on answers 200 each time with only `is_active` changed, S1 answers 401 after the switch-off; (3) answers 200, only the password changes (name stays, `is_active` stays true), S2 answers 401, `staff-pass-C3` signs in; (4) answers 200 with name "Tc Both", `staff-pass-D4` signs in and `staff-pass-C3` does not; (5) answers 200 with name "Tc Pair" and `is_active` false. Each answer has exactly the five account keys.
Traces: SC-38 (ADMIN-031 item 1)
Check: fetch_url

**Scenario**: SC-38b — a body with `email` or another undefined field is 400 and nothing changes
**Given**: FX-STAFF-A; the session is ADMIN; the account is read first.
**When**: ADMIN sends `PUT /v1/admin/staff/<id>` with `{"email":"changed@example.com"}`; `{"name":"Tc Mixed","email":"changed@example.com"}`; `{"nickname":"x"}`; `{"id":"00000000-0000-0000-0000-000000000000"}`; `{"is_active":"false"}` (wrong type); `{"name":5}` (wrong type); `{"password":12345678901}` (wrong type); then reads the list.
**Then**: Every answer is 400 `{"error":"bad_request","message":<non-empty>,"errors":{<the offending field>:<non-empty>}}` (never 200, never 422); the account still has name "Tc Staff A", email `tc-sa@example.com`, `is_active` true, and its sessions still answer 200.
Traces: SC-38 (ADMIN-031 item 1; SC-36); contract (§2.5, §3.27 any other undefined field is 400)
Check: fetch_url

**Scenario**: SC-38c — name and password limits on `PUT`
**Given**: FX-STAFF-A; the session is ADMIN.
**When**: ADMIN sends `PUT` with `{"name":""}`, `{"name":"   "}`, `{"name":<101 characters>}`, `{"name":<100 characters>}`, `{"password":<9 characters>}`, `{"password":<10 characters>}`, `{"password":<100 characters>}`, `{"password":<101 characters>}`.
**Then**: The empty, spaces-only and 101-character names answer 422 naming `name` and the name is unchanged; the 100-character name answers 200; the 9- and 101-character passwords answer 422 naming `password` and the earlier password still signs in; the 10- and 100-character passwords answer 200 and sign in.
Traces: SC-38 (ADMIN-031 item 1; boundary as SC-16, SC-17)
Check: fetch_url

## 13. Refusals in the page, and the whole path — SC-39 … SC-41

**Scenario**: SC-39a — a real 404 (the account was removed behind the page's back) shows its sentence in the question and the table redraws
**Given**: FX-STAFF-A and FX-STAFF-B. The browser signs in as the admin and opens `/admin/staff` (both rows shown); then the "Tc Staff A" account is removed without the page knowing (`DELETE /v1/admin/staff/<id>` sent with the ADMIN cookie, or `DELETE FROM users WHERE email = 'tc-sa@example.com'`).
**When**: The run opens the stale "Tc Staff A" row's menu, presses "Switch off", presses the question's "Switch off", and reads the page.
**Then**: The `PUT` answers 404; a non-empty sentence from the shop is shown inside the open question (not the "could not save" text); after the table redraws, the "Tc Staff A" row is gone and the "Tc Staff B" row and the admin's row remain.
Traces: SC-39 (ADMIN-031 item 6; ADMIN-030 item 7)
Check: interact_page

**Scenario**: SC-39b — a 409 or a 422 answer shows the shop's sentence in the open dialog and leaves the row unchanged
**Given**: FX-STAFF-A. The browser signs in as the admin and opens `/admin/staff`; the answer to `PUT /api/v1/admin/staff/<id>` is replaced, in turn, by 409 `{"error":"conflict","message":"The admin account cannot be switched off, changed or removed."}` and by 422 `{"error":"invalid","message":"Check the form.","errors":{"password":"The password needs at least 10 characters."}}`; the list answer stays real.
**When**: The run presses "Switch off" and confirms (409); then opens "Reset password", types a valid password and presses "Save" (422).
**Then**: For the 409 the sentence "The admin account cannot be switched off, changed or removed." is shown inside the open question and the "Tc Staff A" row still reads "Active" after the table redraws from the real list; for the 422 the dialog stays open, the sentence "The password needs at least 10 characters." is beside "New password", and "Saved." is not shown.
Traces: SC-39 (ADMIN-031 items 5, 6)
Check: interact_page

**Scenario**: SC-39c — when the shop cannot be reached a dialog shows "We could not save. Please try again." and the row is unchanged
**Given**: FX-STAFF-A. The browser signs in as the admin and opens `/admin/staff`; every `PUT` and `DELETE` to `/api/v1/admin/staff/<id>` is aborted (no network).
**When**: The run confirms "Switch off" for "Tc Staff A"; then, in a fresh question, confirms "Remove"; then opens "Reset password", types a valid password and presses "Save".
**Then**: Each time the dialog or question stays open and shows "We could not save. Please try again."; the "Tc Staff A" row still reads "Active" and is still in the table; "Saved." is not shown; ADMIN's read of the list shows the account unchanged.
Traces: SC-39 (ADMIN-031 item 6; ADMIN-030 item 7)
Check: interact_page

**Scenario**: SC-40a — a staff account the admin made works the whole admin area except this page
**Given**: ADMIN has created "Lan Tran", `lan@example.com`, `lan-pass-0012`; the shop holds its seeded shoes and orders. A fresh browser signs in as that account.
**When**: In the same session the run opens `/admin` (overview), `/admin/orders` and `/admin/products`, reads each page and the left menu's foot, then opens `/admin/staff`; the same cookie is used for `GET /v1/admin/staff`.
**Then**: The overview shows its figures with data (not zeros and not an error); the Orders list shows at least one order row; the Shoes list shows at least one shoe row; the menu's foot shows the initial `L`, the name "Lan Tran" and the role `staff`; opening `/admin/staff` ends on `/admin`; `GET /api/v1/admin/staff` answers 403.
Traces: SC-40 (ADMIN-030 item 3; ADMIN-002 AC-20; ADMIN-004 AC-10)
Check: interact_page

**Scenario**: SC-40b — the same account's cookie reaches the dealer data endpoints and is refused only the account ones
**Given**: As SC-40a; the cookie is STAFF.
**When**: STAFF sends `GET /v1/orders?limit=1`, `GET /v1/products?limit=1`, `GET /v1/admin/overview`, then `GET /v1/admin/staff`.
**Then**: The first three answer 200 (the orders list with a `total` ≥ 1, the products list with drafts visible in the dealer view, the overview with its figures); the last answers 403 `{"error":"forbidden",…}`.
Traces: SC-40 (ADMIN-030 item 3; ADMIN-004 AC-10, AC-11)
Check: fetch_url

**Scenario**: SC-41a — the page's words are exactly the story's, in the right places
**Given**: FX-STAFF-A and FX-STAFF-B exist. The browser signs in as the admin and opens `/admin/staff`.
**When**: The run reads the page's visible text in each state: default; the add dialog; the reset dialog; the switch-off question; the removal question; the empty-field refusal of the add dialog.
**Then**: The visible strings are word for word those in §2's "Browser words": "Staff", "Add staff member", "Name", "Email", "Role", "Status", "Admin", "Staff", "Active", "Switched off", "Reset password", "Switch off", "Remove", "Starting password", "At least 10 characters.", "Cancel", "Add", "Reset password for Tc Staff A", "New password", "Save", "Switch off Tc Staff A?", "Remove Tc Staff A?", "The account is deleted and cannot be restored. Their work stays."; no raw key such as `pages.admin.staff.title`, no `undefined`, no `{name}` placeholder is shown; the left menu reads "Staff".
Traces: SC-41 (ADMIN-030 item 7; open question 9)
Check: interact_page

**Scenario**: SC-41b — new words are content keys, nothing imports the mock, the client uses `api.ts`, and the checks pass
**Given**: The backend PR of the story is the head being verified.
**When**: A person reads the diff of `code/frontend/src/content.json`, `src/pages/AdminStaff.tsx`, `src/components/StaffAccounts.tsx`, `src/lib/staff.ts`, `src/App.tsx`, `src/pages/admin-nav.ts`, and the PR's CI checks.
**Then**: Every new word is a key under `pages.admin.staff.*` or `pages.admin.nav.staff`, added only (no existing key changed or removed); the page and its components draw them through `useContent` / `<T>`; nothing imports a file under `src/lib/mock/` and none is left; `src/lib/staff.ts` calls the API through `src/lib/api.ts`; the `src/App.tsx` and `admin-nav.ts` edits are the one route line and the one `roles: ["admin"]` item; each new file is under 300 lines; `npm run lint`, `npm run build`, `npm test`, `go build ./...`, `go vet ./...` and `go test ./...` pass.
Traces: SC-41 (ADMIN-030 item 7; open question 9)
Check: manual — these are rules about the contents of source files and the CI result, which no HTTP response or rendered page shows; the build and test commands themselves are run with `run_tests` and the PR's CI is read

## 14. The contract's own rules for these endpoints — C-1 … C-8

**Scenario**: C-1 — every error has the one body shape, and every answer carries a request id
**Given**: FX-STAFF-A and FX-VICTIM; the sessions ADMIN, STAFF and NONE.
**When**: One request of each kind is sent: NONE `GET /v1/admin/staff` (401); STAFF `GET /v1/admin/staff` (403); ADMIN `PUT` on an unknown uuid (404); ADMIN `PUT` on the admin's id (409); ADMIN `POST` with `{}` (422); ADMIN `POST` with `{"x":1}` (400). A seventh, `GET /v1/admin/staff`, carries the header `X-Request-Id: tc-request-0001` and an eighth carries none.
**Then**: Each non-2xx answer has `Content-Type: application/json; charset=utf-8` and exactly the keys `error` and `message`, plus `errors` on the 422 and the 400 only; the `error` codes are `unauthenticated`, `forbidden`, `not_found`, `conflict`, `invalid`, `bad_request`; no body contains SQL, a path, a stack trace or a hash. The answer to the seventh echoes `X-Request-Id: tc-request-0001`; the eighth carries a generated `X-Request-Id` of 16 hexadecimal characters.
Traces: contract (§2.1, §2.3; §3.26, §3.27)
Check: fetch_url

**Scenario**: C-2 — a body that is not JSON, or JSON of the wrong shape, is 400
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with the body `not json`; with `[]`; with `"text"`; with `{"name":5,"email":"a@b.co","password":"long-enough-pass"}`; with `{"name":"Ok","email":["a@b.co"],"password":"long-enough-pass"}`; with `{"name":"Ok","email":"a@b.co","password":12345678901}`; and an empty body.
**Then**: Every answer is 400 `{"error":"bad_request","message":<non-empty>}` (with `errors` naming the field for the wrong-typed ones); none is 422, 500 or 201; the team's `total` is unchanged.
Traces: contract (§3.26 `bad_request`; §2.5)
Check: fetch_url

**Scenario**: C-3 — a body field the contract does not define is 400 on `POST`
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with `{"name":"Tc Extra","email":"tc-extra@example.com","password":"extra-pass-0012","is_active":false}` and with the same plus `"phone":"0912345678"`.
**Then**: Both answer 400 `{"error":"bad_request","message":<non-empty>,"errors":{<the extra field>:<non-empty>}}` (never 201 with the field ignored); `tc-extra@example.com` is absent from the list.
Traces: contract (§2.5; §3.26 an undefined field is 400)
Check: fetch_url

**Scenario**: C-4 — a JSON body is capped and the content type must be JSON
**Given**: The session is ADMIN.
**When**: ADMIN sends `POST /v1/admin/staff` with a valid body padded with a 70 KiB `name` string (valid JSON, 71,680 bytes); `POST` with the valid body `{"name":"Tc Plain","email":"tc-plain@example.com","password":"plain-pass-0012"}` and `Content-Type: text/plain`; `PUT /v1/admin/staff/<FX-STAFF-A id>` with `{"name":"Tc Plain"}` and `Content-Type: text/plain`.
**Then**: The first answers 413 `{"error":"payload_too_large","message":<non-empty>}`; the second and third answer 415 `{"error":"unsupported_media_type","message":<non-empty>}`; nothing is created and FX-STAFF-A's name is unchanged.
Traces: contract (§2.1, §2.3, §2.5; §3.26, §3.27)
Check: fetch_url

**Scenario**: C-5 — the list's paging parameters follow the project's rules
**Given**: The session is ADMIN; FX-STAFF-A exists.
**When**: ADMIN sends `GET /v1/admin/staff?page=0`; `?page=-1`; `?page=abc`; `?limit=0`; `?limit=abc`; `?limit=101`; `?limit=1000`; `?zzz=1`.
**Then**: The first three answer 400 `{"error":"bad_request",…,"errors":{"page":<non-empty>}}`; the fourth and fifth 400 with `errors.limit`; `?limit=101` and `?limit=1000` answer 200 with `limit` 100 in the body; `?zzz=1` answers 200 with the same items as no parameter.
Traces: contract (§2.4 paging; §3.26)
Check: fetch_url

**Scenario**: C-6 — `DELETE` and `PUT` target a uuid path segment only
**Given**: The session is ADMIN; FX-STAFF-A exists.
**When**: ADMIN sends `DELETE /v1/admin/staff/` followed by the upper-cased id of FX-STAFF-A; and `PUT /v1/admin/staff/<id>/extra` with `{"name":"X"}`; and `GET /v1/admin/staff/<id>`; and `PATCH /v1/admin/staff/<id>`.
**Then**: The account is unchanged afterwards (name "Tc Staff A", active, present); the `GET` and `PATCH` on the item path answer 404 or 405 with a body of the one error shape, never 200 and never a modification; the upper-cased id answers either 404 (a segment that does not match the lower-case id pattern) or 204 with the account removed — whichever, the answer is 404 or 204 and nothing else.
Traces: contract (§2.5 path segments; §3.27 `{id}` is a uuid)
Check: fetch_url

**Scenario**: C-7 — a failed write is 500 `internal` with a generic body and nothing half-saved
**Given**: FX-STAFF-A with a live session S1; the session is ADMIN (signed in before the arrangement). The verifier installs, through `set_db_state`, `CREATE FUNCTION tc_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'tc fail'; END $$; CREATE TRIGGER tc_fail_users BEFORE INSERT OR UPDATE OR DELETE ON users FOR EACH ROW WHEN (NEW.email IS DISTINCT FROM 'tc-never' OR TRUE) EXECUTE FUNCTION tc_fail();` (for a `DELETE` the `WHEN` clause is dropped from the trigger so it fires).
**When**: ADMIN sends `POST /v1/admin/staff` with a valid new account; `PUT /v1/admin/staff/<FX-STAFF-A id>` with `{"is_active":false}`; `DELETE /v1/admin/staff/<FX-STAFF-A id>`; then the trigger and function are dropped; then ADMIN reads the list and S1 sends `GET /v1/auth/me`.
**Then**: All three answers are 500 `{"error":"internal","message":<generic sentence>}` with no `errors` key, and no body contains `tc fail`, `trigger`, `users`, `SQL` or a stack trace; after the trigger is dropped the list does not hold the new account, FX-STAFF-A is still `is_active: true`, and S1 answers 200 (the switch-off's session delete was rolled back with the failed update).
Traces: contract (§2.3 `internal`; §3.26, §3.27 `internal`; §3.27 same transaction)
Check: fetch_url

**Scenario**: C-8 — the guard is on every route, for every method
**Given**: FX-STAFF-A; the sessions NONE, STAFF and CUSTOMER.
**When**: Each of `GET`, `POST` on `/v1/admin/staff` and `PUT`, `DELETE` on `/v1/admin/staff/<id>` is sent with NONE, then STAFF, then CUSTOMER, with a valid body where one is needed (24 requests), and the verifier then reads the team through ADMIN.
**Then**: The NONE answers are 401 `unauthenticated`; the STAFF and CUSTOMER answers are 403 `forbidden`; no answer is 200, 201, 204, 404 or 409; the team is exactly as before.
Traces: contract (§2.2 route table `/v1/admin/staff…`: 401 / 403 / 403 / yes)
Check: fetch_url

## 15. Coverage walk — every acceptance criterion has a case

| Criterion | Cases | | Criterion | Cases |
|---|---|---|---|---|
| SC-1 | SC-1a, SC-1b | | SC-22 | SC-22a, SC-22b |
| SC-2 | SC-2a, SC-2b, SC-2c | | SC-23 | SC-23a, SC-23b |
| SC-3 | SC-3a, SC-3b, SC-3c | | SC-24 | SC-24 |
| SC-4 | SC-4 | | SC-25 | SC-25 |
| SC-5 | SC-5a, SC-5b | | SC-26 | SC-26a, SC-26b |
| SC-6 | SC-6a, SC-6b | | SC-27 | SC-27a, SC-27b, SC-27c |
| SC-7 | SC-7a, SC-7b | | SC-28 | SC-28a, SC-28b |
| SC-8 | SC-8a, SC-8b | | SC-29 | SC-29a, SC-29b |
| SC-9 | SC-9a, SC-9b, C-8 | | SC-30 | SC-30 |
| SC-10 | SC-10, C-8 | | SC-31 | SC-31 |
| SC-11 | SC-11a, SC-11b, SC-11c | | SC-32 | SC-32a, SC-32b |
| SC-12 | SC-12a, SC-12b | | SC-33 | SC-33a, SC-33b |
| SC-13 | SC-13a, SC-13b, SC-13c | | SC-34 | SC-34a |
| SC-14 | SC-14a, SC-14b | | SC-35 | SC-35a, SC-35b |
| SC-15 | SC-15a, SC-15b, SC-15c | | SC-36 | SC-36a, SC-36b |
| SC-16 | SC-16a, SC-16b, SC-16c | | SC-37 | SC-37a, SC-37b |
| SC-17 | SC-17a, SC-17b | | SC-38 | SC-38a, SC-38b, SC-38c |
| SC-18 | SC-18a, SC-18b, SC-18c | | SC-39 | SC-39a, SC-39b, SC-39c |
| SC-19 | SC-19 | | SC-40 | SC-40a, SC-40b |
| SC-20 | SC-20 | | SC-41 | SC-41a, SC-41b |
| SC-21 | SC-21a, SC-21b | | Contract | C-1 … C-8 |

All 41 criteria are covered. The two `manual` cases (SC-21b, SC-41b) are the only ones no instrument can observe; every case that is somebody typing, clicking or pressing is `interact_page`.
