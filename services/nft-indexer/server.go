package main

import (
	"github.com/gin-gonic/gin"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cadence"
)

type NFTIndexerServer struct {
	apiToken string
	route    *gin.Engine

	cadenceWorker *cadence.CadenceWorkerClient
	indexerStore  indexer.IndexerStore
}

func NewNFTIndexerServer(cadenceWorker *cadence.CadenceWorkerClient, indexerStore indexer.IndexerStore, apiToken string) *NFTIndexerServer {
	r := gin.New()

	return &NFTIndexerServer{
		apiToken: apiToken,
		route:    r,

		cadenceWorker: cadenceWorker,
		indexerStore:  indexerStore,
	}
}

func (s *NFTIndexerServer) Run(port string) error {
	return s.route.Run(port)
}
