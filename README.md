# Monity — Finance Tracker API

Backend API for a personal finance tracker: assets (crypto, stocks), income, expenses, saving goals, portfolio, and insights.

## Tech stack

- **Go 1.24+** — stdlib `net/http`, no framework
- **PostgreSQL** — via GORM
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
   # Edit .env: DATABASE_*, JWT_SECRET, CRYPTO_PRICE_API_KEY (optional)
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
   # Default: http://localhost:8080
   ```

## API overview

- **Base URL:** `http://localhost:8080` (or your deployment domain)
- **API prefix:** `/api/v1`

| Area         | Example endpoints                      | Auth   |
|-------------|-----------------------------------------|--------|
| Root        | `GET /` → `{"status":"ok"}`             | —      |
| Health      | `GET /health` → status + DB            | —      |
| Auth        | `POST /api/v1/auth/register`, `.../login` | —    |
| Assets      | CRUD assets (crypto, stock, etc.)       | Bearer |
| Incomes     | CRUD income                             | Bearer |
| Expenses    | CRUD expenses                           | Bearer |
| Saving goals| CRUD saving goals                       | Bearer |
| Price       | Crypto/stock prices (external API)      | Bearer |
| Portfolio   | Portfolio summary                       | Bearer |
| Performance | Asset performance                       | Bearer |
| Insight     | Financial insights                      | Bearer |

Protected routes require header: `Authorization: Bearer <token>`.

## Security & middleware

- **Rate limit** — in-memory per IP (`RATE_LIMIT_TTL`, `RATE_LIMIT_LIMIT`); returns 429 when exceeded.
- **Security headers** — `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Referrer-Policy`.
- **CORS** — controlled via `CORS_ALLOWED_ORIGINS` (`*` or comma-separated list of origins).
- **Auth** — JWT middleware for protected routes.

## Important env variables

| Variable               | Description                    |
|------------------------|--------------------------------|
| `DATABASE_HOST`, `*`   | PostgreSQL connection         |
| `JWT_SECRET`           | Secret for signing JWT        |
| `CRYPTO_PRICE_API_KEY` | CoinMarketCap API key (optional) |
| `RATE_LIMIT_TTL`       | Rate limit window (seconds)   |
| `RATE_LIMIT_LIMIT`     | Max requests per window per IP |
| `CORS_ALLOWED_ORIGINS` | `*` or comma-separated origins |

See `.env.example` for the full list.
