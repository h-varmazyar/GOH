package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
)

type TSEMarketRepository struct{ db *gorm.DB }

func NewTSEMarketRepository(db *gorm.DB) *TSEMarketRepository {
	return &TSEMarketRepository{db: db}
}

// TSEMarketFilter selects TSE markets in the panel; empty fields match everything.
type TSEMarketFilter struct {
	Query                 string
	MarketID              uint
	MarketType            consts.TSEMarketType
	Kind                  consts.TSEInstrumentKind
	IndustryGroupCode     string
	Delisted              *bool
	DataCollectionEnabled *bool
	EvaluationEnabled     *bool
}

func (f TSEMarketFilter) apply(tx *gorm.DB) *gorm.DB {
	if query := strings.TrimSpace(f.Query); query != "" {
		like := "%" + query + "%"
		tx = tx.Where("label ILIKE ? OR name ILIKE ? OR ins_code = ? OR instrument_id ILIKE ?", like, like, query, like)
	}
	if f.MarketID != 0 {
		tx = tx.Where("market_id = ?", f.MarketID)
	}
	if f.MarketType != "" {
		tx = tx.Where("market_type = ?", f.MarketType)
	}
	if f.Kind != "" {
		tx = tx.Where("kind = ?", f.Kind)
	}
	if f.IndustryGroupCode != "" {
		tx = tx.Where("industry_group_code = ?", f.IndustryGroupCode)
	}
	if f.Delisted != nil {
		tx = tx.Where("delisted = ?", *f.Delisted)
	}
	if f.DataCollectionEnabled != nil {
		tx = tx.Where("data_collection_enabled = ?", *f.DataCollectionEnabled)
	}
	if f.EvaluationEnabled != nil {
		tx = tx.Where("evaluation_enabled = ?", *f.EvaluationEnabled)
	}
	return tx
}

type TSEIndustryGroup struct {
	Code string
	Name string
}

func (r *TSEMarketRepository) List(ctx context.Context, conditions map[string]any) ([]*models.TSEMarket, error) {
	var markets []*models.TSEMarket
	tx := r.db.WithContext(ctx)
	for key, value := range conditions {
		tx = tx.Where(key, value)
	}

	err := tx.Preload("Market").Order("market_id ASC").Find(&markets).Error
	return markets, err
}

// ListDataCollection returns the markets whose data collection is enabled in the panel.
func (r *TSEMarketRepository) ListDataCollection(ctx context.Context) ([]*models.TSEMarket, error) {
	return r.List(ctx, map[string]any{"data_collection_enabled = ?": true})
}

// ListEvaluation returns the markets whose evaluation is enabled in the panel.
func (r *TSEMarketRepository) ListEvaluation(ctx context.Context) ([]*models.TSEMarket, error) {
	return r.List(ctx, map[string]any{"evaluation_enabled = ?": true})
}

func (r *TSEMarketRepository) Page(ctx context.Context, filter TSEMarketFilter, page paginator.Params) ([]*models.TSEMarket, int64, error) {
	var total int64
	if err := filter.apply(r.db.WithContext(ctx).Model(&models.TSEMarket{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	markets := make([]*models.TSEMarket, 0)
	err := page.Apply(filter.apply(r.db.WithContext(ctx).Model(&models.TSEMarket{}))).
		Order("delisted ASC").Order("label ASC").Order("id ASC").Find(&markets).Error
	return markets, total, err
}

func (r *TSEMarketRepository) IndustryGroups(ctx context.Context) ([]TSEIndustryGroup, error) {
	groups := make([]TSEIndustryGroup, 0)
	err := r.db.WithContext(ctx).Model(&models.TSEMarket{}).
		Select("industry_group_code AS code, MAX(industry_group_name) AS name").
		Where("industry_group_code <> ''").
		Group("industry_group_code").Order("name ASC").Scan(&groups).Error
	return groups, err
}

func (r *TSEMarketRepository) GetByID(ctx context.Context, id uint) (*models.TSEMarket, error) {
	market := new(models.TSEMarket)
	return market, r.db.WithContext(ctx).Preload("Market").First(market, id).Error
}

func (r *TSEMarketRepository) GetByInsCode(ctx context.Context, insCode string) (*models.TSEMarket, error) {
	market := new(models.TSEMarket)
	return market, r.db.WithContext(ctx).Preload("Market").Where("ins_code = ?", insCode).First(market).Error
}

// SetFlags updates panel flags of the given markets, or of every market matching
// filter when ids is empty. It returns the number of updated markets.
func (r *TSEMarketRepository) SetFlags(ctx context.Context, ids []uint, filter *TSEMarketFilter, columns map[string]any) (int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.TSEMarket{})
	if len(ids) > 0 {
		tx = tx.Where("id IN ?", ids)
	} else if filter != nil {
		tx = filter.apply(tx)
	} else {
		return 0, nil
	}
	result := tx.UpdateColumns(columns)
	return result.RowsAffected, result.Error
}

func (r *TSEMarketRepository) Create(ctx context.Context, market *models.TSEMarket) error {
	return r.db.WithContext(ctx).Omit("Market").Create(market).Error
}

func (r *TSEMarketRepository) CreateWithMarket(ctx context.Context, market *models.Market, tseMarket *models.TSEMarket) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(market).Error; err != nil {
			return err
		}
		tseMarket.MarketID = market.ID
		return tx.Omit("Market").Create(tseMarket).Error
	})
}

func (r *TSEMarketRepository) Update(ctx context.Context, id uint, columns map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.TSEMarket{}).Where("id = ?", id).UpdateColumns(columns).Error
}

// MemberCounts returns the number of TSE instruments linked to each of the markets.
func (r *TSEMarketRepository) MemberCounts(ctx context.Context, marketIDs []uint) (map[uint]int64, error) {
	rows := make([]struct {
		MarketID uint
		Count    int64
	}, 0)
	err := r.db.WithContext(ctx).Model(&models.TSEMarket{}).
		Select("market_id, COUNT(*) AS count").Where("market_id IN ?", marketIDs).
		Group("market_id").Scan(&rows).Error
	result := make(map[uint]int64, len(rows))
	for _, row := range rows {
		result[row.MarketID] = row.Count
	}
	return result, err
}

// HasPrices reports whether daily prices of the TSE instrument are stored.
func (r *TSEMarketRepository) HasPrices(ctx context.Context, id uint) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Raw("SELECT EXISTS (SELECT 1 FROM tse_daily_closing_prices WHERE tse_market_id = ?)", id).
		Scan(&exists).Error
	return exists, err
}

// Link links a TSE instrument to market, creating market first when it has no ID.
// The previous market of the instrument is deleted when no instrument is left in
// it, so that a symbol keeps a single market. It reports whether it was deleted.
func (r *TSEMarketRepository) Link(ctx context.Context, id uint, market *models.Market) (bool, error) {
	deleted := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var previous uint
		if err := tx.Model(&models.TSEMarket{}).Where("id = ?", id).Pluck("market_id", &previous).Error; err != nil {
			return err
		}
		if market.ID == 0 {
			if err := tx.Create(market).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.TSEMarket{}).Where("id = ?", id).UpdateColumn("market_id", market.ID).Error; err != nil {
			return err
		}
		if previous == 0 || previous == market.ID {
			return nil
		}
		emptied, err := deleteEmptyMarkets(tx, []uint{previous})
		deleted = len(emptied) > 0
		return err
	})
	return deleted, err
}

// TSEDeleteResult lists what Delete removed.
type TSEDeleteResult struct {
	InsCodes []string
	// MarketIDs are the markets deleted because no instrument was left in them.
	MarketIDs []uint
}

// Delete deletes TSE instruments with their daily prices and the candles built
// only from those prices. Markets left without instruments are deleted with all
// of their data.
func (r *TSEMarketRepository) Delete(ctx context.Context, ids []uint) (TSEDeleteResult, error) {
	var result TSEDeleteResult
	if len(ids) == 0 {
		return result, nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var marketIDs []uint
		if err := tx.Model(&models.TSEMarket{}).Where("id IN ?", ids).Distinct().Pluck("market_id", &marketIDs).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.TSEMarket{}).Where("id IN ?", ids).Pluck("ins_code", &result.InsCodes).Error; err != nil {
			return err
		}
		// Prices are deleted by the CTE; candles that other instruments' prices
		// still use are kept.
		if err := tx.Exec(`
			WITH removed AS (
				DELETE FROM tse_daily_closing_prices WHERE tse_market_id IN ? RETURNING candle_id
			)
			DELETE FROM candles c
			WHERE c.id IN (SELECT candle_id FROM removed WHERE candle_id IS NOT NULL)
				AND NOT EXISTS (
					SELECT 1 FROM tse_daily_closing_prices p WHERE p.candle_id = c.id AND p.tse_market_id NOT IN ?
				)`, ids, ids).Error; err != nil {
			return fmt.Errorf("delete prices: %w", err)
		}
		if err := tx.Where("id IN ?", ids).Delete(&models.TSEMarket{}).Error; err != nil {
			return err
		}
		emptied, err := deleteEmptyMarkets(tx, marketIDs)
		result.MarketIDs = emptied
		return err
	})
	return result, err
}

// deleteEmptyMarkets deletes those of the markets that have no TSE instrument,
// with every row that belongs to them. It returns the deleted market IDs.
func deleteEmptyMarkets(tx *gorm.DB, marketIDs []uint) ([]uint, error) {
	var empty []uint
	if err := tx.Model(&models.Market{}).Unscoped().
		Where("id IN ? AND NOT EXISTS (SELECT 1 FROM tse_markets t WHERE t.market_id = markets.id)", marketIDs).
		Pluck("id", &empty).Error; err != nil {
		return nil, err
	}
	if len(empty) == 0 {
		return empty, nil
	}
	for _, table := range []string{
		"tse_daily_closing_prices", "tse_indicator_snapshots", "indicator_definitions",
		"stock_daily_data", "market_timeframes", "candles",
	} {
		if err := tx.Exec("DELETE FROM "+table+" WHERE market_id IN ?", empty).Error; err != nil {
			return nil, fmt.Errorf("delete %s of markets: %w", table, err)
		}
	}
	// Markets are deleted for good, so that their labels can be used again.
	if err := tx.Unscoped().Where("id IN ?", empty).Delete(&models.Market{}).Error; err != nil {
		return nil, fmt.Errorf("delete markets: %w", err)
	}
	return empty, nil
}

func (r *TSEMarketRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	return count, r.db.WithContext(ctx).Model(&models.TSEMarket{}).Count(&count).Error
}

// ListRelated returns the TSE markets with the given insCodes or linked to the given markets.
func (r *TSEMarketRepository) ListRelated(ctx context.Context, insCodes []string, marketIDs []uint) ([]*models.TSEMarket, error) {
	markets := make([]*models.TSEMarket, 0)
	if len(insCodes) == 0 && len(marketIDs) == 0 {
		return markets, nil
	}
	tx := r.db.WithContext(ctx).Preload("Market")
	switch {
	case len(insCodes) > 0 && len(marketIDs) > 0:
		tx = tx.Where("ins_code IN ? OR market_id IN ?", insCodes, marketIDs)
	case len(insCodes) > 0:
		tx = tx.Where("ins_code IN ?", insCodes)
	default:
		tx = tx.Where("market_id IN ?", marketIDs)
	}
	return markets, tx.Order("market_id ASC").Order("delisted ASC").Order("id ASC").Find(&markets).Error
}
