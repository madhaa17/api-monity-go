package app

import (
	"context"
	"log"
	"net/http"

	"monity/internal/adapter/handler"
	"monity/internal/adapter/middleware"
	"monity/internal/adapter/repository"
	"monity/internal/config"
	"monity/internal/core/service"

	"gorm.io/gorm"
)

type App struct {
	cfg                *config.Config
	db                 *gorm.DB
	srv                *http.Server
	authHandler        *handler.AuthHandler
	assetHandler       *handler.AssetHandler
	expenseHandler     *handler.ExpenseHandler
	incomeHandler      *handler.IncomeHandler
	savingGoalHandler  *handler.SavingGoalHandler
	authMiddleware     *middleware.AuthMiddleware
}

func New(ctx context.Context, cfg *config.Config, db *gorm.DB) *App {
	// Repositories
	userRepo := repository.NewUserRepository(db)
	assetRepo := repository.NewAssetRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	incomeRepo := repository.NewIncomeRepository(db)
	savingGoalRepo := repository.NewSavingGoalRepository(db)

	// Services
	authService := service.NewAuthService(userRepo, cfg)
	assetService := service.NewAssetService(assetRepo)
	expenseService := service.NewExpenseService(expenseRepo)
	incomeService := service.NewIncomeService(incomeRepo)
	savingGoalService := service.NewSavingGoalService(savingGoalRepo)

	// Handlers & Middleware
	authHandler := handler.NewAuthHandler(authService)
	assetHandler := handler.NewAssetHandler(assetService)
	expenseHandler := handler.NewExpenseHandler(expenseService)
	incomeHandler := handler.NewIncomeHandler(incomeService)
	savingGoalHandler := handler.NewSavingGoalHandler(savingGoalService)
	authMiddleware := middleware.NewAuthMiddleware(cfg)

	app := &App{
		cfg:               cfg,
		db:                db,
		authHandler:       authHandler,
		assetHandler:      assetHandler,
		expenseHandler:    expenseHandler,
		incomeHandler:     incomeHandler,
		savingGoalHandler: savingGoalHandler,
		authMiddleware:    authMiddleware,
	}
	app.srv = &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: app.routes(),
	}
	return app
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		sqlDB, err := a.db.DB()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","database":"error"}`))
			return
		}

		if err := sqlDB.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","database":"down"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","database":"connected"}`))
	})

	// Public Routes
	mux.HandleFunc("POST /auth/register", a.authHandler.Register)
	mux.HandleFunc("POST /auth/login", a.authHandler.Login)

	// Protected Routes (Assets)
	mux.HandleFunc("POST /assets", a.authMiddleware.RequireAuth(a.assetHandler.Create))
	mux.HandleFunc("GET /assets", a.authMiddleware.RequireAuth(a.assetHandler.List))
	mux.HandleFunc("GET /assets/{uuid}", a.authMiddleware.RequireAuth(a.assetHandler.Get))
	mux.HandleFunc("PUT /assets/{uuid}", a.authMiddleware.RequireAuth(a.assetHandler.Update))
	mux.HandleFunc("DELETE /assets/{uuid}", a.authMiddleware.RequireAuth(a.assetHandler.Delete))

	// Protected Routes (Expenses)
	mux.HandleFunc("POST /expenses", a.authMiddleware.RequireAuth(a.expenseHandler.Create))
	mux.HandleFunc("GET /expenses", a.authMiddleware.RequireAuth(a.expenseHandler.List))
	mux.HandleFunc("GET /expenses/{uuid}", a.authMiddleware.RequireAuth(a.expenseHandler.Get))
	mux.HandleFunc("PUT /expenses/{uuid}", a.authMiddleware.RequireAuth(a.expenseHandler.Update))
	mux.HandleFunc("DELETE /expenses/{uuid}", a.authMiddleware.RequireAuth(a.expenseHandler.Delete))

	// Protected Routes (Incomes)
	mux.HandleFunc("POST /incomes", a.authMiddleware.RequireAuth(a.incomeHandler.Create))
	mux.HandleFunc("GET /incomes", a.authMiddleware.RequireAuth(a.incomeHandler.List))
	mux.HandleFunc("GET /incomes/{uuid}", a.authMiddleware.RequireAuth(a.incomeHandler.Get))
	mux.HandleFunc("PUT /incomes/{uuid}", a.authMiddleware.RequireAuth(a.incomeHandler.Update))
	mux.HandleFunc("DELETE /incomes/{uuid}", a.authMiddleware.RequireAuth(a.incomeHandler.Delete))

	// Protected Routes (Saving Goals)
	mux.HandleFunc("POST /saving-goals", a.authMiddleware.RequireAuth(a.savingGoalHandler.Create))
	mux.HandleFunc("GET /saving-goals", a.authMiddleware.RequireAuth(a.savingGoalHandler.List))
	mux.HandleFunc("GET /saving-goals/{uuid}", a.authMiddleware.RequireAuth(a.savingGoalHandler.Get))
	mux.HandleFunc("PUT /saving-goals/{uuid}", a.authMiddleware.RequireAuth(a.savingGoalHandler.Update))
	mux.HandleFunc("DELETE /saving-goals/{uuid}", a.authMiddleware.RequireAuth(a.savingGoalHandler.Delete))

	return mux
}

func (a *App) Run() error {
	log.Printf("server listening on :%s (env=%s)", a.cfg.App.Port, a.cfg.App.Env)
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.srv != nil {
		return a.srv.Shutdown(ctx)
	}
	return nil
}
