package routes

func (r *Router) registerPriceRoutes() {
	r.mux.HandleFunc("GET "+APIPrefix+"/prices/search", r.h.Price.SearchSymbols)
	// Chart endpoints (more specific paths first)
	r.mux.HandleFunc("GET "+APIPrefix+"/prices/crypto/{symbol}/chart", r.h.Price.GetCryptoChart)
	r.mux.HandleFunc("GET "+APIPrefix+"/prices/stock/{symbol}/chart", r.h.Price.GetStockChart)
	r.mux.HandleFunc("GET "+APIPrefix+"/prices/crypto/{symbol}", r.h.Price.GetCryptoPrice)
	r.mux.HandleFunc("GET "+APIPrefix+"/prices/stock/{symbol}", r.h.Price.GetStockPrice)
}
