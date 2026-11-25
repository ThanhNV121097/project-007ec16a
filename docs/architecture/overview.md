# Architecture — Đôi

Shape: **fullstack** — an online adidas dealer: a customer storefront, a dealer admin area, a Go API and PostgreSQL.
Requirements are in `docs/{accounts,admin,catalogue,data,orders,payments,site}/SRS.md`; the owner's lines are traced in
`docs/brief-coverage.md`. This file fixes the stack, the layout and the conventions every story builds on. Table and
endpoint design is NOT here: `docs/architecture/erd.md` and `services.md` are written next, story by story.

## 1. Stack and the decisions behind it

| Part | Choice | Rejected, and the cost of the choice |
|---|---|---|
| Frontend | Vite 6 + React 19 + TypeScript + Tailwind 4, react-router 6 — the approved design in `code/frontend/`, built to static files and served by nginx on :3000 | *Default, not a decision.* Next.js is not in this product |
| Backend | Go (`go 1.26.0` in go.mod, matching the committed `golang:1.26-alpine` builder), stdlib `net/http` with method+path patterns (`GET /v1/{collection}/{id}`) | chi / gin: one more dependency for what Go 1.22 patterns already do. Cost: path values are read with `r.PathValue` and middleware is composed by hand (`internal/httpx`) |
| Database | PostgreSQL 16, driver **pgx v5** (`pgxpool`), plain parameterised SQL beside each handler | `database/sql` + lib/pq (pq is in maintenance); an ORM or sqlc (codegen step CI does not run, and the data story's generic list needs allow-listed column names an ORM hides). Cost: hand-written SQL |
| Migrations | Own runner, `internal/db/migrate.go`, SQL embedded in the binary, applied on boot before listening, tracked in `schema_migrations`, one advisory lock | golang-migrate / goose: a dependency (or a CLI in the image) for ~150 lines, and neither is wired to "apply on boot". Cost: we own the runner (tested in `migrate_test.go`) and the server never runs `.down.sql` |
| Sessions | Opaque random token in an `HttpOnly` cookie, session row in Postgres (7 days) | JWT: sign-out must kill the cookie (ADMIN-002 AC-11) and a switched-off staff account must stop working at once; a stateless token can do neither. Cost: one lookup per request |
| Passwords | argon2id (`golang.org/x/crypto/argon2`), OWASP minimum parameters (19 MiB, t=2, p=1), per-password salt | bcrypt: refuses input over 72 bytes and the SRS allows 100 characters |
| Picture uploads | Stored **in Postgres** (`bytea`, ≤ 5 MB, JPEG/PNG/WebP by content), served by `GET /v1/uploads/{name}` with long cache headers | Disk or a volume: the backend container is `read_only: true` with only `/tmp` writable and the committed compose/deploy files give it no volume. S3: no account in the brief. Cost: database grows; fine at shop scale |
| Search | Postgres: accent-folded `ILIKE` on words (`unaccent`, or a folded column the data story maintains — the ERD decides) | Elasticsearch / Meilisearch: a service to run for a few dozen shoes. Cost: query is built in Go from an allow-list of fields |
| Background work | In-process ticker for the 30-minute unpaid-order expiry (PAY-003); idempotent under row locks | A queue or cron container: one backend instance, nothing else to run. Cost: if two instances ever run, the row lock keeps it correct |
| Rate limits | Failure counters in a Postgres table per email and per address (5 per 15 min) | In memory: a restart would reset the lock. Redis: not in the stack |
| Money and ids | VND as `bigint`, never a float. Catalogue ids are the **text slugs the JSON used** (`samba-og-w-core-black`, `doi-24817`) | uuid ids: the routes, the seed and the SRS acceptance criteria already name these ids |

Not decisions, only defaults: Postgres for data, Tailwind tokens for styling, `docker compose` to run.

## 2. Runtime topology

```
browser ─► edge proxy ─┬─ /api/*  → backend:8080   prefix STRIPPED: /api/v1/products reaches Go as /v1/products
                       └─ /*      → frontend:3000  nginx, dist/ with SPA fallback to index.html
backend ─► PostgreSQL 16   (compose service `postgres`, profile `local`; a deployment uses the shared server)
VNPay   ─► backend         the payment notice (IPN) arrives from outside; the backend never polls VNPay
```

- **Backend routes carry no `/api` prefix** and `services.md` writes them without it. The browser's base is
  `import.meta.env.VITE_API_URL ?? "/api"`, so cookies stay same-origin and no CORS is needed in a deployment.
- The nginx in the frontend image has no `/api` proxy (committed file, not ours). So `docker compose up` on its own serves
  the pages on :3000 but not a working `/api`; the edge proxy of a deployment and of `run_project` supplies it.
- Contact form: `src/site.ts` posts to Agent Team's own API with the key in `src/site.json`; the owner reads messages
  there. The backend has no endpoint and no table for it.
- Compose hands the backend only `PORT` and `DATABASE_URL`. The owner's secrets reach it through `.env.deploy` in a
  deployment and the console in `run_project`. Without them the server still starts: sign-in is refused, payment answers 503.
- `NEXT_PUBLIC_API_URL` and `API_ORIGIN` in `docker-compose.yml` / `deploy.yml` are inherited from the generic scaffold and
  do nothing here: Vite reads only `VITE_*`, at build time, and the frontend is static.

## 3. Repository layout

```
docs/{module}/SRS.md, stories/, test-cases/     PM / Test      docs/architecture/{overview,erd,services}.md   TL
design/                                          the approved design (design-system.md is the token source of truth)
docker-compose.yml, deploy.yml, .github/         committed with the repo; nobody edits them
code/backend/                                    Go module github.com/ThanhNV121097/project-007ec16a/backend
  cmd/api/main.go                                config → connect → migrate → listen; the ONLY main package
  internal/config/                               environment, read once
  internal/db/                                   pool + migration runner
  internal/httpx/                                JSON helper, error envelope, Recover / AccessLog / CORS middleware
  internal/server/                               routes + middleware chain; `/healthz`; one Register line per module
  internal/<module>/                             one package per backend module: handler.go, store.go, *_test.go
  migrations/                                    embed.go + YYYYMMDDHHMMSS_name.{up,down}.sql (see its README)
code/frontend/                                   the design; pages in src/pages, parts in src/components, records in src/data
```

Backend module packages follow the SRS modules: `data` (collections, paged read), `catalogue` (query language over `data`),
`orders`, `payments`, `auth` (users, sessions, sign-in — serves admin, staff and customers alike), `admin` (writes,
uploads, overview, staff), `account` (customer account). A package exposes `Register(mux *http.ServeMux, deps)`;
`server.New` calls it. Packages do not import each other's stores — they share through `httpx` and `auth`.

## 4. Backend conventions

- **Start order is a contract** (`cmd/api/main.go`): read `DATABASE_URL` → connect (waits up to 30 s) → apply every pending
  migration → listen on `$PORT`, then `$APP_PORT`, then 8080. `/healthz` is 200 only while `SELECT 1` works, and 503
  with the generic envelope otherwise. The DSN is never assembled from `DB_*` parts and never logged.
- **Migrations:** `YYYYMMDDHHMMSS_what.up.sql` + `.down.sql`, each in one transaction with its `schema_migrations` row. A file whose
  SQL says `CONCURRENTLY` runs outside a transaction, one `;`-separated statement at a time, each safe to run twice
  (`IF NOT EXISTS`) — use it for any index on a table that already has rows. Never edit a merged file. The scaffold ships no
  schema: the first pair arrives with "Site data from the database".
- **Errors:** every non-2xx body is `{"error": "<code>", "message": "<sentence>", "errors": {"<field>": "<sentence>"}}`
  (`httpx.Error`; `errors` only on 400 and 422, matching the SRS's `{"errors":{…}}`). 401 no session, 403 wrong role, 404 for
  "not yours" exactly like "does not exist" (same body), 409 conflict, 422 invalid field, 429 with the wait, 503 when a
  dependency (VNPay keys) is missing. Never SQL, stack, or another person's data in a body; log the cause, answer generic.
- **Lists:** `GET /v1/<collection>` answers `{"items": [], "total": n, "page": p, "limit": l}`; `page` ≥ 1, `limit` default 12,
  capped at 100, a bad value is 400 naming it. A stable order is part of the contract (seed order, then insertion; ties broken by it).
- **SQL:** always parameterised (`$1`). The only text spliced into a query is a column or table name taken from an allow-list
  built from `schema.json`, never from the request.
- **Privacy:** the private order fields (`phone`, `email`, `address`, `note`) are never selected into a public type. Public and
  dealer reads are two structs, not one struct with fields deleted afterwards. Order tokens are ≥ 128 bits from `crypto/rand`,
  stored hashed.
- **Auth:** one `users` table, role `admin` | `staff` | `customer`; `auth.Require(roles…)` wraps every dealer route (401 / 403).
  Cookie `session`: `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` when the request came over HTTPS (`X-Forwarded-Proto`), 7 days.
  The admin account is made at start-up from `ADMIN_EMAIL` / `ADMIN_PASSWORD` (≥ 10 characters or it is skipped and logged).
- **Time and money:** `timestamptz` in UTC; business rules (cut-off 15:00, working days) use `Asia/Ho_Chi_Minh` (tzdata is in the
  image). VND is an integer. The shop computes every total from stored prices; a client never sends one.
- **Concurrency:** stock moves with a conditional update inside the order's transaction
  (`UPDATE … SET n = n - $q WHERE n >= $q`); "paid" is set by one `UPDATE … WHERE NOT paid` so a repeated VNPay notice pays once.
- **Logging:** `log/slog` JSON on stdout. Never a password, hash, cookie, token, DSN or VNPay secret.
- **Tests:** `go test ./...` must pass with no database. A test that needs one reads `TEST_DATABASE_URL` (CI sets it) and
  `t.Skip`s without it, working in a schema of its own — the pattern is `testPool` in `internal/db/migrate_test.go`.

## 5. Frontend conventions

- **Styling is tokens.** `src/theme.css` (colours, faces, radius, spacing, shadow, motion, plus the `[data-dashboard]` scope for
  the admin area) is frozen: a story never edits it, nor `src/index.css`, `src/content.json`, `package.json`,
  `eslint.config.js` or `package-lock.json`. Use the Tailwind names (`bg-ground`, `text-ink`, `border-line`, `p-4`); no raw
  colour, no arbitrary-value class, no Tailwind palette colour — eslint fails the build on them. There are no CSS modules.
  Every colour token §1.1 documents exists in `theme.css`; the typography, tracking and pill-radius names §4 asks for do not (section 9).
- **Gates (`npm run lint`):** `max-lines` 300 per file, `complexity` 20, no unused vars, no `any`. Split a page before it grows past 300.
- **Records are read only through `src/data/index.ts`** (`useCollection`, `useRecord`). The data story rewrites that one module to
  call `GET <base>/v1/<name>?page=&limit=&sort=&q=&<field>=<value>` and deletes every JSON file in `src/data/` except `schema.json`,
  which stays (the deploy gate and the admin forms read it). Components and their loading states do not change.
- **One API client.** The data story creates `src/lib/api.ts`: base URL, `credentials: "include"`, the error envelope parsed into
  a thrown `ApiError`. Later stories write `src/lib/{story-slug}.ts` on top of it. A second file with `api` in its name
  makes CI warn.
- **Session:** `src/lib/session.ts` keeps `useSession`, `signIn`, `signOut`, `useRole`; the dealer sign-in story rewrites it over
  `GET /v1/auth/me`. Nothing about who is signed in is kept in `localStorage`; an `/admin` page draws nothing until `/v1/auth/me` answers.
- **Mocks:** a UI story may add `src/lib/mock/{slug}.ts`; its backend story deletes it. CI warns while a mock is imported.
- **Routes:** a story adds an import and one `<Route>` to `src/App.tsx` and nothing else there. A page nobody routes to is dead code.
- **Phone first:** one column from 320 px, no horizontal scroll at 390 px; the call/Zalo dock stays on screens under 1024 px.

## 6. Environment variables

| Key | Service | Read by | Notes |
|---|---|---|---|
| `PORT` (`APP_PORT`) | backend | scaffold | listen port; image sets `PORT=8080` |
| `DATABASE_URL` | backend | scaffold | injected by compose / the runtime; required |
| `CORS_ALLOWED_ORIGINS` | backend | scaffold | empty in deployment; for `npm run dev` against a local API |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | backend | dealer sign-in | owner supplies in the console; password ≥ 10 chars |
| `VNPAY_TMN_CODE`, `VNPAY_HASH_SECRET` | backend | VNPay story | owner supplies; sandbox first; never in a `VITE_*` variable |
| `VNPAY_PAY_URL`, `PUBLIC_BASE_URL` | backend | VNPay story | production page address; override for the return/notice address when the proxy's `X-Forwarded-*` is not enough. Each is added to `.env.example` by that story |
| `VITE_API_URL` | frontend | data story | build time; unset means `/api`. Empty is not unset |
| `POSTGRES_USER/PASSWORD/DB`, `BACKEND_PORT`, `FRONTEND_PORT`, `*_MEM_LIMIT`, `COMPOSE_PROFILES` | compose | root `.env.example` | local only |

Every service has a tracked `.env.example` (root, `code/backend`, `code/frontend`): key names and comments, no secrets.

## 7. How to run and what CI checks

```
cp .env.example .env              # COMPOSE_PROFILES=local makes a plain `up` start Postgres too
docker compose up --build         # postgres → backend :8080 (/healthz) → frontend :3000   (CI: --profile local up -d --wait)
cd code/backend  && go build ./... && go vet ./... && go test ./...
cd code/frontend && npm ci && npm run lint && npm run build && npm test
# DB-backed Go tests: docker run --rm -p 5432:5432 -e POSTGRES_USER=ci -e POSTGRES_PASSWORD=ci -e POSTGRES_DB=ci postgres:16-alpine
#   then TEST_DATABASE_URL=postgres://ci:ci@localhost:5432/ci?sslmode=disable go test ./...
```

`ci.yml` gates every PR: **backend** `go mod download`, `build`, `vet`, `test` (with a Postgres service and `TEST_DATABASE_URL`);
**frontend** lock file present, `npm ci`, `lint`, `build`, `test`; **tokens** (no hex / `rgb()` / px ≥ 3 / `var(--x, fallback)` in any CSS
file except `theme.css` and `index.css`, every used token defined, the palette not left `UNSET`); **wiring** (a story PR may not delete
frontend files except under `src/lib/mock/` and `src/data/`, and never `src/data/schema.json`; warnings for mocks still imported,
components or pages nothing reaches, a second API client). `container.yml` builds all three images and boots them with `--wait`,
so the compose health checks are the assertion. A backend change that needs `go.sum` goes through `run_go_mod_tidy`, never by hand.

## 8. The stories, and what each must decide

Order: 2 → 3 → {4, 5, 8}; 5 → {6, 7, 14}; 8 → {9, 10, 13, 14}; 3 → 12; 7 → 11. The ERD (`erd.md`) and the contracts (`services.md`) are extended by TL in each story's design step.

| # | Story | Technical notes a later agent must not rediscover |
|---|---|---|
| 1 | The site as designed | Done: the frontend on JSON records. Nothing further |
| 2 | Free delivery on every order | Frontend/JSON only, **no backend**: one `delivery` record (`standard`, fee 0); `shop` loses `free_delivery_from`, `express_*`, `exchange_days`, `refund_days`. Must land before 3 — the seed copies the files as they stand |
| 3 | Site data from the database | Ten tables, one per `schema.json` collection, a column per field, text ids, `bigint` prices, list fields as `text[]`, references as foreign keys, `_source` kept. First migrations: schema, then seed from the JSON (re-running never duplicates or overwrites). `GET /v1/{collection}` and `/{id}`; collection and column names come from an allow-list; private order fields are hidden and filtering on them is 400 (DATA-004). Creates `src/lib/api.ts`, rewrites `src/data/index.ts`, deletes the JSON except `schema.json`. The pages need loading and error views the design never drew — an open design question for this story |
| 4 | Search, filters, sorting | `q` splits into words, every word must hit one searched field (joined line and category names for shoes), accents and `đ` folded. `a,b` any-of, `_from`/`_to` inclusive, `sort=-price` with the seed position as tie-break, `size=` = listed and not sold out. Same envelope as 3. Index on a table with rows → `CONCURRENTLY` |
| 5 | Checkout and placing an order | One transaction: re-read prices, conditional stock decrement per EU size, save the order and its pairs (an order holds several; the sample records are one-pair orders), number `DOI-<n>` never repeating. The answer carries the order token once (stored hashed). 409 names every failing pair; 503 and no order when VNPay keys are missing; delivery is always `standard`, fee 0. Order reads: token, dealer session, or the linked customer; every other case is 404 with one body |
| 6 | Track by number and phone | `POST` (not a GET) so the phone never sits in a URL: number accepted with or without `DOI-`, phone compared by digits with a leading `84` = `0`; every miss is the same 404; the hit returns only id, number, time and token |
| 7 | Online payment with VNPay | HMAC-SHA512 over the sorted parameters; a new reference per attempt; amount read from the database. The notice endpoint (`/v1/payments/vnpay/ipn`) checks signature, reference and amount and answers VNPay's `RspCode` bodies; paid exactly once (`02` for a repeat). `/payment/return` is a frontend route that asks the backend — it never trusts its own query string. A ticker cancels orders unpaid after 30 minutes and returns their pairs once. VNPay cannot reach a `.localhost` run: test with notices signed by the sandbox secret |
| 8 | Dealer sign-in, protected admin | `users`, `sessions`, failure-counter tables. `POST /v1/auth/sign-in`, `POST /v1/auth/sign-out` (204 even with no session), `GET /v1/auth/me`. Unknown email and wrong password cost the same (hash a dummy), same status, same body. 5 failures per email or address in 15 min → 429. Rewrites `src/lib/session.ts` and `AdminSignIn.tsx`; the `/admin/*` gate draws nothing until `/v1/auth/me` answers; a customer sees "This area is for the shop's team" |
| 9 | Manage shoes, lines, categories, reviews, orders | `POST/PUT/DELETE /v1/{collection}` validated server-side from `schema.json` (required, type, option, reference) → 422 `{"errors":…}`; a visitor never gets draft shoes or hidden reviews whatever they ask. Uploads: `POST /v1/uploads`, `http.MaxBytesReader` 5 MB (413), type by content (415), server-chosen name. Per-size counts save with the count last seen (409 if an order moved it). `paid` is never writable by hand (422) |
| 10 | Dealer dashboard | One `GET /v1/admin/overview` answering all figures from SQL aggregates; days cut at Vietnam midnight and zero-filled. The existing `AdminBars` draws the charts — no chart library |
| 11 | VietQR transfer and MoMo (P3) | A payment row per provider attempt; MoMo's own signed notice; the dealer marks a bank-transfer order paid and no other. New secrets are requested when this story starts |
| 12 | Product page missing-shoe bug | Frontend only, after 3: no "not found" until the record's request has finished |
| 13 | Staff accounts | `/v1/admin/staff` for role `admin` only; the admin's own row is protected (409); switching an account off ends its sessions, because every request re-reads `users.active` |
| 14 | Customer accounts | `/v1/account/sign-up`, `/v1/account`, `/v1/account/orders`; `orders.account_id` is set only for role `customer`; email unique across all roles, case-insensitive; sign-up 10 per hour per address; no email or SMS is sent anywhere |

## 9. Risks and open items

- **Plain compose has no `/api`.** The frontend image's nginx cannot proxy (committed, fixed). The pages load on :3000; a working API
  path needs the edge proxy. Verify UI-to-API behaviour through `run_project`, not `docker compose up` alone.
- **Tokens the design system asks for and `theme.css` lacks** (design-system §4): `--text-label`, `--tracking-caps` and a pill-radius
  utility name; also a ≥ 3:1 dashboard `--field-line`, a public focus ring and 44 px touch targets. The file is the owner's approved
  design, so **this task did not add them**; each needs the owner's agreement and a separate TL change. Meanwhile stories write the
  arbitrary sizes the design already uses (eslint refuses arbitrary colours only).
- **VNPay's notice address** must be registered by the owner in VNPay's portal (`/api/v1/payments/vnpay/ipn` on the deployed
  site); declared to the console with `request_setup`. Keys are declared with `request_secret`; none is ever committed.
- **Pictures in Postgres** grow the database by up to 5 MB each; acceptable for a shop's catalogue, revisit if a volume becomes possible.
- **Client address for the sign-in limit** comes from the proxy's `X-Forwarded-For`; the service design for story 8 pins which hop is trusted.
- **`go 1.26.0` and `golang:1.26-alpine`** move together; a go.mod newer than the builder fails the image build.
