package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	bitmarksdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
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

	ctx := context.Background()

	bitmarksdk.Init(&bitmarksdk.Config{
		Network: bitmarksdk.Network(viper.GetString("network.bitmark")),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		APIToken: viper.GetString("bitmarksdk.apikey"),
	})

	bitmarkListener, err := NewListener(viper.GetString("bitmark.db_uri"))
	if err != nil {
		log.Panic("fail to initiate bitmark listener", zap.Error(err))
	}

	// connect to the processor
	conn, err := grpc.Dial(viper.GetString("event_processor_server.address"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Sugar().Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := processor.NewEventProcessorClient(conn)

	bitmarkEventsEmitter := New(bitmarkListener, c)

	if err := bitmarkEventsEmitter.Watch(); err != nil {
		panic(err)
	}

	bitmarkEventsEmitter.Run(ctx)

	log.Info("Bitmark Emitter terminated")
}
