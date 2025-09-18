package indexer

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/managedblockchainquery"
	"github.com/bitmark-inc/tzkt-go"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/feral-file/ff-indexer/cache"
	"github.com/feral-file/ff-indexer/externals/fxhash"
	"github.com/feral-file/ff-indexer/externals/objkt"
	"github.com/feral-file/ff-indexer/externals/opensea"
)

type IndexEngine struct {
	environment  string
	ipfsGateways []string

	minterGateways map[string]string

	http       *http.Client
	opensea    *opensea.Client
	tzkt       *tzkt.TZKT
	fxhash     *fxhash.Client
	objkt      *objkt.Client
	ethereum   *ethclient.Client
	cacheStore cache.Store

	blockchainQueryClient *managedblockchainquery.ManagedBlockchainQuery
}

func New(
	environment string,
	ipfsGateways []string,
	minterGateways map[string]string,
	opensea *opensea.Client,
	tzkt *tzkt.TZKT,
	fxhash *fxhash.Client,
	objkt *objkt.Client,
	ethereum *ethclient.Client,
	cacheStore cache.Store,
	blockchainQueryClient *managedblockchainquery.ManagedBlockchainQuery,
) *IndexEngine {
	if len(ipfsGateways) == 0 {
		ipfsGateways = []string{DefaultIPFSGateway}
	}

	return &IndexEngine{
		environment:  environment,
		ipfsGateways: ipfsGateways,

		minterGateways: minterGateways,

		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		opensea:    opensea,
		tzkt:       tzkt,
		fxhash:     fxhash,
		objkt:      objkt,
		ethereum:   ethereum,
		cacheStore: cacheStore,

		blockchainQueryClient: blockchainQueryClient,
	}
}
