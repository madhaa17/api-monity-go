package handler

import (
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type PerformanceHandler struct {
	svc port.AssetPerformanceService
}

func NewPerformanceHandler(svc port.AssetPerformanceService) *PerformanceHandler {
	return &PerformanceHandler{svc: svc}
}

func (h *PerformanceHandler) GetAssetPerformance(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.Error(w, http.StatusBadRequest, "invalid asset uuid", nil)
		return
	}

	// Get currency from query params (optional)
	currency := r.URL.Query().Get("currency")

	performance, err := h.svc.GetAssetPerformance(r.Context(), userID, uuid, currency)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.Error(w, http.StatusNotFound, "asset not found", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get asset performance", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "asset performance retrieved", performance)
}

func (h *PerformanceHandler) GetPortfolioPerformance(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	// Get currency from query params (optional)
	currency := r.URL.Query().Get("currency")

	performance, err := h.svc.GetPortfolioPerformance(r.Context(), userID, currency)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get portfolio performance", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "portfolio performance retrieved", performance)
}
