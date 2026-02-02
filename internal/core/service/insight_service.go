package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monity/internal/core/port"

	"github.com/shopspring/decimal"
)

type InsightService struct {
	repo port.InsightRepository
}

func NewInsightService(repo port.InsightRepository) port.InsightService {
	return &InsightService{repo: repo}
}

func (s *InsightService) GetCashflowSummary(ctx context.Context, userID int64, month string) (*port.CashflowSummary, error) {
	// Parse month (format: YYYY-MM)
	startDate, endDate, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	// Get totals
	totalIncome, err := s.repo.GetTotalIncomeByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get total income: %w", err)
	}

	totalExpense, err := s.repo.GetTotalExpenseByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get total expense: %w", err)
	}

	// Calculate net saving
	netSaving := totalIncome.Sub(totalExpense)

	// Calculate saving rate
	var savingRate float64
	if !totalIncome.IsZero() {
		rate := netSaving.Div(totalIncome).Mul(decimal.NewFromInt(100))
		savingRate, _ = rate.Float64()
	}

	// Get expenses by category
	expenseByCategory, err := s.repo.GetExpensesByCategory(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get expenses by category: %w", err)
	}

	if expenseByCategory == nil {
		expenseByCategory = []port.CategoryTotal{}
	}

	return &port.CashflowSummary{
		Period:            month,
		TotalIncome:       totalIncome,
		TotalExpense:      totalExpense,
		NetSaving:         netSaving,
		SavingRate:        savingRate,
		ExpenseByCategory: expenseByCategory,
	}, nil
}

func (s *InsightService) GetFinancialOverview(ctx context.Context, userID int64) (*port.FinancialOverview, error) {
	// Get current month range
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	// Get monthly income and expense
	monthlyIncome, err := s.repo.GetTotalIncomeByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get monthly income: %w", err)
	}

	monthlyExpense, err := s.repo.GetTotalExpenseByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get monthly expense: %w", err)
	}

	monthlyNetSaving := monthlyIncome.Sub(monthlyExpense)

	// Get total assets (count for now)
	totalAssets, err := s.repo.GetTotalAssetValue(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get total assets: %w", err)
	}

	// Get saving goal summary
	savingGoalSummary, err := s.repo.GetTotalSavingGoalProgress(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get saving goal summary: %w", err)
	}

	return &port.FinancialOverview{
		TotalAssets:        totalAssets,
		TotalSavingGoals:   savingGoalSummary.TotalGoals,
		TotalTargetAmount:  savingGoalSummary.TotalTarget,
		TotalCurrentAmount: savingGoalSummary.TotalCurrent,
		SavingGoalProgress: savingGoalSummary.OverallProgress,
		MonthlyIncome:      monthlyIncome,
		MonthlyExpense:     monthlyExpense,
		MonthlyNetSaving:   monthlyNetSaving,
	}, nil
}

// parseMonthRange parses a month string (YYYY-MM) and returns start and end dates
func parseMonthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		// Default to current month
		now := time.Now()
		startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)
		return startDate, endDate, nil
	}

	// Parse YYYY-MM format
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid month format, use YYYY-MM")
	}

	startDate := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	return startDate, endDate, nil
}
