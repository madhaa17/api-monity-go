package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"monity/internal/adapter/middleware"
	"monity/internal/core/port"
	"monity/internal/pkg/response"
)

type AuthHandler struct {
	svc port.AuthService
}

func NewAuthHandler(svc port.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req port.RegistryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "missing required fields", map[string]string{
			"email":    "required",
			"password": "required",
			"name":     "required",
		})
		return
	}

	resp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		if err.Error() == "email already registered" {
			slog.Warn("register_failed", "email", req.Email, "reason", err.Error())
			response.ErrorWithLog(w, r, http.StatusConflict, "registration failed", err.Error())
			return
		}
		if err.Error() == "invalid email format" || strings.HasPrefix(err.Error(), "password ") || strings.HasPrefix(err.Error(), "name must be") || strings.HasPrefix(err.Error(), "email must be") {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		slog.Error("register_error", "email", req.Email, "error", err)
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "internal server error", nil)
		return
	}
	slog.Info("register_success", "email", req.Email, "user_id", resp.User.ID)
	response.Success(w, http.StatusCreated, "registration successful", resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req port.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "missing required fields", map[string]string{
			"email":    "required",
			"password": "required",
		})
		return
	}

	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		if err.Error() == "invalid email or password" {
			slog.Warn("login_failed", "email", req.Email, "reason", "invalid email or password")
			response.ErrorWithLog(w, r, http.StatusUnauthorized, "login failed", "invalid email or password")
			return
		}
		if err.Error() == "invalid email format" {
			response.ErrorWithLog(w, r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		slog.Error("login_error", "email", req.Email, "error", err)
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "internal server error", nil)
		return
	}
	slog.Info("login_success", "email", req.Email, "user_id", resp.User.ID)
	response.Success(w, http.StatusOK, "login successful", resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if req.RefreshToken == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "refresh_token required", nil)
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		switch err.Error() {
		case "refresh token required", "invalid or expired refresh token", "invalid refresh token claims", "invalid refresh token payload", "user not found":
			response.ErrorWithLog(w, r, http.StatusUnauthorized, "refresh failed", err.Error())
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Success(w, http.StatusOK, "token refreshed", resp)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	user, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			response.ErrorWithLog(w, r, http.StatusNotFound, "user not found", nil)
			return
		}
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "failed to get user", nil)
		return
	}
	response.Success(w, http.StatusOK, "ok", user)
}

// Logout revokes the access token from the Authorization header and the refresh token from the
// body. It needs no valid session, so a client whose access token has expired can still revoke its
// refresh token; at least one of the two must be sent.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var accessToken string
	if parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2); len(parts) == 2 && parts[0] == "Bearer" {
		accessToken = parts[1]
	}

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if accessToken == "" && body.RefreshToken == "" {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "access token or refresh_token required", nil)
		return
	}

	if err := h.svc.Logout(r.Context(), accessToken, body.RefreshToken); err != nil {
		response.ErrorWithLog(w, r, http.StatusInternalServerError, "logout failed", err.Error())
		return
	}
	slog.Info("logout")
	response.Success(w, http.StatusOK, "logged out", nil)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxKeyUserID).(int64)
	if !ok {
		response.ErrorWithLog(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.ErrorWithLog(w, r, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	resp, err := h.svc.ChangePassword(r.Context(), userID, port.ChangePasswordRequest{
		CurrentPassword: body.CurrentPassword,
		NewPassword:     body.NewPassword,
	})
	if err != nil {
		var validationErr *port.ValidationError
		if errors.As(err, &validationErr) {
			slog.Warn("change_password_failed", "user_id", userID, "reason", validationErr.Message)
		}
		writeServiceError(w, r, err, "user not found", "failed to change password")
		return
	}
	slog.Info("password_changed", "user_id", userID)
	response.Success(w, http.StatusOK, "password changed", resp)
}
