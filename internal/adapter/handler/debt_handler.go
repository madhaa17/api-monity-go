package handler

import (
	"encoding/json"
	"net/http"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type DebtHandler struct {
	svc port.DebtService
}

func NewDebtHandler(svc port.DebtService) *DebtHandler {
	return &DebtHandler{svc: svc}
}

func (h *DebtHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var req port.CreateDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	debt, err := h.svc.CreateDebt(r.Context(), userID, req)
	if err != nil {
		writeServiceError(w, r, err, "debt not found", "failed to create debt")
		return
	}
	response.Success(w, http.StatusCreated, "debt created", debt)
}

func (h *DebtHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	page, limit := parsePageLimit(r, 1, 20, 100)
	dueFrom, dueTo := parseDueFilter(r)
	status := parseObligationStatus(r)
	debts, meta, err := h.svc.ListDebts(r.Context(), userID, status, dueFrom, dueTo, page, limit)
	if err != nil {
		writeServiceError(w, r, err, "debt not found", "failed to list debts")
		return
	}
	response.Success(w, http.StatusOK, "debts retrieved", port.ListResponse{Items: debts, Meta: meta})
}

func (h *DebtHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	debt, err := h.svc.GetDebt(r.Context(), userID, r.PathValue("uuid"))
	if err != nil {
		writeServiceError(w, r, err, "debt not found", "failed to get debt")
		return
	}
	response.Success(w, http.StatusOK, "debt retrieved", debt)
}

func (h *DebtHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var req port.UpdateDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	debt, err := h.svc.UpdateDebt(r.Context(), userID, r.PathValue("uuid"), req)
	if err != nil {
		writeServiceError(w, r, err, "debt not found", "failed to update debt")
		return
	}
	response.Success(w, http.StatusOK, "debt updated", debt)
}

func (h *DebtHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	if err := h.svc.DeleteDebt(r.Context(), userID, r.PathValue("uuid")); err != nil {
		writeServiceError(w, r, err, "debt not found", "failed to delete debt")
		return
	}
	response.Success(w, http.StatusOK, "debt deleted", nil)
}

func (h *DebtHandler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var req port.CreateDebtPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	payment, err := h.svc.RecordDebtPayment(r.Context(), userID, r.PathValue("uuid"), req)
	if err != nil {
		writeServiceError(w, r, err, "debt not found", "failed to record payment")
		return
	}
	response.Success(w, http.StatusCreated, "payment recorded", payment)
}

func (h *DebtHandler) ListPayments(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	payments, err := h.svc.ListDebtPayments(r.Context(), userID, r.PathValue("uuid"))
	if err != nil {
		writeServiceError(w, r, err, "debt not found", "failed to list payments")
		return
	}
	response.Success(w, http.StatusOK, "payments retrieved", payments)
}

func (h *DebtHandler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	err := h.svc.DeleteDebtPayment(r.Context(), userID, r.PathValue("uuid"), r.PathValue("paymentUuid"))
	if err != nil {
		writeServiceError(w, r, err, "debt or payment not found", "failed to delete payment")
		return
	}
	response.Success(w, http.StatusOK, "payment deleted", nil)
}
