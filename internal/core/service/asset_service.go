package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, port.NewValidationError("name is required")
	}
	if err := validateAssetText("name", &req.Name, validation.MaxAssetNameLen); err != nil {
		return nil, err
	}
	if err := validateAssetType(req.Type); err != nil {
		return nil, err
	}
	if req.Status != nil {
		if err := validateAssetStatus(*req.Status); err != nil {
			return nil, err
		}
	}
	if err := validateAssetText("symbol", req.Symbol, validation.MaxSymbolLen); err != nil {
		return nil, err
	}
	if err := validateAssetText("description", req.Description, validation.MaxDescriptionLen); err != nil {
		return nil, err
	}
	if err := validateAssetText("notes", req.Notes, validation.MaxNoteLen); err != nil {
		return nil, err
	}
	if err := validateAssetText("yieldPeriod", req.YieldPeriod, validation.MaxYieldPeriodLen); err != nil {
		return nil, err
	}
	for field, v := range map[string]float64{"quantity": req.Quantity, "purchasePrice": req.PurchasePrice, "totalCost": req.TotalCost} {
		if err := validateAssetNumber(field, v); err != nil {
			return nil, err
		}
	}
	for field, v := range map[string]*float64{
		"transactionFee": req.TransactionFee, "maintenanceCost": req.MaintenanceCost,
		"targetPrice": req.TargetPrice, "estimatedYield": req.EstimatedYield,
	} {
		if err := validateOptionalAssetNumber(field, v); err != nil {
			return nil, err
		}
	}

	purchaseDate := time.Now()
	if strings.TrimSpace(req.PurchaseDate) != "" {
		parsed, err := parseAssetDate("purchaseDate", req.PurchaseDate)
		if err != nil {
			return nil, err
		}
		purchaseDate = parsed
	}
	var targetDate *time.Time
	if req.TargetDate != nil {
		parsed, err := parseOptionalAssetDate("targetDate", *req.TargetDate)
		if err != nil {
			return nil, err
		}
		targetDate = parsed
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
	asset.TargetDate = targetDate
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
		return nil, fmt.Errorf("asset: %w", port.ErrNotFound)
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

	if err := validateAssetUpdate(&req); err != nil {
		return nil, err
	}

	// Basic fields
	if req.Name != nil {
		asset.Name = strings.TrimSpace(*req.Name)
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

	// Purchase Information
	if req.PurchasePrice != nil {
		asset.PurchasePrice = decimal.NewFromFloat(*req.PurchasePrice)
	}
	if req.PurchaseDate != nil {
		asset.PurchaseDate, _ = parseAssetDate("purchaseDate", *req.PurchaseDate) // validated above
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
		asset.TargetDate, _ = parseOptionalAssetDate("targetDate", *req.TargetDate)
	}

	// Real Asset Specific
	if req.EstimatedYield != nil {
		yield := decimal.NewFromFloat(*req.EstimatedYield)
		asset.EstimatedYield = &yield
	}
	if req.YieldPeriod != nil {
		asset.YieldPeriod = req.YieldPeriod
	}

	// Documentation
	if req.Description != nil {
		asset.Description = req.Description
	}
	if req.Notes != nil {
		asset.Notes = req.Notes
	}

	// Status
	if req.Status != nil {
		asset.Status = *req.Status
	}
	if req.SoldAt != nil {
		asset.SoldAt, _ = parseOptionalAssetDate("soldAt", *req.SoldAt)
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
	err := s.repo.Delete(ctx, uuid, userID)
	if errors.Is(err, port.ErrConflict) {
		return port.NewValidationError("this asset is still used by incomes, expenses, debts or receivables; delete or move those first")
	}
	if err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}
	return nil
}

// validateAssetUpdate checks every field of a partial update before any of it is applied.
func validateAssetUpdate(req *port.UpdateAssetRequest) error {
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return port.NewValidationError("name cannot be empty")
		}
		if err := validateAssetText("name", req.Name, validation.MaxAssetNameLen); err != nil {
			return err
		}
	}
	if req.Type != nil {
		if err := validateAssetType(*req.Type); err != nil {
			return err
		}
	}
	if req.Status != nil {
		if err := validateAssetStatus(*req.Status); err != nil {
			return err
		}
	}
	for field, v := range map[string]*string{"symbol": req.Symbol, "description": req.Description, "notes": req.Notes, "yieldPeriod": req.YieldPeriod} {
		max := map[string]int{"symbol": validation.MaxSymbolLen, "description": validation.MaxDescriptionLen, "notes": validation.MaxNoteLen, "yieldPeriod": validation.MaxYieldPeriodLen}[field]
		if err := validateAssetText(field, v, max); err != nil {
			return err
		}
	}
	for field, v := range map[string]*float64{
		"quantity": req.Quantity, "purchasePrice": req.PurchasePrice, "totalCost": req.TotalCost,
		"transactionFee": req.TransactionFee, "maintenanceCost": req.MaintenanceCost,
		"targetPrice": req.TargetPrice, "estimatedYield": req.EstimatedYield, "soldPrice": req.SoldPrice,
	} {
		if err := validateOptionalAssetNumber(field, v); err != nil {
			return err
		}
	}
	if req.PurchaseDate != nil {
		if _, err := parseAssetDate("purchaseDate", *req.PurchaseDate); err != nil {
			return err
		}
	}
	for field, v := range map[string]*string{"targetDate": req.TargetDate, "soldAt": req.SoldAt} {
		if v != nil {
			if _, err := parseOptionalAssetDate(field, *v); err != nil {
				return err
			}
		}
	}
	return nil
}
