package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

// Cash direction of a receivable: the lent amount goes out of the linked CASH asset, and each
// repayment comes into the asset it is received in.
func receivableOpenDelta(amount decimal.Decimal) decimal.Decimal    { return amount.Neg() }
func receivablePaymentDelta(amount decimal.Decimal) decimal.Decimal { return amount }

const (
	receivableOpenInsufficientMsg    = "the cash asset balance is too low to lend this amount"
	receivablePaymentInsufficientMsg = "the cash asset balance is too low"
)

type ReceivableService struct {
	repo        port.ReceivableRepository
	paymentRepo port.ReceivablePaymentRepository
	assetRepo   port.AssetRepository
	tx          port.Transactor
}

func NewReceivableService(repo port.ReceivableRepository, paymentRepo port.ReceivablePaymentRepository, assetRepo port.AssetRepository, tx port.Transactor) port.ReceivableService {
	return &ReceivableService{repo: repo, paymentRepo: paymentRepo, assetRepo: assetRepo, tx: tx}
}

func (s *ReceivableService) CreateReceivable(ctx context.Context, userID int64, req port.CreateReceivableRequest) (*models.Receivable, error) {
	party, err := validatePartyName(req.PartyName)
	if err != nil {
		return nil, err
	}
	if err := validateCashAmount(req.Amount); err != nil {
		return nil, err
	}
	note, err := normalizeNote(req.Note)
	if err != nil {
		return nil, err
	}

	receivable := &models.Receivable{
		UserID:     userID,
		PartyName:  party,
		Amount:     req.Amount,
		PaidAmount: decimal.Zero,
		DueDate:    req.DueDate,
		Status:     models.ObligationStatusPending,
		Note:       note,
	}
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		asset, err := lookupOptionalCashAsset(ctx, s.assetRepo, req.AssetUUID, userID)
		if err != nil {
			return err
		}
		receivable.AssetID = assetIDOf(asset)
		deltas := cashDeltas{}
		deltas.add(receivable.AssetID, receivableOpenDelta(receivable.Amount))
		if err := deltas.apply(ctx, s.assetRepo, receivableOpenInsufficientMsg); err != nil {
			return err
		}
		return s.repo.Create(ctx, receivable)
	})
	if err != nil {
		return nil, err
	}
	slog.Info("receivable_created", "user_id", userID, "uuid", receivable.UUID, "amount", receivable.Amount.String())
	return s.GetReceivable(ctx, userID, receivable.UUID)
}

func (s *ReceivableService) GetReceivable(ctx context.Context, userID int64, uuid string) (*models.Receivable, error) {
	receivable, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get receivable: %w", err)
	}
	if receivable == nil {
		return nil, fmt.Errorf("receivable: %w", port.ErrNotFound)
	}
	return receivable, nil
}

func (s *ReceivableService) ListReceivables(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Receivable, port.ListMeta, error) {
	page, limit = normalizePageLimit(page, limit)
	receivables, total, err := s.repo.ListByUserID(ctx, userID, status, dueFrom, dueTo, page, limit)
	if err != nil {
		return nil, port.ListMeta{}, fmt.Errorf("list receivables: %w", err)
	}
	if receivables == nil {
		receivables = []models.Receivable{}
	}
	return receivables, listMeta(total, page, limit), nil
}

func (s *ReceivableService) UpdateReceivable(ctx context.Context, userID int64, uuid string, req port.UpdateReceivableRequest) (*models.Receivable, error) {
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		receivable, err := s.lockReceivable(ctx, uuid, userID)
		if err != nil {
			return err
		}
		deltas := cashDeltas{}
		deltas.add(receivable.AssetID, receivableOpenDelta(receivable.Amount).Neg())

		if req.PartyName != nil {
			if receivable.PartyName, err = validatePartyName(*req.PartyName); err != nil {
				return err
			}
		}
		if req.Amount != nil {
			if err := validateCashAmount(*req.Amount); err != nil {
				return err
			}
			if req.Amount.LessThan(receivable.PaidAmount) {
				return port.NewValidationError("amount cannot be less than the " + receivable.PaidAmount.StringFixed(2) + " already paid")
			}
			receivable.Amount = *req.Amount
		}
		if req.DueDate.Set {
			receivable.DueDate = req.DueDate.Value
		}
		if req.Note.Set {
			if receivable.Note, err = normalizeNote(req.Note.Value); err != nil {
				return err
			}
		}
		if req.AssetUUID.Set {
			asset, err := lookupOptionalCashAsset(ctx, s.assetRepo, req.AssetUUID.Value, userID)
			if err != nil {
				return err
			}
			receivable.AssetID = assetIDOf(asset)
		}

		deltas.add(receivable.AssetID, receivableOpenDelta(receivable.Amount))
		if err := deltas.apply(ctx, s.assetRepo, obligationReverseMsg); err != nil {
			return err
		}
		receivable.Status = models.ObligationStatusFor(receivable.Amount, receivable.PaidAmount)
		return s.repo.Update(ctx, receivable)
	})
	if err != nil {
		return nil, err
	}
	return s.GetReceivable(ctx, userID, uuid)
}

// DeleteReceivable removes the receivable with its payments and undoes their cash movements.
func (s *ReceivableService) DeleteReceivable(ctx context.Context, userID int64, uuid string) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		receivable, err := s.lockReceivable(ctx, uuid, userID)
		if err != nil {
			return err
		}
		payments, err := s.paymentRepo.ListByReceivableID(ctx, receivable.ID)
		if err != nil {
			return fmt.Errorf("list receivable payments: %w", err)
		}
		deltas := cashDeltas{}
		deltas.add(receivable.AssetID, receivableOpenDelta(receivable.Amount).Neg())
		for _, p := range payments {
			deltas.add(p.AssetID, receivablePaymentDelta(p.Amount).Neg())
		}
		if err := deltas.apply(ctx, s.assetRepo, obligationReverseMsg); err != nil {
			return err
		}
		return s.repo.Delete(ctx, uuid, userID)
	})
}

func (s *ReceivableService) RecordReceivablePayment(ctx context.Context, userID int64, receivableUUID string, req port.CreateReceivablePaymentRequest) (*models.ReceivablePayment, error) {
	if err := validateCashAmount(req.Amount); err != nil {
		return nil, err
	}
	if err := validateTransactionDate(req.Date); err != nil {
		return nil, err
	}
	note, err := normalizeNote(req.Note)
	if err != nil {
		return nil, err
	}

	var payment *models.ReceivablePayment
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		receivable, err := s.lockReceivable(ctx, receivableUUID, userID)
		if err != nil {
			return err
		}
		remaining := receivable.Amount.Sub(receivable.PaidAmount)
		if !remaining.IsPositive() {
			return port.NewValidationError("receivable is already fully collected")
		}
		if req.Amount.GreaterThan(remaining) {
			return port.NewValidationError("payment exceeds the remaining " + remaining.StringFixed(2))
		}
		asset, err := lookupOptionalCashAsset(ctx, s.assetRepo, req.AssetUUID, userID)
		if err != nil {
			return err
		}
		deltas := cashDeltas{}
		deltas.add(assetIDOf(asset), receivablePaymentDelta(req.Amount))
		if err := deltas.apply(ctx, s.assetRepo, receivablePaymentInsufficientMsg); err != nil {
			return err
		}

		payment = &models.ReceivablePayment{ReceivableID: receivable.ID, Amount: req.Amount, Date: req.Date, Note: note, AssetID: assetIDOf(asset)}
		if err := s.paymentRepo.Create(ctx, payment); err != nil {
			return err
		}
		payment.Asset = asset
		receivable.PaidAmount = receivable.PaidAmount.Add(req.Amount)
		receivable.Status = models.ObligationStatusFor(receivable.Amount, receivable.PaidAmount)
		return s.repo.Update(ctx, receivable)
	})
	if err != nil {
		return nil, err
	}
	slog.Info("receivable_payment_recorded", "user_id", userID, "receivable_uuid", receivableUUID, "amount", req.Amount.String())
	return payment, nil
}

func (s *ReceivableService) ListReceivablePayments(ctx context.Context, userID int64, receivableUUID string) ([]models.ReceivablePayment, error) {
	receivable, err := s.GetReceivable(ctx, userID, receivableUUID)
	if err != nil {
		return nil, err
	}
	payments, err := s.paymentRepo.ListByReceivableID(ctx, receivable.ID)
	if err != nil {
		return nil, fmt.Errorf("list receivable payments: %w", err)
	}
	if payments == nil {
		payments = []models.ReceivablePayment{}
	}
	return payments, nil
}

// DeleteReceivablePayment removes a payment recorded by mistake and undoes its cash movement.
func (s *ReceivableService) DeleteReceivablePayment(ctx context.Context, userID int64, receivableUUID, paymentUUID string) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		receivable, err := s.lockReceivable(ctx, receivableUUID, userID)
		if err != nil {
			return err
		}
		payment, err := s.paymentRepo.GetByUUID(ctx, paymentUUID, receivable.ID)
		if err != nil {
			return fmt.Errorf("get receivable payment: %w", err)
		}
		if payment == nil {
			return fmt.Errorf("receivable payment: %w", port.ErrNotFound)
		}
		deltas := cashDeltas{}
		deltas.add(payment.AssetID, receivablePaymentDelta(payment.Amount).Neg())
		if err := deltas.apply(ctx, s.assetRepo, obligationReverseMsg); err != nil {
			return err
		}
		if err := s.paymentRepo.Delete(ctx, payment.ID); err != nil {
			return err
		}
		receivable.PaidAmount = decimal.Max(receivable.PaidAmount.Sub(payment.Amount), decimal.Zero)
		receivable.Status = models.ObligationStatusFor(receivable.Amount, receivable.PaidAmount)
		return s.repo.Update(ctx, receivable)
	})
}

func (s *ReceivableService) lockReceivable(ctx context.Context, uuid string, userID int64) (*models.Receivable, error) {
	receivable, err := s.repo.GetByUUIDForUpdate(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get receivable: %w", err)
	}
	if receivable == nil {
		return nil, fmt.Errorf("receivable: %w", port.ErrNotFound)
	}
	return receivable, nil
}
