package dynamicconfig

import (
	"context"
	"fmt"
	// "strconv"
	// "strings"
	"sync"
	// "time"

	// "github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto/tabdeal"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/worker"
)

type SectionValues map[string]string

const (
	configTypeString   = "string"
	configTypeBoolean  = "boolean"
	configTypeInteger  = "integer"
	configTypeDuration = "duration"
	configTypeSecret   = "secret"
)

type SectionItem struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}

type Service struct {
	repo           *repositories.DynamicConfigRepository
	tabdealAdapter *tabdeal.Adapter
	// workerFactory  *worker.WorkerFactory
	mu sync.Mutex
}

func NewService(repo *repositories.DynamicConfigRepository, adapter *tabdeal.Adapter) *Service {
	return &Service{repo: repo, tabdealAdapter: adapter}
}

// func (s *Service) SetWorkerFactory(factory *worker.WorkerFactory) {
// 	s.workerFactory = factory
// }

// func (s *Service) Initialize(ctx context.Context, adapterCfg configs.Tabdeal, workerCfg *configs.Workers) error {
// 	defaults := s.defaults(adapterCfg, workerCfg)
// 	if err := s.repo.EnsureMany(ctx, defaults); err != nil {
// 		return fmt.Errorf("seed dynamic configs: %w", err)
// 	}
// 	sections := make([]string, 0, len(defaults))
// 	for section := range defaults {
// 		sections = append(sections, section)
// 	}
// 	items, err := s.repo.ListBySections(ctx, sections)
// 	if err != nil {
// 		return fmt.Errorf("load dynamic configs: %w", err)
// 	}
// 	grouped := make(map[string]SectionValues)
// 	for _, item := range items {
// 		if grouped[item.Section] == nil {
// 			grouped[item.Section] = SectionValues{}
// 		}
// 		grouped[item.Section][item.Key] = item.Value
// 	}
// 	for section, values := range grouped {
// 		if err := s.apply(section, values, false); err != nil {
// 			return fmt.Errorf("apply section %s: %w", section, err)
// 		}
// 	}
// 	return nil
// }

func (s *Service) ListSections(ctx context.Context, sections []string) (map[string][]SectionItem, error) {
	if len(sections) == 0 {
		return nil, fmt.Errorf("at least one section is required")
	}
	for _, section := range sections {
		if !s.knownSection(section) {
			return nil, fmt.Errorf("unknown section %q", section)
		}
	}
	items, err := s.repo.ListBySections(ctx, sections)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]SectionItem, len(sections))
	for _, section := range sections {
		result[section] = []SectionItem{}
	}
	for _, item := range items {
		value := interface{}(item.Value)
		if item.Section == models.DynamicConfigSectionTabdeal && item.Key == models.DynamicConfigKeyAuthToken {
			value = map[string]interface{}{"configured": item.Value != "", "secret": true}
		}
		result[item.Section] = append(result[item.Section], SectionItem{
			Name:  item.Key,
			Value: value,
			Type:  configValueType(item.Key),
		})
	}
	return result, nil
}

func configValueType(key string) string {
	switch key {
	case models.DynamicConfigKeyAuthToken:
		return configTypeSecret
	case "enabled":
		return configTypeBoolean
	case "signal_pipeline_count", "provider_id", "report_chat_id", "schedule_hour", "schedule_min":
		return configTypeInteger
	case "check_interval", "stale_duration", "proxy_timeout", "sync_interval", "execution_iteration_duration", "request_interval":
		return configTypeDuration
	default:
		return configTypeString
	}
}

// func (s *Service) UpdateSection(ctx context.Context, section string, values SectionValues) error {
// 	if len(values) == 0 {
// 		return fmt.Errorf("values are required")
// 	}
// 	if !s.knownSection(section) {
// 		return fmt.Errorf("unknown section %q", section)
// 	}
// 	if strings.HasPrefix(section, "worker.") {
// 		if _, ok := values["enabled"]; ok {
// 			return fmt.Errorf("use the worker enabled endpoint to change enabled")
// 		}
// 	}
// 	if err := s.validate(section, values); err != nil {
// 		return err
// 	}

// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	if err := s.repo.UpsertMany(ctx, section, values); err != nil {
// 		return fmt.Errorf("persist config section: %w", err)
// 	}
// 	return s.apply(section, values, true)
// }

// func (s *Service) UpdateTabdealAuthToken(ctx context.Context, token string) error {
// 	return s.UpdateSection(ctx, models.DynamicConfigSectionTabdeal, SectionValues{models.DynamicConfigKeyAuthToken: token})
// }

// func (s *Service) SetWorkerEnabled(ctx context.Context, name string, enabled bool) error {
// 	if s.workerFactory == nil {
// 		return fmt.Errorf("worker factory is not initialized")
// 	}
// 	section := worker.SectionForWorker(name)
// 	if !s.knownSection(section) {
// 		return fmt.Errorf("worker %q not found", name)
// 	}

// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	if err := s.workerFactory.SetEnabled(ctx, name, enabled); err != nil {
// 		return err
// 	}
// 	value := strconv.FormatBool(enabled)
// 	if err := s.repo.UpsertMany(ctx, section, map[string]string{"enabled": value}); err != nil {
// 		_ = s.workerFactory.SetEnabled(ctx, name, !enabled)
// 		return fmt.Errorf("persist worker state: %w", err)
// 	}
// 	return nil
// }

// func (s *Service) ListWorkers() ([]worker.WorkerInfo, error) {
// 	if s.workerFactory == nil {
// 		return nil, fmt.Errorf("worker factory is not initialized")
// 	}
// 	return s.workerFactory.ListWorkerInfo(), nil
// }

func (s *Service) knownSection(section string) bool {
	if section == models.DynamicConfigSectionTabdeal {
		return true
	}
	for _, name := range worker.WorkerNames() {
		if section == worker.SectionForWorker(name) {
			return true
		}
	}
	return false
}

// func (s *Service) validate(section string, values SectionValues) error {
// 	// allowed := allowedKeys(section)
// 	for key, value := range values {
// 		if _, ok := allowed[key]; !ok {
// 			return fmt.Errorf("unknown key %q for section %q", key, section)
// 		}
// 		if strings.TrimSpace(value) == "" && !(section == worker.SectionForWorker("stock-market-sync") && (key == "historical_start_date" || key == "historical_end_date")) {
// 			return fmt.Errorf("value for %q is required", key)
// 		}
// 	}
// 	if section == models.DynamicConfigSectionTabdeal {
// 		return nil
// 	}
// 	for key, value := range values {
// 		switch key {
// 		case "enabled":
// 			if _, err := strconv.ParseBool(value); err != nil {
// 				return fmt.Errorf("%s must be boolean", key)
// 			}
// 		case "check_interval", "stale_duration", "proxy_timeout", "sync_interval", "execution_iteration_duration", "request_interval":
// 			if d, err := time.ParseDuration(value); err != nil || d < 0 {
// 				return fmt.Errorf("%s must be a valid non-negative duration", key)
// 			}
// 		case "signal_pipeline_count", "provider_id", "report_chat_id":
// 			if n, err := strconv.ParseUint(value, 10, 64); err != nil || (key == "signal_pipeline_count" && n == 0) || (key == "report_chat_id" && n == 0) {
// 				return fmt.Errorf("%s must be a positive integer", key)
// 			}
// 		case "schedule_hour":
// 			n, err := strconv.Atoi(value)
// 			if err != nil || n < 0 || n > 23 {
// 				return fmt.Errorf("schedule_hour must be between 0 and 23")
// 			}
// 		case "schedule_min":
// 			n, err := strconv.Atoi(value)
// 			if err != nil || n < 0 || n > 59 {
// 				return fmt.Errorf("schedule_min must be between 0 and 59")
// 			}
// 		case "mode":
// 			if value != "daily" && value != "historical" {
// 				return fmt.Errorf("mode must be daily or historical")
// 			}
// 		}
// 	}
// 	return nil
// }

// func allowedKeys(section string) map[string]struct{} {
// 	keys := map[string][]string{
// 		models.DynamicConfigSectionTabdeal: {"auth_token"},
// 		"worker.candle-polling":            {"enabled"},
// 		"worker.market-sync":               {"enabled", "sync_interval"},
// 		"worker.proxy-health-checker":      {"enabled", "check_interval", "stale_duration", "proxy_timeout"},
// 		"worker.connection-monitor":        {"enabled", "check_interval"},
// 		"worker.indicator-calculator":      {"enabled", "execution_iteration_duration", "signal_pipeline_count"},
// 		"worker.stock-market-sync":         {"enabled", "provider_id", "mode", "schedule_hour", "schedule_min", "historical_start_date", "historical_end_date", "request_interval"},
// 		"worker.tse-market-history-sync":   {"enabled", "request_interval"},
// 		"worker.tse-market-daily-sync":     {"enabled", "schedule_hour", "schedule_min", "request_interval", "report_chat_id"},
// 	}
// 	result := make(map[string]struct{}, len(keys[section]))
// 	for _, key := range keys[section] {
// 		result[key] = struct{}{}
// 	}
// 	return result
// }

// func (s *Service) apply(section string, values SectionValues, restart bool) error {
// 	if err := s.validate(section, values); err != nil {
// 		return err
// 	}
// 	if section == models.DynamicConfigSectionTabdeal {
// 		if token, ok := values[models.DynamicConfigKeyAuthToken]; ok {
// 			s.tabdealAdapter.SetAuthToken(strings.TrimSpace(token))
// 		}
// 		return nil
// 	}
// 	if s.workerFactory == nil {
// 		return nil
// 	}
// 	cfg := s.workerFactory.Config()
// 	name := strings.TrimPrefix(section, "worker.")
// 	for key, value := range values {
// 		switch section + "/" + key {
// 		case "worker.candle-polling/enabled":
// 			cfg.CandlePollingEnabled, _ = strconv.ParseBool(value)
// 		case "worker.market-sync/enabled":
// 			cfg.MarketSyncEnabled, _ = strconv.ParseBool(value)
// 		case "worker.market-sync/sync_interval":
// 			cfg.MarketSync.SyncInterval, _ = time.ParseDuration(value)
// 		case "worker.proxy-health-checker/enabled":
// 			cfg.ProxyHealthCheckerEnabled, _ = strconv.ParseBool(value)
// 		case "worker.proxy-health-checker/check_interval":
// 			cfg.ProxyHealthChecker.CheckInterval, _ = time.ParseDuration(value)
// 		case "worker.proxy-health-checker/stale_duration":
// 			cfg.ProxyHealthChecker.StaleDuration, _ = time.ParseDuration(value)
// 		case "worker.proxy-health-checker/proxy_timeout":
// 			cfg.ProxyHealthChecker.ProxyTimeout, _ = time.ParseDuration(value)
// 		case "worker.connection-monitor/enabled":
// 			cfg.ConnectionMonitorEnabled, _ = strconv.ParseBool(value)
// 		case "worker.connection-monitor/check_interval":
// 			cfg.ConnectionMonitor.CheckInterval, _ = time.ParseDuration(value)
// 		case "worker.indicator-calculator/enabled":
// 			cfg.IndicatorCalculatorEnabled, _ = strconv.ParseBool(value)
// 		case "worker.indicator-calculator/execution_iteration_duration":
// 			cfg.IndicatorCalculator.ExecutionIttrationDuration, _ = time.ParseDuration(value)
// 		case "worker.indicator-calculator/signal_pipeline_count":
// 			n, _ := strconv.ParseUint(value, 10, 64)
// 			cfg.IndicatorCalculator.SignalPipelineCount = uint(n)
// 		case "worker.stock-market-sync/enabled":
// 			cfg.StockMarketSyncEnabled, _ = strconv.ParseBool(value)
// 		case "worker.stock-market-sync/provider_id":
// 			n, _ := strconv.ParseUint(value, 10, 64)
// 			cfg.StockMarketSync.ProviderID = uint(n)
// 		case "worker.stock-market-sync/mode":
// 			cfg.StockMarketSync.Mode = value
// 		case "worker.stock-market-sync/schedule_hour":
// 			cfg.StockMarketSync.ScheduleHour, _ = strconv.Atoi(value)
// 		case "worker.stock-market-sync/schedule_min":
// 			cfg.StockMarketSync.ScheduleMin, _ = strconv.Atoi(value)
// 		case "worker.stock-market-sync/historical_start_date":
// 			cfg.StockMarketSync.HistoricalStartDate = value
// 		case "worker.stock-market-sync/historical_end_date":
// 			cfg.StockMarketSync.HistoricalEndDate = value
// 		case "worker.stock-market-sync/request_interval":
// 			cfg.StockMarketSync.RequestInterval, _ = time.ParseDuration(value)
// 		case "worker.tse-market-history-sync/enabled":
// 			cfg.TSEMarketHistorySyncEnabled, _ = strconv.ParseBool(value)
// 		case "worker.tse-market-history-sync/request_interval":
// 			cfg.TSEMarketHistorySync.RequestInterval, _ = time.ParseDuration(value)
// 		// case "worker.tse-market-daily-sync/enabled":
// 		// 	cfg.TSEMarketDailySyncEnabled, _ = strconv.ParseBool(value)
// 		case "worker.tse-market-daily-sync/schedule_hour":
// 			cfg.TSEMarketDailySync.ScheduleHour, _ = strconv.Atoi(value)
// 		case "worker.tse-market-daily-sync/schedule_min":
// 			cfg.TSEMarketDailySync.ScheduleMin, _ = strconv.Atoi(value)
// 		case "worker.tse-market-daily-sync/request_interval":
// 			cfg.TSEMarketDailySync.RequestInterval, _ = time.ParseDuration(value)
// 		case "worker.tse-market-daily-sync/report_chat_id":
// 			cfg.TSEMarketDailySync.ReportChatID, _ = strconv.ParseInt(value, 10, 64)
// 		}
// 	}
// 	if restart {
// 		return s.workerFactory.RestartWorker(context.Background(), name)
// 	}
// 	return nil
// }

// func (s *Service) defaults(adapterCfg configs.Tabdeal, cfg *configs.Workers) map[string]map[string]string {
// 	if cfg.StockMarketSync == nil {
// 		cfg.StockMarketSync = &configs.StockMarketSync{Mode: "daily", ScheduleHour: 16, RequestInterval: time.Minute}
// 	}
// 	if cfg.TSEMarketHistorySync == nil {
// 		cfg.TSEMarketHistorySync = &configs.TSEMarketHistorySync{RequestInterval: 250 * time.Millisecond}
// 	}
// 	if cfg.TSEMarketDailySync == nil {
// 		cfg.TSEMarketDailySync = &configs.TSEMarketDailySync{ScheduleHour: 19, RequestInterval: 250 * time.Millisecond, ReportChatID: 1689926}
// 	}
// 	if cfg.MarketSync == nil {
// 		cfg.MarketSync = &configs.MarketSync{SyncInterval: time.Hour}
// 	}
// 	if cfg.ProxyHealthChecker == nil {
// 		cfg.ProxyHealthChecker = &configs.ProxyHealthChecker{CheckInterval: time.Minute, StaleDuration: 24 * time.Hour, ProxyTimeout: 10 * time.Second}
// 	}
// 	if cfg.ConnectionMonitor == nil {
// 		cfg.ConnectionMonitor = &configs.ConnectionMonitor{CheckInterval: 30 * time.Second}
// 	}
// 	if cfg.IndicatorCalculator.SignalPipelineCount == 0 {
// 		cfg.IndicatorCalculator.SignalPipelineCount = 1
// 	}
// 	return map[string]map[string]string{
// 		models.DynamicConfigSectionTabdeal: {"auth_token": adapterCfg.AuthToken},
// 		"worker.candle-polling":            {"enabled": strconv.FormatBool(cfg.CandlePollingEnabled)},
// 		"worker.market-sync":               {"enabled": strconv.FormatBool(cfg.MarketSyncEnabled), "sync_interval": cfg.MarketSync.SyncInterval.String()},
// 		"worker.proxy-health-checker":      {"enabled": strconv.FormatBool(cfg.ProxyHealthCheckerEnabled), "check_interval": cfg.ProxyHealthChecker.CheckInterval.String(), "stale_duration": cfg.ProxyHealthChecker.StaleDuration.String(), "proxy_timeout": cfg.ProxyHealthChecker.ProxyTimeout.String()},
// 		"worker.connection-monitor":        {"enabled": strconv.FormatBool(cfg.ConnectionMonitorEnabled), "check_interval": cfg.ConnectionMonitor.CheckInterval.String()},
// 		"worker.indicator-calculator":      {"enabled": strconv.FormatBool(cfg.IndicatorCalculatorEnabled), "execution_iteration_duration": cfg.IndicatorCalculator.ExecutionIttrationDuration.String(), "signal_pipeline_count": strconv.FormatUint(uint64(cfg.IndicatorCalculator.SignalPipelineCount), 10)},
// 		"worker.stock-market-sync":         {"enabled": strconv.FormatBool(cfg.StockMarketSyncEnabled), "provider_id": strconv.FormatUint(uint64(cfg.StockMarketSync.ProviderID), 10), "mode": cfg.StockMarketSync.Mode, "schedule_hour": strconv.Itoa(cfg.StockMarketSync.ScheduleHour), "schedule_min": strconv.Itoa(cfg.StockMarketSync.ScheduleMin), "historical_start_date": cfg.StockMarketSync.HistoricalStartDate, "historical_end_date": cfg.StockMarketSync.HistoricalEndDate, "request_interval": cfg.StockMarketSync.RequestInterval.String()},
// 		"worker.tse-market-history-sync":   {"enabled": strconv.FormatBool(cfg.TSEMarketHistorySyncEnabled), "request_interval": cfg.TSEMarketHistorySync.RequestInterval.String()},
// 		// "worker.tse-market-daily-sync":     {"enabled": strconv.FormatBool(cfg.TSEMarketDailySyncEnabled), "schedule_hour": strconv.Itoa(cfg.TSEMarketDailySync.ScheduleHour), "schedule_min": strconv.Itoa(cfg.TSEMarketDailySync.ScheduleMin), "request_interval": cfg.TSEMarketDailySync.RequestInterval.String(), "report_chat_id": strconv.FormatInt(cfg.TSEMarketDailySync.ReportChatID, 10)},
// 	}
// }
