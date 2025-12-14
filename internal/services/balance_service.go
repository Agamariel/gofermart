package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidWithdrawalNumber = errors.New("invalid order number")
	ErrInvalidWithdrawalSum    = errors.New("invalid withdrawal sum")
)

// BalanceService описывает операции по списаниям и истории.
type BalanceService interface {
	Withdraw(ctx context.Context, userID uuid.UUID, orderNumber string, sum decimal.Decimal) error
	GetWithdrawals(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error)
}

type BalanceServiceImpl struct {
	pool              *pgxpool.Pool
	userStorage       UserStorage
	withdrawalStorage WithdrawalStorage
}

// NewBalanceService создаёт сервис баланса.
func NewBalanceService(pool *pgxpool.Pool, userStorage UserStorage, withdrawalStorage WithdrawalStorage) *BalanceServiceImpl {
	return &BalanceServiceImpl{
		pool:              pool,
		userStorage:       userStorage,
		withdrawalStorage: withdrawalStorage,
	}
}

// Withdraw выполняет списание средств.
func (s *BalanceServiceImpl) Withdraw(ctx context.Context, userID uuid.UUID, orderNumber string, sum decimal.Decimal) error {
	orderNumber = strings.TrimSpace(orderNumber)
	if orderNumber == "" || !utils.ValidateLuhn(orderNumber) {
		return ErrInvalidWithdrawalNumber
	}
	if sum.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidWithdrawalSum
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// списание с баланса
	if err := s.userStorage.WithdrawTx(ctx, tx, userID, sum); err != nil {
		return err
	}

	// запись списания
	withdrawal := &models.Withdrawal{
		UserID:      userID,
		OrderNumber: orderNumber,
		Sum:         sum,
		ProcessedAt: time.Now(),
	}
	if err := s.withdrawalStorage.CreateWithTx(ctx, tx, withdrawal); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// GetWithdrawals возвращает историю списаний пользователя.
func (s *BalanceServiceImpl) GetWithdrawals(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error) {
	list, err := s.withdrawalStorage.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return list, nil
}
