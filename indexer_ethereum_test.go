package indexer

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/tzkt-go"
)

func TestIndexETHToken(t *testing.T) {
	if err := log.Initialize("", false); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	engine := New(
		"",
		[]string{},
		map[string]string{},
		opensea.New("livenet", "", 1),
		tzkt.New(""),
		fxhash.New("https://api.fxhash.xyz/graphql"),
		objkt.New("https://data.objkt.com/v3/graphql"),
	)

	assetUpdates, err := engine.IndexETHToken(context.Background(), "0xbc4ca0eda7647a8ab7c2061c2e118a18a936f13d", "9616")

	assert.NoError(t, err)
	assert.Equal(t, assetUpdates.Tokens[0].Balance, int64(0))
	assert.Equal(t, assetUpdates.Tokens[0].Owner, "")
}
