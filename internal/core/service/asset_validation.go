package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"
)

// maxAssetNumber is the exclusive bound for the DECIMAL(20,8) asset columns (12 integer digits).
const maxAssetNumber = 1e12

// parseAssetDate accepts RFC 3339 ("2026-01-01T00:00:00Z") or a plain date ("2026-01-01", UTC).
func parseAssetDate(field, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.DateOnly, value); err == nil {
		return t, nil
	}
	return time.Time{}, port.NewValidationError(field + " must be a date like 2026-01-31 or 2026-01-31T00:00:00Z")
}

// parseOptionalAssetDate is parseAssetDate where an empty string clears the date.
func parseOptionalAssetDate(field, value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	t, err := parseAssetDate(field, value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func validateAssetType(t models.AssetType) error {
	switch t {
	case models.AssetTypeCrypto, models.AssetTypeStock, models.AssetTypeCash,
		models.AssetTypeRealEstate, models.AssetTypeLivestock, models.AssetTypeOther:
		return nil
	}
	return port.NewValidationError("type must be one of CRYPTO, STOCK, CASH, REAL_ESTATE, LIVESTOCK, OTHER")
}

func validateAssetStatus(s models.AssetStatus) error {
	switch s {
	case models.AssetStatusActive, models.AssetStatusSold, models.AssetStatusPlanned:
		return nil
	}
	return port.NewValidationError("status must be one of ACTIVE, SOLD, PLANNED")
}

// validateAssetNumber rejects negative values and values that do not fit the column.
func validateAssetNumber(field string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) >= maxAssetNumber {
		return port.NewValidationError(field + " is too large")
	}
	if v < 0 {
		return port.NewValidationError(field + " cannot be negative")
	}
	return nil
}

func validateOptionalAssetNumber(field string, v *float64) error {
	if v == nil {
		return nil
	}
	return validateAssetNumber(field, *v)
}

func validateAssetText(field string, v *string, max int) error {
	if v == nil {
		return nil
	}
	if err := validation.CheckMaxLen(*v, max); err != nil {
		return port.NewValidationError(fmt.Sprintf("%s %s", field, err.Error()))
	}
	return nil
}
