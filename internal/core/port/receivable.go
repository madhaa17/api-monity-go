package port

import (
	"context"
	"time"

	"monity/internal/models"

	"github.com/shopspring/decimal"
)

// ReceivableRepository persists receivables (money someone owes the user). Calls made with a transaction ctx
// (see Transactor) take part in that transaction.
type ReceivableRepository interface {
	Create(ctx context.Context, receivable *models.Receivable) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Receivable, error)
	// GetByUUIDForUpdate is GetByUUID that also row-locks the receivable until the transaction ends.
	GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Receivable, error)
	ListByUserID(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Receivable, int64, error)
	Update(ctx context.Context, receivable *models.Receivable) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type ReceivablePaymentRepository interface {
	Create(ctx context.Context, payment *models.ReceivablePayment) error
	GetByUUID(ctx context.Context, uuid string, receivableID int64) (*models.ReceivablePayment, error)
	ListByReceivableID(ctx context.Context, receivableID int64) ([]models.ReceivablePayment, error)
	Delete(ctx context.Context, id int64) error
}

type ReceivableService interface {
	CreateReceivable(ctx context.Context, userID int64, req CreateReceivableRequest) (*models.Receivable, error)
	GetReceivable(ctx context.Context, userID int64, uuid string) (*models.Receivable, error)
	ListReceivables(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Receivable, ListMeta, error)
	UpdateReceivable(ctx context.Context, userID int64, uuid string, req UpdateReceivableRequest) (*models.Receivable, error)
	DeleteReceivable(ctx context.Context, userID int64, uuid string) error
	RecordReceivablePayment(ctx context.Context, userID int64, receivableUUID string, req CreateReceivablePaymentRequest) (*models.ReceivablePayment, error)
	ListReceivablePayments(ctx context.Context, userID int64, receivableUUID string) ([]models.ReceivablePayment, error)
	DeleteReceivablePayment(ctx context.Context, userID int64, receivableUUID, paymentUUID string) error
}

// CreateReceivableRequest records money lent to someone. When AssetUUID names a CASH asset, the lent
// amount is taken out of it; without one the receivable is only a record.
type CreateReceivableRequest struct {
	PartyName string          `json:"partyName"`
	Amount    decimal.Decimal `json:"amount"`
	DueDate   *time.Time      `json:"dueDate,omitempty"`
	Note      *string         `json:"note,omitempty"`
	AssetUUID *string         `json:"assetUuid,omitempty"`
}

// UpdateReceivableRequest is a partial update: absent fields are left unchanged, and null clears
// DueDate, Note or AssetUUID.
type UpdateReceivableRequest struct {
	PartyName *string             `json:"partyName,omitempty"`
	Amount    *decimal.Decimal    `json:"amount,omitempty"`
	DueDate   Optional[time.Time] `json:"dueDate"`
	Note      Optional[string]    `json:"note"`
	AssetUUID Optional[string]    `json:"assetUuid"`
}

// CreateReceivablePaymentRequest records money received back. When AssetUUID names a CASH asset, the amount is
// added to it.
type CreateReceivablePaymentRequest struct {
	Amount    decimal.Decimal `json:"amount"`
	Date      time.Time       `json:"date"`
	Note      *string         `json:"note,omitempty"`
	AssetUUID *string         `json:"assetUuid,omitempty"`
}
