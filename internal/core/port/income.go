package port

import (
	"context"
	"monity/internal/models"
	"time"
)

type IncomeRepository interface {
	Create(ctx context.Context, income *models.Income) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Income, error)
	ListByUserID(ctx context.Context, userID int64) ([]models.Income, error)
	Update(ctx context.Context, income *models.Income) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type IncomeService interface {
	CreateIncome(ctx context.Context, userID int64, req CreateIncomeRequest) (*models.Income, error)
	GetIncome(ctx context.Context, userID int64, uuid string) (*models.Income, error)
	ListIncomes(ctx context.Context, userID int64) ([]models.Income, error)
	UpdateIncome(ctx context.Context, userID int64, uuid string, req UpdateIncomeRequest) (*models.Income, error)
	DeleteIncome(ctx context.Context, userID int64, uuid string) error
}

type CreateIncomeRequest struct {
	AssetUUID string    `json:"assetUuid"`
	Amount    float64   `json:"amount"`
	Source    string    `json:"source"`
	Note      *string   `json:"note,omitempty"`
	Date      time.Time `json:"date"`
}

type UpdateIncomeRequest struct {
	AssetUUID *string    `json:"assetUuid,omitempty"`
	Amount    *float64   `json:"amount,omitempty"`
	Source    *string    `json:"source,omitempty"`
	Note      *string    `json:"note,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
}
