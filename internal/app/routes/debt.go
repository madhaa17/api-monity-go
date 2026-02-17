package routes

func (r *Router) registerDebtRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/debts", r.auth.RequireAuth(r.h.Debt.Create))
	r.mux.HandleFunc("GET "+APIPrefix+"/debts", r.auth.RequireAuth(r.h.Debt.List))
	r.mux.HandleFunc("GET "+APIPrefix+"/debts/{uuid}/payments", r.auth.RequireAuth(r.h.Debt.ListPayments))
	r.mux.HandleFunc("POST "+APIPrefix+"/debts/{uuid}/payments", r.auth.RequireAuth(r.h.Debt.RecordPayment))
	r.mux.HandleFunc("GET "+APIPrefix+"/debts/{uuid}", r.auth.RequireAuth(r.h.Debt.Get))
	r.mux.HandleFunc("PUT "+APIPrefix+"/debts/{uuid}", r.auth.RequireAuth(r.h.Debt.Update))
	r.mux.HandleFunc("DELETE "+APIPrefix+"/debts/{uuid}", r.auth.RequireAuth(r.h.Debt.Delete))
}
