package port

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// AssetPerformanceService handles asset performance calculations
type AssetPerformanceService interface {
	GetAssetPerformance(ctx context.Context, userID int64, assetUUID string, currency string) (*AssetPerformanceResponse, error)
	GetPortfolioPerformance(ctx context.Context, userID int64, currency string) (*PortfolioPerformanceResponse, error)
}

// AssetPerformanceResponse contains performance metrics for a single asset
type AssetPerformanceResponse struct {
	AssetUUID   string              `json:"assetUuid"`
	AssetName   string              `json:"assetName"`
	Type        string              `json:"type"`
	Symbol      string              `json:"symbol,omitempty"`
	Investment  InvestmentInfo      `json:"investment"`
	CurrentData CurrentValueInfo    `json:"currentValue"`
	Performance PerformanceMetrics  `json:"performance"`
	Analysis    PerformanceAnalysis `json:"analysis"`
}

type InvestmentInfo struct {
	Quantity         decimal.Decimal `json:"quantity"`
	PurchasePrice    decimal.Decimal `json:"purchasePrice"`
	PurchaseDate     time.Time       `json:"purchaseDate"`
	TotalCost        decimal.Decimal `json:"totalCost"`
	Currency         string          `json:"currency"`
	TransactionFee   decimal.Decimal `json:"transactionFee,omitempty"`
	DaysSinceHolding int             `json:"daysSinceHolding"`
}

type CurrentValueInfo struct {
	CurrentPrice   decimal.Decimal `json:"currentPrice"`
	CurrentValue   decimal.Decimal `json:"currentValue"`
	PriceChange24h float64         `json:"priceChange24h,omitempty"`
	LastUpdated    time.Time       `json:"lastUpdated"`
}

type PerformanceMetrics struct {
	ProfitLoss        decimal.Decimal `json:"profitLoss"`
	ProfitLossPercent decimal.Decimal `json:"profitLossPercent"`
	ROI               decimal.Decimal `json:"roi"`
	Status            string          `json:"status"`        // profit, loss, break-even
	HoldingPeriod     int             `json:"holdingPeriod"` // days
	AnnualizedReturn  decimal.Decimal `json:"annualizedReturn"`
}

type PerformanceAnalysis struct {
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
	TargetReached  bool   `json:"targetReached"`
}

// PortfolioPerformanceResponse contains overall portfolio performance
type PortfolioPerformanceResponse struct {
	Overview        PortfolioOverview              `json:"overview"`
	AssetAllocation map[string]AssetTypeAllocation `json:"assetAllocation"`
	TopPerformers   PerformersInfo                 `json:"topPerformers"`
	StatusSummary   StatusSummary                  `json:"statusSummary"`
	LastUpdated     time.Time                      `json:"lastUpdated"`
}

type PortfolioOverview struct {
	TotalInvested          decimal.Decimal `json:"totalInvested"`
	CurrentValue           decimal.Decimal `json:"currentValue"`
	TotalProfitLoss        decimal.Decimal `json:"totalProfitLoss"`
	TotalProfitLossPercent decimal.Decimal `json:"totalProfitLossPercent"`
	TotalROI               decimal.Decimal `json:"totalROI"`
	Currency               string          `json:"currency"`
}

type AssetTypeAllocation struct {
	Count         int             `json:"count"`
	TotalInvested decimal.Decimal `json:"totalInvested"`
	CurrentValue  decimal.Decimal `json:"currentValue"`
	ProfitLoss    decimal.Decimal `json:"profitLoss"`
	Percentage    decimal.Decimal `json:"percentage"` // % of total portfolio value
	ROI           decimal.Decimal `json:"roi"`
}

type PerformersInfo struct {
	Gainers []PerformerSummary `json:"gainers"`
	Losers  []PerformerSummary `json:"losers"`
}

type PerformerSummary struct {
	UUID              string          `json:"uuid"`
	Name              string          `json:"name"`
	Type              string          `json:"type"`
	ProfitLossPercent decimal.Decimal `json:"profitLossPercent"`
	ProfitLoss        decimal.Decimal `json:"profitLoss"`
}

type StatusSummary struct {
	Active  int `json:"active"`
	Sold    int `json:"sold"`
	Planned int `json:"planned"`
}
