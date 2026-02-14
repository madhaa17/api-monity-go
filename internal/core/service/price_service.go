package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/pkg/cache"
)

// cryptoIDMap maps ticker symbols to CoinGecko IDs.
// Add more entries as needed.
var cryptoIDMap = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"SOL":   "solana",
	"USDT":  "tether",
	"BNB":   "binancecoin",
	"XRP":   "ripple",
	"ADA":   "cardano",
	"DOGE":  "dogecoin",
	"AVAX":  "avalanche-2",
	"DOT":   "polkadot",
	"MATIC": "matic-network",
	"LINK":  "chainlink",
	"ATOM":  "cosmos",
	"UNI":   "uniswap",
	"LTC":   "litecoin",
}

type PriceService struct {
	cfg        *config.PriceAPIConfig
	httpClient *http.Client
	cache      cache.Cache
}

func NewPriceService(cfg *config.PriceAPIConfig, c cache.Cache) port.PriceService {
	if c == nil {
		c = cache.NewMemoryCache()
	}
	// Force IPv4 to avoid IPv6 connection issues with some API providers (e.g. CoinGecko/Cloudflare)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &PriceService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		cache: c,
	}
}

func (s *PriceService) GetPrice(ctx context.Context, assetType string, symbol string) (*port.PriceData, error) {
	return s.GetPriceWithCurrency(ctx, assetType, symbol, port.DefaultCurrency)
}

func (s *PriceService) GetPriceWithCurrency(ctx context.Context, assetType string, symbol string, currency string) (*port.PriceData, error) {
	switch strings.ToUpper(assetType) {
	case "CRYPTO":
		return s.GetCryptoPriceWithCurrency(ctx, symbol, currency)
	case "STOCK":
		return s.GetStockPriceWithCurrency(ctx, symbol, currency)
	default:
		return nil, errors.New("unsupported asset type for price lookup")
	}
}

// ---------------------------------------------------------------------------
// Crypto — CoinGecko (free, no API key needed)
// https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd
// ---------------------------------------------------------------------------

func (s *PriceService) GetCryptoPrice(ctx context.Context, symbol string) (*port.PriceData, error) {
	return s.GetCryptoPriceWithCurrency(ctx, symbol, port.DefaultCurrency)
}

func (s *PriceService) GetCryptoPriceWithCurrency(ctx context.Context, symbol string, currency string) (*port.PriceData, error) {
	symbol = strings.ToUpper(symbol)
	currency = strings.ToUpper(currency)
	if currency == "" {
		currency = port.DefaultCurrency
	}

	cacheKey := fmt.Sprintf("crypto:%s:%s", symbol, currency)
	if cached := s.getFromCache(ctx, cacheKey); cached != nil {
		slog.Debug("cache_hit", "key", cacheKey)
		return cached, nil
	}
	slog.Debug("cache_miss", "key", cacheKey)

	// Map ticker to CoinGecko id (e.g. SOL -> solana)
	coinID, ok := cryptoIDMap[symbol]
	if !ok {
		return nil, fmt.Errorf("unsupported crypto symbol: %s (add to cryptoIDMap)", symbol)
	}

	vsCurrency := strings.ToLower(currency)
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=%s", coinID, vsCurrency)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		slog.Warn("price_api_error", "symbol", symbol, "source", "coingecko", "error", err)
		return nil, fmt.Errorf("fetch crypto price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("price_api_error", "symbol", symbol, "source", "coingecko", "status", resp.StatusCode)
		return nil, fmt.Errorf("coingecko API returned status %d", resp.StatusCode)
	}

	// Response: {"solana":{"usd":86.59}}
	var result map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	coinData, ok := result[coinID]
	if !ok {
		return nil, fmt.Errorf("price not found for %s", symbol)
	}

	price, ok := coinData[vsCurrency]
	if !ok {
		return nil, fmt.Errorf("price in %s not found for %s", currency, symbol)
	}

	priceData := &port.PriceData{
		Symbol:    symbol,
		Price:     price,
		Currency:  currency,
		Source:    "coingecko",
		FetchedAt: time.Now(),
	}

	s.setCache(ctx, cacheKey, priceData)
	slog.Info("price_fetched", "symbol", symbol, "price", price, "source", "coingecko")
	return priceData, nil
}

// ---------------------------------------------------------------------------
// Stock — Yahoo Finance (free, no API key needed)
// https://query1.finance.yahoo.com/v8/finance/chart/BBRI.JK?interval=1d&range=1d
// IDX stocks need .JK suffix (e.g. BBRI -> BBRI.JK, BBCA -> BBCA.JK)
// ---------------------------------------------------------------------------

// IDXStocks lists common IDX (Jakarta) stock tickers that need .JK suffix.
// Exported so other services (portfolio) can check lot-based calculation.
var IDXStocks = map[string]bool{
	"BBRI": true, "BBCA": true, "BMRI": true, "BBNI": true, "BRIS": true,
	"TLKM": true, "ASII": true, "UNVR": true, "HMSP": true, "GGRM": true,
	"ICBP": true, "INDF": true, "KLBF": true, "PGAS": true, "SMGR": true,
	"ANTM": true, "PTBA": true, "ADRO": true, "ITMG": true, "INCO": true,
	"EXCL": true, "ISAT": true, "TOWR": true, "MNCN": true, "SIDO": true,
	"EMTK": true, "BUKA": true, "GOTO": true, "ACES": true, "MDKA": true,
}

// IDXLotSize is the number of shares per lot in the Indonesian stock exchange.
const IDXLotSize = 100

// IsIDXStock checks if a ticker is a known IDX stock.
func IsIDXStock(symbol string) bool {
	return IDXStocks[strings.ToUpper(symbol)]
}

func (s *PriceService) GetStockPrice(ctx context.Context, symbol string) (*port.PriceData, error) {
	return s.GetStockPriceWithCurrency(ctx, symbol, port.DefaultCurrency)
}

func (s *PriceService) GetStockPriceWithCurrency(ctx context.Context, symbol string, currency string) (*port.PriceData, error) {
	symbol = strings.ToUpper(symbol)
	currency = strings.ToUpper(currency)
	if currency == "" {
		currency = port.DefaultCurrency
	}

	cacheKey := fmt.Sprintf("stock:%s:%s", symbol, currency)
	if cached := s.getFromCache(ctx, cacheKey); cached != nil {
		slog.Debug("cache_hit", "key", cacheKey)
		return cached, nil
	}
	slog.Debug("cache_miss", "key", cacheKey)

	// Auto-append .JK for known IDX stocks
	yahooSymbol := symbol
	if IDXStocks[symbol] {
		yahooSymbol = symbol + ".JK"
	}

	url := fmt.Sprintf("%s/v8/finance/chart/%s?interval=1d&range=1d", s.cfg.StockAPI, yahooSymbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		slog.Warn("price_api_error", "symbol", yahooSymbol, "source", "yahoo", "error", err)
		return nil, fmt.Errorf("fetch stock price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("price_api_error", "symbol", yahooSymbol, "source", "yahoo", "status", resp.StatusCode)
		return nil, fmt.Errorf("stock API returned status %d for %s", resp.StatusCode, yahooSymbol)
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					Currency           string  `json:"currency"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo finance error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no price data found for %s", yahooSymbol)
	}

	meta := result.Chart.Result[0].Meta
	price := meta.RegularMarketPrice
	sourceCurrency := strings.ToUpper(meta.Currency)

	// Convert currency if needed (e.g. BBRI returns IDR, but user wants USD)
	if currency != sourceCurrency {
		exchangeRate, err := s.getExchangeRate(ctx, sourceCurrency, currency)
		if err != nil {
			// If conversion fails, return in source currency
			return &port.PriceData{
				Symbol:    symbol,
				Price:     price,
				Currency:  sourceCurrency,
				Source:    "yahoo",
				FetchedAt: time.Now(),
			}, nil
		}
		price = price * exchangeRate
	}

	priceData := &port.PriceData{
		Symbol:    symbol,
		Price:     price,
		Currency:  currency,
		Source:    "yahoo",
		FetchedAt: time.Now(),
	}

	s.setCache(ctx, cacheKey, priceData)
	slog.Info("price_fetched", "symbol", symbol, "price", price, "source", "yahoo")
	return priceData, nil
}

// ---------------------------------------------------------------------------
// Exchange rate — Yahoo Finance (e.g. USDIDR=X)
// ---------------------------------------------------------------------------

func (s *PriceService) getExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	cacheKey := fmt.Sprintf("fx:%s:%s", fromCurrency, toCurrency)

	if cached := s.getFromCache(ctx, cacheKey); cached != nil {
		return cached.Price, nil
	}

	fxSymbol := fmt.Sprintf("%s%s=X", fromCurrency, toCurrency)
	url := fmt.Sprintf("%s/v8/finance/chart/%s?interval=1d&range=1d", s.cfg.StockAPI, fxSymbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch exchange rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("exchange rate API returned status %d", resp.StatusCode)
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	if result.Chart.Error != nil {
		return 0, fmt.Errorf("exchange rate error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return 0, fmt.Errorf("exchange rate not found for %s to %s", fromCurrency, toCurrency)
	}

	rate := result.Chart.Result[0].Meta.RegularMarketPrice

	s.setCache(ctx, cacheKey, &port.PriceData{
		Symbol:    fxSymbol,
		Price:     rate,
		Currency:  toCurrency,
		Source:    "yahoo",
		FetchedAt: time.Now(),
	})

	return rate, nil
}

// ---------------------------------------------------------------------------
// Historical — CoinGecko (simplified, no CMC key needed)
// ---------------------------------------------------------------------------

func (s *PriceService) GetHistoricalCryptoPrice(ctx context.Context, symbol string, timestamp time.Time) (*port.PriceData, error) {
	symbol = strings.ToUpper(symbol)

	coinID, ok := cryptoIDMap[symbol]
	if !ok {
		return nil, fmt.Errorf("unsupported crypto symbol: %s", symbol)
	}

	// CoinGecko history endpoint: /coins/{id}/history?date=dd-mm-yyyy
	dateStr := timestamp.UTC().Format("02-01-2006")
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/history?date=%s&localization=false", coinID, dateStr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch historical crypto price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko history API returned status %d", resp.StatusCode)
	}

	var result struct {
		MarketData struct {
			CurrentPrice map[string]float64 `json:"current_price"`
		} `json:"market_data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	price, ok := result.MarketData.CurrentPrice["idr"]
	if !ok {
		return nil, fmt.Errorf("historical price not found for %s at %s", symbol, dateStr)
	}

	return &port.PriceData{
		Symbol:    symbol,
		Price:     price,
		Currency:  port.DefaultCurrency,
		Source:    "coingecko",
		FetchedAt: timestamp,
	}, nil
}

func (s *PriceService) GetHistoricalCryptoOHLCV(ctx context.Context, symbol string, timeStart, timeEnd time.Time, interval string) ([]port.OHLCVData, error) {
	symbol = strings.ToUpper(symbol)

	coinID, ok := cryptoIDMap[symbol]
	if !ok {
		return nil, fmt.Errorf("unsupported crypto symbol: %s", symbol)
	}

	// CoinGecko OHLC endpoint: /coins/{id}/ohlc?vs_currency=usd&days=30
	days := int(timeEnd.Sub(timeStart).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/ohlc?vs_currency=idr&days=%d", coinID, days)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch historical OHLCV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko OHLC API returned status %d", resp.StatusCode)
	}

	// Response: [[timestamp, open, high, low, close], ...]
	var rawData [][]float64
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	ohlcvData := make([]port.OHLCVData, 0, len(rawData))
	for _, candle := range rawData {
		if len(candle) < 5 {
			continue
		}
		ts := time.UnixMilli(int64(candle[0]))
		ohlcvData = append(ohlcvData, port.OHLCVData{
			Symbol:    symbol,
			TimeOpen:  ts,
			TimeClose: ts,
			Open:      candle[1],
			High:      candle[2],
			Low:       candle[3],
			Close:     candle[4],
			Volume:    0, // CoinGecko OHLC doesn't include volume
			Currency:  port.DefaultCurrency,
			Source:    "coingecko",
		})
	}

	return ohlcvData, nil
}

// ---------------------------------------------------------------------------
// Crypto chart — CoinGecko market_chart
// ---------------------------------------------------------------------------

const (
	chartCacheTTL   = 15 * time.Minute
	maxChartPoints  = 200
)

func (s *PriceService) GetCryptoChart(ctx context.Context, symbol string, currency string, days int) (*port.ChartResponse, error) {
	symbol = strings.ToUpper(symbol)
	currency = strings.ToUpper(currency)
	if currency == "" {
		currency = port.DefaultCurrency
	}

	cacheKey := fmt.Sprintf("chart:crypto:%s:%s:%d", symbol, currency, days)
	if cached := s.getChartFromCache(ctx, cacheKey); cached != nil {
		slog.Debug("cache_hit", "key", cacheKey)
		return cached, nil
	}

	coinID, ok := cryptoIDMap[symbol]
	if !ok {
		return nil, fmt.Errorf("unsupported crypto symbol: %s", symbol)
	}

	vsCurrency := strings.ToLower(currency)
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/market_chart?vs_currency=%s&days=%d", coinID, vsCurrency, days)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch crypto chart: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko market_chart API returned status %d", resp.StatusCode)
	}

	var result struct {
		Prices [][]float64 `json:"prices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode chart response: %w", err)
	}

	data := make([]port.ChartDataPoint, 0, len(result.Prices))
	for _, pair := range result.Prices {
		if len(pair) < 2 {
			continue
		}
		ms := int64(pair[0])
		data = append(data, port.ChartDataPoint{T: ms / 1000, P: pair[1]})
	}
	data = downsampleChartData(data, maxChartPoints)
	out := &port.ChartResponse{Symbol: symbol, Currency: currency, Data: data}
	s.setChartCache(ctx, cacheKey, out)
	return out, nil
}

// downsampleChartData returns at most max points with even stride (keeps first and last).
func downsampleChartData(data []port.ChartDataPoint, max int) []port.ChartDataPoint {
	n := len(data)
	if n <= max || max <= 0 {
		return data
	}
	if max == 1 {
		return data[:1]
	}
	out := make([]port.ChartDataPoint, max)
	for i := 0; i < max; i++ {
		idx := i * (n - 1) / (max - 1)
		out[i] = data[idx]
	}
	return out
}

// ---------------------------------------------------------------------------
// Stock chart — Yahoo Finance chart
// ---------------------------------------------------------------------------

func (s *PriceService) GetStockChart(ctx context.Context, symbol string, rangeParam string, interval string) (*port.ChartResponse, error) {
	symbol = strings.ToUpper(symbol)

	yahooSymbol := symbol
	if IDXStocks[symbol] {
		yahooSymbol = symbol + ".JK"
	}

	cacheKey := fmt.Sprintf("chart:stock:%s:%s:%s", yahooSymbol, rangeParam, interval)
	if cached := s.getChartFromCache(ctx, cacheKey); cached != nil {
		slog.Debug("cache_hit", "key", cacheKey)
		return cached, nil
	}

	url := fmt.Sprintf("%s/v8/finance/chart/%s?range=%s&interval=%s", s.cfg.StockAPI, yahooSymbol, rangeParam, interval)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch stock chart: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stock chart API returned status %d", resp.StatusCode)
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Currency string `json:"currency"`
				} `json:"meta"`
				Timestamp []int64   `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error *struct {
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode chart response: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo chart error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data found for %s", yahooSymbol)
	}

	r := result.Chart.Result[0]
	currency := strings.ToUpper(r.Meta.Currency)
	if currency == "" {
		currency = port.DefaultCurrency
	}

	times := r.Timestamp
	quotes := r.Indicators.Quote
	if len(quotes) == 0 || len(quotes[0].Close) == 0 {
		return nil, fmt.Errorf("no quote data for %s", yahooSymbol)
	}
	closes := quotes[0].Close

	n := len(times)
	if len(closes) < n {
		n = len(closes)
	}

	data := make([]port.ChartDataPoint, 0, n)
	for i := 0; i < n; i++ {
		data = append(data, port.ChartDataPoint{T: times[i], P: closes[i]})
	}
	data = downsampleChartData(data, maxChartPoints)
	out := &port.ChartResponse{Symbol: symbol, Currency: currency, Data: data}
	s.setChartCache(ctx, cacheKey, out)
	return out, nil
}

// ---------------------------------------------------------------------------
// Cache helpers
// ---------------------------------------------------------------------------

func (s *PriceService) getFromCache(ctx context.Context, key string) *port.PriceData {
	raw, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil
	}
	var data port.PriceData
	if json.Unmarshal(raw, &data) != nil {
		return nil
	}
	return &data
}

func (s *PriceService) setCache(ctx context.Context, key string, data *port.PriceData) {
	ttl := time.Duration(s.cfg.CacheTTL) * time.Second
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	raw, _ := json.Marshal(data)
	_ = s.cache.Set(ctx, key, raw, ttl)
}

func (s *PriceService) getChartFromCache(ctx context.Context, key string) *port.ChartResponse {
	raw, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil
	}
	var data port.ChartResponse
	if json.Unmarshal(raw, &data) != nil {
		return nil
	}
	return &data
}

func (s *PriceService) setChartCache(ctx context.Context, key string, data *port.ChartResponse) {
	raw, _ := json.Marshal(data)
	_ = s.cache.Set(ctx, key, raw, chartCacheTTL)
}
