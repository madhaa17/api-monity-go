package routes

func (r *Router) registerPriceRoutes() {
	r.mux.HandleFunc("GET "+APIPrefix+"/prices/crypto/{symbol}", r.h.Price.GetCryptoPrice)
	r.mux.HandleFunc("GET "+APIPrefix+"/prices/stock/{symbol}", r.h.Price.GetStockPrice)
}
