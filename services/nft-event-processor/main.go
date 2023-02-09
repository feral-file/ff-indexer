package main

import (
	"context"
	"fmt"

	"github.com/bitmark-inc/autonomy-account/storage"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/log"

	"go.uber.org/zap"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	notification "github.com/bitmark-inc/autonomy-notification/sdk"
	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
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

	accountDb, err := gorm.Open(postgres.Open(viper.GetString("account.db_uri")))
	if err != nil {
		log.Fatal("fail to connect database", zap.Error(err))
	}

	accountStore := storage.NewAccountInformationStorage(accountDb)

	if err := store.AutoMigrate(); err != nil {
		panic(err)
	}

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("indexer_store.db_uri"), viper.GetString("indexer_store.db_name"))
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	notification := notification.New(viper.GetString("notification.endpoint"), nil)

	feedServer := NewFeedClient(viper.GetString("feed.endpoint"), viper.GetString("feed.api_token"), viper.GetBool("feed.debug"))

	p := NewEventProcessor(
		viper.GetString("server.network"),
		viper.GetString("server.address"),
		NewPostgresEventStore(db),
		indexerStore,
		cadenceClient,
		accountStore,
		notification,
		feedServer,
	)
	p.Run(ctx)
}
