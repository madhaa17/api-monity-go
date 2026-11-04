package repository

import (
	"context"
	"errors"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReceivablePaymentRepo struct {
	db *gorm.DB
}

func NewReceivablePaymentRepository(db *gorm.DB) port.ReceivablePaymentRepository {
	return &ReceivablePaymentRepo{db: db}
}

func (r *ReceivablePaymentRepo) Create(ctx context.Context, payment *models.ReceivablePayment) error {
	if err := conn(ctx, r.db).Omit(clause.Associations).Create(payment).Error; err != nil {
		return fmt.Errorf("create receivable payment: %w", err)
	}
	return nil
}

func (r *ReceivablePaymentRepo) GetByUUID(ctx context.Context, uuid string, receivableID int64) (*models.ReceivablePayment, error) {
	if !validation.ValidUUID(uuid) {
		return nil, nil
	}
	var payment models.ReceivablePayment
	err := conn(ctx, r.db).Where("uuid = ? AND receivable_id = ?", uuid, receivableID).First(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get receivable payment: %w", err)
	}
	return &payment, nil
}

func (r *ReceivablePaymentRepo) ListByReceivableID(ctx context.Context, receivableID int64) ([]models.ReceivablePayment, error) {
	var payments []models.ReceivablePayment
	err := conn(ctx, r.db).Preload("Asset").Where("receivable_id = ?", receivableID).
		Order("date DESC, created_at DESC").Find(&payments).Error
	if err != nil {
		return nil, fmt.Errorf("list receivable payments: %w", err)
	}
	return payments, nil
}

func (r *ReceivablePaymentRepo) Delete(ctx context.Context, id int64) error {
	if err := conn(ctx, r.db).Delete(&models.ReceivablePayment{}, id).Error; err != nil {
		return fmt.Errorf("delete receivable payment: %w", err)
	}
	return nil
}
