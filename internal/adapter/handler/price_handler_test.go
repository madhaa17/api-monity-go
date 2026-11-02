package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"monity/internal/core/port"
)

// chartOnlyPriceService answers GetStockChart with an empty chart; the rest is unused.
type chartOnlyPriceService struct{ port.PriceService }

func (chartOnlyPriceService) GetStockChart(ctx context.Context, symbol, rangeParam, interval string) (*port.ChartResponse, error) {
	return &port.ChartResponse{Symbol: symbol}, nil
}

func TestGetStockChart_IntervalValidation(t *testing.T) {
	h := NewPriceHandler(chartOnlyPriceService{})
	tests := []struct {
		query string
		want  int
	}{
		{"range=1d&interval=5m", http.StatusOK},
		{"range=5d&interval=1h", http.StatusOK},
		{"range=1y&interval=1d", http.StatusOK},
		{"range=1mo&interval=5m", http.StatusBadRequest}, // intraday only for 1d and 5d
		{"range=1d&interval=1m", http.StatusBadRequest},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/prices/stock/BBCA.JK/chart?"+tt.query, nil)
		req.SetPathValue("symbol", "BBCA.JK")
		rec := httptest.NewRecorder()
		h.GetStockChart(rec, req)
		if rec.Code != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.query, rec.Code, tt.want)
		}
	}
}
