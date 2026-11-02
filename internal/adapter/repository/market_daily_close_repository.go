package repository

import (
	"context"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MarketDailyCloseRepo struct {
	db *gorm.DB
}

func NewMarketDailyCloseRepository(db *gorm.DB) port.MarketDailyCloseRepository {
	return &MarketDailyCloseRepo{db: db}
}

func (r *MarketDailyCloseRepo) ListPricedSymbols(ctx context.Context) ([]port.PricedSymbol, error) {
	var out []port.PricedSymbol
	result := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT type AS asset_type, upper(trim(symbol)) AS symbol
		FROM assets
		WHERE type IN (?, ?) AND symbol IS NOT NULL AND trim(symbol) <> ''
		ORDER BY 1, 2`, models.AssetTypeCrypto, models.AssetTypeStock).Scan(&out)
	if result.Error != nil {
		return nil, fmt.Errorf("list priced symbols: %w", result.Error)
	}
	return out, nil
}

func (r *MarketDailyCloseRepo) Upsert(ctx context.Context, closes []models.MarketDailyClose) error {
	if len(closes) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "asset_type"}, {Name: "symbol"}, {Name: "currency"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"close", "source", "recorded_at"}),
	}).CreateInBatches(closes, 500)
	if result.Error != nil {
		return fmt.Errorf("upsert daily closes: %w", result.Error)
	}
	return nil
}
