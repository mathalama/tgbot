package repository

import (
	"context"
	"sync"

	"tgbot/internal/domain"
)

// InMemoryVacancyRepository хранит вакансии в памяти
type InMemoryVacancyRepository struct {
	mu        sync.RWMutex
	vacancies map[string]*domain.Vacancy
	analyzed  map[string]bool
}

// NewInMemoryVacancyRepository создает новый репозиторий
func NewInMemoryVacancyRepository() *InMemoryVacancyRepository {
	return &InMemoryVacancyRepository{
		vacancies: make(map[string]*domain.Vacancy),
		analyzed:  make(map[string]bool),
	}
}

// Save сохраняет вакансию
func (r *InMemoryVacancyRepository) Save(ctx context.Context, vacancy *domain.Vacancy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.vacancies[vacancy.ID] = vacancy
	return nil
}

// FindByID находит вакансию по ID
func (r *InMemoryVacancyRepository) FindByID(ctx context.Context, id string) (*domain.Vacancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vacancy, exists := r.vacancies[id]
	if !exists {
		return nil, nil
	}
	return vacancy, nil
}

// GetUnanalyzed получает неанализированные вакансии
func (r *InMemoryVacancyRepository) GetUnanalyzed(ctx context.Context, limit int) ([]*domain.Vacancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Vacancy
	for id, vacancy := range r.vacancies {
		if !r.analyzed[id] && len(result) < limit {
			result = append(result, vacancy)
		}
	}
	return result, nil
}

// MarkAsAnalyzed отмечает вакансию как проанализированную
func (r *InMemoryVacancyRepository) MarkAsAnalyzed(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.analyzed[id] = true
	return nil
}

// GetBySource получает вакансии из определенного источника
func (r *InMemoryVacancyRepository) GetBySource(ctx context.Context, source string) ([]*domain.Vacancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Vacancy
	for _, vacancy := range r.vacancies {
		if vacancy.Source == source {
			result = append(result, vacancy)
		}
	}
	return result, nil
}

// GetAllVacancies получает все вакансии
func (r *InMemoryVacancyRepository) GetAllVacancies(ctx context.Context) ([]*domain.Vacancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Vacancy
	for _, vacancy := range r.vacancies {
		result = append(result, vacancy)
	}
	return result, nil
}

// IsSent проверяет была ли вакансия отправлена пользователю (заглушка в памяти)
func (r *InMemoryVacancyRepository) IsSent(ctx context.Context, id string) (bool, error) {
	// В памяти не отслеживаем - всегда возвращаем false
	return false, nil
}

// MarkAsSent отмечает вакансию как отправленную (заглушка в памяти)
func (r *InMemoryVacancyRepository) MarkAsSent(ctx context.Context, id string) error {
	// В памяти не отслеживаем
	return nil
}

// DeleteOldVacancies удаляет вакансии старше N дней (заглушка)
func (r *InMemoryVacancyRepository) DeleteOldVacancies(ctx context.Context, daysOld int) (int, error) {
	// В памяти просто игнорируем
	return 0, nil
}

// DeleteAll удаляет все вакансии
func (r *InMemoryVacancyRepository) DeleteAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.vacancies = make(map[string]*domain.Vacancy)
	r.analyzed = make(map[string]bool)
	return nil
}

// InMemoryChannelRepository хранит каналы в памяти
type InMemoryChannelRepository struct {
	mu       sync.RWMutex
	channels map[string]*domain.Channel
}

// NewInMemoryChannelRepository создает новый репозиторий
func NewInMemoryChannelRepository() *InMemoryChannelRepository {
	return &InMemoryChannelRepository{
		channels: make(map[string]*domain.Channel),
	}
}

// Save сохраняет канал
func (r *InMemoryChannelRepository) Save(ctx context.Context, channel *domain.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.channels[channel.ID] = channel
	return nil
}

// GetAll получает все каналы
func (r *InMemoryChannelRepository) GetAll(ctx context.Context) ([]*domain.Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Channel
	for _, channel := range r.channels {
		result = append(result, channel)
	}
	return result, nil
}

// GetByID получает канал по ID
func (r *InMemoryChannelRepository) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	channel, exists := r.channels[id]
	if !exists {
		return nil, nil
	}
	return channel, nil
}

// Delete удаляет канал
func (r *InMemoryChannelRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.channels, id)
	return nil
}

// InMemoryAnalysisRepository хранит результаты анализа в памяти
type InMemoryAnalysisRepository struct {
	mu       sync.RWMutex
	analysis map[string]*domain.AnalysisResult
}

// NewInMemoryAnalysisRepository создает новый репозиторий
func NewInMemoryAnalysisRepository() *InMemoryAnalysisRepository {
	return &InMemoryAnalysisRepository{
		analysis: make(map[string]*domain.AnalysisResult),
	}
}

// SaveResult сохраняет результат анализа
func (r *InMemoryAnalysisRepository) SaveResult(ctx context.Context, result *domain.AnalysisResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.analysis[result.VacancyID] = result
	return nil
}

// GetByVacancyID получает результат анализа по ID вакансии
func (r *InMemoryAnalysisRepository) GetByVacancyID(ctx context.Context, vacancyID string) (*domain.AnalysisResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, exists := r.analysis[vacancyID]
	if !exists {
		return nil, nil
	}
	return result, nil
}
