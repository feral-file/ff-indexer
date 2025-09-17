package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/config-loader"
	indexer "github.com/feral-file/ff-indexer"
	"github.com/feral-file/ff-indexer/cache"
)

func main() {
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	if err := log.Initialize(viper.GetBool("debug"), nil); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	environment := viper.GetString("environment")
	store, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"), environment)
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}
	cache, err := cache.NewMongoDBCacheStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.Panic("fail to initiate cache store", zap.Error(err))
	}
	ethClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		log.Panic("fail to initiate eth client: %s", zap.Error(err))
	}

	server, err := NewServer(
		viper.GetString("server.grpc_network"),
		viper.GetInt("server.grpc_port"),
		store,
		cache,
		ethClient,
	)
	if err != nil {
		log.Panic("fail to initiate indexer GRPC server", zap.Error(err))
	}

	if err := server.Run(ctx); err != nil {
		log.Panic("fail to run indexer GRPC server", zap.Error(err))
	}
}
