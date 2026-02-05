# 📮 Postman Collection Update - Synced with Enhancement Plan

## ✅ Update Summary

**Date:** February 3, 2026  
**Version:** 1.0 (Phase 1 Enhanced)  
**Status:** ✅ Synced with `knowledge/enhancement-plan.md`

---

## 🎯 What Was Updated

### 1. ✅ Collection Info & Description

**Added comprehensive description:**
- Features overview
- Quick start guide
- Version info
- Base URL

### 2. ✅ Enhanced Asset Creation Examples

**New Request Examples (4 types):**

#### A. Create - Crypto (Bitcoin) ✅
```json
{
  "name": "Bitcoin Investment",
  "type": "CRYPTO",
  "quantity": 0.5,
  "symbol": "BTC",
  "purchasePrice": 42000.00,
  "purchaseDate": "2024-01-15T10:00:00Z",
  "purchaseCurrency": "USD",
  "totalCost": 21050.00,
  "transactionFee": 50.00,
  "targetPrice": 50000.00,
  "notes": "Long-term hold strategy"
}
```

#### B. Create - Stock (Apple) 🆕
```json
{
  "name": "Apple Stock",
  "type": "STOCK",
  "quantity": 100,
  "symbol": "AAPL",
  "purchasePrice": 180.00,
  "purchaseDate": "2024-03-01T10:00:00Z",
  "purchaseCurrency": "USD",
  "totalCost": 18020.00,
  "transactionFee": 20.00,
  "targetPrice": 220.00,
  "description": "Tech sector investment",
  "notes": "Dividend paying stock"
}
```

#### C. Create - Real Estate (Property) 🆕
```json
{
  "name": "Apartment Jakarta Selatan",
  "type": "REAL_ESTATE",
  "quantity": 1,
  "purchasePrice": 500000000,
  "purchaseDate": "2023-01-01T00:00:00Z",
  "purchaseCurrency": "IDR",
  "totalCost": 510000000,
  "transactionFee": 10000000,
  "estimatedYield": 5000000,
  "yieldPeriod": "monthly",
  "description": "2BR apartment, 45m2, fully furnished",
  "notes": "Rental income property"
}
```

#### D. Create - Cash 🆕
```json
{
  "name": "Emergency Fund",
  "type": "CASH",
  "quantity": 50000000,
  "purchasePrice": 1,
  "purchaseDate": "2024-01-01T00:00:00Z",
  "purchaseCurrency": "IDR",
  "totalCost": 50000000,
  "description": "Savings account",
  "notes": "6 months emergency fund"
}
```

### 3. ✅ Enhanced Asset Update Examples

**New Update Requests:**

#### A. Update - Quantity & Target Price 🆕
```json
{
  "quantity": 0.75,
  "targetPrice": 70000.00,
  "notes": "Increased position & adjusted target"
}
```

#### B. Update - Mark as SOLD 🆕
```json
{
  "status": "SOLD",
  "soldAt": "2026-02-03T15:00:00Z",
  "soldPrice": 65000.00,
  "notes": "Sold at profit, target reached"
}
```

### 4. ✅ Performance Endpoints Enhanced

#### A. Get Asset Performance ✅
- Added detailed description
- Documented what metrics are returned
- Explained currency parameter

**Description:**
> Get detailed performance metrics for a specific asset including:
> - Investment info (purchase price, date, total cost)
> - Current value with real-time prices
> - Profit/Loss calculation
> - ROI and annualized return
> - Smart recommendations
> - Target price analysis

#### B. Get Portfolio Performance - USD ✅
**Description:**
> Get comprehensive portfolio performance dashboard including:
> - Total invested & current value
> - Overall profit/loss & ROI
> - Asset allocation breakdown by type
> - Top gainers & losers (top 5 each)
> - Status summary (active/sold/planned)
> - Performance metrics per asset type

#### C. Get Portfolio Performance - IDR 🆕
Separate example for IDR currency conversion

---

## 📊 Collection Structure

### Total Endpoints: **37** (was 33)

#### Auth (2):
- ✅ Register
- ✅ Login

#### Assets (10): ⬆️ +4 new examples
- ✅ Create - Crypto (Bitcoin)
- 🆕 Create - Stock (Apple)
- 🆕 Create - Real Estate (Property)
- 🆕 Create - Cash
- ✅ List
- ✅ Get
- 🆕 Update - Quantity & Target Price
- 🆕 Update - Mark as SOLD
- ✅ Delete
- ✅ Get Asset Performance

#### Asset Price History (3):
- ✅ Get Price History
- ✅ Record Price
- ✅ Fetch Price

#### Expenses (5):
- ✅ Create, List, Get, Update, Delete

#### Incomes (5):
- ✅ Create, List, Get, Update, Delete

#### Saving Goals (5):
- ✅ Create, List, Get, Update, Delete

#### Prices (5):
- ✅ Crypto BTC/USD
- ✅ Crypto BTC/IDR
- ✅ Crypto ETH/USD
- ✅ Stock AAPL/USD
- ✅ Stock AAPL/IDR

#### Insights (3):
- ✅ Cashflow
- ✅ Cashflow by Month
- ✅ Overview

#### Portfolio (4): ⬆️ +1 new
- ✅ Get Portfolio
- ✅ Get Asset Value
- ✅ Get Portfolio Performance - USD
- 🆕 Get Portfolio Performance - IDR

---

## 🎨 Key Features

### 1. Real-World Examples
Each asset type has realistic example data:
- **Crypto**: Bitcoin with realistic prices
- **Stock**: Apple with dividend info
- **Real Estate**: Indonesian property with rental yield
- **Cash**: Emergency fund in IDR

### 2. Purchase Info Complete
All create requests include:
- ✅ Purchase price
- ✅ Purchase date
- ✅ Purchase currency
- ✅ Total cost
- ✅ Transaction fees
- ✅ Target prices
- ✅ Notes & descriptions

### 3. Status Management
Examples for asset lifecycle:
- ✅ Active assets
- ✅ Sold assets (with sold price & date)
- ✅ Update status tracking

### 4. Multi-Currency Support
Examples for both:
- ✅ USD (international)
- ✅ IDR (Indonesian Rupiah)

---

## 🧪 Testing Workflow

### Recommended Test Sequence:

1. **Authentication Flow**
   ```
   POST /auth/register
   POST /auth/login (save token)
   ```

2. **Create Multiple Assets**
   ```
   POST /assets (Bitcoin - CRYPTO)
   POST /assets (Apple - STOCK)
   POST /assets (Property - REAL_ESTATE)
   POST /assets (Emergency Fund - CASH)
   ```

3. **View Asset Performance**
   ```
   GET /assets/{uuid}/performance
   (Test with each asset UUID)
   ```

4. **View Portfolio Dashboard**
   ```
   GET /portfolio/performance?currency=USD
   GET /portfolio/performance?currency=IDR
   ```

5. **Update Asset**
   ```
   PUT /assets/{uuid} (increase quantity)
   GET /assets/{uuid}/performance (see updated performance)
   ```

6. **Sell Asset**
   ```
   PUT /assets/{uuid} (mark as SOLD)
   GET /portfolio/performance (see status change)
   ```

---

## 📝 Usage Notes

### Variables
The collection uses these variables:
- `baseUrl` - API base URL (default: http://localhost:8080/api/v1)
- `authToken` - JWT token (auto-set after login)
- `assetUuid` - Last created asset UUID (auto-set)
- `expenseUuid` - Last created expense UUID (auto-set)
- `incomeUuid` - Last created income UUID (auto-set)
- `savingGoalUuid` - Last created goal UUID (auto-set)

### Auto-Save Features
Collection has test scripts that automatically:
- Save `authToken` after successful login
- Save `assetUuid` after asset creation
- Save other resource UUIDs for easy testing

### Path Parameters
All endpoints using `:uuid` format for consistency:
```
/assets/:uuid
/assets/:uuid/performance
/assets/:uuid/prices
/portfolio/assets/:uuid
```

---

## ✅ Validation

**JSON Syntax:** ✅ Valid  
**Schema Version:** v2.1.0  
**Sync with Enhancement Plan:** ✅ Complete

---

## 🎯 What's Different from Before

### Before Update:
- ❌ Simple asset creation (no purchase info)
- ❌ Limited to 1 asset type example
- ❌ No performance examples
- ❌ No status management examples
- ❌ No multi-currency examples

### After Update:
- ✅ Complete purchase info in all examples
- ✅ 4 different asset type examples
- ✅ Performance endpoints with descriptions
- ✅ Status lifecycle examples (ACTIVE → SOLD)
- ✅ Multi-currency support (USD & IDR)
- ✅ Real-world realistic data
- ✅ Comprehensive documentation

---

## 🚀 Next Steps

### For Testing:
1. Import updated `knowledge/postman_collection.json` to Postman
2. Set `baseUrl` to your server URL
3. Run "Register" → "Login" to get token
4. Test creating different asset types
5. View performance metrics
6. Test portfolio dashboard

### For Phase 2 (Future):
When implementing Phase 2, add:
- ⚪ Asset Transactions endpoints
- ⚪ Asset Income tracking endpoints
- ⚪ Transaction history examples
- ⚪ Income recording examples

---

## 📚 Documentation References

- Full API Specs: `knowledge/enhancement-plan.md`
- Database Schema: `knowledge/schema.md`
- Implementation Guide: `PHASE1_IMPLEMENTATION.md`
- Project Summary: `knowledge/summary.md`

---

**Status:** ✅ Ready for Testing  
**Collection File:** `knowledge/postman_collection.json`  
**Last Updated:** February 3, 2026
