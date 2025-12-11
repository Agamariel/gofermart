package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/agamariel/gofermart/internal/auth"
	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/storage"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmptyCredentials   = errors.New("login and password are required")
)

// UserService определяет интерфейс для работы с пользователями.
type UserService interface {
	Register(ctx context.Context, login, password string) (*models.User, string, error)
	Login(ctx context.Context, login, password string) (*models.User, string, error)
	GetBalance(ctx context.Context, userID uuid.UUID) (*models.BalanceResponse, error)
}

// UserServiceImpl реализует UserService.
type UserServiceImpl struct {
	userStorage     storage.UserStorage
	jwtSecret       string
	tokenExpiration time.Duration
}

// NewUserService создаёт новый экземпляр UserService.
func NewUserService(userStorage storage.UserStorage, jwtSecret string, tokenExpiration time.Duration) *UserServiceImpl {
	return &UserServiceImpl{
		userStorage:     userStorage,
		jwtSecret:       jwtSecret,
		tokenExpiration: tokenExpiration,
	}
}

// Register регистрирует нового пользователя.
func (s *UserServiceImpl) Register(ctx context.Context, login, password string) (*models.User, string, error) {
	if login == "" || password == "" {
		return nil, "", ErrEmptyCredentials
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: passwordHash,
	}

	err = s.userStorage.Create(ctx, user)
	if err != nil {
		if errors.Is(err, storage.ErrLoginExists) {
			return nil, "", storage.ErrLoginExists
		}
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}

// Login аутентифицирует пользователя.
func (s *UserServiceImpl) Login(ctx context.Context, login, password string) (*models.User, string, error) {
	if login == "" || password == "" {
		return nil, "", ErrEmptyCredentials
	}

	user, err := s.userStorage.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	if !auth.CheckPassword(password, user.PasswordHash) {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}

// GetBalance возвращает баланс пользователя.
func (s *UserServiceImpl) GetBalance(ctx context.Context, userID uuid.UUID) (*models.BalanceResponse, error) {
	user, err := s.userStorage.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, storage.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Конвертация decimal в float64
	current, _ := user.Balance.Float64()
	withdrawn, _ := user.Withdrawn.Float64()

	return &models.BalanceResponse{
		Current:   current,
		Withdrawn: withdrawn,
	}, nil
}

// generateToken генерирует JWT токен для пользователя.
func (s *UserServiceImpl) generateToken(user *models.User) (string, error) {
	exp := s.tokenExpiration
	if exp <= 0 {
		exp = 24 * time.Hour
	}
	token, err := auth.GenerateToken(user, s.jwtSecret, exp)
	if err != nil {
		return "", err
	}
	return token, nil
}
