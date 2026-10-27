package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto/tabdeal"
	"github.com/h-varmazyar/gatec/internal/consumers"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	airdropService "github.com/h-varmazyar/gatec/internal/services/airdrop"
	tgAccountService "github.com/h-varmazyar/gatec/internal/services/tgaccount"
	gormext "github.com/h-varmazyar/gatec/pkg/gormOld"
	"github.com/h-varmazyar/gatec/pkg/logger"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := configs.Load()
	if err != nil {
		logrus.WithError(err).Fatal("load configuration")
	}
	logFile, err := logger.Configure(cfg.LogFile, cfg.LogLevel)
	if err != nil {
		logrus.WithError(err).Fatal("configure airdrop catcher logger")
	}
	defer logFile.Close()

	db, err := gormext.Open(cfg.Postgres)
	if err != nil {
		logrus.WithError(err).Fatal("connect to database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		logrus.WithError(err).Fatal("get database connection")
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(
		&models.TabdealAirdrop{},
		&models.TabdealAirdropGift{},
		&models.TelegramAccount{},
		&models.TelegramSubscriptionFilter{},
		&models.Proxy{},
	); err != nil {
		logrus.WithError(fmt.Errorf("migration failed: %w", err)).Fatal("migrate airdrop catcher models")
	}

	telegramRepo := repositories.NewTelegramAccountRepository(db)
	filterRepo := repositories.NewTelegramSubscriptionFilterRepository(db)
	airdropRepo := repositories.NewWallexAirdropRepository(db)
	giftAllocationRepo := repositories.NewTabdealAllocationRepository(db)
	userRepo := repositories.NewUserRepository(db)
	proxyRepo := repositories.NewProxyRepository(db)
	telegramService := tgAccountService.NewService(cfg.Services.TgAccount, telegramRepo, filterRepo, proxyRepo)
	service := airdropService.NewService(
		cfg.Services.AirdropService,
		airdropRepo,
		giftAllocationRepo,
		userRepo,
		tabdeal.NewAdapter(&cfg.Adapters.Tabdeal),
		telegramService,
	)

	if err := telegramService.LoadAccounts(ctx); err != nil {
		logrus.WithError(err).Fatal("load telegram accounts")
	}
	consumers.NewAirdropCatcher(telegramService, service).Start(ctx)

	<-ctx.Done()
}
