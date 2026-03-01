package telegram

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"shop-bot/internal/domain"
	"shop-bot/internal/repository"
	"shop-bot/internal/repository/redis"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api             *tgbotapi.BotAPI
	logger          *slog.Logger
	userRepo        domain.UserRepository
	shopService     domain.ShopService
	trackingService domain.TrackingService
	stateMgr        *redis.StateManager
}

func NewBot(token string,
	logger *slog.Logger,
	userRepo domain.UserRepository,
	shopService domain.ShopService,
	trackingService domain.TrackingService,
	stateMgr *redis.StateManager,
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	api.Debug = false

	logger.Info("Authorized on account", "account_name", api.Self.UserName)

	return &Bot{
		api:             api,
		logger:          logger,
		userRepo:        userRepo,
		shopService:     shopService,
		trackingService: trackingService,
		stateMgr:        stateMgr,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	b.logger.Info("Bot started, waiting for messages")

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("Stopping bot")
			b.api.StopReceivingUpdates()
			return

		case update := <-updates:
			go b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleStart(userID int64, user *domain.User) {
	var shopInfo string
	if user.SelectedShopID != "" {
		shopInfo = fmt.Sprintf("\n\n🏪 Выбранный магазин: <b>%s</b>", user.SelectedShopID)
	}

	text := fmt.Sprintf(
		"👋 Привет, %s!\n\n"+
			"Я бот для мониторинга товаров.\n\n"+
			"📋 <b>Доступные команды:</b>\n"+
			"/shops — список магазинов\n"+
			"/setshop — выбрать магазин\n"+
			"/products — товары выбранного магазина\n"+
			"/search — поиск товара\n"+
			"/track — отследить появление товара%s",
		user.FirstName, shopInfo,
	)

	b.sendMessage(userID, text)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		go b.handleCallback(ctx, update.CallbackQuery)
		return
	}
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID
	text := msg.Text

	user, err := b.getOrCreateUser(ctx, msg.From)
	if err != nil {
		b.logger.Error("Error getting/creating user",
			"error", err)
		b.sendMessage(userID, "❌ Ошибка. Попробуйте позже.")
		return
	}

	b.logger.Debug("message received",
		"user_id", msg.From.ID,
		"username", msg.From.UserName,
		"text", msg.Text,
	)

	state, err := b.stateMgr.GetState(ctx, userID)
	if err != nil {
		b.logger.Error("failed to get state", "error", err)
		state = &redis.UserState{Step: "idle"}
	}

	if state.Step != "idle" && !msg.IsCommand() {
		b.handleStateInput(ctx, user, state, text)
		return
	}

	if state.Step != "idle" && !msg.IsCommand() {
		log.Println("clear")
		b.stateMgr.ClearState(ctx, userID)
	}

	b.logger.Info("debug state",
		"step", state.Step,
		"is_command", msg.IsCommand(),
		"text", text,
	)

	switch {
	case text == "/start":
		b.handleStart(userID, user)
	case text == "/shops":
		b.handleShops(ctx, userID)
	case strings.HasPrefix(text, "/setshop"):
		args := strings.TrimSpace(strings.TrimPrefix(text, "/setshop"))
		b.handleSetShop(ctx, userID, user, args)
	case text == "/products":
		b.handleProducts(ctx, userID, user, 0)
	case strings.HasPrefix(text, "/search"):
		args := strings.TrimSpace(strings.TrimPrefix(text, "/search"))
		b.handleSearch(ctx, userID, user, args)
	case strings.HasPrefix(text, "/track"):
		args := strings.TrimSpace(strings.TrimPrefix(text, "/track"))
		b.handleTrack(ctx, userID, user, args)
	default:
		b.sendMessage(userID, "❓ Неизвестная команда. Используйте /start")
	}
}

func (b *Bot) handleStateInput(ctx context.Context, user *domain.User, state *redis.UserState, text string) {
	switch state.Step {
	case "waiting_shop":
		b.handleShopInput(ctx, user.TelegramID, text)
	case "waiting_search":
		b.handleSearchInput(ctx, user, text)
	}
}

func (b *Bot) getOrCreateUser(ctx context.Context, from *tgbotapi.User) (*domain.User, error) {
	user, err := b.userRepo.GetByTelegramID(ctx, from.ID)
	if err == nil {
		return user, nil
	}

	if err == repository.ErrNotFound {
		newUser := &domain.User{
			TelegramID: from.ID,
			Username:   from.UserName,
			FirstName:  from.FirstName,
			LastName:   from.LastName,
		}

		if err := b.userRepo.Create(ctx, newUser); err != nil {
			return nil, err
		}

		return newUser, nil
	}

	return nil, err
}

// обработка команды /shops
func (b *Bot) handleShops(ctx context.Context, userID int64) {
	shops, err := b.shopService.GetShops(ctx)
	if err != nil {
		b.logger.Error("Error getting shops",
			"error", err)
		b.sendMessage(userID, "❌ Не удалось получить список магазинов")
		return
	}

	if len(shops) == 0 {
		b.sendMessage(userID, "📭 Нет доступных магазинов")
		return
	}

	var sb strings.Builder
	sb.WriteString("🏪 <b>Доступные магазины:</b>\n\n")

	for _, shop := range shops {
		sb.WriteString(fmt.Sprintf("• <b>%s</b> (ID: <code>%s</code>)\n", shop.Name, shop.ID))
	}

	sb.WriteString("\nИспользуйте /setshop <i>ID</i> для выбора")

	b.sendMessage(userID, sb.String())
}

func (b *Bot) handleShopInput(ctx context.Context, userID int64, shopID string) {
	b.stateMgr.ClearState(ctx, userID)

	user, err := b.userRepo.GetByTelegramID(ctx, userID)
	if err != nil {
		b.logger.Error("failed to get user", "error", err)
		b.sendMessage(userID, "❌ Ошибка. Попробуйте /setshop снова.")
		return
	}

	b.setShop(ctx, userID, user, shopID)
}

// обработка команды /setshop
func (b *Bot) setShop(ctx context.Context, userID int64, user *domain.User, args string) {
	shops, err := b.shopService.GetShops(ctx)
	if err != nil {
		b.logger.Error("Error validating shop",
			"error", err)
		b.sendMessage(userID, "❌ Ошибка проверки магазина")
		return
	}

	valid := false
	var shopName string
	for _, s := range shops {
		if s.ID == args {
			valid = true
			shopName = s.Name
			break
		}
	}

	if !valid {
		b.sendMessage(userID, "❌ Магазин не найден. Используйте /shops для списка.")
		return
	}

	if err := b.userRepo.UpdateSelectedShop(ctx, user.TelegramID, args); err != nil {
		b.logger.Error("Error saving shop",
			"error", err)
		b.sendMessage(userID, "❌ Не удалось сохранить выбор")
		return
	}

	user.SelectedShopID = args

	b.sendMessage(userID, fmt.Sprintf("✅ Выбран магазин: <b>%s</b>\n\nТеперь доступны:\n/products — все товары\n/search — поиск", shopName))
}

func (b *Bot) handleSetShop(ctx context.Context, userID int64, user *domain.User, args string) {
	if args != "" {
		b.setShop(ctx, userID, user, args)
		return
	}

	b.stateMgr.SetState(ctx, userID, &redis.UserState{
		Step: "waiting_shop",
	})

	b.sendMessage(userID, "Введите ID магазина:\n\nИли используйте /shops для списка")
}

const productsPerPage = 6

// обрабатывает получение списка продуктов в магазине /products
func (b *Bot) handleProducts(ctx context.Context, userID int64, user *domain.User, page int) {
	// проверка выбран ли магазин
	if user.SelectedShopID == "" {
		b.sendMessage(userID, "❌ Сначала выберите магазин: /shops")
		return
	}

	shopName, err := b.shopService.GetShopName(ctx, user.SelectedShopID)
	if err != nil {
		shopName = user.SelectedShopID
	}

	// получаем товары
	products, err := b.shopService.GetProducts(ctx, user.SelectedShopID)
	if err != nil {
		b.logger.Error("Error getting products",
			"error", err)
		b.sendMessage(userID, "❌ Не удалось получить товары")
		return
	}

	if len(products) == 0 {
		b.sendMessage(userID, "📭 В магазине нет товаров")
		return
	}

	totalPages := (len(products) + productsPerPage - 1) / productsPerPage
	if page < 0 {
		page = 0
	}

	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * productsPerPage
	end := start + productsPerPage
	if end > len(products) {
		end = len(products)
	}

	var sb strings.Builder
	// sb.WriteString(fmt.Sprintf("📦 <b>Товары магазина %s</b>\n\n", shopName))
	sb.WriteString(fmt.Sprintf("📦 <b>%s</b> (страница %d/%d)\n\n", shopName, page+1, totalPages))

	for i := start; i < end; i++ {
		p := products[i]
		num := i + 1
		stock := "❌ Нет"
		if p.Availability > 0 {
			stock = fmt.Sprintf("✅ %d шт.", p.Availability)
		}
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n   %s | <a href=\"%s\">Ссылка</a>\n\n",
			num, p.Name, stock, p.URL))
	}

	var keyboard [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	if page > 0 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("◀️ Назад",
			fmt.Sprintf("products:%d", page-1),
		))
	}

	if page < totalPages-1 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			"Вперёд ▶️",
			fmt.Sprintf("products:%d", page+1),
		))
	}

	if len(row) > 0 {
		keyboard = append(keyboard, row)
	}

	msg := tgbotapi.NewMessage(userID, sb.String())
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	userID := callback.From.ID

	if strings.HasPrefix(data, "products:") {
		var page int
		fmt.Sscanf(data, "products:%d", &page)

		user, err := b.userRepo.GetByTelegramID(ctx, userID)
		if err != nil {
			b.api.Request(tgbotapi.NewCallback(callback.ID, "Ошибка"))
			return
		}

		b.api.Request(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))

		b.handleProducts(ctx, userID, user, page)

		b.api.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

func (b *Bot) handleSearch(ctx context.Context, userID int64, user *domain.User, query string) {
	if user.SelectedShopID == "" {
		b.sendMessage(userID, "❌ Сначала выберите магазин: /shops")
		return
	}

	if query == "" {
		b.stateMgr.SetState(ctx, userID, &redis.UserState{
			Step: "waiting_search",
		})

		b.sendMessage(userID, "Введите название товара для поиска:")
		return
	}

	b.productSearch(ctx, user, query)
}

func (b *Bot) handleSearchInput(ctx context.Context, user *domain.User, query string) {
	b.stateMgr.ClearState(ctx, user.TelegramID)

	b.productSearch(ctx, user, query)
}

func (b *Bot) productSearch(ctx context.Context, user *domain.User, query string) {
	if len(query) < 2 {
		b.sendMessage(user.TelegramID, "❌ Запрос слишком короткий (минимум 2 символа)")
		return
	}

	products, err := b.shopService.SearchProducts(ctx, user.SelectedShopID, query)
	if err != nil {
		b.logger.Error("failed to search products", "error", err)
		b.sendMessage(user.TelegramID, "❌ Ошибка поиска")
		return
	}

	if len(products) == 0 {
		b.sendMessage(user.TelegramID, fmt.Sprintf("🔍 По запросу <b>%s</b> ничего не найдено", query))
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 <b>Результаты поиска \"%s\"</b>\n\n", query))

	for i, p := range products {
		stock := "❌ Нет"
		if p.Availability > 0 {
			stock = fmt.Sprintf("✅ %d шт.", p.Availability)
		}
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n   %s | <a href=\"%s\">Ссылка</a>\n\n", i+1, p.Name, stock, p.URL))
	}

	b.sendMessage(user.TelegramID, sb.String())
}

func (b *Bot) handleTrack(ctx context.Context, userID int64, user *domain.User, query string) {
	b.logger.Debug("handleTrack", "User ID", user.ID, "Telegram ID", user.TelegramID)
	// Проверяем, выбран ли магазин
	if user.SelectedShopID == "" {
		b.sendMessage(userID, "❌ Сначала выберите магазин: /shops")
		return
	}

	shopName, err := b.shopService.GetShopName(ctx, user.SelectedShopID)
	if err != nil {
		shopName = user.SelectedShopID
	}

	// Проверка на пустой запрос
	if query == "" {
		b.sendMessage(userID, "Введите название товара для отслеживания:\n<code>/track молоко 3.2</code>")
		return
	}

	if len(query) < 2 {
		b.sendMessage(userID, "❌ Запрос слишком короткий (минимум 2 символа)")
		return
	}

	task, err := b.trackingService.CreateTask(ctx, user.ID, user.SelectedShopID, query)
	if err != nil {
		b.logger.Error("failed to create tracking task",
			"error", err,
			"user_id", userID,
			"query", query)
		b.sendMessage(userID, "❌ Не удалось создать задачу отслеживания")
		return
	}

	var targetInfo string
	if task.TargetName != "" {
		targetInfo = fmt.Sprintf("\n\nНайден товар: <b>%s</b>", task.TargetName)
	}

	b.sendMessage(userID, fmt.Sprintf(
		"✅ Задача отслеживания создана!\n\n"+
			"🔍 Ищем: <b>%s</b>\n"+
			"🏪 Магазин: %s\n"+"%s\n\n"+
			"Когда товар появится в наличии — пришлю уведомление.",
		task.Query, shopName, targetInfo,
	))
}

func (b *Bot) SendNotification(userID int64, text string) error {
	b.sendMessage(userID, text)
	return nil
}
