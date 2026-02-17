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

type ReceivableRepo struct {
	db *gorm.DB
}

func NewReceivableRepository(db *gorm.DB) port.ReceivableRepository {
	return &ReceivableRepo{db: db}
}

func (r *ReceivableRepo) Create(ctx context.Context, rec *models.Receivable) error {
	result := r.db.WithContext(ctx).Create(rec)
	if result.Error != nil {
		return fmt.Errorf("create receivable: %w", result.Error)
	}
	return nil
}

func (r *ReceivableRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Receivable, error) {
	var rec models.Receivable
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).First(&rec)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get receivable: %w", result.Error)
	}
	return &rec, nil
}

func (r *ReceivableRepo) ListByUserID(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Receivable, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Receivable{}).Where("user_id = ?", userID)
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
		return nil, 0, fmt.Errorf("count receivables: %w", err)
	}
	var recs []models.Receivable
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
	result := listQ.Order("due_date ASC NULLS LAST, created_at DESC").Offset(offset).Limit(limit).Find(&recs)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("list receivables: %w", result.Error)
	}
	return recs, total, nil
}

func (r *ReceivableRepo) Update(ctx context.Context, rec *models.Receivable) error {
	result := r.db.WithContext(ctx).Save(rec)
	if result.Error != nil {
		return fmt.Errorf("update receivable: %w", result.Error)
	}
	return nil
}

func (r *ReceivableRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Receivable{})
	if result.Error != nil {
		return fmt.Errorf("delete receivable: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("receivable not found or not owned by user")
	}
	return nil
}
