package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ScheduledRunRepository records the runs of daily schedules.
type ScheduledRunRepository struct{ db *gorm.DB }

func NewScheduledRunRepository(db *gorm.DB) *ScheduledRunRepository {
	return &ScheduledRunRepository{db: db}
}

// ClaimScheduledRun records the run of schedule name on date. It reports false
// when the run was already made, by this process or another.
func (r *ScheduledRunRepository) ClaimScheduledRun(ctx context.Context, name string, date time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&models.ScheduledRun{Name: name, RunDate: date})
	return result.RowsAffected == 1, result.Error
}

// FinishScheduledRun stores what a run queued.
func (r *ScheduledRunRepository) FinishScheduledRun(ctx context.Context, name string, date time.Time, detail string) error {
	return r.db.WithContext(ctx).Model(&models.ScheduledRun{}).
		Where("name = ? AND run_date = ?", name, date).Update("detail", detail).Error
}

// ReleaseScheduledRun forgets a run that failed, so it is made again.
func (r *ScheduledRunRepository) ReleaseScheduledRun(ctx context.Context, name string, date time.Time) error {
	err := r.db.WithContext(ctx).Where("name = ? AND run_date = ?", name, date).Delete(&models.ScheduledRun{}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}
