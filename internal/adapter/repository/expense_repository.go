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
	if !validation.ValidUUID(uuid) {
		return nil, nil // a malformed UUID cannot match; avoid a database cast error
	}
	var expense models.Expense
	result := expensesTable.withRelations(db).Where("expenses.uuid = ? AND expenses.user_id = ?", uuid, userID).First(&expense)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get expense: %w", result.Error)
	}
	return &expense, nil
}

func (r *ExpenseRepo) ListByUserID(ctx context.Context, userID int64, filter port.TransactionFilter, sort port.TransactionSort, page, limit int) ([]models.Expense, int64, error) {
	var expenses []models.Expense
	total, err := expensesTable.list(ctx, r.db, userID, filter, sort, page, limit, &expenses)
	if err != nil {
		return nil, 0, err
	}
	return expenses, total, nil
}

func (r *ExpenseRepo) Summarize(ctx context.Context, userID int64, filter port.TransactionFilter, groupBy port.SummaryGroupBy) (*port.TransactionSummary, error) {
	return expensesTable.summarize(ctx, r.db, userID, filter, groupBy)
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
