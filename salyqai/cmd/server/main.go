package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"salyqai/internal/api"         // Путь к вашему API модулю
	"salyqai/internal/calculation" // Путь к вашему модулю расчета
	"salyqai/internal/config"      // Путь к вашей конфигурации
	"salyqai/internal/services"    // Путь к вашему AI сервису
)

func main() {
	// 1. Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		// В LoadConfig мы пока не возвращаем ошибку, а только логируем,
		// но если бы возвращали, здесь был бы log.Fatalf
		log.Printf("Warning: Failed to load config: %v\n", err)
		// Продолжаем работу, т.к. AI сервис может работать в режиме заглушки
	}

	// 2. Инициализация зависимостей
	calculator := calculation.NewCalculator()
	aiService, err := services.NewGeminiService(cfg)
	if err != nil {
		// Если создание AI сервиса КРИТИЧНО и мы НЕ хотим заглушку,
		// то здесь нужно прервать выполнение:
		// log.Fatalf("Failed to initialize AI service: %v", err)
		log.Printf("Warning: Failed to initialize full AI service: %v. Using NoOp service if key was missing.\n", err)
	}
	// Убедимся, что закрываем клиент AI при выходе
	defer aiService.Close()

	// 3. Настройка роутера Gin
	router := api.SetupRouter(calculator, aiService)
	log.Println("Router setup complete.")

	// 4. Запуск сервера (с Graceful Shutdown)
	port := os.Getenv("PORT") // Порт для Heroku, Render и т.д.
	if port == "" {
		port = "8080" // Стандартный порт для локальной разработки
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second, // Добавим таймаут для безопасности
	}

	log.Printf("Starting server on %s\n", addr)

	// Запускаем сервер в горутине, чтобы не блокировать основной поток
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Ожидаем сигнал для graceful shutdown
	quit := make(chan os.Signal, 1)
	// SIGINT: Ctrl+C
	// SIGTERM: Стандартный сигнал для завершения от систем управления (Docker, Kubernetes)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Блокируемся, пока не получим сигнал
	log.Println("Shutting down server...")

	// Даем 5 секунд на завершение текущих запросов
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
