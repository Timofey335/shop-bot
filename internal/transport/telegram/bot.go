package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"shop-bot/internal/domain"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	userRepo    domain.UserRepository
	shopService domain.ShopService
	// Хранилище состояний пользователя
	userStates map[int64]UserState
}

type UserState struct {
	Step   string
	ShopID string
}
