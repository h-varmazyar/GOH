package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/clickhouse"
)

type TradeHistoryRepository struct {
	client *clickhouse.Client
}

func NewTradeHistoryRepository(client *clickhouse.Client) (*TradeHistoryRepository, error) {
	if client == nil {
		return nil, fmt.Errorf("ClickHouse client is required")
	}
	return &TradeHistoryRepository{client: client}, nil
}

func (r *TradeHistoryRepository) EnsureSchema(ctx context.Context) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		market_id UInt64,
		ins_code String,
		trade_date Date,
		trade_index UInt32,
		canceled Bool,
		trade_time Int64,
		price Int64,
		quantity Int64,
		version UInt64
	) ENGINE = ReplacingMergeTree(version)
	PARTITION BY toYYYYMM(trade_date)
	ORDER BY (ins_code, trade_date, trade_index)`, models.TSETradeHistory{}.TableName())
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("create ClickHouse trade history table: %w", err)
	}
	query = fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS market_id UInt64 AFTER ins_code", models.TSETradeHistory{}.TableName())
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("add market_id to ClickHouse trade history table: %w", err)
	}
	return nil
}

func (r *TradeHistoryRepository) Insert(ctx context.Context, rows []models.TSETradeHistory) error {
	if len(rows) == 0 {
		return nil
	}
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("encode TSE trade history row: %w", err)
		}
	}
	query := fmt.Sprintf("INSERT INTO %s FORMAT JSONEachRow\n%s", models.TSETradeHistory{}.TableName(), body.String())
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("insert TSE trade history rows: %w", err)
	}
	return nil
}
