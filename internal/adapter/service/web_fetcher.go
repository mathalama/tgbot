package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"tgbot/internal/domain"
)

// WebFetcher получает вакансии путем парсинга t.me/s/channel_name
type WebFetcher struct {
	httpClient *http.Client
	baseURL    string
}

// NewWebFetcher создает новый web fetcher
func NewWebFetcher(httpClient *http.Client) *WebFetcher {
	return &WebFetcher{
		httpClient: httpClient,
		baseURL:    "https://t.me/s",
	}
}

// FetchVacancies получает вакансии из публичного канала
func (wf *WebFetcher) FetchVacancies(ctx context.Context, channelUsername string) ([]*domain.Vacancy, error) {
	// Очистить @ если есть
	channelUsername = strings.TrimPrefix(channelUsername, "@")

	// URL публичного канала
	url := fmt.Sprintf("%s/%s", wf.baseURL, channelUsername)

	log.Printf("[DEBUG] Fetching from URL: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем User-Agent чтобы выглядеть как браузер
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := wf.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] Response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	log.Printf("[DEBUG] Response body size: %d bytes", len(body))

	// Парсим HTML
	vacancies := wf.parseMessages(string(body), channelUsername)

	log.Printf("[DEBUG] Parsed %d vacancies from channel %s", len(vacancies), channelUsername)

	return vacancies, nil
}

// parseMessages парсит сообщения из HTML архива канала
func (wf *WebFetcher) parseMessages(html string, channelUsername string) []*domain.Vacancy {
	var vacancies []*domain.Vacancy

	// Пытаемся несколько regex паттернов для разных версий Telegram

	// Паттерн 1: <div class="tgme_widget_message_text">...</div>
	pattern1 := regexp.MustCompile(`<div[^>]*class="[^"]*tgme_widget_message_text[^"]*"[^>]*>(.*?)</div>`)
	matches1 := pattern1.FindAllStringSubmatch(html, -1)

	if len(matches1) == 0 {
		// Паттерн 2: <div class="message-text">...</div>
		pattern2 := regexp.MustCompile(`<div[^>]*class="[^"]*message-text[^"]*"[^>]*>(.*?)</div>`)
		matches1 = pattern2.FindAllStringSubmatch(html, -1)
	}

	if len(matches1) == 0 {
		// Паттерн 3: любой div с текстом сообщения (более либеральный)
		pattern3 := regexp.MustCompile(`<div[^>]*class="[^"]*message[^"]*"[^>]*>(.*?)</div>`)
		matches1 = pattern3.FindAllStringSubmatch(html, -1)
	}

	log.Printf("[DEBUG] Found %d message blocks in HTML", len(matches1))

	for _, match := range matches1 {
		if len(match) < 2 {
			continue
		}

		messageContent := match[1]

		// Очищаем HTML теги
		text := wf.cleanHTML(messageContent)
		text = strings.TrimSpace(text)

		// Пропускаем системные сообщения
		systemMessages := []string{
			"channel created",
			"user joined",
			"user left",
			"message deleted",
			"message edited",
			"pinned",
			"unpinned",
		}

		textLower := strings.ToLower(text)
		isSystemMessage := false
		for _, sysMsg := range systemMessages {
			if strings.Contains(textLower, sysMsg) {
				isSystemMessage = true
				break
			}
		}

		if isSystemMessage {
			continue
		}

		if text == "" || len(text) < 10 {
			// Пропускаем пустые или слишком короткие сообщения
			continue
		}

		// Извлекаем заголовок
		title := wf.extractTitle(text)
		if title == "" {
			// Если заголовок не найден, пропускаем
			continue
		}

		// Генерируем стабильный ID на основе хеша контента
		// Это гарантирует что один и тот же вакансия всегда будет иметь одинаковый ID
		contentHash := md5.Sum([]byte(title + text))
		messageID := fmt.Sprintf("%s_%x", channelUsername, contentHash[:8])

		vacancy := &domain.Vacancy{
			ID:          messageID,
			Title:       title,
			Content:     text,
			Link:        fmt.Sprintf("https://t.me/s/%s/%s", channelUsername, messageID),
			Recruiter:   "Unknown",
			Source:      channelUsername,
			PublishedAt: time.Now().Format(time.RFC3339),
		}

		log.Printf("[DEBUG] Parsed vacancy: %s (ID: %s)", vacancy.Title, vacancy.ID)

		vacancies = append(vacancies, vacancy)
	}

	// Если не найдено через текстовые блоки, пробуем через все элементы с href
	if len(vacancies) == 0 {
		log.Printf("[WARN] No vacancies found with text parsing, trying alternative method")
		vacancies = wf.parseMessagesAlternative(html, channelUsername)
	}

	return vacancies
}

// parseMessagesAlternative альтернативный парсинг через ссылки и структур
func (wf *WebFetcher) parseMessagesAlternative(html string, channelUsername string) []*domain.Vacancy {
	var vacancies []*domain.Vacancy

	// Ищем блоки сообщений через data атрибуты или другие маркеры
	// <article> или <div> с определенными классами
	articleRegex := regexp.MustCompile(`<article[^>]*>(.*?)</article>`)
	articles := articleRegex.FindAllStringSubmatch(html, -1)

	log.Printf("[DEBUG] Found %d article blocks", len(articles))

	for _, article := range articles {
		if len(article) < 2 {
			continue
		}

		content := article[1]
		// Очищаем
		text := wf.cleanHTML(content)
		text = strings.TrimSpace(text)

		if len(text) < 20 {
			continue
		}

		title := wf.extractTitle(text)
		if title == "" {
			continue
		}

		// Генерируем стабильный ID на основе хеша контента
		contentHash := md5.Sum([]byte(title + text))
		messageID := fmt.Sprintf("%s_%x", channelUsername, contentHash[:8])

		vacancy := &domain.Vacancy{
			ID:          messageID,
			Title:       title,
			Content:     text,
			Link:        fmt.Sprintf("https://t.me/s/%s/%s", channelUsername, messageID),
			Recruiter:   "Unknown",
			Source:      channelUsername,
			PublishedAt: time.Now().Format(time.RFC3339),
		}

		vacancies = append(vacancies, vacancy)
	}

	return vacancies
}

// cleanHTML удаляет HTML теги из строки
func (wf *WebFetcher) cleanHTML(html string) string {
	// Удаляем <br>
	html = regexp.MustCompile(`<br\s*/?>`).ReplaceAllString(html, "\n")

	// Удаляем теги
	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "")

	// Декодируем HTML entities
	html = strings.NewReplacer(
		"&nbsp;", " ",
		"&quot;", "\"",
		"&#39;", "'",
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
	).Replace(html)

	// Убираем лишние пробелы
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")

	return strings.TrimSpace(html)
}

// extractTitle извлекает заголовок из текста (отсеивая системные сообщения)
func (wf *WebFetcher) extractTitle(text string) string {
	// Список системных сообщений и шаблонов которые следует игнорировать
	systemPatterns := []string{
		"channel created",
		"user joined",
		"user left",
		"user removed",
		"user was banned",
		"message deleted",
		"message edited",
		"pinned",
		"unpinned",
		"was set as channel topic",
		"was removed from pinned messages",
		"as group admin",
	}

	// Первая строка или первые 100 символов
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		title := strings.TrimSpace(lines[0])
		titleLower := strings.ToLower(title)

		// Проверяем не системное ли это сообщение
		for _, pattern := range systemPatterns {
			if strings.Contains(titleLower, pattern) {
				return "" // Это системное сообщение
			}
		}

		if len(title) > 10 {
			return title
		}
	}

	// Если первая строка коротка, берем до первой точки
	if idx := strings.Index(text, "."); idx > 0 && idx < 150 {
		candidate := text[:idx]
		candidateLower := strings.ToLower(candidate)
		for _, pattern := range systemPatterns {
			if strings.Contains(candidateLower, pattern) {
				return "" // Это системное сообщение
			}
		}
		return candidate
	}

	// Иначе первые 100 символов
	if len(text) > 100 {
		return text[:100] + "..."
	}

	return text
}
