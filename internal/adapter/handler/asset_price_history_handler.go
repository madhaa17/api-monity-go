package handler

import (
	"encoding/json"
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
		if strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "asset not found", nil)
			return
		}
		if strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "required") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to record price", err.Error())
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
		if strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "asset not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get price history", err.Error())
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
		if strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "asset not found", nil)
			return
		}
		if strings.Contains(err.Error(), "no symbol") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "unsupported") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to fetch and record price", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, "price fetched and recorded", history)
}
