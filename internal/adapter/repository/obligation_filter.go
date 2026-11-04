package repository

import (
	"time"

	"monity/internal/models"

	"gorm.io/gorm"
)

// obligationOrder lists debts and receivables soonest-due first, undated ones last.
const obligationOrder = "due_date ASC NULLS LAST, created_at DESC"

// obligationFilter scopes a debts or receivables query to the user and the list filters.
// OVERDUE is not stored, so it and PENDING/PARTIAL are matched against the due date.
func obligationFilter(q *gorm.DB, userID int64, status *string, dueFrom, dueTo *time.Time) *gorm.DB {
	q = q.Where("user_id = ?", userID)
	today := models.StartOfDayUTC(time.Now())
	if status != nil && *status != "" {
		switch models.ObligationStatus(*status) {
		case models.ObligationStatusOverdue:
			q = q.Where("status <> ? AND due_date < ?", models.ObligationStatusPaid, today)
		case models.ObligationStatusPaid:
			q = q.Where("status = ?", *status)
		default:
			q = q.Where("status = ? AND (due_date IS NULL OR due_date >= ?)", *status, today)
		}
	}
	if dueFrom != nil {
		q = q.Where("due_date >= ?", dueFrom)
	}
	if dueTo != nil {
		q = q.Where("due_date < ?", dueTo.AddDate(0, 0, 1))
	}
	return q
}
