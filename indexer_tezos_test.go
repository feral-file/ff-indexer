package indexer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

func TestIndexTezosTokenProvenance(t *testing.T) {
	engine := New(nil, tzkt.New("api.mainnet.tzkt.io"), nil, nil)
	provenances, err := engine.IndexTezosTokenProvenance(context.Background(), "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "178227")
	assert.NoError(t, err)

	b, _ := json.MarshalIndent(provenances, "", "  ")
	t.Log(string(b))
}

func TestIndexTezosTokenOwnersWithNFT(t *testing.T) {
	engine := New(nil, tzkt.New("api.mainnet.tzkt.io"), nil, nil)
	owners, err := engine.IndexTezosTokenOwners(context.Background(), "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "178227")
	assert.NoError(t, err)
	assert.Len(t, owners, 1)
}

func TestIndexTezosTokenOwnersFT(t *testing.T) {
	engine := New(nil, tzkt.New("api.mainnet.tzkt.io"), nil, nil)
	owners, err := engine.IndexTezosTokenOwners(context.Background(), "KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW", "24216")
	assert.NoError(t, err)
	assert.NotEmpty(t, owners)
}
