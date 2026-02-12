//go:build integration
// +build integration

package storage

import (
	"context"
	"os"
	"testing"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

func getTestDBPool(t *testing.T) *pgxpool.Pool {
	dbURI := os.Getenv("DATABASE_URI")
	if dbURI == "" {
		t.Skip("DATABASE_URI not set, skipping integration tests")
	}

	pool, err := pgxpool.New(context.Background(), dbURI)
	if err != nil {
		t.Fatalf("Unable to connect to database: %v", err)
	}

	return pool
}

func TestPostgresUserStorage_Create(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	storage := NewPostgresUserStorage(pool)
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Login:        "test_" + uuid.New().String() + "@example.com",
			PasswordHash: "hashed_password",
			Balance:      decimal.Zero,
			Withdrawn:    decimal.Zero,
		}

		err := storage.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Проверяем, что пользователь создан
		retrieved, err := storage.GetByLogin(ctx, user.Login)
		if err != nil {
			t.Fatalf("GetByLogin() error = %v", err)
		}

		if retrieved.Login != user.Login {
			t.Errorf("Login mismatch: got %v, want %v", retrieved.Login, user.Login)
		}
	})

	t.Run("duplicate login", func(t *testing.T) {
		login := "duplicate_" + uuid.New().String() + "@example.com"

		user1 := &models.User{
			ID:           uuid.New(),
			Login:        login,
			PasswordHash: "hash1",
		}

		err := storage.Create(ctx, user1)
		if err != nil {
			t.Fatalf("First Create() error = %v", err)
		}

		user2 := &models.User{
			ID:           uuid.New(),
			Login:        login,
			PasswordHash: "hash2",
		}

		err = storage.Create(ctx, user2)
		if err != ErrLoginExists {
			t.Errorf("Expected ErrLoginExists, got %v", err)
		}
	})
}

func TestPostgresUserStorage_GetByLogin(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	storage := NewPostgresUserStorage(pool)
	ctx := context.Background()

	// Создаем тестового пользователя
	user := &models.User{
		ID:           uuid.New(),
		Login:        "getbylogin_" + uuid.New().String() + "@example.com",
		PasswordHash: "hashed_password",
	}

	err := storage.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("existing user", func(t *testing.T) {
		retrieved, err := storage.GetByLogin(ctx, user.Login)
		if err != nil {
			t.Fatalf("GetByLogin() error = %v", err)
		}

		if retrieved.ID != user.ID {
			t.Errorf("ID mismatch: got %v, want %v", retrieved.ID, user.ID)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		_, err := storage.GetByLogin(ctx, "nonexistent@example.com")
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})
}

func TestPostgresUserStorage_GetByID(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	storage := NewPostgresUserStorage(pool)
	ctx := context.Background()

	user := &models.User{
		ID:           uuid.New(),
		Login:        "getbyid_" + uuid.New().String() + "@example.com",
		PasswordHash: "hashed_password",
	}

	err := storage.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("existing user", func(t *testing.T) {
		retrieved, err := storage.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if retrieved.Login != user.Login {
			t.Errorf("Login mismatch: got %v, want %v", retrieved.Login, user.Login)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		_, err := storage.GetByID(ctx, uuid.New())
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})
}

func TestPostgresUserStorage_UpdateBalance(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	storage := NewPostgresUserStorage(pool)
	ctx := context.Background()

	user := &models.User{
		ID:           uuid.New(),
		Login:        "updatebalance_" + uuid.New().String() + "@example.com",
		PasswordHash: "hashed_password",
		Balance:      decimal.NewFromFloat(100),
	}

	err := storage.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("successful update", func(t *testing.T) {
		addAmount := decimal.NewFromFloat(50)
		err := storage.UpdateBalance(ctx, user.ID, addAmount)
		if err != nil {
			t.Fatalf("UpdateBalance() error = %v", err)
		}

		// Проверяем новый баланс
		retrieved, err := storage.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		expectedBalance := decimal.NewFromFloat(150)
		if !retrieved.Balance.Equal(expectedBalance) {
			t.Errorf("Balance = %v, want %v", retrieved.Balance, expectedBalance)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		err := storage.UpdateBalance(ctx, uuid.New(), decimal.NewFromFloat(10))
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})
}

func TestPostgresUserStorage_Withdraw(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	storage := NewPostgresUserStorage(pool)
	ctx := context.Background()

	t.Run("successful withdraw", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Login:        "withdraw_success_" + uuid.New().String() + "@example.com",
			PasswordHash: "hashed_password",
			Balance:      decimal.NewFromFloat(100),
			Withdrawn:    decimal.Zero,
		}

		err := storage.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		withdrawAmount := decimal.NewFromFloat(30)
		err = storage.Withdraw(ctx, user.ID, withdrawAmount)
		if err != nil {
			t.Fatalf("Withdraw() error = %v", err)
		}

		// Проверяем баланс и withdrawn
		retrieved, err := storage.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		expectedBalance := decimal.NewFromFloat(70)
		expectedWithdrawn := decimal.NewFromFloat(30)

		if !retrieved.Balance.Equal(expectedBalance) {
			t.Errorf("Balance = %v, want %v", retrieved.Balance, expectedBalance)
		}
		if !retrieved.Withdrawn.Equal(expectedWithdrawn) {
			t.Errorf("Withdrawn = %v, want %v", retrieved.Withdrawn, expectedWithdrawn)
		}
	})

	t.Run("insufficient balance", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Login:        "withdraw_insufficient_" + uuid.New().String() + "@example.com",
			PasswordHash: "hashed_password",
			Balance:      decimal.NewFromFloat(10),
		}

		err := storage.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		withdrawAmount := decimal.NewFromFloat(20)
		err = storage.Withdraw(ctx, user.ID, withdrawAmount)
		if err != ErrInsufficientBalance {
			t.Errorf("Expected ErrInsufficientBalance, got %v", err)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		err := storage.Withdraw(ctx, uuid.New(), decimal.NewFromFloat(10))
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})
}
