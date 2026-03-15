package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"tgbot/internal/domain"
)

// CVAnalyzer анализирует CV и извлекает навыки
type CVAnalyzer struct {
	apiKey string
}

// SkillExtractResult результат извлечения навыков
type SkillExtractResult struct {
	Skills []SkillItem `json:"skills"`
}

type SkillItem struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	What        string `json:"what"`
	Why         string `json:"why"`
	Description string `json:"description"`
}

// NewCVAnalyzer создает новый CV анализатор
func NewCVAnalyzer(apiKey string) *CVAnalyzer {
	return &CVAnalyzer{
		apiKey: apiKey,
	}
}

// ExtractSkills извлекает навыки из текста CV с подробным анализом
func (ca *CVAnalyzer) ExtractSkills(ctx context.Context, cvText string) ([]*domain.UserSkill, error) {
	if ca.apiKey == "" {
		// Fallback: простой парсинг без API
		return ca.extractSkillsFallback(cvText), nil
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(ca.apiKey))
	if err != nil {
		// Fallback на простой парсинг если API не работает
		return ca.extractSkillsFallback(cvText), nil
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-pro")

	prompt := fmt.Sprintf(`Analyze this CV and extract ALL technical skills with detailed explanations.
Return a JSON object with "skills" array. For each skill include:
- "name": exact skill name (Go, React, PostgreSQL, Docker, etc)
- "category": Language/Framework/Database/Tool/Other
- "what": SHORT (1-2 sentences) what this technology is
- "why": SHORT (1-2 sentences) why/when it's used
- "description": SHORT (2-3 sentences) how the candidate uses it based on CV

Be thorough and extract ALL mentioned technologies.

CV TEXT:
%s

Return ONLY valid JSON, no markdown or extra text.`, cvText)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		// Fallback
		return ca.extractSkillsFallback(cvText), nil
	}

	if len(resp.Candidates) == 0 {
		return ca.extractSkillsFallback(cvText), nil
	}

	content := resp.Candidates[0].Content
	if len(content.Parts) == 0 {
		return ca.extractSkillsFallback(cvText), nil
	}

	responseText := fmt.Sprintf("%v", content.Parts[0])

	// Парсим JSON ответ
	var result SkillExtractResult
	jsonStr := extractJSON(responseText)
	err = json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		// Fallback
		return ca.extractSkillsFallback(cvText), nil
	}

	// Конвертим в domain модель
	skills := make([]*domain.UserSkill, 0)
	for _, skill := range result.Skills {
		what := skill.What
		why := skill.Why
		desc := skill.Description
		skills = append(skills, &domain.UserSkill{
			Skill:       strings.ToLower(strings.TrimSpace(skill.Name)),
			Category:    skill.Category,
			What:        stringPtr(what),
			Why:         stringPtr(why),
			Description: stringPtr(desc),
		})
	}

	return skills, nil
}

// extractSkillsFallback простое извлечение навыков без API
func (ca *CVAnalyzer) extractSkillsFallback(cvText string) []*domain.UserSkill {
	text := strings.ToLower(cvText)
	skills := make([]*domain.UserSkill, 0)

	// Правильная классификация технологий по категориям
	skillKeywords := map[string]string{
		// ════════ ЯЗЫКИ ПРОГРАММИРОВАНИЯ ════════
		"go":         "Language",
		"golang":     "Language",
		"python":     "Language",
		"rust":       "Language",
		"java":       "Language",
		"javascript": "Language",
		"typescript": "Language",
		"c++":        "Language",
		"c#":         "Language",
		"csharp":     "Language",
		"ruby":       "Language",
		"php":        "Language",
		"swift":      "Language",
		"kotlin":     "Language",
		"scala":      "Language",
		"perl":       "Language",
		"sql":        "Language",

		// ════════ WEB ФРЕЙМВОРКИ ════════
		"react":       "Framework",
		"vue":         "Framework",
		"angular":     "Framework",
		"svelte":      "Framework",
		"ember":       "Framework",
		"next.js":     "Framework",
		"nuxt":        "Framework",
		"remix":       "Framework",
		"express":     "Framework",
		"fastapi":     "Framework",
		"django":      "Framework",
		"flask":       "Framework",
		"gin":         "Framework",
		"fiber":       "Framework",
		"spring":      "Framework",
		"spring boot": "Framework",
		"laravel":     "Framework",
		"rails":       "Framework",
		"asp.net":     "Framework",
		"dotnet":      "Framework",
		"phoenix":     "Framework",

		// ════════ БАЗЫ ДАННЫХ ════════
		"postgresql":    "Database",
		"postgres":      "Database",
		"mysql":         "Database",
		"mongodb":       "Database",
		"redis":         "Database",
		"cassandra":     "Database",
		"dynamodb":      "Database",
		"elasticsearch": "Database",
		"neo4j":         "Database",
		"sqlite":        "Database",
		"oracle":        "Database",
		"mssql":         "Database",
		"mariadb":       "Database",

		// ════════ ОБЛАЧНЫЕ ПЛАТФОРМЫ ════════
		"aws":          "Platform",
		"gcp":          "Platform",
		"azure":        "Platform",
		"heroku":       "Platform",
		"digitalocean": "Platform",
		"linode":       "Platform",

		// ════════ ИНСТРУМЕНТЫ И УТИЛИТЫ ════════
		"docker":     "Tool",
		"kubernetes": "Tool",
		"k8s":        "Tool",
		"jenkins":    "Tool",
		"gitlab":     "Tool",
		"github":     "Tool",
		"git":        "Tool",
		"circleci":   "Tool",
		"travis":     "Tool",
		"terraform":  "Tool",
		"ansible":    "Tool",
		"graphql":    "Tool",
		"rest":       "Tool",
		"grpc":       "Tool",
		"prometheus": "Tool",
		"grafana":    "Tool",

		// ════════ ФРОНТЕНД ИНСТРУМЕНТЫ ════════
		"webpack":   "Tool",
		"babel":     "Tool",
		"npm":       "Tool",
		"yarn":      "Tool",
		"tailwind":  "Tool",
		"bootstrap": "Tool",
		"html":      "Language",
		"css":       "Language",
		"sass":      "Language",
		"scss":      "Language",

		// ════════ SOFT SKILLS ════════
		"remote":          "Soft Skill",
		"problem solving": "Soft Skill",
		"agile":           "Soft Skill",
		"leadership":      "Soft Skill",
	}

	added := make(map[string]bool)

	for keyword, category := range skillKeywords {
		if strings.Contains(text, keyword) && !added[keyword] {
			what := "Используется в проектах"
			why := "Автоматически определено из CV"
			skills = append(skills, &domain.UserSkill{
				Skill:    keyword,
				Category: category,
				What:     &what,
				Why:      &why,
			})
			added[keyword] = true
		}
	}

	return skills
}

// extractJSON извлекает JSON из текста
func extractJSON(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start >= end {
		// Попробуем как JSON массив
		start = strings.Index(text, "[")
		end = strings.LastIndex(text, "]")
		if start == -1 || end == -1 {
			return "{}"
		}
		// Обернем массив в объект
		return fmt.Sprintf(`{"skills":%s}`, text[start:end+1])
	}

	return text[start : end+1]
}

// CompareVacancyWithCV сравнивает требуемые скиллы вакансии с навыками из CV
func (ca *CVAnalyzer) CompareVacancyWithCV(vacancyText string, userSkills []*domain.UserSkill) *domain.SkillMatch {
	vacancyLower := strings.ToLower(vacancyText)

	// Создаем map навыков пользователя
	userSkillMap := make(map[string]bool)
	for _, skill := range userSkills {
		userSkillMap[skill.Skill] = true
	}

	matched := make([]string, 0)
	missing := make([]string, 0)

	// Ищем требуемые скиллы в вакансии
	for skill := range userSkillMap {
		if strings.Contains(vacancyLower, strings.ToLower(skill)) {
			matched = append(matched, skill)
		} else {
			// Проверяем как отсутствующий только если это был требуемый скилл
			if isSkillMentionedInVacancy(skill, vacancyText) {
				missing = append(missing, skill)
			}
		}
	}

	// Считаем процент совпадения
	totalSkills := len(matched) + len(missing)
	matchPercentage := float32(0)
	if totalSkills > 0 {
		matchPercentage = float32(len(matched)) / float32(totalSkills) * 100
	}

	return &domain.SkillMatch{
		MatchedSkills:   matched,
		MissingSkills:   missing,
		MatchPercentage: matchPercentage,
	}
}

// isSkillMentionedInVacancy проверяет упоминается ли скилл в вакансии как требуемый
// Упрощенная версия - проверяет наличие в тексте
func isSkillMentionedInVacancy(skill, vacancyText string) bool {
	keywords := []string{"require", "need", "must", "experience", "skilled", "proficiency", "знаний", "требуется", "необходимо", "опыт"}
	vacancyLower := strings.ToLower(vacancyText)

	for _, keyword := range keywords {
		// Проверяем есть ли скилл и ключевое слово рядом
		skillLower := strings.ToLower(skill)
		idx := strings.Index(vacancyLower, skillLower)
		if idx != -1 {
			// Проверяем в окружающем контексте
			start := idx - 100
			if start < 0 {
				start = 0
			}
			end := idx + len(skill) + 100
			if end > len(vacancyLower) {
				end = len(vacancyLower)
			}
			context := vacancyLower[start:end]
			if strings.Contains(context, keyword) {
				return true
			}
		}
	}
	return false
}

// AnalyzeCV делает умный анализ всего CV
func (ca *CVAnalyzer) AnalyzeCV(ctx context.Context, cvText string) (string, error) {
	if ca.apiKey == "" {
		return "Анализ недоступен (API не настроен)", nil
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(ca.apiKey))
	if err != nil {
		return "Ошибка при подключении к ИИ", nil
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-pro")

	prompt := fmt.Sprintf(`Analyze this CV from the perspective of the job market and tech industry.
Provide a professional report covering:

1. PROFILE: Brief summary of candidate's experience level and specialization
2. MARKET POSITION: How this profile aligns with current job market demand
3. SKILL STACK: Technologies and why they matter in 2026
4. MARKET REALITY: 
   - What junior developers need to know
   - What middle-level developers focus on
   - What senior developers do differently
5. RECOMMENDATIONS: What this candidate should focus on

Be honest, insightful, and practical. Use exactly these headers.

CV TEXT:
%s

Provide a well-structured, readable analysis (600-800 words).`, cvText)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "Ошибка при анализе CV", nil
	}

	if len(resp.Candidates) == 0 {
		return "Не удалось получить анализ", nil
	}

	content := resp.Candidates[0].Content
	if len(content.Parts) == 0 {
		return "Пустой ответ от ИИ", nil
	}

	return fmt.Sprintf("%v", content.Parts[0]), nil
}

// stringPtr возвращает указатель на строку, или nil если строка пустая
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
