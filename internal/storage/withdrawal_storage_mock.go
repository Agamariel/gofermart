package storage

import (
	"context"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MockWithdrawalStorage - мок для тестов.
type MockWithdrawalStorage struct {
	CreateFunc       func(ctx context.Context, w *models.Withdrawal) error
	CreateWithTxFunc func(ctx context.Context, tx pgx.Tx, w *models.Withdrawal) error
	GetByUserIDFunc  func(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error)
}

func (m *MockWithdrawalStorage) Create(ctx context.Context, w *models.Withdrawal) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, w)
	}
	return nil
}

func (m *MockWithdrawalStorage) CreateWithTx(ctx context.Context, tx pgx.Tx, w *models.Withdrawal) error {
	if m.CreateWithTxFunc != nil {
		return m.CreateWithTxFunc(ctx, tx, w)
	}
	return nil
}

func (m *MockWithdrawalStorage) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return []*models.Withdrawal{}, nil
}
