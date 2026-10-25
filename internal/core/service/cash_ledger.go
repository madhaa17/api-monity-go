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

const (
	maxSearchLen         = 100
	maxCategoryFilterLen = 50
)

// validateTransactionFilter rejects filters that cannot match anything or are unreasonably large.
func validateTransactionFilter(f port.TransactionFilter) error {
	if f.DateFrom != nil && f.DateTo != nil && f.DateFrom.After(*f.DateTo) {
		return port.NewValidationError("date_from must not be after date_to")
	}
	if f.MinAmount != nil && f.MaxAmount != nil && f.MinAmount.GreaterThan(*f.MaxAmount) {
		return port.NewValidationError("min_amount must not be greater than max_amount")
	}
	if len(f.CategoryUUIDs) > maxCategoryFilterLen {
		return port.NewValidationError(fmt.Sprintf("at most %d categories can be filtered at once", maxCategoryFilterLen))
	}
	if len([]rune(f.Search)) > maxSearchLen {
		return port.NewValidationError(fmt.Sprintf("search must be at most %d characters", maxSearchLen))
	}
	return nil
}

// parseTransactionSort parses "date", "-date", "amount" or "-amount"; empty means newest first.
func parseTransactionSort(sort string) (port.TransactionSort, error) {
	sort = strings.TrimSpace(sort)
	if sort == "" {
		return port.DefaultTransactionSort, nil
	}
	order := port.TransactionSort{Field: strings.TrimPrefix(sort, "-"), Desc: strings.HasPrefix(sort, "-")}
	if order.Field != "date" && order.Field != "amount" {
		return port.TransactionSort{}, port.NewValidationError("sort must be one of date, -date, amount, -amount")
	}
	return order, nil
}

func parseSummaryGroupBy(groupBy string) (port.SummaryGroupBy, error) {
	switch port.SummaryGroupBy(strings.ToLower(strings.TrimSpace(groupBy))) {
	case "", port.SummaryGroupByDay:
		return port.SummaryGroupByDay, nil
	case port.SummaryGroupByMonth:
		return port.SummaryGroupByMonth, nil
	}
	return "", port.NewValidationError("group_by must be day or month")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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
