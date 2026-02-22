package telegram

import (
	"context"
	"fmt"
	"log"
	"shop-bot/internal/domain"
	"shop-bot/internal/repository"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	userRepo    domain.UserRepository
	shopService domain.ShopService
}

// type UserState struct {
// 	Step   string
// 	ShopID string
// }

func NewBot(token string, userRepo domain.UserRepository, shopService domain.ShopService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	api.Debug = false

	log.Printf("Authorized on account %s", api.Self.UserName)

	return &Bot{
		api:         api,
		userRepo:    userRepo,
		shopService: shopService,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	log.Println("Bot started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping bot...")
			b.api.StopReceivingUpdates()
			return

		case update := <-updates:
			// if update.Message != nil {
			// 	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			// }
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
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID
	text := msg.Text

	log.Printf("[%s] %s", msg.From.UserName, text)

	user, err := b.getOrCreateUser(ctx, msg.From)
	if err != nil {
		log.Printf("Error getting/creating user: %v", err)
		b.sendMessage(userID, "❌ Ошибка. Попробуйте позже.")
		return
	}

	switch {
	case text == "/start":
		b.handleStart(userID, user)
	case text == "/shops":
		b.handleShops(ctx, userID)
	case strings.HasPrefix(text, "/setshop"):
		args := strings.TrimSpace(strings.TrimPrefix(text, "/setshop"))
		b.handleSetShop(ctx, userID, user, args)
	case text == "/products":
		b.handleProducts(ctx, userID, user)
	default:
		b.sendMessage(userID, "❓ Неизвестная команда. Используйте /start")
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
		log.Printf("Error getting shops: %v", err)
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

// обработка команды /setshop
func (b *Bot) handleSetShop(ctx context.Context, userID int64, user *domain.User, args string) {
	// если id не передан, то просим ввести
	if args == "" {
		b.sendMessage(userID, "Введите ID магазина:\n<code>/setshop 221918</code>")
		return
	}

	shops, err := b.shopService.GetShops(ctx)
	if err != nil {
		log.Printf("Error validating shop: %v", err)
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
		log.Printf("error saving shop: %v", err)
		b.sendMessage(userID, "❌ Не удалось сохранить выбор")
		return
	}

	user.SelectedShopID = args

	b.sendMessage(userID, fmt.Sprintf("✅ Выбран магазин: <b>%s</b>\n\nТеперь доступны:\n/products — все товары\n/search — поиск", shopName))
}

// обрабатывает получение списка продуктов в магазине /products
func (b *Bot) handleProducts(ctx context.Context, userID int64, user *domain.User) {
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
		log.Printf("Error getting products: %v", err)
		b.sendMessage(userID, "❌ Не удалось получить товары")
		return
	}

	if len(products) == 0 {
		b.sendMessage(userID, "📭 В магазине нет товаров")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📦 <b>Товары магазина %s</b>\n\n", shopName))

	limit := 10
	if len(products) < limit {
		limit = len(products)
	}

	for i := 0; i < limit; i++ {
		p := products[i]
		stock := "❌ Нет"
		if p.Availability > 0 {
			stock = fmt.Sprintf("✅ %d шт.", p.Availability)
		}
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n   %s | <a href=\"%s\">Ссылка</a>\n\n",
			i+1, p.Name, stock, p.URL))

	}

	if len(products) > 10 {
		sb.WriteString(fmt.Sprintf("... и ещё %d товаров\n", len(products)-10))
	}

	b.sendMessage(userID, sb.String())
}
