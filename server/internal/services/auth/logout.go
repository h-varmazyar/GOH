package auth

import (
	"context"
	"fmt"
	"strings"

	jwtPkg "github.com/h-varmazyar/gopack/jwt"
)

func (s *Service) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer"))
	fmt.Println(token)
	return jwtPkg.Logout(token)
}
