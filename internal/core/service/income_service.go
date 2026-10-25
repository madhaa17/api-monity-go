package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"
)

// Removing or shrinking an income takes money back out of the CASH asset, which may already be spent.
const incomeInsufficientMsg = "this change would make the cash asset balance negative"

type IncomeService struct {
	repo         port.IncomeRepository
	assetRepo    port.AssetRepository
	categoryRepo port.CategoryRepository
	tx           port.Transactor
}

func NewIncomeService(repo port.IncomeRepository, assetRepo port.AssetRepository, categoryRepo port.CategoryRepository, tx port.Transactor) port.IncomeService {
	return &IncomeService{repo: repo, assetRepo: assetRepo, categoryRepo: categoryRepo, tx: tx}
}

func (s *IncomeService) CreateIncome(ctx context.Context, userID int64, req port.CreateIncomeRequest) (*models.Income, error) {
	if err := validateCashAmount(req.Amount); err != nil {
		return nil, err
	}
	if err := validateIncomeSource(req.Source); err != nil {
		return nil, err
	}
	if err := validateTransactionDate(req.Date); err != nil {
		return nil, err
	}
	if err := validateNote(req.Note); err != nil {
		return nil, err
	}

	// Incomes predate categories, so an omitted category falls back to "Other".
	categoryName := ""
	if strings.TrimSpace(req.CategoryUUID) == "" {
		categoryName = models.CategoryOther
	}
	category, err := resolveCategory(ctx, s.categoryRepo, userID, models.CategoryTypeIncome, req.CategoryUUID, categoryName)
	if err != nil {
		return nil, err
	}

	var income *models.Income
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		asset, err := lookupCashAsset(ctx, s.assetRepo, req.AssetUUID, userID)
		if err != nil {
			return err
		}
		if err := adjustCashBalance(ctx, s.assetRepo, asset.ID, req.Amount, incomeInsufficientMsg); err != nil {
			return err
		}
		income = &models.Income{
			UserID:       userID,
			AssetID:      asset.ID,
			Amount:       req.Amount,
			Source:       req.Source,
			CategoryID:   category.ID,
			CategoryUUID: category.UUID,
			CategoryName: category.Name,
			Note:         req.Note,
			Date:         req.Date,
		}
		if err := s.repo.Create(ctx, income); err != nil {
			return fmt.Errorf("create income: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slog.Info("income_created", "user_id", userID, "amount", req.Amount.String(), "source", req.Source, "asset_uuid", req.AssetUUID)
	return income, nil
}

func (s *IncomeService) GetIncome(ctx context.Context, userID int64, uuid string) (*models.Income, error) {
	income, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get income: %w", err)
	}
	if income == nil {
		return nil, fmt.Errorf("income: %w", port.ErrNotFound)
	}
	return income, nil
}

func (s *IncomeService) ListIncomes(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Income, port.ListMeta, error) {
	page, limit = normalizePageLimit(page, limit)
	incomes, total, err := s.repo.ListByUserID(ctx, userID, dateFrom, dateTo, page, limit)
	if err != nil {
		return nil, port.ListMeta{}, fmt.Errorf("list incomes: %w", err)
	}
	if incomes == nil {
		incomes = []models.Income{}
	}
	return incomes, listMeta(total, page, limit), nil
}

func (s *IncomeService) UpdateIncome(ctx context.Context, userID int64, uuid string, req port.UpdateIncomeRequest) (*models.Income, error) {
	if req.Amount != nil {
		if err := validateCashAmount(*req.Amount); err != nil {
			return nil, err
		}
	}
	if req.Source != nil {
		if err := validateIncomeSource(*req.Source); err != nil {
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
	if req.CategoryUUID != nil {
		var err error
		category, err = resolveCategory(ctx, s.categoryRepo, userID, models.CategoryTypeIncome, *req.CategoryUUID, "")
		if err != nil {
			return nil, err
		}
	}

	var income *models.Income
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		income, err = s.repo.GetByUUIDForUpdate(ctx, uuid, userID)
		if err != nil {
			return fmt.Errorf("get income: %w", err)
		}
		if income == nil {
			return fmt.Errorf("income: %w", port.ErrNotFound)
		}

		oldAmount, oldAssetID := income.Amount, income.AssetID
		newAssetID := oldAssetID
		if req.AssetUUID != nil {
			asset, err := lookupCashAsset(ctx, s.assetRepo, *req.AssetUUID, userID)
			if err != nil {
				return err
			}
			newAssetID = asset.ID
		}
		if req.Amount != nil {
			income.Amount = *req.Amount
		}

		if newAssetID != oldAssetID {
			// Take the income back out of the old asset, then put it into the new one.
			if err := adjustCashBalance(ctx, s.assetRepo, oldAssetID, oldAmount.Neg(), incomeInsufficientMsg); err != nil {
				return err
			}
			if err := adjustCashBalance(ctx, s.assetRepo, newAssetID, income.Amount, incomeInsufficientMsg); err != nil {
				return err
			}
		} else if err := adjustCashBalance(ctx, s.assetRepo, oldAssetID, income.Amount.Sub(oldAmount), incomeInsufficientMsg); err != nil {
			return err
		}

		income.AssetID = newAssetID
		if req.Source != nil {
			income.Source = *req.Source
		}
		if category != nil {
			income.CategoryID, income.CategoryUUID, income.CategoryName = category.ID, category.UUID, category.Name
		}
		if req.Note != nil {
			income.Note = req.Note
		}
		if req.Date != nil {
			income.Date = *req.Date
		}
		if err := s.repo.Update(ctx, income); err != nil {
			return fmt.Errorf("update income: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return income, nil
}

func (s *IncomeService) DeleteIncome(ctx context.Context, userID int64, uuid string) error {
	var income *models.Income
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		income, err = s.repo.GetByUUIDForUpdate(ctx, uuid, userID)
		if err != nil {
			return fmt.Errorf("get income: %w", err)
		}
		if income == nil {
			return fmt.Errorf("income: %w", port.ErrNotFound)
		}
		if err := adjustCashBalance(ctx, s.assetRepo, income.AssetID, income.Amount.Neg(), incomeInsufficientMsg); err != nil {
			return err
		}
		if err := s.repo.Delete(ctx, uuid, userID); err != nil {
			return fmt.Errorf("delete income: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("income_deleted", "user_id", userID, "uuid", uuid, "amount", income.Amount.String())
	return nil
}

func validateIncomeSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return port.NewValidationError("source is required")
	}
	if err := validation.CheckMaxLen(source, validation.MaxSourceLen); err != nil {
		return port.NewValidationError("source " + err.Error())
	}
	return nil
}
