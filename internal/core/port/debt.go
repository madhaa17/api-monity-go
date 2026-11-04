package port

import (
	"context"
	"time"

	"monity/internal/models"

	"github.com/shopspring/decimal"
)

// DebtRepository persists debts (money the user owes someone). Calls made with a transaction ctx
// (see Transactor) take part in that transaction.
type DebtRepository interface {
	Create(ctx context.Context, debt *models.Debt) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Debt, error)
	// GetByUUIDForUpdate is GetByUUID that also row-locks the debt until the transaction ends.
	GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Debt, error)
	ListByUserID(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Debt, int64, error)
	Update(ctx context.Context, debt *models.Debt) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type DebtPaymentRepository interface {
	Create(ctx context.Context, payment *models.DebtPayment) error
	GetByUUID(ctx context.Context, uuid string, debtID int64) (*models.DebtPayment, error)
	ListByDebtID(ctx context.Context, debtID int64) ([]models.DebtPayment, error)
	Delete(ctx context.Context, id int64) error
}

type DebtService interface {
	CreateDebt(ctx context.Context, userID int64, req CreateDebtRequest) (*models.Debt, error)
	GetDebt(ctx context.Context, userID int64, uuid string) (*models.Debt, error)
	ListDebts(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Debt, ListMeta, error)
	UpdateDebt(ctx context.Context, userID int64, uuid string, req UpdateDebtRequest) (*models.Debt, error)
	DeleteDebt(ctx context.Context, userID int64, uuid string) error
	RecordDebtPayment(ctx context.Context, userID int64, debtUUID string, req CreateDebtPaymentRequest) (*models.DebtPayment, error)
	ListDebtPayments(ctx context.Context, userID int64, debtUUID string) ([]models.DebtPayment, error)
	DeleteDebtPayment(ctx context.Context, userID int64, debtUUID, paymentUUID string) error
}

// CreateDebtRequest records money the user owes. When AssetUUID names a CASH asset, the borrowed
// amount is added to it; without one the debt is only a record.
type CreateDebtRequest struct {
	PartyName string          `json:"partyName"`
	Amount    decimal.Decimal `json:"amount"`
	DueDate   *time.Time      `json:"dueDate,omitempty"`
	Note      *string         `json:"note,omitempty"`
	AssetUUID *string         `json:"assetUuid,omitempty"`
}

// UpdateDebtRequest is a partial update: absent fields are left unchanged, and null clears
// DueDate, Note or AssetUUID.
type UpdateDebtRequest struct {
	PartyName *string             `json:"partyName,omitempty"`
	Amount    *decimal.Decimal    `json:"amount,omitempty"`
	DueDate   Optional[time.Time] `json:"dueDate"`
	Note      Optional[string]    `json:"note"`
	AssetUUID Optional[string]    `json:"assetUuid"`
}

// CreateDebtPaymentRequest records a repayment. When AssetUUID names a CASH asset, the amount is
// taken out of it.
type CreateDebtPaymentRequest struct {
	Amount    decimal.Decimal `json:"amount"`
	Date      time.Time       `json:"date"`
	Note      *string         `json:"note,omitempty"`
	AssetUUID *string         `json:"assetUuid,omitempty"`
}
