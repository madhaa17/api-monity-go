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

type ReceivableService struct {
	repo       port.ReceivableRepository
	paymentRepo port.ReceivablePaymentRepository
	assetRepo  port.AssetRepository
}

func NewReceivableService(repo port.ReceivableRepository, paymentRepo port.ReceivablePaymentRepository, assetRepo port.AssetRepository) port.ReceivableService {
	return &ReceivableService{repo: repo, paymentRepo: paymentRepo, assetRepo: assetRepo}
}

func (s *ReceivableService) resolveAssetID(ctx context.Context, assetUUID *string, userID int64) (*int64, error) {
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

func (s *ReceivableService) CreateReceivable(ctx context.Context, userID int64, req port.CreateReceivableRequest) (*models.Receivable, error) {
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
	rec := &models.Receivable{
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
	if err := s.repo.Create(ctx, rec); err != nil {
		return nil, fmt.Errorf("create receivable: %w", err)
	}
	return rec, nil
}

func (s *ReceivableService) GetReceivable(ctx context.Context, userID int64, uuid string) (*models.Receivable, error) {
	rec, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get receivable: %w", err)
	}
	if rec == nil {
		return nil, errors.New("receivable not found")
	}
	return rec, nil
}

func (s *ReceivableService) ListReceivables(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Receivable, port.ListMeta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	recs, total, err := s.repo.ListByUserID(ctx, userID, status, dueFrom, dueTo, page, limit)
	if err != nil {
		return nil, port.ListMeta{}, fmt.Errorf("list receivables: %w", err)
	}
	if recs == nil {
		recs = []models.Receivable{}
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages < 0 {
		totalPages = 0
	}
	meta := port.ListMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
	return recs, meta, nil
}

func (s *ReceivableService) UpdateReceivable(ctx context.Context, userID int64, uuid string, req port.UpdateReceivableRequest) (*models.Receivable, error) {
	rec, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get receivable: %w", err)
	}
	if rec == nil {
		return nil, errors.New("receivable not found")
	}
	if req.PartyName != nil {
		if strings.TrimSpace(*req.PartyName) == "" {
			return nil, errors.New("party name cannot be empty")
		}
		rec.PartyName = strings.TrimSpace(*req.PartyName)
	}
	if req.Amount != nil {
		if *req.Amount <= 0 {
			return nil, errors.New("amount must be positive")
		}
		rec.Amount = decimal.NewFromFloat(*req.Amount)
	}
	if req.DueDate != nil {
		rec.DueDate = req.DueDate
	}
	if req.Note != nil {
		if err := validation.CheckMaxLen(*req.Note, validation.MaxNoteLen); err != nil {
			return nil, fmt.Errorf("note %w", err)
		}
		rec.Note = req.Note
	}
	if req.AssetUUID != nil {
		assetID, err := s.resolveAssetID(ctx, req.AssetUUID, userID)
		if err != nil {
			return nil, err
		}
		rec.AssetID = assetID
	}
	rec.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, rec); err != nil {
		return nil, fmt.Errorf("update receivable: %w", err)
	}
	return rec, nil
}

func (s *ReceivableService) DeleteReceivable(ctx context.Context, userID int64, uuid string) error {
	if err := s.repo.Delete(ctx, uuid, userID); err != nil {
		if err.Error() == "receivable not found or not owned by user" {
			return errors.New("receivable not found")
		}
		return fmt.Errorf("delete receivable: %w", err)
	}
	return nil
}

func (s *ReceivableService) RecordReceivablePayment(ctx context.Context, userID int64, receivableUUID string, req port.CreateReceivablePaymentRequest) (*models.ReceivablePayment, error) {
	if req.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	rec, err := s.repo.GetByUUID(ctx, receivableUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get receivable: %w", err)
	}
	if rec == nil {
		return nil, errors.New("receivable not found")
	}
	if rec.Status == models.ObligationStatusPaid {
		return nil, errors.New("receivable is already fully paid")
	}
	assetID, err := s.resolveAssetID(ctx, req.AssetUUID, userID)
	if err != nil {
		return nil, err
	}

	payment := &models.ReceivablePayment{
		ReceivableID: rec.ID,
		Amount:       decimal.NewFromFloat(req.Amount),
		Date:         req.Date,
		Note:         req.Note,
		AssetID:      assetID,
		CreatedAt:    time.Now(),
	}
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("create receivable payment: %w", err)
	}

	payments, err := s.paymentRepo.ListByReceivableID(ctx, rec.ID)
	if err != nil {
		return nil, fmt.Errorf("list receivable payments: %w", err)
	}
	var sum decimal.Decimal
	for _, p := range payments {
		sum = sum.Add(p.Amount)
	}
	if sum.GreaterThan(rec.Amount) {
		return nil, errors.New("total payments cannot exceed receivable amount")
	}
	rec.PaidAmount = sum
	if sum.GreaterThanOrEqual(rec.Amount) {
		rec.Status = models.ObligationStatusPaid
	} else {
		rec.Status = models.ObligationStatusPartial
	}
	rec.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, rec); err != nil {
		return nil, fmt.Errorf("update receivable after payment: %w", err)
	}
	return payment, nil
}

func (s *ReceivableService) ListReceivablePayments(ctx context.Context, userID int64, receivableUUID string) ([]models.ReceivablePayment, error) {
	rec, err := s.repo.GetByUUID(ctx, receivableUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get receivable: %w", err)
	}
	if rec == nil {
		return nil, errors.New("receivable not found")
	}
	payments, err := s.paymentRepo.ListByReceivableID(ctx, rec.ID)
	if err != nil {
		return nil, fmt.Errorf("list receivable payments: %w", err)
	}
	if payments == nil {
		return []models.ReceivablePayment{}, nil
	}
	return payments, nil
}
