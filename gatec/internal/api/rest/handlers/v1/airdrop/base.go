package airdrop

import service "github.com/h-varmazyar/gatec/internal/services/airdrop"

type Handler struct {
	service *service.Service
}

func New(service *service.Service) Handler { return Handler{service: service} }

func (h Handler) HasService() bool { return h.service != nil }
