package app

import (
	"context"
	"log"
	"net/http"

	"monity/internal/adapter/handler"
	"monity/internal/adapter/middleware"
	"monity/internal/adapter/repository"
	"monity/internal/app/routes"
	"monity/internal/config"
	"monity/internal/core/service"
	"monity/internal/pkg/cache"

	"gorm.io/gorm"
)

type App struct {
	cfg *config.Config
	db  *gorm.DB
	srv *http.Server
}

func New(ctx context.Context, cfg *config.Config, db *gorm.DB, c cache.Cache) *App {
	if c == nil {
		c = cache.NewMemoryCache()
	}
	userRepo := repository.NewUserRepository(db)
	assetRepo := repository.NewAssetRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	incomeRepo := repository.NewIncomeRepository(db)
	savingGoalRepo := repository.NewSavingGoalRepository(db)
	assetPriceHistoryRepo := repository.NewAssetPriceHistoryRepository(db)
	insightRepo := repository.NewInsightRepository(db)

	authSvc := service.NewAuthService(userRepo, cfg)
	assetSvc := service.NewAssetService(assetRepo)
	expenseSvc := service.NewExpenseService(expenseRepo)
	incomeSvc := service.NewIncomeService(incomeRepo)
	savingGoalSvc := service.NewSavingGoalService(savingGoalRepo)
	priceSvc := service.NewPriceService(&cfg.PriceAPI, c)
	assetPriceHistorySvc := service.NewAssetPriceHistoryService(assetPriceHistoryRepo, assetRepo, priceSvc)
	insightSvc := service.NewInsightService(insightRepo)
	portfolioSvc := service.NewPortfolioService(assetRepo, priceSvc, assetPriceHistoryRepo)
	performanceSvc := service.NewPerformanceService(assetRepo, priceSvc)

	authMiddleware := middleware.NewAuthMiddleware(cfg)

	handlers := &routes.Handlers{
		Auth:              handler.NewAuthHandler(authSvc),
		Asset:             handler.NewAssetHandler(assetSvc),
		Expense:           handler.NewExpenseHandler(expenseSvc),
		Income:            handler.NewIncomeHandler(incomeSvc),
		SavingGoal:        handler.NewSavingGoalHandler(savingGoalSvc),
		Price:             handler.NewPriceHandler(priceSvc),
		AssetPriceHistory: handler.NewAssetPriceHistoryHandler(assetPriceHistorySvc),
		Insight:           handler.NewInsightHandler(insightSvc),
		Portfolio:         handler.NewPortfolioHandler(portfolioSvc),
		Performance:       handler.NewPerformanceHandler(performanceSvc),
	}

	router := routes.New(authMiddleware, handlers)
	mux := router.Setup()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		sqlDB, err := db.DB()
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

		w.Write([]byte(`{"status":"ok","database":"connected"}`))
	})

	rateLimit := middleware.NewRateLimitMiddleware(&cfg.RateLimit)
	chain := middleware.CORS(cfg.Security.CORSAllowedOrigins)(
		middleware.SecurityHeaders(
			rateLimit.Handler(mux),
		),
	)

	app := &App{
		cfg: cfg,
		db:  db,
		srv: &http.Server{
			Addr:    ":" + cfg.App.Port,
			Handler: chain,
		},
	}

	return app
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
