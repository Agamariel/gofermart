package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderStatus описывает статус обработки заказа.
type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

// Order представляет заказ пользователя.
type Order struct {
	ID         uuid.UUID        `db:"id"`
	UserID     uuid.UUID        `db:"user_id"`
	Number     string           `db:"number"`
	Status     OrderStatus      `db:"status"`
	Accrual    *decimal.Decimal `db:"accrual"`
	UploadedAt time.Time        `db:"uploaded_at"`
	UpdatedAt  time.Time        `db:"updated_at"`
}

// OrderResponse ответ для списка заказов.
type OrderResponse struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}
