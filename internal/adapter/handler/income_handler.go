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
		response.Error(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req port.CreateIncomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Amount <= 0 || strings.TrimSpace(req.Source) == "" {
		response.Error(w, http.StatusBadRequest, "missing required fields", nil)
		return
	}

	income, err := h.svc.CreateIncome(r.Context(), userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "positive") {
			response.Error(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create income", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, "income created", income)
}

func (h *IncomeHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	incomes, err := h.svc.ListIncomes(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list incomes", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "incomes retrieved", incomes)
}

func (h *IncomeHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.Error(w, http.StatusBadRequest, "invalid income uuid", nil)
		return
	}

	income, err := h.svc.GetIncome(r.Context(), userID, uuid)
	if err != nil {
		if err.Error() == "income not found" {
			response.Error(w, http.StatusNotFound, "income not found", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get income", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "income retrieved", income)
}

func (h *IncomeHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.Error(w, http.StatusBadRequest, "invalid income uuid", nil)
		return
	}

	var req port.UpdateIncomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	income, err := h.svc.UpdateIncome(r.Context(), userID, uuid, req)
	if err != nil {
		if err.Error() == "income not found" {
			response.Error(w, http.StatusNotFound, "income not found", nil)
			return
		}
		if strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "positive") {
			response.Error(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update income", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "income updated", income)
}

func (h *IncomeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.Error(w, http.StatusBadRequest, "invalid income uuid", nil)
		return
	}

	if err := h.svc.DeleteIncome(r.Context(), userID, uuid); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not owned") {
			response.Error(w, http.StatusNotFound, "income not found", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to delete income", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "income deleted", nil)
}
