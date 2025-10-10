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
		log.Panic("fail to initiate eth client", zap.Error(err))
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
		opensea.New(viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
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
	workflow.Register(worker.IndexETHTokenWorkflow)
	workflow.Register(worker.IndexTezosTokenWorkflow)
	workflow.RegisterWithOptions(worker.IndexTokenWorkflow, workflow.RegisterOptions{
		Name: "IndexTokenWorkflow",
	})
	workflow.RegisterWithOptions(worker.IndexEthereumTokenSaleInBlockRange, workflow.RegisterOptions{
		Name: "IndexEthereumTokenSaleInBlockRange"})
	workflow.RegisterWithOptions(worker.IndexEthereumTokenSale, workflow.RegisterOptions{
		Name: "IndexEthereumTokenSale",
	})
	workflow.RegisterWithOptions(worker.ParseEthereumTokenSale, workflow.RegisterOptions{
		Name: "ParseEthereumTokenSale"})
	workflow.RegisterWithOptions(worker.IndexTezosObjktTokenSaleFromTime, workflow.RegisterOptions{
		Name: "IndexTezosObjktTokenSaleFromTime"})
	workflow.RegisterWithOptions(worker.IndexTezosTokenSaleFromTzktTxID, workflow.RegisterOptions{
		Name: "IndexTezosTokenSaleFromTzktTxID",
	})
	workflow.RegisterWithOptions(worker.IndexTezosObjktTokenSale, workflow.RegisterOptions{
		Name: "IndexTezosTokenSale",
	})
	workflow.RegisterWithOptions(worker.CrawlHistoricalExchangeRate, workflow.RegisterOptions{
		Name: "CrawlHistoricalExchangeRate",
	})
	workflow.RegisterWithOptions(worker.CrawlExchangeRateByCurrencyPair, workflow.RegisterOptions{
		Name: "CrawlExchangeRateByCurrencyPair",
	})

	// all blockchain
	activity.Register(worker.IndexToken)

	// ethereum
	activity.Register(worker.IndexETHTokenByOwner)
	activity.Register(worker.GetEthereumTxReceipt)
	activity.Register(worker.GetEthereumTx)
	activity.Register(worker.GetEthereumBlockHeaderHash)
	activity.Register(worker.GetEthereumBlockHeaderByNumber)
	activity.Register(worker.GetEthereumInternalTxs)
	activity.Register(worker.FilterEthereumNFTTxByEventLogs)
	activity.Register(worker.ParseTokenSaleToGenericSalesTimeSeries)

	// tezos
	activity.Register(worker.IndexTezosTokenByOwner)
	activity.Register(worker.GetTezosTxHashFromTzktTransactionID)
	activity.Register(worker.GetObjktSaleTransactionHashes)
	activity.Register(worker.ParseTezosObjktTokenSale)

	// index store
	activity.Register(worker.IndexAsset)
	activity.Register(worker.GetTokenBalanceOfOwner)
	activity.Register(worker.RefreshTokenProvenance)
	activity.Register(worker.GetTokenByIndexID)
	activity.Register(worker.WriteSaleTimeSeriesData)
	activity.Register(worker.IndexedSaleTx)
	activity.Register(worker.WriteHistoricalExchangeRate)
	activity.Register(worker.CrawlExchangeRateFromCoinbase)
	activity.Register(worker.GetExchangeRateLastTime)

	// index account tokens
	activity.Register(worker.IndexAccountTokens)
	activity.Register(worker.MarkAccountTokenChanged)

	workerServiceClient := cadence.BuildCadenceServiceClient(hostPort, indexerWorker.ClientName, CadenceService)

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)
	if err := indexerWorker.StartIndexExchangeRateCronWorkflow(ctx, cadenceClient); err != nil {
		panic(err)
	}

	cadence.StartWorker(log.Logger(), workerServiceClient, viper.GetString("cadence.domain"), indexerWorker.TaskListName)
}
