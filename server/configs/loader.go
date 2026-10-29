package configs

import (
	"fmt"
	"os"
	"strings"
	"time"

	"errors"

	"github.com/h-varmazyar/gatec/internal/consts"
	gormext "github.com/h-varmazyar/gatec/pkg/gormOld"
	"github.com/h-varmazyar/gatec/pkg/tgclient"
	"github.com/h-varmazyar/gopack/jwt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Load reads configuration from .env file and environment variables
// The .env file is expected to be at root .env or ./apps/backend/.env
func Load() (*Config, error) {
	// Note: Logger not initialized yet, so we can't log here

	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.AllowEmptyEnv(true)

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	cfg := &Config{
		AppEnv:      consts.Environment(loadString("APP_ENV")),
		AppDebug:    loadBool("APP_DEBUG"),
		Locale:      loadString("LOCALE"),
		LogFile:     loadOptionalString("LOG_FILE", "logs/gatec.log"),
		HealthToken: loadString("HEALTH_TOKEN"),
		ServerAddr:  loadString("SERVER_ADDR"),
		ServerPort:  loadString("SERVER_PORT"),
		GinMode:     loadString("GIN_MODE"),
		Postgres: gormext.Configs{
			Host:        loadString("POSTGRES_HOST"),
			Port:        uint16(loadInt("POSTGRES_PORT")),
			Username:    loadString("POSTGRES_USER"),
			Password:    loadString("POSTGRES_PASSWORD"),
			Name:        loadString("POSTGRES_DB_NAME"),
			DbType:      gormext.Type(loadString("DB_TYPE")),
			IsSSLEnable: loadBool("IS_SSL_ENABLE"),
		},
		Jwt: &jwt.Configs{
			PublicKey:  loadFile(loadString("JWT_PUBLIC_KEY_PATH")),
			PrivateKey: loadFile(loadString("JWT_PRIVATE_KEY_PATH")),
			Method:     jwt.SigningMethod(loadString("SIGNING_METHOD")),
		},
		Adapters: Adapters{
			Nobitex: Nobitex{
				APIBaseURL: loadString("ADAPTER_NOBITEX_API_BASE_URL"),
				Socket: &SocketConfig{
					URL:                     loadString("ADAPTER_NOBITEX_SOCKET_URL"),
					MaxConnection:           loadInt("ADAPTER_NOBITEX_MAX_CONNECTION"),
					MaxChannelPerConnection: loadInt("ADAPTER_NOBITEX_MAX_CHANNEL_PER_CONNECTION"),
				},
			},
			Tabdeal: Tabdeal{
				APIBaseURL: loadString("ADAPTER_TABDEAL_API_BASE_URL"),
			},
		},

		Workers: Workers{
			// CandlePollingEnabled:        loadBool("WORKER_CANDLE_POLLING_ENABLED"),
			// MarketSyncEnabled:           loadBool("WORKER_MARKET_SYNC_ENABLED"),
			// ProxyHealthCheckerEnabled:   loadBool("WORKER_PROXY_HEALTH_CHECKER_ENABLED"),
			// ConnectionMonitorEnabled:    loadBool("WORKER_CONNECTION_MONITOR_ENABLED"),
			// IndicatorCalculatorEnabled:  loadBool("WORKER_INDICATOR_CALCULATOR_ENABLED"),
			// StockMarketSyncEnabled:      loadBool("WORKER_STOCK_MARKET_SYNC_ENABLED"),
			// StockInstrumentSyncEnabled:  loadOptionalBool("WORKER_STOCK_INSTRUMENT_SYNC_ENABLED", false),
			// TSEMarketHistorySyncEnabled: loadOptionalBool("WORKER_TSE_MARKET_HISTORY_SYNC_ENABLED", false),
			// TSEMarketDataSyncEnabled:    loadBool("WORKER_TSE_MARKET_DATA_SYNC_ENABLED"),
			// TSEMarketDailySyncEnabled:   loadOptionalBool("WORKER_TSE_MARKET_DAILY_SYNC_ENABLED", false),
			TSEMarketWatchEnabled: loadOptionalBool("WORKER_TSE_MARKET_WATCH_ENABLED", false),
			IndicatorCalculator: &IndicatorCalculator{
				ExecutionIttrationDuration: loadDuration("WORKER_INDICATOR_CALCULATOR_EXECUTION_ITTRATION_DURATION"),
				SignalPipelineCount:        1,
				CryptoP0Enabled:            loadOptionalBool("WORKER_INDICATOR_CRYPTO_P0_ENABLED", true),
				CryptoATRPercentPeriod:     loadOptionalInt("WORKER_INDICATOR_ATR_PCT_PERIOD", 14),
				CryptoDonchianPeriod:       loadOptionalInt("WORKER_INDICATOR_DONCHIAN_PERIOD", 20),
				CryptoROCPeriods:           []int{loadOptionalInt("WORKER_INDICATOR_ROC_PERIOD_SHORT", 6), loadOptionalInt("WORKER_INDICATOR_ROC_PERIOD_LONG", 12)},
				CryptoMFIPeriod:            loadOptionalInt("WORKER_INDICATOR_MFI_PERIOD", 14),
				CryptoBollingerPeriod:      loadOptionalInt("WORKER_INDICATOR_BOLLINGER_PERIOD", 20),
				CryptoBollingerDeviation:   loadOptionalInt("WORKER_INDICATOR_BOLLINGER_DEVIATION", 2),
			},
			CandlePolling: &CandlePolling{},
			MarketSync: &MarketSync{
				SyncInterval: time.Hour,
			},
			TSEInstrumentSync: &TSEInstrumentSync{
				SingleCheck:  loadBool("WORKER_TSE_INSTRUMENT_SYNC_SINGLE_CHECK"),
				ScheduleHour: loadOptionalInt("WORKER_TSE_INSTRUMENT_SYNC_SCHEDULE_HOUR", 8),
				ScheduleMin:  loadOptionalInt("WORKER_TSE_INSTRUMENT_SYNC_SCHEDULE_MIN", 0),
			},
			ProxyHealthChecker: &ProxyHealthChecker{
				CheckInterval: time.Minute,
				StaleDuration: 24 * time.Hour,
				ProxyTimeout:  10 * time.Second,
			},
			ConnectionMonitor: &ConnectionMonitor{
				CheckInterval: 30 * time.Second,
			},
			SignalChecker: &SignalChecker{
				ExecutionIttrationDuration: loadDuration("WORKER_SIGNAL_CHECKER_EXECUTION_ITTRATION_DURATION"),
			},
			SignalWorker: &SignalWorker{
				StrategySource:   loadOptionalString("WORKER_SIGNAL_STRATEGY_SOURCE", "both"),
				StrategiesDir:    loadOptionalString("WORKER_SIGNAL_STRATEGIES_DIR", "strategies"),
				PollInterval:     loadOptionalDuration("WORKER_SIGNAL_POLL_INTERVAL", 2*time.Second),
				MaxEventAge:      loadOptionalDuration("WORKER_SIGNAL_MAX_EVENT_AGE", 0),
				Shards:           loadOptionalInt("WORKER_SIGNAL_SHARDS", 4),
				FailureThreshold: loadOptionalInt("WORKER_SIGNAL_FAILURE_THRESHOLD", 3),
				WarmupMargin:     loadOptionalInt("WORKER_SIGNAL_WARMUP_MARGIN", 2),
				StateCapacity:    loadOptionalInt("WORKER_SIGNAL_STATE_CAPACITY", 100),
			},
		},
		Services: Services{
			Auth: AuthService{
				BcryptCost:     loadInt("SERVICE_AUTH_BCRYPT_COST"),
				ExpirationTime: loadDuration("SERVICE_AUTH_EXPIRATION_TIME"),
				HashingSalt:    loadString("SERVICE_AUTH_HASHING_SALT"),
			},
			TgAccount: &TgAccountService{
				AuthManager: &tgclient.Config{
					APIID:           int32(loadInt("SERVICE_TELEGRAM_API_ID")),
					APIHash:         loadString("SERVICE_TELEGRAM_API_HASH"),
					DatabaseBaseDir: loadString("SERVICE_TELEGRAM_DATABASE_DIR"),
					AuthTimeout:     loadDuration("SERVICE_TELEGRAM_AUTH_TIMEOUT"),
				},
			},
		},
		NATS: NATS{
			URL:             loadOptionalString("NATS_URL", ""),
			Stream:          loadOptionalString("NATS_WORKER_STREAM", "GATEC_WORKERS"),
			CandleSubject:   loadOptionalString("NATS_CANDLE_SUBJECT", "market.candle.updated"),
			SnapshotSubject: loadOptionalString("NATS_CANDLE_SNAPSHOT_SUBJECT", "market.candle.indicators"),
			ConsumerPrefix:  loadOptionalString("NATS_WORKER_CONSUMER_PREFIX", "gatec-indicator"),
		},
	}
	if !cfg.Workers.TSEInstrumentSync.SingleCheck && (cfg.Workers.TSEInstrumentSync.ScheduleHour < 0 || cfg.Workers.TSEInstrumentSync.ScheduleHour > 23) {
		return nil, fmt.Errorf("WORKER_TSE_INSTRUMENT_SYNC_SCHEDULE_HOUR must be between 0 and 23")
	}
	if !cfg.Workers.TSEInstrumentSync.SingleCheck && (cfg.Workers.TSEInstrumentSync.ScheduleMin < 0 || cfg.Workers.TSEInstrumentSync.ScheduleMin > 59) {
		return nil, fmt.Errorf("WORKER_TSE_INSTRUMENT_SYNC_SCHEDULE_MIN must be between 0 and 59")
	}

	// if cfg.Workers.TSEMarketHistorySyncEnabled {
	cfg.Workers.TSEMarketHistorySync = &TSEMarketHistorySync{
		Mode:            loadOptionalString("WORKER_TSE_MARKET_HISTORY_SYNC_MODE", "all"),
		Date:            loadOptionalString("WORKER_TSE_MARKET_HISTORY_SYNC_DATE", ""),
		StartDate:       loadOptionalString("WORKER_TSE_MARKET_HISTORY_SYNC_START_DATE", ""),
		EndDate:         loadOptionalString("WORKER_TSE_MARKET_HISTORY_SYNC_END_DATE", ""),
		RequestInterval: loadOptionalDuration("WORKER_TSE_MARKET_HISTORY_SYNC_REQUEST_INTERVAL", 250*time.Millisecond),
	}
	if cfg.Workers.TSEMarketHistorySync.Mode != "all" && cfg.Workers.TSEMarketHistorySync.Mode != "date" && cfg.Workers.TSEMarketHistorySync.Mode != "range" && cfg.Workers.TSEMarketHistorySync.Mode != "today" {
		return nil, fmt.Errorf("invalid WORKER_TSE_MARKET_HISTORY_SYNC_MODE %q", cfg.Workers.TSEMarketHistorySync.Mode)
	}
	parseHistoryDate := func(name, value string) error {
		if value == "" {
			return nil
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("%s must use YYYY-MM-DD: %w", name, err)
		}
		return nil
	}
	if err := parseHistoryDate("WORKER_TSE_MARKET_HISTORY_SYNC_DATE", cfg.Workers.TSEMarketHistorySync.Date); err != nil {
		return nil, err
	}
	if err := parseHistoryDate("WORKER_TSE_MARKET_HISTORY_SYNC_START_DATE", cfg.Workers.TSEMarketHistorySync.StartDate); err != nil {
		return nil, err
	}
	if err := parseHistoryDate("WORKER_TSE_MARKET_HISTORY_SYNC_END_DATE", cfg.Workers.TSEMarketHistorySync.EndDate); err != nil {
		return nil, err
	}
	if cfg.Workers.TSEMarketHistorySync.Mode == "date" && cfg.Workers.TSEMarketHistorySync.Date == "" {
		return nil, fmt.Errorf("WORKER_TSE_MARKET_HISTORY_SYNC_DATE is required in date mode")
	}
	if cfg.Workers.TSEMarketHistorySync.Mode == "range" && (cfg.Workers.TSEMarketHistorySync.StartDate == "" || cfg.Workers.TSEMarketHistorySync.EndDate == "") {
		return nil, fmt.Errorf("history start and end dates are required in range mode")
	}
	// }

	cfg.Workers.TSEMarketDailySync = &TSEMarketDailySync{
		ScheduleHour:    loadOptionalInt("WORKER_TSE_MARKET_DAILY_SYNC_SCHEDULE_HOUR", 21),
		ScheduleMin:     loadOptionalInt("WORKER_TSE_MARKET_DAILY_SYNC_SCHEDULE_MIN", 22),
		RequestInterval: loadOptionalDuration("WORKER_TSE_MARKET_DAILY_SYNC_REQUEST_INTERVAL", 250*time.Millisecond),
		BotToken:        loadOptionalString("WORKER_TSE_MARKET_DAILY_SYNC_TELEGRAM_BOT_TOKEN", ""),
		ReportChatID:    loadOptionalInt64("WORKER_TSE_MARKET_DAILY_SYNC_REPORT_CHAT_ID", 1689926),
	}
	cfg.Workers.TSEMarketDataSync = &TSEMarketDataSync{
		Mode:             consts.TSEMarketDataMode(loadString("TSE_MARKET_DATA_SYNC_MODE")),
		CollectionMethod: loadOptionalString("TSE_MARKET_DATA_SYNC_COLLECTION_METHOD", "all"),
		InsCodes:         splitCSV(loadOptionalString("TSE_MARKET_DATA_SYNC_INS_CODES", "")),
		RequestInterval:  loadDuration("TSE_MARKET_DATA_SYNC_REQUEST_INTERVAL"),
		ScheduleHour:     loadOptionalInt("TSE_MARKET_DATA_SYNC_SCHEDULE_HOUR", 21),
		ScheduleMin:      loadOptionalInt("TSE_MARKET_DATA_SYNC_SCHEDULE_MIN", 0),
	}
	if cfg.Workers.TSEMarketDataSync.CollectionMethod != "all" && cfg.Workers.TSEMarketDataSync.CollectionMethod != "missed" && cfg.Workers.TSEMarketDataSync.CollectionMethod != "ins_codes" {
		return nil, fmt.Errorf("invalid TSE_MARKET_DATA_SYNC_COLLECTION_METHOD %q (expected all, missed, or ins_codes)", cfg.Workers.TSEMarketDataSync.CollectionMethod)
	}
	if cfg.Workers.TSEMarketDataSync.CollectionMethod == "ins_codes" && len(cfg.Workers.TSEMarketDataSync.InsCodes) == 0 {
		return nil, fmt.Errorf("TSE_MARKET_DATA_SYNC_INS_CODES is required when collection method is ins_codes")
	}
	if cfg.Workers.TSEMarketDataSync.ScheduleHour < 0 || cfg.Workers.TSEMarketDataSync.ScheduleHour > 23 ||
		cfg.Workers.TSEMarketDataSync.ScheduleMin < 0 || cfg.Workers.TSEMarketDataSync.ScheduleMin > 59 {
		return nil, fmt.Errorf("TSE_MARKET_DATA_SYNC schedule must contain valid clock values")
	}
	cfg.Workers.TSEMarketWatch = &TSEMarketWatch{
		StartHour:    loadOptionalInt("WORKER_TSE_MARKET_WATCH_START_HOUR", 8),
		StartMin:     loadOptionalInt("WORKER_TSE_MARKET_WATCH_START_MIN", 0),
		EndHour:      loadOptionalInt("WORKER_TSE_MARKET_WATCH_END_HOUR", 16),
		EndMin:       loadOptionalInt("WORKER_TSE_MARKET_WATCH_END_MIN", 0),
		PollInterval: loadOptionalDuration("WORKER_TSE_MARKET_WATCH_POLL_INTERVAL", 30*time.Second),
	}
	if cfg.Workers.TSEMarketWatch.StartHour < 0 || cfg.Workers.TSEMarketWatch.StartHour > 23 ||
		cfg.Workers.TSEMarketWatch.EndHour < 0 || cfg.Workers.TSEMarketWatch.EndHour > 23 ||
		cfg.Workers.TSEMarketWatch.StartMin < 0 || cfg.Workers.TSEMarketWatch.StartMin > 59 ||
		cfg.Workers.TSEMarketWatch.EndMin < 0 || cfg.Workers.TSEMarketWatch.EndMin > 59 {
		return nil, fmt.Errorf("TSE market watch schedule must contain valid clock values")
	}
	if cfg.Workers.TSEMarketDailySync.ScheduleHour < 0 || cfg.Workers.TSEMarketDailySync.ScheduleHour > 23 {
		return nil, fmt.Errorf("WORKER_TSE_MARKET_DAILY_SYNC_SCHEDULE_HOUR must be between 0 and 23")
	}
	if cfg.Workers.TSEMarketDailySync.ScheduleMin < 0 || cfg.Workers.TSEMarketDailySync.ScheduleMin > 59 {
		return nil, fmt.Errorf("WORKER_TSE_MARKET_DAILY_SYNC_SCHEDULE_MIN must be between 0 and 59")
	}
	// if cfg.Workers.TSEMarketDailySyncEnabled && cfg.Workers.TSEMarketDailySync.BotToken == "" {
	// 	return nil, fmt.Errorf("WORKER_TSE_MARKET_DAILY_SYNC_TELEGRAM_BOT_TOKEN is required when daily TSE sync is enabled")
	// }

	// if cfg.Workers.StockMarketSyncEnabled {
	stockConfig := &StockMarketSync{
		ProviderID: loadUint("WORKER_STOCK_MARKET_SYNC_PROVIDER_ID"),
		Mode:       loadString("WORKER_STOCK_MARKET_SYNC_MODE"),
	}
	switch stockConfig.Mode {
	case "daily":
		stockConfig.ScheduleHour = loadInt("WORKER_STOCK_MARKET_SYNC_SCHEDULE_HOUR")
		stockConfig.ScheduleMin = loadInt("WORKER_STOCK_MARKET_SYNC_SCHEDULE_MIN")
	case "historical":
		stockConfig.HistoricalStartDate = loadString("WORKER_STOCK_MARKET_SYNC_HISTORICAL_START_DATE")
		stockConfig.HistoricalEndDate = loadString("WORKER_STOCK_MARKET_SYNC_HISTORICAL_END_DATE")
		stockConfig.RequestInterval = loadDuration("WORKER_STOCK_MARKET_SYNC_REQUEST_INTERVAL")
	default:
		return nil, fmt.Errorf("invalid WORKER_STOCK_MARKET_SYNC_MODE %q", stockConfig.Mode)
	}
	cfg.Workers.StockMarketSync = stockConfig
	// }

	var err error
	cfg.LogLevel, err = logrus.ParseLevel(loadString("LOG_LEVEL"))
	if err != nil {
		return nil, fmt.Errorf("parsing log level: %w", err)
	}
	return cfg, nil
}

func loadString(envName string) string {
	validate(envName)
	return viper.GetString(envName)
}

func loadInt(envName string) int {
	validate(envName)
	return viper.GetInt(envName)
}

func loadInt64(envName string) int64 {
	validate(envName)
	return viper.GetInt64(envName)
}

func loadUint(envName string) uint {
	validate(envName)
	return viper.GetUint(envName)
}

//nolint:unused
func loadUint64(envName string) uint64 {
	validate(envName)
	return viper.GetUint64(envName)
}

func loadBool(envName string) bool {
	validate(envName)
	return viper.GetBool(envName)
}

func loadOptionalBool(envName string, defaultValue bool) bool {
	if !viper.IsSet(envName) {
		return defaultValue
	}
	return viper.GetBool(envName)
}

func loadOptionalString(envName, defaultValue string) string {
	if !viper.IsSet(envName) {
		return defaultValue
	}
	return viper.GetString(envName)
}

func splitCSV(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func loadOptionalInt(envName string, defaultValue int) int {
	if !viper.IsSet(envName) {
		return defaultValue
	}
	return viper.GetInt(envName)
}

func loadOptionalInt64(envName string, defaultValue int64) int64 {
	if !viper.IsSet(envName) {
		return defaultValue
	}
	return viper.GetInt64(envName)
}

func loadOptionalDuration(envName string, defaultValue time.Duration) time.Duration {
	if !viper.IsSet(envName) {
		return defaultValue
	}
	return viper.GetDuration(envName)
}

func loadFile(filePath string) string {
	if filePath == "" {
		return ""
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("reading file [%s]: %v", filePath, err))
	}

	return string(content)
}

func loadFloat(envName string) float64 {
	validate(envName)
	return viper.GetFloat64(envName)
}

func loadStringSlice(envName string) []string {
	validate(envName)
	return viper.GetStringSlice(envName)
}

func loadDuration(envName string) time.Duration {
	validate(envName)
	return viper.GetDuration(envName)
}

func validate(envName string) {
	exists := viper.IsSet(envName)
	if !exists {
		panic(fmt.Sprintf("environment variable [%s] does not exist", envName))
	}
}
