package service

import (
	"context"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

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

// maxClockSkew is how far in the future a recorded price's time may be, for clients whose clock
// runs slightly ahead.
const maxClockSkew = 5 * time.Minute

// maxPriceHistoryLimit caps how many price history entries one request returns.
const maxPriceHistoryLimit = 1000

func (s *AssetPriceHistoryService) RecordPrice(ctx context.Context, userID int64, assetUUID string, req port.RecordPriceRequest) (*models.AssetPriceHistory, error) {
	// Validate
	if req.Price <= 0 {
		return nil, port.NewValidationError("price must be positive")
	}
	if req.Source == "" {
		return nil, port.NewValidationError("source is required")
	}
	if err := validation.CheckMaxLen(req.Source, validation.MaxSourceLen); err != nil {
		return nil, port.NewValidationError("source " + err.Error())
	}
	recordedAt := req.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now()
	}
	// The latest entry values cash, livestock and real estate, so a future one would hide every
	// update made before its time comes.
	if recordedAt.After(time.Now().Add(maxClockSkew)) {
		return nil, port.NewValidationError("recordedAt must not be in the future")
	}

	asset, err := s.ownedAsset(ctx, userID, assetUUID)
	if err != nil {
		return nil, err
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
	asset, err := s.ownedAsset(ctx, userID, assetUUID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 30 // Default limit
	}
	limit = min(limit, maxPriceHistoryLimit)

	histories, err := s.historyRepo.ListByAssetID(ctx, asset.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("list price history: %w", err)
	}

	if histories == nil {
		return []models.AssetPriceHistory{}, nil
	}

	return histories, nil
}

// FetchAndRecordPrice records the current market price of a crypto or stock asset, per unit and in
// the asset's purchase currency. Price history entries carry no currency, so a price that could
// not be converted, or a stale one repeated from an earlier fetch, is refused instead of recorded.
func (s *AssetPriceHistoryService) FetchAndRecordPrice(ctx context.Context, userID int64, assetUUID string) (*models.AssetPriceHistory, error) {
	asset, err := s.ownedAsset(ctx, userID, assetUUID)
	if err != nil {
		return nil, err
	}

	if asset.Type != models.AssetTypeCrypto && asset.Type != models.AssetTypeStock {
		return nil, port.NewValidationError("market prices are only available for crypto and stock assets")
	}
	if asset.Symbol == nil || *asset.Symbol == "" {
		return nil, port.NewValidationError("asset has no symbol for price lookup")
	}

	currency := asset.PurchaseCurrency
	if currency == "" {
		currency = port.DefaultCurrency
	}
	priceData, err := s.priceService.GetPriceWithCurrency(ctx, string(asset.Type), *asset.Symbol, currency)
	if err != nil {
		return nil, fmt.Errorf("fetch price: %w", err)
	}
	if priceData.Stale {
		return nil, fmt.Errorf("price source unreachable for %s: %w", *asset.Symbol, port.ErrPriceUnavailable)
	}
	if priceData.Currency != currency {
		return nil, fmt.Errorf("no %s to %s exchange rate for %s: %w", priceData.Currency, currency, *asset.Symbol, port.ErrPriceUnavailable)
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

// ownedAsset returns the user's asset, or an error wrapping port.ErrNotFound.
func (s *AssetPriceHistoryService) ownedAsset(ctx context.Context, userID int64, assetUUID string) (*models.Asset, error) {
	asset, err := s.assetRepo.GetByUUID(ctx, assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, fmt.Errorf("asset: %w", port.ErrNotFound)
	}
	return asset, nil
}
