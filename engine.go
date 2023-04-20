package indexer

import (
	"net/http"
	"time"

	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

type IndexEngine struct {
	environment string
	http        *http.Client
	opensea     *opensea.Client
	tzkt        *tzkt.TZKT
	fxhash      *fxhash.Client
	objkt       *objkt.Client
}

func New(
	environment string,
	opensea *opensea.Client,
	tzkt *tzkt.TZKT,
	fxhash *fxhash.Client,
	objkt *objkt.Client,
) *IndexEngine {
	return &IndexEngine{
		environment: environment,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		opensea: opensea,
		tzkt:    tzkt,
		fxhash:  fxhash,
		objkt:   objkt,
	}
}
