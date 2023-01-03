package main

import (
	"context"
	"log"

	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	config.LoadConfig("NFT_INDEXER")

	ctx := context.Background()

	wsClient, err := ethclient.Dial(viper.GetString("ethereum.ws_url"))
	if err != nil {
		logrus.WithError(err).Panic(err)
	}

	// connect to the processor
	conn, err := grpc.Dial(viper.GetString("event_processor_server.address"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := processor.NewEventProcessorClient(conn)
	ethereumEventsEmitter := NewEthereumEventsEmitter(wsClient, c)
	ethereumEventsEmitter.Run(ctx)

	logrus.Info("Ethereum Emitter terminated")
}
