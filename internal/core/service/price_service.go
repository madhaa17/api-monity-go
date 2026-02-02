package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"monity/internal/config"
	"monity/internal/core/port"
)

type PriceService struct {
	cfg        *config.PriceAPIConfig
	httpClient *http.Client
	cache      map[string]*cachedPrice
	cacheMu    sync.RWMutex
}

type cachedPrice struct {
	data      *port.PriceData
	expiresAt time.Time
}

func NewPriceService(cfg *config.PriceAPIConfig) port.PriceService {
	transport := &http.Transport{
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
		cache: make(map[string]*cachedPrice),
	}
}

func (s *PriceService) GetPrice(ctx context.Context, assetType string, symbol string) (*port.PriceData, error) {
	return s.GetPriceWithCurrency(ctx, assetType, symbol, port.CurrencyUSD)
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

func (s *PriceService) GetCryptoPrice(ctx context.Context, symbol string) (*port.PriceData, error) {
	return s.GetCryptoPriceWithCurrency(ctx, symbol, port.CurrencyUSD)
}

func (s *PriceService) GetCryptoPriceWithCurrency(ctx context.Context, symbol string, currency string) (*port.PriceData, error) {
	symbol = strings.ToUpper(symbol)
	currency = strings.ToUpper(currency)

	if currency == "" {
		currency = port.CurrencyUSD
	}

	cacheKey := fmt.Sprintf("crypto:%s:%s", symbol, currency)

	if cached := s.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}

	url := fmt.Sprintf("%s/v2/cryptocurrency/quotes/latest?symbol=%s&convert=%s", s.cfg.CryptoAPI, symbol, currency)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-CMC_PRO_API_KEY", s.cfg.CryptoAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch crypto price: %w", err)
	}
	defer resp.Body.Close()

	if err := s.handleCMCError(resp); err != nil {
		return nil, err
	}

	var result struct {
		Status struct {
			ErrorCode    int    `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		} `json:"status"`
		Data map[string][]struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
			Quote  map[string]struct {
				Price            float64 `json:"price"`
				PercentChange24h float64 `json:"percent_change_24h"`
				MarketCap        float64 `json:"market_cap"`
				Volume24h        float64 `json:"volume_24h"`
			} `json:"quote"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Status.ErrorCode != 0 {
		return nil, fmt.Errorf("CoinMarketCap API error: %s", result.Status.ErrorMessage)
	}

	coins, ok := result.Data[symbol]
	if !ok || len(coins) == 0 {
		return nil, fmt.Errorf("price not found for %s", symbol)
	}

	coinData := coins[0]

	quote, ok := coinData.Quote[currency]
	if !ok {
		return nil, fmt.Errorf("price in %s not found for %s", currency, symbol)
	}

	priceData := &port.PriceData{
		Symbol:    symbol,
		Price:     quote.Price,
		Currency:  currency,
		Source:    "coinmarketcap",
		FetchedAt: time.Now(),
	}

	s.setCache(cacheKey, priceData)

	return priceData, nil
}

func (s *PriceService) GetStockPrice(ctx context.Context, symbol string) (*port.PriceData, error) {
	return s.GetStockPriceWithCurrency(ctx, symbol, port.CurrencyUSD)
}

func (s *PriceService) GetStockPriceWithCurrency(ctx context.Context, symbol string, currency string) (*port.PriceData, error) {
	symbol = strings.ToUpper(symbol)
	currency = strings.ToUpper(currency)

	if currency == "" {
		currency = port.CurrencyUSD
	}

	cacheKey := fmt.Sprintf("stock:%s:%s", symbol, currency)

	if cached := s.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}

	url := fmt.Sprintf("%s/v8/finance/chart/%s?interval=1d&range=1d", s.cfg.StockAPI, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch stock price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stock API returned status %d", resp.StatusCode)
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
		return nil, fmt.Errorf("no price data found for %s", symbol)
	}

	meta := result.Chart.Result[0].Meta
	price := meta.RegularMarketPrice
	sourceCurrency := strings.ToUpper(meta.Currency)

	if currency != sourceCurrency {
		exchangeRate, err := s.getExchangeRate(ctx, sourceCurrency, currency)
		if err != nil {
			return nil, fmt.Errorf("get exchange rate: %w", err)
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

	s.setCache(cacheKey, priceData)

	return priceData, nil
}

func (s *PriceService) getExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	cacheKey := fmt.Sprintf("fx:%s:%s", fromCurrency, toCurrency)

	if cached := s.getFromCache(cacheKey); cached != nil {
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

	s.setCache(cacheKey, &port.PriceData{
		Symbol:    fxSymbol,
		Price:     rate,
		Currency:  toCurrency,
		Source:    "yahoo",
		FetchedAt: time.Now(),
	})

	return rate, nil
}

func (s *PriceService) getFromCache(key string) *port.PriceData {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	cached, ok := s.cache[key]
	if !ok {
		return nil
	}

	if time.Now().After(cached.expiresAt) {
		return nil
	}

	return cached.data
}

func (s *PriceService) setCache(key string, data *port.PriceData) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cache[key] = &cachedPrice{
		data:      data,
		expiresAt: time.Now().Add(time.Duration(s.cfg.CacheTTL) * time.Second),
	}
}

func (s *PriceService) GetHistoricalCryptoPrice(ctx context.Context, symbol string, timestamp time.Time) (*port.PriceData, error) {
	symbol = strings.ToUpper(symbol)
	timeStr := timestamp.UTC().Format(time.RFC3339)

	url := fmt.Sprintf("%s/v2/cryptocurrency/quotes/historical?symbol=%s&time_start=%s&time_end=%s&count=1&interval=hourly&convert=USD",
		s.cfg.CryptoAPI, symbol, timeStr, timeStr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-CMC_PRO_API_KEY", s.cfg.CryptoAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch historical crypto price: %w", err)
	}
	defer resp.Body.Close()

	if err := s.handleCMCError(resp); err != nil {
		return nil, err
	}

	var result struct {
		Status struct {
			ErrorCode    int    `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		} `json:"status"`
		Data map[string][]struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
			Quotes []struct {
				Timestamp time.Time `json:"timestamp"`
				Quote     struct {
					USD struct {
						Price     float64 `json:"price"`
						Volume24h float64 `json:"volume_24h"`
						MarketCap float64 `json:"market_cap"`
					} `json:"USD"`
				} `json:"quote"`
			} `json:"quotes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Status.ErrorCode != 0 {
		return nil, fmt.Errorf("CoinMarketCap API error: %s", result.Status.ErrorMessage)
	}

	coins, ok := result.Data[symbol]
	if !ok || len(coins) == 0 {
		return nil, fmt.Errorf("historical price not found for %s", symbol)
	}

	coinData := coins[0]
	if len(coinData.Quotes) == 0 {
		return nil, fmt.Errorf("no historical quotes found for %s at %s", symbol, timeStr)
	}

	quote := coinData.Quotes[0]

	return &port.PriceData{
		Symbol:    symbol,
		Price:     quote.Quote.USD.Price,
		Currency:  "USD",
		Source:    "coinmarketcap",
		FetchedAt: quote.Timestamp,
	}, nil
}

func (s *PriceService) GetHistoricalCryptoOHLCV(ctx context.Context, symbol string, timeStart, timeEnd time.Time, interval string) ([]port.OHLCVData, error) {
	symbol = strings.ToUpper(symbol)

	if interval == "" {
		interval = "daily"
	}

	timeStartStr := timeStart.UTC().Format(time.RFC3339)
	timeEndStr := timeEnd.UTC().Format(time.RFC3339)

	url := fmt.Sprintf("%s/v2/cryptocurrency/ohlcv/historical?symbol=%s&time_start=%s&time_end=%s&time_period=%s&convert=USD",
		s.cfg.CryptoAPI, symbol, timeStartStr, timeEndStr, interval)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-CMC_PRO_API_KEY", s.cfg.CryptoAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch historical OHLCV: %w", err)
	}
	defer resp.Body.Close()

	if err := s.handleCMCError(resp); err != nil {
		return nil, err
	}

	var result struct {
		Status struct {
			ErrorCode    int    `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		} `json:"status"`
		Data map[string][]struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
			Quotes []struct {
				TimeOpen  time.Time `json:"time_open"`
				TimeClose time.Time `json:"time_close"`
				TimeHigh  time.Time `json:"time_high"`
				TimeLow   time.Time `json:"time_low"`
				Quote     struct {
					USD struct {
						Open      float64 `json:"open"`
						High      float64 `json:"high"`
						Low       float64 `json:"low"`
						Close     float64 `json:"close"`
						Volume    float64 `json:"volume"`
						MarketCap float64 `json:"market_cap"`
					} `json:"USD"`
				} `json:"quote"`
			} `json:"quotes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Status.ErrorCode != 0 {
		return nil, fmt.Errorf("CoinMarketCap API error: %s", result.Status.ErrorMessage)
	}

	coins, ok := result.Data[symbol]
	if !ok || len(coins) == 0 {
		return nil, fmt.Errorf("historical OHLCV not found for %s", symbol)
	}

	coinData := coins[0]
	ohlcvData := make([]port.OHLCVData, 0, len(coinData.Quotes))

	for _, quote := range coinData.Quotes {
		ohlcvData = append(ohlcvData, port.OHLCVData{
			Symbol:    symbol,
			TimeOpen:  quote.TimeOpen,
			TimeClose: quote.TimeClose,
			Open:      quote.Quote.USD.Open,
			High:      quote.Quote.USD.High,
			Low:       quote.Quote.USD.Low,
			Close:     quote.Quote.USD.Close,
			Volume:    quote.Quote.USD.Volume,
			MarketCap: quote.Quote.USD.MarketCap,
			Currency:  "USD",
			Source:    "coinmarketcap",
		})
	}

	return ohlcvData, nil
}

func (s *PriceService) handleCMCError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("bad request: invalid parameters")
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: invalid or missing API key")
	case http.StatusPaymentRequired:
		return fmt.Errorf("payment required: API subscription issue")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: API plan doesn't support this endpoint")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited, please try again later")
	case http.StatusInternalServerError:
		return fmt.Errorf("internal server error")
	default:
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
}

