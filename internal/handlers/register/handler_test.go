package register

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	internal_errors "github.com/ruslantos/gophemart-service/internal/errors"
)

func TestUserHandler_Handle(t *testing.T) {
	mockService := newMockService(t)
	h := NewUserHandler(mockService)
	req := RegisterRequest{
		Login:    "test",
		Password: "test",
	}
	jsonReq, err := json.Marshal(req)
	assert.Nil(t, err)
	r, err := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonReq))
	assert.Nil(t, err)
	w := httptest.NewRecorder()
	mockService.EXPECT().Register(mock.Anything, req.Login, req.Password).Return(nil)
	h.Handle(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_Handle_InvalidRequest(t *testing.T) {
	mockService := newMockService(t)
	h := NewUserHandler(mockService)
	r, err := http.NewRequest("POST", "/register", bytes.NewBuffer([]byte("invalid")))
	assert.Nil(t, err)
	w := httptest.NewRecorder()
	h.Handle(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Handle_LoginAlreadyExists(t *testing.T) {
	mockService := newMockService(t)
	h := NewUserHandler(mockService)
	req := RegisterRequest{
		Login:    "test",
		Password: "test",
	}
	jsonReq, err := json.Marshal(req)
	assert.Nil(t, err)
	r, err := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonReq))
	assert.Nil(t, err)
	w := httptest.NewRecorder()
	mockService.EXPECT().Register(mock.Anything, req.Login, req.Password).Return(internal_errors.ErrLoginAlreadyExists)
	h.Handle(w, r)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUserHandler_Handle_InternalServerError(t *testing.T) {
	mockService := newMockService(t)
	h := NewUserHandler(mockService)
	req := RegisterRequest{
		Login:    "test",
		Password: "test",
	}
	jsonReq, err := json.Marshal(req)
	assert.Nil(t, err)
	r, err := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonReq))
	assert.Nil(t, err)
	w := httptest.NewRecorder()
	mockService.EXPECT().Register(mock.Anything, req.Login, req.Password).Return(assert.AnError)
	h.Handle(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
