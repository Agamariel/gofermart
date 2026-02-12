package config

import (
	"flag"
	"os"
	"time"
)

// Config содержит конфигурацию приложения.
type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	JWTSecret            string
	TokenExpiration      time.Duration
}

// Load загружает конфигурацию из флагов командной строки и переменных окружения.
// Приоритет: переменные окружения > флаги > значения по умолчанию.
func Load() *Config {
	cfg := &Config{}

	const defaultTokenExp = 24 * time.Hour

	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "адрес и порт запуска сервиса")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "строка подключения к PostgreSQL")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "адрес системы расчёта начислений")
	flag.DurationVar(&cfg.TokenExpiration, "t", defaultTokenExp, "время жизни JWT токена (Go duration)")
	flag.Parse()

	if envRunAddr := os.Getenv("RUN_ADDRESS"); envRunAddr != "" {
		cfg.RunAddress = envRunAddr
	}
	if envDBURI := os.Getenv("DATABASE_URI"); envDBURI != "" {
		cfg.DatabaseURI = envDBURI
	}
	if envAccrual := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrual != "" {
		cfg.AccrualSystemAddress = envAccrual
	}

	// JWT секрет
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "default-secret-change-in-production"
	}

	// Время жизни токена: env имеет приоритет над флагами
	if envExp := os.Getenv("TOKEN_EXPIRATION"); envExp != "" {
		if dur, err := time.ParseDuration(envExp); err == nil {
			cfg.TokenExpiration = dur
		} else {
			cfg.TokenExpiration = defaultTokenExp
		}
	}
	if cfg.TokenExpiration <= 0 {
		cfg.TokenExpiration = defaultTokenExp
	}

	return cfg
}
