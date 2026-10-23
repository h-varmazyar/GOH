package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByMobile(ctx context.Context, mobile string) (*models.User, error) {
	user := new(models.User)
	return user, r.db.WithContext(ctx).Model(&models.User{}).Where("mobile = ?", mobile).First(user).Error
}

func (r *UserRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	user := new(models.User)
	return user, r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).First(user).Error
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.Create(user).Error
}
