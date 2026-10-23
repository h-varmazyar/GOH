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

// ProxyHealthChecker periodically checks the connectivity of all proxies
// and updates their status in the database. It also removes proxies that
// have been failing for a long time.
type ProxyHealthChecker struct {
	proxyRepo     *repositories.ProxyRepository
	tgManager     *tgclient.Manager
	checkInterval time.Duration
	staleDuration time.Duration
	proxyTimeout  time.Duration
	cancel        context.CancelFunc
}

func NewProxyHealthChecker(
	cfg *configs.ProxyHealthChecker,
	proxyRepo *repositories.ProxyRepository,
	tgManager *tgclient.Manager,
) *ProxyHealthChecker {
	checkInterval := time.Minute
	staleDuration := 24 * time.Hour
	proxyTimeout := 10 * time.Second

	if cfg != nil {
		if cfg.CheckInterval > 0 {
			checkInterval = cfg.CheckInterval
		}
		if cfg.StaleDuration > 0 {
			staleDuration = cfg.StaleDuration
		}
		if cfg.ProxyTimeout > 0 {
			proxyTimeout = cfg.ProxyTimeout
		}
	}

	return &ProxyHealthChecker{
		proxyRepo:     proxyRepo,
		tgManager:     tgManager,
		checkInterval: checkInterval,
		staleDuration: staleDuration,
		proxyTimeout:  proxyTimeout,
	}
}

func (w *ProxyHealthChecker) Name() string {
	return "proxy-health-checker"
}

func (w *ProxyHealthChecker) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	log := logrus.WithField("worker", w.Name())
	log.Info("starting proxy health checker worker")

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.checkAllProxies(ctx)
	// w.cleanupStaleProxies(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info("proxy health checker stopped")
			return nil
		case <-ticker.C:
			w.checkAllProxies(ctx)
			// w.cleanupStaleProxies(ctx)
		}
	}
}

func (w *ProxyHealthChecker) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// checkAllProxies tests each enabled proxy and updates its status
func (w *ProxyHealthChecker) checkAllProxies(ctx context.Context) {
	log := logrus.WithField("worker", w.Name())

	proxies, err := w.proxyRepo.ListEnabled(ctx)
	if err != nil {
		log.WithError(err).Error("failed to list enabled proxies")
		return
	}

	if len(proxies) == 0 {
		return
	}

	log.Infof("checking %d proxies", len(proxies))

	for _, proxy := range proxies {
		select {
		case <-ctx.Done():
			return
		default:
		}

		proxyConfig := modelToProxyConfig(&proxy)
		err := w.tgManager.TestProxyStandalone(proxyConfig, w.proxyTimeout)
		if err != nil {
			log.WithFields(logrus.Fields{
				"proxy_id": proxy.ID,
				"server":   proxy.Server,
				"port":     proxy.Port,
			}).WithError(err).Warn("proxy check failed")
			_ = w.proxyRepo.UpdateStatus(ctx, proxy.ID, models.ProxyStatusInactive, false)
		} else {
			log.WithFields(logrus.Fields{
				"proxy_id": proxy.ID,
				"server":   proxy.Server,
				"port":     proxy.Port,
			}).Debug("proxy check succeeded")
			_ = w.proxyRepo.UpdateStatus(ctx, proxy.ID, models.ProxyStatusActive, true)
		}
	}
}

// cleanupStaleProxies removes proxies that haven't worked for a long time
func (w *ProxyHealthChecker) cleanupStaleProxies(ctx context.Context) {
	log := logrus.WithField("worker", w.Name())

	deleted, err := w.proxyRepo.DeleteStaleProxies(ctx, w.staleDuration)
	if err != nil {
		log.WithError(err).Error("failed to cleanup stale proxies")
		return
	}

	if deleted > 0 {
		log.Infof("deleted %d stale proxies (no success for %v)", deleted, w.staleDuration)
	}
}

// modelToProxyConfig converts a database Proxy model to tgclient.ProxyConfig
func modelToProxyConfig(proxy *models.Proxy) *tgclient.ProxyConfig {
	// Log for debugging
	logrus.WithFields(logrus.Fields{
		"proxy_id":       proxy.ID,
		"db_type":        proxy.Type,
		"converted_type": tgclient.ProxyType(proxy.Type),
		"server":         proxy.Server,
		"port":           proxy.Port,
	}).Debug("converting proxy model to config")

	return &tgclient.ProxyConfig{
		Server:   proxy.Server,
		Port:     proxy.Port,
		Type:     tgclient.ProxyType(proxy.Type),
		Secret:   proxy.Secret,
		Username: proxy.Username,
		Password: proxy.Password,
		HTTPOnly: proxy.HTTPOnly,
	}
}
