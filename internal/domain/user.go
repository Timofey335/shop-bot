package domain

import (
	"context"
	"time"
)

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
	Create(ctx context.Context, user *User) error
	GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)
	UpdateSelectedShop(ctx context.Context, telegramID int64, shopID string) error
}
