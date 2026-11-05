package app

import (
	"context"
	"log/slog"
	"net/http"

	"monity/internal/adapter/handler"
	"monity/internal/adapter/middleware"
	"monity/internal/adapter/repository"
	"monity/internal/app/routes"
	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/core/service"
	"monity/internal/pkg/cache"

	"gorm.io/gorm"
)

type App struct {
	cfg *config.Config
	db  *gorm.DB
	srv *http.Server

	dailyCloses port.DailyCloseService
	// jobsCtx is cancelled by Shutdown to stop the background jobs.
	jobsCtx    context.Context
	cancelJobs context.CancelFunc
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
	debtRepo := repository.NewDebtRepository(db)
	debtPaymentRepo := repository.NewDebtPaymentRepository(db)
	receivableRepo := repository.NewReceivableRepository(db)
	receivablePaymentRepo := repository.NewReceivablePaymentRepository(db)
	assetPriceHistoryRepo := repository.NewAssetPriceHistoryRepository(db)
	insightRepo := repository.NewInsightRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	marketDailyCloseRepo := repository.NewMarketDailyCloseRepository(db)
	transactor := repository.NewTransactor(db)

	authSvc := service.NewAuthService(userRepo, categoryRepo, transactor, cfg, c)
	assetSvc := service.NewAssetService(assetRepo)
	activitySvc := service.NewActivityService(expenseRepo, incomeRepo, debtRepo, receivableRepo)
	expenseSvc := service.NewExpenseService(expenseRepo, assetRepo, categoryRepo, transactor)
	incomeSvc := service.NewIncomeService(incomeRepo, assetRepo, categoryRepo, transactor)
	categorySvc := service.NewCategoryService(categoryRepo, transactor)
	savingGoalSvc := service.NewSavingGoalService(savingGoalRepo)
	debtSvc := service.NewDebtService(debtRepo, debtPaymentRepo, assetRepo, transactor)
	receivableSvc := service.NewReceivableService(receivableRepo, receivablePaymentRepo, assetRepo, transactor)
	priceSvc := service.NewPriceService(&cfg.PriceAPI, c)
	assetPriceHistorySvc := service.NewAssetPriceHistoryService(assetPriceHistoryRepo, assetRepo, priceSvc)
	insightSvc := service.NewInsightService(insightRepo)
	portfolioSvc := service.NewPortfolioService(assetRepo, priceSvc, assetPriceHistoryRepo)
	performanceSvc := service.NewPerformanceService(assetRepo, priceSvc)
	dailyCloseSvc := service.NewDailyCloseService(marketDailyCloseRepo, priceSvc)

	authMiddleware := middleware.NewAuthMiddleware(cfg, c)

	handlers := &routes.Handlers{
		Auth:              handler.NewAuthHandler(authSvc),
		Asset:             handler.NewAssetHandler(assetSvc),
		Activity:          handler.NewActivityHandler(activitySvc),
		Expense:           handler.NewExpenseHandler(expenseSvc),
		Income:            handler.NewIncomeHandler(incomeSvc),
		Category:          handler.NewCategoryHandler(categorySvc),
		SavingGoal:        handler.NewSavingGoalHandler(savingGoalSvc),
		Debt:              handler.NewDebtHandler(debtSvc),
		Receivable:        handler.NewReceivableHandler(receivableSvc),
		Price:             handler.NewPriceHandler(priceSvc),
		AssetPriceHistory: handler.NewAssetPriceHistoryHandler(assetPriceHistorySvc),
		Insight:           handler.NewInsightHandler(insightSvc),
		Portfolio:         handler.NewPortfolioHandler(portfolioSvc),
		Performance:       handler.NewPerformanceHandler(performanceSvc),
	}

	router := routes.New(authMiddleware, handlers)
	mux := router.Setup()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
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
	var routed http.Handler = jsonNotFound(mux)
	chain := middleware.Gzip(
		middleware.RequestLogger(
			middleware.CORS(cfg.Security.CORSAllowedOrigins)(
				middleware.SecurityHeaders(
					middleware.BodyLimit(middleware.MaxBodyBytes)(rateLimit.Handler(routed)),
				),
			),
		),
	)

	jobsCtx, cancelJobs := context.WithCancel(context.WithoutCancel(ctx))
	app := &App{
		cfg: cfg,
		db:  db,
		srv: &http.Server{
			Addr:    ":" + cfg.App.Port,
			Handler: chain,
		},
		dailyCloses: dailyCloseSvc,
		jobsCtx:     jobsCtx,
		cancelJobs:  cancelJobs,
	}

	return app
}

func (a *App) Run() error {
	slog.Info("server listening", "port", a.cfg.App.Port, "env", a.cfg.App.Env)
	go a.dailyCloses.Run(a.jobsCtx)
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.cancelJobs()
	if a.srv != nil {
		return a.srv.Shutdown(ctx)
	}
	return nil
}
