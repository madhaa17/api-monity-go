package repository

import (
	"context"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type InsightRepo struct {
	db *gorm.DB
}

func NewInsightRepository(db *gorm.DB) port.InsightRepository {
	return &InsightRepo{db: db}
}

func (r *InsightRepo) GetTotalIncomeByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time) (decimal.Decimal, error) {
	var total decimal.NullDecimal
	
	result := r.db.WithContext(ctx).
		Model(&models.Income{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND date >= ? AND date < ?", userID, startDate, endDate).
		Scan(&total)
	
	if result.Error != nil {
		return decimal.Zero, fmt.Errorf("get total income: %w", result.Error)
	}
	
	if total.Valid {
		return total.Decimal, nil
	}
	return decimal.Zero, nil
}

func (r *InsightRepo) GetTotalExpenseByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time) (decimal.Decimal, error) {
	var total decimal.NullDecimal
	
	result := r.db.WithContext(ctx).
		Model(&models.Expense{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND date >= ? AND date < ?", userID, startDate, endDate).
		Scan(&total)
	
	if result.Error != nil {
		return decimal.Zero, fmt.Errorf("get total expense: %w", result.Error)
	}
	
	if total.Valid {
		return total.Decimal, nil
	}
	return decimal.Zero, nil
}

func (r *InsightRepo) GetExpensesByCategory(ctx context.Context, userID int64, startDate, endDate time.Time) ([]port.CategoryTotal, error) {
	var results []struct {
		Category string          `gorm:"column:category"`
		Total    decimal.Decimal `gorm:"column:total"`
	}
	
	err := r.db.WithContext(ctx).
		Model(&models.Expense{}).
		Select("category, SUM(amount) as total").
		Where("user_id = ? AND date >= ? AND date < ?", userID, startDate, endDate).
		Group("category").
		Order("total desc").
		Scan(&results).Error
	
	if err != nil {
		return nil, fmt.Errorf("get expenses by category: %w", err)
	}
	
	// Calculate total for percentages
	var grandTotal decimal.Decimal
	for _, r := range results {
		grandTotal = grandTotal.Add(r.Total)
	}
	
	categories := make([]port.CategoryTotal, len(results))
	for i, r := range results {
		var percentage float64
		if !grandTotal.IsZero() {
			pct := r.Total.Div(grandTotal).Mul(decimal.NewFromInt(100))
			percentage, _ = pct.Float64()
		}
		categories[i] = port.CategoryTotal{
			Category:   r.Category,
			Total:      r.Total,
			Percentage: percentage,
		}
	}
	
	return categories, nil
}

func (r *InsightRepo) GetTotalAssetValue(ctx context.Context, userID int64) (decimal.Decimal, error) {
	// For now, just count assets. Later this can be enhanced to calculate
	// actual value based on quantity * latest price
	var count int64
	result := r.db.WithContext(ctx).
		Model(&models.Asset{}).
		Where("user_id = ?", userID).
		Count(&count)
	
	if result.Error != nil {
		return decimal.Zero, fmt.Errorf("get asset count: %w", result.Error)
	}
	
	return decimal.NewFromInt(count), nil
}

func (r *InsightRepo) GetTotalSavingGoalProgress(ctx context.Context, userID int64) (*port.SavingGoalSummary, error) {
	var result struct {
		TotalGoals   int64           `gorm:"column:total_goals"`
		TotalTarget  decimal.Decimal `gorm:"column:total_target"`
		TotalCurrent decimal.Decimal `gorm:"column:total_current"`
	}
	
	err := r.db.WithContext(ctx).
		Model(&models.SavingGoal{}).
		Select("COUNT(*) as total_goals, COALESCE(SUM(target_amount), 0) as total_target, COALESCE(SUM(current_amount), 0) as total_current").
		Where("user_id = ?", userID).
		Scan(&result).Error
	
	if err != nil {
		return nil, fmt.Errorf("get saving goal summary: %w", err)
	}
	
	var progress float64
	if !result.TotalTarget.IsZero() {
		pct := result.TotalCurrent.Div(result.TotalTarget).Mul(decimal.NewFromInt(100))
		progress, _ = pct.Float64()
	}
	
	return &port.SavingGoalSummary{
		TotalGoals:      int(result.TotalGoals),
		TotalTarget:     result.TotalTarget,
		TotalCurrent:    result.TotalCurrent,
		OverallProgress: progress,
	}, nil
}
