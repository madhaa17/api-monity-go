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
}

func NewPortfolioService(assetRepo port.AssetRepository, priceService port.PriceService) port.PortfolioService {
	return &PortfolioService{
		assetRepo:    assetRepo,
		priceService: priceService,
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
		totalValue = totalValue.Add(assetValue.Value)
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
	if asset.Symbol == nil || *asset.Symbol == "" {
		return &port.AssetValueResponse{
			UUID:         asset.UUID,
			Name:         asset.Name,
			Type:         string(asset.Type),
			Quantity:     asset.Quantity,
			CurrentPrice: decimal.Zero,
			Value:        decimal.Zero,
			Currency:     currency,
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
