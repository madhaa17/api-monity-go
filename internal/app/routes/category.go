package routes

func (r *Router) registerCategoryRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/categories", r.auth.RequireAuth(r.h.Category.Create))
	r.mux.HandleFunc("GET "+APIPrefix+"/categories", r.auth.RequireAuth(r.h.Category.List))
	r.mux.HandleFunc("GET "+APIPrefix+"/categories/{uuid}", r.auth.RequireAuth(r.h.Category.Get))
	r.mux.HandleFunc("PUT "+APIPrefix+"/categories/{uuid}", r.auth.RequireAuth(r.h.Category.Update))
	r.mux.HandleFunc("DELETE "+APIPrefix+"/categories/{uuid}", r.auth.RequireAuth(r.h.Category.Delete))
}
