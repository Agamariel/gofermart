package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestMigrationsEmbedded(t *testing.T) {
	// Проверяем наличие файлов
	entries, err := embedMigrations.ReadDir(".")
	if err != nil {
		t.Fatalf("Failed to read embedded migrations: %v", err)
	}

	if len(entries) == 0 {
		t.Error("No migration files found in embedFS")
	}

	// Проверяем, что есть хотя бы одна SQL миграция
	foundSQL := false
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 4 && entry.Name()[len(entry.Name())-4:] == ".sql" {
			foundSQL = true
			t.Logf("Found migration: %s", entry.Name())
		}
	}

	if !foundSQL {
		t.Error("No .sql migration files found")
	}
}

func TestRunWithInvalidDB(t *testing.T) {
	// Тест с невалидным подключением
	db, err := sql.Open("pgx", "invalid://connection")
	if err != nil {
		t.Skipf("Cannot create test DB connection: %v", err)
	}
	defer db.Close()

	// Run должен вернуть ошибку для невалидного подключения
	err = Run(db)
	if err == nil {
		t.Error("Expected error for invalid DB connection, got nil")
	}
}

func TestVersionWithInvalidDB(t *testing.T) {
	// Тест с невалидным подключением
	db, err := sql.Open("pgx", "invalid://connection")
	if err != nil {
		t.Skipf("Cannot create test DB connection: %v", err)
	}
	defer db.Close()

	// Version должен вернуть ошибку для невалидного подключения
	_, err = Version(db)
	if err == nil {
		t.Error("Expected error for invalid DB connection, got nil")
	}
}
