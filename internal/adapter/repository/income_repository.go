package repository

import (
	"context"
	"errors"
	"fmt"

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

func (r *IncomeRepo) ListByUserID(ctx context.Context, userID int64) ([]models.Income, error) {
	var incomes []models.Income
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("date desc, created_at desc").Find(&incomes)
	if result.Error != nil {
		return nil, fmt.Errorf("list incomes: %w", result.Error)
	}
	return incomes, nil
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
