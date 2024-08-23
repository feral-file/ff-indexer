package worker

import (
	"net/http"
	"time"

	assetSDK "github.com/bitmark-inc/autonomy-asset-server/sdk/api"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cache"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
)

var ClientName = "nft-indexer-worker"
var TaskListName = "nft-indexer"
var ProvenanceTaskListName = "nft-provenance-indexer"

type NFTIndexerWorker struct {
	http *http.Client

	ipfsCacheBucketName string

	indexerEngine *indexer.IndexEngine
	indexerStore  indexer.Store
	cacheStore    cache.Store
	assetClient   *assetSDK.Client
	ethClient     *ethclient.Client

	bitmarkZeroAddress string
	bitmarkAPIEndpoint string

	Environment            string
	TaskListName           string
	ProvenanceTaskListName string
}

func New(environment string,
	indexerEngine *indexer.IndexEngine,
	cacheStore cache.Store,
	store indexer.Store,
	assetClient *assetSDK.Client,
) *NFTIndexerWorker {

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
		ethClient: wsClient,

		ipfsCacheBucketName: viper.GetString("cache.bucket_name"),

		indexerEngine: indexerEngine,
		indexerStore:  store,
		cacheStore:    cacheStore,
		assetClient:   assetClient,

		bitmarkZeroAddress: bitmarkZeroAddress,
		bitmarkAPIEndpoint: bitmarkAPIEndpoint,

		Environment:            environment,
		TaskListName:           TaskListName,
		ProvenanceTaskListName: ProvenanceTaskListName,
	}
}
