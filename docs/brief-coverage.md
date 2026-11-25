# Brief coverage

Every line the owner chose on the brief form, quoted as the brief writes it, with the requirements that meet it.
Requirement ids are in `docs/{module}/SRS.md` (`SITE` → `site`, `DATA` → `data`, `CAT` → `catalogue`, `ORD` → `orders`, `PAY` → `payments`, `ADMIN` → `admin`, `ACC` → `accounts`).

## WHAT IT MUST DO

| Brief line | Requirements | What meets it |
|---|---|---|
| "show what is offered — the menu, the products, the services — with prices" | SITE-004, SITE-005, SITE-006, CAT-002, DATA-001, DATA-002 | The home page, the shoe page and the shop page show every live shoe with its price; the records come from the database |
| "a cart and online ordering" | SITE-007, SITE-016, ORD-001, ORD-002, ORD-003, ORD-004, ORD-005, ORD-006, ORD-007, ORD-009, ORD-010 | The bag (drawer and page) and the checkout that places an order, takes the pairs off the shelf and shows the order; delivery is free on every order, so the total is the sum of the pairs |
| "online payment" | PAY-001, PAY-002, PAY-003, PAY-004, PAY-005, ORD-006, PAY-020, PAY-021, PAY-022 | VNPay (card, domestic ATM card, QR) as the only way to pay at launch — no cash on delivery — with the notice as the only thing that marks an order paid and no way for the dealer to mark one paid by hand; VietQR transfer and MoMo in the later story |
| "a dashboard: figures and charts at a glance" | ADMIN-020, ADMIN-021 | The dealer's overview with four figures, a 14-day sales chart, pairs sold by line, the latest orders and what is running low |
| "an admin area where the owner manages the content and the records" | ADMIN-001, ADMIN-002, ADMIN-003, ADMIN-004, ADMIN-010, ADMIN-011, ADMIN-012, ADMIN-013, ADMIN-014, ADMIN-015, ADMIN-016, ADMIN-030, ADMIN-031 | Sign-in for admin, staff and customers, the protected area, staff accounts managed by the admin, and saving shoes (with a plain count for each EU size), lines, categories, reviews and orders, with picture upload |
| "search across the site, with filters" | CAT-001, CAT-002, CAT-003, CAT-004, SITE-001 | Free-text search with accents ignored, filters by category, line, colour, price band and EU size, sorting, all in the address; the header's search lands on the same results |
| "a way to call or message" | SITE-003, SITE-012 | The call and Zalo dock on a phone, the contact page's ways to reach the shop, and the message form |
| "Order status tracking (in the owner's words)" | ORD-007, ORD-020, ORD-021, ORD-022, ORD-023, ADMIN-016 | The order page with its five steps, the track page that finds an order by number and phone, and the dealer moving the order along |
| "Return policy page (in the owner's words)" | SITE-010 | The Return policy page: "No exchanges", stating no window, condition, fee or refund because the owner gave none |
| "Shipping information page (in the owner's words)" | SITE-009, SITE-016 | The Delivery page: free delivery on every order |
| "Stock availability display (in the owner's words)" | SITE-005, ORD-004, ORD-005, ADMIN-015 | Sizes in stock, on last pairs and sold out on the shoe page, kept true as pairs sell and as staff type the pairs for each size |
| "Shoe size chart (in the owner's words)" | SITE-008 | The Size chart page with the foot-length finder and the full chart |

## WHAT THE FEATURES NEED

The brief of this project has no "WHAT THE FEATURES NEED" lines. The payment gateway the owner's business needs was decided in the plan: card payments go through VNPay (PAY-001 to PAY-005), with cash on delivery at launch (ORD-006) and VietQR and MoMo later (PAY-020 to PAY-022).

## HOW CUSTOMERS GET IN TOUCH

| Brief line | Requirements | What meets it |
|---|---|---|
| "a message form on the site — name, phone or email, message; the owner reads what is sent in Agent Team" | SITE-012 | The form on `/contact`: "Your name", "Phone or email", "Your message" and "Send message", sent to the owner's Agent Team inbox, with sending, sent, failed and not-connected states |
