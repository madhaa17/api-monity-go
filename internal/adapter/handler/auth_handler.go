package handler

import (
	"encoding/json"
	"net/http"

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
		response.Error(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "missing required fields", map[string]string{
			"email":    "required",
			"password": "required",
			"name":     "required",
		})
		return
	}

	resp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		if err.Error() == "email already registered" {
			response.Error(w, http.StatusConflict, "registration failed", err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Success(w, http.StatusCreated, "registration successful", resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req port.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "missing required fields", map[string]string{
			"email":    "required",
			"password": "required",
		})
		return
	}

	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		if err.Error() == "invalid email or password" {
			response.Error(w, http.StatusUnauthorized, "login failed", "invalid email or password")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Success(w, http.StatusOK, "login successful", resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "refresh_token required", nil)
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		switch err.Error() {
		case "refresh token required", "invalid or expired refresh token", "invalid refresh token claims", "invalid refresh token payload", "user not found":
			response.Error(w, http.StatusUnauthorized, "refresh failed", err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Success(w, http.StatusOK, "token refreshed", resp)
}
