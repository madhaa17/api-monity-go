package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

	"github.com/shopspring/decimal"
)

// parsePageLimit reads page and limit from query. Defaults: defaultPage, defaultLimit. limit is capped at maxLimit.
func parsePageLimit(r *http.Request, defaultPage, defaultLimit, maxLimit int) (page, limit int) {
	page = defaultPage
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	limit = defaultLimit
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}
	return page, limit
}

// parseDateFilter returns dateFrom, dateTo (inclusive days, UTC) from query: date_from and/or date_to
// (YYYY-MM-DD), or month=YYYY-MM, or year=YYYY, in that order of precedence. Malformed values are errors.
func parseDateFilter(r *http.Request) (dateFrom, dateTo *time.Time, err error) {
	q := r.URL.Query()
	from, to, month, year := q.Get("date_from"), q.Get("date_to"), q.Get("month"), q.Get("year")
	switch {
	case from != "" || to != "":
		if from != "" {
			t, err := time.Parse("2006-01-02", from)
			if err != nil {
				return nil, nil, errors.New("date_from must be YYYY-MM-DD")
			}
			dateFrom = &t
		}
		if to != "" {
			t, err := time.Parse("2006-01-02", to)
			if err != nil {
				return nil, nil, errors.New("date_to must be YYYY-MM-DD")
			}
			dateTo = &t
		}
	case month != "":
		t, err := time.Parse("2006-01", month)
		if err != nil {
			return nil, nil, errors.New("month must be YYYY-MM")
		}
		end := t.AddDate(0, 1, -1) // last day of month
		dateFrom, dateTo = &t, &end
	case year != "":
		y, err := strconv.Atoi(year)
		if err != nil || y < 1 || y > 9999 {
			return nil, nil, errors.New("year must be YYYY")
		}
		start := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(y, 12, 31, 0, 0, 0, 0, time.UTC)
		dateFrom, dateTo = &start, &end
	}
	return dateFrom, dateTo, nil
}

// parseTransactionFilter reads the income/expense filters from the query: the date filters of
// parseDateFilter, category (comma-separated UUIDs), asset (UUID), q (search), min_amount, max_amount.
func parseTransactionFilter(r *http.Request) (port.TransactionFilter, error) {
	var f port.TransactionFilter
	var err error
	if f.DateFrom, f.DateTo, err = parseDateFilter(r); err != nil {
		return f, err
	}
	q := r.URL.Query()
	for _, c := range strings.Split(q.Get("category"), ",") {
		if c = strings.TrimSpace(c); c == "" {
			continue
		}
		if !validation.ValidUUID(c) {
			return f, errors.New("category must be a comma-separated list of category UUIDs")
		}
		f.CategoryUUIDs = append(f.CategoryUUIDs, c)
	}
	if a := strings.TrimSpace(q.Get("asset")); a != "" {
		if !validation.ValidUUID(a) {
			return f, errors.New("asset must be an asset UUID")
		}
		f.AssetUUID = a
	}
	f.Search = strings.TrimSpace(q.Get("q"))
	if f.MinAmount, err = parseDecimalParam(q.Get("min_amount"), "min_amount"); err != nil {
		return f, err
	}
	if f.MaxAmount, err = parseDecimalParam(q.Get("max_amount"), "max_amount"); err != nil {
		return f, err
	}
	return f, nil
}

func parseDecimalParam(value, name string) (*decimal.Decimal, error) {
	if value == "" {
		return nil, nil
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be a number", name)
	}
	return &d, nil
}

// parseDueFilter returns dueFrom, dueTo from query: due_from, due_to (ISO date).
func parseDueFilter(r *http.Request) (dueFrom, dueTo *time.Time) {
	from := r.URL.Query().Get("due_from")
	to := r.URL.Query().Get("due_to")
	if from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			dueFrom = &start
		}
	}
	if to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
			dueTo = &end
		}
	}
	return dueFrom, dueTo
}

// parseObligationStatus reads the debts/receivables status filter; unknown values are ignored.
func parseObligationStatus(r *http.Request) *string {
	s := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	switch models.ObligationStatus(s) {
	case models.ObligationStatusPending, models.ObligationStatusPartial,
		models.ObligationStatusPaid, models.ObligationStatusOverdue:
		return &s
	}
	return nil
}
