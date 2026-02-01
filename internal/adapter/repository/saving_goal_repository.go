package repository

import (
	"context"
	"errors"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
)

type SavingGoalRepo struct {
	db *gorm.DB
}

func NewSavingGoalRepository(db *gorm.DB) port.SavingGoalRepository {
	return &SavingGoalRepo{db: db}
}

func (r *SavingGoalRepo) Create(ctx context.Context, goal *models.SavingGoal) error {
	result := r.db.WithContext(ctx).Create(goal)
	if result.Error != nil {
		return fmt.Errorf("create saving goal: %w", result.Error)
	}
	return nil
}

func (r *SavingGoalRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.SavingGoal, error) {
	var goal models.SavingGoal
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).First(&goal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get saving goal: %w", result.Error)
	}
	return &goal, nil
}

func (r *SavingGoalRepo) ListByUserID(ctx context.Context, userID int64) ([]models.SavingGoal, error) {
	var goals []models.SavingGoal
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&goals)
	if result.Error != nil {
		return nil, fmt.Errorf("list saving goals: %w", result.Error)
	}
	return goals, nil
}

func (r *SavingGoalRepo) Update(ctx context.Context, goal *models.SavingGoal) error {
	result := r.db.WithContext(ctx).Save(goal)
	if result.Error != nil {
		return fmt.Errorf("update saving goal: %w", result.Error)
	}
	return nil
}

func (r *SavingGoalRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	result := r.db.WithContext(ctx).Where("uuid = ? AND user_id = ?", uuid, userID).Delete(&models.SavingGoal{})
	if result.Error != nil {
		return fmt.Errorf("delete saving goal: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("saving goal not found or not owned by user")
	}
	return nil
}
