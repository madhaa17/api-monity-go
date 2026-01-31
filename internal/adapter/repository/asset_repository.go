package repository

import (
	"context"
	"errors"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
)

type AssetRepo struct {
	db *gorm.DB
}

func NewAssetRepository(db *gorm.DB) port.AssetRepository {
	return &AssetRepo{db: db}
}

func (r *AssetRepo) Create(ctx context.Context, asset *models.Asset) error {
	result := r.db.WithContext(ctx).Create(asset)
	if result.Error != nil {
		return fmt.Errorf("create asset: %w", result.Error)
	}
	return nil
}

func (r *AssetRepo) GetByID(ctx context.Context, id int64, userID int64) (*models.Asset, error) {
	var asset models.Asset
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&asset)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get asset: %w", result.Error)
	}
	return &asset, nil
}

func (r *AssetRepo) ListByUserID(ctx context.Context, userID int64) ([]models.Asset, error) {
	var assets []models.Asset
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&assets)
	if result.Error != nil {
		return nil, fmt.Errorf("list assets: %w", result.Error)
	}
	return assets, nil
}

func (r *AssetRepo) Update(ctx context.Context, asset *models.Asset) error {
	// Use Save to update all fields including zero values if they were scanned into the struct
	// But since we want to be careful about what we update, and asset comes from service with values set
	// However, GORM's Save will update all fields.
	// Alternatively we can use Updates.
	// Given the service passes a struct with modified values, Save is appropriate if ID is set.
	result := r.db.WithContext(ctx).Save(asset)
	if result.Error != nil {
		return fmt.Errorf("update asset: %w", result.Error)
	}
	return nil
}

func (r *AssetRepo) Delete(ctx context.Context, id int64, userID int64) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.Asset{})
	if result.Error != nil {
		return fmt.Errorf("delete asset: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("asset not found or not owned by user")
	}
	return nil
}
