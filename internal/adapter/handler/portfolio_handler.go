package handler

import (
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type PortfolioHandler struct {
	svc port.PortfolioService
}

func NewPortfolioHandler(svc port.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{svc: svc}
}

func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = port.DefaultCurrency
	}
	currency = strings.ToUpper(currency)

	portfolio, err := h.svc.GetPortfolio(r.Context(), userID, currency)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get portfolio", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "portfolio retrieved", portfolio)
}

func (h *PortfolioHandler) GetAssetValue(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid asset uuid", nil)
		return
	}

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = port.DefaultCurrency
	}
	currency = strings.ToUpper(currency)

	assetValue, err := h.svc.GetAssetValue(r.Context(), userID, uuid, currency)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "asset not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get asset value", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "asset value retrieved", assetValue)
}
