package auth

import "github.com/h-varmazyar/gatec/internal/services/auth"

type Handler struct {
	authService *auth.Service
}

func New(authService *auth.Service) Handler {
	return Handler{
		authService: authService,
	}
}
