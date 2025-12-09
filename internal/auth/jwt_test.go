package auth

import (
	"testing"
	"time"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/google/uuid"
)

func TestGenerateToken(t *testing.T) {
	secret := "test-secret"
	expiration := 1 * time.Hour

	user := &models.User{
		ID:    uuid.New(),
		Login: "test@example.com",
	}

	tests := []struct {
		name       string
		user       *models.User
		secret     string
		expiration time.Duration
		wantErr    bool
	}{
		{
			name:       "valid user",
			user:       user,
			secret:     secret,
			expiration: expiration,
			wantErr:    false,
		},
		{
			name: "user with empty login",
			user: &models.User{
				ID:    uuid.New(),
				Login: "",
			},
			secret:     secret,
			expiration: expiration,
			wantErr:    false, // JWT не валидирует пустой login
		},
		{
			name: "user with nil UUID",
			user: &models.User{
				ID:    uuid.Nil,
				Login: "test@example.com",
			},
			secret:     secret,
			expiration: expiration,
			wantErr:    false,
		},
		{
			name:       "empty secret",
			user:       user,
			secret:     "",
			expiration: expiration,
			wantErr:    false, // Токен создастся, но будет легко взломать
		},
		{
			name:       "zero expiration",
			user:       user,
			secret:     secret,
			expiration: 0,
			wantErr:    false, // Токен истекает сразу
		},
		{
			name:       "negative expiration",
			user:       user,
			secret:     secret,
			expiration: -1 * time.Hour,
			wantErr:    false, // Токен уже истёк
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.user, tt.secret, tt.expiration)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("GenerateToken() returned empty token")
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret"
	wrongSecret := "wrong-secret"
	expiration := 1 * time.Hour

	user := &models.User{
		ID:    uuid.New(),
		Login: "test@example.com",
	}

	validToken, err := GenerateToken(user, secret, expiration)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	expiredToken, err := GenerateToken(user, secret, -1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	tests := []struct {
		name    string
		token   string
		secret  string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   validToken,
			secret:  secret,
			wantErr: false,
		},
		{
			name:    "wrong secret",
			token:   validToken,
			secret:  wrongSecret,
			wantErr: true,
		},
		{
			name:    "expired token",
			token:   expiredToken,
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "invalid token format",
			token:   "invalid.token.here",
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			secret:  secret,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.token, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if claims == nil {
					t.Error("ValidateToken() returned nil claims")
					return
				}
				if claims.UserID != user.ID {
					t.Errorf("ValidateToken() UserID = %v, want %v", claims.UserID, user.ID)
				}
				if claims.Login != user.Login {
					t.Errorf("ValidateToken() Login = %v, want %v", claims.Login, user.Login)
				}
			}
		})
	}
}

func TestTokenRoundTrip(t *testing.T) {
	secret := "test-secret"
	expiration := 1 * time.Hour

	tests := []struct {
		name string
		user *models.User
	}{
		{
			name: "standard user",
			user: &models.User{
				ID:    uuid.New(),
				Login: "user1@example.com",
			},
		},
		{
			name: "user with special characters",
			user: &models.User{
				ID:    uuid.New(),
				Login: "user+test@example.com",
			},
		},
		{
			name: "user with unicode",
			user: &models.User{
				ID:    uuid.New(),
				Login: "пользователь@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Генерируем токен
			token, err := GenerateToken(tt.user, secret, expiration)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}

			// Валидируем токен
			claims, err := ValidateToken(token, secret)
			if err != nil {
				t.Fatalf("ValidateToken() error = %v", err)
			}

			// Проверяем, что данные совпадают
			if claims.UserID != tt.user.ID {
				t.Errorf("UserID mismatch: got %v, want %v", claims.UserID, tt.user.ID)
			}
			if claims.Login != tt.user.Login {
				t.Errorf("Login mismatch: got %v, want %v", claims.Login, tt.user.Login)
			}

			// Проверяем время истечения
			if claims.ExpiresAt == nil {
				t.Error("ExpiresAt is nil")
			}
			if claims.IssuedAt == nil {
				t.Error("IssuedAt is nil")
			}
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	secret := "test-secret"
	user := &models.User{
		ID:    uuid.New(),
		Login: "test@example.com",
	}

	// Создаем токен с очень коротким временем жизни
	shortExpiration := 500 * time.Millisecond
	token, err := GenerateToken(user, secret, shortExpiration)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Сразу должен быть валидным
	_, err = ValidateToken(token, secret)
	if err != nil {
		t.Errorf("ValidateToken() immediately after generation failed: %v", err)
	}

	// Ждём истечения (с запасом)
	time.Sleep(700 * time.Millisecond)

	// Теперь должен быть невалидным
	_, err = ValidateToken(token, secret)
	if err == nil {
		t.Error("ValidateToken() should fail for expired token")
	}
}

func TestValidateTokenReturnsError(t *testing.T) {
	secret := "test-secret"

	// Тест с токеном, подписанным неправильным алгоритмом
	// (это сложно сделать без внешних библиотек, но можем проверить базовые случаи)

	t.Run("modified token", func(t *testing.T) {
		user := &models.User{
			ID:    uuid.New(),
			Login: "test@example.com",
		}

		token, err := GenerateToken(user, secret, time.Hour)
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		// Модифицируем токен
		modifiedToken := token + "modified"

		_, err = ValidateToken(modifiedToken, secret)
		if err == nil {
			t.Error("ValidateToken() should fail for modified token")
		}
	})
}

func BenchmarkGenerateToken(b *testing.B) {
	secret := "test-secret"
	expiration := 1 * time.Hour
	user := &models.User{
		ID:    uuid.New(),
		Login: "bench@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateToken(user, secret, expiration)
	}
}

func BenchmarkValidateToken(b *testing.B) {
	secret := "test-secret"
	expiration := 1 * time.Hour
	user := &models.User{
		ID:    uuid.New(),
		Login: "bench@example.com",
	}

	token, _ := GenerateToken(user, secret, expiration)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateToken(token, secret)
	}
}
