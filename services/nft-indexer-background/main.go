package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/cache"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/tzkt-go"
)

var CadenceService = "cadence-frontend"

func main() {
	config.LoadConfig("NFT_INDEXER")
	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	hostPort := viper.GetString("cadence.host_port")

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
		log.Panic("fail to initiate eth client", zap.Error(err))
	}

	indexerEngine := indexer.New(
		environment,
		viper.GetStringSlice("ipfs.preferred_gateways"),
		minterGateways,
		opensea.New(viper.GetString("network.ethereum"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New(viper.GetString("network.tezos")),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
		ethClient,
		cacheStore,
	)

	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(viper.GetString("aws.region")),
	}))

	worker := indexerWorker.New(environment, indexerEngine, cacheStore, awsSession, indexerStore)

	// workflows
	workflow.Register(worker.IndexETHTokenWorkflow)
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
	activity.Register(worker.GetTokenBalanceOfOwner)
	activity.Register(worker.RefreshTokenProvenance)

	// index account tokens
	activity.Register(worker.IndexAccountTokens)

	workerServiceClient := cadence.BuildCadenceServiceClient(hostPort, indexerWorker.ClientName, CadenceService)
	cadence.StartWorker(log.DefaultLogger(), workerServiceClient, viper.GetString("cadence.domain"), indexerWorker.TaskListName)
}
