package routes

func (r *Router) registerIncomeRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/incomes", r.auth.RequireAuth(r.h.Income.Create))
	r.mux.HandleFunc("GET "+APIPrefix+"/incomes", r.auth.RequireAuth(r.h.Income.List))
	r.mux.HandleFunc("GET "+APIPrefix+"/incomes/{uuid}", r.auth.RequireAuth(r.h.Income.Get))
	r.mux.HandleFunc("PUT "+APIPrefix+"/incomes/{uuid}", r.auth.RequireAuth(r.h.Income.Update))
	r.mux.HandleFunc("DELETE "+APIPrefix+"/incomes/{uuid}", r.auth.RequireAuth(r.h.Income.Delete))
}
