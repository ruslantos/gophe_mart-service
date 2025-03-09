package models

type Order struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Accrual int64  `json:"accrual"`
	UserID  string `json:"user_id"`
}
