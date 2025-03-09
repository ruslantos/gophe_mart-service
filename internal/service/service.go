package service

import (
	"context"
	"errors"
	"sync"
	"time"

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
	SaveOrder(ctx context.Context, order model.Order) error
}

type Service struct {
	repo      repo
	client    *clients.LoyaltyClient
	orderChan chan string
	wg        sync.WaitGroup
}

func NewService(repo repo, client *clients.LoyaltyClient) *Service {
	return &Service{
		repo:      repo,
		client:    client,
		orderChan: make(chan string, 100),
		wg:        sync.WaitGroup{},
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
			case orderNumber := <-s.orderChan:
				s.processOrder(ctx, orderNumber)
			case <-ctx.Done():
				logger.Get().Info("Worker stopped")
				return
			}
		}
	}()
}
func (s *Service) processOrder(ctx context.Context, orderNumber string) {
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
				// todo возможно надо проставить стаус INVALID
				return
			}
			logger.Get().Info("Order processed:", zap.String("orderNumber", orderNumber), zap.String("status", orderInfo.Status), zap.Float64("accrual", orderInfo.Accrual))

			order := model.Order{
				OrderID: orderInfo.Order,
			}

			// сохраняем предварительный результат и запрашиваем дальше
			if orderInfo.Status == model.STATE_REGISTERED || orderInfo.Status == model.STATE_PROCESSING {
				order.Status = orderInfo.Status
				err := s.repo.SaveOrder(ctx, order)
				if err != nil {
					logger.Get().Error("Failed to save order", zap.String("orderNumber", orderNumber), zap.Error(err))
					return
				}
				continue
			}

			//сохраняем результат расчета баллов и выходим
			if orderInfo.Status == model.STATE_PROCESSED || orderInfo.Status == model.STATE_INVALID {
				order.Status = orderInfo.Status
				order.Accrual = orderInfo.Accrual

				err := s.repo.SaveOrder(ctx, order)
				if err != nil {
					logger.Get().Error("Failed to save order", zap.String("orderNumber", orderNumber), zap.Error(err))
					return
				}
				return
			}

			time.Sleep(1 * time.Second)
		}
	}
}
func (s *Service) StopWorker() {
	close(s.orderChan)
	s.wg.Wait()
}
func (s *Service) SendOrderToLoyaltyClient(orderNumber string) {
	s.orderChan <- orderNumber
}
