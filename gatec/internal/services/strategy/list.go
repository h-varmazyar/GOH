package strategy

import (
	"context"
)

func (s *Service) List(ctx context.Context, userID uint) (StrategyListDTO, error) {
	strategies, err := s.straegyRepo.List(ctx, userID)
	if err != nil {
		return StrategyListDTO{}, err
	}

	strategyDTOs := make([]*StrategyDTO, len(strategies))
	for i, strategy := range strategies {
		strategyDTOs[i] = &StrategyDTO{
			Name:        strategy.Name,
			Description: strategy.Description,
			Status:      strategy.Status.PersianLabel(),
			// Provider:    strategy.Provider.PersianLabel(),
		}
	}

	return StrategyListDTO{Strategies: strategyDTOs}, nil
}
