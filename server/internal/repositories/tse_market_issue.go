package repositories

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TSEIssueFilter selects TSE catalog issues in the panel; empty fields match
// everything. InsCode and MarketID match issues about either of them.
type TSEIssueFilter struct {
	Query    string
	Type     consts.TSEIssueType
	Severity consts.TSEIssueSeverity
	Status   consts.TSEIssueStatus
	InsCode  string
	MarketID uint
}

func (f TSEIssueFilter) apply(tx *gorm.DB) *gorm.DB {
	if query := strings.TrimSpace(f.Query); query != "" {
		like := "%" + query + "%"
		tx = tx.Where("symbol ILIKE ? OR name ILIKE ? OR detail ILIKE ? OR ins_codes::text LIKE ?", like, like, like, like)
	}
	if f.Type != "" {
		tx = tx.Where("type = ?", f.Type)
	}
	if f.Severity != "" {
		tx = tx.Where("severity = ?", f.Severity)
	}
	if f.Status != "" {
		tx = tx.Where("status = ?", f.Status)
	}
	switch insCodes, marketIDs := jsonArray(f.InsCode), jsonArray(f.MarketID); {
	case f.InsCode != "" && f.MarketID != 0:
		tx = tx.Where("ins_codes @> ?::jsonb OR market_ids @> ?::jsonb", insCodes, marketIDs)
	case f.InsCode != "":
		tx = tx.Where("ins_codes @> ?::jsonb", insCodes)
	case f.MarketID != 0:
		tx = tx.Where("market_ids @> ?::jsonb", marketIDs)
	}
	return tx
}

func jsonArray(value any) string {
	encoded, _ := json.Marshal([]any{value})
	return string(encoded)
}

// SaveIssues stores detected issues by key. A resolved issue found again is
// reopened; open and ignored issues keep their status.
func (r *TSEMarketRepository) SaveIssues(ctx context.Context, issues []*models.TSEMarketIssue) error {
	if len(issues) == 0 {
		return nil
	}
	set := clause.AssignmentColumns([]string{
		"updated_at", "type", "severity", "symbol", "name", "detail", "ins_codes", "market_ids", "detected_at",
	})
	set = append(set,
		clause.Assignment{Column: clause.Column{Name: "status"}, Value: gorm.Expr(
			"CASE WHEN tse_market_issues.status = ? THEN ? ELSE tse_market_issues.status END",
			consts.TSEIssueStatusResolved, consts.TSEIssueStatusOpen)},
		clause.Assignment{Column: clause.Column{Name: "resolved_at"}, Value: gorm.Expr(
			"CASE WHEN tse_market_issues.status = ? THEN NULL ELSE tse_market_issues.resolved_at END",
			consts.TSEIssueStatusResolved)},
	)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoUpdates: set}).
		CreateInBatches(issues, 200).Error
}

// ResolveStaleIssues resolves open issues of the given types that were last
// detected before the given time.
func (r *TSEMarketRepository) ResolveStaleIssues(ctx context.Context, types []consts.TSEIssueType, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.TSEMarketIssue{}).
		Where("status = ? AND type IN ? AND detected_at < ?", consts.TSEIssueStatusOpen, types, before).
		UpdateColumns(map[string]any{"status": consts.TSEIssueStatusResolved, "resolved_at": time.Now(), "updated_at": time.Now()})
	return result.RowsAffected, result.Error
}

// ResolveIssuesOf resolves the open issues that are only about the given insCodes.
func (r *TSEMarketRepository) ResolveIssuesOf(ctx context.Context, insCodes []string) (int64, error) {
	if len(insCodes) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Model(&models.TSEMarketIssue{}).
		Where("status = ? AND jsonb_array_length(ins_codes) > 0", consts.TSEIssueStatusOpen).
		Where("NOT EXISTS (SELECT 1 FROM jsonb_array_elements_text(ins_codes) e(ins_code) WHERE e.ins_code NOT IN ?)", insCodes).
		UpdateColumns(map[string]any{"status": consts.TSEIssueStatusResolved, "resolved_at": time.Now(), "updated_at": time.Now()})
	return result.RowsAffected, result.Error
}

func (r *TSEMarketRepository) ListIssues(ctx context.Context, filter TSEIssueFilter) ([]*models.TSEMarketIssue, error) {
	issues := make([]*models.TSEMarketIssue, 0)
	return issues, filter.apply(r.db.WithContext(ctx).Model(&models.TSEMarketIssue{})).Find(&issues).Error
}

// PageIssues lists issues, most severe and newest first.
func (r *TSEMarketRepository) PageIssues(ctx context.Context, filter TSEIssueFilter, page paginator.Params) ([]*models.TSEMarketIssue, int64, error) {
	var total int64
	if err := filter.apply(r.db.WithContext(ctx).Model(&models.TSEMarketIssue{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	issues := make([]*models.TSEMarketIssue, 0)
	err := page.Apply(filter.apply(r.db.WithContext(ctx).Model(&models.TSEMarketIssue{}))).
		Order(clause.Expr{SQL: "CASE severity WHEN ? THEN 0 WHEN ? THEN 1 ELSE 2 END",
			Vars: []any{consts.TSEIssueSeverityError, consts.TSEIssueSeverityWarning}}).
		Order("type ASC").Order("symbol ASC").Order("id ASC").Find(&issues).Error
	return issues, total, err
}

// SetIssueStatus changes the status of the given issues, or of every issue that
// matches filter when ids is empty. It returns the number of changed issues.
func (r *TSEMarketRepository) SetIssueStatus(ctx context.Context, ids []uint, filter *TSEIssueFilter, columns map[string]any) (int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.TSEMarketIssue{})
	if len(ids) > 0 {
		tx = tx.Where("id IN ?", ids)
	} else if filter != nil {
		// Issues that are no longer found keep their status.
		tx = filter.apply(tx).Where("status <> ?", consts.TSEIssueStatusResolved)
	} else {
		return 0, nil
	}
	result := tx.UpdateColumns(columns)
	return result.RowsAffected, result.Error
}

type TSEIssueCount struct {
	Type     consts.TSEIssueType
	Severity consts.TSEIssueSeverity
	Status   consts.TSEIssueStatus
	Count    int64
}

func (r *TSEMarketRepository) IssueCounts(ctx context.Context) ([]TSEIssueCount, error) {
	counts := make([]TSEIssueCount, 0)
	err := r.db.WithContext(ctx).Model(&models.TSEMarketIssue{}).
		Select("type, severity, status, COUNT(*) AS count").
		Group("type, severity, status").Scan(&counts).Error
	return counts, err
}

// TSEPriceOverlap is a pair or more of insCodes of one market with prices on the same days.
type TSEPriceOverlap struct {
	MarketID uint
	// InsCodes are comma separated in ascending order.
	InsCodes string
	Days     int64
}

// PriceOverlaps returns the insCodes of each market whose daily prices share candles.
func (r *TSEMarketRepository) PriceOverlaps(ctx context.Context) ([]TSEPriceOverlap, error) {
	overlaps := make([]TSEPriceOverlap, 0)
	err := r.db.WithContext(ctx).Raw(`
		SELECT market_id, ins_codes, COUNT(*) AS days FROM (
			SELECT p.market_id, string_agg(DISTINCT t.ins_code, ',' ORDER BY t.ins_code) AS ins_codes
			FROM tse_daily_closing_prices p
			JOIN tse_markets t ON t.id = p.tse_market_id
			GROUP BY p.market_id, p.candle_id
			HAVING COUNT(DISTINCT p.tse_market_id) > 1
		) shared
		GROUP BY market_id, ins_codes
		ORDER BY market_id`).Scan(&overlaps).Error
	return overlaps, err
}
