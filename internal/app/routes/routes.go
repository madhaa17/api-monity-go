package routes

import (
	"net/http"

	"monity/internal/adapter/handler"
	"monity/internal/adapter/middleware"
)

const APIPrefix = "/api/v1"

type Handlers struct {
	Auth              *handler.AuthHandler
	Asset             *handler.AssetHandler
	Expense           *handler.ExpenseHandler
	Income            *handler.IncomeHandler
	SavingGoal        *handler.SavingGoalHandler
	Price             *handler.PriceHandler
	AssetPriceHistory *handler.AssetPriceHistoryHandler
	Insight           *handler.InsightHandler
	Portfolio         *handler.PortfolioHandler
	Performance       *handler.PerformanceHandler
}

type Router struct {
	mux  *http.ServeMux
	auth *middleware.AuthMiddleware
	h    *Handlers
}

func New(auth *middleware.AuthMiddleware, h *Handlers) *Router {
	return &Router{
		mux:  http.NewServeMux(),
		auth: auth,
		h:    h,
	}
}

func (r *Router) Setup() *http.ServeMux {
	r.registerAuthRoutes()
	r.registerAssetRoutes()
	r.registerExpenseRoutes()
	r.registerIncomeRoutes()
	r.registerSavingGoalRoutes()
	r.registerPriceRoutes()
	r.registerInsightRoutes()
	r.registerPortfolioRoutes()
	r.registerPerformanceRoutes()
	return r.mux
}

func (r *Router) Mux() *http.ServeMux {
	return r.mux
}
