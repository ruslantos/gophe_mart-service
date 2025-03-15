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

type User struct {
	Login    string `db:"login"`
	Password string `db:"password"`
}

type UserBalance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type Withdrawal struct {
	OrderID     string    `json:"order_id"`
	UserID      string    `json:"user_id"`
	Withdrawal  float64   `json:"withdrawn"`
	ProcessedAt time.Time `json:"processed_at"`
}
