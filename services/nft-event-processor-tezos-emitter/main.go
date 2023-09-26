package main

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/config-loader/external/aws/ssm"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/bitmark-inc/tzkt-go"
)

func main() {
	config.LoadConfig("NFT_INDEXER")
	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	ctx := context.Background()

	parameterStore, err := ssm.NewParameterStore(ctx)
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
	tezosEventsEmitter := NewTezosEventsEmitter(viper.GetString("tzkt.transferLastBlockKeyName"), viper.GetString("tzkt.bigmapLastBlockKeyName"),
		parameterStore, c, viper.GetString("tzkt.ws_url"), tzkt.New(viper.GetString("tzkt.network")))
	tezosEventsEmitter.Run(ctx)

	log.Info("Tezos Emitter terminated")
}
