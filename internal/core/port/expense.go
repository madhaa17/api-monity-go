package port

import (
	"context"
	"monity/internal/models"
	"time"
)

type ExpenseRepository interface {
	Create(ctx context.Context, expense *models.Expense) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Expense, error)
	ListByUserID(ctx context.Context, userID int64) ([]models.Expense, error)
	Update(ctx context.Context, expense *models.Expense) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type ExpenseService interface {
	CreateExpense(ctx context.Context, userID int64, req CreateExpenseRequest) (*models.Expense, error)
	GetExpense(ctx context.Context, userID int64, uuid string) (*models.Expense, error)
	ListExpenses(ctx context.Context, userID int64) ([]models.Expense, error)
	UpdateExpense(ctx context.Context, userID int64, uuid string, req UpdateExpenseRequest) (*models.Expense, error)
	DeleteExpense(ctx context.Context, userID int64, uuid string) error
}

type CreateExpenseRequest struct {
	AssetUUID string                 `json:"assetUuid"`
	Amount    float64                `json:"amount"`
	Category  models.ExpenseCategory `json:"category"`
	Note      *string                `json:"note,omitempty"`
	Date      time.Time              `json:"date"`
}

type UpdateExpenseRequest struct {
	AssetUUID *string                 `json:"assetUuid,omitempty"`
	Amount    *float64                `json:"amount,omitempty"`
	Category  *models.ExpenseCategory `json:"category,omitempty"`
	Note      *string                 `json:"note,omitempty"`
	Date      *time.Time              `json:"date,omitempty"`
}
