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
