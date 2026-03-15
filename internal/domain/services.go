package domain

import "context"

// RSSFetcher интерфейс для получения данных из RSS
type RSSFetcher interface {
	FetchVacancies(ctx context.Context, channelUsername string) ([]*Vacancy, error)
}

// AIAnalyzer интерфейс для анализа вакансий через AI
type AIAnalyzer interface {
	AnalyzeVacancy(ctx context.Context, vacancy *Vacancy) (*AnalysisResult, error)
}

// TelegramNotifier интерфейс для отправки уведомлений в Telegram
type TelegramNotifier interface {
	SendVacancy(ctx context.Context, vacancy *Vacancy, analysis *AnalysisResult) error
}
