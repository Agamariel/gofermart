package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agamariel/gofermart/internal/auth"
	"github.com/agamariel/gofermart/internal/config"
	"github.com/agamariel/gofermart/internal/handlers"
	"github.com/agamariel/gofermart/internal/migrations"
	"github.com/agamariel/gofermart/internal/services"
	"github.com/agamariel/gofermart/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := config.Load()

	if cfg.DatabaseURI == "" {
		log.Fatal("DATABASE_URI is required")
	}

	// Применение миграций
	log.Println("Running database migrations...")
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		log.Fatalf("Unable to open database connection: %v", err)
	}
	defer sqlDB.Close()

	if err := migrations.Run(sqlDB); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations completed successfully")

	// Подключение к базе данных через pgxpool
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURI)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}
	log.Println("Successfully connected to database")

	userStorage := storage.NewPostgresUserStorage(dbPool)
	userService := services.NewUserService(userStorage, cfg.JWTSecret, cfg.TokenExpiration)
	userHandler := handlers.NewUserHandler(userService)

	orderStorage := storage.NewPostgresOrderStorage(dbPool)
	orderService := services.NewOrderService(orderStorage)
	orderHandler := handlers.NewOrderHandler(orderService)

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	// Публичные маршруты (не требуют аутентификации)
	e.POST("/api/user/register", userHandler.Register)
	e.POST("/api/user/login", userHandler.Login)

	// Защищённые маршруты (требуют аутентификации)
	protected := e.Group("/api/user")
	protected.Use(auth.JWTMiddleware(cfg.JWTSecret))
	protected.GET("/balance", userHandler.GetBalance)
	protected.POST("/orders", orderHandler.SubmitOrder)
	protected.GET("/orders", orderHandler.GetOrders)

	// Запуск сервера
	go func() {
		log.Printf("Starting server on %s", cfg.RunAddress)
		if err := e.Start(cfg.RunAddress); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Server gracefully stopped")
}
