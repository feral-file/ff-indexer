package worker

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cache"
)

var ClientName = "nft-indexer-worker"
var TaskListName = "nft-indexer"
var ProvenanceTaskListName = "nft-provenance-indexer"
var AccountTokenTaskListName = "nft-account-token-indexer"

type NFTIndexerWorker struct {
	http       *http.Client
	awsSession *session.Session

	ipfsCacheBucketName string

	indexerEngine *indexer.IndexEngine
	indexerStore  indexer.Store
	cacheClient   *cache.Client
	ethClient     *ethclient.Client

	bitmarkZeroAddress string
	bitmarkAPIEndpoint string

	Environment              string
	TaskListName             string
	ProvenanceTaskListName   string
	AccountTokenTaskListName string
}

func New(environment string,
	indexerEngine *indexer.IndexEngine,
	cacheClient *cache.Client,
	awsSession *session.Session,
	store indexer.Store) *NFTIndexerWorker {

	wsClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		panic(err)
	}

	bitmarkZeroAddress := indexer.LivenetZeroAddress
	bitmarkAPIEndpoint := "https://api.bitmark.com"

	if environment == indexer.DevelopmentEnvironment {
		// staging / development
		bitmarkZeroAddress = indexer.TestnetZeroAddress
		bitmarkAPIEndpoint = "https://api.test.bitmark.com"
	}

	return &NFTIndexerWorker{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		ethClient:  wsClient,
		awsSession: awsSession,

		ipfsCacheBucketName: viper.GetString("cache.bucket_name"),

		indexerEngine: indexerEngine,
		indexerStore:  store,
		cacheClient:   cacheClient,

		bitmarkZeroAddress: bitmarkZeroAddress,
		bitmarkAPIEndpoint: bitmarkAPIEndpoint,

		Environment:              environment,
		TaskListName:             TaskListName,
		ProvenanceTaskListName:   ProvenanceTaskListName,
		AccountTokenTaskListName: AccountTokenTaskListName,
	}
}
