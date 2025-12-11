package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/agamariel/gofermart/internal/auth"
	"github.com/agamariel/gofermart/internal/services"
	"github.com/labstack/echo/v4"
)

// OrderHandler обрабатывает запросы, связанные с заказами.
type OrderHandler struct {
	orderService services.OrderService
}

func NewOrderHandler(orderService services.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// SubmitOrder обрабатывает POST /api/user/orders.
func (h *OrderHandler) SubmitOrder(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "unable to read body")
	}
	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "empty order number")
	}

	err = h.orderService.SubmitOrder(c.Request().Context(), userID, orderNumber)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidOrderNumber):
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid order number")
		case errors.Is(err, services.ErrOrderAlreadyUploaded):
			return c.NoContent(http.StatusOK)
		case errors.Is(err, services.ErrOrderOwnedByAnotherUser):
			return echo.NewHTTPError(http.StatusConflict, "order uploaded by another user")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.NoContent(http.StatusAccepted)
}

// GetOrders обрабатывает GET /api/user/orders.
func (h *OrderHandler) GetOrders(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return err
	}

	orders, err := h.orderService.GetUserOrders(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	if len(orders) == 0 {
		return c.NoContent(http.StatusNoContent)
	}

	return c.JSON(http.StatusOK, orders)
}
