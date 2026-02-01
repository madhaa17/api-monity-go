package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

type IncomeService struct {
	repo port.IncomeRepository
}

func NewIncomeService(repo port.IncomeRepository) port.IncomeService {
	return &IncomeService{repo: repo}
}

func (s *IncomeService) CreateIncome(ctx context.Context, userID int64, req port.CreateIncomeRequest) (*models.Income, error) {
	// Validate amount
	if req.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	// Validate source
	if strings.TrimSpace(req.Source) == "" {
		return nil, errors.New("source is required")
	}

	income := &models.Income{
		UserID: userID,
		Amount: decimal.NewFromFloat(req.Amount),
		Source: req.Source,
		Note:   req.Note,
		Date:   req.Date,
	}

	if err := s.repo.Create(ctx, income); err != nil {
		return nil, fmt.Errorf("create income: %w", err)
	}
	return income, nil
}

func (s *IncomeService) GetIncome(ctx context.Context, userID int64, uuid string) (*models.Income, error) {
	income, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get income: %w", err)
	}
	if income == nil {
		return nil, errors.New("income not found")
	}
	return income, nil
}

func (s *IncomeService) ListIncomes(ctx context.Context, userID int64) ([]models.Income, error) {
	incomes, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list incomes: %w", err)
	}
	if incomes == nil {
		return []models.Income{}, nil
	}
	return incomes, nil
}

func (s *IncomeService) UpdateIncome(ctx context.Context, userID int64, uuid string, req port.UpdateIncomeRequest) (*models.Income, error) {
	income, err := s.GetIncome(ctx, userID, uuid)
	if err != nil {
		return nil, err
	}

	// Validate and update amount
	if req.Amount != nil {
		if *req.Amount <= 0 {
			return nil, errors.New("amount must be positive")
		}
		income.Amount = decimal.NewFromFloat(*req.Amount)
	}

	// Validate and update source
	if req.Source != nil {
		if strings.TrimSpace(*req.Source) == "" {
			return nil, errors.New("source cannot be empty")
		}
		income.Source = *req.Source
	}

	if req.Note != nil {
		income.Note = req.Note
	}

	if req.Date != nil {
		income.Date = *req.Date
	}

	if err := s.repo.Update(ctx, income); err != nil {
		return nil, fmt.Errorf("update income: %w", err)
	}
	return income, nil
}

func (s *IncomeService) DeleteIncome(ctx context.Context, userID int64, uuid string) error {
	if err := s.repo.Delete(ctx, uuid, userID); err != nil {
		return fmt.Errorf("delete income: %w", err)
	}
	return nil
}
