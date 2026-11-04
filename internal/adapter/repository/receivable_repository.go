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

type ReceivableRepo struct {
	db *gorm.DB
}

func NewReceivableRepository(db *gorm.DB) port.ReceivableRepository {
	return &ReceivableRepo{db: db}
}

func (r *ReceivableRepo) Create(ctx context.Context, receivable *models.Receivable) error {
	if err := conn(ctx, r.db).Omit(clause.Associations).Create(receivable).Error; err != nil {
		return fmt.Errorf("create receivable: %w", err)
	}
	return nil
}

func (r *ReceivableRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Receivable, error) {
	return r.getByUUID(conn(ctx, r.db).Preload("Asset"), uuid, userID)
}

func (r *ReceivableRepo) GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Receivable, error) {
	return r.getByUUID(conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}), uuid, userID)
}

func (r *ReceivableRepo) getByUUID(db *gorm.DB, uuid string, userID int64) (*models.Receivable, error) {
	if !validation.ValidUUID(uuid) {
		return nil, nil // a malformed UUID cannot match; avoid a database cast error
	}
	var receivable models.Receivable
	if err := db.Where("uuid = ? AND user_id = ?", uuid, userID).First(&receivable).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get receivable: %w", err)
	}
	return &receivable, nil
}

func (r *ReceivableRepo) ListByUserID(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Receivable, int64, error) {
	q := obligationFilter(conn(ctx, r.db).Model(&models.Receivable{}), userID, status, dueFrom, dueTo)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count receivables: %w", err)
	}
	var receivables []models.Receivable
	offset := max((page-1)*limit, 0)
	err := obligationFilter(conn(ctx, r.db).Preload("Asset"), userID, status, dueFrom, dueTo).
		Order(obligationOrder).Offset(offset).Limit(limit).Find(&receivables).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list receivables: %w", err)
	}
	return receivables, total, nil
}

func (r *ReceivableRepo) Update(ctx context.Context, receivable *models.Receivable) error {
	// Omit associations: saving a preloaded Asset would overwrite its balance with a stale copy.
	if err := conn(ctx, r.db).Omit(clause.Associations).Save(receivable).Error; err != nil {
		return fmt.Errorf("update receivable: %w", err)
	}
	return nil
}

func (r *ReceivableRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	if !validation.ValidUUID(uuid) {
		return fmt.Errorf("receivable: %w", port.ErrNotFound)
	}
	result := conn(ctx, r.db).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Receivable{})
	if result.Error != nil {
		return fmt.Errorf("delete receivable: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("receivable: %w", port.ErrNotFound)
	}
	return nil
}
