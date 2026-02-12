package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Withdrawal представляет списание средств по заказу.
type Withdrawal struct {
	ID          uuid.UUID       `db:"id"`
	UserID      uuid.UUID       `db:"user_id"`
	OrderNumber string          `db:"order_number"`
	Sum         decimal.Decimal `db:"sum"`
	ProcessedAt time.Time       `db:"processed_at"`
}

// WithdrawRequest DTO для запроса списания.
type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// WithdrawalResponse DTO для ответа по списаниям.
type WithdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
