package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GeminiAPIKey string
	// Можно добавить другие параметры, если нужны
}

// LoadConfig загружает конфигурацию из переменных окружения
func LoadConfig() (*Config, error) {
	// Загружаем .env файл (игнорируем ошибку, если файла нет - актуально для деплоя)
	_ = godotenv.Load()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("WARNING: GEMINI_API_KEY environment variable not set.")
		// Можно вернуть ошибку, если ключ обязателен для работы
		// return nil, errors.New("GEMINI_API_KEY environment variable not set")
	}

	return &Config{
		GeminiAPIKey: apiKey,
	}, nil
}

// GetDisclaimer возвращает текст дисклеймера
func GetDisclaimer() string {
	return "ВНИМАНИЕ! Этот инструмент предоставляет расчеты в ознакомительных целях и находится в стадии разработки. Данные могут быть неточными или не учитывать все детали вашей ситуации. Сервис не является официальной налоговой консультацией и не заменяет профессионального бухгалтера. Ответственность за правильность и своевременность уплаты налогов лежит на вас. Всегда сверяйте информацию с официальными источниками (Налоговый Кодекс РК, kgd.gov.kz) и/или консультируйтесь со специалистом."
}
