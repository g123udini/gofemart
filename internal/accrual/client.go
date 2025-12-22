package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Status string

const (
	StatusRegistered Status = "REGISTERED"
	StatusInvalid    Status = "INVALID"
	StatusProcessing Status = "PROCESSING"
	StatusProcessed  Status = "PROCESSED"
)

type OrderInfo struct {
	Order   string   `json:"order"`
	Status  Status   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"` // важно: может отсутствовать
}

var ErrNotRegistered = errors.New("order not registered (204)")

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("accrual rate limited, retry after %s", e.RetryAfter)
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Second}
	}
	return &Client{baseURL: baseURL, http: httpClient}
}

func (c *Client) GetOrder(ctx context.Context, number string) (OrderInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/orders/"+number, nil)
	if err != nil {
		return OrderInfo{}, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return OrderInfo{}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK: // 200
		var oi OrderInfo
		if err := json.NewDecoder(resp.Body).Decode(&oi); err != nil {
			return OrderInfo{}, err
		}
		return oi, nil

	case http.StatusNoContent: // 204
		return OrderInfo{}, ErrNotRegistered

	case http.StatusTooManyRequests: // 429
		return OrderInfo{}, RateLimitError{RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}

	default:
		return OrderInfo{}, fmt.Errorf("accrual unexpected status code=%d", resp.StatusCode)
	}
}

func parseRetryAfter(v string) time.Duration {
	sec, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || sec <= 0 {
		return 60 * time.Second
	}
	return time.Duration(sec) * time.Second
}
