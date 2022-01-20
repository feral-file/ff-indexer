package indexerWorker

import (
	"net/http"
	"time"

	"github.com/bitmark-inc/nft-indexer/externals/bettercall"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/spf13/viper"

	ethereum "github.com/bitmark-inc/account-vault-ethereum"
	indexer "github.com/bitmark-inc/nft-indexer"
)

var ClientName = "nft-indexer-worker"
var TaskListName = "nft-indexer"

type NFTIndexerWorker struct {
	http         *http.Client
	opensea      *opensea.OpenseaClient
	bettercall   *bettercall.BetterCall
	indexerStore indexer.IndexerStore
	wallet       *ethereum.Wallet

	bitmarkZeroAddress string
	bitmarkAPIEndpoint string

	Network      string
	TaskListName string
}

func New(network string,
	openseaClient *opensea.OpenseaClient,
	bettercall *bettercall.BetterCall,
	store indexer.IndexerStore) *NFTIndexerWorker {

	w, err := ethereum.NewWalletFromMnemonic(
		viper.GetString("ethereum.worker_account_mnemonic"),
		network, viper.GetString("ethereum.rpc_url"))
	if err != nil {
		panic(err)
	}

	bitmarkZeroAddress := indexer.LivenetZeroAddress
	bitmarkAPIEndpoint := "https://api.bitmark.com"
	if network != "livenet" {
		bitmarkZeroAddress = indexer.TestnetZeroAddress
		bitmarkAPIEndpoint = "https://api.test.bitmark.com"
	}

	return &NFTIndexerWorker{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		wallet:       w,
		opensea:      openseaClient,
		bettercall:   bettercall,
		indexerStore: store,

		bitmarkZeroAddress: bitmarkZeroAddress,
		bitmarkAPIEndpoint: bitmarkAPIEndpoint,

		Network:      network,
		TaskListName: TaskListName,
	}
}
