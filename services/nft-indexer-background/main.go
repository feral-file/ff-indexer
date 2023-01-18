package main

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
	log "github.com/bitmark-inc/nft-indexer/zapLog"
)

var CadenceService = "cadence-frontend"

func main() {
	config.LoadConfig("NFT_INDEXER")

	hostPort := viper.GetString("cadence.host_port")
	logLevel := viper.GetInt("cadence.log_level")

	environment := viper.GetString("environment")

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		log.Logger.Panic("Sentry initialization failed", zap.Error(err))
	}

	ctx := context.Background()

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.Logger.Panic("fail to initiate indexer store", zap.Error(err))
	}

	indexerEngine := indexer.New(
		environment,
		opensea.New(viper.GetString("network.ethereum"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New(viper.GetString("network.tezos")),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
	)

	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(viper.GetString("aws.region")),
	}))

	worker := indexerWorker.New(environment, indexerEngine, awsSession, indexerStore)

	// workflows
	workflow.Register(worker.IndexOpenseaTokenWorkflow)
	workflow.Register(worker.IndexTezosTokenWorkflow)
	workflow.RegisterWithOptions(worker.IndexTokenWorkflow, workflow.RegisterOptions{
		Name: "IndexTokenWorkflow",
	})

	// cache
	activity.Register(worker.CacheIPFSArtifactInS3)

	// all blockchain
	activity.Register(worker.IndexToken)
	// ethereum
	activity.Register(worker.IndexETHTokenByOwner)
	// tezos
	activity.Register(worker.IndexTezosTokenByOwner)
	// index store
	activity.Register(worker.IndexAsset)
	activity.Register(worker.GetTokenIDsByOwner)
	activity.Register(worker.GetOutdatedTokensByOwner)
	activity.Register(worker.RefreshTokenProvenance)

	// index account tokens
	activity.Register(worker.IndexAccountTokens)

	workerServiceClient := cadence.BuildCadenceServiceClient(hostPort, indexerWorker.ClientName, CadenceService)
	workerLogger := cadence.BuildCadenceLogger(logLevel)
	cadence.StartWorker(workerLogger, workerServiceClient, viper.GetString("cadence.domain"), indexerWorker.TaskListName)
}
