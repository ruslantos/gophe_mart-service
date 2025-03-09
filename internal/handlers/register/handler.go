package register

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	internalErrors "github.com/ruslantos/gophemart-service/internal/errors"
	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
)

type service interface {
	Register(ctx context.Context, login, password string) error
}

type UserHandler struct {
	service service
}

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func NewUserHandler(service service) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()
	var req RegisterRequest
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
	if err := h.service.Register(ctx, req.Login, req.Password); err != nil {
		if err == internalErrors.ErrLoginAlreadyExists {
			http.Error(w, "Login already exists", http.StatusConflict)
			return
		}
		log.Error("Failed to register user", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newCookie := auth.CreateSignedCookie(req.Login)
	http.SetCookie(w, &newCookie)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User registered and authenticated successfully"))
}
