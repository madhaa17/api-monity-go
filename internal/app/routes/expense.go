package routes

func (r *Router) registerExpenseRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/expenses", r.auth.RequireAuth(r.h.Expense.Create))
	r.mux.HandleFunc("GET "+APIPrefix+"/expenses", r.auth.RequireAuth(r.h.Expense.List))
	r.mux.HandleFunc("GET "+APIPrefix+"/expenses/{uuid}", r.auth.RequireAuth(r.h.Expense.Get))
	r.mux.HandleFunc("PUT "+APIPrefix+"/expenses/{uuid}", r.auth.RequireAuth(r.h.Expense.Update))
	r.mux.HandleFunc("DELETE "+APIPrefix+"/expenses/{uuid}", r.auth.RequireAuth(r.h.Expense.Delete))
}
