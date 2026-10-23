package gormext

// import (
// 	"context"
// 	"errors"
// 	"gorm.io/gorm"
// )

// type commonBehavior[T DBModel] struct {
// 	*DBManager
// }

// func NewCommonBehavior[T DBModel](dbManager *DBManager) Repository[T] {
// 	return &commonBehavior[T]{
// 		dbManager,
// 	}
// }

// // GetByID implements CommonBehaviorRepository.
// func (c *commonBehavior[T]) GetByID(ctx context.Context, id uint, preload ...string) (T, error) {
// 	var t T
// 	t, err := c.GetByField(ctx, "id", id, preload...)
// 	if err != nil {
// 		return t, err
// 	}

// 	return t, nil
// }

// // GetByField implements CommonBehaviorRepository.
// func (c *commonBehavior[T]) GetByField(ctx context.Context, field string, value any, preload ...string) (T, error) {
// 	var t T
// 	db := c.DB(ctx).WithContext(ctx).Where(field+"=?", value)
// 	for _, p := range preload {
// 		db = db.Preload(p)
// 	}
// 	err := db.First(&t).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			cErr := dexception.NewRecordNotFoundException(err)
// 			code := cErr.GetCode()
// 			message := cErr.GetMessage()
// 			return t, exception.NewNotFoundError(err, &code, &message)
// 		}
// 		return t, exception.NewUnExpectedError(err)

// 	}
// 	return t, err
// }

// func (c *commonBehavior[T]) GetByFields(ctx context.Context, conditions map[string]interface{}, preload ...string) (T, error) {
// 	var t T

// 	db := c.DB(ctx).WithContext(ctx)

// 	for condition, args := range conditions {
// 		switch v := args.(type) {
// 		case []interface{}:
// 			db = db.Where(condition, v...)
// 		case interface{}:
// 			db = db.Where(condition, v)
// 		}
// 	}

// 	for _, p := range preload {
// 		db = db.Preload(p)
// 	}

// 	err := db.First(&t).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			cErr := dexception.NewRecordNotFoundException(err)
// 			code := cErr.GetCode()
// 			message := cErr.GetMessage()
// 			return t, exception.NewNotFoundError(err, &code, &message)
// 		}
// 		return t, exception.NewUnExpectedError(err)
// 	}

// 	return t, err
// }

// func (c *commonBehavior[T]) ListByFields(ctx context.Context, selectFields []string, preload []string, conditions map[string]interface{}, order string) ([]T, error) {
// 	var results []T

// 	db := c.DB(ctx).WithContext(ctx)

// 	for condition, args := range conditions {
// 		switch v := args.(type) {
// 		case []interface{}:
// 			db = db.Where(condition, v...)
// 		case interface{}:
// 			db = db.Where(condition, v)
// 		}
// 	}

// 	for _, p := range preload {
// 		db = db.Preload(p)
// 	}

// 	if len(selectFields) > 0 {
// 		db = db.Select(selectFields)
// 	}

// 	err := db.
// 		Order(order).
// 		Find(&results).
// 		Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			cErr := dexception.NewRecordNotFoundException(err)
// 			code := cErr.GetCode()
// 			message := cErr.GetMessage()
// 			return results, exception.NewNotFoundError(err, &code, &message)
// 		}
// 		return results, exception.NewUnExpectedError(err)
// 	}

// 	return results, err
// }

// func (c *commonBehavior[T]) Paginate(ctx context.Context, conditions map[string]interface{}, paginator paginator.Paginate, order string, preload ...string) ([]T, int64, error) {
// 	var t T
// 	var results []T
// 	var total int64

// 	db := c.DB(ctx).WithContext(ctx).
// 		Model(t)

// 	for condition, args := range conditions {
// 		switch v := args.(type) {
// 		case []interface{}:
// 			db = db.Where(condition, v...)
// 		case interface{}:
// 			db = db.Where(condition, v)
// 		}
// 	}

// 	for _, p := range preload {
// 		db = db.Preload(p)
// 	}

// 	err :=
// 		db.
// 			Count(&total).
// 			Offset(paginator.From).
// 			Limit(paginator.Size).
// 			Order(order).
// 			Find(&results).
// 			Error

// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			cErr := dexception.NewRecordNotFoundException(err)
// 			code := cErr.GetCode()
// 			message := cErr.GetMessage()
// 			return results, 0, exception.NewNotFoundError(err, &code, &message)
// 		}
// 		return results, 0, exception.NewUnExpectedError(err)
// 	}

// 	return results, total, nil
// }

// func (c *commonBehavior[T]) Create(ctx context.Context, models *T) (*T, error) {
// 	result := c.DB(ctx).WithContext(ctx).Create(models)

// 	return models, result.Error
// }

// func (c *commonBehavior[T]) Update(ctx context.Context, conditions map[string]interface{}, models *T, selectQuery ...string) error {
// 	db := c.DB(ctx).WithContext(ctx).Select(selectQuery)
// 	for condition, args := range conditions {
// 		switch v := args.(type) {
// 		case []interface{}:
// 			db = db.Where(condition, v...)
// 		case interface{}:
// 			db = db.Where(condition, v)
// 		}
// 	}

// 	err := db.Updates(models).Error
// 	if err != nil {
// 		cErr := dexception.NewUnExpectedException(err)
// 		code := cErr.GetCode()
// 		message := cErr.GetMessage()
// 		return exception.NewNotFoundError(err, &code, &message)
// 	}

// 	return nil
// }

// // Save implements CommonBehaviorRepository.
// func (c *commonBehavior[T]) Save(ctx context.Context, models *T) error {
// 	err := c.DB(ctx).WithContext(ctx).Save(models).Error
// 	if err != nil {
// 		cErr := dexception.NewUnExpectedException(err)
// 		code := cErr.GetCode()
// 		message := cErr.GetMessage()
// 		return exception.NewNotFoundError(err, &code, &message)
// 	}

// 	return nil
// }

// func (c *commonBehavior[T]) Upsert(ctx context.Context, conditions map[string]interface{}, models *T, selectQuery ...string) error {
// 	_, err := c.GetByFields(ctx, conditions)
// 	if err != nil {
// 		_, err := c.Create(ctx, models)
// 		if err != nil {
// 			cErr := dexception.NewUnExpectedException(err)
// 			code := cErr.GetCode()
// 			message := cErr.GetMessage()
// 			return exception.NewNotFoundError(err, &code, &message)
// 		}
// 		return nil
// 	}

// 	err = c.Update(ctx, conditions, models, selectQuery...)
// 	if err != nil {
// 		cErr := dexception.NewUnExpectedException(err)
// 		code := cErr.GetCode()
// 		message := cErr.GetMessage()
// 		return exception.NewNotFoundError(err, &code, &message)
// 	}

// 	return nil
// }

// func (c *commonBehavior[T]) CountByFields(ctx context.Context, conditions map[string]interface{}) (int64, error) {
// 	var count int64
// 	var table T
// 	db := c.DB(ctx).WithContext(ctx).Model(&table)

// 	for condition, args := range conditions {
// 		switch v := args.(type) {
// 		case []interface{}:
// 			db = db.Where(condition, v...)
// 		case interface{}:
// 			db = db.Where(condition, v)
// 		}
// 	}

// 	err := db.Count(&count).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return 0, nil
// 		}
// 		return count, exception.NewUnExpectedError(err)
// 	}

// 	return count, err
// }

// func (c *commonBehavior[T]) DeleteByFields(ctx context.Context, conditions map[string]interface{}) error {
// 	var count int64
// 	var table T
// 	db := c.DB(ctx).WithContext(ctx).Model(&table)

// 	for condition, args := range conditions {
// 		switch v := args.(type) {
// 		case []interface{}:
// 			db = db.Where(condition, v...)
// 		case interface{}:
// 			db = db.Where(condition, v)
// 		}
// 	}

// 	err := db.Delete(&count).Error
// 	if err != nil {
// 		cErr := dexception.NewUnExpectedException(err)
// 		code := cErr.GetCode()
// 		message := cErr.GetMessage()
// 		return exception.NewNotFoundError(err, &code, &message)
// 	}

// 	return err
// }

// func (c *commonBehavior[T]) GetByFieldsUnscoped(ctx context.Context, conditions map[string]interface{}, preload ...string) (T, error) {
// 	var t T

// 	db := c.DB(ctx).WithContext(ctx)

// 	for condition, args := range conditions {
// 		switch v := args.(type) {
// 		case []interface{}:
// 			db = db.Where(condition, v...)
// 		case interface{}:
// 			db = db.Where(condition, v)
// 		}
// 	}

// 	for _, p := range preload {
// 		db = db.Preload(p)
// 	}

// 	err := db.Unscoped().First(&t).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			cErr := dexception.NewRecordNotFoundException(err)
// 			code := cErr.GetCode()
// 			message := cErr.GetMessage()
// 			return t, exception.NewNotFoundError(err, &code, &message)
// 		}
// 		return t, exception.NewUnExpectedError(err)
// 	}

// 	return t, err
// }

// func (c *commonBehavior[T]) UpdateUnscoped(ctx context.Context, conditions map[string]interface{}, models *T, selectQuery ...string) error {
// 	err := c.DB(ctx).WithContext(ctx).Unscoped().Select(selectQuery).Where(conditions).Updates(models).Error
// 	if err != nil {
// 		cErr := dexception.NewUnExpectedException(err)
// 		code := cErr.GetCode()
// 		message := cErr.GetMessage()
// 		return exception.NewNotFoundError(err, &code, &message)
// 	}

// 	return nil
// }
