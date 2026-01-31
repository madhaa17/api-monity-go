package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"monity/internal/config"
	"monity/internal/pkg/response"

	"github.com/golang-jwt/jwt/v5"
)

type CtxKey string

const (
	CtxKeyUserID CtxKey = "userID"
	CtxKeyUUID   CtxKey = "uuid"
	CtxKeyRole   CtxKey = "role"
)

type AuthMiddleware struct {
	cfg *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg}
}

func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, http.StatusUnauthorized, "authorization header required", nil)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(w, http.StatusUnauthorized, "invalid authorization format", nil)
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.cfg.Jwt.Secret), nil
		})

		if err != nil || !token.Valid {
			response.Error(w, http.StatusUnauthorized, "invalid or expired token", nil)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "invalid token claims", nil)
			return
		}

		// Extract claims (safely handling types)
		userIDFloat, okID := claims["sub"].(float64)
		uuid, okUUID := claims["uuid"].(string) 
		role, okRole := claims["role"].(string)

		if !okID || !okUUID {
			response.Error(w, http.StatusUnauthorized, "invalid token payload", nil)
			return
		}

		ctx := context.WithValue(r.Context(), CtxKeyUserID, int64(userIDFloat))
		ctx = context.WithValue(ctx, CtxKeyUUID, uuid)
		if okRole {
			ctx = context.WithValue(ctx, CtxKeyRole, role)
		}

		next(w, r.WithContext(ctx))
	}
}
