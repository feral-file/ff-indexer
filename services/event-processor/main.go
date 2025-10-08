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

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/config-loader"

	indexer "github.com/feral-file/ff-indexer"
	indexerWorker "github.com/feral-file/ff-indexer/background/worker"
	"github.com/feral-file/ff-indexer/cadence"
	grpcGateway "github.com/feral-file/ff-indexer/sdk/grpc-gateway"
)

func main() {
	// FIXME: add context for graceful shutdown
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	environment := viper.GetString("environment")
	if err := log.Initialize(viper.GetBool("debug"), &sentry.ClientOptions{
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

	indexerGRPC, err := grpcGateway.NewGRPCClient(viper.GetString("indexer_grpc.endpoint"))
	if err != nil {
		log.Fatal("fail to connect indexer grpc", zap.Error(err))
	}

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("indexer.store.db_uri"), viper.GetString("indexer.store.db_name"), environment)
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	checkInterval, err := time.ParseDuration(viper.GetString("default_check_interval"))
	if err != nil {
		log.WarnWithContext(ctx, "invalid check interval. set to default 10s",
			zap.Int64("check_interval", int64(checkInterval)), zap.Error(err))
		checkInterval = DefaultCheckInterval
	}

	eventExpiryDays := viper.GetInt64("events.expiry_days")
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
		viper.GetString("contract.series_registry"),
		viper.GetStringSlice("ipfs.preferred_gateways"),
		checkInterval,
		eventExpiryDuration,
		viper.GetString("server.network"),
		viper.GetString("server.address"),
		NewPostgresEventStore(db),
		indexerGRPC,
		cadenceClient,
		indexerStore,
		rpcClient,
	)
	p.Run(ctx)
}
