# 🤖 VacancyBot - CV-Based Job Matching

Умный Telegram бот, который мониторит вакансии в каналах и отправляет вам только те, которые подходят вашему резюме!

## ✨ Возможности

✅ **Автоматический мониторинг каналов** - проверяет каждые 30 минут  
✅ **Интеллектуальное сравнение** - сравнивает требования вакансий с вашим резюме  
✅ **Мгновенные уведомления** - получайте релевантные вакансии сразу в Telegram  
✅ **Автоматическая очистка** - удаляет вакансии старше 7 дней  
✅ **Анализ технологий** - определяет категорию каждой технологии (Language, Framework, Database и т.д.)
✅ **AI-анализ** - использует Google Gemini AI для продвинутого анализа

## 🚀 Быстрый старт

### Требования
- Go 1.20+
- PostgreSQL 15+
- Docker & Docker Compose (опционально)
- Telegram Bot Token
- Google Gemini API Key

### Установка

1. **Clone и перейти в репозиторий:**
```bash
git clone https://github.com/yourusername/vacancies-bot.git
cd vacancies-bot
```

2. **Создать .env файл:**
```bash
cp .env.example .env
# Отредактировать .env с вашими данными
```

3. **Переменные окружения:**
```
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_CHAT_ID=your_chat_id_here
GEMINI_API_KEY=your_gemini_api_key
DATABASE_URL=postgres://user:password@localhost:5432/vacancies?sslmode=disable
```

4. **Запустить с Docker:**
```bash
docker compose up -d
```

Или локально:
```bash
go build ./cmd/bot
./bot
```

## 📋 Команды

### Управление резюме
- **Отправить файл** - просто пришлите .txt, .pdf или .tex файл своего резюме

### Управление каналами
- `/add_channel @golang_jobs` - добавить канал для мониторинга
- `/remove_channel @golang_jobs` - удалить канал
- `/my_channels` - список активных каналов

### Просмотр информации
- `/my_cv` - анализ вашего резюме (все найденные технологии)
- `/vacancies` - все сохраненные вакансии
- `/stats` - статистика совпадений

### Управление
- `/check` - вручную проверить все каналы
- `/clear_vacancies` - удалить все вакансии из БД
- `/help` - справка

## 🏗️ Архитектура

```
vacancies-bot/
├── cmd/bot/              # Entry point
├── internal/
│   ├── adapter/          # Адаптеры (БД, HTTP, Telegram)
│   ├── domain/           # Бизнес логика (entities, interfaces)
│   └── usecase/          # Use cases (применение логики)
├── pkg/                  # Общие утилиты
├── migrations.sql        # Миграции БД
├── Dockerfile            # Docker образ
├── docker-compose.yml    # Оркестрация
└── go.mod               # Зависимости
```

## 🔄 Как это работает

1. **Каждые 30 минут запускается периодическая проверка**
   - Бот получает список всех добавленных каналов
   - Парсит сообщения из каждого канала через Telegram Web API

2. **Для каждой вакансии:**
   - Извлекает текст и заголовок
   - Анализирует требуемые технологии и навыки
   - Сравнивает с вашим резюме

3. **Если совпадение ≥ 50%:**
   - Отправляет красиво оформленное уведомление в Telegram
   - Показывает найденные и недостающие навыки

4. **Автоматическая очистка:**
   - Вакансии старше 7 дней удаляются из БД

## 📊 Технологии

- **Backend:** Go 1.25
- **Database:** PostgreSQL 15
- **AI:** Google Gemini API
- **Bot Framework:** telegram-bot-api/v5
- **Containers:** Docker

## 📝 Логирование

Бот ведет подробное логирование всех операций:
```
[INFO] Connected to PostgreSQL
[INFO] 💾 Saving vacancy: Junior Java Developer
[INFO] ⏳ Periodic check: Checking vacancies...
[INFO] ✅ Periodic check completed
```

## 🤝 Contributing

Contributions приветствуются! Пожалуйста:
1. Fork репозиторий
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

MIT License - см. LICENSE файл для деталей

## 🙋 Support

Если у вас есть вопросы или проблемы, откройте Issue в GitHub.

---

**Сделано с ❤️ для поиска идеальной работы**


