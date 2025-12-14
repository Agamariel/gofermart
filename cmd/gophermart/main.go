package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agamariel/gofermart/internal/config"
)

func main() {
	cfg := config.Load()
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// Инициализация приложения
	app, err := NewApp(rootCtx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Запуск сервера в отдельной горутине
	go func() {
		if err := app.Start(rootCtx); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	rootCancel()

	// Остановка приложения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
