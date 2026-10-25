package repository

import (
	"context"
	"errors"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AssetRepo struct {
	db *gorm.DB
}

func NewAssetRepository(db *gorm.DB) port.AssetRepository {
	return &AssetRepo{db: db}
}

func (r *AssetRepo) Create(ctx context.Context, asset *models.Asset) error {
	result := conn(ctx, r.db).Create(asset)
	if result.Error != nil {
		return fmt.Errorf("create asset: %w", result.Error)
	}
	return nil
}

func (r *AssetRepo) GetByID(ctx context.Context, id int64) (*models.Asset, error) {
	var asset models.Asset
	result := conn(ctx, r.db).First(&asset, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get asset by id: %w", result.Error)
	}
	return &asset, nil
}

func (r *AssetRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Asset, error) {
	if !validation.ValidUUID(uuid) {
		return nil, nil // a malformed UUID cannot match; avoid a database cast error
	}
	var asset models.Asset
	result := conn(ctx, r.db).Where("uuid = ? AND user_id = ?", uuid, userID).First(&asset)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get asset: %w", result.Error)
	}
	return &asset, nil
}

func (r *AssetRepo) ListByUserID(ctx context.Context, userID int64, page, limit int) ([]models.Asset, int64, error) {
	var total int64
	if err := conn(ctx, r.db).Model(&models.Asset{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count assets: %w", err)
	}
	var assets []models.Asset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	result := conn(ctx, r.db).Where("user_id = ?", userID).Order("created_at desc").Offset(offset).Limit(limit).Find(&assets)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("list assets: %w", result.Error)
	}
	return assets, total, nil
}

func (r *AssetRepo) Update(ctx context.Context, asset *models.Asset) error {
	// Use Save to update all fields including zero values if they were scanned into the struct
	// But since we want to be careful about what we update, and asset comes from service with values set
	// However, GORM's Save will update all fields.
	// Alternatively we can use Updates.
	// Given the service passes a struct with modified values, Save is appropriate if ID is set.
	result := conn(ctx, r.db).Save(asset)
	if result.Error != nil {
		return fmt.Errorf("update asset: %w", result.Error)
	}
	return nil
}

func (r *AssetRepo) AdjustQuantity(ctx context.Context, assetID int64, delta decimal.Decimal) error {
	db := conn(ctx, r.db)
	// A single conditional UPDATE keeps concurrent adjustments from overwriting each other.
	result := db.Model(&models.Asset{}).
		Where("id = ? AND quantity + ? >= 0", assetID, delta).
		Update("quantity", gorm.Expr("quantity + ?", delta))
	if result.Error != nil {
		return fmt.Errorf("adjust asset quantity: %w", result.Error)
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var count int64
	if err := db.Model(&models.Asset{}).Where("id = ?", assetID).Count(&count).Error; err != nil {
		return fmt.Errorf("check asset exists: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("adjust asset quantity: %w", port.ErrNotFound)
	}
	return port.ErrInsufficientBalance
}

func (r *AssetRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	result := conn(ctx, r.db).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Asset{})
	if result.Error != nil {
		return fmt.Errorf("delete asset: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("asset not found or not owned by user")
	}
	return nil
}
