package storage

import (
	"context"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MockUserStorage - мок для тестирования (экспортируемый для использования в других пакетах)
type MockUserStorage struct {
	CreateFunc        func(ctx context.Context, user *models.User) error
	GetByLoginFunc    func(ctx context.Context, login string) (*models.User, error)
	GetByIDFunc       func(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateBalanceFunc func(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	WithdrawFunc      func(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
}

func (m *MockUserStorage) Create(ctx context.Context, user *models.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserStorage) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	if m.GetByLoginFunc != nil {
		return m.GetByLoginFunc(ctx, login)
	}
	return nil, ErrUserNotFound
}

func (m *MockUserStorage) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, ErrUserNotFound
}

func (m *MockUserStorage) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	if m.UpdateBalanceFunc != nil {
		return m.UpdateBalanceFunc(ctx, id, amount)
	}
	return nil
}

func (m *MockUserStorage) Withdraw(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	if m.WithdrawFunc != nil {
		return m.WithdrawFunc(ctx, id, amount)
	}
	return nil
}

