package indexer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

func TestIndexTezosTokenProvenance(t *testing.T) {
	engine := New("", nil, tzkt.New(""), nil, nil)
	provenances, err := engine.IndexTezosTokenProvenance(context.Background(), "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "178227")
	assert.NoError(t, err)
	assert.Len(t, provenances, 6)
}

func TestIndexTezosTokenOwnersWithNFT(t *testing.T) {
	engine := New("", nil, tzkt.New(""), nil, nil)
	owners, err := engine.IndexTezosTokenOwners(context.Background(), "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "178227")
	assert.NoError(t, err)
	assert.Len(t, owners, 1)
}

func TestGetTezosTokenByOwner(t *testing.T) {
	engine := New("", nil, tzkt.New(""), nil, nil)
	owners, err := engine.GetTezosTokenByOwner(context.Background(), "tz1YiYx6TwBnsAgEnXSyhFiM9bqFD54QVhy4", 0) // incorrect metadata format case
	assert.NoError(t, err)
	assert.NotEmpty(t, owners)
}

func TestIndexTezosTokenOwnersFT(t *testing.T) {
	engine := New("", nil, tzkt.New(""), nil, nil)
	owners, err := engine.IndexTezosTokenOwners(context.Background(), "KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW", "24216")
	assert.NoError(t, err)
	assert.NotEmpty(t, owners)
}
