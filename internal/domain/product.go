package domain

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
	GetShops() ([]Shop, error)
	GetProducts(shopID string) ([]Product, error)
	SearchProducts(shopID string, query string) ([]Product, error)
}
