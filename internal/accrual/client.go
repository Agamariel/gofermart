package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrNotFound    = errors.New("accrual not found")
	ErrRateLimited = errors.New("accrual rate limited")
)

// RateLimitError содержит паузу, которую рекомендует сервис.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after %s", e.RetryAfter)
}

// AccrualResponse описывает ответ сервиса начислений.
type AccrualResponse struct {
	Order   string          `json:"order"`
	Status  string          `json:"status"`
	Accrual decimal.Decimal `json:"accrual,omitempty"`
}

// AccrualClient интерфейс получения начислений.
type AccrualClient interface {
	GetOrderAccrual(ctx context.Context, orderNumber string) (*AccrualResponse, error)
}

type HTTPAccrualClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPAccrualClient создаёт HTTP-клиент.
func NewHTTPAccrualClient(baseURL string, timeout time.Duration) *HTTPAccrualClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &HTTPAccrualClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetOrderAccrual получает данные по заказу.
func (c *HTTPAccrualClient) GetOrderAccrual(ctx context.Context, orderNumber string) (*AccrualResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid accrual base url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/api/orders/%s", u.Path, orderNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf("decode accrual response: %w", err)
		}
		return &payload, nil
	case http.StatusNoContent:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, RateLimitError{RetryAfter: retryAfter}
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("accrual service error 500")
	default:
		return nil, fmt.Errorf("unexpected accrual status: %d", resp.StatusCode)
	}
}

func parseRetryAfter(val string) time.Duration {
	if val == "" {
		return 5 * time.Second
	}
	// support seconds value
	if secs, err := strconv.Atoi(val); err == nil {
		return time.Duration(secs) * time.Second
	}
	// try http-date
	if t, err := http.ParseTime(val); err == nil {
		return time.Until(t)
	}
	return 5 * time.Second
}
