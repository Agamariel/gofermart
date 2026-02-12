package config

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Сохраняем оригинальные значения для восстановления
	originalArgs := os.Args
	originalEnv := make(map[string]string)
	envVars := []string{"RUN_ADDRESS", "DATABASE_URI", "ACCRUAL_SYSTEM_ADDRESS", "JWT_SECRET", "TOKEN_EXPIRATION"}
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Восстанавливаем после всех тестов
	defer func() {
		os.Args = originalArgs
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	tests := []struct {
		name         string
		args         []string
		envVars      map[string]string
		wantAddress  string
		wantDBURI    string
		wantAccrual  string
		wantSecret   string
		wantTokenExp time.Duration
	}{
		{
			name:         "default values",
			args:         []string{"cmd"},
			envVars:      map[string]string{},
			wantAddress:  "localhost:8080",
			wantDBURI:    "",
			wantAccrual:  "",
			wantSecret:   "default-secret-change-in-production",
			wantTokenExp: 24 * time.Hour,
		},
		{
			name:         "flags only",
			args:         []string{"cmd", "-a", "localhost:9090", "-d", "postgresql://db", "-r", "http://accrual", "-t", "36h"},
			envVars:      map[string]string{},
			wantAddress:  "localhost:9090",
			wantDBURI:    "postgresql://db",
			wantAccrual:  "http://accrual",
			wantSecret:   "default-secret-change-in-production",
			wantTokenExp: 36 * time.Hour,
		},
		{
			name: "env only",
			args: []string{"cmd"},
			envVars: map[string]string{
				"RUN_ADDRESS":            "localhost:7070",
				"DATABASE_URI":           "postgresql://envdb",
				"ACCRUAL_SYSTEM_ADDRESS": "http://envaccrual",
				"JWT_SECRET":             "env-secret",
				"TOKEN_EXPIRATION":       "48h",
			},
			wantAddress:  "localhost:7070",
			wantDBURI:    "postgresql://envdb",
			wantAccrual:  "http://envaccrual",
			wantSecret:   "env-secret",
			wantTokenExp: 48 * time.Hour,
		},
		{
			name: "env overrides flags",
			args: []string{"cmd", "-a", "localhost:9090", "-d", "postgresql://flagdb", "-r", "http://flagaccrual", "-t", "72h"},
			envVars: map[string]string{
				"RUN_ADDRESS":            "localhost:7070",
				"DATABASE_URI":           "postgresql://envdb",
				"ACCRUAL_SYSTEM_ADDRESS": "http://envaccrual",
				"TOKEN_EXPIRATION":       "12h",
			},
			wantAddress:  "localhost:7070",
			wantDBURI:    "postgresql://envdb",
			wantAccrual:  "http://envaccrual",
			wantSecret:   "default-secret-change-in-production",
			wantTokenExp: 12 * time.Hour,
		},
		{
			name: "partial env",
			args: []string{"cmd", "-a", "localhost:9090", "-d", "postgresql://flagdb"},
			envVars: map[string]string{
				"RUN_ADDRESS": "localhost:7070",
				"JWT_SECRET":  "custom-secret",
			},
			wantAddress:  "localhost:7070",
			wantDBURI:    "postgresql://flagdb",
			wantAccrual:  "",
			wantSecret:   "custom-secret",
			wantTokenExp: 24 * time.Hour,
		},
		{
			name: "invalid token expiration env fallback",
			args: []string{"cmd"},
			envVars: map[string]string{
				"TOKEN_EXPIRATION": "invalid",
			},
			wantAddress:  "localhost:8080",
			wantDBURI:    "",
			wantAccrual:  "",
			wantSecret:   "default-secret-change-in-production",
			wantTokenExp: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем env переменные
			for _, key := range envVars {
				os.Unsetenv(key)
			}

			// Устанавливаем env переменные для теста
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Устанавливаем аргументы командной строки
			os.Args = tt.args

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Загружаем конфигурацию
			cfg := Load()

			// Проверяем результаты
			if cfg.RunAddress != tt.wantAddress {
				t.Errorf("RunAddress = %v, want %v", cfg.RunAddress, tt.wantAddress)
			}
			if cfg.DatabaseURI != tt.wantDBURI {
				t.Errorf("DatabaseURI = %v, want %v", cfg.DatabaseURI, tt.wantDBURI)
			}
			if cfg.AccrualSystemAddress != tt.wantAccrual {
				t.Errorf("AccrualSystemAddress = %v, want %v", cfg.AccrualSystemAddress, tt.wantAccrual)
			}
			if cfg.JWTSecret != tt.wantSecret {
				t.Errorf("JWTSecret = %v, want %v", cfg.JWTSecret, tt.wantSecret)
			}
			if cfg.TokenExpiration != tt.wantTokenExp {
				t.Errorf("TokenExpiration = %v, want %v", cfg.TokenExpiration, tt.wantTokenExp)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Очищаем env
	envVars := []string{"RUN_ADDRESS", "DATABASE_URI", "ACCRUAL_SYSTEM_ADDRESS", "JWT_SECRET", "TOKEN_EXPIRATION"}
	originalEnv := make(map[string]string)
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg := Load()

	if cfg.RunAddress != "localhost:8080" {
		t.Errorf("Expected default RunAddress 'localhost:8080', got %v", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "" {
		t.Errorf("Expected empty DatabaseURI, got %v", cfg.DatabaseURI)
	}
	if cfg.TokenExpiration != 24*time.Hour {
		t.Errorf("Expected TokenExpiration 24h, got %v", cfg.TokenExpiration)
	}
	if cfg.JWTSecret != "default-secret-change-in-production" {
		t.Errorf("Expected default JWT secret, got %v", cfg.JWTSecret)
	}
}

func TestJWTSecretPriority(t *testing.T) {
	originalEnv := os.Getenv("JWT_SECRET")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("JWT_SECRET")
		} else {
			os.Setenv("JWT_SECRET", originalEnv)
		}
	}()

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name       string
		envSecret  string
		wantSecret string
	}{
		{
			name:       "env JWT secret set",
			envSecret:  "custom-jwt-secret",
			wantSecret: "custom-jwt-secret",
		},
		{
			name:       "env JWT secret empty",
			envSecret:  "",
			wantSecret: "default-secret-change-in-production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSecret == "" {
				os.Unsetenv("JWT_SECRET")
			} else {
				os.Setenv("JWT_SECRET", tt.envSecret)
			}

			os.Args = []string{"cmd"}
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg := Load()

			if cfg.JWTSecret != tt.wantSecret {
				t.Errorf("JWTSecret = %v, want %v", cfg.JWTSecret, tt.wantSecret)
			}
		})
	}
}
