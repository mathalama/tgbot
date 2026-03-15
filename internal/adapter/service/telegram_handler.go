package service

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	bot              *tgbotapi.BotAPI
	chatID           int64
	cvRepo           domain.CVRepository
	skillRepo        domain.SkillRepository
	channelRepo      domain.ChannelRepository
	vacancyRepo      domain.VacancyRepository
	cvAnalyzer       *CVAnalyzer
	analyzeCVUC      AnalyzeCVExecutor
	manageChannelsUC ManageChannelsExecutor
	fetchAnalyzeUC   FetchAndAnalyzeExecutor
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
	return &TelegramCommandHandler{
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
	case "/check":
		return h.cmdCheck(ctx)
	case "/vacancies":
		return h.cmdVacancies(ctx)
	case "/clear_vacancies":
		return h.cmdClearVacancies(ctx)
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
		return h.sendMessage("❌ Файл слишком большой (макс 1MB)")
	}

	// Проверяем расширение
	if !strings.HasSuffix(fileName, ".txt") && !strings.HasSuffix(fileName, ".pdf") && !strings.HasSuffix(fileName, ".tex") {
		return h.sendMessage("❌ Поддерживаются только .txt, .pdf и .tex файлы")
	}

	log.Printf("[TG] Received CV file: %s (%d bytes)", fileName, fileSize)

	// Преобразуем bytes в текст
	cvText := string(fileBytes)

	if len(cvText) == 0 {
		return h.sendMessage("❌ Файл пустой")
	}

	// Анализируем CV
	h.sendMessage("⏳ Анализирую резюме...")

	err := h.analyzeCVUC.Execute(ctx, cvText)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при анализе: %v", err))
	}

	// Получаем количество навыков
	skills, err := h.skillRepo.GetUserSkills(ctx)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при сохранении: %v", err))
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

	return h.sendMessage(fmt.Sprintf(
		"✅ Резюме загружено!\n"+
			"📊 Извлечено навыков: %d\n"+
			"\n"+
			"Категории:\n"+
			"%s",
		len(skills),
		categoryText,
	))
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
		return h.sendMessage("❌ Неверное название канала")
	}

	// Добавляем канал
	err := h.manageChannelsUC.AddChannel(ctx, channelName)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при добавлении: %v", err))
	}

	return h.sendMessage(fmt.Sprintf("✅ Канал @%s добавлен!\n\n⏰ Я буду проверять его каждые 30 минут и отправлять подходящие вакансии.", channelName))
}

// cmdRemoveChannel - удалить канал
func (h *TelegramCommandHandler) cmdRemoveChannel(ctx context.Context, channelName string) error {
	// Очищаем имя канала
	channelName = strings.TrimPrefix(channelName, "@")
	channelName = strings.ToLower(strings.TrimSpace(channelName))

	if channelName == "" {
		return h.sendMessage("❌ Неверное название канала")
	}

	// Удаляем канал
	err := h.manageChannelsUC.RemoveChannel(ctx, channelName)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при удалении: %v", err))
	}

	return h.sendMessage(fmt.Sprintf("✅ Канал @%s удален из отслеживаемых.", channelName))
}

// cmdMyChannels - список каналов
func (h *TelegramCommandHandler) cmdMyChannels(ctx context.Context) error {
	channels, err := h.manageChannelsUC.GetAllChannels(ctx)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка: %v", err))
	}

	if len(channels) == 0 {
		return h.sendMessage("📭 Ты еще не добавил ни один канал для мониторинга\n\nНапиши /add_channel @channel_name чтобы начать 🚀")
	}

	text := "📋 Отслеживаемые каналы:\n\n"
	for i, ch := range channels {
		status := "✅"
		if !ch.IsActive {
			status = "❌"
		}
		text += fmt.Sprintf("%d. %s @%s\n", i+1, status, ch.Username)
	}

	return h.sendMessage(text)
}

// cmdMyCV - показать полный профиль и навыки
func (h *TelegramCommandHandler) cmdMyCV(ctx context.Context) error {
	skills, err := h.skillRepo.GetUserSkills(ctx)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка: %v", err))
	}

	if len(skills) == 0 {
		return h.sendMessage("📭 Резюме еще не загружено\n\nПришли файл своего резюме чтобы я мог сделать подробный анализ и сравнивать с вакансиями")
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
	text := "📊 Статистика:\n\n"
	text += "⏳ Функция в разработке\n"

	return h.sendMessage(text)
}

// cmdCheck - проверить вакансии в каналах
func (h *TelegramCommandHandler) cmdCheck(ctx context.Context) error {
	// Получаем все активные каналы
	channels, err := h.manageChannelsUC.GetAllChannels(ctx)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при получении каналов: %v", err))
	}

	if len(channels) == 0 {
		return h.sendMessage("❌ Ты не добавил ни один канал для проверки\n\nНапиши /add_channel @channel_name для начала")
	}

	h.sendMessage(fmt.Sprintf("⏳ Вручную проверяю %d канал(ов) прямо сейчас...", len(channels)))

	// Запускаем проверку вакансий
	err = h.fetchAnalyzeUC.Execute(ctx, channels)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при проверке: %v", err))
	}

	return h.sendMessage("✅ Проверка завершена! Если найдены подходящие вакансии - отправлены выше.\n\n(Обычно это происходит автоматически каждые 30 минут ⏰)")
}

// cmdVacancies - показать все сохраненные вакансии
func (h *TelegramCommandHandler) cmdVacancies(ctx context.Context) error {
	vacancies, err := h.vacancyRepo.GetAllVacancies(ctx)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при получении вакансий: %v", err))
	}

	if len(vacancies) == 0 {
		return h.sendMessage("📋 На данный момент нет сохраненных вакансий\n\nОжидайте ⏰ Проверка идет автоматически каждые 30 минут")
	}

	// Форматируем список вакансий
	text := fmt.Sprintf("📋 Все вакансии (%d):\n\n", len(vacancies))
	for i, vacancy := range vacancies {
		analyzed := "❌"
		if vacancy.Analyzed {
			analyzed = "✅"
		}
		text += fmt.Sprintf("%d. %s\n   📍 @%s | 💼 %s\n   %s Анализировано\n\n",
			i+1,
			vacancy.Title,
			vacancy.Source,
			vacancy.Recruiter,
			analyzed,
		)
	}

	// Если слишком длинный текст, отправляем частями
	if len(text) > 4000 {
		parts := strings.Split(text, "\n\n")
		currentMsg := fmt.Sprintf("📋 Все вакансии (%d):\n\n", len(vacancies))

		for _, part := range parts {
			if len(currentMsg)+len(part)+2 > 4000 {
				h.sendMessage(currentMsg)
				currentMsg = part + "\n\n"
			} else {
				currentMsg += part + "\n\n"
			}
		}

		if currentMsg != "" {
			return h.sendMessage(currentMsg)
		}
	}

	return h.sendMessage(text)
}

// cmdClearVacancies - очистить все вакансии
func (h *TelegramCommandHandler) cmdClearVacancies(ctx context.Context) error {
	err := h.vacancyRepo.DeleteAll(ctx)
	if err != nil {
		return h.sendMessage(fmt.Sprintf("❌ Ошибка при удалении: %v", err))
	}
	return h.sendMessage("✅ Все вакансии успешно удалены!\n\nИспользуйте /check для загрузки новых вакансий.")
}

// sendMessage отправляет сообщение
func (h *TelegramCommandHandler) sendMessage(text string) error {
	msg := tgbotapi.NewMessage(h.chatID, text)

	_, err := h.bot.Send(msg)
	return err
}
