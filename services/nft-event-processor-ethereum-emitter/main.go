package main

import (
	"context"
	"fmt"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/config-loader/external/aws/ssm"
	"github.com/bitmark-inc/nft-indexer/cache"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
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

	ctx := context.Background()

	wsClient, err := ethclient.Dial(viper.GetString("ethereum.ws_url"))
	if err != nil {
		log.Panic(err.Error(), zap.Error(err))
	}

	parameterStore, err := ssm.NewParameterStore(ctx)
	if err != nil {
		log.Panic("can not create new parameter store", zap.Error(err))
	}

	cacheStore, err := cache.NewMongoDBCacheStore(ctx, viper.GetString("cache_store.db_uri"), viper.GetString("cache_store.db_name"))
	if err != nil {
		log.Panic("fail to initiate cache store", zap.Error(err))
	}

	// connect to the processor
	conn, err := grpc.Dial(viper.GetString("event_processor_server.address"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Sugar().Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := processor.NewEventProcessorClient(conn)
	ethereumEventsEmitter := NewEthereumEventsEmitter(viper.GetString("ethereum.lastBlockKeyName"), wsClient, parameterStore, cacheStore, c)
	ethereumEventsEmitter.Run(ctx)

	log.Info("Ethereum Emitter terminated")
}
