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
	var categories []port.CategoryTotal
	err := r.db.WithContext(ctx).
		Model(&models.Expense{}).
		Select("categories.uuid AS category_uuid, categories.name AS category, categories.color AS color, SUM(expenses.amount) AS total, COUNT(*) AS count").
		Joins("JOIN categories ON categories.id = expenses.category_id").
		Where("expenses.user_id = ? AND expenses.date >= ? AND expenses.date < ?", userID, startDate, endDate).
		Group("categories.id").
		Order("total desc").
		Scan(&categories).Error
	if err != nil {
		return nil, fmt.Errorf("get expenses by category: %w", err)
	}

	var grandTotal decimal.Decimal
	for _, c := range categories {
		grandTotal = grandTotal.Add(c.Total)
	}
	setPercentages(categories, grandTotal)
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

func (r *InsightRepo) GetTotalDebt(ctx context.Context, userID int64) (decimal.Decimal, error) {
	var total decimal.NullDecimal
	err := r.db.WithContext(ctx).
		Model(&models.Debt{}).
		Select("COALESCE(SUM(amount - paid_amount), 0)").
		Where("user_id = ? AND status != ?", userID, "PAID").
		Scan(&total).Error
	if err != nil {
		return decimal.Zero, fmt.Errorf("get total debt: %w", err)
	}
	if total.Valid {
		return total.Decimal, nil
	}
	return decimal.Zero, nil
}

func (r *InsightRepo) GetTotalReceivable(ctx context.Context, userID int64) (decimal.Decimal, error) {
	var total decimal.NullDecimal
	err := r.db.WithContext(ctx).
		Model(&models.Receivable{}).
		Select("COALESCE(SUM(amount - paid_amount), 0)").
		Where("user_id = ? AND status != ?", userID, "PAID").
		Scan(&total).Error
	if err != nil {
		return decimal.Zero, fmt.Errorf("get total receivable: %w", err)
	}
	if total.Valid {
		return total.Decimal, nil
	}
	return decimal.Zero, nil
}

func (r *InsightRepo) GetDebtOverdueCount(ctx context.Context, userID int64) (int, error) {
	var count int64
	now := models.StartOfDayUTC(time.Now()) // due today is not yet overdue
	err := r.db.WithContext(ctx).
		Model(&models.Debt{}).
		Where("user_id = ? AND due_date < ? AND status != ?", userID, now, "PAID").
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("get debt overdue count: %w", err)
	}
	return int(count), nil
}

func (r *InsightRepo) GetReceivableOverdueCount(ctx context.Context, userID int64) (int, error) {
	var count int64
	now := models.StartOfDayUTC(time.Now()) // due today is not yet overdue
	err := r.db.WithContext(ctx).
		Model(&models.Receivable{}).
		Where("user_id = ? AND due_date < ? AND status != ?", userID, now, "PAID").
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("get receivable overdue count: %w", err)
	}
	return int(count), nil
}
