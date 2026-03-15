package domain

import "context"

// VacancyRepository интерфейс для работы с вакансиями
type VacancyRepository interface {
	Save(ctx context.Context, vacancy *Vacancy) error
	FindByID(ctx context.Context, id string) (*Vacancy, error)
	GetUnanalyzed(ctx context.Context, limit int) ([]*Vacancy, error)
	GetAllVacancies(ctx context.Context) ([]*Vacancy, error)
	MarkAsAnalyzed(ctx context.Context, id string) error
	GetBySource(ctx context.Context, source string) ([]*Vacancy, error)
	IsSent(ctx context.Context, id string) (bool, error)
	MarkAsSent(ctx context.Context, id string) error
	DeleteOldVacancies(ctx context.Context, daysOld int) (int, error)
	DeleteAll(ctx context.Context) error
}

// ChannelRepository интерфейс для работы с каналами
type ChannelRepository interface {
	Save(ctx context.Context, channel *Channel) error
	GetAll(ctx context.Context) ([]*Channel, error)
	GetByID(ctx context.Context, id string) (*Channel, error)
	Delete(ctx context.Context, id string) error
}

// AnalysisRepository интерфейс для хранения результатов анализа
type AnalysisRepository interface {
	SaveResult(ctx context.Context, result *AnalysisResult) error
	GetByVacancyID(ctx context.Context, vacancyID string) (*AnalysisResult, error)
}

// CVRepository интерфейс для работы с CV пользователя
type CVRepository interface {
	SaveCV(ctx context.Context, cv *UserCV) (int, error)
	GetLatestCV(ctx context.Context) (*UserCV, error)
	MarkAsAnalyzed(ctx context.Context, cvID int) error
}

// SkillRepository интерфейс для работы с навыками из CV
type SkillRepository interface {
	SaveSkills(ctx context.Context, cvID int, skills []*UserSkill) error
	GetUserSkills(ctx context.Context) ([]*UserSkill, error)
	GetSkillsByCategory(ctx context.Context, category string) ([]*UserSkill, error)
}
