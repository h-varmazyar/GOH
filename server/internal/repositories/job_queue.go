package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/pkg/pgnotify"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// JobQueue holds the queue operations shared by the job tables, whose rows are
// claimed and run by job runners; see the jobrunner package. T is the job
// model; its table has the columns id, status, worker, heartbeat_at,
// started_at, finished_at, cancel_requested, error and updated_at, and its
// statuses are the consts.JobStatus values.
//
// Every change that may make a job runnable, such as queueing a job, is
// announced on Channel with a Postgres NOTIFY, so idle runners wake at once
// instead of polling the table.
type JobQueue[T any] struct {
	db      *gorm.DB
	Channel string
	// runnable narrows the pending jobs that can be claimed; nil claims any.
	runnable func(tx *gorm.DB) *gorm.DB
}

func NewJobQueue[T any](db *gorm.DB, channel string, runnable func(tx *gorm.DB) *gorm.DB) *JobQueue[T] {
	return &JobQueue[T]{db: db, Channel: channel, runnable: runnable}
}

func (q *JobQueue[T]) model() *T { return new(T) }

// Create stores a queued job and wakes the runners once it is committed.
func (q *JobQueue[T]) Create(ctx context.Context, job *T) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(job).Error; err != nil {
			return err
		}
		return pgnotify.Notify(tx, q.Channel)
	})
}

func (q *JobQueue[T]) Get(ctx context.Context, id uint) (*T, error) {
	job := q.model()
	return job, q.db.WithContext(ctx).First(job, id).Error
}

// Notify wakes the runners of the queue.
func (q *JobQueue[T]) Notify(ctx context.Context) error {
	return pgnotify.Notify(q.db.WithContext(ctx), q.Channel)
}

// Claim marks the oldest runnable pending job as running by worker and returns
// it, or nil when none is pending. Workers never claim the same job.
func (q *JobQueue[T]) Claim(ctx context.Context, worker string) (*T, error) {
	var claimed *T
	err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job := q.model()
		query := tx.Model(job).Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", consts.JobPending)
		if q.runnable != nil {
			query = q.runnable(query)
		}
		found := query.Order("id ASC").Limit(1).Find(job)
		if found.Error != nil || found.RowsAffected == 0 {
			return found.Error
		}
		now := time.Now()
		columns := map[string]any{
			"status": consts.JobRunning, "worker": worker, "heartbeat_at": now, "updated_at": now, "error": "",
			"started_at": gorm.Expr("COALESCE(started_at, ?)", now),
		}
		if err := tx.Model(job).UpdateColumns(columns).Error; err != nil {
			return err
		}
		claimed = job
		return nil
	})
	return claimed, err
}

// Cancel cancels a pending job, or asks the worker of a running job to stop
// it. It reports whether the job was active.
func (q *JobQueue[T]) Cancel(ctx context.Context, id uint) (bool, error) {
	now := time.Now()
	result := q.db.WithContext(ctx).Model(q.model()).
		Where("id = ? AND status = ?", id, consts.JobPending).
		UpdateColumns(map[string]any{"status": consts.JobCanceled, "finished_at": now, "updated_at": now})
	if result.Error != nil || result.RowsAffected > 0 {
		return result.RowsAffected > 0, result.Error
	}
	result = q.db.WithContext(ctx).Model(q.model()).
		Where("id = ? AND status = ?", id, consts.JobRunning).
		UpdateColumns(map[string]any{"cancel_requested": true, "updated_at": now})
	return result.RowsAffected > 0, result.Error
}

// RequeueStale queues again the running jobs without a heartbeat since before
// and wakes the runners when it queued any.
func (q *JobQueue[T]) RequeueStale(ctx context.Context, before time.Time) (int64, error) {
	result := q.db.WithContext(ctx).Model(q.model()).
		Where("status = ? AND (heartbeat_at IS NULL OR heartbeat_at < ?)", consts.JobRunning, before).
		UpdateColumns(map[string]any{"status": consts.JobPending, "updated_at": time.Now()})
	if result.Error != nil || result.RowsAffected == 0 {
		return result.RowsAffected, result.Error
	}
	return result.RowsAffected, q.Notify(ctx)
}

// OldestRunningHeartbeat returns the oldest heartbeat of the running jobs; nil
// when no job is running.
func (q *JobQueue[T]) OldestRunningHeartbeat(ctx context.Context) (*time.Time, error) {
	var oldest sql.NullTime
	err := q.db.WithContext(ctx).Model(q.model()).
		Where("status = ?", consts.JobRunning).
		Select("MIN(heartbeat_at)").Scan(&oldest).Error
	if err != nil || !oldest.Valid {
		return nil, err
	}
	return &oldest.Time, nil
}

// UpdateOwned updates a job only while worker runs it. owned is false when the
// job was queued again or claimed by another worker; cancelRequested reports a
// cancel request of the panel.
func (q *JobQueue[T]) UpdateOwned(ctx context.Context, id uint, worker string, columns map[string]any) (owned, cancelRequested bool, err error) {
	columns["updated_at"] = time.Now()
	err = q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		update := tx.Model(q.model()).
			Where("id = ? AND worker = ? AND status = ?", id, worker, consts.JobRunning).
			UpdateColumns(columns)
		if update.Error != nil || update.RowsAffected == 0 {
			return update.Error
		}
		owned = true
		// The row is locked by the update, so the flag is read as it was updated.
		return tx.Model(q.model()).Where("id = ?", id).Select("cancel_requested").Scan(&cancelRequested).Error
	})
	if err != nil {
		return false, false, err
	}
	return owned, cancelRequested, nil
}
