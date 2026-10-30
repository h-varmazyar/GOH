package repositories

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/models"
	clickhouseClient "github.com/h-varmazyar/gatec/pkg/clickhouse"
)

func TestTradeHistoryRepositoryEnsureSchemaAndInsert(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("database") != "gatec" {
			t.Errorf("unexpected ClickHouse database: %q", r.URL.Query().Get("database"))
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "user" || password != "password" {
			t.Errorf("unexpected ClickHouse credentials: %q, %q, authenticated=%t", username, password, ok)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read ClickHouse request: %v", err)
			return
		}
		requests = append(requests, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := clickhouseClient.New(configs.ClickHouse{
		URL:      server.URL,
		Database: "gatec",
		Username: "user",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("create ClickHouse client: %v", err)
	}
	defer client.Close()
	repo, err := NewTradeHistoryRepository(client)
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if err := repo.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	err = repo.Insert(context.Background(), []models.TSETradeHistory{{
		MarketID:   7,
		InsCode:    "123",
		TradeDate:  "2026-10-30",
		TradeIndex: 2,
		Canceled:   true,
		TradeTime:  101530,
		Price:      2500,
		Quantity:   10,
		Version:    42,
	}})
	if err != nil {
		t.Fatalf("insert history: %v", err)
	}
	if len(requests) != 3 {
		t.Fatalf("got %d requests, want schema creation, migration, and insert", len(requests))
	}
	if !strings.Contains(requests[0], "ReplacingMergeTree(version)") {
		t.Errorf("schema query is missing replacement engine: %s", requests[0])
	}
	if !strings.Contains(requests[0], "market_id UInt64") {
		t.Errorf("schema query is missing market_id: %s", requests[0])
	}
	if !strings.Contains(requests[0], "ORDER BY (ins_code, trade_date, trade_index)") {
		t.Errorf("schema query is missing composite sorting/primary key: %s", requests[0])
	}
	if !strings.Contains(requests[1], "ADD COLUMN IF NOT EXISTS market_id UInt64") {
		t.Errorf("schema migration is missing market_id column: %s", requests[1])
	}
	for _, expected := range []string{
		"INSERT INTO tse_trade_history FORMAT JSONEachRow",
		`"ins_code":"123"`,
		`"market_id":7`,
		`"trade_date":"2026-10-30"`,
		`"trade_index":2`,
		`"canceled":true`,
	} {
		if !strings.Contains(requests[2], expected) {
			t.Errorf("insert query is missing %q: %s", expected, requests[2])
		}
	}
}

func TestNewTradeHistoryRepositoryNeedsClient(t *testing.T) {
	_, err := NewTradeHistoryRepository(nil)
	if err == nil {
		t.Fatal("expected missing ClickHouse client to return an error")
	}
}
