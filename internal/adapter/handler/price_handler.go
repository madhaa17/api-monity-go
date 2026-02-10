package handler

import (
	"net/http"
	"strings"

	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type PriceHandler struct {
	svc port.PriceService
}

func NewPriceHandler(svc port.PriceService) *PriceHandler {
	return &PriceHandler{svc: svc}
}

func (h *PriceHandler) GetCryptoPrice(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	if strings.TrimSpace(symbol) == "" {
		response.Error(w, http.StatusBadRequest, "symbol is required", nil)
		return
	}

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = port.DefaultCurrency
	}
	currency = strings.ToUpper(currency)

	price, err := h.svc.GetCryptoPriceWithCurrency(r.Context(), symbol, currency)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.Error(w, http.StatusNotFound, "price not found", err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to fetch price", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "crypto price retrieved", price)
}

func (h *PriceHandler) GetStockPrice(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	if strings.TrimSpace(symbol) == "" {
		response.Error(w, http.StatusBadRequest, "symbol is required", nil)
		return
	}

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = port.DefaultCurrency
	}
	currency = strings.ToUpper(currency)

	price, err := h.svc.GetStockPriceWithCurrency(r.Context(), symbol, currency)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.Error(w, http.StatusNotFound, "price not found", err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to fetch price", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "stock price retrieved", price)
}
