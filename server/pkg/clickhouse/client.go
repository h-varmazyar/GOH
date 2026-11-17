package clickhouse

import (
	"bufio"
	"context"
	"encoding/json"
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
	response, err := c.do(ctx, query, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("read ClickHouse response: %w", err)
	}
	return nil
}

// Summary is the X-ClickHouse-Summary of a query.
type Summary struct {
	ReadRows    int64 `json:"read_rows,string"`
	WrittenRows int64 `json:"written_rows,string"`
}

// ExecSummary runs a statement, such as INSERT ... SELECT, to its end and
// returns how many rows it read and wrote.
func (c *Client) ExecSummary(ctx context.Context, query string) (Summary, error) {
	// The summary header is complete only when the response waits for the end
	// of the query.
	response, err := c.do(ctx, query, url.Values{"wait_end_of_query": {"1"}})
	if err != nil {
		return Summary{}, err
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return Summary{}, fmt.Errorf("read ClickHouse response: %w", err)
	}
	var summary Summary
	if header := response.Header.Get("X-ClickHouse-Summary"); header != "" {
		if err := json.Unmarshal([]byte(header), &summary); err != nil {
			return Summary{}, fmt.Errorf("decode ClickHouse summary %q: %w", header, err)
		}
	}
	return summary, nil
}

// Query runs a SELECT and calls scan with every row. The query must end with
// FORMAT JSONEachRow; scan decodes one JSON object per row.
func (c *Client) Query(ctx context.Context, query string, scan func(row []byte) error) error {
	response, err := c.do(ctx, query, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		row := scanner.Bytes()
		if len(row) == 0 {
			continue
		}
		if err := scan(row); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read ClickHouse response: %w", err)
	}
	return nil
}

// do sends query and returns a successful response.
func (c *Client) do(ctx context.Context, query string, settings url.Values) (*http.Response, error) {
	endpoint := *c.endpoint
	parameters := endpoint.Query()
	parameters.Set("database", c.database)
	for key, values := range settings {
		parameters[key] = values
	}
	endpoint.RawQuery = parameters.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("create ClickHouse request: %w", err)
	}
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if c.username != "" {
		request.SetBasicAuth(c.username, c.password)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send ClickHouse request: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		defer response.Body.Close()
		message, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			return nil, fmt.Errorf("ClickHouse returned HTTP %d (read response: %w)", response.StatusCode, readErr)
		}
		return nil, fmt.Errorf("ClickHouse returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	return response, nil
}

// Quote returns value as a ClickHouse string literal.
func Quote(value string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value) + "'"
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
