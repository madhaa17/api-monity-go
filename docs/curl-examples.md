# cURL examples

## Price chart (no auth)

Kedua endpoint chart (crypto dan stock) mengembalikan response dengan format yang sama. Cocok untuk line chart: sumbu X = waktu dari `t`, sumbu Y = harga dari `p`.

### Chart response format

- **Envelope:** `success` (boolean), `message` (string), `data` (object chart).
- **Isi `data`:**
  - `symbol` — ticker yang diminta (contoh: `"BTC"`, `"BBCA"`).
  - `currency` — mata uang harga (contoh: `"IDR"`, `"USD"`).
  - `data` — array titik waktu–harga, urutan kronologis (lama → baru).
- **Setiap elemen di `data.data[]`:**
  - `t` — **Unix timestamp dalam detik** (UTC); untuk sumbu X / label tanggal.
  - `p` — **harga**; untuk sumbu Y.
  - **Crypto:** harga spot pada waktu tersebut (dari CoinGecko market_chart).
  - **Stock:** harga **penutupan (close)** candle untuk periode itu; dengan `interval=1d` = close per hari, `1wk` = per minggu, `1mo` = per bulan.

Contoh response (disederhanakan):

```json
{
  "success": true,
  "message": "stock chart retrieved",
  "data": {
    "symbol": "BBCA",
    "currency": "IDR",
    "data": [
      { "t": 1768269600, "p": 8075 },
      { "t": 1768356000, "p": 8000 },
      { "t": 1768442400, "p": 8075 }
    ]
  }
}
```

Chart dibatasi maksimal 200 titik per response agar response cepat; data tetap mewakili seluruh range. Response dapat dikompresi gzip jika client mengirim header `Accept-Encoding: gzip`.

Query params (days, range, interval) dijelaskan di section curl di bawah.

### Get Crypto Chart

CoinGecko market_chart. Query: `days` (1, 7, 14, 30, 90), `currency` (e.g. idr, usd).

```bash
curl -s "http://localhost:8080/api/v1/prices/crypto/BTC/chart?days=7&currency=idr"
```

With default currency (IDR) and default days (7):

```bash
curl -s "http://localhost:8080/api/v1/prices/crypto/ETH/chart"
```

### Get Stock Chart

Yahoo Finance chart. Query: `range` (1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max), `interval` (1d, 1wk, 1mo). Defaults: range=1mo, interval=1d.

```bash
curl -s "http://localhost:8080/api/v1/prices/stock/BBCA/chart?range=1mo&interval=1d"
```

IDX symbol (auto .JK):

```bash
curl -s "http://localhost:8080/api/v1/prices/stock/BBCA/chart?range=1y&interval=1wk"
```
