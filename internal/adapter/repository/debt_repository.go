package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
)

type DebtRepo struct {
	db *gorm.DB
}

func NewDebtRepository(db *gorm.DB) port.DebtRepository {
	return &DebtRepo{db: db}
}

func (r *DebtRepo) Create(ctx context.Context, debt *models.Debt) error {
	result := r.db.WithContext(ctx).Create(debt)
	if result.Error != nil {
		return fmt.Errorf("create debt: %w", result.Error)
	}
	return nil
}

func (r *DebtRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Debt, error) {
	var debt models.Debt
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).First(&debt)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get debt: %w", result.Error)
	}
	return &debt, nil
}

func (r *DebtRepo) ListByUserID(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Debt, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Debt{}).Where("user_id = ?", userID)
	if status != nil && *status != "" {
		q = q.Where("status = ?", *status)
	}
	if dueFrom != nil {
		q = q.Where("due_date >= ?", dueFrom)
	}
	if dueTo != nil {
		end := dueTo.AddDate(0, 0, 1)
		q = q.Where("due_date < ?", end)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count debts: %w", err)
	}
	var debts []models.Debt
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	listQ := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != nil && *status != "" {
		listQ = listQ.Where("status = ?", *status)
	}
	if dueFrom != nil {
		listQ = listQ.Where("due_date >= ?", dueFrom)
	}
	if dueTo != nil {
		end := dueTo.AddDate(0, 0, 1)
		listQ = listQ.Where("due_date < ?", end)
	}
	result := listQ.Order("due_date ASC NULLS LAST, created_at DESC").Offset(offset).Limit(limit).Find(&debts)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("list debts: %w", result.Error)
	}
	return debts, total, nil
}

func (r *DebtRepo) Update(ctx context.Context, debt *models.Debt) error {
	result := r.db.WithContext(ctx).Save(debt)
	if result.Error != nil {
		return fmt.Errorf("update debt: %w", result.Error)
	}
	return nil
}

func (r *DebtRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Debt{})
	if result.Error != nil {
		return fmt.Errorf("delete debt: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("debt not found or not owned by user")
	}
	return nil
}
