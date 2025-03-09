package getorders

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

type service interface {
	GetOrders(ctx context.Context, userID string) ([]model.Order, error)
}

type OrderResponse struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()
	ctx := r.Context()

	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Получаем заказы пользователя
	orders, err := h.service.GetOrders(ctx, userID)
	if err != nil {
		log.Error("Failed to get user orders", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Если заказов нет
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	respBody := makeResponse(orders)

	result, err := json.Marshal(respBody)
	if err != nil {
		http.Error(w, "Marshalling error", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func makeResponse(orders []model.Order) []OrderResponse {
	var response []OrderResponse
	for _, order := range orders {
		response = append(response, OrderResponse{
			Number:     order.OrderID,
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt.Format(time.RFC3339),
		})
	}
	return response
}
