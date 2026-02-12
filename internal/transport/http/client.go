package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
		maxRetries: 3,
	}
}

func (c *Client) SetMaxRetries(n int) {
	c.maxRetries = n
}

func (c *Client) doRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {

	fullURL := c.baseURL + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-time.After(backoff):
				//Продолжаем
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "ShopBot/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		fmt.Println("doRequest")
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("server error: %d, body: %s", resp.StatusCode, string(body))
				continue
			}
			return nil, fmt.Errorf("HTTP error: %d: %s", resp.StatusCode, string(body))
		}
		return body, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
