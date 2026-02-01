package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

type SavingGoalService struct {
	repo port.SavingGoalRepository
}

func NewSavingGoalService(repo port.SavingGoalRepository) port.SavingGoalService {
	return &SavingGoalService{repo: repo}
}

func (s *SavingGoalService) CreateSavingGoal(ctx context.Context, userID int64, req port.CreateSavingGoalRequest) (*models.SavingGoal, error) {
	// Validate title
	if strings.TrimSpace(req.Title) == "" {
		return nil, errors.New("title is required")
	}

	// Validate target amount
	if req.TargetAmount <= 0 {
		return nil, errors.New("target amount must be positive")
	}

	// Validate current amount
	if req.CurrentAmount < 0 {
		return nil, errors.New("current amount cannot be negative")
	}

	now := time.Now()
	goal := &models.SavingGoal{
		UserID:        userID,
		Title:         strings.TrimSpace(req.Title),
		TargetAmount:  decimal.NewFromFloat(req.TargetAmount),
		CurrentAmount: decimal.NewFromFloat(req.CurrentAmount),
		Deadline:      req.Deadline,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, goal); err != nil {
		return nil, fmt.Errorf("create saving goal: %w", err)
	}
	return goal, nil
}

func (s *SavingGoalService) GetSavingGoal(ctx context.Context, userID int64, uuid string) (*models.SavingGoal, error) {
	goal, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get saving goal: %w", err)
	}
	if goal == nil {
		return nil, errors.New("saving goal not found")
	}
	return goal, nil
}

func (s *SavingGoalService) ListSavingGoals(ctx context.Context, userID int64) ([]models.SavingGoal, error) {
	goals, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list saving goals: %w", err)
	}
	if goals == nil {
		return []models.SavingGoal{}, nil
	}
	return goals, nil
}

func (s *SavingGoalService) UpdateSavingGoal(ctx context.Context, userID int64, uuid string, req port.UpdateSavingGoalRequest) (*models.SavingGoal, error) {
	goal, err := s.GetSavingGoal(ctx, userID, uuid)
	if err != nil {
		return nil, err
	}

	// Validate and update title
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, errors.New("title cannot be empty")
		}
		goal.Title = strings.TrimSpace(*req.Title)
	}

	// Validate and update target amount
	if req.TargetAmount != nil {
		if *req.TargetAmount <= 0 {
			return nil, errors.New("target amount must be positive")
		}
		goal.TargetAmount = decimal.NewFromFloat(*req.TargetAmount)
	}

	// Validate and update current amount
	if req.CurrentAmount != nil {
		if *req.CurrentAmount < 0 {
			return nil, errors.New("current amount cannot be negative")
		}
		goal.CurrentAmount = decimal.NewFromFloat(*req.CurrentAmount)
	}

	// Update deadline (can be set to nil)
	if req.Deadline != nil {
		goal.Deadline = req.Deadline
	}

	goal.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, goal); err != nil {
		return nil, fmt.Errorf("update saving goal: %w", err)
	}
	return goal, nil
}

func (s *SavingGoalService) DeleteSavingGoal(ctx context.Context, userID int64, uuid string) error {
	if err := s.repo.Delete(ctx, uuid, userID); err != nil {
		return fmt.Errorf("delete saving goal: %w", err)
	}
	return nil
}
