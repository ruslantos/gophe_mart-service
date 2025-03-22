package errors

import (
	"errors"
)

var ErrLoginAlreadyExists = errors.New("login already exists")
var ErrOrderNotFound = errors.New("order not found")
var ErrInsufficientFunds = errors.New("insufficient funds")
