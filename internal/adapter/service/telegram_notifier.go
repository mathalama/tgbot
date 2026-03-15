package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tgbot/internal/domain"
)

// TelegramNotifier отправляет уведомления в Telegram
type TelegramNotifier struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

// NewTelegramNotifier создает новый notifier
func NewTelegramNotifier(botToken string, chatID int64) (*TelegramNotifier, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &TelegramNotifier{
		bot:    bot,
		chatID: chatID,
	}, nil
}

// GetBot возвращает Telegram bot API
func (tn *TelegramNotifier) GetBot() *tgbotapi.BotAPI {
	return tn.bot
}

// SendVacancy отправляет вакансию в Telegram красиво отформатированную
func (tn *TelegramNotifier) SendVacancy(ctx context.Context, vacancy *domain.Vacancy, analysis *domain.AnalysisResult) error {
	// Форматируем источник с @
	source := "@" + vacancy.Source

	message := fmt.Sprintf(
		"🎉 <b>%s</b>\n\n"+
			"💼 <b>%s</b> | 📍 <code>%s</code>\n\n"+
			"%s\n"+
			"✨ Уверенность: <code>%.0f%%</code>\n\n"+
			"<a href=\"%s\">🔗 Смотреть</a>",
		vacancy.Title,
		vacancy.Recruiter,
		source,
		formatAnalysisReason(analysis.Reason),
		analysis.Confidence*100,
		vacancy.Link,
	)

	msg := tgbotapi.NewMessage(tn.chatID, message)
	msg.ParseMode = "html"

	_, err := tn.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// formatAnalysisReason форматирует результат анализа красиво
func formatAnalysisReason(reason string) string {
	// Парсим строку типа:
	// "Matched: 6 skills, Missing: 0 skills (100.0% match)\nYour skills: [java git postgres ...]\nRequired: []"

	// Извлекаем навыки
	var matchedSkills []string
	var missingSkills []string

	// Находим "Your skills: [...]"
	skillsRegex := regexp.MustCompile(`Your skills: \[(.*?)\]`)
	if matches := skillsRegex.FindStringSubmatch(reason); len(matches) > 1 {
		skillsStr := matches[1]
		if skillsStr != "" {
			matchedSkills = strings.Fields(skillsStr)
		}
	}

	// Находим "Required: [...]"
	requiredRegex := regexp.MustCompile(`Required: \[(.*?)\]`)
	if matches := requiredRegex.FindStringSubmatch(reason); len(matches) > 1 {
		reqStr := matches[1]
		if reqStr != "" {
			missingSkills = strings.Fields(reqStr)
		}
	}

	// Форматируем компактно
	result := "<b>✅ Навыки:</b> "
	if len(matchedSkills) > 0 {
		for i, skill := range matchedSkills {
			skillName := strings.ToUpper(skill[:1]) + skill[1:]
			result += "<b>" + skillName + "</b>"
			if i < len(matchedSkills)-1 {
				result += " • "
			}
		}
	}

	if len(missingSkills) > 0 {
		result += "\n<b>❌ Нужно:</b> "
		for i, skill := range missingSkills {
			skillName := strings.ToUpper(skill[:1]) + skill[1:]
			result += skillName
			if i < len(missingSkills)-1 {
				result += " • "
			}
		}
	} else {
		result += "\n🎯 <i>Отлично! Все требования есть.</i>"
	}

	return result
}
