package repository

import (
	"context"
	"errors"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
)

type AssetPriceHistoryRepo struct {
	db *gorm.DB
}

func NewAssetPriceHistoryRepository(db *gorm.DB) port.AssetPriceHistoryRepository {
	return &AssetPriceHistoryRepo{db: db}
}

func (r *AssetPriceHistoryRepo) Create(ctx context.Context, history *models.AssetPriceHistory) error {
	result := r.db.WithContext(ctx).Create(history)
	if result.Error != nil {
		return fmt.Errorf("create price history: %w", result.Error)
	}
	return nil
}

func (r *AssetPriceHistoryRepo) ListByAssetID(ctx context.Context, assetID int64, limit int) ([]models.AssetPriceHistory, error) {
	var histories []models.AssetPriceHistory
	
	query := r.db.WithContext(ctx).Where("asset_id = ?", assetID).Order("recorded_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	result := query.Find(&histories)
	if result.Error != nil {
		return nil, fmt.Errorf("list price history: %w", result.Error)
	}
	return histories, nil
}

func (r *AssetPriceHistoryRepo) GetLatestByAssetID(ctx context.Context, assetID int64) (*models.AssetPriceHistory, error) {
	var history models.AssetPriceHistory
	result := r.db.WithContext(ctx).Where("asset_id = ?", assetID).Order("recorded_at desc").First(&history)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest price: %w", result.Error)
	}
	return &history, nil
}
