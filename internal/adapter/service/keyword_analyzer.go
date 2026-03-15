package service

import (
	"context"
	"fmt"
	"strings"

	"tgbot/internal/domain"
)

// KeywordAnalyzer анализирует вакансии на основе ключевых слов
type KeywordAnalyzer struct {
	positiveKeywords []string
	negativeKeywords []string
}

// NewKeywordAnalyzer создает новый анализатор
func NewKeywordAnalyzer() *KeywordAnalyzer {
	return &KeywordAnalyzer{
		positiveKeywords: []string{
			"senior", "lead", "опытный", "опыт 5+", "опыт 4+",
			"высокая зарплата", "от 200", "от 300", "от 400", "$", "€",
			"удаленно", "remote", "мгу", "яндекс", "сбер",
			"интересные задачи", "менторство", "рост", "development",
			"golang", "go developer", "python", "java", "rust",
			"c++", "backend", "архитектор", "стартап", "venture",
		},
		negativeKeywords: []string{
			"junior", "стажер", "trainee", "низкая зарплата",
			"по договоренности", "не обсуждается", "зарплата договор",
			"no experience", "junior only", "начальный",
		},
	}
}

// AnalyzeVacancy анализирует вакансию
func (ka *KeywordAnalyzer) AnalyzeVacancy(ctx context.Context, vacancy *domain.Vacancy) (*domain.AnalysisResult, error) {
	text := strings.ToLower(vacancy.Content + " " + vacancy.Title)

	positiveCount := 0
	for _, keyword := range ka.positiveKeywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			positiveCount++
		}
	}

	negativeCount := 0
	for _, keyword := range ka.negativeKeywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			negativeCount++
		}
	}

	result := &domain.AnalysisResult{
		VacancyID: vacancy.ID,
	}

	// Логика принятия решения
	result.IsSuitable = positiveCount > negativeCount && positiveCount > 0
	result.Confidence = 0.7

	if positiveCount > 0 {
		result.Reason = fmt.Sprintf("Found %d positive and %d negative indicators. This vacancy matches your criteria.", positiveCount, negativeCount)
	} else if negativeCount > 0 {
		result.Reason = fmt.Sprintf("Found %d negative indicators. This vacancy doesn't match your criteria.", negativeCount)
	} else {
		result.Reason = "No specific indicators found. Review manually."
		result.IsSuitable = false
	}

	return result, nil
}
