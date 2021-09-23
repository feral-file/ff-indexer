package main

import (
	"context"

	"github.com/bitmark-inc/config-loader"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	indexerStore, err := NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.WithError(err).Panic("fail to initiate indexer store")
	}

	s := NewNFTIndexerServer(indexerStore, viper.GetString("server.api_token"))
	s.SetupRoute()
	if err := s.Run(viper.GetString("server.port")); err != nil {
		log.WithError(err).Panic("server interrupted")
	}
}
