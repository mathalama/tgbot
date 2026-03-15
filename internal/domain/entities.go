package domain

// Vacancy представляет вакансию
type Vacancy struct {
	ID          string
	Title       string
	Content     string
	Link        string
	Recruiter   string // Контакт рекрутера
	Source      string // Источник (название канала)
	PublishedAt string
	Analyzed    bool // Проанализирована ли вакансия
}

// AnalysisResult результат анализа вакансии через AI
type AnalysisResult struct {
	VacancyID  string
	IsSuitable bool
	Reason     string
	Confidence float32
}

// Channel представляет Telegram канал для мониторинга
type Channel struct {
	ID        string
	Username  string
	IsActive  bool
	CreatedAt string
	UpdatedAt string
}

// UserCV представляет CV пользователя
type UserCV struct {
	ID         int
	CVText     string
	UploadedAt string
	Analyzed   bool
	Analysis   string // Умный анализ от ИИ
}

// UserSkill представляет навык из CV
type UserSkill struct {
	ID          int
	CVId        int
	Skill       string  // Название навыка (Go, React, PostgreSQL, etc)
	Category    string  // Категория (Language, Framework, Database, Tool)
	What        *string // Что это (краткое описание технологии)
	Why         *string // Зачем используется (для чего применяется)
	Description *string // Полное описание использования
}

// SkillMatch результат сравнения требуемых скиллов с CV
type SkillMatch struct {
	MatchedSkills   []string // Совпавшие навыки
	MissingSkills   []string // Отсутствующие в CV
	MatchPercentage float32  // Процент совпадения
}
