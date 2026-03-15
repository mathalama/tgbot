-- Таблица вакансий
CREATE TABLE IF NOT EXISTS vacancies (
    id VARCHAR(255) PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    link TEXT NOT NULL,
    recruiter VARCHAR(255),
    source VARCHAR(255) NOT NULL,
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    analyzed BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_source ON vacancies(source);
CREATE INDEX IF NOT EXISTS idx_created_at ON vacancies(created_at);
CREATE INDEX IF NOT EXISTS idx_analyzed ON vacancies(analyzed);

-- Таблица результатов анализа
CREATE TABLE IF NOT EXISTS analysis_results (
    vacancy_id VARCHAR(255) PRIMARY KEY,
    is_suitable BOOLEAN NOT NULL,
    reason TEXT,
    confidence REAL,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (vacancy_id) REFERENCES vacancies(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_is_suitable ON analysis_results(is_suitable);

-- Таблица отправленных вакансий (для дедупликации)
CREATE TABLE IF NOT EXISTS sent_vacancies (
    vacancy_id VARCHAR(255) PRIMARY KEY,
    sent_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (vacancy_id) REFERENCES vacancies(id) ON DELETE CASCADE
);

-- Таблица CV пользователя
CREATE TABLE IF NOT EXISTS user_cv (
    id SERIAL PRIMARY KEY,
    cv_text TEXT NOT NULL,
    uploaded_at TIMESTAMP DEFAULT NOW(),
    analyzed BOOLEAN DEFAULT FALSE,
    analysis TEXT
);

-- Таблица навыков из CV
CREATE TABLE IF NOT EXISTS user_skills (
    id SERIAL PRIMARY KEY,
    cv_id INTEGER NOT NULL,
    skill VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    what TEXT,
    why TEXT,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (cv_id) REFERENCES user_cv(id) ON DELETE CASCADE,
    UNIQUE(skill)
);

CREATE INDEX IF NOT EXISTS idx_user_skills_cv ON user_skills(cv_id);
CREATE INDEX IF NOT EXISTS idx_user_skills_category ON user_skills(category);

-- Таблица отслеживаемых каналов
CREATE TABLE IF NOT EXISTS channels (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_username ON channels(username);
CREATE INDEX IF NOT EXISTS idx_channels_active ON channels(is_active);
