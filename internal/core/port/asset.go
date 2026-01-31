package port

import (
	"context"
	"monity/internal/models"
)

type AssetRepository interface {
	Create(ctx context.Context, asset *models.Asset) error
	GetByID(ctx context.Context, id int64, userID int64) (*models.Asset, error)
	ListByUserID(ctx context.Context, userID int64) ([]models.Asset, error)
	Update(ctx context.Context, asset *models.Asset) error
	Delete(ctx context.Context, id int64, userID int64) error
}

type AssetService interface {
	CreateAsset(ctx context.Context, userID int64, req CreateAssetRequest) (*models.Asset, error)
	GetAsset(ctx context.Context, userID int64, assetID int64) (*models.Asset, error)
	ListAssets(ctx context.Context, userID int64) ([]models.Asset, error)
	UpdateAsset(ctx context.Context, userID int64, assetID int64, req UpdateAssetRequest) (*models.Asset, error)
	DeleteAsset(ctx context.Context, userID int64, assetID int64) error
}

type CreateAssetRequest struct {
	Name     string
	Type     models.AssetType
	Quantity float64 // Keeping as float64 for JSON request convenience, will convert to decimal
	Symbol   *string
}

type UpdateAssetRequest struct {
	Name     *string
	Type     *models.AssetType
	Quantity *float64
	Symbol   *string
}
