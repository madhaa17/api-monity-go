package service

import (
	"context"
	"errors"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

type AssetService struct {
	repo port.AssetRepository
}

func NewAssetService(repo port.AssetRepository) port.AssetService {
	return &AssetService{repo: repo}
}

func (s *AssetService) CreateAsset(ctx context.Context, userID int64, req port.CreateAssetRequest) (*models.Asset, error) {
	asset := &models.Asset{
		UserID:   userID,
		Name:     req.Name,
		Type:     req.Type,
		Quantity: decimal.NewFromFloat(req.Quantity),
		Symbol:   req.Symbol,
	}

	if err := s.repo.Create(ctx, asset); err != nil {
		return nil, fmt.Errorf("create asset: %w", err)
	}
	return asset, nil
}

func (s *AssetService) GetAsset(ctx context.Context, userID int64, uuid string) (*models.Asset, error) {
	asset, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, errors.New("asset not found")
	}
	return asset, nil
}

func (s *AssetService) ListAssets(ctx context.Context, userID int64) ([]models.Asset, error) {
	assets, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	if assets == nil {
		return []models.Asset{}, nil
	}
	return assets, nil
}

func (s *AssetService) UpdateAsset(ctx context.Context, userID int64, uuid string, req port.UpdateAssetRequest) (*models.Asset, error) {
	asset, err := s.GetAsset(ctx, userID, uuid)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		asset.Name = *req.Name
	}
	if req.Type != nil {
		asset.Type = *req.Type
	}
	if req.Quantity != nil {
		asset.Quantity = decimal.NewFromFloat(*req.Quantity)
	}
	if req.Symbol != nil {
		asset.Symbol = req.Symbol
	}

	if err := s.repo.Update(ctx, asset); err != nil {
		return nil, fmt.Errorf("update asset: %w", err)
	}
	return asset, nil
}

func (s *AssetService) DeleteAsset(ctx context.Context, userID int64, uuid string) error {
	if err := s.repo.Delete(ctx, uuid, userID); err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}
	return nil
}
