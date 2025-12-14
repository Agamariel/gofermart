package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/agamariel/gofermart/internal/auth"
	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/services"
	"github.com/agamariel/gofermart/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

// BalanceHandler обрабатывает списания и историю списаний.
type BalanceHandler struct {
	balanceService services.BalanceService
}

// NewBalanceHandler создаёт новый handler.
func NewBalanceHandler(balanceService services.BalanceService) *BalanceHandler {
	return &BalanceHandler{balanceService: balanceService}
}

// Withdraw обрабатывает POST /api/user/balance/withdraw.
func (h *BalanceHandler) Withdraw(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req models.WithdrawRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request format")
	}

	sum := decimal.NewFromFloat(req.Sum)
	if req.Sum <= 0 {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid sum")
	}

	if err := h.balanceService.Withdraw(c.Request().Context(), userID, req.Order, sum); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidWithdrawalNumber):
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid order number")
		case errors.Is(err, services.ErrInvalidWithdrawalSum):
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid sum")
		case errors.Is(err, storage.ErrInsufficientBalance):
			return echo.NewHTTPError(http.StatusPaymentRequired, "insufficient balance")
		case errors.Is(err, storage.ErrUserNotFound):
			return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
		case errors.Is(err, storage.ErrWithdrawalExists):
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "order already withdrawn")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.NoContent(http.StatusOK)
}

// GetWithdrawals обрабатывает GET /api/user/withdrawals.
func (h *BalanceHandler) GetWithdrawals(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return err
	}

	withdrawals, err := h.balanceService.GetWithdrawals(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	if len(withdrawals) == 0 {
		return c.NoContent(http.StatusNoContent)
	}

	// Маппинг domain моделей в DTO
	response := h.mapWithdrawalsToResponse(withdrawals)
	return c.JSON(http.StatusOK, response)
}

// mapWithdrawalsToResponse преобразует domain модели списаний в DTO для HTTP-ответа.
func (h *BalanceHandler) mapWithdrawalsToResponse(withdrawals []*models.Withdrawal) []*models.WithdrawalResponse {
	var response []*models.WithdrawalResponse
	for _, w := range withdrawals {
		sum, _ := w.Sum.Float64()
		response = append(response, &models.WithdrawalResponse{
			Order:       w.OrderNumber,
			Sum:         sum,
			ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
		})
	}
	return response
}
