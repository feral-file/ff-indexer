package main

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/bettercall"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
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

	worker := indexerWorker.New(network,
		opensea.New(viper.GetString("network"), viper.GetString("opensea.api_key")),
		bettercall.New(), indexerStore)

	// workflows
	workflow.Register(worker.IndexOpenseaTokenWorkflow)
	workflow.Register(worker.IndexTezosTokenWorkflow)
	workflow.Register(worker.RefreshTokenProvenanceWorkflow)
	workflow.Register(worker.RefreshTokenProvenancePeriodicallyWorkflow)

	// opensea
	activity.Register(worker.IndexTokenDataFromFromOpensea)
	activity.Register(worker.IndexTokenDataFromFromTezos)

	// index store
	activity.Register(worker.IndexAsset)
	activity.Register(worker.GetOutdatedTokens)
	activity.Register(worker.GetTokenIDsByOwner)
	activity.Register(worker.RefreshTokenProvenance)

	workerServiceClient := cadence.BuildCadenceServiceClient(hostPort, indexerWorker.ClientName, CadenceService)
	workerLogger := cadence.BuildCadenceLogger(logLevel)
	cadence.StartWorker(workerLogger, workerServiceClient, viper.GetString("cadence.domain"), indexerWorker.TaskListName)
}
