package orders

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/model"
)

func TestGetOrderHandler_Handle(t *testing.T) {

	mockService := newMockServiceGet(t)

	handler := &GetOrderHandler{serviceGet: mockService}
	req, err := http.NewRequest("GET", "/orders", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user_id"))

	ctx := req.Context()

	orders := []model.Order{
		{OrderID: "order_id", Status: "status", Accrual: 10.0, UploadedAt: time.Now()},
	}
	mockService.EXPECT().GetOrders(ctx, "user_id").Return(orders, nil)

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetOrderHandler_Handle_UserNotFound(t *testing.T) {

	mockService := newMockServiceGet(t)

	handler := &GetOrderHandler{serviceGet: mockService}
	req, err := http.NewRequest("GET", "/orders", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetOrderHandler_Handle_InternalServerError(t *testing.T) {

	mockService := newMockServiceGet(t)

	handler := &GetOrderHandler{serviceGet: mockService}
	req, err := http.NewRequest("GET", "/orders", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user_id"))

	ctx := req.Context()

	mockService.EXPECT().GetOrders(ctx, "user_id").Return(nil, errors.New("internal error"))

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetOrderHandler_Handle_NoOrders(t *testing.T) {

	mockService := newMockServiceGet(t)

	handler := &GetOrderHandler{serviceGet: mockService}
	req, err := http.NewRequest("GET", "/orders", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user_id"))

	ctx := req.Context()

	mockService.EXPECT().GetOrders(ctx, "user_id").Return([]model.Order{}, nil)

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
