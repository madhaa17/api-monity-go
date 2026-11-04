package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DebtRepo struct {
	db *gorm.DB
}

func NewDebtRepository(db *gorm.DB) port.DebtRepository {
	return &DebtRepo{db: db}
}

func (r *DebtRepo) Create(ctx context.Context, debt *models.Debt) error {
	if err := conn(ctx, r.db).Omit(clause.Associations).Create(debt).Error; err != nil {
		return fmt.Errorf("create debt: %w", err)
	}
	return nil
}

func (r *DebtRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Debt, error) {
	return r.getByUUID(conn(ctx, r.db).Preload("Asset"), uuid, userID)
}

func (r *DebtRepo) GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Debt, error) {
	return r.getByUUID(conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}), uuid, userID)
}

func (r *DebtRepo) getByUUID(db *gorm.DB, uuid string, userID int64) (*models.Debt, error) {
	if !validation.ValidUUID(uuid) {
		return nil, nil // a malformed UUID cannot match; avoid a database cast error
	}
	var debt models.Debt
	if err := db.Where("uuid = ? AND user_id = ?", uuid, userID).First(&debt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get debt: %w", err)
	}
	return &debt, nil
}

func (r *DebtRepo) ListByUserID(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Debt, int64, error) {
	q := obligationFilter(conn(ctx, r.db).Model(&models.Debt{}), userID, status, dueFrom, dueTo)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count debts: %w", err)
	}
	var debts []models.Debt
	offset := max((page-1)*limit, 0)
	err := obligationFilter(conn(ctx, r.db).Preload("Asset"), userID, status, dueFrom, dueTo).
		Order(obligationOrder).Offset(offset).Limit(limit).Find(&debts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list debts: %w", err)
	}
	return debts, total, nil
}

func (r *DebtRepo) Update(ctx context.Context, debt *models.Debt) error {
	// Omit associations: saving a preloaded Asset would overwrite its balance with a stale copy.
	if err := conn(ctx, r.db).Omit(clause.Associations).Save(debt).Error; err != nil {
		return fmt.Errorf("update debt: %w", err)
	}
	return nil
}

func (r *DebtRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	if !validation.ValidUUID(uuid) {
		return fmt.Errorf("debt: %w", port.ErrNotFound)
	}
	result := conn(ctx, r.db).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Debt{})
	if result.Error != nil {
		return fmt.Errorf("delete debt: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("debt: %w", port.ErrNotFound)
	}
	return nil
}
