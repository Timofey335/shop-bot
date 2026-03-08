package domain

import (
	"context"
	"time"
)

type TrackingStatus string

const (
	StatusActive    TrackingStatus = "active"
	StatusDone      TrackingStatus = "done"
	StatusCancelled TrackingStatus = "cancelled"
)

type TrackingTask struct {
	ID         int64
	UserID     int64
	ShopID     string
	Query      string
	TargetName string
	Status     TrackingStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	NotifiedAt *time.Time
}

type TrackingRepository interface {
	Create(ctx context.Context, task *TrackingTask) error
	// Возвращает активные задачи пользователя
	GetActiveByUser(ctx context.Context, userID int64) ([]TrackingTask, error)
	// Возвращает все задачи для воркера
	// Он будет проходить по ним и проверять наличие товаров
	GetAllActive(ctx context.Context) ([]TrackingTask, error)
	// Меняет статус задачи
	UpdateStatus(ctx context.Context, taskID int64, status TrackingStatus) error
	Delete(ctx context.Context, taskID int64) error
}

type TrackingService interface {
	CreateTask(ctx context.Context, userID int64, shopID, query string) (*TrackingTask, error)
	GetActiveTasks(ctx context.Context) ([]TrackingTask, error)
	CheckTask(ctx context.Context, task *TrackingTask) (bool, error)
	MarkDone(ctx context.Context, taskID int64) error
}
