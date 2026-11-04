package handler

import (
	"encoding/json"
	"net/http"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type ReceivableHandler struct {
	svc port.ReceivableService
}

func NewReceivableHandler(svc port.ReceivableService) *ReceivableHandler {
	return &ReceivableHandler{svc: svc}
}

func (h *ReceivableHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var req port.CreateReceivableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	receivable, err := h.svc.CreateReceivable(r.Context(), userID, req)
	if err != nil {
		writeServiceError(w, r, err, "receivable not found", "failed to create receivable")
		return
	}
	response.Success(w, http.StatusCreated, "receivable created", receivable)
}

func (h *ReceivableHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	page, limit := parsePageLimit(r, 1, 20, 100)
	dueFrom, dueTo := parseDueFilter(r)
	status := parseObligationStatus(r)
	receivables, meta, err := h.svc.ListReceivables(r.Context(), userID, status, dueFrom, dueTo, page, limit)
	if err != nil {
		writeServiceError(w, r, err, "receivable not found", "failed to list receivables")
		return
	}
	response.Success(w, http.StatusOK, "receivables retrieved", port.ListResponse{Items: receivables, Meta: meta})
}

func (h *ReceivableHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	receivable, err := h.svc.GetReceivable(r.Context(), userID, r.PathValue("uuid"))
	if err != nil {
		writeServiceError(w, r, err, "receivable not found", "failed to get receivable")
		return
	}
	response.Success(w, http.StatusOK, "receivable retrieved", receivable)
}

func (h *ReceivableHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var req port.UpdateReceivableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	receivable, err := h.svc.UpdateReceivable(r.Context(), userID, r.PathValue("uuid"), req)
	if err != nil {
		writeServiceError(w, r, err, "receivable not found", "failed to update receivable")
		return
	}
	response.Success(w, http.StatusOK, "receivable updated", receivable)
}

func (h *ReceivableHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	if err := h.svc.DeleteReceivable(r.Context(), userID, r.PathValue("uuid")); err != nil {
		writeServiceError(w, r, err, "receivable not found", "failed to delete receivable")
		return
	}
	response.Success(w, http.StatusOK, "receivable deleted", nil)
}

func (h *ReceivableHandler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var req port.CreateReceivablePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	payment, err := h.svc.RecordReceivablePayment(r.Context(), userID, r.PathValue("uuid"), req)
	if err != nil {
		writeServiceError(w, r, err, "receivable not found", "failed to record payment")
		return
	}
	response.Success(w, http.StatusCreated, "payment recorded", payment)
}

func (h *ReceivableHandler) ListPayments(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	payments, err := h.svc.ListReceivablePayments(r.Context(), userID, r.PathValue("uuid"))
	if err != nil {
		writeServiceError(w, r, err, "receivable not found", "failed to list payments")
		return
	}
	response.Success(w, http.StatusOK, "payments retrieved", payments)
}

func (h *ReceivableHandler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	err := h.svc.DeleteReceivablePayment(r.Context(), userID, r.PathValue("uuid"), r.PathValue("paymentUuid"))
	if err != nil {
		writeServiceError(w, r, err, "receivable or payment not found", "failed to delete payment")
		return
	}
	response.Success(w, http.StatusOK, "payment deleted", nil)
}
