package workers

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/h-varmazyar/gatec/internal/adapters/stock"
// 	"github.com/h-varmazyar/gatec/internal/models"
// 	"github.com/h-varmazyar/gatec/internal/repositories"
// 	"github.com/shopspring/decimal"
// 	"github.com/sirupsen/logrus"
// )

// const dailyTimeframe = 1

// type TSEMarketHistorySync struct {
// 	adapter         *stock.Adapter
// 	marketRepo      *repositories.TSEMarketRepository
// 	dailyRepo       *repositories.TSEDailyClosingPriceRepository
// 	requestInterval time.Duration
// 	cancel          context.CancelFunc
// 	logger          *logrus.Entry
// }

// func NewTSEMarketHistorySync(
// 	marketRepo *repositories.TSEMarketRepository,
// 	dailyRepo *repositories.TSEDailyClosingPriceRepository,
// 	requestInterval time.Duration,
// ) *TSEMarketHistorySync {
// 	if requestInterval < 0 {
// 		requestInterval = 0
// 	}
// 	return &TSEMarketHistorySync{
// 		adapter:         stock.NewAdapter(),
// 		marketRepo:      marketRepo,
// 		dailyRepo:       dailyRepo,
// 		requestInterval: requestInterval,
// 		logger:          logrus.WithField("worker", "tse-market-history-sync"),
// 	}
// }

// func (w *TSEMarketHistorySync) Name() string { return "tse-market-history-sync" }

// func (w *TSEMarketHistorySync) Stop() error {
// 	if w.cancel != nil {
// 		w.cancel()
// 	}
// 	return nil
// }

// func (w *TSEMarketHistorySync) Start(ctx context.Context) error {
// 	ctx, w.cancel = context.WithCancel(ctx)

// 	instrumentIDs, err := w.dailyRepo.DistinctMarkets(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	condition := map[string]any{}
// 	if len(instrumentIDs) > 0 {
// 		condition["id NOT IN ?"] = instrumentIDs
// 	}
// 	markets, err := w.marketRepo.List(ctx, condition)
// 	if err != nil {
// 		return fmt.Errorf("list TSE markets: %w", err)
// 	}

// 	w.logger.WithField("markets", len(markets)).Info("starting TSE market history sync")
// 	failed := 0
// 	for index, market := range markets {
// 		if err := ctx.Err(); err != nil {
// 			return nil
// 		}
// 		if market.InsCode == "" {
// 			failed++
// 			w.logger.WithField("market_id", market.MarketID).Warn("market has no ins_code")
// 			continue
// 		}
// 		if err := w.syncMarket(ctx, market); err != nil {
// 			failed++
// 			w.logger.WithFields(logrus.Fields{"market_id": market.MarketID, "ins_code": market.InsCode}).
// 				WithError(err).Error("failed to sync TSE market history")
// 		}
// 		if index < len(markets)-1 && !w.wait(ctx) {
// 			return nil
// 		}
// 	}
// 	w.logger.WithFields(logrus.Fields{"markets": len(markets), "failed": failed}).Info("TSE market history sync completed")
// 	return nil
// }

// func (w *TSEMarketHistorySync) syncMarket(ctx context.Context, market *models.TSEMarket) error {
// 	info, err := w.adapter.FetchInstrumentInfo(ctx, market.InsCode)
// 	if err != nil {
// 		return fmt.Errorf("fetch instrument info: %w", err)
// 	}

// 	marketUpdateColumns := map[string]any{
// 		"label": info.Symbol,
// 		"name":  info.Name,
// 		"nav":   info.Nav,
// 		"esp":   info.ESP,
// 	}

// 	states, err := w.adapter.FetchInstrumentStateHistory(ctx, info.InsCode)
// 	if err != nil {
// 		return fmt.Errorf("fetch instrument state history: %w", err)
// 	}
// 	if len(states) > 0 {
// 		latest := states[0]
// 		marketUpdateColumns["instrument_state_code"] = latest.Code
// 		marketUpdateColumns["state_updated_at"] = latest.Date
// 		marketUpdateColumns["status"] = "tradeable"
// 	} else {
// 		marketUpdateColumns["instrument_state_code"] = "N/A"
// 		marketUpdateColumns["state_updated_at"] = time.Now()
// 		marketUpdateColumns["status"] = "untradeable"
// 	}

// 	if err = w.marketRepo.Update(ctx, market.ID, marketUpdateColumns); err != nil {
// 		return err
// 	}

// 	prices, priceErr := w.adapter.FetchClosingPriceHistory(ctx, info.InsCode)
// 	if priceErr == nil {
// 		rows := make([]*models.TSEDailyClosingPrice, 0, len(prices))
// 		for _, item := range prices {
// 			lastPercent, closeChange, closePercent := 0.0, item.Close-item.Yesterday, 0.0
// 			if item.Yesterday != 0 {
// 				lastPercent = 100 * item.LastChange / item.Yesterday
// 				closePercent = 100 * closeChange / item.Yesterday
// 			}
// 			rows = append(rows, &models.TSEDailyClosingPrice{
// 				MarketID:    market.MarketID,
// 				TSEMarketID: market.ID,
// 				Candle: models.Candle{
// 					MarketID:    market.Market.ID,
// 					TimeframeID: dailyTimeframe,
// 					Time:        item.Date.Unix(),
// 					Open:        decimal.NewFromFloat(item.First),
// 					High:        decimal.NewFromFloat(item.High),
// 					Low:         decimal.NewFromFloat(item.Low),
// 					Close:       decimal.NewFromFloat(item.Close),
// 					Volume:      decimal.NewFromInt(item.Volume),
// 				},
// 				// InstrumentID:       market.MarketID,
// 				// Date:               item.Date,
// 				TradeCount: item.TradeCount,
// 				// Volume:             item.Volume,
// 				Value:     item.Value,
// 				Yesterday: item.Yesterday,
// 				// First:              item.First,
// 				Last:              item.Last,
// 				LastChange:        item.LastChange,
// 				LastChangePercent: lastPercent,
// 				// Close:              item.Close,
// 				CloseChange:        closeChange,
// 				CloseChangePercent: closePercent,
// 				// Low:                item.Low,
// 				// High:               item.High,
// 			})
// 		}
// 		// if err := w.dailyRepo.UpsertPrices(ctx, rows); err != nil {
// 		// 	priceErr = fmt.Errorf("upsert closing prices: %w", err)
// 		// }
// 	}

// 	participants, participantErr := w.adapter.FetchClientTypeHistory(ctx, info.InsCode)
// 	if participantErr == nil {
// 		rows := make([]*models.TSEDailyClosingPrice, 0, len(participants))
// 		for _, item := range participants {
// 			rows = append(rows, &models.TSEDailyClosingPrice{
// 				MarketID:    market.MarketID,
// 				TSEMarketID: market.ID,
// 				// InstrumentID:            market.MarketID,
// 				// Date:                    item.Date,
// 				BuyIndividualVolume:     item.BuyIndividualVolume,
// 				BuyIndividualValue:      item.BuyIndividualValue,
// 				BuyIndividualCount:      item.BuyIndividualCount,
// 				BuyInstitutionalVolume:  item.BuyInstitutionalVolume,
// 				BuyInstitutionalValue:   item.BuyInstitutionalValue,
// 				BuyInstitutionalCount:   item.BuyInstitutionalCount,
// 				SellIndividualVolume:    item.SellIndividualVolume,
// 				SellIndividualValue:     item.SellIndividualValue,
// 				SellIndividualCount:     item.SellIndividualCount,
// 				SellInstitutionalVolume: item.SellInstitutionalVolume,
// 				SellInstitutionalValue:  item.SellInstitutionalValue,
// 				SellInstitutionalCount:  item.SellInstitutionalCount,
// 			})
// 		}
// 		// if err := w.dailyRepo.UpsertParticipants(ctx, rows); err != nil {
// 		// 	participantErr = fmt.Errorf("upsert client types: %w", err)
// 		// }
// 	}

// 	if priceErr != nil || participantErr != nil {
// 		return fmt.Errorf("partial sync: closing prices: %v; client types: %v", priceErr, participantErr)
// 	}
// 	w.logger.WithFields(logrus.Fields{"market_id": market.MarketID, "prices": len(prices), "participants": len(participants)}).
// 		Info("synced TSE market history")
// 	return nil
// }

// func (w *TSEMarketHistorySync) wait(ctx context.Context) bool {
// 	if w.requestInterval == 0 {
// 		return true
// 	}
// 	timer := time.NewTimer(w.requestInterval)
// 	defer timer.Stop()
// 	select {
// 	case <-ctx.Done():
// 		return false
// 	case <-timer.C:
// 		return true
// 	}
// }
