# Story — Staff accounts managed by the owner

Module: `admin` · Plan item 14 · Requirements: ADMIN-030, ADMIN-031, and the account-endpoint half of ADMIN-004 (`docs/admin/SRS.md` §4.4, §4.1)
Design system: `design/design-system.md` (§1.1 management area) · Architecture: `docs/architecture/overview.md`, `erd.md` (`users`, `sessions`), `services.md` (§3.26, §3.27)

## 1. User story

As the **admin** (the owner, signed in with the account made from `ADMIN_EMAIL` / `ADMIN_PASSWORD`), I want a Staff page where I add a staff member, reset their password, switch their account off and on, and remove it, so that my team can sign in and run the shop without sharing my password, and a person who leaves or forgets never keeps or loses access wrongly.

Other actors this story touches:

- **Staff** — signs in with the account the admin made, does everything in `/admin` except this page, is refused by this page's endpoints, and loses the session the moment the admin switches the account off, resets its password or removes it.
- **Customer** and **Visitor** — are refused: a customer sees "This area is for the shop's team" and gets 403; a visitor with no session goes to `/admin/sign-in` and gets 401.

## 2. In scope

**Backend (Go, PostgreSQL) — admin session only**

- `GET /v1/admin/staff` — the team's accounts (role `admin` or `staff`; the admin first, then by creation): `id`, `name`, `email`, `role`, `is_active`. Never a password or a hash. Wire name is `is_active` (`services.md` §3.26); the SRS calls it "active".
- `POST /v1/admin/staff` — `name` (1–100), `email` (valid, ≤ 200, stored lower-case, not belonging to any account of any role), `password` (10–100). Creates role `staff`, active, answers 201 with the account; 422 `{"errors":{…}}` naming every field at fault, a taken email included. The new account can sign in at once.
- `PUT /v1/admin/staff/{id}` — any of `name`, `is_active`, `password` (a field left out is unchanged). Switching off and a password reset end the account's sessions in the same transaction; switching on does not bring them back. `role` is refused with 422 naming `role`; any other undefined field (`email` included) is 400.
- `DELETE /v1/admin/staff/{id}` — 204; the account and its sessions go; the email can be added again.
- The admin account (any row of role `admin`) answers 409 "The admin account cannot be switched off, changed or removed." to `PUT` and `DELETE`, checked before the body is read. A customer's id, an unknown id and a non-uuid id are 404 with one body.
- Every route is `auth.Require("admin")`: 401 with no session, 403 for staff and customers (the guard exists from "Dealer sign-in and protected admin area"). Passwords are stored only as argon2id hashes and never returned, shown or logged. A switched-off account is refused at sign-in with the same status and body as a wrong password, and every request of a signed-in person re-reads `users.is_active` (so no session of an account that is off stays valid).
- No new table is expected: `users` and `sessions` exist from the dealer sign-in story. If TL's technical design needs a migration it is TL's call.

**Frontend (Vite + React + Tailwind tokens)**

- New page `/admin/staff` (`src/pages/AdminStaff.tsx`, its own route line in `src/App.tsx`), built from the admin's existing parts — `DashboardLayout`, `DataTable`, `FormDialog`, `Confirm`, `RowMenu`, `Badge`, `Field`, `Input`, `Notice`, `Button`. It does not use `ListPage`: that is driven by the collections of `schema.json`, and staff are not one.
- One link "Staff" in the left menu (`src/pages/admin-nav.ts`, one added item with `roles: ["admin"]`, in a last group with no heading), shown to the admin only.
- The add dialog, the reset-password dialog, the switch-off and removal questions, the loading, load-failed, field-error, refusal and could-not-save states (section 5).
- The page redirects a staff session to `/admin` without drawing or requesting anything; a customer and a visitor are handled by the `/admin/*` gate that "Dealer sign-in and protected admin area" builds.
- Files (guide for TL's UI review; each under 300 lines, split if not): `src/pages/AdminStaff.tsx`, `src/components/StaffAccounts.tsx` (and one more component file if it grows), `src/lib/staff.ts` (the API client on `src/lib/api.ts`), `src/lib/mock/staff.ts` (UI stage only; deleted by the backend stage).
- New words in `src/content.json`, additive only (section 5).

## 3. Out of scope — and where each piece is proven

| Not here | Why | Proven in |
|---|---|---|
| Changing one's own password, "forgot password", a first-sign-in password change | No screen in the design and no plan item (ADMIN SRS §3 out of scope); the admin resets a staff password | — (owner's decision, see ⚠️ in the reply) |
| An email or text telling the new staff member their password or that they were switched off | The shop sends no email or SMS anywhere; the admin tells them in person (SRS open question 8) | — |
| Changing an account's email or role, a second admin, promoting or demoting | No endpoint changes a role (ADMIN-031 item 5); `PUT` with `email` is refused (SC-38) | SC-36, SC-38 |
| A "Rename" action in the page's row menu | The menu is Reset password, Switch off/on, Remove (ADMIN-030 item 1); renaming is the API's `PUT name` only | SC-38 (API) |
| Listing, editing, switching off or deleting **customers** | Customers are the `accounts` module; no endpoint here touches one (a customer's id is 404) | "Customer accounts: sign up, sign in, my orders" |
| The sign-in page, `/v1/auth/me`, the session cookie, the failure counters, the `/admin/*` gate and the "team only" message | Built by "Dealer sign-in and protected admin area"; this story only proves them against accounts it creates | That item (ADMIN-001…003) |
| What a staff member can do to shoes, lines, categories, reviews and orders (writes, uploads, the per-size counts) | "Manage shoes, lines, categories, reviews and orders"; this story proves a staff session reaches those lists and the overview, not their writes | That item (ADMIN-004 AC-10, ADMIN-010…016) |
| Search, filter, sort or page numbers on the staff table | A team of a few; the page asks `limit=100` and draws everything it gets | — |
| A log of who added, switched off or removed whom, last sign-in time, a list of someone's open sessions, "sign out everywhere" | Not in the SRS | — (see ⚠️) |
| Unlocking an account locked by five failed sign-ins | The lock lasts 15 minutes by itself (ADMIN-003) | Dealer sign-in item |
| The customer site's header "Account" entry | `accounts` module (ACC-007) | Customer accounts item |

## 4. Decisions made here (the SRS leaves them open)

1. **Menu placement:** "Staff" is the only item of a last group with no heading, so a staff session's menu has no empty group. `DashboardLayout` already hides an item whose `roles` do not include the session's.
2. **Where the staff redirect lives:** the page component (`role === "staff"` → `<Navigate to="/admin" replace />`) before any request is made. A staff member never sees the list even for a frame, and the browser makes no `GET /v1/admin/staff` from that page. The gate's own 401/403 handling is item 9's.
3. **Saved notice:** after a reset the dialog closes and the page shows `Notice` "Saved." above the table. After add, switch off/on and remove the table redraws from the shop's answer (no notice).
4. **Starting password field:** hidden characters (`type="password"`, `autocomplete="new-password"`) with the hint "At least 10 characters." The admin chooses it and tells the staff member in person.
5. **Switch on asks no question** (only Switch off and Remove do, ADMIN-031 items 3, 4).
6. **Field sentences from the API** (English, default wording; tests assert the field name and that a sentence shows beside that field, so TL may reword): `name` "A name is needed" / "The name can have at most 100 characters."; `email` "An email address is needed" / "That does not look like an email address." / "That email already belongs to an account."; `password` "The password needs at least 10 characters." / "The password can have at most 100 characters."; `role` "The role cannot be changed.".
7. **Wire names:** the API says `is_active`; the SRS and this story say "active" in prose only.
8. **Test fixtures:** customer accounts cannot be created by the sign-up screen until the customer-accounts story lands, so the tests for "a customer is refused" insert a customer row straight into `users` (the sign-in endpoint serves any row, ADMIN-003 AC-12).

## 5. UI scope

The approved design does not draw this page (SRS §5: "a new page defined by the plan item"). It is built from the admin's own list, dialogs and notices in the management palette (`[data-dashboard]`: slate neutrals, blue accent, 6px radius, Inter), tokens only — no raw colour, no arbitrary-value class, no CSS modules; copy through `useContent` / `<T>` because `eslint` refuses bare words in pages.

| Screen / state | What it shows | SRS |
|---|---|---|
| `/admin/staff`, default (1280 px and 390 px) | Left menu with "Staff" lit; title "Staff"; "Add staff member" button; table with columns Name, Email, Role, Status. The admin's row first: role "Admin", status "Active" (`Badge` good), no menu. Each staff row: role "Staff", status "Active" or "Switched off" (`Badge` off), a row menu | ADMIN-030 items 1 |
| Row menu | "Reset password", "Switch off" (Active rows) or "Switch on" (Switched-off rows), then "Remove" (danger, last) | ADMIN-030 item 1, ADMIN-031 |
| Loading | "Loading the staff…", no empty table | ADMIN-030 item 7 |
| Load failed | "We could not load the staff." and a "Try again" button | ADMIN-030 item 7 |
| Add dialog, default | Title "Add staff member"; labelled fields "Name", "Email", "Starting password" (hint); "Cancel" and "Add" | ADMIN-030 item 2 |
| Add dialog, field errors | The sentence beside each faulty field, field marked invalid, typed values kept | ADMIN-030 item 2 |
| Add dialog, saving / could not save | "Add" reads "Saving…" and is disabled; on an unreachable shop the dialog stays open, values kept, "We could not save. Please try again." | ADMIN-030 item 7 |
| Reset-password dialog | Title "Reset password for <name>"; labelled "New password"; "Cancel" and "Save"; field error and saving states as above; on success it closes and "Saved." shows | ADMIN-031 item 2 |
| Switch-off question | `Confirm` "Switch off <name>?" with a one-line note and "Cancel" / "Switch off" | ADMIN-031 item 3 |
| Removal question | `Confirm` "Remove <name>?" with a one-line note ("The account is deleted and cannot be restored. Their work stays.") and "Cancel" / "Remove" | ADMIN-031 item 4 |
| Refusal | The shop's sentence (for example "The admin account cannot be switched off, changed or removed.") shown in the open dialog; the row unchanged | ADMIN-031 items 5, 6 |
| Staff session on the address | Goes to `/admin` at once; nothing of the page drawn | ADMIN-030 item 6 |
| Customer / visitor on the address | The gate's "This area is for the shop's team" with "Back to the shop" to `/`; or `/admin/sign-in` | ADMIN-030 item 6 |

Phone first: one column from 320 px, no horizontal page scroll at 390 px (the table may scroll inside its own container), the menu is the sheet below 1024 px, dialogs fit the screen height with the buttons reachable. Every field has a visible label; field errors and the load / save messages are announced to assistive technology.

**New words** (`src/content.json` is frozen for stories by the architecture overview; this story is a second place, after "Free delivery on every order", where it is lifted — for **new keys only**, existing keys untouched): `pages.admin.nav.staff`; `pages.admin.staff.title`, `.add`, `.columns.{name,email,role,status}`, `.roles.{admin,staff}`, `.status.{active,off}`, `.menu.{reset,off,on,remove}`, `.dialog.add.{title,name,email,password,hint,cancel,submit}`, `.dialog.reset.{title,password,cancel,submit,saved}`, `.confirm.off.{title,note,verb}`, `.confirm.remove.{title,note,verb}`, `.loading`, `.error`, `.retry`, `.saving`, `.couldNotSave` — wording exactly as in the table above and SRS open question 9. `{name}` is filled in at draw time. TL confirms the exception in the UI review; if TL prefers to add the keys himself the UI PR lists them instead.

## 6. Acceptance criteria

Forty-one criteria; the SRS gives this item 26 of its own (14 in ADMIN-030, 12 in ADMIN-031), and each is traced. Every one is checked against the running story — the page in a browser, the endpoints with real sessions: an admin, a staff account the admin made, a customer row inserted for the test, and no session. "The wrong-password answer" means the status and body of a right email with a wrong password (ADMIN-003 AC-8).

### 6.1 The page and its menu link

- **SC-1 [ADMIN-030 AC-1, item 1]:** The admin opens `/admin/staff` at 1280 px. The page shows the title "Staff", an "Add staff member" button, and a table with the columns Name, Email, Role, Status. The first row is the admin's own (role "Admin", status "Active", no menu); then one row per staff member (role "Staff", status "Active" or "Switched off"). With no staff member yet the table holds only the admin's row and the button is still shown.
- **SC-2 [ADMIN-030 AC-12, item 1, 6]:** An admin session's left menu — the sidebar from 1024 px and the sheet below it — holds a "Staff" link that opens `/admin/staff` and is lit while there. A staff session's menu, on the same pages, holds no "Staff" link at all (no empty group either).
- **SC-3 [ADMIN-030 item 1; ADMIN-031 item 1]:** Each staff row has a menu offering "Reset password", then "Switch off" when the row is Active or "Switch on" when it is Switched off, then "Remove" (marked as dangerous, last). The admin's row has no menu button.
- **SC-4 [ADMIN-030 AC-13, item 7]:** With the staff answer delayed 3 seconds, the page shows "Loading the staff…" and no table and no empty-state text until the answer arrives.
- **SC-5 [ADMIN-030 item 7]:** When the list cannot be fetched (server error or no network) the page shows "We could not load the staff." and a "Try again" button, and no table. Pressing it asks again and draws the table when the shop answers.
- **SC-6 [ADMIN-030 item 1; Responsive NFR]:** At 320 px and 390 px the page has no horizontal page scroll; the menu is the sheet; the table may scroll inside its own container; the row menu, the add dialog and its buttons are reachable on the screen.

### 6.2 Who may reach it

- **SC-7 [ADMIN-030 AC-7, item 6]:** A staff session opens `/admin/staff` by typing the address or by reload: the address becomes `/admin`, the overview shows, the title "Staff" and any row of the list are never drawn, and the browser sends no `GET /v1/admin/staff`.
- **SC-8 [ADMIN-030 AC-9, item 6; ADMIN-002 item 9]:** A signed-in customer who opens `/admin/staff` sees "This area is for the shop's team" with a "Back to the shop" link to `/` and no list, no button, no figure. A visitor with no session who opens it goes to `/admin/sign-in` and sees nothing of the area.
- **SC-9 [ADMIN-030 AC-8, AC-9; ADMIN-031 AC-11; ADMIN-004 AC-11]:** With a staff session, and again with a customer session, `GET /v1/admin/staff`, `POST /v1/admin/staff`, `PUT /v1/admin/staff/{id}` and `DELETE /v1/admin/staff/{id}` each answer 403 with the generic error body; nothing is created, changed or removed.
- **SC-10 [ADMIN-030 AC-10; ADMIN-004 AC-12; ADMIN-031 "Not permitted"]:** With no session the same four calls each answer 401.

### 6.3 Reading the team

- **SC-11 [ADMIN-030 AC-11, items 4, 5]:** As the admin, `GET /v1/admin/staff` answers 200 `{"items","total","page","limit"}` (`limit=100` accepted); each item holds exactly `id`, `name`, `email`, `role`, `is_active`; the admin's row comes first; accounts of role `customer` are never listed. No response of the four endpoints, on any status, contains a password, a hash or a field named like one.

### 6.4 Adding a staff member

- **SC-12 [ADMIN-030 item 2]:** "Add staff member" opens a dialog with three labelled fields — "Name", "Email", "Starting password" (characters hidden, hint "At least 10 characters.") — and "Cancel" and "Add". "Cancel" and Escape close it, send nothing and keep the table as it was; opening it again shows empty fields.
- **SC-13 [ADMIN-030 AC-2, item 4]:** Adding "Lan Tran", `lan@example.com` and a 12-character password sends `POST /v1/admin/staff`, answers 201 with `role: "staff"` and `is_active: true`, closes the dialog, and the table shows Lan Tran, `lan@example.com`, Staff, Active without a reload. An email typed `Lan@Example.com` is stored and shown as `lan@example.com`.
- **SC-14 [ADMIN-030 AC-3, item 3; ADMIN-002 AC-16]:** The staff member just added signs in at `/admin/sign-in` with that email and the starting password: the address becomes `/admin` and `GET /v1/auth/me` answers `role: "staff"` and the name "Lan Tran".
- **SC-15 [ADMIN-030 AC-4, item 2]:** An email is refused with 422 naming `email` — the dialog stays open and the sentence shows beside Email — when it belongs to an existing staff, admin or customer account (`Lan@Example.com` against `lan@example.com` included), when it is empty or does not look like an email, and when it is over 200 characters. Nothing is added.
- **SC-16 [ADMIN-030 AC-5; boundary]:** A password of 9 characters is refused with 422 naming `password`; 10 characters is accepted (201); 100 is accepted; 101 is refused with 422. Nothing is added by a refusal.
- **SC-17 [ADMIN-030 AC-6, item 2]:** "Add" with the name empty or only spaces is refused with 422 naming `name`, as is a name of 101 characters; the dialog stays open and the other fields keep what was typed.
- **SC-18 [ADMIN-030 item 2; Accessibility NFR]:** When several fields are wrong at once, one 422 answer names every one and each sentence shows beside its own field, which is marked invalid; the typed values (the password too) are kept; the sentences are announced to assistive technology. While the request is in flight "Add" reads "Saving…" and is disabled, so a double press creates one account.
- **SC-19 [ADMIN-030 AC-14, item 7]:** With the shop unreachable, pressing "Add" leaves the dialog open with the typed values and shows "We could not save. Please try again."; the table is unchanged. Pressing "Add" again once the shop is back creates the account once.
- **SC-20 [ADMIN-030 "Conflict"]:** Two `POST`s of one email sent at the same moment answer one 201 and one 422 naming `email`; the table and the `users` table hold one such account.
- **SC-21 [ADMIN-030 item 5; ADMIN-003 AC-2]:** The stored password of a new account is an argon2id hash: it differs from the typed one, differs between two accounts given the same password, and neither the typed password nor the hash appears in any response or in the server's log lines for the request.

### 6.5 Reset a password

- **SC-22 [ADMIN-031 AC-1, item 2]:** "Reset password" on a staff row opens a dialog "Reset password for <name>" with a labelled "New password" field and "Cancel" / "Save". Saving a 12-character password B closes it and shows "Saved."; sign-in with the old password A is refused with "Wrong email or password." and with B succeeds.
- **SC-23 [ADMIN-031 AC-2]:** A new password of 9 characters (or empty, or over 100) is refused with 422 naming `password`: the dialog stays open, the sentence shows beside the field, "Saved." does not show, and A still signs in.
- **SC-24 [ADMIN-031 "Data touched", item 1]:** A staff member signed in before the reset sends their next request after it (`GET /v1/auth/me`, `GET /v1/orders`) and gets 401; they sign in again with B.

### 6.6 Switch an account off and on

- **SC-25 [ADMIN-031 AC-12, item 3]:** "Switch off" asks "Switch off <name>?" with "Cancel" and "Switch off". "Cancel" (or Escape) closes it, sends no request and the row stays "Active".
- **SC-26 [ADMIN-031 AC-3, item 3]:** Confirming "Switch off" sends `PUT {"is_active": false}`; the row reads "Switched off" (redrawn from the shop's answer) and its menu now offers "Switch on".
- **SC-27 [ADMIN-031 AC-3, item 3]:** A staff member signed in in another browser sends their next request after the switch-off (any endpoint, `GET /v1/auth/me` included) and gets 401, and their open admin page, on its next request or navigation, ends on `/admin/sign-in`. A reload does the same.
- **SC-28 [ADMIN-031 AC-4; ADMIN-003 AC-13]:** Signing in to the switched-off account with its right password answers the same status and body as a wrong password, and the sign-in page shows "Wrong email or password.".
- **SC-29 [ADMIN-031 AC-5, item 3]:** "Switch on" asks no question; the row reads "Active" again; the staff member signs in again with the same password; the cookie from before the switch-off still answers 401.
- **SC-30 [ADMIN-031 "Concurrency"]:** When the admin switches an account off while its owner signs in, the sign-in either fails or the session it created answers 401 once both have finished; no session of an account that is switched off ever answers 200.

### 6.7 Remove an account

- **SC-31 [ADMIN-031 AC-12, item 4]:** "Remove" asks "Remove <name>?" with "Cancel" and "Remove". "Cancel" (or Escape) closes it, sends no request and the row stays.
- **SC-32 [ADMIN-031 AC-6, item 4]:** Confirming "Remove" sends `DELETE` and gets 204; the row is gone; the staff member's next request answers 401; the same email can then be added again (201, a new id, no earlier session valid).
- **SC-33 [ADMIN-031 item 4]:** Removing a staff member changes no order, shoe, line, category or review: the counts of those collections before and after are equal, and a picture row the removed account uploaded, if the uploads table exists by then, is kept with its uploader link cleared.

### 6.8 Never lock the owner out

- **SC-34 [ADMIN-031 AC-7, item 5]:** As the admin, `PUT /v1/admin/staff/{admin's own id}` with `{"is_active": false}` answers 409 "The admin account cannot be switched off, changed or removed."; so does a `PUT` with a new name, a new password or an invalid body (the target is checked before the body). The admin stays signed in and `is_active` stays true.
- **SC-35 [ADMIN-031 AC-8, item 5]:** `DELETE /v1/admin/staff/{admin's own id}` answers 409 with the same sentence; the account remains and `GET /v1/auth/me` still answers 200. After every call of SC-34 and SC-35 exactly one account of role `admin` exists.
- **SC-36 [ADMIN-031 AC-9, item 5]:** `PUT` with `role` (`"admin"` or `"staff"`) on a staff account answers 422 naming `role` and the role is unchanged. `POST` with a `role` field answers 400 and creates nothing, so no endpoint can make or change an admin.
- **SC-37 [ADMIN-031 AC-10, item 1]:** `PUT` and `DELETE` with a customer's id, an unknown uuid and a string that is not a uuid each answer 404 with one and the same body; the customer's account is unchanged and can still sign in.
- **SC-38 [ADMIN-031 item 1]:** `PUT` applies exactly the fields sent: a name alone renames the account and ends no session; `is_active` alone; `password` alone; two together; a body with `email` answers 400 and nothing changes. Name length 1–100 and password length 10–100 are enforced as in SC-16 and SC-17.

### 6.9 Refusals in the page, and the whole path

- **SC-39 [ADMIN-031 item 6; ADMIN-030 item 7]:** When the shop refuses an action from the page — 409, 404 or 422 — its sentence shows in the open dialog or question and the table redraws from the shop's answer; when the shop cannot be reached the dialog shows "We could not save. Please try again." and the row is unchanged. (Reachable from the browser only by stubbing the answer; the API side of each refusal is SC-34 to SC-37.)
- **SC-40 [ADMIN-030 item 3; ADMIN-002 AC-20; ADMIN-004 AC-10]:** A staff account the admin created signs in and, in the same session, loads the Orders list, the Shoes list and the overview with data, and its menu foot shows its name and the role "staff". It gets 403 from `/v1/admin/staff` and is sent to `/admin` from `/admin/staff`. (What a staff member may write to shoes and orders is proven in "Manage shoes, lines, categories, reviews and orders".)
- **SC-41 [ADMIN-030 item 7; open question 9]:** Every new word of the page and its dialogs is a key under `pages.admin.staff.*` or `pages.admin.nav.staff` in `src/content.json` (additive only; no existing key changes) and is drawn through `useContent` / `<T>`. After the backend PR no file under `src/lib/mock/` is imported by the page, `src/lib/staff.ts` calls the API through `src/lib/api.ts`, and `npm run lint`, `npm run build` and `npm test`, and `go build ./...`, `go vet ./...` and `go test ./...` all pass.

## 7. Dependencies

- **Must land first:** "Dealer sign-in and protected admin area" — the `users` and `sessions` tables, `auth.Require`, `GET /v1/auth/me`, the admin made from `ADMIN_EMAIL` / `ADMIN_PASSWORD`, sign-in that refuses an inactive account with the wrong-password answer, the real `useSession` / `useRole`, and the `/admin/*` gate with its team-only message. "Site data from the database" before it supplies `src/lib/api.ts` (`ApiError`, `credentials: "include"`).
- **Assumed from the gate (flag to TL):** when any admin request answers 401 the browser ends on `/admin/sign-in` (SC-27, ADMIN-031 item 3). If the gate as built only checks `/v1/auth/me` at load, a reload is the guaranteed path; fixing the gate is that item's work, not this one's, and SC-27 is the check that finds it.
- **Needs from the environment:** `ADMIN_EMAIL` and `ADMIN_PASSWORD` (≥ 10 characters) set in the console for the verification run; a customer row inserted by the test (section 4, item 8). No other account, key or provider; the shop sends no email.
- **Independent of:** "Manage shoes, lines, categories, reviews and orders" and "Dealer dashboard figures and charts" — SC-40 reads whichever of those pages are on `main`; their writes are proven in their own stories. `src/pages/admin-nav.ts` is edited by this story only by adding one item.
- **Unblocks:** nothing in the plan waits on it.
