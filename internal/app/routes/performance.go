package routes

func (r *Router) registerPerformanceRoutes() {
	r.mux.HandleFunc("GET "+APIPrefix+"/assets/{uuid}/performance", r.auth.RequireAuth(r.h.Performance.GetAssetPerformance))
	r.mux.HandleFunc("GET "+APIPrefix+"/portfolio/performance", r.auth.RequireAuth(r.h.Performance.GetPortfolioPerformance))
}
