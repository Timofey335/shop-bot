package worker

import (
	"context"
	"fmt"
	"log"
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
	interval    time.Duration
}

func NewTracker(svc domain.TrackingService, users domain.UserRepository, shopService domain.ShopService, notifier Notifier, interval time.Duration) *Tracker {
	return &Tracker{
		trackingSvc: svc,
		userRepo:    users,
		shopService: shopService,
		notifier:    notifier,
		interval:    interval,
	}
}

func (t *Tracker) Start(ctx context.Context) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	log.Println("Tracking worker started")

	t.checkAll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Trackinf worker stopped")
			return
		case <-ticker.C:
			t.checkAll(ctx)
		}
	}
}

func (t *Tracker) checkAll(ctx context.Context) {
	log.Println("Checking tracking tasks...")

	tasks, err := t.trackingSvc.GetActiveTasks(ctx)
	if err != nil {
		log.Printf("Error getting tasks: %v", err)
		return
	}

	log.Printf("Found %d active tasks", len(tasks))

	for _, task := range tasks {
		taskCtx, cancel := context.WithTimeout(ctx, 100*time.Second)

		found, err := t.trackingSvc.CheckTask(taskCtx, &task)
		cancel()

		if err != nil {
			log.Printf("Error checking task %d: %v", task.ID, err)
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
		log.Printf("Error getting user %d: %v", task.UserID, err)
		return
	}

	shopName, err := t.shopService.GetShopName(ctx, task.ShopID)
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
		log.Printf("Error sending notification: %v", err)
		return
	}

	if err := t.trackingSvc.MarkDone(ctx, task.ID); err != nil {
		log.Printf("Error marking task done: %v", err)
		return
	}

	log.Printf("Task %d completed, user notified", task.ID)
}
