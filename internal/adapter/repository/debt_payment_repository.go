package repository

import (
	"context"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
)

type DebtPaymentRepo struct {
	db *gorm.DB
}

func NewDebtPaymentRepository(db *gorm.DB) port.DebtPaymentRepository {
	return &DebtPaymentRepo{db: db}
}

func (r *DebtPaymentRepo) Create(ctx context.Context, payment *models.DebtPayment) error {
	result := r.db.WithContext(ctx).Create(payment)
	if result.Error != nil {
		return fmt.Errorf("create debt payment: %w", result.Error)
	}
	return nil
}

func (r *DebtPaymentRepo) ListByDebtID(ctx context.Context, debtID int64) ([]models.DebtPayment, error) {
	var payments []models.DebtPayment
	result := r.db.WithContext(ctx).Where("debt_id = ?", debtID).Order("date ASC").Find(&payments)
	if result.Error != nil {
		return nil, fmt.Errorf("list debt payments: %w", result.Error)
	}
	return payments, nil
}
