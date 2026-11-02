package port

import (
	"context"

	"monity/internal/models"
)

// PricedSymbol is a crypto or stock symbol held in at least one asset.
type PricedSymbol struct {
	AssetType models.AssetType
	Symbol    string
}

type MarketDailyCloseRepository interface {
	// ListPricedSymbols returns the distinct crypto and stock symbols of all assets, upper case.
	ListPricedSymbols(ctx context.Context) ([]PricedSymbol, error)
	// Upsert stores closes, replacing any close already stored for the same symbol, currency and day.
	Upsert(ctx context.Context, closes []models.MarketDailyClose) error
}

type DailyCloseService interface {
	// Run records the daily closes at once and then periodically, until ctx is done.
	Run(ctx context.Context)
	// RecordDailyCloses stores the daily closes of the past year for every priced symbol and returns
	// how many were stored.
	RecordDailyCloses(ctx context.Context) (int, error)
}
