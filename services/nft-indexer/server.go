package main

import (
	"github.com/gin-gonic/gin"

	indexer "github.com/bitmark-inc/nft-indexer"
)

type NFTIndexerServer struct {
	apiToken     string
	route        *gin.Engine
	indexerStore indexer.IndexerStore
}

func NewNFTIndexerServer(indexerStore indexer.IndexerStore, apiToken string) *NFTIndexerServer {
	r := gin.New()

	return &NFTIndexerServer{
		apiToken:     apiToken,
		route:        r,
		indexerStore: indexerStore,
	}
}

func (s *NFTIndexerServer) Run(port string) error {
	return s.route.Run(port)
}
