package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/infura"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/tzkt-go"
)

var CadenceService = "cadence-frontend"

func main() {
	config.LoadConfig("NFT_INDEXER")
	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	hostPort := viper.GetString("cadence.host_port")
	logLevel := viper.GetInt("cadence.log_level")

	environment := viper.GetString("environment")

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		log.Panic("Sentry initialization failed", zap.Error(err))
	}

	ctx := context.Background()

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	indexerEngine := indexer.New(
		environment,
		viper.GetStringSlice("ipfs.preferred_gateways"),
		opensea.New(viper.GetString("network.ethereum"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New(viper.GetString("network.tezos")),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
		infura.New(viper.GetString("network.ethereum"), viper.GetString("ethereum.infura_api_key"), viper.GetString("ethereum.infura_api_key_secret")),
	)

	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(viper.GetString("aws.region")),
	}))

	worker := indexerWorker.New(environment, indexerEngine, awsSession, indexerStore)

	// workflows
	workflow.Register(worker.RefreshTokenProvenanceWorkflow)
	workflow.Register(worker.RefreshTokenOwnershipWorkflow)

	// activities
	activity.Register(worker.RefreshTokenProvenance)
	activity.Register(worker.RefreshTokenOwnership)

	workerServiceClient := cadence.BuildCadenceServiceClient(hostPort, indexerWorker.ClientName, CadenceService)
	workerLogger := cadence.BuildCadenceLogger(logLevel)
	cadence.StartWorker(workerLogger, workerServiceClient, viper.GetString("cadence.domain"), indexerWorker.ProvenanceTaskListName)
}
