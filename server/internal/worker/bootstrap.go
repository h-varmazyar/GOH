package worker

import (
	"context"
	"fmt"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto/nobitex"
	stockTSE "github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	tgAccountService "github.com/h-varmazyar/gatec/internal/services/tgaccount"
	"github.com/h-varmazyar/gatec/internal/worker/events"
	clickhouseClient "github.com/h-varmazyar/gatec/pkg/clickhouse"
	gormext "github.com/h-varmazyar/gatec/pkg/gormOld"
	"github.com/h-varmazyar/gatec/pkg/tgclient"
	"gorm.io/gorm"
)

type dependencyPlan struct {
	provider             bool
	market               bool
	asset                bool
	timeframe            bool
	candle               bool
	indicator            bool
	proxy                bool
	telegram             bool
	proxyAssignment      bool
	stockDailyData       bool
	tseMarket            bool
	tseDailyClosingPrice bool
	tseTradeHistory      bool
	signalWorker         bool
	cryptoProviders      bool
	stockProviders       bool
}

func dependencyNeeds(names []string, cfg *configs.Config) dependencyPlan {
	var needs dependencyPlan
	for _, name := range names {
		switch name {
		case "candle-polling":
			needs.market = true
			needs.candle = true
			needs.cryptoProviders = true
		case "market-sync":
			needs.market = true
			needs.asset = true
			needs.timeframe = true
			needs.cryptoProviders = true
		case "proxy-health-checker":
			needs.proxy = true
			needs.telegram = true
		case "connection-monitor":
			needs.telegram = true
			needs.proxy = true
			needs.proxyAssignment = true
		case "indicator-calculator":
			needs.candle = true
			needs.indicator = true
			needs.market = true
		case "signal-worker":
			needs.signalWorker = true
			needs.provider = true
			needs.market = true
			needs.candle = true
			needs.timeframe = true
			needs.indicator = true
		case "stock-market-sync":
			needs.asset = true
			needs.market = true
			needs.stockDailyData = true
		case "tse-instrument-sync":
			needs.market = true
			needs.asset = true
			needs.tseMarket = true
			needs.stockProviders = true
		case "tse-market-history-sync":
			needs.market = true
			needs.candle = true
			needs.tseMarket = true
			needs.tseDailyClosingPrice = true
			needs.stockProviders = true
		case "tse-market-data-sync":
			needs.market = true
			needs.candle = true
			needs.tseMarket = true
			needs.tseDailyClosingPrice = true
			needs.stockProviders = true
			if cfg == nil ||
				cfg.Workers.TSEMarketDataSync == nil ||
				cfg.Workers.TSEMarketDataSync.Mode == "" ||
				cfg.Workers.TSEMarketDataSync.Mode == consts.TSEMarketDataToday {
				needs.tseTradeHistory = true
			}
		case "tse-trade-history":
			needs.tseMarket = true
			needs.stockProviders = true
			needs.tseTradeHistory = true
		case "tse-market-watch":
			needs.market = true
			needs.tseMarket = true
			needs.stockProviders = true
		}
	}
	needs.provider = needs.provider || needs.cryptoProviders || needs.stockProviders
	return needs
}

// StandaloneFactory opens the dependencies required by worker processes and
// returns a factory that can create only the workers selected by the caller.
func StandaloneFactory(ctx context.Context, cfg *configs.Config, ch *clickhouseClient.Client, names ...string) (*WorkerFactory, func(), error) {
	db, err := gormext.Open(cfg.Postgres)
	if err != nil {
		return nil, nil, err
	}
	closeDB := func() {
		sqlDB, closeErr := db.DB()
		if closeErr == nil {
			_ = sqlDB.Close()
		}
	}

	needs := dependencyNeeds(names, cfg)
	if err := migrateWorkerModels(db, needs); err != nil {
		closeDB()
		return nil, nil, fmt.Errorf("worker migration failed: %w", err)
	}
	var eventBus events.EventBus = events.NewChannelBus()
	if cfg.NATS.URL != "" {
		eventBus, err = events.NewNATSBus(cfg.NATS)
		if err != nil {
			closeDB()
			return nil, nil, err
		}
	}
	closeDependencies := func() {
		_ = eventBus.Close()
		closeDB()
	}

	candleChan := make(chan crypto.Candle, 1000)
	var providerRepo *repositories.ProviderRepository
	if needs.provider || needs.cryptoProviders || needs.stockProviders {
		providerRepo = repositories.NewProviderRepository(db)
	}
	var marketRepo *repositories.MarketRepository
	if needs.market {
		marketRepo = repositories.NewMarketRepository(db)
	}
	var assetRepo *repositories.AssetRepository
	if needs.asset {
		assetRepo = repositories.NewAssetRepository(db)
	}
	var timeframeRepo *repositories.TimeframeRepository
	if needs.timeframe {
		timeframeRepo = repositories.NewTimeframeRepository(db)
	}
	var candleRepo *repositories.CandleRepository
	if needs.candle {
		candleRepo = repositories.NewCandleRepository(db)
	}
	var indicatorRepo *repositories.IndicatorDefinitionRepository
	if needs.indicator {
		indicatorRepo = repositories.NewIndicatorDefinitionRepository(db)
	}
	var strategyDefinitionRepo *repositories.StrategyDefinitionRepository
	var signalRepo *repositories.SignalRepository
	if needs.signalWorker {
		strategyDefinitionRepo = repositories.NewStrategyDefinitionRepository(db)
		signalRepo = repositories.NewSignalRepository(db)
	}
	var proxyRepo *repositories.ProxyRepository
	if needs.proxy {
		proxyRepo = repositories.NewProxyRepository(db)
	}
	var telegramRepo *repositories.TelegramAccountRepository
	var filterRepo *repositories.TelegramSubscriptionFilterRepository
	if needs.telegram {
		telegramRepo = repositories.NewTelegramAccountRepository(db)
		filterRepo = repositories.NewTelegramSubscriptionFilterRepository(db)
	}
	var proxyAssignmentRepo *repositories.ProxyAssignmentRepository
	if needs.proxyAssignment {
		proxyAssignmentRepo = repositories.NewProxyAssignmentRepository(db)
	}
	var stockDailyRepo *repositories.StockDailyDataRepository
	if needs.stockDailyData {
		stockDailyRepo = repositories.NewStockDailyDataRepository(db)
	}
	var tseMarketRepo *repositories.TSEMarketRepository
	if needs.tseMarket {
		tseMarketRepo = repositories.NewTSEMarketRepository(db)
	}
	var dailyClosingRepo *repositories.TSEDailyClosingPriceRepository
	if needs.tseDailyClosingPrice {
		dailyClosingRepo = repositories.NewTSEDailyClosingPriceRepository(db)
	}
	var tradeHistoryRepo *repositories.TradeHistoryRepository
	if needs.tseTradeHistory {
		if ch == nil {
			closeDB()
			return nil, nil, fmt.Errorf("ClickHouse client is required for TSE trade history")
		}
		tradeHistoryRepo, err = repositories.NewTradeHistoryRepository(ch)
		if err != nil {
			closeDB()
			return nil, nil, err
		}
		if err := tradeHistoryRepo.EnsureSchema(ctx); err != nil {
			closeDB()
			return nil, nil, err
		}
	}

	var cryptoProviders []crypto.Crypto
	if needs.cryptoProviders {
		cryptoProviders, err = activeCryptoProviders(ctx, providerRepo, cfg, candleChan)
		if err != nil {
			closeDB()
			return nil, nil, err
		}
	}
	var stockProviders []*stockTSE.Adapter
	if needs.stockProviders {
		stockProviders, err = activeStockProviders(ctx, providerRepo, cfg, candleChan)
		if err != nil {
			closeDB()
			return nil, nil, err
		}
	}

	var tgManagerService *tgAccountService.Service
	if needs.telegram {
		tgManagerService = tgAccountService.NewService(cfg.Services.TgAccount, telegramRepo, filterRepo, proxyRepo)
	}
	bundles := &Bundles{
		Repositories: &RepositoriesBundle{
			Market:               marketRepo,
			Candle:               candleRepo,
			Asset:                assetRepo,
			Provider:             providerRepo,
			Timeframe:            timeframeRepo,
			Indicator:            indicatorRepo,
			Proxy:                proxyRepo,
			Telegram:             telegramRepo,
			ProxyAssignment:      proxyAssignmentRepo,
			StockDailyData:       stockDailyRepo,
			TSEMarket:            tseMarketRepo,
			TSEDailyClosingPrice: dailyClosingRepo,
			TSETradeHistory:      tradeHistoryRepo,
			StrategyDefinition:   strategyDefinitionRepo,
			Signal:               signalRepo,
		},
		Adapters: &AdaptersBundle{
			CryptoProviders: cryptoProviders,
			StockProviders:  stockProviders,
		},
		Services: &ServicesBundle{
			TgAccount: tgManagerService,
		},
		Channels: &ChannelsBundle{
			CandleChan: candleChan,
		},
		EventBus:  eventBus,
		TgManager: managerOf(tgManagerService),
	}
	return NewWorkerFactory(&cfg.Workers, bundles), closeDependencies, nil
}

func managerOf(service *tgAccountService.Service) *tgclient.Manager {
	if service == nil {
		return nil
	}
	return service.GetManager()
}

func migrateWorkerModels(db *gorm.DB, needs dependencyPlan) error {
	modelsToMigrate := make([]any, 0, 15)
	modelsToMigrate = append(modelsToMigrate, &models.StrategyDefinition{})
	if needs.provider {
		modelsToMigrate = append(modelsToMigrate, &models.Provider{})
	}
	if needs.asset {
		modelsToMigrate = append(modelsToMigrate, &models.Asset{})
	}
	if needs.timeframe {
		modelsToMigrate = append(modelsToMigrate, &models.Timeframe{})
	}
	if needs.market {
		modelsToMigrate = append(modelsToMigrate, &models.Market{})
	}
	if needs.candle {
		modelsToMigrate = append(modelsToMigrate, &models.Candle{})
	}
	if needs.indicator {
		modelsToMigrate = append(modelsToMigrate, &models.IndicatorDefinition{})
	}
	if needs.telegram {
		modelsToMigrate = append(modelsToMigrate, &models.TelegramAccount{}, &models.TelegramSubscriptionFilter{})
	}
	if needs.proxy {
		modelsToMigrate = append(modelsToMigrate, &models.Proxy{})
	}
	if needs.proxyAssignment {
		modelsToMigrate = append(modelsToMigrate, &models.ProxyAssignment{})
	}
	if needs.stockDailyData {
		modelsToMigrate = append(modelsToMigrate, &models.StockDailyData{})
	}
	if needs.tseMarket {
		modelsToMigrate = append(modelsToMigrate, &models.TSEMarket{})
	}
	if needs.tseDailyClosingPrice {
		modelsToMigrate = append(modelsToMigrate, &models.TSEDailyClosingPrice{})
	}
	if needs.signalWorker {
		modelsToMigrate = append(modelsToMigrate, &models.SignalRecord{})
	}
	return db.AutoMigrate(modelsToMigrate...)
}

func activeCryptoProviders(ctx context.Context, repo *repositories.ProviderRepository, cfg *configs.Config, candleChan chan crypto.Candle) ([]crypto.Crypto, error) {
	providers, err := repo.Active(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]crypto.Crypto, 0, len(providers))
	for _, provider := range providers {
		if provider.Name != consts.ProviderNobitex {
			continue
		}
		adapter, adapterErr := nobitex.NewAdapter(&cfg.Adapters.Nobitex, candleChan, provider.ID)
		if adapterErr != nil {
			return nil, fmt.Errorf("create crypto adapter %q: %w", provider.Name, adapterErr)
		}
		result = append(result, adapter)
	}
	return result, nil
}

func activeStockProviders(ctx context.Context, repo *repositories.ProviderRepository, cfg *configs.Config, candleChan chan crypto.Candle) ([]*stockTSE.Adapter, error) {
	providers, err := repo.ActiveStock(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*stockTSE.Adapter, 0, len(providers))
	for _, provider := range providers {
		adapter, adapterErr := stockTSE.NewAdapter(&cfg.Adapters.TSE, candleChan, provider.ID)
		if adapterErr != nil {
			return nil, fmt.Errorf("create stock adapter %q: %w", provider.Name, adapterErr)
		}
		result = append(result, adapter)
	}
	return result, nil
}
