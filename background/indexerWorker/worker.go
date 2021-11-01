package indexerWorker

import (
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
)

var ClientName = "nft-indexer-worker"
var TaskListName = "nft-indexer"

type NFTIndexerWorker struct {
	opensea      *opensea.OpenseaClient
	indexerStore indexer.IndexerStore

	Network      string
	TaskListName string
}

func New(network string,
	openseaClient *opensea.OpenseaClient,
	store indexer.IndexerStore) *NFTIndexerWorker {
	return &NFTIndexerWorker{
		opensea:      openseaClient,
		indexerStore: store,

		Network:      network,
		TaskListName: TaskListName,
	}
}
