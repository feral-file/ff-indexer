package main

import (
	"context"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

var CadenceService = "cadence-frontend"

func main() {
	config.LoadConfig("NFT_INDEXER")

	hostPort := viper.GetString("cadence.host_port")
	logLevel := viper.GetInt("cadence.log_level")

	network := viper.GetString("network")

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: network,
	}); err != nil {
		log.WithError(err).Panic("Sentry initialization failed")
	}

	ctx := context.Background()

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.WithError(err).Panic("fail to initiate indexer store")
	}

	indexerEngine := indexer.New(
		opensea.New(viper.GetString("network"), viper.GetString("opensea.api_key")),
		tzkt.New("api.mainnet.tzkt.io"),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
	)

	worker := indexerWorker.New(network, indexerEngine, indexerStore)

	// workflows
	workflow.Register(worker.IndexOpenseaTokenWorkflow)
	workflow.Register(worker.IndexTezosTokenWorkflow)
	workflow.Register(worker.IndexTokenWorkflow)
	workflow.Register(worker.RefreshTokenProvenanceWorkflow)
	workflow.Register(worker.RefreshTokenProvenancePeriodicallyWorkflow)

	// opensea
	activity.Register(worker.IndexOwnerTokenDataFromOpensea)
	activity.Register(worker.IndexOwnerTokenDataFromTezos)
	activity.Register(worker.IndexToken)

	// index store
	activity.Register(worker.IndexAsset)
	activity.Register(worker.GetOutdatedTokens)
	activity.Register(worker.GetTokenIDsByOwner)
	activity.Register(worker.RefreshTokenProvenance)

	workerServiceClient := cadence.BuildCadenceServiceClient(hostPort, indexerWorker.ClientName, CadenceService)
	workerLogger := cadence.BuildCadenceLogger(logLevel)
	cadence.StartWorker(workerLogger, workerServiceClient, viper.GetString("cadence.domain"), indexerWorker.TaskListName)
}
