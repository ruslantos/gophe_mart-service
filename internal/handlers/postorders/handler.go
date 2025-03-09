package postorders

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	internal_errors "github.com/ruslantos/gophemart-service/internal/errors"
	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/middlware/auth"
	"github.com/ruslantos/gophemart-service/internal/model"
)

type service interface {
	GetOrder(ctx context.Context, orderNumber string) (model.Order, error)
	SendOrderToLoyaltyClient(orderNumber string)
	SaveOrder(ctx context.Context, order model.Order) error
}

type OrderHandler struct {
	service service
}

func NewHandler(service service) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	userID := auth.GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Получаем номер заказа
	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		http.Error(w, "Order number is required", http.StatusBadRequest)
		return
	}

	// Проверяем номер заказа с помощью алгоритма Луна
	if !validateLuhn(orderNumber) {
		http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
		return
	}

	// проверяем заказ и пользователя на наличие в базе данных
	orderDB, err := h.service.GetOrder(r.Context(), orderNumber)
	switch {
	case err != nil && errors.Is(err, internal_errors.ErrOrderNotFound):
		break
	case orderDB.UserID == userID:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Order already uploaded by this user"))
		return
	case orderDB.UserID != userID:
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("Order already uploaded by another user"))
		return
	default:
		log.Error("Failed to get order", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// сохраняем заказ в БД
	err = h.service.SaveOrder(r.Context(), model.Order{
		OrderID:    orderNumber,
		UserID:     userID,
		Status:     model.STATE_NEW,
		UploadedAt: time.Now(),
	})
	if err != nil {
		log.Error("Failed to save order", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Добавляем заказ в воркер
	h.service.SendOrderToLoyaltyClient(orderNumber)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Order accepted for processing"))
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
