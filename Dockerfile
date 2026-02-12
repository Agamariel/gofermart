# Build stage
FROM golang:1.24-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git

WORKDIR /app

# Копирование go.mod и go.sum для кеширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /gofermart ./cmd/gophermart

# Run stage
FROM alpine:latest

# Установка ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копирование бинарника из builder stage
COPY --from=builder /gofermart .

# Открытие порта
EXPOSE 8080

# Запуск приложения
CMD ["./gofermart"]

