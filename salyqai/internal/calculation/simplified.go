package calculation

import (
	"math"

	"salyqai/internal/models" // Убедись, что путь к твоим моделям правильный
)

// Константы для расчета на 2024 год (КАЗАХСТАН)
const (
	mrp2024 float64 = 3692  // Месячный расчетный показатель
	mzp2024 float64 = 85000 // Минимальная заработная плата

	simplifiedRegimeRate float64 = 0.03  // Ставка Упрощенки (3%)
	ipnRate              float64 = 0.015 // Ставка ИПН (1.5% из 3%)
	snRate               float64 = 0.015 // Ставка СН (1.5% из 3%)

	revenueLimitMRP float64 = 24038 // Лимит дохода в МРП за полугодие

	// Базы для социальных платежей ИП за себя (в месяц)
	opvDeclaredIncomeBaseMin float64 = 1 * mzp2024  // База для ОПВ (мин 1 МЗП)
	opvDeclaredIncomeBaseMax float64 = 50 * mzp2024 // База для ОПВ (макс 50 МЗП)
	opvRate                  float64 = 0.10         // Ставка ОПВ (10%)

	soDeclaredIncomeBaseMin float64 = 1 * mzp2024 // База для СО (мин 1 МЗП)
	soDeclaredIncomeBaseMax float64 = 7 * mzp2024 // База для СО (макс 7 МЗП)
	soRate                  float64 = 0.035       // Ставка СО (3.5%)

	vosmsBaseMultiplier float64 = 1.4  // Множитель базы ВОСМС
	vosmsRate           float64 = 0.05 // Ставка ВОСМС (5%)
)

// Calculator - структура для выполнения расчетов (пока простая)
type Calculator struct {
	// Здесь могут быть зависимости в будущем (например, доступ к кэшу ставок)
}

// NewCalculator - конструктор для Calculator
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculateSimplifiedTax выполняет расчет налогов и платежей для Упрощенки
func (c *Calculator) CalculateSimplifiedTax(req models.TaxCalculationRequest) models.CalculationResult {
	result := models.CalculationResult{
		InputData: req, // Сохраняем входные данные
		Warnings:  []string{},
	}

	// 1. Рассчитываем лимит дохода на полугодие
	revenueLimit := revenueLimitMRP * mrp2024
	result.LimitPercentage = (req.Revenue / revenueLimit) * 100
	result.RevenueLimitValue = revenueLimit
	if req.Revenue > revenueLimit {
		result.Warnings = append(result.Warnings, "ПРЕДУПРЕЖДЕНИЕ: Ваш доход превышает лимит для Упрощенного режима!")
	} else if result.LimitPercentage > 80 { // Предупреждаем о приближении к лимиту
		result.Warnings = append(result.Warnings, "ВНИМАНИЕ: Ваш доход приближается к лимиту для Упрощенного режима.")
	}

	// 2. Расчет Социальных платежей ИП за себя (за 1 месяц)
	// Используем 1 МЗП как "заявленный доход" для ИП на Упрощенке (самый частый случай)
	// Важно: В реальной жизни ИП может заявить доход больше для ОПВ/СО, но для MVP берем минимум.
	declaredIncomeMonthly := math.Max(mzp2024, 0) // Берем МЗП как базу, но не меньше 0

	// ОПВ (Пенсионные)
	opvBaseMonthly := math.Max(opvDeclaredIncomeBaseMin, math.Min(declaredIncomeMonthly, opvDeclaredIncomeBaseMax)) // Учитываем мин/макс базу
	opvMonthly := opvBaseMonthly * opvRate

	// СО (Соцотчисления)
	// База для СО = Заявленный доход (с учетом мин/макс для СО) - ОПВ
	soBaseMonthly := math.Max(soDeclaredIncomeBaseMin, math.Min(declaredIncomeMonthly, soDeclaredIncomeBaseMax))
	soMonthly := math.Max(0, (soBaseMonthly-opvMonthly)*soRate) // Учитываем вычет ОПВ, СО не может быть < 0

	// ВОСМС (Медстрах) - база фиксированная
	vosmsBaseMonthly := 1.4 * mzp2024
	vosmsMonthly := vosmsBaseMonthly * vosmsRate

	// 3. Расчет Соц. платежей за весь период работы
	months := float64(req.MonthsWorked)
	result.OPV = roundToTiyn(opvMonthly * months)
	result.SO = roundToTiyn(soMonthly * months)
	result.VOSMS = roundToTiyn(vosmsMonthly * months)
	result.TotalSocial = roundToTiyn(result.OPV + result.SO + result.VOSMS)

	// 4. Расчет Налога по Упрощенке (3%)
	ipnCalculated := req.Revenue * ipnRate
	snCalculated := req.Revenue * snRate

	// 5. Корректировка Социального Налога (СН)
	// СН уменьшается на сумму СО за период, но не может быть меньше нуля
	snAdjusted := math.Max(0, snCalculated-result.SO)

	result.IPN = roundToTiyn(ipnCalculated)
	result.SN = roundToTiyn(snAdjusted)
	result.TotalTax = roundToTiyn(result.IPN + result.SN) // Итого налог к уплате

	return result
}

// roundToTiyn округляет до 2 знаков после запятой (до тиынов)
func roundToTiyn(value float64) float64 {
	return math.Round(value*100) / 100
}
