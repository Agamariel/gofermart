package services

import (
	"context"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// OrderStorage определяет интерфейс для работы с заказами.
type OrderStorage interface {
	Create(ctx context.Context, order *models.Order) error
	GetByNumber(ctx context.Context, number string) (*models.Order, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Order, error)
	UpdateStatus(ctx context.Context, number string, status models.OrderStatus, accrual *decimal.Decimal) error
	GetPendingOrders(ctx context.Context) ([]*models.Order, error)
}

// UserStorage определяет интерфейс для работы с пользователями.
type UserStorage interface {
	Create(ctx context.Context, user *models.User) error
	GetByLogin(ctx context.Context, login string) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	Withdraw(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	WithdrawTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, amount decimal.Decimal) error
}

// WithdrawalStorage определяет интерфейс для работы со списаниями.
type WithdrawalStorage interface {
	Create(ctx context.Context, withdrawal *models.Withdrawal) error
	CreateWithTx(ctx context.Context, tx pgx.Tx, withdrawal *models.Withdrawal) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error)
}
