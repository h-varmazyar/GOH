package gormext

// import (
// 	"context"

// 	"gorm.io/driver/mysql"
// 	"gorm.io/gorm"
// )

// const trxKeyPostfix = "_trx"

type trxContextKey string

// type TRXManager struct {
// 	db *gorm.DB
// 	// logger log.ILogger
// 	trxKey trxContextKey
// }

// func NewTxManager(logger any, db *gorm.DB, cfg mysql.Config) (*TRXManager, *DBManager) {
// 	trxKey := trxContextKey(cfg.DatabaseName + trxKeyPostfix)
// 	trxManager := &TRXManager{
// 		db: db,
// 		// logger: logger,
// 		trxKey: trxKey,
// 	}

// 	return trxManager, &DBManager{
// 		db:     db,
// 		trxKey: trxKey,
// 	}
// }

// func (r *TRXManager) Do(ctx context.Context, fn func(context.Context) error) error {
// 	var trx *gorm.DB
// 	if tr, ok := ctx.Value(r.trxKey).(*gorm.DB); ok {
// 		trx = tr
// 	} else {
// 		trx = r.db.Begin()
// 		ctx = context.WithValue(ctx, r.trxKey, trx)
// 	}

// 	defer func() {
// 		if err := recover(); err != nil {
// 			_ = r.rollback(ctx, trx)
// 			panic(err)
// 		}
// 	}()

// 	var err error
// 	if err = fn(ctx); err != nil {
// 		err := r.rollback(ctx, trx)
// 		if err != nil {
// 			return err
// 		}
// 	} else {
// 		if err := trx.Commit().Error; err != nil {
// 			// r.logger.Error(ctx, "failed to commit", err)
// 			return err
// 		}
// 	}

// 	return err
// }

// func (r *TRXManager) rollback(ctx context.Context, trx *gorm.DB) error {
// 	if err := trx.Rollback().Error; err != nil {
// 		// r.logger.Error(ctx, "failed to rollback", err)
// 		return err
// 	}

// 	return nil
// }
