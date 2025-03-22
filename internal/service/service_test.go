package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ruslantos/gophemart-service/internal/clients"
	internal_errors "github.com/ruslantos/gophemart-service/internal/errors"
	"github.com/ruslantos/gophemart-service/internal/model"
)

func TestService_Register_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	login := "testuser"
	password := "testpass"

	mockRepo.EXPECT().CreateUser(ctx, login, password).Return(nil)
	err := s.Register(ctx, login, password)
	assert.NoError(t, err)
}

func TestService_Register_LoginAlreadyExists(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	login := "testuser"
	password := "testpass"

	mockRepo.EXPECT().CreateUser(ctx, login, password).Return(internal_errors.ErrLoginAlreadyExists)
	err := s.Register(ctx, login, password)
	assert.ErrorIs(t, err, internal_errors.ErrLoginAlreadyExists)
}

func TestService_Register_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	login := "testuser"
	password := "testpass"

	mockRepo.EXPECT().CreateUser(ctx, login, password).Return(errors.New("internal error"))
	err := s.Register(ctx, login, password)
	assert.Error(t, err)
}

func TestService_Authenticate_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	login := "testuser"
	password := "testpass"

	mockRepo.EXPECT().GetUserByLogin(ctx, login).Return(&model.User{Login: login, Password: password}, nil)
	result := s.Authenticate(ctx, login, password)
	assert.True(t, result)
}

func TestService_Authenticate_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	login := "testuser"
	password := "testpass"

	mockRepo.EXPECT().GetUserByLogin(ctx, login).Return(nil, errors.New("internal error"))
	result := s.Authenticate(ctx, login, password)
	assert.False(t, result)
}

func TestService_Authenticate_WrongPassword(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	login := "testuser"
	password := "testpass"

	mockRepo.EXPECT().GetUserByLogin(ctx, login).Return(&model.User{Login: login, Password: "wrongpass"}, nil)
	result := s.Authenticate(ctx, login, password)
	assert.False(t, result)
}

func TestService_GetOrder_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	orderNumber := "12345"

	mockRepo.EXPECT().GetOrder(ctx, orderNumber).Return(model.Order{OrderID: orderNumber}, nil)
	order, err := s.GetOrder(ctx, orderNumber)
	assert.NoError(t, err)
	assert.Equal(t, orderNumber, order.OrderID)
}

func TestService_GetOrder_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	orderNumber := "12345"

	mockRepo.EXPECT().GetOrder(ctx, orderNumber).Return(model.Order{}, errors.New("internal error"))
	_, err := s.GetOrder(ctx, orderNumber)
	assert.Error(t, err)
}

func TestService_GetOrders_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	userID := "12345"

	mockRepo.EXPECT().GetOrders(ctx, userID).Return([]model.Order{{OrderID: "12345"}}, nil)
	orders, err := s.GetOrders(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
}

func TestService_GetOrders_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	userID := "12345"

	mockRepo.EXPECT().GetOrders(ctx, userID).Return([]model.Order{}, errors.New("internal error"))
	_, err := s.GetOrders(ctx, userID)
	assert.Error(t, err)
}

func TestService_SaveOrder_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	order := model.Order{OrderID: "12345"}

	mockRepo.EXPECT().SaveOrder(ctx, order).Return(nil)
	err := s.SaveOrder(ctx, order)
	assert.NoError(t, err)
}

func TestService_SaveOrder_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	order := model.Order{OrderID: "12345"}

	mockRepo.EXPECT().SaveOrder(ctx, order).Return(errors.New("internal error"))
	err := s.SaveOrder(ctx, order)
	assert.Error(t, err)
}

func TestService_processOrder_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	order := model.Order{OrderID: "12345"}

	mockClient.EXPECT().GetOrderInfo(ctx, order.OrderID).Return(&clients.OrderResponse{Status: "PROCESSED", Accrual: 10.0}, nil)
	mockRepo.EXPECT().SaveOrder(ctx, mock.Anything).Return(nil)
	s.processOrder(ctx, order)
}

func TestService_processOrder_TooManyRequests(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	order := model.Order{OrderID: "12345"}

	mockClient.EXPECT().GetOrderInfo(ctx, order.OrderID).Return(&clients.OrderResponse{RetryAfter: 1}, clients.ErrTooManyRequests).Once()
	mockClient.EXPECT().GetOrderInfo(ctx, order.OrderID).Return(&clients.OrderResponse{Status: "PROCESSED", Accrual: 10.0}, nil)
	mockRepo.EXPECT().SaveOrder(ctx, mock.Anything).Return(nil)
	s.processOrder(ctx, order)
}

func TestService_processOrder_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	order := model.Order{OrderID: "12345"}

	mockClient.EXPECT().GetOrderInfo(ctx, order.OrderID).Return(nil, errors.New("internal error"))
	s.processOrder(ctx, order)
}

func TestService_SendOrderToLoyaltyClient_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()

	mockRepo.EXPECT().GetOrdersByStates(ctx, []string{"NEW", "REGISTERED", "PROCESSING"}).Return([]model.Order{{OrderID: "12345"}}, nil)
	s.SendOrderToLoyaltyClient()
}

func TestService_SendOrderToLoyaltyClient_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()

	mockRepo.EXPECT().GetOrdersByStates(ctx, []string{"NEW", "REGISTERED", "PROCESSING"}).Return([]model.Order{}, errors.New("internal error"))
	s.SendOrderToLoyaltyClient()
}

func TestService_GetUserBalance_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	userID := "12345"

	mockRepo.EXPECT().GetUserAccrualSum(ctx, userID).Return(model.UserBalance{Withdrawn: 11.0, Current: 10.0}, nil)
	balance, err := s.GetUserBalance(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 11.0, balance.Withdrawn)
}

func TestService_GetUserBalance_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	userID := "12345"

	mockRepo.EXPECT().GetUserAccrualSum(ctx, userID).Return(model.UserBalance{}, errors.New("internal error"))
	_, err := s.GetUserBalance(ctx, userID)
	assert.Error(t, err)
}

func TestService_Withdraw_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	withdrawal := model.Withdrawal{UserID: "12345", OrderID: "12345", Withdrawal: 10.0}

	mockRepo.EXPECT().Withdraw(ctx, withdrawal).Return(nil)
	err := s.Withdraw(ctx, withdrawal)
	assert.NoError(t, err)
}

func TestService_Withdraw_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	withdrawal := model.Withdrawal{UserID: "12345", OrderID: "12345", Withdrawal: 10.0}

	mockRepo.EXPECT().Withdraw(ctx, withdrawal).Return(errors.New("internal error"))
	err := s.Withdraw(ctx, withdrawal)
	assert.Error(t, err)
}

func TestService_GetWithdrawalsByUserID_Success(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	userID := "12345"

	mockRepo.EXPECT().GetWithdrawalsByUserID(ctx, userID).Return([]model.Withdrawal{{UserID: userID, OrderID: "12345", Withdrawal: 10.0}}, nil)
	withdrawals, err := s.GetWithdrawalsByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, withdrawals, 1)
}

func TestService_GetWithdrawalsByUserID_InternalError(t *testing.T) {
	mockRepo := newMockRepo(t)
	mockClient := newMockClient(t)

	s := NewService(mockRepo, mockClient)

	ctx := context.Background()
	userID := "12345"

	mockRepo.EXPECT().GetWithdrawalsByUserID(ctx, userID).Return([]model.Withdrawal{}, errors.New("internal error"))
	_, err := s.GetWithdrawalsByUserID(ctx, userID)
	assert.Error(t, err)
}
