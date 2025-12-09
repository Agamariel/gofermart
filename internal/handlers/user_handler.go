package handlers

import (
	"errors"
	"net/http"

	"github.com/agamariel/gofermart/internal/auth"
	"github.com/agamariel/gofermart/internal/models"
	"github.com/agamariel/gofermart/internal/services"
	"github.com/agamariel/gofermart/internal/storage"
	"github.com/labstack/echo/v4"
)

// UserHandler обрабатывает HTTP-запросы для работы с пользователями.
type UserHandler struct {
	userService services.UserService
}

// NewUserHandler создаёт новый экземпляр UserHandler.
func NewUserHandler(userService services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// Register обрабатывает POST /api/user/register.
func (h *UserHandler) Register(c echo.Context) error {
	var req models.RegisterRequest

	// Парсинг JSON body
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request format")
	}

	// Вызов сервиса регистрации
	user, token, err := h.userService.Register(c.Request().Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrEmptyCredentials) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, storage.ErrLoginExists) {
			return echo.NewHTTPError(http.StatusConflict, "login already exists")
		}
		c.Logger().Errorf("failed to register user: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	// Установка токена в cookie и заголовок
	setAuthToken(c, token)

	// Возврат успешного ответа
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id": user.ID,
		"login":   user.Login,
	})
}

// Login обрабатывает POST /api/user/login.
func (h *UserHandler) Login(c echo.Context) error {
	var req models.LoginRequest

	// Парсинг JSON body
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request format")
	}

	// Вызов сервиса аутентификации
	user, token, err := h.userService.Login(c.Request().Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrEmptyCredentials) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, services.ErrInvalidCredentials) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid login or password")
		}
		c.Logger().Errorf("failed to login user: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	// Установка токена в cookie и заголовок
	setAuthToken(c, token)

	// Возврат успешного ответа
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id": user.ID,
		"login":   user.Login,
	})
}

// GetBalance обрабатывает GET /api/user/balance.
func (h *UserHandler) GetBalance(c echo.Context) error {
	// Получение ID пользователя из контекста (установлен middleware)
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return err // Уже HTTP-ошибка
	}

	// Получение баланса
	balance, err := h.userService.GetBalance(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
		}
		c.Logger().Errorf("failed to get balance: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, balance)
}

// setAuthToken устанавливает токен в cookie и заголовок ответа.
func setAuthToken(c echo.Context, token string) {
	// Установка cookie
	cookie := &http.Cookie{
		Name:     "Authorization",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 часа
	}
	c.SetCookie(cookie)

	// Также устанавливаем в заголовок для удобства
	c.Response().Header().Set("Authorization", "Bearer "+token)
}
