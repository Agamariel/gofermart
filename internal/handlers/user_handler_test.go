package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/services"
	"github.com/agamariel/gofermart/internal/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MockUserService - мок для тестирования handlers
type MockUserService struct {
	RegisterFunc   func(ctx context.Context, login, password string) (*models.User, string, error)
	LoginFunc      func(ctx context.Context, login, password string) (*models.User, string, error)
	GetBalanceFunc func(ctx context.Context, userID uuid.UUID) (*models.BalanceResponse, error)
}

func (m *MockUserService) Register(ctx context.Context, login, password string) (*models.User, string, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, login, password)
	}
	return nil, "", nil
}

func (m *MockUserService) Login(ctx context.Context, login, password string) (*models.User, string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, login, password)
	}
	return nil, "", nil
}

func (m *MockUserService) GetBalance(ctx context.Context, userID uuid.UUID) (*models.BalanceResponse, error) {
	if m.GetBalanceFunc != nil {
		return m.GetBalanceFunc(ctx, userID)
	}
	return nil, nil
}

func TestUserHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockService    *MockUserService
		expectedStatus int
		checkCookie    bool
	}{
		{
			name:        "successful registration",
			requestBody: `{"login":"test@example.com","password":"password123"}`,
			mockService: &MockUserService{
				RegisterFunc: func(ctx context.Context, login, password string) (*models.User, string, error) {
					return &models.User{
						ID:    uuid.New(),
						Login: login,
					}, "test-token", nil
				},
			},
			expectedStatus: http.StatusOK,
			checkCookie:    true,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{"login":"test@example.com"`,
			mockService:    &MockUserService{},
			expectedStatus: http.StatusBadRequest,
			checkCookie:    false,
		},
		{
			name:        "empty credentials",
			requestBody: `{"login":"","password":""}`,
			mockService: &MockUserService{
				RegisterFunc: func(ctx context.Context, login, password string) (*models.User, string, error) {
					return nil, "", services.ErrEmptyCredentials
				},
			},
			expectedStatus: http.StatusBadRequest,
			checkCookie:    false,
		},
		{
			name:        "login already exists",
			requestBody: `{"login":"existing@example.com","password":"password123"}`,
			mockService: &MockUserService{
				RegisterFunc: func(ctx context.Context, login, password string) (*models.User, string, error) {
					return nil, "", storage.ErrLoginExists
				},
			},
			expectedStatus: http.StatusConflict,
			checkCookie:    false,
		},
		{
			name:        "internal error",
			requestBody: `{"login":"test@example.com","password":"password123"}`,
			mockService: &MockUserService{
				RegisterFunc: func(ctx context.Context, login, password string) (*models.User, string, error) {
					return nil, "", errors.New("database error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			checkCookie:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := NewUserHandler(tt.mockService)
			err := handler.Register(c)

			if tt.expectedStatus < 400 {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if he, ok := err.(*echo.HTTPError); ok {
					if he.Code != tt.expectedStatus {
						t.Errorf("Expected status %d, got %d", tt.expectedStatus, he.Code)
					}
				}
			}

			if tt.checkCookie {
				cookies := rec.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "Authorization" {
						found = true
						if cookie.Value == "" {
							t.Error("Cookie value is empty")
						}
					}
				}
				if !found {
					t.Error("Authorization cookie not set")
				}
			}
		})
	}
}

func TestUserHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockService    *MockUserService
		expectedStatus int
		checkCookie    bool
	}{
		{
			name:        "successful login",
			requestBody: `{"login":"test@example.com","password":"password123"}`,
			mockService: &MockUserService{
				LoginFunc: func(ctx context.Context, login, password string) (*models.User, string, error) {
					return &models.User{
						ID:    uuid.New(),
						Login: login,
					}, "test-token", nil
				},
			},
			expectedStatus: http.StatusOK,
			checkCookie:    true,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{"login":"test@example.com"`,
			mockService:    &MockUserService{},
			expectedStatus: http.StatusBadRequest,
			checkCookie:    false,
		},
		{
			name:        "empty credentials",
			requestBody: `{"login":"","password":""}`,
			mockService: &MockUserService{
				LoginFunc: func(ctx context.Context, login, password string) (*models.User, string, error) {
					return nil, "", services.ErrEmptyCredentials
				},
			},
			expectedStatus: http.StatusBadRequest,
			checkCookie:    false,
		},
		{
			name:        "invalid credentials",
			requestBody: `{"login":"test@example.com","password":"wrongpassword"}`,
			mockService: &MockUserService{
				LoginFunc: func(ctx context.Context, login, password string) (*models.User, string, error) {
					return nil, "", services.ErrInvalidCredentials
				},
			},
			expectedStatus: http.StatusUnauthorized,
			checkCookie:    false,
		},
		{
			name:        "internal error",
			requestBody: `{"login":"test@example.com","password":"password123"}`,
			mockService: &MockUserService{
				LoginFunc: func(ctx context.Context, login, password string) (*models.User, string, error) {
					return nil, "", errors.New("database error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			checkCookie:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := NewUserHandler(tt.mockService)
			err := handler.Login(c)

			if tt.expectedStatus < 400 {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if he, ok := err.(*echo.HTTPError); ok {
					if he.Code != tt.expectedStatus {
						t.Errorf("Expected status %d, got %d", tt.expectedStatus, he.Code)
					}
				}
			}

			if tt.checkCookie {
				cookies := rec.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "Authorization" {
						found = true
					}
				}
				if !found {
					t.Error("Authorization cookie not set")
				}
			}
		})
	}
}

func TestUserHandler_GetBalance(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		setupContext   func(*echo.Context)
		mockService    *MockUserService
		expectedStatus int
		checkResponse  bool
	}{
		{
			name: "successful get balance",
			setupContext: func(c *echo.Context) {
				(*c).Set("user_id", userID)
			},
			mockService: &MockUserService{
				GetBalanceFunc: func(ctx context.Context, id uuid.UUID) (*models.BalanceResponse, error) {
					return &models.BalanceResponse{
						Current:   100.50,
						Withdrawn: 42.00,
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name: "user not in context",
			setupContext: func(c *echo.Context) {
				// Не устанавливаем user_id
			},
			mockService:    &MockUserService{},
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  false,
		},
		{
			name: "user not found",
			setupContext: func(c *echo.Context) {
				(*c).Set("user_id", userID)
			},
			mockService: &MockUserService{
				GetBalanceFunc: func(ctx context.Context, id uuid.UUID) (*models.BalanceResponse, error) {
					return nil, storage.ErrUserNotFound
				},
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  false,
		},
		{
			name: "internal error",
			setupContext: func(c *echo.Context) {
				(*c).Set("user_id", userID)
			},
			mockService: &MockUserService{
				GetBalanceFunc: func(ctx context.Context, id uuid.UUID) (*models.BalanceResponse, error) {
					return nil, errors.New("database error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupContext(&c)

			handler := NewUserHandler(tt.mockService)
			err := handler.GetBalance(c)

			if tt.expectedStatus < 400 {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			}

			if tt.checkResponse {
				body := rec.Body.String()
				if !strings.Contains(body, "current") {
					t.Error("Response doesn't contain 'current' field")
				}
				if !strings.Contains(body, "withdrawn") {
					t.Error("Response doesn't contain 'withdrawn' field")
				}
			}
		})
	}
}

func TestSetAuthToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	token := "test-token-value"
	setAuthToken(c, token)

	// Проверяем cookie
	cookies := rec.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "Authorization" {
			authCookie = cookie
			break
		}
	}

	if authCookie == nil {
		t.Fatal("Authorization cookie not set")
	}

	if authCookie.Value != token {
		t.Errorf("Cookie value = %v, want %v", authCookie.Value, token)
	}

	if authCookie.HttpOnly != true {
		t.Error("Cookie should be HttpOnly")
	}

	if authCookie.Path != "/" {
		t.Errorf("Cookie path = %v, want /", authCookie.Path)
	}

	if authCookie.MaxAge != 86400 {
		t.Errorf("Cookie MaxAge = %v, want 86400", authCookie.MaxAge)
	}

	// Проверяем header
	authHeader := rec.Header().Get("Authorization")
	expectedHeader := "Bearer " + token
	if authHeader != expectedHeader {
		t.Errorf("Authorization header = %v, want %v", authHeader, expectedHeader)
	}
}
