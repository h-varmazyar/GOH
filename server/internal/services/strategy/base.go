package strategy

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/repositories"
)

type Service struct {
	indicatorDefinitionRepo *repositories.IndicatorDefinitionRepository
	marketRepo              *repositories.MarketRepository
	providerRepo            *repositories.ProviderRepository
	straegyRepo             *repositories.StrategyRepository
}

func NewService(indicatorDefinitionRepo *repositories.IndicatorDefinitionRepository, strategyRepo *repositories.StrategyRepository, marketRepo *repositories.MarketRepository, providerRepo *repositories.ProviderRepository) *Service {
	strategyService := &Service{
		indicatorDefinitionRepo: indicatorDefinitionRepo,
		straegyRepo:             strategyRepo,
		marketRepo:              marketRepo,
		providerRepo:            providerRepo,
	}

	strategyService.ImportNewStrategy(context.Background(), "./assets/sample_strategy.yaml")

	return strategyService
}
