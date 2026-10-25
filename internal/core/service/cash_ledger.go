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

// maxCashAmount is the exclusive upper bound that fits the DECIMAL(20,2) amount columns.
var maxCashAmount = decimal.New(1, 18)

// validateCashAmount checks that amount is positive, has at most 2 decimal places and fits the column.
func validateCashAmount(amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return port.NewValidationError("amount must be positive")
	}
	if !amount.Equal(amount.Truncate(2)) {
		return port.NewValidationError("amount must have at most 2 decimal places")
	}
	if amount.GreaterThanOrEqual(maxCashAmount) {
		return port.NewValidationError("amount is too large")
	}
	return nil
}

func validateTransactionDate(date time.Time) error {
	if date.IsZero() {
		return port.NewValidationError("date is required")
	}
	return nil
}

func validateNote(note *string) error {
	if note == nil {
		return nil
	}
	if err := validation.CheckMaxLen(*note, validation.MaxNoteLen); err != nil {
		return port.NewValidationError("note " + err.Error())
	}
	return nil
}

// lookupCashAsset returns the user's asset with the given UUID, which must be of type CASH.
func lookupCashAsset(ctx context.Context, assetRepo port.AssetRepository, assetUUID string, userID int64) (*models.Asset, error) {
	if strings.TrimSpace(assetUUID) == "" {
		return nil, port.NewValidationError("assetUuid is required")
	}
	asset, err := assetRepo.GetByUUID(ctx, assetUUID, userID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}
	if asset == nil {
		return nil, port.NewValidationError("asset not found")
	}
	if asset.Type != models.AssetTypeCash {
		return nil, port.NewValidationError("asset must be of type CASH")
	}
	return asset, nil
}

// adjustCashBalance adds delta to the asset balance, turning ErrInsufficientBalance into a
// validation error carrying insufficientMsg.
func adjustCashBalance(ctx context.Context, assetRepo port.AssetRepository, assetID int64, delta decimal.Decimal, insufficientMsg string) error {
	if delta.IsZero() {
		return nil
	}
	err := assetRepo.AdjustQuantity(ctx, assetID, delta)
	if errors.Is(err, port.ErrInsufficientBalance) {
		return port.NewValidationError(insufficientMsg)
	}
	if err != nil {
		return fmt.Errorf("adjust asset balance: %w", err)
	}
	return nil
}

func listMeta(total int64, page, limit int) port.ListMeta {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return port.ListMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
}

func normalizePageLimit(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
