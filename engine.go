package indexer

import (
	"net/http"
	"time"

	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/tzkt-go"
)

type IndexEngine struct {
	environment  string
	ipfsGateways []string

	http    *http.Client
	opensea *opensea.Client
	tzkt    *tzkt.TZKT
	fxhash  *fxhash.Client
	objkt   *objkt.Client
}

func New(
	environment string,
	ipfsGateways []string,
	opensea *opensea.Client,
	tzkt *tzkt.TZKT,
	fxhash *fxhash.Client,
	objkt *objkt.Client,
) *IndexEngine {
	if len(ipfsGateways) == 0 {
		ipfsGateways = []string{DefaultIPFSGateway}
	}

	return &IndexEngine{
		environment:  environment,
		ipfsGateways: ipfsGateways,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		opensea: opensea,
		tzkt:    tzkt,
		fxhash:  fxhash,
		objkt:   objkt,
	}
}
