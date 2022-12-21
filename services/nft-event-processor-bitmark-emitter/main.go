package main

import (
	"context"
	"log"
	"net/http"
	"time"

	bitmarksdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	config.LoadConfig("NFT_INDEXER")

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
		logrus.WithError(err).Panic("fail to initiate bitmark listener")
	}

	// connect to the processor
	conn, err := grpc.Dial(viper.GetString("event_processor_server.address"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := processor.NewEventProcessorClient(conn)

	bitmarkEventsEmitter := New(bitmarkListener, c)

	if err := bitmarkEventsEmitter.Watch(); err != nil {
		panic(err)
	}

	bitmarkEventsEmitter.Run(ctx)

	logrus.Info("Ethereum Emitter terminated")
}
