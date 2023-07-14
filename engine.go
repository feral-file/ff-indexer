package indexer

import (
	"net/http"
	"time"

	"github.com/bitmark-inc/nft-indexer/cache"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/tzkt-go"
	"github.com/ethereum/go-ethereum/ethclient"
)

type IndexEngine struct {
	environment  string
	ipfsGateways []string

	minterGateways map[string]string

	http        *http.Client
	opensea     *opensea.Client
	tzkt        *tzkt.TZKT
	fxhash      *fxhash.Client
	objkt       *objkt.Client
	ethereum    *ethclient.Client
	cacheClient *cache.Client
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
	cacheClient *cache.Client,
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
		opensea:     opensea,
		tzkt:        tzkt,
		fxhash:      fxhash,
		objkt:       objkt,
		ethereum:    ethereum,
		cacheClient: cacheClient,
	}
}
