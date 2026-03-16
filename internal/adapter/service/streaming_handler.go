package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StreamConfig конфигурация для стриминга
type StreamConfig struct {
	UpdateInterval   time.Duration // Интервал между обновлениями (default 500ms)
	BufferThreshold  int           // Минимальное количество символов для обновления (default 100)
	MaxBufferSize    int           // Максимальный размер буфера перед обновлением (default 1000)
	ShowTypingStatus bool          // Показывать ли "typing..." статус (default true)
}

// DefaultStreamConfig возвращает конфигурацию по умолчанию
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		UpdateInterval:   500 * time.Millisecond,
		BufferThreshold:  100,
		MaxBufferSize:    1000,
		ShowTypingStatus: true,
	}
}

// StreamingMessageUpdater управляет real-time обновлениями сообщения
type StreamingMessageUpdater struct {
	bot       *tgbotapi.BotAPI
	chatID    int64
	messageID int
	config    StreamConfig

	buffer         string
	lastUpdateTime time.Time
	mutex          sync.Mutex
	isActive       bool
}

// NewStreamingMessageUpdater создает новый updater для стриминга
func NewStreamingMessageUpdater(
	bot *tgbotapi.BotAPI,
	chatID int64,
	messageID int,
	config StreamConfig,
) *StreamingMessageUpdater {
	if config.UpdateInterval == 0 {
		config = DefaultStreamConfig()
	}

	return &StreamingMessageUpdater{
		bot:            bot,
		chatID:         chatID,
		messageID:      messageID,
		config:         config,
		lastUpdateTime: time.Now(),
		isActive:       true,
	}
}

// PushToken добавляет токен в буфер и периодически обновляет сообщение
// Возвращает true если было обновление, false если токен просто добавлен в буфер
func (s *StreamingMessageUpdater) PushToken(ctx context.Context, token string) (updated bool, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isActive {
		return false, fmt.Errorf("streamer not active")
	}

	s.buffer += token

	// Проверяем нужно ли обновлять
	shouldUpdate := false

	// Условие 1: Прошло достаточно времени
	if time.Since(s.lastUpdateTime) >= s.config.UpdateInterval {
		shouldUpdate = true
	}

	// Условие 2: Буфер слишком большой
	if len(s.buffer) >= s.config.MaxBufferSize {
		shouldUpdate = true
	}

	// Условие 3: Завершающий токен (пустая строка или специальный маркер)
	if token == "\n" && len(s.buffer) >= s.config.BufferThreshold {
		shouldUpdate = true
	}

	if shouldUpdate {
		return true, s.updateMessage(ctx, s.buildDisplayText())
	}

	return false, nil
}

// Flush отправляет все оставшееся в буфере и завершает стриминг
func (s *StreamingMessageUpdater) Flush(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isActive {
		return fmt.Errorf("streamer not active")
	}

	s.isActive = false

	if s.buffer != "" {
		return s.updateMessage(ctx, s.buildDisplayText())
	}

	return nil
}

// Cancel отменяет стриминг и оставляет сообщение как есть
func (s *StreamingMessageUpdater) Cancel() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.isActive = false
	s.buffer = ""

	return nil
}

// GetCurrentContent возвращает текущее содержимое без блокировки
// (используется для отладки)
func (s *StreamingMessageUpdater) GetCurrentContent() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.buffer
}

// updateMessage обновляет сообщение в Telegram
func (s *StreamingMessageUpdater) updateMessage(ctx context.Context, text string) error {
	edit := tgbotapi.NewEditMessageText(s.chatID, s.messageID, text)

	// Обновляем timestamp последнего обновления
	s.lastUpdateTime = time.Now()

	_, err := s.bot.Send(edit)
	if err != nil {
		// Логируем но не ломаемся - может быть rate limit
		fmt.Printf("[STREAM] Edit message error: %v\n", err)
		return nil // Silently continue на случай rate limit
	}

	return nil
}

// buildDisplayText строит текст для отображения с прогресс-индикатором
func (s *StreamingMessageUpdater) buildDisplayText() string {
	if s.isActive {
		// Добавляем мигающий курсор для активного стриминга
		return fmt.Sprintf("%s█\n\n_⏳ Генерирую..._", s.buffer)
	}

	// Финальный текст без курсора
	return s.buffer
}

// StreamTokenChannel создает канал для streaming и запускает goroutine для обработки
// Возвращает канал для отправки токенов и функцию для завершения
func (s *StreamingMessageUpdater) StreamTokenChannel(ctx context.Context) (chan string, func() error) {
	tokenChan := make(chan string, 100) // Буферизация до 100 токенов

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = s.Flush(ctx)
				close(tokenChan)
				return

			case token, ok := <-tokenChan:
				if !ok {
					_ = s.Flush(ctx)
					return
				}

				_, _ = s.PushToken(ctx, token)
			}
		}
	}()

	// Функция для graceful завершения
	finishFn := func() error {
		close(tokenChan)
		return s.Flush(ctx)
	}

	return tokenChan, finishFn
}

// StreamWriter адаптер для io.Writer интерфейса
type StreamWriter struct {
	updater *StreamingMessageUpdater
	ctx     context.Context
}

// Write реализует io.Writer интерфейс
func (sw *StreamWriter) Write(p []byte) (n int, err error) {
	token := string(p)

	if _, err := sw.updater.PushToken(sw.ctx, token); err != nil {
		return 0, err
	}

	return len(p), nil
}

// NewStreamWriter создает writer для стриминга
func (s *StreamingMessageUpdater) NewStreamWriter(ctx context.Context) *StreamWriter {
	return &StreamWriter{
		updater: s,
		ctx:     ctx,
	}
}

// ============================================================================
// Helper функции для создания streaming сессии
// ============================================================================

// SendInitialStreamingMessage отправляет начальное сообщение и возвращает updater
func SendInitialStreamingMessage(
	bot *tgbotapi.BotAPI,
	chatID int64,
	title string,
	config StreamConfig,
) (*StreamingMessageUpdater, error) {
	// Отправляем начальное сообщение
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⏳ %s\n\n_загружаю..._", title))
	msg.ParseMode = "Markdown"

	result, err := bot.Send(msg)
	if err != nil {
		return nil, err
	}

	// Создаем updater
	updater := NewStreamingMessageUpdater(bot, chatID, result.MessageID, config)

	return updater, nil
}

// ============================================================================
// Rate limiter для throttling обновлений
// ============================================================================

// RateLimitedStreamer обертка вокруг StreamingMessageUpdater с более агрессивным rate limiting
type RateLimitedStreamer struct {
	streamer            *StreamingMessageUpdater
	minTimeBetweenEdits time.Duration
	lastEdit            time.Time
}

// NewRateLimitedStreamer создает rate-limited streamer
func NewRateLimitedStreamer(streamer *StreamingMessageUpdater, minInterval time.Duration) *RateLimitedStreamer {
	return &RateLimitedStreamer{
		streamer:            streamer,
		minTimeBetweenEdits: minInterval,
		lastEdit:            time.Now().Add(-minInterval), // Позволяем первое обновление
	}
}

// PushToken добавляет токен с rate limiting
func (r *RateLimitedStreamer) PushToken(ctx context.Context, token string) (updated bool, err error) {
	updated, err = r.streamer.PushToken(ctx, token)

	// Если было обновление, проверяем rate limit
	if updated && time.Since(r.lastEdit) < r.minTimeBetweenEdits {
		// Слишком частые обновления - отменяем это обновление
		return false, nil
	}

	if updated {
		r.lastEdit = time.Now()
	}

	return updated, err
}

// Proxy методы
func (r *RateLimitedStreamer) Flush(ctx context.Context) error {
	return r.streamer.Flush(ctx)
}

func (r *RateLimitedStreamer) Cancel() error {
	return r.streamer.Cancel()
}

func (r *RateLimitedStreamer) GetCurrentContent() string {
	return r.streamer.GetCurrentContent()
}
