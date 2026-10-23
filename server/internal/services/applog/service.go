package applog

import (
	"context"
	"errors"
	"strings"

	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

var ErrComponentRequired = errors.New("component is required")

type Service struct {
	repository *repositories.ApplicationLogRepository
}

func NewService(repository *repositories.ApplicationLogRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(
	ctx context.Context,
	component string,
	pg paginator.Params,
) ([]map[string]any, int64, error) {
	component = strings.TrimSpace(component)
	if component == "" {
		return nil, 0, ErrComponentRequired
	}
	return s.repository.List(ctx, component, pg)
}
