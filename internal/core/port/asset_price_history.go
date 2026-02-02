package port

import (
	"context"
	"monity/internal/models"
	"time"
)

type AssetPriceHistoryRepository interface {
	Create(ctx context.Context, history *models.AssetPriceHistory) error
	ListByAssetID(ctx context.Context, assetID int64, limit int) ([]models.AssetPriceHistory, error)
	GetLatestByAssetID(ctx context.Context, assetID int64) (*models.AssetPriceHistory, error)
}

type AssetPriceHistoryService interface {
	RecordPrice(ctx context.Context, userID int64, assetUUID string, req RecordPriceRequest) (*models.AssetPriceHistory, error)
	GetPriceHistory(ctx context.Context, userID int64, assetUUID string, limit int) ([]models.AssetPriceHistory, error)
	FetchAndRecordPrice(ctx context.Context, userID int64, assetUUID string) (*models.AssetPriceHistory, error)
}

type RecordPriceRequest struct {
	Price      float64   `json:"price"`
	Source     string    `json:"source"`
	RecordedAt time.Time `json:"recordedAt"`
}
