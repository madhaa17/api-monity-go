# Monity — Finance Tracker API

Backend API for a personal finance tracker: assets (cash, crypto, stocks, …), income and expenses with custom categories, debts and receivables, saving goals, portfolio, and insights.

## Tech stack

- **Go 1.24+** — stdlib `net/http`, no framework
- **PostgreSQL** — via GORM
- **Redis** — optional; used for price cache (crypto/stock) to reduce external API calls and improve response time
- **Price sources** — CoinGecko (crypto), Yahoo Finance (stock); both free, no API key
- **JWT** — auth (Bearer token)
- **Decimal** — `shopspring/decimal` for money/quantity values

## Project structure

```
backend/
├── cmd/server/          # Entrypoint (main.go)
├── internal/
│   ├── adapter/         # HTTP handlers, middleware, repository (GORM)
│   ├── app/             # Wiring, routes
│   ├── config/         # Env & config structs
│   ├── core/            # Ports (interfaces) & services (business logic)
│   ├── database/        # DB connection
│   ├── models/          # Domain entities
│   └── pkg/response/    # JSON response helpers
├── migrations/          # SQL migrations (001–009), applied in order
├── Dockerfile
├── docker-compose.yml   # Dokploy / Docker Compose
└── .env.example
```

## Local setup

1. **Clone & dependencies**

   ```bash
   cd backend
   go mod download
   ```

2. **Environment**

   ```bash
   cp .env.example .env
   # Edit .env: DATABASE_*, JWT_SECRET, JWT_REFRESH_SECRET (required; crypto/stock prices need no API key)
   ```

3. **Database**

   - Create a PostgreSQL database (e.g. `monity_db`).
   - Run every migration in `migrations/`, in filename order. Each file runs once; there is no migration tool or tracking table.

   ```bash
   for f in migrations/*.up.sql; do
     psql -h localhost -p 5432 -U postgres -d monity_db -v ON_ERROR_STOP=1 -f "$f" || break
   done
   ```

   | File | What it does |
   |------|--------------|
   | `001_initial_schema` | Users, assets, asset price history, incomes, expenses, saving goals |
   | `002_enhance_assets` | Purchase details, costs, targets and status for assets |
   | `003_asset_type_livestock` | Adds the `LIVESTOCK` asset type |
   | `004_asset_type_cash_real_estate` | Adds the `CASH` and `REAL_ESTATE` asset types |
   | `005_link_cash_asset` | Links incomes/expenses to the CASH asset they move money in/out of (**deletes all existing incomes and expenses**) |
   | `006_debts_and_receivables` | Debts, receivables and their installment payments |
   | `007_categories` | Per-user categories; creates the defaults for existing users and moves `expenses.category` (enum) to `category_id` |
| `008_market_daily_closes` | Daily closing prices per crypto/stock symbol, filled by the backend every 6 hours (past year each run) |
| `009_user_password_changed_at` | Records when a user last changed their password, so older tokens can be rejected |

   On an existing database, run only the files that have not been applied yet. Back up first before `007`: it drops the old `expenses.category` column after copying it.

   No users are seeded. Register through `POST /api/v1/auth/register`; registration also creates the user's default categories.

4. **Run the server**

   ```bash
   go run ./cmd/server
   # Local: http://localhost:8080 (APP_PORT in .env)
   ```

## API overview

- **Local:** `http://localhost:8080` · **Production:** port 8386 (or your deployment domain)
- **API prefix:** `/api/v1`
- **OpenAPI:** [docs/openapi.yaml](docs/openapi.yaml) — full spec for codegen (e.g. openapi-generator, orval) and reference (paths, query params, request/response schemas, list pagination `items` + `meta`, error object shape).

| Area         | Example endpoints                      | Auth   |
|-------------|-----------------------------------------|--------|
| Root        | `GET /` → `{"status":"ok"}`             | —      |
| Health      | `GET /health` → status + DB             | —      |
| Auth        | `POST /api/v1/auth/register`, `.../login`, `.../refresh`, `GET .../me`, `POST .../logout`, `PUT .../password` | Bearer (me, password) |
| Activities  | `GET /api/v1/activities?group_by=day&date=YYYY-MM-DD` — incomes, expenses, debts, receivables grouped (day/month/year), optional `date`, `tz` | Bearer |
| Assets      | CRUD assets (crypto, stock, etc.)       | Bearer |
| Categories  | `GET/POST /api/v1/categories`, `GET/PUT/DELETE .../categories/{uuid}` — per-user INCOME/EXPENSE categories; delete an in-use category with `?reassign_to=<uuid>` | Bearer |
| Incomes     | CRUD income, `GET .../incomes/summary` — list filters and sort, see [Incomes and expenses](#incomes-and-expenses) | Bearer |
| Expenses    | CRUD expenses, `GET .../expenses/summary` — same filters as incomes | Bearer |
| Saving goals| CRUD saving goals                       | Bearer |
| Debts       | CRUD debts (hutang), `POST/GET .../debts/{uuid}/payments` for installments | Bearer |
| Receivables | CRUD receivables (piutang), `POST/GET .../receivables/{uuid}/payments` for installments | Bearer |
| Price       | Crypto (CoinGecko) / stock (Yahoo Finance) — free, no API key | —      |
| Price chart| `GET .../prices/crypto/:symbol/chart?days=7&currency=idr`, `GET .../prices/stock/:symbol/chart?range=1mo&interval=1d`. Response: time series `data[]` dengan `t` (Unix second) dan `p` (price); lihat [docs/curl-examples.md](docs/curl-examples.md) untuk format lengkap. | —      |
| Portfolio   | Portfolio summary                       | Bearer |
| Performance | Asset performance                       | Bearer |
| Insight     | Financial insights (overview includes totalDebt, totalReceivable, overdue counts) | Bearer |

Protected routes require header: `Authorization: Bearer <token>`.

- **Postman:** Import [docs/postman_collection.json](docs/postman_collection.json). Login/Register auto-save `token`.

### Incomes and expenses

Every income and expense moves money into or out of a **CASH** asset, so the asset balance always equals the recorded transactions.

- **Atomic balance:** create, update and delete change the transaction and the asset balance in one database transaction. An expense (or an edit, or deleting an income) that would make the balance negative returns **400** and changes nothing, even with concurrent requests.
- **Amounts** are decimals with at most 2 decimal places. Requests accept a number (`45000`) or a string (`"45000.50"`); responses always return a string (`"45000"`) so no precision is lost.
- **Categories:** send `categoryUuid` (from `GET /categories?type=EXPENSE|INCOME`). Incomes without a category go to the user's "Other" category. For expenses, the old `category: "FOOD"` (a category name, case-insensitive) still works when `categoryUuid` is omitted.
- **Responses** include `categoryUuid`, `category` (name), `assetUuid` and `assetName`.

List filters for `GET /incomes` and `GET /expenses` (all optional, combined with AND):

| Query | Example | Meaning |
|-------|---------|---------|
| `date_from`, `date_to` | `2025-02-01` | Inclusive date range (UTC); either one alone works |
| `month` / `year` | `2025-02` / `2025` | Whole month or year, used when no date range is given |
| `category` | `uuid1,uuid2` | Any of these category UUIDs |
| `asset` | `uuid` | One CASH asset |
| `q` | `kopi` | Case-insensitive search in note (and source, for incomes) |
| `min_amount`, `max_amount` | `10000` | Inclusive amount range |
| `sort` | `-amount` | `-date` (default), `date`, `-amount`, `amount` |
| `page`, `limit` | `1`, `20` | Paging; `limit` max 100 |

`GET /incomes/summary` and `GET /expenses/summary` take the same filters (without paging and sort) plus `group_by=day|month`, and return `total`, `count`, `average`, `byCategory[]` (largest first, with percentage and color) and a `series[]` per day or month.

Invalid query values (malformed date or UUID, `date_from` after `date_to`, unknown `sort`) return **400** instead of being ignored.

### Date and time

- All dates and times from the API use **ISO 8601 / RFC3339** (e.g. `2025-02-10T00:00:00Z`).
- Body fields such as `date` on income/expense use that format. Date filters in query parameters (`date_from`, `date_to`) are plain dates (`YYYY-MM-DD`).
- **Timezone:** Server uses **UTC** for storage and range calculations. When sending `date_from`/`date_to` or `month`/`year`, values are interpreted in UTC. For local-time grouping (e.g. activities), use the optional `tz` query (IANA timezone) where supported.

### Currency

Endpoints that support a `currency` query parameter (for display/conversion). Default is **IDR** when omitted.

| Endpoint | Query | Default | Notes |
|----------|--------|--------|--------|
| `GET /portfolio` | `currency` | IDR | Portfolio summary in given currency |
| `GET /portfolio/assets/{uuid}` | `currency` | IDR | Single asset value |
| `GET /assets/{uuid}/performance` | `currency` | IDR | Asset performance |
| `GET /portfolio/performance` | `currency` | IDR | Portfolio performance |
| `GET /prices/crypto/{symbol}` | `currency` | IDR | Current crypto price |
| `GET /prices/stock/{symbol}` | `currency` | IDR | Current stock price |
| `GET /prices/crypto/{symbol}/chart` | `currency` | IDR | Chart series |
| `GET /prices/stock/{symbol}/chart` | `currency` | IDR | Chart series |

Example: `GET /api/v1/portfolio?currency=USD`.

### Auth and refresh token flow

- **Login / Register** return the access JWT and the refresh JWT. The response fields are currently PascalCase — `data.Token`, `data.RefreshToken`, `data.User` — not the camelCase used elsewhere; the web client accepts both. Store both (e.g. in memory + `localStorage`, or secure cookies) as needed for your app.
- **Protected requests:** Send the access token in the header: `Authorization: Bearer <access_token>`.
- **When to refresh:** Call `POST /api/v1/auth/refresh` with body `{ "refresh_token": "<refresh_token>" }` either:
  - **Proactively** before the access token expires (e.g. when you know TTL and refresh a bit earlier), or
  - **On 401** when any protected request returns Unauthorized — then retry the request after storing the new tokens from the refresh response.
- **Refresh response** has the same shape as login. Replace stored access and refresh with the new values: each refresh rotates the refresh token, and the old one stops working 30 seconds after its first use (the grace period lets tabs that refresh at the same time all succeed).
- **Logout:** `POST /api/v1/auth/logout` with the access token in `Authorization: Bearer` and/or `refresh_token` in the body revokes both; a revoked refresh token is rejected by `/auth/refresh`. It needs no valid session, so it works after the access token has expired. Discard stored tokens on the client.
- **Change password:** `PUT /api/v1/auth/password` with body `{ "current_password": "...", "new_password": "..." }`. Every token issued before the change stops working (all other sessions are signed out); store the new token pair from the response, which has the same shape as login. Limited to 5 requests per 15 minutes per client.

## Security & middleware

- **Rate limit** — in-memory per IP (`RATE_LIMIT_TTL`, `RATE_LIMIT_LIMIT`); returns 429 when exceeded.
- **Security headers** — `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Referrer-Policy`.
- **CORS** — controlled via `CORS_ALLOWED_ORIGINS` (`*` or comma-separated list of origins).
- **Auth** — JWT middleware for protected routes.

## Important env variables

| Variable               | Description                    |
|------------------------|--------------------------------|
| `APP_PORT`             | Server port (default 8080)     |
| `DATABASE_HOST`, `*`   | PostgreSQL connection         |
| `JWT_SECRET`, `JWT_REFRESH_SECRET` | **Required**, no defaults. Must differ; ≥ 32 chars when `APP_ENV=production` (`openssl rand -hex 32`). The server refuses to start otherwise. |
| `CRYPTO_PRICE_API_KEY` | Optional CoinGecko key (higher rate limit) |
| `STOCK_API_URL` | Optional Yahoo base URL override, e.g. a relay (`relay/yahoo-worker.js`) when Yahoo answers 429 to the server IP |
| `STOCK_API_PROXY` | Optional `http://` / `socks5://` proxy for Yahoo requests only |
| `STOCK_API_RELAY_TOKEN` | Sent as `X-Relay-Token`; must match the relay's `RELAY_TOKEN` |
| `RATE_LIMIT_TTL`       | Rate limit window (seconds)   |
| `RATE_LIMIT_LIMIT`     | Max requests per window per IP |
| `CORS_ALLOWED_ORIGINS` | `*` or comma-separated origins |
| `REDIS_HOST`           | Redis host for cache (empty = in-memory cache) |
| `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | Redis connection |
| `REDIS_TTL_PRICE`      | Price cache TTL in seconds    |

**Prices:** Crypto prices use **CoinGecko** (free, no API key). Stock prices use **Yahoo Finance** (free, no API key; IDX symbols get `.JK` suffix). The provider base URLs (CoinGecko, Yahoo Finance, open.er-api.com for fallback FX) are constants in `internal/config/config.go`, not env variables.

If `REDIS_HOST` is set, the app uses Redis for caching crypto/stock prices and FX rates, improving performance and sharing cache across instances. Otherwise, an in-memory cache is used (single instance only).

See `.env.example` for the full list.
