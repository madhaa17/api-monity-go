package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/cache"

	"github.com/shopspring/decimal"
	"golang.org/x/sync/singleflight"
)

// cryptoIDMap maps common ticker symbols to CoinGecko IDs without a lookup. Other tickers are
// resolved through the CoinGecko search API (see resolveCoinID).
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
	// flight collapses concurrent cache misses for the same key into one upstream request.
	flight singleflight.Group
}

// shared runs fn once for all concurrent callers with the same key, so a burst of requests for one
// symbol makes a single upstream call. fn runs detached from the caller's cancellation, since other
// callers may still be waiting on its result (the HTTP client timeout still bounds it).
func shared[T any](s *PriceService, ctx context.Context, key string, fn func(ctx context.Context) (T, error)) (T, error) {
	v, err, _ := s.flight.Do(key, func() (any, error) {
		return fn(context.WithoutCancel(ctx))
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return v.(T), nil
}

const (
	// priceRevalidateWindow and chartRevalidateWindow are how old a last-known price or chart may
	// be to be served at once when its cache entry has expired, while it is refreshed in the
	// background (stale-while-revalidate).
	priceRevalidateWindow = 10 * time.Minute
	chartRevalidateWindow = time.Hour
)

// revalidate runs fn for key in the background, joining a refresh already in flight, so the
// caller can answer from the last known data without waiting for the upstream API.
func revalidate[T any](s *PriceService, ctx context.Context, key string, fn func(ctx context.Context) (T, error)) {
	go func() {
		if _, err := shared(s, ctx, key, fn); err != nil {
			slog.Warn("revalidate_failed", "key", key, "error", err)
		}
	}()
}

// recent reports whether data fetched at fetchedAt is within window of now.
func recent(fetchedAt time.Time, window time.Duration) bool {
	return !fetchedAt.IsZero() && time.Since(fetchedAt) < window
}

// yahooTradingPeriod is the regular session Yahoo reports in a chart's meta.currentTradingPeriod.
type yahooTradingPeriod struct {
	Regular struct {
		Start int64 `json:"start"`
		End   int64 `json:"end"`
	} `json:"regular"`
}

const (
	// closedMinTTL is the cache lifetime after a session closes when the next open is unknown
	// (e.g. a weekend, where Yahoo still reports the last session).
	closedMinTTL = time.Hour
	// closedMaxTTL bounds how long data is cached while the market is closed.
	closedMaxTTL = 12 * time.Hour
)

// marketAwareTTL returns openTTL while the market is open (or the session is unknown). While it is
// closed prices cannot change, so data is cached until the next session opens: the reported start
// when it is still ahead, otherwise the same time a day later.
func marketAwareTTL(now time.Time, period yahooTradingPeriod, openTTL time.Duration) time.Duration {
	if period.Regular.Start == 0 || period.Regular.End == 0 {
		return openTTL
	}
	start := time.Unix(period.Regular.Start, 0)
	end := time.Unix(period.Regular.End, 0)
	var ttl time.Duration
	switch {
	case now.Before(start):
		ttl = start.Sub(now)
	case now.Before(end):
		return openTTL
	default:
		ttl = start.Add(24 * time.Hour).Sub(now)
		if ttl < closedMinTTL {
			ttl = closedMinTTL
		}
	}
	return min(max(ttl, openTTL), closedMaxTTL)
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
// Crypto — CoinGecko (free; an optional Demo key raises the rate limit)
// https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd
// ---------------------------------------------------------------------------

const (
	defaultCoinGeckoAPI = "https://api.coingecko.com/api/v3"
	// coinIDCacheTTL is how long a ticker's resolved CoinGecko id is kept.
	coinIDCacheTTL = 24 * time.Hour
	// searchCacheTTL is how long symbol search results are kept.
	searchCacheTTL   = time.Hour
	maxSearchResults = 10
)

// newCoinGeckoRequest builds a GET request for path on the configured CoinGecko API, sending the
// API key when one is set: as a Pro key for pro-api.coingecko.com and as a Demo key otherwise.
func (s *PriceService) newCoinGeckoRequest(ctx context.Context, path string) (*http.Request, error) {
	base := strings.TrimRight(s.cfg.CryptoAPI, "/")
	if base == "" {
		base = defaultCoinGeckoAPI
	}
	req, err := http.NewRequestWithContext(ctx, "GET", base+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if key := s.cfg.CryptoAPIKey; key != "" {
		if strings.Contains(base, "pro-api.") {
			req.Header.Set("x-cg-pro-api-key", key)
		} else {
			req.Header.Set("x-cg-demo-api-key", key)
		}
	}
	return req, nil
}

// resolveCoinID returns the CoinGecko id of a crypto ticker: from cryptoIDMap, else the
// highest-ranked coin CoinGecko lists with exactly that ticker, which is also the coin the
// symbol search offers first.
func (s *PriceService) resolveCoinID(ctx context.Context, symbol string) (string, error) {
	symbol = strings.ToUpper(symbol)
	if id, ok := cryptoIDMap[symbol]; ok {
		return id, nil
	}
	cacheKey := "coinid:" + symbol
	if raw, err := s.cache.Get(ctx, cacheKey); err == nil && len(raw) > 0 {
		return string(raw), nil
	}
	coins, err := s.searchCoinGecko(ctx, symbol)
	if err != nil {
		return "", fmt.Errorf("resolve crypto symbol %s: %w", symbol, err)
	}
	for _, c := range coins {
		if c.Symbol == symbol {
			_ = s.cache.Set(ctx, cacheKey, []byte(c.CoinID), coinIDCacheTTL)
			return c.CoinID, nil
		}
	}
	return "", fmt.Errorf("unsupported crypto symbol: %s", symbol)
}

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
	lastKey := "last:" + cacheKey
	if cached := s.getFromCache(ctx, cacheKey); cached != nil {
		slog.Debug("cache_hit", "key", cacheKey)
		return cached, nil
	}
	slog.Debug("cache_miss", "key", cacheKey)

	refresh := func(ctx context.Context) (*port.PriceData, error) {
		price, err := s.fetchCryptoPrice(ctx, symbol, currency)
		if err != nil {
			return nil, err
		}
		s.setCache(ctx, cacheKey, price)
		s.setCacheTTL(ctx, lastKey, price, lastKnownTTL)
		return price, nil
	}
	if last := s.getFromCache(ctx, lastKey); last != nil && recent(last.FetchedAt, priceRevalidateWindow) {
		revalidate(s, ctx, cacheKey, refresh)
		return last, nil
	}
	price, err := shared(s, ctx, cacheKey, refresh)
	if err != nil {
		if last := s.getFromCache(ctx, lastKey); last != nil {
			slog.Warn("price_stale_fallback", "symbol", symbol, "fetched_at", last.FetchedAt, "error", err)
			last.Stale = true
			return last, nil
		}
		return nil, err
	}
	return price, nil
}

// fetchCryptoPrice gets the current price of symbol in currency from CoinGecko.
func (s *PriceService) fetchCryptoPrice(ctx context.Context, symbol, currency string) (*port.PriceData, error) {
	// Map ticker to CoinGecko id (e.g. SOL -> solana)
	coinID, err := s.resolveCoinID(ctx, symbol)
	if err != nil {
		return nil, err
	}

	vsCurrency := strings.ToLower(currency)
	req, err := s.newCoinGeckoRequest(ctx, fmt.Sprintf("/simple/price?ids=%s&vs_currencies=%s", coinID, vsCurrency))
	if err != nil {
		return nil, err
	}

	body, status, err := s.doWithRetry(req)
	if err != nil {
		slog.Warn("price_api_error", "symbol", symbol, "source", "coingecko", "error", err)
		return nil, fmt.Errorf("fetch crypto price: %w", err)
	}
	if status != http.StatusOK {
		slog.Warn("price_api_error", "symbol", symbol, "source", "coingecko", "status", status)
		return nil, fmt.Errorf("coingecko API returned status %d", status)
	}

	// Response: {"solana":{"usd":86.59}}
	var result map[string]map[string]float64
	if err := json.Unmarshal(body, &result); err != nil {
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

	slog.Info("price_fetched", "symbol", symbol, "price", price, "source", "coingecko")
	return &port.PriceData{
		Symbol:    symbol,
		Price:     price,
		Currency:  currency,
		Source:    "coingecko",
		FetchedAt: time.Now(),
	}, nil
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

// IsIDXStock checks if a ticker is a known IDX stock or carries the IDX (.JK) suffix.
func IsIDXStock(symbol string) bool {
	symbol = strings.ToUpper(symbol)
	return IDXStocks[symbol] || strings.HasSuffix(symbol, ".JK")
}

// lotSizeOf returns the units per lot of an asset whose quantity is counted in lots (IDX stocks),
// or 0. price is the asset's current price, or nil when it could not be fetched; then only known
// IDX tickers (IsIDXStock) are treated as lots.
func lotSizeOf(asset *models.Asset, price *port.PriceData) int64 {
	if asset.Type != models.AssetTypeStock || asset.Symbol == nil {
		return 0
	}
	if price != nil {
		return price.LotSize
	}
	if IsIDXStock(*asset.Symbol) {
		return IDXLotSize
	}
	return 0
}

// effectiveQuantity returns the number of units to value an asset by: lots times lotSizeOf.
func effectiveQuantity(asset *models.Asset, price *port.PriceData) decimal.Decimal {
	if lotSize := lotSizeOf(asset, price); lotSize > 1 {
		return asset.Quantity.Mul(decimal.NewFromInt(lotSize))
	}
	return asset.Quantity
}

// yahooSymbolCandidates returns the Yahoo symbols to try for a ticker, in order. A symbol with an
// exchange suffix (BUMI.JK) is used as is and a known IDX stock gets .JK. Any other bare ticker is
// tried on IDX first and then without a suffix, so both BUMI and AAPL resolve.
func yahooSymbolCandidates(symbol string) []string {
	if strings.Contains(symbol, ".") {
		return []string{symbol}
	}
	if IDXStocks[symbol] {
		return []string{symbol + ".JK"}
	}
	return []string{symbol + ".JK", symbol}
}

// retryDelays are the waits before each retry of a request that failed transiently (network
// error, 429 or 5xx). The free Yahoo and CoinGecko APIs drop or throttle requests from time to time.
var retryDelays = []time.Duration{300 * time.Millisecond, time.Second}

// maxRetryAfter is the longest Retry-After a request waits out; a longer one ends the retries, so
// the caller falls back to the last known data instead of hanging.
const maxRetryAfter = 2 * time.Second

// doWithRetry sends req and returns the body and status, retrying transient failures after each
// of retryDelays, or after the server's Retry-After when that is longer. A 4xx other than 429 is
// returned at once.
func (s *PriceService) doWithRetry(req *http.Request) ([]byte, int, error) {
	for attempt := 0; ; attempt++ {
		body, status, retryAfter, err := s.do(req)
		transient := err != nil || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
		if !transient || attempt >= len(retryDelays) || retryAfter > maxRetryAfter || req.Context().Err() != nil {
			return body, status, err
		}
		select {
		case <-time.After(max(retryDelays[attempt], retryAfter)):
		case <-req.Context().Done():
			return nil, 0, req.Context().Err()
		}
	}
}

// do sends req and returns the body, status and the wait the server asks for in Retry-After.
func (s *PriceService) do(req *http.Request) ([]byte, int, time.Duration, error) {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("read response: %w", err)
	}
	return body, resp.StatusCode, parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()), nil
}

// parseRetryAfter returns the wait a Retry-After header asks for, given in seconds or as an HTTP
// date, or 0 when it is absent or invalid.
func parseRetryAfter(v string, now time.Time) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return max(time.Duration(secs)*time.Second, 0)
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(t.Sub(now), 0)
	}
	return 0
}

// yahooSymbolTTL is how long the Yahoo symbol a bare ticker resolved to (BUMI -> BUMI.JK) is kept.
const yahooSymbolTTL = 30 * 24 * time.Hour

// fetchYahooChart calls the Yahoo chart API with query for the first candidate of symbol that
// Yahoo knows, and returns the body and the Yahoo symbol used. Only a 404 moves on to the next one.
// The candidate that worked is remembered, so later calls skip the ones that did not.
func (s *PriceService) fetchYahooChart(ctx context.Context, symbol, query string) ([]byte, string, error) {
	candidates := yahooSymbolCandidates(symbol)
	resolvedKey := "yahoosym:" + symbol
	probing := len(candidates) > 1
	if probing {
		if raw, err := s.cache.Get(ctx, resolvedKey); err == nil && len(raw) > 0 {
			candidates, probing = []string{string(raw)}, false
		}
	}
	for i, yahooSymbol := range candidates {
		url := fmt.Sprintf("%s/v8/finance/chart/%s?%s", s.cfg.StockAPI, yahooSymbol, query)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, yahooSymbol, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0")

		body, status, err := s.doWithRetry(req)
		if err != nil {
			slog.Warn("price_api_error", "symbol", yahooSymbol, "source", "yahoo", "error", err)
			return nil, yahooSymbol, fmt.Errorf("fetch stock data: %w", err)
		}
		if status == http.StatusNotFound && i < len(candidates)-1 {
			continue
		}
		if status != http.StatusOK {
			slog.Warn("price_api_error", "symbol", yahooSymbol, "source", "yahoo", "status", status)
			return nil, yahooSymbol, fmt.Errorf("stock API returned status %d for %s", status, yahooSymbol)
		}
		if probing {
			_ = s.cache.Set(ctx, resolvedKey, []byte(yahooSymbol), yahooSymbolTTL)
		}
		return body, yahooSymbol, nil
	}
	return nil, symbol, errors.New("no stock symbol to look up")
}

func (s *PriceService) GetStockPrice(ctx context.Context, symbol string) (*port.PriceData, error) {
	return s.GetStockPriceWithCurrency(ctx, symbol, port.DefaultCurrency)
}

// GetStockPriceWithCurrency returns the current stock price from the cache or Yahoo. When Yahoo
// cannot be reached, it returns the last price fetched in the past lastKnownTTL, marked Stale.
func (s *PriceService) GetStockPriceWithCurrency(ctx context.Context, symbol string, currency string) (*port.PriceData, error) {
	symbol = strings.ToUpper(symbol)
	currency = strings.ToUpper(currency)
	if currency == "" {
		currency = port.DefaultCurrency
	}

	// v2: entries carry LotSize; older cached entries without it must not be reused.
	cacheKey := fmt.Sprintf("stock:v2:%s:%s", symbol, currency)
	lastKey := "last:" + cacheKey
	if cached := s.getFromCache(ctx, cacheKey); cached != nil {
		slog.Debug("cache_hit", "key", cacheKey)
		return s.preferConverted(ctx, lastKey, currency, cached), nil
	}
	slog.Debug("cache_miss", "key", cacheKey)

	refresh := func(ctx context.Context) (*port.PriceData, error) {
		price, period, err := s.fetchStockPrice(ctx, symbol, currency)
		if err != nil {
			return nil, err
		}
		if price.Currency != currency {
			// Exchange rate unavailable: keep the unconverted price only briefly, so Yahoo is not
			// called on every request while the conversion is retried soon.
			s.setCacheTTL(ctx, cacheKey, price, s.priceTTL())
			return price, nil
		}
		s.setCacheTTL(ctx, cacheKey, price, marketAwareTTL(time.Now(), period, s.priceTTL()))
		s.setCacheTTL(ctx, lastKey, price, lastKnownTTL)
		return price, nil
	}
	if last := s.getFromCache(ctx, lastKey); last != nil && recent(last.FetchedAt, priceRevalidateWindow) {
		revalidate(s, ctx, cacheKey, refresh)
		return last, nil
	}
	price, err := shared(s, ctx, cacheKey, refresh)
	if err != nil {
		if last := s.getFromCache(ctx, lastKey); last != nil {
			slog.Warn("price_stale_fallback", "symbol", symbol, "fetched_at", last.FetchedAt, "error", err)
			last.Stale = true
			return last, nil
		}
		return nil, err
	}
	return s.preferConverted(ctx, lastKey, currency, price), nil
}

// preferConverted returns price, or, when it is still in its source currency because the exchange
// rate was unavailable, the last price fetched in currency marked Stale, if there is one.
func (s *PriceService) preferConverted(ctx context.Context, lastKey, currency string, price *port.PriceData) *port.PriceData {
	if price.Currency == currency {
		return price
	}
	if last := s.getFromCache(ctx, lastKey); last != nil {
		slog.Warn("price_stale_fallback", "symbol", price.Symbol, "fetched_at", last.FetchedAt, "error", "exchange rate unavailable")
		last.Stale = true
		return last
	}
	return price
}

// fetchStockPrice gets the current price of symbol from Yahoo, converted to currency when possible,
// with the trading session it belongs to.
func (s *PriceService) fetchStockPrice(ctx context.Context, symbol string, currency string) (*port.PriceData, yahooTradingPeriod, error) {
	var period yahooTradingPeriod
	body, yahooSymbol, err := s.fetchYahooChart(ctx, symbol, "interval=1d&range=1d")
	if err != nil {
		return nil, period, err
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice   float64            `json:"regularMarketPrice"`
					Currency             string             `json:"currency"`
					CurrentTradingPeriod yahooTradingPeriod `json:"currentTradingPeriod"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, period, fmt.Errorf("decode response: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, period, fmt.Errorf("yahoo finance error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, period, fmt.Errorf("no price data found for %s", yahooSymbol)
	}

	meta := result.Chart.Result[0].Meta
	period = meta.CurrentTradingPeriod
	price := meta.RegularMarketPrice
	sourceCurrency := strings.ToUpper(meta.Currency)
	var lotSize int64
	if strings.HasSuffix(yahooSymbol, ".JK") {
		lotSize = IDXLotSize
	}

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
				LotSize:   lotSize,
			}, period, nil
		}
		price = price * exchangeRate
	}

	priceData := &port.PriceData{
		Symbol:    symbol,
		Price:     price,
		Currency:  currency,
		Source:    "yahoo",
		FetchedAt: time.Now(),
		LotSize:   lotSize,
	}

	slog.Info("price_fetched", "symbol", symbol, "price", price, "source", "yahoo")
	return priceData, period, nil
}

// ---------------------------------------------------------------------------
// Exchange rate — Yahoo Finance (e.g. USDIDR=X), falling back to open.er-api.com
// ---------------------------------------------------------------------------

const defaultFXAPI = "https://open.er-api.com/v6"

// getExchangeRate returns how many toCurrency one fromCurrency buys, from the cache, Yahoo
// (intraday) or, when Yahoo fails, the daily rates of the FX API.
func (s *PriceService) getExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	cacheKey := fmt.Sprintf("fx:%s:%s", fromCurrency, toCurrency)

	if cached := s.getFromCache(ctx, cacheKey); cached != nil {
		return cached.Price, nil
	}

	source := "yahoo"
	rate, err := s.yahooExchangeRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		slog.Warn("fx_api_error", "from", fromCurrency, "to", toCurrency, "source", source, "error", err)
		source = "er-api"
		var fbErr error
		rate, fbErr = s.fallbackExchangeRate(ctx, fromCurrency, toCurrency)
		if fbErr != nil {
			return 0, fmt.Errorf("%w; fallback: %v", err, fbErr)
		}
	}

	s.setCache(ctx, cacheKey, &port.PriceData{
		Symbol:    fromCurrency + toCurrency,
		Price:     rate,
		Currency:  toCurrency,
		Source:    source,
		FetchedAt: time.Now(),
	})

	return rate, nil
}

func (s *PriceService) yahooExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	fxSymbol := fmt.Sprintf("%s%s=X", fromCurrency, toCurrency)
	url := fmt.Sprintf("%s/v8/finance/chart/%s?interval=1d&range=1d", s.cfg.StockAPI, fxSymbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	body, status, err := s.doWithRetry(req)
	if err != nil {
		return 0, fmt.Errorf("fetch exchange rate: %w", err)
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("exchange rate API returned status %d", status)
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

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	if result.Chart.Error != nil {
		return 0, fmt.Errorf("exchange rate error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 || result.Chart.Result[0].Meta.RegularMarketPrice <= 0 {
		return 0, fmt.Errorf("exchange rate not found for %s to %s", fromCurrency, toCurrency)
	}

	return result.Chart.Result[0].Meta.RegularMarketPrice, nil
}

// fallbackExchangeRate gets the rate from the FX API: GET {FXAPI}/latest/USD returns
// {"result":"success","rates":{"IDR":16250.5,...}}.
func (s *PriceService) fallbackExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	base := strings.TrimRight(s.cfg.FXAPI, "/")
	if base == "" {
		base = defaultFXAPI
	}
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/latest/"+fromCurrency, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	body, status, err := s.doWithRetry(req)
	if err != nil {
		return 0, fmt.Errorf("fetch exchange rate: %w", err)
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("fx API returned status %d", status)
	}

	var result struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	rate, ok := result.Rates[toCurrency]
	if result.Result != "success" || !ok || rate <= 0 {
		return 0, fmt.Errorf("exchange rate not found for %s to %s", fromCurrency, toCurrency)
	}
	return rate, nil
}

// ---------------------------------------------------------------------------
// Crypto chart — CoinGecko market_chart
// ---------------------------------------------------------------------------

const (
	chartCacheTTL = 15 * time.Minute
	// intradayChartTTL is how long an intraday (minute or hour bars) stock chart is cached while
	// its market is open.
	intradayChartTTL = 2 * time.Minute
	// maxChartPoints caps a chart's size; daily data for 1Y (~250 trading days) fits untouched.
	maxChartPoints = 500
	// lastKnownTTL is how long the last successfully fetched stock price and chart stay available
	// as a fallback for when Yahoo cannot be reached.
	lastKnownTTL = 30 * 24 * time.Hour
)

// GetCryptoChart returns a coin's price history from the cache or CoinGecko. When CoinGecko cannot
// be reached, it returns the last chart fetched in the past lastKnownTTL, marked Stale.
func (s *PriceService) GetCryptoChart(ctx context.Context, symbol string, currency string, days int) (*port.ChartResponse, error) {
	symbol = strings.ToUpper(symbol)
	currency = strings.ToUpper(currency)
	if currency == "" {
		currency = port.DefaultCurrency
	}

	cacheKey := fmt.Sprintf("chart:crypto:%s:%s:%d", symbol, currency, days)
	lastKey := "last:" + cacheKey
	if cached := s.getChartFromCache(ctx, cacheKey); cached != nil {
		slog.Debug("cache_hit", "key", cacheKey)
		return cached, nil
	}

	refresh := func(ctx context.Context) (*port.ChartResponse, error) {
		chart, err := s.fetchCryptoChart(ctx, symbol, currency, days)
		if err != nil {
			return nil, err
		}
		s.setChartCache(ctx, cacheKey, chart, cryptoChartTTL(days))
		s.setChartCache(ctx, lastKey, chart, lastKnownTTL)
		return chart, nil
	}
	if last := s.getChartFromCache(ctx, lastKey); last != nil && recent(last.FetchedAt, chartRevalidateWindow) {
		revalidate(s, ctx, cacheKey, refresh)
		return last, nil
	}
	chart, err := shared(s, ctx, cacheKey, refresh)
	if err != nil {
		if last := s.getChartFromCache(ctx, lastKey); last != nil {
			slog.Warn("chart_stale_fallback", "symbol", symbol, "error", err)
			last.Stale = true
			return last, nil
		}
		return nil, err
	}
	return chart, nil
}

func (s *PriceService) fetchCryptoChart(ctx context.Context, symbol, currency string, days int) (*port.ChartResponse, error) {
	coinID, err := s.resolveCoinID(ctx, symbol)
	if err != nil {
		return nil, err
	}

	vsCurrency := strings.ToLower(currency)
	req, err := s.newCoinGeckoRequest(ctx, fmt.Sprintf("/coins/%s/market_chart?vs_currency=%s&days=%d", coinID, vsCurrency, days))
	if err != nil {
		return nil, err
	}

	body, status, err := s.doWithRetry(req)
	if err != nil {
		slog.Warn("price_api_error", "symbol", symbol, "source", "coingecko", "error", err)
		return nil, fmt.Errorf("fetch crypto chart: %w", err)
	}
	if status != http.StatusOK {
		slog.Warn("price_api_error", "symbol", symbol, "source", "coingecko", "status", status)
		return nil, fmt.Errorf("coingecko market_chart API returned status %d", status)
	}

	var result struct {
		Prices [][]float64 `json:"prices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
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
	return &port.ChartResponse{Symbol: symbol, Currency: currency, Data: data, FetchedAt: time.Now()}, nil
}

// cryptoChartTTL is how long a crypto chart of the given days is cached. Crypto trades around the
// clock, so it depends only on the chart's granularity: 5-minute points for a day, hourly up to
// 90 days, daily beyond.
func cryptoChartTTL(days int) time.Duration {
	switch {
	case days <= 1:
		return 5 * time.Minute
	case days <= 90:
		return chartCacheTTL
	default:
		return time.Hour
	}
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

// dailyChartMonths are the daily-chart ranges cut from one cached 1y daily chart, so switching
// between them never calls Yahoo again. 0 means the last five trading days (Yahoo's 5d).
var dailyChartMonths = map[string]int{"5d": 0, "1mo": 1, "3mo": 3, "6mo": 6, "1y": 12}

// GetStockChart returns a stock's price history from the cache or Yahoo. When Yahoo cannot be
// reached, it returns the last chart fetched in the past lastKnownTTL, marked Stale.
func (s *PriceService) GetStockChart(ctx context.Context, symbol string, rangeParam string, interval string) (*port.ChartResponse, error) {
	symbol = strings.ToUpper(symbol)

	months, sliced := dailyChartMonths[rangeParam]
	if !sliced || interval != "1d" {
		return s.getStockChart(ctx, symbol, rangeParam, interval)
	}
	chart, err := s.getStockChart(ctx, symbol, "1y", interval)
	if err != nil {
		return nil, err
	}
	return sliceChart(chart, months), nil
}

// sliceChart keeps the points of a daily chart in the last months before its last point, or the
// last five points when months is 0.
func sliceChart(chart *port.ChartResponse, months int) *port.ChartResponse {
	data := chart.Data
	if len(data) == 0 || months >= 12 {
		return chart
	}
	from := max(len(data)-5, 0)
	if months > 0 {
		cutoff := time.Unix(data[len(data)-1].T, 0).AddDate(0, -months, 0).Unix()
		from = sort.Search(len(data), func(i int) bool { return data[i].T >= cutoff })
	}
	out := *chart
	out.Data = data[from:]
	return &out
}

func (s *PriceService) getStockChart(ctx context.Context, symbol string, rangeParam string, interval string) (*port.ChartResponse, error) {
	cacheKey := fmt.Sprintf("chart:stock:%s:%s:%s", symbol, rangeParam, interval)
	lastKey := "last:" + cacheKey
	if cached := s.getChartFromCache(ctx, cacheKey); cached != nil {
		slog.Debug("cache_hit", "key", cacheKey)
		return cached, nil
	}

	refresh := func(ctx context.Context) (*port.ChartResponse, error) {
		chart, period, err := s.fetchStockChart(ctx, symbol, rangeParam, interval)
		if err != nil {
			return nil, err
		}
		openTTL := chartCacheTTL
		if isIntradayInterval(interval) {
			openTTL = intradayChartTTL
		}
		s.setChartCache(ctx, cacheKey, chart, marketAwareTTL(time.Now(), period, openTTL))
		s.setChartCache(ctx, lastKey, chart, lastKnownTTL)
		return chart, nil
	}
	if last := s.getChartFromCache(ctx, lastKey); last != nil && recent(last.FetchedAt, chartRevalidateWindow) {
		revalidate(s, ctx, cacheKey, refresh)
		return last, nil
	}
	chart, err := shared(s, ctx, cacheKey, refresh)
	if err != nil {
		if last := s.getChartFromCache(ctx, lastKey); last != nil {
			slog.Warn("chart_stale_fallback", "symbol", symbol, "error", err)
			last.Stale = true
			return last, nil
		}
		return nil, err
	}
	return chart, nil
}

// isIntradayInterval reports whether a Yahoo interval is minutes or hours (5m, 1h), not days or more.
func isIntradayInterval(interval string) bool {
	return strings.HasSuffix(interval, "m") && !strings.HasSuffix(interval, "mo") || strings.HasSuffix(interval, "h")
}

func (s *PriceService) fetchStockChart(ctx context.Context, symbol string, rangeParam string, interval string) (*port.ChartResponse, yahooTradingPeriod, error) {
	var period yahooTradingPeriod
	body, yahooSymbol, err := s.fetchYahooChart(ctx, symbol, fmt.Sprintf("range=%s&interval=%s", rangeParam, interval))
	if err != nil {
		return nil, period, err
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Currency             string             `json:"currency"`
					CurrentTradingPeriod yahooTradingPeriod `json:"currentTradingPeriod"`
					ChartPreviousClose   float64            `json:"chartPreviousClose"`
				} `json:"meta"`
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						// Yahoo sends null for bars without trades (holidays, suspensions).
						Close []*float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error *struct {
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, period, fmt.Errorf("decode chart response: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, period, fmt.Errorf("yahoo chart error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, period, fmt.Errorf("no chart data found for %s", yahooSymbol)
	}

	r := result.Chart.Result[0]
	period = r.Meta.CurrentTradingPeriod
	currency := strings.ToUpper(r.Meta.Currency)
	if currency == "" {
		currency = port.DefaultCurrency
	}

	times := r.Timestamp
	quotes := r.Indicators.Quote
	if len(quotes) == 0 || len(quotes[0].Close) == 0 {
		return nil, period, fmt.Errorf("no quote data for %s", yahooSymbol)
	}
	closes := quotes[0].Close

	n := len(times)
	if len(closes) < n {
		n = len(closes)
	}

	data := make([]port.ChartDataPoint, 0, n)
	for i := 0; i < n; i++ {
		if closes[i] == nil {
			continue
		}
		data = append(data, port.ChartDataPoint{T: times[i], P: *closes[i]})
	}
	if len(data) == 0 {
		return nil, period, fmt.Errorf("no chart data found for %s", yahooSymbol)
	}
	data = downsampleChartData(data, maxChartPoints)
	out := &port.ChartResponse{Symbol: symbol, Currency: currency, Data: data, FetchedAt: time.Now()}
	// Yahoo's chartPreviousClose is the close before the range starts; only for 1d is that the
	// previous session's close, the baseline of the day's change.
	if rangeParam == "1d" {
		out.PreviousClose = r.Meta.ChartPreviousClose
	}
	return out, period, nil
}

// ---------------------------------------------------------------------------
// Symbol search — Yahoo Finance (stocks) and CoinGecko (crypto)
// ---------------------------------------------------------------------------

// SearchSymbols returns up to maxSearchResults tickers matching query, so the user picks the exact
// instrument instead of typing a ticker that may resolve to a different one (BBCA is a US ETF on
// Yahoo; the Jakarta stock is BBCA.JK).
func (s *PriceService) SearchSymbols(ctx context.Context, assetType string, query string) ([]port.SymbolSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []port.SymbolSearchResult{}, nil
	}
	switch strings.ToUpper(assetType) {
	case "CRYPTO":
		return s.searchCoinGecko(ctx, query)
	case "STOCK":
		return s.searchYahoo(ctx, query)
	default:
		return nil, errors.New("unsupported asset type for symbol search")
	}
}

func (s *PriceService) getSearchFromCache(ctx context.Context, key string) ([]port.SymbolSearchResult, bool) {
	raw, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, false
	}
	var out []port.SymbolSearchResult
	if json.Unmarshal(raw, &out) != nil {
		return nil, false
	}
	return out, true
}

func (s *PriceService) setSearchCache(ctx context.Context, key string, results []port.SymbolSearchResult) {
	raw, _ := json.Marshal(results)
	_ = s.cache.Set(ctx, key, raw, searchCacheTTL)
}

// searchCoinGecko returns the coins matching query, best market cap first. Symbol is upper case.
func (s *PriceService) searchCoinGecko(ctx context.Context, query string) ([]port.SymbolSearchResult, error) {
	cacheKey := "search:crypto:" + strings.ToLower(query)
	if cached, ok := s.getSearchFromCache(ctx, cacheKey); ok {
		return cached, nil
	}

	req, err := s.newCoinGeckoRequest(ctx, "/search?query="+url.QueryEscape(query))
	if err != nil {
		return nil, err
	}
	body, status, err := s.doWithRetry(req)
	if err != nil {
		slog.Warn("price_api_error", "query", query, "source", "coingecko", "error", err)
		return nil, fmt.Errorf("search crypto: %w", err)
	}
	if status != http.StatusOK {
		slog.Warn("price_api_error", "query", query, "source", "coingecko", "status", status)
		return nil, fmt.Errorf("coingecko search API returned status %d", status)
	}

	var result struct {
		Coins []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Symbol        string `json:"symbol"`
			MarketCapRank int    `json:"market_cap_rank"`
		} `json:"coins"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	out := make([]port.SymbolSearchResult, 0, maxSearchResults)
	for _, c := range result.Coins {
		out = append(out, port.SymbolSearchResult{
			Symbol: strings.ToUpper(c.Symbol),
			Name:   c.Name,
			Kind:   "COIN",
			CoinID: c.ID,
			Rank:   c.MarketCapRank,
		})
	}
	// Ranked coins first, by rank; unranked ones keep CoinGecko's order after them.
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := out[i].Rank, out[j].Rank
		if ri == 0 || rj == 0 {
			return ri != 0 && rj == 0
		}
		return ri < rj
	})
	// An asset stores only the ticker, which resolveCoinID maps to the best-ranked coin with that
	// ticker, so offer only that coin per ticker. Mapped tickers use the cryptoIDMap coin.
	seen := make(map[string]bool, len(out))
	deduped := out[:0]
	for _, c := range out {
		if seen[c.Symbol] {
			continue
		}
		if id, ok := cryptoIDMap[c.Symbol]; ok && id != c.CoinID {
			continue
		}
		seen[c.Symbol] = true
		deduped = append(deduped, c)
	}
	out = deduped
	if len(out) > maxSearchResults {
		out = out[:maxSearchResults]
	}
	s.setSearchCache(ctx, cacheKey, out)
	return out, nil
}

// yahooSearchKinds are the Yahoo quote types offered for stock assets.
var yahooSearchKinds = map[string]bool{"EQUITY": true, "ETF": true}

// searchYahoo returns the stocks and ETFs matching query, IDX (.JK) listings first.
func (s *PriceService) searchYahoo(ctx context.Context, query string) ([]port.SymbolSearchResult, error) {
	cacheKey := "search:stock:" + strings.ToLower(query)
	if cached, ok := s.getSearchFromCache(ctx, cacheKey); ok {
		return cached, nil
	}

	searchURL := fmt.Sprintf("%s/v1/finance/search?q=%s&quotesCount=20&newsCount=0&listsCount=0",
		s.cfg.StockAPI, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	body, status, err := s.doWithRetry(req)
	if err != nil {
		slog.Warn("price_api_error", "query", query, "source", "yahoo", "error", err)
		return nil, fmt.Errorf("search stock: %w", err)
	}
	if status != http.StatusOK {
		slog.Warn("price_api_error", "query", query, "source", "yahoo", "status", status)
		return nil, fmt.Errorf("stock search API returned status %d", status)
	}

	var result struct {
		Quotes []struct {
			Symbol    string `json:"symbol"`
			ShortName string `json:"shortname"`
			LongName  string `json:"longname"`
			ExchDisp  string `json:"exchDisp"`
			QuoteType string `json:"quoteType"`
		} `json:"quotes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	out := make([]port.SymbolSearchResult, 0, maxSearchResults)
	for _, q := range result.Quotes {
		if !yahooSearchKinds[q.QuoteType] || q.Symbol == "" {
			continue
		}
		name := q.LongName
		if name == "" {
			name = q.ShortName
		}
		r := port.SymbolSearchResult{
			Symbol:   strings.ToUpper(q.Symbol),
			Name:     name,
			Exchange: q.ExchDisp,
			Kind:     q.QuoteType,
		}
		if strings.HasSuffix(r.Symbol, ".JK") {
			r.LotSize = IDXLotSize
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].LotSize == IDXLotSize && out[j].LotSize != IDXLotSize
	})
	if len(out) > maxSearchResults {
		out = out[:maxSearchResults]
	}
	s.setSearchCache(ctx, cacheKey, out)
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
	s.setCacheTTL(ctx, key, data, s.priceTTL())
}

// priceTTL is how long a current price is cached while its market is open.
func (s *PriceService) priceTTL() time.Duration {
	ttl := time.Duration(s.cfg.CacheTTL) * time.Second
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return ttl
}

func (s *PriceService) setCacheTTL(ctx context.Context, key string, data *port.PriceData, ttl time.Duration) {
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

func (s *PriceService) setChartCache(ctx context.Context, key string, data *port.ChartResponse, ttl time.Duration) {
	raw, _ := json.Marshal(data)
	_ = s.cache.Set(ctx, key, raw, ttl)
}
