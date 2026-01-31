package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monity/internal/app"
	"monity/internal/config"
	"monity/internal/database"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.Database.User == "" || cfg.Database.Name == "" {
		log.Fatal("DATABASE_USER and DATABASE_NAME must be set (e.g. in .env)")
	}

	db, err := database.NewDB(ctx, &cfg.Database)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	// GORM's generic DB interface doesn't have a Close method directly on *gorm.DB,
	// but the underlying sql.DB does. We'll handle cleanup in Shutdown or main.
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get sql db: %v", err)
	}
	defer sqlDB.Close()

	application := app.New(ctx, cfg, db)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := application.Run(); err != nil && err != context.Canceled {
			log.Printf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = application.Shutdown(shutdownCtx)
	log.Println("done")
}
