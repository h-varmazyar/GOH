package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/signal/definition"
	"github.com/h-varmazyar/gatec/internal/worker/events"
	"github.com/h-varmazyar/gatec/internal/worker/workers"
	gormext "github.com/h-varmazyar/gatec/pkg/gormOld"
	"github.com/sirupsen/logrus"
)

func main() {
	flags := flag.NewFlagSet("signal-replay", flag.ExitOnError)
	strategy := flags.String("strategy", "", "strategy ID or YAML file path")
	market := flags.String("market", "crypto", "market alias")
	symbols := flags.String("symbols", "", "comma-separated symbols; empty means all configured symbols")
	from := flags.String("from", "", "inclusive start, YYYY-MM-DD or RFC3339")
	to := flags.String("to", "", "inclusive end, YYYY-MM-DD or RFC3339")
	source := flags.String("source", "", "strategy source: db, file, or both")
	fee := flags.Float64("fee", 0.1, "fee percent per side")
	slippage := flags.Float64("slippage", 0.05, "entry slippage percent")
	split := flags.Float64("split", 0.7, "in-sample fraction")
	_ = flags.Parse(os.Args[1:])
	if *strategy == "" || *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/signal-replay --strategy <id|file.yaml> --from <date> --to <date> [--market crypto --symbols BTCUSDT,ETHUSDT --fee 0.1 --slippage 0.05 --split 0.7]")
		os.Exit(2)
	}
	start, err := parseTime(*from, false)
	if err != nil {
		logrus.WithError(err).Fatal("invalid --from")
	}
	end, err := parseTime(*to, true)
	if err != nil {
		logrus.WithError(err).Fatal("invalid --to")
	}
	cfg, err := configs.Load()
	if err != nil {
		logrus.WithError(err).Fatal("load configuration")
	}
	strategyID := *strategy
	strategySource := strings.ToLower(strings.TrimSpace(*source))
	if strings.HasSuffix(strings.ToLower(strategyID), ".yaml") || strings.HasSuffix(strings.ToLower(strategyID), ".yml") {
		path, absErr := filepath.Abs(strategyID)
		if absErr != nil {
			logrus.WithError(absErr).Fatal("resolve strategy path")
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			logrus.WithError(readErr).Fatal("read strategy YAML")
		}
		parsed, parseErr := definition.ParseYAML(data)
		if parseErr != nil {
			logrus.WithError(parseErr).Fatal("parse strategy YAML")
		}
		strategyID = parsed.ID
		cfg.Workers.SignalWorker.StrategiesDir = filepath.Dir(path)
		if strategySource == "" {
			strategySource = "file"
		}
	} else if strategySource == "" {
		strategySource = "db"
	}
	cfg.Workers.SignalWorker.StrategySource = strategySource
	db, err := gormext.Open(cfg.Postgres)
	if err != nil {
		logrus.WithError(err).Fatal("open database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		logrus.WithError(err).Fatal("get database handle")
	}
	defer sqlDB.Close()
	worker := workers.NewSignalWorker(cfg.Workers.SignalWorker, events.NewChannelBus(), repositories.NewMarketRepository(db), repositories.NewProviderRepository(db), repositories.NewCandleRepository(db), repositories.NewIndicatorDefinitionRepository(db), repositories.NewStrategyDefinitionRepository(db), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	report, err := worker.Replay(ctx, workers.ReplayOptions{StrategyID: strategyID, Market: *market, Symbols: splitSymbols(*symbols), From: start, To: end, FeePercent: *fee, SlippagePercent: *slippage, Split: *split})
	if err != nil {
		logrus.WithError(err).Fatal("run signal replay")
	}
	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		logrus.WithError(err).Fatal("encode replay report")
	}
	fmt.Println(string(output))
}

func parseTime(value string, end bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	if end {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return parsed.UTC(), nil
}

func splitSymbols(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
