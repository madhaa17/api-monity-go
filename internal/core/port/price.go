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
	GetHistoricalCryptoPrice(ctx context.Context, symbol string, timestamp time.Time) (*PriceData, error)
	GetHistoricalCryptoOHLCV(ctx context.Context, symbol string, timeStart, timeEnd time.Time, interval string) ([]OHLCVData, error)
}

type PriceData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetchedAt"`
}

type OHLCVData struct {
	Symbol    string    `json:"symbol"`
	TimeOpen  time.Time `json:"timeOpen"`
	TimeClose time.Time `json:"timeClose"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	MarketCap float64   `json:"marketCap"`
	Currency  string    `json:"currency"`
	Source    string    `json:"source"`
}
