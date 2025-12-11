package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderAlreadyExists = errors.New("order already exists")
)

// OrderStorage определяет интерфейс для работы с заказами.
type OrderStorage interface {
	Create(ctx context.Context, order *models.Order) error
	GetByNumber(ctx context.Context, number string) (*models.Order, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Order, error)
	UpdateStatus(ctx context.Context, number string, status models.OrderStatus, accrual *decimal.Decimal) error
	GetPendingOrders(ctx context.Context) ([]*models.Order, error)
}

// PostgresOrderStorage реализует OrderStorage для PostgreSQL.
type PostgresOrderStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresOrderStorage создаёт новый экземпляр PostgresOrderStorage.
func NewPostgresOrderStorage(pool *pgxpool.Pool) *PostgresOrderStorage {
	return &PostgresOrderStorage{pool: pool}
}

// Create создаёт новый заказ.
func (s *PostgresOrderStorage) Create(ctx context.Context, order *models.Order) error {
	query := `
		INSERT INTO orders (user_id, number, status, accrual, uploaded_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, uploaded_at, updated_at
	`

	accrualVal := sql.NullString{}
	if order.Accrual != nil {
		accrualVal = sql.NullString{Valid: true, String: order.Accrual.String()}
	}

	err := s.pool.QueryRow(ctx, query,
		order.UserID,
		order.Number,
		order.Status,
		accrualVal,
	).Scan(&order.ID, &order.UploadedAt, &order.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrOrderAlreadyExists
		}
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// GetByNumber возвращает заказ по номеру.
func (s *PostgresOrderStorage) GetByNumber(ctx context.Context, number string) (*models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE number = $1
	`

	return scanOrder(s.pool.QueryRow(ctx, query, number))
}

// GetByUserID возвращает список заказов пользователя (сортировка по uploaded_at DESC).
func (s *PostgresOrderStorage) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows error: %w", rows.Err())
	}

	return orders, nil
}

// UpdateStatus обновляет статус и начисление заказа.
func (s *PostgresOrderStorage) UpdateStatus(ctx context.Context, number string, status models.OrderStatus, accrual *decimal.Decimal) error {
	query := `
		UPDATE orders
		SET status = $1, accrual = $2, updated_at = NOW()
		WHERE number = $3
	`

	accrualVal := sql.NullString{}
	if accrual != nil {
		accrualVal = sql.NullString{Valid: true, String: accrual.String()}
	}

	result, err := s.pool.Exec(ctx, query, status, accrualVal, number)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrOrderNotFound
	}

	return nil
}

// GetPendingOrders возвращает заказы в статусах NEW и PROCESSING.
func (s *PostgresOrderStorage) GetPendingOrders(ctx context.Context) ([]*models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE status IN ('NEW', 'PROCESSING')
		ORDER BY uploaded_at ASC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows error: %w", rows.Err())
	}

	return orders, nil
}

// scanOrder помогает читать заказ из строки результата.
func scanOrder(row pgx.Row) (*models.Order, error) {
	var (
		order      models.Order
		accrualStr sql.NullString
	)

	err := row.Scan(
		&order.ID,
		&order.UserID,
		&order.Number,
		&order.Status,
		&accrualStr,
		&order.UploadedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to scan order: %w", err)
	}

	if accrualStr.Valid {
		if dec, derr := decimal.NewFromString(accrualStr.String); derr == nil {
			order.Accrual = &dec
		}
	}

	return &order, nil
}
