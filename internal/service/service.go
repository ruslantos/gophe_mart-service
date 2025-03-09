package service

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/clients"
	internalErrors "github.com/ruslantos/gophemart-service/internal/errors"
	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/models"
	"github.com/ruslantos/gophemart-service/internal/repository"
)

type repo interface {
	GetOrder(ctx context.Context, orderID string) (models.Order, error)
}

type UserService struct {
	repo      *repository.UserRepository
	client    *clients.LoyaltyClient
	orderChan chan string
	wg        sync.WaitGroup
}

func NewUserService(repo *repository.UserRepository, client *clients.LoyaltyClient) *UserService {
	return &UserService{
		repo:      repo,
		client:    client,
		orderChan: make(chan string, 100),
		wg:        sync.WaitGroup{},
	}
}

func (s *UserService) Register(ctx context.Context, login, password string) error {
	err := s.repo.CreateUser(ctx, login, password)
	if err != nil {
		if err == internalErrors.ErrLoginAlreadyExists {
			return err
		}
		logger.Get().Error("Failed to register user", zap.Error(err))
		return errors.New("internal server error")
	}
	return nil
}
func (s *UserService) Authenticate(ctx context.Context, login, password string) bool {
	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil || user == nil {
		return false
	}
	return user.Password == password
}
func (s *UserService) GetOrder(ctx context.Context, orderNumber string) (models.Order, error) {
	order, err := s.repo.GetOrder(ctx, orderNumber)
	if err != nil {
		return models.Order{}, err
	}
	return order, nil
}

// воркер отправки заказов в сервис loyalty
func (s *UserService) StartWorker(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case orderNumber := <-s.orderChan:
				s.processOrder(ctx, orderNumber)
			case <-ctx.Done():
				logger.Get().Info("Worker stopped")
				return
			}
		}
	}()
}
func (s *UserService) processOrder(ctx context.Context, orderNumber string) {
	orderCtx, cancel := context.WithTimeout(ctx, 5*time.Minute) // Таймаут 5 минут
	defer cancel()
	for {
		select {
		case <-orderCtx.Done():
			logger.Get().Error("Order processing timeout", zap.String("orderNumber", orderNumber))
			return
		default:
			orderInfo, err := s.client.GetOrderInfo(orderNumber)
			if err != nil {
				if errors.Is(err, clients.ErrTooManyRequests) {
					time.Sleep(1 * time.Second)
					continue
				}
				logger.Get().Error("Failed to process order", zap.String("orderNumber", orderNumber), zap.Error(err))
				return
			}
			logger.Get().Info("Order processed", zap.String("orderNumber", orderNumber), zap.String("status", orderInfo.Status), zap.Int64("accrual", orderInfo.Accrual))

			//сохраняем результат расчета баллов
			if orderInfo.Status == "PROCESSED" || orderInfo.Status == "INVALID" {
				order := models.Order{
					OrderID: orderInfo.Order,
					Status:  orderInfo.Status,
					Accrual: orderInfo.Accrual,
				}
				if err := s.repo.SaveOrder(ctx, order); err != nil {
					log.Printf("Failed to save order %s: %v\n", orderInfo.Order, err)
				}
				return
			}

			time.Sleep(1 * time.Second)
		}
	}
}
func (s *UserService) StopWorker() {
	close(s.orderChan)
	s.wg.Wait()
}
func (s *UserService) SendOrderToLoyaltyClient(orderNumber string) {
	s.orderChan <- orderNumber
}
