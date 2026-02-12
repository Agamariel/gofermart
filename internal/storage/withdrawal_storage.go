package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWithdrawalExists = errors.New("withdrawal already exists for order")
)

// PostgresWithdrawalStorage реализует WithdrawalStorage для PostgreSQL.
type PostgresWithdrawalStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresWithdrawalStorage создаёт новый экземпляр.
func NewPostgresWithdrawalStorage(pool *pgxpool.Pool) *PostgresWithdrawalStorage {
	return &PostgresWithdrawalStorage{pool: pool}
}

// Create создаёт списание вне явной транзакции.
func (s *PostgresWithdrawalStorage) Create(ctx context.Context, withdrawal *models.Withdrawal) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.CreateWithTx(ctx, tx, withdrawal); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit withdrawal: %w", err)
	}
	return nil
}

// CreateWithTx создаёт списание в рамках переданной транзакции.
func (s *PostgresWithdrawalStorage) CreateWithTx(ctx context.Context, tx pgx.Tx, withdrawal *models.Withdrawal) error {
	if withdrawal.ID == uuid.Nil {
		withdrawal.ID = uuid.New()
	}

	query := `
		INSERT INTO withdrawals (id, user_id, order_number, sum, processed_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING processed_at
	`

	_, err := tx.Exec(ctx, query, withdrawal.ID, withdrawal.UserID, withdrawal.OrderNumber, withdrawal.Sum)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrWithdrawalExists
		}
		return fmt.Errorf("failed to create withdrawal: %w", err)
	}

	return nil
}

// GetByUserID возвращает списания пользователя, отсортированные по времени (новые первыми).
func (s *PostgresWithdrawalStorage) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error) {
	query := `
		SELECT id, user_id, order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []*models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		if err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, fmt.Errorf("failed to scan withdrawal: %w", err)
		}
		withdrawals = append(withdrawals, &w)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows error: %w", rows.Err())
	}

	return withdrawals, nil
}
