package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

	"github.com/shopspring/decimal"
)

type AssetService struct {
	repo port.AssetRepository
}

func NewAssetService(repo port.AssetRepository) port.AssetService {
	return &AssetService{repo: repo}
}

func (s *AssetService) CreateAsset(ctx context.Context, userID int64, req port.CreateAssetRequest) (*models.Asset, error) {
	if err := validation.CheckMaxLen(req.Name, validation.MaxAssetNameLen); err != nil {
		return nil, fmt.Errorf("name %w", err)
	}
	if req.Symbol != nil {
		if err := validation.CheckMaxLen(*req.Symbol, validation.MaxSymbolLen); err != nil {
			return nil, fmt.Errorf("symbol %w", err)
		}
	}
	if req.Description != nil {
		if err := validation.CheckMaxLen(*req.Description, validation.MaxDescriptionLen); err != nil {
			return nil, fmt.Errorf("description %w", err)
		}
	}
	if req.Notes != nil {
		if err := validation.CheckMaxLen(*req.Notes, validation.MaxNoteLen); err != nil {
			return nil, fmt.Errorf("notes %w", err)
		}
	}
	if req.YieldPeriod != nil {
		if err := validation.CheckMaxLen(*req.YieldPeriod, validation.MaxYieldPeriodLen); err != nil {
			return nil, fmt.Errorf("yieldPeriod %w", err)
		}
	}
	// Parse purchase date
	purchaseDate := time.Now()
	if req.PurchaseDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.PurchaseDate)
		if err != nil {
			return nil, fmt.Errorf("invalid purchase date format: %w", err)
		}
		purchaseDate = parsed
	}

	// Default values
	purchaseCurrency := "IDR"
	if req.PurchaseCurrency != "" {
		purchaseCurrency = req.PurchaseCurrency
	}

	status := models.AssetStatusActive
	if req.Status != nil {
		status = *req.Status
	}

	asset := &models.Asset{
		UserID:   userID,
		Name:     req.Name,
		Type:     req.Type,
		Quantity: decimal.NewFromFloat(req.Quantity),
		Symbol:   req.Symbol,

		// Purchase Information
		PurchasePrice:    decimal.NewFromFloat(req.PurchasePrice),
		PurchaseDate:     purchaseDate,
		PurchaseCurrency: purchaseCurrency,
		TotalCost:        decimal.NewFromFloat(req.TotalCost),

		// Documentation
		Description: req.Description,
		Notes:       req.Notes,

		// Status
		Status: status,
	}

	// Optional fields
	if req.TransactionFee != nil {
		fee := decimal.NewFromFloat(*req.TransactionFee)
		asset.TransactionFee = &fee
	}
	if req.MaintenanceCost != nil {
		cost := decimal.NewFromFloat(*req.MaintenanceCost)
		asset.MaintenanceCost = &cost
	}
	if req.TargetPrice != nil {
		price := decimal.NewFromFloat(*req.TargetPrice)
		asset.TargetPrice = &price
	}
	if req.TargetDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.TargetDate)
		if err == nil {
			asset.TargetDate = &parsed
		}
	}
	if req.EstimatedYield != nil {
		yield := decimal.NewFromFloat(*req.EstimatedYield)
		asset.EstimatedYield = &yield
	}
	if req.YieldPeriod != nil {
		asset.YieldPeriod = req.YieldPeriod
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

func (s *AssetService) ListAssets(ctx context.Context, userID int64, page, limit int) ([]models.Asset, port.ListMeta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	assets, total, err := s.repo.ListByUserID(ctx, userID, page, limit)
	if err != nil {
		return nil, port.ListMeta{}, fmt.Errorf("list assets: %w", err)
	}
	if assets == nil {
		assets = []models.Asset{}
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages < 0 {
		totalPages = 0
	}
	meta := port.ListMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
	return assets, meta, nil
}

func (s *AssetService) UpdateAsset(ctx context.Context, userID int64, uuid string, req port.UpdateAssetRequest) (*models.Asset, error) {
	asset, err := s.GetAsset(ctx, userID, uuid)
	if err != nil {
		return nil, err
	}

	// Basic fields
	if req.Name != nil {
		if err := validation.CheckMaxLen(*req.Name, validation.MaxAssetNameLen); err != nil {
			return nil, fmt.Errorf("name %w", err)
		}
		asset.Name = *req.Name
	}
	if req.Type != nil {
		asset.Type = *req.Type
	}
	if req.Quantity != nil {
		asset.Quantity = decimal.NewFromFloat(*req.Quantity)
	}
	if req.Symbol != nil {
		if err := validation.CheckMaxLen(*req.Symbol, validation.MaxSymbolLen); err != nil {
			return nil, fmt.Errorf("symbol %w", err)
		}
		asset.Symbol = req.Symbol
	}

	// Purchase Information
	if req.PurchasePrice != nil {
		asset.PurchasePrice = decimal.NewFromFloat(*req.PurchasePrice)
	}
	if req.PurchaseDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.PurchaseDate)
		if err != nil {
			return nil, fmt.Errorf("invalid purchase date format: %w", err)
		}
		asset.PurchaseDate = parsed
	}
	if req.PurchaseCurrency != nil {
		asset.PurchaseCurrency = *req.PurchaseCurrency
	}
	if req.TotalCost != nil {
		asset.TotalCost = decimal.NewFromFloat(*req.TotalCost)
	}

	// Additional Costs
	if req.TransactionFee != nil {
		fee := decimal.NewFromFloat(*req.TransactionFee)
		asset.TransactionFee = &fee
	}
	if req.MaintenanceCost != nil {
		cost := decimal.NewFromFloat(*req.MaintenanceCost)
		asset.MaintenanceCost = &cost
	}

	// Target & Planning
	if req.TargetPrice != nil {
		price := decimal.NewFromFloat(*req.TargetPrice)
		asset.TargetPrice = &price
	}
	if req.TargetDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.TargetDate)
		if err == nil {
			asset.TargetDate = &parsed
		}
	}

	// Real Asset Specific
	if req.EstimatedYield != nil {
		yield := decimal.NewFromFloat(*req.EstimatedYield)
		asset.EstimatedYield = &yield
	}
	if req.YieldPeriod != nil {
		if err := validation.CheckMaxLen(*req.YieldPeriod, validation.MaxYieldPeriodLen); err != nil {
			return nil, fmt.Errorf("yieldPeriod %w", err)
		}
		asset.YieldPeriod = req.YieldPeriod
	}

	// Documentation
	if req.Description != nil {
		if err := validation.CheckMaxLen(*req.Description, validation.MaxDescriptionLen); err != nil {
			return nil, fmt.Errorf("description %w", err)
		}
		asset.Description = req.Description
	}
	if req.Notes != nil {
		if err := validation.CheckMaxLen(*req.Notes, validation.MaxNoteLen); err != nil {
			return nil, fmt.Errorf("notes %w", err)
		}
		asset.Notes = req.Notes
	}

	// Status
	if req.Status != nil {
		asset.Status = *req.Status
	}
	if req.SoldAt != nil {
		parsed, err := time.Parse(time.RFC3339, *req.SoldAt)
		if err == nil {
			asset.SoldAt = &parsed
		}
	}
	if req.SoldPrice != nil {
		price := decimal.NewFromFloat(*req.SoldPrice)
		asset.SoldPrice = &price
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
