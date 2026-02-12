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
	"github.com/shopspring/decimal"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrLoginExists         = errors.New("login already exists")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// PostgresUserStorage реализует UserStorage для PostgreSQL.
type PostgresUserStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresUserStorage создаёт новый экземпляр PostgresUserStorage.
func NewPostgresUserStorage(pool *pgxpool.Pool) *PostgresUserStorage {
	return &PostgresUserStorage{pool: pool}
}

// Create создаёт нового пользователя.
func (s *PostgresUserStorage) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, login, password_hash, balance, withdrawn, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	// Генерируем UUID, если не задан
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	// Устанавливаем начальные значения
	if user.Balance.IsZero() {
		user.Balance = decimal.Zero
	}
	if user.Withdrawn.IsZero() {
		user.Withdrawn = decimal.Zero
	}

	err := s.pool.QueryRow(ctx, query,
		user.ID,
		user.Login,
		user.PasswordHash,
		user.Balance,
		user.Withdrawn,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Проверка на уникальность логина
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrLoginExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByLogin ищет пользователя по логину.
func (s *PostgresUserStorage) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `
		SELECT id, login, password_hash, balance, withdrawn, created_at, updated_at
		FROM users
		WHERE login = $1
	`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.Balance,
		&user.Withdrawn,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	return user, nil
}

// GetByID ищет пользователя по ID.
func (s *PostgresUserStorage) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, login, password_hash, balance, withdrawn, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.Balance,
		&user.Withdrawn,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// UpdateBalance увеличивает баланс пользователя на указанную сумму.
func (s *PostgresUserStorage) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	query := `
		UPDATE users
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := s.pool.Exec(ctx, query, amount, id)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Withdraw списывает средства с баланса пользователя транзакционно.
func (s *PostgresUserStorage) Withdraw(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.WithdrawTx(ctx, tx, id, amount); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithdrawTx списывает средства в рамках переданной транзакции.
func (s *PostgresUserStorage) WithdrawTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, amount decimal.Decimal) error {
	// Проверяем текущий баланс
	var currentBalance decimal.Decimal
	checkQuery := `SELECT balance FROM users WHERE id = $1 FOR UPDATE`
	err := tx.QueryRow(ctx, checkQuery, id).Scan(&currentBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to check balance: %w", err)
	}

	// Проверяем достаточность средств
	if currentBalance.LessThan(amount) {
		return ErrInsufficientBalance
	}

	// Списываем средства
	updateQuery := `
		UPDATE users
		SET balance = balance - $1, withdrawn = withdrawn + $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err = tx.Exec(ctx, updateQuery, amount, id)
	if err != nil {
		return fmt.Errorf("failed to withdraw: %w", err)
	}

	return nil
}
