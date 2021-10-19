package indexerWorker

import (
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/externals/artblocks"
)

var ClientName = "nft-indexer-worker"
var TaskListName = "nft-indexer"

type NFTIndexerWorker struct {
	artblocks    *artblocks.ArtblocksClient
	indexerStore indexer.IndexerStore

	Network      string
	TaskListName string
}

func New(network string, artblocksClient *artblocks.ArtblocksClient, store indexer.IndexerStore) *NFTIndexerWorker {
	return &NFTIndexerWorker{
		artblocks:    artblocksClient,
		indexerStore: store,

		Network:      network,
		TaskListName: TaskListName,
	}
}
