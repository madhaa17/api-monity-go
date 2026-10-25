package port

import (
	"context"
	"monity/internal/models"
	"time"

	"github.com/shopspring/decimal"
)

type ExpenseRepository interface {
	Create(ctx context.Context, expense *models.Expense) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Expense, error)
	// GetByUUIDForUpdate is GetByUUID with a row lock; call it inside Transactor.WithinTransaction.
	GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Expense, error)
	ListByUserID(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Expense, int64, error)
	Update(ctx context.Context, expense *models.Expense) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type ExpenseService interface {
	CreateExpense(ctx context.Context, userID int64, req CreateExpenseRequest) (*models.Expense, error)
	GetExpense(ctx context.Context, userID int64, uuid string) (*models.Expense, error)
	ListExpenses(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Expense, ListMeta, error)
	UpdateExpense(ctx context.Context, userID int64, uuid string, req UpdateExpenseRequest) (*models.Expense, error)
	DeleteExpense(ctx context.Context, userID int64, uuid string) error
}

type CreateExpenseRequest struct {
	AssetUUID string          `json:"assetUuid"`
	Amount    decimal.Decimal `json:"amount"`
	// CategoryUUID selects a user EXPENSE category. Category is the legacy alternative: a category
	// name or old enum code (e.g. "FOOD"), matched case-insensitively. One of them is required.
	CategoryUUID string    `json:"categoryUuid"`
	Category     string    `json:"category"`
	Note         *string   `json:"note,omitempty"`
	Date         time.Time `json:"date"`
}

type UpdateExpenseRequest struct {
	AssetUUID    *string          `json:"assetUuid,omitempty"`
	Amount       *decimal.Decimal `json:"amount,omitempty"`
	CategoryUUID *string          `json:"categoryUuid,omitempty"`
	Category     *string          `json:"category,omitempty"` // legacy, see CreateExpenseRequest
	Note         *string          `json:"note,omitempty"`
	Date         *time.Time       `json:"date,omitempty"`
}
