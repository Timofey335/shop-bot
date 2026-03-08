package http

type shopsResponse map[string]string

type productItem struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Availability int    `json:"availability"`
}

type productResponse struct {
	ShopID   int           `json:"shop_id"`
	Count    int           `json:"count"`
	Products []productItem `json:"products"`
}

type searchResponse productResponse
