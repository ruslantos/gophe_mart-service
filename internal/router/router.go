package router

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/handlers/balance"
	login "github.com/ruslantos/gophemart-service/internal/handlers/login"
	"github.com/ruslantos/gophemart-service/internal/handlers/orders"
	"github.com/ruslantos/gophemart-service/internal/handlers/register"
	authMiddlware "github.com/ruslantos/gophemart-service/internal/middlware/auth"
	loggerMiddleware "github.com/ruslantos/gophemart-service/internal/middlware/logger"
	gophemartService "github.com/ruslantos/gophemart-service/internal/service"
)

func SetupRouter(service *gophemartService.Service) *chi.Mux {
	log, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("cannot create logger", zap.Error(err))
	}

	registerHandler := register.NewUserHandler(service)
	loginHandler := login.NewHandler(service)
	postOrders := orders.NewPostHandler(service)
	getOrders := orders.NewGetHandler(service)
	postWithdraw := balance.NewWithdrawHandler(service)
	getBalance := balance.NewGetHandler(service)
	getWithdrawals := balance.NewGetWithdrawalsHandler(service)

	r := chi.NewRouter()
	r.Use(authMiddlware.AuthMiddleware(service), loggerMiddleware.Logger(log))

	r.Post("/api/user/register", registerHandler.Handle)
	r.Post("/api/user/login", loginHandler.Handle)
	r.Post("/api/user/orders", postOrders.Handle)
	r.Get("/api/user/orders", getOrders.Handle)
	r.Post("/api/user/balance/withdraw", postWithdraw.Handle)
	r.Get("/api/user/balance", getBalance.Handle)
	r.Get("/api/user/withdrawals", getWithdrawals.Handle)

	return r
}
