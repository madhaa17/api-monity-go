package port

import (
	"context"
	"monity/internal/models"
	"time"
)

type ReceivableRepository interface {
	Create(ctx context.Context, rec *models.Receivable) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Receivable, error)
	ListByUserID(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Receivable, int64, error)
	Update(ctx context.Context, rec *models.Receivable) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type ReceivablePaymentRepository interface {
	Create(ctx context.Context, payment *models.ReceivablePayment) error
	ListByReceivableID(ctx context.Context, receivableID int64) ([]models.ReceivablePayment, error)
}

type ReceivableService interface {
	CreateReceivable(ctx context.Context, userID int64, req CreateReceivableRequest) (*models.Receivable, error)
	GetReceivable(ctx context.Context, userID int64, uuid string) (*models.Receivable, error)
	ListReceivables(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Receivable, ListMeta, error)
	UpdateReceivable(ctx context.Context, userID int64, uuid string, req UpdateReceivableRequest) (*models.Receivable, error)
	DeleteReceivable(ctx context.Context, userID int64, uuid string) error
	RecordReceivablePayment(ctx context.Context, userID int64, receivableUUID string, req CreateReceivablePaymentRequest) (*models.ReceivablePayment, error)
	ListReceivablePayments(ctx context.Context, userID int64, receivableUUID string) ([]models.ReceivablePayment, error)
}

type CreateReceivableRequest struct {
	PartyName string     `json:"partyName"`
	Amount    float64    `json:"amount"`
	DueDate   *time.Time `json:"dueDate,omitempty"`
	Note      *string    `json:"note,omitempty"`
	AssetUUID *string    `json:"assetUuid,omitempty"`
}

type UpdateReceivableRequest struct {
	PartyName *string    `json:"partyName,omitempty"`
	Amount    *float64   `json:"amount,omitempty"`
	DueDate   *time.Time `json:"dueDate,omitempty"`
	Note      *string    `json:"note,omitempty"`
	AssetUUID *string    `json:"assetUuid,omitempty"`
}

type CreateReceivablePaymentRequest struct {
	Amount    float64    `json:"amount"`
	Date      time.Time  `json:"date"`
	Note      *string    `json:"note,omitempty"`
	AssetUUID *string    `json:"assetUuid,omitempty"`
}
