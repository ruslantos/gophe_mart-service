package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/clients"
	internalErrors "github.com/ruslantos/gophemart-service/internal/errors"
	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/model"
)

type repo interface {
	CreateUser(ctx context.Context, login, password string) error
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	GetOrder(ctx context.Context, orderID string) (model.Order, error)
	GetOrders(ctx context.Context, userID string) ([]model.Order, error)
	GetOrdersByStates(ctx context.Context, states []string) ([]model.Order, error)
	SaveOrder(ctx context.Context, order model.Order) error
	UpdateUserAccrualSum(ctx context.Context, order model.Order) error
	GetUserAccrualSum(ctx context.Context, userID string) (model.UserBalance, error)
	Withdraw(ctx context.Context, withdrawal model.Withdrawal) error
	GetWithdrawalsByUserID(ctx context.Context, userID string) ([]model.Withdrawal, error)
}

type Service struct {
	repo      repo
	client    *clients.LoyaltyClient
	orderChan chan model.Order
	wg        sync.WaitGroup
	Cron      *cron.Cron
}

func NewService(repo repo, client *clients.LoyaltyClient) *Service {
	return &Service{
		repo:      repo,
		client:    client,
		orderChan: make(chan model.Order, 100),
		wg:        sync.WaitGroup{},
		Cron:      cron.New(cron.WithSeconds()),
	}
}

func (s *Service) Register(ctx context.Context, login, password string) error {
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
func (s *Service) Authenticate(ctx context.Context, login, password string) bool {
	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil || user == nil {
		return false
	}
	return user.Password == password
}
func (s *Service) GetOrder(ctx context.Context, orderNumber string) (model.Order, error) {
	order, err := s.repo.GetOrder(ctx, orderNumber)
	if err != nil {
		return model.Order{}, err
	}
	return order, nil
}
func (s *Service) GetOrders(ctx context.Context, userID string) ([]model.Order, error) {
	order, err := s.repo.GetOrders(ctx, userID)
	if err != nil {
		return []model.Order{}, err
	}
	return order, nil
}
func (s *Service) SaveOrder(ctx context.Context, order model.Order) error {
	err := s.repo.SaveOrder(ctx, order)
	if err != nil {
		return err
	}
	return nil
}

// StartWorker воркер отправки заказов в сервис loyalty
func (s *Service) StartWorker(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case order := <-s.orderChan:
				s.processOrder(ctx, order)
			case <-ctx.Done():
				logger.Get().Info("Worker stopped")
				return
			}
		}
	}()
}
func (s *Service) processOrder(ctx context.Context, order model.Order) {
	for {
		select {
		case <-ctx.Done():
			logger.Get().Error("Order processing timeout", zap.String("orderNumber", order.OrderID))
			return
		default:
			orderInfo, err := s.client.GetOrderInfo(ctx, order.OrderID)
			if err != nil {
				if errors.Is(err, clients.ErrTooManyRequests) {
					time.Sleep(orderInfo.RetryAfter * time.Second)
					continue
				}
				logger.Get().Error("Failed to process order", zap.String("orderNumber", order.OrderID), zap.Error(err))
				return
			}
			logger.Get().Info("Order processed:",
				zap.String("orderNumber", order.OrderID),
				zap.String("status", orderInfo.Status),
				zap.Float64("accrual", orderInfo.Accrual))

			order.Status = orderInfo.Status
			order.Accrual = orderInfo.Accrual
			if err = s.repo.SaveOrder(ctx, order); err != nil {
				logger.Get().Error("Failed to save order", zap.String("orderNumber", order.OrderID), zap.Error(err))
				return
			}

			time.Sleep(1 * time.Second)
			return
		}
	}
}
func (s *Service) StopWorker() {
	close(s.orderChan)
	s.wg.Wait()
}
func (s *Service) SendOrderToLoyaltyClient() {
	ctx := context.Background()
	orders, err := s.repo.GetOrdersByStates(ctx, []string{
		model.StateNew,
		model.StateRegistered,
		model.StateProcessing,
	})
	if err != nil {
		logger.Get().Error("Failed to get orders", zap.Error(err))
		return
	}
	for _, order := range orders {
		s.orderChan <- order
	}
}

func (s *Service) GetUserBalance(ctx context.Context, userID string) (model.UserBalance, error) {
	return s.repo.GetUserAccrualSum(ctx, userID)
}

func (s *Service) Withdraw(ctx context.Context, withdrawal model.Withdrawal) error {
	return s.repo.Withdraw(ctx, withdrawal)
}

func (s *Service) GetWithdrawalsByUserID(ctx context.Context, userID string) ([]model.Withdrawal, error) {
	return s.repo.GetWithdrawalsByUserID(ctx, userID)
}
