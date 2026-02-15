package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

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
	if err := validation.CheckMaxLen(req.Title, validation.MaxTitleLen); err != nil {
		return nil, fmt.Errorf("title %w", err)
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

func (s *SavingGoalService) ListSavingGoals(ctx context.Context, userID int64, page, limit int) ([]models.SavingGoal, port.ListMeta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	goals, total, err := s.repo.ListByUserID(ctx, userID, page, limit)
	if err != nil {
		return nil, port.ListMeta{}, fmt.Errorf("list saving goals: %w", err)
	}
	if goals == nil {
		goals = []models.SavingGoal{}
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages < 0 {
		totalPages = 0
	}
	meta := port.ListMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
	return goals, meta, nil
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
		if err := validation.CheckMaxLen(*req.Title, validation.MaxTitleLen); err != nil {
			return nil, fmt.Errorf("title %w", err)
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
