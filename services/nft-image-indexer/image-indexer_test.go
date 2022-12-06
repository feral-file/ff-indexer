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
}
