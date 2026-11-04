package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ObligationStatusFor is the stored status of a debt or receivable given how much has been paid.
// OVERDUE is never stored: it is derived on read from the due date (see EffectiveObligationStatus).
func ObligationStatusFor(amount, paid decimal.Decimal) ObligationStatus {
	switch {
	case paid.GreaterThanOrEqual(amount):
		return ObligationStatusPaid
	case paid.IsPositive():
		return ObligationStatusPartial
	default:
		return ObligationStatusPending
	}
}

// StartOfDayUTC is the cut-off for overdue: an obligation due today is not yet overdue.
func StartOfDayUTC(now time.Time) time.Time {
	return now.UTC().Truncate(24 * time.Hour)
}

// EffectiveObligationStatus reports OVERDUE for an unpaid obligation whose due date has passed.
func EffectiveObligationStatus(stored ObligationStatus, due *time.Time, now time.Time) ObligationStatus {
	if stored != ObligationStatusPaid && due != nil && due.Before(StartOfDayUTC(now)) {
		return ObligationStatusOverdue
	}
	return stored
}
