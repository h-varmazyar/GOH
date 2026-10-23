package user

import "github.com/h-varmazyar/gatec/internal/services/user"

type Handler struct {
	userService *user.Service
}

func New(userService *user.Service) Handler {
	return Handler{
		userService: userService,
	}
}
