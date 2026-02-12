package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestJWTMiddleware(t *testing.T) {
	secret := "test-secret"
	user := &models.User{
		ID:    uuid.New(),
		Login: "test@example.com",
	}

	validToken, _ := GenerateToken(user, secret, time.Hour)
	expiredToken, _ := GenerateToken(user, secret, -time.Hour)

	tests := []struct {
		name           string
		token          string
		tokenLocation  string // "header" or "cookie"
		expectedStatus int
		checkContext   bool
	}{
		{
			name:           "valid token in header",
			token:          validToken,
			tokenLocation:  "header",
			expectedStatus: http.StatusOK,
			checkContext:   true,
		},
		{
			name:           "valid token in cookie",
			token:          validToken,
			tokenLocation:  "cookie",
			expectedStatus: http.StatusOK,
			checkContext:   true,
		},
		{
			name:           "missing token",
			token:          "",
			tokenLocation:  "",
			expectedStatus: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name:           "invalid token in header",
			token:          "invalid.token.here",
			tokenLocation:  "header",
			expectedStatus: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name:           "expired token",
			token:          expiredToken,
			tokenLocation:  "header",
			expectedStatus: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name:           "malformed bearer token",
			token:          "NotBearer " + validToken,
			tokenLocation:  "header",
			expectedStatus: http.StatusUnauthorized,
			checkContext:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Устанавливаем токен в зависимости от location
			switch tt.tokenLocation {
			case "header":
				req.Header.Set("Authorization", "Bearer "+tt.token)
			case "cookie":
				req.AddCookie(&http.Cookie{
					Name:  "Authorization",
					Value: tt.token,
				})
			}

			// Handler, который вызывается после middleware
			handler := func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			}

			// Создаём middleware
			middleware := JWTMiddleware(secret)
			h := middleware(handler)

			// Вызываем
			err := h(c)

			// Проверяем статус
			if tt.expectedStatus == http.StatusOK {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if he, ok := err.(*echo.HTTPError); ok {
					if he.Code != tt.expectedStatus {
						t.Errorf("Expected status %d, got %d", tt.expectedStatus, he.Code)
					}
				}
			}

			// Проверяем контекст
			if tt.checkContext {
				userID, ok := c.Get(string(UserIDKey)).(uuid.UUID)
				if !ok {
					t.Error("UserID not found in context")
				}
				if userID != user.ID {
					t.Errorf("UserID mismatch: got %v, want %v", userID, user.ID)
				}

				login, ok := c.Get(string(UserLoginKey)).(string)
				if !ok {
					t.Error("Login not found in context")
				}
				if login != user.Login {
					t.Errorf("Login mismatch: got %v, want %v", login, user.Login)
				}
			}
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	userID := uuid.New()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "valid user ID in context",
			setup: func() {
				c.Set(string(UserIDKey), userID)
			},
			wantErr: false,
		},
		{
			name: "no user ID in context",
			setup: func() {
				// Не устанавливаем ничего
			},
			wantErr: true,
		},
		{
			name: "wrong type in context",
			setup: func() {
				c.Set(string(UserIDKey), "not-a-uuid")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем контекст
			c = e.NewContext(req, rec)
			tt.setup()

			got, err := GetUserIDFromContext(c)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserIDFromContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != userID {
				t.Errorf("GetUserIDFromContext() = %v, want %v", got, userID)
			}
		})
	}
}

func TestGetUserLoginFromContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	login := "test@example.com"

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "valid login in context",
			setup: func() {
				c.Set(string(UserLoginKey), login)
			},
			wantErr: false,
		},
		{
			name: "no login in context",
			setup: func() {
				// Не устанавливаем ничего
			},
			wantErr: true,
		},
		{
			name: "wrong type in context",
			setup: func() {
				c.Set(string(UserLoginKey), 12345)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем контекст
			c = e.NewContext(req, rec)
			tt.setup()

			got, err := GetUserLoginFromContext(c)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserLoginFromContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != login {
				t.Errorf("GetUserLoginFromContext() = %v, want %v", got, login)
			}
		})
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "valid bearer token",
			header: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			want:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:   "bearer lowercase",
			header: "bearer token123",
			want:   "token123",
		},
		{
			name:   "no bearer prefix",
			header: "token123",
			want:   "",
		},
		{
			name:   "empty header",
			header: "",
			want:   "",
		},
		{
			name:   "only bearer",
			header: "Bearer",
			want:   "",
		},
		{
			name:   "extra spaces",
			header: "Bearer  token123",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			got := extractTokenFromHeader(c)
			if got != tt.want {
				t.Errorf("extractTokenFromHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTokenFromCookie(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name   string
		cookie *http.Cookie
		want   string
	}{
		{
			name: "valid cookie",
			cookie: &http.Cookie{
				Name:  "Authorization",
				Value: "token123",
			},
			want: "token123",
		},
		{
			name:   "no cookie",
			cookie: nil,
			want:   "",
		},
		{
			name: "wrong cookie name",
			cookie: &http.Cookie{
				Name:  "WrongName",
				Value: "token123",
			},
			want: "",
		},
		{
			name: "empty cookie value",
			cookie: &http.Cookie{
				Name:  "Authorization",
				Value: "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			got := extractTokenFromCookie(c)
			if got != tt.want {
				t.Errorf("extractTokenFromCookie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJWTMiddlewarePriority(t *testing.T) {
	// Тестируем, что токен из header имеет приоритет над cookie
	secret := "test-secret"
	user := &models.User{
		ID:    uuid.New(),
		Login: "test@example.com",
	}

	validToken, _ := GenerateToken(user, secret, time.Hour)
	invalidToken := "invalid.token"

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Устанавливаем валидный токен в header
	req.Header.Set("Authorization", "Bearer "+validToken)
	// И невалидный в cookie
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: invalidToken,
	})

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	middleware := JWTMiddleware(secret)
	h := middleware(handler)

	err := h(c)
	if err != nil {
		t.Errorf("Expected no error with valid header token, got %v", err)
	}
}
