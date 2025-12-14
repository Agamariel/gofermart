package services

import (
	"context"
	"log"
	"time"

	"github.com/agamariel/gofermart/internal/accrual"
	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// AccrualWorker периодически обновляет статусы заказов и начисляет баллы.
type AccrualWorker struct {
	pool         *pgxpool.Pool
	orderStorage OrderStorage
	userStorage  UserStorage
	client       accrual.AccrualClient
	interval     time.Duration
	logger       *log.Logger
}

func NewAccrualWorker(pool *pgxpool.Pool, orderStorage OrderStorage, userStorage UserStorage, client accrual.AccrualClient, interval time.Duration, logger *log.Logger) *AccrualWorker {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if logger == nil {
		logger = log.Default()
	}
	return &AccrualWorker{
		pool:         pool,
		orderStorage: orderStorage,
		userStorage:  userStorage,
		client:       client,
		interval:     interval,
		logger:       logger,
	}
}

// Start запускает воркер в отдельной горутине и останавливается по ctx.Done().
func (w *AccrualWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	go func() {
		defer ticker.Stop()
		if err := w.processBatch(ctx); err != nil {
			w.logger.Printf("accrual worker error on initial batch: %v", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.processBatch(ctx); err != nil {
					w.logger.Printf("accrual worker error: %v", err)
				}
			}
		}
	}()
}

func (w *AccrualWorker) processBatch(ctx context.Context) error {
	orders, err := w.orderStorage.GetPendingOrders(ctx)
	if err != nil {
		w.logger.Printf("failed to get pending orders: %v", err)
		return err
	}

	if len(orders) > 0 {
		w.logger.Printf("processing %d pending orders", len(orders))
	}

	for _, o := range orders {
		if err := w.processOrder(ctx, o); err != nil {
			w.logger.Printf("process order %s error: %v", o.Number, err)
		}
	}
	return nil
}

func (w *AccrualWorker) processOrder(ctx context.Context, order *models.Order) error {
	w.logger.Printf("fetching accrual for order %s", order.Number)
	resp, err := w.client.GetOrderAccrual(ctx, order.Number)
	if err != nil {
		if rl, ok := err.(accrual.RateLimitError); ok {
			w.logger.Printf("rate limited for order %s, retrying after %s", order.Number, rl.RetryAfter)
			time.Sleep(rl.RetryAfter)
			return nil
		}
		if err == accrual.ErrNotFound {
			w.logger.Printf("order %s not found in accrual system, skipping", order.Number)
			return nil
		}
		w.logger.Printf("error fetching accrual for order %s: %v", order.Number, err)
		return err
	}

	w.logger.Printf("order %s status: %s, accrual: %v", order.Number, resp.Status, resp.Accrual)
	switch resp.Status {
	case "REGISTERED":
		return w.orderStorage.UpdateStatus(ctx, order.Number, models.OrderStatusProcessing, nil)
	case "PROCESSING":
		return w.orderStorage.UpdateStatus(ctx, order.Number, models.OrderStatusProcessing, nil)
	case "INVALID":
		return w.orderStorage.UpdateStatus(ctx, order.Number, models.OrderStatusInvalid, nil)
	case "PROCESSED":
		w.logger.Printf("applying processed accrual for order %s: %s", order.Number, resp.Accrual.String())
		return w.applyProcessed(ctx, order.UserID, order.Number, resp.Accrual)
	default:
		w.logger.Printf("unknown status %s for order %s", resp.Status, order.Number)
		return nil
	}
}

func (w *AccrualWorker) applyProcessed(ctx context.Context, userID uuid.UUID, orderNumber string, accrual decimal.Decimal) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}

	// Обновляем заказ
	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET status = $1, accrual = $2, updated_at = NOW()
		WHERE number = $3
	`, models.OrderStatusProcessed, accrual, orderNumber)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	// Начисляем баланс
	_, err = tx.Exec(ctx, `
		UPDATE users
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2
	`, accrual, userID)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		w.logger.Printf("failed to commit accrual transaction for order %s: %v", orderNumber, err)
		return err
	}
	w.logger.Printf("successfully committed accrual for order %s: %s", orderNumber, accrual.String())
	return nil
}
