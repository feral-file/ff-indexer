package indexer

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/config-loader"
)

func TestGetPageCount(t *testing.T) {
	assert.Equal(t, 1, getPageCounts(25, 25))
	assert.Equal(t, 2, getPageCounts(26, 25))
	assert.Equal(t, 5, getPageCounts(101, 25))
	assert.Equal(t, 1, getPageCounts(88, 100))
	assert.Equal(t, 2, getPageCounts(101, 100))
}

func TestGetTokensByIndexID(t *testing.T) {
	ctx := context.Background()
	config.LoadConfig("NFT_INDEXER")

	indexerStore, err := NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		panic(err)
	}

	token, err := indexerStore.GetTokensByIndexID(ctx, "eth-0xb43c51447405008AEBf7a35B4D15e1f29b7Ce823-84379833228553110502734947101839209675161105358737778734002435191848727499610")

	assert.Nil(t, err)
	fmt.Println(token)
}
