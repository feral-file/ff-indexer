package indexer

import (
	"context"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"strings"
	"testing"
	"time"

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
	owners, err := engine.GetTezosTokenByOwner(context.Background(), "tz1YiYx6TwBnsAgEnXSyhFiM9bqFD54QVhy4", time.Time{}, 0) // incorrect metadata format case
	assert.NoError(t, err)
	assert.NotEmpty(t, owners)
}

func TestIndexTezosTokenOwnersFT(t *testing.T) {
	engine := New("", nil, tzkt.New(""), nil, nil)
	owners, err := engine.IndexTezosTokenOwners(context.Background(), "KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW", "24216")
	assert.NoError(t, err)
	assert.NotEmpty(t, owners)
}

func TestIndexTezosToken(t *testing.T) {
	engine := New("", nil, tzkt.New(""), nil, objkt.New("https://data.objkt.com/v3/graphql"))
	assetUpdates, err := engine.IndexTezosToken(context.Background(), "tz1RBi5DCVBYh1EGrcoJszkte1hDjrFfXm5C", "KT1FDfoj9s7ZLE9ycGyTf2QDq32dfvrEsSp8", "0")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "tz1RBi5DCVBYh1EGrcoJszkte1hDjrFfXm5C", "KT1BRUT7JxudQTHJYefuepxfzCeNjuFSybk7", "11")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "tz2ErwNKmtUZAx2UhJ5aRibRjqQDhd4yBRZ6", "KT1TnVQhjxeNvLutGvzwZvYtC7vKRpwPWhc6", "401790")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "tz1SDVkGWcd7AKuEmgMNjvo1chbShRV3Ctm5", "KT1AFq5XorPduoYyWxs5gEyrFK6fVjJVbtCj", "0")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	// case fxhash api fail
	//assetUpdates, err := engine.IndexTezosToken(context.Background(), "tz1NkDtLboBk6gYG2jUyLmKcQHxzDDiSU3Kn", "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "286488")
	//assert.NoError(t, err)
	//assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	//assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	//assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)
}
