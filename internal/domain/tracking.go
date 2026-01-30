package domain

import "time"

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
	Create(task *TrackingTask) error
	// Возвращает активные задачи пользователя
	GetActiveByUser(userID int64) ([]TrackingTask, error)
	// Возвращает все задачи для воркера
	// Он будет проходить по ним и проверять наличие товаров
	GetAllActive() ([]TrackingTask, error)
	// Меняет статус задачи
	UpdateStatus(taskID int64, status TrackingStatus) error
	Delete(taskID int64) error
}
