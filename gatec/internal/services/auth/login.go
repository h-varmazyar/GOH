package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	jwtPkg "github.com/h-varmazyar/gopack/jwt"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) LoginByPassword(ctx context.Context, req *LoginByPasswordReq) (*LoginByPasswordResp, error) {
	user, err := s.userRepository.FindByMobile(ctx, req.Mobile)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password))
	if err != nil {
		return nil, err
	}

	accessToken, err := jwtPkg.Login(jwtPkg.LoginOptions{
		UserID:    fmt.Sprintf("%d", user.ID),
		Username:  user.Mobile,
		ExpiresIn: s.cfg.ExpirationTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := jwtPkg.Login(jwtPkg.LoginOptions{
		UserID:    fmt.Sprintf("%d", user.ID),
		Username:  user.Mobile,
		ExpiresIn: 7 * 24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	resp := LoginByPasswordResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.ExpirationTime.Seconds()),
		HashedID:     s.generateHashedID(user.ID),
	}

	return &resp, nil
}

func (s *Service) generateHashedID(originalID uint) string {
	// In production, use a per-environment secret salt or HMAC
	hash := sha256.Sum256([]byte(fmt.Sprintf("%v%v", originalID, s.cfg.HashingSalt)))
	return hex.EncodeToString(hash[:])
}
