package main

import (
	"context"
	"fmt"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto/nobitex"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto/tabdeal"
	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	stockTSE "github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/api/rest"
	"github.com/h-varmazyar/gatec/internal/consts"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	airdropService "github.com/h-varmazyar/gatec/internal/services/airdrop"
	applogService "github.com/h-varmazyar/gatec/internal/services/applog"
	authService "github.com/h-varmazyar/gatec/internal/services/auth"
	dynamicConfigService "github.com/h-varmazyar/gatec/internal/services/dynamicconfig"
	messagingService "github.com/h-varmazyar/gatec/internal/services/messaging"
	proxyService "github.com/h-varmazyar/gatec/internal/services/proxy"
	strategyService "github.com/h-varmazyar/gatec/internal/services/strategy"
	tgAccountService "github.com/h-varmazyar/gatec/internal/services/tgaccount"
	tseMarketService "github.com/h-varmazyar/gatec/internal/services/tsemarket"
	userService "github.com/h-varmazyar/gatec/internal/services/user"

	clickhouseClient "github.com/h-varmazyar/gatec/pkg/clickhouse"
	gormext "github.com/h-varmazyar/gatec/pkg/gormOld"
	"github.com/h-varmazyar/gatec/pkg/logger"
	jwtPkg "github.com/h-varmazyar/gopack/jwt"
)

func main() {
	cfg, err := configs.Load()
	if err != nil {
		panic(err)
	}

	logFile, err := logger.Configure(cfg.LogFile, cfg.LogLevel)
	if err != nil {
		panic(fmt.Errorf("configure application logger: %w", err))
	}
	defer logFile.Close()

	ctx := context.Background()

	dep, err := initializeDependencies(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer dep.ClickHouse.Close()

	if err := jwtPkg.Load(cfg.Jwt); err != nil {
		panic("failed to load jwt config:" + err.Error())
	}
	jwtStorage, err := jwtPkg.NewGormClaimsStore(dep.DB)
	jwtPkg.SetClaimStore(jwtStorage)

	// بارگذاری قوانین دسترسی پیام‌رسانی
	if err := dep.Service.Messaging.LoadRules(ctx); err != nil {
		logrus.WithError(err).Error("failed to load messaging access rules")
	}

	{
		serverAddress := cfg.ServerAddr + ":" + cfg.ServerPort
		err = rest.NewServer(
			cfg.AppEnv,
			dep.Service.Auth,
			dep.Service.ApplicationLog,
			dep.Service.User,
			dep.Service.Airdrop,
			dep.Service.Proxy,
			dep.Service.TgAccount,
			dep.Service.Messaging,
			dep.Service.DynamicConfig,
			dep.Service.TSEMarket,
		).Serve(ctx, serverAddress)
		if err != nil {
			panic(err)
		}
	}

	select {}
}

type depencencies struct {
	Repositories struct {
		Asset                      *repositories.AssetRepository
		Candle                     *repositories.CandleRepository
		Indicator                  *repositories.IndicatorDefinitionRepository
		Market                     *repositories.MarketRepository
		Provider                   *repositories.ProviderRepository
		Strategy                   *repositories.StrategyRepository
		Timeframe                  *repositories.TimeframeRepository
		User                       *repositories.UserRepository
		TabdealAirdrop             *repositories.TabdealAirdropRepository
		GiftAllocaitionRepo        *repositories.TabdealAllocationRepository
		Telegram                   *repositories.TelegramAccountRepository
		TelegramSubscriptionFilter *repositories.TelegramSubscriptionFilterRepository
		Proxy                      *repositories.ProxyRepository
		ProxyAssignment            *repositories.ProxyAssignmentRepository
		StockDailyData             *repositories.StockDailyDataRepository
		TSEMarket                  *repositories.TSEMarketRepository
		TSEDailyClosingPrice       *repositories.TSEDailyClosingPriceRepository
		TSEIndicatorSnapshot       *repositories.TSEIndicatorSnapshotRepository
		TSETradeHistory            *repositories.TradeHistoryRepository
		BotUser                    *repositories.BotUserRepository
		Message                    *repositories.MessageRepository
		AccessRule                 *repositories.AccessRuleRepository
		DynamicConfig              *repositories.DynamicConfigRepository
		ApplicationLog             *repositories.ApplicationLogRepository
	}
	Service struct {
		Airdrop        *airdropService.Service
		ApplicationLog *applogService.Service
		Auth           *authService.Service
		Proxy          *proxyService.Service
		Strategy       *strategyService.Service
		User           *userService.Service
		TgAccount      *tgAccountService.Service
		Messaging      *messagingService.Service
		DynamicConfig  *dynamicConfigService.Service
		TSEMarket      *tseMarketService.Service
	}
	Adapters struct {
		Nobitex *nobitex.Adapter
		tabdeal *tabdeal.Adapter
	}
	DB             *gorm.DB
	ClickHouse     *clickhouseClient.Client
	CandleChan     chan crypto.Candle
	Providers      []crypto.Crypto
	StockProviders []*tse.Adapter
	// WorkerFactory       *worker.WorkerFactory
}

func initializeDependencies(ctx context.Context, cfg *configs.Config) (*depencencies, error) {
	dep := &depencencies{}
	var err error

	dep.DB, err = gormext.Open(cfg.Postgres)
	if err != nil {
		return nil, err
	}
	dep.ClickHouse, err = clickhouseClient.New(cfg.ClickHouse)
	if err != nil {
		sqlDB, closeErr := dep.DB.DB()
		if closeErr == nil {
			_ = sqlDB.Close()
		}
		return nil, fmt.Errorf("initialize ClickHouse client: %w", err)
	}
	clickHouseReady := false
	defer func() {
		if !clickHouseReady {
			dep.ClickHouse.Close()
		}
	}()
	if err = dep.ClickHouse.Ping(ctx); err != nil {
		sqlDB, closeErr := dep.DB.DB()
		if closeErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}
	if dep.DB.Migrator().HasIndex("dynamic_configs", "idx_dynamic_configs_key") {
		if err = dep.DB.Migrator().DropIndex("dynamic_configs", "idx_dynamic_configs_key"); err != nil {
			return nil, err
		}
	}
	if err = dep.DB.AutoMigrate(
		&models.Provider{},
		&models.Asset{},
		&models.Timeframe{},
		&models.Market{},
		&models.Candle{},
		&models.IndicatorDefinition{},
		&models.StrategyDefinition{},
		&models.SignalRecord{},
		&models.User{},
		&models.TabdealAirdrop{},
		&models.TabdealAirdropGift{},
		&models.TelegramAccount{},
		&models.TelegramSubscriptionFilter{},
		&models.Proxy{},
		&models.ProxyAssignment{},
		&models.BotUser{},
		&models.Message{},
		&models.AccessRule{},
		&models.SendRecord{},
		&models.StockDailyData{},
		&models.TSEMarket{},
		&models.TSEMarketIssue{},
		&models.TSEDailyClosingPrice{},
		&models.TSEIndicatorSnapshot{},
		&models.DynamicConfig{},
	); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	dep.CandleChan = make(chan crypto.Candle, 1000)

	initializeRepositories(ctx, dep, cfg.LogFile)
	dep.Repositories.TSETradeHistory, err = repositories.NewTradeHistoryRepository(dep.ClickHouse)
	if err != nil {
		return nil, err
	}
	if err = dep.Repositories.TSETradeHistory.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("initialize TSE trade history repository: %w", err)
	}
	if err = initializeAdapters(ctx, dep, cfg.Adapters); err != nil {
		return nil, err
	}

	dep.Providers = make([]crypto.Crypto, 0)

	activeProviders, err := dep.Repositories.Provider.Active(ctx)
	if err != nil {
		panic(err)
	}

	for _, p := range activeProviders {
		var provider crypto.Crypto
		switch p.Name {
		case consts.ProviderNobitex:
			provider, err = nobitex.NewAdapter(&cfg.Adapters.Nobitex, dep.CandleChan, p.ID)
			if err != nil {
				logrus.WithError(err).Errorf("failed to add provider %v", p.Name)
			}
		}

		dep.Providers = append(dep.Providers, provider)
	}

	initializeServices(ctx, dep, cfg.Services)

	clickHouseReady = true
	return dep, nil
}

func initializeRepositories(_ context.Context, dep *depencencies, logFile string) {
	dep.Repositories.ApplicationLog = repositories.NewApplicationLogRepository(logFile)
	dep.Repositories.Market = repositories.NewMarketRepository(dep.DB)
	dep.Repositories.Provider = repositories.NewProviderRepository(dep.DB)
	dep.Repositories.Asset = repositories.NewAssetRepository(dep.DB)
	dep.Repositories.Timeframe = repositories.NewTimeframeRepository(dep.DB)
	dep.Repositories.Candle = repositories.NewCandleRepository(dep.DB)
	dep.Repositories.Indicator = repositories.NewIndicatorDefinitionRepository(dep.DB)
	dep.Repositories.User = repositories.NewUserRepository(dep.DB)
	dep.Repositories.Strategy = repositories.NewStrategyRepository(dep.DB)
	dep.Repositories.TabdealAirdrop = repositories.NewTabdealAirdropRepository(dep.DB)
	dep.Repositories.GiftAllocaitionRepo = repositories.NewTabdealAllocationRepository(dep.DB)
	dep.Repositories.Telegram = repositories.NewTelegramAccountRepository(dep.DB)
	dep.Repositories.TelegramSubscriptionFilter = repositories.NewTelegramSubscriptionFilterRepository(dep.DB)
	dep.Repositories.Proxy = repositories.NewProxyRepository(dep.DB)
	dep.Repositories.ProxyAssignment = repositories.NewProxyAssignmentRepository(dep.DB)
	dep.Repositories.StockDailyData = repositories.NewStockDailyDataRepository(dep.DB)
	dep.Repositories.TSEMarket = repositories.NewTSEMarketRepository(dep.DB)
	dep.Repositories.TSEDailyClosingPrice = repositories.NewTSEDailyClosingPriceRepository(dep.DB)
	dep.Repositories.TSEIndicatorSnapshot = repositories.NewTSEIndicatorSnapshotRepository(dep.DB)
	dep.Repositories.BotUser = repositories.NewBotUserRepository(dep.DB)
	dep.Repositories.Message = repositories.NewMessageRepository(dep.DB)
	dep.Repositories.AccessRule = repositories.NewAccessRuleRepository(dep.DB)
}

func initializeAdapters(ctx context.Context, dep *depencencies, cfg configs.Adapters) error {
	dep.Adapters.tabdeal = tabdeal.NewAdapter(&cfg.Tabdeal)

	activeStockProviders, err := dep.Repositories.Provider.ActiveStock(ctx)
	if err != nil {
		return err
	}
	dep.StockProviders = make([]*tse.Adapter, 0, len(activeStockProviders))
	for _, provider := range activeStockProviders {
		switch provider.Name {
		case consts.ProviderTSE:
			adapter, err := stockTSE.NewAdapter(&cfg.TSE, dep.CandleChan, provider.ID)
			if err != nil {
				return err
			}
			dep.StockProviders = append(dep.StockProviders, adapter)
		}
	}

	return nil
}

func initializeServices(_ context.Context, dep *depencencies, cfg configs.Services) {
	dep.Service.ApplicationLog = applogService.NewService(dep.Repositories.ApplicationLog)
	dep.Service.Strategy = strategyService.NewService(dep.Repositories.Indicator, dep.Repositories.Strategy, dep.Repositories.Market, dep.Repositories.Provider)
	dep.Service.Auth = authService.NewService(&cfg.Auth, dep.Repositories.User)
	dep.Service.User = userService.NewService(dep.Repositories.User)
	dep.Service.Proxy = proxyService.NewService(dep.Repositories.Proxy, dep.Repositories.ProxyAssignment)
	dep.Service.TgAccount = tgAccountService.NewService(cfg.TgAccount, dep.Repositories.Telegram, dep.Repositories.TelegramSubscriptionFilter, dep.Repositories.Proxy)
	dep.Service.Airdrop = airdropService.NewService(cfg.AirdropService, dep.Repositories.TabdealAirdrop, dep.Repositories.GiftAllocaitionRepo, dep.Repositories.User, dep.Adapters.tabdeal, dep.Service.TgAccount)
	dep.Service.Messaging = messagingService.NewService(dep.Repositories.Message, dep.Repositories.BotUser, dep.Repositories.AccessRule)
	var tseAdapter *tse.Adapter
	if len(dep.StockProviders) > 0 {
		tseAdapter = dep.StockProviders[0]
	}
	dep.Service.TSEMarket = tseMarketService.NewService(dep.Repositories.TSEMarket, dep.Repositories.Market, dep.Repositories.Asset, tseAdapter)
}
