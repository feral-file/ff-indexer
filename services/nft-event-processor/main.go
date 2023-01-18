package main

import (
	"context"

	"github.com/bitmark-inc/autonomy-account/storage"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
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

	environment := viper.GetString("environment")

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		log.Logger.Panic("Sentry initialization failed", zap.Error(err))
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
		log.Logger.Fatal("fail to connect database", zap.Error(err))
	}

	accountStore := storage.NewAccountInformationStorage(accountDb)

	if err := store.AutoMigrate(); err != nil {
		panic(err)
	}

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("indexer_store.db_uri"), viper.GetString("indexer_store.db_name"))
	if err != nil {
		log.Logger.Panic("fail to initiate indexer store", zap.Error(err))
	}

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	indexerEngine := indexer.New(
		environment,
		opensea.New(viper.GetString("network.ethereum"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New(viper.GetString("network.tezos")),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
	)

	notification := notification.New(viper.GetString("notification.endpoint"), nil)

	feedServer := NewFeedClient(viper.GetString("feed.endpoint"), viper.GetString("feed.api_token"), viper.GetBool("feed.debug"))

	p := NewEventProcessor(
		viper.GetString("server.network"),
		viper.GetString("server.address"),
		NewPostgresEventStore(db),
		indexerStore,
		cadenceClient,
		accountStore,
		indexerEngine,
		notification,
		feedServer,
	)
	p.Run(ctx)
}
