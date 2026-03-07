package service

import (
	"context"
	"fmt"
	"shop-bot/internal/domain"
	"shop-bot/internal/transport/http"
)

// shop service реализует domain.ShopService
type ShopService struct {
	client *http.Client
}

func NewShopService(apiURL string) *ShopService {
	return &ShopService{
		client: http.NewClient(apiURL, 0),
	}
}

func (s *ShopService) GetShops(ctx context.Context) ([]domain.Shop, error) {
	// fmt.Println("shop.getshops")
	shops, err := s.client.GetShops(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch shops: %w", err)
	}
	return shops, nil
}

func (s *ShopService) GetProducts(ctx context.Context, shopID string) ([]domain.Product, error) {
	if shopID == "" {
		return nil, fmt.Errorf("shop_id is required")
	}

	products, err := s.client.GetProducts(ctx, shopID)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch products: %w", err)
	}

	return products, nil
}

func (s *ShopService) SearchProducts(ctx context.Context, shopID, query string) ([]domain.Product, error) {
	if shopID == "" {
		return nil, fmt.Errorf("shop_id is required")
	}

	if len(query) < 2 {
		return nil, fmt.Errorf("query too short (min 2 characters)")
	}

	products, err := s.client.SearchProducts(ctx, shopID, query)
	if err != nil {
		return nil, fmt.Errorf("cannot search products: %w", err)
	}

	return products, nil
}

var shopNames = map[string]string{
	"218999": "СИЗО 1",
	"219013": "СИЗО 3",
	"221918": "СИЗО 4",
	"221917": "ЛИУ 15",
}

func (s *ShopService) GetShopName(shopID string) (string, error) {
	if name, ok := shopNames[shopID]; ok {
		return name, nil
	}
	return shopID, nil // Fallback на ID
}

func (s *ShopService) _GetShopName(ctx context.Context, shopID string) (string, error) {
	shops, err := s.GetShops(ctx)
	if err != nil {
		return "", err
	}

	for _, shop := range shops {
		if shop.ID == shopID {
			return shop.Name, nil
		}
	}

	return "", fmt.Errorf("shop not found: %s", shopID)
}
