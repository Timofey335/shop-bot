package http

import (
	"context"
	"encoding/json"
	"fmt"

	"shop-bot/internal/domain"
)

// Получить список доступных магазинов
func (c *Client) GetShops(ctx context.Context) ([]domain.Shop, error) {
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
