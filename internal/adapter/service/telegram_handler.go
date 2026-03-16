package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tgbot/internal/domain"
)

// AnalyzeCVExecutor интерфейс для анализа CV
type AnalyzeCVExecutor interface {
	Execute(ctx context.Context, cvText string) error
}

// ManageChannelsExecutor интерфейс для управления каналами
type ManageChannelsExecutor interface {
	AddChannel(ctx context.Context, username string) error
	RemoveChannel(ctx context.Context, username string) error
	GetAllChannels(ctx context.Context) ([]*domain.Channel, error)
}

// FetchAndAnalyzeExecutor интерфейс для проверки вакансий
type FetchAndAnalyzeExecutor interface {
	Execute(ctx context.Context, channels []*domain.Channel) error
}

// TelegramCommandHandler обрабатывает Telegram команды
type TelegramCommandHandler struct {
	bot               *tgbotapi.BotAPI
	chatID            int64
	cvRepo            domain.CVRepository
	skillRepo         domain.SkillRepository
	channelRepo       domain.ChannelRepository
	vacancyRepo       domain.VacancyRepository
	cvAnalyzer        *CVAnalyzer
	streamingAnalyzer *StreamingCVAnalyzer
	analyzeCVUC       AnalyzeCVExecutor
	manageChannelsUC  ManageChannelsExecutor
	fetchAnalyzeUC    FetchAndAnalyzeExecutor
	vacancyPage       int // текущая страница вакансий
}

// NewTelegramCommandHandler создает новый обработчик команд
func NewTelegramCommandHandler(
	bot *tgbotapi.BotAPI,
	chatID int64,
	cvRepo domain.CVRepository,
	skillRepo domain.SkillRepository,
	channelRepo domain.ChannelRepository,
	vacancyRepo domain.VacancyRepository,
	cvAnalyzer *CVAnalyzer,
	analyzeCVUC AnalyzeCVExecutor,
	manageChannelsUC ManageChannelsExecutor,
	fetchAnalyzeUC FetchAndAnalyzeExecutor,
) *TelegramCommandHandler {
	handler := &TelegramCommandHandler{
		bot:              bot,
		chatID:           chatID,
		cvRepo:           cvRepo,
		skillRepo:        skillRepo,
		channelRepo:      channelRepo,
		vacancyRepo:      vacancyRepo,
		cvAnalyzer:       cvAnalyzer,
		analyzeCVUC:      analyzeCVUC,
		manageChannelsUC: manageChannelsUC,
		fetchAnalyzeUC:   fetchAnalyzeUC,
	}

	// Инициализируем streaming analyzer если есть API key
	// (ленивая инициализация в cmdStreamAnalyze)

	return handler
}

// HandleMessage обрабатывает входящее сообщение
func (h *TelegramCommandHandler) HandleMessage(ctx context.Context, update tgbotapi.Update) error {
	if update.Message == nil {
		return nil
	}

	// Игнорируем сообщения не из нашего чата
	if update.Message.Chat.ID != h.chatID {
		return nil
	}

	message := update.Message.Text
	log.Printf("[TG] Received: %s", message)

	// Проверяем на нажатие кнопок меню
	switch message {
	case "📋 Мои вакансии":
		h.vacancyPage = 1
		return h.cmdVacancies(ctx, 1)
	case "📊 Статистика":
		return h.cmdStats(ctx)
	case "👤 Мой CV":
		return h.cmdMyCV(ctx)
	case "🔗 Каналы":
		return h.cmdMyChannels(ctx)
	case "⚡ Проверить":
		return h.cmdCheck(ctx)
	case "ℹ️ Помощь":
		return h.cmdHelp(ctx)
	}

	// Обработка команд
	if strings.HasPrefix(message, "/") {
		return h.handleCommand(ctx, update)
	}

	return nil
}

// handleCommand обрабатывает Telegram команду
func (h *TelegramCommandHandler) handleCommand(ctx context.Context, update tgbotapi.Update) error {
	message := update.Message
	text := message.Text
	args := strings.Fields(text)

	if len(args) == 0 {
		return nil
	}

	command := args[0]

	switch command {
	case "/start":
		return h.cmdStart(ctx)
	case "/menu":
		return h.sendMessageWithMenu("📋 Главное меню", "Выберите нужное действие:")
	case "/help":
		return h.cmdHelp(ctx)
	case "/add_channel":
		if len(args) < 2 {
			return h.sendMessage("❌ Использование: /add_channel @channel_name")
		}
		return h.cmdAddChannel(ctx, args[1])
	case "/remove_channel":
		if len(args) < 2 {
			return h.sendMessage("❌ Использование: /remove_channel @channel_name")
		}
		return h.cmdRemoveChannel(ctx, args[1])
	case "/my_channels":
		return h.cmdMyChannels(ctx)
	case "/my_cv":
		return h.cmdMyCV(ctx)
	case "/stats":
		return h.cmdStats(ctx)
	case "/status":
		return h.cmdStatus(ctx)
	case "/check":
		return h.cmdCheck(ctx)
	case "/vacancies":
		h.vacancyPage = 1
		return h.cmdVacancies(ctx, 1)
	case "/clear_vacancies":
		return h.cmdClearVacanciesConfirm(ctx)
	case "/stream_analyze":
		return h.cmdStreamAnalyze(ctx)
	default:
		return h.sendMessage(fmt.Sprintf("❌ Неизвестная команда: %s\nУпишите /help для справки", command))
	}
}

// handleDocument обрабатывает загруженный документ (CV)
func (h *TelegramCommandHandler) HandleDocument(ctx context.Context, update tgbotapi.Update, fileBytes []byte) error {
	if update.Message == nil || update.Message.Document == nil {
		return nil
	}

	// Игнорируем сообщения не из нашего чата
	if update.Message.Chat.ID != h.chatID {
		return nil
	}

	fileName := update.Message.Document.FileName
	fileSize := update.Message.Document.FileSize

	if fileSize > 1000000 { // 1MB max
		return h.sendMessage("❌ Файл слишком большой (макс 1MB)\n\n💡 Подсказка: Убедитесь что отправляете только текст CV без шрифтов и изображений")
	}

	// Проверяем расширение
	if !strings.HasSuffix(fileName, ".txt") && !strings.HasSuffix(fileName, ".pdf") && !strings.HasSuffix(fileName, ".tex") {
		return h.sendMessage(fmt.Sprintf("❌ Формат файла не поддерживается: %s\n\n✅ Поддерживаемые форматы:\n• .txt (текстовый файл)\n• .pdf (документ PDF)\n• .tex (LaTeX)", fileName))
	}

	log.Printf("[TG] Received CV file: %s (%d bytes)", fileName, fileSize)

	// Преобразуем bytes в текст
	cvText := string(fileBytes)

	if len(cvText) == 0 {
		return h.sendMessage("❌ Файл пустой\n\n💡 Подсказка: Отправьте непустой файл с вашим резюме")
	}

	if len(cvText) < 50 {
		return h.sendMessage(fmt.Sprintf("⚠️ Файл содержит очень мало текста (%d символов)\n\n💡 Рекомендуем добавить больше информации в резюме для точнее анализа", len(cvText)))
	}

	// Анализируем CV
	waitMsg, _ := h.sendMessageSimple("⏳ Анализирую резюме...")

	err := h.analyzeCVUC.Execute(ctx, cvText)
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Ошибка при анализе резюме\n\n%v", err)
		if strings.Contains(err.Error(), "API") || strings.Contains(err.Error(), "api") {
			errorMsg = "❌ Ошибка подключения к AI сервису\n\n💡 Проверьте интернет соединение и попробуйте позже"
		}
		h.editMessageText(waitMsg, errorMsg)
		return nil
	}

	// Получаем количество навыков
	skills, err := h.skillRepo.GetUserSkills(ctx)
	if err != nil {
		h.editMessageText(waitMsg, fmt.Sprintf("❌ Ошибка при сохранении: %v\n\n💡 Попробуйте загрузить резюме снова", err))
		return nil
	}

	if len(skills) == 0 {
		h.editMessageText(waitMsg, "⚠️ Резюме загружено, но не удалось найти технологии\n\n💡 Убедитесь что в резюме есть названия известных инструментов, фреймворков или языков программирования")
		return nil
	}

	// Группируем по категориям
	categories := make(map[string]int)
	for _, skill := range skills {
		categories[skill.Category]++
	}

	// Строим текст с категориями
	categoryText := ""
	for cat, count := range categories {
		categoryText += fmt.Sprintf("  • %s: %d\n", cat, count)
	}

	h.editMessageText(waitMsg, fmt.Sprintf(
		"✅ Резюме успешно загружено!\n"+
			"📊 Найдено навыков: %d\n"+
			"\n"+
			"Категории:\n"+
			"%s\n"+
			"📌 Теперь добавьте каналы командой /add_channel",
		len(skills),
		categoryText,
	))

	return nil
}

// cmdStart - команда /start
func (h *TelegramCommandHandler) cmdStart(ctx context.Context) error {
	text := `🤖 Добро пожаловать в VacancyBot!

Я постоянно мониторю Telegram каналы и отправляю тебе вакансии, которые идеально подходят твоему резюме.

⚙️ Как это работает:
1. Загрузи свое резюме (.txt, .pdf или .tex)
2. Добавь каналы для мониторинга (/add_channel)
3. Я буду проверять все каналы каждые 30 минут ⏰
4. Когда найду подходящие вакансии - сразу отправлю тебе

✨ Фишки:
🔄 Автоматическая проверка каждые 30 минут
🗑️ Автоматическое удаление вакансий старше 7 дней
📊 Интеллектуальное сравнение с твоим резюме
💬 Мгновенные уведомления о новых вакансиях

📖 Основные команды:
/help - полная справка
/my_cv - мой анализ резюме
/my_channels - мои каналы
/stats - статистика совпадений

Поехали? 🚀`

	return h.sendMessage(text)
}

// cmdHelp - команда /help
func (h *TelegramCommandHandler) cmdHelp(ctx context.Context) error {
	text := `📖 ПОЛНАЯ СПРАВКА

━━━━━━━━━━━━━━━━━━━━━━━━━━
📄 УПРАВЛЕНИЕ РЕЗЮМЕ
━━━━━━━━━━━━━━━━━━━━━━━━━━
Пошли файл вашего резюме (.txt, .pdf или .tex)
И я буду сравнивать с ним все найденные вакансии

━━━━━━━━━━━━━━━━━━━━━━━━━━
🔗 УПРАВЛЕНИЕ КАНАЛАМИ
━━━━━━━━━━━━━━━━━━━━━━━━━━
/add_channel @golang_jobs
   → Добавить канал для мониторинга

/remove_channel @golang_jobs
   → Удалить канал из отслеживаемых

/my_channels
   → Список всех активных каналов

━━━━━━━━━━━━━━━━━━━━━━━━━━
💼 ПРОСМОТР ВАКАНСИЙ
━━━━━━━━━━━━━━━━━━━━━━━━━━
/check
   → Вручную проверить прямо сейчас
   (обычно не нужна - проверка идет автоматически каждые 30 минут ⏰)

/vacancies
   → Показать все сохраненные вакансии

/clear_vacancies
   → Удалить все сохраненные вакансии из БД

━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 ИНФОРМАЦИЯ
━━━━━━━━━━━━━━━━━━━━━━━━━━
/my_cv
   → Как я распарсил вашу резюме
   → Список найденных навыков

/stats
   → Статистика совпадений

/stream_analyze (ЭКСПЕРИМЕНТАЛЬНО)
   → Анализ CV с real-time стримингом 🚀
   → Вы видите результаты в реальном времени!

/help
   → Эта справка

━━━━━━━━━━━━━━━━━━━━━━━━━━
⚙️ КАК ЭТО РАБОТАЕТ?
━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Каждые 30 минут я проверяю все ваши каналы
✓ Сравниваю каждую вакансию с вашим резюме
✓ Если совпадение ≥50% - отправляю уведомление
✓ Вакансии удаляются через 7 дней автоматически

Вы получаете только релевантные предложения! 🎯`

	return h.sendMessage(text)
}

// cmdAddChannel - добавить канал
func (h *TelegramCommandHandler) cmdAddChannel(ctx context.Context, channelName string) error {
	// Очищаем имя канала
	channelName = strings.TrimPrefix(channelName, "@")
	channelName = strings.ToLower(strings.TrimSpace(channelName))

	if channelName == "" {
		return h.sendMessage("❌ Неверное название канала\n\n✅ Правильное использование:\n/add_channel @golang_jobs\n/add_channel @python_vacancies")
	}

	// Добавляем канал
	err := h.manageChannelsUC.AddChannel(ctx, channelName)
	if err != nil {
		log.Printf("[ERROR] Failed to add channel: %v", err)
		return h.sendMessage("❌ Ошибка при добавлении канала\n\n💡 Проверьте:\n• Правильное ли название канала?\n• Публичный ли это канал?\n• Доступен ли бот в этом канале?")
	}

	return h.sendMessage(fmt.Sprintf("✅ Канал @%s добавлен!\n\n⏰ Я буду проверять его каждые 30 минут\n📌 Используй /my_channels чтобы увидеть все каналы", channelName))
}

// cmdRemoveChannel - удалить канал
func (h *TelegramCommandHandler) cmdRemoveChannel(ctx context.Context, channelName string) error {
	// Очищаем имя канала
	channelName = strings.TrimPrefix(channelName, "@")
	channelName = strings.ToLower(strings.TrimSpace(channelName))

	if channelName == "" {
		return h.sendMessage("❌ Неверное название канала\n\n✅ Правильное использование:\n/remove_channel @golang_jobs")
	}

	// Удаляем канал
	err := h.manageChannelsUC.RemoveChannel(ctx, channelName)
	if err != nil {
		log.Printf("[ERROR] Failed to remove channel: %v", err)
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при удалении канала @%s\n\n💡 Канал не найден или уже удален\n\nПосмотреть все каналы: /my_channels", channelName))
	}

	return h.sendMessage(fmt.Sprintf("✅ Канал @%s удален из отслеживаемых", channelName))
}

// cmdMyChannels - список каналов
func (h *TelegramCommandHandler) cmdMyChannels(ctx context.Context) error {
	channels, err := h.manageChannelsUC.GetAllChannels(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to get channels: %v", err)
		return h.sendMessage("❌ Ошибка при получении списка каналов\n\n💡 Попробуйте позже или используйте /add_channel для добавления нового")
	}

	if len(channels) == 0 {
		return h.sendMessage("📭 Нет добавленных каналов\n\n✅ Добавьте первый канал:\n/add_channel @golang_jobs\n/add_channel @python_vacancies\n\n📌 Совет: Найдите каналы на Telegram по ключевым словам типа \"jobs\", \"vacancies\", \"recruitment\"")
	}

	text := "📋 Отслеживаемые каналы:\n\n"
	for i, ch := range channels {
		status := "✅ Активний"
		if !ch.IsActive {
			status = "⏸️ Неактивен"
		}
		text += fmt.Sprintf("%d. %s @%s\n", i+1, status, ch.Username)
	}

	text += fmt.Sprintf("\n💾 Всего: %d канал(ов)", len(channels))

	return h.sendMessage(text)
}

// cmdMyCV - показать полный профиль и навыки
func (h *TelegramCommandHandler) cmdMyCV(ctx context.Context) error {
	skills, err := h.skillRepo.GetUserSkills(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to get skills: %v", err)
		return h.sendMessage("❌ Ошибка при получении навыков\n\n💡 Загрузите резюме командой отправив файл")
	}

	if len(skills) == 0 {
		return h.sendMessage("📭 Резюме еще не загружено\n\n✅ Что делать:\n1. Напишите резюме в файл (формат .txt, .pdf или .tex)\n2. Отправьте файл боту\n3. Я анализирую навыки и буду сравнивать с вакансиями\n\n📌 Пример резюме:\n- Go, Python, JavaScript\n- React, Vue, Django\n- PostgreSQL, MongoDB\n- Docker, Kubernetes")
	}

	// Группируем по категориям с правильным порядком
	categories := make(map[string][]*domain.UserSkill)

	// Фиксированный порядок категорий для вывода
	categoryOrder := []string{
		"Language",
		"Framework",
		"Database",
		"Platform",
		"Tool",
		"Soft Skill",
	}

	categoryIcons := map[string]string{
		"Language":   "🎨",
		"Database":   "💾",
		"Framework":  "🏗️",
		"Tool":       "🔧",
		"Platform":   "☁️",
		"Soft Skill": "👤",
	}

	for _, skill := range skills {
		categories[skill.Category] = append(categories[skill.Category], skill)
	}

	// Начало сообщения
	text := fmt.Sprintf(`🎯 МОЙ ТЕХНИЧЕСКИЙ ПРОФИЛЬ

📊 Всего технологий: %d
📈 Уровень: Опытный разработчик

`, len(skills))

	text += "━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"

	// Выводим по категориям в нужном порядке
	outputCount := 0
	for _, category := range categoryOrder {
		catSkills, exists := categories[category]
		if !exists || len(catSkills) == 0 {
			continue // Пропускаем пустые категории
		}

		icon := categoryIcons[category]
		if icon == "" {
			icon = "▸"
		}

		text += fmt.Sprintf("%s <b>%s</b> (%d)\n", icon, strings.ToUpper(category), len(catSkills))

		for j, skill := range catSkills {
			isLast := (j == len(catSkills)-1)
			prefix := "└─ "
			if !isLast {
				prefix = "├─ "
			}

			text += fmt.Sprintf("%s<code>%s</code>\n", prefix, skill.Skill)
		}

		outputCount++
		if outputCount < len(categoryOrder) {
			text += "\n"
		}
	}

	text += "\n━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"
	text += "✨ Как это помогает?\n"
	text += "🎯 Я сравниваю эти технологии с требованиями вакансий\n"
	text += "📊 Отправляю только подходящие предложения (совпадение ≥50%)\n"
	text += "🚀 Помогу найти идеальную позицию!\n"

	return h.sendMessage(text)
}

// cmdStats - статистика
func (h *TelegramCommandHandler) cmdStats(ctx context.Context) error {
	// Получаем статистику
	channels, errCh := h.manageChannelsUC.GetAllChannels(ctx)
	if errCh != nil {
		log.Printf("[ERROR] Failed to get channels for stats: %v", errCh)
		channels = []*domain.Channel{}
	}

	vacancies, errVac := h.vacancyRepo.GetAllVacancies(ctx)
	if errVac != nil {
		log.Printf("[ERROR] Failed to get vacancies for stats: %v", errVac)
		vacancies = []*domain.Vacancy{}
	}

	skills, errSkills := h.skillRepo.GetUserSkills(ctx)
	if errSkills != nil {
		log.Printf("[ERROR] Failed to get skills for stats: %v", errSkills)
		skills = []*domain.UserSkill{}
	}

	latestCV, _ := h.cvRepo.GetLatestCV(ctx)

	// Считаем анализированные вакансии
	analyzedCount := 0
	for _, v := range vacancies {
		if v.Analyzed {
			analyzedCount++
		}
	}

	text := `📊 СТАТИСТИКА

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📈 ОСНОВНАЯ ИНФОРМАЦИЯ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`

	if latestCV != nil {
		text += fmt.Sprintf("✅ Резюме загружено: %s назад\n", formatTime(time.Until(parseTime(latestCV.UploadedAt))))
	} else {
		text += "❌ Резюме не загружено\n"
	}

	text += fmt.Sprintf(`
🔗 Отслеживаемые каналы: %d
📋 Всего вакансий: %d
✅ Анализировано: %d / %d
💾 Найденных навыков: %d

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎯 НАВЫКИ ПО КАТЕГОРИЯМ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━`,
		len(channels),
		len(vacancies),
		analyzedCount,
		len(vacancies),
		len(skills),
	)

	// Группируем навыки по категориям
	categories := make(map[string]int)
	for _, skill := range skills {
		categories[skill.Category]++
	}

	// Фиксированный порядок категорий
	categoryOrder := []string{
		"Language",
		"Framework",
		"Database",
		"Platform",
		"Tool",
		"Soft Skill",
	}

	for _, cat := range categoryOrder {
		count := categories[cat]
		if count > 0 {
			text += fmt.Sprintf("\n%-12s: %d", cat, count)
		}
	}

	text += `

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✨ СОВЕТЫ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`

	// Даём советы на основе статистики
	if len(channels) == 0 {
		text += "📌 Добавь хотя бы один канал (/add_channel)\n"
	}
	if len(skills) == 0 {
		text += "📌 Загрузи своё резюме для анализа\n"
	}
	if len(vacancies) == 0 {
		text += "📌 Жди автоматической проверки каждые 30 минут\n"
	}
	if len(vacancies) > 0 && analyzedCount < len(vacancies)/2 {
		text += "📌 Много непроанализированных вакансий - проверь их командой /check\n"
	}

	text += `

⏰ Автоматическая проверка: каждые 30 минут
🗑️ Вакансии удаляются: через 7 дней`

	return h.sendMessage(text)
}

// cmdCheck - проверить вакансии в каналах
func (h *TelegramCommandHandler) cmdCheck(ctx context.Context) error {
	// Получаем все активные каналы
	channels, err := h.manageChannelsUC.GetAllChannels(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to get channels: %v", err)
		return h.sendMessage("❌ Ошибка при получении каналов\n\n💡 Попробуйте позже")
	}

	if len(channels) == 0 {
		return h.sendMessage("❌ Нет добавленных каналов\n\n✅ Добавьте каналы:\n/add_channel @golang_jobs")
	}

	h.sendMessage(fmt.Sprintf("⏳ Проверяю %d канал(ов)...\n\n🔍 Это может занять несколько секунд", len(channels)))

	// Запускаем проверку вакансий
	err = h.fetchAnalyzeUC.Execute(ctx, channels)
	if err != nil {
		log.Printf("[ERROR] Failed to check vacancies: %v", err)
		errorMsg := fmt.Sprintf("❌ Ошибка при проверке\n\n%v", err)
		if strings.Contains(err.Error(), "network") || strings.Contains(err.Error(), "connection") {
			errorMsg = "❌ Ошибка подключения\n\n💡 Проверьте интернет и попробуйте позже"
		}
		return h.sendMessage(errorMsg)
	}

	return h.sendMessage("✅ Проверка завершена!\n\n📋 Используйте /vacancies чтобы увидеть все найденные вакансии\n\n⏰ Автоматическая проверка идет каждые 30 минут")
}

// cmdVacancies - показать все сохраненные вакансии с pagination
func (h *TelegramCommandHandler) cmdVacancies(ctx context.Context, page int) error {
	const itemsPerPage = 5

	vacancies, err := h.vacancyRepo.GetAllVacancies(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to get vacancies: %v", err)
		return h.sendMessage("❌ Ошибка при получении вакансий\n\n💡 Попробуйте позже или используйте /check для новой проверки")
	}

	if len(vacancies) == 0 {
		return h.sendMessage("📋 Вакансий пока не найдено\n\n⏰ Авто проверка каждые 30 минут\n✅ Или проверьте вручную: /check\n\n💡 Убедитесь что:\n• Загружено резюме\n• Добавлены каналы в отслеживание")
	}

	// Рассчитываем pagination
	totalPages := (len(vacancies) + itemsPerPage - 1) / itemsPerPage
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	startIdx := (page - 1) * itemsPerPage
	endIdx := startIdx + itemsPerPage
	if endIdx > len(vacancies) {
		endIdx = len(vacancies)
	}

	// Формируем текст
	text := fmt.Sprintf("📋 Вакансии (страница %d из %d)\n\n", page, totalPages)

	for i := startIdx; i < endIdx; i++ {
		v := vacancies[i]
		analyzed := "✅"
		if !v.Analyzed {
			analyzed = "⏳"
		}

		// Укороченная вакансия для экономии места
		title := v.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		text += fmt.Sprintf("%d. %s\n", i+1, title)
		text += fmt.Sprintf("   %s %s | @%s\n\n",
			analyzed,
			v.Recruiter,
			v.Source,
		)
	}

	// Создаём inline кнопки для pagination
	row := []tgbotapi.InlineKeyboardButton{}

	if page > 1 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("vac_page_%d", page-1)))
	}

	row = append(row, tgbotapi.NewInlineKeyboardButtonData(
		fmt.Sprintf("%d/%d", page, totalPages),
		"vac_info",
	))

	if page < totalPages {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("Далее ➡️", fmt.Sprintf("vac_page_%d", page+1)))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)

	msg := tgbotapi.NewMessage(h.chatID, text)
	msg.ReplyMarkup = keyboard

	_, err = h.bot.Send(msg)
	return err
}

// cmdClearVacanciesConfirm - подтверждение перед очисткой
func (h *TelegramCommandHandler) cmdClearVacanciesConfirm(ctx context.Context) error {
	// Inline кнопки для подтверждения
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить ВСЕ", "clear_confirm_yes"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "clear_confirm_no"),
		),
	)

	msg := tgbotapi.NewMessage(h.chatID, "⚠️ Вы уверены?\n\nЭто удалит ВСЕ вакансии из базы данных 🗑️")
	msg.ReplyMarkup = keyboard

	_, err := h.bot.Send(msg)
	return err
}

// cmdClearVacanciesExecute - непосредственное удаление
func (h *TelegramCommandHandler) cmdClearVacanciesExecute(ctx context.Context) error {
	err := h.vacancyRepo.DeleteAll(ctx)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при удалении: %v", err))
	}
	return h.sendMessage("✅ Все вакансии успешно удалены!\n\nИспользуйте /check для загрузки новых вакансий.")
}

// cmdStatus - статус бота
func (h *TelegramCommandHandler) cmdStatus(ctx context.Context) error {
	channels, _ := h.manageChannelsUC.GetAllChannels(ctx)
	vacancies, _ := h.vacancyRepo.GetAllVacancies(ctx)
	skills, _ := h.skillRepo.GetUserSkills(ctx)

	activeChannels := 0
	for _, ch := range channels {
		if ch.IsActive {
			activeChannels++
		}
	}

	text := `🤖 СТАТУС БОТА

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ БОТ АКТИВЕН И РАБОТАЕТ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Активные каналы: %d / %d
📊 Вакансий в БД: %d
💾 Навыков найдено: %d

🔄 Расписание:
• Автоматическая проверка: каждые 30 минут
• Удаление старых вакансий: через 7 дней

⚙️ Готов к работе!
Напиши /menu для быстрого доступа к функциям`

	return h.sendMessage(fmt.Sprintf(text, activeChannels, len(channels), len(vacancies), len(skills)))
}

// sendMessage отправляет сообщение с главным меню
func (h *TelegramCommandHandler) sendMessage(text string) error {
	msg := tgbotapi.NewMessage(h.chatID, text)
	msg.ReplyMarkup = h.getMainKeyboard()

	_, err := h.bot.Send(msg)
	return err
}

// sendMessageSimple отправляет сообщение БЕЗ меню и возвращает ID
func (h *TelegramCommandHandler) sendMessageSimple(text string) (int, error) {
	msg := tgbotapi.NewMessage(h.chatID, text)

	result, err := h.bot.Send(msg)
	if err != nil {
		return 0, err
	}
	return result.MessageID, nil
}

// sendMessageWithMenu отправляет сообщение с меню
func (h *TelegramCommandHandler) sendMessageWithMenu(title, subtitle string) error {
	text := fmt.Sprintf("💬 %s\n%s", title, subtitle)
	msg := tgbotapi.NewMessage(h.chatID, text)
	msg.ReplyMarkup = h.getMainKeyboard()

	_, err := h.bot.Send(msg)
	return err
}

// getMainKeyboard возвращает главное меню
func (h *TelegramCommandHandler) getMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📋 Мои вакансии"),
			tgbotapi.NewKeyboardButton("📊 Статистика"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👤 Мой CV"),
			tgbotapi.NewKeyboardButton("🔗 Каналы"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚡ Проверить"),
			tgbotapi.NewKeyboardButton("ℹ️ Помощь"),
		),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

// HandleCallbackQuery обрабатывает нажатие inline кнопок
func (h *TelegramCommandHandler) HandleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) error {
	data := query.Data

	// Удаляем "loading" эффект
	cb := tgbotapi.NewCallback(query.ID, "")
	h.bot.Request(cb)

	switch {
	case strings.HasPrefix(data, "vac_page_"):
		pageStr := strings.TrimPrefix(data, "vac_page_")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			return nil
		}
		return h.updateVacanciesPage(ctx, query.Message.MessageID, page)

	case data == "vac_info":
		// Не делаем ничего при нажатии на номер страницы
		return nil

	case data == "clear_confirm_yes":
		h.editMessageText(query.Message.MessageID, "⏳ Удаляю вакансии...")
		return h.cmdClearVacanciesExecute(ctx)

	case data == "clear_confirm_no":
		return h.editMessageText(query.Message.MessageID, "❌ Удаление отменено")

	default:
		return nil
	}
}

// updateVacanciesPage обновляет страницу вакансий
func (h *TelegramCommandHandler) updateVacanciesPage(ctx context.Context, messageID int, page int) error {
	const itemsPerPage = 5

	vacancies, err := h.vacancyRepo.GetAllVacancies(ctx)
	if err != nil {
		return nil
	}

	if len(vacancies) == 0 {
		return nil
	}

	// Рассчитываем pagination
	totalPages := (len(vacancies) + itemsPerPage - 1) / itemsPerPage
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	startIdx := (page - 1) * itemsPerPage
	endIdx := startIdx + itemsPerPage
	if endIdx > len(vacancies) {
		endIdx = len(vacancies)
	}

	// Формируем текст
	text := fmt.Sprintf("📋 Вакансии (страница %d из %d)\n\n", page, totalPages)

	for i := startIdx; i < endIdx; i++ {
		v := vacancies[i]
		analyzed := "✅"
		if !v.Analyzed {
			analyzed = "⏳"
		}

		title := v.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		text += fmt.Sprintf("%d. %s\n", i+1, title)
		text += fmt.Sprintf("   %s %s | @%s\n\n",
			analyzed,
			v.Recruiter,
			v.Source,
		)
	}

	// Создаём inline кнопки
	row := []tgbotapi.InlineKeyboardButton{}

	if page > 1 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("vac_page_%d", page-1)))
	}

	row = append(row, tgbotapi.NewInlineKeyboardButtonData(
		fmt.Sprintf("%d/%d", page, totalPages),
		"vac_info",
	))

	if page < totalPages {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("Далее ➡️", fmt.Sprintf("vac_page_%d", page+1)))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)

	// Редактируем сообщение
	edit := tgbotapi.NewEditMessageText(h.chatID, messageID, text)
	edit.ReplyMarkup = &keyboard

	_, err = h.bot.Send(edit)
	return err
}

// editMessageText редактирует текст сообщения
func (h *TelegramCommandHandler) editMessageText(messageID int, text string) error {
	edit := tgbotapi.NewEditMessageText(h.chatID, messageID, text)
	_, err := h.bot.Send(edit)
	return err
}

// formatTime красиво форматирует время
func formatTime(duration time.Duration) string {
	if duration < 0 {
		return "неизвестно"
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours == 0 && minutes == 0 {
		return "только что"
	}
	if hours == 0 {
		return fmt.Sprintf("%d мин", minutes)
	}
	if hours < 24 {
		return fmt.Sprintf("%d ч %d мин", hours, minutes)
	}

	days := hours / 24
	hours = hours % 24
	if days == 1 {
		return "вчера"
	}
	return fmt.Sprintf("%d дн назад", days)
}

// cmdStreamAnalyze - команда для демонстрации streaming анализа
// Требует загруженного CV файла
func (h *TelegramCommandHandler) cmdStreamAnalyze(ctx context.Context) error {
	// Получаем последнее загруженное резюме
	latestCV, err := h.cvRepo.GetLatestCV(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to get latest CV: %v", err)
		return h.sendMessage("❌ Ошибка при получении резюме\n\n💡 Загрузите резюме сначала")
	}

	if latestCV == nil {
		return h.sendMessage("📭 Резюме не загружено\n\n💡 Отправьте файл своего резюме сначала")
	}

	// Инициализируем streaming analyzer при первом использовании
	if h.streamingAnalyzer == nil {
		// Используем API key из существующего CVAnalyzer
		h.streamingAnalyzer = NewStreamingCVAnalyzer(h.cvAnalyzer.apiKey)
	}

	// Отправляем начальное сообщение
	initialMsg, err := h.sendMessageSimple("⏳ 🔄 Анализирую CV с стримингом...\n\n_Результаты будут обновляться в реальном времени_")
	if err != nil {
		return err
	}

	// Создаем streamer для обновления этого сообщения
	config := StreamConfig{
		UpdateInterval:   500 * time.Millisecond,
		BufferThreshold:  100,
		MaxBufferSize:    1500,
		ShowTypingStatus: true,
	}

	updater := NewStreamingMessageUpdater(h.bot, h.chatID, initialMsg, config)

	// Создаем writer для streaming
	writer := updater.NewStreamWriter(ctx)

	// Запускаем анализ с streaming
	go func() {
		log.Printf("[STREAM] Starting streaming analysis...")

		// Анализируем CV с выводом в stream writer
		skillMap, err := h.streamingAnalyzer.AnalyzeWithStreaming(ctx, latestCV.CVText, writer)
		if err != nil {
			log.Printf("[ERROR] Streaming analysis failed: %v", err)
			updater.Cancel()
			_ = h.sendMessage(fmt.Sprintf("❌ Ошибка при анализе: %v", err))
			return
		}

		// Завершаем streaming
		_ = updater.Flush(ctx)

		log.Printf("[STREAM] Analysis complete, found skills: %d", len(skillMap))

		// Отправляем итоговое резюме
		summary := h.buildSkillSummary(skillMap)
		_ = h.sendMessage(summary)
	}()

	return nil
}

// buildSkillSummary строит красивое резюме найденных навыков
func (h *TelegramCommandHandler) buildSkillSummary(skillMap map[string][]string) string {
	if len(skillMap) == 0 {
		return "⚠️ Навыки не найдены"
	}

	text := "✅ Анализ завершен!\n\n📊 Найденные навыки:\n\n"

	categoryOrder := []string{
		"Language",
		"Framework",
		"Database",
		"Platform",
		"Tool",
		"Soft Skill",
	}

	categoryIcons := map[string]string{
		"Language":   "🎨",
		"Database":   "💾",
		"Framework":  "🏗️",
		"Tool":       "🔧",
		"Platform":   "☁️",
		"Soft Skill": "👤",
	}

	for _, category := range categoryOrder {
		skills, exists := skillMap[category]
		if !exists || len(skills) == 0 {
			continue
		}

		icon := categoryIcons[category]
		if icon == "" {
			icon = "▸"
		}

		text += fmt.Sprintf("%s **%s** (%d):\n", icon, category, len(skills))
		for j, skill := range skills {
			isLast := (j == len(skills)-1)
			prefix := "├─ "
			if isLast {
				prefix = "└─ "
			}
			text += fmt.Sprintf("%s`%s`\n", prefix, skill)
		}
		text += "\n"
	}

	return text
}

// parseTime парсит время из строки
func parseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		// Пытаемся другие форматы
		t, err = time.Parse("2006-01-02 15:04:05", timeStr)
		if err != nil {
			return time.Now()
		}
	}
	return t
}
