package indexer

import (
	"github.com/bitmark-inc/nft-indexer/externals/bettercall"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

type IndexEngine struct {
	opensea    *opensea.OpenseaClient
	bettercall *bettercall.BetterCall
	tzkt       *tzkt.TZKT
	fxhash     *fxhash.FxHashAPI
	objkt      *objkt.ObjktAPI
}

func New(
	opensea *opensea.OpenseaClient,
	bettercall *bettercall.BetterCall,
	tzkt *tzkt.TZKT,
	fxhash *fxhash.FxHashAPI,
	objkt *objkt.ObjktAPI,
) *IndexEngine {
	return &IndexEngine{
		opensea:    opensea,
		bettercall: bettercall,
		tzkt:       tzkt,
		fxhash:     fxhash,
		objkt:      objkt,
	}
}
