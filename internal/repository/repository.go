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
	"github.com/ruslantos/gophemart-service/internal/models"
)

type User struct {
	Login    string `db:"login"`
	Password string `db:"password"`
}

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
		`CREATE TABLE IF NOT EXISTS orders(order_id TEXT,status TEXT, accrual INT, user_id TEXT);
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
func (r *UserRepository) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	var user User
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

// order
func (r *UserRepository) SaveOrder(ctx context.Context, order models.Order) error {
	q := `
		INSERT INTO orders (order_ud, status, accrual, user_id)
		VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, q, order.OrderID, order.Status, order.Accrual, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}
	return nil
}
func (r *UserRepository) GetOrder(ctx context.Context, orderID string) (models.Order, error) {
	var order models.Order
	q := `SELECT order_id, status, accrual, user_id FROM orders WHERE order_id = $1`
	err := r.db.QueryRowContext(ctx, q, orderID).Scan(&order.OrderID, &order.Status, &order.Accrual, &order.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return order, internalErrors.ErrOrderNotFound // Заказ не найден
		}
		return order, err
	}

	return order, nil
}
