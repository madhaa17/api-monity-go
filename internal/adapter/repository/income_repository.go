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

type IncomeRepo struct {
	db *gorm.DB
}

func NewIncomeRepository(db *gorm.DB) port.IncomeRepository {
	return &IncomeRepo{db: db}
}

func (r *IncomeRepo) Create(ctx context.Context, income *models.Income) error {
	result := r.db.WithContext(ctx).Create(income)
	if result.Error != nil {
		return fmt.Errorf("create income: %w", result.Error)
	}
	return nil
}

func (r *IncomeRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Income, error) {
	var income models.Income
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).First(&income)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get income: %w", result.Error)
	}
	return &income, nil
}

func (r *IncomeRepo) ListByUserID(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Income, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Income{}).Where("user_id = ?", userID)
	if dateFrom != nil {
		q = q.Where("date >= ?", dateFrom)
	}
	if dateTo != nil {
		end := dateTo.AddDate(0, 0, 1)
		q = q.Where("date < ?", end)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count incomes: %w", err)
	}
	var incomes []models.Income
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
	result = result.Offset(offset).Limit(limit).Find(&incomes)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("list incomes: %w", result.Error)
	}
	return incomes, total, nil
}

func (r *IncomeRepo) Update(ctx context.Context, income *models.Income) error {
	result := r.db.WithContext(ctx).Save(income)
	if result.Error != nil {
		return fmt.Errorf("update income: %w", result.Error)
	}
	return nil
}

func (r *IncomeRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.Income{})
	if result.Error != nil {
		return fmt.Errorf("delete income: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("income not found or not owned by user")
	}
	return nil
}
