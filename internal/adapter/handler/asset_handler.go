package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type AssetHandler struct {
	svc port.AssetService
}

func NewAssetHandler(svc port.AssetService) *AssetHandler {
	return &AssetHandler{svc: svc}
}

func (h *AssetHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req port.CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Name == "" || req.Type == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "missing required fields", nil)
		return
	}

	asset, err := h.svc.CreateAsset(r.Context(), userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "must be") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to create asset", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, "asset created", asset)
}

func (h *AssetHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	assets, err := h.svc.ListAssets(r.Context(), userID)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list assets", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "assets retrieved", assets)
}

func (h *AssetHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	asset, err := h.svc.GetAsset(r.Context(), userID, uuid)
	if err != nil {
		if err.Error() == "asset not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "asset not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get asset", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "asset retrieved", asset)
}

func (h *AssetHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req port.UpdateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	asset, err := h.svc.UpdateAsset(r.Context(), userID, uuid, req)
	if err != nil {
		if err.Error() == "asset not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "asset not found", nil)
			return
		}
		if strings.Contains(err.Error(), "must be") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to update asset", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "asset updated", asset)
}

func (h *AssetHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	if err := h.svc.DeleteAsset(r.Context(), userID, uuid); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not owned") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "asset not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to delete asset", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "asset deleted", nil)
}
