package routes

func (r *Router) registerActivityRoutes() {
	r.mux.HandleFunc("GET "+APIPrefix+"/activities", r.auth.RequireAuth(r.h.Activity.List))
}
