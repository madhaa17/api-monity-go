package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monity/internal/app"
	"monity/internal/config"
	"monity/internal/database"
	"monity/internal/pkg/cache"
	"monity/internal/pkg/logger"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	_ = logger.New(cfg.App.Env)

	if cfg.Database.User == "" || cfg.Database.Name == "" {
		slog.Error("DATABASE_USER and DATABASE_NAME must be set (e.g. in .env)")
		os.Exit(1)
	}

	db, err := database.NewDB(ctx, &cfg.Database)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	// GORM's generic DB interface doesn't have a Close method directly on *gorm.DB,
	// but the underlying sql.DB does. We'll handle cleanup in Shutdown or main.
	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("get sql db", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	var c cache.Cache
	if cfg.Redis.Enabled() {
		rdb, err := database.NewRedis(ctx, &cfg.Redis)
		if err != nil {
			slog.Error("redis", "error", err)
			os.Exit(1)
		}
		defer rdb.Close()
		c = cache.NewRedisCache(rdb)
		slog.Info("redis: connected, using Redis cache for prices")
	} else {
		c = cache.NewMemoryCache()
		slog.Info("redis: not configured, using in-memory cache for prices")
	}

	application := app.New(ctx, cfg, db, c)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := application.Run(); err != nil && err != context.Canceled {
			slog.Error("server error", "error", err)
		}
	}()

	<-quit
	slog.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = application.Shutdown(shutdownCtx)
	slog.Info("done")
}
