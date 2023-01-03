package indexerWorker

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/viper"

	ethereum "github.com/bitmark-inc/account-vault-ethereum"
	indexer "github.com/bitmark-inc/nft-indexer"
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
	indexerStore  indexer.IndexerStore
	wallet        *ethereum.Wallet

	bitmarkZeroAddress string
	bitmarkAPIEndpoint string

	Environment              string
	TaskListName             string
	ProvenanceTaskListName   string
	AccountTokenTaskListName string
}

func New(environment string,
	indexerEngine *indexer.IndexEngine,
	awsSession *session.Session,
	store indexer.IndexerStore) *NFTIndexerWorker {

	w, err := ethereum.NewWalletFromMnemonic(
		viper.GetString("ethereum.worker_account_mnemonic"),
		viper.GetString("network.ethereum"),
		viper.GetString("ethereum.rpc_url"))
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
		wallet:     w,
		awsSession: awsSession,

		ipfsCacheBucketName: viper.GetString("cache.bucket_name"),

		indexerEngine: indexerEngine,
		indexerStore:  store,

		bitmarkZeroAddress: bitmarkZeroAddress,
		bitmarkAPIEndpoint: bitmarkAPIEndpoint,

		Environment:              environment,
		TaskListName:             TaskListName,
		ProvenanceTaskListName:   ProvenanceTaskListName,
		AccountTokenTaskListName: AccountTokenTaskListName,
	}
}
