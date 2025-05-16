package api

import (
	"net/http" // Добавляем импорт

	"github.com/gin-gonic/gin"

	"salyqai/internal/calculation"
	"salyqai/internal/services"
)

// SetupRouter - обновленная функция
func SetupRouter(calc *calculation.Calculator, ai services.AIService) *gin.Engine {
	router := gin.Default()

	// CORS Middleware (оставляем как есть)
	router.Use(func(c *gin.Context) {
		// ... (код CORS без изменений) ...
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Разрешить все источники
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Создаем обработчики
	calcHandler := NewCalculationHandler(calc, ai) // Старый обработчик для формы
	chatHandler := NewChatHandler(ai)              // Новый обработчик для чата

	// Группа роутов для API v1
	apiV1 := router.Group("/api/v1")
	{
		// --- НОВЫЙ РОУТ ЧАТА ---
		apiV1.POST("/chat", chatHandler.HandleChatMessage)

		// --- СТАРЫЙ РОУТ ДЛЯ ФОРМЫ (можно переименовать) ---
		apiV1.POST("/calculate_from_form", calcHandler.HandleCalculateSimplified) // Переименован?
	}

	// Health-check (оставляем)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	return router
}
