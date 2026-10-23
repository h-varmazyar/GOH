package workers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type MarketSync struct {
	marketRepo    *repositories.MarketRepository
	assetRepo     *repositories.AssetRepository
	timeframeRepo *repositories.TimeframeRepository
	providers     []crypto.Crypto
	// candlePollingWorker *CandlePolling
	syncInterval time.Duration
}

func NewMarketSync(
	marketRepo *repositories.MarketRepository,
	assetRepo *repositories.AssetRepository,
	timeframeRepo *repositories.TimeframeRepository,
	providers []crypto.Crypto,
	// candlePollingWorker *CandlePolling,
	syncInterval time.Duration,
) *MarketSync {
	if syncInterval <= 0 {
		syncInterval = time.Hour
	}
	ms := &MarketSync{
		marketRepo:    marketRepo,
		assetRepo:     assetRepo,
		timeframeRepo: timeframeRepo,
		providers:     providers,
		// candlePollingWorker: candlePollingWorker,
		syncInterval: syncInterval,
	}
	return ms
}

func (w *MarketSync) Name() string {
	return "market-sync"
}

func (w *MarketSync) Start(ctx context.Context) error {
	logrus.Info("starting market sync")

	go w.getMarkets(ctx)

	return nil
}

func (w *MarketSync) Stop() error { return nil }

func (w *MarketSync) getMarkets(ctx context.Context) {
	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			logrus.Infof("updating markets: %v", len(w.providers))
			for _, provider := range w.providers {
				localMarkets, err := w.marketRepo.List(ctx, map[string]any{
					"provider_id = ?": provider.GetID(),
				})
				if err != nil {
					continue
				}

				remoteMarkets, err := provider.UpdateMarkets(ctx)
				if err != nil {
					logrus.WithError(err).Errorf("failed to update markets for provider %v", provider.GetID())
					continue
				}

				logrus.Infof("remote counts %v", len(remoteMarkets))

				marketMap := make(map[string]*models.Market)
				for _, market := range localMarkets {
					marketMap[market.Name] = market
				}

				for _, market := range remoteMarkets {
					// todo: consider to remove obtaned remote marekts
					localMarket, ok := marketMap[market.Name]
					if !ok {
						market, err := w.createNewMarket(ctx, &market, provider.GetID())
						if err != nil {
							logrus.WithError(err).Errorf("failed to create new market %v for provider %v", market.Name, provider.GetID())
							continue
						}
						logrus.Infof("Inserting new market: %v", market)
						// if w.candlePollingWorker != nil {
						// 	w.candlePollingWorker.InsertMarket(ctx, provider, market)
						// }
						continue
					}
					if (localMarket.Status == consts.MarketStatusClose || localMarket.Status == consts.MarketStatusOpen) && localMarket.Status != market.Status {
						err = w.marketRepo.UpdateStatus(ctx, localMarket.ID, market.Status)
						if err != nil {
							logrus.WithError(err).Errorf("failed to update market status: %v -> %v", localMarket.Status, market.Status)
							continue
						}
						// if w.candlePollingWorker != nil && localMarket.Status == consts.MarketStatusClose {
						// 	w.candlePollingWorker.RemoveMarket(ctx, provider, localMarket)
						// }
						//
						// if w.candlePollingWorker != nil && localMarket.Status == consts.MarketStatusOpen {
						// 	logrus.Infof("Inserting new market: %v", localMarket)
						// 	w.candlePollingWorker.InsertMarket(ctx, provider, localMarket)
						// }
					}
				}

			}

		case <-ctx.Done():
			return
		}
	}
}

func (w *MarketSync) createNewMarket(ctx context.Context, market *crypto.Market, providerID uint) (*models.Market, error) {
	labels := strings.Split(market.Name, "-")

	if len(labels) != 2 {
		return nil, errors.New("invalid market name")
	}

	tradingAsset, err := w.fetchAsset(ctx, labels[0])
	if err != nil {
		return nil, err
	}

	pricingAsset, err := w.fetchAsset(ctx, labels[1])
	if err != nil {
		return nil, err
	}

	timeframes, err := w.timeframeRepo.List(ctx, map[string]any{
		"provider_id = ?": providerID,
	})
	if err != nil {
		return nil, err
	}

	localMarket := &models.Market{
		ProviderID:     providerID,
		PricingAssetID: pricingAsset.ID,
		TradingAssetID: tradingAsset.ID,
		Name:           market.Name,
		Status:         market.Status,
		TimeFrames:     timeframes,
	}
	if err := w.marketRepo.Create(ctx, localMarket); err != nil {
		return nil, err
	}

	return localMarket, nil
}

func (w *MarketSync) fetchAsset(ctx context.Context, asseyLabel string) (*models.Asset, error) {
	asset, err := w.assetRepo.ReturnByLabel(ctx, asseyLabel)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			asset = &models.Asset{
				Label: asseyLabel,
				Type:  consts.AssetCrypto,
			}
			if err := w.assetRepo.Create(ctx, asset); err != nil {
				return nil, err
			}
		}
		return nil, err
	}

	return asset, nil
}
