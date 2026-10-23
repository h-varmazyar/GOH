package repositories

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type TSEMarketRepository struct{ db *gorm.DB }

func NewTSEMarketRepository(db *gorm.DB) *TSEMarketRepository {
	return &TSEMarketRepository{db: db}
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

func (r *TSEMarketRepository) UpdateByMarketID(ctx context.Context, marketID uint, columns map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.TSEMarket{}).Where("market_id = ?", marketID).Updates(columns).Error
}

func (r *TSEMarketRepository) Update(ctx context.Context, id uint, columns map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.TSEMarket{}).Where("id = ?", id).UpdateColumns(columns).Error
}

func (r *TSEMarketRepository) UpdateInstrumentInfo(ctx context.Context, id uint, symbol, name string, nav, esp *float64) error {
	values := map[string]any{}
	if symbol != "" {
		values["symbol"] = symbol
	}
	if name != "" {
		values["name"] = name
	}
	if nav != nil {
		values["nav"] = *nav
	}
	if esp != nil {
		values["esp"] = *esp
	}
	if len(values) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&models.TSEMarket{}).
		Where("id = ?", id).Updates(values).Error
}

func (r *TSEMarketRepository) UpdateInstrumentState(
	ctx context.Context,
	id uint,
	code, title string,
	underSupervision int64,
	lastPriceDate *time.Time,
) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.TSEMarket{}).Where("id = ?", id).Updates(map[string]any{
		"instrument_state_code":  code,
		"instrument_state_title": title,
		"under_supervision":      underSupervision,
		"last_price_date":        lastPriceDate,
		"state_updated_at":       now,
	}).Error
}
