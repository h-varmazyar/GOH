package user

import "github.com/h-varmazyar/gatec/internal/repositories"

type Service struct {
	userRepo *repositories.UserRepository
}

func NewService(userRepository *repositories.UserRepository) *Service {
	return &Service{
		userRepo: userRepository,
	}
}
