package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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

	_, err = r.db.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS accruals(user_id TEXT, accruals_sum FLOAT, withdrawals FLOAT DEFAULT 0);
				CREATE UNIQUE INDEX IF NOT EXISTS idx_user_id ON accruals(user_id);`)
	if err != nil {
		logger.Get().Error("Failed to create withdrawals", zap.Error(err))
		return err
	}

	_, err = r.db.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS withdrawals(user_id TEXT, order_id TEXT, withdrawals FLOAT, processed_at TIMESTAMP);`)
	if err != nil {
		logger.Get().Error("Failed to create withdrawals", zap.Error(err))
		return err
	}

	return nil
}

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
	if rows.Err() != nil {
		return orders, rows.Err()
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

func (r *UserRepository) UpdateUserAccrualSum(ctx context.Context, order model.Order) error {
	q := `
INSERT INTO  accruals(user_id, accruals_sum) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET accruals_sum = accruals.accruals_sum + EXCLUDED.accruals_sum;
`
	rows, err := r.db.ExecContext(ctx, q, order.UserID, order.Accrual)
	if err != nil {
		return fmt.Errorf("failed to save user accrual: %w", err)
	}
	rowsAffected, err := rows.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to save user accrual: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("failed to save user accrual: no rows affected")
	}
	return nil
}
func (r *UserRepository) GetUserAccrualSum(ctx context.Context, userID string) (model.UserBalance, error) {
	var balance model.UserBalance
	q := `SELECT accruals_sum, withdrawals FROM accruals WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil && err != sql.ErrNoRows {
		balance.Current = 0
	} else {
		logger.Get().Error("Failed to get user accrual sum", zap.Error(err))
		return balance, err
	}

	return balance, nil
}
func (r *UserRepository) Withdraw(ctx context.Context, withdrawal model.Withdrawal) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// проверяем баланс
	var currentBalance float64
	q := `SELECT accruals_sum FROM accruals WHERE user_id = $1 FOR UPDATE;`
	err = tx.GetContext(ctx, &currentBalance, q, withdrawal.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return internalErrors.ErrInsufficientFunds
		}
		logger.Get().Error("Failed to get user accrual sum", zap.Error(err))
	}

	if currentBalance < withdrawal.Withdrawal {
		return internalErrors.ErrInsufficientFunds
	}

	// списываем баллы
	q = `UPDATE accruals SET accruals_sum = accruals_sum - $1, withdrawals = withdrawals + $1
                WHERE user_id = $2`
	_, err = tx.ExecContext(ctx, q, withdrawal.Withdrawal, withdrawal.UserID)
	if err != nil {
		return fmt.Errorf("failed to update user accrual: %w", err)
	}

	// делаем запись о списании
	q = `INSERT INTO withdrawals (user_id, order_id, withdrawals, processed_at) VALUES ($1, $2, $3, $4)`
	_, err = tx.ExecContext(ctx, q, withdrawal.UserID, withdrawal.OrderID, withdrawal.Withdrawal, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save withdrawal: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) GetWithdrawalsByUserID(ctx context.Context, userID string) ([]model.Withdrawal, error) {
	var withdrawals []model.Withdrawal
	q := `SELECT user_id, order_id, withdrawals, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	defer rows.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			return withdrawals, nil
		}
		return withdrawals, err
	}

	for rows.Next() {
		var withdrawal model.Withdrawal
		err := rows.Scan(&withdrawal.UserID, &withdrawal.OrderID, &withdrawal.Withdrawal, &withdrawal.ProcessedAt)
		if err != nil {
			return withdrawals, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}

	return withdrawals, nil
}
