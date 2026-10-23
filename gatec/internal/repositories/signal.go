package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SignalRepository struct{ db *gorm.DB }

func NewSignalRepository(db *gorm.DB) *SignalRepository { return &SignalRepository{db: db} }

func (r *SignalRepository) Save(ctx context.Context, signal domain.Signal) error {
	payload, err := json.Marshal(signal)
	if err != nil {
		return fmt.Errorf("encode signal: %w", err)
	}
	status := "CLOSED"
	if signal.Type == domain.SignalEntry {
		status = "OPEN"
	}
	row := models.SignalRecord{SignalID: signal.SignalID, StrategyID: signal.StrategyID, Market: signal.Market, Symbol: signal.Symbol, Timeframe: signal.Timeframe, Type: string(signal.Type), Mode: string(signal.Mode), Status: status, Published: false, Payload: payload, CreatedAt: signal.EmittedAt, UpdatedAt: time.Now().UTC()}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "signal_id"}}, DoNothing: true}).Create(&row).Error
}

func (r *SignalRepository) MarkPublished(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Model(&models.SignalRecord{}).Where("signal_id = ?", id).Updates(map[string]any{"published": true, "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("signal %q was not found", id)
	}
	return nil
}

func (r *SignalRepository) Get(ctx context.Context, id string) (models.SignalRecord, error) {
	var row models.SignalRecord
	err := r.db.WithContext(ctx).Where("signal_id = ?", id).First(&row).Error
	return row, err
}

func (r *SignalRepository) Pending(ctx context.Context, limit int) ([]domain.Signal, error) {
	if limit < 1 {
		limit = 100
	}
	rows := make([]models.SignalRecord, 0)
	if err := r.db.WithContext(ctx).Where("published = false").Order("created_at ASC, signal_id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Signal, 0, len(rows))
	for _, row := range rows {
		var signal domain.Signal
		if err := json.Unmarshal(row.Payload, &signal); err != nil {
			return nil, fmt.Errorf("decode pending signal %q: %w", row.SignalID, err)
		}
		out = append(out, signal)
	}
	return out, nil
}

func (r *SignalRepository) OpenPositions(ctx context.Context) ([]domain.Position, error) {
	rows := make([]models.SignalRecord, 0)
	if err := r.db.WithContext(ctx).Where("type = ? AND status = ?", string(domain.SignalEntry), "OPEN").Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Position, 0, len(rows))
	for _, row := range rows {
		var signal domain.Signal
		if err := json.Unmarshal(row.Payload, &signal); err != nil {
			return nil, fmt.Errorf("decode open entry %q: %w", row.SignalID, err)
		}
		if signal.StopLoss == nil {
			return nil, fmt.Errorf("open entry %q has no stop loss", row.SignalID)
		}
		position := domain.Position{EntrySignalID: signal.SignalID, StrategyID: signal.StrategyID, StrategyVersion: signal.StrategyVersion, StrategyMode: signal.Mode, StrategyConfig: append([]byte(nil), signal.StrategyConfig...), Market: signal.Market, Symbol: signal.Symbol, Timeframe: signal.Timeframe, Direction: signal.Direction, EntryTime: signal.CandleCloseTime, EntryPrice: signal.Price, StopLoss: *signal.StopLoss}
		if signal.TakeProfit != nil {
			position.TakeProfit = *signal.TakeProfit
		}
		out = append(out, position)
	}
	return out, nil
}

func (r *SignalRepository) CloseEntry(ctx context.Context, entrySignalID string, exit domain.Signal) error {
	if exit.Exit == nil {
		return fmt.Errorf("exit signal details are required")
	}
	pnl := exit.Exit.PnLPercent.String()
	result := r.db.WithContext(ctx).Model(&models.SignalRecord{}).Where("signal_id = ? AND type = ? AND status = ?", entrySignalID, string(domain.SignalEntry), "OPEN").
		Updates(map[string]any{"status": "CLOSED", "exit_signal_id": exit.SignalID, "pnl_percent": pnl, "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var row models.SignalRecord
		if err := r.db.WithContext(ctx).Select("status", "exit_signal_id").Where("signal_id = ?", entrySignalID).First(&row).Error; err != nil {
			return err
		}
		if row.Status == "CLOSED" && row.ExitSignalID != nil && *row.ExitSignalID == exit.SignalID {
			return nil
		}
		return fmt.Errorf("open entry signal %q was not found", entrySignalID)
	}
	return nil
}

func (r *SignalRepository) List(ctx context.Context, from, to time.Time) ([]domain.Signal, error) {
	rows := make([]models.SignalRecord, 0)
	query := r.db.WithContext(ctx).Order("created_at ASC, signal_id ASC")
	if !from.IsZero() {
		query = query.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("created_at < ?", to)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Signal, 0, len(rows))
	for _, row := range rows {
		var signal domain.Signal
		if err := json.Unmarshal(row.Payload, &signal); err != nil {
			return nil, err
		}
		out = append(out, signal)
	}
	return out, nil
}

var _ domain.SignalStore = (*SignalRepository)(nil)
