package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
)

type service interface {
	Authenticate(ctx context.Context, login, password string) bool
}

type UserHandler struct {
	service service
}

func NewHandler(service service) *UserHandler {
	return &UserHandler{service: service}
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *UserHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	authenticated := h.service.Authenticate(ctx, req.Login, req.Password)
	if !authenticated {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	newCookie := auth.CreateSignedCookie(req.Login)
	http.SetCookie(w, &newCookie)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User authenticated successfully"))
}
