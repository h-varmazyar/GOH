package gormext

import (
	"context"
	"gorm.io/gorm"
)

type DBManager struct {
	db     *gorm.DB
	trxKey trxContextKey
}

func (r *DBManager) DB(ctx context.Context) *gorm.DB {
	if tr, ok := ctx.Value(r.trxKey).(*gorm.DB); ok {
		return tr
	}

	return r.db
}
