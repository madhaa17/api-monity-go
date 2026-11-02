package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type AssetPriceHistoryHandler struct {
	svc port.AssetPriceHistoryService
}

func NewAssetPriceHistoryHandler(svc port.AssetPriceHistoryService) *AssetPriceHistoryHandler {
	return &AssetPriceHistoryHandler{svc: svc}
}

func (h *AssetPriceHistoryHandler) RecordPrice(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	assetUUID := r.PathValue("uuid")
	if strings.TrimSpace(assetUUID) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid asset uuid", nil)
		return
	}

	var req port.RecordPriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	history, err := h.svc.RecordPrice(r.Context(), userID, assetUUID, req)
	if err != nil {
		writeServiceError(w, r, err, "asset not found", "failed to record price")
		return
	}

	response.Success(w, http.StatusCreated, "price recorded", history)
}

func (h *AssetPriceHistoryHandler) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	assetUUID := r.PathValue("uuid")
	if strings.TrimSpace(assetUUID) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid asset uuid", nil)
		return
	}

	// Parse limit from query params
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	histories, err := h.svc.GetPriceHistory(r.Context(), userID, assetUUID, limit)
	if err != nil {
		writeServiceError(w, r, err, "asset not found", "failed to get price history")
		return
	}

	response.Success(w, http.StatusOK, "price history retrieved", histories)
}

func (h *AssetPriceHistoryHandler) FetchAndRecordPrice(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	assetUUID := r.PathValue("uuid")
	if strings.TrimSpace(assetUUID) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid asset uuid", nil)
		return
	}

	history, err := h.svc.FetchAndRecordPrice(r.Context(), userID, assetUUID)
	if err != nil {
		switch {
		case errors.Is(err, port.ErrPriceUnavailable):
			response.ErrorWithLog(w, r, http.StatusServiceUnavailable, "current price unavailable, nothing recorded", err.Error())
		// The price sources report an unknown symbol only in their error text.
		case strings.Contains(err.Error(), "unsupported crypto symbol") || strings.Contains(err.Error(), "status 404"):
			response.ErrorWithLog(w, r, http.StatusBadRequest, "symbol not found at the price source", err.Error())
		default:
			writeServiceError(w, r, err, "asset not found", "failed to fetch and record price")
		}
		return
	}

	response.Success(w, http.StatusCreated, "price fetched and recorded", history)
}
