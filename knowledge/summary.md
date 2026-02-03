# Project Summary - Finance Tracker Backend

## 🎯 Tujuan Project

Project **Finance Tracker Backend** bertujuan untuk membangun sistem backend yang mampu membantu pengguna dalam:

- Memantau kondisi keuangan pribadi secara terstruktur
- Melacak aset, tabungan, pendapatan, dan pengeluaran
- Menyediakan data harga saham dan crypto yang ter-update secara real-time
- Menghasilkan insight cashflow & kesehatan finansial berbasis data
- Memantau nilai portfolio secara real-time dengan multi-currency support

Project ini dirancang sebagai:

- ✅ Aplikasi **real-world use case** (bukan sekadar CRUD)
- ✅ **Portfolio project** untuk Backend / Fullstack / Cloud Engineer
- ✅ Fondasi backend yang **scalable & maintainable**
- ✅ **Multi-user application** yang siap production

---

## ⚙️ Fungsi Utama Sistem

### 1️⃣ Multi-User Authentication & Authorization

- ✅ Mendukung banyak pengguna (multi-user)
- ✅ Fitur registrasi & login (email/password)
- ✅ JWT-based authentication dengan token expiration
- ✅ Setiap user hanya bisa mengakses datanya sendiri
- ✅ Role-based access support (USER/ADMIN)
- ✅ Middleware authentication untuk protected routes

**API Endpoints:**
- `POST /api/v1/auth/register` - Registrasi user baru
- `POST /api/v1/auth/login` - Login dan generate JWT token

### 2️⃣ Data Isolation (Per User Security)

- ✅ Semua data (asset, expense, income, saving) terikat ke `userId`
- ✅ Query **selalu difilter** berdasarkan user yang sedang login
- ✅ Mencegah kebocoran data antar pengguna
- ✅ Cascade delete untuk data integrity
- ✅ Database index pada `userId` untuk performance

### 3️⃣ Asset Management

- ✅ Mencatat berbagai jenis aset: **CRYPTO, STOCK, CASH, REAL_ESTATE**
- ✅ Setiap aset dimiliki oleh satu user
- ✅ Menyimpan quantity dengan high precision (Decimal 20,8)
- ✅ Symbol tracking untuk crypto & stocks
- ✅ Full CRUD operations

**API Endpoints:**
- `POST /api/v1/assets` - Create asset
- `GET /api/v1/assets` - List all user assets
- `GET /api/v1/assets/:uuid` - Get asset detail
- `PUT /api/v1/assets/:uuid` - Update asset
- `DELETE /api/v1/assets/:uuid` - Delete asset

### 4️⃣ Asset Price History Tracking

- ✅ Menyimpan historical prices per asset
- ✅ Manual price recording
- ✅ Auto-fetch price dari external API
- ✅ Source tracking (manual/api)
- ✅ Timeline-based price analysis

**API Endpoints:**
- `GET /api/v1/assets/:uuid/prices` - Get price history
- `POST /api/v1/assets/:uuid/prices` - Manually record price
- `POST /api/v1/assets/:uuid/prices/fetch` - Fetch & record current price from API

### 5️⃣ Real-time & Cached Price Service

- ✅ Mengambil harga crypto secara **near real-time** dari CoinMarketCap API
- ✅ Mengambil harga saham dari Yahoo Finance API
- ✅ **Multi-currency support** (USD, IDR, dan currencies lainnya)
- ✅ Automatic currency conversion via forex rates
- ✅ In-memory caching dengan configurable TTL (default 60s)
- ✅ Cache global untuk semua users (efficient rate limiting)

**API Endpoints:**
- `GET /api/v1/prices/crypto/:symbol` - Get crypto price (e.g., BTC)
- `GET /api/v1/prices/stock/:symbol` - Get stock price (e.g., AAPL)
- Query params: `?currency=IDR` untuk currency conversion

**External APIs:**
- CoinMarketCap API - Real-time crypto prices
- Yahoo Finance API - Stock prices & forex rates

### 6️⃣ Portfolio Management

- ✅ Real-time portfolio valuation
- ✅ Aggregate total value dari semua assets
- ✅ Per-asset value calculation dengan current market prices
- ✅ Multi-currency portfolio display
- ✅ Automatic price fetching & calculation
- ✅ Graceful handling untuk assets tanpa price data

**API Endpoints:**
- `GET /api/v1/portfolio` - Get full portfolio value & breakdown
- `GET /api/v1/portfolio/assets/:uuid` - Get specific asset current value

### 7️⃣ Expense & Income Tracking

- ✅ Mencatat pengeluaran dengan kategorisasi
- ✅ Mencatat pemasukan dari berbagai sumber
- ✅ Date-based tracking untuk timeline analysis
- ✅ Note/description support
- ✅ Sorting by date (most recent first)
- ✅ Menjadi dasar perhitungan cashflow

**Expense Categories:**
- FOOD, TRANSPORT, HOUSING, UTILITIES, HEALTHCARE, ENTERTAINMENT, SHOPPING, EDUCATION, OTHER

**API Endpoints:**
- `POST /api/v1/expenses` - Create expense
- `GET /api/v1/expenses` - List expenses
- `GET /api/v1/expenses/:uuid` - Get expense detail
- `PUT /api/v1/expenses/:uuid` - Update expense
- `DELETE /api/v1/expenses/:uuid` - Delete expense

- `POST /api/v1/incomes` - Create income
- `GET /api/v1/incomes` - List incomes
- `GET /api/v1/incomes/:uuid` - Get income detail
- `PUT /api/v1/incomes/:uuid` - Update income
- `DELETE /api/v1/incomes/:uuid` - Delete income

### 8️⃣ Saving & Goal Tracking

- ✅ Membantu pengguna menetapkan target tabungan pribadi
- ✅ Track current progress vs target amount
- ✅ Deadline tracking untuk goals
- ✅ Progress percentage calculation
- ✅ Multiple saving goals per user

**API Endpoints:**
- `POST /api/v1/saving-goals` - Create saving goal
- `GET /api/v1/saving-goals` - List saving goals
- `GET /api/v1/saving-goals/:uuid` - Get goal detail
- `PUT /api/v1/saving-goals/:uuid` - Update goal progress
- `DELETE /api/v1/saving-goals/:uuid` - Delete goal

### 9️⃣ Cashflow & Financial Insights

- ✅ Menghitung **net saving** (income - expense) per user
- ✅ Calculate **expense ratio** dan **saving rate**
- ✅ Monthly cashflow breakdown
- ✅ Overall financial overview
- ✅ Date range filtering
- ✅ Category-wise expense analysis
- ✅ Ready untuk dikembangkan menjadi financial health score

**API Endpoints:**
- `GET /api/v1/insights/cashflow` - Overall cashflow
- `GET /api/v1/insights/cashflow?month=2026-01` - Monthly cashflow
- `GET /api/v1/insights/overview` - Financial overview & metrics

---

## 🏗️ Technology Stack

### Backend Framework & Language
- **Go 1.21+** - High-performance, compiled language
- **net/http** - Standard library HTTP server
- **Clean Architecture** - Hexagonal architecture pattern

### Database & ORM
- **PostgreSQL** - Production-grade relational database
- **GORM** - Go ORM for database operations
- **Migrations** - SQL-based database migrations

### Authentication & Security
- **JWT (JSON Web Tokens)** - Stateless authentication
- **bcrypt** - Password hashing
- **Middleware-based** authorization

### External APIs
- **CoinMarketCap API** - Crypto price data
- **Yahoo Finance API** - Stock prices & forex rates

### Caching
- **In-memory cache** with sync.RWMutex
- Configurable TTL via environment variables
- Ready to upgrade to Redis for horizontal scaling

### Development Tools
- **Postman Collection** - Complete API documentation & testing
- **Environment Configuration** - .env based config management
- **Health Check** endpoint - `/health` for monitoring

---

## 📁 Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── adapter/
│   │   ├── handler/             # HTTP handlers (controllers)
│   │   ├── middleware/          # Auth middleware
│   │   └── repository/          # Database repositories
│   ├── app/
│   │   ├── app.go               # Application setup
│   │   └── routes/              # Route definitions
│   ├── config/                  # Configuration management
│   ├── core/
│   │   ├── port/                # Interface definitions (ports)
│   │   └── service/             # Business logic services
│   ├── database/                # Database connection
│   ├── models/                  # Domain models
│   └── pkg/
│       └── response/            # Standardized API responses
├── migrations/                  # Database migrations
└── knowledge/                   # Project documentation
    ├── postman_collection.json  # Postman API collection
    ├── rules.md                 # Development rules
    ├── schema.md                # Database schema
    └── summary.md               # This file
```

---

## 🗄️ Database Schema

### Core Tables

1. **users** - User accounts & authentication
2. **assets** - User-owned assets (crypto, stocks, cash, real estate)
3. **asset_price_histories** - Historical price tracking per asset
4. **incomes** - Income transactions
5. **expenses** - Expense transactions with categories
6. **saving_goals** - Savings targets & progress

### Key Features
- UUID-based public identifiers
- BigInt internal IDs
- Decimal precision for financial data
- Comprehensive indexing for performance
- Foreign key constraints with cascade delete
- Timestamps (createdAt, updatedAt)

---

## 🚀 Key Features Summary

### ✅ Implemented Features

| Feature | Status | Description |
|---------|--------|-------------|
| Multi-user Authentication | ✅ | JWT-based with bcrypt password hashing |
| Data Isolation | ✅ | Per-user data filtering on all queries |
| Asset Management | ✅ | CRUD for crypto, stock, cash, real estate |
| Price History | ✅ | Track & analyze asset price over time |
| Real-time Prices | ✅ | Live crypto & stock prices with caching |
| Portfolio Valuation | ✅ | Real-time portfolio value calculation |
| Multi-currency | ✅ | USD, IDR, and other currencies |
| Expense Tracking | ✅ | Categorized expense management |
| Income Tracking | ✅ | Income source tracking |
| Saving Goals | ✅ | Target & progress tracking |
| Cashflow Insights | ✅ | Net saving, expense ratio, saving rate |
| Financial Overview | ✅ | Comprehensive financial metrics |

### 🔄 Ready for Production

- ✅ Environment-based configuration
- ✅ Database connection pooling
- ✅ Graceful error handling
- ✅ Standardized API responses
- ✅ Health check endpoint
- ✅ API documentation (Postman)
- ✅ Clean architecture (maintainable & testable)

### 🎯 Future Enhancements

- 🔄 Redis integration for distributed caching
- 🔄 Background job for scheduled price updates
- 🔄 WebSocket support for real-time price streaming
- 🔄 Financial health score calculation
- 🔄 Budget planning & forecasting
- 🔄 Email notifications for goals & alerts
- 🔄 API rate limiting per user
- 🔄 Admin dashboard & analytics
- 🔄 Export data (CSV, PDF reports)
- 🔄 OAuth2 social login integration

---

## 📊 API Overview

**Total Endpoints:** 31+

### Authentication (2)
- Register, Login

### Assets (5)
- Create, List, Get, Update, Delete

### Asset Price History (3)
- Get history, Record manually, Fetch from API

### Expenses (5)
- Create, List, Get, Update, Delete

### Incomes (5)
- Create, List, Get, Update, Delete

### Saving Goals (5)
- Create, List, Get, Update, Delete

### Prices (2)
- Get crypto price, Get stock price

### Portfolio (2)
- Get portfolio, Get asset value

### Insights (3)
- Cashflow, Cashflow by month, Financial overview

### Health (1)
- Health check

---

## 🎓 Learning Outcomes

Projek ini cocok untuk portfolio karena mencakup:

1. ✅ **Backend Development** - REST API design & implementation
2. ✅ **Database Design** - Relational modeling with proper indexes
3. ✅ **Authentication & Security** - JWT, password hashing, data isolation
4. ✅ **External API Integration** - Third-party API consumption
5. ✅ **Caching Strategy** - Performance optimization
6. ✅ **Clean Architecture** - Separation of concerns, testability
7. ✅ **Financial Domain** - Real-world business logic
8. ✅ **Multi-tenancy** - Secure multi-user application

---

## 🔧 Getting Started

### Prerequisites
- Go 1.21 or higher
- PostgreSQL 14+
- CoinMarketCap API Key (free tier)

### Environment Variables
```env
APP_ENV=development
APP_PORT=8080

DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=your_user
DATABASE_PASSWORD=your_password
DATABASE_NAME=finance_tracker

JWT_SECRET=your-secret-key
JWT_EXPIRATION_TIME=24h

CRYPTO_PRICE_API=https://pro-api.coinmarketcap.com
CRYPTO_PRICE_API_KEY=your-coinmarketcap-api-key
STOCK_PRICE_API=https://query1.finance.yahoo.com
REDIS_TTL_PRICE=60
```

### Run Application
```bash
# Install dependencies
go mod download

# Run migrations
# (Apply migrations/001_initial_schema.up.sql to your database)

# Run server
go run cmd/server/main.go
```

Server will start at `http://localhost:8080`

---

## 📝 Notes

- Aplikasi ini **production-ready** untuk multi-user deployment
- All financial calculations use **Decimal** type untuk precision
- Price caching menggunakan in-memory (bisa upgrade ke Redis)
- Complete Postman collection tersedia untuk testing
- Schema documentation tersedia di `knowledge/schema.md`

---

**Project Status:** ✅ **Complete & Production Ready**

**Version:** 1.0.0

**Last Updated:** February 2026
