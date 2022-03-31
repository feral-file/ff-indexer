package indexerWorker

import (
	"net/http"
	"time"

	"github.com/bitmark-inc/nft-indexer/externals/bettercall"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
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
	fxhash       *fxhash.FxHashAPI
	objkt        *objkt.ObjktAPI
	indexerStore indexer.IndexerStore
	wallet       *ethereum.Wallet

	bitmarkZeroAddress string
	bitmarkAPIEndpoint string

	txEndpoints map[string]string

	Network      string
	TaskListName string
}

func New(network string,
	openseaClient *opensea.OpenseaClient,
	bettercall *bettercall.BetterCall,
	fxhash *fxhash.FxHashAPI,
	objkt *objkt.ObjktAPI,
	store indexer.IndexerStore) *NFTIndexerWorker {

	w, err := ethereum.NewWalletFromMnemonic(
		viper.GetString("ethereum.worker_account_mnemonic"),
		network, viper.GetString("ethereum.rpc_url"))
	if err != nil {
		panic(err)
	}

	bitmarkZeroAddress := indexer.LivenetZeroAddress
	bitmarkAPIEndpoint := "https://api.bitmark.com"

	txEndpoints := map[string]string{
		indexer.BitmarkBlockchain:  "https://registry.bitmark.com/transaction",
		indexer.EthereumBlockchain: "https://etherscan.io/tx",
		indexer.TezosBlockchain:    "https://tzkt.io",
	}

	if network != "livenet" {
		bitmarkZeroAddress = indexer.TestnetZeroAddress
		bitmarkAPIEndpoint = "https://api.test.bitmark.com"
		txEndpoints[indexer.BitmarkBlockchain] = "https://registry.test.bitmark.com/transaction"
		txEndpoints[indexer.EthereumBlockchain] = "https://rinkeby.etherscan.io"
	}

	return &NFTIndexerWorker{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		wallet:       w,
		opensea:      openseaClient,
		bettercall:   bettercall,
		fxhash:       fxhash,
		objkt:        objkt,
		indexerStore: store,

		bitmarkZeroAddress: bitmarkZeroAddress,
		bitmarkAPIEndpoint: bitmarkAPIEndpoint,

		txEndpoints: txEndpoints,

		Network:      network,
		TaskListName: TaskListName,
	}
}
