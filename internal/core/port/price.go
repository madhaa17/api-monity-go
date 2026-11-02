package port

import (
	"context"
	"time"
)

const (
	CurrencyUSD = "USD"
	CurrencyIDR = "IDR"

	// DefaultCurrency is IDR — all prices default to Indonesian Rupiah.
	DefaultCurrency = CurrencyIDR
)

type PriceService interface {
	GetCryptoPriceWithCurrency(ctx context.Context, symbol string, currency string) (*PriceData, error)
	GetStockPriceWithCurrency(ctx context.Context, symbol string, currency string) (*PriceData, error)
	GetPriceWithCurrency(ctx context.Context, assetType string, symbol string, currency string) (*PriceData, error)
	GetCryptoPrice(ctx context.Context, symbol string) (*PriceData, error)
	GetStockPrice(ctx context.Context, symbol string) (*PriceData, error)
	GetPrice(ctx context.Context, assetType string, symbol string) (*PriceData, error)
	GetCryptoChart(ctx context.Context, symbol string, currency string, days int) (*ChartResponse, error)
	GetStockChart(ctx context.Context, symbol string, rangeParam string, interval string) (*ChartResponse, error)
	// SearchSymbols finds tickers matching query for assetType (CRYPTO or STOCK).
	SearchSymbols(ctx context.Context, assetType string, query string) ([]SymbolSearchResult, error)
}

// SymbolSearchResult is one ticker match for the asset form's symbol picker. Symbol is the value
// to store on the asset: the exchange-qualified Yahoo symbol for stocks (BBCA.JK, AAPL) and the
// ticker for crypto (BTC).
type SymbolSearchResult struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Exchange string `json:"exchange,omitempty"`
	// Kind is the instrument kind, e.g. EQUITY, ETF, or COIN.
	Kind string `json:"kind,omitempty"`
	// CoinID is the CoinGecko id the crypto ticker resolves to.
	CoinID string `json:"coinId,omitempty"`
	// Rank is the CoinGecko market-cap rank (0 when unknown).
	Rank int `json:"rank,omitempty"`
	// LotSize is the number of units per lot when quantity is counted in lots (100 for IDX).
	LotSize int64 `json:"lotSize,omitempty"`
}

// ChartDataPoint is one point for a line chart (t = Unix second, p = price).
type ChartDataPoint struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

// ChartResponse is the response for chart endpoints (crypto and stock).
type ChartResponse struct {
	Symbol   string           `json:"symbol"`
	Currency string           `json:"currency"`
	Data     []ChartDataPoint `json:"data"`
	// PreviousClose is the close of the session before the chart's, the baseline of an intraday
	// (1d) stock chart; 0 when not given.
	PreviousClose float64 `json:"previousClose,omitempty"`
	// FetchedAt is when the chart was fetched from the price source.
	FetchedAt time.Time `json:"fetchedAt"`
	// Stale is true when the price source could not be reached and this is the last chart
	// fetched earlier.
	Stale bool `json:"stale,omitempty"`
}

type PriceData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetchedAt"`
	// LotSize is the number of units per lot when an asset's quantity is counted in lots
	// (100 for IDX stocks); 0 means the quantity is in units.
	LotSize int64 `json:"lotSize,omitempty"`
	// Stale is true when the price source could not be reached and this is the last price fetched
	// earlier, at FetchedAt.
	Stale bool `json:"stale,omitempty"`
}
