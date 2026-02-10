package handler

import (
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type InsightHandler struct {
	svc port.InsightService
}

func NewInsightHandler(svc port.InsightService) *InsightHandler {
	return &InsightHandler{svc: svc}
}

func (h *InsightHandler) GetCashflow(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	// Get month from query params (optional, defaults to current month)
	month := r.URL.Query().Get("month")

	summary, err := h.svc.GetCashflowSummary(r.Context(), userID, month)
	if err != nil {
		if strings.Contains(err.Error(), "invalid month format") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get cashflow summary", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "cashflow summary retrieved", summary)
}

func (h *InsightHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	overview, err := h.svc.GetFinancialOverview(r.Context(), userID)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get financial overview", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "financial overview retrieved", overview)
}
