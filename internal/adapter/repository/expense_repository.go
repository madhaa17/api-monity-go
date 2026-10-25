package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExpenseRepo struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) port.ExpenseRepository {
	return &ExpenseRepo{db: db}
}

func (r *ExpenseRepo) Create(ctx context.Context, expense *models.Expense) error {
	result := conn(ctx, r.db).Create(expense)
	if result.Error != nil {
		return fmt.Errorf("create expense: %w", result.Error)
	}
	return nil
}

func (r *ExpenseRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Expense, error) {
	return r.getByUUID(conn(ctx, r.db), uuid, userID)
}

func (r *ExpenseRepo) GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Expense, error) {
	lock := clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "expenses"}}
	return r.getByUUID(conn(ctx, r.db).Clauses(lock), uuid, userID)
}

func (r *ExpenseRepo) getByUUID(db *gorm.DB, uuid string, userID int64) (*models.Expense, error) {
	var expense models.Expense
	result := withCategory(db, "expenses").Where("expenses.uuid = ? AND expenses.user_id = ?", uuid, userID).First(&expense)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get expense: %w", result.Error)
	}
	return &expense, nil
}

func (r *ExpenseRepo) ListByUserID(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Expense, int64, error) {
	filter := func(q *gorm.DB) *gorm.DB {
		q = q.Where("expenses.user_id = ?", userID)
		if dateFrom != nil {
			q = q.Where("expenses.date >= ?", dateFrom)
		}
		if dateTo != nil {
			q = q.Where("expenses.date < ?", dateTo.AddDate(0, 0, 1))
		}
		return q
	}
	var total int64
	if err := filter(conn(ctx, r.db).Model(&models.Expense{})).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count expenses: %w", err)
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	var expenses []models.Expense
	result := filter(withCategory(conn(ctx, r.db), "expenses")).
		Order("expenses.date desc, expenses.created_at desc").
		Offset(offset).Limit(limit).Find(&expenses)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", result.Error)
	}
	return expenses, total, nil
}

func (r *ExpenseRepo) Update(ctx context.Context, expense *models.Expense) error {
	result := conn(ctx, r.db).Save(expense)
	if result.Error != nil {
		return fmt.Errorf("update expense: %w", result.Error)
	}
	return nil
}

func (r *ExpenseRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	result := conn(ctx, r.db).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Expense{})
	if result.Error != nil {
		return fmt.Errorf("delete expense: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete expense: %w", port.ErrNotFound)
	}
	return nil
}
