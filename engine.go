package indexer

import (
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

type IndexEngine struct {
	network string
	opensea *opensea.OpenseaClient
	tzkt    *tzkt.TZKT
	fxhash  *fxhash.FxHashAPI
	objkt   *objkt.ObjktAPI
}

func New(
	network string,
	opensea *opensea.OpenseaClient,
	tzkt *tzkt.TZKT,
	fxhash *fxhash.FxHashAPI,
	objkt *objkt.ObjktAPI,
) *IndexEngine {
	return &IndexEngine{
		network: network,
		opensea: opensea,
		tzkt:    tzkt,
		fxhash:  fxhash,
		objkt:   objkt,
	}
}
