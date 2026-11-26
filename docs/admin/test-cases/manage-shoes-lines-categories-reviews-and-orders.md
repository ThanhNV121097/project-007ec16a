# Test cases — Manage shoes, lines, categories, reviews and orders

Module: `admin` · Plan item 10 · Story: `docs/admin/stories/manage-shoes-lines-categories-reviews-and-orders.md` (SC-1 … SC-39)
Requirements: ADMIN-010, ADMIN-011, ADMIN-012, ADMIN-013, ADMIN-014, ADMIN-015 and the write half of ADMIN-004 (`docs/admin/SRS.md` §4.1, §4.2) · Contract: `docs/architecture/services.md` §2.2 (roles), §2.3 (error contract), §2.4 (views), §2.5 (validation boundary), §2.8 (the product shape), §3.22 (`POST` / `PUT` / `DELETE` on products, lines, categories, reviews), §3.23 (`PATCH /v1/products/{id}/sizes`), §3.24 (`POST /v1/uploads`), §3.6 (`GET /v1/uploads/{name}`)

Written before the implementation exists and before this story's technical design, from the requirements, the story and the contract that is already merged. The order half of the original item is **not** here (story §4: it is the plan item "Dealer handles orders: status, courier, payment").

## 1. Risk and depth

| Part | Risk | Why | Depth |
|---|---|---|---|
| Who may write (SC-1 … SC-3) | **High** | Every write changes what the public shop shows and what customers pay; a hole here is a public defacement | All 14 write calls × no session, forged cookie, expired cookie, customer, staff, admin; the guard answers before the body is read; nothing changed after each refusal; the five collections that have no write route answer 404 |
| Saving records and validation (SC-4 … SC-10) | **High** | The shop never shows a broken record; two fields at fault must both be named; a price below 0 or a 201-character name must not reach the database | Every rule of SC-5 … SC-7 value by value with the boundary one either side, several faults at once, nothing saved on refusal, partial `PUT`, the later of two saves, the one featured shoe under a race |
| Visitor view and ratings (SC-11 … SC-13) | **High** | A draft shoe or a hidden review shown to the public is a leak; a rating that does not follow the reviews is a false claim | Every public read (list, search, size filter, filters, by id) × draft and hidden, both dealer roles and the customer, every review event, the shop's figures, the seeded figures untouched until an event |
| Deleting safely (SC-14 … SC-16) | **High** | A delete that breaks an order or leaves a shoe without its line is unrecoverable | Order-referenced shoe, placeholder included; free shoe with reviews and counts; line and category in use then freed; review in both statuses; second delete 404 |
| Pictures (SC-17 … SC-19) | **High** | Public file store written by a login: type judged by content, size capped at the byte, name chosen by the shop | JPEG / PNG / WebP, GIF, a text file named `.jpg`, a PNG named `.txt`, empty, 5,242,880 and 5,242,881 bytes, a path in the file name, multipart faults |
| The pairs in each size (SC-20 … SC-25) | **High** | The one stock figure; an order takes pairs from it; a silent overwrite or a count below 0 sells a pair that is not there | Derived lists and total, `PATCH` compare-and-set, 409 after an order, a whole-form `PUT` guarded the same way, the race between an order and a change, no second copy of stock read through a real order |
| The screens (SC-26 … SC-38) | Medium–High | The states the design never drew (saving, refused, unreachable, session ended) decide whether work is lost | Saving, field errors with the values kept, unreachable, session ended, delete refusals, record not here, the lists, pictures, stock counts, one count saving by itself, 390 and 320 px |
| Speed (SC-39) | Low | A number the story fixes | 100 saves and 100 deletes, p95 ≤ 1 s |
| Contract rules (C-1 … C-9) | Medium | The backend is reviewed against them | Envelope, `X-Request-Id`, 415, 400, 413, `Location`, a failed write (500), an undefined field on `PATCH` (400) |

## 2. How to read this file

- **Addresses.** Paths are written as the backend mounts them (`/v1/products`); from a browser or `fetch_url` they are `/api/v1/products`. A request that needs a session sends `Cookie: session=<value>` taken from the `Set-Cookie` of `POST /v1/auth/sign-in`; JSON bodies carry `Content-Type: application/json`.
- **Sessions.** `ADMIN`: sign-in with the run's `ADMIN_EMAIL` / `ADMIN_PASSWORD`. `STAFF`: sign-in as FX-STAFF (made with `POST /v1/admin/staff` as ADMIN: name "Tc Staff", `tc-staff@example.com`, password `staff-pass-A1`). `CUSTOMER`: a customer row made as FX-STAFF is, then `UPDATE users SET role = 'customer' WHERE email = 'tc-customer@example.com'` (name "Tc Customer", password `customer-pass-1`); it signs in with the shared sign-in. `NONE`: no cookie. `FORGED`: `session=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA`. `EXPIRED`: STAFF's cookie after `UPDATE sessions SET created_at = now() - interval '8 days', expires_at = now() - interval '1 day' WHERE user_id = (SELECT id FROM users WHERE email = 'tc-staff@example.com')` (a second sign-in then makes a live one again). In a browser case "signs in as X" means typing the email and password on `/admin/sign-in` and pressing "Sign in".
- **The error body.** Every non-2xx answer is JSON `{"error":<code>,"message":<non-empty sentence>}` with `Content-Type` starting `application/json`, plus `"errors":{<field>:<non-empty sentence>}` **only** on 400, 422 and the 409 `stock_conflict` (`services.md` §2.3); a 401, 403, 404, 409 `conflict`, 413, 415 and 500 body has no `errors` key. Codes used: `unauthenticated` 401, `forbidden` 403, `not_found` 404, `conflict` 409, `payload_too_large` 413, `unsupported_media_type` 415, `invalid` 422, `bad_request` 400, `internal` 500. No body holds SQL, a stack trace, a path, a hash or a token. Every answer carries `X-Request-Id`. "Plain English" for a field sentence means: non-empty, at most 100 characters, no character outside printable ASCII except `’` and `…`, so it can sit beside its field.
- **Seeded records the cases name.** Shoes (12): `samba-og-w-core-black` (featured; an order refers to it), `samba-og-w-crystal-white`, `samba-og-w-wonder-quartz`, `handball-spezial-w-earth-strata`, `gazelle-indoor-solar-gold`, `adizero-adios-wonder-blue`, `adizero-evo-sl-charcoal`, `megaride-s2-cloud-white`, `megaride-bones-halo-green`, `samba-og-kids-wonder-mauve`, `samba-og-kids-cloud-white`, and `megaride-ag-xl-core-black` (the only one no sample order refers to; it has the reviews `r-34`, `r-35`, `r-36`). Lines (5): `samba`, `handball-spezial`, `gazelle`, `adizero`, `megaride`. Categories (3): `women`, `men`, `kids`. Reviews `r-01` … `r-36`. Size counts of a seeded shoe: 0 for a size in its sold-out list, 1 for a size in its last-pairs list, 3 for every other size, so `samba-og-w-core-black` size `38` holds 3.
- **SNAP** — the read that proves "nothing changed", through `set_db_state`: `SELECT (SELECT count(*) FROM products) AS p, (SELECT count(*) FROM lines) AS l, (SELECT count(*) FROM categories) AS c, (SELECT count(*) FROM reviews) AS r, (SELECT count(*) FROM uploads) AS u, (SELECT count(*) FROM product_sizes) AS ps, (SELECT price FROM products WHERE id = 'samba-og-w-core-black') AS price, (SELECT pairs FROM product_sizes WHERE product_id = 'samba-og-w-core-black' AND size = '38') AS pairs38, (SELECT blurb FROM lines WHERE id = 'samba') AS line_blurb, (SELECT blurb FROM categories WHERE id = 'women') AS cat_blurb, (SELECT rating FROM reviews WHERE id = 'r-01') AS r01`. Taken before and after a case; "nothing changed" means the two rows are equal.
- **SHELF(product, size, n)** — `UPDATE product_sizes SET pairs = <n> WHERE product_id = '<product>' AND size = '<size>'`; the old value is read first and put back.
- **PICS** — three picture addresses made once by FX-PIC below.
- **REQ** — the body of an order, sent as `POST /v1/orders` with a fresh `Idempotency-Key: tc-<case>-<n>`: `{"items":[{"product":"<id>","size":"<size>","qty":<n>}],"delivery":"standard","payment":"vnpay","customer":"Lan Nguyen","phone":"0912 345 678","address":"12 Example Street, Ward 1","city":"Da Nang","note":""}`.
- **GUARD(shoe).** Story §5 item 2 and §8 note 1 make a whole-shoe `PUT` that carries counts for existing sizes refuse a size an order has moved since the form was read, and the field that carries "what the form last read" is named by the technical design, which does not exist yet. Every case below whose own rule is something else and that sends `pairs` for an existing size on a `PUT` also sends that guard field set to the counts **currently stored** (read from the dealer view's `pairs`), so the case tests its own rule only. The case that tests the guard itself is TC-57 (its race with an order is TC-58, and TC-71 reads the request of a form whose counts were not touched); the verifier reads the guard's name from the extended `services.md` (§3.22) and, if it is not there, reports TC-57 UNVERIFIED with that reason.
- **Concurrent requests** ("at the same moment") are sent by one `interact_page` run that signs in as the account in the browser and fires the requests with `Promise.all` from the page (`fetch` with `credentials: "include"`); the case says how many rounds. Binary uploads are built in that page too (`new Blob([bytes])` in a `FormData` with the single part `file`), because a 1 MB or 5 MB body cannot be typed.
- **Byte prefixes** (`http.DetectContentType` reads the first 512 bytes): JPEG `FF D8 FF E0`, PNG `89 50 4E 47 0D 0A 1A 0A`, WebP `52 49 46 46 <4 bytes> 57 45 42 50 56 50 38 20`, GIF `47 49 46 38 39 61`, a plain text file `Hello, this is not a picture.`; the rest of a test file is zero bytes up to the size the case names. A picture that must *draw* in the browser (TC-62, TC-78, TC-79) is a real PNG the page makes with a canvas (`canvas.toBlob`) and hands to the file input with a `DataTransfer`.
- **Slow and failing arrangements** (a browser case that needs the shop to take 2 seconds or to fail): the page's request to the named URL is held, answered or aborted by request interception inside the `interact_page` run when the instrument has it. Where it has not, the slowness is made in the database — `CREATE FUNCTION tc_sleep() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(2); RETURN NEW; END $$; CREATE TRIGGER tc_slow BEFORE INSERT ON products FOR EACH ROW EXECUTE FUNCTION tc_sleep();` (and the same on `uploads`, `INSERT` or `UPDATE` as the case needs), removed after with `DROP TRIGGER tc_slow ON <table>; DROP FUNCTION tc_sleep();` — and "the shop is down" is `set_backend_state` stopping the backend (the gateway answers 502) and starting it again. "The session ends" is `DELETE FROM sessions WHERE user_id = (SELECT id FROM users WHERE email = '<account>')` while the page is open. A case that needs one of these and finds no instrument for it is UNVERIFIED for that reason, not failed.
- **Elements are found by role, label or visible text, never by class.** The page title is `main h1`; a dialog is `[role=dialog]` (or `[role=alertdialog]` for the delete question); a field is found by its visible label; "the sentence beside a field" is the text of the element the field's `aria-describedby` names or of the element in the same field wrapper, and the field carries `aria-invalid="true"`; "announced" means that sentence, "Saving…", "Saved." or a refusal sits inside an element with `role="alert"`, `role="status"` or an `aria-live` region; a row's menu button is the one `button` in the row's last cell; the left menu is `nav`. The shoe page's size pills are `[data-testid='size-<size>']` (a size such as `36 2/3` is `size-36-2-3`); the home page's lead link is `[data-testid='home-lead']` (its `href` is `/shop/<id of the featured shoe>`).
- **Browser words** are checked word for word: lists "Shoes", "Lines", "Categories", "Reviews"; "Add"; dialog titles "Add to shoes", "Add to lines", "Add to categories", "Add to reviews" (the list's title in lower case, as the approved `ListPage` draws it) and "Edit <record name>" with "Changes show on the site as soon as they are saved."; "Cancel"; "Save changes"; "Saving…"; "Saved."; "Delete <record name>?"; "Nothing here yet"; "Nothing matches those filters"; "Clear filters"; "This record is not here"; "It may have been deleted since the link was made."; "Back to the list"; "Choose a picture", "Replace", "Remove"; "We could not save. Please try again."; "Your session ended. Sign in again."; "We could not upload that picture. Please try again."; "Choose a JPEG, PNG or WebP picture."; "That picture is too large (5 MB at most)."; "Pairs were just taken by an order. Check the number and save again."; "Orders refer to this shoe. Set it to draft instead."; "<n> shoes use this. Move them first."; "Pairs in EU <size>"; "Last pairs in"; "Sold out in"; "Pairs in stock"; "That pair is not in the shop any more.".

### Fixtures

| Id | What is made | Used by |
|---|---|---|
| FX-PIC | Three PNG uploads as ADMIN (`POST /v1/uploads`), each a 1 KB file starting with the PNG prefix; the three returned `url`s are PICS (`/api/v1/uploads/<32 hex>.png`) | TC-04, TC-49 and every case that says "three pictures" |
| FX-SHOE | `POST /v1/products` as STAFF: `{"name":"Tc Shoe","price":3290000,"line":"samba","category":"women","status":"live","image":PICS[0],"image_back":PICS[1],"image_detail":PICS[2],"pairs":{"38":4,"39":2,"40":0}}` → id `tc-shoe`; dealer view `sizes` `["38","39","40"]`, `sizes_low` `["39"]`, `sizes_out` `["40"]`, `stock` 6 | most of §8 and §10 |
| FX-REVIEWS | Two live reviews on `tc-shoe` made with `POST /v1/reviews` as STAFF: `{"product":"tc-shoe","name":"Tc Five","rating":5}` and `{"product":"tc-shoe","name":"Tc Four","rating":4}` | TC-30, TC-32 … TC-35, TC-43 |
| FX-OTHER | A second live shoe `tc-other` (`{"name":"Tc Other","price":2000000,"status":"live","pairs":{"38":3}}`) | TC-22, TC-33, TC-37, TC-38 |

### Restoring the database after a case or a group

Seeded records the cases touch are put back so no case depends on another and the shared database is left as it was found. **Before the first case** the verifier records, through `set_db_state`, `SELECT id, price, status, featured, rating, review_count FROM products`, `SELECT product_id, size, pairs FROM product_sizes`, `SELECT id, rating, status, product_id FROM reviews` and `SELECT rating, review_count FROM shop_terms`. **After a group:** `DELETE FROM orders WHERE is_sample = false;` then `DELETE FROM reviews WHERE length(id) <> 4;` then `DELETE FROM products WHERE id NOT IN (<the 12 seeded ids>);` (counts and reviews go with them; the orders are removed first because a shoe an order refers to cannot be deleted); `DELETE FROM lines WHERE id NOT IN ('samba','handball-spezial','gazelle','adizero','megaride');` `DELETE FROM categories WHERE id NOT IN ('women','men','kids');` then every recorded price, status, `rating`, `review_count`, size count and review row written back, `UPDATE products SET featured = false; UPDATE products SET featured = true WHERE id = 'samba-og-w-core-black';` and `UPDATE shop_terms SET rating = 4.8, review_count = 181 WHERE id = 'shop';` (or the recorded values). Staff and customer fixtures: `DELETE FROM users WHERE email LIKE 'tc-%';`. Uploads made by the run: `DELETE FROM uploads WHERE uploaded_by IN (SELECT id FROM users WHERE email LIKE 'tc-%') OR uploaded_by = (SELECT id FROM users WHERE role = 'admin');` only for rows created after the run began.

**Order of the run.** TC-36 (the seeded figures are untouched) runs **first of all**, before any case that creates, changes or deletes a review or switches a shoe's status, because the first such event replaces the seeded 4.8 / 181 for good.

## 3. Open points found while writing (reported, not resolved here)

1. **The whole-form guard has no name yet.** Story §5 item 2 and §8 note 1 require a whole-shoe `PUT` that carries a count an order has moved to be refused with 409 (SC-23), and a form with no touched count to carry none (SC-24), but the field that carries "the counts the form last read" is the technical design's. TC-71 and TC-73 are written against the behaviour, and the verifier fills in the field's name from the extended contract. If the design settles on something that does not need a field (for example a version stamp), the two cases change their `Given` and keep their `Then`.
2. **A review with no stars.** SC-7 says "an empty value is accepted" and `services.md` §3.22 says `rating` is "integer 1–5 or `null`". The cases send `null` and an omitted field and expect 201; they do not pin what `""` does. They also do not pin whether such a review counts in `review_count` and the average (the story says "average of its live reviews' stars"); TC-43 therefore never mixes a review with no stars into a figure it checks.
3. **How the shop's own average is rounded.** ADMIN-013 item 3 says the shop's rating follows the live reviews "in the same way" as a shoe's, so the cases expect one decimal (the seeded 4.8 has one). TC-48 computes the expectation in SQL with `round(avg(...), 1)`.
4. **`id` in the body of a `POST`.** `services.md` §2.5 tolerates an `id` only on a `PUT`; §3.22 does not list it as writable, which makes it "a field the collection does not have" (422). The story (§3 item 9) says no id made in the browser is ever sent. TC-79 pins the browser side (no `id` in the request); no API case pins what a `POST` carrying an `id` answers.
5. **Whether a client-side check may refuse a blank name before any request.** The approved `ListPage` has a `missing()` check. SC-29 asks for "the shop's sentence beside the faulty field". TC-80 accepts a sentence that came from either, and pins the shop's words only for the price, whose rule the story states.
6. **A refused database.** `services.md` §2.3 makes a database that cannot be reached 500 `internal` on a write; with no database even the session lookup fails, so no case here pins that status. C-8 arranges a failed *write* (a renamed table), which is `internal` 500 under the rule the backend is built to.
7. **Deleting a placeholder.** TC-56 deletes the placeholder `megaride-ag-xl-core-black`, which the cases cannot recreate with an API call (the shop makes ids). The verifier backs the rows up (column lists written out, the generated `search_text` column left out) and puts them back; if it cannot do that faithfully it runs TC-56 on FX-SHOE's twin and reports the placeholder half UNVERIFIED with that reason rather than leave the shared database damaged.
8. **Instruments the browser cases lean on.** TC-78 … TC-97 need a held, failed or aborted request, a stopped backend, a deleted session mid-run and a file handed to a file input. Each says which; a case whose instrument is missing is UNVERIFIED for that reason.

## 4. The write matrix W1 … W14

The fourteen writes the story's SC-1 and SC-2 name. Each is sent with a **valid** body, so a refusal can only be the guard's. Ids are seeded records that must still be there after a refusal.

| Id | Call | Body |
|---|---|---|
| W1 | `POST /v1/products` | `{"name":"Tc Denied","price":1000}` |
| W2 | `PUT /v1/products/samba-og-w-core-black` | `{"price":1}` |
| W3 | `DELETE /v1/products/megaride-ag-xl-core-black` | — |
| W4 | `POST /v1/lines` | `{"name":"Tc Denied"}` |
| W5 | `PUT /v1/lines/samba` | `{"blurb":"Tc denied"}` |
| W6 | `DELETE /v1/lines/megaride` | — |
| W7 | `POST /v1/categories` | `{"name":"Tc Denied"}` |
| W8 | `PUT /v1/categories/women` | `{"blurb":"Tc denied"}` |
| W9 | `DELETE /v1/categories/kids` | — |
| W10 | `POST /v1/reviews` | `{"product":"samba-og-w-core-black","name":"Tc Denied","rating":1}` |
| W11 | `PUT /v1/reviews/r-01` | `{"rating":1}` |
| W12 | `DELETE /v1/reviews/r-02` | — |
| W13 | `POST /v1/uploads` | `multipart/form-data`, one part `file`: PNG prefix + 32 zero bytes, named `x.png` |
| W14 | `PATCH /v1/products/samba-og-w-core-black/sizes` | `{"size":"38","pairs":9,"seen_pairs":3}` |

## 5. Who may write (SC-1 … SC-3)

**Scenario**: TC-01 — with no session every write answers 401 and changes nothing
**Given**: SNAP taken. No cookie.
**When**: W1 … W14 are sent one after the other; then W1 is sent again with the body `{` (not JSON) and W13 with a body that is not multipart.
**Then**: All sixteen answers are 401 with `error` `unauthenticated`, a non-empty `message` and no `errors` key; no body carries a record, an id or a count. The two malformed requests answer 401 as well — never 400 or 415 — because the guard answers before the body is read. SNAP after equals SNAP before (no shoe, line, category, review, upload or count was created, changed or removed).
Traces: SC-1 (ADMIN-004 AC-1, AC-2, AC-3, AC-4, AC-9; ADMIN-015 AC-12)
Check: fetch_url

**Scenario**: TC-02 — a forged session cookie is a visitor's
**Given**: SNAP taken. The cookie is FORGED.
**When**: W1 … W14 are sent with `Cookie: session=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA`.
**Then**: Every answer is 401 `unauthenticated` with the error body of TC-01; SNAP is unchanged.
Traces: SC-1 (ADMIN-004 AC-1 … AC-4)
Check: fetch_url

**Scenario**: TC-03 — an expired session cookie is a visitor's
**Given**: STAFF is signed in (cookie `C`); `C` is then expired as the `EXPIRED` arrangement says (created 8 days ago, expired a day ago). SNAP taken.
**When**: W1 … W14 are sent with `C`. A second sign-in as STAFF is then made and W1 is sent with the new cookie.
**Then**: All fourteen answers with `C` are 401 `unauthenticated` and SNAP is unchanged. The new cookie's W1 answers 201 (the account was never the problem; the old session was) and its shoe is removed afterwards.
Traces: SC-1 (ADMIN-004 AC-1 … AC-4)
Check: fetch_url

**Scenario**: TC-04 — the public reads stay open to a visitor
**Given**: No cookie. FX-PIC exists (so an upload can be read).
**When**: `GET /v1/products`, `/v1/lines`, `/v1/categories`, `/v1/reviews`, `/v1/product-filters`, `/v1/shop` and `GET /v1/uploads/<name of a FX-PIC picture>` are sent.
**Then**: Every answer is 200; the list answers carry `items` (an array), `total`, `page` and `limit`; the picture answer's `Content-Type` is `image/png`. `GET /v1/products` has `total` 12 or more.
Traces: SC-1 (ADMIN-004 AC-8)
Check: fetch_url

**Scenario**: TC-05 — a customer session is refused on every write with 403 and changes nothing
**Given**: SNAP taken. CUSTOMER is signed in.
**When**: W1 … W14 are sent with the customer's cookie; then W1 is sent with the body `{` and W13 with a body that is not multipart.
**Then**: All sixteen answers are 403 with `error` `forbidden`, a non-empty `message`, no `errors` key and no record in the body (the malformed requests answer 403, not 400 or 415). SNAP is unchanged.
Traces: SC-2 (ADMIN-004 AC-7; ADMIN-015 AC-12)
Check: fetch_url

**Scenario**: TC-06 — a customer who reads the shop is served as a visitor
**Given**: ADMIN has made the draft shoe `tc-draft` (`{"name":"Tc Draft","price":1000,"status":"draft","pairs":{"38":3}}`). CUSTOMER is signed in.
**When**: With the customer's cookie: `GET /v1/products?limit=100`, `GET /v1/products/tc-draft`, `GET /v1/products?status=draft`, `GET /v1/products/samba-og-w-core-black`.
**Then**: The first list does not contain `tc-draft`; the second answers 404 `not_found`; the third has `total` 0; the fourth answers 200 and its body has no `pairs`, no `price_listed`, no `_source` and no `_invented` key.
Traces: SC-2 (ADMIN-004 AC-7; ADMIN-013 AC-1 … AC-3)
Check: fetch_url

**Scenario**: TC-07 — an admin and a staff member each do the whole shoe life with the same answers
**Given**: Run the steps twice, once as ADMIN and once as STAFF; the shoe is named "Tc Role A" and "Tc Role S".
**When**: (1) `POST /v1/products` `{"name":"Tc Role A","price":2500000,"status":"draft","pairs":{"38":3}}`. (2) `PUT /v1/products/<id>` `{"price":2600000}`. (3) `PATCH /v1/products/<id>/sizes` `{"size":"38","pairs":5,"seen_pairs":3}`. (4) `POST /v1/uploads` with a PNG (prefix + 1 KB). (5) `DELETE /v1/products/<id>`. (6) `POST` a line, a category and a review (`{"product":"samba-og-w-core-black","name":"Tc Role","rating":4}`) and `DELETE` each.
**Then**: Both roles get the same statuses: 201, 200, 200, 201, 204, then 201 ×3 and 204 ×3. (2) returns `price` 2600000; (3) returns `size` `"38"`, `pairs` 5 and `stock` 5; (4) returns a `url` starting `/api/v1/uploads/`; (5) has an empty body; the deleted shoe answers 404 afterwards.
Traces: SC-3 (ADMIN-004 AC-10; ADMIN-015 AC-13; ADMIN-010 item 1)
Check: fetch_url

**Scenario**: TC-08 — the collections that have no write route answer 404 to a dealer
**Given**: ADMIN, then STAFF. The bodies of `GET /v1/delivery`, `/v1/payments`, `/v1/shop`, `/v1/sizes` and `/v1/faqs` are saved first (dealer view).
**When**: For each of `delivery` (id `standard`), `payments` (`vnpay`), `shop` (`shop`), `sizes` (`eu-38`) and `faqs` (`f-delivery-time`): `POST /v1/<c>` `{"name":"Tc"}`, `PUT /v1/<c>/<id>` `{"name":"Tc"}` and `DELETE /v1/<c>/<id>`; then the five `GET`s again.
**Then**: All 15 write answers (per role) are 404 `not_found` with the error body; the five `GET` bodies are equal to the saved ones (200, nothing changed).
Traces: SC-3 (ADMIN-010 item 1; contract §3.22)
Check: fetch_url

## 6. Saving records (SC-4 … SC-10)

**Scenario**: TC-09 — creating a shoe answers 201 with its slug id and the dealer view; a visitor reads it at once; the same name gets -2, then -3
**Given**: STAFF is signed in; no shoe has an id starting `samba-og-new-colour`.
**When**: `POST /v1/products` `{"name":"Samba OG New Colour","price":3290000,"line":"samba","category":"women","status":"live"}`; then `GET /v1/products/samba-og-new-colour` with no cookie; then the same `POST` twice more.
**Then**: The first answer is 201 with header `Location: /v1/products/samba-og-new-colour` and a body with `id` `samba-og-new-colour`, `name` "Samba OG New Colour", `price` 3290000, `line` `samba`, `category` `women`, `status` `live`, `rating` 0, `review_count` 0, `sizes` `[]`, `stock` 0, and the dealer-only keys `pairs` (`{}`), `price_listed` and `_source`. The visitor's `GET` answers 200 with the same `id`, `name`, `price`, `line`, `category`, `status` and no `pairs`, `price_listed` or `_source`. The second `POST` answers 201 with `id` `samba-og-new-colour-2` and a matching `Location`; the third `samba-og-new-colour-3`.
Traces: SC-4 (ADMIN-010 AC-1, AC-2, AC-3, items 5, 9)
Check: fetch_url

**Scenario**: TC-10 — the id is the slug of the name: accents removed, `đ` as `d`, hyphens, never empty, never over 80 characters
**Given**: ADMIN is signed in; none of the ids below exist.
**When**: A shoe is created for each name (price 1000): "Đôi Giày Mới 38", "ĐEN Ấn Độ", "Samba/OG_2", "!!!", "???", and a name of 100 `a`.
**Then**: The ids are `doi-giay-moi-38`, `den-an-do`, `samba-og-2`, `item`, `item-2`, and for the 100-character name an id of 80 characters or fewer made only of `a` (and hyphens). Every id matches `^[a-z0-9][a-z0-9._-]*$` and every `Location` ends with its id.
Traces: SC-4 (ADMIN-010 item 5; contract §3.22 "Ids")
Check: fetch_url

**Scenario**: TC-11 — a line and a category get their name's slug; a review gets r- and eight hex characters
**Given**: STAFF is signed in; no line `tc-line-one`, no category `tc-category-one`.
**When**: `POST /v1/lines` `{"name":"Tc Line One"}` twice; `POST /v1/categories` `{"name":"Tc Category One"}`; `POST /v1/reviews` `{"product":"samba-og-w-core-black","name":"Tc Reviewer","rating":4}` twice. A visitor then reads each.
**Then**: The lines answer 201 with ids `tc-line-one` and `tc-line-one-2` and `Location: /v1/lines/<id>`; the category 201 with `tc-category-one`. Each review answers 201 with an `id` matching `^r-[0-9a-f]{8}$`, `Location: /v1/reviews/<id>`, `status` `live` and `date` equal to today's date in Vietnam (`SELECT (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date`); the two review ids differ. A visitor's `GET /v1/lines/tc-line-one`, `/v1/categories/tc-category-one` and `/v1/reviews/<id>` each answer 200 with the saved values.
Traces: SC-4 (ADMIN-010 item 5; contract §3.22 "Ids")
Check: fetch_url

**Scenario**: TC-12 — a missing required field is 422 naming it, and nothing is saved
**Given**: STAFF is signed in; SNAP taken.
**When**: `POST /v1/products` `{"price":1000}`; `{"name":"Tc NoPrice"}`; `{"name":"   ","price":1000}`. `POST /v1/lines` `{}`. `POST /v1/categories` `{}`. `POST /v1/reviews` `{"name":"Tc"}` and `{"product":"samba-og-w-core-black"}`.
**Then**: Every answer is 422 `invalid` with `errors` holding exactly the missing field: `name`; `price`; `name` (a name of spaces is empty after trimming); `name`; `name`; `product`; `name`. Each sentence is plain English. SNAP after equals SNAP before.
Traces: SC-5 (ADMIN-010 AC-4, item 3)
Check: fetch_url

**Scenario**: TC-13 — two faults at once name both fields
**Given**: STAFF is signed in; SNAP taken.
**When**: `POST /v1/products` `{"price":-1}`; `POST /v1/reviews` `{"rating":9}`.
**Then**: The first answers 422 with `errors` holding both `name` and `price`; the second 422 with `errors` holding `product`, `name` and `rating`. Each key has its own plain-English sentence. SNAP is unchanged.
Traces: SC-5 (ADMIN-010 AC-10)
Check: fetch_url

**Scenario**: TC-14 — a field the collection does not have, a derived one and a read-only one are 422 naming the field
**Given**: STAFF is signed in; SNAP taken.
**When**: `POST /v1/products` `{"name":"Tc X","price":1000,"colour_code":"x"}`; the same on `POST /v1/lines`, `/v1/categories` and `/v1/reviews` (valid body plus `colour_code`); `POST /v1/products` with each of `"rating":4.9`, `"review_count":3`, `"stock":10`, `"sizes_low":["38"]`, `"sizes_out":["40"]`, `"_source":"x"`, `"_invented":["x"]` alone; `POST /v1/products` with `colour_code`, `rating` and no name together; `PUT /v1/products/samba-og-w-core-black` `{"rating":1}`.
**Then**: Every answer is 422 `invalid` and `errors` names exactly the offending field (`colour_code`; `rating`; `review_count`; `stock`; `sizes_low`; `sizes_out`; `_source`; `_invented`); the combined request names `colour_code`, `rating` and `name`; the `PUT` names `rating`. None is 200 or 400. SNAP is unchanged.
Traces: SC-5 (ADMIN-010 AC-11, item 3; services §2.5 exception (a))
Check: fetch_url

**Scenario**: TC-15 — a body that is not a JSON object is 400
**Given**: STAFF is signed in; SNAP taken.
**When**: `POST /v1/products` and `PUT /v1/products/samba-og-w-core-black` are each sent with the bodies `[]`, `"text"`, `12`, `null`, `{` and an empty body.
**Then**: All twelve answers are 400 `bad_request` with a non-empty `message`; SNAP is unchanged.
Traces: SC-5 (ADMIN-010 item 3; contract §3.22 error table)
Check: fetch_url

**Scenario**: TC-16 — the price rules: below 0, not a number, a fraction, and 0
**Given**: STAFF is signed in; SNAP taken.
**When**: `POST /v1/products` `{"name":"Tc Price","price":P}` for P = `-1`, `"abc"`, `2.5`, `"3290000"` (a string), `true`, `[]`, and then `0`.
**Then**: The first six answer 422 with `errors.price` and no other key (for `-1` the sentence is exactly "Price cannot be below 0"); nothing is saved. `0` answers 201 and `GET /v1/products/tc-price` by ADMIN reads `price` 0.
Traces: SC-6 (ADMIN-010 AC-5, AC-6, AC-7)
Check: fetch_url

**Scenario**: TC-17 — closed sets, references, date format; an empty line or category unsets it; a new shoe is a draft unless told
**Given**: STAFF is signed in; SNAP taken; FX-SHOE exists (line `samba`, category `women`).
**When**: `POST /v1/products` with `status` `archived`; `colour` `red`; `line` `zzz`; `category` `zzz`; `released` `2026-13-45`, `26/11/2026`, `today` and `2026-02-30`; then valid ones: `colour` `""` and `black`, `released` `2026-11-26`. `POST /v1/reviews` `{"product":"zzz","name":"Tc"}`. `PUT /v1/products/tc-shoe` `{"line":"","category":""}`. `POST /v1/products` `{"name":"Tc Default","price":1}`.
**Then**: The refusals answer 422 naming, in turn, `status`, `colour`, `line`, `category`, `released` (×4) and `product`; nothing is saved. The valid ones answer 201. The `PUT` answers 200 with `line` `""` and `category` `""` (the shoe is still there). The last `POST` answers 201 with `status` `draft`.
Traces: SC-6 (ADMIN-010 AC-8, AC-9; `erd.md` Q2, Q6)
Check: fetch_url

**Scenario**: TC-18 — a review's fit, status and date, and a line's first-made year are checked
**Given**: STAFF is signed in; SNAP taken.
**When**: `POST /v1/reviews` with `{"product":"samba-og-w-core-black","name":"Tc"}` plus `fit` `huge`; `status` `archived`; `date` `26-11-2026`; then `fit` each of `""`, `small`, `true`, `large`, `status` `hidden`, and `date` `2026-11-20`. `POST /v1/lines` `{"name":"Tc Since","since":V}` for V = `0`, `-5`, `1.5`, `"abc"`, then `1950` and `null`.
**Then**: The first three review answers are 422 naming `fit`, `status` and `date`; the four fits, the hidden status and the date are 201. The line answers for `0`, `-5`, `1.5`, `"abc"` are 422 naming `since`; `1950` and `null` are 201 (`since` is 1950 and `null` on read).
Traces: SC-6 (ADMIN-010 item 2; contract §3.22 table)
Check: fetch_url

**Scenario**: TC-19 — a review's stars are a whole number from 1 to 5, or empty
**Given**: STAFF is signed in; a draft shoe `tc-box` made for this group (`{"name":"Tc Box","price":1000,"status":"draft"}`); SNAP taken.
**When**: `POST /v1/reviews` `{"product":"tc-box","name":"Tc","rating":R}` for R = `6`, `0`, `2.5`, `-1`, `"5"`, `true`, then `1`, `5`, `null`; and one with `rating` left out.
**Then**: The first six answer 422 with `errors.rating`; `1` and `5` answer 201 with that `rating`; `null` and the omitted field answer 201 with `rating` `null`.
Traces: SC-7 (ADMIN-010 AC-12)
Check: fetch_url

**Scenario**: TC-20 — the length limits are counted in characters, one either side of each
**Given**: STAFF is signed in; SNAP taken; `tc-box` exists.
**When**: Saves are made with text of 200 and 201 characters for `products.name`, `products.model`, `lines.name`, `categories.name`, `reviews.name`; with 5,000 and 5,001 for `products.description`, `lines.blurb`, `reviews.body`. The text is `a` repeated, and again for `name` the single letter `ế` (3 bytes, 1 character) repeated 200 and 201 times.
**Then**: Every 200 / 5,000 save answers 201 (and a read returns the text whole, 200 `ế` characters for the Vietnamese name); every 201 / 5,001 save answers 422 naming that field with a sentence that contains the limit (`200`, `5,000` or `5000`). Nothing is saved for a refusal (SNAP counts rise only by the accepted saves).
Traces: SC-7 (ADMIN-010 AC-13)
Check: fetch_url

**Scenario**: TC-21 — a PUT of the price alone changes the price and nothing else
**Given**: FX-SHOE exists; ADMIN has set `{"model":"Tc Model","colourway":"Tc Colourway","description":"Tc description","colour":"blue","is_new":true,"released":"2026-11-01"}` by `PUT`; the dealer record `R0` of `tc-shoe` is read.
**When**: `PUT /v1/products/tc-shoe` `{"price":3490000}`; then `R1` = the dealer `GET /v1/products/tc-shoe`; then a visitor reads `GET /v1/products/tc-shoe` and `GET /v1/products?q=Tc Shoe`.
**Then**: The `PUT` answers 200 with `price` 3490000. `R1` equals `R0` in every key except `price`: `id`, `name`, `model`, `line`, `category`, `status`, `colour`, `released`, the three picture addresses, `pairs` (`{"38":4,"39":2,"40":0}`), `sizes`, `sizes_low`, `sizes_out`, `stock`, `rating` and `review_count`. The visitor's single read and the list item show `price` 3490000 on the next read.
Traces: SC-8 (ADMIN-010 AC-14, item 4, item 9)
Check: fetch_url

**Scenario**: TC-22 — a different id in the body of a PUT is ignored
**Given**: FX-SHOE and FX-OTHER exist; ADMIN is signed in.
**When**: `PUT /v1/products/tc-shoe` `{"id":"tc-other","price":3500000}`; then `GET /v1/products/tc-shoe`, `/v1/products/tc-other`. The same for `PUT /v1/lines/tc-line-one` `{"id":"samba","blurb":"Tc blurb"}` and `PUT /v1/reviews/<a review id>` `{"id":"r-01","rating":3}`.
**Then**: The first answer is 200 with `id` `tc-shoe` and `price` 3500000; `tc-shoe` reads 3500000 and `tc-other` is unchanged (its `price` is still 2000000); the line answer has `id` `tc-line-one` and the seeded `samba` still has its original `blurb`; the review answer has its own id and `r-01` keeps its `rating` 5.
Traces: SC-8 (ADMIN-010 AC-15)
Check: fetch_url

**Scenario**: TC-23 — PUT and DELETE of an id that does not exist are 404 on every collection
**Given**: STAFF is signed in; SNAP taken.
**When**: `PUT` (body `{"name":"Tc"}`) and `DELETE` on `/v1/products/zzz-none`, `/v1/lines/zzz-none`, `/v1/categories/zzz-none` and `/v1/reviews/r-ffffffff`; then `PUT` and `DELETE` on `/v1/products/Bad_ID`, `/v1/products/a%2Fb` and `/v1/products/..%2Fx`; then `GET /v1/products/zzz-none`.
**Then**: All 14 write answers are 404 `not_found` with the error body (no `errors` key); the id was not created (`GET` 404). SNAP is unchanged.
Traces: SC-8 (ADMIN-010 AC-18; ADMIN-014 "Not found"; services §2.5 path segments)
Check: fetch_url

**Scenario**: TC-24 — a required field sent as null is 422; an optional one sent as null is cleared
**Given**: STAFF is signed in; FX-SHOE, the line `tc-line-one`, the category `tc-category-one` and a review `RV` (`{"product":"tc-shoe","name":"Tc Null","rating":3,"body":"x"}`) exist.
**When**: `PUT /v1/products/tc-shoe` `{"name":null}` and `{"price":null}`; `PUT /v1/lines/tc-line-one` `{"name":null}`; `PUT /v1/categories/tc-category-one` `{"name":null}`; `PUT /v1/reviews/RV` `{"product":null}` and `{"name":null}`. Then `PUT /v1/products/tc-shoe` `{"model":null,"description":null}`; `PUT /v1/reviews/RV` `{"rating":null,"body":null}`; `PUT /v1/lines/tc-line-one` `{"since":null}`.
**Then**: The first six answer 422 naming `name`, `price`, `name`, `name`, `product`, `name`, and the records keep their values. The last three answer 200: `model` `""` and `description` `""`; `rating` `null` and `body` `""`; `since` `null`.
Traces: SC-8 (ADMIN-010 item 4; contract §2.9 "PATCH for partial updates")
Check: fetch_url

**Scenario**: TC-25 — two PUTs of one shoe at the same moment both answer 200 and leave a whole record
**Given**: FX-SHOE exists; ADMIN is signed in in the browser.
**When**: In 10 rounds, one `interact_page` run sends at once `PUT /v1/products/tc-shoe` `{"price":1111111,"model":"Tc A","colourway":"Tc A"}` and `{"price":2222222,"model":"Tc B","colourway":"Tc B"}`, then reads the dealer record.
**Then**: All 20 answers are 200 (never 409, 422 or 5xx). After every round `(price, model, colourway)` is exactly `(1111111, "Tc A", "Tc A")` or `(2222222, "Tc B", "Tc B")` — never a mix — and `name`, `pairs` and `sizes` are as before. In at least one round each answer's body is a whole record that equals one of the two saves.
Traces: SC-9 (ADMIN-010 AC-19, item 7)
Check: interact_page

**Scenario**: TC-26 — saving a shoe with "Opens the home page" on turns it off on every other shoe
**Given**: ADMIN is signed in. `tc-feat-a` is made live with `featured` true (this turns off the seeded `samba-og-w-core-black`); `tc-feat-b` is made live with `featured` false.
**When**: `PUT /v1/products/tc-feat-b` `{"featured":true}`; then `GET /v1/products/tc-feat-a`, `/v1/products/tc-feat-b`, `/v1/products/samba-og-w-core-black` and `GET /v1/products?featured=true` with no cookie.
**Then**: The `PUT` answers 200 with `featured` true. `tc-feat-a` reads `featured` false, `tc-feat-b` true, the seeded one false; the list has exactly one item, `tc-feat-b`.
Traces: SC-10 (ADMIN-010 AC-16, item 6)
Check: fetch_url

**Scenario**: TC-27 — the home page shows the shoe that opens it
**Given**: The state at the end of TC-26 (`tc-feat-b` live and featured; it has FX-PIC pictures).
**When**: `/` is rendered at 1280×800.
**Then**: The link `[data-testid='home-lead']` has `href` `/shop/tc-feat-b`; the page text contains the name "Tc Feat B" (case-insensitive) and does not contain "Tc Feat A".
Traces: SC-10 (ADMIN-010 AC-16)
Check: render_url

**Scenario**: TC-28 — two saves that both turn "Opens the home page" on leave exactly one shoe with it
**Given**: Three live shoes `tc-f1`, `tc-f2`, `tc-f3`, none featured; ADMIN is signed in in the browser.
**When**: In 10 rounds one `interact_page` run sends at once `PUT /v1/products/tc-f1` `{"featured":true}` and `PUT /v1/products/tc-f2` `{"featured":true}` (then `tc-f3` with `tc-f1` in the next round, rotating), then reads `GET /v1/products?featured=true&limit=100` as a visitor.
**Then**: After each round the list holds exactly one shoe, one of the two just saved; no answer is 5xx; every answer is 200 (a loser of the race is retried by the shop, not refused).
Traces: SC-10 (ADMIN-010 AC-16; services §3.22 side effects)
Check: interact_page

## 7. What visitors see, and the ratings (SC-11 … SC-13)

**Scenario**: TC-29 — a draft shoe is absent from every public read and answers 404 like an id that does not exist
**Given**: The body of a visitor's `GET /v1/product-filters` is saved as `F0`. ADMIN then makes the draft shoe `tc-draft` (`{"name":"Tc Draft Unique","price":1000,"status":"draft","colour":"purple","pairs":{"47":4}}`) and the live shoe FX-SHOE.
**When**: With no cookie: `GET /v1/products?limit=100`; `GET /v1/products?q=Tc Draft Unique`; `GET /v1/products?size=47`; `GET /v1/product-filters`; `GET /v1/products/tc-draft`; `GET /v1/products/zzz-none`; `GET /v1/products?status=draft`.
**Then**: The first list does not contain `tc-draft` (and does contain `tc-shoe`); the search has `total` 0; the size filter `47` has `total` 0; the filters body equals `F0` and holds no `47` (a draft's size and colour are not offered); `GET /v1/products/tc-draft` answers 404 `not_found` with exactly the `error` and `message` of `GET /v1/products/zzz-none`; `?status=draft` answers 200 with `total` 0 and `items` `[]`.
Traces: SC-11 (ADMIN-013 AC-1, AC-2; ADMIN-010 AC-17; item 1)
Check: fetch_url

**Scenario**: TC-30 — a hidden review is absent from every public read and answers 404 by id
**Given**: FX-SHOE exists. ADMIN makes the review `HR` (`{"product":"tc-shoe","name":"Tc Hidden","rating":2,"status":"hidden"}`) and a live review `LR` on the same shoe.
**When**: With no cookie: `GET /v1/reviews?limit=100`; `GET /v1/reviews?product=tc-shoe`; `GET /v1/reviews/<HR>`; `GET /v1/reviews/r-ffffffff`; `GET /v1/reviews?status=hidden`.
**Then**: Neither list contains `HR` (the second contains `LR`); `GET /v1/reviews/<HR>` answers 404 with the `error` and `message` of the unknown id; `?status=hidden` has `total` 0.
Traces: SC-11 (ADMIN-013 AC-4)
Check: fetch_url

**Scenario**: TC-31 — the dealer sees drafts, hidden reviews and the bookkeeping; a customer is a visitor
**Given**: The state of TC-29 and TC-30 (`tc-draft`, `HR`). ADMIN, STAFF and CUSTOMER are signed in.
**When**: Each of the three sessions reads `GET /v1/products?limit=100`, `GET /v1/products/tc-draft`, `GET /v1/products?status=draft`, `GET /v1/products?size=47`, `GET /v1/reviews?product=tc-shoe`, `GET /v1/reviews/<HR>`.
**Then**: For ADMIN and STAFF every answer is 200 and holds `tc-draft` / `HR`; `GET /v1/products/tc-draft` carries `pairs` (`{"47":4}`), `price_listed` and `_source`, and so does each item of the list; `?status=draft` has `total` of 1 or more. For CUSTOMER the answers are exactly those of TC-29 and TC-30: no draft, no hidden review, 404 by id, no `pairs`, `price_listed` or `_source` on any shoe.
Traces: SC-11 (ADMIN-013 AC-3, item 1; services §2.4 views)
Check: fetch_url

**Scenario**: TC-32 — a shoe's rating is the average of its live reviews to one decimal
**Given**: FX-SHOE exists with no review (`rating` 0, `review_count` 0). STAFF is signed in.
**When**: `POST /v1/reviews` `{"product":"tc-shoe","name":"Tc Five","rating":5}` and `{"product":"tc-shoe","name":"Tc Four","rating":4}`; a visitor reads `GET /v1/products/tc-shoe`; then `POST /v1/reviews` `{"product":"tc-shoe","name":"Tc Four B","rating":4}` and the visitor reads it and `GET /v1/products?q=Tc Shoe` again.
**Then**: After two reviews `rating` is 4.5 and `review_count` 2; after the third `rating` is 4.3 (not 4.33) and `review_count` 3, in the single read and in the list item.
Traces: SC-12 (ADMIN-013 AC-5, AC-6, item 2)
Check: fetch_url

**Scenario**: TC-33 — hiding, re-starring, moving and deleting a review each recompute both figures of every shoe affected
**Given**: FX-SHOE has the live reviews `A` (5 stars), `B` (4) and `C` (4); FX-OTHER has none. STAFF is signed in.
**When**: After each step a visitor reads `GET /v1/products/tc-shoe` and `/v1/products/tc-other`. (a) `PUT /v1/reviews/C` `{"status":"hidden"}`. (b) `PUT /v1/reviews/B` `{"rating":1}`. (c) `PUT /v1/reviews/A` `{"product":"tc-other"}`. (d) `DELETE /v1/reviews/B`. (e) `PUT /v1/reviews/C` `{"status":"live"}`.
**Then**: (a) tc-shoe 4.5 / 2 (A and B). (b) tc-shoe 3 / 2 (5 and 1). (c) tc-shoe 1 / 1 (B only) and tc-other 5 / 1 — both read right after the `PUT`'s own answer, with no delay. (d) tc-shoe 0 / 0. (e) tc-shoe 4 / 1 (C). Every `PUT` and the `DELETE` answer 200 / 204.
Traces: SC-12 (ADMIN-013 AC-7, item 2; services §3.22 side effects)
Check: fetch_url

**Scenario**: TC-34 — saving the shoe recomputes its rating and count from its live reviews
**Given**: FX-SHOE has two live reviews (5 and 4). The stored figures are then spoiled with `set_db_state`: `UPDATE products SET rating = 1.0, review_count = 9 WHERE id = 'tc-shoe'`.
**When**: STAFF sends `PUT /v1/products/tc-shoe` `{"price":3300000}`; a visitor then reads the shoe.
**Then**: The `PUT` answers 200 with `rating` 4.5 and `review_count` 2, and the visitor's read agrees.
Traces: SC-12 (ADMIN-013 item 2: "and when its shoe is saved")
Check: fetch_url

**Scenario**: TC-35 — when the last live review goes, the shoe reads 0 and 0 and its page shows no rating line
**Given**: FX-SHOE has one live review (4 stars). `/shop/tc-shoe` is rendered first: its text contains "4 · 1 reviews" (rating, a middle dot, the count).
**When**: STAFF deletes that review (`DELETE /v1/reviews/<id>`); `GET /v1/products/tc-shoe` is read; `/shop/tc-shoe` is rendered again at 1280×800.
**Then**: `GET /v1/products/tc-shoe` has `rating` 0 and `review_count` 0. The page text has no match for `/\d(\.\d)?\s·\s\d+ reviews/`, and the product title area shows no star rating line.
Traces: SC-12 (ADMIN-013 AC-8)
Check: render_url

**Scenario**: TC-36 — the figures that came with the first design stay until a review event recalculates them (runs first of the run)
**Given**: The database as seeded; no review has been created, changed or deleted since the migration (this case runs before any other that does). The seeded `rating` of `samba-og-w-crystal-white` is recorded.
**When**: A visitor reads `GET /v1/shop`, `GET /v1/products/samba-og-w-core-black`, `/v1/products/samba-og-w-crystal-white`. STAFF then creates one review (`{"product":"samba-og-w-core-black","name":"Tc First Event","rating":4}`); the same three reads are repeated; the expected figures are computed with `set_db_state`: `SELECT round(avg(rating)::numeric, 1), count(*) FROM reviews r WHERE status = 'live' AND product_id = 'samba-og-w-core-black'` and the same over every live review of a live shoe.
**Then**: Before the event: the shop `rating` 4.8 and `review_count` 181; `samba-og-w-core-black` `rating` 4.8 and `review_count` 23. After it: the shop's figures equal the computed average and count over all live reviews of live shoes (`review_count` 37, no longer 181); `samba-og-w-core-black` reads the figures computed from its 4 live reviews; `samba-og-w-crystal-white` still reads its seeded `rating` (a review event recalculates only the shoes it touches).
Traces: SC-13 (ADMIN-013 AC-9, item 4; story §5 item 6)
Check: fetch_url

**Scenario**: TC-37 — the shop's average and count follow the live reviews of live shoes
**Given**: FX-SHOE has live reviews of 5 and 4 stars; FX-OTHER has a live review of 2 stars. STAFF is signed in.
**When**: A visitor reads `GET /v1/shop`; the expected figures are computed with `set_db_state`: `SELECT round(avg(r.rating)::numeric, 1) AS rating, count(*) AS n FROM reviews r JOIN products p ON p.id = r.product_id WHERE r.status = 'live' AND p.status = 'live'`.
**Then**: The shop's `rating` equals the computed `rating` and `review_count` equals `n` (both shoes' reviews are in them).
Traces: SC-13 (ADMIN-013 AC-9, item 3)
Check: fetch_url

**Scenario**: TC-38 — a shoe going to draft, being deleted or coming back changes the shop's figures; with no live review both are 0
**Given**: The state of TC-37 with the shop's figures `S0` read.
**When**: (a) `PUT /v1/products/tc-shoe` `{"status":"draft"}`; read `GET /v1/shop`. (b) `PUT` `{"status":"live"}`; read. (c) Delete the order-free `tc-other` (`DELETE /v1/products/tc-other`); read. (d) With `set_db_state`: `UPDATE reviews SET status = 'hidden'`; then STAFF creates and deletes one hidden review (`{"product":"tc-shoe","name":"Tc Zero","status":"hidden"}`) so the shop recomputes; read. The statuses are then put back from the recorded list.
**Then**: (a) the figures equal the computed ones without tc-shoe's two reviews (`review_count` lower by 2); (b) equal `S0` again; (c) lower by tc-other's one review; (d) `rating` 0 and `review_count` 0.
Traces: SC-13 (ADMIN-013 AC-10, items 3, 4; boundary "no live review exists anywhere")
Check: fetch_url

## 8. Deleting safely (SC-14 … SC-16)

**Scenario**: TC-39 — a shoe an order refers to cannot be deleted; setting it to draft works and the order still shows it
**Given**: ADMIN is signed in. Two shoes: the placeholder `samba-og-w-core-black` (sample order `doi-24810` refers to it; it has the reviews `r-01` … `r-03` and 9 count rows) and FX-SHOE after one order was placed for it (REQ with `tc-shoe`, size `38`, qty 1; the answer's order id and token are kept). SNAP taken.
**When**: `DELETE /v1/products/samba-og-w-core-black`; `DELETE /v1/products/tc-shoe`; then `PUT /v1/products/tc-shoe` `{"status":"draft"}`; then a visitor reads `GET /v1/products?limit=100` and `GET /v1/products/tc-shoe`; ADMIN reads `GET /v1/products/tc-shoe`; the order is read with `GET /v1/orders/<id>` and `X-Order-Token`.
**Then**: Both deletes answer 409 `conflict` with `message` "Orders refer to this shoe. Set it to draft instead." and no `errors` key; both shoes, `r-01` … `r-03` (still 200 for ADMIN), the 9 count rows of the placeholder (`SELECT count(*) FROM product_sizes WHERE product_id = 'samba-og-w-core-black'` is 9) and the order are untouched. The `PUT` answers 200 with `status` `draft`; the visitor's list lacks `tc-shoe` and its single read is 404; ADMIN's read is 200; the order's `items[0].name` is still "Tc Shoe".
Traces: SC-14 (ADMIN-014 AC-1, AC-2; ADMIN-015 AC-15)
Check: fetch_url

**Scenario**: TC-40 — a shoe no order refers to is deleted with its reviews and its counts
**Given**: ADMIN makes `tc-free` (`{"name":"Tc Free","price":1000,"status":"live","pairs":{"38":3,"39":2,"40":0}}`) and two reviews on it (5 and 3 stars, both live). The shop's `rating` and `review_count` are read as `S0`.
**When**: `DELETE /v1/products/tc-free`; then ADMIN reads `GET /v1/products/tc-free`, each of its two review ids, `GET /v1/shop`, and sends `DELETE /v1/products/tc-free` again; `SELECT count(*) FROM product_sizes WHERE product_id = 'tc-free'` and `SELECT count(*) FROM reviews WHERE product_id = 'tc-free'` are read.
**Then**: The delete answers 204 with an empty body. The shoe and both reviews answer 404; both counts are 0. The shop's `review_count` is `S0`'s minus 2 and `rating` equals the SQL average over the remaining live reviews of live shoes. The second `DELETE` answers 404 `not_found`.
Traces: SC-15 (ADMIN-014 AC-3, item 2; ADMIN-015 AC-15)
Check: fetch_url

**Scenario**: TC-41 — a placeholder shoe no order refers to can be deleted, and the shop's figures are recomputed
**Given**: `megaride-ag-xl-core-black` (no sample order refers to it; reviews `r-34`, `r-35`, `r-36`; its count rows) is backed up as open point 7 says. ADMIN is signed in. A review event has already made the shop's figures true (a review was created and deleted first).
**When**: `DELETE /v1/products/megaride-ag-xl-core-black`; `GET /v1/products/megaride-ag-xl-core-black` and `GET /v1/reviews/r-34`; `GET /v1/shop`; a second `DELETE`; then the rows are put back.
**Then**: 204; both reads 404; `/v1/shop` `review_count` is lower by the number of the shoe's live reviews (3) and `rating` equals the SQL average over the remaining live reviews of live shoes; the second delete is 404. After the restore the shoe is read again with its three reviews.
Traces: SC-15 (ADMIN-014 AC-3; ADMIN-015 AC-15; story §5 item 6)
Check: fetch_url

**Scenario**: TC-42 — a line or category some shoe uses cannot be deleted; once the shoes are moved it can; an unused one can
**Given**: ADMIN makes the line `tc-used-line` and three shoes on it, the category `tc-used-cat` and two shoes in it, and the unused line `tc-free-line` and category `tc-free-cat`. SNAP taken.
**When**: `DELETE /v1/lines/tc-used-line`; `DELETE /v1/categories/tc-used-cat`; `DELETE /v1/lines/samba`; each shoe is then moved (`PUT` `{"line":"gazelle"}` / `{"category":"men"}`), the two deletes are repeated; `DELETE /v1/lines/tc-free-line` and `DELETE /v1/categories/tc-free-cat`.
**Then**: The first answers 409 `conflict` with `message` "3 shoes use this. Move them first."; the second "2 shoes use this. Move them first."; the third "<n> shoes use this. Move them first." with `<n>` equal to `SELECT count(*) FROM products WHERE line_id = 'samba'` (drafts count); none has `errors`; nothing is deleted. After the moves both repeated deletes answer 204; the two unused ones answer 204; a later `GET` of each is 404.
Traces: SC-16 (ADMIN-014 AC-4, AC-5, AC-6, items 3, 4)
Check: fetch_url

**Scenario**: TC-43 — a review can be deleted whatever its status
**Given**: FX-SHOE has one live review `LV` and one hidden review `HD`. ADMIN is signed in.
**When**: `DELETE /v1/reviews/LV`; `DELETE /v1/reviews/HD`; both are read as ADMIN.
**Then**: Both deletes answer 204 with empty bodies; both reads answer 404; the shoe's `review_count` is 0.
Traces: SC-16 (ADMIN-014 AC-9, item 4)
Check: fetch_url

## 9. Pictures (SC-17 … SC-19)

**Scenario**: TC-44 — a JPEG, a PNG and a WebP of 1 MB are stored; the address serves the same bytes to a visitor
**Given**: ADMIN is signed in in the browser. Three files of 1,048,576 bytes each: the JPEG, PNG and WebP prefixes, then zeros.
**When**: One `interact_page` run posts each as the single part `file` to `/api/v1/uploads` and reads the answer; then fetches each returned address **without** the session (`credentials: "omit"`) and compares the bytes and headers.
**Then**: Each upload answers 201 with exactly `{"url":"/api/v1/uploads/<name>"}`, `<name>` being 32 lower-case hex characters then `.jpg`, `.png` or `.webp` in the order sent, and a `Location` header equal to the `url`. Each `GET` answers 200 with the same 1,048,576 bytes, `Content-Type` `image/jpeg`, `image/png`, `image/webp`, `X-Content-Type-Options: nosniff` and a `Cache-Control` whose `max-age` is at least 86,400.
Traces: SC-17 (ADMIN-012 AC-1, AC-2, items 1, 2)
Check: interact_page

**Scenario**: TC-45 — a GIF and a text file named .jpg are refused 415 by their content; a PNG named .txt is accepted
**Given**: ADMIN is signed in in the browser; `SELECT count(*) FROM uploads` is read as `U0`.
**When**: Three uploads: a file starting `GIF89a` named `x.gif` (declared type `image/gif`); the text `Hello, this is not a picture.` named `x.jpg` and declared `image/jpeg`; a real PNG prefix + zeros named `x.txt` declared `text/plain`.
**Then**: The first two answer 415 `unsupported_media_type` with `message` "Choose a JPEG, PNG or WebP picture." and no `errors` key; the third answers 201 with a `url` ending `.png`. `SELECT count(*) FROM uploads` is `U0 + 1` (nothing was stored for the two refusals).
Traces: SC-18 (ADMIN-012 AC-3, AC-4, item 3)
Check: interact_page

**Scenario**: TC-46 — an empty file is 422; a request that is not multipart, has no file part or has a second part is 400; nothing is stored
**Given**: ADMIN is signed in in the browser; `U0` read.
**When**: Four posts to `/api/v1/uploads`: a `file` part of 0 bytes named `x.png`; a JSON body (`Content-Type: application/json`, `{}`); a multipart body whose only part is named `other`; a multipart body with the parts `file` (a PNG) and `extra` (text).
**Then**: Answers 422 `invalid`, 400 `bad_request`, 400 `bad_request`, 400 `bad_request`, each with the error body; `SELECT count(*) FROM uploads` is still `U0`.
Traces: SC-18 (ADMIN-012 items 2, 3; contract §3.24)
Check: interact_page

**Scenario**: TC-47 — exactly 5,242,880 bytes is accepted; one byte more is 413 and stores nothing
**Given**: ADMIN is signed in in the browser; `U0` read. Files starting with the JPEG prefix of 5,242,880, 5,242,881 and 6,291,456 bytes.
**When**: Each is posted as the single part `file`; the accepted one is fetched back.
**Then**: The first answers 201 and its address serves 5,242,880 bytes; the other two answer 413 `payload_too_large` with `message` "That picture is too large (5 MB at most)." and no `errors` key. `SELECT count(*) FROM uploads` is `U0 + 1`.
Traces: SC-19 (ADMIN-012 AC-5; boundary)
Check: interact_page

**Scenario**: TC-48 — a path in the file name is thrown away; the shop's own name is used
**Given**: ADMIN is signed in in the browser; `U0` read.
**When**: A PNG (prefix + 1 KB) is posted with the part's file name `../../x.png`; then `GET /api/v1/uploads/..%2F..%2Fx.png` and `GET /api/v1/uploads/x.png` are sent; `SELECT name FROM uploads ORDER BY created_at DESC LIMIT 1` is read.
**Then**: The upload answers 201 with a `url` that matches `^/api/v1/uploads/[0-9a-f]{32}\.png$` and holds neither `..` nor `x.png`; the stored row's `name` is that 32-hex name; the two `GET`s answer 404; `SELECT count(*) FROM uploads` is `U0 + 1` (nothing else was written).
Traces: SC-19 (ADMIN-012 AC-6; contract §3.24 "uploaded file name is discarded")
Check: interact_page

## 10. The pairs in each size (SC-20 … SC-25)

**Scenario**: TC-49 — a shoe created with counts for three sizes reads the derived lists and total, and the size filter follows
**Given**: FX-PIC exists. STAFF is signed in.
**When**: `POST /v1/products` as FX-SHOE (price, three pictures, `status` `live`, `pairs` `{"38":4,"39":2,"40":0}`); the dealer reads `GET /v1/products/tc-shoe`; a visitor reads it; a visitor reads `GET /v1/products?size=40&limit=100`, `?size=38&limit=100` and `?size=39&limit=100`.
**Then**: The `POST` answers 201. The dealer view has `pairs` `{"38":4,"39":2,"40":0}`, `sizes` `["38","39","40"]`, `sizes_low` `["39"]`, `sizes_out` `["40"]`, `stock` 6. The visitor view has the same four lists and total and **no** `pairs` key. `?size=40` does not contain `tc-shoe`; `?size=38` and `?size=39` do (a size on its last pairs still counts).
Traces: SC-20 (ADMIN-015 AC-1, AC-2, item 6; ADMIN-012 AC-8)
Check: fetch_url

**Scenario**: TC-50 — the total is the sum of the counts; sizes read in growing order
**Given**: STAFF is signed in.
**When**: `POST /v1/products` `{"name":"Tc Sum","price":1000,"status":"live","pairs":{"38":10,"39":6,"40":2}}`; then `POST /v1/products` `{"name":"Tc Order","price":1000,"status":"live","pairs":{"40":1,"38":1,"36 2/3":1,"35 1/2":2,"36":1}}`; the dealer reads both.
**Then**: The first reads `stock` 18, `sizes_low` `["40"]`, `sizes_out` `[]`. The second reads `sizes` exactly `["35 1/2","36","36 2/3","38","40"]` and `stock` 6.
Traces: SC-20 (ADMIN-015 AC-7, item 2; services §2.8 "growing order")
Check: fetch_url

**Scenario**: TC-51 — changing one size to 0 puts it in the sold-out list and out of the size filter
**Given**: FX-SHOE exists (38 holds 4). STAFF is signed in.
**When**: `PATCH /v1/products/tc-shoe/sizes` `{"size":"38","pairs":0,"seen_pairs":4}`; then the dealer reads `GET /v1/products/tc-shoe`, a visitor reads `GET /v1/products?size=38&limit=100`; `SELECT pairs FROM product_sizes WHERE product_id = 'tc-shoe' AND size = '38'` is read.
**Then**: The `PATCH` answers 200 with exactly the keys `size` (`"38"`), `pairs` (0), `sizes` (`["38","39","40"]`), `sizes_low` (`["39"]`), `sizes_out` (`["38","40"]`), `stock` (2). The dealer view agrees; the size filter no longer contains `tc-shoe`; the stored count is 0.
Traces: SC-21 (ADMIN-015 AC-3, item 4)
Check: fetch_url

**Scenario**: TC-52 — a count of 2 is last pairs, a count of 3 is in neither list
**Given**: FX-SHOE with 38 at 0 (after TC-51, or set with `SHELF('tc-shoe','38',0)`). STAFF is signed in.
**When**: In turn: `PATCH` `{"size":"38","pairs":5,"seen_pairs":0}`; `PATCH` `{"size":"38","pairs":2,"seen_pairs":5}`; `PATCH` `{"size":"38","pairs":3,"seen_pairs":2}`; after each the answer is read and `GET /v1/products?size=38&limit=100` is read as a visitor.
**Then**: After 5: 38 in neither list (`sizes_low` `["39"]`, `sizes_out` `["40"]`), `stock` 11. After 2: `sizes_low` `["38","39"]`, `stock` 8. After 3: `sizes_low` `["39"]`, 38 in neither list, `stock` 9. Every answer is 200 and the size filter contains `tc-shoe` each time (5, 2 and 3 pairs are all available).
Traces: SC-21 (ADMIN-015 AC-4, AC-5)
Check: fetch_url

**Scenario**: TC-53 — a size the shoe does not list, a missing or bad number, and an undefined field on the size change
**Given**: FX-SHOE exists; STAFF is signed in; counts read as before.
**When**: `PATCH /v1/products/tc-shoe/sizes` with: `{"size":"99","pairs":1,"seen_pairs":0}`; `PATCH /v1/products/zzz-none/sizes` with a valid body; `{"size":"38","seen_pairs":4}`; `{"size":"38","pairs":4}`; `{"pairs":1,"seen_pairs":4}`; `pairs` `2.5`, `"abc"`, `-1`; `seen_pairs` `2.5` and `-1`; `{"size":"38","pairs":1,"seen_pairs":4,"note":"x"}`; the body `{`; the body `[]`.
**Then**: The size `99` and the unknown shoe answer 404 `not_found`; the missing `pairs`, missing `seen_pairs`, missing `size`, the three bad `pairs` and the two bad `seen_pairs` answer 422 `invalid` with `errors` naming `pairs`, `seen_pairs`, `size`, `pairs` ×3, `seen_pairs` ×2; the undefined field `note`, `{` and `[]` answer 400 `bad_request`. No count changed (`SELECT pairs FROM product_sizes WHERE product_id = 'tc-shoe'` is as before).
Traces: SC-21 (ADMIN-015 item 4; contract §3.23 error table)
Check: fetch_url

**Scenario**: TC-54 — a count of -1, 2.5 or "abc" is 422 naming that size's count, on POST and on PUT
**Given**: STAFF is signed in; FX-SHOE exists; its dealer record `R0` is read.
**When**: `POST /v1/products` `{"name":"Tc Bad","price":1000,"pairs":{"38":V}}` and `PUT /v1/products/tc-shoe` `{"pairs":{"38":V,"39":2,"40":0}}` (with the GUARD set) for V = `-1`, `2.5`, `"abc"`.
**Then**: All six answer 422 `invalid` with `errors` holding the key `pairs.38` (a plain-English sentence) and no other key; no shoe `tc-bad` exists; `tc-shoe` reads exactly `R0`.
Traces: SC-22 (ADMIN-015 AC-8; ADMIN-010 item 2)
Check: fetch_url

**Scenario**: TC-55 — a size added without a count is refused; 0 is accepted; unreadable, duplicate and disagreeing sizes are 422; typed forms are stored as written
**Given**: STAFF is signed in; FX-SHOE exists; `R0` read.
**When**: `POST /v1/products` with: `{"name":"Tc A","price":1,"sizes":["38","39"],"pairs":{"38":2}}`; `{"name":"Tc B","price":1,"sizes":["38"]}`; `{"name":"Tc C","price":1,"pairs":{"38":0}}`; `{"name":"Tc D","price":1,"pairs":{"abc":1}}`; `{"name":"Tc E","price":1,"pairs":{"36 2/3":1,"36  2/3":2}}`; `{"name":"Tc F","price":1,"pairs":{"<21 characters>":1}}`; `{"name":"Tc G","price":1,"sizes":["38","39"],"pairs":{"38":1,"40":2}}`; then the accepted forms `{"name":"Tc H","price":1,"pairs":{"38":1,"36   2/3":1,"35 1/2":1,"35.5":1}}`. `PUT /v1/products/tc-shoe` `{"sizes":["38","39","40","41"],"pairs":{"38":4,"39":2,"40":0}}` (GUARD set).
**Then**: A: 422 with `errors` key `pairs.39` and the sentence "Enter the pairs in this size". B: 422 with `pairs.38`. C: 201 with `sizes` `["38"]`, `sizes_out` `["38"]`, `stock` 0 (0 is a count). D: 422 naming `pairs.abc` or `sizes`. E: 422 naming `pairs.36 2/3` or `sizes` (a duplicate after whitespace is collapsed). F: 422 naming that size or `sizes`. G: 422 with `errors.sizes`. H: 201 and `sizes` hold `"36 2/3"` (whitespace collapsed), `"35 1/2"`, `"35.5"` and `"38"` — each as typed, once. The `PUT` adding `41` answers 422 with `errors` key `pairs.41`. After every 422 nothing changed: `tc-a` … `tc-g` do not exist and `tc-shoe` reads `R0`.
Traces: SC-22 (ADMIN-015 AC-8, item 3; story §5 item 3)
Check: fetch_url

**Scenario**: TC-56 — a count an order has moved is refused with 409 and the stored count stays
**Given**: FX-SHOE exists; 38 holds 4 and the page showed 4. An order for 2 pairs of size 38 is then placed (REQ, `tc-shoe`, `38`, qty 2); `SELECT pairs ...` reads 2. STAFF is signed in.
**When**: `PATCH /v1/products/tc-shoe/sizes` `{"size":"38","pairs":6,"seen_pairs":4}`; the count is read; then `PATCH` `{"size":"38","pairs":6,"seen_pairs":2}`.
**Then**: The first answers 409 `conflict` with `message` "Pairs were just taken by an order. Check the number and save again." and no `errors` key; the stored count is still 2 and the dealer view reads `pairs["38"]` 2. The second answers 200 with `pairs` 6 and `stock` 8 (6 + 2 + 0).
Traces: SC-23 (ADMIN-015 AC-9)
Check: fetch_url

**Scenario**: TC-57 — a whole-shoe PUT that carries counts read before an order is refused with 409 and stores nothing
**Given**: FX-SHOE exists with 38 at 4, 39 at 2, 40 at 0. The form's read is recorded as the GUARD (`{"38":4,"39":2,"40":0}`). An order for 2 pairs of 38 is then placed (REQ). STAFF is signed in.
**When**: `PUT /v1/products/tc-shoe` `{"name":"Tc Shoe Renamed","price":3900000,"pairs":{"38":4,"39":2,"40":0}}` with the GUARD of the old read; then the same `PUT` with the GUARD set to the current counts (`{"38":2,"39":2,"40":0}`).
**Then**: The first answers 409 and its body names size `38` (in `message`, or as an `errors` key `pairs.38`); the shoe is exactly as before — `name` "Tc Shoe", `price` 3290000, `pairs["38"]` 2 (the order's pairs are not given back) and `pairs["39"]` 2. The second answers 200 with the new name, price and `pairs` `{"38":4,"39":2,"40":0}`.
Traces: SC-23 (ADMIN-015 AC-9; story §5 item 2, §8 note 1)
Check: fetch_url

**Scenario**: TC-58 — an order and a count change arriving together never leave a count below 0 and never overwrite pairs an order took
**Given**: FX-SHOE exists. ADMIN is signed in in the browser. Each round starts with `SHELF('tc-shoe','38',1)` and no non-sample orders.
**When**: In 10 rounds one `interact_page` run sends at once `POST /v1/orders` (REQ, `tc-shoe`, `38`, qty 1, a fresh key) and `PATCH /v1/products/tc-shoe/sizes` `{"size":"38","pairs":5,"seen_pairs":1}`; then `SELECT pairs FROM product_sizes WHERE product_id = 'tc-shoe' AND size = '38'` and the number of orders are read; the orders are deleted after the round.
**Then**: Every round ends in exactly one of two states: the order answered 201, the `PATCH` answered 409 and the count is 0; or the `PATCH` answered 200, the order answered 201 and the count is 4. Never a count below 0, never a count of 5 with an order placed, never a 5xx. Both outcomes are seen or one of them is seen in all 10 rounds.
Traces: SC-23 (ADMIN-015 Concurrency; AC-9)
Check: interact_page

**Scenario**: TC-59 — a PUT that leaves a size out removes it and its count; a price-only PUT leaves every count and list alone
**Given**: FX-SHOE exists (38: 4, 39: 2, 40: 0). STAFF is signed in; the GUARD is the current counts.
**When**: `PUT /v1/products/tc-shoe` `{"sizes":["38","39"],"pairs":{"38":4,"39":2}}`; the dealer reads the shoe; `SELECT size, pairs FROM product_sizes WHERE product_id = 'tc-shoe' ORDER BY size`; then `PUT` `{"price":3300000}` and the dealer reads again.
**Then**: The first `PUT` answers 200; the shoe reads `pairs` `{"38":4,"39":2}`, `sizes` `["38","39"]`, `sizes_low` `["39"]`, `sizes_out` `[]`, `stock` 6; the table holds two rows (38 → 4, 39 → 2) and none for 40. After the price-only `PUT` the same lists, counts and total are unchanged and `price` is 3300000.
Traces: SC-24 (ADMIN-015 AC-10, AC-11, item 3)
Check: fetch_url

**Scenario**: TC-60 — an order for the last pair takes the count to 0; the dealer's read, the admin page after a reload and the shoe page agree
**Given**: STAFF made `tc-last` live with pictures and `pairs` `{"38":1,"39":5}`. STAFF is signed in in the browser.
**When**: In the browser an order is placed through checkout's own endpoint (`POST /api/v1/orders`, REQ with `tc-last`, `38`, qty 1); then the run reads `GET /api/v1/products/tc-last`, opens `/admin/products/tc-last`, opens the Stock tab, reloads the page and reads the field labelled "Pairs in EU 38"; then opens `/shop/tc-last`.
**Then**: The dealer read has `pairs["38"]` 0, `sizes_out` holding `"38"`, `stock` 5. After the reload the field "Pairs in EU 38" has the value `0` and "Pairs in EU 39" the value `5`. On `/shop/tc-last` the pill `[data-testid='size-38']` is disabled and its computed `text-decoration-line` is `line-through`; `[data-testid='size-39']` is enabled.
Traces: SC-25 (ADMIN-015 AC-6; owner acceptance)
Check: interact_page

**Scenario**: TC-61 — a count staff typed is what the next order takes from
**Given**: `tc-last` as TC-60 (39 holds 5). STAFF is signed in in the browser on `/admin/products/tc-last`, Stock tab.
**When**: The run types `3` in the field "Pairs in EU 39" and presses Tab (the count saves by itself); then an order for 2 pairs of size 39 is placed through `POST /api/v1/orders`; then the run reloads the admin page and opens `/shop/tc-last`.
**Then**: The order answers 201. `GET /api/v1/products/tc-last` as dealer reads `pairs["39"]` 1 and `sizes_low` holding `"39"`; after the reload the field "Pairs in EU 39" has the value `1` and the read-only text for "Last pairs in" contains `39`; on `/shop/tc-last` `[data-testid='size-39']` is enabled and holds the last-pairs dot (a child `span` with `aria-hidden="true"`).
Traces: SC-25 (ADMIN-015 AC-6; owner acceptance "a count staff typed is what the next order takes from")
Check: interact_page

## 11. The screens (SC-26 … SC-38)

Every browser case signs in as STAFF (typing `tc-staff@example.com` and its password on `/admin/sign-in`) at 1280×800 unless it names another size, and reads the page's own request log where it says "the request". The Add / Edit dialog is `[role=dialog]`; the delete question is `[role=alertdialog]`.

**Scenario**: TC-62 — staff add a shoe with its pictures and counts; it is in the list and survives a reload
**Given**: STAFF is signed in in the browser; `/admin/products` is open. The run has three real PNGs it made with a canvas (40×40 px).
**When**: The run presses "Add"; types the Name `Tc Browser Shoe` and Price `3290000`; chooses Line "Samba", For "Women" and Status "On sale"; chooses a picture for "Side view", "Front pair" and "Back pair" (each file handed to its file input); adds the sizes 38, 39 and 40 with the sizes control (label "Add to Sizes (EU)": type the size, press Enter) and types 4, 2 and 0 in "Pairs in EU 38", "Pairs in EU 39" and "Pairs in EU 40"; presses "Add"; then reloads the page and searches the list for `Tc Browser Shoe`.
**Then**: The dialog title before "Add" reads "Add to shoes". After it the dialog is gone and the list has a row for the shoe: its picture cell holds an `img` whose `naturalWidth` is above 0 and whose `src` starts `/api/v1/uploads/`; the row's name cell reads "Tc Browser Shoe"; its price reads `3.290.000 ₫` (the space may be a no-break space); its status "On sale"; its stock cell `6`. After the reload the same row is there. The request log holds one `POST /api/v1/products` whose body has `status` `live`, `pairs` `{"38":4,"39":2,"40":0}`, the three picture addresses starting `/api/v1/uploads/`, and no `id` key.
Traces: SC-26 (ADMIN-011 AC-1, AC-2; ADMIN-012 AC-8; ADMIN-015 AC-1; items 1, 6)
Check: interact_page

**Scenario**: TC-63 — the new shoe is in /shop with its pictures; its page shows 38 available, 39 last pairs, 40 sold out; the size filter follows
**Given**: The shoe of TC-62 exists (`tc-browser-shoe`), live.
**When**: `/shop` is rendered and the card `[data-testid='card-tc-browser-shoe']` is looked for (the run pages or searches for the name when the card is not on the first screen); `/shop/tc-browser-shoe` is rendered; on `/shop` the size button `[data-testid='filter-size-38']` is pressed, then pressed again and `[data-testid='filter-size-40']` pressed.
**Then**: The card exists and its `img` has `naturalWidth` above 0. On the shoe's page three gallery pictures have `naturalWidth` above 0; `[data-testid='size-38']` is enabled with no last-pairs dot; `[data-testid='size-39']` is enabled and holds a child `span[aria-hidden='true']` (the dot); `[data-testid='size-40']` is `disabled` and its computed `text-decoration-line` is `line-through`. With size 38 chosen the card is among the results; with only size 40 chosen it is not (`GET /api/v1/products?size=40&limit=100` as a visitor agrees).
Traces: SC-26 (ADMIN-011 AC-2; ADMIN-015 AC-1, AC-2, item 6)
Check: interact_page

**Scenario**: TC-64 — setting the shoe to Draft removes it from the shop, search and the home page; On sale brings it back
**Given**: `tc-browser-shoe` is live and opens the home page (`PUT /api/v1/products/tc-browser-shoe` `{"featured":true}` as ADMIN). STAFF is signed in in the browser on `/admin/products/tc-browser-shoe`.
**When**: The run sets Status to "Draft" and presses "Save changes"; then reads `/shop`, the header search for `Tc Browser Shoe` (type it, press Enter), `/` and `/shop/tc-browser-shoe` as a visitor (a fresh browser context with no cookie), and `/admin/products` (searching the name); then sets Status back to "On sale", saves, and reads `/shop` again.
**Then**: "Saved." shows after each save. As a visitor: `/shop` has no `[data-testid='card-tc-browser-shoe']`; the search results hold no card or link for it; the link `[data-testid='home-lead']` on `/` does not point to `/shop/tc-browser-shoe`; `/shop/tc-browser-shoe` shows the text "That pair is not in the shop any more.". The admin list still has the row with the status "Draft". After "On sale" the card is back on `/shop`.
Traces: SC-27 (ADMIN-011 AC-3; ADMIN-013 AC-11; ADMIN-010 AC-17)
Check: interact_page

**Scenario**: TC-65 — while a save takes 2 seconds the button reads "Saving…" and is disabled, and a second press sends nothing
**Given**: The shop is slow for 2 seconds on the create (interception holds `POST /api/v1/products`, or `CREATE TRIGGER tc_slow BEFORE INSERT ON products` with `pg_sleep(2)`). STAFF is on `/admin/products`.
**When**: The run opens "Add", types Name `Tc Slow` and Price `1000`, chooses Status "On sale", presses "Add" and, inside the first second, presses it twice more and presses Enter in the Price field; the button text and `disabled` property are read at 200 ms and at 1 s and the request log after the answer. Then on `/admin/products/tc-slow` the Price is changed and "Save changes" pressed with `BEFORE UPDATE` slowed the same way.
**Then**: At 200 ms and at 1 s the dialog's confirm button reads "Saving…" and its `disabled` property is true; the "Saving…" text sits inside an element with `role="status"` or an `aria-live` region. The request log holds exactly one `POST /api/v1/products`; `SELECT count(*) FROM products WHERE name = 'Tc Slow'` reads 1; the dialog closes after the answer. On the record page "Save changes" reads "Saving…" and is disabled for the same time, then "Saved." shows.
Traces: SC-28 (ADMIN-011 AC-4; item 5)
Check: interact_page

**Scenario**: TC-66 — the list is drawn from the shop's answer: a trimmed name and the id the shop made show as stored
**Given**: STAFF is on `/admin/products`; no shoe named `Tc Trimmed`.
**When**: The run adds a shoe named `  Tc Trimmed  ` (two spaces either side), price 1000, then adds another with the same text; it reads the two rows' name cells and their link addresses (`href` of the row or its name link).
**Then**: The first row's name reads exactly `Tc Trimmed` (no surrounding spaces) and its address ends `/admin/products/tc-trimmed`; the second row's ends `/admin/products/tc-trimmed-2` — the ids the shop made, not an id made in the browser. Neither `POST` body carries an `id` key.
Traces: SC-28 (ADMIN-011 item 5; story §3 item 9, §5 item 4)
Check: interact_page

**Scenario**: TC-67 — refused fields keep the dialog open with every typed value and the sentence beside the field
**Given**: STAFF is on `/admin/products` with the Add dialog open.
**When**: The run types Name `Tc Faulty`, Price `-1`, Description `kept text`, and presses "Add"; then clears the Name and presses "Add" again; then the same in the Edit dialog of `tc-shoe` (Price `-1`, "Save changes").
**Then**: After the first press the dialog is still open; Name still reads `Tc Faulty`, Price `-1`, Description `kept text`; beside Price the sentence "Price cannot be below 0" shows, Price has `aria-invalid="true"`, and the sentence is inside a `role="alert"` or live region. After the second press both Name and Price have their own non-empty sentence at the same time (two sentences) and Description is still `kept text`. In the Edit dialog the same sentence shows beside Price and the other fields keep their values. `SELECT count(*) FROM products WHERE name = 'Tc Faulty'` is 0 and `tc-shoe` reads its old price.
Traces: SC-29 (ADMIN-011 AC-5, item 6)
Check: interact_page

**Scenario**: TC-68 — a refused count shows beside that size's count; the record page keeps typed values on a refusal
**Given**: STAFF is on `/admin/products` with the Add dialog open; then on `/admin/products/tc-shoe`.
**When**: In the dialog the run types Name `Tc Count`, Price `1000`, adds size 38 and types `-1` in "Pairs in EU 38", presses "Add". On the record page it types Price `-1` on the Basics tab and presses "Save changes".
**Then**: The dialog stays open; the sentence beside "Pairs in EU 38" is non-empty plain English, the field has `aria-invalid="true"`, and Name, Price and the size 38 are still there. On the record page the page stays on the shoe, the Price field still reads `-1` and the sentence "Price cannot be below 0" shows beside it, announced; no "Saved." shows.
Traces: SC-29 (ADMIN-011 AC-5; ADMIN-015 AC-8)
Check: interact_page

**Scenario**: TC-69 — with the shop unreachable the dialog and the record page stay open with the typed values and say so
**Given**: STAFF is on `/admin/products` with the Add dialog filled (Name `Tc Down`, Price `1000`, Status "On sale"). Then the backend is stopped (`set_backend_state`; the gateway answers 502).
**When**: The run presses "Add"; reads the dialog; the backend is started; the run presses "Add" again. Then on `/admin/products/tc-shoe` with the backend stopped again it changes Price and presses "Save changes".
**Then**: After the first press the dialog is open with Name `Tc Down` and Price `1000` kept, the text "We could not save. Please try again." is shown inside a `role="alert"` or live region, and the confirm button reads "Add" and is enabled (not "Saving…"). After the backend is back the second press closes the dialog and the shoe is in the list. On the record page the same message shows with the typed Price kept and "Save changes" enabled; no "Saved." shows.
Traces: SC-30 (ADMIN-011 AC-6, item 7)
Check: interact_page

**Scenario**: TC-70 — when the session has ended the save says so, links to sign-in and keeps the values
**Given**: STAFF is on `/admin/products` with the Add dialog filled (Name `Tc Ended`, Price `1000`). STAFF's session row is then deleted (`DELETE FROM sessions WHERE user_id = (SELECT id FROM users WHERE email = 'tc-staff@example.com')`).
**When**: The run presses "Add"; reads the dialog; presses the link; then, on a second sign-in, repeats on `/admin/products/tc-shoe` with "Save changes" after the session is deleted again.
**Then**: The dialog stays open with Name and Price kept and shows "Your session ended. Sign in again." (announced) with a link whose `href` is `/admin/sign-in`; "We could not save. Please try again." is not shown. Following the link lands on `/admin/sign-in`. On the record page the same sentence and link show and the typed Price stays. `SELECT count(*) FROM products WHERE name = 'Tc Ended'` is 0.
Traces: SC-30 (ADMIN-011 "Not permitted" row; item 7)
Check: interact_page

**Scenario**: TC-71 — changing the price on a shoe's page shows "Saved."; the list shows it after a reload; the fields are those the shop answered
**Given**: STAFF is on `/admin/products/tc-shoe`.
**When**: The run sets Name to `  Tc Shoe  ` (spaces either side) and Price to `3490000`, presses "Save changes"; reads the page; opens `/admin/products` after a reload and searches `Tc Shoe`.
**Then**: "Saved." shows (announced). The Name field now reads `Tc Shoe` (the shop's trimmed answer, not what was typed). The list row shows `3.490.000 ₫`. `GET /api/v1/products/tc-shoe` as visitor reads `price` 3490000.
Traces: SC-31 (ADMIN-011 AC-7, item 4)
Check: interact_page

**Scenario**: TC-72 — the delete question names the record; Cancel sends nothing; confirming a review removes it for good
**Given**: A review `RV` by "Tc Reviewer" on `tc-shoe` (API). STAFF is on `/admin/reviews`; the list is searched for `Tc Reviewer`.
**When**: The run opens the row menu, chooses "Delete", reads the dialog, presses "Cancel", reads the request log and the list; opens the question again and presses "Delete"; then reloads the page.
**Then**: The `[role=alertdialog]` reads "Delete Tc Reviewer?" with the buttons "Cancel" and "Delete". After "Cancel" the dialog is gone, the request log holds no `DELETE` and the row is still listed. After "Delete" the request log holds one `DELETE /api/v1/reviews/<id>` answered 204, the row is gone from the list and, after the reload, from `/admin/reviews` and from `GET /api/v1/reviews/<id>` (404).
Traces: SC-32 (ADMIN-011 AC-8, AC-9, item 8)
Check: interact_page

**Scenario**: TC-73 — a refused delete leaves the record and shows the shop's sentence inside the open dialog
**Given**: STAFF is on `/admin/products`; a sample order refers to `samba-og-w-core-black`; the line `tc-used-line` has two shoes; STAFF is also on `/admin/lines` later.
**When**: The run chooses "Delete" on the row of "Samba OG Black Crystal" and presses "Delete" in the question; then on `/admin/lines` chooses "Delete" on the row "Tc Used Line" and confirms.
**Then**: After the first confirm the question dialog is **still open** and shows "Orders refer to this shoe. Set it to draft instead." inside a `role="alert"` or live region; the shoe is still in the list after the dialog is cancelled and the page reloaded. After the second the dialog stays open and shows "2 shoes use this. Move them first."; the line is still listed.
Traces: SC-32 (ADMIN-011 item 8; ADMIN-014 AC-1, AC-4, item 6)
Check: interact_page

**Scenario**: TC-74 — an id that is not here says so only after the shop answered, never while the record is fetched
**Given**: STAFF is signed in. The answer to `GET /api/v1/products/tc-shoe` is held for 2 seconds (interception; with no interception this half is UNVERIFIED for that reason).
**When**: The run opens `/admin/products/nope` and reads the page; then opens `/admin/products/tc-shoe` and reads the page 1 second after the open and again 3 seconds after it.
**Then**: `/admin/products/nope` shows "This record is not here", "It may have been deleted since the link was made." and a button "Back to the list" (pressing it goes to `/admin/products`). On the held record the text "This record is not here" is absent at 1 second and at 3 seconds; the shoe's own fields (Name `Tc Shoe`) are there at 3 seconds.
Traces: SC-33 (ADMIN-011 AC-10, AC-11, item 9)
Check: interact_page

**Scenario**: TC-75 — the Shoes list has its columns, placeholder and filters, and shows drafts to the dealer
**Given**: `tc-draft` (draft) and the seeded shoes exist. STAFF is on `/admin/products` at 1280×800.
**When**: The run reads the title, the search box, the filters and the table; searches `Tc Draft Unique`; sets the Status filter to "Draft", then to "On sale".
**Then**: `main h1` reads "Shoes"; a button reads "Add"; the search field's placeholder is "Search by name or style code"; the filters are labelled "Status" (choices "On sale", "Draft") and "For" (the categories "Women", "Men", "Kids"). The table has seven columns in the order picture, name, line, category, price, status, stock: for the row "Samba OG Black Crystal" the first cell holds an `img`, then `Samba OG Black Crystal`, `Samba`, `Women`, `3.290.000 ₫`, `On sale`, and a stock number equal to `stock` in `GET /api/v1/products/samba-og-w-core-black` (dealer). The search shows the row of `tc-draft` with the status "Draft"; the Draft filter keeps it and the On sale filter drops it.
Traces: SC-34 (ADMIN-011 AC-13, items 2, 3; ADMIN-013 AC-3)
Check: interact_page

**Scenario**: TC-76 — the Lines, Categories and Reviews lists keep their words, and the Reviews filter reaches hidden reviews
**Given**: A hidden review `HR` by "Tc Hidden". STAFF is signed in.
**When**: The run opens `/admin/lines`, `/admin/categories`, `/admin/reviews` and reads each title and search placeholder; on `/admin/reviews` sets the Status filter to "Hidden", then "Shown".
**Then**: Titles "Lines", "Categories", "Reviews"; placeholders "Search lines", "Search categories", "Search reviews". The Status filter on Reviews offers "Shown" and "Hidden"; with "Hidden" the row of "Tc Hidden" is listed; with "Shown" it is not.
Traces: SC-34 (ADMIN-011 item 2, AC-13)
Check: interact_page

**Scenario**: TC-77 — search, filters, order and page are in the address; no match and no records have their own messages
**Given**: STAFF is on `/admin/products` with at least 14 shoes (the 12 seeded and FX shoes) and page size 5 chosen in the list's own size control.
**When**: The run types `Samba` in the search box, sets Status "On sale", sorts a column, goes to page 2, and copies `location.href`; then opens that address in a fresh page and reads the box, filter, sort indicator (`aria-sort`) and the rows' ids. Then it types `zzzqqq` in the search box and presses "Clear filters". Then `/admin/reviews` is opened with the reviews moved aside through `set_db_state` (`CREATE TABLE tc_reviews_bak AS TABLE reviews; DELETE FROM reviews;`, put back afterwards).
**Then**: The fresh page shows the same search text, the same filter value, the same sorted column and the same row ids as before. With `zzzqqq` the page shows "Nothing matches those filters" and a button "Clear filters"; pressing it clears the box and the full list returns. With no reviews at all `/admin/reviews` shows "Nothing here yet" and a button "Add".
Traces: SC-34 (ADMIN-011 AC-16, AC-17, AC-18, item 11)
Check: interact_page

**Scenario**: TC-78 — the picture control uploads the file, shows it is working, makes Save wait, and no data: text is ever saved
**Given**: The shop is slow for 2 seconds on `POST /api/v1/uploads` (interception, or `CREATE TRIGGER tc_slow BEFORE INSERT ON uploads` with `pg_sleep(2)`). STAFF has the Add dialog of Shoes open with Name `Tc Pics` and Price `1000`; and later the Add dialog of Categories.
**When**: The run chooses a real PNG for "Side view"; reads the control at 300 ms and after the answer; presses "Add" at 500 ms; then repeats the picture on the Add dialog of Categories (field "Picture") with Name `Tc Cat Pic` and presses "Add".
**Then**: At 300 ms the control shows a working state (an element inside it with `aria-busy="true"` or `role="status"`, or visible text that is not "Choose a picture" nor "Replace", changing again when the answer arrives). The request log shows one `POST /api/v1/uploads` (multipart) and no request body anywhere contains `data:image`; the `POST /api/v1/products` is not sent before the upload's answer (or the "Add" button was disabled until then). After the answer the preview `img` has `src` starting `/api/v1/uploads/` and `naturalWidth` above 0, and the button reads "Replace" (and "Remove" is offered). The saved shoe's `image` (dealer `GET`) and the saved category's `image` start `/api/v1/uploads/`.
Traces: SC-35 (ADMIN-012 items 1, 4; AC-9)
Check: interact_page

**Scenario**: TC-79 — a GIF, a 6 MB file and an unreachable shop are refused beside the control with the other values kept
**Given**: STAFF has the Add dialog of Shoes open with Name `Tc Refuse` and Price `1000`.
**When**: The run chooses (a) a real GIF; (b) a file starting with the PNG prefix of 6,291,456 bytes; then (c) with the backend stopped, a real PNG; then starts the backend and chooses the PNG again.
**Then**: (a) "Choose a JPEG, PNG or WebP picture." shows beside the control (the control's `aria-describedby` element, inside a live region). (b) "That picture is too large (5 MB at most)." shows there. (c) "We could not upload that picture. Please try again." shows there. In all three Name still reads `Tc Refuse`, Price `1000`, no preview is drawn and the dialog is open; after the backend returns the PNG uploads and the preview is drawn. `SELECT count(*) FROM uploads` rose only for the accepted file.
Traces: SC-35 (ADMIN-012 AC-10, item 3; "Upstream failure")
Check: interact_page

**Scenario**: TC-80 — the shoe form has a labelled count for each size; the three lists are read-only and redrawn from the shop's answer
**Given**: STAFF is on `/admin/products/tc-shoe` (counts 38: 4, 39: 2, 40: 0), Stock tab.
**When**: The run reads the Stock tab; types `7` in "Pairs in EU 38" and presses Tab; reads the three lists; then opens the Add dialog of Shoes and reads its stock section after adding sizes 38 and 39.
**Then**: The tab holds the fields labelled "Pairs in EU 38", "Pairs in EU 39" and "Pairs in EU 40" with the values 4, 2 and 0, each a whole-number input (an `input` with `inputmode="numeric"` or `type="number"` with `step="1"` and `min="0"`). "Last pairs in" reads `39`, "Sold out in" reads `40` and "Pairs in stock" reads `6`, and none of the three is an editable control (no enabled `input` or `textarea` carries those labels). After the change of 38 they read "Last pairs in 39", "Sold out in 40" and "Pairs in stock 9", redrawn from the shop's answer. In the Add dialog each added size shows its own "Pairs in EU <size>" field and the three lists are read-only there too.
Traces: SC-36 (ADMIN-015 items 1, 2, 3; AC-7)
Check: interact_page

**Scenario**: TC-81 — adding a size asks for its count; taking a size off removes its count
**Given**: STAFF is on `/admin/products/tc-shoe`, Stock tab (38: 7, 39: 2, 40: 0 after TC-80, or as set).
**When**: The run adds size `41` with the sizes control, leaves "Pairs in EU 41" empty and presses "Save changes"; then types `0` and presses "Save changes"; then takes size 39 off with the sizes control's own remove button and presses "Save changes".
**Then**: The first save is refused: a non-empty sentence ("Enter the pairs in this size") shows beside "Pairs in EU 41" and the size is not saved (`SELECT count(*) FROM product_sizes WHERE product_id = 'tc-shoe' AND size = '41'` is 0). After `0` the save answers "Saved." and "Sold out in" now reads `40, 41` (or lists both). After removing 39 and saving, the field "Pairs in EU 39" is gone and `SELECT count(*) FROM product_sizes WHERE product_id = 'tc-shoe' AND size = '39'` is 0; the counts of 38, 40 and 41 are as before. The request of the third save carries no `pairs` entry for a size the person did not change other than as the design's guard needs.
Traces: SC-36 (ADMIN-015 item 3, AC-10; ADMIN-015 AC-8)
Check: interact_page

**Scenario**: TC-82 — one size's count saves by itself on blur or Enter, and an unchanged number sends nothing
**Given**: STAFF is on `/admin/products/tc-shoe`, Stock tab; the shop is slow for about 1 second on the count change (`CREATE TRIGGER tc_slow BEFORE UPDATE ON product_sizes` with `pg_sleep(1)`, or interception on `PATCH`).
**When**: The run types `9` in "Pairs in EU 38" and presses Tab; reads that size's row at 300 ms and after the answer; types `4` in "Pairs in EU 39" and presses Enter; then focuses "Pairs in EU 40" and leaves it without changing the number.
**Then**: Each change sends exactly one `PATCH /api/v1/products/tc-shoe/sizes` whose body is `{"size":"38","pairs":9,"seen_pairs":<the count the page showed>}` (then 39). At 300 ms the row says "Saving…" (announced); afterwards "Saved." shows on that row and "Pairs in stock" and the two lists are redrawn from the answer (stock up by the difference). Focusing and leaving the unchanged 40 sends no request. `SELECT pairs FROM product_sizes` reads 9 and 4.
Traces: SC-37 (ADMIN-015 item 4, AC-3, AC-14)
Check: interact_page

**Scenario**: TC-83 — with the shop down the count row says it could not save and keeps the typed number
**Given**: STAFF is on `/admin/products/tc-shoe`, Stock tab. The backend is stopped.
**When**: The run types `9` in "Pairs in EU 39" and presses Tab.
**Then**: The row shows "We could not save. Please try again." (announced) and the field still holds `9`; no "Saved." shows.
Traces: SC-37 (ADMIN-015 AC-14)
Check: interact_page

**Scenario**: TC-84 — when an order took pairs since the page loaded, the row shows the current count and the 409 sentence and nothing is overwritten
**Given**: STAFF is on `/admin/products/tc-shoe`, Stock tab, where "Pairs in EU 38" shows 4. An order for 2 pairs of size 38 is then placed through `POST /api/v1/orders` (the page is not reloaded).
**When**: The run types `6` in "Pairs in EU 38" and presses Tab.
**Then**: The row shows "Pairs were just taken by an order. Check the number and save again." (announced) and the field now shows `2` (the current count); `SELECT pairs FROM product_sizes WHERE product_id = 'tc-shoe' AND size = '38'` reads 2. Typing `6` again and pressing Tab saves it ("Saved.") because the page now knows 2.
Traces: SC-37 (ADMIN-015 AC-9)
Check: interact_page

**Scenario**: TC-85 — on a phone the lists and the Add / Edit dialog fit the screen
**Given**: STAFF is signed in; `tc-shoe` exists.
**When**: At 390×844 and at 320×568 the run opens `/admin/products`, `/admin/lines`, `/admin/categories` and `/admin/reviews`, reads `document.documentElement.scrollWidth` and `clientWidth` and whether any sidebar link is visible; opens the Add dialog of Shoes and of Lines, reads the dialog's box and the box of "Cancel" and "Add" (scrolling the dialog's own container when it scrolls); presses "Cancel".
**Then**: On every list `scrollWidth` is not greater than `clientWidth` (a table scrolls inside its own container if wider); no sidebar link is visible and the header's menu button opens a sheet with the links; each dialog's height is not greater than the viewport's and "Cancel" and "Add" can be reached and pressed without the page scrolling sideways; each field in the dialog has a visible `label`.
Traces: SC-38 (Responsive NFR; ADMIN-011 item 1)
Check: interact_page

**Scenario**: TC-86 — on a phone the shoe page and its Stock tab fit; every count has a visible label; the states are announced
**Given**: STAFF is signed in; `tc-shoe` has the sizes 38, 39, 40.
**When**: At 390×844 and at 320×568 the run opens `/admin/products/tc-shoe`, reads `scrollWidth` and `clientWidth`, opens the Stock tab, reads each count field's box and label, types a number in one count and presses Tab (with the shop slow for 1 second), and reads which element holds "Saving…" and "Saved.".
**Then**: `scrollWidth` is not greater than `clientWidth` on the page and on the Stock tab; each "Pairs in EU <size>" field has a visible label, is at least 40 px high and fully inside the viewport width (a thumb can reach it); "Saving…" and "Saved." each sit inside an element with `role="status"`, `role="alert"` or an `aria-live` region. (The other states — field errors, the refusals, the unreachable and session-ended sentences — are asserted as announced in TC-65, TC-67, TC-69, TC-70, TC-73, TC-79 and TC-84.)
Traces: SC-38 (Responsive and Accessibility NFR)
Check: interact_page

**Scenario**: TC-87 — nothing the browser sends for a save carries an id it made or a data: picture
**Given**: STAFF is on `/admin/products`; the request log is on.
**When**: The run adds a shoe, a line, a category (with a picture) and a review through their Add dialogs, edits each once, and deletes the review; the request log is read.
**Then**: No `POST` body has an `id` key; every `PUT` is sent to `/api/v1/<collection>/<id>` with the shop's id in the address and carries only fields of the collection; no request body in the whole run contains `data:`; every request is sent with the session cookie (`credentials: "include"`: the requests are answered 201, 200 and 204, not 401); none of the writes is sent twice.
Traces: SC-28 (story §3 item 9); SC-35 (ADMIN-012 item 1)
Check: interact_page

## 12. Speed (SC-39)

**Scenario**: TC-88 — 100 saves and 100 deletes each answer within 1 second at p95
**Given**: ADMIN is signed in in the browser against the seeded database.
**When**: One `interact_page` run sends, one after another, 100 `POST /api/v1/products` (`{"name":"Tc Perf <n>","price":1000,"status":"live","pairs":{"38":3,"39":2,"40":0}}`) timing each with `performance.now()`, then 100 `DELETE` of those shoes, timing each.
**Then**: All 200 answers are 201 and 204; the 95th percentile of the 100 save times and of the 100 delete times is at most 1,000 ms.
Traces: SC-39 (Performance NFR; item 8)
Check: interact_page

## 13. Contract cases (the endpoints `services.md` already defines)

**Scenario**: C-1 — every refusal of a write has the shared error body and an X-Request-Id
**Given**: ADMIN is signed in.
**When**: Seven writes that fail: `POST /v1/products` `{}` (422); `PUT /v1/products/zzz-none` `{"price":1}` (404); `DELETE /v1/products/samba-og-w-core-black` (409); `PATCH /v1/products/tc-shoe/sizes` `{"size":"38","pairs":1,"seen_pairs":99}` (409); `POST /v1/uploads` with a GIF (415); `POST /v1/products` with `{` (400); `POST /v1/products` with no cookie (401). The request `X-Request-Id: tc-check-0001` is sent on the first.
**Then**: Each body is JSON with `Content-Type` starting `application/json`, `error` equal to `invalid`, `not_found`, `conflict`, `conflict`, `unsupported_media_type`, `bad_request`, `unauthenticated`, a non-empty `message`; `errors` only on the 422 and the 400; no SQL, path, stack or hash in any body; every answer has an `X-Request-Id` and the first echoes `tc-check-0001`.
Traces: contract (services §2.3; §3.22, §3.23, §3.24)
Check: fetch_url

**Scenario**: C-2 — a write that is not JSON is 415
**Given**: STAFF is signed in.
**When**: `POST /v1/products` and `PUT /v1/products/tc-shoe` are sent with `Content-Type: text/plain` and the body `{"name":"x","price":1}`; and `POST /v1/lines` with `Content-Type: application/x-www-form-urlencoded` and the body `name=x`.
**Then**: All three answer 415 `unsupported_media_type` with the error body; nothing is created or changed.
Traces: contract (services §3.22 error table, §2.3)
Check: fetch_url

**Scenario**: C-3 — a JSON body over its cap is 413
**Given**: STAFF is signed in; SNAP taken.
**When**: `POST /v1/products` is sent with a valid JSON object whose `description` is 70,000 characters (the body is over 64 KiB).
**Then**: 413 `payload_too_large` with the error body and no `errors` key; SNAP is unchanged.
Traces: contract (services §2.5 "caps the body")
Check: fetch_url

**Scenario**: C-4 — the success shapes: 201 with Location and the record, 200 with the record, 204 with nothing
**Given**: ADMIN is signed in.
**When**: `POST /v1/lines` `{"name":"Tc Shape"}`; `PUT /v1/lines/tc-shape` `{"blurb":"Tc shape blurb"}`; `DELETE /v1/lines/tc-shape`; `POST /v1/categories` and `POST /v1/reviews` once each.
**Then**: The `POST`s answer 201 with a `Location` header `/v1/<collection>/<id>` and a JSON body holding the record (`id`, `name` for a line and a category; `id`, `product`, `name`, `rating`, `date`, `status` for a review); the `PUT` answers 200 with the whole line (`id`, `name`, `blurb`, `since`, `fit_note`); the `DELETE` answers 204 with an empty body and no `Content-Type` body.
Traces: contract (services §3.22 "Responses")
Check: fetch_url

**Scenario**: C-5 — the file route 404s for a name that is not one of the shop's
**Given**: No cookie.
**When**: `GET /v1/uploads/nope.png`, `/v1/uploads/<32 hex>.png` that was never stored, `/v1/uploads/<32 hex>.gif`, `/v1/uploads/..%2Fx` and `/v1/uploads/<name of a stored picture in capitals>` are sent.
**Then**: Each answers 404 `not_found` with the error body (not 200, not 400).
Traces: contract (services §3.6, §2.5 path segments)
Check: fetch_url

**Scenario**: C-6 — a write whose query fails is 500 internal with a generic message
**Given**: ADMIN is signed in; `categories` is renamed (`ALTER TABLE categories RENAME TO categories_tc`); SNAP is taken before the rename.
**When**: `POST /v1/categories` `{"name":"Tc Broken"}` is sent; then the table is renamed back.
**Then**: 500 with `error` `internal` and a generic `message` that names no table, column, SQL or path; `X-Request-Id` is present; after the rename back `GET /v1/categories` answers 200 and holds no `Tc Broken`.
Traces: contract (services §2.3 `internal`; §3.22)
Check: fetch_url

**Scenario**: C-7 — every request field the contract does not define on the size change and the upload is refused 400
**Given**: STAFF is signed in; FX-SHOE exists.
**When**: `PATCH /v1/products/tc-shoe/sizes` `{"size":"38","pairs":1,"seen_pairs":4,"extra":1}`; `POST /v1/uploads` as multipart with the part `file` (a PNG) and a second part `caption`.
**Then**: Both answer 400 `bad_request` with the error body; no count changed and no upload was stored.
Traces: contract (services §3.23, §3.24, §2.5)
Check: fetch_url

**Scenario**: C-8 — the guard answers before the body is read, and a customer is told apart from a visitor
**Given**: NONE, then CUSTOMER.
**When**: `PATCH /v1/products/tc-shoe/sizes` is sent with the body `{` by each; `DELETE /v1/products/zzz-none` is sent by each.
**Then**: With no cookie both answer 401 `unauthenticated`; with the customer's cookie both answer 403 `forbidden` — never 400 or 404.
Traces: contract (services §2.2 roles; §3.22, §3.23)
Check: fetch_url

## 14. Coverage — every acceptance criterion walked

No case is `Check: manual`: every criterion is a response (`fetch_url`), a rendered page (`render_url`) or an operated page (`interact_page`). The cases that need an instrument the run may not have (a held, failed or aborted request; a file handed to a file input; a backend stopped and started; a deleted session) say so in section 2 and are UNVERIFIED for that reason if the instrument is missing, never failed.

| Criterion | Tag | Cases |
|---|---|---|
| SC-1 no session, forged or expired cookie: 401 on every write; public read still 200 | BE | TC-01, TC-02, TC-03, TC-04, C-8 |
| SC-2 a customer: 403 on every write; read as a visitor | BE | TC-05, TC-06, C-8 |
| SC-3 admin and staff the same; no write route on delivery, payments, shop, sizes, faqs | BE | TC-07, TC-08 |
| SC-4 create: 201, Location, dealer view, slug id, -2 / -3, line / category / review ids, next visitor read | BE | TC-09, TC-10, TC-11, C-4 |
| SC-5 required fields, two faults, unknown / derived field, body not an object, English sentences | BE | TC-12, TC-13, TC-14, TC-15 |
| SC-6 price, status, colour, references, released, unset line / category | BE | TC-16, TC-17, TC-18 |
| SC-7 stars, 200 / 201 and 5,000 / 5,001 characters, characters not bytes | BE | TC-19, TC-20 |
| SC-8 partial PUT, body id ignored, 404 on four collections, null on a required field | BE | TC-21, TC-22, TC-23, TC-24 |
| SC-9 two PUTs at once | BE | TC-25 |
| SC-10 one featured shoe, the home page, two saves at once | BE | TC-26, TC-27, TC-28 |
| SC-11 draft shoe and hidden review hidden from visitors; dealer sees them | BE | TC-29, TC-30, TC-31 |
| SC-12 rating and count follow live reviews on every event; last review → 0 and no rating line | BE | TC-32, TC-33, TC-34, TC-35 |
| SC-13 the shop's figures; seeded figures until an event | BE | TC-36, TC-37, TC-38 |
| SC-14 a shoe an order refers to: 409; set to draft instead | BE | TC-39 |
| SC-15 a free shoe goes with its reviews and counts; placeholder included | BE | TC-40, TC-41 |
| SC-16 line / category in use, freed, unused; review always | BE | TC-42, TC-43 |
| SC-17 JPEG / PNG / WebP stored and served | BE | TC-44 |
| SC-18 GIF, renamed text file, PNG named .txt, empty, not multipart, extra part | BE | TC-45, TC-46 |
| SC-19 5,242,880 and 5,242,881 bytes; path in the file name | BE | TC-47, TC-48 |
| SC-20 create with counts; derived lists; size filter; total 18 | BE | TC-49, TC-50 |
| SC-21 the size change; sold out, last pairs, neither; 404 and 422 | BE | TC-51, TC-52, TC-53 |
| SC-22 bad counts; size without a count; unreadable, duplicate, disagreeing sizes | BE | TC-54, TC-55 |
| SC-23 409 after an order; whole-form 409; the race | BE | TC-56, TC-57, TC-58 |
| SC-24 a size left out; price alone | BE | TC-59 (and TC-71 for the request of an untouched form) |
| SC-25 no second copy of stock; an order takes from the typed count | UI+BE | TC-60, TC-61 |
| SC-26 add a shoe in the browser; the shop, its page, the size filter | UI+BE | TC-62, TC-63 |
| SC-27 Draft removes it; On sale brings it back | UI+BE | TC-64 |
| SC-28 Saving…, one request; list drawn from the answer | UI | TC-65, TC-66, TC-87 |
| SC-29 field errors with the values kept | UI | TC-67, TC-68 |
| SC-30 unreachable; session ended | UI | TC-69, TC-70 |
| SC-31 Saved. and the reload | UI+BE | TC-71 |
| SC-32 the delete question, Cancel, confirm, the refusals in the dialog | UI+BE | TC-72, TC-73 |
| SC-33 record not here, and not while fetching | UI+BE | TC-74 |
| SC-34 the lists, placeholders, filters, address, empty and no-match | UI+BE | TC-75, TC-76, TC-77 |
| SC-35 the picture control: upload, working, Save waits, refusals, unreachable | UI+BE | TC-78, TC-79, TC-87 |
| SC-36 the count for each size; read-only lists; add and remove a size | UI+BE | TC-80, TC-81 |
| SC-37 one count saves by itself; down; 409 | UI+BE | TC-82, TC-83, TC-84 |
| SC-38 phone: no sideways scroll, reachable buttons, labels, announcements | UI | TC-85, TC-86 (and the "announced" assertions of TC-65, TC-67, TC-69, TC-70, TC-73, TC-79, TC-84) |
| SC-39 100 saves and 100 deletes at p95 ≤ 1 s | BE | TC-88 |
| Contract (`services.md` §2.3, §2.5, §3.22, §3.23, §3.24, §3.6) | — | C-1 … C-8 |

**Counts:** 39 acceptance criteria, all covered; 88 `TC` cases and 8 `C` cases, 96 in all. 52 are `fetch_url`, 1 is `render_url`, 43 are `interact_page`, none is `manual`.
