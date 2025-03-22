package balance

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/model"
)

//go:generate mockery --name=serviceGetBalance --output . --inpackage --with-expecter
type serviceGetBalance interface {
	GetUserBalance(ctx context.Context, userID string) (model.UserBalance, error)
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type GetHandler struct {
	service serviceGetBalance
}

func NewGetHandler(service serviceGetBalance) *GetHandler {
	return &GetHandler{service: service}
}

func (h *GetHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()
	ctx := r.Context()

	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	balance, err := h.service.GetUserBalance(ctx, userID)
	if err != nil {
		log.Error("Failed to get user balance", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Возвращаем ответ в формате JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(balance); err != nil {
		log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
