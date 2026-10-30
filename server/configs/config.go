package configs

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	gormext "github.com/h-varmazyar/gatec/pkg/gormOld"
	"github.com/h-varmazyar/gatec/pkg/tgclient"
	jwtPkg "github.com/h-varmazyar/gopack/jwt"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AppEnv      consts.AppEnv
	AppDebug    bool
	Locale      string
	LogLevel    logrus.Level
	LogFile     string
	HealthToken string
	// Server config
	ServerAddr string
	ServerPort string
	GinMode    string

	PostgresURI string
	Postgres    gormext.Configs
	Jwt         *jwtPkg.Configs

	// Environment
	Environment string

	Adapters   Adapters
	Workers    Workers
	Services   Services
	NATS       NATS
	ClickHouse ClickHouse
}

type ClickHouse struct {
	URL      string
	Database string
	Username string
	Password string
}

type NATS struct {
	URL             string
	Stream          string
	CandleSubject   string
	SnapshotSubject string
	ConsumerPrefix  string
}

type Adapters struct {
	Nobitex Nobitex
	Tabdeal Tabdeal
	TSE     TSE
}
type Nobitex struct {
	Socket     *SocketConfig
	APIBaseURL string
}

type TSE struct {
}

type Tabdeal struct {
	APIBaseURL string
	AuthToken  string
}

type Workers struct {
	// CandlePollingEnabled        bool
	// MarketSyncEnabled           bool
	// ProxyHealthCheckerEnabled   bool
	// ConnectionMonitorEnabled    bool
	// IndicatorCalculatorEnabled  bool
	// StockMarketSyncEnabled      bool
	// StockInstrumentSyncEnabled  bool
	// TSEMarketDataSyncEnabled    bool
	// TSEMarketHistorySyncEnabled bool
	// TSEMarketDailySyncEnabled   bool
	TSEMarketWatchEnabled bool

	CandlePolling        *CandlePolling
	MarketSync           *MarketSync
	ProxyHealthChecker   *ProxyHealthChecker
	ConnectionMonitor    *ConnectionMonitor
	IndicatorCalculator  *IndicatorCalculator
	SignalChecker        *SignalChecker
	SignalWorker         *SignalWorker
	StockMarketSync      *StockMarketSync
	TSEInstrumentSync    *TSEInstrumentSync
	TSEMarketHistorySync *TSEMarketHistorySync
	TSEMarketDailySync   *TSEMarketDailySync
	TSEMarketWatch       *TSEMarketWatch
	TSEMarketDataSync    *TSEMarketDataSync
	TSETradeHistory      *TSETradeHistory
}

type TSEMarketHistorySync struct {
	Mode            string
	Date            string
	StartDate       string
	EndDate         string
	RequestInterval time.Duration
}

type TSEMarketDailySync struct {
	ScheduleHour    int
	ScheduleMin     int
	RequestInterval time.Duration
	BotToken        string
	ReportChatID    int64
}

type TSEMarketWatch struct {
	StartHour    int
	StartMin     int
	EndHour      int
	EndMin       int
	PollInterval time.Duration
}

type TSETradeHistory struct{}

type TSEMarketDataSync struct {
	Mode             consts.TSEMarketDataMode
	CollectionMethod string
	InsCodes         []string
	Date             time.Time
	StartDate        time.Time
	EndDate          time.Time
	RequestInterval  time.Duration
	ScheduleHour     int
	ScheduleMin      int
}

type StockMarketSync struct {
	ProviderID          uint
	Mode                string        // daily or historical
	ScheduleHour        int           // Daily mode: hour of day (0-23), default 16
	ScheduleMin         int           // Daily mode: minute of hour (0-59), default 0
	HistoricalStartDate string        // Historical mode: newest Jalali date, YYYY/MM/DD
	HistoricalEndDate   string        // Historical mode: oldest Jalali date, YYYY/MM/DD
	RequestInterval     time.Duration // Historical mode: delay between requests, default 1m
}

type TSEInstrumentSync struct {
	SingleCheck  bool
	ScheduleHour int
	ScheduleMin  int
}

type CandlePolling struct {
}

type MarketSync struct {
	SyncInterval time.Duration
}

type ProxyHealthChecker struct {
	CheckInterval time.Duration
	StaleDuration time.Duration
	ProxyTimeout  time.Duration
}

type ConnectionMonitor struct {
	CheckInterval time.Duration
}

type IndicatorCalculator struct {
	ExecutionIttrationDuration time.Duration
	SignalPipelineCount        uint
	CryptoP0Enabled            bool
	CryptoATRPercentPeriod     int
	CryptoDonchianPeriod       int
	CryptoROCPeriods           []int
	CryptoMFIPeriod            int
	CryptoBollingerPeriod      int
	CryptoBollingerDeviation   int
}

type SignalChecker struct {
	ExecutionIttrationDuration time.Duration
}

type SignalWorker struct {
	StrategySource   string
	StrategiesDir    string
	PollInterval     time.Duration
	MaxEventAge      time.Duration
	Shards           int
	FailureThreshold int
	WarmupMargin     int
	StateCapacity    int
}

type Services struct {
	Auth           AuthService
	AirdropService AirdropService
	TgAccount      *TgAccountService
}

type AuthService struct {
	BcryptCost     int
	ExpirationTime time.Duration
	JWTSecretKey   string
	HashingSalt    string
}

type AirdropService struct {
	AdminChatID int64
}

type TgAccountService struct {
	AuthManager *tgclient.Config
}
type SocketConfig struct {
	URL                     string
	MaxConnection           int
	MaxChannelPerConnection int
}
