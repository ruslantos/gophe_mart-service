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
	"github.com/ruslantos/gophemart-service/internal/handlers/getorders"
	"github.com/ruslantos/gophemart-service/internal/handlers/getuserbalance"
	"github.com/ruslantos/gophemart-service/internal/handlers/getuserwithdrawals"
	login "github.com/ruslantos/gophemart-service/internal/handlers/login"
	"github.com/ruslantos/gophemart-service/internal/handlers/postorders"
	"github.com/ruslantos/gophemart-service/internal/handlers/register"
	"github.com/ruslantos/gophemart-service/internal/handlers/userbalancewithdraw"
	"github.com/ruslantos/gophemart-service/internal/logger"
	authMiddlware "github.com/ruslantos/gophemart-service/internal/middlware/auth"
	loggerMiddleware "github.com/ruslantos/gophemart-service/internal/middlware/logger"
	"github.com/ruslantos/gophemart-service/internal/repository"
	gophemartService "github.com/ruslantos/gophemart-service/internal/service"
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
	service := gophemartService.NewService(userRepo, loyaltyClient)

	// Старт воркера
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.StartWorker(ctx)
	defer service.StopWorker()

	r := setupRouter(service)

	err = http.ListenAndServe(cfg.RunAddress, r)
	if err != nil {
		logger.Get().Fatal("cannot start server", zap.Error(err))
	}
}

func setupRouter(service *gophemartService.Service) *chi.Mux {
	log, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("cannot create logger", zap.Error(err))
	}

	registerHandler := register.NewUserHandler(service)
	loginHandler := login.NewHandler(service)
	postordersHandler := postorders.NewHandler(service)
	getordersHandler := getorders.NewHandler(service)
	getUserbalanceHandler := getuserbalance.NewHandler(service)
	userbalancewithdrawHandler := userbalancewithdraw.NewHandler(service)
	getuserwithdrawalsHandler := getuserwithdrawals.NewHandler(service)

	r := chi.NewRouter()
	r.Use(authMiddlware.AuthMiddleware(service), loggerMiddleware.Logger(log))

	r.Post("/api/user/register", registerHandler.Handle)
	r.Post("/api/user/login", loginHandler.Handle)
	r.Post("/api/user/orders", postordersHandler.Handle)
	r.Get("/api/user/orders", getordersHandler.Handle)
	r.Get("/api/user/balance", getUserbalanceHandler.Handle)
	r.Post("/api/user/balance/withdraw", userbalancewithdrawHandler.Handle)
	r.Get("/api/user/withdrawals", getuserwithdrawalsHandler.Handle)

	return r
}
