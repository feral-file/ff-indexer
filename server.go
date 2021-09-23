package main

import (
	"github.com/gin-gonic/gin"
)

type NFTIndexerServer struct {
	apiToken     string
	route        *gin.Engine
	indexerStore IndexerStore
}

func NewNFTIndexerServer(indexerStore IndexerStore, apiToken string) *NFTIndexerServer {
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
