package main

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/ens"
	"github.com/bitmark-inc/nft-indexer/externals/feralfile"
	tezosDomain "github.com/bitmark-inc/nft-indexer/externals/tezos-domain"
)

func main() {
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.WithError(err).Panic("fail to initiate indexer store")
	}

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	ensClient := ens.New(viper.GetString("ens.rpc_url"))
	tezosDomain := tezosDomain.New(viper.GetString("tezos.domain_api_url"))
	feralfileClient := feralfile.New(viper.GetString("feralfile.api_url"))

	s := NewNFTIndexerServer(cadenceClient, ensClient, tezosDomain, feralfileClient, indexerStore, viper.GetString("server.api_token"))
	s.SetupRoute()
	if err := s.Run(viper.GetString("server.port")); err != nil {
		log.WithError(err).Panic("server interrupted")
	}
}
