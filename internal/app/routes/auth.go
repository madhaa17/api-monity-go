package routes

func (r *Router) registerAuthRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/auth/register", r.h.Auth.Register)
	r.mux.HandleFunc("POST "+APIPrefix+"/auth/login", r.h.Auth.Login)
	r.mux.HandleFunc("POST "+APIPrefix+"/auth/refresh", r.h.Auth.Refresh)
	r.mux.HandleFunc("GET "+APIPrefix+"/auth/me", r.auth.RequireAuth(r.h.Auth.Me))
	r.mux.HandleFunc("POST "+APIPrefix+"/auth/logout", r.auth.RequireAuth(r.h.Auth.Logout))
}
