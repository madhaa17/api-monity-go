package handler

import (
	"encoding/json"
	"net/http"
	"strings"

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

	rec, err := h.svc.CreateReceivable(r.Context(), userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to create receivable", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, "receivable created", rec)
}

func (h *ReceivableHandler) List(w http.ResponseWriter, r *http.Request) {
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
	recs, meta, err := h.svc.ListReceivables(r.Context(), userID, status, dueFrom, dueTo, page, limit)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list receivables", err.Error())
		return
	}
	response.Success(w, http.StatusOK, "receivables retrieved", port.ListResponse{Items: recs, Meta: meta})
}

func (h *ReceivableHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid receivable uuid", nil)
		return
	}

	rec, err := h.svc.GetReceivable(r.Context(), userID, uuid)
	if err != nil {
		if err.Error() == "receivable not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "receivable not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get receivable", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "receivable retrieved", rec)
}

func (h *ReceivableHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid receivable uuid", nil)
		return
	}

	var req port.UpdateReceivableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	rec, err := h.svc.UpdateReceivable(r.Context(), userID, uuid, req)
	if err != nil {
		if err.Error() == "receivable not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "receivable not found", nil)
			return
		}
		if strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "must be") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to update receivable", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "receivable updated", rec)
}

func (h *ReceivableHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid receivable uuid", nil)
		return
	}

	if err := h.svc.DeleteReceivable(r.Context(), userID, uuid); err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "receivable not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to delete receivable", err.Error())
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

	recUUID := r.PathValue("uuid")
	if strings.TrimSpace(recUUID) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid receivable uuid", nil)
		return
	}

	var req port.CreateReceivablePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	payment, err := h.svc.RecordReceivablePayment(r.Context(), userID, recUUID, req)
	if err != nil {
		if err.Error() == "receivable not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "receivable not found", nil)
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

func (h *ReceivableHandler) ListPayments(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	recUUID := r.PathValue("uuid")
	if strings.TrimSpace(recUUID) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid receivable uuid", nil)
		return
	}

	payments, err := h.svc.ListReceivablePayments(r.Context(), userID, recUUID)
	if err != nil {
		if err.Error() == "receivable not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "receivable not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list payments", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "payments retrieved", payments)
}
