package main

import (
	"context"
	"fmt"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/config-loader/external/aws/ssm"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/bitmark-inc/tzkt-go"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	config.LoadConfig("NFT_INDEXER")

	environment := viper.GetString("environment")
	if err := log.Initialize(viper.GetBool("debug"), &sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	ctx := context.Background()

	parameterStore, err := ssm.New(ctx)
	if err != nil {
		log.Panic("can not create new parameter store", zap.Error(err))
	}

	// connect to the processor
	conn, err := grpc.Dial(viper.GetString("event_processor_server.address"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Sugar().Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := processor.NewEventProcessorClient(conn)
	tezosEventsEmitter := NewTezosEventsEmitter(ctx, viper.GetString("tzkt.lastBlockKeyName"), parameterStore, c, viper.GetString("tzkt.ws_url"), tzkt.New(viper.GetString("tzkt.network")))
	tezosEventsEmitter.Run(ctx)

	log.InfoWithContext(ctx, "Tezos Emitter terminated")
}
