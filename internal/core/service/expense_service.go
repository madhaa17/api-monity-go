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

type ExpenseService struct {
	repo      port.ExpenseRepository
	assetRepo port.AssetRepository
}

func NewExpenseService(repo port.ExpenseRepository, assetRepo port.AssetRepository) port.ExpenseService {
	return &ExpenseService{repo: repo, assetRepo: assetRepo}
}

// lookupCashAsset validates that the asset exists, belongs to the user, and is type CASH.
func (s *ExpenseService) lookupCashAsset(ctx context.Context, assetUUID string, userID int64) (*models.Asset, error) {
	if strings.TrimSpace(assetUUID) == "" {
		return nil, errors.New("assetUuid is required")
	}
	asset, err := s.assetRepo.GetByUUID(ctx, assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, errors.New("asset not found")
	}
	if asset.Type != models.AssetTypeCash {
		return nil, errors.New("asset must be of type CASH")
	}
	return asset, nil
}

func (s *ExpenseService) CreateExpense(ctx context.Context, userID int64, req port.CreateExpenseRequest) (*models.Expense, error) {
	if req.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if !isValidExpenseCategory(req.Category) {
		return nil, errors.New("invalid expense category")
	}

	asset, err := s.lookupCashAsset(ctx, req.AssetUUID, userID)
	if err != nil {
		return nil, err
	}

	amount := decimal.NewFromFloat(req.Amount)

	// Deduct from CASH asset
	asset.Quantity = asset.Quantity.Sub(amount)
	if err := s.assetRepo.Update(ctx, asset); err != nil {
		return nil, fmt.Errorf("update asset balance: %w", err)
	}

	expense := &models.Expense{
		UserID:   userID,
		AssetID:  asset.ID,
		Amount:   amount,
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

	oldAmount := expense.Amount
	oldAssetID := expense.AssetID

	// Update fields
	if req.Amount != nil {
		if *req.Amount <= 0 {
			return nil, errors.New("amount must be positive")
		}
		expense.Amount = decimal.NewFromFloat(*req.Amount)
	}
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

	// Adjust CASH asset balances
	if req.AssetUUID != nil {
		newAsset, err := s.lookupCashAsset(ctx, *req.AssetUUID, userID)
		if err != nil {
			return nil, err
		}

		if newAsset.ID != oldAssetID {
			// Restore old asset: add back old amount
			oldAsset, err := s.assetRepo.GetByID(ctx, oldAssetID)
			if err != nil {
				return nil, fmt.Errorf("get old asset: %w", err)
			}
			if oldAsset != nil {
				oldAsset.Quantity = oldAsset.Quantity.Add(oldAmount)
				if err := s.assetRepo.Update(ctx, oldAsset); err != nil {
					return nil, fmt.Errorf("restore old asset balance: %w", err)
				}
			}

			// Deduct new amount from new asset
			newAsset.Quantity = newAsset.Quantity.Sub(expense.Amount)
			if err := s.assetRepo.Update(ctx, newAsset); err != nil {
				return nil, fmt.Errorf("update new asset balance: %w", err)
			}
			expense.AssetID = newAsset.ID
		} else {
			// Same asset, adjust difference
			diff := expense.Amount.Sub(oldAmount)
			if !diff.IsZero() {
				newAsset.Quantity = newAsset.Quantity.Sub(diff)
				if err := s.assetRepo.Update(ctx, newAsset); err != nil {
					return nil, fmt.Errorf("update asset balance: %w", err)
				}
			}
		}
	} else if req.Amount != nil {
		// Only amount changed, same asset
		diff := expense.Amount.Sub(oldAmount)
		if !diff.IsZero() {
			asset, err := s.assetRepo.GetByID(ctx, oldAssetID)
			if err != nil {
				return nil, fmt.Errorf("get asset: %w", err)
			}
			if asset != nil {
				asset.Quantity = asset.Quantity.Sub(diff)
				if err := s.assetRepo.Update(ctx, asset); err != nil {
					return nil, fmt.Errorf("update asset balance: %w", err)
				}
			}
		}
	}

	if err := s.repo.Update(ctx, expense); err != nil {
		return nil, fmt.Errorf("update expense: %w", err)
	}
	return expense, nil
}

func (s *ExpenseService) DeleteExpense(ctx context.Context, userID int64, uuid string) error {
	expense, err := s.GetExpense(ctx, userID, uuid)
	if err != nil {
		return err
	}

	// Restore CASH asset balance
	asset, err := s.assetRepo.GetByID(ctx, expense.AssetID)
	if err != nil {
		return fmt.Errorf("get asset: %w", err)
	}
	if asset != nil {
		asset.Quantity = asset.Quantity.Add(expense.Amount)
		if err := s.assetRepo.Update(ctx, asset); err != nil {
			return fmt.Errorf("restore asset balance: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, uuid, userID); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

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
