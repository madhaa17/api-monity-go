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

type ExpenseRepo struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) port.ExpenseRepository {
	return &ExpenseRepo{db: db}
}

func (r *ExpenseRepo) Create(ctx context.Context, expense *models.Expense) error {
	result := r.db.WithContext(ctx).Create(expense)
	if result.Error != nil {
		return fmt.Errorf("create expense: %w", result.Error)
	}
	return nil
}

func (r *ExpenseRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Expense, error) {
	var expense models.Expense
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).First(&expense)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get expense: %w", result.Error)
	}
	return &expense, nil
}

func (r *ExpenseRepo) ListByUserID(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Expense, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Expense{}).Where("user_id = ?", userID)
	if dateFrom != nil {
		q = q.Where("date >= ?", dateFrom)
	}
	if dateTo != nil {
		end := dateTo.AddDate(0, 0, 1)
		q = q.Where("date < ?", end)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count expenses: %w", err)
	}
	var expenses []models.Expense
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("date desc, created_at desc")
	if dateFrom != nil {
		result = result.Where("date >= ?", dateFrom)
	}
	if dateTo != nil {
		end := dateTo.AddDate(0, 0, 1)
		result = result.Where("date < ?", end)
	}
	result = result.Offset(offset).Limit(limit).Find(&expenses)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", result.Error)
	}
	return expenses, total, nil
}

func (r *ExpenseRepo) Update(ctx context.Context, expense *models.Expense) error {
	result := r.db.WithContext(ctx).Save(expense)
	if result.Error != nil {
		return fmt.Errorf("update expense: %w", result.Error)
	}
	return nil
}

func (r *ExpenseRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Expense{})
	if result.Error != nil {
		return fmt.Errorf("delete expense: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("expense not found or not owned by user")
	}
	return nil
}
