package workers

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/pipeline"
	indicatorsPkg "github.com/h-varmazyar/gatec/pkg/indicators"
	"github.com/h-varmazyar/gatec/pkg/logger"
)

type event struct {
	MarketID   uint
	Candle     models.Candle
	Indicators map[uint]indicatorsPkg.IndicatorValue
}

type SignalPipeline struct {
	marketRepo      repositories.MarketRepository
	queue           chan event
	pipelineService *pipeline.Service
}

func NewSignalPipelineWorker(marketRepo repositories.MarketRepository) *SignalPipeline {
	return &SignalPipeline{
		marketRepo:      marketRepo,
		queue:           make(chan event),
		pipelineService: pipeline.NewService(),
	}
}

func (w *SignalPipeline) Start() {
	go func() {
		for e := range w.queue {
			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
			market, err := w.marketRepo.ReturnByID(ctx, e.MarketID)
			if err != nil {
				logger.Logger.WithContext(ctx).WithError(err).Errorf("failed to fetch market with id %v", e.MarketID)
				cancelFunc()
				continue
			}
			event := pipeline.MarketEvent{
				Market:     *market,
				Candle:     e.Candle,
				Indicators: e.Indicators,
			}
			w.pipelineService.Process(ctx, event)

			cancelFunc()
		}
	}()
}

func (w *SignalPipeline) EnqueueNewEvent(marketID uint, candle models.Candle, indicatorsValue map[uint]indicatorsPkg.IndicatorValue) {
	w.queue <- event{
		MarketID:   marketID,
		Candle:     candle,
		Indicators: indicatorsValue,
	}
}

// type SignalChecker struct {
// 	cfg   *configs.SignalChecker
// 	cache *cache.Cache

// 	marketRepo    repositories.MarketRepository
// 	timeframeRepo repositories.TimeframeRepository
// }

// type SignalType string

// const (
// 	SignalTypeBuy  = "BUY"
// 	SignalTypeSell = "SELL"
// )

// type Signal struct {
// 	Strength float64
// 	Type     SignalType
// }

// func NewSignalChecker(cfg *configs.SignalChecker, cache *cache.Cache, marketRepo repositories.MarketRepository, timeframeRepo repositories.TimeframeRepository) *SignalChecker {
// 	return &SignalChecker{
// 		cfg:           cfg,
// 		cache:         cache,
// 		marketRepo:    marketRepo,
// 		timeframeRepo: timeframeRepo,
// 	}
// }

// func (w *SignalChecker) Name() string {
// 	return "signal-checker"
// }

// func (w *SignalChecker) Start(ctx context.Context) error {
// 	logrus.Info("starting signal checker: currently not acting as a worker!!!")
// 	return nil
// }

// func (w *SignalChecker) Stop() {}

// func (w *SignalChecker) CheckForSignal(ctx context.Context, marketID, timeframeID uint, candle models.Candle, indicators []indicatorsPkg.Indicator) error {
// 	signal := Signal{}
// 	signalValues := 0.0
// 	for _, indicator := range indicators {
// 		switch indicator.GetType() {
// 		case indicatorsPkg.IndicatorTypeRSI:
// 			signalValues += w.checkRSI(ctx, indicator.GetValues(ctx, 2))
// 		case indicatorsPkg.IndicatorTypeStochastic:
// 			signalValues += w.checkStochastic(ctx, indicator.GetValues(ctx, 2))
// 		case indicatorsPkg.IndicatorTypeBollingerBands:
// 			signalValues += w.checkBollingerBands(ctx, indicator.GetValues(ctx, 2))
// 		case indicatorsPkg.IndicatorTypeSMA:

// 		}
// 	}

// 	signal.Strength = signalValues / float64(len(indicators))
// 	if signal.Strength < 0 {
// 		signal.Type = SignalTypeSell
// 	} else if signal.Strength > 0 {
// 		signal.Type = SignalTypeBuy
// 	}

// 	market, err := w.marketRepo.ReturnByID(ctx, marketID)
// 	if err != nil {
// 		return err
// 	}
// 	tf, err := w.timeframeRepo.ReturnByID(ctx, timeframeID)
// 	if err != nil {
// 		return err
// 	}

// 	msg := "new signal raised"
// 	if signal.Type == SignalTypeBuy && signal.Strength > 0.8 {
// 		msg = fmt.Sprintf("%v\nBuy Asset: %v in timeframe %v\nSignal strength: %v\nEnter price: %v", msg, market.Name, tf.Label, signal.Strength, candle.Close)

// 	} else if signal.Type == SignalTypeSell && signal.Strength < -0.8 {

// 	}

// 	key := fmt.Sprintf("%v:%v:%v", market.ProviderID, marketID, timeframeID)
// 	w.cache.Add(key, signal, time.Minute*15)
// 	w.sendMessage(ctx, msg)

// 	return nil
// }

// func (w *SignalChecker) sendMessage(ctx context.Context, msg string) error {
// 	bodyPayload := struct {
// 		ChatID int
// 		Text   string
// 	}{
// 		ChatID: 9421,
// 		Text:   msg,
// 	}

// 	body, err := json.Marshal(bodyPayload)
// 	if err != nil {
// 		return err
// 	}

// 	req, err := http.NewRequest(http.MethodPost, "https://tapi.bale.ai/793303281:H44gsVDAQuhPgfvjRIFOtXxUswWubpzYZtw/sendMessage", bytes.NewBuffer(body))
// 	if err != nil {
// 		return err
// 	}

// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Accept", "application/json")

// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	return nil
// }

// func (w *SignalChecker) checkRSI(ctx context.Context, values []indicatorsPkg.IndicatorValue) float64 {
// 	if len(values) < 2 {
// 		return 0
// 	}

// 	first := (values[0].Value).(*indicatorsPkg.RSIValue).RSI
// 	second := (values[1].Value).(*indicatorsPkg.RSIValue).RSI

// 	buyLevel := decimal.NewFromInt(30)
// 	sellLevel := decimal.NewFromInt(70)

// 	if first.LessThan(buyLevel) && second.GreaterThanOrEqual(buyLevel) {
// 		return 1
// 	}

// 	if first.GreaterThan(sellLevel) && second.LessThanOrEqual(sellLevel) {
// 		return -1
// 	}

// 	return 0
// }

// func (w *SignalChecker) checkStochastic(ctx context.Context, values []indicatorsPkg.IndicatorValue) float64 {
// 	if len(values) < 2 {
// 		return 0
// 	}

// 	score := 0.0

// 	first := (values[0].Value).(*indicatorsPkg.StochasticValue)
// 	second := (values[1].Value).(*indicatorsPkg.StochasticValue)

// 	buyLevel := decimal.NewFromInt(20)
// 	sellLevel := decimal.NewFromInt(80)

// 	if first.IndexK.LessThanOrEqual(first.IndexD) && second.IndexK.GreaterThan(second.IndexD) {
// 		score += 0.5
// 		if first.IndexK.GreaterThanOrEqual(second.IndexK) {
// 			score += 0.25
// 		} else if first.IndexK.LessThanOrEqual(second.IndexK) {
// 			score += 0.5
// 		}
// 	} else if first.IndexK.GreaterThanOrEqual(first.IndexD) && second.IndexK.GreaterThanOrEqual(second.IndexD) {
// 		score += 0.5
// 		if first.IndexK.GreaterThanOrEqual(second.IndexK) {
// 			score += 0.25
// 		} else if first.IndexK.LessThanOrEqual(second.IndexK) {
// 			score += 0.5
// 		}
// 	} else if first.IndexK.LessThanOrEqual(first.IndexD) && second.IndexK.GreaterThan(second.IndexD) {
// 		score -= 0.5
// 		if first.IndexD.LessThanOrEqual(second.IndexD) {
// 			score += 0.25
// 		} else if first.IndexK.GreaterThanOrEqual(second.IndexK) {
// 			score += 0.5
// 		}
// 	} else if first.IndexK.LessThanOrEqual(first.IndexD) && second.IndexK.LessThanOrEqual(second.IndexD) {
// 		score -= 0.5
// 	}

// 	// if first.IndexK

// 	return score

// }

// func (w *SignalChecker) checkBollingerBands(ctx context.Context, values []indicatorsPkg.IndicatorValue) float64 {

// }
