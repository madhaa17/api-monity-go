package handler

import (
	"net/http"
	"strconv"
	"strings"

	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

var (
	cryptoChartDaysAllowed = map[int]bool{1: true, 7: true, 14: true, 30: true, 90: true}
	stockChartRangeAllowed = map[string]bool{
		"1d": true, "5d": true, "1mo": true, "3mo": true, "6mo": true,
		"1y": true, "2y": true, "5y": true, "10y": true, "ytd": true, "max": true,
	}
	stockChartIntervalAllowed = map[string]bool{"1d": true, "1wk": true, "1mo": true}
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
		response.ErrorWithLog(w, r, http.StatusBadRequest, "symbol is required", nil)
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
			response.ErrorWithLog(w, r, http.StatusNotFound, "price not found", err.Error())
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to fetch price", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "crypto price retrieved", price)
}

func (h *PriceHandler) GetStockPrice(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	if strings.TrimSpace(symbol) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "symbol is required", nil)
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
			response.ErrorWithLog(w, r, http.StatusNotFound, "price not found", err.Error())
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to fetch price", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "stock price retrieved", price)
}

func (h *PriceHandler) GetCryptoChart(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(r.PathValue("symbol"))
	if symbol == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "symbol is required", nil)
		return
	}

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = port.DefaultCurrency
	}
	currency = strings.ToUpper(currency)

	daysStr := r.URL.Query().Get("days")
	if daysStr == "" {
		daysStr = "7"
	}
	days, err := strconv.Atoi(daysStr)
	if err != nil || !cryptoChartDaysAllowed[days] {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "days must be one of: 1, 7, 14, 30, 90", nil)
		return
	}

	chart, err := h.svc.GetCryptoChart(r.Context(), symbol, currency, days)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported crypto symbol") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "symbol not supported", err.Error())
			return
		}
		response.ErrorWithLog(w, r, http.StatusBadGateway, "failed to fetch crypto chart", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "crypto chart retrieved", chart)
}

func (h *PriceHandler) GetStockChart(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(r.PathValue("symbol"))
	if symbol == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "symbol is required", nil)
		return
	}

	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "1mo"
	}
	if !stockChartRangeAllowed[rangeParam] {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "range must be one of: 1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max", nil)
		return
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1d"
	}
	if !stockChartIntervalAllowed[interval] {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "interval must be one of: 1d, 1wk, 1mo", nil)
		return
	}

	chart, err := h.svc.GetStockChart(r.Context(), symbol, rangeParam, interval)
	if err != nil {
		if strings.Contains(err.Error(), "no chart data") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "chart not found", err.Error())
			return
		}
		response.ErrorWithLog(w, r, http.StatusBadGateway, "failed to fetch stock chart", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "stock chart retrieved", chart)
}
