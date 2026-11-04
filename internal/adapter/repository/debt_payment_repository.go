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

type DebtPaymentRepo struct {
	db *gorm.DB
}

func NewDebtPaymentRepository(db *gorm.DB) port.DebtPaymentRepository {
	return &DebtPaymentRepo{db: db}
}

func (r *DebtPaymentRepo) Create(ctx context.Context, payment *models.DebtPayment) error {
	if err := conn(ctx, r.db).Omit(clause.Associations).Create(payment).Error; err != nil {
		return fmt.Errorf("create debt payment: %w", err)
	}
	return nil
}

func (r *DebtPaymentRepo) GetByUUID(ctx context.Context, uuid string, debtID int64) (*models.DebtPayment, error) {
	if !validation.ValidUUID(uuid) {
		return nil, nil
	}
	var payment models.DebtPayment
	err := conn(ctx, r.db).Where("uuid = ? AND debt_id = ?", uuid, debtID).First(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get debt payment: %w", err)
	}
	return &payment, nil
}

func (r *DebtPaymentRepo) ListByDebtID(ctx context.Context, debtID int64) ([]models.DebtPayment, error) {
	var payments []models.DebtPayment
	err := conn(ctx, r.db).Preload("Asset").Where("debt_id = ?", debtID).
		Order("date DESC, created_at DESC").Find(&payments).Error
	if err != nil {
		return nil, fmt.Errorf("list debt payments: %w", err)
	}
	return payments, nil
}

func (r *DebtPaymentRepo) Delete(ctx context.Context, id int64) error {
	if err := conn(ctx, r.db).Delete(&models.DebtPayment{}, id).Error; err != nil {
		return fmt.Errorf("delete debt payment: %w", err)
	}
	return nil
}
