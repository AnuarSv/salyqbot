package services

import (
	"context"
	"encoding/json" // <<-- Добавим для парсинга JSON ответа классификатора
	"errors"        // <<-- Добавим для кастомных ошибок
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"salyqai/internal/config"
	"salyqai/internal/models"
)

const (
	geminiModelName         = "gemini-1.5-flash-latest"
	defaultTimeout          = 30 * time.Second
	mzp2024Services float64 = 85000 // МЗП для использования в промптах
)

// IntentRecognitionResult - структура для ответа от классификатора
type IntentRecognitionResult struct {
	Intent   string            `json:"intent"`
	Entities map[string]string `json:"entities"` // Можно будет использовать позже
}

var ErrIntentRecognitionFailed = errors.New("intent recognition failed") // Ошибка для классификации

// AIService - обновленный интерфейс
type AIService interface {
	// Классифицирует намерение пользователя
	ClassifyIntent(ctx context.Context, userMessage string) (*IntentRecognitionResult, error)
	// Отвечает на общий вопрос пользователя
	GenerateGeneralAnswer(ctx context.Context, userMessage string, intentHint string) (string, error)
	// Объясняет результаты расчета (старый метод)
	GenerateExplanation(ctx context.Context, result models.CalculationResult) (string, error)
	Close()
}

// GeminiService - реализация AIService
type GeminiService struct {
	client *genai.Client
	cfg    *config.Config
}

// NewGeminiService - конструктор (без изменений)
func NewGeminiService(cfg *config.Config) (AIService, error) {
	// ... (код конструктора без изменений) ...
	if cfg.GeminiAPIKey == "" {
		log.Println("WARNING: Gemini API Key is not configured. AI explanations will be disabled.")
		return &NoOpAIService{}, nil
	}
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		log.Printf("ERROR: Failed to create Gemini client: %v\n", err)
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}
	log.Println("Gemini client created successfully.")
	return &GeminiService{
		client: client,
		cfg:    cfg,
	}, nil
}

// --- Новые Методы ---

// ClassifyIntent классифицирует намерение пользователя
func (s *GeminiService) ClassifyIntent(ctx context.Context, userMessage string) (*IntentRecognitionResult, error) {
	model := s.client.GenerativeModel(geminiModelName)
	// Важно: Указываем модели, чтобы она отвечала ТОЛЬКО JSON
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockOnlyHigh, // Менее строгие настройки безопасности для классификатора
		},
		// Можно добавить другие категории, если нужно
	}
	// model.GenerationConfig = // Можно настроить температуру и т.д.

	prompt := buildIntentPrompt(userMessage)

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	log.Println("Sending Intent Classification prompt to Gemini:", prompt)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("ERROR: Failed to generate content for intent classification: %v\n", err)
		return nil, fmt.Errorf("%w: %v", ErrIntentRecognitionFailed, err)
	}

	rawJson := extractTextFromResponse(resp)
	log.Println("Received raw classification response from Gemini:", rawJson)

	// Пытаемся распарсить JSON
	var result IntentRecognitionResult
	// Убираем возможные ```json и ``` маркеры, которые иногда добавляет Gemini
	cleanedJson := strings.TrimSpace(rawJson)
	cleanedJson = strings.TrimPrefix(cleanedJson, "```json")
	cleanedJson = strings.TrimSuffix(cleanedJson, "```")
	cleanedJson = strings.TrimSpace(cleanedJson)

	if err := json.Unmarshal([]byte(cleanedJson), &result); err != nil {
		log.Printf("ERROR: Failed to unmarshal intent classification JSON response: %v. Raw response: %s\n", err, rawJson)
		// Возвращаем дефолтное намерение или ошибку
		return &IntentRecognitionResult{Intent: "unknown", Entities: nil}, fmt.Errorf("%w: failed to parse JSON: %v", ErrIntentRecognitionFailed, err)
	}

	if result.Intent == "" {
		log.Println("WARNING: Intent classification returned empty intent.")
		return &IntentRecognitionResult{Intent: "unknown", Entities: nil}, nil // Не ошибка, но не распознано
	}

	log.Printf("Intent classified as: %s, Entities: %v\n", result.Intent, result.Entities)
	return &result, nil
}

// GenerateGeneralAnswer отвечает на общий вопрос
func (s *GeminiService) GenerateGeneralAnswer(ctx context.Context, userMessage string, intentHint string) (string, error) {
	model := s.client.GenerativeModel(geminiModelName)
	// Можно настроить SafetySettings и GenerationConfig по аналогии, если нужно

	prompt := buildGeneralAnswerPrompt(userMessage, intentHint)

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	log.Println("Sending General Answer prompt to Gemini:", prompt)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("ERROR: Failed to generate general answer: %v\n", err)
		return "Извините, произошла ошибка при генерации ответа.", fmt.Errorf("general answer generation failed: %w", err)
	}

	answer := extractTextFromResponse(resp)
	log.Println("Received general answer from Gemini:", answer)

	if answer == "" {
		return "Извините, не могу сейчас ответить на этот вопрос.", nil
	}
	return answer, nil
}

// --- Промпты для новых методов ---

func buildIntentPrompt(userMessage string) string {
	// Промпт для классификации намерения
	// TODO: Уточнить список интентов и формат ответа
	return fmt.Sprintf(`АНАЛИЗ ЗАПРОСА:
Ты – ИИ-анализатор для налогового помощника SalyqAI (Казахстан, Упрощенка для ИП).
Твоя задача: проанализировать сообщение пользователя и определить его основное НАМЕРЕНИЕ (intent).
Возможные намерения:
- "calculate_tax": Пользователь хочет рассчитать налоги (явно или неявно).
- "ask_deadline": Вопрос о сроках уплаты или сдачи отчетности.
- "ask_limit": Вопрос о лимитах дохода для Упрощенки.
- "ask_kkm": Вопрос о кассовом аппарате (ККМ/онлайн-касса).
- "ask_social_payments": Вопрос о социальных платежах (ОПВ, СО, ВОСМС).
- "greeting": Просто приветствие или начало разговора.
- "general_question": Другой вопрос по теме Упрощенки, не подходящий под категории выше.
- "off_topic": Вопрос не по теме налогов ИП на Упрощенке в РК.
- "unknown": Намерение неясно.

Извлеки также СУЩНОСТИ (entities), если они упоминаются: "revenue" (сумма дохода), "period" (упомянутый период).

ОТВЕТЬ ТОЛЬКО В ФОРМАТЕ JSON и никак иначе:
{
  "intent": "НАЗВАНИЕ_НАМЕРЕНИЯ",
  "entities": {
    "revenue": "УПОМЯНУТАЯ_СУММА_ИЛИ_null",
    "period": "УПОМЯНУТЫЙ_ПЕРИОД_ИЛИ_null"
  }
}

Сообщение пользователя: "%s"`, userMessage)
}

func buildGeneralAnswerPrompt(userMessage string, intentHint string) string {
	// Промпт для ответа на общие вопросы
	// Можно использовать intentHint для уточнения контекста
	return fmt.Sprintf(`Ты – SalyqAI, дружелюбный и компетентный ИИ-ассистент для индивидуальных предпринимателей (ИП) в Казахстане, работающих на Упрощенке (Форма 910) в 2024 году.
Твоя задача – ответить на вопрос пользователя кратко, ясно и на основе актуальных правил Налогового и Социального кодексов РК, а также Закона об ОСМС.
Не выдумывай информацию. Если не знаешь точного ответа, лучше скажи об этом. Не давай финансовых или юридических советов.

(Контекст: Пользователь, вероятно, спрашивает о '%s')

Вопрос пользователя: "%s"

Твой ответ:`, intentHint, userMessage)
}

// --- Старый метод и промпт для объяснения расчета (оставляем как есть) ---

// GenerateExplanation генерирует объяснение для результатов расчета
func (s *GeminiService) GenerateExplanation(ctx context.Context, result models.CalculationResult) (string, error) {
	model := s.client.GenerativeModel(geminiModelName)
	prompt := s.buildExplanationPrompt(result) // Используем старый, доработанный промпт

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	log.Println("Sending Explanation prompt to Gemini:", prompt)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("ERROR: Failed to generate explanation: %v\n", err)
		return "Извините, не удалось сгенерировать объяснение расчета.", fmt.Errorf("explanation generation failed: %w", err)
	}

	explanation := extractTextFromResponse(resp)
	log.Println("Received explanation from Gemini:", explanation)

	if explanation == "" {
		return "Извините, получено пустое объяснение расчета от AI.", nil
	}
	return explanation, nil
}

// buildExplanationPrompt - переименовали старый buildPrompt
func (s *GeminiService) buildExplanationPrompt(result models.CalculationResult) string {
	// !!! ВСТАВЬТЕ СЮДА ВАШ ПОСЛЕДНИЙ ДОРАБОТАННЫЙ ПРОМПТ ДЛЯ ОБЪЯСНЕНИЯ РАСЧЕТОВ !!!
	// (Тот, который мы делали для случая с доходом 32 тг)
	// Я вставлю его структуру, но проверьте текст внимательно.
	promptTemplate := `Ты – дружелюбный и понятный налоговый помощник SalyqAI для индивидуальных предпринимателей (ИП) в Казахстане, работающих на Упрощенном режиме налогообложения (Упрощенка, форма 910) без работников.

Твоя задача – объяснить простыми словами результаты расчета налогов и социальных платежей за полугодие, используя ТОЛЬКО те цифры, которые предоставлены ниже.

Критически важно:
1.  НЕ пытайся самостоятельно пересчитывать налоги или платежи. Доверяй предоставленным цифрам.
2.  НЕ округляй и НЕ изменяй предоставленные цифры дохода или расчетов в своем объяснении.
3.  Объясняй значение КАЖДОЙ предоставленной цифры.
4.  Если видишь, что соц. платежи большие по сравнению с доходом, объясни, что они рассчитаны от минимальной базы (МЗП) и являются обязательными.
5.  Не давай финансовых советов, только объясняй расчеты и правила. Будь кратким, но ясным.

Вот ТОЧНЫЕ данные для объяснения:
*   Доход за полугодие: %.2f тенге
*   Количество месяцев работы ИП в полугодии: %d
*   Итого налог по Упрощенке (3%%): %.2f тенге, из них:
    *   Индивидуальный подоходный налог (ИПН) к уплате: %.2f тенге (это 1.5%% от дохода)
    *   Социальный налог (СН) к уплате: %.2f тенге (это 1.5%% от дохода, уменьшенные на сумму СО, но не меньше нуля)
*   Итого Социальные платежи за ИП (рассчитаны за %d месяцев): %.2f тенге. Эти платежи обязательны для ИП и рассчитываются от установленных баз (в данном случае, минимальных), даже если доход был низким. Они включают:
    *   Обязательные пенсионные взносы (ОПВ): %.2f тенге (рассчитаны как 10%% от минимальной базы МЗП=%.0f тг/мес * %d мес.)
    *   Социальные отчисления (СО): %.2f тенге (рассчитаны как 3.5%% от (МЗП=%.0f тг/мес минус ОПВ за месяц) * %d мес.)
    *   Взносы на мед. страхование (ВОСМС): %.2f тенге (рассчитаны как 5%% от фиксированной базы 1.4*МЗП=%.0f тг/мес * %d мес.)
*   Ваш доход составляет %.1f%% от разрешенного лимита на Упрощенке (%.0f тенге в 2024 году).

Кратко объясни значение каждой суммы (ИПН, СН, ОПВ, СО, ВОСМС), используя предоставленные цифры. Подчеркни, почему СН может быть равен нулю.

Обязательно укажи крайние сроки:
*   Уплаты ОПВ, СО, ВОСМС: ежемесячно до 25 числа следующего месяца.
*   Уплаты ИПН и СН: до 25 августа (за 1 полугодие) или до 25 февраля (за 2 полугодие).
*   Сдачи декларации (форма 910): до 15 августа (за 1 полугодие) или до 15 февраля (за 2 полугодие).

Также упомяни важные "подводные камни" для Упрощенки:
*   Необходимость использования Онлайн-ККМ при приеме наличных денег или оплате картой.
*   Важность не превышать лимит дохода (%.0f тенге в 2024 году), чтобы остаться на Упрощенке. %s
*   Напомни про ежемесячную уплату обязательных социальных платежей (ОПВ, СО, ВОСМС), рассчитанных от МЗП, даже если доход маленький или его нет.

Говори просто, понятно и ободряюще. Используй точные цифры из данных выше.`
	// ... (остальная часть функции с fmt.Sprintf, использующая mzp2024Services) ...
	limitWarningText := ""
	if len(result.Warnings) > 0 {
		limitWarningText = strings.Join(result.Warnings, " ")
	}
	mzpBase := mzp2024Services
	vosmsBaseMonthlyValue := 1.4 * mzpBase

	return fmt.Sprintf(promptTemplate,
		result.InputData.Revenue,      // Доход
		result.InputData.MonthsWorked, // Месяцев работы
		result.TotalTax,               // Итого налог
		result.IPN,                    // ИПН
		result.SN,                     // СН
		result.InputData.MonthsWorked, // Месяцев работы (для соц. платежей)
		result.TotalSocial,            // Итого соц. платежи
		result.OPV,                    // ОПВ
		mzpBase,                       // База МЗП для ОПВ
		result.InputData.MonthsWorked, // Месяцев для ОПВ
		result.SO,                     // СО
		mzpBase,                       // База МЗП для СО
		result.InputData.MonthsWorked, // Месяцев для СО
		result.VOSMS,                  // ВОСМС
		vosmsBaseMonthlyValue,         // База для ВОСМС
		result.InputData.MonthsWorked, // Месяцев для ВОСМС
		result.LimitPercentage,        // % от лимита
		result.RevenueLimitValue,      // Значение лимита дохода
		result.RevenueLimitValue,      // Значение лимита (для подводных камней)
		limitWarningText,              // Предупреждения о лимите
	)
}

// --- Остальные функции (extractTextFromResponse, Close) ---
// ... (без изменений) ...
func extractTextFromResponse(resp *genai.GenerateContentResponse) string {
	// ... (код без изменений) ...
	var builder strings.Builder
	if resp != nil && resp.Candidates != nil {
		for _, cand := range resp.Candidates {
			if cand.Content != nil {
				for _, part := range cand.Content.Parts {
					if text, ok := part.(genai.Text); ok {
						builder.WriteString(string(text))
					}
				}
			}
		}
	}
	return builder.String()
}

func (s *GeminiService) Close() {
	if s.client != nil {
		s.client.Close()
	}
}

// --- Заглушка NoOpAIService (нужно добавить новые методы) ---
type NoOpAIService struct{}

func (s *NoOpAIService) ClassifyIntent(ctx context.Context, userMessage string) (*IntentRecognitionResult, error) {
	log.Println("AI Service is disabled (No API Key). Returning default intent.")
	// Возвращаем намерение, которое не вызовет ошибку, например 'unknown' или 'general_question'
	return &IntentRecognitionResult{Intent: "general_question", Entities: nil}, nil
}

func (s *NoOpAIService) GenerateGeneralAnswer(ctx context.Context, userMessage string, intentHint string) (string, error) {
	log.Println("AI Service is disabled (No API Key). Returning default message.")
	return "AI сервис временно недоступен для ответа на общие вопросы.", nil
}

func (s *NoOpAIService) GenerateExplanation(ctx context.Context, result models.CalculationResult) (string, error) {
	log.Println("AI Service is disabled (No API Key). Returning default message.")
	return "AI-объяснение расчета временно недоступно.", nil
}

func (s *NoOpAIService) Close() {}
