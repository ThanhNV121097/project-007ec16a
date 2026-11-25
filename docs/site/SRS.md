# SRS — Site as designed

Module: `site`
Design: [View the approved design](http://localhost:9401)
Design system: `design/design-system.md`

> One file per module. This one covers the customer-facing pages that read records but take no order:
> header, footer, home, shoe page, bag, size chart, delivery, returns, about and contact. The catalogue
> page is in `docs/catalogue/SRS.md`, checkout / order / track in `docs/orders/SRS.md`.

## 1. Purpose

Đôi is an independent online dealer of genuine adidas shoes in Vietnam. This module is the part of the
site a customer reads before buying: which pairs there are, what they cost, which sizes are in stock,
how delivery and exchange work, and how to reach the shop. Most customers are on a phone, so every page
works at phone width first. Without it a customer cannot choose a pair or trust the shop enough to order.

## 2. Actors

| Actor | Who they are | What they may do in this module |
|---|---|---|
| Visitor | Anyone with the address, as a guest or signed in as a customer (customer accounts are in `docs/accounts/SRS.md`) | Read every page, fill the bag, use search, send a message |
| Dealer | The shop owner or staff | Read the messages visitors send, in Agent Team; change copy and colours in the Studio |

## 3. Scope

**In scope** — the plan items, by title:

- The site as designed
- Product page waits for the shoe before saying it is missing
- Free delivery on every order, wording to match

**Out of scope**

- Filtering, sorting and the paged list on `/shop` — belongs to the `catalogue` module.
- Checkout, the order page and order tracking — belong to the `orders` module.
- Paying online — belongs to the `payments` module.
- Where the records come from (database, API, paging) — belongs to the `data` module; this module only requires what the pages show.
- Dealer screens under `/admin` — belong to the `admin` module.
- Customer accounts — belong to the `accounts` module; the header's "Account" entry arrives with it (ACC-007). Wish lists and customers writing reviews — not in the brief or the design; reviews are shown here and managed by the shop team in `admin`.
- Example bag lines that the design shows to reviewers in the Studio (demo mode) — never shown on the live site.

## 4. Functional requirements

### 4.1 The site as designed

**Requirement SITE-001 — Header, search and bag button**

*As a* visitor, *I want to* reach the shop's sections, search and my bag from every page, *so that* I can move around without going back to the home page.

Behaviour:

1. Every customer page shows the header: the wordmark "Đôi" (links to the home page), five links — Women, Men, Kids, Size chart, Track order — a search button, a bag button, and below 1024px a menu button instead of the five links. A hairline runs under it. The header scrolls away with the page.
2. The link whose address equals the current address is underlined and marked as the current page.
3. The search button opens a search line under the header (1024px and up) or a full-screen panel (below 1024px). The input shows the placeholder "Samba, Gazelle, pink, a style code…", has focus on opening, and sits beside a "Search" button. The full-screen panel also shows "Start from a line" and one link per line, each to the shop narrowed to that line.
4. Submitting words goes to the shop with those words as the search; submitting nothing goes to the shop unfiltered. The panel closes on submit.
5. The bag button opens the bag drawer (SITE-007). A badge shows the number of pairs in the bag when it is more than zero and is absent at zero.
6. The menu button (below 1024px) opens a full-screen menu: the five links at large size, a hairline, then the help links Size chart, Delivery, Return policy, Track an order, Contact.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A 1280px-wide window on any customer page | The page loads | The header shows the wordmark and the links Women, Men, Kids, Size chart, Track order in that order, then the search and bag buttons; this holds until the customer-accounts story lands, which adds an "Account" entry (ACC-007) |
| AC-2 | A 390px-wide window | The page loads | The five links are not shown and a menu button is |
| AC-3 | The menu is open at 390px | The visitor reads it | It lists the five links, then Size chart, Delivery, Return policy, Track an order, Contact |
| AC-4 | The search line is open | The visitor types `samba og` and presses "Search" | The address becomes `/shop?q=samba%20og` |
| AC-5 | The search line is open and empty | The visitor presses "Search" | The address becomes `/shop` |
| AC-6 | The search panel is open at 390px | The visitor reads it | "Start from a line" is shown with one link per line, each to `/shop?line=<line id>` |
| AC-7 | The bag holds 3 pairs | Any page loads | The bag badge reads 3 |
| AC-8 | The bag is empty | Any page loads | The bag badge is not shown |
| AC-9 | The visitor is on `/track` at 1280px | The header renders | "Track order" is underlined and marked as the current page |
| AC-10 | The search line is open | The visitor reads the input | Its placeholder is `Samba, Gazelle, pink, a style code…` |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: the approved design has no error or empty state for the header, and it has no roles |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Lines (id, name) | read | — | One search-panel link per line |
| Bag pair count | read | — | Sum of the quantities in the browser's bag |

**Requirement SITE-002 — Footer**

*As a* visitor, *I want to* find help, policies and the shop's identity at the foot of every page, *so that* I trust the shop before ordering.

Behaviour:

1. Every customer page ends with a full-width dark band holding: the wordmark and the line "Đôi — genuine adidas, chosen online and delivered across Vietnam, with a size chart on every pair."
2. Three columns of links. "Shop": All shoes, Women, Men, Kids, Gifts (the shop narrowed to the price band under 3,000,000 ₫). "Help": Size chart, Delivery, Return policy, Track an order, Contact. "The shop": About Đôi, Your bag, Dealer sign-in.
3. Under a hairline: the independence notice ("Đôi is an independent dealer. adidas, Samba, Gazelle, Handball Spezial, Adizero and Megaride are trademarks of adidas AG; this shop is not adidas and is not run by adidas.") and the photo credits.
4. "Dealer sign-in" in the footer is the only link to the dealer area on a customer page.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Any customer page | The visitor scrolls to the end | The band shows the wordmark and the line "Đôi — genuine adidas, chosen online and delivered across Vietnam, with a size chart on every pair." |
| AC-2 | The footer | The visitor reads the "Shop" column | It lists All shoes (`/shop`), Women, Men, Kids (`/shop?category=<id>`), Gifts (`/shop?price=under-3m`) |
| AC-3 | The footer | The visitor reads the "Help" column | It lists Size chart, Delivery, Return policy, Track an order, Contact, linking to `/size-chart`, `/shipping`, `/returns`, `/track`, `/contact` |
| AC-4 | The footer | The visitor reads "The shop" | It lists About Đôi (`/about`), Your bag (`/cart`), Dealer sign-in (`/admin/sign-in`) |
| AC-5 | The footer | The visitor reads the legal line | It states Đôi is an independent dealer and that adidas marks belong to adidas AG |
| AC-6 | Any customer page except the footer | The page is searched for a link to `/admin` | The only such link is "Dealer sign-in" in the footer (the header, menu and body have none) |
| AC-7 | The site holds photo credits | The footer is read | One line begins "Photos:" and lists each photographer once, linking to the photo's own page where it has one, with the licence in brackets |
| AC-8 | The site holds no photo credits | The footer is read | No credits line is drawn |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: the footer is static copy; the approved design has no error state and no roles for it |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Footer copy and links | read | yes | From the site's content file; the owner edits it in the Studio |
| Photo credits | read | no | Listed when the site has any |

**Requirement SITE-003 — Call and message dock**

*As a* visitor on a phone, *I want to* call or message the shop in one tap from any page, *so that* I can ask before I buy.

Behaviour:

1. Below 1024px two round buttons float at the lower right corner of every customer page: "Zalo" (opens a chat at `zalo.me/<number>`) and a phone button labelled "Call us" (opens `tel:<number>`). From 1024px they are not shown.
2. Each button is shown only when its number is set in the site's content (phone, Zalo); with neither set, no dock is shown. A number keeps a leading `+` and drops every other non-digit.
3. The page's last line (footer, a submit button) always scrolls clear of the dock.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Phone `0901 234 567` and Zalo `0901234567` are set; 390px window | Any page loads | Two round buttons float at the lower right; "Call us" links to `tel:0901234567` and "Message us on Zalo" links to `https://zalo.me/0901234567` |
| AC-2 | Only the phone number is set | Any page loads at 390px | Only the call button is shown |
| AC-3 | Neither number is set | Any page loads at 390px | No dock is shown |
| AC-4 | Both numbers are set; 1280px window | Any page loads | No dock is shown |
| AC-5 | A phone number `+84 901 234 567` | The dock renders | The call link is `tel:+84901234567` |
| AC-6 | The dock is shown | The visitor scrolls to the page end | The last footer line is fully visible and not covered by a button |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Absent data | No phone and no Zalo number in the content | Nothing is drawn (no empty button); the message form (SITE-012) remains the way to get in touch |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Phone number | text | no | Owner-supplied; today empty (open question 1) |
| Zalo number | text | no | Owner-supplied; today empty (open question 1) |

**Requirement SITE-004 — Home page**

*As a* visitor, *I want to* see the featured pair, the new pairs, who the shoes are for and the lines, *so that* I can start browsing from the home page.

Behaviour:

1. The first section shows one pair at full size: the eyebrow "Featured pair · New this month", the pair's name as the page heading, its EU size range (`EU <first size> – <last size>`), its price, its colour dots (SITE-005 rules), its description, a "See this pair" pill to its page, and the line "In stock in {n} sizes" with a tick, where n is the number of its sizes not sold out. Its picture runs off the right edge on 1024px and up, links to the pair's page, and shows the pair's second picture under the pointer.
2. The pair shown is the live shoe the dealer marked as opening the home page; when none is marked, the first live shoe.
3. "New this month" (with an "All shoes" pill to `/shop`) shows up to four live shoes marked new, excluding the opening pair, as shoe cards.
4. "Who it's for" shows one tile per category (picture, name in capitals, "{n} in the shop", one-line blurb) linking to the shop narrowed to that category, plus a final "Gifts" tile — "{n} in the shop" counting live shoes under 3,000,000 ₫, line "Pairs under 3.000.000 ₫, sent with no price inside." — linking to `/shop?price=under-3m`.
5. "By line" shows the heading, the line "Five adidas lines, from the 1950 Samba to this year's runners.", and one row per line: name, "Since {year}" (hidden below 640px; empty when the line has no year), "{n} in the shop", an arrow; each row links to the shop narrowed to that line.
6. Counts "{n} in the shop" count live shoes of that category or line.
7. The page ends with the four promises: Free delivery ("{days} working days, anywhere in Vietnam."), "No exchanges" ("Check the size chart before you order."), "Pay online" ("Card, domestic ATM card or QR, through VNPay."), Genuine adidas ("Every pair boxed, tagged and sent with our receipt."), with days taken from the shop's terms. No promise mentions cash on delivery, a fee or a threshold.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Shoe A is live and marked as opening the home page | The home page loads | The first section shows A's name as the heading, A's price, "See this pair" linking to A's page, and the eyebrow "Featured pair · New this month" |
| AC-2 | No shoe is marked as opening the home page | The home page loads | The first section shows the first live shoe |
| AC-3 | A's sizes are `36…41 1/3` with `41 1/3` sold out | The home page loads | The range reads `EU 36 – 41 1/3` and the stock line reads "In stock in 8 sizes" |
| AC-4 | Six live shoes are marked new, one of them is the opening pair | The home page loads | "New this month" shows four cards, none of them the opening pair |
| AC-5 | The store holds 3 categories | The home page loads | "Who it's for" shows 3 category tiles plus a "Gifts" tile |
| AC-6 | 4 live shoes are in category "women" | The home page loads | The Women tile reads "4 in the shop" |
| AC-7 | 5 live shoes cost less than 3.000.000 ₫ | The home page loads | The Gifts tile reads "5 in the shop" and links to `/shop?price=under-3m` |
| AC-8 | A category's stored name is changed to "Ladies" | The home page loads | The tile reads "Ladies" (the name is not typed into the page) |
| AC-9 | A line has no "first made" year | The home page loads | Its row shows no "Since" text |
| AC-10 | The shop terms say standard delivery takes "2–4" days | The home page loads | The four promises read "Free delivery" with "2–4 working days, anywhere in Vietnam.", "No exchanges", "Pay online" and "Genuine adidas"; none mentions cash on delivery, a fee, a threshold or a number of exchange days |
| AC-11 | A draft shoe exists | The home page loads | The draft shoe appears in no section and in no count |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Absent data | No live shoe exists | The first section and "New this month" cards are not drawn; the rest of the page still draws |
| Boundary | Fewer than four shoes are new | Only those are shown; the grid centres the short row |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoe: name, price, sizes, sizes out, pictures, colour swatch, model, colourway, description, new, featured, status, category, line | read | yes | Live shoes only |
| Category: name, picture, blurb | read | yes | |
| Line: name, first-made year | read | yes | |
| Shop terms: standard days | read | yes | |

**Requirement SITE-005 — Shoe page: pictures, sizes and stock availability**

*As a* visitor, *I want to* see a pair from several sides and see which sizes are in stock, *so that* I can pick my size and add it to the bag.

Behaviour:

1. The address `/shop/<shoe id>` shows a "All shoes" back link (to `/shop`), the shoe's name as the heading, a line `<line> · <for> · <style code>` (parts that exist), the price, and — when the shoe has a rating above zero — stars with `<rating> · <n> reviews` linking to the reviews section.
2. The gallery shows the shoe's pictures in the order side view, front pair, back pair (missing ones left out). The first is large. With more than one picture, small thumbnails below switch the large picture; a single picture shows no thumbnails.
3. Colour dots show one dot per live shoe with the same model, filled with that shoe's swatch colour; the dot of the page being viewed is ringed; each dot links to that shoe's page. The label reads "Colour — <colourway>".
4. The size picker is headed "Size (EU)" with a "Size chart" link to `/size-chart`. It shows one pill per size, in the shoe's order. A sold-out size is struck through and cannot be chosen. A last-pairs size carries a small dot. Choosing a size selects it.
5. A status line under the sizes reads: with no size chosen and nothing wrong — "In stock in all {n} sizes." when no size is on its last pairs, or "In stock in {n} sizes. Last pairs in {sizes}." when some are; with a size chosen — "In stock in EU {size}. Sent today if you order before {cutoff}." or, for a last-pairs size, "Last pairs in EU {size} — order soon." (alert style). {n} counts sizes not sold out; {cutoff} is the shop's same-day dispatch time.
6. "Add to bag" with a size chosen adds one pair of that shoe in that size to the bag and opens the bag drawer. With no size chosen it adds nothing and the status line reads "Choose a size first." in alert style; choosing a size clears that message.
7. "Ask about this pair" goes to `/contact?about=<shoe name>`.
8. The shoe's description is shown under the buttons when it has one.
9. The sizes and their stock shown are the ones the shop holds when the page loads: given the stored sold-out list says a size is sold out, the page shows it sold out.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A live shoe "Samba OG Crystal White", style code IH9055, line Samba, category Women, price 3.090.000 ₫ | `/shop/samba-og-w-crystal-white` loads | The heading reads the name, the line reads `Samba · Women · IH9055`, the price reads `3.090.000 ₫` |
| AC-2 | The shoe has rating 4.9 and 17 reviews | The page loads | `4.9 · 17 reviews` is shown and links to the reviews section |
| AC-3 | The shoe has rating 0 or none | The page loads | No rating line is shown |
| AC-4 | The shoe has three pictures | The page loads | One large picture and three thumbnails are shown; pressing thumbnail 2 shows the front-pair picture large |
| AC-5 | The shoe has one picture | The page loads | No thumbnails are shown |
| AC-6 | Three live shoes share one model | The page loads | Three colour dots are shown, the current one ringed, each linking to its shoe's page |
| AC-7 | Size `41 1/3` is in the shoe's sold-out list | The page loads | The `41 1/3` pill is struck through and cannot be chosen |
| AC-8 | Size `40` is in the shoe's last-pairs list | The page loads | The `40` pill carries a dot |
| AC-9 | 9 sizes, 1 sold out, none on last pairs, no size chosen | The page loads | The status line reads "In stock in all 8 sizes." |
| AC-10 | 9 sizes, 1 sold out, `37 1/3` and `40` on last pairs, no size chosen | The page loads | The status line reads "In stock in 8 sizes. Last pairs in 37 1/3, 40." |
| AC-11 | Size `38` is in stock and not on last pairs; cut-off 15:00 | The visitor chooses `38` | The status line reads "In stock in EU 38. Sent today if you order before 15:00." |
| AC-12 | Size `40` is on last pairs | The visitor chooses `40` | The status line reads "Last pairs in EU 40 — order soon." in alert style |
| AC-13 | No size is chosen | The visitor presses "Add to bag" | The bag stays unchanged and the status line reads "Choose a size first." |
| AC-14 | Size `38` is chosen | The visitor presses "Add to bag" | The bag holds one pair of this shoe with variant `EU 38` and the bag drawer opens |
| AC-15 | The same shoe in `EU 38` is already in the bag | The visitor adds `38` again | That line's quantity is 2; no second line appears |
| AC-16 | The shoe is "Samba OG Black Crystal" | The visitor presses "Ask about this pair" | The address is `/contact?about=Samba%20OG%20Black%20Crystal` |
| AC-17 | The stored sold-out list is changed from `[]` to `["40"]` | The page loads again | The `40` pill is struck through and cannot be chosen |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | The address names a shoe that does not exist or is not live | The page shows the heading "That pair is not in the shop any more." and an "All shoes" pill to `/shop` (requirement SITE-014 governs when) |
| Boundary | Every size is sold out | Every pill is struck through and none can be chosen; "Add to bag" with no size chosen shows "Choose a size first." |
| Boundary | The shoe has fewer than 4 sizes or more than 9 | The size pills wrap with no lone pill in the last row (3, 4 or 5 columns chosen by count) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoe: name, line, category, style code, price, rating, review count, pictures, model, swatch, colourway, sizes, last-pairs sizes, sold-out sizes, description | read | yes | |
| Shop terms: same-day dispatch time | read | yes | |
| Bag line: shoe id, name, price, picture, variant `EU <size>`, quantity | write | — | Kept in the browser only |

**Requirement SITE-006 — Shoe page: details, payment options, delivery, reviews and related pairs**

*As a* visitor, *I want to* read a pair's details, the ways to pay, delivery terms and what buyers said, *so that* I feel safe buying it.

Behaviour:

1. Under the buy box three closed rows open on tap: "Details", "Payment options", "Delivery & return policy". Opening one shows its content; a row with nothing to show is not drawn.
2. "Details" lists, label and value: Line, For, Style code, Colourway, Upper, Lining, Sole, Fit, Care. A row whose value is empty is left out.
3. "Payment options" lists the ways to pay the shop offers at checkout (ORD-006): VNPay alone at launch, with its icon, its name followed by a full stop, and its note. The records of cash on delivery, bank transfer and MoMo stay on record and are not listed.
4. "Delivery & return policy" reads "Free standard delivery in {days} working days, sent the same day if you order before {cutoff}. Exchanges are not accepted." with the shop's terms filled in, then links "Delivery" (`/shipping`) and "Return policy" (`/returns`).
5. When the shoe has at least one live review, a section headed "What buyers said" shows the shoe's rating as a large number, its stars, "{n} reviews", and the line saying what most reviewers found about fit ("Most say it runs large." / "Most say it runs small." / "Most say it is true to size."). To its right up to three of the newest live reviews are listed: customer name, date, stars, review text, and "Bought EU {size} · {fit}" where fit reads "Runs small", "True to size" or "Runs large".
6. A shoe with no live review shows no reviews section.
7. "Looking for more?" with a "More" pill to `/shop` shows up to four other live shoes as cards — those of the same category first, then the others; it is not drawn when there are none.
8. Hidden reviews are never shown here.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A shoe whose Lining is empty | The visitor opens "Details" | The list has no "Lining" row; the other filled rows are shown |
| AC-2 | The shop has four ways to pay on record and offers VNPay at checkout | The visitor opens "Payment options" | One entry is listed, `VNPay — ATM, Visa, Mastercard. <note>` with its icon; cash on delivery, bank transfer and MoMo are not listed |
| AC-3 | Terms: standard "2–4" days, cut-off 15:00 | The visitor opens "Delivery & return policy" | The text reads "Free standard delivery in 2–4 working days, sent the same day if you order before 15:00. Exchanges are not accepted." |
| AC-4 | The same row | The visitor reads its links | "Delivery" goes to `/shipping` and "Return policy" goes to `/returns` |
| AC-5 | The shoe has five live reviews | The page loads | Exactly three reviews are listed, the newest first |
| AC-6 | A review by "Phương", dated 2026-11-21, 5 stars, size 38, fit "true" | The page loads | The row shows `Phương`, `21/11/2026`, five stars, the review text and "Bought EU 38 · True to size" |
| AC-7 | Most live reviews of the shoe say "small" | The page loads | The line "Most say it runs small." is shown |
| AC-8 | The shoe has one live review and one hidden review | The page loads | Only the live review is shown |
| AC-9 | The shoe has no live review | The page loads | No "What buyers said" section is drawn |
| AC-10 | The shoe has rating 4.8 and 23 reviews on record | The page loads | The section shows `4.8` large and "23 reviews" |
| AC-11 | Five other live shoes exist, two in the same category | The page loads | "Looking for more?" shows four cards, the two of the same category first |
| AC-12 | The shoe is the only live shoe | The page loads | "Looking for more?" is not drawn |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: the approved design draws no error state for these rows; a missing shoe is covered in SITE-005 and SITE-014 |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoe details: line, category, style code, colourway, upper, lining, sole, fit, care | read | no | Empty values are not drawn |
| Payment methods: name, note, icon | read | yes | Those the shop offers at checkout (VNPay) |
| Reviews: customer name, stars, date, size bought, fit, text, status | read | no | Live reviews only; three newest |
| Shop terms | read | yes | Standard days, same-day cut-off |

**Requirement SITE-007 — Bag: drawer and bag page**

*As a* visitor, *I want to* see and change the pairs I picked and what they come to, *so that* I know the total before I check out.

Behaviour:

1. The bag lives in this browser only: it survives a reload and closing the tab, and the drawer and the bag page show the same bag.
2. The drawer slides in from the right with the title "Your bag" and a close button. Each line shows the pair's picture, its name, its variant (`EU <size>`), its line total, a quantity stepper (minimum 1) and a remove button. Under the lines: "Total" and the sum, and a "Check out" button to `/checkout`. An empty bag shows "Nothing in your bag yet." and no total or button.
3. The bag page `/cart` shows the heading "Your bag", "{n} in your bag" (pairs in all) and a picture. Each line shows the picture (a link to the shoe), the name (a link to the shoe), the variant, "{price} each", the line total, a quantity stepper from 1 to 5, and "Remove" (text hidden below 640px, the button stays).
4. The summary on the bag page shows Subtotal, Delivery and Total. Delivery reads "Free" (the site's own word for it, never `0 ₫`) for every bag, whatever the subtotal, and Total equals Subtotal.
5. Below the summary: a "Check out" button with a lock icon to `/checkout`, the line "You pay online with VNPay — card, domestic ATM card or QR — at the next step.", the four promises in compact form, and a "Keep shopping" pill to `/shop`.
6. An empty bag on `/cart` shows the heading "Your bag is empty.", the line "Pick a pair — the size chart is one tap from every shoe.", a "Keep shopping" pill, and under a hairline "New this month" with up to four live new shoes as cards.
7. The figures on the bag page are an estimate for the visitor; the order's own total is computed by the shop at checkout (`orders` module).

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | One pair in the bag | The visitor reloads the page | The bag still holds the pair |
| AC-2 | The bag holds two lines | The visitor opens the drawer | Both lines show picture, name, `EU <size>`, line total and stepper, and "Total" shows their sum |
| AC-3 | The drawer is open with one line | The visitor presses its remove button | The drawer shows "Nothing in your bag yet." and no "Check out" button |
| AC-4 | The drawer is open | The visitor presses "Check out" | The address is `/checkout` and the drawer closes |
| AC-5 | A line of quantity 2 at 3.090.000 ₫ | The bag page loads | The line shows "3.090.000 ₫ each" and the line total `6.180.000 ₫`; the heading shows "2 in your bag" |
| AC-6 | A line of quantity 5 | The visitor looks at the stepper on the bag page | The plus button is disabled |
| AC-7 | A line of quantity 1 | The visitor looks at the stepper | The minus button is disabled |
| AC-8 | A subtotal of 3.090.000 ₫ | The bag page loads | Delivery reads "Free" and Total reads `3.090.000 ₫` |
| AC-9 | A subtotal of 1.500.000 ₫ (under 2.000.000 ₫) | The bag page loads | Delivery reads "Free", not `0 ₫`, and Total reads `1.500.000 ₫` |
| AC-10 | A non-empty bag | The visitor reads under the "Check out" button | The line "You pay online with VNPay — card, domestic ATM card or QR — at the next step." is shown, and the page mentions neither cash on delivery, nor bank transfer, nor MoMo |
| AC-11 | An empty bag | `/cart` loads | The heading "Your bag is empty.", the line "Pick a pair — the size chart is one tap from every shoe.", a "Keep shopping" pill and the "New this month" row are shown |
| AC-12 | A non-empty bag at 390px | The visitor reads a line | The "Remove" text is hidden and the remove button is still present |
| AC-13 | A non-empty bag | The visitor presses "Keep shopping" | The address is `/shop` |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Storage refused | The browser refuses local storage (private window) | The bag lasts for the open page only; adding, removing and totals still work |
| Conflict | The bag is changed in a second tab | The first tab shows the changed bag without a reload |
| Boundary | A shoe in the bag was made a draft or deleted afterwards | The line stays visible with the price it was added at; checkout (`orders` module) refuses the line |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Bag line: shoe id, name, price, picture, variant, quantity | read / write | yes | Browser-only |
| Delivery option "standard": fee | read | yes | Always 0; drawn as "Free" |

**Requirement SITE-008 — Shoe size chart**

*As a* visitor, *I want to* find my EU size from my foot length and read the whole chart, *so that* I order the right size.

Behaviour:

1. `/size-chart` shows the heading "Find your size", a lead, a picture, three numbered measuring steps, an input "Your foot length" (placeholder `24.5`, unit "cm"), an answer area, the chart, a note, "How each line fits", and "Questions about size".
2. The answer area has three states. Nothing typed, not a number, or a length outside 15 to 35 cm: the hint "Type a length such as 24.5 to see your size.". A length from 15 to 35 cm that the chart covers: the first chart row whose foot length is at least the typed length — "EU {eu}" large with "UK {uk} · US men {us_m} · US women {us_w}" below. A length from 15 to 35 cm beyond the last row: "That is beyond the chart — message us and we will check the pair for you."
3. A comma is accepted as the decimal mark (`24,5`).
4. The chart table, captioned "adidas sizes, adults and older kids", has the columns EU, UK, US men, US women and cm (one decimal), one row per size; the row that answers the typed length is highlighted.
5. Under the chart a note explains the lengths and that kids' sizes start at EU 35 1/2.
6. "How each line fits" has one closed row per line, named for the line, opening onto the line's fit note.
7. "Questions about size" lists the shop's questions of topic "sizing" as closed rows that open onto their answers.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The page has just loaded | The visitor reads the answer area | It reads "Type a length such as 24.5 to see your size." |
| AC-2 | The standard chart | The visitor types `25` | The answer reads "EU 40 2/3" with "UK 7 · US men 7.5 · US women 8.5", and the `40 2/3` row is highlighted |
| AC-3 | The standard chart | The visitor types `24.5` | The answer reads "EU 40" |
| AC-4 | The standard chart | The visitor types `24,5` | The answer reads "EU 40" |
| AC-5 | The last chart row is 28.4 cm | The visitor types `30` | The answer reads "That is beyond the chart — message us and we will check the pair for you." |
| AC-6 | Any chart | The visitor types `12` | The hint "Type a length such as 24.5 to see your size." is shown |
| AC-7 | Any chart | The visitor types `abc` | The hint is shown |
| AC-8 | The chart has 17 sizes | The page loads | The table has 17 rows with the columns EU, UK, US men, US women, cm; the cm values show one decimal (`21.6`) |
| AC-9 | Five lines exist | The visitor opens "How each line fits" | Five closed rows, one per line, each opening onto that line's fit note |
| AC-10 | The shop has questions of topic "sizing" | The visitor reads "Questions about size" | One closed row per such question; opening one shows its answer |
| AC-11 | The page loads | The visitor reads the page | The three measuring steps are numbered 01, 02, 03 and the note about kids' sizes is shown under the table |
| AC-12 | A chart row's stored EU label is changed | The page loads | The table and the answer show the changed label |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | The length is not a number, or below 15 or above 35 | The hint is shown; nothing else changes (AC-6, AC-7) |
| Boundary | Length exactly 15 and exactly 35 | Both count as asked: 15 answers with the first row; 35 answers "That is beyond the chart…" when no row reaches it |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Size rows: EU, UK, US men, US women, foot length (cm) | read | yes | Ordered as stored |
| Lines: name, fit note | read | yes | |
| Questions: topic, question, answer | read | no | Topic "sizing" |

**Requirement SITE-009 — Shipping information page**

*As a* visitor, *I want to* know what delivery costs, how long it takes and where express reaches, *so that* I can order without asking.

Behaviour:

1. `/shipping` shows the heading "Delivery", a lead ("Every pair leaves boxed, tagged and tracked, and delivery is free on every order."), and a picture.
2. Two large figures sit on a hairline: "Free" with "Delivery on every order, whatever it comes to — that is every pair in the shop."; and the standard working days (for example "2–4") with "Working days by standard delivery, anywhere in Vietnam." The days come from the shop's terms. There is no third (express) figure.
3. "Delivery options" is headed by a table with columns Option, Arrives, Where, Cost and one row, for the one delivery option: name, time, where and how, and in the Cost cell "Free" right-aligned (the site's own word for it, never `0 ₫`). There is no "Free over" line, no express row and no dispatch cut-off time on this page.
4. "Questions about delivery" lists the shop's questions of topic "shipping" as closed rows opening onto answers, beside a "Track an order" pill to `/track`.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The delivery record `standard` has fee 0 | `/shipping` loads | The lead reads "Every pair leaves boxed, tagged and tracked, and delivery is free on every order."; the first figure reads "Free" and its line reads "Delivery on every order, whatever it comes to — that is every pair in the shop." |
| AC-2 | Standard days "2–4" | The page loads | The second figure reads `2–4` with "Working days by standard delivery, anywhere in Vietnam."; there is no third figure |
| AC-3 | The shop's same-day cut-off is 15:00 | The page loads | The page does not state the cut-off; the words "express" and "Free over", the figures `30.000` and `50.000`, `0 ₫`, and any amount followed by ₫ appear nowhere on it |
| AC-4 | One delivery option, Standard | The page loads | The table headers read Option, Arrives, Where, Cost and one row follows, its Cost cell reading "Free" |
| AC-5 | The same | The visitor reads the Standard row | "Free" is its only cost; no "Free over" line appears under it and no Express row exists |
| AC-6 | Four questions of topic "shipping" | The page loads | Four closed rows are shown (one per such question), each opening onto its answer |
| AC-7 | The page loads | The visitor presses "Track an order" | The address is `/track` |
| AC-8 | The stored standard days are changed to "3–5" | The page loads | The second figure reads `3–5` and the row's Arrives cell shows the changed days |
| AC-9 | A 390px window | The page loads | The table scrolls sideways inside its own box; the page itself has no horizontal scroll |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: the approved design draws no error or empty state for this page |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shop terms: standard days | read | yes | |
| Delivery option "standard": name, days, note, fee | read | yes | The one option; its fee is always 0, drawn as "Free" |
| Questions: topic, question, answer | read | no | Topic "shipping" |

**Requirement SITE-010 — Return policy page**

*As a* visitor, *I want to* read how to exchange a size or return a pair, *so that* I am not afraid to guess my size online.

Behaviour:

1. `/returns`, titled "Return policy" (tab title and menu label), shows the heading "No exchanges", the lead "Exchanges are not accepted. Please check your size on the size chart before you order, and contact us if you have a question.", and a picture. The owner's answer to the return question is "no exchanges allowed"; they gave no return window, shoe condition, return-shipping rule or refund terms, so the page states none.
2. Three large words on a hairline, each with a line under it: "No" — "Exchanges are not accepted."; "Size" — "Check the size chart before you order."; "Ask" — "Call us, message us on Zalo or use the contact form." They are words, not numbers: none counts up and none comes from the shop's terms.
3. "What you can do" shows three numbered steps, each with its sentence: 01 "Check your size" ("Measure your foot and find your size on the size chart before you order."), 02 "Ask before you order" ("Not sure about a size or a pair? Call us, message us on Zalo or send the contact form."), 03 "Report a problem" ("If your delivery is wrong or damaged, contact us with your order number.").
4. "In short" shows two lines, each with a tick: "Exchanges are not accepted." and "Contact us with any question about an order, or to report a wrong or damaged delivery."
5. "Questions about exchanges" lists the shop's questions of topic "returns" as closed rows, beside a "Contact us" pill to `/contact`. No answer offers an exchange, a refund, a window, a condition or a fee.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The page loads | The visitor reads the figures | The words `No`, `Size` and `Ask` are shown, each with its line: "Exchanges are not accepted.", "Check the size chart before you order.", "Call us, message us on Zalo or use the contact form." |
| AC-2 | The page loads | The visitor searches the page text | The words "refund", "unworn", "original box", "within 7 days", "within 14 days" and "three working days" and the figure `0 ₫` appear nowhere on it, and no figure counts up |
| AC-3 | The page loads | The visitor reads "What you can do" | Steps 01, 02, 03 are titled "Check your size", "Ask before you order", "Report a problem" |
| AC-4 | The page loads | The visitor reads "In short" | Two lines are listed, each with a tick |
| AC-5 | Two questions of topic "returns" (the size one and the wrong-or-damaged-delivery one) | The page loads | Two closed rows are shown under "Questions about exchanges", each opening onto its answer |
| AC-6 | The page loads | The visitor presses "Contact us" | The address is `/contact` |
| AC-7 | The page loads | The visitor reads the top | The heading is "No exchanges", the lead sentence is shown and the browser tab is titled "Return policy" |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: the approved design draws no error or empty state for this page |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shop terms: exchange days, refund days | read | yes | |
| Questions: topic, question, answer | read | no | Topic "returns" |

**Requirement SITE-011 — About page**

*As a* visitor, *I want to* know who sells the shoes, *so that* I trust the dealer.

Behaviour:

1. `/about` shows the statement "Đôi sells genuine adidas, one pair at a time, to people who would rather not guess their size." as the heading, right-aligned and large; two photographs each beside a short paragraph; a "See the shoes" pill to `/shop` after the second paragraph.
2. Under a hairline three figures: the number of live shoes ("Shoes in the shop today, each with its sizes in stock."), the number of lines ("adidas lines, from the 1950 Samba to this year's runners."), and the shop's average rating ("Average rating from {n} reviews by people who bought here.").

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | 12 live shoes, 1 draft | `/about` loads | The first figure reads `12` |
| AC-2 | 5 lines | The page loads | The second figure reads `5` |
| AC-3 | Shop rating 4.8 from 181 reviews | The page loads | The third figure reads `4.8` with "Average rating from 181 reviews by people who bought here." |
| AC-4 | The page loads | The visitor reads it | The heading is the statement; both photographs and both paragraphs are shown |
| AC-5 | The page loads | The visitor presses "See the shoes" | The address is `/shop` |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: the approved design draws no error or empty state for this page |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Count of live shoes, count of lines | read | yes | |
| Shop terms: average rating, review count | read | yes | |

**Requirement SITE-012 — Contact page and message form**

*As a* visitor, *I want to* send the shop a message and see the other ways to reach it, *so that* I can ask about a size, an order or a gift.

Behaviour:

1. `/contact` shows the heading "Ask about a size or an order", the lead "Send a message about a size, an order or a gift. We answer {reply}." (reply time from the shop's terms), and a picture. When the address carries `?about=<shoe name>`, the line "You are asking about <shoe name>" is shown under the heading.
2. A list of ways to reach the shop shows Phone (a `tel:` link), Zalo, Email (a `mailto:` link), Shop (address) and Hours, each with its icon and label. A row whose value is empty is not drawn.
3. "Already answered" lists four links: Find your size (`/size-chart`), Delivery times and fees (`/shipping`), Returns and size exchange (`/returns`), Track an order (`/track`), each with an arrow.
4. "Send a message" holds the form: "Your name" (required), "Phone or email" (required), "Your message" (a four-line box), and a "Send message" button. A hidden field named "website" is off-screen, not tabbable and hidden from assistive technology; a person never fills it.
5. Pressing "Send message" with both required fields filled sends the visitor's name, phone-or-email, message and the page path to the shop's Agent Team inbox, where the owner reads it. While it is sending the button reads "Sending…" and is disabled. When Agent Team accepts it the form is replaced by "Thank you — we have your message and will answer the same working day."
6. The message box is optional. A message with a name and a way to reply and no text is accepted.
7. When Agent Team refuses or cannot be reached, the form stays, with the typed values kept, and shows the sentence Agent Team returned, or "The message could not be sent. Please try again, or call us." when it returned none; the button is enabled again.
8. When the site is not connected to Agent Team the form shows "This form is not connected yet — please try again later." above its button.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The form is connected; the visitor typed a name `Lan`, `0912345678` and a message | The visitor presses "Send message" | Agent Team receives one message with name `Lan`, reply-to `0912345678`, the message text and the page path `/contact`; the form is replaced by "Thank you — we have your message and will answer the same working day." |
| AC-2 | The message was sent | The owner opens Agent Team | The message is listed with the name, the way to reply and the text |
| AC-3 | The request is in flight | The visitor looks at the button | It reads "Sending…" and is disabled |
| AC-4 | The name field is empty | The visitor presses "Send message" | Nothing is sent and the browser points at the name field |
| AC-5 | The phone-or-email field is empty | The visitor presses "Send message" | Nothing is sent and the browser points at that field |
| AC-6 | Name and phone filled, message empty | The visitor presses "Send message" | The message is sent with an empty text and the thank-you shows |
| AC-7 | Agent Team answers with the sentence "Too many messages" | The visitor presses "Send message" | The form stays, shows "Too many messages", keeps the typed values, and the button is enabled |
| AC-8 | The network is down | The visitor presses "Send message" | The form stays, keeps the typed values and shows a sentence that the message could not be sent (the default sentence when none was returned); the button is enabled |
| AC-9 | The site has no Agent Team connection | The page loads | "This form is not connected yet — please try again later." is shown above the button |
| AC-10 | The form | The visitor tabs through it | Focus moves name, phone-or-email, message, button; the "website" field is skipped and has no visible box |
| AC-11 | `/contact?about=Samba%20OG` | The page loads | "You are asking about Samba OG" is shown under the heading |
| AC-12 | A phone number and an email are set in the content, address and hours are not | The page loads | Phone and Email rows are shown; Zalo, Shop and Hours rows are not |
| AC-13 | The page loads | The visitor reads "Already answered" | Four links lead to `/size-chart`, `/shipping`, `/returns`, `/track` |
| AC-14 | Reply time "the same working day" | The page loads | The lead reads "Send a message about a size, an order or a gift. We answer the same working day." |
| AC-15 | The "website" field contains text when the form is submitted | The submission is sent | The text is sent in the "website" field of the message, unchanged |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Invalid input | A required field is empty | The browser's own required-field prompt; nothing is sent (AC-4, AC-5) |
| Upstream failure | Agent Team refuses or is unreachable | The form stays with the values kept and shows the returned sentence or the default sentence; no partial send (AC-7, AC-8) |
| Absent data | No phone, Zalo, email, address or hours set | The ways-to-reach list has no rows; the form is unaffected |
| Concurrency | The visitor presses "Send message" twice quickly | The second press is ignored while the first is in flight (the button is disabled), so one message is sent |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Name | text | yes | Sent to Agent Team |
| Phone or email | text | yes | One field, any text; sent as "reply to" |
| Message | text | no | |
| Page path | text | — | Added automatically |
| Honeypot "website" | text | no | Sent as typed; only a bot fills it |

**Requirement SITE-013 — Page titles and descriptions**

*As a* visitor arriving from a search engine or a shared link, *I want* each page to carry its own title and description, *so that* I can tell the pages apart.

Behaviour:

1. Every customer page sets the browser tab title and the page description from the site's content (for example "All shoes", "Size chart", "Delivery").

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | Any customer page | It loads | The document title is that page's title from the content and the description is that page's description |
| AC-2 | Two different pages | Both load | Their document titles differ |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| — | — | Not applicable: no error state exists for titles |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Page title and description | read | yes | From the site's content file |

**Requirement SITE-015 — Motion at the level the owner chose**

*As the* owner, *I want* the site to move as much as I chose, and not at all for a visitor who asked their device for less motion, *so that* it feels alive without bothering anyone.

Behaviour:

1. The site's motion level is "lively", as the owner chose. When a page opens its content fades up by about 10px over about 360 ms; sections arrive as they scroll into view.
2. The home page's headline words rise in one after another.
3. The shop's count, and the first two figures of the returns page and the figures of the about page, count up to their value.
4. On a device with a fine pointer, the "New this month" cards tilt under the pointer; the "See this pair" and "Add to bag" buttons lean toward it; a shoe card's picture grows to 1.05 times its size over about 0.6 s and shows the shoe's second picture; the shoe page's photograph is looked into at twice its size.
5. A closed row's "+" turns a quarter turn when it opens; the arrow of a line row on the home page moves 4px to the right on hover.
6. A visitor whose device asks for reduced motion gets none of items 1 to 5: each page draws finished, every figure shows its final value at once, nothing is hidden waiting to move. Colour, underline and open/closed changes still happen.
7. A device without a fine pointer gets none of the pointer effects in item 4.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | No reduced-motion request | A page opens | Its content starts about 10px lower and transparent and ends in place within 500 ms |
| AC-2 | The device asks for reduced motion | A page opens | The content is in place at once, with no animation |
| AC-3 | No reduced-motion request | The home page opens | The headline's words rise in one after another |
| AC-4 | The device asks for reduced motion | The home page opens | The whole headline is shown at once |
| AC-5 | No reduced-motion request | `/shop` opens with 12 matches | The count shown rises from 0 to 12 |
| AC-6 | The device asks for reduced motion | `/shop` opens with 12 matches | The count shows 12 at once |
| AC-7 | A fine pointer and no reduced-motion request | The pointer rests on a shoe card | Its picture grows to 1.05 times and shows the second picture |
| AC-8 | A fine pointer and the device asks for reduced motion | The pointer rests on a shoe card | The picture does not grow and the second picture is not shown |
| AC-9 | A touch-only device | A shoe card is touched | The picture does not grow and no tilt happens |
| AC-10 | Any device | A closed row is pressed | It opens and shows its content (with or without reduced motion) |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Boundary | Reduced motion asked for | Nothing is hidden waiting to move; every page is complete without its animation script (AC-2, AC-4, AC-6) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Motion level | read | yes | "lively", from the site's settings |

### 4.2 Product page waits for the shoe before saying it is missing

**Requirement SITE-014 — The "not in the shop" message appears only once the shop has answered**

*As a* visitor, *I want to* see a shoe's page without a flash saying the pair is gone, *so that* I do not think the pair has been withdrawn when it is only loading.

Behaviour:

1. When `/shop/<id>` opens, the page does not show "That pair is not in the shop any more." while the shoe is still being fetched.
2. The message is shown only after the shop has answered that no live shoe has that id.
3. When the shop cannot be reached, the message is not shown; the page shows nothing in place of the shoe until the shop answers.
4. When the shoe arrives the page draws it at once.

**Acceptance criteria**

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | The shop answers a shoe request after 2 seconds | `/shop/samba-og-w-crystal-white` is opened | Until the answer arrives the text "That pair is not in the shop any more." is not on the page |
| AC-2 | The shop answers "not found" for the id | The answer arrives | The heading "That pair is not in the shop any more." and the "All shoes" pill are shown |
| AC-3 | The id belongs to a draft shoe | The page is opened | The same not-found message is shown after the answer |
| AC-4 | The shop is unreachable | The page is opened | The "not in the shop" message is never shown |

**Failure, boundary and permission behaviour**

| Case | Condition | Expected behaviour |
|---|---|---|
| Not found | The id has no live shoe | The message and pill, after the answer (AC-2, AC-3) |
| Upstream failure | The shop is unreachable | Nothing replaces the shoe; no "not in the shop" message (AC-4) |

**Data touched**

| Field | Type | Required | Rule |
|---|---|---|---|
| Shoe by id | read | yes | |

## 5. Screens

The design's pages and their states. A screen the design draws once has the single state `default`.

| Screen | Section in the design | Functions it serves | States that must exist |
|---|---|---|---|
| Header, search, menu, bag button | Every customer page, top | SITE-001, SITE-007 | default; search open; menu open (below 1024px) |
| Footer | Every customer page, bottom | SITE-002 | default |
| Call and message dock | Every customer page below 1024px | SITE-003 | default (hidden when no number is set) |
| Home `/` | Home | SITE-004 | default |
| Shoe `/shop/:id` | Shoe | SITE-005, SITE-006, SITE-014 | default; size chosen; no size chosen after "Add to bag"; shoe missing |
| Bag drawer | Bag drawer, from any page | SITE-007 | default; empty |
| Bag `/cart` | Bag | SITE-007 | default; empty bag |
| Size chart `/size-chart` | Size chart | SITE-008 | hint; answered; beyond the chart |
| Delivery `/shipping` | Delivery | SITE-009 | default |
| Returns `/returns` | Returns | SITE-010 | default |
| About `/about` | About | SITE-011 | default |
| Contact `/contact` | Contact | SITE-012 | default; sending; sent; failed; not connected |

Every page above is drawn once for a phone width and widens at 1024px, as `design/design-system.md` §1.5 describes. The drawer, the menu panel and the search panel are overlays on those pages.

## 6. Non-functional requirements

| Area | Requirement |
|---|---|
| Responsive | Every page in §5 works from a 320px window up with no horizontal page scroll (a table may scroll inside its own box) |
| Performance | On the Slow 4G preset (1.6 Mbps down, 150 ms round trip), 390px window, empty browser cache, the home page's largest visible element appears within 4 s |
| Accessibility | Every form field has a visible label; every picture has alt text, or an empty alt when decorative; the header, menu, search, bag drawer, size pills and contact form can be used with the keyboard alone, with a visible focus; colour contrast follows the audit in `design/design-system.md` §1.1 |
| Localisation | Copy is in English; prices are shown as in `vi-VN` (for example `3.290.000 ₫`); dates are shown day first (`21/11/2026`) |
| Privacy | The bag stays in the visitor's browser. The contact form sends the name, the phone-or-email, the message and the page path to the shop's Agent Team inbox and stores nothing else about the visitor |

## 7. Dependencies and assumptions

- **Depends on:** the `data` module, for every record these pages show (shoes, lines, categories, reviews, shop terms, delivery options, payment methods, questions, size rows).
- **Depends on:** Agent Team's site API (already configured in the site's settings) for the contact form and the dealer's message inbox. No backend story is needed for the form.
- **Assumption:** until the `data` story ships, the pages read the records from the files in the repository; the acceptance criteria above hold in both cases because they are about what the page shows for the stored value.
- **Assumption:** the catalogue stays under 100 live shoes, so one page of the data API (at most 100 records) holds every shoe the home, shoe and bag pages need.

| # | Open question | Proposed default | Who decides |
|---|---|---|---|
| 1 | The shop's phone, Zalo, email, address and opening hours are empty in the content today, so the dock and four of the five contact rows are not drawn. The brief asks for "a way to call or message". | The message form is the way in until the owner types the numbers into the Studio; the dock and rows appear by themselves when they do | Stakeholder |
| 2 | The shoe page lists all four ways to pay and the bag says "Pay by cash on delivery, bank transfer, VNPay or MoMo at the next step." but checkout offers only cash on delivery and, later, VNPay (project decision; VietQR and MoMo come in a later story) | Leave the shoe page list and the bag line as designed; the owner may edit the line in the Studio | Stakeholder |
| 3 | The design leaves the message box of the contact form optional; the brief lists "name, phone or email, message" | Keep it optional as designed | Stakeholder |
| 4 | With a sold-out size and no last-pairs size the design reads "In stock in all {n} sizes." with n counting only the sizes still in stock | Keep the design's wording | Stakeholder |
| 5 | The design has no screen for an unknown address (for example `/abc`) | Leave it as designed: no page is drawn beyond the header and footer | Stakeholder |

## 8. Traceability

| Plan item | Requirement ids | Test cases |
|---|---|---|
| The site as designed | SITE-001, SITE-002, SITE-003, SITE-004, SITE-005, SITE-006, SITE-007, SITE-008, SITE-009, SITE-010, SITE-011, SITE-012, SITE-013, SITE-015 | `test-cases/the-site-as-designed.md` |
| Product page waits for the shoe before saying it is missing | SITE-014 | `test-cases/product-page-waits-for-the-shoe-before-saying-it-is-missing.md` |
