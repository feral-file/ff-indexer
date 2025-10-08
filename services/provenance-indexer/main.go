package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/managedblockchainquery"
	bitmarkd "github.com/bitmark-inc/bitmarkdClient"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/tzkt-go"

	indexer "github.com/feral-file/ff-indexer"
	indexerWorker "github.com/feral-file/ff-indexer/background/worker"
	"github.com/feral-file/ff-indexer/cache"
	"github.com/feral-file/ff-indexer/cadence"
	"github.com/feral-file/ff-indexer/externals/fxhash"
	"github.com/feral-file/ff-indexer/externals/objkt"
	"github.com/feral-file/ff-indexer/externals/opensea"
)

var CadenceService = "cadence-frontend"

func main() {
	config.LoadConfig("NFT_INDEXER")

	environment := viper.GetString("environment")
	if err := log.Initialize(viper.GetBool("debug"), &sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	hostPort := viper.GetString("cadence.host_port")

	ctx := context.Background()

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"), environment)
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}
	cacheStore, err := cache.NewMongoDBCacheStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.Panic("fail to initiate cache store", zap.Error(err))
	}

	var minterGateways map[string]string
	if err := yaml.Unmarshal([]byte(viper.GetString("ipfs.minter_gateways")), &minterGateways); err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	ethClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		log.Panic("fail to initiate eth client: %s", zap.Error(err))
	}

	awsSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-east-1"),
		},
	})
	if err != nil {
		log.Panic("fail to set up aws session", zap.Error(err))
	}

	indexerEngine := indexer.New(
		environment,
		viper.GetStringSlice("ipfs.preferred_gateways"),
		minterGateways,
		opensea.New(viper.GetString("network.ethereum"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New(viper.GetString("network.tezos")),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("network.tezos")),
		ethClient,
		cacheStore,
		managedblockchainquery.New(awsSession),
	)

	bitmarkdClient := bitmarkd.New(strings.Split(viper.GetString("bitmarkd.rpc_conn"), ","), time.Minute)

	worker := indexerWorker.New(environment, indexerEngine, cacheStore, indexerStore, bitmarkdClient)

	// workflows
	workflow.Register(worker.RefreshTokenProvenanceWorkflow)
	workflow.Register(worker.RefreshTokenOwnershipWorkflow)
	workflow.Register(worker.RefreshTokenProvenanceByOwnerWorkflow)

	// activities

	activity.Register(worker.GetOwnedTokenIDsByOwner)
	activity.Register(worker.FilterTokenIDsWithInconsistentProvenanceForOwner)
	activity.Register(worker.RefreshTokenProvenance)
	activity.Register(worker.RefreshTokenOwnership)
	activity.Register(worker.MarkAccountTokenChanged)

	workerServiceClient := cadence.BuildCadenceServiceClient(hostPort, indexerWorker.ClientName, CadenceService)
	cadence.StartWorker(log.Logger(), workerServiceClient, viper.GetString("cadence.domain"), indexerWorker.ProvenanceTaskListName)
}
