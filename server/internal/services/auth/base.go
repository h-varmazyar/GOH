package auth

import (
	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	cfg            *configs.AuthService
	userRepository *repositories.UserRepository
}

func NewService(cfg *configs.AuthService, userRepository *repositories.UserRepository) *Service {
	if cfg.BcryptCost < bcrypt.MinCost || cfg.BcryptCost > bcrypt.MaxCost {
		logger.Logger.Errorf("⚠️ Invalid BCRYPT_COST=%v, using default=%d\n", cfg.BcryptCost, bcrypt.DefaultCost)
		cfg.BcryptCost = bcrypt.DefaultCost
	}
	return &Service{
		cfg:            cfg,
		userRepository: userRepository,
	}
}
