package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/agamariel/gofermart/internal/accrual"
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

// App структура для управления приложением и его зависимостями.
type App struct {
	cfg    *config.Config
	dbPool *pgxpool.Pool
	echo   *echo.Echo
	worker *services.AccrualWorker

	// Handlers
	userHandler    *handlers.UserHandler
	orderHandler   *handlers.OrderHandler
	balanceHandler *handlers.BalanceHandler
}

// NewApp создаёт и инициализирует новое приложение.
func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	app := &App{
		cfg: cfg,
	}

	if err := app.initDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := app.initDependencies(); err != nil {
		return nil, fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	app.initServer()

	return app, nil
}

// initDatabase инициализирует подключение к базе данных и выполняет миграции.
func (app *App) initDatabase(ctx context.Context) error {
	if app.cfg.DatabaseURI == "" {
		return fmt.Errorf("DATABASE_URI is required")
	}

	// Применение миграций
	log.Println("Running database migrations...")
	sqlDB, err := sql.Open("pgx", app.cfg.DatabaseURI)
	if err != nil {
		return fmt.Errorf("unable to open database connection: %w", err)
	}
	defer sqlDB.Close()

	if err := migrations.Run(sqlDB); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Println("Migrations completed successfully")

	// Подключение к базе данных через pgxpool
	dbPool, err := pgxpool.New(ctx, app.cfg.DatabaseURI)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	app.dbPool = dbPool
	log.Println("Successfully connected to database")

	return nil
}

// initDependencies инициализирует все зависимости приложения (storage, services, handlers).
func (app *App) initDependencies() error {
	// Storage layer
	userStorage := storage.NewPostgresUserStorage(app.dbPool)
	orderStorage := storage.NewPostgresOrderStorage(app.dbPool)
	withdrawalStorage := storage.NewPostgresWithdrawalStorage(app.dbPool)

	// Service layer
	userService := services.NewUserService(userStorage, app.cfg.JWTSecret, app.cfg.TokenExpiration)
	orderService := services.NewOrderService(orderStorage)
	balanceService := services.NewBalanceService(app.dbPool, userStorage, withdrawalStorage)

	// Handler layer
	app.userHandler = handlers.NewUserHandler(userService)
	app.orderHandler = handlers.NewOrderHandler(orderService)
	app.balanceHandler = handlers.NewBalanceHandler(balanceService)

	// Воркер начислений
	if app.cfg.AccrualSystemAddress != "" {
		log.Printf("Initializing accrual worker with address: %s", app.cfg.AccrualSystemAddress)
		client := accrual.NewHTTPAccrualClient(app.cfg.AccrualSystemAddress, 5*time.Second)
		app.worker = services.NewAccrualWorker(app.dbPool, orderStorage, userStorage, client, 5*time.Second, log.Default())
		log.Println("Accrual worker initialized successfully")
	} else {
		log.Println("WARNING: AccrualSystemAddress is not configured. Orders will not be processed for accruals!")
	}

	return nil
}

// initServer инициализирует HTTP-сервер и настраивает маршруты.
func (app *App) initServer() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	// Публичные маршруты (не требуют аутентификации)
	e.POST("/api/user/register", app.userHandler.Register)
	e.POST("/api/user/login", app.userHandler.Login)

	// Защищённые маршруты (требуют аутентификации)
	protected := e.Group("/api/user")
	protected.Use(auth.JWTMiddleware(app.cfg.JWTSecret))
	protected.GET("/balance", app.userHandler.GetBalance)
	protected.POST("/orders", app.orderHandler.SubmitOrder)
	protected.GET("/orders", app.orderHandler.GetOrders)
	protected.POST("/balance/withdraw", app.balanceHandler.Withdraw)
	protected.GET("/withdrawals", app.balanceHandler.GetWithdrawals)

	app.echo = e
}

// Start запускает приложение.
func (app *App) Start(ctx context.Context) error {
	// Запуск воркера начислений
	if app.worker != nil {
		log.Println("Starting accrual worker...")
		app.worker.Start(ctx)
		log.Println("Accrual worker started")
	} else {
		log.Println("Accrual worker is not configured")
	}

	// Запуск сервера
	log.Printf("Starting server on %s", app.cfg.RunAddress)
	if err := app.echo.Start(app.cfg.RunAddress); err != nil {
		return fmt.Errorf("server stopped: %w", err)
	}

	return nil
}

// Shutdown корректно завершает работу приложения.
func (app *App) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	if err := app.echo.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	if app.dbPool != nil {
		app.dbPool.Close()
	}

	log.Println("Server gracefully stopped")
	return nil
}
