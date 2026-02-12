package handler

import (
	"net/http"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type ActivityHandler struct {
	svc port.ActivityService
}

func NewActivityHandler(svc port.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "day"
	}

	resp, err := h.svc.ListActivities(r.Context(), userID, groupBy)
	if err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to list activities", err.Error())
		return
	}

	response.Success(w, http.StatusOK, "ok", resp)
}
