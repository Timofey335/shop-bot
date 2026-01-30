package domain

import "time"

type User struct {
	ID             int64
	TelegramID     int64
	Username       string
	FirstName      string
	LastName       string
	SelectedShopID string
	CreatedAt      time.Time
	UpdateAt       time.Time
}

type UserRepository interface {
	Create(user *User) error
	GetByTelegramID(telegramID int64) (*User, error)
	UpdateSelectedShop(telegramID int64, shopID string) error
}
