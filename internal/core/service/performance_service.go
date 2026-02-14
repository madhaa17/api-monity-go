package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

type PerformanceService struct {
	assetRepo    port.AssetRepository
	priceService port.PriceService
}

func NewPerformanceService(assetRepo port.AssetRepository, priceService port.PriceService) port.AssetPerformanceService {
	return &PerformanceService{
		assetRepo:    assetRepo,
		priceService: priceService,
	}
}

func (s *PerformanceService) GetAssetPerformance(ctx context.Context, userID int64, assetUUID string, currency string) (*port.AssetPerformanceResponse, error) {
	// Get asset
	asset, err := s.assetRepo.GetByUUID(ctx, assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, fmt.Errorf("asset not found")
	}

	// Default currency
	if currency == "" {
		currency = asset.PurchaseCurrency
		if currency == "" {
			currency = port.DefaultCurrency
		}
	}

	// Calculate current value
	currentPrice := decimal.Zero
	currentValue := decimal.Zero
	priceChange24h := 0.0

	if asset.Symbol != nil && *asset.Symbol != "" {
		var priceData *port.PriceData
		switch asset.Type {
		case models.AssetTypeCrypto:
			priceData, err = s.priceService.GetCryptoPriceWithCurrency(ctx, *asset.Symbol, currency)
		case models.AssetTypeStock:
			priceData, err = s.priceService.GetStockPriceWithCurrency(ctx, *asset.Symbol, currency)
		}

		if err == nil && priceData != nil {
			currentPrice = decimal.NewFromFloat(priceData.Price)
			effectiveQty := s.effectiveQuantity(asset)
			currentValue = effectiveQty.Mul(currentPrice)
		}
	}

	// If no current price available, use purchase price
	if currentPrice.IsZero() {
		currentPrice = asset.PurchasePrice
		effectiveQty := s.effectiveQuantity(asset)
		currentValue = effectiveQty.Mul(currentPrice)
	}

	// Calculate performance metrics
	profitLoss := currentValue.Sub(asset.TotalCost)
	profitLossPercent := decimal.Zero
	if !asset.TotalCost.IsZero() {
		profitLossPercent = profitLoss.Div(asset.TotalCost).Mul(decimal.NewFromInt(100))
	}

	// Holding period in days
	holdingPeriod := int(time.Since(asset.PurchaseDate).Hours() / 24)
	if holdingPeriod < 1 {
		holdingPeriod = 1
	}

	// Annualized return
	annualizedReturn := decimal.Zero
	if holdingPeriod > 0 {
		years := decimal.NewFromFloat(float64(holdingPeriod) / 365.0)
		if !years.IsZero() {
			annualizedReturn = profitLossPercent.Div(years)
		}
	}

	// Performance status
	status := "break-even"
	if profitLoss.GreaterThan(decimal.Zero) {
		status = "profit"
	} else if profitLoss.LessThan(decimal.Zero) {
		status = "loss"
	}

	// Analysis
	message := s.generatePerformanceMessage(asset.Name, profitLossPercent, status)
	recommendation := s.generateRecommendation(asset, currentPrice, profitLossPercent)
	targetReached := false
	if asset.TargetPrice != nil && currentPrice.GreaterThanOrEqual(*asset.TargetPrice) {
		targetReached = true
	}

	// Transaction fee
	transactionFee := decimal.Zero
	if asset.TransactionFee != nil {
		transactionFee = *asset.TransactionFee
	}

	return &port.AssetPerformanceResponse{
		AssetUUID: asset.UUID,
		AssetName: asset.Name,
		Type:      string(asset.Type),
		Symbol:    s.getSymbolString(asset.Symbol),
		Investment: port.InvestmentInfo{
			Quantity:         asset.Quantity,
			PurchasePrice:    asset.PurchasePrice,
			PurchaseDate:     asset.PurchaseDate,
			TotalCost:        asset.TotalCost,
			Currency:         currency,
			TransactionFee:   transactionFee,
			DaysSinceHolding: holdingPeriod,
		},
		CurrentData: port.CurrentValueInfo{
			CurrentPrice:   currentPrice,
			CurrentValue:   currentValue,
			PriceChange24h: priceChange24h,
			LastUpdated:    time.Now(),
		},
		Performance: port.PerformanceMetrics{
			ProfitLoss:        profitLoss,
			ProfitLossPercent: profitLossPercent,
			ROI:               profitLossPercent,
			Status:            status,
			HoldingPeriod:     holdingPeriod,
			AnnualizedReturn:  annualizedReturn,
		},
		Analysis: port.PerformanceAnalysis{
			Message:        message,
			Recommendation: recommendation,
			TargetReached:  targetReached,
		},
	}, nil
}

func (s *PerformanceService) GetPortfolioPerformance(ctx context.Context, userID int64, currency string) (*port.PortfolioPerformanceResponse, error) {
	// Get all user assets
	assets, err := s.assetRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}

	if currency == "" {
		currency = port.DefaultCurrency
	}

	// Initialize aggregates
	totalInvested := decimal.Zero
	totalCurrentValue := decimal.Zero
	allocationMap := make(map[string]port.AssetTypeAllocation)
	var performers []port.PerformerSummary
	statusSummary := port.StatusSummary{}

	// Process each asset
	for _, asset := range assets {
		// Count by status
		switch asset.Status {
		case models.AssetStatusActive:
			statusSummary.Active++
		case models.AssetStatusSold:
			statusSummary.Sold++
		case models.AssetStatusPlanned:
			statusSummary.Planned++
		}

		// Skip planned assets from calculations
		if asset.Status == models.AssetStatusPlanned {
			continue
		}

		// Calculate current value
		currentPrice := asset.PurchasePrice
		if asset.Symbol != nil && *asset.Symbol != "" {
			var priceData *port.PriceData
			switch asset.Type {
			case models.AssetTypeCrypto:
				priceData, _ = s.priceService.GetCryptoPriceWithCurrency(ctx, *asset.Symbol, currency)
			case models.AssetTypeStock:
				priceData, _ = s.priceService.GetStockPriceWithCurrency(ctx, *asset.Symbol, currency)
			}
			if priceData != nil {
				currentPrice = decimal.NewFromFloat(priceData.Price)
			}
		}

		effectiveQty := s.effectiveQuantity(&asset)
		currentValue := effectiveQty.Mul(currentPrice)
		profitLoss := currentValue.Sub(asset.TotalCost)
		profitLossPercent := decimal.Zero
		if !asset.TotalCost.IsZero() {
			profitLossPercent = profitLoss.Div(asset.TotalCost).Mul(decimal.NewFromInt(100))
		}

		// Aggregate totals
		totalInvested = totalInvested.Add(asset.TotalCost)
		totalCurrentValue = totalCurrentValue.Add(currentValue)

		// Asset type allocation
		assetType := string(asset.Type)
		allocation := allocationMap[assetType]
		allocation.Count++
		allocation.TotalInvested = allocation.TotalInvested.Add(asset.TotalCost)
		allocation.CurrentValue = allocation.CurrentValue.Add(currentValue)
		allocation.ProfitLoss = allocation.ProfitLoss.Add(profitLoss)
		allocationMap[assetType] = allocation

		// Add to performers list
		performers = append(performers, port.PerformerSummary{
			UUID:              asset.UUID,
			Name:              asset.Name,
			Type:              assetType,
			ProfitLossPercent: profitLossPercent,
			ProfitLoss:        profitLoss,
		})
	}

	// Calculate portfolio totals
	totalProfitLoss := totalCurrentValue.Sub(totalInvested)
	totalProfitLossPercent := decimal.Zero
	if !totalInvested.IsZero() {
		totalProfitLossPercent = totalProfitLoss.Div(totalInvested).Mul(decimal.NewFromInt(100))
	}

	// Calculate allocation percentages and ROI
	for assetType, allocation := range allocationMap {
		if !totalCurrentValue.IsZero() {
			allocation.Percentage = allocation.CurrentValue.Div(totalCurrentValue).Mul(decimal.NewFromInt(100))
		}
		if !allocation.TotalInvested.IsZero() {
			allocation.ROI = allocation.ProfitLoss.Div(allocation.TotalInvested).Mul(decimal.NewFromInt(100))
		}
		allocationMap[assetType] = allocation
	}

	// Sort performers
	sort.Slice(performers, func(i, j int) bool {
		return performers[i].ProfitLossPercent.GreaterThan(performers[j].ProfitLossPercent)
	})

	// Get top 5 gainers and losers
	gainers := []port.PerformerSummary{}
	losers := []port.PerformerSummary{}

	for _, p := range performers {
		if p.ProfitLossPercent.GreaterThan(decimal.Zero) {
			if len(gainers) < 5 {
				gainers = append(gainers, p)
			}
		}
	}

	// Reverse for losers (worst first)
	for i := len(performers) - 1; i >= 0; i-- {
		if performers[i].ProfitLossPercent.LessThan(decimal.Zero) {
			if len(losers) < 5 {
				losers = append(losers, performers[i])
			}
		}
	}

	return &port.PortfolioPerformanceResponse{
		Overview: port.PortfolioOverview{
			TotalInvested:          totalInvested,
			CurrentValue:           totalCurrentValue,
			TotalProfitLoss:        totalProfitLoss,
			TotalProfitLossPercent: totalProfitLossPercent,
			TotalROI:               totalProfitLossPercent,
			Currency:               currency,
		},
		AssetAllocation: allocationMap,
		TopPerformers: port.PerformersInfo{
			Gainers: gainers,
			Losers:  losers,
		},
		StatusSummary: statusSummary,
		LastUpdated:   time.Now(),
	}, nil
}

func (s *PerformanceService) generatePerformanceMessage(assetName string, profitLossPercent decimal.Decimal, status string) string {
	emoji := ""
	verb := ""

	switch status {
	case "profit":
		emoji = "🎉"
		verb = "up"
	case "loss":
		emoji = "📉"
		verb = "down"
	default:
		emoji = "⚪"
		return fmt.Sprintf("Your %s investment is at break-even %s", assetName, emoji)
	}

	return fmt.Sprintf("Your %s investment is %s %.2f%% %s", assetName, verb, profitLossPercent.Abs().InexactFloat64(), emoji)
}

func (s *PerformanceService) generateRecommendation(asset *models.Asset, currentPrice decimal.Decimal, profitLossPercent decimal.Decimal) string {
	// Target price reached
	if asset.TargetPrice != nil && currentPrice.GreaterThanOrEqual(*asset.TargetPrice) {
		return "Target price reached! Consider taking profit."
	}

	// High profit
	if profitLossPercent.GreaterThan(decimal.NewFromInt(50)) {
		return "Strong performance! Consider taking partial profits or rebalancing."
	}

	// Moderate profit
	if profitLossPercent.GreaterThan(decimal.NewFromInt(20)) {
		return "Good performance! Continue holding or consider your exit strategy."
	}

	// Significant loss
	if profitLossPercent.LessThan(decimal.NewFromInt(-20)) {
		return "Significant loss. Review your investment thesis and consider cutting losses."
	}

	// Moderate loss
	if profitLossPercent.LessThan(decimal.NewFromInt(-10)) {
		return "Currently in loss. Hold if you believe in long-term prospects."
	}

	return "Monitor regularly and stick to your investment plan."
}

// effectiveQuantity returns the actual number of units for value calculation.
// For IDX stocks, quantity is in lots (1 lot = 100 shares), so we multiply by 100.
func (s *PerformanceService) effectiveQuantity(asset *models.Asset) decimal.Decimal {
	if asset.Type == models.AssetTypeStock && asset.Symbol != nil && IsIDXStock(strings.ToUpper(*asset.Symbol)) {
		return asset.Quantity.Mul(decimal.NewFromInt(IDXLotSize))
	}
	return asset.Quantity
}

func (s *PerformanceService) getSymbolString(symbol *string) string {
	if symbol == nil {
		return ""
	}
	return *symbol
}
