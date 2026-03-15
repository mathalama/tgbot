package usecase

import (
	"context"
	"fmt"
	"log"

	"tgbot/internal/domain"
)

// FetchAndAnalyzeVacanciesUseCase получает вакансии и анализирует их
type FetchAndAnalyzeVacanciesUseCase struct {
	rssFetcher   domain.RSSFetcher
	vacancyRepo  domain.VacancyRepository
	skillRepo    domain.SkillRepository
	analysisRepo domain.AnalysisRepository
	notifier     domain.TelegramNotifier
	compareUC    *CompareVacancyUseCase
}

// NewFetchAndAnalyzeVacanciesUseCase создает новый use case
func NewFetchAndAnalyzeVacanciesUseCase(
	rssFetcher domain.RSSFetcher,
	vacancyRepo domain.VacancyRepository,
	skillRepo domain.SkillRepository,
	analysisRepo domain.AnalysisRepository,
	notifier domain.TelegramNotifier,
	compareUC *CompareVacancyUseCase,
) *FetchAndAnalyzeVacanciesUseCase {
	return &FetchAndAnalyzeVacanciesUseCase{
		rssFetcher:   rssFetcher,
		vacancyRepo:  vacancyRepo,
		skillRepo:    skillRepo,
		analysisRepo: analysisRepo,
		notifier:     notifier,
		compareUC:    compareUC,
	}
}

// Execute выполняет основную логику
// 1. Парсит каналы и получает вакансии
// 2. Сохраняет новые вакансии в БД
// 3. Проверяет вакансии против CV
func (uc *FetchAndAnalyzeVacanciesUseCase) Execute(ctx context.Context, channels []*domain.Channel) error {
	// Проверяем есть ли навыки в CV
	userSkills, err := uc.skillRepo.GetUserSkills(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user skills: %w", err)
	}

	if len(userSkills) == 0 {
		log.Printf("[WARN] No skills found in CV. Please upload your CV first!")
		return nil
	}

	log.Printf("[INFO] Using %d skills from your CV for matching", len(userSkills))

	allNewVacancies := make([]*domain.Vacancy, 0)

	// Этап 1: ПАРСИМ КАНАЛЫ И ПОЛУЧАЕМ ВАКАНСИИ
	for _, channel := range channels {
		if !channel.IsActive {
			continue
		}

		log.Printf("[INFO] 📡 Fetching vacancies from channel: %s", channel.Username)

		// Получаем вакансии из RSS/Telegram
		vacancies, err := uc.rssFetcher.FetchVacancies(ctx, channel.Username)
		if err != nil {
			log.Printf("[ERROR] Error fetching vacancies from channel %s: %v", channel.Username, err)
			continue
		}

		log.Printf("[INFO] Found %d vacancies in channel %s", len(vacancies), channel.Username)

		// Этап 2: СОХРАНЯЕМ НОВЫЕ ВАКАНСИИ В БД
		for _, vacancy := range vacancies {
			log.Printf("[INFO] 💾 Saving vacancy: %s", vacancy.Title)

			if err := uc.vacancyRepo.Save(ctx, vacancy); err != nil {
				log.Printf("[ERROR] Error saving vacancy: %v", err)
				continue
			}

			allNewVacancies = append(allNewVacancies, vacancy)
		}
	}

	if len(allNewVacancies) == 0 {
		log.Printf("[INFO] No new vacancies found in channels")
	} else {
		log.Printf("[INFO] ✅ Saved %d new vacancies to database", len(allNewVacancies))
	}

	// Этап 3: ПРОВЕРЯЕМ ВСЕ ВАКАНСИИ ПРОТИВ CV
	vacancies, err := uc.vacancyRepo.GetUnanalyzed(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to get unanalyzed vacancies: %w", err)
	}

	if len(vacancies) == 0 {
		log.Printf("[INFO] No vacancies to analyze")
		return nil
	}

	log.Printf("[INFO] Analyzing %d vacancies...", len(vacancies))

	// Обрабатываем каждую вакансию
	for _, vacancy := range vacancies {
		// Сравниваем с CV
		skillMatch, err := uc.compareUC.Execute(ctx, vacancy)
		if err != nil {
			log.Printf("[WARN] Error comparing vacancy with CV: %v", err)
			continue
		}

		// Определяем подходит ли вакансия
		// Считаем что подходит если % совпадения > 50% или есть ключевые навыки
		isSuitable := skillMatch.MatchPercentage >= 50 && len(skillMatch.MatchedSkills) > 0

		reason := fmt.Sprintf("Matched: %d skills, Missing: %d skills (%.1f%% match)\nYour skills: %v\nRequired: %v",
			len(skillMatch.MatchedSkills),
			len(skillMatch.MissingSkills),
			skillMatch.MatchPercentage,
			skillMatch.MatchedSkills,
			skillMatch.MissingSkills,
		)

		analysis := &domain.AnalysisResult{
			VacancyID:  vacancy.ID,
			IsSuitable: isSuitable,
			Reason:     reason,
			Confidence: skillMatch.MatchPercentage / 100,
		}

		// Сохраняем результат анализа
		if err := uc.analysisRepo.SaveResult(ctx, analysis); err != nil {
			log.Printf("[ERROR] Error saving analysis: %v", err)
			continue
		}

		// Отмечаем вакансию как проанализированную
		if err := uc.vacancyRepo.MarkAsAnalyzed(ctx, vacancy.ID); err != nil {
			log.Printf("[ERROR] Error marking vacancy as analyzed: %v", err)
		}

		log.Printf("[INFO] Vacancy '%s' - Match: %.1f%% (Suitable: %v)",
			vacancy.Title, skillMatch.MatchPercentage, isSuitable)

		// Если подходит - отправляем уведомление (но только если еще не отправляли)
		if isSuitable {
			// Проверяем была ли вакансия уже отправлена
			alreadySent, err := uc.vacancyRepo.IsSent(ctx, vacancy.ID)
			if err != nil {
				log.Printf("[WARN] Error checking if vacancy was sent: %v", err)
			}

			if alreadySent {
				log.Printf("[DEBUG] Vacancy already sent, skipping: %s", vacancy.Title)
			} else {
				log.Printf("[INFO] 📧 Sending vacancy: %s", vacancy.Title)
				if err := uc.notifier.SendVacancy(ctx, vacancy, analysis); err != nil {
					log.Printf("[ERROR] Error sending notification: %v", err)
				} else {
					// Отмечаем вакансию как отправленную только если успешно отправилась
					if err := uc.vacancyRepo.MarkAsSent(ctx, vacancy.ID); err != nil {
						log.Printf("[ERROR] Error marking vacancy as sent: %v", err)
					}
				}
			}
		}
	}

	return nil
}
