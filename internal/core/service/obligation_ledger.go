package service

import (
	"context"
	"slices"
	"strings"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

	"github.com/shopspring/decimal"
)

// Reversing a debt or receivable (editing, deleting, removing a payment) moves money back, which
// may already be spent.
const obligationReverseMsg = "this change would make the cash asset balance negative"

// cashDeltas collects balance changes per CASH asset so they are applied once, as a net amount:
// moving money out and back into the same asset must not fail on an intermediate negative balance.
type cashDeltas map[int64]decimal.Decimal

func (d cashDeltas) add(assetID *int64, delta decimal.Decimal) {
	if assetID != nil {
		d[*assetID] = d[*assetID].Add(delta)
	}
}

// apply adjusts each asset in ID order, so concurrent requests lock assets in the same order.
func (d cashDeltas) apply(ctx context.Context, assetRepo port.AssetRepository, insufficientMsg string) error {
	ids := make([]int64, 0, len(d))
	for id := range d {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		if err := adjustCashBalance(ctx, assetRepo, id, d[id], insufficientMsg); err != nil {
			return err
		}
	}
	return nil
}

// lookupOptionalCashAsset is lookupCashAsset for an optional link: nil or blank means no asset.
func lookupOptionalCashAsset(ctx context.Context, assetRepo port.AssetRepository, assetUUID *string, userID int64) (*models.Asset, error) {
	if assetUUID == nil || strings.TrimSpace(*assetUUID) == "" {
		return nil, nil
	}
	return lookupCashAsset(ctx, assetRepo, strings.TrimSpace(*assetUUID), userID)
}

func assetIDOf(asset *models.Asset) *int64 {
	if asset == nil {
		return nil
	}
	return &asset.ID
}

func validatePartyName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", port.NewValidationError("party name is required")
	}
	if err := validation.CheckMaxLen(name, validation.MaxNameLen); err != nil {
		return "", port.NewValidationError("party name " + err.Error())
	}
	return name, nil
}

// normalizeNote validates a note and turns a blank one into nil.
func normalizeNote(note *string) (*string, error) {
	if note == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*note)
	if trimmed == "" {
		return nil, nil
	}
	if err := validateNote(&trimmed); err != nil {
		return nil, err
	}
	return &trimmed, nil
}
