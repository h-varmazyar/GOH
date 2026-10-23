package worker

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/worker/events"
	"github.com/h-varmazyar/gatec/internal/worker/workers"
	"github.com/h-varmazyar/gatec/pkg/tgclient"
	"github.com/sirupsen/logrus"
)

var workerNames = []string{
	"candle-polling",
	"market-sync",
	"proxy-health-checker",
	"connection-monitor",
	"indicator-calculator",
	"signal-worker",
	"stock-market-sync",
	"tse-instrument-sync",
	"tse-market-history-sync",
	// "tse-market-daily-sync",
	"tse-market-data-sync",
	"tse-market-watch",
}

type WorkerInfo struct {
	Name      string `json:"name"`
	Section   string `json:"section"`
	Running   bool   `json:"running"`
	LastError string `json:"last_error,omitempty"`
}

type workerEntry struct {
	worker    Worker
	cancel    context.CancelFunc
	running   bool
	lastError string
	done      chan struct{}
}

// WorkerFactory creates and manages worker instances.
type WorkerFactory struct {
	cfg         *configs.Workers
	bundles     *Bundles
	workers     map[string]*workerEntry
	workerMutex sync.RWMutex
	ctx         context.Context
	logger      *logrus.Entry
}

type Bundles struct {
	Repositories *RepositoriesBundle
	Adapters     *AdaptersBundle
	Services     *ServicesBundle
	Channels     *ChannelsBundle
	EventBus     events.EventBus
	TgManager    *tgclient.Manager
}

type RepositoriesBundle struct {
	Market               *repositories.MarketRepository
	Candle               *repositories.CandleRepository
	Asset                *repositories.AssetRepository
	Provider             *repositories.ProviderRepository
	Timeframe            *repositories.TimeframeRepository
	Indicator            *repositories.IndicatorDefinitionRepository
	Proxy                *repositories.ProxyRepository
	Telegram             *repositories.TelegramAccountRepository
	ProxyAssignment      *repositories.ProxyAssignmentRepository
	StockDailyData       *repositories.StockDailyDataRepository
	TSEMarket            *repositories.TSEMarketRepository
	TSEDailyClosingPrice *repositories.TSEDailyClosingPriceRepository
	TSEIndicatorSnapshot *repositories.TSEIndicatorSnapshotRepository
	StrategyDefinition   *repositories.StrategyDefinitionRepository
	Signal               *repositories.SignalRepository
}

type AdaptersBundle struct {
	CryptoProviders []crypto.Crypto
	StockProviders  []*tse.Adapter
}

type ServicesBundle struct {
	TgAccount any
}

type ChannelsBundle struct {
	CandleChan chan crypto.Candle
}

func NewWorkerFactory(cfg *configs.Workers, bundles *Bundles) *WorkerFactory {
	return &WorkerFactory{
		cfg:     cfg,
		bundles: bundles,
		workers: make(map[string]*workerEntry, len(workerNames)),
		ctx:     context.Background(),
		logger:  logrus.WithField("component", "worker-manager"),
	}
}

func SectionForWorker(name string) string { return "worker." + name }

func WorkerNames() []string {
	result := make([]string, len(workerNames))
	copy(result, workerNames)
	return result
}

// func (f *WorkerFactory) CreateAllWorkers(context.Context) error {
// 	f.workerMutex.Lock()
// 	defer f.workerMutex.Unlock()
// 	for _, name := range workerNames {
// 		if _, ok := f.workers[name]; !ok {
// 			f.workers[name] = &workerEntry{}
// 		}
// 		if f.enabledLocked(name) {
// 			w, err := f.createWorkerLocked(name)
// 			if err != nil {
// 				return err
// 			}
// 			f.workers[name].worker = w
// 		}
// 	}
// 	return nil
// }

// CreateWorker creates one named worker without changing its enabled state.
func (f *WorkerFactory) CreateWorker(name string) error {
	f.workerMutex.Lock()
	defer f.workerMutex.Unlock()
	if !f.isKnownWorkerLocked(name) {
		return fmt.Errorf("worker %q not found", name)
	}
	if _, ok := f.workers[name]; !ok {
		f.workers[name] = &workerEntry{}
	}
	if f.workers[name].worker != nil {
		return nil
	}
	created, err := f.createWorkerLocked(name)
	if err != nil {
		return err
	}
	f.workers[name].worker = created
	return nil
}

// CreateWorkers creates the requested workers in factory order. This keeps
// dependencies such as market-sync and candle-polling deterministic.
func (f *WorkerFactory) CreateWorkers(names ...string) error {
	requested := make(map[string]bool, len(names))
	for _, name := range names {
		requested[name] = true
	}
	for _, name := range workerNames {
		if requested[name] {
			if err := f.CreateWorker(name); err != nil {
				return err
			}
		}
	}
	for name := range requested {
		if !f.isKnownWorker(name) {
			return fmt.Errorf("worker %q not found", name)
		}
	}
	if requested["tse-market-watch"] && !requested["indicator-calculator"] {
		return fmt.Errorf("worker %q requires worker %q to be selected", "tse-market-watch", "indicator-calculator")
	}
	return nil
}

func (f *WorkerFactory) isKnownWorker(name string) bool {
	f.workerMutex.RLock()
	defer f.workerMutex.RUnlock()
	return f.isKnownWorkerLocked(name)
}

func (f *WorkerFactory) isKnownWorkerLocked(name string) bool {
	for _, workerName := range workerNames {
		if workerName == name {
			return true
		}
	}
	return false
}

func (f *WorkerFactory) createWorkerLocked(name string) (Worker, error) {
	switch name {
	case "candle-polling":
		return workers.NewCandlePolling(
			f.cfg.CandlePolling,
			f.bundles.Repositories.Market,
			f.bundles.Repositories.Candle,
			f.bundles.Adapters.CryptoProviders,
			f.bundles.Channels.CandleChan,
			f.bundles.EventBus,
		), nil
	// case "market-sync":
	// 	var candle *workers.CandlePolling
	// 	if entry := f.workers["candle-polling"]; entry != nil && entry.worker != nil {
	// 		candle, _ = entry.worker.(*workers.CandlePolling)
	// 	}
	// 	return workers.NewMarketSync(
	// 		f.bundles.Repositories.Market,
	// 		f.bundles.Repositories.Asset,
	// 		f.bundles.Repositories.Timeframe,
	// 		f.bundles.Adapters.CryptoProviders,
	// 		candle, f.cfg.MarketSync.SyncInterval,
	// 	), nil
	// case "proxy-health-checker":
	// 	return workers.NewProxyHealthChecker(
	// 		f.cfg.ProxyHealthChecker,
	// 		f.bundles.Repositories.Proxy,
	// 		f.bundles.TgManager,
	// 	), nil
	// case "connection-monitor":
	// 	return workers.NewConnectionMonitor(
	// 		f.cfg.ConnectionMonitor,
	// 		f.bundles.TgManager,
	// 		f.bundles.Repositories.Telegram,
	// 		f.bundles.Repositories.Proxy,
	// 		f.bundles.Repositories.ProxyAssignment,
	// 	), nil
	case "indicator-calculator":
		return workers.NewIndicatorCalculator(
			f.cfg.IndicatorCalculator,
			f.bundles.Repositories.Candle,
			f.bundles.Repositories.Indicator,
			f.bundles.Repositories.Market,
			f.bundles.EventBus,
		), nil
	case "signal-worker":
		if f.cfg.SignalWorker == nil {
			return nil, fmt.Errorf("signal-worker config is missing")
		}
		return workers.NewSignalWorker(f.cfg.SignalWorker, f.bundles.EventBus, f.bundles.Repositories.Market, f.bundles.Repositories.Provider, f.bundles.Repositories.Candle, f.bundles.Repositories.Indicator, f.bundles.Repositories.StrategyDefinition, f.bundles.Repositories.Signal), nil
	// case "stock-market-sync":
	// 	if f.cfg.StockMarketSync == nil {
	// 		return nil, fmt.Errorf("stock-market-sync config is missing")
	// 	}
	// 	return workers.NewStockMarketSync(
	// 		f.cfg.StockMarketSync,
	// 		f.bundles.Repositories.Asset,
	// 		f.bundles.Repositories.Market,
	// 		f.bundles.Repositories.StockDailyData,
	// 	), nil
	case "tse-instrument-sync":
		fmt.Println("----------------", len(f.bundles.Adapters.StockProviders))
		fmt.Println("----------------", f.cfg.TSEInstrumentSync)
		if f.cfg.TSEInstrumentSync == nil || len(f.bundles.Adapters.StockProviders) == 0 {
			return nil, fmt.Errorf("tse-instrument-sync dependencies are missing")
		}
		return workers.NewTSEInstrumentSync(
			f.cfg.TSEInstrumentSync,
			f.bundles.Adapters.StockProviders,
			f.bundles.Repositories.Market,
			f.bundles.Repositories.Asset,
			f.bundles.Repositories.TSEMarket,
		), nil
		// case "tse-market-history-sync":
		// 	if f.cfg.TSEMarketHistorySync == nil {
		// 		return nil, fmt.Errorf("tse-market-history-sync config is missing")
		// 	}
		// 	if len(f.bundles.Adapters.StockProviders) == 0 {
		// 		return nil, fmt.Errorf("tse-market-history-sync adapter is missing")
		// 	}
		// 	mode := workers.TSEMarketDataMode(f.cfg.TSEMarketHistorySync.Mode)
		// 	if mode == "" {
		// 		mode = workers.TSEMarketDataAll
		// 	}
		// 	parseDate := func(value string) time.Time { parsed, _ := time.Parse("2006-01-02", value); return parsed }
		// 	cfg := workers.TSEMarketDataConfig{Mode: mode, Date: parseDate(f.cfg.TSEMarketHistorySync.Date), StartDate: parseDate(f.cfg.TSEMarketHistorySync.StartDate), EndDate: parseDate(f.cfg.TSEMarketHistorySync.EndDate), RequestInterval: f.cfg.TSEMarketHistorySync.RequestInterval}
		// 	return workers.NewTSEMarketDataSync(f.bundles.Adapters.StockProviders[0], cfg, f.bundles.Repositories.TSEMarket, f.bundles.Repositories.TSEDailyClosingPrice), nil
		// case "tse-market-daily-sync":
		// 	if f.cfg.TSEMarketDailySync == nil {
		// 		return nil, fmt.Errorf("tse-market-daily-sync config is missing")
		// 	}
		// 	cfg := f.cfg.TSEMarketDailySync
		// 	if cfg.BotToken == "" {
		// 		return nil, fmt.Errorf("tse-market-daily-sync telegram bot token is missing")
		// 	}
		// 	if len(f.bundles.Adapters.StockProviders) == 0 {
		// 		return nil, fmt.Errorf("tse-market-daily-sync adapter is missing")
		// 	}
		// 	dataCfg := workers.TSEMarketDataConfig{Mode: workers.TSEMarketDataToday, RequestInterval: cfg.RequestInterval}
		// 	return workers.NewTSEMarketDataSync(f.bundles.Adapters.StockProviders[0], dataCfg, f.bundles.Repositories.TSEMarket, f.bundles.Repositories.TSEDailyClosingPrice), nil
	case "tse-market-data-sync":
		if f.cfg.TSEMarketDataSync == nil || len(f.bundles.Adapters.StockProviders) == 0 {
			return nil, fmt.Errorf("tse-market-data-sync dependencies are missing")
		}
		return workers.NewTSEMarketDataSyncWithBus(
			f.bundles.Adapters.StockProviders[0],
			f.cfg.TSEMarketDataSync,
			f.bundles.Repositories.TSEMarket,
			f.bundles.Repositories.TSEDailyClosingPrice,
			f.bundles.EventBus,
		), nil
	case "tse-market-watch":
		if f.cfg.TSEMarketWatch == nil || len(f.bundles.Adapters.StockProviders) == 0 {
			return nil, fmt.Errorf("tse-market-watch dependencies are missing. %t, %t", f.cfg.TSEMarketWatch == nil, len(f.bundles.Adapters.StockProviders) == 0)
		}
		return workers.NewTSEMarketWatch(
			f.cfg.TSEMarketWatch,
			f.bundles.Adapters.StockProviders[0],
			f.bundles.Repositories.TSEMarket,
			f.bundles.EventBus,
		), nil
	default:
		return nil, fmt.Errorf("worker %q not found", name)
	}
}

// func (f *WorkerFactory) enabledLocked(name string) bool {
// 	switch name {
// 	case "candle-polling":
// 		return f.cfg.CandlePollingEnabled
// 	case "market-sync":
// 		return f.cfg.MarketSyncEnabled
// 	case "proxy-health-checker":
// 		return f.cfg.ProxyHealthCheckerEnabled
// 	case "connection-monitor":
// 		return f.cfg.ConnectionMonitorEnabled
// 	case "indicator-calculator":
// 		return f.cfg.IndicatorCalculatorEnabled
// 	case "stock-market-sync":
// 		return f.cfg.StockMarketSyncEnabled
// 	case "stock-instrument-sync":
// 		return f.cfg.StockInstrumentSyncEnabled
// 	case "tse-market-data-sync":
// 		return f.cfg.TSEMarketDataSyncEnabled
// 	case "tse-market-history-sync":
// 		return f.cfg.TSEMarketHistorySyncEnabled
// 	// case "tse-market-daily-sync":
// 	// 	return f.cfg.TSEMarketDailySyncEnabled
// 	case "tse-market-watch":
// 		return f.cfg.TSEMarketWatchEnabled
// 	default:
// 		return false
// 	}
// }

// func (f *WorkerFactory) setEnabledLocked(name string, enabled bool) error {
// 	switch name {
// 	case "candle-polling":
// 		f.cfg.CandlePollingEnabled = enabled
// 	case "market-sync":
// 		f.cfg.MarketSyncEnabled = enabled
// 	case "proxy-health-checker":
// 		f.cfg.ProxyHealthCheckerEnabled = enabled
// 	case "connection-monitor":
// 		f.cfg.ConnectionMonitorEnabled = enabled
// 	case "indicator-calculator":
// 		f.cfg.IndicatorCalculatorEnabled = enabled
// 	case "stock-market-sync":
// 		f.cfg.StockMarketSyncEnabled = enabled
// 	case "stock-instrument-sync":
// 		f.cfg.StockInstrumentSyncEnabled = enabled
// 	case "tse-market-history-sync":
// 		f.cfg.TSEMarketHistorySyncEnabled = enabled
// 	// case "tse-market-daily-sync":
// 	// 	f.cfg.TSEMarketDailySyncEnabled = enabled
// 	case "tse-market-watch":
// 		f.cfg.TSEMarketWatchEnabled = enabled
// 	default:
// 		return fmt.Errorf("worker %q not found", name)
// 	}
// 	return nil
// }

// func (f *WorkerFactory) StartAllWorkers(ctx context.Context) error {
// 	for _, name := range workerNames {
// 		f.workerMutex.RLock()
// 		enabled := f.enabledLocked(name)
// 		f.workerMutex.RUnlock()
// 		if enabled {
// 			if err := f.StartWorker(ctx, name); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

func (f *WorkerFactory) StartWorker(parentCtx context.Context, name string) error {
	f.workerMutex.Lock()
	entry, exists := f.workers[name]
	if !exists {
		f.workerMutex.Unlock()
		return fmt.Errorf("worker %q not found", name)
	}
	if entry.running {
		f.workerMutex.Unlock()
		return nil
	}
	if entry.worker == nil {
		w, err := f.createWorkerLocked(name)
		if err != nil {
			f.workerMutex.Unlock()
			return err
		}
		entry.worker = w
	}
	if parentCtx == nil {
		parentCtx = f.ctx
	}
	ctx, cancel := context.WithCancel(parentCtx)
	entry.cancel, entry.running, entry.lastError = cancel, true, ""
	entry.done = make(chan struct{})
	done := entry.done
	w := entry.worker
	f.workerMutex.Unlock()

	go func() {
		defer close(done)
		err := w.Start(ctx)
		f.workerMutex.Lock()
		defer f.workerMutex.Unlock()
		current := f.workers[name]
		if current.worker == w {
			current.running = false
			if err != nil {
				current.lastError = err.Error()
			}
		}
	}()
	return nil
}

func (f *WorkerFactory) WaitWorker(ctx context.Context, name string) error {
	f.workerMutex.RLock()
	entry, exists := f.workers[name]
	if !exists || entry.done == nil {
		f.workerMutex.RUnlock()
		return fmt.Errorf("worker %q is not running", name)
	}
	done := entry.done
	f.workerMutex.RUnlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		f.workerMutex.RLock()
		defer f.workerMutex.RUnlock()
		if current := f.workers[name]; current != nil && current.lastError != "" {
			return fmt.Errorf("worker %q: %s", name, current.lastError)
		}
		return nil
	}
}

func (f *WorkerFactory) StopWorker(name string) error {
	f.workerMutex.Lock()
	entry, exists := f.workers[name]
	if !exists {
		f.workerMutex.Unlock()
		return fmt.Errorf("worker %q not found", name)
	}
	if entry.cancel != nil {
		entry.cancel()
	}
	entry.running = false
	w := entry.worker
	f.workerMutex.Unlock()
	if w != nil {
		return w.Stop()
	}
	return nil
}

// func (f *WorkerFactory) SetEnabled(ctx context.Context, name string, enabled bool) error {
// 	f.workerMutex.Lock()
// 	if err := f.setEnabledLocked(name, enabled); err != nil {
// 		f.workerMutex.Unlock()
// 		return err
// 	}
// 	f.workerMutex.Unlock()
// 	if enabled {
// 		return f.StartWorker(ctx, name)
// 	}
// 	return f.StopWorker(name)
// }

func (f *WorkerFactory) RestartWorker(ctx context.Context, name string) error {
	f.workerMutex.RLock()
	entry, exists := f.workers[name]
	// enabled := exists && f.enabledLocked(name)
	f.workerMutex.RUnlock()
	if !exists {
		return fmt.Errorf("worker %q not found", name)
	}
	if err := f.StopWorker(name); err != nil {
		return err
	}
	f.workerMutex.Lock()
	entry.worker = nil
	f.workerMutex.Unlock()
	// if enabled {
	return f.StartWorker(ctx, name)
	// }
}

func (f *WorkerFactory) StopAllWorkers() error {
	var firstErr error
	f.workerMutex.RLock()
	names := make([]string, 0, len(f.workers))
	for name, entry := range f.workers {
		if entry != nil && entry.running {
			names = append(names, name)
		}
	}
	f.workerMutex.RUnlock()
	sort.Strings(names)
	for _, name := range names {
		if err := f.StopWorker(name); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (f *WorkerFactory) GetWorker(name string) (Worker, bool) {
	f.workerMutex.RLock()
	defer f.workerMutex.RUnlock()
	entry, exists := f.workers[name]
	if !exists || entry.worker == nil {
		return nil, false
	}
	return entry.worker, true
}

func (f *WorkerFactory) ListWorkers() []string {
	return WorkerNames()
}

func (f *WorkerFactory) ListWorkerInfo() []WorkerInfo {
	f.workerMutex.RLock()
	defer f.workerMutex.RUnlock()
	result := make([]WorkerInfo, 0, len(workerNames))
	for _, name := range workerNames {
		entry := f.workers[name]
		info := WorkerInfo{
			Name:    name,
			Section: SectionForWorker(name),
		}
		if entry != nil {
			info.Running = entry.running
			info.LastError = entry.lastError
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (f *WorkerFactory) WorkerStatus(name string) string {
	f.workerMutex.RLock()
	defer f.workerMutex.RUnlock()
	entry, exists := f.workers[name]
	if !exists {
		return "not_found"
	}
	if entry.running {
		return "running"
	}
	return "stopped"
}

func (f *WorkerFactory) Config() *configs.Workers { return f.cfg }
