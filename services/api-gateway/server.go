package main

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"

	indexer "github.com/feral-file/ff-indexer"
	"github.com/feral-file/ff-indexer/cache"
	"github.com/feral-file/ff-indexer/cadence"
	"github.com/feral-file/ff-indexer/externals/ens"
	tezosDomain "github.com/feral-file/ff-indexer/externals/tezos-domain"
)

type Server struct {
	apiToken      string
	adminAPIToken string
	route         *gin.Engine
	ensClient     *ens.ENS
	tezosDomain   *tezosDomain.Client
	ethClient     *ethclient.Client
	cadenceWorker *cadence.WorkerClient
	indexerStore  indexer.Store
	cacheStore    cache.Store
	indexerEngine *indexer.IndexEngine
}

func NewServer(cadenceWorker *cadence.WorkerClient,
	ensClient *ens.ENS,
	tezosDomain *tezosDomain.Client,
	ethClient *ethclient.Client,
	indexerStore indexer.Store,
	cacheStore cache.Store,
	indexerEngine *indexer.IndexEngine,
	apiToken string,
	adminAPIToken string) *Server {
	r := gin.New()

	return &Server{
		apiToken:      apiToken,
		adminAPIToken: adminAPIToken,
		route:         r,
		ensClient:     ensClient,
		tezosDomain:   tezosDomain,
		ethClient:     ethClient,
		cadenceWorker: cadenceWorker,
		indexerStore:  indexerStore,
		cacheStore:    cacheStore,
		indexerEngine: indexerEngine,
	}
}

func (s *Server) Run(port string) error {
	return s.route.Run(port)
}
