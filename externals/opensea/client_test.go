package opensea

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrieveTokenOwner(t *testing.T) {
	openseaKey := os.Getenv("OPENSEA_KEY")
	client := New("livenet", openseaKey)

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
	client := New("livenet", openseaKey)

	balance, err := client.GetTokenBalanceForOwner("0x28472a58a490c5e09a238847f66a68a47cc76f0f", "0", "0x1a44a11ec3cda9767c9d82f9405d926e22f3df2c")
	assert.NoError(t, err)
	assert.Equal(t, balance, int64(2))
}
