package domain

import "context"

// Продукт из магазина
// Получаем из python api
type Product struct {
	Name         string
	URL          string
	Availability int
}

// Магазин на сайте
type Shop struct {
	ID   string
	Name string
}

type ShopService interface {
	GetShops(ctx context.Context) ([]Shop, error)
	GetProducts(ctx context.Context, shopID string) ([]Product, error)
	SearchProducts(ctx context.Context, shopID string, query string) ([]Product, error)
	GetShopName(shopID string) (string, error)
}
