package routes

import (
	"monity/internal/adapter/middleware"
	"monity/internal/config"
)

func (r *Router) registerAuthRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/auth/register", r.h.Auth.Register)
	r.mux.HandleFunc("POST "+APIPrefix+"/auth/login", r.h.Auth.Login)
	r.mux.HandleFunc("POST "+APIPrefix+"/auth/refresh", r.h.Auth.Refresh)
	r.mux.HandleFunc("GET "+APIPrefix+"/auth/me", r.auth.RequireAuth(r.h.Auth.Me))
	r.mux.HandleFunc("POST "+APIPrefix+"/auth/logout", r.h.Auth.Logout)

	// 5 attempts per 15 minutes per client, so a stolen session cannot brute-force the current password.
	passwordLimit := middleware.NewRateLimitMiddleware(&config.RateLimitConfig{TTLSeconds: 15 * 60, Limit: 5})
	r.mux.Handle("PUT "+APIPrefix+"/auth/password", passwordLimit.Handler(r.auth.RequireAuth(r.h.Auth.ChangePassword)))
}
