# SRS — Admin: dealer sign-in, records and dashboard

Module: `admin`
Design: [View the approved design](http://localhost:9401)
Design system: `design/design-system.md`

> One file per module. This one covers the dealer's side of the shop: the door into it, keeping the
> catalogue and the orders, and the overview. Placing orders and customer-facing order pages are in
> `docs/orders/SRS.md`; the list query language is in `docs/catalogue/SRS.md`.

## 1. Purpose

The owner and their staff run the shop from `/admin`: they sign in, change a shoe's price or hide a review, move an
order from placed to delivered, and see on one screen how the month is going and what is running out. Today every
`/admin` page is open to anyone, and Save writes into the browser's memory and is lost on reload. Without this
module the owner cannot manage the content and the records the brief asks for, and a stranger can read customers'
phones and addresses.

## 2. Actors

| Actor | Who they are | What they may do in this module |
|---|---|---|
| Visitor | No session | Sign in. Nothing else: every dealer endpoint answers 401 |
| Admin | The owner, signed in with the account made from the owner's settings (role "admin") | Everything in this module, and the only one who manages accounts (ADMIN-030, ADMIN-031) |
| Staff | A team member the admin added (role "staff") | Everything in this module except account management: shoes, lines, categories, reviews, orders, uploads and the overview |
| Customer | A signed-in customer (role "customer", `accounts` module) | Nothing in this module: every dealer endpoint answers 403 and `/admin` shows "This area is for the shop's team" |

In this file "the dealer" means a session whose role is admin or staff.

## 3. Scope

**In scope** — the plan items, by title:

- Dealer sign-in and protected admin area
- Manage shoes, lines, categories, reviews and orders
- Dealer dashboard figures and charts
- Staff accounts managed by the owner

**Out of scope**

- Changing one's own password and "forgot password" — no screen in the design and no plan item (the admin resets a staff member's password, ADMIN-031). The sign-in page's note "Accounts are for the dealer and staff. Ask the owner for one." is the owner's wording and stays.
- Customer sign-up, the customer sign-in screens and "My account" — the `accounts` module. The sign-in, sign-out and `/v1/auth/me` endpoints in this module serve all three roles.
- How customers place and read orders — the `orders` module.
- Marking an order paid — only the payment provider's notice does that (`payments` module); nobody marks an order paid by hand here, and there is no cash on delivery.
- Editing the pairs, sizes or prices of a placed order — not built (open question 4).
- Refunds — made by the dealer in the gateway's portal; not recorded here.
- Exporting or importing records, charts beyond the two on the overview — not in the design.

## 4. Functional requirements

### 4.1 Dealer sign-in and protected admin area

**Requirement ADMIN-001 — The sign-in page**

*As the* dealer, *I want to* sign in with my email and password, *so that* I can reach the area.

Behaviour:

1. `/admin/sign-in` is a split screen: the stockroom picture (alt "Boxes stacked in a stockroom") at the left, above the form on a phone; at the right the wordmark, the heading "Đôi — dealer sign-in", the line "Sign in to see orders, stock and sales.", three lines each with an icon — "Orders to confirm, pack and send, newest first.", "Stock for every pair, size by size, and what sold this month.", "The sizes running out, before a customer finds them gone." — and the form.
2. The form has "Email" (placeholder `you@example.com`), "Password" (hidden characters) and a full-width "Sign in" button; Enter in either field submits. Under it the note "Accounts are for the dealer and staff. Ask the owner for one."
3. Submitting with the email empty shows "An email address is needed" beside the email field and sends nothing.
4. Right credentials of an admin or a staff account take the person to `/admin`. A customer's credentials sign the customer in too (one sign-in serves all three roles), and `/admin` then shows the team-only message (ADMIN-002 item 9).
5. Wrong credentials — an unknown email or a wrong password, alike — show "Wrong email or password." and nothing else about which was wrong. The password field is emptied.
6. While the shop is answering, the button reads "Signing in…" and is disabled.
7. When the shop cannot be reached the page shows "We cannot reach the server. Please try again." and the button is enabled.
8. When the sign-in is refused for too many tries (ADMIN-003) the page shows "Too many tries. Please wait 15 minutes."
9. A dealer who is already signed in sees under the form "Signed in as <email>. Continue", and Continue goes to `/admin`.
10. The page uses the management palette and type described in `design/design-system.md` §1.1 and the customer pages' header and footer do not appear.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | `/admin/sign-in` at 1280px | It loads | The picture is at the left; the wordmark, the heading "Đôi — dealer sign-in", the line "Sign in to see orders, stock and sales.", the three lines, the form and the note are at the right |
| AC-2 | The same at 390px | It loads | The picture is above the form |
| AC-3 | The form | The visitor reads it | "Email" has the placeholder `you@example.com`; "Password" hides what is typed |
| AC-4 | The right email and password | "Sign in" is pressed | The address becomes `/admin` and the overview shows |
| AC-5 | The right email and password | Enter is pressed in the password field | The same as AC-4 |
| AC-6 | The right email with a wrong password | "Sign in" is pressed | "Wrong email or password." is shown and the password field is empty |
| AC-7 | An email that has no account | "Sign in" is pressed | The same text and the same status as AC-6 |
| AC-8 | The email field is empty | "Sign in" is pressed | "An email address is needed" is shown and no request is sent |
| AC-9 | The shop answers slowly (3 seconds) | "Sign in" is pressed | The button reads "Signing in…" and is disabled until the answer arrives |
| AC-10 | The shop cannot be reached | "Sign in" is pressed | "We cannot reach the server. Please try again." is shown and the button is enabled |
| AC-11 | Five wrong tries were made | A sixth is made | "Too many tries. Please wait 15 minutes." is shown |
| AC-12 | A dealer signed in | `/admin/sign-in` is opened | "Signed in as <email>. Continue" is shown and Continue goes to `/admin` |
| AC-13 | A page of the customer site | It is searched for a link to `/admin` | Only the footer's "Dealer sign-in" is found; this holds until the customer-accounts story adds the header's "Account" entry (ACC-007), which links to `/account/sign-in`, never to `/admin`, and after it the footer's "Dealer sign-in" is still the only link to `/admin` |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | Email empty | AC-8 |
| Not permitted | Wrong credentials | One generic message (AC-6, AC-7) |
| Upstream failure | Shop unreachable | AC-10 |
| Boundary | Email typed in other case | `Dealer@Example.com` and `dealer@example.com` are the same account |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Email | text | yes | Compared without regard to case |
| Password | text | yes | Never shown, stored or logged in plain text |

**Requirement ADMIN-002 — The session and the gate on every `/admin` page**

*As the* dealer, *I want* my sign-in to last and to end when I say, *so that* nobody else gets in on my phone.

Behaviour:

1. Signing in gives the browser a session that cannot be read by the page's scripts, is sent only to this site, and expires 7 days after sign-in. `GET /v1/auth/me` answers the signed-in person's email, name and role ("admin", "staff" or "customer"), or 401.
2. The name is the account's own name when it has one (a staff member's, given by the admin; a customer's, given at sign-up); otherwise — the admin account, made from the owner's settings — the part of the email before `@`, with its words capitalised. The role is "admin", "staff" or "customer".
3. Every `/admin` page (except `/admin/sign-in`) works from the shop's answer to `/v1/auth/me`, not from anything stored in the browser. While that answer is awaited the page draws nothing of the area and does not redirect.
4. With no session, `/admin/*` goes to `/admin/sign-in`. A reload of an admin page with a session stays on that page.
5. A value placed in the browser's local storage by hand (for example under the old key `agentteam:session`) does not make a visitor signed in and does not open any page; the dealer's endpoints still answer 401.
6. Signing out — the sign-out button at the foot of the left menu, from any admin page — ends the session at the shop and goes to `/admin/sign-in`. After it, the same cookie no longer opens anything, and Back does not show data.
7. The left menu's foot shows the dealer's initial, name and role as drawn in the design.
8. `useSession`, `signIn`, `signOut` and `useRole` keep their names, so no other page changes. `useRole` returns "admin", "staff" or "customer", or nothing when nobody is signed in.
9. The gate lets an admin session and a staff session into every `/admin` page. A signed-in customer who reaches any `/admin` page except sign-in sees the plain message "This area is for the shop's team" with a "Back to the shop" link to `/`, and none of the area's pages or data; every dealer endpoint answers that session 403.
10. There is one users table with a role of admin, staff or customer. A staff row can sign in and is let into `/admin`; a customer row can sign in and is not (item 9).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The dealer signs in | The response is read | It sets one cookie, not readable by scripts, sent only to this site, expiring in 7 days |
| AC-2 | An admin session | `GET /v1/auth/me` | 200 with `{"email":…,"name":…,"role":"admin"}` |
| AC-3 | No session | `GET /v1/auth/me` | 401 |
| AC-4 | A session for `lan.tran@example.com` | The left menu's foot is read | The initial `L`, the name `Lan Tran` and the role `admin` |
| AC-5 | A signed-in dealer on `/admin/orders` | The page is reloaded | The page stays and shows the orders |
| AC-6 | No session | `/admin/orders` is opened | The address becomes `/admin/sign-in` |
| AC-7 | The answer to `/v1/auth/me` is slow (2 seconds) | An admin page is opened with a session | No admin content and no redirect appears until the answer arrives |
| AC-8 | The browser's storage holds a forged `agentteam:session` | `/admin` is opened | The address becomes `/admin/sign-in` |
| AC-9 | The same | `GET /v1/orders` is called | 401 |
| AC-10 | A signed-in dealer | The sign-out button is pressed | The address becomes `/admin/sign-in` and `GET /v1/auth/me` answers 401 |
| AC-11 | The same | The old cookie is sent again to `GET /v1/auth/me` | 401 |
| AC-12 | The same | The browser's Back button is pressed | The admin page is not shown with data; `/admin/sign-in` is |
| AC-13 | A session 8 days old | `GET /v1/auth/me` | 401 |
| AC-14 | `POST /v1/auth/sign-out` with no session | It is called | 204 (nothing to end) |
| AC-15 | Every page route of the area except sign-in | Each is opened with no session | Each goes to `/admin/sign-in` |
| AC-16 | A staff account | It signs in at `/admin/sign-in` | The address becomes `/admin`, the overview shows and `GET /v1/auth/me` says role `staff` |
| AC-17 | A signed-in customer | `/admin` and `/admin/orders` are opened | The text "This area is for the shop's team" and a link to `/` are shown; no admin page, list or figure is drawn |
| AC-18 | A signed-in customer | Any dealer endpoint is called | 403 |
| AC-19 | A session of each role, and no session | `useRole` is read | It returns "admin", "staff", "customer", and nothing |
| AC-20 | An admin account and a staff account | Both sign in | Both reach `/admin` and both see the orders, the shoes and the overview |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not permitted | No session | Redirect to sign-in; endpoints 401 |
| Upstream failure | `/v1/auth/me` cannot be reached | The area is not drawn and the visitor is not treated as signed in; the sign-in page is shown |
| Boundary | The session is exactly 7 days old | Expired |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Session: account, role, created time, expiry | write | yes | 7 days; ended by sign-out |

**Requirement ADMIN-003 — The dealer account and protection against guessing**

*As the* owner, *I want* the dealer account to come from my own settings and to resist guessing, *so that* nobody else gets in.

Behaviour:

1. When the server starts it makes sure one account exists with the email `ADMIN_EMAIL` and the role "admin", with `ADMIN_PASSWORD` as its password, stored only as a salted slow hash. If the account exists and the password setting changed, the stored password is replaced. It makes no staff or customer account: staff are added by the admin (ADMIN-030) and customers sign up themselves (`accounts` module).
2. The owner supplies both settings in the console.
3. A password shorter than 10 characters is refused at start: the account is not made or changed, the server still starts and logs why.
4. When `ADMIN_EMAIL` or `ADMIN_PASSWORD` is not set, no account is made and signing in is refused for everyone.
5. After 5 failed sign-ins for one email, or from one network address, within 15 minutes, further sign-ins for that email or from that address are refused for 15 minutes, even with the right password, with status 429 and a note of the wait.
6. A wrong password and an unknown email answer with the same status, the same body and a similar time.
7. Staff and customer accounts are rows of the same users table (email, name, password hash, role, active) and sign in through the same endpoint, with the same failure messages and the same limits. An account that is switched off is refused with the same "Wrong email or password." answer.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | `ADMIN_EMAIL=dealer@example.com` and a 12-character `ADMIN_PASSWORD` | The server starts on an empty database | Signing in with them succeeds and `/v1/auth/me` says role `admin` |
| AC-2 | The same | The stored password is read | It is not the typed password and is not reversible |
| AC-3 | The password setting is changed and the server restarts | The old password is tried | It is refused; the new one signs in |
| AC-4 | `ADMIN_PASSWORD` has 9 characters | The server starts | It starts; no account is made; the log says why; sign-in is refused |
| AC-5 | The settings are empty | The server starts | It starts; sign-in is refused for any email |
| AC-6 | Five wrong passwords for the dealer's email | A sixth try with the right password | 429 with the wait |
| AC-7 | Five failures from one address across five emails | A sixth try from it | 429 |
| AC-8 | An unknown email and a wrong password | Their answers are compared | Same status and same body |
| AC-9 | 15 minutes after the fifth failure | The right password is tried | It signs in |
| AC-10 | The settings example file | It is read | It names `ADMIN_EMAIL` and `ADMIN_PASSWORD` with no value |
| AC-11 | A staff row in the users table | It signs in with its password | Sign-in succeeds and `/v1/auth/me` says role `staff` |
| AC-12 | A customer row in the users table | It signs in with its password | Sign-in succeeds and `/v1/auth/me` says role `customer` |
| AC-13 | A staff account switched off | Its right password is sent | The same status and body as a wrong password (AC-8) |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Boundary | Password of exactly 10 characters | Accepted |
| Absent data | Settings not set | AC-5 |
| Conflict | Two sign-ins at the same moment | Both succeed and have separate sessions |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Dealer account: email, password hash, role | write | yes | One account from the settings |
| Failure counts per email and per address | write | yes | Reset after the wait |

**Requirement ADMIN-004 — Every dealer endpoint answers 401 or 403**

*As the* owner, *I want* every endpoint that changes records or reads private data to refuse outsiders, *so that* hiding a page is never the only lock.

Behaviour:

1. These answer 401 with no session and 403 with a session whose role is not "admin": creating, changing and deleting shoes, lines, categories, reviews and orders; uploading a picture; listing orders; reading an order without its customer token; the overview; and every other endpoint added for the dealer.
2. A visitor's reads of public collections (shoes, lines, categories, reviews, shop terms, delivery options, ways to pay, questions, size rows) stay open, within ADMIN-013.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | No session | `POST /v1/products` | 401 |
| AC-2 | No session | `PUT /v1/products/<id>` | 401 |
| AC-3 | No session | `DELETE /v1/reviews/<id>` | 401 |
| AC-4 | No session | `POST /v1/uploads` | 401 |
| AC-5 | No session | `GET /v1/admin/overview` | 401 |
| AC-6 | No session | `PUT /v1/orders/<id>` | 401 |
| AC-7 | A session with role "staff" (test fixture) | Each of the above | 403 |
| AC-8 | No session | `GET /v1/products` | 200 (public read) |
| AC-9 | A visitor | A write is tried on lines, categories, reviews, orders | 401 for each |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not permitted | No session / wrong role | 401 / 403 with a generic body, nothing changed |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Session role | read | yes | |

### 4.2 Manage shoes, lines, categories, reviews and orders

**Requirement ADMIN-010 — Saving records, checked against the schema**

*As the* dealer, *I want* what I save to be kept and checked, *so that* the shop never shows a broken record.

Behaviour:

1. The dealer can create (`POST /v1/<collection>`), change (`PUT /v1/<collection>/<id>`) and delete (`DELETE /v1/<collection>/<id>`) shoes, lines, categories and reviews. Orders are changed and deleted by the dealer (ADMIN-016) and created only by customers (`orders` module).
2. A save is checked against the collection's description in `schema.json`: a required field is present and not empty (shoe: name, price; line: name; category: name; review: shoe, customer); a number or price is a number, and a price is not below 0; a true/false field is true or false; a list is a list of text; a field with a closed set of choices (a shoe's status, main colour; a review's fit and status) holds one of them; a reference (a shoe's line and category, a review's shoe) names an existing record; a review's stars are a whole number from 1 to 5; text is at most 200 characters and long text at most 5,000.
3. A save that breaks a rule is refused with 422 and a body naming every field at fault with a sentence for it (for example `{"errors":{"price":"Price cannot be below 0"}}`); nothing is saved. A field the collection does not have is refused the same way.
4. A change keeps the fields it does not carry as they were; the id never changes.
5. A new record gets a unique id from the shop: for a shoe, its name in lower case with hyphens (`samba-og-new-colour`), with `-2`, `-3` added when taken; for other collections, a short unique text.
6. Saving a shoe with "Opens the home page" on turns that off on every other shoe, so one shoe at a time opens the home page.
7. When two saves of one record arrive, the later one wins and the record shows what was saved last.
8. Saving a shoe's sizes, last-pairs, sold-out and stock follows ADMIN-015.
9. A change shows on the public site at once: the next read of the shoe or list returns it.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A signed-in dealer | `POST /v1/products` with a name, price 3290000, line `samba`, category `women`, status `live` | 201 with the record and a new id `<slug of the name>` |
| AC-2 | The same | `GET /v1/products/<new id>` as a visitor | 200 with the saved values |
| AC-3 | A shoe named "Samba OG" exists with id `samba-og` | Another "Samba OG" is created | Its id is `samba-og-2` |
| AC-4 | The same | `POST /v1/products` with no name | 422 `{"errors":{"name":…}}`; no shoe is added |
| AC-5 | The same | A price of −1 | 422 naming `price` |
| AC-6 | The same | A price of `abc` | 422 naming `price` |
| AC-7 | The same | A price of 0 | 201 (zero is allowed) |
| AC-8 | The same | A status of `archived` | 422 naming `status` |
| AC-9 | The same | A line `zzz` that does not exist | 422 naming `line` |
| AC-10 | The same | Two bad fields (no name, price −1) | 422 naming both |
| AC-11 | The same | A field `colour_code` the collection does not have | 422 naming it |
| AC-12 | A review | Stars 6, then 0, then 2.5 | 422 naming `rating` each time; stars 5 is accepted |
| AC-13 | A name of 200 characters, then 201 | Each is saved | The first is accepted; the second is refused naming the limit |
| AC-14 | `PUT /v1/products/<id>` carrying only `price` | It is sent | The price changes; every other field is as it was; the id is as it was |
| AC-15 | `PUT` with another `id` in the body | It is sent | The id in the address is the one changed; the body's id is ignored |
| AC-16 | Shoe A opens the home page | Shoe B is saved with it on | A no longer opens the home page and the home page shows B |
| AC-17 | A shoe was set from live to draft | The next `GET /v1/products` as a visitor | It is not in the list |
| AC-18 | `PUT` of a record that does not exist | It is sent | 404 |
| AC-19 | Two `PUT`s of one shoe's price at the same moment | Both are sent | Both answer 200 and the stored price is that of the later one |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | Any rule in item 2 | 422 with per-field sentences (AC-4 to AC-13) |
| Not permitted | No session / not the dealer | 401 / 403 (ADMIN-004) |
| Not found | `PUT` or `DELETE` of an id that does not exist | 404 |
| Conflict | Two saves at once | Last write wins (AC-19) |
| Boundary | Price 0; name 200 characters | Accepted |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoe, line, category, review fields as in `schema.json` | write | per schema | As item 2 |

**Requirement ADMIN-011 — The dealer's screens save to the shop**

*As the* dealer, *I want* Save and Delete on the screens to really save and delete, *so that* my work survives a reload.

Behaviour:

1. The screens keep the design: the left menu (Overview; Sales: Orders with a count of orders waiting; Catalogue: Shoes, Lines, Categories, Reviews; the dealer at the foot with the sign-out button), the lists, the record pages and the dialogs. `saveRow`, `deleteRow` and `newId` keep their names and now call the shop.
2. A list shows its title, an "Add" button (not on Orders), a search box (placeholders "Search by name or style code" on Shoes, "Search by number or customer" on Orders, "Search reviews", "Search lines", "Search categories"), its filters, the table with its columns, the page numbers and page size, and for each row a menu with "Edit" and "Delete".
   - Shoes: columns picture, name, line, category, price, status, stock; filters Status ("On sale", "Draft") and For (the categories); a row opens `/admin/products/<id>`.
   - Orders: columns number, placed, status, shoe, total, payment, customer; filters Status (Placed, Confirmed, Packed, On its way, Delivered, Cancelled) and Payment (the ways to pay); a row opens `/admin/orders/<id>`.
   - Reviews: columns shoe, customer, stars, date, status; filter Status ("Shown", "Hidden").
   - Lines and categories: their fields in a dialog, with the explanatory line above the table.
3. The table can be sorted by a column, paged, and its search, filters, order and page are in the address.
4. "Add" and "Edit" open a dialog titled "Add to <list name>" or "Edit <record name>" (the latter with "Changes show on the site as soon as they are saved."), with the fields from the schema, "Cancel", and "Add" or "Save changes". A record page has a breadcrumb, a header with "Delete" and "Save changes", tabs by the schema's sections, and the notice "Saved." after a save.
5. Pressing Save shows a saving state: the button reads "Saving…" and is disabled. When the shop accepts the save the dialog closes (or the notice "Saved." shows) and the list redraws from the shop's answer, not from what was typed.
6. When the shop refuses with field errors, the dialog or page stays open, the typed values stay, and each sentence shows against its field.
7. When the shop cannot be reached or fails otherwise, the dialog or page stays open with the typed values and shows "We could not save. Please try again." and the button is enabled again.
8. "Delete" asks "Delete <record name>?" first; confirming deletes at the shop and redraws the list; a refusal (ADMIN-014) shows its sentence in the dialog.
9. A record page for an id that does not exist shows "This record is not here" with "It may have been deleted since the link was made." and a "Back to the list" button — only after the shop has answered, never while the record is being fetched.
10. The private order fields (phone, email, address, note) are on the dealer's order page only.
11. Lists with nothing show "Nothing here yet" with an "Add" button; a filtered list with no match shows "Nothing matches those filters" with "Clear filters".

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A signed-in dealer on `/admin/products` | "Add" is pressed, a name, a price and the other required fields are typed and "Add" is pressed | The dialog closes and the new shoe is in the list; after a reload it is still there |
| AC-2 | The same | The new shoe is `live` | `/shop` shows it after Save and after a reload |
| AC-3 | The same | The shoe is set to draft and saved | It is gone from `/shop`, from search and from the home page |
| AC-4 | A save in flight (2 seconds) | The dealer looks at the button | It reads "Saving…" and is disabled |
| AC-5 | The price is −1 | "Save changes" is pressed | The dialog stays open with the typed values and "Price cannot be below 0" (or the shop's sentence) shows against Price |
| AC-6 | The shop is down | "Save changes" is pressed | The dialog stays open with the typed values and "We could not save. Please try again." |
| AC-7 | A record page of a shoe | The price is changed and "Save changes" is pressed | "Saved." shows and the list shows the new price after a reload |
| AC-8 | The row menu | "Delete" is chosen and confirmed on a review | The review is gone after a reload |
| AC-9 | The same | "Delete" is chosen and "Cancel" pressed | Nothing is deleted |
| AC-10 | `/admin/products/nope` | The page opens | After the shop's answer, "This record is not here" with the note and "Back to the list" |
| AC-11 | The record is being fetched (2 seconds) | The page opens | "This record is not here" is not shown before the answer |
| AC-12 | `/admin/orders` | The page loads | There is no "Add" button |
| AC-13 | The Shoes list | The dealer reads it | The columns are picture, name, line, category, price, status, stock; the search placeholder is "Search by name or style code" |
| AC-14 | The Orders list | The dealer reads it | The columns are number, placed, status, shoe, total, payment, customer; the filters Status and Payment have the choices above |
| AC-15 | Two orders waiting (placed or confirmed) | Any admin page loads | The menu's Orders item shows the count 2 |
| AC-16 | A filter that matches nothing | The list loads | "Nothing matches those filters" and a "Clear filters" button |
| AC-17 | An empty list | The list loads | "Nothing here yet" and an "Add" button |
| AC-18 | A filtered, sorted, second page of the Shoes list | The address is copied and opened | The same filters, order and page show |
| AC-19 | A visitor on `/admin/products` | The page opens | The sign-in page shows (ADMIN-002) |
| AC-20 | An order's page | The dealer opens it | Its customer, phone, email, address and note are shown; no customer-site page shows them |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | The shop refuses with field errors | AC-5 |
| Upstream failure | The shop is down | AC-6; nothing is lost from the form |
| Not found | The record is gone | AC-10 |
| Not permitted | The session ends while the dealer works | The save answers 401; the dialog shows "Your session ended. Sign in again." with a link to `/admin/sign-in`, and the typed values stay |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoes, lines, categories, reviews, orders | read / write | per schema | Through the shop's interface only |

**Requirement ADMIN-012 — Shoe pictures are uploaded**

*As the* dealer, *I want to* add a picture of a shoe from my phone or computer, *so that* the shoe shows on the shop.

Behaviour:

1. The picture controls of the shoe form (side view, front pair, back pair) and of the category form send the chosen file to the shop (`POST /v1/uploads`) and keep the address the shop returns on the record; the picture is no longer copied into the form as text.
2. The shop accepts JPEG, PNG and WebP files, judged by their content, not their name, of at most 5 MB. It stores the file under a name it makes itself, serves it to everyone at its address, and returns that address.
3. Another type is refused with 415 and "Choose a JPEG, PNG or WebP picture."; a larger file with 413 and "That picture is too large (5 MB at most)."; both show next to the control and keep the form.
4. While a file is being sent the control shows it is working; the Save button waits for it.
5. A shoe with its pictures appears in `/shop` after Save and survives a reload.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A signed-in dealer | A 1 MB JPEG is uploaded | 201 with an address; opening it returns the same picture to a visitor |
| AC-2 | The same | A PNG and a WebP of 1 MB are uploaded | Both are accepted |
| AC-3 | A GIF | It is uploaded | 415 |
| AC-4 | A text file renamed `x.jpg` | It is uploaded | 415 |
| AC-5 | A JPEG of 5 MB, then 5 MB + 1 byte | Each is uploaded | The first is accepted; the second answers 413 |
| AC-6 | A file named `../../x.png` | It is uploaded | The stored name is the shop's own; no file is written outside the pictures' folder |
| AC-7 | A visitor | `POST /v1/uploads` | 401 |
| AC-8 | A new shoe with three uploaded pictures, status live | "Add" is pressed | The shoe appears in `/shop` with its pictures, and still does after a reload |
| AC-9 | An upload in flight | The dealer looks at the form | The control shows it is working and "Add" waits |
| AC-10 | A refused file | The form is read | The sentence shows beside the control and the other typed values are kept |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | Wrong type / too large | 415 / 413 (AC-3 to AC-5) |
| Boundary | Exactly 5 MB | Accepted |
| Not permitted | No session | 401 |
| Upstream failure | The shop cannot be reached | The control shows "We could not upload that picture. Please try again." |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Uploaded file | write | no | JPEG, PNG or WebP, 5 MB at most |
| Picture addresses on a shoe or category | write | no | The address the shop returned |

**Requirement ADMIN-013 — Visitors see only what is on sale or shown; ratings follow the shown reviews**

*As a* customer, *I want* only live shoes and shown reviews, with a rating that matches them, *so that* the shop shows what is true.

Behaviour:

1. For a visitor, lists and single reads of shoes return only those with status "live", and of reviews only those with status "live"; asking for a draft shoe by its id answers 404; asking for drafts by status returns none. The dealer sees all of them.
2. A shoe's rating is the average of the stars of its live reviews, to one decimal, and its review count is the number of its live reviews; with none, both are 0. They are recalculated each time a review is created, changed (stars, status, shoe) or deleted, and when its shoe is saved.
3. The shop's average rating and review count (shown on the About page) follow the live reviews of live shoes in the same way, and are recalculated each time one of those events happens or a shoe goes live or to draft.
4. The ratings seeded with the shoes are placeholders; they become the true figures the first time a review of that shoe changes.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A shoe in draft | A visitor reads `GET /v1/products` and `GET /v1/products/<id>` | The first does not contain it; the second answers 404 |
| AC-2 | The same | `GET /v1/products?status=draft` as a visitor | `total` is 0 |
| AC-3 | The same | The dealer reads them | The draft shoe is in the list and answers 200 |
| AC-4 | A hidden review | A visitor reads `GET /v1/reviews` and `GET /v1/reviews?product=<id>` | It is not there |
| AC-5 | A shoe with live reviews of 5 and 4 stars | The shoe is read | `rating` is 4.5 and `review_count` is 2 |
| AC-6 | The same, a third live review of 4 stars is added | The shoe is read | `rating` is 4.3 and `review_count` is 3 |
| AC-7 | The same, a review is hidden | The shoe is read | The rating and count are those of the remaining live reviews |
| AC-8 | The shoe's last live review is deleted | The shoe is read | `rating` is 0 and `review_count` is 0, and the shoe page shows no rating line (SITE-005) |
| AC-9 | Two live shoes with live reviews | The shop's terms are read | `rating` and `review_count` are the average and the number of all those reviews |
| AC-10 | A shoe is set to draft | The shop's terms are read | Its reviews are no longer in the shop's figures |
| AC-11 | A visitor | A draft shoe's id is opened in `/shop/<id>` | The shop answers 404 and the page shows "That pair is not in the shop any more." |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Boundary | No live review exists anywhere | The shop's rating and review count are 0 |
| Concurrency | Two reviews change at once | The figures equal the average and count of the final state |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoe: status, rating, review count | read / write | yes | Rating and count are derived |
| Review: stars, status, shoe | read / write | yes | |
| Shop terms: rating, review count | write | yes | Derived |

**Requirement ADMIN-014 — Deleting safely**

*As the* dealer, *I want* a delete never to break an order or leave a shoe without its line, *so that* the records stay whole.

Behaviour:

1. A shoe that any order refers to cannot be deleted: the shop answers 409 and the dialog says "Orders refer to this shoe. Set it to draft instead."
2. A shoe no order refers to is deleted together with its reviews.
3. A line or category that any shoe uses cannot be deleted: 409 and "<n> shoes use this. Move them first."
4. A review can always be deleted.
5. An order can be deleted only when it is cancelled (its pairs are already back on the shelf): otherwise 409 and "Cancel the order first."
6. The "Delete <name>?" question is asked before any of this (ADMIN-011).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A shoe with an order | `DELETE` it | 409; the shoe and its order remain; the dialog says "Orders refer to this shoe. Set it to draft instead." |
| AC-2 | The same shoe | Set to draft and saved | It is gone from the shop and still readable by the dealer and by the order |
| AC-3 | A shoe with no order and two reviews | `DELETE` it | 204; the shoe and its two reviews are gone |
| AC-4 | A line used by 3 shoes | `DELETE` it | 409 "3 shoes use this. Move them first." |
| AC-5 | A line used by no shoe | `DELETE` it | 204 |
| AC-6 | A category used by a shoe | `DELETE` it | 409 |
| AC-7 | A placed order | `DELETE` it | 409 "Cancel the order first." |
| AC-8 | A cancelled order | `DELETE` it | 204 |
| AC-9 | A review | `DELETE` it | 204 |
| AC-10 | A visitor | Any delete | 401 |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Conflict | The record is in use | 409 with the sentence above; nothing deleted |
| Not found | The id is gone | 404 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Orders' references to shoes; shoes' references to lines and categories | read | yes | Block deletes |

**Requirement ADMIN-015 — Stock stays one story when the dealer saves a shoe**

*As the* dealer, *I want* the sizes I mark in stock, on last pairs or sold out to be the ones customers see, *so that* the shoe page and the shelf agree.

Behaviour:

1. The shop keeps a count of pairs for each size (`orders` module, ORD-005). Saving a shoe with its sizes, last-pairs list and sold-out list sets the counts so they agree with those lists: a size in the sold-out list has 0 pairs; a size in the last-pairs list keeps its count if it is 1 or 2 and otherwise gets 2; any other size keeps its count if it is 3 or more and otherwise gets 3; a size taken off the shoe's sizes loses its count; a size added gets the count its list implies.
2. "Pairs in stock" is the sum of the counts after the save. A number typed into that field that differs is replaced by the sum, and the form shows the saved value.
3. A size in both the sold-out and last-pairs lists is treated as sold out.
4. Saving a shoe without touching its stock fields leaves its counts alone.
5. The lists are the dealer's control of stock; the form has no field for exact pairs per size (open question 3).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Size `40` has 2 pairs and is in the last-pairs list | The shoe is saved with `40` moved to the sold-out list | `40` has 0 pairs and stock falls by 2 |
| AC-2 | Size `41 1/3` is sold out | It is removed from the sold-out list and saved | It has 3 pairs and stock rises by 3 |
| AC-3 | Size `38` has 8 pairs | `38` is added to the last-pairs list and saved | `38` has 2 pairs |
| AC-4 | Size `38` has 1 pair and is in the last-pairs list | The shoe is saved unchanged | `38` still has 1 pair |
| AC-5 | A shoe whose counts sum to 18 | It is saved with "Pairs in stock" typed as 99 | The saved stock is 18 and the form shows 18 |
| AC-6 | A size in both lists | The shoe is saved | The size is sold out and has 0 pairs |
| AC-7 | A size is removed from the shoe's sizes | The shoe is saved | The size and its count are gone |
| AC-8 | A price-only save | `PUT` carries only `price` | Counts and lists are unchanged |
| AC-9 | After any save | The shoe's lists are read | They agree with the counts: 0 pairs in the sold-out list only, 1–2 in the last-pairs list only |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Concurrency | A customer's order and the dealer's save arrive together | The final counts equal the dealer's lists applied to the counts after the order |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoe: sizes, last-pairs sizes, sold-out sizes, pairs in stock | write | no | Counts derived as above |

**Requirement ADMIN-016 — The dealer handles an order**

*As the* dealer, *I want to* move an order along, give the courier's details and mark cash received, *so that* customers see the truth.

Behaviour:

1. An order moves one step at a time along placed → confirmed → packed → shipped → delivered. Skipping a step, going backwards, and leaving "delivered" or "cancelled" are refused with 422 and "An order cannot go from <status> to <status>."
2. An order that is placed, confirmed or packed can be cancelled. Cancelling puts its pairs back on the shelf once, and the shoes' lists and totals follow (ORD-005). A shipped order cannot be cancelled (open question 2).
3. The dealer can set the carrier, tracking number and expected date of an order that is not delivered or cancelled; the customer sees them on the order page at once.
4. The dealer can mark an order paid when its way to pay is cash on delivery and its status is shipped or delivered. An order paid online is marked paid only by the payment provider's notice; the dealer's attempt is refused with 422 "Only the payment provider marks this order paid." Marking an order unpaid is refused.
5. The dealer can change the customer's name, phone, email, address and city of an order that is not delivered or cancelled, and its note at any time. The order's number, placed time, pairs, total, way to pay and delivery option cannot be changed; trying answers 422 naming the field.
6. An order that holds more than one pair shows its pairs on the dealer's order page: under the summary sentence a list of each pair — picture, name, EU size, quantity, unit price. The sentence for such an order reads "<customer> in <city> ordered <n> pairs on <date> — <total>." (open question 4). An order of one pair keeps the sentence in the design.
7. The orders list shows, for a multi-pair order, the first pair's shoe in the "shoe" column.
8. The dealer's order page shows the private fields: phone, email, address and note.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A placed order | The dealer saves it as confirmed | 200; status is confirmed |
| AC-2 | A placed order | The dealer saves it as packed | 422 "An order cannot go from placed to packed." |
| AC-3 | A shipped order | The dealer saves it as packed | 422 |
| AC-4 | A delivered order | The dealer saves it as shipped | 422 |
| AC-5 | A packed order of 2 pairs of size `38` (3 pairs on the shelf after the order) | The dealer saves it as cancelled | Status is cancelled and `38` has 5 pairs |
| AC-6 | The same order | The dealer saves cancelled again | 422 and the shelf is unchanged (the pairs are not returned twice) |
| AC-7 | A shipped order | The dealer saves it as cancelled | 422 |
| AC-8 | A shipped order | The dealer sets carrier `GHN`, tracking `GHN-PT27XM`, expected `2026-11-26` | They are saved and the customer's order page shows them |
| AC-9 | A delivered cash-on-delivery order | The dealer marks it paid | `paid` is true and the order page says "Cash on delivery · paid" |
| AC-10 | A placed cash-on-delivery order | The dealer marks it paid | 422 (not yet shipped) |
| AC-11 | A shipped VNPay order | The dealer marks it paid | 422 "Only the payment provider marks this order paid." |
| AC-12 | A paid order | The dealer marks it unpaid | 422 |
| AC-13 | A packed order | The dealer changes the address | Saved |
| AC-14 | A delivered order | The dealer changes the address | 422 naming `address`; the note can still change |
| AC-15 | Any order | The dealer changes `total` | 422 naming `total` |
| AC-16 | An order of two pairs | The dealer opens it | Under the sentence "<customer> in <city> ordered 2 pairs on <date> — <total>." both pairs are listed |
| AC-17 | An order of one pair | The dealer opens it | The sentence of the design is shown: "<customer> in <city> ordered <qty> × <shoe>, EU <size>, on <date> — <total>." |
| AC-18 | An order | The dealer opens it | Phone, email, address and note are shown on the Customer tab |
| AC-19 | A visitor | `PUT /v1/orders/<id>` | 401 |
| AC-20 | The dealer's orders list for an order of two pairs | The list loads | The "shoe" column shows the first pair's shoe |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | An impossible step, a refused field | 422 with its sentence (AC-2 to AC-4, AC-15) |
| Conflict | The customer's payment notice and the dealer's cancel arrive together | One wins; a cancelled order that is also paid keeps status cancelled and paid true (`payments` module) |
| Not permitted | No session / not the dealer | 401 / 403 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Order: status, carrier, tracking, expected date, paid, customer, phone, email, address, city, note | write | per rule | As above |
| Per-size pairs | write | yes | Returned once on cancel |

### 4.3 Dealer dashboard figures and charts

**Requirement ADMIN-020 — The overview's figures, computed by the shop**

*As the* dealer, *I want* the shop to add up my sales and stock, *so that* the overview is right however many orders there are and no customer's order is sent to my browser to be added up.

Behaviour:

1. `GET /v1/admin/overview`, for the dealer only (ADMIN-004), answers, in one request, everything the overview page shows, added up in the database:
   - this month's sales (the sum of the totals of orders placed in the current calendar month) and the number of those orders;
   - the number of orders waiting to be packed (status placed or confirmed);
   - pairs on the shelf (the sum of every shoe's pairs in stock, drafts included) and the number of shoes;
   - the shoes running low — those with a size that has one or two pairs left and fewer than 15 pairs in all — each with its id, name, category, picture, pairs in stock and last-pairs sizes, fewest pairs first;
   - sales for each of the last 14 days, today and the 13 before it, oldest first, every day present with 0 when nothing sold;
   - pairs sold by line, for every line (0 when none), most pairs first, equal counts in the lines' own order;
   - the five latest orders, newest first, of any status: id, number, customer, shoe of the first pair, EU size of the first pair, status, placed time, total;
2. Cancelled orders are not counted in any figure or chart. They can appear in the five latest orders, shown as cancelled.
3. "This month", "today" and each day of the 14-day series use Vietnam time (UTC+7). A seeded or stored time without a zone is read as Vietnam time.
4. For an order of several pairs, "pairs sold by line" adds the quantity of each pair to its shoe's line.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The 14 seeded orders DOI-24804 to DOI-24817 and the clock at 25/11/2026 12:00 Vietnam time | `GET /v1/admin/overview` as the dealer | This month's sales are 47.810.000 ₫ over 13 orders (DOI-24811 is cancelled and left out) |
| AC-2 | The same | The same | Orders waiting to be packed: 2 (DOI-24816 placed, DOI-24815 confirmed) |
| AC-3 | The same | The same | The 14-day series runs from 12/11 to 25/11, 14 entries; 25/11 is 3.290.000 ₫, 23/11 is 5.380.000 ₫, 20/11 and 16/11 are 0, and the 14 amounts add up to 47.810.000 ₫ |
| AC-4 | The same | The same | Pairs sold by line: Samba 8, Gazelle 2, Megaride 2, Handball Spezial 1, Adizero 1, in that order |
| AC-5 | The same | The same | The five latest orders are DOI-24816, DOI-24815, DOI-24814, DOI-24817, DOI-24813 in that order |
| AC-6 | DOI-24811 (cancelled, 4.090.000 ₫, 20/11) | The 20/11 entry of the series is read | It is 0 |
| AC-7 | An order placed at 23:30 Vietnam time on 30/11/2026 and one at 00:30 on 01/12/2026 | The clock is 01/12/2026 12:00 | The first is in November's month and the second in December's, and the 01/12 entry holds the second |
| AC-8 | Shoes with 5, 14, 15, 18 pairs, the first three with a last-pairs size and the fourth without | The low list is read | The shoes with 5 and 14 pairs are listed, 5 first; the 15-pair shoe (not fewer than 15) and the 18-pair shoe are not |
| AC-9 | A shoe with 9 pairs, no last-pairs size | The low list is read | It is not listed |
| AC-10 | Shoes whose pairs in stock are 18, 24, 31 | Pairs on the shelf is read | It is 73 and the shoe count is 3 |
| AC-11 | A shoe in draft with 10 pairs | Pairs on the shelf is read | The 10 pairs are counted |
| AC-12 | No orders at all | The overview is read | Sales 0, order count 0, waiting 0, 14 entries of 0, every line 0, no latest orders |
| AC-13 | An order of two pairs, one Samba and one Gazelle | The lines are read | Each line gains that pair's quantity |
| AC-14 | No session | `GET /v1/admin/overview` | 401 |
| AC-15 | A session that is not the dealer's | `GET /v1/admin/overview` | 403 |
| AC-16 | A new order is placed | The overview is read again | This month's sales and order count include it |
| AC-17 | An order is cancelled | The overview is read again | It is no longer in any figure |
| AC-18 | 10,000 orders | The overview is read | The answer is one request's worth of figures, not the orders themselves: it holds no order other than the five latest and no customer phone, email, address or note |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not permitted | No session / not the dealer | 401 / 403 (AC-14, AC-15) |
| Absent data | No orders, no shoes, no lines | Zeros and empty lists, not an error (AC-12) |
| Boundary | An order at 00:00:00 Vietnam time on the first of a month | It belongs to the new month |
| Upstream failure | The database cannot be reached | 500 with a generic body |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Orders: status, total, placed time, pairs | read | yes | Cancelled excluded from figures |
| Shoes: pairs in stock, last-pairs sizes, line, category | read | yes | |
| Lines | read | yes | |

**Requirement ADMIN-021 — The overview page**

*As the* dealer, *I want to* open the area and see what needs me today, *so that* I act on the day's work first.

Behaviour:

1. `/admin` shows the title "Overview" and four figures, each opening a list: "Sales this month" (the amount, note "<n> orders") opens `/admin/orders`; "To pack" (the count, note "placed or confirmed") opens `/admin/orders?status=confirmed`; "Pairs in stock" (the count, note "across <n> shoes") opens `/admin/products`; "Running low" (the count of shoes, note "last pairs left") opens `/admin/products`.
2. Under them, five panels: "Today" with the sentence "<waiting> orders are waiting to be packed and <low> shoes are down to their last pairs. This month so far: <n> orders, <sales>."; "Sales, last 14 days" as 14 columns labelled with the day of the month, each carrying its amount as text (a day with no sales is a short pale column); "Pairs sold by line" as one bar per line with its count; "Latest orders" as up to five rows, each `<number> · <customer> · <shoe> · EU <size>` with, at the right, `<status> · <date, time> · <total>`, linking to `/admin/orders/<id>`; "Running low" as rows with the shoe's picture, name, "Last pairs in <sizes>" and pairs left, linking to `/admin/products/<id>`, or "Nothing is running low." when none.
3. The page reads the one overview answer (ADMIN-020) and nothing else for its figures and charts; it does not download the orders or shoes lists.
4. While the answer is awaited the page shows "Loading the overview…" under the title. When it cannot be fetched it shows "We could not load the overview." with a "Try again" button.
5. The page works on a phone: the figures stack one, then two across; the left menu opens as a sheet.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The seeded orders at 25/11/2026 | `/admin` loads | "Sales this month" reads `47.810.000 ₫` with the note "13 orders" |
| AC-2 | The same | The page loads | "To pack" reads 2 with the note "placed or confirmed" |
| AC-3 | Shoes with 73 pairs over 3 shoes | The page loads | "Pairs in stock" reads 73 with "across 3 shoes" |
| AC-4 | Two shoes running low | The page loads | "Running low" reads 2 with "last pairs left" |
| AC-5 | The four figures | Each is pressed | They open `/admin/orders`, `/admin/orders?status=confirmed`, `/admin/products`, `/admin/products` |
| AC-6 | 2 waiting, 5 low, 13 orders, `47.810.000 ₫` | The page loads | "Today" reads "2 orders are waiting to be packed and 5 shoes are down to their last pairs. This month so far: 13 orders, 47.810.000 ₫." |
| AC-7 | The seeded orders | The 14-day chart is read | 14 columns labelled 12 to 25, each with its amount as text; 16 and 20 are short pale columns |
| AC-8 | The seeded orders | "Pairs sold by line" is read | Samba 8, Gazelle 2, Megaride 2, Handball Spezial 1, Adizero 1, each with a bar and its count |
| AC-9 | The seeded orders | "Latest orders" is read | Five rows, the first `DOI-24816 · Trang · Handball Spezial Earth Strata · EU 37 1/3` with `Placed · 25/11/2026, 08:40 · 3.290.000 ₫` |
| AC-10 | The same | The first row is pressed | The address is `/admin/orders/doi-24816` |
| AC-11 | A shoe running low | "Running low" is read | A row with its picture, name, `Last pairs in <sizes>` and its pairs left, linking to `/admin/products/<id>` |
| AC-12 | No shoe running low | "Running low" is read | "Nothing is running low." |
| AC-13 | A network log | `/admin` loads | One request for the overview; no request for a list of orders or of shoes with a limit above 1 (the menu's order count asks for one order at most) |
| AC-14 | The overview answer is slow (3 seconds) | `/admin` loads | "Loading the overview…" is shown, and no figures of 0 |
| AC-15 | The overview answers 500 | `/admin` loads | "We could not load the overview." and a "Try again" button; pressing it asks again |
| AC-16 | A 390px window | `/admin` loads | The four figures stack in one column; the panels fit with no horizontal scroll; the menu is behind its button |
| AC-17 | No session | `/admin` is opened | The sign-in page shows (ADMIN-002) |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Upstream failure | The overview cannot be fetched | AC-15 |
| Absent data | No orders | Figures of 0, the 14-day chart of short columns, every line at 0, "Latest orders" without rows |
| Not permitted | No session | AC-17 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| The overview answer | read | yes | Dealer only |

## 5. Screens

| Screen | Section in the design | Functions it serves | States that must exist |
|---|---|---|---|
| Sign-in `/admin/sign-in` | Sign in (split screen) | ADMIN-001, ADMIN-002 | default; "An email address is needed"; already signed in ("Signed in as … Continue") |
| Sign-in — signing in, wrong credentials, cannot reach, too many tries | Not drawn in the approved design; required by the plan item | ADMIN-001, ADMIN-003 | signing in; wrong credentials; cannot reach the server; too many tries |
| Area gate (every `/admin/*` page) | Not drawn: nothing is shown while the shop answers `/v1/auth/me` | ADMIN-002 | checking (nothing drawn); signed out (redirect) |
| Overview `/admin` | Overview | ADMIN-020, ADMIN-021 | default (figures, five panels); "Nothing is running low." |
| Overview — loading, load failed | Not drawn in the approved design; required by the plan item | ADMIN-021 | loading; error |
| Lists: Shoes, Lines, Categories, Reviews, Orders | Lists, with the Add / Edit dialog and the delete question | ADMIN-010, ADMIN-011, ADMIN-012, ADMIN-014 | default; empty ("Nothing here yet"); filtered empty ("Nothing matches those filters"); dialog open; delete question |
| Lists and dialogs — saving, field errors, could not save, session ended | Not drawn in the approved design; required by the plan item | ADMIN-011 | saving; field errors; could not save; session ended |
| Shoe `/admin/products/:id` | Record page | ADMIN-010, ADMIN-011, ADMIN-012, ADMIN-015 | default (tabs); saved ("Saved."); record not here |
| Order `/admin/orders/:id` | Record page with the summary sentence | ADMIN-011, ADMIN-016 | default (tabs Order, Delivery, Customer); saved; record not here |
| Order — pairs list of a multi-pair order | Not drawn in the approved design; required by the plan item's multi-pair orders | ADMIN-016 | multi-pair |

## 6. Non-functional requirements

| Area | Requirement |
|---|---|
| Security | The session cookie is not readable by scripts, is sent only to this site, is marked secure over HTTPS and expires 7 days after sign-in; passwords are kept only as salted slow hashes; after 5 failed sign-ins for one email or from one address, further sign-ins are refused for 15 minutes |
| Security | No dealer endpoint answers a visitor's request with data; every one answers 401 or 403 (checked for every route listed in ADMIN-004) |
| Privacy | An order's phone, email, address and note are returned only to the dealer; the overview holds none of them |
| Performance | The overview answers within 500 ms at p95 with 10,000 orders and 1,000 shoes over 100 sequential requests against the running API |
| Performance | A save or delete answers within 1 s at p95 over 100 sequential requests |
| Upload | A file over 5 MB is refused; only JPEG, PNG and WebP (by content) are stored |
| Responsive | The sign-in page and every dealer page work from 320px up with no horizontal page scroll; the left menu is a sheet below 1024px |
| Accessibility | Every form field has a visible label; wrong-credentials, field errors and the overview's loading and error messages are announced to assistive technology; the sign-in form works with Enter |
| Localisation | Copy in English; money as `vi-VN`; dates `dd/mm/yyyy`; "this month" and "today" in Vietnam time |

## 7. Dependencies and assumptions

- **Depends on:** the `data` module (collections and the paged list), the `catalogue` module (search, "any of" filters, sorting on the dealer's lists and the menu's order count), the `orders` module (per-size counts, order pairs, tokens), the `payments` module (online orders' paid state and the 30-minute rule).
- **Depends on:** the owner supplying `ADMIN_EMAIL` and `ADMIN_PASSWORD` in the console.
- **Assumption:** the shop has one dealer account; no staff accounts are created in this release.
- **Assumption:** until the dealer sign-in story ships, the old pages work on the mock session; from this story they need the real session.

| # | Open question | Proposed default | Who decides |
|---|---|---|---|
| 1 | Session length and password rule | A session lasts 7 days; the owner's password is at least 10 characters | Stakeholder |
| 2 | May the dealer cancel an order that has shipped? | No: only placed, confirmed or packed orders can be cancelled; a shipped order that comes back is handled as an exchange outside the screens | Stakeholder |
| 3 | The design's stock tab has "Pairs in stock" and three lists but no exact pairs per size, so a restock is done by moving a size out of "Sold out" or "Last pairs in" | As ADMIN-015: the lists are the dealer's control and counts follow; no per-size count form is added | Stakeholder |
| 4 | An order can now hold several pairs, which the design's order page and sentence (one shoe per order) do not draw; the order page copy also says the size can change until the order is packed | Multi-pair orders get a pairs list and the sentence in ADMIN-016; changing the pairs of a placed order is not built (cancel and reorder) | Stakeholder |
| 5 | Copy for states the design does not draw: "Signing in…", "Wrong email or password.", "We cannot reach the server. Please try again.", "Too many tries. Please wait 15 minutes.", "Saving…", "We could not save. Please try again.", "Your session ended. Sign in again.", "Loading the overview…", "We could not load the overview.", "Try again", "Choose a JPEG, PNG or WebP picture.", "That picture is too large (5 MB at most).", "We could not upload that picture. Please try again." | Use this copy, added to the site's content so the owner can edit it | Stakeholder |
| 6 | "To pack" counts placed and confirmed orders but opens the Orders list filtered to "Confirmed" only, as the design has it | Keep as designed | Stakeholder |
| 7 | The ratings and review counts seeded with the shoes (and the shop's 4.8 from 181 reviews) are placeholders; they become the true figures the first time a review changes | As ADMIN-013 item 4 | Stakeholder |

## 8. Traceability

| Plan item | Requirement ids | Test cases |
|---|---|---|
| Dealer sign-in and protected admin area | ADMIN-001, ADMIN-002, ADMIN-003, ADMIN-004 | `test-cases/dealer-sign-in-and-protected-admin-area.md` |
| Manage shoes, lines, categories, reviews and orders | ADMIN-010, ADMIN-011, ADMIN-012, ADMIN-013, ADMIN-014, ADMIN-015, ADMIN-016 | `test-cases/manage-shoes-lines-categories-reviews-and-orders.md` |
| Dealer dashboard figures and charts | ADMIN-020, ADMIN-021 | `test-cases/dealer-dashboard-figures-and-charts.md` |
