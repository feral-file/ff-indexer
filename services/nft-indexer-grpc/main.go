package main

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/log"
)

// implement main function here

func main() {
	// FIXME: add context for graceful shutdown
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER_GRPC")

	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("indexer_store.db_uri"), viper.GetString("indexer_store.db_name"))
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	indexerServer, err := NewIndexerServer(
		viper.GetString("grpc.network"),
		viper.GetInt("grpc.port"),
		indexerStore,
	)
	if err != nil {
		log.Panic("fail to initiate indexer GRPC server", zap.Error(err))
	}

	indexerServer.Run(ctx)
}
