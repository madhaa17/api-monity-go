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
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to create debt", err.Error())
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
	var status *string
	if s := strings.TrimSpace(r.URL.Query().Get("status")); s != "" && isValidObligationStatus(s) {
		status = &s
	}
	debts, meta, err := h.svc.ListDebts(r.Context(), userID, status, dueFrom, dueTo, page, limit)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list debts", err.Error())
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

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid debt uuid", nil)
		return
	}

	debt, err := h.svc.GetDebt(r.Context(), userID, uuid)
	if err != nil {
		if err.Error() == "debt not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "debt not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get debt", err.Error())
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

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid debt uuid", nil)
		return
	}

	var req port.UpdateDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	debt, err := h.svc.UpdateDebt(r.Context(), userID, uuid, req)
	if err != nil {
		if err.Error() == "debt not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "debt not found", nil)
			return
		}
		if strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "must be") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to update debt", err.Error())
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

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid debt uuid", nil)
		return
	}

	if err := h.svc.DeleteDebt(r.Context(), userID, uuid); err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "debt not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to delete debt", err.Error())
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

	debtUUID := r.PathValue("uuid")
	if strings.TrimSpace(debtUUID) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid debt uuid", nil)
		return
	}

	var req port.CreateDebtPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	payment, err := h.svc.RecordDebtPayment(r.Context(), userID, debtUUID, req)
	if err != nil {
		if err.Error() == "debt not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "debt not found", nil)
			return
		}
		if strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "already fully paid") || strings.Contains(err.Error(), "cannot exceed") || strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to record payment", err.Error())
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

	debtUUID := r.PathValue("uuid")
	if strings.TrimSpace(debtUUID) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid debt uuid", nil)
		return
	}

	payments, err := h.svc.ListDebtPayments(r.Context(), userID, debtUUID)
	if err != nil {
		if err.Error() == "debt not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "debt not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list payments", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "payments retrieved", payments)
}

func isValidObligationStatus(s string) bool {
	switch s {
	case string(models.ObligationStatusPending), string(models.ObligationStatusPartial),
		string(models.ObligationStatusPaid), string(models.ObligationStatusOverdue):
		return true
	}
	return false
}
