package main

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/bitmark-inc/autonomy-account/storage"
	log "github.com/bitmark-inc/autonomy-logger"
	notification "github.com/bitmark-inc/autonomy-notification/sdk"
	"github.com/bitmark-inc/config-loader"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	indexerGRPCSDK "github.com/bitmark-inc/nft-indexer/sdk/nft-indexer-grpc"
)

func main() {
	// FIXME: add context for graceful shutdown
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	environment := viper.GetString("environment")

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		log.Panic("Sentry initialization failed", zap.Error(err))
	}

	db, err := gorm.Open(postgres.Open(viper.GetString("store.dsn")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(viper.GetInt("store.log_level"))),
	})
	if err != nil {
		panic(err)
	}

	store := NewPostgresEventStore(db)

	if err := store.AutoMigrate(); err != nil {
		panic(err)
	}

	accountDb, err := gorm.Open(postgres.Open(viper.GetString("account.db_uri")))
	if err != nil {
		log.Fatal("fail to connect database", zap.Error(err))
	}

	accountStore := storage.NewAccountInformationStorage(accountDb)

	indexerGRPC, err := indexerGRPCSDK.NewIndexerClient(viper.GetString("indexer_grpc.endpoint"))
	if err != nil {
		log.Fatal("fail to connect indexer grpc", zap.Error(err))
	}

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	notification := notification.New(viper.GetString("notification.endpoint"), nil)

	feedServer := NewFeedClient(viper.GetString("feed.endpoint"), viper.GetString("feed.api_token"), viper.GetBool("feed.debug"))

	checkInterval, err := time.ParseDuration(viper.GetString("default_check_interval"))
	if err != nil {
		log.Warn("invalid check interval. set to default 10s",
			zap.String("check_interval", viper.GetString("check_interval")), zap.Error(err))
		checkInterval = DefaultCheckInterval
	}

	p := NewEventProcessor(
		environment,
		checkInterval,
		viper.GetString("server.network"),
		viper.GetString("server.address"),
		NewPostgresEventStore(db),
		indexerGRPC,
		cadenceClient,
		accountStore,
		notification,
		feedServer,
	)
	p.Run(ctx)
}
