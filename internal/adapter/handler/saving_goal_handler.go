package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type SavingGoalHandler struct {
	svc port.SavingGoalService
}

func NewSavingGoalHandler(svc port.SavingGoalService) *SavingGoalHandler {
	return &SavingGoalHandler{svc: svc}
}

func (h *SavingGoalHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req port.CreateSavingGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if strings.TrimSpace(req.Title) == "" || req.TargetAmount <= 0 {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "missing required fields", nil)
		return
	}

	goal, err := h.svc.CreateSavingGoal(r.Context(), userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "negative") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to create saving goal", err.Error())
		return
	}

	response.Success(w, http.StatusCreated, "saving goal created", goal)
}

func (h *SavingGoalHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	goals, err := h.svc.ListSavingGoals(r.Context(), userID)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list saving goals", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "saving goals retrieved", goals)
}

func (h *SavingGoalHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid saving goal uuid", nil)
		return
	}

	goal, err := h.svc.GetSavingGoal(r.Context(), userID, uuid)
	if err != nil {
		if err.Error() == "saving goal not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "saving goal not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get saving goal", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "saving goal retrieved", goal)
}

func (h *SavingGoalHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid saving goal uuid", nil)
		return
	}

	var req port.UpdateSavingGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	goal, err := h.svc.UpdateSavingGoal(r.Context(), userID, uuid, req)
	if err != nil {
		if err.Error() == "saving goal not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "saving goal not found", nil)
			return
		}
		if strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "negative") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to update saving goal", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "saving goal updated", goal)
}

func (h *SavingGoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	uuid := r.PathValue("uuid")
	if strings.TrimSpace(uuid) == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid saving goal uuid", nil)
		return
	}

	if err := h.svc.DeleteSavingGoal(r.Context(), userID, uuid); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not owned") {
			response.ErrorWithLog(w, r, http.StatusNotFound, "saving goal not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to delete saving goal", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "saving goal deleted", nil)
}
