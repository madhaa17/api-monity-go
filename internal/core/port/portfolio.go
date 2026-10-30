package port

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type PortfolioService interface {
	GetPortfolio(ctx context.Context, userID int64, currency string) (*PortfolioResponse, error)
	GetAssetValue(ctx context.Context, userID int64, assetUUID string, currency string) (*AssetValueResponse, error)
}

type PortfolioResponse struct {
	TotalValue decimal.Decimal      `json:"totalValue"`
	Currency   string               `json:"currency"`
	Assets     []AssetValueResponse `json:"assets"`
	FetchedAt  time.Time            `json:"fetchedAt"`
}

type AssetValueResponse struct {
	UUID         string          `json:"uuid"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	Symbol       string          `json:"symbol,omitempty"`
	Quantity     decimal.Decimal `json:"quantity"`
	CurrentPrice decimal.Decimal `json:"currentPrice"`
	Value        decimal.Decimal `json:"value"`
	Currency     string          `json:"currency"`
	PriceSource  string          `json:"priceSource"`
	// PriceAsOf is when CurrentPrice was fetched from the market; omitted for manual or purchase
	// prices. PriceStale is true when the market source was unreachable and CurrentPrice is the
	// last price fetched earlier.
	PriceAsOf  *time.Time `json:"priceAsOf,omitempty"`
	PriceStale bool       `json:"priceStale,omitempty"`
	// LotSize is the number of units per lot when Quantity is counted in lots (IDX stocks).
	LotSize int64 `json:"lotSize,omitempty"`
}
