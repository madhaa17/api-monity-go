package port

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type InsightRepository interface {
	GetTotalIncomeByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time) (decimal.Decimal, error)
	GetTotalExpenseByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time) (decimal.Decimal, error)
	GetExpensesByCategory(ctx context.Context, userID int64, startDate, endDate time.Time) ([]CategoryTotal, error)
	GetTotalAssetValue(ctx context.Context, userID int64) (decimal.Decimal, error)
	GetTotalSavingGoalProgress(ctx context.Context, userID int64) (*SavingGoalSummary, error)
}

type InsightService interface {
	GetCashflowSummary(ctx context.Context, userID int64, month string) (*CashflowSummary, error)
	GetFinancialOverview(ctx context.Context, userID int64) (*FinancialOverview, error)
}

type CashflowSummary struct {
	Period            string            `json:"period"`
	TotalIncome       decimal.Decimal   `json:"totalIncome"`
	TotalExpense      decimal.Decimal   `json:"totalExpense"`
	NetSaving         decimal.Decimal   `json:"netSaving"`
	SavingRate        float64           `json:"savingRate"`
	ExpenseByCategory []CategoryTotal   `json:"expenseByCategory"`
}

type CategoryTotal struct {
	Category string          `json:"category"`
	Total    decimal.Decimal `json:"total"`
	Percentage float64       `json:"percentage"`
}

type FinancialOverview struct {
	TotalAssets          decimal.Decimal    `json:"totalAssets"`
	TotalSavingGoals     int                `json:"totalSavingGoals"`
	TotalTargetAmount    decimal.Decimal    `json:"totalTargetAmount"`
	TotalCurrentAmount   decimal.Decimal    `json:"totalCurrentAmount"`
	SavingGoalProgress   float64            `json:"savingGoalProgress"`
	MonthlyIncome        decimal.Decimal    `json:"monthlyIncome"`
	MonthlyExpense       decimal.Decimal    `json:"monthlyExpense"`
	MonthlyNetSaving     decimal.Decimal    `json:"monthlyNetSaving"`
}

type SavingGoalSummary struct {
	TotalGoals        int             `json:"totalGoals"`
	TotalTarget       decimal.Decimal `json:"totalTarget"`
	TotalCurrent      decimal.Decimal `json:"totalCurrent"`
	OverallProgress   float64         `json:"overallProgress"`
}
