# 🤖 VacancyBot - CV-Based Job Matching

Умный Telegram бот, который мониторит вакансии в каналах и отправляет вам только те, которые подходят вашему резюме!

## ✨ Возможности

✅ **Автоматический мониторинг каналов** - проверяет каждые 30 минут  
✅ **Интеллектуальное сравнение** - сравнивает требования вакансий с вашим резюме  
✅ **Мгновенные уведомления** - получайте релевантные вакансии сразу в Telegram  
✅ **Автоматическая очистка** - удаляет вакансии старше 7 дней  
✅ **Анализ технологий** - определяет категорию каждой технологии (Language, Framework, Database и т.д.)
✅ **AI-анализ** - использует Google Gemini AI для продвинутого анализа
✅ **Интуитивный UI** - главное меню с кнопками, inline навигация, подтверждения  
✅ **Полная статистика** - информация о каналах, вакансиях, навыках  
✅ **Дружественные ошибки** - понятные сообщения об ошибках с подсказками
✅ **🚀 Real-time LLM Streaming** - видите анализ в реальном времени по мере генерации AI

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

### 📱 Главное меню
Нажимайте кнопки в главном меню для быстрого доступа или используйте команды:

### Управление резюме
- **Отправить файл** - просто пришлите .txt, .pdf или .tex файл своего резюме

### Управление каналами
- `/add_channel @golang_jobs` - добавить канал для мониторинга
- `/remove_channel @golang_jobs` - удалить канал
- `/my_channels` - список активных каналов

### Просмотр информации
- `/my_cv` - анализ вашего резюме (все найденные технологии с категориями)
- `/stream_analyze` - ⭐ **NEW!** реальное время анализ резюме с потоковым выводом
- `/vacancies` - все сохраненные вакансии (с pagination)
- `/stats` - подробная статистика с рекомендациями
- `/status` - статус бота

### Управление
- `/check` - вручную проверить все каналы
- `/clear_vacancies` - удалить все вакансии (требует подтверждения)
- `/menu` - показать главное меню
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

## � Real-Time LLM Streaming

Наш бот использует **Real-Time Token Streaming** для вывода результатов анализа в реальном времени!

### Как это работает

Вместо того, чтобы ждать полного завершения анализа AI, вы видите:
- ✅ Токены появляются в сообщении по мере их генерации
- ✅ 500ms throttling предотвращает перегрузку API
- ✅ Плавный, визуальный процесс анализа

### Пример использования

```
Вы: /stream_analyze
Бот: ⏳ Analyzing...
     [через 0.5s]
Бот: ⏳ Analyzing...
     Language
     [через 0.5s]
Бот: ⏳ Analyzing...
     Language: Go█
     [через 0.5s]
Бот: ⏳ Analyzing...
     Language: Go, Python, JavaScript█
     [через 0.5s]
Бот: ✅ Found 45 skills across 8 categories!
     Language: Go, Python...
```

📖 **Полная документация:** смотрите [STREAMING.md](STREAMING.md)

Features:
- 🔄 Интеллектуальное батчирование (500ms + размер буфера)
- 📡 Автоматическое регулирование частоты обновлений
- ⚡ Поддержка Google Gemini API streaming
- 🎯 Graceful error handling

## �🔄 Как это работает

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

## 🎨 Интерфейс

🎛️ **Главное меню**
- Быстрые кнопки для всех основных функций
- Доступно после каждого сообщения

📄 **Страницы с навигацией**
- /vacancies показывает 5 вакансий за раз
- Inline кнопки "⬅️ Назад" и "Далее ➡️" для перемещения

✅ **Подтверждение действий**
- Опасные операции требуют активного согласия
- Inline кнопки "✅ Подтвердить" и "❌ Отмена"

❌ **Дружественные ошибки**
- Понятные сообщения об ошибках
- Подсказки "💡" что может быть не так
- Примеры правильного использования команд

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


