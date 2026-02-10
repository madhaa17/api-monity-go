package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type ExpenseHandler struct {
	svc port.ExpenseService
}

func NewExpenseHandler(svc port.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{svc: svc}
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req port.CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Amount <= 0 || req.Category == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "missing required fields", nil)
		return
	}

	expense, err := h.svc.CreateExpense(r.Context(), userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "positive") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to create expense", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, "expense created", expense)
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	expenses, err := h.svc.ListExpenses(r.Context(), userID)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list expenses", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "expenses retrieved", expenses)
}

func (h *ExpenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid expense uuid", nil)
		return
	}

	expense, err := h.svc.GetExpense(r.Context(), userID, uuid)
	if err != nil {
		if err.Error() == "expense not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "expense not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get expense", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "expense retrieved", expense)
}

func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid expense uuid", nil)
		return
	}

	var req port.UpdateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	expense, err := h.svc.UpdateExpense(r.Context(), userID, uuid, req)
	if err != nil {
		if err.Error() == "expense not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "expense not found", nil)
			return
		}
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "positive") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to update expense", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "expense updated", expense)
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid expense uuid", nil)
		return
	}

	if err := h.svc.DeleteExpense(r.Context(), userID, uuid); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not owned") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "expense not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to delete expense", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "expense deleted", nil)
}
