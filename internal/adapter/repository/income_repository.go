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

type IncomeRepo struct {
	db *gorm.DB
}

func NewIncomeRepository(db *gorm.DB) port.IncomeRepository {
	return &IncomeRepo{db: db}
}

func (r *IncomeRepo) Create(ctx context.Context, income *models.Income) error {
	result := conn(ctx, r.db).Create(income)
	if result.Error != nil {
		return fmt.Errorf("create income: %w", result.Error)
	}
	return nil
}

func (r *IncomeRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Income, error) {
	return r.getByUUID(conn(ctx, r.db), uuid, userID)
}

func (r *IncomeRepo) GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Income, error) {
	lock := clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "incomes"}}
	return r.getByUUID(conn(ctx, r.db).Clauses(lock), uuid, userID)
}

func (r *IncomeRepo) getByUUID(db *gorm.DB, uuid string, userID int64) (*models.Income, error) {
	if !validation.ValidUUID(uuid) {
		return nil, nil // a malformed UUID cannot match; avoid a database cast error
	}
	var income models.Income
	result := incomesTable.withRelations(db).Where("incomes.uuid = ? AND incomes.user_id = ?", uuid, userID).First(&income)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get income: %w", result.Error)
	}
	return &income, nil
}

func (r *IncomeRepo) ListByUserID(ctx context.Context, userID int64, filter port.TransactionFilter, sort port.TransactionSort, page, limit int) ([]models.Income, int64, error) {
	var incomes []models.Income
	total, err := incomesTable.list(ctx, r.db, userID, filter, sort, page, limit, &incomes)
	if err != nil {
		return nil, 0, err
	}
	return incomes, total, nil
}

func (r *IncomeRepo) Summarize(ctx context.Context, userID int64, filter port.TransactionFilter, groupBy port.SummaryGroupBy) (*port.TransactionSummary, error) {
	return incomesTable.summarize(ctx, r.db, userID, filter, groupBy)
}

func (r *IncomeRepo) Update(ctx context.Context, income *models.Income) error {
	result := conn(ctx, r.db).Save(income)
	if result.Error != nil {
		return fmt.Errorf("update income: %w", result.Error)
	}
	return nil
}

func (r *IncomeRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	result := conn(ctx, r.db).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Income{})
	if result.Error != nil {
		return fmt.Errorf("delete income: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete income: %w", port.ErrNotFound)
	}
	return nil
}
