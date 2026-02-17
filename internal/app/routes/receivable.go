package routes

func (r *Router) registerReceivableRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/receivables", r.auth.RequireAuth(r.h.Receivable.Create))
	r.mux.HandleFunc("GET "+APIPrefix+"/receivables", r.auth.RequireAuth(r.h.Receivable.List))
	r.mux.HandleFunc("GET "+APIPrefix+"/receivables/{uuid}/payments", r.auth.RequireAuth(r.h.Receivable.ListPayments))
	r.mux.HandleFunc("POST "+APIPrefix+"/receivables/{uuid}/payments", r.auth.RequireAuth(r.h.Receivable.RecordPayment))
	r.mux.HandleFunc("GET "+APIPrefix+"/receivables/{uuid}", r.auth.RequireAuth(r.h.Receivable.Get))
	r.mux.HandleFunc("PUT "+APIPrefix+"/receivables/{uuid}", r.auth.RequireAuth(r.h.Receivable.Update))
	r.mux.HandleFunc("DELETE "+APIPrefix+"/receivables/{uuid}", r.auth.RequireAuth(r.h.Receivable.Delete))
}
