package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/clients"
	"github.com/ruslantos/gophemart-service/internal/config"
	login "github.com/ruslantos/gophemart-service/internal/handlers/login"
	"github.com/ruslantos/gophemart-service/internal/handlers/postorders"
	"github.com/ruslantos/gophemart-service/internal/handlers/register"
	"github.com/ruslantos/gophemart-service/internal/logger"
	authMiddlware "github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/repository"
	"github.com/ruslantos/gophemart-service/internal/service"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}

	var db *sqlx.DB
	db, err = sqlx.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		logger.Get().Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		logger.Get().Fatal("Failed to ping database", zap.Error(err))
	}

	// Инициализация клиента
	loyaltyClient := clients.NewLoyaltyClient(cfg.AccrualURL)

	// Инициализация репозитория
	userRepo := repository.NewUserRepository(db)
	err = userRepo.InitStorage()

	// Инициализация сервиса
	userService := service.NewUserService(userRepo, loyaltyClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	userService.StartWorker(ctx)
	defer userService.StopWorker()

	r := setupRouter(userService)

	err = http.ListenAndServe(cfg.RunAddress, r)
	if err != nil {
		logger.Get().Fatal("cannot start server", zap.Error(err))
	}
}

func setupRouter(userService *service.UserService) *chi.Mux {
	registerHandler := register.NewUserHandler(userService)
	loginHandler := login.NewHandler(userService)
	postordersHandler := postorders.NewHandler(userService)

	r := chi.NewRouter()
	r.Use(authMiddlware.AuthMiddleware(userService))

	r.Post("/api/user/register", registerHandler.Handle)
	r.Post("/api/user/login", loginHandler.Handle)
	r.Post("/api/user/orders", postordersHandler.Handle)

	return r
}
