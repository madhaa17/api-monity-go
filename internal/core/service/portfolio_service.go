package service

import (
	"context"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

type PortfolioService struct {
	assetRepo    port.AssetRepository
	priceService port.PriceService
	historyRepo  port.AssetPriceHistoryRepository
}

func NewPortfolioService(assetRepo port.AssetRepository, priceService port.PriceService, historyRepo port.AssetPriceHistoryRepository) port.PortfolioService {
	return &PortfolioService{
		assetRepo:    assetRepo,
		priceService: priceService,
		historyRepo:  historyRepo,
	}
}

func (s *PortfolioService) GetPortfolio(ctx context.Context, userID int64, currency string) (*port.PortfolioResponse, error) {
	if currency == "" {
		currency = port.CurrencyUSD
	}

	assets, err := s.assetRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}

	var assetValues []port.AssetValueResponse
	totalValue := decimal.Zero

	for _, asset := range assets {
		assetValue, err := s.calculateAssetValue(ctx, &asset, currency)
		if err != nil {
			assetValue = &port.AssetValueResponse{
				UUID:         asset.UUID,
				Name:         asset.Name,
				Type:         string(asset.Type),
				Symbol:       s.getSymbolString(asset.Symbol),
				Quantity:     asset.Quantity,
				CurrentPrice: decimal.Zero,
				Value:        decimal.Zero,
				Currency:     currency,
				PriceSource:  "unavailable",
			}
		}
		assetValues = append(assetValues, *assetValue)
		// Only sum into total when asset value is in the requested currency (avoid mixing IDR+USD)
		if assetValue.Currency == currency {
			totalValue = totalValue.Add(assetValue.Value)
		}
	}

	return &port.PortfolioResponse{
		TotalValue: totalValue,
		Currency:   currency,
		Assets:     assetValues,
		FetchedAt:  time.Now(),
	}, nil
}

func (s *PortfolioService) GetAssetValue(ctx context.Context, userID int64, assetUUID string, currency string) (*port.AssetValueResponse, error) {
	if currency == "" {
		currency = port.CurrencyUSD
	}

	asset, err := s.assetRepo.GetByUUID(ctx, assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, fmt.Errorf("asset not found")
	}

	return s.calculateAssetValue(ctx, asset, currency)
}

func (s *PortfolioService) calculateAssetValue(ctx context.Context, asset *models.Asset, currency string) (*port.AssetValueResponse, error) {
	assetCurrency := asset.PurchaseCurrency
	if assetCurrency == "" {
		assetCurrency = "USD"
	}

	// Non-digital assets: CASH, LIVESTOCK, REAL_ESTATE
	// Priority: 1) latest price history (manual update)  2) purchase price  3) zero
	switch asset.Type {
	case models.AssetTypeCash:
		// CASH: check price history first (user may update total cash amount via RecordPrice)
		unitPrice := decimal.NewFromInt(1)
		source := "cash_unit"
		if latest, _ := s.historyRepo.GetLatestByAssetID(ctx, asset.ID); latest != nil {
			unitPrice = latest.Price
			source = "manual_update"
		}
		value := asset.Quantity.Mul(unitPrice)
		return &port.AssetValueResponse{
			UUID:         asset.UUID,
			Name:         asset.Name,
			Type:         string(asset.Type),
			Symbol:       s.getSymbolString(asset.Symbol),
			Quantity:     asset.Quantity,
			CurrentPrice: unitPrice,
			Value:        value,
			Currency:     assetCurrency,
			PriceSource:  source,
		}, nil

	case models.AssetTypeLivestock, models.AssetTypeRealEstate:
		// Priority: latest price history → purchase price → zero
		unitPrice := decimal.Zero
		source := "no_price"
		if latest, _ := s.historyRepo.GetLatestByAssetID(ctx, asset.ID); latest != nil {
			unitPrice = latest.Price
			source = "manual_update"
		} else if !asset.PurchasePrice.IsZero() {
			unitPrice = asset.PurchasePrice
			source = "purchase_price"
		}
		value := asset.Quantity.Mul(unitPrice)
		return &port.AssetValueResponse{
			UUID:         asset.UUID,
			Name:         asset.Name,
			Type:         string(asset.Type),
			Symbol:       s.getSymbolString(asset.Symbol),
			Quantity:     asset.Quantity,
			CurrentPrice: unitPrice,
			Value:        value,
			Currency:     assetCurrency,
			PriceSource:  source,
		}, nil
	}

	// Digital assets with no symbol: nothing to look up
	if asset.Symbol == nil || *asset.Symbol == "" {
		return &port.AssetValueResponse{
			UUID:         asset.UUID,
			Name:         asset.Name,
			Type:         string(asset.Type),
			Symbol:       s.getSymbolString(asset.Symbol),
			Quantity:     asset.Quantity,
			CurrentPrice: decimal.Zero,
			Value:        decimal.Zero,
			Currency:     assetCurrency,
			PriceSource:  "no_symbol",
		}, nil
	}

	var priceData *port.PriceData
	var err error

	switch asset.Type {
	case models.AssetTypeCrypto:
		priceData, err = s.priceService.GetCryptoPriceWithCurrency(ctx, *asset.Symbol, currency)
	case models.AssetTypeStock:
		priceData, err = s.priceService.GetStockPriceWithCurrency(ctx, *asset.Symbol, currency)
	default:
		return &port.AssetValueResponse{
			UUID:         asset.UUID,
			Name:         asset.Name,
			Type:         string(asset.Type),
			Symbol:       *asset.Symbol,
			Quantity:     asset.Quantity,
			CurrentPrice: decimal.Zero,
			Value:        decimal.Zero,
			Currency:     currency,
			PriceSource:  "unsupported_type",
		}, nil
	}

	if err != nil {
		// Fallback: use purchase price when external price is unavailable (e.g. API down, no key)
		if !asset.PurchasePrice.IsZero() {
			value := asset.Quantity.Mul(asset.PurchasePrice)
			return &port.AssetValueResponse{
				UUID:         asset.UUID,
				Name:         asset.Name,
				Type:         string(asset.Type),
				Symbol:       *asset.Symbol,
				Quantity:     asset.Quantity,
				CurrentPrice: asset.PurchasePrice,
				Value:        value,
				Currency:     assetCurrency,
				PriceSource:  "fallback",
			}, nil
		}
		return nil, fmt.Errorf("get price for %s: %w", *asset.Symbol, err)
	}

	currentPrice := decimal.NewFromFloat(priceData.Price)
	value := asset.Quantity.Mul(currentPrice)

	return &port.AssetValueResponse{
		UUID:         asset.UUID,
		Name:         asset.Name,
		Type:         string(asset.Type),
		Symbol:       *asset.Symbol,
		Quantity:     asset.Quantity,
		CurrentPrice: currentPrice,
		Value:        value,
		Currency:     currency,
		PriceSource:  priceData.Source,
	}, nil
}

func (s *PortfolioService) getSymbolString(symbol *string) string {
	if symbol == nil {
		return ""
	}
	return *symbol
}
