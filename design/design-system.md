# Design System — Đôi

> Source of truth: the approved design in `code/frontend/src` — `theme.css` (every colour, face and
> shape), `index.css` (the Tailwind names for them), `content.json` (every word), `App.tsx` and
> `components/`. Every value below is extracted from it. Changing a value here without changing the
> approved design is a defect.
>
> Đôi is an independent adidas dealer, set in the **Hairline** template: white page, 1px rules, big quiet
> type, the shoes on pale grey tiles, and no colour anywhere except inside the photographs
> (`design/moodboard.md`). The management area (`/admin`) is a separate, fixed look scoped by
> `[data-dashboard]` and documented in §2.2.

Last updated: 2026-11-25

How to read the token names: a token is a CSS custom property in `src/theme.css`; its **Tailwind name** is the
utility a component writes (`--ground` → `bg-ground`, `--ink-soft` → `text-ink-soft`, `--line` → `border-line`,
`--field-line` → `border-field`). A component never writes a raw colour. Only the tokens the design uses are
listed.

## 1. Foundations

### 1.1 Color

The site is one light palette: **no dark mode** (`color-scheme: light`, no `[data-theme]`, no
`prefers-color-scheme`). Names are by job, exactly as `theme.css` has them.

#### Public site (`:root`)

| Token | Tailwind name | Value | Used for |
|---|---|---|---|
| `--ground` | `bg-ground` | `#F8F8F8` | Page background, form-field fill, drawer and mobile-menu panel |
| `--surface` | `bg-surface` | `#E8E8E8` | The grey of the photographs' own seamless: full-width bands behind a photograph or a working panel (`Bleed ground`), the catalogue's search/sort band, hover fill of the menu button, listbox and the contact dock's Zalo button |
| `--ink` | `text-ink`, `bg-ink`, `border-ink` | `#080808` | All text, filled pill buttons, selected size pill, ringed colour dot, bag-count badge |
| `--ink-soft` | `text-ink-soft` | `#6B6B6B` | Secondary text, captions, labels, disabled and struck-through sizes, placeholder text |
| `--accent` | `bg-accent`, `text-accent` | `#080808` | The footer band; the active option in a `Select` list; checkbox tick (`accent-[var(--accent)]`). Equal to `--ink` |
| `--accent-ink` | `text-accent-ink` | `#FFFFFF` | Text on `--accent` (footer, active listbox option) |
| `--line` | `border-line`, `bg-line` | `rgba(8,8,8,0.12)` (reads `#DBDBDB` on `--ground`) | The hairline: under the header, between every band, between list and table rows, disclosure rows, quiet control edges |
| `--field-line` | `border-field` | `rgba(8,8,8,0.45)` (reads `#8C8C8C` on `--ground`) | The edge of every form field, checkbox, radio dot and `Select` button — a field must be findable, so this is not `--line` |
| `--danger` | `text-danger`, `bg-danger` | `#B42318` | What is about to be lost or has gone wrong only: error text, cancelled-order mark, the "last pairs" dot on a size pill, low-stock line |
| `--danger-ink` | `text-danger-ink` | `#FFFFFF` | Text on `--danger` |
| `--wash` | `bg-wash` | `color-mix(in srgb, var(--ink) 6%, var(--surface))` (≈ `#DBDBDB`) | Quiet fill of a selected or hovered item in the kit's menus and rows; derived, never chosen |
| `--product-ground` | inline `var(--product-ground)` | `#E8E8E8` | The tile behind a product picture, seen until the photograph loads and round a transparent `.png` |
| `--product-ground-ink` | inline | `#6B6B6B` | The caption of a picture block where a product has no photograph yet |
| `--picture-ground` / `--picture-ink` | inline | `var(--surface)` / `var(--ink-soft)` | The ground and ink of a `<Picture>` placeholder block; `ProductImage` re-points both to the tile's values |

One-off tints the design computes from these (not tokens): the footer draws its text and rule in
`--accent-ink` at 60% / 70% (`text-accent-ink/60`, `/70`) and its hairline at 15% (`bg-accent-ink/15`); the cart
overlay is `bg-black/40`.

#### Management area (`[data-dashboard]`)

A scope on one area of one site, not a second mode: reached only by signing in at `/admin`. It overrides
colour, radius and elevation; it does **not** override the typeface. It uses Tailwind's neutral slate scale and blue-600.

| Token | Value | Used for |
|---|---|---|
| `--ground` | `#F8FAFC` | Page, sidebar inset, `Empty` icon tile |
| `--surface` | `#FFFFFF` | Panels, cards, fields, table, sticky header, sidebar |
| `--ink` | `#0F172A` | Text |
| `--ink-soft` | `#64748B` | Secondary text, descriptions, icons |
| `--accent` | `#2563EB` | Primary button, checked checkbox/switch, focus border and ring (at 50%), active link |
| `--accent-ink` | `#FFFFFF` | Text on `--accent` |
| `--line` | `#E2E8F0` | Panel and row borders, dividers |
| `--field-line` | `#CBD5E1` | The edge of inputs, selects, checkboxes |
| `--danger` | `#DC2626` | Destructive button, error text, invalid ring (at 20%) |
| `--danger-ink` | `#FFFFFF` | Text on `--danger` (the kit writes `text-white`) |
| `--wash` | `#F1F5F9` | Hover and selected fill of ghost/outline buttons, menu items, active sidebar row |
| `--picture-ground` / `--picture-ink` | `#F1F5F9` / `#64748B` | A product thumbnail's tile inside a row |

#### Contrast audit

Ratios are WCAG relative-luminance ratios. Translucent colours are measured as the colour they resolve to over the
named background. Body/small text needs ≥ 4.5:1, large text (≥ 24px, or ≥ 18.66px bold) ≥ 3:1, a control's boundary ≥ 3:1.

**Public site**

| Foreground | Background | Ratio | Passes |
|---|---|---|---|
| `--ink` `#080808` | `--ground` `#F8F8F8` | `18.9:1` | AA, AAA |
| `--ink` | `--surface` `#E8E8E8` | `16.3:1` | AA, AAA |
| `--ground` (on filled pill, selected size pill, bag badge) | `--ink` | `18.9:1` | AA, AAA |
| `--ink-soft` `#6B6B6B` | `--ground` | `5.0:1` | AA |
| `--ink-soft` | `--surface` / `--product-ground` `#E8E8E8` | `4.3:1` | **FAIL** for small text; AA Large only — see §4 |
| `--accent-ink` `#FFFFFF` | `--accent` `#080808` | `20.0:1` | AA, AAA |
| `--accent-ink` at 70% (footer paragraph, credits) | `--accent` | `9.8:1` | AA, AAA |
| `--accent-ink` at 60% (footer column headings, legal line) | `--accent` | `7.3:1` | AA |
| `--danger` `#B42318` | `--ground` | `6.2:1` | AA |
| `--danger-ink` `#FFFFFF` | `--danger` | `6.6:1` | AA |
| `--field-line` (resolves `#8C8C8C`) | `--ground` | `3.2:1` | AA (non-text boundary) |
| `--line` (resolves `#DBDBDB`) | `--ground` | `1.3:1` | **FAIL** as a control boundary; fine as a decorative rule — see §4 |

**Management area**

| Foreground | Background | Ratio | Passes |
|---|---|---|---|
| `--ink` `#0F172A` | `--ground` `#F8FAFC` | `17.1:1` | AA, AAA |
| `--ink` | `--surface` `#FFFFFF` | `17.9:1` | AA, AAA |
| `--ink-soft` `#64748B` | `--ground` | `4.5:1` | AA (at the limit) |
| `--ink-soft` | `--surface` | `4.8:1` | AA |
| `--accent-ink` `#FFFFFF` | `--accent` `#2563EB` | `5.2:1` | AA |
| `--accent` (link, focus border) | `--surface` | `5.2:1` | AA |
| white | `--danger` `#DC2626` | `4.8:1` | AA |
| `--danger` | `--ground` | `4.6:1` | AA |
| `--field-line` `#CBD5E1` | `--surface` | `1.5:1` | **FAIL** as a control boundary — see §4 |

### 1.2 Spacing

Base unit: **4px** — Tailwind's spacing scale *is* the `--space-*` scale (`p-4` is `--space-4`). Every margin, padding and gap is a
multiple of 4px except the half-steps listed in §4.

| Token | Value | Where it shows up |
|---|---|---|
| `--space-1` | `4px` | `gap-1` between a caption's name and price when stacked; `pt-1` above a card's note |
| `--space-2` | `8px` | Gap between size pills (`gap-2`), gallery thumbnails, icon + label |
| `--space-3` | `12px` | Caption row top margin (`mt-3`), search-field gap, gallery stack gap |
| `--space-4` | `16px` | Form-field gap (`gap-4`), checkout row padding (`py-4`), dock offset from the screen edge |
| `--space-6` | `24px` | **Grid gap between product cards** (`gap-[var(--space-6)]` in `Grid`), cart drawer side padding (`px-6`) |
| `--space-8` | `32px` | 12-column gap from 1024px (`lg:gap-8`), top of a section on desktop |
| `--space-12` | `48px` | Footer's vertical padding, bottom of the hero section |
| `--space-16` | `64px` | Gap above the footer (`mt-16`) |

Other 4px-grid values the design writes through Tailwind and uses as the band rhythm: `20px` (`py-5`, a list row and a promise cell), `28px`
(`py-7`, the band that holds a heading between two rules), `40px` (`py-10`, a section and the gap between 12-column blocks on a phone),
`56px` (`py-14`, a closing section). A heading band is always **rule · `py-7` heading · rule · `py-10` content**.

Sizes of fixed things: header `--header-h` = `64px`; dashboard header `56px` (`h-14`); dashboard sidebar `16rem`
(`18rem` as the phone sheet); contact dock buttons `56px`; header icon buttons `40px`; colour-dot hit area `36px`.

### 1.3 Typography

Font families (both tokens name Inter; loaded from Google Fonts in `index.html`, `Inter` variable 100–900, `display=swap`):

- Body: `var(--font-body)` = `"Inter"`, fallback `sans-serif` (Tailwind `font-body`)
- Headings: `var(--font-display)` = `"Inter"`, fallback `system-ui, sans-serif` (Tailwind `font-display`)
- Mono: none — the design has no monospace text; figures use `tabular-nums`

Base: `body` is `17px / 1.6`, weight 400 (unset), `-webkit-font-smoothing: antialiased`; `h1–h3` are `--weight-display` 500, `-0.02em`, line-height `1.04`,
`text-wrap: balance`. Inter 500 is the heaviest weight in the public site: headings are confident, not heavy.

The scale below is what the design draws. Sizes written `[…]` are arbitrary values (see §4 — they repeat, so they are the de-facto scale).

| Role | Size | Line height | Weight | Letter-spacing | Case | Used for |
|---|---|---|---|---|---|---|
| Display / h1 (`--size-h1`) | `clamp(2.75rem, 5vw, 4.5rem)` = 44px → **72px at 1440** | `1.02` (`leading-[1.02]`; `1.04` default) | 500 | `-0.02em` | Sentence | The one `h1` of every page; the large figures on Shipping and Returns (`leading-none tabular-nums tracking-tight`) |
| h2 (`--size-h2`) | `clamp(1.5rem, 2.2vw, 2rem)` = 24px → **32px** | `1.1` | 500 | `-0.02em` | Sentence | Section headings inside a page (lines index, steps, rules, questions) |
| Band heading (h2) | `text-lg` 18px | `1.04` | 500 | `-0.02em` | Sentence | The heading held between two rules ("New in", "Summary") |
| Large number | `text-2xl` 24px | 1.33 | 400 | — | — | A pair's price on the home page, the order total (`tabular-nums`) |
| Menu link | `text-3xl` 30px (`text-2xl` for search lines) | 1.2 | 400 | `-0.025em` (`tracking-tight`) | Sentence | Mobile menu links, the lines index from `sm` |
| Lead / drawer title | `text-lg` 18px | 1.55 | 400 | — | Sentence | Drawer title (display face), contact "sent" confirmation, checkout step legend |
| Body / secondary | `text-sm` 14px | 1.43 | 400 | — | Sentence | Nearly all running text, nav links, form values (`text-base` 16px on a phone so a field never zooms), table cells |
| Caption row name | `[0.8125rem]` 13px | 1.54 | 500 | `0.1em` | UPPER | Under every tile: the shoe's name; cart and summary line names use 11px/`0.1em` |
| Label / eyebrow | `[0.6875rem]` 11px | 1.6 | 500 | `0.12em`–`0.14em` | UPPER | Eyebrow above the home headline, field labels, outline-pill labels, footer column headings |
| Button label (filled) | `text-xs` 12px | 1.33 | 500 | `0.14em` | UPPER | Add to bag, Check out, Place order, Send, Track |
| Small label | `text-xs` 12px | 1.33 | 400 | `0.05em` (`tracking-wider`) | UPPER | Size and colour group labels, filter group headings |
| Helper / error | `text-xs` 12px | 1.33 | 400 | — | Sentence | Stock line, field errors |
| Footer legal | `[0.8125rem]` 13px | 1.625 | 400 | — | Sentence | "Independent dealer" line, photo credits |
| Bag count | `[0.625rem]` 10px | 1.6 | 500 | — | — | The number on the bag icon (`tabular-nums`) |

Heading levels are used in order and never skipped for visual sizing: one `h1` per page; band headings and section headings are `h2`.

| Token | Value | Used for |
|---|---|---|
| `--weight-display` | `500` | h1–h3, the drawer title (via `font-display`) |
| body weight | `400` (browser default; `body` sets none) | Running text |
| label weight | `500` (`font-medium`) | Every uppercase label, caption name and button |
| `--tracking-display` | `-0.02em` | h1–h3 |
| `tracking-tight` | `-0.025em` | Large numbers, menu links, wordmark |
| letterspaced caps | `0.1em` caption, `0.12em` pill/label, `0.14em` filled button | Uppercase micro-labels |

Management area: the page title `h2` is `font-body text-2xl font-semibold tracking-tight` (600, not the display 500), the header `h1` is `text-base font-medium tracking-normal`,
panel titles `text-base`; figures in a `Stat` are `text-2xl font-semibold tabular-nums`. Same face (Inter), working sizes.

### 1.4 Radius, border, shadow, motion

**Radius.** Square tiles and panels, round controls.

| Token | Value | Used for |
|---|---|---|
| `--radius` | `0px` (dashboard: `6px`) | Product tiles, `Select`, quantity stepper, contact-form fields (bare `rounded`, which `index.css` maps to `--radius`) |
| `--radius-sm` | `0px` (dashboard: `4px`) | Checkout fields (`rounded-sm`), checkboxes, gallery thumbnails |
| `--radius-pill` | `999px` | Every button, outline pill, size pill, filter chip, menu button |
| `rounded-full` | `9999px` | Colour dots, order-step dots, bag-count badge, the two contact-dock buttons, the dashboard `Badge` |

The management kit also uses Tailwind's own steps `rounded-md` (6px), `rounded-lg` (8px) and `rounded-xl` (12px) — see §4.

**Border.** `--border-width` is 1px everywhere (`border`); the footer and order step line are 1px (`h-px`). Active/selected emphasis is a 1px `border-ink` or a 1px
`outline` offset 2px (colour dot, current order step) — never a thicker border, except the active tab's `border-b-2`.

**Shadow.** The public site has none on tiles, cards or panels: separation is a hairline and space. Shadows exist only on layers that float:

| Token / utility | Value | Used for |
|---|---|---|
| `shadow-xl` | Tailwind default | Cart drawer |
| `shadow-lg` | Tailwind default | Contact-dock buttons, `Select` listbox, kit dialogs, kit popovers and sheets |
| `--shadow-sm` | `0 1px 2px rgba(27,25,23,.06)` (dashboard `rgba(15,23,42,.06)`) | Defined in `theme.css`; the kit's `shadow-xs` / `shadow-sm` on cards, inputs and buttons |
| `--shadow-md` | `0 8px 24px -12px rgba(27,25,23,.25)` (dashboard `0 4px 12px -6px rgba(15,23,42,.18)`) | Defined in `theme.css`; the kit's `shadow-md` on menus |

**Motion.** The owner chose **lively** (`site.json` → `"motion": "lively"`, written to `<html data-motion="lively">`).

| Token | Value | Used for |
|---|---|---|
| `--duration-fast` | `120ms` | Defined; the design's hover colour changes use Tailwind's `transition-colors` (150ms) |
| `--duration-base` | `220ms` | Defined; kit dialogs and menus use `duration-200`; sheets `300ms` out / `500ms` in |
| `--ease-out` | `cubic-bezier(0.2, 0.7, 0.2, 1)` | Page arrival, product picture zoom, `EASE` in `motion/level.ts` |

Motion the design draws at *lively*:

| What | Behaviour |
|---|---|
| Page arrival | `main` fades up 10px over 360ms on every route change; sections arrive as they scroll in |
| Home headline | `SplitHeading`: the words rise in |
| "See this pair", "Add to bag" | `data-magnetic`: lean toward the pointer (fine pointer only) |
| New-in cards | `data-tilt`: tilt under the pointer |
| Catalogue count | `data-count`: counts up |
| Product card picture | Scales to `1.05` over 0.6s and swaps to the back picture (opacity 0.35s) under the pointer |
| Product page photograph | `data-lens`: looked into at 2× under the pointer |
| Disclosure `+` | Rotates 45° when open |
| Lines index arrow | Moves 4px right on row hover |

Motion respects `prefers-reduced-motion: reduce`: `movement()` in `motion/level.ts` returns `still`, so every arrival, tilt, magnet, count, lens and the
picture hover rules (which only match `data-motion="lively|showpiece"`) stop; state changes (colour, underline, open/closed) remain. Native scrolling; nothing is pinned
or scrubbed. The progress line and cursor dot are `showpiece`-only and are not drawn at *lively*.

### 1.5 Layout and breakpoints

Mobile first. Breakpoints are Tailwind's defaults; the design's real switch is **`lg` (1024px)**.

| Name | Min width | What changes |
|---|---|---|
| (base) | 0 | One column; menu and search open as full-screen panels; filters fold behind their headings; contact dock shown; product grids 2 across |
| `sm` | 640px | Order steps become one line; checkout two-up fields; product grids up to 3 across; lines index shows its year column |
| `md` | 768px | Footer in 3 columns |
| `lg` | 1024px | 12-column layouts (gap 32px); header shows the five links; filter rail is permanent at the left; contact dock hidden; `Bleed` photographs run off one edge only; product grids up to 4 across |
| `xl` | 1280px | Not used by a layout rule |

| Item | Value |
|---|---|
| Page measure | `--page` = `1200px` (`max-w-page`), centred, including the gutter |
| Gutter | `--gutter` = `clamp(20px, 5.5vw, 72px)` on both sides (`px-[var(--gutter)]`) |
| Bleed | `--bleed` = `max(var(--gutter), calc((100vw - var(--page)) / 2 + var(--gutter)))` — how far a photograph reaches past the measure to the screen edge |
| Columns | 12 from `lg`; typical splits 5/7, 6/6, 7/5, 8/4. Below `lg`, one column |
| Product grid | `Grid`: flex-wrap, `var(--space-6)` gap, orphan row centred; 2 across on a phone, ≤ 3 from `sm`, ≤ 4 (max 6 for a dense row) from `lg`. One tile ratio per grid |
| Footer | Full-width band, the only element that runs edge to edge in `--accent`; its content keeps the 1200px measure |
| Dock clearance | Below 1024px, while the dock is shown, `body` keeps `--dock-space` (`88px + safe-area`) clear at its foot |

Z-index scale (the only values the design uses):

| Layer | Value | Used by |
|---|---|---|
| Base | `0` | Page content |
| Dashboard sticky header, sidebar | `10` | `DashboardLayout` header, sidebar rail |
| Dropdown | `20` | `Select` listbox, sidebar rail handle |
| Modal backdrop, modal / drawer | `50` | Cart drawer and its overlay, kit dialogs/menus/popovers/tooltips, contact dock, motion progress line and cursor dot |
| Full-screen menu | `60` | `MobileMenu` (search and navigation panels) — above the dock |

The public header is not sticky.

## 2. Components

Where a component is a page-local pattern repeated verbatim across pages (the filled pill, the checkout field, the radio row), it is documented once here. Components that exist in `src/components`
but that no page draws (`VariantPicker`, `ProductTabs`) are not part of the design and are not documented.

### 2.1 Public site

#### 2.1.1 Rule (hairline)

**Purpose** — the only separator between bands; no shaded panel, card or box does this job.
**Anatomy** — `<hr>`, 1px, full content width.
**Tokens** — `--line`.

| State | Visual change | Tokens |
|---|---|---|
| Default | 1px line across the measure | `bg-line` |

**Accessibility** — decorative (`hr`).

#### 2.1.2 Logo

**Purpose** — the wordmark in the header and footer.
**Anatomy** — `[LogoMark 28px] [site name]`, display face `text-lg`, `tracking-tight`; the owner's own logo image (`h-8`) replaces both when `site.logo` is set. The mark draws in `currentColor`.

| State | Visual change | Tokens |
|---|---|---|
| Default | Mark and name in the current text colour (`--ink` in the header, `--accent-ink` in the footer) | `font-display` |

**Accessibility** — the header link carries `aria-label` = the site name.

#### 2.1.3 SiteHeader (with search and bag)

**Purpose** — wordmark left; five links, search and bag right; a hairline under all of it.
**Anatomy** — `[Logo] … [nav ×5 (≥lg)] [search button] [bag button + count] [menu toggle (<lg)]`, height `--header-h` 64px, then `Rule`. Search opens a line under the links from `lg` and a full-screen panel (with the lines to start from) below it.

| State | Visual change | Tokens |
|---|---|---|
| Default | Links at `text-sm`, icons at 20px in 40×40 buttons | `text-ink` |
| Hover (link) | Underline, offset 8px | `underline-offset-8` |
| Hover (icon button) | Opacity 70% | — |
| Current page (link) | Underlined, `aria-current="page"` | — |
| Search open | Search icon becomes a close icon (`aria-expanded`); the search line appears under the header with a hairline above it | `border-line` |
| Bag has items | A 16px round badge at the icon's top-right with the count, `tabular-nums` | `bg-ink`, `text-ground` |

**Accessibility** — nav landmarks `Main` and the search label; icon buttons have `aria-label`; the search field is `role="search"` and autofocuses when opened.

#### 2.1.4 MobileMenu and MenuToggle

**Purpose** — below 1024px, the navigation and the search as a solid panel over the whole screen under the header.
**Anatomy** — portal into `body`, `fixed` from `--header-h` to the bottom, `px-[var(--gutter)] py-10`, `gap-6`; navigation links at 30px, then a `Rule`, then help links at 16px in `--ink-soft`. The toggle is a 40×40 pill-radius button with a 22px menu / X icon.

| State | Visual change | Tokens |
|---|---|---|
| Closed | Not rendered | — |
| Open | Panel covers the page; the page does not scroll; Esc closes | `bg-ground` |
| Toggle default | Menu icon | `text-ink` |
| Toggle hover | Fill `--surface` | `hover:bg-surface` |
| Toggle open | X icon, `aria-expanded="true"` | — |

**Accessibility** — `role="dialog" aria-modal="true"`; Esc to close; the toggle's label reads `nav.menu.open` / `nav.menu.close`.

#### 2.1.5 OutlinePill (the one call to action)

**Purpose** — the single link-button of the site: "See this pair", "All shoes", "Keep shopping", "Track an order". Never filled.
**Anatomy** — `[label, 11px/500/uppercase/0.12em] [arrow icon 14px]`, `border border-ink`, `px-4 py-2`, `--radius-pill`.

| State | Visual change | Tokens |
|---|---|---|
| Default | 1px ink outline, ink text | `border-ink`, `text-ink` |
| Hover | Fills with ink, text becomes ground (`transition-colors`) | `bg-ink`, `text-ground` |

The same classes are used inline for the header's search "Go" (`py-1.5`), the catalogue's "Filters" toggle (below `lg`) and the empty-results action (`px-5 py-2.5`).

**Accessibility** — a real link; the arrow is decorative.

#### 2.1.6 Filled pill button

**Purpose** — the one *commit* action on a page: "Add to bag", "Check out" (bag and drawer), "Place order", "Send", "Track order". Only one per view.
**Anatomy** — `[icon 16px?] [label, 12px/500/uppercase/0.14em]`, `bg-ink text-ground`, `px-5/6 py-3.5`, `--radius-pill`; full width inside the summary column and drawer, `self-start` in forms.

| State | Visual change | Tokens |
|---|---|---|
| Default | Ink fill, ground text | `bg-ink`, `text-ground` |
| Hover | Opacity 85% | — |
| Disabled (while sending / placing) | Opacity 60%; label becomes the progress sentence ("Sending…", "Placing your order…") | — |

**Accessibility** — native `button`/`link`; the disabled state uses the `disabled` attribute.

#### 2.1.7 ProductImage (the tile)

**Purpose** — every shoe photograph: a fixed-ratio frame filled edge to edge by the photograph.
**Anatomy** — `div` on `--product-ground`, `aspect-ratio` by prop, `overflow-hidden`, `--radius`; the photograph (`object-cover` for `.jpg`/`.webp`; a `.png` is shown whole with an 8% margin); an optional second picture revealed on hover. No border, no shadow, no label inside the tile.

| Variant | Ratio | When to use |
|---|---|---|
| `1/1` | 800×800 | Cards, bag lines, summary lines, thumbnails |
| `4/5` | 800×1000 | Available; the gallery's default |
| `4/3` | 800×600 | Home lead pair, product gallery, order picture |

| State | Visual change | Tokens |
|---|---|---|
| Default | The photograph fills the tile | `--product-ground` |
| No photograph yet | Flat `<Picture>` block with a caption, drawn on the tile | `--product-ground`, `--product-ground-ink` |
| Hover (lively, on a card) | Picture scales to 1.05 (0.6s) and the back picture fades in (0.35s) | `--ease-out` |

**Accessibility** — `alt` = the shoe's name; the second picture is `alt=""`.

#### 2.1.8 ProductCard

**Purpose** — one shoe in a grid, as a link.
**Anatomy** — `[ProductImage 1/1] [caption row: NAME (13px caps, left) · price (tabular-nums, right)] [one soft line: "{line} · {n} sizes in stock"]`. The caption stacks name over price below a `13rem` container width (`@container`).

| State | Visual change | Tokens |
|---|---|---|
| Default | As above | `text-ink`, `text-ink-soft` |
| Hover | Name underlined (offset 4px); picture zoom and back picture (see 2.1.7) | — |
| Tilt (New-in row only) | `data-tilt` tilts the card under the pointer at lively | — |

**Accessibility** — the whole card is one link (`/shop/:id`); price is text, not colour-coded.

#### 2.1.9 Grid

**Purpose** — a grid of like things that never leaves a hole in its last row (see 1.5).
**Anatomy** — flex-wrap, `gap` = `--space-6`, centred last row, each child a basis-sized cell.

| State | Visual change | Tokens |
|---|---|---|
| Default | Column count chosen from the item count and `max` | `--space-6` |

#### 2.1.10 ColourDots

**Purpose** — the colourways of one model, each a link to that pair.
**Anatomy** — `[label, 12px caps ink-soft]` over a row of 14px circles filled with the colourway's swatch, each in a 36×36 link, `border border-line`.

| State | Visual change | Tokens |
|---|---|---|
| Default | Filled dot with a hairline edge | `border-line` |
| Hover | Scales to 110% | — |
| Current | A 1px ink ring offset 2px; `aria-current="page"` | `outline-ink` |

**Accessibility** — each link has `aria-label` = the colourway name and an `sr-only` name; a model with one colourway shows its single dot, ringed.

#### 2.1.11 SizePicker

**Purpose** — choose an EU size with the stock in it; the size chart is one tap away on the label line.
**Anatomy** — `[label, 12px caps] … [Size chart link, 12px underlined]` over a `radiogroup` of pills: 3 or 4 across on a phone, 5 from `sm`, the count chosen so no pill is alone in the last row. Pill: `border`, `px-2 py-2.5`, `text-sm tabular-nums`, `--radius-pill`.

| State | Visual change | Tokens |
|---|---|---|
| Default | `border-line`, ink text | `border-line`, `text-ink` |
| Hover | Border becomes ink | `border-ink` |
| Selected | Ink fill, ground text; `aria-checked="true"` | `bg-ink`, `text-ground` |
| Last pairs | 6px dot at the pill's top-right (danger on an unselected pill, ground on a selected one) | `bg-danger`, `bg-ground` |
| Sold out | Struck through, `--ink-soft`, not choosable (`disabled`, `cursor-not-allowed`) | `text-ink-soft`, `border-line` |

A stock line under the picker (`role="status"`, 12px, `--danger` when it names a last pair) says which sizes are low or gone.

**Accessibility** — `radiogroup` labelled by the label; each pill `role="radio"`.

#### 2.1.12 QuantityStepper

**Purpose** — how many: a number between a minus and a plus, never below 1 or above what there is.
**Anatomy** — `[− button] [number, aria-live] [+ button]`, `inline-flex`, `border border-line`, `--radius`; icons 16px in `p-2.5` buttons.

| State | Visual change | Tokens |
|---|---|---|
| Default | Ink icons and number | `text-ink`, `border-line` |
| Disabled (at 1 for −, at stock for +) | Icon in `--ink-soft`, button `disabled` | `text-ink-soft` |

**Accessibility** — `role="group" aria-label="Quantity"`; buttons labelled "One fewer" / "One more"; the number is `aria-live="polite"`.

#### 2.1.13 Disclosure (and FaqRows)

**Purpose** — the questions a buyer asks next, behind a row: Details, Payment options, Delivery & returns on a shoe; the questions on policy pages.
**Anatomy** — `<details>` rows between hairlines: `[label, 14px] … [+ 18px, ink-soft]`, `py-3.5`; content `text-sm ink-soft`, `pb-5`. A row with no content is not drawn.

| State | Visual change | Tokens |
|---|---|---|
| Closed | Label and `+` | `border-line`, `text-ink-soft` |
| Open | `+` rotates 45° (becomes ×); content shown | — |

**Accessibility** — native `details/summary`; the `+` is `aria-hidden`.

#### 2.1.14 FilterRail

**Purpose** — narrow the catalogue by line, who it is for, colour, price band and EU size; the choices live in the address bar.
**Anatomy** — `aside` of `details` groups, each `[group heading 12px caps ink-soft] [selected count?]` over checkbox rows (16px checkbox, `--field-line` edge, label `text-sm`), a `border-b border-line` under each group; "Clear all" as an underlined text button once filtering. A group with fewer than two options is not shown.

| State | Visual change | Tokens |
|---|---|---|
| Default | All groups open at `lg`; their headings are not clickable | `border-line` |
| Folded (below `lg`) | Groups collapse behind their headings so a phone opens on the goods | — |
| Option checked | Checkbox filled in `--accent`; the group heading shows the count in ink | `accent-[var(--accent)]` |
| Filtering | "Clear all" appears | `text-ink-soft`, hover `text-ink` |

**Accessibility** — `aria-label="Filters"`; each option is a real `label` + `checkbox`.

#### 2.1.15 Select

**Purpose** — every choice from a list (the catalogue's sort). Never a bare `<select>`.
**Anatomy** — `[button: chosen label, 14px] [chevron 12px]`, `border-field bg-surface px-3 py-2 --radius`; the list is a `listbox` under it, `z-20`, `shadow-lg`, `max-h-64`.

| State | Visual change | Tokens |
|---|---|---|
| Default | Chosen label in ink | `border-field`, `bg-surface` |
| Placeholder | Label in `--ink-soft` | `text-ink-soft` |
| Open | Listbox below, 4px gap | `shadow-lg` |
| Option active (hover / arrow keys) | Ink fill, white text | `bg-accent`, `text-accent-ink` |

**Accessibility** — `role="combobox"` with `aria-controls`/`aria-expanded`; arrows, Home, End, Enter, Space, Escape and type-ahead; closes on outside click or scroll.

#### 2.1.16 Text field and label (checkout and contact)

**Purpose** — one text input, phone input or textarea with its label.
**Anatomy** — `[label, 11px caps ink-soft, mb 6px]` over a field: `w-full border border-field bg-ground px-3 py-3`, `text-base` (16px on a phone) → `text-sm` from `sm`, placeholder `--ink-soft`. The contact form's fields are `px-4 py-3` with `text-ink` and a 14px `--ink-soft` label above.

| State | Visual change | Tokens |
|---|---|---|
| Default | `--field-line` edge on `--ground` | `border-field`, `bg-ground` |
| Focus (checkout) | Border becomes ink; the browser outline is removed | `focus:border-ink` |
| Error | Not on the field: the page lists what is wrong under the summary, 12px `--danger` with an alert icon (`role="alert"`) | `text-danger` |

**Accessibility** — every field has a visible `label` linked by `htmlFor`; phone fields use `type="tel"` and `inputMode="tel"`; `autoComplete` set for name, phone, address, city.

#### 2.1.17 Radio row (delivery and payment)

**Purpose** — delivery speed and payment method as marked options (COD, bank transfer, VNPay, MoMo).
**Anatomy** — hairline-separated `label` rows: `[16px radio dot] [icon 18px] [name · detail 14px over note 14px ink-soft] [fee, tabular-nums, right]`; the real `input[type=radio]` is `sr-only`.

| State | Visual change | Tokens |
|---|---|---|
| Default | Hollow dot with `--field-line` ring | `border-field` |
| Selected | Dot ringed and filled in ink | `border-ink`, `bg-ink` |

**Accessibility** — the group is `role="radiogroup"` in a `fieldset` with a numbered `legend`; the whole row is the click target.

#### 2.1.18 ContactForm

**Purpose** — the message form: name, phone or email, message; what is sent reaches the owner in Agent Team.
**Anatomy** — fields (2.1.16) in a `flex-col gap-4`, a hidden honeypot field `website`, a failure line, an offline note, the filled pill (2.1.6, label "Send").

| State | Visual change | Tokens |
|---|---|---|
| Idle | Empty form | — |
| Sending | Button disabled at 60% opacity, label "Sending…" (idle label "Send message") | — |
| Sent | The form is replaced by the thank-you sentence at 18px | `text-lg` |
| Failed | `role="alert"` line (14px ink) with the problem above the button | `text-ink` |
| Offline | A 14px `--ink-soft` note ("This form is not connected yet — please try again later.") sits above the button | `text-ink-soft` |

#### 2.1.19 CartDrawer

**Purpose** — the bag as a panel from the right edge, so adding to it does not take the visitor away from the page.
**Anatomy** — overlay (`bg-black/40`) + panel `max-w-md`, `bg-ground`, `shadow-xl`, `z-50`: `[title (display, 18px) … close icon]` · hairline · lines (`[80px tile] [name + variant · price] [QuantityStepper · remove icon]`, `divide-y divide-line`) · hairline · `[Total · price 18px]` + filled pill "Check out".

| State | Visual change | Tokens |
|---|---|---|
| Default (has lines) | As above | — |
| Empty | One centred sentence in `--ink-soft`, no total and no button | `text-ink-soft` |
| Icon buttons hover | Remove and close icons go from `--ink-soft` to ink | — |

**Accessibility** — Radix dialog: focus trap, scroll lock, Esc; close and remove buttons have `aria-label`.

#### 2.1.20 OrderSteps

**Purpose** — where an order is, as one line: Placed · Confirmed · Packed · Shipped · Delivered.
**Anatomy** — `<ol>`: a 5-column hairline with a 10px dot per step at `sm`+, a vertical list below it; label 11–12px caps.

| State | Visual change | Tokens |
|---|---|---|
| Done | Filled ink dot, ink label, ink line segment | `bg-ink`, `text-ink` |
| Current | As done, plus a 1px ink ring offset 2px; `aria-current="step"` | `outline-ink` |
| Upcoming | Hollow dot with `--field-line` edge, `--ink-soft` label, `--line` segment | `border-field`, `text-ink-soft`, `bg-line` |
| Cancelled | Replaces the line: an alert icon in `--danger` and the sentence, between two hairlines | `text-danger` |

#### 2.1.21 PromiseRow

**Purpose** — what the shop promises, in four cells on a hairline: delivery, size exchange, cash on delivery, genuine pairs.
**Anatomy** — `ul` of cells `[icon 20px] [title 14px ink / line 14px ink-soft]`, `py-5`, `border-b border-line` under a top hairline; 2 across, 4 from `lg` (compact: 1 → 2 across).

| State | Visual change | Tokens |
|---|---|---|
| Default | The shop's own terms filled into the sentences | `text-ink`, `text-ink-soft` |

#### 2.1.22 ProductGallery

**Purpose** — the large picture of one shoe and the others under it to switch to.
**Anatomy** — `ProductImage` (4:3 on the shoe page) in a lens wrapper, then a row of 64px (80px from `sm`) 1:1 thumbnails, `--radius-sm`, 1px edge. One picture draws no thumbnail strip.

| State | Visual change | Tokens |
|---|---|---|
| Thumbnail default | 70% opacity, `--line` edge | `border-line` |
| Thumbnail hover | Full opacity | — |
| Thumbnail selected | Full ink edge; `aria-pressed="true"` | `border-ink` |

#### 2.1.23 SpecList and Rating

**SpecList** — label/value table of a shoe's specification (upper, lining, outsole, style code, colourway) and of contact details: `dl`, 2 columns (`minmax(7rem, auto)` / `1fr`), `text-sm`, a hairline above every row, `py-3`; label `--ink-soft`, value ink; a row with no value is not drawn. State: `default`.
**Rating** — five 16px stars, the rounded value filled in `--accent`, the rest outlined in `--line`; read as "4 out of 5" (`role="img"`). State: `default`.

#### 2.1.24 Bleed and HeadPicture

**Purpose** — what runs off the edge of the screen. Words keep the 1200px measure; the photograph beside a page's title (and the panel where a visitor acts) does not.
**Anatomy** — `Bleed` widens by negative margins (`--bleed`): `side="right"` / `"left"` run to one edge from 1024px and to both below it; `ground` paints `--surface` behind it and pads the inside by the same reach. `HeadPicture` is a `Bleed` holding one photograph, 16:9 on a phone, at least 14rem tall beside the title from `lg`.

| State | Visual change | Tokens |
|---|---|---|
| Default | Photograph meets the screen edge; no text is laid over it | `--surface` |

#### 2.1.25 EmptyBag

**Purpose** — an empty bag or an empty checkout is a page, not a blank.
**Anatomy** — the sentence at display size (`--size-h1`, ≤ 16ch), one `text-sm` line saying what to do, an `OutlinePill` "Keep shopping", then a rule, a band heading and the New-in row of up to four `ProductCard`s.

| State | Visual change | Tokens |
|---|---|---|
| Default | As above | `text-ink`, `text-ink-soft` |

#### 2.1.26 ContactDock

**Purpose** — Call and Zalo within reach on a phone. Hidden from 1024px.
**Anatomy** — two 56px round buttons fixed at `bottom: 1rem + safe-area`, `right: 16px`, `gap-3`, `z-50`, `shadow-lg`: Zalo (`--surface`, ink text "Zalo", ring `--line`) and Call (`--accent` fill, phone icon). A button shows only when its number is in `content.json`; with neither, nothing shows.

| State | Visual change | Tokens |
|---|---|---|
| Default | Two buttons; the gap between them lets taps through | `bg-surface`, `bg-accent` |

**Accessibility** — each has `aria-label`; the Zalo link opens `zalo.me/<number>` in a new tab; the phone link is `tel:`.

#### 2.1.27 SiteFooter

**Purpose** — every page ends on the template's own colour.
**Anatomy** — full-width `--accent` band: wordmark + one line (14px at 70%) · three link columns (heading 11px caps at 60%; links 14px, underline on hover) · hairline at 15% · the "independent dealer, not adidas" line and the photo credits (13px).

| State | Visual change | Tokens |
|---|---|---|
| Default | Ground-coloured type on ink | `bg-accent`, `text-accent-ink` |
| Link hover | Underline, offset 4px | — |

#### 2.1.28 Icon

**Purpose** — the one icon set: `lucide-react`, stroke `1.75`, default 20px, `shrink-0`, named by meaning ("shipping" is a truck everywhere). Decorative (`aria-hidden`) unless given a `label`. Sizes used: 14, 16, 18, 20, 22 (menu button).

| State | Visual change | Tokens |
|---|---|---|
| Default | Draws in `currentColor` | — |

### 2.2 Management area (`/admin`, scope `[data-dashboard]`)

An application, not the brand: a menu down the left, no marketing header, no hero, no footer, dense rather than airy. It is the same in every project and a design does not restyle it. Built on the vendored shadcn/ui kit in `src/components/ui`; the project's wrappers are in `src/components/admin`. Colours are §1.1's management table; radius `6px` / `4px`; Inter at working sizes.

#### 2.2.1 DashboardLayout

**Purpose** — the frame of every management page.
**Anatomy** — sidebar (`16rem`; `18rem` as a sheet on a phone): `[site name, 14px/600]` · grouped nav (`[group label]` + rows) · footer `[32px initial tile] [name over role, 14px/12px] [sign-out icon]`. Inset: sticky header `56px` `[sidebar trigger] [divider] [page title, 16px/500] … [actions]` on `--surface` with a bottom hairline; `main` is `p-4` (`p-6` from `lg`) with `gap-4` (`gap-6`).

| State | Visual change | Tokens |
|---|---|---|
| Nav row default | 14px label | `text-ink` |
| Nav row hover | `--wash` fill | `bg-wash` |
| Nav row active | `--wash` fill with `aria-current="page"`; a count badge may sit at the right | `bg-wash` |
| Sidebar collapsed (off-canvas) | Slides out; the trigger in the header reopens it | — |
| Sign-out hover | Fill `--ground`, icon goes to ink | `bg-ground` |

**Accessibility** — nav landmark "Sections"; the sign-out icon button has `aria-label`.

#### 2.2.2 PageHeader, Panel, Stat

**PageHeader** — `h2` `text-2xl` `font-semibold` `tracking-tight` (truncated) with an optional 14px `--ink-soft` count under it; actions at the right, wrapping under it on a phone. State: `default`.
**Panel** — the box for related things: `Card` (`rounded-xl border bg-surface shadow-xs`, `py-5`), title `text-base`, optional description in `--ink-soft`, content `px-5`, optional footer under a hairline. State: `default`.
**Stat** — the figure at the top of the overview: label 14px `--ink-soft` over the value `text-2xl` `font-semibold` `tabular-nums`, an optional 16px icon at the top-right and a note under. Four to a row. When it links (`to`): hover fills `--ground` (`transition-colors`). States: `default`, `hover` (linked only).

#### 2.2.3 Button

**Purpose** — every action in the management area. Four kinds, because five would say nothing about which one matters.
**Anatomy** — `[icon 16px?] [label, 14px/500]`, `h-8`, `px-3` (`px-2.5` with an icon), `rounded-md`.

| Variant (`kind`) | Tokens | When to use |
|---|---|---|
| `primary` | `bg-accent` `text-accent-ink` | The one action of a view (Save, Add) |
| `default` | `border bg-surface` `shadow-xs`, ink text | Secondary actions |
| `danger` | `bg-danger` `text-white` | Delete and other destructive actions — never drawn as `default` |
| `quiet` | transparent, ink text | Tertiary, in-row actions |

| State | Visual change | Tokens |
|---|---|---|
| Default | As the variant | — |
| Hover | `primary` → 90% accent; `default`/`quiet` → `--wash` fill; `danger` → 90% danger | `bg-wash` |
| Focus (keyboard) | 3px ring at 50% accent (`ring-accent/50`) and an accent border; the browser outline removed | `focus-visible:ring-[3px]` |
| Invalid | Danger border and a 20% danger ring | `aria-invalid:border-danger` |
| Disabled | 50% opacity, no pointer events | `disabled:opacity-50` |

#### 2.2.4 Field, Input, Textarea, Select

**Purpose** — a labelled control and what went wrong with it. A control without a label is never drawn.
**Anatomy** — `[label, 14px/500]` over the control (`h-9` input, `rounded-md`, `border-field`, `bg-surface`, `px-3`; textarea 4 rows), then a hint (`text-xs` `--ink-soft`) or an error (`text-xs` `--danger`). A `group` field names a control made of several (a picture with its buttons, a row of tags).

| State | Visual change | Tokens |
|---|---|---|
| Default | `--field-line` edge on `--surface`, placeholder `--ink-soft` | `border-field` |
| Focus | Accent border and 3px ring at 50% | `ring-accent/50` |
| Invalid | Danger border, 20% danger ring, the error line under the control | `text-danger` |
| Disabled | 50% opacity, `not-allowed` | `disabled:opacity-50` |

The kit's `Select` (Radix) is the control for lists here, with the same four states plus an open listbox (`z-50`, `rounded-md`, `shadow-md`) whose active option fills `--wash`.

#### 2.2.5 Badge

**Purpose** — a state as a word with a mark, never a bare colour.
**Anatomy** — outline pill `rounded-full border bg-surface px-2 py-0.5 text-xs`, `[6px dot] [word]`.

| Tone | Dot | Meaning |
|---|---|---|
| `neutral` | `--ink-soft` | Default state |
| `good` | `--accent` | Live, paid, delivered |
| `warn` | `--ink` | Needs attention |
| `off` | `--line` | Hidden, draft |
| `danger` | `--danger` | Cancelled, failed, out of stock |

State: `default` (each tone).

#### 2.2.6 Empty

**Purpose** — a list with nothing in it is the state every account has on day one; it says what is missing and what to do next.
**Anatomy** — dashed `--line` box (`rounded-lg`, `py-12`): `[40px icon tile on --ground]` · title 14px/500 · note 14px `--ink-soft` (max 24rem) · optional action.

| State | Visual change | Tokens |
|---|---|---|
| Default | Centred, dashed border | `border-dashed border-line` |

#### 2.2.7 Notice

**Purpose** — an inline message about the page: info, done or alert, with an icon and an optional close.
**Anatomy** — `role="status"`, `rounded-md border bg-surface px-3 py-2 text-sm`, `[icon 20px] [message] [close?]`.

| State | Visual change | Tokens |
|---|---|---|
| Default | Icon by tone (`info`, `done`, `alert`) | `border-line` |

#### 2.2.8 Dialogs, row menu, toggle and sections

| Part | Anatomy | States the design draws |
|---|---|---|
| `Confirm` / `FormDialog` | Radix dialog on a 50% black overlay, centred panel `rounded-lg border bg-surface p-6 shadow-lg`, `max-w-lg` from `sm`, title + description + footer buttons | Closed; open (fade and zoom 95% → 100%, 200ms); close icon hover (opacity 70% → 100%) and focus (2px accent ring) |
| `RowMenu` | The actions on a row behind one "more" icon button; safe actions first, destructive ones after a separator in `--danger` | Closed; open; item focus (`--wash` fill); destructive item focus (10% danger fill); disabled item (50%) |
| `Toggle` | On or off as a switch (`2rem × 1.15rem`) with a bordered label row (`rounded-md border bg-surface px-3 py-2.5`) | Off (`--field-line` track); on (`--accent` track); focus (3px ring); disabled (50%) |
| `Sections` | Tabs for the areas of one record, 14px/500 | Default (60% ink); hover (ink); active (ink); focus (ring); disabled (50%) |
| `Breadcrumb` | Where you are when a record sits under a list; the last crumb is the page | Link default; link hover (ink); current |
| `DataTable` | Rows inside `rounded-lg border border-line bg-surface shadow-xs`, with the kit's `Table` | Default |

## 3. Content and formatting

- **Voice.** An independent dealer talking plainly to a shopper, often on a phone: short sentences, concrete (sizes, days, prices), reassuring about size, delivery and returns ("Nothing in your bag yet.", "No pair matches that."). English throughout; shoe line names and model names as adidas writes them.
- **Where words live.** Every word is in `src/content.json`, addressed by dot path (`<T k="…">`, `useContent`), so the owner edits it in the Studio. A component never carries a sentence.
- **Currency.** Vietnamese dong, through `formatPrice` only: `3.290.000 ₫` (locale `vi-VN`, symbol after, dot thousands separator, no decimals); prices use `tabular-nums`. A price is never typed or built with `${price} ₫`.
- **Dates and times.** Through `formatDate` / `formatTime` only: `16/10/2026` (short, the default), `16 Oct 2026` (medium), `Friday, 16 October 2026` (long); time `14:30` (24-hour).
- **Sizes.** EU, written with thirds as the trade does: `36`, `36 2/3`, `37 1/3`; a range as `EU 36 – 41 1/3`.
- **Capitalisation.** Sentence case in `content.json` for everything; the design applies capitals itself with CSS (`uppercase` plus letter-spacing) to captions, labels, pill and button text — never type words in capitals. Headings are sentence case.
- **Numbers as type.** Policy figures (days, prices, counts) are set at `--size-h1` with `tabular-nums` and stand alone on a hairline.
- **Empty states.** Say what is missing, then give the next step: an empty bag shows the sentence, a line saying what to do, "Keep shopping" and four new pairs; an empty search says "No pair matches that." with the way to clear the filters. Never a blank area.
- **Errors.** Name the problem and what to do, in the shop's voice: "Fill in your name, street address and city.", "Check the phone number — we text you when the pair is on its way.", "We can't find that order." Errors are never colour alone: they carry an alert icon and a sentence.
- **Confirmation.** The answer first, then the detail: an order page opens on "Your order is on its way." / "Thank you — order placed.", then the number and date; a sent message says "Thank you — we have your message and will answer the same working day."
- **Imagery.** Every shoe is a photograph on the pale grey seamless, one shoe per frame, in a square-cornered tile with no border, shadow or label; colour appears only inside the photographs.

## 4. Known deviations

Places where the approved design does not follow its own rules, the contrast thresholds, or the anti-patterns in `references/ai-defaults.md`. Recorded, not fixed — the owner approved what they approved.

| Where | Deviation | Why it stands | Follow-up |
|---|---|---|---|
| Body type size | `index.css` sets `body` to `17px`; `design/moodboard.md` says body is 15px; components set `text-sm` (14px) on almost all text, so the 17px rarely applies | The Hairline template's sizes are set per element | Treat `text-sm` 14px as the body size in new work; reconcile the moodboard line with the scaffold |
| `--ink-soft` `#6B6B6B` on `--surface` / `--product-ground` `#E8E8E8` | `4.3:1`, below 4.5:1 for small text: the placeholder caption of a product with no photograph, and any `--ink-soft` text inside a `Bleed ground` or catalogue search/sort band | The moodboard's rule is that the palette is the template's, untouched | If the owner agrees, set `--product-ground-ink` (and `--ink-soft`) to `#666666` — `4.7:1` on `#E8E8E8`, `5.4:1` on `#F8F8F8` |
| `--line` as the edge of a control | Unselected size pills, quantity stepper, gallery thumbnails and colour-dot edges use `--line` (`1.3:1`); a control's boundary needs 3:1. Selected and hover states go to ink | `--line` is the hairline between bands and is meant to be nearly invisible | Use `--field-line` (`3.2:1`) for the edge of an unselected control |
| Dashboard `--field-line` `#CBD5E1` | `1.5:1` on a `#FFFFFF` field; `theme.css` itself asks for ≥ 3:1 on the public site | The management scope is fixed by the scaffold and is the same in every project | Scaffold change (TL): a field edge of at least 3:1, e.g. `#64748B` (`4.8:1`) |
| Focus indicators, public site | No designed focus ring. The header search field is `outline-none` with nothing in its place; checkout fields swap the outline for a border colour change (`#8C8C8C` → `#080808`); size pills, outline pills, the `Select` button, links and checkboxes fall back to the browser outline. The management kit has a 3px accent ring | The template draws none | One `focus-visible` rule in the base layer (2px ink outline, 2px offset) as a scaffold change by TL, with the owner's agreement |
| Touch targets | Several controls are under 44×44px: outline pills ≈ 34px high, quantity stepper buttons 36px, colour-dot links 36px, header icon buttons 40px, size pills ≈ 40px, cart-drawer close and remove ≈ 28–32px. The site is mostly used on a phone | Pills are drawn small by the template; hit areas are padded in a few places only | Pad hit areas to 44px without changing what is drawn |
| Arbitrary values in components | Repeated text sizes `text-[0.625rem]`, `text-[0.6875rem]`, `text-[0.8125rem]`, tracking `tracking-[0.1em]` / `[0.12em]` / `[0.14em]`, `leading-[1.02]`, `rounded-[var(--radius-pill)]` — they are the de-facto scale (§1.3) but have no tokens | The lint refuses arbitrary colours, not sizes | TL adds `--text-label`, `--tracking-caps` and a pill-radius utility name to `theme.css` |
| Half-step spacing | `py-3.5` (14px) on every filled pill and list row, `py-2.5` (10px), `gap-2.5` (10px), `py-1.5` / `mb-1.5` (6px), `size-3.5` (14px dot) are off the 4px grid | Optical sizing of pills and rows | Keep as drawn; do not extend to new components |
| Tokens defined but not used by name | `--duration-fast`, `--duration-base`, `--ease-out` (the cubic-bezier is written out in CSS and `motion/level.ts`), `--shadow-sm`, `--shadow-md` (the kit uses Tailwind's `shadow-*`), and most `--space-*` (Tailwind's spacing scale is used instead) | They exist so a story can name a value | Leave |
| Raw colour in components | `bg-black/40` on the cart overlay, `bg-black/50` on kit overlays, `text-white` on the kit's danger button and badge | A scrim over the page has no token | Leave |
| Radius scale | The kit uses Tailwind's own `rounded-md` (6px), `rounded-lg` (8px), `rounded-xl` (12px); only `--radius` and `--radius-sm` are tokens. A `Panel` is `rounded-xl` (12px) beside the dashboard's `--radius` of 6px | Vendored shadcn/ui; the management area is fixed | Leave |
| Buttons as pills (ai-defaults §3, "pills only for pills") | Every button, outline pill, size button and filter chip on the public site is `--radius-pill`; tiles and panels are square | The Hairline template and the moodboard ("full pills on every button, chip and the size buttons") set it; the square tiles keep the structure readable | Leave |
| Hover-only affordances (ai-defaults, additional) | The back picture of a shoe on a card and the 2× lens on the product photograph appear only under a fine pointer at *lively* | Both are enhancements: every picture is also reachable on the product page's gallery on touch | Leave |
| Strings in components | A few labels are written in the component, not `content.json`: "Quantity", "One fewer", "One more", "Close", "Remove {name}", "Main", "Help", "Filters", "More pictures", "Picture n of m", "Sign out" — the owner cannot edit them in the Studio | The site is English throughout | Move to `content.json` if the owner wants to edit them |
| Loading and error views | The design draws no loading or error view for any data-driven screen (the grid, an order, tracking, the dashboard lists): it renders from local records. The next story, "Site data from the database", makes these screens asynchronous | Nothing was drawn, so nothing is documented here | Open design question for that story — not specified here, to avoid inventing it |

**Checked and clean** against `references/ai-defaults.md`: no purple or indigo (the public palette is monochrome by design; the management area is slate with blue-600 `#2563EB`); no gradients anywhere; no shadow on a resting tile, card or panel on the public site (shadows only on the drawer, dock and listbox); no emoji — one icon set (`lucide-react`, stroke 1.75); no text laid over a photograph, so no scrim is needed; realistic copy of realistic length (real model names, prices in dong, EU sizes in thirds); empty states say what is missing and what to do; padding is proportional (generous on public bands, `p-4`–`p-6` in the dense management area).

## 5. Change log

| Date | Change | Design PR |
|---|---|---|
| 2026-11-25 | First version, extracted from the approved design | #1 |
