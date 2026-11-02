package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/tsemarket"
	"github.com/sirupsen/logrus"
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

// Start runs the catalog sync once in single check mode. Otherwise it runs it right
// away when no TSE market is stored, and then every IntervalDays days at the
// scheduled time to verify the stored markets.
func (w *TseInstrumentSync) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	logger := w.logger()
	logger.WithFields(logrus.Fields{
		"single_check": w.cfg.SingleCheck, "interval_days": max(w.cfg.IntervalDays, 1),
		"schedule": fmt.Sprintf("%02d:%02d", w.cfg.ScheduleHour, w.cfg.ScheduleMin), "providers": len(w.adapters),
	}).Info("starting tse instrument sync worker")
	if w.cfg.SingleCheck {
		if err := w.sync(ctx); err != nil {
			logger.WithError(err).Error("tse instrument sync failed")
			return err
		}
		w.Stop()
		logger.Info("tse instrument sync done; single check mode, the worker stops")
		return nil
	}
	if count, err := w.tseRepo.Count(ctx); err != nil {
		logger.WithError(err).Error("count tse markets; the first sync waits for its scheduled time")
	} else if count == 0 {
		logger.Info("no TSE market is stored; syncing the instrument catalog now")
		if err := w.sync(ctx); err != nil {
			logger.WithError(err).Error("tse instrument sync failed")
		}
	} else {
		logger.WithField("tse_markets", count).Info("TSE markets are stored; the catalog is verified at the scheduled time")
	}
	next := w.firstRunTime()
	for {
		logger.WithFields(logrus.Fields{"next_run": next.Format(time.DateTime), "in": time.Until(next).Round(time.Minute)}).
			Info("next tse instrument sync scheduled")
		timer := time.NewTimer(time.Until(next))
		select {
		case <-timer.C:
			if err := w.sync(ctx); err != nil {
				logger.WithError(err).Error("tse instrument sync failed")
			}
			next = next.AddDate(0, 0, max(w.cfg.IntervalDays, 1))
		case <-ctx.Done():
			timer.Stop()
			logger.Info("tse instrument sync worker stopped")
			return nil
		}
	}
}

func (w *TseInstrumentSync) logger() *logrus.Entry {
	return logrus.WithField("worker", tseInstrumentSyncWorkerName)
}

func (w *TseInstrumentSync) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// firstRunTime is the scheduled time IntervalDays days from today.
func (w *TseInstrumentSync) firstRunTime() time.Time {
	now := time.Now().In(w.location)
	return time.Date(now.Year(), now.Month(), now.Day(), w.cfg.ScheduleHour, w.cfg.ScheduleMin, 0, 0, w.location).
		AddDate(0, 0, max(w.cfg.IntervalDays, 1))
}

func (w *TseInstrumentSync) sync(ctx context.Context) error {
	if len(w.adapters) == 0 {
		w.logger().Warn("no TSE provider is active; nothing to sync")
		return nil
	}
	for _, adapter := range w.adapters {
		service := tsemarket.NewService(w.tseRepo, w.marketRepo, w.assetRepo, adapter)
		logger := w.logger().WithField("provider", adapter.GetID())
		logger.Info("syncing TSE instrument catalog")
		start := time.Now()
		result, err := service.SyncCatalog(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.WithError(ctx.Err()).Warn("tse instrument sync canceled")
				return ctx.Err()
			}
			logger.WithError(err).WithField("duration", time.Since(start).Round(time.Second)).Error("failed to sync tse provider")
			continue
		}
		logger.WithFields(logrus.Fields{
			"instruments": result.Instruments, "created": result.Created, "updated": result.Updated, "failed": result.Failed,
			"markets_created": result.Markets, "issues": result.Issues, "duration": time.Since(start).Round(time.Second),
		}).Info("stored TSE instrument catalog; review the issues in the panel before collecting data")
	}
	return nil
}
