package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/models"
)

type failingHistoryService struct {
	port.AssetPriceHistoryService
	err error
}

func (s failingHistoryService) FetchAndRecordPrice(context.Context, int64, string) (*models.AssetPriceHistory, error) {
	return nil, s.err
}

func TestFetchAndRecordPrice_ErrorStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"asset missing", fmt.Errorf("asset: %w", port.ErrNotFound), http.StatusNotFound},
		{"price unavailable", fmt.Errorf("no rate: %w", port.ErrPriceUnavailable), http.StatusServiceUnavailable},
		{"unknown coin", errors.New("fetch price: unsupported crypto symbol: NOPE"), http.StatusBadRequest},
		// A price source's "not found" is not a missing asset.
		{"price not found", errors.New("fetch price: price not found for SOL"), http.StatusInternalServerError},
		{"no symbol", port.NewValidationError("asset has no symbol for price lookup"), http.StatusBadRequest},
	}
	for _, tt := range tests {
		h := NewAssetPriceHistoryHandler(failingHistoryService{err: tt.err})
		req := httptest.NewRequest(http.MethodPost, "/assets/a-1/prices/fetch", nil)
		req.SetPathValue("uuid", "a-1")
		req = req.WithContext(context.WithValue(req.Context(), middleware.CtxKeyUserID, int64(1)))
		rec := httptest.NewRecorder()
		h.FetchAndRecordPrice(rec, req)
		if rec.Code != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.name, rec.Code, tt.want)
		}
	}
}
