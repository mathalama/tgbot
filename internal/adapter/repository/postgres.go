package repository

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"

	"tgbot/internal/domain"
)

// PostgresVacancyRepository хранит вакансии в PostgreSQL
type PostgresVacancyRepository struct {
	db *sql.DB
}

// NewPostgresVacancyRepository создает новый репозиторий
func NewPostgresVacancyRepository(db *sql.DB) *PostgresVacancyRepository {
	return &PostgresVacancyRepository{db: db}
}

// Save сохраняет вакансию (не переписывает существующую)
func (r *PostgresVacancyRepository) Save(ctx context.Context, vacancy *domain.Vacancy) error {
	query := `
		INSERT INTO vacancies (id, title, content, link, recruiter, source, published_at, created_at, analyzed)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), FALSE)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query,
		vacancy.ID,
		vacancy.Title,
		vacancy.Content,
		vacancy.Link,
		vacancy.Recruiter,
		vacancy.Source,
		vacancy.PublishedAt,
	)

	return err
}

// FindByID находит вакансию по ID
func (r *PostgresVacancyRepository) FindByID(ctx context.Context, id string) (*domain.Vacancy, error) {
	query := `SELECT id, title, content, link, recruiter, source, published_at, analyzed FROM vacancies WHERE id = $1`

	var vacancy domain.Vacancy
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&vacancy.ID,
		&vacancy.Title,
		&vacancy.Content,
		&vacancy.Link,
		&vacancy.Recruiter,
		&vacancy.Source,
		&vacancy.PublishedAt,
		&vacancy.Analyzed,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &vacancy, err
}

// GetUnanalyzed получает неанализированные вакансии
func (r *PostgresVacancyRepository) GetUnanalyzed(ctx context.Context, limit int) ([]*domain.Vacancy, error) {
	query := `
		SELECT id, title, content, link, recruiter, source, published_at, analyzed
		FROM vacancies 
		WHERE analyzed = FALSE 
		ORDER BY created_at DESC 
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vacancies []*domain.Vacancy
	for rows.Next() {
		var v domain.Vacancy
		if err := rows.Scan(&v.ID, &v.Title, &v.Content, &v.Link, &v.Recruiter, &v.Source, &v.PublishedAt, &v.Analyzed); err != nil {
			return nil, err
		}
		vacancies = append(vacancies, &v)
	}

	return vacancies, rows.Err()
}

// GetAllVacancies получает все вакансии из БД
func (r *PostgresVacancyRepository) GetAllVacancies(ctx context.Context) ([]*domain.Vacancy, error) {
	query := `
		SELECT id, title, content, link, recruiter, source, published_at, analyzed
		FROM vacancies 
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vacancies []*domain.Vacancy
	for rows.Next() {
		var v domain.Vacancy
		if err := rows.Scan(&v.ID, &v.Title, &v.Content, &v.Link, &v.Recruiter, &v.Source, &v.PublishedAt, &v.Analyzed); err != nil {
			return nil, err
		}
		vacancies = append(vacancies, &v)
	}

	return vacancies, rows.Err()
}

// MarkAsAnalyzed отмечает вакансию как проанализированную
func (r *PostgresVacancyRepository) MarkAsAnalyzed(ctx context.Context, id string) error {
	query := `UPDATE vacancies SET analyzed = TRUE WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// IsSent проверяет была ли вакансия отправлена пользователю
func (r *PostgresVacancyRepository) IsSent(ctx context.Context, id string) (bool, error) {
	query := `SELECT COUNT(*) FROM sent_vacancies WHERE vacancy_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// MarkAsSent отмечает вакансию как отправленную
func (r *PostgresVacancyRepository) MarkAsSent(ctx context.Context, id string) error {
	query := `INSERT INTO sent_vacancies (vacancy_id, sent_at) VALUES ($1, NOW()) ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetBySource получает вакансии из определенного источника
func (r *PostgresVacancyRepository) GetBySource(ctx context.Context, source string) ([]*domain.Vacancy, error) {
	query := `
		SELECT id, title, content, link, recruiter, source, published_at, analyzed
		FROM vacancies 
		WHERE source = $1 
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vacancies []*domain.Vacancy
	for rows.Next() {
		var v domain.Vacancy
		if err := rows.Scan(&v.ID, &v.Title, &v.Content, &v.Link, &v.Recruiter, &v.Source, &v.PublishedAt, &v.Analyzed); err != nil {
			return nil, err
		}
		vacancies = append(vacancies, &v)
	}

	return vacancies, rows.Err()
}

// DeleteOldVacancies удаляет вакансии старше N дней
func (r *PostgresVacancyRepository) DeleteOldVacancies(ctx context.Context, daysOld int) (int, error) {
	query := `DELETE FROM vacancies WHERE created_at < NOW() - INTERVAL '1 day' * $1`
	result, err := r.db.ExecContext(ctx, query, daysOld)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	return int(rows), err
}

// DeleteAll удаляет все вакансии
func (r *PostgresVacancyRepository) DeleteAll(ctx context.Context) error {
	query := `DELETE FROM vacancies`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// PostgresAnalysisRepository хранит результаты анализа
type PostgresAnalysisRepository struct {
	db *sql.DB
}

// NewPostgresAnalysisRepository создает новый репозиторий
func NewPostgresAnalysisRepository(db *sql.DB) *PostgresAnalysisRepository {
	return &PostgresAnalysisRepository{db: db}
}

// SaveResult сохраняет результат анализа
func (r *PostgresAnalysisRepository) SaveResult(ctx context.Context, result *domain.AnalysisResult) error {
	query := `
		INSERT INTO analysis_results (vacancy_id, is_suitable, reason, confidence, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (vacancy_id) DO UPDATE SET
			is_suitable = $2,
			reason = $3,
			confidence = $4
	`

	_, err := r.db.ExecContext(ctx, query,
		result.VacancyID,
		result.IsSuitable,
		result.Reason,
		result.Confidence,
	)

	return err
}

// GetByVacancyID получает результат анализа по ID вакансии
func (r *PostgresAnalysisRepository) GetByVacancyID(ctx context.Context, vacancyID string) (*domain.AnalysisResult, error) {
	query := `
		SELECT vacancy_id, is_suitable, reason, confidence 
		FROM analysis_results 
		WHERE vacancy_id = $1
	`

	var result domain.AnalysisResult
	err := r.db.QueryRowContext(ctx, query, vacancyID).Scan(
		&result.VacancyID,
		&result.IsSuitable,
		&result.Reason,
		&result.Confidence,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &result, err
}

// PostgresCVRepository хранит CV пользователя
type PostgresCVRepository struct {
	db *sql.DB
}

// NewPostgresCVRepository создает новый репозиторий для CV
func NewPostgresCVRepository(db *sql.DB) *PostgresCVRepository {
	return &PostgresCVRepository{db: db}
}

// SaveCV сохраняет CV пользователя
func (r *PostgresCVRepository) SaveCV(ctx context.Context, cv *domain.UserCV) (int, error) {
	query := `
		INSERT INTO user_cv (cv_text, uploaded_at, analyzed)
		VALUES ($1, NOW(), false)
		RETURNING id
	`

	var cvID int
	err := r.db.QueryRowContext(ctx, query, cv.CVText).Scan(&cvID)
	return cvID, err
}

// GetLatestCV получает последнее загруженное CV
func (r *PostgresCVRepository) GetLatestCV(ctx context.Context) (*domain.UserCV, error) {
	query := `
		SELECT id, cv_text, uploaded_at, analyzed 
		FROM user_cv 
		ORDER BY uploaded_at DESC 
		LIMIT 1
	`

	var cv domain.UserCV
	err := r.db.QueryRowContext(ctx, query).Scan(
		&cv.ID,
		&cv.CVText,
		&cv.UploadedAt,
		&cv.Analyzed,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &cv, err
}

// MarkAsAnalyzed отмечает CV как проанализированное
func (r *PostgresCVRepository) MarkAsAnalyzed(ctx context.Context, cvID int) error {
	query := `UPDATE user_cv SET analyzed = true WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, cvID)
	return err
}

// PostgresSkillRepository хранит навыки из CV
type PostgresSkillRepository struct {
	db *sql.DB
}

// NewPostgresSkillRepository создает новый репозиторий для навыков
func NewPostgresSkillRepository(db *sql.DB) *PostgresSkillRepository {
	return &PostgresSkillRepository{db: db}
}

// SaveSkills сохраняет навыки во время анализа CV
func (r *PostgresSkillRepository) SaveSkills(ctx context.Context, cvID int, skills []*domain.UserSkill) error {
	// Сначала удаляем старые навыки для этого CV
	deleteQuery := `DELETE FROM user_skills WHERE cv_id = $1`
	_, err := r.db.ExecContext(ctx, deleteQuery, cvID)
	if err != nil {
		return err
	}

	// Вставляем новые навыки
	insertQuery := `
		INSERT INTO user_skills (cv_id, skill, category, what, why, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (skill) DO UPDATE SET
			cv_id = $1,
			category = $3,
			what = $4,
			why = $5,
			description = $6
	`

	for _, skill := range skills {
		_, err := r.db.ExecContext(ctx, insertQuery,
			cvID,
			skill.Skill,
			skill.Category,
			skill.What,
			skill.Why,
			skill.Description,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetUserSkills получает все навыки пользователя
func (r *PostgresSkillRepository) GetUserSkills(ctx context.Context) ([]*domain.UserSkill, error) {
	query := `
		SELECT id, cv_id, skill, category, what, why, description 
		FROM user_skills 
		ORDER BY category, skill
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*domain.UserSkill
	for rows.Next() {
		var skill domain.UserSkill
		if err := rows.Scan(&skill.ID, &skill.CVId, &skill.Skill, &skill.Category, &skill.What, &skill.Why, &skill.Description); err != nil {
			return nil, err
		}
		skills = append(skills, &skill)
	}

	return skills, rows.Err()
}

// GetSkillsByCategory получает навыки по категории
func (r *PostgresSkillRepository) GetSkillsByCategory(ctx context.Context, category string) ([]*domain.UserSkill, error) {
	query := `
		SELECT id, cv_id, skill, category, what, why, description 
		FROM user_skills 
		WHERE category = $1 
		ORDER BY skill
	`

	rows, err := r.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*domain.UserSkill
	for rows.Next() {
		var skill domain.UserSkill
		if err := rows.Scan(&skill.ID, &skill.CVId, &skill.Skill, &skill.Category, &skill.What, &skill.Why, &skill.Description); err != nil {
			return nil, err
		}
		skills = append(skills, &skill)
	}

	return skills, rows.Err()
}

// PostgresChannelRepository хранит каналы в PostgreSQL
type PostgresChannelRepository struct {
	db *sql.DB
}

// NewPostgresChannelRepository создает новый репозиторий для каналов
func NewPostgresChannelRepository(db *sql.DB) *PostgresChannelRepository {
	return &PostgresChannelRepository{db: db}
}

// Save сохраняет канал
func (r *PostgresChannelRepository) Save(ctx context.Context, channel *domain.Channel) error {
	query := `
		INSERT INTO channels (username, is_active, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (username) DO UPDATE SET
			is_active = $2,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query,
		channel.Username,
		channel.IsActive,
	)

	return err
}

// GetAll получает все каналы
func (r *PostgresChannelRepository) GetAll(ctx context.Context) ([]*domain.Channel, error) {
	query := `
		SELECT id, username, is_active, created_at, updated_at
		FROM channels
		WHERE is_active = true
		ORDER BY username
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*domain.Channel
	for rows.Next() {
		var ch domain.Channel
		if err := rows.Scan(&ch.ID, &ch.Username, &ch.IsActive, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, &ch)
	}

	return channels, rows.Err()
}

// GetByID получает канал по ID
func (r *PostgresChannelRepository) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	query := `
		SELECT id, username, is_active, created_at, updated_at
		FROM channels
		WHERE username = $1
	`

	var ch domain.Channel
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ch.ID,
		&ch.Username,
		&ch.IsActive,
		&ch.CreatedAt,
		&ch.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &ch, err
}

// Delete удаляет канал
func (r *PostgresChannelRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE channels SET is_active = false WHERE username = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
