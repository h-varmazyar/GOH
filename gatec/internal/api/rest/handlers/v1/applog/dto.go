package applog

import "github.com/h-varmazyar/gatec/pkg/paginator"

type listResponse struct {
	Data       []map[string]any   `json:"data"`
	Pagination paginator.Response `json:"pagination"`
}

type errorResponse struct {
	Error string `json:"error" example:"failed to read application logs"`
}
