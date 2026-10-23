package workers

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/tgclient"
	"github.com/sirupsen/logrus"
)

// ConnectionMonitor periodically checks the connection status of each
// active telegram client session. If a session is disconnected, it rotates
// to a different proxy and attempts reconnection.
type ConnectionMonitor struct {
	tgManager      *tgclient.Manager
	accountRepo    *repositories.TelegramAccountRepository
	proxyRepo      *repositories.ProxyRepository
	assignmentRepo *repositories.ProxyAssignmentRepository
	checkInterval  time.Duration
	cancel         context.CancelFunc
}

func NewConnectionMonitor(
	cfg *configs.ConnectionMonitor,
	tgManager *tgclient.Manager,
	accountRepo *repositories.TelegramAccountRepository,
	proxyRepo *repositories.ProxyRepository,
	assignmentRepo *repositories.ProxyAssignmentRepository,
) *ConnectionMonitor {
	checkInterval := 30 * time.Second
	if cfg != nil && cfg.CheckInterval > 0 {
		checkInterval = cfg.CheckInterval
	}

	return &ConnectionMonitor{
		tgManager:      tgManager,
		accountRepo:    accountRepo,
		proxyRepo:      proxyRepo,
		assignmentRepo: assignmentRepo,
		checkInterval:  checkInterval,
	}
}

func (w *ConnectionMonitor) Name() string {
	return "connection-monitor"
}

func (w *ConnectionMonitor) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	log := logrus.WithField("worker", w.Name())
	log.Info("starting connection monitor worker")

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("connection monitor stopped")
			return nil
		case <-ticker.C:
			w.checkConnections(ctx)
		}
	}
}

func (w *ConnectionMonitor) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// checkConnections iterates over all ready accounts and checks their connection
func (w *ConnectionMonitor) checkConnections(ctx context.Context) {
	log := logrus.WithField("worker", w.Name())

	accounts, err := w.accountRepo.ListAll(ctx)
	if err != nil {
		log.WithError(err).Error("failed to list accounts")
		return
	}

	for _, account := range accounts {
		if account.Status != models.TelegramAccountStatusReady {
			continue
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		sessionID := account.ID.String()
		err := w.tgManager.CheckConnection(sessionID)
		if err != nil {
			log.WithFields(logrus.Fields{
				"account_id": sessionID,
			}).WithError(err).Warn("connection check failed, rotating proxy")

			w.rotateProxy(ctx, sessionID)
		}
	}
}

// rotateProxy assigns a new proxy to the account after the current one fails
func (w *ConnectionMonitor) rotateProxy(ctx context.Context, accountID string) {
	log := logrus.WithFields(logrus.Fields{
		"worker":     w.Name(),
		"account_id": accountID,
	})

	// Find current assignment to exclude its proxy
	excludeIDs := make([]uint, 0)
	currentAssignment, err := w.assignmentRepo.GetActiveAssignment(ctx, accountID)
	if err == nil && currentAssignment != nil {
		excludeIDs = append(excludeIDs, currentAssignment.ProxyID)
		// Decrement old proxy's assigned count
		_ = w.proxyRepo.DecrementAssignedAccounts(ctx, currentAssignment.ProxyID)
	}

	// Get a new proxy
	newProxy, err := w.proxyRepo.GetLeastUsedActiveProxy(ctx, excludeIDs)
	if err != nil {
		log.WithError(err).Error("no active proxy available for rotation")
		// Update account status to disconnected since we can't assign a proxy
		_ = w.accountRepo.UpdateStatus(ctx, accountID, models.TelegramAccountStatusDisconnected)
		return
	}

	// Assign new proxy
	if err := w.assignmentRepo.Assign(ctx, accountID, newProxy.ID); err != nil {
		log.WithError(err).Error("failed to assign new proxy")
		return
	}
	_ = w.proxyRepo.IncrementAssignedAccounts(ctx, newProxy.ID)

	// Apply the proxy to the TDLib session
	proxyConfig := modelToProxyConfig(newProxy)
	if err := w.tgManager.SetProxy(accountID, proxyConfig); err != nil {
		log.WithError(err).Error("failed to set proxy on session")
		return
	}

	log.WithFields(logrus.Fields{
		"new_proxy_id": newProxy.ID,
		"server":       newProxy.Server,
		"port":         newProxy.Port,
	}).Info("proxy rotated successfully")
}

// AssignProxyToSession assigns the best available proxy to a newly created session.
// This should be called when a session is first created to ensure it uses a proxy.
func (w *ConnectionMonitor) AssignProxyToSession(ctx context.Context, accountID string) error {
	log := logrus.WithFields(logrus.Fields{
		"worker":     w.Name(),
		"account_id": accountID,
	})

	proxy, err := w.proxyRepo.GetLeastUsedActiveProxy(ctx, nil)
	if err != nil {
		log.WithError(err).Warn("no active proxy available for new session")
		return err
	}

	// Create the assignment
	if err := w.assignmentRepo.Assign(ctx, accountID, proxy.ID); err != nil {
		return err
	}
	_ = w.proxyRepo.IncrementAssignedAccounts(ctx, proxy.ID)

	// Apply proxy to session
	proxyConfig := modelToProxyConfig(proxy)
	if err := w.tgManager.SetProxy(accountID, proxyConfig); err != nil {
		log.WithError(err).Warn("failed to apply proxy to session (may not be ready yet)")
		return err
	}

	log.WithFields(logrus.Fields{
		"proxy_id": proxy.ID,
		"server":   proxy.Server,
		"port":     proxy.Port,
	}).Info("proxy assigned to session")

	return nil
}
