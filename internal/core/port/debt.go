package port

import (
	"context"
	"monity/internal/models"
	"time"
)

type DebtRepository interface {
	Create(ctx context.Context, debt *models.Debt) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Debt, error)
	ListByUserID(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Debt, int64, error)
	Update(ctx context.Context, debt *models.Debt) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type DebtPaymentRepository interface {
	Create(ctx context.Context, payment *models.DebtPayment) error
	ListByDebtID(ctx context.Context, debtID int64) ([]models.DebtPayment, error)
}

type DebtService interface {
	CreateDebt(ctx context.Context, userID int64, req CreateDebtRequest) (*models.Debt, error)
	GetDebt(ctx context.Context, userID int64, uuid string) (*models.Debt, error)
	ListDebts(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Debt, ListMeta, error)
	UpdateDebt(ctx context.Context, userID int64, uuid string, req UpdateDebtRequest) (*models.Debt, error)
	DeleteDebt(ctx context.Context, userID int64, uuid string) error
	RecordDebtPayment(ctx context.Context, userID int64, debtUUID string, req CreateDebtPaymentRequest) (*models.DebtPayment, error)
	ListDebtPayments(ctx context.Context, userID int64, debtUUID string) ([]models.DebtPayment, error)
}

type CreateDebtRequest struct {
	PartyName string     `json:"partyName"`
	Amount    float64    `json:"amount"`
	DueDate   *time.Time `json:"dueDate,omitempty"`
	Note      *string    `json:"note,omitempty"`
	AssetUUID *string    `json:"assetUuid,omitempty"`
}

type UpdateDebtRequest struct {
	PartyName *string    `json:"partyName,omitempty"`
	Amount    *float64   `json:"amount,omitempty"`
	DueDate   *time.Time `json:"dueDate,omitempty"`
	Note      *string    `json:"note,omitempty"`
	AssetUUID *string    `json:"assetUuid,omitempty"`
}

type CreateDebtPaymentRequest struct {
	Amount    float64    `json:"amount"`
	Date      time.Time  `json:"date"`
	Note      *string    `json:"note,omitempty"`
	AssetUUID *string    `json:"assetUuid,omitempty"`
}
