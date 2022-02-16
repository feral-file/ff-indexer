package main

import (
	"github.com/gin-gonic/gin"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/ens"
	"github.com/bitmark-inc/nft-indexer/externals/feralfile"
	tezosDomain "github.com/bitmark-inc/nft-indexer/externals/tezos-domain"
)

type NFTIndexerServer struct {
	apiToken string
	route    *gin.Engine

	ensClient     *ens.ENS
	tezosDomain   *tezosDomain.TezosDomainAPI
	feralfile     *feralfile.Feralfile
	cadenceWorker *cadence.CadenceWorkerClient
	indexerStore  indexer.IndexerStore
}

func NewNFTIndexerServer(cadenceWorker *cadence.CadenceWorkerClient,
	ensClient *ens.ENS,
	tezosDomain *tezosDomain.TezosDomainAPI,
	feralfileClient *feralfile.Feralfile,
	indexerStore indexer.IndexerStore,
	apiToken string) *NFTIndexerServer {
	r := gin.New()

	return &NFTIndexerServer{
		apiToken: apiToken,
		route:    r,

		ensClient:     ensClient,
		tezosDomain:   tezosDomain,
		feralfile:     feralfileClient,
		cadenceWorker: cadenceWorker,
		indexerStore:  indexerStore,
	}
}

func (s *NFTIndexerServer) Run(port string) error {
	return s.route.Run(port)
}
