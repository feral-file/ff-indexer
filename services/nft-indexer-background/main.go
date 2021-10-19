package main

import (
	"context"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/artblocks"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

var CadenceService = "cadence-frontend"

func main() {
	config.LoadConfig("NFT_INDEXER")

	hostPort := viper.GetString("cadence.host_port")
	logLevel := viper.GetInt("cadence.log_level")

	network := viper.GetString("network")

	ctx := context.Background()

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.WithError(err).Panic("fail to initiate indexer store")
	}

	worker := indexerWorker.New(network, artblocks.New(), indexerStore)

	// workflows
	workflow.Register(worker.IndexArtblocksTokenWorkflow)

	// artblocks
	activity.Register(worker.GetOwnedERC721TokenIDByContract)
	activity.Register(worker.IndexTokenDataFromArtblocks)

	// index store
	activity.Register(worker.IndexAsset)

	workerServiceClient := cadence.BuildCadenceServiceClient(hostPort, indexerWorker.ClientName, CadenceService)
	workerLogger := cadence.BuildCadenceLogger(logLevel)
	cadence.StartWorker(workerLogger, workerServiceClient, viper.GetString("cadence.domain"), indexerWorker.TaskListName)
}
