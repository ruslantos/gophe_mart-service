package balance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	internal_errors "github.com/ruslantos/gophemart-service/internal/errors"
	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/model"
)

func TestWithdrawHandler_Handle(t *testing.T) {
	mockService := newMockServiceWithdraw(t)
	handler := NewWithdrawHandler(mockService)

	req := WithdrawRequest{
		Order: "123455",
		Sum:   100.0,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("POST", "/withdraw", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatal(err)
	}

	r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_id"))
	w := httptest.NewRecorder()

	mockService.EXPECT().Withdraw(mock.Anything, model.Withdrawal{
		OrderID:    req.Order,
		UserID:     "user_id",
		Withdrawal: req.Sum,
	}).Return(nil)

	handler.Handle(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWithdrawHandler_Handle_InvalidRequest(t *testing.T) {
	mockService := newMockServiceWithdraw(t)

	handler := NewWithdrawHandler(mockService)

	r, err := http.NewRequest("POST", "/withdraw", bytes.NewBufferString("invalid_json"))
	if err != nil {
		t.Fatal(err)
	}

	r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_id"))

	w := httptest.NewRecorder()

	handler.Handle(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWithdrawHandler_Handle_InvalidOrderNumber(t *testing.T) {
	mockService := newMockServiceWithdraw(t)
	handler := NewWithdrawHandler(mockService)

	req := WithdrawRequest{
		Order: "invalid_order",
		Sum:   100.0,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("POST", "/withdraw", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatal(err)
	}
	r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_id"))
	w := httptest.NewRecorder()

	handler.Handle(w, r)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestWithdrawHandler_Handle_ServiceError(t *testing.T) {
	mockService := newMockServiceWithdraw(t)
	handler := NewWithdrawHandler(mockService)
	req := WithdrawRequest{
		Order: "123455",
		Sum:   100.0,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("POST", "/withdraw", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatal(err)
	}
	r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_id"))
	w := httptest.NewRecorder()

	mockService.EXPECT().Withdraw(mock.Anything, model.Withdrawal{
		OrderID:    req.Order,
		UserID:     "user_id",
		Withdrawal: req.Sum,
	}).Return(errors.New("service error"))

	handler.Handle(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWithdrawHandler_Handle_ServiceInsufficientFundsError(t *testing.T) {
	mockService := newMockServiceWithdraw(t)
	handler := NewWithdrawHandler(mockService)
	req := WithdrawRequest{
		Order: "123455",
		Sum:   100.0,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("POST", "/withdraw", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatal(err)
	}

	r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_id"))
	w := httptest.NewRecorder()

	mockService.EXPECT().Withdraw(mock.Anything, model.Withdrawal{
		OrderID:    req.Order,
		UserID:     "user_id",
		Withdrawal: req.Sum,
	}).Return(internal_errors.ErrInsufficientFunds)

	handler.Handle(w, r)

	assert.Equal(t, http.StatusPaymentRequired, w.Code)
}
