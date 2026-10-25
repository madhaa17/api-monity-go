package port

import (
	"context"
	"monity/internal/models"
	"time"

	"github.com/shopspring/decimal"
)

type IncomeRepository interface {
	Create(ctx context.Context, income *models.Income) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Income, error)
	// GetByUUIDForUpdate is GetByUUID with a row lock; call it inside Transactor.WithinTransaction.
	GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Income, error)
	ListByUserID(ctx context.Context, userID int64, filter TransactionFilter, sort TransactionSort, page, limit int) ([]models.Income, int64, error)
	Summarize(ctx context.Context, userID int64, filter TransactionFilter, groupBy SummaryGroupBy) (*TransactionSummary, error)
	Update(ctx context.Context, income *models.Income) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type IncomeService interface {
	CreateIncome(ctx context.Context, userID int64, req CreateIncomeRequest) (*models.Income, error)
	GetIncome(ctx context.Context, userID int64, uuid string) (*models.Income, error)
	// ListIncomes returns a page of matching incomes; sort is "date", "-date" (default), "amount" or "-amount".
	ListIncomes(ctx context.Context, userID int64, filter TransactionFilter, sort string, page, limit int) ([]models.Income, ListMeta, error)
	// SummarizeIncomes aggregates the incomes matching filter; groupBy is "day" (default) or "month".
	SummarizeIncomes(ctx context.Context, userID int64, filter TransactionFilter, groupBy string) (*TransactionSummary, error)
	UpdateIncome(ctx context.Context, userID int64, uuid string, req UpdateIncomeRequest) (*models.Income, error)
	DeleteIncome(ctx context.Context, userID int64, uuid string) error
}

type CreateIncomeRequest struct {
	AssetUUID string          `json:"assetUuid"`
	Amount    decimal.Decimal `json:"amount"`
	Source    string          `json:"source"`
	// CategoryUUID selects a user INCOME category; when empty the user's "Other" category is used.
	CategoryUUID string    `json:"categoryUuid"`
	Note         *string   `json:"note,omitempty"`
	Date         time.Time `json:"date"`
}

type UpdateIncomeRequest struct {
	AssetUUID    *string          `json:"assetUuid,omitempty"`
	Amount       *decimal.Decimal `json:"amount,omitempty"`
	Source       *string          `json:"source,omitempty"`
	CategoryUUID *string          `json:"categoryUuid,omitempty"`
	Note         *string          `json:"note,omitempty"`
	Date         *time.Time       `json:"date,omitempty"`
}
