package routes

func (r *Router) registerInsightRoutes() {
	r.mux.HandleFunc("GET "+APIPrefix+"/insights/cashflow", r.auth.RequireAuth(r.h.Insight.GetCashflow))
	r.mux.HandleFunc("GET "+APIPrefix+"/insights/overview", r.auth.RequireAuth(r.h.Insight.GetOverview))
}
