package fxhash

import (
	"context"
	"fmt"
	"testing"

	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	if err := log.Initialize("debug", true); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}
}

func TestGetObjectDetailForV0Token(t *testing.T) {
	ctx := context.Background()

	fxObjktID := "FX0-12524"
	api := New("https://api.fxhash.xyz/graphql")
	obj, err := api.GetObjectDetail(ctx, fxObjktID)
	assert.NoError(t, err)

	assert.Equal(t, obj.Name, "Chromatic Squares #11")
	assert.Equal(t, obj.Issuer.Author.ID, "tz1Xx1KnngqcdWXxjaWHknEgWVcGLgStfTqv")
	assert.Equal(t, obj.Issuer.Author.Name, "nicthib")
}

func TestGetObjectDetailForV1Token(t *testing.T) {
	ctx := context.Background()

	fxObjktID := "FX1-8739"
	api := New("https://api.fxhash.xyz/graphql")
	obj, err := api.GetObjectDetail(ctx, fxObjktID)
	assert.NoError(t, err)

	assert.Equal(t, obj.Name, "Brutal Nature #1")
	assert.Equal(t, obj.Issuer.Author.ID, "KT1ASfRKswaGb3KdikjL5DwchYsn9KynHZ7G")
	assert.Equal(t, obj.Issuer.Author.Name, "")
}
