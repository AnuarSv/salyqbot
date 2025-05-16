package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"salyqai/internal/calculation"
	"salyqai/internal/config"
	"salyqai/internal/models"
	"salyqai/internal/services"
)

// --- Структуры для API Ответов Чата ---

type ChatResponse struct {
	Type         string `json:"type"`                    // "ai_message", "show_calculation_form", "error"
	AiMessage    string `json:"ai_message,omitempty"`    // Текст ответа AI или приглашение к форме
	ErrorMessage string `json:"error_message,omitempty"` // Сообщение об ошибке
	// Можно добавить другие поля, если нужно передать что-то еще фронтенду
}

// --- Обработчик для Расчета из Формы (старый, возможно переименованный) ---

// CalculationHandler (без изменений)
type CalculationHandler struct {
	calculator *calculation.Calculator
	aiService  services.AIService
}

// NewCalculationHandler (без изменений)
func NewCalculationHandler(calc *calculation.Calculator, ai services.AIService) *CalculationHandler {
	return &CalculationHandler{
		calculator: calc,
		aiService:  ai,
	}
}

// HandleCalculateSimplified (без изменений, но вызывается роутом /calculate_from_form)
func (h *CalculationHandler) HandleCalculateSimplified(c *gin.Context) {
	// ... (весь код этого обработчика остается как был) ...
	// Он принимает точные данные, считает, вызывает GenerateExplanation, отдает JSON
	var req models.TaxCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR: Failed to bind JSON request for calculation: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный формат запроса для расчета.", "details": err.Error()})
		return
	}
	log.Printf("Received calculation request from form: %+v\n", req)
	calcResult := h.calculator.CalculateSimplifiedTax(req)
	log.Printf("Calculation result: %+v\n", calcResult)
	explanation, err := h.aiService.GenerateExplanation(c.Request.Context(), calcResult)
	if err != nil {
		log.Printf("WARNING: Failed to generate AI explanation for calculation: %v.\n", err)
	}
	response := models.TaxCalculationResponse{
		Calculation: calcResult,
		Explanation: explanation,
		Disclaimer:  config.GetDisclaimer(),
	}
	c.JSON(http.StatusOK, response)
}

// --- НОВЫЙ Обработчик для Чата ---

// ChatHandler содержит зависимости для обработчика чата
type ChatHandler struct {
	aiService services.AIService
}

// NewChatHandler создает новый экземпляр ChatHandler
func NewChatHandler(ai services.AIService) *ChatHandler {
	return &ChatHandler{
		aiService: ai,
	}
}

// ChatRequest - структура для запроса чата
type ChatRequest struct {
	Message string   `json:"message" binding:"required"`
	History []string `json:"history,omitempty"` // Опционально: история диалога
}

// HandleChatMessage обрабатывает сообщение от пользователя в чате
func (h *ChatHandler) HandleChatMessage(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR: Failed to bind JSON request for chat: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный формат запроса чата."})
		return
	}

	log.Printf("Received chat message: %s\n", req.Message)

	// 1. Определяем намерение пользователя
	intentResult, err := h.aiService.ClassifyIntent(c.Request.Context(), req.Message)
	if err != nil {
		// Если классификация не удалась, пытаемся ответить как на общий вопрос
		log.Printf("WARNING: Intent classification failed: %v. Handling as general question.\n", err)
		intentResult = &services.IntentRecognitionResult{Intent: "general_question"} // или unknown
	}

	// 2. Действуем в зависимости от намерения
	switch intentResult.Intent {
	case "calculate_tax":
		// Просим фронтенд показать форму
		log.Println("Intent: calculate_tax. Signaling frontend to show form.")
		c.JSON(http.StatusOK, ChatResponse{
			Type:      "show_calculation_form",
			AiMessage: "Хорошо, давайте рассчитаем! Чтобы всё было точно, пожалуйста, введите данные ниже:",
		})

	case "ask_deadline", "ask_limit", "ask_kkm", "ask_social_payments", "general_question", "greeting", "unknown":
		// Отвечаем на общий вопрос
		log.Printf("Intent: %s. Generating general answer.\n", intentResult.Intent)
		answer, err := h.aiService.GenerateGeneralAnswer(c.Request.Context(), req.Message, intentResult.Intent)
		if err != nil {
			log.Printf("ERROR: Failed to generate general answer: %v\n", err)
			c.JSON(http.StatusInternalServerError, ChatResponse{
				Type:         "error",
				ErrorMessage: "Извините, не удалось сгенерировать ответ.",
			})
			return
		}
		c.JSON(http.StatusOK, ChatResponse{
			Type:      "ai_message",
			AiMessage: answer,
		})

	case "off_topic":
		log.Println("Intent: off_topic.")
		c.JSON(http.StatusOK, ChatResponse{
			Type:      "ai_message",
			AiMessage: "Извините, я специализируюсь только на налогах для ИП на Упрощенке в Казахстане. По другим вопросам помочь не смогу.",
		})

	default:
		// Неизвестное намерение от классификатора (хотя мы обработали unknown выше)
		log.Printf("WARNING: Unknown intent received from classifier: %s\n", intentResult.Intent)
		c.JSON(http.StatusOK, ChatResponse{
			Type:      "ai_message",
			AiMessage: "Хм, не уверен, как на это ответить. Можете переформулировать?",
		})
	}
}
