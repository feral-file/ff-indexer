package indexer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/tzkt-go"
)

func TestMain(m *testing.M) {
	if err := log.Initialize("", false); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}
	os.Exit(m.Run())
}

func TestIndexTezosTokenProvenance(t *testing.T) {
	engine := New("", []string{}, nil, tzkt.New(""), nil, nil)
	provenances, err := engine.IndexTezosTokenProvenance("KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "178227")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(provenances), 6)
}

func TestIndexTezosTokenOwnersWithNFT(t *testing.T) {
	engine := New("", []string{}, nil, tzkt.New(""), nil, nil)
	ownerBalances, err := engine.IndexTezosTokenOwners("KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "178227")
	assert.NoError(t, err)
	assert.Len(t, ownerBalances, 1)
	assert.NotEqual(t, ownerBalances, []OwnerBalance{})
}

func TestGetTezosTokenByOwner(t *testing.T) {
	engine := New("", []string{}, nil, tzkt.New(""), nil, nil)
	owners, err := engine.GetTezosTokenByOwner("tz1YiYx6TwBnsAgEnXSyhFiM9bqFD54QVhy4", time.Time{}, 0) // incorrect metadata format case
	assert.NoError(t, err)
	assert.NotEmpty(t, owners)
}

func TestIndexTezosTokenOwnersFT(t *testing.T) {
	engine := New("", []string{}, nil, tzkt.New(""), nil, nil)
	ownerBalances, err := engine.IndexTezosTokenOwners("KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW", "24216")
	assert.NoError(t, err)
	assert.Len(t, ownerBalances, 13)
	assert.NotEqual(t, ownerBalances, []OwnerBalance{})
}

func TestIndexTezosTokenOwnersWithNFTOwnByManyAddress(t *testing.T) {
	engine := New("", []string{}, nil, tzkt.New(""), nil, nil)
	ownerBalances, err := engine.IndexTezosTokenOwners("KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "784317")
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(ownerBalances), 333)
	assert.Greater(t, len(ownerBalances), 0)
}

func TestIndexTezosToken(t *testing.T) {

	engine := New("", []string{}, nil, tzkt.New(""), fxhash.New("https://api.fxhash.xyz/graphql"), objkt.New("https://data.objkt.com/v3/graphql"))
	assetUpdates, err := engine.IndexTezosToken(context.Background(), "KT1EfsNuqwLAWDd3o4pvfUx1CAh5GMdTrRvr", "17446")
	assert.NoError(t, err)
	assert.NotEqual(t, assetUpdates.ProjectMetadata.Artists, nil)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "0")
	assert.NoError(t, err)
	assert.NotEqual(t, assetUpdates.ProjectMetadata.Artists, nil)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT1FDfoj9s7ZLE9ycGyTf2QDq32dfvrEsSp8", "0")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT1BRUT7JxudQTHJYefuepxfzCeNjuFSybk7", "11")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT1TnVQhjxeNvLutGvzwZvYtC7vKRpwPWhc6", "401790")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT1AFq5XorPduoYyWxs5gEyrFK6fVjJVbtCj", "0")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "76777")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "28713")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ArtistName, "tz1bWudFpgfnyknWa6MVjYH5VJbA3d9PHpma"), true)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT18pVpRXKPY2c4U2yFEGSH3ZnhB2kL8kwXS", "3164")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://"), true)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT1MhSRKsujc4q5b5KsXvmsvkFyht9h4meZs", "608")
	assert.NoError(t, err)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.PreviewURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, strings.Contains(assetUpdates.ProjectMetadata.ThumbnailURL, "https://assets.objkt.media/file/assets-003/"), true)
	assert.Equal(t, assetUpdates.ProjectMetadata.Title == "", false)

	assetUpdates, err = engine.IndexTezosToken(context.Background(), "KT195VeAcEJ1wioXjDhqjmQ6CrgfZYKtqhro", "2")
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

func TestIndexTezosTokenByOwner(t *testing.T) {

	engine := New("", []string{}, nil, tzkt.New(""), nil, objkt.New("https://data.objkt.com/v3/graphql"))
	_, _, err := engine.IndexTezosTokenByOwner(context.Background(), "tz1eZUHkQDC1bBEbvrrUxkbWEagdZJXQyszc", time.Now().Add(-100*24*time.Hour), 0)
	assert.NoError(t, err)
}
