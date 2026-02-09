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
	repo      port.IncomeRepository
	assetRepo port.AssetRepository
}

func NewIncomeService(repo port.IncomeRepository, assetRepo port.AssetRepository) port.IncomeService {
	return &IncomeService{repo: repo, assetRepo: assetRepo}
}

// lookupCashAsset validates that the asset exists, belongs to the user, and is type CASH.
func (s *IncomeService) lookupCashAsset(ctx context.Context, assetUUID string, userID int64) (*models.Asset, error) {
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

func (s *IncomeService) CreateIncome(ctx context.Context, userID int64, req port.CreateIncomeRequest) (*models.Income, error) {
	if req.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if strings.TrimSpace(req.Source) == "" {
		return nil, errors.New("source is required")
	}

	asset, err := s.lookupCashAsset(ctx, req.AssetUUID, userID)
	if err != nil {
		return nil, err
	}

	amount := decimal.NewFromFloat(req.Amount)

	// Add to CASH asset
	asset.Quantity = asset.Quantity.Add(amount)
	if err := s.assetRepo.Update(ctx, asset); err != nil {
		return nil, fmt.Errorf("update asset balance: %w", err)
	}

	income := &models.Income{
		UserID:  userID,
		AssetID: asset.ID,
		Amount:  amount,
		Source:  req.Source,
		Note:    req.Note,
		Date:    req.Date,
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

	oldAmount := income.Amount
	oldAssetID := income.AssetID

	// Update fields
	if req.Amount != nil {
		if *req.Amount <= 0 {
			return nil, errors.New("amount must be positive")
		}
		income.Amount = decimal.NewFromFloat(*req.Amount)
	}
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

	// Adjust CASH asset balances
	if req.AssetUUID != nil {
		newAsset, err := s.lookupCashAsset(ctx, *req.AssetUUID, userID)
		if err != nil {
			return nil, err
		}

		if newAsset.ID != oldAssetID {
			// Reverse old asset: subtract old amount
			oldAsset, err := s.assetRepo.GetByID(ctx, oldAssetID)
			if err != nil {
				return nil, fmt.Errorf("get old asset: %w", err)
			}
			if oldAsset != nil {
				oldAsset.Quantity = oldAsset.Quantity.Sub(oldAmount)
				if err := s.assetRepo.Update(ctx, oldAsset); err != nil {
					return nil, fmt.Errorf("reverse old asset balance: %w", err)
				}
			}

			// Add new amount to new asset
			newAsset.Quantity = newAsset.Quantity.Add(income.Amount)
			if err := s.assetRepo.Update(ctx, newAsset); err != nil {
				return nil, fmt.Errorf("update new asset balance: %w", err)
			}
			income.AssetID = newAsset.ID
		} else {
			// Same asset, adjust difference
			diff := income.Amount.Sub(oldAmount) // positive = more income
			if !diff.IsZero() {
				newAsset.Quantity = newAsset.Quantity.Add(diff)
				if err := s.assetRepo.Update(ctx, newAsset); err != nil {
					return nil, fmt.Errorf("update asset balance: %w", err)
				}
			}
		}
	} else if req.Amount != nil {
		// Only amount changed, same asset
		diff := income.Amount.Sub(oldAmount)
		if !diff.IsZero() {
			asset, err := s.assetRepo.GetByID(ctx, oldAssetID)
			if err != nil {
				return nil, fmt.Errorf("get asset: %w", err)
			}
			if asset != nil {
				asset.Quantity = asset.Quantity.Add(diff)
				if err := s.assetRepo.Update(ctx, asset); err != nil {
					return nil, fmt.Errorf("update asset balance: %w", err)
				}
			}
		}
	}

	if err := s.repo.Update(ctx, income); err != nil {
		return nil, fmt.Errorf("update income: %w", err)
	}
	return income, nil
}

func (s *IncomeService) DeleteIncome(ctx context.Context, userID int64, uuid string) error {
	income, err := s.GetIncome(ctx, userID, uuid)
	if err != nil {
		return err
	}

	// Reverse CASH asset balance
	asset, err := s.assetRepo.GetByID(ctx, income.AssetID)
	if err != nil {
		return fmt.Errorf("get asset: %w", err)
	}
	if asset != nil {
		asset.Quantity = asset.Quantity.Sub(income.Amount)
		if err := s.assetRepo.Update(ctx, asset); err != nil {
			return fmt.Errorf("reverse asset balance: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, uuid, userID); err != nil {
		return fmt.Errorf("delete income: %w", err)
	}
	return nil
}
