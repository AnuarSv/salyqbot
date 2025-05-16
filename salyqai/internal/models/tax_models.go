package models

// TaxCalculationRequest - Структура запроса от фронтенда
type TaxCalculationRequest struct {
	Revenue      float64 `json:"revenue" binding:"required,gte=0"`             // Доход за полугодие
	MonthsWorked int     `json:"months_worked" binding:"required,min=1,max=6"` // Кол-во месяцев работы в полугодии
	// EmployeeCount int     `json:"employee_count" binding:"gte=0"`      // Пока не используем в MVP
}

// CalculationResult - Результат расчета налогов (до объяснения AI)
type CalculationResult struct {
	IPN               float64               `json:"ipn"`              // ИПН к уплате
	SN                float64               `json:"sn"`               // Соц.налог к уплате (уже скорректированный)
	OPV               float64               `json:"opv"`              // ОПВ за ИП
	SO                float64               `json:"so"`               // Соц.отчисления за ИП
	VOSMS             float64               `json:"vosms"`            // Взносы ОСМС за ИП
	TotalTax          float64               `json:"total_tax"`        // Итого налог (ИПН + СН)
	TotalSocial       float64               `json:"total_social"`     // Итого соц. платежи (ОПВ + СО + ВОСМС)
	LimitPercentage   float64               `json:"limit_percentage"` // Процент дохода от лимита
	RevenueLimitValue float64               `json:"-"`                // Добавлено: Численное значение лимита (не отдаем в JSON)
	Warnings          []string              `json:"warnings"`         // Предупреждения (например, о лимите)
	InputData         TaxCalculationRequest `json:"-"`                // Сохраняем исходные данные для передачи в AI
}

// TaxCalculationResponse - Структура ответа API
type TaxCalculationResponse struct {
	Calculation CalculationResult `json:"calculation"` // Результаты расчета
	Explanation string            `json:"explanation"` // Объяснение от AI
	Disclaimer  string            `json:"disclaimer"`  // Дисклеймер
}
