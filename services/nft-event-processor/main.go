package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
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
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	indexerGRPCSDK "github.com/bitmark-inc/nft-indexer/sdk/nft-indexer-grpc"
)

func main() {
	// FIXME: add context for graceful shutdown
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	environment := viper.GetString("environment")
	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug"), &sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
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

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"), environment)
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
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

	eventExpiryDays := viper.GetInt64("event_expiry_days")
	if eventExpiryDays <= 0 {
		eventExpiryDays = DefaultEventExpiryDays
	}
	eventExpiryDuration := time.Duration(eventExpiryDays) * time.Hour * 24

	rpcClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		log.Panic(err.Error(), zap.Error(err))
	}

	p := NewEventProcessor(
		environment,
		checkInterval,
		eventExpiryDuration,
		viper.GetString("server.network"),
		viper.GetString("server.address"),
		NewPostgresEventStore(db),
		indexerGRPC,
		cadenceClient,
		accountStore,
		indexerStore,
		notification,
		feedServer,
		rpcClient,
	)
	p.Run(ctx)
}
