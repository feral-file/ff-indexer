package main

import (
	"testing"

	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/imageStore"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/utils"
	"github.com/stretchr/testify/assert"
)

func TestDownloadFile(t *testing.T) {
	url := "https://openseauserdata.com/files/dccefc262d24e3c43c1421efbc0e56f1.svg"
	_, _, err := DownloadFile(url)

	assert.NoError(t, err)
}

func TestIsSupportedImageType(t *testing.T) {
	mimeType := "image/svg+xml"
	result := imageStore.IsSupportedImageType(mimeType)

	assert.Equal(t, result, true)
}

func TestConvertSVGToPNG(t *testing.T) {
	url := "https://openseauserdata.com/files/dccefc262d24e3c43c1421efbc0e56f1.svg"
	buf, err := utils.ConvertSVGToPNG(url)

	assert.NotNil(t, buf)
	assert.NoError(t, err)

	url = "https://assets.objkt.media/file/assets-003/QmfYFFvQ5cY7Y7H4mxmeZWmHxum8EhibDfea2ggi2PuEY8/artifact?objkt=1243&creator=tz1exNcVPJdNSqiKDxYkiZRAaxB74jq1m4CQ&viewer=tz1SidNQb9XcwP7L3MzCZD9JHmWw2ebDzgyX&danger=ignored"
	buf, err = utils.ConvertSVGToPNG(url)
	assert.Nil(t, buf)
	assert.NoError(t, err)
}
