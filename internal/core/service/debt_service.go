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

// Cash direction of a debt: the borrowed amount comes into the linked CASH asset, and each
// repayment goes out of the asset it is paid from.
func debtOpenDelta(amount decimal.Decimal) decimal.Decimal    { return amount }
func debtPaymentDelta(amount decimal.Decimal) decimal.Decimal { return amount.Neg() }

const (
	debtOpenInsufficientMsg    = "the cash asset balance is too low"
	debtPaymentInsufficientMsg = "the cash asset balance is too low for this payment"
)

type DebtService struct {
	repo        port.DebtRepository
	paymentRepo port.DebtPaymentRepository
	assetRepo   port.AssetRepository
	tx          port.Transactor
}

func NewDebtService(repo port.DebtRepository, paymentRepo port.DebtPaymentRepository, assetRepo port.AssetRepository, tx port.Transactor) port.DebtService {
	return &DebtService{repo: repo, paymentRepo: paymentRepo, assetRepo: assetRepo, tx: tx}
}

func (s *DebtService) CreateDebt(ctx context.Context, userID int64, req port.CreateDebtRequest) (*models.Debt, error) {
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

	debt := &models.Debt{
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
		debt.AssetID = assetIDOf(asset)
		deltas := cashDeltas{}
		deltas.add(debt.AssetID, debtOpenDelta(debt.Amount))
		if err := deltas.apply(ctx, s.assetRepo, debtOpenInsufficientMsg); err != nil {
			return err
		}
		return s.repo.Create(ctx, debt)
	})
	if err != nil {
		return nil, err
	}
	slog.Info("debt_created", "user_id", userID, "uuid", debt.UUID, "amount", debt.Amount.String())
	return s.GetDebt(ctx, userID, debt.UUID)
}

func (s *DebtService) GetDebt(ctx context.Context, userID int64, uuid string) (*models.Debt, error) {
	debt, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get debt: %w", err)
	}
	if debt == nil {
		return nil, fmt.Errorf("debt: %w", port.ErrNotFound)
	}
	return debt, nil
}

func (s *DebtService) ListDebts(ctx context.Context, userID int64, status *string, dueFrom, dueTo *time.Time, page, limit int) ([]models.Debt, port.ListMeta, error) {
	page, limit = normalizePageLimit(page, limit)
	debts, total, err := s.repo.ListByUserID(ctx, userID, status, dueFrom, dueTo, page, limit)
	if err != nil {
		return nil, port.ListMeta{}, fmt.Errorf("list debts: %w", err)
	}
	if debts == nil {
		debts = []models.Debt{}
	}
	return debts, listMeta(total, page, limit), nil
}

func (s *DebtService) UpdateDebt(ctx context.Context, userID int64, uuid string, req port.UpdateDebtRequest) (*models.Debt, error) {
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		debt, err := s.lockDebt(ctx, uuid, userID)
		if err != nil {
			return err
		}
		deltas := cashDeltas{}
		deltas.add(debt.AssetID, debtOpenDelta(debt.Amount).Neg())

		if req.PartyName != nil {
			if debt.PartyName, err = validatePartyName(*req.PartyName); err != nil {
				return err
			}
		}
		if req.Amount != nil {
			if err := validateCashAmount(*req.Amount); err != nil {
				return err
			}
			if req.Amount.LessThan(debt.PaidAmount) {
				return port.NewValidationError("amount cannot be less than the " + debt.PaidAmount.StringFixed(2) + " already paid")
			}
			debt.Amount = *req.Amount
		}
		if req.DueDate.Set {
			debt.DueDate = req.DueDate.Value
		}
		if req.Note.Set {
			if debt.Note, err = normalizeNote(req.Note.Value); err != nil {
				return err
			}
		}
		if req.AssetUUID.Set {
			asset, err := lookupOptionalCashAsset(ctx, s.assetRepo, req.AssetUUID.Value, userID)
			if err != nil {
				return err
			}
			debt.AssetID = assetIDOf(asset)
		}

		deltas.add(debt.AssetID, debtOpenDelta(debt.Amount))
		if err := deltas.apply(ctx, s.assetRepo, obligationReverseMsg); err != nil {
			return err
		}
		debt.Status = models.ObligationStatusFor(debt.Amount, debt.PaidAmount)
		return s.repo.Update(ctx, debt)
	})
	if err != nil {
		return nil, err
	}
	return s.GetDebt(ctx, userID, uuid)
}

// DeleteDebt removes the debt with its payments and undoes their cash movements.
func (s *DebtService) DeleteDebt(ctx context.Context, userID int64, uuid string) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		debt, err := s.lockDebt(ctx, uuid, userID)
		if err != nil {
			return err
		}
		payments, err := s.paymentRepo.ListByDebtID(ctx, debt.ID)
		if err != nil {
			return fmt.Errorf("list debt payments: %w", err)
		}
		deltas := cashDeltas{}
		deltas.add(debt.AssetID, debtOpenDelta(debt.Amount).Neg())
		for _, p := range payments {
			deltas.add(p.AssetID, debtPaymentDelta(p.Amount).Neg())
		}
		if err := deltas.apply(ctx, s.assetRepo, obligationReverseMsg); err != nil {
			return err
		}
		return s.repo.Delete(ctx, uuid, userID)
	})
}

func (s *DebtService) RecordDebtPayment(ctx context.Context, userID int64, debtUUID string, req port.CreateDebtPaymentRequest) (*models.DebtPayment, error) {
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

	var payment *models.DebtPayment
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		debt, err := s.lockDebt(ctx, debtUUID, userID)
		if err != nil {
			return err
		}
		remaining := debt.Amount.Sub(debt.PaidAmount)
		if !remaining.IsPositive() {
			return port.NewValidationError("debt is already fully paid")
		}
		if req.Amount.GreaterThan(remaining) {
			return port.NewValidationError("payment exceeds the remaining " + remaining.StringFixed(2))
		}
		asset, err := lookupOptionalCashAsset(ctx, s.assetRepo, req.AssetUUID, userID)
		if err != nil {
			return err
		}
		deltas := cashDeltas{}
		deltas.add(assetIDOf(asset), debtPaymentDelta(req.Amount))
		if err := deltas.apply(ctx, s.assetRepo, debtPaymentInsufficientMsg); err != nil {
			return err
		}

		payment = &models.DebtPayment{DebtID: debt.ID, Amount: req.Amount, Date: req.Date, Note: note, AssetID: assetIDOf(asset)}
		if err := s.paymentRepo.Create(ctx, payment); err != nil {
			return err
		}
		payment.Asset = asset
		debt.PaidAmount = debt.PaidAmount.Add(req.Amount)
		debt.Status = models.ObligationStatusFor(debt.Amount, debt.PaidAmount)
		return s.repo.Update(ctx, debt)
	})
	if err != nil {
		return nil, err
	}
	slog.Info("debt_payment_recorded", "user_id", userID, "debt_uuid", debtUUID, "amount", req.Amount.String())
	return payment, nil
}

func (s *DebtService) ListDebtPayments(ctx context.Context, userID int64, debtUUID string) ([]models.DebtPayment, error) {
	debt, err := s.GetDebt(ctx, userID, debtUUID)
	if err != nil {
		return nil, err
	}
	payments, err := s.paymentRepo.ListByDebtID(ctx, debt.ID)
	if err != nil {
		return nil, fmt.Errorf("list debt payments: %w", err)
	}
	if payments == nil {
		payments = []models.DebtPayment{}
	}
	return payments, nil
}

// DeleteDebtPayment removes a payment recorded by mistake and undoes its cash movement.
func (s *DebtService) DeleteDebtPayment(ctx context.Context, userID int64, debtUUID, paymentUUID string) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		debt, err := s.lockDebt(ctx, debtUUID, userID)
		if err != nil {
			return err
		}
		payment, err := s.paymentRepo.GetByUUID(ctx, paymentUUID, debt.ID)
		if err != nil {
			return fmt.Errorf("get debt payment: %w", err)
		}
		if payment == nil {
			return fmt.Errorf("debt payment: %w", port.ErrNotFound)
		}
		deltas := cashDeltas{}
		deltas.add(payment.AssetID, debtPaymentDelta(payment.Amount).Neg())
		if err := deltas.apply(ctx, s.assetRepo, obligationReverseMsg); err != nil {
			return err
		}
		if err := s.paymentRepo.Delete(ctx, payment.ID); err != nil {
			return err
		}
		debt.PaidAmount = decimal.Max(debt.PaidAmount.Sub(payment.Amount), decimal.Zero)
		debt.Status = models.ObligationStatusFor(debt.Amount, debt.PaidAmount)
		return s.repo.Update(ctx, debt)
	})
}

func (s *DebtService) lockDebt(ctx context.Context, uuid string, userID int64) (*models.Debt, error) {
	debt, err := s.repo.GetByUUIDForUpdate(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get debt: %w", err)
	}
	if debt == nil {
		return nil, fmt.Errorf("debt: %w", port.ErrNotFound)
	}
	return debt, nil
}
