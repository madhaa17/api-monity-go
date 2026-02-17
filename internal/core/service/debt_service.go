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

type DebtService struct {
	repo      port.DebtRepository
	paymentRepo port.DebtPaymentRepository
	assetRepo port.AssetRepository
}

func NewDebtService(repo port.DebtRepository, paymentRepo port.DebtPaymentRepository, assetRepo port.AssetRepository) port.DebtService {
	return &DebtService{repo: repo, paymentRepo: paymentRepo, assetRepo: assetRepo}
}

func (s *DebtService) resolveAssetID(ctx context.Context, assetUUID *string, userID int64) (*int64, error) {
	if assetUUID == nil || strings.TrimSpace(*assetUUID) == "" {
		return nil, nil
	}
	asset, err := s.assetRepo.GetByUUID(ctx, *assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, errors.New("asset not found")
	}
	if asset.Type != models.AssetTypeCash {
		return nil, errors.New("asset must be of type CASH")
	}
	return &asset.ID, nil
}

func (s *DebtService) CreateDebt(ctx context.Context, userID int64, req port.CreateDebtRequest) (*models.Debt, error) {
	if strings.TrimSpace(req.PartyName) == "" {
		return nil, errors.New("party name is required")
	}
	if req.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if req.Note != nil {
		if err := validation.CheckMaxLen(*req.Note, validation.MaxNoteLen); err != nil {
			return nil, fmt.Errorf("note %w", err)
		}
	}
	assetID, err := s.resolveAssetID(ctx, req.AssetUUID, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	debt := &models.Debt{
		UserID:     userID,
		PartyName:  strings.TrimSpace(req.PartyName),
		Amount:     decimal.NewFromFloat(req.Amount),
		PaidAmount: decimal.Zero,
		DueDate:    req.DueDate,
		Status:     models.ObligationStatusPending,
		Note:       req.Note,
		AssetID:    assetID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.Create(ctx, debt); err != nil {
		return nil, fmt.Errorf("create debt: %w", err)
	}
	return debt, nil
}

func (s *DebtService) GetDebt(ctx context.Context, userID int64, uuid string) (*models.Debt, error) {
	debt, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get debt: %w", err)
	}
	if debt == nil {
		return nil, errors.New("debt not found")
	}
	return debt, nil
}

func (s *DebtService) ListDebts(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Debt, port.ListMeta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	debts, total, err := s.repo.ListByUserID(ctx, userID, status, dueFrom, dueTo, page, limit)
	if err != nil {
		return nil, port.ListMeta{}, fmt.Errorf("list debts: %w", err)
	}
	if debts == nil {
		debts = []models.Debt{}
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages < 0 {
		totalPages = 0
	}
	meta := port.ListMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
	return debts, meta, nil
}

func (s *DebtService) UpdateDebt(ctx context.Context, userID int64, uuid string, req port.UpdateDebtRequest) (*models.Debt, error) {
	debt, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get debt: %w", err)
	}
	if debt == nil {
		return nil, errors.New("debt not found")
	}
	if req.PartyName != nil {
		if strings.TrimSpace(*req.PartyName) == "" {
			return nil, errors.New("party name cannot be empty")
		}
		debt.PartyName = strings.TrimSpace(*req.PartyName)
	}
	if req.Amount != nil {
		if *req.Amount <= 0 {
			return nil, errors.New("amount must be positive")
		}
		debt.Amount = decimal.NewFromFloat(*req.Amount)
	}
	if req.DueDate != nil {
		debt.DueDate = req.DueDate
	}
	if req.Note != nil {
		if err := validation.CheckMaxLen(*req.Note, validation.MaxNoteLen); err != nil {
			return nil, fmt.Errorf("note %w", err)
		}
		debt.Note = req.Note
	}
	if req.AssetUUID != nil {
		assetID, err := s.resolveAssetID(ctx, req.AssetUUID, userID)
		if err != nil {
			return nil, err
		}
		debt.AssetID = assetID
	}
	debt.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, debt); err != nil {
		return nil, fmt.Errorf("update debt: %w", err)
	}
	return debt, nil
}

func (s *DebtService) DeleteDebt(ctx context.Context, userID int64, uuid string) error {
	if err := s.repo.Delete(ctx, uuid, userID); err != nil {
		if err.Error() == "debt not found or not owned by user" {
			return errors.New("debt not found")
		}
		return fmt.Errorf("delete debt: %w", err)
	}
	return nil
}

func (s *DebtService) RecordDebtPayment(ctx context.Context, userID int64, debtUUID string, req port.CreateDebtPaymentRequest) (*models.DebtPayment, error) {
	if req.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	debt, err := s.repo.GetByUUID(ctx, debtUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get debt: %w", err)
	}
	if debt == nil {
		return nil, errors.New("debt not found")
	}
	if debt.Status == models.ObligationStatusPaid {
		return nil, errors.New("debt is already fully paid")
	}
	assetID, err := s.resolveAssetID(ctx, req.AssetUUID, userID)
	if err != nil {
		return nil, err
	}

	payment := &models.DebtPayment{
		DebtID:   debt.ID,
		Amount:   decimal.NewFromFloat(req.Amount),
		Date:     req.Date,
		Note:     req.Note,
		AssetID:  assetID,
		CreatedAt: time.Now(),
	}
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("create debt payment: %w", err)
	}

	payments, err := s.paymentRepo.ListByDebtID(ctx, debt.ID)
	if err != nil {
		return nil, fmt.Errorf("list debt payments: %w", err)
	}
	var sum decimal.Decimal
	for _, p := range payments {
		sum = sum.Add(p.Amount)
	}
	if sum.GreaterThan(debt.Amount) {
		return nil, errors.New("total payments cannot exceed debt amount")
	}
	debt.PaidAmount = sum
	if sum.GreaterThanOrEqual(debt.Amount) {
		debt.Status = models.ObligationStatusPaid
	} else {
		debt.Status = models.ObligationStatusPartial
	}
	debt.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, debt); err != nil {
		return nil, fmt.Errorf("update debt after payment: %w", err)
	}
	return payment, nil
}

func (s *DebtService) ListDebtPayments(ctx context.Context, userID int64, debtUUID string) ([]models.DebtPayment, error) {
	debt, err := s.repo.GetByUUID(ctx, debtUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get debt: %w", err)
	}
	if debt == nil {
		return nil, errors.New("debt not found")
	}
	payments, err := s.paymentRepo.ListByDebtID(ctx, debt.ID)
	if err != nil {
		return nil, fmt.Errorf("list debt payments: %w", err)
	}
	if payments == nil {
		return []models.DebtPayment{}, nil
	}
	return payments, nil
}
