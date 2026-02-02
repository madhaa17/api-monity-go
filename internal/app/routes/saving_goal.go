package routes

func (r *Router) registerSavingGoalRoutes() {
	r.mux.HandleFunc("POST "+APIPrefix+"/saving-goals", r.auth.RequireAuth(r.h.SavingGoal.Create))
	r.mux.HandleFunc("GET "+APIPrefix+"/saving-goals", r.auth.RequireAuth(r.h.SavingGoal.List))
	r.mux.HandleFunc("GET "+APIPrefix+"/saving-goals/{uuid}", r.auth.RequireAuth(r.h.SavingGoal.Get))
	r.mux.HandleFunc("PUT "+APIPrefix+"/saving-goals/{uuid}", r.auth.RequireAuth(r.h.SavingGoal.Update))
	r.mux.HandleFunc("DELETE "+APIPrefix+"/saving-goals/{uuid}", r.auth.RequireAuth(r.h.SavingGoal.Delete))
}
