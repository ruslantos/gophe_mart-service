package balance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/model"
)

func TestGetWithdrawalsHandler_Handle_UserNotFound(t *testing.T) {
	mockService := newMockServiceGetWithdrawals(t)
	handler := NewGetWithdrawalsHandler(mockService)

	req, err := http.NewRequest("GET", "/withdrawals", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, ""))
	w := httptest.NewRecorder()
	handler.Handle(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetWithdrawalsHandler_Handle_InternalServerError(t *testing.T) {
	mockService := newMockServiceGetWithdrawals(t)
	handler := NewGetWithdrawalsHandler(mockService)

	req, err := http.NewRequest("GET", "/withdrawals", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-id"))
	mockService.EXPECT().GetWithdrawalsByUserID(mock.Anything, "user-id").Return(nil, assert.AnError)
	w := httptest.NewRecorder()
	handler.Handle(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetWithdrawalsHandler_Handle_NoContent(t *testing.T) {
	mockService := newMockServiceGetWithdrawals(t)
	handler := NewGetWithdrawalsHandler(mockService)

	req, err := http.NewRequest("GET", "/withdrawals", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-id"))
	mockService.EXPECT().GetWithdrawalsByUserID(mock.Anything, "user-id").Return([]model.Withdrawal{}, nil)
	w := httptest.NewRecorder()
	handler.Handle(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestGetWithdrawalsHandler_Handle_OK(t *testing.T) {
	mockService := newMockServiceGetWithdrawals(t)
	handler := NewGetWithdrawalsHandler(mockService)

	req, err := http.NewRequest("GET", "/withdrawals", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-id"))
	withdrawals := []model.Withdrawal{
		{
			OrderID:     "order-1",
			Withdrawal:  10.99,
			ProcessedAt: time.Now(),
		},
		{
			OrderID:     "order-2",
			Withdrawal:  20.99,
			ProcessedAt: time.Now(),
		},
	}
	mockService.EXPECT().GetWithdrawalsByUserID(mock.Anything, "user-id").Return(withdrawals, nil)
	w := httptest.NewRecorder()
	handler.Handle(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var response []WithdrawalResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
}
