package handler

import (
	"net/http"
	"strconv"
	"strings"

	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

var (
	cryptoChartDaysAllowed = map[int]bool{1: true, 7: true, 14: true, 30: true, 90: true, 180: true, 365: true}
	stockChartRangeAllowed = map[string]bool{
		"1d": true, "5d": true, "1mo": true, "3mo": true, "6mo": true,
		"1y": true, "2y": true, "5y": true, "10y": true, "ytd": true, "max": true,
	}
	stockChartIntervalAllowed = map[string]bool{
		"5m": true, "15m": true, "30m": true, "1h": true, "1d": true, "1wk": true, "1mo": true,
	}
	// stockIntradayIntervals are the intervals only allowed for the stockIntradayRanges.
	stockIntradayIntervals = map[string]bool{"5m": true, "15m": true, "30m": true, "1h": true}
	stockIntradayRanges    = map[string]bool{"1d": true, "5d": true}
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
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unsupported crypto symbol") {
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
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "status 404") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "price not found", err.Error())
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to fetch price", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "stock price retrieved", price)
}

// SearchSymbols handles GET /prices/search?type=STOCK|CRYPTO&q=bbca for the asset form's symbol picker.
func (h *PriceHandler) SearchSymbols(w http.ResponseWriter, r *http.Request) {
	assetType := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("type")))
	if assetType != "STOCK" && assetType != "CRYPTO" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "type must be one of: STOCK, CRYPTO", nil)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 50 {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "q must be at most 50 characters", nil)
		return
	}

	results, err := h.svc.SearchSymbols(r.Context(), assetType, query)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusBadGateway, "failed to search symbols", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "symbols retrieved", results)
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
		response.ErrorWithLog(w, r, http.StatusBadRequest, "days must be one of: 1, 7, 14, 30, 90, 180, 365", nil)
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
		response.ErrorWithLog(w, r, http.StatusBadRequest, "interval must be one of: 5m, 15m, 30m, 1h, 1d, 1wk, 1mo", nil)
		return
	}
	if stockIntradayIntervals[interval] && !stockIntradayRanges[rangeParam] {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "intraday intervals (5m, 15m, 30m, 1h) need range 1d or 5d", nil)
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
