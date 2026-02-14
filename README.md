# Monity — Finance Tracker API

Backend API for a personal finance tracker: assets (crypto, stocks), income, expenses, saving goals, portfolio, and insights.

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
├── migrations/          # SQL migrations (001, 002)
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
   # Edit .env: DATABASE_*, JWT_SECRET (crypto/stock prices use CoinGecko & Yahoo Finance, no API key needed)
   ```

3. **Database**

   - Create a PostgreSQL database (e.g. `monity_db`).
   - Run migrations in order:

   ```bash
   psql -h localhost -p 5432 -U postgres -d monity_db -f migrations/001_initial_schema.up.sql
   psql -h localhost -p 5432 -U postgres -d monity_db -f migrations/002_enhance_assets.up.sql
   ```

4. **Run the server**

   ```bash
   go run ./cmd/server
   # Local: http://localhost:8080 (APP_PORT in .env)
   ```

## API overview

- **Local:** `http://localhost:8080` · **Production:** port 8386 (or your deployment domain)
- **API prefix:** `/api/v1`

| Area         | Example endpoints                      | Auth   |
|-------------|-----------------------------------------|--------|
| Root        | `GET /` → `{"status":"ok"}`             | —      |
| Health      | `GET /health` → status + DB             | —      |
| Auth        | `POST /api/v1/auth/register`, `.../login`, `.../refresh`, `GET .../me`, `POST .../logout` | Bearer (me, logout) |
| Activities  | `GET /api/v1/activities?group_by=day&date=YYYY-MM-DD` — incomes + expenses grouped (day/month/year), optional `date`, `tz` | Bearer |
| Assets      | CRUD assets (crypto, stock, etc.)       | Bearer |
| Incomes     | CRUD income                             | Bearer |
| Expenses    | CRUD expenses                           | Bearer |
| Saving goals| CRUD saving goals                       | Bearer |
| Price       | Crypto (CoinGecko) / stock (Yahoo Finance) — free, no API key | Bearer |
| Portfolio   | Portfolio summary                       | Bearer |
| Performance | Asset performance                       | Bearer |
| Insight     | Financial insights                      | Bearer |

Protected routes require header: `Authorization: Bearer <token>`.

- **Postman:** Import [docs/Monity_API.postman_collection.json](docs/Monity_API.postman_collection.json). Login/Register auto-save `token`;

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
| `JWT_SECRET`           | Secret for signing JWT        |
| `STOCK_PRICE_API`      | Yahoo Finance base URL (optional override; default in .env.example) |
| `RATE_LIMIT_TTL`       | Rate limit window (seconds)   |
| `RATE_LIMIT_LIMIT`     | Max requests per window per IP |
| `CORS_ALLOWED_ORIGINS` | `*` or comma-separated origins |
| `REDIS_HOST`           | Redis host for cache (empty = in-memory cache) |
| `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | Redis connection |
| `REDIS_TTL_PRICE`      | Price cache TTL in seconds    |

**Prices:** Crypto prices use **CoinGecko** (free, no API key). Stock prices use **Yahoo Finance** (free, no API key; IDX symbols get `.JK` suffix). See `.env.example` for `STOCK_PRICE_API` if you need to override the Yahoo base URL.

If `REDIS_HOST` is set, the app uses Redis for caching crypto/stock prices and FX rates, improving performance and sharing cache across instances. Otherwise, an in-memory cache is used (single instance only).

See `.env.example` for the full list.
