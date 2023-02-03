package main

import (
	"context"

	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	config.LoadConfig("NFT_INDEXER")

	ctx := context.Background()

	wsClient, err := ethclient.Dial(viper.GetString("ethereum.ws_url"))
	if err != nil {
		log.Panic(err.Error(), zap.Error(err))
	}

	// connect to the processor
	conn, err := grpc.Dial(viper.GetString("server.address"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Sugar().Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := processor.NewEventProcessorClient(conn)
	ethereumEventsEmitter := NewEthereumEventsEmitter(wsClient, c)
	ethereumEventsEmitter.Run(ctx)

	log.Info("Ethereum Emitter terminated")
}
