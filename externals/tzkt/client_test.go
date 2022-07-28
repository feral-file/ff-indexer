package tzkt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetContractToken(t *testing.T) {
	tc := New("api.mainnet.tzkt.io")

	token, err := tc.GetContractToken("KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW", "24216")
	assert.NoError(t, err)
	assert.Equal(t, token.Contract.Alias, "Versum Items")
}

func TestRetrieveTokens(t *testing.T) {
	tc := New("api.mainnet.tzkt.io")

	tokens, err := tc.RetrieveTokens("tz1RBi5DCVBYh1EGrcoJszkte1hDjrFfXm5C", 0)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tokens), 1)
}

func TestGetTokenTransfers(t *testing.T) {
	tc := New("api.mainnet.tzkt.io")

	transfers, err := tc.GetTokenTransfers("KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi", "905625")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(transfers), 1)
	assert.Nil(t, transfers[0].From)
	assert.Equal(t, transfers[0].TransactionID, uint64(265770894))

	transfers2, err := tc.GetTokenTransfers("KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "178227")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(transfers2), 1)
	assert.Nil(t, transfers2[0].From)
	assert.Equal(t, transfers2[0].TransactionID, uint64(138631754))
}

func TestGetTransaction(t *testing.T) {
	tc := New("api.mainnet.tzkt.io")

	transaction, err := tc.GetTransaction(123186632)
	assert.NoError(t, err)
	assert.Equal(t, transaction.Hash, "ooL9AXhccM4Jeb525QhRtbb94fozC9rmB4mRanXGU9kHSm42cWX")
}
