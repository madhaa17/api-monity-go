package routes

func (r *Router) registerPortfolioRoutes() {
	r.mux.HandleFunc("GET "+APIPrefix+"/portfolio", r.auth.RequireAuth(r.h.Portfolio.GetPortfolio))
	r.mux.HandleFunc("GET "+APIPrefix+"/portfolio/assets/{uuid}", r.auth.RequireAuth(r.h.Portfolio.GetAssetValue))
}
