package routes

func (r *Router) registerAssetRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/assets", r.auth.RequireAuth(r.h.Asset.Create))
	r.mux.HandleFunc("GET "+APIPrefix+"/assets", r.auth.RequireAuth(r.h.Asset.List))
	r.mux.HandleFunc("GET "+APIPrefix+"/assets/{uuid}", r.auth.RequireAuth(r.h.Asset.Get))
	r.mux.HandleFunc("PUT "+APIPrefix+"/assets/{uuid}", r.auth.RequireAuth(r.h.Asset.Update))
	r.mux.HandleFunc("DELETE "+APIPrefix+"/assets/{uuid}", r.auth.RequireAuth(r.h.Asset.Delete))

	r.mux.HandleFunc("GET "+APIPrefix+"/assets/{uuid}/prices", r.auth.RequireAuth(r.h.AssetPriceHistory.GetPriceHistory))
	r.mux.HandleFunc("POST "+APIPrefix+"/assets/{uuid}/prices", r.auth.RequireAuth(r.h.AssetPriceHistory.RecordPrice))
	r.mux.HandleFunc("POST "+APIPrefix+"/assets/{uuid}/prices/fetch", r.auth.RequireAuth(r.h.AssetPriceHistory.FetchAndRecordPrice))
}
