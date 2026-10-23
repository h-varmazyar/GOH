package pipeline

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/indicators"
	strategies "github.com/h-varmazyar/gatec/pkg/strategy"
)

type MarketEvent struct {
	Market     models.Market
	Candle     models.Candle
	Indicators map[uint]indicators.IndicatorValue
}

type StateRepository interface {
	Get(models.Market) *strategies.SymbolState
}

type ExecutionRequest struct {
	Symbol   string
	Signal   strategies.Signal
	Strategy string
}

type Executor interface{ Execute(ExecutionRequest) }

type Service struct {
	strategies []strategies.Strategy
	stateRepo  StateRepository
	executor   Executor
}

func NewService(strategyList ...strategies.Strategy) *Service {
	return &Service{strategies: strategyList}
}

func (s *Service) Process(ctx context.Context, event MarketEvent) {
	_ = ctx
	marketCtx := s.buildContext(event)

	for _, strat := range s.strategies {

		features := strat.ExtractFeatures(marketCtx)

		regime := strat.DetectRegime(marketCtx, features)

		signal := strat.GenerateSignal(marketCtx, features, regime)

		risk := strat.EvaluateRisk(marketCtx, signal)

		if risk.AllowTrade {
			symbol := event.Market.Name
			if s.executor != nil {
				s.executor.Execute(ExecutionRequest{
					Symbol:   symbol,
					Signal:   signal,
					Strategy: strat.Name(),
				})
			}
		}
	}
}

func (s *Service) buildContext(event MarketEvent) strategies.MarketContext {
	var state *strategies.SymbolState
	if s.stateRepo != nil {
		state = s.stateRepo.Get(event.Market)
	}
	if state == nil {
		state = &strategies.SymbolState{}
	}

	return strategies.MarketContext{
		Event: strategies.MarketEvent{
			Market:     event.Market,
			Candle:     event.Candle,
			Indicators: event.Indicators,
		},
		Close:      event.Candle.Close,
		PrevClose:  event.Candle.Close,
		Indicators: event.Indicators,
		State:      state,
	}
}
