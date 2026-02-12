.PHONY: build run test clean help
.PHONY: docker-build docker-up docker-down docker-restart docker-logs docker-clean
.PHONY: dev dev-stop test-integration test-docker

# Основные команды
build: ## Сборка бинарника
	go build -o bin/gophermart.exe ./cmd/gophermart

run: ## Запуск приложения
	go run ./cmd/gophermart

deps: ## Установка зависимостей
	go mod download
	go mod tidy

test: ## Запуск unit тестов
	go test -v -race -short ./...

test-coverage: ## Запуск тестов с покрытием
	go test -v -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "HTML отчет создан: coverage.html"

clean: ## Очистка
	rm -rf bin/
	go clean

fmt: ## Форматирование кода
	go fmt ./...

lint: ## Проверка кода (требует golangci-lint)
	golangci-lint run

# Docker команды
docker-build: ## Сборка Docker образа
	docker-compose build

docker-up: ## Запуск всех сервисов в Docker
	docker-compose up -d

docker-down: ## Остановка Docker контейнеров
	docker-compose down

docker-restart: ## Перезапуск приложения в Docker
	docker-compose restart app

docker-logs: ## Просмотр логов приложения
	docker-compose logs -f app

docker-clean: ## Полная очистка Docker (включая volumes)
	docker-compose down -v
	docker system prune -f

# Разработка
dev: ## Запуск БД в Docker + локальное приложение (для Windows PowerShell запустите вручную)
	docker-compose up -d postgres
	@echo "Waiting for PostgreSQL..."
	@sleep 3
	DATABASE_URI=postgresql://postgres:postgres@localhost:5432/gofermart?sslmode=disable go run ./cmd/gophermart

dev-stop: ## Остановка БД
	docker-compose down

# Тестирование
test-integration: ## Integration тесты с Docker БД
	docker-compose up -d postgres
	@echo "Waiting for PostgreSQL..."
	@sleep 3
	DATABASE_URI=postgresql://postgres:postgres@localhost:5432/gofermart?sslmode=disable go test -v -tags=integration ./...
	docker-compose down

test-docker: ## Тестирование в Docker
	docker-compose up --build --abort-on-container-exit --exit-code-from app

# Help
help: ## Показать эту справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
