# SRS — Accounts: customer sign-up, sign-in and my orders

Module: `accounts`
Design: [View the approved design](http://localhost:9401)
Design system: `design/design-system.md`

> One file per module. This one covers customer accounts: signing up and in, the My account page, saved delivery details and linking orders to the account. The sign-in, sign-out and `/v1/auth/me` endpoints, the users table, the three roles and the team's own sign-in are in `docs/admin/SRS.md`; placing and reading orders are in `docs/orders/SRS.md`.

## 1. Purpose

A customer who buys more than once should not retype their name, phone and address, and should find their orders without keeping order numbers. The owner answered that three kinds of people sign in: customers, staff and admin. This module is the customer part: sign up, sign in, see my orders, keep my delivery details, and have a new order land in my account. An account is optional. Buying as a guest and tracking an order by number and phone keep working for everyone who does not want one. Without this module a returning customer starts from nothing every time and the header has no "Account" entry.

## 2. Actors

| Actor | Who they are | What they may do in this module |
|---|---|---|
| Visitor | No session | Buy as a guest, track an order by number and phone, sign up, sign in |
| Customer | Signed in, role "customer" | Everything a visitor does, plus: see their own orders, keep saved delivery details, add a guest order to the account, sign out |
| Another customer | A different signed-in customer | Nothing of the first customer's: their orders answer 404 |
| Shop team | An admin or a staff session (`admin` module) | Not served by these screens; they sign in at `/admin/sign-in`. A team member who signs in on `/account/sign-in` is taken to `/admin` |

## 3. Scope

**In scope** — the plan item, by title:

- Customer accounts: sign up, sign in, my orders

**Out of scope**

- The shared sign-in, sign-out and `/v1/auth/me` endpoints, the session cookie, the roles, the users table and the sign-in limits — belong to the `admin` module (ADMIN-001 to ADMIN-003); this module uses them.
- Placing an order, the order page and tracking by number and phone — belong to the `orders` module and stay unchanged for a guest; this module adds the link between an order and an account (ORD-003 item 9, ORD-008).
- Staff accounts — `admin` module (ADMIN-030, ADMIN-031).
- "Forgot password", changing the password or the email, deleting the account, sign-in with another service, verifying the email address — no screen in the design and no plan item.
- Email or text messages of any kind — not built.

## 4. Functional requirements

### 4.1 Customer accounts: sign up, sign in, my orders

**Requirement ACC-001 — Signing up**

*As a* visitor, *I want to* make an account with my name, email, phone and a password, *so that* my details and orders are kept for next time.

Behaviour:

1. `/account/sign-up` shows the heading "Create your account", the line "Optional — you can always buy without one.", a form with "Name", "Email", "Phone", "Password" (hidden characters) and a "Create account" button, and under it "Already have an account? Sign in" linking to `/account/sign-in`.
2. All four fields are required. Name has at most 100 characters; email must look like an email and has at most 200 characters; phone follows the checkout rule (ORD-002): at least 9 digits, spaces, `+`, `-` and brackets not counted, at most 20 characters; password has 8 to 100 characters. Each failure shows its sentence beside the field: "Fill in your name", "Enter a valid email address", "Check the phone number", "Use at least 8 characters".
3. `POST /v1/account/sign-up` with name, email, phone and password creates an account of role "customer", stores the password only as a salted slow hash, signs the customer in with the session of ADMIN-002 and answers 201 with the customer's name, email and role. The browser goes to `/account`. The shop checks the same rules again and answers 422 naming each field at fault.
4. An email that belongs to any account — customer, staff or admin — compared without regard to case, is refused with 409; the page shows "An account with this email already exists. Sign in instead." beside the email field.
5. While the shop answers, the button reads "Creating your account…" and is disabled. When the shop cannot be reached the page shows "We cannot reach the server. Please try again." and keeps the typed values except the password.
6. More than 10 sign-ups from one network address within an hour are refused with 429 and the page shows "Too many tries. Please wait and try again." (open question 2).
7. A signed-in customer who opens this page is taken to `/account`.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | `/account/sign-up` at 1280px | It loads | The heading, the line, the four labelled fields, the button and the sign-in link are shown |
| AC-2 | Name `Lan Nguyen`, email `lan@example.com`, phone `0912 345 678`, a 10-character password | "Create account" is pressed | 201; the customer is signed in, the address becomes `/account` and `GET /v1/auth/me` says role `customer` |
| AC-3 | The stored account | It is read | The password is not stored in the clear; no response ever carries a password or a hash |
| AC-4 | An account with `lan@example.com` exists | `LAN@example.com` is used to sign up | 409; "An account with this email already exists. Sign in instead." shows beside the email field |
| AC-5 | A staff or admin account with the email | The same email is used to sign up | The same 409 |
| AC-6 | Any field empty | "Create account" is pressed | The sentence for each empty field is shown; nothing is sent |
| AC-7 | A password of 7 characters, then of 8 | Each is sent | 422 naming `password` the first time; the second is accepted |
| AC-8 | A phone `0912 34` | It is sent | 422 naming `phone`; `+84 912 345 678` is accepted |
| AC-9 | The shop answers slowly (3 seconds) | The button is pressed | It reads "Creating your account…" and is disabled |
| AC-10 | The shop cannot be reached | The button is pressed | "We cannot reach the server. Please try again." is shown; the typed values stay, the password does not |
| AC-11 | Eleven sign-ups from one address within an hour | The eleventh is sent | 429 and "Too many tries. Please wait and try again." |
| AC-12 | A signed-in customer | `/account/sign-up` is opened | The address becomes `/account` |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | A missing or bad field | AC-6 to AC-8, with 422 from the shop |
| Conflict | Two sign-ups with one email at the same moment | One is created; the other gets 409 (AC-4) |
| Upstream failure | The shop is unreachable | AC-10 |
| Boundary | A name of 100 characters; 101 | Accepted; refused naming the limit |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Name | text | yes | At most 100 characters |
| Email | text | yes | Unique across all roles, compared without regard to case |
| Phone | text | yes | At least 9 digits, at most 20 characters |
| Password | text | yes | 8 to 100 characters; stored only as a salted slow hash |
| Role | text | yes | "customer", set by the shop, never by the request |

**Requirement ACC-002 — Signing in as a customer**

*As a* customer, *I want to* sign in with my email and password, *so that* I reach my account.

Behaviour:

1. `/account/sign-in` shows the heading "Sign in", a form with "Email" and "Password" and a "Sign in" button, "New here? Create an account" linking to `/account/sign-up`, and the line "Buying as a guest needs no account."
2. It uses the shared sign-in of ADMIN-001 to ADMIN-003, so its states, words and limits are the same: "Wrong email or password." for an unknown email and a wrong password alike, "Signing in…", "We cannot reach the server. Please try again.", and "Too many tries. Please wait 15 minutes." after 5 failures for one email or one network address within 15 minutes.
3. A customer with the right credentials is taken to `/account`.
4. A team member (admin or staff) who signs in here is signed in and taken to `/admin`.
5. A signed-in customer who opens this page is taken to `/account`.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | `/account/sign-in` | It loads | The heading, the two labelled fields, the button, the sign-up link and the guest line are shown |
| AC-2 | A customer's right credentials | "Sign in" is pressed | The address becomes `/account` |
| AC-3 | A right email with a wrong password, and an unknown email | Each is tried | "Wrong email or password." for both, with the same status and body |
| AC-4 | Five wrong tries for one email | A sixth is made, even with the right password | 429 and "Too many tries. Please wait 15 minutes." |
| AC-5 | The shop answers slowly (3 seconds), then cannot be reached | The button is pressed | "Signing in…" with the button disabled; then "We cannot reach the server. Please try again." |
| AC-6 | A staff account's right credentials | They are used here | The address becomes `/admin` |
| AC-7 | A signed-in customer | `/account/sign-in` is opened | The address becomes `/account` |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not permitted | Wrong credentials, switched-off account | One generic message (AC-3) |
| Upstream failure | The shop is unreachable | AC-5 |
| Boundary | Email typed in other case | The same account |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Email, password | text | yes | As ADMIN-001; the password is never shown, stored or logged in the clear |

**Requirement ACC-003 — My account: my orders**

*As a* customer, *I want to* see my own orders, *so that* I find them without an order number.

Behaviour:

1. `/account` for a signed-in customer shows the heading "My account", the line "Hello, <name>", the blocks "My orders", "Saved delivery details" (ACC-004) and "Add an order" (ACC-006), and a "Sign out" button (ACC-007).
2. "My orders" lists the customer's own orders, newest first, ten at a time with a "Show more" button. Each row shows the order number, the date (`dd/mm/yyyy`), the total, the status (Placed, Confirmed, Packed, On its way, Delivered or Cancelled) and links to `/orders/<id>`. With none it shows "You have no orders yet." and a "Keep shopping" pill to `/shop`.
3. `GET /v1/account/orders` answers the signed-in customer's own orders in the paged envelope of DATA-002; each item holds id, number, placed time, status and total, and no phone, email, address or note. `GET /v1/account/orders/<id>` answers one of them. An order that is not the customer's, or does not exist, answers 404 with one and the same body. No session answers 401; a team session answers 403.
4. A row opens the existing Order page, which a signed-in customer reads without a token for an order linked to their account (ORD-008).
5. With no session, `/account` goes to `/account/sign-in`. While `/v1/auth/me` is awaited nothing of the page is drawn.
6. While the orders are awaited the block shows "Loading your orders…"; when they cannot be fetched it shows "We could not load your orders." with a "Try again" button.
7. The page works on a phone: one column from 320px up.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A customer who placed two orders while signed in | `/account` opens | "My account", "Hello, <name>" and two rows, newest first, each with number, date, total, status and a link to its order |
| AC-2 | The same customer on another device | They sign in | The same two orders are listed |
| AC-3 | Twelve orders | `/account` opens | Ten rows and a "Show more" button; pressing it adds the other two |
| AC-4 | A customer with no order | `/account` opens | "You have no orders yet." and a "Keep shopping" pill to `/shop` |
| AC-5 | A row | It is pressed | The Order page of that order opens and shows it without any token held in this browser |
| AC-6 | Customer B and customer A's order id | `GET /v1/account/orders/<id>` is called as B | 404 with the same body as for an id that does not exist |
| AC-7 | The same | `/orders/<id>` is opened as B, with no token | "We can't find that order." |
| AC-8 | Any item of `GET /v1/account/orders` | It is read | It has no `phone`, `email`, `address` or `note` |
| AC-9 | No session, then a staff session | `GET /v1/account/orders` | 401, then 403 |
| AC-10 | No session | `/account` is opened | The address becomes `/account/sign-in` |
| AC-11 | The orders answer takes 3 seconds, then fails | `/account` opens | "Loading your orders…"; then "We could not load your orders." and a "Try again" button that asks again |
| AC-12 | A 390px window | `/account` opens | One column; no horizontal scroll |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not permitted | No session / a team session | 401 / 403 (AC-9) |
| Permission | Another customer's order | 404 (AC-6, AC-7) |
| Upstream failure | The shop is unreachable | AC-11 |
| Absent data | No orders | AC-4 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Orders of this account: id, number, placed time, status, total | read | yes | Own orders only; never phone, email, address or note |

**Requirement ACC-004 — Saved delivery details**

*As a* customer, *I want to* keep my name, phone and address, *so that* checkout is filled in for me.

Behaviour:

1. "Saved delivery details" on `/account` shows "Name", "Phone", "Street address, ward, district" and "City or province" filled with the saved values, and a "Save details" button. Name and phone come from sign-up; address and city are empty until saved.
2. `GET /v1/account` answers the signed-in customer's own name, email, phone, address and city, never a password or hash. `PUT /v1/account` changes name, phone, address and city with the limits of checkout (ORD-002): name at most 100 characters, phone at least 9 digits and at most 20 characters, address at most 200, city at most 100. A field left out is unchanged. The email, the role and the password cannot be changed here: a body carrying `email`, `role` or `password` is refused with 422 naming the field. Both calls work on the session's own account; neither takes an id.
3. Pressing "Save details" shows "Saving…" with the button disabled, then "Saved." A refusal shows its sentence against the field; when the shop cannot be reached the block shows "We could not save. Please try again." and keeps the typed values.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A new customer | `/account` opens | Name and phone hold the sign-up values; address and city are empty |
| AC-2 | The block | Address `12 Example Street, Ward 1` and city `Da Nang` are typed and saved | "Saved."; after a reload the fields hold them |
| AC-3 | The same | `PUT /v1/account` carries only `phone` | The phone changes; name, address and city are as they were |
| AC-4 | A body with `email`, then with `role` | `PUT /v1/account` | 422 naming `email`, then `role`; nothing changes |
| AC-5 | A phone of 8 digits, then an address of 201 characters | Each is saved | 422 naming `phone`, then `address` and the limit |
| AC-6 | A save in flight (2 seconds) | The button is read | "Saving…" and disabled |
| AC-7 | The shop is down | "Save details" is pressed | "We could not save. Please try again."; the typed values stay |
| AC-8 | No session, then a team session | `GET /v1/account` and `PUT /v1/account` | 401, then 403 |
| AC-9 | `GET /v1/account` | It is read | No password or hash is present |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | A rule above is broken | 422 with a sentence per field (AC-4, AC-5) |
| Not permitted | No session / a team session | 401 / 403 (AC-8) |
| Upstream failure | The shop is unreachable | AC-7 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Name, phone, address, city | read / write | per rule | The limits of ORD-002 |
| Email, role, password | read | — | Not changed here |

**Requirement ACC-005 — Checkout with an account, and checkout as a guest**

*As a* customer, *I want* checkout to use my saved details and keep the order in my account, *so that* I do not retype them; *as a* guest, *I want* to buy with no account at all.

Behaviour:

1. A signed-in customer who opens `/checkout` finds "Full name", "Phone", "Street address, ward, district" and "City or province" filled from the saved details. Every field stays editable and the note is empty. A field with no saved value is empty.
2. An order placed while a customer is signed in is linked to that account, whatever details were typed, and appears in My account at once and on any device after signing in. The order's own details are the typed ones; the saved details are not changed by an order.
3. A guest checks out exactly as in `docs/orders/SRS.md`: no account, no sign-in prompt, the order token and the lookup by number and phone as before. An order placed as a guest is linked to no account.
4. If the session ends between opening checkout and pressing "Place order", the order is placed as a guest, not refused.
5. An order placed by an admin or a staff session is linked to no account: only role "customer" links.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A customer with saved details | `/checkout` opens with a pair in the bag | The four fields hold the saved values and are editable |
| AC-2 | The same customer | An order is placed with typed details that differ | The order holds the typed details; the account's saved details are unchanged |
| AC-3 | The same | `/account` is opened after a reload | The order is listed; it is also listed after signing in on another device |
| AC-4 | A guest | A pair is bought and the order tracked by number and phone | Both work with no account, and the order is in no account's list |
| AC-5 | A guest order and a second customer | The second customer reads the order's number in My account | It is not there |
| AC-6 | The session ends before "Place order" | The order is placed | It is saved as a guest order and its page opens with its token |
| AC-7 | A staff session | An order is placed | It is linked to no account |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Absent data | A customer with no saved address | The fields stay empty (AC-1) |
| Not applicable | Roles | A guest needs no account (AC-4) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Order: customer account | write | no | Set only for a session of role "customer"; empty for a guest |
| Saved details | read | no | Fill the form; never changed by an order |

**Requirement ACC-006 — Adding a guest order to my account**

*As a* customer, *I want to* add an order I placed as a guest, *so that* it sits with my others.

Behaviour:

1. "Add an order" on `/account` has "Order number" (placeholder `DOI-24817`), "Phone" and an "Add order" button.
2. `POST /v1/account/orders` with the number and the phone checks them exactly as the lookup does (ORD-021): the number is accepted with or without `DOI-`, in any case, with spaces or dashes; the phone is compared by its digits, a leading `84` standing for `0`, and must match in full. A right pair for an order linked to no account links it to this account; the page shows "Added." and My orders redraws with it.
3. A wrong number, a wrong phone, a number that does not exist and an order already linked to another customer's account all answer 404 with one and the same body; the page shows "No order matches that number and phone. Check both, or message us." in alert style.
4. These attempts count against the lookup's limits (ORD-022): after 5 failures within 15 minutes for one order number, or from one network address, further attempts are refused for 15 minutes with 429 and "Too many tries. Please wait 15 minutes, or message us with your order number."
5. Adding an order that is already linked to this account answers "Added." and adds no second row.
6. Linking does not remove the token a guest browser holds: that browser still opens the order, and tracking by number and phone still works.
7. No session answers 401; a team session answers 403.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A guest order `DOI-24818` for phone `0912 345 678` | A customer adds `doi 24818` and `+84912345678` | "Added."; the order is in My orders |
| AC-2 | The same | The wrong phone, then an unknown number | The same 404 and the same message for both |
| AC-3 | The order is already in customer A's account | Customer B adds it with the right pair | The same 404 and message as AC-2 |
| AC-4 | Five failures for one number | A sixth try, even with the right pair | 429 and "Too many tries. Please wait 15 minutes, or message us with your order number." |
| AC-5 | The order is already in this account | It is added again | "Added." and one row |
| AC-6 | The order was linked | The guest browser opens `/orders/<id>` with its token, and tracks the order by number and phone | Both still work |
| AC-7 | No session, then a team session | `POST /v1/account/orders` | 401, then 403 |
| AC-8 | The phone is empty | "Add order" is pressed | 404 and the not-found message |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | Wrong pair / another account's order | One generic answer (AC-2, AC-3) |
| Permission | Locked by too many tries | 429 (AC-4) |
| Upstream failure | The shop is unreachable | "We could not add that order. Please try again." |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Typed order number and phone | read | yes | Compared on digits; never stored |
| Order: customer account | write | yes | Set once, on a right pair |

**Requirement ACC-007 — The header's "Account" entry, and signing out**

*As a* visitor, *I want* the header to offer sign-in, and to show who I am once I have, *so that* I can find my account from any page.

Behaviour:

1. From this story on, every customer page's header has an "Account" entry beside the search and bag buttons from 1024px, and as a link in the full-screen menu below it. Signed out it reads "Sign in" and goes to `/account/sign-in`; signed in as a customer it shows the customer's name and goes to `/account`; signed in as admin or staff it shows their name and goes to `/admin`.
2. The footer's "Dealer sign-in" stays the team's entrance and the only link to `/admin` that a signed-out visitor sees on a customer page (ADMIN-001 AC-13).
3. "Sign out" on My account ends the session at the shop (`POST /v1/auth/sign-out`) and goes to `/`. After it the same cookie opens nothing, and `/account` goes to `/account/sign-in`.
4. The header's state comes from `/v1/auth/me`, not from the browser's storage. While the answer is awaited the entry shows neither a name nor "Sign in".
5. This story changes SITE-001: its header (AC-1) and menu (AC-3) hold with the entry added.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A signed-out visitor, 1280px | Any customer page loads | The header shows "Sign in" beside the search and bag buttons and it links to `/account/sign-in` |
| AC-2 | A signed-in customer "Lan Nguyen" | Any page loads | The entry shows "Lan Nguyen" and links to `/account` |
| AC-3 | A staff member "Minh" signed in | Any customer page loads | The entry shows "Minh" and links to `/admin` |
| AC-4 | A 390px window | The menu is opened | It lists the five links, the entry, then the help links |
| AC-5 | A customer page | It is searched for links to `/admin` | Only the footer's "Dealer sign-in" (for a signed-out visitor) |
| AC-6 | A signed-in customer | "Sign out" is pressed | The address becomes `/`, `GET /v1/auth/me` answers 401 and the header shows "Sign in" |
| AC-7 | The same | The old cookie is sent again | 401 |
| AC-8 | A forged value in the browser's storage | A page loads | The header shows "Sign in" |
| AC-9 | The answer of `/v1/auth/me` is slow (2 seconds) | A page loads | The entry shows neither a name nor "Sign in" until it arrives |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Upstream failure | `/v1/auth/me` cannot be reached | The header shows "Sign in" |
| Not applicable | Roles | Every role sees the entry (AC-1 to AC-3) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Session: name, role | read | no | From `/v1/auth/me` only |

**Requirement ACC-008 — A customer sees only their own data, and never the team's**

*As a* customer, *I want* nobody else to see my orders or details, *so that* my phone and address stay mine.

Behaviour:

1. A customer session on any dealer endpoint answers 403 (ADMIN-004), and `/admin` shows "This area is for the shop's team" (ADMIN-002 item 9).
2. No account endpoint takes an account id: they work on the session's own account, so a customer cannot name another's.
3. Another customer's order answers 404 on `GET /v1/account/orders/<id>` and on `GET /v1/orders/<id>` without a token. The lookup by number and phone still works for anyone who holds the right pair, signed in or not.
4. No response carries another customer's data, and no account response carries a password or a hash.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A customer session | `GET /v1/orders`, `POST /v1/products`, `GET /v1/admin/overview`, `GET /v1/admin/staff` | 403 for each |
| AC-2 | A customer session | `/admin` is opened | "This area is for the shop's team" and a link to `/`; no admin page |
| AC-3 | Customer A's order | Customer B reads it through `/v1/account/orders/<id>` and `/v1/orders/<id>` | 404 for both, with the same body as for an id that does not exist |
| AC-4 | A second customer who knows A's order number but not the phone | They look it up | The generic not-found answer |
| AC-5 | The account endpoints | Their routes are listed | None takes an account id |
| AC-6 | Any account response | It is read | No password or hash |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Permission | Another customer's data | 404 (AC-3) |
| Not permitted | A customer on a dealer endpoint | 403 (AC-1) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Account, orders of the account | read | yes | The session's own only |

## 5. Screens

| Screen | Section in the design | Functions it serves | States that must exist |
|---|---|---|---|
| Sign up `/account/sign-up` | Not in the approved design: a new page defined by the plan item, built from the site's own parts (display heading, text fields, filled pill, hairlines) | ACC-001 | default; field errors; email taken; creating; cannot reach the server; too many tries |
| Sign in `/account/sign-in` | Not in the approved design: as above | ACC-002 | default; signing in; wrong credentials; cannot reach the server; too many tries |
| My account `/account` | Not in the approved design: as above | ACC-003, ACC-004, ACC-006 | default; no orders; loading; failed; show more; saving; saved; field errors; adding an order; order not found; too many tries |
| Header "Account" entry | Header (SITE-001), changed by this story | ACC-007 | signed out ("Sign in"); signed in (name); in the menu below 1024px; loading (nothing drawn) |
| Checkout `/checkout` | Checkout (ORD-001) | ACC-005 | default with saved details filled in |

## 6. Non-functional requirements

| Area | Requirement |
|---|---|
| Security | Passwords are kept only as salted slow hashes; the session cookie is the one of ADMIN-002; sign-in failures are limited as in ADMIN-003; wrong credentials never say which part was wrong |
| Privacy | A customer's phone, address and orders are returned to that customer and to the shop team only; no response to another customer, to a guest or to a stranger holds them |
| Responsive | Sign up, sign in and My account work from 320px up with no horizontal page scroll |
| Accessibility | Every field has a visible label; error and "Wrong email or password." messages are announced to assistive technology; the forms work with Enter and the keyboard alone |
| Performance | `GET /v1/account/orders` answers within 300 ms at p95 for an account with 100 orders over 100 sequential requests |
| Localisation | Copy in English; money as `vi-VN`; dates `dd/mm/yyyy` |

## 7. Dependencies and assumptions

- **Depends on:** the `admin` module's sign-in story (the users table, the roles, the session, `/v1/auth/sign-in`, `/v1/auth/sign-out`, `/v1/auth/me`, ADMIN-001 to ADMIN-004), which must be built first.
- **Depends on:** the `orders` module's checkout story (orders, the order token, the lookup rules ORD-021 and ORD-022, the account link in ORD-003 item 9 and ORD-008), which must be built first.
- **Assumption:** an email address is not verified and no email is sent; there is no "forgot password" (out of scope).
- **Assumption:** the customer's password has at least 8 characters; the owner's own password has at least 10 (ADMIN-003).

| # | Open question | Proposed default | Who decides |
|---|---|---|---|
| 1 | The minimum length of a customer's password | 8 characters | Stakeholder |
| 2 | Limit on sign-ups per network address, so the form cannot be used to fill the database | 10 an hour per address, then 429 | Stakeholder |
| 3 | A team member who signs in on the customer sign-in page | Signed in and sent to `/admin` | Stakeholder |
| 4 | Copy for the states the design does not draw ("Create your account", "Optional — you can always buy without one.", "You have no orders yet.", "Added.", "Saved.", the error sentences) | Use this copy, added to the site's content so the owner can edit it | Stakeholder |
| 5 | No "forgot password" or email check exists, so a customer who forgets the password has no way back in | Not built; the owner may ask for a story later | Stakeholder |

## 8. Traceability

| Plan item | Requirement ids | Test cases |
|---|---|---|
| Customer accounts: sign up, sign in, my orders | ACC-001, ACC-002, ACC-003, ACC-004, ACC-005, ACC-006, ACC-007, ACC-008 | `test-cases/customer-accounts-sign-up-sign-in-my-orders.md` |
