package gormext

import "context"

type DBModel interface {
	Table() string
}

type Repository[T DBModel] interface {
	GetByID(ctx context.Context, id uint, preload ...string) (T, error)
	GetByField(ctx context.Context, field string, value any, preload ...string) (T, error)
	GetByFields(ctx context.Context, conditions map[string]interface{}, preload ...string) (T, error)
	CountByFields(ctx context.Context, conditions map[string]interface{}) (int64, error)
	ListByFields(ctx context.Context, selectFields []string, preload []string, conditions map[string]interface{}, order string) ([]T, error)
	// Paginate(ctx context.Context, conditions map[string]interface{}, paginator paginator.Paginate, order string, preload ...string) ([]T, int64, error)
	Create(ctx context.Context, model *T) (*T, error)
	Update(ctx context.Context, conditions map[string]interface{}, model *T, selectQuery ...string) error
	Save(ctx context.Context, model *T) error
	Upsert(ctx context.Context, conditions map[string]interface{}, model *T, selectQuery ...string) error
	DeleteByFields(ctx context.Context, conditions map[string]interface{}) error
	GetByFieldsUnscoped(ctx context.Context, conditions map[string]interface{}, preload ...string) (T, error)
	UpdateUnscoped(ctx context.Context, conditions map[string]interface{}, models *T, selectQuery ...string) error
}
