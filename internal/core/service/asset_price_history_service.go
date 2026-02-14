package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

type AssetPriceHistoryService struct {
	historyRepo  port.AssetPriceHistoryRepository
	assetRepo    port.AssetRepository
	priceService port.PriceService
}

func NewAssetPriceHistoryService(
	historyRepo port.AssetPriceHistoryRepository,
	assetRepo port.AssetRepository,
	priceService port.PriceService,
) port.AssetPriceHistoryService {
	return &AssetPriceHistoryService{
		historyRepo:  historyRepo,
		assetRepo:    assetRepo,
		priceService: priceService,
	}
}

func (s *AssetPriceHistoryService) RecordPrice(ctx context.Context, userID int64, assetUUID string, req port.RecordPriceRequest) (*models.AssetPriceHistory, error) {
	// Validate
	if req.Price <= 0 {
		return nil, errors.New("price must be positive")
	}
	if req.Source == "" {
		return nil, errors.New("source is required")
	}

	// Get asset to verify ownership and get ID
	asset, err := s.assetRepo.GetByUUID(ctx, assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, errors.New("asset not found")
	}

	recordedAt := req.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now()
	}

	history := &models.AssetPriceHistory{
		AssetID:    asset.ID,
		Price:      decimal.NewFromFloat(req.Price),
		Source:     req.Source,
		RecordedAt: recordedAt,
	}

	if err := s.historyRepo.Create(ctx, history); err != nil {
		return nil, fmt.Errorf("create price history: %w", err)
	}

	return history, nil
}

func (s *AssetPriceHistoryService) GetPriceHistory(ctx context.Context, userID int64, assetUUID string, limit int) ([]models.AssetPriceHistory, error) {
	// Get asset to verify ownership and get ID
	asset, err := s.assetRepo.GetByUUID(ctx, assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, errors.New("asset not found")
	}

	if limit <= 0 {
		limit = 30 // Default limit
	}

	histories, err := s.historyRepo.ListByAssetID(ctx, asset.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("list price history: %w", err)
	}

	if histories == nil {
		return []models.AssetPriceHistory{}, nil
	}

	return histories, nil
}

func (s *AssetPriceHistoryService) FetchAndRecordPrice(ctx context.Context, userID int64, assetUUID string) (*models.AssetPriceHistory, error) {
	// Get asset to verify ownership
	asset, err := s.assetRepo.GetByUUID(ctx, assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, errors.New("asset not found")
	}

	// Asset must have a symbol for price lookup
	if asset.Symbol == nil || *asset.Symbol == "" {
		return nil, errors.New("asset has no symbol for price lookup")
	}

	// Fetch price from external API
	priceData, err := s.priceService.GetPrice(ctx, string(asset.Type), *asset.Symbol)
	if err != nil {
		return nil, fmt.Errorf("fetch price: %w", err)
	}

	// Record the price
	history := &models.AssetPriceHistory{
		AssetID:    asset.ID,
		Price:      decimal.NewFromFloat(priceData.Price),
		Source:     priceData.Source,
		RecordedAt: priceData.FetchedAt,
	}

	if err := s.historyRepo.Create(ctx, history); err != nil {
		return nil, fmt.Errorf("create price history: %w", err)
	}

	return history, nil
}
