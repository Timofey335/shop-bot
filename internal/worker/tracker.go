package worker

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"shop-bot/internal/domain"
	"time"
)

type Notifier interface {
	SendNotification(userID int64, text string) error
}

type Tracker struct {
	trackingSvc domain.TrackingService
	userRepo    domain.UserRepository
	shopService domain.ShopService
	notifier    Notifier
	logger      *slog.Logger
	interval    time.Duration
}

func NewTracker(svc domain.TrackingService, users domain.UserRepository, shopService domain.ShopService, logger *slog.Logger, notifier Notifier, interval time.Duration) *Tracker {
	return &Tracker{
		trackingSvc: svc,
		userRepo:    users,
		shopService: shopService,
		notifier:    notifier,
		logger:      logger,
		interval:    interval,
	}
}

func (t *Tracker) Start(ctx context.Context) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	t.logger.Info("Tracking worker started")

	t.checkAll(ctx)

	for {
		select {
		case <-ctx.Done():
			t.logger.Info("Tracking worker stopped")
			return
		case <-ticker.C:
			t.checkAll(ctx)
		}
	}
}

func (t *Tracker) checkAll(ctx context.Context) {
	t.logger.Info("Checking tracking tasks...")

	tasks, err := t.trackingSvc.GetActiveTasks(ctx)
	if err != nil {
		t.logger.Error("error getting tasks")
		return
	}

	t.logger.Info("Found active tasks", "num of tasks", len(tasks))

	for _, task := range tasks {
		taskCtx, cancel := context.WithTimeout(ctx, 100*time.Second)

		found, err := t.trackingSvc.CheckTask(taskCtx, &task)
		cancel()

		if err != nil {
			t.logger.Error("error checking task",
				"task_id", task.ID,
				"error", err,
			)
			continue
		}

		if found {
			t.notify(ctx, &task)
		}
	}
}

func (t *Tracker) notify(ctx context.Context, task *domain.TrackingTask) {
	user, err := t.userRepo.GetByID(ctx, task.UserID)
	if err != nil {
		t.logger.Error("error getting user",
			"user ID", task.UserID,
			"error", err,
		)
		return
	}

	shopName, err := t.shopService.GetShopName(task.ShopID)
	if err != nil {
		shopName = task.ShopID
	}

	text := fmt.Sprintf(
		"🔔 <b>Товар появился в наличии!</b>\n\n"+
			"<b>%s</b>\n"+
			"Магазин: %s\n\n"+
			"Больше не отслеживается.",
		task.TargetName, shopName,
	)

	if err := t.notifier.SendNotification(user.TelegramID, text); err != nil {
		t.logger.Error("failed to send notification", "error", err)
		return
	}

	if err := t.trackingSvc.MarkDone(ctx, task.ID); err != nil {
		t.logger.Error("failed to mark task done", "error", err)
		return
	}

	log.Printf("Task %d completed, user notified", task.ID)
	t.logger.Info("task complited, user notified",
		"task_id", task.ID,
		"user_id", user.TelegramID,
	)
}
