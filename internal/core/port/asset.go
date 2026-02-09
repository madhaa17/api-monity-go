package port

import (
	"context"
	"monity/internal/models"
)

type AssetRepository interface {
	Create(ctx context.Context, asset *models.Asset) error
	GetByID(ctx context.Context, id int64) (*models.Asset, error)
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Asset, error)
	ListByUserID(ctx context.Context, userID int64) ([]models.Asset, error)
	Update(ctx context.Context, asset *models.Asset) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type AssetService interface {
	CreateAsset(ctx context.Context, userID int64, req CreateAssetRequest) (*models.Asset, error)
	GetAsset(ctx context.Context, userID int64, uuid string) (*models.Asset, error)
	ListAssets(ctx context.Context, userID int64) ([]models.Asset, error)
	UpdateAsset(ctx context.Context, userID int64, uuid string, req UpdateAssetRequest) (*models.Asset, error)
	DeleteAsset(ctx context.Context, userID int64, uuid string) error
}

type CreateAssetRequest struct {
	Name     string
	Type     models.AssetType
	Quantity float64 // Keeping as float64 for JSON request convenience, will convert to decimal
	Symbol   *string
	
	// Purchase Information (required for new assets)
	PurchasePrice    float64
	PurchaseDate     string // ISO 8601 format
	PurchaseCurrency string
	TotalCost        float64
	
	// Additional Costs (optional)
	TransactionFee  *float64
	MaintenanceCost *float64
	
	// Target & Planning (optional)
	TargetPrice *float64
	TargetDate  *string
	
	// Real Asset Specific (optional)
	EstimatedYield *float64
	YieldPeriod    *string
	
	// Documentation (optional)
	Description *string
	Notes       *string
	
	// Status (defaults to ACTIVE)
	Status *models.AssetStatus
}

type UpdateAssetRequest struct {
	Name     *string
	Type     *models.AssetType
	Quantity *float64
	Symbol   *string
	
	// Purchase Information
	PurchasePrice    *float64
	PurchaseDate     *string
	PurchaseCurrency *string
	TotalCost        *float64
	
	// Additional Costs
	TransactionFee  *float64
	MaintenanceCost *float64
	
	// Target & Planning
	TargetPrice *float64
	TargetDate  *string
	
	// Real Asset Specific
	EstimatedYield *float64
	YieldPeriod    *string
	
	// Documentation
	Description *string
	Notes       *string
	
	// Status
	Status    *models.AssetStatus
	SoldAt    *string
	SoldPrice *float64
}
