package repository

import (
	"context"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
)

type ReceivablePaymentRepo struct {
	db *gorm.DB
}

func NewReceivablePaymentRepository(db *gorm.DB) port.ReceivablePaymentRepository {
	return &ReceivablePaymentRepo{db: db}
}

func (r *ReceivablePaymentRepo) Create(ctx context.Context, payment *models.ReceivablePayment) error {
	result := r.db.WithContext(ctx).Create(payment)
	if result.Error != nil {
		return fmt.Errorf("create receivable payment: %w", result.Error)
	}
	return nil
}

func (r *ReceivablePaymentRepo) ListByReceivableID(ctx context.Context, receivableID int64) ([]models.ReceivablePayment, error) {
	var payments []models.ReceivablePayment
	result := r.db.WithContext(ctx).Where("receivable_id = ?", receivableID).Order("date ASC").Find(&payments)
	if result.Error != nil {
		return nil, fmt.Errorf("list receivable payments: %w", result.Error)
	}
	return payments, nil
}
