package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/response"
)

type CategoryHandler struct {
	svc port.CategoryService
}

func NewCategoryHandler(svc port.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// List handles GET /categories with an optional ?type=INCOME|EXPENSE filter.
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var categoryType *models.CategoryType
	if t := strings.TrimSpace(r.URL.Query().Get("type")); t != "" {
		ct := models.CategoryType(strings.ToUpper(t))
		categoryType = &ct
	}

	categories, err := h.svc.ListCategories(r.Context(), userID, categoryType)
	if err != nil {
		writeServiceError(w, r, err, "category not found", "failed to list categories")
		return
	}
	response.Success(w, http.StatusOK, "categories retrieved", categories)
}

func (h *CategoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid category uuid", nil)
		return
	}

	category, err := h.svc.GetCategory(r.Context(), userID, uuid)
	if err != nil {
		writeServiceError(w, r, err, "category not found", "failed to get category")
		return
	}
	response.Success(w, http.StatusOK, "category retrieved", category)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req port.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	req.Type = models.CategoryType(strings.ToUpper(strings.TrimSpace(string(req.Type))))

	category, err := h.svc.CreateCategory(r.Context(), userID, req)
	if err != nil {
		writeServiceError(w, r, err, "category not found", "failed to create category")
		return
	}
	response.Success(w, http.StatusCreated, "category created", category)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid category uuid", nil)
		return
	}

	var req port.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	category, err := h.svc.UpdateCategory(r.Context(), userID, uuid, req)
	if err != nil {
		writeServiceError(w, r, err, "category not found", "failed to update category")
		return
	}
	response.Success(w, http.StatusOK, "category updated", category)
}

// Delete handles DELETE /categories/{uuid}; a category still in use needs ?reassign_to=<uuid>.
func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid category uuid", nil)
		return
	}

	if err := h.svc.DeleteCategory(r.Context(), userID, uuid, r.URL.Query().Get("reassign_to")); err != nil {
		writeServiceError(w, r, err, "category not found", "failed to delete category")
		return
	}
	response.Success(w, http.StatusOK, "category deleted", nil)
}
