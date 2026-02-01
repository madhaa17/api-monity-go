package repository

import (
	"context"
	"errors"
	"fmt"

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

func (r *ExpenseRepo) ListByUserID(ctx context.Context, userID int64) ([]models.Expense, error) {
	var expenses []models.Expense
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("date desc, created_at desc").Find(&expenses)
	if result.Error != nil {
		return nil, fmt.Errorf("list expenses: %w", result.Error)
	}
	return expenses, nil
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
