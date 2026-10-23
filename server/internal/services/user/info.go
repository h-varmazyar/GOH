package user

import (
	"context"

	"github.com/h-varmazyar/gatec/pkg/logger"
)

func (s *Service) GetUserInfo(ctx context.Context, userID uint) (*UserInfo, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	logger.Logger.Infof("user is: %v", user)

	return &UserInfo{
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		// Kyc:       user.Kyc,
		Mobile: user.Mobile,
	}, nil
}
