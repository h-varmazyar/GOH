package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
)

type TabdealAirdropRepository struct {
	db *gorm.DB
}

func NewWallexAirdropRepository(db *gorm.DB) *TabdealAirdropRepository {
	return &TabdealAirdropRepository{db: db}
}

func (r *TabdealAirdropRepository) UpsertRoute(ctx context.Context, route *models.TabdealAirdrop) error {
	var existing models.TabdealAirdrop
	err := r.db.WithContext(ctx).Where("phrase = ?", route.Phrase).First(&existing).Error
	if err == nil {
		existing.Route = route.Route
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return r.db.WithContext(ctx).Create(route).Error
}

func (r *TabdealAirdropRepository) FindRouteByPhrase(ctx context.Context, phrase string) (*models.TabdealAirdrop, error) {
	var route models.TabdealAirdrop
	if err := r.db.WithContext(ctx).Where("phrase = ?", phrase).First(&route).Error; err != nil {
		return nil, err
	}
	return &route, nil
}

func (r *TabdealAirdropRepository) ListRoutes(ctx context.Context, pg paginator.Params) ([]models.TabdealAirdrop, int64, error) {
	var routes []models.TabdealAirdrop
	var total int64

	r.db.WithContext(ctx).Model(&models.TabdealAirdrop{}).Count(&total)

	err := pg.Apply(r.db.WithContext(ctx).Order("created_at desc")).Find(&routes).Error
	return routes, total, err
}

func (r *TabdealAirdropRepository) CreateGift(ctx context.Context, gift *models.TabdealAirdropGift) error {
	return r.db.WithContext(ctx).Create(gift).Error
}

func (r *TabdealAirdropRepository) UpdateGift(ctx context.Context, gift *models.TabdealAirdropGift) error {
	return r.db.WithContext(ctx).Save(gift).Error
}

func (r *TabdealAirdropRepository) GetGift(ctx context.Context, id uint) (*models.TabdealAirdropGift, error) {
	var gift models.TabdealAirdropGift
	if err := r.db.WithContext(ctx).First(&gift, id).Error; err != nil {
		return nil, err
	}
	return &gift, nil
}

func (r *TabdealAirdropRepository) ListGifts(ctx context.Context, pg paginator.Params) ([]models.TabdealAirdropGift, int64, error) {
	var gifts []models.TabdealAirdropGift
	var total int64

	r.db.WithContext(ctx).Model(&models.TabdealAirdropGift{}).Count(&total)

	err := pg.Apply(r.db.WithContext(ctx).Order("created_at desc")).Find(&gifts).Error
	return gifts, total, err
}
