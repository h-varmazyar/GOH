package clickhouse

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/configs"
)

type Client struct {
	endpoint *url.URL
	database string
	username string
	password string
	http     *http.Client
}

func New(cfg configs.ClickHouse) (*Client, error) {
	endpoint, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse ClickHouse URL: %w", err)
	}
	if (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" {
		return nil, fmt.Errorf("invalid ClickHouse URL %q", cfg.URL)
	}
	if strings.TrimSpace(cfg.Database) == "" {
		return nil, fmt.Errorf("ClickHouse database is required")
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/")

	return &Client{
		endpoint: endpoint,
		database: cfg.Database,
		username: cfg.Username,
		password: cfg.Password,
		http:     &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) Exec(ctx context.Context, query string) error {
	endpoint := *c.endpoint
	parameters := endpoint.Query()
	parameters.Set("database", c.database)
	endpoint.RawQuery = parameters.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(query))
	if err != nil {
		return fmt.Errorf("create ClickHouse request: %w", err)
	}
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if c.username != "" {
		request.SetBasicAuth(c.username, c.password)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("send ClickHouse request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("ClickHouse returned HTTP %d (read response: %w)", response.StatusCode, readErr)
		}
		return fmt.Errorf("ClickHouse returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("read ClickHouse response: %w", err)
	}
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	if err := c.Exec(ctx, "SELECT 1"); err != nil {
		return fmt.Errorf("ping ClickHouse: %w", err)
	}
	return nil
}

func (c *Client) Close() {
	c.http.CloseIdleConnections()
}
