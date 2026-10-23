package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const tseInstrumentSyncWorkerName = "tse-instrument-sync"

type TseInstrumentSync struct {
	cfg        *configs.TSEInstrumentSync
	adapters   []*tse.Adapter
	marketRepo *repositories.MarketRepository
	assetRepo  *repositories.AssetRepository
	tseRepo    *repositories.TSEMarketRepository
	location   *time.Location
	cancel     context.CancelFunc
}

func NewTSEInstrumentSync(
	cfg *configs.TSEInstrumentSync,
	adapters []*tse.Adapter,
	marketRepo *repositories.MarketRepository,
	assetRepo *repositories.AssetRepository,
	tseRepo *repositories.TSEMarketRepository,
) *TseInstrumentSync {
	location, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		location = time.FixedZone("Asia/Tehran", 3*60*60+30*60)
	}
	return &TseInstrumentSync{
		cfg:        cfg,
		adapters:   adapters,
		marketRepo: marketRepo,
		assetRepo:  assetRepo,
		tseRepo:    tseRepo,
		location:   location,
	}
}

func (w *TseInstrumentSync) Name() string { return tseInstrumentSyncWorkerName }

func (w *TseInstrumentSync) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	if w.cfg.SingleCheck {
		if err := w.sync(ctx); err != nil {
			logrus.WithError(err).Error("tse instrument sync failed")
			return err
		}
		w.Stop()
		logrus.WithContext(ctx).Info("tse instrument sync done")
		return nil
	}
	for {
		next := w.nextRunTime()
		timer := time.NewTimer(time.Until(next))
		select {
		case <-timer.C:
			if err := w.sync(ctx); err != nil {
				logrus.WithError(err).Error("tse instrument sync failed")
			}
		case <-ctx.Done():
			timer.Stop()
			return nil
		}
	}
}

func (w *TseInstrumentSync) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

func (w *TseInstrumentSync) nextRunTime() time.Time {
	now := time.Now().In(w.location)
	next := time.Date(now.Year(), now.Month(), now.Day(), w.cfg.ScheduleHour, w.cfg.ScheduleMin, 0, 0, w.location)
	if !now.Before(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (w *TseInstrumentSync) sync(ctx context.Context) error {
	destination, err := w.ensureAsset(ctx, "IRI", "ریال ایران")
	if err != nil {
		return fmt.Errorf("ensure destination asset: %w", err)
	}
	for _, adapter := range w.adapters {
		if err := w.syncProvider(ctx, adapter, destination); err != nil {
			logrus.WithError(err).WithField("provider", adapter.GetID()).Error("failed to sync tse provider")
		}
	}
	return nil
}

func (w *TseInstrumentSync) syncProvider(ctx context.Context, adapter *tse.Adapter, destination *models.Asset) error {
	instruments, err := adapter.UpdateInstruments(ctx)
	if err != nil {
		return fmt.Errorf("fetch instruments: %w", err)
	}
	localMarkets, err := w.marketRepo.List(ctx, map[string]any{"provider_id = ?": adapter.GetID()})
	if err != nil {
		return fmt.Errorf("list markets: %w", err)
	}

	localTSEMarkets, err := w.tseRepo.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("list TSE markets: %w", err)
	}

	marketsByLabel := make(map[string]*models.Market, len(localMarkets))
	for _, market := range localMarkets {
		marketsByLabel[market.Label] = market
	}
	marketsByInsCode := make(map[string]*models.TSEMarket, len(localTSEMarkets))
	for _, market := range localTSEMarkets {
		marketsByInsCode[market.InsCode] = market
	}

	for _, instrument := range instruments {
		if err := w.syncInstrument(ctx, instrument, adapter.GetID(), destination, marketsByLabel, marketsByInsCode); err != nil {
			logrus.WithError(err).WithField("instrument", instrument.ID).Error("failed to sync tse instrument")
		}
	}
	return nil
}

func (w *TseInstrumentSync) syncInstrument(
	ctx context.Context,
	instrument tse.Instrument,
	providerID uint,
	destination *models.Asset,
	marketsByLabel map[string]*models.Market,
	marketsByInsCode map[string]*models.TSEMarket,
) error {
	logrus.WithFields(logrus.Fields{
		"instrument": instrument.ID,
		"label":      instrument.Label,
		"name":       instrument.Name,
	}).Info("syncing tse instrument")
	status := instrumentMarketStatus(instrument.Status)
	tseMarket := marketsByInsCode[instrument.ID]
	market := marketsByLabel[instrument.Label]

	if market != nil && tseMarket != nil && tseMarket.MarketID == market.ID {
		return nil
	}

	trading, err := w.ensureAsset(ctx, instrument.Label, instrument.Name)
	if err != nil {
		return fmt.Errorf("ensure trading asset: %w", err)
	}

	if market == nil {
		market = &models.Market{
			ProviderID:     providerID,
			PricingAssetID: destination.ID,
			TradingAssetID: trading.ID,
			Name:           instrument.Name,
			Label:          instrument.Label,
			Status:         status,
		}
		if tseMarket != nil {
			if err := w.marketRepo.Create(ctx, market); err != nil {
				return fmt.Errorf("create market %q: %w", instrument.Label, err)
			}
		} else {
			tseMarket = &models.TSEMarket{
				InsCode:           instrument.ID,
				Label:             instrument.Label,
				MarketType:        instrument.Type,
				IndustryGroupCode: instrument.Group,
				IndustrySubCode:   instrument.SubGroup,
			}
			if err := w.tseRepo.CreateWithMarket(ctx, market, tseMarket); err != nil {
				return fmt.Errorf("create market %q: %w", instrument.Label, err)
			}
		}
		marketsByLabel[market.Label] = market
	}

	if err := w.marketRepo.Update(ctx, market.ID, map[string]any{
		"pricing_asset_id": destination.ID,
		"trading_asset_id": trading.ID,
		"name":             instrument.Name,
		"label":            instrument.Label,
		"status":           status,
	}); err != nil {
		return fmt.Errorf("update market %d: %w", market.ID, err)
	}
	market.PricingAssetID = destination.ID
	market.TradingAssetID = trading.ID
	market.Name = instrument.Name
	market.Label = instrument.Label
	market.Status = status

	if tseMarket == nil {
		tseMarket = &models.TSEMarket{
			MarketID:          market.ID,
			InsCode:           instrument.ID,
			Label:             instrument.Label,
			MarketType:        instrument.Type,
			IndustryGroupCode: instrument.Group,
			IndustrySubCode:   instrument.SubGroup,
		}
		if err := w.tseRepo.Create(ctx, tseMarket); err != nil {
			return fmt.Errorf("create TSE market for %d: %w", market.ID, err)
		}
	} else if tseMarket.MarketID != market.ID || tseMarket.Label != market.Label {
		if err := w.tseRepo.Update(ctx, tseMarket.ID, map[string]any{
			"market_id": market.ID,
			"label":     instrument.Label,
		}); err != nil {
			return fmt.Errorf("associate TSE market %d with market %d: %w", tseMarket.ID, market.ID, err)
		}
		tseMarket.MarketID = market.ID
		tseMarket.Label = instrument.Label
	}
	tseMarket.Market = market
	tseMarket.Label = instrument.Label
	tseMarket.MarketType = instrument.Type
	tseMarket.IndustryGroupCode = instrument.Group
	tseMarket.IndustrySubCode = instrument.SubGroup
	marketsByInsCode[instrument.ID] = tseMarket
	return nil
}

func (w *TseInstrumentSync) ensureAsset(ctx context.Context, label, name string) (*models.Asset, error) {
	asset, err := w.assetRepo.ReturnByLabel(ctx, label)
	if err == nil {
		return asset, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	asset = &models.Asset{Label: label, Name: name, Type: consts.AssetStock}
	if err := w.assetRepo.Create(ctx, asset); err != nil {
		return nil, err
	}
	return asset, nil
}

func instrumentMarketStatus(status consts.InstrumentStatus) consts.MarketStatus {
	if status == consts.InstrumentStatusEnable {
		return consts.MarketStatusOpen
	}
	return consts.MarketStatusClose
}
