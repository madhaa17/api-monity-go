package handler

import (
	"net/http"
	"strings"
	"time"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

// ActivityHandler handles HTTP requests for the activities API.
type ActivityHandler struct {
	svc port.ActivityService
}

// NewActivityHandler returns a new ActivityHandler with the given service.
func NewActivityHandler(svc port.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

// List returns activities for the authenticated user, optionally filtered by date and grouped by day, month, or year.
func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "day"
	}

	dateParam := strings.TrimSpace(r.URL.Query().Get("date"))
	tzParam := strings.TrimSpace(r.URL.Query().Get("tz"))

	dateFilter, timezone := resolveDateFilter(dateParam, tzParam)

	resp, err := h.svc.ListActivities(r.Context(), userID, groupBy, dateFilter, timezone)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list activities", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "ok", resp)
}

// resolveDateFilter derives dateFilter and timezone from query params.
// For dateParam "today", uses time.Now() in tz or server local. For a specific date use query date=YYYY-MM-DD.
func resolveDateFilter(dateParam, tzParam string) (dateFilter, timezone string) {
	var loc *time.Location
	if tzParam != "" {
		if l, err := time.LoadLocation(tzParam); err == nil {
			loc = l
		}
	}

	switch {
	case dateParam == "":
		return "", ""
	case strings.EqualFold(dateParam, "today"):
		if loc != nil {
			return time.Now().In(loc).Format("2006-01-02"), tzParam
		}
		return time.Now().Format("2006-01-02"), ""
	default:
		dateFilter = dateParam
		if loc != nil {
			timezone = tzParam
		}
		return dateFilter, timezone
	}
}
