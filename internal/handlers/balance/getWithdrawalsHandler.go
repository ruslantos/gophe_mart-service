package balance

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/model"
)

//go:generate mockery --name=serviceGetWithdrawals --output . --inpackage --with-expecter
type serviceGetWithdrawals interface {
	GetWithdrawalsByUserID(ctx context.Context, userID string) ([]model.Withdrawal, error)
}

type WithdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type GetWithdrawalsHandler struct {
	service serviceGetWithdrawals
}

func NewGetWithdrawalsHandler(service serviceGetWithdrawals) *GetWithdrawalsHandler {
	return &GetWithdrawalsHandler{service: service}
}

func (h *GetWithdrawalsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()
	ctx := r.Context()

	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.service.GetWithdrawalsByUserID(ctx, userID)
	if err != nil {
		log.Error("Failed to get user withdrawals", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var response []WithdrawalResponse
	for _, withdrawal := range withdrawals {
		response = append(response, WithdrawalResponse{
			Order:       withdrawal.OrderID,
			Sum:         withdrawal.Withdrawal,
			ProcessedAt: withdrawal.ProcessedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
