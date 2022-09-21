package indexer

import (
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

type IndexEngine struct {
	environment string
	opensea     *opensea.OpenseaClient
	tzkt        *tzkt.TZKT
	fxhash      *fxhash.FxHashAPI
	objkt       *objkt.ObjktAPI
}

func New(
	environment string,
	opensea *opensea.OpenseaClient,
	tzkt *tzkt.TZKT,
	fxhash *fxhash.FxHashAPI,
	objkt *objkt.ObjktAPI,
) *IndexEngine {
	return &IndexEngine{
		environment: environment,
		opensea:     opensea,
		tzkt:        tzkt,
		fxhash:      fxhash,
		objkt:       objkt,
	}
}
