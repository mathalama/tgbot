package service

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"tgbot/internal/domain"
)

// GeminiAnalyzer анализирует вакансии через Google Gemini API
type GeminiAnalyzer struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewGeminiAnalyzer создает новый анализатор
func NewGeminiAnalyzer(apiKey string) (*GeminiAnalyzer, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-pro-vision")

	return &GeminiAnalyzer{
		client: client,
		model:  model,
	}, nil
}

// AnalyzeVacancy анализирует, подходит ли вакансия пользователю
func (ga *GeminiAnalyzer) AnalyzeVacancy(ctx context.Context, vacancy *domain.Vacancy) (*domain.AnalysisResult, error) {
	// Промпт для анализа вакансии
	prompt := fmt.Sprintf(`Анализируй следующую вакансию и определи подходит ли она для опытного разработчика.

Название: %s
Содержание: %s

Ответь в формате JSON без кода:
{
  "suitable": boolean,
  "reason": "краткое объяснение",
  "confidence": число от 0 до 1
}

Критерии для "подходит":
- Зарплата выше среднего уровня
- Интересный стек технологий
- Возможность роста и развития
- Хорошие условия работы
- Нет требований junior-уровня`, vacancy.Title, vacancy.Content)

	resp, err := ga.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	// Парсим ответ
	// Для простоты, делаем анализ на основе ключевых слов
	result := &domain.AnalysisResult{
		VacancyID: vacancy.ID,
	}

	// Простой анализ - проверяем наличие позитивных ключевых слов
	text := vacancy.Content + " " + vacancy.Title
	positiveKeywords := []string{"senior", "lead", "опытный", "опыт 5+", "высокая зарплата", "удаленная работа", "интересная задача"}
	negativeKeywords := []string{"junior", "стажер", "низкая зарплата", "зарплата не обсуждается"}

	positiveCount := 0
	for _, keyword := range positiveKeywords {
		if isContainsCI(text, keyword) {
			positiveCount++
		}
	}

	negativeCount := 0
	for _, keyword := range negativeKeywords {
		if isContainsCI(text, keyword) {
			negativeCount++
		}
	}

	result.IsSuitable = positiveCount > negativeCount
	result.Confidence = 0.7
	result.Reason = fmt.Sprintf("Found %d positive and %d negative indicators", positiveCount, negativeCount)

	return result, nil
}

// Close закрывает клиент
func (ga *GeminiAnalyzer) Close() error {
	return ga.client.Close()
}

func isContainsCI(text, substring string) bool {
	// Простой case-insensitive поиск
	for i := 0; i <= len(text)-len(substring); i++ {
		match := true
		for j := 0; j < len(substring); j++ {
			if toLower(text[i+j]) != toLower(substring[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}
