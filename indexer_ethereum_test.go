package indexer

import (
	"fmt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/tzkt-go"
)

func TestIndexETHToken(t *testing.T) {
	if err := log.Initialize("", false); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	engine := New(
		"",
		opensea.New("livenet", "", 1),
		tzkt.New(""),
		fxhash.New("https://api.fxhash.xyz/graphql"),
		objkt.New("https://data.objkt.com/v3/graphql"),
	)

	assetUpdates, err := engine.IndexETHToken("0xb6bbc4C740dc49A80Db4D00e7fb7E15819f578aF", "0xbc4ca0eda7647a8ab7c2061c2e118a18a936f13d", "9616")

	assert.NoError(t, err)
	fmt.Println(assetUpdates)
}
