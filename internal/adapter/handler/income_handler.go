package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type IncomeHandler struct {
	svc port.IncomeService
}

func NewIncomeHandler(svc port.IncomeService) *IncomeHandler {
	return &IncomeHandler{svc: svc}
}

func (h *IncomeHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req port.CreateIncomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	income, err := h.svc.CreateIncome(r.Context(), userID, req)
	if err != nil {
		writeServiceError(w, r, err, "income not found", "failed to create income")
		return
	}

	response.Success(w, http.StatusCreated, "income created", income)
}

func (h *IncomeHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	page, limit := parsePageLimit(r, 1, 20, 100)
	dateFrom, dateTo := parseDateFilter(r)
	incomes, meta, err := h.svc.ListIncomes(r.Context(), userID, dateFrom, dateTo, page, limit)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list incomes", err.Error())
		return
	}
	response.Success(w, http.StatusOK, "incomes retrieved", port.ListResponse{Items: incomes, Meta: meta})
}

func (h *IncomeHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid income uuid", nil)
		return
	}

	income, err := h.svc.GetIncome(r.Context(), userID, uuid)
	if err != nil {
		writeServiceError(w, r, err, "income not found", "failed to get income")
		return
	}

	response.Success(w, http.StatusOK, "income retrieved", income)
}

func (h *IncomeHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid income uuid", nil)
		return
	}

	var req port.UpdateIncomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	income, err := h.svc.UpdateIncome(r.Context(), userID, uuid, req)
	if err != nil {
		writeServiceError(w, r, err, "income not found", "failed to update income")
		return
	}

	response.Success(w, http.StatusOK, "income updated", income)
}

func (h *IncomeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid income uuid", nil)
		return
	}

	if err := h.svc.DeleteIncome(r.Context(), userID, uuid); err != nil {
		writeServiceError(w, r, err, "income not found", "failed to delete income")
		return
	}

	response.Success(w, http.StatusOK, "income deleted", nil)
}
