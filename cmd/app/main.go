package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kareempaes/planning/internal/handler"
	"github.com/kareempaes/planning/internal/infra"
	"github.com/kareempaes/planning/internal/repo"
	"github.com/kareempaes/planning/internal/service"
)

func main() {
	cfg := LoadConfig()
	ctx := context.Background()

	// 1. Database
	driverType := infra.SQLite
	driverName := "sqlite"
	if cfg.DBDriver == "pgx" || cfg.DBDriver == "postgres" {
		driverType = infra.Postgres
		driverName = "pgx"
	}

	db, err := infra.NewDB(ctx, driverType, cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// 2. Migrations
	if err := infra.RunMigrations(db, driverName, cfg.MigrationsPath); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// 3. Repositories
	store, err := repo.NewStore(repo.SQLStore, db)
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}

	// 4. Services
	authCfg := service.AuthConfig{
		JWTSecret:          cfg.JWTSecret,
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	registry, err := service.NewRegistry(service.DefaultRegistry, store, authCfg)
	if err != nil {
		log.Fatalf("failed to create service registry: %v", err)
	}

	// 5. WebSocket Hub
	hub := infra.NewHub()
	go hub.Run()

	// 6. Router
	router := handler.NewRouter(registry, hub, cfg.JWTSecret)

	// 7. HTTP Server
	srv := &http.Server{
		Addr:        ":" + cfg.Port,
		Handler:     router,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server exited")
}
