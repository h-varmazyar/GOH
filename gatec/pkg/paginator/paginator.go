package paginator

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Params holds pagination parameters
type Params struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Response holds paginated response metadata
type Response struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// New creates a Params with defaults
func New(page, PageSize int) Params {
	if page < 1 {
		page = DefaultPage
	}
	if PageSize < 1 {
		PageSize = DefaultPageSize
	}
	if PageSize > MaxPageSize {
		PageSize = MaxPageSize
	}
	return Params{Page: page, PageSize: PageSize}
}

// FromContext extracts pagination params from gin query parameters
// Expected query params: ?page=1&page_size=20
func FromContext(ctx *gin.Context) Params {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", strconv.Itoa(DefaultPage)))
	PageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", strconv.Itoa(DefaultPageSize)))
	return New(page, PageSize)
}

// Offset calculates the offset for database queries
func (p Params) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the limit for database queries
func (p Params) Limit() int {
	return p.PageSize
}

// Apply applies pagination to a gorm query
func (p Params) Apply(db *gorm.DB) *gorm.DB {
	return db.Offset(p.Offset()).Limit(p.Limit())
}

// ToResponse creates a Response from params and total count
func (p Params) ToResponse(totalItems int64) Response {
	totalPages := int(totalItems) / p.PageSize
	if int(totalItems)%p.PageSize > 0 {
		totalPages++
	}
	return Response{
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
