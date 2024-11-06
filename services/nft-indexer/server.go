package main

import (
	"crypto/rsa"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cache"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/ens"
	tezosDomain "github.com/bitmark-inc/nft-indexer/externals/tezos-domain"
)

type NFTIndexerServer struct {
	apiToken           string
	adminAPIToken      string
	secretSymmetricKey string
	jwtPubkey          *rsa.PublicKey
	route              *gin.Engine
	ensClient          *ens.ENS
	tezosDomain        *tezosDomain.Client
	ethClient          *ethclient.Client
	cadenceWorker      *cadence.WorkerClient
	indexerStore       indexer.Store
	cacheStore         cache.Store
	indexerEngine      *indexer.IndexEngine
}

func NewNFTIndexerServer(cadenceWorker *cadence.WorkerClient,
	ensClient *ens.ENS,
	tezosDomain *tezosDomain.Client,
	ethClient *ethclient.Client,
	indexerStore indexer.Store,
	cacheStore cache.Store,
	indexerEngine *indexer.IndexEngine,
	jwtPubkey *rsa.PublicKey,
	apiToken string,
	adminAPIToken string,
	secretSymmetricKey string) *NFTIndexerServer {
	r := gin.New()

	return &NFTIndexerServer{
		apiToken:           apiToken,
		adminAPIToken:      adminAPIToken,
		secretSymmetricKey: secretSymmetricKey,
		jwtPubkey:          jwtPubkey,
		route:              r,
		ensClient:          ensClient,
		tezosDomain:        tezosDomain,
		ethClient:          ethClient,
		cadenceWorker:      cadenceWorker,
		indexerStore:       indexerStore,
		cacheStore:         cacheStore,
		indexerEngine:      indexerEngine,
	}
}

func (s *NFTIndexerServer) Run(port string) error {
	return s.route.Run(port)
}
