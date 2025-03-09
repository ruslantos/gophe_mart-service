package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	internalErrors "github.com/ruslantos/gophemart-service/internal/errors"
	"github.com/ruslantos/gophemart-service/internal/logger"
	"github.com/ruslantos/gophemart-service/internal/model"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) InitStorage() error {
	_, err := r.db.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS users(login TEXT,password TEXT);
				CREATE UNIQUE INDEX IF NOT EXISTS idx_login ON users(login);`)
	if err != nil {
		logger.Get().Error("Failed to create users", zap.Error(err))
		return err
	}

	_, err = r.db.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS orders(order_id TEXT,status TEXT, accrual FLOAT, user_id TEXT, uploaded_at TIMESTAMP);
				CREATE UNIQUE INDEX IF NOT EXISTS idx_order_id ON orders(order_id);`)
	if err != nil {
		logger.Get().Error("Failed to create orders", zap.Error(err))
		return err
	}

	return nil
}

// user
func (r *UserRepository) CreateUser(ctx context.Context, login, password string) error {
	query := `INSERT INTO users (login, password) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, login, password)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return internalErrors.ErrLoginAlreadyExists
		}
		logger.Get().Error("Failed to create user", zap.Error(err))
		return err
	}
	return nil
}
func (r *UserRepository) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User
	query := `SELECT login, password FROM users WHERE login = $1`
	err := r.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Get().Error("Failed to get user by login", zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) SaveOrder(ctx context.Context, order model.Order) error {
	q := `
INSERT INTO orders (order_id, status, accrual, user_id, uploaded_at) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (order_id) 
DO UPDATE SET status = EXCLUDED.status, accrual = EXCLUDED.accrual
`
	_, err := r.db.ExecContext(ctx, q, order.OrderID, order.Status, order.Accrual, order.UserID, order.UploadedAt)
	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}
	return nil
}
func (r *UserRepository) GetOrder(ctx context.Context, orderID string) (model.Order, error) {
	var order model.Order
	q := `SELECT order_id, status, accrual, user_id, uploaded_at FROM orders WHERE order_id = $1 `
	err := r.db.QueryRowContext(ctx, q, orderID).Scan(&order.OrderID, &order.Status, &order.Accrual, &order.UserID, &order.UploadedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return order, internalErrors.ErrOrderNotFound // Заказ не найден
		}
		return order, err
	}

	return order, nil
}
func (r *UserRepository) GetOrders(ctx context.Context, userID string) ([]model.Order, error) {
	var orders []model.Order
	q := `SELECT order_id, status, accrual, user_id, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return orders, internalErrors.ErrOrderNotFound // Заказ не найден
		}
		return orders, err
	}

	for rows.Next() {
		var order model.Order
		err := rows.Scan(&order.OrderID, &order.Status, &order.Accrual, &order.UserID, &order.UploadedAt)
		if err != nil {
			return orders, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}
