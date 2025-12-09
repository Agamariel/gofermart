package services

import (
	"context"
	"errors"
	"testing"

	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/storage"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestUserServiceImpl_Register(t *testing.T) {
	ctx := context.Background()
	secret := "test-secret"

	tests := []struct {
		name        string
		login       string
		password    string
		mockStorage *storage.MockUserStorage
		wantErr     bool
		errType     error
	}{
		{
			name:     "successful registration",
			login:    "test@example.com",
			password: "password123",
			mockStorage: &storage.MockUserStorage{
				CreateFunc: func(ctx context.Context, user *models.User) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name:        "empty login",
			login:       "",
			password:    "password123",
			mockStorage: &storage.MockUserStorage{},
			wantErr:     true,
			errType:     ErrEmptyCredentials,
		},
		{
			name:        "empty password",
			login:       "test@example.com",
			password:    "",
			mockStorage: &storage.MockUserStorage{},
			wantErr:     true,
			errType:     ErrEmptyCredentials,
		},
		{
			name:     "login already exists",
			login:    "existing@example.com",
			password: "password123",
			mockStorage: &storage.MockUserStorage{
				CreateFunc: func(ctx context.Context, user *models.User) error {
					return storage.ErrLoginExists
				},
			},
			wantErr: true,
			errType: storage.ErrLoginExists,
		},
		{
			name:     "storage error",
			login:    "test@example.com",
			password: "password123",
			mockStorage: &storage.MockUserStorage{
				CreateFunc: func(ctx context.Context, user *models.User) error {
					return errors.New("database error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewUserService(tt.mockStorage, secret, "24h")

			user, token, err := service.Register(ctx, tt.login, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Register() error type = %T, want %T", err, tt.errType)
				}
				return
			}

		// Проверки для успешной регистрации
		if user == nil {
			t.Error("Register() returned nil user")
			return
		}
		if user.Login != tt.login {
			t.Errorf("Register() user.Login = %v, want %v", user.Login, tt.login)
		}
		if token == "" {
			t.Error("Register() returned empty token")
		}
		})
	}
}

func TestUserServiceImpl_Login(t *testing.T) {
	ctx := context.Background()
	secret := "test-secret"
	correctPassword := "password123"

	// Создаём хеш для правильного пароля
	hash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy" // bcrypt hash для "password123"

	existingUser := &models.User{
		ID:           uuid.New(),
		Login:        "test@example.com",
		PasswordHash: hash,
	}

	tests := []struct {
		name        string
		login       string
		password    string
		mockStorage *storage.MockUserStorage
		wantErr     bool
		errType     error
	}{
		{
			name:     "successful login",
			login:    "test@example.com",
			password: correctPassword,
			mockStorage: &storage.MockUserStorage{
				GetByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return existingUser, nil
				},
			},
			wantErr: false,
		},
		{
			name:        "empty login",
			login:       "",
			password:    correctPassword,
			mockStorage: &storage.MockUserStorage{},
			wantErr:     true,
			errType:     ErrEmptyCredentials,
		},
		{
			name:        "empty password",
			login:       "test@example.com",
			password:    "",
			mockStorage: &storage.MockUserStorage{},
			wantErr:     true,
			errType:     ErrEmptyCredentials,
		},
		{
			name:     "user not found",
			login:    "nonexistent@example.com",
			password: correctPassword,
			mockStorage: &storage.MockUserStorage{
				GetByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return nil, storage.ErrUserNotFound
				},
			},
			wantErr: true,
			errType: ErrInvalidCredentials,
		},
		{
			name:     "wrong password",
			login:    "test@example.com",
			password: "wrongpassword",
			mockStorage: &storage.MockUserStorage{
				GetByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return existingUser, nil
				},
			},
			wantErr: true,
			errType: ErrInvalidCredentials,
		},
		{
			name:     "storage error",
			login:    "test@example.com",
			password: correctPassword,
			mockStorage: &storage.MockUserStorage{
				GetByLoginFunc: func(ctx context.Context, login string) (*models.User, error) {
					return nil, errors.New("database error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewUserService(tt.mockStorage, secret, "24h")

			user, token, err := service.Login(ctx, tt.login, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Login() error type = %v, want %v", err, tt.errType)
				}
				return
			}

			// Проверки для успешного логина
			if user == nil {
				t.Error("Login() returned nil user")
			}
			if token == "" {
				t.Error("Login() returned empty token")
			}
		})
	}
}

func TestUserServiceImpl_GetBalance(t *testing.T) {
	ctx := context.Background()
	secret := "test-secret"
	userID := uuid.New()

	user := &models.User{
		ID:        userID,
		Login:     "test@example.com",
		Balance:   decimal.NewFromFloat(100.50),
		Withdrawn: decimal.NewFromFloat(42.00),
	}

	tests := []struct {
		name          string
		userID        uuid.UUID
		mockStorage   *storage.MockUserStorage
		wantErr       bool
		wantCurrent   float64
		wantWithdrawn float64
	}{
		{
			name:   "successful get balance",
			userID: userID,
			mockStorage: &storage.MockUserStorage{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
					return user, nil
				},
			},
			wantErr:       false,
			wantCurrent:   100.50,
			wantWithdrawn: 42.00,
		},
		{
			name:   "user not found",
			userID: uuid.New(),
			mockStorage: &storage.MockUserStorage{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
					return nil, storage.ErrUserNotFound
				},
			},
			wantErr: true,
		},
		{
			name:   "storage error",
			userID: userID,
			mockStorage: &storage.MockUserStorage{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
					return nil, errors.New("database error")
				},
			},
			wantErr: true,
		},
		{
			name:   "zero balance",
			userID: userID,
			mockStorage: &storage.MockUserStorage{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
					return &models.User{
						ID:        userID,
						Balance:   decimal.Zero,
						Withdrawn: decimal.Zero,
					}, nil
				},
			},
			wantErr:       false,
			wantCurrent:   0,
			wantWithdrawn: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewUserService(tt.mockStorage, secret, "24h")

			balance, err := service.GetBalance(ctx, tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBalance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if balance == nil {
				t.Fatal("GetBalance() returned nil balance")
			}

			if balance.Current != tt.wantCurrent {
				t.Errorf("GetBalance() Current = %v, want %v", balance.Current, tt.wantCurrent)
			}
			if balance.Withdrawn != tt.wantWithdrawn {
				t.Errorf("GetBalance() Withdrawn = %v, want %v", balance.Withdrawn, tt.wantWithdrawn)
			}
		})
	}
}

func TestUserServiceImpl_RegisterHashesPassword(t *testing.T) {
	ctx := context.Background()
	secret := "test-secret"
	password := "testpassword123"

	var storedHash string
	mockStorage := &storage.MockUserStorage{
		CreateFunc: func(ctx context.Context, user *models.User) error {
			storedHash = user.PasswordHash
			return nil
		},
	}

	service := NewUserService(mockStorage, secret, "24h")
	_, _, err := service.Register(ctx, "test@example.com", password)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Проверяем, что пароль был хеширован
	if storedHash == password {
		t.Error("Register() did not hash the password")
	}
	if storedHash == "" {
		t.Error("Register() stored empty password hash")
	}
}
