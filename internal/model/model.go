package model

import (
	"time"
)

const (
	STATE_NEW        = "NEW"
	STATE_REGISTERED = "REGISTERED"
	STATE_INVALID    = "INVALID"
	STATE_PROCESSING = "PROCESSING"
	STATE_PROCESSED  = "PROCESSED"
)

type Order struct {
	OrderID    string    `json:"order_id"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual"`
	UserID     string    `json:"user_id"`
	UploadedAt time.Time `json:"uploaded_at"`
}
