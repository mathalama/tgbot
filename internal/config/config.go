package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config конфигурация приложения
type Config struct {
	TelegramBotToken string
	TelegramChatID   int64
	GeminiAPIKey     string
	RSSHubURL        string
	Channels         []string
}

// LoadFromEnv загружает конфигурацию из переменных окружения или .env файла
func LoadFromEnv() (*Config, error) {
	// Сначала пытаемся загрузить из .env файла
	loadEnvFile(".env")

	cfg := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		RSSHubURL:        os.Getenv("RSSHUB_URL"),
	}

	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	if cfg.GeminiAPIKey == "" {
		cfg.GeminiAPIKey = "not-set"
	}

	if cfg.RSSHubURL == "" {
		cfg.RSSHubURL = "https://rsshub.app"
	}

	return cfg, nil
}

// loadEnvFile загружает переменные окружения из .env файла
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		// Если .env не найден, просто игнорируем
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем комментарии и пустые строки
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Парсим KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Удаляем кавычки если есть
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}

			// Устанавливаем переменную окружения если она не установлена
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}

	return scanner.Err()
}
