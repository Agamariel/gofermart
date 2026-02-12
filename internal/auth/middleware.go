package auth

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ContextKey - тип для ключей контекста.
type ContextKey string

const (
	// UserIDKey - ключ для хранения ID пользователя в контексте.
	UserIDKey ContextKey = "user_id"
	// UserLoginKey - ключ для хранения логина пользователя в контексте.
	UserLoginKey ContextKey = "user_login"
)

// JWTMiddleware создаёт middleware для проверки JWT токена.
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractTokenFromHeader(c)

			if token == "" {
				token = extractTokenFromCookie(c)
			}

			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid token")
			}

			claims, err := ValidateToken(token, secret)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			// Сохранение данных пользователя в контексте
			c.Set(string(UserIDKey), claims.UserID)
			c.Set(string(UserLoginKey), claims.Login)

			return next(c)
		}
	}
}

// extractTokenFromHeader извлекает токен из заголовка Authorization.
func extractTokenFromHeader(c echo.Context) string {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Проверка формата "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}

	return ""
}

// extractTokenFromCookie извлекает токен из cookie.
func extractTokenFromCookie(c echo.Context) string {
	cookie, err := c.Cookie("Authorization")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetUserIDFromContext извлекает ID пользователя из контекста.
func GetUserIDFromContext(c echo.Context) (uuid.UUID, error) {
	userID, ok := c.Get(string(UserIDKey)).(uuid.UUID)
	if !ok {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, "user not found in context")
	}
	return userID, nil
}

// GetUserLoginFromContext извлекает логин пользователя из контекста.
func GetUserLoginFromContext(c echo.Context) (string, error) {
	login, ok := c.Get(string(UserLoginKey)).(string)
	if !ok {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "user not found in context")
	}
	return login, nil
}
