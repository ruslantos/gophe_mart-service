package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_Handle_LoginRequestDecodeError(t *testing.T) {
	mockService := newMockService(t)
	handler := NewHandler(mockService)
	req, err := http.NewRequest("POST", "/login", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.Handle(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Handle_LoginAndPasswordRequired(t *testing.T) {
	mockService := newMockService(t)
	handler := NewHandler(mockService)
	loginReq := LoginRequest{
		Login: "",
	}
	jsonReq, err := json.Marshal(loginReq)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonReq))
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.Handle(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Handle_InvalidCredentials(t *testing.T) {
	mockService := newMockService(t)
	mockService.EXPECT().Authenticate(mock.Anything, "test", "test").Return(false)
	handler := NewHandler(mockService)
	loginReq := LoginRequest{
		Login:    "test",
		Password: "test",
	}
	jsonReq, err := json.Marshal(loginReq)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonReq))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Handle_Authenticated(t *testing.T) {
	mockService := newMockService(t)
	mockService.EXPECT().Authenticate(mock.Anything, "test", "test").Return(true)
	handler := NewHandler(mockService)
	loginReq := LoginRequest{
		Login:    "test",
		Password: "test",
	}
	jsonReq, err := json.Marshal(loginReq)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonReq))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	handler.Handle(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
