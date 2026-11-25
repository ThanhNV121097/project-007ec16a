# Đôi — an adidas dealer, set in Hairline

The owner chose the template **Hairline** by looking at it (`design/template.jpg`): white, hairline
rules, big quiet type, the goods on pale grey tiles and nothing else coloured. This file is not a
direction — the template is one. It says how this business sits in it.

## The business, in two lines each

- **What it sells, to whom.** Genuine adidas shoes — men's, women's and kids', by line (Samba, Handball
  Spezial, Gazelle, Adizero, Megaride) — sold online by an authorised-style dealer to people in Vietnam
  buying for themselves or as a gift, mostly on a phone.
- **The one action.** Choose a pair in the right size and order it; afterwards, check where the order is.
- **What exists.** Nothing from the owner: no name, logo, photos, prices, stock, contact details or
  policies. Every record below is crawled or invented, and marked.
- **Currency.** The brief is Vietnamese, so prices are VND, written `3.290.000 ₫` (through `formatPrice`).
- **The name.** The owner gave none, so the shop is called **Đôi** — Vietnamese for "a pair". It is short,
  it is what a customer buys, and it is not adidas's name: the footer says the shop is an independent
  dealer and not adidas.

## Sites learned from

| Site | What I took | How it became this shop's |
|---|---|---|
| `design/template.jpg` — Hairline (the owner's choice) | The whole look: hairline rules, the product page's arrangement (name large left, square tile right), caption rows, disclosure rows, outline pills, colour dots | Built as is; the product page and the home page's first screen are that picture with a shoe in it |
| https://www.featuresneakerboutique.com/collections/adidas/products.json?limit=40 | The trade's catalogue convention: one shoe per frame on a pale grey seamless, a side view, a front pair and a back pair; a spec list per shoe (upper, lining, outsole, style code, colourway) | Every product's three pictures and its spec table come from Feature's pages; the grey seamless IS the template's tile |
| https://sneakerdaily.vn/danh-muc-san-pham/adidas-samba/ | How a Vietnamese sneaker shop narrows a list: by price band in VND and by EU size in thirds (36 2/3, 37 1/3…), with a count against each | The catalogue's filter rail: line, who it's for, colour, price band, and a grid of EU sizes in thirds |
| https://sneakerdaily.vn/san-pham/giay-adidas-samba-og-white-black-gum-b75806/ | A size guide one tap from the size buttons; "free size exchange" said beside the price; instalment/COD reassurance under the button | The size chart link sits on the size picker's label row; the stock line and delivery promise sit under "Add to bag" |
| https://baymard.com/mcommerce-usability/benchmark/mobile-page-types/product-list | What a list holds on a phone: the count, filters behind one button, sort, cards that carry enough to choose | Count beside the heading, filters fold on a phone, sort as a Select, each card a caption row plus one line of what it is |
| https://baymard.com/checkout-usability/benchmark/step-type/cart | The bag shows the total cost up front, delivery included, with the lines' pictures | The bag's summary column states delivery as "Free" before checkout, and every line keeps its tile |
| https://baymard.com/checkout-usability/benchmark/step-type/payment | Payment methods as marked options, the order summary kept in view beside the form | Checkout's three numbered groups at the left, the summary with pictures at the right; COD, bank transfer, VNPay and MoMo as radio rows |
| https://baymard.com/checkout-usability/benchmark/step-type/receipt | The answer first ("Order placed"), then the number, the delivery date, the items | The order page leads with the status sentence at display size, then the five-step line |
| https://dribbble.com/search/contact-page | Ways to reach set BESIDE the form, not under it; the form as the right half | Contact's left half is the routes a customer has (questions answered elsewhere, as hairline rows), the right half the form |

## What moved from the template, and what did not

- **Palette: nothing moved.** `--ground #F8F8F8 · --surface #E8E8E8 · --ink #080808 · --ink-soft #6B6B6B ·
  --accent #080808 · --accent-ink #FFFFFF · --line rgba(8,8,8,.12)`. The reason is the photographs:
  adidas colourways are the loudest thing a shoe shop has (gold suede, wonder-quartz pink, halo green),
  and the template's rule that colour arrives only inside the pictures is exactly right for them. A
  re-tint would compete with the shoes.
- **Faces: Inter 500 / 400, as the template.** One neutral grotesque; the shoes carry the character.
- **Scale:** h1 72px at 1440 (the house ceiling; the template measures 88), h2 32px, body 15px. The
  catalogue's own heading uses the same h1, because its first screen still has to show shoes.
- **Shape:** 0 radius on tiles and panels, full pills on every button, chip and the size buttons.
- **Unlocks:** none taken. The template already sets the arrangement; nothing else was needed.

## How this business fills the devices

1. **HAIRLINE RULE** — under the header, above and below every band; also between the size chart's rows
   and the bag's lines. (`Rule.tsx`, `border-line`)
2. **PRODUCT TILE** — every shoe photograph: Feature's pale grey seamless is the tile itself, so the
   photograph fills a square frame edge to edge. (`ProductImage.tsx`, aspect 1/1 everywhere)
3. **DISCLOSURE ROW** — under a shoe: Details, Payment options, Delivery & returns; on the shipping,
   returns and size pages: the questions customers ask. (`Disclosure.tsx`)
4. **OUTLINE PILL** — "More", "All shoes", "See this pair", "Track an order". (`OutlinePill.tsx`)
5. **CAPTION ROW** — under every tile: the shoe's name in letterspaced capitals left, its price right.
   (`ProductCard.tsx`)
6. **COLOUR DOTS** — the colourways of one model (the three women's Samba OG, the two kids' Samba OG),
   each dot a link to that pair; the chosen one ringed. (`ColourDots.tsx`)

## What the template has that this shop cannot make

The template's goods were shot one to a frame on a flat grey ground by somebody with a studio. The
owner has no photographs yet, so each shoe's three pictures are **borrowed** from Feature Sneaker
Boutique's product pages — which shoot exactly that way — and saved under `public/images/ref/`. They
are stand-ins: the Studio marks them and the site cannot go online until the owner replaces them with
their own (or adidas's, under their dealer agreement). The about page's two scenes are stock
(Pixabay), credited in the footer.

## Motion — LIVELY, from the kit only

The hero's headline words rise (`SplitHeading`), its tile drifts a few percent against the scroll
(`data-parallax` inside a clipping frame), "See this pair" and "Add to bag" lean to the pointer
(`data-magnetic`), the New-in cards tilt (`data-tilt`), the catalogue's count counts up
(`data-count`), each card shows its second picture under the pointer (`ProductImage hover`) and the
product photograph is looked into under the pointer (`ProductGallery`'s lens). Every section arrives
through `<Motion />`. Native scrolling; nothing pinned or scrubbed.

## THE PAGES NOBODY DREW

`/` and `/shop/:id` are the template's own picture (the product page; the home page opens on the same
arrangement), so they are not here. The management area is the scaffold's dashboard and is not here.

### /shop
- **Who arrives:** someone who tapped Men, Women, Kids or a line, or searched — they came to narrow
  eleven-to-a-hundred pairs to the two they would open.
- **First screen at 1440×790:** the heading ("All shoes" / "Women") at h1 with the count right-aligned
  and counting up; under the rule, the search box and sort, the filter rail at the left and the first
  row of three tiles at the right.
- **The look:** HAIRLINE RULE between heading, toolbar and grid; PRODUCT TILE + CAPTION ROW for every
  card, with one soft line under it (line · how many sizes left). The filter rail's groups are divided
  by hairlines, and EU sizes are pills in thirds.
- **From what I looked at:** https://baymard.com/mcommerce-usability/benchmark/mobile-page-types/product-list
  (the count beside the heading; filters folded on a phone; cards that say enough to choose) ·
  https://sneakerdaily.vn/danh-muc-san-pham/adidas-samba/ (price bands in VND; EU sizes in thirds).

### /cart
- **Who arrives:** someone with a pair in the bag deciding whether to pay — or a visitor with nothing
  in it.
- **First screen:** "Your bag" at h1 and the count; the lines at the left (tile, name in caps, size,
  colourway, a − / + stepper, remove, line price right-aligned); at the right the summary — subtotal,
  delivery "Free", the total large, the black "Check out" pill, and the three promises with icons.
- **The look:** PRODUCT TILE on every line; HAIRLINE RULE between lines; OUTLINE PILL "Keep shopping".
  Empty: the sentence, then the New-in row of four tiles with caption rows.
- **From what I looked at:** https://baymard.com/checkout-usability/benchmark/step-type/cart (total
  cost, delivery included, before checkout; pictures on every line).

### /checkout
- **Who arrives:** a buyer who pressed "Check out" — to give an address, choose delivery and payment,
  and place the order.
- **First screen:** "Checkout" at h1; three numbered groups at the left (1 Contact and address,
  2 Delivery, 3 Payment — numbered because they are a sequence); the summary at the right with each
  pair's tile, the total and "what happens after you press Place order".
- **The look:** HAIRLINE RULE between the groups; delivery and payment as hairline radio rows with
  icons; PRODUCT TILE in the summary.
- **From what I looked at:** https://baymard.com/checkout-usability/benchmark/step-type/payment
  (methods as marked options; the summary in view beside the form).

### /orders/:id
- **Who arrives:** a buyer straight from checkout, or one who looked an order up — to know whether it is
  paid and when it arrives.
- **First screen:** the answer at h1 ("Your order is on its way." / "Thank you — order placed."), the
  order number and expected date; the five-step line (Placed · Confirmed · Packed · Shipped · Delivered)
  with the current step marked; the items with their tiles.
- **The look:** HAIRLINE RULE as the step line itself; PRODUCT TILE + CAPTION ROW for the items;
  DISCLOSURE ROW for "Need to change something?".
- **From what I looked at:** https://baymard.com/checkout-usability/benchmark/step-type/receipt (the
  answer first, then the number, the date and the items).

### /track
- **Who arrives:** a buyer with an order number from the confirmation message.
- **First screen:** "Track an order" at h1, the two fields (order number, phone) and the pill at the
  left; at the right, what each of the five statuses means, as disclosure rows.
- **The look:** DISCLOSURE ROW for the status meanings; OUTLINE PILL to submit; HAIRLINE RULE.
- **From what I looked at:** https://baymard.com/checkout-usability/benchmark/step-type/receipt (the
  five steps a buyer is told about).

### /size-chart
- **Who arrives:** someone between two sizes, often buying for somebody else.
- **First screen:** "Find your size" at h1; at the left, the three numbered steps to measure a foot and a
  foot-length box that answers with the EU size; at the right, the adidas chart (EU · UK · US M · US W ·
  cm), numbers right-aligned, the answered row marked.
- **The look:** HAIRLINE RULE between every chart row; DISCLOSURE ROW for how each line fits.
- **From what I looked at:** https://sneakerdaily.vn/san-pham/giay-adidas-samba-og-white-black-gum-b75806/
  (the size guide a tap from the sizes; EU sizes in thirds).

### /shipping
- **Who arrives:** someone deciding whether to order online at all.
- **First screen:** "Delivery" at h1; three oversized figures (free standard delivery, 2–4 working days,
  next day by express in the two cities) on a hairline; then the delivery options as a right-aligned table.
- **The look:** HAIRLINE RULE; DISCLOSURE ROW for the questions; numbers set large as type.
- **From what I looked at:** https://baymard.com/checkout-usability/benchmark/step-type/cart (delivery
  cost stated before it is a surprise) · https://dribbble.com/search/contact-page (a page of words that
  opens on a strong first block, not a bare heading).

### /returns
- **Who arrives:** someone afraid the size will be wrong.
- **First screen:** "Returns and size exchange" at h1; the three figures (14 days to exchange a size,
  7 days to return for a refund, the first exchange's delivery on us); the three numbered steps.
- **The look:** HAIRLINE RULE; DISCLOSURE ROW for the questions; OUTLINE PILL to the contact form.
- **From what I looked at:** https://baymard.com/checkout-usability/benchmark/step-type/cart (the
  promise said where the doubt is) · https://dribbble.com/search/contact-page (one strong first block).

### /about
- **Who arrives:** someone checking the shop is real before paying.
- **First screen:** the template's second board: one statement set large and right-aligned; under the
  rule a photograph with a short paragraph beside it.
- **The look:** HAIRLINE RULE; OUTLINE PILL "See the shoes"; two photographs used large.
- **From what I looked at:** `design/template.jpg` (its about board) · https://dribbble.com/search/contact-page
  (a page of words that opens on its largest thing).

### /contact
- **Who arrives:** someone with a question about a size, an order or a gift.
- **First screen:** "Ask about a size or an order" at h1; at the left the quick answers as hairline rows
  with arrows (size chart, delivery, returns, track an order) and the reply time; at the right the form.
- **The look:** HAIRLINE RULE; hairline link rows; OUTLINE PILL.
- **From what I looked at:** https://dribbble.com/search/contact-page (the ways to reach beside the form).

## COMMITMENTS

1. A 1px hairline in --line runs the full content width under the header and between every band of every page; no shaded panel separates two sections.
2. Every shoe photograph fills a square, square-cornered pale grey tile with no border, no shadow and no label inside it — in the grid, the bag, the checkout summary and the order page.
3. Under every tile in a grid, one caption row: the shoe's name in letterspaced capitals at the left, its price in tabular figures at the right.
4. The home page's first screen is one pair: its name at 72px over two lines at the left with its size range, price, colour dots and three lines of description, its square tile filling the right half, and no photograph across the top.
5. A shoe's page carries three disclosure rows — Details, Payment options, Delivery & returns — each a label at the left and a + at the right with a hairline between.
6. The colour dots on a shoe link the other colourways of the same model; the chosen dot carries a thin ring.
7. The size picker shows stock per size: sold-out sizes struck through and not choosable, last-pair sizes marked, and a stock line under the picker that names them.
8. Motion: the home headline's words rise, the home tile drifts against the scroll, "See this pair" and "Add to bag" lean to the pointer, the New-in cards tilt, the catalogue count counts up, and a shoe's photograph is looked into under the pointer.
