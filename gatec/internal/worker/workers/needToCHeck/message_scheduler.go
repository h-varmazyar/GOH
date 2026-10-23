package workers

import (
	"context"
	"time"

	messagingService "github.com/h-varmazyar/gatec/internal/services/messaging"
	"github.com/sirupsen/logrus"
)

// MessageSchedulerConfig holds configuration for the message scheduler worker.
type MessageSchedulerConfig struct {
	// CheckInterval is how often to check for due scheduled messages.
	CheckInterval time.Duration
}

// MessageScheduler is a worker that processes scheduled messages.
type MessageScheduler struct {
	cfg     *MessageSchedulerConfig
	service *messagingService.Service
	logger  *logrus.Entry
}

// NewMessageScheduler creates a new MessageScheduler worker.
func NewMessageScheduler(cfg *MessageSchedulerConfig, service *messagingService.Service) *MessageScheduler {
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 30 * time.Second
	}

	return &MessageScheduler{
		cfg:     cfg,
		service: service,
		logger:  logrus.WithField("worker", "message-scheduler"),
	}
}

// Start begins the periodic check for due scheduled messages.
func (w *MessageScheduler) Start(ctx context.Context) error {
	w.logger.Info("message scheduler worker started")

	ticker := time.NewTicker(w.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("message scheduler worker stopping")
			return nil
		case <-ticker.C:
			w.processDueMessages(ctx)
		}
	}
}

// Stop gracefully stops the worker.
func (w *MessageScheduler) Stop() error {
	w.logger.Info("message scheduler worker stopped")
	return nil
}

func (w *MessageScheduler) processDueMessages(ctx context.Context) {
	processed, err := w.service.ProcessDueScheduledMessages(ctx)
	if err != nil {
		w.logger.WithError(err).Error("failed to process scheduled messages")
		return
	}

	if processed > 0 {
		w.logger.WithField("count", processed).Info("processed scheduled messages")
	}
}
