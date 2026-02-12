package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// User представляет пользователя системы.
type User struct {
	ID           uuid.UUID       `db:"id"`
	Login        string          `db:"login"`
	PasswordHash string          `db:"password_hash"`
	Balance      decimal.Decimal `db:"balance"`
	Withdrawn    decimal.Decimal `db:"withdrawn"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
}

// RegisterRequest - запрос на регистрацию пользователя.
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// LoginRequest - запрос на аутентификацию пользователя.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// BalanceResponse - ответ с балансом пользователя.
type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
