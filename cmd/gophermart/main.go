package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/clients"
	"github.com/ruslantos/gophemart-service/internal/config"
	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/repository"
	"github.com/ruslantos/gophemart-service/internal/router"
	gophemartService "github.com/ruslantos/gophemart-service/internal/service"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}

	db, err := sqlx.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		logger.Get().Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		logger.Get().Fatal("Failed to ping database", zap.Error(err))
	}

	// Инициализация клиента
	loyaltyClient := clients.NewLoyaltyClient(cfg.AccrualURL)

	// Инициализация репозитория
	userRepo := repository.NewUserRepository(db)
	if err = userRepo.InitStorage(); err != nil {
		logger.Get().Fatal("Failed to initialize storage for user", zap.Error(err))
	}

	// Инициализация сервиса
	service := gophemartService.NewService(userRepo, loyaltyClient)

	// Старт воркера
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.StartWorker(ctx)
	defer service.StopWorker()

	r := router.SetupRouter(service)

	if err = http.ListenAndServe(cfg.RunAddress, r); err != nil {
		logger.Get().Fatal("cannot start server", zap.Error(err))
	}
}
