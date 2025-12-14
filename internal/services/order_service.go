package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/storage"
	"github.com/agamariel/gofermart/internal/utils"
	"github.com/google/uuid"
)

var (
	ErrInvalidOrderNumber      = errors.New("invalid order number")
	ErrOrderOwnedByAnotherUser = errors.New("order already uploaded by another user")
	ErrOrderAlreadyUploaded    = errors.New("order already uploaded by the same user")
)

// OrderService определяет интерфейс работы с заказами.
type OrderService interface {
	SubmitOrder(ctx context.Context, userID uuid.UUID, orderNumber string) error
	GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*models.Order, error)
}

// OrderServiceImpl реализует OrderService.
type OrderServiceImpl struct {
	orderStorage OrderStorage
}

// NewOrderService создаёт новый сервис заказов.
func NewOrderService(orderStorage OrderStorage) *OrderServiceImpl {
	return &OrderServiceImpl{orderStorage: orderStorage}
}

// SubmitOrder обрабатывает загрузку номера заказа.
func (s *OrderServiceImpl) SubmitOrder(ctx context.Context, userID uuid.UUID, orderNumber string) error {
	orderNumber = normalizeOrderNumber(orderNumber)
	if orderNumber == "" {
		return ErrInvalidOrderNumber
	}

	if !utils.ValidateLuhn(orderNumber) {
		return ErrInvalidOrderNumber
	}

	// Проверяем существование заказа
	existing, err := s.orderStorage.GetByNumber(ctx, orderNumber)
	if err == nil && existing != nil {
		if existing.UserID == userID {
			return ErrOrderAlreadyUploaded
		}
		return ErrOrderOwnedByAnotherUser
	}
	if err != nil && !errors.Is(err, storage.ErrOrderNotFound) {
		return fmt.Errorf("check existing order: %w", err)
	}

	// Создаём новый заказ
	order := &models.Order{
		UserID: userID,
		Number: orderNumber,
		Status: models.OrderStatusNew,
	}

	if err := s.orderStorage.Create(ctx, order); err != nil {
		if errors.Is(err, storage.ErrOrderAlreadyExists) {
			// На случай гонки: проверяем владельца ещё раз
			existing, gErr := s.orderStorage.GetByNumber(ctx, orderNumber)
			if gErr == nil && existing != nil {
				if existing.UserID == userID {
					return ErrOrderAlreadyUploaded
				}
				return ErrOrderOwnedByAnotherUser
			}
		}
		return fmt.Errorf("create order: %w", err)
	}

	return nil
}

// GetUserOrders возвращает список заказов пользователя.
func (s *OrderServiceImpl) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*models.Order, error) {
	orders, err := s.orderStorage.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user orders: %w", err)
	}

	return orders, nil
}

// normalizeOrderNumber убирает пробелы и переносы.
func normalizeOrderNumber(number string) string {
	return strings.TrimSpace(number)
}
