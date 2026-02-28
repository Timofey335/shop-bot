package service

import (
	"context"
	"fmt"
	"log"
	"shop-bot/internal/domain"
)

type TrackingService struct {
	repo        domain.TrackingRepository
	shopService domain.ShopService
}

func NewTrackingService(repo domain.TrackingRepository, shop domain.ShopService) *TrackingService {
	return &TrackingService{
		repo:        repo,
		shopService: shop,
	}
}

// CreateTask создает задачу отслеживания
func (s *TrackingService) CreateTask(ctx context.Context, userID int64, shopID, query string) (*domain.TrackingTask, error) {
	log.Printf("Creating task: userID=%d, shopID=%s", userID, shopID)
	products, err := s.shopService.SearchProducts(ctx, shopID, query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	task := &domain.TrackingTask{
		UserID: userID,
		ShopID: shopID,
		Query:  query,
		Status: domain.StatusActive,
	}

	if len(products) > 0 {
		task.TargetName = products[0].Name
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetActiveTasks получает все активные задачи
func (s *TrackingService) GetActiveTasks(ctx context.Context) ([]domain.TrackingTask, error) {
	return s.repo.GetAllActive(ctx)
}

// CheckTask проверяет, появился ли товар
func (s *TrackingService) CheckTask(ctx context.Context, task *domain.TrackingTask) (bool, error) {
	products, err := s.shopService.SearchProducts(ctx, task.ShopID, task.Query)
	if err != nil {
		return false, err
	}

	if len(products) == 0 {
		return false, nil
	}

	// Обновляем имя, если изменилось
	if task.TargetName != products[0].Name {
		task.TargetName = products[0].Name
	}

	return products[0].Availability > 0, nil
}

func (s *TrackingService) MarkDone(ctx context.Context, taskID int64) error {
	return s.repo.UpdateStatus(ctx, taskID, domain.StatusDone)
}
