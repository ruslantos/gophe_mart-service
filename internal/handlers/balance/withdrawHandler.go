package balance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	internal_errors "github.com/ruslantos/gophemart-service/internal/errors"
	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/model"
)

//go:generate mockery --name=serviceWithdraw --output . --inpackage --with-expecter
type serviceWithdraw interface {
	Withdraw(ctx context.Context, withdrawal model.Withdrawal) error
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type WithdrawHandler struct {
	service serviceWithdraw
}

func NewWithdrawHandler(service serviceWithdraw) *WithdrawHandler {
	return &WithdrawHandler{service: service}
}

func (h *WithdrawHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()
	ctx := r.Context()

	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if !validateLuhn(req.Order) {
		http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
		return
	}

	err := h.service.Withdraw(ctx, model.Withdrawal{
		OrderID:    req.Order,
		UserID:     userID,
		Withdrawal: req.Sum,
	})
	if err != nil {
		switch {
		case errors.Is(err, internal_errors.ErrInsufficientFunds):
			http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
		default:
			log.Error("Failed to withdraw balance", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Withdrawal successful"))
}

func validateLuhn(number string) bool {
	sum := 0
	parity := len(number) % 2
	for i, char := range number {
		digit, err := strconv.Atoi(string(char))
		if err != nil {
			return false
		}
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}
