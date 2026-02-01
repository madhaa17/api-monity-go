package service

import (
	"context"
	"errors"
	"fmt"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

type ExpenseService struct {
	repo port.ExpenseRepository
}

func NewExpenseService(repo port.ExpenseRepository) port.ExpenseService {
	return &ExpenseService{repo: repo}
}

func (s *ExpenseService) CreateExpense(ctx context.Context, userID int64, req port.CreateExpenseRequest) (*models.Expense, error) {
	// Validate amount
	if req.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	// Validate category
	if !isValidExpenseCategory(req.Category) {
		return nil, errors.New("invalid expense category")
	}

	expense := &models.Expense{
		UserID:   userID,
		Amount:   decimal.NewFromFloat(req.Amount),
		Category: req.Category,
		Note:     req.Note,
		Date:     req.Date,
	}

	if err := s.repo.Create(ctx, expense); err != nil {
		return nil, fmt.Errorf("create expense: %w", err)
	}
	return expense, nil
}

func (s *ExpenseService) GetExpense(ctx context.Context, userID int64, uuid string) (*models.Expense, error) {
	expense, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get expense: %w", err)
	}
	if expense == nil {
		return nil, errors.New("expense not found")
	}
	return expense, nil
}

func (s *ExpenseService) ListExpenses(ctx context.Context, userID int64) ([]models.Expense, error) {
	expenses, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	if expenses == nil {
		return []models.Expense{}, nil
	}
	return expenses, nil
}

func (s *ExpenseService) UpdateExpense(ctx context.Context, userID int64, uuid string, req port.UpdateExpenseRequest) (*models.Expense, error) {
	expense, err := s.GetExpense(ctx, userID, uuid)
	if err != nil {
		return nil, err
	}

	// Validate and update amount
	if req.Amount != nil {
		if *req.Amount <= 0 {
			return nil, errors.New("amount must be positive")
		}
		expense.Amount = decimal.NewFromFloat(*req.Amount)
	}

	// Validate and update category
	if req.Category != nil {
		if !isValidExpenseCategory(*req.Category) {
			return nil, errors.New("invalid expense category")
		}
		expense.Category = *req.Category
	}

	if req.Note != nil {
		expense.Note = req.Note
	}

	if req.Date != nil {
		expense.Date = *req.Date
	}

	if err := s.repo.Update(ctx, expense); err != nil {
		return nil, fmt.Errorf("update expense: %w", err)
	}
	return expense, nil
}

func (s *ExpenseService) DeleteExpense(ctx context.Context, userID int64, uuid string) error {
	if err := s.repo.Delete(ctx, uuid, userID); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

// isValidExpenseCategory validates if the category is a valid enum value
func isValidExpenseCategory(category models.ExpenseCategory) bool {
	validCategories := []models.ExpenseCategory{
		models.ExpenseCategoryFood,
		models.ExpenseCategoryTransport,
		models.ExpenseCategoryHousing,
		models.ExpenseCategoryUtilities,
		models.ExpenseCategoryHealth,
		models.ExpenseCategoryEntertainment,
		models.ExpenseCategoryShopping,
		models.ExpenseCategoryOther,
	}
	for _, valid := range validCategories {
		if category == valid {
			return true
		}
	}
	return false
}
