package usecase

import (
	"context"
	"fmt"
	"log"

	"tgbot/internal/adapter/service"
	"tgbot/internal/domain"
)

// AnalyzeCVUseCase анализирует загруженное CV пользователя
type AnalyzeCVUseCase struct {
	cvRepo     domain.CVRepository
	skillRepo  domain.SkillRepository
	cvAnalyzer *service.CVAnalyzer
}

// NewAnalyzeCVUseCase создает новый use case
func NewAnalyzeCVUseCase(
	cvRepo domain.CVRepository,
	skillRepo domain.SkillRepository,
	cvAnalyzer *service.CVAnalyzer,
) *AnalyzeCVUseCase {
	return &AnalyzeCVUseCase{
		cvRepo:     cvRepo,
		skillRepo:  skillRepo,
		cvAnalyzer: cvAnalyzer,
	}
}

// Execute загружает CV и извлекает навыки
func (uc *AnalyzeCVUseCase) Execute(ctx context.Context, cvText string) error {
	log.Printf("[INFO] Loading and analyzing CV (%d characters)", len(cvText))

	// Делаем умный анализ CV
	log.Printf("[INFO] Running AI analysis of CV...")
	analysis, err := uc.cvAnalyzer.AnalyzeCV(ctx, cvText)
	if err != nil {
		log.Printf("[ERROR] Failed to analyze CV: %v", err)
		analysis = "Анализ недоступен"
	}

	// Сохраняем CV с анализом
	newCV := &domain.UserCV{
		CVText:   cvText,
		Analysis: analysis,
	}

	cvID, err := uc.cvRepo.SaveCV(ctx, newCV)
	if err != nil {
		return fmt.Errorf("failed to save CV: %w", err)
	}

	log.Printf("[INFO] CV saved with ID: %d", cvID)

	// Извлекаем навыки из CV с подробным описанием
	skills, err := uc.cvAnalyzer.ExtractSkills(ctx, cvText)
	if err != nil {
		return fmt.Errorf("failed to extract skills: %w", err)
	}

	log.Printf("[INFO] Extracted %d skills from CV", len(skills))

	// Сохраняем навыки
	err = uc.skillRepo.SaveSkills(ctx, cvID, skills)
	if err != nil {
		return fmt.Errorf("failed to save skills: %w", err)
	}

	// Отмечаем CV как проанализированное
	err = uc.cvRepo.MarkAsAnalyzed(ctx, cvID)
	if err != nil {
		return fmt.Errorf("failed to mark CV as analyzed: %w", err)
	}

	// Выводим извлеченные навыки
	log.Printf("[INFO] Extracted skills:")
	for _, skill := range skills {
		log.Printf("  • %s (%s)", skill.Skill, skill.Category)
	}

	return nil
}

// CompareVacancyUseCase сравнивает вакансию с CV пользователя
type CompareVacancyUseCase struct {
	skillRepo  domain.SkillRepository
	cvAnalyzer *service.CVAnalyzer
}

// NewCompareVacancyUseCase создает новый use case
func NewCompareVacancyUseCase(
	skillRepo domain.SkillRepository,
	cvAnalyzer *service.CVAnalyzer,
) *CompareVacancyUseCase {
	return &CompareVacancyUseCase{
		skillRepo:  skillRepo,
		cvAnalyzer: cvAnalyzer,
	}
}

// Execute сравнивает требуемые навыки вакансии с CV
func (uc *CompareVacancyUseCase) Execute(ctx context.Context, vacancy *domain.Vacancy) (*domain.SkillMatch, error) {
	// Получаем навыки пользователя
	userSkills, err := uc.skillRepo.GetUserSkills(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user skills: %w", err)
	}

	if len(userSkills) == 0 {
		return nil, fmt.Errorf("no skills found in CV, please upload your CV first")
	}

	// Сравниваем вакансию с CV
	vacancyText := vacancy.Title + " " + vacancy.Content

	match := uc.cvAnalyzer.CompareVacancyWithCV(vacancyText, userSkills)

	log.Printf("[DEBUG] Vacancy '%s' - Matched: %d, Missing: %d, Match%%: %.1f",
		vacancy.Title, len(match.MatchedSkills), len(match.MissingSkills), match.MatchPercentage)

	return match, nil
}
