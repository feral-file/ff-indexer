package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	bitmarksdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
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

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	// connect to the processor
	addr := flag.String("addr", viper.GetString("event_processor_server.address"), "the address to connect to")
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
