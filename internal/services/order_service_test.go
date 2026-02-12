package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/storage"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockOrderStorage struct {
	CreateFunc       func(ctx context.Context, order *models.Order) error
	GetByNumberFunc  func(ctx context.Context, number string) (*models.Order, error)
	GetByUserIDFunc  func(ctx context.Context, userID uuid.UUID) ([]*models.Order, error)
	UpdateStatusFunc func(ctx context.Context, number string, status models.OrderStatus, accrual *decimal.Decimal) error
	GetPendingFunc   func(ctx context.Context) ([]*models.Order, error)
}

func (m *mockOrderStorage) Create(ctx context.Context, order *models.Order) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, order)
	}
	return nil
}

func (m *mockOrderStorage) GetByNumber(ctx context.Context, number string) (*models.Order, error) {
	if m.GetByNumberFunc != nil {
		return m.GetByNumberFunc(ctx, number)
	}
	return nil, storage.ErrOrderNotFound
}

func (m *mockOrderStorage) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Order, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return []*models.Order{}, nil
}

func (m *mockOrderStorage) UpdateStatus(ctx context.Context, number string, status models.OrderStatus, accrual *decimal.Decimal) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, number, status, accrual)
	}
	return nil
}

func (m *mockOrderStorage) GetPendingOrders(ctx context.Context) ([]*models.Order, error) {
	if m.GetPendingFunc != nil {
		return m.GetPendingFunc(ctx)
	}
	return []*models.Order{}, nil
}

func TestOrderService_SubmitOrder(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()

	validNumber := "79927398713" // проходит Луна

	t.Run("invalid number", func(t *testing.T) {
		svc := NewOrderService(&mockOrderStorage{})
		if err := svc.SubmitOrder(ctx, userID, "12345"); !errors.Is(err, ErrInvalidOrderNumber) {
			t.Fatalf("expected ErrInvalidOrderNumber, got %v", err)
		}
	})

	t.Run("order already uploaded by same user", func(t *testing.T) {
		svc := NewOrderService(&mockOrderStorage{
			GetByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
				return &models.Order{UserID: userID, Number: validNumber}, nil
			},
		})
		if err := svc.SubmitOrder(ctx, userID, validNumber); !errors.Is(err, ErrOrderAlreadyUploaded) {
			t.Fatalf("expected ErrOrderAlreadyUploaded, got %v", err)
		}
	})

	t.Run("order owned by another user", func(t *testing.T) {
		svc := NewOrderService(&mockOrderStorage{
			GetByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
				return &models.Order{UserID: otherUserID, Number: validNumber}, nil
			},
		})
		if err := svc.SubmitOrder(ctx, userID, validNumber); !errors.Is(err, ErrOrderOwnedByAnotherUser) {
			t.Fatalf("expected ErrOrderOwnedByAnotherUser, got %v", err)
		}
	})

	t.Run("create new order", func(t *testing.T) {
		created := false
		svc := NewOrderService(&mockOrderStorage{
			GetByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
				return nil, storage.ErrOrderNotFound
			},
			CreateFunc: func(ctx context.Context, order *models.Order) error {
				created = true
				if order.Status != models.OrderStatusNew {
					t.Fatalf("unexpected status %v", order.Status)
				}
				return nil
			},
		})
		if err := svc.SubmitOrder(ctx, userID, validNumber); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !created {
			t.Fatal("order not created")
		}
	})

	t.Run("storage error on lookup", func(t *testing.T) {
		svc := NewOrderService(&mockOrderStorage{
			GetByNumberFunc: func(ctx context.Context, number string) (*models.Order, error) {
				return nil, errors.New("db error")
			},
		})
		if err := svc.SubmitOrder(ctx, userID, validNumber); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestOrderService_GetUserOrders(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	accrual := decimal.NewFromFloat(150.50)

	orders := []*models.Order{
		{
			ID:         uuid.New(),
			UserID:     userID,
			Number:     "79927398713",
			Status:     models.OrderStatusProcessed,
			Accrual:    &accrual,
			UploadedAt: now,
			UpdatedAt:  now,
		},
	}

	svc := NewOrderService(&mockOrderStorage{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.Order, error) {
			return orders, nil
		},
	})

	resp, err := svc.GetUserOrders(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 order, got %d", len(resp))
	}
	if resp[0].Number != orders[0].Number {
		t.Errorf("number mismatch: got %s, want %s", resp[0].Number, orders[0].Number)
	}
	expectedAccrual := decimal.NewFromFloat(150.50)
	if resp[0].Accrual == nil || !resp[0].Accrual.Equal(expectedAccrual) {
		t.Errorf("accrual mismatch: got %v, want %v", resp[0].Accrual, expectedAccrual)
	}
}

func TestOrderService_GetUserOrdersEmpty(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	svc := NewOrderService(&mockOrderStorage{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.Order, error) {
			return []*models.Order{}, nil
		},
	})

	resp, err := svc.GetUserOrders(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty slice, got %d", len(resp))
	}
}
