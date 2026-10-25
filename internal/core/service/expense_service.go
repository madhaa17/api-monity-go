package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
)

const expenseInsufficientMsg = "expense amount cannot exceed the selected asset balance"

type ExpenseService struct {
	repo         port.ExpenseRepository
	assetRepo    port.AssetRepository
	categoryRepo port.CategoryRepository
	tx           port.Transactor
}

func NewExpenseService(repo port.ExpenseRepository, assetRepo port.AssetRepository, categoryRepo port.CategoryRepository, tx port.Transactor) port.ExpenseService {
	return &ExpenseService{repo: repo, assetRepo: assetRepo, categoryRepo: categoryRepo, tx: tx}
}

func (s *ExpenseService) CreateExpense(ctx context.Context, userID int64, req port.CreateExpenseRequest) (*models.Expense, error) {
	if err := validateCashAmount(req.Amount); err != nil {
		return nil, err
	}
	if err := validateTransactionDate(req.Date); err != nil {
		return nil, err
	}
	if err := validateNote(req.Note); err != nil {
		return nil, err
	}

	category, err := resolveCategory(ctx, s.categoryRepo, userID, models.CategoryTypeExpense, req.CategoryUUID, req.Category)
	if err != nil {
		return nil, err
	}

	var expense *models.Expense
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		asset, err := lookupCashAsset(ctx, s.assetRepo, req.AssetUUID, userID)
		if err != nil {
			return err
		}
		if err := adjustCashBalance(ctx, s.assetRepo, asset.ID, req.Amount.Neg(), expenseInsufficientMsg); err != nil {
			return err
		}
		expense = &models.Expense{
			UserID:       userID,
			AssetID:      asset.ID,
			Amount:       req.Amount,
			CategoryID:   category.ID,
			CategoryUUID: category.UUID,
			CategoryName: category.Name,
			Note:         req.Note,
			Date:         req.Date,
		}
		if err := s.repo.Create(ctx, expense); err != nil {
			return fmt.Errorf("create expense: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slog.Info("expense_created", "user_id", userID, "amount", req.Amount.String(), "category", category.Name, "asset_uuid", req.AssetUUID)
	return expense, nil
}

func (s *ExpenseService) GetExpense(ctx context.Context, userID int64, uuid string) (*models.Expense, error) {
	expense, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get expense: %w", err)
	}
	if expense == nil {
		return nil, fmt.Errorf("expense: %w", port.ErrNotFound)
	}
	return expense, nil
}

func (s *ExpenseService) ListExpenses(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Expense, port.ListMeta, error) {
	page, limit = normalizePageLimit(page, limit)
	expenses, total, err := s.repo.ListByUserID(ctx, userID, dateFrom, dateTo, page, limit)
	if err != nil {
		return nil, port.ListMeta{}, fmt.Errorf("list expenses: %w", err)
	}
	if expenses == nil {
		expenses = []models.Expense{}
	}
	return expenses, listMeta(total, page, limit), nil
}

func (s *ExpenseService) UpdateExpense(ctx context.Context, userID int64, uuid string, req port.UpdateExpenseRequest) (*models.Expense, error) {
	if req.Amount != nil {
		if err := validateCashAmount(*req.Amount); err != nil {
			return nil, err
		}
	}
	if req.Date != nil {
		if err := validateTransactionDate(*req.Date); err != nil {
			return nil, err
		}
	}
	if err := validateNote(req.Note); err != nil {
		return nil, err
	}

	var category *models.Category
	if req.CategoryUUID != nil || req.Category != nil {
		var err error
		category, err = resolveCategory(ctx, s.categoryRepo, userID, models.CategoryTypeExpense, deref(req.CategoryUUID), deref(req.Category))
		if err != nil {
			return nil, err
		}
	}

	var expense *models.Expense
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		expense, err = s.repo.GetByUUIDForUpdate(ctx, uuid, userID)
		if err != nil {
			return fmt.Errorf("get expense: %w", err)
		}
		if expense == nil {
			return fmt.Errorf("expense: %w", port.ErrNotFound)
		}

		oldAmount, oldAssetID := expense.Amount, expense.AssetID
		newAssetID := oldAssetID
		if req.AssetUUID != nil {
			asset, err := lookupCashAsset(ctx, s.assetRepo, *req.AssetUUID, userID)
			if err != nil {
				return err
			}
			newAssetID = asset.ID
		}
		if req.Amount != nil {
			expense.Amount = *req.Amount
		}

		if newAssetID != oldAssetID {
			// Refund the old asset, then charge the new one.
			if err := adjustCashBalance(ctx, s.assetRepo, oldAssetID, oldAmount, expenseInsufficientMsg); err != nil {
				return err
			}
			if err := adjustCashBalance(ctx, s.assetRepo, newAssetID, expense.Amount.Neg(), expenseInsufficientMsg); err != nil {
				return err
			}
		} else if err := adjustCashBalance(ctx, s.assetRepo, oldAssetID, oldAmount.Sub(expense.Amount), expenseInsufficientMsg); err != nil {
			return err
		}

		expense.AssetID = newAssetID
		if category != nil {
			expense.CategoryID, expense.CategoryUUID, expense.CategoryName = category.ID, category.UUID, category.Name
		}
		if req.Note != nil {
			expense.Note = req.Note
		}
		if req.Date != nil {
			expense.Date = *req.Date
		}
		if err := s.repo.Update(ctx, expense); err != nil {
			return fmt.Errorf("update expense: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return expense, nil
}

func (s *ExpenseService) DeleteExpense(ctx context.Context, userID int64, uuid string) error {
	var expense *models.Expense
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		expense, err = s.repo.GetByUUIDForUpdate(ctx, uuid, userID)
		if err != nil {
			return fmt.Errorf("get expense: %w", err)
		}
		if expense == nil {
			return fmt.Errorf("expense: %w", port.ErrNotFound)
		}
		// Deleting an expense refunds its amount to the CASH asset.
		if err := adjustCashBalance(ctx, s.assetRepo, expense.AssetID, expense.Amount, expenseInsufficientMsg); err != nil {
			return err
		}
		if err := s.repo.Delete(ctx, uuid, userID); err != nil {
			return fmt.Errorf("delete expense: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("expense_deleted", "user_id", userID, "uuid", uuid, "amount", expense.Amount.String())
	return nil
}
