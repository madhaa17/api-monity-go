package handler

import (
	"encoding/json"
	"net/http"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
	"monity/internal/pkg/validation"
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

	expense, err := h.svc.CreateExpense(r.Context(), userID, req)
	if err != nil {
		writeServiceError(w, r, err, "expense not found", "failed to create expense")
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
	page, limit := parsePageLimit(r, 1, 20, 100)
	filter, err := parseTransactionFilter(r)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}
	expenses, meta, err := h.svc.ListExpenses(r.Context(), userID, filter, r.URL.Query().Get("sort"), page, limit)
	if err != nil {
		writeServiceError(w, r, err, "expense not found", "failed to list expenses")
		return
	}
	response.Success(w, http.StatusOK, "expenses retrieved", port.ListResponse{Items: expenses, Meta: meta})
}

// Summary handles GET /expenses/summary: totals for the same filters as List, plus ?group_by=day|month.
func (h *ExpenseHandler) Summary(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	filter, err := parseTransactionFilter(r)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}
	summary, err := h.svc.SummarizeExpenses(r.Context(), userID, filter, r.URL.Query().Get("group_by"))
	if err != nil {
		writeServiceError(w, r, err, "expense not found", "failed to summarize expenses")
		return
	}
	response.Success(w, http.StatusOK, "expense summary retrieved", summary)
}

func (h *ExpenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if !validation.ValidUUID(uuid) {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid expense uuid", nil)
		return
	}

	expense, err := h.svc.GetExpense(r.Context(), userID, uuid)
	if err != nil {
		writeServiceError(w, r, err, "expense not found", "failed to get expense")
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
	if !validation.ValidUUID(uuid) {
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
		writeServiceError(w, r, err, "expense not found", "failed to update expense")
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
	if !validation.ValidUUID(uuid) {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid expense uuid", nil)
		return
	}

	if err := h.svc.DeleteExpense(r.Context(), userID, uuid); err != nil {
		writeServiceError(w, r, err, "expense not found", "failed to delete expense")
		return
	}

	response.Success(w, http.StatusOK, "expense deleted", nil)
}
