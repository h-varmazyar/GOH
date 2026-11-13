// Command worker runs one or more background workers selected by name.
//
// Usage:
//
//	go run ./cmd/worker candle-polling
//	go run ./cmd/worker candle-polling market-sync
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/worker"
	clickhouseClient "github.com/h-varmazyar/gatec/pkg/clickhouse"
	"github.com/h-varmazyar/gatec/pkg/logger"
	"github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := configs.Load()
	if err != nil {
		logrus.WithError(err).Fatal("load configuration")
	}
	logFile, err := logger.Configure(cfg.LogFile, cfg.LogLevel)
	if err != nil {
		logrus.WithError(err).Fatal("configure worker logger")
	}
	defer logFile.Close()

	names, collectionMethod, insCodes, err := parseWorkerArgs(os.Args[1:])
	if err != nil {
		logrus.WithError(err).Fatal("parse worker arguments")
	}
	if len(names) == 0 {
		printUsage()
		os.Exit(1)
	}
	if collectionMethod != "" {
		cfg.Workers.TSEMarketDataSync.CollectionMethod = collectionMethod
	}
	if insCodes != "" {
		cfg.Workers.TSEMarketDataSync.InsCodes = splitInsCodes(insCodes)
	}
	if cfg.Workers.TSEMarketDataSync.CollectionMethod == "ins_codes" && len(cfg.Workers.TSEMarketDataSync.InsCodes) == 0 {
		logrus.Fatal("--ins-codes is required when --collection-method=ins_codes")
	}
	var ch *clickhouseClient.Client
	needsTradeHistory := false
	for _, name := range names {
		if name == "tse-trade-history" ||
			(name == "tse-market-data-sync" &&
				(cfg.Workers.TSEMarketDataSync.Mode == "" || cfg.Workers.TSEMarketDataSync.Mode == consts.TSEMarketDataToday)) {
			needsTradeHistory = true
			break
		}
	}
	if needsTradeHistory {
		ch, err = clickhouseClient.New(cfg.ClickHouse)
		if err != nil {
			logrus.WithError(err).Fatal("initialize ClickHouse client")
		}
		defer ch.Close()
	}

	factory, closeDependencies, err := worker.StandaloneFactory(ctx, cfg, ch, names...)
	if err != nil {
		logrus.WithError(err).Fatal("initialize worker dependencies")
	}
	defer closeDependencies()

	if err := factory.CreateWorkers(names...); err != nil {
		logrus.WithError(err).Fatal("create workers")
	}
	for _, name := range names {
		if err := factory.StartWorker(ctx, name); err != nil {
			logrus.WithError(err).WithField("worker", name).Fatal("start worker")
		}
	}
	// tse-trade-history keeps running and takes the jobs queued from the panel.
	if len(names) == 1 && names[0] == "tse-market-data-sync" && cfg.Workers.TSEMarketDataSync.Mode != "today" {
		if err := factory.WaitWorker(ctx, names[0]); err != nil && ctx.Err() == nil {
			logrus.WithError(err).Error("one-shot worker failed")
		}
		return
	}

	<-ctx.Done()
	if err := factory.StopAllWorkers(); err != nil {
		logrus.WithError(err).Error("stop workers")
	}
}

func parseWorkerArgs(args []string) ([]string, string, string, error) {
	names := make([]string, 0, len(args))
	method, codes := "", ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--collection-method="):
			method = strings.TrimPrefix(arg, "--collection-method=")
		case arg == "--collection-method":
			if i+1 == len(args) {
				return nil, "", "", fmt.Errorf("--collection-method requires a value")
			}
			i++
			method = args[i]
		case strings.HasPrefix(arg, "--ins-codes="):
			codes = strings.TrimPrefix(arg, "--ins-codes=")
		case arg == "--ins-codes":
			if i+1 == len(args) {
				return nil, "", "", fmt.Errorf("--ins-codes requires a comma-separated value")
			}
			i++
			codes = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, "", "", fmt.Errorf("unknown option %q", arg)
			}
			names = append(names, arg)
		}
	}
	if method != "" && method != "all" && method != "missed" && method != "ins_codes" {
		return nil, "", "", fmt.Errorf("invalid collection method %q", method)
	}
	return names, method, codes, nil
}

func splitInsCodes(value string) []string {
	items := []string{}
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func printUsage() {
	workers := worker.WorkerNames()
	fmt.Fprintf(os.Stderr, "usage: worker <worker-name> [worker-name ...] [--collection-method all|missed|ins_codes] [--ins-codes CODE[,CODE...]]\navailable workers: %v\n", workers)
}
