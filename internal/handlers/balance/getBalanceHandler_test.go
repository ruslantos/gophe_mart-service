package balance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/model"
)

func TestGetHandler_Handle(t *testing.T) {
	mockService := newMockServiceGetBalance(t)
	handler := NewGetHandler(mockService)

	req, err := http.NewRequest("GET", "/balance", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "test_user_id"))

	balance := model.UserBalance{
		Current:   100.0,
		Withdrawn: 50.0,
	}
	mockService.EXPECT().GetUserBalance(mock.Anything, "test_user_id").Return(balance, nil)

	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response BalanceResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, balance.Current, response.Current)
	assert.Equal(t, balance.Withdrawn, response.Withdrawn)
}

func TestGetHandler_Handle_InternalServerError(t *testing.T) {
	mockService := newMockServiceGetBalance(t)
	handler := NewGetHandler(mockService)

	req, err := http.NewRequest("GET", "/balance", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "test_user_id"))

	mockService.EXPECT().GetUserBalance(mock.Anything, "test_user_id").Return(model.UserBalance{}, assert.AnError)

	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestGetHandler_Handle_Unauthorized(t *testing.T) {
	mockService := newMockServiceGetBalance(t)
	handler := NewGetHandler(mockService)

	req, err := http.NewRequest("GET", "/balance", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
