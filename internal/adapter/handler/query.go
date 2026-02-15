package handler

import (
	"net/http"
	"strconv"
	"time"
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

// parseDateFilter returns dateFrom, dateTo from query: date_from, date_to (ISO), or month=YYYY-MM, or year=YYYY.
func parseDateFilter(r *http.Request) (dateFrom, dateTo *time.Time) {
	from := r.URL.Query().Get("date_from")
	to := r.URL.Query().Get("date_to")
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	if from != "" && to != "" {
		t1, err1 := time.Parse("2006-01-02", from)
		t2, err2 := time.Parse("2006-01-02", to)
		if err1 == nil && err2 == nil {
			start := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
			end := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
			return &start, &end
		}
	}
	if month != "" {
		t, err := time.Parse("2006-01", month)
		if err == nil {
			start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			end := start.AddDate(0, 1, -1) // last day of month
			return &start, &end
		}
	}
	if year != "" {
		y, err := strconv.Atoi(year)
		if err == nil && y >= 1 && y <= 9999 {
			start := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
			end := time.Date(y, 12, 31, 0, 0, 0, 0, time.UTC)
			return &start, &end
		}
	}
	return nil, nil
}
