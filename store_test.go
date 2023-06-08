package indexer

import (
	"context"
	"fmt"
	"testing"
	"time"

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

	indexerStore, err := NewMongodbIndexerStore(ctx, viper.GetString("indexer_store.db_uri"), viper.GetString("indexer_store.db_name"))
	if err != nil {
		panic(err)
	}

	token, err := indexerStore.GetTokensByIndexID(ctx, "eth-0xb43c51447405008AEBf7a35B4D15e1f29b7Ce823-84379833228553110502734947101839209675161105358737778734002435191848727499610")

	assert.Nil(t, err)
	fmt.Println(token)
}

// TestGetAccountTokensByOwners test get account tokens by owners
func TestGetAccountTokensByOwners(t *testing.T) {
	ctx := context.Background()
	config.LoadConfig("NFT_INDEXER")

	indexerStore, err := NewMongodbIndexerStore(ctx, viper.GetString("indexer_store.db_uri"), viper.GetString("indexer_store.db_name"))
	if err != nil {
		panic(err)
	}

	accountTokens, err := indexerStore.GetAccountTokensByOwners(ctx, []string{"tz1LPJ34B1Z8XsxtgoCv5NRBTHTXoeG49A9h"}, FilterParameter{
		IDs: []string{"tez-KT1DPFXN2NeFjg1aQGNkVXYS1FAy4BymcbZz-1685693490216"},
	})

	assert.Nil(t, err)
	fmt.Println(accountTokens)
}

// TestGetDetailedAccountTokensByOwners test get detailed account tokens by owners
func TestGetDetailedAccountTokensByOwners(t *testing.T) {
	ctx := context.Background()
	config.LoadConfig("NFT_INDEXER")

	indexerStore, err := NewMongodbIndexerStore(ctx, viper.GetString("indexer_store.db_uri"), viper.GetString("indexer_store.db_name"))
	if err != nil {
		panic(err)
	}

	detailedTokenV2, err := indexerStore.GetDetailedAccountTokensByOwners(
		ctx,
		[]string{"tz1LPJ34B1Z8XsxtgoCv5NRBTHTXoeG49A9h"},
		FilterParameter{
			Source: "tzkt",
			IDs:    nil,
		},
		time.Time{},
		"",
		0,
		1,
	)

	assert.Nil(t, err)
	fmt.Println(detailedTokenV2)
}
