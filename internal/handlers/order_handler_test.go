package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agamariel/gofermart/internal/auth"
	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/services"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

type mockOrderService struct {
	SubmitFunc func(ctx context.Context, userID uuid.UUID, orderNumber string) error
	ListFunc   func(ctx context.Context, userID uuid.UUID) ([]*models.Order, error)
}

func (m *mockOrderService) SubmitOrder(ctx context.Context, userID uuid.UUID, orderNumber string) error {
	if m.SubmitFunc != nil {
		return m.SubmitFunc(ctx, userID, orderNumber)
	}
	return nil
}

func (m *mockOrderService) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*models.Order, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return []*models.Order{}, nil
}

func TestOrderHandler_SubmitOrder(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		body           string
		mockService    *mockOrderService
		expectedStatus int
	}{
		{
			name: "accepted new order",
			body: "79927398713",
			mockService: &mockOrderService{
				SubmitFunc: func(ctx context.Context, uid uuid.UUID, number string) error {
					return nil
				},
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "already uploaded by same user",
			body: "79927398713",
			mockService: &mockOrderService{
				SubmitFunc: func(ctx context.Context, uid uuid.UUID, number string) error {
					return services.ErrOrderAlreadyUploaded
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "order owned by another user",
			body: "79927398713",
			mockService: &mockOrderService{
				SubmitFunc: func(ctx context.Context, uid uuid.UUID, number string) error {
					return services.ErrOrderOwnedByAnotherUser
				},
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "invalid number",
			body: "12345",
			mockService: &mockOrderService{
				SubmitFunc: func(ctx context.Context, uid uuid.UUID, number string) error {
					return services.ErrInvalidOrderNumber
				},
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "empty body",
			body: "",
			mockService: &mockOrderService{
				SubmitFunc: func(ctx context.Context, uid uuid.UUID, number string) error {
					return nil
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			body: "79927398713",
			mockService: &mockOrderService{
				SubmitFunc: func(ctx context.Context, uid uuid.UUID, number string) error {
					return errors.New("db error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMETextPlain)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set(string(auth.UserIDKey), userID)

			handler := NewOrderHandler(tt.mockService)
			err := handler.SubmitOrder(c)

			if tt.expectedStatus < 400 {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Fatalf("status = %d, want %d", rec.Code, tt.expectedStatus)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if he, ok := err.(*echo.HTTPError); ok {
					if he.Code != tt.expectedStatus {
						t.Fatalf("status = %d, want %d", he.Code, tt.expectedStatus)
					}
				}
			}
		})
	}
}

func TestOrderHandler_GetOrders(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		setupContext   func(c *echo.Context)
		mockService    *mockOrderService
		expectedStatus int
		validateBody   func(t *testing.T, body string)
	}{
		{
			name: "success list",
			setupContext: func(c *echo.Context) {
				(*c).Set(string(auth.UserIDKey), userID)
			},
			mockService: &mockOrderService{
				ListFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.Order, error) {
					uploadedAt, _ := time.Parse(time.RFC3339, "2025-12-09T15:04:05Z")
					return []*models.Order{
						{
							Number:     "79927398713",
							Status:     models.OrderStatusNew,
							UploadedAt: uploadedAt,
						},
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "79927398713") {
					t.Errorf("response body does not contain order number: %s", body)
				}
			},
		},
		{
			name: "accrual returned",
			setupContext: func(c *echo.Context) {
				(*c).Set(string(auth.UserIDKey), userID)
			},
			mockService: &mockOrderService{
				ListFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.Order, error) {
					uploadedAt, _ := time.Parse(time.RFC3339, "2025-12-09T15:04:05Z")
					accrual := decimal.NewFromFloat(729.98)
					return []*models.Order{
						{
							Number:     "2357281120545",
							Status:     models.OrderStatusProcessed,
							Accrual:    &accrual,
							UploadedAt: uploadedAt,
						},
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body string) {
				var resp []map[string]interface{}
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp) != 1 {
					t.Fatalf("unexpected response length: %d", len(resp))
				}
				if _, ok := resp[0]["accrual"]; !ok {
					t.Fatalf("accrual field is missing in response: %s", body)
				}
			},
		},
		{
			name: "no content",
			setupContext: func(c *echo.Context) {
				(*c).Set(string(auth.UserIDKey), userID)
			},
			mockService: &mockOrderService{
				ListFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.Order, error) {
					return []*models.Order{}, nil
				},
			},
			expectedStatus: http.StatusNoContent,
			validateBody:   nil,
		},
		{
			name: "missing user in context",
			setupContext: func(c *echo.Context) {
				// не ставим user_id
			},
			mockService:    &mockOrderService{},
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
		{
			name: "internal error",
			setupContext: func(c *echo.Context) {
				(*c).Set(string(auth.UserIDKey), userID)
			},
			mockService: &mockOrderService{
				ListFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.Order, error) {
					return nil, errors.New("db error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupContext(&c)

			handler := NewOrderHandler(tt.mockService)
			err := handler.GetOrders(c)

			if tt.expectedStatus < 400 {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Fatalf("status = %d, want %d", rec.Code, tt.expectedStatus)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if he, ok := err.(*echo.HTTPError); ok {
					if he.Code != tt.expectedStatus {
						t.Fatalf("status = %d, want %d", he.Code, tt.expectedStatus)
					}
				}
			}

			if tt.validateBody != nil {
				tt.validateBody(t, rec.Body.String())
			}
		})
	}
}
