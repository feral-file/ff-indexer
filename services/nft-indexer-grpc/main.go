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

func main() {
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER_GRPC")

	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	indexerServer, err := NewIndexerGRPCServer(
		viper.GetString("grpc.network"),
		viper.GetInt("grpc.port"),
		indexerStore,
	)
	if err != nil {
		log.Panic("fail to initiate indexer GRPC server", zap.Error(err))
	}

	if err := indexerServer.Run(ctx); err != nil {
		log.Panic("fail to run indexer GRPC server", zap.Error(err))
	}
}
