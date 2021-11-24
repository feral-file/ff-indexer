package indexerWorker

import (
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/externals/bettercall"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
)

var ClientName = "nft-indexer-worker"
var TaskListName = "nft-indexer"

type NFTIndexerWorker struct {
	opensea      *opensea.OpenseaClient
	bettercall   *bettercall.BetterCall
	indexerStore indexer.IndexerStore

	Network      string
	TaskListName string
}

func New(network string,
	openseaClient *opensea.OpenseaClient,
	bettercall *bettercall.BetterCall,
	store indexer.IndexerStore) *NFTIndexerWorker {
	return &NFTIndexerWorker{
		opensea:      openseaClient,
		bettercall:   bettercall,
		indexerStore: store,

		Network:      network,
		TaskListName: TaskListName,
	}
}
