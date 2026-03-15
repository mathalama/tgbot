package service

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tgbot/internal/domain"
)

// TelegramDirectFetcher получает вакансии прямо из Telegram канала
type TelegramDirectFetcher struct {
	bot       *tgbotapi.BotAPI
	channelID int64
}

// NewTelegramDirectFetcher создает новый fetcher
func NewTelegramDirectFetcher(botToken string, channelID string) (*TelegramDirectFetcher, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	// Парсим channel ID
	chatID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid channel ID: %w", err)
	}

	return &TelegramDirectFetcher{
		bot:       bot,
		channelID: chatID,
	}, nil
}

// FetchVacancies получает последние сообщения из канала
func (tf *TelegramDirectFetcher) FetchVacancies(ctx context.Context, channelUsername string) ([]*domain.Vacancy, error) {
	// TODO: Реализовать получение реальных вакансий
	// Текущая реализация ограничена telegram-bot-api v5
	//
	// Решения:
	// 1. RSS Hub инстанс (self-hosted через Docker)
	// 2. Прямой парсинг сообщений через более новый API
	// 3. Webhook подписка на канал

	vacancies := []*domain.Vacancy{}

	return vacancies, nil
}
