package auth

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/h-varmazyar/gopack/validator"

	"github.com/h-varmazyar/gatec/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (h *Service) Register(ctx context.Context, req *RegisterReq) (*RegisterResp, error) {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	if req.FirstName == "" {
		return nil, fmt.Errorf("first name cannot be empty")
	}

	if req.LastName == "" {
		return nil, fmt.Errorf("last name cannot be empty")
	}

	if !validator.IsValidMobile(req.Mobile) {
		return nil, fmt.Errorf("invalid mobile number")
	}

	if !isStrongPassword(req.Password) {
		return nil, fmt.Errorf("password must be at least 8 characters and include uppercase, lowercase, numeric, and special characters")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.cfg.BcryptCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Mobile:         req.Mobile,
		Username:       req.Username,
		HashedPassword: string(hashedPassword),
	}

	if err := h.userRepository.Create(ctx, user); err != nil {
		return nil, err
	}

	return &RegisterResp{
		HashedID: h.generateHashedID(user.ID),
	}, nil
}

func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		case unicode.IsSpace(r):
			return false
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}
