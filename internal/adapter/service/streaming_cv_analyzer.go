package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// StreamingCVAnalyzer анализирует CV с поддержкой стриминга
type StreamingCVAnalyzer struct {
	apiKey string
}

// NewStreamingCVAnalyzer создает analyzer со стримингом
func NewStreamingCVAnalyzer(apiKey string) *StreamingCVAnalyzer {
	return &StreamingCVAnalyzer{
		apiKey: apiKey,
	}
}

// AnalyzeWithStreaming анализирует CV и отправляет результаты в streaming writer
// Возвращает распарсенные навыки и ошибку
func (sa *StreamingCVAnalyzer) AnalyzeWithStreaming(
	ctx context.Context,
	cvText string,
	streamWriter io.Writer,
) (map[string][]string, error) {

	client, err := genai.NewClient(ctx, option.WithAPIKey(sa.apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")

	prompt := fmt.Sprintf(`Analyze this CV/Resume and extract technical skills. 
For each skill, determine its category.

Format your response as:
Category: [CATEGORY]
Skills: [SKILL1], [SKILL2], [SKILL3]

Categories:
- Language: Go, Python, Java, JavaScript, C++, SQL, Rust, etc.
- Framework: React, Vue, Spring, Django, FastAPI, Gin, Express, etc.
- Database: PostgreSQL, MongoDB, Redis, MySQL, Cassandra, etc.
- Platform: AWS, GCP, Azure, Heroku, DigitalOcean, etc.
- Tool: Docker, Kubernetes, Git, Jenkins, Terraform, REST, GraphQL, etc.
- Soft Skill: Remote, Leadership, Agile, Communication, etc.

CV Text:
%s

Please extract and categorize the skills found in this CV:`, cvText)

	// Используем streaming API
	iter := model.GenerateContentStream(ctx, genai.Text(prompt))

	skillMap := make(map[string][]string)
	fullResponse := ""

	for {
		resp, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("streaming error: %w", err)
		}

		// Получаем текст из ответа
		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			continue
		}

		for _, part := range resp.Candidates[0].Content.Parts {
			chunk := fmt.Sprintf("%v", part)
			fullResponse += chunk

			// Пишем в stream writer
			_, _ = streamWriter.Write([]byte(chunk))
		}
	}

	// Парсим ответ и извлекаем навыки
	skillMap = sa.parseStreamingResponse(fullResponse)

	return skillMap, nil
}

// parseStreamingResponse парсит потоковый ответ и извлекает навыки
func (sa *StreamingCVAnalyzer) parseStreamingResponse(response string) map[string][]string {
	skillMap := make(map[string][]string)

	lines := strings.Split(response, "\n")
	currentCategory := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Ищем строку с категорией
		if strings.HasPrefix(line, "Category:") {
			currentCategory = strings.TrimPrefix(line, "Category:")
			currentCategory = strings.TrimSpace(currentCategory)
			continue
		}

		// Ищем строку с навыками
		if strings.HasPrefix(line, "Skills:") && currentCategory != "" {
			skillsStr := strings.TrimPrefix(line, "Skills:")
			skillsStr = strings.TrimSpace(skillsStr)

			// Парсим навыки (разделены запятыми)
			skills := strings.Split(skillsStr, ",")
			for _, skill := range skills {
				skill = strings.TrimSpace(skill)
				if skill != "" {
					skillMap[currentCategory] = append(skillMap[currentCategory], skill)
				}
			}
		}
	}

	return skillMap
}

// ExtractSkillsFromResponseStream извлекает навыки из потока ответа
// Используется для реального времени парсинга результатов
func (sa *StreamingCVAnalyzer) ExtractSkillsFromResponseStream(responsePart string) map[string][]string {
	return sa.parseStreamingResponse(responsePart)
}

// ============================================================================
// Batch streaming analyzer для анализа нескольких CV одновременно
// ============================================================================

// StreamingAnalysisJob задача для анализа
type StreamingAnalysisJob struct {
	ID     string
	CVText string
	Writer io.Writer
	Done   chan error
}

// BatchStreamingAnalyzer анализирует несколько CV с контролем одновременности
type BatchStreamingAnalyzer struct {
	analyzer      *StreamingCVAnalyzer
	maxConcurrent int
	jobs          chan StreamingAnalysisJob
}

// NewBatchStreamingAnalyzer создает batch analyzer
func NewBatchStreamingAnalyzer(apiKey string, maxConcurrent int) *BatchStreamingAnalyzer {
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}

	bsa := &BatchStreamingAnalyzer{
		analyzer:      NewStreamingCVAnalyzer(apiKey),
		maxConcurrent: maxConcurrent,
		jobs:          make(chan StreamingAnalysisJob, maxConcurrent),
	}

	// Запускаем worker goroutines
	for i := 0; i < maxConcurrent; i++ {
		go bsa.worker()
	}

	return bsa
}

// Submit добавляет задачу в очередь
func (bsa *BatchStreamingAnalyzer) Submit(ctx context.Context, job StreamingAnalysisJob) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case bsa.jobs <- job:
		return nil
	}
}

// worker обрабатывает задачи из очереди
func (bsa *BatchStreamingAnalyzer) worker() {
	for job := range bsa.jobs {
		ctx := context.Background()
		_, err := bsa.analyzer.AnalyzeWithStreaming(ctx, job.CVText, job.Writer)
		job.Done <- err
	}
}

// Close закрывает batch analyzer
func (bsa *BatchStreamingAnalyzer) Close() {
	close(bsa.jobs)
}
