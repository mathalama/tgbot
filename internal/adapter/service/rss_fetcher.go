package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tgbot/internal/domain"
)

// RSSHubFetcher получает вакансии из RSS Hub
type RSSHubFetcher struct {
	baseURL    string
	httpClient *http.Client
}

// NewRSSHubFetcher создает новый fetcher
func NewRSSHubFetcher(baseURL string, httpClient *http.Client) *RSSHubFetcher {
	if baseURL == "" {
		baseURL = "https://rsshub.app"
	}
	return &RSSHubFetcher{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

type RSSItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Link        string `json:"link"`
	PubDate     string `json:"pubDate"`
	Author      string `json:"author"`
	Content     string `json:"content:encoded"`
}

type RSSFeed struct {
	Items []RSSItem `json:"items"`
}

// FetchVacancies получает вакансии из канала
func (f *RSSHubFetcher) FetchVacancies(ctx context.Context, channelUsername string) ([]*domain.Vacancy, error) {
	// Формируем URL: https://rsshub.app/telegram/channel/USERNAME
	url := fmt.Sprintf("%s/telegram/channel/%s", f.baseURL, strings.TrimPrefix(channelUsername, "@"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Добавляем User-Agent для обхода ограничений
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// RSShub возвращает JSON в формате RSS Feed
	var feed RSSFeed
	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var vacancies []*domain.Vacancy
	for _, item := range feed.Items {
		vacancy := &domain.Vacancy{
			ID:          item.Link, // используем ссылку как уникальный ID
			Title:       item.Title,
			Content:     item.Content,
			Link:        item.Link,
			Recruiter:   item.Author,
			Source:      channelUsername,
			PublishedAt: item.PubDate,
		}
		vacancies = append(vacancies, vacancy)
	}

	return vacancies, nil
}
