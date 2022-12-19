package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	ethereum "github.com/bitmark-inc/account-vault-ethereum"
	bitmarksdk "github.com/bitmark-inc/bitmark-sdk-go"
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

	bitmarksdk.Init(&bitmarksdk.Config{
		Network: bitmarksdk.Network(viper.GetString("network.bitmark")),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		APIToken: viper.GetString("bitmarksdk.apikey"),
	})

	w, err := ethereum.NewWalletFromMnemonic(
		viper.GetString("ethereum.worker_account_mnemonic"),
		viper.GetString("network.ethereum"),
		viper.GetString("ethereum.rpc_url"),
	)
	if err != nil {
		logrus.WithError(err).Panic(err)
	}

	wsClient, err := ethclient.Dial(viper.GetString("ethereum.ws_url"))
	if err != nil {
		logrus.WithError(err).Panic(err)
	}

	// connect to the processor
	addr := flag.String("addr", viper.GetString("event_processor_server.address"), "the address to connect to")
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := processor.NewEventProcessorClient(conn)
	ethereumEventsEmitter := NewEthereumEventsEmitter(w, wsClient, c)

	go ethereumEventsEmitter.Watch(ctx)
	ethereumEventsEmitter.Run(ctx)

	logrus.Info("Ethereum Emitter terminated")
}
