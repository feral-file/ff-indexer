package opensea

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/nft-indexer/log"
)

func TestMain(m *testing.M) {
	if err := log.Initialize("", false); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}
	os.Exit(m.Run())
}

func TestRetrieveTokenOwner(t *testing.T) {
	openseaKey := os.Getenv("OPENSEA_KEY")
	client := New("livenet", openseaKey, 1)

	owners, next1, err := client.RetrieveTokenOwners("0x28472a58a490c5e09a238847f66a68a47cc76f0f", "1", nil)
	assert.NoError(t, err)
	assert.NotNil(t, next1)

	assert.Equal(t, len(owners), 50)

	owners, next2, err := client.RetrieveTokenOwners("0x28472a58a490c5e09a238847f66a68a47cc76f0f", "1", next1)
	assert.NoError(t, err)
	assert.NotNil(t, next2)

	assert.NotEqual(t, *next1, *next2)

	assert.Equal(t, len(owners), 50)
}

func TestGetTokenBalanceForOwner(t *testing.T) {
	openseaKey := os.Getenv("OPENSEA_KEY")
	client := New("livenet", openseaKey, 1)

	balance, err := client.GetTokenBalanceForOwner("0x28472a58a490c5e09a238847f66a68a47cc76f0f", "0", "0x1a44a11ec3cda9767c9d82f9405d926e22f3df2c")
	assert.NoError(t, err)
	assert.Equal(t, balance, int64(2))
}

func TestGetTokensForOwner(t *testing.T) {
	openseaKey := os.Getenv("OPENSEA_KEY")
	client := New("livenet", openseaKey, 1)

	tokens, err := client.RetrieveAssets("0xb858A3F45840E76076c6c4DBa9f0f8958F11C1E8", 0)
	assert.NoError(t, err)
	assert.Len(t, tokens, 50)
}
