package clients

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/logger"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrTooManyRequests    = errors.New("too many requests")
	ErrInternalServer     = errors.New("internal server error")
	ErrUnexpectedResponse = errors.New("unexpected response")
)

type OrderResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type LoyaltyClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewLoyaltyClient(baseURL string) *LoyaltyClient {
	return &LoyaltyClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Таймаут для запросов
		},
	}
}

func (c *LoyaltyClient) GetOrderInfo(orderNumber string) (*OrderResponse, error) {
	var URL strings.Builder
	URL.Grow(128)
	URL.WriteString(c.baseURL)
	URL.WriteString("/api/orders/")
	URL.WriteString(orderNumber)

	req, err := http.NewRequest(http.MethodGet, URL.String(), bytes.NewBuffer(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	logger.Get().Info("got response from accrual:", zap.Int("status code", resp.StatusCode))

	switch resp.StatusCode {
	case http.StatusOK:
		var orderResp OrderResponse
		if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &orderResp, nil

	case http.StatusNoContent:
		return nil, ErrOrderNotFound

	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("%w: retry after %s seconds", ErrTooManyRequests, retryAfter)

	case http.StatusInternalServerError:
		return nil, ErrInternalServer

	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status code %d, body: %s", ErrUnexpectedResponse, resp.StatusCode, string(body))
	}
}
