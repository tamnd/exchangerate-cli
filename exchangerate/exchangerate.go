// Package exchangerate is the library behind the exchangerate command line:
// the HTTP client, request shaping, and the typed data models for the
// ExchangeRate-API open tier at open.er-api.com.
//
// No API key required. The client sets a real User-Agent, paces requests, and
// retries transient failures (429 and 5xx) with exponential backoff.
package exchangerate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "open.er-api.com"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://open.er-api.com/v6",
		UserAgent: "Mozilla/5.0 (compatible; exchangerate-cli/dev; +https://github.com/tamnd/exchangerate-cli)",
		Rate:      500 * time.Millisecond,
		Timeout:   15 * time.Second,
		Retries:   3,
	}
}

// Client talks to open.er-api.com over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Latest fetches the current exchange rates for the given base currency.
// base defaults to "USD" if empty. Rates are sorted alphabetically by
// currency code; if limit > 0 only the first limit entries are returned.
func (c *Client) Latest(ctx context.Context, base string, limit int) ([]Rate, error) {
	resp, err := c.latestRaw(ctx, base)
	if err != nil {
		return nil, err
	}

	rates := make([]Rate, 0, len(resp.Rates))
	for code, rate := range resp.Rates {
		rates = append(rates, Rate{Currency: code, Rate: rate, Base: resp.BaseCode})
	}
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Currency < rates[j].Currency
	})
	for i := range rates {
		rates[i].Rank = i + 1
	}
	if limit > 0 && limit < len(rates) {
		rates = rates[:limit]
	}
	return rates, nil
}

// Convert fetches the rate from base to target and multiplies by amount.
func (c *Client) Convert(ctx context.Context, from, to string, amount float64) (*Conversion, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)

	resp, err := c.latestRaw(ctx, from)
	if err != nil {
		return nil, err
	}

	rate, ok := resp.Rates[to]
	if !ok {
		return nil, fmt.Errorf("currency %q not found in rates for %s", to, from)
	}

	return &Conversion{
		From:      from,
		To:        to,
		Amount:    amount,
		Result:    amount * rate,
		Rate:      rate,
		UpdatedAt: resp.UpdatedAt,
	}, nil
}

// latestRaw fetches and decodes the raw /latest/{base} response.
func (c *Client) latestRaw(ctx context.Context, base string) (*latestResponse, error) {
	if base == "" {
		base = "USD"
	}
	base = strings.ToUpper(base)
	u := fmt.Sprintf("%s/latest/%s", c.cfg.BaseURL, base)

	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}

	var resp latestResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode latest: %w", err)
	}
	if resp.Result != "success" {
		return nil, fmt.Errorf("api error: %s", resp.Result)
	}
	return &resp, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
