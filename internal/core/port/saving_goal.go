package port

import (
	"context"
	"monity/internal/models"
	"time"
)

type SavingGoalRepository interface {
	Create(ctx context.Context, goal *models.SavingGoal) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.SavingGoal, error)
	ListByUserID(ctx context.Context, userID int64) ([]models.SavingGoal, error)
	Update(ctx context.Context, goal *models.SavingGoal) error
	Delete(ctx context.Context, uuid string, userID int64) error
}

type SavingGoalService interface {
	CreateSavingGoal(ctx context.Context, userID int64, req CreateSavingGoalRequest) (*models.SavingGoal, error)
	GetSavingGoal(ctx context.Context, userID int64, uuid string) (*models.SavingGoal, error)
	ListSavingGoals(ctx context.Context, userID int64) ([]models.SavingGoal, error)
	UpdateSavingGoal(ctx context.Context, userID int64, uuid string, req UpdateSavingGoalRequest) (*models.SavingGoal, error)
	DeleteSavingGoal(ctx context.Context, userID int64, uuid string) error
}

type CreateSavingGoalRequest struct {
	Title         string     `json:"title"`
	TargetAmount  float64    `json:"targetAmount"`
	CurrentAmount float64    `json:"currentAmount"`
	Deadline      *time.Time `json:"deadline,omitempty"`
}

type UpdateSavingGoalRequest struct {
	Title         *string    `json:"title,omitempty"`
	TargetAmount  *float64   `json:"targetAmount,omitempty"`
	CurrentAmount *float64   `json:"currentAmount,omitempty"`
	Deadline      *time.Time `json:"deadline,omitempty"`
}
