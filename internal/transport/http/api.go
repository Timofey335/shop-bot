package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"shop-bot/internal/domain"
)

// Получить список доступных магазинов
func (c *Client) GetShops(ctx context.Context) ([]domain.Shop, error) {
	// fmt.Println("http.getshops")
	body, err := c.doRequest(ctx, "GET", "/shops", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops: %w", err)
	}

	var resp shopsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode shops: %w", err)
	}

	shops := make([]domain.Shop, 0, len(resp))
	for name, id := range resp {
		shops = append(shops, domain.Shop{
			ID:   id,
			Name: name,
		})
	}

	return shops, nil
}

// Получить все продукты магазина
func (c *Client) GetProducts(ctx context.Context, shopID string) ([]domain.Product, error) {
	params := url.Values{}
	params.Set("shop_id", shopID)

	body, err := c.doRequest(ctx, "GET", "/products", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	var resp productResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode products: %w", err)
	}

	products := make([]domain.Product, len(resp.Products))
	for i, p := range resp.Products {
		products[i] = domain.Product{
			Name:         p.Name,
			URL:          p.URL,
			Availability: p.Availability,
		}
	}

	return products, nil
}

// Поиск товаров по запросу
func (c *Client) SearchProducts(ctx context.Context, shopID, query string) ([]domain.Product, error) {
	params := url.Values{}
	params.Set("shop_id", shopID)
	params.Set("q", query)

	body, err := c.doRequest(ctx, "GET", "/search", params)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	products := make([]domain.Product, len(resp.Products))
	for i, p := range resp.Products {
		products[i] = domain.Product{
			Name:         p.Name,
			URL:          p.URL,
			Availability: p.Availability,
		}
	}

	return products, nil
}
